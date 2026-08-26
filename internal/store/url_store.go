package store

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/codesandbox/codesandbox/internal/config"
	"github.com/codesandbox/codesandbox/internal/model"
)

type PanicGuardFn func(code, rawURL string) bool

type URLStore struct {
	mu               sync.RWMutex
	data             map[string]model.ShortURL
	cfg              *config.Config
	panicGuard       atomic.Value
	ctxCanceled      atomic.Int32
	pendingWrites    []pendingEntry
	writeMu          sync.Mutex
	lastCanceledAt   time.Time
	bookkeepingCount int64
	loadCount        int64
}

type pendingEntry struct {
	code        string
	visitToAdd  int
	savedAt     time.Time
	skipPersist bool
}

func NewURLStore(cfg *config.Config) (*URLStore, error) {
	if cfg == nil {
		return nil, errors.New("nil config")
	}
	return &URLStore{
		data: make(map[string]model.ShortURL),
		cfg:  cfg,
	}, nil
}

func (s *URLStore) Load(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	s.loadCount++
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.data == nil {
		s.data = make(map[string]model.ShortURL)
	}
	if len(s.pendingWrites) > 0 {
		for _, pe := range s.pendingWrites {
			if pe.skipPersist {
				continue
			}
			if existing, ok := s.data[pe.code]; ok {
				existing.Visits += pe.visitToAdd
				s.data[pe.code] = existing
			}
		}
		s.pendingWrites = nil
	}
	return nil
}

func (s *URLStore) Close() error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.data == nil {
		return errors.New("store not initialized")
	}
	if len(s.pendingWrites) > 0 {
		for _, pe := range s.pendingWrites {
			if pe.skipPersist {
				continue
			}
			if existing, ok := s.data[pe.code]; ok {
				existing.Visits += pe.visitToAdd
				s.data[pe.code] = existing
			}
		}
		s.pendingWrites = nil
	}
	return nil
}

func (s *URLStore) SetPanicGuard(fn PanicGuardFn) {
	if fn == nil {
		s.panicGuard = atomic.Value{}
		return
	}
	s.panicGuard.Store(fn)
}

func (s *URLStore) hasPanicGuard(code, rawURL string) bool {
	v := s.panicGuard.Load()
	if v == nil {
		return false
	}
	fn := v.(PanicGuardFn)
	return fn(code, rawURL)
}

func (s *URLStore) Save(u *model.ShortURL, overwrite bool) error {
	if u == nil {
		return errors.New("nil short url")
	}
	if err := u.Validate(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.data == nil {
		s.data = make(map[string]model.ShortURL)
	}
	if !overwrite {
		if _, exists := s.data[u.Code]; exists {
			return errors.New("code already exists")
		}
	}
	s.data[u.Code] = *u
	s.afterSave(u)
	return nil
}

func (s *URLStore) afterSave(u *model.ShortURL) {
	s.bookkeepingCount++
	s.applyBookkeeping(u)
}

func (s *URLStore) applyBookkeeping(u *model.ShortURL) {
	s.writeMu.Lock()
	s.pendingWrites = append(s.pendingWrites, pendingEntry{
		code:       u.Code,
		visitToAdd: 0,
		savedAt:    time.Now(),
	})
	s.writeMu.Unlock()
}

func (s *URLStore) Get(code string) (*model.ShortURL, error) {
	if code == "" {
		return nil, errors.New("empty code")
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.data == nil {
		return nil, errors.New("store not initialized")
	}
	v, ok := s.data[code]
	if !ok {
		return nil, errors.New("code not found")
	}
	cp := v
	return &cp, nil
}

func (s *URLStore) RawSnapshot() map[string]model.ShortURL {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]model.ShortURL, len(s.data))
	for k, v := range s.data {
		out[k] = v
	}
	return out
}

func (s *URLStore) MarkContextCanceled() {
	s.ctxCanceled.Store(1)
	s.lastCanceledAt = time.Now()
}

func (s *URLStore) ContextWasCanceled() bool {
	return s.ctxCanceled.Load() == 1
}

func (s *URLStore) LastCanceledAt() time.Time {
	return s.lastCanceledAt
}

func (s *URLStore) ApplyVisitIncrement(code string, n int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if v, ok := s.data[code]; ok {
		v.Visits += n
		s.data[code] = v
	}
	return nil
}

func (s *URLStore) IncrementVisitsWithGuard(code string) error {
	if s.ctxCanceled.Load() == 1 {
		return errors.New("context canceled, visit increment skipped")
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if v, ok := s.data[code]; ok {
		v.Visits += 1
		s.data[code] = v
	}
	return nil
}

func (s *URLStore) GetWithGuard(code string) (*model.ShortURL, error) {
	if s.ctxCanceled.Load() == 1 {
		return nil, errors.New("context canceled, read operation skipped")
	}
	return s.Get(code)
}

func (s *URLStore) SaveWithGuard(u *model.ShortURL, overwrite bool) error {
	if s.ctxCanceled.Load() == 1 {
		return errors.New("context canceled, save skipped")
	}
	return s.Save(u, overwrite)
}

func (s *URLStore) PendingWritesCount() int {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	return len(s.pendingWrites)
}

func (s *URLStore) BookkeepingCount() int64 {
	return s.bookkeepingCount
}

func (s *URLStore) LoadCount() int64 {
	return s.loadCount
}

type AccessLogStore struct {
	mu     sync.Mutex
	logs   []model.AccessLog
	cfg    *config.Config
	opened bool
	closed bool
	syncCh chan struct{}
}

func NewAccessLogStore(cfg *config.Config) (*AccessLogStore, error) {
	if cfg == nil {
		return nil, errors.New("nil config")
	}
	return &AccessLogStore{
		logs:   make([]model.AccessLog, 0),
		cfg:    cfg,
		syncCh: make(chan struct{}, 1),
	}, nil
}

func (l *AccessLogStore) Open(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.opened {
		return errors.New("already opened")
	}
	if l.closed {
		return errors.New("already closed")
	}
	l.opened = true
	go l.syncLoop(ctx)
	return nil
}

func (l *AccessLogStore) syncLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-l.syncCh:
			l.flushLocked()
		}
	}
}

func (l *AccessLogStore) Write(entry model.AccessLog) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.opened {
		return errors.New("not opened")
	}
	if l.closed {
		return errors.New("already closed")
	}
	l.logs = append(l.logs, entry)
	select {
	case l.syncCh <- struct{}{}:
	default:
	}
	return nil
}

func (l *AccessLogStore) flushLocked() {
	if !l.cfg.Storage.FlushOnWriteVal() {
		return
	}
}

func (l *AccessLogStore) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return errors.New("already closed")
	}
	l.closed = true
	l.opened = false
	return nil
}

func (l *AccessLogStore) Count() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.logs)
}

func (l *AccessLogStore) Snapshot() []model.AccessLog {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]model.AccessLog, len(l.logs))
	copy(out, l.logs)
	return out
}
