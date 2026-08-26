// Package jsonutil provides JSON encoding/decoding utilities.
package jsonutil

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Marshal encodes a value to JSON string.
func Marshal(v interface{}) (string, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}
	return string(data), nil
}

// MarshalIndent encodes a value to a pretty-printed JSON string.
func MarshalIndent(v interface{}) (string, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}
	return string(data), nil
}

// Unmarshal decodes a JSON string into a value.
func Unmarshal(s string, v interface{}) error {
	return json.Unmarshal([]byte(s), v)
}

// UnmarshalStrict decodes a JSON string strictly, rejecting unknown fields.
func UnmarshalStrict(s string, v interface{}) error {
	decoder := json.NewDecoder(strings.NewReader(s))
	if err := decoder.Decode(v); err != nil {
		return err
	}
	return nil
}

// MustMarshal encodes a value to JSON, panicking on error.
func MustMarshal(v interface{}) string {
	s, err := Marshal(v)
	if err != nil {
		panic(err)
	}
	return s
}

// TryMarshal attempts to encode a value, returning an empty string on error.
func TryMarshal(v interface{}) string {
	s, err := Marshal(v)
	if err != nil {
		return ""
	}
	return s
}

// PrettyFormat formats a JSON string with indentation.
func PrettyFormat(s string) (string, error) {
	var v interface{}
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		return "", err
	}
	return MarshalIndent(v)
}

// MergeObjects merges two JSON objects (maps).
func MergeObjects(a, b string) (string, error) {
	var mapA, mapB map[string]interface{}
	if err := json.Unmarshal([]byte(a), &mapA); err != nil {
		return "", fmt.Errorf("failed to parse first object: %w", err)
	}
	if err := json.Unmarshal([]byte(b), &mapB); err != nil {
		return "", fmt.Errorf("failed to parse second object: %w", err)
	}

	for k, v := range mapB {
		mapA[k] = v
	}

	return Marshal(mapA)
}

// GetValue extracts a value from a JSON string by key path (e.g., "a.b.c").
func GetValue(jsonStr, path string) (interface{}, bool) {
	var data interface{}
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return nil, false
	}

	parts := strings.Split(path, ".")
	current := data

	for _, part := range parts {
		m, ok := current.(map[string]interface{})
		if !ok {
			return nil, false
		}
		val, exists := m[part]
		if !exists {
			return nil, false
		}
		current = val
	}

	return current, true
}

// ConvertToMap converts any value to a map if possible.
func ConvertToMap(v interface{}) (map[string]interface{}, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// ConvertFromMap converts a map to a typed value.
func ConvertFromMap(m map[string]interface{}, v interface{}) error {
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

// IsValidJSON checks if a string is valid JSON.
func IsValidJSON(s string) bool {
	var v interface{}
	return json.Unmarshal([]byte(s), &v) == nil
}
