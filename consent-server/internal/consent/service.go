package consent

import (
	"context"
	"encoding/json"
	"fmt"

	authmodel "github.com/wso2/consent-management-api/internal/authresource/model"
	"github.com/wso2/consent-management-api/internal/consent/model"
	"github.com/wso2/consent-management-api/internal/consent/validator"
	"github.com/wso2/consent-management-api/internal/system/database"
	"github.com/wso2/consent-management-api/internal/system/error/serviceerror"
	"github.com/wso2/consent-management-api/internal/system/utils"
)

// ConsentService defines the exported service interface
type ConsentService interface {
	CreateConsent(ctx context.Context, req model.ConsentAPIRequest, clientID, orgID string) (*model.ConsentResponse, *serviceerror.ServiceError)
	GetConsent(ctx context.Context, consentID, orgID string) (*model.ConsentResponse, *serviceerror.ServiceError)
	ListConsents(ctx context.Context, orgID string, limit, offset int) ([]model.ConsentResponse, int, *serviceerror.ServiceError)
	UpdateConsent(ctx context.Context, consentID string, req model.ConsentAPIUpdateRequest, orgID string) (*model.ConsentResponse, *serviceerror.ServiceError)
	UpdateConsentStatus(ctx context.Context, consentID, orgID string, req model.ConsentRevokeRequest) *serviceerror.ServiceError
	DeleteConsent(ctx context.Context, consentID, orgID string) *serviceerror.ServiceError
	GetConsentsByClientID(ctx context.Context, clientID, orgID string) ([]model.ConsentResponse, *serviceerror.ServiceError)
}

// AuthResourceStore interface for auth resource operations needed by consent service
// Exported to allow dependency injection from other modules
type AuthResourceStore interface {
	CreateWithTx(ctx context.Context, tx *database.Tx, authResource *authmodel.AuthResource) error
	UpdateAllStatusByConsentIDWithTx(ctx context.Context, tx *database.Tx, consentID, orgID, status string, updatedTime int64) error
}

// ConsentPurposeStore interface for purpose operations needed by consent service
// Exported to allow dependency injection from other modules
type ConsentPurposeStore interface {
	LinkPurposeToConsentWithTx(ctx context.Context, tx *database.Tx, consentID, purposeID, orgID string, value *string, isUserApproved, isMandatory bool) error
}

// consentService implements the ConsentService interface
type consentService struct {
	store               consentStore
	authResourceStore   AuthResourceStore
	consentPurposeStore ConsentPurposeStore
	db                  *database.DB
}

// newConsentService creates a new consent service
func newConsentService(store consentStore, authResourceStore AuthResourceStore, consentPurposeStore ConsentPurposeStore, db *database.DB) ConsentService {
	return &consentService{
		store:               store,
		authResourceStore:   authResourceStore,
		consentPurposeStore: consentPurposeStore,
		db:                  db,
	}
}

// CreateConsent creates a new consent with all related entities in a single transaction
func (s *consentService) CreateConsent(ctx context.Context, req model.ConsentAPIRequest, clientID, orgID string) (*model.ConsentResponse, *serviceerror.ServiceError) {
	// Validate request
	if err := utils.ValidateOrgID(orgID); err != nil {
		return nil, serviceerror.CustomServiceError(serviceerror.ValidationError, err.Error())
	}
	if err := utils.ValidateClientID(clientID); err != nil {
		return nil, serviceerror.CustomServiceError(serviceerror.ValidationError, err.Error())
	}
	if err := validator.ValidateConsentCreateRequest(req, clientID, orgID); err != nil {
		return nil, serviceerror.CustomServiceError(serviceerror.ValidationError, err.Error())
	}

	// Convert API request to internal format
	createReq, err := req.ToConsentCreateRequest()
	if err != nil {
		return nil, serviceerror.CustomServiceError(serviceerror.ValidationError, err.Error())
	}

	// Start transaction
	tx, err := s.db.BeginTx(ctx)
	if err != nil {
		return nil, serviceerror.CustomServiceError(serviceerror.DatabaseError, fmt.Sprintf("failed to begin transaction: %v", err))
	}
	defer tx.Rollback()

	// Generate IDs and timestamp
	consentID := utils.GenerateUUID()
	currentTime := utils.GetCurrentTimeMillis()

	// Create consent entity
	consent := &model.Consent{
		ConsentID:                  consentID,
		CreatedTime:                currentTime,
		UpdatedTime:                currentTime,
		ClientID:                   clientID,
		ConsentType:                createReq.ConsentType,
		CurrentStatus:              createReq.CurrentStatus,
		ConsentFrequency:           createReq.ConsentFrequency,
		ValidityTime:               createReq.ValidityTime,
		RecurringIndicator:         createReq.RecurringIndicator,
		DataAccessValidityDuration: createReq.DataAccessValidityDuration,
		OrgID:                      orgID,
	}

	// Store consent within transaction
	if err := s.store.CreateWithTx(ctx, tx, consent); err != nil {
		return nil, serviceerror.CustomServiceError(serviceerror.DatabaseError, fmt.Sprintf("failed to create consent: %v", err))
	}

	// Store attributes if provided
	if len(createReq.Attributes) > 0 {
		attributes := make([]model.ConsentAttribute, 0, len(createReq.Attributes))
		for key, value := range createReq.Attributes {
			attr := model.ConsentAttribute{
				ConsentID: consentID,
				AttKey:    key,
				AttValue:  value,
				OrgID:     orgID,
			}
			attributes = append(attributes, attr)
		}

		if err := s.store.CreateAttributesWithTx(ctx, tx, attributes); err != nil {
			return nil, serviceerror.CustomServiceError(serviceerror.DatabaseError, fmt.Sprintf("failed to create attributes: %v", err))
		}
	}

	// Create initial status audit
	auditID := utils.GenerateUUID()
	audit := &model.ConsentStatusAudit{
		StatusAuditID: auditID,
		ConsentID:     consentID,
		CurrentStatus: createReq.CurrentStatus,
		ActionTime:    currentTime,
		OrgID:         orgID,
	}

	if err := s.store.CreateStatusAuditWithTx(ctx, tx, audit); err != nil {
		return nil, serviceerror.CustomServiceError(serviceerror.DatabaseError, fmt.Sprintf("failed to create status audit: %v", err))
	}

	// Create authorization resources if provided
	for _, authReq := range req.Authorizations {
		authID := utils.GenerateUUID()

		// Marshal resources to JSON if present
		var resourcesJSON *string
		if authReq.Resources != nil {
			resourcesBytes, err := json.Marshal(authReq.Resources)
			if err != nil {
				return nil, serviceerror.CustomServiceError(serviceerror.ValidationError, fmt.Sprintf("failed to marshal resources: %v", err))
			}
			resourcesStr := string(resourcesBytes)
			resourcesJSON = &resourcesStr
		}

		// Convert to internal format
		var userIDPtr *string
		if authReq.UserID != "" {
			userIDPtr = &authReq.UserID
		}

		authResource := &authmodel.AuthResource{
			AuthID:      authID,
			ConsentID:   consentID,
			AuthType:    authReq.Type,
			UserID:      userIDPtr,
			AuthStatus:  authReq.Status,
			UpdatedTime: currentTime,
			Resources:   resourcesJSON,
			OrgID:       orgID,
		}

		if err := s.authResourceStore.CreateWithTx(ctx, tx, authResource); err != nil {
			return nil, serviceerror.CustomServiceError(serviceerror.DatabaseError, fmt.Sprintf("failed to create auth resource: %v", err))
		}
	}

	// Link consent purposes if provided (from ConsentPurpose field)
	for _, purposeItem := range req.ConsentPurpose {
		// ConsentPurpose items from API don't have the same structure as the mapping table
		// For now, we'll skip linking until we clarify the purpose structure
		// This can be added later when purpose linking is needed
		_ = purposeItem
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, serviceerror.CustomServiceError(serviceerror.DatabaseError, fmt.Sprintf("failed to commit transaction: %v", err))
	}

	// Build response
	response := &model.ConsentResponse{
		ConsentID:                  consent.ConsentID,
		ConsentPurpose:             createReq.ConsentPurpose,
		CreatedTime:                consent.CreatedTime,
		UpdatedTime:                consent.UpdatedTime,
		ClientID:                   consent.ClientID,
		ConsentType:                consent.ConsentType,
		CurrentStatus:              consent.CurrentStatus,
		ConsentFrequency:           consent.ConsentFrequency,
		ValidityTime:               consent.ValidityTime,
		RecurringIndicator:         consent.RecurringIndicator,
		DataAccessValidityDuration: consent.DataAccessValidityDuration,
		OrgID:                      consent.OrgID,
		Attributes:                 createReq.Attributes,
	}

	return response, nil
}

// GetConsent retrieves a consent by ID
func (s *consentService) GetConsent(ctx context.Context, consentID, orgID string) (*model.ConsentResponse, *serviceerror.ServiceError) {
	consent, err := s.store.GetByID(ctx, consentID, orgID)
	if err != nil {
		return nil, serviceerror.CustomServiceError(serviceerror.DatabaseError, err.Error())
	}
	if consent == nil {
		return nil, serviceerror.CustomServiceError(serviceerror.ResourceNotFoundError, fmt.Sprintf("Consent with ID '%s' not found", consentID))
	}

	// Load attributes
	_, err = s.store.GetAttributesByConsentID(ctx, consentID, orgID)
	if err != nil {
		return nil, serviceerror.CustomServiceError(serviceerror.DatabaseError, err.Error())
	}

	// Build response
	response := &model.ConsentResponse{
		ConsentID:                  consent.ConsentID,
		CreatedTime:                consent.CreatedTime,
		UpdatedTime:                consent.UpdatedTime,
		ClientID:                   consent.ClientID,
		ConsentType:                consent.ConsentType,
		CurrentStatus:              consent.CurrentStatus,
		ConsentFrequency:           consent.ConsentFrequency,
		ValidityTime:               consent.ValidityTime,
		RecurringIndicator:         consent.RecurringIndicator,
		DataAccessValidityDuration: consent.DataAccessValidityDuration,
		OrgID:                      consent.OrgID,
	}

	return response, nil
}

// ListConsents retrieves paginated list of consents
func (s *consentService) ListConsents(ctx context.Context, orgID string, limit, offset int) ([]model.ConsentResponse, int, *serviceerror.ServiceError) {
	if limit <= 0 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}

	consents, total, err := s.store.List(ctx, orgID, limit, offset)
	if err != nil {
		return nil, 0, serviceerror.CustomServiceError(serviceerror.DatabaseError, err.Error())
	}

	// Convert to responses
	responses := make([]model.ConsentResponse, 0, len(consents))
	for _, c := range consents {
		responses = append(responses, model.ConsentResponse{
			ConsentID:                  c.ConsentID,
			CreatedTime:                c.CreatedTime,
			UpdatedTime:                c.UpdatedTime,
			ClientID:                   c.ClientID,
			ConsentType:                c.ConsentType,
			CurrentStatus:              c.CurrentStatus,
			ConsentFrequency:           c.ConsentFrequency,
			ValidityTime:               c.ValidityTime,
			RecurringIndicator:         c.RecurringIndicator,
			DataAccessValidityDuration: c.DataAccessValidityDuration,
			OrgID:                      c.OrgID,
		})
	}

	return responses, total, nil
}

// UpdateConsent updates an existing consent
func (s *consentService) UpdateConsent(ctx context.Context, consentID string, req model.ConsentAPIUpdateRequest, orgID string) (*model.ConsentResponse, *serviceerror.ServiceError) {
	// Convert to internal format
	updateReq, convertErr := req.ToConsentUpdateRequest()
	if convertErr != nil {
		return nil, serviceerror.CustomServiceError(serviceerror.ValidationError, convertErr.Error())
	}

	// Check if consent exists
	existing, err := s.store.GetByID(ctx, consentID, orgID)
	if err != nil {
		return nil, serviceerror.CustomServiceError(serviceerror.DatabaseError, err.Error())
	}
	if existing == nil {
		return nil, serviceerror.CustomServiceError(serviceerror.ResourceNotFoundError, fmt.Sprintf("Consent with ID '%s' not found", consentID))
	}

	// Update consent fields
	consent := &model.Consent{
		ConsentID:                  consentID,
		UpdatedTime:                utils.GetCurrentTimeMillis(),
		ConsentType:                updateReq.ConsentType,
		ConsentFrequency:           updateReq.ConsentFrequency,
		ValidityTime:               updateReq.ValidityTime,
		RecurringIndicator:         updateReq.RecurringIndicator,
		DataAccessValidityDuration: updateReq.DataAccessValidityDuration,
		OrgID:                      orgID,
	}

	if updateErr := s.store.Update(ctx, consent); updateErr != nil {
		return nil, serviceerror.CustomServiceError(serviceerror.DatabaseError, updateErr.Error())
	}

	// Update attributes - delete old and create new
	if len(updateReq.Attributes) > 0 {
		// Delete existing attributes
		if delErr := s.store.DeleteAttributesByConsentID(ctx, consentID, orgID); delErr != nil {
			return nil, serviceerror.CustomServiceError(serviceerror.DatabaseError, delErr.Error())
		}

		// Create new attributes
		attributes := make([]model.ConsentAttribute, 0, len(updateReq.Attributes))
		for key, value := range updateReq.Attributes {
			attr := model.ConsentAttribute{
				ConsentID: consentID,
				AttKey:    key,
				AttValue:  value,
				OrgID:     orgID,
			}
			attributes = append(attributes, attr)
		}

		if createErr := s.store.CreateAttributes(ctx, attributes); createErr != nil {
			return nil, serviceerror.CustomServiceError(serviceerror.DatabaseError, createErr.Error())
		}
	}

	// Get updated consent
	updated, getErr := s.store.GetByID(ctx, consentID, orgID)
	if getErr != nil {
		return nil, serviceerror.CustomServiceError(serviceerror.DatabaseError, getErr.Error())
	}

	// Build response
	response := &model.ConsentResponse{
		ConsentID:                  updated.ConsentID,
		CreatedTime:                updated.CreatedTime,
		UpdatedTime:                updated.UpdatedTime,
		ClientID:                   updated.ClientID,
		ConsentType:                updated.ConsentType,
		CurrentStatus:              updated.CurrentStatus,
		ConsentFrequency:           updated.ConsentFrequency,
		ValidityTime:               updated.ValidityTime,
		RecurringIndicator:         updated.RecurringIndicator,
		DataAccessValidityDuration: updated.DataAccessValidityDuration,
		OrgID:                      updated.OrgID,
		Attributes:                 updateReq.Attributes,
	}

	return response, nil
}

// UpdateConsentStatus updates consent status and creates audit entry
func (s *consentService) UpdateConsentStatus(ctx context.Context, consentID, orgID string, req model.ConsentRevokeRequest) *serviceerror.ServiceError {
	// Validate action by
	if req.ActionBy == "" {
		return serviceerror.CustomServiceError(serviceerror.ValidationError, "ActionBy is required")
	}

	status := "REVOKED"

	// Check if consent exists
	existing, err := s.store.GetByID(ctx, consentID, orgID)
	if err != nil {
		return serviceerror.CustomServiceError(serviceerror.DatabaseError, err.Error())
	}
	if existing == nil {
		return serviceerror.CustomServiceError(serviceerror.ValidationError, fmt.Sprintf("Consent with ID '%s' not found", consentID))
	}

	currentTime := utils.GetCurrentTimeMillis()

	// Update status
	if err := s.store.UpdateStatus(ctx, consentID, orgID, status, currentTime); err != nil {
		return serviceerror.CustomServiceError(serviceerror.DatabaseError, err.Error())
	}

	// Create audit entry
	auditID := utils.GenerateUUID()
	reason := req.RevocationReason
	audit := &model.ConsentStatusAudit{
		StatusAuditID:  auditID,
		ConsentID:      consentID,
		CurrentStatus:  status,
		ActionTime:     currentTime,
		Reason:         &reason,
		ActionBy:       &req.ActionBy,
		PreviousStatus: &existing.CurrentStatus,
		OrgID:          orgID,
	}

	if err := s.store.CreateStatusAudit(ctx, audit); err != nil {
		return serviceerror.CustomServiceError(serviceerror.DatabaseError, err.Error())
	}

	return nil
}

// DeleteConsent deletes a consent
func (s *consentService) DeleteConsent(ctx context.Context, consentID, orgID string) *serviceerror.ServiceError {
	// Check if consent exists
	existing, err := s.store.GetByID(ctx, consentID, orgID)
	if err != nil {
		return serviceerror.CustomServiceError(serviceerror.DatabaseError, err.Error())
	}
	if existing == nil {
		return serviceerror.CustomServiceError(serviceerror.ValidationError, fmt.Sprintf("Consent with ID '%s' not found", consentID))
	}

	// Delete attributes first
	if attrErr := s.store.DeleteAttributesByConsentID(ctx, consentID, orgID); attrErr != nil {
		return serviceerror.CustomServiceError(serviceerror.DatabaseError, attrErr.Error())
	}

	// Delete consent (audit entries remain for history)
	if delErr := s.store.Delete(ctx, consentID, orgID); delErr != nil {
		return serviceerror.CustomServiceError(serviceerror.DatabaseError, delErr.Error())
	}

	return nil
}

// GetConsentsByClientID retrieves all consents for a client
func (s *consentService) GetConsentsByClientID(ctx context.Context, clientID, orgID string) ([]model.ConsentResponse, *serviceerror.ServiceError) {
	if clientID == "" {
		return nil, serviceerror.CustomServiceError(serviceerror.ValidationError, "Client ID is required")
	}

	consents, err := s.store.GetByClientID(ctx, clientID, orgID)
	if err != nil {
		return nil, serviceerror.CustomServiceError(serviceerror.DatabaseError, err.Error())
	}

	// Convert to responses
	responses := make([]model.ConsentResponse, 0, len(consents))
	for _, c := range consents {
		responses = append(responses, model.ConsentResponse{
			ConsentID:                  c.ConsentID,
			CreatedTime:                c.CreatedTime,
			UpdatedTime:                c.UpdatedTime,
			ClientID:                   c.ClientID,
			ConsentType:                c.ConsentType,
			CurrentStatus:              c.CurrentStatus,
			ConsentFrequency:           c.ConsentFrequency,
			ValidityTime:               c.ValidityTime,
			RecurringIndicator:         c.RecurringIndicator,
			DataAccessValidityDuration: c.DataAccessValidityDuration,
			OrgID:                      c.OrgID,
		})
	}

	return responses, nil
}
