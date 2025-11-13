package consentpurpose

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

func TestGetConsentPurpose_Success(t *testing.T) {
	env := setupConsentPurposeAPITestEnvironment(t)

	// First create a purpose
	createReq := []map[string]interface{}{
		{
			"name":        "TestPurpose",
			"description": "Test Description",
			"type":        "string",
			"attributes": map[string]string{
				"value": "test:purpose",
			},
		},
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

	// Now get the purpose
	getReq := httptest.NewRequest("GET", "/api/v1/consent-purposes/"+purposeID, nil)
	getReq.Header.Set("org-id", "TEST_ORG")

	w = httptest.NewRecorder()
	env.Router.ServeHTTP(w, getReq)

	assert.Equal(t, http.StatusOK, w.Code)

	var retrievedPurpose map[string]interface{}
	err2 := json.Unmarshal(w.Body.Bytes(), &retrievedPurpose)
	require.NoError(t, err2)

	assert.Equal(t, purposeID, retrievedPurpose["id"])
	assert.Equal(t, "TestPurpose", retrievedPurpose["name"])
	assert.Equal(t, "Test Description", retrievedPurpose["description"])
	assert.Equal(t, "string", retrievedPurpose["type"])
	
	// Check attributes
	if attrs, ok := retrievedPurpose["attributes"].(map[string]interface{}); ok {
		assert.Equal(t, "test:purpose", attrs["value"])
	}

	t.Logf("✓ Successfully retrieved purpose: %s", purposeID)

	// Cleanup
	_ = env.PurposeDAO.Delete(req.Context(), purposeID, "TEST_ORG")
}

// TestGetConsentPurpose_NotFound tests retrieving non-existent purpose
func TestGetConsentPurpose_NotFound(t *testing.T) {
	env := setupConsentPurposeAPITestEnvironment(t)

	req := httptest.NewRequest("GET", "/api/v1/consent-purposes/NONEXISTENT-ID", nil)
	req.Header.Set("org-id", "TEST_ORG")

	w := httptest.NewRecorder()
	env.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	t.Log("✓ Correctly handled non-existent purpose")
}

// TestListConsentPurposes_Success tests listing consent purposes
func TestListConsentPurposes_Success(t *testing.T) {
	env := setupConsentPurposeAPITestEnvironment(t)

	// Create multiple purposes
	createReq := []map[string]interface{}{
		{"name": "Purpose1", "description": "Desc1", "type": "string", "attributes": map[string]string{"value": "purpose:1"}},
		{"name": "Purpose2", "description": "Desc2", "type": "string", "attributes": map[string]string{"value": "purpose:2"}},
		{"name": "Purpose3", "description": "Desc3", "type": "string", "attributes": map[string]string{"value": "purpose:3"}},
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

	// Collect IDs for cleanup
	purposeIDs := make([]string, 0, 3)
	for _, item := range data {
		purpose := item.(map[string]interface{})
		purposeIDs = append(purposeIDs, purpose["id"].(string))
	}

	// Now list purposes
	listReq := httptest.NewRequest("GET", "/api/v1/consent-purposes", nil)
	listReq.Header.Set("org-id", "TEST_ORG")

	w = httptest.NewRecorder()
	env.Router.ServeHTTP(w, listReq)

	assert.Equal(t, http.StatusOK, w.Code)

	var listResponse models.ConsentPurposeListResponse
	err := json.Unmarshal(w.Body.Bytes(), &listResponse)
	require.NoError(t, err)

	purposes := listResponse.Purposes
	total := listResponse.Total

	assert.GreaterOrEqual(t, len(purposes), 3)
	assert.GreaterOrEqual(t, total, 3)

	// Verify that type and value fields are present in listed purposes
	foundCount := 0
	for _, p := range purposes {
		// Check if this is one of our created purposes
		for i, id := range purposeIDs {
			if p.ID == id {
				assert.Equal(t, createReq[i]["name"], p.Name)
				assert.Equal(t, createReq[i]["type"], p.Type)
				foundCount++
			}
		}
	}
	assert.Equal(t, 3, foundCount, "All created purposes should be found in the list")

	t.Logf("✓ Successfully listed %d purposes (total: %d)", len(purposes), total)

	// Cleanup
	for _, purposeID := range purposeIDs {
		_ = env.PurposeDAO.Delete(req.Context(), purposeID, "TEST_ORG")
	}
}

// TestListConsentPurposes_WithPagination tests pagination
func TestListConsentPurposes_WithPagination(t *testing.T) {
	env := setupConsentPurposeAPITestEnvironment(t)

	// Create 5 purposes
	createReq := []map[string]interface{}{
		{"name": "PurposeA", "type": "string", "attributes": map[string]string{"value": "purpose:a"}},
		{"name": "PurposeB", "type": "string", "attributes": map[string]string{"value": "purpose:b"}},
		{"name": "PurposeC", "type": "string", "attributes": map[string]string{"value": "purpose:c"}},
		{"name": "PurposeD", "type": "string", "attributes": map[string]string{"value": "purpose:d"}},
		{"name": "PurposeE", "type": "string", "attributes": map[string]string{"value": "purpose:e"}},
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

	purposeIDs := make([]string, 0, 5)
	for _, item := range data {
		purpose := item.(map[string]interface{})
		purposeIDs = append(purposeIDs, purpose["id"].(string))
	}

	// List with limit=2
	listReq := httptest.NewRequest("GET", "/api/v1/consent-purposes?limit=2&offset=0", nil)
	listReq.Header.Set("org-id", "TEST_ORG")

	w = httptest.NewRecorder()
	env.Router.ServeHTTP(w, listReq)

	assert.Equal(t, http.StatusOK, w.Code)

	var listResponse models.ConsentPurposeListResponse
	json.Unmarshal(w.Body.Bytes(), &listResponse)

	purposes := listResponse.Purposes

	assert.LessOrEqual(t, len(purposes), 2)
	t.Logf("✓ Pagination working: got %d purposes with limit=2", len(purposes))

	// Cleanup
	for _, purposeID := range purposeIDs {
		_ = env.PurposeDAO.Delete(req.Context(), purposeID, "TEST_ORG")
	}
}
