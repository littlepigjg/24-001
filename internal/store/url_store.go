package store

import (
	"encoding/json"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
	"sync"

	"github.com/codesandbox/codesandbox/internal/config"
	"github.com/codesandbox/codesandbox/internal/model"
	"github.com/codesandbox/codesandbox/pkg/logger"
)

// PanicGuardFn is a function type for guarding against panic during Save operations.
type PanicGuardFn func(code, rawURL string) bool

// URLStore handles storage of shortened URLs.
type URLStore struct {
	mu          sync.RWMutex
	cfg         *config.Config
	data        map[string]model.ShortURL
	panicGuard  PanicGuardFn
	persistMu   sync.Mutex
}

// AccessLogStore handles access logging for redirects.
type AccessLogStore struct {
	mu     sync.RWMutex
	cfg    *config.Config
	logs   []AccessLog
	opened bool
}

// AccessLog represents a single access log entry.
type AccessLog struct {
	Code      string    `json:"code"`
	RawURL    string    `json:"raw_url"`
	Timestamp time.Time `json:"timestamp"`
	Status    int       `json:"status"`
}

// NewURLStore creates a new URLStore.
func NewURLStore(cfg *config.Config) (*URLStore, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}
	store := &URLStore{
		cfg:  cfg,
		data: make(map[string]model.ShortURL),
	}
	if err := store.loadData(); err != nil {
		return nil, fmt.Errorf("failed to load URL data: %w", err)
	}
	logger.GetGlobal().Infof("URLStore initialized with %d entries", len(store.data))
	return store, nil
}

// NewAccessLogStore creates a new AccessLogStore.
func NewAccessLogStore(cfg *config.Config) (*AccessLogStore, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}
	return &AccessLogStore{
		cfg:  cfg,
		logs: make([]AccessLog, 0),
	}, nil
}

// Open initializes the access log store.
func (s *AccessLogStore) Open(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.opened {
		return fmt.Errorf("access log store already opened")
	}
	s.opened = true
	logger.GetGlobal().Info("AccessLogStore opened")
	return nil
}

// Close closes the access log store.
func (s *AccessLogStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.opened {
		return nil
	}
	s.opened = false
	return nil
}

// Load loads data from storage.
func (s *URLStore) Load(ctx context.Context) error {
	return s.loadData()
}

// Close closes the URL store and persists data.
func (s *URLStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveData()
}

// SetPanicGuard sets a guard function for panic during Save operations.
func (s *URLStore) SetPanicGuard(fn PanicGuardFn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.panicGuard = fn
}

// Save persists a ShortURL entry.
func (s *URLStore) Save(u *model.ShortURL, overwrite bool) error {
	if u == nil {
		return fmt.Errorf("short URL is nil")
	}
	if u.Code == "" {
		return fmt.Errorf("code is required")
	}
	
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if !overwrite {
		if _, exists := s.data[u.Code]; exists {
			return fmt.Errorf("code %s already exists", u.Code)
		}
	}
	
	if s.panicGuard != nil && s.panicGuard(u.Code, u.RawURL) {
		panic("panic guard triggered")
	}
	
	s.data[u.Code] = *u
	
	if s.cfg.Storage.GetFlushOnWrite() {
		return s.saveData()
	}
	return nil
}

// Get retrieves a ShortURL by code.
func (s *URLStore) Get(code string) (*model.ShortURL, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	url, exists := s.data[code]
	if !exists {
		return nil, fmt.Errorf("code %s not found", code)
	}
	return &url, nil
}

// RawSnapshot returns a raw snapshot of all URL entries.
func (s *URLStore) RawSnapshot() map[string]model.ShortURL {
	s.mu.RLock()
	defer s.mu.RUnlock()
	snapshot := make(map[string]model.ShortURL, len(s.data))
	for k, v := range s.data {
		snapshot[k] = v
	}
	return snapshot
}

// IncrementVisits increments the visit count for a code.
func (s *URLStore) IncrementVisits(code string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	url, exists := s.data[code]
	if !exists {
		return fmt.Errorf("code %s not found", code)
	}
	url.Visits++
	s.data[code] = url
	return nil
}

// ListByExpiry returns entries that match the expiry filter.
func (s *URLStore) ListByExpiry(maxAgeHours int) ([]model.ShortURL, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now().UTC()
	ttl := time.Duration(maxAgeHours) * time.Hour
	var results []model.ShortURL

	for _, url := range s.data {
		if url.Disabled {
			continue
		}
		if now.Sub(url.CreatedAt) <= ttl {
			results = append(results, url)
		}
	}

	return results, nil
}

// Cleanup removes expired entries based on max age in hours.
func (s *URLStore) Cleanup(maxAgeHours int) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().UTC().Add(-time.Duration(maxAgeHours) * time.Hour)
	removed := 0

	for id, url := range s.data {
		if url.CreatedAt.Before(cutoff) {
			delete(s.data, id)
			removed++
		}
	}

	if removed > 0 {
		if err := s.saveData(); err != nil {
			return removed, fmt.Errorf("failed to save after cleanup: %w", err)
		}
		logger.GetGlobal().Infof("Cleaned up %d expired URL entries", removed)
	}

	return removed, nil
}

// LogAccess logs an access event.
func (s *AccessLogStore) LogAccess(log AccessLog) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.opened {
		return fmt.Errorf("access log store not opened")
	}
	s.logs = append(s.logs, log)
	return nil
}

// GetLogs returns all access logs.
func (s *AccessLogStore) GetLogs() []AccessLog {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]AccessLog, len(s.logs))
	copy(result, s.logs)
	return result
}

// ClearLogs clears all access logs.
func (s *AccessLogStore) ClearLogs() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.logs = make([]AccessLog, 0)
	return nil
}

// loadData loads URL data from file.
func (s *URLStore) loadData() error {
	path := s.cfg.Storage.GetURLFilePath()
	if path == "" {
		return nil
	}
	
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	
	if len(data) == 0 {
		return nil
	}
	
	var urls map[string]model.ShortURL
	if err := json.Unmarshal(data, &urls); err != nil {
		return fmt.Errorf("failed to unmarshal URL data: %w", err)
	}
	
	s.data = urls
	return nil
}

// saveData saves URL data to file.
func (s *URLStore) saveData() error {
	s.persistMu.Lock()
	defer s.persistMu.Unlock()
	
	path := s.cfg.Storage.GetURLFilePath()
	if path == "" {
		return nil
	}
	
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}
	
	s.mu.RLock()
	urls := make(map[string]model.ShortURL, len(s.data))
	for k, v := range s.data {
		urls[k] = v
	}
	s.mu.RUnlock()
	
	data, err := json.MarshalIndent(urls, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal URL data: %w", err)
	}
	
	return os.WriteFile(path, data, 0644)
}
