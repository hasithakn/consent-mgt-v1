package models

// ConsentPurpose represents a consent purpose entity
type ConsentPurpose struct {
	ID          string  `json:"id" db:"ID"`
	Name        string  `json:"name" db:"NAME"`
	Description *string `json:"description,omitempty" db:"DESCRIPTION"`
	OrgID       string  `json:"orgId" db:"ORG_ID"`
}

// ConsentPurposeMapping represents the mapping between consent and purpose
type ConsentPurposeMapping struct {
	ConsentID string `json:"consentId" db:"CONSENT_ID"`
	OrgID     string `json:"orgId" db:"ORG_ID"`
	PurposeID string `json:"purposeId" db:"PURPOSE_ID"`
}

// ConsentPurposeCreateRequest represents the request to create a consent purpose
type ConsentPurposeCreateRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description,omitempty"`
}

// ConsentPurposeUpdateRequest represents the request to update a consent purpose
type ConsentPurposeUpdateRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
}

// ConsentPurposeResponse represents the response for consent purpose operations
type ConsentPurposeResponse struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
}

// ConsentPurposeListResponse represents a list of consent purposes
type ConsentPurposeListResponse struct {
	Purposes []ConsentPurposeResponse `json:"purposes"`
	Total    int                      `json:"total"`
}

// ToConsentPurposeResponse converts ConsentPurpose to ConsentPurposeResponse
func (cp *ConsentPurpose) ToConsentPurposeResponse() *ConsentPurposeResponse {
	return &ConsentPurposeResponse{
		ID:          cp.ID,
		Name:        cp.Name,
		Description: cp.Description,
	}
}
