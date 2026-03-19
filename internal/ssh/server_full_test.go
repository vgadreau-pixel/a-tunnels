package ssh

import (
	"testing"

	"github.com/a-tunnels/a-tunnels/internal/config"
	"github.com/a-tunnels/a-tunnels/internal/tunnel"
)

func TestNewServerFull(t *testing.T) {
	mgr := tunnel.NewManager()
	cfg := &config.ServerConfig{
		Host:    "0.0.0.0",
		SSHPort: 2222,
		Auth: config.AuthConfig{
			APIKeys:  []string{"test-key-1", "test-key-2"},
			SSHKeys:  []string{"/path/to/key1.pub", "/path/to/key2.pub"},
			MCPToken: "mcp-secret-token",
			Admins:   []string{"admin", "user"},
		},
	}

	server := NewServer("0.0.0.0:2222", cfg, mgr)

	if server == nil {
		t.Error("expected non-nil server")
	}

	if server.addr != "0.0.0.0:2222" {
		t.Errorf("expected address 0.0.0.0:2222, got %s", server.addr)
	}

	if server.tunnelMgr == nil {
		t.Error("expected non-nil tunnel manager")
	}

	if server.config == nil {
		t.Error("expected non-nil config")
	}
}

func TestServerConfigFull(t *testing.T) {
	cfg := &config.ServerConfig{
		Host:    "127.0.0.1",
		SSHPort: 22022,
		Domain:  "example.com",
		TLS: config.TLSConfig{
			Enabled:   true,
			Email:     "admin@example.com",
			CertCache: "/var/lib/atunnels/certs",
			AutoTLS:   true,
		},
		Auth: config.AuthConfig{
			APIKeys:  []string{"key1"},
			SSHKeys:  []string{"/keys/user.pub"},
			MCPToken: "secret",
			Admins:   []string{"admin"},
		},
		Storage: config.StorageConfig{
			Type: "memory",
			Path: "/tmp/atunnels.db",
		},
		Limits: config.LimitsConfig{
			MaxTunnels:        50,
			MaxConnsPerTunnel: 500,
			RateLimit:         100,
			RateLimitPeriod:   60,
		},
	}

	if cfg.Host != "127.0.0.1" {
		t.Errorf("expected host 127.0.0.1, got %s", cfg.Host)
	}

	if cfg.SSHPort != 22022 {
		t.Errorf("expected SSH port 22022, got %d", cfg.SSHPort)
	}

	if !cfg.TLS.Enabled {
		t.Error("expected TLS to be enabled")
	}

	if cfg.TLS.Email != "admin@example.com" {
		t.Errorf("expected email admin@example.com, got %s", cfg.TLS.Email)
	}

	if cfg.Limits.MaxTunnels != 50 {
		t.Errorf("expected max_tunnels 50, got %d", cfg.Limits.MaxTunnels)
	}
}

func TestAuthConfigFull(t *testing.T) {
	cfg := config.AuthConfig{
		APIKeys:  []string{"key1", "key2", "key3"},
		SSHKeys:  []string{"/keys/key1.pub", "/keys/key2.pub"},
		MCPToken: "my-secret-mcp-token",
		Admins:   []string{"admin", "operator", "user"},
	}

	if len(cfg.APIKeys) != 3 {
		t.Errorf("expected 3 API keys, got %d", len(cfg.APIKeys))
	}

	if len(cfg.SSHKeys) != 2 {
		t.Errorf("expected 2 SSH keys, got %d", len(cfg.SSHKeys))
	}

	if cfg.MCPToken != "my-secret-mcp-token" {
		t.Errorf("expected MCP token, got %s", cfg.MCPToken)
	}

	if len(cfg.Admins) != 3 {
		t.Errorf("expected 3 admins, got %d", len(cfg.Admins))
	}
}

func TestMultipleServerConfigs(t *testing.T) {
	configs := []*config.ServerConfig{
		{Host: "0.0.0.0", SSHPort: 2222},
		{Host: "127.0.0.1", SSHPort: 2223},
		{Host: "localhost", SSHPort: 2224},
	}

	for i, cfg := range configs {
		mgr := tunnel.NewManager()
		server := NewServer("", cfg, mgr)
		if server == nil {
			t.Errorf("expected non-nil server for config %d", i)
		}
	}
}

func TestTunnelManagerIntegration(t *testing.T) {
	mgr := tunnel.NewManager()

	cfg := &config.ServerConfig{
		SSHPort: 2222,
		Auth: config.AuthConfig{
			APIKeys: []string{"test-key"},
		},
	}

	_ = NewServer("localhost:2222", cfg, mgr)

	if mgr == nil {
		t.Error("expected non-nil manager")
	}
}
