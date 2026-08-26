// Package fileutil provides file and path utility functions.
package fileutil

import (
	"os"
	"path/filepath"
	"strings"
)

// FileExists checks if a file exists at the given path.
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// IsDirectory checks if a path is a directory.
func IsDirectory(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// EnsureDirectory ensures a directory exists, creating it if necessary.
func EnsureDirectory(dir string) error {
	return os.MkdirAll(dir, 0755)
}

// GetFileExtension returns the file extension (lowercase, without dot).
func GetFileExtension(filename string) string {
	ext := filepath.Ext(filename)
	return strings.TrimPrefix(strings.ToLower(ext), ".")
}

// GetFileSize returns the size of a file in bytes.
func GetFileSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// FormatSize formats a byte count to a human-readable string.
func FormatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return formatNumber(bytes) + " B"
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	suffixes := []string{"KB", "MB", "GB", "TB"}
	return formatNumber(float64(bytes)/float64(div)) + " " + suffixes[exp]
}

func formatNumber(n interface{}) string {
	switch v := n.(type) {
	case int64:
		return intToStr(v)
	case float64:
		return floatToStr(v)
	default:
		return "0"
	}
}

func intToStr(n int64) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func floatToStr(f float64) string {
	// Simple float to string with 2 decimal places
	neg := f < 0
	if neg {
		f = -f
	}
	intPart := int64(f)
	decPart := int64((f - float64(intPart)) * 100)
	if decPart < 0 {
		decPart = 0
	}
	intStr := intToStr(intPart)
	decStr := intToStr(decPart)
	if len(decStr) < 2 {
		decStr = "0" + decStr
	}
	result := intStr + "." + decStr
	if neg {
		return "-" + result
	}
	return result
}

// ReadFileContent reads a file and returns its content as a string.
func ReadFileContent(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// WriteFileContent writes content to a file.
func WriteFileContent(path, content string) error {
	dir := filepath.Dir(path)
	if err := EnsureDirectory(dir); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0644)
}

// Cleanup removes a temporary file or directory.
func Cleanup(path string) error {
	if err := os.RemoveAll(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// ListFiles lists files in a directory with optional extension filter.
func ListFiles(dir string, extensions ...string) ([]string, error) {
	var files []string
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if len(extensions) > 0 {
			ext := GetFileExtension(entry.Name())
			matched := false
			for _, e := range extensions {
				if ext == e {
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
