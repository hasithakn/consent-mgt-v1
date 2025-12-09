package mocks

import (
	"context"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/mock"
)

// MockConsentPurposeAttributeDAO is a mock implementation of ConsentPurposeAttributeDAO
type MockConsentPurposeAttributeDAO struct {
	mock.Mock
}

func (m *MockConsentPurposeAttributeDAO) GetAttributes(ctx context.Context, purposeID, orgID string) (map[string]string, error) {
	args := m.Called(ctx, purposeID, orgID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]string), args.Error(1)
}

func (m *MockConsentPurposeAttributeDAO) SaveAttributesWithTx(ctx context.Context, tx *sqlx.Tx, purposeID, orgID string, attributes map[string]string) error {
	args := m.Called(ctx, tx, purposeID, orgID, attributes)
	return args.Error(0)
}

func (m *MockConsentPurposeAttributeDAO) DeleteAttributesWithTx(ctx context.Context, tx *sqlx.Tx, purposeID, orgID string) error {
	args := m.Called(ctx, tx, purposeID, orgID)
	return args.Error(0)
}
