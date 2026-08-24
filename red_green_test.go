package main

import (
	"fmt"
	"testing"

	"github.com/codesandbox/codesandbox/internal/config"
	"github.com/codesandbox/codesandbox/internal/model"
	"github.com/codesandbox/codesandbox/internal/service"
	"github.com/codesandbox/codesandbox/internal/store"
	"github.com/codesandbox/codesandbox/pkg/process"
)

func TestRedGreen(t *testing.T) {
	s := store.NewMemoryStore()
	s.SetPanicGuard(func(id string) bool {
		return id == "missing-exec-id"
	})

	cfgMgr := config.NewManager()
	historySvc := service.NewHistoryService(s)
	templateSvc := service.NewTemplateService(s)
	execSvc := process.NewExecutor()
	langSvc := service.NewLanguageService(execSvc)
	svc := service.NewExecutionService(s, historySvc, templateSvc, langSvc, cfgMgr)

	t.Run("Cancel non-existent execution", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				fmt.Println("RED (红灯，缺陷未修复)")
				t.FailNow()
			}
		}()

		err := svc.Cancel("missing-exec-id")
		if err != nil {
			fmt.Println("GREEN (绿灯，缺陷已修复)")
		} else {
			fmt.Println("RED (红灯，缺陷未修复)")
			t.FailNow()
		}
	})

	t.Run("GetResult non-existent execution", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				fmt.Println("RED (红灯，缺陷未修复)")
				t.FailNow()
			}
		}()

		exec, err := svc.GetResult("missing-exec-id")
		if exec == nil && err == nil {
			fmt.Println("RED (红灯，缺陷未修复)")
			t.FailNow()
		}
		if err != nil {
			fmt.Println("GREEN (绿灯，缺陷已修复)")
		} else {
			fmt.Println("RED (红灯，缺陷未修复)")
			t.FailNow()
		}
	})

	t.Run("Cancel existing execution still works", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				fmt.Println("RED (红灯，缺陷未修复)")
				t.FailNow()
			}
		}()

		exec := model.NewExecution("existing-id", "python", "print('ok')")
		if err := s.CreateExecution(exec); err != nil {
			t.Fatalf("failed to create execution: %v", err)
		}

		err := svc.Cancel("existing-id")
		if err != nil {
			fmt.Println("RED (红灯，缺陷未修复) - cancel existing should not error")
			t.FailNow()
		} else {
			fmt.Println("GREEN (绿灯，缺陷已修复)")
		}
	})
}
