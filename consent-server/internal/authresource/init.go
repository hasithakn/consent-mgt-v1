package authresource

import (
	"net/http"

	"github.com/wso2/consent-management-api/internal/system/database/provider"
	"github.com/wso2/consent-management-api/internal/system/middleware"
)

// Initialize creates and wires up all auth resource components and registers routes
func Initialize(mux *http.ServeMux, dbClient provider.DBClientInterface) AuthResourceServiceInterface {
	// Create layers: store -> service -> handler
	store := newAuthResourceStore(dbClient)
	service := newAuthResourceService(store)
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

	// Create authorization (POST /consents/{consentId}/authorizations)
	mux.HandleFunc(middleware.WithCORS(
		"POST /consents/{consentId}/authorizations",
		handler.handleCreate,
		corsOpts,
	))

	// List authorizations by consent (GET /consents/{consentId}/authorizations)
	mux.HandleFunc(middleware.WithCORS(
		"GET /consents/{consentId}/authorizations",
		handler.handleListByConsent,
		corsOpts,
	))

	// Get single authorization (GET /consents/{consentId}/authorizations/{authorizationId})
	mux.HandleFunc(middleware.WithCORS(
		"GET /consents/{consentId}/authorizations/{authorizationId}",
		handler.handleGet,
		corsOpts,
	))

	// Update authorization (PUT /consents/{consentId}/authorizations/{authorizationId})
	mux.HandleFunc(middleware.WithCORS(
		"PUT /consents/{consentId}/authorizations/{authorizationId}",
		handler.handleUpdate,
		corsOpts,
	))
}
