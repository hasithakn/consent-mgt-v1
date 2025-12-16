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

package utils

import (
	"encoding/json"
	"net/http"

	"github.com/wso2/consent-management-api/internal/system/constants"
	"github.com/wso2/consent-management-api/internal/system/error/apierror"
	"github.com/wso2/consent-management-api/internal/system/error/codes"
	"github.com/wso2/consent-management-api/internal/system/error/serviceerror"
	"github.com/wso2/consent-management-api/internal/system/log"
)

func DecodeJSONBody(r *http.Request, v interface{}) error {
	return json.NewDecoder(r.Body).Decode(v)
}

func JSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// WriteJSONError writes a JSON error response with the new format.
// Deprecated: Use SendError instead which provides better error handling with trace IDs.
func WriteJSONError(w http.ResponseWriter, code, description string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{
		"error":             code,
		"error_description": description,
	})
}

// SendError writes a ServiceError as an HTTP response with appropriate status code and trace ID.
// This function extracts the trace ID from the request context, logs the error, and includes it in the error response.
func SendError(w http.ResponseWriter, r *http.Request, err *serviceerror.ServiceError) {
	// Determine HTTP status code based on error type and code
	statusCode := mapErrorToStatusCode(err)

	// Extract trace ID from request context
	traceID := extractTraceID(r)

	// Log the error with context
	logger := log.GetLogger().WithContext(r.Context())
	if err.Type == serviceerror.ServerErrorType {
		logger.Error("Server error occurred",
			log.String("code", err.Code),
			log.String("message", err.Message),
			log.String("description", err.Description),
			log.Int("http_status", statusCode),
		)
	} else {
		logger.Warn("Client error occurred",
			log.String("code", err.Code),
			log.String("message", err.Message),
			log.String("description", err.Description),
			log.Int("http_status", statusCode),
		)
	}

	// Create error response with new format
	errorResponse := apierror.NewErrorResponse(
		err.Code,
		err.Message,
		err.Description,
		traceID,
	)

	w.Header().Set(constants.HeaderContentType, constants.ContentTypeJSON)
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(errorResponse)
}

// mapErrorToStatusCode maps service error codes to HTTP status codes
func mapErrorToStatusCode(err *serviceerror.ServiceError) int {
	if err.Type == serviceerror.ServerErrorType {
		return http.StatusInternalServerError
	}

	// Client error type - map specific codes
	switch err.Code {
	case codes.ResourceNotFound, codes.ConsentNotFound, codes.PurposeNotFound, codes.AuthResourceNotFound:
		return http.StatusNotFound
	case codes.ConflictError, codes.PurposeInUse:
		return http.StatusConflict
	case codes.ValidationError, codes.InvalidRequest:
		return http.StatusBadRequest
	default:
		return http.StatusBadRequest
	}
}

// extractTraceID extracts the trace ID (correlation ID) from the request context
func extractTraceID(r *http.Request) string {
	if r == nil || r.Context() == nil {
		return ""
	}

	traceID := r.Context().Value(log.ContextKeyTraceID)
	if traceID != nil {
		if tid, ok := traceID.(string); ok {
			return tid
		}
	}
	return ""
}
