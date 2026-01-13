package consentpurposegroup

import (
	"net/http"

	"github.com/wso2/consent-management-api/internal/system/constants"
	"github.com/wso2/consent-management-api/internal/system/middleware"
	"github.com/wso2/consent-management-api/internal/system/stores"
)

// Initialize sets up the consent purpose group module and registers routes
func Initialize(mux *http.ServeMux, registry *stores.StoreRegistry) ConsentPurposeGroupService {
	// Create service and handler using the registry
	service := NewConsentPurposeGroupService(registry)
	handler := newConsentPurposeGroupHandler(service)

	// Register routes with CORS middleware
	registerRoutes(mux, handler)

	return service
}

// registerRoutes registers all purpose group routes
func registerRoutes(mux *http.ServeMux, handler *consentPurposeGroupHandler) {
	corsOptions := middleware.CORSOptions{
		AllowOrigin:  "*",
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"Content-Type", "x-org-id", "TPP-client-id", "Authorization"},
	}

	// POST /api/v1/consent-purpose-groups - Create purpose group
	mux.HandleFunc(middleware.WithCORS("POST "+constants.APIBasePath+"/consent-purpose-groups", handler.createPurposeGroup, corsOptions))

	// GET /api/v1/consent-purpose-groups/{groupId} - Get purpose group by ID
	mux.HandleFunc(middleware.WithCORS("GET "+constants.APIBasePath+"/consent-purpose-groups/{groupId}", handler.getPurposeGroup, corsOptions))

	// GET /api/v1/consent-purpose-groups - List purpose groups
	mux.HandleFunc(middleware.WithCORS("GET "+constants.APIBasePath+"/consent-purpose-groups", handler.listPurposeGroups, corsOptions))

	// PUT /api/v1/consent-purpose-groups/{groupId} - Update purpose group
	mux.HandleFunc(middleware.WithCORS("PUT "+constants.APIBasePath+"/consent-purpose-groups/{groupId}", handler.updatePurposeGroup, corsOptions))

	// DELETE /api/v1/consent-purpose-groups/{groupId} - Delete purpose group
	mux.HandleFunc(middleware.WithCORS("DELETE "+constants.APIBasePath+"/consent-purpose-groups/{groupId}", handler.deletePurposeGroup, corsOptions))
}
