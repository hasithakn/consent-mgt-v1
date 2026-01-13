package consentpurposegroup

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/wso2/consent-management-api/internal/consentpurposegroup/model"
	"github.com/wso2/consent-management-api/internal/system/constants"
	"github.com/wso2/consent-management-api/internal/system/error/serviceerror"
	"github.com/wso2/consent-management-api/internal/system/utils"
)

// consentPurposeGroupHandler handles HTTP requests for purpose groups
type consentPurposeGroupHandler struct {
	service ConsentPurposeGroupService
}

// newConsentPurposeGroupHandler creates a new purpose group handler
func newConsentPurposeGroupHandler(service ConsentPurposeGroupService) *consentPurposeGroupHandler {
	return &consentPurposeGroupHandler{
		service: service,
	}
}

// createPurposeGroup handles POST /consent-purpose-groups
func (h *consentPurposeGroupHandler) createPurposeGroup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := r.Header.Get(constants.HeaderOrgID)
	clientID := r.Header.Get(constants.HeaderTPPClientID)

	// Validate required headers
	if orgID == "" {
		utils.SendError(w, r, serviceerror.CustomServiceError(serviceerror.InvalidRequestError, "org-id header is required"))
		return
	}
	if clientID == "" {
		utils.SendError(w, r, serviceerror.CustomServiceError(serviceerror.InvalidRequestError, "TPP-client-id header is required"))
		return
	}

	// Decode request
	var req model.CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.SendError(w, r, serviceerror.CustomServiceError(serviceerror.InvalidRequestError, "invalid request body"))
		return
	}

	// Create purpose group
	group, serviceErr := h.service.CreatePurposeGroup(ctx, req, orgID, clientID)
	if serviceErr != nil {
		utils.SendError(w, r, serviceErr)
		return
	}

	// Return response
	response := group.ToResponse()
	w.Header().Set(constants.HeaderContentType, "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// getPurposeGroup handles GET /consent-purpose-groups/{groupId}
func (h *consentPurposeGroupHandler) getPurposeGroup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := r.Header.Get(constants.HeaderOrgID)
	groupID := r.PathValue("groupId")

	// Validate required headers
	if orgID == "" {
		utils.SendError(w, r, serviceerror.CustomServiceError(serviceerror.InvalidRequestError, "org-id header is required"))
		return
	}

	if groupID == "" {
		utils.SendError(w, r, serviceerror.CustomServiceError(serviceerror.InvalidRequestError, "groupId is required"))
		return
	}

	// Get purpose group
	group, serviceErr := h.service.GetPurposeGroup(ctx, groupID, orgID)
	if serviceErr != nil {
		utils.SendError(w, r, serviceErr)
		return
	}

	// Return response
	response := group.ToResponse()
	w.Header().Set(constants.HeaderContentType, "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// listPurposeGroups handles GET /consent-purpose-groups
func (h *consentPurposeGroupHandler) listPurposeGroups(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := r.Header.Get(constants.HeaderOrgID)

	// Validate required headers
	if orgID == "" {
		utils.SendError(w, r, serviceerror.CustomServiceError(serviceerror.InvalidRequestError, "org-id header is required"))
		return
	}

	// Parse query parameters
	queryParams := r.URL.Query()

	// Parse name filter
	name := strings.TrimSpace(queryParams.Get("name"))

	// Parse clientIds (comma-separated)
	var clientIDs []string
	if clientIDsParam := queryParams.Get("clientIds"); clientIDsParam != "" {
		clientIDs = strings.Split(clientIDsParam, ",")
		// Trim spaces
		for i := range clientIDs {
			clientIDs[i] = strings.TrimSpace(clientIDs[i])
		}
	}

	// Parse purposeNames (comma-separated)
	var purposeNames []string
	if purposeNamesParam := queryParams.Get("purposeNames"); purposeNamesParam != "" {
		purposeNames = strings.Split(purposeNamesParam, ",")
		// Trim spaces
		for i := range purposeNames {
			purposeNames[i] = strings.TrimSpace(purposeNames[i])
		}
	}

	// Parse pagination parameters
	limit := 100
	if limitParam := queryParams.Get("limit"); limitParam != "" {
		if l, err := strconv.Atoi(limitParam); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	offset := 0
	if offsetParam := queryParams.Get("offset"); offsetParam != "" {
		if o, err := strconv.Atoi(offsetParam); err == nil && o >= 0 {
			offset = o
		}
	}

	// List purpose groups
	groups, total, serviceErr := h.service.ListPurposeGroups(ctx, orgID, name, clientIDs, purposeNames, offset, limit)
	if serviceErr != nil {
		utils.SendError(w, r, serviceErr)
		return
	}

	// Convert to response format
	responses := make([]model.Response, 0, len(groups))
	for _, g := range groups {
		responses = append(responses, g.ToResponse())
	}

	// Build response with pagination metadata
	response := model.ListResponse{
		Data: responses,
		Metadata: model.PaginationMetadata{
			Total:  total,
			Offset: offset,
			Count:  len(responses),
			Limit:  limit,
		},
	}

	w.Header().Set(constants.HeaderContentType, "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// updatePurposeGroup handles PUT /consent-purpose-groups/{groupId}
func (h *consentPurposeGroupHandler) updatePurposeGroup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := r.Header.Get(constants.HeaderOrgID)
	clientID := r.Header.Get(constants.HeaderTPPClientID)
	groupID := r.PathValue("groupId")

	// Validate required headers and parameters
	if orgID == "" {
		utils.SendError(w, r, serviceerror.CustomServiceError(serviceerror.InvalidRequestError, "org-id header is required"))
		return
	}
	if clientID == "" {
		utils.SendError(w, r, serviceerror.CustomServiceError(serviceerror.InvalidRequestError, "TPP-client-id header is required"))
		return
	}
	if groupID == "" {
		utils.SendError(w, r, serviceerror.CustomServiceError(serviceerror.InvalidRequestError, "groupId is required"))
		return
	}

	// Decode request
	var req model.UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.SendError(w, r, serviceerror.CustomServiceError(serviceerror.InvalidRequestError, "invalid request body"))
		return
	}

	// Update purpose group
	group, serviceErr := h.service.UpdatePurposeGroup(ctx, groupID, req, orgID, clientID)
	if serviceErr != nil {
		utils.SendError(w, r, serviceErr)
		return
	}

	// Return response
	response := group.ToResponse()
	w.Header().Set(constants.HeaderContentType, "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// deletePurposeGroup handles DELETE /consent-purpose-groups/{groupId}
func (h *consentPurposeGroupHandler) deletePurposeGroup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := r.Header.Get(constants.HeaderOrgID)
	groupID := r.PathValue("groupId")

	// Validate required headers and parameters
	if orgID == "" {
		utils.SendError(w, r, serviceerror.CustomServiceError(serviceerror.InvalidRequestError, "org-id header is required"))
		return
	}
	if groupID == "" {
		utils.SendError(w, r, serviceerror.CustomServiceError(serviceerror.InvalidRequestError, "groupId is required"))
		return
	}

	// Delete purpose group
	if serviceErr := h.service.DeletePurposeGroup(ctx, groupID, orgID); serviceErr != nil {
		utils.SendError(w, r, serviceErr)
		return
	}

	// Return no content
	w.WriteHeader(http.StatusNoContent)
}
