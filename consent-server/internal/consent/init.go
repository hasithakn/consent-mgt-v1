package consent

import (
	"net/http"

	"github.com/wso2/consent-management-api/internal/system/constants"
	"github.com/wso2/consent-management-api/internal/system/stores"
)

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
	// POST /api/v1/consents - Create consent
	mux.HandleFunc("POST "+constants.APIBasePath+"/consents", handler.createConsent)

	// GET /api/v1/consents/{consentId} - Get consent by ID
	mux.HandleFunc("GET "+constants.APIBasePath+"/consents/{consentId}", handler.getConsent)

	// GET /api/v1/consents - List/search consents
	mux.HandleFunc("GET "+constants.APIBasePath+"/consents", handler.listConsents)

	// PUT /api/v1/consents/{consentId} - Update consent
	mux.HandleFunc("PUT "+constants.APIBasePath+"/consents/{consentId}", handler.updateConsent)

	// PUT /api/v1/consents/{consentId}/revoke - Revoke consent
	mux.HandleFunc("PUT "+constants.APIBasePath+"/consents/{consentId}/revoke", handler.revokeConsent)

	// POST /api/v1/consents/validate - Validate consent
	mux.HandleFunc("POST "+constants.APIBasePath+"/consents/validate", handler.validateConsent)

	// GET /api/v1/consents/attributes - Search consents by attribute
	mux.HandleFunc("GET "+constants.APIBasePath+"/consents/attributes", handler.searchConsentsByAttribute)
}
