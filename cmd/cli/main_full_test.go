package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	atunnels "github.com/a-tunnels/a-tunnels/pkg/sdk/go"
)

func TestListTunnelsMultiple(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[
			{"id":"1","name":"tunnel1","protocol":"http","localAddr":"localhost:3000","status":"active"},
			{"id":"2","name":"tunnel2","protocol":"tcp","localAddr":"localhost:5432","status":"stopped"},
			{"id":"3","name":"tunnel3","protocol":"websocket","localAddr":"localhost:8080","status":"active"}
		]`))
	}))
	defer server.Close()

	client := atunnels.NewClient(server.URL, "test-token")
	err := listTunnels(client)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGetTunnelWithSubdomain(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id":"123","name":"my-tunnel","protocol":"http","localAddr":"localhost:3000","subdomain":"myapp","status":"active"}`))
	}))
	defer server.Close()

	client := atunnels.NewClient(server.URL, "test-token")
	err := getTunnel(client, []string{"my-tunnel"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCreateTunnelWithSubdomain(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":"456","name":"new-tunnel","protocol":"http","localAddr":"localhost:4000","subdomain":"newapp","status":"active"}`))
	}))
	defer server.Close()

	client := atunnels.NewClient(server.URL, "test-token")
	err := createTunnel(client, []string{"new-tunnel", "http", "localhost:4000"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestDeleteTunnelMultiple(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := atunnels.NewClient(server.URL, "test-token")

	err := deleteTunnel(client, []string{"tunnel1"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	err = deleteTunnel(client, []string{"tunnel2"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGetStatsMultiple(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"activeConnections":10,"totalRequests":500,"totalBytesIn":10240,"totalBytesOut":20480}`))
	}))
	defer server.Close()

	client := atunnels.NewClient(server.URL, "test-token")

	err := getStats(client, []string{"tunnel1"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	err = getStats(client, []string{"tunnel2"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRestartTunnelMultiple(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := atunnels.NewClient(server.URL, "test-token")

	err := restartTunnel(client, []string{"tunnel1"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	err = restartTunnel(client, []string{"tunnel2"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCheckHealthMultiple(t *testing.T) {
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

func TestVersion(t *testing.T) {
	showVersion()
}

func TestMinValues(t *testing.T) {
	if min(0, 5) != 0 {
		t.Errorf("expected 0, got %d", min(0, 5))
	}
	if min(10, 0) != 0 {
		t.Errorf("expected 0, got %d", min(10, 0))
	}
	if min(5, 5) != 5 {
		t.Errorf("expected 5, got %d", min(5, 5))
	}
	if min(100, 200) != 100 {
		t.Errorf("expected 100, got %d", min(100, 200))
	}
}
