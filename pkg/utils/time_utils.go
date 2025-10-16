package utils

import (
	"time"
)

// GetCurrentTimeMillis returns current time in milliseconds since epoch
func GetCurrentTimeMillis() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

// MillisToTime converts milliseconds since epoch to time.Time
func MillisToTime(millis int64) time.Time {
	return time.Unix(0, millis*int64(time.Millisecond))
}

// TimeToMillis converts time.Time to milliseconds since epoch
func TimeToMillis(t time.Time) int64 {
	return t.UnixNano() / int64(time.Millisecond)
}

// FormatTime formats time in ISO 8601 format
func FormatTime(t time.Time) string {
	return t.Format(time.RFC3339)
}

// ParseTime parses ISO 8601 formatted time string
func ParseTime(timeStr string) (time.Time, error) {
	return time.Parse(time.RFC3339, timeStr)
}

// IsExpired checks if a given validity time (in millis) has expired
func IsExpired(validityTime int64) bool {
	if validityTime == 0 {
		return false // No expiry set
	}
	return GetCurrentTimeMillis() > validityTime
}

// GetExpiryTime calculates expiry time from current time and duration in seconds
func GetExpiryTime(durationSeconds int64) int64 {
	return GetCurrentTimeMillis() + (durationSeconds * 1000)
}

// DaysFromNow returns the time in milliseconds for a given number of days from now
func DaysFromNow(days int) int64 {
	return GetCurrentTimeMillis() + (int64(days) * 24 * 60 * 60 * 1000)
}
