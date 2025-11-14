package utils

import (
	"encoding/json"

	"github.com/sirupsen/logrus"
	"github.com/wso2/consent-management-api/internal/models"
)

// BuildConsentResponse builds a ConsentResponse from consent data and related entities
func BuildConsentResponse(
	consent *models.Consent,
	attributes map[string]string,
	authResources []models.ConsentAuthResource,
	purposeMappings []models.ConsentPurposeMapping,
	logger *logrus.Logger,
) *models.ConsentResponse {
	// Build ConsentPurpose array from mappings
	var consentPurpose []models.ConsentPurposeItem
	if len(purposeMappings) > 0 {
		consentPurpose = make([]models.ConsentPurposeItem, len(purposeMappings))
		for i, mapping := range purposeMappings {
			consentPurpose[i] = models.ConsentPurposeItem{
				Name:       mapping.Name,
				Value:      mapping.Value,
				IsSelected: &mapping.IsSelected,
			}
		}
	}

	// Convert auth resources to response format
	var authResourceResponses []models.ConsentAuthResource
	if authResources != nil {
		authResourceResponses = make([]models.ConsentAuthResource, len(authResources))
		for i, ar := range authResources {
			authResourceResponses[i] = ar
			// Unmarshal resources if present
			if ar.Resources != nil && *ar.Resources != "" {
				var resources interface{}
				if err := json.Unmarshal([]byte(*ar.Resources), &resources); err != nil {
					logger.WithError(err).Warn("Failed to unmarshal resources")
				} else {
					authResourceResponses[i].ResourceObj = resources
				}
			}
		}
	}

	return &models.ConsentResponse{
		ConsentID:                  consent.ConsentID,
		ConsentPurpose:             consentPurpose,
		CreatedTime:                consent.CreatedTime,
		UpdatedTime:                consent.UpdatedTime,
		ClientID:                   consent.ClientID,
		ConsentType:                consent.ConsentType,
		CurrentStatus:              consent.CurrentStatus,
		ConsentFrequency:           consent.ConsentFrequency,
		ValidityTime:               consent.ValidityTime,
		RecurringIndicator:         consent.RecurringIndicator,
		DataAccessValidityDuration: consent.DataAccessValidityDuration,
		OrgID:                      consent.OrgID,
		Attributes:                 attributes,
		AuthResources:              authResourceResponses,
	}
}
