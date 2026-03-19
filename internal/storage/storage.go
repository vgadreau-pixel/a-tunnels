package storage

import (
	"fmt"
	"time"

	"github.com/a-tunnels/a-tunnels/internal/config"
	"github.com/a-tunnels/a-tunnels/internal/tunnel"
)

type Storage interface {
	SaveTunnel(t *tunnel.Tunnel) error
	GetTunnel(id string) (*tunnel.Tunnel, error)
	DeleteTunnel(id string) error
	ListTunnels() ([]*tunnel.Tunnel, error)
	Close() error
}

type MemoryStorage struct {
	tunnels map[string]*tunnel.Tunnel
}

func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		tunnels: make(map[string]*tunnel.Tunnel),
	}
}

func (s *MemoryStorage) SaveTunnel(t *tunnel.Tunnel) error {
	s.tunnels[t.ID] = t
	return nil
}

func (s *MemoryStorage) GetTunnel(id string) (*tunnel.Tunnel, error) {
	t, ok := s.tunnels[id]
	if !ok {
		return nil, fmt.Errorf("tunnel not found: %s", id)
	}
	return t, nil
}

func (s *MemoryStorage) DeleteTunnel(id string) error {
	delete(s.tunnels, id)
	return nil
}

func (s *MemoryStorage) ListTunnels() ([]*tunnel.Tunnel, error) {
	result := make([]*tunnel.Tunnel, 0, len(s.tunnels))
	for _, t := range s.tunnels {
		result = append(result, t)
	}
	return result, nil
}

func (s *MemoryStorage) Close() error {
	return nil
}

type FileStorage struct {
	MemoryStorage
	path string
}

func NewFileStorage(cfg config.StorageConfig) (*FileStorage, error) {
	return &FileStorage{
		MemoryStorage: *NewMemoryStorage(),
		path:          cfg.Path,
	}, nil
}

type LogEntry struct {
	TunnelID  string    `json:"tunnel_id"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

func init() {
	_ = LogEntry{}
}
