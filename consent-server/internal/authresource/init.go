package authresource

import (
	"net/http"

	"github.com/wso2/consent-management-api/internal/system/constants"
	"github.com/wso2/consent-management-api/internal/system/stores"
)

// Initialize sets up the auth resource module and registers routes
func Initialize(mux *http.ServeMux, registry *stores.StoreRegistry) AuthResourceServiceInterface {
	// Create service and handler using the registry
	service := newAuthResourceService(registry)
	handler := newAuthResourceHandler(service)

	// Register routes
	registerRoutes(mux, handler)

	return service
}

// registerRoutes registers all auth resource HTTP routes
func registerRoutes(mux *http.ServeMux, handler *authResourceHandler) {
	// Create authorization (POST /api/v1/consents/{consentId}/authorizations)
	mux.HandleFunc(
		"POST "+constants.APIBasePath+"/consents/{consentId}/authorizations",
		handler.handleCreate,
	)

	// List authorizations by consent (GET /api/v1/consents/{consentId}/authorizations)
	mux.HandleFunc(
		"GET "+constants.APIBasePath+"/consents/{consentId}/authorizations",
		handler.handleListByConsent,
	)

	// Get single authorization (GET /api/v1/consents/{consentId}/authorizations/{authorizationId})
	mux.HandleFunc(
		"GET "+constants.APIBasePath+"/consents/{consentId}/authorizations/{authorizationId}",
		handler.handleGet,
	)

	// Update authorization (PUT /api/v1/consents/{consentId}/authorizations/{authorizationId})
	mux.HandleFunc(
		"PUT "+constants.APIBasePath+"/consents/{consentId}/authorizations/{authorizationId}",
		handler.handleUpdate,
	)
}
