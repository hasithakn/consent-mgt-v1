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
			{Name: "data_access", Value: "Account information", IsUserApproved: BoolPtr(true), IsMandatory: BoolPtr(true)},
			{Name: "transaction_history", Value: "Transaction details", IsUserApproved: BoolPtr(false), IsMandatory: BoolPtr(false)},
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
	assert.NotNil(t, dataAccess.IsUserApproved, "data_access IsUserApproved should not be nil")
	assert.True(t, *dataAccess.IsUserApproved, "data_access should be selected")
	assert.NotNil(t, dataAccess.IsMandatory, "IsMandatory should not be nil")
	assert.True(t, *dataAccess.IsMandatory, "data_access should be mandatory")

	// Verify transaction_history purpose
	txHistory, exists := purposeMap["transaction_history"]
	assert.True(t, exists, "transaction_history purpose should exist")
	assert.Equal(t, "Transaction details", txHistory.Value, "transaction_history value should match")
	assert.NotNil(t, txHistory.IsUserApproved, "transaction_history IsUserApproved should not be nil")
	assert.False(t, *txHistory.IsUserApproved, "transaction_history should not be selected")
	assert.NotNil(t, txHistory.IsMandatory, "IsMandatory should not be nil")
	assert.False(t, *txHistory.IsMandatory, "transaction_history should not be mandatory")

	// Assert attributes
	require.NotNil(t, response.Attributes, "Attributes should not be nil")
	assert.Equal(t, "mobile", response.Attributes["channel"], "channel attribute should match")
	assert.Equal(t, "US", response.Attributes["country"], "country attribute should match")
	assert.Equal(t, "ACC-12345", response.Attributes["accountId"], "accountId attribute should match")

	// Assert authorizations (should be empty array)
	assert.NotNil(t, response.Authorizations, "Authorizations should not be nil")
	assert.Empty(t, response.Authorizations, "Authorizations should be empty")

	t.Logf("✓ Successfully retrieved consent with attributes")
}

// TestGetConsent_AllFieldsReturned tests that all fields are returned even when empty (no omitempty)
func TestGetConsent_AllFieldsReturned(t *testing.T) {
	env := SetupTestEnvironment(t)
	
	// Create test purpose
	purposes := CreateTestPurposes(t, env, map[string]string{
		"minimal_purpose": "Minimal purpose for testing",
	})
	defer CleanupTestPurposes(t, env, purposes)
	
	// Create a minimal consent without optional fields
	createReq := &models.ConsentAPIRequest{
		Type: "accounts",
		ConsentPurpose: []models.ConsentPurposeItem{
			{Name: "minimal_purpose", Value: "test", IsUserApproved: BoolPtr(true), IsMandatory: BoolPtr(true)},
		},
		// No Attributes, no Authorizations, no DataAccessValidityDuration
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
	consentID := createResp.ID

	// Get the consent
	getReq, err := http.NewRequest("GET", "/api/v1/consents/"+consentID, nil)
	require.NoError(t, err)
	getReq.Header.Set("org-id", "TEST_ORG")
	getReq.Header.Set("client-id", "TEST_CLIENT")

	getRecorder := httptest.NewRecorder()
	env.Router.ServeHTTP(getRecorder, getReq)
	require.Equal(t, http.StatusOK, getRecorder.Code)

	// Parse response as raw JSON to check field presence
	var rawResponse map[string]interface{}
	err = json.Unmarshal(getRecorder.Body.Bytes(), &rawResponse)
	require.NoError(t, err)

	// Verify that all fields are present in JSON, even if empty
	assert.Contains(t, rawResponse, "consentPurpose", "consentPurpose field should be present")
	assert.Contains(t, rawResponse, "dataAccessValidityDuration", "dataAccessValidityDuration field should be present")
	
	// Verify consentPurpose is an array (not null) - should have 1 purpose we created
	consentPurpose, ok := rawResponse["consentPurpose"].([]interface{})
	assert.True(t, ok, "consentPurpose should be an array")
	assert.NotNil(t, consentPurpose, "consentPurpose should not be null")
	assert.Len(t, consentPurpose, 1, "consentPurpose should have 1 purpose")
	
	// Verify dataAccessValidityDuration is null (not omitted)
	assert.Nil(t, rawResponse["dataAccessValidityDuration"], "dataAccessValidityDuration should be null (not omitted)")

	t.Log("✓ All fields are present in response, no fields omitted")
}

// TestGetConsent_AuthorizationResourcesAlwaysPresent tests that authorization resources are always an object
func TestGetConsent_AuthorizationResourcesAlwaysPresent(t *testing.T) {
	env := SetupTestEnvironment(t)
	
	// Create test purpose
	purposes := CreateTestPurposes(t, env, map[string]string{
		"minimal_purpose": "Minimal purpose for testing",
	})
	defer CleanupTestPurposes(t, env, purposes)
	
	// Create consent with authorization but no resources
	createReq := &models.ConsentAPIRequest{
		Type: "accounts",
		ConsentPurpose: []models.ConsentPurposeItem{
			{Name: "minimal_purpose", Value: "test", IsUserApproved: BoolPtr(true), IsMandatory: BoolPtr(true)},
		},
		Authorizations: []models.AuthorizationAPIRequest{
			{
				Type:   "authorization",
				Status: "APPROVED",
				// No Resources field
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

	// Verify authorizations are present
	require.NotNil(t, createResp.Authorizations, "Authorizations should not be nil")
	require.Len(t, createResp.Authorizations, 1, "Should have 1 authorization")
	
	// Verify resources field is present and is an object (not null)
	auth := createResp.Authorizations[0]
	assert.NotNil(t, auth.Resources, "Resources should not be nil")
	
	// Resources should be an empty object, not null
	resources, ok := auth.Resources.(map[string]interface{})
	assert.True(t, ok, "Resources should be a map/object")
	assert.NotNil(t, resources, "Resources should be an empty object, not null")
	assert.Len(t, resources, 0, "Resources should be an empty object")

	t.Log("✓ Authorization resources field is always present as an object")
}

// TestGetConsent_ExpiryCheck tests that expired consents get EXPIRED status during GET
func TestGetConsent_ExpiryCheck(t *testing.T) {
	env := SetupTestEnvironment(t)
	
	// Create test purpose
	purposes := CreateTestPurposes(t, env, map[string]string{
		"test_purpose": "Test purpose",
	})
	defer CleanupTestPurposes(t, env, purposes)
	
	// Create a consent with an expired validity time (in the past)
	expiredValidityTime := int64(1000) // Very old timestamp (1970)
	createReq := &models.ConsentAPIRequest{
		Type:         "accounts",
		ValidityTime: &expiredValidityTime,
		ConsentPurpose: []models.ConsentPurposeItem{
			{Name: "test_purpose", Value: "test", IsUserApproved: BoolPtr(true), IsMandatory: BoolPtr(true)},
		},
		Authorizations: []models.AuthorizationAPIRequest{
			{
				Type:   "authorization",
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
	consentID := createResp.ID

	// Initial status might be APPROVED from authorization
	t.Logf("Initial consent status: %s", createResp.Status)

	// Get the consent (should detect expiry and update status to EXPIRED)
	getReq, err := http.NewRequest("GET", "/api/v1/consents/"+consentID, nil)
	require.NoError(t, err)
	getReq.Header.Set("org-id", "TEST_ORG")
	getReq.Header.Set("client-id", "TEST_CLIENT")

	getRecorder := httptest.NewRecorder()
	env.Router.ServeHTTP(getRecorder, getReq)
	require.Equal(t, http.StatusOK, getRecorder.Code)

	var getResp models.ConsentAPIResponse
	err = json.Unmarshal(getRecorder.Body.Bytes(), &getResp)
	require.NoError(t, err)

	// Verify status is EXPIRED
	assert.Equal(t, "EXPIRED", getResp.Status, "Consent status should be EXPIRED due to expired validityTime")

	// Get again to ensure status persisted
	getRecorder2 := httptest.NewRecorder()
	env.Router.ServeHTTP(getRecorder2, getReq)
	require.Equal(t, http.StatusOK, getRecorder2.Code)

	var getResp2 models.ConsentAPIResponse
	err = json.Unmarshal(getRecorder2.Body.Bytes(), &getResp2)
	require.NoError(t, err)

	// Verify status is still EXPIRED
	assert.Equal(t, "EXPIRED", getResp2.Status, "Consent status should remain EXPIRED")

	t.Log("✓ Expired consent correctly gets EXPIRED status during GET and persists")
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
			{Name: "data_access", Value: "Test", IsUserApproved: BoolPtr(true), IsMandatory: BoolPtr(true)},
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
			{Name: "data_access", Value: "Test", IsUserApproved: BoolPtr(true), IsMandatory: BoolPtr(true)},
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
			{Name: "data_access", Value: "Test", IsUserApproved: BoolPtr(true), IsMandatory: BoolPtr(true)},
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
			{Name: "payment_initiation", Value: "Payment consent", IsUserApproved: BoolPtr(true), IsMandatory: BoolPtr(true)},
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
			{Name: "data_access", Value: "Account data", IsUserApproved: BoolPtr(true), IsMandatory: BoolPtr(true)},
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
