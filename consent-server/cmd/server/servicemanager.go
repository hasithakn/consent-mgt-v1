package main

import (
	"net/http"

	"github.com/wso2/consent-management-api/internal/authresource"
	"github.com/wso2/consent-management-api/internal/consent"
	"github.com/wso2/consent-management-api/internal/consentelement"
	"github.com/wso2/consent-management-api/internal/consentpurpose"
	"github.com/wso2/consent-management-api/internal/system/healthcheck/handler"
	"github.com/wso2/consent-management-api/internal/system/log"
	"github.com/wso2/consent-management-api/internal/system/stores"
)

// registerServices registers all consent management services with the provided HTTP multiplexer.
func registerServices(
	mux *http.ServeMux,
) {
	logger := log.GetLogger()

	// Create Store Registry with all stores
	storeRegistry := stores.NewStoreRegistry(
		consent.NewConsentStore(),
		authresource.NewAuthResourceStore(),
		consentelement.NewConsentElementStore(),
		consentpurpose.NewPurposeStore(),
	)
	logger.Info("Store Registry initialized with all stores")

	// Initialize all services with the registry
	authresource.Initialize(mux, storeRegistry)
	logger.Info("AuthResource module initialized")

	consentelement.Initialize(mux, storeRegistry)
	logger.Info("ConsentPurpose module initialized")

	consentpurpose.Initialize(mux, storeRegistry)
	logger.Info("ConsentPurpose module initialized")

	consent.Initialize(mux, storeRegistry)
	logger.Info("Consent module initialized")

	// Register health check endpoints
	registerHealthCheckEndpoints(mux)
	logger.Info("Health check endpoints registered")
}

// registerHealthCheckEndpoints registers the health check endpoints.
func registerHealthCheckEndpoints(mux *http.ServeMux) {
	healthCheckHandler := handler.NewHealthCheckHandler()

	// Liveness endpoint - simple check if server is running
	mux.HandleFunc("GET /health/liveness", healthCheckHandler.HandleLivenessRequest)

	// Readiness endpoint - checks if server and dependencies are ready
	mux.HandleFunc("GET /health/readiness", healthCheckHandler.HandleReadinessRequest)

	// Legacy health endpoint (for backward compatibility)
	mux.HandleFunc("GET /health", healthCheckHandler.HandleLivenessRequest)
}

// TODO : compare with tunder and see if we need to add anything below mwthod. if not needed we can remove it
// unregisterServices performs cleanup of all services during shutdown.
// Currently a placeholder for future service cleanup needs.
func unregisterServices() {
	// Future: Add any service-specific cleanup logic here
	// e.g., closing connections, flushing caches, etc.
}
