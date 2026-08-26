package service

import (
	"context"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/codesandbox/codesandbox/internal/config"
	"github.com/codesandbox/codesandbox/internal/model"
	"github.com/codesandbox/codesandbox/internal/store"
)

func newTestService(t *testing.T) (*RedirectService, *store.URLStore, *store.AccessLogStore) {
	t.Helper()
	cfg := config.Default()
	us, err := store.NewURLStore(cfg)
	if err != nil {
		t.Fatalf("NewURLStore: %v", err)
	}
	if err := us.Load(context.Background()); err != nil {
		t.Fatalf("Load: %v", err)
	}
	ls, err := store.NewAccessLogStore(cfg)
	if err != nil {
		t.Fatalf("NewAccessLogStore: %v", err)
	}
	if err := ls.Open(context.Background()); err != nil {
		t.Fatalf("Open: %v", err)
	}
	svc, err := NewRedirectService(us, ls)
	if err != nil {
		t.Fatalf("NewRedirectService: %v", err)
	}
	return svc, us, ls
}

func seedURL(t *testing.T, us *store.URLStore, code string) *model.ShortURL {
	t.Helper()
	u := &model.ShortURL{
		Code:      code,
		RawURL:    "https://example.com/" + code,
		CreatedAt: time.Now(),
	}
	if err := us.Save(u, false); err != nil {
		t.Fatalf("seed Save: %v", err)
	}
	// Return the freshly-fetched record so callers start from the canonical
	// stored value (Visits == 0).
	got, err := us.Get(code)
	if err != nil {
		t.Fatalf("seed Get: %v", err)
	}
	return got
}

// baseline captures the bookkeeping counters so tests can assert on the delta
// caused by a redirect — independent of the pending entry that the seed Save
// itself appends (the seed creates data, not a visit).
type baseline struct {
	visits        int
	pendingWrites int
	logCount      int
}

func captureBaseline(us *store.URLStore, ls *store.AccessLogStore, code string) baseline {
	u, _ := us.Get(code)
	visits := 0
	if u != nil {
		visits = u.Visits
	}
	return baseline{
		visits:        visits,
		pendingWrites: us.PendingWritesCount(),
		logCount:      ls.Count(),
	}
}

// A canceled context must result in zero bookkeeping delta: no visit increment,
// no pendingWrites entry, and no access log entry.
func TestHandleRedirect_CanceledContextDoesNoBookkeeping(t *testing.T) {
	svc, us, ls := newTestService(t)
	seedURL(t, us, "canceled")
	base := captureBaseline(us, ls, "canceled")

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // client timed out / disconnected before we even started

	res, err := svc.HandleRedirect(ctx, &RedirectRequest{Code: "canceled", Timestamp: time.Now()})
	if err == nil {
		t.Fatalf("expected error from canceled context, got result %+v", res)
	}

	got, err := us.Get("canceled")
	if err != nil {
		t.Fatalf("URL record should still exist after cancellation: %v", err)
	}
	if got.Visits != base.visits {
		t.Fatalf("Visits delta must be 0 on a canceled request; visits %d -> %d", base.visits, got.Visits)
	}
	if n := us.PendingWritesCount(); n != base.pendingWrites {
		t.Fatalf("pendingWrites delta must be 0 on a canceled request; %d -> %d", base.pendingWrites, n)
	}
	if n := ls.Count(); n != base.logCount {
		t.Fatalf("access log delta must be 0 on a canceled request; %d -> %d", base.logCount, n)
	}
}

// Under load, each canceled request must contribute exactly zero to
// bookkeeping while each live request contributes exactly one visit, one
// pendingWrite, and one log entry. pendingWrites and access logs are global
// slices, so we assert on the global delta (== number of live requests) plus
// per-code Visits. Each goroutine uses its own code so the assertion is about
// cancellation semantics, not concurrent read-modify-write races on a shared
// counter.
func TestHandleRedirect_ConcurrentMixedCancellation(t *testing.T) {
	svc, us, ls := newTestService(t)

	const total = 200
	const canceledCount = total / 2
	const liveCount = total - canceledCount
	_ = canceledCount

	// Seed one URL per goroutine.
	codes := make([]string, total)
	for i := 0; i < total; i++ {
		codes[i] = "u-" + strconv.Itoa(i)
		seedURL(t, us, codes[i])
	}

	// Global baseline captured once after all seeds (pendingWrites/logs are
	// global slices shared across all codes).
	pendingBefore := us.PendingWritesCount()
	logBefore := ls.Count()

	var wg sync.WaitGroup
	wg.Add(total)
	start := make(chan struct{})
	for i := 0; i < total; i++ {
		canceled := i%2 == 0
		code := codes[i]
		go func() {
			defer wg.Done()
			<-start
			ctx := context.Background()
			if canceled {
				c, cancel := context.WithCancel(ctx)
				cancel()
				ctx = c
			}
			_, _ = svc.HandleRedirect(ctx, &RedirectRequest{Code: code, Timestamp: time.Now()})
		}()
	}
	close(start)
	wg.Wait()

	// Each code's Visits must reflect exactly its own request: 0 if canceled, 1
	// if live.
	for i, code := range codes {
		got, err := us.Get(code)
		if err != nil {
			t.Fatalf("Get(%s): %v", code, err)
		}
		wantVisits := 0
		if i%2 != 0 { // live
			wantVisits = 1
		}
		if got.Visits != wantVisits {
			t.Fatalf("code %s: Visits = %d, want %d", code, got.Visits, wantVisits)
		}
	}

	// Global slices must have grown by exactly liveCount — canceled requests
	// appended nothing.
	if delta := us.PendingWritesCount() - pendingBefore; delta != liveCount {
		t.Fatalf("pendingWrites global delta = %d, want %d", delta, liveCount)
	}
	if delta := ls.Count() - logBefore; delta != liveCount {
		t.Fatalf("access log global delta = %d, want %d", delta, liveCount)
	}
}

// A live request after a canceled one must still work and be counted, proving
// cancellation does not "stick" globally (per-request semantics).
func TestHandleRedirect_CancellationDoesNotStick(t *testing.T) {
	svc, us, ls := newTestService(t)
	seedURL(t, us, "stick")
	base := captureBaseline(us, ls, "stick")

	// First: canceled.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, _ = svc.HandleRedirect(ctx, &RedirectRequest{Code: "stick", Timestamp: time.Now()})
	if delta := us.PendingWritesCount() - base.pendingWrites; delta != 0 {
		t.Fatalf("canceled request added pendingWrites: delta %d, want 0", delta)
	}

	// Second: live.
	if _, err := svc.HandleRedirect(context.Background(), &RedirectRequest{Code: "stick", Timestamp: time.Now()}); err != nil {
		t.Fatalf("live redirect after cancellation failed: %v", err)
	}

	got, err := us.Get("stick")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if delta := got.Visits - base.visits; delta != 1 {
		t.Fatalf("Visits delta = %d, want 1 (only the live request counts)", delta)
	}
	if delta := us.PendingWritesCount() - base.pendingWrites; delta != 1 {
		t.Fatalf("pendingWrites delta = %d, want 1", delta)
	}
	if delta := ls.Count() - base.logCount; delta != 1 {
		t.Fatalf("access log delta = %d, want 1", delta)
	}
}

// Sanity: a normal (non-canceled) request still does full bookkeeping.
func TestHandleRedirect_LiveRequestBookkeeps(t *testing.T) {
	svc, us, ls := newTestService(t)
	seedURL(t, us, "live")
	base := captureBaseline(us, ls, "live")

	if _, err := svc.HandleRedirect(context.Background(), &RedirectRequest{Code: "live", Timestamp: time.Now()}); err != nil {
		t.Fatalf("HandleRedirect: %v", err)
	}

	got, err := us.Get("live")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if delta := got.Visits - base.visits; delta != 1 {
		t.Fatalf("Visits delta = %d, want 1", delta)
	}
	if delta := us.PendingWritesCount() - base.pendingWrites; delta != 1 {
		t.Fatalf("pendingWrites delta = %d, want 1", delta)
	}
	if delta := ls.Count() - base.logCount; delta != 1 {
		t.Fatalf("access log delta = %d, want 1", delta)
	}
}
