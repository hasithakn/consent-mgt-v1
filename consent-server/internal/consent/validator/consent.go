package validator

import (
	"fmt"

	authvalidator "github.com/wso2/consent-management-api/internal/authresource/validator"
	"github.com/wso2/consent-management-api/internal/consent/model"
	"github.com/wso2/consent-management-api/internal/system/config"
)

// ValidateConsentCreateRequest validates consent creation request
func ValidateConsentCreateRequest(req model.ConsentAPIRequest, clientID, orgID string) error {
	// Required fields
	if req.Type == "" {
		return fmt.Errorf("type is required")
	}
	if clientID == "" {
		return fmt.Errorf("clientID is required")
	}
	if orgID == "" {
		return fmt.Errorf("orgID is required")
	}

	// Validate auth resources (Authorizations field)
	for i, authReq := range req.Authorizations {
		if authReq.Type == "" {
			return fmt.Errorf("authorizations[%d].type is required", i)
		}
		// Status is optional and defaults to "approved" in the ToAuthResourceCreateRequest method
		if authReq.Status != "" {
			if err := authvalidator.ValidateAuthStatus(authReq.Status); err != nil {
				return fmt.Errorf("authorizations[%d]: %w", i, err)
			}
		}
	}

	// Validate validity time if provided
	if req.ValidityTime != nil && *req.ValidityTime < 0 {
		return fmt.Errorf("validityTime must be non-negative")
	}

	// Validate frequency if provided
	if req.Frequency != nil && *req.Frequency < 0 {
		return fmt.Errorf("frequency must be non-negative")
	}

	return nil
}

// ValidateConsentStatus validates consent status value
func ValidateConsentStatus(status string) error {
	cfg := config.Get().Consent
	if !cfg.IsStatusAllowed(config.ConsentStatus(status)) {
		return fmt.Errorf("invalid consent status: %s", status)
	}
	return nil
}

// ValidateConsentUpdateRequest validates consent update request (keeping for future use)
func ValidateConsentUpdateRequest(req model.ConsentAPIUpdateRequest) error {
	// At least one field must be provided
	if req.Type == "" && req.Frequency == nil &&
		req.ValidityTime == nil && req.RecurringIndicator == nil &&
		req.Attributes == nil && len(req.Authorizations) == 0 {
		return fmt.Errorf("at least one field must be provided for update")
	}

	// Validate validity time if provided
	if req.ValidityTime != nil && *req.ValidityTime < 0 {
		return fmt.Errorf("validityTime must be non-negative")
	}

	// Validate frequency if provided
	if req.Frequency != nil && *req.Frequency < 0 {
		return fmt.Errorf("frequency must be non-negative")
	}

	return nil
}
