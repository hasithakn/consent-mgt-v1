package authresource

// Store Access Pattern:
// - AuthResourceStore: Typed interface (same package)
// - ConsentStore: Inline interface (prevents import cycle - consent already imports authresource)
//
// Note: Cannot import consent package as it would create circular dependency:
//   consent/service.go → imports authresource
//   authresource/service.go → would import consent ❌
//
// Solution: Use inline anonymous interface with only needed methods

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/wso2/consent-management-api/internal/authresource/model"
	"github.com/wso2/consent-management-api/internal/consent/validator"
	dbmodel "github.com/wso2/consent-management-api/internal/system/database/model"
	"github.com/wso2/consent-management-api/internal/system/error/serviceerror"
	"github.com/wso2/consent-management-api/internal/system/log"
	"github.com/wso2/consent-management-api/internal/system/stores"
	"github.com/wso2/consent-management-api/internal/system/utils"
)

// AuthResourceServiceInterface defines the contract for auth resource business operations
type AuthResourceServiceInterface interface {
	CreateAuthResource(ctx context.Context, consentID, orgID string, request *model.CreateRequest) (*model.Response, *serviceerror.ServiceError)
	GetAuthResource(ctx context.Context, authID, orgID string) (*model.Response, *serviceerror.ServiceError)
	GetAuthResourcesByConsentID(ctx context.Context, consentID, orgID string) (*model.ListResponse, *serviceerror.ServiceError)
	GetAuthResourcesByUserID(ctx context.Context, userID, orgID string) (*model.ListResponse, *serviceerror.ServiceError)
	UpdateAuthResource(ctx context.Context, authID, orgID string, request *model.UpdateRequest) (*model.Response, *serviceerror.ServiceError)
	DeleteAuthResource(ctx context.Context, authID, orgID string) *serviceerror.ServiceError
	DeleteAuthResourcesByConsentID(ctx context.Context, consentID, orgID string) *serviceerror.ServiceError
	UpdateAllStatusByConsentID(ctx context.Context, consentID, orgID string, status string) *serviceerror.ServiceError
}

// authResourceService implements the AuthResourceServiceInterface
type authResourceService struct {
	stores *stores.StoreRegistry
}

// newAuthResourceService creates a new auth resource service
func newAuthResourceService(registry *stores.StoreRegistry) AuthResourceServiceInterface {
	return &authResourceService{
		stores: registry,
	}
}

// CreateAuthResource creates a new authorization resource for a consent
func (s *authResourceService) CreateAuthResource(
	ctx context.Context,
	consentID, orgID string,
	request *model.CreateRequest,
) (*model.Response, *serviceerror.ServiceError) {
	logger := log.GetLogger().WithContext(ctx)

	logger.Info("Creating authorization resource",
		log.String("consent_id", consentID),
		log.String("org_id", orgID),
		log.String("auth_type", request.AuthType),
		log.String("auth_status", request.AuthStatus))

	// Validate inputs
	if err := s.validateCreateRequest(consentID, orgID, request); err != nil {
		logger.Warn("Auth resource create request validation failed", log.String("error", err.Error))
		return nil, err
	}

	// Generate auth ID
	authID := utils.GenerateUUID()
	logger.Debug("Generated auth ID", log.String("auth_id", authID))

	// Marshal resources to JSON if present
	var resourcesJSON *string
	if request.Resources != nil {
		resourcesBytes, err := json.Marshal(request.Resources)
		if err != nil {
			logger.Error("Failed to marshal authorization resources", log.Error(err), log.String("auth_id", authID))
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

	// Create auth resource and update consent status in a transaction
	store := s.stores.AuthResource.(AuthResourceStore)

	err := s.stores.ExecuteTransaction([]func(tx dbmodel.TxInterface) error{
		func(tx dbmodel.TxInterface) error {
			return store.Create(tx, authResource)
		},
		func(tx dbmodel.TxInterface) error {
			// After creating auth resource, derive consent status from all auth resources
			allAuthResources, err := store.GetByConsentID(ctx, consentID, orgID)
			if err != nil {
				return fmt.Errorf("failed to retrieve auth resources: %w", err)
			}

			// Extract auth statuses
			authStatuses := make([]string, 0, len(allAuthResources))
			for _, ar := range allAuthResources {
				authStatuses = append(authStatuses, ar.AuthStatus)
			}

			// Derive consent status based on all authorization statuses
			// Use validator function to maintain consistency with consent creation logic
			derivedConsentStatus := validator.EvaluateConsentStatusFromAuthStatuses(authStatuses)

			// Get current consent to check if status changed using reflection
			getByIDMethod := reflect.ValueOf(s.stores.Consent).MethodByName("GetByID")
			getResults := getByIDMethod.Call([]reflect.Value{
				reflect.ValueOf(ctx),
				reflect.ValueOf(consentID),
				reflect.ValueOf(orgID),
			})
			if !getResults[1].IsNil() {
				return fmt.Errorf("failed to retrieve consent: %w", getResults[1].Interface().(error))
			}
			currentConsentInterface := getResults[0].Interface()

			// Extract current status using JSON marshal/unmarshal
			type consentWithStatus struct {
				CurrentStatus string `json:"currentStatus"`
			}
			currentConsentBytes, _ := json.Marshal(currentConsentInterface)
			var currentConsent consentWithStatus
			json.Unmarshal(currentConsentBytes, &currentConsent)

			// Check if status actually changed
			if currentConsent.CurrentStatus == derivedConsentStatus {
				// Status hasn't changed, skip update and audit
				return nil
			}

			// Status changed - update consent status using reflection
			updatedTime := utils.GetCurrentTimeMillis()
			updateStatusMethod := reflect.ValueOf(s.stores.Consent).MethodByName("UpdateStatus")
			updateResults := updateStatusMethod.Call([]reflect.Value{
				reflect.ValueOf(tx),
				reflect.ValueOf(consentID),
				reflect.ValueOf(orgID),
				reflect.ValueOf(derivedConsentStatus),
				reflect.ValueOf(updatedTime),
			})
			if !updateResults[0].IsNil() {
				return updateResults[0].Interface().(error)
			}

			// Create status audit record
			auditID := utils.GenerateUUID()
			reason := fmt.Sprintf("Authorization %s created with status %s", authID, request.AuthStatus)
			audit := map[string]interface{}{
				"statusAuditId":  auditID,
				"consentId":      consentID,
				"currentStatus":  derivedConsentStatus,
				"actionTime":     updatedTime,
				"reason":         reason,
				"actionBy":       nil,
				"previousStatus": currentConsent.CurrentStatus,
				"orgId":          orgID,
			}

			// Marshal to JSON then unmarshal to consent.model.ConsentStatusAudit
			auditBytes, _ := json.Marshal(audit)
			// We need to create the right type using reflection
			createMethod := reflect.ValueOf(s.stores.Consent).MethodByName("CreateStatusAudit")
			auditType := createMethod.Type().In(1).Elem() // Get the type of the second parameter (dereferenced)
			consentAuditPtr := reflect.New(auditType)
			json.Unmarshal(auditBytes, consentAuditPtr.Interface())

			// Use reflection to call CreateStatusAudit
			results := createMethod.Call([]reflect.Value{
				reflect.ValueOf(tx),
				consentAuditPtr,
			})
			if !results[0].IsNil() {
				return results[0].Interface().(error)
			}
			return nil
		},
	})
	logger.Debug("Executing transaction for auth resource creation")
	if err != nil {
		logger.Error("Transaction failed for auth resource creation",
			log.Error(err),
			log.String("consent_id", consentID),
		)
		return nil, serviceerror.CustomServiceError(
			serviceerror.DatabaseError,
			fmt.Sprintf("failed to create auth resource: %v", err),
		)
	}

	logger.Info("Auth resource created successfully",
		log.String("auth_id", authResource.AuthID),
		log.String("consent_id", authResource.ConsentID),
		log.String("auth_type", authResource.AuthType),
		log.String("auth_status", authResource.AuthStatus),
	)
	return s.buildResponse(authResource), nil
}

// GetAuthResource retrieves an authorization resource by ID
func (s *authResourceService) GetAuthResource(
	ctx context.Context,
	authID, orgID string,
) (*model.Response, *serviceerror.ServiceError) {
	logger := log.GetLogger().WithContext(ctx)
	logger.Debug("Retrieving auth resource",
		log.String("auth_id", authID),
		log.String("org_id", orgID),
	)

	// Validate inputs
	if err := s.validateAuthIDAndOrgID(authID, orgID); err != nil {
		logger.Warn("Validation failed for get auth resource", log.String("error", err.Error))
		return nil, err
	}

	store := s.stores.AuthResource.(AuthResourceStore)
	authResource, err := store.GetByID(ctx, authID, orgID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			logger.Warn("Auth resource not found",
				log.String("auth_id", authID),
			)
			return nil, serviceerror.CustomServiceError(
				serviceerror.ResourceNotFoundError,
				fmt.Sprintf("auth resource not found: %s", authID),
			)
		}
		logger.Error("Failed to retrieve auth resource",
			log.Error(err),
			log.String("auth_id", authID),
		)
		return nil, serviceerror.CustomServiceError(
			serviceerror.DatabaseError,
			fmt.Sprintf("failed to retrieve auth resource: %v", err),
		)
	}

	logger.Debug("Auth resource retrieved successfully",
		log.String("auth_id", authResource.AuthID),
		log.String("auth_status", authResource.AuthStatus),
	)
	return s.buildResponse(authResource), nil
}

// GetAuthResourcesByConsentID retrieves all authorization resources for a consent
func (s *authResourceService) GetAuthResourcesByConsentID(
	ctx context.Context,
	consentID, orgID string,
) (*model.ListResponse, *serviceerror.ServiceError) {
	logger := log.GetLogger().WithContext(ctx)
	logger.Debug("Retrieving auth resources by consent ID",
		log.String("consent_id", consentID),
		log.String("org_id", orgID),
	)

	// Validate inputs
	if err := s.validateConsentIDAndOrgID(consentID, orgID); err != nil {
		logger.Warn("Validation failed for get auth resources by consent", log.String("error", err.Error))
		return nil, err
	}

	store := s.stores.AuthResource.(AuthResourceStore)
	authResources, err := store.GetByConsentID(ctx, consentID, orgID)
	if err != nil {
		logger.Error("Failed to fetch auth resources by consent ID",
			log.Error(err),
			log.String("consent_id", consentID),
		)
		return nil, serviceerror.CustomServiceError(
			serviceerror.DatabaseError,
			fmt.Sprintf("failed to fetch auth resources: %v", err),
		)
	}

	// Initialize as empty slice to ensure JSON serialization returns [] instead of null
	responses := make([]model.Response, 0, len(authResources))
	for _, ar := range authResources {
		responses = append(responses, *s.buildResponse(&ar))
	}

	logger.Debug("Auth resources retrieved successfully",
		log.String("consent_id", consentID),
		log.Int("count", len(authResources)),
	)
	return &model.ListResponse{
		Data: responses,
	}, nil
}

// GetAuthResourcesByUserID retrieves all authorization resources for a user
func (s *authResourceService) GetAuthResourcesByUserID(
	ctx context.Context,
	userID, orgID string,
) (*model.ListResponse, *serviceerror.ServiceError) {
	logger := log.GetLogger().WithContext(ctx)
	logger.Debug("Retrieving auth resources by user ID",
		log.String("user_id", userID),
		log.String("org_id", orgID),
	)

	// Validate inputs
	if userID == "" {
		logger.Warn("User ID is required")
		return nil, serviceerror.CustomServiceError(
			serviceerror.InvalidRequestError,
			"user ID is required",
		)
	}
	if err := s.validateOrgID(orgID); err != nil {
		logger.Warn("Validation failed for get auth resources by user", log.String("error", err.Error))
		return nil, err
	}

	store := s.stores.AuthResource.(AuthResourceStore)
	authResources, err := store.GetByUserID(ctx, userID, orgID)
	if err != nil {
		logger.Error("Failed to fetch auth resources by user ID",
			log.Error(err),
			log.String("user_id", userID),
		)
		return nil, serviceerror.CustomServiceError(
			serviceerror.DatabaseError,
			fmt.Sprintf("failed to fetch auth resources: %v", err),
		)
	}

	// Initialize as empty slice to ensure JSON serialization returns [] instead of null
	responses := make([]model.Response, 0, len(authResources))
	for _, ar := range authResources {
		responses = append(responses, *s.buildResponse(&ar))
	}

	logger.Debug("Auth resources retrieved successfully",
		log.String("user_id", userID),
		log.Int("count", len(authResources)),
	)
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
	logger := log.GetLogger().WithContext(ctx)
	logger.Info("Updating auth resource",
		log.String("auth_id", authID),
		log.String("org_id", orgID),
		log.String("new_auth_status", request.AuthStatus),
	)

	// Validate inputs
	if err := s.validateAuthIDAndOrgID(authID, orgID); err != nil {
		logger.Warn("Validation failed for update auth resource",
			log.String("error", err.Error),
			log.String("auth_id", authID),
		)
		return nil, err
	}

	// Get existing auth resource
	store := s.stores.AuthResource.(AuthResourceStore)
	existingAuthResource, err := store.GetByID(ctx, authID, orgID)
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

	statusChanged := false
	if request.AuthStatus != "" {
		updatedAuthResource.AuthStatus = request.AuthStatus
		statusChanged = (existingAuthResource.AuthStatus != request.AuthStatus)
		if statusChanged {
			logger.Debug("Auth status changed",
				log.String("auth_id", authID),
				log.String("old_status", existingAuthResource.AuthStatus),
				log.String("new_status", request.AuthStatus),
			)
		}
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

	// Update auth resource and potentially consent status in transaction
	transactionSteps := []func(tx dbmodel.TxInterface) error{
		func(tx dbmodel.TxInterface) error {
			return store.Update(tx, &updatedAuthResource)
		},
	}

	// If auth status changed, update consent status accordingly
	if statusChanged {
		transactionSteps = append(transactionSteps, func(tx dbmodel.TxInterface) error {
			// Get all auth resources for this consent
			allAuthResources, err := store.GetByConsentID(ctx, existingAuthResource.ConsentID, orgID)
			if err != nil {
				return fmt.Errorf("failed to retrieve auth resources: %w", err)
			}

			// Extract auth statuses (including the updated one)
			authStatuses := make([]string, 0, len(allAuthResources))
			for _, ar := range allAuthResources {
				if ar.AuthID == authID {
					// Use the new status for this auth resource
					authStatuses = append(authStatuses, updatedAuthResource.AuthStatus)
				} else {
					authStatuses = append(authStatuses, ar.AuthStatus)
				}
			}

			// Derive consent status
			derivedConsentStatus := validator.EvaluateConsentStatusFromAuthStatuses(authStatuses)
			logger.Debug("Derived consent status from auth statuses",
				log.String("consent_id", existingAuthResource.ConsentID),
				log.String("derived_status", derivedConsentStatus),
				log.Int("auth_count", len(authStatuses)),
			)

			// Get current consent to check if status changed using reflection
			getByIDMethod := reflect.ValueOf(s.stores.Consent).MethodByName("GetByID")
			getResults := getByIDMethod.Call([]reflect.Value{
				reflect.ValueOf(ctx),
				reflect.ValueOf(existingAuthResource.ConsentID),
				reflect.ValueOf(orgID),
			})
			if !getResults[1].IsNil() {
				return fmt.Errorf("failed to retrieve consent: %w", getResults[1].Interface().(error))
			}
			currentConsentInterface := getResults[0].Interface()

			// Extract current status using JSON marshal/unmarshal
			type consentWithStatus struct {
				CurrentStatus string `json:"currentStatus"`
				OrgID         string `json:"orgId"`
			}
			currentConsentBytes, _ := json.Marshal(currentConsentInterface)
			var currentConsent consentWithStatus
			json.Unmarshal(currentConsentBytes, &currentConsent)

			// Only update if consent status actually changed
			if currentConsent.CurrentStatus != derivedConsentStatus {
				updatedTime := utils.GetCurrentTimeMillis()

				// Update consent status using reflection
				updateStatusMethod := reflect.ValueOf(s.stores.Consent).MethodByName("UpdateStatus")
				updateResults := updateStatusMethod.Call([]reflect.Value{
					reflect.ValueOf(tx),
					reflect.ValueOf(existingAuthResource.ConsentID),
					reflect.ValueOf(orgID),
					reflect.ValueOf(derivedConsentStatus),
					reflect.ValueOf(updatedTime),
				})
				if !updateResults[0].IsNil() {
					return updateResults[0].Interface().(error)
				}

				// Verify the consent exists by doing a SELECT after UPDATE
				verifyMethod := reflect.ValueOf(s.stores.Consent).MethodByName("GetByID")
				verifyResults := verifyMethod.Call([]reflect.Value{
					reflect.ValueOf(ctx),
					reflect.ValueOf(existingAuthResource.ConsentID),
					reflect.ValueOf(orgID),
				})
				if !verifyResults[1].IsNil() {
					return verifyResults[1].Interface().(error)
				}

				// Create status audit record
				auditID := utils.GenerateUUID()
				reason := fmt.Sprintf("Authorization %s status updated from %s to %s", authID, existingAuthResource.AuthStatus, updatedAuthResource.AuthStatus)
				audit := map[string]interface{}{
					"statusAuditId":  auditID,
					"consentId":      existingAuthResource.ConsentID,
					"currentStatus":  derivedConsentStatus,
					"actionTime":     updatedTime,
					"reason":         reason,
					"actionBy":       nil,
					"previousStatus": currentConsent.CurrentStatus,
					"orgId":          orgID,
				}

				// Marshal to JSON then unmarshal to consent.model.ConsentStatusAudit
				auditBytes, _ := json.Marshal(audit)
				// Create the right type using reflection
				createMethod := reflect.ValueOf(s.stores.Consent).MethodByName("CreateStatusAudit")
				auditType := createMethod.Type().In(1).Elem() // Get the type of the second parameter (dereferenced)
				consentAuditPtr := reflect.New(auditType)
				json.Unmarshal(auditBytes, consentAuditPtr.Interface())

				// Use reflection to call CreateStatusAudit
				results := createMethod.Call([]reflect.Value{
					reflect.ValueOf(tx),
					consentAuditPtr,
				})
				if !results[0].IsNil() {
					return results[0].Interface().(error)
				}
				return nil
			}
			return nil
		})
	}

	logger.Debug("Executing transaction for auth resource update")
	err = s.stores.ExecuteTransaction(transactionSteps)
	if err != nil {
		logger.Error("Transaction failed for auth resource update",
			log.Error(err),
			log.String("auth_id", authID),
		)
		return nil, serviceerror.CustomServiceError(
			serviceerror.DatabaseError,
			fmt.Sprintf("failed to update auth resource: %v", err),
		)
	}

	logger.Info("Auth resource updated successfully",
		log.String("auth_id", updatedAuthResource.AuthID),
		log.String("consent_id", updatedAuthResource.ConsentID),
		log.String("auth_status", updatedAuthResource.AuthStatus),
		log.Bool("status_changed", statusChanged),
	)
	return s.buildResponse(&updatedAuthResource), nil
}

// DeleteAuthResource deletes an authorization resource
func (s *authResourceService) DeleteAuthResource(
	ctx context.Context,
	authID, orgID string,
) *serviceerror.ServiceError {
	logger := log.GetLogger().WithContext(ctx)
	logger.Info("Deleting auth resource",
		log.String("auth_id", authID),
		log.String("org_id", orgID),
	)

	// Validate inputs
	if err := s.validateAuthIDAndOrgID(authID, orgID); err != nil {
		logger.Warn("Validation failed for delete auth resource", log.String("error", err.Error))
		return err
	}

	// Get existing auth resource to retrieve consent ID
	store := s.stores.AuthResource.(AuthResourceStore)
	existingAuthResource, err := store.GetByID(ctx, authID, orgID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return serviceerror.CustomServiceError(
				serviceerror.ResourceNotFoundError,
				fmt.Sprintf("auth resource not found: %s", authID),
			)
		}
		return serviceerror.CustomServiceError(
			serviceerror.DatabaseError,
			fmt.Sprintf("failed to retrieve auth resource: %v", err),
		)
	}

	// Delete auth resource and update consent status in transaction
	err = s.stores.ExecuteTransaction([]func(tx dbmodel.TxInterface) error{
		func(tx dbmodel.TxInterface) error {
			return store.Delete(tx, authID, orgID)
		},
		func(tx dbmodel.TxInterface) error {
			// Get remaining auth resources for this consent
			allAuthResources, err := store.GetByConsentID(ctx, existingAuthResource.ConsentID, orgID)
			if err != nil {
				return fmt.Errorf("failed to retrieve auth resources: %w", err)
			}

			// Filter out the deleted auth resource
			authStatuses := make([]string, 0, len(allAuthResources))
			for _, ar := range allAuthResources {
				if ar.AuthID != authID {
					authStatuses = append(authStatuses, ar.AuthStatus)
				}
			}

			// Derive consent status from remaining auth resources
			derivedConsentStatus := validator.EvaluateConsentStatusFromAuthStatuses(authStatuses)
			logger.Debug("Derived consent status after deletion",
				log.String("consent_id", existingAuthResource.ConsentID),
				log.String("derived_status", derivedConsentStatus),
				log.Int("remaining_auth_count", len(authStatuses)),
			)

			// Get current consent to check if status changed using reflection
			getByIDMethod := reflect.ValueOf(s.stores.Consent).MethodByName("GetByID")
			getResults := getByIDMethod.Call([]reflect.Value{
				reflect.ValueOf(ctx),
				reflect.ValueOf(existingAuthResource.ConsentID),
				reflect.ValueOf(orgID),
			})
			if !getResults[1].IsNil() {
				return fmt.Errorf("failed to retrieve consent: %w", getResults[1].Interface().(error))
			}
			currentConsentInterface := getResults[0].Interface()

			// Extract current status using JSON marshal/unmarshal
			type consentWithStatus struct {
				CurrentStatus string `json:"currentStatus"`
			}
			currentConsentBytes, _ := json.Marshal(currentConsentInterface)
			var currentConsent consentWithStatus
			json.Unmarshal(currentConsentBytes, &currentConsent)

			// Only update if consent status actually changed
			if currentConsent.CurrentStatus != derivedConsentStatus {
				updatedTime := utils.GetCurrentTimeMillis()

				// Update consent status using reflection
				updateStatusMethod := reflect.ValueOf(s.stores.Consent).MethodByName("UpdateStatus")
				updateResults := updateStatusMethod.Call([]reflect.Value{
					reflect.ValueOf(tx),
					reflect.ValueOf(existingAuthResource.ConsentID),
					reflect.ValueOf(orgID),
					reflect.ValueOf(derivedConsentStatus),
					reflect.ValueOf(updatedTime),
				})
				if !updateResults[0].IsNil() {
					return updateResults[0].Interface().(error)
				}

				// Create status audit record
				auditID := utils.GenerateUUID()
				reason := fmt.Sprintf("Authorization %s deleted with status %s", authID, existingAuthResource.AuthStatus)
				audit := map[string]interface{}{
					"statusAuditId":  auditID,
					"consentId":      existingAuthResource.ConsentID,
					"currentStatus":  derivedConsentStatus,
					"actionTime":     updatedTime,
					"reason":         reason,
					"actionBy":       nil,
					"previousStatus": currentConsent.CurrentStatus,
					"orgId":          orgID,
				}

				// Marshal to JSON then unmarshal to consent.model.ConsentStatusAudit
				auditBytes, _ := json.Marshal(audit)
				// Create the right type using reflection
				createMethod := reflect.ValueOf(s.stores.Consent).MethodByName("CreateStatusAudit")
				auditType := createMethod.Type().In(1).Elem() // Get the type of the second parameter (dereferenced)
				consentAuditPtr := reflect.New(auditType)
				json.Unmarshal(auditBytes, consentAuditPtr.Interface())

				// Use reflection to call CreateStatusAudit
				results := createMethod.Call([]reflect.Value{
					reflect.ValueOf(tx),
					consentAuditPtr,
				})
				if !results[0].IsNil() {
					return results[0].Interface().(error)
				}
				return nil
			}
			return nil
		},
	})
	logger.Debug("Executing transaction for auth resource deletion")
	if err != nil {
		logger.Error("Transaction failed for auth resource deletion",
			log.Error(err),
			log.String("auth_id", authID),
		)
		return serviceerror.CustomServiceError(
			serviceerror.DatabaseError,
			fmt.Sprintf("failed to delete auth resource: %v", err),
		)
	}

	logger.Info("Auth resource deleted successfully",
		log.String("auth_id", authID),
		log.String("consent_id", existingAuthResource.ConsentID),
	)
	return nil
}

// DeleteAuthResourcesByConsentID deletes all authorization resources for a consent
func (s *authResourceService) DeleteAuthResourcesByConsentID(
	ctx context.Context,
	consentID, orgID string,
) *serviceerror.ServiceError {
	logger := log.GetLogger().WithContext(ctx)
	logger.Info("Deleting all auth resources for consent",
		log.String("consent_id", consentID),
		log.String("org_id", orgID),
	)

	// Validate inputs
	if err := s.validateConsentIDAndOrgID(consentID, orgID); err != nil {
		logger.Warn("Validation failed for delete auth resources by consent", log.String("error", err.Error))
		return err
	}

	// Delete all auth resources for the consent
	store := s.stores.AuthResource.(AuthResourceStore)
	logger.Debug("Executing transaction for auth resources deletion")
	err := s.stores.ExecuteTransaction([]func(tx dbmodel.TxInterface) error{
		func(tx dbmodel.TxInterface) error {
			return store.DeleteByConsentID(tx, consentID, orgID)
		},
	})
	if err != nil {
		logger.Error("Transaction failed for auth resources deletion",
			log.Error(err),
			log.String("consent_id", consentID),
		)
		return serviceerror.CustomServiceError(
			serviceerror.DatabaseError,
			fmt.Sprintf("failed to delete auth resources: %v", err),
		)
	}

	logger.Info("Auth resources deleted successfully for consent",
		log.String("consent_id", consentID),
	)
	return nil
}

// UpdateAllStatusByConsentID updates status for all auth resources of a consent
func (s *authResourceService) UpdateAllStatusByConsentID(
	ctx context.Context,
	consentID, orgID string,
	status string,
) *serviceerror.ServiceError {
	logger := log.GetLogger().WithContext(ctx)
	logger.Info("Updating all auth resource statuses for consent",
		log.String("consent_id", consentID),
		log.String("org_id", orgID),
		log.String("new_status", status),
	)

	// Validate inputs
	if err := s.validateConsentIDAndOrgID(consentID, orgID); err != nil {
		logger.Warn("Validation failed for update auth statuses", log.String("error", err.Error))
		return err
	}
	if status == "" {
		logger.Warn("Status is required")
		return serviceerror.CustomServiceError(
			serviceerror.InvalidRequestError,
			"status is required",
		)
	}

	// Update all statuses
	store := s.stores.AuthResource.(AuthResourceStore)
	updatedTime := utils.GetCurrentTimeMillis()
	logger.Debug("Executing transaction for auth statuses update")
	err := s.stores.ExecuteTransaction([]func(tx dbmodel.TxInterface) error{
		func(tx dbmodel.TxInterface) error {
			return store.UpdateAllStatusByConsentID(tx, consentID, orgID, status, updatedTime)
		},
	})
	if err != nil {
		logger.Error("Transaction failed for auth statuses update",
			log.Error(err),
			log.String("consent_id", consentID),
		)
		return serviceerror.CustomServiceError(
			serviceerror.DatabaseError,
			fmt.Sprintf("failed to update auth resource statuses: %v", err),
		)
	}

	logger.Info("Auth resource statuses updated successfully",
		log.String("consent_id", consentID),
		log.String("status", status),
	)
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
		AuthType:    authResource.AuthType,
		UserID:      authResource.UserID,
		AuthStatus:  authResource.AuthStatus,
		UpdatedTime: authResource.UpdatedTime,
		Resources:   resources,
	}
}
