package gateway

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/a-tunnels/a-tunnels/internal/shortener"
	"github.com/a-tunnels/a-tunnels/internal/tunnel"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: checkOrigin,
}

type RateLimiter struct {
	visitors map[string]*Visitor
	mutex    sync.RWMutex
	limit    int
	window   time.Duration
}

type Visitor struct {
	Requests int
	LastSeen time.Time
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]*Visitor),
		limit:    limit,
		window:   window,
	}

	// Start cleanup goroutine to remove old entries
	go rl.cleanup()

	return rl
}

func (rl *RateLimiter) Allow(ip string) bool {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()

	visitor, exists := rl.visitors[ip]
	if !exists {
		// New visitor
		rl.visitors[ip] = &Visitor{
			Requests: 1,
			LastSeen: now,
		}
		return true
	}

	// Check if we need to reset counters (window passed)
	if now.Sub(visitor.LastSeen) > rl.window {
		visitor.Requests = 1
		visitor.LastSeen = now
		return true
	}

	// Check if limit exceeded
	if visitor.Requests >= rl.limit {
		// Limit exceeded
		return false
	}

	// Update counters
	visitor.Requests++
	visitor.LastSeen = now
	return true
}

func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.mutex.Lock()
		now := time.Now()

		// Remove visitors that haven't been seen in a window period
		for ip, visitor := range rl.visitors {
			if now.Sub(visitor.LastSeen) > rl.window {
				delete(rl.visitors, ip)
			}
		}

		rl.mutex.Unlock()
	}
}

type Gateway struct {
	httpServer     *http.Server
	httpsServer    *http.Server
	tcpListener    net.Listener
	wsServer       *http.Server
	tunnelMgr      tunnel.Manager
	shortener      *shortener.Shortener
	config         *GatewayConfig
	rateLimiter    *RateLimiter
	shortenLimiter *RateLimiter // Separate limiter for URL shortening to prevent abuse
	connections    map[string]*ClientConnection
	mu             sync.RWMutex
}

type GatewayConfig struct {
	HTTPPort           int
	HTTPSPort          int
	TCPPort            int
	WSPort             int
	Domain             string
	TLSCert            string
	TLSKey             string
	AllowedOrigins     []string
	AllowedIPs         []string
	AuthToken          string
	RateLimit          int // Requests per time window
	ShortenerRateLimit int // Requests per time window (typically lower)
	Shortener          GatewayShortenerConfig
}

type GatewayShortenerConfig struct {
	Enabled     bool
	DefaultTTL  int
	MaxTTL      int
	MaxLength   int
	BasePath    string
	CleanupFreq int
}

type ClientConnection struct {
	ID        string
	TunnelID  string
	LocalAddr string
	Conn      net.Conn
	CreatedAt time.Time
}

func NewGateway(cfg *GatewayConfig, mgr tunnel.Manager) *Gateway {
	s := shortener.New()

	// Default to 100/minute for general API requests and 20/hour for URL shortening
	rateWindow := time.Minute
	if cfg.Shortener.Enabled && cfg.Shortener.CleanupFreq > 0 {
		rateWindow = time.Duration(cfg.Shortener.CleanupFreq) * time.Minute
	}

	g := &Gateway{
		tunnelMgr:      mgr,
		shortener:      s,
		config:         cfg,
		rateLimiter:    NewRateLimiter(cfg.RateLimit, rateWindow),
		shortenLimiter: NewRateLimiter(cfg.ShortenerRateLimit, time.Hour), // Fewer creates per hour for shortening
		connections:    make(map[string]*ClientConnection),
	}

	if cfg.Shortener.CleanupFreq > 0 {
		go g.startCleanupTicker()
	}

	return g
}

func NewGatewayWithStorage(cfg *GatewayConfig, mgr tunnel.Manager, storage shortener.Storage) *Gateway {
	s := shortener.NewWithStorage(storage)

	// Default to 100/minute for general API requests and 20/hour for URL shortening
	rateWindow := time.Minute
	if cfg.Shortener.Enabled && cfg.Shortener.CleanupFreq > 0 {
		rateWindow = time.Duration(cfg.Shortener.CleanupFreq) * time.Minute
	}

	g := &Gateway{
		tunnelMgr:      mgr,
		shortener:      s,
		config:         cfg,
		rateLimiter:    NewRateLimiter(cfg.RateLimit, rateWindow),
		shortenLimiter: NewRateLimiter(cfg.ShortenerRateLimit, time.Hour), // Fewer creates per hour for shortening
		connections:    make(map[string]*ClientConnection),
	}

	if cfg.Shortener.CleanupFreq > 0 {
		go g.startCleanupTicker()
	}

	return g
}

func (g *Gateway) startCleanupTicker() {
	if g.config.Shortener.CleanupFreq <= 0 {
		return
	}

	ticker := time.NewTicker(time.Duration(g.config.Shortener.CleanupFreq) * time.Minute)
	go func() {
		for range ticker.C {
			g.shortener.Cleanup()
		}
	}()
}

func (g *Gateway) StartHTTP(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", g.handleHTTPRequestRateLimited) // Use rate-limited version
	mux.HandleFunc("/health", g.handleHealth)
	mux.HandleFunc("/metrics", g.handleMetricsRateLimited)     // Protect metrics with rate limiting
	mux.HandleFunc("/s/", g.handleShortURLRedirectRateLimited) // Use rate-limited redirect
	mux.HandleFunc("/api/shorten", g.handleShortenURL)

	g.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", g.config.HTTPPort),
		Handler: mux,
	}

	go func() {
		if err := g.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	log.Printf("HTTP gateway started on :%d", g.config.HTTPPort)
	return nil
}

func (g *Gateway) StartHTTPS(ctx context.Context) error {
	if g.config.TLSCert == "" || g.config.TLSKey == "" {
		log.Printf("HTTPS not configured, skipping")
		return nil
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", g.handleHTTPRequestRateLimited) // Use rate-limited version
	mux.HandleFunc("/health", g.handleHealth)
	mux.HandleFunc("/metrics", g.handleMetricsRateLimited)     // Protect metrics with rate limiting
	mux.HandleFunc("/s/", g.handleShortURLRedirectRateLimited) // Use rate-limited redirect
	mux.HandleFunc("/api/shorten", g.handleShortenURL)

	g.httpsServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", g.config.HTTPSPort),
		Handler: mux,
	}

	go func() {
		if err := g.httpsServer.ListenAndServeTLS(g.config.TLSCert, g.config.TLSKey); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTPS server error: %v", err)
		}
	}()

	log.Printf("HTTPS gateway started on :%d", g.config.HTTPSPort)
	return nil
}

func (g *Gateway) handleHTTPRequestRateLimited(w http.ResponseWriter, r *http.Request) {
	clientIP := getClientIP(r)
	if !g.rateLimiter.Allow(clientIP) {
		http.Error(w, "Rate limit exceeded, try again later", http.StatusTooManyRequests)
		return
	}

	g.handleHTTPRequest(w, r)
}

func (g *Gateway) handleMetricsRateLimited(w http.ResponseWriter, r *http.Request) {
	clientIP := getClientIP(r)
	if !g.rateLimiter.Allow(clientIP) {
		http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
		return
	}

	g.handleMetrics(w, r)
}

func (g *Gateway) handleShortURLRedirectRateLimited(w http.ResponseWriter, r *http.Request) {
	clientIP := getClientIP(r)
	if !g.rateLimiter.Allow(clientIP) {
		http.Error(w, "Rate limit exceeded, try again later", http.StatusTooManyRequests)
		return
	}

	g.handleShortURLRedirect(w, r)
}

func (g *Gateway) handleShortenURL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	clientIP := getClientIP(r)

	// Apply specific rate limiting for URL shortening
	if !g.shortenLimiter.Allow(clientIP) {
		http.Error(w, "Rate limit exceeded for URL shortening, try again later", http.StatusTooManyRequests)
		return
	}

	var req ShortenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.URL == "" {
		http.Error(w, "URL is required", http.StatusBadRequest)
		return
	}

	// Use provided TTL or default, limited by max allowed TTL from config
	ttlHours := g.config.Shortener.DefaultTTL
	if req.TTL > 0 && req.TTL <= g.config.Shortener.MaxTTL {
		// Respect the shorter limit between user request and server config
		ttlHours = req.TTL
	} else if req.TTL > g.config.Shortener.MaxTTL {
		ttlHours = g.config.Shortener.MaxTTL
	}

	url, err := g.shortener.Create(req.URL, time.Duration(ttlHours)*time.Hour)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	scheme := "http"
	if g.config.HTTPSPort > 0 {
		scheme = "https"
	}

	resp := ShortenResponse{
		ShortCode: url.ShortCode,
		ShortURL:  fmt.Sprintf("%s://%s%s%s", scheme, r.Host, g.config.Shortener.BasePath, url.ShortCode),
		Original:  url.Original,
		ExpiresAt: url.ExpiresAt.Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (g *Gateway) StartTCP(ctx context.Context) error {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", g.config.TCPPort))
	if err != nil {
		return fmt.Errorf("failed to start TCP listener: %w", err)
	}
	g.tcpListener = ln

	go g.acceptTCPConnections(ctx)
	log.Printf("TCP gateway started on :%d", g.config.TCPPort)
	return nil
}

func (g *Gateway) StartWebSocket(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", g.handleWebSocket)

	g.wsServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", g.config.WSPort),
		Handler: mux,
	}

	go func() {
		if err := g.wsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("WebSocket server error: %v", err)
		}
	}()

	log.Printf("WebSocket gateway started on :%d", g.config.WSPort)
	return nil
}

func (g *Gateway) handleHTTPRequest(w http.ResponseWriter, r *http.Request) {
	if g.config.HTTPSPort > 0 && g.config.TLSCert != "" {
		host := strings.Split(r.Host, ":")[0]
		httpsHost := fmt.Sprintf("https://%s:%d%s", host, g.config.HTTPSPort, r.URL.Path)
		if r.URL.RawQuery != "" {
			httpsHost += "?" + r.URL.RawQuery
		}
		http.Redirect(w, r, httpsHost, http.StatusMovedPermanently)
		return
	}

	clientIP := getClientIP(r)
	if !g.isIPAllowed(clientIP) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	host := r.Host
	subdomain := strings.Split(host, ".")[0]

	tunnel, err := g.tunnelMgr.GetByName(subdomain)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if tunnel.Status != "active" {
		http.Error(w, "Tunnel not active", http.StatusServiceUnavailable)
		return
	}

	if isLocalAddress(tunnel.LocalAddr) {
		http.Error(w, "Forbidden: local addresses not allowed", http.StatusForbidden)
		return
	}

	req, err := http.NewRequest(r.Method, "http://"+tunnel.LocalAddr+r.URL.Path, r.Body)
	if err != nil {
		http.Error(w, "Bad gateway", http.StatusBadGateway)
		return
	}

	for k, v := range r.Header {
		req.Header.Set(k, v[0])
	}

	if tunnel.Config.Headers != nil {
		for k, v := range tunnel.Config.Headers {
			req.Header.Set(k, v)
		}
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "Bad gateway", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	for k, v := range resp.Header {
		w.Header().Set(k, v[0])
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)

	g.tunnelMgr.UpdateStats(tunnel.ID, 1, 0, 0)
}

func (g *Gateway) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

type ShortenRequest struct {
	URL string `json:"url"`
	TTL int    `json:"ttl"`
}

type ShortenResponse struct {
	ShortCode string `json:"short_code"`
	ShortURL  string `json:"short_url"`
	Original  string `json:"original"`
	ExpiresAt int64  `json:"expires_at"`
}

func (g *Gateway) handleMetrics(w http.ResponseWriter, r *http.Request) {
	if !g.validateAuthToken(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	tunnels := g.tunnelMgr.List()
	var lines []string
	lines = append(lines, "# HELP atunnels_tunnels Total tunnels")
	lines = append(lines, "# TYPE atunnels_tunnels gauge")
	lines = append(lines, fmt.Sprintf("atunnels_tunnels %d", len(tunnels)))

	for _, t := range tunnels {
		stats := t.GetStats()
		lines = append(lines, fmt.Sprintf("# TYPE atunnels_tunnel_requests_total counter"))
		lines = append(lines, fmt.Sprintf("atunnels_tunnel_requests_total{tunnel=\"%s\"} %d", t.Name, stats.TotalRequests))
		lines = append(lines, fmt.Sprintf("# TYPE atunnels_tunnel_bytes_in_total counter"))
		lines = append(lines, fmt.Sprintf("atunnels_tunnel_bytes_in_total{tunnel=\"%s\"} %d", t.Name, stats.TotalBytesIn))
		lines = append(lines, fmt.Sprintf("# TYPE atunnels_tunnel_bytes_out_total counter"))
		lines = append(lines, fmt.Sprintf("atunnels_tunnel_bytes_out_total{tunnel=\"%s\"} %d", t.Name, stats.TotalBytesOut))
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(strings.Join(lines, "\n")))
}

func (g *Gateway) handleShortURLRedirect(w http.ResponseWriter, r *http.Request) {
	code := strings.TrimPrefix(r.URL.Path, "/s/")
	if code == "" {
		http.NotFound(w, r)
		return
	}

	original, err := g.shortener.Resolve(code)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	http.Redirect(w, r, original, http.StatusMovedPermanently)
}

func (g *Gateway) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	tunnelName := r.URL.Query().Get("tunnel")
	if tunnelName == "" {
		http.Error(w, "tunnel parameter required", http.StatusBadRequest)
		return
	}

	tunnel, err := g.tunnelMgr.GetByName(tunnelName)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	localConn, err := net.Dial("tcp", tunnel.LocalAddr)
	if err != nil {
		log.Printf("Failed to connect to local: %v", err)
		return
	}
	defer localConn.Close()

	go func() {
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}
			localConn.Write(msg)
		}
	}()

	buf := make([]byte, 4096)
	for {
		n, err := localConn.Read(buf)
		if err != nil {
			return
		}
		conn.WriteMessage(1, buf[:n])
	}
}

func (g *Gateway) acceptTCPConnections(ctx context.Context) {
	for {
		conn, err := g.tcpListener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return
			default:
				log.Printf("TCP accept error: %v", err)
				continue
			}
		}

		go g.handleTCPConnection(conn)
	}
}

func (g *Gateway) handleTCPConnection(conn net.Conn) {
	defer conn.Close()

	localConn, err := net.Dial("tcp", "localhost:0")
	if err != nil {
		log.Printf("Failed to connect to local: %v", err)
		return
	}
	defer localConn.Close()

	go io.Copy(localConn, conn)
	io.Copy(conn, localConn)
}

func (g *Gateway) Stop() error {
	if g.httpServer != nil {
		g.httpServer.Shutdown(context.Background())
	}
	if g.tcpListener != nil {
		g.tcpListener.Close()
	}
	if g.wsServer != nil {
		g.wsServer.Shutdown(context.Background())
	}
	if g.httpsServer != nil {
		g.httpsServer.Shutdown(context.Background())
	}
	return nil
}

func checkOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}

	parsedOrigin, err := url.Parse(origin)
	if err != nil {
		return false
	}

	originHost := parsedOrigin.Host
	allowedHost := "localhost"
	if r.Host != "" {
		if idx := strings.Index(r.Host, ":"); idx > 0 {
			allowedHost = r.Host[:idx]
		} else {
			allowedHost = r.Host
		}
	}

	if originHost == allowedHost || originHost == "localhost" || originHost == "127.0.0.1" {
		return true
	}

	return false
}

func isLocalAddress(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}

	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}

	return ip.IsLoopback() || ip.IsUnspecified() || ip.IsLinkLocalUnicast()
}

func getClientIP(r *http.Request) string {
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		parts := strings.Split(forwarded, ",")
		return strings.TrimSpace(parts[0])
	}

	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}

	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

func (g *Gateway) isIPAllowed(ipStr string) bool {
	if len(g.config.AllowedIPs) == 0 {
		return true
	}

	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}

	for _, allowed := range g.config.AllowedIPs {
		_, cidr, err := net.ParseCIDR(allowed)
		if err != nil {
			continue
		}
		if cidr.Contains(ip) {
			return true
		}
	}

	return false
}

func (g *Gateway) validateAuthToken(r *http.Request) bool {
	if g.config.AuthToken == "" {
		return true
	}

	token := r.Header.Get("Authorization")
	if token != "" && strings.HasPrefix(token, "Bearer ") {
		token = token[7:]
	}

	return subtle.ConstantTimeCompare([]byte(token), []byte(g.config.AuthToken)) == 1
}
