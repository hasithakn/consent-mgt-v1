package service

import (
	"github.com/sirupsen/logrus"
	"github.com/wso2/consent-management-api/internal/service/mocks"
)

// TestSetup contains common test dependencies
type TestSetup struct {
	MockPurposeDAO   *mocks.MockConsentPurposeDAO
	MockAttributeDAO *mocks.MockConsentPurposeAttributeDAO
	MockConsentDAO   *mocks.MockConsentDAO
	Service          *ConsentPurposeService
	Logger           *logrus.Logger
}

// NewTestSetup creates a new test setup with mocks
// Note: The service needs to be created with mocks manually in each test
func NewTestSetup() *TestSetup {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	mockPurposeDAO := &mocks.MockConsentPurposeDAO{}
	mockAttributeDAO := &mocks.MockConsentPurposeAttributeDAO{}
	mockConsentDAO := &mocks.MockConsentDAO{}

	return &TestSetup{
		MockPurposeDAO:   mockPurposeDAO,
		MockAttributeDAO: mockAttributeDAO,
		MockConsentDAO:   mockConsentDAO,
		Logger:           logger,
	}
}

// Helper to create a valid create request
func NewValidCreateRequest() *ConsentPurposeCreateRequest {
	return &ConsentPurposeCreateRequest{
		Name:        "Test Purpose",
		Description: strPtr("Test Description"),
		Type:        "string",
		Attributes:  map[string]string{},
	}
}

// Helper to create a valid update request
func NewValidUpdateRequest() *ConsentPurposeUpdateRequest {
	return &ConsentPurposeUpdateRequest{
		Name:        "Updated Purpose",
		Description: strPtr("Updated Description"),
		Type:        "string",
		Attributes:  map[string]string{},
	}
}

// Helper to create a pointer to a string
func strPtr(s string) *string {
	return &s
}
