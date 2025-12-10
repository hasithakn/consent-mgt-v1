package consentpurpose

import (
	"context"
	"fmt"

	"github.com/wso2/consent-management-api/internal/consentpurpose/model"
	"github.com/wso2/consent-management-api/internal/consentpurpose/validators"
	dbmodel "github.com/wso2/consent-management-api/internal/system/database/model"
	"github.com/wso2/consent-management-api/internal/system/error/serviceerror"
	"github.com/wso2/consent-management-api/internal/system/stores"
	"github.com/wso2/consent-management-api/internal/system/utils"
)

// ConsentPurposeService defines the exported service interface
type ConsentPurposeService interface {
	CreatePurpose(ctx context.Context, req model.CreateRequest, orgID string) (*model.ConsentPurpose, *serviceerror.ServiceError)
	GetPurpose(ctx context.Context, purposeID, orgID string) (*model.ConsentPurpose, *serviceerror.ServiceError)
	ListPurposes(ctx context.Context, orgID string, limit, offset int) ([]model.ConsentPurpose, int, *serviceerror.ServiceError)
	UpdatePurpose(ctx context.Context, purposeID string, req model.UpdateRequest, orgID string) (*model.ConsentPurpose, *serviceerror.ServiceError)
	DeletePurpose(ctx context.Context, purposeID, orgID string) *serviceerror.ServiceError
}

// consentPurposeService implements the ConsentPurposeService interface
type consentPurposeService struct {
	stores *stores.StoreRegistry
}

// newConsentPurposeService creates a new consent purpose service
func newConsentPurposeService(registry *stores.StoreRegistry) ConsentPurposeService {
	return &consentPurposeService{
		stores: registry,
	}
}

// CreatePurpose creates a new consent purpose
func (s *consentPurposeService) CreatePurpose(ctx context.Context, req model.CreateRequest, orgID string) (*model.ConsentPurpose, *serviceerror.ServiceError) {
	// Validate request
	if err := s.validateCreateRequest(req); err != nil {
		return nil, err
	}

	// Check if purpose name already exists
	store := s.stores.ConsentPurpose.(ConsentPurposeStore)
	exists, dbErr := store.CheckNameExists(ctx, req.Name, orgID)
	if dbErr != nil {
		return nil, serviceerror.CustomServiceError(serviceerror.DatabaseError, fmt.Sprintf("failed to check name existence: %v", dbErr))
	}
	if exists {
		return nil, serviceerror.CustomServiceError(serviceerror.ConflictError, fmt.Sprintf("purpose with name '%s' already exists", req.Name))
	}

	// Create purpose entity
	purposeID := utils.GenerateUUID()
	desc := req.Description
	purpose := &model.ConsentPurpose{
		ID:          purposeID,
		Name:        req.Name,
		Description: &desc,
		Type:        req.Type,
		OrgID:       orgID,
		Attributes:  req.Attributes,
	}

	// Prepare attributes if provided
	var attributes []model.ConsentPurposeAttribute
	if len(req.Attributes) > 0 {
		attributes = make([]model.ConsentPurposeAttribute, 0, len(req.Attributes))
		for key, value := range req.Attributes {
			attr := model.ConsentPurposeAttribute{
				ID:        utils.GenerateUUID(),
				PurposeID: purposeID,
				Key:       key,
				Value:     value,
				OrgID:     orgID,
			}
			attributes = append(attributes, attr)
		}
	}

	// Store purpose and attributes in a transaction
	queries := []func(tx dbmodel.TxInterface) error{
		func(tx dbmodel.TxInterface) error {
			return store.Create(tx, purpose)
		},
	}
	if len(attributes) > 0 {
		queries = append(queries, func(tx dbmodel.TxInterface) error {
			return store.CreateAttributes(tx, attributes)
		})
	}

	err := s.stores.ExecuteTransaction(queries)
	if err != nil {
		return nil, serviceerror.CustomServiceError(serviceerror.DatabaseError, fmt.Sprintf("failed to create purpose: %v", err))
	}

	return purpose, nil
}

// GetPurpose retrieves a consent purpose by ID
func (s *consentPurposeService) GetPurpose(ctx context.Context, purposeID, orgID string) (*model.ConsentPurpose, *serviceerror.ServiceError) {
	store := s.stores.ConsentPurpose.(ConsentPurposeStore)
	purpose, err := store.GetByID(ctx, purposeID, orgID)
	if err != nil {
		return nil, serviceerror.CustomServiceError(serviceerror.DatabaseError, fmt.Sprintf("failed to retrieve purpose: %v", err))
	}
	if purpose == nil {
		return nil, serviceerror.CustomServiceError(serviceerror.ResourceNotFoundError, fmt.Sprintf("purpose with ID '%s' not found", purposeID))
	}

	// Load attributes
	attributes, err := store.GetAttributesByPurposeID(ctx, purposeID, orgID)
	if err != nil {
		return nil, serviceerror.CustomServiceError(serviceerror.DatabaseError, fmt.Sprintf("failed to load attributes: %v", err))
	}

	// Convert attributes to map
	if purpose.Attributes == nil {
		purpose.Attributes = make(map[string]string)
	}
	for _, attr := range attributes {
		purpose.Attributes[attr.Key] = attr.Value
	}

	return purpose, nil
}

// ListPurposes retrieves paginated list of consent purposes
func (s *consentPurposeService) ListPurposes(ctx context.Context, orgID string, limit, offset int) ([]model.ConsentPurpose, int, *serviceerror.ServiceError) {
	if limit <= 0 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}

	store := s.stores.ConsentPurpose.(ConsentPurposeStore)
	purposes, total, err := store.List(ctx, orgID, limit, offset)
	if err != nil {
		return nil, 0, serviceerror.CustomServiceError(serviceerror.DatabaseError, fmt.Sprintf("failed to list purposes: %v", err))
	}

	// Load attributes for each purpose
	for i := range purposes {
		attributes, attrErr := store.GetAttributesByPurposeID(ctx, purposes[i].ID, orgID)
		if attrErr != nil {
			return nil, 0, serviceerror.CustomServiceError(serviceerror.DatabaseError, fmt.Sprintf("failed to load attributes: %v", attrErr))
		}

		if purposes[i].Attributes == nil {
			purposes[i].Attributes = make(map[string]string)
		}
		for _, attr := range attributes {
			purposes[i].Attributes[attr.Key] = attr.Value
		}
	}

	return purposes, total, nil
}

// UpdatePurpose updates an existing consent purpose
func (s *consentPurposeService) UpdatePurpose(ctx context.Context, purposeID string, req model.UpdateRequest, orgID string) (*model.ConsentPurpose, *serviceerror.ServiceError) {
	// Validate request
	if err := s.validateUpdateRequest(req); err != nil {
		return nil, err
	}

	// Check if purpose exists
	store := s.stores.ConsentPurpose.(ConsentPurposeStore)
	existing, err := store.GetByID(ctx, purposeID, orgID)
	if err != nil {
		return nil, serviceerror.CustomServiceError(serviceerror.DatabaseError, fmt.Sprintf("failed to retrieve purpose: %v", err))
	}
	if existing == nil {
		return nil, serviceerror.CustomServiceError(serviceerror.ResourceNotFoundError, fmt.Sprintf("purpose with ID '%s' not found", purposeID))
	}

	// Update purpose fields
	purpose := &model.ConsentPurpose{
		ID:          purposeID,
		Name:        req.Name,
		Description: req.Description,
		Type:        req.Type,
		OrgID:       orgID,
	}

	// Prepare attributes if provided
	var attributes []model.ConsentPurposeAttribute
	if len(req.Attributes) > 0 {
		attributes = make([]model.ConsentPurposeAttribute, 0, len(req.Attributes))
		for key, value := range req.Attributes {
			attr := model.ConsentPurposeAttribute{
				ID:        utils.GenerateUUID(),
				PurposeID: purposeID,
				Key:       key,
				Value:     value,
				OrgID:     orgID,
			}
			attributes = append(attributes, attr)
		}
		purpose.Attributes = req.Attributes
	}

	// Execute all updates in a transaction
	queries := []func(tx dbmodel.TxInterface) error{
		func(tx dbmodel.TxInterface) error {
			return store.Update(tx, purpose)
		},
	}
	if len(attributes) > 0 {
		queries = append(queries, func(tx dbmodel.TxInterface) error {
			return store.DeleteAttributesByPurposeID(tx, purposeID, orgID)
		})
		queries = append(queries, func(tx dbmodel.TxInterface) error {
			return store.CreateAttributes(tx, attributes)
		})
	}

	err = s.stores.ExecuteTransaction(queries)
	if err != nil {
		return nil, serviceerror.CustomServiceError(serviceerror.DatabaseError, fmt.Sprintf("failed to update purpose: %v", err))
	}

	return purpose, nil
}

// DeletePurpose deletes a consent purpose
func (s *consentPurposeService) DeletePurpose(ctx context.Context, purposeID, orgID string) *serviceerror.ServiceError {
	// Check if purpose exists
	store := s.stores.ConsentPurpose.(ConsentPurposeStore)
	existing, err := store.GetByID(ctx, purposeID, orgID)
	if err != nil {
		return serviceerror.CustomServiceError(serviceerror.DatabaseError, fmt.Sprintf("failed to retrieve purpose: %v", err))
	}
	if existing == nil {
		return serviceerror.CustomServiceError(serviceerror.ResourceNotFoundError, fmt.Sprintf("purpose with ID '%s' not found", purposeID))
	}

	// Delete attributes and purpose in a transaction
	err = s.stores.ExecuteTransaction([]func(tx dbmodel.TxInterface) error{
		func(tx dbmodel.TxInterface) error {
			return store.DeleteAttributesByPurposeID(tx, purposeID, orgID)
		},
		func(tx dbmodel.TxInterface) error {
			return store.Delete(tx, purposeID, orgID)
		},
	})
	if err != nil {
		return serviceerror.CustomServiceError(serviceerror.DatabaseError, fmt.Sprintf("failed to delete purpose: %v", err))
	}

	return nil
}

// validateCreateRequest validates create request
func (s *consentPurposeService) validateCreateRequest(req model.CreateRequest) *serviceerror.ServiceError {
	if req.Name == "" {
		return serviceerror.CustomServiceError(serviceerror.ValidationError, "purpose name is required")
	}
	if req.Type == "" {
		return serviceerror.CustomServiceError(serviceerror.ValidationError, "purpose type is required")
	}

	// Validate purpose type using validators
	handler, err := validators.GetHandler(req.Type)
	if err != nil {
		return serviceerror.CustomServiceError(serviceerror.ValidationError, fmt.Sprintf("invalid purpose type: %s", req.Type))
	}

	// Validate attributes using type handler
	if validationErr := handler.ValidateAttributes(req.Attributes); validationErr != nil {
		return serviceerror.CustomServiceError(serviceerror.ValidationError, fmt.Sprintf("attribute validation failed: %v", validationErr))
	}

	return nil
}

// validateUpdateRequest validates update request
func (s *consentPurposeService) validateUpdateRequest(req model.UpdateRequest) *serviceerror.ServiceError {
	if req.Name == "" {
		return serviceerror.CustomServiceError(serviceerror.ValidationError, "purpose name is required")
	}
	if req.Type == "" {
		return serviceerror.CustomServiceError(serviceerror.ValidationError, "purpose type is required")
	}

	// Validate purpose type using validators
	handler, err := validators.GetHandler(req.Type)
	if err != nil {
		return serviceerror.CustomServiceError(serviceerror.ValidationError, fmt.Sprintf("invalid purpose type: %s", req.Type))
	}

	// Validate attributes using type handler
	if validationErr := handler.ValidateAttributes(req.Attributes); validationErr != nil {
		return serviceerror.CustomServiceError(serviceerror.ValidationError, fmt.Sprintf("attribute validation failed: %v", validationErr))
	}

	return nil
}
