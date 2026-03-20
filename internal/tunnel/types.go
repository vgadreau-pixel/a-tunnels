package tunnel

import (
	"context"
	crand "crypto/rand"
	"encoding/hex"
	"fmt"
	mrand "math/rand"
	"sync"
	"time"

	"github.com/a-tunnels/a-tunnels/internal/config"
)

var rng = mrand.New(mrand.NewSource(time.Now().UnixNano()))

type Tunnel struct {
	ID         string
	Name       string
	Protocol   string
	LocalAddr  string
	Subdomain  string
	RemotePort int
	Status     TunnelStatus
	Config     *config.TunnelConfig
	Stats      *TunnelStats
	CreatedAt  time.Time
	UpdatedAt  time.Time
	mu         sync.RWMutex
}

type TunnelStatus string

const (
	TunnelStatusPending TunnelStatus = "pending"
	TunnelStatusActive  TunnelStatus = "active"
	TunnelStatusPaused  TunnelStatus = "paused"
	TunnelStatusError   TunnelStatus = "error"
	TunnelStatusStopped TunnelStatus = "stopped"
)

type TunnelStats struct {
	ActiveConnections int64
	TotalRequests     int64
	TotalBytesIn      int64
	TotalBytesOut     int64
	LastRequestAt     time.Time
	ErrorCount        int64
	mu                sync.RWMutex
}

type TunnelEvent struct {
	Type    string
	Tunnel  *Tunnel
	Message string
	Time    time.Time
}

type Manager interface {
	Create(ctx context.Context, cfg *config.TunnelConfig) (*Tunnel, error)
	Get(id string) (*Tunnel, error)
	GetByName(name string) (*Tunnel, error)
	List() []*Tunnel
	Delete(id string) error
	Start(id string) error
	Stop(id string) error
	Restart(id string) error
	UpdateStats(id string, conns int64, bytesIn, bytesOut int64)
	GetStats(id string) (*TunnelStats, error)
	Subscribe() chan *TunnelEvent
}

func generateID() string {
	bytes := make([]byte, 6)
	crand.Read(bytes)
	return hex.EncodeToString(bytes)
}

var adjectives = []string{
	"brave", "calm", "eager", "gentle", "happy", "jolly", "kind", "lively",
	"merry", "nice", "proud", "silly", "swift", "wise", "young", "bold",
	"cool", "dawn", "eager", "fair", "gold", "hero", "iron", "jade",
	"keen", "light", "mist", "neon", "opal", "pearl", "quiet", "rose",
	"silver", "true", "urban", "vivid", "wild", "zen", "amber", "bright",
	"coral", "dream", "echo", "flame", "grace", "haze", "ivory", "joy",
	"kindle", "lunar", "magic", "nova", "ocean", "peace", "quest", "river",
	"stone", "turbo", "ultra", "violet", "wave", "xenon", "yonder", "zest",
}

var nouns = []string{
	"river", "forest", "mountain", "ocean", "valley", "prairie", "meadow",
	"canyon", "glacier", "island", "desert", "jungle", "rainforest", "tundra",
	"waterfall", "sunset", "sunrise", "horizon", "galaxy", "nebula", "comet",
	"planet", "star", "moon", "sun", "sky", "cloud", "storm", "rain", "snow",
	"wind", "fire", "earth", "stone", "rock", "tree", "flower", "garden",
	"lake", "pond", "stream", "beach", "shore", "reef", "cave", "cliff",
	"field", "hill", "peak", "ridge", "valley", "grove", "wood", "marsh",
	"swamp", "delta", "ford", "bridge", "tower", "castle", "palace", "temple",
}

func GenerateRandomName() string {
	adjIdx := rng.Intn(len(adjectives))
	nounIdx := rng.Intn(len(nouns))
	num := rng.Intn(10000)

	return fmt.Sprintf("%s-%s-%04d", adjectives[adjIdx], nouns[nounIdx], num)
}

func NewTunnel(cfg *config.TunnelConfig) *Tunnel {
	return &Tunnel{
		ID:         generateID(),
		Name:       cfg.Name,
		Protocol:   cfg.Protocol,
		LocalAddr:  cfg.LocalAddr,
		Subdomain:  cfg.Subdomain,
		RemotePort: cfg.RemotePort,
		Status:     TunnelStatusPending,
		Config:     cfg,
		Stats:      &TunnelStats{},
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
}

func (t *Tunnel) SetStatus(status TunnelStatus) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Status = status
	t.UpdatedAt = time.Now()
}

func (t *Tunnel) GetStatus() TunnelStatus {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.Status
}

func (t *Tunnel) UpdateStats(conns int64, bytesIn, bytesOut int64) {
	t.Stats.mu.Lock()
	defer t.Stats.mu.Unlock()
	t.Stats.ActiveConnections = conns
	t.Stats.TotalBytesIn += bytesIn
	t.Stats.TotalBytesOut += bytesOut
	t.Stats.TotalRequests++
	t.Stats.LastRequestAt = time.Now()
}

func (t *Tunnel) GetStats() TunnelStats {
	t.Stats.mu.RLock()
	defer t.Stats.mu.RUnlock()
	return *t.Stats
}

type manager struct {
	tunnels map[string]*Tunnel
	byName  map[string]*Tunnel
	events  chan *TunnelEvent
	mu      sync.RWMutex
}

func NewManager() Manager {
	return &manager{
		tunnels: make(map[string]*Tunnel),
		byName:  make(map[string]*Tunnel),
		events:  make(chan *TunnelEvent, 100),
	}
}

func (m *manager) Create(ctx context.Context, cfg *config.TunnelConfig) (*Tunnel, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.byName[cfg.Name]; exists {
		return nil, fmt.Errorf("tunnel with name %s already exists", cfg.Name)
	}

	tunnel := NewTunnel(cfg)
	m.tunnels[tunnel.ID] = tunnel
	m.byName[tunnel.Name] = tunnel

	m.emit(&TunnelEvent{
		Type:    "created",
		Tunnel:  tunnel,
		Message: fmt.Sprintf("Tunnel %s created", tunnel.Name),
	})

	return tunnel, nil
}

func (m *manager) Get(id string) (*Tunnel, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tunnel, ok := m.tunnels[id]
	if !ok {
		return nil, fmt.Errorf("tunnel not found: %s", id)
	}
	return tunnel, nil
}

func (m *manager) GetByName(name string) (*Tunnel, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tunnel, ok := m.byName[name]
	if !ok {
		return nil, fmt.Errorf("tunnel not found: %s", name)
	}
	return tunnel, nil
}

func (m *manager) List() []*Tunnel {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*Tunnel, 0, len(m.tunnels))
	for _, t := range m.tunnels {
		result = append(result, t)
	}
	return result
}

func (m *manager) Delete(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	tunnel, ok := m.tunnels[id]
	if !ok {
		return fmt.Errorf("tunnel not found: %s", id)
	}

	delete(m.tunnels, id)
	delete(m.byName, tunnel.Name)

	m.emit(&TunnelEvent{
		Type:    "deleted",
		Tunnel:  tunnel,
		Message: fmt.Sprintf("Tunnel %s deleted", tunnel.Name),
	})

	return nil
}

func (m *manager) Start(id string) error {
	tunnel, err := m.Get(id)
	if err != nil {
		return err
	}

	tunnel.SetStatus(TunnelStatusActive)

	m.emit(&TunnelEvent{
		Type:    "started",
		Tunnel:  tunnel,
		Message: fmt.Sprintf("Tunnel %s started", tunnel.Name),
	})

	return nil
}

func (m *manager) Stop(id string) error {
	tunnel, err := m.Get(id)
	if err != nil {
		return err
	}

	tunnel.SetStatus(TunnelStatusStopped)

	m.emit(&TunnelEvent{
		Type:    "stopped",
		Tunnel:  tunnel,
		Message: fmt.Sprintf("Tunnel %s stopped", tunnel.Name),
	})

	return nil
}

func (m *manager) Restart(id string) error {
	if err := m.Stop(id); err != nil {
		return err
	}
	return m.Start(id)
}

func (m *manager) UpdateStats(id string, conns int64, bytesIn, bytesOut int64) {
	tunnel, err := m.Get(id)
	if err != nil {
		return
	}
	tunnel.UpdateStats(conns, bytesIn, bytesOut)
}

func (m *manager) GetStats(id string) (*TunnelStats, error) {
	tunnel, err := m.Get(id)
	if err != nil {
		return nil, err
	}

	stats := tunnel.GetStats()
	return &stats, nil
}

func (m *manager) Subscribe() chan *TunnelEvent {
	return m.events
}

func (m *manager) emit(event *TunnelEvent) {
	select {
	case m.events <- event:
	default:
	}
}
