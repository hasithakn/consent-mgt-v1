package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/wso2/consent-management-api/internal/dao"
	"github.com/wso2/consent-management-api/internal/database"
	"github.com/wso2/consent-management-api/internal/models"
	"github.com/wso2/consent-management-api/internal/utils"

	"github.com/sirupsen/logrus"
)

// AuthResourceService handles business logic for authorization resource operations
type AuthResourceService struct {
	authResourceDAO *dao.AuthResourceDAO
	consentDAO      *dao.ConsentDAO
	db              *database.DB
	logger          *logrus.Logger
}

// NewAuthResourceService creates a new auth resource service instance
func NewAuthResourceService(
	authResourceDAO *dao.AuthResourceDAO,
	consentDAO *dao.ConsentDAO,
	db *database.DB,
	logger *logrus.Logger,
) *AuthResourceService {
	return &AuthResourceService{
		authResourceDAO: authResourceDAO,
		consentDAO:      consentDAO,
		db:              db,
		logger:          logger,
	}
}

// CreateAuthResource creates a new authorization resource for a consent
func (s *AuthResourceService) CreateAuthResource(ctx context.Context, consentID, orgID string, request *models.ConsentAuthResourceCreateRequest) (*models.ConsentAuthResourceResponse, error) {
	if err := utils.ValidateConsentID(consentID); err != nil {
		return nil, err
	}
	if err := utils.ValidateOrgID(orgID); err != nil {
		return nil, err
	}
	if err := s.validateAuthResourceCreateRequest(request); err != nil {
		return nil, err
	}

	// Verify consent exists
	_, err := s.consentDAO.GetByID(ctx, consentID, orgID)
	if err != nil {
		return nil, fmt.Errorf("consent not found: %w", err)
	}

	// Marshal approvedPurposeDetails to JSON if present
	var approvedPurposeDetailsJSON *string
	if request.ApprovedPurposeDetails != nil {
		detailsBytes, err := json.Marshal(request.ApprovedPurposeDetails)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal approvedPurposeDetails: %w", err)
		}
		detailsStr := string(detailsBytes)
		approvedPurposeDetailsJSON = &detailsStr
	}

	// Build auth resource model
	authResource := &models.ConsentAuthResource{
		AuthID:                 utils.GenerateAuthID(),
		ConsentID:              consentID,
		AuthType:               request.AuthType,
		UserID:                 request.UserID,
		AuthStatus:             request.AuthStatus,
		UpdatedTime:            utils.GetCurrentTimeMillis(),
		ApprovedPurposeDetails: approvedPurposeDetailsJSON,
		OrgID:                  orgID,
	}

	// Create auth resource
	if err := s.authResourceDAO.Create(ctx, authResource); err != nil {
		return nil, fmt.Errorf("failed to create auth resource: %w", err)
	}

	return s.buildAuthResourceResponse(authResource), nil
}

// GetAuthResource retrieves an authorization resource by ID
func (s *AuthResourceService) GetAuthResource(ctx context.Context, authID, orgID string) (*models.ConsentAuthResourceResponse, error) {
	// Validate inputs
	if authID == "" {
		return nil, fmt.Errorf("auth ID is required")
	}
	if len(authID) > 255 {
		return nil, fmt.Errorf("auth ID too long: maximum 255 characters")
	}
	if err := utils.ValidateOrgID(orgID); err != nil {
		return nil, err
	}

	authResource, err := s.authResourceDAO.GetByID(ctx, authID, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve auth resource: %w", err)
	}

	return s.buildAuthResourceResponse(authResource), nil
}

// GetAuthResourcesByConsentID retrieves all authorization resources for a consent
func (s *AuthResourceService) GetAuthResourcesByConsentID(ctx context.Context, consentID, orgID string) (*models.ConsentAuthResourceListResponse, error) {
	// Validate inputs
	if err := utils.ValidateConsentID(consentID); err != nil {
		return nil, err
	}
	if err := utils.ValidateOrgID(orgID); err != nil {
		return nil, err
	}

	// Verify consent exists
	_, err := s.consentDAO.GetByID(ctx, consentID, orgID)
	if err != nil {
		return nil, fmt.Errorf("consent not found: %w", err)
	}

	authResources, err := s.authResourceDAO.GetByConsentID(ctx, consentID, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve auth resources: %w", err)
	}

	var responses []models.ConsentAuthResourceResponse
	for _, ar := range authResources {
		responses = append(responses, *s.buildAuthResourceResponse(&ar))
	}

	return &models.ConsentAuthResourceListResponse{
		Data: responses,
	}, nil
}

// UpdateAuthResource updates an existing authorization resource
func (s *AuthResourceService) UpdateAuthResource(ctx context.Context, authID, orgID string, request *models.ConsentAuthResourceUpdateRequest) (*models.ConsentAuthResourceResponse, error) {
	// Validate inputs
	if authID == "" {
		return nil, fmt.Errorf("auth ID is required")
	}
	if err := utils.ValidateOrgID(orgID); err != nil {
		return nil, err
	}

	// Get existing auth resource
	existingAuthResource, err := s.authResourceDAO.GetByID(ctx, authID, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve auth resource: %w", err)
	}

	// Update fields if provided
	updatedAuthResource := *existingAuthResource
	updatedAuthResource.UpdatedTime = utils.GetCurrentTimeMillis()

	if request.AuthStatus != "" {
		updatedAuthResource.AuthStatus = request.AuthStatus
	}

	if request.UserID != nil {
		updatedAuthResource.UserID = request.UserID
	}

	if request.ApprovedPurposeDetails != nil {
		detailsBytes, err := json.Marshal(request.ApprovedPurposeDetails)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal approvedPurposeDetails: %w", err)
		}
		detailsStr := string(detailsBytes)
		updatedAuthResource.ApprovedPurposeDetails = &detailsStr
	}

	// Update auth resource
	if err := s.authResourceDAO.Update(ctx, &updatedAuthResource); err != nil {
		return nil, fmt.Errorf("failed to update auth resource: %w", err)
	}

	return s.buildAuthResourceResponse(&updatedAuthResource), nil
}

// DeleteAuthResource deletes an authorization resource
func (s *AuthResourceService) DeleteAuthResource(ctx context.Context, authID, orgID string) error {
	// Validate inputs
	if authID == "" {
		return fmt.Errorf("auth ID is required")
	}
	if err := utils.ValidateOrgID(orgID); err != nil {
		return err
	}

	// Verify auth resource exists
	_, err := s.authResourceDAO.GetByID(ctx, authID, orgID)
	if err != nil {
		return fmt.Errorf("auth resource not found: %w", err)
	}

	// Delete auth resource
	if err := s.authResourceDAO.Delete(ctx, authID, orgID); err != nil {
		return fmt.Errorf("failed to delete auth resource: %w", err)
	}

	return nil
}

// Helper methods

func (s *AuthResourceService) validateAuthResourceCreateRequest(request *models.ConsentAuthResourceCreateRequest) error {
	if request.AuthType == "" {
		return fmt.Errorf("auth type is required")
	}
	if request.AuthStatus == "" {
		return fmt.Errorf("auth status is required")
	}
	return nil
}

func (s *AuthResourceService) buildAuthResourceResponse(authResource *models.ConsentAuthResource) *models.ConsentAuthResourceResponse {
	var approvedPurposeDetails *models.ApprovedPurposeDetails
	if authResource.ApprovedPurposeDetails != nil {
		var details models.ApprovedPurposeDetails
		if err := json.Unmarshal([]byte(*authResource.ApprovedPurposeDetails), &details); err == nil {
			approvedPurposeDetails = &details
		}
	}

	return &models.ConsentAuthResourceResponse{
		AuthID:                 authResource.AuthID,
		ConsentID:              authResource.ConsentID,
		AuthType:               authResource.AuthType,
		UserID:                 authResource.UserID,
		AuthStatus:             authResource.AuthStatus,
		UpdatedTime:            authResource.UpdatedTime,
		ApprovedPurposeDetails: approvedPurposeDetails,
		OrgID:                  authResource.OrgID,
	}
}
