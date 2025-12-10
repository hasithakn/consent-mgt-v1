package validator

import (
	"fmt"

	"github.com/wso2/consent-management-api/internal/authresource/model"
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

// ValidateAuthResourceCreateRequest validates auth resource creation request
func ValidateAuthResourceCreateRequest(req model.ConsentAuthResourceCreateRequest, consentID, orgID string) error {
	if consentID == "" {
		return fmt.Errorf("consentID is required")
	}
	if orgID == "" {
		return fmt.Errorf("orgID is required")
	}
	if req.AuthType == "" {
		return fmt.Errorf("authType is required")
	}
	if req.AuthStatus == "" {
		return fmt.Errorf("authStatus is required")
	}

	// Validate auth type
	if err := ValidateAuthType(req.AuthType); err != nil {
		return err
	}

	// Validate auth status
	if err := ValidateAuthStatus(req.AuthStatus); err != nil {
		return err
	}

	return nil
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

// ValidateAuthResourceUpdateRequest validates auth resource update request
func ValidateAuthResourceUpdateRequest(req model.ConsentAuthResourceUpdateRequest) error {
	// At least one field must be provided
	if req.AuthStatus == "" && req.UserID == nil && req.Resources == nil {
		return fmt.Errorf("at least one field must be provided for update")
	}

	// Validate status if provided
	if req.AuthStatus != "" {
		if err := ValidateAuthStatus(req.AuthStatus); err != nil {
			return err
		}
	}

	return nil
}
