package consentpurpose

import (
	"net/http"

	"github.com/wso2/consent-management-api/internal/system/constants"
	"github.com/wso2/consent-management-api/internal/system/database/provider"
	"github.com/wso2/consent-management-api/internal/system/middleware"
	"github.com/wso2/consent-management-api/internal/system/stores"
)

// NewStore creates and returns a new consent purpose store (exported for registry)
func NewStore(dbClient provider.DBClientInterface) interface{} {
	return newConsentPurposeStore(dbClient)
}

// Initialize sets up the consent purpose module and registers routes
func Initialize(mux *http.ServeMux, registry *stores.StoreRegistry) ConsentPurposeService {
	// Create service and handler using the registry
	service := newConsentPurposeService(registry)
	handler := newConsentPurposeHandler(service)

	// Register routes with CORS middleware
	registerRoutes(mux, handler)

	return service
}

// registerRoutes registers all consent purpose routes
func registerRoutes(mux *http.ServeMux, handler *consentPurposeHandler) {
	corsOptions := middleware.CORSOptions{
		AllowOrigin:  "*",
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"Content-Type", "x-org-id", "Authorization"},
	}

	// POST /api/v1/consent-purposes - Create purpose
	mux.HandleFunc(middleware.WithCORS("POST "+constants.APIBasePath+"/consent-purposes", handler.createPurpose, corsOptions))

	// GET /api/v1/consent-purposes/{purposeId} - Get purpose by ID
	mux.HandleFunc(middleware.WithCORS("GET "+constants.APIBasePath+"/consent-purposes/{purposeId}", handler.getPurpose, corsOptions))

	// GET /api/v1/consent-purposes - List purposes
	mux.HandleFunc(middleware.WithCORS("GET "+constants.APIBasePath+"/consent-purposes", handler.listPurposes, corsOptions))

	// PUT /api/v1/consent-purposes/{purposeId} - Update purpose
	mux.HandleFunc(middleware.WithCORS("PUT "+constants.APIBasePath+"/consent-purposes/{purposeId}", handler.updatePurpose, corsOptions))

	// DELETE /api/v1/consent-purposes/{purposeId} - Delete purpose
	mux.HandleFunc(middleware.WithCORS("DELETE "+constants.APIBasePath+"/consent-purposes/{purposeId}", handler.deletePurpose, corsOptions))
}
