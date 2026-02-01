package consentpurpose

import (
	"net/http"

	"github.com/wso2/consent-management-api/internal/system/constants"
	"github.com/wso2/consent-management-api/internal/system/stores"
)

// Initialize sets up the consent purpose module and registers routes
func Initialize(mux *http.ServeMux, registry *stores.StoreRegistry) ConsentPurposeService {
	// Create service and handler using the registry
	service := NewConsentPurposeService(registry)
	handler := newConsentPurposeHandler(service)

	// Register routes with CORS middleware
	registerRoutes(mux, handler)

	return service
}

// registerRoutes registers all consent purpose routes
func registerRoutes(mux *http.ServeMux, handler *consentPurposeHandler) {
	// POST /api/v1/consent-purposes - Create consent purpose
	mux.HandleFunc("POST "+constants.APIBasePath+"/consent-purposes", handler.createPurpose)

	// GET /api/v1/consent-purposes/{purposeId} - Get consent purpose by ID
	mux.HandleFunc("GET "+constants.APIBasePath+"/consent-purposes/{purposeId}", handler.getPurpose)

	// GET /api/v1/consent-purposes - List consent purposes
	mux.HandleFunc("GET "+constants.APIBasePath+"/consent-purposes", handler.listPurposes)

	// PUT /api/v1/consent-purposes/{purposeId} - Update consent purpose
	mux.HandleFunc("PUT "+constants.APIBasePath+"/consent-purposes/{purposeId}", handler.updatePurpose)

	// DELETE /api/v1/consent-purposes/{purposeId} - Delete consent purpose
	mux.HandleFunc("DELETE "+constants.APIBasePath+"/consent-purposes/{purposeId}", handler.deletePurpose)
}
