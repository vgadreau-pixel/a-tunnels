package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/a-tunnels/a-tunnels/internal/config"
	"github.com/a-tunnels/a-tunnels/internal/tunnel"
)

func TestAPIWithTunnels(t *testing.T) {
	mgr := tunnel.NewManager()
	ctx := context.Background()

	cfg := &config.ServerConfig{
		APIPort: 8080,
		Auth: config.AuthConfig{
			APIKeys: []string{"test-key"},
		},
	}

	_ = NewAPI(mgr, cfg)

	mgr.Create(ctx, &config.TunnelConfig{
		Name:      "test-tunnel",
		Protocol:  "http",
		LocalAddr: "localhost:3000",
	})

	tunnels := mgr.List()
	if len(tunnels) != 1 {
		t.Errorf("expected 1 tunnel, got %d", len(tunnels))
	}
}

func TestAPIAuthMiddlewareIntegration(t *testing.T) {
	mgr := tunnel.NewManager()
	cfg := &config.ServerConfig{
		APIPort: 8080,
		Auth: config.AuthConfig{
			APIKeys: []string{"valid-key", "another-key"},
		},
	}

	api := NewAPI(mgr, cfg)

	handler := api.auth.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name       string
		authHeader string
		wantStatus int
	}{
		{"valid key 1", "Bearer valid-key", http.StatusOK},
		{"valid key 2", "Bearer another-key", http.StatusOK},
		{"no bearer prefix", "valid-key", http.StatusUnauthorized},
		{"empty bearer", "Bearer ", http.StatusUnauthorized},
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

func TestAPIHealthResponseFormat(t *testing.T) {
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

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)

	if _, ok := resp["status"]; !ok {
		t.Error("expected status in response")
	}
}

func TestAPIMetricsWithNoTunnels(t *testing.T) {
	mgr := tunnel.NewManager()
	cfg := &config.ServerConfig{
		APIPort: 8080,
		Auth: config.AuthConfig{
			APIKeys: []string{"test-key"},
		},
	}

	api := NewAPI(mgr, cfg)
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	api.handleMetrics(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	body := w.Body.String()
	if body == "" {
		t.Error("expected non-empty body")
	}
}

func TestAPIWithMultipleAuthKeys(t *testing.T) {
	keys := make([]string, 10)
	for i := 0; i < 10; i++ {
		keys[i] = "key-" + string(rune('0'+i))
	}

	cfg := config.AuthConfig{
		APIKeys: keys,
	}

	auth := NewAuthMiddleware(cfg)

	for i, key := range keys {
		if !auth.ValidateToken(key) {
			t.Errorf("expected key-%d to be valid", i)
		}
	}

	if auth.ValidateToken("invalid-key") {
		t.Error("expected invalid key to be rejected")
	}
}

func TestAPI(t *testing.T) {
	mgr := tunnel.NewManager()
	cfg := &config.ServerConfig{
		APIPort: 8080,
		Auth: config.AuthConfig{
			APIKeys: []string{"test-key"},
		},
	}

	api := NewAPI(mgr, cfg)
	if api == nil {
		t.Error("expected non-nil API")
	}

	if api.tunnelMgr == nil {
		t.Error("expected non-nil tunnel manager")
	}

	if api.config == nil {
		t.Error("expected non-nil config")
	}
}
