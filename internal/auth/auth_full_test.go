package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/a-tunnels/a-tunnels/internal/config"
)

func TestAuthMiddlewareComplex(t *testing.T) {
	cfg := config.AuthConfig{
		APIKeys:  []string{"key1", "key2", "key3"},
		SSHKeys:  []string{"/path/to/key1.pub", "/path/to/key2.pub"},
		MCPToken: "mcp-secret-token",
		Admins:   []string{"admin1", "admin2", "admin3"},
	}

	auth := NewAuthMiddleware(cfg)

	tests := []struct {
		name  string
		token string
		valid bool
	}{
		{"key1", "key1", true},
		{"key2", "key2", true},
		{"key3", "key3", true},
		{"invalid", "invalid", false},
		{"empty", "", false},
		{"wrong", "key", false},
		{"partial", "key1extra", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := auth.ValidateToken(tt.token)
			if result != tt.valid {
				t.Errorf("ValidateToken(%s) = %v, want %v", tt.token, result, tt.valid)
			}
		})
	}
}

func TestAuthMiddlewareHTTP(t *testing.T) {
	cfg := config.AuthConfig{
		APIKeys: []string{"valid-token"},
	}

	auth := NewAuthMiddleware(cfg)

	handler := auth.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name       string
		authHeader string
		wantStatus int
	}{
		{"valid bearer", "Bearer valid-token", http.StatusOK},
		{"missing", "", http.StatusUnauthorized},
		{"invalid", "Bearer wrong-token", http.StatusUnauthorized},
		{"malformed 1", "Bearer", http.StatusUnauthorized},
		{"malformed 2", "Basic dXNlcjpwYXNz", http.StatusUnauthorized},
		{"malformed 3", "OAuth abc", http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
			if w.Code != tt.wantStatus {
				t.Errorf("got %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}

func TestAuthConfigMultiple(t *testing.T) {
	cfg := config.AuthConfig{
		APIKeys: []string{},
		SSHKeys: []string{},
		Admins:  []string{},
	}

	auth := NewAuthMiddleware(cfg)

	if auth.ValidateToken("any") {
		t.Error("expected false for empty config")
	}
}

func TestAuthWithManyKeys(t *testing.T) {
	keys := make([]string, 50)
	for i := 0; i < 50; i++ {
		keys[i] = "very-long-token-name-" + string(rune('0'+i%10))
	}

	cfg := config.AuthConfig{
		APIKeys: keys,
	}

	auth := NewAuthMiddleware(cfg)

	for _, key := range keys {
		if !auth.ValidateToken(key) {
			t.Errorf("expected key to be valid: %s", key)
		}
	}

	if auth.ValidateToken("not-in-list") {
		t.Error("expected false for key not in list")
	}
}
