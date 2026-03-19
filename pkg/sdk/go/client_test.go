package atunnels

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewClient(t *testing.T) {
	client := NewClient("http://localhost:8080", "test-token")

	if client.ServerURL != "http://localhost:8080" {
		t.Errorf("expected server URL http://localhost:8080, got %s", client.ServerURL)
	}

	if client.Token != "test-token" {
		t.Errorf("expected token test-token, got %s", client.Token)
	}

	if client.HTTP == nil {
		t.Error("expected non-nil HTTP client")
	}
}

func TestClientListTunnels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET method, got %s", r.Method)
		}

		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-token" {
			t.Errorf("expected Bearer test-token, got %s", auth)
		}

		tunnels := []Tunnel{
			{ID: "1", Name: "tunnel1", Protocol: "http", LocalAddr: "localhost:3000", Status: "active"},
			{ID: "2", Name: "tunnel2", Protocol: "tcp", LocalAddr: "localhost:5432", Status: "active"},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tunnels)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	tunnels, err := client.ListTunnels()

	if err != nil {
		t.Fatalf("failed to list tunnels: %v", err)
	}

	if len(tunnels) != 2 {
		t.Errorf("expected 2 tunnels, got %d", len(tunnels))
	}

	if tunnels[0].Name != "tunnel1" {
		t.Errorf("expected first tunnel name tunnel1, got %s", tunnels[0].Name)
	}

	if tunnels[1].Protocol != "tcp" {
		t.Errorf("expected second tunnel protocol tcp, got %s", tunnels[1].Protocol)
	}
}

func TestClientGetTunnel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/tunnels/my-tunnel" {
			t.Errorf("expected path /api/v1/tunnels/my-tunnel, got %s", r.URL.Path)
		}

		tunnel := Tunnel{
			ID:        "123",
			Name:      "my-tunnel",
			Protocol:  "http",
			LocalAddr: "localhost:3000",
			Subdomain: "myapp",
			Status:    "active",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tunnel)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	tunnel, err := client.GetTunnel("my-tunnel")

	if err != nil {
		t.Fatalf("failed to get tunnel: %v", err)
	}

	if tunnel.Name != "my-tunnel" {
		t.Errorf("expected name my-tunnel, got %s", tunnel.Name)
	}

	if tunnel.Subdomain != "myapp" {
		t.Errorf("expected subdomain myapp, got %s", tunnel.Subdomain)
	}
}

func TestClientCreateTunnel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST method, got %s", r.Method)
		}

		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		var req Tunnel
		json.NewDecoder(r.Body).Decode(&req)

		if req.Name != "new-tunnel" {
			t.Errorf("expected name new-tunnel, got %s", req.Name)
		}

		if req.Protocol != "http" {
			t.Errorf("expected protocol http, got %s", req.Protocol)
		}

		if req.LocalAddr != "localhost:5000" {
			t.Errorf("expected localAddr localhost:5000, got %s", req.LocalAddr)
		}

		created := Tunnel{
			ID:        "new-id",
			Name:      req.Name,
			Protocol:  req.Protocol,
			LocalAddr: req.LocalAddr,
			Status:    "active",
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(created)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	tunnel, err := client.CreateTunnel(&Tunnel{
		Name:      "new-tunnel",
		Protocol:  "http",
		LocalAddr: "localhost:5000",
	})

	if err != nil {
		t.Fatalf("failed to create tunnel: %v", err)
	}

	if tunnel.ID != "new-id" {
		t.Errorf("expected ID new-id, got %s", tunnel.ID)
	}

	if tunnel.Status != "active" {
		t.Errorf("expected status active, got %s", tunnel.Status)
	}
}

func TestClientDeleteTunnel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE method, got %s", r.Method)
		}

		if r.URL.Path != "/api/v1/tunnels/delete-me" {
			t.Errorf("expected path /api/v1/tunnels/delete-me, got %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	err := client.DeleteTunnel("delete-me")

	if err != nil {
		t.Fatalf("failed to delete tunnel: %v", err)
	}
}

func TestClientGetTunnelStats(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		stats := TunnelStats{
			ActiveConnections: 5,
			TotalRequests:     100,
			TotalBytesIn:      1024,
			TotalBytesOut:     2048,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(stats)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	stats, err := client.GetTunnelStats("my-tunnel")

	if err != nil {
		t.Fatalf("failed to get stats: %v", err)
	}

	if stats.ActiveConnections != 5 {
		t.Errorf("expected 5 active connections, got %d", stats.ActiveConnections)
	}

	if stats.TotalRequests != 100 {
		t.Errorf("expected 100 total requests, got %d", stats.TotalRequests)
	}

	if stats.TotalBytesIn != 1024 {
		t.Errorf("expected 1024 bytes in, got %d", stats.TotalBytesIn)
	}

	if stats.TotalBytesOut != 2048 {
		t.Errorf("expected 2048 bytes out, got %d", stats.TotalBytesOut)
	}
}

func TestClientRestartTunnel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST method, got %s", r.Method)
		}

		if r.URL.Path != "/api/v1/tunnels/my-tunnel/restart" {
			t.Errorf("expected path /api/v1/tunnels/my-tunnel/restart, got %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	err := client.RestartTunnel("my-tunnel")

	if err != nil {
		t.Fatalf("failed to restart tunnel: %v", err)
	}
}

func TestClientHealth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	healthy, err := client.Health()

	if err != nil {
		t.Fatalf("failed to check health: %v", err)
	}

	if !healthy {
		t.Error("expected health to be true")
	}
}

func TestClientHealthFail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	healthy, err := client.Health()

	if err == nil {
		t.Error("expected error for failed health check")
	}

	if healthy {
		t.Error("expected health to be false")
	}
}

func TestClientErrorHandling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	_, err := client.GetTunnel("nonexistent")

	if err == nil {
		t.Error("expected error for 404 response")
	}
}
