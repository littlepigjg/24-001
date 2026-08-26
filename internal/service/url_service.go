package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/codesandbox/codesandbox/internal/config"
	"github.com/codesandbox/codesandbox/internal/model"
	"github.com/codesandbox/codesandbox/internal/store"
	"github.com/codesandbox/codesandbox/pkg/logger"
)

type RedirectRequest struct {
	Code      string    `json:"code"`
	Timestamp time.Time `json:"timestamp"`
}

type RedirectResult struct {
	RawURL string `json:"raw_url"`
	Status int    `json:"status"`
}

type URLService struct {
	cfg    *config.Config
	store  *store.URLStore
	logger *logger.Logger
}

func NewURLService(cfg *config.Config, s *store.URLStore) (*URLService, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}
	if s == nil {
		return nil, fmt.Errorf("url store is nil")
	}
	return &URLService{
		cfg:    cfg,
		store:  s,
		logger: logger.GetGlobal(),
	}, nil
}

func (svc *URLService) Create(ctx context.Context, req *model.CreateReq) (*model.ShortURL, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	code := req.CustomCode
	if code == "" {
		var err error
		code, err = svc.generateCode()
		if err != nil {
			return nil, fmt.Errorf("failed to generate code: %w", err)
		}
	}

	shortURL := &model.ShortURL{
		Code:      code,
		RawURL:    req.RawURL,
		CreatedAt: time.Now(),
		Visits:    0,
		Custom:    req.CustomCode != "",
		Disabled:  false,
	}

	if err := svc.store.Save(shortURL, false); err != nil {
		return nil, fmt.Errorf("failed to save short url: %w", err)
	}

	svc.logger.Infof("Short URL created: %s -> %s", code, req.RawURL)
	return shortURL, nil
}

func (svc *URLService) Get(code string) (*model.ShortURL, error) {
	return svc.store.Get(code)
}

func (svc *URLService) Update(code string, rawURL string) (*model.ShortURL, error) {
	url, err := svc.store.Get(code)
	if err != nil {
		return nil, err
	}

	url.RawURL = rawURL
	if err := svc.store.Save(url, true); err != nil {
		return nil, fmt.Errorf("failed to update short url: %w", err)
	}

	svc.logger.Infof("Short URL updated: %s", code)
	return url, nil
}

func (svc *URLService) Delete(code string) error {
	url, err := svc.store.Get(code)
	if err != nil {
		return err
	}

	url.Disabled = true
	if err := svc.store.Save(url, true); err != nil {
		return fmt.Errorf("failed to disable short url: %w", err)
	}

	svc.logger.Infof("Short URL disabled: %s", code)
	return nil
}

func (svc *URLService) List() ([]model.ShortURL, error) {
	snapshot := svc.store.RawSnapshot()
	var urls []model.ShortURL
	for _, u := range snapshot {
		urls = append(urls, u)
	}
	return urls, nil
}

func (svc *URLService) generateCode() (string, error) {
	bytes := make([]byte, 4)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

type RedirectService struct {
	urlStore  *store.URLStore
	logStore  *store.AccessLogStore
	logger    *logger.Logger
}

func NewRedirectService(us *store.URLStore, ls *store.AccessLogStore) (*RedirectService, error) {
	if us == nil {
		return nil, fmt.Errorf("url store is nil")
	}
	if ls == nil {
		return nil, fmt.Errorf("access log store is nil")
	}
	return &RedirectService{
		urlStore: us,
		logStore: ls,
		logger:   logger.GetGlobal(),
	}, nil
}

func (rs *RedirectService) HandleRedirect(ctx context.Context, req *RedirectRequest) (*RedirectResult, error) {
	url, err := rs.urlStore.Get(req.Code)
	if err != nil {
		rs.logger.Warnf("Redirect not found for code: %s", req.Code)
		return &RedirectResult{
			RawURL: "",
			Status: 404,
		}, nil
	}

	if url.Disabled {
		rs.logger.Warnf("Redirect disabled for code: %s", req.Code)
		return &RedirectResult{
			RawURL: "",
			Status: 410,
		}, nil
	}

	if err := rs.logStore.Append(store.AccessLogEntry{
		Code:       req.Code,
		RawURL:     url.RawURL,
		AccessedAt: req.Timestamp,
		IP:         "",
		Referer:    "",
		Status:     302,
	}); err != nil {
		rs.logger.Warnf("Failed to log access for %s: %v", req.Code, err)
	}

	rs.logger.Infof("Redirect: %s -> %s", req.Code, url.RawURL)
	return &RedirectResult{
		RawURL: url.RawURL,
		Status: 302,
	}, nil
}

func validateURLCode(code string) error {
	if code == "" {
		return fmt.Errorf("code is empty")
	}
	if len(code) < 3 || len(code) > 16 {
		return fmt.Errorf("code must be between 3 and 16 characters")
	}
	for _, r := range code {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '-' && r != '_' {
			return fmt.Errorf("code contains invalid character: %c", r)
		}
	}
	lower := strings.ToLower(code)
	for _, dangerous := range []string{"..", "/", "\\", "\x00", "."} {
		if strings.Contains(lower, dangerous) {
			return fmt.Errorf("code contains forbidden sequence: %s", dangerous)
		}
	}
	return nil
}
