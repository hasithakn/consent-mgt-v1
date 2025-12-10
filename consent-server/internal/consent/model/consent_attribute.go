package model

// ConsentAttribute represents the CONSENT_ATTRIBUTE table
type ConsentAttribute struct {
	ConsentID string `db:"CONSENT_ID" json:"consentId"`
	AttKey    string `db:"ATT_KEY" json:"key"`
	AttValue  string `db:"ATT_VALUE" json:"value"`
	OrgID     string `db:"ORG_ID" json:"orgId"`
}

// ConsentAttributeCreateRequest represents the request for creating consent attributes
type ConsentAttributeCreateRequest struct {
	ConsentID  string            `json:"consentId" binding:"required"`
	Attributes map[string]string `json:"attributes" binding:"required"`
}

// ConsentAttributeUpdateRequest represents the request for updating consent attributes
type ConsentAttributeUpdateRequest struct {
	Attributes map[string]string `json:"attributes" binding:"required"`
}

// ConsentAttributeResponse represents the response for attribute operations
type ConsentAttributeResponse struct {
	ConsentID  string            `json:"consentId"`
	Attributes map[string]string `json:"attributes"`
	OrgID      string            `json:"orgId"`
}

// ConsentAttributeSearchResponse represents the response for attribute search
type ConsentAttributeSearchResponse struct {
	ConsentIDs []string `json:"consentIds"`
	Count      int      `json:"count"`
}
