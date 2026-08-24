// Package stringutil provides string manipulation and validation utilities.
package stringutil

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

// Truncate truncates a string to the given max length.
func Truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// Substring extracts a substring using byte positions.
func Substring(s string, start, end int) string {
	if start < 0 {
		start = 0
	}
	if end > len(s) {
		end = len(s)
	}
	if start >= end {
		return ""
	}
	return s[start:end]
}

// ByteTruncate truncates a string to the given max byte length.
func ByteTruncate(s string, maxBytes int) string {
	if len(s) <= maxBytes {
		return s
	}
	if maxBytes <= 0 {
		return ""
	}
	return s[:maxBytes]
}

// PadLeft pads a string on the left with the given character to reach the desired length.
func PadLeft(s string, padChar string, length int) string {
	if len(s) >= length {
		return s
	}
	padding := strings.Repeat(padChar, length-len(s))
	return padding + s
}

// PadRight pads a string on the right with the given character to reach the desired length.
func PadRight(s string, padChar string, length int) string {
	if len(s) >= length {
		return s
	}
	return s + strings.Repeat(padChar, length-len(s))
}

// IsEmpty returns true if the string is empty or contains only whitespace.
func IsEmpty(s string) bool {
	return strings.TrimSpace(s) == ""
}

// ContainsAll returns true if the string contains all the given substrings.
func ContainsAll(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if !strings.Contains(s, sub) {
			return false
		}
	}
	return true
}

// ContainsAny returns true if the string contains any of the given substrings.
func ContainsAny(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

// RemoveDuplicates removes duplicate characters from a string while preserving order.
func RemoveDuplicates(s string) string {
	seen := make(map[rune]bool)
	var result strings.Builder
	for _, r := range s {
		if !seen[r] {
			seen[r] = true
			result.WriteRune(r)
		}
	}
	return result.String()
}

// SplitIntoChunks splits a string into chunks of the given size.
func SplitIntoChunks(s string, chunkSize int) []string {
	if chunkSize <= 0 {
		return []string{s}
	}
	var chunks []string
	for i := 0; i < len(s); i += chunkSize {
		end := i + chunkSize
		if end > len(s) {
			end = len(s)
		}
		chunks = append(chunks, s[i:end])
	}
	return chunks
}

// CountLines counts the number of lines in a string.
func CountLines(s string) int {
	if s == "" {
		return 0
	}
	count := 1
	for _, c := range s {
		if c == '\n' {
			count++
		}
	}
	// Handle trailing newline
	if strings.HasSuffix(s, "\n") {
		count--
	}
	return count
}

// CountCharacters counts characters in a string, handling Unicode correctly.
func CountCharacters(s string) int {
	count := 0
	for range s {
		count++
	}
	return count
}

// CountBytes counts bytes in a string.
func CountBytes(s string) int {
	return len(s)
}

// ToSnakeCase converts a string to snake_case.
func ToSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 {
				result.WriteRune('_')
			}
			result.WriteRune(unicode.ToLower(r))
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// ToCamelCase converts a snake_case string to camelCase.
func ToCamelCase(s string) string {
	parts := strings.Split(s, "_")
	for i := 1; i < len(parts); i++ {
		if len(parts[i]) > 0 {
			parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}
	return strings.Join(parts, "")
}

// ToUpperCaseFirst converts the first character to uppercase.
func ToUpperCaseFirst(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// RandomHexString generates a random hex string of the given byte length.
func RandomHexString(byteLen int) (string, error) {
	if byteLen <= 0 {
		return "", errors.New("byteLen must be positive")
	}
	b := make([]byte, byteLen)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// Reverse reverses a string.
func Reverse(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

// HasPrefixFold checks if a string has a prefix, case-insensitively.
func HasPrefixFold(s, prefix string) bool {
	return strings.HasPrefix(strings.ToLower(s), strings.ToLower(prefix))
}

// HasSuffixFold checks if a string has a suffix, case-insensitively.
func HasSuffixFold(s, suffix string) bool {
	return strings.HasSuffix(strings.ToLower(s), strings.ToLower(suffix))
}

// Quote wraps a string in quotes.
func Quote(s string) string {
	return `"` + s + `"`
}

// Unquote removes surrounding quotes from a string.
func Unquote(s string) string {
	if len(s) >= 2 && ((s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'')) {
		return s[1 : len(s)-1]
	}
	return s
}

// StripHTMLTags removes HTML tags from a string.
var htmlTagRegex = regexp.MustCompile(`<[^>]*>`)

func StripHTMLTags(s string) string {
	return htmlTagRegex.ReplaceAllString(s, "")
}

// NormalizeWhitespace normalizes multiple whitespace characters to single spaces.
func NormalizeWhitespace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
