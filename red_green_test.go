package main

import (
	"context"
	"fmt"
	"strings"
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

	logStore, err := store.NewAccessLogStore(cfg)
	if err != nil {
		t.Fatalf("failed to create access log store: %v", err)
	}

	urlSvc, err := service.NewURLService(cfg, urlStore)
	if err != nil {
		t.Fatalf("failed to create URL service: %v", err)
	}

	redirectSvc, err := service.NewRedirectService(urlStore, logStore)
	if err != nil {
		t.Fatalf("failed to create redirect service: %v", err)
	}

	ctx := context.Background()

	testsPassed := true
	totalTests := 0
	passedTests := 0

	// Test 1: Create with duplicate custom code should preserve "code already exists"
	t.Run("create_duplicate_code_preserves_error", func(t *testing.T) {
		totalTests++
		req1 := &model.CreateReq{
			RawURL:     "https://example.com/page1",
			CustomCode: "duptest",
			MaxVisits:  100,
		}
		_, err1 := urlSvc.Create(ctx, req1)
		if err1 != nil {
			t.Fatalf("unexpected error on first create: %v", err1)
		}

		req2 := &model.CreateReq{
			RawURL:     "https://example.com/page2",
			CustomCode: "duptest",
			MaxVisits:  100,
		}
		_, err2 := urlSvc.Create(ctx, req2)
		if err2 == nil {
			t.Errorf("expected error for duplicate code, got nil")
			testsPassed = false
			return
		}

		errMsg := err2.Error()
		if !strings.Contains(errMsg, "code already exists") && !strings.Contains(errMsg, "already exists") {
			t.Errorf("error message lost specific info, got: %s", errMsg)
			testsPassed = false
		} else {
			passedTests++
		}
	})

	// Test 2: Create with invalid URL should preserve validation error
	t.Run("create_invalid_url_preserves_error", func(t *testing.T) {
		totalTests++
		req := &model.CreateReq{
			RawURL:     "not-a-valid-url",
			CustomCode: "invurl",
			MaxVisits:  10,
		}
		_, err := urlSvc.Create(ctx, req)
		if err == nil {
			t.Errorf("expected error for invalid URL, got nil")
			testsPassed = false
			return
		}

		errMsg := err.Error()
		if !strings.Contains(errMsg, "invalid") && !strings.Contains(errMsg, "URL") && !strings.Contains(errMsg, "scheme") {
			t.Errorf("error message lost validation info, got: %s", errMsg)
			testsPassed = false
		} else {
			passedTests++
		}
	})

	// Test 3: Redirect with non-existent code should preserve "not found"
	t.Run("redirect_nonexistent_code_preserves_error", func(t *testing.T) {
		totalTests++
		req := &service.RedirectRequest{
			Code:      "nonexist123",
			Timestamp: time.Now(),
		}
		_, err := redirectSvc.HandleRedirect(ctx, req)
		if err == nil {
			t.Errorf("expected error for non-existent code, got nil")
			testsPassed = false
			return
		}

		errMsg := err.Error()
		if !strings.Contains(errMsg, "not found") {
			t.Errorf("error message lost 'not found' info, got: %s", errMsg)
			testsPassed = false
		} else {
			passedTests++
		}
	})

	// Test 4: Get with non-existent code should preserve "not found"
	t.Run("get_nonexistent_code_preserves_error", func(t *testing.T) {
		totalTests++
		_, err := urlSvc.Get(ctx, "nofoundcode")
		if err == nil {
			t.Errorf("expected error for non-existent code, got nil")
			testsPassed = false
			return
		}

		errMsg := err.Error()
		if !strings.Contains(errMsg, "not found") {
			t.Errorf("error message lost 'not found' info in Get, got: %s", errMsg)
			testsPassed = false
		} else {
			passedTests++
		}
	})

	// Test 5: Redirect with disabled code should preserve "disabled"
	t.Run("redirect_disabled_code_preserves_error", func(t *testing.T) {
		totalTests++
		createReq := &model.CreateReq{
			RawURL:     "https://example.com/disabled",
			CustomCode: "discode",
			MaxVisits:  10,
		}
		_, err := urlSvc.Create(ctx, createReq)
		if err != nil {
			t.Fatalf("failed to create short URL: %v", err)
		}

		err = urlSvc.Delete(ctx, "discode")
		if err != nil {
			t.Fatalf("failed to disable short URL: %v", err)
		}

		req := &service.RedirectRequest{
			Code:      "discode",
			Timestamp: time.Now(),
		}
		_, err = redirectSvc.HandleRedirect(ctx, req)
		if err == nil {
			t.Errorf("expected error for disabled code, got nil")
			testsPassed = false
			return
		}

		errMsg := err.Error()
		if !strings.Contains(errMsg, "disabled") {
			t.Errorf("error message lost 'disabled' info, got: %s", errMsg)
			testsPassed = false
		} else {
			passedTests++
		}
	})

	// Test 6: Redirect with max visits reached should preserve "expired"
	t.Run("redirect_maxvisits_reached_preserves_error", func(t *testing.T) {
		totalTests++
		createReq := &model.CreateReq{
			RawURL:     "https://example.com/visited",
			CustomCode: "maxvis",
			MaxVisits:  1,
		}
		_, err := urlSvc.Create(ctx, createReq)
		if err != nil {
			t.Fatalf("failed to create short URL: %v", err)
		}

		req1 := &service.RedirectRequest{
			Code:      "maxvis",
			Timestamp: time.Now(),
		}
		_, err1 := redirectSvc.HandleRedirect(ctx, req1)
		if err1 != nil {
			t.Fatalf("first redirect should succeed: %v", err1)
		}

		req2 := &service.RedirectRequest{
			Code:      "maxvis",
			Timestamp: time.Now(),
		}
		_, err2 := redirectSvc.HandleRedirect(ctx, req2)
		if err2 == nil {
			t.Errorf("expected error for expired/maxvisits, got nil")
			testsPassed = false
			return
		}

		errMsg := err2.Error()
		if !strings.Contains(errMsg, "expired") && !strings.Contains(errMsg, "max") && !strings.Contains(errMsg, "visits") {
			t.Errorf("error message lost 'expired/maxvisits' info, got: %s", errMsg)
			testsPassed = false
		} else {
			passedTests++
		}
	})

	fmt.Printf("\n========== TEST SUMMARY ==========\n")
	fmt.Printf("Passed: %d / %d\n", passedTests, totalTests)

	if testsPassed {
		fmt.Printf("Result: GREEN (绿灯，缺陷已修复)\n")
	} else {
		fmt.Printf("Result: RED (红灯，缺陷未修复)\n")
	}
	fmt.Printf("==================================\n\n")

	if !testsPassed {
		t.Fail()
	}
}
