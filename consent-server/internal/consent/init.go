package consent

import (
	"net/http"

	"github.com/wso2/consent-management-api/internal/system/constants"
	"github.com/wso2/consent-management-api/internal/system/database/provider"
	"github.com/wso2/consent-management-api/internal/system/middleware"
	"github.com/wso2/consent-management-api/internal/system/stores"
)

// NewStore creates and returns a new consent store (exported for registry)
func NewStore(dbClient provider.DBClientInterface) interface{} {
	return newConsentStore(dbClient)
}

// Initialize sets up the consent module and registers routes
func Initialize(mux *http.ServeMux, registry *stores.StoreRegistry) ConsentService {
	// Create service and handler using the registry
	service := newConsentService(registry)
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

	// POST /api/v1/consents - Create consent
	mux.HandleFunc(middleware.WithCORS("POST "+constants.APIBasePath+"/consents", handler.createConsent, corsOpts))

	// GET /api/v1/consents/{consentId} - Get consent by ID
	mux.HandleFunc(middleware.WithCORS("GET "+constants.APIBasePath+"/consents/{consentId}", handler.getConsent, corsOpts))

	// GET /api/v1/consents - List/search consents
	mux.HandleFunc(middleware.WithCORS("GET "+constants.APIBasePath+"/consents", handler.listConsents, corsOpts))

	// PUT /api/v1/consents/{consentId} - Update consent
	mux.HandleFunc(middleware.WithCORS("PUT "+constants.APIBasePath+"/consents/{consentId}", handler.updateConsent, corsOpts))

	// POST /api/v1/consents/{consentId}/revoke - Revoke consent
	mux.HandleFunc(middleware.WithCORS("POST "+constants.APIBasePath+"/consents/{consentId}/revoke", handler.revokeConsent, corsOpts))
}
