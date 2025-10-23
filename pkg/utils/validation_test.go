package utils

import (
	"testing"

	"github.com/wso2/consent-management-api/internal/config"
)

// mockConfig creates a test configuration
func mockConfig() *config.Config {
	return &config.Config{
		Consent: config.ConsentConfig{
			AllowedStatuses: []string{
				"CREATED",
				"awaitingAuthorization",
				"AUTHORIZED",
				"ACTIVE",
				"REJECTED",
				"REVOKED",
				"EXPIRED",
			},
			StatusMappings: config.ConsentStatusMappings{
				ActiveStatus:   "AUTHORIZED",
				ExpiredStatus:  "EXPIRED",
				RevokedStatus:  "REVOKED",
				CreatedStatus:  "CREATED",
				RejectedStatus: "REJECTED",
			},
		},
	}
}

func TestValidateStatus_ValidStatuses(t *testing.T) {
	// Setup: Load test config
	cfg := mockConfig()
	config.SetGlobal(cfg)

	validStatuses := []string{
		"CREATED",
		"awaitingAuthorization",
		"AUTHORIZED",
		"ACTIVE",
		"REJECTED",
		"REVOKED",
		"EXPIRED",
	}

	for _, status := range validStatuses {
		t.Run(status, func(t *testing.T) {
			err := ValidateStatus(status)
			if err != nil {
				t.Errorf("Expected status '%s' to be valid, but got error: %v", status, err)
			}
		})
	}
}

func TestValidateStatus_InvalidStatuses(t *testing.T) {
	// Setup: Load test config
	cfg := mockConfig()
	config.SetGlobal(cfg)

	invalidStatuses := []string{
		"INVALID",
		"PENDING",
		"COMPLETED",
		"",
		"lowercase",
		"MIXED_Case",
	}

	for _, status := range invalidStatuses {
		t.Run(status, func(t *testing.T) {
			err := ValidateStatus(status)
			if err == nil {
				t.Errorf("Expected status '%s' to be invalid, but validation passed", status)
			}
		})
	}
}

func TestValidateStatus_EmptyStatus(t *testing.T) {
	// Setup: Load test config
	cfg := mockConfig()
	config.SetGlobal(cfg)

	err := ValidateStatus("")
	if err == nil {
		t.Error("Expected empty status to be invalid, but validation passed")
	}

	expectedMsg := "status cannot be empty"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestValidateStatus_NilConfig(t *testing.T) {
	// Setup: Set nil config
	config.SetGlobal(nil)

	err := ValidateStatus("ACTIVE")
	if err == nil {
		t.Error("Expected error when config is nil, but validation passed")
	}

	expectedMsg := "configuration not loaded"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
	}

	// Restore config for other tests
	config.SetGlobal(mockConfig())
}

func TestValidateStatus_ErrorMessage(t *testing.T) {
	// Setup: Load test config
	cfg := mockConfig()
	config.SetGlobal(cfg)

	err := ValidateStatus("INVALID_STATUS")
	if err == nil {
		t.Fatal("Expected error for invalid status")
	}

	// Check that error message includes the invalid status and list of allowed statuses
	errMsg := err.Error()
	if !contains(errMsg, "INVALID_STATUS") {
		t.Errorf("Expected error message to contain 'INVALID_STATUS', got: %s", errMsg)
	}
	if !contains(errMsg, "allowed statuses") {
		t.Errorf("Expected error message to contain 'allowed statuses', got: %s", errMsg)
	}
}

func TestValidateConsentID(t *testing.T) {
	tests := []struct {
		name        string
		consentID   string
		expectError bool
	}{
		{"Valid ID", "CONSENT-123", false},
		{"Empty ID", "", true},
		{"Long ID", string(make([]byte, 256)), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConsentID(tt.consentID)
			if (err != nil) != tt.expectError {
				t.Errorf("ValidateConsentID(%s) error = %v, expectError %v", tt.consentID, err, tt.expectError)
			}
		})
	}
}

func TestValidateClientID(t *testing.T) {
	tests := []struct {
		name        string
		clientID    string
		expectError bool
	}{
		{"Valid ID", "client123", false},
		{"Empty ID", "", true},
		{"Long ID", string(make([]byte, 256)), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateClientID(tt.clientID)
			if (err != nil) != tt.expectError {
				t.Errorf("ValidateClientID(%s) error = %v, expectError %v", tt.clientID, err, tt.expectError)
			}
		})
	}
}

func TestValidateOrgID(t *testing.T) {
	tests := []struct {
		name        string
		orgID       string
		expectError bool
	}{
		{"Valid ID", "org123", false},
		{"Empty ID", "", true},
		{"Long ID", string(make([]byte, 256)), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateOrgID(tt.orgID)
			if (err != nil) != tt.expectError {
				t.Errorf("ValidateOrgID(%s) error = %v, expectError %v", tt.orgID, err, tt.expectError)
			}
		})
	}
}

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name        string
		email       string
		expectError bool
	}{
		{"Valid email", "test@example.com", false},
		{"Valid email with dots", "test.user@example.co.uk", false},
		{"Empty email", "", true},
		{"Invalid format", "notanemail", true},
		{"Missing @", "test.example.com", true},
		{"Missing domain", "test@", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEmail(tt.email)
			if (err != nil) != tt.expectError {
				t.Errorf("ValidateEmail(%s) error = %v, expectError %v", tt.email, err, tt.expectError)
			}
		})
	}
}

func TestSanitizeString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Normal string", "hello", "hello"},
		{"With whitespace", "  hello  ", "hello"},
		{"With null byte", "hello\x00world", "helloworld"},
		{"Multiple null bytes", "a\x00b\x00c", "abc"},
		{"Empty string", "", ""},
		{"Only whitespace", "   ", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeString(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeString(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestValidateLimit(t *testing.T) {
	tests := []struct {
		name     string
		limit    int
		expected int
	}{
		{"Valid limit", 50, 50},
		{"Zero limit", 0, 10},
		{"Negative limit", -5, 10},
		{"Exceeds max", 150, 100},
		{"Max limit", 100, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateLimit(tt.limit)
			if result != tt.expected {
				t.Errorf("ValidateLimit(%d) = %d, want %d", tt.limit, result, tt.expected)
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
