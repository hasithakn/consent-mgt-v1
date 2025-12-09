package apierror

type ErrorResponse struct {
	Code        string `json:"error"`
	Description string `json:"error_description"`
}

func NewErrorResponse(code, description string) *ErrorResponse {
	return &ErrorResponse{
		Code:        code,
		Description: description,
	}
}
