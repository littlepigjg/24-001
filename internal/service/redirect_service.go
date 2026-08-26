package service

import (
	"context"
	"fmt"
	"time"

	"github.com/codesandbox/codesandbox/internal/store"
)

type RedirectRequest struct {
	Code      string    `json:"code"`
	Timestamp time.Time `json:"timestamp"`
}

type RedirectResult struct {
	RawURL string `json:"raw_url"`
	Status int    `json:"status"`
}

type RedirectService struct {
	urlStore *store.URLStore
	logStore *store.AccessLogStore
}

func NewRedirectService(us *store.URLStore, ls *store.AccessLogStore) (*RedirectService, error) {
	if us == nil {
		return nil, fmt.Errorf("url store cannot be nil")
	}
	if ls == nil {
		return nil, fmt.Errorf("log store cannot be nil")
	}
	return &RedirectService{
		urlStore: us,
		logStore: ls,
	}, nil
}

func (s *RedirectService) HandleRedirect(ctx context.Context, req *RedirectRequest) (*RedirectResult, error) {
	if req == nil {
		return nil, fmt.Errorf("redirect request cannot be nil")
	}
	if req.Code == "" {
		return nil, fmt.Errorf("redirect code is required")
	}

	u, err := s.urlStore.Get(req.Code)
	if err != nil {
		return nil, fmt.Errorf("failed to get short URL: %w", err)
	}

	if u.IsExpired(req.Timestamp) {
		return &RedirectResult{
			RawURL: "",
			Status: 410,
		}, nil
	}

	u.Visits++
	if err := s.urlStore.Save(u, true); err != nil {
		return nil, fmt.Errorf("failed to update visit count: %w", err)
	}

	if s.logStore != nil {
		s.logStore.WriteLog(fmt.Sprintf("redirect code=%s url=%s ip=internal", req.Code, u.RawURL))
	}

	return &RedirectResult{
		RawURL: u.RawURL,
		Status: 302,
	}, nil
}