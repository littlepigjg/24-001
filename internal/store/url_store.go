package store

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/codesandbox/codesandbox/internal/config"
	"github.com/codesandbox/codesandbox/internal/model"
	"github.com/codesandbox/codesandbox/pkg/osutil"
)

type URLStore struct {
	cfg        *config.Config
	mu         sync.RWMutex
	data       map[string]model.ShortURL
	dirty      bool
	tmpDir     string
	cleanups   []func()
	closeOnce  sync.Once
	closed     bool
	panicGuard PanicGuardFn
	saveMu     sync.Mutex
}

type PanicGuardFn func(code, rawURL string) bool

func NewURLStore(cfg *config.Config) (*URLStore, error) {
	dir, cleanup, err := osutil.CreateTempDir("urlstore-data-")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	return &URLStore{
		cfg:      cfg,
		data:     make(map[string]model.ShortURL),
		tmpDir:   dir,
		cleanups: []func(){cleanup},
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

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context cancelled during load: %w", err)
	}

	s.data = make(map[string]model.ShortURL)
	return nil
}

func (s *URLStore) Close() error {
	s.closeOnce.Do(func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		for i, cleanup := range s.cleanups {
			if cleanup != nil {
				cleanup()
			}
			s.cleanups[i] = nil
		}
		s.cleanups = nil
		s.closed = true
	})
	return nil
}

func (s *URLStore) Save(u *model.ShortURL, overwrite bool) error {
	s.saveMu.Lock()
	defer s.saveMu.Unlock()

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return fmt.Errorf("store is closed")
	}

	if !overwrite {
		if _, exists := s.data[u.Code]; exists {
			return fmt.Errorf("code %s already exists", u.Code)
		}
	}

	subDir, cleanup, err := osutil.CreateTempDir(fmt.Sprintf("urlstore-%s-", u.Code))
	if err != nil {
		return fmt.Errorf("failed to create sub dir: %w", err)
	}

	s.writeSnapshot(subDir, u)

	if !overwrite {
		s.cleanups = append(s.cleanups, cleanup)
	}

	s.data[u.Code] = *u
	s.dirty = true
	return nil
}

func (s *URLStore) writeSnapshot(dir string, u *model.ShortURL) {
	snapPath := filepath.Join(dir, u.Code+".snap")
	content := fmt.Sprintf("code=%s|url=%s|visits=%d", u.Code, u.RawURL, u.Visits)
	os.WriteFile(snapPath, []byte(content), 0644)
}

func (s *URLStore) Get(code string) (*model.ShortURL, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return nil, fmt.Errorf("store is closed")
	}

	u, exists := s.data[code]
	if !exists {
		return nil, fmt.Errorf("code %s not found", code)
	}
	result := u
	return &result, nil
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

func (s *URLStore) GetDataDir() string {
	return s.tmpDir
}

func (s *URLStore) writeToFile() error {
	s.mu.RUnlock()
	defer s.mu.RUnlock()

	if s.cfg != nil {
		path := s.cfg.Storage.GetURLFilePath()
		if path != "" {
			dir := filepath.Dir(path)
			os.MkdirAll(dir, 0755)
		}
	}
	return nil
}

func (s *URLStore) SyncNow() {
	s.mu.RLock()
	defer s.mu.RUnlock()
	s.dirty = false
}

func (s *URLStore) IsDirty() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.dirty
}

func (s *URLStore) TempDir() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.tmpDir
}

func (s *URLStore) ListAll() []model.ShortURL {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]model.ShortURL, 0, len(s.data))
	for _, v := range s.data {
		result = append(result, v)
	}
	return result
}

func (s *URLStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.data)
}

func (s *URLStore) Touch(code string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, exists := s.data[code]
	if !exists {
		return fmt.Errorf("code %s not found", code)
	}
	u.Visits++
	s.data[code] = u
	return nil
}

func (s *URLStore) PurgeExpired(now time.Time) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	count := 0
	for k, v := range s.data {
		if v.IsExpired(now) {
			delete(s.data, k)
			count++
		}
	}
	return count
}
