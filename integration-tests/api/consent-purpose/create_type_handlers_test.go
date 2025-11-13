package consentpurpose

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =====================================================
// STRING TYPE HANDLER TESTS
// String type has NO mandatory attributes
// =====================================================

func TestStringTypeHandler_WithNoAttributes(t *testing.T) {
	env := setupConsentPurposeAPITestEnvironment(t)

	requestBody := []map[string]interface{}{
		{
			"name":        "string_handler_test_no_attrs",
			"description": "String type with no attributes",
			"type":        "string",
			"attributes":  map[string]string{}, // Empty is valid for string type
		},
	}

	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("POST", "/api/v1/consent-purposes", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("org-id", "TEST_ORG")
	req.Header.Set("client-id", "TEST_CLIENT")

	w := httptest.NewRecorder()
	env.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	data := response["data"].([]interface{})
	purpose := data[0].(map[string]interface{})
	
	assert.Equal(t, "string", purpose["type"])
	t.Log("✓ String type handler correctly allows creation with no attributes")

	// Cleanup
	purposeID := purpose["id"].(string)
	_ = env.PurposeDAO.Delete(req.Context(), purposeID, "TEST_ORG")
}

func TestStringTypeHandler_WithOptionalAttributes(t *testing.T) {
	env := setupConsentPurposeAPITestEnvironment(t)

	requestBody := []map[string]interface{}{
		{
			"name":        "string_handler_test_optional_attrs",
			"description": "String type with optional attributes",
			"type":        "string",
			"attributes": map[string]string{
				"value":        "custom:value",
				"resourcePath": "/accounts",
				"jsonPath":     "$.data.amount",
			},
		},
	}

	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("POST", "/api/v1/consent-purposes", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("org-id", "TEST_ORG")
	req.Header.Set("client-id", "TEST_CLIENT")

	w := httptest.NewRecorder()
	env.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	data := response["data"].([]interface{})
	purpose := data[0].(map[string]interface{})
	
	assert.Equal(t, "string", purpose["type"])
	
	// Check optional attributes were stored
	if attrs, ok := purpose["attributes"].(map[string]interface{}); ok {
		assert.Equal(t, "custom:value", attrs["value"])
		assert.Equal(t, "/accounts", attrs["resourcePath"])
		assert.Equal(t, "$.data.amount", attrs["jsonPath"])
	}
	
	t.Log("✓ String type handler correctly allows optional attributes")

	// Cleanup
	purposeID := purpose["id"].(string)
	_ = env.PurposeDAO.Delete(req.Context(), purposeID, "TEST_ORG")
}

// =====================================================
// JSON-SCHEMA TYPE HANDLER TESTS
// JSON-Schema type REQUIRES validationSchema attribute
// =====================================================

func TestJsonSchemaTypeHandler_ValidSchema(t *testing.T) {
	env := setupConsentPurposeAPITestEnvironment(t)

	schemaJSON := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"accountNumber": map[string]interface{}{
				"type": "string",
			},
			"balance": map[string]interface{}{
				"type": "number",
			},
		},
		"required": []string{"accountNumber"},
	}
	schemaBytes, _ := json.Marshal(schemaJSON)

	requestBody := []map[string]interface{}{
		{
			"name":        "json_schema_handler_test_valid",
			"description": "JSON schema type with valid schema",
			"type":        "json-schema",
			"attributes": map[string]string{
				"validationSchema": string(schemaBytes),
			},
		},
	}

	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("POST", "/api/v1/consent-purposes", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("org-id", "TEST_ORG")
	req.Header.Set("client-id", "TEST_CLIENT")

	w := httptest.NewRecorder()
	env.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	data := response["data"].([]interface{})
	purpose := data[0].(map[string]interface{})
	
	assert.Equal(t, "json-schema", purpose["type"])
	
	// Check validationSchema was stored
	if attrs, ok := purpose["attributes"].(map[string]interface{}); ok {
		assert.Contains(t, attrs, "validationSchema")
		// Verify it's still valid JSON
		var schema map[string]interface{}
		err := json.Unmarshal([]byte(attrs["validationSchema"].(string)), &schema)
		assert.NoError(t, err)
	}
	
	t.Log("✓ JSON-Schema type handler correctly accepts valid schema")

	// Cleanup
	purposeID := purpose["id"].(string)
	_ = env.PurposeDAO.Delete(req.Context(), purposeID, "TEST_ORG")
}

func TestJsonSchemaTypeHandler_MissingValidationSchema(t *testing.T) {
	env := setupConsentPurposeAPITestEnvironment(t)

	requestBody := []map[string]interface{}{
		{
			"name":        "json_schema_handler_test_missing_schema",
			"description": "JSON schema type without required validationSchema",
			"type":        "json-schema",
			"attributes":  map[string]string{}, // Missing validationSchema
		},
	}

	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("POST", "/api/v1/consent-purposes", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("org-id", "TEST_ORG")
	req.Header.Set("client-id", "TEST_CLIENT")

	w := httptest.NewRecorder()
	env.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var errorResp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &errorResp)
	require.NoError(t, err)

	assert.Equal(t, "Validation failed", errorResp["error"])
	assert.Contains(t, errorResp["message"], "validationSchema is required for json-schema type")
	
	t.Log("✓ JSON-Schema type handler correctly rejects missing validationSchema")
}

func TestJsonSchemaTypeHandler_InvalidJSONInValidationSchema(t *testing.T) {
	env := setupConsentPurposeAPITestEnvironment(t)

	requestBody := []map[string]interface{}{
		{
			"name":        "json_schema_handler_test_invalid_json",
			"description": "JSON schema type with invalid JSON in validationSchema",
			"type":        "json-schema",
			"attributes": map[string]string{
				"validationSchema": `{invalid json}`, // Invalid JSON
			},
		},
	}

	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("POST", "/api/v1/consent-purposes", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("org-id", "TEST_ORG")
	req.Header.Set("client-id", "TEST_CLIENT")

	w := httptest.NewRecorder()
	env.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var errorResp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &errorResp)
	require.NoError(t, err)

	assert.Equal(t, "Validation failed", errorResp["error"])
	assert.Contains(t, errorResp["message"], "validationSchema must be valid JSON")
	
	t.Log("✓ JSON-Schema type handler correctly rejects invalid JSON")
}

func TestJsonSchemaTypeHandler_EmptyValidationSchema(t *testing.T) {
	env := setupConsentPurposeAPITestEnvironment(t)

	requestBody := []map[string]interface{}{
		{
			"name":        "json_schema_handler_test_empty_schema",
			"description": "JSON schema type with empty validationSchema",
			"type":        "json-schema",
			"attributes": map[string]string{
				"validationSchema": "", // Empty string
			},
		},
	}

	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("POST", "/api/v1/consent-purposes", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("org-id", "TEST_ORG")
	req.Header.Set("client-id", "TEST_CLIENT")

	w := httptest.NewRecorder()
	env.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var errorResp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &errorResp)
	require.NoError(t, err)

	assert.Equal(t, "Validation failed", errorResp["error"])
	assert.Contains(t, errorResp["message"], "validationSchema is required for json-schema type")
	
	t.Log("✓ JSON-Schema type handler correctly rejects empty validationSchema")
}

// =====================================================
// ATTRIBUTE TYPE HANDLER TESTS
// Attribute type REQUIRES resourcePath AND jsonPath
// =====================================================

func TestAttributeTypeHandler_ValidAttributes(t *testing.T) {
	env := setupConsentPurposeAPITestEnvironment(t)

	requestBody := []map[string]interface{}{
		{
			"name":        "attribute_handler_test_valid",
			"description": "Attribute type with valid required attributes",
			"type":        "attribute",
			"attributes": map[string]string{
				"resourcePath": "/accounts/{accountId}",
				"jsonPath":     "$.Data.Account.AccountNumber",
			},
		},
	}

	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("POST", "/api/v1/consent-purposes", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("org-id", "TEST_ORG")
	req.Header.Set("client-id", "TEST_CLIENT")

	w := httptest.NewRecorder()
	env.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	data := response["data"].([]interface{})
	purpose := data[0].(map[string]interface{})
	
	assert.Equal(t, "attribute", purpose["type"])
	
	// Check required attributes were stored
	if attrs, ok := purpose["attributes"].(map[string]interface{}); ok {
		assert.Equal(t, "/accounts/{accountId}", attrs["resourcePath"])
		assert.Equal(t, "$.Data.Account.AccountNumber", attrs["jsonPath"])
	}
	
	t.Log("✓ Attribute type handler correctly accepts valid resourcePath and jsonPath")

	// Cleanup
	purposeID := purpose["id"].(string)
	_ = env.PurposeDAO.Delete(req.Context(), purposeID, "TEST_ORG")
}

func TestAttributeTypeHandler_WithOptionalValidationSchema(t *testing.T) {
	env := setupConsentPurposeAPITestEnvironment(t)

	schemaJSON := map[string]interface{}{
		"type": "string",
		"pattern": "^[0-9]{10}$",
	}
	schemaBytes, _ := json.Marshal(schemaJSON)

	requestBody := []map[string]interface{}{
		{
			"name":        "attribute_handler_test_with_schema",
			"description": "Attribute type with optional validationSchema",
			"type":        "attribute",
			"attributes": map[string]string{
				"resourcePath":     "/accounts/{accountId}",
				"jsonPath":         "$.Data.Account.AccountNumber",
				"validationSchema": string(schemaBytes), // Optional for attribute type
			},
		},
	}

	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("POST", "/api/v1/consent-purposes", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("org-id", "TEST_ORG")
	req.Header.Set("client-id", "TEST_CLIENT")

	w := httptest.NewRecorder()
	env.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	data := response["data"].([]interface{})
	purpose := data[0].(map[string]interface{})
	
	assert.Equal(t, "attribute", purpose["type"])
	
	// Check all attributes were stored
	if attrs, ok := purpose["attributes"].(map[string]interface{}); ok {
		assert.Equal(t, "/accounts/{accountId}", attrs["resourcePath"])
		assert.Equal(t, "$.Data.Account.AccountNumber", attrs["jsonPath"])
		assert.Contains(t, attrs, "validationSchema")
	}
	
	t.Log("✓ Attribute type handler correctly accepts optional validationSchema")

	// Cleanup
	purposeID := purpose["id"].(string)
	_ = env.PurposeDAO.Delete(req.Context(), purposeID, "TEST_ORG")
}

func TestAttributeTypeHandler_MissingResourcePath(t *testing.T) {
	env := setupConsentPurposeAPITestEnvironment(t)

	requestBody := []map[string]interface{}{
		{
			"name":        "attribute_handler_test_missing_resource_path",
			"description": "Attribute type missing required resourcePath",
			"type":        "attribute",
			"attributes": map[string]string{
				"jsonPath": "$.Data.Account.AccountNumber",
				// Missing resourcePath
			},
		},
	}

	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("POST", "/api/v1/consent-purposes", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("org-id", "TEST_ORG")
	req.Header.Set("client-id", "TEST_CLIENT")

	w := httptest.NewRecorder()
	env.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var errorResp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &errorResp)
	require.NoError(t, err)

	assert.Equal(t, "Validation failed", errorResp["error"])
	assert.Contains(t, errorResp["message"], "resourcePath is required for attribute type")
	
	t.Log("✓ Attribute type handler correctly rejects missing resourcePath")
}

func TestAttributeTypeHandler_MissingJsonPath(t *testing.T) {
	env := setupConsentPurposeAPITestEnvironment(t)

	requestBody := []map[string]interface{}{
		{
			"name":        "attribute_handler_test_missing_json_path",
			"description": "Attribute type missing required jsonPath",
			"type":        "attribute",
			"attributes": map[string]string{
				"resourcePath": "/accounts/{accountId}",
				// Missing jsonPath
			},
		},
	}

	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("POST", "/api/v1/consent-purposes", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("org-id", "TEST_ORG")
	req.Header.Set("client-id", "TEST_CLIENT")

	w := httptest.NewRecorder()
	env.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var errorResp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &errorResp)
	require.NoError(t, err)

	assert.Equal(t, "Validation failed", errorResp["error"])
	assert.Contains(t, errorResp["message"], "jsonPath is required for attribute type")
	
	t.Log("✓ Attribute type handler correctly rejects missing jsonPath")
}

func TestAttributeTypeHandler_MissingBothRequiredAttributes(t *testing.T) {
	env := setupConsentPurposeAPITestEnvironment(t)

	requestBody := []map[string]interface{}{
		{
			"name":        "attribute_handler_test_missing_both",
			"description": "Attribute type missing both required attributes",
			"type":        "attribute",
			"attributes":  map[string]string{}, // Missing both resourcePath and jsonPath
		},
	}

	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("POST", "/api/v1/consent-purposes", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("org-id", "TEST_ORG")
	req.Header.Set("client-id", "TEST_CLIENT")

	w := httptest.NewRecorder()
	env.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var errorResp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &errorResp)
	require.NoError(t, err)

	assert.Equal(t, "Validation failed", errorResp["error"])
	// Should contain both error messages
	assert.Contains(t, errorResp["message"], "resourcePath is required for attribute type")
	assert.Contains(t, errorResp["message"], "jsonPath is required for attribute type")
	
	t.Log("✓ Attribute type handler correctly rejects when both required attributes are missing")
}

func TestAttributeTypeHandler_EmptyResourcePath(t *testing.T) {
	env := setupConsentPurposeAPITestEnvironment(t)

	requestBody := []map[string]interface{}{
		{
			"name":        "attribute_handler_test_empty_resource_path",
			"description": "Attribute type with empty resourcePath",
			"type":        "attribute",
			"attributes": map[string]string{
				"resourcePath": "", // Empty
				"jsonPath":     "$.Data.Account.AccountNumber",
			},
		},
	}

	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("POST", "/api/v1/consent-purposes", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("org-id", "TEST_ORG")
	req.Header.Set("client-id", "TEST_CLIENT")

	w := httptest.NewRecorder()
	env.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var errorResp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &errorResp)
	require.NoError(t, err)

	assert.Equal(t, "Validation failed", errorResp["error"])
	assert.Contains(t, errorResp["message"], "resourcePath is required for attribute type")
	
	t.Log("✓ Attribute type handler correctly rejects empty resourcePath")
}

func TestAttributeTypeHandler_EmptyJsonPath(t *testing.T) {
	env := setupConsentPurposeAPITestEnvironment(t)

	requestBody := []map[string]interface{}{
		{
			"name":        "attribute_handler_test_empty_json_path",
			"description": "Attribute type with empty jsonPath",
			"type":        "attribute",
			"attributes": map[string]string{
				"resourcePath": "/accounts/{accountId}",
				"jsonPath":     "", // Empty
			},
		},
	}

	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("POST", "/api/v1/consent-purposes", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("org-id", "TEST_ORG")
	req.Header.Set("client-id", "TEST_CLIENT")

	w := httptest.NewRecorder()
	env.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var errorResp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &errorResp)
	require.NoError(t, err)

	assert.Equal(t, "Validation failed", errorResp["error"])
	assert.Contains(t, errorResp["message"], "jsonPath is required for attribute type")
	
	t.Log("✓ Attribute type handler correctly rejects empty jsonPath")
}
