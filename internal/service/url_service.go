package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/codesandbox/codesandbox/internal/config"
	"github.com/codesandbox/codesandbox/internal/model"
	"github.com/codesandbox/codesandbox/internal/store"
	"github.com/codesandbox/codesandbox/pkg/logger"
)

// URLService handles URL shortener operations.
type URLService struct {
	cfg   *config.Config
	store *store.URLStore
	log   *logger.Logger
}

// NewURLService creates a new URLService.
func NewURLService(cfg *config.Config, s *store.URLStore) (*URLService, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}
	if s == nil {
		return nil, fmt.Errorf("URL store is required")
	}
	return &URLService{
		cfg:   cfg,
		store: s,
		log:   logger.GetGlobal(),
	}, nil
}

// Create creates a new short URL.
func (s *URLService) Create(ctx context.Context, req *model.CreateReq) (*model.ShortURL, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	code := req.CustomCode
	if code == "" {
		var err error
		code, err = generateCode(6)
		if err != nil {
			return nil, fmt.Errorf("failed to generate code: %w", err)
		}
	}

	shortURL := &model.ShortURL{
		Code:      code,
		RawURL:    req.RawURL,
		CreatedAt: time.Now().UTC(),
		Visits:    0,
		Custom:    req.CustomCode != "",
		Disabled:  false,
	}

	if req.MaxVisits > 0 {
		shortURL.ExpiresAt = time.Now().UTC().Add(time.Duration(req.MaxVisits) * time.Hour)
	}

	if err := shortURL.Validate(); err != nil {
		return nil, fmt.Errorf("invalid short URL: %w", err)
	}

	if err := s.store.Save(shortURL, false); err != nil {
		return nil, fmt.Errorf("failed to save: %w", err)
	}

	s.log.Infof("Created short URL: %s -> %s", code, req.RawURL)
	return shortURL, nil
}

// Get retrieves a short URL by code.
func (s *URLService) Get(code string) (*model.ShortURL, error) {
	return s.store.Get(code)
}

// Delete deletes a short URL.
func (s *URLService) Delete(code string) error {
	url, err := s.store.Get(code)
	if err != nil {
		return err
	}
	url.Disabled = true
	return s.store.Save(url, true)
}

// Update updates a short URL.
func (s *URLService) Update(code string, rawURL string) (*model.ShortURL, error) {
	url, err := s.store.Get(code)
	if err != nil {
		return nil, err
	}
	url.RawURL = rawURL
	if err := s.store.Save(url, true); err != nil {
		return nil, err
	}
	return url, nil
}

// List returns all short URLs.
func (s *URLService) List() ([]model.ShortURL, error) {
	snapshot := s.store.RawSnapshot()
	urls := make([]model.ShortURL, 0, len(snapshot))
	for _, url := range snapshot {
		urls = append(urls, url)
	}
	return urls, nil
}

// Cleanup removes expired entries.
func (s *URLService) Cleanup(maxAge time.Duration) (int, error) {
	if maxAge <= 0 {
		return 0, fmt.Errorf("max age must be positive")
	}

	cutoff := time.Now().UTC().Add(-maxAge)
	removed := 0

	snapshot := s.store.RawSnapshot()
	for id, url := range snapshot {
		if url.CreatedAt.Before(cutoff) {
			url.Disabled = true
			if err := s.store.Save(&url, true); err != nil {
				s.log.Errorf("Failed to disable expired URL %s: %v", id, err)
				continue
			}
			removed++
		}
	}

	s.log.Infof("Cleaned up %d expired URLs", removed)
	return removed, nil
}

// GetExpiringSoon returns URLs that will expire within the given duration.
func (s *URLService) GetExpiringSoon(within time.Duration) ([]model.ShortURL, error) {
	snapshot := s.store.RawSnapshot()
	var results []model.ShortURL

	now := time.Now().UTC()
	threshold := now.Add(within)
	for _, url := range snapshot {
		if url.ExpiresAt.IsZero() {
			continue
		}
		if threshold.After(url.ExpiresAt) {
			results = append(results, url)
		}
	}

	return results, nil
}

// ValidateExpiry checks if a URL has expired based on its ExpiresAt.
func (s *URLService) ValidateExpiry(url *model.ShortURL) bool {
	if url.ExpiresAt.IsZero() {
		return false
	}
	return url.IsExpired(time.Now().UTC())
}

func generateCode(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b)[:n], nil
}
