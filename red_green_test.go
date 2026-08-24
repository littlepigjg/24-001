package codesandbox

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/codesandbox/codesandbox/internal/config"
	"github.com/codesandbox/codesandbox/internal/model"
	"github.com/codesandbox/codesandbox/internal/service"
	"github.com/codesandbox/codesandbox/internal/store"
)

func TestRedGreen(t *testing.T) {
	cfg := config.Default()
	cfg.Storage.FlushOnWrite(true)
	cfg.Storage.SyncInterval(time.Hour)

	us, err := store.NewURLStore(cfg)
	if err != nil {
		t.Fatalf("NewURLStore error: %v", err)
	}
	ls, err := store.NewAccessLogStore(cfg)
	if err != nil {
		t.Fatalf("NewAccessLogStore error: %v", err)
	}
	if err := ls.Open(context.Background()); err != nil {
		t.Fatalf("Open access log error: %v", err)
	}

	urlSvc, err := service.NewURLService(cfg, us)
	if err != nil {
		t.Fatalf("NewURLService error: %v", err)
	}
	redirSvc, err := service.NewRedirectService(us, ls)
	if err != nil {
		t.Fatalf("NewRedirectService error: %v", err)
	}

	ctxBg := context.Background()
	created, err := urlSvc.Create(ctxBg, &model.CreateReq{
		RawURL: "https://example.com/very/long/path",
	})
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	code := created.Code

	saved, err := us.Get(code)
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if saved.Visits != 0 {
		t.Fatalf("expected 0 visits after create, got %d", saved.Visits)
	}
	if us.BookkeepingCount() != 1 {
		t.Fatalf("expected bookkeeping count 1 after create, got %d", us.BookkeepingCount())
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = redirSvc.HandleRedirect(ctx, &service.RedirectRequest{
		Code:      code,
		Timestamp: time.Now(),
	})
	if err != nil {
		t.Fatalf("HandleRedirect error: %v", err)
	}

	after, err := us.Get(code)
	if err != nil {
		t.Fatalf("Get after redirect error: %v", err)
	}
	visitsAfter := after.Visits
	bookkeepingAfter := us.BookkeepingCount()

	_ = ls.Close()
	_ = us.Close()

	if visitsAfter == 0 && bookkeepingAfter == 1 {
		fmt.Println("GREEN (绿灯，缺陷已修复)")
	} else {
		t.Fatalf("RED (红灯，缺陷未修复) — visitsAfter=%d bookkeepingAfter=%d expected=0/1",
			visitsAfter, bookkeepingAfter)
	}
}
