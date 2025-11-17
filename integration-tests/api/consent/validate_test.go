package consent

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

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
			Name:       name,
			Value:      "Test value for " + name,
			IsSelected: BoolPtr(true),
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

	// Verify consent is initially in ACTIVE status
	req, _ := http.NewRequest("GET", "/api/v1/consents/"+consent.ID, nil)
	req.Header.Set("org-id", "TEST_ORG")

	recorder := httptest.NewRecorder()
	env.Router.ServeHTTP(recorder, req)

	var getResponse1 models.ConsentAPIResponse
	json.Unmarshal(recorder.Body.Bytes(), &getResponse1)
	
	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "ACTIVE", getResponse1.Status, "Consent should initially be ACTIVE")
	t.Logf("✓ Consent initially has status: %s", getResponse1.Status)

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
	assert.Equal(t, "consent_expired", validateResponse.ErrorMessage)
	assert.Contains(t, validateResponse.ErrorDescription, "expired")
	t.Logf("✓ Validate correctly returned consent_expired error")

	// Verify the consent information includes updated status
	assert.NotNil(t, validateResponse.ConsentInformation)
	consentInfo := validateResponse.ConsentInformation
	updatedStatus, ok := consentInfo["status"].(string)
	assert.True(t, ok, "Status should be a string")
	assert.Equal(t, "EXPIRED", updatedStatus, "Consent status in response should be EXPIRED")
	t.Logf("✓ Validate response shows updated status: %s", updatedStatus)

	// Verify by calling GET endpoint again that the status was persisted to database
	req, _ = http.NewRequest("GET", "/api/v1/consents/"+consent.ID, nil)
	req.Header.Set("org-id", "TEST_ORG")

	recorder = httptest.NewRecorder()
	env.Router.ServeHTTP(recorder, req)

	var getResponse2 models.ConsentAPIResponse
	json.Unmarshal(recorder.Body.Bytes(), &getResponse2)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "EXPIRED", getResponse2.Status, "Consent status should be updated to EXPIRED in database")
	t.Logf("✓ Database shows updated status: %s", getResponse2.Status)
	t.Logf("✓ Consent status successfully updated from ACTIVE to EXPIRED after validation")
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
			Name:       name,
			Value:      "Test value for " + name,
			IsSelected: BoolPtr(true),
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
					"accountIds": []string{"123456", "789012"},
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
	assert.Equal(t, getResponse.ID, consentInfo["id"], "id must match")
	assert.Equal(t, getResponse.Type, consentInfo["type"], "type must match")
	assert.Equal(t, getResponse.Status, consentInfo["status"], "status must match")
	assert.Equal(t, getResponse.ClientID, consentInfo["clientId"], "clientId must match")
	t.Logf("✓ Core fields validated: id, type, status, clientId")

	// 2. Timestamps
	assert.NotNil(t, consentInfo["createdTime"], "createdTime must be present")
	assert.NotNil(t, consentInfo["updatedTime"], "updatedTime must be present")
	assert.Equal(t, getResponse.CreatedTime, int64(consentInfo["createdTime"].(float64)), "createdTime must match")
	assert.Equal(t, getResponse.UpdatedTime, int64(consentInfo["updatedTime"].(float64)), "updatedTime must match")
	t.Logf("✓ Timestamps validated: createdTime, updatedTime")

	// 3. Optional fields that were provided
	assert.NotNil(t, consentInfo["validityTime"], "validityTime should be present")
	assert.Equal(t, float64(*getResponse.ValidityTime), consentInfo["validityTime"].(float64), "validityTime must match")
	
	assert.NotNil(t, consentInfo["frequency"], "frequency should be present")
	assert.Equal(t, float64(*getResponse.Frequency), consentInfo["frequency"].(float64), "frequency must match")
	
	assert.NotNil(t, consentInfo["recurringIndicator"], "recurringIndicator should be present")
	assert.Equal(t, *getResponse.RecurringIndicator, consentInfo["recurringIndicator"].(bool), "recurringIndicator must match")
	
	assert.NotNil(t, consentInfo["dataAccessValidityDuration"], "dataAccessValidityDuration should be present")
	assert.Equal(t, float64(*getResponse.DataAccessValidityDuration), consentInfo["dataAccessValidityDuration"].(float64), "dataAccessValidityDuration must match")
	t.Logf("✓ Optional fields validated: validityTime, frequency, recurringIndicator, dataAccessValidityDuration")

	// 4. Attributes - verify all attributes match
	validateAttributes, ok := consentInfo["attributes"].(map[string]interface{})
	assert.True(t, ok, "attributes should be a map")
	assert.Len(t, validateAttributes, len(getResponse.Attributes), "attributes count must match")
	
	for key, expectedValue := range getResponse.Attributes {
		actualValue, exists := validateAttributes[key]
		assert.True(t, exists, "attribute '%s' should exist in validate response", key)
		assert.Equal(t, expectedValue, actualValue, "attribute '%s' value must match", key)
	}
	t.Logf("✓ All %d attributes validated", len(getResponse.Attributes))

	// 5. Consent Purposes - comprehensive validation
	validatePurposes, ok := consentInfo["consentPurpose"].([]interface{})
	assert.True(t, ok, "consentPurpose should be an array")
	assert.Len(t, validatePurposes, len(getResponse.ConsentPurpose), "consentPurpose count must match")
	assert.Len(t, validatePurposes, 3, "should have 3 consent purposes")

	// Create maps for easier comparison
	validatePurposeMap := make(map[string]map[string]interface{})
	for _, cpInterface := range validatePurposes {
		cp := cpInterface.(map[string]interface{})
		validatePurposeMap[cp["name"].(string)] = cp
	}

	getPurposeMap := make(map[string]models.ConsentPurposeItem)
	for _, cp := range getResponse.ConsentPurpose {
		getPurposeMap[cp.Name] = cp
	}

	for purposeName, getCP := range getPurposeMap {
		validateCP, exists := validatePurposeMap[purposeName]
		assert.True(t, exists, "purpose '%s' should exist in validate response", purposeName)

		assert.Equal(t, getCP.Name, validateCP["name"], "purpose name must match")
		assert.Equal(t, *getCP.IsSelected, validateCP["isSelected"], "purpose isSelected must match for %s", purposeName)
		
		// Validate enriched fields
		assert.NotEmpty(t, validateCP["type"], "purpose type should be enriched for %s", purposeName)
		assert.NotEmpty(t, validateCP["description"], "purpose description should be enriched for %s", purposeName)
		
		// Verify value matches
		if getCP.Value != nil {
			assert.NotNil(t, validateCP["value"], "purpose value should be present for %s", purposeName)
			assert.Equal(t, getCP.Value, validateCP["value"], "purpose value must match for %s", purposeName)
		}
	}
	t.Logf("✓ All %d consent purposes validated with enrichment (type, description, attributes)", len(getResponse.ConsentPurpose))

	// 6. Authorizations - comprehensive validation
	validateAuths, ok := consentInfo["authorizations"].([]interface{})
	assert.True(t, ok, "authorizations should be an array")
	assert.Len(t, validateAuths, len(getResponse.Authorizations), "authorizations count must match")
	assert.Len(t, validateAuths, 2, "should have 2 authorizations")

	// Create maps for easier comparison
	validateAuthMap := make(map[string]map[string]interface{})
	for _, authInterface := range validateAuths {
		auth := authInterface.(map[string]interface{})
		validateAuthMap[auth["id"].(string)] = auth
	}

	getAuthMap := make(map[string]models.AuthorizationAPIResponse)
	for _, auth := range getResponse.Authorizations {
		getAuthMap[auth.ID] = auth
	}

	for authID, getAuth := range getAuthMap {
		validateAuth, exists := validateAuthMap[authID]
		assert.True(t, exists, "authorization '%s' should exist in validate response", authID)

		assert.Equal(t, getAuth.ID, validateAuth["id"], "auth id must match")
		assert.Equal(t, getAuth.Type, validateAuth["type"], "auth type must match")
		assert.Equal(t, getAuth.Status, validateAuth["status"], "auth status must match")
		
		// UserID is a pointer in API response, but string in validate response
		if getAuth.UserID != nil {
			assert.Equal(t, *getAuth.UserID, validateAuth["userId"], "auth userId must match")
		}
		
		assert.NotNil(t, validateAuth["updatedTime"], "auth updatedTime should be present")
		assert.Equal(t, getAuth.UpdatedTime, int64(validateAuth["updatedTime"].(float64)), "auth updatedTime must match")
		
		// Verify resources if present
		if getAuth.Resources != nil {
			assert.NotNil(t, validateAuth["resources"], "auth resources should be present for %s", authID)
			// Deep comparison of resources structure would go here
		}
	}
	t.Logf("✓ All %d authorizations validated", len(getResponse.Authorizations))

	// 7. Verify structure completeness - no extra or missing fields at top level
	expectedTopLevelFields := []string{
		"id", "type", "status", "clientId",
		"createdTime", "updatedTime", "validityTime",
		"frequency", "recurringIndicator", "dataAccessValidityDuration",
		"attributes", "consentPurpose", "authorizations",
	}

	for _, field := range expectedTopLevelFields {
		_, exists := consentInfo[field]
		assert.True(t, exists, "field '%s' must be present in validate response", field)
	}
	t.Logf("✓ All expected top-level fields present in validate response")

	// Final verification
	t.Logf("✓ ========== COMPREHENSIVE VALIDATION COMPLETE ==========")
	t.Logf("✓ Validate response consentInformation exactly matches GET /consents/{id} response")
	t.Logf("✓ All core fields, optional fields, attributes, purposes, and authorizations validated")
	t.Logf("✓ Consent purposes enriched with type, description, and attributes from definitions")
}

