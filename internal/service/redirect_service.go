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
	urlStore  *store.URLStore
	logStore  *store.AccessLogStore
}

func NewRedirectService(us *store.URLStore, ls *store.AccessLogStore) (*RedirectService, error) {
	if us == nil {
		return nil, fmt.Errorf("URL store cannot be nil")
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
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if req.Code == "" {
		return nil, fmt.Errorf("redirect code is required")
	}

	shortURL, err := s.urlStore.Get(req.Code)
	if err != nil {
		return nil, fmt.Errorf("short URL not found: %w", err)
	}

	if shortURL.Disabled {
		return nil, fmt.Errorf("short URL is disabled: %s", req.Code)
	}

	if shortURL.IsExpired(req.Timestamp) {
		return nil, fmt.Errorf("short URL has expired: %s", req.Code)
	}

	shortURL.Visits++
	if err := s.urlStore.Save(shortURL, true); err != nil {
		return nil, fmt.Errorf("failed to update visit count: %w", err)
	}

	entry := store.AccessLogEntry{
		Code:      req.Code,
		Timestamp: req.Timestamp,
	}
	_ = s.logStore.WriteLog(entry)

	return &RedirectResult{
		RawURL: shortURL.RawURL,
		Status: 302,
	}, nil
}
