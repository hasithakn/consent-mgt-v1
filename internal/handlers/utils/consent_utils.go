package utils

import (
	"github.com/gin-gonic/gin"
	"github.com/wso2/consent-management-api/internal/models"
	"github.com/wso2/consent-management-api/internal/service"
)

// DeriveConsentStatus derives the consent status from authorization statuses
// Rules:
// - If any auth has status "rejected" -> consent status is "rejected"
// - If any auth has status "revoked" -> consent status is "revoked"
// - If any auth has status "created" -> consent status is "created"
// - If all auths have status "approved" (or empty) -> consent status is "active"
// - For custom states -> TODO: call extension to resolve (for now treat as active)
// Priority: rejected > revoked > created > custom/active
// DeriveConsentStatus derives the consent status from authorization statuses
// If existingStatus is provided and a custom authorization status is encountered,
// the existing status is preserved instead of defaulting to ACTIVE
func DeriveConsentStatus(authResources []models.ConsentAuthResourceCreateRequest, existingStatus string) string {
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
			// For now, we'll preserve existing status or default to created if no existing status
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

	// Priority: rejected > revoked > created > custom (preserve existing) > active
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
		// If we have an existing status, preserve it; otherwise default to created
		if existingStatus != "" {
			return existingStatus
		}
		return string(models.ConsentStatusCreated)
	}

	// All approved or empty -> active
	return string(models.ConsentStatusActive)
}

// BuildEnrichedConsentAPIResponse creates a consent information map matching GET /consents/{consentId} response
// with enriched consent purpose details from the consent purpose service.
// This reuses the ToAPIResponse() method and only adds enrichment of consent purposes.
func BuildEnrichedConsentAPIResponse(c *gin.Context, purposeService *service.ConsentPurposeService, consent *models.ConsentResponse, orgID string) *models.ConsentAPIResponse {
	if consent == nil {
		return nil
	}

	// Use ToAPIResponse to build the complete base response structure
	apiResponse := consent.ToAPIResponse()

	// Enrich consent purposes with full purpose details (type, description, attributes)
	if purposeService != nil && len(apiResponse.ConsentPurpose) > 0 {
		enrichedPurposes := make([]models.ConsentPurposeItem, 0, len(apiResponse.ConsentPurpose))

		for _, cp := range apiResponse.ConsentPurpose {
			// Convert base purpose to enriched purpose
			enrichedPurpose := cp

			// Fetch full purpose details from consent purpose service
			if cp.Name != "" {
				purpose, err := purposeService.GetPurposeByName(c.Request.Context(), cp.Name, orgID)
				if err == nil && purpose != nil {
					// Enrich with type, description, and attributes from the purpose definition
					enrichedPurpose.Type = &purpose.Type
					enrichedPurpose.Description = purpose.Description

					// Convert map[string]string to map[string]interface{}
					if len(purpose.Attributes) > 0 {
						attrs := make(map[string]interface{}, len(purpose.Attributes))
						for k, v := range purpose.Attributes {
							attrs[k] = v
						}
						enrichedPurpose.Attributes = attrs
					} else {
						enrichedPurpose.Attributes = map[string]interface{}{}
					}
				} else {
					// If we can't fetch the purpose, add empty values for enriched fields
					emptyType := ""
					emptyDesc := ""
					enrichedPurpose.Type = &emptyType
					enrichedPurpose.Description = &emptyDesc
					enrichedPurpose.Attributes = map[string]interface{}{}
				}
			}

			enrichedPurposes = append(enrichedPurposes, enrichedPurpose)
		}

		// Set enriched purposes
		apiResponse.ConsentPurpose = enrichedPurposes
	}

	return apiResponse
}
