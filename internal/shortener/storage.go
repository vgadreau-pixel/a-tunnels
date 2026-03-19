package shortener

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

type MemoryStorage struct {
	urls      map[string]*URL
	codeIndex map[string]string
	mu        sync.RWMutex
}

func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		urls:      make(map[string]*URL),
		codeIndex: make(map[string]string),
	}
}

func (s *MemoryStorage) SaveURL(url *URL) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.urls[url.ID] = url
	s.codeIndex[url.ShortCode] = url.ID
	return nil
}

func (s *MemoryStorage) GetURL(id string) (*URL, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	url, ok := s.urls[id]
	if !ok {
		return nil, fmt.Errorf("URL not found: %s", id)
	}
	return url, nil
}

func (s *MemoryStorage) DeleteURL(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	url, ok := s.urls[id]
	if !ok {
		return fmt.Errorf("URL not found: %s", id)
	}

	delete(s.urls, id)
	delete(s.codeIndex, url.ShortCode)
	return nil
}

func (s *MemoryStorage) ListURLs() ([]*URL, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*URL, 0, len(s.urls))
	for _, url := range s.urls {
		result = append(result, url)
	}
	return result, nil
}

func (s *MemoryStorage) Close() error {
	return nil
}

type FileStorage struct {
	MemoryStorage
	path string
	mu   sync.Mutex // Separate mutex for file operations
}

func NewFileStorage(path string) (*FileStorage, error) {
	storage := &FileStorage{
		MemoryStorage: *NewMemoryStorage(),
		path:          path,
	}

	// Load existing data on startup
	if err := storage.loadFromFile(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to load from file: %w", err)
	}

	return storage, nil
}

func (s *FileStorage) loadFromFile() error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return err
	}

	var urls []*URL
	if err := json.Unmarshal(data, &urls); err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	for _, url := range urls {
		s.MemoryStorage.urls[url.ID] = url
		s.MemoryStorage.codeIndex[url.ShortCode] = url.ID
	}

	return nil
}

func (s *FileStorage) SaveURL(url *URL) error {
	if err := s.MemoryStorage.SaveURL(url); err != nil {
		return err
	}

	// Persist to file asynchronously
	go func() {
		_ = s.saveToFile() // Just ignore error here - would log in production
	}()

	return nil
}

func (s *FileStorage) saveToFile() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	urls, err := s.MemoryStorage.ListURLs()
	if err != nil {
		return fmt.Errorf("failed to list URLs: %w", err)
	}

	data, err := json.MarshalIndent(urls, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	tempPath := s.path + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temporary file: %w", err)
	}

	if err := os.Rename(tempPath, s.path); err != nil {
		return fmt.Errorf("failed to rename temporary file: %w", err)
	}

	return nil
}

func (s *FileStorage) Close() error {
	return s.saveToFile()
}
