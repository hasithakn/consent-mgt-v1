package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wso2/consent-management-api/internal/client"
	"github.com/wso2/consent-management-api/internal/config"
	"github.com/wso2/consent-management-api/internal/dao"
	"github.com/wso2/consent-management-api/internal/database"
	"github.com/wso2/consent-management-api/internal/handlers"
	"github.com/wso2/consent-management-api/internal/models"
	"github.com/wso2/consent-management-api/internal/service"
	"github.com/wso2/consent-management-api/pkg/utils"
)

// MockExtensionServer represents a mock extension service for testing
type MockExtensionServer struct {
	server   *httptest.Server
	handler  func(w http.ResponseWriter, r *http.Request)
	requests []*http.Request
	mu       sync.RWMutex
}

// NewMockExtensionServer creates a new mock extension server
func NewMockExtensionServer() *MockExtensionServer {
	mock := &MockExtensionServer{
		requests: make([]*http.Request, 0),
	}

	mock.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mock.mu.Lock()
		mock.requests = append(mock.requests, r)
		handler := mock.handler
		mock.mu.Unlock()

		if handler != nil {
			handler(w, r)
		}
	}))

	return mock
}

// SetHandler sets the handler function for the mock server
func (m *MockExtensionServer) SetHandler(handler func(w http.ResponseWriter, r *http.Request)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handler = handler
}

// GetRequests returns all requests received by the mock server
func (m *MockExtensionServer) GetRequests() []*http.Request {
	return m.requests
}

// Close closes the mock server
func (m *MockExtensionServer) Close() {
	m.server.Close()
}

// URL returns the URL of the mock server
func (m *MockExtensionServer) URL() string {
	return m.server.URL
}

// setupExtensionTestEnvironment sets up test environment with mock extension service
func setupExtensionTestEnvironment(t *testing.T, mockServer *MockExtensionServer) (*gin.Engine, *database.DB, *service.ConsentPurposeService) {
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
	authResourceService := service.NewAuthResourceService(authResourceDAO, consentDAO, db, logger)
	consentPurposeService := service.NewConsentPurposeService(consentPurposeDAO, consentDAO, db.DB, logger)

	// Create extension client pointing to mock server
	extensionConfig := &config.ExtensionConfig{
		Enabled: true,
		BaseURL: mockServer.URL(),
		Endpoints: config.ExtensionEndpoints{
			PreProcessConsentCreation: "/pre-process-consent-creation",
			PreProcessConsentUpdate:   "/pre-process-consent-update",
		},
		Timeout: 30 * time.Second, // 30 seconds timeout for tests
	}
	extensionClient := client.NewExtensionClient(extensionConfig, logger)

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

	// Create handlers with extension client
	consentHandler := handlers.NewConsentHandler(consentService, consentPurposeService, extensionClient)
	authResourceHandler := handlers.NewAuthResourceHandler(authResourceService)

	// API v1 routes
	v1 := testRouter.Group("/api/v1")
	{
		consents := v1.Group("/consents")
		{
			consents.POST("", consentHandler.CreateConsent)
			consents.GET("/:consentId", consentHandler.GetConsent)
			consents.PUT("/:consentId", consentHandler.UpdateConsent)

			consents.POST("/:consentId/authorizations", authResourceHandler.CreateAuthResource)
		}
	}

	return testRouter, db, consentPurposeService
}

// TestExtension_PreCreateConsent_WithValidPurposes tests extension with valid consent purposes
func TestExtension_PreCreateConsent_WithValidPurposes(t *testing.T) {
	// Create mock extension server
	mockServer := NewMockExtensionServer()
	defer mockServer.Close()

	// Configure mock extension to return SUCCESS with resolved purposes FIRST
	mockServer.SetHandler(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		assert.Equal(t, "/pre-process-consent-creation", r.URL.Path)
		assert.Equal(t, "POST", r.Method)

		// Parse request
		var extRequest models.PreProcessConsentCreationRequest
		err := json.NewDecoder(r.Body).Decode(&extRequest)
		require.NoError(t, err)

		// Return SUCCESS response with resolved purposes
		response := models.PreProcessConsentCreationResponse{
			ResponseID: extRequest.RequestID,
			Status:     "SUCCESS",
			Data: &models.PreProcessConsentCreationResponseData{
				ConsentResource: models.DetailedConsentResourceData{
					Type:           extRequest.Data.ConsentInitiationData.Type,
					Status:         extRequest.Data.ConsentInitiationData.Status,
					RequestPayload: extRequest.Data.ConsentInitiationData.RequestPayload,
				},
				ResolvedConsentPurposes: []string{"ReadAccountsBasic", "ReadAccountsDetail"},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	})

	// Setup test environment with extension client
	router, db, purposeService := setupExtensionTestEnvironment(t, mockServer)

	// Clean up test data
	defer func() {
		_, _ = db.Exec("DELETE FROM CONSENT WHERE ORG_ID = 'TEST_ORG'")
		_, _ = db.Exec("DELETE FROM CONSENT_PURPOSE WHERE ORG_ID = 'TEST_ORG'")
	}()

	// Create test consent purposes first
	ctx := context.Background()
	purpose1, err := purposeService.CreatePurpose(ctx, "TEST_ORG", &service.ConsentPurposeCreateRequest{
		Name:        "ReadAccountsBasic",
		Description: stringPtr("Marketing communications"),
	})
	require.NoError(t, err)

	purpose2, err := purposeService.CreatePurpose(ctx, "TEST_ORG", &service.ConsentPurposeCreateRequest{
		Name:        "ReadAccountsDetail",
		Description: stringPtr("Data analytics"),
	})
	require.NoError(t, err)

	t.Logf("Created test purposes: %s, %s", purpose1.Name, purpose2.Name)

	// Create consent via API with full payload
	validityTime := int64(86400)
	frequency := int32(10)
	dataAccessValidityDuration := int64(172800)
	recurringIndicator := true

	requestBody := map[string]interface{}{
		"type":                       "accounts",
		"status":                     "awaitingAuthorization",
		"validityTime":               validityTime,
		"recurringIndicator":         recurringIndicator,
		"frequency":                  frequency,
		"dataAccessValidityDuration": dataAccessValidityDuration,
		"requestPayload": map[string]interface{}{
			"Data": map[string]interface{}{
				"Permissions": []string{
					"ReadAccountsBasic",
					"ReadAccountsDetail",
				},
				"ExpirationDateTime":      "2025-12-31T23:59:59.000Z",
				"TransactionFromDateTime": "2025-01-01T00:00:00.000Z",
				"TransactionToDateTime":   "2025-12-31T23:59:59.000Z",
			},
			"Risk": map[string]interface{}{
				"PaymentContextCode": "PartyToParty",
			},
		},
		"attributes": map[string]string{
			"consentType":        "recurring",
			"maxFrequencyPerDay": "10",
			"customerType":       "retail",
			"channel":            "online_banking",
			"ipAddress":          "192.168.1.100",
		},
		"authorizations": []map[string]interface{}{
			{
				"userId": "user001@example.com",
				"type":   "authorisation",
				"status": "created",
				"resource": map[string]interface{}{
					"authMethod":   "SMS",
					"authLevel":    "SCA",
					"mobileNumber": "+44-7700-900000",
				},
			},
		},
	}
	body, _ := json.Marshal(requestBody)

	req := httptest.NewRequest("POST", "/api/v1/consents", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("org-id", "TEST_ORG")
	req.Header.Set("client-id", "TEST_CLIENT")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Verify response
	assert.Equal(t, http.StatusCreated, w.Code, "Response body: %s", w.Body.String())

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	t.Logf("✓ Extension successfully validated purposes and consent created: %v", response["id"])

	// Verify extension was called
	requests := mockServer.GetRequests()
	assert.Equal(t, 1, len(requests), "Extension should be called once")
}

// TestExtension_PreCreateConsent_WithInvalidPurposes tests extension with invalid consent purposes
func TestExtension_PreCreateConsent_WithInvalidPurposes(t *testing.T) {
	mockServer := NewMockExtensionServer()
	defer mockServer.Close()

	// Configure mock to return purposes that don't exist
	mockServer.SetHandler(func(w http.ResponseWriter, r *http.Request) {
		var extRequest models.PreProcessConsentCreationRequest
		json.NewDecoder(r.Body).Decode(&extRequest)

		response := models.PreProcessConsentCreationResponse{
			ResponseID: extRequest.RequestID,
			Status:     "SUCCESS",
			Data: &models.PreProcessConsentCreationResponseData{
				ConsentResource: models.DetailedConsentResourceData{
					Type:   extRequest.Data.ConsentInitiationData.Type,
					Status: extRequest.Data.ConsentInitiationData.Status,
				},
				ResolvedConsentPurposes: []string{"NonExistentPurpose"},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	})

	router, db, _ := setupExtensionTestEnvironment(t, mockServer)

	defer func() {
		_, _ = db.Exec("DELETE FROM CONSENT WHERE ORG_ID = 'TEST_ORG'")
		_, _ = db.Exec("DELETE FROM CONSENT_PURPOSE WHERE ORG_ID = 'TEST_ORG'")
	}()

	// Create consent via API
	requestBody := map[string]interface{}{
		"type":           "accounts",
		"status":         "awaitingAuthorization",
		"requestPayload": map[string]interface{}{"data": "test"},
	}
	body, _ := json.Marshal(requestBody)

	req := httptest.NewRequest("POST", "/api/v1/consents", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("org-id", "TEST_ORG")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Verify error response
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var errorResponse map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
	require.NoError(t, err)

	assert.Contains(t, errorResponse["details"], "NonExistentPurpose")
	t.Logf("✓ Correctly rejected invalid purpose: %v", errorResponse["details"])
}

// TestExtension_PreCreateConsent_ExtensionError tests extension returning ERROR status
func TestExtension_PreCreateConsent_ExtensionError(t *testing.T) {
	mockServer := NewMockExtensionServer()
	defer mockServer.Close()

	// Configure mock to return ERROR
	mockServer.SetHandler(func(w http.ResponseWriter, r *http.Request) {
		var extRequest models.PreProcessConsentCreationRequest
		json.NewDecoder(r.Body).Decode(&extRequest)

		errorCode := 403
		response := models.PreProcessConsentCreationResponse{
			ResponseID: extRequest.RequestID,
			Status:     "ERROR",
			ErrorCode:  &errorCode,
			ErrorData: map[string]interface{}{
				"errorMessage": "User not authorized for this consent type",
				"errorCode":    "INSUFFICIENT_PERMISSIONS",
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	})

	router, db, _ := setupExtensionTestEnvironment(t, mockServer)

	defer func() {
		_, _ = db.Exec("DELETE FROM CONSENT WHERE ORG_ID = 'TEST_ORG'")
	}()

	// Create consent via API
	requestBody := map[string]interface{}{
		"type":           "accounts",
		"status":         "awaitingAuthorization",
		"requestPayload": map[string]interface{}{"data": "test"},
	}
	body, _ := json.Marshal(requestBody)

	req := httptest.NewRequest("POST", "/api/v1/consents", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("org-id", "TEST_ORG")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Verify error response
	assert.Equal(t, http.StatusForbidden, w.Code)

	var errorResponse map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
	require.NoError(t, err)

	assert.Equal(t, "User not authorized for this consent type", errorResponse["error"])
	t.Logf("✓ Extension error correctly propagated: %v", errorResponse)
}

// TestExtension_PreCreateConsent_ModifiedConsentData tests extension modifying consent data
func TestExtension_PreCreateConsent_ModifiedConsentData(t *testing.T) {
	mockServer := NewMockExtensionServer()
	defer mockServer.Close()

	// Configure mock to modify consent data
	mockServer.SetHandler(func(w http.ResponseWriter, r *http.Request) {
		var extRequest models.PreProcessConsentCreationRequest
		json.NewDecoder(r.Body).Decode(&extRequest)

		// Extension enriches the consent with additional attributes
		validityTime := int64(7200)
		recurringIndicator := true

		response := models.PreProcessConsentCreationResponse{
			ResponseID: extRequest.RequestID,
			Status:     "SUCCESS",
			Data: &models.PreProcessConsentCreationResponseData{
				ConsentResource: models.DetailedConsentResourceData{
					Type:               extRequest.Data.ConsentInitiationData.Type,
					Status:             extRequest.Data.ConsentInitiationData.Status, // Keep same status
					ValidityTime:       &validityTime,
					RecurringIndicator: &recurringIndicator,
					RequestPayload:     extRequest.Data.ConsentInitiationData.RequestPayload,
					Attributes: map[string]interface{}{
						"enrichedBy":  "extension",
						"riskScore":   "low",
						"autoApprove": "true",
					},
				},
				ResolvedConsentPurposes: []string{},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	})

	router, db, _ := setupExtensionTestEnvironment(t, mockServer)

	defer func() {
		_, _ = db.Exec("DELETE FROM CONSENT WHERE ORG_ID = 'TEST_ORG'")
	}()

	// Create consent via API
	requestBody := map[string]interface{}{
		"type":           "accounts",
		"status":         "awaitingAuthorization",
		"requestPayload": map[string]interface{}{"data": "test"},
	}
	body, _ := json.Marshal(requestBody)

	req := httptest.NewRequest("POST", "/api/v1/consents", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("org-id", "TEST_ORG")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Verify response
	assert.Equal(t, http.StatusCreated, w.Code, "Response body: %s", w.Body.String())

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// Verify modified data
	assert.Equal(t, "awaitingAuthorization", response["status"], "Status should be preserved")
	assert.Equal(t, float64(7200), response["validityTime"], "ValidityTime should be added by extension")
	assert.Equal(t, true, response["recurringIndicator"], "RecurringIndicator should be added by extension")

	attributes := response["attributes"].(map[string]interface{})
	assert.Equal(t, "extension", attributes["enrichedBy"])
	assert.Equal(t, "low", attributes["riskScore"])

	t.Logf("✓ Extension successfully modified consent data: validityTime=%v, recurringIndicator=%v", response["validityTime"], response["recurringIndicator"])
}

// TestExtension_PreUpdateConsent_WithValidPurposes tests pre-update extension with valid purposes
func TestExtension_PreUpdateConsent_WithValidPurposes(t *testing.T) {
	mockServer := NewMockExtensionServer()
	defer mockServer.Close()

	router, db, purposeService := setupExtensionTestEnvironment(t, mockServer)

	defer func() {
		_, _ = db.Exec("DELETE FROM CONSENT WHERE ORG_ID = 'TEST_ORG'")
		_, _ = db.Exec("DELETE FROM CONSENT_PURPOSE WHERE ORG_ID = 'TEST_ORG'")
	}()

	// Create test purposes
	ctx := context.Background()
	purpose, err := purposeService.CreatePurpose(ctx, "TEST_ORG", &service.ConsentPurposeCreateRequest{
		Name:        "DataSharing",
		Description: stringPtr("Data sharing consent"),
	})
	require.NoError(t, err)
	t.Logf("Created test purpose: %s", purpose.Name)

	// First create a consent without extension
	mockServer.SetHandler(func(w http.ResponseWriter, r *http.Request) {
		var extRequest models.PreProcessConsentCreationRequest
		json.NewDecoder(r.Body).Decode(&extRequest)

		// Get the original request payload to include in response
		requestPayload := extRequest.Data.ConsentInitiationData.RequestPayload

		// Simple SUCCESS response with receipt
		response := models.PreProcessConsentCreationResponse{
			ResponseID: extRequest.RequestID,
			Status:     "SUCCESS",
			Data: &models.PreProcessConsentCreationResponseData{
				ConsentResource: models.DetailedConsentResourceData{
					Type:           "accounts",
					Status:         "awaitingAuthorization",
					RequestPayload: requestPayload,
				},
				ResolvedConsentPurposes: []string{},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	// Create initial consent
	createBody := map[string]interface{}{
		"type":   "accounts",
		"status": "awaitingAuthorization",
		"requestPayload": map[string]interface{}{
			"Data": map[string]interface{}{
				"Permissions": []string{"ReadAccountsBasic"},
			},
		},
	}
	body, _ := json.Marshal(createBody)

	req := httptest.NewRequest("POST", "/api/v1/consents", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("org-id", "TEST_ORG")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Logf("Create consent failed with status %d: %s", w.Code, w.Body.String())
	}
	require.Equal(t, http.StatusCreated, w.Code)

	var createResponse map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &createResponse)
	consentID := createResponse["id"].(string)

	// Now configure mock for update with resolved purposes
	mockServer.SetHandler(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/pre-process-consent-update" {
			var extRequest models.PreProcessConsentUpdateRequest
			json.NewDecoder(r.Body).Decode(&extRequest)

			response := models.PreProcessConsentUpdateResponse{
				ResponseID: extRequest.RequestID,
				Status:     "SUCCESS",
				Data: &models.PreProcessConsentUpdateResponseData{
					ConsentResource: models.DetailedConsentResourceData{
						Type:   "accounts",
						Status: "AUTHORIZED",
					},
					ResolvedConsentPurposes: []string{"DataSharing"},
				},
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}
	})

	// Update consent
	updateBody := map[string]interface{}{
		"status": "AUTHORIZED",
	}
	body, _ = json.Marshal(updateBody)

	req = httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/consents/%s", consentID), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("org-id", "TEST_ORG")

	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "Response body: %s", w.Body.String())

	var updateResponse map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &updateResponse)

	assert.Equal(t, "AUTHORIZED", updateResponse["status"])
	t.Logf("✓ Pre-update extension successfully validated purposes and consent updated")
}

// TestExtension_PreUpdateConsent_WithInvalidPurposes tests pre-update extension with invalid purposes
func TestExtension_PreUpdateConsent_WithInvalidPurposes(t *testing.T) {
	mockServer := NewMockExtensionServer()
	defer mockServer.Close()

	// Configure mock for create - return SUCCESS with no purposes
	mockServer.SetHandler(func(w http.ResponseWriter, r *http.Request) {
		var extRequest map[string]interface{}
		json.NewDecoder(r.Body).Decode(&extRequest)

		response := map[string]interface{}{
			"responseId": extRequest["requestId"],
			"status":     "SUCCESS",
			"data": map[string]interface{}{
				"consentResource": map[string]interface{}{
					"type":           "accounts",
					"status":         "awaitingAuthorization",
					"requestPayload": map[string]interface{}{"Data": map[string]interface{}{"Permissions": []string{"ReadAccountsBasic"}}},
				},
				"resolvedConsentPurposes": []string{},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	router, db, _ := setupExtensionTestEnvironment(t, mockServer)

	defer func() {
		_, _ = db.Exec("DELETE FROM CONSENT WHERE ORG_ID = 'TEST_ORG'")
		_, _ = db.Exec("DELETE FROM CONSENT_PURPOSE WHERE ORG_ID = 'TEST_ORG'")
	}()

	// Create initial consent
	createBody := map[string]interface{}{
		"type":   "accounts",
		"status": "awaitingAuthorization",
		"requestPayload": map[string]interface{}{
			"Data": map[string]interface{}{
				"Permissions": []string{"ReadAccountsBasic"},
			},
		},
	}
	body, _ := json.Marshal(createBody)

	req := httptest.NewRequest("POST", "/api/v1/consents", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("org-id", "TEST_ORG")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)

	var createResponse map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &createResponse)
	consentID := createResponse["id"].(string)

	// Configure mock for update to return invalid purpose
	mockServer.SetHandler(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/pre-process-consent-update" {
			var extRequest models.PreProcessConsentUpdateRequest
			json.NewDecoder(r.Body).Decode(&extRequest)

			response := models.PreProcessConsentUpdateResponse{
				ResponseID: extRequest.RequestID,
				Status:     "SUCCESS",
				Data: &models.PreProcessConsentUpdateResponseData{
					ConsentResource: models.DetailedConsentResourceData{
						Type:   "accounts",
						Status: "authorized",
					},
					ResolvedConsentPurposes: []string{"InvalidPurposeForUpdate"},
				},
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}
	})

	// Try to update consent
	updateBody := map[string]interface{}{
		"status": "authorized",
	}
	body, _ = json.Marshal(updateBody)

	req = httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/consents/%s", consentID), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("org-id", "TEST_ORG")

	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Verify error response
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var errorResponse map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &errorResponse)
	require.NoError(t, err)

	assert.Contains(t, errorResponse["details"], "InvalidPurposeForUpdate")
	t.Logf("✓ Pre-update extension correctly rejected invalid purpose: %v", errorResponse["details"])
}
