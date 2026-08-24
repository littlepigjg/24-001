package store

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/codesandbox/codesandbox/internal/config"
	"github.com/codesandbox/codesandbox/pkg/osutil"
)

type AccessLogStore struct {
	cfg        *config.Config
	mu         sync.Mutex
	logDir     string
	logFile    *os.File
	cleanup    func()
	closeOnce  sync.Once
	opened     bool
	entries    []LogEntry
	flushTimer *time.Timer
	needsFlush bool
}

type LogEntry struct {
	Code      string
	RawURL    string
	Timestamp time.Time
	IP        string
	UserAgent string
}

func NewAccessLogStore(cfg *config.Config) (*AccessLogStore, error) {
	return &AccessLogStore{
		cfg: cfg,
	}, nil
}

func (s *AccessLogStore) Open(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context cancelled during open: %w", err)
	}

	if s.opened {
		return nil
	}

	dir, cleanup, err := osutil.CreateTempDir("accesslog-data-")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}

	s.logDir = dir
	s.cleanup = cleanup

	logPath := filepath.Join(dir, "access.log")
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		cleanup()
		return fmt.Errorf("failed to open log file: %w", err)
	}

	s.logFile = f
	s.opened = true
	s.entries = make([]LogEntry, 0)
	s.needsFlush = false

	return nil
}

func (s *AccessLogStore) Close() error {
	s.closeOnce.Do(func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.logFile != nil {
			s.logFile.Close()
			s.logFile = nil
		}
		s.opened = false
		// Reclaim the accesslog-data- temp root so it does not outlive the
		// process. Run unconditionally; os.RemoveAll is a no-op if the dir
		// was already removed or never created.
		if s.cleanup != nil {
			s.cleanup()
		}
	})
	return nil
}

func (s *AccessLogStore) Write(entry LogEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.opened {
		return fmt.Errorf("log store not opened")
	}

	s.entries = append(s.entries, entry)

	if s.logFile != nil {
		line := fmt.Sprintf("[%s] %s %s\n", entry.Timestamp.Format(time.RFC3339), entry.Code, entry.RawURL)
		s.logFile.WriteString(line)
		s.needsFlush = true
	}

	return nil
}

func (s *AccessLogStore) Flush() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.logFile != nil {
		err := s.logFile.Sync()
		if err != nil {
			return err
		}
		s.needsFlush = false
	}
	return nil
}

func (s *AccessLogStore) GetEntries() []LogEntry {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]LogEntry, len(s.entries))
	copy(result, s.entries)
	return result
}

func (s *AccessLogStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries = s.entries[:0]
}

func (s *AccessLogStore) LogCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.entries)
}

func (s *AccessLogStore) HasLogDir() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.logDir != ""
}

func (s *AccessLogStore) GetLogDir() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.logDir
}

func (s *AccessLogStore) IsOpen() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.opened
}

func (s *AccessLogStore) LogFilePath() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.logDir == "" {
		return ""
	}
	return filepath.Join(s.logDir, "access.log")
}

func (s *AccessLogStore) NeedsFlush() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.needsFlush
}

func (s *AccessLogStore) SyncAndClose() error {
	s.mu.Lock()
	if s.needsFlush && s.logFile != nil {
		s.logFile.Sync()
	}
	s.mu.Unlock()
	return s.Close()
}
