package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/wso2/consent-management-api/internal/dao"
	"github.com/wso2/consent-management-api/internal/database"
	"github.com/wso2/consent-management-api/internal/models"
	"github.com/wso2/consent-management-api/pkg/utils"

	"github.com/sirupsen/logrus"
)

// ConsentService handles business logic for consent operations
type ConsentService struct {
	consentDAO        *dao.ConsentDAO
	statusAuditDAO    *dao.StatusAuditDAO
	attributeDAO      *dao.ConsentAttributeDAO
	authResourceDAO   *dao.AuthResourceDAO
	consentPurposeDAO *dao.ConsentPurposeDAO
	db                *database.DB
	logger            *logrus.Logger
}

// NewConsentService creates a new consent service instance
func NewConsentService(
	consentDAO *dao.ConsentDAO,
	statusAuditDAO *dao.StatusAuditDAO,
	attributeDAO *dao.ConsentAttributeDAO,
	authResourceDAO *dao.AuthResourceDAO,
	consentPurposeDAO *dao.ConsentPurposeDAO,
	db *database.DB,
	logger *logrus.Logger,
) *ConsentService {
	return &ConsentService{
		consentDAO:        consentDAO,
		statusAuditDAO:    statusAuditDAO,
		attributeDAO:      attributeDAO,
		authResourceDAO:   authResourceDAO,
		consentPurposeDAO: consentPurposeDAO,
		db:                db,
		logger:            logger,
	}
}

// CreateConsent creates a new consent
func (s *ConsentService) CreateConsent(ctx context.Context, request *models.ConsentCreateRequest, clientID, orgID string) (*models.ConsentResponse, error) {
	return s.CreateConsentWithPurposes(ctx, request, clientID, orgID, nil)
}

// CreateConsentWithPurposes creates a new consent and links it to purposes
func (s *ConsentService) CreateConsentWithPurposes(ctx context.Context, request *models.ConsentCreateRequest, clientID, orgID string, purposeNames []string) (*models.ConsentResponse, error) {
	// Validate request
	if err := s.validateConsentCreateRequest(request, clientID, orgID); err != nil {
		return nil, err
	}

	// Convert consent purpose array to JSON
	consentPurposeJSON, err := json.Marshal(request.ConsentPurpose)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal consent purpose: %w", err)
	}

	// Build consent model
	consent := &models.Consent{
		ConsentID:                  utils.GenerateConsentID(),
		ConsentPurposes:            models.JSON(consentPurposeJSON),
		ClientID:                   clientID,
		ConsentType:                request.ConsentType,
		CurrentStatus:              request.CurrentStatus,
		ConsentFrequency:           request.ConsentFrequency,
		ValidityTime:               request.ValidityTime,
		RecurringIndicator:         request.RecurringIndicator,
		DataAccessValidityDuration: request.DataAccessValidityDuration,
		OrgID:                      orgID,
		CreatedTime:                utils.GetCurrentTimeMillis(),
		UpdatedTime:                utils.GetCurrentTimeMillis(),
	}

	// Start transaction
	tx, err := s.db.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Create consent
	if err := s.consentDAO.CreateWithTx(ctx, tx, consent); err != nil {
		return nil, fmt.Errorf("failed to create consent: %w", err)
	}

	// Create audit record
	actionBy := clientID
	previousStatus := ""
	reason := "Initial consent creation"
	audit := &models.ConsentStatusAudit{
		StatusAuditID:  utils.GenerateAuditID(),
		ConsentID:      consent.ConsentID,
		CurrentStatus:  consent.CurrentStatus,
		ActionTime:     consent.CreatedTime,
		ActionBy:       &actionBy,
		PreviousStatus: &previousStatus,
		Reason:         &reason,
		OrgID:          consent.OrgID,
	}

	if err := s.statusAuditDAO.CreateWithTx(ctx, tx, audit); err != nil {
		return nil, fmt.Errorf("failed to create audit record: %w", err)
	}

	// Create attributes
	if len(request.Attributes) > 0 {
		if err := s.attributeDAO.CreateWithTx(ctx, tx, consent.ConsentID, consent.OrgID, request.Attributes); err != nil {
			return nil, fmt.Errorf("failed to create attributes: %w", err)
		}
	}

	// Create authorization resources
	if len(request.AuthResources) > 0 {
		for _, authReq := range request.AuthResources {
			// Marshal approved purpose details to JSON if present, else set to empty struct
			var approvedPurposeDetailsJSON *string
			if authReq.ApprovedPurposeDetails == nil {
				emptyDetailsBytes, _ := json.Marshal(models.ApprovedPurposeDetails{})
				emptyDetailsStr := string(emptyDetailsBytes)
				approvedPurposeDetailsJSON = &emptyDetailsStr
			} else {
				approvedPurposeDetailsBytes, err := json.Marshal(authReq.ApprovedPurposeDetails)
				if err != nil {
					return nil, fmt.Errorf("failed to marshal approved purpose details: %w", err)
				}
				approvedPurposeDetailsStr := string(approvedPurposeDetailsBytes)
				approvedPurposeDetailsJSON = &approvedPurposeDetailsStr
			}

			authResource := &models.ConsentAuthResource{
				AuthID:                 utils.GenerateAuthID(),
				ConsentID:              consent.ConsentID,
				AuthType:               authReq.AuthType,
				UserID:                 authReq.UserID,
				AuthStatus:             authReq.AuthStatus,
				UpdatedTime:            consent.CreatedTime,
				ApprovedPurposeDetails: approvedPurposeDetailsJSON,
				OrgID:                  consent.OrgID,
			}

			if err := s.authResourceDAO.CreateWithTx(ctx, tx, authResource); err != nil {
				return nil, fmt.Errorf("failed to create auth resource: %w", err)
			}
		}
	}

	// Link consent purposes if provided
	if len(purposeNames) > 0 {
		// Get purpose ID's by names
		purposeIDMap, err := s.consentPurposeDAO.GetIDsByNames(ctx, purposeNames, orgID)
		if err != nil {
			return nil, fmt.Errorf("failed to get purpose IDs: %w", err)
		}

		// Verify all purposes were found
		if len(purposeIDMap) != len(purposeNames) {
			missingPurposes := []string{}
			for _, name := range purposeNames {
				if _, found := purposeIDMap[name]; !found {
					missingPurposes = append(missingPurposes, name)
				}
			}
			return nil, fmt.Errorf("purposes not found: %v", missingPurposes)
		}

		// Link each purpose to the consent within transaction
		for _, purposeID := range purposeIDMap {
			if err := s.consentPurposeDAO.LinkPurposeToConsentWithTx(ctx, tx.Tx, consent.ConsentID, purposeID, orgID); err != nil {
				return nil, fmt.Errorf("failed to link purpose: %w", err)
			}
		}

		s.logger.WithFields(logrus.Fields{
			"consent_id":    consent.ConsentID,
			"purpose_count": len(purposeNames),
		}).Info("Linked purposes to consent within transaction")
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Retrieve auth resources after creation
	authResources, _ := s.authResourceDAO.GetByConsentID(ctx, consent.ConsentID, consent.OrgID)

	return s.buildConsentResponse(consent, request.Attributes, authResources), nil
}

// GetConsent retrieves a consent by ID
func (s *ConsentService) GetConsent(ctx context.Context, consentID, orgID string) (*models.ConsentResponse, error) {
	if err := utils.ValidateConsentID(consentID); err != nil {
		return nil, err
	}
	if err := utils.ValidateOrgID(orgID); err != nil {
		return nil, err
	}

	consent, err := s.consentDAO.GetByID(ctx, consentID, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve consent: %w", err)
	}

	attributes, _ := s.attributeDAO.GetByConsentID(ctx, consentID, orgID)
	authResources, _ := s.authResourceDAO.GetByConsentID(ctx, consentID, orgID)

	return s.buildConsentResponse(consent, attributes, authResources), nil
}

// UpdateConsent updates an existing consent
func (s *ConsentService) UpdateConsent(ctx context.Context, consentID, orgID string, request *models.ConsentUpdateRequest) (*models.ConsentResponse, error) {
	return s.UpdateConsentWithPurposes(ctx, consentID, orgID, request, nil)
}

// UpdateConsentWithPurposes updates a consent and replaces its purpose mappings
func (s *ConsentService) UpdateConsentWithPurposes(ctx context.Context, consentID, orgID string, request *models.ConsentUpdateRequest, purposeNames []string) (*models.ConsentResponse, error) {
	// Validate inputs
	if err := utils.ValidateConsentID(consentID); err != nil {
		return nil, err
	}
	if err := utils.ValidateOrgID(orgID); err != nil {
		return nil, err
	}
	if request.CurrentStatus != "" {
		if err := utils.ValidateStatus(request.CurrentStatus); err != nil {
			return nil, err
		}
	}
	if request.ConsentType != "" {
		if err := utils.ValidateConsentType(request.ConsentType); err != nil {
			return nil, err
		}
	}

	// Start transaction
	tx, err := s.db.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Get existing consent
	existingConsent, err := s.consentDAO.GetByIDWithTx(ctx, tx, consentID, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve consent: %w", err)
	}

	// Track if status changed
	statusChanged := false
	previousStatus := existingConsent.CurrentStatus

	// Update consent fields
	updatedConsent := *existingConsent
	updatedConsent.UpdatedTime = utils.GetCurrentTimeMillis()

	if request.ConsentPurpose != nil && len(request.ConsentPurpose) > 0 {
		consentPurposeJSON, err := json.Marshal(request.ConsentPurpose)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal consent purpose: %w", err)
		}
		updatedConsent.ConsentPurposes = models.JSON(consentPurposeJSON)
	}

	if request.ConsentType != "" {
		updatedConsent.ConsentType = request.ConsentType
	}

	if request.CurrentStatus != "" && request.CurrentStatus != existingConsent.CurrentStatus {
		updatedConsent.CurrentStatus = request.CurrentStatus
		statusChanged = true
	}

	if request.ConsentFrequency != nil {
		updatedConsent.ConsentFrequency = request.ConsentFrequency
	}

	if request.ValidityTime != nil {
		updatedConsent.ValidityTime = request.ValidityTime
	}

	if request.RecurringIndicator != nil {
		updatedConsent.RecurringIndicator = request.RecurringIndicator
	}

	if request.DataAccessValidityDuration != nil {
		// Validate that it's non-negative
		if *request.DataAccessValidityDuration < 0 {
			return nil, fmt.Errorf("dataAccessValidityDuration must be non-negative")
		}
		updatedConsent.DataAccessValidityDuration = request.DataAccessValidityDuration
	}

	// Update consent
	if err := s.consentDAO.UpdateWithTx(ctx, tx, &updatedConsent); err != nil {
		return nil, fmt.Errorf("failed to update consent: %w", err)
	}

	// Create status audit if status changed
	if statusChanged {
		actionBy := updatedConsent.ClientID
		reason := "Consent status updated"
		audit := &models.ConsentStatusAudit{
			StatusAuditID:  utils.GenerateAuditID(),
			ConsentID:      consentID,
			CurrentStatus:  updatedConsent.CurrentStatus,
			ActionTime:     updatedConsent.UpdatedTime,
			ActionBy:       &actionBy,
			PreviousStatus: &previousStatus,
			Reason:         &reason,
			OrgID:          orgID,
		}

		if err := s.statusAuditDAO.CreateWithTx(ctx, tx, audit); err != nil {
			return nil, fmt.Errorf("failed to create audit record: %w", err)
		}
	}

	// Update attributes if provided
	if request.Attributes != nil {
		// Delete existing attributes and create new ones
		if err := s.attributeDAO.DeleteByConsentIDWithTx(ctx, tx, consentID, orgID); err != nil {
			return nil, fmt.Errorf("failed to delete existing attributes: %w", err)
		}

		if len(request.Attributes) > 0 {
			if err := s.attributeDAO.CreateWithTx(ctx, tx, consentID, orgID, request.Attributes); err != nil {
				return nil, fmt.Errorf("failed to create updated attributes: %w", err)
			}
		}
	}

	// Update auth resources if provided
	if request.AuthResources != nil {
		// Delete existing auth resources and create new ones
		if err := s.authResourceDAO.DeleteByConsentIDWithTx(ctx, tx, consentID, orgID); err != nil {
			return nil, fmt.Errorf("failed to delete existing auth resources: %w", err)
		}

		if len(request.AuthResources) > 0 {
			for _, authReq := range request.AuthResources {
				// Marshal approved purpose details to JSON if present, else set to empty struct
				var approvedPurposeDetailsJSON *string
				if authReq.ApprovedPurposeDetails == nil {
					emptyDetailsBytes, _ := json.Marshal(models.ApprovedPurposeDetails{})
					emptyDetailsStr := string(emptyDetailsBytes)
					approvedPurposeDetailsJSON = &emptyDetailsStr
				} else {
					approvedPurposeDetailsBytes, err := json.Marshal(authReq.ApprovedPurposeDetails)
					if err != nil {
						return nil, fmt.Errorf("failed to marshal approved purpose details: %w", err)
					}
					approvedPurposeDetailsStr := string(approvedPurposeDetailsBytes)
					approvedPurposeDetailsJSON = &approvedPurposeDetailsStr
				}

				authResource := &models.ConsentAuthResource{
					AuthID:                 utils.GenerateAuthID(),
					ConsentID:              consentID,
					AuthType:               authReq.AuthType,
					UserID:                 authReq.UserID,
					AuthStatus:             authReq.AuthStatus,
					UpdatedTime:            updatedConsent.UpdatedTime,
					ApprovedPurposeDetails: approvedPurposeDetailsJSON,
					OrgID:                  orgID,
				}

				if err := s.authResourceDAO.CreateWithTx(ctx, tx, authResource); err != nil {
					return nil, fmt.Errorf("failed to create auth resource: %w", err)
				}
			}
		}
	}

	// Update consent purposes if provided
	if purposeNames != nil {
		// Clear existing purpose mappings
		if err := s.consentPurposeDAO.ClearConsentPurposesWithTx(ctx, tx.Tx, consentID, orgID); err != nil {
			return nil, fmt.Errorf("failed to clear existing purposes: %w", err)
		}

		// Link new purposes if any
		if len(purposeNames) > 0 {
			// Get purpose IDs by names
			purposeIDMap, err := s.consentPurposeDAO.GetIDsByNames(ctx, purposeNames, orgID)
			if err != nil {
				return nil, fmt.Errorf("failed to get purpose IDs: %w", err)
			}

			// Verify all purposes were found
			if len(purposeIDMap) != len(purposeNames) {
				missingPurposes := []string{}
				for _, name := range purposeNames {
					if _, found := purposeIDMap[name]; !found {
						missingPurposes = append(missingPurposes, name)
					}
				}
				return nil, fmt.Errorf("purposes not found: %v", missingPurposes)
			}

			// Link each purpose to the consent within transaction
			for _, purposeID := range purposeIDMap {
				if err := s.consentPurposeDAO.LinkPurposeToConsentWithTx(ctx, tx.Tx, consentID, purposeID, orgID); err != nil {
					return nil, fmt.Errorf("failed to link purpose: %w", err)
				}
			}

			s.logger.WithFields(logrus.Fields{
				"consent_id":    consentID,
				"purpose_count": len(purposeNames),
			}).Info("Updated consent purposes within transaction")
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Retrieve and return updated consent
	attributes, _ := s.attributeDAO.GetByConsentID(ctx, consentID, orgID)
	authResources, _ := s.authResourceDAO.GetByConsentID(ctx, consentID, orgID)

	return s.buildConsentResponse(&updatedConsent, attributes, authResources), nil
}

// RevokeConsent revokes a consent
func (s *ConsentService) RevokeConsent(ctx context.Context, consentID, orgID, reason, actionBy string) error {
	tx, err := s.db.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	consent, err := s.consentDAO.GetByIDWithTx(ctx, tx, consentID, orgID)
	if err != nil {
		return fmt.Errorf("failed to retrieve consent: %w", err)
	}

	previousStatus := consent.CurrentStatus
	newStatus := "REVOKED"
	currentTime := utils.GetCurrentTimeMillis()

	if err := s.consentDAO.UpdateStatusWithTx(ctx, tx, consentID, orgID, newStatus, currentTime); err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	audit := &models.ConsentStatusAudit{
		StatusAuditID:  utils.GenerateAuditID(),
		ConsentID:      consentID,
		CurrentStatus:  newStatus,
		ActionTime:     currentTime,
		ActionBy:       &actionBy,
		PreviousStatus: &previousStatus,
		Reason:         &reason,
		OrgID:          orgID,
	}

	if err := s.statusAuditDAO.CreateWithTx(ctx, tx, audit); err != nil {
		return fmt.Errorf("failed to create audit record: %w", err)
	}

	return tx.Commit()
}

// SearchConsents searches for consents
func (s *ConsentService) SearchConsents(ctx context.Context, params *models.ConsentSearchParams) ([]*models.ConsentResponse, *utils.PaginationMetadata, error) {
	if params.Limit <= 0 {
		params.Limit = 10
	}
	if params.Limit > 100 {
		params.Limit = 100
	}

	consents, total, err := s.consentDAO.Search(ctx, params)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to search consents: %w", err)
	}

	responses := make([]*models.ConsentResponse, 0, len(consents))
	for _, consent := range consents {
		attributes, _ := s.attributeDAO.GetByConsentID(ctx, consent.ConsentID, consent.OrgID)
		authResources, _ := s.authResourceDAO.GetByConsentID(ctx, consent.ConsentID, consent.OrgID)
		responses = append(responses, s.buildConsentResponse(&consent, attributes, authResources))
	}

	pagination := utils.CalculatePaginationMetadata(total, params.Limit, params.Offset)

	return responses, pagination, nil
}

func (s *ConsentService) validateConsentCreateRequest(request *models.ConsentCreateRequest, clientID, orgID string) error {
	if err := utils.ValidateClientID(clientID); err != nil {
		return err
	}
	if err := utils.ValidateOrgID(orgID); err != nil {
		return err
	}
	if err := utils.ValidateConsentType(request.ConsentType); err != nil {
		return err
	}
	if err := utils.ValidateStatus(request.CurrentStatus); err != nil {
		return err
	}
	if request.ConsentPurpose == nil || len(request.ConsentPurpose) == 0 {
		return fmt.Errorf("consentPurpose is required")
	}
	// Validate DataAccessValidityDuration if provided (must be non-negative)
	if request.DataAccessValidityDuration != nil && *request.DataAccessValidityDuration < 0 {
		return fmt.Errorf("dataAccessValidityDuration must be non-negative")
	}
	return nil
}

func (s *ConsentService) buildConsentResponse(consent *models.Consent, attributes map[string]string, authResources []models.ConsentAuthResource) *models.ConsentResponse {
	// Unmarshal consent purposes from JSON
	var consentPurpose []models.ConsentPurposeItem
	if len(consent.ConsentPurposes) > 0 {
		if err := json.Unmarshal(consent.ConsentPurposes, &consentPurpose); err != nil {
			s.logger.WithError(err).Warn("Failed to unmarshal consent purposes")
			consentPurpose = nil
		}
	}

	// Convert auth resources to response format
	var authResourceResponses []models.ConsentAuthResource
	if authResources != nil {
		authResourceResponses = make([]models.ConsentAuthResource, len(authResources))
		for i, ar := range authResources {
			authResourceResponses[i] = ar
			// Unmarshal approved purpose details if present
			if ar.ApprovedPurposeDetails != nil {
				var approvedPurposeDetails models.ApprovedPurposeDetails
				if err := json.Unmarshal([]byte(*ar.ApprovedPurposeDetails), &approvedPurposeDetails); err != nil {
					s.logger.WithError(err).Warn("Failed to unmarshal approved purpose details")
				} else {
					authResourceResponses[i].ApprovedPurposeDetailsObj = &approvedPurposeDetails
				}
			}
		}
	}

	return &models.ConsentResponse{
		ConsentID:                  consent.ConsentID,
		ConsentPurpose:             consentPurpose,
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
		AuthResources:              authResourceResponses,
	}
}

// SearchConsentIDsByAttribute searches for consent IDs by attribute key and/or value
func (s *ConsentService) SearchConsentIDsByAttribute(ctx context.Context, key, value, orgID string) ([]string, error) {
	// Validate that key is provided
	if key == "" {
		return nil, fmt.Errorf("attribute key is required")
	}

	// Validate orgID
	if orgID == "" {
		return nil, fmt.Errorf("organization ID is required")
	}

	var consentIDs []string
	var err error

	// Search based on parameters
	if value == "" {
		// Search by key only
		consentIDs, err = s.attributeDAO.FindConsentIDsByAttributeKey(ctx, key, orgID)
	} else {
		// Search by key and value
		consentIDs, err = s.attributeDAO.FindConsentIDsByAttribute(ctx, key, value, orgID)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to search consent IDs by attribute: %w", err)
	}

	// Return empty slice instead of nil if no results
	if consentIDs == nil {
		consentIDs = []string{}
	}

	return consentIDs, nil
}
