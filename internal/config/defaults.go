package config

import (
	"github.com/codesandbox/codesandbox/internal/model"
)

// Predefined templates for common languages.
var predefinedTemplates = map[string][]model.Template{
	"python": {
		{
			ID:          "tpl-python-hello",
			Name:        "Hello World",
			Description: "A simple hello world program",
			Language:    "python",
			Code:        `print("Hello, World!")`,
			Category:    "basic",
			IsPublic:    true,
		},
		{
			ID:          "tpl-python-fibonacci",
			Name:        "Fibonacci Sequence",
			Description: "Generate Fibonacci sequence using generator",
			Language:    "python",
			Code: `def fibonacci(n):
    a, b = 0, 1
    for _ in range(n):
        yield a
        a, b = b, a + b

print(list(fibonacci(10)))`,
			Category:    "algorithms",
			IsPublic:    true,
		},
		{
			ID:          "tpl-python-quick-sort",
			Name:        "Quick Sort",
			Description: "Quick sort algorithm implementation",
			Language:    "python",
			Code: `def quicksort(arr):
    if len(arr) <= 1:
        return arr
    pivot = arr[len(arr) // 2]
    left = [x for x in arr if x < pivot]
    middle = [x for x in arr if x == pivot]
    right = [x for x in arr if x > pivot]
    return quicksort(left) + middle + quicksort(right)

print(quicksort([3, 6, 8, 10, 1, 2, 1]))`,
			Category:    "algorithms",
			IsPublic:    true,
		},
	},
	"javascript": {
		{
			ID:          "tpl-js-hello",
			Name:        "Hello World",
			Description: "A simple hello world program in JavaScript",
			Language:    "javascript",
			Code:        `console.log("Hello, World!");`,
			Category:    "basic",
			IsPublic:    true,
		},
		{
			ID:          "tpl-js-fibonacci",
			Name:        "Fibonacci Sequence",
			Description: "Generate Fibonacci sequence",
			Language:    "javascript",
			Code: `function fibonacci(n) {
  if (n <= 1) return n;
  return fibonacci(n - 1) + fibonacci(n - 2);
}

const result = [];
for (let i = 0; i < 10; i++) {
  result.push(fibonacci(i));
}
console.log(result.join(", "));`,
			Category:    "algorithms",
			IsPublic:    true,
		},
	},
	"shell": {
		{
			ID:          "tpl-shell-hello",
			Name:        "Hello World",
			Description: "A simple shell script",
			Language:    "shell",
			Code:        `echo "Hello, World!"`,
			Category:    "basic",
			IsPublic:    true,
		},
	},
	"java": {
		{
			ID:          "tpl-java-hello",
			Name:        "Hello World",
			Description: "A simple Java hello world program",
			Language:    "java",
			Code: `public class Main {
    public static void main(String[] args) {
        System.out.println("Hello, World!");
    }
}`,
			Category:    "basic",
			IsPublic:    true,
		},
	},
	"c": {
		{
			ID:          "tpl-c-hello",
			Name:        "Hello World",
			Description: "A simple C hello world program",
			Language:    "c",
			Code: `#include <stdio.h>

int main() {
    printf("Hello, World!\n");
    return 0;
}`,
			Category:    "basic",
			IsPublic:    true,
		},
	},
}

// GetPredefinedTemplates returns predefined templates for a given language.
func GetPredefinedTemplates(language string) []model.Template {
	return predefinedTemplates[language]
}

// GetAllPredefinedTemplates returns all predefined templates.
func GetAllPredefinedTemplates() []model.Template {
	var all []model.Template
	for _, templates := range predefinedTemplates {
		all = append(all, templates...)
	}
	return all
}

// DefaultTimeoutMapping maps languages to their default timeouts.
var DefaultTimeoutMapping = map[string]int{
	"python":     30,
	"javascript": 30,
	"shell":      30,
	"java":       30,
	"c":          30,
}

// DefaultMemoryMapping maps languages to their default memory limits.
var DefaultMemoryMapping = map[string]int64{
	"python":     256 * 1024 * 1024,
	"javascript": 256 * 1024 * 1024,
	"shell":      128 * 1024 * 1024,
	"java":       512 * 1024 * 1024,
	"c":          256 * 1024 * 1024,
}

// GetDefaultTimeout returns the default timeout for a language.
func GetDefaultTimeout(language string) int {
	if timeout, ok := DefaultTimeoutMapping[language]; ok {
		return timeout
	}
	return 30
}

// GetDefaultMemory returns the default memory limit for a language.
func GetDefaultMemory(language string) int64 {
	if memory, ok := DefaultMemoryMapping[language]; ok {
		return memory
	}
	return 256 * 1024 * 1024
}
