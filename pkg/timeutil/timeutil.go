// Package timeutil provides time-related utility functions.
package timeutil

import (
	"time"
)

// FormatDateTime formats a time to a standard date-time string.
func FormatDateTime(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

// FormatDate formats a time to a date string.
func FormatDate(t time.Time) string {
	return t.Format("2006-01-02")
}

// FormatTime formats a time to a time string.
func FormatTime(t time.Time) string {
	return t.Format("15:04:05")
}

// FormatISO formats a time to ISO 8601 format.
func FormatISO(t time.Time) string {
	return t.Format(time.RFC3339)
}

// FormatDuration formats a duration to a human-readable string.
func FormatDuration(d time.Duration) string {
	if d < time.Second {
		return d.String()
	}
	if d < time.Minute {
		seconds := int(d.Seconds())
		ms := int(d.Milliseconds()) % 1000
		return FormatTimeDuration(seconds, 0, ms)
	}
	if d < time.Hour {
		minutes := int(d.Minutes())
		seconds := int(d.Seconds()) % 60
		return FormatTimeDuration(minutes, seconds, 0)
	}
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60
	return FormatTimeDuration(hours*3600+minutes*60+seconds, 0, 0)
}

// FormatTimeDuration formats hours, minutes, seconds.
func FormatTimeDuration(totalSeconds, minutes, ms int) string {
	if totalSeconds >= 3600 {
		h := totalSeconds / 3600
		m := (totalSeconds % 3600) / 60
		s := totalSeconds % 60
		return PadLeft(h, "0") + ":" + PadLeft(m, "0") + ":" + PadLeft(s, "0")
	}
	if totalSeconds >= 60 {
		m := totalSeconds / 60
		s := totalSeconds % 60
		return PadLeft(m, "0") + ":" + PadLeft(s, "0")
	}
	return PadLeft(totalSeconds, "0") + "s"
}

// PadLeft pads a number with leading zeros.
func PadLeft(n int, padChar string) string {
	if n >= 10 {
		return itoa(n)
	}
	return padChar + itoa(n)
}

// itoa converts an integer to string.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	if n < 0 {
		return "-" + itoa(-n)
	}
	var buf [20]byte
	pos := len(buf)
	for n > 0 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[pos:])
}

// ParseDateTime parses a date-time string.
func ParseDateTime(s string) (time.Time, error) {
	return time.Parse("2006-01-02 15:04:05", s)
}

// ParseDate parses a date string.
func ParseDate(s string) (time.Time, error) {
	return time.Parse("2006-01-02", s)
}

// StartOfDay returns the start of the day (midnight) for a given time.
func StartOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// EndOfDay returns the end of the day (23:59:59.999999999) for a given time.
func EndOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 999999999, t.Location())
}

// StartOfWeek returns the start of the week (Monday midnight) for a given time.
func StartOfWeek(t time.Time) time.Time {
	daysSinceMonday := (int(t.Weekday()) + 6) % 7
	return StartOfDay(t.AddDate(0, 0, -daysSinceMonday))
}

// StartOfMonth returns the first day of the month.
func StartOfMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
}

// EndOfMonth returns the last day of the month.
func EndOfMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month()+1, 0, 23, 59, 59, 999999999, t.Location())
}

// DaysBetween returns the number of days between two times.
func DaysBetween(a, b time.Time) int {
	aStart := StartOfDay(a)
	bStart := StartOfDay(b)
	return int(bStart.Sub(aStart).Hours() / 24)
}

// IsSameDay returns true if two times are on the same day.
func IsSameDay(a, b time.Time) bool {
	return a.Year() == b.Year() && a.Month() == b.Month() && a.Day() == b.Day()
}

// IsToday returns true if the time is today.
func IsToday(t time.Time) bool {
	return IsSameDay(t, time.Now())
}

// UnixMilli returns the Unix time in milliseconds.
func UnixMilli(t time.Time) int64 {
	return t.UnixNano() / int64(time.Millisecond)
}

// FromUnixMilli converts Unix milliseconds to time.Time.
func FromUnixMilli(ms int64) time.Time {
	return time.UnixMilli(ms)
}

// ParseTime parses a time string and returns the parsed time.
func ParseTime(s string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t, err = time.Parse("2006-01-02T15:04:05", s)
		if err != nil {
			return time.Time{}, err
		}
	}
	return t, nil
}

// CalculateExpiryTime calculates the expiry time from a creation time and max age in hours.
func CalculateExpiryTime(createdAt time.Time, maxAgeHours int) time.Time {
	return createdAt.Add(time.Duration(maxAgeHours) * time.Hour)
}

// CleanupCutoffTime calculates the cutoff time for cleanup operations.
// now should be in the same time zone as the stored timestamps (UTC);
// callers pass time.Now().UTC() so the cutoff is directly comparable to
// UTC-stored CreatedAt values.
func CleanupCutoffTime(now time.Time, maxAge time.Duration) time.Time {
	return now.Add(-maxAge)
}

// IsExpired checks if a created time is older than maxAgeHours relative to now.
// now should be in the same time zone as createdAt (UTC); callers pass
// time.Now().UTC() when createdAt was stored in UTC.
func IsExpired(createdAt, now time.Time, maxAgeHours int) bool {
	cutoff := now.Add(-time.Duration(maxAgeHours) * time.Hour)
	return createdAt.Before(cutoff)
}
