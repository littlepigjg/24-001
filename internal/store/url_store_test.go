package store

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/codesandbox/codesandbox/internal/config"
	"github.com/codesandbox/codesandbox/internal/model"
)

func newTestURLStore(t *testing.T) *URLStore {
	t.Helper()
	dir := t.TempDir()
	cfg := config.Default()
	cfg.BaseDir = dir
	cfg.Storage.FlushOnWrite(true)
	s, err := NewURLStore(cfg)
	if err != nil {
		t.Fatalf("NewURLStore: %v", err)
	}
	if err := s.Load(context.Background()); err != nil {
		t.Fatalf("Load: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func mkShortURL(i int) *model.ShortURL {
	return &model.ShortURL{
		Code:      fmt.Sprintf("code%03d", i),
		RawURL:    fmt.Sprintf("https://example.com/%d", i),
		MaxVisits: 0,
	}
}

// TestURLStoreConcurrentSaves reproduces the reported scenario: many goroutines
// creating short links concurrently. With the bug, this races on the urls map
// (panic under -race) and loses records because each Save re-snapshots the
// whole map without a lock. After the fix, every code must be present.
func TestURLStoreConcurrentSaves(t *testing.T) {
	s := newTestURLStore(t)

	const goroutines = 30
	const perG = 5
	const total = goroutines * perG

	var wg sync.WaitGroup
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < perG; i++ {
				idx := g*perG + i
				if err := s.Save(mkShortURL(idx), false); err != nil {
					t.Errorf("Save(%d): %v", idx, err)
				}
			}
		}(g)
	}
	wg.Wait()

	got := s.RawSnapshot()
	if len(got) != total {
		t.Fatalf("expected %d records, got %d (data loss)", total, len(got))
	}
	for i := 0; i < total; i++ {
		code := fmt.Sprintf("code%03d", i)
		if _, ok := got[code]; !ok {
			t.Errorf("missing record %s", code)
		}
	}
}

// TestURLStorePersistsAllRecords verifies the persisted file actually contains
// every record after concurrent writes (guards against the lost-snapshot bug).
func TestURLStorePersistsAllRecords(t *testing.T) {
	s := newTestURLStore(t)

	const total = 150
	var wg sync.WaitGroup
	for i := 0; i < total; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if err := s.Save(mkShortURL(i), false); err != nil {
				t.Errorf("Save(%d): %v", i, err)
			}
		}(i)
	}
	wg.Wait()

	// Read the on-disk file directly and confirm nothing was lost.
	data, err := os.ReadFile(s.dataFile)
	if err != nil {
		t.Fatalf("read data file: %v", err)
	}
	var persisted map[string]model.ShortURL
	if err := json.Unmarshal(data, &persisted); err != nil {
		t.Fatalf("parse data file: %v (corrupt/torn write)", err)
	}
	if len(persisted) != total {
		t.Fatalf("persisted expected %d, got %d (lost data on disk)", total, len(persisted))
	}
}

// TestURLStoreUpdateConcurrentVisits checks the atomic read-modify-write
// path used for visit counting. Lost updates were the redirect bug.
func TestURLStoreUpdateConcurrentVisits(t *testing.T) {
	s := newTestURLStore(t)
	if err := s.Save(&model.ShortURL{Code: "visit", RawURL: "https://example.com/v"}, true); err != nil {
		t.Fatalf("Save: %v", err)
	}

	const goroutines = 50
	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := s.Update("visit", func(u *model.ShortURL) error {
				u.Visits++
				return nil
			}); err != nil {
				t.Errorf("Update: %v", err)
			}
		}()
	}
	wg.Wait()

	u, err := s.Get("visit")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if u.Visits != goroutines {
		t.Fatalf("expected %d visits, got %d (lost updates)", goroutines, u.Visits)
	}
}

// TestURLStoreGetReturnsCopy ensures callers cannot mutate the shared map via
// the returned pointer.
func TestURLStoreGetReturnsCopy(t *testing.T) {
	s := newTestURLStore(t)
	if err := s.Save(&model.ShortURL{Code: "c1", RawURL: "https://example.com/1"}, true); err != nil {
		t.Fatalf("Save: %v", err)
	}

	u, err := s.Get("c1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	u.Visits = 999
	u2, _ := s.Get("c1")
	if u2.Visits == 999 {
		t.Fatalf("Get returned a pointer into the map; mutation leaked into the store")
	}
}

func TestURLStoreAtomicFileNotCorruptOnConcurrentWrites(t *testing.T) {
	s := newTestURLStore(t)

	const goroutines = 30
	var wg sync.WaitGroup
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < 5; i++ {
				idx := g*5 + i
				_ = s.Save(mkShortURL(idx), false)
			}
		}(g)
	}
	wg.Wait()

	// The data file must be valid JSON (no torn writes).
	if _, err := os.ReadFile(s.dataFile); err != nil {
		t.Fatalf("read data file: %v", err)
	}
	// Reopen a fresh store from the same file to confirm loadability.
	dir := filepath.Dir(s.dataFile)
	cfg := config.Default()
	cfg.BaseDir = dir
	cfg.Storage.FlushOnWrite(true)
	s2, err := NewURLStore(cfg)
	if err != nil {
		t.Fatalf("NewURLStore: %v", err)
	}
	if err := s2.Load(context.Background()); err != nil {
		t.Fatalf("reload store: %v", err)
	}
	if got := len(s2.RawSnapshot()); got != goroutines*5 {
		t.Fatalf("after reload expected %d, got %d", goroutines*5, got)
	}
}
