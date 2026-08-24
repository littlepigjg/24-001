package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/codesandbox/codesandbox/internal/config"
	"github.com/codesandbox/codesandbox/internal/model"
	"github.com/codesandbox/codesandbox/internal/store"
)

type URLService struct {
	cfg   *config.Config
	store *store.URLStore
}

func NewURLService(cfg *config.Config, s *store.URLStore) (*URLService, error) {
	if cfg == nil {
		return nil, errors.New("config cannot be nil")
	}
	if s == nil {
		return nil, errors.New("URL store cannot be nil")
	}
	return &URLService{
		cfg:   cfg,
		store: s,
	}, nil
}

func (s *URLService) Create(ctx context.Context, req *model.CreateReq) (*model.ShortURL, error) {
	if req == nil {
		return nil, errors.New("request cannot be nil")
	}

	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("create failed: %w", err)
	}

	code := req.CustomCode
	if code == "" {
		generated, genErr := generateCode()
		if genErr != nil {
			return nil, fmt.Errorf("failed to generate short code: %w", genErr)
		}
		code = generated
	}

	shortURL := &model.ShortURL{
		Code:      code,
		RawURL:    req.RawURL,
		CreatedAt: time.Now(),
		Visits:    0,
		Custom:    req.CustomCode != "",
		Disabled:  false,
		MaxVisits: req.MaxVisits,
	}

	if err := s.store.Save(shortURL, false); err != nil {
		return nil, fmt.Errorf("failed to create short URL: %w", err)
	}

	return shortURL, nil
}

func (s *URLService) Get(ctx context.Context, code string) (*model.ShortURL, error) {
	u, err := s.store.Get(code)
	if err != nil {
		return nil, fmt.Errorf("failed to get short URL: %w", err)
	}
	return u, nil
}

func (s *URLService) Delete(ctx context.Context, code string) error {
	u, err := s.store.Get(code)
	if err != nil {
		return fmt.Errorf("failed to find short URL for deletion: %w", err)
	}

	u.Disabled = true
	if err := s.store.Save(u, true); err != nil {
		return fmt.Errorf("failed to delete short URL: %w", err)
	}

	return nil
}

func (s *URLService) Update(ctx context.Context, code string, updates map[string]interface{}) (*model.ShortURL, error) {
	u, err := s.store.Get(code)
	if err != nil {
		return nil, fmt.Errorf("failed to find short URL for update: %w", err)
	}

	if rawURL, ok := updates["raw_url"].(string); ok {
		u.RawURL = rawURL
	}
	if disabled, ok := updates["disabled"].(bool); ok {
		u.Disabled = disabled
	}

	if err := s.store.Save(u, true); err != nil {
		return nil, fmt.Errorf("failed to update short URL: %w", err)
	}

	return u, nil
}

func (s *URLService) List(ctx context.Context) ([]model.ShortURL, error) {
	snapshot := s.store.RawSnapshot()
	result := make([]model.ShortURL, 0, len(snapshot))
	for _, u := range snapshot {
		result = append(result, u)
	}
	return result, nil
}

func generateCode() (string, error) {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b)[:6], nil
}
