package tunnel

import (
	"context"
	"testing"

	"github.com/a-tunnels/a-tunnels/internal/config"
)

func TestNewTunnel(t *testing.T) {
	cfg := &config.TunnelConfig{
		Name:      "test-tunnel",
		Protocol:  "http",
		LocalAddr: "localhost:3000",
	}

	tunnel := NewTunnel(cfg)

	if tunnel.Name != "test-tunnel" {
		t.Errorf("expected name test-tunnel, got %s", tunnel.Name)
	}

	if tunnel.Protocol != "http" {
		t.Errorf("expected protocol http, got %s", tunnel.Protocol)
	}

	if tunnel.LocalAddr != "localhost:3000" {
		t.Errorf("expected localAddr localhost:3000, got %s", tunnel.LocalAddr)
	}

	if tunnel.Status != TunnelStatusPending {
		t.Errorf("expected status pending, got %s", tunnel.Status)
	}

	if tunnel.ID == "" {
		t.Error("expected non-empty ID")
	}
}

func TestTunnelStatus(t *testing.T) {
	tunnel := &Tunnel{}

	tunnel.SetStatus(TunnelStatusActive)
	if tunnel.GetStatus() != TunnelStatusActive {
		t.Errorf("expected active status")
	}

	tunnel.SetStatus(TunnelStatusStopped)
	if tunnel.GetStatus() != TunnelStatusStopped {
		t.Errorf("expected stopped status")
	}
}

func TestTunnelStats(t *testing.T) {
	tunnel := &Tunnel{
		Stats: &TunnelStats{},
	}

	tunnel.UpdateStats(5, 100, 200)
	stats := tunnel.GetStats()

	if stats.ActiveConnections != 5 {
		t.Errorf("expected 5 connections, got %d", stats.ActiveConnections)
	}

	if stats.TotalBytesIn != 100 {
		t.Errorf("expected 100 bytes in, got %d", stats.TotalBytesIn)
	}

	if stats.TotalBytesOut != 200 {
		t.Errorf("expected 200 bytes out, got %d", stats.TotalBytesOut)
	}

	if stats.TotalRequests != 1 {
		t.Errorf("expected 1 request, got %d", stats.TotalRequests)
	}
}

func TestManager(t *testing.T) {
	mgr := NewManager()

	cfg := &config.TunnelConfig{
		Name:      "test-tunnel",
		Protocol:  "http",
		LocalAddr: "localhost:3000",
	}

	ctx := context.Background()

	// Test Create
	tunnel, err := mgr.Create(ctx, cfg)
	if err != nil {
		t.Fatalf("failed to create tunnel: %v", err)
	}

	if tunnel.Name != "test-tunnel" {
		t.Errorf("expected name test-tunnel")
	}

	// Test Get
	tRet, err := mgr.Get(tunnel.ID)
	if err != nil {
		t.Fatalf("failed to get tunnel: %v", err)
	}

	if tRet.ID != tunnel.ID {
		t.Errorf("expected same ID")
	}

	// Test GetByName
	tRet, err = mgr.GetByName("test-tunnel")
	if err != nil {
		t.Fatalf("failed to get tunnel by name: %v", err)
	}

	// Test List
	tunnels := mgr.List()
	if len(tunnels) != 1 {
		t.Errorf("expected 1 tunnel, got %d", len(tunnels))
	}

	// Test Start
	err = mgr.Start(tunnel.ID)
	if err != nil {
		t.Fatalf("failed to start tunnel: %v", err)
	}

	tRet, _ = mgr.Get(tunnel.ID)
	if tRet.Status != TunnelStatusActive {
		t.Errorf("expected active status after start")
	}

	// Test Stop
	err = mgr.Stop(tunnel.ID)
	if err != nil {
		t.Fatalf("failed to stop tunnel: %v", err)
	}

	tRet, _ = mgr.Get(tunnel.ID)
	if tRet.Status != TunnelStatusStopped {
		t.Errorf("expected stopped status after stop")
	}

	// Test Delete
	err = mgr.Delete(tunnel.ID)
	if err != nil {
		t.Fatalf("failed to delete tunnel: %v", err)
	}

	tunnels = mgr.List()
	if len(tunnels) != 0 {
		t.Errorf("expected 0 tunnels after delete")
	}
}

func TestDuplicateTunnel(t *testing.T) {
	mgr := NewManager()

	cfg := &config.TunnelConfig{
		Name:      "duplicate-test",
		Protocol:  "http",
		LocalAddr: "localhost:3000",
	}

	ctx := context.Background()

	_, err := mgr.Create(ctx, cfg)
	if err != nil {
		t.Fatalf("first create failed: %v", err)
	}

	_, err = mgr.Create(ctx, cfg)
	if err == nil {
		t.Error("expected error for duplicate tunnel")
	}
}
