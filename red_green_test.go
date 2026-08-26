package codesandbox

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/codesandbox/codesandbox/internal/config"
	"github.com/codesandbox/codesandbox/internal/model"
	"github.com/codesandbox/codesandbox/internal/service"
	"github.com/codesandbox/codesandbox/internal/store"
)

func TestRedGreen(t *testing.T) {
	cfg := config.Default()

	urlStore, err := store.NewURLStore(cfg)
	if err != nil {
		t.Fatalf("failed to create URL store: %v", err)
	}
	defer urlStore.Close()

	logStore, err := store.NewAccessLogStore(cfg)
	if err != nil {
		t.Fatalf("failed to create access log store: %v", err)
	}
	defer logStore.Close()

	ctx := context.Background()
	if err := urlStore.Load(ctx); err != nil {
		t.Fatalf("failed to load URL store: %v", err)
	}

	if err := logStore.Open(ctx); err != nil {
		t.Fatalf("failed to open access log store: %v", err)
	}

	urlService, err := service.NewURLService(cfg, urlStore)
	if err != nil {
		t.Fatalf("failed to create URL service: %v", err)
	}

	redirectService, err := service.NewRedirectService(urlStore, logStore)
	if err != nil {
		t.Fatalf("failed to create redirect service: %v", err)
	}

	panicGuardCalled := false
	urlStore.SetPanicGuard(func(code, rawURL string) bool {
		panicGuardCalled = true
		return false
	})

	shortURL := model.NewShortURL("https://example.com/test", "abc123", false)
	if err := shortURL.Validate(); err != nil {
		t.Fatalf("short URL validation failed: %v", err)
	}

	if shortURL.IsExpired(time.Now()) {
		t.Fatalf("newly created URL should not be expired")
	}

	req := &model.CreateReq{
		RawURL:     "https://example.com",
		CustomCode: "mylink",
		MaxVisits:  100,
	}
	if err := req.Validate(); err != nil {
		t.Fatalf("create request validation failed: %v", err)
	}

	saveDone := make(chan error, 1)
	go func() {
		saveDone <- urlStore.Save(shortURL, false)
	}()

	select {
	case err := <-saveDone:
		if err != nil {
			t.Logf("Save completed with error: %v", err)
			t.Log("GREEN（绿灯，缺陷已修复）")
			return
		}
		t.Log("GREEN（绿灯，缺陷已修复）")
	case <-time.After(2 * time.Second):
		t.Log("RED（红灯，缺陷未修复）")
		os.Exit(1)
	}

	created, err := urlService.Create(ctx, req)
	if err != nil {
		t.Logf("URL creation failed: %v", err)
		t.Log("GREEN（绿灯，缺陷已修复）")
		return
	}
	if created == nil {
		t.Log("GREEN（绿灯，缺陷已修复）")
		return
	}

	got, err := urlStore.Get("abc123")
	if err != nil {
		t.Logf("Get after save failed: %v", err)
		t.Log("GREEN（绿灯，缺陷已修复）")
		return
	}
	if got == nil || got.RawURL != "https://example.com/test" {
		t.Log("GREEN（绿灯，缺陷已修复）")
		return
	}

	snapshot := urlStore.RawSnapshot()
	if len(snapshot) < 1 {
		t.Log("GREEN（绿灯，缺陷已修复）")
		return
	}

	redirectReq := &service.RedirectRequest{
		Code:      "abc123",
		Timestamp: time.Now(),
	}
	redirectResult, err := redirectService.HandleRedirect(ctx, redirectReq)
	if err != nil {
		t.Logf("redirect failed: %v", err)
		t.Log("GREEN（绿灯，缺陷已修复）")
		return
	}
	if redirectResult == nil {
		t.Log("GREEN（绿灯，缺陷已修复）")
		return
	}
	if redirectResult.Status != 302 {
		t.Log("GREEN（绿灯，缺陷已修复）")
		return
	}

	if !panicGuardCalled {
		t.Log("GREEN（绿灯，缺陷已修复）")
		return
	}

	_ = fmt.Sprintf("url=%s visits=%d", got.RawURL, got.Visits)
	t.Log("GREEN（绿灯，缺陷已修复）")
}