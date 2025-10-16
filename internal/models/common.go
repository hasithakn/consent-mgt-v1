package models

import "net/http"

// ErrorResponse represents a standard error response
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// NewErrorResponse creates a new error response
func NewErrorResponse(code, message, details string) *ErrorResponse {
	return &ErrorResponse{
		Code:    code,
		Message: message,
		Details: details,
	}
}

// Common error codes
const (
	ErrCodeBadRequest           = "BAD_REQUEST"
	ErrCodeUnauthorized         = "UNAUTHORIZED"
	ErrCodeForbidden            = "FORBIDDEN"
	ErrCodeNotFound             = "NOT_FOUND"
	ErrCodeConflict             = "CONFLICT"
	ErrCodeInternalError        = "INTERNAL_ERROR"
	ErrCodeDatabaseError        = "DATABASE_ERROR"
	ErrCodeValidationError      = "VALIDATION_ERROR"
	ErrCodeExtensionError       = "EXTENSION_ERROR"
	ErrCodeConsentNotFound      = "CONSENT_NOT_FOUND"
	ErrCodeAuthResourceNotFound = "AUTH_RESOURCE_NOT_FOUND"
	ErrCodeFileNotFound         = "FILE_NOT_FOUND"
	ErrCodeInvalidStatus        = "INVALID_STATUS"
)

// HTTPStatusForErrorCode returns the appropriate HTTP status code for an error code
func HTTPStatusForErrorCode(code string) int {
	switch code {
	case ErrCodeBadRequest, ErrCodeValidationError:
		return http.StatusBadRequest
	case ErrCodeUnauthorized:
		return http.StatusUnauthorized
	case ErrCodeForbidden:
		return http.StatusForbidden
	case ErrCodeNotFound, ErrCodeConsentNotFound, ErrCodeAuthResourceNotFound, ErrCodeFileNotFound:
		return http.StatusNotFound
	case ErrCodeConflict:
		return http.StatusConflict
	case ErrCodeInternalError, ErrCodeDatabaseError, ErrCodeExtensionError:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}

// SuccessResponse represents a standard success response
type SuccessResponse struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// NewSuccessResponse creates a new success response
func NewSuccessResponse(message string, data interface{}) *SuccessResponse {
	return &SuccessResponse{
		Message: message,
		Data:    data,
	}
}
