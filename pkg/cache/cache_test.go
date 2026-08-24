package cache

import (
	"testing"
	"time"
)

// TestGetReturnsMixedTypes reproduces the original panic: Cache.Get used to
// hard-code item.Value.(string), so retrieving a non-string value (e.g. an
// int64 stored by URLStore for the "visits:" key) panicked with
// "interface conversion: interface {} is int64, not string". After the fix,
// Get returns the value as-is and lets the caller type-assert.
func TestGetReturnsMixedTypes(t *testing.T) {
	c := New()

	// Mirror what URLStore.Save does: store values of several concrete types
	// under different keys in the same cache.
	c.Set("url:abc", "https://example.com", 24*time.Hour)
	c.Set("visits:abc", int64(0), 24*time.Hour)
	c.Set("custom:abc", false, 24*time.Hour)
	c.Set("disabled:abc", false, 24*time.Hour)

	// Retrieving the int64 value must not panic.
	v, ok := c.Get("visits:abc")
	if !ok {
		t.Fatal("expected visits key to be present")
	}
	n, ok := v.(int64)
	if !ok || n != 0 {
		t.Fatalf("expected int64(0), got %T %v (ok=%v)", v, v, ok)
	}

	// The string value is still returned correctly.
	u, ok := c.Get("url:abc")
	if !ok {
		t.Fatal("expected url key to be present")
	}
	if s, ok := u.(string); !ok || s != "https://example.com" {
		t.Fatalf("expected string url, got %T %v (ok=%v)", u, u, ok)
	}

	// Missing key returns (nil, false).
	if _, ok := c.Get("missing"); ok {
		t.Fatal("expected missing key to be absent")
	}
}

// TestGetExpiredItemReturnsFalse ensures expired items are reported as absent
// rather than returned.
func TestGetExpiredItemReturnsFalse(t *testing.T) {
	c := New()
	c.Set("k", "v", 1*time.Millisecond)
	time.Sleep(5 * time.Millisecond)
	if _, ok := c.Get("k"); ok {
		t.Fatal("expected expired item to be absent")
	}
}
