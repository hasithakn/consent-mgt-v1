package consentpurpose

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/wso2/consent-management-api/internal/consentpurpose/model"
	"github.com/wso2/consent-management-api/internal/system/constants"
	"github.com/wso2/consent-management-api/internal/system/error/serviceerror"
	"github.com/wso2/consent-management-api/internal/system/utils"
)

// consentPurposeHandler handles HTTP requests for consent purposes
type consentPurposeHandler struct {
	service ConsentPurposeService
}

// newConsentPurposeHandler creates a new consent purpose handler
func newConsentPurposeHandler(service ConsentPurposeService) *consentPurposeHandler {
	return &consentPurposeHandler{
		service: service,
	}
}

// createPurpose handles POST /purposes
func (h *consentPurposeHandler) createPurpose(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := r.Header.Get(constants.HeaderOrgID)

	if orgID == "" {
		utils.SendError(w, serviceerror.CustomServiceError(serviceerror.ValidationError, "organization ID is required"))
		return
	}

	var req model.CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.SendError(w, serviceerror.CustomServiceError(serviceerror.InvalidRequestError, "invalid request body"))
		return
	}

	purpose, serviceErr := h.service.CreatePurpose(ctx, req, orgID)
	if serviceErr != nil {
		utils.SendError(w, serviceErr)
		return
	}

	response := model.Response{
		ID:          purpose.ID,
		Name:        purpose.Name,
		Description: purpose.Description,
		Type:        purpose.Type,
		Attributes:  purpose.Attributes,
	}

	w.Header().Set(constants.HeaderContentType, "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// getPurpose handles GET /consent-purposes/{purposeId}
func (h *consentPurposeHandler) getPurpose(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	purposeID := r.PathValue("purposeId")
	orgID := r.Header.Get(constants.HeaderOrgID)

	if orgID == "" {
		utils.SendError(w, serviceerror.CustomServiceError(serviceerror.ValidationError, "organization ID is required"))
		return
	}

	purpose, serviceErr := h.service.GetPurpose(ctx, purposeID, orgID)
	if serviceErr != nil {
		utils.SendError(w, serviceErr)
		return
	}

	response := model.Response{
		ID:          purpose.ID,
		Name:        purpose.Name,
		Description: purpose.Description,
		Type:        purpose.Type,
		Attributes:  purpose.Attributes,
	}

	w.Header().Set(constants.HeaderContentType, "application/json")
	json.NewEncoder(w).Encode(response)
}

// listPurposes handles GET /purposes
func (h *consentPurposeHandler) listPurposes(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := r.Header.Get(constants.HeaderOrgID)

	if orgID == "" {
		utils.SendError(w, serviceerror.CustomServiceError(serviceerror.ValidationError, "organization ID is required"))
		return
	}

	// Parse pagination parameters
	limit := 10
	offset := 0

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil {
			offset = o
		}
	}

	purposes, total, serviceErr := h.service.ListPurposes(ctx, orgID, limit, offset)
	if serviceErr != nil {
		utils.SendError(w, serviceErr)
		return
	}

	// Convert to response models
	purposeResponses := make([]model.Response, 0, len(purposes))
	for _, p := range purposes {
		purposeResponses = append(purposeResponses, model.Response{
			ID:          p.ID,
			Name:        p.Name,
			Description: p.Description,
			Type:        p.Type,
			Attributes:  p.Attributes,
		})
	}

	response := model.ListResponse{
		Purposes: purposeResponses,
		Total:    total,
	}

	w.Header().Set(constants.HeaderContentType, "application/json")
	json.NewEncoder(w).Encode(response)
}

// updatePurpose handles PUT /consent-purposes/{purposeId}
func (h *consentPurposeHandler) updatePurpose(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	purposeID := r.PathValue("purposeId")
	orgID := r.Header.Get(constants.HeaderOrgID)

	if orgID == "" {
		utils.SendError(w, serviceerror.CustomServiceError(serviceerror.ValidationError, "organization ID is required"))
		return
	}

	var req model.UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.SendError(w, serviceerror.CustomServiceError(serviceerror.InvalidRequestError, "invalid request body"))
		return
	}

	purpose, serviceErr := h.service.UpdatePurpose(ctx, purposeID, req, orgID)
	if serviceErr != nil {
		utils.SendError(w, serviceErr)
		return
	}

	response := model.Response{
		ID:          purpose.ID,
		Name:        purpose.Name,
		Description: purpose.Description,
		Type:        purpose.Type,
		Attributes:  purpose.Attributes,
	}

	w.Header().Set(constants.HeaderContentType, "application/json")
	json.NewEncoder(w).Encode(response)
}

// deletePurpose handles DELETE /consent-purposes/{purposeId}
func (h *consentPurposeHandler) deletePurpose(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	purposeID := r.PathValue("purposeId")
	orgID := r.Header.Get(constants.HeaderOrgID)

	if orgID == "" {
		utils.SendError(w, serviceerror.CustomServiceError(serviceerror.ValidationError, "organization ID is required"))
		return
	}

	if serviceErr := h.service.DeletePurpose(ctx, purposeID, orgID); serviceErr != nil {
		utils.SendError(w, serviceErr)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// sendError sends an error response based on ServiceError type
