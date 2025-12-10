package consentpurpose

import (
	"net/http"

	"github.com/wso2/consent-management-api/internal/system/database/provider"
	"github.com/wso2/consent-management-api/internal/system/middleware"
)

// Initialize sets up the consent purpose module and registers routes
func Initialize(mux *http.ServeMux, dbClient provider.DBClientInterface) ConsentPurposeService {
	// Create store, service, and handler
	store := newConsentPurposeStore(dbClient)
	service := newConsentPurposeService(store)
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

	// POST /consent-purposes - Create purpose
	mux.HandleFunc(middleware.WithCORS("POST /consent-purposes", handler.createPurpose, corsOptions))

	// GET /consent-purposes/{purposeId} - Get purpose by ID
	mux.HandleFunc(middleware.WithCORS("GET /consent-purposes/{purposeId}", handler.getPurpose, corsOptions))

	// GET /consent-purposes - List purposes
	mux.HandleFunc(middleware.WithCORS("GET /consent-purposes", handler.listPurposes, corsOptions))

	// PUT /consent-purposes/{purposeId} - Update purpose
	mux.HandleFunc(middleware.WithCORS("PUT /consent-purposes/{purposeId}", handler.updatePurpose, corsOptions))

	// DELETE /consent-purposes/{purposeId} - Delete purpose
	mux.HandleFunc(middleware.WithCORS("DELETE /consent-purposes/{purposeId}", handler.deletePurpose, corsOptions))
}
