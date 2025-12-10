package model

// ConsentStatusAudit represents the CONSENT_STATUS_AUDIT table
type ConsentStatusAudit struct {
	StatusAuditID  string  `db:"STATUS_AUDIT_ID" json:"statusAuditId"`
	ConsentID      string  `db:"CONSENT_ID" json:"consentId"`
	CurrentStatus  string  `db:"CURRENT_STATUS" json:"currentStatus"`
	ActionTime     int64   `db:"ACTION_TIME" json:"actionTime"`
	Reason         *string `db:"REASON" json:"reason,omitempty"`
	ActionBy       *string `db:"ACTION_BY" json:"actionBy,omitempty"`
	PreviousStatus *string `db:"PREVIOUS_STATUS" json:"previousStatus,omitempty"`
	OrgID          string  `db:"ORG_ID" json:"orgId"`
}

// StatusAudit is an alias for ConsentStatusAudit for backward compatibility
type StatusAudit = ConsentStatusAudit

// ConsentStatusAuditCreateRequest represents the request for creating a status audit entry
type ConsentStatusAuditCreateRequest struct {
	ConsentID      string  `json:"consentId" binding:"required"`
	CurrentStatus  string  `json:"currentStatus" binding:"required"`
	Reason         *string `json:"reason,omitempty"`
	ActionBy       *string `json:"actionBy,omitempty"`
	PreviousStatus *string `json:"previousStatus,omitempty"`
}

// ConsentStatusAuditResponse represents the response for status audit operations
type ConsentStatusAuditResponse struct {
	StatusAuditID  string  `json:"statusAuditId"`
	ConsentID      string  `json:"consentId"`
	CurrentStatus  string  `json:"currentStatus"`
	ActionTime     int64   `json:"actionTime"`
	Reason         *string `json:"reason,omitempty"`
	ActionBy       *string `json:"actionBy,omitempty"`
	PreviousStatus *string `json:"previousStatus,omitempty"`
	OrgID          string  `json:"orgId"`
}

// ConsentStatusAuditListResponse represents the list of audit entries
type ConsentStatusAuditListResponse struct {
	Data []ConsentStatusAuditResponse `json:"data"`
}
