package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/wso2/consent-management-api/internal/system/config"
	"github.com/wso2/consent-management-api/internal/system/database"
	"github.com/wso2/consent-management-api/internal/system/database/provider"
	"github.com/wso2/consent-management-api/internal/system/log"
	"github.com/wso2/consent-management-api/internal/system/middleware"
)

// Version information (set by build script)
var (
	version   = "dev"
	buildDate = "unknown"
)

func main() {
	// Initialize logger
	logger := log.GetLogger()

	logger.Info("Starting Consent Management API Server...", log.String("version", version),
		log.String("build_date", buildDate))

	// Load configuration
	// Priority: CONFIG_PATH env var > repository/conf/deployment.yaml > cmd/server/repository/conf/deployment.yaml
	configPath := os.Getenv("CONFIG_PATH")

	cfg, err := config.Load(configPath)
	if err != nil {
		logger.Fatal("Failed to load configuration", log.Error(err))
	}

	logger.Info("Configuration loaded successfully", log.String("config_path", configPath))

	// Update log level from configuration
	if cfg.Logging.Level != "" {
		if err := log.SetLogLevel(cfg.Logging.Level); err != nil {
			logger.Error("Failed to set log level from configuration", log.Error(err))
		} else {
			logger.Debug("Log level updated from configuration", log.String("level", cfg.Logging.Level))
		}
	}

	// Initialize database
	db, err := database.Initialize(&cfg.Database.Consent)
	if err != nil {
		logger.Fatal("Failed to initialize database", log.Error(err))
	}

	// Verify database connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.HealthCheck(ctx); err != nil {
		logger.Fatal("Database health check failed", log.Error(err))
	}

	logger.Info("Database connection established successfully")

	// Initialize DBProvider singleton
	provider.InitDBProvider(db)
	dbProvider := provider.GetDBProvider()

	// Get database client from provider
	dbClient, err := dbProvider.GetConsentDBClient()
	if err != nil {
		logger.Fatal("Failed to get database client", log.Error(err))
	}

	// Create HTTP mux
	mux := http.NewServeMux()

	// Register all services
	registerServices(mux, dbClient)

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
		logger.Info("Starting HTTP server...",
			log.String("hostname", cfg.Server.Hostname),
			log.Int("port", cfg.Server.Port),
			log.String("addr", serverAddr))

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", log.Error(err))
		}
	}()

	logger.Info("✓ Server is running", log.String("address", serverAddr))
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
		logger.Fatal("Server forced to shutdown", log.Error(err))
	}

	// Unregister services
	unregisterServices()
	logger.Info("Services unregistered")

	// Close database connections
	dbCloser := provider.GetDBProviderCloser()
	if err := dbCloser.Close(); err != nil {
		logger.Error("Error closing database connections", log.Error(err))
	} else {
		logger.Debug("Database connections closed successfully")
	}

	// Close the database connection itself
	if err := db.Close(); err != nil {
		logger.Error("Error closing database", log.Error(err))
	}

	logger.Info("Server exited gracefully")
}
