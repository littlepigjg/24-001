package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/codesandbox/codesandbox/config"
	"github.com/codesandbox/codesandbox/internal/model"
	"github.com/codesandbox/codesandbox/internal/store"
)

type URLService struct {
	cfg   *config.Config
	store *store.URLStore
}

func NewURLService(cfg *config.Config, s *store.URLStore) (*URLService, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	if s == nil {
		return nil, fmt.Errorf("URL store cannot be nil")
	}

	return &URLService{
		cfg:   cfg,
		store: s,
	}, nil
}

func (s *URLService) Create(ctx context.Context, req *model.CreateReq) (*model.ShortURL, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	var code string
	isCustom := false

	if req.CustomCode != "" {
		code = req.CustomCode
		isCustom = true
	} else {
		var err error
		code, err = generateCode()
		if err != nil {
			return nil, fmt.Errorf("failed to generate code: %w", err)
		}
	}

	shortURL := &model.ShortURL{
		Code:      code,
		RawURL:    req.RawURL,
		CreatedAt: time.Now(),
		Visits:    0,
		Custom:    isCustom,
		Disabled:  false,
	}

	if err := shortURL.Validate(); err != nil {
		return nil, fmt.Errorf("invalid short URL: %w", err)
	}

	if err := s.store.Save(shortURL, false); err != nil {
		return nil, fmt.Errorf("failed to save short URL: %w", err)
	}

	return shortURL, nil
}

func generateCode() (string, error) {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b)[:8], nil
}
