package consent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/wso2/consent-management-api/internal/authresource"
	authmodel "github.com/wso2/consent-management-api/internal/authresource/model"
	"github.com/wso2/consent-management-api/internal/consent/model"
	"github.com/wso2/consent-management-api/internal/consent/validator"
	"github.com/wso2/consent-management-api/internal/consentpurpose"
	purposemodel "github.com/wso2/consent-management-api/internal/consentpurpose/model"
	"github.com/wso2/consent-management-api/internal/system/config"
	dbmodel "github.com/wso2/consent-management-api/internal/system/database/model"
	"github.com/wso2/consent-management-api/internal/system/error/serviceerror"
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
	// Validate request
	if err := utils.ValidateOrgID(orgID); err != nil {
		return nil, serviceerror.CustomServiceError(serviceerror.ValidationError, err.Error())
	}
	if err := utils.ValidateClientID(clientID); err != nil {
		return nil, serviceerror.CustomServiceError(serviceerror.ValidationError, err.Error())
	}
	if err := validator.ValidateConsentCreateRequest(req, clientID, orgID); err != nil {
		return nil, serviceerror.CustomServiceError(serviceerror.ValidationError, err.Error())
	}

	// Convert API request to internal format
	createReq, err := req.ToConsentCreateRequest()
	if err != nil {
		return nil, serviceerror.CustomServiceError(serviceerror.ValidationError, err.Error())
	}

	// Extract auth statuses
	authStatuses := make([]string, 0, len(createReq.AuthResources))
	for _, ar := range createReq.AuthResources {
		authStatuses = append(authStatuses, ar.AuthStatus)
	}

	// Derive consent status from authorization states
	consentStatus := validator.EvaluateConsentStatusFromAuthStatuses(authStatuses)

	// Generate IDs and timestamp
	consentID := utils.GenerateUUID()
	currentTime := utils.GetCurrentTimeMillis()

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
	consentStore := consentService.stores.Consent.(ConsentStore)
	authResourceStore := consentService.stores.AuthResource.(authresource.AuthResourceStore)

	// Build list of transactional operations
	queries := []func(tx dbmodel.TxInterface) error{
		// Create consent
		func(tx dbmodel.TxInterface) error {
			return consentStore.Create(tx, consent)
		},
	}

	// Add attributes if provided
	if len(createReq.Attributes) > 0 {
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
	for _, authReq := range req.Authorizations {
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
		purposeStore := consentService.stores.ConsentPurpose.(consentpurpose.ConsentPurposeStore)

		// Extract purpose names
		purposeNames := make([]string, len(createReq.ConsentPurpose))
		for i, p := range createReq.ConsentPurpose {
			purposeNames[i] = p.Name
		}

		// Get purpose IDs by names
		purposeIDMap, err := purposeStore.GetIDsByNames(ctx, purposeNames, orgID)
		if err != nil {
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
	if err := consentService.stores.ExecuteTransaction(queries); err != nil {
		return nil, serviceerror.CustomServiceError(serviceerror.DatabaseError, fmt.Sprintf("failed to create consent: %v", err))
	}

	// TODO : check consent expireation and handle accordingly.

	// Retrieve related data after creation
	authResources, _ := authResourceStore.GetByConsentID(ctx, consentID, orgID)
	purposeMappings, _ := consentService.stores.ConsentPurpose.(consentpurpose.ConsentPurposeStore).GetMappingsByConsentID(ctx, consentID, orgID)
	attributes, _ := consentService.stores.Consent.(ConsentStore).GetAttributesByConsentID(ctx, consentID, orgID)

	// Convert attributes slice to map[string]string
	attributesMap := make(map[string]string)
	for _, a := range attributes {
		attributesMap[a.AttKey] = a.AttValue
	}

	// Build complete response
	response := buildConsentResponse(consent, attributesMap, authResources, purposeMappings)

	return response, nil
}

// GetConsent retrieves a consent by ID with all related data
func (consentService *consentService) GetConsent(ctx context.Context, consentID, orgID string) (*model.ConsentResponse, *serviceerror.ServiceError) {

	// Get stores
	consentStore := consentService.stores.Consent.(ConsentStore)
	authResourceStore := consentService.stores.AuthResource.(authresource.AuthResourceStore)
	purposeStore := consentService.stores.ConsentPurpose.(consentpurpose.ConsentPurposeStore)

	// Get consent
	consent, err := consentStore.GetByID(ctx, consentID, orgID)
	if err != nil {
		return nil, serviceerror.CustomServiceError(serviceerror.DatabaseError, err.Error())
	}
	if consent == nil {
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

	return response, nil
}

// ListConsents retrieves paginated list of consents
func (consentService *consentService) ListConsents(ctx context.Context, orgID string, limit, offset int) ([]model.ConsentResponse, int, *serviceerror.ServiceError) {
	if limit <= 0 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}

	store := consentService.stores.Consent.(ConsentStore)
	consents, total, err := store.List(ctx, orgID, limit, offset)
	if err != nil {
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

	return responses, total, nil
}

// SearchConsents retrieves consents based on search filters with pagination
func (consentService *consentService) SearchConsents(ctx context.Context, filters model.ConsentSearchFilters) ([]model.ConsentResponse, int, *serviceerror.ServiceError) {
	// Validate pagination
	if filters.Limit <= 0 {
		filters.Limit = 10
	}
	if filters.Offset < 0 {
		filters.Offset = 0
	}

	store := consentService.stores.Consent.(ConsentStore)
	consents, total, err := store.Search(ctx, filters)
	if err != nil {
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

	return responses, total, nil
}

// SearchConsentsDetailed retrieves consents with nested authorization resources, purposes, and attributes
func (consentService *consentService) SearchConsentsDetailed(ctx context.Context, filters model.ConsentSearchFilters) (*model.ConsentDetailSearchResponse, *serviceerror.ServiceError) {
	// Validate pagination
	if filters.Limit <= 0 {
		filters.Limit = 10
	}
	if filters.Offset < 0 {
		filters.Offset = 0
	}

	// Step 1: Search consents
	consentStore := consentService.stores.Consent.(ConsentStore)
	consents, total, err := consentStore.Search(ctx, filters)
	if err != nil {
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
	authResourceStore := consentService.stores.AuthResource.(authresource.AuthResourceStore)
	purposeStore := consentService.stores.ConsentPurpose.(consentpurpose.ConsentPurposeStore)

	authResources, err := authResourceStore.GetByConsentIDs(ctx, consentIDs, filters.OrgID)
	if err != nil {
		return nil, serviceerror.CustomServiceError(serviceerror.DatabaseError, err.Error())
	}

	purposeMappings, err := purposeStore.GetMappingsByConsentIDs(ctx, consentIDs, filters.OrgID)
	if err != nil {
		return nil, serviceerror.CustomServiceError(serviceerror.DatabaseError, err.Error())
	}

	attributesByConsent, err := consentStore.GetAttributesByConsentIDs(ctx, consentIDs, filters.OrgID)
	if err != nil {
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

	// Get stores
	authResourceStore := consentService.stores.AuthResource.(authresource.AuthResourceStore)
	consentStore := consentService.stores.Consent.(ConsentStore)
	purposeStore := consentService.stores.ConsentPurpose.(consentpurpose.ConsentPurposeStore)

	// Validate request
	if err := utils.ValidateOrgID(orgID); err != nil {
		return nil, serviceerror.CustomServiceError(serviceerror.ValidationError, err.Error())
	}
	if err := validator.ValidateConsentUpdateRequest(req); err != nil {
		return nil, serviceerror.CustomServiceError(serviceerror.ValidationError, err.Error())
	}

	// Convert to internal format
	updateReq, convertErr := req.ToConsentUpdateRequest()
	if convertErr != nil {
		return nil, serviceerror.CustomServiceError(serviceerror.ValidationError, convertErr.Error())
	}

	// Check if consent exists
	existing, err := consentStore.GetByID(ctx, consentID, orgID)
	if err != nil {
		return nil, serviceerror.CustomServiceError(serviceerror.DatabaseError, err.Error())
	}
	if existing == nil {
		return nil, serviceerror.CustomServiceError(serviceerror.ResourceNotFoundError, fmt.Sprintf("Consent with ID '%s' not found", consentID))
	}

	currentTime := utils.GetCurrentTimeMillis()
	previousStatus := existing.CurrentStatus

	if req.DataAccessValidityDuration != nil {
		// Validate that it's non-negative
		if *req.DataAccessValidityDuration < 0 {
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
	if err := consentService.stores.ExecuteTransaction(queries); err != nil {
		return nil, serviceerror.CustomServiceError(serviceerror.DatabaseError, err.Error())
	}

	// Get updated consent
	updated, getErr := consentStore.GetByID(ctx, consentID, orgID)
	if getErr != nil {
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

	return response, nil
}

// RevokeConsent updates consent status and creates audit entry
func (consentService *consentService) RevokeConsent(ctx context.Context, consentID, orgID string, req model.ConsentRevokeRequest) (*model.ConsentRevokeResponse, *serviceerror.ServiceError) {
	// Validate action by
	if req.ActionBy == "" {
		return nil, serviceerror.CustomServiceError(serviceerror.ValidationError, "ActionBy is required")
	}

	revokedStatusName := config.Get().Consent.GetRevokedConsentStatus()

	// Check if consent exists
	store := consentService.stores.Consent.(ConsentStore)
	existing, err := store.GetByID(ctx, consentID, orgID)
	if err != nil {
		return nil, serviceerror.CustomServiceError(serviceerror.DatabaseError, err.Error())
	}
	if existing == nil {
		return nil, serviceerror.CustomServiceError(serviceerror.ValidationError, fmt.Sprintf("Consent with ID '%s' not found", consentID))
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
	authResourceStore := consentService.stores.AuthResource.(authresource.AuthResourceStore)

	// Execute transaction - update consent status, all auth resource statuses, and create audit
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
		return nil, serviceerror.CustomServiceError(serviceerror.DatabaseError, err.Error())
	}

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
	// Initialize response with invalid state
	response := &model.ValidateResponse{
		IsValid: false,
	}

	// Validate request
	if req.ConsentID == "" {
		return nil, serviceerror.CustomServiceError(serviceerror.ValidationError, "ConsentID is required")
	}

	// Get consent
	consentStore := consentService.stores.Consent.(ConsentStore)
	consent, err := consentStore.GetByID(ctx, req.ConsentID, orgID)
	if err != nil {
		// return nil, serviceerror.CustomServiceError(serviceerror.DatabaseError, err.Error())
		response.ErrorCode = 500
		response.ErrorMessage = "database_error"
		response.ErrorDescription = "Database error while retrieving consent"
	}
	if consent == nil {
		// return nil, serviceerror.CustomServiceError(serviceerror.ValidationError, fmt.Sprintf("Consent with ID '%s' not found", req.ConsentID))
		response.ErrorCode = 404
		response.ErrorMessage = "not_found"
		response.ErrorDescription = "Consent not found"
	}

	// Check if consent is expired and update status accordingly
	expiredStatusName := string(config.Get().Consent.GetExpiredConsentStatus())
	if consent.ValidityTime != nil && validator.IsConsentExpired(*consent.ValidityTime) {
		// Update consent status to expired if not already expired
		if consent.CurrentStatus != expiredStatusName {
			if err := consentService.expireConsent(ctx, consent, orgID); err != nil {
				// Log error but continue with validation
				// The consent object is already updated in-memory by expireConsent
			} else {
				consent, err = consentStore.GetByID(ctx, req.ConsentID, orgID)
			}
		}
	}

	// Check consent status - only active consents are valid
	activeStatusName := string(config.Get().Consent.GetActiveConsentStatus())
	if consent.CurrentStatus != activeStatusName && response.ErrorCode == 0 {
		response.ErrorCode = 401
		response.ErrorMessage = "invalid_consent_status"
		response.ErrorDescription = fmt.Sprintf("Consent status is '%s', expected '%s'", consent.CurrentStatus, activeStatusName)
	}

	// If no errors, mark as valid
	if response.ErrorCode == 0 {
		response.IsValid = true
	}

	// Retrieve related data for consent information
	authResourceStore := consentService.stores.AuthResource.(authresource.AuthResourceStore)
	purposeStore := consentService.stores.ConsentPurpose.(consentpurpose.ConsentPurposeStore)

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

	return response, nil
}

// expireConsent updates consent and all related auth resources to expired status
func (consentService *consentService) expireConsent(ctx context.Context, consent *model.Consent, orgID string) error {
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
	consentStore := consentService.stores.Consent.(ConsentStore)
	authResourceStore := consentService.stores.AuthResource.(authresource.AuthResourceStore)

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
		return err
	}

	// Update local consent object
	consent.CurrentStatus = expiredStatusName
	consent.UpdatedTime = currentTime

	return nil
}

func (consentService *consentService) EnrichedConsentAPIResponseWithPurposeDetails(ctx context.Context, consent *model.ConsentResponse, orgID string) *model.ConsentAPIResponse {

	purposeStore := consentService.stores.ConsentPurpose.(consentpurpose.ConsentPurposeStore)

	if consent == nil {
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
	consentStore := consentService.stores.Consent.(ConsentStore)

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
		return nil, serviceerror.CustomServiceError(serviceerror.DatabaseError, err.Error())
	}

	return &model.ConsentAttributeSearchResponse{
		ConsentIDs: consentIDs,
		Count:      len(consentIDs),
	}, nil
}
