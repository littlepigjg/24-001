// Package hash provides hashing utilities for security and integrity checks.
package hash

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash/fnv"
)

// MD5 calculates the MD5 hash of a string.
func MD5(s string) string {
	h := md5.Sum([]byte(s))
	return hex.EncodeToString(h[:])
}

// SHA256 calculates the SHA-256 hash of a string.
func SHA256(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

// FNV32 calculates the FNV-32 hash of a string.
func FNV32(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()
}

// FNV64 calculates the FNV-64 hash of a string.
func FNV64(s string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(s))
	return h.Sum64()
}

// CodeHash generates a hash of code content for deduplication.
func CodeHash(language, code string) string {
	combined := fmt.Sprintf("%s:%s", language, code)
	return SHA256(combined)
}

// UniqueID generates a unique identifier from content.
func UniqueID(parts ...string) string {
	combined := ""
	for _, p := range parts {
		combined += p + "|"
	}
	return MD5(combined)
}
