package consent

import (
	"net/http"

	"github.com/wso2/consent-management-api/internal/system/database/provider"
	"github.com/wso2/consent-management-api/internal/system/middleware"
)

// Initialize sets up the consent module and registers routes
func Initialize(mux *http.ServeMux, dbClient provider.DBClientInterface) ConsentService {
	// Create store, service, and handler
	store := newConsentStore(dbClient)
	service := newConsentService(store)
	handler := newConsentHandler(service)

	// Register routes with CORS middleware
	registerRoutes(mux, handler)

	return service
}

// registerRoutes registers all consent routes
func registerRoutes(mux *http.ServeMux, handler *consentHandler) {
	corsOpts := middleware.CORSOptions{
		AllowOrigin:      "*",
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type", "Authorization", "X-Organization-ID", "X-Correlation-ID"},
		AllowCredentials: true,
	}

	// POST /consents - Create consent
	mux.HandleFunc(middleware.WithCORS("POST /consents", handler.createConsent, corsOpts))

	// GET /consents/{id} - Get consent by ID
	mux.HandleFunc(middleware.WithCORS("GET /consents/{id}", handler.getConsent, corsOpts))

	// GET /consents - List consents
	mux.HandleFunc(middleware.WithCORS("GET /consents", handler.listConsents, corsOpts))

	// PUT /consents/{id} - Update consent
	mux.HandleFunc(middleware.WithCORS("PUT /consents/{id}", handler.updateConsent, corsOpts))

	// PATCH /consents/{id}/status - Update consent status
	mux.HandleFunc(middleware.WithCORS("PATCH /consents/{id}/status", handler.updateConsentStatus, corsOpts))

	// DELETE /consents/{id} - Delete consent
	mux.HandleFunc(middleware.WithCORS("DELETE /consents/{id}", handler.deleteConsent, corsOpts))
}
