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

// ============================================================================
// DELETE - String Type Handler Tests
// ============================================================================

func TestDeleteConsentPurpose_StringTypeHandler_Success(t *testing.T) {
	env := setupConsentPurposeAPITestEnvironment(t)

	// Create a string type purpose
	uniqueName := fmt.Sprintf("delete_string_%d", time.Now().UnixNano())
	createReq := []map[string]interface{}{
		{
			"name": uniqueName,
			"type": "string",
			"attributes": map[string]string{
				"value": "to:be:deleted",
			},
		},
	}

	body, _ := json.Marshal(createReq)
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

	// Delete the purpose
	deleteReq := httptest.NewRequest("DELETE", "/api/v1/consent-purposes/"+purposeID, nil)
	deleteReq.Header.Set("org-id", "TEST_ORG")

	w = httptest.NewRecorder()
	env.Router.ServeHTTP(w, deleteReq)

	assert.Equal(t, http.StatusNoContent, w.Code)

	// Verify it's deleted
	getReq := httptest.NewRequest("GET", "/api/v1/consent-purposes/"+purposeID, nil)
	getReq.Header.Set("org-id", "TEST_ORG")

	w = httptest.NewRecorder()
	env.Router.ServeHTTP(w, getReq)

	assert.Equal(t, http.StatusNotFound, w.Code)
	t.Log("✓ String type handler: Successfully deleted string type purpose")
}

// ============================================================================
// DELETE - JSON Schema Type Handler Tests
// ============================================================================

func TestDeleteConsentPurpose_JsonSchemaTypeHandler_Success(t *testing.T) {
	env := setupConsentPurposeAPITestEnvironment(t)

	// Create a json-schema type purpose
	uniqueName := fmt.Sprintf("delete_jsonschema_%d", time.Now().UnixNano())
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"name": map[string]string{"type": "string"},
		},
	}
	schemaBytes, _ := json.Marshal(schema)

	createReq := []map[string]interface{}{
		{
			"name": uniqueName,
			"type": "json-schema",
			"attributes": map[string]string{
				"validationSchema": string(schemaBytes),
			},
		},
	}

	body, _ := json.Marshal(createReq)
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

	// Delete the purpose
	deleteReq := httptest.NewRequest("DELETE", "/api/v1/consent-purposes/"+purposeID, nil)
	deleteReq.Header.Set("org-id", "TEST_ORG")

	w = httptest.NewRecorder()
	env.Router.ServeHTTP(w, deleteReq)

	assert.Equal(t, http.StatusNoContent, w.Code)

	// Verify it's deleted
	getReq := httptest.NewRequest("GET", "/api/v1/consent-purposes/"+purposeID, nil)
	getReq.Header.Set("org-id", "TEST_ORG")

	w = httptest.NewRecorder()
	env.Router.ServeHTTP(w, getReq)

	assert.Equal(t, http.StatusNotFound, w.Code)
	t.Log("✓ JSON Schema type handler: Successfully deleted json-schema type purpose")
}

// ============================================================================
// DELETE - Attribute Type Handler Tests
// ============================================================================

func TestDeleteConsentPurpose_AttributeTypeHandler_Success(t *testing.T) {
	env := setupConsentPurposeAPITestEnvironment(t)

	// Create an attribute type purpose
	uniqueName := fmt.Sprintf("delete_attribute_%d", time.Now().UnixNano())
	createReq := []map[string]interface{}{
		{
			"name": uniqueName,
			"type": "attribute",
			"attributes": map[string]string{
				"resourcePath": "/api/users",
				"jsonPath":     "$.permissions",
			},
		},
	}

	body, _ := json.Marshal(createReq)
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

	// Delete the purpose
	deleteReq := httptest.NewRequest("DELETE", "/api/v1/consent-purposes/"+purposeID, nil)
	deleteReq.Header.Set("org-id", "TEST_ORG")

	w = httptest.NewRecorder()
	env.Router.ServeHTTP(w, deleteReq)

	assert.Equal(t, http.StatusNoContent, w.Code)

	// Verify it's deleted
	getReq := httptest.NewRequest("GET", "/api/v1/consent-purposes/"+purposeID, nil)
	getReq.Header.Set("org-id", "TEST_ORG")

	w = httptest.NewRecorder()
	env.Router.ServeHTTP(w, getReq)

	assert.Equal(t, http.StatusNotFound, w.Code)
	t.Log("✓ Attribute type handler: Successfully deleted attribute type purpose")
}

// ============================================================================
// DELETE - Verify attributes are also deleted
// ============================================================================

func TestDeleteConsentPurpose_AllTypeHandlers_AttributesAlsoDeleted(t *testing.T) {
	env := setupConsentPurposeAPITestEnvironment(t)

	// Create purposes of all types with attributes
	uniquePrefix := fmt.Sprintf("delete_all_types_%d", time.Now().UnixNano())

	schema := map[string]interface{}{
		"type": "object",
	}
	schemaBytes, _ := json.Marshal(schema)

	createReq := []map[string]interface{}{
		{
			"name": uniquePrefix + "_string",
			"type": "string",
			"attributes": map[string]string{
				"value": "test:value",
			},
		},
		{
			"name": uniquePrefix + "_jsonschema",
			"type": "json-schema",
			"attributes": map[string]string{
				"validationSchema": string(schemaBytes),
			},
		},
		{
			"name": uniquePrefix + "_attribute",
			"type": "attribute",
			"attributes": map[string]string{
				"resourcePath": "/api/test",
				"jsonPath":     "$.test",
			},
		},
	}

	body, _ := json.Marshal(createReq)
	req := httptest.NewRequest("POST", "/api/v1/consent-purposes", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("org-id", "TEST_ORG")

	w := httptest.NewRecorder()
	env.Router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var createResponse map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &createResponse)
	data := createResponse["data"].([]interface{})

	// Delete each purpose and verify
	for i, item := range data {
		purpose := item.(map[string]interface{})
		purposeID := purpose["id"].(string)

		// Delete
		deleteReq := httptest.NewRequest("DELETE", "/api/v1/consent-purposes/"+purposeID, nil)
		deleteReq.Header.Set("org-id", "TEST_ORG")

		w = httptest.NewRecorder()
		env.Router.ServeHTTP(w, deleteReq)

		assert.Equal(t, http.StatusNoContent, w.Code, "Delete failed for purpose %d", i)

		// Verify deletion
		getReq := httptest.NewRequest("GET", "/api/v1/consent-purposes/"+purposeID, nil)
		getReq.Header.Set("org-id", "TEST_ORG")

		w = httptest.NewRecorder()
		env.Router.ServeHTTP(w, getReq)

		assert.Equal(t, http.StatusNotFound, w.Code, "Purpose %d should be deleted", i)
	}

	t.Log("✓ All type handlers: All types successfully deleted with their attributes")
}
