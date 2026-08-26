package service

import (
	"context"
	"crypto/md5"
	"fmt"
	"time"

	"github.com/codesandbox/codesandbox/config"
	"github.com/codesandbox/codesandbox/model"
	"github.com/codesandbox/codesandbox/pkg/process"
	"github.com/codesandbox/codesandbox/store"
)

type URLService struct {
	cfg      *config.Config
	store    *store.URLStore
	executor *process.Executor
}

func NewURLService(cfg *config.Config, s *store.URLStore) (*URLService, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}
	if s == nil {
		return nil, fmt.Errorf("URL store is required")
	}
	return &URLService{
		cfg:      cfg,
		store:    s,
		executor: process.NewExecutor(),
	}, nil
}

func (s *URLService) Create(ctx context.Context, req *model.CreateReq) (*model.ShortURL, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	code := req.CustomCode
	if code == "" {
		code = generateCode(req.RawURL)
	}

	shortURL := &model.ShortURL{
		Code:      code,
		RawURL:    req.RawURL,
		CreatedAt: time.Now(),
		Visits:    0,
		MaxVisits: req.MaxVisits,
		Custom:    req.CustomCode != "",
		Disabled:  false,
	}

	if err := shortURL.Validate(); err != nil {
		return nil, err
	}

	if err := s.store.Save(shortURL, false); err != nil {
		return nil, err
	}

	return shortURL, nil
}

func (s *URLService) Get(ctx context.Context, code string) (*model.ShortURL, error) {
	return s.store.Get(code)
}

func generateCode(rawURL string) string {
	hash := md5.Sum([]byte(rawURL + time.Now().String()))
	return fmt.Sprintf("%x", hash[:4])
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

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
		return nil, fmt.Errorf("URL store is required")
	}
	return &RedirectService{
		urlStore: us,
		logStore: ls,
	}, nil
}

func (s *RedirectService) HandleRedirect(ctx context.Context, req *RedirectRequest) (*RedirectResult, error) {
	if req.Code == "" {
		return nil, fmt.Errorf("code is required")
	}

	shortURL, err := s.urlStore.Get(req.Code)
	if err != nil {
		return &RedirectResult{
			RawURL: "",
			Status: 404,
		}, nil
	}

	if shortURL.Disabled {
		return &RedirectResult{
			RawURL: "",
			Status: 410,
		}, nil
	}

	if shortURL.MaxVisits > 0 && shortURL.Visits >= shortURL.MaxVisits {
		return &RedirectResult{
			RawURL: "",
			Status: 410,
		}, nil
	}

	shortURL.IncrVisits()
	_ = s.urlStore.IncrementVisitsWithGuard(req.Code)

	if s.logStore != nil {
		_ = s.logStore.Append(store.AccessLogEntry{
			Code:      req.Code,
			Timestamp: req.Timestamp,
			IP:        "127.0.0.1",
		})
	}

	return &RedirectResult{
		RawURL: shortURL.RawURL,
		Status: 302,
	}, nil
}
