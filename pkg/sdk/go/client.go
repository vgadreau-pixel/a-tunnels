package atunnels

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	ServerURL string
	Token     string
	HTTP      *http.Client
}

type Tunnel struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Protocol   string            `json:"protocol"`
	LocalAddr  string            `json:"local_addr"`
	Subdomain  string            `json:"subdomain,omitempty"`
	RemotePort int               `json:"remote_port,omitempty"`
	Status     string            `json:"status"`
	Headers    map[string]string `json:"headers,omitempty"`
}

type TunnelStats struct {
	ActiveConnections int64  `json:"active_connections"`
	TotalRequests     int64  `json:"total_requests"`
	TotalBytesIn      int64  `json:"total_bytes_in"`
	TotalBytesOut     int64  `json:"total_bytes_out"`
	LastRequestAt     string `json:"last_request_at"`
}

func NewClient(serverURL, token string) *Client {
	return &Client{
		ServerURL: serverURL,
		Token:     token,
		HTTP: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) doRequest(method, path string, reqBody interface{}) ([]byte, error) {
	var payload []byte
	if reqBody != nil {
		payload, _ = json.Marshal(reqBody)
	}

	req, err := http.NewRequest(method, c.ServerURL+path, bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("request failed with status: %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (c *Client) ListTunnels() ([]Tunnel, error) {
	data, err := c.doRequest("GET", "/api/v1/tunnels", nil)
	if err != nil {
		return nil, err
	}

	var tunnels []Tunnel
	json.Unmarshal(data, &tunnels)
	return tunnels, nil
}

func (c *Client) GetTunnel(name string) (*Tunnel, error) {
	data, err := c.doRequest("GET", "/api/v1/tunnels/"+name, nil)
	if err != nil {
		return nil, err
	}

	var tunnel Tunnel
	json.Unmarshal(data, &tunnel)
	return &tunnel, nil
}

func (c *Client) CreateTunnel(tunnel *Tunnel) (*Tunnel, error) {
	data, err := c.doRequest("POST", "/api/v1/tunnels", tunnel)
	if err != nil {
		return nil, err
	}

	var created Tunnel
	json.Unmarshal(data, &created)
	return &created, nil
}

func (c *Client) DeleteTunnel(name string) error {
	_, err := c.doRequest("DELETE", "/api/v1/tunnels/"+name, nil)
	return err
}

func (c *Client) GetTunnelStats(name string) (*TunnelStats, error) {
	data, err := c.doRequest("GET", "/api/v1/tunnels/"+name+"/stats", nil)
	if err != nil {
		return nil, err
	}

	var stats TunnelStats
	json.Unmarshal(data, &stats)
	return &stats, nil
}

func (c *Client) RestartTunnel(name string) error {
	_, err := c.doRequest("POST", "/api/v1/tunnels/"+name+"/restart", nil)
	return err
}

func (c *Client) Health() (bool, error) {
	data, err := c.doRequest("GET", "/health", nil)
	if err != nil {
		return false, err
	}
	return bytes.Contains(data, []byte("ok")), nil
}
