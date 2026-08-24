package service

import (
	"context"
	"errors"
	"time"

	"github.com/codesandbox/codesandbox/internal/model"
	"github.com/codesandbox/codesandbox/internal/store"
)

type RedirectRequest struct {
	Code      string
	Timestamp time.Time
}

type RedirectResult struct {
	RawURL string
	Status int
}

type RedirectService struct {
	urlStore *store.URLStore
	logStore *store.AccessLogStore
}

func NewRedirectService(us *store.URLStore, ls *store.AccessLogStore) (*RedirectService, error) {
	if us == nil {
		return nil, errors.New("nil url store")
	}
	if ls == nil {
		return nil, errors.New("nil access log store")
	}
	return &RedirectService{
		urlStore: us,
		logStore: ls,
	}, nil
}

func (svc *RedirectService) HandleRedirect(ctx context.Context, req *RedirectRequest) (*RedirectResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if req == nil {
		return nil, errors.New("nil redirect request")
	}
	if req.Code == "" {
		return nil, errors.New("empty code")
	}

	u, err := svc.fetchURL(req.Code)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, errors.New("url not found")
	}
	if u.Disabled {
		return nil, errors.New("url is disabled")
	}

	// Context already canceled (client timed out / disconnected)? Do no
	// bookkeeping at all: skip the visit increment, pendingWrites append, and
	// access log entry. The URL record itself (created elsewhere) is kept.
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if err := svc.recordVisitIncrement(ctx, u); err != nil {
		return nil, err
	}
	if err := svc.appendAccessLog(ctx, u); err != nil {
		return nil, err
	}

	return svc.buildResult(u, req)
}

func (svc *RedirectService) fetchURL(code string) (*model.ShortURL, error) {
	return svc.urlStore.Get(code)
}

func (svc *RedirectService) recordVisitIncrement(ctx context.Context, u *model.ShortURL) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	updated := *u
	updated.Visits += 1
	if err := svc.urlStore.Save(&updated, true); err != nil {
		return err
	}
	return nil
}

func (svc *RedirectService) appendAccessLog(ctx context.Context, u *model.ShortURL) error {
	if svc.logStore == nil {
		return nil
	}
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	return svc.logStore.Write(model.AccessLog{
		Code:      u.Code,
		RawURL:    u.RawURL,
		Timestamp: time.Now(),
		IP:        "127.0.0.1",
		UserAgent: "redirect-service",
	})
}

func (svc *RedirectService) buildResult(u *model.ShortURL, req *RedirectRequest) (*RedirectResult, error) {
	if u.IsExpired(req.Timestamp) {
		return &RedirectResult{RawURL: u.RawURL, Status: 410}, nil
	}
	return &RedirectResult{RawURL: u.RawURL, Status: 302}, nil
}

func (svc *RedirectService) finalizeResult(u *model.ShortURL, ts time.Time) (*RedirectResult, error) {
	if u == nil {
		return nil, errors.New("nil url")
	}
	if u.IsExpired(ts) {
		return &RedirectResult{RawURL: u.RawURL, Status: 410}, nil
	}
	return &RedirectResult{RawURL: u.RawURL, Status: 302}, nil
}

func (svc *RedirectService) validateRequest(req *RedirectRequest) error {
	if req == nil {
		return errors.New("nil request")
	}
	if req.Code == "" {
		return errors.New("empty code")
	}
	return nil
}

var _ = time.Now
