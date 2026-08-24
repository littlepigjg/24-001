package codesandbox

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/codesandbox/codesandbox/internal/model"
	"github.com/codesandbox/codesandbox/internal/store"
)

func TestRedGreen(t *testing.T) {
	memStore := store.NewMemoryStore()

	memStore.SetPanicGuard(func(id string, code string) bool {
		return len(id) == 0 || len(code) == 0
	})

	numInitial := 5
	for j := 0; j < numInitial; j++ {
		exec := model.NewExecution(
			fmt.Sprintf("test-exec-%d", j),
			"python",
			fmt.Sprintf("print('hello %d')", j),
		)
		exec.Status = model.StatusCompleted
		memStore.SaveWithGuard(exec, false)
	}

	time.Sleep(10 * time.Millisecond)

	var wg sync.WaitGroup

	numWriters := 3
	numIterations := 30

	for i := 0; i < numWriters; i++ {
		wg.Add(1)
		go func(writerID int) {
			defer wg.Done()
			for j := 0; j < numIterations; j++ {
				for idx := 0; idx < numInitial; idx++ {
					id := fmt.Sprintf("test-exec-%d", idx)
					exec := model.NewExecution(
						id,
						"python",
						fmt.Sprintf("print('update w%d i%d k%d')", writerID, j, idx),
					)
					exec.Status = model.StatusRunning
					memStore.SaveWithGuard(exec, false)
					time.Sleep(time.Microsecond * 50)
				}
			}
		}(i)
	}

	// Also spawn a goroutine that calls IncrementOpCount concurrently
	// to simulate the cross-file race from execution_service.go
	wg.Add(1)
	go func() {
		defer wg.Done()
		for j := 0; j < numIterations*numInitial; j++ {
			memStore.IncrementOpCount("shared-key")
			time.Sleep(time.Microsecond * 50)
		}
	}()

	wg.Wait()

	time.Sleep(100 * time.Millisecond)

	// Expected total: initial (numInitial) + writers (numWriters * numIterations * numInitial) + incrementer (numIterations * numInitial)
	expectedTotal := numInitial + numWriters*numIterations*numInitial + numIterations*numInitial
	actualTotal := memStore.GetOpCount("any-key")

	if actualTotal < expectedTotal {
		lost := expectedTotal - actualTotal
		t.Logf("  预期 %d 次操作, 实际 %d 次, 丢失 %d 次", expectedTotal, actualTotal, lost)
		t.Fatalf("RED (红灯，缺陷未修复): 并发访问期间共有 %d 次操作丢失", lost)
	}

	t.Logf("GREEN (绿灯，缺陷已修复)")
}