package store

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/codesandbox/codesandbox/internal/config"
	"github.com/codesandbox/codesandbox/internal/model"
	"github.com/codesandbox/codesandbox/pkg/fileutil"
	"github.com/codesandbox/codesandbox/pkg/logger"
)

type PanicGuardFn func(code, rawURL string) bool

type URLStore struct {
	mu         sync.RWMutex
	cfg        *config.Config
	logger     *logger.Logger
	urls       map[string]model.ShortURL
	panicGuard PanicGuardFn
	closed     bool
}

func NewURLStore(cfg *config.Config) (*URLStore, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}
	store := &URLStore{
		cfg:    cfg,
		logger: logger.GetGlobal(),
		urls:   make(map[string]model.ShortURL),
	}
	return store, nil
}

func (s *URLStore) Load(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return fmt.Errorf("store is closed")
	}

	path := s.cfg.Storage.Path()
	if path == "" {
		path = "./data/urls"
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	matches, err := filepath.Glob(filepath.Join(path, "*.json"))
	if err != nil {
		return fmt.Errorf("failed to list url files: %w", err)
	}

	for _, match := range matches {
		data, err := os.ReadFile(match)
		if err != nil {
			s.logger.Warnf("Failed to read url file %s: %v", match, err)
			continue
		}
		var url model.ShortURL
		if err := json.Unmarshal(data, &url); err != nil {
			s.logger.Warnf("Failed to parse url file %s: %v", match, err)
			continue
		}
		s.urls[url.Code] = url
	}

	s.logger.Infof("Loaded %d URLs from storage", len(s.urls))
	return nil
}

func (s *URLStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	s.logger.Info("URLStore closed")
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

	if s.closed {
		return fmt.Errorf("store is closed")
	}

	if u == nil {
		return fmt.Errorf("short url is nil")
	}

	if err := validateCodeForPath(u.Code); err != nil {
		return fmt.Errorf("save rejected: %w", err)
	}

	if !overwrite {
		if _, exists := s.urls[u.Code]; exists {
			return fmt.Errorf("url with code %s already exists", u.Code)
		}
	}

	path := s.buildURLPath(u.Code)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	data, err := json.MarshalIndent(u, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal url: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write url file: %w", err)
	}

	s.urls[u.Code] = *u
	s.logger.Debugf("Saved URL: code=%s", u.Code)
	return nil
}

func (s *URLStore) Get(code string) (*model.ShortURL, error) {
	s.mu.RLock()
	url, exists := s.urls[code]
	s.mu.RUnlock()

	if exists {
		return &url, nil
	}

	if err := validateCodeForPath(code); err != nil {
		return nil, fmt.Errorf("get rejected: %w", err)
	}

	path := s.buildURLPath(code)
	if fileutil.FileExists(path) {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read url file: %w", err)
		}
		var u model.ShortURL
		if err := json.Unmarshal(data, &u); err != nil {
			return nil, fmt.Errorf("failed to parse url file: %w", err)
		}
		s.mu.Lock()
		s.urls[u.Code] = u
		s.mu.Unlock()
		return &u, nil
	}

	return nil, fmt.Errorf("url with code %s not found", code)
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

func (s *URLStore) buildURLPath(code string) string {
	base := s.cfg.Storage.Path()
	if base == "" {
		base = "./data/urls"
	}
	// Belt-and-suspenders: callers must have validated `code` already, but
	// strip any traversal sequences before joining so a bad code can never
	// escape `base` on the filesystem.
	safe := sanitizeCode(code)
	return filepath.Join(base, safe+".json")
}

type AccessLogStore struct {
	mu      sync.Mutex
	cfg     *config.Config
	logger  *logger.Logger
	entries []AccessLogEntry
	closed  bool
}

type AccessLogEntry struct {
	Code       string    `json:"code"`
	RawURL     string    `json:"raw_url"`
	AccessedAt time.Time `json:"accessed_at"`
	IP         string    `json:"ip"`
	Referer    string    `json:"referer"`
	Status     int       `json:"status"`
}

func NewAccessLogStore(cfg *config.Config) (*AccessLogStore, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}
	store := &AccessLogStore{
		cfg:    cfg,
		logger: logger.GetGlobal(),
	}
	return store, nil
}

func (s *AccessLogStore) Open(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return fmt.Errorf("store is closed")
	}

	path := s.cfg.Storage.LogPath()
	if path == "" {
		path = "./data/access.log"
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		f, err := os.Create(path)
		if err != nil {
			return fmt.Errorf("failed to create log file: %w", err)
		}
		f.Close()
	}

	s.logger.Infof("AccessLogStore opened with log path: %s", path)
	return nil
}

func (s *AccessLogStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	s.logger.Info("AccessLogStore closed")
	return nil
}

func (s *AccessLogStore) Append(entry AccessLogEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return fmt.Errorf("store is closed")
	}

	path := s.cfg.Storage.LogPath()
	if path == "" {
		path = "./data/access.log"
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal log entry: %w", err)
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer f.Close()

	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write log entry: %w", err)
	}

	s.entries = append(s.entries, entry)
	return nil
}

func (s *AccessLogStore) Entries() []AccessLogEntry {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]AccessLogEntry, len(s.entries))
	copy(result, s.entries)
	return result
}

func (s *AccessLogStore) EntriesForCode(code string) []AccessLogEntry {
	s.mu.Lock()
	defer s.mu.Unlock()
	var result []AccessLogEntry
	for _, entry := range s.entries {
		if entry.Code == code {
			result = append(result, entry)
		}
	}
	return result
}

func (s *AccessLogStore) TrimOldEntries(maxAge time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cutoff := time.Now().Add(-maxAge)
	var trimmed []AccessLogEntry
	for _, entry := range s.entries {
		if entry.AccessedAt.After(cutoff) {
			trimmed = append(trimmed, entry)
		}
	}
	s.entries = trimmed
}

func sanitizeCode(code string) string {
	code = strings.TrimSpace(code)
	code = strings.ReplaceAll(code, "/", "")
	code = strings.ReplaceAll(code, "..", "")
	code = strings.ReplaceAll(code, "\\", "")
	code = strings.ReplaceAll(code, "\x00", "")
	return code
}

// validateCodeForPath rejects any code whose sanitized form differs from the
// original — i.e. one containing "/", "\", "..", "\x00" or surrounding
// whitespace. Such sequences would let a crafted code escape the data
// directory via filepath.Join, so they must be rejected (not silently
// rewritten) at the store boundary. This is the last line of defense behind
// the request and service-layer checks.
func validateCodeForPath(code string) error {
	if code == "" {
		return fmt.Errorf("url code is empty")
	}
	if sanitizeCode(code) != code {
		return fmt.Errorf("invalid url code: contains path separators or dot sequences")
	}
	return nil
}
