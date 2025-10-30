package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"

	"github.com/wso2/consent-management-api/internal/dao"
	"github.com/wso2/consent-management-api/internal/models"
	"github.com/wso2/consent-management-api/pkg/utils"
)

// ConsentPurposeService handles business logic for consent purposes
type ConsentPurposeService struct {
	purposeDAO *dao.ConsentPurposeDAO
	consentDAO *dao.ConsentDAO
	db         *sqlx.DB
	logger     *logrus.Logger
}

// NewConsentPurposeService creates a new ConsentPurposeService
func NewConsentPurposeService(
	purposeDAO *dao.ConsentPurposeDAO,
	consentDAO *dao.ConsentDAO,
	db *sqlx.DB,
	logger *logrus.Logger,
) *ConsentPurposeService {
	return &ConsentPurposeService{
		purposeDAO: purposeDAO,
		consentDAO: consentDAO,
		db:         db,
		logger:     logger,
	}
}

// ConsentPurposeCreateRequest represents request to create a consent purpose
type ConsentPurposeCreateRequest struct {
	Name        string
	Description *string
	Type        string
	Value       interface{}
}

// ConsentPurposeUpdateRequest represents request to update a consent purpose
// All fields are required - no partial updates allowed
type ConsentPurposeUpdateRequest struct {
	Name        string
	Description *string
	Type        string
	Value       interface{}
}

// convertToJSONValue converts an interface{} to *models.JSONValue
func convertToJSONValue(value interface{}) (*models.JSONValue, error) {
	if value == nil {
		return nil, nil
	}

	// Marshal the value to JSON
	jsonBytes, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal value to JSON: %w", err)
	}

	jsonValue := models.JSONValue(jsonBytes)
	return &jsonValue, nil
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
	purposeID := "PURPOSE-" + utils.GenerateID()

	// Convert value to JSONValue
	jsonValue, err := convertToJSONValue(req.Value)
	if err != nil {
		s.logger.WithError(err).Error("Failed to convert value to JSON")
		return nil, fmt.Errorf("invalid value format: %w", err)
	}

	// Create purpose object
	purpose := &models.ConsentPurpose{
		ID:          purposeID,
		Name:        req.Name,
		Description: req.Description,
		Type:        req.Type,
		Value:       jsonValue,
		OrgID:       orgID,
	}

	// Save to database
	if err := s.purposeDAO.Create(ctx, purpose); err != nil {
		s.logger.WithError(err).Error("Failed to create consent purpose")
		return nil, fmt.Errorf("failed to create consent purpose: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"purpose_id": purposeID,
		"org_id":     orgID,
	}).Info("Consent purpose created successfully")

	return s.buildPurposeResponse(purpose), nil
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
		// Generate purpose ID
		purposeID := "PURPOSE-" + utils.GenerateID()

		// Convert value to JSONValue
		jsonValue, err := convertToJSONValue(req.Value)
		if err != nil {
			s.logger.WithError(err).Error("Failed to convert value to JSON")
			return nil, fmt.Errorf("invalid value format at index %d: %w", i, err)
		}

		// Create purpose object
		purpose := &models.ConsentPurpose{
			ID:          purposeID,
			Name:        req.Name,
			Description: req.Description,
			Type:        req.Type,
			Value:       jsonValue,
			OrgID:       orgID,
		}

		// Save to database using transaction
		if err := s.purposeDAO.CreateWithTx(ctx, tx, purpose); err != nil {
			s.logger.WithError(err).Error("Failed to create consent purpose")
			return nil, fmt.Errorf("failed to create consent purpose at index %d: %w", i, err)
		}

		// Add to response list
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

	// Build response
	purposeResponses := make([]models.ConsentPurposeResponse, 0, len(purposes))
	for _, purpose := range purposes {
		purposeResponses = append(purposeResponses, *s.buildPurposeResponse(&purpose))
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

	// Build response
	purposeResponses := make([]models.ConsentPurposeResponse, 0, len(purposes))
	for _, purpose := range purposes {
		purposeResponses = append(purposeResponses, *s.buildPurposeResponse(&purpose))
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

	// Validate purpose type (now required)
	if err := models.ValidatePurposeType(req.Type); err != nil {
		return nil, fmt.Errorf("invalid purpose type: %w", err)
	}

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

	// Convert value to JSONValue
	jsonValue, err := convertToJSONValue(req.Value)
	if err != nil {
		s.logger.WithError(err).Error("Failed to convert value to JSON")
		return nil, fmt.Errorf("invalid value format: %w", err)
	}

	// Update all fields (no partial updates)
	existingPurpose.Name = req.Name
	existingPurpose.Description = req.Description
	existingPurpose.Type = req.Type
	existingPurpose.Value = jsonValue

	// Update in database
	if err := s.purposeDAO.Update(ctx, existingPurpose); err != nil {
		s.logger.WithError(err).Error("Failed to update consent purpose")
		return nil, fmt.Errorf("failed to update consent purpose: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"purpose_id": purposeID,
		"org_id":     orgID,
	}).Info("Consent purpose updated successfully")

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

	// Delete purpose
	if err := s.purposeDAO.Delete(ctx, purposeID, orgID); err != nil {
		s.logger.WithError(err).Error("Failed to delete consent purpose")
		return fmt.Errorf("failed to delete consent purpose: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"purpose_id": purposeID,
		"org_id":     orgID,
	}).Info("Consent purpose deleted successfully")

	return nil
}

// LinkPurposeToConsent links a purpose to a consent
func (s *ConsentPurposeService) LinkPurposeToConsent(ctx context.Context, consentID, purposeID, orgID string) error {
	// Validate inputs
	if consentID == "" {
		return fmt.Errorf("consent ID is required")
	}
	if purposeID == "" {
		return fmt.Errorf("purpose ID is required")
	}

	if err := utils.ValidateOrgID(orgID); err != nil {
		return err
	}

	// Verify consent exists
	_, err := s.consentDAO.GetByID(ctx, consentID, orgID)
	if err != nil {
		return fmt.Errorf("consent not found: %w", err)
	}

	// Verify purpose exists
	_, err = s.purposeDAO.GetByID(ctx, purposeID, orgID)
	if err != nil {
		return fmt.Errorf("purpose not found: %w", err)
	}

	// Create the link
	if err := s.purposeDAO.LinkPurposeToConsent(ctx, consentID, purposeID, orgID); err != nil {
		s.logger.WithError(err).Error("Failed to link purpose to consent")
		return fmt.Errorf("failed to link purpose to consent: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"consent_id": consentID,
		"purpose_id": purposeID,
		"org_id":     orgID,
	}).Info("Purpose linked to consent successfully")

	return nil
}

// UnlinkPurposeFromConsent removes a purpose link from a consent
func (s *ConsentPurposeService) UnlinkPurposeFromConsent(ctx context.Context, consentID, purposeID, orgID string) error {
	// Validate inputs
	if consentID == "" {
		return fmt.Errorf("consent ID is required")
	}
	if purposeID == "" {
		return fmt.Errorf("purpose ID is required")
	}

	if err := utils.ValidateOrgID(orgID); err != nil {
		return err
	}

	// Unlink the purpose
	if err := s.purposeDAO.UnlinkPurposeFromConsent(ctx, consentID, purposeID, orgID); err != nil {
		s.logger.WithError(err).Error("Failed to unlink purpose from consent")
		return fmt.Errorf("failed to unlink purpose from consent: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"consent_id": consentID,
		"purpose_id": purposeID,
		"org_id":     orgID,
	}).Info("Purpose unlinked from consent successfully")

	return nil
}

// GetPurposesForConsent retrieves all purposes linked to a consent
func (s *ConsentPurposeService) GetPurposesForConsent(ctx context.Context, consentID, orgID string) (*models.ConsentPurposeListResponse, error) {
	// Validate inputs
	if consentID == "" {
		return nil, fmt.Errorf("consent ID is required")
	}

	if err := utils.ValidateOrgID(orgID); err != nil {
		return nil, err
	}

	// Retrieve purposes
	purposes, err := s.purposeDAO.GetByConsentID(ctx, consentID, orgID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get purposes for consent")
		return nil, fmt.Errorf("failed to get purposes for consent: %w", err)
	}

	// Build response
	purposeResponses := make([]models.ConsentPurposeResponse, 0, len(purposes))
	for _, purpose := range purposes {
		purposeResponses = append(purposeResponses, *s.buildPurposeResponse(&purpose))
	}

	return &models.ConsentPurposeListResponse{
		Purposes: purposeResponses,
		Total:    len(purposeResponses),
	}, nil
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

	// Value is required (can be any valid JSON value, but must be present)
	if req.Value == nil {
		return fmt.Errorf("purpose value is required")
	}

	return nil
}

// buildPurposeResponse converts a ConsentPurpose model to a response object
func (s *ConsentPurposeService) buildPurposeResponse(purpose *models.ConsentPurpose) *models.ConsentPurposeResponse {
	return &models.ConsentPurposeResponse{
		ID:          purpose.ID,
		Name:        purpose.Name,
		Description: purpose.Description,
		Type:        purpose.Type,
		Value:       purpose.Value,
	}
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

// LinkPurposesToConsent links multiple purposes to a consent by purpose names
// This should be called within a transaction
func (s *ConsentPurposeService) LinkPurposesToConsent(ctx context.Context, consentID, orgID string, purposeNames []string) error {
	if len(purposeNames) == 0 {
		return nil // Nothing to link
	}

	if consentID == "" {
		return fmt.Errorf("consent ID is required")
	}
	if orgID == "" {
		return fmt.Errorf("organization ID is required")
	}

	// Get purpose IDs by names
	purposeIDMap, err := s.purposeDAO.GetIDsByNames(ctx, purposeNames, orgID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get purpose IDs by names")
		return fmt.Errorf("failed to get purpose IDs: %w", err)
	}

	// Verify all purposes were found
	if len(purposeIDMap) != len(purposeNames) {
		missingPurposes := []string{}
		for _, name := range purposeNames {
			if _, found := purposeIDMap[name]; !found {
				missingPurposes = append(missingPurposes, name)
			}
		}
		return fmt.Errorf("purposes not found: %v", missingPurposes)
	}

	// Link each purpose to the consent
	for _, purposeID := range purposeIDMap {
		if err := s.purposeDAO.LinkPurposeToConsent(ctx, consentID, purposeID, orgID); err != nil {
			s.logger.WithError(err).WithFields(logrus.Fields{
				"consent_id": consentID,
				"purpose_id": purposeID,
			}).Error("Failed to link purpose to consent")
			return fmt.Errorf("failed to link purpose: %w", err)
		}
	}

	s.logger.WithFields(logrus.Fields{
		"consent_id":    consentID,
		"purpose_count": len(purposeNames),
	}).Info("Successfully linked purposes to consent")

	return nil
}

// UpdateConsentPurposes updates the purpose mappings for a consent
// Clears existing mappings and creates new ones
// This should be called within a transaction
func (s *ConsentPurposeService) UpdateConsentPurposes(ctx context.Context, consentID, orgID string, purposeNames []string) error {
	if consentID == "" {
		return fmt.Errorf("consent ID is required")
	}
	if orgID == "" {
		return fmt.Errorf("organization ID is required")
	}

	// Clear existing mappings
	if err := s.purposeDAO.ClearConsentPurposes(ctx, consentID, orgID); err != nil {
		s.logger.WithError(err).WithField("consent_id", consentID).Error("Failed to clear existing purpose mappings")
		return fmt.Errorf("failed to clear existing purposes: %w", err)
	}

	// Link new purposes if any
	if len(purposeNames) > 0 {
		if err := s.LinkPurposesToConsent(ctx, consentID, orgID, purposeNames); err != nil {
			return err
		}
	}

	s.logger.WithFields(logrus.Fields{
		"consent_id":    consentID,
		"purpose_count": len(purposeNames),
	}).Info("Successfully updated consent purposes")

	return nil
}
