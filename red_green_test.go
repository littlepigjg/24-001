package codesandbox_test

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/codesandbox/codesandbox/internal/config"
	"github.com/codesandbox/codesandbox/internal/model"
	"github.com/codesandbox/codesandbox/internal/service"
	"github.com/codesandbox/codesandbox/internal/store"
)

func TestRedGreen(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "urlstore-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := config.Default()
	cfg.BaseDir = tmpDir

	us, err := store.NewURLStore(cfg)
	if err != nil {
		t.Fatal(err)
	}

	if err := us.Load(context.Background()); err != nil {
		t.Fatal(err)
	}

	ls, err := store.NewAccessLogStore(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := ls.Open(context.Background()); err != nil {
		t.Fatal(err)
	}

	urlSvc, err := service.NewURLService(cfg, us)
	if err != nil {
		t.Fatal(err)
	}

	redirSvc, err := service.NewRedirectService(us, ls)
	if err != nil {
		t.Fatal(err)
	}

	const numWorkers = 30
	const numOps = 5

	var wg sync.WaitGroup
	errCh := make(chan error, numWorkers*numOps)

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for op := 0; op < numOps; op++ {
				code := fmt.Sprintf("w%d-op%d", workerID, op)
				req := &model.CreateReq{
					RawURL: fmt.Sprintf("https://example.com/w%d/op%d", workerID, op),
				}
				_, err := urlSvc.Create(context.Background(), req)
				if err != nil {
					errCh <- fmt.Errorf("create failed for %s: %w", code, err)
					continue
				}

				rr, err := redirSvc.HandleRedirect(context.Background(), &service.RedirectRequest{
					Code:      code,
					Timestamp: time.Now(),
				})
				if err != nil {
					errCh <- fmt.Errorf("redirect failed for %s: %w", code, err)
					continue
				}
				if rr.Status != 302 {
					errCh <- fmt.Errorf("unexpected redirect status for %s: %d", code, rr.Status)
				}
			}
		}(w)
	}

	wg.Wait()
	close(errCh)

	errCount := 0
	for e := range errCh {
		if e != nil {
			errCount++
			t.Log("Error:", e)
		}
	}

	snap := us.RawSnapshot()
	expectedCount := numWorkers * numOps

	if errCount == 0 && len(snap) == expectedCount {
		fmt.Println("GREEN（绿灯，缺陷已修复）")
	} else {
		fmt.Printf("RED（红灯，缺陷未修复）：错误数=%d，快照大小=%d/%d\n", errCount, len(snap), expectedCount)
		t.FailNow()
	}

	us.Close()
	ls.Close()
}