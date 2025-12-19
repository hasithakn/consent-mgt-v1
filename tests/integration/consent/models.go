package consent

// ConsentPurposeItem represents a consent purpose in the request/response
type ConsentPurposeItem struct {
	Name           string      `json:"name"`
	Value          interface{} `json:"value,omitempty"`
	IsUserApproved bool        `json:"isUserApproved"`
	IsMandatory    bool        `json:"isMandatory"`
}

// AuthorizationRequest represents authorization data in consent creation/update
type AuthorizationRequest struct {
	UserID         string   `json:"userId"`
	Type           string   `json:"type"`
	Status         string   `json:"status"`
	Resources      []string `json:"resources,omitempty"`
	Permissions    []string `json:"permissions,omitempty"`
	ExpirationDate string   `json:"expirationDate,omitempty"`
}

// ConsentCreateRequest represents the payload for creating a consent
type ConsentCreateRequest struct {
	Type               string                 `json:"type"`
	ConsentPurpose     []ConsentPurposeItem   `json:"consentPurpose,omitempty"`
	Authorizations     []AuthorizationRequest `json:"authorizations"`
	Attributes         map[string]string      `json:"attributes,omitempty"`
	ValidityTime       int64                  `json:"validityTime,omitempty"`
	RecurringIndicator bool                   `json:"recurringIndicator,omitempty"`
	Frequency          int                    `json:"frequency,omitempty"`
}

// ConsentUpdateRequest represents the payload for updating a consent
type ConsentUpdateRequest struct {
	Type               string                 `json:"type,omitempty"`
	ConsentPurpose     []ConsentPurposeItem   `json:"consentPurpose,omitempty"`
	Authorizations     []AuthorizationRequest `json:"authorizations,omitempty"`
	Attributes         map[string]string      `json:"attributes,omitempty"`
	ValidityTime       int64                  `json:"validityTime,omitempty"`
	RecurringIndicator *bool                  `json:"recurringIndicator,omitempty"`
	Frequency          *int                   `json:"frequency,omitempty"`
}

// ConsentRevokeRequest represents the payload for revoking a consent
type ConsentRevokeRequest struct {
	Reason string `json:"reason,omitempty"`
}

// AuthorizationResponse represents authorization data in consent response
type AuthorizationResponse struct {
	ID          string      `json:"id"`
	UserID      *string     `json:"userId,omitempty"`
	Type        string      `json:"type"`
	Status      string      `json:"status"`
	UpdatedTime int64       `json:"updatedTime"`
	Resources   interface{} `json:"resources,omitempty"`
}

// ConsentResponse represents the API response for a consent
type ConsentResponse struct {
	ID                         string                  `json:"id"`
	ClientID                   string                  `json:"clientId"`
	Type                       string                  `json:"type"`
	Status                     string                  `json:"status"`
	ConsentPurpose             []ConsentPurposeItem    `json:"consentPurpose"`
	Authorizations             []AuthorizationResponse `json:"authorizations"`
	Attributes                 map[string]string       `json:"attributes"`
	ValidityTime               *int64                  `json:"validityTime,omitempty"`
	RecurringIndicator         *bool                   `json:"recurringIndicator,omitempty"`
	Frequency                  *int                    `json:"frequency,omitempty"`
	DataAccessValidityDuration *int64                  `json:"dataAccessValidityDuration,omitempty"`
	CreatedTime                int64                   `json:"createdTime"`
	UpdatedTime                int64                   `json:"updatedTime"`
}

// ConsentListResponse represents the API response for listing consents
type ConsentListResponse struct {
	Data []ConsentResponse `json:"data"`
	Meta struct {
		Total  int `json:"total"`
		Offset int `json:"offset"`
		Limit  int `json:"limit"`
	} `json:"meta"`
}

// ErrorResponse represents error responses from the API
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}
