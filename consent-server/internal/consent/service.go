package consent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/wso2/consent-management-api/internal/authresource"
	authmodel "github.com/wso2/consent-management-api/internal/authresource/model"
	"github.com/wso2/consent-management-api/internal/consent/model"
	"github.com/wso2/consent-management-api/internal/consent/validator"
	"github.com/wso2/consent-management-api/internal/consentpurpose"
	purposemodel "github.com/wso2/consent-management-api/internal/consentpurpose/model"
	dbmodel "github.com/wso2/consent-management-api/internal/system/database/model"
	"github.com/wso2/consent-management-api/internal/system/error/serviceerror"
	"github.com/wso2/consent-management-api/internal/system/stores"
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

// consentService implements the ConsentService interface
type consentService struct {
	stores *stores.StoreRegistry
}

// newConsentService creates a new consent service
func newConsentService(registry *stores.StoreRegistry) ConsentService {
	return &consentService{
		stores: registry,
	}
}

// CreateConsent creates a new consent with all related entities in a single transaction
func (consentService *consentService) CreateConsent(ctx context.Context, req model.ConsentAPIRequest, clientID, orgID string) (*model.ConsentResponse, *serviceerror.ServiceError) {
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

	// Derive consent status from authorization states
	consentStatus := validator.EvaluateConsentStatus(createReq.AuthResources)

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
		CurrentStatus:              consentStatus,
		ConsentFrequency:           createReq.ConsentFrequency,
		ValidityTime:               createReq.ValidityTime,
		RecurringIndicator:         createReq.RecurringIndicator,
		DataAccessValidityDuration: createReq.DataAccessValidityDuration,
		OrgID:                      orgID,
	}

	// Get stores from registry
	consentStore := consentService.stores.Consent.(ConsentStore)
	authResourceStore := consentService.stores.AuthResource.(authresource.AuthResourceStore)

	// Build list of transactional operations
	queries := []func(tx dbmodel.TxInterface) error{
		// Create consent
		func(tx dbmodel.TxInterface) error {
			return consentStore.Create(tx, consent)
		},
	}

	// Add attributes if provided
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
		queries = append(queries, func(tx dbmodel.TxInterface) error {
			return consentStore.CreateAttributes(tx, attributes)
		})
	}

	// Add status audit
	auditID := utils.GenerateUUID()
	audit := &model.ConsentStatusAudit{
		StatusAuditID: auditID,
		ConsentID:     consentID,
		CurrentStatus: consent.CurrentStatus,
		ActionTime:    currentTime,
		OrgID:         orgID,
	}
	queries = append(queries, func(tx dbmodel.TxInterface) error {
		return consentStore.CreateStatusAudit(tx, audit)
	})

	// Add authorization resources if provided
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

		queries = append(queries, func(tx dbmodel.TxInterface) error {
			return authResourceStore.Create(tx, authResource)
		})
	}

	// Link consent purposes if provided
	if len(createReq.ConsentPurpose) > 0 {
		purposeStore := consentService.stores.ConsentPurpose.(consentpurpose.ConsentPurposeStore)

		// Extract purpose names
		purposeNames := make([]string, len(createReq.ConsentPurpose))
		for i, p := range createReq.ConsentPurpose {
			purposeNames[i] = p.Name
		}

		// Get purpose IDs by names
		purposeIDMap, err := purposeStore.GetIDsByNames(ctx, purposeNames, orgID)
		if err != nil {
			return nil, serviceerror.CustomServiceError(serviceerror.DatabaseError, fmt.Sprintf("failed to get purpose IDs: %v", err))
		}

		// Verify all purposes were found
		if len(purposeIDMap) != len(purposeNames) {
			missingPurposes := []string{}
			for _, name := range purposeNames {
				if _, found := purposeIDMap[name]; !found {
					missingPurposes = append(missingPurposes, name)
				}
			}
			return nil, serviceerror.CustomServiceError(serviceerror.ValidationError, fmt.Sprintf("purposes not found: %v", missingPurposes))
		}

		// Link each purpose
		for _, purposeItem := range createReq.ConsentPurpose {
			purposeID := purposeIDMap[purposeItem.Name]

			// Marshal value to JSON string if present
			var valueJSON *string
			if purposeItem.Value != nil {
				valueBytes, err := json.Marshal(purposeItem.Value)
				if err != nil {
					return nil, serviceerror.CustomServiceError(serviceerror.ValidationError, fmt.Sprintf("failed to marshal purpose value: %v", err))
				}
				valueStr := string(valueBytes)
				valueJSON = &valueStr
			}

			// Get boolean values with defaults
			isUserApproved := false
			if purposeItem.IsUserApproved != nil {
				isUserApproved = *purposeItem.IsUserApproved
			}

			isMandatory := true
			if purposeItem.IsMandatory != nil {
				isMandatory = *purposeItem.IsMandatory
			}

			// Add to transaction queries
			queries = append(queries, func(tx dbmodel.TxInterface) error {
				return purposeStore.LinkPurposeToConsent(tx, consentID, purposeID, orgID, valueJSON, isUserApproved, isMandatory)
			})
		}
	}

	// Execute all operations in a single transaction
	if err := consentService.stores.ExecuteTransaction(queries); err != nil {
		return nil, serviceerror.CustomServiceError(serviceerror.DatabaseError, fmt.Sprintf("failed to create consent: %v", err))
	}

	// Retrieve related data after creation
	authResources, _ := authResourceStore.GetByConsentID(ctx, consentID, orgID)
	purposeMappings, _ := consentService.stores.ConsentPurpose.(consentpurpose.ConsentPurposeStore).GetMappingsByConsentID(ctx, consentID, orgID)
	attributes, _ := consentService.stores.Consent.(ConsentStore).GetAttributesByConsentID(ctx, consentID, orgID)

	// Convert attributes slice to map[string]string
	attributesMap := make(map[string]string)
	for _, a := range attributes {
		attributesMap[a.AttKey] = a.AttValue
	}

	// Build complete response
	response := buildConsentResponse(consent, attributesMap, authResources, purposeMappings)

	return response, nil
}

// GetConsent retrieves a consent by ID
func (consentService *consentService) GetConsent(ctx context.Context, consentID, orgID string) (*model.ConsentResponse, *serviceerror.ServiceError) {

	consentStore := consentService.stores.Consent.(ConsentStore)
	consent, err := consentStore.GetByID(ctx, consentID, orgID)
	if err != nil {
		return nil, serviceerror.CustomServiceError(serviceerror.DatabaseError, err.Error())
	}
	if consent == nil {
		return nil, serviceerror.CustomServiceError(serviceerror.ResourceNotFoundError, fmt.Sprintf("Consent with ID '%s' not found", consentID))
	}

	// Load attributes
	_, err = consentStore.GetAttributesByConsentID(ctx, consentID, orgID)
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
func (consentService *consentService) ListConsents(ctx context.Context, orgID string, limit, offset int) ([]model.ConsentResponse, int, *serviceerror.ServiceError) {
	if limit <= 0 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}

	store := consentService.stores.Consent.(ConsentStore)
	consents, total, err := store.List(ctx, orgID, limit, offset)
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
func (consentService *consentService) UpdateConsent(ctx context.Context, consentID string, req model.ConsentAPIUpdateRequest, orgID string) (*model.ConsentResponse, *serviceerror.ServiceError) {
	// Convert to internal format
	updateReq, convertErr := req.ToConsentUpdateRequest()
	if convertErr != nil {
		return nil, serviceerror.CustomServiceError(serviceerror.ValidationError, convertErr.Error())
	}

	// Check if consent exists
	store := consentService.stores.Consent.(ConsentStore)
	existing, err := store.GetByID(ctx, consentID, orgID)
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

	// Build transactional operations
	queries := []func(tx dbmodel.TxInterface) error{
		func(tx dbmodel.TxInterface) error {
			return store.Update(tx, consent)
		},
	}

	// Update attributes - delete old and create new
	if len(updateReq.Attributes) > 0 {
		// Delete existing attributes
		queries = append(queries, func(tx dbmodel.TxInterface) error {
			return store.DeleteAttributesByConsentID(tx, consentID, orgID)
		})

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

		queries = append(queries, func(tx dbmodel.TxInterface) error {
			return store.CreateAttributes(tx, attributes)
		})
	}

	// Execute transaction
	if err := consentService.stores.ExecuteTransaction(queries); err != nil {
		return nil, serviceerror.CustomServiceError(serviceerror.DatabaseError, err.Error())
	}

	// Get updated consent
	updated, getErr := store.GetByID(ctx, consentID, orgID)
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
func (consentService *consentService) UpdateConsentStatus(ctx context.Context, consentID, orgID string, req model.ConsentRevokeRequest) *serviceerror.ServiceError {
	// Validate action by
	if req.ActionBy == "" {
		return serviceerror.CustomServiceError(serviceerror.ValidationError, "ActionBy is required")
	}

	status := "REVOKED"

	// Check if consent exists
	store := consentService.stores.Consent.(ConsentStore)
	existing, err := store.GetByID(ctx, consentID, orgID)
	if err != nil {
		return serviceerror.CustomServiceError(serviceerror.DatabaseError, err.Error())
	}
	if existing == nil {
		return serviceerror.CustomServiceError(serviceerror.ValidationError, fmt.Sprintf("Consent with ID '%s' not found", consentID))
	}

	currentTime := utils.GetCurrentTimeMillis()

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

	// Execute transaction
	err = consentService.stores.ExecuteTransaction([]func(tx dbmodel.TxInterface) error{
		func(tx dbmodel.TxInterface) error {
			return store.UpdateStatus(tx, consentID, orgID, status, currentTime)
		},
		func(tx dbmodel.TxInterface) error {
			return store.CreateStatusAudit(tx, audit)
		},
	})
	if err != nil {
		return serviceerror.CustomServiceError(serviceerror.DatabaseError, err.Error())
	}

	return nil
}

// DeleteConsent deletes a consent
func (consentService *consentService) DeleteConsent(ctx context.Context, consentID, orgID string) *serviceerror.ServiceError {
	// Check if consent exists
	store := consentService.stores.Consent.(ConsentStore)
	existing, err := store.GetByID(ctx, consentID, orgID)
	if err != nil {
		return serviceerror.CustomServiceError(serviceerror.DatabaseError, err.Error())
	}
	if existing == nil {
		return serviceerror.CustomServiceError(serviceerror.ValidationError, fmt.Sprintf("Consent with ID '%s' not found", consentID))
	}

	// Delete attributes and consent in transaction
	err = consentService.stores.ExecuteTransaction([]func(tx dbmodel.TxInterface) error{
		func(tx dbmodel.TxInterface) error {
			return store.DeleteAttributesByConsentID(tx, consentID, orgID)
		},
		func(tx dbmodel.TxInterface) error {
			return store.Delete(tx, consentID, orgID)
		},
	})
	if err != nil {
		return serviceerror.CustomServiceError(serviceerror.DatabaseError, err.Error())
	}

	return nil
}

// GetConsentsByClientID retrieves all consents for a client
func (consentService *consentService) GetConsentsByClientID(ctx context.Context, clientID, orgID string) ([]model.ConsentResponse, *serviceerror.ServiceError) {
	if clientID == "" {
		return nil, serviceerror.CustomServiceError(serviceerror.ValidationError, "Client ID is required")
	}

	store := consentService.stores.Consent.(ConsentStore)
	consents, err := store.GetByClientID(ctx, clientID, orgID)
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

// buildConsentResponse constructs a complete ConsentResponse with related data
func buildConsentResponse(
	consent *model.Consent,
	attributes map[string]string,
	authResources []authmodel.AuthResource,
	purposeMappings []purposemodel.ConsentPurposeMapping,
) *model.ConsentResponse {
	// Convert purposeMappings to ConsentPurposeItem
	purposes := make([]model.ConsentPurposeItem, len(purposeMappings))
	for i, mapping := range purposeMappings {
		purposes[i] = model.ConsentPurposeItem{
			Name:           mapping.Name,
			Value:          mapping.Value,
			IsUserApproved: &mapping.IsUserApproved,
			IsMandatory:    &mapping.IsMandatory,
		}
	}

	// AuthResource is already a type alias for ConsentAuthResource - use directly
	authResourcesResp := authResources

	return &model.ConsentResponse{
		ConsentID:                  consent.ConsentID,
		ConsentPurpose:             purposes,
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
		Attributes:                 attributes,
		AuthResources:              authResourcesResp,
	}
}
