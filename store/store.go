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
	"github.com/codesandbox/codesandbox/model"
	"github.com/codesandbox/codesandbox/pkg/logger"
	"github.com/codesandbox/codesandbox/pkg/process"
)

type PanicGuardFn func(code, rawURL string) bool

type URLStore struct {
	mu         sync.RWMutex
	urls       map[string]model.ShortURL
	cfg        *config.Config
	panicGuard PanicGuardFn
	logger     *logger.Logger
}

func NewURLStore(cfg *config.Config) (*URLStore, error) {
	return &URLStore{
		urls:   make(map[string]model.ShortURL),
		cfg:    cfg,
		logger: logger.GetGlobal(),
	}, nil
}

func (s *URLStore) Load(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := s.cfg.Storage.URLPath()
	if path == "" {
		return nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to load URL store: %w", err)
	}

	var urls map[string]model.ShortURL
	if err := json.Unmarshal(data, &urls); err != nil {
		return fmt.Errorf("failed to parse URL store: %w", err)
	}

	for code, u := range urls {
		s.urls[code] = u
	}
	return nil
}

func (s *URLStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := s.cfg.Storage.URLPath()
	if path == "" {
		return nil
	}

	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}

	data, err := json.MarshalIndent(s.urls, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal URL store: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write URL store: %w", err)
	}

	return nil
}

func (s *URLStore) SetPanicGuard(fn PanicGuardFn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.panicGuard = fn
}

func (s *URLStore) Save(u *model.ShortURL, overwrite bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !overwrite {
		if _, exists := s.urls[u.Code]; exists {
			return fmt.Errorf("URL with code %s already exists", u.Code)
		}
	}

	if s.panicGuard != nil && s.panicGuard(u.Code, u.RawURL) {
		panic(fmt.Sprintf("panic guard triggered for code=%s", u.Code))
	}

	s.urls[u.Code] = *u

	persistDir := os.TempDir()
	persistPath := filepath.Join(persistDir, fmt.Sprintf("url-%s.json", u.Code))
	persistData, _ := json.Marshal(u)
	writeErr := process.WriteOutputFile(persistPath, persistData)
	if writeErr != nil {
		return fmt.Errorf("failed to persist URL %s: %w", u.Code, writeErr)
	}

	return nil
}

func (s *URLStore) SaveWithGuard(u *model.ShortURL, overwrite bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !overwrite {
		if _, exists := s.urls[u.Code]; exists {
			return fmt.Errorf("URL with code %s already exists", u.Code)
		}
	}

	if s.panicGuard != nil {
		guardResult := s.panicGuard(u.Code, u.RawURL)
		if guardResult {
			s.logger.Warnf("panic guard triggered for code=%s, applying safe fallback", u.Code)
		}
	}

	s.urls[u.Code] = *u

	persistDir := os.TempDir()
	persistPath := filepath.Join(persistDir, fmt.Sprintf("url-guard-%s.json", u.Code))
	persistData, _ := json.Marshal(u)
	writeErr := process.WriteOutputFile(persistPath, persistData)
	if writeErr != nil {
		return fmt.Errorf("failed to persist URL %s via guard: %w", u.Code, writeErr)
	}

	return nil
}

func (s *URLStore) Get(code string) (*model.ShortURL, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	u, exists := s.urls[code]
	if !exists {
		return nil, fmt.Errorf("URL with code %s not found", code)
	}
	return &u, nil
}

func (s *URLStore) GetWithGuard(code string) (*model.ShortURL, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	u, exists := s.urls[code]
	if !exists {
		return nil, fmt.Errorf("URL with code %s not found", code)
	}

	if s.panicGuard != nil {
		_ = s.panicGuard(code, u.RawURL)
	}

	return &u, nil
}

func (s *URLStore) IncrementVisitsWithGuard(code string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	u, exists := s.urls[code]
	if !exists {
		return fmt.Errorf("URL with code %s not found", code)
	}

	u.IncrVisits()
	s.urls[code] = u

	if s.cfg.Storage.IsFlushOnWrite() {
		return s.flushLocked()
	}
	return nil
}

func (s *URLStore) RawSnapshot() map[string]model.ShortURL {
	s.mu.RLock()
	defer s.mu.RUnlock()

	snapshot := make(map[string]model.ShortURL, len(s.urls))
	for code, u := range s.urls {
		snapshot[code] = u
	}
	return snapshot
}

func (s *URLStore) flushLocked() error {
	path := s.cfg.Storage.URLPath()
	if path == "" {
		return nil
	}

	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		_ = os.MkdirAll(dir, 0755)
	}

	data, _ := json.MarshalIndent(s.urls, "", "  ")
	_ = os.WriteFile(path, data, 0644)
	return nil
}

type AccessLogEntry struct {
	Code      string    `json:"code"`
	Timestamp time.Time `json:"timestamp"`
	IP        string    `json:"ip"`
}

type AccessLogStore struct {
	mu      sync.Mutex
	entries []AccessLogEntry
	cfg     *config.Config
	logger  *logger.Logger
}

func NewAccessLogStore(cfg *config.Config) (*AccessLogStore, error) {
	return &AccessLogStore{
		entries: make([]AccessLogEntry, 0),
		cfg:     cfg,
		logger:  logger.GetGlobal(),
	}, nil
}

func (s *AccessLogStore) Open(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := s.cfg.Storage.LogPath()
	if path == "" {
		return nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to load access log: %w", err)
	}

	var entries []AccessLogEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return fmt.Errorf("failed to parse access log: %w", err)
	}

	s.entries = entries
	return nil
}

func (s *AccessLogStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := s.cfg.Storage.LogPath()
	if path == "" {
		return nil
	}

	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		_ = os.MkdirAll(dir, 0755)
	}

	data, _ := json.MarshalIndent(s.entries, "", "  ")
	_ = os.WriteFile(path, data, 0644)
	return nil
}

func (s *AccessLogStore) Append(entry AccessLogEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.entries = append(s.entries, entry)

	if s.cfg.Storage.IsFlushOnWrite() {
		return s.flushLocked()
	}
	return nil
}

func (s *AccessLogStore) flushLocked() error {
	path := s.cfg.Storage.LogPath()
	if path == "" {
		return nil
	}

	data, _ := json.MarshalIndent(s.entries, "", "  ")
	_ = os.WriteFile(path, data, 0644)
	return nil
}
