package utils

import (
	"github.com/wso2/consent-management-api/internal/models"
)

// DeriveConsentStatus derives the consent status from authorization statuses
// Rules:
// - If any auth has status "rejected" -> consent status is "rejected"
// - If any auth has status "revoked" -> consent status is "revoked"
// - If any auth has status "created" -> consent status is "created"
// - If all auths have status "approved" (or empty) -> consent status is "active"
// - For custom states -> TODO: call extension to resolve (for now treat as active)
// Priority: rejected > revoked > created > custom/active
func DeriveConsentStatus(authResources []models.ConsentAuthResourceCreateRequest) string {
	if len(authResources) == 0 {
		// No authorizations: default to created
		return string(models.ConsentStatusCreated)
	}

	hasRejected := false
	hasRevoked := false
	hasCreated := false
	hasCustom := false

	for _, auth := range authResources {
		status, canDerive := models.DeriveConsentStatusFromAuthState(auth.AuthStatus)

		if !canDerive {
			// Custom/unknown state
			hasCustom = true
			// TODO: Call extension service to resolve custom state to known consent status
			// For now, we'll treat custom states as active after all checks
			continue
		}

		// Check for rejected, revoked, or created states
		if status == models.ConsentStatusRejected {
			hasRejected = true
		} else if status == models.ConsentStatusRevoked {
			hasRevoked = true
		} else if status == models.ConsentStatusCreated {
			hasCreated = true
		}
	}

	// Priority: rejected > revoked > created > custom/active
	if hasRejected {
		return string(models.ConsentStatusRejected)
	}
	if hasRevoked {
		return string(models.ConsentStatusRevoked)
	}
	if hasCreated {
		return string(models.ConsentStatusCreated)
	}
	if hasCustom {
		// TODO: Extension resolution for custom states
		// For now, default to active
		return string(models.ConsentStatusActive)
	}

	// All approved or empty -> active
	return string(models.ConsentStatusActive)
}
