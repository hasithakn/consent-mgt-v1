package serviceerror

type ServiceErrorType string

const (
	ClientErrorType ServiceErrorType = "client_error"
	ServerErrorType ServiceErrorType = "server_error"
)

type ServiceError struct {
	Code             string           `json:"code"`
	Type             ServiceErrorType `json:"type"`
	Error            string           `json:"error"`
	ErrorDescription string           `json:"error_description,omitempty"`
}

var (
	InternalServerError = ServiceError{
		Type:             ServerErrorType,
		Code:             "SSE-5000",
		Error:            "internal_server_error",
		ErrorDescription: "An unexpected error occurred",
	}

	DatabaseError = ServiceError{
		Type:             ServerErrorType,
		Code:             "SSE-5001",
		Error:            "database_error",
		ErrorDescription: "A database error occurred",
	}

	InvalidRequestError = ServiceError{
		Type:             ClientErrorType,
		Code:             "CSE-4000",
		Error:            "invalid_request",
		ErrorDescription: "The request is invalid",
	}

	ResourceNotFoundError = ServiceError{
		Type:             ClientErrorType,
		Code:             "CSE-4004",
		Error:            "resource_not_found",
		ErrorDescription: "Resource not found",
	}

	ConflictError = ServiceError{
		Type:             ClientErrorType,
		Code:             "CSE-4009",
		Error:            "conflict",
		ErrorDescription: "Request conflicts with current state",
	}

	ValidationError = ServiceError{
		Type:             ClientErrorType,
		Code:             "CSE-4001",
		Error:            "validation_error",
		ErrorDescription: "Validation failed",
	}
)

func CustomServiceError(baseError ServiceError, description string) *ServiceError {
	return &ServiceError{
		Type:             baseError.Type,
		Code:             baseError.Code,
		Error:            baseError.Error,
		ErrorDescription: description,
	}
}
