package handlers

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/wso2/consent-management-api/internal/config"
	extensionclient "github.com/wso2/consent-management-api/internal/extension-client"
	handlerutils "github.com/wso2/consent-management-api/internal/handlers/utils"
	"github.com/wso2/consent-management-api/internal/models"
	"github.com/wso2/consent-management-api/internal/service"
	"github.com/wso2/consent-management-api/internal/utils"
)

// ConsentHandler handles consent-related HTTP requests
type ConsentHandler struct {
	consentService        *service.ConsentService
	consentPurposeService *service.ConsentPurposeService
	extensionClient       *extensionclient.ExtensionClient
}

// NewConsentHandler creates a new consent handler instance
func NewConsentHandler(consentService *service.ConsentService, consentPurposeService *service.ConsentPurposeService, extensionClient *extensionclient.ExtensionClient) *ConsentHandler {
	return &ConsentHandler{
		consentService:        consentService,
		consentPurposeService: consentPurposeService,
		extensionClient:       extensionClient,
	}
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

	// Call pre-create consent extension if configured and enabled
	cfg := config.Get()
	if cfg != nil && cfg.Extension.Enabled && h.extensionClient != nil {
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
		}
	}

	// Derive consent status from authorization statuses
	request.CurrentStatus = handlerutils.DeriveConsentStatus(request.AuthResources)

	// Create consent with purpose validation
	var consent *models.ConsentResponse
	var consentErr error
	if len(request.ConsentPurpose) > 0 {
		consent, consentErr = h.consentService.CreateConsentWithPurposes(c.Request.Context(), request, clientID, orgID, request.ConsentPurpose)
	} else {
		consent, consentErr = h.consentService.CreateConsent(c.Request.Context(), request, clientID, orgID)
	}

	if consentErr != nil {
		// Check if it's a validation error
		if strings.Contains(consentErr.Error(), "must be") || strings.Contains(consentErr.Error(), "invalid") || strings.Contains(consentErr.Error(), "required") || strings.Contains(consentErr.Error(), "not found") {
			utils.SendBadRequestError(c, "Failed to create consent", consentErr.Error())
			return
		}
		utils.SendInternalServerError(c, "Failed to create consent", consentErr.Error())
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

	// Call pre-update consent extension if configured and enabled
	cfg := config.Get()
	if cfg != nil && cfg.Extension.Enabled && h.extensionClient != nil {
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
		}
	}

	// Derive consent status from authorization statuses
	updateRequest.CurrentStatus = handlerutils.DeriveConsentStatus(updateRequest.AuthResources)

	// Update consent with purposes from request body
	var updatedConsent *models.ConsentResponse
	if len(updateRequest.ConsentPurpose) > 0 {
		updatedConsent, err = h.consentService.UpdateConsentWithPurposes(c.Request.Context(), consentID, orgID, updateRequest, updateRequest.ConsentPurpose)
	} else {
		updatedConsent, err = h.consentService.UpdateConsent(c.Request.Context(), consentID, orgID, updateRequest)
	}

	if err != nil {
		// Check if it's a validation error
		if strings.Contains(err.Error(), "cannot be empty") || strings.Contains(err.Error(), "too long") ||
			strings.Contains(err.Error(), "invalid status") || strings.Contains(err.Error(), "must be") ||
			strings.Contains(err.Error(), "invalid") || strings.Contains(err.Error(), "required") ||
			strings.Contains(err.Error(), "purposes not found") {
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

// Validate handles POST /validate
func (h *ConsentHandler) Validate(c *gin.Context) {
	var req models.ValidateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response := models.ValidateResponse{
			IsValid:          false,
			ModifiedPayload:  nil,
			ErrorCode:        400,
			ErrorMessage:     "invalid_request",
			ErrorDescription: "Invalid request body: " + err.Error(),
		}
		c.JSON(200, response)
		return
	}

	// Basic validation
	if req.ConsentID == "" {
		response := models.ValidateResponse{
			IsValid:          false,
			ModifiedPayload:  nil,
			ErrorCode:        400,
			ErrorMessage:     "invalid_request",
			ErrorDescription: "consentId is required",
		}
		c.JSON(200, response)
		return
	}

	if err := utils.ValidateConsentID(req.ConsentID); err != nil {
		response := models.ValidateResponse{
			IsValid:          false,
			ModifiedPayload:  nil,
			ErrorCode:        400,
			ErrorMessage:     "invalid_request",
			ErrorDescription: "Invalid consentId: " + err.Error(),
		}
		c.JSON(200, response)
		return
	}

	if req.UserID == "" {
		response := models.ValidateResponse{
			IsValid:          false,
			ModifiedPayload:  nil,
			ErrorCode:        400,
			ErrorMessage:     "invalid_request",
			ErrorDescription: "userId is required",
		}
		c.JSON(200, response)
		return
	}

	// Get orgID from context
	orgID := utils.GetOrgIDFromContext(c)

	// Retrieve the consent to validate
	consent, err := h.consentService.GetConsent(c.Request.Context(), req.ConsentID, orgID)
	if err != nil {
		// Check if consent not found
		if strings.Contains(err.Error(), "consent not found") {
			response := models.ValidateResponse{
				IsValid:          false,
				ModifiedPayload:  nil,
				ErrorCode:        404,
				ErrorMessage:     "consent_not_found",
				ErrorDescription: "Consent not found",
			}
			c.JSON(200, response)
			return
		}

		// Internal error
		response := models.ValidateResponse{
			IsValid:          false,
			ModifiedPayload:  nil,
			ErrorCode:        500,
			ErrorMessage:     "internal_error",
			ErrorDescription: "Failed to retrieve consent: " + err.Error(),
		}
		c.JSON(200, response)
		return
	}

	// Get the active status from config
	cfg := config.Get()
	if cfg == nil {
		response := models.ValidateResponse{
			IsValid:          false,
			ModifiedPayload:  nil,
			ErrorCode:        500,
			ErrorMessage:     "internal_error",
			ErrorDescription: "Configuration not loaded",
		}
		c.JSON(200, response)
		return
	}

	// Check if consent is in active status (config-based)
	if !cfg.Consent.IsActiveStatus(consent.CurrentStatus) {
		response := models.ValidateResponse{
			IsValid:            false,
			ModifiedPayload:    nil,
			ErrorCode:          401,
			ErrorMessage:       "invalid_consent_status",
			ErrorDescription:   fmt.Sprintf("Consent is not in active state. Current status: %s, Expected: %s", consent.CurrentStatus, cfg.Consent.StatusMappings.ActiveStatus),
			ConsentInformation: handlerutils.BuildEnrichedConsentAPIResponse(c, h.consentPurposeService, consent, orgID),
		}
		c.JSON(200, response)
		return
	}

	// Check if consent has expired based on validityTime
	if consent.ValidityTime != nil && utils.IsExpired(*consent.ValidityTime) {
		// Consent has expired - update the status to expired in DB
		expiredStatus := cfg.Consent.StatusMappings.ExpiredStatus

		// Use the dedicated UpdateConsentStatus method which safely updates only the status
		// without needing to provide the full consent payload
		actionBy := consent.ClientID
		reason := "Consent expired based on validity time"
		updatedConsent, err := h.consentService.UpdateConsentStatus(
			c.Request.Context(),
			req.ConsentID,
			orgID,
			expiredStatus,
			actionBy,
			reason,
		)
		if err != nil {
			// Log the error but continue with the expired status response
			response := models.ValidateResponse{
				IsValid:            false,
				ModifiedPayload:    nil,
				ErrorCode:          401,
				ErrorMessage:       "consent_expired",
				ErrorDescription:   fmt.Sprintf("Consent has expired. Failed to update status: %s", err.Error()),
				ConsentInformation: handlerutils.BuildEnrichedConsentAPIResponse(c, h.consentPurposeService, consent, orgID),
			}
			c.JSON(200, response)
			return
		}

		// Return expired response with updated consent data
		response := models.ValidateResponse{
			IsValid:            false,
			ModifiedPayload:    nil,
			ErrorCode:          401,
			ErrorMessage:       "consent_expired",
			ErrorDescription:   fmt.Sprintf("Consent has expired. Status updated to: %s", expiredStatus),
			ConsentInformation: handlerutils.BuildEnrichedConsentAPIResponse(c, h.consentPurposeService, updatedConsent, orgID),
		}
		c.JSON(200, response)
		return
	}

	// TODO: Add more sophisticated validation logic here:
	// - Check if the electedResource matches consent permissions
	// - Validate resource params against consent scope
	// - Check user authorization details

	// Return success with full consent data
	response := models.ValidateResponse{
		IsValid:            true,
		ModifiedPayload:    nil,
		ConsentInformation: handlerutils.BuildEnrichedConsentAPIResponse(c, h.consentPurposeService, consent, orgID),
	}
	c.JSON(200, response)
}

// SearchConsentsByAttribute handles GET /consents/attributes
func (h *ConsentHandler) SearchConsentsByAttribute(c *gin.Context) {
	// Get query parameters
	key := c.Query("key")
	value := c.Query("value")

	// Validate that key is provided
	if key == "" {
		utils.SendBadRequestError(c, "Invalid request", "key parameter is required")
		return
	}

	// Get orgID from context
	orgID := utils.GetOrgIDFromContext(c)

	// Search for consent IDs
	consentIDs, err := h.consentService.SearchConsentIDsByAttribute(c.Request.Context(), key, value, orgID)
	if err != nil {
		utils.SendInternalServerError(c, "Failed to search consents by attribute", err.Error())
		return
	}

	// Build response
	response := models.ConsentAttributeSearchResponse{
		ConsentIDs: consentIDs,
		Count:      len(consentIDs),
	}

	utils.SendOKResponse(c, response)
}

// RevokeConsent handles PUT /consents/:consentId/revoke
func (h *ConsentHandler) RevokeConsent(c *gin.Context) {
	// Get consentId from path parameter
	consentID := c.Param("consentId")
	if consentID == "" {
		utils.SendBadRequestError(c, "Invalid request", "consentId is required")
		return
	}

	// Parse request body
	var request models.ConsentRevokeRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		utils.SendBadRequestError(c, "Invalid request body", err.Error())
		return
	}

	// Get orgID from context
	orgID := utils.GetOrgIDFromContext(c)

	// Validate required fields
	if err := utils.ValidateRequired("actionBy", request.ActionBy); err != nil {
		utils.SendValidationError(c, err.Error())
		return
	}

	// Call the service to revoke the consent
	response, err := h.consentService.RevokeConsent(c.Request.Context(), consentID, orgID, &request)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			utils.SendNotFoundError(c, "Consent not found")
			return
		}
		utils.SendInternalServerError(c, "Failed to revoke consent", err.Error())
		return
	}

	utils.SendOKResponse(c, response)
}

// DeleteConsent handles DELETE /consents/:consentId
func (h *ConsentHandler) DeleteConsent(c *gin.Context) {
	// Get consentID from path parameter
	consentID := c.Param("consentId")

	// Get orgID from context (set by middleware)
	orgID := utils.GetOrgIDFromContext(c)

	// Call the service to delete the consent
	err := h.consentService.DeleteConsent(c.Request.Context(), consentID, orgID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			utils.SendNotFoundError(c, "Consent not found")
			return
		}
		utils.SendInternalServerError(c, "Failed to delete consent", err.Error())
		return
	}

	// Return 204 No Content on successful deletion
	c.Status(204)
}
