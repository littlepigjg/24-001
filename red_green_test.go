package main

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
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("RED (红灯，缺陷未修复): panic triggered: %v\n", r)
			os.Exit(1)
		}
	}()

	ctx := context.Background()

	cfg := config.Default()
	cfg.Storage.SetPageSize(-1)

	us, err := store.NewURLStore(cfg)
	if err != nil {
		fmt.Printf("RED (红灯，缺陷未修复): NewURLStore error: %v\n", err)
		os.Exit(1)
	}

	ls, err := store.NewAccessLogStore(cfg)
	if err != nil {
		fmt.Printf("RED (红灯，缺陷未修复): NewAccessLogStore error: %v\n", err)
		os.Exit(1)
	}

	if err := ls.Open(ctx); err != nil {
		fmt.Printf("RED (红灯，缺陷未修复): Open error: %v\n", err)
		os.Exit(1)
	}

	urlSvc, err := service.NewURLService(cfg, us)
	if err != nil {
		fmt.Printf("RED (红灯，缺陷未修复): NewURLService error: %v\n", err)
		os.Exit(1)
	}

	req := &model.CreateReq{
		RawURL:     "https://example.com/test",
		CustomCode: "",
		MaxVisits:  -1,
	}

	_, err = urlSvc.Create(ctx, req)
	if err != nil {
		fmt.Printf("RED (红灯，缺陷未修复): Create error: %v\n", err)
		os.Exit(1)
	}

	_, err = urlSvc.List(ctx, 2, -1)
	if err != nil {
		fmt.Printf("RED (红灯，缺陷未修复): List error: %v\n", err)
		os.Exit(1)
	}

	code := "testcode123"
	createReq2 := &model.CreateReq{
		RawURL:     "https://example.com/another",
		CustomCode: code,
		MaxVisits:  -1,
	}
	_, err = urlSvc.Create(ctx, createReq2)
	if err != nil {
		fmt.Printf("RED (红灯，缺陷未修复): Create error: %v\n", err)
		os.Exit(1)
	}

	redirectSvc, err := service.NewRedirectService(us, ls)
	if err != nil {
		fmt.Printf("RED (红灯，缺陷未修复): NewRedirectService error: %v\n", err)
		os.Exit(1)
	}

	redirectReq := &service.RedirectRequest{
		Code:      code,
		Timestamp: time.Now(),
	}

	result, err := redirectSvc.HandleRedirect(ctx, redirectReq)
	if err != nil {
		fmt.Printf("RED (红灯，缺陷未修复): HandleRedirect error: %v\n", err)
		os.Exit(1)
	}
	if result.Status != 302 {
		fmt.Printf("RED (红灯，缺陷未修复): unexpected status %d\n", result.Status)
		os.Exit(1)
	}

	fmt.Println("GREEN (绿灯，缺陷已修复)")
}
