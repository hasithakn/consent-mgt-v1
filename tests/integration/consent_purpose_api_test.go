package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wso2/consent-management-api/internal/config"
	"github.com/wso2/consent-management-api/internal/dao"
	"github.com/wso2/consent-management-api/internal/database"
	"github.com/wso2/consent-management-api/internal/models"
	"github.com/wso2/consent-management-api/internal/router"
	"github.com/wso2/consent-management-api/internal/service"

	"github.com/sirupsen/logrus"
)

// TestConsentPurposeAPIEnvironment holds test dependencies for API tests
type TestConsentPurposeAPIEnvironment struct {
	Router         http.Handler
	PurposeService *service.ConsentPurposeService
	PurposeDAO     *dao.ConsentPurposeDAO
}

// setupConsentPurposeAPITestEnvironment initializes test environment for API tests
func setupConsentPurposeAPITestEnvironment(t *testing.T) *TestConsentPurposeAPIEnvironment {
	// Load configuration
	cfg, err := config.Load("../../configs/config.yaml")
	require.NoError(t, err, "Failed to load config")

	// Initialize logger
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel) // Reduce noise in tests

	// Initialize database
	db, err := database.Initialize(&cfg.Database, logger)
	require.NoError(t, err, "Failed to initialize database")

	// Initialize DAOs
	consentDAO := dao.NewConsentDAO(db)
	statusAuditDAO := dao.NewStatusAuditDAO(db)
	attributeDAO := dao.NewConsentAttributeDAO(db)
	authResourceDAO := dao.NewAuthResourceDAO(db)
	purposeDAO := dao.NewConsentPurposeDAO(db.DB)

	// Initialize services
	consentService := service.NewConsentService(
		consentDAO,
		statusAuditDAO,
		attributeDAO,
		authResourceDAO,
		purposeDAO,
		db,
		logger,
	)

	authResourceService := service.NewAuthResourceService(
		authResourceDAO,
		consentDAO,
		db,
		logger,
	)

	purposeService := service.NewConsentPurposeService(
		purposeDAO,
		consentDAO,
		db.DB,
		logger,
	)

	// Setup router (pass nil for extension client in tests)
	ginRouter := router.SetupRouter(consentService, authResourceService, purposeService, nil)

	return &TestConsentPurposeAPIEnvironment{
		Router:         ginRouter,
		PurposeService: purposeService,
		PurposeDAO:     purposeDAO,
	}
}

// TestCreateConsentPurposes_SinglePurpose tests creating a single consent purpose
func TestCreateConsentPurposes_SinglePurpose(t *testing.T) {
	env := setupConsentPurposeAPITestEnvironment(t)

	requestBody := []map[string]interface{}{
		{
			"name":        "readAccountBasic",
			"description": "Allows reading basic account information.",
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
		},
		{
			"name":        "readAccountDetailed",
			"description": "Allows reading detailed account information.",
		},
		{
			"name":        "readTransactions",
			"description": "Allows reading transaction history.",
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

	assert.Contains(t, response, "code")
	assert.Contains(t, response, "message")
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

	assert.Contains(t, response, "code")
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

	assert.Contains(t, response, "code")
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

// TestGetConsentPurpose_Success tests retrieving a specific consent purpose
func TestGetConsentPurpose_Success(t *testing.T) {
	env := setupConsentPurposeAPITestEnvironment(t)

	// First create a purpose
	createReq := []map[string]interface{}{
		{
			"name":        "TestPurpose",
			"description": "Test Description",
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
		{"name": "Purpose1", "description": "Desc1"},
		{"name": "Purpose2", "description": "Desc2"},
		{"name": "Purpose3", "description": "Desc3"},
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
		{"name": "PurposeA"},
		{"name": "PurposeB"},
		{"name": "PurposeC"},
		{"name": "PurposeD"},
		{"name": "PurposeE"},
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
	t.Logf("✓ Pagination working: got %d purposes with limit=2", len(purposes)) // Cleanup
	for _, purposeID := range purposeIDs {
		_ = env.PurposeDAO.Delete(req.Context(), purposeID, "TEST_ORG")
	}
}

// TestUpdateConsentPurpose_Success tests updating a consent purpose
func TestUpdateConsentPurpose_Success(t *testing.T) {
	env := setupConsentPurposeAPITestEnvironment(t)

	// Create a purpose
	createReq := []map[string]interface{}{
		{"name": "OriginalName", "description": "Original Description"},
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

	// Update the purpose
	updateReq := map[string]interface{}{
		"name":        "UpdatedName",
		"description": "Updated Description",
	}

	body, _ = json.Marshal(updateReq)
	putReq := httptest.NewRequest("PUT", "/api/v1/consent-purposes/"+purposeID, bytes.NewBuffer(body))
	putReq.Header.Set("Content-Type", "application/json")
	putReq.Header.Set("org-id", "TEST_ORG")

	w = httptest.NewRecorder()
	env.Router.ServeHTTP(w, putReq)

	assert.Equal(t, http.StatusOK, w.Code)

	var updatedPurpose map[string]interface{}
	err2 := json.Unmarshal(w.Body.Bytes(), &updatedPurpose)
	require.NoError(t, err2)

	assert.Equal(t, "UpdatedName", updatedPurpose["name"])
	assert.Equal(t, "Updated Description", updatedPurpose["description"])

	t.Log("✓ Successfully updated purpose")

	// Cleanup
	_ = env.PurposeDAO.Delete(req.Context(), purposeID, "TEST_ORG")
}

// TestUpdateConsentPurpose_NotFound tests updating non-existent purpose
func TestUpdateConsentPurpose_NotFound(t *testing.T) {
	env := setupConsentPurposeAPITestEnvironment(t)

	updateReq := map[string]interface{}{
		"name": "UpdatedName",
	}

	body, _ := json.Marshal(updateReq)
	req := httptest.NewRequest("PUT", "/api/v1/consent-purposes/NONEXISTENT-ID", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("org-id", "TEST_ORG")

	w := httptest.NewRecorder()
	env.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	t.Log("✓ Correctly handled update of non-existent purpose")
}

// TestDeleteConsentPurpose_Success tests deleting a consent purpose
func TestDeleteConsentPurpose_Success(t *testing.T) {
	env := setupConsentPurposeAPITestEnvironment(t)

	// Create a purpose
	createReq := []map[string]interface{}{
		{"name": "ToBeDeleted", "description": "Will be deleted"},
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
