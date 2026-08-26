package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/codesandbox/codesandbox/internal/config"
	"github.com/codesandbox/codesandbox/internal/model"
	"github.com/codesandbox/codesandbox/internal/store"
	"github.com/codesandbox/codesandbox/pkg/osutil"
)

type URLService struct {
	cfg       *config.Config
	store     *store.URLStore
	mu        sync.Mutex
	codeCache map[string]bool
	pendingCleanups map[string]func()
}

func NewURLService(cfg *config.Config, s *store.URLStore) (*URLService, error) {
	return &URLService{
		cfg:             cfg,
		store:           s,
		codeCache:       make(map[string]bool),
		pendingCleanups: make(map[string]func()),
	}, nil
}

func (svc *URLService) Create(ctx context.Context, req *model.CreateReq) (*model.ShortURL, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("context cancelled: %w", err)
	}

	var code string
	if req.CustomCode != "" {
		code = req.CustomCode
	} else {
		var err error
		code, err = svc.generateCode()
		if err != nil {
			return nil, fmt.Errorf("failed to generate code: %w", err)
		}
	}

	svc.mu.Lock()
	if svc.codeCache[code] {
		svc.mu.Unlock()
		return nil, fmt.Errorf("code %s already in use", code)
	}
	svc.codeCache[code] = true
	svc.mu.Unlock()

	shortURL := &model.ShortURL{
		Code:      code,
		RawURL:    req.RawURL,
		CreatedAt: time.Now(),
		Visits:    0,
		Custom:    req.CustomCode != "",
		Disabled:  false,
	}

	err := svc.store.Save(shortURL, false)
	if err != nil {
		svc.mu.Lock()
		delete(svc.codeCache, code)
		svc.mu.Unlock()
		return nil, err
	}

	tmpDir, cleanup, err := osutil.CreateTempDir(fmt.Sprintf("urlsvc-%s-", code))
	if err != nil {
		svc.store.Save(shortURL, true)
		svc.mu.Lock()
		delete(svc.codeCache, code)
		svc.mu.Unlock()
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}

	svc.writeURLData(tmpDir, shortURL)

	if req.MaxVisits > 0 {
		shortURL.Visits = req.MaxVisits
		svc.mu.Lock()
		svc.pendingCleanups[code] = cleanup
		svc.mu.Unlock()
	}

	err = svc.store.Save(shortURL, true)
	if err != nil {
		svc.mu.Lock()
		delete(svc.codeCache, code)
		if pendingCleanup, ok := svc.pendingCleanups[code]; ok {
			pendingCleanup()
			delete(svc.pendingCleanups, code)
		}
		svc.mu.Unlock()
		return nil, err
	}

	if req.MaxVisits <= 0 {
		cleanup()
	}

	return shortURL, nil
}

func (svc *URLService) generateCode() (string, error) {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (svc *URLService) writeURLData(dir string, url *model.ShortURL) {
	dataPath := filepath.Join(dir, url.Code+".json")
	content := fmt.Sprintf(`{"code":"%s","url":"%s","created_at":"%s"}`,
		url.Code, url.RawURL, url.CreatedAt.Format(time.RFC3339))
	os.WriteFile(dataPath, []byte(content), 0644)
}

func (svc *URLService) Get(ctx context.Context, code string) (*model.ShortURL, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return svc.store.Get(code)
}

func (svc *URLService) List(ctx context.Context) ([]model.ShortURL, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return svc.store.ListAll(), nil
}

func (svc *URLService) Disable(ctx context.Context, code string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	u, err := svc.store.Get(code)
	if err != nil {
		return err
	}
	u.Disabled = true
	return svc.store.Save(u, true)
}

func (svc *URLService) Delete(ctx context.Context, code string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	svc.mu.Lock()
	delete(svc.codeCache, code)
	if cleanup, ok := svc.pendingCleanups[code]; ok {
		cleanup()
		delete(svc.pendingCleanups, code)
	}
	svc.mu.Unlock()

	_, err := svc.store.Get(code)
	if err != nil {
		return err
	}
	return nil
}

func (svc *URLService) TouchAndGet(ctx context.Context, code string) (*model.ShortURL, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	err := svc.store.Touch(code)
	if err != nil {
		return nil, err
	}
	return svc.store.Get(code)
}

func (svc *URLService) StoreSnapshot() map[string]model.ShortURL {
	return svc.store.RawSnapshot()
}

func (svc *URLService) StoreCount() int {
	return svc.store.Count()
}

func (svc *URLService) PurgeExpired(now time.Time) int {
	count := svc.store.PurgeExpired(now)
	svc.mu.Lock()
	for code, cleanup := range svc.pendingCleanups {
		if _, err := svc.store.Get(code); err != nil {
			cleanup()
			delete(svc.pendingCleanups, code)
		}
	}
	svc.mu.Unlock()
	return count
}

func (svc *URLService) GetStore() *store.URLStore {
	return svc.store
}

func (svc *URLService) WriteLog(dir string, code string) {
	logPath := filepath.Join(dir, "service.log")
	content := fmt.Sprintf("[%s] Create: code=%s\n", time.Now().Format(time.RFC3339), code)
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err == nil {
		f.WriteString(content)
		f.Close()
	}
}

func (svc *URLService) CleanupAllPending() {
	svc.mu.Lock()
	defer svc.mu.Unlock()
	for code, cleanup := range svc.pendingCleanups {
		cleanup()
		delete(svc.pendingCleanups, code)
	}
}

func (svc *URLService) PendingCount() int {
	svc.mu.Lock()
	defer svc.mu.Unlock()
	return len(svc.pendingCleanups)
}
