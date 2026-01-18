package consentpurpose

// PurposeCreateRequest represents the request payload for creating a purpose
type PurposeCreateRequest struct {
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	Elements    []PurposeElement `json:"elements"`
}

// PurposeUpdateRequest represents the request payload for updating a purpose
type PurposeUpdateRequest struct {
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	Elements    []PurposeElement `json:"elements"`
}

// PurposeElement represents an element within a purpose
type PurposeElement struct {
	Name        string `json:"name"`
	IsMandatory bool   `json:"isMandatory"`
}

// PurposeResponse represents the response for a purpose
type PurposeResponse struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Description *string          `json:"description,omitempty"`
	ClientID    string           `json:"clientId"`
	Elements    []PurposeElement `json:"elements"`
	CreatedTime int64            `json:"createdTime"`
	UpdatedTime int64            `json:"updatedTime"`
}

// PurposeListResponse represents the response for listing purposes
type PurposeListResponse struct {
	Data     []PurposeResponse   `json:"data"`
	Metadata PurposeListMetadata `json:"metadata"`
}

// PurposeListMetadata represents metadata for list operations
type PurposeListMetadata struct {
	Total  int `json:"total"`
	Offset int `json:"offset"`
	Limit  int `json:"limit"`
}

// ErrorResponse represents error response from the API
type ErrorResponse struct {
	Code        string `json:"code"`
	Message     string `json:"message"`
	Description string `json:"description"`
	TraceID     string `json:"traceId"`
}
