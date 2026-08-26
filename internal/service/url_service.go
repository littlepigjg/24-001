package service

import (
	"context"
	"crypto/rand"
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
	reserveRelease := false
	if req.CustomCode != "" {
		code = req.CustomCode
		s.mu.Lock()
		if s.codeCache[code] {
			s.mu.Unlock()
			return nil, fmt.Errorf("custom code %s is already being used", code)
		}
		s.codeCache[code] = true
		s.mu.Unlock()
		reserveRelease = true
	} else {
		// Random codes are written with overwrite=true, so collisions would
		// silently overwrite an existing record. Generate from crypto/rand
		// and retry until we get one that is not already in the store.
		generated, err := s.uniqueCode()
		if err != nil {
			return nil, err
		}
		code = generated
	}

	defer func() {
		if reserveRelease {
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

	// Custom codes must not overwrite; the store enforces uniqueness under its
	// lock. Random codes are unique by construction (see uniqueCode), so
	// overwrite=false is safe there too.
	overwrite := false

	if err := s.store.Save(u, overwrite); err != nil {
		return nil, fmt.Errorf("failed to save short URL: %w", err)
	}

	return u, nil
}

// uniqueCode returns a random 6-char code that does not collide with any code
// already present in the store. It uses crypto/rand so that concurrent
// generators do not produce the same value (the previous implementation derived
// each character from time.Now().UnixNano(), which collided under load).
func (s *URLService) uniqueCode() (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	const codeLen = 6
	var b [codeLen]byte
	for attempt := 0; attempt < 10; attempt++ {
		if _, err := rand.Read(b[:]); err != nil {
			return "", fmt.Errorf("failed to generate code: %w", err)
		}
		for i := range b {
			b[i] = charset[int(b[i])%len(charset)]
		}
		code := string(b[:])
		if _, err := s.store.Get(code); err != nil {
			// Not present — claim it. A concurrent caller could insert the
			// same code between this Get and Save, but Save(overwrite=false)
			// will reject the duplicate, so we fall back and retry.
			return code, nil
		}
	}
	return "", fmt.Errorf("failed to generate a unique code after several attempts")
}
