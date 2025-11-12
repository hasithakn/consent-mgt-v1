package mocks

import (
	"context"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/mock"
	"github.com/wso2/consent-management-api/internal/models"
)

// MockConsentPurposeDAO is a mock implementation of ConsentPurposeDAO
type MockConsentPurposeDAO struct {
	mock.Mock
}

func (m *MockConsentPurposeDAO) Create(ctx context.Context, purpose *models.ConsentPurpose) error {
	args := m.Called(ctx, purpose)
	return args.Error(0)
}

func (m *MockConsentPurposeDAO) CreateWithTx(ctx context.Context, tx *sqlx.Tx, purpose *models.ConsentPurpose) error {
	args := m.Called(ctx, tx, purpose)
	return args.Error(0)
}

func (m *MockConsentPurposeDAO) GetByID(ctx context.Context, purposeID, orgID string) (*models.ConsentPurpose, error) {
	args := m.Called(ctx, purposeID, orgID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ConsentPurpose), args.Error(1)
}

func (m *MockConsentPurposeDAO) List(ctx context.Context, orgID string, limit, offset int) ([]models.ConsentPurpose, int, error) {
	args := m.Called(ctx, orgID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]models.ConsentPurpose), args.Int(1), args.Error(2)
}

func (m *MockConsentPurposeDAO) Update(ctx context.Context, purpose *models.ConsentPurpose) error {
	args := m.Called(ctx, purpose)
	return args.Error(0)
}

func (m *MockConsentPurposeDAO) UpdateWithTx(ctx context.Context, tx *sqlx.Tx, purpose *models.ConsentPurpose) error {
	args := m.Called(ctx, tx, purpose)
	return args.Error(0)
}

func (m *MockConsentPurposeDAO) Delete(ctx context.Context, purposeID, orgID string) error {
	args := m.Called(ctx, purposeID, orgID)
	return args.Error(0)
}

func (m *MockConsentPurposeDAO) DeleteWithTx(ctx context.Context, tx *sqlx.Tx, purposeID, orgID string) error {
	args := m.Called(ctx, tx, purposeID, orgID)
	return args.Error(0)
}

func (m *MockConsentPurposeDAO) ExistsByName(ctx context.Context, name, orgID string) (bool, error) {
	args := m.Called(ctx, name, orgID)
	return args.Bool(0), args.Error(1)
}

func (m *MockConsentPurposeDAO) ExistsByNameWithTx(ctx context.Context, tx *sqlx.Tx, name, orgID string) (bool, error) {
	args := m.Called(ctx, tx, name, orgID)
	return args.Bool(0), args.Error(1)
}

func (m *MockConsentPurposeDAO) GetByConsentID(ctx context.Context, consentID, orgID string) ([]models.ConsentPurpose, error) {
	args := m.Called(ctx, consentID, orgID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.ConsentPurpose), args.Error(1)
}

func (m *MockConsentPurposeDAO) LinkPurposeToConsent(ctx context.Context, consentID, purposeID, orgID string) error {
	args := m.Called(ctx, consentID, purposeID, orgID)
	return args.Error(0)
}

func (m *MockConsentPurposeDAO) LinkPurposeToConsentWithTx(ctx context.Context, tx *sqlx.Tx, consentID, purposeID, orgID string) error {
	args := m.Called(ctx, tx, consentID, purposeID, orgID)
	return args.Error(0)
}

func (m *MockConsentPurposeDAO) UnlinkPurposeFromConsent(ctx context.Context, consentID, purposeID, orgID string) error {
	args := m.Called(ctx, consentID, purposeID, orgID)
	return args.Error(0)
}

func (m *MockConsentPurposeDAO) GetConsentsByPurpose(ctx context.Context, purposeID, orgID string, limit, offset int) ([]string, int, error) {
	args := m.Called(ctx, purposeID, orgID, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]string), args.Int(1), args.Error(2)
}

func (m *MockConsentPurposeDAO) ClearConsentPurposes(ctx context.Context, consentID, orgID string) error {
	args := m.Called(ctx, consentID, orgID)
	return args.Error(0)
}

func (m *MockConsentPurposeDAO) ClearConsentPurposesWithTx(ctx context.Context, tx *sqlx.Tx, consentID, orgID string) error {
	args := m.Called(ctx, tx, consentID, orgID)
	return args.Error(0)
}

func (m *MockConsentPurposeDAO) GetIDsByNames(ctx context.Context, names []string, orgID string) (map[string]string, error) {
	args := m.Called(ctx, names, orgID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]string), args.Error(1)
}

func (m *MockConsentPurposeDAO) ValidatePurposeNames(ctx context.Context, names []string, orgID string) ([]string, error) {
	args := m.Called(ctx, names, orgID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}
