package main

import (
	"net/http"

	"github.com/wso2/consent-management-api/internal/authresource"
	"github.com/wso2/consent-management-api/internal/consent"
	"github.com/wso2/consent-management-api/internal/consentpurpose"
	"github.com/wso2/consent-management-api/internal/system/database/provider"
	"github.com/wso2/consent-management-api/internal/system/log"
	"github.com/wso2/consent-management-api/internal/system/stores"
)

// registerServices registers all consent management services with the provided HTTP multiplexer.
func registerServices(
	mux *http.ServeMux,
	dbClient provider.DBClientInterface,
) {
	logger := log.GetLogger()

	// Create Store Registry with all stores
	storeRegistry := stores.NewStoreRegistry(
		dbClient,
		consent.NewConsentStore(dbClient),
		authresource.NewAuthResourceStore(dbClient),
		consentpurpose.NewConsentPurposeStore(dbClient),
	)
	logger.Info("Store Registry initialized with all stores")

	// Initialize all services with the registry
	authresource.Initialize(mux, storeRegistry)
	logger.Info("AuthResource module initialized")

	consentpurpose.Initialize(mux, storeRegistry)
	logger.Info("ConsentPurpose module initialized")

	consent.Initialize(mux, storeRegistry)
	logger.Info("Consent module initialized")

	// TODO : refacter health check endpoint here.
	// Register health check endpoint
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy"}`))
	})
}

// TODO : compare with tunder and see if we need to add anything below mwthod. if not needed we can remove it
// unregisterServices performs cleanup of all services during shutdown.
// Currently a placeholder for future service cleanup needs.
func unregisterServices() {
	// Future: Add any service-specific cleanup logic here
	// e.g., closing connections, flushing caches, etc.
}
