package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/codesandbox/codesandbox/internal/handler"
	"github.com/codesandbox/codesandbox/pkg/response"
)

func TestRedGreen(t *testing.T) {
	cfg := response.DefaultCORSConfig()

	var h http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Test", "ok")
		w.WriteHeader(http.StatusOK)
	})

	h = handler.WithCORS(h, cfg)

	req := httptest.NewRequest("GET", "/api/test", nil)
	req.Header.Set("Origin", "http://evil.example.com")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	originHeader := w.Header().Get("Access-Control-Allow-Origin")
	credentialsHeader := w.Header().Get("Access-Control-Allow-Credentials")
	methodsHeader := w.Header().Get("Access-Control-Allow-Methods")

	if originHeader == "*" && credentialsHeader == "true" {
		t.Log("RED（红灯，缺陷未修复）")
		t.Logf("检测到危险组合: Access-Control-Allow-Origin=%s, Access-Control-Allow-Credentials=%s", originHeader, credentialsHeader)
		t.Log("当允许所有来源（*）同时启用凭证时，存在CSRF攻击风险")
		t.Fail()
		return
	}

	if methodsHeader == "" {
		t.Log("RED（红灯，缺陷未修复）")
		t.Log("CORS 配置缺少必要的 Allow-Methods 头")
		t.Fail()
		return
	}

	t.Log("GREEN（绿灯，缺陷已修复）")
}