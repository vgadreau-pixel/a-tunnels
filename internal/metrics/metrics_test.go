package metrics

import (
	"testing"
)

func TestIncTunnelRequests(t *testing.T) {
	IncTunnelRequests("test-tunnel", "http")
}

func TestIncTunnelBytesIn(t *testing.T) {
	IncTunnelBytesIn("test-tunnel", 1024)
}

func TestIncTunnelBytesOut(t *testing.T) {
	IncTunnelBytesOut("test-tunnel", 2048)
}

func TestSetTunnelConnections(t *testing.T) {
	SetTunnelConnections("test-tunnel", 5)
}

func TestIncServerRequests(t *testing.T) {
	IncServerRequests("/api/tunnels", "GET")
}

func TestIncServerErrors(t *testing.T) {
	IncServerErrors("connection_error")
}

func TestMetricsMultipleTunnels(t *testing.T) {
	IncTunnelRequests("tunnel1", "http")
	IncTunnelRequests("tunnel2", "tcp")
	IncTunnelRequests("tunnel1", "http")

	IncTunnelBytesIn("tunnel1", 100)
	IncTunnelBytesIn("tunnel2", 200)

	IncTunnelBytesOut("tunnel1", 150)
	IncTunnelBytesOut("tunnel2", 250)

	SetTunnelConnections("tunnel1", 3)
	SetTunnelConnections("tunnel2", 7)

	IncServerRequests("/api/v1/tunnels", "GET")
	IncServerRequests("/api/v1/tunnels", "POST")
	IncServerRequests("/api/v1/tunnels", "DELETE")

	IncServerErrors("timeout")
	IncServerErrors("connection_refused")
}
