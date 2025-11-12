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
	assert.Contains(t, err.Error(), "required")
}
