package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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

// TestAPIEnvironment sets up test environment for API integration tests
type TestAPIEnvironment struct {
	Router              *gin.Engine
	ConsentService      *service.ConsentService
	AuthResourceService *service.AuthResourceService
	ConsentDAO          *dao.ConsentDAO
	StatusAuditDAO      *dao.StatusAuditDAO
	AttributeDAO        *dao.ConsentAttributeDAO
	FileDAO             *dao.ConsentFileDAO
	AuthResourceDAO     *dao.AuthResourceDAO
}

func setupAPITestEnvironment(t *testing.T) *TestAPIEnvironment {
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
	fileDAO := dao.NewConsentFileDAO(db)
	authResourceDAO := dao.NewAuthResourceDAO(db)

	// Create services
	consentService := service.NewConsentService(consentDAO, statusAuditDAO, attributeDAO, authResourceDAO, db, logger)
	authResourceService := service.NewAuthResourceService(authResourceDAO, consentDAO, db, logger)

	// Create router
	gin.SetMode(gin.TestMode)
	testRouter := gin.New()

	// Add middleware FIRST to set context values
	testRouter.Use(func(c *gin.Context) {
		// Set default org and client IDs if not present
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

	// Create handlers
	consentHandler := handlers.NewConsentHandler(consentService)
	authResourceHandler := handlers.NewAuthResourceHandler(authResourceService)

	// API v1 routes
	v1 := testRouter.Group("/api/v1")
	{
		// Consent routes
		consents := v1.Group("/consents")
		{
			consents.POST("", consentHandler.CreateConsent)
			consents.GET("/:consentId", consentHandler.GetConsent)
			consents.PUT("/:consentId", consentHandler.UpdateConsent)

			// Authorization resource routes
			consents.POST("/:consentId/authorizations", authResourceHandler.CreateAuthResource)
		}
	}

	return &TestAPIEnvironment{
		Router:              testRouter,
		ConsentService:      consentService,
		AuthResourceService: authResourceService,
		ConsentDAO:          consentDAO,
		StatusAuditDAO:      statusAuditDAO,
		AttributeDAO:        attributeDAO,
		FileDAO:             fileDAO,
		AuthResourceDAO:     authResourceDAO,
	}
}

func cleanupAPITestData(t *testing.T, env *TestAPIEnvironment, consentIDs ...string) {
	ctx := context.Background()
	for _, consentID := range consentIDs {
		err := env.ConsentDAO.Delete(ctx, consentID, "TEST_ORG")
		if err != nil {
			t.Logf("Warning: Failed to cleanup consent %s: %v", consentID, err)
		}
	}
}

func TestAPI_CreateConsent(t *testing.T) {
	env := setupAPITestEnvironment(t)
	defer func() {
		// No specific consent IDs to clean up here, empty variadic call
	}()

	// Prepare request using new API format
	validityTime := int64(7776000) // ~90 days in seconds
	frequency := 1
	recurringIndicator := false

	createReq := &models.ConsentAPIRequest{
		Type:               "accounts",
		Status:             "awaitingAuthorization",
		ValidityTime:       &validityTime,
		RecurringIndicator: &recurringIndicator,
		Frequency:          &frequency,
		RequestPayload: map[string]interface{}{
			"data":    "API test consent",
			"purpose": "testing",
		},
		Attributes: map[string]string{
			"source": "api-test",
		},
	}

	reqBody, err := json.Marshal(createReq)
	require.NoError(t, err)

	// Make request
	req, err := http.NewRequest("POST", "/api/v1/consents", bytes.NewBuffer(reqBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("org-id", "TEST_ORG")
	req.Header.Set("client-id", "TEST_CLIENT")

	// Execute request
	recorder := httptest.NewRecorder()
	env.Router.ServeHTTP(recorder, req)

	// Assert response
	assert.Equal(t, http.StatusCreated, recorder.Code, "Expected 201 Created status")

	var response models.ConsentAPIResponse
	err = json.Unmarshal(recorder.Body.Bytes(), &response)
	require.NoError(t, err)

	// Verify response data
	assert.NotEmpty(t, response.ID, "Consent ID should not be empty")
	assert.Equal(t, "accounts", response.Type)
	assert.Equal(t, "TEST_CLIENT", response.ClientID)
	assert.Equal(t, "awaitingAuthorization", response.Status)
	assert.NotNil(t, response.RequestPayload)
	assert.NotNil(t, response.Attributes)
	assert.NotNil(t, response.Authorizations) // Should be empty array

	// Verify consent was created in database
	ctx := context.Background()
	dbConsent, err := env.ConsentDAO.GetByID(ctx, response.ID, "TEST_ORG")
	require.NoError(t, err)
	assert.Equal(t, response.ID, dbConsent.ConsentID)
	assert.Equal(t, "accounts", dbConsent.ConsentType)

	// Cleanup
	cleanupAPITestData(t, env, response.ID)
}

func TestAPI_CreateConsentWithAuthResources(t *testing.T) {
	env := setupAPITestEnvironment(t)
	defer func() {}()

	// Prepare request with auth resources using new API format
	validityTime := int64(2592000) // ~30 days in seconds
	recurringIndicator := true
	frequency := 5

	createReq := &models.ConsentAPIRequest{
		Type:               "payments",
		Status:             "awaitingAuthorization",
		ValidityTime:       &validityTime,
		RecurringIndicator: &recurringIndicator,
		Frequency:          &frequency,
		RequestPayload: map[string]interface{}{
			"data": "consent with auth",
		},
		Attributes: map[string]string{
			"test": "value",
		},
		Authorizations: []models.AuthorizationAPIRequest{
			{
				UserID: "user-789",
				Type:   "authorization_code",
				Status: "authorized",
				Resource: map[string]interface{}{
					"scopes": []string{"read", "write"},
				},
			},
		},
	}

	reqBody, err := json.Marshal(createReq)
	require.NoError(t, err)

	// Make request
	req, err := http.NewRequest("POST", "/api/v1/consents", bytes.NewBuffer(reqBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("org-id", "TEST_ORG")
	req.Header.Set("client-id", "TEST_CLIENT")

	// Execute request
	recorder := httptest.NewRecorder()
	env.Router.ServeHTTP(recorder, req)

	// Assert response
	assert.Equal(t, http.StatusCreated, recorder.Code)

	var response models.ConsentAPIResponse
	err = json.Unmarshal(recorder.Body.Bytes(), &response)
	require.NoError(t, err)

	// Verify auth resources in response
	assert.NotEmpty(t, response.ID, "Consent ID should not be empty")
	assert.Len(t, response.Authorizations, 1, "Should have 1 authorization")
	assert.Equal(t, "authorization_code", response.Authorizations[0].Type)
	assert.Equal(t, "authorized", response.Authorizations[0].Status)
	assert.NotNil(t, response.Authorizations[0].UserID)
	assert.Equal(t, "user-789", *response.Authorizations[0].UserID)

	// Verify auth resources were created in database
	ctx := context.Background()
	authResources, err := env.AuthResourceDAO.GetByConsentID(ctx, response.ID, "TEST_ORG")
	require.NoError(t, err)
	assert.Len(t, authResources, 1)
	assert.Equal(t, "authorization_code", authResources[0].AuthType)
	assert.Equal(t, "authorized", authResources[0].AuthStatus)

	// Cleanup
	cleanupAPITestData(t, env, response.ID)
}

func TestAPI_CreateConsentInvalidRequest(t *testing.T) {
	env := setupAPITestEnvironment(t)
	defer func() {}()

	tests := []struct {
		name           string
		requestBody    string
		expectedStatus int
	}{
		{
			name:           "Missing consent type",
			requestBody:    `{"requestPayload": {"data": "test"}, "status": "awaitingAuthorization"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Missing status",
			requestBody:    `{"type": "accounts", "requestPayload": {"data": "test"}}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Missing requestPayload",
			requestBody:    `{"type": "accounts", "status": "awaitingAuthorization"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Invalid JSON",
			requestBody:    `{invalid json}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Empty request",
			requestBody:    `{}`,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("POST", "/api/v1/consents", bytes.NewBufferString(tt.requestBody))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("org-id", "TEST_ORG")
			req.Header.Set("client-id", "TEST_CLIENT")

			recorder := httptest.NewRecorder()
			env.Router.ServeHTTP(recorder, req)

			assert.Equal(t, tt.expectedStatus, recorder.Code)
		})
	}
}

func TestAPI_GetConsent(t *testing.T) {
	env := setupAPITestEnvironment(t)
	defer func() {}()

	// Step 1: Create a consent first
	validityTime := int64(7776000)
	frequency := 1
	recurringIndicator := false

	createReq := &models.ConsentAPIRequest{
		Type:               "accounts",
		Status:             "awaitingAuthorization",
		ValidityTime:       &validityTime,
		RecurringIndicator: &recurringIndicator,
		Frequency:          &frequency,
		RequestPayload: map[string]interface{}{
			"data":    "Test consent for GET",
			"purpose": "testing",
		},
		Attributes: map[string]string{
			"test": "get-endpoint",
		},
	}

	reqBody, err := json.Marshal(createReq)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", "/api/v1/consents", bytes.NewBuffer(reqBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("org-id", "TEST_ORG")
	req.Header.Set("client-id", "TEST_CLIENT")

	recorder := httptest.NewRecorder()
	env.Router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusCreated, recorder.Code)

	var createResponse models.ConsentAPIResponse
	err = json.Unmarshal(recorder.Body.Bytes(), &createResponse)
	require.NoError(t, err)
	require.NotEmpty(t, createResponse.ID)

	// Step 2: Now GET the created consent
	getReq, err := http.NewRequest("GET", "/api/v1/consents/"+createResponse.ID, nil)
	require.NoError(t, err)
	getReq.Header.Set("org-id", "TEST_ORG")

	getRecorder := httptest.NewRecorder()
	env.Router.ServeHTTP(getRecorder, getReq)

	// Assert response
	assert.Equal(t, http.StatusOK, getRecorder.Code, "Expected 200 OK status")

	var getResponse models.ConsentAPIResponse
	err = json.Unmarshal(getRecorder.Body.Bytes(), &getResponse)
	require.NoError(t, err)

	// Verify response data
	assert.Equal(t, createResponse.ID, getResponse.ID)
	assert.Equal(t, "accounts", getResponse.Type)
	assert.Equal(t, "TEST_CLIENT", getResponse.ClientID)
	assert.Equal(t, "awaitingAuthorization", getResponse.Status)
	assert.NotNil(t, getResponse.RequestPayload)
	assert.NotNil(t, getResponse.Attributes)
	assert.Equal(t, "get-endpoint", getResponse.Attributes["test"])

	// Cleanup
	cleanupAPITestData(t, env, createResponse.ID)
}

func TestAPI_GetConsentNotFound(t *testing.T) {
	env := setupAPITestEnvironment(t)
	defer func() {}()

	// Try to GET a non-existent consent
	req, err := http.NewRequest("GET", "/api/v1/consents/CONSENT-nonexistent", nil)
	require.NoError(t, err)
	req.Header.Set("org-id", "TEST_ORG")

	recorder := httptest.NewRecorder()
	env.Router.ServeHTTP(recorder, req)

	// Assert 404 response
	assert.Equal(t, http.StatusNotFound, recorder.Code, "Expected 404 Not Found status")

	var errorResponse map[string]interface{}
	err = json.Unmarshal(recorder.Body.Bytes(), &errorResponse)
	require.NoError(t, err)
	assert.Contains(t, errorResponse, "message")
}

func TestAPI_GetConsentInvalidID(t *testing.T) {
	env := setupAPITestEnvironment(t)
	defer func() {}()

	// Try to GET with invalid consent ID (too long - over 255 characters)
	longID := "CONSENT-" + strings.Repeat("x", 300)
	req, err := http.NewRequest("GET", "/api/v1/consents/"+longID, nil)
	require.NoError(t, err)
	req.Header.Set("org-id", "TEST_ORG")

	recorder := httptest.NewRecorder()
	env.Router.ServeHTTP(recorder, req)

	// Assert 400 Bad Request for validation error
	assert.Equal(t, http.StatusBadRequest, recorder.Code, "Expected 400 Bad Request for invalid ID")
}

func TestAPI_UpdateConsent(t *testing.T) {
	env := setupAPITestEnvironment(t)
	defer func() {}()

	// Step 1: Create a consent first
	validityTime := int64(7776000)
	frequency := 1
	recurringIndicator := false

	createReq := &models.ConsentAPIRequest{
		Type:               "accounts",
		Status:             "awaitingAuthorization",
		ValidityTime:       &validityTime,
		RecurringIndicator: &recurringIndicator,
		Frequency:          &frequency,
		RequestPayload: map[string]interface{}{
			"data":    "Test consent for UPDATE",
			"purpose": "testing",
		},
		Attributes: map[string]string{
			"test": "update-endpoint",
		},
	}

	reqBody, err := json.Marshal(createReq)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", "/api/v1/consents", bytes.NewBuffer(reqBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("org-id", "TEST_ORG")
	req.Header.Set("client-id", "TEST_CLIENT")

	recorder := httptest.NewRecorder()
	env.Router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusCreated, recorder.Code)

	var createResponse models.ConsentAPIResponse
	err = json.Unmarshal(recorder.Body.Bytes(), &createResponse)
	require.NoError(t, err)
	require.NotEmpty(t, createResponse.ID)

	// Step 2: Now UPDATE the created consent
	newValidityTime := int64(15552000) // Double the original
	newFrequency := 10
	newRecurringIndicator := true

	updateReq := &models.ConsentAPIUpdateRequest{
		Status:             "AUTHORIZED",
		ValidityTime:       &newValidityTime,
		RecurringIndicator: &newRecurringIndicator,
		Frequency:          &newFrequency,
		RequestPayload: map[string]interface{}{
			"data":    "Updated consent data",
			"purpose": "updated testing",
		},
		Attributes: map[string]string{
			"test":    "updated",
			"version": "2",
		},
	}

	updateBody, err := json.Marshal(updateReq)
	require.NoError(t, err)

	putReq, err := http.NewRequest("PUT", "/api/v1/consents/"+createResponse.ID, bytes.NewBuffer(updateBody))
	require.NoError(t, err)
	putReq.Header.Set("Content-Type", "application/json")
	putReq.Header.Set("org-id", "TEST_ORG")

	putRecorder := httptest.NewRecorder()
	env.Router.ServeHTTP(putRecorder, putReq)

	// Assert response
	assert.Equal(t, http.StatusOK, putRecorder.Code, "Expected 200 OK status")

	var updateResponse models.ConsentAPIResponse
	err = json.Unmarshal(putRecorder.Body.Bytes(), &updateResponse)
	require.NoError(t, err)

	// Verify response data
	assert.Equal(t, createResponse.ID, updateResponse.ID)
	assert.Equal(t, "AUTHORIZED", updateResponse.Status)
	assert.NotNil(t, updateResponse.ValidityTime)
	assert.Equal(t, newValidityTime, *updateResponse.ValidityTime)
	assert.NotNil(t, updateResponse.Frequency)
	assert.Equal(t, newFrequency, *updateResponse.Frequency)
	assert.NotNil(t, updateResponse.RecurringIndicator)
	assert.Equal(t, newRecurringIndicator, *updateResponse.RecurringIndicator)
	assert.NotNil(t, updateResponse.Attributes)
	assert.Equal(t, "updated", updateResponse.Attributes["test"])
	assert.Equal(t, "2", updateResponse.Attributes["version"])

	// Cleanup
	cleanupAPITestData(t, env, createResponse.ID)
}

func TestAPI_UpdateConsentType(t *testing.T) {
	env := setupAPITestEnvironment(t)
	defer func() {}()

	// Step 1: Create a consent first
	validityTime := int64(7776000)
	frequency := 1
	recurringIndicator := false

	createReq := &models.ConsentAPIRequest{
		Type:               "accounts",
		Status:             "awaitingAuthorization",
		ValidityTime:       &validityTime,
		RecurringIndicator: &recurringIndicator,
		Frequency:          &frequency,
		RequestPayload: map[string]interface{}{
			"data":    "Test consent for type update",
			"purpose": "testing",
		},
	}

	reqBody, err := json.Marshal(createReq)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", "/api/v1/consents", bytes.NewBuffer(reqBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("org-id", "TEST_ORG")
	req.Header.Set("client-id", "TEST_CLIENT")

	recorder := httptest.NewRecorder()
	env.Router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusCreated, recorder.Code)

	var createResponse models.ConsentAPIResponse
	err = json.Unmarshal(recorder.Body.Bytes(), &createResponse)
	require.NoError(t, err)
	require.NotEmpty(t, createResponse.ID)
	assert.Equal(t, "accounts", createResponse.Type)

	// Step 2: Update the consent type
	updateReq := &models.ConsentAPIUpdateRequest{
		Type: "payments",
	}

	updateBody, err := json.Marshal(updateReq)
	require.NoError(t, err)

	putReq, err := http.NewRequest("PUT", "/api/v1/consents/"+createResponse.ID, bytes.NewBuffer(updateBody))
	require.NoError(t, err)
	putReq.Header.Set("Content-Type", "application/json")
	putReq.Header.Set("org-id", "TEST_ORG")

	putRecorder := httptest.NewRecorder()
	env.Router.ServeHTTP(putRecorder, putReq)

	// Assert response
	assert.Equal(t, http.StatusOK, putRecorder.Code, "Expected 200 OK status")

	var updateResponse models.ConsentAPIResponse
	err = json.Unmarshal(putRecorder.Body.Bytes(), &updateResponse)
	require.NoError(t, err)

	// Verify the type was updated
	assert.Equal(t, createResponse.ID, updateResponse.ID)
	assert.Equal(t, "payments", updateResponse.Type, "Consent type should be updated to payments")
	assert.Equal(t, "awaitingAuthorization", updateResponse.Status, "Status should remain unchanged")

	// Cleanup
	cleanupAPITestData(t, env, createResponse.ID)
}

func TestAPI_UpdateConsentNotFound(t *testing.T) {
	env := setupAPITestEnvironment(t)
	defer func() {}()

	// Try to UPDATE a non-existent consent
	validityTime := int64(7776000)
	frequency := 1
	recurringIndicator := false

	updateReq := &models.ConsentAPIUpdateRequest{
		Status:             "AUTHORIZED",
		ValidityTime:       &validityTime,
		RecurringIndicator: &recurringIndicator,
		Frequency:          &frequency,
		RequestPayload: map[string]interface{}{
			"data": "test",
		},
	}

	reqBody, err := json.Marshal(updateReq)
	require.NoError(t, err)

	req, err := http.NewRequest("PUT", "/api/v1/consents/CONSENT-nonexistent", bytes.NewBuffer(reqBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("org-id", "TEST_ORG")

	recorder := httptest.NewRecorder()
	env.Router.ServeHTTP(recorder, req)

	// Assert 404 Not Found
	assert.Equal(t, http.StatusNotFound, recorder.Code, "Expected 404 Not Found status")

	var errorResponse map[string]interface{}
	err = json.Unmarshal(recorder.Body.Bytes(), &errorResponse)
	require.NoError(t, err)
	assert.Contains(t, errorResponse, "message")
}

func TestAPI_UpdateConsentInvalidStatus(t *testing.T) {
	env := setupAPITestEnvironment(t)
	defer func() {}()

	// Step 1: Create a consent first
	validityTime := int64(7776000)
	frequency := 1
	recurringIndicator := false

	createReq := &models.ConsentAPIRequest{
		Type:               "accounts",
		Status:             "awaitingAuthorization",
		ValidityTime:       &validityTime,
		RecurringIndicator: &recurringIndicator,
		Frequency:          &frequency,
		RequestPayload: map[string]interface{}{
			"data": "test",
		},
	}

	reqBody, err := json.Marshal(createReq)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", "/api/v1/consents", bytes.NewBuffer(reqBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("org-id", "TEST_ORG")
	req.Header.Set("client-id", "TEST_CLIENT")

	recorder := httptest.NewRecorder()
	env.Router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusCreated, recorder.Code)

	var createResponse models.ConsentAPIResponse
	err = json.Unmarshal(recorder.Body.Bytes(), &createResponse)
	require.NoError(t, err)

	// Step 2: Try to UPDATE with invalid status
	updateReq := &models.ConsentAPIUpdateRequest{
		Status: "INVALID_STATUS",
		RequestPayload: map[string]interface{}{
			"data": "test",
		},
	}

	updateBody, err := json.Marshal(updateReq)
	require.NoError(t, err)

	putReq, err := http.NewRequest("PUT", "/api/v1/consents/"+createResponse.ID, bytes.NewBuffer(updateBody))
	require.NoError(t, err)
	putReq.Header.Set("Content-Type", "application/json")
	putReq.Header.Set("org-id", "TEST_ORG")

	putRecorder := httptest.NewRecorder()
	env.Router.ServeHTTP(putRecorder, putReq)

	// Assert 400 Bad Request for invalid status
	assert.Equal(t, http.StatusBadRequest, putRecorder.Code, "Expected 400 Bad Request for invalid status")

	// Cleanup
	cleanupAPITestData(t, env, createResponse.ID)
}

func TestAPI_CreateAuthResource(t *testing.T) {
	env := setupAPITestEnvironment(t)
	defer func() {}()

	// Step 1: Create a consent first
	validityTime := int64(7776000)
	frequency := 1
	recurringIndicator := false

	createReq := &models.ConsentAPIRequest{
		Type:               "accounts",
		Status:             "awaitingAuthorization",
		ValidityTime:       &validityTime,
		RecurringIndicator: &recurringIndicator,
		Frequency:          &frequency,
		RequestPayload: map[string]interface{}{
			"data": "Test consent for authorization",
		},
	}

	reqBody, err := json.Marshal(createReq)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", "/api/v1/consents", bytes.NewBuffer(reqBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("org-id", "TEST_ORG")
	req.Header.Set("client-id", "TEST_CLIENT")

	recorder := httptest.NewRecorder()
	env.Router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusCreated, recorder.Code)

	var consentResponse models.ConsentAPIResponse
	err = json.Unmarshal(recorder.Body.Bytes(), &consentResponse)
	require.NoError(t, err)
	require.NotEmpty(t, consentResponse.ID)

	// Step 2: Create an authorization resource for the consent
	authReq := &models.AuthorizationAPIRequest{
		UserID: "user123",
		Type:   "account",
		Status: "authorized",
		Resource: map[string]interface{}{
			"accountId": "ACC-123",
		},
	}

	authReqBody, err := json.Marshal(authReq)
	require.NoError(t, err)

	authReqHTTP, err := http.NewRequest("POST", "/api/v1/consents/"+consentResponse.ID+"/authorizations", bytes.NewBuffer(authReqBody))
	require.NoError(t, err)
	authReqHTTP.Header.Set("Content-Type", "application/json")
	authReqHTTP.Header.Set("org-id", "TEST_ORG")

	authRecorder := httptest.NewRecorder()
	env.Router.ServeHTTP(authRecorder, authReqHTTP)

	// Assert response
	assert.Equal(t, http.StatusCreated, authRecorder.Code, "Expected 201 Created status")

	var authResponse models.AuthorizationAPIResponse
	err = json.Unmarshal(authRecorder.Body.Bytes(), &authResponse)
	require.NoError(t, err)

	// Verify response data
	assert.NotEmpty(t, authResponse.ID, "Authorization ID should not be empty")
	assert.NotNil(t, authResponse.UserID)
	assert.Equal(t, "user123", *authResponse.UserID)
	assert.Equal(t, "account", authResponse.Type)
	assert.Equal(t, "authorized", authResponse.Status)
	assert.NotNil(t, authResponse.Resource)
	assert.Equal(t, "ACC-123", authResponse.Resource["accountId"])

	// Cleanup
	cleanupAPITestData(t, env, consentResponse.ID)
}

func TestAPI_CreateAuthResourceConsentNotFound(t *testing.T) {
	env := setupAPITestEnvironment(t)
	defer func() {}()

	// Try to create authorization for non-existent consent
	authReq := &models.AuthorizationAPIRequest{
		UserID: "user123",
		Type:   "account",
		Status: "authorized",
		Resource: map[string]interface{}{
			"accountId": "ACC-123",
		},
	}

	authReqBody, err := json.Marshal(authReq)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", "/api/v1/consents/CONSENT-nonexistent/authorizations", bytes.NewBuffer(authReqBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("org-id", "TEST_ORG")

	recorder := httptest.NewRecorder()
	env.Router.ServeHTTP(recorder, req)

	// Assert 404 Not Found
	assert.Equal(t, http.StatusNotFound, recorder.Code, "Expected 404 Not Found status for non-existent consent")
}

func TestAPI_CreateAuthResourceInvalidRequest(t *testing.T) {
	env := setupAPITestEnvironment(t)
	defer func() {}()

	// Step 1: Create a consent first
	validityTime := int64(7776000)
	frequency := 1
	recurringIndicator := false

	createReq := &models.ConsentAPIRequest{
		Type:               "accounts",
		Status:             "awaitingAuthorization",
		ValidityTime:       &validityTime,
		RecurringIndicator: &recurringIndicator,
		Frequency:          &frequency,
		RequestPayload: map[string]interface{}{
			"data": "Test consent",
		},
	}

	reqBody, err := json.Marshal(createReq)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", "/api/v1/consents", bytes.NewBuffer(reqBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("org-id", "TEST_ORG")
	req.Header.Set("client-id", "TEST_CLIENT")

	recorder := httptest.NewRecorder()
	env.Router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusCreated, recorder.Code)

	var consentResponse models.ConsentAPIResponse
	err = json.Unmarshal(recorder.Body.Bytes(), &consentResponse)
	require.NoError(t, err)

	// Step 2: Try to create authorization with missing required fields
	tests := []struct {
		name           string
		requestBody    string
		expectedStatus int
	}{
		{
			name:           "Missing type",
			requestBody:    `{"userId": "user123", "status": "authorized"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Missing status",
			requestBody:    `{"userId": "user123", "type": "account"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Invalid JSON",
			requestBody:    `{invalid json}`,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authReq, err := http.NewRequest("POST", "/api/v1/consents/"+consentResponse.ID+"/authorizations", bytes.NewBufferString(tt.requestBody))
			require.NoError(t, err)
			authReq.Header.Set("Content-Type", "application/json")
			authReq.Header.Set("org-id", "TEST_ORG")

			authRecorder := httptest.NewRecorder()
			env.Router.ServeHTTP(authRecorder, authReq)

			assert.Equal(t, tt.expectedStatus, authRecorder.Code)
		})
	}

	// Cleanup
	cleanupAPITestData(t, env, consentResponse.ID)
}
