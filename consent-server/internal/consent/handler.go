package consent

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/wso2/consent-management-api/internal/consent/model"
	"github.com/wso2/consent-management-api/internal/system/constants"
	"github.com/wso2/consent-management-api/internal/system/error/apierror"
	"github.com/wso2/consent-management-api/internal/system/error/serviceerror"
)

type consentHandler struct {
	service ConsentService
}

func newConsentHandler(service ConsentService) *consentHandler {
	return &consentHandler{
		service: service,
	}
}

// createConsent handles POST /consents
func (h *consentHandler) createConsent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := r.Header.Get(constants.HeaderOrgID)
	clientID := r.Header.Get("TPP-client-id")

	if orgID == "" {
		sendError(w, serviceerror.CustomServiceError(serviceerror.InvalidRequestError, "Organization ID is required"))
		return
	}

	if clientID == "" {
		sendError(w, serviceerror.CustomServiceError(serviceerror.InvalidRequestError, "TPP-client-id header is required"))
		return
	}

	var req model.ConsentAPIRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, serviceerror.CustomServiceError(serviceerror.InvalidRequestError, "Invalid request body"))
		return
	}

	response, serviceErr := h.service.CreateConsent(ctx, req, clientID, orgID)
	if serviceErr != nil {
		sendError(w, serviceErr)
		return
	}

	w.Header().Set(constants.HeaderContentType, "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// getConsent handles GET /consents/{id}
func (h *consentHandler) getConsent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	consentID := r.PathValue("id")
	orgID := r.Header.Get(constants.HeaderOrgID)

	if orgID == "" {
		sendError(w, serviceerror.CustomServiceError(serviceerror.InvalidRequestError, "Organization ID is required"))
		return
	}

	response, serviceErr := h.service.GetConsent(ctx, consentID, orgID)
	if serviceErr != nil {
		sendError(w, serviceErr)
		return
	}

	w.Header().Set(constants.HeaderContentType, "application/json")
	json.NewEncoder(w).Encode(response)
}

// listConsents handles GET /consents
func (h *consentHandler) listConsents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := r.Header.Get(constants.HeaderOrgID)

	if orgID == "" {
		sendError(w, serviceerror.CustomServiceError(serviceerror.InvalidRequestError, "Organization ID is required"))
		return
	}

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

	consents, total, serviceErr := h.service.ListConsents(ctx, orgID, limit, offset)
	if serviceErr != nil {
		sendError(w, serviceErr)
		return
	}

	response := model.ConsentSearchResponse{
		Data: consents,
		Metadata: model.ConsentSearchMetadata{
			Total:  total,
			Limit:  limit,
			Offset: offset,
		},
	}

	w.Header().Set(constants.HeaderContentType, "application/json")
	json.NewEncoder(w).Encode(response)
}

// updateConsent handles PUT /consents/{id}
func (h *consentHandler) updateConsent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	consentID := r.PathValue("id")
	orgID := r.Header.Get(constants.HeaderOrgID)

	if orgID == "" {
		sendError(w, serviceerror.CustomServiceError(serviceerror.InvalidRequestError, "Organization ID is required"))
		return
	}

	var req model.ConsentAPIUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, serviceerror.CustomServiceError(serviceerror.InvalidRequestError, "Invalid request body"))
		return
	}

	response, serviceErr := h.service.UpdateConsent(ctx, consentID, req, orgID)
	if serviceErr != nil {
		sendError(w, serviceErr)
		return
	}

	w.Header().Set(constants.HeaderContentType, "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// updateConsentStatus handles PATCH /consents/{id}/status
func (h *consentHandler) updateConsentStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	consentID := r.PathValue("id")
	orgID := r.Header.Get(constants.HeaderOrgID)

	if orgID == "" {
		sendError(w, serviceerror.CustomServiceError(serviceerror.InvalidRequestError, "Organization ID is required"))
		return
	}

	var req model.ConsentRevokeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, serviceerror.CustomServiceError(serviceerror.InvalidRequestError, "Invalid request body"))
		return
	}

	if serviceErr := h.service.UpdateConsentStatus(ctx, consentID, orgID, req); serviceErr != nil {
		sendError(w, serviceErr)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// deleteConsent handles DELETE /consents/{id}
func (h *consentHandler) deleteConsent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	consentID := r.PathValue("id")
	orgID := r.Header.Get(constants.HeaderOrgID)

	if orgID == "" {
		sendError(w, serviceerror.CustomServiceError(serviceerror.InvalidRequestError, "Organization ID is required"))
		return
	}

	if serviceErr := h.service.DeleteConsent(ctx, consentID, orgID); serviceErr != nil {
		sendError(w, serviceErr)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func sendError(w http.ResponseWriter, err *serviceerror.ServiceError) {
	statusCode := http.StatusInternalServerError
	if err.Type == serviceerror.ClientErrorType {
		if err.Code == serviceerror.ResourceNotFoundError.Code {
			statusCode = http.StatusNotFound
		} else if err.Code == serviceerror.ConflictError.Code {
			statusCode = http.StatusConflict
		} else {
			statusCode = http.StatusBadRequest
		}
	}

	errorResponse := apierror.ErrorResponse{
		Code:        err.Error,
		Description: err.ErrorDescription,
	}

	w.Header().Set(constants.HeaderContentType, constants.ContentTypeJSON)
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(errorResponse)
}
