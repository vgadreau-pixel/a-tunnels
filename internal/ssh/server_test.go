package ssh

import (
	"testing"

	"github.com/a-tunnels/a-tunnels/internal/config"
	"github.com/a-tunnels/a-tunnels/internal/tunnel"
)

func TestNewServer(t *testing.T) {
	mgr := tunnel.NewManager()
	cfg := &config.ServerConfig{
		SSHPort: 2222,
		Auth: config.AuthConfig{
			APIKeys: []string{"test-key"},
		},
	}

	server := NewServer("localhost:2222", cfg, mgr)

	if server == nil {
		t.Error("expected non-nil server")
	}

	if server.addr != "localhost:2222" {
		t.Errorf("expected address localhost:2222, got %s", server.addr)
	}
}

func TestServerConfig(t *testing.T) {
	cfg := &config.ServerConfig{
		Host:    "0.0.0.0",
		SSHPort: 2222,
		Auth: config.AuthConfig{
			APIKeys:  []string{"key1", "key2"},
			SSHKeys:  []string{"/path/to/key1.pub"},
			MCPToken: "mcp-secret",
			Admins:   []string{"admin"},
		},
	}

	if cfg.Host != "0.0.0.0" {
		t.Errorf("expected host 0.0.0.0, got %s", cfg.Host)
	}

	if cfg.SSHPort != 2222 {
		t.Errorf("expected port 2222, got %d", cfg.SSHPort)
	}

	if len(cfg.Auth.APIKeys) != 2 {
		t.Errorf("expected 2 API keys, got %d", len(cfg.Auth.APIKeys))
	}

	if cfg.Auth.MCPToken != "mcp-secret" {
		t.Errorf("expected MCP token mcp-secret, got %s", cfg.Auth.MCPToken)
	}
}

func TestServerAuthConfig(t *testing.T) {
	authCfg := config.AuthConfig{
		APIKeys:  []string{"valid-key"},
		SSHKeys:  []string{"/etc/atunnels/keys/user.pub"},
		MCPToken: "secret-token",
		Admins:   []string{"admin", "user1"},
	}

	if len(authCfg.APIKeys) != 1 {
		t.Errorf("expected 1 API key, got %d", len(authCfg.APIKeys))
	}

	if len(authCfg.Admins) != 2 {
		t.Errorf("expected 2 admins, got %d", len(authCfg.Admins))
	}
}
