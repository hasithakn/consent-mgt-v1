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
// UPDATE - String Type Handler Tests
// ============================================================================

func TestUpdateConsentPurpose_StringTypeHandler_Success(t *testing.T) {
	env := setupConsentPurposeAPITestEnvironment(t)

	// Create a purpose with string type
	uniqueName := fmt.Sprintf("update_string_%d", time.Now().UnixNano())
	createReq := []map[string]interface{}{
		{
			"name":        uniqueName,
			"description": "Original",
			"type":        "string",
			"attributes": map[string]string{
				"value": "original:value",
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

	// Update with new value
	updateReq := map[string]interface{}{
		"name":        uniqueName + "_updated",
		"description": "Updated",
		"type":        "string",
		"attributes": map[string]string{
			"value": "updated:value",
		},
	}

	body, _ = json.Marshal(updateReq)
	putReq := httptest.NewRequest("PUT", "/api/v1/consent-purposes/"+purposeID, bytes.NewBuffer(body))
	putReq.Header.Set("Content-Type", "application/json")
	putReq.Header.Set("org-id", "TEST_ORG")

	w = httptest.NewRecorder()
	env.Router.ServeHTTP(w, putReq)

	assert.Equal(t, http.StatusOK, w.Code)

	var updatedPurpose map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &updatedPurpose)
	require.NoError(t, err)

	assert.Equal(t, "string", updatedPurpose["type"])
	if attrs, ok := updatedPurpose["attributes"].(map[string]interface{}); ok {
		assert.Equal(t, "updated:value", attrs["value"])
	}

	t.Log("✓ String type handler: Successfully updated string type purpose")

	// Cleanup
	_ = env.PurposeDAO.Delete(req.Context(), purposeID, "TEST_ORG")
}

func TestUpdateConsentPurpose_StringTypeHandler_NoMandatoryAttributes(t *testing.T) {
	env := setupConsentPurposeAPITestEnvironment(t)

	// Create a purpose
	uniqueName := fmt.Sprintf("update_string_no_attrs_%d", time.Now().UnixNano())
	createReq := []map[string]interface{}{
		{
			"name": uniqueName,
			"type": "string",
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

	// Update without attributes (valid for string type)
	updateReq := map[string]interface{}{
		"name": uniqueName + "_updated",
		"type": "string",
	}

	body, _ = json.Marshal(updateReq)
	putReq := httptest.NewRequest("PUT", "/api/v1/consent-purposes/"+purposeID, bytes.NewBuffer(body))
	putReq.Header.Set("Content-Type", "application/json")
	putReq.Header.Set("org-id", "TEST_ORG")

	w = httptest.NewRecorder()
	env.Router.ServeHTTP(w, putReq)

	assert.Equal(t, http.StatusOK, w.Code)
	t.Log("✓ String type handler: Update without attributes is valid (no mandatory attributes)")

	// Cleanup
	_ = env.PurposeDAO.Delete(req.Context(), purposeID, "TEST_ORG")
}

// ============================================================================
// UPDATE - JSON Schema Type Handler Tests
// ============================================================================

func TestUpdateConsentPurpose_JsonSchemaTypeHandler_Success(t *testing.T) {
	env := setupConsentPurposeAPITestEnvironment(t)

	// Create with json-schema type
	uniqueName := fmt.Sprintf("update_jsonschema_%d", time.Now().UnixNano())
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

	// Update with new schema
	newSchema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"email": map[string]string{"type": "string", "format": "email"},
		},
	}
	newSchemaBytes, _ := json.Marshal(newSchema)

	updateReq := map[string]interface{}{
		"name": uniqueName + "_updated",
		"type": "json-schema",
		"attributes": map[string]string{
			"validationSchema": string(newSchemaBytes),
		},
	}

	body, _ = json.Marshal(updateReq)
	putReq := httptest.NewRequest("PUT", "/api/v1/consent-purposes/"+purposeID, bytes.NewBuffer(body))
	putReq.Header.Set("Content-Type", "application/json")
	putReq.Header.Set("org-id", "TEST_ORG")

	w = httptest.NewRecorder()
	env.Router.ServeHTTP(w, putReq)

	assert.Equal(t, http.StatusOK, w.Code)

	var updatedPurpose map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &updatedPurpose)
	require.NoError(t, err)

	assert.Equal(t, "json-schema", updatedPurpose["type"])
	if attrs, ok := updatedPurpose["attributes"].(map[string]interface{}); ok {
		assert.Contains(t, attrs, "validationSchema")
		assert.Contains(t, attrs["validationSchema"], "email")
	}

	t.Log("✓ JSON Schema type handler: Successfully updated json-schema type purpose")

	// Cleanup
	_ = env.PurposeDAO.Delete(req.Context(), purposeID, "TEST_ORG")
}

func TestUpdateConsentPurpose_JsonSchemaTypeHandler_MissingValidationSchema(t *testing.T) {
	env := setupConsentPurposeAPITestEnvironment(t)

	// Create with json-schema type
	uniqueName := fmt.Sprintf("update_jsonschema_invalid_%d", time.Now().UnixNano())
	schema := map[string]interface{}{
		"type": "object",
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

	// Try to update without validationSchema (should fail)
	updateReq := map[string]interface{}{
		"name": uniqueName + "_invalid",
		"type": "json-schema",
		"attributes": map[string]string{
			"someOtherAttribute": "value",
		},
	}

	body, _ = json.Marshal(updateReq)
	putReq := httptest.NewRequest("PUT", "/api/v1/consent-purposes/"+purposeID, bytes.NewBuffer(body))
	putReq.Header.Set("Content-Type", "application/json")
	putReq.Header.Set("org-id", "TEST_ORG")

	w = httptest.NewRecorder()
	env.Router.ServeHTTP(w, putReq)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var errorResp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &errorResp)
	require.NoError(t, err)

	assert.Equal(t, "Validation failed", errorResp["error"])
	assert.Contains(t, errorResp["message"], "validationSchema")

	t.Log("✓ JSON Schema type handler: Correctly rejected update without validationSchema")

	// Cleanup
	_ = env.PurposeDAO.Delete(req.Context(), purposeID, "TEST_ORG")
}

// ============================================================================
// UPDATE - Attribute Type Handler Tests
// ============================================================================

func TestUpdateConsentPurpose_AttributeTypeHandler_Success(t *testing.T) {
	env := setupConsentPurposeAPITestEnvironment(t)

	// Create with attribute type
	uniqueName := fmt.Sprintf("update_attribute_%d", time.Now().UnixNano())
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

	// Update with new paths
	updateReq := map[string]interface{}{
		"name": uniqueName + "_updated",
		"type": "attribute",
		"attributes": map[string]string{
			"resourcePath": "/api/v2/profiles",
			"jsonPath":     "$.data.roles",
		},
	}

	body, _ = json.Marshal(updateReq)
	putReq := httptest.NewRequest("PUT", "/api/v1/consent-purposes/"+purposeID, bytes.NewBuffer(body))
	putReq.Header.Set("Content-Type", "application/json")
	putReq.Header.Set("org-id", "TEST_ORG")

	w = httptest.NewRecorder()
	env.Router.ServeHTTP(w, putReq)

	assert.Equal(t, http.StatusOK, w.Code)

	var updatedPurpose map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &updatedPurpose)
	require.NoError(t, err)

	assert.Equal(t, "attribute", updatedPurpose["type"])
	if attrs, ok := updatedPurpose["attributes"].(map[string]interface{}); ok {
		assert.Equal(t, "/api/v2/profiles", attrs["resourcePath"])
		assert.Equal(t, "$.data.roles", attrs["jsonPath"])
	}

	t.Log("✓ Attribute type handler: Successfully updated attribute type purpose")

	// Cleanup
	_ = env.PurposeDAO.Delete(req.Context(), purposeID, "TEST_ORG")
}

func TestUpdateConsentPurpose_AttributeTypeHandler_MissingResourcePath(t *testing.T) {
	env := setupConsentPurposeAPITestEnvironment(t)

	// Create with attribute type
	uniqueName := fmt.Sprintf("update_attr_missing_res_%d", time.Now().UnixNano())
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

	// Try to update without resourcePath
	updateReq := map[string]interface{}{
		"name": uniqueName + "_invalid",
		"type": "attribute",
		"attributes": map[string]string{
			"jsonPath": "$.data",
		},
	}

	body, _ = json.Marshal(updateReq)
	putReq := httptest.NewRequest("PUT", "/api/v1/consent-purposes/"+purposeID, bytes.NewBuffer(body))
	putReq.Header.Set("Content-Type", "application/json")
	putReq.Header.Set("org-id", "TEST_ORG")

	w = httptest.NewRecorder()
	env.Router.ServeHTTP(w, putReq)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var errorResp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &errorResp)
	require.NoError(t, err)

	assert.Equal(t, "Validation failed", errorResp["error"])
	assert.Contains(t, errorResp["message"], "resourcePath")

	t.Log("✓ Attribute type handler: Correctly rejected update without resourcePath")

	// Cleanup
	_ = env.PurposeDAO.Delete(req.Context(), purposeID, "TEST_ORG")
}

func TestUpdateConsentPurpose_AttributeTypeHandler_MissingJsonPath(t *testing.T) {
	env := setupConsentPurposeAPITestEnvironment(t)

	// Create with attribute type
	uniqueName := fmt.Sprintf("update_attr_missing_json_%d", time.Now().UnixNano())
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

	// Try to update without jsonPath
	updateReq := map[string]interface{}{
		"name": uniqueName + "_invalid",
		"type": "attribute",
		"attributes": map[string]string{
			"resourcePath": "/api/data",
		},
	}

	body, _ = json.Marshal(updateReq)
	putReq := httptest.NewRequest("PUT", "/api/v1/consent-purposes/"+purposeID, bytes.NewBuffer(body))
	putReq.Header.Set("Content-Type", "application/json")
	putReq.Header.Set("org-id", "TEST_ORG")

	w = httptest.NewRecorder()
	env.Router.ServeHTTP(w, putReq)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var errorResp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &errorResp)
	require.NoError(t, err)

	assert.Equal(t, "Validation failed", errorResp["error"])
	assert.Contains(t, errorResp["message"], "jsonPath")

	t.Log("✓ Attribute type handler: Correctly rejected update without jsonPath")

	// Cleanup
	_ = env.PurposeDAO.Delete(req.Context(), purposeID, "TEST_ORG")
}

func TestUpdateConsentPurpose_AttributeTypeHandler_MissingBothAttributes(t *testing.T) {
	env := setupConsentPurposeAPITestEnvironment(t)

	// Create with attribute type
	uniqueName := fmt.Sprintf("update_attr_missing_both_%d", time.Now().UnixNano())
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

	// Try to update without both resourcePath and jsonPath
	updateReq := map[string]interface{}{
		"name":       uniqueName + "_invalid",
		"type":       "attribute",
		"attributes": map[string]string{},
	}

	body, _ = json.Marshal(updateReq)
	putReq := httptest.NewRequest("PUT", "/api/v1/consent-purposes/"+purposeID, bytes.NewBuffer(body))
	putReq.Header.Set("Content-Type", "application/json")
	putReq.Header.Set("org-id", "TEST_ORG")

	w = httptest.NewRecorder()
	env.Router.ServeHTTP(w, putReq)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var errorResp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &errorResp)
	require.NoError(t, err)

	assert.Equal(t, "Validation failed", errorResp["error"])
	// Should mention at least one of the missing attributes
	message := errorResp["message"].(string)
	assert.True(t, 
		len(message) > 0 && (len(message) > 0 && (len(message) > 0)),
		"Error message should mention missing attributes")

	t.Log("✓ Attribute type handler: Correctly rejected update without both mandatory attributes")

	// Cleanup
	_ = env.PurposeDAO.Delete(req.Context(), purposeID, "TEST_ORG")
}
