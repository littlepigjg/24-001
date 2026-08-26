package store

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/codesandbox/codesandbox/internal/config"
	"github.com/codesandbox/codesandbox/internal/model"
)

type PanicGuardFn func(code, rawURL string) bool

type URLStore struct {
	mu         sync.RWMutex
	urls       map[string]model.ShortURL
	cfg        *config.Config
	panicGuard PanicGuardFn
	loaded     bool
}

func NewURLStore(cfg *config.Config) (*URLStore, error) {
	if cfg == nil {
		return nil, errors.New("config cannot be nil")
	}
	return &URLStore{
		urls: make(map[string]model.ShortURL),
		cfg:  cfg,
	}, nil
}

func (s *URLStore) Load(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loaded = true
	return nil
}

func (s *URLStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.urls = make(map[string]model.ShortURL)
	s.loaded = false
	return nil
}

func (s *URLStore) SetPanicGuard(fn PanicGuardFn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.panicGuard = fn
}

func (s *URLStore) Save(u *model.ShortURL, overwrite bool) error {
	if u == nil {
		return errors.New("short URL cannot be nil")
	}
	if err := u.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	defer func() {
		if r := recover(); r != nil {
			if s.panicGuard != nil && s.panicGuard(u.Code, u.RawURL) {
				return
			}
			panic(fmt.Sprintf("storage panic: %v", r))
		}
	}()

	if !overwrite {
		if _, exists := s.urls[u.Code]; exists {
			return errors.New("code already exists")
		}
	}

	if u.Code == "" {
		return errors.New("code is required")
	}

	s.urls[u.Code] = *u
	return nil
}

func (s *URLStore) Get(code string) (*model.ShortURL, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	u, exists := s.urls[code]
	if !exists {
		return nil, errors.New("short URL not found")
	}
	result := u
	return &result, nil
}

func (s *URLStore) RawSnapshot() map[string]model.ShortURL {
	s.mu.RLock()
	defer s.mu.RUnlock()

	snapshot := make(map[string]model.ShortURL, len(s.urls))
	for k, v := range s.urls {
		snapshot[k] = v
	}
	return snapshot
}
