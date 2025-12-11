package utils

import (
	"encoding/json"
	"net/http"

	"github.com/wso2/consent-management-api/internal/system/constants"
	"github.com/wso2/consent-management-api/internal/system/error/apierror"
	"github.com/wso2/consent-management-api/internal/system/error/serviceerror"
)

func DecodeJSONBody(r *http.Request, v interface{}) error {
	return json.NewDecoder(r.Body).Decode(v)
}

func JSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

func WriteJSONError(w http.ResponseWriter, code, description string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{
		"error":             code,
		"error_description": description,
	})
}

// SendError writes a ServiceError as an HTTP response with appropriate status code
func SendError(w http.ResponseWriter, err *serviceerror.ServiceError) {
	statusCode := http.StatusInternalServerError
	if err.Type == serviceerror.ClientErrorType {
		if err.Code == serviceerror.ResourceNotFoundError.Code {
			statusCode = http.StatusNotFound
		} else if err.Code == serviceerror.ConflictError.Code {
			statusCode = http.StatusConflict
		} else {
			statusCode = http.StatusBadRequest
		}
	}

	errorResponse := apierror.ErrorResponse{
		Code:        err.Error,
		Description: err.ErrorDescription,
	}

	w.Header().Set(constants.HeaderContentType, constants.ContentTypeJSON)
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(errorResponse)
}
