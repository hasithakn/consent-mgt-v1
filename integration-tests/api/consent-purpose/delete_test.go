package consentpurpose

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestDeleteConsentPurpose_Success tests deleting a consent purpose
func TestDeleteConsentPurpose_Success(t *testing.T) {
	env := setupConsentPurposeAPITestEnvironment(t)

	// Create a purpose
	createReq := []map[string]interface{}{
		{"name": "ToBeDeleted", "description": "Will be deleted", "type": "string", "attributes": map[string]string{"value": "delete:me"}},
	}

	body, _ := json.Marshal(createReq)
	req := httptest.NewRequest("POST", "/api/v1/consent-purposes", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("org-id", "TEST_ORG")

	w := httptest.NewRecorder()
	env.Router.ServeHTTP(w, req)

	var createResponse map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &createResponse)
	data := createResponse["data"].([]interface{})
	purpose := data[0].(map[string]interface{})
	purposeID := purpose["id"].(string)

	// Delete the purpose
	deleteReq := httptest.NewRequest("DELETE", "/api/v1/consent-purposes/"+purposeID, nil)
	deleteReq.Header.Set("org-id", "TEST_ORG")

	w = httptest.NewRecorder()
	env.Router.ServeHTTP(w, deleteReq)

	assert.Equal(t, http.StatusNoContent, w.Code)

	// Verify it's deleted by trying to get it
	getReq := httptest.NewRequest("GET", "/api/v1/consent-purposes/"+purposeID, nil)
	getReq.Header.Set("org-id", "TEST_ORG")

	w = httptest.NewRecorder()
	env.Router.ServeHTTP(w, getReq)

	assert.Equal(t, http.StatusNotFound, w.Code)
	t.Log("✓ Successfully deleted purpose")
}

// TestDeleteConsentPurpose_NotFound tests deleting non-existent purpose
func TestDeleteConsentPurpose_NotFound(t *testing.T) {
	env := setupConsentPurposeAPITestEnvironment(t)

	req := httptest.NewRequest("DELETE", "/api/v1/consent-purposes/NONEXISTENT-ID", nil)
	req.Header.Set("org-id", "TEST_ORG")

	w := httptest.NewRecorder()
	env.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	t.Log("✓ Correctly handled delete of non-existent purpose")
}
