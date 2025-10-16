package models

// ConsentAuthResource represents the FS_CONSENT_AUTH_RESOURCE table
type ConsentAuthResource struct {
	AuthID      string  `db:"AUTH_ID" json:"authId"`
	ConsentID   string  `db:"CONSENT_ID" json:"consentId"`
	AuthType    string  `db:"AUTH_TYPE" json:"authType"`
	UserID      *string `db:"USER_ID" json:"userId,omitempty"`
	AuthStatus  string  `db:"AUTH_STATUS" json:"authStatus"`
	UpdatedTime int64   `db:"UPDATED_TIME" json:"updatedTime"`
	Resource    *string `db:"RESOURCE" json:"resource,omitempty"`
	OrgID       string  `db:"ORG_ID" json:"orgId"`
}

// ConsentAuthResourceCreateRequest represents the request payload for creating an authorization resource
type ConsentAuthResourceCreateRequest struct {
	AuthType   string                 `json:"authType" binding:"required"`
	UserID     *string                `json:"userId,omitempty"`
	AuthStatus string                 `json:"authStatus" binding:"required"`
	Resource   map[string]interface{} `json:"resource,omitempty"`
}

// ConsentAuthResourceUpdateRequest represents the request payload for updating an authorization resource
type ConsentAuthResourceUpdateRequest struct {
	AuthStatus string                 `json:"authStatus,omitempty"`
	UserID     *string                `json:"userId,omitempty"`
	Resource   map[string]interface{} `json:"resource,omitempty"`
}

// ConsentAuthResourceResponse represents the response for authorization resource operations
type ConsentAuthResourceResponse struct {
	AuthID      string                 `json:"authId"`
	ConsentID   string                 `json:"consentId"`
	AuthType    string                 `json:"authType"`
	UserID      *string                `json:"userId,omitempty"`
	AuthStatus  string                 `json:"authStatus"`
	UpdatedTime int64                  `json:"updatedTime"`
	Resource    map[string]interface{} `json:"resource,omitempty"`
	OrgID       string                 `json:"orgId"`
}

// ConsentAuthResourceListResponse represents the response for listing authorization resources
type ConsentAuthResourceListResponse struct {
	Data []ConsentAuthResourceResponse `json:"data"`
}
