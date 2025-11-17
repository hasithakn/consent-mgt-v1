package utils

import (
	"context"

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

// BuildConsentInformation creates a consent information map matching GET /consents/{consentId} response
// with enriched consent purpose details from the consent purpose service
func BuildConsentInformation(c *gin.Context, purposeService *service.ConsentPurposeService, consent *models.ConsentResponse, orgID string) map[string]interface{} {
	if consent == nil {
		return nil
	}

	// Enrich consent purposes with full purpose details
	enrichedPurposes := EnrichConsentPurposes(c.Request.Context(), purposeService, consent.ConsentPurpose, orgID)

	// Convert consent to API response to get the correct field names
	apiResponse := consent.ToAPIResponse()

	// Build the consent information matching GET /consents/{id} response structure
	consentInfo := map[string]interface{}{
		"id":                         apiResponse.ID,
		"status":                     apiResponse.Status,
		"type":                       apiResponse.Type,
		"clientId":                   apiResponse.ClientID,
		"consentPurpose":             enrichedPurposes,
		"createdTime":                apiResponse.CreatedTime,
		"updatedTime":                apiResponse.UpdatedTime,
		"validityTime":               apiResponse.ValidityTime,
		"recurringIndicator":         apiResponse.RecurringIndicator,
		"frequency":                  apiResponse.Frequency,
		"dataAccessValidityDuration": apiResponse.DataAccessValidityDuration,
		"attributes":                 apiResponse.Attributes,
		"authorizations":             apiResponse.Authorizations,
	}

	// Include modifiedResponse if present
	if len(apiResponse.ModifiedResponse) > 0 {
		consentInfo["modifiedResponse"] = apiResponse.ModifiedResponse
	}

	return consentInfo
}

// EnrichConsentPurposes fetches full purpose details and enriches the consent purposes
func EnrichConsentPurposes(ctx context.Context, purposeService *service.ConsentPurposeService, consentPurposes []models.ConsentPurposeItem, orgID string) []map[string]interface{} {
	enrichedPurposes := make([]map[string]interface{}, 0, len(consentPurposes))

	for _, cp := range consentPurposes {
		enrichedPurpose := map[string]interface{}{
			"name":       cp.Name,
			"value":      cp.Value,
			"isSelected": cp.IsSelected,
		}

		// Fetch full purpose details from consent purpose service
		if purposeService != nil && cp.Name != "" {
			purpose, err := purposeService.GetPurposeByName(ctx, cp.Name, orgID)
			if err == nil && purpose != nil {
				// Add type, description, and attributes from the purpose
				enrichedPurpose["type"] = purpose.Type
				enrichedPurpose["description"] = purpose.Description
				if len(purpose.Attributes) > 0 {
					enrichedPurpose["attributes"] = purpose.Attributes
				} else {
					enrichedPurpose["attributes"] = map[string]interface{}{}
				}
			} else {
				// Log the error for debugging (errors are silently ignored to not break validation)
				// If we can't fetch the purpose, add empty values
				enrichedPurpose["type"] = ""
				enrichedPurpose["description"] = ""
				enrichedPurpose["attributes"] = map[string]interface{}{}
			}
		}

		enrichedPurposes = append(enrichedPurposes, enrichedPurpose)
	}

	return enrichedPurposes
}
