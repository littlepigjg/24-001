package service

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/codesandbox/codesandbox/pkg/logger"
)

// CodeValidator performs safety checks on submitted code.
type CodeValidator struct {
	logger *logger.Logger

	// Patterns that indicate potentially dangerous code
	dangerousPatterns []dangerousPattern
}

type dangerousPattern struct {
	pattern string
	message string
	severity string // "warning" or "error"
}

// NewCodeValidator creates a new CodeValidator with default patterns.
func NewCodeValidator() *CodeValidator {
	return &CodeValidator{
		logger: logger.GetGlobal(),
		dangerousPatterns: []dangerousPattern{
			{
				pattern: `rm\s+-rf\s+/`,
				message: "Destructive command detected: rm -rf /",
				severity: "error",
			},
			{
				pattern: `mkfs\s+`,
				message: "Disk formatting command detected",
				severity: "error",
			},
			{
				pattern: `:\(\)\{.*;\}:.*&\s*/dev/null`,
				message: "Fork bomb detected",
				severity: "error",
			},
			{
				pattern: `wget\s+.*\|\s*(sh|bash)`,
				message: "Suspicious download-and-execute pattern",
				severity: "error",
			},
			{
				pattern: `curl\s+.*\|\s*(sh|bash)`,
				message: "Suspicious download-and-execute pattern",
				severity: "error",
			},
			{
				pattern: `chmod\s+\+s`,
				message: "Setuid bit modification detected",
				severity: "warning",
			},
			{
				pattern: `iptables`,
				message: "Firewall modification detected",
				severity: "warning",
			},
		},
	}
}

// ValidationResult holds the result of code validation.
type ValidationResult struct {
	Valid   bool     `json:"valid"`
	Errors  []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

// Validate performs safety checks on code.
func (v *CodeValidator) Validate(code, language string) ValidationResult {
	result := ValidationResult{
		Valid: true,
	}

	// Check for empty code
	if strings.TrimSpace(code) == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "Code cannot be empty")
		return result
	}

	// Check code size
	if len(code) > 100000 {
		result.Valid = false
		result.Errors = append(result.Errors, "Code exceeds maximum size of 100000 characters")
	}

	// Check line count
	lineCount := strings.Count(code, "\n") + 1
	if lineCount > 10000 {
		result.Valid = false
		result.Errors = append(result.Errors, "Code exceeds maximum line count of 10000")
	}

	// Check for dangerous patterns
	for _, dp := range v.dangerousPatterns {
		matched, err := regexp.MatchString(dp.pattern, code)
		if err != nil {
			v.logger.Warnf("Invalid regex pattern: %s", dp.pattern)
			continue
		}
		if matched {
			if dp.severity == "error" {
				result.Valid = false
				result.Errors = append(result.Errors, dp.message)
			} else {
				result.Warnings = append(result.Warnings, dp.message)
			}
		}
	}

	// Language-specific checks
	switch language {
	case "python":
		v.validatePython(code, &result)
	case "javascript":
		v.validateJavaScript(code, &result)
	case "shell":
		v.validateShell(code, &result)
	case "java":
		v.validateJava(code, &result)
	case "c":
		v.validateC(code, &result)
	}

	return result
}

// validatePython performs Python-specific checks.
func (v *CodeValidator) validatePython(code string, result *ValidationResult) {
	dangerousCalls := []string{
		"import os",
		"import sys",
		"import subprocess",
		"import socket",
		"import urllib",
		"import requests",
		"import shutil",
		"import ctypes",
	}

	for _, call := range dangerousCalls {
		if strings.Contains(code, call) {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("Potentially dangerous import: %s", call))
		}
	}
}

// validateJavaScript performs JavaScript-specific checks.
func (v *CodeValidator) validateJavaScript(code string, result *ValidationResult) {
	dangerousPatterns := []string{
		"require('fs')",
		"require('child_process')",
		"require('os')",
		"require('net')",
		"require('http')",
		"require('https')",
		"eval(",
		"Function(",
	}

	for _, pattern := range dangerousPatterns {
		if strings.Contains(code, pattern) {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("Potentially dangerous pattern: %s", pattern))
		}
	}
}

// validateShell performs shell-specific checks.
func (v *CodeValidator) validateShell(code string, result *ValidationResult) {
	dangerousCommands := []string{
		"sudo",
		"su ",
		"passwd",
		"useradd",
		"usermod",
		"groupadd",
		"shutdown",
		"reboot",
		"init 0",
	}

	for _, cmd := range dangerousCommands {
		if strings.Contains(code, cmd) {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("Potentially dangerous command: %s", cmd))
		}
	}
}

// validateJava performs Java-specific checks.
func (v *CodeValidator) validateJava(code string, result *ValidationResult) {
	if !strings.Contains(code, "public class") && !strings.Contains(code, "class") {
		result.Warnings = append(result.Warnings, "Code does not contain a class definition")
	}
	if !strings.Contains(code, "public static void main") {
		result.Warnings = append(result.Warnings, "Code does not contain a main method")
	}
}

// validateC performs C-specific checks.
func (v *CodeValidator) validateC(code string, result *ValidationResult) {
	if !strings.Contains(code, "#include") && !strings.Contains(code, "int main") {
		result.Warnings = append(result.Warnings, "Code may not be a valid C program")
	}
}

// AddDangerousPattern adds a custom dangerous pattern.
func (v *CodeValidator) AddDangerousPattern(pattern, message, severity string) {
	v.dangerousPatterns = append(v.dangerousPatterns, dangerousPattern{
		pattern:  pattern,
		message:  message,
		severity: severity,
	})
}

// RemoveDangerousPattern removes a dangerous pattern by index.
func (v *CodeValidator) RemoveDangerousPattern(index int) error {
	if index < 0 || index >= len(v.dangerousPatterns) {
		return fmt.Errorf("invalid index: %d", index)
	}
	v.dangerousPatterns = append(v.dangerousPatterns[:index], v.dangerousPatterns[index+1:]...)
	return nil
}
