// Package service provides security scanning for code submissions.
package service

import (
	"regexp"
	"strings"
)

// SecurityScanner scans code for potential security issues.
type SecurityScanner struct {
	patterns []SecurityPattern
}

// SecurityPattern represents a security check pattern.
type SecurityPattern struct {
	ID          string
	Name        string
	Description string
	Language    string
	Regex       *regexp.Regexp
	Severity    string // "low", "medium", "high", "critical"
}

// SecurityIssue represents a found security issue.
type SecurityIssue struct {
	PatternID   string
	Name        string
	Description string
	Line        int
	Match       string
	Severity    string
}

// NewSecurityScanner creates a new SecurityScanner with default patterns.
func NewSecurityScanner() *SecurityScanner {
	patterns := []SecurityPattern{
		{
			ID:          "py-os-command",
			Name:        "Python OS Command Execution",
			Description: "Direct OS command execution using os.system or subprocess",
			Language:    "python",
			Regex:       regexp.MustCompile(`(?i)(os\.system\s*\(|subprocess\.call\s*\(|subprocess\.Popen\s*\(|subprocess\.run\s*\()`),
			Severity:    "high",
		},
		{
			ID:          "py-eval",
			Name:        "Python Eval Usage",
			Description: "Use of eval() which can execute arbitrary code",
			Language:    "python",
			Regex:       regexp.MustCompile(`\beval\s*\(`),
			Severity:    "high",
		},
		{
			ID:          "js-functions",
			Name:        "JavaScript Dangerous Functions",
			Description: "Use of dangerous functions like eval, Function, setTimeout with string",
			Language:    "javascript",
			Regex:       regexp.MustCompile(`(?i)(eval\s*\(|new\s+Function\s*\(|setTimeout\s*\(\s*['"])`),
			Severity:    "high",
		},
		{
			ID:          "java-runtime",
			Name:        "Java Runtime Execution",
			Description: "Direct command execution via Runtime.exec or ProcessBuilder",
			Language:    "java",
			Regex:       regexp.MustCompile(`(?i)(Runtime\.getRuntime\(\)\.exec\s*\(|new\s+ProcessBuilder\s*\()`),
			Severity:    "high",
		},
		{
			ID:          "sql-injection",
			Name:        "SQL Injection Pattern",
			Description: "Potential SQL injection through string concatenation",
			Language:    "any",
			Regex:       regexp.MustCompile(`(?i)(executeQuery\s*\(\s*"[^"]*"\s*\+|\.exec\s*\(\s*"[^"]*"\s*\+|query\s*\(\s*"[^"]*"\s*\+)`),
			Severity:    "medium",
		},
		{
			ID:          "file-system",
			Name:        "File System Access",
			Description: "Direct file system access outside sandbox",
			Language:    "any",
			Regex:       regexp.MustCompile(`(?i)(os\.file|open\s*\(|FileInputStream|FileOutputStream|files\.write)`),
			Severity:    "medium",
		},
	}
	return &SecurityScanner{patterns: patterns}
}

// Scan checks code for security issues.
func (s *SecurityScanner) Scan(code, language string) []SecurityIssue {
	var issues []SecurityIssue
	lines := strings.Split(code, "\n")

	for _, pattern := range s.patterns {
		if pattern.Language != "any" && pattern.Language != language {
			continue
		}

		for lineNum, line := range lines {
			matches := pattern.Regex.FindStringSubmatch(line)
			if matches != nil {
				issue := SecurityIssue{
					PatternID:   pattern.ID,
					Name:        pattern.Name,
					Description: pattern.Description,
					Line:        lineNum + 1,
					Match:       strings.TrimSpace(line),
					Severity:    pattern.Severity,
				}
				issues = append(issues, issue)
			}
		}
	}
	return issues
}

// HasCriticalIssues checks if the code has any security issues at or above the given severity.
func (s *SecurityScanner) HasCriticalIssues(code, language string) bool {
	issues := s.Scan(code, language)
	for _, issue := range issues {
		if issue.Severity == "high" || issue.Severity == "critical" {
			return true
		}
	}
	return false
}
