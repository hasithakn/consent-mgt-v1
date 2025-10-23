package models

// Extension API Request/Response Models
// These models represent the payloads for extension service communication

// PreProcessConsentCreationRequest represents the request payload for pre-process-consent-creation endpoint
type PreProcessConsentCreationRequest struct {
	RequestID string                               `json:"requestId"`
	Data      PreProcessConsentCreationRequestData `json:"data"`
}

// PreProcessConsentCreationRequestData contains the consent initiation data and headers
type PreProcessConsentCreationRequestData struct {
	ConsentInitiationData ConsentInitiationData `json:"consentInitiationData"`
	RequestHeaders        map[string]string     `json:"requestHeaders"`
}

// ConsentInitiationData contains the detailed consent information
type ConsentInitiationData struct {
	Type                       string                 `json:"type"`
	Status                     string                 `json:"status"`
	ValidityTime               *int64                 `json:"validityTime,omitempty"`
	RecurringIndicator         *bool                  `json:"recurringIndicator,omitempty"`
	Frequency                  *int                   `json:"frequency,omitempty"`
	DataAccessValidityDuration *int64                 `json:"dataAccessValidityDuration,omitempty"`
	RequestPayload             map[string]interface{} `json:"requestPayload,omitempty"`
	Attributes                 map[string]interface{} `json:"attributes,omitempty"`
	Authorizations             []AuthorizationPayload `json:"authorizations,omitempty"`
}

// AuthorizationPayload represents authorization data in extension requests
type AuthorizationPayload struct {
	UserID   string                 `json:"userId"`
	Type     string                 `json:"type"`
	Status   string                 `json:"status"`
	Resource map[string]interface{} `json:"resource,omitempty"`
}

// PreProcessConsentCreationResponse represents the response from pre-process-consent-creation endpoint
type PreProcessConsentCreationResponse struct {
	ResponseID string                                 `json:"responseId"`
	Status     string                                 `json:"status"` // SUCCESS or ERROR
	Data       *PreProcessConsentCreationResponseData `json:"data,omitempty"`
	ErrorCode  *int                                   `json:"errorCode,omitempty"`
	ErrorData  map[string]interface{}                 `json:"errorData,omitempty"`
}

// PreProcessConsentCreationResponseData contains the consent resource and resolved purposes
type PreProcessConsentCreationResponseData struct {
	ConsentResource         DetailedConsentResourceData `json:"consentResource"`
	ResolvedConsentPurposes []string                    `json:"resolvedConsentPurposes,omitempty"`
}

// DetailedConsentResourceData represents detailed consent data from extension
type DetailedConsentResourceData struct {
	ID                         string                 `json:"id,omitempty"`
	RequestPayload             map[string]interface{} `json:"requestPayload,omitempty"`
	CreatedTime                *int64                 `json:"createdTime,omitempty"`
	UpdatedTime                *int64                 `json:"updatedTime,omitempty"`
	ClientID                   string                 `json:"clientId,omitempty"`
	Type                       string                 `json:"type"`
	Status                     string                 `json:"status"`
	Frequency                  *int                   `json:"frequency,omitempty"`
	ValidityTime               *int64                 `json:"validityTime,omitempty"`
	RecurringIndicator         *bool                  `json:"recurringIndicator,omitempty"`
	DataAccessValidityDuration *int64                 `json:"dataAccessValidityDuration,omitempty"`
	Attributes                 map[string]interface{} `json:"attributes,omitempty"`
	Authorizations             []AuthorizationPayload `json:"authorizations,omitempty"`
}

// ToConsentInitiationData converts ConsentCreateRequest to ConsentInitiationData
func (c *ConsentCreateRequest) ToConsentInitiationData() ConsentInitiationData {
	// Convert attributes map[string]string to map[string]interface{}
	var attrs map[string]interface{}
	if c.Attributes != nil {
		attrs = make(map[string]interface{}, len(c.Attributes))
		for k, v := range c.Attributes {
			attrs[k] = v
		}
	}

	initData := ConsentInitiationData{
		Type:                       c.ConsentType,
		Status:                     c.CurrentStatus,
		ValidityTime:               c.ValidityTime,
		RecurringIndicator:         c.RecurringIndicator,
		Frequency:                  c.ConsentFrequency,
		DataAccessValidityDuration: c.DataAccessValidityDuration,
		Attributes:                 attrs,
	}

	// Convert receipt to request payload
	if c.Receipt != nil {
		initData.RequestPayload = c.Receipt
	}

	// Convert authorizations
	if len(c.AuthResources) > 0 {
		initData.Authorizations = make([]AuthorizationPayload, len(c.AuthResources))
		for i, auth := range c.AuthResources {
			userID := ""
			if auth.UserID != nil {
				userID = *auth.UserID
			}
			initData.Authorizations[i] = AuthorizationPayload{
				UserID:   userID,
				Type:     auth.AuthType,
				Status:   auth.AuthStatus,
				Resource: auth.Resource,
			}
		}
	}

	return initData
}

// ToConsentCreateRequest converts extension response data to ConsentCreateRequest
func (d *DetailedConsentResourceData) ToConsentCreateRequest() *ConsentCreateRequest {
	// Convert attributes map[string]interface{} to map[string]string
	var attrs map[string]string
	if d.Attributes != nil {
		attrs = make(map[string]string, len(d.Attributes))
		for k, v := range d.Attributes {
			if strVal, ok := v.(string); ok {
				attrs[k] = strVal
			}
		}
	}

	req := &ConsentCreateRequest{
		ConsentType:                d.Type,
		CurrentStatus:              d.Status,
		ValidityTime:               d.ValidityTime,
		RecurringIndicator:         d.RecurringIndicator,
		ConsentFrequency:           d.Frequency,
		DataAccessValidityDuration: d.DataAccessValidityDuration,
		Attributes:                 attrs,
		Receipt:                    d.RequestPayload,
	}

	// Convert authorizations
	if len(d.Authorizations) > 0 {
		req.AuthResources = make([]ConsentAuthResourceCreateRequest, len(d.Authorizations))
		for i, auth := range d.Authorizations {
			userID := auth.UserID
			req.AuthResources[i] = ConsentAuthResourceCreateRequest{
				UserID:     &userID,
				AuthType:   auth.Type,
				AuthStatus: auth.Status,
				Resource:   auth.Resource,
			}
		}
	}

	return req
}

// PreProcessConsentUpdateRequest represents the request payload for pre-process-consent-update endpoint
type PreProcessConsentUpdateRequest struct {
	RequestID string                             `json:"requestId"`
	Data      PreProcessConsentUpdateRequestData `json:"data"`
}

// PreProcessConsentUpdateRequestData contains the consent update data and headers
type PreProcessConsentUpdateRequestData struct {
	ConsentID             string                `json:"consentId"`
	ConsentInitiationData ConsentInitiationData `json:"consentInitiationData"`
	RequestHeaders        map[string]string     `json:"requestHeaders"`
}

// PreProcessConsentUpdateResponse represents the response from pre-process-consent-update endpoint
type PreProcessConsentUpdateResponse struct {
	ResponseID string                               `json:"responseId"`
	Status     string                               `json:"status"` // SUCCESS or ERROR
	Data       *PreProcessConsentUpdateResponseData `json:"data,omitempty"`
	ErrorCode  *int                                 `json:"errorCode,omitempty"`
	ErrorData  map[string]interface{}               `json:"errorData,omitempty"`
}

// PreProcessConsentUpdateResponseData contains the modified consent resource and resolved purposes
type PreProcessConsentUpdateResponseData struct {
	ConsentResource         DetailedConsentResourceData `json:"consentResource"`
	ResolvedConsentPurposes []string                    `json:"resolvedConsentPurposes,omitempty"`
}

// ToConsentInitiationDataFromUpdate converts ConsentUpdateRequest to ConsentInitiationData
func (c *ConsentUpdateRequest) ToConsentInitiationData() ConsentInitiationData {
	// Convert attributes map[string]string to map[string]interface{}
	var attrs map[string]interface{}
	if c.Attributes != nil {
		attrs = make(map[string]interface{}, len(c.Attributes))
		for k, v := range c.Attributes {
			attrs[k] = v
		}
	}

	initData := ConsentInitiationData{
		Type:                       c.ConsentType,
		Status:                     c.CurrentStatus,
		ValidityTime:               c.ValidityTime,
		RecurringIndicator:         c.RecurringIndicator,
		Frequency:                  c.ConsentFrequency,
		DataAccessValidityDuration: c.DataAccessValidityDuration,
		Attributes:                 attrs,
	}

	// Convert receipt to request payload
	if c.Receipt != nil {
		initData.RequestPayload = c.Receipt
	}

	// Convert authorizations
	if len(c.AuthResources) > 0 {
		initData.Authorizations = make([]AuthorizationPayload, len(c.AuthResources))
		for i, auth := range c.AuthResources {
			userID := ""
			if auth.UserID != nil {
				userID = *auth.UserID
			}
			initData.Authorizations[i] = AuthorizationPayload{
				UserID:   userID,
				Type:     auth.AuthType,
				Status:   auth.AuthStatus,
				Resource: auth.Resource,
			}
		}
	}

	return initData
}

// ToConsentUpdateRequest converts extension response data to ConsentUpdateRequest
func (d *DetailedConsentResourceData) ToConsentUpdateRequest() *ConsentUpdateRequest {
	// Convert attributes map[string]interface{} to map[string]string
	var attrs map[string]string
	if d.Attributes != nil {
		attrs = make(map[string]string, len(d.Attributes))
		for k, v := range d.Attributes {
			if strVal, ok := v.(string); ok {
				attrs[k] = strVal
			}
		}
	}

	req := &ConsentUpdateRequest{
		ConsentType:                d.Type,
		CurrentStatus:              d.Status,
		ValidityTime:               d.ValidityTime,
		RecurringIndicator:         d.RecurringIndicator,
		ConsentFrequency:           d.Frequency,
		DataAccessValidityDuration: d.DataAccessValidityDuration,
		Attributes:                 attrs,
		Receipt:                    d.RequestPayload,
	}

	// Convert authorizations
	if len(d.Authorizations) > 0 {
		req.AuthResources = make([]ConsentAuthResourceCreateRequest, len(d.Authorizations))
		for i, auth := range d.Authorizations {
			userID := auth.UserID
			req.AuthResources[i] = ConsentAuthResourceCreateRequest{
				UserID:     &userID,
				AuthType:   auth.Type,
				AuthStatus: auth.Status,
				Resource:   auth.Resource,
			}
		}
	}

	return req
}
