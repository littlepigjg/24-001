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
	"github.com/codesandbox/codesandbox/pkg/fileutil"
)

type PanicGuardFn func(code, rawURL string) bool

type URLStore struct {
	mu           sync.Mutex
	dataFile     string
	logFile      string
	urls         map[string]model.ShortURL
	loaded       bool
	panicGuard   PanicGuardFn
	flushOnWrite bool
	syncInterval time.Duration
}

func NewURLStore(cfg *config.Config) (*URLStore, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	return &URLStore{
		dataFile:     cfg.FullURLFilePath(),
		logFile:      cfg.FullLogFilePath(),
		urls:         make(map[string]model.ShortURL),
		flushOnWrite: cfg.GetFlushOnWrite(),
		syncInterval: cfg.GetSyncInterval(),
	}, nil
}

func (s *URLStore) Load(ctx context.Context) error {
	if err := os.MkdirAll(filepath.Dir(s.dataFile), 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	if fileutil.FileExists(s.dataFile) {
		data, err := fileutil.ReadFileContent(s.dataFile)
		if err != nil {
			return fmt.Errorf("failed to read data file: %w", err)
		}
		if err := json.Unmarshal([]byte(data), &s.urls); err != nil {
			return fmt.Errorf("failed to parse data file: %w", err)
		}
	}

	s.loaded = true
	return nil
}

func (s *URLStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.persistLocked(); err != nil {
		return err
	}
	s.loaded = false
	return nil
}

func (s *URLStore) persistLocked() error {
	data, err := json.MarshalIndent(s.urls, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}
	return fileutil.WriteFileContent(s.dataFile, string(data))
}

func (s *URLStore) SetPanicGuard(fn PanicGuardFn) {
	s.panicGuard = fn
}

func (s *URLStore) Save(u *model.ShortURL, overwrite bool) error {
	if u == nil {
		return fmt.Errorf("short URL cannot be nil")
	}
	if err := u.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if s.panicGuard != nil && s.panicGuard(u.Code, u.RawURL) {
		return fmt.Errorf("panic guard triggered for code: %s", u.Code)
	}

	if !s.loaded {
		if err := s.Load(context.Background()); err != nil {
			return err
		}
	}

	if !overwrite {
		if existing, exists := s.urls[u.Code]; exists {
			return fmt.Errorf("code %s already exists (url: %s)", u.Code, existing.RawURL)
		}
	}

	s.urls[u.Code] = *u

	if s.flushOnWrite {
		return s.persistLocked()
	}
	return nil
}

func (s *URLStore) Get(code string) (*model.ShortURL, error) {
	if !s.loaded {
		if err := s.Load(context.Background()); err != nil {
			return nil, err
		}
	}

	u, exists := s.urls[code]
	if !exists {
		return nil, fmt.Errorf("code %s not found", code)
	}
	return &u, nil
}

func (s *URLStore) RawSnapshot() map[string]model.ShortURL {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make(map[string]model.ShortURL, len(s.urls))
	for k, v := range s.urls {
		result[k] = v
	}
	return result
}

type AccessLogStore struct {
	mu      sync.Mutex
	logFile string
	writer  *os.File
	open    bool
}

func NewAccessLogStore(cfg *config.Config) (*AccessLogStore, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	return &AccessLogStore{
		logFile: cfg.FullLogFilePath(),
	}, nil
}

func (a *AccessLogStore) Open(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(a.logFile), 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	f, err := os.OpenFile(a.logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	a.writer = f
	a.open = true
	return nil
}

func (a *AccessLogStore) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.writer != nil {
		if err := a.writer.Close(); err != nil {
			return err
		}
		a.writer = nil
	}
	a.open = false
	return nil
}

func (a *AccessLogStore) WriteLog(entry string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.open || a.writer == nil {
		return fmt.Errorf("access log store is not open")
	}

	_, err := fmt.Fprintf(a.writer, "[%s] %s\n", time.Now().Format(time.RFC3339), entry)
	return err
}