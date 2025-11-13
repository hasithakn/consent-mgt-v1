package service

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"

	"github.com/wso2/consent-management-api/internal/dao"
	"github.com/wso2/consent-management-api/internal/models"
	"github.com/wso2/consent-management-api/internal/purpose_type_handlers"
	"github.com/wso2/consent-management-api/internal/utils"
)

// ConsentPurposeService handles business logic for consent purposes
type ConsentPurposeService struct {
	purposeDAO          *dao.ConsentPurposeDAO
	purposeAttributeDAO *dao.ConsentPurposeAttributeDAO
	consentDAO          *dao.ConsentDAO
	db                  *sqlx.DB
	logger              *logrus.Logger
}

// NewConsentPurposeService creates a new ConsentPurposeService
func NewConsentPurposeService(
	purposeDAO *dao.ConsentPurposeDAO,
	purposeAttributeDAO *dao.ConsentPurposeAttributeDAO,
	consentDAO *dao.ConsentDAO,
	db *sqlx.DB,
	logger *logrus.Logger,
) *ConsentPurposeService {
	return &ConsentPurposeService{
		purposeDAO:          purposeDAO,
		purposeAttributeDAO: purposeAttributeDAO,
		consentDAO:          consentDAO,
		db:                  db,
		logger:              logger,
	}
}

// ConsentPurposeCreateRequest represents request to create a consent purpose
type ConsentPurposeCreateRequest struct {
	Name        string
	Description *string
	Type        string
	Attributes  map[string]string // Purpose-specific attributes
}

// ConsentPurposeUpdateRequest represents request to update a consent purpose
// All fields are required - no partial updates allowed
type ConsentPurposeUpdateRequest struct {
	Name        string
	Description *string
	Type        string
	Attributes  map[string]string // Purpose-specific attributes
}

// validateAttributesForType validates attributes based on the purpose type using the appropriate handler
func (s *ConsentPurposeService) validateAttributesForType(purposeType string, attributes map[string]string) []purpose_type_handlers.ValidationError {
	handler, err := purpose_type_handlers.GetHandler(purposeType)
	if err != nil {
		// If handler not found, return as validation error
		return []purpose_type_handlers.ValidationError{
			{
				Field:   "type",
				Message: fmt.Sprintf("unknown purpose type: %s", purposeType),
			},
		}
	}

	// Validate attributes using the handler
	return handler.ValidateAttributes(attributes)
}

// processAttributesForType processes (normalizes) attributes based on the purpose type
func (s *ConsentPurposeService) processAttributesForType(purposeType string, attributes map[string]string) map[string]string {
	handler, err := purpose_type_handlers.GetHandler(purposeType)
	if err != nil {
		// If handler not found, return as-is
		return attributes
	}

	// Process attributes using the handler
	return handler.ProcessAttributes(attributes)
}

// CreatePurpose creates a new consent purpose
func (s *ConsentPurposeService) CreatePurpose(ctx context.Context, orgID string, req *ConsentPurposeCreateRequest) (*models.ConsentPurposeResponse, error) {
	// Validate inputs
	if err := s.validateCreateRequest(req); err != nil {
		return nil, err
	}

	if err := utils.ValidateOrgID(orgID); err != nil {
		return nil, err
	}

	// Validate purpose type
	if err := models.ValidatePurposeType(req.Type); err != nil {
		return nil, fmt.Errorf("invalid purpose type: %w", err)
	}

	// Validate attributes based on purpose type
	validationErrors := s.validateAttributesForType(req.Type, req.Attributes)
	if len(validationErrors) > 0 {
		var errMsgs []string
		for _, ve := range validationErrors {
			errMsgs = append(errMsgs, fmt.Sprintf("[%s: %s]", ve.Field, ve.Message))
		}
		return nil, fmt.Errorf("attribute validation failed for type %q: %v", req.Type, errMsgs)
	}

	// Process attributes (normalize, set defaults, etc.)
	processedAttrs := s.processAttributesForType(req.Type, req.Attributes)

	// Check if purpose name already exists for this organization
	exists, err := s.purposeDAO.ExistsByName(ctx, req.Name, orgID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to check purpose name existence")
		return nil, fmt.Errorf("failed to validate purpose name: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("purpose name '%s' already exists for this organization", req.Name)
	}

	// Generate unique ID
	purposeID := utils.GenerateID()

	// Create purpose object
	purpose := &models.ConsentPurpose{
		ID:          purposeID,
		Name:        req.Name,
		Description: req.Description,
		Type:        req.Type,
		OrgID:       orgID,
	}

	// Begin transaction
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		s.logger.WithError(err).Error("Failed to begin transaction")
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Save purpose to database using transaction
	if err := s.purposeDAO.CreateWithTx(ctx, tx, purpose); err != nil {
		s.logger.WithError(err).Error("Failed to create consent purpose")
		return nil, fmt.Errorf("failed to create consent purpose: %w", err)
	}

	// Save the processed attributes within transaction
	if len(processedAttrs) > 0 {
		if err := s.purposeAttributeDAO.SaveAttributesWithTx(ctx, tx, purposeID, orgID, processedAttrs); err != nil {
			s.logger.WithError(err).Error("Failed to save consent purpose attributes")
			return nil, fmt.Errorf("failed to save purpose attributes: %w", err)
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		s.logger.WithError(err).Error("Failed to commit transaction")
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"purpose_id":       purposeID,
		"org_id":           orgID,
		"attributes_count": len(processedAttrs),
	}).Info("Consent purpose created successfully")

	purpose.Attributes = processedAttrs
	response := s.buildPurposeResponse(purpose)
	return response, nil
}

// CreatePurposesInBatch creates multiple consent purposes in a single transaction
// Either all purposes are created or none (atomic operation)
func (s *ConsentPurposeService) CreatePurposesInBatch(ctx context.Context, orgID string, requests []*ConsentPurposeCreateRequest) ([]models.ConsentPurposeResponse, error) {
	// Validate inputs
	if len(requests) == 0 {
		return nil, fmt.Errorf("at least one purpose must be provided")
	}

	if err := utils.ValidateOrgID(orgID); err != nil {
		return nil, err
	}

	// Start transaction
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		s.logger.WithError(err).Error("Failed to start transaction")
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback() // Rollback if not committed

	createdPurposes := make([]models.ConsentPurposeResponse, 0, len(requests))

	// Pre-validate all requests and check for duplicate names within the batch
	namesSeen := make(map[string]bool)
	for i, req := range requests {
		// Validate request
		if err := s.validateCreateRequest(req); err != nil {
			return nil, fmt.Errorf("invalid request at index %d: %w", i, err)
		}

		// Validate purpose type
		if err := models.ValidatePurposeType(req.Type); err != nil {
			return nil, fmt.Errorf("invalid purpose type at index %d: %w", i, err)
		}

		// Validate attributes based on purpose type
		validationErrors := s.validateAttributesForType(req.Type, req.Attributes)
		if len(validationErrors) > 0 {
			var errMsgs []string
			for _, ve := range validationErrors {
				errMsgs = append(errMsgs, fmt.Sprintf("[%s: %s]", ve.Field, ve.Message))
			}
			return nil, fmt.Errorf("attribute validation failed for type %q at index %d: %v", req.Type, i, errMsgs)
		}

		// Check for duplicate names within the batch
		if namesSeen[req.Name] {
			return nil, fmt.Errorf("duplicate purpose name '%s' in request batch at index %d", req.Name, i)
		}
		namesSeen[req.Name] = true

		// Check if purpose name already exists in database (using transaction)
		exists, err := s.purposeDAO.ExistsByNameWithTx(ctx, tx, req.Name, orgID)
		if err != nil {
			s.logger.WithError(err).Error("Failed to check purpose name existence")
			return nil, fmt.Errorf("failed to validate purpose name at index %d: %w", i, err)
		}
		if exists {
			return nil, fmt.Errorf("purpose name '%s' already exists for this organization (at index %d)", req.Name, i)
		}
	}

	// Create all purposes within the transaction
	for i, req := range requests {
		// Process attributes (normalize, set defaults, etc.)
		processedAttrs := s.processAttributesForType(req.Type, req.Attributes)

		// Generate purpose ID
		purposeID := utils.GenerateID()

		// Create purpose object
		purpose := &models.ConsentPurpose{
			ID:          purposeID,
			Name:        req.Name,
			Description: req.Description,
			Type:        req.Type,
			OrgID:       orgID,
		}

		// Save to database using transaction
		if err := s.purposeDAO.CreateWithTx(ctx, tx, purpose); err != nil {
			s.logger.WithError(err).Error("Failed to create consent purpose")
			return nil, fmt.Errorf("failed to create consent purpose at index %d: %w", i, err)
		}

		// Save attributes within transaction
		if len(processedAttrs) > 0 {
			if err := s.purposeAttributeDAO.SaveAttributesWithTx(ctx, tx, purposeID, orgID, processedAttrs); err != nil {
				s.logger.WithError(err).Error("Failed to save consent purpose attributes")
				return nil, fmt.Errorf("failed to save purpose attributes at index %d: %w", i, err)
			}
		}

		// Log processed attributes for audit trail
		s.logger.WithFields(logrus.Fields{
			"purpose_id":       purposeID,
			"org_id":           orgID,
			"attributes_count": len(processedAttrs),
		}).Debug("Purpose attributes processed")

		// Add to response list
		purpose.Attributes = processedAttrs
		createdPurposes = append(createdPurposes, *s.buildPurposeResponse(purpose))

		s.logger.WithFields(logrus.Fields{
			"purpose_id": purposeID,
			"org_id":     orgID,
		}).Info("Consent purpose created successfully within transaction")
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		s.logger.WithError(err).Error("Failed to commit transaction")
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"count":  len(createdPurposes),
		"org_id": orgID,
	}).Info("Batch consent purpose creation completed successfully")

	return createdPurposes, nil
}

// GetPurpose retrieves a consent purpose by ID
func (s *ConsentPurposeService) GetPurpose(ctx context.Context, purposeID, orgID string) (*models.ConsentPurposeResponse, error) {
	// Validate inputs
	if purposeID == "" {
		return nil, fmt.Errorf("purpose ID is required")
	}

	if err := utils.ValidateOrgID(orgID); err != nil {
		return nil, err
	}

	// Retrieve purpose
	purpose, err := s.purposeDAO.GetByID(ctx, purposeID, orgID)
	if err != nil {
		return nil, fmt.Errorf("purpose not found: %w", err)
	}

	// Retrieve attributes
	attributes, err := s.purposeAttributeDAO.GetAttributes(ctx, purposeID, orgID)
	if err != nil {
		s.logger.WithError(err).Warn("Failed to retrieve purpose attributes")
		// Don't fail, just warn
	} else {
		purpose.Attributes = attributes
	}

	return s.buildPurposeResponse(purpose), nil
}

// ListPurposes retrieves all consent purposes for an organization
func (s *ConsentPurposeService) ListPurposes(ctx context.Context, orgID string, limit, offset int) (*models.ConsentPurposeListResponse, error) {
	// Validate inputs
	if err := utils.ValidateOrgID(orgID); err != nil {
		return nil, err
	}

	// Set default pagination values
	if limit <= 0 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	// Retrieve purposes
	purposes, total, err := s.purposeDAO.List(ctx, orgID, limit, offset)
	if err != nil {
		s.logger.WithError(err).Error("Failed to list consent purposes")
		return nil, fmt.Errorf("failed to list consent purposes: %w", err)
	}

	// Build response (attributes will be loaded on demand if needed)
	purposeResponses := make([]models.ConsentPurposeResponse, 0, len(purposes))
	for i := range purposes {
		// Optionally load attributes for each purpose
		// Note: For large lists, you might want to batch fetch attributes
		attributes, err := s.purposeAttributeDAO.GetAttributes(ctx, purposes[i].ID, orgID)
		if err != nil {
			s.logger.WithError(err).Warnf("Failed to retrieve attributes for purpose %s", purposes[i].ID)
		} else {
			purposes[i].Attributes = attributes
		}
		purposeResponses = append(purposeResponses, *s.buildPurposeResponse(&purposes[i]))
	}

	return &models.ConsentPurposeListResponse{
		Purposes: purposeResponses,
		Total:    total,
	}, nil
}

// GetPurposesByConsentID retrieves all purposes linked to a specific consent
func (s *ConsentPurposeService) GetPurposesByConsentID(ctx context.Context, consentID, orgID string) ([]models.ConsentPurposeResponse, error) {
	if consentID == "" {
		return nil, fmt.Errorf("consent ID is required")
	}
	if err := utils.ValidateOrgID(orgID); err != nil {
		return nil, err
	}

	// Retrieve purposes from DAO
	purposes, err := s.purposeDAO.GetByConsentID(ctx, consentID, orgID)
	if err != nil {
		s.logger.WithError(err).WithField("consent_id", consentID).Error("Failed to get purposes for consent")
		return nil, fmt.Errorf("failed to get purposes for consent: %w", err)
	}

	// Build response with attributes
	purposeResponses := make([]models.ConsentPurposeResponse, 0, len(purposes))
	for i := range purposes {
		// Load attributes for each purpose
		attributes, err := s.purposeAttributeDAO.GetAttributes(ctx, purposes[i].ID, orgID)
		if err != nil {
			s.logger.WithError(err).Warnf("Failed to retrieve attributes for purpose %s", purposes[i].ID)
		} else {
			purposes[i].Attributes = attributes
		}
		purposeResponses = append(purposeResponses, *s.buildPurposeResponse(&purposes[i]))
	}

	return purposeResponses, nil
}

// UpdatePurpose updates an existing consent purpose
func (s *ConsentPurposeService) UpdatePurpose(ctx context.Context, purposeID, orgID string, req *ConsentPurposeUpdateRequest) (*models.ConsentPurposeResponse, error) {
	// Validate inputs
	if purposeID == "" {
		return nil, fmt.Errorf("purpose ID is required")
	}

	if err := utils.ValidateOrgID(orgID); err != nil {
		return nil, err
	}

	if err := s.validateUpdateRequest(req); err != nil {
		return nil, err
	}

	// Validate purpose type
	if err := models.ValidatePurposeType(req.Type); err != nil {
		return nil, fmt.Errorf("invalid purpose type: %w", err)
	}

	// Validate attributes based on type
	validationErrors := s.validateAttributesForType(req.Type, req.Attributes)
	if len(validationErrors) > 0 {
		var errMsgs []string
		for _, ve := range validationErrors {
			errMsgs = append(errMsgs, fmt.Sprintf("[%s: %s]", ve.Field, ve.Message))
		}
		return nil, fmt.Errorf("attribute validation failed for type %q: %v", req.Type, errMsgs)
	}

	// Process attributes (normalize, set defaults, etc.)
	processedAttrs := s.processAttributesForType(req.Type, req.Attributes)

	// Check if purpose exists
	existingPurpose, err := s.purposeDAO.GetByID(ctx, purposeID, orgID)
	if err != nil {
		return nil, fmt.Errorf("purpose not found: %w", err)
	}

	// Check if name is being changed and if the new name already exists
	if req.Name != existingPurpose.Name {
		exists, err := s.purposeDAO.ExistsByName(ctx, req.Name, orgID)
		if err != nil {
			s.logger.WithError(err).Error("Failed to check purpose name existence")
			return nil, fmt.Errorf("failed to validate purpose name: %w", err)
		}
		if exists {
			return nil, fmt.Errorf("purpose name '%s' already exists for this organization", req.Name)
		}
	}

	// Begin transaction
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		s.logger.WithError(err).Error("Failed to begin transaction")
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Update all fields (no partial updates)
	existingPurpose.Name = req.Name
	existingPurpose.Description = req.Description
	existingPurpose.Type = req.Type
	// Attributes are stored separately (handled by DAO)

	// Update purpose in database within transaction
	if err := s.purposeDAO.UpdateWithTx(ctx, tx, existingPurpose); err != nil {
		s.logger.WithError(err).Error("Failed to update consent purpose")
		return nil, fmt.Errorf("failed to update consent purpose: %w", err)
	}

	// Update attributes within transaction
	if err := s.purposeAttributeDAO.SaveAttributesWithTx(ctx, tx, purposeID, orgID, processedAttrs); err != nil {
		s.logger.WithError(err).Error("Failed to save consent purpose attributes")
		return nil, fmt.Errorf("failed to save purpose attributes: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		s.logger.WithError(err).Error("Failed to commit transaction")
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"purpose_id":       purposeID,
		"org_id":           orgID,
		"attributes_count": len(processedAttrs),
	}).Info("Consent purpose updated successfully")

	existingPurpose.Attributes = processedAttrs
	return s.buildPurposeResponse(existingPurpose), nil
}

// DeletePurpose deletes a consent purpose
func (s *ConsentPurposeService) DeletePurpose(ctx context.Context, purposeID, orgID string) error {
	// Validate inputs
	if purposeID == "" {
		return fmt.Errorf("purpose ID is required")
	}

	if err := utils.ValidateOrgID(orgID); err != nil {
		return err
	}

	// Check if purpose exists
	_, err := s.purposeDAO.GetByID(ctx, purposeID, orgID)
	if err != nil {
		return fmt.Errorf("purpose not found: %w", err)
	}

	// Begin transaction
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		s.logger.WithError(err).Error("Failed to begin transaction")
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete attributes first
	if err := s.purposeAttributeDAO.DeleteAttributesWithTx(ctx, tx, purposeID, orgID); err != nil {
		s.logger.WithError(err).Error("Failed to delete consent purpose attributes")
		return fmt.Errorf("failed to delete purpose attributes: %w", err)
	}

	// Delete purpose within transaction
	if err := s.purposeDAO.DeleteWithTx(ctx, tx, purposeID, orgID); err != nil {
		s.logger.WithError(err).Error("Failed to delete consent purpose")
		return fmt.Errorf("failed to delete consent purpose: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		s.logger.WithError(err).Error("Failed to commit transaction")
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"purpose_id": purposeID,
		"org_id":     orgID,
	}).Info("Consent purpose deleted successfully")

	return nil
}

// validateCreateRequest validates the create request
func (s *ConsentPurposeService) validateCreateRequest(req *ConsentPurposeCreateRequest) error {
	if req.Name == "" {
		return fmt.Errorf("purpose name is required")
	}

	if len(req.Name) > 255 {
		return fmt.Errorf("purpose name too long: maximum 255 characters")
	}

	if req.Description != nil && len(*req.Description) > 1024 {
		return fmt.Errorf("purpose description too long: maximum 1024 characters")
	}

	// Type is required
	if req.Type == "" {
		return fmt.Errorf("purpose type is required")
	}

	return nil
}

// validateUpdateRequest validates the update request
// All fields are required for update - no partial updates
func (s *ConsentPurposeService) validateUpdateRequest(req *ConsentPurposeUpdateRequest) error {
	// Name is required
	if req.Name == "" {
		return fmt.Errorf("purpose name is required")
	}
	if len(req.Name) > 255 {
		return fmt.Errorf("purpose name too long: maximum 255 characters")
	}

	// Description can be nil (optional), but if provided must be within limits
	if req.Description != nil && len(*req.Description) > 1024 {
		return fmt.Errorf("purpose description too long: maximum 1024 characters")
	}

	// Type is required
	if req.Type == "" {
		return fmt.Errorf("purpose type is required")
	}

	return nil
}

// ExistsByName checks if a purpose with the given name exists in the organization
func (s *ConsentPurposeService) ExistsByName(ctx context.Context, name, orgID string) (bool, error) {
	if name == "" {
		return false, fmt.Errorf("purpose name is required")
	}
	if orgID == "" {
		return false, fmt.Errorf("organization ID is required")
	}

	exists, err := s.purposeDAO.ExistsByName(ctx, name, orgID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to check purpose name existence")
		return false, fmt.Errorf("failed to check purpose existence: %w", err)
	}

	return exists, nil
}

// ValidatePurposeNames validates a list of purpose names against the database
// Returns only the names that exist in the database for the given organization
func (s *ConsentPurposeService) ValidatePurposeNames(ctx context.Context, orgID string, names []string) ([]string, error) {
	if orgID == "" {
		return nil, fmt.Errorf("organization ID is required")
	}

	if len(names) == 0 {
		return nil, fmt.Errorf("at least one purpose name must be provided")
	}

	// Validate each name format
	for _, name := range names {
		if name == "" {
			return nil, fmt.Errorf("purpose name cannot be empty")
		}
		if len(name) > 100 {
			return nil, fmt.Errorf("purpose name too long (max 100 characters): %s", name)
		}
	}

	// Get valid purpose names from database
	validNames, err := s.purposeDAO.ValidatePurposeNames(ctx, names, orgID)
	if err != nil {
		s.logger.WithError(err).WithField("org_id", orgID).Error("Failed to validate purpose names")
		return nil, fmt.Errorf("failed to validate purpose names: %w", err)
	}

	// Return error if no valid purposes found
	if len(validNames) == 0 {
		return nil, fmt.Errorf("no valid purposes found")
	}

	s.logger.WithFields(logrus.Fields{
		"org_id":    orgID,
		"requested": len(names),
		"valid":     len(validNames),
	}).Info("Validated purpose names")

	return validNames, nil
}

// buildPurposeResponse converts a ConsentPurpose model to a response object
func (s *ConsentPurposeService) buildPurposeResponse(purpose *models.ConsentPurpose) *models.ConsentPurposeResponse {
	return &models.ConsentPurposeResponse{
		ID:          purpose.ID,
		Name:        purpose.Name,
		Description: purpose.Description,
		Type:        purpose.Type,
		Attributes:  purpose.Attributes,
	}
}
