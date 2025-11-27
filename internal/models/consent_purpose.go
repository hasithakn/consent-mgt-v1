package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"github.com/wso2/consent-management-api/internal/purpose_type_handlers"
)

// JSONValue represents a JSON value that can be stored in the database
type JSONValue json.RawMessage

// Scan implements the sql.Scanner interface for JSONValue
func (j *JSONValue) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to unmarshal JSON value: %v", value)
	}
	*j = JSONValue(bytes)
	return nil
}

// Value implements the driver.Valuer interface for JSONValue
func (j JSONValue) Value() (driver.Value, error) {
	if len(j) == 0 {
		return nil, nil
	}
	return []byte(j), nil
}

// MarshalJSON implements the json.Marshaler interface
func (j JSONValue) MarshalJSON() ([]byte, error) {
	if len(j) == 0 {
		return []byte("null"), nil
	}
	return j, nil
}

// UnmarshalJSON implements the json.Unmarshaler interface
func (j *JSONValue) UnmarshalJSON(data []byte) error {
	if j == nil {
		return fmt.Errorf("JSONValue: UnmarshalJSON on nil pointer")
	}
	*j = append((*j)[0:0], data...)
	return nil
}

// ConsentPurpose represents a consent purpose entity
type ConsentPurpose struct {
	ID          string            `json:"id" db:"ID"`
	Name        string            `json:"name" db:"NAME"`
	Description *string           `json:"description,omitempty" db:"DESCRIPTION"`
	Type        string            `json:"type" db:"TYPE"`
	Attributes  map[string]string `json:"attributes,omitempty" db:"-"`
	OrgID       string            `json:"orgId" db:"ORG_ID"`
}

// ConsentPurposeMapping represents the CONSENT_PURPOSE_MAPPING table
type ConsentPurposeMapping struct {
	ConsentID      string      `db:"CONSENT_ID" json:"consentId"`
	OrgID          string      `db:"ORG_ID" json:"orgId"`
	PurposeID      string      `db:"PURPOSE_ID" json:"purposeId"`
	Value          interface{} `db:"VALUE" json:"value,omitempty"`
	IsUserApproved bool        `db:"IS_USER_APPROVED" json:"isUserApproved"`
	IsMandatory    bool        `db:"IS_MANDATORY" json:"isMandatory"`
	Name           string      `db:"-" json:"name"` // Purpose name for convenience (not in mapping table)
}

// ConsentPurposeCreateRequest represents the request to create a consent purpose
type ConsentPurposeCreateRequest struct {
	Name        string            `json:"name" binding:"required"`
	Description string            `json:"description,omitempty"`
	Type        string            `json:"type" binding:"required"`
	Attributes  map[string]string `json:"attributes,omitempty"`
}

// ConsentPurposeUpdateRequest represents the request to update a consent purpose
// All fields are required - no partial updates allowed
type ConsentPurposeUpdateRequest struct {
	Name        string            `json:"name" binding:"required,max=255"`
	Description *string           `json:"description,omitempty" binding:"omitempty,max=1024"`
	Type        string            `json:"type" binding:"required"`
	Attributes  map[string]string `json:"attributes,omitempty"`
}

// ConsentPurposeResponse represents the response for consent purpose operations
type ConsentPurposeResponse struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description *string           `json:"description,omitempty"`
	Type        string            `json:"type"`
	Attributes  map[string]string `json:"attributes,omitempty"`
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
		Type:        cp.Type,
		Attributes:  cp.Attributes,
	}
}

// ValidatePurposeType validates that the purpose type is registered in the handler registry
func ValidatePurposeType(typeVal string) error {
	_, err := purpose_type_handlers.GetHandler(typeVal)
	if err != nil {
		// Get all registered types for helpful error message
		registeredTypes := purpose_type_handlers.GetAllHandlerTypes()
		return fmt.Errorf("invalid purpose type '%s': must be one of %v", typeVal, registeredTypes)
	}
	return nil
}
