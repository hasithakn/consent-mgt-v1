package authresource

import (
	"net/http"

	"github.com/wso2/consent-management-api/internal/system/constants"
	"github.com/wso2/consent-management-api/internal/system/middleware"
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

// registerRoutes registers all auth resource HTTP routes with CORS support
func registerRoutes(mux *http.ServeMux, handler *authResourceHandler) {
	// CORS configuration
	corsOpts := middleware.CORSOptions{
		AllowOrigin:      "*",
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type", "Authorization", "X-Organization-ID", "X-Correlation-ID"},
		AllowCredentials: true,
	}

	// Create authorization (POST /api/v1/consents/{consentId}/authorizations)
	mux.HandleFunc(middleware.WithCORS(
		"POST "+constants.APIBasePath+"/consents/{consentId}/authorizations",
		handler.handleCreate,
		corsOpts,
	))

	// List authorizations by consent (GET /api/v1/consents/{consentId}/authorizations)
	mux.HandleFunc(middleware.WithCORS(
		"GET "+constants.APIBasePath+"/consents/{consentId}/authorizations",
		handler.handleListByConsent,
		corsOpts,
	))

	// Get single authorization (GET /api/v1/consents/{consentId}/authorizations/{authorizationId})
	mux.HandleFunc(middleware.WithCORS(
		"GET "+constants.APIBasePath+"/consents/{consentId}/authorizations/{authorizationId}",
		handler.handleGet,
		corsOpts,
	))

	// Update authorization (PUT /api/v1/consents/{consentId}/authorizations/{authorizationId})
	mux.HandleFunc(middleware.WithCORS(
		"PUT "+constants.APIBasePath+"/consents/{consentId}/authorizations/{authorizationId}",
		handler.handleUpdate,
		corsOpts,
	))
}
