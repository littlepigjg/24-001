package service

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/codesandbox/codesandbox/internal/config"
	"github.com/codesandbox/codesandbox/internal/model"
	"github.com/codesandbox/codesandbox/internal/store"
)

// newTestURLService builds a URLService backed by a URLStore whose data file
// lives in the test's temp directory, so flush-on-write never touches the
// real repo's ./data/urls.json.
func newTestURLService(t *testing.T) *URLService {
	t.Helper()
	cfg := config.Default()
	cfg.Storage.URLFilePath(filepath.Join(t.TempDir(), "urls.json"))
	cfg.Storage.FlushOnWrite(false)

	urlStore, err := store.NewURLStore(cfg)
	if err != nil {
		t.Fatalf("NewURLStore: %v", err)
	}
	svc, err := NewURLService(cfg, urlStore)
	if err != nil {
		t.Fatalf("NewURLService: %v", err)
	}
	return svc
}

// saveURL inserts (or overwrites) a ShortURL entry in the store.
func saveURL(t *testing.T, svc *URLService, u *model.ShortURL) {
	t.Helper()
	if err := svc.store.Save(u, true); err != nil {
		t.Fatalf("Save(%s): %v", u.Code, err)
	}
}

func TestValidateExpiry_NotExpiredAt23h(t *testing.T) {
	// Regression: a 23h-old link with a 24h TTL (ExpiresAt 1h in the future)
	// must NOT be reported expired. The old +8h implementation reported it
	// expired on a UTC+8 host.
	now := time.Now().UTC()
	svc := newTestURLService(t)
	url := &model.ShortURL{
		Code:      "abc123",
		RawURL:     "https://example.com",
		CreatedAt:  now.Add(-23 * time.Hour),
		ExpiresAt:  now.Add(time.Hour), // 24h TTL from creation, still 1h left
	}
	saveURL(t, svc, url)

	if svc.ValidateExpiry(url) {
		t.Fatalf("23h-old link with 24h TTL reported expired (now=%v, ExpiresAt=%v)", now, url.ExpiresAt)
	}
}

func TestValidateExpiry_ExpiredPastExpiresAt(t *testing.T) {
	now := time.Now().UTC()
	svc := newTestURLService(t)
	url := &model.ShortURL{
		Code:     "abc123",
		RawURL:   "https://example.com",
		ExpiresAt: now.Add(-time.Hour), // expired 1h ago
	}
	saveURL(t, svc, url)

	if !svc.ValidateExpiry(url) {
		t.Fatalf("link past ExpiresAt reported not expired (ExpiresAt=%v)", url.ExpiresAt)
	}
}

func TestValidateExpiry_ZeroExpiresAtNeverExpires(t *testing.T) {
	svc := newTestURLService(t)
	url := &model.ShortURL{
		Code:    "never",
		RawURL:  "https://example.com",
		ExpiresAt: time.Time{}, // no expiry
	}
	saveURL(t, svc, url)

	if svc.ValidateExpiry(url) {
		t.Fatalf("link with zero ExpiresAt reported expired")
	}
}

func TestCleanup_Preserves23hLink(t *testing.T) {
	// The headline scenario: cleanup with a 24h maxAge must keep a 23h-old link.
	now := time.Now().UTC()
	svc := newTestURLService(t)
	url := &model.ShortURL{
		Code:      "keep23",
		RawURL:    "https://example.com",
		CreatedAt: now.Add(-23 * time.Hour),
		ExpiresAt: now.Add(time.Hour),
	}
	saveURL(t, svc, url)

	removed, err := svc.Cleanup(24 * time.Hour)
	if err != nil {
		t.Fatalf("Cleanup: %v", err)
	}
	if removed != 0 {
		t.Fatalf("Cleanup removed %d entries, want 0 (23h link must survive a 24h maxAge)", removed)
	}
	got, err := svc.Get(url.Code)
	if err != nil {
		t.Fatalf("Get after cleanup: %v", err)
	}
	if got.Disabled {
		t.Fatalf("23h-old link was disabled by 24h cleanup")
	}
}

func TestCleanup_Removes25hLink(t *testing.T) {
	now := time.Now().UTC()
	svc := newTestURLService(t)
	url := &model.ShortURL{
		Code:      "rm25",
		RawURL:    "https://example.com",
		CreatedAt: now.Add(-25 * time.Hour),
		ExpiresAt: now.Add(-time.Hour), // also genuinely past ExpiresAt
	}
	saveURL(t, svc, url)

	removed, err := svc.Cleanup(24 * time.Hour)
	if err != nil {
		t.Fatalf("Cleanup: %v", err)
	}
	if removed != 1 {
		t.Fatalf("Cleanup removed %d entries, want 1", removed)
	}
	got, err := svc.Get(url.Code)
	if err != nil {
		t.Fatalf("Get after cleanup: %v", err)
	}
	if !got.Disabled {
		t.Fatalf("25h-old link was not disabled by 24h cleanup")
	}
}

func TestCleanup_RemovesOnlyOlderLink(t *testing.T) {
	now := time.Now().UTC()
	svc := newTestURLService(t)
	old := &model.ShortURL{
		Code:      "old25",
		RawURL:    "https://example.com/old",
		CreatedAt: now.Add(-25 * time.Hour),
		ExpiresAt: now.Add(-time.Hour),
	}
	fresh := &model.ShortURL{
		Code:      "fresh1",
		RawURL:    "https://example.com/fresh",
		CreatedAt: now.Add(-time.Hour),
		ExpiresAt: now.Add(23 * time.Hour),
	}
	saveURL(t, svc, old)
	saveURL(t, svc, fresh)

	removed, err := svc.Cleanup(24 * time.Hour)
	if err != nil {
		t.Fatalf("Cleanup: %v", err)
	}
	if removed != 1 {
		t.Fatalf("Cleanup removed %d entries, want 1 (only the 25h link)", removed)
	}
	gotFresh, err := svc.Get(fresh.Code)
	if err != nil {
		t.Fatalf("Get fresh: %v", err)
	}
	if gotFresh.Disabled {
		t.Fatalf("1h-old link was disabled by 24h cleanup")
	}
}
