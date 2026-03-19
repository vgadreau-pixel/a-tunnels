package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	TunnelRequests = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "atunnels_tunnel_requests_total",
			Help: "Total number of requests",
		},
		[]string{"tunnel", "protocol"},
	)

	TunnelBytesIn = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "atunnels_tunnel_bytes_in_total",
			Help: "Total bytes received",
		},
		[]string{"tunnel"},
	)

	TunnelBytesOut = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "atunnels_tunnel_bytes_out_total",
			Help: "Total bytes sent",
		},
		[]string{"tunnel"},
	)

	TunnelConnections = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "atunnels_tunnel_connections",
			Help: "Active connections",
		},
		[]string{"tunnel"},
	)

	ServerRequests = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "atunnels_server_requests_total",
			Help: "Total server requests",
		},
		[]string{"endpoint", "method"},
	)

	ServerErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "atunnels_server_errors_total",
			Help: "Total server errors",
		},
		[]string{"type"},
	)
)

func IncTunnelRequests(tunnel, protocol string) {
	TunnelRequests.WithLabelValues(tunnel, protocol).Inc()
}

func IncTunnelBytesIn(tunnel string, bytes int64) {
	TunnelBytesIn.WithLabelValues(tunnel).Add(float64(bytes))
}

func IncTunnelBytesOut(tunnel string, bytes int64) {
	TunnelBytesOut.WithLabelValues(tunnel).Add(float64(bytes))
}

func SetTunnelConnections(tunnel string, conns int) {
	TunnelConnections.WithLabelValues(tunnel).Set(float64(conns))
}

func IncServerRequests(endpoint, method string) {
	ServerRequests.WithLabelValues(endpoint, method).Inc()
}

func IncServerErrors(errType string) {
	ServerErrors.WithLabelValues(errType).Inc()
}
