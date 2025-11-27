package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/wso2/consent-management-api/internal/config"
	"github.com/wso2/consent-management-api/internal/dao"
	"github.com/wso2/consent-management-api/internal/database"
	"github.com/wso2/consent-management-api/internal/models"
	serviceutils "github.com/wso2/consent-management-api/internal/service/utils"
	"github.com/wso2/consent-management-api/internal/utils"

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
func (s *ConsentService) CreateConsentWithPurposes(ctx context.Context, request *models.ConsentCreateRequest, clientID, orgID string, consentPurposes []models.ConsentPurposeItem) (*models.ConsentResponse, error) {
	// Validate request
	if err := s.validateConsentCreateRequest(request, clientID, orgID); err != nil {
		return nil, err
	}

	// Note: ConsentPurpose array will be stored in CONSENT_PURPOSE_MAPPING table,
	// not in the CONSENT table's JSON column anymore

	// Build consent model
	consent := &models.Consent{
		ConsentID:                  utils.GenerateConsentID(),
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
			// Marshal resources to JSON if present (resources can be any valid JSON)
			var resourcesJSON *string
			if authReq.Resources != nil {
				resourcesBytes, err := json.Marshal(authReq.Resources)
				if err != nil {
					return nil, fmt.Errorf("failed to marshal resources: %w", err)
				}
				resourcesStr := string(resourcesBytes)
				resourcesJSON = &resourcesStr
			}

			authResource := &models.ConsentAuthResource{
				AuthID:      utils.GenerateAuthID(),
				ConsentID:   consent.ConsentID,
				AuthType:    authReq.AuthType,
				UserID:      authReq.UserID,
				AuthStatus:  authReq.AuthStatus,
				UpdatedTime: consent.CreatedTime,
				Resources:   resourcesJSON,
				OrgID:       consent.OrgID,
			}

			if err := s.authResourceDAO.CreateWithTx(ctx, tx, authResource); err != nil {
				return nil, fmt.Errorf("failed to create auth resource: %w", err)
			}
		}
	}

	// Link consent purposes if provided
	if len(consentPurposes) > 0 {
		// Extract purpose names for ID lookup
		purposeNames := make([]string, len(consentPurposes))
		for i, p := range consentPurposes {
			purposeNames[i] = p.Name
		}

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

		// Link each purpose to the consent within transaction with value and isSelected
		for _, purposeItem := range consentPurposes {
			purposeName := purposeIDMap[purposeItem.Name]

			// Marshal value to JSON string if present
			var valueJSON *string
			if purposeItem.Value != nil {
				valueBytes, err := json.Marshal(purposeItem.Value)
				if err != nil {
					return nil, fmt.Errorf("failed to marshal purpose value: %w", err)
				}
				valueStr := string(valueBytes)
				valueJSON = &valueStr
			}

			// Dereference IsUserApproved and IsMandatory pointers (should not be nil at this point due to defaulting in ToConsentCreateRequest)
			isUserApproved := false
			if purposeItem.IsUserApproved != nil {
				isUserApproved = *purposeItem.IsUserApproved
			}

			isMandatory := true
			if purposeItem.IsMandatory != nil {
				isMandatory = *purposeItem.IsMandatory
			}

			if err := s.consentPurposeDAO.LinkPurposeToConsentWithTx(ctx, tx.Tx, consent.ConsentID, purposeName, orgID, valueJSON, isUserApproved, isMandatory); err != nil {
				return nil, fmt.Errorf("failed to link purpose: %w", err)
			}
		}

		s.logger.WithFields(logrus.Fields{
			"consent_id":    consent.ConsentID,
			"purpose_count": len(consentPurposes),
		}).Info("Linked purposes to consent within transaction")
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Retrieve related data after creation
	authResources, _ := s.authResourceDAO.GetByConsentID(ctx, consent.ConsentID, consent.OrgID)
	purposeMappings, _ := s.consentPurposeDAO.GetMappingsByConsentID(ctx, consent.ConsentID, consent.OrgID)

	return serviceutils.BuildConsentResponse(consent, request.Attributes, authResources, purposeMappings, s.logger), nil
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
	purposeMappings, _ := s.consentPurposeDAO.GetMappingsByConsentID(ctx, consentID, orgID)

	return serviceutils.BuildConsentResponse(consent, attributes, authResources, purposeMappings, s.logger), nil
}

// UpdateConsentStatus updates only the status of a consent and its authorization resources
// This is a safer method for status-only updates (e.g., expiration) without affecting other fields
func (s *ConsentService) UpdateConsentStatus(ctx context.Context, consentID, orgID, newStatus, actionBy, reason string) (*models.ConsentResponse, error) {
	// Validate inputs
	if err := utils.ValidateConsentID(consentID); err != nil {
		return nil, err
	}
	if err := utils.ValidateOrgID(orgID); err != nil {
		return nil, err
	}
	if newStatus == "" {
		return nil, fmt.Errorf("status cannot be empty")
	}

	// Validate that the new status is one of the allowed statuses from config
	cfg := config.Get()
	if !cfg.Consent.IsStatusAllowed(newStatus) {
		return nil, fmt.Errorf("invalid status '%s': must be one of %v", newStatus, cfg.Consent.AllowedStatuses)
	}

	// Start transaction
	tx, err := s.db.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Get existing consent to capture previous status
	existingConsent, err := s.consentDAO.GetByIDWithTx(ctx, tx, consentID, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve consent: %w", err)
	}

	// Update consent status
	updatedTime := utils.GetCurrentTimeMillis()
	if err := s.consentDAO.UpdateStatusWithTx(ctx, tx, consentID, orgID, newStatus, updatedTime); err != nil {
		return nil, fmt.Errorf("failed to update consent status: %w", err)
	}

	// Update authorization statuses when consent is expired or revoked
	if cfg.Consent.IsExpiredStatus(newStatus) {
		// Update all authorization statuses to SYS_EXPIRED
		if err := s.authResourceDAO.UpdateAllStatusByConsentIDWithTx(ctx, tx, consentID, orgID, string(models.AuthStateSysExpired), updatedTime); err != nil {
			return nil, fmt.Errorf("failed to update authorization statuses to SYS_EXPIRED: %w", err)
		}
	} else if cfg.Consent.IsRevokedStatus(newStatus) {
		// Update all authorization statuses to SYS_REVOKED
		if err := s.authResourceDAO.UpdateAllStatusByConsentIDWithTx(ctx, tx, consentID, orgID, string(models.AuthStateSysRevoked), updatedTime); err != nil {
			return nil, fmt.Errorf("failed to update authorization statuses to SYS_REVOKED: %w", err)
		}
	}

	// Create status audit record
	previousStatus := existingConsent.CurrentStatus
	audit := &models.ConsentStatusAudit{
		StatusAuditID:  utils.GenerateAuditID(),
		ConsentID:      consentID,
		CurrentStatus:  newStatus,
		ActionTime:     updatedTime,
		ActionBy:       &actionBy,
		PreviousStatus: &previousStatus,
		Reason:         &reason,
		OrgID:          orgID,
	}

	if err := s.statusAuditDAO.CreateWithTx(ctx, tx, audit); err != nil {
		return nil, fmt.Errorf("failed to create audit record: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Return updated consent
	return s.GetConsent(ctx, consentID, orgID)
}

// UpdateConsent updates an existing consent
func (s *ConsentService) UpdateConsent(ctx context.Context, consentID, orgID string, request *models.ConsentUpdateRequest) (*models.ConsentResponse, error) {
	return s.UpdateConsentWithPurposes(ctx, consentID, orgID, request, nil)
}

// UpdateConsentWithPurposes updates a consent and replaces its purpose mappings
func (s *ConsentService) UpdateConsentWithPurposes(ctx context.Context, consentID, orgID string, request *models.ConsentUpdateRequest, consentPurposes []models.ConsentPurposeItem) (*models.ConsentResponse, error) {
	// Validate inputs
	if err := utils.ValidateConsentID(consentID); err != nil {
		return nil, err
	}
	if err := utils.ValidateOrgID(orgID); err != nil {
		return nil, err
	}
	if request.ConsentType != "" {
		if err := utils.ValidateConsentType(request.ConsentType); err != nil {
			return nil, err
		}
	}

	// Validate consent purposes (mandatory)
	if len(consentPurposes) == 0 {
		return nil, fmt.Errorf("consentPurpose is required")
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

	// Update consent fields
	updatedConsent := *existingConsent
	updatedConsent.UpdatedTime = utils.GetCurrentTimeMillis()

	// Note: CurrentStatus is derived from authorization states, not set directly
	// Status will be updated after processing auth resources

	if request.ConsentType != "" {
		updatedConsent.ConsentType = request.ConsentType
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
				// Marshal resources to JSON if present (resources can be any valid JSON)
				var resourcesJSON *string
				if authReq.Resources != nil {
					resourcesBytes, err := json.Marshal(authReq.Resources)
					if err != nil {
						return nil, fmt.Errorf("failed to marshal resources: %w", err)
					}
					resourcesStr := string(resourcesBytes)
					resourcesJSON = &resourcesStr
				}

				authResource := &models.ConsentAuthResource{
					AuthID:      utils.GenerateAuthID(),
					ConsentID:   consentID,
					AuthType:    authReq.AuthType,
					UserID:      authReq.UserID,
					AuthStatus:  authReq.AuthStatus,
					UpdatedTime: updatedConsent.UpdatedTime,
					Resources:   resourcesJSON,
					OrgID:       orgID,
				}

				if err := s.authResourceDAO.CreateWithTx(ctx, tx, authResource); err != nil {
					return nil, fmt.Errorf("failed to create auth resource: %w", err)
				}
			}
		}
	}

	// Derive consent status from authorization states
	var statusChanged bool
	previousStatus := existingConsent.CurrentStatus

	// Check if status changed - can happen with or without auth resources provided
	// (e.g., expiry detection in handler, or status derived from auth states)
	if request.CurrentStatus != "" && updatedConsent.CurrentStatus != request.CurrentStatus {
		updatedConsent.CurrentStatus = request.CurrentStatus
		statusChanged = true
	} else if request.AuthResources != nil {
		// If auth resources were provided but status wasn't explicitly set,
		// the status derivation already happened in the handler
	}

	// Update consent with potentially new status
	if err := s.consentDAO.UpdateWithTx(ctx, tx, &updatedConsent); err != nil {
		return nil, fmt.Errorf("failed to update consent: %w", err)
	}

	// Create status audit if status changed
	if statusChanged {
		actionBy := updatedConsent.ClientID
		reason := "Consent status updated based on authorization states"
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

	// Always update consent purposes (mandatory field)
	// Clear existing purpose mappings
	if err := s.consentPurposeDAO.ClearConsentPurposesWithTx(ctx, tx.Tx, consentID, orgID); err != nil {
		return nil, fmt.Errorf("failed to clear existing purposes: %w", err)
	}

	// Extract purpose names for ID lookup
	purposeNames := make([]string, len(consentPurposes))
	for i, p := range consentPurposes {
		purposeNames[i] = p.Name
	}

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

	// Link each purpose to the consent within transaction with value and isSelected
	for _, purposeItem := range consentPurposes {
		purposeID := purposeIDMap[purposeItem.Name]

		// Marshal value to JSON string if present
		var valueJSON *string
		if purposeItem.Value != nil {
			valueBytes, err := json.Marshal(purposeItem.Value)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal purpose value: %w", err)
			}
			valueStr := string(valueBytes)
			valueJSON = &valueStr
		}

		// Dereference IsUserApproved and IsMandatory pointers (should not be nil at this point due to defaulting in ToConsentUpdateRequest)
		isUserApproved := false
		if purposeItem.IsUserApproved != nil {
			isUserApproved = *purposeItem.IsUserApproved
		}

		isMandatory := true
		if purposeItem.IsMandatory != nil {
			isMandatory = *purposeItem.IsMandatory
		}

		if err := s.consentPurposeDAO.LinkPurposeToConsentWithTx(ctx, tx.Tx, consentID, purposeID, orgID, valueJSON, isUserApproved, isMandatory); err != nil {
			return nil, fmt.Errorf("failed to link purpose: %w", err)
		}
	}

	s.logger.WithFields(logrus.Fields{
		"consent_id":    consentID,
		"purpose_count": len(consentPurposes),
	}).Info("Updated consent purposes within transaction")

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Retrieve and return updated consent
	attributes, _ := s.attributeDAO.GetByConsentID(ctx, consentID, orgID)
	authResources, _ := s.authResourceDAO.GetByConsentID(ctx, consentID, orgID)
	purposeMappings, _ := s.consentPurposeDAO.GetMappingsByConsentID(ctx, consentID, orgID)

	return serviceutils.BuildConsentResponse(&updatedConsent, attributes, authResources, purposeMappings, s.logger), nil
}

// RevokeConsent revokes a consent
// RevokeConsent revokes a consent and returns the revocation details
func (s *ConsentService) RevokeConsent(ctx context.Context, consentID, orgID string, request *models.ConsentRevokeRequest) (*models.ConsentRevokeResponse, error) {
	tx, err := s.db.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	consent, err := s.consentDAO.GetByIDWithTx(ctx, tx, consentID, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve consent: %w", err)
	}

	previousStatus := consent.CurrentStatus
	newStatus := "REVOKED"
	currentTime := utils.GetCurrentTimeMillis()

	if err := s.consentDAO.UpdateStatusWithTx(ctx, tx, consentID, orgID, newStatus, currentTime); err != nil {
		return nil, fmt.Errorf("failed to update status: %w", err)
	}

	// Update all authorization statuses to SYS_REVOKED
	if err := s.authResourceDAO.UpdateAllStatusByConsentIDWithTx(ctx, tx, consentID, orgID, string(models.AuthStateSysRevoked), currentTime); err != nil {
		return nil, fmt.Errorf("failed to update authorization statuses: %w", err)
	}

	audit := &models.ConsentStatusAudit{
		StatusAuditID:  utils.GenerateAuditID(),
		ConsentID:      consentID,
		CurrentStatus:  newStatus,
		ActionTime:     currentTime,
		ActionBy:       &request.ActionBy,
		PreviousStatus: &previousStatus,
		Reason:         &request.RevocationReason,
		OrgID:          orgID,
	}

	if err := s.statusAuditDAO.CreateWithTx(ctx, tx, audit); err != nil {
		return nil, fmt.Errorf("failed to create audit record: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Build response
	response := &models.ConsentRevokeResponse{
		ActionTime:       currentTime / 1000, // Convert milliseconds to seconds
		ActionBy:         request.ActionBy,
		RevocationReason: request.RevocationReason,
	}

	return response, nil
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
		purposeMappings, _ := s.consentPurposeDAO.GetMappingsByConsentID(ctx, consent.ConsentID, consent.OrgID)
		responses = append(responses, serviceutils.BuildConsentResponse(&consent, attributes, authResources, purposeMappings, s.logger))
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
	if len(request.ConsentPurpose) == 0 {
		return fmt.Errorf("consentPurpose is required")
	}
	// Validate DataAccessValidityDuration if provided (must be non-negative)
	if request.DataAccessValidityDuration != nil && *request.DataAccessValidityDuration < 0 {
		return fmt.Errorf("dataAccessValidityDuration must be non-negative")
	}
	return nil
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

// DeleteConsent deletes a consent by ID and orgID
func (s *ConsentService) DeleteConsent(ctx context.Context, consentID, orgID string) error {
	return s.consentDAO.Delete(ctx, consentID, orgID)
}
