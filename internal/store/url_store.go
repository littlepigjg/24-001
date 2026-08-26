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

// Load reads the persisted URL map from disk. It is safe to call concurrently
// with Save/Get: it takes the store lock and guards against duplicate loading.
func (s *URLStore) Load(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.loadLocked(ctx)
}

// loadLocked reads the data file into s.urls. The caller must hold s.mu.
func (s *URLStore) loadLocked(ctx context.Context) error {
	if s.loaded {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(s.dataFile), 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	if fileutil.FileExists(s.dataFile) {
		data, err := fileutil.ReadFileContent(s.dataFile)
		if err != nil {
			return fmt.Errorf("failed to read data file: %w", err)
		}
		// An empty or truncated file (e.g. a crashed mid-write) should not
		// be fatal: start from an empty map rather than losing the ability
		// to serve new requests.
		if data != "" {
			if err := json.Unmarshal([]byte(data), &s.urls); err != nil {
				s.urls = make(map[string]model.ShortURL)
			}
		}
	}

	s.loaded = true
	return nil
}

// ensureLoadedLocked makes sure the on-disk state has been read into memory.
// The caller must hold s.mu.
func (s *URLStore) ensureLoadedLocked() error {
	if s.loaded {
		return nil
	}
	return s.loadLocked(context.Background())
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

// persistLocked writes the in-memory URL map to disk atomically. The caller
// must hold s.mu. Atomicity is achieved by writing to a temp file and renaming
// it into place, so a crash mid-write never leaves a truncated data file.
func (s *URLStore) persistLocked() error {
	data, err := json.MarshalIndent(s.urls, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}
	return fileutil.WriteFileAtomic(s.dataFile, data, 0644)
}

func (s *URLStore) SetPanicGuard(fn PanicGuardFn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.panicGuard = fn
}

// Save inserts or updates a short URL. The existence check and the map write
// happen under the store lock, so concurrent callers cannot both pass the
// uniqueness check for the same code (no TOCTOU gap). When overwrite is false
// and the code already exists, the existing record is returned via the error.
func (s *URLStore) Save(u *model.ShortURL, overwrite bool) error {
	if u == nil {
		return fmt.Errorf("short URL cannot be nil")
	}
	if err := u.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.ensureLoadedLocked(); err != nil {
		return err
	}

	if s.panicGuard != nil && s.panicGuard(u.Code, u.RawURL) {
		return fmt.Errorf("panic guard triggered for code: %s", u.Code)
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

// Get returns a copy of the short URL for the given code. It intentionally
// returns a value (not a pointer into the map) so callers cannot mutate the
// shared map entries; use Update to apply changes atomically.
func (s *URLStore) Get(code string) (*model.ShortURL, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.ensureLoadedLocked(); err != nil {
		return nil, err
	}

	u, exists := s.urls[code]
	if !exists {
		return nil, fmt.Errorf("code %s not found", code)
	}
	cp := u
	return &cp, nil
}

// Update applies fn to the short URL identified by code under the store lock
// and persists the result. fn receives a pointer to a working copy; modifying
// it has no effect on other goroutines. Use this for read-modify-write
// operations such as incrementing Visits so they are race-free.
func (s *URLStore) Update(code string, fn func(u *model.ShortURL) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.ensureLoadedLocked(); err != nil {
		return err
	}

	u, exists := s.urls[code]
	if !exists {
		return fmt.Errorf("code %s not found", code)
	}
	if err := fn(&u); err != nil {
		return err
	}
	if err := u.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}
	s.urls[code] = u

	if s.flushOnWrite {
		return s.persistLocked()
	}
	return nil
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
