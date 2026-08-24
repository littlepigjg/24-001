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
	"github.com/codesandbox/codesandbox/pkg/logger"
	"github.com/codesandbox/codesandbox/pkg/process"
)

func TestRedGreen(t *testing.T) {
	log := logger.NewLogger(os.Stdout, logger.LevelWarn)
	logger.SetGlobal(log)

	memStore := store.NewMemoryStore()

	cfgMgr := config.NewManager()
	cfgMgr.UpdateConfig(func(cfg *model.AppConfig) {
		cfg.MaxConcurrent = 1
		cfg.StorageType = "memory"
		cfg.DefaultTimeout = 30
		cfg.MaxTimeout = 300
	})

	histSvc := service.NewHistoryService(memStore)
	tmplSvc := service.NewTemplateService(memStore)
	executor := process.NewExecutor()
	langSvc := service.NewLanguageService(executor)

	execSvc := service.NewExecutionService(memStore, histSvc, tmplSvc, langSvc, cfgMgr)

	ctx := context.Background()

	// Use a command that takes ~500ms to complete
	req := &model.ExecutionRequest{
		Language: "shell",
		Code:     "echo 'start' && sleep 0.5 && echo 'end'",
		Timeout:  30,
	}

	exec, err := execSvc.Execute(ctx, req)
	if err != nil {
		t.Fatalf("Failed to submit execution: %v", err)
	}

	t.Log("Execution submitted, waiting for it to start running...")
	time.Sleep(200 * time.Millisecond)

	// Shutdown with a 2-second timeout - enough for the execution to complete
	// if we wait first (the fix), but the bug kills processes first
	t.Log("Calling Shutdown (kills processes first, then waits for goroutines)...")
	shutdownErr := execSvc.Shutdown(2 * time.Second)
	if shutdownErr != nil {
		t.Logf("Shutdown returned error: %v", shutdownErr)
	}

	result, err := execSvc.GetResult(exec.ID)
	if err != nil {
		t.Fatalf("Failed to get execution result: %v", err)
	}

	t.Logf("Execution status: %s", result.Status)
	t.Logf("Execution error message: '%s'", result.ErrorMessage)
	t.Logf("Execution result is nil: %v", result.Result == nil)
	if result.Result != nil {
		t.Logf("Execution result stdout: '%s'", result.Result.Stdout)
		t.Logf("Execution result exit code: %d", result.Result.ExitCode)
	}

	executionCompleted := result.Status != model.StatusPending &&
		result.Status != model.StatusRunning

	if executionCompleted && result.Result != nil {
		fmt.Println("GREEN (绿灯，缺陷已修复)")
		t.Log("Execution result was properly saved after shutdown - DEFECT FIXED")
	} else {
		fmt.Println("RED (红灯，缺陷未修复)")
		t.Log("Execution result was NOT properly saved after shutdown - DEFECT EXISTS")
		if result.Status == model.StatusRunning || result.Status == model.StatusPending {
			t.Logf("Execution stuck in %s state - data loss confirmed", result.Status)
		}
		t.Fail()
	}
}

func TestRedGreenWithCancelContext(t *testing.T) {
	log := logger.NewLogger(os.Stdout, logger.LevelWarn)
	logger.SetGlobal(log)

	memStore := store.NewMemoryStore()

	cfgMgr := config.NewManager()
	cfgMgr.UpdateConfig(func(cfg *model.AppConfig) {
		cfg.MaxConcurrent = 1
		cfg.StorageType = "memory"
		cfg.DefaultTimeout = 30
		cfg.MaxTimeout = 300
	})

	histSvc := service.NewHistoryService(memStore)
	tmplSvc := service.NewTemplateService(memStore)
	executor := process.NewExecutor()
	langSvc := service.NewLanguageService(executor)

	execSvc := service.NewExecutionService(memStore, histSvc, tmplSvc, langSvc, cfgMgr)

	ctx := context.Background()

	// Use a command that takes ~500ms to complete
	req := &model.ExecutionRequest{
		Language: "shell",
		Code:     "echo 'start' && sleep 0.5 && echo 'end'",
		Timeout:  30,
	}

	exec, err := execSvc.Execute(ctx, req)
	if err != nil {
		t.Fatalf("Failed to submit execution: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	// Shutdown with a 2-second timeout
	completed := execSvc.ShutdownWithCancelContext(2 * time.Second)
	if !completed {
		t.Log("Shutdown did not complete all goroutines within timeout")
	}

	result, err := execSvc.GetResult(exec.ID)
	if err != nil {
		t.Fatalf("Failed to get execution result: %v", err)
	}

	executionCompleted := result.Status != model.StatusPending &&
		result.Status != model.StatusRunning

	if executionCompleted && result.Result != nil {
		fmt.Println("GREEN (绿灯，缺陷已修复)")
		t.Log("Execution result was properly saved after shutdown")
	} else {
		fmt.Println("RED (红灯，缺陷未修复)")
		t.Log("Execution result was NOT properly saved after shutdown")
		if result.Status == model.StatusRunning || result.Status == model.StatusPending {
			t.Logf("Execution stuck in %s state - data loss confirmed", result.Status)
		}
		t.Fail()
	}
}
