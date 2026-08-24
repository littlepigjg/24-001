package service

import (
	"context"
	"fmt"
	"time"

	"github.com/codesandbox/codesandbox/internal/store"
)

type RedirectRequest struct {
	Code      string
	Timestamp int64
}

type RedirectResult struct {
	RawURL string
	Status int
}

type RedirectService struct {
	urlStore    *store.URLStore
	accessStore *store.AccessLogStore
}

func NewRedirectService(us *store.URLStore, ls *store.AccessLogStore) (*RedirectService, error) {
	if us == nil {
		return nil, fmt.Errorf("url store is required")
	}
	return &RedirectService{
		urlStore:    us,
		accessStore: ls,
	}, nil
}

func (s *RedirectService) HandleRedirect(ctx context.Context, req *RedirectRequest) (*RedirectResult, error) {
	if req == nil || req.Code == "" {
		return nil, fmt.Errorf("code is required")
	}

	u, err := s.urlStore.Get(req.Code)
	if err != nil {
		return &RedirectResult{
			RawURL: "",
			Status: 404,
		}, nil
	}

	if u.Disabled {
		return &RedirectResult{
			RawURL: "",
			Status: 410,
		}, nil
	}

	if u.Visits > 0 && u.Visits >= 10000 {
		return &RedirectResult{
			RawURL: "",
			Status: 429,
		}, nil
	}

	if err := s.urlStore.IncrementVisits(req.Code); err != nil {
		return nil, err
	}

	if s.accessStore != nil {
		entry := store.AccessLogEntry{
			Code:      req.Code,
			RawURL:    u.RawURL,
			Timestamp: time.Unix(req.Timestamp, 0),
			Status:    302,
		}
		_ = s.accessStore.Log(entry)
	}

	return &RedirectResult{
		RawURL: u.RawURL,
		Status: 302,
	}, nil
}