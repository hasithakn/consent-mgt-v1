package main

import (
	"net/http"

	"github.com/wso2/consent-management-api/internal/authresource"
	"github.com/wso2/consent-management-api/internal/consent"
	"github.com/wso2/consent-management-api/internal/consentpurpose"
	"github.com/wso2/consent-management-api/internal/system/database"
	"github.com/wso2/consent-management-api/internal/system/database/provider"
	"github.com/wso2/consent-management-api/internal/system/log"
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
	db *database.DB,
) {
	logger := log.GetLogger()

	// Initialize AuthResource module (returns service and store)
	// Store is used by consent module for cross-module transactions
	var authStore consent.AuthResourceStore
	authResourceService, authStore = authresource.Initialize(mux, dbClient)
	logger.Info("AuthResource module initialized")

	// Initialize ConsentPurpose module (returns service and store)
	// Store is used by consent module for cross-module transactions
	var purposeStore consent.ConsentPurposeStore
	consentPurposeService, purposeStore = consentpurpose.Initialize(mux, dbClient)
	logger.Info("ConsentPurpose module initialized")

	// Initialize Consent module (needs stores from other modules for transactions)
	consentService = consent.Initialize(mux, dbClient, db, authStore, purposeStore)
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
