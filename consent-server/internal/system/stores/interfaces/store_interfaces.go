package interfaces

import (
	"context"

	authResourceModel "github.com/wso2/consent-management-api/internal/authresource/model"
	consentModel "github.com/wso2/consent-management-api/internal/consent/model"
	consentPurposeModel "github.com/wso2/consent-management-api/internal/consentpurpose/model"
	dbmodel "github.com/wso2/consent-management-api/internal/system/database/model"
)

// ConsentStore defines the interface for consent data operations
type ConsentStore interface {
	GetByID(ctx context.Context, consentID, orgID string) (*consentModel.Consent, error)
	List(ctx context.Context, orgID string, limit, offset int) ([]consentModel.Consent, int, error)
	Search(ctx context.Context, filters consentModel.ConsentSearchFilters) ([]consentModel.Consent, int, error)
	GetByClientID(ctx context.Context, clientID, orgID string) ([]consentModel.Consent, error)
	GetAttributesByConsentID(ctx context.Context, consentID, orgID string) ([]consentModel.ConsentAttribute, error)
	GetAttributesByConsentIDs(ctx context.Context, consentIDs []string, orgID string) (map[string]map[string]string, error)
	GetStatusAuditByConsentID(ctx context.Context, consentID, orgID string) ([]consentModel.ConsentStatusAudit, error)
	FindConsentIDsByAttributeKey(ctx context.Context, key, orgID string) ([]string, error)
	FindConsentIDsByAttribute(ctx context.Context, key, value, orgID string) ([]string, error)
	Create(tx dbmodel.TxInterface, consent *consentModel.Consent) error
	Update(tx dbmodel.TxInterface, consent *consentModel.Consent) error
	UpdateStatus(tx dbmodel.TxInterface, consentID, orgID, status string, updatedTime int64) error
	Delete(tx dbmodel.TxInterface, consentID, orgID string) error
	CreateAttributes(tx dbmodel.TxInterface, attributes []consentModel.ConsentAttribute) error
	DeleteAttributesByConsentID(tx dbmodel.TxInterface, consentID, orgID string) error
	CreateStatusAudit(tx dbmodel.TxInterface, audit *consentModel.ConsentStatusAudit) error
}

// AuthResourceStore defines the interface for authorization resource data operations
type AuthResourceStore interface {
	GetByID(ctx context.Context, authID, orgID string) (*authResourceModel.AuthResource, error)
	GetByConsentID(ctx context.Context, consentID, orgID string) ([]authResourceModel.AuthResource, error)
	GetByConsentIDs(ctx context.Context, consentIDs []string, orgID string) ([]authResourceModel.AuthResource, error)
	Exists(ctx context.Context, authID, orgID string) (bool, error)
	GetByUserID(ctx context.Context, userID, orgID string) ([]authResourceModel.AuthResource, error)
	Create(tx dbmodel.TxInterface, authResource *authResourceModel.AuthResource) error
	Update(tx dbmodel.TxInterface, authResource *authResourceModel.AuthResource) error
	UpdateStatus(tx dbmodel.TxInterface, authID, orgID, status string, updatedTime int64) error
	Delete(tx dbmodel.TxInterface, authID, orgID string) error
	DeleteByConsentID(tx dbmodel.TxInterface, consentID, orgID string) error
	UpdateAllStatusByConsentID(tx dbmodel.TxInterface, consentID, orgID, status string, updatedTime int64) error
}

// ConsentPurposeStore defines the interface for consent purpose data operations
type ConsentPurposeStore interface {
	GetByID(ctx context.Context, purposeID, orgID string) (*consentPurposeModel.ConsentPurpose, error)
	GetByName(ctx context.Context, name, orgID string) (*consentPurposeModel.ConsentPurpose, error)
	List(ctx context.Context, orgID string, limit, offset int, name string) ([]consentPurposeModel.ConsentPurpose, int, error)
	CheckNameExists(ctx context.Context, name, orgID string) (bool, error)
	GetAttributesByPurposeID(ctx context.Context, purposeID, orgID string) ([]consentPurposeModel.ConsentPurposeAttribute, error)
	GetPurposesByConsentID(ctx context.Context, consentID, orgID string) ([]consentPurposeModel.ConsentPurpose, error)
	GetMappingsByConsentID(ctx context.Context, consentID, orgID string) ([]consentPurposeModel.ConsentPurposeMapping, error)
	GetMappingsByConsentIDs(ctx context.Context, consentIDs []string, orgID string) ([]consentPurposeModel.ConsentPurposeMapping, error)
	GetIDsByNames(ctx context.Context, names []string, orgID string) (map[string]string, error)
	Create(tx dbmodel.TxInterface, purpose *consentPurposeModel.ConsentPurpose) error
	Update(tx dbmodel.TxInterface, purpose *consentPurposeModel.ConsentPurpose) error
	Delete(tx dbmodel.TxInterface, purposeID, orgID string) error
	CreateAttributes(tx dbmodel.TxInterface, attributes []consentPurposeModel.ConsentPurposeAttribute) error
	DeleteAttributesByPurposeID(tx dbmodel.TxInterface, purposeID, orgID string) error
	LinkPurposeToConsent(tx dbmodel.TxInterface, consentID, purposeID, orgID string, value *string, isUserApproved, isMandatory bool) error
	DeleteMappingsByConsentID(tx dbmodel.TxInterface, consentID, orgID string) error
}
