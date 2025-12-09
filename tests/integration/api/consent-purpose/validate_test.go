package consentpurpose

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestValidateConsentPurposes_AllValid tests validating all valid purpose names
func TestValidateConsentPurposes_AllValid(t *testing.T) {
	env := setupConsentPurposeAPITestEnvironment(t)

	// Create some purposes first
	createReq := []map[string]interface{}{
		{"name": "validate_test_utility_read", "description": "Utility Read", "type": "string", "attributes": map[string]string{"value": "utility:read"}},
		{"name": "validate_test_taxes_read", "description": "Taxes Read", "type": "string", "attributes": map[string]string{"value": "taxes:read"}},
		{"name": "validate_test_profile_read", "description": "Profile Read", "type": "string", "attributes": map[string]string{"value": "profile:read"}},
	}

	body, _ := json.Marshal(createReq)
	req := httptest.NewRequest("POST", "/api/v1/consent-purposes", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("org-id", "TEST_ORG")

	w := httptest.NewRecorder()
	env.Router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	// Extract purpose IDs for cleanup
	var createResponse map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &createResponse)
	require.NoError(t, err)

	data := createResponse["data"].([]interface{})
	var purposeIDs []string
	for _, item := range data {
		purpose := item.(map[string]interface{})
		purposeIDs = append(purposeIDs, purpose["id"].(string))
	}

	// Cleanup function
	defer func() {
		for _, id := range purposeIDs {
			_ = env.PurposeDAO.Delete(req.Context(), id, "TEST_ORG")
		}
		t.Log("✓ Cleaned up created purposes")
	}()

	// Now validate the purpose names
	validateReq := []string{"validate_test_utility_read", "validate_test_taxes_read"}
	body, _ = json.Marshal(validateReq)
	req = httptest.NewRequest("POST", "/api/v1/consent-purposes/validate", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("org-id", "TEST_ORG")

	w = httptest.NewRecorder()
	env.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response []string
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Len(t, response, 2)
	assert.Contains(t, response, "validate_test_utility_read")
	assert.Contains(t, response, "validate_test_taxes_read")

	t.Log("✓ Successfully validated all valid purpose names")
}

// TestValidateConsentPurposes_PartialValid tests validating with some valid and some invalid purpose names
func TestValidateConsentPurposes_PartialValid(t *testing.T) {
	env := setupConsentPurposeAPITestEnvironment(t)

	// Create some purposes
	createReq := []map[string]interface{}{
		{"name": "partial_valid_utility", "description": "Utility Read", "type": "string", "attributes": map[string]string{"value": "utility:read"}},
		{"name": "partial_valid_taxes", "description": "Taxes Read", "type": "string", "attributes": map[string]string{"value": "taxes:read"}},
	}

	body, _ := json.Marshal(createReq)
	req := httptest.NewRequest("POST", "/api/v1/consent-purposes", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("org-id", "TEST_ORG")

	w := httptest.NewRecorder()
	env.Router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	// Extract purpose IDs for cleanup
	var createResponse map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &createResponse)
	require.NoError(t, err)

	data := createResponse["data"].([]interface{})
	var purposeIDs []string
	for _, item := range data {
		purpose := item.(map[string]interface{})
		purposeIDs = append(purposeIDs, purpose["id"].(string))
	}

	// Cleanup function
	defer func() {
		for _, id := range purposeIDs {
			_ = env.PurposeDAO.Delete(req.Context(), id, "TEST_ORG")
		}
		t.Log("✓ Cleaned up created purposes")
	}()

	// Validate with some valid and some invalid names
	validateReq := []string{"partial_valid_utility", "nonexistent_purpose", "partial_valid_taxes", "another_invalid"}
	body, _ = json.Marshal(validateReq)
	req = httptest.NewRequest("POST", "/api/v1/consent-purposes/validate", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("org-id", "TEST_ORG")

	w = httptest.NewRecorder()
	env.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response []string
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// Should only return the valid ones
	assert.Len(t, response, 2)
	assert.Contains(t, response, "partial_valid_utility")
	assert.Contains(t, response, "partial_valid_taxes")
	assert.NotContains(t, response, "nonexistent_purpose")
	assert.NotContains(t, response, "another_invalid")

	t.Log("✓ Successfully returned only valid purpose names")
}

// TestValidateConsentPurposes_NoneValid tests validating with all invalid purpose names
func TestValidateConsentPurposes_NoneValid(t *testing.T) {
	env := setupConsentPurposeAPITestEnvironment(t)

	// Validate with all invalid names
	validateReq := []string{"nonexistent1", "nonexistent2", "invalid_purpose"}
	body, _ := json.Marshal(validateReq)
	req := httptest.NewRequest("POST", "/api/v1/consent-purposes/validate", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("org-id", "TEST_ORG")

	w := httptest.NewRecorder()
	env.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var errorResp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &errorResp)
	require.NoError(t, err)

	assert.Equal(t, "BAD_REQUEST", errorResp["code"])
	assert.Contains(t, errorResp["details"], "no valid purposes found")
	t.Log("✓ Correctly returned error when no valid purposes found")
}

// TestValidateConsentPurposes_EmptyRequest tests validating with empty request
func TestValidateConsentPurposes_EmptyRequest(t *testing.T) {
	env := setupConsentPurposeAPITestEnvironment(t)

	validateReq := []string{}
	body, _ := json.Marshal(validateReq)
	req := httptest.NewRequest("POST", "/api/v1/consent-purposes/validate", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("org-id", "TEST_ORG")

	w := httptest.NewRecorder()
	env.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var errorResp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &errorResp)
	require.NoError(t, err)

	assert.Equal(t, "BAD_REQUEST", errorResp["code"])
	assert.Contains(t, errorResp["details"], "at least one purpose name must be provided")
	t.Log("✓ Correctly rejected empty request")
}

// TestValidateConsentPurposes_MissingOrgID tests validating without org-id header
func TestValidateConsentPurposes_MissingOrgID(t *testing.T) {
	env := setupConsentPurposeAPITestEnvironment(t)

	validateReq := []string{"utility_read"}
	body, _ := json.Marshal(validateReq)
	req := httptest.NewRequest("POST", "/api/v1/consent-purposes/validate", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	// Not setting org-id header

	w := httptest.NewRecorder()
	env.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var errorResp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &errorResp)
	require.NoError(t, err)

	// When org-id is missing, it's treated as empty string and no purposes will be found
	assert.Equal(t, "BAD_REQUEST", errorResp["code"])
	t.Log("✓ Correctly rejected request without org-id")
}
