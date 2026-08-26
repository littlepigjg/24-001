// Package uuid provides UUID generation utilities using crypto/rand.
package uuid

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
)

// UUID represents a Universally Unique Identifier.
type UUID [16]byte

// New generates a new random UUID v4.
func New() (UUID, error) {
	var uuid UUID
	_, err := rand.Read(uuid[:])
	if err != nil {
		return uuid, fmt.Errorf("failed to generate UUID: %w", err)
	}
	// Set version 4
	uuid[6] = (uuid[6] & 0x0f) | 0x40
	// Set variant 1
	uuid[8] = (uuid[8] & 0x3f) | 0x80
	return uuid, nil
}

// NewString generates a new UUID and returns it as a string.
func NewString() (string, error) {
	u, err := New()
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

// MustNewString generates a new UUID string, panicking on error.
func MustNewString() string {
	s, err := NewString()
	if err != nil {
		panic(err)
	}
	return s
}

// String returns the string representation of a UUID (xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx).
func (u UUID) String() string {
	var buf [36]byte
	hex.Encode(buf[0:8], u[0:4])
	buf[8] = '-'
	hex.Encode(buf[9:13], u[4:6])
	buf[13] = '-'
	hex.Encode(buf[14:18], u[6:8])
	buf[18] = '-'
	hex.Encode(buf[19:23], u[8:10])
	buf[23] = '-'
	hex.Encode(buf[24:36], u[10:16])
	return string(buf[:])
}

// StringSimple returns the string representation without dashes.
func (u UUID) StringSimple() string {
	var buf [32]byte
	hex.Encode(buf[0:], u[0:])
	return string(buf[:])
}

// Parse parses a UUID string into a UUID.
func Parse(s string) (UUID, error) {
	var uuid UUID
	// Remove dashes
	s = strings.ReplaceAll(s, "-", "")
	s = strings.ToLower(s)

	if len(s) != 32 {
		return uuid, fmt.Errorf("invalid UUID string length: %d", len(s))
	}

	b, err := hex.DecodeString(s)
	if err != nil {
		return uuid, fmt.Errorf("invalid UUID hex string: %w", err)
	}

	copy(uuid[:], b)
	return uuid, nil
}

// IsValid checks if a string is a valid UUID.
func IsValid(s string) bool {
	_, err := Parse(s)
	return err == nil
}

// Version returns the version of the UUID.
func (u UUID) Version() int {
	return int(u[6] >> 4)
}

// Variant returns the variant of the UUID.
func (u UUID) Variant() string {
	switch {
	case u[8]&0x80 == 0x80:
		return "RFC 4122"
	case u[8]&0xc0 == 0xc0:
		return "Microsoft"
	case u[8]&0xe0 == 0xe0:
		return "Future"
	default:
		return "Reserved"
	}
}

// Equal checks if two UUIDs are equal.
func (u UUID) Equal(other UUID) bool {
	for i := 0; i < 16; i++ {
		if u[i] != other[i] {
			return false
		}
	}
	return true
}
