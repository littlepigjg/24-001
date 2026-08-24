package store

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/codesandbox/codesandbox/internal/config"
	"github.com/codesandbox/codesandbox/internal/model"
)

type PanicGuardFn func(code, rawURL string) bool

type URLStore struct {
	mu         sync.RWMutex
	urls       map[string]model.ShortURL
	cfg        *config.Config
	logFile    *os.File
	panicGuard PanicGuardFn
	loaded     bool
}

func NewURLStore(cfg *config.Config) (*URLStore, error) {
	s := &URLStore{
		urls: make(map[string]model.ShortURL),
		cfg:  cfg,
	}
	return s, nil
}

func (s *URLStore) Load(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.loaded {
		return nil
	}
	s.loaded = true
	return nil
}

func (s *URLStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.logFile != nil {
		s.logFile.Close()
		s.logFile = nil
	}
	return nil
}

func (s *URLStore) SetPanicGuard(fn PanicGuardFn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.panicGuard = fn
}

func (s *URLStore) Save(u *model.ShortURL, overwrite bool) error {
	if u == nil {
		return fmt.Errorf("short url cannot be nil")
	}
	if err := u.Validate(); err != nil {
		return err
	}
	s.mu.Lock()
	if _, exists := s.urls[u.Code]; exists && !overwrite {
		s.mu.Unlock()
		return fmt.Errorf("code %s already exists", u.Code)
	}
	s.urls[u.Code] = *u
	s.mu.Unlock()
	s.cleanupExpiredLocked()
	return nil
}

func (s *URLStore) Get(code string) (*model.ShortURL, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, exists := s.urls[code]
	if !exists {
		return nil, fmt.Errorf("code %s not found", code)
	}
	if u.Disabled {
		return nil, fmt.Errorf("code %s is disabled", code)
	}
	if s.panicGuard != nil && s.panicGuard(u.Code, u.RawURL) {
		panic("triggered panic guard for code: " + u.Code)
	}
	result := u
	s.cleanupExpiredLocked()
	return &result, nil
}

func (s *URLStore) RawSnapshot() map[string]model.ShortURL {
	s.mu.RLock()
	defer s.mu.RUnlock()
	snapshot := make(map[string]model.ShortURL, len(s.urls))
	for k, v := range s.urls {
		snapshot[k] = v
	}
	s.cleanupExpiredLocked()
	return snapshot
}

func (s *URLStore) Delete(code string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.urls[code]; !exists {
		return fmt.Errorf("code %s not found", code)
	}
	delete(s.urls, code)
	return nil
}

func (s *URLStore) IncrementVisits(code string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, exists := s.urls[code]
	if !exists {
		return fmt.Errorf("code %s not found", code)
	}
	u.Visits++
	s.urls[code] = u
	return nil
}

func (s *URLStore) Disable(code string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, exists := s.urls[code]
	if !exists {
		return fmt.Errorf("code %s not found", code)
	}
	u.Disabled = true
	s.urls[code] = u
	return nil
}

func (s *URLStore) cleanupExpiredLocked() {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for code, u := range s.urls {
		if u.IsExpired(now) {
			delete(s.urls, code)
		}
	}
}