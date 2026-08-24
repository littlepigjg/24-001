package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/codesandbox/codesandbox/config"
	"github.com/codesandbox/codesandbox/model"
	"github.com/codesandbox/codesandbox/service"
	"github.com/codesandbox/codesandbox/store"
)

func countOpenFDs() int {
	entries, err := os.ReadDir(fmt.Sprintf("/proc/%d/fd", os.Getpid()))
	if err != nil {
		return -1
	}
	return len(entries)
}

func TestRedGreen(t *testing.T) {
	runtime.GC()
	beforeFDs := countOpenFDs()
	t.Logf("FD count before test: %d", beforeFDs)

	cfg := config.Default()
	cfg.Storage.URLFilePath(filepath.Join(os.TempDir(), fmt.Sprintf("shurl_test_%d.json", os.Getpid())))
	cfg.Storage.LogFilePath(filepath.Join(os.TempDir(), fmt.Sprintf("shurl_log_%d.json", os.Getpid())))

	urlStore, err := store.NewURLStore(cfg)
	if err != nil {
		t.Fatalf("failed to create URL store: %v", err)
	}

	svc, err := service.NewURLService(cfg, urlStore)
	if err != nil {
		t.Fatalf("failed to create URL service: %v", err)
	}

	const numURLs = 500

	for i := 0; i < numURLs; i++ {
		rawURL := fmt.Sprintf("http://example.com/deep/nested/path/resource/%d?param=value&extra=stuff", i)
		req := &model.CreateReq{
			RawURL: rawURL,
		}
		_, createErr := svc.Create(context.Background(), req)
		if createErr != nil {
			fmt.Printf("RED（红灯，缺陷未修复）\n创建第 %d 个短链接时出错: %v\n", i, createErr)
			urlStore.Close()
			t.FailNow()
		}
	}

	afterCreateFDs := countOpenFDs()
	t.Logf("FD count after creating %d URLs: %d (delta: %d)", numURLs, afterCreateFDs, afterCreateFDs-beforeFDs)

	if afterCreateFDs-beforeFDs > numURLs/4 {
		fmt.Printf("RED（红灯，缺陷未修复）\n文件描述符泄露检测: 基线 %d, 创建后 %d, 泄露增量 %d\n",
			beforeFDs, afterCreateFDs, afterCreateFDs-beforeFDs)
		urlStore.Close()
		t.FailNow()
	}

	snapshot := urlStore.RawSnapshot()
	if len(snapshot) != numURLs {
		fmt.Printf("RED（红灯，缺陷未修复）\n数据快照不完整: 期望 %d, 实际 %d\n", numURLs, len(snapshot))
		urlStore.Close()
		t.FailNow()
	}

	redirectSvc, redirectErr := service.NewRedirectService(urlStore, nil)
	if redirectErr != nil {
		t.Fatalf("failed to create redirect service: %v", redirectErr)
	}

	var firstCode string
	var firstShortURL *model.ShortURL
	for code, u := range snapshot {
		firstCode = code
		uCopy := u
		firstShortURL = &uCopy
		break
	}

	if firstCode == "" {
		fmt.Printf("RED（红灯，缺陷未修复）\n快照为空，无法测试重定向\n")
		urlStore.Close()
		t.FailNow()
	}

	redirectReq := &service.RedirectRequest{
		Code:      firstCode,
		Timestamp: time.Now(),
	}

	redirectResult, handleErr := redirectSvc.HandleRedirect(context.Background(), redirectReq)
	if handleErr != nil {
		fmt.Printf("RED（红灯，缺陷未修复）\n重定向处理出错: %v\n", handleErr)
		urlStore.Close()
		t.FailNow()
	}

	expectedRawURL := firstShortURL.RawURL
	if redirectResult.RawURL != expectedRawURL {
		fmt.Printf("RED（红灯，缺陷未修复）\n重定向返回的 URL 不匹配: 期望 %s, 实际 %s\n",
			expectedRawURL, redirectResult.RawURL)
		urlStore.Close()
		t.FailNow()
	}

	if redirectResult.Status != 302 {
		fmt.Printf("RED（红灯，缺陷未修复）\n重定向状态码错误: 期望 302, 实际 %d\n", redirectResult.Status)
		urlStore.Close()
		t.FailNow()
	}

	secondReq := &service.RedirectRequest{
		Code:      "nonexistent_code_xyz",
		Timestamp: time.Now(),
	}
	secondResult, secondErr := redirectSvc.HandleRedirect(context.Background(), secondReq)
	if secondErr != nil {
		fmt.Printf("RED（红灯，缺陷未修复）\n查询不存在的代码出错: %v\n", secondErr)
		urlStore.Close()
		t.FailNow()
	}
	if secondResult.Status != 404 {
		fmt.Printf("RED（红灯，缺陷未修复）\n不存在的代码应返回 404，实际: %d\n", secondResult.Status)
		urlStore.Close()
		t.FailNow()
	}

	runtime.GC()
	runtime.GC()
	afterGCFDs := countOpenFDs()
	t.Logf("FD count after GC: %d (delta from baseline: %d)", afterGCFDs, afterGCFDs-beforeFDs)

	if closeErr := urlStore.Close(); closeErr != nil {
		t.Logf("warning: error closing store: %v", closeErr)
	}

	fmt.Printf("GREEN（绿灯，缺陷已修复）\n成功创建 %d 个短链接并完成重定向测试\n创建后 FD 增量: %d\n",
		numURLs, afterCreateFDs-beforeFDs)
}
