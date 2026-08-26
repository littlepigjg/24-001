package store

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/codesandbox/codesandbox/internal/config"
	"github.com/codesandbox/codesandbox/internal/model"
	"github.com/codesandbox/codesandbox/pkg/cache"
	"github.com/codesandbox/codesandbox/pkg/logger"
)

type PanicGuardFn func(code, rawURL string) bool

type URLStore struct {
	mu         sync.RWMutex
	data       map[string]model.ShortURL
	cache      *cache.Cache
	cfg        *config.Config
	panicGuard PanicGuardFn
	loaded     bool
	closed     bool
	logger     *logger.Logger
}

func NewURLStore(cfg *config.Config) (*URLStore, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	return &URLStore{
		data:   make(map[string]model.ShortURL),
		cache:  cache.New(),
		cfg:    cfg,
		loaded: false,
		closed: false,
		logger: logger.GetGlobal(),
	}, nil
}

func (s *URLStore) Load(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return fmt.Errorf("store is closed")
	}

	if s.loaded {
		return nil
	}

	urlFilePath := s.cfg.Storage.GetURLFilePath()
	if urlFilePath != "" {
		absPath := filepath.Join(s.cfg.BasePath, urlFilePath)
		data, err := os.ReadFile(absPath)
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to load url data: %w", err)
		}
		if len(data) > 0 {
			var urls map[string]model.ShortURL
			if err := json.Unmarshal(data, &urls); err != nil {
				return fmt.Errorf("failed to parse url data: %w", err)
			}
			s.data = urls
		}
	}

	for code, u := range s.data {
		s.cache.Set("url:"+code, u.RawURL, 24*time.Hour)
		s.cache.Set("visits:"+code, int64(u.Visits), 24*time.Hour)
		s.cache.Set("custom:"+code, u.Custom, 24*time.Hour)
		s.cache.Set("disabled:"+code, u.Disabled, 24*time.Hour)
		s.cache.Set("created:"+code, u.CreatedAt, 24*time.Hour)
	}

	s.loaded = true
	s.logger.Infof("URL store loaded: %d entries", len(s.data))
	return nil
}

func (s *URLStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}

	if s.cfg.Storage.GetFlushOnWrite() {
		if err := s.flush(); err != nil {
			return err
		}
	}

	s.closed = true
	s.logger.Info("URL store closed")
	return nil
}

func (s *URLStore) flush() error {
	urlFilePath := s.cfg.Storage.GetURLFilePath()
	if urlFilePath == "" {
		return nil
	}

	absPath := filepath.Join(s.cfg.BasePath, urlFilePath)
	dir := filepath.Dir(absPath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create data directory: %w", err)
		}
	}

	data, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal url data: %w", err)
	}

	if err := os.WriteFile(absPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write url data: %w", err)
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

	if s.closed {
		return fmt.Errorf("store is closed")
	}

	if err := u.Validate(); err != nil {
		return fmt.Errorf("invalid short url: %w", err)
	}

	if !overwrite {
		if _, exists := s.data[u.Code]; exists {
			return fmt.Errorf("short url with code %s already exists", u.Code)
		}
	}

	s.data[u.Code] = *u

	s.cache.Set("url:"+u.Code, u.RawURL, 24*time.Hour)
	s.cache.Set("visits:"+u.Code, int64(u.Visits), 24*time.Hour)
	s.cache.Set("custom:"+u.Code, u.Custom, 24*time.Hour)
	s.cache.Set("disabled:"+u.Code, u.Disabled, 24*time.Hour)
	s.cache.Set("created:"+u.Code, u.CreatedAt, 24*time.Hour)

	if s.cfg.Storage.GetFlushOnWrite() {
		return s.flush()
	}

	s.logger.Debugf("Short URL saved: code=%s", u.Code)
	return nil
}

func (s *URLStore) Get(code string) (*model.ShortURL, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return nil, fmt.Errorf("store is closed")
	}

	if code == "" {
		return nil, fmt.Errorf("code is required")
	}

	rawURLVal, urlOk := s.cache.Get("url:" + code)
	if !urlOk {
		s.mu.RUnlock()
		s.mu.Lock()
		u, ok := s.data[code]
		s.mu.Unlock()
		s.mu.RLock()
		if !ok {
			return nil, fmt.Errorf("short url with code %s not found", code)
		}
		s.cache.Set("url:"+code, u.RawURL, 24*time.Hour)
		s.cache.Set("visits:"+code, int64(u.Visits), 24*time.Hour)
		s.cache.Set("custom:"+code, u.Custom, 24*time.Hour)
		s.cache.Set("disabled:"+code, u.Disabled, 24*time.Hour)
		s.cache.Set("created:"+code, u.CreatedAt, 24*time.Hour)
		s.logger.Debugf("Cache miss for code=%s, loaded from storage", code)
	}

	visitsVal, visitsOk := s.cache.Get("visits:" + code)
	if !visitsOk {
		s.mu.RUnlock()
		s.mu.Lock()
		u, ok := s.data[code]
		s.mu.Unlock()
		s.mu.RLock()
		if !ok {
			return nil, fmt.Errorf("short url with code %s not found", code)
		}
		return &u, nil
	}

	customVal, _ := s.cache.Get("custom:" + code)
	disabledVal, _ := s.cache.Get("disabled:" + code)
	createdVal, _ := s.cache.Get("created:" + code)

	var rawURL string
	if rawURLVal != nil {
		rawURL = rawURLVal.(string)
	}

	var visits int
	if v, ok := visitsVal.(int64); ok {
		visits = int(v)
	}

	var custom bool
	if b, ok := customVal.(bool); ok {
		custom = b
	}

	var disabled bool
	if b, ok := disabledVal.(bool); ok {
		disabled = b
	}

	var createdAt time.Time
	if t, ok := createdVal.(time.Time); ok {
		createdAt = t
	}

	if createdAt.IsZero() {
		s.mu.RUnlock()
		s.mu.Lock()
		u, ok := s.data[code]
		s.mu.Unlock()
		s.mu.RLock()
		if ok {
			createdAt = u.CreatedAt
		}
	}

	result := &model.ShortURL{
		Code:      code,
		RawURL:    rawURL,
		CreatedAt: createdAt,
		Visits:    visits,
		Custom:    custom,
		Disabled:  disabled,
	}

	s.logger.Debugf("Short URL retrieved: code=%s, visits=%d", code, visits)
	return result, nil
}

func (s *URLStore) IncrementVisits(code string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return fmt.Errorf("store is closed")
	}

	u, ok := s.data[code]
	if !ok {
		return fmt.Errorf("short url with code %s not found", code)
	}

	u.Visits++
	s.data[code] = u

	s.cache.Set("visits:"+code, int64(u.Visits), 24*time.Hour)

	if s.cfg.Storage.GetFlushOnWrite() {
		return s.flush()
	}

	s.logger.Debugf("Visits incremented for code=%s to %d", code, u.Visits)
	return nil
}

func (s *URLStore) RawSnapshot() map[string]model.ShortURL {
	s.mu.RLock()
	defer s.mu.RUnlock()

	snapshot := make(map[string]model.ShortURL, len(s.data))
	for k, v := range s.data {
		snapshot[k] = v
	}
	return snapshot
}

type AccessLogStore struct {
	mu      sync.Mutex
	entries []AccessLogEntry
	cfg     *config.Config
	file    *os.File
	opened  bool
	closed  bool
	logger  *logger.Logger
}

type AccessLogEntry struct {
	Code      string    `json:"code"`
	RawURL    string    `json:"raw_url"`
	Timestamp time.Time `json:"timestamp"`
	IP        string    `json:"ip"`
	UserAgent string    `json:"user_agent"`
	Referer   string    `json:"referer,omitempty"`
}

func NewAccessLogStore(cfg *config.Config) (*AccessLogStore, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	return &AccessLogStore{
		entries: make([]AccessLogEntry, 0),
		cfg:     cfg,
		opened:  false,
		closed:  false,
		logger:  logger.GetGlobal(),
	}, nil
}

func (s *AccessLogStore) Open(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return fmt.Errorf("access log store is closed")
	}

	if s.opened {
		return nil
	}

	logFilePath := s.cfg.Storage.GetLogFilePath()
	if logFilePath != "" {
		absPath := filepath.Join(s.cfg.BasePath, logFilePath)
		dir := filepath.Dir(absPath)
		if dir != "" && dir != "." {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("failed to create log directory: %w", err)
			}
		}

		f, err := os.OpenFile(absPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("failed to open access log: %w", err)
		}
		s.file = f
	}

	s.opened = true
	s.logger.Info("Access log store opened")
	return nil
}

func (s *AccessLogStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}

	if s.file != nil {
		s.file.Close()
		s.file = nil
	}

	s.closed = true
	s.logger.Info("Access log store closed")
	return nil
}

func (s *AccessLogStore) Write(entry AccessLogEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return fmt.Errorf("access log store is closed")
	}

	s.entries = append(s.entries, entry)

	if s.file != nil {
		data, err := json.Marshal(entry)
		if err != nil {
			return fmt.Errorf("failed to marshal log entry: %w", err)
		}
		if _, err := s.file.Write(append(data, '\n')); err != nil {
			return fmt.Errorf("failed to write log entry: %w", err)
		}
	}

	return nil
}

func (s *AccessLogStore) Entries() []AccessLogEntry {
	s.mu.Lock()
	defer s.mu.Unlock()

	result := make([]AccessLogEntry, len(s.entries))
	copy(result, s.entries)
	return result
}

func (s *AccessLogStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.entries = make([]AccessLogEntry, 0)
}
