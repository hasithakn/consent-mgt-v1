package authresource

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/wso2/consent-management-api/internal/authresource/model"
	"github.com/wso2/consent-management-api/internal/system/constants"
	"github.com/wso2/consent-management-api/internal/system/error/serviceerror"
	"github.com/wso2/consent-management-api/internal/system/utils"
)

// authResourceHandler handles HTTP requests for auth resources
type authResourceHandler struct {
	service AuthResourceServiceInterface
}

// newAuthResourceHandler creates a new auth resource handler
func newAuthResourceHandler(service AuthResourceServiceInterface) *authResourceHandler {
	return &authResourceHandler{
		service: service,
	}
}

// handleCreate handles POST /consents/{consentId}/authorizations
func (h *authResourceHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract path parameters
	consentID := r.PathValue("consentId")
	if consentID == "" {
		utils.SendError(w, serviceerror.CustomServiceError(
			serviceerror.InvalidRequestError,
			"consent ID is required",
		))
		return
	}

	// Extract organization ID from header
	orgID := r.Header.Get(constants.HeaderOrgID)
	if orgID == "" {
		utils.SendError(w, serviceerror.CustomServiceError(
			serviceerror.InvalidRequestError,
			"organization ID header is required",
		))
		return
	}

	// Parse request body
	var request model.CreateRequest
	if err := utils.DecodeJSONBody(r, &request); err != nil {
		utils.SendError(w, serviceerror.CustomServiceError(
			serviceerror.InvalidRequestError,
			fmt.Sprintf("invalid request body: %v", err),
		))
		return
	}

	// Call service
	response, serviceErr := h.service.CreateAuthResource(ctx, consentID, orgID, &request)
	if serviceErr != nil {
		utils.SendError(w, serviceErr)
		return
	}

	// Send response
	w.Header().Set(constants.HeaderContentType, constants.ContentTypeJSON)
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// handleGet handles GET /consents/{consentId}/authorizations/{authorizationId}
func (h *authResourceHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract path parameters
	consentID := r.PathValue("consentId")
	authID := r.PathValue("authorizationId")
	if consentID == "" || authID == "" {
		utils.SendError(w, serviceerror.CustomServiceError(
			serviceerror.InvalidRequestError,
			"consent ID and auth ID are required",
		))
		return
	}

	// Extract organization ID from header
	orgID := r.Header.Get(constants.HeaderOrgID)
	if orgID == "" {
		utils.SendError(w, serviceerror.CustomServiceError(
			serviceerror.InvalidRequestError,
			"organization ID header is required",
		))
		return
	}

	// Call service
	response, serviceErr := h.service.GetAuthResource(ctx, authID, orgID)
	if serviceErr != nil {
		utils.SendError(w, serviceErr)
		return
	}

	// Send response
	w.Header().Set(constants.HeaderContentType, constants.ContentTypeJSON)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// handleListByConsent handles GET /consents/{consentId}/authorizations
func (h *authResourceHandler) handleListByConsent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract path parameters
	consentID := r.PathValue("consentId")
	if consentID == "" {
		utils.SendError(w, serviceerror.CustomServiceError(
			serviceerror.InvalidRequestError,
			"consent ID is required",
		))
		return
	}

	// Extract organization ID from header
	orgID := r.Header.Get(constants.HeaderOrgID)
	if orgID == "" {
		utils.SendError(w, serviceerror.CustomServiceError(
			serviceerror.InvalidRequestError,
			"organization ID header is required",
		))
		return
	}

	// Call service
	response, serviceErr := h.service.GetAuthResourcesByConsentID(ctx, consentID, orgID)
	if serviceErr != nil {
		utils.SendError(w, serviceErr)
		return
	}

	// Send response
	w.Header().Set(constants.HeaderContentType, constants.ContentTypeJSON)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// handleListByUser handles GET /auth-resources?userId=xxx
func (h *authResourceHandler) handleListByUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract query parameters
	userID := r.URL.Query().Get("userId")
	if userID == "" {
		utils.SendError(w, serviceerror.CustomServiceError(
			serviceerror.InvalidRequestError,
			"userId query parameter is required",
		))
		return
	}

	// Extract organization ID from header
	orgID := r.Header.Get(constants.HeaderOrgID)
	if orgID == "" {
		utils.SendError(w, serviceerror.CustomServiceError(
			serviceerror.InvalidRequestError,
			"organization ID header is required",
		))
		return
	}

	// Call service
	response, serviceErr := h.service.GetAuthResourcesByUserID(ctx, userID, orgID)
	if serviceErr != nil {
		utils.SendError(w, serviceErr)
		return
	}

	// Send response
	w.Header().Set(constants.HeaderContentType, constants.ContentTypeJSON)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// handleUpdate handles PUT /consents/{consentId}/authorizations/{authorizationId}
func (h *authResourceHandler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract path parameters
	consentID := r.PathValue("consentId")
	authID := r.PathValue("authorizationId")
	if consentID == "" || authID == "" {
		utils.SendError(w, serviceerror.CustomServiceError(
			serviceerror.InvalidRequestError,
			"consent ID and auth ID are required",
		))
		return
	}

	// Extract organization ID from header
	orgID := r.Header.Get(constants.HeaderOrgID)
	if orgID == "" {
		utils.SendError(w, serviceerror.CustomServiceError(
			serviceerror.InvalidRequestError,
			"organization ID header is required",
		))
		return
	}

	// Parse request body
	var request model.UpdateRequest
	if err := utils.DecodeJSONBody(r, &request); err != nil {
		utils.SendError(w, serviceerror.CustomServiceError(
			serviceerror.InvalidRequestError,
			fmt.Sprintf("invalid request body: %v", err),
		))
		return
	}

	// Call service
	response, serviceErr := h.service.UpdateAuthResource(ctx, authID, orgID, &request)
	if serviceErr != nil {
		utils.SendError(w, serviceErr)
		return
	}

	// Send response
	w.Header().Set(constants.HeaderContentType, constants.ContentTypeJSON)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// handleUpdateStatus handles PATCH /consents/{consentId}/auth-resources/{authId}/status
func (h *authResourceHandler) handleUpdateStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract path parameters
	consentID := r.PathValue("consentId")
	authID := r.PathValue("authId")
	if consentID == "" || authID == "" {
		utils.SendError(w, serviceerror.CustomServiceError(
			serviceerror.InvalidRequestError,
			"consent ID and auth ID are required",
		))
		return
	}

	// Extract organization ID from header
	orgID := r.Header.Get(constants.HeaderOrgID)
	if orgID == "" {
		utils.SendError(w, serviceerror.CustomServiceError(
			serviceerror.InvalidRequestError,
			"organization ID header is required",
		))
		return
	}

	// Parse request body
	var statusRequest struct {
		Status string `json:"status"`
	}
	if err := utils.DecodeJSONBody(r, &statusRequest); err != nil {
		utils.SendError(w, serviceerror.CustomServiceError(
			serviceerror.InvalidRequestError,
			fmt.Sprintf("invalid request body: %v", err),
		))
		return
	}

	// Call service
	response, serviceErr := h.service.UpdateAuthResourceStatus(ctx, authID, orgID, statusRequest.Status)
	if serviceErr != nil {
		utils.SendError(w, serviceErr)
		return
	}

	// Send response
	w.Header().Set(constants.HeaderContentType, constants.ContentTypeJSON)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// handleDelete handles DELETE /consents/{consentId}/auth-resources/{authId}
func (h *authResourceHandler) handleDelete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract path parameters
	consentID := r.PathValue("consentId")
	authID := r.PathValue("authId")
	if consentID == "" || authID == "" {
		utils.SendError(w, serviceerror.CustomServiceError(
			serviceerror.InvalidRequestError,
			"consent ID and auth ID are required",
		))
		return
	}

	// Extract organization ID from header
	orgID := r.Header.Get(constants.HeaderOrgID)
	if orgID == "" {
		utils.SendError(w, serviceerror.CustomServiceError(
			serviceerror.InvalidRequestError,
			"organization ID header is required",
		))
		return
	}

	// Call service
	serviceErr := h.service.DeleteAuthResource(ctx, authID, orgID)
	if serviceErr != nil {
		utils.SendError(w, serviceErr)
		return
	}

	// Send no content response
	w.WriteHeader(http.StatusNoContent)
}

// handleDeleteByConsent handles DELETE /consents/{consentId}/auth-resources
func (h *authResourceHandler) handleDeleteByConsent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract path parameters
	consentID := r.PathValue("consentId")
	if consentID == "" {
		utils.SendError(w, serviceerror.CustomServiceError(
			serviceerror.InvalidRequestError,
			"consent ID is required",
		))
		return
	}

	// Extract organization ID from header
	orgID := r.Header.Get(constants.HeaderOrgID)
	if orgID == "" {
		utils.SendError(w, serviceerror.CustomServiceError(
			serviceerror.InvalidRequestError,
			"organization ID header is required",
		))
		return
	}

	// Call service
	serviceErr := h.service.DeleteAuthResourcesByConsentID(ctx, consentID, orgID)
	if serviceErr != nil {
		utils.SendError(w, serviceErr)
		return
	}

	// Send no content response
	w.WriteHeader(http.StatusNoContent)
}

// handleUpdateAllStatusByConsent handles PATCH /consents/{consentId}/auth-resources/status
func (h *authResourceHandler) handleUpdateAllStatusByConsent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract path parameters
	consentID := r.PathValue("consentId")
	if consentID == "" {
		utils.SendError(w, serviceerror.CustomServiceError(
			serviceerror.InvalidRequestError,
			"consent ID is required",
		))
		return
	}

	// Extract organization ID from header
	orgID := r.Header.Get(constants.HeaderOrgID)
	if orgID == "" {
		utils.SendError(w, serviceerror.CustomServiceError(
			serviceerror.InvalidRequestError,
			"organization ID header is required",
		))
		return
	}

	// Parse request body
	var statusRequest struct {
		Status string `json:"status"`
	}
	if err := utils.DecodeJSONBody(r, &statusRequest); err != nil {
		utils.SendError(w, serviceerror.CustomServiceError(
			serviceerror.InvalidRequestError,
			fmt.Sprintf("invalid request body: %v", err),
		))
		return
	}

	// Call service
	serviceErr := h.service.UpdateAllStatusByConsentID(ctx, consentID, orgID, statusRequest.Status)
	if serviceErr != nil {
		utils.SendError(w, serviceErr)
		return
	}

	// Send no content response
	w.WriteHeader(http.StatusNoContent)
}

// sendError maps service errors to HTTP responses
