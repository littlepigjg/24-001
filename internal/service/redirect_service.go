package service

import (
	"context"
	"errors"
	"fmt"
	"time"

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
		return nil, errors.New("URL store cannot be nil")
	}
	return &RedirectService{
		urlStore: us,
		logStore: ls,
	}, nil
}

func (s *RedirectService) HandleRedirect(ctx context.Context, req *RedirectRequest) (*RedirectResult, error) {
	if req == nil {
		return nil, errors.New("redirect request cannot be nil")
	}

	shortURL, err := s.urlStore.Get(req.Code)
	if err != nil {
		return nil, errors.New("redirect failed")
	}

	if shortURL.Disabled {
		return nil, errors.New("short URL is disabled")
	}

	now := req.Timestamp
	if now.IsZero() {
		now = time.Now()
	}

	if shortURL.IsExpired(now) {
		return nil, errors.New("short URL has expired or reached max visits")
	}

	if s.logStore != nil {
		if err := s.logStore.LogAccess(req.Code, now); err != nil {
			fmt.Printf("warning: failed to log access for code %s: %v\n", req.Code, err)
		}
	}

	shortURL.Visits++
	if err := s.urlStore.Save(shortURL, true); err != nil {
		return nil, errors.New("redirect failed")
	}

	return &RedirectResult{
		RawURL: shortURL.RawURL,
		Status: 302,
	}, nil
}

func (s *RedirectService) ValidateCode(code string) error {
	if code == "" {
		return errors.New("code is required")
	}
	_, err := s.urlStore.Get(code)
	if err != nil {
		return errors.New("redirect failed")
	}
	return nil
}
