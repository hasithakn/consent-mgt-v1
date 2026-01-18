package consentelement

import (
	"net/http"

	"github.com/wso2/consent-management-api/internal/system/constants"
	"github.com/wso2/consent-management-api/internal/system/middleware"
	"github.com/wso2/consent-management-api/internal/system/stores"
)

// Initialize sets up the consent element module and registers routes
func Initialize(mux *http.ServeMux, registry *stores.StoreRegistry) ConsentElementService {
	// Create service and handler using the registry
	service := newConsentElementService(registry)
	handler := newConsentElementHandler(service)

	// Register routes with CORS middleware
	registerRoutes(mux, handler)

	return service
}

// registerRoutes registers all consent element routes
func registerRoutes(mux *http.ServeMux, handler *consentElementHandler) {
	corsOptions := middleware.CORSOptions{
		AllowOrigin:  "*",
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"Content-Type", "x-org-id", "Authorization"},
	}

	// POST /api/v1/consent-elements - Create element
	mux.HandleFunc(middleware.WithCORS("POST "+constants.APIBasePath+"/consent-elements", handler.createElement, corsOptions))

	// GET /api/v1/consent-elements/{elementId} - Get element by ID
	mux.HandleFunc(middleware.WithCORS("GET "+constants.APIBasePath+"/consent-elements/{elementId}", handler.getElement, corsOptions))

	// GET /api/v1/consent-elements - List elements
	mux.HandleFunc(middleware.WithCORS("GET "+constants.APIBasePath+"/consent-elements", handler.listElements, corsOptions))

	// POST /api/v1/consent-elements/validate - Validate element names
	mux.HandleFunc(middleware.WithCORS("POST "+constants.APIBasePath+"/consent-elements/validate", handler.validateElements, corsOptions))

	// PUT /api/v1/consent-elements/{elementId} - Update element
	mux.HandleFunc(middleware.WithCORS("PUT "+constants.APIBasePath+"/consent-elements/{elementId}", handler.updateElement, corsOptions))

	// DELETE /api/v1/consent-elements/{elementId} - Delete element
	mux.HandleFunc(middleware.WithCORS("DELETE "+constants.APIBasePath+"/consent-elements/{elementId}", handler.deleteElement, corsOptions))
}
