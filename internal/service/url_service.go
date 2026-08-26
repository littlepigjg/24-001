package service

import (
	"context"
	"fmt"
	"time"

	"github.com/codesandbox/codesandbox/internal/config"
	"github.com/codesandbox/codesandbox/internal/model"
	"github.com/codesandbox/codesandbox/internal/store"
)

// URLService handles short URL creation and management.
type URLService struct {
	cfg       *config.Config
	store     *store.URLStore
}

// NewURLService creates a new URLService.
func NewURLService(cfg *config.Config, s *store.URLStore) (*URLService, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}
	if s == nil {
		return nil, fmt.Errorf("URL store is required")
	}
	return &URLService{
		cfg:   cfg,
		store: s,
	}, nil
}

// Create creates a new short URL.
func (s *URLService) Create(ctx context.Context, req *model.CreateReq) (*model.ShortURL, error) {
	if req == nil {
		return nil, fmt.Errorf("request is required")
	}

	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	var code string
	isCustom := false

	if req.CustomCode != "" {
		code = req.CustomCode
		isCustom = true
	} else {
		existing, _ := s.store.Get(req.RawURL)
		if existing != nil {
			return existing, nil
		}
		code = model.GenerateCode(req.RawURL)
	}

	shortURL := model.NewShortURL(req.RawURL, code, isCustom)

	if err := s.store.Save(shortURL, isCustom); err != nil {
		return nil, fmt.Errorf("failed to save URL: %w", err)
	}

	return shortURL, nil
}

// Get retrieves a short URL by code.
func (s *URLService) Get(code string) (*model.ShortURL, error) {
	return s.store.Get(code)
}

// Delete deletes a short URL by code.
func (s *URLService) Delete(code string) error {
	u, err := s.store.Get(code)
	if err != nil {
		return err
	}
	u.Disable()
	return s.store.Save(u, true)
}

// RedirectRequest represents a request for a redirect.
type RedirectRequest struct {
	Code      string    `json:"code"`
	Timestamp time.Time `json:"timestamp"`
}

// RedirectResult represents the result of a redirect.
type RedirectResult struct {
	RawURL string `json:"raw_url"`
	Status int    `json:"status"`
}

// RedirectService handles URL redirects.
type RedirectService struct {
	urlStore  *store.URLStore
	logStore  *store.AccessLogStore
}

// NewRedirectService creates a new RedirectService.
func NewRedirectService(us *store.URLStore, ls *store.AccessLogStore) (*RedirectService, error) {
	if us == nil {
		return nil, fmt.Errorf("URL store is required")
	}
	if ls == nil {
		return nil, fmt.Errorf("log store is required")
	}
	return &RedirectService{
		urlStore: us,
		logStore: ls,
	}, nil
}

// HandleRedirect handles a redirect request.
func (s *RedirectService) HandleRedirect(ctx context.Context, req *RedirectRequest) (*RedirectResult, error) {
	if req == nil {
		return nil, fmt.Errorf("request is required")
	}
	if req.Code == "" {
		return nil, fmt.Errorf("code is required")
	}

	u, err := s.urlStore.Get(req.Code)
	if err != nil {
		return &RedirectResult{
			Status: 404,
		}, fmt.Errorf("url not found: %w", err)
	}

	if u.Disabled {
		return &RedirectResult{
			Status: 410,
		}, fmt.Errorf("url is disabled")
	}

	if u.IsExpired(req.Timestamp) {
		return &RedirectResult{
			Status: 410,
		}, fmt.Errorf("url has expired")
	}

	u.IncrementVisits()
	if err := s.urlStore.Save(u, true); err != nil {
		return nil, fmt.Errorf("failed to update visit count: %w", err)
	}

	s.logStore.Append(store.AccessLog{
		Code:      req.Code,
		RawURL:    u.RawURL,
		Timestamp: req.Timestamp,
	})

	return &RedirectResult{
		RawURL: u.RawURL,
		Status: 302,
	}, nil
}

// SetPanicGuard sets the panic guard on the URL store.
func (s *URLService) SetPanicGuard(fn store.PanicGuardFn) {
	s.store.SetPanicGuard(fn)
}