package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// Consent represents the FS_CONSENT table
type Consent struct {
	ConsentID                  string `db:"CONSENT_ID" json:"consentId"`
	Receipt                    JSON   `db:"RECEIPT" json:"receipt"`
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

// ConsentAPIRequest represents the API payload for creating a consent (external format)
type ConsentAPIRequest struct {
	Type                       string                    `json:"type" binding:"required"`
	Status                     string                    `json:"status" binding:"required"`
	ValidityTime               *int64                    `json:"validityTime,omitempty"`
	RecurringIndicator         *bool                     `json:"recurringIndicator,omitempty"`
	Frequency                  *int                      `json:"frequency,omitempty"`
	DataAccessValidityDuration *int64                    `json:"dataAccessValidityDuration,omitempty"`
	RequestPayload             map[string]interface{}    `json:"requestPayload" binding:"required"`
	Attributes                 map[string]string         `json:"attributes,omitempty"`
	Authorizations             []AuthorizationAPIRequest `json:"authorizations,omitempty"`
}

// AuthorizationAPIRequest represents the API payload for authorization resource (external format)
type AuthorizationAPIRequest struct {
	UserID   string                 `json:"userId,omitempty"`
	Type     string                 `json:"type" binding:"required"`
	Status   string                 `json:"status" binding:"required"`
	Resource map[string]interface{} `json:"resource,omitempty"`
}

// ToAuthResourceCreateRequest converts API request format to internal format
func (req *AuthorizationAPIRequest) ToAuthResourceCreateRequest() *ConsentAuthResourceCreateRequest {
	var userID *string
	if req.UserID != "" {
		userID = &req.UserID
	}

	return &ConsentAuthResourceCreateRequest{
		AuthType:   req.Type,
		UserID:     userID,
		AuthStatus: req.Status,
		Resource:   req.Resource,
	}
}

// AuthorizationAPIUpdateRequest represents the API payload for updating authorization resource (external format)
type AuthorizationAPIUpdateRequest struct {
	UserID   string                 `json:"userId,omitempty"`
	Type     string                 `json:"type,omitempty"`
	Status   string                 `json:"status,omitempty"`
	Resource map[string]interface{} `json:"resource,omitempty"`
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
		Resource:   req.Resource,
	}
}

// ConsentAPIUpdateRequest represents the API payload for updating a consent (external format)
type ConsentAPIUpdateRequest struct {
	Type                       string                    `json:"type,omitempty"`
	Status                     string                    `json:"status,omitempty"`
	ValidityTime               *int64                    `json:"validityTime,omitempty"`
	RecurringIndicator         *bool                     `json:"recurringIndicator,omitempty"`
	Frequency                  *int                      `json:"frequency,omitempty"`
	DataAccessValidityDuration *int64                    `json:"dataAccessValidityDuration,omitempty"`
	RequestPayload             map[string]interface{}    `json:"requestPayload,omitempty"`
	Attributes                 map[string]string         `json:"attributes,omitempty"`
	Authorizations             []AuthorizationAPIRequest `json:"authorizations,omitempty"`
}

// ConsentCreateRequest represents the internal request payload for creating a consent
type ConsentCreateRequest struct {
	Receipt                    map[string]interface{}             `json:"receipt" binding:"required"`
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
	Receipt                    map[string]interface{}             `json:"receipt,omitempty"`
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
	ConsentID                  string                 `json:"consentId"`
	Receipt                    map[string]interface{} `json:"receipt"`
	CreatedTime                int64                  `json:"createdTime"`
	UpdatedTime                int64                  `json:"updatedTime"`
	ClientID                   string                 `json:"clientId"`
	ConsentType                string                 `json:"consentType"`
	CurrentStatus              string                 `json:"currentStatus"`
	ConsentFrequency           *int                   `json:"consentFrequency,omitempty"`
	ValidityTime               *int64                 `json:"validityTime,omitempty"`
	RecurringIndicator         *bool                  `json:"recurringIndicator,omitempty"`
	DataAccessValidityDuration *int64                 `json:"dataAccessValidityDuration,omitempty"`
	OrgID                      string                 `json:"orgId"`
	Attributes                 map[string]string      `json:"attributes,omitempty"`
	AuthResources              []ConsentAuthResource  `json:"authResources,omitempty"`
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
	Reason   string `json:"reason,omitempty"`
	ActionBy string `json:"actionBy,omitempty"`
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
func (req *ConsentAPIRequest) ToConsentCreateRequest() (*ConsentCreateRequest, error) {
	createReq := &ConsentCreateRequest{
		Receipt:                    req.RequestPayload,
		ConsentType:                req.Type,
		CurrentStatus:              req.Status,
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

			createReq.AuthResources[i] = ConsentAuthResourceCreateRequest{
				AuthType:   auth.Type,
				UserID:     userID,
				AuthStatus: auth.Status,
				Resource:   auth.Resource,
			}
		}
	}

	return createReq, nil
}

// ToConsentUpdateRequest converts API update request format to internal format
func (req *ConsentAPIUpdateRequest) ToConsentUpdateRequest() (*ConsentUpdateRequest, error) {
	updateReq := &ConsentUpdateRequest{
		Receipt:                    req.RequestPayload,
		ConsentType:                req.Type,
		CurrentStatus:              req.Status,
		Attributes:                 req.Attributes,
		ValidityTime:               req.ValidityTime,
		ConsentFrequency:           req.Frequency,
		RecurringIndicator:         req.RecurringIndicator,
		DataAccessValidityDuration: req.DataAccessValidityDuration,
	}

	// Map authorizations to auth resources
	if len(req.Authorizations) > 0 {
		updateReq.AuthResources = make([]ConsentAuthResourceCreateRequest, len(req.Authorizations))
		for i, auth := range req.Authorizations {
			var userID *string
			if auth.UserID != "" {
				userID = &auth.UserID
			}

			updateReq.AuthResources[i] = ConsentAuthResourceCreateRequest{
				AuthType:   auth.Type,
				UserID:     userID,
				AuthStatus: auth.Status,
				Resource:   auth.Resource,
			}
		}
	}

	return updateReq, nil
}

// ConsentAPIResponse represents the API response format for consent (external format)
type ConsentAPIResponse struct {
	ID                         string                     `json:"id"`
	RequestPayload             map[string]interface{}     `json:"requestPayload"`
	CreatedTime                int64                      `json:"createdTime"`
	UpdatedTime                int64                      `json:"updatedTime"`
	ClientID                   string                     `json:"clientId"`
	Type                       string                     `json:"type"`
	Status                     string                     `json:"status"`
	Frequency                  *int                       `json:"frequency"`
	ValidityTime               *int64                     `json:"validityTime"`
	RecurringIndicator         *bool                      `json:"recurringIndicator"`
	DataAccessValidityDuration *int64                     `json:"dataAccessValidityDuration,omitempty"`
	Attributes                 map[string]string          `json:"attributes"`
	Authorizations             []AuthorizationAPIResponse `json:"authorizations"`
	ModifiedResponse           map[string]interface{}     `json:"modifiedResponse"`
}

// AuthorizationAPIResponse represents the API response format for authorization resource (external format)
type AuthorizationAPIResponse struct {
	ID          string                 `json:"id"`
	UserID      *string                `json:"userId"`
	Type        string                 `json:"type"`
	Status      string                 `json:"status"`
	UpdatedTime int64                  `json:"updatedTime"`
	Resource    map[string]interface{} `json:"resource"`
}

// ToAPIResponse converts internal response format to API response format
func (resp *ConsentResponse) ToAPIResponse() *ConsentAPIResponse {
	// Initialize RequestPayload with empty object if nil
	requestPayload := resp.Receipt
	if requestPayload == nil {
		requestPayload = make(map[string]interface{})
	}

	// Initialize Attributes with empty object if nil
	attributes := resp.Attributes
	if attributes == nil {
		attributes = make(map[string]string)
	}

	apiResp := &ConsentAPIResponse{
		ID:                         resp.ConsentID,
		RequestPayload:             requestPayload,
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
			// Parse resource JSON string to map
			var resourceMap map[string]interface{}
			if auth.Resource != nil && *auth.Resource != "" {
				if err := json.Unmarshal([]byte(*auth.Resource), &resourceMap); err != nil {
					// If unmarshal fails, use empty map
					resourceMap = make(map[string]interface{})
				}
			} else {
				resourceMap = make(map[string]interface{})
			}

			apiResp.Authorizations[i] = AuthorizationAPIResponse{
				ID:          auth.AuthID,
				UserID:      auth.UserID,
				Type:        auth.AuthType,
				Status:      auth.AuthStatus,
				UpdatedTime: auth.UpdatedTime,
				Resource:    resourceMap,
			}
		}
	}

	return apiResp
}
