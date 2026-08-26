package main

import (
	"context"
	"fmt"
	"testing"

	"github.com/codesandbox/codesandbox/internal/config"
	"github.com/codesandbox/codesandbox/internal/model"
	"github.com/codesandbox/codesandbox/internal/service"
	"github.com/codesandbox/codesandbox/internal/store"
)

func TestRedGreen(t *testing.T) {
	fmt.Println("========================================")
	fmt.Println("开始缺陷检测测试")
	fmt.Println("========================================")

	// Test 1: 验证空代码请求的Validate方法
	t.Run("空代码请求验证", func(t *testing.T) {
		req := &model.ExecutionRequest{
			Language: "python",
			Code:     "",
		}

		errors := req.Validate()

		fmt.Printf("测试1: 空代码Validate - 错误数量: %d\n", len(errors))
		for k, v := range errors {
			fmt.Printf("  字段: %s, 错误: %s\n", k, v)
		}

		if len(errors) == 0 {
			fmt.Println("  结果: 空代码未被校验拦截 - 有缺陷!")
		} else {
			fmt.Println("  结果: 空代码被正确拦截")
		}
	})

	// Test 2: 验证空代码通过ExecutionService.Execute
	t.Run("空代码执行服务验证", func(t *testing.T) {
		cfgMgr := config.NewManager()
		dataStore := store.NewMemoryStore()
		langSvc := service.NewLanguageService(nil)
		historySvc := service.NewHistoryService(dataStore)
		templateSvc := service.NewTemplateService(dataStore)
		execSvc := service.NewExecutionService(dataStore, historySvc, templateSvc, langSvc, cfgMgr)

		ctx := context.Background()
		req := &model.ExecutionRequest{
			Language: "python",
			Code:     "",
		}

		exec, err := execSvc.Execute(ctx, req)

		fmt.Printf("测试2: 空代码Execute - err: %v\n", err)
		if exec != nil {
			fmt.Printf("  返回执行ID: %s, 状态: %s\n", exec.ID, exec.Status)
			if exec.Result != nil {
				fmt.Printf("  结果: stdout=%q, exitCode=%d\n", exec.Result.Stdout, exec.Result.ExitCode)
			}
		}

		if err == nil && exec != nil && exec.Status == model.StatusCompleted {
			fmt.Println("  结果: 空代码执行成功完成 - 有缺陷!")
		} else if err != nil {
			fmt.Println("  结果: 空代码被正确拒绝")
		}
	})

	// 综合判定
	hasDefect := false

	req := &model.ExecutionRequest{
		Language: "python",
		Code:     "",
	}
	errors := req.Validate()
	if len(errors) == 0 {
		hasDefect = true
	}

	cfgMgr := config.NewManager()
	dataStore := store.NewMemoryStore()
	langSvc := service.NewLanguageService(nil)
	historySvc := service.NewHistoryService(dataStore)
	templateSvc := service.NewTemplateService(dataStore)
	execSvc := service.NewExecutionService(dataStore, historySvc, templateSvc, langSvc, cfgMgr)

	ctx := context.Background()
	exec, err := execSvc.Execute(ctx, req)

	if err == nil && exec != nil && exec.Status == model.StatusCompleted {
		hasDefect = true
	}

	fmt.Println()
	fmt.Println("========================================")
	if hasDefect {
		fmt.Println("RED（红灯，缺陷未修复）")
		fmt.Println("缺陷: 空代码请求绕过校验并成功执行")
		fmt.Println("========================================")
		t.Fatal("RED（红灯，缺陷未修复）: 空代码请求绕过校验并成功执行")
	} else {
		fmt.Println("GREEN（绿灯，缺陷已修复）")
		fmt.Println("========================================")
	}
}
