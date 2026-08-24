package main

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/codesandbox/codesandbox/internal/config"
	"github.com/codesandbox/codesandbox/internal/model"
	"github.com/codesandbox/codesandbox/internal/service"
	"github.com/codesandbox/codesandbox/internal/store"
)

func TestRedGreen(t *testing.T) {
	cfg := config.Default()
	us, err := store.NewURLStore(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer us.Close()

	ls, err := store.NewAccessLogStore(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer ls.Close()

	us.SetPanicGuard(func(code, rawURL string) bool {
		return false
	})

	usvc, err := service.NewURLService(cfg, us)
	if err != nil {
		t.Fatal(err)
	}

	rsvc, err := service.NewRedirectService(us, ls)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	createDone := make(chan error, 1)
	go func() {
		req := &model.CreateReq{
			RawURL:     "https://example.com/test-page",
			CustomCode: "abcde001",
			MaxVisits:  500,
		}
		u, err := usvc.Create(ctx, req)
		if err != nil {
			createDone <- err
			return
		}
		if u == nil {
			createDone <- fmt.Errorf("create returned nil")
			return
		}
		if u.Code != "abcde001" {
			createDone <- fmt.Errorf("unexpected code: %s", u.Code)
			return
		}
		if u.RawURL != "https://example.com/test-page" {
			createDone <- fmt.Errorf("unexpected raw_url: %s", u.RawURL)
			return
		}
		createDone <- nil
	}()

	select {
	case err := <-createDone:
		if err != nil {
			fmt.Println("RED（红灯，缺陷未修复）")
			t.Fatalf("Create failed: %v", err)
		}
	case <-time.After(500 * time.Millisecond):
		fmt.Println("RED（红灯，缺陷未修复）")
		t.Fatal("Create timed out - possible deadlock detected")
	}

	create2Done := make(chan error, 1)
	go func() {
		req2 := &model.CreateReq{
			RawURL:     "https://example.org/another",
			CustomCode: "abcde002",
			MaxVisits:  300,
		}
		u2, err := usvc.Create(ctx, req2)
		if err != nil {
			create2Done <- err
			return
		}
		if u2 == nil {
			create2Done <- fmt.Errorf("create2 returned nil")
			return
		}
		create2Done <- nil
	}()

	select {
	case err := <-create2Done:
		if err != nil {
			fmt.Println("RED（红灯，缺陷未修复）")
			t.Fatalf("Second Create failed: %v", err)
		}
	case <-time.After(500 * time.Millisecond):
		fmt.Println("RED（红灯，缺陷未修复）")
		t.Fatal("Second Create timed out - possible deadlock detected")
	}

	redirectDone := make(chan error, 1)
	go func() {
		result, err := rsvc.HandleRedirect(ctx, &service.RedirectRequest{
			Code:      "abcde001",
			Timestamp: time.Now().Unix(),
		})
		if err != nil {
			redirectDone <- err
			return
		}
		if result == nil {
			redirectDone <- fmt.Errorf("redirect returned nil")
			return
		}
		if result.RawURL != "https://example.com/test-page" {
			redirectDone <- fmt.Errorf("unexpected redirect url: %s", result.RawURL)
			return
		}
		if result.Status != 302 {
			redirectDone <- fmt.Errorf("unexpected status: %d", result.Status)
			return
		}
		redirectDone <- nil
	}()

	select {
	case err := <-redirectDone:
		if err != nil {
			fmt.Println("RED（红灯，缺陷未修复）")
			t.Fatalf("HandleRedirect failed: %v", err)
		}
	case <-time.After(500 * time.Millisecond):
		fmt.Println("RED（红灯，缺陷未修复）")
		t.Fatal("HandleRedirect timed out - possible deadlock detected")
	}

	var wg sync.WaitGroup
	concurrentDone := make(chan struct{}, 1)
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			code := fmt.Sprintf("concurrent%03d", idx)
			req := &model.CreateReq{
				RawURL:     fmt.Sprintf("https://example.com/concurr/%d", idx),
				CustomCode: code,
				MaxVisits:  100,
			}
			_, err := usvc.Create(ctx, req)
			if err != nil {
				t.Errorf("Concurrent create %d failed: %v", idx, err)
			}
		}(i)
	}

	go func() {
		wg.Wait()
		close(concurrentDone)
	}()

	select {
	case <-concurrentDone:
	case <-time.After(1 * time.Second):
		fmt.Println("RED（红灯，缺陷未修复）")
		t.Fatal("Concurrent operations timed out - deadlock under concurrency")
	}

	if t.Failed() {
		return
	}

	fmt.Println("GREEN（绿灯，缺陷已修复）")
}