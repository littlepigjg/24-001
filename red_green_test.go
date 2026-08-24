package codesandbox

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/codesandbox/codesandbox/internal/config"
	"github.com/codesandbox/codesandbox/internal/model"
	"github.com/codesandbox/codesandbox/internal/service"
	"github.com/codesandbox/codesandbox/internal/store"
)

func TestRedGreen(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("TMPDIR", tmpDir)

	countTempDirs := func(prefix string) int {
		count := 0
		entries, err := os.ReadDir(os.TempDir())
		if err != nil {
			return 0
		}
		for _, e := range entries {
			if e.IsDir() && strings.HasPrefix(e.Name(), prefix) {
				count++
			}
		}
		return count
	}

	cfg := config.Default()
	cfg.Storage.URLFilePath(filepath.Join(tmpDir, "urls.json"))
	cfg.Storage.LogFilePath(filepath.Join(tmpDir, "access.log"))

	urlStore, err := store.NewURLStore(cfg)
	if err != nil {
		t.Fatalf("failed to create URLStore: %v", err)
	}

	logStore, err := store.NewAccessLogStore(cfg)
	if err != nil {
		t.Fatalf("failed to create AccessLogStore: %v", err)
	}

	if err := logStore.Open(context.Background()); err != nil {
		t.Fatalf("failed to open log store: %v", err)
	}

	urlService, err := service.NewURLService(cfg, urlStore)
	if err != nil {
		t.Fatalf("failed to create URLService: %v", err)
	}

	redirectService, err := service.NewRedirectService(urlStore, logStore)
	if err != nil {
		t.Fatalf("failed to create RedirectService: %v", err)
	}

	svcBeforeCount := countTempDirs("urlsvc-")
	redirectBeforeCount := countTempDirs("redirect-")

	ctx := context.Background()

	for i := 0; i < 50; i++ {
		req := &model.CreateReq{
			RawURL:    "https://example.com/path",
			CustomCode: "",
			MaxVisits: 100,
		}
		_, err := urlService.Create(ctx, req)
		if err != nil {
			t.Fatalf("Create failed at iteration %d: %v", i, err)
		}
	}

	for i := 0; i < 20; i++ {
		req := &model.CreateReq{
			RawURL:    "https://example.com/clean",
			CustomCode: "",
			MaxVisits: 0,
		}
		_, err := urlService.Create(ctx, req)
		if err != nil {
			t.Fatalf("Create (clean) failed at iteration %d: %v", i, err)
		}
	}

	urlStore.SyncNow()
	time.Sleep(50 * time.Millisecond)

	svcAfterCount := countTempDirs("urlsvc-")
	redirectAfterCount := countTempDirs("redirect-")

	for i := 0; i < 10; i++ {
		snapshot := urlStore.RawSnapshot()
		for code := range snapshot {
			req := &model.RedirectRequest{
				Code:      code,
				Timestamp: time.Now(),
			}
			redirectService.HandleRedirect(ctx, req)
		}
		time.Sleep(20 * time.Millisecond)
	}

	redirectFinalCount := countTempDirs("redirect-")
	svcFinalCount := countTempDirs("urlsvc-")

	urlStore.Close()
	logStore.Close()

	storeFinalCount := countTempDirs("urlstore-")
	logFinalCount := countTempDirs("accesslog-")
	svcLeaked := countTempDirs("urlsvc-")
	redirectLeaked := countTempDirs("redirect-")

	if svcLeaked > 0 || redirectLeaked > 0 {
		t.Log("RED（红灯，缺陷未修复）")
		t.Logf("  urlsvc leaked dirs: %d (before=%d, after_create=%d, final=%d)", svcLeaked, svcBeforeCount, svcAfterCount, svcFinalCount)
		t.Logf("  redirect leaked dirs: %d (before=%d, after_create=%d, final=%d)", redirectLeaked, redirectBeforeCount, redirectAfterCount, redirectFinalCount)
		t.Logf("  urlstore remaining: %d, accesslog remaining: %d", storeFinalCount, logFinalCount)
		t.FailNow()
	} else {
		t.Log("GREEN（绿灯，缺陷已修复）")
		t.Logf("  urlsvc dirs: %d, redirect dirs: %d, urlstore dirs: %d, accesslog dirs: %d",
			svcLeaked, redirectLeaked, storeFinalCount, logFinalCount)
	}
}
