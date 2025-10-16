package utils

import (
	"fmt"
	"regexp"
	"strings"
)

// ValidateConsentID validates consent ID format
func ValidateConsentID(consentID string) error {
	if consentID == "" {
		return fmt.Errorf("consent ID cannot be empty")
	}
	if len(consentID) > 255 {
		return fmt.Errorf("consent ID too long (max 255 characters)")
	}
	return nil
}

// ValidateClientID validates client ID format
func ValidateClientID(clientID string) error {
	if clientID == "" {
		return fmt.Errorf("client ID cannot be empty")
	}
	if len(clientID) > 255 {
		return fmt.Errorf("client ID too long (max 255 characters)")
	}
	return nil
}

// ValidateOrgID validates organization ID
func ValidateOrgID(orgID string) error {
	if orgID == "" {
		return fmt.Errorf("organization ID cannot be empty")
	}
	if len(orgID) > 255 {
		return fmt.Errorf("organization ID too long (max 255 characters)")
	}
	return nil
}

// ValidateStatus validates consent status
func ValidateStatus(status string) error {
	if status == "" {
		return fmt.Errorf("status cannot be empty")
	}

	validStatuses := map[string]bool{
		"CREATED":               true,
		"awaitingAuthorization": true,
		"AUTHORIZED":            true,
		"ACTIVE":                true,
		"REJECTED":              true,
		"REVOKED":               true,
		"EXPIRED":               true,
	}

	if !validStatuses[status] {
		return fmt.Errorf("invalid status: %s", status)
	}

	return nil
}

// ValidateConsentType validates consent type
func ValidateConsentType(consentType string) error {
	if consentType == "" {
		return fmt.Errorf("consent type cannot be empty")
	}
	if len(consentType) > 64 {
		return fmt.Errorf("consent type too long (max 64 characters)")
	}
	return nil
}

// ValidateEmail validates email format
func ValidateEmail(email string) error {
	if email == "" {
		return fmt.Errorf("email cannot be empty")
	}

	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(email) {
		return fmt.Errorf("invalid email format")
	}

	return nil
}

// SanitizeString removes dangerous characters from user input
func SanitizeString(input string) string {
	// Remove null bytes
	input = strings.ReplaceAll(input, "\x00", "")
	// Trim whitespace
	input = strings.TrimSpace(input)
	return input
}

// ValidateLimit validates pagination limit
func ValidateLimit(limit int) int {
	if limit <= 0 {
		return 20 // Default limit
	}
	if limit > 100 {
		return 100 // Max limit
	}
	return limit
}

// ValidateOffset validates pagination offset
func ValidateOffset(offset int) int {
	if offset < 0 {
		return 0
	}
	return offset
}

// ValidateRequired validates that a field is not empty
func ValidateRequired(fieldName, value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s is required", fieldName)
	}
	return nil
}

// ValidateMaxLength validates maximum string length
func ValidateMaxLength(fieldName, value string, maxLength int) error {
	if len(value) > maxLength {
		return fmt.Errorf("%s exceeds maximum length of %d characters", fieldName, maxLength)
	}
	return nil
}

// ValidateMinLength validates minimum string length
func ValidateMinLength(fieldName, value string, minLength int) error {
	if len(value) < minLength {
		return fmt.Errorf("%s must be at least %d characters", fieldName, minLength)
	}
	return nil
}

// IsAlphanumeric checks if a string contains only alphanumeric characters
func IsAlphanumeric(s string) bool {
	for _, char := range s {
		if !((char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9')) {
			return false
		}
	}
	return true
}
