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
	"github.com/codesandbox/codesandbox/pkg/logger"
	"github.com/codesandbox/codesandbox/pkg/process"
)

func TestRedGreen(t *testing.T) {
	logger.GetGlobal().SetLevel(logger.LevelError)

	memStore := store.NewMemoryStore()
	cfgMgr := config.NewManager()
	historySvc := service.NewHistoryService(memStore)
	templateSvc := service.NewTemplateService(memStore)
	executor := process.NewExecutor()
	langSvc := service.NewLanguageService(executor)
	execSvc := service.NewExecutionService(memStore, historySvc, templateSvc, langSvc, cfgMgr)

	t.Run("context_cancellation_propagates", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		req := &model.ExecutionRequest{
			Language: "shell",
			Code:     "sleep 5",
			Timeout:  30,
		}

		exec, err := execSvc.Execute(ctx, req)
		if err != nil {
			t.Fatalf("failed to execute: %v", err)
		}

		time.Sleep(100 * time.Millisecond)
		cancel()

		result, err := execSvc.WaitForResult(exec.ID, 3*time.Second)
		if err != nil {
			t.Errorf("RED (红灯，缺陷未修复) - 等待执行结果超时，执行未被原始上下文取消: %v", err)
			return
		}

		if result.Status == model.StatusCanceled || result.Status == model.StatusTimedOut {
			t.Log("GREEN (绿灯，缺陷已修复) - 原始上下文取消正确传播，执行被终止")
		} else {
			t.Errorf("RED (红灯，缺陷未修复) - 执行未被原始上下文取消终止，状态为: %s (期望 canceled 或 timed_out)", result.Status)
		}
	})

	t.Run("deadline_propagation", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		req := &model.ExecutionRequest{
			Language: "shell",
			Code:     "sleep 5",
			Timeout:  30,
		}

		exec, err := execSvc.Execute(ctx, req)
		if err != nil {
			t.Fatalf("failed to execute: %v", err)
		}

		result, err := execSvc.WaitForResult(exec.ID, 3*time.Second)
		if err != nil {
			t.Errorf("RED (红灯，缺陷未修复) - 等待执行结果超时，上下文截止时间未传播: %v", err)
			return
		}

		if result.Status == model.StatusTimedOut || result.Status == model.StatusCanceled {
			t.Log("GREEN (绿灯，缺陷已修复) - 上下文截止时间正确传播，执行被按时终止")
		} else {
			t.Errorf("RED (红灯，缺陷未修复) - 执行未在上下文截止时间后终止，状态为: %s (期望 timed_out 或 canceled)", result.Status)
		}
	})

	t.Run("concurrent_snapshot_consistency", func(t *testing.T) {
		ctx := context.Background()

		req := &model.ExecutionRequest{
			Language: "shell",
			Code:     "sleep 1",
			Timeout:  10,
		}

		exec, err := execSvc.Execute(ctx, req)
		if err != nil {
			t.Fatalf("failed to execute: %v", err)
		}

		time.Sleep(50 * time.Millisecond)

		got, err := execSvc.GetResult(exec.ID)
		if err != nil {
			t.Fatalf("failed to get execution: %v", err)
		}

		snapshot, err := execSvc.GetResultWithSnapshot(exec.ID)
		if err != nil {
			t.Fatalf("failed to get snapshot: %v", err)
		}

		firstStatus := got.Status
		snapshotStatus := snapshot.Status

		time.Sleep(2 * time.Second)

		gotFinal, _ := execSvc.GetResult(exec.ID)

		if gotFinal.Status != snapshotStatus {
			t.Errorf("RED (红灯，缺陷未修复) - GetResult 返回的指针在执行完成后状态被并发修改: 原始快照=%s, 当前指针=%s",
				snapshotStatus, gotFinal.Status)
		} else {
			fmt.Printf("  初始状态: got=%s, snapshot=%s\n", firstStatus, snapshotStatus)
			fmt.Printf("  最终状态: got=%s, snapshot=%s\n", gotFinal.Status, snapshotStatus)
			t.Log("GREEN (绿灯，缺陷已修复) - GetResult 返回的指针与快照状态一致")
		}
	})
}