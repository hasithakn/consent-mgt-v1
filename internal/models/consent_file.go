package models

// ConsentFile represents the FS_CONSENT_FILE table
type ConsentFile struct {
	ConsentID   string `db:"CONSENT_ID" json:"consentId"`
	ConsentFile []byte `db:"CONSENT_FILE" json:"-"`
	OrgID       string `db:"ORG_ID" json:"orgId"`
}

// ConsentFileUploadRequest represents the request for uploading a consent file
type ConsentFileUploadRequest struct {
	ConsentID string `json:"consentId" binding:"required"`
	File      []byte `json:"file" binding:"required"`
}

// ConsentFileUpdateRequest represents the request for updating a consent file
type ConsentFileUpdateRequest struct {
	File []byte `json:"file" binding:"required"`
}

// ConsentFileResponse represents the response for file operations
type ConsentFileResponse struct {
	ConsentID string `json:"consentId"`
	FileSize  int    `json:"fileSize"`
	OrgID     string `json:"orgId"`
	Message   string `json:"message"`
}

// ConsentFileDownloadResponse represents the file download response
type ConsentFileDownloadResponse struct {
	ConsentID   string `json:"consentId"`
	File        []byte `json:"file"`
	OrgID       string `json:"orgId"`
	ContentType string `json:"contentType,omitempty"`
}
