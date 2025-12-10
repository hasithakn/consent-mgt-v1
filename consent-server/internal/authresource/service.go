package authresource

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/wso2/consent-management-api/internal/authresource/model"
	"github.com/wso2/consent-management-api/internal/system/error/serviceerror"
	"github.com/wso2/consent-management-api/internal/utils"
)

// AuthResourceServiceInterface defines the contract for auth resource business operations
type AuthResourceServiceInterface interface {
	CreateAuthResource(ctx context.Context, consentID, orgID string, request *model.CreateRequest) (*model.Response, *serviceerror.ServiceError)
	GetAuthResource(ctx context.Context, authID, orgID string) (*model.Response, *serviceerror.ServiceError)
	GetAuthResourcesByConsentID(ctx context.Context, consentID, orgID string) (*model.ListResponse, *serviceerror.ServiceError)
	GetAuthResourcesByUserID(ctx context.Context, userID, orgID string) (*model.ListResponse, *serviceerror.ServiceError)
	UpdateAuthResource(ctx context.Context, authID, orgID string, request *model.UpdateRequest) (*model.Response, *serviceerror.ServiceError)
	UpdateAuthResourceStatus(ctx context.Context, authID, orgID string, status string) (*model.Response, *serviceerror.ServiceError)
	DeleteAuthResource(ctx context.Context, authID, orgID string) *serviceerror.ServiceError
	DeleteAuthResourcesByConsentID(ctx context.Context, consentID, orgID string) *serviceerror.ServiceError
	UpdateAllStatusByConsentID(ctx context.Context, consentID, orgID string, status string) *serviceerror.ServiceError
}

// authResourceService implements AuthResourceServiceInterface
type authResourceService struct {
	store authResourceStore
}

// newAuthResourceService creates a new auth resource service
func newAuthResourceService(store authResourceStore) AuthResourceServiceInterface {
	return &authResourceService{
		store: store,
	}
}

// CreateAuthResource creates a new authorization resource for a consent
func (s *authResourceService) CreateAuthResource(
	ctx context.Context,
	consentID, orgID string,
	request *model.CreateRequest,
) (*model.Response, *serviceerror.ServiceError) {
	// Validate inputs
	if err := s.validateCreateRequest(consentID, orgID, request); err != nil {
		return nil, err
	}

	// Generate auth ID
	authID := utils.GenerateUUID()

	// Marshal resources to JSON if present
	var resourcesJSON *string
	if request.Resources != nil {
		resourcesBytes, err := json.Marshal(request.Resources)
		if err != nil {
			return nil, serviceerror.CustomServiceError(
				serviceerror.ValidationError,
				fmt.Sprintf("failed to marshal resources: %v", err),
			)
		}
		resourcesStr := string(resourcesBytes)
		resourcesJSON = &resourcesStr
	}

	// Build auth resource model
	authResource := &model.AuthResource{
		AuthID:      authID,
		ConsentID:   consentID,
		AuthType:    request.AuthType,
		UserID:      request.UserID,
		AuthStatus:  request.AuthStatus,
		UpdatedTime: utils.GetCurrentTimeMillis(),
		Resources:   resourcesJSON,
		OrgID:       orgID,
	}

	// Create auth resource
	if err := s.store.Create(ctx, authResource); err != nil {
		return nil, serviceerror.CustomServiceError(
			serviceerror.DatabaseError,
			fmt.Sprintf("failed to create auth resource: %v", err),
		)
	}

	return s.buildResponse(authResource), nil
}

// GetAuthResource retrieves an authorization resource by ID
func (s *authResourceService) GetAuthResource(
	ctx context.Context,
	authID, orgID string,
) (*model.Response, *serviceerror.ServiceError) {
	// Validate inputs
	if err := s.validateAuthIDAndOrgID(authID, orgID); err != nil {
		return nil, err
	}

	authResource, err := s.store.GetByID(ctx, authID, orgID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, serviceerror.CustomServiceError(
				serviceerror.ResourceNotFoundError,
				fmt.Sprintf("auth resource not found: %s", authID),
			)
		}
		return nil, serviceerror.CustomServiceError(
			serviceerror.DatabaseError,
			fmt.Sprintf("failed to retrieve auth resource: %v", err),
		)
	}

	return s.buildResponse(authResource), nil
}

// GetAuthResourcesByConsentID retrieves all authorization resources for a consent
func (s *authResourceService) GetAuthResourcesByConsentID(
	ctx context.Context,
	consentID, orgID string,
) (*model.ListResponse, *serviceerror.ServiceError) {
	// Validate inputs
	if err := s.validateConsentIDAndOrgID(consentID, orgID); err != nil {
		return nil, err
	}

	authResources, err := s.store.GetByConsentID(ctx, consentID, orgID)
	if err != nil {
		return nil, serviceerror.CustomServiceError(
			serviceerror.DatabaseError,
			fmt.Sprintf("failed to retrieve auth resources: %v", err),
		)
	}

	var responses []model.Response
	for _, ar := range authResources {
		responses = append(responses, *s.buildResponse(&ar))
	}

	return &model.ListResponse{
		Data: responses,
	}, nil
}

// GetAuthResourcesByUserID retrieves all authorization resources for a user
func (s *authResourceService) GetAuthResourcesByUserID(
	ctx context.Context,
	userID, orgID string,
) (*model.ListResponse, *serviceerror.ServiceError) {
	// Validate inputs
	if userID == "" {
		return nil, serviceerror.CustomServiceError(
			serviceerror.InvalidRequestError,
			"user ID is required",
		)
	}
	if err := s.validateOrgID(orgID); err != nil {
		return nil, err
	}

	authResources, err := s.store.GetByUserID(ctx, userID, orgID)
	if err != nil {
		return nil, serviceerror.CustomServiceError(
			serviceerror.DatabaseError,
			fmt.Sprintf("failed to retrieve auth resources: %v", err),
		)
	}

	var responses []model.Response
	for _, ar := range authResources {
		responses = append(responses, *s.buildResponse(&ar))
	}

	return &model.ListResponse{
		Data: responses,
	}, nil
}

// UpdateAuthResource updates an existing authorization resource
func (s *authResourceService) UpdateAuthResource(
	ctx context.Context,
	authID, orgID string,
	request *model.UpdateRequest,
) (*model.Response, *serviceerror.ServiceError) {
	// Validate inputs
	if err := s.validateAuthIDAndOrgID(authID, orgID); err != nil {
		return nil, err
	}

	// Get existing auth resource
	existingAuthResource, err := s.store.GetByID(ctx, authID, orgID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, serviceerror.CustomServiceError(
				serviceerror.ResourceNotFoundError,
				fmt.Sprintf("auth resource not found: %s", authID),
			)
		}
		return nil, serviceerror.CustomServiceError(
			serviceerror.DatabaseError,
			fmt.Sprintf("failed to retrieve auth resource: %v", err),
		)
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

	if request.Resources != nil {
		resourcesBytes, err := json.Marshal(request.Resources)
		if err != nil {
			return nil, serviceerror.CustomServiceError(
				serviceerror.ValidationError,
				fmt.Sprintf("failed to marshal resources: %v", err),
			)
		}
		resourcesStr := string(resourcesBytes)
		updatedAuthResource.Resources = &resourcesStr
	}

	// Update auth resource
	if err := s.store.Update(ctx, &updatedAuthResource); err != nil {
		return nil, serviceerror.CustomServiceError(
			serviceerror.DatabaseError,
			fmt.Sprintf("failed to update auth resource: %v", err),
		)
	}

	return s.buildResponse(&updatedAuthResource), nil
}

// UpdateAuthResourceStatus updates the status of an authorization resource
func (s *authResourceService) UpdateAuthResourceStatus(
	ctx context.Context,
	authID, orgID string,
	status string,
) (*model.Response, *serviceerror.ServiceError) {
	// Validate inputs
	if err := s.validateAuthIDAndOrgID(authID, orgID); err != nil {
		return nil, err
	}
	if status == "" {
		return nil, serviceerror.CustomServiceError(
			serviceerror.InvalidRequestError,
			"status is required",
		)
	}

	// Get existing auth resource to return updated response
	existingAuthResource, err := s.store.GetByID(ctx, authID, orgID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, serviceerror.CustomServiceError(
				serviceerror.ResourceNotFoundError,
				fmt.Sprintf("auth resource not found: %s", authID),
			)
		}
		return nil, serviceerror.CustomServiceError(
			serviceerror.DatabaseError,
			fmt.Sprintf("failed to retrieve auth resource: %v", err),
		)
	}

	// Update status
	updatedTime := utils.GetCurrentTimeMillis()
	if err := s.store.UpdateStatus(ctx, authID, orgID, status, updatedTime); err != nil {
		return nil, serviceerror.CustomServiceError(
			serviceerror.DatabaseError,
			fmt.Sprintf("failed to update auth resource status: %v", err),
		)
	}

	// Update the model for response
	existingAuthResource.AuthStatus = status
	existingAuthResource.UpdatedTime = updatedTime

	return s.buildResponse(existingAuthResource), nil
}

// DeleteAuthResource deletes an authorization resource
func (s *authResourceService) DeleteAuthResource(
	ctx context.Context,
	authID, orgID string,
) *serviceerror.ServiceError {
	// Validate inputs
	if err := s.validateAuthIDAndOrgID(authID, orgID); err != nil {
		return err
	}

	// Check if auth resource exists
	exists, err := s.store.Exists(ctx, authID, orgID)
	if err != nil {
		return serviceerror.CustomServiceError(
			serviceerror.DatabaseError,
			fmt.Sprintf("failed to check auth resource existence: %v", err),
		)
	}
	if !exists {
		return serviceerror.CustomServiceError(
			serviceerror.ResourceNotFoundError,
			fmt.Sprintf("auth resource not found: %s", authID),
		)
	}

	// Delete auth resource
	if err := s.store.Delete(ctx, authID, orgID); err != nil {
		return serviceerror.CustomServiceError(
			serviceerror.DatabaseError,
			fmt.Sprintf("failed to delete auth resource: %v", err),
		)
	}

	return nil
}

// DeleteAuthResourcesByConsentID deletes all authorization resources for a consent
func (s *authResourceService) DeleteAuthResourcesByConsentID(
	ctx context.Context,
	consentID, orgID string,
) *serviceerror.ServiceError {
	// Validate inputs
	if err := s.validateConsentIDAndOrgID(consentID, orgID); err != nil {
		return err
	}

	// Delete all auth resources for the consent
	if err := s.store.DeleteByConsentID(ctx, consentID, orgID); err != nil {
		return serviceerror.CustomServiceError(
			serviceerror.DatabaseError,
			fmt.Sprintf("failed to delete auth resources: %v", err),
		)
	}

	return nil
}

// UpdateAllStatusByConsentID updates status for all auth resources of a consent
func (s *authResourceService) UpdateAllStatusByConsentID(
	ctx context.Context,
	consentID, orgID string,
	status string,
) *serviceerror.ServiceError {
	// Validate inputs
	if err := s.validateConsentIDAndOrgID(consentID, orgID); err != nil {
		return err
	}
	if status == "" {
		return serviceerror.CustomServiceError(
			serviceerror.InvalidRequestError,
			"status is required",
		)
	}

	// Update all statuses
	updatedTime := utils.GetCurrentTimeMillis()
	if err := s.store.UpdateAllStatusByConsentID(ctx, consentID, orgID, status, updatedTime); err != nil {
		return serviceerror.CustomServiceError(
			serviceerror.DatabaseError,
			fmt.Sprintf("failed to update auth resource statuses: %v", err),
		)
	}

	return nil
}

// Helper methods for validation

func (s *authResourceService) validateCreateRequest(consentID, orgID string, request *model.CreateRequest) *serviceerror.ServiceError {
	if err := s.validateConsentIDAndOrgID(consentID, orgID); err != nil {
		return err
	}
	if request == nil {
		return serviceerror.CustomServiceError(
			serviceerror.InvalidRequestError,
			"request body is required",
		)
	}
	if request.AuthType == "" {
		return serviceerror.CustomServiceError(
			serviceerror.InvalidRequestError,
			"auth type is required",
		)
	}
	if request.AuthStatus == "" {
		return serviceerror.CustomServiceError(
			serviceerror.InvalidRequestError,
			"auth status is required",
		)
	}
	return nil
}

func (s *authResourceService) validateAuthIDAndOrgID(authID, orgID string) *serviceerror.ServiceError {
	if authID == "" {
		return serviceerror.CustomServiceError(
			serviceerror.InvalidRequestError,
			"auth ID is required",
		)
	}
	if len(authID) > 255 {
		return serviceerror.CustomServiceError(
			serviceerror.InvalidRequestError,
			"auth ID too long: maximum 255 characters",
		)
	}
	return s.validateOrgID(orgID)
}

func (s *authResourceService) validateConsentIDAndOrgID(consentID, orgID string) *serviceerror.ServiceError {
	if consentID == "" {
		return serviceerror.CustomServiceError(
			serviceerror.InvalidRequestError,
			"consent ID is required",
		)
	}
	if len(consentID) > 255 {
		return serviceerror.CustomServiceError(
			serviceerror.InvalidRequestError,
			"consent ID too long: maximum 255 characters",
		)
	}
	return s.validateOrgID(orgID)
}

func (s *authResourceService) validateOrgID(orgID string) *serviceerror.ServiceError {
	if orgID == "" {
		return serviceerror.CustomServiceError(
			serviceerror.InvalidRequestError,
			"organization ID is required",
		)
	}
	if len(orgID) > 255 {
		return serviceerror.CustomServiceError(
			serviceerror.InvalidRequestError,
			"organization ID too long: maximum 255 characters",
		)
	}
	return nil
}

func (s *authResourceService) buildResponse(authResource *model.AuthResource) *model.Response {
	var resources interface{}
	if authResource.Resources != nil && *authResource.Resources != "" {
		// Try to unmarshal resources
		json.Unmarshal([]byte(*authResource.Resources), &resources)
	}

	return &model.Response{
		AuthID:      authResource.AuthID,
		ConsentID:   authResource.ConsentID,
		AuthType:    authResource.AuthType,
		UserID:      authResource.UserID,
		AuthStatus:  authResource.AuthStatus,
		UpdatedTime: authResource.UpdatedTime,
		Resources:   resources,
		OrgID:       authResource.OrgID,
	}
}
