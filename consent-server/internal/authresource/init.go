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

	// Create auth resource (POST /consents/{consentId}/auth-resources)
	mux.HandleFunc(middleware.WithCORS(
		"POST /consents/{consentId}/auth-resources",
		handler.handleCreate,
		corsOpts,
	))

	// List auth resources by consent (GET /consents/{consentId}/auth-resources)
	mux.HandleFunc(middleware.WithCORS(
		"GET /consents/{consentId}/auth-resources",
		handler.handleListByConsent,
		corsOpts,
	))

	// Delete all auth resources by consent (DELETE /consents/{consentId}/auth-resources)
	mux.HandleFunc(middleware.WithCORS(
		"DELETE /consents/{consentId}/auth-resources",
		handler.handleDeleteByConsent,
		corsOpts,
	))

	// Update all statuses by consent (PATCH /consents/{consentId}/auth-resources/status)
	mux.HandleFunc(middleware.WithCORS(
		"PATCH /consents/{consentId}/auth-resources/status",
		handler.handleUpdateAllStatusByConsent,
		corsOpts,
	))

	// Get single auth resource (GET /consents/{consentId}/auth-resources/{authId})
	mux.HandleFunc(middleware.WithCORS(
		"GET /consents/{consentId}/auth-resources/{authId}",
		handler.handleGet,
		corsOpts,
	))

	// Update auth resource (PUT /consents/{consentId}/auth-resources/{authId})
	mux.HandleFunc(middleware.WithCORS(
		"PUT /consents/{consentId}/auth-resources/{authId}",
		handler.handleUpdate,
		corsOpts,
	))

	// Delete auth resource (DELETE /consents/{consentId}/auth-resources/{authId})
	mux.HandleFunc(middleware.WithCORS(
		"DELETE /consents/{consentId}/auth-resources/{authId}",
		handler.handleDelete,
		corsOpts,
	))

	// Update auth resource status (PATCH /consents/{consentId}/auth-resources/{authId}/status)
	mux.HandleFunc(middleware.WithCORS(
		"PATCH /consents/{consentId}/auth-resources/{authId}/status",
		handler.handleUpdateStatus,
		corsOpts,
	))

	// List auth resources by user (GET /auth-resources?userId=xxx)
	mux.HandleFunc(middleware.WithCORS(
		"GET /auth-resources",
		handler.handleListByUser,
		corsOpts,
	))
}
