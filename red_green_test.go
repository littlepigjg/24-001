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
	"github.com/codesandbox/codesandbox/pkg/logger"
)

func TestRedGreen(t *testing.T) {
	log := logger.NewLogger(os.Stdout, logger.LevelError)
	logger.SetGlobal(log)

	cfgMgr := config.GetGlobal()
	cfg := cfgMgr.GetConfig()
	cfg.MaxTimeout = 30
	cfgMgr.SetConfig(cfg)

	dataStore := store.NewMemoryStore()
	execSvc := service.NewExecutionService(dataStore, nil, nil, nil, cfgMgr)

	allPassed := true

	cfg2 := cfgMgr.GetConfig()
	if cfg2.MaxTimeout != 30 {
		fmt.Printf("FAIL: Test 1 - Config MaxTimeout corrupted: expected 30, got %d\n", cfg2.MaxTimeout)
		allPassed = false
	} else {
		fmt.Println("PASS: Test 1 - Config MaxTimeout correctly preserved")
	}

	req := &model.ExecutionRequest{
		Language: "shell",
		Code:     "echo hello",
		Timeout:  60,
	}
	exec, err := execSvc.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("Failed to execute: %v", err)
	}
	if exec.Timeout != 30 {
		fmt.Printf("FAIL: Test 2 - Timeout not clamped: expected 30, got %d\n", exec.Timeout)
		allPassed = false
	} else {
		fmt.Println("PASS: Test 2 - Timeout correctly clamped to MaxTimeout")
	}

	req2 := &model.ExecutionRequest{
		Language: "shell",
		Code:     "exit 1",
		Timeout:  5,
	}
	exec2, err := execSvc.Execute(context.Background(), req2)
	if err != nil {
		t.Fatalf("Failed to execute: %v", err)
	}

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		got, err := dataStore.GetExecution(exec2.ID)
		if err == nil && got.Status != model.StatusPending && got.Status != model.StatusRunning {
			if got.Status == model.StatusFailed {
				fmt.Println("PASS: Test 3 - Error correctly mapped to StatusFailed")
			} else {
				fmt.Printf("FAIL: Test 3 - Error mapped to wrong status: expected failed, got %s\n", got.Status)
				allPassed = false
			}
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	req3 := &model.ExecutionRequest{
		Language: "shell",
		Code:     "sleep 3",
		Timeout:  10,
	}
	exec3, err := execSvc.Execute(ctx, req3)
	if err != nil {
		t.Fatalf("Failed to execute: %v", err)
	}

	deadline3 := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline3) {
		got, err := dataStore.GetExecution(exec3.ID)
		if err == nil && got.Status != model.StatusPending && got.Status != model.StatusRunning {
			if got.Status == model.StatusTimedOut || got.Status == model.StatusCanceled {
				fmt.Println("PASS: Test 4 - Context timeout propagated correctly")
			} else {
				fmt.Printf("FAIL: Test 4 - Execution not cancelled by context: expected timed_out/canceled, got %s\n", got.Status)
				allPassed = false
			}
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	if allPassed {
		fmt.Println("GREEN (绿灯，缺陷已修复)")
	} else {
		fmt.Println("RED (红灯，缺陷未修复)")
		t.Fail()
	}
}