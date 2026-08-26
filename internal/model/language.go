package model

// LanguageInfo describes a programming language supported by the sandbox.
type LanguageInfo struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	DisplayName  string   `json:"display_name"`
	Extension    string   `json:"extension"`
	Executable   string   `json:"executable"`
	CompileCmd   string   `json:"compile_cmd,omitempty"`
	RunCmd       string   `json:"run_cmd,omitempty"`
	IsCompiled   bool     `json:"is_compiled"`
	TimeLimit    int      `json:"time_limit"`    // seconds
	MemoryLimit  int64    `json:"memory_limit"`  // bytes
	MaxFileSize  int64    `json:"max_file_size"` // bytes
	Available    bool     `json:"available"`
	Version      string   `json:"version,omitempty"`
	Icon         string   `json:"icon,omitempty"`
	Description  string   `json:"description,omitempty"`
	Examples     []string `json:"examples,omitempty"`
}

// LanguageList is a collection of supported languages.
type LanguageList struct {
	Languages []LanguageInfo `json:"languages"`
	Count     int            `json:"count"`
}

// DefaultLanguages returns the list of default supported languages.
func DefaultLanguages() []LanguageInfo {
	return []LanguageInfo{
		{
			ID:          "python",
			Name:        "Python",
			DisplayName: "Python 3",
			Extension:   ".py",
			Executable:  "python3",
			RunCmd:      "python3",
			IsCompiled:  false,
			TimeLimit:   30,
			MemoryLimit: 256 * 1024 * 1024,
			MaxFileSize: 10 * 1024 * 1024,
			Icon:        "🐍",
			Description: "Python 3 programming language",
			Examples: []string{
				`print("Hello, World!")`,
				`def fibonacci(n):
    a, b = 0, 1
    for _ in range(n):
        yield a
        a, b = b, a + b

print(list(fibonacci(10)))`,
			},
		},
		{
			ID:          "javascript",
			Name:        "JavaScript",
			DisplayName: "JavaScript (Node.js)",
			Extension:   ".js",
			Executable:  "node",
			RunCmd:      "node",
			IsCompiled:  false,
			TimeLimit:   30,
			MemoryLimit: 256 * 1024 * 1024,
			MaxFileSize: 10 * 1024 * 1024,
			Icon:        "📜",
			Description: "JavaScript via Node.js runtime",
			Examples: []string{
				`console.log("Hello, World!");`,
				`const fibonacci = (n) => {
  if (n <= 1) return n;
  return fibonacci(n - 1) + fibonacci(n - 2);
};
console.log(fibonacci(10));`,
			},
		},
		{
			ID:          "shell",
			Name:        "Shell",
			DisplayName: "Shell Script",
			Extension:   ".sh",
			Executable:  "/bin/sh",
			RunCmd:      "/bin/sh",
			IsCompiled:  false,
			TimeLimit:   30,
			MemoryLimit: 128 * 1024 * 1024,
			MaxFileSize: 10 * 1024 * 1024,
			Icon:        "🐚",
			Description: "Bourne shell scripting",
			Examples: []string{
				`echo "Hello, World!"`,
				`for i in $(seq 1 10); do
  echo "Number: $i"
done`,
			},
		},
		{
			ID:          "java",
			Name:        "Java",
			DisplayName: "Java",
			Extension:   ".java",
			Executable:  "java",
			CompileCmd:  "javac",
			RunCmd:      "java",
			IsCompiled:  true,
			TimeLimit:   30,
			MemoryLimit: 512 * 1024 * 1024,
			MaxFileSize: 10 * 1024 * 1024,
			Icon:        "☕",
			Description: "Java programming language",
			Examples: []string{
				`public class Main {
    public static void main(String[] args) {
        System.out.println("Hello, World!");
    }
}`,
			},
		},
		{
			ID:          "c",
			Name:        "C",
			DisplayName: "C",
			Extension:   ".c",
			Executable:  "gcc",
			CompileCmd:  "gcc",
			RunCmd:      "./a.out",
			IsCompiled:  true,
			TimeLimit:   30,
			MemoryLimit: 256 * 1024 * 1024,
			MaxFileSize: 10 * 1024 * 1024,
			Icon:        "🔧",
			Description: "C programming language",
			Examples: []string{
				`#include <stdio.h>
int main() {
    printf("Hello, World!\n");
    return 0;
}`,
			},
		},
	}
}

// GetLanguage returns language info by ID.
func GetLanguage(id string) (LanguageInfo, bool) {
	for _, lang := range DefaultLanguages() {
		if lang.ID == id {
			return lang, true
		}
	}
	return LanguageInfo{}, false
}

// GetLanguageIDs returns all supported language IDs.
func GetLanguageIDs() []string {
	langs := DefaultLanguages()
	ids := make([]string, len(langs))
	for i, l := range langs {
		ids[i] = l.ID
	}
	return ids
}

// IsLanguageSupported checks if a language is supported.
func IsLanguageSupported(id string) bool {
	_, ok := GetLanguage(id)
	return ok
}
