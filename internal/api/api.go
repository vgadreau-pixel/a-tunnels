package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/a-tunnels/a-tunnels/internal/config"
	"github.com/a-tunnels/a-tunnels/internal/tunnel"
	"github.com/gorilla/mux"
)

type API struct {
	server    *http.Server
	tunnelMgr tunnel.Manager
	auth      *AuthMiddleware
	config    *config.ServerConfig
}

type Handler func(w http.ResponseWriter, r *http.Request) error

func NewAPI(mgr tunnel.Manager, cfg *config.ServerConfig) *API {
	api := &API{
		tunnelMgr: mgr,
		config:    cfg,
		auth:      NewAuthMiddleware(cfg.Auth),
	}

	router := mux.NewRouter()
	router.Use(api.auth.Middleware)
	router.HandleFunc("/health", api.handleHealth)
	router.HandleFunc("/metrics", api.handleMetrics)

	api.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.APIPort),
		Handler: router,
	}

	return api
}

func (a *API) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (a *API) handleMetrics(w http.ResponseWriter, r *http.Request) {
	tunnels := a.tunnelMgr.List()
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, "atunnels_tunnels %d\n", len(tunnels))
	for _, t := range tunnels {
		stats := t.GetStats()
		fmt.Fprintf(w, "atunnels_tunnel_requests{tunnel=\"%s\"} %d\n", t.Name, stats.TotalRequests)
	}
}

func (a *API) Start() error {
	return a.server.ListenAndServe()
}

func (a *API) Stop() error {
	return a.server.Shutdown(nil)
}
