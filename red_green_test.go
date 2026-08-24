package codesandbox

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/codesandbox/codesandbox/internal/config"
	"github.com/codesandbox/codesandbox/internal/model"
	"github.com/codesandbox/codesandbox/internal/store"
	"github.com/codesandbox/codesandbox/internal/service"
)

func TestRedGreen(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "url-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := config.Default()
	cfg.Storage.URLFilePath(filepath.Join(tmpDir, "urls.json"))
	cfg.Storage.LogFilePath(filepath.Join(tmpDir, "access.log"))
	cfg.Storage.FlushOnWrite(false)

	urlStore, err := store.NewURLStore(cfg)
	if err != nil {
		t.Fatalf("Failed to create URL store: %v", err)
	}

	logStore, err := store.NewAccessLogStore(cfg)
	if err != nil {
		t.Fatalf("Failed to create access log store: %v", err)
	}

	ctx := context.Background()
	if err := logStore.Open(ctx); err != nil {
		t.Fatalf("Failed to open access log store: %v", err)
	}

	urlService, err := service.NewURLService(cfg, urlStore)
	if err != nil {
		t.Fatalf("Failed to create URL service: %v", err)
	}

	redirectService, err := service.NewRedirectService(urlStore, logStore)
	if err != nil {
		t.Fatalf("Failed to create redirect service: %v", err)
	}

	now := time.Now()
	createdAt := now.Add(-23 * time.Hour)

	testURL := &model.ShortURL{
		Code:      "testcode",
		RawURL:    "https://example.com/test",
		CreatedAt: createdAt.UTC(),
		ExpiresAt: createdAt.UTC().Add(24 * time.Hour),
		Visits:    0,
		Custom:    false,
		Disabled:  false,
	}

	if err := urlStore.Save(testURL, false); err != nil {
		t.Fatalf("Failed to save test URL: %v", err)
	}

	isExpired := redirectService.IsExpired(testURL)

	if isExpired {
		fmt.Println("RED（红灯，缺陷未修复）")
		fmt.Println("URL incorrectly detected as expired due to timezone mismatch")
		t.Error("URL should NOT be expired - only 23 hours old with 24 hour max age")
	} else {
		fmt.Println("GREEN（绿灯，缺陷已修复）")
		fmt.Println("URL correctly identified as not expired")
	}

	removed, err := urlService.Cleanup(24 * time.Hour)
	if err != nil {
		t.Fatalf("Cleanup failed: %v", err)
	}

	if removed > 0 {
		fmt.Println("RED（红灯，缺陷未修复）")
		fmt.Printf("Cleanup incorrectly removed %d URL(s) due to timezone mismatch\n", removed)
		t.Errorf("Cleanup should NOT remove URLs that are only 23 hours old with 24 hour max age")
	} else {
		fmt.Println("GREEN（绿灯，缺陷已修复）")
		fmt.Println("Cleanup correctly preserved the valid URL")
	}

	urlAfter, err := urlStore.Get("testcode")
	if err != nil {
		t.Fatalf("Failed to get URL after cleanup: %v", err)
	}

	if urlAfter.Disabled {
		fmt.Println("RED（红灯，缺陷未修复）")
		fmt.Println("URL incorrectly disabled due to timezone mismatch in cleanup")
		t.Error("URL should remain enabled")
	} else {
		fmt.Println("GREEN（绿灯，缺陷已修复）")
		fmt.Println("URL correctly remains enabled after cleanup")
	}

	logStore.Close()
	urlStore.Close()

	if t.Failed() {
		fmt.Println("最终判定: RED（红灯，缺陷未修复）")
	} else {
		fmt.Println("最终判定: GREEN（绿灯，缺陷已修复）")
	}
}
