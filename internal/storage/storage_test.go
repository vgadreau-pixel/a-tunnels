package storage

import (
	"testing"

	"github.com/a-tunnels/a-tunnels/internal/config"
	"github.com/a-tunnels/a-tunnels/internal/tunnel"
)

func TestMemoryStorage(t *testing.T) {
	storage := NewMemoryStorage()

	cfg := &config.TunnelConfig{
		Name:      "test-tunnel",
		Protocol:  "http",
		LocalAddr: "localhost:3000",
	}

	tun := tunnel.NewTunnel(cfg)

	err := storage.SaveTunnel(tun)
	if err != nil {
		t.Fatalf("failed to save tunnel: %v", err)
	}

	retrieved, err := storage.GetTunnel(tun.ID)
	if err != nil {
		t.Fatalf("failed to get tunnel: %v", err)
	}

	if retrieved.ID != tun.ID {
		t.Errorf("expected ID %s, got %s", tun.ID, retrieved.ID)
	}

	if retrieved.Name != tun.Name {
		t.Errorf("expected name %s, got %s", tun.Name, retrieved.Name)
	}

	if retrieved.Protocol != tun.Protocol {
		t.Errorf("expected protocol %s, got %s", tun.Protocol, retrieved.Protocol)
	}

	tunnels, err := storage.ListTunnels()
	if err != nil {
		t.Fatalf("failed to list tunnels: %v", err)
	}

	if len(tunnels) != 1 {
		t.Errorf("expected 1 tunnel, got %d", len(tunnels))
	}

	err = storage.DeleteTunnel(tun.ID)
	if err != nil {
		t.Fatalf("failed to delete tunnel: %v", err)
	}

	_, err = storage.GetTunnel(tun.ID)
	if err == nil {
		t.Error("expected error after deletion")
	}
}

func TestMemoryStorageNotFound(t *testing.T) {
	storage := NewMemoryStorage()

	_, err := storage.GetTunnel("nonexistent-id")
	if err == nil {
		t.Error("expected error for nonexistent tunnel")
	}
}

func TestMemoryStorageDeleteNotFound(t *testing.T) {
	storage := NewMemoryStorage()

	err := storage.DeleteTunnel("nonexistent-id")
	if err != nil {
		t.Errorf("expected no error for deleting nonexistent tunnel, got %v", err)
	}
}

func TestMemoryStorageMultipleTunnels(t *testing.T) {
	storage := NewMemoryStorage()

	tunnels := []*tunnel.Tunnel{
		tunnel.NewTunnel(&config.TunnelConfig{Name: "tunnel1", Protocol: "http", LocalAddr: "localhost:3000"}),
		tunnel.NewTunnel(&config.TunnelConfig{Name: "tunnel2", Protocol: "tcp", LocalAddr: "localhost:5432"}),
		tunnel.NewTunnel(&config.TunnelConfig{Name: "tunnel3", Protocol: "websocket", LocalAddr: "localhost:8080"}),
	}

	for _, tun := range tunnels {
		if err := storage.SaveTunnel(tun); err != nil {
			t.Fatalf("failed to save tunnel %s: %v", tun.Name, err)
		}
	}

	list, err := storage.ListTunnels()
	if err != nil {
		t.Fatalf("failed to list tunnels: %v", err)
	}

	if len(list) != 3 {
		t.Errorf("expected 3 tunnels, got %d", len(list))
	}

	for _, tun := range tunnels {
		retrieved, err := storage.GetTunnel(tun.ID)
		if err != nil {
			t.Errorf("failed to get tunnel %s: %v", tun.Name, err)
		}
		if retrieved.Name != tun.Name {
			t.Errorf("expected name %s, got %s", tun.Name, retrieved.Name)
		}
	}
}

func TestMemoryStorageClose(t *testing.T) {
	storage := NewMemoryStorage()

	err := storage.Close()
	if err != nil {
		t.Errorf("expected no error on close, got %v", err)
	}
}

func TestFileStorage(t *testing.T) {
	cfg := config.StorageConfig{
		Type: "memory",
		Path: "/tmp/test.db",
	}

	storage, err := NewFileStorage(cfg)
	if err != nil {
		t.Fatalf("failed to create file storage: %v", err)
	}

	if storage == nil {
		t.Error("expected non-nil storage")
	}

	tun := tunnel.NewTunnel(&config.TunnelConfig{
		Name:      "file-tunnel",
		Protocol:  "http",
		LocalAddr: "localhost:4000",
	})

	err = storage.SaveTunnel(tun)
	if err != nil {
		t.Fatalf("failed to save tunnel: %v", err)
	}

	retrieved, err := storage.GetTunnel(tun.ID)
	if err != nil {
		t.Fatalf("failed to get tunnel: %v", err)
	}

	if retrieved.Name != "file-tunnel" {
		t.Errorf("expected name file-tunnel, got %s", retrieved.Name)
	}
}
