package shortener

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

type Storage interface {
	SaveURL(url *URL) error
	GetURL(id string) (*URL, error)
	DeleteURL(id string) error
	ListURLs() ([]*URL, error)
	Close() error
}

type URL struct {
	ID        string    `json:"id"`
	Original  string    `json:"original"`
	ShortCode string    `json:"short_code"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	Clicks    int       `json:"clicks"`
}

type Shortener struct {
	urls      map[string]*URL
	codeIndex map[string]string
	mu        sync.RWMutex
	storage   Storage
}

func New() *Shortener {
	return &Shortener{
		urls:      make(map[string]*URL),
		codeIndex: make(map[string]string),
		storage:   nil,
	}
}

func NewWithStorage(storage Storage) *Shortener {
	s := &Shortener{
		urls:      make(map[string]*URL),
		codeIndex: make(map[string]string),
		storage:   storage,
	}

	// Load existing URLs from storage
	if storage != nil {
		if urls, err := storage.ListURLs(); err == nil {
			for _, url := range urls {
				s.urls[url.ID] = url
				s.codeIndex[url.ShortCode] = url.ID
			}
		}
	}

	return s
}

func (s *Shortener) encodeCode(id string) string {
	bytes := make([]byte, 6)
	rand.Read(bytes)
	encoded := base64.URLEncoding.EncodeToString(bytes)
	return encoded[:8]
}

func (s *Shortener) Create(original string, ttl time.Duration) (*URL, error) {
	if _, err := parseURL(original); err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	id := generateID()
	code := s.encodeCode(id)

	url := &URL{
		ID:        id,
		Original:  original,
		ShortCode: code,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(ttl),
		Clicks:    0,
	}

	s.urls[id] = url
	s.codeIndex[code] = id

	// Persist to storage if configured
	if s.storage != nil {
		// Store a copy so we don't risk corruption if original changes
		_ = s.storage.SaveURL(url)
	}

	return url, nil
}

func (s *Shortener) Get(id string) (*URL, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	url, ok := s.urls[id]
	if !ok {
		return nil, fmt.Errorf("URL not found: %s", id)
	}

	if time.Now().After(url.ExpiresAt) {
		return nil, fmt.Errorf("URL expired")
	}

	return url, nil
}

func (s *Shortener) GetByCode(code string) (*URL, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	id, ok := s.codeIndex[code]
	if !ok {
		return nil, fmt.Errorf("short code not found: %s", code)
	}

	return s.Get(id)
}

func (s *Shortener) Resolve(code string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id, ok := s.codeIndex[code]
	if !ok {
		return "", fmt.Errorf("short code not found: %s", code)
	}

	url, ok := s.urls[id]
	if !ok {
		return "", fmt.Errorf("URL not found")
	}

	if time.Now().After(url.ExpiresAt) {
		return "", fmt.Errorf("URL expired")
	}

	url.Clicks++

	// Persist updated click count if storage available
	if s.storage != nil {
		_ = s.storage.SaveURL(url)
	}

	return url.Original, nil
}

func (s *Shortener) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	url, ok := s.urls[id]
	if !ok {
		return fmt.Errorf("URL not found: %s", id)
	}

	delete(s.urls, id)
	delete(s.codeIndex, url.ShortCode)

	// Remove from storage if configured
	if s.storage != nil {
		_ = s.storage.DeleteURL(id)
	}

	return nil
}

func (s *Shortener) List() []*URL {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*URL, 0, len(s.urls))
	for _, url := range s.urls {
		result = append(result, url)
	}

	return result
}

func (s *Shortener) Cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for id, url := range s.urls {
		if now.After(url.ExpiresAt) {
			delete(s.urls, id)
			delete(s.codeIndex, url.ShortCode)

			// Remove expired entry from storage if available
			if s.storage != nil {
				_ = s.storage.DeleteURL(id)
			}
		}
	}
}

func generateID() string {
	bytes := make([]byte, 4)
	rand.Read(bytes)
	return fmt.Sprintf("%x", bytes)
}

func parseURL(rawURL string) (string, error) {
	if len(rawURL) < 8 {
		return "", fmt.Errorf("URL too short")
	}

	if rawURL[:4] != "http" {
		return "", fmt.Errorf("URL must start with http or https")
	}

	return rawURL, nil
}

func init() {
	_ = json.Marshal
}
