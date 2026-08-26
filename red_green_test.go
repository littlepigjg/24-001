package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/codesandbox/codesandbox/config"
	"github.com/codesandbox/codesandbox/internal/model"
	"github.com/codesandbox/codesandbox/internal/service"
	"github.com/codesandbox/codesandbox/internal/store"
)

func TestRedGreen(t *testing.T) {
	fmt.Println("=== TestRedGreen ===")
	fmt.Println("Testing HTTPS configuration defect...")

	cfg := config.Default()
	fmt.Printf("Default config - HTTPS enabled: %v, CertFile: '%s', KeyFile: '%s'\n",
		cfg.Storage.HTTPSEnabled(),
		cfg.Storage.HTTPSCertFile(),
		cfg.Storage.HTTPSKeyFile())

	if !cfg.Storage.HTTPSEnabled() {
		fmt.Println("HTTPS not enabled - configuration is correct")
		return
	}

	urlStore, err := store.NewURLStore(cfg)
	if err != nil {
		fmt.Printf("NewURLStore failed: %v\n", err)
		return
	}

	var panicTriggered bool
	urlStore.SetPanicGuard(func(code, rawURL string) bool {
		panicTriggered = true
		fmt.Printf("PanicGuard triggered for code='%s', rawURL='%s'\n", code, rawURL)
		return true
	})

	ctx := context.Background()

	func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("Recovered from panic: %v\n", r)
				panicTriggered = true
			}
		}()
		_ = urlStore.Load(ctx)
	}()

	fmt.Printf("After Load - Panic triggered: %v\n", panicTriggered)

	shortURL := &model.ShortURL{
		Code:      "test123",
		RawURL:    "https://example.com",
		CreatedAt: time.Now(),
		Visits:    0,
		Custom:    false,
		Disabled:  false,
	}

	func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("Recovered from Save panic: %v\n", r)
				panicTriggered = true
			}
		}()
		_ = urlStore.Save(shortURL, false)
	}()

	fmt.Printf("After Save - Panic triggered: %v\n", panicTriggered)

	accessLogStore, _ := store.NewAccessLogStore(cfg)
	urlSvc, svcErr := service.NewURLService(cfg, urlStore)
	if svcErr != nil {
		fmt.Printf("NewURLService failed: %v\n", svcErr)
	}

	_, _ = service.NewRedirectService(urlStore, accessLogStore)

	createReq := &model.CreateReq{
		RawURL:    "https://example.com/test",
		CustomCode: "",
		MaxVisits:  0,
	}

	func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("Recovered from Create panic: %v\n", r)
				panicTriggered = true
			}
		}()
		_, _ = urlSvc.Create(ctx, createReq)
	}()

	fmt.Printf("Final - Panic triggered: %v\n", panicTriggered)

	_ = urlStore.Close()

	if panicTriggered {
		fmt.Println("RESULT: RED (红灯，缺陷未修复)")
		t.Log("RESULT: RED (红灯，缺陷未修复)")
	} else {
		fmt.Println("RESULT: GREEN (绿灯，缺陷已修复)")
		t.Log("RESULT: GREEN (绿灯，缺陷已修复)")
	}
}
