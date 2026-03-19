package auth

import (
	"testing"

	"github.com/a-tunnels/a-tunnels/internal/config"
)

func TestAuthMiddlewareValidate(t *testing.T) {
	cfg := config.AuthConfig{
		APIKeys: []string{"key1", "key2", "key3"},
	}

	auth := NewAuthMiddleware(cfg)

	tests := []struct {
		name  string
		token string
		want  bool
	}{
		{"valid token 1", "key1", true},
		{"valid token 2", "key2", true},
		{"valid token 3", "key3", true},
		{"invalid token", "invalid", false},
		{"empty token", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := auth.ValidateToken(tt.token)
			if result != tt.want {
				t.Errorf("ValidateToken(%s) = %v, want %v", tt.token, result, tt.want)
			}
		})
	}
}

func TestAuthMiddlewareEmptyAPIKeys(t *testing.T) {
	cfg := config.AuthConfig{
		APIKeys: []string{},
	}

	auth := NewAuthMiddleware(cfg)

	if auth.ValidateToken("any-token") {
		t.Error("expected false for empty API keys")
	}
}

func TestAuthMiddlewareSingleAPIKey(t *testing.T) {
	cfg := config.AuthConfig{
		APIKeys: []string{"only-key"},
	}

	auth := NewAuthMiddleware(cfg)

	if !auth.ValidateToken("only-key") {
		t.Error("expected true for valid token")
	}

	if auth.ValidateToken("other-key") {
		t.Error("expected false for invalid token")
	}
}
