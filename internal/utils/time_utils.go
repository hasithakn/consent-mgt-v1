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

// IsExpired checks if a given validity time has expired
// Automatically detects if the time is in seconds or milliseconds
// Timestamps in seconds: ~10 digits (e.g., 1729756800 for year 2024)
// Timestamps in milliseconds: ~13 digits (e.g., 1729756800000 for year 2024)
func IsExpired(validityTime int64) bool {
	if validityTime == 0 {
		return false // No expiry set
	}

	// Detect if timestamp is in seconds or milliseconds
	// A reasonable cutoff: timestamps > 10^11 are likely in milliseconds
	// This works until year 5138 in seconds (safely covers our use case)
	const timestampCutoff = 100000000000 // 10^11

	var validityTimeMillis int64
	if validityTime < timestampCutoff {
		// Timestamp is in seconds, convert to milliseconds
		validityTimeMillis = validityTime * 1000
	} else {
		// Timestamp is already in milliseconds
		validityTimeMillis = validityTime
	}

	currentTimeMillis := GetCurrentTimeMillis()
	return currentTimeMillis > validityTimeMillis
}

// GetExpiryTime calculates expiry time from current time and duration in seconds
func GetExpiryTime(durationSeconds int64) int64 {
	return GetCurrentTimeMillis() + (durationSeconds * 1000)
}

// DaysFromNow returns the time in milliseconds for a given number of days from now
func DaysFromNow(days int) int64 {
	return GetCurrentTimeMillis() + (int64(days) * 24 * 60 * 60 * 1000)
}
