package integration

import (
	"bytes"
	"context"
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
	env := setupAPITestEnvironment(t)
	ctx := context.Background()

	// Create test purpose first
	desc := "Test purpose for revoke"
	purpose := &models.ConsentPurpose{
		ID:          "PURPOSE-revoke-test",
		Name:        "revoke_test_purpose",
		Description: &desc,
		OrgID:       "TEST_ORG",
	}

	err := env.ConsentPurposeDAO.Create(ctx, purpose)
	require.NoError(t, err)
	defer env.ConsentPurposeDAO.Delete(ctx, purpose.ID, "TEST_ORG")

	// Step 1: Create a consent first
	validityTime := int64(7776000)
	frequency := 1
	recurringIndicator := false

	createReq := &models.ConsentAPIRequest{
		Type:               "accounts",
		Status:             "ACTIVE",
		ValidityTime:       &validityTime,
		RecurringIndicator: &recurringIndicator,
		Frequency:          &frequency,
		ConsentPurpose: []models.ConsentPurposeItem{
			{Name: "revoke_test_purpose", Value: "Test consent for revoke"},
		},
		Attributes: map[string]string{
			"test": "revoke",
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

	require.Equal(t, http.StatusCreated, recorder.Code, "Failed to create consent: %s", recorder.Body.String())

	var createResponse models.ConsentAPIResponse
	err = json.Unmarshal(recorder.Body.Bytes(), &createResponse)
	require.NoError(t, err)

	consentID := createResponse.ID
	assert.NotEmpty(t, consentID)
	defer cleanupAPITestData(t, env, consentID)

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
	env := setupAPITestEnvironment(t)

	revokeReq := &models.ConsentRevokeRequest{
		ActionBy:         "admin@wso2.com",
		RevocationReason: "Admin revoke",
	}

	revokeReqBody, err := json.Marshal(revokeReq)
	require.NoError(t, err)

	req, err := http.NewRequest("PUT", "/api/v1/consents/NON_EXISTENT_ID/revoke", bytes.NewBuffer(revokeReqBody))
	require.NoError(t, err)
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
	env := setupAPITestEnvironment(t)
	ctx := context.Background()

	// Create test purpose
	desc := "Test purpose"
	purpose := &models.ConsentPurpose{
		ID:          "PURPOSE-revoke-missing-actionby",
		Name:        "revoke_missing_actionby",
		Description: &desc,
		OrgID:       "TEST_ORG",
	}

	err := env.ConsentPurposeDAO.Create(ctx, purpose)
	require.NoError(t, err)
	defer env.ConsentPurposeDAO.Delete(ctx, purpose.ID, "TEST_ORG")

	// Create a consent
	validityTime := int64(7776000)
	frequency := 1
	recurringIndicator := false

	createReq := &models.ConsentAPIRequest{
		Type:               "accounts",
		Status:             "ACTIVE",
		ValidityTime:       &validityTime,
		RecurringIndicator: &recurringIndicator,
		Frequency:          &frequency,
		ConsentPurpose: []models.ConsentPurposeItem{
			{Name: "revoke_missing_actionby", Value: "Test"},
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

	require.Equal(t, http.StatusCreated, recorder.Code)

	var createResponse models.ConsentAPIResponse
	err = json.Unmarshal(recorder.Body.Bytes(), &createResponse)
	require.NoError(t, err)

	consentID := createResponse.ID
	defer cleanupAPITestData(t, env, consentID)

	// Try to revoke without actionBy
	revokeReq := map[string]interface{}{
		"revocationReason": "Test revoke",
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

	assert.Equal(t, http.StatusBadRequest, revokeRecorder.Code)
	t.Logf("✓ Correctly rejected revoke request without actionBy")
}
