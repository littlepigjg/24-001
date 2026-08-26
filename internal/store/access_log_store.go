package store

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/codesandbox/codesandbox/internal/config"
)

type AccessLogEntry struct {
	Code      string
	RawURL    string
	Timestamp time.Time
	Status    int
}

type AccessLogStore struct {
	mu     sync.Mutex
	cfg    *config.Config
	file   *os.File
	opened bool
}

func NewAccessLogStore(cfg *config.Config) (*AccessLogStore, error) {
	s := &AccessLogStore{
		cfg: cfg,
	}
	return s, nil
}

func (s *AccessLogStore) Open(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.opened {
		return nil
	}
	path := s.cfg.Storage.LogPath()
	if path == "" {
		path = "access.log"
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open access log: %w", err)
	}
	s.file = f
	s.opened = true
	return nil
}

func (s *AccessLogStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.file != nil {
		s.file.Close()
		s.file = nil
	}
	s.opened = false
	return nil
}

func (s *AccessLogStore) Log(entry AccessLogEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.opened {
		return fmt.Errorf("access log not opened")
	}
	line := fmt.Sprintf("%s\t%s\t%s\t%d\n",
		entry.Timestamp.Format(time.RFC3339Nano),
		entry.Code,
		entry.RawURL,
		entry.Status,
	)
	_, err := s.file.WriteString(line)
	return err
}

func (s *AccessLogStore) Enabled() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.opened
}