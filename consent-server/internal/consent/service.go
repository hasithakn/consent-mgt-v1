package consent

import (
	"context"
	"encoding/json"
	"fmt"

	authmodel "github.com/wso2/consent-management-api/internal/authresource/model"
	"github.com/wso2/consent-management-api/internal/consent/model"
	"github.com/wso2/consent-management-api/internal/consent/validator"
	"github.com/wso2/consent-management-api/internal/system/config"
	dbmodel "github.com/wso2/consent-management-api/internal/system/database/model"
	"github.com/wso2/consent-management-api/internal/system/error/serviceerror"
	"github.com/wso2/consent-management-api/internal/system/log"
	"github.com/wso2/consent-management-api/internal/system/stores"
	"github.com/wso2/consent-management-api/internal/system/utils"
)

// ConsentService defines the exported service interface
type ConsentService interface {
	CreateConsent(ctx context.Context, req model.ConsentAPIRequest, clientID, orgID string) (*model.ConsentResponse, *serviceerror.ServiceError)
	GetConsent(ctx context.Context, consentID, orgID string) (*model.ConsentResponse, *serviceerror.ServiceError)
	ListConsents(ctx context.Context, orgID string, limit, offset int) ([]model.ConsentResponse, int, *serviceerror.ServiceError)
	SearchConsents(ctx context.Context, filters model.ConsentSearchFilters) ([]model.ConsentResponse, int, *serviceerror.ServiceError)
	SearchConsentsDetailed(ctx context.Context, filters model.ConsentSearchFilters) (*model.ConsentDetailSearchResponse, *serviceerror.ServiceError)
	UpdateConsent(ctx context.Context, req model.ConsentAPIUpdateRequest, orgID, consentID string) (*model.ConsentResponse, *serviceerror.ServiceError)
	RevokeConsent(ctx context.Context, consentID, orgID string, req model.ConsentRevokeRequest) (*model.ConsentRevokeResponse, *serviceerror.ServiceError)
	ValidateConsent(ctx context.Context, req model.ValidateRequest, orgID string) (*model.ValidateResponse, *serviceerror.ServiceError)
	SearchConsentsByAttribute(ctx context.Context, key, value, orgID string) (*model.ConsentAttributeSearchResponse, *serviceerror.ServiceError)
}

// consentService implements the ConsentService interface
type consentService struct {
	stores *stores.StoreRegistry
}

// newConsentService creates a new consent service
func newConsentService(registry *stores.StoreRegistry) ConsentService {
	return &consentService{
		stores: registry,
	}
}

// CreateConsent creates a new consent with all related entities in a single transaction
func (consentService *consentService) CreateConsent(ctx context.Context, req model.ConsentAPIRequest, clientID, orgID string) (*model.ConsentResponse, *serviceerror.ServiceError) {
	logger := log.GetLogger().WithContext(ctx)

	logger.Info("Creating consent",
		log.String("client_id", clientID),
		log.String("org_id", orgID),
		log.String("consent_type", req.Type))

	// Validate request
	if err := utils.ValidateOrgID(orgID); err != nil {
		logger.Warn("Invalid organization ID", log.Error(err), log.String("org_id", orgID))
		return nil, serviceerror.CustomServiceError(serviceerror.ValidationError, err.Error())
	}
	if err := utils.ValidateClientID(clientID); err != nil {
		logger.Warn("Invalid client ID", log.Error(err), log.String("client_id", clientID))
		return nil, serviceerror.CustomServiceError(serviceerror.ValidationError, err.Error())
	}
	if err := validator.ValidateConsentCreateRequest(req, clientID, orgID); err != nil {
		logger.Warn("Consent create request validation failed", log.Error(err))
		return nil, serviceerror.CustomServiceError(serviceerror.ValidationError, err.Error())
	}

	logger.Debug("Request validation successful")

	logger.Debug("Request validation successful")

	// Convert API request to internal format
	createReq, err := req.ToConsentCreateRequest()
	if err != nil {
		logger.Error("Failed to convert API request to internal format", log.Error(err))
		return nil, serviceerror.CustomServiceError(serviceerror.ValidationError, err.Error())
	}

	// HANDLE PURPOSE GROUPS (validate and resolve all purposes)
	var resolvedPurposeGroups []model.ConsentPurposeGroupCreateRequest
	if len(createReq.PurposeGroups) > 0 {
		var err error
		resolvedPurposeGroups, err = consentService.validatePurposeGroups(ctx, createReq.PurposeGroups, clientID, orgID)
		if err != nil {
			logger.Error("Purpose group validation failed", log.Error(err))
			return nil, serviceerror.CustomServiceError(serviceerror.ValidationError, err.Error())
		}
		logger.Debug("Purpose groups validated and resolved",
			log.Int("group_count", len(resolvedPurposeGroups)))
	}

	// Extract auth statuses
	authStatuses := make([]string, 0, len(createReq.AuthResources))
	for _, ar := range createReq.AuthResources {
		authStatuses = append(authStatuses, ar.AuthStatus)
	}

	// Derive consent status from authorization states
	consentStatus := validator.EvaluateConsentStatusFromAuthStatuses(authStatuses)
	logger.Debug("Consent status derived from authorizations",
		log.String("consent_status", consentStatus),
		log.Int("auth_count", len(authStatuses)))

	// Generate IDs and timestamp
	consentID := utils.GenerateUUID()
	currentTime := utils.GetCurrentTimeMillis()

	logger.Debug("Generated consent ID", log.String("consent_id", consentID))

	// Create consent entity
	consent := &model.Consent{
		ConsentID:                  consentID,
		CreatedTime:                currentTime,
		UpdatedTime:                currentTime,
		ClientID:                   clientID,
		ConsentType:                createReq.ConsentType,
		CurrentStatus:              consentStatus,
		ConsentFrequency:           createReq.ConsentFrequency,
		ValidityTime:               createReq.ValidityTime,
		RecurringIndicator:         createReq.RecurringIndicator,
		DataAccessValidityDuration: createReq.DataAccessValidityDuration,
		OrgID:                      orgID,
	}

	// Get stores from registry
	consentStore := consentService.stores.Consent
	authResourceStore := consentService.stores.AuthResource

	// Build list of transactional operations
	queries := []func(tx dbmodel.TxInterface) error{
		// Create consent
		func(tx dbmodel.TxInterface) error {
			return consentStore.Create(tx, consent)
		},
	}

	// Add attributes if provided
	if len(createReq.Attributes) > 0 {
		logger.Debug("Adding consent attributes", log.Int("attribute_count", len(createReq.Attributes)))
		attributes := make([]model.ConsentAttribute, 0, len(createReq.Attributes))
		for key, value := range createReq.Attributes {
			attr := model.ConsentAttribute{
				ConsentID: consentID,
				AttKey:    key,
				AttValue:  value,
				OrgID:     orgID,
			}
			attributes = append(attributes, attr)
		}
		queries = append(queries, func(tx dbmodel.TxInterface) error {
			return consentStore.CreateAttributes(tx, attributes)
		})
	}

	// Create audit record
	auditID := utils.GenerateUUID()
	actionBy := clientID // Client ID as the action initiator
	reason := "Initial consent creation"
	audit := &model.ConsentStatusAudit{
		StatusAuditID:  auditID,
		ConsentID:      consentID,
		CurrentStatus:  consent.CurrentStatus,
		ActionTime:     currentTime,
		Reason:         &reason,   // Pointer to string value
		ActionBy:       &actionBy, // Pointer to string value
		PreviousStatus: nil,       // nil = no previous status (first creation)
		OrgID:          orgID,
	}
	queries = append(queries, func(tx dbmodel.TxInterface) error {
		return consentStore.CreateStatusAudit(tx, audit)
	})

	// Add authorization resources if provided
	if len(req.Authorizations) > 0 {
		logger.Debug("Adding authorization resources", log.Int("authorization_count", len(req.Authorizations)))
	}
	for _, authReq := range req.Authorizations {
		authID := utils.GenerateUUID()

		// Marshal resources to JSON if present
		var resourcesJSON *string
		if authReq.Resources != nil {
			resourcesBytes, err := json.Marshal(authReq.Resources)
			if err != nil {
				logger.Error("Failed to marshal authorization resources",
					log.Error(err),
					log.String("auth_id", authID))
				return nil, serviceerror.CustomServiceError(serviceerror.ValidationError, fmt.Sprintf("failed to marshal resources: %v", err))
			}
			resourcesStr := string(resourcesBytes)
			resourcesJSON = &resourcesStr
		}

		// Convert to internal format
		var userIDPtr *string
		if authReq.UserID != "" {
			userIDPtr = &authReq.UserID
		}

		authResource := &authmodel.AuthResource{
			AuthID:      authID,
			ConsentID:   consentID,
			AuthType:    authReq.Type,
			UserID:      userIDPtr,
			AuthStatus:  authReq.Status,
			UpdatedTime: currentTime,
			Resources:   resourcesJSON,
			OrgID:       orgID,
		}

		queries = append(queries, func(tx dbmodel.TxInterface) error {
			return authResourceStore.Create(tx, authResource)
		})
	}

	// Add purpose group and approval records
	for _, pg := range resolvedPurposeGroups {
		// Link consent to purpose group
		groupID := pg.GroupID
		queries = append(queries, func(tx dbmodel.TxInterface) error {
			return consentStore.CreatePurposeGroupConsent(tx, consentID, groupID, orgID)
		})

		// Create approval records for each purpose in the group
		for _, purpose := range pg.Purposes {
			approval := &model.ConsentPurposeApprovalRecord{
				ConsentID:      consentID,
				GroupID:        groupID,
				PurposeID:      purpose.PurposeID,
				IsUserApproved: purpose.IsUserApproved,
				Value:          purpose.Value,
				OrgID:          orgID,
			}

			queries = append(queries, func(tx dbmodel.TxInterface) error {
				return consentStore.CreatePurposeApproval(tx, approval)
			})
		}
	}

	// Execute all operations in a single transaction
	logger.Debug("Executing transaction", log.Int("operation_count", len(queries)))
	if err := consentService.stores.ExecuteTransaction(queries); err != nil {
		logger.Error("Failed to create consent in transaction",
			log.Error(err),
			log.String("consent_id", consentID))
		return nil, serviceerror.CustomServiceError(serviceerror.DatabaseError, fmt.Sprintf("failed to create consent: %v", err))
	}

	logger.Info("Consent created successfully", log.String("consent_id", consentID))

	// TODO : check consent expireation and handle accordingly.

	// Retrieve related data after creation
	logger.Debug("Retrieving related data for response")
	authResources, _ := authResourceStore.GetByConsentID(ctx, consentID, orgID)
	attributes, _ := consentService.stores.Consent.GetAttributesByConsentID(ctx, consentID, orgID)

	// Convert attributes slice to map[string]string
	attributesMap := make(map[string]string)
	for _, a := range attributes {
		attributesMap[a.AttKey] = a.AttValue
	}

	// Use the generic method to resolve purpose groups with all purposes
	purposeGroups, err := consentService.getResolvedConsentPurposesWithGroups(ctx, consentID, orgID)
	if err != nil {
		logger.Error("Failed to resolve purpose groups for response",
			log.String("consent_id", consentID),
			log.Error(err))
		// Return with empty purpose groups on error rather than failing the whole response
		purposeGroups = []model.ConsentPurposeGroupItem{}
	}

	// Build complete response using the resolved purpose groups data
	response := buildConsentResponse(consent, purposeGroups, attributesMap, authResources)

	logger.Info("Consent creation completed",
		log.String("consent_id", consentID),
		log.String("status", consent.CurrentStatus),
		log.Int("auth_resources", len(authResources)),
		log.Int("purpose_groups", len(purposeGroups)),
		log.Int("attributes", len(attributesMap)))

	return response, nil
}

// GetConsent retrieves a consent by ID with all related data
func (consentService *consentService) GetConsent(ctx context.Context, consentID, orgID string) (*model.ConsentResponse, *serviceerror.ServiceError) {
	logger := log.GetLogger().WithContext(ctx)
	logger.Debug("Retrieving consent",
		log.String("consent_id", consentID),
		log.String("org_id", orgID),
	)

	// Get stores
	consentStore := consentService.stores.Consent
	authResourceStore := consentService.stores.AuthResource

	// Get consent
	consent, err := consentStore.GetByID(ctx, consentID, orgID)
	if err != nil {
		logger.Error("Failed to retrieve consent",
			log.Error(err),
			log.String("consent_id", consentID),
		)
		return nil, serviceerror.CustomServiceError(serviceerror.DatabaseError, err.Error())
	}
	if consent == nil {
		logger.Warn("Consent not found", log.String("consent_id", consentID))
		return nil, serviceerror.CustomServiceError(serviceerror.ResourceNotFoundError, fmt.Sprintf("Consent with ID '%s' not found", consentID))
	}

	// TODO : check consent expireation and handle accordingly.

	// Retrieve all related data
	attributes, _ := consentStore.GetAttributesByConsentID(ctx, consentID, orgID)
	authResources, _ := authResourceStore.GetByConsentID(ctx, consentID, orgID)

	// Convert attributes slice to map
	attributesMap := make(map[string]string)
	for _, a := range attributes {
		attributesMap[a.AttKey] = a.AttValue
	}

	// Resolve purpose groups with all purposes
	purposeGroups, err := consentService.getResolvedConsentPurposesWithGroups(ctx, consentID, orgID)
	if err != nil {
		logger.Error("Failed to resolve purpose groups", log.Error(err))
		return nil, serviceerror.CustomServiceError(serviceerror.DatabaseError, fmt.Sprintf("failed to resolve purpose groups: %v", err))
	}

	// Build complete response with all related data
	response := buildConsentResponse(consent, purposeGroups, attributesMap, authResources)

	logger.Debug("Consent retrieved successfully",
		log.String("consent_id", consentID),
		log.String("status", consent.CurrentStatus),
		log.Int("auth_resources", len(authResources)),
		log.Int("purpose_groups", len(response.PurposeGroups)),
	)
	return response, nil
}

// ListConsents retrieves paginated list of consents
func (consentService *consentService) ListConsents(ctx context.Context, orgID string, limit, offset int) ([]model.ConsentResponse, int, *serviceerror.ServiceError) {
	logger := log.GetLogger().WithContext(ctx)
	logger.Debug("Listing consents",
		log.String("org_id", orgID),
		log.Int("limit", limit),
		log.Int("offset", offset),
	)

	if limit <= 0 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}

	store := consentService.stores.Consent
	consents, total, err := store.List(ctx, orgID, limit, offset)
	if err != nil {
		logger.Error("Failed to list consents",
			log.Error(err),
			log.String("org_id", orgID),
		)
		return nil, 0, serviceerror.CustomServiceError(serviceerror.DatabaseError, err.Error())
	}

	// Convert to responses
	responses := make([]model.ConsentResponse, 0, len(consents))
	for _, c := range consents {
		responses = append(responses, model.ConsentResponse{
			ConsentID:                  c.ConsentID,
			CreatedTime:                c.CreatedTime,
			UpdatedTime:                c.UpdatedTime,
			ClientID:                   c.ClientID,
			ConsentType:                c.ConsentType,
			CurrentStatus:              c.CurrentStatus,
			ConsentFrequency:           c.ConsentFrequency,
			ValidityTime:               c.ValidityTime,
			RecurringIndicator:         c.RecurringIndicator,
			DataAccessValidityDuration: c.DataAccessValidityDuration,
			OrgID:                      c.OrgID,
		})
	}

	logger.Info("Consents listed successfully",
		log.Int("count", len(responses)),
		log.Int("total", total))

	return responses, total, nil
}

// SearchConsents retrieves consents based on search filters with pagination
func (consentService *consentService) SearchConsents(ctx context.Context, filters model.ConsentSearchFilters) ([]model.ConsentResponse, int, *serviceerror.ServiceError) {
	logger := log.GetLogger().WithContext(ctx)
	logger.Debug("Searching consents",
		log.String("org_id", filters.OrgID),
		log.Int("client_ids_count", len(filters.ClientIDs)),
		log.Int("user_ids_count", len(filters.UserIDs)),
		log.Int("statuses_count", len(filters.ConsentStatuses)),
		log.Int("limit", filters.Limit),
	)

	// Validate pagination
	if filters.Limit <= 0 {
		filters.Limit = 10
	}
	if filters.Offset < 0 {
		filters.Offset = 0
	}

	store := consentService.stores.Consent
	consents, total, err := store.Search(ctx, filters)
	if err != nil {
		logger.Error("Failed to search consents",
			log.Error(err),
			log.String("org_id", filters.OrgID),
		)
		return nil, 0, serviceerror.CustomServiceError(serviceerror.DatabaseError, err.Error())
	}

	// Convert to responses
	responses := make([]model.ConsentResponse, 0, len(consents))
	for _, c := range consents {
		responses = append(responses, model.ConsentResponse{
			ConsentID:                  c.ConsentID,
			CreatedTime:                c.CreatedTime,
			UpdatedTime:                c.UpdatedTime,
			ClientID:                   c.ClientID,
			ConsentType:                c.ConsentType,
			CurrentStatus:              c.CurrentStatus,
			ConsentFrequency:           c.ConsentFrequency,
			ValidityTime:               c.ValidityTime,
			RecurringIndicator:         c.RecurringIndicator,
			DataAccessValidityDuration: c.DataAccessValidityDuration,
			OrgID:                      c.OrgID,
		})
	}

	logger.Info("Consents searched successfully",
		log.Int("count", len(responses)),
		log.Int("total", total))

	return responses, total, nil
}

// SearchConsentsDetailed retrieves consents with nested authorization resources, purposes, and attributes
func (consentService *consentService) SearchConsentsDetailed(ctx context.Context, filters model.ConsentSearchFilters) (*model.ConsentDetailSearchResponse, *serviceerror.ServiceError) {
	logger := log.GetLogger().WithContext(ctx)
	logger.Info("Searching consents with detailed data",
		log.String("org_id", filters.OrgID),
		log.Int("client_ids_count", len(filters.ClientIDs)),
		log.Int("user_ids_count", len(filters.UserIDs)),
		log.Int("statuses_count", len(filters.ConsentStatuses)),
		log.Int("limit", filters.Limit))

	// Validate pagination
	if filters.Limit <= 0 {
		filters.Limit = 10
	}
	if filters.Offset < 0 {
		filters.Offset = 0
	}

	// Step 1: Search consents
	consentStore := consentService.stores.Consent
	consents, total, err := consentStore.Search(ctx, filters)
	if err != nil {
		logger.Error("Failed to search consents", log.Error(err))
		return nil, serviceerror.CustomServiceError(serviceerror.DatabaseError, err.Error())
	}

	if len(consents) == 0 {
		return &model.ConsentDetailSearchResponse{
			Data: []model.ConsentDetailResponse{},
			Metadata: model.ConsentSearchMetadata{
				Total:  0,
				Limit:  filters.Limit,
				Offset: filters.Offset,
				Count:  0,
			},
		}, nil
	}

	// Step 2: Extract consent IDs
	consentIDs := make([]string, len(consents))
	for i, c := range consents {
		consentIDs[i] = c.ConsentID
	}

	// Step 3: Batch fetch related data
	authResourceStore := consentService.stores.AuthResource

	authResources, err := authResourceStore.GetByConsentIDs(ctx, consentIDs, filters.OrgID)
	if err != nil {
		logger.Error("Failed to get authorization resources", log.Error(err))
		return nil, serviceerror.CustomServiceError(serviceerror.DatabaseError, err.Error())
	}

	attributesByConsent, err := consentStore.GetAttributesByConsentIDs(ctx, consentIDs, filters.OrgID)
	if err != nil {
		logger.Error("Failed to get consent attributes", log.Error(err))
		return nil, serviceerror.CustomServiceError(serviceerror.DatabaseError, err.Error())
	}

	// Step 4: Group auth resources by consent ID
	authsByConsent := make(map[string][]authmodel.AuthResource)
	for _, auth := range authResources {
		authsByConsent[auth.ConsentID] = append(authsByConsent[auth.ConsentID], auth)
	}

	// Step 5: Assemble detailed responses
	detailedResponses := make([]model.ConsentDetailResponse, 0, len(consents))
	for _, consent := range consents {
		// Build authorizations - initialize as empty slice
		authorizations := make([]model.AuthorizationDetail, 0)
		for _, auth := range authsByConsent[consent.ConsentID] {
			var resources interface{}
			if auth.Resources != nil && *auth.Resources != "" {
				_ = json.Unmarshal([]byte(*auth.Resources), &resources)
			}

			userID := ""
			if auth.UserID != nil {
				userID = *auth.UserID
			}

			authorizations = append(authorizations, model.AuthorizationDetail{
				ID:          auth.AuthID,
				UserID:      userID,
				Type:        auth.AuthType,
				Status:      auth.AuthStatus,
				UpdatedTime: auth.UpdatedTime,
				Resources:   resources,
			})
		}

		// Resolve purpose groups for this consent
		purposeGroups, err := consentService.getResolvedConsentPurposesWithGroups(ctx, consent.ConsentID, filters.OrgID)
		if err != nil {
			logger.Warn("Failed to resolve purpose groups for consent",
				log.String("consent_id", consent.ConsentID),
				log.Error(err))
			// Continue with empty purpose groups rather than failing
			purposeGroups = []model.ConsentPurposeGroupItem{}
		}

		// Get attributes (already grouped by consent ID)
		attributes := attributesByConsent[consent.ConsentID]
		if attributes == nil {
			attributes = make(map[string]string)
		}

		// Dereference pointer fields for response
		frequency := 0
		if consent.ConsentFrequency != nil {
			frequency = *consent.ConsentFrequency
		}
		validityTime := int64(0)
		if consent.ValidityTime != nil {
			validityTime = *consent.ValidityTime
		}
		recurringIndicator := false
		if consent.RecurringIndicator != nil {
			recurringIndicator = *consent.RecurringIndicator
		}
		dataAccessValidityDuration := int64(0)
		if consent.DataAccessValidityDuration != nil {
			dataAccessValidityDuration = *consent.DataAccessValidityDuration
		}

		detailedResponses = append(detailedResponses, model.ConsentDetailResponse{
			ID:                         consent.ConsentID,
			PurposeGroups:              purposeGroups,
			CreatedTime:                consent.CreatedTime,
			UpdatedTime:                consent.UpdatedTime,
			ClientID:                   consent.ClientID,
			Type:                       consent.ConsentType,
			Status:                     consent.CurrentStatus,
			Frequency:                  frequency,
			ValidityTime:               validityTime,
			RecurringIndicator:         recurringIndicator,
			DataAccessValidityDuration: dataAccessValidityDuration,
			Attributes:                 attributes,
			Authorizations:             authorizations,
		})
	}

	logger.Info("Consents searched with details successfully",
		log.Int("count", len(detailedResponses)),
		log.Int("total", total))

	return &model.ConsentDetailSearchResponse{
		Data: detailedResponses,
		Metadata: model.ConsentSearchMetadata{
			Total:  total,
			Limit:  filters.Limit,
			Offset: filters.Offset,
			Count:  len(detailedResponses),
		},
	}, nil
}

// UpdateConsent updates an existing consent
func (consentService *consentService) UpdateConsent(ctx context.Context, req model.ConsentAPIUpdateRequest, orgID, consentID string) (*model.ConsentResponse, *serviceerror.ServiceError) {
	logger := log.GetLogger().WithContext(ctx)
	logger.Info("Updating consent",
		log.String("consent_id", consentID),
		log.String("org_id", orgID))

	// Get stores
	authResourceStore := consentService.stores.AuthResource
	consentStore := consentService.stores.Consent

	// Validate request
	if err := utils.ValidateOrgID(orgID); err != nil {
		logger.Warn("Invalid organization ID", log.Error(err), log.String("org_id", orgID))
		return nil, serviceerror.CustomServiceError(serviceerror.ValidationError, err.Error())
	}
	if err := validator.ValidateConsentUpdateRequest(req); err != nil {
		logger.Warn("Consent update request validation failed", log.Error(err))
		return nil, serviceerror.CustomServiceError(serviceerror.ValidationError, err.Error())
	}

	// Convert to internal format
	updateReq, convertErr := req.ToConsentUpdateRequest()
	if convertErr != nil {
		logger.Warn("Failed to convert update request", log.Error(convertErr))
		return nil, serviceerror.CustomServiceError(serviceerror.ValidationError, convertErr.Error())
	}

	logger.Debug("Request validation successful")

	// Check if consent exists
	existing, err := consentStore.GetByID(ctx, consentID, orgID)
	if err != nil {
		logger.Error("Failed to retrieve consent", log.Error(err), log.String("consent_id", consentID))
		return nil, serviceerror.CustomServiceError(serviceerror.DatabaseError, err.Error())
	}
	if existing == nil {
		logger.Warn("Consent not found", log.String("consent_id", consentID))
		return nil, serviceerror.CustomServiceError(serviceerror.ResourceNotFoundError, fmt.Sprintf("Consent with ID '%s' not found", consentID))
	}

	currentTime := utils.GetCurrentTimeMillis()
	previousStatus := existing.CurrentStatus

	if req.DataAccessValidityDuration != nil {
		// Validate that it's non-negative
		if *req.DataAccessValidityDuration < 0 {
			logger.Warn("Invalid data access validity duration", log.Any("duration", *req.DataAccessValidityDuration))
			return nil, serviceerror.CustomServiceError(serviceerror.ResourceNotFoundError, "dataAccessValidityDuration must be non-negative")
		}
		updateReq.DataAccessValidityDuration = req.DataAccessValidityDuration
	}

	// Derive new consent status from authorization states if auth resources are being updated
	var newStatus string
	var statusChanged bool
	if updateReq.AuthResources != nil {

		// Extract auth statuses
		authStatuses := make([]string, 0, len(updateReq.AuthResources))
		for _, ar := range updateReq.AuthResources {
			authStatuses = append(authStatuses, ar.AuthStatus)
		}

		newStatus = validator.EvaluateConsentStatusFromAuthStatuses(authStatuses)
		statusChanged = (newStatus != previousStatus)
		if statusChanged {
			logger.Debug("Consent status changed",
				log.String("previous_status", previousStatus),
				log.String("new_status", newStatus))
		}
	} else {
		newStatus = existing.CurrentStatus
		statusChanged = false
	}

	// Update consent fields
	consent := &model.Consent{
		ConsentID:                  consentID,
		UpdatedTime:                currentTime,
		CurrentStatus:              newStatus,
		ConsentType:                updateReq.ConsentType,
		ConsentFrequency:           updateReq.ConsentFrequency,
		ValidityTime:               updateReq.ValidityTime,
		RecurringIndicator:         updateReq.RecurringIndicator,
		DataAccessValidityDuration: updateReq.DataAccessValidityDuration,
		OrgID:                      orgID,
	}

	// Build transactional operations
	queries := []func(tx dbmodel.TxInterface) error{
		func(tx dbmodel.TxInterface) error {
			return consentStore.Update(tx, consent)
		},
	}

	if statusChanged {

		queries = []func(tx dbmodel.TxInterface) error{
			func(tx dbmodel.TxInterface) error {
				return consentStore.UpdateStatus(tx, consentID, orgID, newStatus, currentTime)
			},
		}

		// Create status audit if status changed
		auditID := utils.GenerateUUID()
		actionBy := existing.ClientID // Use client ID as action initiator
		reason := "Consent status updated based on authorization states during consent update"
		audit := &model.ConsentStatusAudit{
			StatusAuditID:  auditID,
			ConsentID:      consentID,
			CurrentStatus:  newStatus,
			ActionTime:     currentTime,
			Reason:         &reason,
			ActionBy:       &actionBy,
			PreviousStatus: &previousStatus,
			OrgID:          orgID,
		}

		queries = append(queries, func(tx dbmodel.TxInterface) error {
			return consentStore.CreateStatusAudit(tx, audit)
		})

	}

	// Update attributes - delete old and create new if provided
	if updateReq.Attributes != nil {
		// Delete existing attributes
		queries = append(queries, func(tx dbmodel.TxInterface) error {
			return consentStore.DeleteAttributesByConsentID(tx, consentID, orgID)
		})

		// Create new attributes if not empty
		if len(updateReq.Attributes) > 0 {
			attributes := make([]model.ConsentAttribute, 0, len(updateReq.Attributes))
			for key, value := range updateReq.Attributes {
				attr := model.ConsentAttribute{
					ConsentID: consentID,
					AttKey:    key,
					AttValue:  value,
					OrgID:     orgID,
				}
				attributes = append(attributes, attr)
			}

			queries = append(queries, func(tx dbmodel.TxInterface) error {
				return consentStore.CreateAttributes(tx, attributes)
			})
		}
	}

	// Update authorization resources if provided
	if updateReq.AuthResources != nil {

		// Delete existing auth resources
		queries = append(queries, func(tx dbmodel.TxInterface) error {
			return authResourceStore.DeleteByConsentID(tx, consentID, orgID)
		})

		// Create new auth resources if not empty
		if len(updateReq.AuthResources) > 0 {
			for _, authReq := range updateReq.AuthResources {
				authID := utils.GenerateUUID()

				// Marshal resources to JSON if present
				var resourcesJSON *string
				if authReq.Resources != nil {
					resourcesBytes, err := json.Marshal(authReq.Resources)
					if err != nil {
						return nil, serviceerror.CustomServiceError(serviceerror.ValidationError, fmt.Sprintf("failed to marshal resources: %v", err))
					}
					resourcesStr := string(resourcesBytes)
					resourcesJSON = &resourcesStr
				}

				authResource := &authmodel.AuthResource{
					AuthID:      authID,
					ConsentID:   consentID,
					AuthType:    authReq.AuthType,
					UserID:      authReq.UserID,
					AuthStatus:  authReq.AuthStatus,
					UpdatedTime: currentTime,
					Resources:   resourcesJSON,
					OrgID:       orgID,
				}

				queries = append(queries, func(tx dbmodel.TxInterface) error {
					return authResourceStore.Create(tx, authResource)
				})
			}
		}
	}

	// HANDLE PURPOSE GROUPS UPDATE (validate and resolve all purposes if provided)
	var resolvedPurposeGroups []model.ConsentPurposeGroupCreateRequest
	if updateReq.PurposeGroups != nil {
		var err error
		resolvedPurposeGroups, err = consentService.validatePurposeGroups(ctx, updateReq.PurposeGroups, existing.ClientID, orgID)
		if err != nil {
			logger.Error("Purpose group validation failed", log.Error(err))
			return nil, serviceerror.CustomServiceError(serviceerror.ValidationError, err.Error())
		}
		logger.Debug("Purpose groups validated and resolved",
			log.Int("group_count", len(resolvedPurposeGroups)))

		// Delete existing purpose group mappings and approvals
		queries = append(queries, func(tx dbmodel.TxInterface) error {
			return consentStore.DeletePurposeGroupsByConsentID(tx, consentID, orgID)
		})

		queries = append(queries, func(tx dbmodel.TxInterface) error {
			return consentStore.DeletePurposeApprovalsByConsentID(tx, consentID, orgID)
		})

		// Add new purpose group and approval records
		for _, pg := range resolvedPurposeGroups {
			// Link consent to purpose group
			groupID := pg.GroupID
			queries = append(queries, func(tx dbmodel.TxInterface) error {
				return consentStore.CreatePurposeGroupConsent(tx, consentID, groupID, orgID)
			})

			// Create approval records for each purpose in the group
			for _, purpose := range pg.Purposes {
				approval := &model.ConsentPurposeApprovalRecord{
					ConsentID:      consentID,
					GroupID:        groupID,
					PurposeID:      purpose.PurposeID,
					IsUserApproved: purpose.IsUserApproved,
					Value:          purpose.Value,
					OrgID:          orgID,
				}

				queries = append(queries, func(tx dbmodel.TxInterface) error {
					return consentStore.CreatePurposeApproval(tx, approval)
				})
			}
		}
	}

	// Execute transaction
	logger.Debug("Executing update transaction", log.Int("operation_count", len(queries)))
	if err := consentService.stores.ExecuteTransaction(queries); err != nil {
		logger.Error("Failed to update consent in transaction",
			log.Error(err),
			log.String("consent_id", consentID))
		return nil, serviceerror.CustomServiceError(serviceerror.DatabaseError, err.Error())
	}

	// Get updated consent
	logger.Debug("Retrieving updated consent data")
	updated, getErr := consentStore.GetByID(ctx, consentID, orgID)
	if getErr != nil {
		logger.Error("Failed to retrieve updated consent", log.Error(getErr))
		return nil, serviceerror.CustomServiceError(serviceerror.DatabaseError, getErr.Error())
	}

	authResources, _ := authResourceStore.GetByConsentID(ctx, consentID, orgID)
	attributes, _ := consentStore.GetAttributesByConsentID(ctx, consentID, orgID)

	// Convert attributes slice to map[string]string
	attributesMap := make(map[string]string)
	for _, a := range attributes {
		attributesMap[a.AttKey] = a.AttValue
	}

	// Resolve purpose groups with all purposes
	purposeGroups, err := consentService.getResolvedConsentPurposesWithGroups(ctx, consentID, orgID)
	if err != nil {
		logger.Error("Failed to resolve purpose groups", log.Error(err))
		return nil, serviceerror.CustomServiceError(serviceerror.DatabaseError, fmt.Sprintf("failed to resolve purpose groups: %v", err))
	}

	// Build complete response
	response := buildConsentResponse(updated, purposeGroups, attributesMap, authResources)

	logger.Info("Consent updated successfully",
		log.String("consent_id", consentID),
		log.String("status", updated.CurrentStatus),
		log.Int("auth_resources", len(authResources)),
		log.Int("purpose_groups", len(response.PurposeGroups)),
		log.Int("attributes", len(attributesMap)))

	return response, nil
}

// RevokeConsent updates consent status and creates audit entry
func (consentService *consentService) RevokeConsent(ctx context.Context, consentID, orgID string, req model.ConsentRevokeRequest) (*model.ConsentRevokeResponse, *serviceerror.ServiceError) {
	logger := log.GetLogger().WithContext(ctx)
	logger.Info("Revoking consent",
		log.String("consent_id", consentID),
		log.String("org_id", orgID),
		log.String("action_by", req.ActionBy))

	// Validate action by
	if req.ActionBy == "" {
		logger.Warn("Validation failed: ActionBy is required")
		return nil, serviceerror.CustomServiceError(serviceerror.ValidationError, "ActionBy is required")
	}

	logger.Debug("Request validation successful")

	revokedStatusName := config.Get().Consent.GetRevokedConsentStatus()

	// Check if consent exists
	store := consentService.stores.Consent
	existing, err := store.GetByID(ctx, consentID, orgID)
	if err != nil {
		logger.Error("Failed to retrieve consent", log.Error(err), log.String("consent_id", consentID))
		return nil, serviceerror.CustomServiceError(serviceerror.DatabaseError, err.Error())
	}
	if existing == nil {
		logger.Warn("Consent not found", log.String("consent_id", consentID))
		return nil, serviceerror.CustomServiceError(serviceerror.ResourceNotFoundError, fmt.Sprintf("Consent with ID '%s' not found", consentID))
	}

	// Check if consent is already revoked
	if existing.CurrentStatus == string(revokedStatusName) {
		logger.Warn("Consent is already revoked",
			log.String("consent_id", consentID),
			log.String("status", existing.CurrentStatus))
		return nil, serviceerror.CustomServiceError(serviceerror.ConflictError, fmt.Sprintf("Consent with ID '%s' is already revoked", consentID))
	}

	currentTime := utils.GetCurrentTimeMillis()

	// Create audit entry
	auditID := utils.GenerateUUID()
	reason := req.RevocationReason
	audit := &model.ConsentStatusAudit{
		StatusAuditID:  auditID,
		ConsentID:      consentID,
		CurrentStatus:  string(revokedStatusName),
		ActionTime:     currentTime,
		Reason:         &reason,
		ActionBy:       &req.ActionBy,
		PreviousStatus: &existing.CurrentStatus,
		OrgID:          orgID,
	}

	// Get auth resource store for cascading status update
	authResourceStore := consentService.stores.AuthResource

	// Execute transaction - update consent status, all auth resource statuses, and create audit
	logger.Debug("Executing revocation transaction")
	err = consentService.stores.ExecuteTransaction([]func(tx dbmodel.TxInterface) error{
		func(tx dbmodel.TxInterface) error {
			return store.UpdateStatus(tx, consentID, orgID, string(revokedStatusName), currentTime)
		},
		func(tx dbmodel.TxInterface) error {
			// Update all authorization statuses to SYS_REVOKED when consent is revoked
			return authResourceStore.UpdateAllStatusByConsentID(tx, consentID, orgID, "SYS_REVOKED", currentTime)
		},
		func(tx dbmodel.TxInterface) error {
			return store.CreateStatusAudit(tx, audit)
		},
	})
	if err != nil {
		logger.Error("Failed to revoke consent in transaction",
			log.Error(err),
			log.String("consent_id", consentID))
		return nil, serviceerror.CustomServiceError(serviceerror.DatabaseError, err.Error())
	}

	logger.Info("Consent revoked successfully",
		log.String("consent_id", consentID),
		log.String("previous_status", existing.CurrentStatus),
		log.String("new_status", string(revokedStatusName)))

	// Build and return response
	response := &model.ConsentRevokeResponse{
		ActionTime:       currentTime / 1000, // Convert milliseconds to seconds
		ActionBy:         req.ActionBy,
		RevocationReason: req.RevocationReason,
	}

	return response, nil
}

// ValidateConsent validates a consent for data access
func (consentService *consentService) ValidateConsent(ctx context.Context, req model.ValidateRequest, orgID string) (*model.ValidateResponse, *serviceerror.ServiceError) {
	logger := log.GetLogger().WithContext(ctx)
	logger.Info("Validating consent",
		log.String("consent_id", req.ConsentID),
		log.String("org_id", orgID))

	// Initialize response with invalid state
	response := &model.ValidateResponse{
		IsValid: false,
	}

	// Validate request
	if req.ConsentID == "" {
		logger.Warn("Validation failed: ConsentID is required")
		return nil, serviceerror.CustomServiceError(serviceerror.ValidationError, "ConsentID is required")
	}

	logger.Debug("Request validation successful")

	// Get consent
	consentStore := consentService.stores.Consent
	consent, err := consentStore.GetByID(ctx, req.ConsentID, orgID)
	if err != nil {
		logger.Error("Failed to retrieve consent", log.Error(err), log.String("consent_id", req.ConsentID))
		// return nil, serviceerror.CustomServiceError(serviceerror.DatabaseError, err.Error())
		response.ErrorCode = 500
		response.ErrorMessage = "database_error"
		response.ErrorDescription = "Database error while retrieving consent"
	}
	if consent == nil {
		logger.Warn("Consent not found", log.String("consent_id", req.ConsentID))
		// return nil, serviceerror.CustomServiceError(serviceerror.ValidationError, fmt.Sprintf("Consent with ID '%s' not found", req.ConsentID))
		response.ErrorCode = 404
		response.ErrorMessage = "not_found"
		response.ErrorDescription = "Consent not found"
	} else {
		// Check if consent is expired and update status accordingly (only if consent exists)
		expiredStatusName := string(config.Get().Consent.GetExpiredConsentStatus())
		if consent.ValidityTime != nil && validator.IsConsentExpired(*consent.ValidityTime) {
			// Update consent status to expired if not already expired
			if consent.CurrentStatus != expiredStatusName {
				if err := consentService.expireConsent(ctx, consent, orgID); err != nil {
					// Log error but continue with validation
					// The consent object is already updated in-memory by expireConsent
				} else {
					// Re-fetch consent after expiring to get latest state
					if updatedConsent, fetchErr := consentStore.GetByID(ctx, req.ConsentID, orgID); fetchErr == nil && updatedConsent != nil {
						consent = updatedConsent
					}
					// If re-fetch fails, continue with in-memory consent object
				}
			}
		}
	}

	// Check consent status - only active consents are valid
	activeStatusName := string(config.Get().Consent.GetActiveConsentStatus())
	if consent != nil && consent.CurrentStatus != activeStatusName && response.ErrorCode == 0 {
		response.ErrorCode = 401
		response.ErrorMessage = "invalid_consent_status"
		response.ErrorDescription = fmt.Sprintf("Consent status is '%s', expected '%s'", consent.CurrentStatus, activeStatusName)
	}

	// If no errors, mark as valid
	if response.ErrorCode == 0 {
		response.IsValid = true
		logger.Info("Consent validation successful",
			log.String("consent_id", req.ConsentID),
			log.Bool("is_valid", true))
	} else {
		logger.Warn("Consent validation failed",
			log.String("consent_id", req.ConsentID),
			log.Bool("is_valid", false),
			log.Int("error_code", response.ErrorCode),
			log.String("error_message", response.ErrorMessage))
	}

	// Retrieve related data for consent information (only if consent exists)
	if consent != nil {
		authResourceStore := consentService.stores.AuthResource

		attributes, _ := consentStore.GetAttributesByConsentID(ctx, consent.ConsentID, orgID)
		authResources, _ := authResourceStore.GetByConsentID(ctx, consent.ConsentID, orgID)

		// Convert attributes slice to map
		attributesMap := make(map[string]string)
		for _, a := range attributes {
			attributesMap[a.AttKey] = a.AttValue
		}

		// Resolve purpose groups with all purposes
		purposeGroups, err := consentService.getResolvedConsentPurposesWithGroups(ctx, consent.ConsentID, orgID)
		if err != nil {
			logger.Error("Failed to resolve purpose groups", log.Error(err))
			// Continue with validation, but set error in response
			response.ErrorCode = 500
			response.ErrorMessage = "response_build_error"
			response.ErrorDescription = "Failed to resolve purpose groups"
		} else {
			// Build complete consent response
			consentResponse := buildConsentResponse(consent, purposeGroups, attributesMap, authResources)
			// Convert to ValidateConsentAPIResponse with enriched purpose details
			response.ConsentInformation = consentService.EnrichedValidateConsentAPIResponse(ctx, consentResponse, orgID)
		}
	}

	return response, nil
}

// expireConsent updates consent and all related auth resources to expired status
func (consentService *consentService) expireConsent(ctx context.Context, consent *model.Consent, orgID string) error {
	logger := log.GetLogger().WithContext(ctx)
	logger.Debug("Expiring consent",
		log.String("consent_id", consent.ConsentID),
		log.String("org_id", orgID))

	expiredStatusName := string(config.Get().Consent.GetExpiredConsentStatus())
	currentTime := utils.GetCurrentTimeMillis()

	// Create audit entry
	auditID := utils.GenerateUUID()
	reason := "Consent expired based on validityTime"
	actionBy := "SYSTEM"
	previousStatus := consent.CurrentStatus
	audit := &model.ConsentStatusAudit{
		StatusAuditID:  auditID,
		ConsentID:      consent.ConsentID,
		CurrentStatus:  expiredStatusName,
		ActionTime:     currentTime,
		Reason:         &reason,
		ActionBy:       &actionBy,
		PreviousStatus: &previousStatus,
		OrgID:          orgID,
	}

	// Get stores for cascading status update
	consentStore := consentService.stores.Consent
	authResourceStore := consentService.stores.AuthResource

	// Execute transaction - update consent status, all auth resource statuses, and create audit
	err := consentService.stores.ExecuteTransaction([]func(tx dbmodel.TxInterface) error{
		func(tx dbmodel.TxInterface) error {
			return consentStore.UpdateStatus(tx, consent.ConsentID, orgID, expiredStatusName, currentTime)
		},
		func(tx dbmodel.TxInterface) error {
			// Update all authorization statuses to SYS_EXPIRED when consent expires
			return authResourceStore.UpdateAllStatusByConsentID(tx, consent.ConsentID, orgID, "SYS_EXPIRED", currentTime)
		},
		func(tx dbmodel.TxInterface) error {
			return consentStore.CreateStatusAudit(tx, audit)
		},
	})
	if err != nil {
		logger.Error("Failed to expire consent in transaction",
			log.Error(err),
			log.String("consent_id", consent.ConsentID))
		return err
	}

	// Update local consent object
	consent.CurrentStatus = expiredStatusName
	consent.UpdatedTime = currentTime

	logger.Debug("Consent expired successfully",
		log.String("consent_id", consent.ConsentID),
		log.String("new_status", expiredStatusName))

	return nil
}

// EnrichedValidateConsentAPIResponse builds ValidateConsentAPIResponse with enriched purpose details (type, description, attributes, isMandatory)
func (consentService *consentService) EnrichedValidateConsentAPIResponse(ctx context.Context, consent *model.ConsentResponse, orgID string) *model.ValidateConsentAPIResponse {
	logger := log.GetLogger().WithContext(ctx)
	logger.Debug("Building enriched validate response with purpose details",
		log.String("consent_id", consent.ConsentID),
		log.String("org_id", orgID))

	purposeStore := consentService.stores.ConsentPurpose

	if consent == nil {
		logger.Debug("Consent is nil, returning nil")
		return nil
	}

	// Build base response
	validateResponse := &model.ValidateConsentAPIResponse{
		ID:                         consent.ConsentID,
		Type:                       consent.ConsentType,
		ClientID:                   consent.ClientID,
		Status:                     consent.CurrentStatus,
		CreatedTime:                consent.CreatedTime,
		UpdatedTime:                consent.UpdatedTime,
		ValidityTime:               consent.ValidityTime,
		RecurringIndicator:         consent.RecurringIndicator,
		Frequency:                  consent.ConsentFrequency,
		DataAccessValidityDuration: consent.DataAccessValidityDuration,
		Attributes:                 consent.Attributes,
	}

	// Convert authorizations
	if len(consent.AuthResources) > 0 {
		validateResponse.Authorizations = make([]model.AuthorizationAPIResponse, 0, len(consent.AuthResources))
		for _, auth := range consent.AuthResources {
			// Parse resources JSON string to interface
			var resources interface{}
			if auth.Resources != nil && *auth.Resources != "" {
				if err := json.Unmarshal([]byte(*auth.Resources), &resources); err != nil {
					// If parsing fails, set to empty object
					resources = make(map[string]interface{})
				}
			} else {
				// If resources is nil or empty, set to empty object
				resources = make(map[string]interface{})
			}

			validateResponse.Authorizations = append(validateResponse.Authorizations, model.AuthorizationAPIResponse{
				ID:          auth.AuthID,
				UserID:      auth.UserID,
				Type:        auth.AuthType,
				Status:      auth.AuthStatus,
				UpdatedTime: auth.UpdatedTime,
				Resources:   resources,
			})
		}
	}

	// Enrich purpose groups with full purpose details (type, description, attributes, isMandatory)
	if len(consent.PurposeGroups) > 0 {
		enrichedGroups := make([]model.ConsentPurposeGroupItemValidate, 0, len(consent.PurposeGroups))

		for _, group := range consent.PurposeGroups {
			enrichedGroup := model.ConsentPurposeGroupItemValidate{
				PurposeGroupName: group.PurposeGroupName,
				Purposes:         make([]model.ConsentPurposeApprovalItemValidate, 0, len(group.Purposes)),
			}

			for _, p := range group.Purposes {
				enrichedPurpose := model.ConsentPurposeApprovalItemValidate{
					PurposeName:    p.PurposeName,
					IsUserApproved: p.IsUserApproved,
					Value:          p.Value,
					IsMandatory:    p.IsMandatory,
				}

				// Fetch full purpose details from consent purpose service
				if p.PurposeName != "" {
					purpose, err := purposeStore.GetByName(ctx, p.PurposeName, orgID)
					if err == nil && purpose != nil {
						// Enrich with purpose details
						enrichedPurpose.Type = purpose.Type

						// Dereference description pointer if not nil
						if purpose.Description != nil {
							enrichedPurpose.Description = *purpose.Description
						}

						// Fetch attributes from CONSENT_PURPOSE_ATTRIBUTE table
						attributes, attrErr := purposeStore.GetAttributesByPurposeID(ctx, purpose.ID, orgID)
						if attrErr == nil && len(attributes) > 0 {
							enrichedPurpose.Attributes = make(map[string]interface{})
							for _, attr := range attributes {
								enrichedPurpose.Attributes[attr.Key] = attr.Value
							}
						}

						logger.Debug("Purpose details enriched for validate",
							log.String("purpose", p.PurposeName),
							log.String("type", purpose.Type),
							log.String("description", enrichedPurpose.Description),
							log.Bool("isMandatory", enrichedPurpose.IsMandatory),
							log.Int("attributes_count", len(enrichedPurpose.Attributes)))
					} else if err != nil {
						logger.Warn("Failed to fetch purpose details",
							log.String("purpose", p.PurposeName),
							log.Error(err))
					} else {
						logger.Warn("Purpose not found in database",
							log.String("purpose", p.PurposeName),
							log.String("org_id", orgID))
					}
				}

				enrichedGroup.Purposes = append(enrichedGroup.Purposes, enrichedPurpose)
			}

			enrichedGroups = append(enrichedGroups, enrichedGroup)
		}

		// Set enriched purpose groups
		validateResponse.PurposeGroups = enrichedGroups
	}

	logger.Debug("Validate response enriched successfully",
		log.Int("purpose_group_count", len(validateResponse.PurposeGroups)))

	return validateResponse
}

// EnrichedConsentAPIResponseWithPurposeDetails enriches consent response - kept for potential future use
func (consentService *consentService) EnrichedConsentAPIResponseWithPurposeDetails(ctx context.Context, consent *model.ConsentResponse, orgID string) *model.ConsentAPIResponse {
	if consent == nil {
		return nil
	}
	// Use ToAPIResponse to build the complete base response structure
	return consent.ToAPIResponse()
}

// buildConsentResponse constructs a complete ConsentResponse from already-resolved data.
// This is a pure data transformation function that takes pre-fetched data and builds the response.
// No database access is performed here - all data must be provided as parameters.
// This makes the function easily testable and free of side effects.
func buildConsentResponse(
	consent *model.Consent,
	purposeGroups []model.ConsentPurposeGroupItem,
	attributes map[string]string,
	authResources []authmodel.AuthResource,
) *model.ConsentResponse {
	// AuthResource is already a type alias for ConsentAuthResource - use directly
	authResourcesResp := authResources

	return &model.ConsentResponse{
		ConsentID:                  consent.ConsentID,
		PurposeGroups:              purposeGroups,
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
		AuthResources:              authResourcesResp,
	}
}

// SearchConsentsByAttribute searches for consents by attribute key and optionally value
// If value is empty, it searches by key only
func (consentService *consentService) SearchConsentsByAttribute(ctx context.Context, key, value, orgID string) (*model.ConsentAttributeSearchResponse, *serviceerror.ServiceError) {
	logger := log.GetLogger().WithContext(ctx)
	logger.Info("Searching consents by attribute",
		log.String("key", key),
		log.String("value", value),
		log.String("org_id", orgID))

	consentStore := consentService.stores.Consent

	var consentIDs []string
	var err error

	// If value is provided and not empty, search by key-value pair
	// Otherwise, search by key only
	if value != "" {
		consentIDs, err = consentStore.FindConsentIDsByAttribute(ctx, key, value, orgID)
	} else {
		consentIDs, err = consentStore.FindConsentIDsByAttributeKey(ctx, key, orgID)
	}

	if err != nil {
		logger.Error("Failed to search consents by attribute",
			log.Error(err),
			log.String("key", key),
			log.String("value", value))
		return nil, serviceerror.CustomServiceError(serviceerror.DatabaseError, err.Error())
	}

	logger.Info("Consents searched by attribute successfully",
		log.Int("count", len(consentIDs)))

	return &model.ConsentAttributeSearchResponse{
		ConsentIDs: consentIDs,
		Count:      len(consentIDs),
	}, nil
}

// getResolvedConsentPurposesWithGroups fetches purpose groups for a consent and resolves all purposes.
// This is a generic method that constructs the complete purpose group structure with all purposes:
// 1. Fetches purpose group mappings from CONSENT_PURPOSE_GROUP_CONSENT table
// 2. For each linked group, fetches ALL purposes defined in that group from DB
// 3. Creates a fresh map with default values (isUserApproved=false, value=nil)
// 4. Fetches approval records and uses them ONLY to update approval values
// Returns fully resolved purpose groups ready for response serialization.
func (s *consentService) getResolvedConsentPurposesWithGroups(
	ctx context.Context,
	consentID, orgID string,
) ([]model.ConsentPurposeGroupItem, error) {
	logger := log.GetLogger().WithContext(ctx)

	consentStore := s.stores.Consent
	purposeGroupStore := s.stores.ConsentPurposeGroup

	// Step 1: Fetch purpose group mappings from CONSENT_PURPOSE_GROUP_CONSENT table
	// This is the source of truth for which groups are linked to this consent
	groupMappings, err := consentStore.GetPurposeGroupsByConsentID(ctx, consentID, orgID)
	if err != nil {
		logger.Error("Failed to fetch purpose group mappings",
			log.String("consent_id", consentID),
			log.Error(err))
		return nil, fmt.Errorf("failed to fetch purpose group mappings: %w", err)
	}

	// If no groups are mapped, return empty list
	if len(groupMappings) == 0 {
		logger.Debug("No purpose groups mapped to consent",
			log.String("consent_id", consentID))
		return []model.ConsentPurposeGroupItem{}, nil
	}

	logger.Debug("Identified purpose groups from mappings",
		log.String("consent_id", consentID),
		log.Int("group_count", len(groupMappings)))

	// Step 2: For each mapped group, fetch ALL purposes defined in that group
	// and build a fresh map with default values
	purposeGroupsMap := make(map[string]*model.ConsentPurposeGroupItem)

	for _, mapping := range groupMappings {
		groupID := mapping.GroupID
		groupName := mapping.GroupName

		// Fetch all purposes in this group from database
		groupPurposes, err := purposeGroupStore.GetGroupPurposes(ctx, groupID, orgID)
		if err != nil {
			logger.Error("Failed to fetch purposes for group",
				log.String("group_name", groupName),
				log.String("group_id", groupID),
				log.Error(err))
			return nil, fmt.Errorf("failed to fetch purposes for group '%s': %w", groupName, err)
		}

		// Initialize group with all purposes having default values
		purposeGroupsMap[groupName] = &model.ConsentPurposeGroupItem{
			PurposeGroupName: groupName,
			Purposes:         make([]model.ConsentPurposeApprovalItem, 0, len(groupPurposes)),
		}

		// Add all purposes with default values (isUserApproved=false, value=nil, isMandatory from group definition)
		for _, gp := range groupPurposes {
			purposeGroupsMap[groupName].Purposes = append(
				purposeGroupsMap[groupName].Purposes,
				model.ConsentPurposeApprovalItem{
					PurposeName:    gp.PurposeName,
					IsUserApproved: false,          // Default: not approved
					Value:          nil,            // Default: no value
					IsMandatory:    gp.IsMandatory, // From group definition
				},
			)
		}

		logger.Debug("Initialized purpose group with default values",
			log.String("group_name", groupName),
			log.Int("purpose_count", len(groupPurposes)))
	}

	// Step 3: Fetch approval records from CONSENT_PURPOSE_APPROVAL table
	// These are used ONLY to update the approval status and values
	approvals, err := consentStore.GetPurposeApprovalsByConsentID(ctx, consentID, orgID)
	if err != nil {
		logger.Error("Failed to fetch purpose approvals",
			log.String("consent_id", consentID),
			log.Error(err))
		return nil, fmt.Errorf("failed to fetch purpose approvals: %w", err)
	}

	// Step 4: Update the map with actual approval values from database
	for _, approval := range approvals {
		// Parse value from JSON string
		var value interface{}
		if approval.Value != nil && *approval.Value != "" {
			if err := json.Unmarshal([]byte(*approval.Value), &value); err != nil {
				logger.Warn("Failed to unmarshal purpose value",
					log.String("purpose", approval.PurposeName),
					log.String("group", approval.GroupName),
					log.Error(err))
			}
		}

		// Find and update the purpose in the group
		if group, exists := purposeGroupsMap[approval.GroupName]; exists {
			for i := range group.Purposes {
				if group.Purposes[i].PurposeName == approval.PurposeName {
					// Update with actual approval values from DB
					group.Purposes[i].IsUserApproved = approval.IsUserApproved
					group.Purposes[i].Value = value
					break
				}
			}
		}
	}

	logger.Debug("Updated purpose groups with approval values",
		log.String("consent_id", consentID),
		log.Int("approval_count", len(approvals)))

	// Step 5: Convert map to slice for response
	purposeGroups := make([]model.ConsentPurposeGroupItem, 0, len(purposeGroupsMap))
	for _, pg := range purposeGroupsMap {
		purposeGroups = append(purposeGroups, *pg)
	}

	logger.Debug("Resolved consent purpose groups",
		log.String("consent_id", consentID),
		log.Int("group_count", len(purposeGroups)))

	return purposeGroups, nil
}

// validateNoDuplicatePurposesAcrossGroups ensures no purpose appears in multiple groups.
// This validation is called AFTER purpose resolution from database, so it checks the
// complete set of purposes (including auto-filled ones), not just what user provided.
// This prevents a purpose from being assigned to multiple groups, which would create
// ambiguity in consent management.
func (s *consentService) validateNoDuplicatePurposesAcrossGroups(
	purposeGroups []model.ConsentPurposeGroupCreateRequest,
) error {
	// Track which group each purpose belongs to
	purposeNamesSeen := make(map[string]string) // purpose name -> group name

	for _, pg := range purposeGroups {
		for _, p := range pg.Purposes {
			if existingGroup, found := purposeNamesSeen[p.PurposeName]; found {
				// Found duplicate - same purpose in multiple groups
				return fmt.Errorf(
					"duplicate purpose '%s' found in groups '%s' and '%s'",
					p.PurposeName,
					existingGroup,
					pg.GroupName,
				)
			}
			purposeNamesSeen[p.PurposeName] = pg.GroupName
		}
	}

	return nil
}

// validatePurposeGroups validates purpose groups and resolves all purposes from database.
// This method:
// 1. Fetches purpose group definitions from DB (validates group existence)
// 2. Fetches all purposes within each group from DB
// 3. Validates that user-provided purposes belong to their respective groups
// 4. Resolves missing purposes (not in request) with isUserApproved=false
// 5. Validates no duplicate purposes exist across all resolved groups
// Returns fully enriched purpose groups ready for database insertion.
func (s *consentService) validatePurposeGroups(
	ctx context.Context,
	purposeGroups []model.ConsentPurposeGroupCreateRequest,
	clientID, orgID string,
) ([]model.ConsentPurposeGroupCreateRequest, error) {

	logger := log.GetLogger().WithContext(ctx)

	// Step 1: Resolve purpose groups and get their full definitions from database
	resolvedGroups := make([]model.ConsentPurposeGroupCreateRequest, 0, len(purposeGroups))
	purposeGroupStore := s.stores.ConsentPurposeGroup

	for _, pg := range purposeGroups {
		// Fetch purpose group metadata by name for this client
		// Note: Using limit=1 since group names should be unique per client
		groups, _, err := purposeGroupStore.ListGroups(ctx, orgID, pg.GroupName, []string{clientID}, nil, 0, 1)
		if err != nil {
			logger.Error("Failed to fetch purpose group from database",
				log.String("group_name", pg.GroupName),
				log.String("client_id", clientID),
				log.Error(err))
			return nil, fmt.Errorf("failed to get purpose group '%s': %w", pg.GroupName, err)
		}

		if len(groups) == 0 {
			logger.Warn("Purpose group not found",
				log.String("group_name", pg.GroupName),
				log.String("client_id", clientID))
			return nil, fmt.Errorf("purpose group '%s' not found for client '%s'", pg.GroupName, clientID)
		}

		group := groups[0]
		logger.Debug("Purpose group found",
			log.String("group_name", pg.GroupName),
			log.String("group_id", group.ID))

		// Step 2: Fetch ALL purposes defined in this group from database
		// This gives us the complete list of purposes with their IDs, names, and mandatory flags
		groupPurposesFromDB, err := purposeGroupStore.GetGroupPurposes(ctx, group.ID, orgID)
		if err != nil {
			logger.Error("Failed to fetch purposes for group",
				log.String("group_name", pg.GroupName),
				log.String("group_id", group.ID),
				log.Error(err))
			return nil, fmt.Errorf("failed to get purposes for group '%s': %w", pg.GroupName, err)
		}

		logger.Debug("Fetched purposes for group",
			log.String("group_name", pg.GroupName),
			log.Int("total_purposes", len(groupPurposesFromDB)),
			log.Int("requested_purposes", len(pg.Purposes)))

		// Step 3: Create lookup map for purposes provided in the request
		// Key: purpose name, Value: approval details from request
		requestedPurposes := make(map[string]model.ConsentPurposeApprovalCreateRequest)
		for _, p := range pg.Purposes {
			requestedPurposes[p.PurposeName] = p
		}

		// Step 4: Create lookup map for valid purposes from database
		// This is used to validate that user-provided purposes actually belong to this group
		validPurposeNames := make(map[string]bool)
		for _, gp := range groupPurposesFromDB {
			validPurposeNames[gp.PurposeName] = true
		}

		// Step 5: Validate that all requested purposes belong to this group
		for purposeName := range requestedPurposes {
			if !validPurposeNames[purposeName] {
				logger.Warn("Purpose does not belong to group",
					log.String("purpose_name", purposeName),
					log.String("group_name", pg.GroupName))
				return nil, fmt.Errorf("purpose '%s' does not belong to group '%s'", purposeName, pg.GroupName)
			}
		}

		// Step 6: Resolve ALL purposes in the group (merge requested + missing)
		// For purposes in request: use their approval status and values
		// For purposes not in request: add with isUserApproved=false (user didn't approve)
		allPurposes := make([]model.ConsentPurposeApprovalCreateRequest, 0, len(groupPurposesFromDB))

		for _, dbPurpose := range groupPurposesFromDB {
			if requestedPurpose, found := requestedPurposes[dbPurpose.PurposeName]; found {
				// Purpose was explicitly provided in request - use user's approval status
				requestedPurpose.PurposeID = dbPurpose.PurposeID
				requestedPurpose.IsMandatory = dbPurpose.IsMandatory
				allPurposes = append(allPurposes, requestedPurpose)
				logger.Debug("Using requested purpose approval",
					log.String("purpose", dbPurpose.PurposeName),
					log.Bool("approved", requestedPurpose.IsUserApproved))
			} else {
				// Purpose was NOT in request - auto-fill with isUserApproved=false
				allPurposes = append(allPurposes, model.ConsentPurposeApprovalCreateRequest{
					PurposeID:      dbPurpose.PurposeID,
					PurposeName:    dbPurpose.PurposeName,
					IsUserApproved: false, // Not approved since user didn't provide it
					Value:          nil,
					IsMandatory:    dbPurpose.IsMandatory,
				})
				logger.Debug("Auto-filling missing purpose",
					log.String("purpose", dbPurpose.PurposeName),
					log.Bool("approved", false))
			}
		}

		// Add fully resolved group to result
		resolvedGroups = append(resolvedGroups, model.ConsentPurposeGroupCreateRequest{
			GroupName: pg.GroupName,
			GroupID:   group.ID,
			Purposes:  allPurposes, // Contains ALL purposes (requested + auto-filled)
		})
	}

	// Step 7: Validate no duplicate purposes across ALL resolved groups
	// This check happens AFTER resolution because we now have the complete picture
	// of all purposes across all groups (including auto-filled ones)
	if err := s.validateNoDuplicatePurposesAcrossGroups(resolvedGroups); err != nil {
		logger.Warn("Duplicate purpose validation failed", log.Error(err))
		return nil, err
	}

	logger.Info("Purpose groups validated and resolved successfully",
		log.Int("group_count", len(resolvedGroups)))

	return resolvedGroups, nil
}
