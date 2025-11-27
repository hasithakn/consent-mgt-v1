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

// TestDeleteConsent_Success tests successful deletion of a consent
func TestDeleteConsent_Success(t *testing.T) {
	env := SetupTestEnvironment(t)

	purposes := CreateTestPurposes(t, env, map[string]string{
		"data_access": "Data access purpose",
	})
	defer CleanupTestPurposes(t, env, purposes)

	// Create a consent to delete
	createReq := &models.ConsentAPIRequest{
		Type: "accounts",
		ConsentPurpose: []models.ConsentPurposeItem{
			{Name: "data_access", Value: "Test consent", IsUserApproved: BoolPtr(true), IsMandatory: BoolPtr(true)},
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

	// Delete the consent
	deleteReq, err := http.NewRequest("DELETE", "/api/v1/consents/"+createResp.ID, nil)
	require.NoError(t, err)
	deleteReq.Header.Set("org-id", "TEST_ORG")
	deleteReq.Header.Set("client-id", "TEST_CLIENT")

	deleteRecorder := httptest.NewRecorder()
	env.Router.ServeHTTP(deleteRecorder, deleteReq)

	assert.Equal(t, http.StatusNoContent, deleteRecorder.Code, "Delete should return 204 No Content")

	// Verify consent is deleted by trying to GET it
	getReq, err := http.NewRequest("GET", "/api/v1/consents/"+createResp.ID, nil)
	require.NoError(t, err)
	getReq.Header.Set("org-id", "TEST_ORG")
	getReq.Header.Set("client-id", "TEST_CLIENT")

	getRecorder := httptest.NewRecorder()
	env.Router.ServeHTTP(getRecorder, getReq)

	assert.Equal(t, http.StatusNotFound, getRecorder.Code, "GET after DELETE should return 404")

	t.Logf("✓ Successfully deleted consent %s", createResp.ID)
}

// TestDeleteConsent_NotFound tests deleting a non-existent consent
func TestDeleteConsent_NotFound(t *testing.T) {
	env := SetupTestEnvironment(t)

	// Try to delete a non-existent consent
	req, err := http.NewRequest("DELETE", "/api/v1/consents/CONSENT-nonexistent-12345", nil)
	require.NoError(t, err)
	req.Header.Set("org-id", "TEST_ORG")
	req.Header.Set("client-id", "TEST_CLIENT")

	recorder := httptest.NewRecorder()
	env.Router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusNotFound, recorder.Code, "Should return 404 for non-existent consent")

	t.Logf("✓ Correctly returned 404 for non-existent consent")
}

// TestDeleteConsent_DifferentOrgID tests that consent deletion is scoped by organization
func TestDeleteConsent_DifferentOrgID(t *testing.T) {
	env := SetupTestEnvironment(t)

	purposes := CreateTestPurposes(t, env, map[string]string{
		"data_access": "Data access purpose",
	})
	defer CleanupTestPurposes(t, env, purposes)

	// Create a consent with one org
	createReq := &models.ConsentAPIRequest{
		Type: "accounts",
		ConsentPurpose: []models.ConsentPurposeItem{
			{Name: "data_access", Value: "Test consent", IsUserApproved: BoolPtr(true), IsMandatory: BoolPtr(true)},
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

	// Try to delete with a different org-id
	deleteReq, err := http.NewRequest("DELETE", "/api/v1/consents/"+createResp.ID, nil)
	require.NoError(t, err)
	deleteReq.Header.Set("org-id", "DIFFERENT_ORG")
	deleteReq.Header.Set("client-id", "TEST_CLIENT")

	deleteRecorder := httptest.NewRecorder()
	env.Router.ServeHTTP(deleteRecorder, deleteReq)

	assert.Equal(t, http.StatusNotFound, deleteRecorder.Code, "Should return 404 when trying to delete consent from different org")

	// Verify consent still exists with original org
	getReq, err := http.NewRequest("GET", "/api/v1/consents/"+createResp.ID, nil)
	require.NoError(t, err)
	getReq.Header.Set("org-id", "TEST_ORG")
	getReq.Header.Set("client-id", "TEST_CLIENT")

	getRecorder := httptest.NewRecorder()
	env.Router.ServeHTTP(getRecorder, getReq)

	assert.Equal(t, http.StatusOK, getRecorder.Code, "Consent should still exist for original org")

	t.Logf("✓ Correctly prevented deletion of consent from different org")
}

// TestDeleteConsent_WithAuthorizationResources tests deleting a consent with authorization resources
func TestDeleteConsent_WithAuthorizationResources(t *testing.T) {
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
			{
				UserID: "user-456",
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
	require.Len(t, createResp.Authorizations, 2, "Should have 2 authorizations")

	// Delete the consent
	deleteReq, err := http.NewRequest("DELETE", "/api/v1/consents/"+createResp.ID, nil)
	require.NoError(t, err)
	deleteReq.Header.Set("org-id", "TEST_ORG")
	deleteReq.Header.Set("client-id", "TEST_CLIENT")

	deleteRecorder := httptest.NewRecorder()
	env.Router.ServeHTTP(deleteRecorder, deleteReq)

	assert.Equal(t, http.StatusNoContent, deleteRecorder.Code, "Delete should return 204 No Content")

	// Verify consent and authorizations are deleted
	getReq, err := http.NewRequest("GET", "/api/v1/consents/"+createResp.ID, nil)
	require.NoError(t, err)
	getReq.Header.Set("org-id", "TEST_ORG")
	getReq.Header.Set("client-id", "TEST_CLIENT")

	getRecorder := httptest.NewRecorder()
	env.Router.ServeHTTP(getRecorder, getReq)

	assert.Equal(t, http.StatusNotFound, getRecorder.Code, "Consent should be deleted along with authorizations")

	t.Logf("✓ Successfully deleted consent %s with authorization resources", createResp.ID)
}

// TestDeleteConsent_WithAttributes tests deleting a consent with attributes
func TestDeleteConsent_WithAttributes(t *testing.T) {
	env := SetupTestEnvironment(t)

	purposes := CreateTestPurposes(t, env, map[string]string{
		"data_access": "Data access purpose",
	})
	defer CleanupTestPurposes(t, env, purposes)

	// Create consent with attributes
	createReq := &models.ConsentAPIRequest{
		Type: "accounts",
		ConsentPurpose: []models.ConsentPurposeItem{
			{Name: "data_access", Value: "Test consent", IsUserApproved: BoolPtr(true), IsMandatory: BoolPtr(true)},
		},
		Attributes: map[string]string{
			"account_id":   "ACC-12345",
			"account_type": "SAVINGS",
			"customer_id":  "CUST-67890",
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
	require.NotEmpty(t, createResp.Attributes, "Should have attributes")

	// Delete the consent
	deleteReq, err := http.NewRequest("DELETE", "/api/v1/consents/"+createResp.ID, nil)
	require.NoError(t, err)
	deleteReq.Header.Set("org-id", "TEST_ORG")
	deleteReq.Header.Set("client-id", "TEST_CLIENT")

	deleteRecorder := httptest.NewRecorder()
	env.Router.ServeHTTP(deleteRecorder, deleteReq)

	assert.Equal(t, http.StatusNoContent, deleteRecorder.Code, "Delete should return 204 No Content")

	// Verify consent and attributes are deleted
	getReq, err := http.NewRequest("GET", "/api/v1/consents/"+createResp.ID, nil)
	require.NoError(t, err)
	getReq.Header.Set("org-id", "TEST_ORG")
	getReq.Header.Set("client-id", "TEST_CLIENT")

	getRecorder := httptest.NewRecorder()
	env.Router.ServeHTTP(getRecorder, getReq)

	assert.Equal(t, http.StatusNotFound, getRecorder.Code, "Consent should be deleted along with attributes")

	t.Logf("✓ Successfully deleted consent %s with attributes", createResp.ID)
}

// TestDeleteConsent_WithMultiplePurposes tests deleting a consent with multiple purposes
func TestDeleteConsent_WithMultiplePurposes(t *testing.T) {
	env := SetupTestEnvironment(t)

	purposes := CreateTestPurposes(t, env, map[string]string{
		"data_access":    "Data access purpose",
		"data_sharing":   "Data sharing purpose",
		"data_analytics": "Data analytics purpose",
	})
	defer CleanupTestPurposes(t, env, purposes)

	// Create consent with multiple purposes
	createReq := &models.ConsentAPIRequest{
		Type: "accounts",
		ConsentPurpose: []models.ConsentPurposeItem{
			{Name: "data_access", Value: "Access data", IsUserApproved: BoolPtr(true), IsMandatory: BoolPtr(true)},
			{Name: "data_sharing", Value: "Share data", IsUserApproved: BoolPtr(true), IsMandatory: BoolPtr(true)},
			{Name: "data_analytics", Value: "Analyze data", IsUserApproved: BoolPtr(false), IsMandatory: BoolPtr(false)},
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
	require.Len(t, createResp.ConsentPurpose, 3, "Should have 3 purposes")

	// Delete the consent
	deleteReq, err := http.NewRequest("DELETE", "/api/v1/consents/"+createResp.ID, nil)
	require.NoError(t, err)
	deleteReq.Header.Set("org-id", "TEST_ORG")
	deleteReq.Header.Set("client-id", "TEST_CLIENT")

	deleteRecorder := httptest.NewRecorder()
	env.Router.ServeHTTP(deleteRecorder, deleteReq)

	assert.Equal(t, http.StatusNoContent, deleteRecorder.Code, "Delete should return 204 No Content")

	// Verify consent and purpose mappings are deleted
	getReq, err := http.NewRequest("GET", "/api/v1/consents/"+createResp.ID, nil)
	require.NoError(t, err)
	getReq.Header.Set("org-id", "TEST_ORG")
	getReq.Header.Set("client-id", "TEST_CLIENT")

	getRecorder := httptest.NewRecorder()
	env.Router.ServeHTTP(getRecorder, getReq)

	assert.Equal(t, http.StatusNotFound, getRecorder.Code, "Consent should be deleted along with purpose mappings")

	t.Logf("✓ Successfully deleted consent %s with multiple purposes", createResp.ID)
}

// TestDeleteConsent_WithAllRelatedData tests deleting a consent with all related data (purposes, attributes, authorizations)
func TestDeleteConsent_WithAllRelatedData(t *testing.T) {
	env := SetupTestEnvironment(t)

	purposes := CreateTestPurposes(t, env, map[string]string{
		"data_access":  "Data access purpose",
		"data_sharing": "Data sharing purpose",
	})
	defer CleanupTestPurposes(t, env, purposes)

	// Create consent with all related data
	createReq := &models.ConsentAPIRequest{
		Type:         "accounts",
		ValidityTime: Int64Ptr(7776000),
		ConsentPurpose: []models.ConsentPurposeItem{
			{Name: "data_access", Value: "Access data", IsUserApproved: BoolPtr(true), IsMandatory: BoolPtr(true)},
			{Name: "data_sharing", Value: "Share data", IsUserApproved: BoolPtr(true), IsMandatory: BoolPtr(true)},
		},
		Attributes: map[string]string{
			"account_id":  "ACC-12345",
			"customer_id": "CUST-67890",
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

	// Verify all data was created
	require.Len(t, createResp.ConsentPurpose, 2, "Should have 2 purposes")
	require.Len(t, createResp.Attributes, 2, "Should have 2 attributes")
	require.Len(t, createResp.Authorizations, 1, "Should have 1 authorization")

	// Delete the consent
	deleteReq, err := http.NewRequest("DELETE", "/api/v1/consents/"+createResp.ID, nil)
	require.NoError(t, err)
	deleteReq.Header.Set("org-id", "TEST_ORG")
	deleteReq.Header.Set("client-id", "TEST_CLIENT")

	deleteRecorder := httptest.NewRecorder()
	env.Router.ServeHTTP(deleteRecorder, deleteReq)

	assert.Equal(t, http.StatusNoContent, deleteRecorder.Code, "Delete should return 204 No Content")

	// Verify consent and all related data are deleted
	getReq, err := http.NewRequest("GET", "/api/v1/consents/"+createResp.ID, nil)
	require.NoError(t, err)
	getReq.Header.Set("org-id", "TEST_ORG")
	getReq.Header.Set("client-id", "TEST_CLIENT")

	getRecorder := httptest.NewRecorder()
	env.Router.ServeHTTP(getRecorder, getReq)

	assert.Equal(t, http.StatusNotFound, getRecorder.Code, "Consent should be deleted along with all related data")

	t.Logf("✓ Successfully deleted consent %s with all related data (purposes, attributes, authorizations)", createResp.ID)
}

// TestDeleteConsent_InvalidConsentID tests deleting a consent with invalid ID format
func TestDeleteConsent_InvalidConsentID(t *testing.T) {
	env := SetupTestEnvironment(t)

	invalidIDs := []string{
		"",
		"INVALID",
		"123",
		"consent-invalid",
		"CONSENT_12345",
	}

	for _, invalidID := range invalidIDs {
		t.Run("InvalidID_"+invalidID, func(t *testing.T) {
			req, err := http.NewRequest("DELETE", "/api/v1/consents/"+invalidID, nil)
			require.NoError(t, err)
			req.Header.Set("org-id", "TEST_ORG")
			req.Header.Set("client-id", "TEST_CLIENT")

			recorder := httptest.NewRecorder()
			env.Router.ServeHTTP(recorder, req)

			// Should return either 400 (Bad Request) or 404 (Not Found) depending on validation
			assert.True(t, recorder.Code == http.StatusBadRequest || recorder.Code == http.StatusNotFound,
				"Should return 400 or 404 for invalid consent ID: %s, got: %d", invalidID, recorder.Code)

			t.Logf("✓ Correctly rejected invalid consent ID: %s", invalidID)
		})
	}
}
