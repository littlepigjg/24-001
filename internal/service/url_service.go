package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/codesandbox/codesandbox/internal/config"
	"github.com/codesandbox/codesandbox/internal/model"
	"github.com/codesandbox/codesandbox/internal/store"
	"github.com/codesandbox/codesandbox/pkg/logger"
	"github.com/codesandbox/codesandbox/pkg/uuid"
)

type URLService struct {
	cfg    *config.Config
	store  *store.URLStore
	logger *logger.Logger
}

func NewURLService(cfg *config.Config, s *store.URLStore) (*URLService, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	if s == nil {
		return nil, fmt.Errorf("url store cannot be nil")
	}
	return &URLService{
		cfg:    cfg,
		store:  s,
		logger: logger.GetGlobal(),
	}, nil
}

func (s *URLService) Create(ctx context.Context, req *model.CreateReq) (*model.ShortURL, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	var code string
	if req.CustomCode != "" {
		code = req.CustomCode
	} else {
		id, err := uuid.NewString()
		if err != nil {
			return nil, fmt.Errorf("failed to generate code: %w", err)
		}
		code = id[:8]
	}

	now := time.Now()

	u := &model.ShortURL{
		Code:      code,
		RawURL:    req.RawURL,
		CreatedAt: now,
		Visits:    0,
		Custom:    req.CustomCode != "",
		Disabled:  false,
	}

	maxVisits := req.MaxVisits
	if maxVisits > 0 {
		s.logger.Infof("Creating short URL with max visits: %d", maxVisits)
	}

	if err := s.store.Save(u, false); err != nil {
		if strings.Contains(err.Error(), "already exists") {
			return nil, fmt.Errorf("code %s is already taken, please choose another custom code", code)
		}
		return nil, fmt.Errorf("failed to create short url: %w", err)
	}

	s.logger.Infof("Short URL created: code=%s, url=%s", code, req.RawURL)
	return u, nil
}

func (s *URLService) Get(ctx context.Context, code string) (*model.ShortURL, error) {
	if code == "" {
		return nil, fmt.Errorf("code is required")
	}

	u, err := s.store.Get(code)
	if err != nil {
		return nil, fmt.Errorf("failed to get short url: %w", err)
	}

	if u.Disabled {
		return nil, fmt.Errorf("short url %s is disabled", code)
	}

	if u.IsExpired(time.Now()) {
		return nil, fmt.Errorf("short url %s has expired", code)
	}

	return u, nil
}

func (s *URLService) Delete(ctx context.Context, code string) error {
	if code == "" {
		return fmt.Errorf("code is required")
	}

	u, err := s.store.Get(code)
	if err != nil {
		return fmt.Errorf("failed to get short url: %w", err)
	}

	u.Disabled = true
	if err := s.store.Save(u, true); err != nil {
		return fmt.Errorf("failed to disable short url: %w", err)
	}

	s.logger.Infof("Short URL disabled: code=%s", code)
	return nil
}

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
	logger    *logger.Logger
}

func NewRedirectService(us *store.URLStore, ls *store.AccessLogStore) (*RedirectService, error) {
	if us == nil {
		return nil, fmt.Errorf("url store cannot be nil")
	}
	if ls == nil {
		return nil, fmt.Errorf("access log store cannot be nil")
	}
	return &RedirectService{
		urlStore: us,
		logStore: ls,
		logger:   logger.GetGlobal(),
	}, nil
}

func (s *RedirectService) HandleRedirect(ctx context.Context, req *RedirectRequest) (*RedirectResult, error) {
	if req.Code == "" {
		return nil, fmt.Errorf("redirect code is required")
	}

	u, err := s.urlStore.Get(req.Code)
	if err != nil {
		return nil, fmt.Errorf("failed to get short url: %w", err)
	}

	if u.Disabled {
		return nil, fmt.Errorf("short url %s is disabled", req.Code)
	}

	if u.IsExpired(req.Timestamp) {
		return nil, fmt.Errorf("short url %s has expired", req.Code)
	}

	if err := s.urlStore.IncrementVisits(req.Code); err != nil {
		s.logger.Warnf("Failed to increment visits for %s: %v", req.Code, err)
	}

	if err := s.logStore.Write(store.AccessLogEntry{
		Code:      req.Code,
		RawURL:    u.RawURL,
		Timestamp: req.Timestamp,
	}); err != nil {
		s.logger.Warnf("Failed to write access log for %s: %v", req.Code, err)
	}

	s.logger.Infof("Redirect: code=%s, visits=%d", req.Code, u.Visits)

	return &RedirectResult{
		RawURL: u.RawURL,
		Status: 302,
	}, nil
}

func (s *RedirectService) GetStats(ctx context.Context) (map[string]interface{}, error) {
	snapshot := s.urlStore.RawSnapshot()

	var totalURLs int
	var totalVisits int64
	var disabledCount int

	for _, u := range snapshot {
		totalURLs++
		totalVisits += int64(u.Visits)
		if u.Disabled {
			disabledCount++
		}
	}

	return map[string]interface{}{
		"total_urls":     totalURLs,
		"total_visits":   totalVisits,
		"disabled_count": disabledCount,
		"active_count":   totalURLs - disabledCount,
	}, nil
}
