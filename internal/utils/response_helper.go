package utils

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/wso2/consent-management-api/internal/models"
)

// SendSuccessResponse sends a successful JSON response
func SendSuccessResponse(c *gin.Context, statusCode int, data interface{}) {
	c.JSON(statusCode, data)
}

// SendErrorResponse sends an error JSON response
func SendErrorResponse(c *gin.Context, statusCode int, errCode, message, details string) {
	c.JSON(statusCode, models.ErrorResponse{
		Code:    errCode,
		Message: message,
		Details: details,
	})
}

// SendCreatedResponse sends a 201 Created response
func SendCreatedResponse(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, data)
}

// SendOKResponse sends a 200 OK response
func SendOKResponse(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, data)
}

// SendNoContentResponse sends a 204 No Content response
func SendNoContentResponse(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

// SendBadRequestError sends a 400 Bad Request error
func SendBadRequestError(c *gin.Context, message, details string) {
	SendErrorResponse(c, http.StatusBadRequest, models.ErrCodeBadRequest, message, details)
}

// SendUnauthorizedError sends a 401 Unauthorized error
func SendUnauthorizedError(c *gin.Context, message string) {
	SendErrorResponse(c, http.StatusUnauthorized, models.ErrCodeUnauthorized, message, "")
}

// SendForbiddenError sends a 403 Forbidden error
func SendForbiddenError(c *gin.Context, message string) {
	SendErrorResponse(c, http.StatusForbidden, models.ErrCodeForbidden, message, "")
}

// SendNotFoundError sends a 404 Not Found error
func SendNotFoundError(c *gin.Context, message string) {
	SendErrorResponse(c, http.StatusNotFound, models.ErrCodeNotFound, message, "")
}

// SendConflictError sends a 409 Conflict error
func SendConflictError(c *gin.Context, message string) {
	SendErrorResponse(c, http.StatusConflict, models.ErrCodeConflict, message, "")
}

// SendInternalServerError sends a 500 Internal Server Error
func SendInternalServerError(c *gin.Context, message, details string) {
	SendErrorResponse(c, http.StatusInternalServerError, models.ErrCodeInternalError, message, details)
}

// SendValidationError sends a validation error response
func SendValidationError(c *gin.Context, details string) {
	SendErrorResponse(c, http.StatusBadRequest, models.ErrCodeValidationError, "Validation failed", details)
}

// GetOrgIDFromContext extracts organization ID from context
func GetOrgIDFromContext(c *gin.Context) string {
	orgID, exists := c.Get("orgID")
	if !exists {
		return "DEFAULT_ORG"
	}
	return orgID.(string)
}

// GetClientIDFromContext extracts client ID from context
func GetClientIDFromContext(c *gin.Context) string {
	clientID, exists := c.Get("clientID")
	if !exists {
		return ""
	}
	return clientID.(string)
}

// GetUserIDFromContext extracts user ID from context
func GetUserIDFromContext(c *gin.Context) string {
	userID, exists := c.Get("userID")
	if !exists {
		return ""
	}
	return userID.(string)
}

// GetCorrelationIDFromContext extracts correlation ID from context
func GetCorrelationIDFromContext(c *gin.Context) string {
	correlationID, exists := c.Get("correlationID")
	if !exists {
		return GenerateID()
	}
	return correlationID.(string)
}

// SetContextValue sets a value in the Gin context
func SetContextValue(c *gin.Context, key string, value interface{}) {
	c.Set(key, value)
}
