package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// Purpose type constants
const (
	PurposeTypeString     = "string"
	PurposeTypeJSONSchema = "json-schema"
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
	ID          string     `json:"id" db:"ID"`
	Name        string     `json:"name" db:"NAME"`
	Description *string    `json:"description,omitempty" db:"DESCRIPTION"`
	Type        string     `json:"type" db:"TYPE"`
	Value       *JSONValue `json:"value,omitempty" db:"VALUE"`
	OrgID       string     `json:"orgId" db:"ORG_ID"`
}

// ConsentPurposeMapping represents the mapping between consent and purpose
type ConsentPurposeMapping struct {
	ConsentID string `json:"consentId" db:"CONSENT_ID"`
	OrgID     string `json:"orgId" db:"ORG_ID"`
	PurposeID string `json:"purposeId" db:"PURPOSE_ID"`
}

// ConsentPurposeCreateRequest represents the request to create a consent purpose
type ConsentPurposeCreateRequest struct {
	Name        string      `json:"name" binding:"required"`
	Description string      `json:"description,omitempty"`
	Type        string      `json:"type" binding:"required,oneof=string json-schema"`
	Value       interface{} `json:"value" binding:"required"`
}

// ConsentPurposeUpdateRequest represents the request to update a consent purpose
// All fields are required - no partial updates allowed
type ConsentPurposeUpdateRequest struct {
	Name        string      `json:"name" binding:"required,max=255"`
	Description *string     `json:"description,omitempty" binding:"omitempty,max=1024"`
	Type        string      `json:"type" binding:"required,oneof=string json-schema"`
	Value       interface{} `json:"value" binding:"required"`
}

// ConsentPurposeResponse represents the response for consent purpose operations
type ConsentPurposeResponse struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Description *string    `json:"description,omitempty"`
	Type        string     `json:"type"`
	Value       *JSONValue `json:"value,omitempty"`
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
		Value:       cp.Value,
	}
}

// ValidatePurposeType validates that the purpose type is one of the allowed values
func ValidatePurposeType(typeVal string) error {
	if typeVal != PurposeTypeString && typeVal != PurposeTypeJSONSchema {
		return fmt.Errorf("invalid purpose type '%s': must be one of [%s, %s]",
			typeVal, PurposeTypeString, PurposeTypeJSONSchema)
	}
	return nil
}
