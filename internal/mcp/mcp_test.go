package mcp

import (
	"encoding/json"
	"testing"

	"github.com/a-tunnels/a-tunnels/internal/tunnel"
)

func TestMCPRequest(t *testing.T) {
	tests := []struct {
		name         string
		jsonReq      string
		wantMethod   string
		wantParseErr bool
	}{
		{
			name:         "tools/list request",
			jsonReq:      `{"jsonrpc":"2.0","method":"tools/list","id":1}`,
			wantMethod:   "tools/list",
			wantParseErr: false,
		},
		{
			name:         "tools/call request",
			jsonReq:      `{"jsonrpc":"2.0","method":"tools/call","params":{"name":"list_tunnels"},"id":2}`,
			wantMethod:   "tools/call",
			wantParseErr: false,
		},
		{
			name:         "unknown method",
			jsonReq:      `{"jsonrpc":"2.0","method":"unknown","id":3}`,
			wantMethod:   "unknown",
			wantParseErr: false,
		},
		{
			name:         "invalid json",
			jsonReq:      `not json`,
			wantMethod:   "",
			wantParseErr: true,
		},
	}

	mgr := tunnel.NewManager()
	_ = mgr

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req MCPRequest
			err := json.Unmarshal([]byte(tt.jsonReq), &req)

			if (err != nil) != tt.wantParseErr {
				t.Errorf("parse error = %v, wantParseErr %v", err, tt.wantParseErr)
				return
			}

			if !tt.wantParseErr && err == nil && req.Method != tt.wantMethod {
				t.Errorf("method = %s, want %s", req.Method, tt.wantMethod)
			}
		})
	}
}

func TestMCPResponse(t *testing.T) {
	resp := MCPResponse{
		JSONRPC: "2.0",
		Result:  map[string]interface{}{"status": "ok"},
		ID:      1,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal response: %v", err)
	}

	var parsed MCPResponse
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if parsed.JSONRPC != "2.0" {
		t.Errorf("expected jsonrpc 2.0, got %s", parsed.JSONRPC)
	}

	id, ok := parsed.ID.(float64)
	if !ok || int(id) != 1 {
		t.Errorf("expected id 1, got %v", parsed.ID)
	}

	resultMap, ok := parsed.Result.(map[string]interface{})
	if !ok {
		t.Fatal("expected result to be map")
	}

	if resultMap["status"] != "ok" {
		t.Errorf("expected status ok, got %v", resultMap["status"])
	}
}

func TestMCPErrorResponse(t *testing.T) {
	resp := MCPResponse{
		JSONRPC: "2.0",
		Error:   &MCPError{Code: -32601, Message: "Method not found"},
		ID:      1,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal error response: %v", err)
	}

	var parsed MCPResponse
	json.Unmarshal(data, &parsed)

	if parsed.Error == nil {
		t.Fatal("expected error in response")
	}

	if parsed.Error.Code != -32601 {
		t.Errorf("expected error code -32601, got %d", parsed.Error.Code)
	}

	if parsed.Error.Message != "Method not found" {
		t.Errorf("expected error message, got %s", parsed.Error.Message)
	}
}

func TestMCPTools(t *testing.T) {
	tools := MCPTools()

	if len(tools) != 6 {
		t.Errorf("expected 6 tools, got %d", len(tools))
	}

	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolNames[tool.Name] = true

		if tool.Description == "" {
			t.Errorf("tool %s has empty description", tool.Name)
		}
	}

	expectedTools := []string{
		"list_tunnels",
		"create_tunnel",
		"delete_tunnel",
		"get_tunnel_stats",
		"get_tunnel_logs",
		"restart_tunnel",
	}

	for _, name := range expectedTools {
		if !toolNames[name] {
			t.Errorf("expected tool %s not found", name)
		}
	}
}

func TestHandleToolsList(t *testing.T) {
	mgr := tunnel.NewManager()
	server := NewServer("localhost:0", mgr, "")

	req := MCPRequest{
		JSONRPC: "2.0",
		Method:  "tools/list",
		ID:      1,
	}

	resp := server.handleRequest(req)

	if resp.JSONRPC != "2.0" {
		t.Errorf("expected jsonrpc 2.0, got %s", resp.JSONRPC)
	}

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

func TestHandleToolCall(t *testing.T) {
	mgr := tunnel.NewManager()
	server := NewServer("localhost:0", mgr, "")

	params := map[string]interface{}{
		"name": "test-tunnel",
	}
	paramsBytes, _ := json.Marshal(params)

	req := MCPRequest{
		JSONRPC: "2.0",
		Method:  "tools/call",
		Params:  paramsBytes,
		ID:      1,
	}

	resp := server.handleRequest(req)

	if resp.Error != nil {
		t.Logf("Expected error for non-existent tunnel: %v", resp.Error)
	}

	req2 := MCPRequest{
		JSONRPC: "2.0",
		Method:  "tools/call",
		Params:  json.RawMessage(`{"name":"list_tunnels"}`),
		ID:      2,
	}

	resp2 := server.handleRequest(req2)
	if resp2.Error != nil {
		t.Errorf("unexpected error: %v", resp2.Error)
	}
}

func TestHandleUnknownMethod(t *testing.T) {
	mgr := tunnel.NewManager()
	server := NewServer("localhost:0", mgr, "")

	req := MCPRequest{
		JSONRPC: "2.0",
		Method:  "unknown_method",
		ID:      1,
	}

	resp := server.handleRequest(req)

	if resp.Error == nil {
		t.Error("expected error for unknown method")
	}

	if resp.Error.Code != -32601 {
		t.Errorf("expected error code -32601, got %d", resp.Error.Code)
	}
}
