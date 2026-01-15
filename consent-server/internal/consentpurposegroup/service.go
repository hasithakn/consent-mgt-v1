package consentpurposegroup

import (
	"context"
	"fmt"
	"time"

	"github.com/wso2/consent-management-api/internal/consentpurposegroup/model"
	dbmodel "github.com/wso2/consent-management-api/internal/system/database/model"
	"github.com/wso2/consent-management-api/internal/system/error/serviceerror"
	"github.com/wso2/consent-management-api/internal/system/log"
	"github.com/wso2/consent-management-api/internal/system/stores"
	"github.com/wso2/consent-management-api/internal/system/utils"
)

// Service defines the exported service interface
type ConsentPurposeGroupService interface {
	CreatePurposeGroup(ctx context.Context, req model.CreateRequest, orgID, clientID string) (*model.PurposeGroup, *serviceerror.ServiceError)
	GetPurposeGroup(ctx context.Context, groupID, orgID string) (*model.PurposeGroup, *serviceerror.ServiceError)
	ListPurposeGroups(ctx context.Context, orgID, name string, clientIDs []string, purposeNames []string, offset, limit int) ([]model.PurposeGroup, int, *serviceerror.ServiceError)
	UpdatePurposeGroup(ctx context.Context, groupID string, req model.UpdateRequest, orgID, clientID string) (*model.PurposeGroup, *serviceerror.ServiceError)
	DeletePurposeGroup(ctx context.Context, groupID, orgID string) *serviceerror.ServiceError
}

// service implements the Service interface
type consentPurposeGroupService struct {
	stores *stores.StoreRegistry
}

// NewService creates a new purpose group service
func NewConsentPurposeGroupService(registry *stores.StoreRegistry) ConsentPurposeGroupService {
	return &consentPurposeGroupService{
		stores: registry,
	}
}

// CreatePurposeGroup creates a new purpose group
func (s *consentPurposeGroupService) CreatePurposeGroup(ctx context.Context, req model.CreateRequest, orgID, clientID string) (*model.PurposeGroup, *serviceerror.ServiceError) {
	logger := log.GetLogger().WithContext(ctx)

	logger.Info("Creating purpose group",
		log.String("name", req.Name),
		log.String("client_id", clientID),
		log.String("org_id", orgID))

	// Validate request
	if err := s.validateCreateRequest(req); err != nil {
		logger.Warn("Purpose group create request validation failed", log.String("error", err.Error()))
		return nil, err
	}

	// Check if group name already exists for this client
	exists, dbErr := s.stores.ConsentPurposeGroup.CheckGroupNameExists(ctx, req.Name, clientID, orgID, nil)
	if dbErr != nil {
		logger.Error("Failed to check group name existence", log.Error(dbErr), log.String("name", req.Name))
		return nil, serviceerror.CustomServiceError(serviceerror.DatabaseError, fmt.Sprintf("failed to check name existence: %v", dbErr))
	}
	if exists {
		logger.Warn("Group name already exists for this client", log.String("name", req.Name), log.String("client_id", clientID))
		return nil, serviceerror.CustomServiceError(serviceerror.ConflictError, fmt.Sprintf("purpose group with name '%s' already exists for this client", req.Name))
	}

	// Validate that all purpose names exist
	purposeNameToID, err := s.validatePurposeNamesExist(ctx, req.Purposes, orgID)
	if err != nil {
		return nil, err
	}

	// Check for duplicate purpose names within the request
	if duplicateErr := s.checkDuplicatePurposeNames(req.Purposes); duplicateErr != nil {
		return nil, duplicateErr
	}

	// Create purpose group entity
	groupID := utils.GenerateUUID()
	now := time.Now().Unix()
	desc := &req.Description
	if req.Description == "" {
		desc = nil
	}

	group := &model.PurposeGroup{
		ID:          groupID,
		Name:        req.Name,
		Description: desc,
		ClientID:    clientID,
		CreatedTime: now,
		UpdatedTime: now,
		OrgID:       orgID,
	}

	// Execute transaction for group creation and purpose linking
	queries := []func(tx dbmodel.TxInterface) error{
		func(tx dbmodel.TxInterface) error {
			return s.stores.ConsentPurposeGroup.CreateGroup(tx, group)
		},
	}

	// Add purpose linking operations
	for _, purpose := range req.Purposes {
		purposeID := purposeNameToID[purpose.PurposeName]
		isMandatory := purpose.IsMandatory
		purposeName := purpose.PurposeName

		queries = append(queries, func(tx dbmodel.TxInterface) error {
			return s.stores.ConsentPurposeGroup.LinkPurposeToGroup(tx, groupID, purposeID, orgID, isMandatory)
		})

		group.Purposes = append(group.Purposes, model.PurposeGroupPurpose{
			PurposeID:   purposeID,
			PurposeName: purposeName,
			IsMandatory: isMandatory,
		})
	}

	if err := s.stores.ExecuteTransaction(queries); err != nil {
		logger.Error("Failed to create purpose group", log.Error(err))
		return nil, serviceerror.CustomServiceError(serviceerror.DatabaseError, fmt.Sprintf("failed to create group: %v", err))
	}

	logger.Info("Purpose group created successfully", log.String("group_id", groupID))
	return group, nil
}

// GetPurposeGroup retrieves a purpose group by ID
func (s *consentPurposeGroupService) GetPurposeGroup(ctx context.Context, groupID, orgID string) (*model.PurposeGroup, *serviceerror.ServiceError) {
	logger := log.GetLogger().WithContext(ctx)

	logger.Debug("Retrieving purpose group", log.String("group_id", groupID), log.String("org_id", orgID))

	group, err := s.stores.ConsentPurposeGroup.GetGroupByID(ctx, groupID, orgID)
	if err != nil {
		logger.Error("Failed to retrieve purpose group", log.Error(err), log.String("group_id", groupID))
		return nil, serviceerror.CustomServiceError(serviceerror.ResourceNotFoundError, "purpose group not found")
	}

	return group, nil
}

// ListPurposeGroups retrieves a list of purpose groups with optional filters
func (s *consentPurposeGroupService) ListPurposeGroups(ctx context.Context, orgID, name string, clientIDs []string, purposeNames []string, offset, limit int) ([]model.PurposeGroup, int, *serviceerror.ServiceError) {
	logger := log.GetLogger().WithContext(ctx)

	logger.Debug("Listing purpose groups",
		log.String("org_id", orgID),
		log.String("name", name),
		log.Int("offset", offset),
		log.Int("limit", limit))

	groups, total, err := s.stores.ConsentPurposeGroup.ListGroups(ctx, orgID, name, clientIDs, purposeNames, offset, limit)
	if err != nil {
		logger.Error("Failed to list purpose groups", log.Error(err))
		return nil, 0, serviceerror.CustomServiceError(serviceerror.DatabaseError, fmt.Sprintf("failed to list groups: %v", err))
	}

	return groups, total, nil
}

// UpdatePurposeGroup updates an existing purpose group
func (s *consentPurposeGroupService) UpdatePurposeGroup(ctx context.Context, groupID string, req model.UpdateRequest, orgID, clientID string) (*model.PurposeGroup, *serviceerror.ServiceError) {
	logger := log.GetLogger().WithContext(ctx)

	logger.Info("Updating purpose group",
		log.String("group_id", groupID),
		log.String("name", req.Name),
		log.String("org_id", orgID))

	// Validate request
	if err := s.validateUpdateRequest(req); err != nil {
		logger.Warn("Purpose group update request validation failed", log.String("error", err.Error()))
		return nil, err
	}

	// Check if group exists
	existingGroup, err := s.stores.ConsentPurposeGroup.GetGroupByID(ctx, groupID, orgID)
	if err != nil {
		logger.Error("Failed to retrieve purpose group", log.Error(err), log.String("group_id", groupID))
		return nil, serviceerror.CustomServiceError(serviceerror.ResourceNotFoundError, "purpose group not found")
	}

	// Verify client ownership
	if existingGroup.ClientID != clientID {
		logger.Warn("Client does not own this purpose group",
			log.String("group_client_id", existingGroup.ClientID),
			log.String("request_client_id", clientID))
		return nil, serviceerror.CustomServiceError(serviceerror.ConflictError, "you do not have permission to update this purpose group")
	}

	// Check if group is being used in any consents
	inUse, checkErr := s.stores.Consent.CheckGroupUsedInConsents(ctx, groupID, orgID)
	if checkErr != nil {
		logger.Error("Failed to check if group is in use", log.Error(checkErr))
		return nil, serviceerror.CustomServiceError(serviceerror.DatabaseError, "failed to check group usage")
	}
	if inUse {
		logger.Warn("Cannot update purpose group that is in use by consents", log.String("group_id", groupID))
		return nil, serviceerror.CustomServiceError(serviceerror.ConflictError, "cannot update purpose group that is currently used in consents")
	}

	// Check if new name conflicts with another group (excluding current group)
	exists, dbErr := s.stores.ConsentPurposeGroup.CheckGroupNameExists(ctx, req.Name, clientID, orgID, &groupID)
	if dbErr != nil {
		logger.Error("Failed to check group name existence", log.Error(dbErr))
		return nil, serviceerror.CustomServiceError(serviceerror.DatabaseError, "failed to check name existence")
	}
	if exists {
		logger.Warn("Group name already exists for this client", log.String("name", req.Name))
		return nil, serviceerror.CustomServiceError(serviceerror.ConflictError, fmt.Sprintf("purpose group with name '%s' already exists for this client", req.Name))
	}

	// Validate purpose names exist
	purposeNameToID, validationErr := s.validatePurposeNamesExist(ctx, req.Purposes, orgID)
	if validationErr != nil {
		return nil, validationErr
	}

	// Check for duplicate purpose names
	if duplicateErr := s.checkDuplicatePurposeNames(req.Purposes); duplicateErr != nil {
		return nil, duplicateErr
	}

	// Update group
	now := time.Now().Unix()
	desc := &req.Description
	if req.Description == "" {
		desc = nil
	}

	group := &model.PurposeGroup{
		ID:          groupID,
		Name:        req.Name,
		Description: desc,
		ClientID:    clientID,
		CreatedTime: existingGroup.CreatedTime,
		UpdatedTime: now,
		OrgID:       orgID,
	}

	// Execute transaction for group update
	queries := []func(tx dbmodel.TxInterface) error{
		func(tx dbmodel.TxInterface) error {
			return s.stores.ConsentPurposeGroup.UpdateGroup(tx, group)
		},
		func(tx dbmodel.TxInterface) error {
			return s.stores.ConsentPurposeGroup.DeleteGroupPurposes(tx, groupID, orgID)
		},
	}

	// Add new purpose mappings
	for _, purpose := range req.Purposes {
		purposeID := purposeNameToID[purpose.PurposeName]
		isMandatory := purpose.IsMandatory
		purposeName := purpose.PurposeName

		queries = append(queries, func(tx dbmodel.TxInterface) error {
			return s.stores.ConsentPurposeGroup.LinkPurposeToGroup(tx, groupID, purposeID, orgID, isMandatory)
		})

		group.Purposes = append(group.Purposes, model.PurposeGroupPurpose{
			PurposeID:   purposeID,
			PurposeName: purposeName,
			IsMandatory: isMandatory,
		})
	}

	if err := s.stores.ExecuteTransaction(queries); err != nil {
		logger.Error("Failed to update purpose group", log.Error(err))
		return nil, serviceerror.CustomServiceError(serviceerror.DatabaseError, fmt.Sprintf("failed to update group: %v", err))
	}

	logger.Info("Purpose group updated successfully", log.String("group_id", groupID))
	return group, nil
}

// DeletePurposeGroup deletes a purpose group
func (s *consentPurposeGroupService) DeletePurposeGroup(ctx context.Context, groupID, orgID string) *serviceerror.ServiceError {
	logger := log.GetLogger().WithContext(ctx)

	logger.Info("Deleting purpose group", log.String("group_id", groupID), log.String("org_id", orgID))

	// Check if group exists
	_, err := s.stores.ConsentPurposeGroup.GetGroupByID(ctx, groupID, orgID)
	if err != nil {
		logger.Error("Failed to retrieve purpose group", log.Error(err))
		return serviceerror.CustomServiceError(serviceerror.ResourceNotFoundError, "purpose group not found")
	}

	// Check if group is being used in any consents
	inUse, checkErr := s.stores.Consent.CheckGroupUsedInConsents(ctx, groupID, orgID)
	if checkErr != nil {
		logger.Error("Failed to check if group is in use", log.Error(checkErr))
		return serviceerror.CustomServiceError(serviceerror.DatabaseError, "failed to check group usage")
	}
	if inUse {
		logger.Warn("Cannot delete purpose group that is in use by consents", log.String("group_id", groupID))
		return serviceerror.CustomServiceError(serviceerror.ConflictError, "cannot delete purpose group that is currently used in consents")
	}

	// Execute transaction for deletion
	queries := []func(tx dbmodel.TxInterface) error{
		func(tx dbmodel.TxInterface) error {
			return s.stores.ConsentPurposeGroup.DeleteGroup(tx, groupID, orgID)
		},
	}

	if err := s.stores.ExecuteTransaction(queries); err != nil {
		logger.Error("Failed to delete purpose group", log.Error(err))
		return serviceerror.CustomServiceError(serviceerror.DatabaseError, fmt.Sprintf("failed to delete group: %v", err))
	}

	logger.Info("Purpose group deleted successfully", log.String("group_id", groupID))
	return nil
}

// validateCreateRequest validates the create request
func (s *consentPurposeGroupService) validateCreateRequest(req model.CreateRequest) *serviceerror.ServiceError {
	if req.Name == "" {
		return serviceerror.CustomServiceError(serviceerror.InvalidRequestError, "name is required")
	}
	if len(req.Name) > 255 {
		return serviceerror.CustomServiceError(serviceerror.InvalidRequestError, "name must not exceed 255 characters")
	}
	if len(req.Description) > 1024 {
		return serviceerror.CustomServiceError(serviceerror.InvalidRequestError, "description must not exceed 1024 characters")
	}
	if len(req.Purposes) == 0 {
		return serviceerror.CustomServiceError(serviceerror.InvalidRequestError, "at least one purpose is required")
	}
	for _, purpose := range req.Purposes {
		if purpose.PurposeName == "" {
			return serviceerror.CustomServiceError(serviceerror.InvalidRequestError, "purpose name is required")
		}
	}
	return nil
}

// validateUpdateRequest validates the update request
func (s *consentPurposeGroupService) validateUpdateRequest(req model.UpdateRequest) *serviceerror.ServiceError {
	if req.Name == "" {
		return serviceerror.CustomServiceError(serviceerror.InvalidRequestError, "name is required")
	}
	if len(req.Name) > 255 {
		return serviceerror.CustomServiceError(serviceerror.InvalidRequestError, "name must not exceed 255 characters")
	}
	if len(req.Description) > 1024 {
		return serviceerror.CustomServiceError(serviceerror.InvalidRequestError, "description must not exceed 1024 characters")
	}
	if len(req.Purposes) == 0 {
		return serviceerror.CustomServiceError(serviceerror.InvalidRequestError, "at least one purpose is required")
	}
	for _, purpose := range req.Purposes {
		if purpose.PurposeName == "" {
			return serviceerror.CustomServiceError(serviceerror.InvalidRequestError, "purpose name is required")
		}
	}
	return nil
}

// validatePurposeNamesExist validates that all purpose names exist and returns a map of name -> ID
func (s *consentPurposeGroupService) validatePurposeNamesExist(ctx context.Context, purposes []model.PurposeInput, orgID string) (map[string]string, *serviceerror.ServiceError) {
	purposeNames := make([]string, len(purposes))
	for i, p := range purposes {
		purposeNames[i] = p.PurposeName
	}

	purposeNameToID, err := s.stores.ConsentPurposeGroup.ValidatePurposeNames(ctx, purposeNames, orgID)
	if err != nil {
		return nil, serviceerror.CustomServiceError(serviceerror.DatabaseError, fmt.Sprintf("failed to validate purpose names: %v", err))
	}

	// Check that all purposes were found
	for _, purpose := range purposes {
		if _, found := purposeNameToID[purpose.PurposeName]; !found {
			return nil, serviceerror.CustomServiceError(serviceerror.InvalidRequestError, fmt.Sprintf("purpose '%s' does not exist", purpose.PurposeName))
		}
	}

	return purposeNameToID, nil
}

// checkDuplicatePurposeNames checks for duplicate purpose names within the request
func (s *consentPurposeGroupService) checkDuplicatePurposeNames(purposes []model.PurposeInput) *serviceerror.ServiceError {
	seen := make(map[string]bool)
	for _, purpose := range purposes {
		if seen[purpose.PurposeName] {
			return serviceerror.CustomServiceError(serviceerror.InvalidRequestError, fmt.Sprintf("duplicate purpose '%s' found in request", purpose.PurposeName))
		}
		seen[purpose.PurposeName] = true
	}
	return nil
}
