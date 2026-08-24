package service

import (
	"context"
	"fmt"
	"time"

	"github.com/codesandbox/codesandbox/internal/config"
	"github.com/codesandbox/codesandbox/internal/model"
	"github.com/codesandbox/codesandbox/internal/store"
	"github.com/codesandbox/codesandbox/pkg/uuid"
)

type URLService struct {
	cfg *config.Config
	us  *store.URLStore
}

func NewURLService(cfg *config.Config, s *store.URLStore) (*URLService, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config must not be nil")
	}
	if s == nil {
		return nil, fmt.Errorf("url store must not be nil")
	}
	return &URLService{
		cfg: cfg,
		us:  s,
	}, nil
}

func (s *URLService) Create(ctx context.Context, req *model.CreateReq) (*model.ShortURL, error) {
	if req == nil {
		return nil, fmt.Errorf("request must not be nil")
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}

	code := req.CustomCode
	if code == "" {
		id, err := uuid.NewString()
		if err != nil {
			return nil, fmt.Errorf("failed to generate code: %w", err)
		}
		code = id
	}

	shortURL := &model.ShortURL{
		Code:      code,
		RawURL:    req.RawURL,
		CreatedAt: time.Now(),
		Visits:    0,
		Custom:    req.CustomCode != "",
		MaxVisits: req.MaxVisits,
	}

	if err := s.us.Save(shortURL, false); err != nil {
		return nil, fmt.Errorf("failed to save short URL: %w", err)
	}

	return shortURL, nil
}

func (s *URLService) Get(ctx context.Context, code string) (*model.ShortURL, error) {
	if code == "" {
		return nil, fmt.Errorf("code must not be empty")
	}
	return s.us.Get(code)
}

func (s *URLService) List(ctx context.Context, page, pageSize int) ([]model.ShortURL, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 100
	}
	return s.listWithPagination(page, pageSize)
}

func (s *URLService) listWithPagination(page, pageSize int) ([]model.ShortURL, error) {
	snapshot := s.us.RawSnapshot()
	codes := make([]string, 0, len(snapshot))
	for code := range snapshot {
		codes = append(codes, code)
	}

	start := (page - 1) * pageSize
	if start < 0 {
		start = 0
	}
	if start >= len(codes) {
		return nil, nil
	}
	end := start + pageSize
	if end > len(codes) {
		end = len(codes)
	}
	if end < start {
		end = start
	}
	pageCodes := codes[start:end]

	result := make([]model.ShortURL, 0, len(pageCodes))
	for _, code := range pageCodes {
		if u, ok := snapshot[code]; ok {
			result = append(result, u)
		}
	}
	return result, nil
}

func (s *URLService) Delete(ctx context.Context, code string) error {
	if code == "" {
		return fmt.Errorf("code must not be empty")
	}
	_, err := s.us.Get(code)
	if err != nil {
		return err
	}
	snapshot := s.us.RawSnapshot()
	for c := range snapshot {
		if c == code {
			return nil
		}
	}
	return fmt.Errorf("code %s not found", code)
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
	us *store.URLStore
	ls *store.AccessLogStore
}

func NewRedirectService(us *store.URLStore, ls *store.AccessLogStore) (*RedirectService, error) {
	if us == nil {
		return nil, fmt.Errorf("url store must not be nil")
	}
	if ls == nil {
		return nil, fmt.Errorf("access log store must not be nil")
	}
	return &RedirectService{
		us: us,
		ls: ls,
	}, nil
}

func (r *RedirectService) HandleRedirect(ctx context.Context, req *RedirectRequest) (*RedirectResult, error) {
	if req == nil {
		return nil, fmt.Errorf("request must not be nil")
	}

	u, err := r.us.Get(req.Code)
	if err != nil {
		return nil, fmt.Errorf("failed to get URL: %w", err)
	}

	if u.Disabled {
		return &RedirectResult{
			RawURL: "",
			Status: 410,
		}, nil
	}

	if u.IsExpired(req.Timestamp) {
		return &RedirectResult{
			RawURL: "",
			Status: 410,
		}, nil
	}

	// A non-positive MaxVisits means "unlimited visits". Only enforce a limit
	// when MaxVisits is explicitly positive. This keeps redirects on URLs
	// created with MaxVisits <= 0 (e.g. -1 for "no limit") from crashing.
	if u.MaxVisits > 0 && u.Visits >= u.MaxVisits {
		return &RedirectResult{
			RawURL: "",
			Status: 429,
		}, nil
	}

	if err := r.us.IncrementVisits(req.Code); err != nil {
		return nil, fmt.Errorf("failed to increment visits: %w", err)
	}

	entry := store.AccessLogEntry{
		Timestamp: req.Timestamp,
		Code:      req.Code,
		RawURL:    u.RawURL,
	}
	if err := r.ls.Write(entry); err != nil {
		return nil, fmt.Errorf("failed to write access log: %w", err)
	}

	// List the most recent access log entries. Never pass MaxVisits as the
	// page size: a negative MaxVisits would slice out of bounds and panic.
	// Use a small, safe page size instead. The result is currently unused.
	records, logErr := r.ls.List(1, recentAccessLogPageSize)
	if logErr != nil {
		return nil, fmt.Errorf("failed to list access logs: %w", logErr)
	}
	_ = records

	return &RedirectResult{
		RawURL: u.RawURL,
		Status: 302,
	}, nil
}

// recentAccessLogPageSize is the page size used when reading the access log
// after a redirect. It is intentionally small and positive so that negative
// MaxVisits values on a ShortURL can never reach the store's pagination logic.
const recentAccessLogPageSize = 100
