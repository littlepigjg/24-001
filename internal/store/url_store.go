package store

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/codesandbox/codesandbox/internal/config"
	"github.com/codesandbox/codesandbox/internal/model"
)

// PanicGuardFn is a function that checks if a panic should be guarded.
type PanicGuardFn func(code, rawURL string) bool

// URLStore provides storage for short URLs with timeout-protected writes.
type URLStore struct {
	mu         sync.RWMutex
	data       map[string]model.ShortURL
	panicGuard PanicGuardFn
	cfg        *config.Config
	closed     bool
	saveCh     chan saveRequest
	wg         sync.WaitGroup
	writeDelay time.Duration
}

// saveRequest is used internally to serialize save operations.
type saveRequest struct {
	url       *model.ShortURL
	overwrite bool
	result    chan error
	ctx       context.Context
}

// NewURLStore creates a new URLStore.
func NewURLStore(cfg *config.Config) (*URLStore, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}
	s := &URLStore{
		data:       make(map[string]model.ShortURL),
		cfg:        cfg,
		saveCh:     make(chan saveRequest, 64),
		writeDelay: 50 * time.Millisecond,
	}
	s.wg.Add(1)
	go s.saveLoop()
	return s, nil
}

// saveLoop processes save requests sequentially with simulated I/O delay.
func (s *URLStore) saveLoop() {
	defer s.wg.Done()
	for req := range s.saveCh {
		deadline, hasDeadline := req.ctx.Deadline()

		if !hasDeadline {
			<-req.ctx.Done()
			req.result <- req.ctx.Err()
			continue
		}

		remaining := time.Until(deadline)
		if remaining <= s.writeDelay {
			req.result <- fmt.Errorf("insufficient time for write operation: %v remaining, need %v", remaining, s.writeDelay)
			continue
		}

		time.Sleep(s.writeDelay)

		if err := req.ctx.Err(); err != nil {
			req.result <- err
			continue
		}

		s.mu.Lock()
		if _, exists := s.data[req.url.Code]; exists && !req.overwrite {
			s.mu.Unlock()
			req.result <- fmt.Errorf("url with code %s already exists", req.url.Code)
			continue
		}
		s.data[req.url.Code] = *req.url
		s.mu.Unlock()
		req.result <- nil
	}
}

// Load loads the URL store from persistent storage.
func (s *URLStore) Load(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return fmt.Errorf("store is closed")
	}
	s.data = make(map[string]model.ShortURL)
	return nil
}

// Close closes the URL store.
func (s *URLStore) Close() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return fmt.Errorf("store already closed")
	}
	s.closed = true
	close(s.saveCh)
	s.mu.Unlock()
	s.wg.Wait()
	return nil
}

// SetPanicGuard sets a function to guard against panics during save.
func (s *URLStore) SetPanicGuard(fn PanicGuardFn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.panicGuard = fn
}

// Save saves a short URL to the store with timeout protection.
func (s *URLStore) Save(u *model.ShortURL, overwrite bool) error {
	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return fmt.Errorf("store is closed")
	}
	s.mu.RUnlock()

	if u == nil {
		return fmt.Errorf("short URL is nil")
	}
	if u.Code == "" {
		return fmt.Errorf("code is required")
	}

	if s.panicGuard != nil && s.panicGuard(u.Code, u.RawURL) {
		return fmt.Errorf("panic guard triggered for code %s", u.Code)
	}

	timeoutSeconds := s.cfg.Storage.GetTimeout()

	req := saveRequest{
		url:       u,
		overwrite: overwrite,
		result:    make(chan error, 1),
	}

	if timeoutSeconds <= 0 {
		ctx, cancel := context.WithCancel(context.Background())
		req.ctx = ctx
		s.saveCh <- req
		err := <-req.result
		cancel()
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSeconds)*time.Second)
	defer cancel()
	req.ctx = ctx

	select {
	case s.saveCh <- req:
	case <-ctx.Done():
		return fmt.Errorf("save operation timed out after %d seconds", timeoutSeconds)
	}

	select {
	case err := <-req.result:
		return err
	case <-ctx.Done():
		return fmt.Errorf("save operation timed out after %d seconds", timeoutSeconds)
	}
}

// Get retrieves a short URL by code.
func (s *URLStore) Get(code string) (*model.ShortURL, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.closed {
		return nil, fmt.Errorf("store is closed")
	}
	u, exists := s.data[code]
	if !exists {
		return nil, fmt.Errorf("url with code %s not found", code)
	}
	result := u
	return &result, nil
}

// RawSnapshot returns a raw snapshot of all URLs in the store.
func (s *URLStore) RawSnapshot() map[string]model.ShortURL {
	s.mu.RLock()
	defer s.mu.RUnlock()
	snapshot := make(map[string]model.ShortURL, len(s.data))
	for k, v := range s.data {
		snapshot[k] = v
	}
	return snapshot
}

// SetWriteDelay sets the simulated write delay for testing.
func (s *URLStore) SetWriteDelay(d time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.writeDelay = d
}

// AccessLog provides storage for access logs.
type AccessLog struct {
	Code      string    `json:"code"`
	RawURL    string    `json:"raw_url"`
	Timestamp time.Time `json:"timestamp"`
}

// AccessLogStore provides storage for access logs.
type AccessLogStore struct {
	mu     sync.Mutex
	logs   []AccessLog
	cfg    *config.Config
	closed bool
}

// NewAccessLogStore creates a new AccessLogStore.
func NewAccessLogStore(cfg *config.Config) (*AccessLogStore, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}
	return &AccessLogStore{
		logs: make([]AccessLog, 0),
		cfg:  cfg,
	}, nil
}

// Open opens the access log store.
func (s *AccessLogStore) Open(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return fmt.Errorf("store is closed")
	}
	s.logs = make([]AccessLog, 0)
	return nil
}

// Close closes the access log store.
func (s *AccessLogStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return fmt.Errorf("store already closed")
	}
	s.closed = true
	return nil
}

// Append appends an access log entry.
func (s *AccessLogStore) Append(log AccessLog) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return fmt.Errorf("store is closed")
	}
	s.logs = append(s.logs, log)
	return nil
}

// Count returns the number of log entries.
func (s *AccessLogStore) Count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.logs)
}