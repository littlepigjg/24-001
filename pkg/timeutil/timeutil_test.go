package timeutil

import (
	"testing"
	"time"
)

// fixedNow is a deterministic UTC reference time used across all cases so the
// results never depend on the host machine's wall clock or timezone.
var fixedNow = time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)

func TestIsExpired(t *testing.T) {
	cases := []struct {
		name        string
		createdAt   time.Time
		maxAgeHours int
		want        bool
	}{
		{"23h old within 24h ttl is not expired", fixedNow.Add(-23 * time.Hour), 24, false},
		{"25h old beyond 24h ttl is expired", fixedNow.Add(-25 * time.Hour), 24, true},
		{"exactly maxAge old is not expired (Before is strict)", fixedNow.Add(-24 * time.Hour), 24, false},
		{"future createdAt is not expired", fixedNow.Add(time.Hour), 24, false},
		{"1h old within 24h ttl is not expired", fixedNow.Add(-time.Hour), 24, false},
		{"1h old beyond 1h ttl is expired", fixedNow.Add(-2 * time.Hour), 1, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := IsExpired(tc.createdAt, fixedNow, tc.maxAgeHours)
			if got != tc.want {
				t.Fatalf("IsExpired(createdAt=%v, now=%v, maxAgeHours=%d) = %v, want %v",
					tc.createdAt, fixedNow, tc.maxAgeHours, got, tc.want)
			}
		})
	}
}

func TestIsExpired_TimezoneInvariant(t *testing.T) {
	// The same logical instant must yield the same verdict regardless of the
	// time.Location attached to now. This is the core regression: previously
	// now had +8h added, so a UTC+8 host falsely aged a 23h link into expiry.
	createdAt := fixedNow.Add(-23 * time.Hour)
	cst, _ := time.LoadLocation("Asia/Shanghai")
	pst, _ := time.LoadLocation("America/Los_Angeles")
	nows := []time.Time{
		fixedNow,
		fixedNow.In(cst),
		fixedNow.In(pst),
	}
	for _, now := range nows {
		if IsExpired(createdAt, now, 24) {
			t.Fatalf("23h-old link reported expired when now is in %v: timezone leaked into comparison", now.Location())
		}
	}
}

func TestCleanupCutoffTime(t *testing.T) {
	got := CleanupCutoffTime(fixedNow, 24*time.Hour)
	want := fixedNow.Add(-24 * time.Hour)
	if !got.Equal(want) {
		t.Fatalf("CleanupCutoffTime = %v, want %v (no offset shift)", got, want)
	}

	// 23h-old link: cutoff is 24h before now, so a 23h-old CreatedAt is after
	// the cutoff and must NOT be treated as expired.
	createdAt23h := fixedNow.Add(-23 * time.Hour)
	if createdAt23h.Before(got) {
		t.Fatalf("23h-old link (CreatedAt=%v) is before cutoff %v — would be wrongly cleaned up", createdAt23h, got)
	}

	// 25h-old link is before the cutoff and is correctly eligible for cleanup.
	createdAt25h := fixedNow.Add(-25 * time.Hour)
	if !createdAt25h.Before(got) {
		t.Fatalf("25h-old link (CreatedAt=%v) is not before cutoff %v — should be eligible for cleanup", createdAt25h, got)
	}
}
