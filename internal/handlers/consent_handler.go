package handlers

import (
	"context"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/wso2/consent-management-api/internal/client"
	"github.com/wso2/consent-management-api/internal/models"
	"github.com/wso2/consent-management-api/internal/service"
	"github.com/wso2/consent-management-api/pkg/utils"
)

// ConsentHandler handles consent-related HTTP requests
type ConsentHandler struct {
	consentService        *service.ConsentService
	consentPurposeService *service.ConsentPurposeService
	extensionClient       *client.ExtensionClient
}

// NewConsentHandler creates a new consent handler instance
func NewConsentHandler(consentService *service.ConsentService, consentPurposeService *service.ConsentPurposeService, extensionClient *client.ExtensionClient) *ConsentHandler {
	return &ConsentHandler{
		consentService:        consentService,
		consentPurposeService: consentPurposeService,
		extensionClient:       extensionClient,
	}
}

// validateResolvedConsentPurposes validates that all resolved consent purposes exist in the organization
func (h *ConsentHandler) validateResolvedConsentPurposes(ctx context.Context, purposeNames []string, orgID string) error {
	if len(purposeNames) == 0 {
		return nil
	}

	// Check each purpose name exists in the organization
	for _, purposeName := range purposeNames {
		exists, err := h.consentPurposeService.ExistsByName(ctx, purposeName, orgID)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("consent purpose '%s' does not exist in organization", purposeName)
		}
	}

	return nil
}

// CreateConsent handles POST /consents
func (h *ConsentHandler) CreateConsent(c *gin.Context) {
	// Parse API request body (external format)
	var apiRequest models.ConsentAPIRequest
	if err := c.ShouldBindJSON(&apiRequest); err != nil {
		utils.SendBadRequestError(c, "Invalid request body", err.Error())
		return
	}

	// Convert API request to internal format
	request, err := apiRequest.ToConsentCreateRequest()
	if err != nil {
		utils.SendBadRequestError(c, "Failed to convert request", err.Error())
		return
	}

	// Get orgID and clientID from context (set by middleware)
	orgID := utils.GetOrgIDFromContext(c)
	clientID := utils.GetClientIDFromContext(c)

	// Validate required fields from the request
	if err := utils.ValidateRequired("ConsentType", request.ConsentType); err != nil {
		utils.SendValidationError(c, err.Error())
		return
	}

	// Call pre-create consent extension if configured
	if h.extensionClient != nil {
		// Extract request headers to pass to extension
		headers := make(map[string]string)
		for key, values := range c.Request.Header {
			if len(values) > 0 {
				headers[key] = values[0]
			}
		}

		extResponse, err := h.extensionClient.PreProcessConsentCreation(c.Request.Context(), request, headers)
		if err != nil {
			utils.SendInternalServerError(c, "Extension service error", err.Error())
			return
		}

		// Check if extension returned an error
		if extResponse != nil && extResponse.Status == "ERROR" {
			errorMessage := "Extension validation failed"
			if extResponse.ErrorData != nil {
				if msg, ok := extResponse.ErrorData["errorMessage"].(string); ok {
					errorMessage = msg
				}
			}
			statusCode := 400
			if extResponse.ErrorCode != nil {
				statusCode = *extResponse.ErrorCode
			}
			c.JSON(statusCode, gin.H{
				"error":   errorMessage,
				"details": extResponse.ErrorData,
			})
			return
		}

		// If extension returned modified consent data, use it
		if extResponse != nil && extResponse.Data != nil {
			modifiedRequest := extResponse.Data.ConsentResource.ToConsentCreateRequest()
			if modifiedRequest != nil {
				request = modifiedRequest
			}

			// Validate resolved consent purposes if provided
			if len(extResponse.Data.ResolvedConsentPurposes) > 0 {
				if err := h.validateResolvedConsentPurposes(c.Request.Context(), extResponse.Data.ResolvedConsentPurposes, orgID); err != nil {
					utils.SendBadRequestError(c, "Invalid consent purposes from extension", err.Error())
					return
				}

				// Store resolved consent purposes if provided (to be used after consent creation)
				utils.SetContextValue(c, "resolvedConsentPurposes", extResponse.Data.ResolvedConsentPurposes)
			}
		}
	}

	// Create consent with purposes if resolved from extension
	var consent *models.ConsentResponse

	if resolvedPurposesRaw, exists := c.Get("resolvedConsentPurposes"); exists {
		if purposeNames, ok := resolvedPurposesRaw.([]string); ok && len(purposeNames) > 0 {
			consent, err = h.consentService.CreateConsentWithPurposes(c.Request.Context(), request, clientID, orgID, purposeNames)
		} else {
			consent, err = h.consentService.CreateConsent(c.Request.Context(), request, clientID, orgID)
		}
	} else {
		consent, err = h.consentService.CreateConsent(c.Request.Context(), request, clientID, orgID)
	}

	if err != nil {
		// Check if it's a validation error
		if strings.Contains(err.Error(), "must be") || strings.Contains(err.Error(), "invalid") || strings.Contains(err.Error(), "required") {
			utils.SendBadRequestError(c, "Failed to create consent", err.Error())
			return
		}
		utils.SendInternalServerError(c, "Failed to create consent", err.Error())
		return
	}

	// Convert internal response to API response format
	apiResponse := consent.ToAPIResponse()
	utils.SendCreatedResponse(c, apiResponse)
}

// GetConsent handles GET /consents/{consentId}
func (h *ConsentHandler) GetConsent(c *gin.Context) {
	// Get consent ID from path parameter
	consentID := c.Param("consentId")
	if consentID == "" {
		utils.SendBadRequestError(c, "Consent ID is required", "")
		return
	}

	// Get orgID from context (set by middleware)
	orgID := utils.GetOrgIDFromContext(c)

	// Validate consent ID format
	if err := utils.ValidateConsentID(consentID); err != nil {
		utils.SendValidationError(c, err.Error())
		return
	}

	// Get consent from service
	consent, err := h.consentService.GetConsent(c.Request.Context(), consentID, orgID)
	if err != nil {
		// Check if it's a validation error (invalid ID format)
		if err.Error() == "consent ID cannot be empty" || err.Error() == "consent ID too long (max 255 characters)" ||
			err.Error() == "organization ID cannot be empty" || err.Error() == "organization ID too long (max 255 characters)" {
			utils.SendBadRequestError(c, "Invalid request", err.Error())
			return
		}
		// Check if it's a not found error
		if strings.Contains(err.Error(), "consent not found") {
			utils.SendNotFoundError(c, "Consent not found")
			return
		}
		utils.SendInternalServerError(c, "Failed to retrieve consent", err.Error())
		return
	}

	// Convert to API response format
	apiResponse := consent.ToAPIResponse()
	utils.SendOKResponse(c, apiResponse)
}

// UpdateConsent handles PUT /consents/:consentId - Update an existing consent
func (h *ConsentHandler) UpdateConsent(c *gin.Context) {
	// Get consent ID from path
	consentID := c.Param("consentId")

	// Get orgID from context (set by middleware)
	orgID := utils.GetOrgIDFromContext(c)

	// Parse request body
	var apiRequest models.ConsentAPIUpdateRequest
	if err := c.ShouldBindJSON(&apiRequest); err != nil {
		utils.SendBadRequestError(c, "Invalid request body", err.Error())
		return
	}

	// Convert API request to internal format
	updateRequest, err := apiRequest.ToConsentUpdateRequest()
	if err != nil {
		utils.SendBadRequestError(c, "Invalid request format", err.Error())
		return
	}

	// Call pre-update consent extension if configured
	if h.extensionClient != nil {
		// Extract request headers to pass to extension
		headers := make(map[string]string)
		for key, values := range c.Request.Header {
			if len(values) > 0 {
				headers[key] = values[0]
			}
		}

		extResponse, err := h.extensionClient.PreProcessConsentUpdate(c.Request.Context(), consentID, updateRequest, headers)
		if err != nil {
			utils.SendInternalServerError(c, "Extension service error", err.Error())
			return
		}

		// Check if extension returned an error
		if extResponse != nil && extResponse.Status == "ERROR" {
			errorMessage := "Extension validation failed"
			if extResponse.ErrorData != nil {
				if msg, ok := extResponse.ErrorData["errorMessage"].(string); ok {
					errorMessage = msg
				}
			}
			statusCode := 400
			if extResponse.ErrorCode != nil {
				statusCode = *extResponse.ErrorCode
			}
			c.JSON(statusCode, gin.H{
				"error":   errorMessage,
				"details": extResponse.ErrorData,
			})
			return
		}

		// If extension returned modified consent data, use it
		if extResponse != nil && extResponse.Data != nil {
			modifiedRequest := extResponse.Data.ConsentResource.ToConsentUpdateRequest()
			if modifiedRequest != nil {
				updateRequest = modifiedRequest
			}

			// Validate resolved consent purposes if provided
			if len(extResponse.Data.ResolvedConsentPurposes) > 0 {
				if err := h.validateResolvedConsentPurposes(c.Request.Context(), extResponse.Data.ResolvedConsentPurposes, orgID); err != nil {
					utils.SendBadRequestError(c, "Invalid consent purposes from extension", err.Error())
					return
				}

				// Store resolved consent purposes if provided (to be used after consent update)
				utils.SetContextValue(c, "resolvedConsentPurposes", extResponse.Data.ResolvedConsentPurposes)
			}
		}
	}

	// Update consent with purposes if resolved from extension
	var updatedConsent *models.ConsentResponse

	if resolvedPurposesRaw, exists := c.Get("resolvedConsentPurposes"); exists {
		if purposeNames, ok := resolvedPurposesRaw.([]string); ok {
			updatedConsent, err = h.consentService.UpdateConsentWithPurposes(c.Request.Context(), consentID, orgID, updateRequest, purposeNames)
		} else {
			updatedConsent, err = h.consentService.UpdateConsent(c.Request.Context(), consentID, orgID, updateRequest)
		}
	} else {
		updatedConsent, err = h.consentService.UpdateConsent(c.Request.Context(), consentID, orgID, updateRequest)
	}

	if err != nil {
		// Check if it's a validation error
		if strings.Contains(err.Error(), "cannot be empty") || strings.Contains(err.Error(), "too long") ||
			strings.Contains(err.Error(), "invalid status") || strings.Contains(err.Error(), "must be") ||
			strings.Contains(err.Error(), "invalid") || strings.Contains(err.Error(), "required") {
			utils.SendBadRequestError(c, "Failed to update consent", err.Error())
			return
		}
		// Check if it's a not found error
		if strings.Contains(err.Error(), "consent not found") {
			utils.SendNotFoundError(c, "Consent not found")
			return
		}
		utils.SendInternalServerError(c, "Failed to update consent", err.Error())
		return
	}

	// Convert to API response format
	apiResponse := updatedConsent.ToAPIResponse()
	utils.SendOKResponse(c, apiResponse)
}
