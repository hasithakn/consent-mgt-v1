package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/wso2/consent-management-api/internal/authresource"
	"github.com/wso2/consent-management-api/internal/config"
	"github.com/wso2/consent-management-api/internal/consent"
	"github.com/wso2/consent-management-api/internal/consentpurpose"
	"github.com/wso2/consent-management-api/internal/dao"
	"github.com/wso2/consent-management-api/internal/database"
	extensionclient "github.com/wso2/consent-management-api/internal/extension-client"
	"github.com/wso2/consent-management-api/internal/router"
	"github.com/wso2/consent-management-api/internal/service"
	"github.com/wso2/consent-management-api/internal/system/database/provider"
	"github.com/wso2/consent-management-api/internal/system/middleware"
)

// Version information (set by build script)
var (
	version   = "dev"
	buildDate = "unknown"
)

func main() {
	// Initialize logger
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetLevel(logrus.InfoLevel)

	logger.WithFields(logrus.Fields{
		"version":    version,
		"build_date": buildDate,
	}).Info("Starting Consent Management API Server...")

	// Load configuration
	// Priority: CONFIG_PATH env var > repository/conf/deployment.yaml > cmd/server/repository/conf/deployment.yaml
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		// Auto-discovery: will search in repository/conf/deployment.yaml first (production)
		// then cmd/server/repository/conf/deployment.yaml (development)
		configPath = ""
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		logger.WithError(err).Fatal("Failed to load configuration")
	}

	// Set log level from config
	if level, err := logrus.ParseLevel(cfg.Logging.Level); err == nil {
		logger.SetLevel(level)
	}

	logger.WithFields(logrus.Fields{
		"config_path": configPath,
		"log_level":   logger.GetLevel().String(),
	}).Info("Configuration loaded successfully")

	// Initialize database
	db, err := database.Initialize(&cfg.Database.Consent, logger)
	if err != nil {
		logger.WithError(err).Fatal("Failed to initialize database")
	}

	// Verify database connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.HealthCheck(ctx); err != nil {
		logger.WithError(err).Fatal("Database health check failed")
	}

	logger.Info("Database connection established successfully")

	// Create DBClient provider for new system architecture
	dbClient := provider.NewDBClient(db, "mysql", logger)

	// Initialize DAOs (old architecture - to be migrated)
	consentDAO := dao.NewConsentDAO(db)
	statusAuditDAO := dao.NewStatusAuditDAO(db)
	attributeDAO := dao.NewConsentAttributeDAO(db)
	authResourceDAO := dao.NewAuthResourceDAO(db)
	purposeDAO := dao.NewConsentPurposeDAO(db.DB)
	purposeAttributeDAO := dao.NewConsentPurposeAttributeDAO(db.DB)

	logger.Info("DAOs initialized successfully")

	// Initialize services (old architecture - to be migrated)
	consentService := service.NewConsentService(
		consentDAO,
		statusAuditDAO,
		attributeDAO,
		authResourceDAO,
		purposeDAO,
		db,
		logger,
	)

	authResourceServiceOld := service.NewAuthResourceService(
		authResourceDAO,
		consentDAO,
		db,
		logger,
	)

	purposeService := service.NewConsentPurposeService(
		purposeDAO,
		purposeAttributeDAO,
		consentDAO,
		db.DB,
		logger,
	)

	logger.Info("Services initialized successfully")

	// Initialize extension client
	extensionClient := extensionclient.NewExtensionClient(&cfg.ServiceExtension, logger)
	logger.WithField("enabled", extensionClient.IsExtensionEnabled()).Info("Extension client initialized")

	// Create http.ServeMux for new architecture
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy"}`))
	})

	// Initialize new AuthResource package (new architecture)
	_ = authresource.Initialize(mux, dbClient)
	logger.Info("AuthResource module initialized (new architecture)")

	// Initialize new ConsentPurpose package (new architecture)
	_ = consentpurpose.Initialize(mux, dbClient)
	logger.Info("ConsentPurpose module initialized (new architecture)")

	// Initialize new Consent package (new architecture)
	_ = consent.Initialize(mux, dbClient)
	logger.Info("Consent module initialized (new architecture)")

	// Setup Gin router for old endpoints (to be migrated)
	ginRouter := router.SetupRouter(consentService, authResourceServiceOld, purposeService, extensionClient)

	// Mount Gin router under /api/v1 (old architecture)
	mux.Handle("/api/v1/", http.StripPrefix("/api/v1", ginRouter))

	// Wrap with correlation ID middleware
	httpHandler := middleware.WrapWithCorrelationID(mux)

	// Configure HTTP server
	serverAddr := fmt.Sprintf("%s:%d", cfg.Server.Hostname, cfg.Server.Port)
	server := &http.Server{
		Addr:           serverAddr,
		Handler:        httpHandler,
		ReadTimeout:    15 * time.Second,
		WriteTimeout:   15 * time.Second,
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1 MB
	}

	// Start server in a goroutine
	go func() {
		logger.WithFields(logrus.Fields{
			"hostname": cfg.Server.Hostname,
			"port":     cfg.Server.Port,
			"addr":     serverAddr,
		}).Info("Starting HTTP server...")

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.WithError(err).Fatal("Failed to start server")
		}
	}()

	logger.WithField("address", serverAddr).Info("✓ Server is running")
	logger.Info("Press Ctrl+C to stop the server")

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.WithError(err).Fatal("Server forced to shutdown")
	}

	logger.Info("Server exited gracefully")
}
