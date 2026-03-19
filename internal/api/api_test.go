package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/a-tunnels/a-tunnels/internal/config"
	"github.com/a-tunnels/a-tunnels/internal/tunnel"
)

func TestAuthMiddleware(t *testing.T) {
	cfg := config.AuthConfig{
		APIKeys: []string{"valid-key", "another-key"},
	}

	auth := NewAuthMiddleware(cfg)

	tests := []struct {
		name       string
		authHeader string
		wantStatus int
	}{
		{"no auth", "", http.StatusUnauthorized},
		{"invalid key", "Bearer invalid", http.StatusUnauthorized},
		{"valid key", "Bearer valid-key", http.StatusOK},
		{"valid key 2", "Bearer another-key", http.StatusOK},
		{"malformed auth", "InvalidFormat", http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := auth.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest("GET", "/test", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}

func TestAPIHealth(t *testing.T) {
	mgr := tunnel.NewManager()
	cfg := &config.ServerConfig{
		APIPort: 8080,
		Auth: config.AuthConfig{
			APIKeys: []string{"test-key"},
		},
	}

	api := NewAPI(mgr, cfg)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	api.handleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("got status %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["status"] != "ok" {
		t.Errorf("got status %s, want ok", resp["status"])
	}
}

func TestAPIMetrics(t *testing.T) {
	mgr := tunnel.NewManager()

	cfg := &config.ServerConfig{
		APIPort: 8080,
		Auth: config.AuthConfig{
			APIKeys: []string{"test-key"},
		},
	}

	api := NewAPI(mgr, cfg)

	tunnels := mgr.List()
	if len(tunnels) != 0 {
		t.Errorf("expected 0 tunnels initially, got %d", len(tunnels))
	}

	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	api.handleMetrics(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("got status %d, want %d", w.Code, http.StatusOK)
	}

	body := w.Body.String()
	if !bytes.Contains([]byte(body), []byte("atunnels_tunnels 0")) {
		t.Errorf("expected metrics to contain tunnel count")
	}
}
