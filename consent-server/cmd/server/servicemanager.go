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

// Package-level service references for cleanup during shutdown
var (
	authResourceService   authresource.AuthResourceServiceInterface
	consentPurposeService consentpurpose.ConsentPurposeService
	consentService        consent.ConsentService
)

// registerServices registers all consent management services with the provided HTTP multiplexer.
// This follows Thunder's service manager pattern for clean separation of concerns.
func registerServices(
	mux *http.ServeMux,
	dbClient provider.DBClientInterface,
) {
	logger := log.GetLogger()

	// Create all stores first
	consentStore := consent.NewStore(dbClient)
	authResourceStore := authresource.NewStore(dbClient)
	consentPurposeStore := consentpurpose.NewStore(dbClient)

	// Create Store Registry with all stores
	storeRegistry := stores.NewStoreRegistry(
		dbClient,
		consentStore,
		authResourceStore,
		consentPurposeStore,
	)
	logger.Info("Store Registry initialized with all stores")

	// Initialize all services with the registry
	authResourceService = authresource.Initialize(mux, storeRegistry)
	logger.Info("AuthResource module initialized")

	consentPurposeService = consentpurpose.Initialize(mux, storeRegistry)
	logger.Info("ConsentPurpose module initialized")

	consentService = consent.Initialize(mux, storeRegistry)
	logger.Info("Consent module initialized")

	// Register health check endpoint
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy"}`))
	})
}

// unregisterServices performs cleanup of all services during shutdown.
// Currently a placeholder for future service cleanup needs.
func unregisterServices() {
	// Future: Add any service-specific cleanup logic here
	// e.g., closing connections, flushing caches, etc.
}
