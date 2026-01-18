package consentelement

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/wso2/consent-management-api/internal/consentelement/model"
	"github.com/wso2/consent-management-api/internal/system/constants"
	"github.com/wso2/consent-management-api/internal/system/error/serviceerror"
	"github.com/wso2/consent-management-api/internal/system/utils"
)

// consentElementHandler handles HTTP requests for consent elements
type consentElementHandler struct {
	service ConsentElementService
}

// newConsentElementHandler creates a new consent element handler
func newConsentElementHandler(service ConsentElementService) *consentElementHandler {
	return &consentElementHandler{
		service: service,
	}
}

// createElement handles POST /consent-elements
// Supports both single and batch creation (array input)
func (h *consentElementHandler) createElement(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := r.Header.Get(constants.HeaderOrgID)

	// Validate required headers
	if err := utils.ValidateOrgID(orgID); err != nil {
		utils.SendError(w, r, serviceerror.CustomServiceError(serviceerror.InvalidRequestError, err.Error()))
		return
	}

	// Decode as array of requests (batch creation)
	var requests []model.ConsentElementCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&requests); err != nil {
		utils.SendError(w, r, serviceerror.CustomServiceError(serviceerror.InvalidRequestError, "invalid request body"))
		return
	}

	// Validate at least one element provided
	if len(requests) == 0 {
		utils.SendError(w, r, serviceerror.CustomServiceError(serviceerror.InvalidRequestError, "at least one element must be provided"))
		return
	}

	// Create elements in batch (atomic transaction)
	elements, serviceErr := h.service.CreateElementsInBatch(ctx, requests, orgID)
	if serviceErr != nil {
		utils.SendError(w, r, serviceErr)
		return
	}

	// Convert to response format
	responses := make([]model.ConsentElementResponse, 0, len(elements))
	for _, p := range elements {
		responses = append(responses, model.ConsentElementResponse{
			ID:          p.ID,
			Name:        p.Name,
			Description: p.Description,
			Type:        p.Type,
			Properties:  p.Properties,
		})
	}

	// Return response with data wrapper
	response := map[string]interface{}{
		"data":    responses,
		"message": "Consent elements created successfully",
	}

	w.Header().Set(constants.HeaderContentType, "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// getElement handles GET /consent-elements/{elementId}
func (h *consentElementHandler) getElement(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	elementID := r.PathValue("elementId")
	orgID := r.Header.Get(constants.HeaderOrgID)

	if orgID == "" {
		utils.SendError(w, r, serviceerror.CustomServiceError(serviceerror.ValidationError, "organization ID is required"))
		return
	}

	element, serviceErr := h.service.GetElement(ctx, elementID, orgID)
	if serviceErr != nil {
		utils.SendError(w, r, serviceErr)
		return
	}

	response := model.ConsentElementResponse{
		ID:          element.ID,
		Name:        element.Name,
		Description: element.Description,
		Type:        element.Type,
		Properties:  element.Properties,
	}

	w.Header().Set(constants.HeaderContentType, "application/json")
	json.NewEncoder(w).Encode(response)
}

// listElements handles GET /consent-elements
func (h *consentElementHandler) listElements(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := r.Header.Get(constants.HeaderOrgID)

	if orgID == "" {
		utils.SendError(w, r, serviceerror.CustomServiceError(serviceerror.ValidationError, "organization ID is required"))
		return
	}

	// Parse pagination parameters
	limit := 100 // default from swagger spec
	offset := 0

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	// Parse optional name filter
	name := r.URL.Query().Get("name")

	elements, total, serviceErr := h.service.ListElements(ctx, orgID, limit, offset, name)
	if serviceErr != nil {
		utils.SendError(w, r, serviceErr)
		return
	}

	// Convert to response models
	elementResponses := make([]model.ConsentElementResponse, 0, len(elements))
	for _, p := range elements {
		elementResponses = append(elementResponses, model.ConsentElementResponse{
			ID:          p.ID,
			Name:        p.Name,
			Description: p.Description,
			Type:        p.Type,
			Properties:  p.Properties,
		})
	}

	// Build response with metadata as per swagger spec
	response := map[string]interface{}{
		"data": elementResponses,
		"metadata": map[string]int{
			"total":  total,
			"offset": offset,
			"count":  len(elementResponses),
			"limit":  limit,
		},
	}

	w.Header().Set(constants.HeaderContentType, "application/json")
	json.NewEncoder(w).Encode(response)
}

// updateElement handles PUT /consent-elements/{elementId}
func (h *consentElementHandler) updateElement(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	elementID := r.PathValue("elementId")
	orgID := r.Header.Get(constants.HeaderOrgID)

	// Validate required headers
	if err := utils.ValidateOrgID(orgID); err != nil {
		utils.SendError(w, r, serviceerror.CustomServiceError(serviceerror.InvalidRequestError, err.Error()))
		return
	}

	var req model.ConsentElementUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.SendError(w, r, serviceerror.CustomServiceError(serviceerror.InvalidRequestError, "invalid request body"))
		return
	}

	element, serviceErr := h.service.UpdateElement(ctx, elementID, req, orgID)
	if serviceErr != nil {
		utils.SendError(w, r, serviceErr)
		return
	}

	response := model.ConsentElementResponse{
		ID:          element.ID,
		Name:        element.Name,
		Description: element.Description,
		Type:        element.Type,
		Properties:  element.Properties,
	}

	w.Header().Set(constants.HeaderContentType, "application/json")
	json.NewEncoder(w).Encode(response)
}

// deleteElement handles DELETE /consent-elements/{elementId}
func (h *consentElementHandler) deleteElement(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	elementID := r.PathValue("elementId")
	orgID := r.Header.Get(constants.HeaderOrgID)

	if orgID == "" {
		utils.SendError(w, r, serviceerror.CustomServiceError(serviceerror.ValidationError, "organization ID is required"))
		return
	}

	if serviceErr := h.service.DeleteElement(ctx, elementID, orgID); serviceErr != nil {
		utils.SendError(w, r, serviceErr)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// validateElements handles POST /consent-elements/validate
func (h *consentElementHandler) validateElements(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := r.Header.Get(constants.HeaderOrgID)

	if orgID == "" {
		utils.SendError(w, r, serviceerror.CustomServiceError(serviceerror.ValidationError, "organization ID is required"))
		return
	}

	var elementNames []string
	if err := json.NewDecoder(r.Body).Decode(&elementNames); err != nil {
		utils.SendError(w, r, serviceerror.CustomServiceError(serviceerror.InvalidRequestError, "invalid request body"))
		return
	}

	validNames, serviceErr := h.service.ValidateElementNames(ctx, orgID, elementNames)
	if serviceErr != nil {
		utils.SendError(w, r, serviceErr)
		return
	}

	w.Header().Set(constants.HeaderContentType, "application/json")
	json.NewEncoder(w).Encode(validNames)
}

// sendError sends an error response based on ServiceError type
