package service

import (
	"context"
	"testing"
	"time"

	"github.com/codesandbox/codesandbox/internal/config"
	"github.com/codesandbox/codesandbox/internal/model"
	"github.com/codesandbox/codesandbox/internal/store"
)

// newTestURLService wires up a URLService + RedirectService backed by stores
// using the default config. It exists to exercise the panic paths reported in
// the field: negative page sizes and negative MaxVisits must degrade safely
// instead of triggering "slice bounds out of range".
func newTestURLService(t *testing.T) (*URLService, *RedirectService) {
	t.Helper()
	cfg := config.Default()
	us, err := store.NewURLStore(cfg)
	if err != nil {
		t.Fatalf("NewURLStore: %v", err)
	}
	ls, err := store.NewAccessLogStore(cfg)
	if err != nil {
		t.Fatalf("NewAccessLogStore: %v", err)
	}
	if err := ls.Open(context.Background()); err != nil {
		t.Fatalf("open access log: %v", err)
	}
	urlSvc, err := NewURLService(cfg, us)
	if err != nil {
		t.Fatalf("NewURLService: %v", err)
	}
	redirectSvc, err := NewRedirectService(us, ls)
	if err != nil {
		t.Fatalf("NewRedirectService: %v", err)
	}
	return urlSvc, redirectSvc
}

// TestCreateNegativeMaxVisitsNormalizesToUnlimited ensures a create request
// with MaxVisits=-1 (the documented "unlimited" intent) does not panic and is
// stored as 0 (unlimited) rather than a negative number.
func TestCreateNegativeMaxVisitsNormalizesToUnlimited(t *testing.T) {
	urlSvc, _ := newTestURLService(t)

	shortURL, err := urlSvc.Create(context.Background(), &model.CreateReq{
		RawURL:    "https://example.com",
		MaxVisits: -1,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if shortURL.MaxVisits != 0 {
		t.Fatalf("expected MaxVisits normalized to 0 (unlimited), got %d", shortURL.MaxVisits)
	}
}

// TestListNegativePageSizeDoesNotPanic reproduces the list endpoint crash
// where page_size=-1 reached the slice expression and panicked.
func TestListNegativePageSizeDoesNotPanic(t *testing.T) {
	urlSvc, _ := newTestURLService(t)

	// Seed a few URLs so pagination actually exercises the slice.
	for i := 0; i < 3; i++ {
		if _, err := urlSvc.Create(context.Background(), &model.CreateReq{
			RawURL: "https://example.com/" + string(rune('a'+i)),
		}); err != nil {
			t.Fatalf("Create: %v", err)
		}
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("List with negative pageSize panicked: %v", r)
		}
	}()

	got, err := urlSvc.List(context.Background(), 1, -1)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil result slice for negative pageSize fallback")
	}
}

// TestRedirectOnUnlimitedURLDoesNotPanic reproduces the redirect crash: a
// short URL created with MaxVisits=-1 used to panic on redirect because
// MaxVisits was passed straight into the access-log pager as a page size.
func TestRedirectOnUnlimitedURLDoesNotPanic(t *testing.T) {
	urlSvc, redirectSvc := newTestURLService(t)

	shortURL, err := urlSvc.Create(context.Background(), &model.CreateReq{
		RawURL:    "https://example.com",
		MaxVisits: -1,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("HandleRedirect panicked on unlimited URL: %v", r)
		}
	}()

	res, err := redirectSvc.HandleRedirect(context.Background(), &RedirectRequest{
		Code:      shortURL.Code,
		Timestamp: time.Now(),
	})
	if err != nil {
		t.Fatalf("HandleRedirect: %v", err)
	}
	if res.Status != 302 {
		t.Fatalf("expected 302 redirect, got %d", res.Status)
	}
}

// TestRedirectEnforcesPositiveMaxVisits confirms that a real positive limit
// still trips the 429 path once visits are exhausted (regression guard so the
// "unlimited" fix doesn't accidentally disable limiting for positive values).
func TestRedirectEnforcesPositiveMaxVisits(t *testing.T) {
	urlSvc, redirectSvc := newTestURLService(t)

	shortURL, err := urlSvc.Create(context.Background(), &model.CreateReq{
		RawURL:    "https://example.com",
		MaxVisits: 1,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// First visit succeeds.
	res, err := redirectSvc.HandleRedirect(context.Background(), &RedirectRequest{
		Code:      shortURL.Code,
		Timestamp: time.Now(),
	})
	if err != nil {
		t.Fatalf("HandleRedirect #1: %v", err)
	}
	if res.Status != 302 {
		t.Fatalf("expected first visit to 302, got %d", res.Status)
	}

	// Second visit is rate-limited (visits >= MaxVisits).
	res, err = redirectSvc.HandleRedirect(context.Background(), &RedirectRequest{
		Code:      shortURL.Code,
		Timestamp: time.Now(),
	})
	if err != nil {
		t.Fatalf("HandleRedirect #2: %v", err)
	}
	if res.Status != 429 {
		t.Fatalf("expected second visit to be 429, got %d", res.Status)
	}
}
