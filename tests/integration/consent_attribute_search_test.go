package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wso2/consent-management-api/internal/config"
	"github.com/wso2/consent-management-api/internal/dao"
	"github.com/wso2/consent-management-api/internal/database"
	"github.com/wso2/consent-management-api/internal/handlers"
	"github.com/wso2/consent-management-api/internal/models"
	"github.com/wso2/consent-management-api/internal/service"
	"github.com/wso2/consent-management-api/pkg/utils"
)

// setupAttributeSearchTestEnvironment sets up test environment for attribute search
func setupAttributeSearchTestEnvironment(t *testing.T) (*gin.Engine, *service.ConsentService, *dao.ConsentDAO) {
	// Load configuration
	cfg, err := config.Load("../../configs/config.yaml")
	require.NoError(t, err, "Failed to load config")

	// Initialize logger
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// Initialize database
	db, err := database.Initialize(&cfg.Database, logger)
	require.NoError(t, err, "Failed to initialize database")

	// Create DAOs
	consentDAO := dao.NewConsentDAO(db)
	statusAuditDAO := dao.NewStatusAuditDAO(db)
	attributeDAO := dao.NewConsentAttributeDAO(db)
	authResourceDAO := dao.NewAuthResourceDAO(db)
	consentPurposeDAO := dao.NewConsentPurposeDAO(db.DB)

	// Create services
	consentService := service.NewConsentService(consentDAO, statusAuditDAO, attributeDAO, authResourceDAO, consentPurposeDAO, db, logger)
	consentPurposeService := service.NewConsentPurposeService(consentPurposeDAO, consentDAO, db.DB, logger)

	// Create router
	gin.SetMode(gin.TestMode)
	testRouter := gin.New()

	// Add middleware to set context values
	testRouter.Use(func(c *gin.Context) {
		if c.GetHeader("org-id") != "" {
			utils.SetContextValue(c, "orgID", c.GetHeader("org-id"))
		} else {
			utils.SetContextValue(c, "orgID", "TEST_ORG")
		}
		if c.GetHeader("client-id") != "" {
			utils.SetContextValue(c, "clientID", c.GetHeader("client-id"))
		} else {
			utils.SetContextValue(c, "clientID", "TEST_CLIENT")
		}
		c.Next()
	})

	// Create handler
	consentHandler := handlers.NewConsentHandler(consentService, consentPurposeService, nil)

	// API v1 routes
	v1 := testRouter.Group("/api/v1")
	{
		consents := v1.Group("/consents")
		{
			consents.POST("", consentHandler.CreateConsent)
			consents.GET("/attributes", consentHandler.SearchConsentsByAttribute)
		}
	}

	return testRouter, consentService, consentDAO
}

// cleanupConsents removes multiple test consents from database
func cleanupConsents(t *testing.T, consentDAO *dao.ConsentDAO, consentIDs ...string) {
	ctx := context.Background()
	for _, consentID := range consentIDs {
		if consentID != "" {
			err := consentDAO.Delete(ctx, consentID, "TEST_ORG")
			if err != nil {
				t.Logf("Warning: Failed to cleanup consent %s: %v", consentID, err)
			}
		}
	}
}

// TestSearchConsentsByAttribute_ByKeyOnly tests searching by attribute key only
func TestSearchConsentsByAttribute_ByKeyOnly(t *testing.T) {
	router, consentService, consentDAO := setupAttributeSearchTestEnvironment(t)

	// Create test consents with attributes
	consent1Request := &models.ConsentCreateRequest{
		ConsentPurpose: []models.ConsentPurposeItem{
			{Name: "test", Value: "data1"},
		},
		ConsentType:   "ACCOUNT_ACCESS",
		CurrentStatus: "AUTHORIZED",
		Attributes: map[string]string{
			"source":      "mobile-app",
			"environment": "production",
		},
	}

	consent2Request := &models.ConsentCreateRequest{
		ConsentPurpose: []models.ConsentPurposeItem{
			{Name: "test", Value: "data2"},
		},
		ConsentType:   "PAYMENT",
		CurrentStatus: "AUTHORIZED",
		Attributes: map[string]string{
			"source":  "web-app",
			"channel": "online",
		},
	}

	consent3Request := &models.ConsentCreateRequest{
		ConsentPurpose: []models.ConsentPurposeItem{
			{Name: "test", Value: "data3"},
		},
		ConsentType:   "ACCOUNT_ACCESS",
		CurrentStatus: "AUTHORIZED",
		Attributes: map[string]string{
			"channel": "mobile",
		},
	}

	consent1, err := consentService.CreateConsent(context.Background(), consent1Request, "TEST_CLIENT", "TEST_ORG")
	require.NoError(t, err)
	defer cleanupConsents(t, consentDAO, consent1.ConsentID)

	consent2, err := consentService.CreateConsent(context.Background(), consent2Request, "TEST_CLIENT", "TEST_ORG")
	require.NoError(t, err)
	defer cleanupConsents(t, consentDAO, consent2.ConsentID)

	consent3, err := consentService.CreateConsent(context.Background(), consent3Request, "TEST_CLIENT", "TEST_ORG")
	require.NoError(t, err)
	defer cleanupConsents(t, consentDAO, consent3.ConsentID)

	t.Logf("Created test consents: %s, %s, %s", consent1.ConsentID, consent2.ConsentID, consent3.ConsentID)

	// Search by key "source" - should return consent1 and consent2
	req := httptest.NewRequest(http.MethodGet, "/api/v1/consents/attributes?key=source", nil)
	req.Header.Set("org-id", "TEST_ORG")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response models.ConsentAttributeSearchResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, 2, response.Count, "Expected 2 consents with 'source' attribute")
	assert.Len(t, response.ConsentIDs, 2)
	assert.Contains(t, response.ConsentIDs, consent1.ConsentID)
	assert.Contains(t, response.ConsentIDs, consent2.ConsentID)
	assert.NotContains(t, response.ConsentIDs, consent3.ConsentID)

	t.Logf("Search by key 'source' returned: %v", response.ConsentIDs)
}

// TestSearchConsentsByAttribute_ByKeyAndValue tests searching by key-value pair
func TestSearchConsentsByAttribute_ByKeyAndValue(t *testing.T) {
	router, consentService, consentDAO := setupAttributeSearchTestEnvironment(t)

	// Create test consents
	consent1Request := &models.ConsentCreateRequest{
		ConsentPurpose: []models.ConsentPurposeItem{
			{Name: "test", Value: "data1"},
		},
		ConsentType:   "ACCOUNT_ACCESS",
		CurrentStatus: "AUTHORIZED",
		Attributes: map[string]string{
			"source":      "mobile-app",
			"environment": "production",
		},
	}

	consent2Request := &models.ConsentCreateRequest{
		ConsentPurpose: []models.ConsentPurposeItem{
			{Name: "test", Value: "data2"},
		},
		ConsentType:   "PAYMENT",
		CurrentStatus: "AUTHORIZED",
		Attributes: map[string]string{
			"source":      "mobile-app",
			"environment": "staging",
		},
	}

	consent3Request := &models.ConsentCreateRequest{
		ConsentPurpose: []models.ConsentPurposeItem{
			{Name: "test", Value: "data3"},
		},
		ConsentType:   "ACCOUNT_ACCESS",
		CurrentStatus: "AUTHORIZED",
		Attributes: map[string]string{
			"source": "web-app",
		},
	}

	consent1, err := consentService.CreateConsent(context.Background(), consent1Request, "TEST_CLIENT", "TEST_ORG")
	require.NoError(t, err)
	defer cleanupConsents(t, consentDAO, consent1.ConsentID)

	consent2, err := consentService.CreateConsent(context.Background(), consent2Request, "TEST_CLIENT", "TEST_ORG")
	require.NoError(t, err)
	defer cleanupConsents(t, consentDAO, consent2.ConsentID)

	consent3, err := consentService.CreateConsent(context.Background(), consent3Request, "TEST_CLIENT", "TEST_ORG")
	require.NoError(t, err)
	defer cleanupConsents(t, consentDAO, consent3.ConsentID)

	// Search by key="source" and value="mobile-app" - should return consent1 and consent2
	req := httptest.NewRequest(http.MethodGet, "/api/v1/consents/attributes?key=source&value=mobile-app", nil)
	req.Header.Set("org-id", "TEST_ORG")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response models.ConsentAttributeSearchResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, 2, response.Count)
	assert.Len(t, response.ConsentIDs, 2)
	assert.Contains(t, response.ConsentIDs, consent1.ConsentID)
	assert.Contains(t, response.ConsentIDs, consent2.ConsentID)
	assert.NotContains(t, response.ConsentIDs, consent3.ConsentID)

	t.Logf("Search by source=mobile-app returned: %v", response.ConsentIDs)
}

// TestSearchConsentsByAttribute_NoResults tests search with no matching results
func TestSearchConsentsByAttribute_NoResults(t *testing.T) {
	router, consentService, consentDAO := setupAttributeSearchTestEnvironment(t)

	// Create a test consent with different attributes
	consentRequest := &models.ConsentCreateRequest{
		ConsentPurpose: []models.ConsentPurposeItem{
			{Name: "test", Value: "data"},
		},
		ConsentType:   "ACCOUNT_ACCESS",
		CurrentStatus: "AUTHORIZED",
		Attributes: map[string]string{
			"source": "mobile-app",
		},
	}

	consent, err := consentService.CreateConsent(context.Background(), consentRequest, "TEST_CLIENT", "TEST_ORG")
	require.NoError(t, err)
	defer cleanupConsents(t, consentDAO, consent.ConsentID)

	// Search for non-existent attribute
	req := httptest.NewRequest(http.MethodGet, "/api/v1/consents/attributes?key=non_existent_key", nil)
	req.Header.Set("org-id", "TEST_ORG")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response models.ConsentAttributeSearchResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, 0, response.Count)
	assert.Empty(t, response.ConsentIDs)

	t.Logf("Search for non-existent key returned empty results")
}

// TestSearchConsentsByAttribute_MissingKey tests error when key parameter is missing
func TestSearchConsentsByAttribute_MissingKey(t *testing.T) {
	router, _, _ := setupAttributeSearchTestEnvironment(t)

	// Request without key parameter
	req := httptest.NewRequest(http.MethodGet, "/api/v1/consents/attributes", nil)
	req.Header.Set("org-id", "TEST_ORG")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response["details"], "key parameter is required")
}

// TestSearchConsentsByAttribute_OrganizationIsolation tests org isolation
func TestSearchConsentsByAttribute_OrganizationIsolation(t *testing.T) {
	router, consentService, consentDAO := setupAttributeSearchTestEnvironment(t)

	// Create consent for ORG1
	consent1Request := &models.ConsentCreateRequest{
		ConsentPurpose: []models.ConsentPurposeItem{
			{Name: "test", Value: "data1"},
		},
		ConsentType:   "ACCOUNT_ACCESS",
		CurrentStatus: "AUTHORIZED",
		Attributes: map[string]string{
			"source": "mobile-app",
		},
	}

	consent1, err := consentService.CreateConsent(context.Background(), consent1Request, "TEST_CLIENT", "TEST_ORG")
	require.NoError(t, err)
	defer cleanupConsents(t, consentDAO, consent1.ConsentID)

	// Search from different org - should not find the consent
	req := httptest.NewRequest(http.MethodGet, "/api/v1/consents/attributes?key=source", nil)
	req.Header.Set("org-id", "DIFFERENT_ORG")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response models.ConsentAttributeSearchResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, 0, response.Count, "Should not find consents from different organization")
	assert.Empty(t, response.ConsentIDs)

	t.Logf("Organization isolation test passed - no cross-org data leakage")
}

// TestSearchConsentsByAttribute_EmptyValue tests search with empty value parameter
func TestSearchConsentsByAttribute_EmptyValue(t *testing.T) {
	router, consentService, consentDAO := setupAttributeSearchTestEnvironment(t)

	// Create consent
	consentRequest := &models.ConsentCreateRequest{
		ConsentPurpose: []models.ConsentPurposeItem{
			{Name: "test", Value: "data"},
		},
		ConsentType:   "ACCOUNT_ACCESS",
		CurrentStatus: "AUTHORIZED",
		Attributes: map[string]string{
			"source": "mobile-app",
		},
	}

	consent, err := consentService.CreateConsent(context.Background(), consentRequest, "TEST_CLIENT", "TEST_ORG")
	require.NoError(t, err)
	defer cleanupConsents(t, consentDAO, consent.ConsentID)

	// Search with key and empty value - should work as key-only search
	req := httptest.NewRequest(http.MethodGet, "/api/v1/consents/attributes?key=source&value=", nil)
	req.Header.Set("org-id", "TEST_ORG")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response models.ConsentAttributeSearchResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, 1, response.Count)
	assert.Contains(t, response.ConsentIDs, consent.ConsentID)

	t.Logf("Empty value parameter treated as key-only search")
}
