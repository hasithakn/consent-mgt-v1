package consent

import (
	"context"
	"encoding/json"
	"fmt"

	authmodel "github.com/wso2/consent-management-api/internal/authresource/model"
	"github.com/wso2/consent-management-api/internal/consent/model"
	"github.com/wso2/consent-management-api/internal/consent/validator"
	purposemodel "github.com/wso2/consent-management-api/internal/consentpurpose/model"
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

	// Link consent purposes if provided
	if len(createReq.ConsentPurpose) > 0 {
		logger.Debug("Linking consent purposes", log.Int("purpose_count", len(createReq.ConsentPurpose)))
		purposeStore := consentService.stores.ConsentPurpose

		// Extract purpose names
		purposeNames := make([]string, len(createReq.ConsentPurpose))
		for i, p := range createReq.ConsentPurpose {
			purposeNames[i] = p.Name
		}

		// Get purpose IDs by names
		purposeIDMap, err := purposeStore.GetIDsByNames(ctx, purposeNames, orgID)
		if err != nil {
			logger.Error("Failed to get purpose IDs by names", log.Error(err))
			return nil, serviceerror.CustomServiceError(serviceerror.DatabaseError, fmt.Sprintf("failed to get purpose IDs: %v", err))
		}

		// Verify all purposes were found
		if len(purposeIDMap) != len(purposeNames) {
			missingPurposes := []string{}
			for _, name := range purposeNames {
				if _, found := purposeIDMap[name]; !found {
					missingPurposes = append(missingPurposes, name)
				}
			}
			logger.Warn("Some consent purposes not found", log.Any("missing_purposes", missingPurposes))
			return nil, serviceerror.CustomServiceError(serviceerror.ValidationError, fmt.Sprintf("purposes not found: %v", missingPurposes))
		}

		// Link each purpose
		for _, purposeItem := range createReq.ConsentPurpose {
			purposeID := purposeIDMap[purposeItem.Name]

			// Marshal value to JSON string if present
			var valueJSON *string
			if purposeItem.Value != nil {
				valueBytes, err := json.Marshal(purposeItem.Value)
				if err != nil {
					logger.Error("Failed to marshal consent purpose value",
						log.Error(err),
						log.String("purpose_name", purposeItem.Name))
					return nil, serviceerror.CustomServiceError(serviceerror.ValidationError, fmt.Sprintf("failed to marshal purpose value: %v", err))
				}
				valueStr := string(valueBytes)
				valueJSON = &valueStr
			}

			// Get boolean values with defaults
			isUserApproved := false
			if purposeItem.IsUserApproved != nil {
				isUserApproved = *purposeItem.IsUserApproved
			}

			isMandatory := true
			if purposeItem.IsMandatory != nil {
				isMandatory = *purposeItem.IsMandatory
			}

			// Add to transaction queries
			queries = append(queries, func(tx dbmodel.TxInterface) error {
				return purposeStore.LinkPurposeToConsent(tx, consentID, purposeID, orgID, valueJSON, isUserApproved, isMandatory)
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
	purposeMappings, _ := consentService.stores.ConsentPurpose.GetMappingsByConsentID(ctx, consentID, orgID)
	attributes, _ := consentService.stores.Consent.GetAttributesByConsentID(ctx, consentID, orgID)

	// Convert attributes slice to map[string]string
	attributesMap := make(map[string]string)
	for _, a := range attributes {
		attributesMap[a.AttKey] = a.AttValue
	}

	// Build complete response
	response := buildConsentResponse(consent, attributesMap, authResources, purposeMappings)

	logger.Info("Consent creation completed",
		log.String("consent_id", consentID),
		log.String("status", consent.CurrentStatus),
		log.Int("auth_resources", len(authResources)),
		log.Int("purposes", len(purposeMappings)),
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
	purposeStore := consentService.stores.ConsentPurpose

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
	purposeMappings, _ := purposeStore.GetMappingsByConsentID(ctx, consentID, orgID)

	// Convert attributes slice to map
	attributesMap := make(map[string]string)
	for _, a := range attributes {
		attributesMap[a.AttKey] = a.AttValue
	}

	// Build complete response with all related data
	response := buildConsentResponse(consent, attributesMap, authResources, purposeMappings)

	logger.Debug("Consent retrieved successfully",
		log.String("consent_id", consentID),
		log.String("status", consent.CurrentStatus),
		log.Int("auth_resources", len(authResources)),
		log.Int("purposes", len(purposeMappings)),
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

	// Step 3: Batch fetch related data in parallel
	authResourceStore := consentService.stores.AuthResource
	purposeStore := consentService.stores.ConsentPurpose

	authResources, err := authResourceStore.GetByConsentIDs(ctx, consentIDs, filters.OrgID)
	if err != nil {
		logger.Error("Failed to get authorization resources", log.Error(err))
		return nil, serviceerror.CustomServiceError(serviceerror.DatabaseError, err.Error())
	}

	purposeMappings, err := purposeStore.GetMappingsByConsentIDs(ctx, consentIDs, filters.OrgID)
	if err != nil {
		logger.Error("Failed to get purpose mappings", log.Error(err))
		return nil, serviceerror.CustomServiceError(serviceerror.DatabaseError, err.Error())
	}

	attributesByConsent, err := consentStore.GetAttributesByConsentIDs(ctx, consentIDs, filters.OrgID)
	if err != nil {
		logger.Error("Failed to get consent attributes", log.Error(err))
		return nil, serviceerror.CustomServiceError(serviceerror.DatabaseError, err.Error())
	}

	// Step 4: Group by consent ID
	authsByConsent := make(map[string][]authmodel.AuthResource)
	for _, auth := range authResources {
		authsByConsent[auth.ConsentID] = append(authsByConsent[auth.ConsentID], auth)
	}

	purposesByConsent := make(map[string][]purposemodel.ConsentPurposeMapping)
	for _, mapping := range purposeMappings {
		purposesByConsent[mapping.ConsentID] = append(purposesByConsent[mapping.ConsentID], mapping)
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

		// Build consent purposes - initialize as empty slice
		consentPurposes := make([]model.ConsentPurposeItem, 0)
		for _, mapping := range purposesByConsent[consent.ConsentID] {
			var value interface{}
			// mapping.Value is already interface{}, check if it's string and unmarshal
			if mapping.Value != nil {
				if strVal, ok := mapping.Value.(string); ok && strVal != "" {
					_ = json.Unmarshal([]byte(strVal), &value)
				} else {
					value = mapping.Value
				}
			}

			// Convert bool to *bool for optional fields
			isUserApproved := mapping.IsUserApproved
			isMandatory := mapping.IsMandatory

			consentPurposes = append(consentPurposes, model.ConsentPurposeItem{
				Name:           mapping.Name,
				Value:          value,
				IsUserApproved: &isUserApproved,
				IsMandatory:    &isMandatory,
			})
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
			ConsentPurposes:            consentPurposes,
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
	purposeStore := consentService.stores.ConsentPurpose

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

	// Update consent purposes if provided
	if updateReq.ConsentPurpose != nil {

		// Clear existing purpose mappings
		queries = append(queries, func(tx dbmodel.TxInterface) error {
			return purposeStore.DeleteMappingsByConsentID(tx, consentID, orgID)
		})

		// Link new purposes if not empty
		if len(updateReq.ConsentPurpose) > 0 {
			// Extract purpose names
			purposeNames := make([]string, len(updateReq.ConsentPurpose))
			for i, p := range updateReq.ConsentPurpose {
				purposeNames[i] = p.Name
			}

			// Get purpose IDs by names
			purposeIDMap, err := purposeStore.GetIDsByNames(ctx, purposeNames, orgID)
			if err != nil {
				logger.Error("Failed to get purpose IDs by names", log.Error(err))
				return nil, serviceerror.CustomServiceError(serviceerror.DatabaseError, fmt.Sprintf("failed to get purpose IDs: %v", err))
			}

			// Verify all purposes were found
			if len(purposeIDMap) != len(purposeNames) {
				missingPurposes := []string{}
				for _, name := range purposeNames {
					if _, found := purposeIDMap[name]; !found {
						missingPurposes = append(missingPurposes, name)
					}
				}
				logger.Warn("Some consent purposes not found", log.Any("missing_purposes", missingPurposes))
				return nil, serviceerror.CustomServiceError(serviceerror.ValidationError, fmt.Sprintf("purposes not found: %v", missingPurposes))
			}

			// Link each purpose
			for _, purposeItem := range updateReq.ConsentPurpose {
				purposeID := purposeIDMap[purposeItem.Name]

				// Marshal value to JSON string if present
				var valueJSON *string
				if purposeItem.Value != nil {
					valueBytes, err := json.Marshal(purposeItem.Value)
					if err != nil {
						return nil, serviceerror.CustomServiceError(serviceerror.ValidationError, fmt.Sprintf("failed to marshal purpose value: %v", err))
					}
					valueStr := string(valueBytes)
					valueJSON = &valueStr
				}

				// Get boolean values with defaults
				isUserApproved := false
				if purposeItem.IsUserApproved != nil {
					isUserApproved = *purposeItem.IsUserApproved
				}

				isMandatory := true
				if purposeItem.IsMandatory != nil {
					isMandatory = *purposeItem.IsMandatory
				}

				// Add to transaction queries
				queries = append(queries, func(tx dbmodel.TxInterface) error {
					return purposeStore.LinkPurposeToConsent(tx, consentID, purposeID, orgID, valueJSON, isUserApproved, isMandatory)
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
	purposeMappings, _ := purposeStore.GetMappingsByConsentID(ctx, consentID, orgID)
	attributes, _ := consentStore.GetAttributesByConsentID(ctx, consentID, orgID)

	// Convert attributes slice to map[string]string
	attributesMap := make(map[string]string)
	for _, a := range attributes {
		attributesMap[a.AttKey] = a.AttValue
	}

	// Build complete response
	response := buildConsentResponse(updated, attributesMap, authResources, purposeMappings)

	logger.Info("Consent updated successfully",
		log.String("consent_id", consentID),
		log.String("status", updated.CurrentStatus),
		log.Int("auth_resources", len(authResources)),
		log.Int("purposes", len(purposeMappings)),
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
		purposeStore := consentService.stores.ConsentPurpose

		attributes, _ := consentStore.GetAttributesByConsentID(ctx, consent.ConsentID, orgID)
		authResources, _ := authResourceStore.GetByConsentID(ctx, consent.ConsentID, orgID)
		purposeMappings, _ := purposeStore.GetMappingsByConsentID(ctx, consent.ConsentID, orgID)

		// Convert attributes slice to map
		attributesMap := make(map[string]string)
		for _, a := range attributes {
			attributesMap[a.AttKey] = a.AttValue
		}

		// Build complete consent response
		consentResponse := buildConsentResponse(consent, attributesMap, authResources, purposeMappings)

		// Convert to API response and then to ValidateConsentAPIResponse (which excludes modifiedResponse)
		apiResponse := consentService.EnrichedConsentAPIResponseWithPurposeDetails(ctx, consentResponse, orgID)
		response.ConsentInformation = apiResponse.ToValidateConsentAPIResponse()
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

func (consentService *consentService) EnrichedConsentAPIResponseWithPurposeDetails(ctx context.Context, consent *model.ConsentResponse, orgID string) *model.ConsentAPIResponse {
	logger := log.GetLogger().WithContext(ctx)
	logger.Debug("Enriching consent response with purpose details",
		log.String("consent_id", consent.ConsentID),
		log.String("org_id", orgID))

	purposeStore := consentService.stores.ConsentPurpose

	if consent == nil {
		logger.Debug("Consent is nil, returning nil")
		return nil
	}

	// Use ToAPIResponse to build the complete base response structure
	apiResponse := consent.ToAPIResponse()

	// Enrich consent purposes with full purpose details (type, description, attributes)
	if len(apiResponse.ConsentPurpose) > 0 {
		enrichedPurposes := make([]model.ConsentPurposeItem, 0, len(apiResponse.ConsentPurpose))

		for _, cp := range apiResponse.ConsentPurpose {
			// Convert base purpose to enriched purpose
			enrichedPurpose := cp

			// Fetch full purpose details from consent purpose service
			if cp.Name != "" {
				purpose, err := purposeStore.GetByName(ctx, cp.Name, orgID)
				if err == nil && purpose != nil {
					// Enrich with type, description, and attributes from the purpose definition
					enrichedPurpose.Type = &purpose.Type
					enrichedPurpose.Description = purpose.Description

					attributes, _ := purposeStore.GetAttributesByPurposeID(ctx, purpose.ID, orgID)

					// Convert []ConsentPurposeAttribute to map[string]interface{}
					if len(attributes) > 0 {
						attrs := make(map[string]interface{}, len(attributes))
						for _, attr := range attributes {
							attrs[attr.Key] = attr.Value
						}
						enrichedPurpose.Attributes = attrs
					} else {
						enrichedPurpose.Attributes = map[string]interface{}{}
					}
				} else {
					// If we can't fetch the purpose, add empty values for enriched fields
					emptyType := ""
					emptyDesc := ""
					enrichedPurpose.Type = &emptyType
					enrichedPurpose.Description = &emptyDesc
					enrichedPurpose.Attributes = map[string]interface{}{}
				}
			}

			enrichedPurposes = append(enrichedPurposes, enrichedPurpose)
		}

		// Set enriched purposes
		apiResponse.ConsentPurpose = enrichedPurposes
	}

	logger.Debug("Consent response enriched successfully",
		log.Int("purpose_count", len(apiResponse.ConsentPurpose)))

	return apiResponse
}

// buildConsentResponse constructs a complete ConsentResponse with related data
func buildConsentResponse(
	consent *model.Consent,
	attributes map[string]string,
	authResources []authmodel.AuthResource,
	purposeMappings []purposemodel.ConsentPurposeMapping,
) *model.ConsentResponse {
	// Convert purposeMappings to ConsentPurposeItem
	purposes := make([]model.ConsentPurposeItem, len(purposeMappings))
	for i, mapping := range purposeMappings {
		purposes[i] = model.ConsentPurposeItem{
			Name:           mapping.Name,
			Value:          mapping.Value,
			IsUserApproved: &mapping.IsUserApproved,
			IsMandatory:    &mapping.IsMandatory,
		}
	}

	// AuthResource is already a type alias for ConsentAuthResource - use directly
	authResourcesResp := authResources

	return &model.ConsentResponse{
		ConsentID:                  consent.ConsentID,
		ConsentPurpose:             purposes,
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
