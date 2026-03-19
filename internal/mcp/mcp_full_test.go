package mcp

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/a-tunnels/a-tunnels/internal/config"
	"github.com/a-tunnels/a-tunnels/internal/tunnel"
)

func TestMCPFullProtocol(t *testing.T) {
	mgr := tunnel.NewManager()
	ctx := context.Background()

	mgr.Create(ctx, &config.TunnelConfig{
		Name:      "test-tunnel",
		Protocol:  "http",
		LocalAddr: "localhost:3000",
	})

	server := NewServer("localhost:0", mgr, "")

	req := MCPRequest{
		JSONRPC: "2.0",
		Method:  "tools/list",
		ID:      1,
	}

	resp := server.handleRequest(req)
	if resp.Error != nil {
		t.Errorf("unexpected error: %v", resp.Error)
	}

	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatal("expected result to be map")
	}

	tools, ok := result["tools"].([]Tool)
	if !ok {
		t.Fatal("expected tools in result")
	}

	if len(tools) != 6 {
		t.Errorf("expected 6 tools, got %d", len(tools))
	}
}

func TestMCPErrorCodes(t *testing.T) {
	mgr := tunnel.NewManager()
	server := NewServer("localhost:0", mgr, "")

	tests := []struct {
		method string
		want   int
	}{
		{"tools/call", -32601},
		{"unknown", -32601},
		{"", -32601},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			req := MCPRequest{
				JSONRPC: "2.0",
				Method:  tt.method,
				ID:      1,
			}
			resp := server.handleRequest(req)
			if resp.Error == nil {
				t.Error("expected error")
			}
			if resp.Error.Code != tt.want {
				t.Errorf("got %d, want %d", resp.Error.Code, tt.want)
			}
		})
	}
}

func TestMCPToolsDefinitionsComplete(t *testing.T) {
	tools := MCPTools()

	expected := map[string]bool{
		"list_tunnels":     true,
		"create_tunnel":    true,
		"delete_tunnel":    true,
		"get_tunnel_stats": true,
		"get_tunnel_logs":  true,
		"restart_tunnel":   true,
	}

	if len(tools) != len(expected) {
		t.Errorf("expected %d tools, got %d", len(expected), len(tools))
	}

	for _, tool := range tools {
		if !expected[tool.Name] {
			t.Errorf("unexpected tool: %s", tool.Name)
		}
		if tool.Description == "" {
			t.Error("description cannot be empty")
		}
	}
}

func TestMCPWithTunnelManager(t *testing.T) {
	mgr := tunnel.NewManager()
	ctx := context.Background()

	tunnels := []struct {
		name     string
		protocol string
		addr     string
	}{
		{"http-tunnel", "http", "localhost:3000"},
		{"tcp-tunnel", "tcp", "localhost:5432"},
		{"ws-tunnel", "websocket", "localhost:8080"},
	}

	for _, tt := range tunnels {
		mgr.Create(ctx, &config.TunnelConfig{
			Name:      tt.name,
			Protocol:  tt.protocol,
			LocalAddr: tt.addr,
		})
	}

	list := mgr.List()
	if len(list) != 3 {
		t.Errorf("expected 3 tunnels, got %d", len(list))
	}
}

func TestMCPRequestVariants(t *testing.T) {
	tests := []struct {
		name string
		json string
	}{
		{"with string id", `{"jsonrpc":"2.0","method":"tools/list","id":"abc"}`},
		{"with null id", `{"jsonrpc":"2.0","method":"tools/list","id":null}`},
		{"with float id", `{"jsonrpc":"2.0","method":"tools/list","id":1.5}`},
		{"empty params", `{"jsonrpc":"2.0","method":"tools/list","params":{}}`},
		{"array params", `{"jsonrpc":"2.0","method":"tools/list","params":[]}`},
		{"null params", `{"jsonrpc":"2.0","method":"tools/list","params":null}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req MCPRequest
			err := json.Unmarshal([]byte(tt.json), &req)
			if err != nil {
				t.Errorf("unexpected parse error: %v", err)
			}
		})
	}
}

func init() {
	_ = context.Background()
	_ = config.TunnelConfig{}
}
