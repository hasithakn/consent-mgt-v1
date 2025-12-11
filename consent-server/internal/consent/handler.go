package consent

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/wso2/consent-management-api/internal/consent/model"
	"github.com/wso2/consent-management-api/internal/system/constants"
	"github.com/wso2/consent-management-api/internal/system/error/serviceerror"
	"github.com/wso2/consent-management-api/internal/system/utils"
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
	clientID := r.Header.Get(constants.HeaderTPPClientID)

	if err := utils.ValidateOrgIdAndClientIdIsPresent(r); err != nil {
		utils.SendError(w, serviceerror.CustomServiceError(serviceerror.InvalidRequestError, err.Error()))
		return
	}

	var req model.ConsentAPIRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.SendError(w, serviceerror.CustomServiceError(serviceerror.InvalidRequestError, "Invalid request body"))
		return
	}

	consent, serviceErr := h.service.CreateConsent(ctx, req, clientID, orgID)
	if serviceErr != nil {
		utils.SendError(w, serviceErr)
		return
	}

	apiResponse := consent.ToAPIResponse()
	w.Header().Set(constants.HeaderContentType, "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(apiResponse)
}

// getConsent handles GET /consents/{consentId}
func (h *consentHandler) getConsent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	consentID := r.PathValue("consentId")
	orgID := r.Header.Get(constants.HeaderOrgID)

	// TODO: Is clientID validation needed?

	if err := utils.ValidateOrgIdAndClientIdIsPresent(r); err != nil {
		utils.SendError(w, serviceerror.CustomServiceError(serviceerror.InvalidRequestError, err.Error()))
		return
	}

	if err := utils.ValidateConsentID(consentID); err != nil {
		utils.SendError(w, serviceerror.CustomServiceError(serviceerror.InvalidRequestError, err.Error()))
		return
	}

	consent, serviceErr := h.service.GetConsent(ctx, consentID, orgID)
	if serviceErr != nil {
		utils.SendError(w, serviceErr)
		return
	}

	apiResponse := consent.ToAPIResponse()
	w.Header().Set(constants.HeaderContentType, "application/json")
	json.NewEncoder(w).Encode(apiResponse)
}

// listConsents handles GET /consents
func (h *consentHandler) listConsents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := r.Header.Get(constants.HeaderOrgID)

	if orgID == "" {
		utils.SendError(w, serviceerror.CustomServiceError(serviceerror.InvalidRequestError, "Organization ID is required"))
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
		utils.SendError(w, serviceErr)
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

// updateConsent handles PUT /consents/{consentId}
func (h *consentHandler) updateConsent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	consentID := r.PathValue("consentId")
	orgID := r.Header.Get(constants.HeaderOrgID)

	if err := utils.ValidateOrgIdAndClientIdIsPresent(r); err != nil {
		utils.SendError(w, serviceerror.CustomServiceError(serviceerror.InvalidRequestError, err.Error()))
		return
	}

	if err := utils.ValidateConsentID(consentID); err != nil {
		utils.SendError(w, serviceerror.CustomServiceError(serviceerror.InvalidRequestError, err.Error()))
		return
	}

	var req model.ConsentAPIUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.SendError(w, serviceerror.CustomServiceError(serviceerror.InvalidRequestError, "Invalid request body"))
		return
	}

	consent, serviceErr := h.service.UpdateConsent(ctx, req, orgID, consentID)
	if serviceErr != nil {
		utils.SendError(w, serviceErr)
		return
	}

	apiResponse := consent.ToAPIResponse()
	w.Header().Set(constants.HeaderContentType, "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(apiResponse)
}

// revokeConsent handles POST /consents/{consentId}/revoke
func (h *consentHandler) revokeConsent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	consentID := r.PathValue("consentId")
	orgID := r.Header.Get(constants.HeaderOrgID)

	if err := utils.ValidateOrgIdAndClientIdIsPresent(r); err != nil {
		utils.SendError(w, serviceerror.CustomServiceError(serviceerror.InvalidRequestError, err.Error()))
		return
	}

	if err := utils.ValidateConsentID(consentID); err != nil {
		utils.SendError(w, serviceerror.CustomServiceError(serviceerror.InvalidRequestError, err.Error()))
		return
	}

	var req model.ConsentRevokeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.SendError(w, serviceerror.CustomServiceError(serviceerror.InvalidRequestError, "Invalid request body"))
		return
	}

	revokeResponse, serviceErr := h.service.RevokeConsent(ctx, consentID, orgID, req)
	if serviceErr != nil {
		utils.SendError(w, serviceErr)
		return
	}

	w.Header().Set(constants.HeaderContentType, "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(revokeResponse)
}

// validateConsent handles POST /consents/validate
func (h *consentHandler) validateConsent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := r.Header.Get(constants.HeaderOrgID)

	if err := utils.ValidateOrgIdAndClientIdIsPresent(r); err != nil {
		utils.SendError(w, serviceerror.CustomServiceError(serviceerror.InvalidRequestError, err.Error()))
		return
	}

	var req model.ValidateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.SendError(w, serviceerror.CustomServiceError(serviceerror.InvalidRequestError, "Invalid request body"))
		return
	}

	// Call service to validate consent
	response, serviceErr := h.service.ValidateConsent(ctx, req, orgID)
	if serviceErr != nil {
		utils.SendError(w, serviceErr)
		return
	}

	// Always return HTTP 200, check isValid field in response
	w.Header().Set(constants.HeaderContentType, "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// searchConsentsByAttribute handles GET /consents/attributes
func (h *consentHandler) searchConsentsByAttribute(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orgID := r.Header.Get(constants.HeaderOrgID)

	if err := utils.ValidateOrgIdAndClientIdIsPresent(r); err != nil {
		utils.SendError(w, serviceerror.CustomServiceError(serviceerror.InvalidRequestError, err.Error()))
		return
	}

	// Get query parameters
	key := r.URL.Query().Get("key")
	value := r.URL.Query().Get("value")

	// Validate that key parameter is present
	if key == "" {
		utils.SendError(w, serviceerror.CustomServiceError(serviceerror.InvalidRequestError, "key parameter is required"))
		return
	}

	// Call service to search consents by attribute
	response, serviceErr := h.service.SearchConsentsByAttribute(ctx, key, value, orgID)
	if serviceErr != nil {
		utils.SendError(w, serviceErr)
		return
	}

	w.Header().Set(constants.HeaderContentType, "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
