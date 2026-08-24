// Package osutil provides OS-level utility functions.
package osutil

import (
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// FileExists checks if a file exists.
func FileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// DirExists checks if a directory exists.
func DirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// EnsureDir ensures a directory exists, creating it if necessary.
func EnsureDir(path string) error {
	return os.MkdirAll(path, 0755)
}

// CreateTempFile creates a temporary file and returns its path and a cleanup function.
func CreateTempFile(prefix, content string) (string, func(), error) {
	f, err := os.CreateTemp("", prefix)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create temp file: %w", err)
	}

	if _, err := f.WriteString(content); err != nil {
		f.Close()
		os.Remove(f.Name())
		return "", nil, fmt.Errorf("failed to write to temp file: %w", err)
	}
	f.Close()

	cleanup := func() {
		os.Remove(f.Name())
	}

	return f.Name(), cleanup, nil
}

// CreateTempDir creates a temporary directory and returns its path and a cleanup function.
func CreateTempDir(prefix string) (string, func(), error) {
	dir, err := os.MkdirTemp("", prefix)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create temp dir: %w", err)
	}

	cleanup := func() {
		os.RemoveAll(dir)
	}

	return dir, cleanup, nil
}

// CopyFile copies a file from source to destination.
func CopyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	return nil
}

// ReadFileContent reads the entire content of a file as a string.
func ReadFileContent(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}
	return string(data), nil
}

// WriteFileContent writes a string to a file.
func WriteFileContent(path, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}

// GetFileSize returns the size of a file in bytes.
func GetFileSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// GetFileExtension returns the file extension (with dot).
func GetFileExtension(path string) string {
	return filepath.Ext(path)
}

// GetFileBaseName returns the file name without extension.
func GetFileBaseName(path string) string {
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	return strings.TrimSuffix(base, ext)
}

// ListFiles lists files in a directory with optional extension filter.
func ListFiles(dir string, extensions ...string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if len(extensions) > 0 {
			ext := filepath.Ext(entry.Name())
			matched := false
			for _, e := range extensions {
				if strings.EqualFold(ext, e) {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}
		files = append(files, filepath.Join(dir, entry.Name()))
	}
	return files, nil
}

// FindExecutable finds an executable in the PATH.
func FindExecutable(name string) (string, error) {
	return exec.LookPath(name)
}

// EncodeBase64 encodes a string to base64.
func EncodeBase64(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

// DecodeBase64 decodes a base64 string.
func DecodeBase64(s string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// GetTempDir returns the system's temporary directory path.
func GetTempDir() string {
	return os.TempDir()
}

// GetWorkingDir returns the current working directory.
func GetWorkingDir() (string, error) {
	return os.Getwd()
}

// IsExecutable checks if a file has executable permissions.
func IsExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.Mode()&0111 != 0
}

// MakeExecutable makes a file executable.
func MakeExecutable(path string) error {
	return os.Chmod(path, 0755)
}

// IsProcessAlive checks if a process with the given PID exists.
func IsProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	path := fmt.Sprintf("/proc/%d", pid)
	_, err := os.Stat(path)
	return err == nil
}

// ReadProcessStatus reads the State field from /proc/[pid]/status.
func ReadProcessStatus(pid int) (string, error) {
	if pid <= 0 {
		return "", fmt.Errorf("invalid pid: %d", pid)
	}
	statusPath := fmt.Sprintf("/proc/%d/status", pid)
	data, err := os.ReadFile(statusPath)
	if err != nil {
		return "", fmt.Errorf("failed to read process status: %w", err)
	}
	content := string(data)
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, "State:") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				return parts[1], nil
			}
		}
	}
	return "", fmt.Errorf("state not found in process status")
}

// IsProcessZombie checks if a process is in zombie state (Z).
func IsProcessZombie(pid int) bool {
	state, err := ReadProcessStatus(pid)
	if err != nil {
		return false
	}
	return state == "Z"
}

// KillProcess sends SIGKILL to the process with the given PID.
func KillProcess(pid int) error {
	if pid <= 0 {
		return fmt.Errorf("invalid pid: %d", pid)
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find process: %w", err)
	}
	return process.Kill()
}
