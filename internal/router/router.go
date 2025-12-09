package router

import (
	"github.com/gin-gonic/gin"

	client "github.com/wso2/consent-management-api/internal/extension-client"
	"github.com/wso2/consent-management-api/internal/handlers"
	"github.com/wso2/consent-management-api/internal/service"
	"github.com/wso2/consent-management-api/internal/utils"
)

// SetupRouter configures all API routes
func SetupRouter(
	consentService *service.ConsentService,
	authResourceService *service.AuthResourceService,
	purposeService *service.ConsentPurposeService,
	extensionClient *client.ExtensionClient,
) *gin.Engine {
	router := gin.Default()

	// Global middleware to extract headers and set context.
	// Accept both the new header 'tpp-client-id' and the legacy 'client-id' for tests/clients.
	router.Use(func(c *gin.Context) {
		// Extract and set org ID
		orgID := c.GetHeader("org-id")
		if orgID != "" {
			utils.SetContextValue(c, "orgID", orgID)
		}

		// Prefer the standard header 'tpp-client-id', but fall back to legacy 'client-id'
		clientID := c.GetHeader("tpp-client-id")
		if clientID == "" {
			clientID = c.GetHeader("client-id")
		}
		if clientID != "" {
			utils.SetContextValue(c, "clientID", clientID)
		}

		c.Next()
	})

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy"})
	})

	// Create handlers
	consentHandler := handlers.NewConsentHandler(consentService, purposeService, extensionClient)
	authResourceHandler := handlers.NewAuthResourceHandler(authResourceService)
	purposeHandler := handlers.NewConsentPurposeHandler(purposeService)

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Consent purpose routes
		v1.POST("/consent-purposes", purposeHandler.CreateConsentPurposes)
		v1.GET("/consent-purposes", purposeHandler.ListConsentPurposes)
		v1.POST("/consent-purposes/validate", purposeHandler.ValidateConsentPurposes)
		v1.GET("/consent-purposes/:purposeId", purposeHandler.GetConsentPurpose)
		v1.PUT("/consent-purposes/:purposeId", purposeHandler.UpdateConsentPurpose)
		v1.DELETE("/consent-purposes/:purposeId", purposeHandler.DeleteConsentPurpose)

		// Consent routes
		consents := v1.Group("/consents")
		{
			consents.POST("", consentHandler.CreateConsent)

			// Attribute search endpoint (must be before /:consentId)
			consents.GET("/attributes", consentHandler.SearchConsentsByAttribute)

			// Validation endpoint (must be before /:consentId)
			consents.POST("/validate", consentHandler.Validate)

			// Specific paths before parameterized paths
			consents.PUT("/:consentId/revoke", consentHandler.RevokeConsent)

			// General consent operations
			consents.GET("/:consentId", consentHandler.GetConsent)
			consents.PUT("/:consentId", consentHandler.UpdateConsent)
			consents.DELETE("/:consentId", consentHandler.DeleteConsent)

			// Authorization resource routes under consent
			consents.POST("/:consentId/authorizations", authResourceHandler.CreateAuthResource)
			consents.GET("/:consentId/authorizations/:authId", authResourceHandler.GetAuthResource)
			consents.PUT("/:consentId/authorizations/:authId", authResourceHandler.UpdateAuthResource)
		}
	}

	return router
}
