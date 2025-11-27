package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// Consent represents the CONSENT table
type Consent struct {
	ConsentID                  string `db:"CONSENT_ID" json:"consentId"`
	CreatedTime                int64  `db:"CREATED_TIME" json:"createdTime"`
	UpdatedTime                int64  `db:"UPDATED_TIME" json:"updatedTime"`
	ClientID                   string `db:"CLIENT_ID" json:"clientId"`
	ConsentType                string `db:"CONSENT_TYPE" json:"consentType"`
	CurrentStatus              string `db:"CURRENT_STATUS" json:"currentStatus"`
	ConsentFrequency           *int   `db:"CONSENT_FREQUENCY" json:"consentFrequency,omitempty"`
	ValidityTime               *int64 `db:"VALIDITY_TIME" json:"validityTime,omitempty"`
	RecurringIndicator         *bool  `db:"RECURRING_INDICATOR" json:"recurringIndicator,omitempty"`
	DataAccessValidityDuration *int64 `db:"DATA_ACCESS_VALIDITY_DURATION" json:"dataAccessValidityDuration,omitempty"`
	OrgID                      string `db:"ORG_ID" json:"orgId"`
}

// JSON type for handling JSON fields in MySQL
type JSON json.RawMessage

// Scan implements the sql.Scanner interface for JSON
func (j *JSON) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("unsupported type for JSON: %T", value)
	}

	// Validate that it's valid JSON by attempting to unmarshal and remarshal
	var temp interface{}
	if err := json.Unmarshal(bytes, &temp); err != nil {
		return fmt.Errorf("invalid JSON data: %w", err)
	}

	// Remarshal to ensure clean JSON
	cleanBytes, err := json.Marshal(temp)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	*j = JSON(cleanBytes)
	return nil
}

// Value implements the driver.Valuer interface for JSON
func (j JSON) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return []byte(j), nil
}

// MarshalJSON implements json.Marshaler
func (j JSON) MarshalJSON() ([]byte, error) {
	if j == nil {
		return []byte("null"), nil
	}
	return j, nil
}

// UnmarshalJSON implements json.Unmarshaler
func (j *JSON) UnmarshalJSON(data []byte) error {
	if j == nil {
		return nil
	}
	*j = JSON(data)
	return nil
}

// ConsentPurposeItem represents a single consent purpose with name, value, and selection status
type ConsentPurposeItem struct {
	Name           string                 `json:"name"`
	Value          interface{}            `json:"value"`                    // Can be string, object, or array
	IsUserApproved *bool                  `json:"isUserApproved,omitempty"` // Optional: defaults to false if not provided
	IsMandatory    *bool                  `json:"isMandatory,omitempty"`    // Optional: defaults to true if not provided
	Type           *string                `json:"type,omitempty"`           // Enriched from purpose definition (optional)
	Description    *string                `json:"description,omitempty"`    // Enriched from purpose definition (optional)
	Attributes     map[string]interface{} `json:"attributes,omitempty"`     // Enriched from purpose definition (optional)
}

// ConsentAPIRequest represents the API payload for creating a consent (external format)
// Note: Status is not included in the request - it will be derived from authorization states
type ConsentAPIRequest struct {
	Type                       string                    `json:"type" binding:"required"`
	ValidityTime               *int64                    `json:"validityTime,omitempty"`
	RecurringIndicator         *bool                     `json:"recurringIndicator,omitempty"`
	Frequency                  *int                      `json:"frequency,omitempty"`
	DataAccessValidityDuration *int64                    `json:"dataAccessValidityDuration,omitempty"`
	ConsentPurpose             []ConsentPurposeItem      `json:"consentPurpose,omitempty"`
	Attributes                 map[string]string         `json:"attributes,omitempty"`
	Authorizations             []AuthorizationAPIRequest `json:"authorizations"` // Remove omitempty to allow explicit empty array in updates
}

// AuthorizationAPIRequest represents the API payload for authorization resource (external format)
// Status field represents the authorization status/state (created, approved, rejected, or custom)
type AuthorizationAPIRequest struct {
	UserID    string      `json:"userId,omitempty"`
	Type      string      `json:"type" binding:"required"`
	Status    string      `json:"status,omitempty"` // Optional: defaults to "approved" if not provided
	Resources interface{} `json:"resources,omitempty"`
}

// ToAuthResourceCreateRequest converts API request format to internal format
func (req *AuthorizationAPIRequest) ToAuthResourceCreateRequest() *ConsentAuthResourceCreateRequest {
	var userID *string
	if req.UserID != "" {
		userID = &req.UserID
	}

	// Default status to "approved" if not provided
	status := req.Status
	if status == "" {
		status = string(AuthStateCreated)
	}

	return &ConsentAuthResourceCreateRequest{
		AuthType:   req.Type,
		UserID:     userID,
		AuthStatus: status, // Store the status value in AuthStatus field
		Resources:  req.Resources,
	}
}

// AuthorizationAPIUpdateRequest represents the API payload for updating authorization resource (external format)
type AuthorizationAPIUpdateRequest struct {
	UserID    string      `json:"userId,omitempty"`
	Type      string      `json:"type,omitempty"`
	Status    string      `json:"status,omitempty"`
	Resources interface{} `json:"resources,omitempty"`
}

// ToAuthResourceUpdateRequest converts API update request format to internal format
func (req *AuthorizationAPIUpdateRequest) ToAuthResourceUpdateRequest() *ConsentAuthResourceUpdateRequest {
	var userID *string
	if req.UserID != "" {
		userID = &req.UserID
	}

	return &ConsentAuthResourceUpdateRequest{
		AuthStatus: req.Status,
		UserID:     userID,
		Resources:  req.Resources,
	}
}

// ConsentAPIUpdateRequest represents the API payload for updating a consent (external format)
// Note: Status is not included in the request - it will be derived from authorization states
type ConsentAPIUpdateRequest struct {
	Type                       string                    `json:"type,omitempty"`
	ValidityTime               *int64                    `json:"validityTime,omitempty"`
	RecurringIndicator         *bool                     `json:"recurringIndicator,omitempty"`
	Frequency                  *int                      `json:"frequency,omitempty"`
	DataAccessValidityDuration *int64                    `json:"dataAccessValidityDuration,omitempty"`
	ConsentPurpose             []ConsentPurposeItem      `json:"consentPurpose,omitempty"`
	Attributes                 map[string]string         `json:"attributes,omitempty"`
	Authorizations             []AuthorizationAPIRequest `json:"authorizations"` // Remove omitempty to allow explicit empty array
}

// ConsentCreateRequest represents the internal request payload for creating a consent
type ConsentCreateRequest struct {
	ConsentPurpose             []ConsentPurposeItem               `json:"consentPurpose" binding:"required"`
	ConsentType                string                             `json:"consentType" binding:"required"`
	CurrentStatus              string                             `json:"currentStatus" binding:"required"`
	ConsentFrequency           *int                               `json:"consentFrequency,omitempty"`
	ValidityTime               *int64                             `json:"validityTime,omitempty"`
	RecurringIndicator         *bool                              `json:"recurringIndicator,omitempty"`
	DataAccessValidityDuration *int64                             `json:"dataAccessValidityDuration,omitempty"`
	Attributes                 map[string]string                  `json:"attributes,omitempty"`
	AuthResources              []ConsentAuthResourceCreateRequest `json:"authResources,omitempty"`
}

// ConsentUpdateRequest represents the request payload for updating a consent
type ConsentUpdateRequest struct {
	ConsentPurpose             []ConsentPurposeItem               `json:"consentPurpose,omitempty"`
	ConsentType                string                             `json:"consentType,omitempty"`
	CurrentStatus              string                             `json:"currentStatus,omitempty"`
	ConsentFrequency           *int                               `json:"consentFrequency,omitempty"`
	ValidityTime               *int64                             `json:"validityTime,omitempty"`
	RecurringIndicator         *bool                              `json:"recurringIndicator,omitempty"`
	DataAccessValidityDuration *int64                             `json:"dataAccessValidityDuration,omitempty"`
	Attributes                 map[string]string                  `json:"attributes,omitempty"`
	AuthResources              []ConsentAuthResourceCreateRequest `json:"authResources,omitempty"`
}

// ConsentResponse represents the response after consent creation/retrieval
type ConsentResponse struct {
	ConsentID                  string                `json:"consentId"`
	ConsentPurpose             []ConsentPurposeItem  `json:"consentPurpose,omitempty"`
	CreatedTime                int64                 `json:"createdTime"`
	UpdatedTime                int64                 `json:"updatedTime"`
	ClientID                   string                `json:"clientId"`
	ConsentType                string                `json:"consentType"`
	CurrentStatus              string                `json:"currentStatus"`
	ConsentFrequency           *int                  `json:"consentFrequency,omitempty"`
	ValidityTime               *int64                `json:"validityTime,omitempty"`
	RecurringIndicator         *bool                 `json:"recurringIndicator,omitempty"`
	DataAccessValidityDuration *int64                `json:"dataAccessValidityDuration,omitempty"`
	OrgID                      string                `json:"orgId"`
	Attributes                 map[string]string     `json:"attributes,omitempty"`
	AuthResources              []ConsentAuthResource `json:"authResources,omitempty"`
}

// ConsentSearchParams represents search parameters for consent queries
type ConsentSearchParams struct {
	ConsentIDs      []string `form:"consentIds"`
	ClientIDs       []string `form:"clientIds"`
	ConsentTypes    []string `form:"consentTypes"`
	ConsentStatuses []string `form:"consentStatuses"`
	UserIDs         []string `form:"userIds"`
	FromTime        *int64   `form:"fromTime"`
	ToTime          *int64   `form:"toTime"`
	Limit           int      `form:"limit"`
	Offset          int      `form:"offset"`
	OrgID           string   `form:"-"` // Extracted from header
}

// ConsentSearchResponse represents the response for consent search
type ConsentSearchResponse struct {
	Data     []ConsentResponse     `json:"data"`
	Metadata ConsentSearchMetadata `json:"metadata"`
}

// ConsentSearchMetadata represents pagination metadata
type ConsentSearchMetadata struct {
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

// ConsentRevokeRequest represents the request to revoke a consent
type ConsentRevokeRequest struct {
	ActionBy         string `json:"actionBy" binding:"required"`
	RevocationReason string `json:"revocationReason,omitempty"`
}

// GetCreatedTime returns the created time as a time.Time
func (c *Consent) GetCreatedTime() time.Time {
	return time.Unix(0, c.CreatedTime*int64(time.Millisecond))
}

// GetUpdatedTime returns the updated time as a time.Time
func (c *Consent) GetUpdatedTime() time.Time {
	return time.Unix(0, c.UpdatedTime*int64(time.Millisecond))
}

// ToConsentCreateRequest converts API request format to internal format
// Note: CurrentStatus will be set by the handler based on authorization states
func (req *ConsentAPIRequest) ToConsentCreateRequest() (*ConsentCreateRequest, error) {
	// Apply defaults and validate purposes
	consentPurposes := make([]ConsentPurposeItem, len(req.ConsentPurpose))
	for i, cp := range req.ConsentPurpose {
		consentPurposes[i] = cp

		// Apply default: isUserApproved = false if not provided
		if consentPurposes[i].IsUserApproved == nil {
			falseVal := false
			consentPurposes[i].IsUserApproved = &falseVal
		}

		// Apply default: isMandatory = true if not provided
		if consentPurposes[i].IsMandatory == nil {
			trueVal := true
			consentPurposes[i].IsMandatory = &trueVal
		}

		// Validation: if isMandatory is true, isUserApproved must be true
		if *consentPurposes[i].IsMandatory && !*consentPurposes[i].IsUserApproved {
			return nil, fmt.Errorf("purpose '%s': when isMandatory is true, isUserApproved must also be true", cp.Name)
		}
	}

	// Validate no duplicate purpose names
	purposeNames := make(map[string]bool)
	for _, cp := range consentPurposes {
		purposeName := cp.Name
		if purposeNames[purposeName] {
			return nil, fmt.Errorf("duplicate purpose name found: %s", purposeName)
		}
		purposeNames[purposeName] = true
	}

	createReq := &ConsentCreateRequest{
		ConsentPurpose:             consentPurposes,
		ConsentType:                req.Type,
		CurrentStatus:              "", // Will be set by handler based on auth states
		Attributes:                 req.Attributes,
		ValidityTime:               req.ValidityTime,
		ConsentFrequency:           req.Frequency,
		RecurringIndicator:         req.RecurringIndicator,
		DataAccessValidityDuration: req.DataAccessValidityDuration,
	}

	// Map authorizations to auth resources
	if len(req.Authorizations) > 0 {
		createReq.AuthResources = make([]ConsentAuthResourceCreateRequest, len(req.Authorizations))
		for i, auth := range req.Authorizations {
			var userID *string
			if auth.UserID != "" {
				userID = &auth.UserID
			}

			// Default status to "approved" if not provided
			status := auth.Status
			if status == "" {
				status = string(AuthStateApproved)
			}

			createReq.AuthResources[i] = ConsentAuthResourceCreateRequest{
				AuthType:   auth.Type,
				UserID:     userID,
				AuthStatus: status, // Store the status value
				Resources:  auth.Resources,
			}
		}
	}

	return createReq, nil
}

// ToConsentUpdateRequest converts API update request format to internal format
// Note: CurrentStatus will be set by the handler based on authorization states
func (req *ConsentAPIUpdateRequest) ToConsentUpdateRequest() (*ConsentUpdateRequest, error) {
	// Apply defaults and validate purposes
	var consentPurposes []ConsentPurposeItem
	if req.ConsentPurpose != nil {
		consentPurposes = make([]ConsentPurposeItem, len(req.ConsentPurpose))
		for i, cp := range req.ConsentPurpose {
			consentPurposes[i] = cp

			// Apply default: isUserApproved = false if not provided
			if consentPurposes[i].IsUserApproved == nil {
				falseVal := false
				consentPurposes[i].IsUserApproved = &falseVal
			}

			// Apply default: isMandatory = true if not provided
			if consentPurposes[i].IsMandatory == nil {
				trueVal := true
				consentPurposes[i].IsMandatory = &trueVal
			}

			// Validation: if isMandatory is true, isUserApproved must be true
			if *consentPurposes[i].IsMandatory && !*consentPurposes[i].IsUserApproved {
				return nil, fmt.Errorf("purpose '%s': when isMandatory is true, isUserApproved must also be true", cp.Name)
			}
		}

		// Validate no duplicate purpose names
		purposeNames := make(map[string]bool)
		for _, cp := range consentPurposes {
			purposeName := cp.Name
			if purposeNames[purposeName] {
				return nil, fmt.Errorf("duplicate purpose name found: %s", purposeName)
			}
			purposeNames[purposeName] = true
		}
	}

	updateReq := &ConsentUpdateRequest{
		ConsentPurpose:             consentPurposes,
		ConsentType:                req.Type,
		CurrentStatus:              "", // Will be set by handler based on auth states
		Attributes:                 req.Attributes,
		ValidityTime:               req.ValidityTime,
		ConsentFrequency:           req.Frequency,
		RecurringIndicator:         req.RecurringIndicator,
		DataAccessValidityDuration: req.DataAccessValidityDuration,
	}

	// Map authorizations to auth resources
	// If Authorizations is not nil (even if empty), set AuthResources to indicate intent to update
	if req.Authorizations != nil {
		updateReq.AuthResources = make([]ConsentAuthResourceCreateRequest, len(req.Authorizations))
		for i, auth := range req.Authorizations {
			var userID *string
			if auth.UserID != "" {
				userID = &auth.UserID
			}

			// Default status to "approved" if not provided
			status := auth.Status
			if status == "" {
				status = string(AuthStateApproved)
			}

			updateReq.AuthResources[i] = ConsentAuthResourceCreateRequest{
				AuthType:   auth.Type,
				UserID:     userID,
				AuthStatus: status, // Store the status value
				Resources:  auth.Resources,
			}
		}
	}

	return updateReq, nil
}

// ConsentAPIResponse represents the API response format for consent (external format)
type ConsentAPIResponse struct {
	ID                         string                     `json:"id"`
	ConsentPurpose             []ConsentPurposeItem       `json:"consentPurpose"`
	CreatedTime                int64                      `json:"createdTime"`
	UpdatedTime                int64                      `json:"updatedTime"`
	ClientID                   string                     `json:"clientId"`
	Type                       string                     `json:"type"`
	Status                     string                     `json:"status"`
	Frequency                  *int                       `json:"frequency"`
	ValidityTime               *int64                     `json:"validityTime"`
	RecurringIndicator         *bool                      `json:"recurringIndicator"`
	DataAccessValidityDuration *int64                     `json:"dataAccessValidityDuration"`
	Attributes                 map[string]string          `json:"attributes"`
	Authorizations             []AuthorizationAPIResponse `json:"authorizations"`
	ModifiedResponse           interface{}                `json:"modifiedResponse"` // Present in GET/POST/PUT, excluded in validate
}

// AuthorizationAPIResponse represents the API response format for authorization resource (external format)
type AuthorizationAPIResponse struct {
	ID          string      `json:"id"`
	UserID      *string     `json:"userId"`
	Type        string      `json:"type"`
	Status      string      `json:"status"`
	UpdatedTime int64       `json:"updatedTime"`
	Resources   interface{} `json:"resources"`
}

// ToAPIResponse converts internal response format to API response format
func (resp *ConsentResponse) ToAPIResponse() *ConsentAPIResponse {
	// Initialize Attributes with empty object if nil
	attributes := resp.Attributes
	if attributes == nil {
		attributes = make(map[string]string)
	}

	// Initialize ConsentPurpose with empty array if nil
	consentPurpose := resp.ConsentPurpose
	if consentPurpose == nil {
		consentPurpose = make([]ConsentPurposeItem, 0)
	}

	apiResp := &ConsentAPIResponse{
		ID:                         resp.ConsentID,
		ConsentPurpose:             consentPurpose,
		CreatedTime:                resp.CreatedTime,
		UpdatedTime:                resp.UpdatedTime,
		ClientID:                   resp.ClientID,
		Type:                       resp.ConsentType,
		Status:                     resp.CurrentStatus,
		Frequency:                  resp.ConsentFrequency,
		ValidityTime:               resp.ValidityTime,
		RecurringIndicator:         resp.RecurringIndicator,
		DataAccessValidityDuration: resp.DataAccessValidityDuration,
		Attributes:                 attributes,
		ModifiedResponse:           make(map[string]interface{}),
		Authorizations:             make([]AuthorizationAPIResponse, 0),
	}

	// Map auth resources to authorizations
	if len(resp.AuthResources) > 0 {
		apiResp.Authorizations = make([]AuthorizationAPIResponse, len(resp.AuthResources))
		for i, auth := range resp.AuthResources {
			// Parse resources JSON string to interface
			var resources interface{}
			if auth.Resources != nil && *auth.Resources != "" {
				if err := json.Unmarshal([]byte(*auth.Resources), &resources); err != nil {
					// If parsing fails, set to empty object
					resources = make(map[string]interface{})
				}
			} else {
				// If resources is nil or empty, set to empty object
				resources = make(map[string]interface{})
			}

			apiResp.Authorizations[i] = AuthorizationAPIResponse{
				ID:          auth.AuthID,
				UserID:      auth.UserID,
				Type:        auth.AuthType,
				Status:      auth.AuthStatus,
				UpdatedTime: auth.UpdatedTime,
				Resources:   resources,
			}
		}
	}

	return apiResp
}

// ValidateRequest represents the payload for validation API
type ValidateRequest struct {
	Headers         map[string]interface{} `json:"headers"`
	Payload         map[string]interface{} `json:"payload"`
	ElectedResource string                 `json:"electedResource"`
	ConsentID       string                 `json:"consentId"`
	UserID          string                 `json:"userId"`
	ClientID        string                 `json:"clientId"`
	ResourceParams  struct {
		Resource   string `json:"resource"`
		HTTPMethod string `json:"httpMethod"`
		Context    string `json:"context"`
	} `json:"resourceParams"`
}

// ValidateResponse represents the response for validation API
type ValidateResponse struct {
	IsValid            bool                        `json:"isValid"`
	ModifiedPayload    interface{}                 `json:"modifiedPayload,omitempty"`
	ErrorCode          int                         `json:"errorCode,omitempty"`
	ErrorMessage       string                      `json:"errorMessage,omitempty"`
	ErrorDescription   string                      `json:"errorDescription,omitempty"`
	ConsentInformation *ValidateConsentAPIResponse `json:"consentInformation,omitempty"`
}

// ValidateConsentAPIResponse represents consent information in validate response (excludes modifiedResponse)
type ValidateConsentAPIResponse struct {
	ID                         string                     `json:"id"`
	Type                       string                     `json:"type"`
	ClientID                   string                     `json:"clientId"`
	Status                     string                     `json:"status"`
	CreatedTime                int64                      `json:"createdTime"`
	UpdatedTime                int64                      `json:"updatedTime"`
	ValidityTime               *int64                     `json:"validityTime"`
	RecurringIndicator         *bool                      `json:"recurringIndicator"`
	Frequency                  *int                       `json:"frequency"`
	DataAccessValidityDuration *int64                     `json:"dataAccessValidityDuration"`
	ConsentPurpose             []ConsentPurposeItem       `json:"consentPurpose"`
	Attributes                 map[string]string          `json:"attributes,omitempty"`
	Authorizations             []AuthorizationAPIResponse `json:"authorizations,omitempty"`
}

// ToValidateConsentAPIResponse converts ConsentAPIResponse to ValidateConsentAPIResponse (excludes modifiedResponse)
func (c *ConsentAPIResponse) ToValidateConsentAPIResponse() *ValidateConsentAPIResponse {
	if c == nil {
		return nil
	}
	return &ValidateConsentAPIResponse{
		ID:                         c.ID,
		Type:                       c.Type,
		ClientID:                   c.ClientID,
		Status:                     c.Status,
		CreatedTime:                c.CreatedTime,
		UpdatedTime:                c.UpdatedTime,
		ValidityTime:               c.ValidityTime,
		RecurringIndicator:         c.RecurringIndicator,
		Frequency:                  c.Frequency,
		DataAccessValidityDuration: c.DataAccessValidityDuration,
		ConsentPurpose:             c.ConsentPurpose,
		Attributes:                 c.Attributes,
		Authorizations:             c.Authorizations,
	}
}

// ConsentRevokeResponse represents the response after revoking a consent
type ConsentRevokeResponse struct {
	ActionTime       int64  `json:"actionTime"`
	ActionBy         string `json:"actionBy"`
	RevocationReason string `json:"revocationReason,omitempty"`
}
