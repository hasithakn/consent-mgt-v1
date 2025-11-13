package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wso2/consent-management-api/internal/models"
)

// TestCreatePurpose_ValidatesEmptyName tests that CreatePurpose rejects empty names
func TestCreatePurpose_ValidatesEmptyName(t *testing.T) {
	service := &ConsentPurposeService{
		purposeDAO:          nil,
		purposeAttributeDAO: nil,
		consentDAO:          nil,
		db:                  nil,
		logger:              nil,
	}

	request := &ConsentPurposeCreateRequest{
		Name:       "",
		Type:       "string",
		Attributes: map[string]string{},
	}

	resp, err := service.CreatePurpose(context.Background(), "org-123", request)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, "purpose name is required", err.Error())
}

// TestCreatePurpose_ValidatesEmptyType tests that CreatePurpose rejects empty type
func TestCreatePurpose_ValidatesEmptyType(t *testing.T) {
	service := &ConsentPurposeService{
		purposeDAO:          nil,
		purposeAttributeDAO: nil,
		consentDAO:          nil,
		db:                  nil,
		logger:              nil,
	}

	request := &ConsentPurposeCreateRequest{
		Name:       "Marketing",
		Type:       "",
		Attributes: map[string]string{},
	}

	resp, err := service.CreatePurpose(context.Background(), "org-123", request)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, "purpose type is required", err.Error())
}

// TestCreatePurpose_ValidatesEmptyOrgID tests that CreatePurpose rejects empty org ID
func TestCreatePurpose_ValidatesEmptyOrgID(t *testing.T) {
	service := &ConsentPurposeService{
		purposeDAO:          nil,
		purposeAttributeDAO: nil,
		consentDAO:          nil,
		db:                  nil,
		logger:              nil,
	}

	request := &ConsentPurposeCreateRequest{
		Name:       "Marketing",
		Type:       "string",
		Attributes: map[string]string{},
	}

	resp, err := service.CreatePurpose(context.Background(), "", request)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, "organization ID cannot be empty", err.Error())
}

// TestUpdatePurpose_ValidatesEmptyPurposeID tests that UpdatePurpose rejects empty purpose ID
func TestUpdatePurpose_ValidatesEmptyPurposeID(t *testing.T) {
	service := &ConsentPurposeService{
		purposeDAO:          nil,
		purposeAttributeDAO: nil,
		consentDAO:          nil,
		db:                  nil,
		logger:              nil,
	}

	request := &ConsentPurposeUpdateRequest{
		Name:       "Updated",
		Type:       "string",
		Attributes: map[string]string{},
	}

	resp, err := service.UpdatePurpose(context.Background(), "", "org-123", request)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, "purpose ID is required", err.Error())
}

// TestUpdatePurpose_ValidatesEmptyOrgID tests that UpdatePurpose rejects empty org ID
func TestUpdatePurpose_ValidatesEmptyOrgID(t *testing.T) {
	service := &ConsentPurposeService{
		purposeDAO:          nil,
		purposeAttributeDAO: nil,
		consentDAO:          nil,
		db:                  nil,
		logger:              nil,
	}

	request := &ConsentPurposeUpdateRequest{
		Name:       "Updated",
		Type:       "string",
		Attributes: map[string]string{},
	}

	resp, err := service.UpdatePurpose(context.Background(), "purpose-123", "", request)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, "organization ID cannot be empty", err.Error())
}

// TestDeletePurpose_ValidatesEmptyPurposeID tests that DeletePurpose rejects empty purpose ID
func TestDeletePurpose_ValidatesEmptyPurposeID(t *testing.T) {
	service := &ConsentPurposeService{
		purposeDAO:          nil,
		purposeAttributeDAO: nil,
		consentDAO:          nil,
		db:                  nil,
		logger:              nil,
	}

	err := service.DeletePurpose(context.Background(), "", "org-123")

	assert.Error(t, err)
	assert.Equal(t, "purpose ID is required", err.Error())
}

// TestDeletePurpose_ValidatesEmptyOrgID tests that DeletePurpose rejects empty org ID
func TestDeletePurpose_ValidatesEmptyOrgID(t *testing.T) {
	service := &ConsentPurposeService{
		purposeDAO:          nil,
		purposeAttributeDAO: nil,
		consentDAO:          nil,
		db:                  nil,
		logger:              nil,
	}

	err := service.DeletePurpose(context.Background(), "purpose-123", "")

	assert.Error(t, err)
	assert.Equal(t, "organization ID cannot be empty", err.Error())
}

// TestGetPurpose_ValidatesEmptyPurposeID tests that GetPurpose rejects empty purpose ID
func TestGetPurpose_ValidatesEmptyPurposeID(t *testing.T) {
	service := &ConsentPurposeService{
		purposeDAO:          nil,
		purposeAttributeDAO: nil,
		consentDAO:          nil,
		db:                  nil,
		logger:              nil,
	}

	resp, err := service.GetPurpose(context.Background(), "", "org-123")

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, "purpose ID is required", err.Error())
}

// TestGetPurpose_ValidatesEmptyOrgID tests that GetPurpose rejects empty org ID
func TestGetPurpose_ValidatesEmptyOrgID(t *testing.T) {
	service := &ConsentPurposeService{
		purposeDAO:          nil,
		purposeAttributeDAO: nil,
		consentDAO:          nil,
		db:                  nil,
		logger:              nil,
	}

	resp, err := service.GetPurpose(context.Background(), "purpose-123", "")

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, "organization ID cannot be empty", err.Error())
}

// TestListPurposes_ValidatesEmptyOrgID tests that ListPurposes rejects empty org ID
func TestListPurposes_ValidatesEmptyOrgID(t *testing.T) {
	service := &ConsentPurposeService{
		purposeDAO:          nil,
		purposeAttributeDAO: nil,
		consentDAO:          nil,
		db:                  nil,
		logger:              nil,
	}

	resp, err := service.ListPurposes(context.Background(), "", 10, 0)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, "organization ID cannot be empty", err.Error())
}

// ============================================================================
// CREATE OPERATION - COMPREHENSIVE TESTS
// ============================================================================

// TestCreatePurpose_ValidatesInvalidType tests that CreatePurpose rejects invalid purpose type
func TestCreatePurpose_ValidatesInvalidType(t *testing.T) {
	service := &ConsentPurposeService{
		purposeDAO:          nil,
		purposeAttributeDAO: nil,
		consentDAO:          nil,
		db:                  nil,
		logger:              nil,
	}

	request := &ConsentPurposeCreateRequest{
		Name:       "Marketing",
		Type:       "unknown-type",
		Attributes: map[string]string{},
	}

	resp, err := service.CreatePurpose(context.Background(), "org-123", request)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "invalid purpose type")
}

// ============================================================================
// UPDATE OPERATION - COMPREHENSIVE TESTS
// ============================================================================

// TestUpdatePurpose_ValidatesEmptyName tests that UpdatePurpose rejects empty name
func TestUpdatePurpose_ValidatesEmptyName(t *testing.T) {
	service := &ConsentPurposeService{
		purposeDAO:          nil,
		purposeAttributeDAO: nil,
		consentDAO:          nil,
		db:                  nil,
		logger:              nil,
	}

	request := &ConsentPurposeUpdateRequest{
		Name:       "",
		Type:       "string",
		Attributes: map[string]string{},
	}

	resp, err := service.UpdatePurpose(context.Background(), "purpose-123", "org-123", request)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, "purpose name is required", err.Error())
}

// TestUpdatePurpose_ValidatesEmptyType tests that UpdatePurpose rejects empty type
func TestUpdatePurpose_ValidatesEmptyType(t *testing.T) {
	service := &ConsentPurposeService{
		purposeDAO:          nil,
		purposeAttributeDAO: nil,
		consentDAO:          nil,
		db:                  nil,
		logger:              nil,
	}

	request := &ConsentPurposeUpdateRequest{
		Name:       "Updated",
		Type:       "",
		Attributes: map[string]string{},
	}

	resp, err := service.UpdatePurpose(context.Background(), "purpose-123", "org-123", request)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, "purpose type is required", err.Error())
}

// TestUpdatePurpose_ValidatesInvalidType tests that UpdatePurpose rejects invalid type
func TestUpdatePurpose_ValidatesInvalidType(t *testing.T) {
	service := &ConsentPurposeService{
		purposeDAO:          nil,
		purposeAttributeDAO: nil,
		consentDAO:          nil,
		db:                  nil,
		logger:              nil,
	}

	request := &ConsentPurposeUpdateRequest{
		Name:       "Updated",
		Type:       "invalid-type",
		Attributes: map[string]string{},
	}

	resp, err := service.UpdatePurpose(context.Background(), "purpose-123", "org-123", request)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "invalid purpose type")
}

// ============================================================================
// HELPER FUNCTION TESTS
// ============================================================================

// TestBuildPurposeResponse tests the buildPurposeResponse helper
func TestBuildPurposeResponse(t *testing.T) {
	service := &ConsentPurposeService{
		purposeDAO:          nil,
		purposeAttributeDAO: nil,
		consentDAO:          nil,
		db:                  nil,
		logger:              nil,
	}

	tests := []struct {
		name     string
		purpose  *models.ConsentPurpose
		validate func(*testing.T, *models.ConsentPurposeResponse)
	}{
		{
			name: "builds response with all fields",
			purpose: &models.ConsentPurpose{
				ID:          "purpose-123",
				Name:        "Marketing",
				Description: strPtr("Marketing purposes"),
				Type:        "string",
				OrgID:       "org-123",
				Attributes: map[string]string{
					"category": "marketing",
					"region":   "US",
				},
			},
			validate: func(t *testing.T, resp *models.ConsentPurposeResponse) {
				assert.NotNil(t, resp)
				assert.Equal(t, "purpose-123", resp.ID)
				assert.Equal(t, "Marketing", resp.Name)
				assert.NotNil(t, resp.Description)
				assert.Equal(t, "Marketing purposes", *resp.Description)
				assert.Equal(t, "string", resp.Type)
				assert.Len(t, resp.Attributes, 2)
				assert.Equal(t, "marketing", resp.Attributes["category"])
				assert.Equal(t, "US", resp.Attributes["region"])
			},
		},
		{
			name: "builds response without description",
			purpose: &models.ConsentPurpose{
				ID:          "purpose-456",
				Name:        "Analytics",
				Description: nil,
				Type:        "json-schema",
				OrgID:       "org-456",
				Attributes:  map[string]string{},
			},
			validate: func(t *testing.T, resp *models.ConsentPurposeResponse) {
				assert.NotNil(t, resp)
				assert.Equal(t, "purpose-456", resp.ID)
				assert.Equal(t, "Analytics", resp.Name)
				assert.Nil(t, resp.Description)
				assert.Equal(t, "json-schema", resp.Type)
				assert.Empty(t, resp.Attributes)
			},
		},
		{
			name: "builds response with nil attributes",
			purpose: &models.ConsentPurpose{
				ID:          "purpose-789",
				Name:        "Sales",
				Description: strPtr("Sales tracking"),
				Type:        "attribute",
				OrgID:       "org-789",
				Attributes:  nil,
			},
			validate: func(t *testing.T, resp *models.ConsentPurposeResponse) {
				assert.NotNil(t, resp)
				assert.Equal(t, "purpose-789", resp.ID)
				assert.Equal(t, "Sales", resp.Name)
				assert.NotNil(t, resp.Description)
				assert.Equal(t, "Sales tracking", *resp.Description)
				assert.Equal(t, "attribute", resp.Type)
				assert.Nil(t, resp.Attributes)
			},
		},
		{
			name: "builds response with empty name",
			purpose: &models.ConsentPurpose{
				ID:          "purpose-000",
				Name:        "",
				Description: nil,
				Type:        "string",
				OrgID:       "org-000",
				Attributes:  map[string]string{},
			},
			validate: func(t *testing.T, resp *models.ConsentPurposeResponse) {
				assert.NotNil(t, resp)
				assert.Equal(t, "purpose-000", resp.ID)
				assert.Equal(t, "", resp.Name)
				assert.Nil(t, resp.Description)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := service.buildPurposeResponse(tt.purpose)
			tt.validate(t, resp)
		})
	}
}

// TestValidateCreateRequest tests the validateCreateRequest helper
func TestValidateCreateRequest(t *testing.T) {
	service := &ConsentPurposeService{
		purposeDAO:          nil,
		purposeAttributeDAO: nil,
		consentDAO:          nil,
		db:                  nil,
		logger:              nil,
	}

	tests := []struct {
		name        string
		request     *ConsentPurposeCreateRequest
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid request",
			request: &ConsentPurposeCreateRequest{
				Name:       "Marketing",
				Type:       "string",
				Attributes: map[string]string{},
			},
			expectError: false,
		},
		{
			name: "empty name",
			request: &ConsentPurposeCreateRequest{
				Name:       "",
				Type:       "string",
				Attributes: map[string]string{},
			},
			expectError: true,
			errorMsg:    "purpose name is required",
		},
		{
			name: "name too long",
			request: &ConsentPurposeCreateRequest{
				Name:       string(make([]byte, 300)),
				Type:       "string",
				Attributes: map[string]string{},
			},
			expectError: true,
			errorMsg:    "purpose name too long",
		},
		{
			name: "description too long",
			request: &ConsentPurposeCreateRequest{
				Name:        "Marketing",
				Description: strPtr(string(make([]byte, 2000))),
				Type:        "string",
				Attributes:  map[string]string{},
			},
			expectError: true,
			errorMsg:    "purpose description too long",
		},
		{
			name: "empty type",
			request: &ConsentPurposeCreateRequest{
				Name:       "Marketing",
				Type:       "",
				Attributes: map[string]string{},
			},
			expectError: true,
			errorMsg:    "purpose type is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.validateCreateRequest(tt.request)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidateUpdateRequest tests the validateUpdateRequest helper
func TestValidateUpdateRequest(t *testing.T) {
	service := &ConsentPurposeService{
		purposeDAO:          nil,
		purposeAttributeDAO: nil,
		consentDAO:          nil,
		db:                  nil,
		logger:              nil,
	}

	tests := []struct {
		name        string
		request     *ConsentPurposeUpdateRequest
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid request",
			request: &ConsentPurposeUpdateRequest{
				Name:       "Updated Marketing",
				Type:       "string",
				Attributes: map[string]string{},
			},
			expectError: false,
		},
		{
			name: "empty name",
			request: &ConsentPurposeUpdateRequest{
				Name:       "",
				Type:       "string",
				Attributes: map[string]string{},
			},
			expectError: true,
			errorMsg:    "purpose name is required",
		},
		{
			name: "name too long",
			request: &ConsentPurposeUpdateRequest{
				Name:       string(make([]byte, 300)),
				Type:       "string",
				Attributes: map[string]string{},
			},
			expectError: true,
			errorMsg:    "purpose name too long",
		},
		{
			name: "description too long",
			request: &ConsentPurposeUpdateRequest{
				Name:        "Updated",
				Description: strPtr(string(make([]byte, 2000))),
				Type:        "string",
				Attributes:  map[string]string{},
			},
			expectError: true,
			errorMsg:    "purpose description too long",
		},
		{
			name: "empty type",
			request: &ConsentPurposeUpdateRequest{
				Name:       "Updated",
				Type:       "",
				Attributes: map[string]string{},
			},
			expectError: true,
			errorMsg:    "purpose type is required",
		},
		{
			name: "valid with description",
			request: &ConsentPurposeUpdateRequest{
				Name:        "Updated Marketing",
				Description: strPtr("New description"),
				Type:        "string",
				Attributes:  map[string]string{"key": "value"},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.validateUpdateRequest(tt.request)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidateAttributesForType tests the validateAttributesForType method
func TestValidateAttributesForType(t *testing.T) {
	service := &ConsentPurposeService{}

	tests := []struct {
		name        string
		purposeType string
		attributes  map[string]string
		expectError bool
		errorField  string
		errorMsg    string
	}{
		{
			name:        "valid string type with optional attributes",
			purposeType: "string",
			attributes: map[string]string{
				"validationSchema": `{"type":"string"}`,
			},
			expectError: false,
		},
		{
			name:        "string type has no mandatory attributes",
			purposeType: "string",
			attributes:  map[string]string{},
			expectError: false, // String type has no mandatory attributes
		},
		{
			name:        "unknown purpose type",
			purposeType: "unknown_type",
			attributes:  map[string]string{},
			expectError: true,
			errorField:  "type",
			errorMsg:    "unknown purpose type: unknown_type",
		},
		{
			name:        "valid json-schema type with required validationSchema",
			purposeType: "json-schema",
			attributes: map[string]string{
				"validationSchema": `{"type": "object"}`,
			},
			expectError: false,
		},
		{
			name:        "json-schema type missing required validationSchema",
			purposeType: "json-schema",
			attributes:  map[string]string{},
			expectError: true,
			errorField:  "validationSchema",
			errorMsg:    "validationSchema is required for json-schema type",
		},
		{
			name:        "json-schema type with invalid JSON",
			purposeType: "json-schema",
			attributes: map[string]string{
				"validationSchema": `not valid json`,
			},
			expectError: true,
			errorField:  "validationSchema",
			errorMsg:    "validationSchema must be valid JSON",
		},
		{
			name:        "valid attribute type with required fields",
			purposeType: "attribute",
			attributes: map[string]string{
				"resourcePath": "/accounts",
				"jsonPath":     "Data.amount",
			},
			expectError: false,
		},
		{
			name:        "attribute type missing required resourcePath",
			purposeType: "attribute",
			attributes: map[string]string{
				"jsonPath": "Data.amount",
			},
			expectError: true,
			errorField:  "resourcePath",
			errorMsg:    "resourcePath is required for attribute type",
		},
		{
			name:        "attribute type missing required jsonPath",
			purposeType: "attribute",
			attributes: map[string]string{
				"resourcePath": "/accounts",
			},
			expectError: true,
			errorField:  "jsonPath",
			errorMsg:    "jsonPath is required for attribute type",
		},
		{
			name:        "attribute type missing both required fields",
			purposeType: "attribute",
			attributes:  map[string]string{},
			expectError: true,
			errorField:  "resourcePath",
			errorMsg:    "resourcePath is required for attribute type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := service.validateAttributesForType(tt.purposeType, tt.attributes)

			if tt.expectError {
				assert.NotEmpty(t, errors)
				assert.Equal(t, tt.errorField, errors[0].Field)
				assert.Equal(t, tt.errorMsg, errors[0].Message)
			} else {
				assert.Empty(t, errors)
			}
		})
	}
}

// TestProcessAttributesForType tests the processAttributesForType method
func TestProcessAttributesForType(t *testing.T) {
	service := &ConsentPurposeService{}

	tests := []struct {
		name        string
		purposeType string
		attributes  map[string]string
		expected    map[string]string
	}{
		{
			name:        "string type returns attributes as-is",
			purposeType: "string",
			attributes: map[string]string{
				"validationSchema": `{"type":"string"}`,
				"resourcePath":     "/accounts",
			},
			expected: map[string]string{
				"validationSchema": `{"type":"string"}`,
				"resourcePath":     "/accounts",
			},
		},
		{
			name:        "json-schema type returns attributes as-is",
			purposeType: "json-schema",
			attributes: map[string]string{
				"validationSchema": `{"type": "object"}`,
			},
			expected: map[string]string{
				"validationSchema": `{"type": "object"}`,
			},
		},
		{
			name:        "attribute type returns attributes as-is",
			purposeType: "attribute",
			attributes: map[string]string{
				"resourcePath": "/accounts",
				"jsonPath":     "Data.amount",
			},
			expected: map[string]string{
				"resourcePath": "/accounts",
				"jsonPath":     "Data.amount",
			},
		},
		{
			name:        "unknown type returns attributes as-is",
			purposeType: "unknown_type",
			attributes: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
			expected: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
		},
		{
			name:        "empty attributes map",
			purposeType: "string",
			attributes:  map[string]string{},
			expected:    map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.processAttributesForType(tt.purposeType, tt.attributes)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestExistsByName_ValidationOnly tests only the validation logic of ExistsByName
func TestExistsByName_ValidationOnly(t *testing.T) {
	service := &ConsentPurposeService{}

	tests := []struct {
		name      string
		nameProp  string
		orgID     string
		expectErr string
	}{
		{
			name:      "empty name",
			nameProp:  "",
			orgID:     "org123",
			expectErr: "purpose name is required",
		},
		{
			name:      "empty org ID",
			nameProp:  "test purpose",
			orgID:     "",
			expectErr: "organization ID is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.ExistsByName(context.Background(), tt.nameProp, tt.orgID)
			assert.Error(t, err)
			assert.Equal(t, tt.expectErr, err.Error())
		})
	}
}

// TestValidatePurposeNames_ValidationOnly tests only the validation logic of ValidatePurposeNames
func TestValidatePurposeNames_ValidationOnly(t *testing.T) {
	service := &ConsentPurposeService{}

	tests := []struct {
		name      string
		orgID     string
		names     []string
		expectErr string
	}{
		{
			name:      "empty org ID",
			orgID:     "",
			names:     []string{"purpose1"},
			expectErr: "organization ID is required",
		},
		{
			name:      "empty names list",
			orgID:     "org123",
			names:     []string{},
			expectErr: "at least one purpose name must be provided",
		},
		{
			name:      "name is empty string",
			orgID:     "org123",
			names:     []string{"purpose1", ""},
			expectErr: "purpose name cannot be empty",
		},
		{
			name:      "name too long - exactly 101 chars",
			orgID:     "org123",
			names:     []string{string(make([]byte, 101))},
			expectErr: "purpose name too long (max 100 characters)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.ValidatePurposeNames(context.Background(), tt.orgID, tt.names)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectErr)
		})
	}
}
