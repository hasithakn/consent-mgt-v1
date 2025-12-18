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

	"github.com/wso2/consent-management-api/internal/models"
)

// ============================================================================
// READ - String Type Handler Tests
// ============================================================================

func TestGetConsentPurpose_StringTypeHandler_Success(t *testing.T) {
	env := setupConsentPurposeAPITestEnvironment(t)

	// Create a purpose with string type
	uniqueName := fmt.Sprintf("read_string_%d", time.Now().UnixNano())
	createReq := []map[string]interface{}{
		{
			"name":        uniqueName,
			"description": "String type purpose",
			"type":        "string",
			"attributes": map[string]string{
				"value": "test:string:value",
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

	// Get the purpose
	getReq := httptest.NewRequest("GET", "/api/v1/consent-purposes/"+purposeID, nil)
	getReq.Header.Set("org-id", "TEST_ORG")

	w = httptest.NewRecorder()
	env.Router.ServeHTTP(w, getReq)

	assert.Equal(t, http.StatusOK, w.Code)

	var retrievedPurpose map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &retrievedPurpose)
	require.NoError(t, err)

	assert.Equal(t, purposeID, retrievedPurpose["id"])
	assert.Equal(t, uniqueName, retrievedPurpose["name"])
	assert.Equal(t, "string", retrievedPurpose["type"])

	if attrs, ok := retrievedPurpose["attributes"].(map[string]interface{}); ok {
		assert.Equal(t, "test:string:value", attrs["value"])
	}

	t.Log("✓ String type handler: Successfully retrieved string type purpose")

	// Cleanup
	_ = env.PurposeDAO.Delete(req.Context(), purposeID, "TEST_ORG")
}

func TestListConsentPurposes_StringTypeHandler_IncludedInList(t *testing.T) {
	env := setupConsentPurposeAPITestEnvironment(t)

	// Create multiple string type purposes
	uniquePrefix := fmt.Sprintf("list_string_%d", time.Now().UnixNano())
	createReq := []map[string]interface{}{
		{
			"name": uniquePrefix + "_1",
			"type": "string",
			"attributes": map[string]string{
				"value": "value:1",
			},
		},
		{
			"name": uniquePrefix + "_2",
			"type": "string",
			"attributes": map[string]string{
				"value": "value:2",
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

	purposeIDs := make([]string, 0, 2)
	for _, item := range data {
		purpose := item.(map[string]interface{})
		purposeIDs = append(purposeIDs, purpose["id"].(string))
	}

	// List purposes
	listReq := httptest.NewRequest("GET", "/api/v1/consent-purposes", nil)
	listReq.Header.Set("org-id", "TEST_ORG")

	w = httptest.NewRecorder()
	env.Router.ServeHTTP(w, listReq)

	assert.Equal(t, http.StatusOK, w.Code)

	var listResponse models.ConsentPurposeListResponse
	err := json.Unmarshal(w.Body.Bytes(), &listResponse)
	require.NoError(t, err)

	// Verify our purposes are in the list with correct type and attributes
	foundCount := 0
	for _, p := range listResponse.Purposes {
		for i, id := range purposeIDs {
			if p.ID == id {
				assert.Equal(t, "string", p.Type)
				assert.NotNil(t, p.Attributes)
				assert.Equal(t, fmt.Sprintf("value:%d", i+1), p.Attributes["value"])
				foundCount++
			}
		}
	}
	assert.Equal(t, 2, foundCount)

	t.Log("✓ String type handler: String type purposes correctly listed with attributes")

	// Cleanup
	for _, purposeID := range purposeIDs {
		_ = env.PurposeDAO.Delete(req.Context(), purposeID, "TEST_ORG")
	}
}

// ============================================================================
// READ - JSON Schema Type Handler Tests
// ============================================================================

func TestGetConsentPurpose_JsonSchemaTypeHandler_Success(t *testing.T) {
	env := setupConsentPurposeAPITestEnvironment(t)

	// Create a purpose with json-schema type
	uniqueName := fmt.Sprintf("read_jsonschema_%d", time.Now().UnixNano())
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"email": map[string]string{"type": "string", "format": "email"},
		},
		"required": []string{"email"},
	}
	schemaBytes, _ := json.Marshal(schema)

	createReq := []map[string]interface{}{
		{
			"name":        uniqueName,
			"description": "JSON Schema type purpose",
			"type":        "json-schema",
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

	var createResponse map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &createResponse)
	data := createResponse["data"].([]interface{})
	purpose := data[0].(map[string]interface{})
	purposeID := purpose["id"].(string)

	// Get the purpose
	getReq := httptest.NewRequest("GET", "/api/v1/consent-purposes/"+purposeID, nil)
	getReq.Header.Set("org-id", "TEST_ORG")

	w = httptest.NewRecorder()
	env.Router.ServeHTTP(w, getReq)

	assert.Equal(t, http.StatusOK, w.Code)

	var retrievedPurpose map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &retrievedPurpose)
	require.NoError(t, err)

	assert.Equal(t, purposeID, retrievedPurpose["id"])
	assert.Equal(t, uniqueName, retrievedPurpose["name"])
	assert.Equal(t, "json-schema", retrievedPurpose["type"])

	if attrs, ok := retrievedPurpose["attributes"].(map[string]interface{}); ok {
		assert.Contains(t, attrs, "validationSchema")
		schemaStr := attrs["validationSchema"].(string)
		assert.Contains(t, schemaStr, "email")
		assert.Contains(t, schemaStr, "object")
	}

	t.Log("✓ JSON Schema type handler: Successfully retrieved json-schema type purpose with validationSchema")

	// Cleanup
	_ = env.PurposeDAO.Delete(req.Context(), purposeID, "TEST_ORG")
}

func TestListConsentPurposes_JsonSchemaTypeHandler_IncludedInList(t *testing.T) {
	env := setupConsentPurposeAPITestEnvironment(t)

	// Create json-schema type purpose
	uniqueName := fmt.Sprintf("list_jsonschema_%d", time.Now().UnixNano())
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"age": map[string]interface{}{
				"type":    "integer",
				"minimum": 0,
			},
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

	var createResponse map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &createResponse)
	data := createResponse["data"].([]interface{})
	purpose := data[0].(map[string]interface{})
	purposeID := purpose["id"].(string)

	// List purposes
	listReq := httptest.NewRequest("GET", "/api/v1/consent-purposes", nil)
	listReq.Header.Set("org-id", "TEST_ORG")

	w = httptest.NewRecorder()
	env.Router.ServeHTTP(w, listReq)

	assert.Equal(t, http.StatusOK, w.Code)

	var listResponse models.ConsentPurposeListResponse
	err := json.Unmarshal(w.Body.Bytes(), &listResponse)
	require.NoError(t, err)

	// Find our purpose
	found := false
	for _, p := range listResponse.Purposes {
		if p.ID == purposeID {
			assert.Equal(t, "json-schema", p.Type)
			assert.NotNil(t, p.Attributes)
			assert.Contains(t, p.Attributes, "validationSchema")
			found = true
			break
		}
	}
	assert.True(t, found, "json-schema purpose should be in the list")

	t.Log("✓ JSON Schema type handler: json-schema type purpose correctly listed with validationSchema")

	// Cleanup
	_ = env.PurposeDAO.Delete(req.Context(), purposeID, "TEST_ORG")
}

// ============================================================================
// READ - Attribute Type Handler Tests
// ============================================================================

func TestGetConsentPurpose_AttributeTypeHandler_Success(t *testing.T) {
	env := setupConsentPurposeAPITestEnvironment(t)

	// Create a purpose with attribute type
	uniqueName := fmt.Sprintf("read_attribute_%d", time.Now().UnixNano())
	createReq := []map[string]interface{}{
		{
			"name":        uniqueName,
			"description": "Attribute type purpose",
			"type":        "attribute",
			"attributes": map[string]string{
				"resourcePath": "/api/v1/users",
				"jsonPath":     "$.permissions.scopes",
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

	// Get the purpose
	getReq := httptest.NewRequest("GET", "/api/v1/consent-purposes/"+purposeID, nil)
	getReq.Header.Set("org-id", "TEST_ORG")

	w = httptest.NewRecorder()
	env.Router.ServeHTTP(w, getReq)

	assert.Equal(t, http.StatusOK, w.Code)

	var retrievedPurpose map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &retrievedPurpose)
	require.NoError(t, err)

	assert.Equal(t, purposeID, retrievedPurpose["id"])
	assert.Equal(t, uniqueName, retrievedPurpose["name"])
	assert.Equal(t, "attribute", retrievedPurpose["type"])

	if attrs, ok := retrievedPurpose["attributes"].(map[string]interface{}); ok {
		assert.Equal(t, "/api/v1/users", attrs["resourcePath"])
		assert.Equal(t, "$.permissions.scopes", attrs["jsonPath"])
	}

	t.Log("✓ Attribute type handler: Successfully retrieved attribute type purpose with resourcePath and jsonPath")

	// Cleanup
	_ = env.PurposeDAO.Delete(req.Context(), purposeID, "TEST_ORG")
}

func TestListConsentPurposes_AttributeTypeHandler_IncludedInList(t *testing.T) {
	env := setupConsentPurposeAPITestEnvironment(t)

	// Create attribute type purpose
	uniqueName := fmt.Sprintf("list_attribute_%d", time.Now().UnixNano())
	createReq := []map[string]interface{}{
		{
			"name": uniqueName,
			"type": "attribute",
			"attributes": map[string]string{
				"resourcePath": "/api/v2/data",
				"jsonPath":     "$.user.roles",
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

	// List purposes
	listReq := httptest.NewRequest("GET", "/api/v1/consent-purposes", nil)
	listReq.Header.Set("org-id", "TEST_ORG")

	w = httptest.NewRecorder()
	env.Router.ServeHTTP(w, listReq)

	assert.Equal(t, http.StatusOK, w.Code)

	var listResponse models.ConsentPurposeListResponse
	err := json.Unmarshal(w.Body.Bytes(), &listResponse)
	require.NoError(t, err)

	// Find our purpose
	found := false
	for _, p := range listResponse.Purposes {
		if p.ID == purposeID {
			assert.Equal(t, "attribute", p.Type)
			assert.NotNil(t, p.Attributes)
			assert.Equal(t, "/api/v2/data", p.Attributes["resourcePath"])
			assert.Equal(t, "$.user.roles", p.Attributes["jsonPath"])
			found = true
			break
		}
	}
	assert.True(t, found, "attribute purpose should be in the list")

	t.Log("✓ Attribute type handler: attribute type purpose correctly listed with resourcePath and jsonPath")

	// Cleanup
	_ = env.PurposeDAO.Delete(req.Context(), purposeID, "TEST_ORG")
}

// ============================================================================
// READ - Mixed Types in List Test
// ============================================================================

func TestListConsentPurposes_AllTypeHandlers_MixedTypes(t *testing.T) {
	env := setupConsentPurposeAPITestEnvironment(t)

	uniquePrefix := fmt.Sprintf("mixed_types_%d", time.Now().UnixNano())

	// Create purposes of all three types
	schema := map[string]interface{}{
		"type": "object",
	}
	schemaBytes, _ := json.Marshal(schema)

	createReq := []map[string]interface{}{
		{
			"name": uniquePrefix + "_string",
			"type": "string",
			"attributes": map[string]string{
				"value": "string:value",
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

	var createResponse map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &createResponse)
	data := createResponse["data"].([]interface{})

	purposeIDs := make([]string, 0, 3)
	for _, item := range data {
		purpose := item.(map[string]interface{})
		purposeIDs = append(purposeIDs, purpose["id"].(string))
	}

	// List purposes
	listReq := httptest.NewRequest("GET", "/api/v1/consent-purposes", nil)
	listReq.Header.Set("org-id", "TEST_ORG")

	w = httptest.NewRecorder()
	env.Router.ServeHTTP(w, listReq)

	assert.Equal(t, http.StatusOK, w.Code)

	var listResponse models.ConsentPurposeListResponse
	err := json.Unmarshal(w.Body.Bytes(), &listResponse)
	require.NoError(t, err)

	// Verify all three types are correctly returned with their specific attributes
	typesFound := make(map[string]bool)
	for _, p := range listResponse.Purposes {
		for _, id := range purposeIDs {
			if p.ID == id {
				typesFound[p.Type] = true

				switch p.Type {
				case "string":
					assert.Contains(t, p.Attributes, "value")
				case "json-schema":
					assert.Contains(t, p.Attributes, "validationSchema")
				case "attribute":
					assert.Contains(t, p.Attributes, "resourcePath")
					assert.Contains(t, p.Attributes, "jsonPath")
				}
			}
		}
	}

	assert.True(t, typesFound["string"], "string type should be found")
	assert.True(t, typesFound["json-schema"], "json-schema type should be found")
	assert.True(t, typesFound["attribute"], "attribute type should be found")

	t.Log("✓ All type handlers: All three types correctly returned in list with their specific attributes")

	// Cleanup
	for _, purposeID := range purposeIDs {
		_ = env.PurposeDAO.Delete(req.Context(), purposeID, "TEST_ORG")
	}
}
