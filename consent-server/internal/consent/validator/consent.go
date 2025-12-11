package validator

import (
	"fmt"
	"strings"

	authmodel "github.com/wso2/consent-management-api/internal/authresource/model"
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

// ValidateConsentGetRequest validates consent retrieval request parameters
func ValidateConsentGetRequest(consentID, orgID string) error {
	if consentID == "" {
		return fmt.Errorf("consent ID cannot be empty")
	}
	if len(consentID) > 255 {
		return fmt.Errorf("consent ID too long (max 255 characters)")
	}
	if orgID == "" {
		return fmt.Errorf("organization ID cannot be empty")
	}
	if len(orgID) > 255 {
		return fmt.Errorf("organization ID too long (max 255 characters)")
	}
	return nil
}

// DeriveConsentStatusFromAuthState maps an authorization status to a ConsentStatus when possible.
// Returns the derived status and true when derivation succeeded. For unknown states it returns
// empty string and false to indicate that the extension point should be invoked to resolve the final status.
func DeriveConsentStatusFromAuthState(authState string) (config.ConsentStatus, bool) {

	consentConfig := config.Get().Consent

	authStateString := strings.ToLower(strings.TrimSpace(authState))
	if authStateString == "" {
		// default when not defined: treat as approved -> active
		return consentConfig.GetCreatedConsentStatus(), true
	}
	switch authStateString {
	case strings.ToLower(string(consentConfig.GetApprovedAuthStatus())):
		return consentConfig.GetActiveConsentStatus(), true
	case strings.ToLower(string(consentConfig.GetRejectedAuthStatus())):
		return consentConfig.GetRejectedConsentStatus(), true
	case strings.ToLower(string(consentConfig.GetCreatedAuthStatus())):
		return consentConfig.GetCreatedConsentStatus(), true
	default:
		// unknown/custom state - extension should resolve to one of known ConsentStatus values
		return "", false
	}
}

// EvaluateConsentStatus determines the consent status based on authorization resource states.
// This is the main entry point for deriving consent status from auth resources.
func EvaluateConsentStatus(authResources []authmodel.ConsentAuthResourceCreateRequest) string {
	if len(authResources) == 0 {
		// No auth resources - default to created status
		return string(config.Get().Consent.GetCreatedConsentStatus())
	}

	// Derive status from the first authorization resource's status
	// In a multi-auth scenario, you could implement more complex logic (e.g., all must be approved)
	firstAuthStatus := authResources[0].AuthStatus

	derivedStatus, ok := DeriveConsentStatusFromAuthState(firstAuthStatus)
	if !ok {
		// If derivation fails, default to created status
		return string(config.Get().Consent.GetCreatedConsentStatus())
	}

	return string(derivedStatus)
}
