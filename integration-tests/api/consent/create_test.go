package consent

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wso2/consent-management-api/internal/models"
)

// TestCreateConsent_Success tests successful consent creation with basic fields
func TestCreateConsent_Success(t *testing.T) {
	env := SetupTestEnvironment(t)

	// Create test purposes
	purposes := CreateTestPurposes(t, env, map[string]string{
		"test_data_access":  "Test data access purpose",
		"test_account_info": "Test account info purpose",
	})
	defer CleanupTestPurposes(t, env, purposes)

	// Prepare request with isUserApproved field
	// validityTime should be a future timestamp (in milliseconds or seconds from epoch)
	// Set to 90 days from now in milliseconds
	validityTime := time.Now().Add(90 * 24 * time.Hour).UnixMilli()
	frequency := 1
	recurringIndicator := false

	createReq := &models.ConsentAPIRequest{
		Type:               "accounts",
		ValidityTime:       &validityTime,
		RecurringIndicator: &recurringIndicator,
		Frequency:          &frequency,
		ConsentPurpose: []models.ConsentPurposeItem{
			{Name: "test_data_access", Value: "Read account data", IsUserApproved: BoolPtr(true), IsMandatory: BoolPtr(true)},
			{Name: "test_account_info", Value: "Read account info", IsUserApproved: BoolPtr(false), IsMandatory: BoolPtr(false)},
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

	// Log error response if not successful
	if recorder.Code != http.StatusCreated {
		t.Logf("Create consent failed with status %d: %s", recorder.Code, recorder.Body.String())
	}

	// Assert response
	assert.Equal(t, http.StatusCreated, recorder.Code, "Expected 201 Created status")

	var response models.ConsentAPIResponse
	err = json.Unmarshal(recorder.Body.Bytes(), &response)
	require.NoError(t, err)

	// Verify all response fields
	assert.NotEmpty(t, response.ID, "Consent ID should not be empty")
	assert.Equal(t, "accounts", response.Type, "Type should match")
	assert.Equal(t, "TEST_CLIENT", response.ClientID, "ClientID should match")
	assert.Equal(t, "CREATED", response.Status, "Status should be CREATED (no authorizations)")

	// Verify timestamps
	assert.Greater(t, response.CreatedTime, int64(0), "CreatedTime should be positive")
	assert.Greater(t, response.UpdatedTime, int64(0), "UpdatedTime should be positive")

	// Verify consent fields
	assert.NotNil(t, response.ValidityTime, "ValidityTime should not be nil")
	assert.Equal(t, validityTime, *response.ValidityTime, "ValidityTime should match")
	assert.NotNil(t, response.RecurringIndicator, "RecurringIndicator should not be nil")
	assert.Equal(t, recurringIndicator, *response.RecurringIndicator, "RecurringIndicator should match")
	assert.NotNil(t, response.Frequency, "Frequency should not be nil")
	assert.Equal(t, frequency, *response.Frequency, "Frequency should match")
	assert.Nil(t, response.DataAccessValidityDuration, "DataAccessValidityDuration should be nil")

	// Verify consent purposes
	assert.NotNil(t, response.ConsentPurpose, "ConsentPurpose should not be nil")
	assert.Len(t, response.ConsentPurpose, 2, "Should have 2 consent purposes")

	// Verify isUserApproved field and all purpose fields
	foundDataAccess := false
	foundAccountInfo := false
	for _, cp := range response.ConsentPurpose {
		assert.NotEmpty(t, cp.Name, "Purpose name should not be empty")
		assert.NotEmpty(t, cp.Value, "Purpose value should not be empty")

		if cp.Name == "test_data_access" {
			foundDataAccess = true
			assert.Equal(t, "Read account data", cp.Value, "Value should match")
			assert.NotNil(t, cp.IsUserApproved, "IsUserApproved should not be nil")
			assert.True(t, *cp.IsUserApproved, "test_data_access should be selected")
			assert.NotNil(t, cp.IsMandatory, "IsMandatory should not be nil")
			assert.True(t, *cp.IsMandatory, "test_data_access should be mandatory")
		} else if cp.Name == "test_account_info" {
			foundAccountInfo = true
			assert.Equal(t, "Read account info", cp.Value, "Value should match")
			assert.NotNil(t, cp.IsUserApproved, "IsUserApproved should not be nil")
			assert.False(t, *cp.IsUserApproved, "test_account_info should not be selected")
			assert.NotNil(t, cp.IsMandatory, "IsMandatory should not be nil")
			assert.False(t, *cp.IsMandatory, "test_account_info should not be mandatory")
		}
	}
	assert.True(t, foundDataAccess, "Should find test_data_access purpose")
	assert.True(t, foundAccountInfo, "Should find test_account_info purpose")

	// Verify attributes
	assert.NotNil(t, response.Attributes, "Attributes should not be nil")
	assert.Equal(t, "api-test", response.Attributes["source"], "Source attribute should match")

	// Verify authorizations (should be empty)
	assert.NotNil(t, response.Authorizations, "Authorizations should not be nil")
	assert.Empty(t, response.Authorizations, "Authorizations should be empty")

	// Verify consent by retrieving it via GET API
	getReq, err := http.NewRequest("GET", "/api/v1/consents/"+response.ID, nil)
	require.NoError(t, err)
	getReq.Header.Set("org-id", "TEST_ORG")
	getReq.Header.Set("client-id", "TEST_CLIENT")

	getRecorder := httptest.NewRecorder()
	env.Router.ServeHTTP(getRecorder, getReq)

	assert.Equal(t, http.StatusOK, getRecorder.Code, "Should retrieve created consent")

	var retrievedConsent models.ConsentAPIResponse
	err = json.Unmarshal(getRecorder.Body.Bytes(), &retrievedConsent)
	require.NoError(t, err)

	// Verify all fields match the created consent
	assert.Equal(t, response.ID, retrievedConsent.ID, "ID should match")
	assert.Equal(t, response.Type, retrievedConsent.Type, "Type should match")
	assert.Equal(t, response.ClientID, retrievedConsent.ClientID, "ClientID should match")
	assert.Equal(t, response.Status, retrievedConsent.Status, "Status should match")
	assert.Equal(t, response.CreatedTime, retrievedConsent.CreatedTime, "CreatedTime should match")
	assert.Equal(t, response.UpdatedTime, retrievedConsent.UpdatedTime, "UpdatedTime should match")
	assert.Equal(t, *response.ValidityTime, *retrievedConsent.ValidityTime, "ValidityTime should match")
	assert.Equal(t, *response.RecurringIndicator, *retrievedConsent.RecurringIndicator, "RecurringIndicator should match")
	assert.Equal(t, *response.Frequency, *retrievedConsent.Frequency, "Frequency should match")
	assert.Len(t, retrievedConsent.ConsentPurpose, 2, "Retrieved consent should have 2 purposes")

	// Verify purposes in GET response
	for _, cp := range retrievedConsent.ConsentPurpose {
		if cp.Name == "test_data_access" {
			assert.Equal(t, "Read account data", cp.Value, "Value should match in GET")
			assert.NotNil(t, cp.IsUserApproved, "IsUserApproved should not be nil in GET")
			assert.True(t, *cp.IsUserApproved, "IsUserApproved should be true in GET")
			assert.NotNil(t, cp.IsMandatory, "IsMandatory should not be nil")
			assert.True(t, *cp.IsMandatory, "IsUserApproved should be true in GET")
		} else if cp.Name == "test_account_info" {
			assert.Equal(t, "Read account info", cp.Value, "Value should match in GET")
			assert.NotNil(t, cp.IsUserApproved, "IsUserApproved should not be nil in GET")
			assert.False(t, *cp.IsUserApproved, "IsUserApproved should be false in GET")
			assert.NotNil(t, cp.IsMandatory, "IsMandatory should not be nil")
			assert.False(t, *cp.IsMandatory, "IsUserApproved should be false in GET")
		}
	}
	assert.Equal(t, response.Attributes["source"], retrievedConsent.Attributes["source"], "Attributes should match")
	assert.Empty(t, retrievedConsent.Authorizations, "Authorizations should still be empty in GET")

	t.Logf("✓ Successfully created consent %s with isUserApproved fields", response.ID)

	// Cleanup
	CleanupTestData(t, env, response.ID)
}

// TestCreateConsent_WithAuthResources tests consent creation with authorization resources
func TestCreateConsent_WithAuthResources(t *testing.T) {
	env := SetupTestEnvironment(t)

	// Create test purpose
	purposes := CreateTestPurposes(t, env, map[string]string{
		"auth_resources_test": "Test purpose with auth resources",
	})
	defer CleanupTestPurposes(t, env, purposes)

	validityTime := time.Now().Add(30 * 24 * time.Hour).UnixMilli() // 30 days from now in milliseconds
	recurringIndicator := true
	frequency := 5

	// Updated to use resources field instead of approvedPurposeDetails
	createReq := &models.ConsentAPIRequest{
		Type:               "payments",
		ValidityTime:       &validityTime,
		RecurringIndicator: &recurringIndicator,
		Frequency:          &frequency,
		ConsentPurpose: []models.ConsentPurposeItem{
			{Name: "auth_resources_test", Value: "consent with auth", IsUserApproved: BoolPtr(true), IsMandatory: BoolPtr(true)},
		},
		Attributes: map[string]string{
			"test": "value",
		},
		Authorizations: []models.AuthorizationAPIRequest{
			{
				UserID: "user-789",
				Type:   "authorization_code",
				Status: "APPROVED", // Use uppercase status
				Resources: map[string]interface{}{
					"accountIds":       []string{"ACC-001", "ACC-002"},
					"permissions":      []string{"read", "write"},
					"additionalScopes": "utility_read taxes_read",
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
	if recorder.Code != http.StatusCreated {
		t.Logf("Create consent with auth failed with status %d: %s", recorder.Code, recorder.Body.String())
	}
	assert.Equal(t, http.StatusCreated, recorder.Code)

	var response models.ConsentAPIResponse
	err = json.Unmarshal(recorder.Body.Bytes(), &response)
	require.NoError(t, err)

	// Verify all consent fields
	assert.NotEmpty(t, response.ID, "Consent ID should not be empty")
	assert.Equal(t, "payments", response.Type, "Type should match")
	assert.Equal(t, "TEST_CLIENT", response.ClientID, "ClientID should match")
	assert.Equal(t, "ACTIVE", response.Status, "Status should be ACTIVE (has approved authorization)")

	// Verify timestamps
	assert.Greater(t, response.CreatedTime, int64(0), "CreatedTime should be positive")
	assert.Greater(t, response.UpdatedTime, int64(0), "UpdatedTime should be positive")

	// Verify consent metadata
	assert.NotNil(t, response.ValidityTime, "ValidityTime should not be nil")
	assert.Equal(t, validityTime, *response.ValidityTime, "ValidityTime should match")
	assert.NotNil(t, response.RecurringIndicator, "RecurringIndicator should not be nil")
	assert.Equal(t, recurringIndicator, *response.RecurringIndicator, "RecurringIndicator should match")
	assert.NotNil(t, response.Frequency, "Frequency should not be nil")
	assert.Equal(t, frequency, *response.Frequency, "Frequency should match")

	// Verify consent purpose
	assert.NotNil(t, response.ConsentPurpose, "ConsentPurpose should not be nil")
	assert.Len(t, response.ConsentPurpose, 1, "Should have 1 consent purpose")
	assert.Equal(t, "auth_resources_test", response.ConsentPurpose[0].Name, "Purpose name should match")
	assert.Equal(t, "consent with auth", response.ConsentPurpose[0].Value, "Purpose value should match")
	assert.NotNil(t, response.ConsentPurpose[0].IsUserApproved, "IsUserApproved should not be nil")
	assert.True(t, *response.ConsentPurpose[0].IsUserApproved, "Purpose should be selected")

	// Verify attributes
	assert.NotNil(t, response.Attributes, "Attributes should not be nil")
	assert.Equal(t, "value", response.Attributes["test"], "Test attribute should match")

	// Verify authorizations - comprehensive check
	assert.NotNil(t, response.Authorizations, "Authorizations should not be nil")
	require.Len(t, response.Authorizations, 1, "Should have 1 authorization")

	auth := response.Authorizations[0]
	assert.NotEmpty(t, auth.ID, "Authorization ID should not be empty")
	assert.Equal(t, "authorization_code", auth.Type, "Authorization type should match")
	assert.Equal(t, "APPROVED", auth.Status, "Authorization status should be APPROVED (uppercase)")
	assert.NotNil(t, auth.UserID, "UserID should not be nil")
	assert.Equal(t, "user-789", *auth.UserID, "UserID should match")
	assert.Greater(t, auth.UpdatedTime, int64(0), "Authorization UpdatedTime should be positive")

	// Verify resources field (instead of approvedPurposeDetails)
	assert.NotNil(t, auth.Resources, "Resources should not be nil")
	resourcesMap, ok := auth.Resources.(map[string]interface{})
	require.True(t, ok, "Resources should be a map")
	assert.Contains(t, resourcesMap, "accountIds", "Resources should contain accountIds")
	assert.Contains(t, resourcesMap, "permissions", "Resources should contain permissions")
	assert.Contains(t, resourcesMap, "additionalScopes", "Resources should contain additionalScopes")

	// Verify specific resource values
	accountIds, ok := resourcesMap["accountIds"].([]interface{})
	require.True(t, ok, "accountIds should be an array")
	assert.Len(t, accountIds, 2, "Should have 2 account IDs")
	assert.Equal(t, "ACC-001", accountIds[0], "First account ID should match")
	assert.Equal(t, "ACC-002", accountIds[1], "Second account ID should match")

	permissions, ok := resourcesMap["permissions"].([]interface{})
	require.True(t, ok, "permissions should be an array")
	assert.Len(t, permissions, 2, "Should have 2 permissions")
	assert.Contains(t, permissions, "read", "Should contain read permission")
	assert.Contains(t, permissions, "write", "Should contain write permission")

	additionalScopes, ok := resourcesMap["additionalScopes"].(string)
	require.True(t, ok, "additionalScopes should be a string")
	assert.Equal(t, "utility_read taxes_read", additionalScopes, "additionalScopes should match")

	// Verify by retrieving the consent via GET API
	getReq, err := http.NewRequest("GET", "/api/v1/consents/"+response.ID, nil)
	require.NoError(t, err)
	getReq.Header.Set("org-id", "TEST_ORG")
	getReq.Header.Set("client-id", "TEST_CLIENT")

	getRecorder := httptest.NewRecorder()
	env.Router.ServeHTTP(getRecorder, getReq)

	assert.Equal(t, http.StatusOK, getRecorder.Code, "Should retrieve created consent")

	var retrievedConsent models.ConsentAPIResponse
	err = json.Unmarshal(getRecorder.Body.Bytes(), &retrievedConsent)
	require.NoError(t, err)

	// Verify all fields match
	assert.Equal(t, response.ID, retrievedConsent.ID, "ID should match")
	assert.Equal(t, response.Type, retrievedConsent.Type, "Type should match")
	assert.Equal(t, response.ClientID, retrievedConsent.ClientID, "ClientID should match")
	assert.Equal(t, response.Status, retrievedConsent.Status, "Status should match")
	assert.Equal(t, response.CreatedTime, retrievedConsent.CreatedTime, "CreatedTime should match")

	// Verify authorizations in GET response
	require.Len(t, retrievedConsent.Authorizations, 1, "Retrieved consent should have 1 authorization")
	retrievedAuth := retrievedConsent.Authorizations[0]
	assert.Equal(t, auth.ID, retrievedAuth.ID, "Authorization ID should match")
	assert.Equal(t, "authorization_code", retrievedAuth.Type, "Authorization type should match")
	assert.Equal(t, "APPROVED", retrievedAuth.Status, "Authorization status should match")
	assert.NotNil(t, retrievedAuth.UserID, "UserID should not be nil in GET")
	assert.Equal(t, "user-789", *retrievedAuth.UserID, "UserID should match in GET")
	assert.NotNil(t, retrievedAuth.Resources, "Resources should be present in GET response")

	// Verify resources in GET response
	retrievedResourcesMap, ok := retrievedAuth.Resources.(map[string]interface{})
	require.True(t, ok, "Resources should be a map in GET response")
	assert.Contains(t, retrievedResourcesMap, "accountIds", "accountIds should be in GET response")
	assert.Contains(t, retrievedResourcesMap, "permissions", "permissions should be in GET response")
	assert.Contains(t, retrievedResourcesMap, "additionalScopes", "additionalScopes should be in GET response")

	t.Logf("✓ Successfully created consent %s with authorization resources", response.ID)

	// Cleanup
	CleanupTestData(t, env, response.ID)
}

// TestCreateConsent_InvalidRequest tests various invalid request scenarios
func TestCreateConsent_InvalidRequest(t *testing.T) {
	env := SetupTestEnvironment(t)

	tests := []struct {
		name           string
		requestBody    string
		expectedStatus int
		errorContains  string
	}{
		{
			name:           "Missing consent type",
			requestBody:    `{"status": "created", "consentPurpose": [{"name": "test", "value": "test", "isUserApproved": true}]}`,
			expectedStatus: http.StatusBadRequest,
			errorContains:  "type",
		},
		{
			name:           "Missing status",
			requestBody:    `{"type": "accounts", "consentPurpose": [{"name": "test", "value": "test", "isUserApproved": true}]}`,
			expectedStatus: http.StatusBadRequest,
			errorContains:  "",
		},
		{
			name:           "Missing consentPurpose",
			requestBody:    `{"type": "accounts", "status": "created"}`,
			expectedStatus: http.StatusBadRequest,
			errorContains:  "consentPurpose",
		},
		{
			name:           "Empty consentPurpose array",
			requestBody:    `{"type": "accounts", "status": "created", "consentPurpose": []}`,
			expectedStatus: http.StatusBadRequest,
			errorContains:  "consentPurpose",
		},
		{
			name:           "Invalid JSON",
			requestBody:    `{invalid json}`,
			expectedStatus: http.StatusBadRequest,
			errorContains:  "",
		},
		{
			name:           "Empty request",
			requestBody:    `{}`,
			expectedStatus: http.StatusBadRequest,
			errorContains:  "",
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

			assert.Equal(t, tt.expectedStatus, recorder.Code, "Test: %s", tt.name)

			if tt.errorContains != "" {
				body := strings.ToLower(recorder.Body.String())
				assert.Contains(t, body, strings.ToLower(tt.errorContains), "Test: %s", tt.name)
			}
		})
	}
}

// TestCreateConsent_MissingIsSelected tests that omitting both fields fails validation
// (defaults isMandatory=true, isUserApproved=false violate the validation rule)
func TestCreateConsent_MissingIsSelected(t *testing.T) {
	env := SetupTestEnvironment(t)

	// Create test purpose
	purposes := CreateTestPurposes(t, env, map[string]string{
		"test_missing_selected": "Test purpose without isUserApproved",
	})
	defer CleanupTestPurposes(t, env, purposes)

	validityTime := int64(7776000)
	frequency := 1
	recurringIndicator := false

	createReq := &models.ConsentAPIRequest{
		Type:               "accounts",
		ValidityTime:       &validityTime,
		RecurringIndicator: &recurringIndicator,
		Frequency:          &frequency,
		ConsentPurpose: []models.ConsentPurposeItem{
			{Name: "test_missing_selected", Value: "Test without isUserApproved"}, // Both fields omitted
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

	// Should fail because defaults (isMandatory=true, isUserApproved=false) violate validation
	assert.Equal(t, http.StatusBadRequest, recorder.Code, "Should return 400 for validation failure")

	var errorResponse map[string]interface{}
	err = json.Unmarshal(recorder.Body.Bytes(), &errorResponse)
	require.NoError(t, err)

	// Verify error mentions the validation rule
	errorText := strings.ToLower(errorResponse["message"].(string))
	if details, ok := errorResponse["details"].(string); ok {
		errorText += " " + strings.ToLower(details)
	}
	assert.Contains(t, errorText, "mandatory", "Error should mention mandatory")

	t.Log("✓ Correctly rejected request with default values that violate validation rule")
}

// TestCreateConsent_WithDataAccessValidityDuration tests consent creation with dataAccessValidityDuration
func TestCreateConsent_WithDataAccessValidityDuration(t *testing.T) {
	env := SetupTestEnvironment(t)

	// Create test purposes
	purposes := CreateTestPurposes(t, env, map[string]string{
		"validity_test_data":    "Test data access with validity duration",
		"validity_test_purpose": "Test purpose with validity duration",
	})
	defer CleanupTestPurposes(t, env, purposes)

	// Prepare request with dataAccessValidityDuration
	validityTime := int64(7776000) // ~90 days in seconds
	frequency := 1
	recurringIndicator := false
	dataAccessValidityDuration := int64(86400) // 24 hours

	createReq := &models.ConsentAPIRequest{
		Type:                       "accounts",
		ValidityTime:               &validityTime,
		RecurringIndicator:         &recurringIndicator,
		Frequency:                  &frequency,
		DataAccessValidityDuration: &dataAccessValidityDuration,
		ConsentPurpose: []models.ConsentPurposeItem{
			{Name: "validity_test_data", Value: "Test with dataAccessValidityDuration", IsUserApproved: BoolPtr(true), IsMandatory: BoolPtr(true)},
			{Name: "validity_test_purpose", Value: "testing", IsUserApproved: BoolPtr(true), IsMandatory: BoolPtr(true)},
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

	assert.Equal(t, http.StatusCreated, recorder.Code, "Expected 201 Created status")

	var response models.ConsentAPIResponse
	err = json.Unmarshal(recorder.Body.Bytes(), &response)
	require.NoError(t, err)

	// Verify all response fields
	assert.NotEmpty(t, response.ID, "Consent ID should not be empty")
	assert.Equal(t, "accounts", response.Type, "Type should match")
	assert.Equal(t, "TEST_CLIENT", response.ClientID, "ClientID should match")
	assert.Equal(t, "CREATED", response.Status, "Status should be CREATED")

	// Verify timestamps
	assert.Greater(t, response.CreatedTime, int64(0), "CreatedTime should be positive")
	assert.Greater(t, response.UpdatedTime, int64(0), "UpdatedTime should be positive")

	// Verify dataAccessValidityDuration - the main focus of this test
	assert.NotNil(t, response.DataAccessValidityDuration, "DataAccessValidityDuration should be present")
	assert.Equal(t, dataAccessValidityDuration, *response.DataAccessValidityDuration, "DataAccessValidityDuration should match")

	// Verify other consent fields
	assert.NotNil(t, response.ValidityTime, "ValidityTime should not be nil")
	assert.Equal(t, validityTime, *response.ValidityTime, "ValidityTime should match")
	assert.NotNil(t, response.RecurringIndicator, "RecurringIndicator should not be nil")
	assert.Equal(t, recurringIndicator, *response.RecurringIndicator, "RecurringIndicator should match")
	assert.NotNil(t, response.Frequency, "Frequency should not be nil")
	assert.Equal(t, frequency, *response.Frequency, "Frequency should match")

	// Verify consent purposes
	assert.NotNil(t, response.ConsentPurpose, "ConsentPurpose should not be nil")
	assert.Len(t, response.ConsentPurpose, 2, "Should have 2 consent purposes")
	for _, cp := range response.ConsentPurpose {
		assert.NotEmpty(t, cp.Name, "Purpose name should not be empty")
		assert.NotEmpty(t, cp.Value, "Purpose value should not be empty")
		assert.NotNil(t, cp.IsUserApproved, "IsUserApproved should not be nil")
		assert.True(t, *cp.IsUserApproved, "All purposes should be selected")
		assert.NotNil(t, cp.IsMandatory, "IsMandatory should not be nil")
		assert.True(t, *cp.IsMandatory, "All purposes should be mandatory")
	}

	t.Logf("✓ Created consent %s with dataAccessValidityDuration=%d", response.ID, *response.DataAccessValidityDuration)

	CleanupTestData(t, env, response.ID)
}

// TestCreateConsent_WithoutDataAccessValidityDuration tests that dataAccessValidityDuration can be omitted
func TestCreateConsent_WithoutDataAccessValidityDuration(t *testing.T) {
	env := SetupTestEnvironment(t)

	// Create test purpose
	purposes := CreateTestPurposes(t, env, map[string]string{
		"no_validity_test": "Test without data access validity duration",
	})
	defer CleanupTestPurposes(t, env, purposes)

	validityTime := int64(7776000)
	frequency := 1
	recurringIndicator := false

	createReq := &models.ConsentAPIRequest{
		Type:               "accounts",
		ValidityTime:       &validityTime,
		RecurringIndicator: &recurringIndicator,
		Frequency:          &frequency,
		ConsentPurpose: []models.ConsentPurposeItem{
			{Name: "no_validity_test", Value: "Test without dataAccessValidityDuration", IsUserApproved: BoolPtr(true), IsMandatory: BoolPtr(true)},
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

	assert.Equal(t, http.StatusCreated, recorder.Code, "Expected 201 Created status")

	var response models.ConsentAPIResponse
	err = json.Unmarshal(recorder.Body.Bytes(), &response)
	require.NoError(t, err)

	// Verify all response fields
	assert.NotEmpty(t, response.ID, "Consent ID should not be empty")
	assert.Equal(t, "accounts", response.Type, "Type should match")
	assert.Equal(t, "TEST_CLIENT", response.ClientID, "ClientID should match")
	assert.Equal(t, "CREATED", response.Status, "Status should be CREATED")

	// Verify timestamps
	assert.Greater(t, response.CreatedTime, int64(0), "CreatedTime should be positive")
	assert.Greater(t, response.UpdatedTime, int64(0), "UpdatedTime should be positive")

	// Verify dataAccessValidityDuration is null - the main focus of this test
	assert.Nil(t, response.DataAccessValidityDuration, "DataAccessValidityDuration should be null")

	// Verify other consent fields
	assert.NotNil(t, response.ValidityTime, "ValidityTime should not be nil")
	assert.Equal(t, validityTime, *response.ValidityTime, "ValidityTime should match")
	assert.NotNil(t, response.RecurringIndicator, "RecurringIndicator should not be nil")
	assert.Equal(t, recurringIndicator, *response.RecurringIndicator, "RecurringIndicator should match")
	assert.NotNil(t, response.Frequency, "Frequency should not be nil")
	assert.Equal(t, frequency, *response.Frequency, "Frequency should match")

	// Verify consent purpose
	assert.NotNil(t, response.ConsentPurpose, "ConsentPurpose should not be nil")
	assert.Len(t, response.ConsentPurpose, 1, "Should have 1 consent purpose")
	assert.Equal(t, "no_validity_test", response.ConsentPurpose[0].Name, "Purpose name should match")
	assert.NotNil(t, response.ConsentPurpose[0].IsUserApproved, "IsUserApproved should not be nil")
	assert.True(t, *response.ConsentPurpose[0].IsUserApproved, "Purpose should be selected")

	t.Logf("✓ Created consent %s without dataAccessValidityDuration (null)", response.ID)

	CleanupTestData(t, env, response.ID)
}

// TestCreateConsent_WithNegativeDataAccessValidityDuration tests that negative values are rejected
func TestCreateConsent_WithNegativeDataAccessValidityDuration(t *testing.T) {
	env := SetupTestEnvironment(t)

	// Create test purpose
	purposes := CreateTestPurposes(t, env, map[string]string{
		"neg_validity_test": "Test with negative data access validity duration",
	})
	defer CleanupTestPurposes(t, env, purposes)

	validityTime := int64(7776000)
	frequency := 1
	recurringIndicator := false
	negativeDataAccessValidityDuration := int64(-100)

	createReq := &models.ConsentAPIRequest{
		Type:                       "accounts",
		ValidityTime:               &validityTime,
		RecurringIndicator:         &recurringIndicator,
		Frequency:                  &frequency,
		DataAccessValidityDuration: &negativeDataAccessValidityDuration,
		ConsentPurpose: []models.ConsentPurposeItem{
			{Name: "neg_validity_test", Value: "Test with negative dataAccessValidityDuration", IsUserApproved: BoolPtr(true), IsMandatory: BoolPtr(true)},
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

	// Assert 400 Bad Request
	assert.Equal(t, http.StatusBadRequest, recorder.Code, "Expected 400 for negative dataAccessValidityDuration")

	var errorResponse map[string]interface{}
	err = json.Unmarshal(recorder.Body.Bytes(), &errorResponse)
	require.NoError(t, err)

	// Verify error mentions validation failure
	errorText := strings.ToLower(errorResponse["message"].(string))
	if details, ok := errorResponse["details"].(string); ok {
		errorText += " " + strings.ToLower(details)
	}
	assert.Contains(t, errorText, "dataaccessvalidityduration", "Error should mention dataAccessValidityDuration")

	t.Log("✓ Correctly rejected negative dataAccessValidityDuration")
}

// TestCreateConsent_DuplicatePurposeNames tests that consent creation rejects duplicate purpose names
func TestCreateConsent_DuplicatePurposeNames(t *testing.T) {
	env := SetupTestEnvironment(t)

	// Create test purposes
	purposes := CreateTestPurposes(t, env, map[string]string{
		"test_data_access":  "Test data access purpose",
		"test_account_info": "Test account info purpose",
	})
	defer CleanupTestPurposes(t, env, purposes)

	// Prepare request with duplicate purpose names
	validityTime := int64(7776000) // ~90 days in seconds

	createReq := &models.ConsentAPIRequest{
		Type:         "accounts",
		ValidityTime: &validityTime,
		ConsentPurpose: []models.ConsentPurposeItem{
			{Name: "test_data_access", Value: "Read account data", IsUserApproved: BoolPtr(true), IsMandatory: BoolPtr(true)},
			{Name: "test_data_access", Value: "Duplicate purpose", IsUserApproved: BoolPtr(true), IsMandatory: BoolPtr(true)}, // Duplicate!
			{Name: "test_account_info", Value: "Read account info", IsUserApproved: BoolPtr(false), IsMandatory: BoolPtr(false)},
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

	// Assert error response
	assert.Equal(t, http.StatusBadRequest, recorder.Code, "Expected 400 Bad Request for duplicate purpose names")

	var errorResponse map[string]interface{}
	err = json.Unmarshal(recorder.Body.Bytes(), &errorResponse)
	require.NoError(t, err)

	// Verify error mentions duplicate purpose
	errorText := strings.ToLower(errorResponse["message"].(string))
	if details, ok := errorResponse["details"].(string); ok {
		errorText += " " + strings.ToLower(details)
	}
	assert.Contains(t, errorText, "duplicate", "Error should mention duplicate")
	assert.Contains(t, errorText, "test_data_access", "Error should mention the duplicate purpose name")

	t.Log("✓ Correctly rejected duplicate purpose names in create request")
}

// TestCreateConsent_IsSelectedDefaultsToTrue tests valid combinations with explicit field values
func TestCreateConsent_IsSelectedDefaultsToTrue(t *testing.T) {
	env := SetupTestEnvironment(t)

	// Create test purposes
	purposes := CreateTestPurposes(t, env, map[string]string{
		"test_data_access":  "Test data access purpose",
		"test_account_info": "Test account info purpose",
	})
	defer CleanupTestPurposes(t, env, purposes)

	// Prepare request with explicit values that satisfy validation
	validityTime := int64(7776000) // ~90 days in seconds

	createReq := &models.ConsentAPIRequest{
		Type:         "accounts",
		ValidityTime: &validityTime,
		ConsentPurpose: []models.ConsentPurposeItem{
			// Provide isUserApproved=true to satisfy validation when isMandatory defaults to true
			{Name: "test_data_access", Value: "Read account data", IsUserApproved: BoolPtr(true)},
			{Name: "test_account_info", Value: "Read account info", IsUserApproved: BoolPtr(false), IsMandatory: BoolPtr(false)},
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

	// Verify consent purposes
	require.Len(t, response.ConsentPurpose, 2, "Should have 2 consent purposes")

	for _, cp := range response.ConsentPurpose {
		if cp.Name == "test_data_access" {
			require.NotNil(t, cp.IsUserApproved, "IsUserApproved should not be nil")
			assert.True(t, *cp.IsUserApproved, "IsUserApproved should be true")
			require.NotNil(t, cp.IsMandatory, "IsMandatory should not be nil")
			assert.True(t, *cp.IsMandatory, "IsMandatory should default to true")
		} else if cp.Name == "test_account_info" {
			require.NotNil(t, cp.IsUserApproved, "IsUserApproved should not be nil")
			assert.False(t, *cp.IsUserApproved, "IsUserApproved should be false when explicitly set")
			require.NotNil(t, cp.IsMandatory, "IsMandatory should not be nil")
			assert.False(t, *cp.IsMandatory, "IsMandatory should be false when explicitly set")
		}
	}

	t.Log("✓ Successfully created consent with valid field combinations")
	CleanupTestData(t, env, response.ID)
}
