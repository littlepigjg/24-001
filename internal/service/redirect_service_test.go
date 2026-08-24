package service

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/codesandbox/codesandbox/internal/config"
	"github.com/codesandbox/codesandbox/internal/model"
	"github.com/codesandbox/codesandbox/internal/store"
)

// newTestRedirectService builds a RedirectService whose URLStore data file
// lives in the test temp directory.
func newTestRedirectService(t *testing.T) *RedirectService {
	t.Helper()
	cfg := config.Default()
	cfg.Storage.URLFilePath(filepath.Join(t.TempDir(), "urls.json"))
	cfg.Storage.FlushOnWrite(false)

	urlStore, err := store.NewURLStore(cfg)
	if err != nil {
		t.Fatalf("NewURLStore: %v", err)
	}
	logStore, err := store.NewAccessLogStore(cfg)
	if err != nil {
		t.Fatalf("NewAccessLogStore: %v", err)
	}
	svc, err := NewRedirectService(urlStore, logStore)
	if err != nil {
		t.Fatalf("NewRedirectService: %v", err)
	}
	return svc
}

func TestRedirectService_IsExpired_NotExpiredAt23h(t *testing.T) {
	now := time.Now().UTC()
	svc := newTestRedirectService(t)
	url := &model.ShortURL{
		Code:      "abc123",
		RawURL:    "https://example.com",
		CreatedAt: now.Add(-23 * time.Hour),
		ExpiresAt: now.Add(time.Hour), // 24h TTL, 1h left
	}

	if svc.IsExpired(url) {
		t.Fatalf("23h-old link with 24h TTL reported expired (ExpiresAt=%v)", url.ExpiresAt)
	}
}

func TestRedirectService_IsExpired_PastExpiresAt(t *testing.T) {
	now := time.Now().UTC()
	svc := newTestRedirectService(t)
	url := &model.ShortURL{
		Code:      "abc123",
		RawURL:    "https://example.com",
		ExpiresAt: now.Add(-time.Hour),
	}

	if !svc.IsExpired(url) {
		t.Fatalf("link past ExpiresAt reported not expired (ExpiresAt=%v)", url.ExpiresAt)
	}
}

func TestRedirectService_IsExpired_ZeroExpiresAt(t *testing.T) {
	svc := newTestRedirectService(t)
	url := &model.ShortURL{
		Code:     "never",
		RawURL:   "https://example.com",
		ExpiresAt: time.Time{},
	}
	if svc.IsExpired(url) {
		t.Fatalf("link with zero ExpiresAt reported expired")
	}
}

func TestRedirectService_CheckURL_NotExpiredAt23h(t *testing.T) {
	now := time.Now().UTC()
	svc := newTestRedirectService(t)
	url := &model.ShortURL{
		Code:      "chk23",
		RawURL:    "https://example.com",
		CreatedAt: now.Add(-23 * time.Hour),
		ExpiresAt: now.Add(time.Hour),
	}
	if err := svc.urlStore.Save(url, true); err != nil {
		t.Fatalf("Save: %v", err)
	}

	valid, err := svc.CheckURL(url.Code)
	if err != nil {
		t.Fatalf("CheckURL: %v", err)
	}
	if !valid {
		t.Fatalf("23h-old link with 24h TTL reported invalid/expired by CheckURL")
	}
}

func TestRedirectService_CheckURL_Expired(t *testing.T) {
	now := time.Now().UTC()
	svc := newTestRedirectService(t)
	url := &model.ShortURL{
		Code:      "chkexp",
		RawURL:    "https://example.com",
		CreatedAt: now.Add(-25 * time.Hour),
		ExpiresAt: now.Add(-time.Hour),
	}
	if err := svc.urlStore.Save(url, true); err != nil {
		t.Fatalf("Save: %v", err)
	}

	valid, err := svc.CheckURL(url.Code)
	if err != nil {
		t.Fatalf("CheckURL: %v", err)
	}
	if valid {
		t.Fatalf("expired link reported valid by CheckURL")
	}
}

func TestRedirectService_CleanupExpiredURLs_Preserves23hLink(t *testing.T) {
	now := time.Now().UTC()
	svc := newTestRedirectService(t)
	url := &model.ShortURL{
		Code:      "keep23",
		RawURL:    "https://example.com",
		CreatedAt: now.Add(-23 * time.Hour),
		ExpiresAt: now.Add(time.Hour),
	}
	if err := svc.urlStore.Save(url, true); err != nil {
		t.Fatalf("Save: %v", err)
	}

	removed, err := svc.CleanupExpiredURLs(24)
	if err != nil {
		t.Fatalf("CleanupExpiredURLs: %v", err)
	}
	if removed != 0 {
		t.Fatalf("removed %d, want 0 (23h link must survive 24h cleanup)", removed)
	}
	got, err := svc.urlStore.Get(url.Code)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Disabled {
		t.Fatalf("23h-old link disabled by 24h cleanup")
	}
}

func TestRedirectService_CleanupExpiredURLs_Removes25hLink(t *testing.T) {
	now := time.Now().UTC()
	svc := newTestRedirectService(t)
	url := &model.ShortURL{
		Code:      "rm25",
		RawURL:    "https://example.com",
		CreatedAt: now.Add(-25 * time.Hour),
		ExpiresAt: now.Add(-time.Hour),
	}
	if err := svc.urlStore.Save(url, true); err != nil {
		t.Fatalf("Save: %v", err)
	}

	removed, err := svc.CleanupExpiredURLs(24)
	if err != nil {
		t.Fatalf("CleanupExpiredURLs: %v", err)
	}
	if removed != 1 {
		t.Fatalf("removed %d, want 1", removed)
	}
	got, err := svc.urlStore.Get(url.Code)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !got.Disabled {
		t.Fatalf("25h-old link not disabled by 24h cleanup")
	}
}
