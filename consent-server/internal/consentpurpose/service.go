package consentpurpose

import (
	"context"
	"fmt"

	"github.com/wso2/consent-management-api/internal/consentpurpose/model"
	"github.com/wso2/consent-management-api/internal/consentpurpose/validators"
	dbmodel "github.com/wso2/consent-management-api/internal/system/database/model"
	"github.com/wso2/consent-management-api/internal/system/error/serviceerror"
	"github.com/wso2/consent-management-api/internal/system/log"
	"github.com/wso2/consent-management-api/internal/system/stores"
	"github.com/wso2/consent-management-api/internal/system/utils"
)

// ConsentPurposeService defines the exported service interface
type ConsentPurposeService interface {
	CreatePurpose(ctx context.Context, req model.CreateRequest, orgID string) (*model.ConsentPurpose, *serviceerror.ServiceError)
	CreatePurposesInBatch(ctx context.Context, requests []model.CreateRequest, orgID string) ([]model.ConsentPurpose, *serviceerror.ServiceError)
	GetPurpose(ctx context.Context, purposeID, orgID string) (*model.ConsentPurpose, *serviceerror.ServiceError)
	ListPurposes(ctx context.Context, orgID string, limit, offset int, name string) ([]model.ConsentPurpose, int, *serviceerror.ServiceError)
	UpdatePurpose(ctx context.Context, purposeID string, req model.UpdateRequest, orgID string) (*model.ConsentPurpose, *serviceerror.ServiceError)
	DeletePurpose(ctx context.Context, purposeID, orgID string) *serviceerror.ServiceError
	ValidatePurposeNames(ctx context.Context, orgID string, purposeNames []string) ([]string, *serviceerror.ServiceError)
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
	logger := log.GetLogger().WithContext(ctx)

	logger.Info("Creating consent purpose",
		log.String("name", req.Name),
		log.String("type", req.Type),
		log.String("org_id", orgID))

	// Validate request
	if err := s.validateCreateRequest(req); err != nil {
		logger.Warn("Consent purpose create request validation failed", log.String("error", err.Error))
		return nil, err
	}

	// Check if purpose name already exists
	store := s.stores.ConsentPurpose.(ConsentPurposeStore)
	exists, dbErr := store.CheckNameExists(ctx, req.Name, orgID)
	if dbErr != nil {
		logger.Error("Failed to check purpose name existence", log.Error(dbErr), log.String("name", req.Name))
		return nil, serviceerror.CustomServiceError(serviceerror.DatabaseError, fmt.Sprintf("failed to check name existence: %v", dbErr))
	}
	if exists {
		logger.Warn("Purpose name already exists", log.String("name", req.Name))
		return nil, serviceerror.CustomServiceError(serviceerror.ConflictError, fmt.Sprintf("purpose with name '%s' already exists", req.Name))
	}

	// Create purpose entity
	purposeID := utils.GenerateUUID()
	logger.Debug("Generated purpose ID", log.String("purpose_id", purposeID))
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
		logger.Debug("Adding purpose attributes", log.Int("attribute_count", len(req.Attributes)))
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

	logger.Debug("Executing transaction", log.Int("operation_count", len(queries)))
	err := s.stores.ExecuteTransaction(queries)
	if err != nil {
		logger.Error("Failed to create purpose in transaction", log.Error(err), log.String("purpose_id", purposeID))
		return nil, serviceerror.CustomServiceError(serviceerror.DatabaseError, fmt.Sprintf("failed to create purpose: %v", err))
	}

	logger.Info("Consent purpose created successfully",
		log.String("purpose_id", purposeID),
		log.String("name", req.Name),
		log.Int("attributes", len(attributes)))

	return purpose, nil
}

// CreatePurposesInBatch creates multiple consent purposes in a single transaction
// Either all purposes are created or none (atomic operation)
func (s *consentPurposeService) CreatePurposesInBatch(ctx context.Context, requests []model.CreateRequest, orgID string) ([]model.ConsentPurpose, *serviceerror.ServiceError) {
	// Validate inputs
	if len(requests) == 0 {
		return nil, serviceerror.CustomServiceError(serviceerror.ValidationError, "at least one purpose must be provided")
	}

	store := s.stores.ConsentPurpose.(ConsentPurposeStore)

	// Pre-validate all requests and check for duplicate names within the batch
	namesSeen := make(map[string]bool)
	for i, req := range requests {
		// Validate request
		if valErr := s.validateCreateRequest(req); valErr != nil {
			// Return error with index information
			return nil, serviceerror.CustomServiceError(serviceerror.ValidationError, fmt.Sprintf("invalid request at index %d: %v", i, valErr))
		}

		// Check for duplicate names within the batch
		if namesSeen[req.Name] {
			return nil, serviceerror.CustomServiceError(serviceerror.ValidationError, fmt.Sprintf("duplicate purpose name '%s' in request batch at index %d", req.Name, i))
		}
		namesSeen[req.Name] = true

		// Check if purpose name already exists in database
		exists, dbErr := store.CheckNameExists(ctx, req.Name, orgID)
		if dbErr != nil {
			return nil, serviceerror.CustomServiceError(serviceerror.DatabaseError, fmt.Sprintf("failed to validate purpose name at index %d: %v", i, dbErr))
		}
		if exists {
			return nil, serviceerror.CustomServiceError(serviceerror.ConflictError, fmt.Sprintf("purpose name '%s' already exists for this organization (at index %d)", req.Name, i))
		}
	}

	// Prepare transaction operations
	var queries []func(tx dbmodel.TxInterface) error
	createdPurposes := make([]model.ConsentPurpose, 0, len(requests))

	// Create all purposes within the transaction
	for _, req := range requests {
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

		// Add purpose creation to transaction
		purposeCopy := *purpose // Create a copy for the closure
		queries = append(queries, func(tx dbmodel.TxInterface) error {
			return store.Create(tx, &purposeCopy)
		})

		// Add attributes if provided
		if len(req.Attributes) > 0 {
			attributes := make([]model.ConsentPurposeAttribute, 0, len(req.Attributes))
			for key, value := range req.Attributes {
				attr := model.ConsentPurposeAttribute{
					PurposeID: purposeID,
					Key:       key,
					Value:     value,
					OrgID:     orgID,
				}
				attributes = append(attributes, attr)
			}

			// Capture attributes for this iteration
			attrsCopy := attributes
			queries = append(queries, func(tx dbmodel.TxInterface) error {
				return store.CreateAttributes(tx, attrsCopy)
			})
		}

		createdPurposes = append(createdPurposes, *purpose)
	}

	// Execute all operations in a single transaction
	if err := s.stores.ExecuteTransaction(queries); err != nil {
		return nil, serviceerror.CustomServiceError(serviceerror.DatabaseError, fmt.Sprintf("failed to create purposes in batch: %v", err))
	}

	return createdPurposes, nil
}

// GetPurpose retrieves a consent purpose by ID
func (s *consentPurposeService) GetPurpose(ctx context.Context, purposeID, orgID string) (*model.ConsentPurpose, *serviceerror.ServiceError) {
	logger := log.GetLogger().WithContext(ctx)
	logger.Debug("Retrieving consent purpose",
		log.String("purpose_id", purposeID),
		log.String("org_id", orgID),
	)

	store := s.stores.ConsentPurpose.(ConsentPurposeStore)
	purpose, err := store.GetByID(ctx, purposeID, orgID)
	if err != nil {
		logger.Error("Failed to retrieve purpose",
			log.Error(err),
			log.String("purpose_id", purposeID),
		)
		return nil, serviceerror.CustomServiceError(serviceerror.DatabaseError, fmt.Sprintf("failed to retrieve purpose: %v", err))
	}
	if purpose == nil {
		logger.Warn("Purpose not found", log.String("purpose_id", purposeID))
		return nil, serviceerror.CustomServiceError(serviceerror.ResourceNotFoundError, fmt.Sprintf("purpose with ID '%s' not found", purposeID))
	}

	// Load attributes
	attributes, err := store.GetAttributesByPurposeID(ctx, purposeID, orgID)
	if err != nil {
		logger.Error("Failed to load purpose attributes",
			log.Error(err),
			log.String("purpose_id", purposeID),
		)
		return nil, serviceerror.CustomServiceError(serviceerror.DatabaseError, fmt.Sprintf("failed to load attributes: %v", err))
	}

	// Convert attributes to map
	if purpose.Attributes == nil {
		purpose.Attributes = make(map[string]string)
	}
	for _, attr := range attributes {
		purpose.Attributes[attr.Key] = attr.Value
	}

	logger.Debug("Purpose retrieved successfully",
		log.String("purpose_id", purposeID),
		log.String("name", purpose.Name),
		log.Int("attributes_count", len(attributes)),
	)
	return purpose, nil
}

// ListPurposes retrieves paginated list of consent purposes with optional name filter
func (s *consentPurposeService) ListPurposes(ctx context.Context, orgID string, limit, offset int, name string) ([]model.ConsentPurpose, int, *serviceerror.ServiceError) {
	logger := log.GetLogger().WithContext(ctx)
	logger.Debug("Listing consent purposes",
		log.String("org_id", orgID),
		log.Int("limit", limit),
		log.Int("offset", offset),
		log.String("name_filter", name),
	)

	if limit <= 0 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	store := s.stores.ConsentPurpose.(ConsentPurposeStore)
	purposes, total, err := store.List(ctx, orgID, limit, offset, name)
	if err != nil {
		logger.Error("Failed to list purposes",
			log.Error(err),
			log.String("org_id", orgID),
		)
		return nil, 0, serviceerror.CustomServiceError(serviceerror.DatabaseError, fmt.Sprintf("failed to list purposes: %v", err))
	}

	// Load attributes for each purpose
	for i := range purposes {
		attributes, attrErr := store.GetAttributesByPurposeID(ctx, purposes[i].ID, orgID)
		if attrErr != nil {
			logger.Error("Failed to load attributes for purpose",
				log.Error(attrErr),
				log.String("purpose_id", purposes[i].ID),
			)
			return nil, 0, serviceerror.CustomServiceError(serviceerror.DatabaseError, fmt.Sprintf("failed to load attributes: %v", attrErr))
		}

		if purposes[i].Attributes == nil {
			purposes[i].Attributes = make(map[string]string)
		}
		for _, attr := range attributes {
			purposes[i].Attributes[attr.Key] = attr.Value
		}
	}

	logger.Debug("Purposes listed successfully",
		log.Int("count", len(purposes)),
		log.Int("total", total),
	)
	return purposes, total, nil
}

// UpdatePurpose updates an existing consent purpose
func (s *consentPurposeService) UpdatePurpose(ctx context.Context, purposeID string, req model.UpdateRequest, orgID string) (*model.ConsentPurpose, *serviceerror.ServiceError) {
	logger := log.GetLogger().WithContext(ctx)
	logger.Info("Updating consent purpose",
		log.String("purpose_id", purposeID),
		log.String("name", req.Name),
		log.String("org_id", orgID),
	)

	// Validate request
	if err := s.validateUpdateRequest(req); err != nil {
		logger.Warn("Update purpose request validation failed", log.String("error", err.Error))
		return nil, err
	}

	// Check if purpose exists
	store := s.stores.ConsentPurpose.(ConsentPurposeStore)
	existing, err := store.GetByID(ctx, purposeID, orgID)
	if err != nil {
		logger.Error("Failed to retrieve existing purpose",
			log.Error(err),
			log.String("purpose_id", purposeID),
		)
		return nil, serviceerror.CustomServiceError(serviceerror.DatabaseError, fmt.Sprintf("failed to retrieve purpose: %v", err))
	}
	if existing == nil {
		logger.Warn("Purpose not found for update", log.String("purpose_id", purposeID))
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

	logger.Debug("Executing transaction for purpose update",
		log.Int("attributes_count", len(attributes)),
	)
	err = s.stores.ExecuteTransaction(queries)
	if err != nil {
		logger.Error("Transaction failed for purpose update",
			log.Error(err),
			log.String("purpose_id", purposeID),
		)
		return nil, serviceerror.CustomServiceError(serviceerror.DatabaseError, fmt.Sprintf("failed to update purpose: %v", err))
	}

	logger.Info("Purpose updated successfully",
		log.String("purpose_id", purposeID),
		log.String("name", purpose.Name),
	)
	return purpose, nil
}

// DeletePurpose deletes a consent purpose
func (s *consentPurposeService) DeletePurpose(ctx context.Context, purposeID, orgID string) *serviceerror.ServiceError {
	logger := log.GetLogger().WithContext(ctx)
	logger.Info("Deleting consent purpose",
		log.String("purpose_id", purposeID),
		log.String("org_id", orgID),
	)

	// Check if purpose exists
	store := s.stores.ConsentPurpose.(ConsentPurposeStore)
	existing, err := store.GetByID(ctx, purposeID, orgID)
	if err != nil {
		logger.Error("Failed to retrieve purpose for deletion",
			log.Error(err),
			log.String("purpose_id", purposeID),
		)
		return serviceerror.CustomServiceError(serviceerror.DatabaseError, fmt.Sprintf("failed to retrieve purpose: %v", err))
	}
	if existing == nil {
		logger.Warn("Purpose not found for deletion", log.String("purpose_id", purposeID))
		return serviceerror.CustomServiceError(serviceerror.ResourceNotFoundError, fmt.Sprintf("purpose with ID '%s' not found", purposeID))
	}

	// Delete attributes and purpose in a transaction
	logger.Debug("Executing transaction for purpose deletion")
	err = s.stores.ExecuteTransaction([]func(tx dbmodel.TxInterface) error{
		func(tx dbmodel.TxInterface) error {
			return store.DeleteAttributesByPurposeID(tx, purposeID, orgID)
		},
		func(tx dbmodel.TxInterface) error {
			return store.Delete(tx, purposeID, orgID)
		},
	})
	if err != nil {
		logger.Error("Transaction failed for purpose deletion",
			log.Error(err),
			log.String("purpose_id", purposeID),
		)
		return serviceerror.CustomServiceError(serviceerror.DatabaseError, fmt.Sprintf("failed to delete purpose: %v", err))
	}

	logger.Info("Purpose deleted successfully",
		log.String("purpose_id", purposeID),
		log.String("name", existing.Name),
	)
	return nil
}

// ValidatePurposeNames validates a list of purpose names and returns only the valid ones
func (s *consentPurposeService) ValidatePurposeNames(ctx context.Context, orgID string, purposeNames []string) ([]string, *serviceerror.ServiceError) {
	logger := log.GetLogger().WithContext(ctx)
	logger.Debug("Validating purpose names",
		log.String("org_id", orgID),
		log.Int("name_count", len(purposeNames)),
	)

	// Validate input
	if len(purposeNames) == 0 {
		logger.Warn("No purpose names provided for validation")
		return nil, serviceerror.CustomServiceError(serviceerror.ValidationError, "at least one purpose name must be provided")
	}

	store := s.stores.ConsentPurpose.(ConsentPurposeStore)

	// Get purposes that exist
	purposeIDMap, err := store.GetIDsByNames(ctx, purposeNames, orgID)
	if err != nil {
		logger.Error("Failed to validate purpose names",
			log.Error(err),
			log.String("org_id", orgID),
		)
		return nil, serviceerror.CustomServiceError(serviceerror.DatabaseError, fmt.Sprintf("failed to validate purpose names: %v", err))
	}

	// Extract valid names from the map
	validNames := make([]string, 0, len(purposeIDMap))
	for name := range purposeIDMap {
		validNames = append(validNames, name)
	}

	// Return error if no valid purposes found
	if len(validNames) == 0 {
		logger.Warn("No valid purposes found")
		return nil, serviceerror.CustomServiceError(serviceerror.ValidationError, "no valid purposes found")
	}

	logger.Debug("Purpose names validated",
		log.Int("valid_count", len(validNames)),
		log.Int("requested_count", len(purposeNames)),
	)
	return validNames, nil
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
