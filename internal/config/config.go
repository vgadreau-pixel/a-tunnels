package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server  ServerConfig   `yaml:"server"`
	Client  ClientConfig   `yaml:"client,omitempty"`
	Tunnels []TunnelConfig `yaml:"tunnels,omitempty"`
}

type ServerConfig struct {
	Host           string          `yaml:"host"`
	HTTPPort       int             `yaml:"http_port"`
	HTTPSPort      int             `yaml:"https_port"`
	TCPPortStart   int             `yaml:"tcp_port_start"`
	WSPortStart    int             `yaml:"ws_port_start"`
	APIPort        int             `yaml:"api_port"`
	MCPPort        int             `yaml:"mcp_port"`
	MCPToken       string          `yaml:"mcp_token"`
	SSHPort        int             `yaml:"ssh_port"`
	TLS            TLSConfig       `yaml:"tls"`
	Auth           AuthConfig      `yaml:"auth"`
	Storage        StorageConfig   `yaml:"storage"`
	Shortener      ShortenerConfig `yaml:"shortener"`
	Limits         LimitsConfig    `yaml:"limits"`
	Domain         string          `yaml:"domain"`
	MetricsEnabled bool            `yaml:"metrics_enabled"`

	// Server modes - set to false to disable
	HTTPEnabled  bool `yaml:"http_enabled"`
	HTTPSEnabled bool `yaml:"https_enabled"`
	TCPEnabled   bool `yaml:"tcp_enabled"`
	WSEnabled    bool `yaml:"ws_enabled"`
	APIEnabled   bool `yaml:"api_enabled"`
	SSHEnabled   bool `yaml:"ssh_enabled"`
	MCPEnabled   bool `yaml:"mcp_enabled"`
}

type TLSConfig struct {
	Enabled   bool   `yaml:"enabled"`
	Email     string `yaml:"email"`
	CertCache string `yaml:"cert_cache"`
	AutoTLS   bool   `yaml:"auto_tls"`
}

type AuthConfig struct {
	APIKeys  []string `yaml:"api_keys"`
	SSHKeys  []string `yaml:"ssh_keys"`
	MCPToken string   `yaml:"mcp_token"`
	Admins   []string `yaml:"admins"`
}

type StorageConfig struct {
	Type     string `yaml:"type"`
	Path     string `yaml:"path"`
	RedisURL string `yaml:"redis_url,omitempty"`
}

type ShortenerConfig struct {
	Enabled     bool   `yaml:"enabled"`
	DefaultTTL  int    `yaml:"default_ttl"`  // TTL par défaut (en heures)
	MaxTTL      int    `yaml:"max_ttl"`      // TTL maximum autorisé (en heures)
	MaxLength   int    `yaml:"max_length"`   // Longueur maximale du code
	BasePath    string `yaml:"base_path"`    // Chemin de base pour les URLs courtes
	CleanupFreq int    `yaml:"cleanup_freq"` // Fréquence de nettoyage (en minutes)
}

type LimitsConfig struct {
	MaxTunnels        int   `yaml:"max_tunnels"`
	MaxConnsPerTunnel int   `yaml:"max_conns_per_tunnel"`
	RateLimit         int   `yaml:"rate_limit"`
	RateLimitPeriod   int64 `yaml:"rate_limit_period"`
	ShortenerLimit    int   `yaml:"shortener_limit"`  // Limit for shortener creation
	ShortenerPeriod   int64 `yaml:"shortener_period"` // Period for shortener (in minutes)
}

type ClientConfig struct {
	ServerAddr        string         `yaml:"server_addr"`
	Token             string         `yaml:"token"`
	ReconnectInterval time.Duration  `yaml:"reconnect_interval"`
	Tunnels           []TunnelConfig `yaml:"tunnels,omitempty"`
}

type TunnelConfig struct {
	Name          string            `yaml:"name"`
	Protocol      string            `yaml:"protocol"`
	LocalAddr     string            `yaml:"local_addr"`
	Subdomain     string            `yaml:"subdomain,omitempty"`
	RemotePort    int               `yaml:"remote_port,omitempty"`
	Auth          *TunnelAuth       `yaml:"auth,omitempty"`
	Headers       map[string]string `yaml:"headers,omitempty"`
	Timeout       time.Duration     `yaml:"timeout"`
	MaxConns      int               `yaml:"max_conns"`
	IPWhitelist   []string          `yaml:"ip_whitelist,omitempty"`
	WebhookURL    string            `yaml:"webhook_url,omitempty"`
	WebhookEvents []string          `yaml:"webhook_events,omitempty"`
}

type TunnelAuth struct {
	Type  string `yaml:"type"`
	Token string `yaml:"token"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	cfg.setDefaults()
	return &cfg, nil
}

func LoadDefault() *Config {
	cfg := &Config{}
	cfg.setDefaults()
	return cfg
}

func (c *Config) setDefaults() {
	if c.Server.Host == "" {
		c.Server.Host = "0.0.0.0"
	}
	if c.Server.HTTPPort == 0 {
		c.Server.HTTPPort = 80
	}
	if c.Server.HTTPSPort == 0 {
		c.Server.HTTPSPort = 443
	}
	if c.Server.TCPPortStart == 0 {
		c.Server.TCPPortStart = 10000
	}
	if c.Server.WSPortStart == 0 {
		c.Server.WSPortStart = 11000
	}
	if c.Server.APIPort == 0 {
		c.Server.APIPort = 8080
	}
	if c.Server.MCPPort == 0 {
		c.Server.MCPPort = 27200
	}
	if c.Server.SSHPort == 0 {
		c.Server.SSHPort = 2222
	}
	if c.Server.Shortener.DefaultTTL == 0 {
		c.Server.Shortener.DefaultTTL = 24
	}
	if c.Server.Shortener.MaxTTL == 0 {
		c.Server.Shortener.MaxTTL = 720
	}
	if c.Server.Shortener.MaxLength == 0 {
		c.Server.Shortener.MaxLength = 8
	}
	if c.Server.Shortener.BasePath == "" {
		c.Server.Shortener.BasePath = "/s/"
	}
	if c.Server.Shortener.CleanupFreq == 0 {
		c.Server.Shortener.CleanupFreq = 10
	}
	if c.Server.Limits.ShortenerLimit == 0 {
		c.Server.Limits.ShortenerLimit = 10
	}
	if c.Server.Limits.ShortenerPeriod == 0 {
		c.Server.Limits.ShortenerPeriod = 60 // 60 minutes
	}
	if c.Server.Limits.MaxTunnels == 0 {
		c.Server.Limits.MaxTunnels = 100
	}
	if c.Server.Limits.MaxConnsPerTunnel == 0 {
		c.Server.Limits.MaxConnsPerTunnel = 1000
	}
	if c.Server.Limits.RateLimit == 0 {
		c.Server.Limits.RateLimit = 1000
	}
	if c.Server.Limits.RateLimitPeriod == 0 {
		c.Server.Limits.RateLimitPeriod = 60
	}
	// Server modes - all enabled by default
	if !c.Server.HTTPEnabled {
		c.Server.HTTPEnabled = true
	}
	if !c.Server.HTTPSEnabled {
		c.Server.HTTPSEnabled = true
	}
	if !c.Server.TCPEnabled {
		c.Server.TCPEnabled = true
	}
	if !c.Server.WSEnabled {
		c.Server.WSEnabled = true
	}
	if !c.Server.APIEnabled {
		c.Server.APIEnabled = true
	}
	if !c.Server.SSHEnabled {
		c.Server.SSHEnabled = true
	}
	if !c.Server.MCPEnabled {
		c.Server.MCPEnabled = true
	}
	if c.Client.ReconnectInterval == 0 {
		c.Client.ReconnectInterval = 5 * time.Second
	}
}
