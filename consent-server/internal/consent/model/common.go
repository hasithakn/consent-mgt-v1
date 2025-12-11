package model

import (
	"strings"

	"github.com/wso2/consent-management-api/internal/system/config"
)

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
