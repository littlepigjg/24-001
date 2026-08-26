package store

import (
	"context"
	"testing"
	"time"

	"github.com/codesandbox/codesandbox/internal/config"
	"github.com/codesandbox/codesandbox/internal/model"
)

// TestURLStoreGetAfterSaveNoPanic reproduces the production crash:
// "interface conversion: interface {} is int64, not string". URLStore.Save
// caches the visits count as int64; URLStore.Get then read it via cache.Get,
// which previously hard-coded item.Value.(string) and panicked. After a fresh
// Create the very first access (redirect or detail lookup) crashed with a 500.
// This test performs Save then Get and asserts the struct is returned intact.
func TestURLStoreGetAfterSaveNoPanic(t *testing.T) {
	cfg := &config.Config{} // empty urlFilePath => no disk I/O, flush is a no-op

	s, err := NewURLStore(cfg)
	if err != nil {
		t.Fatalf("NewURLStore: %v", err)
	}
	if err := s.Load(context.Background()); err != nil {
		t.Fatalf("Load: %v", err)
	}

	u := &model.ShortURL{
		Code:      "abc12345",
		RawURL:    "https://example.com",
		Visits:    0,
		Custom:    false,
		Disabled:  false,
		CreatedAt: time.Now(),
	}
	if err := u.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}

	if err := s.Save(u, false); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := s.Get("abc12345")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Code != u.Code || got.RawURL != u.RawURL {
		t.Fatalf("Get returned mismatched record: %+v", got)
	}
	if got.Visits != 0 {
		t.Fatalf("expected visits 0, got %d", got.Visits)
	}

	// IncrementVisits then Get again — the visits cache entry is now a
	// non-zero int64, which is exactly the value type that used to panic.
	if err := s.IncrementVisits("abc12345"); err != nil {
		t.Fatalf("IncrementVisits: %v", err)
	}
	got, err = s.Get("abc12345")
	if err != nil {
		t.Fatalf("Get after increment: %v", err)
	}
	if got.Visits != 1 {
		t.Fatalf("expected visits 1, got %d", got.Visits)
	}
}
