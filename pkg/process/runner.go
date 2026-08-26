package process

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/codesandbox/codesandbox/pkg/logger"
)

// Executor handles the execution of code in sandboxed processes.
type Executor struct {
	runner  *Runner
	limiter *Limiter
	logger  *logger.Logger
}

// NewExecutor creates a new Executor.
func NewExecutor() *Executor {
	return &Executor{
		runner:  NewRunner(),
		limiter: NewLimiter(DefaultLimiterConfig()),
		logger:  logger.GetGlobal(),
	}
}

// ExecuteOptions configures code execution.
type ExecuteOptions struct {
	// Timeout is the maximum execution time.
	Timeout time.Duration
	// MemoryLimit is the maximum memory in bytes.
	MemoryLimit int64
	// WorkDir is the working directory for execution.
	WorkDir string
	// Stdin is the standard input to provide.
	Stdin string
	// Env is additional environment variables.
	Env []string
}

// DefaultExecuteOptions returns default execution options.
func DefaultExecuteOptions() ExecuteOptions {
	return ExecuteOptions{
		Timeout:    30 * time.Second,
		MemoryLimit: 256 * 1024 * 1024,
	}
}

// ExecutePython executes Python code in a sandboxed environment.
func (e *Executor) ExecutePython(ctx context.Context, code string, opts ExecuteOptions) (*Result, error) {
	// Create a temporary file for the code
	tmpFile, err := os.CreateTemp("", "codesandbox-python-*.py")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(code); err != nil {
		tmpFile.Close()
		return nil, fmt.Errorf("failed to write code to temp file: %w", err)
	}
	tmpFile.Close()

	// Build the command with resource limits
	cmdName, cmdArgs, err := e.limiter.ApplyLimits(ctx, "python3", []string{tmpFile.Name()}, opts)
	if err != nil {
		return nil, err
	}

	// Override workdir if specified
	if opts.WorkDir != "" {
		e.runner.WorkDir = opts.WorkDir
	} else {
		e.runner.WorkDir = ""
	}

	if opts.Env != nil {
		e.runner.Env = opts.Env
	}

	// Execute
	var result *Result
	if opts.Stdin != "" {
		result, err = e.runner.RunWithStdinTimeout(ctx, opts.Stdin, opts.Timeout, cmdName, cmdArgs...)
	} else {
		result, err = e.runner.RunWithTimeout(ctx, opts.Timeout, cmdName, cmdArgs...)
	}

	return result, err
}

// ExecuteJavaScript executes JavaScript code in a sandboxed environment.
func (e *Executor) ExecuteJavaScript(ctx context.Context, code string, opts ExecuteOptions) (*Result, error) {
	tmpFile, err := os.CreateTemp("", "codesandbox-js-*.js")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(code); err != nil {
		tmpFile.Close()
		return nil, fmt.Errorf("failed to write code to temp file: %w", err)
	}
	tmpFile.Close()

	cmdName, cmdArgs, err := e.limiter.ApplyLimits(ctx, "node", []string{tmpFile.Name()}, opts)
	if err != nil {
		return nil, err
	}

	if opts.WorkDir != "" {
		e.runner.WorkDir = opts.WorkDir
	} else {
		e.runner.WorkDir = ""
	}

	var result *Result
	if opts.Stdin != "" {
		result, err = e.runner.RunWithStdinTimeout(ctx, opts.Stdin, opts.Timeout, cmdName, cmdArgs...)
	} else {
		result, err = e.runner.RunWithTimeout(ctx, opts.Timeout, cmdName, cmdArgs...)
	}

	return result, err
}

// ExecuteShell executes shell commands in a sandboxed environment.
func (e *Executor) ExecuteShell(ctx context.Context, code string, opts ExecuteOptions) (*Result, error) {
	tmpFile, err := os.CreateTemp("", "codesandbox-shell-*.sh")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(code); err != nil {
		tmpFile.Close()
		return nil, fmt.Errorf("failed to write code to temp file: %w", err)
	}
	tmpFile.Close()

	// Make executable
	if err := os.Chmod(tmpFile.Name(), 0755); err != nil {
		return nil, fmt.Errorf("failed to make temp file executable: %w", err)
	}

	cmdName, cmdArgs, err := e.limiter.ApplyLimits(ctx, "/bin/sh", []string{tmpFile.Name()}, opts)
	if err != nil {
		return nil, err
	}

	if opts.WorkDir != "" {
		e.runner.WorkDir = opts.WorkDir
	} else {
		e.runner.WorkDir = ""
	}

	var result *Result
	if opts.Stdin != "" {
		result, err = e.runner.RunWithStdinTimeout(ctx, opts.Stdin, opts.Timeout, cmdName, cmdArgs...)
	} else {
		result, err = e.runner.RunWithTimeout(ctx, opts.Timeout, cmdName, cmdArgs...)
	}

	return result, err
}

// ExecuteCommand executes an arbitrary command with sandboxing.
func (e *Executor) ExecuteCommand(ctx context.Context, cmdName string, args []string, opts ExecuteOptions) (*Result, error) {
	safeArgs := make([]string, len(args))
	copy(safeArgs, args)

	limitedCmd, limitedArgs, err := e.limiter.ApplyLimits(ctx, cmdName, safeArgs, opts)
	if err != nil {
		return nil, err
	}

	if opts.WorkDir != "" {
		e.runner.WorkDir = opts.WorkDir
	} else {
		e.runner.WorkDir = ""
	}

	var result *Result
	if opts.Stdin != "" {
		result, err = e.runner.RunWithStdinTimeout(ctx, opts.Stdin, opts.Timeout, limitedCmd, limitedArgs...)
	} else {
		result, err = e.runner.RunWithTimeout(ctx, opts.Timeout, limitedCmd, limitedArgs...)
	}

	return result, err
}

// CleanupTempFiles removes old temporary files in the given directory.
func (e *Executor) CleanupTempFiles(dir string, maxAge time.Duration) (int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, err
	}

	removed := 0
	cutoff := time.Now().Add(-maxAge)

	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), "codesandbox-") {
			continue
		}
		fullPath := filepath.Join(dir, entry.Name())
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			if err := os.Remove(fullPath); err == nil {
				removed++
			}
		}
	}

	return removed, nil
}

// CompileAndRun compiles and runs code that requires compilation (like Java).
func (e *Executor) CompileAndRun(ctx context.Context, lang string, code string, opts ExecuteOptions) (*Result, error) {
	switch lang {
	case "java":
		return e.executeJava(ctx, code, opts)
	case "c":
		return e.executeC(ctx, code, opts)
	case "cpp":
		return e.executeC(ctx, code, opts)
	default:
		return nil, fmt.Errorf("unsupported compiled language: %s", lang)
	}
}

// executeJava compiles and runs Java code.
func (e *Executor) executeJava(ctx context.Context, code string, opts ExecuteOptions) (*Result, error) {
	// Create a temporary directory for Java files
	tmpDir, err := os.MkdirTemp("", "codesandbox-java-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Find the main class name from the code
	className := extractClassName(code)
	if className == "" {
		className = "Main"
	}

	// Write Java file
	javaFile := filepath.Join(tmpDir, className+".java")
	if err := os.WriteFile(javaFile, []byte(code), 0644); err != nil {
		return nil, fmt.Errorf("failed to write java file: %w", err)
	}

	// Compile
	compileResult, err := e.runner.RunWithTimeout(ctx, 10*time.Second, "javac", javaFile)
	if err != nil {
		return compileResult, err
	}
	if !compileResult.IsSuccess() {
		return compileResult, nil
	}

	// Run
	if opts.WorkDir == "" {
		opts.WorkDir = tmpDir
	}
	cmdName, cmdArgs, err := e.limiter.ApplyLimits(ctx, "java", []string{className}, opts)
	if err != nil {
		return nil, err
	}

	e.runner.WorkDir = opts.WorkDir
	if opts.Env != nil {
		e.runner.Env = opts.Env
	}

	var result *Result
	if opts.Stdin != "" {
		result, err = e.runner.RunWithStdinTimeout(ctx, opts.Stdin, opts.Timeout, cmdName, cmdArgs...)
	} else {
		result, err = e.runner.RunWithTimeout(ctx, opts.Timeout, cmdName, cmdArgs...)
	}

	return result, err
}

// executeC compiles and runs C/C++ code.
func (e *Executor) executeC(ctx context.Context, code string, opts ExecuteOptions) (*Result, error) {
	tmpDir, err := os.MkdirTemp("", "codesandbox-c-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	srcFile := filepath.Join(tmpDir, "main.c")
	if err := os.WriteFile(srcFile, []byte(code), 0644); err != nil {
		return nil, fmt.Errorf("failed to write c file: %w", err)
	}

	binFile := filepath.Join(tmpDir, "a.out")

	// Compile
	compileResult, err := e.runner.RunWithTimeout(ctx, 10*time.Second, "gcc", "-o", binFile, srcFile)
	if err != nil {
		return compileResult, err
	}
	if !compileResult.IsSuccess() {
		return compileResult, nil
	}

	// Run
	if opts.WorkDir == "" {
		opts.WorkDir = tmpDir
	}

	cmdName, cmdArgs, err := e.limiter.ApplyLimits(ctx, binFile, []string{}, opts)
	if err != nil {
		return nil, err
	}

	e.runner.WorkDir = opts.WorkDir
	if opts.Env != nil {
		e.runner.Env = opts.Env
	}

	var result *Result
	if opts.Stdin != "" {
		result, err = e.runner.RunWithStdinTimeout(ctx, opts.Stdin, opts.Timeout, cmdName, cmdArgs...)
	} else {
		result, err = e.runner.RunWithTimeout(ctx, opts.Timeout, cmdName, cmdArgs...)
	}

	return result, err
}

// extractClassName extracts the public class name from Java code.
func extractClassName(code string) string {
	for _, line := range strings.Split(code, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, "public class") {
			parts := strings.Fields(trimmed)
			for i, part := range parts {
				if part == "class" && i+1 < len(parts) {
					name := strings.TrimSuffix(parts[i+1], "{")
					return strings.TrimSpace(name)
				}
			}
		}
	}
	return ""
}

// GetLimiterSnapshot returns a diagnostic snapshot of the current limiter configuration.
func (e *Executor) GetLimiterSnapshot() LimiterConfig {
	return e.limiter.RawSnapshot()
}

// ListAvailableLanguages returns a list of languages that can be executed.
func (e *Executor) ListAvailableLanguages() []string {
	return []string{"python", "javascript", "shell", "java", "c"}
}

// IsLanguageAvailable checks if the runtime for a language is available.
func (e *Executor) IsLanguageAvailable(lang string) bool {
	switch lang {
	case "python":
		return commandExists("python3")
	case "javascript":
		return commandExists("node")
	case "java":
		return commandExists("java") && commandExists("javac")
	case "c", "cpp":
		return commandExists("gcc")
	case "shell":
		return commandExists("/bin/sh")
	default:
		return false
	}
}

// commandExists checks if a command is available in the system.
func commandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
