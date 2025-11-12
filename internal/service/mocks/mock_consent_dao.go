package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"
	"github.com/wso2/consent-management-api/internal/models"
)

// MockConsentDAO is a mock implementation of ConsentDAO
type MockConsentDAO struct {
	mock.Mock
}

func (m *MockConsentDAO) GetByID(ctx context.Context, consentID, orgID string) (*models.Consent, error) {
	args := m.Called(ctx, consentID, orgID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Consent), args.Error(1)
}

func (m *MockConsentDAO) Create(ctx context.Context, consent *models.Consent) error {
	args := m.Called(ctx, consent)
	return args.Error(0)
}

func (m *MockConsentDAO) Update(ctx context.Context, consent *models.Consent) error {
	args := m.Called(ctx, consent)
	return args.Error(0)
}

func (m *MockConsentDAO) Delete(ctx context.Context, consentID, orgID string) error {
	args := m.Called(ctx, consentID, orgID)
	return args.Error(0)
}

func (m *MockConsentDAO) List(ctx context.Context, orgID string, limit, offset int) ([]models.Consent, int, error) {
	args := m.Called(ctx, orgID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]models.Consent), args.Int(1), args.Error(2)
}
