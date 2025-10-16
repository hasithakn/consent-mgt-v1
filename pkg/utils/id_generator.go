package utils

import (
	"github.com/google/uuid"
)

// GenerateID generates a new UUID for consent, auth, or audit IDs
func GenerateID() string {
	return uuid.New().String()
}

// GenerateConsentID generates a unique consent ID
func GenerateConsentID() string {
	return "CONSENT-" + uuid.New().String()
}

// GenerateAuthID generates a unique authorization ID
func GenerateAuthID() string {
	return "AUTH-" + uuid.New().String()
}

// GenerateAuditID generates a unique status audit ID
func GenerateAuditID() string {
	return "AUDIT-" + uuid.New().String()
}

// IsValidUUID checks if a string is a valid UUID
func IsValidUUID(id string) bool {
	_, err := uuid.Parse(id)
	return err == nil
}
