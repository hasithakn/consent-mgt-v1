package models

import (
	"net/http"
	"strings"
)

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

// AuthorizationState represents known authorization states produced by authorizations
type AuthorizationState string

const (
	// AuthStateCreated indicates the authorization is created but not yet approved/rejected
	AuthStateCreated AuthorizationState = "created"
	// AuthStateApproved indicates the authorization was approved
	AuthStateApproved AuthorizationState = "approved"
	// AuthStateRejected indicates the authorization was rejected
	AuthStateRejected AuthorizationState = "rejected"
	// AuthStateCustom indicates a non-standard/custom state which should be resolved by an extension
	AuthStateCustom AuthorizationState = "custom"
)

// ConsentStatus lists allowed consent lifecycle statuses maintained by consent-mgt API
type ConsentStatus string

const (
	ConsentStatusCreated  ConsentStatus = "created"
	ConsentStatusActive   ConsentStatus = "active"
	ConsentStatusRejected ConsentStatus = "rejected"
	ConsentStatusRevoked  ConsentStatus = "revoked"
	ConsentStatusExpired  ConsentStatus = "expired"
)

// DeriveConsentStatusFromAuthState maps an authorization.state value to a ConsentStatus when possible.
// Returns the derived status and true when derivation succeeded. For custom/unknown states it returns
// empty string and false to indicate that the extension point should be invoked to resolve the final status.
func DeriveConsentStatusFromAuthState(authState string) (ConsentStatus, bool) {
	s := strings.ToLower(strings.TrimSpace(authState))
	if s == "" {
		// default when not defined: treat as approved -> active
		return ConsentStatusActive, true
	}
	switch s {
	case string(AuthStateApproved):
		return ConsentStatusActive, true
	case string(AuthStateRejected):
		return ConsentStatusRejected, true
	case string(AuthStateCreated):
		return ConsentStatusCreated, true
	default:
		// unknown/custom state - extension should resolve to one of known ConsentStatus values
		return "", false
	}
}
