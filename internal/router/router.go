package router

import (
	"github.com/gin-gonic/gin"

	"github.com/wso2/consent-management-api/internal/handlers"
	"github.com/wso2/consent-management-api/internal/service"
	"github.com/wso2/consent-management-api/pkg/utils"
)

// SetupRouter configures all API routes
func SetupRouter(
	consentService *service.ConsentService,
	authResourceService *service.AuthResourceService,
	purposeService *service.ConsentPurposeService,
) *gin.Engine {
	router := gin.Default()

	// Global middleware to extract headers and set context
	router.Use(func(c *gin.Context) {
		// Extract and set org ID
		orgID := c.GetHeader("org-id")
		if orgID != "" {
			utils.SetContextValue(c, "orgID", orgID)
		}

		// Extract and set client ID
		clientID := c.GetHeader("client-id")
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
	consentHandler := handlers.NewConsentHandler(consentService)
	authResourceHandler := handlers.NewAuthResourceHandler(authResourceService)
	purposeHandler := handlers.NewConsentPurposeHandler(purposeService)

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Consent purpose routes
		v1.POST("/consent-purposes", purposeHandler.CreateConsentPurposes)
		v1.GET("/consent-purposes", purposeHandler.ListConsentPurposes)
		v1.GET("/consent-purposes/:purposeId", purposeHandler.GetConsentPurpose)
		v1.PUT("/consent-purposes/:purposeId", purposeHandler.UpdateConsentPurpose)
		v1.DELETE("/consent-purposes/:purposeId", purposeHandler.DeleteConsentPurpose)

		// Consent routes
		consents := v1.Group("/consents")
		{
			consents.POST("", consentHandler.CreateConsent)
			consents.GET("/:consentId", consentHandler.GetConsent)
			consents.PUT("/:consentId", consentHandler.UpdateConsent)

			// Authorization resource routes under consent
			consents.POST("/:consentId/authorizations", authResourceHandler.CreateAuthResource)
			consents.GET("/:consentId/authorizations/:authId", authResourceHandler.GetAuthResource)
			consents.PUT("/:consentId/authorizations/:authId", authResourceHandler.UpdateAuthResource)
		}
	}

	return router
}
