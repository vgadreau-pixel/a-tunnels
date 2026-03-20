package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/a-tunnels/a-tunnels/internal/api"
	"github.com/a-tunnels/a-tunnels/internal/config"
	"github.com/a-tunnels/a-tunnels/internal/gateway"
	"github.com/a-tunnels/a-tunnels/internal/mcp"
	"github.com/a-tunnels/a-tunnels/internal/shortener"
	"github.com/a-tunnels/a-tunnels/internal/ssh"
	"github.com/a-tunnels/a-tunnels/internal/tunnel"
)

var (
	version   = "dev"
	commit    = "unknown"
	buildTime = "unknown"
)

func main() {
	configPath := flag.String("config", "atunnels.yml", "Path to configuration file")
	showVersion := flag.Bool("version", false, "Show version information")
	flag.Parse()

	if *showVersion {
		fmt.Printf("A-Tunnels Server v%s (commit: %s, built: %s)\n", version, commit, buildTime)
		os.Exit(0)
	}

	cfg, err := loadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	log.Printf("Starting A-Tunnels Server v%s", version)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tunnelMgr := tunnel.NewManager()

	var shortenerStorage shortener.Storage
	if cfg.Server.Shortener.Enabled {
		// Initialize shortener storage based on server storage config
		if cfg.Server.Storage.Type == "file" && cfg.Server.Storage.Path != "" {
			shortenerPath := filepath.Join(filepath.Dir(cfg.Server.Storage.Path), "shortener.json")
			storage, err := shortener.NewFileStorage(shortenerPath)
			if err != nil {
				log.Printf("Failed to create shortener file storage: %v, using memory storage", err)
			} else {
				shortenerStorage = storage
			}
		}
	}

	gw := gateway.NewGatewayWithStorage(&gateway.GatewayConfig{
		HTTPPort:           cfg.Server.HTTPPort,
		HTTPSPort:          cfg.Server.HTTPSPort,
		TCPPort:            cfg.Server.TCPPortStart,
		WSPort:             cfg.Server.WSPortStart,
		Domain:             cfg.Server.Domain,
		RateLimit:          cfg.Server.Limits.RateLimit,
		ShortenerRateLimit: cfg.Server.Limits.ShortenerLimit,
		ShortenerPeriod:    int(cfg.Server.Limits.ShortenerPeriod),
		Shortener: gateway.GatewayShortenerConfig{
			Enabled:     cfg.Server.Shortener.Enabled,
			DefaultTTL:  cfg.Server.Shortener.DefaultTTL,
			MaxTTL:      cfg.Server.Shortener.MaxTTL,
			MaxLength:   cfg.Server.Shortener.MaxLength,
			BasePath:    cfg.Server.Shortener.BasePath,
			CleanupFreq: cfg.Server.Shortener.CleanupFreq,
		},
	}, tunnelMgr, shortenerStorage)

	// Start HTTP gateway if enabled
	if cfg.Server.HTTPEnabled {
		if err := gw.StartHTTP(ctx); err != nil {
			log.Printf("HTTP gateway failed: %v", err)
		}
	}

	// Start HTTPS gateway if enabled
	if cfg.Server.HTTPSEnabled {
		if err := gw.StartHTTPS(ctx); err != nil {
			log.Printf("HTTPS gateway failed: %v", err)
		}
	}

	// Start TCP gateway if enabled
	if cfg.Server.TCPEnabled {
		if err := gw.StartTCP(ctx); err != nil {
			log.Printf("TCP gateway failed: %v", err)
		}
	}

	// Start WebSocket gateway if enabled
	if cfg.Server.WSEnabled {
		if err := gw.StartWebSocket(ctx); err != nil {
			log.Printf("WebSocket gateway failed: %v", err)
		}
	}

	// Start API server if enabled
	var apiServer *api.API
	if cfg.Server.APIEnabled {
		apiServer = api.NewAPI(tunnelMgr, &cfg.Server)
		go func() {
			if err := apiServer.Start(); err != nil {
				log.Printf("API server error: %v", err)
			}
		}()
	}

	// Start MCP server if enabled
	var mcpServer *mcp.Server
	if cfg.Server.MCPEnabled {
		mcpServer = mcp.NewServer(fmt.Sprintf(":%d", cfg.Server.MCPPort), tunnelMgr, cfg.Server.MCPToken)
		go func() {
			if err := mcpServer.Start(); err != nil {
				log.Printf("MCP server error: %v", err)
			}
		}()
	}

	// Start SSH server if enabled
	var sshServer *ssh.Server
	if cfg.Server.SSHEnabled {
		sshServer = ssh.NewServer(fmt.Sprintf(":%d", cfg.Server.SSHPort), &cfg.Server, tunnelMgr)
		go func() {
			if err := sshServer.Start(); err != nil {
				log.Printf("SSH server error: %v", err)
			}
		}()
	}

	log.Printf("All services started successfully")

	// Start cleanup goroutine if enabled
	if cfg.Server.CleanupEnabled {
		go runCleanup(ctx, tunnelMgr, &cfg.Server)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("Shutting down...")
	cancel()
	gw.Stop()
	if apiServer != nil {
		apiServer.Stop()
	}
	if mcpServer != nil {
		mcpServer.Stop()
	}
	log.Println("Server stopped")
}

func loadConfig(path string) (*config.Config, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return config.LoadDefault(), nil
	}
	return config.Load(path)
}

func runCleanup(ctx context.Context, mgr tunnel.Manager, cfg *config.ServerConfig) {
	interval := cfg.CleanupInterval
	if interval <= 0 {
		interval = 1 * time.Hour
	}
	if cfg.DisableAfter <= 0 {
		cfg.DisableAfter = 30 * 24 * time.Hour // default 1 month
	}
	if cfg.DeleteAfter <= 0 {
		cfg.DeleteAfter = 365 * 24 * time.Hour // default 1 year
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Cleanup goroutine stopped")
			return
		case <-ticker.C:
			runCleanupOnce(mgr, cfg)
		}
	}
}

func runCleanupOnce(mgr tunnel.Manager, cfg *config.ServerConfig) {
	tunnels := mgr.List()
	now := time.Now()

	for _, t := range tunnels {
		if t.Status == tunnel.TunnelStatusDisabled {
			continue
		}

		lastReq := t.Stats.LastRequestAt
		if lastReq.IsZero() {
			lastReq = t.CreatedAt
		}

		inactive := now.Sub(lastReq)

		if inactive >= cfg.DeleteAfter {
			log.Printf("Cleaning up: deleting tunnel %s (inactive for %v)", t.Name, inactive)
			if err := mgr.Delete(t.ID); err != nil {
				log.Printf("Failed to delete tunnel %s: %v", t.Name, err)
			}
		} else if inactive >= cfg.DisableAfter {
			log.Printf("Cleaning up: disabling tunnel %s (inactive for %v)", t.Name, inactive)
			if err := mgr.Disable(t.ID); err != nil {
				log.Printf("Failed to disable tunnel %s: %v", t.Name, err)
			}
		}
	}
}
