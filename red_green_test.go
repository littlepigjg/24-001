package codesandbox

import (
	"context"
	"fmt"
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
	tmpDir, err := os.MkdirTemp("", "url_store_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	canaryPath := filepath.Join(tmpDir, "..", "canary_target.json")
	canaryContent := `{"code":"canary","raw_url":"http://canary.example.com","created_at":"2024-01-01T00:00:00Z","visits":0,"custom":false,"disabled":false}`
	if err := os.WriteFile(canaryPath, []byte(canaryContent), 0644); err != nil {
		t.Fatalf("failed to create canary file: %v", err)
	}
	defer os.Remove(canaryPath)

	urlStorePath := filepath.Join(tmpDir, "urls")
	logStorePath := filepath.Join(tmpDir, "access.log")

	cfg := config.Default()
	cfg.Storage.URLFilePath(urlStorePath)
	cfg.Storage.LogFilePath(logStorePath)

	urlStore, err := store.NewURLStore(cfg)
	if err != nil {
		t.Fatalf("failed to create URL store: %v", err)
	}

	ctx := context.Background()
	if err := urlStore.Load(ctx); err != nil {
		t.Fatalf("failed to load URL store: %v", err)
	}
	defer urlStore.Close()

	logStore, err := store.NewAccessLogStore(cfg)
	if err != nil {
		t.Fatalf("failed to create access log store: %v", err)
	}
	if err := logStore.Open(ctx); err != nil {
		t.Fatalf("failed to open access log store: %v", err)
	}
	defer logStore.Close()

	urlService, err := service.NewURLService(cfg, urlStore)
	if err != nil {
		t.Fatalf("failed to create URL service: %v", err)
	}

	redirectService, err := service.NewRedirectService(urlStore, logStore)
	if err != nil {
		t.Fatalf("failed to create redirect service: %v", err)
	}

	normalReq := &model.CreateReq{
		RawURL:    "https://example.com/normal",
		CustomCode: "normal01",
		MaxVisits: 100,
	}
	_, err = urlService.Create(ctx, normalReq)
	if err != nil {
		t.Fatalf("failed to create normal URL: %v", err)
	}

	dangerousCode := "../canary_target"
	dangerousReq := &model.CreateReq{
		RawURL:    "https://example.com/dangerous",
		CustomCode: dangerousCode,
		MaxVisits: 100,
	}
	_, err = urlService.Create(ctx, dangerousReq)

	var traversalDetected bool
	if err != nil {
		errStr := err.Error()
		if strings.Contains(errStr, "already exists") {
			traversalDetected = true
		} else if strings.Contains(errStr, "invalid") || strings.Contains(errStr, "forbidden") || strings.Contains(errStr, "path") {
			traversalDetected = false
		} else {
			_, getErr := urlStore.Get(dangerousCode)
			if getErr != nil {
				traversalDetected = true
			} else {
				traversalDetected = true
			}
		}
	} else {
		traversalDetected = true
	}

	if !traversalDetected {
		_, getErr := urlStore.Get(dangerousCode)
		if getErr == nil {
			traversalDetected = true
		} else if !strings.Contains(getErr.Error(), "not found") {
			traversalDetected = true
		}
	}

	redirectReq := &service.RedirectRequest{
		Code:      dangerousCode,
		Timestamp: time.Now(),
	}
	result, redirectErr := redirectService.HandleRedirect(ctx, redirectReq)
	if redirectErr == nil && result != nil && result.Status != 404 && result.Status != 410 {
		if strings.Contains(result.RawURL, "canary") {
			traversalDetected = true
		}
	}

	if traversalDetected {
		fmt.Println("RED (红灯，缺陷未修复)")
		t.Error("RED (红灯，缺陷未修复)")
	} else {
		fmt.Println("GREEN (绿灯，缺陷已修复)")
	}
}
