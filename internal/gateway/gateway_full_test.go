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

func TestGatewayWithMultipleTunnels(t *testing.T) {
	mgr := tunnel.NewManager()
	ctx := context.Background()

	mgr.Create(ctx, &config.TunnelConfig{
		Name:      "webhook",
		Protocol:  "http",
		LocalAddr: "localhost:3000",
	})

	mgr.Create(ctx, &config.TunnelConfig{
		Name:      "api",
		Protocol:  "http",
		LocalAddr: "localhost:8080",
	})

	mgr.Create(ctx, &config.TunnelConfig{
		Name:      "db",
		Protocol:  "tcp",
		LocalAddr: "localhost:5432",
	})

	tunnels := mgr.List()
	if len(tunnels) != 3 {
		t.Errorf("expected 3 tunnels, got %d", len(tunnels))
	}
}

func TestGatewayConfigFull(t *testing.T) {
	cfg := &GatewayConfig{
		HTTPPort:  80,
		HTTPSPort: 443,
		TCPPort:   10000,
		WSPort:    11000,
		Domain:    "example.com",
		TLSCert:   "/path/to/cert",
		TLSKey:    "/path/to/key",
	}

	if cfg.HTTPPort != 80 {
		t.Errorf("expected HTTPPort 80, got %d", cfg.HTTPPort)
	}

	if cfg.HTTPSPort != 443 {
		t.Errorf("expected HTTPSPort 443, got %d", cfg.HTTPSPort)
	}

	if cfg.TCPPort != 10000 {
		t.Errorf("expected TCPPort 10000, got %d", cfg.TCPPort)
	}

	if cfg.WSPort != 11000 {
		t.Errorf("expected WSPort 11000, got %d", cfg.WSPort)
	}

	if cfg.Domain != "example.com" {
		t.Errorf("expected Domain example.com, got %s", cfg.Domain)
	}

	if cfg.TLSCert != "/path/to/cert" {
		t.Errorf("expected TLSCert /path/to/cert, got %s", cfg.TLSCert)
	}

	if cfg.TLSKey != "/path/to/key" {
		t.Errorf("expected TLSKey /path/to/key, got %s", cfg.TLSKey)
	}
}

func TestGatewayConnections(t *testing.T) {
	conn := &ClientConnection{
		ID:        "conn-123",
		TunnelID:  "tunnel-456",
		LocalAddr: "localhost:3000",
	}

	if conn.ID != "conn-123" {
		t.Errorf("expected ID conn-123, got %s", conn.ID)
	}

	if conn.TunnelID != "tunnel-456" {
		t.Errorf("expected TunnelID tunnel-456, got %s", conn.TunnelID)
	}

	if conn.LocalAddr != "localhost:3000" {
		t.Errorf("expected LocalAddr localhost:3000, got %s", conn.LocalAddr)
	}
}

func TestHandleHTTPRequestHeaders(t *testing.T) {
	mgr := tunnel.NewManager()
	ctx := context.Background()

	tun, _ := mgr.Create(ctx, &config.TunnelConfig{
		Name:      "test-tunnel",
		Protocol:  "http",
		LocalAddr: "localhost:3000",
		Headers: map[string]string{
			"X-Custom-Header": "custom-value",
			"X-Forwarded-For": "1.2.3.4",
		},
	})

	mgr.Start(tun.ID)

	gw := NewGateway(&GatewayConfig{}, mgr)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	req := httptest.NewRequest("GET", "/test", nil)
	req.Host = "test-tunnel.example.com"
	w := httptest.NewRecorder()

	gw.handleHTTPRequest(w, req)
}

func TestHandleHTTPRequestWithAuth(t *testing.T) {
	mgr := tunnel.NewManager()
	ctx := context.Background()

	tun, _ := mgr.Create(ctx, &config.TunnelConfig{
		Name:      "auth-tunnel",
		Protocol:  "http",
		LocalAddr: "localhost:3000",
		Auth: &config.TunnelAuth{
			Type:  "bearer",
			Token: "secret-token",
		},
	})

	mgr.Start(tun.ID)

	gw := NewGateway(&GatewayConfig{}, mgr)

	req := httptest.NewRequest("GET", "/", nil)
	req.Host = "auth-tunnel.example.com"
	w := httptest.NewRecorder()

	gw.handleHTTPRequest(w, req)
}

func TestHandleMetricsMultipleFormats(t *testing.T) {
	mgr := tunnel.NewManager()
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		mgr.Create(ctx, &config.TunnelConfig{
			Name:      "tunnel-" + string(rune('a'+i)),
			Protocol:  "http",
			LocalAddr: "localhost:300" + string(rune('0'+i)),
		})
	}

	gw := NewGateway(&GatewayConfig{}, mgr)

	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	gw.handleMetrics(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	body := w.Body.String()
	if len(body) == 0 {
		t.Error("expected non-empty metrics body")
	}
}

func TestGatewayStartStop(t *testing.T) {
	mgr := tunnel.NewManager()
	gw := NewGateway(&GatewayConfig{
		HTTPPort: 18080,
	}, mgr)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := gw.StartHTTP(ctx)
	if err != nil {
		t.Errorf("unexpected error starting HTTP: %v", err)
	}

	err = gw.Stop()
	if err != nil {
		t.Errorf("unexpected error stopping: %v", err)
	}
}

func TestGatewayStartMultiple(t *testing.T) {
	mgr := tunnel.NewManager()
	gw := NewGateway(&GatewayConfig{
		HTTPPort: 18081,
		TCPPort:  19000,
		WSPort:   19001,
	}, mgr)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := gw.StartHTTP(ctx); err != nil {
		t.Errorf("unexpected error starting HTTP: %v", err)
	}

	if err := gw.StartTCP(ctx); err != nil {
		t.Errorf("unexpected error starting TCP: %v", err)
	}

	if err := gw.StartWebSocket(ctx); err != nil {
		t.Errorf("unexpected error starting WebSocket: %v", err)
	}

	gw.Stop()
}
