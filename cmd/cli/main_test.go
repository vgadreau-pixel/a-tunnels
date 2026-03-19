package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	atunnels "github.com/a-tunnels/a-tunnels/pkg/sdk/go"
)

func TestListTunnelsJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/tunnels" {
			t.Errorf("expected /api/v1/tunnels, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"id":"123","name":"test","protocol":"http","localAddr":"localhost:3000","status":"active"}]`))
	}))
	defer server.Close()

	client := atunnels.NewClient(server.URL, "test-token")
	err := listTunnels(client)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestListTunnelsEmpty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[]`))
	}))
	defer server.Close()

	client := atunnels.NewClient(server.URL, "test-token")
	err := listTunnels(client)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGetTunnel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/api/v1/tunnels/my-tunnel") {
			t.Errorf("expected path containing /api/v1/tunnels/my-tunnel, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id":"123","name":"my-tunnel","protocol":"http","localAddr":"localhost:3000","status":"active"}]`))
	}))
	defer server.Close()

	client := atunnels.NewClient(server.URL, "test-token")
	err := getTunnel(client, []string{"my-tunnel"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGetTunnelNoArgs(t *testing.T) {
	client := atunnels.NewClient("http://localhost:8080", "test-token")
	err := getTunnel(client, []string{})
	if err == nil {
		t.Error("expected error for missing args")
	}
}

func TestCreateTunnel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":"456","name":"new-tunnel","protocol":"http","localAddr":"localhost:4000","status":"active"}`))
	}))
	defer server.Close()

	client := atunnels.NewClient(server.URL, "test-token")
	err := createTunnel(client, []string{"new-tunnel", "http", "localhost:4000"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCreateTunnelMissingArgs(t *testing.T) {
	client := atunnels.NewClient("http://localhost:8080", "test-token")
	err := createTunnel(client, []string{"new-tunnel"})
	if err == nil {
		t.Error("expected error for missing args")
	}
}

func TestDeleteTunnel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := atunnels.NewClient(server.URL, "test-token")
	err := deleteTunnel(client, []string{"my-tunnel"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestDeleteTunnelNoArgs(t *testing.T) {
	client := atunnels.NewClient("http://localhost:8080", "test-token")
	err := deleteTunnel(client, []string{})
	if err == nil {
		t.Error("expected error for missing args")
	}
}

func TestGetStats(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/stats") {
			t.Errorf("expected /stats in path, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"activeConnections":5,"totalRequests":100,"totalBytesIn":1024,"totalBytesOut":2048}`))
	}))
	defer server.Close()

	client := atunnels.NewClient(server.URL, "test-token")
	err := getStats(client, []string{"my-tunnel"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGetStatsNoArgs(t *testing.T) {
	client := atunnels.NewClient("http://localhost:8080", "test-token")
	err := getStats(client, []string{})
	if err == nil {
		t.Error("expected error for missing args")
	}
}

func TestRestartTunnel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/restart") {
			t.Errorf("expected /restart in path, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := atunnels.NewClient(server.URL, "test-token")
	err := restartTunnel(client, []string{"my-tunnel"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRestartTunnelNoArgs(t *testing.T) {
	client := atunnels.NewClient("http://localhost:8080", "test-token")
	err := restartTunnel(client, []string{})
	if err == nil {
		t.Error("expected error for missing args")
	}
}

func TestCheckHealth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	client := atunnels.NewClient(server.URL, "test-token")
	err := checkHealth(client)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCheckHealthUnhealthy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := atunnels.NewClient(server.URL, "test-token")
	err := checkHealth(client)
	if err == nil {
		t.Error("expected error for unhealthy server")
	}
}

func TestMin(t *testing.T) {
	if min(1, 5) != 1 {
		t.Errorf("expected 1, got %d", min(1, 5))
	}
	if min(5, 1) != 1 {
		t.Errorf("expected 1, got %d", min(5, 1))
	}
	if min(3, 3) != 3 {
		t.Errorf("expected 3, got %d", min(3, 3))
	}
}
