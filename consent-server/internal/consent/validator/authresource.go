package validator

import (
	"fmt"
)

var validAuthStatuses = map[string]bool{
	"AUTHORIZED":  true,
	"REJECTED":    true,
	"REVOKED":     true,
	"SYS_REVOKED": true,
}

var validAuthTypes = map[string]bool{
	"accounts":       true,
	"payments":       true,
	"funds-confirms": true,
}

// ValidateAuthType validates authorization type
func ValidateAuthType(authType string) error {
	if !validAuthTypes[authType] {
		return fmt.Errorf("invalid auth type: %s (valid: accounts, payments, funds-confirms)", authType)
	}
	return nil
}

// ValidateAuthStatus validates authorization status
func ValidateAuthStatus(status string) error {
	if !validAuthStatuses[status] {
		return fmt.Errorf("invalid auth status: %s (valid: AUTHORIZED, REJECTED, REVOKED, SYS_REVOKED)", status)
	}
	return nil
}
