package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/a-tunnels/a-tunnels/internal/config"
)

type Client struct {
	config     *config.ClientConfig
	httpClient *http.Client
	ctx        context.Context
	cancel     context.CancelFunc
}

func main() {
	configPath := flag.String("config", "client.yml", "Path to client configuration")
	showVersion := flag.Bool("version", false, "Show version information")
	flag.Parse()

	if *showVersion {
		fmt.Println("A-Tunnels Client v1.0.0")
		os.Exit(0)
	}

	cfg, err := loadClientConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	client := NewClient(&cfg.Client)
	if err := client.Connect(); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}

	log.Println("Connected to A-Tunnels server")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	client.Disconnect()
	log.Println("Disconnected")
}

func loadClientConfig(path string) (*config.Config, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file not found: %s", path)
	}
	return config.Load(path)
}

func NewClient(cfg *config.ClientConfig) *Client {
	ctx, cancel := context.WithCancel(context.Background())
	return &Client{
		config: cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		ctx:    ctx,
		cancel: cancel,
	}
}

func (c *Client) Connect() error {
	log.Printf("Connecting to %s", c.config.ServerAddr)

	for _, tunnel := range c.config.Tunnels {
		go c.startTunnel(&tunnel)
	}

	return nil
}

func (c *Client) startTunnel(cfg *config.TunnelConfig) {
	log.Printf("Starting tunnel: %s -> %s (%s)", cfg.Name, cfg.LocalAddr, cfg.Protocol)

	switch cfg.Protocol {
	case "http":
		c.startHTTPTunnel(cfg)
	case "tcp":
		c.startTCPTunnel(cfg)
	case "websocket":
		c.startWebSocketTunnel(cfg)
	}
}

func (c *Client) startHTTPTunnel(cfg *config.TunnelConfig) {
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			if err := c.registerTunnel(cfg); err != nil {
				log.Printf("Failed to register tunnel %s: %v", cfg.Name, err)
				time.Sleep(5 * time.Second)
				continue
			}
			break
		}
		break
	}

	ln, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		log.Printf("Failed to listen: %v", err)
		return
	}
	defer ln.Close()

	log.Printf("Listening on %s", ln.Addr().String())

	for {
		conn, err := ln.Accept()
		if err != nil {
			if c.ctx.Err() != nil {
				return
			}
			log.Printf("Accept error: %v", err)
			continue
		}
		go c.handleConnection(conn, cfg)
	}
}

func (c *Client) handleConnection(conn net.Conn, cfg *config.TunnelConfig) {
	defer conn.Close()

	localConn, err := net.Dial("tcp", cfg.LocalAddr)
	if err != nil {
		log.Printf("Failed to connect to local: %v", err)
		return
	}
	defer localConn.Close()

	go io.Copy(localConn, conn)
	io.Copy(conn, localConn)
}

func (c *Client) startTCPTunnel(cfg *config.TunnelConfig) {
	log.Printf("TCP tunnel %s not implemented", cfg.Name)
}

func (c *Client) startWebSocketTunnel(cfg *config.TunnelConfig) {
	log.Printf("WebSocket tunnel %s not implemented", cfg.Name)
}

func (c *Client) registerTunnel(cfg *config.TunnelConfig) error {
	url := fmt.Sprintf("https://%s/api/v1/tunnels", c.config.ServerAddr)

	tunnelData := map[string]interface{}{
		"name":      cfg.Name,
		"protocol":  cfg.Protocol,
		"localAddr": cfg.LocalAddr,
	}

	data, _ := json.Marshal(tunnelData)

	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(data))
	req.Header.Set("Authorization", "Bearer "+c.config.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("registration failed: %d", resp.StatusCode)
	}

	return nil
}

func (c *Client) Disconnect() {
	c.cancel()
}
