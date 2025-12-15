/*
 * Copyright (c) 2025, WSO2 LLC. (https://www.wso2.com).
 *
 * WSO2 LLC. licenses this file to you under the Apache License,
 * Version 2.0 (the "License"); you may not use this file except
 * in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

package serviceerror

import (
	"github.com/wso2/consent-management-api/internal/system/error/codes"
)

type ServiceErrorType string

const (
	ClientErrorType ServiceErrorType = "client_error"
	ServerErrorType ServiceErrorType = "server_error"
)

// ServiceError represents an error that occurred in the service layer.
// It contains the error code, type, message, and detailed description.
type ServiceError struct {
	Code        string           `json:"code"`        // Error code (e.g., "CSE-4040")
	Type        ServiceErrorType `json:"type"`        // Error type (client_error or server_error)
	Message     string           `json:"message"`     // Human-readable error message
	Description string           `json:"description"` // Detailed error description
}

// Predefined service errors for common scenarios
var (
	InternalServerError = ServiceError{
		Type:        ServerErrorType,
		Code:        codes.InternalServerError,
		Message:     "Internal Server Error",
		Description: "An unexpected error occurred while processing the request",
	}

	DatabaseError = ServiceError{
		Type:        ServerErrorType,
		Code:        codes.DatabaseError,
		Message:     "Database Error",
		Description: "A database error occurred while processing the request",
	}

	InvalidRequestError = ServiceError{
		Type:        ClientErrorType,
		Code:        codes.InvalidRequest,
		Message:     "Invalid Request",
		Description: "The request is invalid or malformed",
	}

	ResourceNotFoundError = ServiceError{
		Type:        ClientErrorType,
		Code:        codes.ResourceNotFound,
		Message:     "Resource Not Found",
		Description: "The requested resource was not found",
	}

	ConflictError = ServiceError{
		Type:        ClientErrorType,
		Code:        codes.ConflictError,
		Message:     "Conflict",
		Description: "The request conflicts with the current state of the resource",
	}

	ValidationError = ServiceError{
		Type:        ClientErrorType,
		Code:        codes.ValidationError,
		Message:     "Validation Error",
		Description: "Request validation failed",
	}
)

// NewServiceError creates a new ServiceError with the specified details.
func NewServiceError(code string, errorType ServiceErrorType, message, description string) *ServiceError {
	return &ServiceError{
		Code:        code,
		Type:        errorType,
		Message:     message,
		Description: description,
	}
}

// CustomServiceError creates a custom service error based on a predefined error with a custom description.
// Deprecated: Use NewServiceError instead for better clarity and consistency.
func CustomServiceError(baseError ServiceError, description string) *ServiceError {
	return &ServiceError{
		Type:        baseError.Type,
		Code:        baseError.Code,
		Message:     baseError.Message,
		Description: description,
	}
}

// Error implements the error interface.
func (e *ServiceError) Error() string {
	return e.Message + ": " + e.Description
}
