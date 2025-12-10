package consent

import (
	"net/http"

	"github.com/wso2/consent-management-api/internal/system/constants"
	"github.com/wso2/consent-management-api/internal/system/database"
	"github.com/wso2/consent-management-api/internal/system/database/provider"
	"github.com/wso2/consent-management-api/internal/system/middleware"
)

// Initialize sets up the consent module and registers routes
// Requires auth resource and consent purpose stores for transactional operations
func Initialize(mux *http.ServeMux, dbClient provider.DBClientInterface, db *database.DB, authResourceStore AuthResourceStore, consentPurposeStore ConsentPurposeStore) ConsentService {
	// Create store, service, and handler
	store := newConsentStore(dbClient)
	service := newConsentService(store, authResourceStore, consentPurposeStore, db)
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
