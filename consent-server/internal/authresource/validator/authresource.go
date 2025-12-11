package validator

import (
	"fmt"

	"github.com/wso2/consent-management-api/internal/authresource/model"
	"github.com/wso2/consent-management-api/internal/system/config"
)

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

	// Validate auth status
	if err := ValidateAuthStatus(req.AuthStatus); err != nil {
		return err
	}

	return nil
}

// ValidateAuthStatus validates authorization status
func ValidateAuthStatus(status string) error {
	cfg := config.Get().Consent
	if cfg.AuthStatusMappings.SystemExpiredState == status ||
		cfg.AuthStatusMappings.SystemRevokedState == status {
		return fmt.Errorf("invalid auth status: %s", status)
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
