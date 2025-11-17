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
	req, _ := http.NewRequest("POST", "/api/v1/validate", bytes.NewReader(requestBody))
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
	req, _ := http.NewRequest("POST", "/api/v1/validate", bytes.NewReader(requestBody))
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
	req, _ := http.NewRequest("POST", "/api/v1/validate", bytes.NewReader(requestBody))
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
	req, _ := http.NewRequest("POST", "/api/v1/validate", bytes.NewReader(requestBody))
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
	req, _ := http.NewRequest("POST", "/api/v1/validate", bytes.NewReader(requestBody))
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
	req, _ := http.NewRequest("POST", "/api/v1/validate", bytes.NewReader(requestBody))
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

// TestValidateConsent_MissingResourceParams tests validation without resource params
func TestValidateConsent_MissingResourceParams(t *testing.T) {
	env := SetupTestEnvironment(t)

	validateRequest := models.ValidateRequest{
		ConsentID: "some-consent-id",
		UserID:    "test-user-123",
	}

	requestBody, _ := json.Marshal(validateRequest)
	req, _ := http.NewRequest("POST", "/api/v1/validate", bytes.NewReader(requestBody))
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
	assert.Contains(t, response.ErrorDescription, "resource")
	t.Logf("✓ Missing resource params validation failed correctly")
}

// TestValidateConsent_InvalidJSON tests validation with invalid JSON
func TestValidateConsent_InvalidJSON(t *testing.T) {
	env := SetupTestEnvironment(t)

	req, _ := http.NewRequest("POST", "/api/v1/validate", bytes.NewReader([]byte("{invalid json")))
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
