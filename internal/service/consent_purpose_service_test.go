package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
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

