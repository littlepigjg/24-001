package store

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/codesandbox/codesandbox/internal/config"
)

type AccessLogEntry struct {
	Code      string
	Timestamp time.Time
	Referer   string
	UserAgent string
}

type AccessLogStore struct {
	mu      sync.RWMutex
	logs    []AccessLogEntry
	cfg     *config.Config
	opened  bool
}

func NewAccessLogStore(cfg *config.Config) (*AccessLogStore, error) {
	if cfg == nil {
		return nil, errors.New("config cannot be nil")
	}
	return &AccessLogStore{
		logs: make([]AccessLogEntry, 0),
		cfg:  cfg,
	}, nil
}

func (s *AccessLogStore) Open(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.opened = true
	return nil
}

func (s *AccessLogStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.opened = false
	return nil
}

func (s *AccessLogStore) LogAccess(code string, timestamp time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.opened {
		return errors.New("access log store is not open")
	}

	if code == "" {
		return errors.New("code is required for access log")
	}

	s.logs = append(s.logs, AccessLogEntry{
		Code:      code,
		Timestamp: timestamp,
	})
	return nil
}

func (s *AccessLogStore) RecentLogs(n int) []AccessLogEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if n <= 0 || n > len(s.logs) {
		n = len(s.logs)
	}
	result := make([]AccessLogEntry, n)
	copy(result, s.logs[len(s.logs)-n:])
	return result
}

func (s *AccessLogStore) LogCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.logs)
}
