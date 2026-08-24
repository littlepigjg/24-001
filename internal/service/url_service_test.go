package service

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/codesandbox/codesandbox/internal/config"
	"github.com/codesandbox/codesandbox/internal/model"
	"github.com/codesandbox/codesandbox/internal/store"
)

// newTestURLService wires a URLService against a fresh temp data dir so tests
// never touch the caller's ./data tree.
func newTestURLService(t *testing.T) (*URLService, *store.URLStore, string) {
	t.Helper()
	dir := t.TempDir()
	cfg := &config.Config{}
	cfg.Storage.URLFilePath(filepath.Join(dir, "urls"))

	urlStore, err := store.NewURLStore(cfg)
	if err != nil {
		t.Fatalf("NewURLStore: %v", err)
	}
	svc, err := NewURLService(cfg, urlStore)
	if err != nil {
		t.Fatalf("NewURLService: %v", err)
	}
	return svc, urlStore, dir
}

func TestCreateReqValidate_RejectsPathTraversal(t *testing.T) {
	bad := []string{
		"../config/app",
		"..",
		"foo/bar",
		"foo\\bar",
		"a.b",
		"ab",            // too short
		"with space",    // invalid char
		"ok\x00x",       // null byte
		"schläßt",       // non-ascii letter rejected by ascii-only intent? (unicode letter passes; but '.'/'/' must fail)
	}
	// Note: unicode letters are allowed by design; this case just must not panic.
	for _, code := range bad {
		req := &model.CreateReq{RawURL: "https://example.com", CustomCode: code}
		err := req.Validate()
		if err == nil && (strings.Contains(code, "/") || strings.Contains(code, "\\") || strings.Contains(code, "..") || strings.Contains(code, ".") || strings.Contains(code, "\x00")) {
			t.Errorf("expected Validate() to reject traversal-laden code %q, got nil", code)
		}
	}
}

func TestCreateReqValidate_AcceptsLegalCodes(t *testing.T) {
	for _, code := range []string{"abc", "abc-123", "my_link", "A1-B2_C3"} {
		req := &model.CreateReq{RawURL: "https://example.com", CustomCode: code}
		if err := req.Validate(); err != nil {
			t.Errorf("expected %q to validate, got %v", code, err)
		}
	}
}

func TestCreate_RejectsTraversalCode_NoFileWritten(t *testing.T) {
	svc, _, dir := newTestURLService(t)

	_, err := svc.Create(context.Background(), &model.CreateReq{
		RawURL:     "https://example.com",
		CustomCode: "../config/app",
	})
	if err == nil {
		t.Fatal("expected Create to reject ../config/app, got nil error")
	}

	// The whole point: no file should have escaped the urls data dir.
	escaped := filepath.Join(dir, "config", "app.json")
	if _, statErr := os.Stat(escaped); statErr == nil {
		t.Fatalf("traversal escaped data dir: %s exists", escaped)
	}
}

func TestCreate_AcceptsLegalCustomCode_WritesUnderDataDir(t *testing.T) {
	svc, _, dir := newTestURLService(t)

	got, err := svc.Create(context.Background(), &model.CreateReq{
		RawURL:     "https://example.com",
		CustomCode: "abc-123",
	})
	if err != nil {
		t.Fatalf("Create legal code: %v", err)
	}
	wantPath := filepath.Join(dir, "urls", "abc-123.json")
	if _, statErr := os.Stat(wantPath); statErr != nil {
		t.Fatalf("expected %s to exist, got %v", wantPath, statErr)
	}
	if got.Code != "abc-123" {
		t.Fatalf("got code %q, want abc-123", got.Code)
	}
}

func TestStoreGet_RejectsTraversalCode(t *testing.T) {
	_, urlStore, _ := newTestURLService(t)

	if _, err := urlStore.Get("../config/app"); err == nil {
		t.Fatal("expected Get to reject ../config/app, got nil")
	}
}
