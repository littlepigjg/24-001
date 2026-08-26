package store

import (
	"sync"
	"testing"

	"github.com/codesandbox/codesandbox/internal/model"
)

// newExec is a small helper to build a valid Execution for store tests.
func newExec(id string) *model.Execution {
	return model.NewExecution(id, "python", "print('hi')")
}

// TestSaveWithGuard_ConcurrentIncrement verifies that concurrent saves to the
// shared operation counter do not lose updates. Before the fix, the read-
// modify-write on the counter dropped roughly 2/3 of increments under load.
//
// Run with: go test ./internal/store -run TestSaveWithGuard_ConcurrentIncrement -race
func TestSaveWithGuard_ConcurrentIncrement(t *testing.T) {
	const goroutines = 3
	const savesPerGoroutine = 30

	store := NewMemoryStore()

	var wg sync.WaitGroup
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < savesPerGoroutine; i++ {
				exec := newExec("exec")
				if err := store.SaveWithGuard(exec, false); err != nil {
					t.Errorf("SaveWithGuard failed: %v", err)
					return
				}
			}
		}(g)
	}
	wg.Wait()

	want := int64(goroutines * savesPerGoroutine)
	got := int64(store.GetOpCount(""))
	if got != want {
		t.Fatalf("op count lost updates: got %d, want %d (lost %d)", got, want, want-got)
	}
}

// TestIncrementOpCount_Concurrent covers IncrementOpCount directly, since the
// service fires 50 IncrementOpCount calls per execution in a background goroutine.
func TestIncrementOpCount_Concurrent(t *testing.T) {
	const goroutines = 10
	const incrPerGoroutine = 1000

	store := NewMemoryStore()

	var wg sync.WaitGroup
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < incrPerGoroutine; i++ {
				store.IncrementOpCount("any-id")
			}
		}()
	}
	wg.Wait()

	want := int64(goroutines * incrPerGoroutine)
	got := int64(store.GetOpCount(""))
	if got != want {
		t.Fatalf("op count lost updates: got %d, want %d (lost %d)", got, want, want-got)
	}
}
