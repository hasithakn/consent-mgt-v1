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

	// GET /consents/{consentId} - Get consent by ID
	mux.HandleFunc(middleware.WithCORS("GET /consents/{consentId}", handler.getConsent, corsOpts))

	// GET /consents - List/search consents
	mux.HandleFunc(middleware.WithCORS("GET /consents", handler.listConsents, corsOpts))

	// PUT /consents/{consentId} - Update consent
	mux.HandleFunc(middleware.WithCORS("PUT /consents/{consentId}", handler.updateConsent, corsOpts))

	// POST /consents/{consentId}/revoke - Revoke consent
	mux.HandleFunc(middleware.WithCORS("POST /consents/{consentId}/revoke", handler.revokeConsent, corsOpts))
}
