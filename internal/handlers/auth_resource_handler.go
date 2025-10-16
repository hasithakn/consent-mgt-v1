package handlers

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/wso2/consent-management-api/internal/models"
	"github.com/wso2/consent-management-api/internal/service"
	"github.com/wso2/consent-management-api/pkg/utils"
)

// AuthResourceHandler handles authorization resource-related HTTP requests
type AuthResourceHandler struct {
	authResourceService *service.AuthResourceService
}

// NewAuthResourceHandler creates a new AuthResourceHandler
func NewAuthResourceHandler(authResourceService *service.AuthResourceService) *AuthResourceHandler {
	return &AuthResourceHandler{
		authResourceService: authResourceService,
	}
}

// CreateAuthResource handles POST /api/v1/consents/{consentId}/authorizations
func (h *AuthResourceHandler) CreateAuthResource(c *gin.Context) {
	consentID := c.Param("consentId")
	orgID := utils.GetOrgIDFromContext(c)

	// Parse request body
	var apiRequest models.AuthorizationAPIRequest
	if err := c.ShouldBindJSON(&apiRequest); err != nil {
		utils.SendBadRequestError(c, "Invalid request payload", err.Error())
		return
	}

	// Convert API format to internal format
	createRequest := apiRequest.ToAuthResourceCreateRequest()

	// Create auth resource
	authResource, err := h.authResourceService.CreateAuthResource(c.Request.Context(), consentID, orgID, createRequest)
	if err != nil {
		// Check if it's a validation error
		if strings.Contains(err.Error(), "consent ID cannot be empty") ||
			strings.Contains(err.Error(), "consent ID too long") ||
			strings.Contains(err.Error(), "organization ID cannot be empty") ||
			strings.Contains(err.Error(), "organization ID too long") ||
			strings.Contains(err.Error(), "invalid") {
			utils.SendBadRequestError(c, "Invalid request", err.Error())
			return
		}
		// Check if consent not found
		if strings.Contains(err.Error(), "consent not found") {
			utils.SendNotFoundError(c, "Consent not found")
			return
		}
		utils.SendInternalServerError(c, "Failed to create authorization resource", err.Error())
		return
	}

	// Convert to API response format
	apiResponse := toAuthResourceAPIResponse(authResource)

	c.JSON(201, apiResponse)
}

// GetAuthResource handles GET /api/v1/consents/{consentId}/authorizations/{authId}
func (h *AuthResourceHandler) GetAuthResource(c *gin.Context) {
	consentID := c.Param("consentId")
	authID := c.Param("authId")
	orgID := utils.GetOrgIDFromContext(c)

	// Validate consent ID
	if err := utils.ValidateConsentID(consentID); err != nil {
		utils.SendBadRequestError(c, "Invalid consent ID", err.Error())
		return
	}

	// Get auth resource
	authResource, err := h.authResourceService.GetAuthResource(c.Request.Context(), authID, orgID)
	if err != nil {
		// Check if it's a validation error
		if strings.Contains(err.Error(), "cannot be empty") ||
			strings.Contains(err.Error(), "too long") ||
			strings.Contains(err.Error(), "invalid") {
			utils.SendBadRequestError(c, "Invalid request", err.Error())
			return
		}
		// Check if not found
		if strings.Contains(err.Error(), "not found") {
			utils.SendNotFoundError(c, "Authorization resource not found")
			return
		}
		utils.SendInternalServerError(c, "Failed to retrieve authorization resource", err.Error())
		return
	}

	// Verify that the auth resource belongs to the specified consent
	if authResource.ConsentID != consentID {
		utils.SendNotFoundError(c, "Authorization resource not found for this consent")
		return
	}

	// Convert to API response format
	apiResponse := toAuthResourceAPIResponse(authResource)

	c.JSON(200, apiResponse)
}

// UpdateAuthResource handles PUT /api/v1/consents/{consentId}/authorizations/{authId}
func (h *AuthResourceHandler) UpdateAuthResource(c *gin.Context) {
	consentID := c.Param("consentId")
	authID := c.Param("authId")
	orgID := utils.GetOrgIDFromContext(c)

	// Validate consent ID
	if err := utils.ValidateConsentID(consentID); err != nil {
		utils.SendBadRequestError(c, "Invalid consent ID", err.Error())
		return
	}

	// Get existing auth resource first to verify it belongs to the specified consent
	existingAuthResource, err := h.authResourceService.GetAuthResource(c.Request.Context(), authID, orgID)
	if err != nil {
		// Check if it's a validation error
		if strings.Contains(err.Error(), "cannot be empty") ||
			strings.Contains(err.Error(), "too long") ||
			strings.Contains(err.Error(), "invalid") {
			utils.SendBadRequestError(c, "Invalid request", err.Error())
			return
		}
		// Check if not found
		if strings.Contains(err.Error(), "not found") {
			utils.SendNotFoundError(c, "Authorization resource not found")
			return
		}
		utils.SendInternalServerError(c, "Failed to retrieve authorization resource", err.Error())
		return
	}

	// Verify that the auth resource belongs to the specified consent
	if existingAuthResource.ConsentID != consentID {
		utils.SendNotFoundError(c, "Authorization resource not found for this consent")
		return
	}

	// Parse request body
	var apiRequest models.AuthorizationAPIUpdateRequest
	if err := c.ShouldBindJSON(&apiRequest); err != nil {
		utils.SendBadRequestError(c, "Invalid request payload", err.Error())
		return
	}

	// Convert API format to internal format
	updateRequest := apiRequest.ToAuthResourceUpdateRequest()

	// Update auth resource
	authResource, err := h.authResourceService.UpdateAuthResource(c.Request.Context(), authID, orgID, updateRequest)
	if err != nil {
		// Check if it's a validation error
		if strings.Contains(err.Error(), "cannot be empty") ||
			strings.Contains(err.Error(), "too long") ||
			strings.Contains(err.Error(), "invalid") {
			utils.SendBadRequestError(c, "Invalid request", err.Error())
			return
		}
		// Check if not found
		if strings.Contains(err.Error(), "not found") {
			utils.SendNotFoundError(c, "Authorization resource not found")
			return
		}
		utils.SendInternalServerError(c, "Failed to update authorization resource", err.Error())
		return
	}

	// Convert to API response format
	apiResponse := toAuthResourceAPIResponse(authResource)

	c.JSON(200, apiResponse)
}

// toAuthResourceAPIResponse converts internal auth resource response to API format
func toAuthResourceAPIResponse(authResource *models.ConsentAuthResourceResponse) *models.AuthorizationAPIResponse {
	// Initialize resource with empty object if nil
	resource := authResource.Resource
	if resource == nil {
		resource = make(map[string]interface{})
	}

	return &models.AuthorizationAPIResponse{
		ID:          authResource.AuthID,
		UserID:      authResource.UserID,
		Type:        authResource.AuthType,
		Status:      authResource.AuthStatus,
		UpdatedTime: authResource.UpdatedTime,
		Resource:    resource,
	}
}
