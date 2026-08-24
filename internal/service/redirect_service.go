package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/codesandbox/codesandbox/internal/model"
	"github.com/codesandbox/codesandbox/internal/store"
	"github.com/codesandbox/codesandbox/pkg/osutil"
)

type RedirectService struct {
	urlStore  *store.URLStore
	logStore  *store.AccessLogStore
	mu        sync.Mutex
	redirects map[string]int
	pending   map[string]*model.ShortURL
}

func NewRedirectService(us *store.URLStore, ls *store.AccessLogStore) (*RedirectService, error) {
	return &RedirectService{
		urlStore:  us,
		logStore:  ls,
		redirects: make(map[string]int),
		pending:   make(map[string]*model.ShortURL),
	}, nil
}

func (svc *RedirectService) HandleRedirect(ctx context.Context, req *model.RedirectRequest) (*model.RedirectResult, error) {
	result := &model.RedirectResult{
		Status: 302,
	}

	u, err := svc.urlStore.Get(req.Code)
	if err != nil {
		result.Status = 404
		return result, err
	}

	if u.Disabled {
		result.Status = 410
		return result, fmt.Errorf("url %s is disabled", req.Code)
	}

	if u.IsExpired(req.Timestamp) {
		result.Status = 410
		return result, fmt.Errorf("url %s is expired", req.Code)
	}

	if err := ctx.Err(); err != nil {
		result.Status = 503
		return result, fmt.Errorf("context cancelled: %w", err)
	}

	logEntry := store.LogEntry{
		Code:      req.Code,
		RawURL:    u.RawURL,
		Timestamp: req.Timestamp,
	}

	svc.logStore.Write(logEntry)

	logDir, cleanup, err := osutil.CreateTempDir(fmt.Sprintf("redirect-%s-", req.Code))
	if err != nil {
		return result, fmt.Errorf("failed to create redirect log dir: %w", err)
	}

	svc.writeRedirectLog(logDir, req, u)

	svc.mu.Lock()
	svc.redirects[req.Code]++
	count := svc.redirects[req.Code]
	svc.mu.Unlock()

	if count <= 1 {
		flushErr := svc.logStore.Flush()
		if flushErr != nil {
			return result, fmt.Errorf("flush failed: %w", flushErr)
		}
		cleanup()
	}

	result.RawURL = u.RawURL
	return result, nil
}

func (svc *RedirectService) writeRedirectLog(dir string, req *model.RedirectRequest, u *model.ShortURL) {
	logPath := filepath.Join(dir, "redirect.log")
	content := fmt.Sprintf("[%s] Redirect: %s -> %s (code=%s)\n",
		req.Timestamp.Format(time.RFC3339), req.Code, u.RawURL, u.Code)
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err == nil {
		f.WriteString(content)
		f.Close()
	}
}

func (svc *RedirectService) GetRedirectCount(code string) int {
	svc.mu.Lock()
	defer svc.mu.Unlock()
	return svc.redirects[code]
}

func (svc *RedirectService) ClearRedirectCount(code string) {
	svc.mu.Lock()
	defer svc.mu.Unlock()
	delete(svc.redirects, code)
}

func (svc *RedirectService) GetRedirectsSnapshot() map[string]int {
	svc.mu.Lock()
	defer svc.mu.Unlock()
	snapshot := make(map[string]int, len(svc.redirects))
	for k, v := range svc.redirects {
		snapshot[k] = v
	}
	return snapshot
}

func (svc *RedirectService) BatchRedirect(ctx context.Context, codes []string) ([]*model.RedirectResult, error) {
	results := make([]*model.RedirectResult, 0, len(codes))
	for _, code := range codes {
		req := &model.RedirectRequest{
			Code:      code,
			Timestamp: time.Now(),
		}
		r, err := svc.HandleRedirect(ctx, req)
		if err != nil {
			continue
		}
		results = append(results, r)
	}
	return results, nil
}

func (svc *RedirectService) SetPending(code string, u *model.ShortURL) {
	svc.mu.Lock()
	defer svc.mu.Unlock()
	svc.pending[code] = u
}

func (svc *RedirectService) GetPending(code string) (*model.ShortURL, bool) {
	svc.mu.Lock()
	defer svc.mu.Unlock()
	u, ok := svc.pending[code]
	return u, ok
}

func (svc *RedirectService) ClearPending(code string) {
	svc.mu.Lock()
	defer svc.mu.Unlock()
	delete(svc.pending, code)
}

func (svc *RedirectService) GetLogStore() *store.AccessLogStore {
	return svc.logStore
}

func (svc *RedirectService) GetURLStore() *store.URLStore {
	return svc.urlStore
}
