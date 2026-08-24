package main

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
	cfg.BasePath = t.TempDir()

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

	urlSvc, err := service.NewURLService(cfg, urlStore)
	if err != nil {
		t.Fatalf("failed to create URL service: %v", err)
	}

	redirectSvc, err := service.NewRedirectService(urlStore, logStore)
	if err != nil {
		t.Fatalf("failed to create redirect service: %v", err)
	}

	req := &model.CreateReq{
		RawURL:    "https://example.com/very/long/url/that/represents/a/valid/redirect/target",
		CustomCode: "",
		MaxVisits:  100,
	}

	u, err := urlSvc.Create(ctx, req)
	if err != nil {
		fmt.Println("RED")
		t.Fatal(err)
	}

	visitReq := &service.RedirectRequest{
		Code:      u.Code,
		Timestamp: time.Now(),
	}

	func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Println("RED")
				t.FailNow()
			}
		}()

		_, err = redirectSvc.HandleRedirect(ctx, visitReq)
		if err != nil {
			fmt.Println("RED")
			t.FailNow()
		}

		_, err = urlSvc.Get(ctx, u.Code)
		if err != nil {
			fmt.Println("RED")
			t.FailNow()
		}

		stats, err := redirectSvc.GetStats(ctx)
		if err != nil {
			fmt.Println("RED")
			t.FailNow()
		}

		totalURLs, ok := stats["total_urls"].(int)
		if !ok || totalURLs < 1 {
			fmt.Println("RED")
			t.FailNow()
		}

		fmt.Println("GREEN")
	}()
}
