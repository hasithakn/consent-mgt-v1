package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wso2/consent-management-api/internal/models"
)

func TestAPI_CreateAuthResource(t *testing.T) {
	env := setupAPITestEnvironment(t)
	defer func() {}()

	ctx := context.Background()
	
	// Create test purpose first
	desc := "Auth resource API test - create"
	purpose := &models.ConsentPurpose{
		ID:          "PURPOSE-auth-create",
		Name:        "auth_create_test",
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
		Status:             "awaitingAuthorization",
		ValidityTime:       &validityTime,
		RecurringIndicator: &recurringIndicator,
		Frequency:          &frequency,
		ConsentPurpose: []models.ConsentPurposeItem{
			{
				Name:  "auth_create_test",
				Value: "Test consent for authorization",
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

	require.Equal(t, http.StatusCreated, recorder.Code)

	var consentResponse models.ConsentAPIResponse
	err = json.Unmarshal(recorder.Body.Bytes(), &consentResponse)
	require.NoError(t, err)
	require.NotEmpty(t, consentResponse.ID)

	// Step 2: Create an authorization resource for the consent
	authReq := &models.AuthorizationAPIRequest{
		UserID: "user123",
		Type:   "account",
		Status: "authorized",
		ApprovedPurposeDetails: &models.ApprovedPurposeDetails{
			ApprovedPurposesNames:       []string{"utility_read", "taxes_read"},
			ApprovedAdditionalResources: []interface{}{},
		},
	}

	authReqBody, err := json.Marshal(authReq)
	require.NoError(t, err)

	authReqHTTP, err := http.NewRequest("POST", "/api/v1/consents/"+consentResponse.ID+"/authorizations", bytes.NewBuffer(authReqBody))
	require.NoError(t, err)
	authReqHTTP.Header.Set("Content-Type", "application/json")
	authReqHTTP.Header.Set("org-id", "TEST_ORG")

	authRecorder := httptest.NewRecorder()
	env.Router.ServeHTTP(authRecorder, authReqHTTP)

	// Assert response
	if authRecorder.Code != http.StatusCreated {
		t.Logf("Response Status: %d", authRecorder.Code)
		t.Logf("Response Body: %s", authRecorder.Body.String())
	}
	assert.Equal(t, http.StatusCreated, authRecorder.Code, "Expected 201 Created status")

	var authResponse models.AuthorizationAPIResponse
	err = json.Unmarshal(authRecorder.Body.Bytes(), &authResponse)
	require.NoError(t, err)

	// Verify response data
	assert.NotEmpty(t, authResponse.ID, "Authorization ID should not be empty")
	assert.NotNil(t, authResponse.UserID)
	assert.Equal(t, "user123", *authResponse.UserID)
	assert.Equal(t, "account", authResponse.Type)
	assert.Equal(t, "authorized", authResponse.Status)
	assert.NotNil(t, authResponse.ApprovedPurposeDetails)
	assert.Equal(t, 2, len(authResponse.ApprovedPurposeDetails.ApprovedPurposesNames))
	assert.Contains(t, authResponse.ApprovedPurposeDetails.ApprovedPurposesNames, "utility_read")
	assert.Contains(t, authResponse.ApprovedPurposeDetails.ApprovedPurposesNames, "taxes_read")

	// Cleanup
	cleanupAPITestData(t, env, consentResponse.ID)
}

func TestAPI_CreateAuthResourceConsentNotFound(t *testing.T) {
	env := setupAPITestEnvironment(t)
	defer func() {}()

	// Try to create authorization for non-existent consent
	authReq := &models.AuthorizationAPIRequest{
		UserID: "user123",
		Type:   "account",
		Status: "authorized",
		ApprovedPurposeDetails: &models.ApprovedPurposeDetails{
			ApprovedPurposesNames:       []string{"utility_read"},
			ApprovedAdditionalResources: []interface{}{},
		},
	}

	authReqBody, err := json.Marshal(authReq)
	require.NoError(t, err)

	req, err := http.NewRequest("POST", "/api/v1/consents/CONSENT-nonexistent/authorizations", bytes.NewBuffer(authReqBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("org-id", "TEST_ORG")

	recorder := httptest.NewRecorder()
	env.Router.ServeHTTP(recorder, req)

	// Assert 404 Not Found
	assert.Equal(t, http.StatusNotFound, recorder.Code, "Expected 404 Not Found status for non-existent consent")
}

func TestAPI_CreateAuthResourceInvalidRequest(t *testing.T) {
	env := setupAPITestEnvironment(t)
	defer func() {}()

	ctx := context.Background()
	
	// Create test purpose first
	desc := "Auth resource API test - invalid request"
	purpose := &models.ConsentPurpose{
		ID:          "PURPOSE-auth-invalid",
		Name:        "auth_invalid_test",
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
		Status:             "awaitingAuthorization",
		ValidityTime:       &validityTime,
		RecurringIndicator: &recurringIndicator,
		Frequency:          &frequency,
		ConsentPurpose: []models.ConsentPurposeItem{
			{Name: "auth_invalid_test", Value: "Test consent"},
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

	var consentResponse models.ConsentAPIResponse
	err = json.Unmarshal(recorder.Body.Bytes(), &consentResponse)
	require.NoError(t, err)

	// Step 2: Try to create authorization with missing required fields
	tests := []struct {
		name           string
		requestBody    string
		expectedStatus int
	}{
		{
			name:           "Missing type",
			requestBody:    `{"userId": "user123", "status": "authorized"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Missing status",
			requestBody:    `{"userId": "user123", "type": "account"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Invalid JSON",
			requestBody:    `{invalid json}`,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authReq, err := http.NewRequest("POST", "/api/v1/consents/"+consentResponse.ID+"/authorizations", bytes.NewBufferString(tt.requestBody))
			require.NoError(t, err)
			authReq.Header.Set("Content-Type", "application/json")
			authReq.Header.Set("org-id", "TEST_ORG")

			authRecorder := httptest.NewRecorder()
			env.Router.ServeHTTP(authRecorder, authReq)

			assert.Equal(t, tt.expectedStatus, authRecorder.Code)
		})
	}

	// Cleanup
	cleanupAPITestData(t, env, consentResponse.ID)
}

func TestAPI_GetAuthResource(t *testing.T) {
	env := setupAPITestEnvironment(t)
	defer func() {}()

	ctx := context.Background()
	
	// Create test purpose first
	desc := "Auth resource API test - get"
	purpose := &models.ConsentPurpose{
		ID:          "PURPOSE-auth-get",
		Name:        "auth_get_test",
		Description: &desc,
		OrgID:       "TEST_ORG",
	}
	
	err := env.ConsentPurposeDAO.Create(ctx, purpose)
	require.NoError(t, err)
	defer env.ConsentPurposeDAO.Delete(ctx, purpose.ID, "TEST_ORG")

	// Step 1: Create a consent
	validityTime := int64(7776000)
	frequency := 1
	recurringIndicator := false

	createReq := &models.ConsentAPIRequest{
		Type:               "accounts",
		Status:             "awaitingAuthorization",
		ValidityTime:       &validityTime,
		RecurringIndicator: &recurringIndicator,
		Frequency:          &frequency,
		ConsentPurpose: []models.ConsentPurposeItem{
			{Name: "auth_get_test", Value: "Test consent for GET auth"},
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

	var consentResponse models.ConsentAPIResponse
	err = json.Unmarshal(recorder.Body.Bytes(), &consentResponse)
	require.NoError(t, err)

	// Step 2: Create an authorization resource
	authReq := &models.AuthorizationAPIRequest{
		UserID: "user456",
		Type:   "payment",
		Status: "pending",
		ApprovedPurposeDetails: &models.ApprovedPurposeDetails{
			ApprovedPurposesNames:       []string{"utility_read", "taxes_read"},
			ApprovedAdditionalResources: []interface{}{},
		},
	}

	authReqBody, err := json.Marshal(authReq)
	require.NoError(t, err)

	authReqHTTP, err := http.NewRequest("POST", "/api/v1/consents/"+consentResponse.ID+"/authorizations", bytes.NewBuffer(authReqBody))
	require.NoError(t, err)
	authReqHTTP.Header.Set("Content-Type", "application/json")
	authReqHTTP.Header.Set("org-id", "TEST_ORG")

	authRecorder := httptest.NewRecorder()
	env.Router.ServeHTTP(authRecorder, authReqHTTP)

	require.Equal(t, http.StatusCreated, authRecorder.Code)

	var authCreateResponse models.AuthorizationAPIResponse
	err = json.Unmarshal(authRecorder.Body.Bytes(), &authCreateResponse)
	require.NoError(t, err)
	require.NotEmpty(t, authCreateResponse.ID)

	// Step 3: GET the authorization resource
	getReq, err := http.NewRequest("GET", "/api/v1/consents/"+consentResponse.ID+"/authorizations/"+authCreateResponse.ID, nil)
	require.NoError(t, err)
	getReq.Header.Set("org-id", "TEST_ORG")

	getRecorder := httptest.NewRecorder()
	env.Router.ServeHTTP(getRecorder, getReq)

	// Assert response
	assert.Equal(t, http.StatusOK, getRecorder.Code, "Expected 200 OK status")

	var authGetResponse models.AuthorizationAPIResponse
	err = json.Unmarshal(getRecorder.Body.Bytes(), &authGetResponse)
	require.NoError(t, err)

	// Verify response data
	assert.Equal(t, authCreateResponse.ID, authGetResponse.ID)
	assert.NotNil(t, authGetResponse.UserID)
	assert.Equal(t, "user456", *authGetResponse.UserID)
	assert.Equal(t, "payment", authGetResponse.Type)
	assert.Equal(t, "pending", authGetResponse.Status)
	assert.NotNil(t, authGetResponse.ApprovedPurposeDetails)
	assert.Equal(t, 2, len(authGetResponse.ApprovedPurposeDetails.ApprovedPurposesNames))
	assert.Contains(t, authGetResponse.ApprovedPurposeDetails.ApprovedPurposesNames, "utility_read")
	assert.Contains(t, authGetResponse.ApprovedPurposeDetails.ApprovedPurposesNames, "taxes_read")

	// Cleanup
	cleanupAPITestData(t, env, consentResponse.ID)
}

func TestAPI_GetAuthResourceNotFound(t *testing.T) {
	env := setupAPITestEnvironment(t)
	defer func() {}()

	ctx := context.Background()
	
	// Create test purpose first
	desc := "Auth resource API test - not found"
	purpose := &models.ConsentPurpose{
		ID:          "PURPOSE-auth-notfound",
		Name:        "auth_notfound_test",
		Description: &desc,
		OrgID:       "TEST_ORG",
	}
	
	err := env.ConsentPurposeDAO.Create(ctx, purpose)
	require.NoError(t, err)
	defer env.ConsentPurposeDAO.Delete(ctx, purpose.ID, "TEST_ORG")

	// Step 1: Create a consent
	validityTime := int64(7776000)
	frequency := 1
	recurringIndicator := false

	createReq := &models.ConsentAPIRequest{
		Type:               "accounts",
		Status:             "awaitingAuthorization",
		ValidityTime:       &validityTime,
		RecurringIndicator: &recurringIndicator,
		Frequency:          &frequency,
		ConsentPurpose: []models.ConsentPurposeItem{
			{Name: "auth_notfound_test", Value: "Test consent"},
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

	var consentResponse models.ConsentAPIResponse
	err = json.Unmarshal(recorder.Body.Bytes(), &consentResponse)
	require.NoError(t, err)

	// Step 2: Try to GET a non-existent authorization
	getReq, err := http.NewRequest("GET", "/api/v1/consents/"+consentResponse.ID+"/authorizations/AUTH-nonexistent", nil)
	require.NoError(t, err)
	getReq.Header.Set("org-id", "TEST_ORG")

	getRecorder := httptest.NewRecorder()
	env.Router.ServeHTTP(getRecorder, getReq)

	// Assert 404 Not Found
	assert.Equal(t, http.StatusNotFound, getRecorder.Code, "Expected 404 Not Found status")

	// Cleanup
	cleanupAPITestData(t, env, consentResponse.ID)
}

func TestAPI_GetAuthResourceInvalidID(t *testing.T) {
	env := setupAPITestEnvironment(t)
	defer func() {}()

	ctx := context.Background()
	
	// Create test purpose first
	desc := "Auth resource API test - invalid ID"
	purpose := &models.ConsentPurpose{
		ID:          "PURPOSE-auth-invalidid",
		Name:        "auth_invalidid_test",
		Description: &desc,
		OrgID:       "TEST_ORG",
	}
	
	err := env.ConsentPurposeDAO.Create(ctx, purpose)
	require.NoError(t, err)
	defer env.ConsentPurposeDAO.Delete(ctx, purpose.ID, "TEST_ORG")

	// Step 1: Create a consent
	validityTime := int64(7776000)
	frequency := 1
	recurringIndicator := false

	createReq := &models.ConsentAPIRequest{
		Type:               "accounts",
		Status:             "awaitingAuthorization",
		ValidityTime:       &validityTime,
		RecurringIndicator: &recurringIndicator,
		Frequency:          &frequency,
		ConsentPurpose: []models.ConsentPurposeItem{
			{Name: "auth_invalidid_test", Value: "Test consent"},
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

	var consentResponse models.ConsentAPIResponse
	err = json.Unmarshal(recorder.Body.Bytes(), &consentResponse)
	require.NoError(t, err)

	// Step 2: Try to GET with invalid auth ID (too long)
	longID := "AUTH-" + strings.Repeat("x", 300)
	getReq, err := http.NewRequest("GET", "/api/v1/consents/"+consentResponse.ID+"/authorizations/"+longID, nil)
	require.NoError(t, err)
	getReq.Header.Set("org-id", "TEST_ORG")

	getRecorder := httptest.NewRecorder()
	env.Router.ServeHTTP(getRecorder, getReq)

	// Assert 400 Bad Request for validation error
	assert.Equal(t, http.StatusBadRequest, getRecorder.Code, "Expected 400 Bad Request for invalid ID")

	// Cleanup
	cleanupAPITestData(t, env, consentResponse.ID)
}

func TestAPI_GetAuthResourceInvalidConsentID(t *testing.T) {
	env := setupAPITestEnvironment(t)
	defer func() {}()

	ctx := context.Background()
	
	// Create test purpose first
	desc := "Auth resource API test - invalid consent ID"
	purpose := &models.ConsentPurpose{
		ID:          "PURPOSE-auth-invalidconsent",
		Name:        "auth_invalidconsent_test",
		Description: &desc,
		OrgID:       "TEST_ORG",
	}
	
	err := env.ConsentPurposeDAO.Create(ctx, purpose)
	require.NoError(t, err)
	defer env.ConsentPurposeDAO.Delete(ctx, purpose.ID, "TEST_ORG")

	// Step 1: Create a consent
	validityTime := int64(7776000)
	frequency := 1
	recurringIndicator := false

	createReq := &models.ConsentAPIRequest{
		Type:               "accounts",
		Status:             "awaitingAuthorization",
		ValidityTime:       &validityTime,
		RecurringIndicator: &recurringIndicator,
		Frequency:          &frequency,
		ConsentPurpose: []models.ConsentPurposeItem{
			{Name: "auth_invalidconsent_test", Value: "Test consent for invalid consent ID test"},
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

	var consentResponse models.ConsentAPIResponse
	err = json.Unmarshal(recorder.Body.Bytes(), &consentResponse)
	require.NoError(t, err)
	require.NotEmpty(t, consentResponse.ID)

	// Step 2: Create an authorization resource
	authReq := &models.AuthorizationAPIRequest{
		UserID: "user123",
		Type:   "account",
		Status: "authorized",
		ApprovedPurposeDetails: &models.ApprovedPurposeDetails{
			ApprovedPurposesNames:       []string{"utility_read"},
			ApprovedAdditionalResources: []interface{}{},
		},
	}

	authReqBody, err := json.Marshal(authReq)
	require.NoError(t, err)

	authCreateReq, err := http.NewRequest("POST", "/api/v1/consents/"+consentResponse.ID+"/authorizations", bytes.NewBuffer(authReqBody))
	require.NoError(t, err)
	authCreateReq.Header.Set("Content-Type", "application/json")
	authCreateReq.Header.Set("org-id", "TEST_ORG")

	authCreateRecorder := httptest.NewRecorder()
	env.Router.ServeHTTP(authCreateRecorder, authCreateReq)

	require.Equal(t, http.StatusCreated, authCreateRecorder.Code)

	var authCreateResponse models.AuthorizationAPIResponse
	err = json.Unmarshal(authCreateRecorder.Body.Bytes(), &authCreateResponse)
	require.NoError(t, err)
	require.NotEmpty(t, authCreateResponse.ID)

	// Step 3: Try to GET the auth resource with invalid consent IDs
	tests := []struct {
		name       string
		consentID  string
		authID     string
		expectCode int
		expectMsg  string
	}{
		{
			name:       "Consent ID too long",
			consentID:  "CONSENT-" + strings.Repeat("x", 300),
			authID:     authCreateResponse.ID,
			expectCode: http.StatusBadRequest,
			expectMsg:  "Expected 400 Bad Request for too long consent ID",
		},
		{
			name:       "Non-existent consent ID",
			consentID:  "CONSENT-nonexistent",
			authID:     authCreateResponse.ID,
			expectCode: http.StatusNotFound,
			expectMsg:  "Expected 404 Not Found for non-existent consent",
		},
		{
			name:       "Invalid consent ID format",
			consentID:  "invalid@consent#id",
			authID:     authCreateResponse.ID,
			expectCode: http.StatusNotFound,
			expectMsg:  "Expected 404 Not Found for invalid consent ID format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			getReq, err := http.NewRequest("GET", "/api/v1/consents/"+tt.consentID+"/authorizations/"+tt.authID, nil)
			require.NoError(t, err)
			getReq.Header.Set("org-id", "TEST_ORG")

			getRecorder := httptest.NewRecorder()
			env.Router.ServeHTTP(getRecorder, getReq)

			assert.Equal(t, tt.expectCode, getRecorder.Code, tt.expectMsg)
		})
	}

	// Cleanup
	cleanupAPITestData(t, env, consentResponse.ID)
}

func TestAPI_UpdateAuthResource(t *testing.T) {
	env := setupAPITestEnvironment(t)
	defer func() {}()

	ctx := context.Background()
	
	// Create test purpose first
	desc := "Auth resource API test - update"
	purpose := &models.ConsentPurpose{
		ID:          "PURPOSE-auth-update",
		Name:        "auth_update_test",
		Description: &desc,
		OrgID:       "TEST_ORG",
	}
	
	err := env.ConsentPurposeDAO.Create(ctx, purpose)
	require.NoError(t, err)
	defer env.ConsentPurposeDAO.Delete(ctx, purpose.ID, "TEST_ORG")

	// Step 1: Create a consent
	validityTime := int64(7776000)
	frequency := 1
	recurringIndicator := false

	createReq := &models.ConsentAPIRequest{
		Type:               "accounts",
		Status:             "awaitingAuthorization",
		ValidityTime:       &validityTime,
		RecurringIndicator: &recurringIndicator,
		Frequency:          &frequency,
		ConsentPurpose: []models.ConsentPurposeItem{
			{Name: "auth_update_test", Value: "Test consent for update"},
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

	var consentResponse models.ConsentAPIResponse
	err = json.Unmarshal(recorder.Body.Bytes(), &consentResponse)
	require.NoError(t, err)
	require.NotEmpty(t, consentResponse.ID)

	// Step 2: Create an authorization resource
	authReq := &models.AuthorizationAPIRequest{
		UserID: "user123",
		Type:   "account",
		Status: "awaitingAuthorization",
		ApprovedPurposeDetails: &models.ApprovedPurposeDetails{
			ApprovedPurposesNames:       []string{"utility_read"},
			ApprovedAdditionalResources: []interface{}{},
		},
	}

	authReqBody, err := json.Marshal(authReq)
	require.NoError(t, err)

	authCreateReq, err := http.NewRequest("POST", "/api/v1/consents/"+consentResponse.ID+"/authorizations", bytes.NewBuffer(authReqBody))
	require.NoError(t, err)
	authCreateReq.Header.Set("Content-Type", "application/json")
	authCreateReq.Header.Set("org-id", "TEST_ORG")

	authCreateRecorder := httptest.NewRecorder()
	env.Router.ServeHTTP(authCreateRecorder, authCreateReq)

	require.Equal(t, http.StatusCreated, authCreateRecorder.Code)

	var authCreateResponse models.AuthorizationAPIResponse
	err = json.Unmarshal(authCreateRecorder.Body.Bytes(), &authCreateResponse)
	require.NoError(t, err)
	require.NotEmpty(t, authCreateResponse.ID)

	// Step 3: Update the authorization resource
	updateReq := &models.AuthorizationAPIUpdateRequest{
		UserID: "user456",
		Status: "authorized",
		ApprovedPurposeDetails: &models.ApprovedPurposeDetails{
			ApprovedPurposesNames:       []string{"utility_read", "taxes_read", "profile_read"},
			ApprovedAdditionalResources: []interface{}{},
		},
	}

	updateReqBody, err := json.Marshal(updateReq)
	require.NoError(t, err)

	authUpdateReq, err := http.NewRequest("PUT", "/api/v1/consents/"+consentResponse.ID+"/authorizations/"+authCreateResponse.ID, bytes.NewBuffer(updateReqBody))
	require.NoError(t, err)
	authUpdateReq.Header.Set("Content-Type", "application/json")
	authUpdateReq.Header.Set("org-id", "TEST_ORG")

	authUpdateRecorder := httptest.NewRecorder()
	env.Router.ServeHTTP(authUpdateRecorder, authUpdateReq)

	// Assert response
	if authUpdateRecorder.Code != http.StatusOK {
		t.Logf("Update failed with status %d: %s", authUpdateRecorder.Code, authUpdateRecorder.Body.String())
	}
	assert.Equal(t, http.StatusOK, authUpdateRecorder.Code, "Expected 200 OK status")

	var authUpdateResponse models.AuthorizationAPIResponse
	err = json.Unmarshal(authUpdateRecorder.Body.Bytes(), &authUpdateResponse)
	require.NoError(t, err)

	// Verify response data
	assert.Equal(t, authCreateResponse.ID, authUpdateResponse.ID)
	assert.NotNil(t, authUpdateResponse.UserID)
	assert.Equal(t, "user456", *authUpdateResponse.UserID)
	assert.Equal(t, "account", authUpdateResponse.Type) // Type should not change
	assert.Equal(t, "authorized", authUpdateResponse.Status)
	assert.NotNil(t, authUpdateResponse.ApprovedPurposeDetails)
	assert.Equal(t, 3, len(authUpdateResponse.ApprovedPurposeDetails.ApprovedPurposesNames))
	assert.Contains(t, authUpdateResponse.ApprovedPurposeDetails.ApprovedPurposesNames, "profile_read")

	// Cleanup
	cleanupAPITestData(t, env, consentResponse.ID)
}

func TestAPI_UpdateAuthResourceNotFound(t *testing.T) {
	env := setupAPITestEnvironment(t)
	defer func() {}()

	// Try to update a non-existent authorization
	updateReq := &models.AuthorizationAPIUpdateRequest{
		Status: "authorized",
	}

	updateReqBody, err := json.Marshal(updateReq)
	require.NoError(t, err)

	req, err := http.NewRequest("PUT", "/api/v1/consents/CONSENT-123/authorizations/AUTH-nonexistent", bytes.NewBuffer(updateReqBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("org-id", "TEST_ORG")

	recorder := httptest.NewRecorder()
	env.Router.ServeHTTP(recorder, req)

	// Assert 404 Not Found
	assert.Equal(t, http.StatusNotFound, recorder.Code, "Expected 404 Not Found status")
}

func TestAPI_UpdateAuthResourceInvalidConsentID(t *testing.T) {
	env := setupAPITestEnvironment(t)
	defer func() {}()

	ctx := context.Background()
	
	// Create test purpose first
	desc := "Auth resource API test - invalid consent ID update"
	purpose := &models.ConsentPurpose{
		ID:          "PURPOSE-auth-update-invalid",
		Name:        "auth_update_invalid_test",
		Description: &desc,
		OrgID:       "TEST_ORG",
	}
	
	err := env.ConsentPurposeDAO.Create(ctx, purpose)
	require.NoError(t, err)
	defer env.ConsentPurposeDAO.Delete(ctx, purpose.ID, "TEST_ORG")

	// Step 1: Create a consent
	validityTime := int64(7776000)
	frequency := 1
	recurringIndicator := false

	createReq := &models.ConsentAPIRequest{
		Type:               "accounts",
		Status:             "awaitingAuthorization",
		ValidityTime:       &validityTime,
		RecurringIndicator: &recurringIndicator,
		Frequency:          &frequency,
		ConsentPurpose: []models.ConsentPurposeItem{
			{Name: "auth_update_invalid_test", Value: "Test consent"},
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

	var consentResponse models.ConsentAPIResponse
	err = json.Unmarshal(recorder.Body.Bytes(), &consentResponse)
	require.NoError(t, err)

	// Step 2: Create an authorization resource
	authReq := &models.AuthorizationAPIRequest{
		UserID: "user123",
		Type:   "account",
		Status: "authorized",
		ApprovedPurposeDetails: &models.ApprovedPurposeDetails{
			ApprovedPurposesNames:       []string{"utility_read"},
			ApprovedAdditionalResources: []interface{}{},
		},
	}

	authReqBody, err := json.Marshal(authReq)
	require.NoError(t, err)

	authCreateReq, err := http.NewRequest("POST", "/api/v1/consents/"+consentResponse.ID+"/authorizations", bytes.NewBuffer(authReqBody))
	require.NoError(t, err)
	authCreateReq.Header.Set("Content-Type", "application/json")
	authCreateReq.Header.Set("org-id", "TEST_ORG")

	authCreateRecorder := httptest.NewRecorder()
	env.Router.ServeHTTP(authCreateRecorder, authCreateReq)

	require.Equal(t, http.StatusCreated, authCreateRecorder.Code)

	var authCreateResponse models.AuthorizationAPIResponse
	err = json.Unmarshal(authCreateRecorder.Body.Bytes(), &authCreateResponse)
	require.NoError(t, err)

	// Step 3: Try to update with wrong consent ID
	updateReq := &models.AuthorizationAPIUpdateRequest{
		Status: "authorized",
	}

	updateReqBody, err := json.Marshal(updateReq)
	require.NoError(t, err)

	updateHTTPReq, err := http.NewRequest("PUT", "/api/v1/consents/CONSENT-wrong/authorizations/"+authCreateResponse.ID, bytes.NewBuffer(updateReqBody))
	require.NoError(t, err)
	updateHTTPReq.Header.Set("Content-Type", "application/json")
	updateHTTPReq.Header.Set("org-id", "TEST_ORG")

	updateRecorder := httptest.NewRecorder()
	env.Router.ServeHTTP(updateRecorder, updateHTTPReq)

	// Assert 404 Not Found (auth resource doesn't belong to this consent)
	assert.Equal(t, http.StatusNotFound, updateRecorder.Code, "Expected 404 Not Found for wrong consent ID")

	// Cleanup
	cleanupAPITestData(t, env, consentResponse.ID)
}
