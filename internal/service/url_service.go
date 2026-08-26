package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
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
		return nil, errors.New("nil config")
	}
	if s == nil {
		return nil, errors.New("nil url store")
	}
	return &URLService{cfg: cfg, store: s}, nil
}

func (svc *URLService) Create(ctx context.Context, req *model.CreateReq) (*model.ShortURL, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New("nil create request")
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}

	code := req.CustomCode
	custom := false
	if code == "" {
		var err error
		code, err = generateCode(6)
		if err != nil {
			return nil, err
		}
	} else {
		custom = true
	}

	u := &model.ShortURL{
		Code:      code,
		RawURL:    req.RawURL,
		CreatedAt: time.Now(),
		Visits:    0,
		Custom:    custom,
		Disabled:  false,
	}
	u.SetMaxVisits(req.MaxVisits)

	if err := svc.store.Save(u, custom); err != nil {
		return nil, err
	}
	return u, nil
}

func generateCode(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b)[:n], nil
}

var _ = time.Now
