package gateway

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/a-tunnels/a-tunnels/internal/config"
	"github.com/a-tunnels/a-tunnels/internal/tunnel"
)

func TestNewGateway(t *testing.T) {
	cfg := &GatewayConfig{
		HTTPPort:  8080,
		HTTPSPort: 8443,
		TCPPort:   10000,
		WSPort:    11000,
		Domain:    "example.com",
	}

	mgr := tunnel.NewManager()
	gw := NewGateway(cfg, mgr)

	if gw == nil {
		t.Error("expected non-nil gateway")
	}

	if gw.config.HTTPPort != 8080 {
		t.Errorf("expected http port 8080, got %d", gw.config.HTTPPort)
	}

	if gw.config.TCPPort != 10000 {
		t.Errorf("expected tcp port 10000, got %d", gw.config.TCPPort)
	}

	if gw.config.WSPort != 11000 {
		t.Errorf("expected ws port 11000, got %d", gw.config.WSPort)
	}

	if gw.config.Domain != "example.com" {
		t.Errorf("expected domain example.com, got %s", gw.config.Domain)
	}
}

func TestHandleHealth(t *testing.T) {
	mgr := tunnel.NewManager()
	gw := NewGateway(&GatewayConfig{}, mgr)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	gw.handleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["status"] != "ok" {
		t.Errorf("expected status ok, got %s", resp["status"])
	}
}

func TestHandleMetrics(t *testing.T) {
	mgr := tunnel.NewManager()

	ctx := context.Background()
	mgr.Create(ctx, &config.TunnelConfig{
		Name:      "test-tunnel",
		Protocol:  "http",
		LocalAddr: "localhost:3000",
	})

	gw := NewGateway(&GatewayConfig{}, mgr)

	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	gw.handleMetrics(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	body := w.Body.String()

	if !contains(body, "atunnels_tunnels") {
		t.Error("expected metrics to contain tunnel count")
	}
}

func TestHandleHTTPRequestTunnelNotFound(t *testing.T) {
	mgr := tunnel.NewManager()
	gw := NewGateway(&GatewayConfig{}, mgr)

	req := httptest.NewRequest("GET", "/", nil)
	req.Host = "nonexistent.example.com"
	w := httptest.NewRecorder()

	gw.handleHTTPRequest(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestGatewayStop(t *testing.T) {
	mgr := tunnel.NewManager()
	gw := NewGateway(&GatewayConfig{}, mgr)

	err := gw.Stop()
	if err != nil {
		t.Errorf("expected no error on stop, got %v", err)
	}
}

func TestClientConnection(t *testing.T) {
	conn := &ClientConnection{
		ID:        "conn-1",
		TunnelID:  "tunnel-1",
		LocalAddr: "localhost:3000",
	}

	if conn.ID != "conn-1" {
		t.Errorf("expected ID conn-1, got %s", conn.ID)
	}

	if conn.TunnelID != "tunnel-1" {
		t.Errorf("expected TunnelID tunnel-1, got %s", conn.TunnelID)
	}

	if conn.LocalAddr != "localhost:3000" {
		t.Errorf("expected LocalAddr localhost:3000, got %s", conn.LocalAddr)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr) >= 0)
}

func containsAt(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
