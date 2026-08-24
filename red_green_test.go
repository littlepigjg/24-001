package codesandbox

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/codesandbox/codesandbox/internal/model"
	"github.com/codesandbox/codesandbox/internal/service"
	"github.com/codesandbox/codesandbox/internal/store"
)

func TestRedGreen(t *testing.T) {
	memStore := store.NewMemoryStore()
	svc := service.NewTemplateService(memStore)

	for i := 0; i < 5; i++ {
		req := &model.TemplateRequest{
			Name:     fmt.Sprintf("test-template-%d", i),
			Language: "python",
			Code:     fmt.Sprintf("print('hello %d')", i),
			Category: "test",
			Tags:     []string{"demo", "test"},
			IsPublic: true,
		}
		_, err := svc.Create(req)
		if err != nil {
			t.Fatalf("Failed to create template: %v", err)
		}
	}

	t.Run("normal_search_works", func(t *testing.T) {
		result, err := svc.Search("template")
		if err != nil {
			t.Errorf("Normal search failed: %v", err)
		}
		if len(result) == 0 {
			t.Errorf("Expected results for normal search, got 0")
		}
	})

	t.Run("malicious_search_detected", func(t *testing.T) {
		maliciousInput := "abcdefghijklmnopqrst"

		var wg sync.WaitGroup
		wg.Add(1)
		done := make(chan struct{})
		var searchErr error
		var searchResult []model.Template

		go func() {
			defer wg.Done()
			searchResult, searchErr = svc.Search(maliciousInput)
			close(done)
		}()

		timer := time.NewTimer(3 * time.Second)
		defer timer.Stop()

		select {
		case <-done:
			if searchErr != nil {
				t.Logf("Search completed with error: %v", searchErr)
			} else {
				t.Logf("Search completed with %d results", len(searchResult))
			}
			fmt.Println("GREEN (绿灯，缺陷已修复)")
		case <-timer.C:
			fmt.Println("RED (红灯，缺陷未修复)")
			t.Fatalf("Search with malicious input took too long (timeout 3s) - defect triggered")
		}
	})

	t.Run("search_with_special_chars", func(t *testing.T) {
		result, err := svc.Search("print")
		if err != nil {
			t.Logf("Special char search returned error: %v", err)
		}
		t.Logf("Special char search returned %d results", len(result))
	})
}
