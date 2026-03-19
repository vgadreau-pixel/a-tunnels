package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/a-tunnels/a-tunnels/internal/api"
	"github.com/a-tunnels/a-tunnels/internal/config"
	"github.com/a-tunnels/a-tunnels/internal/gateway"
	"github.com/a-tunnels/a-tunnels/internal/mcp"
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

	gw := gateway.NewGateway(&gateway.GatewayConfig{
		HTTPPort:  cfg.Server.HTTPPort,
		HTTPSPort: cfg.Server.HTTPSPort,
		TCPPort:   cfg.Server.TCPPortStart,
		WSPort:    cfg.Server.WSPortStart,
		Domain:    cfg.Server.Domain,
	}, tunnelMgr)

	if err := gw.StartHTTP(ctx); err != nil {
		log.Printf("HTTP gateway failed: %v", err)
	}

	if err := gw.StartTCP(ctx); err != nil {
		log.Printf("TCP gateway failed: %v", err)
	}

	if err := gw.StartWebSocket(ctx); err != nil {
		log.Printf("WebSocket gateway failed: %v", err)
	}

	apiServer := api.NewAPI(tunnelMgr, &cfg.Server)
	go func() {
		if err := apiServer.Start(); err != nil {
			log.Printf("API server error: %v", err)
		}
	}()

	mcpServer := mcp.NewServer(fmt.Sprintf(":%d", cfg.Server.MCPPort), tunnelMgr, cfg.Server.MCPToken)
	go func() {
		if err := mcpServer.Start(); err != nil {
			log.Printf("MCP server error: %v", err)
		}
	}()

	sshServer := ssh.NewServer(fmt.Sprintf(":%d", cfg.Server.SSHPort), &cfg.Server, tunnelMgr)
	go func() {
		if err := sshServer.Start(); err != nil {
			log.Printf("SSH server error: %v", err)
		}
	}()

	log.Printf("All services started successfully")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("Shutting down...")
	cancel()
	gw.Stop()
	apiServer.Stop()
	mcpServer.Stop()
	log.Println("Server stopped")
}

func loadConfig(path string) (*config.Config, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return config.LoadDefault(), nil
	}
	return config.Load(path)
}
