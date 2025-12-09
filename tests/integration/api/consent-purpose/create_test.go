package consentpurpose

import (
	"bytes"
	"context"
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

func TestCreateConsentPurposes_SinglePurpose(t *testing.T) {
	env := setupConsentPurposeAPITestEnvironment(t)

	requestBody := []map[string]interface{}{
		{
			"name":        "readAccountBasic",
			"description": "Allows reading basic account information.",
			"type":        "string",
			"attributes": map[string]string{
				"value": "account:read:basic",
			},
		},
	}

	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("POST", "/api/v1/consent-purposes", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("org-id", "TEST_ORG")

	w := httptest.NewRecorder()
	env.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response, "data")
	assert.Contains(t, response, "message")

	data := response["data"].([]interface{})
	assert.Equal(t, 1, len(data))

	purpose := data[0].(map[string]interface{})
	assert.Contains(t, purpose, "id")
	assert.Equal(t, "readAccountBasic", purpose["name"])
	assert.Equal(t, "Allows reading basic account information.", purpose["description"])
	assert.Equal(t, "string", purpose["type"])

	// Check attributes
	if attrs, ok := purpose["attributes"].(map[string]interface{}); ok {
		assert.Equal(t, "account:read:basic", attrs["value"])
	}

	t.Logf("✓ Successfully created purpose with ID: %s", purpose["id"])

	// Cleanup
	purposeID := purpose["id"].(string)
	_ = env.PurposeDAO.Delete(req.Context(), purposeID, "TEST_ORG")
}

// TestCreateConsentPurposes_MultiplePurposes tests batch creation of consent purposes
func TestCreateConsentPurposes_MultiplePurposes(t *testing.T) {
	env := setupConsentPurposeAPITestEnvironment(t)

	requestBody := []map[string]interface{}{
		{
			"name":        "readAccountBasic",
			"description": "Allows reading basic account information.",
			"type":        "string",
			"attributes": map[string]string{
				"value": "account:read:basic",
			},
		},
		{
			"name":        "readAccountDetailed",
			"description": "Allows reading detailed account information.",
			"type":        "string",
			"attributes": map[string]string{
				"value": "account:read:detailed",
			},
		},
		{
			"name":        "readTransactions",
			"description": "Allows reading transaction history.",
			"type":        "string",
			"attributes": map[string]string{
				"value": "account:read:transactions",
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
	assert.Equal(t, 3, len(data))

	// Verify all purposes were created
	purposeIDs := make([]string, 0, 3)
	for i, item := range data {
		purpose := item.(map[string]interface{})
		assert.Contains(t, purpose, "id")
		assert.Equal(t, requestBody[i]["name"], purpose["name"])
		assert.Equal(t, requestBody[i]["description"], purpose["description"])
		assert.Equal(t, requestBody[i]["type"], purpose["type"])
		purposeIDs = append(purposeIDs, purpose["id"].(string))
	}

	t.Logf("✓ Successfully created %d purposes", len(data))

	// Cleanup
	for _, purposeID := range purposeIDs {
		_ = env.PurposeDAO.Delete(req.Context(), purposeID, "TEST_ORG")
	}
}

// TestCreateConsentPurposes_WithoutDescription tests creating purpose without description
func TestCreateConsentPurposes_WithoutDescription(t *testing.T) {
	env := setupConsentPurposeAPITestEnvironment(t)

	requestBody := []map[string]interface{}{
		{
			"name": "Marketing",
			"type": "string",
			"attributes": map[string]string{
				"value": "marketing:access",
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

	assert.Equal(t, "Marketing", purpose["name"])
	assert.Equal(t, "string", purpose["type"])

	// Check attributes
	if attrs, ok := purpose["attributes"].(map[string]interface{}); ok {
		assert.Equal(t, "marketing:access", attrs["value"])
	}

	// Description should be empty string or not present
	if desc, ok := purpose["description"]; ok {
		assert.Equal(t, "", desc)
	}

	t.Log("✓ Successfully created purpose without description")

	// Cleanup
	purposeID := purpose["id"].(string)
	_ = env.PurposeDAO.Delete(req.Context(), purposeID, "TEST_ORG")
}

// TestCreateConsentPurposes_EmptyArray tests validation for empty request array
func TestCreateConsentPurposes_EmptyArray(t *testing.T) {
	env := setupConsentPurposeAPITestEnvironment(t)

	requestBody := []map[string]interface{}{}

	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("POST", "/api/v1/consent-purposes", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("org-id", "TEST_ORG")
	req.Header.Set("client-id", "TEST_CLIENT")

	w := httptest.NewRecorder()
	env.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "BAD_REQUEST", response["code"])
	assert.Contains(t, response["message"], "Empty request")
	t.Log("✓ Correctly rejected empty array request")
}

// TestCreateConsentPurposes_MissingName tests validation for missing name field
func TestCreateConsentPurposes_MissingName(t *testing.T) {
	env := setupConsentPurposeAPITestEnvironment(t)

	requestBody := []map[string]interface{}{
		{
			"description": "Test description without name",
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

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "BAD_REQUEST", response["code"])
	assert.Contains(t, response["details"], "Name") // validation error uses capitalized field name
	t.Log("✓ Correctly rejected request with missing name")
}

// TestCreateConsentPurposes_NameTooLong tests validation for name length
func TestCreateConsentPurposes_NameTooLong(t *testing.T) {
	env := setupConsentPurposeAPITestEnvironment(t)

	// Create a name with 256 characters (max is 255)
	longName := ""
	for i := 0; i < 256; i++ {
		longName += "a"
	}

	requestBody := []map[string]interface{}{
		{
			"name":        longName,
			"description": "Test",
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

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "BAD_REQUEST", response["code"])
	t.Log("✓ Correctly rejected request with name too long")
}

// TestCreateConsentPurposes_InvalidJSON tests handling of invalid JSON
func TestCreateConsentPurposes_InvalidJSON(t *testing.T) {
	env := setupConsentPurposeAPITestEnvironment(t)

	invalidJSON := []byte(`{"invalid": json}`)

	req := httptest.NewRequest("POST", "/api/v1/consent-purposes", bytes.NewBuffer(invalidJSON))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("org-id", "TEST_ORG")
	req.Header.Set("client-id", "TEST_CLIENT")

	w := httptest.NewRecorder()
	env.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	t.Log("✓ Correctly rejected invalid JSON")
}

func TestCreateConsentPurposes_WithJSONObjectValue(t *testing.T) {
	env := setupConsentPurposeAPITestEnvironment(t)

	// For json-schema type, encode the schema as a JSON string in attributes
	schemaJSON := map[string]interface{}{
		"permissions": []string{"read", "write"},
		"scopes":      []string{"account.basic", "account.transactions"},
		"metadata": map[string]interface{}{
			"version":  "1.0",
			"required": true,
		},
	}
	schemaBytes, _ := json.Marshal(schemaJSON)

	requestBody := []map[string]interface{}{
		{
			"name":        "account_schema",
			"description": "Account access schema with permissions",
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

	w := httptest.NewRecorder()
	env.Router.ServeHTTP(w, req)

	// Debug: Print response if not 201
	if w.Code != http.StatusCreated {
		t.Logf("Response Code: %d, Body: %s", w.Code, w.Body.String())
	}

	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	data := response["data"].([]interface{})
	purpose := data[0].(map[string]interface{})

	assert.Equal(t, "account_schema", purpose["name"])
	assert.Equal(t, "json-schema", purpose["type"])

	// Verify the attributes contain the schema
	if attrs, ok := purpose["attributes"].(map[string]interface{}); ok {
		assert.Contains(t, attrs, "validationSchema")

		// Parse the schema JSON
		var parsedSchema map[string]interface{}
		err := json.Unmarshal([]byte(attrs["validationSchema"].(string)), &parsedSchema)
		assert.NoError(t, err)

		assert.Contains(t, parsedSchema, "permissions")
		assert.Contains(t, parsedSchema, "scopes")
		assert.Contains(t, parsedSchema, "metadata")
	}

	t.Log("✓ Successfully created purpose with JSON schema in attributes")

	// Cleanup
	purposeID := purpose["id"].(string)
	_ = env.PurposeDAO.Delete(req.Context(), purposeID, "TEST_ORG")
}

// TestCreateConsentPurposes_WithJSONArrayValue tests creating purpose with JSON array value
func TestCreateConsentPurposes_WithJSONArrayValue(t *testing.T) {
	env := setupConsentPurposeAPITestEnvironment(t)

	// Encode array as JSON string in attributes
	operations := []string{"create", "read", "update", "delete"}
	operationsJSON, _ := json.Marshal(operations)

	requestBody := []map[string]interface{}{
		{
			"name":        "allowed_operations",
			"description": "List of allowed operations",
			"type":        "string",
			"attributes": map[string]string{
				"operations": string(operationsJSON),
			},
		},
	}

	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("POST", "/api/v1/consent-purposes", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("org-id", "TEST_ORG")

	w := httptest.NewRecorder()
	env.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	data := response["data"].([]interface{})
	purpose := data[0].(map[string]interface{})

	assert.Equal(t, "allowed_operations", purpose["name"])
	assert.Equal(t, "string", purpose["type"])

	// Verify the attributes contain the operations array
	if attrs, ok := purpose["attributes"].(map[string]interface{}); ok {
		assert.Contains(t, attrs, "operations")

		// Parse the operations JSON
		var parsedOps []string
		err := json.Unmarshal([]byte(attrs["operations"].(string)), &parsedOps)
		assert.NoError(t, err)
		assert.Equal(t, 4, len(parsedOps))
		assert.Contains(t, parsedOps, "create")
		assert.Contains(t, parsedOps, "read")
		assert.Contains(t, parsedOps, "update")
		assert.Contains(t, parsedOps, "delete")
	}

	t.Log("✓ Successfully created purpose with JSON array in attributes")

	// Cleanup
	purposeID := purpose["id"].(string)
	_ = env.PurposeDAO.Delete(req.Context(), purposeID, "TEST_ORG")
}

// TestCreateConsentPurposes_TransactionalRollback tests that batch create is truly transactional
// If one purpose fails (e.g., duplicate name), the entire batch should be rolled back
func TestCreateConsentPurposes_TransactionalRollback(t *testing.T) {
	env := setupConsentPurposeAPITestEnvironment(t)

	// Use a unique name with timestamp to avoid conflicts
	uniqueName := fmt.Sprintf("ExistingPurpose_%d", time.Now().UnixNano())

	// First, create a purpose that will cause a duplicate conflict
	existingPurpose := []map[string]interface{}{
		{
			"name":        uniqueName,
			"description": "This purpose already exists",
			"type":        "string",
			"attributes": map[string]string{
				"value": "existing_value",
			},
		},
	}

	body, _ := json.Marshal(existingPurpose)
	req := httptest.NewRequest("POST", "/api/v1/consent-purposes", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("org-id", "TEST_ORG")

	w := httptest.NewRecorder()
	env.Router.ServeHTTP(w, req)

	// Log the response for debugging
	if w.Code != http.StatusCreated {
		t.Logf("Failed to create existing purpose. Status: %d, Body: %s", w.Code, w.Body.String())
	}

	require.Equal(t, http.StatusCreated, w.Code, "Failed to create existing purpose")

	var existingResp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &existingResp)
	require.NoError(t, err)
	existingData := existingResp["data"].([]interface{})
	existingPurposeObj := existingData[0].(map[string]interface{})
	existingID := existingPurposeObj["id"].(string)

	// Now attempt to create a batch where the 3rd purpose has a duplicate name
	// The transaction should roll back ALL insertions
	batchRequest := []map[string]interface{}{
		{
			"name":        "NewPurpose1",
			"description": "First new purpose",
			"type":        "string",
			"attributes": map[string]string{
				"value": "value1",
			},
		},
		{
			"name":        "NewPurpose2",
			"description": "Second new purpose",
			"type":        "string",
			"attributes": map[string]string{
				"value": "value2",
			},
		},
		{
			"name":        uniqueName, // This is a duplicate!
			"description": "This will cause failure",
			"type":        "string",
			"attributes": map[string]string{
				"value": "duplicate_value",
			},
		},
	}

	body, _ = json.Marshal(batchRequest)
	req = httptest.NewRequest("POST", "/api/v1/consent-purposes", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("org-id", "TEST_ORG")

	w = httptest.NewRecorder()
	env.Router.ServeHTTP(w, req)

	// Should return 400 Bad Request due to duplicate name
	assert.Equal(t, http.StatusBadRequest, w.Code, "Expected 400 due to duplicate name")

	var errorResp map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &errorResp)
	require.NoError(t, err)

	// Check error message
	if errorMsg, ok := errorResp["error"].(string); ok {
		assert.Contains(t, errorMsg, "already exists", "Error message should mention duplicate name")
	} else if details, ok := errorResp["details"].(string); ok {
		assert.Contains(t, details, "already exists", "Error details should mention duplicate name")
	} else {
		t.Logf("Error response: %v", errorResp)
	}

	t.Log("✓ Batch create correctly rejected due to duplicate name")

	// Verify that NO new purposes were created (transaction was rolled back)
	// We can check this by trying to GET the purposes that should NOT exist
	getReq := httptest.NewRequest("GET", "/api/v1/consent-purposes", nil)
	getReq.Header.Set("org-id", "TEST_ORG")

	w = httptest.NewRecorder()
	env.Router.ServeHTTP(w, getReq)
	require.Equal(t, http.StatusOK, w.Code)

	var listResponse models.ConsentPurposeListResponse
	err = json.Unmarshal(w.Body.Bytes(), &listResponse)
	require.NoError(t, err)

	purposes := listResponse.Purposes

	// Count only our test purposes - should only have the initial one
	foundNewPurpose1 := false
	foundNewPurpose2 := false
	foundExisting := false

	for _, purpose := range purposes {
		if purpose.Name == "NewPurpose1" {
			foundNewPurpose1 = true
		} else if purpose.Name == "NewPurpose2" {
			foundNewPurpose2 = true
		} else if purpose.Name == uniqueName {
			foundExisting = true
		}
	}

	// Only the existing purpose should be found
	assert.True(t, foundExisting, "ExistingPurpose should exist")
	assert.False(t, foundNewPurpose1, "NewPurpose1 should NOT exist - transaction rolled back")
	assert.False(t, foundNewPurpose2, "NewPurpose2 should NOT exist - transaction rolled back")

	t.Log("✓ Transaction correctly rolled back - no partial inserts!")

	// Cleanup
	_ = env.PurposeDAO.Delete(context.Background(), existingID, "TEST_ORG")
}

// TestCreateConsentPurposes_DuplicateWithinBatch tests duplicate detection within the same batch
func TestCreateConsentPurposes_DuplicateWithinBatch(t *testing.T) {
	env := setupConsentPurposeAPITestEnvironment(t)

	// Create a batch with duplicate names within the batch itself
	batchRequest := []map[string]interface{}{
		{
			"name":        "DuplicateName",
			"description": "First instance",
			"type":        "string",
			"attributes": map[string]string{
				"value": "value1",
			},
		},
		{
			"name":        "UniqueName",
			"description": "Unique purpose",
			"type":        "string",
			"attributes": map[string]string{
				"value": "value2",
			},
		},
		{
			"name":        "DuplicateName", // Duplicate within batch!
			"description": "Second instance",
			"type":        "string",
			"attributes": map[string]string{
				"value": "value3",
			},
		},
	}

	body, _ := json.Marshal(batchRequest)
	req := httptest.NewRequest("POST", "/api/v1/consent-purposes", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("org-id", "TEST_ORG")

	w := httptest.NewRecorder()
	env.Router.ServeHTTP(w, req)

	// Should return 400 Bad Request due to duplicate within batch
	assert.Equal(t, http.StatusBadRequest, w.Code, "Expected 400 due to duplicate within batch")

	var errorResp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &errorResp)
	require.NoError(t, err)

	// Check error message
	if errorMsg, ok := errorResp["error"].(string); ok {
		assert.Contains(t, errorMsg, "duplicate", "Error message should mention duplicate")
	} else if details, ok := errorResp["details"].(string); ok {
		assert.Contains(t, details, "duplicate", "Error details should mention duplicate")
	} else {
		t.Logf("Error response: %v", errorResp)
	}

	t.Log("✓ Batch create correctly rejected due to duplicate within batch")

	// Verify that NO purposes were created by listing all purposes
	getReq := httptest.NewRequest("GET", "/api/v1/consent-purposes", nil)
	getReq.Header.Set("org-id", "TEST_ORG")

	w = httptest.NewRecorder()
	env.Router.ServeHTTP(w, getReq)
	require.Equal(t, http.StatusOK, w.Code)

	var listResponse models.ConsentPurposeListResponse
	err = json.Unmarshal(w.Body.Bytes(), &listResponse)
	require.NoError(t, err)

	purposes := listResponse.Purposes

	// Check that none of the batch purposes exist
	for _, purpose := range purposes {
		assert.NotEqual(t, "DuplicateName", purpose.Name, "DuplicateName should not exist")
		assert.NotEqual(t, "UniqueName", purpose.Name, "UniqueName should not exist")
	}

	t.Log("✓ Transaction correctly rolled back - no partial inserts for duplicate within batch")
}
