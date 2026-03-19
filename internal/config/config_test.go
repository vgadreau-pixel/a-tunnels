package config

import (
	"os"
	"testing"
	"time"
)

func TestLoadDefault(t *testing.T) {
	cfg := LoadDefault()

	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("expected host 0.0.0.0, got %s", cfg.Server.Host)
	}

	if cfg.Server.HTTPPort != 80 {
		t.Errorf("expected http_port 80, got %d", cfg.Server.HTTPPort)
	}

	if cfg.Server.HTTPSPort != 443 {
		t.Errorf("expected https_port 443, got %d", cfg.Server.HTTPSPort)
	}

	if cfg.Server.APIPort != 8080 {
		t.Errorf("expected api_port 8080, got %d", cfg.Server.APIPort)
	}

	if cfg.Server.MCPPort != 27200 {
		t.Errorf("expected mcp_port 27200, got %d", cfg.Server.MCPPort)
	}

	if cfg.Server.SSHPort != 2222 {
		t.Errorf("expected ssh_port 2222, got %d", cfg.Server.SSHPort)
	}

	if cfg.Server.Limits.MaxTunnels != 100 {
		t.Errorf("expected max_tunnels 100, got %d", cfg.Server.Limits.MaxTunnels)
	}

	if cfg.Server.Limits.MaxConnsPerTunnel != 1000 {
		t.Errorf("expected max_conns 1000, got %d", cfg.Server.Limits.MaxConnsPerTunnel)
	}
}

func TestLoadConfig(t *testing.T) {
	content := `
server:
  host: "192.168.1.1"
  http_port: 8080
  https_port: 8443
  api_port: 9090
  mcp_port: 27300
  ssh_port: 2223
  domain: "example.com"

  tls:
    enabled: true
    email: "admin@example.com"
    auto_tls: true

  auth:
    api_keys:
      - "key1"
      - "key2"
    mcp_token: "mcp-secret"

  storage:
    type: "memory"

  limits:
    max_tunnels: 50
    max_conns_per_tunnel: 500

client:
  server_addr: "server:8080"
  token: "client-token"
  reconnect_interval: 10s

tunnels:
  - name: "test-tunnel"
    protocol: "http"
    local_addr: "localhost:3000"
`

	tmpFile, err := os.CreateTemp("", "config-*.yml")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}
	tmpFile.Close()

	cfg, err := Load(tmpFile.Name())
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg.Server.Host != "192.168.1.1" {
		t.Errorf("expected host 192.168.1.1, got %s", cfg.Server.Host)
	}

	if cfg.Server.HTTPPort != 8080 {
		t.Errorf("expected http_port 8080, got %d", cfg.Server.HTTPPort)
	}

	if cfg.Server.TLS.Enabled != true {
		t.Errorf("expected tls enabled")
	}

	if cfg.Server.TLS.Email != "admin@example.com" {
		t.Errorf("expected email admin@example.com, got %s", cfg.Server.TLS.Email)
	}

	if len(cfg.Server.Auth.APIKeys) != 2 {
		t.Errorf("expected 2 API keys, got %d", len(cfg.Server.Auth.APIKeys))
	}

	if cfg.Server.Auth.MCPToken != "mcp-secret" {
		t.Errorf("expected mcp-token mcp-secret, got %s", cfg.Server.Auth.MCPToken)
	}

	if cfg.Server.Limits.MaxTunnels != 50 {
		t.Errorf("expected max_tunnels 50, got %d", cfg.Server.Limits.MaxTunnels)
	}

	if cfg.Client.ServerAddr != "server:8080" {
		t.Errorf("expected server_addr server:8080, got %s", cfg.Client.ServerAddr)
	}

	if cfg.Client.Token != "client-token" {
		t.Errorf("expected token client-token, got %s", cfg.Client.Token)
	}

	if len(cfg.Tunnels) != 1 {
		t.Errorf("expected 1 tunnel, got %d", len(cfg.Tunnels))
	}

	if cfg.Tunnels[0].Name != "test-tunnel" {
		t.Errorf("expected tunnel name test-tunnel, got %s", cfg.Tunnels[0].Name)
	}
}

func TestLoadConfigNotFound(t *testing.T) {
	_, err := Load("/nonexistent/config.yml")
	if err == nil {
		t.Error("expected error for nonexistent config file")
	}
}

func TestTunnelConfig(t *testing.T) {
	cfg := &TunnelConfig{
		Name:      "my-tunnel",
		Protocol:  "http",
		LocalAddr: "localhost:8080",
		Subdomain: "myapp",
		Timeout:   30 * time.Second,
		MaxConns:  100,
		Headers: map[string]string{
			"X-Custom": "value",
		},
		IPWhitelist: []string{"192.168.1.0/24"},
		WebhookURL:  "https://example.com/webhook",
	}

	if cfg.Name != "my-tunnel" {
		t.Errorf("expected name my-tunnel, got %s", cfg.Name)
	}

	if cfg.Protocol != "http" {
		t.Errorf("expected protocol http, got %s", cfg.Protocol)
	}

	if cfg.LocalAddr != "localhost:8080" {
		t.Errorf("expected localAddr localhost:8080, got %s", cfg.LocalAddr)
	}

	if cfg.Subdomain != "myapp" {
		t.Errorf("expected subdomain myapp, got %s", cfg.Subdomain)
	}

	if cfg.Timeout != 30*time.Second {
		t.Errorf("expected timeout 30s, got %v", cfg.Timeout)
	}

	if cfg.MaxConns != 100 {
		t.Errorf("expected maxConns 100, got %d", cfg.MaxConns)
	}

	if cfg.Headers["X-Custom"] != "value" {
		t.Errorf("expected header X-Custom value, got %s", cfg.Headers["X-Custom"])
	}

	if len(cfg.IPWhitelist) != 1 || cfg.IPWhitelist[0] != "192.168.1.0/24" {
		t.Errorf("expected IP whitelist")
	}

	if cfg.WebhookURL != "https://example.com/webhook" {
		t.Errorf("expected webhook URL")
	}
}

func TestAuthConfig(t *testing.T) {
	cfg := AuthConfig{
		APIKeys:  []string{"key1", "key2"},
		SSHKeys:  []string{"/path/to/key1.pub"},
		MCPToken: "secret-token",
		Admins:   []string{"admin1", "admin2"},
	}

	if len(cfg.APIKeys) != 2 {
		t.Errorf("expected 2 API keys, got %d", len(cfg.APIKeys))
	}

	if cfg.MCPToken != "secret-token" {
		t.Errorf("expected MCP token secret-token, got %s", cfg.MCPToken)
	}

	if len(cfg.Admins) != 2 {
		t.Errorf("expected 2 admins, got %d", len(cfg.Admins))
	}
}
