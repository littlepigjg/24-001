package main

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/codesandbox/codesandbox/internal/config"
	"github.com/codesandbox/codesandbox/internal/handler"
	"github.com/codesandbox/codesandbox/internal/model"
	"github.com/codesandbox/codesandbox/internal/service"
	"github.com/codesandbox/codesandbox/internal/store"
	"github.com/codesandbox/codesandbox/pkg/process"
	"github.com/codesandbox/codesandbox/pkg/response"
)

type trackingBody struct {
	data   *bytes.Buffer
	closed bool
}

func newTrackingBody(data string) *trackingBody {
	return &trackingBody{data: bytes.NewBufferString(data)}
}

func (t *trackingBody) Read(p []byte) (n int, err error) {
	return t.data.Read(p)
}

func (t *trackingBody) Close() error {
	t.closed = true
	return nil
}

func TestRedGreen(t *testing.T) {
	t.Run("DecodeJSONBody closes body on success", func(t *testing.T) {
		body := newTrackingBody(`{"language":"python","code":"print('hello')"}`)
		req, _ := http.NewRequest("POST", "/test", body)
		var v model.ExecutionRequest
		err := response.DecodeJSONBody(req, &v)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !body.closed {
			t.Log("RED (红灯，缺陷未修复)")
			t.Error("request body was not closed after DecodeJSONBody success")
		} else {
			t.Log("GREEN (绿灯，缺陷已修复)")
		}
	})

	t.Run("DecodeJSONBody closes body on decode error", func(t *testing.T) {
		body := newTrackingBody(`invalid json`)
		req, _ := http.NewRequest("POST", "/test", body)
		var v model.ExecutionRequest
		_ = response.DecodeJSONBody(req, &v)
		if !body.closed {
			t.Log("RED (红灯，缺陷未修复)")
			t.Error("request body was not closed after DecodeJSONBody decode error")
		} else {
			t.Log("GREEN (绿灯，缺陷已修复)")
		}
	})

	t.Run("ValidateRequestBody closes body on success", func(t *testing.T) {
		body := newTrackingBody(`{"language":"python","code":"print('hello')"}`)
		req, _ := http.NewRequest("POST", "/test", body)
		_, err := response.ValidateRequestBody(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !body.closed {
			t.Log("RED (红灯，缺陷未修复)")
			t.Error("request body was not closed after ValidateRequestBody success")
		} else {
			t.Log("GREEN (绿灯，缺陷已修复)")
		}
	})

	t.Run("ValidateRequestBody closes body on read error", func(t *testing.T) {
		body := newTrackingBody("")
		req, _ := http.NewRequest("POST", "/test", body)
		_, err := response.ValidateRequestBody(req)
		if err == nil {
			t.Fatal("expected error for empty body")
		}
		if !body.closed {
			t.Log("RED (红灯，缺陷未修复)")
			t.Error("request body was not closed after ValidateRequestBody read error")
		} else {
			t.Log("GREEN (绿灯，缺陷已修复)")
		}
	})

	t.Run("handler Execute closes body on validation error", func(t *testing.T) {
		dataStore := store.NewMemoryStore()
		cfgMgr := config.NewManager()
		histSvc := service.NewHistoryService(dataStore)
		tmplSvc := service.NewTemplateService(dataStore)
		langSvc := service.NewLanguageService(process.NewExecutor())
		execSvc := service.NewExecutionService(dataStore, histSvc, tmplSvc, langSvc, cfgMgr)
		validator := service.NewCodeValidator()
		h := handler.NewExecutionHandler(execSvc, validator)

		body := newTrackingBody(`{"language":"python","code":"print('hello')"}`)
		req, _ := http.NewRequest("POST", "/api/execute", body)
		req = req.WithContext(context.Background())
		w := httptest.NewRecorder()

		h.Execute(w, req)

		if !body.closed {
			t.Log("RED (红灯，缺陷未修复)")
			t.Error("request body was not closed after handler Execute")
		} else {
			t.Log("GREEN (绿灯，缺陷已修复)")
		}
	})

	t.Run("handler BatchExecute closes body", func(t *testing.T) {
		dataStore := store.NewMemoryStore()
		cfgMgr := config.NewManager()
		histSvc := service.NewHistoryService(dataStore)
		tmplSvc := service.NewTemplateService(dataStore)
		langSvc := service.NewLanguageService(process.NewExecutor())
		execSvc := service.NewExecutionService(dataStore, histSvc, tmplSvc, langSvc, cfgMgr)
		validator := service.NewCodeValidator()
		h := handler.NewExecutionHandler(execSvc, validator)

		body := newTrackingBody(`[{"language":"python","code":"print('hello')"}]`)
		req, _ := http.NewRequest("POST", "/api/execute/batch", body)
		req = req.WithContext(context.Background())
		w := httptest.NewRecorder()

		h.BatchExecute(w, req)

		if !body.closed {
			t.Log("RED (红灯，缺陷未修复)")
			t.Error("request body was not closed after handler BatchExecute")
		} else {
			t.Log("GREEN (绿灯，缺陷已修复)")
		}
	})
}
