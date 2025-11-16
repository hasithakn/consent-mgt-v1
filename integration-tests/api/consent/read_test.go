package consent

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wso2/consent-management-api/internal/models"
)

// TestGetConsent_Success tests successful retrieval of a consent
func TestGetConsent_Success(t *testing.T) {
	env := SetupTestEnvironment(t)

	// Create test purposes
	purposes := CreateTestPurposes(t, env, map[string]string{
		"data_access":         "Data access purpose",
		"transaction_history": "Transaction history purpose",
	})
	defer CleanupTestPurposes(t, env, purposes)

	// Create a consent first
	createReq := &models.ConsentAPIRequest{
		Type: "accounts",
		ConsentPurpose: []models.ConsentPurposeItem{
			{Name: "data_access", Value: "Account information", IsSelected: BoolPtr(true)},
			{Name: "transaction_history", Value: "Transaction details", IsSelected: BoolPtr(false)},
		},
		Attributes: map[string]string{
			"channel":   "mobile",
			"country":   "US",
			"accountId": "ACC-12345",
		},
	}

	reqBody, err := json.Marshal(createReq)
	require.NoError(t, err)

	// Make create request
	createHTTPReq, err := http.NewRequest("POST", "/api/v1/consents", bytes.NewBuffer(reqBody))
	require.NoError(t, err)
	createHTTPReq.Header.Set("Content-Type", "application/json")
	createHTTPReq.Header.Set("org-id", "TEST_ORG")
	createHTTPReq.Header.Set("client-id", "TEST_CLIENT")

	createRecorder := httptest.NewRecorder()
	env.Router.ServeHTTP(createRecorder, createHTTPReq)
	require.Equal(t, http.StatusCreated, createRecorder.Code)

	var createResp models.ConsentAPIResponse
	err = json.Unmarshal(createRecorder.Body.Bytes(), &createResp)
	require.NoError(t, err)
	require.NotEmpty(t, createResp.ID)

	// Cleanup
	defer CleanupTestData(t, env, createResp.ID)

	// Now GET the consent
	req, err := http.NewRequest("GET", "/api/v1/consents/"+createResp.ID, nil)
	require.NoError(t, err)
	req.Header.Set("org-id", "TEST_ORG")
	req.Header.Set("client-id", "TEST_CLIENT")

	recorder := httptest.NewRecorder()
	env.Router.ServeHTTP(recorder, req)

	// Assert response status
	assert.Equal(t, http.StatusOK, recorder.Code, "Should return 200 OK")

	// Parse response
	var response models.ConsentAPIResponse
	err = json.Unmarshal(recorder.Body.Bytes(), &response)
	require.NoError(t, err, "Should parse response successfully")

	// Assert all consent metadata fields
	assert.Equal(t, createResp.ID, response.ID, "Consent ID should match")
	assert.Equal(t, "accounts", response.Type, "Type should match")
	assert.Equal(t, "TEST_CLIENT", response.ClientID, "ClientID should match")
	assert.Equal(t, "CREATED", response.Status, "Status should match")
	assert.NotEmpty(t, response.CreatedTime, "CreatedTime should not be empty")
	assert.NotEmpty(t, response.UpdatedTime, "UpdatedTime should not be empty")

	// Assert consent purposes
	require.Len(t, response.ConsentPurpose, 2, "Should have 2 consent purposes")

	// Find purposes by name (order might vary)
	purposeMap := make(map[string]models.ConsentPurposeItem)
	for _, cp := range response.ConsentPurpose {
		purposeMap[cp.Name] = cp
	}

	// Verify data_access purpose
	dataAccess, exists := purposeMap["data_access"]
	assert.True(t, exists, "data_access purpose should exist")
	assert.Equal(t, "Account information", dataAccess.Value, "data_access value should match")
	assert.NotNil(t, dataAccess.IsSelected, "data_access IsSelected should not be nil")
	assert.True(t, *dataAccess.IsSelected, "data_access should be selected")

	// Verify transaction_history purpose
	txHistory, exists := purposeMap["transaction_history"]
	assert.True(t, exists, "transaction_history purpose should exist")
	assert.Equal(t, "Transaction details", txHistory.Value, "transaction_history value should match")
	assert.NotNil(t, txHistory.IsSelected, "transaction_history IsSelected should not be nil")
	assert.False(t, *txHistory.IsSelected, "transaction_history should not be selected")

	// Assert attributes
	require.NotNil(t, response.Attributes, "Attributes should not be nil")
	assert.Equal(t, "mobile", response.Attributes["channel"], "channel attribute should match")
	assert.Equal(t, "US", response.Attributes["country"], "country attribute should match")
	assert.Equal(t, "ACC-12345", response.Attributes["accountId"], "accountId attribute should match")

	// Assert authorizations (should be empty array)
	assert.NotNil(t, response.Authorizations, "Authorizations should not be nil")
	assert.Empty(t, response.Authorizations, "Authorizations should be empty")

	t.Logf("✓ Successfully retrieved consent %s with all fields", response.ID)
}

// TestGetConsent_NotFound tests GET request for non-existent consent
func TestGetConsent_NotFound(t *testing.T) {
	env := SetupTestEnvironment(t)

	nonExistentID := "CONSENT-nonexistent-12345"

	req, err := http.NewRequest("GET", "/api/v1/consents/"+nonExistentID, nil)
	require.NoError(t, err)
	req.Header.Set("org-id", "TEST_ORG")
	req.Header.Set("client-id", "TEST_CLIENT")

	recorder := httptest.NewRecorder()
	env.Router.ServeHTTP(recorder, req)

	// Assert 404 response
	assert.Equal(t, http.StatusNotFound, recorder.Code, "Should return 404 Not Found")

	var errorResponse map[string]interface{}
	err = json.Unmarshal(recorder.Body.Bytes(), &errorResponse)
	require.NoError(t, err)

	// Check for either "error" or "code" field (different error response formats)
	hasError := errorResponse["error"] != nil || errorResponse["code"] != nil
	assert.True(t, hasError, "Response should contain error or code field")
	t.Logf("✓ Correctly returned 404 for non-existent consent")
}

// TestGetConsent_DifferentOrgID tests that consent from different org cannot be accessed
func TestGetConsent_DifferentOrgID(t *testing.T) {
	env := SetupTestEnvironment(t)

	// Create test purpose
	purposes := CreateTestPurposes(t, env, map[string]string{
		"data_access": "Data access purpose",
	})
	defer CleanupTestPurposes(t, env, purposes)

	// Create consent with TEST_ORG
	createReq := &models.ConsentAPIRequest{
		Type: "accounts",
		ConsentPurpose: []models.ConsentPurposeItem{
			{Name: "data_access", Value: "Test", IsSelected: BoolPtr(true)},
		},
	}

	reqBody, err := json.Marshal(createReq)
	require.NoError(t, err)

	createHTTPReq, err := http.NewRequest("POST", "/api/v1/consents", bytes.NewBuffer(reqBody))
	require.NoError(t, err)
	createHTTPReq.Header.Set("Content-Type", "application/json")
	createHTTPReq.Header.Set("org-id", "TEST_ORG")
	createHTTPReq.Header.Set("client-id", "TEST_CLIENT")

	createRecorder := httptest.NewRecorder()
	env.Router.ServeHTTP(createRecorder, createHTTPReq)
	require.Equal(t, http.StatusCreated, createRecorder.Code)

	var createResp models.ConsentAPIResponse
	err = json.Unmarshal(createRecorder.Body.Bytes(), &createResp)
	require.NoError(t, err)
	defer CleanupTestData(t, env, createResp.ID)

	// Try to GET with different org-id
	req, err := http.NewRequest("GET", "/api/v1/consents/"+createResp.ID, nil)
	require.NoError(t, err)
	req.Header.Set("org-id", "DIFFERENT_ORG")
	req.Header.Set("client-id", "TEST_CLIENT")

	recorder := httptest.NewRecorder()
	env.Router.ServeHTTP(recorder, req)

	// Should return 404 (consent not found for this org)
	assert.Equal(t, http.StatusNotFound, recorder.Code, "Should return 404 when org doesn't match")
	t.Logf("✓ Correctly prevented access to consent from different org")
}

// TestGetConsent_WithValidityTime tests retrieval of consent with validityTime field
func TestGetConsent_WithValidityTime(t *testing.T) {
	env := SetupTestEnvironment(t)

	purposes := CreateTestPurposes(t, env, map[string]string{
		"data_access": "Data access purpose",
	})
	defer CleanupTestPurposes(t, env, purposes)

	validityTime := int64(7776000) // 90 days
	frequency := 10
	recurringIndicator := true

	createReq := &models.ConsentAPIRequest{
		Type:               "accounts",
		ValidityTime:       &validityTime,
		Frequency:          &frequency,
		RecurringIndicator: &recurringIndicator,
		ConsentPurpose: []models.ConsentPurposeItem{
			{Name: "data_access", Value: "Test", IsSelected: BoolPtr(true)},
		},
	}

	reqBody, err := json.Marshal(createReq)
	require.NoError(t, err)

	createHTTPReq, err := http.NewRequest("POST", "/api/v1/consents", bytes.NewBuffer(reqBody))
	require.NoError(t, err)
	createHTTPReq.Header.Set("Content-Type", "application/json")
	createHTTPReq.Header.Set("org-id", "TEST_ORG")
	createHTTPReq.Header.Set("client-id", "TEST_CLIENT")

	createRecorder := httptest.NewRecorder()
	env.Router.ServeHTTP(createRecorder, createHTTPReq)
	require.Equal(t, http.StatusCreated, createRecorder.Code)

	var createResp models.ConsentAPIResponse
	err = json.Unmarshal(createRecorder.Body.Bytes(), &createResp)
	require.NoError(t, err)
	defer CleanupTestData(t, env, createResp.ID)

	// GET the consent
	req, err := http.NewRequest("GET", "/api/v1/consents/"+createResp.ID, nil)
	require.NoError(t, err)
	req.Header.Set("org-id", "TEST_ORG")
	req.Header.Set("client-id", "TEST_CLIENT")

	recorder := httptest.NewRecorder()
	env.Router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)

	var response models.ConsentAPIResponse
	err = json.Unmarshal(recorder.Body.Bytes(), &response)
	require.NoError(t, err)

	// Assert validity fields
	assert.NotNil(t, response.ValidityTime, "ValidityTime should not be nil")
	assert.Equal(t, validityTime, *response.ValidityTime, "ValidityTime should match")
	assert.NotNil(t, response.Frequency, "Frequency should not be nil")
	assert.Equal(t, frequency, *response.Frequency, "Frequency should match")
	assert.NotNil(t, response.RecurringIndicator, "RecurringIndicator should not be nil")
	assert.Equal(t, recurringIndicator, *response.RecurringIndicator, "RecurringIndicator should match")

	t.Logf("✓ Successfully retrieved consent with validityTime=%d", validityTime)
}

// TestGetConsent_WithDataAccessValidityDuration tests retrieval with dataAccessValidityDuration
func TestGetConsent_WithDataAccessValidityDuration(t *testing.T) {
	env := SetupTestEnvironment(t)

	purposes := CreateTestPurposes(t, env, map[string]string{
		"data_access": "Data access purpose",
	})
	defer CleanupTestPurposes(t, env, purposes)

	duration := int64(86400) // 1 day

	createReq := &models.ConsentAPIRequest{
		Type:                       "accounts",
		DataAccessValidityDuration: &duration,
		ConsentPurpose: []models.ConsentPurposeItem{
			{Name: "data_access", Value: "Test", IsSelected: BoolPtr(true)},
		},
	}

	reqBody, err := json.Marshal(createReq)
	require.NoError(t, err)

	createHTTPReq, err := http.NewRequest("POST", "/api/v1/consents", bytes.NewBuffer(reqBody))
	require.NoError(t, err)
	createHTTPReq.Header.Set("Content-Type", "application/json")
	createHTTPReq.Header.Set("org-id", "TEST_ORG")
	createHTTPReq.Header.Set("client-id", "TEST_CLIENT")

	createRecorder := httptest.NewRecorder()
	env.Router.ServeHTTP(createRecorder, createHTTPReq)
	require.Equal(t, http.StatusCreated, createRecorder.Code)

	var createResp models.ConsentAPIResponse
	err = json.Unmarshal(createRecorder.Body.Bytes(), &createResp)
	require.NoError(t, err)
	defer CleanupTestData(t, env, createResp.ID)

	// GET the consent
	req, err := http.NewRequest("GET", "/api/v1/consents/"+createResp.ID, nil)
	require.NoError(t, err)
	req.Header.Set("org-id", "TEST_ORG")
	req.Header.Set("client-id", "TEST_CLIENT")

	recorder := httptest.NewRecorder()
	env.Router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)

	var response models.ConsentAPIResponse
	err = json.Unmarshal(recorder.Body.Bytes(), &response)
	require.NoError(t, err)

	// Assert dataAccessValidityDuration
	assert.NotNil(t, response.DataAccessValidityDuration, "DataAccessValidityDuration should not be nil")
	assert.Equal(t, duration, *response.DataAccessValidityDuration, "DataAccessValidityDuration should match")

	t.Logf("✓ Successfully retrieved consent with dataAccessValidityDuration=%d", duration)
}

// TestGetConsent_WithAuthorizationResources tests retrieval of consent with detailed authorization resources
func TestGetConsent_WithAuthorizationResources(t *testing.T) {
	env := SetupTestEnvironment(t)
	
	purposes := CreateTestPurposes(t, env, map[string]string{
		"payment_initiation": "Payment initiation purpose",
	})
	defer CleanupTestPurposes(t, env, purposes)

	// Create consent with authorization resources
	createReq := &models.ConsentAPIRequest{
		Type: "payments",
		ConsentPurpose: []models.ConsentPurposeItem{
			{Name: "payment_initiation", Value: "Payment consent", IsSelected: BoolPtr(true)},
		},
		Authorizations: []models.AuthorizationAPIRequest{
			{
				UserID: "user-123",
				Type:   "authorization_code",
				Status: "APPROVED",
			},
		},
	}

	reqBody, err := json.Marshal(createReq)
	require.NoError(t, err)

	createHTTPReq, err := http.NewRequest("POST", "/api/v1/consents", bytes.NewBuffer(reqBody))
	require.NoError(t, err)
	createHTTPReq.Header.Set("Content-Type", "application/json")
	createHTTPReq.Header.Set("org-id", "TEST_ORG")
	createHTTPReq.Header.Set("client-id", "TEST_CLIENT")

	createRecorder := httptest.NewRecorder()
	env.Router.ServeHTTP(createRecorder, createHTTPReq)
	require.Equal(t, http.StatusCreated, createRecorder.Code)

	var createResp models.ConsentAPIResponse
	err = json.Unmarshal(createRecorder.Body.Bytes(), &createResp)
	require.NoError(t, err)
	defer CleanupTestData(t, env, createResp.ID)

	// GET the consent
	req, err := http.NewRequest("GET", "/api/v1/consents/"+createResp.ID, nil)
	require.NoError(t, err)
	req.Header.Set("org-id", "TEST_ORG")
	req.Header.Set("client-id", "TEST_CLIENT")

	recorder := httptest.NewRecorder()
	env.Router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)

	var response models.ConsentAPIResponse
	err = json.Unmarshal(recorder.Body.Bytes(), &response)
	require.NoError(t, err)

	// Assert basic consent fields
	assert.Equal(t, createResp.ID, response.ID)
	assert.Equal(t, "payments", response.Type)
	assert.Equal(t, "ACTIVE", response.Status, "Consent status should be ACTIVE when authorization is APPROVED")

	// Assert authorizations
	require.Len(t, response.Authorizations, 1, "Should have 1 authorization")
	
	auth := response.Authorizations[0]
	assert.NotEmpty(t, auth.ID, "Authorization ID should not be empty")
	assert.NotNil(t, auth.UserID, "UserID should not be nil")
	assert.Equal(t, "user-123", *auth.UserID, "UserID should match")
	assert.Equal(t, "authorization_code", auth.Type, "Authorization type should match")
	assert.Equal(t, "APPROVED", auth.Status, "Authorization status should match")
	assert.NotEmpty(t, auth.UpdatedTime, "UpdatedTime should not be empty")

	t.Logf("✓ Successfully retrieved consent %s with authorization resources", response.ID)
}

// TestGetConsent_WithMultipleAuthorizationStatuses tests consent with multiple authorizations
func TestGetConsent_WithMultipleAuthorizationStatuses(t *testing.T) {
	env := SetupTestEnvironment(t)
	
	purposes := CreateTestPurposes(t, env, map[string]string{
		"data_access": "Data access purpose",
	})
	defer CleanupTestPurposes(t, env, purposes)

	// Create consent with multiple authorizations
	createReq := &models.ConsentAPIRequest{
		Type: "accounts",
		ConsentPurpose: []models.ConsentPurposeItem{
			{Name: "data_access", Value: "Account data", IsSelected: BoolPtr(true)},
		},
		Authorizations: []models.AuthorizationAPIRequest{
			{
				UserID: "user-001",
				Type:   "authorization_code",
				Status: "APPROVED",
			},
			{
				UserID: "user-002",
				Type:   "authorization_code",
				Status: "REJECTED",
			},
		},
	}

	reqBody, err := json.Marshal(createReq)
	require.NoError(t, err)

	createHTTPReq, err := http.NewRequest("POST", "/api/v1/consents", bytes.NewBuffer(reqBody))
	require.NoError(t, err)
	createHTTPReq.Header.Set("Content-Type", "application/json")
	createHTTPReq.Header.Set("org-id", "TEST_ORG")
	createHTTPReq.Header.Set("client-id", "TEST_CLIENT")

	createRecorder := httptest.NewRecorder()
	env.Router.ServeHTTP(createRecorder, createHTTPReq)
	require.Equal(t, http.StatusCreated, createRecorder.Code)

	var createResp models.ConsentAPIResponse
	err = json.Unmarshal(createRecorder.Body.Bytes(), &createResp)
	require.NoError(t, err)
	defer CleanupTestData(t, env, createResp.ID)

	// GET the consent
	req, err := http.NewRequest("GET", "/api/v1/consents/"+createResp.ID, nil)
	require.NoError(t, err)
	req.Header.Set("org-id", "TEST_ORG")
	req.Header.Set("client-id", "TEST_CLIENT")

	recorder := httptest.NewRecorder()
	env.Router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)

	var response models.ConsentAPIResponse
	err = json.Unmarshal(recorder.Body.Bytes(), &response)
	require.NoError(t, err)

	// Assert authorizations
	require.Len(t, response.Authorizations, 2, "Should have 2 authorizations")
	
	// Verify both authorizations are present
	statuses := make(map[string]bool)
	for _, auth := range response.Authorizations {
		assert.NotEmpty(t, auth.ID, "Authorization ID should not be empty")
		assert.NotNil(t, auth.UserID, "UserID should not be nil")
		statuses[auth.Status] = true
	}

	assert.True(t, statuses["APPROVED"], "Should have an APPROVED authorization")
	assert.True(t, statuses["REJECTED"], "Should have a REJECTED authorization")

	// Consent status should reflect the "highest priority" authorization status
	// (typically REJECTED or the most restrictive one)
	assert.NotEmpty(t, response.Status, "Consent status should be set")

	t.Logf("✓ Successfully retrieved consent with multiple authorization statuses, consent status: %s", response.Status)
}
