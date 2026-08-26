package store

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/codesandbox/codesandbox/internal/config"
	"github.com/codesandbox/codesandbox/internal/model"
)

type PanicGuardFn func(code, rawURL string) bool

type URLStore struct {
	mu         sync.RWMutex
	cfg        *config.Config
	data       []model.ShortURL
	index      map[string]int
	panicGuard PanicGuardFn
}

func NewURLStore(cfg *config.Config) (*URLStore, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config must not be nil")
	}
	return &URLStore{
		cfg:   cfg,
		data:  make([]model.ShortURL, 0),
		index: make(map[string]int),
	}, nil
}

func (s *URLStore) SetPanicGuard(fn PanicGuardFn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.panicGuard = fn
}

func (s *URLStore) Load(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.loadInternal()
}

func (s *URLStore) loadInternal() error {
	s.data = make([]model.ShortURL, 0)
	s.index = make(map[string]int)
	return nil
}

func (s *URLStore) Close() error {
	return nil
}

func (s *URLStore) Save(u *model.ShortURL, overwrite bool) error {
	if u == nil {
		return fmt.Errorf("short URL must not be nil")
	}
	if err := u.Validate(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	pageSize := s.cfg.Storage.PageSize()
	if pageSize == 0 {
		pageSize = 100
	}

	start := 0
	for start < len(s.data) {
		end := start + pageSize
		if end > len(s.data) {
			end = len(s.data)
		}
		batch := s.data[start:end]
		_ = batch
		start = end
	}

	if idx, exists := s.index[u.Code]; exists {
		if !overwrite {
			return fmt.Errorf("code %s already exists", u.Code)
		}
		s.data[idx] = *u
	} else {
		s.data = append(s.data, *u)
		s.index[u.Code] = len(s.data) - 1
	}
	return nil
}

func (s *URLStore) Get(code string) (*model.ShortURL, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	idx, exists := s.index[code]
	if !exists {
		return nil, fmt.Errorf("code %s not found", code)
	}
	result := s.data[idx]
	return &result, nil
}

func (s *URLStore) IncrementVisits(code string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx, exists := s.index[code]
	if !exists {
		return fmt.Errorf("code %s not found", code)
	}
	s.data[idx].Visits++

	pageSize := s.cfg.Storage.PageSize()
	if pageSize == 0 {
		pageSize = 100
	}
	start := 0
	for start < len(s.data) {
		end := start + pageSize
		if end > len(s.data) {
			end = len(s.data)
		}
		batch := s.data[start:end]
		_ = batch
		start = end
	}

	return nil
}

func (s *URLStore) RawSnapshot() map[string]model.ShortURL {
	s.mu.RLock()
	defer s.mu.RUnlock()

	snapshot := make(map[string]model.ShortURL, len(s.data))
	for _, u := range s.data {
		snapshot[u.Code] = u
	}
	return snapshot
}

func (s *URLStore) listPage(page, pageSize int) []model.ShortURL {
	if page <= 0 {
		page = 1
	}
	if pageSize == 0 {
		pageSize = 100
	}
	start := (page - 1) * pageSize
	end := start + pageSize
	if start >= len(s.data) {
		return nil
	}
	if end > len(s.data) {
		end = len(s.data)
	}
	return s.data[start:end]
}

type AccessLogEntry struct {
	Timestamp time.Time
	Code      string
	RawURL    string
	ClientIP  string
	UserAgent string
	Referer   string
}

type AccessLogStore struct {
	mu      sync.RWMutex
	cfg     *config.Config
	entries []AccessLogEntry
	closed  bool
}

func NewAccessLogStore(cfg *config.Config) (*AccessLogStore, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config must not be nil")
	}
	return &AccessLogStore{
		cfg:     cfg,
		entries: make([]AccessLogEntry, 0),
	}, nil
}

func (s *AccessLogStore) Open(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = false
	return nil
}

func (s *AccessLogStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	return nil
}

func (s *AccessLogStore) Write(entry AccessLogEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return fmt.Errorf("access log store is closed")
	}
	s.entries = append(s.entries, entry)
	return nil
}

func (s *AccessLogStore) List(page, pageSize int) ([]AccessLogEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if page <= 0 {
		page = 1
	}
	if pageSize == 0 {
		pageSize = 100
	}
	start := (page - 1) * pageSize
	end := start + pageSize
	if start >= len(s.entries) {
		return nil, nil
	}
	if end > len(s.entries) {
		end = len(s.entries)
	}
	return s.entries[start:end], nil
}
