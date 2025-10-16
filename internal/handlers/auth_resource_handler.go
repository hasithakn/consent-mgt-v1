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
