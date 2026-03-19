package mcp

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/a-tunnels/a-tunnels/internal/tunnel"
)

type Server struct {
	addr      string
	tunnelMgr tunnel.Manager
	listener  net.Listener
	authToken string
}

func NewServer(addr string, mgr tunnel.Manager, authToken string) *Server {
	return &Server{
		addr:      addr,
		tunnelMgr: mgr,
		authToken: authToken,
	}
}

type MCPRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      interface{}     `json:"id,omitempty"`
}

type MCPResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	Error   *MCPError   `json:"error,omitempty"`
	ID      interface{} `json:"id,omitempty"`
}

type MCPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type Tool struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	InputSchema any    `json:"inputSchema"`
}

func (s *Server) Start() error {
	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}
	s.listener = ln

	go s.acceptConnections()
	log.Printf("MCP server started on %s", s.addr)
	return nil
}

func (s *Server) acceptConnections() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return
		}
		go s.handleConnection(conn)
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()

	buf := make([]byte, 4096)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			return
		}

		var req MCPRequest
		if err := json.Unmarshal(buf[:n], &req); err != nil {
			continue
		}

		resp := s.handleRequest(req)
		data, _ := json.Marshal(resp)
		conn.Write(data)
	}
}

func (s *Server) handleRequest(req MCPRequest) MCPResponse {
	if s.authToken != "" {
		var params map[string]interface{}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return MCPResponse{
				JSONRPC: "2.0",
				Error:   &MCPError{Code: -32001, Message: "Unauthorized: invalid request"},
				ID:      req.ID,
			}
		}
		token, _ := params["_token"].(string)
		if token == "" {
			return MCPResponse{
				JSONRPC: "2.0",
				Error:   &MCPError{Code: -32001, Message: "Unauthorized: token required"},
				ID:      req.ID,
			}
		}
		if subtle.ConstantTimeCompare([]byte(token), []byte(s.authToken)) != 1 {
			return MCPResponse{
				JSONRPC: "2.0",
				Error:   &MCPError{Code: -32001, Message: "Unauthorized: invalid token"},
				ID:      req.ID,
			}
		}
	}

	switch req.Method {
	case "tools/list":
		return MCPResponse{
			JSONRPC: "2.0",
			Result: map[string]interface{}{
				"tools": []Tool{
					{Name: "list_tunnels", Description: "Liste tous les tunnels actifs", InputSchema: nil},
					{Name: "create_tunnel", Description: "Crée un nouveau tunnel", InputSchema: map[string]string{"type": "object"}},
					{Name: "delete_tunnel", Description: "Supprime un tunnel", InputSchema: map[string]string{"type": "object"}},
					{Name: "get_tunnel_stats", Description: "Obtient les stats d'un tunnel", InputSchema: map[string]string{"type": "object"}},
					{Name: "get_tunnel_logs", Description: "Obtient les logs d'un tunnel", InputSchema: map[string]string{"type": "object"}},
					{Name: "restart_tunnel", Description: "Redémarre un tunnel", InputSchema: map[string]string{"type": "object"}},
				},
			},
			ID: req.ID,
		}

	case "tools/call":
		return s.handleToolCall(req)

	default:
		return MCPResponse{
			JSONRPC: "2.0",
			Error:   &MCPError{Code: -32601, Message: "Method not found"},
			ID:      req.ID,
		}
	}
}

func (s *Server) handleToolCall(req MCPRequest) MCPResponse {
	var params map[string]interface{}
	json.Unmarshal(req.Params, &params)

	toolName, _ := params["name"].(string)

	switch toolName {
	case "list_tunnels":
		tunnels := s.tunnelMgr.List()
		return MCPResponse{
			JSONRPC: "2.0",
			Result:  tunnels,
			ID:      req.ID,
		}

	case "get_tunnel_stats":
		name, _ := params["arguments"].(map[string]interface{})["name"].(string)
		stats, err := s.tunnelMgr.GetStats(name)
		if err != nil {
			return MCPResponse{
				JSONRPC: "2.0",
				Error:   &MCPError{Code: -32602, Message: err.Error()},
				ID:      req.ID,
			}
		}
		return MCPResponse{
			JSONRPC: "2.0",
			Result:  stats,
			ID:      req.ID,
		}

	default:
		return MCPResponse{
			JSONRPC: "2.0",
			Error:   &MCPError{Code: -32601, Message: "Tool not found"},
			ID:      req.ID,
		}
	}
}

func (s *Server) Stop() error {
	if s.listener != nil {
		return s.listener.Close()
	}
	return nil
}

func MCPTools() []Tool {
	return []Tool{
		{
			Name:        "list_tunnels",
			Description: "Liste tous les tunnels actifs",
			InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
		},
		{
			Name:        "create_tunnel",
			Description: "Crée un nouveau tunnel",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name":      map[string]any{"type": "string"},
					"protocol":  map[string]any{"type": "string", "enum": []string{"http", "tcp", "websocket"}},
					"localAddr": map[string]any{"type": "string"},
				},
				"required": []string{"name", "protocol", "localAddr"},
			},
		},
		{
			Name:        "delete_tunnel",
			Description: "Supprime un tunnel par son nom",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name": map[string]any{"type": "string"},
				},
				"required": []string{"name"},
			},
		},
		{
			Name:        "get_tunnel_stats",
			Description: "Affiche les statistiques d'un tunnel",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name": map[string]any{"type": "string"},
				},
				"required": []string{"name"},
			},
		},
		{
			Name:        "get_tunnel_logs",
			Description: "Affiche les logs d'un tunnel",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name":  map[string]any{"type": "string"},
					"lines": map[string]any{"type": "number", "default": 100},
				},
				"required": []string{"name"},
			},
		},
		{
			Name:        "restart_tunnel",
			Description: "Redémarre un tunnel",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name": map[string]any{"type": "string"},
				},
				"required": []string{"name"},
			},
		},
	}
}

func init() {
	_ = time.Now()
	_ = fmt.Sprintf("")
}
