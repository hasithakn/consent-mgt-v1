package utils

import (
	"fmt"
	"net/http"

	"github.com/wso2/consent-management-api/internal/system/constants"
)

// Validate orgID and clientID in the request headers.
func ValidateOrgIdAndClientIdIsPresent(r *http.Request) error {
	orgID := r.Header.Get(constants.HeaderOrgID)
	clientID := r.Header.Get(constants.HeaderTPPClientID)

	if err := ValidateOrgID(orgID); err != nil {
		return err
	}
	if err := ValidateClientID(clientID); err != nil {
		return err
	}
	return nil
}

// ValidateOrgID validates organization ID
func ValidateOrgID(orgID string) error {
	if orgID == "" {
		return fmt.Errorf("organization ID is required")
	}
	if len(orgID) > 255 {
		return fmt.Errorf("organization ID too long (max 255 chars)")
	}
	return nil
}

// ValidateClientID validates client ID
func ValidateClientID(clientID string) error {
	if clientID == "" {
		return fmt.Errorf("client ID is required")
	}
	if len(clientID) > 255 {
		return fmt.Errorf("client ID too long (max 255 chars)")
	}
	return nil
}

// ValidateRequired validates a field is not empty
func ValidateRequired(fieldName, value string) error {
	if value == "" {
		return fmt.Errorf("%s is required", fieldName)
	}
	return nil
}

// ValidatePagination validates limit and offset
func ValidatePagination(limit, offset int) error {
	if limit < 1 || limit > 100 {
		return fmt.Errorf("limit must be between 1 and 100")
	}
	if offset < 0 {
		return fmt.Errorf("offset must be non-negative")
	}
	return nil
}

// ValidateUUID validates UUID format using existing IsValidUUID
func ValidateUUID(id string) error {
	if !IsValidUUID(id) {
		return fmt.Errorf("invalid UUID format: %s", id)
	}
	return nil
}

// ValidateConsentID validates consent ID format
func ValidateConsentID(consentID string) error {
	if err := ValidateRequired("consentID", consentID); err != nil {
		return err
	}
	if len(consentID) > 100 {
		return fmt.Errorf("consent ID too long (max 100 chars)")
	}
	return ValidateUUID(consentID)
}
