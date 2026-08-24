package main

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/codesandbox/codesandbox/internal/model"
	"github.com/codesandbox/codesandbox/internal/store"
)

func TestRedGreen(t *testing.T) {
	memStore := store.NewMemoryStore()

	languages := []string{"python", "javascript", "shell"}
	keywords := []string{"ALPHA", "BETA", "GAMMA", "DELTA", "OMEGA"}
	recordsCount := 2000

	for i := 0; i < recordsCount; i++ {
		lang := languages[i%len(languages)]
		keyword := keywords[i%len(keywords)]
		code := fmt.Sprintf("print(\"hello %s world\")", keyword)
		stdout := fmt.Sprintf("output from execution number %d with keyword %s", i, keyword)
		record := &model.HistoryRecord{
			ID:          fmt.Sprintf("hist-%d", i),
			ExecutionID: fmt.Sprintf("exec-%d", i),
			Language:    lang,
			Code:        code,
			Status:      model.StatusCompleted,
			Stdout:      stdout,
			Stderr:      "",
			ExitCode:    0,
			Duration:    int64(i % 100),
			SubmittedBy: fmt.Sprintf("user-%d", i%10),
			ExecutedAt:  time.Now(),
		}
		if err := memStore.CreateHistory(record); err != nil {
			t.Fatalf("Failed to create history record %d: %v", i, err)
		}
	}

	t.Run("concurrent_search_correctness", func(t *testing.T) {
		const numGoroutines = 16
		const numIterations = 10
		errCh := make(chan error, numGoroutines*numIterations)

		var wg sync.WaitGroup
		for g := 0; g < numGoroutines; g++ {
			wg.Add(1)
			go func(goroutineID int) {
				defer wg.Done()
				for iter := 0; iter < numIterations; iter++ {
					keyword := keywords[(goroutineID+iter)%len(keywords)]
					query := model.HistoryQuery{
						Search:   keyword,
						Page:     1,
						PageSize: 100,
					}
					start := time.Now()
					records, total, err := memStore.ListHistory(query)
					elapsed := time.Since(start)

					if err != nil {
						errCh <- fmt.Errorf("goroutine %d iter %d: ListHistory returned error: %v", goroutineID, iter, err)
						continue
					}

					if elapsed > 5*time.Second {
						errCh <- fmt.Errorf("goroutine %d iter %d: ListHistory took too long: %v (total=%d)", goroutineID, iter, elapsed, total)
						continue
					}

					for _, r := range records {
						found := strings.Contains(strings.ToUpper(r.Code), keyword) ||
							strings.Contains(strings.ToUpper(r.Stdout), keyword) ||
							strings.Contains(strings.ToUpper(r.Stderr), keyword)
						if !found {
							errCh <- fmt.Errorf("goroutine %d iter %d: record %s does not contain keyword %q", goroutineID, iter, r.ID, keyword)
							break
						}
					}
				}
			}(g)
		}
		wg.Wait()
		close(errCh)

		var errs []string
		for e := range errCh {
			errs = append(errs, e.Error())
		}

		if len(errs) > 0 {
			t.Errorf("RED (红灯，缺陷未修复): 发现 %d 个并发搜索问题:\n%s", len(errs), strings.Join(errs, "\n"))
		} else {
			fmt.Println("GREEN (绿灯，缺陷已修复)")
		}
	})

	t.Run("keyword_at_string_boundary", func(t *testing.T) {
		singleStore := store.NewMemoryStore()

		boundaryRecord := &model.HistoryRecord{
			ID:          "hist-boundary",
			ExecutionID: "exec-boundary",
			Language:    "python",
			Code:        "print(\"hello world\")",
			Status:      model.StatusCompleted,
			Stdout:      "XALPHA",
			Stderr:      "",
			ExitCode:    0,
			Duration:    10,
			ExecutedAt:  time.Now(),
		}
		if err := singleStore.CreateHistory(boundaryRecord); err != nil {
			t.Fatalf("Failed to create boundary record: %v", err)
		}

		query := model.HistoryQuery{
			Search:   "ALPHA",
			Page:     1,
			PageSize: 20,
		}
		records, total, err := singleStore.ListHistory(query)
		if err != nil {
			t.Errorf("RED (红灯，缺陷未修复): ListHistory returned error: %v", err)
			return
		}
		if total == 0 || len(records) == 0 {
			t.Errorf("RED (红灯，缺陷未修复): Search for 'ALPHA' at string boundary returned 0 results, expected at least 1")
			return
		}
		fmt.Printf("GREEN (绿灯，缺陷已修复): Found %d record(s) for boundary keyword search\n", len(records))
	})

	t.Run("search_non_existent_keyword_perf", func(t *testing.T) {
		perfStore := store.NewMemoryStore()

		for i := 0; i < 3500; i++ {
			record := &model.HistoryRecord{
				ID:          fmt.Sprintf("hist-perf-%d", i),
				ExecutionID: fmt.Sprintf("exec-perf-%d", i),
				Language:    "python",
				Code:        fmt.Sprintf("x = %d\ny = x + 1\nprint(y)", i),
				Status:      model.StatusCompleted,
				Stdout:      fmt.Sprintf("result-%d", i),
				Stderr:      "",
				ExitCode:    0,
				Duration:    int64(i),
				ExecutedAt:  time.Now(),
			}
			if err := perfStore.CreateHistory(record); err != nil {
				t.Fatalf("Failed to create perf record %d: %v", i, err)
			}
		}

		query := model.HistoryQuery{
			Search:   "NONEXISTENT_KEYWORD",
			Language: "python",
			Page:     1,
			PageSize: 50,
		}

		start := time.Now()
		records, total, err := perfStore.ListHistory(query)
		elapsed := time.Since(start)

		if err != nil {
			t.Errorf("RED (红灯，缺陷未修复): ListHistory returned error: %v", err)
			return
		}

		if elapsed > 4*time.Second {
			t.Errorf("RED (红灯，缺陷未修复): Search took %v which exceeds 4s limit (total=%d, returned=%d) - O(n²) performance issue detected", elapsed, total, len(records))
			return
		}

		if len(records) > 0 {
			t.Errorf("RED (红灯，缺陷未修复): Search for non-existent keyword returned %d records, expected 0", len(records))
			return
		}

		fmt.Printf("GREEN (绿灯，缺陷已修复): Search completed in %v, correctly returned 0 results for non-existent keyword\n", elapsed)
	})

	t.Run("final_summary", func(t *testing.T) {
		fmt.Println("=== TestRedGreen Final Summary ===")
		fmt.Println("All subtests executed. Check above for RED/GREEN results.")
	})
}