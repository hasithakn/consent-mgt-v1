package mocks

import (
	"github.com/stretchr/testify/mock"
	"github.com/wso2/consent-management-api/internal/purpose_type_handlers"
)

// MockPurposeTypeHandler is a mock implementation of PurposeTypeHandler
type MockPurposeTypeHandler struct {
	mock.Mock
}

func (m *MockPurposeTypeHandler) GetType() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockPurposeTypeHandler) ValidateAttributes(attributes map[string]string) []purpose_type_handlers.ValidationError {
	args := m.Called(attributes)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).([]purpose_type_handlers.ValidationError)
}

func (m *MockPurposeTypeHandler) ProcessAttributes(attributes map[string]string) map[string]string {
	args := m.Called(attributes)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(map[string]string)
}

func (m *MockPurposeTypeHandler) GetAttributeSpec() *purpose_type_handlers.PurposeAttributeSpec {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*purpose_type_handlers.PurposeAttributeSpec)
}
