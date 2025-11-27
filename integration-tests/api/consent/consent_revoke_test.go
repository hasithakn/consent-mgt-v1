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

// TestAPI_RevokeConsent tests PUT /consents/:consentId/revoke
func TestAPI_RevokeConsent(t *testing.T) {
	env := SetupTestEnvironment(t)

	// Create test purpose first
	purposes := CreateTestPurposes(t, env, map[string]string{
		"revoke_test_purpose": "Test purpose for revoke",
	})
	defer CleanupTestPurposes(t, env, purposes)

	// Create a consent
	frequency := 1
	recurringIndicator := false

	createReq := &models.ConsentAPIRequest{
		Type:               "accounts",
		RecurringIndicator: &recurringIndicator,
		Frequency:          &frequency,
		ConsentPurpose: []models.ConsentPurposeItem{
			{Name: "revoke_test_purpose", Value: "Test consent for revoke", IsUserApproved: BoolPtr(true), IsMandatory: BoolPtr(true)},
		},
		Attributes: map[string]string{
			"test": "revoke",
		},
	}

	reqBody, _ := json.Marshal(createReq)

	req, _ := http.NewRequest("POST", "/api/v1/consents", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("org-id", "TEST_ORG")
	req.Header.Set("client-id", "TEST_CLIENT")

	recorder := httptest.NewRecorder()
	env.Router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusCreated, recorder.Code, "Failed to create consent: %s", recorder.Body.String())

	var createResponse models.ConsentAPIResponse
	json.Unmarshal(recorder.Body.Bytes(), &createResponse)

	consentID := createResponse.ID
	assert.NotEmpty(t, consentID)
	defer CleanupTestData(t, env, consentID)

	// Step 2: Revoke the consent
	revokeReq := &models.ConsentRevokeRequest{
		ActionBy:         "admin@wso2.com",
		RevocationReason: "Admin revoke for testing",
	}

	revokeReqBody, err := json.Marshal(revokeReq)
	require.NoError(t, err)

	revokeHTTPReq, err := http.NewRequest("PUT", "/api/v1/consents/"+consentID+"/revoke", bytes.NewBuffer(revokeReqBody))
	require.NoError(t, err)
	revokeHTTPReq.Header.Set("Content-Type", "application/json")
	revokeHTTPReq.Header.Set("org-id", "TEST_ORG")
	revokeHTTPReq.Header.Set("client-id", "TEST_CLIENT")

	revokeRecorder := httptest.NewRecorder()
	env.Router.ServeHTTP(revokeRecorder, revokeHTTPReq)

	// Assert response
	assert.Equal(t, http.StatusOK, revokeRecorder.Code, "Revoke response: %s", revokeRecorder.Body.String())

	var revokeResponse models.ConsentRevokeResponse
	err = json.Unmarshal(revokeRecorder.Body.Bytes(), &revokeResponse)
	require.NoError(t, err)

	// Verify response fields
	assert.Equal(t, "admin@wso2.com", revokeResponse.ActionBy)
	assert.Equal(t, "Admin revoke for testing", revokeResponse.RevocationReason)
	assert.NotZero(t, revokeResponse.ActionTime)

	t.Logf("✓ Successfully revoked consent %s at timestamp %d", consentID, revokeResponse.ActionTime)

	// Step 3: Verify the consent status was updated to REVOKED
	getReq, err := http.NewRequest("GET", "/api/v1/consents/"+consentID, nil)
	require.NoError(t, err)
	getReq.Header.Set("org-id", "TEST_ORG")
	getReq.Header.Set("client-id", "TEST_CLIENT")

	getRecorder := httptest.NewRecorder()
	env.Router.ServeHTTP(getRecorder, getReq)

	assert.Equal(t, http.StatusOK, getRecorder.Code)

	var getResponse models.ConsentAPIResponse
	err = json.Unmarshal(getRecorder.Body.Bytes(), &getResponse)
	require.NoError(t, err)

	assert.Equal(t, "REVOKED", getResponse.Status, "Consent status should be REVOKED")
	t.Logf("✓ Verified consent status is REVOKED")
}

// TestAPI_RevokeConsent_NotFound tests revoking a non-existent consent
func TestAPI_RevokeConsent_NotFound(t *testing.T) {
	env := SetupTestEnvironment(t)

	revokeReq := &models.ConsentRevokeRequest{
		ActionBy:         "admin@wso2.com",
		RevocationReason: "Admin revoke",
	}

	revokeReqBody, _ := json.Marshal(revokeReq)

	req, _ := http.NewRequest("PUT", "/api/v1/consents/NON_EXISTENT_ID/revoke", bytes.NewBuffer(revokeReqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("org-id", "TEST_ORG")
	req.Header.Set("client-id", "TEST_CLIENT")

	recorder := httptest.NewRecorder()
	env.Router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusNotFound, recorder.Code)
	t.Logf("✓ Correctly handled non-existent consent")
}

// TestAPI_RevokeConsent_MissingActionBy tests revoke without actionBy
func TestAPI_RevokeConsent_MissingActionBy(t *testing.T) {
	env := SetupTestEnvironment(t)

	// Create test purpose
	purposes := CreateTestPurposes(t, env, map[string]string{
		"revoke_missing_actionby": "Test purpose",
	})
	defer CleanupTestPurposes(t, env, purposes)

	// Create a consent
	frequency := 1
	recurringIndicator := false

	createReq := &models.ConsentAPIRequest{
		Type:               "accounts",
		RecurringIndicator: &recurringIndicator,
		Frequency:          &frequency,
		ConsentPurpose: []models.ConsentPurposeItem{
			{Name: "revoke_missing_actionby", Value: "Test", IsUserApproved: BoolPtr(true), IsMandatory: BoolPtr(true)},
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

	consentID := createResponse.ID
	defer CleanupTestData(t, env, consentID)

	// Try to revoke without actionBy
	revokeReq := map[string]interface{}{
		"revocationReason": "Test revoke",
	}

	revokeReqBody, _ := json.Marshal(revokeReq)

	revokeHTTPReq, _ := http.NewRequest("PUT", "/api/v1/consents/"+consentID+"/revoke", bytes.NewBuffer(revokeReqBody))
	revokeHTTPReq.Header.Set("Content-Type", "application/json")
	revokeHTTPReq.Header.Set("org-id", "TEST_ORG")
	revokeHTTPReq.Header.Set("client-id", "TEST_CLIENT")

	revokeRecorder := httptest.NewRecorder()
	env.Router.ServeHTTP(revokeRecorder, revokeHTTPReq)

	assert.Equal(t, http.StatusBadRequest, revokeRecorder.Code)
	t.Logf("✓ Correctly rejected revoke request without actionBy")
}

// TestRevokeConsent_UpdatesAuthStatusToSysRevoked tests that revoking a consent
// updates all authorization statuses to SYS_REVOKED
func TestRevokeConsent_UpdatesAuthStatusToSysRevoked(t *testing.T) {
	env := SetupTestEnvironment(t)

	// Create test purpose
	purposes := CreateTestPurposes(t, env, map[string]string{
		"test_purpose": "Test purpose for revoke",
	})
	defer CleanupTestPurposes(t, env, purposes)

	// Create consent with authorizations (all approved to get ACTIVE status)
	createReq := &models.ConsentAPIRequest{
		Type: "accounts",
		ConsentPurpose: []models.ConsentPurposeItem{
			{Name: "test_purpose", Value: "test", IsUserApproved: BoolPtr(true), IsMandatory: BoolPtr(true)},
		},
		Authorizations: []models.AuthorizationAPIRequest{
			{UserID: "user-123", Type: "authorization_code", Status: "approved"},
			{UserID: "user-456", Type: "authorization_code", Status: "approved"},
			{UserID: "user-789", Type: "authorization_code", Status: "approved"},
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

	// Verify initial consent status is ACTIVE (from all approved auths) and auth statuses are approved
	assert.Equal(t, "ACTIVE", createResponse.Status, "Initial consent status should be ACTIVE with all approved auths")
	require.Len(t, createResponse.Authorizations, 3, "Should have 3 authorizations")
	for i, auth := range createResponse.Authorizations {
		assert.Equal(t, "approved", auth.Status, "Authorization %d initial status should be approved", i)
	}

	// Revoke the consent
	revokeReq := &models.ConsentRevokeRequest{
		ActionBy:         "admin@test.com",
		RevocationReason: "User requested revocation",
	}

	revokeBody, _ := json.Marshal(revokeReq)
	revReq, _ := http.NewRequest("PUT", "/api/v1/consents/"+createResponse.ID+"/revoke", bytes.NewBuffer(revokeBody))
	revReq.Header.Set("Content-Type", "application/json")
	revReq.Header.Set("org-id", "TEST_ORG")
	revReq.Header.Set("client-id", "TEST_CLIENT")

	revRecorder := httptest.NewRecorder()
	env.Router.ServeHTTP(revRecorder, revReq)
	require.Equal(t, http.StatusOK, revRecorder.Code)

	var revokeResponse models.ConsentRevokeResponse
	json.Unmarshal(revRecorder.Body.Bytes(), &revokeResponse)

	assert.Equal(t, "admin@test.com", revokeResponse.ActionBy)
	assert.Equal(t, "User requested revocation", revokeResponse.RevocationReason)

	// GET the consent to verify status updates
	getReq, _ := http.NewRequest("GET", "/api/v1/consents/"+createResponse.ID, nil)
	getReq.Header.Set("org-id", "TEST_ORG")
	getReq.Header.Set("client-id", "TEST_CLIENT")

	getRecorder := httptest.NewRecorder()
	env.Router.ServeHTTP(getRecorder, getReq)
	require.Equal(t, http.StatusOK, getRecorder.Code)

	var getResponse models.ConsentAPIResponse
	json.Unmarshal(getRecorder.Body.Bytes(), &getResponse)

	// Verify consent status is REVOKED
	assert.Equal(t, "REVOKED", getResponse.Status, "Consent status should be REVOKED")

	// Verify all authorization statuses are SYS_REVOKED
	require.Len(t, getResponse.Authorizations, 3, "Should have 3 authorizations")
	for i, auth := range getResponse.Authorizations {
		assert.Equal(t, string(models.AuthStateSysRevoked), auth.Status, "Authorization %d status should be SYS_REVOKED", i)
	}

	t.Log("✓ Revoke consent: consent status updated to REVOKED and all auth statuses updated to SYS_REVOKED")
}
