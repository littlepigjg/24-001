package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/codesandbox/codesandbox/internal/config"
	"github.com/codesandbox/codesandbox/internal/model"
	"github.com/codesandbox/codesandbox/internal/store"
)

type URLService struct {
	cfg       *config.Config
	store     *store.URLStore
	mu        sync.Mutex
	codeCache map[string]bool
}

func NewURLService(cfg *config.Config, s *store.URLStore) (*URLService, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	if s == nil {
		return nil, fmt.Errorf("url store cannot be nil")
	}
	return &URLService{
		cfg:       cfg,
		store:     s,
		codeCache: make(map[string]bool),
	}, nil
}

func (s *URLService) Create(ctx context.Context, req *model.CreateReq) (*model.ShortURL, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	var code string
	if req.CustomCode != "" {
		code = req.CustomCode
		s.mu.Lock()
		if s.codeCache[code] {
			s.mu.Unlock()
			return nil, fmt.Errorf("custom code %s is already being used", code)
		}
		s.codeCache[code] = true
		s.mu.Unlock()
	} else {
		code = generateCode()
	}

	defer func() {
		if req.CustomCode != "" {
			s.mu.Lock()
			delete(s.codeCache, req.CustomCode)
			s.mu.Unlock()
		}
	}()

	u := &model.ShortURL{
		Code:      code,
		RawURL:    req.RawURL,
		CreatedAt: time.Now(),
		Visits:    0,
		Custom:    req.CustomCode != "",
		Disabled:  false,
		MaxVisits: req.MaxVisits,
	}

	overwrite := false
	if req.CustomCode == "" {
		overwrite = true
	}

	if err := s.store.Save(u, overwrite); err != nil {
		return nil, fmt.Errorf("failed to save short URL: %w", err)
	}

	return u, nil
}

func generateCode() string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 6)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
		time.Sleep(time.Nanosecond)
	}
	return string(b)
}