package main

import (
	"context"
	"strings"
	"testing"

	"github.com/codesandbox/codesandbox/internal/model"
	"github.com/codesandbox/codesandbox/pkg/response"
)

func TestRedGreen(t *testing.T) {
	var failedTests []string

	// Test 1: DecodeJSON should allow unknown fields
	t.Run("DecodeJSON allows extra fields", func(t *testing.T) {
		jsonInput := `{"language":"python","code":"print('hello')","extra_field":"value"}`
		req := &model.ExecutionRequest{}
		err := response.DecodeJSON(strings.NewReader(jsonInput), req)
		if err != nil {
			failedTests = append(failedTests, "DecodeJSON rejected valid request with extra fields: "+err.Error())
			t.Errorf("FAIL: DecodeJSON should allow unknown fields, got error: %v", err)
		}
	})

	// Test 2: DecodeJSONWithContext should allow extra fields
	t.Run("DecodeJSONWithContext allows extra fields", func(t *testing.T) {
		jsonInput := `{"language":"javascript","code":"console.log('hi')","unknown_param":"test"}`
		req := &model.ExecutionRequest{}
		ctx := context.Background()
		err := response.DecodeJSONWithContext(ctx, strings.NewReader(jsonInput), req)
		if err != nil {
			failedTests = append(failedTests, "DecodeJSONWithContext rejected valid request with extra fields: "+err.Error())
			t.Errorf("FAIL: DecodeJSONWithContext should allow unknown fields, got error: %v", err)
		}
	})

	// Test 3: DecodeExecutionRequest should handle requests with extra fields
	t.Run("DecodeExecutionRequest handles extra fields", func(t *testing.T) {
		jsonInput := `{"language":"shell","code":"echo hi","meta":"data","version":2}`
		req, err := model.DecodeExecutionRequest(strings.NewReader(jsonInput))
		if err != nil {
			failedTests = append(failedTests, "DecodeExecutionRequest rejected valid request: "+err.Error())
			t.Errorf("FAIL: DecodeExecutionRequest should allow unknown fields, got error: %v", err)
			return
		}
		if req.Language != "shell" {
			failedTests = append(failedTests, "DecodeExecutionRequest parsed wrong language")
			t.Errorf("FAIL: Expected language 'shell', got '%s'", req.Language)
		}
		if req.Code != "echo hi" {
			failedTests = append(failedTests, "DecodeExecutionRequest parsed wrong code")
			t.Errorf("FAIL: Expected code 'echo hi', got '%s'", req.Code)
		}
	})

	// Test 4: DecodeExecutionRequestWithContext should handle extra fields
	t.Run("DecodeExecutionRequestWithContext handles extra fields", func(t *testing.T) {
		jsonInput := `{"language":"python","code":"x=1","comment":"test"}`
		ctx := context.Background()
		req, err := model.DecodeExecutionRequestWithContext(ctx, strings.NewReader(jsonInput))
		if err != nil {
			failedTests = append(failedTests, "DecodeExecutionRequestWithContext rejected valid request: "+err.Error())
			t.Errorf("FAIL: DecodeExecutionRequestWithContext should allow unknown fields, got error: %v", err)
			return
		}
		if req.Language != "python" {
			failedTests = append(failedTests, "DecodeExecutionRequestWithContext parsed wrong language")
			t.Errorf("FAIL: Expected language 'python', got '%s'", req.Language)
		}
	})

	// Test 5: DecodeJSONLenient should allow extra fields (baseline)
	t.Run("DecodeJSONLenient allows extra fields", func(t *testing.T) {
		jsonInput := `{"language":"python","code":"print(1)","bonus":"data"}`
		req := &model.ExecutionRequest{}
		err := response.DecodeJSONLenient(strings.NewReader(jsonInput), req)
		if err != nil {
			failedTests = append(failedTests, "DecodeJSONLenient failed unexpectedly: "+err.Error())
			t.Errorf("FAIL: DecodeJSONLenient should allow unknown fields, got error: %v", err)
		}
	})

	// Test 6: Verify basic decoding without extra fields still works
	t.Run("Basic JSON decoding works", func(t *testing.T) {
		jsonInput := `{"language":"python","code":"print('hi')"}`
		req := &model.ExecutionRequest{}
		err := response.DecodeJSON(strings.NewReader(jsonInput), req)
		if err != nil {
			failedTests = append(failedTests, "Basic decoding failed: "+err.Error())
			t.Errorf("FAIL: Basic decoding without extra fields should work, got error: %v", err)
		}
		if req.Language != "python" {
			failedTests = append(failedTests, "Basic decoding parsed wrong language")
			t.Errorf("FAIL: Expected language 'python', got '%s'", req.Language)
		}
	})

	// Print final RED/GREEN verdict
	if len(failedTests) > 0 {
		t.Log("RED (红灯，缺陷未修复)")
		t.Logf("共 %d 项测试失败:", len(failedTests))
		for _, msg := range failedTests {
			t.Logf("  - %s", msg)
		}
	} else {
		t.Log("GREEN (绿灯，缺陷已修复)")
	}
}