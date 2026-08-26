package codesandbox_test

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/codesandbox/codesandbox/internal/config"
	"github.com/codesandbox/codesandbox/internal/model"
	"github.com/codesandbox/codesandbox/internal/service"
	"github.com/codesandbox/codesandbox/internal/store"
	"github.com/codesandbox/codesandbox/pkg/logger"
)

func TestRedGreen(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "codesandbox-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	log := logger.NewLogger(os.Stdout, logger.LevelError)
	logger.SetGlobal(log)

	cfgMgr := config.NewManager()

	fileStore, err := store.NewFileStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create file store: %v", err)
	}

	historySvc := service.NewHistoryService(fileStore)
	templateSvc := service.NewTemplateService(fileStore)
	langSvc := service.NewLanguageService(nil)

	execSvc := service.NewExecutionService(fileStore, historySvc, templateSvc, langSvc, cfgMgr)

	callCount := 0
	fileStore.SetPanicGuard(func(id string) bool {
		callCount++
		return callCount >= 2
	})

	req := &model.ExecutionRequest{
		Language: "python",
		Code:     "print('hello')",
		Timeout:  5,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var panicRecovered interface{}
	func() {
		defer func() {
			panicRecovered = recover()
		}()
		execSvc.Submit(ctx, req)
	}()

	time.Sleep(100 * time.Millisecond)

	snapshot := fileStore.RawSnapshot()

	zombieCount := 0
	var zombieIDs []string
	for id, exec := range snapshot {
		if exec.Status == model.StatusPending {
			zombieCount++
			zombieIDs = append(zombieIDs, id)
		}
	}

	if panicRecovered != nil {
		if zombieCount > 0 {
			fmt.Println("RED（红灯，缺陷未修复）—— 检测到僵尸执行记录：", zombieIDs)
			t.Errorf("RED（红灯，缺陷未修复）—— 检测到 %d 条僵尸执行记录（status=pending），说明 CreateExecution 和 UpdateExecution 之间无事务保护，panic 后数据不一致", zombieCount)
		} else {
			fmt.Println("GREEN（绿灯，缺陷已修复）—— 虽然发生了 panic，但没有僵尸记录")
		}
	} else {
		if zombieCount > 0 {
			fmt.Println("RED（红灯，缺陷未修复）—— 无 panic 但仍有僵尸记录")
			t.Errorf("RED（红灯，缺陷未修复）—— 检测到 %d 条僵尸执行记录", zombieCount)
		} else {
			fmt.Println("GREEN（绿灯，缺陷已修复）—— 无 panic 且无僵尸记录")
		}
	}
}

func TestConcurrentSubmit(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "codesandbox-concurrent-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	log := logger.NewLogger(os.Stdout, logger.LevelError)
	logger.SetGlobal(log)

	cfgMgr := config.NewManager()

	fileStore, err := store.NewFileStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create file store: %v", err)
	}

	historySvc := service.NewHistoryService(fileStore)
	templateSvc := service.NewTemplateService(fileStore)
	langSvc := service.NewLanguageService(nil)

	execSvc := service.NewExecutionService(fileStore, historySvc, templateSvc, langSvc, cfgMgr)

	const numGoroutines = 5
	errCh := make(chan error, numGoroutines)
	submitted := make(chan string, numGoroutines)

	var wg sync.WaitGroup
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			req := &model.ExecutionRequest{
				Language: "python",
				Code:     fmt.Sprintf("print('test %d')", idx),
				Timeout:  5,
			}
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			exec, err := execSvc.Submit(ctx, req)
			if err != nil {
				errCh <- err
				return
			}
			submitted <- exec.ID
		}(i)
	}

	wg.Wait()
	close(submitted)
	close(errCh)

	submittedIDs := make(map[string]bool)
	for id := range submitted {
		submittedIDs[id] = true
	}

	snapshot := fileStore.RawSnapshot()

	foundCount := 0
	pendingCount := 0
	dataLost := 0
	for id, exec := range snapshot {
		if submittedIDs[id] {
			foundCount++
			if exec.Status == model.StatusPending {
				pendingCount++
			}
		}
	}
	dataLost = len(submittedIDs) - foundCount

	hasError := false
	for err := range errCh {
		if err != nil {
			hasError = true
			t.Logf("concurrent error: %v", err)
		}
	}

	if dataLost > 0 {
		fmt.Printf("RED（红灯，缺陷未修复）—— 并发场景下有 %d 条记录丢失（提交 %d 条，仅找到 %d 条）\n", dataLost, len(submittedIDs), foundCount)
		t.Errorf("RED（红灯，缺陷未修复）—— 并发提交导致数据丢失：提交 %d 条，仅保存 %d 条", len(submittedIDs), foundCount)
	} else if pendingCount > 0 {
		fmt.Printf("RED（红灯，缺陷未修复）—— 并发场景下有 %d 条僵尸记录（status=pending）\n", pendingCount)
		t.Errorf("RED（红灯，缺陷未修复）—— 并发提交后有 %d 条僵尸记录", pendingCount)
	} else if hasError {
		fmt.Printf("RED（红灯，缺陷未修复）—— 并发场景下有错误发生\n")
		t.Errorf("RED（红灯，缺陷未修复）—— 并发提交不稳定，部分请求报错")
	} else {
		fmt.Printf("GREEN（绿灯，缺陷已修复）—— 并发提交稳定，共 %d 条记录全部正确保存并更新\n", foundCount)
	}
}
