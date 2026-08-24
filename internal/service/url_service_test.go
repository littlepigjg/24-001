package service

import (
	"context"
	"strconv"
	"sync"
	"testing"

	"github.com/codesandbox/codesandbox/internal/config"
	"github.com/codesandbox/codesandbox/internal/model"
	"github.com/codesandbox/codesandbox/internal/store"
)

func newTestURLService(t *testing.T) *URLService {
	t.Helper()
	cfg := config.Default()
	cfg.BaseDir = t.TempDir()
	cfg.Storage.FlushOnWrite(true)
	st, err := store.NewURLStore(cfg)
	if err != nil {
		t.Fatalf("NewURLStore: %v", err)
	}
	if err := st.Load(context.Background()); err != nil {
		t.Fatalf("Load: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	svc, err := NewURLService(cfg, st)
	if err != nil {
		t.Fatalf("NewURLService: %v", err)
	}
	return svc
}

// TestURLServiceCreateConcurrent reproduces the reported load test exactly:
// 30 goroutines each create 5 short links (150 requests total). All must
// succeed and produce 150 distinct codes with no duplicates / overwrites.
func TestURLServiceCreateConcurrent(t *testing.T) {
	svc := newTestURLService(t)

	const goroutines = 30
	const perG = 5
	const total = goroutines * perG

	type res struct {
		code string
		err  error
	}
	results := make([]res, total)

	var wg sync.WaitGroup
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < perG; i++ {
				idx := g*perG + i
				u, err := svc.Create(context.Background(), &model.CreateReq{
					RawURL: "https://example.com/" + strconv.Itoa(idx),
				})
				if err != nil {
					results[idx] = res{err: err}
					continue
				}
				results[idx] = res{code: u.Code}
			}
		}(g)
	}
	wg.Wait()

	seen := make(map[string]int, total)
	for i, r := range results {
		if r.err != nil {
			t.Fatalf("Create(%d) error: %v", i, r.err)
		}
		if r.code == "" {
			t.Fatalf("Create(%d) returned empty code", i)
		}
		seen[r.code]++
	}
	if len(seen) != total {
		t.Fatalf("expected %d distinct codes, got %d (duplicate codes collided/overwrote)", total, len(seen))
	}
}

// TestURLServiceCreateCustomCodeConcurrent: two concurrent requests for the
// same custom code must not both succeed.
func TestURLServiceCreateCustomCodeConcurrent(t *testing.T) {
	svc := newTestURLService(t)

	const goroutines = 20
	var wg sync.WaitGroup
	successes := 0
	var mu sync.Mutex
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := svc.Create(context.Background(), &model.CreateReq{
				RawURL:     "https://example.com/custom",
				CustomCode: "shared123",
			})
			if err == nil {
				mu.Lock()
				successes++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	if successes != 1 {
		t.Fatalf("expected exactly 1 success for a shared custom code, got %d", successes)
	}
}
