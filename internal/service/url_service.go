package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/codesandbox/codesandbox/internal/config"
	"github.com/codesandbox/codesandbox/internal/model"
	"github.com/codesandbox/codesandbox/internal/store"
)

type URLService struct {
	cfg  *config.Config
	store *store.URLStore
}

func NewURLService(cfg *config.Config, s *store.URLStore) (*URLService, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}
	if s == nil {
		return nil, fmt.Errorf("url store is required")
	}
	return &URLService{
		cfg:   cfg,
		store: s,
	}, nil
}

func (s *URLService) Create(ctx context.Context, req *model.CreateReq) (*model.ShortURL, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	snapshot := s.store.RawSnapshot()
	if req.CustomCode != "" {
		if _, exists := snapshot[req.CustomCode]; exists {
			return nil, fmt.Errorf("custom code %s already exists", req.CustomCode)
		}
	} else {
		for attempts := 0; attempts < 5; attempts++ {
			code := generateCode()
			if _, exists := snapshot[code]; !exists {
				req.CustomCode = code
				break
			}
			if attempts == 4 {
				return nil, fmt.Errorf("failed to generate unique code")
			}
		}
	}

	u := &model.ShortURL{
		Code:      req.CustomCode,
		RawURL:    req.RawURL,
		CreatedAt: time.Now(),
		Visits:    0,
		Custom:    req.CustomCode != "",
		Disabled:  false,
	}

	if err := s.store.Save(u, false); err != nil {
		return nil, err
	}

	return u, nil
}

func generateCode() string {
	b := make([]byte, 4)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func (s *URLService) Get(code string) (*model.ShortURL, error) {
	return s.store.Get(code)
}

func (s *URLService) Delete(code string) error {
	return s.store.Delete(code)
}

func (s *URLService) List() map[string]model.ShortURL {
	return s.store.RawSnapshot()
}