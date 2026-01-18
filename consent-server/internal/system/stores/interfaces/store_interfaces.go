package interfaces

import (
	"context"

	authResourceModel "github.com/wso2/consent-management-api/internal/authresource/model"
	consentModel "github.com/wso2/consent-management-api/internal/consent/model"
	consentElementModel "github.com/wso2/consent-management-api/internal/consentelement/model"
	consentPurposeGroupModel "github.com/wso2/consent-management-api/internal/consentpurposegroup/model"
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
	// Purpose Group Consent mapping methods
	CreatePurposeGroupConsent(tx dbmodel.TxInterface, consentID, groupID, orgID string) error
	CreatePurposeApproval(tx dbmodel.TxInterface, approval *consentModel.ConsentPurposeApprovalRecord) error
	GetPurposeGroupsByConsentID(ctx context.Context, consentID, orgID string) ([]consentModel.ConsentPurposeGroupMapping, error)
	GetPurposeApprovalsByConsentID(ctx context.Context, consentID, orgID string) ([]consentModel.ConsentPurposeApprovalRecord, error)
	DeletePurposeGroupsByConsentID(tx dbmodel.TxInterface, consentID, orgID string) error
	DeletePurposeApprovalsByConsentID(tx dbmodel.TxInterface, consentID, orgID string) error
	CheckGroupUsedInConsents(ctx context.Context, groupID, orgID string) (bool, error)
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

// ConsentElementStore defines the interface for consent purpose data operations
type ConsentElementStore interface {
	GetByID(ctx context.Context, purposeID, orgID string) (*consentElementModel.ConsentElement, error)
	GetByName(ctx context.Context, name, orgID string) (*consentElementModel.ConsentElement, error)
	List(ctx context.Context, orgID string, limit, offset int, name string) ([]consentElementModel.ConsentElement, int, error)
	CheckNameExists(ctx context.Context, name, orgID string) (bool, error)
	GetPropertiesByElementID(ctx context.Context, elementID, orgID string) ([]consentElementModel.ConsentElementProperty, error)
	GetMappingsByConsentID(ctx context.Context, consentID, orgID string) ([]consentElementModel.ConsentElementMapping, error)
	GetMappingsByConsentIDs(ctx context.Context, consentIDs []string, orgID string) ([]consentElementModel.ConsentElementMapping, error)
	GetIDsByNames(ctx context.Context, names []string, orgID string) (map[string]string, error)
	Create(tx dbmodel.TxInterface, element *consentElementModel.ConsentElement) error
	Update(tx dbmodel.TxInterface, element *consentElementModel.ConsentElement) error
	Delete(tx dbmodel.TxInterface, elementID, orgID string) error
	CreateProperties(tx dbmodel.TxInterface, properties []consentElementModel.ConsentElementProperty) error
	DeletePropertiesByElementID(tx dbmodel.TxInterface, elementID, orgID string) error
	LinkElementToConsent(tx dbmodel.TxInterface, consentID, elementID, orgID string, value *string, isUserApproved, isMandatory bool) error
	DeleteMappingsByConsentID(tx dbmodel.TxInterface, consentID, orgID string) error
}

// ConsentPurposeGroupStore defines the interface for purpose group data operations
type ConsentPurposeGroupStore interface {
	CreateGroup(tx dbmodel.TxInterface, group *consentPurposeGroupModel.PurposeGroup) error
	GetGroupByID(ctx context.Context, groupID, orgID string) (*consentPurposeGroupModel.PurposeGroup, error)
	GetGroupByName(ctx context.Context, name, orgID string) (*consentPurposeGroupModel.PurposeGroup, error)
	ListGroups(ctx context.Context, orgID, name string, clientIDs []string, purposeNames []string, offset, limit int) ([]consentPurposeGroupModel.PurposeGroup, int, error)
	UpdateGroup(tx dbmodel.TxInterface, group *consentPurposeGroupModel.PurposeGroup) error
	DeleteGroup(tx dbmodel.TxInterface, groupID, orgID string) error
	CheckGroupNameExists(ctx context.Context, name, clientID, orgID string, excludeGroupID *string) (bool, error)
	LinkPurposeToGroup(tx dbmodel.TxInterface, groupID, purposeID, orgID string, isMandatory bool) error
	GetGroupPurposes(ctx context.Context, groupID, orgID string) ([]consentPurposeGroupModel.PurposeGroupPurpose, error)
	DeleteGroupPurposes(tx dbmodel.TxInterface, groupID, orgID string) error
	GetPurposeIDByName(ctx context.Context, purposeName, orgID string) (string, error)
	ValidatePurposeNames(ctx context.Context, purposeNames []string, orgID string) (map[string]string, error)
	IsPurposeUsedInGroups(ctx context.Context, purposeID, orgID string) (bool, error)
}
