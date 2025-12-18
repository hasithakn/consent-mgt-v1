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

// TestSearchConsentsByAttribute_ByKeyOnly tests searching by attribute key only
func TestSearchConsentsByAttribute_ByKeyOnly(t *testing.T) {
	env := SetupTestEnvironment(t)

	// Create test purposes
	purposes := CreateTestPurposes(t, env, map[string]string{
		"test_data_access": "Test data access purpose",
	})
	defer CleanupTestPurposes(t, env, purposes)

	// Create test consents with attributes
	consent1 := createConsentWithAttributes(t, env, purposes, map[string]string{
		"source":      "mobile-app",
		"environment": "production",
	})
	defer CleanupTestData(t, env, consent1.ID)

	consent2 := createConsentWithAttributes(t, env, purposes, map[string]string{
		"source":  "web-app",
		"channel": "online",
	})
	defer CleanupTestData(t, env, consent2.ID)

	consent3 := createConsentWithAttributes(t, env, purposes, map[string]string{
		"channel": "mobile",
	})
	defer CleanupTestData(t, env, consent3.ID)

	t.Logf("Created test consents: %s, %s, %s", consent1.ID, consent2.ID, consent3.ID)

	// Search by key "source" - should return consent1 and consent2
	req, err := http.NewRequest("GET", "/api/v1/consents/attributes?key=source", nil)
	require.NoError(t, err)
	req.Header.Set("org-id", "TEST_ORG")

	recorder := httptest.NewRecorder()
	env.Router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)

	var response models.ConsentAttributeSearchResponse
	err = json.Unmarshal(recorder.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, 2, response.Count, "Expected 2 consents with 'source' attribute")
	assert.Len(t, response.ConsentIDs, 2)
	assert.Contains(t, response.ConsentIDs, consent1.ID)
	assert.Contains(t, response.ConsentIDs, consent2.ID)
	assert.NotContains(t, response.ConsentIDs, consent3.ID)

	t.Logf("✓ Search by key 'source' returned: %v", response.ConsentIDs)
}

// TestSearchConsentsByAttribute_ByKeyAndValue tests searching by key-value pair
func TestSearchConsentsByAttribute_ByKeyAndValue(t *testing.T) {
	env := SetupTestEnvironment(t)

	// Create test purposes
	purposes := CreateTestPurposes(t, env, map[string]string{
		"test_data_access": "Test data access purpose",
	})
	defer CleanupTestPurposes(t, env, purposes)

	// Create test consents
	consent1 := createConsentWithAttributes(t, env, purposes, map[string]string{
		"source":      "mobile-app",
		"environment": "production",
	})
	defer CleanupTestData(t, env, consent1.ID)

	consent2 := createConsentWithAttributes(t, env, purposes, map[string]string{
		"source":      "mobile-app",
		"environment": "staging",
	})
	defer CleanupTestData(t, env, consent2.ID)

	consent3 := createConsentWithAttributes(t, env, purposes, map[string]string{
		"source": "web-app",
	})
	defer CleanupTestData(t, env, consent3.ID)

	// Search by key="source" and value="mobile-app" - should return consent1 and consent2
	req, err := http.NewRequest("GET", "/api/v1/consents/attributes?key=source&value=mobile-app", nil)
	require.NoError(t, err)
	req.Header.Set("org-id", "TEST_ORG")

	recorder := httptest.NewRecorder()
	env.Router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)

	var response models.ConsentAttributeSearchResponse
	err = json.Unmarshal(recorder.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, 2, response.Count)
	assert.Len(t, response.ConsentIDs, 2)
	assert.Contains(t, response.ConsentIDs, consent1.ID)
	assert.Contains(t, response.ConsentIDs, consent2.ID)
	assert.NotContains(t, response.ConsentIDs, consent3.ID)

	t.Logf("✓ Search by source=mobile-app returned: %v", response.ConsentIDs)
}

// TestSearchConsentsByAttribute_NoResults tests search with no matching results
func TestSearchConsentsByAttribute_NoResults(t *testing.T) {
	env := SetupTestEnvironment(t)

	// Create test purposes
	purposes := CreateTestPurposes(t, env, map[string]string{
		"test_data_access": "Test data access purpose",
	})
	defer CleanupTestPurposes(t, env, purposes)

	// Create a test consent with different attributes
	consent := createConsentWithAttributes(t, env, purposes, map[string]string{
		"source": "mobile-app",
	})
	defer CleanupTestData(t, env, consent.ID)

	// Search for non-existent attribute
	req, err := http.NewRequest("GET", "/api/v1/consents/attributes?key=non_existent_key", nil)
	require.NoError(t, err)
	req.Header.Set("org-id", "TEST_ORG")

	recorder := httptest.NewRecorder()
	env.Router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)

	var response models.ConsentAttributeSearchResponse
	err = json.Unmarshal(recorder.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, 0, response.Count)
	assert.Empty(t, response.ConsentIDs)

	t.Logf("✓ Search for non-existent key returned empty results")
}

// TestSearchConsentsByAttribute_MissingKey tests error when key parameter is missing
func TestSearchConsentsByAttribute_MissingKey(t *testing.T) {
	env := SetupTestEnvironment(t)

	// Request without key parameter
	req, err := http.NewRequest("GET", "/api/v1/consents/attributes", nil)
	require.NoError(t, err)
	req.Header.Set("org-id", "TEST_ORG")

	recorder := httptest.NewRecorder()
	env.Router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusBadRequest, recorder.Code)

	var response map[string]interface{}
	err = json.Unmarshal(recorder.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response["details"], "key parameter is required")
	t.Logf("✓ Correctly rejected request without key parameter")
}

// TestSearchConsentsByAttribute_OrganizationIsolation tests org isolation
func TestSearchConsentsByAttribute_OrganizationIsolation(t *testing.T) {
	env := SetupTestEnvironment(t)

	// Create test purposes
	purposes := CreateTestPurposes(t, env, map[string]string{
		"test_data_access": "Test data access purpose",
	})
	defer CleanupTestPurposes(t, env, purposes)

	// Create consent for TEST_ORG
	consent := createConsentWithAttributes(t, env, purposes, map[string]string{
		"source": "mobile-app",
	})
	defer CleanupTestData(t, env, consent.ID)

	// Search from different org - should not find the consent
	req, err := http.NewRequest("GET", "/api/v1/consents/attributes?key=source", nil)
	require.NoError(t, err)
	req.Header.Set("org-id", "DIFFERENT_ORG")

	recorder := httptest.NewRecorder()
	env.Router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)

	var response models.ConsentAttributeSearchResponse
	err = json.Unmarshal(recorder.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, 0, response.Count, "Should not find consents from different organization")
	assert.Empty(t, response.ConsentIDs)

	t.Logf("✓ Organization isolation test passed - no cross-org data leakage")
}

// TestSearchConsentsByAttribute_EmptyValue tests search with empty value parameter
func TestSearchConsentsByAttribute_EmptyValue(t *testing.T) {
	env := SetupTestEnvironment(t)

	// Create test purposes
	purposes := CreateTestPurposes(t, env, map[string]string{
		"test_data_access": "Test data access purpose",
	})
	defer CleanupTestPurposes(t, env, purposes)

	// Create consent
	consent := createConsentWithAttributes(t, env, purposes, map[string]string{
		"source": "mobile-app",
	})
	defer CleanupTestData(t, env, consent.ID)

	// Search with key and empty value - should work as key-only search
	req, err := http.NewRequest("GET", "/api/v1/consents/attributes?key=source&value=", nil)
	require.NoError(t, err)
	req.Header.Set("org-id", "TEST_ORG")

	recorder := httptest.NewRecorder()
	env.Router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)

	var response models.ConsentAttributeSearchResponse
	err = json.Unmarshal(recorder.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, 1, response.Count)
	assert.Contains(t, response.ConsentIDs, consent.ID)

	t.Logf("✓ Empty value parameter treated as key-only search")
}

// Helper function to create a consent with specific attributes
func createConsentWithAttributes(t *testing.T, env *TestEnvironment, purposes map[string]*models.ConsentPurpose, attributes map[string]string) *models.ConsentAPIResponse {
	// Build consent purposes array from the created purposes
	var consentPurposes []models.ConsentPurposeItem
	for name := range purposes {
		consentPurposes = append(consentPurposes, models.ConsentPurposeItem{
			Name:           name,
			Value:          "Test value for " + name,
			IsUserApproved: BoolPtr(true),
			IsMandatory:    BoolPtr(true),
		})
	}

	createReq := &models.ConsentAPIRequest{
		Type:           "accounts",
		ConsentPurpose: consentPurposes,
		Attributes:     attributes,
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

	if recorder.Code != http.StatusCreated {
		t.Fatalf("Failed to create consent: %d - %s", recorder.Code, recorder.Body.String())
	}

	var response models.ConsentAPIResponse
	err = json.Unmarshal(recorder.Body.Bytes(), &response)
	require.NoError(t, err)

	return &response
}
