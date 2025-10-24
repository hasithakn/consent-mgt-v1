package utils

import (
	"testing"
	"time"
)

func TestIsExpired_Milliseconds(t *testing.T) {
	tests := []struct {
		name         string
		validityTime int64
		expected     bool
		description  string
	}{
		{
			name:         "Future time in milliseconds",
			validityTime: time.Now().Add(1 * time.Hour).UnixNano() / int64(time.Millisecond),
			expected:     false,
			description:  "Should not be expired for future time in milliseconds",
		},
		{
			name:         "Past time in milliseconds",
			validityTime: time.Now().Add(-1 * time.Hour).UnixNano() / int64(time.Millisecond),
			expected:     true,
			description:  "Should be expired for past time in milliseconds",
		},
		{
			name:         "Zero validity time",
			validityTime: 0,
			expected:     false,
			description:  "Zero validity time means no expiry",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsExpired(tt.validityTime)
			if result != tt.expected {
				t.Errorf("IsExpired(%d) = %v, want %v - %s", tt.validityTime, result, tt.expected, tt.description)
			}
		})
	}
}

func TestIsExpired_Seconds(t *testing.T) {
	tests := []struct {
		name         string
		validityTime int64
		expected     bool
		description  string
	}{
		{
			name:         "Future time in seconds",
			validityTime: time.Now().Add(1 * time.Hour).Unix(),
			expected:     false,
			description:  "Should not be expired for future time in seconds",
		},
		{
			name:         "Past time in seconds",
			validityTime: time.Now().Add(-1 * time.Hour).Unix(),
			expected:     true,
			description:  "Should be expired for past time in seconds",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsExpired(tt.validityTime)
			if result != tt.expected {
				t.Errorf("IsExpired(%d) = %v, want %v - %s", tt.validityTime, result, tt.expected, tt.description)
			}
		})
	}
}

func TestIsExpired_MixedFormats(t *testing.T) {
	// Test that both formats work correctly
	futureTimeSeconds := time.Now().Add(24 * time.Hour).Unix()
	futureTimeMillis := time.Now().Add(24 * time.Hour).UnixNano() / int64(time.Millisecond)

	pastTimeSeconds := time.Now().Add(-24 * time.Hour).Unix()
	pastTimeMillis := time.Now().Add(-24 * time.Hour).UnixNano() / int64(time.Millisecond)

	tests := []struct {
		name         string
		validityTime int64
		expected     bool
	}{
		{"Future in seconds (10 digits)", futureTimeSeconds, false},
		{"Future in milliseconds (13 digits)", futureTimeMillis, false},
		{"Past in seconds (10 digits)", pastTimeSeconds, true},
		{"Past in milliseconds (13 digits)", pastTimeMillis, true},
		{"Specific millisecond timestamp (2024-10-24)", 1729756800000, true}, // Past date
		{"Specific second timestamp (2024-10-24)", 1729756800, true},         // Past date
		{"Future millisecond timestamp (2030-01-01)", 1893456000000, false},
		{"Future second timestamp (2030-01-01)", 1893456000, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsExpired(tt.validityTime)
			if result != tt.expected {
				t.Errorf("IsExpired(%d) = %v, want %v", tt.validityTime, result, tt.expected)
			}
		})
	}
}

func TestGetCurrentTimeMillis(t *testing.T) {
	now := GetCurrentTimeMillis()
	
	// Should be a reasonable timestamp (after 2020 and before 2100)
	minTime := int64(1577836800000) // 2020-01-01 in milliseconds
	maxTime := int64(4102444800000) // 2100-01-01 in milliseconds
	
	if now < minTime || now > maxTime {
		t.Errorf("GetCurrentTimeMillis() = %d, expected between %d and %d", now, minTime, maxTime)
	}
	
	// Should be ~13 digits (milliseconds since epoch)
	if now < 1000000000000 || now > 9999999999999 {
		t.Errorf("GetCurrentTimeMillis() = %d, expected to be 13 digits", now)
	}
}

func TestMillisToTime(t *testing.T) {
	// Test known timestamp
	millis := int64(1729756800000) // 2024-10-24 00:00:00 UTC
	result := MillisToTime(millis)
	
	expected := time.Date(2024, 10, 24, 0, 0, 0, 0, time.UTC)
	if !result.Equal(expected) {
		t.Errorf("MillisToTime(%d) = %v, want %v", millis, result, expected)
	}
}

func TestTimeToMillis(t *testing.T) {
	// Test known time
	testTime := time.Date(2024, 10, 24, 0, 0, 0, 0, time.UTC)
	result := TimeToMillis(testTime)
	
	expected := int64(1729728000000) // 2024-10-24 00:00:00 UTC in milliseconds
	if result != expected {
		t.Errorf("TimeToMillis(%v) = %d, want %d", testTime, result, expected)
	}
}

func TestGetExpiryTime(t *testing.T) {
	before := GetCurrentTimeMillis()
	result := GetExpiryTime(3600) // 1 hour in seconds
	after := GetCurrentTimeMillis()
	
	// Result should be approximately 1 hour (3600000 ms) in the future
	expectedMin := before + 3600000
	expectedMax := after + 3600000
	
	if result < expectedMin || result > expectedMax {
		t.Errorf("GetExpiryTime(3600) = %d, expected between %d and %d", result, expectedMin, expectedMax)
	}
}

func TestDaysFromNow(t *testing.T) {
	before := GetCurrentTimeMillis()
	result := DaysFromNow(7) // 7 days
	after := GetCurrentTimeMillis()
	
	// Result should be approximately 7 days in the future
	sevenDaysInMillis := int64(7 * 24 * 60 * 60 * 1000)
	expectedMin := before + sevenDaysInMillis
	expectedMax := after + sevenDaysInMillis
	
	if result < expectedMin || result > expectedMax {
		t.Errorf("DaysFromNow(7) = %d, expected between %d and %d", result, expectedMin, expectedMax)
	}
}

func TestFormatTime(t *testing.T) {
	testTime := time.Date(2024, 10, 24, 12, 30, 45, 0, time.UTC)
	result := FormatTime(testTime)
	expected := "2024-10-24T12:30:45Z"
	
	if result != expected {
		t.Errorf("FormatTime(%v) = %s, want %s", testTime, result, expected)
	}
}

func TestParseTime(t *testing.T) {
	timeStr := "2024-10-24T12:30:45Z"
	result, err := ParseTime(timeStr)
	
	if err != nil {
		t.Errorf("ParseTime(%s) returned error: %v", timeStr, err)
	}
	
	expected := time.Date(2024, 10, 24, 12, 30, 45, 0, time.UTC)
	if !result.Equal(expected) {
		t.Errorf("ParseTime(%s) = %v, want %v", timeStr, result, expected)
	}
}

func TestParseTime_InvalidFormat(t *testing.T) {
	invalidTimeStr := "not-a-valid-time"
	_, err := ParseTime(invalidTimeStr)
	
	if err == nil {
		t.Errorf("ParseTime(%s) should return error for invalid format", invalidTimeStr)
	}
}
