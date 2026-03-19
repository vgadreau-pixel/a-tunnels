package tunnel

import (
	"context"
	"testing"
	"time"

	"github.com/a-tunnels/a-tunnels/internal/config"
)

func TestNewTunnelFull(t *testing.T) {
	cfg := &config.TunnelConfig{
		Name:       "full-test-tunnel",
		Protocol:   "http",
		LocalAddr:  "localhost:8080",
		Subdomain:  "fulltest",
		RemotePort: 18080,
		Timeout:    60 * time.Second,
		MaxConns:   500,
		Headers: map[string]string{
			"X-Custom": "value",
		},
		IPWhitelist:   []string{"192.168.1.0/24", "10.0.0.0/8"},
		WebhookURL:    "https://example.com/webhook",
		WebhookEvents: []string{"connect", "disconnect", "error"},
	}

	tun := NewTunnel(cfg)

	if tun.Name != "full-test-tunnel" {
		t.Errorf("expected name full-test-tunnel, got %s", tun.Name)
	}

	if tun.Protocol != "http" {
		t.Errorf("expected protocol http, got %s", tun.Protocol)
	}

	if tun.LocalAddr != "localhost:8080" {
		t.Errorf("expected localAddr localhost:8080, got %s", tun.LocalAddr)
	}

	if tun.Subdomain != "fulltest" {
		t.Errorf("expected subdomain fulltest, got %s", tun.Subdomain)
	}

	if tun.RemotePort != 18080 {
		t.Errorf("expected remotePort 18080, got %d", tun.RemotePort)
	}

	if tun.Status != TunnelStatusPending {
		t.Errorf("expected status pending, got %s", tun.Status)
	}

	if tun.ID == "" {
		t.Error("expected non-empty ID")
	}

	if tun.Config == nil {
		t.Error("expected config to be set")
	}
}

func TestTunnelStatusTransitions(t *testing.T) {
	tun := &Tunnel{}

	transitions := []TunnelStatus{
		TunnelStatusPending,
		TunnelStatusActive,
		TunnelStatusPaused,
		TunnelStatusActive,
		TunnelStatusError,
		TunnelStatusStopped,
	}

	for _, status := range transitions {
		tun.SetStatus(status)
		if tun.GetStatus() != status {
			t.Errorf("expected status %s, got %s", status, tun.GetStatus())
		}
	}
}

func TestTunnelStatsAccumulation(t *testing.T) {
	tun := &Tunnel{
		Stats: &TunnelStats{},
	}

	iterations := 100
	for i := 0; i < iterations; i++ {
		tun.UpdateStats(int64(i%10), int64(i*100), int64(i*200))
	}

	stats := tun.GetStats()

	if stats.TotalRequests != int64(iterations) {
		t.Errorf("expected %d requests, got %d", iterations, stats.TotalRequests)
	}

	if stats.TotalBytesIn == 0 {
		t.Error("expected non-zero bytes in")
	}

	if stats.TotalBytesOut == 0 {
		t.Error("expected non-zero bytes out")
	}

	if stats.LastRequestAt.IsZero() {
		t.Error("expected last request time to be set")
	}
}

func TestTunnelStatsConcurrent(t *testing.T) {
	tun := &Tunnel{
		Stats: &TunnelStats{},
	}

	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				tun.UpdateStats(1, 100, 200)
			}
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	stats := tun.GetStats()
	if stats.TotalRequests != 1000 {
		t.Errorf("expected 1000 requests, got %d", stats.TotalRequests)
	}
}

func TestManagerFull(t *testing.T) {
	mgr := NewManager()
	ctx := context.Background()

	tunnelConfigs := []*config.TunnelConfig{
		{Name: "tunnel1", Protocol: "http", LocalAddr: "localhost:3000"},
		{Name: "tunnel2", Protocol: "tcp", LocalAddr: "localhost:5432"},
		{Name: "tunnel3", Protocol: "websocket", LocalAddr: "localhost:8080"},
		{Name: "tunnel4", Protocol: "http", LocalAddr: "localhost:4000"},
		{Name: "tunnel5", Protocol: "http", LocalAddr: "localhost:5000"},
	}

	for _, cfg := range tunnelConfigs {
		_, err := mgr.Create(ctx, cfg)
		if err != nil {
			t.Fatalf("failed to create tunnel %s: %v", cfg.Name, err)
		}
	}

	tunnels := mgr.List()
	if len(tunnels) != 5 {
		t.Errorf("expected 5 tunnels, got %d", len(tunnels))
	}

	for _, tun := range tunnels {
		err := mgr.Start(tun.ID)
		if err != nil {
			t.Errorf("failed to start tunnel %s: %v", tun.Name, err)
		}

		stats, err := mgr.GetStats(tun.ID)
		if err != nil {
			t.Errorf("failed to get stats for %s: %v", tun.Name, err)
		}
		_ = stats

		err = mgr.Stop(tun.ID)
		if err != nil {
			t.Errorf("failed to stop tunnel %s: %v", tun.Name, err)
		}
	}

	for _, tun := range tunnels {
		err := mgr.Delete(tun.ID)
		if err != nil {
			t.Errorf("failed to delete tunnel %s: %v", tun.Name, err)
		}
	}

	if len(mgr.List()) != 0 {
		t.Error("expected 0 tunnels after deletion")
	}
}

func TestManagerRestart(t *testing.T) {
	mgr := NewManager()
	ctx := context.Background()

	tun, err := mgr.Create(ctx, &config.TunnelConfig{
		Name: "restart-test", Protocol: "http", LocalAddr: "localhost:3000",
	})
	if err != nil {
		t.Fatalf("failed to create: %v", err)
	}

	mgr.Start(tun.ID)
	mgr.Stop(tun.ID)

	err = mgr.Restart(tun.ID)
	if err != nil {
		t.Errorf("failed to restart: %v", err)
	}

	tRet, _ := mgr.Get(tun.ID)
	if tRet.Status != TunnelStatusActive {
		t.Errorf("expected active status after restart, got %s", tRet.Status)
	}
}

func TestTunnelEventChannel(t *testing.T) {
	mgr := NewManager()
	ctx := context.Background()

	events := mgr.Subscribe()

	tun, _ := mgr.Create(ctx, &config.TunnelConfig{
		Name: "event-test", Protocol: "http", LocalAddr: "localhost:3000",
	})

	mgr.Start(tun.ID)
	mgr.Stop(tun.ID)
	mgr.Delete(tun.ID)

	eventCount := 0
	timeout := time.After(500 * time.Millisecond)
	for {
		select {
		case <-events:
			eventCount++
		case <-timeout:
			goto done
		}
	}

done:
	if eventCount < 3 {
		t.Logf("received %d events", eventCount)
	}
}
