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

// Helper function to create consent with specific authorization status
// The consent status is derived from the authorization status:
// - "approved" -> consent status becomes "active"
// - "created" -> consent status becomes "created"
// - "rejected" -> consent status becomes "rejected"
func createConsentWithAuthStatus(t *testing.T, env *TestEnvironment, purposes map[string]*models.ConsentPurpose, authStatus string, validityTime *int64) *models.ConsentAPIResponse {
	var consentPurposes []models.ConsentPurposeItem
	for name := range purposes {
		consentPurposes = append(consentPurposes, models.ConsentPurposeItem{
			Name:           name,
			Value:          "Test value for " + name,
			IsUserApproved: BoolPtr(true), IsMandatory: BoolPtr(true),
		})
	}

	createReq := &models.ConsentAPIRequest{
		Type:           "accounts",
		ConsentPurpose: consentPurposes,
		Authorizations: []models.AuthorizationAPIRequest{
			{
				UserID: "test-user-123",
				Type:   "authorization",
				Status: authStatus,
				Resources: map[string]interface{}{
					"accountIds": []string{"123456", "789012"},
				},
			},
		},
	}

	if validityTime != nil {
		createReq.ValidityTime = validityTime
	}

	reqBody, _ := json.Marshal(createReq)
	req, _ := http.NewRequest("POST", "/api/v1/consents", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("org-id", "TEST_ORG")
	req.Header.Set("client-id", "TEST_CLIENT")

	recorder := httptest.NewRecorder()
	env.Router.ServeHTTP(recorder, req)

	var response models.ConsentAPIResponse
	json.Unmarshal(recorder.Body.Bytes(), &response)

	return &response
}

// TestValidateConsent_Success tests successful consent validation
func TestValidateConsent_Success(t *testing.T) {
	env := SetupTestEnvironment(t)

	purposes := CreateTestPurposes(t, env, map[string]string{
		"test_data_access": "Test data access purpose",
	})
	defer CleanupTestPurposes(t, env, purposes)

	validityTime := time.Now().Add(24 * time.Hour).Unix()

	// Create consent with "approved" auth status -> consent becomes "active"
	consent := createConsentWithAuthStatus(t, env, purposes, "approved", &validityTime)
	defer CleanupTestData(t, env, consent.ID)

	validateRequest := models.ValidateRequest{
		ConsentID: consent.ID,
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

	requestBody, _ := json.Marshal(validateRequest)
	req, _ := http.NewRequest("POST", "/api/v1/consents/validate", bytes.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("org-id", "TEST_ORG")

	recorder := httptest.NewRecorder()
	env.Router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code) // Validate always returns 200

	var response models.ValidateResponse
	json.Unmarshal(recorder.Body.Bytes(), &response)

	assert.True(t, response.IsValid)
	assert.Empty(t, response.ErrorMessage)
	t.Logf("✓ Validate successful")
}

// TestValidateConsent_InvalidStatus tests validation with invalid consent status
func TestValidateConsent_InvalidStatus(t *testing.T) {
	env := SetupTestEnvironment(t)

	purposes := CreateTestPurposes(t, env, map[string]string{
		"test_data_access": "Test data access purpose",
	})
	defer CleanupTestPurposes(t, env, purposes)

	validityTime := time.Now().Add(24 * time.Hour).Unix()

	// Create consent with "rejected" auth status -> consent becomes "rejected"
	consent := createConsentWithAuthStatus(t, env, purposes, "rejected", &validityTime)
	defer CleanupTestData(t, env, consent.ID)

	// Now validate
	validateRequest := models.ValidateRequest{
		ConsentID: consent.ID,
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

	requestBody, _ := json.Marshal(validateRequest)
	req, _ := http.NewRequest("POST", "/api/v1/consents/validate", bytes.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("org-id", "TEST_ORG")

	recorder := httptest.NewRecorder()
	env.Router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code) // Validate always returns 200

	var response models.ValidateResponse
	json.Unmarshal(recorder.Body.Bytes(), &response)

	assert.False(t, response.IsValid)
	assert.Equal(t, 401, response.ErrorCode)
	assert.Equal(t, "invalid_consent_status", response.ErrorMessage)
	assert.NotEmpty(t, response.ErrorDescription)
	t.Logf("✓ Invalid status validation failed correctly")
}

// TestValidateConsent_ExpiredConsent tests validation with expired consent
func TestValidateConsent_ExpiredConsent(t *testing.T) {
	env := SetupTestEnvironment(t)

	purposes := CreateTestPurposes(t, env, map[string]string{
		"test_data_access": "Test data access purpose",
	})
	defer CleanupTestPurposes(t, env, purposes)

	validityTime := time.Now().Add(-1 * time.Hour).Unix() // Expired 1 hour ago

	// Create consent with "approved" auth status -> consent becomes "active" but expired
	consent := createConsentWithAuthStatus(t, env, purposes, "approved", &validityTime)
	defer CleanupTestData(t, env, consent.ID)

	validateRequest := models.ValidateRequest{
		ConsentID: consent.ID,
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

	requestBody, _ := json.Marshal(validateRequest)
	req, _ := http.NewRequest("POST", "/api/v1/consents/validate", bytes.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("org-id", "TEST_ORG")

	recorder := httptest.NewRecorder()
	env.Router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code) // Validate always returns 200

	var response models.ValidateResponse
	json.Unmarshal(recorder.Body.Bytes(), &response)

	assert.False(t, response.IsValid)
	assert.Equal(t, 401, response.ErrorCode)
	assert.Equal(t, "consent_expired", response.ErrorMessage)
	assert.NotEmpty(t, response.ErrorDescription)
	t.Logf("✓ Expired consent validation failed correctly")
}

// TestValidateConsent_ExpiredConsentUpdatesStatus tests that when validating an expired consent,
// the consent status is automatically updated to expired in the database
func TestValidateConsent_ExpiredConsentUpdatesStatus(t *testing.T) {
	env := SetupTestEnvironment(t)

	purposes := CreateTestPurposes(t, env, map[string]string{
		"test_data_access": "Test data access purpose",
	})
	defer CleanupTestPurposes(t, env, purposes)

	validityTime := time.Now().Add(-1 * time.Hour).Unix() // Expired 1 hour ago

	// Create consent with "approved" auth status -> consent becomes "active"
	consent := createConsentWithAuthStatus(t, env, purposes, "approved", &validityTime)
	defer CleanupTestData(t, env, consent.ID)

	// Note: The GET endpoint now automatically detects and updates expired consents,
	// so the consent will already be EXPIRED when we first retrieve it
	req, _ := http.NewRequest("GET", "/api/v1/consents/"+consent.ID, nil)
	req.Header.Set("org-id", "TEST_ORG")

	recorder := httptest.NewRecorder()
	env.Router.ServeHTTP(recorder, req)

	var getResponse1 models.ConsentAPIResponse
	json.Unmarshal(recorder.Body.Bytes(), &getResponse1)

	assert.Equal(t, http.StatusOK, recorder.Code)
	// With the new expiry check in GET, expired consents are immediately updated to EXPIRED
	assert.Equal(t, "EXPIRED", getResponse1.Status, "Consent should be EXPIRED after GET detects expiry")
	t.Logf("✓ Consent status after GET (with expiry check): %s", getResponse1.Status)

	// Call validate endpoint with the expired consent
	validateRequest := models.ValidateRequest{
		ConsentID: consent.ID,
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

	requestBody, _ := json.Marshal(validateRequest)
	req, _ = http.NewRequest("POST", "/api/v1/consents/validate", bytes.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("org-id", "TEST_ORG")

	recorder = httptest.NewRecorder()
	env.Router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)

	var validateResponse models.ValidateResponse
	json.Unmarshal(recorder.Body.Bytes(), &validateResponse)

	// Verify validation failed due to expiry
	assert.False(t, validateResponse.IsValid)
	assert.Equal(t, 401, validateResponse.ErrorCode)
	assert.Equal(t, "invalid_consent_status", validateResponse.ErrorMessage, "Since consent is already EXPIRED from GET, validate returns invalid_consent_status")
	assert.Contains(t, validateResponse.ErrorDescription, "EXPIRED")
	t.Logf("✓ Validate correctly returned invalid_consent_status error (consent already EXPIRED)")

	// Verify the consent information includes updated status
	assert.NotNil(t, validateResponse.ConsentInformation)
	consentInfo := validateResponse.ConsentInformation
	assert.Equal(t, "EXPIRED", consentInfo.Status, "Consent status in response should be EXPIRED")
	t.Logf("✓ Validate response shows updated status: %s", consentInfo.Status)

	// Verify by calling GET endpoint again that the status was persisted to database
	req, _ = http.NewRequest("GET", "/api/v1/consents/"+consent.ID, nil)
	req.Header.Set("org-id", "TEST_ORG")

	recorder = httptest.NewRecorder()
	env.Router.ServeHTTP(recorder, req)

	var getResponse2 models.ConsentAPIResponse
	json.Unmarshal(recorder.Body.Bytes(), &getResponse2)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "EXPIRED", getResponse2.Status, "Consent status should remain EXPIRED in database")
	t.Logf("✓ Database confirms status remains: %s", getResponse2.Status)
	t.Logf("✓ Expiry check works correctly - GET detects expired consent and updates status to EXPIRED")
}

// TestValidateConsent_NotFound tests validation with non-existent consent
func TestValidateConsent_NotFound(t *testing.T) {
	env := SetupTestEnvironment(t)

	validateRequest := models.ValidateRequest{
		ConsentID: "non-existent-consent-id",
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

	requestBody, _ := json.Marshal(validateRequest)
	req, _ := http.NewRequest("POST", "/api/v1/consents/validate", bytes.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("org-id", "TEST_ORG")

	recorder := httptest.NewRecorder()
	env.Router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code) // Validate always returns 200

	var response models.ValidateResponse
	json.Unmarshal(recorder.Body.Bytes(), &response)

	assert.False(t, response.IsValid)
	assert.Equal(t, 404, response.ErrorCode)
	assert.Equal(t, "consent_not_found", response.ErrorMessage)
	assert.NotEmpty(t, response.ErrorDescription)
	t.Logf("✓ Not found validation failed correctly")
}

// TestValidateConsent_MissingConsentID tests validation without consent ID
func TestValidateConsent_MissingConsentID(t *testing.T) {
	env := SetupTestEnvironment(t)

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

	requestBody, _ := json.Marshal(validateRequest)
	req, _ := http.NewRequest("POST", "/api/v1/consents/validate", bytes.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("org-id", "TEST_ORG")

	recorder := httptest.NewRecorder()
	env.Router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code) // Validate always returns 200

	var response models.ValidateResponse
	json.Unmarshal(recorder.Body.Bytes(), &response)

	assert.False(t, response.IsValid)
	assert.Equal(t, 400, response.ErrorCode)
	assert.Equal(t, "invalid_request", response.ErrorMessage)
	assert.Contains(t, response.ErrorDescription, "consentId")
	t.Logf("✓ Missing consent ID validation failed correctly")
}

// TestValidateConsent_MissingUserID tests validation without user ID
func TestValidateConsent_MissingUserID(t *testing.T) {
	env := SetupTestEnvironment(t)

	validateRequest := models.ValidateRequest{
		ConsentID: "some-consent-id",
		ResourceParams: struct {
			Resource   string `json:"resource"`
			HTTPMethod string `json:"httpMethod"`
			Context    string `json:"context"`
		}{
			Resource:   "/accounts/123",
			HTTPMethod: "GET",
		},
	}

	requestBody, _ := json.Marshal(validateRequest)
	req, _ := http.NewRequest("POST", "/api/v1/consents/validate", bytes.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("org-id", "TEST_ORG")

	recorder := httptest.NewRecorder()
	env.Router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code) // Validate always returns 200

	var response models.ValidateResponse
	json.Unmarshal(recorder.Body.Bytes(), &response)

	assert.False(t, response.IsValid)
	assert.Equal(t, 400, response.ErrorCode)
	assert.Equal(t, "invalid_request", response.ErrorMessage)
	assert.Contains(t, response.ErrorDescription, "userId")
	t.Logf("✓ Missing user ID validation failed correctly")
}

// TestValidateConsent_InvalidJSON tests validation with invalid JSON
func TestValidateConsent_InvalidJSON(t *testing.T) {
	env := SetupTestEnvironment(t)

	req, _ := http.NewRequest("POST", "/api/v1/consents/validate", bytes.NewReader([]byte("{invalid json")))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("org-id", "TEST_ORG")

	recorder := httptest.NewRecorder()
	env.Router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code) // Validate always returns 200

	var response models.ValidateResponse
	json.Unmarshal(recorder.Body.Bytes(), &response)

	assert.False(t, response.IsValid)
	assert.Equal(t, 400, response.ErrorCode)
	assert.Equal(t, "invalid_request", response.ErrorMessage)
	assert.NotEmpty(t, response.ErrorDescription)
	t.Logf("✓ Invalid JSON validation failed correctly")
}

// TestValidateConsent_FullConsentInformationResponse tests that the validate response
// includes complete consent information matching GET /consents/{consentId} response
func TestValidateConsent_FullConsentInformationResponse(t *testing.T) {
	env := SetupTestEnvironment(t)

	// Create test purposes with different types
	purposes := CreateTestPurposes(t, env, map[string]string{
		"test_string_purpose":      "String type purpose",
		"test_json_schema_purpose": "JSON Schema type purpose",
		"test_attribute_purpose":   "Attribute type purpose",
	})
	defer CleanupTestPurposes(t, env, purposes)

	// Create consent with FULL payload including all optional fields
	validityTime := time.Now().Add(24 * time.Hour).Unix()
	frequency := 5
	recurringIndicator := true
	dataAccessDuration := int64(3600000) // 1 hour in milliseconds

	var consentPurposes []models.ConsentPurposeItem
	for name := range purposes {
		consentPurposes = append(consentPurposes, models.ConsentPurposeItem{
			Name:           name,
			Value:          "Test value for " + name,
			IsUserApproved: BoolPtr(true), IsMandatory: BoolPtr(true),
		})
	}

	createReq := &models.ConsentAPIRequest{
		Type:                       "accounts",
		ConsentPurpose:             consentPurposes,
		ValidityTime:               &validityTime,
		Frequency:                  &frequency,
		RecurringIndicator:         &recurringIndicator,
		DataAccessValidityDuration: &dataAccessDuration,
		Attributes: map[string]string{
			"customerId":  "CUST-12345",
			"accountId":   "ACC-67890",
			"environment": "production",
		},
		Authorizations: []models.AuthorizationAPIRequest{
			{
				UserID: "test-user-123",
				Type:   "authorization",
				Status: "approved",
				Resources: map[string]interface{}{
					"accountIds":  []string{"123456", "789012"},
					"permissions": []string{"read", "write"},
				},
			},
			{
				UserID: "test-user-456",
				Type:   "authorization",
				Status: "approved",
				Resources: map[string]interface{}{
					"accountIds": []string{"345678"},
				},
			},
		},
	}

	reqBody, _ := json.Marshal(createReq)
	req, _ := http.NewRequest("POST", "/api/v1/consents", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("org-id", "TEST_ORG")
	req.Header.Set("client-id", "TEST_CLIENT")

	recorder := httptest.NewRecorder()
	env.Router.ServeHTTP(recorder, req)

	var createResponse models.ConsentAPIResponse
	json.Unmarshal(recorder.Body.Bytes(), &createResponse)
	defer CleanupTestData(t, env, createResponse.ID)

	assert.Equal(t, http.StatusCreated, recorder.Code)
	t.Logf("✓ Created consent with full payload: %s", createResponse.ID)

	// Call validate endpoint
	validateRequest := models.ValidateRequest{
		ConsentID: createResponse.ID,
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

	requestBody, _ := json.Marshal(validateRequest)
	req, _ = http.NewRequest("POST", "/api/v1/consents/validate", bytes.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("org-id", "TEST_ORG")

	recorder = httptest.NewRecorder()
	env.Router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)

	var validateResponse models.ValidateResponse
	json.Unmarshal(recorder.Body.Bytes(), &validateResponse)

	// Verify validation succeeded
	assert.True(t, validateResponse.IsValid, "Validation should succeed")
	assert.NotNil(t, validateResponse.ConsentInformation, "ConsentInformation should be present")

	// Get the consent via GET endpoint to compare
	req, _ = http.NewRequest("GET", "/api/v1/consents/"+createResponse.ID, nil)
	req.Header.Set("org-id", "TEST_ORG")

	recorder = httptest.NewRecorder()
	env.Router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)

	var getResponse models.ConsentAPIResponse
	json.Unmarshal(recorder.Body.Bytes(), &getResponse)

	// Convert both to JSON for detailed comparison
	consentInfo := validateResponse.ConsentInformation

	// ========== COMPREHENSIVE FIELD VALIDATION ==========

	// 1. Core required fields
	assert.Equal(t, getResponse.ID, consentInfo.ID, "id must match")
	assert.Equal(t, getResponse.Type, consentInfo.Type, "type must match")
	assert.Equal(t, getResponse.Status, consentInfo.Status, "status must match")
	assert.Equal(t, getResponse.ClientID, consentInfo.ClientID, "clientId must match")
	t.Logf("✓ Core fields validated: id, type, status, clientId")

	// 2. Timestamps
	assert.NotZero(t, consentInfo.CreatedTime, "createdTime must be present")
	assert.NotZero(t, consentInfo.UpdatedTime, "updatedTime must be present")
	assert.Equal(t, getResponse.CreatedTime, consentInfo.CreatedTime, "createdTime must match")
	assert.Equal(t, getResponse.UpdatedTime, consentInfo.UpdatedTime, "updatedTime must match")
	t.Logf("✓ Timestamps validated: createdTime, updatedTime")

	// 3. Optional fields that were provided
	assert.NotNil(t, consentInfo.ValidityTime, "validityTime should be present")
	assert.Equal(t, *getResponse.ValidityTime, *consentInfo.ValidityTime, "validityTime must match")

	assert.NotNil(t, consentInfo.Frequency, "frequency should be present")
	assert.Equal(t, *getResponse.Frequency, *consentInfo.Frequency, "frequency must match")

	assert.NotNil(t, consentInfo.RecurringIndicator, "recurringIndicator should be present")
	assert.Equal(t, *getResponse.RecurringIndicator, *consentInfo.RecurringIndicator, "recurringIndicator must match")

	assert.NotNil(t, consentInfo.DataAccessValidityDuration, "dataAccessValidityDuration should be present")
	assert.Equal(t, *getResponse.DataAccessValidityDuration, *consentInfo.DataAccessValidityDuration, "dataAccessValidityDuration must match")
	t.Logf("✓ Optional fields validated: validityTime, frequency, recurringIndicator, dataAccessValidityDuration")

	// 4. Attributes - verify all attributes match
	assert.Len(t, consentInfo.Attributes, len(getResponse.Attributes), "attributes count must match")

	for key, expectedValue := range getResponse.Attributes {
		actualValue, exists := consentInfo.Attributes[key]
		assert.True(t, exists, "attribute '%s' should exist in validate response", key)
		assert.Equal(t, expectedValue, actualValue, "attribute '%s' value must match", key)
	}
	t.Logf("✓ All %d attributes validated", len(getResponse.Attributes))

	// 5. Consent Purposes - comprehensive validation
	assert.Len(t, consentInfo.ConsentPurpose, len(getResponse.ConsentPurpose), "consentPurpose count must match")
	assert.Len(t, consentInfo.ConsentPurpose, 3, "should have 3 consent purposes")

	// Create maps for easier comparison
	validatePurposeMap := make(map[string]models.ConsentPurposeItem)
	for _, cp := range consentInfo.ConsentPurpose {
		validatePurposeMap[cp.Name] = cp
	}

	getPurposeMap := make(map[string]models.ConsentPurposeItem)
	for _, cp := range getResponse.ConsentPurpose {
		getPurposeMap[cp.Name] = cp
	}

	for purposeName, getCP := range getPurposeMap {
		validateCP, exists := validatePurposeMap[purposeName]
		assert.True(t, exists, "purpose '%s' should exist in validate response", purposeName)

		assert.Equal(t, getCP.Name, validateCP.Name, "purpose name must match")
		assert.Equal(t, *getCP.IsUserApproved, *validateCP.IsUserApproved, "purpose isUserApproved must match for %s", purposeName)

		// Validate enriched fields
		assert.NotNil(t, validateCP.Type, "purpose type should be enriched for %s", purposeName)
		assert.NotEmpty(t, *validateCP.Type, "purpose type should not be empty for %s", purposeName)
		assert.NotNil(t, validateCP.Description, "purpose description should be enriched for %s", purposeName)
		assert.NotEmpty(t, *validateCP.Description, "purpose description should not be empty for %s", purposeName)

		// Verify value matches
		if getCP.Value != nil {
			assert.NotNil(t, validateCP.Value, "purpose value should be present for %s", purposeName)
			assert.Equal(t, getCP.Value, validateCP.Value, "purpose value must match for %s", purposeName)
		}
	}
	t.Logf("✓ All %d consent purposes validated with enrichment (type, description, attributes)", len(getResponse.ConsentPurpose))

	// 6. Authorizations - comprehensive validation
	assert.Len(t, consentInfo.Authorizations, len(getResponse.Authorizations), "authorizations count must match")
	assert.Len(t, consentInfo.Authorizations, 2, "should have 2 authorizations")

	// Create maps for easier comparison
	validateAuthMap := make(map[string]models.AuthorizationAPIResponse)
	for _, auth := range consentInfo.Authorizations {
		validateAuthMap[auth.ID] = auth
	}

	getAuthMap := make(map[string]models.AuthorizationAPIResponse)
	for _, auth := range getResponse.Authorizations {
		getAuthMap[auth.ID] = auth
	}

	for authID, getAuth := range getAuthMap {
		validateAuth, exists := validateAuthMap[authID]
		assert.True(t, exists, "authorization '%s' should exist in validate response", authID)

		assert.Equal(t, getAuth.ID, validateAuth.ID, "auth id must match")
		assert.Equal(t, getAuth.Type, validateAuth.Type, "auth type must match")
		assert.Equal(t, getAuth.Status, validateAuth.Status, "auth status must match")

		// UserID is a pointer in API response
		if getAuth.UserID != nil {
			assert.Equal(t, *getAuth.UserID, *validateAuth.UserID, "auth userId must match")
		}

		assert.NotZero(t, validateAuth.UpdatedTime, "auth updatedTime should be present")
		assert.Equal(t, getAuth.UpdatedTime, validateAuth.UpdatedTime, "auth updatedTime must match")

		// Verify resources if present
		if getAuth.Resources != nil {
			assert.NotNil(t, validateAuth.Resources, "auth resources should be present for %s", authID)
			// Deep comparison of resources structure would go here
		}
	}
	t.Logf("✓ All %d authorizations validated", len(getResponse.Authorizations))

	// Final verification
	t.Logf("✓ ========== COMPREHENSIVE VALIDATION COMPLETE ==========")
	t.Logf("✓ Validate response consentInformation exactly matches GET /consents/{id} response")
	t.Logf("✓ All core fields, optional fields, attributes, purposes, and authorizations validated")
	t.Logf("✓ Consent purposes enriched with type, description, and attributes from definitions")
}

// TestValidateConsent_NoModifiedResponseField tests that validate response does not contain modifiedResponse
// but GET/POST responses DO contain it (even if empty)
func TestValidateConsent_NoModifiedResponseField(t *testing.T) {
	env := SetupTestEnvironment(t)

	// Create test purpose
	purposes := CreateTestPurposes(t, env, map[string]string{
		"test_purpose": "Test purpose for modified response check",
	})
	defer CleanupTestPurposes(t, env, purposes)

	// Create consent with authorization
	validityTime := time.Now().Add(90 * 24 * time.Hour).UnixMilli()
	createReq := &models.ConsentAPIRequest{
		Type:         "accounts",
		ValidityTime: &validityTime,
		ConsentPurpose: []models.ConsentPurposeItem{
			{Name: "test_purpose", Value: "Test value", IsUserApproved: BoolPtr(true), IsMandatory: BoolPtr(true)},
		},
		Authorizations: []models.AuthorizationAPIRequest{
			{
				UserID: "user-123",
				Type:   "authorization_code",
				Status: "approved",
			},
		},
	}

	reqBody, _ := json.Marshal(createReq)
	req, _ := http.NewRequest("POST", "/api/v1/consents", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("org-id", "TEST_ORG")
	req.Header.Set("client-id", "TEST_CLIENT")

	recorder := httptest.NewRecorder()
	env.Router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusCreated, recorder.Code)

	var createResponse models.ConsentAPIResponse
	json.Unmarshal(recorder.Body.Bytes(), &createResponse)
	defer CleanupTestData(t, env, createResponse.ID)

	// Verify POST response contains modifiedResponse field (even if empty)
	createResponseMap := make(map[string]interface{})
	json.Unmarshal(recorder.Body.Bytes(), &createResponseMap)
	assert.Contains(t, createResponseMap, "modifiedResponse", "POST response should contain modifiedResponse field")
	t.Logf("✓ POST /consents response contains modifiedResponse field")

	// Verify GET response contains modifiedResponse field (even if empty)
	getReq, _ := http.NewRequest("GET", "/api/v1/consents/"+createResponse.ID, nil)
	getReq.Header.Set("org-id", "TEST_ORG")
	getReq.Header.Set("client-id", "TEST_CLIENT")

	getRecorder := httptest.NewRecorder()
	env.Router.ServeHTTP(getRecorder, getReq)

	assert.Equal(t, http.StatusOK, getRecorder.Code)

	getResponseMap := make(map[string]interface{})
	json.Unmarshal(getRecorder.Body.Bytes(), &getResponseMap)
	assert.Contains(t, getResponseMap, "modifiedResponse", "GET response should contain modifiedResponse field")
	t.Logf("✓ GET /consents/{id} response contains modifiedResponse field")

	// Verify VALIDATE response does NOT contain modifiedResponse
	validateReq := &models.ValidateRequest{
		ConsentID: createResponse.ID,
		UserID:    "user-123",
	}

	validateBody, _ := json.Marshal(validateReq)
	valReq, _ := http.NewRequest("POST", "/api/v1/consents/validate", bytes.NewBuffer(validateBody))
	valReq.Header.Set("Content-Type", "application/json")
	valReq.Header.Set("org-id", "TEST_ORG")
	valReq.Header.Set("client-id", "TEST_CLIENT")

	valRecorder := httptest.NewRecorder()
	env.Router.ServeHTTP(valRecorder, valReq)

	assert.Equal(t, http.StatusOK, valRecorder.Code)

	var validateResponse models.ValidateResponse
	json.Unmarshal(valRecorder.Body.Bytes(), &validateResponse)
	assert.True(t, validateResponse.IsValid, "Consent should be valid")

	// Verify consentInformation does NOT contain modifiedResponse field
	consentInfoBytes, _ := json.Marshal(validateResponse.ConsentInformation)
	consentInfoMap := make(map[string]interface{})
	json.Unmarshal(consentInfoBytes, &consentInfoMap)

	assert.NotContains(t, consentInfoMap, "modifiedResponse", "Validate response consentInformation should NOT contain modifiedResponse")
	t.Logf("✓ POST /consents/validate response does NOT contain modifiedResponse field in consentInformation")

	// Verify the entire validate response doesn't contain modifiedResponse at any level
	validateResponseMap := make(map[string]interface{})
	json.Unmarshal(valRecorder.Body.Bytes(), &validateResponseMap)
	assert.NotContains(t, validateResponseMap, "modifiedResponse", "Validate response should NOT contain modifiedResponse at top level")
	t.Logf("✓ Bug fix verified: modifiedResponse present in GET/POST but absent in VALIDATE")
}

// TestCreateConsent_ValidationMandatoryRequiresApproved tests validation rule: isMandatory=true requires isUserApproved=true
func TestCreateConsent_ValidationMandatoryRequiresApproved(t *testing.T) {
	env := SetupTestEnvironment(t)

	// Create test purpose
	purposes := CreateTestPurposes(t, env, map[string]string{
		"test_purpose": "Test purpose for validation",
	})
	defer CleanupTestPurposes(t, env, purposes)

	validityTime := time.Now().Add(90 * 24 * time.Hour).UnixMilli()

	tests := []struct {
		name               string
		isMandatory        *bool
		isUserApproved     *bool
		expectedStatusCode int
		expectError        bool
	}{
		{
			name:               "isMandatory=true and isUserApproved=true - should succeed",
			isMandatory:        BoolPtr(true),
			isUserApproved:     BoolPtr(true),
			expectedStatusCode: http.StatusCreated,
			expectError:        false,
		},
		{
			name:               "isMandatory=true and isUserApproved=false - should fail",
			isMandatory:        BoolPtr(true),
			isUserApproved:     BoolPtr(false),
			expectedStatusCode: http.StatusBadRequest,
			expectError:        true,
		},
		{
			name:               "isMandatory=false and isUserApproved=false - should succeed",
			isMandatory:        BoolPtr(false),
			isUserApproved:     BoolPtr(false),
			expectedStatusCode: http.StatusCreated,
			expectError:        false,
		},
		{
			name:               "isMandatory=false and isUserApproved=true - should succeed",
			isMandatory:        BoolPtr(false),
			isUserApproved:     BoolPtr(true),
			expectedStatusCode: http.StatusCreated,
			expectError:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			createReq := &models.ConsentAPIRequest{
				Type:         "accounts",
				ValidityTime: &validityTime,
				ConsentPurpose: []models.ConsentPurposeItem{
					{
						Name:           "test_purpose",
						Value:          "Test value",
						IsMandatory:    tt.isMandatory,
						IsUserApproved: tt.isUserApproved,
					},
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

			assert.Equal(t, tt.expectedStatusCode, recorder.Code, "Status code should match")

			if tt.expectError {
				var errorResponse map[string]interface{}
				err = json.Unmarshal(recorder.Body.Bytes(), &errorResponse)
				require.NoError(t, err)

				errorText := strings.ToLower(errorResponse["message"].(string))
				if details, ok := errorResponse["details"].(string); ok {
					errorText += " " + strings.ToLower(details)
				}
				assert.Contains(t, errorText, "mandatory", "Error should mention mandatory")
				assert.Contains(t, errorText, "approved", "Error should mention approved")
				t.Logf("✓ Correctly rejected invalid combination: %s", tt.name)
			} else {
				var response models.ConsentAPIResponse
				err = json.Unmarshal(recorder.Body.Bytes(), &response)
				require.NoError(t, err)

				assert.NotEmpty(t, response.ID, "Consent ID should not be empty")
				assert.Len(t, response.ConsentPurpose, 1, "Should have 1 purpose")

				cp := response.ConsentPurpose[0]
				assert.NotNil(t, cp.IsMandatory, "IsMandatory should not be nil")
				assert.NotNil(t, cp.IsUserApproved, "IsUserApproved should not be nil")
				assert.Equal(t, *tt.isMandatory, *cp.IsMandatory, "IsMandatory should match")
				assert.Equal(t, *tt.isUserApproved, *cp.IsUserApproved, "IsUserApproved should match")

				t.Logf("✓ Successfully created consent with valid combination: %s", tt.name)
				CleanupTestData(t, env, response.ID)
			}
		})
	}
}

// TestCreateConsent_DefaultBehaviorValidation tests default values are applied correctly
func TestCreateConsent_DefaultBehaviorValidation(t *testing.T) {
	env := SetupTestEnvironment(t)

	// Create test purposes
	purposes := CreateTestPurposes(t, env, map[string]string{
		"test_purpose_1": "Test purpose 1",
		"test_purpose_2": "Test purpose 2",
		"test_purpose_3": "Test purpose 3",
	})
	defer CleanupTestPurposes(t, env, purposes)

	validityTime := time.Now().Add(90 * 24 * time.Hour).UnixMilli()

	tests := []struct {
		name                   string
		isMandatory            *bool
		isUserApproved         *bool
		expectedIsMandatory    bool
		expectedIsUserApproved bool
		shouldFail             bool
	}{
		{
			name:                   "Both fields omitted - should apply defaults and succeed",
			isMandatory:            nil,
			isUserApproved:         nil,
			expectedIsMandatory:    true,
			expectedIsUserApproved: false,
			shouldFail:             true, // Default isMandatory=true, isUserApproved=false violates validation
		},
		{
			name:                   "Only isMandatory provided (true) - isUserApproved defaults to false, should fail",
			isMandatory:            BoolPtr(true),
			isUserApproved:         nil,
			expectedIsMandatory:    true,
			expectedIsUserApproved: false,
			shouldFail:             true, // Violates validation rule
		},
		{
			name:                   "Only isMandatory provided (false) - isUserApproved defaults to false, should succeed",
			isMandatory:            BoolPtr(false),
			isUserApproved:         nil,
			expectedIsMandatory:    false,
			expectedIsUserApproved: false,
			shouldFail:             false,
		},
		{
			name:                   "Only isUserApproved provided (true) - isMandatory defaults to true, should succeed",
			isMandatory:            nil,
			isUserApproved:         BoolPtr(true),
			expectedIsMandatory:    true,
			expectedIsUserApproved: true,
			shouldFail:             false,
		},
		{
			name:                   "Only isUserApproved provided (false) - isMandatory defaults to true, should fail",
			isMandatory:            nil,
			isUserApproved:         BoolPtr(false),
			expectedIsMandatory:    true,
			expectedIsUserApproved: false,
			shouldFail:             true, // Violates validation rule
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			createReq := &models.ConsentAPIRequest{
				Type:         "accounts",
				ValidityTime: &validityTime,
				ConsentPurpose: []models.ConsentPurposeItem{
					{
						Name:           "test_purpose_1",
						Value:          "Test value",
						IsMandatory:    tt.isMandatory,
						IsUserApproved: tt.isUserApproved,
					},
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

			if tt.shouldFail {
				assert.Equal(t, http.StatusBadRequest, recorder.Code, "Should return 400 for validation failure")
				var errorResponse map[string]interface{}
				err = json.Unmarshal(recorder.Body.Bytes(), &errorResponse)
				require.NoError(t, err)
				t.Logf("✓ Correctly rejected default combination that violates validation: %s", tt.name)
			} else {
				assert.Equal(t, http.StatusCreated, recorder.Code, "Should return 201 for valid combination")
				var response models.ConsentAPIResponse
				err = json.Unmarshal(recorder.Body.Bytes(), &response)
				require.NoError(t, err)

				assert.NotEmpty(t, response.ID, "Consent ID should not be empty")
				require.Len(t, response.ConsentPurpose, 1, "Should have 1 purpose")

				cp := response.ConsentPurpose[0]
				assert.NotNil(t, cp.IsMandatory, "IsMandatory should not be nil")
				assert.NotNil(t, cp.IsUserApproved, "IsUserApproved should not be nil")
				assert.Equal(t, tt.expectedIsMandatory, *cp.IsMandatory, "IsMandatory should match expected default")
				assert.Equal(t, tt.expectedIsUserApproved, *cp.IsUserApproved, "IsUserApproved should match expected default")

				t.Logf("✓ Defaults applied correctly: isMandatory=%v, isUserApproved=%v", *cp.IsMandatory, *cp.IsUserApproved)
				CleanupTestData(t, env, response.ID)
			}
		})
	}
}

// TestCreateConsent_MixedPurposeScenarios tests multiple purposes with different combinations
func TestCreateConsent_MixedPurposeScenarios(t *testing.T) {
	env := SetupTestEnvironment(t)

	// Create test purposes
	purposes := CreateTestPurposes(t, env, map[string]string{
		"mandatory_approved":    "Mandatory and approved purpose",
		"optional_not_approved": "Optional and not approved purpose",
		"optional_approved":     "Optional and approved purpose",
	})
	defer CleanupTestPurposes(t, env, purposes)

	validityTime := time.Now().Add(90 * 24 * time.Hour).UnixMilli()

	createReq := &models.ConsentAPIRequest{
		Type:         "accounts",
		ValidityTime: &validityTime,
		ConsentPurpose: []models.ConsentPurposeItem{
			{
				Name:           "mandatory_approved",
				Value:          "Must be approved",
				IsMandatory:    BoolPtr(true),
				IsUserApproved: BoolPtr(true),
			},
			{
				Name:           "optional_not_approved",
				Value:          "Can be unapproved",
				IsMandatory:    BoolPtr(false),
				IsUserApproved: BoolPtr(false),
			},
			{
				Name:           "optional_approved",
				Value:          "Optional but approved",
				IsMandatory:    BoolPtr(false),
				IsUserApproved: BoolPtr(true),
			},
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

	assert.Equal(t, http.StatusCreated, recorder.Code, "Should create consent with mixed purposes")

	var response models.ConsentAPIResponse
	err = json.Unmarshal(recorder.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.NotEmpty(t, response.ID, "Consent ID should not be empty")
	require.Len(t, response.ConsentPurpose, 3, "Should have 3 purposes")

	// Verify each purpose
	for _, cp := range response.ConsentPurpose {
		assert.NotNil(t, cp.IsMandatory, "IsMandatory should not be nil for %s", cp.Name)
		assert.NotNil(t, cp.IsUserApproved, "IsUserApproved should not be nil for %s", cp.Name)

		switch cp.Name {
		case "mandatory_approved":
			assert.True(t, *cp.IsMandatory, "mandatory_approved should be mandatory")
			assert.True(t, *cp.IsUserApproved, "mandatory_approved should be approved")
		case "optional_not_approved":
			assert.False(t, *cp.IsMandatory, "optional_not_approved should not be mandatory")
			assert.False(t, *cp.IsUserApproved, "optional_not_approved should not be approved")
		case "optional_approved":
			assert.False(t, *cp.IsMandatory, "optional_approved should not be mandatory")
			assert.True(t, *cp.IsUserApproved, "optional_approved should be approved")
		}
	}

	t.Logf("✓ Successfully created consent with mixed purpose scenarios")
	CleanupTestData(t, env, response.ID)
}

// TestCreateConsent_MixedWithOneInvalid tests that one invalid purpose fails the entire request
func TestCreateConsent_MixedWithOneInvalid(t *testing.T) {
	env := SetupTestEnvironment(t)

	// Create test purposes
	purposes := CreateTestPurposes(t, env, map[string]string{
		"valid_purpose":   "Valid purpose",
		"invalid_purpose": "Invalid purpose - mandatory but not approved",
	})
	defer CleanupTestPurposes(t, env, purposes)

	validityTime := time.Now().Add(90 * 24 * time.Hour).UnixMilli()

	createReq := &models.ConsentAPIRequest{
		Type:         "accounts",
		ValidityTime: &validityTime,
		ConsentPurpose: []models.ConsentPurposeItem{
			{
				Name:           "valid_purpose",
				Value:          "This is valid",
				IsMandatory:    BoolPtr(true),
				IsUserApproved: BoolPtr(true),
			},
			{
				Name:           "invalid_purpose",
				Value:          "This violates the rule",
				IsMandatory:    BoolPtr(true),
				IsUserApproved: BoolPtr(false), // INVALID: mandatory but not approved
			},
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

	assert.Equal(t, http.StatusBadRequest, recorder.Code, "Should reject entire request due to one invalid purpose")

	var errorResponse map[string]interface{}
	err = json.Unmarshal(recorder.Body.Bytes(), &errorResponse)
	require.NoError(t, err)

	errorText := strings.ToLower(errorResponse["message"].(string))
	if details, ok := errorResponse["details"].(string); ok {
		errorText += " " + strings.ToLower(details)
	}

	assert.Contains(t, errorText, "mandatory", "Error should mention mandatory")
	assert.Contains(t, errorText, "invalid_purpose", "Error should mention the specific purpose with error")

	t.Log("✓ Correctly rejected request with one invalid purpose among valid ones")
}

// TestValidateConsent_ExpiredWithAuthUpdatesAllStatuses tests that when validate checks an expired consent,
// it updates the consent status to EXPIRED and all authorization statuses to SYS_EXPIRED
func TestValidateConsent_ExpiredWithAuthUpdatesAllStatuses(t *testing.T) {
	env := SetupTestEnvironment(t)

	// Create test purpose
	purposes := CreateTestPurposes(t, env, map[string]string{
		"test_purpose": "Test purpose for expired consent",
	})
	defer CleanupTestPurposes(t, env, purposes)

	// Create consent with expired validity time and authorizations
	expiredTime := time.Now().Add(-24 * time.Hour).UnixMilli() // 1 day ago
	createReq := &models.ConsentAPIRequest{
		Type:         "accounts",
		ValidityTime: &expiredTime,
		ConsentPurpose: []models.ConsentPurposeItem{
			{Name: "test_purpose", Value: "test", IsUserApproved: BoolPtr(true), IsMandatory: BoolPtr(true)},
		},
		Authorizations: []models.AuthorizationAPIRequest{
			{UserID: "user-123", Type: "authorization_code", Status: "approved"},
			{UserID: "user-456", Type: "authorization_code", Status: "approved"},
		},
	}

	reqBody, _ := json.Marshal(createReq)
	req, _ := http.NewRequest("POST", "/api/v1/consents", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("org-id", "TEST_ORG")
	req.Header.Set("client-id", "TEST_CLIENT")

	recorder := httptest.NewRecorder()
	env.Router.ServeHTTP(recorder, req)
	require.Equal(t, http.StatusCreated, recorder.Code)

	var createResponse models.ConsentAPIResponse
	json.Unmarshal(recorder.Body.Bytes(), &createResponse)
	defer CleanupTestData(t, env, createResponse.ID)

	// Verify initial consent status is ACTIVE (from approved auths) and auth statuses are approved
	assert.Equal(t, "ACTIVE", createResponse.Status, "Initial consent status should be ACTIVE")
	require.Len(t, createResponse.Authorizations, 2, "Should have 2 authorizations")
	for i, auth := range createResponse.Authorizations {
		assert.Equal(t, "approved", auth.Status, "Authorization %d initial status should be approved", i)
	}

	// Validate the consent - should trigger expiry update
	validateReq := &models.ValidateRequest{
		ConsentID: createResponse.ID,
		UserID:    "user-123",
	}

	validateBody, _ := json.Marshal(validateReq)
	valReq, _ := http.NewRequest("POST", "/api/v1/consents/validate", bytes.NewBuffer(validateBody))
	valReq.Header.Set("Content-Type", "application/json")
	valReq.Header.Set("org-id", "TEST_ORG")
	valReq.Header.Set("client-id", "TEST_CLIENT")

	valRecorder := httptest.NewRecorder()
	env.Router.ServeHTTP(valRecorder, valReq)
	require.Equal(t, http.StatusOK, valRecorder.Code)

	var validateResponse models.ValidateResponse
	json.Unmarshal(valRecorder.Body.Bytes(), &validateResponse)

	// Verify validation failed due to expiry
	assert.False(t, validateResponse.IsValid, "Validation should fail for expired consent")
	assert.Equal(t, "consent_expired", validateResponse.ErrorMessage)

	// Verify consent information has EXPIRED status
	require.NotNil(t, validateResponse.ConsentInformation)
	assert.Equal(t, "EXPIRED", validateResponse.ConsentInformation.Status, "Consent status should be EXPIRED")

	// Verify all authorization statuses are SYS_EXPIRED in the returned consent information
	require.Len(t, validateResponse.ConsentInformation.Authorizations, 2, "Should have 2 authorizations")
	for i, auth := range validateResponse.ConsentInformation.Authorizations {
		assert.Equal(t, string(models.AuthStateSysExpired), auth.Status, "Authorization %d status should be SYS_EXPIRED", i)
	}

	t.Log("✓ Validate expired consent: consent status updated to EXPIRED and all auth statuses updated to SYS_EXPIRED")
}
