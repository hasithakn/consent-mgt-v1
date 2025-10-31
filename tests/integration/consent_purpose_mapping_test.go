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
	"github.com/wso2/consent-management-api/internal/service"
)

// TestConsentPurposeMapping_CreateWithPurposes tests creating a consent with purposes from extension
func TestConsentPurposeMapping_CreateWithPurposes(t *testing.T) {
	mockServer := NewMockExtensionServer()
	defer mockServer.Close()

	router, db, purposeService := setupExtensionTestEnvironment(t, mockServer)

	defer func() {
		_, _ = db.Exec("DELETE FROM FS_CONSENT WHERE ORG_ID = 'TEST_ORG'")
		_, _ = db.Exec("DELETE FROM CONSENT_PURPOSE WHERE ORG_ID = 'TEST_ORG'")
	}()

	// Create test purposes
	ctx := context.Background()
	purpose1, err := purposeService.CreatePurpose(ctx, "TEST_ORG", &service.ConsentPurposeCreateRequest{
		Name:        "AccountData",
		Description: stringPtr("Access to account data"),
		Type:        "string",
		Value:       "account:data",
	})
	require.NoError(t, err)

	purpose2, err := purposeService.CreatePurpose(ctx, "TEST_ORG", &service.ConsentPurposeCreateRequest{
		Name:        "TransactionHistory",
		Description: stringPtr("Access to transaction history"),
		Type:        "string",
		Value:       "transaction:history",
	})
	require.NoError(t, err)

	t.Logf("Created test purposes: %s (ID: %s), %s (ID: %s)",
		purpose1.Name, purpose1.ID, purpose2.Name, purpose2.ID)

	// Configure mock to return purposes
	mockServer.SetHandler(func(w http.ResponseWriter, r *http.Request) {
		var extRequest models.PreProcessConsentCreationRequest
		json.NewDecoder(r.Body).Decode(&extRequest)

		requestPayload := extRequest.Data.ConsentInitiationData.RequestPayload

		response := models.PreProcessConsentCreationResponse{
			ResponseID: extRequest.RequestID,
			Status:     "SUCCESS",
			Data: &models.PreProcessConsentCreationResponseData{
				ConsentResource: models.DetailedConsentResourceData{
					Type:           "accounts",
					Status:         "awaitingAuthorization",
					RequestPayload: requestPayload,
				},
				ResolvedConsentPurposes: []string{"AccountData", "TransactionHistory"},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	// Create consent via API
	body := map[string]interface{}{
		"type":   "accounts",
		"status": "awaitingAuthorization",
		"requestPayload": map[string]interface{}{
			"Data": map[string]interface{}{
				"Permissions": []string{"ReadAccountsBasic"},
			},
		},
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/api/v1/consents", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("org-id", "TEST_ORG")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code, "Response: %s", w.Body.String())

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	consentID := response["id"].(string)

	t.Logf("✓ Consent created: %s", consentID)

	// Verify purposes were linked in database
	purposes, err := purposeService.GetPurposesByConsentID(ctx, consentID, "TEST_ORG")
	require.NoError(t, err)
	require.Len(t, purposes, 2, "Expected 2 purposes to be linked")

	purposeNames := []string{purposes[0].Name, purposes[1].Name}
	assert.Contains(t, purposeNames, "AccountData")
	assert.Contains(t, purposeNames, "TransactionHistory")

	t.Logf("✓ Verified 2 purposes are linked to consent: %v", purposeNames)
}

// TestConsentPurposeMapping_UpdateWithPurposes tests updating a consent with new purposes
func TestConsentPurposeMapping_UpdateWithPurposes(t *testing.T) {
	mockServer := NewMockExtensionServer()
	defer mockServer.Close()

	router, db, purposeService := setupExtensionTestEnvironment(t, mockServer)

	defer func() {
		_, _ = db.Exec("DELETE FROM FS_CONSENT WHERE ORG_ID = 'TEST_ORG'")
		_, _ = db.Exec("DELETE FROM CONSENT_PURPOSE WHERE ORG_ID = 'TEST_ORG'")
	}()

	// Create test purposes
	ctx := context.Background()
	purpose1, err := purposeService.CreatePurpose(ctx, "TEST_ORG", &service.ConsentPurposeCreateRequest{
		Name:        "InitialPurpose",
		Description: stringPtr("Initial purpose"),
		Type:        "string",
		Value:       "initial:purpose",
	})
	require.NoError(t, err)

	purpose2, err := purposeService.CreatePurpose(ctx, "TEST_ORG", &service.ConsentPurposeCreateRequest{
		Name:        "UpdatedPurpose",
		Description: stringPtr("Updated purpose"),
		Type:        "string",
		Value:       "updated:purpose",
	})
	require.NoError(t, err)

	t.Logf("Created test purposes: %s, %s", purpose1.Name, purpose2.Name)

	// Step 1: Create consent with InitialPurpose
	mockServer.SetHandler(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/pre-process-consent-creation" {
			var extRequest models.PreProcessConsentCreationRequest
			json.NewDecoder(r.Body).Decode(&extRequest)

			response := models.PreProcessConsentCreationResponse{
				ResponseID: extRequest.RequestID,
				Status:     "SUCCESS",
				Data: &models.PreProcessConsentCreationResponseData{
					ConsentResource: models.DetailedConsentResourceData{
						Type:           "accounts",
						Status:         "awaitingAuthorization",
						RequestPayload: extRequest.Data.ConsentInitiationData.RequestPayload,
					},
					ResolvedConsentPurposes: []string{"InitialPurpose"},
				},
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}
	})

	createBody := map[string]interface{}{
		"type":   "accounts",
		"status": "awaitingAuthorization",
		"requestPayload": map[string]interface{}{
			"Data": map[string]interface{}{
				"Permissions": []string{"ReadAccountsBasic"},
			},
		},
	}
	bodyBytes, _ := json.Marshal(createBody)

	req := httptest.NewRequest("POST", "/api/v1/consents", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("org-id", "TEST_ORG")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)

	var createResponse map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &createResponse)
	consentID := createResponse["id"].(string)

	t.Logf("✓ Consent created: %s", consentID)

	// Verify initial purpose
	purposes, err := purposeService.GetPurposesByConsentID(ctx, consentID, "TEST_ORG")
	require.NoError(t, err)
	require.Len(t, purposes, 1)
	assert.Equal(t, "InitialPurpose", purposes[0].Name)

	t.Logf("✓ Verified initial purpose: %s", purposes[0].Name)

	// Step 2: Update consent with UpdatedPurpose
	mockServer.SetHandler(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/pre-process-consent-update" {
			var extRequest models.PreProcessConsentUpdateRequest
			json.NewDecoder(r.Body).Decode(&extRequest)

			response := models.PreProcessConsentUpdateResponse{
				ResponseID: extRequest.RequestID,
				Status:     "SUCCESS",
				Data: &models.PreProcessConsentUpdateResponseData{
					ConsentResource: models.DetailedConsentResourceData{
						Type:   "accounts",
						Status: "AUTHORIZED",
					},
					ResolvedConsentPurposes: []string{"UpdatedPurpose"},
				},
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}
	})

	updateBody := map[string]interface{}{
		"status": "AUTHORIZED",
	}
	bodyBytes, _ = json.Marshal(updateBody)

	req = httptest.NewRequest("PUT", "/api/v1/consents/"+consentID, bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("org-id", "TEST_ORG")

	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, "Response: %s", w.Body.String())

	t.Logf("✓ Consent updated")

	// Verify updated purpose (should replace initial purpose)
	purposes, err = purposeService.GetPurposesByConsentID(ctx, consentID, "TEST_ORG")
	require.NoError(t, err)
	require.Len(t, purposes, 1, "Expected only 1 purpose after update")
	assert.Equal(t, "UpdatedPurpose", purposes[0].Name)

	t.Logf("✓ Verified purpose was replaced: %s", purposes[0].Name)
}

// TestConsentPurposeMapping_MultipleConsents tests that multiple consents can share purposes
func TestConsentPurposeMapping_MultipleConsents(t *testing.T) {
	mockServer := NewMockExtensionServer()
	defer mockServer.Close()

	router, db, purposeService := setupExtensionTestEnvironment(t, mockServer)

	defer func() {
		_, _ = db.Exec("DELETE FROM FS_CONSENT WHERE ORG_ID = 'TEST_ORG'")
		_, _ = db.Exec("DELETE FROM CONSENT_PURPOSE WHERE ORG_ID = 'TEST_ORG'")
	}()

	// Create shared purpose
	ctx := context.Background()
	purpose, err := purposeService.CreatePurpose(ctx, "TEST_ORG", &service.ConsentPurposeCreateRequest{
		Name:        "SharedPurpose",
		Description: stringPtr("Purpose shared across consents"),
		Type:        "string",
		Value:       "shared:purpose",
	})
	require.NoError(t, err)

	t.Logf("Created shared purpose: %s", purpose.Name)

	// Configure mock to return shared purpose
	mockServer.SetHandler(func(w http.ResponseWriter, r *http.Request) {
		var extRequest models.PreProcessConsentCreationRequest
		json.NewDecoder(r.Body).Decode(&extRequest)

		response := models.PreProcessConsentCreationResponse{
			ResponseID: extRequest.RequestID,
			Status:     "SUCCESS",
			Data: &models.PreProcessConsentCreationResponseData{
				ConsentResource: models.DetailedConsentResourceData{
					Type:           "accounts",
					Status:         "awaitingAuthorization",
					RequestPayload: extRequest.Data.ConsentInitiationData.RequestPayload,
				},
				ResolvedConsentPurposes: []string{"SharedPurpose"},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	// Create two consents with same purpose
	consentIDs := []string{}
	for i := 0; i < 2; i++ {
		body := map[string]interface{}{
			"type":   "accounts",
			"status": "awaitingAuthorization",
			"requestPayload": map[string]interface{}{
				"Data": map[string]interface{}{
					"Permissions": []string{"ReadAccountsBasic"},
				},
			},
		}
		bodyBytes, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/api/v1/consents", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("org-id", "TEST_ORG")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusCreated, w.Code)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		consentIDs = append(consentIDs, response["id"].(string))
	}

	t.Logf("✓ Created 2 consents: %s, %s", consentIDs[0], consentIDs[1])

	// Verify both consents have the same purpose
	for i, consentID := range consentIDs {
		purposes, err := purposeService.GetPurposesByConsentID(ctx, consentID, "TEST_ORG")
		require.NoError(t, err)
		require.Len(t, purposes, 1)
		assert.Equal(t, "SharedPurpose", purposes[0].Name)
		t.Logf("✓ Consent %d has purpose: %s", i+1, purposes[0].Name)
	}

	t.Log("✓ Both consents share the same purpose")
}
