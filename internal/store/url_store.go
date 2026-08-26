package store

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/codesandbox/codesandbox/config"
	"github.com/codesandbox/codesandbox/internal/model"
	"github.com/codesandbox/codesandbox/pkg/osutil"
)

type PanicGuardFn func(code, rawURL string) bool

type URLStore struct {
	mu           sync.RWMutex
	cfg          *config.Config
	urls         map[string]model.ShortURL
	panicGuard   PanicGuardFn
	dirty        bool
	lastSyncTime time.Time
}

func NewURLStore(cfg *config.Config) (*URLStore, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	s := &URLStore{
		cfg:          cfg,
		urls:         make(map[string]model.ShortURL),
		lastSyncTime: time.Now(),
	}

	return s, nil
}

func (s *URLStore) SetPanicGuard(fn PanicGuardFn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.panicGuard = fn
}

func (s *URLStore) validateHTTPS() error {
	if s.cfg.Storage.HTTPSEnabled() {
		certFile := s.cfg.Storage.HTTPSCertFile()
		keyFile := s.cfg.Storage.HTTPSKeyFile()
		if certFile == "" || keyFile == "" {
			return fmt.Errorf("HTTPS enabled but certificate and key files not configured")
		}
		if !osutil.FileExists(certFile) {
			return fmt.Errorf("HTTPS certificate file not found: %s", certFile)
		}
		if !osutil.FileExists(keyFile) {
			return fmt.Errorf("HTTPS key file not found: %s", keyFile)
		}
	}
	return nil
}

func (s *URLStore) Load(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := ctx.Err(); err != nil {
		return err
	}

	if err := s.validateHTTPS(); err != nil {
		if s.panicGuard != nil {
			if s.panicGuard("", "") {
				panic(fmt.Sprintf("HTTPS validation failed: %v", err))
			}
		}
		return err
	}

	filePath := s.cfg.Storage.GetURLFilePath()
	if filePath == "" {
		return fmt.Errorf("URL file path not configured")
	}

	dir := filepath.Dir(filePath)
	if dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read URL file: %w", err)
	}

	if len(data) > 0 {
		var urls map[string]model.ShortURL
		if err := json.Unmarshal(data, &urls); err != nil {
			return fmt.Errorf("failed to parse URL file: %w", err)
		}
		s.urls = urls
	}

	s.dirty = false
	s.lastSyncTime = time.Now()
	return nil
}

func (s *URLStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cfg.Storage.GetFlushOnWrite() && s.dirty {
		if err := s.flush(); err != nil {
			return err
		}
	}
	return nil
}

func (s *URLStore) flush() error {
	filePath := s.cfg.Storage.GetURLFilePath()
	if filePath == "" {
		return fmt.Errorf("URL file path not configured")
	}

	dir := filepath.Dir(filePath)
	if dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}

	data, err := json.MarshalIndent(s.urls, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal URLs: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write URL file: %w", err)
	}

	s.dirty = false
	s.lastSyncTime = time.Now()
	return nil
}

func (s *URLStore) Save(u *model.ShortURL, overwrite bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.validateHTTPS(); err != nil {
		if s.panicGuard != nil {
			if s.panicGuard(u.Code, u.RawURL) {
				panic(fmt.Sprintf("HTTPS validation failed for code %s: %v", u.Code, err))
			}
		}
		return err
	}

	if _, exists := s.urls[u.Code]; exists && !overwrite {
		return fmt.Errorf("code already exists: %s", u.Code)
	}

	s.urls[u.Code] = *u
	s.dirty = true

	if s.cfg.Storage.GetFlushOnWrite() {
		return s.flush()
	}
	return nil
}

func (s *URLStore) Get(code string) (*model.ShortURL, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	url, exists := s.urls[code]
	if !exists {
		return nil, fmt.Errorf("short URL not found: %s", code)
	}

	if url.IsExpired(time.Now()) {
		return nil, fmt.Errorf("short URL has expired: %s", code)
	}

	return &url, nil
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
