package model

// PurposeGroup represents a consent purpose group entity
type PurposeGroup struct {
	ID          string                `json:"id" db:"ID"`
	Name        string                `json:"name" db:"NAME"`
	Description *string               `json:"description,omitempty" db:"DESCRIPTION"`
	ClientID    string                `json:"clientId" db:"CLIENT_ID"`
	Purposes    []PurposeGroupPurpose `json:"purposes" db:"-"`
	CreatedTime int64                 `json:"createdTime" db:"CREATED_TIME"`
	UpdatedTime int64                 `json:"updatedTime" db:"UPDATED_TIME"`
	OrgID       string                `json:"-" db:"ORG_ID"`
}

// PurposeGroupPurpose represents a purpose within a group
type PurposeGroupPurpose struct {
	PurposeID   string `json:"-" db:"PURPOSE_ID"` // Internal use
	PurposeName string `json:"purposeName" db:"PURPOSE_NAME"`
	IsMandatory bool   `json:"isMandatory" db:"IS_MANDATORY"`
}

// CreateRequest represents a request to create a purpose group
type CreateRequest struct {
	Name        string         `json:"name" validate:"required,max=255"`
	Description string         `json:"description,omitempty" validate:"max=1024"`
	Purposes    []PurposeInput `json:"purposes" validate:"required,min=1,dive"`
}

// PurposeInput represents purpose input in create/update requests
type PurposeInput struct {
	PurposeName string `json:"purposeName" validate:"required"`
	IsMandatory bool   `json:"isMandatory"`
}

// UpdateRequest represents a request to update a purpose group
type UpdateRequest struct {
	Name        string         `json:"name" validate:"required,max=255"`
	Description string         `json:"description,omitempty" validate:"max=1024"`
	Purposes    []PurposeInput `json:"purposes" validate:"required,min=1,dive"`
}

// Response represents a purpose group response
type Response struct {
	ID          string                `json:"id"`
	Name        string                `json:"name"`
	Description *string               `json:"description,omitempty"`
	ClientID    string                `json:"clientId"`
	Purposes    []PurposeGroupPurpose `json:"purposes"`
	CreatedTime int64                 `json:"createdTime"`
	UpdatedTime int64                 `json:"updatedTime"`
}

// ListResponse represents a paginated list of purpose groups
type ListResponse struct {
	Data     []Response         `json:"data"`
	Metadata PaginationMetadata `json:"metadata"`
}

// PaginationMetadata represents pagination information
type PaginationMetadata struct {
	Total  int `json:"total"`
	Offset int `json:"offset"`
	Count  int `json:"count"`
	Limit  int `json:"limit"`
}

// ToResponse converts a PurposeGroup to Response
func (pg *PurposeGroup) ToResponse() Response {
	return Response{
		ID:          pg.ID,
		Name:        pg.Name,
		Description: pg.Description,
		ClientID:    pg.ClientID,
		Purposes:    pg.Purposes,
		CreatedTime: pg.CreatedTime,
		UpdatedTime: pg.UpdatedTime,
	}
}
