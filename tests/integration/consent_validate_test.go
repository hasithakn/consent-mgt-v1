package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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

// cleanupConsent removes test consent data from database
func cleanupConsent(t *testing.T, consentDAO *dao.ConsentDAO, consentID string) {
	if consentID == "" {
		return
	}
	ctx := context.Background()
	err := consentDAO.Delete(ctx, consentID, "TEST_ORG")
	if err != nil {
		t.Logf("Warning: Failed to cleanup consent %s: %v", consentID, err)
	}
}

// setupValidateTestEnvironment sets up test environment for validate API tests
func setupValidateTestEnvironment(t *testing.T) (*gin.Engine, *service.ConsentService, *dao.ConsentDAO) {
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
		v1.POST("/validate", consentHandler.Validate)

		// Also add consent routes for setup
		consents := v1.Group("/consents")
		{
			consents.POST("", consentHandler.CreateConsent)
			consents.GET("/:consentId", consentHandler.GetConsent)
			consents.PUT("/:consentId", consentHandler.UpdateConsent)
		}
	}

	return testRouter, consentService, consentDAO
}

// TestValidateConsent_Success tests successful consent validation
func TestValidateConsent_Success(t *testing.T) {
	router, consentService, consentDAO := setupValidateTestEnvironment(t)
	defer cleanupConsent(t, consentDAO, "")

	// Create a test consent with active status
	cfg := config.Get()
	activeStatus := cfg.Consent.StatusMappings.ActiveStatus

	validityTime := time.Now().Add(24 * time.Hour).UnixNano() / int64(time.Millisecond)

	createRequest := &models.ConsentCreateRequest{
		ConsentPurpose: []models.ConsentPurposeItem{
			{Name: "test", Value: "data"},
		},
		ConsentType:   "ACCOUNT_ACCESS",
		CurrentStatus: activeStatus,
		ValidityTime:  &validityTime,
	}

	consent, err := consentService.CreateConsent(context.Background(), createRequest, "TEST_CLIENT", "TEST_ORG")
	require.NoError(t, err)
	require.NotNil(t, consent)
	defer cleanupConsent(t, consentDAO, consent.ConsentID)

	t.Logf("Created test consent: %s with status: %s", consent.ConsentID, consent.CurrentStatus)

	// Prepare validate request
	validateRequest := models.ValidateRequest{
		Headers: map[string]interface{}{
			"authorization": "Bearer test-token",
		},
		Payload: map[string]interface{}{
			"test": "payload",
		},
		ElectedResource: "/accounts/123",
		ConsentID:       consent.ConsentID,
		UserID:          "test-user-123",
		ClientID:        "TEST_CLIENT",
		ResourceParams: struct {
			Resource   string `json:"resource"`
			HTTPMethod string `json:"httpMethod"`
			Context    string `json:"context"`
		}{
			Resource:   "/accounts/123",
			HTTPMethod: "GET",
			Context:    "account-access",
		},
	}

	requestBody, err := json.Marshal(validateRequest)
	require.NoError(t, err)

	// Make request
	req := httptest.NewRequest(http.MethodPost, "/api/v1/validate", bytes.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("org-id", "TEST_ORG")
	req.Header.Set("client-id", "TEST_CLIENT")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code, "Expected 200 OK for valid consent")

	var response models.ValidateResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response.IsValid, "Expected isValid to be true")
	assert.Empty(t, response.ErrorCode, "Expected no error code")
	assert.Empty(t, response.ErrorMessage, "Expected no error message")
	assert.NotNil(t, response.ConsentInformation, "Expected consent information")

	// Verify consent information contains all fields
	assert.Equal(t, consent.ConsentID, response.ConsentInformation["consentId"])
	assert.Equal(t, activeStatus, response.ConsentInformation["status"])
	assert.Equal(t, "ACCOUNT_ACCESS", response.ConsentInformation["consentType"])
	assert.NotNil(t, response.ConsentInformation["receipt"])
	assert.NotNil(t, response.ConsentInformation["createdTime"])
	assert.NotNil(t, response.ConsentInformation["updatedTime"])

	t.Logf("Validate response: %+v", response)
}

// TestValidateConsent_InvalidStatus tests validation with non-active status
func TestValidateConsent_InvalidStatus(t *testing.T) {
	router, consentService, consentDAO := setupValidateTestEnvironment(t)
	defer cleanupConsent(t, consentDAO, "")

	// Create a test consent with revoked status
	cfg := config.Get()
	revokedStatus := cfg.Consent.StatusMappings.RevokedStatus

	createRequest := &models.ConsentCreateRequest{
		ConsentPurpose: []models.ConsentPurposeItem{
			{Name: "test", Value: "data"},
		},
		ConsentType:   "ACCOUNT_ACCESS",
		CurrentStatus: revokedStatus,
	}

	consent, err := consentService.CreateConsent(context.Background(), createRequest, "TEST_CLIENT", "TEST_ORG")
	require.NoError(t, err)
	require.NotNil(t, consent)
	defer cleanupConsent(t, consentDAO, consent.ConsentID)

	// Prepare validate request
	validateRequest := models.ValidateRequest{
		ConsentID: consent.ConsentID,
		UserID:    "test-user-123",
		ResourceParams: struct {
			Resource   string `json:"resource"`
			HTTPMethod string `json:"httpMethod"`
			Context    string `json:"context"`
		}{
			Resource:   "/accounts/123",
			HTTPMethod: "GET",
		},
	}

	requestBody, err := json.Marshal(validateRequest)
	require.NoError(t, err)

	// Make request
	req := httptest.NewRequest(http.MethodPost, "/api/v1/validate", bytes.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("org-id", "TEST_ORG")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code, "Expected 200 OK")

	var response models.ValidateResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response.IsValid, "Expected isValid to be false")
	assert.Equal(t, "Unauthorized", response.ErrorCode)
	assert.Contains(t, response.ErrorMessage, "not in active state")
	assert.Equal(t, "401", response.HTTPCode)
}

// TestValidateConsent_ExpiredConsent tests validation with expired consent
func TestValidateConsent_ExpiredConsent(t *testing.T) {
	router, consentService, consentDAO := setupValidateTestEnvironment(t)
	defer cleanupConsent(t, consentDAO, "")

	// Create a test consent with active status but expired validityTime
	cfg := config.Get()
	activeStatus := cfg.Consent.StatusMappings.ActiveStatus

	// Set validity time to 1 hour ago
	expiredTime := time.Now().Add(-1 * time.Hour).UnixNano() / int64(time.Millisecond)

	createRequest := &models.ConsentCreateRequest{
		ConsentPurpose: []models.ConsentPurposeItem{
			{Name: "test", Value: "data"},
		},
		ConsentType:   "ACCOUNT_ACCESS",
		CurrentStatus: activeStatus,
		ValidityTime:  &expiredTime,
	}

	consent, err := consentService.CreateConsent(context.Background(), createRequest, "TEST_CLIENT", "TEST_ORG")
	require.NoError(t, err)
	require.NotNil(t, consent)
	defer cleanupConsent(t, consentDAO, consent.ConsentID)

	t.Logf("Created expired consent: %s with validityTime: %d", consent.ConsentID, expiredTime)

	// Prepare validate request
	validateRequest := models.ValidateRequest{
		ConsentID: consent.ConsentID,
		UserID:    "test-user-123",
		ResourceParams: struct {
			Resource   string `json:"resource"`
			HTTPMethod string `json:"httpMethod"`
			Context    string `json:"context"`
		}{
			Resource:   "/accounts/123",
			HTTPMethod: "GET",
		},
	}

	requestBody, err := json.Marshal(validateRequest)
	require.NoError(t, err)

	// Make request
	req := httptest.NewRequest(http.MethodPost, "/api/v1/validate", bytes.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("org-id", "TEST_ORG")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code, "Expected 200 OK")

	var response models.ValidateResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response.IsValid, "Expected isValid to be false")
	assert.Equal(t, "Expired", response.ErrorCode)
	assert.Contains(t, response.ErrorMessage, "expired")
	assert.Equal(t, "401", response.HTTPCode)

	// Verify the consent status was updated to expired in the database
	updatedConsent, err := consentService.GetConsent(context.Background(), consent.ConsentID, "TEST_ORG")
	require.NoError(t, err)
	assert.Equal(t, cfg.Consent.StatusMappings.ExpiredStatus, updatedConsent.CurrentStatus, "Expected consent status to be updated to expired")

	t.Logf("Consent status updated to: %s", updatedConsent.CurrentStatus)
}

// TestValidateConsent_NotFound tests validation with non-existent consent
func TestValidateConsent_NotFound(t *testing.T) {
	router, _, _ := setupValidateTestEnvironment(t)

	// Prepare validate request with non-existent consent ID
	validateRequest := models.ValidateRequest{
		ConsentID: "NON_EXISTENT_CONSENT",
		UserID:    "test-user-123",
		ResourceParams: struct {
			Resource   string `json:"resource"`
			HTTPMethod string `json:"httpMethod"`
			Context    string `json:"context"`
		}{
			Resource:   "/accounts/123",
			HTTPMethod: "GET",
		},
	}

	requestBody, err := json.Marshal(validateRequest)
	require.NoError(t, err)

	// Make request
	req := httptest.NewRequest(http.MethodPost, "/api/v1/validate", bytes.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("org-id", "TEST_ORG")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code, "Expected 200 OK")

	var response models.ValidateResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response.IsValid, "Expected isValid to be false")
	assert.Equal(t, "NotFound", response.ErrorCode)
	assert.Contains(t, response.ErrorMessage, "not found")
	assert.Equal(t, "404", response.HTTPCode)
}

// TestValidateConsent_MissingConsentID tests validation without consent ID
func TestValidateConsent_MissingConsentID(t *testing.T) {
	router, _, _ := setupValidateTestEnvironment(t)

	// Prepare validate request without consent ID
	validateRequest := models.ValidateRequest{
		UserID: "test-user-123",
		ResourceParams: struct {
			Resource   string `json:"resource"`
			HTTPMethod string `json:"httpMethod"`
			Context    string `json:"context"`
		}{
			Resource:   "/accounts/123",
			HTTPMethod: "GET",
		},
	}

	requestBody, err := json.Marshal(validateRequest)
	require.NoError(t, err)

	// Make request
	req := httptest.NewRequest(http.MethodPost, "/api/v1/validate", bytes.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("org-id", "TEST_ORG")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code, "Expected 200 OK")

	var response models.ValidateResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response.IsValid, "Expected isValid to be false")
	assert.Equal(t, "InvalidRequest", response.ErrorCode)
	assert.Contains(t, response.ErrorMessage, "consentId is required")
	assert.Equal(t, "400", response.HTTPCode)
}

// TestValidateConsent_MissingUserID tests validation without user ID
func TestValidateConsent_MissingUserID(t *testing.T) {
	router, _, _ := setupValidateTestEnvironment(t)

	// Prepare validate request without user ID
	validateRequest := models.ValidateRequest{
		ConsentID: "SOME_CONSENT_ID",
		ResourceParams: struct {
			Resource   string `json:"resource"`
			HTTPMethod string `json:"httpMethod"`
			Context    string `json:"context"`
		}{
			Resource:   "/accounts/123",
			HTTPMethod: "GET",
		},
	}

	requestBody, err := json.Marshal(validateRequest)
	require.NoError(t, err)

	// Make request
	req := httptest.NewRequest(http.MethodPost, "/api/v1/validate", bytes.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("org-id", "TEST_ORG")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code, "Expected 200 OK")

	var response models.ValidateResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response.IsValid, "Expected isValid to be false")
	assert.Equal(t, "InvalidRequest", response.ErrorCode)
	assert.Contains(t, response.ErrorMessage, "userId is required")
	assert.Equal(t, "400", response.HTTPCode)
}

// TestValidateConsent_MissingResourceParams tests validation without resource params
func TestValidateConsent_MissingResourceParams(t *testing.T) {
	router, _, _ := setupValidateTestEnvironment(t)

	// Prepare validate request without resource params
	validateRequest := models.ValidateRequest{
		ConsentID: "SOME_CONSENT_ID",
		UserID:    "test-user-123",
	}

	requestBody, err := json.Marshal(validateRequest)
	require.NoError(t, err)

	// Make request
	req := httptest.NewRequest(http.MethodPost, "/api/v1/validate", bytes.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("org-id", "TEST_ORG")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code, "Expected 200 OK")

	var response models.ValidateResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response.IsValid, "Expected isValid to be false")
	assert.Equal(t, "InvalidRequest", response.ErrorCode)
	assert.Contains(t, response.ErrorMessage, "resourceParams")
	assert.Equal(t, "400", response.HTTPCode)
}

// TestValidateConsent_InvalidJSON tests validation with invalid JSON
func TestValidateConsent_InvalidJSON(t *testing.T) {
	router, _, _ := setupValidateTestEnvironment(t)

	// Make request with invalid JSON
	req := httptest.NewRequest(http.MethodPost, "/api/v1/validate", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("org-id", "TEST_ORG")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code, "Expected 200 OK")

	var response models.ValidateResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.False(t, response.IsValid, "Expected isValid to be false")
	assert.Equal(t, "InvalidRequest", response.ErrorCode)
	assert.Contains(t, response.ErrorMessage, "Invalid request body")
	assert.Equal(t, "400", response.HTTPCode)
}
