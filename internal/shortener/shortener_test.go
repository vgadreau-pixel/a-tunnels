package shortener

import (
	"testing"
	"time"
)

func TestShortenerBasicFunctionality(t *testing.T) {
	s := New()

	originalURL := "https://example.com/long/path/here"
	ttl := 2 * time.Hour

	url, err := s.Create(originalURL, ttl)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if url.Original != originalURL {
		t.Errorf("Expected original URL %s, got %s", originalURL, url.Original)
	}

	if len(url.ShortCode) == 0 {
		t.Error("Expected non-empty short code")
	}

	if url.Clicks != 0 {
		t.Errorf("Expected initial clicks 0, got %d", url.Clicks)
	}
}

func TestShortenerResolve(t *testing.T) {
	s := New()

	originalURL := "https://example.com/test"
	ttl := 1 * time.Hour

	url, err := s.Create(originalURL, ttl)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	original, err := s.Resolve(url.ShortCode)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if original != originalURL {
		t.Errorf("Expected %s, got %s", originalURL, original)
	}

	// Check that clicking increases clicks count
	_, err = s.Resolve(url.ShortCode)
	if err != nil {
		t.Fatalf("Second resolve failed: %v", err)
	}

	retrieved, err := s.Get(url.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if retrieved.Clicks != 2 { // Initial creation counts as 1, first resolve +1, second resolve +1
		t.Errorf("Expected clicks 2 after 2 resolves, got %d", retrieved.Clicks)
	}
}

func TestShortenerExpiry(t *testing.T) {
	s := New()

	originalURL := "https://example.com/expiring"
	ttl := 1 * time.Millisecond // Very short TTL for test

	url, err := s.Create(originalURL, ttl)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	time.Sleep(2 * time.Millisecond) // Wait for expiration

	_, err = s.Resolve(url.ShortCode)
	if err == nil {
		t.Error("Expected error after expiry, got nil")
	}

	_, err = s.Get(url.ID)
	if err == nil {
		t.Error("Expected error getting expired URL, got nil")
	}
}

func TestShortenerNotfound(t *testing.T) {
	s := New()

	_, err := s.Resolve("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent code, got nil")
	}
}

func TestShortenerCleanup(t *testing.T) {
	s := New()

	originalURL := "https://example.com/cleanup"
	ttl := 1 * time.Millisecond // Very short TTL for test

	_, err := s.Create(originalURL, ttl)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	time.Sleep(2 * time.Millisecond) // Wait for expiration

	s.Cleanup() // Should remove expired URLs

	// Try to list, should be empty
	list := s.List()
	if len(list) > 0 {
		t.Errorf("Expected empty list after cleanup, got %d items", len(list))
	}
}
