package consent

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wso2/consent-management-api/internal/models"
)

// TestUpdateConsent_Success tests successful update of consent fields
func TestUpdateConsent_Success(t *testing.T) {
	env := SetupTestEnvironment(t)
	
	// Create test purposes
	purposes := CreateTestPurposes(t, env, map[string]string{
		"data_access":    "Data access purpose",
		"payment_access": "Payment access purpose",
	})
	defer CleanupTestPurposes(t, env, purposes)

	// Create a consent first
	createReq := &models.ConsentAPIRequest{
		Type: "accounts",
		ConsentPurpose: []models.ConsentPurposeItem{
			{Name: "data_access", Value: "Initial value", IsUserApproved: BoolPtr(true), IsMandatory: BoolPtr(true)},
		},
		Attributes: map[string]string{
			"channel": "web",
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

	// Update the consent
	validityTime := int64(86400) // 1 day
	updateReq := &models.ConsentAPIRequest{
		Type:         "accounts",
		ValidityTime: &validityTime,
		ConsentPurpose: []models.ConsentPurposeItem{
			{Name: "data_access", Value: "Updated value", IsUserApproved: BoolPtr(false), IsMandatory: BoolPtr(false)},
			{Name: "payment_access", Value: "Payment data", IsUserApproved: BoolPtr(true), IsMandatory: BoolPtr(true)},
		},
		Attributes: map[string]string{
			"channel": "mobile",
			"version": "2.0",
		},
	}

	updateBody, err := json.Marshal(updateReq)
	require.NoError(t, err)

	updateHTTPReq, err := http.NewRequest("PUT", "/api/v1/consents/"+createResp.ID, bytes.NewBuffer(updateBody))
	require.NoError(t, err)
	updateHTTPReq.Header.Set("Content-Type", "application/json")
	updateHTTPReq.Header.Set("org-id", "TEST_ORG")
	updateHTTPReq.Header.Set("client-id", "TEST_CLIENT")

	updateRecorder := httptest.NewRecorder()
	env.Router.ServeHTTP(updateRecorder, updateHTTPReq)

	assert.Equal(t, http.StatusOK, updateRecorder.Code, "Should return 200 OK")

	var updateResp models.ConsentAPIResponse
	err = json.Unmarshal(updateRecorder.Body.Bytes(), &updateResp)
	require.NoError(t, err)

	// Verify updated fields
	assert.Equal(t, createResp.ID, updateResp.ID, "ID should not change")
	assert.Equal(t, "accounts", updateResp.Type, "Type should remain")
	assert.NotNil(t, updateResp.ValidityTime, "ValidityTime should be set")
	assert.Equal(t, validityTime, *updateResp.ValidityTime, "ValidityTime should match")

	// Verify updated purposes
	require.Len(t, updateResp.ConsentPurpose, 2, "Should have 2 consent purposes after update")
	
	purposeMap := make(map[string]models.ConsentPurposeItem)
	for _, cp := range updateResp.ConsentPurpose {
		purposeMap[cp.Name] = cp
	}

	dataAccess, exists := purposeMap["data_access"]
	assert.True(t, exists, "data_access purpose should exist")
	assert.Equal(t, "Updated value", dataAccess.Value, "data_access value should be updated")
	assert.NotNil(t, dataAccess.IsUserApproved, "IsUserApproved should not be nil")
	assert.False(t, *dataAccess.IsUserApproved, "data_access should not be selected after update")
	assert.NotNil(t, dataAccess.IsMandatory, "IsMandatory should not be nil")
	assert.False(t, *dataAccess.IsMandatory, "data_access should not be mandatory after update")

	paymentAccess, exists := purposeMap["payment_access"]
	assert.True(t, exists, "payment_access purpose should exist")
	assert.Equal(t, "Payment data", paymentAccess.Value, "payment_access value should match")
	assert.NotNil(t, paymentAccess.IsUserApproved, "IsUserApproved should not be nil")
	assert.True(t, *paymentAccess.IsUserApproved, "payment_access should be selected")
	assert.NotNil(t, paymentAccess.IsMandatory, "IsMandatory should not be nil")
	assert.True(t, *paymentAccess.IsMandatory, "payment_access should be mandatory")

	// Verify updated attributes
	assert.Equal(t, "mobile", updateResp.Attributes["channel"], "channel should be updated")
	assert.Equal(t, "2.0", updateResp.Attributes["version"], "version should be added")

	// Verify via GET to ensure persistence
	getReq, err := http.NewRequest("GET", "/api/v1/consents/"+createResp.ID, nil)
	require.NoError(t, err)
	getReq.Header.Set("org-id", "TEST_ORG")
	getReq.Header.Set("client-id", "TEST_CLIENT")

	getRecorder := httptest.NewRecorder()
	env.Router.ServeHTTP(getRecorder, getReq)
	assert.Equal(t, http.StatusOK, getRecorder.Code)

	var getResp models.ConsentAPIResponse
	err = json.Unmarshal(getRecorder.Body.Bytes(), &getResp)
	require.NoError(t, err)

	assert.Equal(t, "mobile", getResp.Attributes["channel"], "Updated attribute should persist")
	assert.Len(t, getResp.ConsentPurpose, 2, "Updated purposes should persist")

	t.Logf("✓ Successfully updated consent %s", createResp.ID)
}

// TestUpdateConsent_AddDataAccessValidityDuration tests adding dataAccessValidityDuration to existing consent
func TestUpdateConsent_AddDataAccessValidityDuration(t *testing.T) {
	env := SetupTestEnvironment(t)
	
	purposes := CreateTestPurposes(t, env, map[string]string{
		"data_access": "Data access purpose",
	})
	defer CleanupTestPurposes(t, env, purposes)

	// Create consent without dataAccessValidityDuration
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

	// Verify no dataAccessValidityDuration initially
	assert.Nil(t, createResp.DataAccessValidityDuration, "Should not have dataAccessValidityDuration initially")

	// Update with dataAccessValidityDuration
	duration := int64(86400)
	updateReq := &models.ConsentAPIRequest{
		Type:                       "accounts",
		DataAccessValidityDuration: &duration,
		ConsentPurpose: []models.ConsentPurposeItem{
			{Name: "data_access", Value: "Test", IsUserApproved: BoolPtr(true), IsMandatory: BoolPtr(true)},
		},
	}

	updateBody, err := json.Marshal(updateReq)
	require.NoError(t, err)

	updateHTTPReq, err := http.NewRequest("PUT", "/api/v1/consents/"+createResp.ID, bytes.NewBuffer(updateBody))
	require.NoError(t, err)
	updateHTTPReq.Header.Set("Content-Type", "application/json")
	updateHTTPReq.Header.Set("org-id", "TEST_ORG")
	updateHTTPReq.Header.Set("client-id", "TEST_CLIENT")

	updateRecorder := httptest.NewRecorder()
	env.Router.ServeHTTP(updateRecorder, updateHTTPReq)

	assert.Equal(t, http.StatusOK, updateRecorder.Code)

	var updateResp models.ConsentAPIResponse
	err = json.Unmarshal(updateRecorder.Body.Bytes(), &updateResp)
	require.NoError(t, err)

	// Verify dataAccessValidityDuration was added
	assert.NotNil(t, updateResp.DataAccessValidityDuration, "DataAccessValidityDuration should be set")
	assert.Equal(t, duration, *updateResp.DataAccessValidityDuration, "DataAccessValidityDuration should match")

	t.Logf("✓ Successfully added dataAccessValidityDuration to consent")
}

// TestUpdateConsent_ChangeDataAccessValidityDuration tests changing dataAccessValidityDuration
func TestUpdateConsent_ChangeDataAccessValidityDuration(t *testing.T) {
	env := SetupTestEnvironment(t)
	
	purposes := CreateTestPurposes(t, env, map[string]string{
		"data_access": "Data access purpose",
	})
	defer CleanupTestPurposes(t, env, purposes)

	// Create consent with initial dataAccessValidityDuration
	initialDuration := int64(86400) // 1 day
	createReq := &models.ConsentAPIRequest{
		Type:                       "accounts",
		DataAccessValidityDuration: &initialDuration,
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

	// Verify initial duration
	assert.Equal(t, initialDuration, *createResp.DataAccessValidityDuration)

	// Update with new duration
	newDuration := int64(172800) // 2 days
	updateReq := &models.ConsentAPIRequest{
		Type:                       "accounts",
		DataAccessValidityDuration: &newDuration,
		ConsentPurpose: []models.ConsentPurposeItem{
			{Name: "data_access", Value: "Test", IsUserApproved: BoolPtr(true), IsMandatory: BoolPtr(true)},
		},
	}

	updateBody, err := json.Marshal(updateReq)
	require.NoError(t, err)

	updateHTTPReq, err := http.NewRequest("PUT", "/api/v1/consents/"+createResp.ID, bytes.NewBuffer(updateBody))
	require.NoError(t, err)
	updateHTTPReq.Header.Set("Content-Type", "application/json")
	updateHTTPReq.Header.Set("org-id", "TEST_ORG")
	updateHTTPReq.Header.Set("client-id", "TEST_CLIENT")

	updateRecorder := httptest.NewRecorder()
	env.Router.ServeHTTP(updateRecorder, updateHTTPReq)

	assert.Equal(t, http.StatusOK, updateRecorder.Code)

	var updateResp models.ConsentAPIResponse
	err = json.Unmarshal(updateRecorder.Body.Bytes(), &updateResp)
	require.NoError(t, err)

	// Verify duration was changed
	assert.Equal(t, newDuration, *updateResp.DataAccessValidityDuration, "DataAccessValidityDuration should be updated")

	t.Logf("✓ Successfully changed dataAccessValidityDuration from %d to %d", initialDuration, newDuration)
}

// TestUpdateConsent_NegativeDataAccessValidityDuration tests update with negative duration
func TestUpdateConsent_NegativeDataAccessValidityDuration(t *testing.T) {
	env := SetupTestEnvironment(t)
	
	purposes := CreateTestPurposes(t, env, map[string]string{
		"data_access": "Data access purpose",
	})
	defer CleanupTestPurposes(t, env, purposes)

	// Create consent
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

	// Try to update with negative duration
	negativeDuration := int64(-1000)
	updateReq := &models.ConsentAPIRequest{
		Type:                       "accounts",
		DataAccessValidityDuration: &negativeDuration,
		ConsentPurpose: []models.ConsentPurposeItem{
			{Name: "data_access", Value: "Test", IsUserApproved: BoolPtr(true), IsMandatory: BoolPtr(true)},
		},
	}

	updateBody, err := json.Marshal(updateReq)
	require.NoError(t, err)

	updateHTTPReq, err := http.NewRequest("PUT", "/api/v1/consents/"+createResp.ID, bytes.NewBuffer(updateBody))
	require.NoError(t, err)
	updateHTTPReq.Header.Set("Content-Type", "application/json")
	updateHTTPReq.Header.Set("org-id", "TEST_ORG")
	updateHTTPReq.Header.Set("client-id", "TEST_CLIENT")

	updateRecorder := httptest.NewRecorder()
	env.Router.ServeHTTP(updateRecorder, updateHTTPReq)

	// Should return 400 Bad Request
	assert.Equal(t, http.StatusBadRequest, updateRecorder.Code, "Should reject negative dataAccessValidityDuration")

	t.Logf("✓ Correctly rejected negative dataAccessValidityDuration in update")
}

// TestUpdateConsent_NotFound tests update of non-existent consent
func TestUpdateConsent_NotFound(t *testing.T) {
	env := SetupTestEnvironment(t)
	
	purposes := CreateTestPurposes(t, env, map[string]string{
		"data_access": "Data access purpose",
	})
	defer CleanupTestPurposes(t, env, purposes)

	nonExistentID := "CONSENT-nonexistent-12345"

	updateReq := &models.ConsentAPIRequest{
		Type: "accounts",
		ConsentPurpose: []models.ConsentPurposeItem{
			{Name: "data_access", Value: "Test", IsUserApproved: BoolPtr(true), IsMandatory: BoolPtr(true)},
		},
	}

	updateBody, err := json.Marshal(updateReq)
	require.NoError(t, err)

	updateHTTPReq, err := http.NewRequest("PUT", "/api/v1/consents/"+nonExistentID, bytes.NewBuffer(updateBody))
	require.NoError(t, err)
	updateHTTPReq.Header.Set("Content-Type", "application/json")
	updateHTTPReq.Header.Set("org-id", "TEST_ORG")
	updateHTTPReq.Header.Set("client-id", "TEST_CLIENT")

	updateRecorder := httptest.NewRecorder()
	env.Router.ServeHTTP(updateRecorder, updateHTTPReq)

	// Should return 404 Not Found
	assert.Equal(t, http.StatusNotFound, updateRecorder.Code, "Should return 404 for non-existent consent")

	t.Logf("✓ Correctly returned 404 for update of non-existent consent")
}

// TestUpdateConsent_DifferentOrgID tests that consent from different org cannot be updated
func TestUpdateConsent_DifferentOrgID(t *testing.T) {
	env := SetupTestEnvironment(t)
	
	purposes := CreateTestPurposes(t, env, map[string]string{
		"data_access": "Data access purpose",
	})
	defer CleanupTestPurposes(t, env, purposes)

	// Create consent with TEST_ORG
	createReq := &models.ConsentAPIRequest{
		Type: "accounts",
		ConsentPurpose: []models.ConsentPurposeItem{
			{Name: "data_access", Value: "Initial", IsUserApproved: BoolPtr(true), IsMandatory: BoolPtr(true)},
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

	// Try to update with different org-id
	updateReq := &models.ConsentAPIRequest{
		Type: "accounts",
		ConsentPurpose: []models.ConsentPurposeItem{
			{Name: "data_access", Value: "Updated", IsUserApproved: BoolPtr(true), IsMandatory: BoolPtr(true)},
		},
	}

	updateBody, err := json.Marshal(updateReq)
	require.NoError(t, err)

	updateHTTPReq, err := http.NewRequest("PUT", "/api/v1/consents/"+createResp.ID, bytes.NewBuffer(updateBody))
	require.NoError(t, err)
	updateHTTPReq.Header.Set("Content-Type", "application/json")
	updateHTTPReq.Header.Set("org-id", "DIFFERENT_ORG")
	updateHTTPReq.Header.Set("client-id", "TEST_CLIENT")

	updateRecorder := httptest.NewRecorder()
	env.Router.ServeHTTP(updateRecorder, updateHTTPReq)

	// Should return 404 (consent not found for this org)
	assert.Equal(t, http.StatusNotFound, updateRecorder.Code, "Should return 404 when org doesn't match")

	t.Logf("✓ Correctly prevented update of consent from different org")
}

// TestUpdateConsent_InvalidRequest tests update with invalid/missing required fields
func TestUpdateConsent_InvalidRequest(t *testing.T) {
	env := SetupTestEnvironment(t)
	
	purposes := CreateTestPurposes(t, env, map[string]string{
		"data_access": "Data access purpose",
	})
	defer CleanupTestPurposes(t, env, purposes)

	// Create a consent first
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

	testCases := []struct {
		name        string
		requestBody string
		expectedMsg string
	}{
		{
			name:        "Empty consentPurpose array",
			requestBody: `{"type": "accounts", "consentPurpose": []}`,
			expectedMsg: "consentPurpose cannot be empty",
		},
		{
			name:        "Invalid JSON",
			requestBody: `{invalid json}`,
			expectedMsg: "invalid JSON",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			updateHTTPReq, err := http.NewRequest("PUT", "/api/v1/consents/"+createResp.ID, bytes.NewBufferString(tc.requestBody))
			require.NoError(t, err)
			updateHTTPReq.Header.Set("Content-Type", "application/json")
			updateHTTPReq.Header.Set("org-id", "TEST_ORG")
			updateHTTPReq.Header.Set("client-id", "TEST_CLIENT")

			updateRecorder := httptest.NewRecorder()
			env.Router.ServeHTTP(updateRecorder, updateHTTPReq)

			assert.Equal(t, http.StatusBadRequest, updateRecorder.Code, "Should return 400 Bad Request")
			t.Logf("✓ Correctly rejected update with %s", tc.name)
		})
	}
}

// TestUpdateConsent_WithNonExistentPurpose tests updating consent with a purpose that doesn't exist
func TestUpdateConsent_WithNonExistentPurpose(t *testing.T) {
	env := SetupTestEnvironment(t)
	
	purposes := CreateTestPurposes(t, env, map[string]string{
		"data_access": "Data access purpose",
	})
	defer CleanupTestPurposes(t, env, purposes)

	// Create consent
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

	// Try to update with multiple purposes where one doesn't exist
	updateReq := &models.ConsentAPIRequest{
		Type: "accounts",
		ConsentPurpose: []models.ConsentPurposeItem{
			{Name: "data_access", Value: "Valid purpose", IsUserApproved: BoolPtr(true), IsMandatory: BoolPtr(true)},
			{Name: "nonexistent_purpose", Value: "This doesn't exist", IsUserApproved: BoolPtr(true), IsMandatory: BoolPtr(true)},
		},
	}

	updateBody, err := json.Marshal(updateReq)
	require.NoError(t, err)

	updateHTTPReq, err := http.NewRequest("PUT", "/api/v1/consents/"+createResp.ID, bytes.NewBuffer(updateBody))
	require.NoError(t, err)
	updateHTTPReq.Header.Set("Content-Type", "application/json")
	updateHTTPReq.Header.Set("org-id", "TEST_ORG")
	updateHTTPReq.Header.Set("client-id", "TEST_CLIENT")

	updateRecorder := httptest.NewRecorder()
	env.Router.ServeHTTP(updateRecorder, updateHTTPReq)

	// Should return 400 Bad Request for non-existent purpose
	assert.Equal(t, http.StatusBadRequest, updateRecorder.Code, "Should reject update with non-existent purpose")

	t.Logf("✓ Correctly rejected update with non-existent purpose")
}

// TestUpdateConsent_UpdateAuthResourceAndCheckStatus tests updating authorization and verifying consent status changes
func TestUpdateConsent_UpdateAuthResourceAndCheckStatus(t *testing.T) {
	env := SetupTestEnvironment(t)
	
	purposes := CreateTestPurposes(t, env, map[string]string{
		"data_access": "Data access purpose",
	})
	defer CleanupTestPurposes(t, env, purposes)

	// Create consent with APPROVED authorization
	createReq := &models.ConsentAPIRequest{
		Type: "accounts",
		ConsentPurpose: []models.ConsentPurposeItem{
			{Name: "data_access", Value: "Test", IsUserApproved: BoolPtr(true), IsMandatory: BoolPtr(true)},
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

	// Verify initial status is ACTIVE (APPROVED auth -> ACTIVE consent)
	assert.Equal(t, "ACTIVE", createResp.Status, "Initial consent status should be ACTIVE")

	// Update consent with authorization changed to REJECTED via PUT /consents
	require.Len(t, createResp.Authorizations, 1, "Should have 1 authorization")

	updateReq := &models.ConsentAPIRequest{
		Type: "accounts",
		ConsentPurpose: []models.ConsentPurposeItem{
			{Name: "data_access", Value: "Test", IsUserApproved: BoolPtr(true), IsMandatory: BoolPtr(true)},
		},
		Authorizations: []models.AuthorizationAPIRequest{
			{
				UserID: "user-123",
				Type:   "authorization_code",
				Status: "REJECTED",
			},
		},
	}

	updateBody, err := json.Marshal(updateReq)
	require.NoError(t, err)

	updateHTTPReq, err := http.NewRequest("PUT", "/api/v1/consents/"+createResp.ID, bytes.NewBuffer(updateBody))
	require.NoError(t, err)
	updateHTTPReq.Header.Set("Content-Type", "application/json")
	updateHTTPReq.Header.Set("org-id", "TEST_ORG")
	updateHTTPReq.Header.Set("client-id", "TEST_CLIENT")

	updateRecorder := httptest.NewRecorder()
	env.Router.ServeHTTP(updateRecorder, updateHTTPReq)

	assert.Equal(t, http.StatusOK, updateRecorder.Code, "Consent update should succeed")

	var updateResp models.ConsentAPIResponse
	err = json.Unmarshal(updateRecorder.Body.Bytes(), &updateResp)
	require.NoError(t, err)

	// Consent status should now be REJECTED
	assert.Equal(t, "REJECTED", updateResp.Status, "Consent status should change to REJECTED when authorization is rejected")
	assert.Len(t, updateResp.Authorizations, 1, "Should still have 1 authorization")
	assert.Equal(t, "REJECTED", updateResp.Authorizations[0].Status, "Authorization status should be REJECTED")

	// GET the consent and verify status persists
	getReq, err := http.NewRequest("GET", "/api/v1/consents/"+createResp.ID, nil)
	require.NoError(t, err)
	getReq.Header.Set("org-id", "TEST_ORG")
	getReq.Header.Set("client-id", "TEST_CLIENT")

	getRecorder := httptest.NewRecorder()
	env.Router.ServeHTTP(getRecorder, getReq)
	assert.Equal(t, http.StatusOK, getRecorder.Code)

	var getResp models.ConsentAPIResponse
	err = json.Unmarshal(getRecorder.Body.Bytes(), &getResp)
	require.NoError(t, err)

	// Consent status should persist as REJECTED
	assert.Equal(t, "REJECTED", getResp.Status, "Consent status should remain REJECTED")
	assert.Len(t, getResp.Authorizations, 1, "Should still have 1 authorization")
	assert.Equal(t, "REJECTED", getResp.Authorizations[0].Status, "Authorization status should remain REJECTED")

	t.Logf("✓ Consent status correctly updated from APPROVED to REJECTED via PUT /consents")
}

// TestUpdateConsent_RemovePurposes tests updating consent to remove purposes
func TestUpdateConsent_RemovePurposes(t *testing.T) {
	env := SetupTestEnvironment(t)
	
	purposes := CreateTestPurposes(t, env, map[string]string{
		"data_access":    "Data access purpose",
		"payment_access": "Payment access purpose",
		"profile_access": "Profile access purpose",
	})
	defer CleanupTestPurposes(t, env, purposes)

	// Create consent with 3 purposes
	createReq := &models.ConsentAPIRequest{
		Type: "accounts",
		ConsentPurpose: []models.ConsentPurposeItem{
			{Name: "data_access", Value: "Data", IsUserApproved: BoolPtr(true), IsMandatory: BoolPtr(true)},
			{Name: "payment_access", Value: "Payment", IsUserApproved: BoolPtr(true), IsMandatory: BoolPtr(true)},
			{Name: "profile_access", Value: "Profile", IsUserApproved: BoolPtr(false), IsMandatory: BoolPtr(false)},
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

	// Verify 3 purposes
	assert.Len(t, createResp.ConsentPurpose, 3, "Should have 3 purposes initially")

	// Update to keep only 1 purpose
	updateReq := &models.ConsentAPIRequest{
		Type: "accounts",
		ConsentPurpose: []models.ConsentPurposeItem{
			{Name: "data_access", Value: "Data only", IsUserApproved: BoolPtr(true), IsMandatory: BoolPtr(true)},
		},
	}

	updateBody, err := json.Marshal(updateReq)
	require.NoError(t, err)

	updateHTTPReq, err := http.NewRequest("PUT", "/api/v1/consents/"+createResp.ID, bytes.NewBuffer(updateBody))
	require.NoError(t, err)
	updateHTTPReq.Header.Set("Content-Type", "application/json")
	updateHTTPReq.Header.Set("org-id", "TEST_ORG")
	updateHTTPReq.Header.Set("client-id", "TEST_CLIENT")

	updateRecorder := httptest.NewRecorder()
	env.Router.ServeHTTP(updateRecorder, updateHTTPReq)

	assert.Equal(t, http.StatusOK, updateRecorder.Code)

	var updateResp models.ConsentAPIResponse
	err = json.Unmarshal(updateRecorder.Body.Bytes(), &updateResp)
	require.NoError(t, err)

	// Verify only 1 purpose remains
	assert.Len(t, updateResp.ConsentPurpose, 1, "Should have only 1 purpose after update")
	assert.Equal(t, "data_access", updateResp.ConsentPurpose[0].Name, "Remaining purpose should be data_access")
	assert.Equal(t, "Data only", updateResp.ConsentPurpose[0].Value, "Purpose value should be updated")

	t.Logf("✓ Successfully removed purposes (3 → 1)")
}

// TestUpdateConsent_RemoveAuthResourcesAndCheckStatus tests removing authorization resources and checking consent status
func TestUpdateConsent_RemoveAuthResourcesAndCheckStatus(t *testing.T) {
	env := SetupTestEnvironment(t)
	
	purposes := CreateTestPurposes(t, env, map[string]string{
		"data_access": "Data access purpose",
	})
	defer CleanupTestPurposes(t, env, purposes)

	// Create consent with 2 authorizations
	createReq := &models.ConsentAPIRequest{
		Type: "accounts",
		ConsentPurpose: []models.ConsentPurposeItem{
			{Name: "data_access", Value: "Test", IsUserApproved: BoolPtr(true), IsMandatory: BoolPtr(true)},
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

	// Verify initial state
	assert.Len(t, createResp.Authorizations, 2, "Should have 2 authorizations initially")
	assert.Equal(t, "ACTIVE", createResp.Status, "Initial consent status should be ACTIVE")

	// Update consent without authorizations (remove them)
	updateReq := &models.ConsentAPIRequest{
		Type: "accounts",
		ConsentPurpose: []models.ConsentPurposeItem{
			{Name: "data_access", Value: "Test", IsUserApproved: BoolPtr(true), IsMandatory: BoolPtr(true)},
		},
		Authorizations: []models.AuthorizationAPIRequest{}, // Empty array to remove authorizations
	}

	updateBody, err := json.Marshal(updateReq)
	require.NoError(t, err)

	updateHTTPReq, err := http.NewRequest("PUT", "/api/v1/consents/"+createResp.ID, bytes.NewBuffer(updateBody))
	require.NoError(t, err)
	updateHTTPReq.Header.Set("Content-Type", "application/json")
	updateHTTPReq.Header.Set("org-id", "TEST_ORG")
	updateHTTPReq.Header.Set("client-id", "TEST_CLIENT")

	updateRecorder := httptest.NewRecorder()
	env.Router.ServeHTTP(updateRecorder, updateHTTPReq)

	assert.Equal(t, http.StatusOK, updateRecorder.Code)

	var updateResp models.ConsentAPIResponse
	err = json.Unmarshal(updateRecorder.Body.Bytes(), &updateResp)
	require.NoError(t, err)

	// Verify authorizations are removed and status changed to CREATED
	assert.Empty(t, updateResp.Authorizations, "Authorizations should be removed")
	assert.Equal(t, "CREATED", updateResp.Status, "Consent status should revert to CREATED when authorizations are removed")

	t.Logf("✓ Successfully removed authorizations and consent status changed from APPROVED to CREATED")
}

// TestUpdateConsent_AddMultipleAuthResourcesWithMixedStatuses tests adding multiple auth resources with different statuses
func TestUpdateConsent_AddMultipleAuthResourcesWithMixedStatuses(t *testing.T) {
	env := SetupTestEnvironment(t)
	
	purposes := CreateTestPurposes(t, env, map[string]string{
		"data_access": "Data access purpose",
	})
	defer CleanupTestPurposes(t, env, purposes)

	// Create consent without authorizations
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

	// Verify initial status is CREATED (no authorizations)
	assert.Equal(t, "CREATED", createResp.Status, "Initial consent status should be CREATED")
	assert.Empty(t, createResp.Authorizations, "Should have no authorizations initially")

	// Update consent with multiple authorizations with mixed statuses
	updateReq := &models.ConsentAPIRequest{
		Type: "accounts",
		ConsentPurpose: []models.ConsentPurposeItem{
			{Name: "data_access", Value: "Test", IsUserApproved: BoolPtr(true), IsMandatory: BoolPtr(true)},
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
			{
				UserID: "user-003",
				Type:   "authorization_code",
				Status: "ACTIVE",
			},
		},
	}

	updateBody, err := json.Marshal(updateReq)
	require.NoError(t, err)

	updateHTTPReq, err := http.NewRequest("PUT", "/api/v1/consents/"+createResp.ID, bytes.NewBuffer(updateBody))
	require.NoError(t, err)
	updateHTTPReq.Header.Set("Content-Type", "application/json")
	updateHTTPReq.Header.Set("org-id", "TEST_ORG")
	updateHTTPReq.Header.Set("client-id", "TEST_CLIENT")

	updateRecorder := httptest.NewRecorder()
	env.Router.ServeHTTP(updateRecorder, updateHTTPReq)

	assert.Equal(t, http.StatusOK, updateRecorder.Code)

	var updateResp models.ConsentAPIResponse
	err = json.Unmarshal(updateRecorder.Body.Bytes(), &updateResp)
	require.NoError(t, err)

	// Verify authorizations were added
	assert.Len(t, updateResp.Authorizations, 3, "Should have 3 authorizations after update")

	// Collect all statuses
	authStatuses := make(map[string]int)
	for _, auth := range updateResp.Authorizations {
		authStatuses[auth.Status]++
	}

	// Verify all expected statuses are present
	assert.Equal(t, 1, authStatuses["APPROVED"], "Should have 1 APPROVED authorization")
	assert.Equal(t, 1, authStatuses["REJECTED"], "Should have 1 REJECTED authorization")
	assert.Equal(t, 1, authStatuses["ACTIVE"], "Should have 1 ACTIVE authorization")

	// Consent status should reflect the "priority" of authorization statuses
	// Typically: REJECTED > APPROVED > ACTIVE > CREATED
	// So if there's a REJECTED, consent should be REJECTED
	assert.Equal(t, "REJECTED", updateResp.Status, "Consent status should be REJECTED when there's at least one REJECTED authorization")

	t.Logf("✓ Successfully added 3 authorizations with mixed statuses (APPROVED, REJECTED, ACTIVE)")
	t.Logf("✓ Consent status correctly set to REJECTED (highest priority)")
}

// TestUpdateConsent_ChangeAuthStatusFromApprovedToRejected tests status change when authorization is rejected
func TestUpdateConsent_ChangeAuthStatusFromApprovedToRejected(t *testing.T) {
	env := SetupTestEnvironment(t)
	
	purposes := CreateTestPurposes(t, env, map[string]string{
		"data_access": "Data access purpose",
	})
	defer CleanupTestPurposes(t, env, purposes)

	// Create consent with APPROVED authorization
	createReq := &models.ConsentAPIRequest{
		Type: "accounts",
		ConsentPurpose: []models.ConsentPurposeItem{
			{Name: "data_access", Value: "Test", IsUserApproved: BoolPtr(true), IsMandatory: BoolPtr(true)},
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

	// Verify initial status
	require.Len(t, createResp.Authorizations, 1)
	assert.Equal(t, "ACTIVE", createResp.Status, "Initial status should be ACTIVE (APPROVED auth -> ACTIVE consent)")

	// Update consent with authorization changed to REJECTED via PUT /consents
	// Note: Use "rejected" as authorization status, not "REVOKED" (REVOKED is a consent status, not auth status)
	updateReq := &models.ConsentAPIRequest{
		Type: "accounts",
		ConsentPurpose: []models.ConsentPurposeItem{
			{Name: "data_access", Value: "Test", IsUserApproved: BoolPtr(true), IsMandatory: BoolPtr(true)},
		},
		Authorizations: []models.AuthorizationAPIRequest{
			{
				UserID: "user-123",
				Type:   "authorization_code",
				Status: "rejected", // Valid auth status (approved, rejected, created)
			},
		},
	}

	updateBody, err := json.Marshal(updateReq)
	require.NoError(t, err)

	updateHTTPReq, err := http.NewRequest("PUT", "/api/v1/consents/"+createResp.ID, bytes.NewBuffer(updateBody))
	require.NoError(t, err)
	updateHTTPReq.Header.Set("Content-Type", "application/json")
	updateHTTPReq.Header.Set("org-id", "TEST_ORG")
	updateHTTPReq.Header.Set("client-id", "TEST_CLIENT")

	updateRecorder := httptest.NewRecorder()
	env.Router.ServeHTTP(updateRecorder, updateHTTPReq)

	assert.Equal(t, http.StatusOK, updateRecorder.Code)

	var updateResp models.ConsentAPIResponse
	err = json.Unmarshal(updateRecorder.Body.Bytes(), &updateResp)
	require.NoError(t, err)

	// Verify status changed to REJECTED (rejected auth -> REJECTED consent)
	assert.Equal(t, "REJECTED", updateResp.Status, "Consent status should be REJECTED when authorization is rejected")

	// GET consent and verify status persists
	getReq, err := http.NewRequest("GET", "/api/v1/consents/"+createResp.ID, nil)
	require.NoError(t, err)
	getReq.Header.Set("org-id", "TEST_ORG")
	getReq.Header.Set("client-id", "TEST_CLIENT")

	getRecorder := httptest.NewRecorder()
	env.Router.ServeHTTP(getRecorder, getReq)

	var getResp models.ConsentAPIResponse
	err = json.Unmarshal(getRecorder.Body.Bytes(), &getResp)
	require.NoError(t, err)

	assert.Equal(t, "REJECTED", getResp.Status, "Consent status should remain REJECTED")

	t.Logf("✓ Consent status correctly changed from APPROVED to REJECTED via PUT /consents")
}

// TestUpdateConsent_DuplicatePurposeNames tests that consent update rejects duplicate purpose names
func TestUpdateConsent_DuplicatePurposeNames(t *testing.T) {
	env := SetupTestEnvironment(t)
	
	// Create test purposes
	purposes := CreateTestPurposes(t, env, map[string]string{
		"data_access":    "Data access purpose",
		"payment_access": "Payment access purpose",
		"account_info":   "Account info purpose",
	})
	defer CleanupTestPurposes(t, env, purposes)

	// Create a consent first
	createReq := &models.ConsentAPIRequest{
		Type: "accounts",
		ConsentPurpose: []models.ConsentPurposeItem{
			{Name: "data_access", Value: "Initial value", IsUserApproved: BoolPtr(true), IsMandatory: BoolPtr(true)},
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

	// Attempt to update with duplicate purpose names
	updateReq := &models.ConsentAPIUpdateRequest{
		ConsentPurpose: []models.ConsentPurposeItem{
			{Name: "data_access", Value: "Updated value", IsUserApproved: BoolPtr(true), IsMandatory: BoolPtr(true)},
			{Name: "payment_access", Value: "Payment info", IsUserApproved: BoolPtr(true), IsMandatory: BoolPtr(true)},
			{Name: "payment_access", Value: "Duplicate payment", IsUserApproved: BoolPtr(false), IsMandatory: BoolPtr(false)}, // Duplicate!
		},
	}

	updateBody, err := json.Marshal(updateReq)
	require.NoError(t, err)

	updateHTTPReq, err := http.NewRequest("PUT", "/api/v1/consents/"+consentID, bytes.NewBuffer(updateBody))
	require.NoError(t, err)
	updateHTTPReq.Header.Set("Content-Type", "application/json")
	updateHTTPReq.Header.Set("org-id", "TEST_ORG")
	updateHTTPReq.Header.Set("client-id", "TEST_CLIENT")

	updateRecorder := httptest.NewRecorder()
	env.Router.ServeHTTP(updateRecorder, updateHTTPReq)

	// Assert error response
	assert.Equal(t, http.StatusBadRequest, updateRecorder.Code, "Expected 400 Bad Request for duplicate purpose names")

	var errorResponse map[string]interface{}
	err = json.Unmarshal(updateRecorder.Body.Bytes(), &errorResponse)
	require.NoError(t, err)

	// Verify error mentions duplicate purpose
	errorMsg := errorResponse["message"].(string)
	var errorText string
	if errorMsg != "" {
		errorText = errorMsg
	}
	if details, ok := errorResponse["details"].(string); ok {
		errorText += " " + details
	}
	
	assert.Contains(t, errorText, "duplicate", "Error should mention duplicate")
	assert.Contains(t, errorText, "payment_access", "Error should mention the duplicate purpose name")

	t.Log("✓ Correctly rejected duplicate purpose names in update request")
}

// TestUpdateConsent_IsSelectedDefaultsToTrue tests that isUserApproved defaults to true when not provided in update
func TestUpdateConsent_IsSelectedDefaultsToTrue(t *testing.T) {
	env := SetupTestEnvironment(t)
	
	// Create test purposes
	purposes := CreateTestPurposes(t, env, map[string]string{
		"data_access":    "Data access purpose",
		"payment_access": "Payment access purpose",
	})
	defer CleanupTestPurposes(t, env, purposes)

	// Create a consent first
	createReq := &models.ConsentAPIRequest{
		Type: "accounts",
		ConsentPurpose: []models.ConsentPurposeItem{
			{Name: "data_access", Value: "Initial value", IsUserApproved: BoolPtr(true), IsMandatory: BoolPtr(true)},
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

	// Update without providing isUserApproved
	updateReq := &models.ConsentAPIUpdateRequest{
		ConsentPurpose: []models.ConsentPurposeItem{
			{Name: "data_access", Value: "Updated value"}, // isUserApproved not provided
			{Name: "payment_access", Value: "Payment info", IsUserApproved: BoolPtr(false), IsMandatory: BoolPtr(false)}, // explicitly false
		},
	}

	updateBody, err := json.Marshal(updateReq)
	require.NoError(t, err)

	updateHTTPReq, err := http.NewRequest("PUT", "/api/v1/consents/"+consentID, bytes.NewBuffer(updateBody))
	require.NoError(t, err)
	updateHTTPReq.Header.Set("Content-Type", "application/json")
	updateHTTPReq.Header.Set("org-id", "TEST_ORG")
	updateHTTPReq.Header.Set("client-id", "TEST_CLIENT")

	updateRecorder := httptest.NewRecorder()
	env.Router.ServeHTTP(updateRecorder, updateHTTPReq)

	// Assert successful update
	assert.Equal(t, http.StatusOK, updateRecorder.Code, "Expected 200 OK for update")

	var updateResp models.ConsentAPIResponse
	err = json.Unmarshal(updateRecorder.Body.Bytes(), &updateResp)
	require.NoError(t, err)

	// Verify consent purposes - first one should default to true, second should be false
	require.Len(t, updateResp.ConsentPurpose, 2, "Should have 2 consent purposes")
	
	for _, cp := range updateResp.ConsentPurpose {
		if cp.Name == "data_access" {
			require.NotNil(t, cp.IsUserApproved, "IsUserApproved should not be nil")
			assert.True(t, *cp.IsUserApproved, "IsUserApproved should default to true when not provided")
		} else if cp.Name == "payment_access" {
			require.NotNil(t, cp.IsUserApproved, "IsUserApproved should not be nil")
			assert.False(t, *cp.IsUserApproved, "IsUserApproved should be false when explicitly set")
		}
	}

	t.Log("✓ isUserApproved correctly defaults to true when not provided in update request")
}

// TestUpdateConsent_ExpiryCheck tests that expired consents get EXPIRED status during update
func TestUpdateConsent_ExpiryCheck(t *testing.T) {
	env := SetupTestEnvironment(t)
	
	// Create test purposes
	purposes := CreateTestPurposes(t, env, map[string]string{
		"data_access": "Data access purpose",
	})
	defer CleanupTestPurposes(t, env, purposes)

	// Create a consent with an expired validity time (in the past)
	expiredValidityTime := int64(1000) // Very old timestamp (1970)
	createReq := &models.ConsentAPIRequest{
		Type:         "accounts",
		ValidityTime: &expiredValidityTime,
		ConsentPurpose: []models.ConsentPurposeItem{
			{Name: "data_access", Value: "Initial value", IsUserApproved: BoolPtr(true), IsMandatory: BoolPtr(true)},
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

	// Update the consent (should detect expiry and set EXPIRED status)
	updateReq := &models.ConsentAPIUpdateRequest{
		ConsentPurpose: []models.ConsentPurposeItem{
			{Name: "data_access", Value: "Updated value", IsUserApproved: BoolPtr(true), IsMandatory: BoolPtr(true)},
		},
		Authorizations: []models.AuthorizationAPIRequest{
			{
				Type:   "authorization",
				Status: "APPROVED", // Even though auth is APPROVED, consent should be EXPIRED
			},
		},
	}

	updateBody, err := json.Marshal(updateReq)
	require.NoError(t, err)

	updateHTTPReq, err := http.NewRequest("PUT", "/api/v1/consents/"+consentID, bytes.NewBuffer(updateBody))
	require.NoError(t, err)
	updateHTTPReq.Header.Set("Content-Type", "application/json")
	updateHTTPReq.Header.Set("org-id", "TEST_ORG")
	updateHTTPReq.Header.Set("client-id", "TEST_CLIENT")

	updateRecorder := httptest.NewRecorder()
	env.Router.ServeHTTP(updateRecorder, updateHTTPReq)

	// Assert successful update
	assert.Equal(t, http.StatusOK, updateRecorder.Code, "Expected 200 OK for update")

	var updateResp models.ConsentAPIResponse
	err = json.Unmarshal(updateRecorder.Body.Bytes(), &updateResp)
	require.NoError(t, err)

	// Verify status is EXPIRED (not APPROVED from authorization)
	assert.Equal(t, "EXPIRED", updateResp.Status, "Consent status should be EXPIRED due to expired validityTime")

	t.Log("✓ Expired consent correctly gets EXPIRED status during update")
}

// TestUpdateConsent_CustomAuthStatusPreservesCreatedStatus tests that adding a custom authorization status
// preserves the CREATED consent status (doesn't change to ACTIVE)
func TestUpdateConsent_CustomAuthStatusPreservesCreatedStatus(t *testing.T) {
	env := SetupTestEnvironment(t)
	
	// Create test purpose
	purposes := CreateTestPurposes(t, env, map[string]string{
		"test_purpose": "Test purpose for custom auth status",
	})
	defer CleanupTestPurposes(t, env, purposes)

	// Create consent WITHOUT authorization (status should be CREATED)
	validityTime := time.Now().Add(90 * 24 * time.Hour).UnixMilli()
	createReq := &models.ConsentAPIRequest{
		Type:         "accounts",
		ValidityTime: &validityTime,
		ConsentPurpose: []models.ConsentPurposeItem{
			{Name: "test_purpose", Value: "Test value", IsUserApproved: BoolPtr(true), IsMandatory: BoolPtr(true)},
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
	
	assert.Equal(t, "CREATED", createResponse.Status, "Initial consent status should be CREATED")
	t.Logf("✓ Created consent with CREATED status: %s", createResponse.ID)

	// Update consent to add custom authorization status
	updateReq := &models.ConsentAPIRequest{
		Type:         "accounts",
		ValidityTime: &validityTime,
		ConsentPurpose: []models.ConsentPurposeItem{
			{Name: "test_purpose", Value: "Test value", IsUserApproved: BoolPtr(true), IsMandatory: BoolPtr(true)},
		},
		Authorizations: []models.AuthorizationAPIRequest{
			{
				UserID: "user-456",
				Type:   "authorization_code",
				Status: "custom_pending", // Custom status (not approved/rejected/created)
			},
		},
	}

	updateBody, _ := json.Marshal(updateReq)
	updateHttpReq, _ := http.NewRequest("PUT", "/api/v1/consents/"+createResponse.ID, bytes.NewBuffer(updateBody))
	updateHttpReq.Header.Set("Content-Type", "application/json")
	updateHttpReq.Header.Set("org-id", "TEST_ORG")
	updateHttpReq.Header.Set("client-id", "TEST_CLIENT")

	updateRecorder := httptest.NewRecorder()
	env.Router.ServeHTTP(updateRecorder, updateHttpReq)
	
	assert.Equal(t, http.StatusOK, updateRecorder.Code)
	
	var updateResponse models.ConsentAPIResponse
	json.Unmarshal(updateRecorder.Body.Bytes(), &updateResponse)
	
	// Verify status is still CREATED (not changed to ACTIVE)
	assert.Equal(t, "CREATED", updateResponse.Status, "Consent status should remain CREATED when custom auth status is added")
	assert.Len(t, updateResponse.Authorizations, 1, "Should have 1 authorization")
	assert.Equal(t, "custom_pending", updateResponse.Authorizations[0].Status, "Authorization status should be custom_pending")
	
	t.Logf("✓ Consent status remains CREATED after adding custom authorization status")
	t.Logf("✓ Custom authorization status preserved existing consent status (bug fix verified)")
}

// TestUpdateConsent_CustomAuthStatusPreservesActiveStatus tests that adding a custom authorization status
// to an ACTIVE consent preserves the ACTIVE status
func TestUpdateConsent_CustomAuthStatusPreservesActiveStatus(t *testing.T) {
	env := SetupTestEnvironment(t)
	
	// Create test purpose
	purposes := CreateTestPurposes(t, env, map[string]string{
		"test_purpose": "Test purpose for custom auth status",
	})
	defer CleanupTestPurposes(t, env, purposes)

	// Create consent with approved authorization (status should be ACTIVE)
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
	
	assert.Equal(t, "ACTIVE", createResponse.Status, "Initial consent status should be ACTIVE (has approved auth)")
	t.Logf("✓ Created consent with ACTIVE status: %s", createResponse.ID)

	// Update consent to change authorization to custom status
	updateReq := &models.ConsentAPIRequest{
		Type:         "accounts",
		ValidityTime: &validityTime,
		ConsentPurpose: []models.ConsentPurposeItem{
			{Name: "test_purpose", Value: "Test value", IsUserApproved: BoolPtr(true), IsMandatory: BoolPtr(true)},
		},
		Authorizations: []models.AuthorizationAPIRequest{
			{
				UserID: "user-123",
				Type:   "authorization_code",
				Status: "custom_processing", // Custom status (not approved/rejected/created)
			},
		},
	}

	updateBody, _ := json.Marshal(updateReq)
	updateHttpReq, _ := http.NewRequest("PUT", "/api/v1/consents/"+createResponse.ID, bytes.NewBuffer(updateBody))
	updateHttpReq.Header.Set("Content-Type", "application/json")
	updateHttpReq.Header.Set("org-id", "TEST_ORG")
	updateHttpReq.Header.Set("client-id", "TEST_CLIENT")

	updateRecorder := httptest.NewRecorder()
	env.Router.ServeHTTP(updateRecorder, updateHttpReq)
	
	assert.Equal(t, http.StatusOK, updateRecorder.Code)
	
	var updateResponse models.ConsentAPIResponse
	json.Unmarshal(updateRecorder.Body.Bytes(), &updateResponse)
	
	// Verify status is still ACTIVE (not changed to CREATED or other)
	assert.Equal(t, "ACTIVE", updateResponse.Status, "Consent status should remain ACTIVE when changing to custom auth status")
	assert.Len(t, updateResponse.Authorizations, 1, "Should have 1 authorization")
	assert.Equal(t, "custom_processing", updateResponse.Authorizations[0].Status, "Authorization status should be custom_processing")
	
	t.Logf("✓ Consent status remains ACTIVE after changing to custom authorization status")
	t.Logf("✓ Custom authorization status preserved existing consent status (bug fix verified)")
}
