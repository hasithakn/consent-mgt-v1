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

	// POST /purposes - Create purpose
	mux.HandleFunc(middleware.WithCORS("POST /purposes", handler.createPurpose, corsOptions))

	// GET /purposes/{id} - Get purpose by ID
	mux.HandleFunc(middleware.WithCORS("GET /purposes/{id}", handler.getPurpose, corsOptions))

	// GET /purposes - List purposes
	mux.HandleFunc(middleware.WithCORS("GET /purposes", handler.listPurposes, corsOptions))

	// PUT /purposes/{id} - Update purpose
	mux.HandleFunc(middleware.WithCORS("PUT /purposes/{id}", handler.updatePurpose, corsOptions))

	// DELETE /purposes/{id} - Delete purpose
	mux.HandleFunc(middleware.WithCORS("DELETE /purposes/{id}", handler.deletePurpose, corsOptions))
}
