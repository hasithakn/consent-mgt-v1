package consentpurposegroup

// PurposeGroupCreateRequest represents the request payload for creating a purpose group
type PurposeGroupCreateRequest struct {
	Name        string                `json:"name"`
	Description string                `json:"description,omitempty"`
	Purposes    []PurposeGroupPurpose `json:"purposes"`
}

// PurposeGroupUpdateRequest represents the request payload for updating a purpose group
type PurposeGroupUpdateRequest struct {
	Name        string                `json:"name"`
	Description string                `json:"description,omitempty"`
	Purposes    []PurposeGroupPurpose `json:"purposes"`
}

// PurposeGroupPurpose represents a purpose within a purpose group
type PurposeGroupPurpose struct {
	PurposeName string `json:"purposeName"`
	IsMandatory bool   `json:"isMandatory"`
}

// PurposeGroupResponse represents the response for a purpose group
type PurposeGroupResponse struct {
	ID          string                `json:"id"`
	Name        string                `json:"name"`
	Description *string               `json:"description,omitempty"`
	ClientID    string                `json:"clientId"`
	Purposes    []PurposeGroupPurpose `json:"purposes"`
	CreatedTime int64                 `json:"createdTime"`
	UpdatedTime int64                 `json:"updatedTime"`
}

// PurposeGroupListResponse represents the response for listing purpose groups
type PurposeGroupListResponse struct {
	Data     []PurposeGroupResponse   `json:"data"`
	Metadata PurposeGroupListMetadata `json:"metadata"`
}

// PurposeGroupListMetadata represents metadata for list operations
type PurposeGroupListMetadata struct {
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
