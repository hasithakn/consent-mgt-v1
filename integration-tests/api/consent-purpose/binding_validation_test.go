package consentpurpose

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUpdateConsentPurpose_NameChangeWithBindings tests that updating a purpose name
// is prevented when it has existing consent bindings
func TestUpdateConsentPurpose_NameChangeWithBindings(t *testing.T) {
	env := setupConsentPurposeAPITestEnvironment(t)

	// Step 1: Create a consent purpose with unique name
	purposeName := fmt.Sprintf("AccountAccess_%d", time.Now().UnixNano())
	createPurposeReq := []map[string]interface{}{
		{
			"name":        purposeName,
			"description": "Access to account information",
			"type":        "string",
			"attributes": map[string]string{
				"value": "account:read",
			},
		},
	}

	body, _ := json.Marshal(createPurposeReq)
	req := httptest.NewRequest("POST", "/api/v1/consent-purposes", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("org-id", "TEST_ORG")

	w := httptest.NewRecorder()
	env.Router.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)

	var createResponse map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &createResponse)
	data := createResponse["data"].([]interface{})
	purpose := data[0].(map[string]interface{})
	purposeID := purpose["id"].(string)

	t.Logf("✓ Created consent purpose: %s", purposeID)

	// Step 2: Create a consent that uses this purpose
	createConsentReq := map[string]interface{}{
		"type": "SHARING",
		"consentPurpose": []map[string]interface{}{
			{
				"name":  purposeName,
				"value": "account:read:write",
			},
		},
		"authorizations": []map[string]interface{}{
			{
				"type":   "authorization",
				"userId": "user123",
				"status": "approved",
			},
		},
	}

	body, _ = json.Marshal(createConsentReq)
	req = httptest.NewRequest("POST", "/api/v1/consents", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("org-id", "TEST_ORG")
	req.Header.Set("client-id", "test-client")

	w = httptest.NewRecorder()
	env.Router.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)

	var consentResponse map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &consentResponse)
	consentID := consentResponse["id"].(string)

	t.Logf("✓ Created consent with purpose binding: %s", consentID)

	// Step 3: Try to update the purpose NAME (should fail due to binding)
	updateReq := map[string]interface{}{
		"name":        "RenamedAccountAccess", // Changing the name
		"description": "Access to account information",
		"type":        "string",
		"attributes": map[string]string{
			"value": "account:read",
		},
	}

	body, _ = json.Marshal(updateReq)
	req = httptest.NewRequest("PUT", "/api/v1/consent-purposes/"+purposeID, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("org-id", "TEST_ORG")

	w = httptest.NewRecorder()
	env.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var errorResponse map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &errorResponse)

	assert.Contains(t, errorResponse["details"].(string), "currently used by")
	assert.Contains(t, errorResponse["details"].(string), "consent(s)")
	assert.Contains(t, errorResponse["details"].(string), "purpose name")

	t.Log("✓ Correctly prevented name change when bindings exist")

	// Cleanup
	_ = env.ConsentDAO.Delete(req.Context(), consentID, "TEST_ORG")
	_ = env.PurposeDAO.Delete(req.Context(), purposeID, "TEST_ORG")
}

// TestUpdateConsentPurpose_DescriptionChangeWithBindings tests that updating
// description, type, or attributes is ALLOWED even with bindings
func TestUpdateConsentPurpose_DescriptionChangeWithBindings(t *testing.T) {
	env := setupConsentPurposeAPITestEnvironment(t)

	// Step 1: Create a consent purpose with unique name
	purposeName := fmt.Sprintf("ProfileAccess_%d", time.Now().UnixNano())
	createPurposeReq := []map[string]interface{}{
		{
			"name":        purposeName,
			"description": "Original description",
			"type":        "string",
			"attributes": map[string]string{
				"value": "profile:read",
			},
		},
	}

	body, _ := json.Marshal(createPurposeReq)
	req := httptest.NewRequest("POST", "/api/v1/consent-purposes", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("org-id", "TEST_ORG")

	w := httptest.NewRecorder()
	env.Router.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)

	var createResponse map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &createResponse)
	data := createResponse["data"].([]interface{})
	purpose := data[0].(map[string]interface{})
	purposeID := purpose["id"].(string)

	t.Logf("✓ Created consent purpose: %s", purposeID)

	// Step 2: Create a consent that uses this purpose
	createConsentReq := map[string]interface{}{
		"type": "SHARING",
		"consentPurpose": []map[string]interface{}{
			{
				"name":  purposeName,
				"value": "profile:read:write",
			},
		},
		"authorizations": []map[string]interface{}{
			{
				"type":   "authorization",
				"userId": "user123",
				"status": "approved",
			},
		},
	}

	body, _ = json.Marshal(createConsentReq)
	req = httptest.NewRequest("POST", "/api/v1/consents", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("org-id", "TEST_ORG")
	req.Header.Set("client-id", "test-client")

	w = httptest.NewRecorder()
	env.Router.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)

	var consentResponse map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &consentResponse)
	consentID := consentResponse["id"].(string)

	t.Logf("✓ Created consent with purpose binding: %s", consentID)

	// Step 3: Update description and attributes (name stays the same - should succeed)
	updateReq := map[string]interface{}{
		"name":        purposeName, // Same name - no change
		"description": "Updated description with more details",
		"type":        "string",
		"attributes": map[string]string{
			"value": "profile:read:write:delete", // Updated attribute
		},
	}

	body, _ = json.Marshal(updateReq)
	req = httptest.NewRequest("PUT", "/api/v1/consent-purposes/"+purposeID, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("org-id", "TEST_ORG")

	w = httptest.NewRecorder()
	env.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var updatedPurpose map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &updatedPurpose)
	require.NoError(t, err)

	assert.Equal(t, purposeName, updatedPurpose["name"])
	assert.Equal(t, "Updated description with more details", updatedPurpose["description"])

	if attrs, ok := updatedPurpose["attributes"].(map[string]interface{}); ok {
		assert.Equal(t, "profile:read:write:delete", attrs["value"])
	}

	t.Log("✓ Successfully updated description and attributes while keeping same name")

	// Cleanup
	_ = env.ConsentDAO.Delete(req.Context(), consentID, "TEST_ORG")
	_ = env.PurposeDAO.Delete(req.Context(), purposeID, "TEST_ORG")
}

// TestDeleteConsentPurpose_WithBindings tests that deleting a purpose
// is prevented when it has existing consent bindings
func TestDeleteConsentPurpose_WithBindings(t *testing.T) {
	env := setupConsentPurposeAPITestEnvironment(t)

	// Step 1: Create a consent purpose with unique name
	purposeName := fmt.Sprintf("PaymentAccess_%d", time.Now().UnixNano())
	createPurposeReq := []map[string]interface{}{
		{
			"name":        purposeName,
			"description": "Access to payment information",
			"type":        "string",
			"attributes": map[string]string{
				"value": "payment:read",
			},
		},
	}

	body, _ := json.Marshal(createPurposeReq)
	req := httptest.NewRequest("POST", "/api/v1/consent-purposes", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("org-id", "TEST_ORG")

	w := httptest.NewRecorder()
	env.Router.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)

	var createResponse map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &createResponse)
	data := createResponse["data"].([]interface{})
	purpose := data[0].(map[string]interface{})
	purposeID := purpose["id"].(string)

	t.Logf("✓ Created consent purpose: %s", purposeID)

	// Step 2: Create a consent that uses this purpose
	createConsentReq := map[string]interface{}{
		"type": "SHARING",
		"consentPurpose": []map[string]interface{}{
			{
				"name":  purposeName,
				"value": "payment:read:write",
			},
		},
		"authorizations": []map[string]interface{}{
			{
				"type":   "authorization",
				"userId": "user123",
				"status": "approved",
			},
		},
	}

	body, _ = json.Marshal(createConsentReq)
	req = httptest.NewRequest("POST", "/api/v1/consents", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("org-id", "TEST_ORG")
	req.Header.Set("client-id", "test-client")

	w = httptest.NewRecorder()
	env.Router.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)

	var consentResponse map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &consentResponse)
	consentID := consentResponse["id"].(string)

	t.Logf("✓ Created consent with purpose binding: %s", consentID)

	// Step 3: Try to delete the purpose (should fail due to binding)
	req = httptest.NewRequest("DELETE", "/api/v1/consent-purposes/"+purposeID, nil)
	req.Header.Set("org-id", "TEST_ORG")

	w = httptest.NewRecorder()
	env.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var errorResponse map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &errorResponse)

	assert.Contains(t, errorResponse["details"].(string), "currently used by")
	assert.Contains(t, errorResponse["details"].(string), "consent(s)")

	t.Log("✓ Correctly prevented deletion when bindings exist")

	// Cleanup
	_ = env.ConsentDAO.Delete(req.Context(), consentID, "TEST_ORG")
	_ = env.PurposeDAO.Delete(req.Context(), purposeID, "TEST_ORG")
}

// TestDeleteConsentPurpose_WithoutBindings tests that deleting a purpose
// succeeds when it has no consent bindings
func TestDeleteConsentPurpose_WithoutBindings(t *testing.T) {
	env := setupConsentPurposeAPITestEnvironment(t)

	// Step 1: Create a consent purpose
	createPurposeReq := []map[string]interface{}{
		{
			"name":        "UnusedPurpose",
			"description": "This purpose has no consents",
			"type":        "string",
			"attributes": map[string]string{
				"value": "unused:read",
			},
		},
	}

	body, _ := json.Marshal(createPurposeReq)
	req := httptest.NewRequest("POST", "/api/v1/consent-purposes", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("org-id", "TEST_ORG")

	w := httptest.NewRecorder()
	env.Router.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)

	var createResponse map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &createResponse)
	data := createResponse["data"].([]interface{})
	purpose := data[0].(map[string]interface{})
	purposeID := purpose["id"].(string)

	t.Logf("✓ Created consent purpose: %s", purposeID)

	// Step 2: Delete the purpose (should succeed - no bindings)
	req = httptest.NewRequest("DELETE", "/api/v1/consent-purposes/"+purposeID, nil)
	req.Header.Set("org-id", "TEST_ORG")

	w = httptest.NewRecorder()
	env.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)

	t.Log("✓ Successfully deleted purpose without bindings")

	// Verify it's actually deleted
	req = httptest.NewRequest("GET", "/api/v1/consent-purposes/"+purposeID, nil)
	req.Header.Set("org-id", "TEST_ORG")

	w = httptest.NewRecorder()
	env.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	t.Log("✓ Verified purpose was actually deleted")
}
