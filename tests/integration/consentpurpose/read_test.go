package consentpurpose

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// ========================================
// GET /consent-purposes/{purposeId} Tests
// ========================================

// TestGetPurposeByID_StringType_ReturnsWithResourcePath tests retrieving a string type purpose with resourcePath
func (ts *PurposeAPITestSuite) TestGetPurposeByID_StringType_ReturnsWithResourcePath() {
	t := ts.T()

	// Create purpose
	payload := []ConsentPurposeCreateRequest{
		{
			Name:        "test_license_read_get",
			Description: "License read permission",
			Type:        "string",
			Attributes: map[string]string{
				"resourcePath": "/licenses",
			},
		},
	}

	resp, body := ts.createPurpose(payload)
	require.Equal(t, http.StatusCreated, resp.StatusCode, "Failed to create purpose: %s", body)

	var createResp PurposeCreateResponse
	err := json.Unmarshal([]byte(body), &createResp)
	require.NoError(t, err, "Failed to parse create response")
	require.Len(t, createResp.Data, 1, "Expected 1 purpose created")

	purposeID := createResp.Data[0].ID
	ts.trackPurpose(purposeID)

	// Get the purpose
	resp, body = ts.getPurpose(purposeID)
	require.Equal(t, http.StatusOK, resp.StatusCode, "Failed to get purpose: %s", body)

	var getResp PurposeResponse
	err = json.Unmarshal([]byte(body), &getResp)
	require.NoError(t, err, "Failed to parse get response")

	// Verify all fields
	require.Equal(t, purposeID, getResp.ID, "ID mismatch")
	require.Equal(t, "test_license_read_get", getResp.Name, "Name mismatch")
	require.NotNil(t, getResp.Description, "Description should not be nil")
	require.Equal(t, "License read permission", *getResp.Description, "Description mismatch")
	require.Equal(t, "string", getResp.Type, "Type mismatch")
	require.NotNil(t, getResp.Attributes, "Attributes should not be nil")
	require.Equal(t, "/licenses", getResp.Attributes["resourcePath"], "ResourcePath mismatch")
}

// TestGetPurposeByID_JsonSchemaType_ReturnsWithValidationSchema tests retrieving json-schema type purpose
func (ts *PurposeAPITestSuite) TestGetPurposeByID_JsonSchemaType_ReturnsWithValidationSchema() {
	t := ts.T()

	validationSchema := `{"type":"object","properties":{"accountNumber":{"type":"string"}}}`

	payload := []ConsentPurposeCreateRequest{
		{
			Name:        "test_account_schema_get",
			Description: "Account schema validation",
			Type:        "json-schema",
			Attributes: map[string]string{
				"validationSchema": validationSchema,
			},
		},
	}

	resp, body := ts.createPurpose(payload)
	require.Equal(t, http.StatusCreated, resp.StatusCode, "Failed to create purpose: %s", body)

	var createResp PurposeCreateResponse
	err := json.Unmarshal([]byte(body), &createResp)
	require.NoError(t, err)
	require.Len(t, createResp.Data, 1)

	purposeID := createResp.Data[0].ID
	ts.trackPurpose(purposeID)

	// Get and verify
	resp, body = ts.getPurpose(purposeID)
	require.Equal(t, http.StatusOK, resp.StatusCode, "Failed to get purpose: %s", body)

	var getResp PurposeResponse
	err = json.Unmarshal([]byte(body), &getResp)
	require.NoError(t, err)

	require.Equal(t, purposeID, getResp.ID)
	require.Equal(t, "test_account_schema_get", getResp.Name)
	require.Equal(t, "json-schema", getResp.Type)
	require.NotNil(t, getResp.Attributes["validationSchema"])
}

// TestGetPurposeByID_AttributeType_ReturnsWithBothPaths tests retrieving attribute type with both paths
func (ts *PurposeAPITestSuite) TestGetPurposeByID_AttributeType_ReturnsWithBothPaths() {
	t := ts.T()

	payload := []ConsentPurposeCreateRequest{
		{
			Name:        "test_first_name_get",
			Description: "First name attribute",
			Type:        "attribute",
			Attributes: map[string]string{
				"resourcePath": "/users",
				"jsonPath":     "$.firstName",
			},
		},
	}

	resp, body := ts.createPurpose(payload)
	require.Equal(t, http.StatusCreated, resp.StatusCode, "Failed to create purpose: %s", body)

	var createResp PurposeCreateResponse
	err := json.Unmarshal([]byte(body), &createResp)
	require.NoError(t, err)

	purposeID := createResp.Data[0].ID
	ts.trackPurpose(purposeID)

	// Get and verify both paths present
	resp, body = ts.getPurpose(purposeID)
	require.Equal(t, http.StatusOK, resp.StatusCode, "Failed to get purpose: %s", body)

	var getResp PurposeResponse
	err = json.Unmarshal([]byte(body), &getResp)
	require.NoError(t, err)

	require.Equal(t, "attribute", getResp.Type)
	require.Equal(t, "/users", getResp.Attributes["resourcePath"], "ResourcePath missing")
	require.Equal(t, "$.firstName", getResp.Attributes["jsonPath"], "JsonPath missing")
}

// TestGetPurposeByID_NonExistent_ReturnsNotFound tests getting non-existent purpose
func (ts *PurposeAPITestSuite) TestGetPurposeByID_NonExistent_ReturnsNotFound() {
	t := ts.T()

	nonExistentID := "00000000-0000-0000-0000-000000000000"
	resp, body := ts.getPurpose(nonExistentID)

	require.Equal(t, http.StatusNotFound, resp.StatusCode, "Should return 404 for non-existent purpose")

	var errResp ErrorResponse
	err := json.Unmarshal([]byte(body), &errResp)
	require.NoError(t, err)
	require.Equal(t, "CSE-4004", errResp.Code)
	require.Contains(t, strings.ToLower(errResp.Description), "not found")
}

// TestGetPurposeByID_AfterDelete_ReturnsNotFound tests getting deleted purpose returns 404
func (ts *PurposeAPITestSuite) TestGetPurposeByID_AfterDelete_ReturnsNotFound() {
	t := ts.T()

	// Create purpose
	payload := []ConsentPurposeCreateRequest{
		{
			Name:        "test_to_delete",
			Description: "Will be deleted",
			Type:        "string",
		},
	}

	resp, body := ts.createPurpose(payload)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var createResp PurposeCreateResponse
	json.Unmarshal([]byte(body), &createResp)
	purposeID := createResp.Data[0].ID

	// Delete it
	deleted := ts.deletePurposeWithCheck(purposeID)
	require.True(t, deleted, "Failed to delete purpose")

	// Try to get - should be 404
	resp, body = ts.getPurpose(purposeID)
	require.Equal(t, http.StatusNotFound, resp.StatusCode, "Should return 404 after deletion")

	var errResp ErrorResponse
	json.Unmarshal([]byte(body), &errResp)
	require.Equal(t, "CSE-4004", errResp.Code)
}

// TestGetPurposeByID_ErrorCases tests error scenarios for GET by ID
func (ts *PurposeAPITestSuite) TestGetPurposeByID_ErrorCases() {
	testCases := []struct {
		name            string
		purposeID       string
		setOrgHeader    bool
		expectedStatus  int
		expectedCode    string
		messageContains string
	}{
		{
			name:            "MissingOrgID_ReturnsValidationError",
			purposeID:       "00000000-0000-0000-0000-000000000000",
			setOrgHeader:    false,
			expectedStatus:  http.StatusBadRequest,
			expectedCode:    "CSE-4001",
			messageContains: "organization ID is required",
		},
		{
			name:            "InvalidUUIDFormat_ReturnsNotFound",
			purposeID:       "invalid-uuid-format",
			setOrgHeader:    true,
			expectedStatus:  http.StatusNotFound,
			expectedCode:    "CSE-4004",
			messageContains: "not found",
		},
	}

	for _, tc := range testCases {
		ts.T().Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", fmt.Sprintf("%s/api/v1/consent-purposes/%s", baseURL, tc.purposeID), nil)

			if tc.setOrgHeader {
				req.Header.Set("org-id", testOrgID)
				req.Header.Set("TPP-client-id", testClientID)
			}

			client := &http.Client{}
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			require.Equal(t, tc.expectedStatus, resp.StatusCode, "Status code mismatch for %s", tc.name)

			var errResp ErrorResponse
			json.NewDecoder(resp.Body).Decode(&errResp)
			require.Equal(t, tc.expectedCode, errResp.Code, "Error code mismatch")
			require.Contains(t, strings.ToLower(errResp.Description), strings.ToLower(tc.messageContains),
				"Error message should contain '%s', got: %s", tc.messageContains, errResp.Description)
		})
	}
}

// ========================================
// GET /consent-purposes (LIST) Tests
// ========================================

// TestListPurposes_DefaultPagination_ReturnsAllPurposes tests listing with default pagination
func (ts *PurposeAPITestSuite) TestListPurposes_DefaultPagination_ReturnsAllPurposes() {
	t := ts.T()

	// Create 3 purposes
	payload := []ConsentPurposeCreateRequest{
		{Name: "test_list_1", Description: "First purpose", Type: "string"},
		{Name: "test_list_2", Description: "Second purpose", Type: "string"},
		{Name: "test_list_3", Description: "Third purpose", Type: "string"},
	}

	resp, body := ts.createPurpose(payload)
	require.Equal(t, http.StatusCreated, resp.StatusCode, "Failed to create purposes: %s", body)

	var createResp PurposeCreateResponse
	json.Unmarshal([]byte(body), &createResp)
	for _, p := range createResp.Data {
		ts.trackPurpose(p.ID)
	}

	// List all purposes
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/api/v1/consent-purposes", baseURL), nil)
	req.Header.Set("org-id", testOrgID)
	req.Header.Set("TPP-client-id", testClientID)

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var listResp PurposeListResponse
	json.NewDecoder(resp.Body).Decode(&listResp)

	// Verify metadata exists
	require.NotNil(t, listResp.Metadata, "Metadata should be present")
	require.GreaterOrEqual(t, listResp.Metadata.Total, 3, "Should have at least 3 purposes")
	require.Equal(t, 0, listResp.Metadata.Offset, "Default offset should be 0")
	require.Equal(t, 100, listResp.Metadata.Limit, "Default limit should be 100")
	require.GreaterOrEqual(t, listResp.Metadata.Count, 3, "Count should be at least 3")

	// Verify data array
	require.NotEmpty(t, listResp.Data, "Data array should not be empty")
}

// TestListPurposes_WithLimit_ReturnsPaginatedResults tests pagination with custom limit
func (ts *PurposeAPITestSuite) TestListPurposes_WithLimit_ReturnsPaginatedResults() {
	t := ts.T()

	// Create 5 purposes
	purposes := []ConsentPurposeCreateRequest{
		{Name: "test_page_1", Description: "Page test 1", Type: "string"},
		{Name: "test_page_2", Description: "Page test 2", Type: "string"},
		{Name: "test_page_3", Description: "Page test 3", Type: "string"},
		{Name: "test_page_4", Description: "Page test 4", Type: "string"},
		{Name: "test_page_5", Description: "Page test 5", Type: "string"},
	}

	resp, body := ts.createPurpose(purposes)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var createResp PurposeCreateResponse
	json.Unmarshal([]byte(body), &createResp)
	for _, p := range createResp.Data {
		ts.trackPurpose(p.ID)
	}

	// Request with limit=2
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/api/v1/consent-purposes?limit=2", baseURL), nil)
	req.Header.Set("org-id", testOrgID)
	req.Header.Set("TPP-client-id", testClientID)

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var listResp PurposeListResponse
	json.NewDecoder(resp.Body).Decode(&listResp)

	require.Equal(t, 2, listResp.Metadata.Limit, "Limit should be 2")
	require.LessOrEqual(t, listResp.Metadata.Count, 2, "Count should not exceed limit")
	require.GreaterOrEqual(t, listResp.Metadata.Total, 5, "Total should be at least 5")
}

// TestListPurposes_WithLimitAndOffset_ReturnsCorrectPage tests pagination with offset
func (ts *PurposeAPITestSuite) TestListPurposes_WithLimitAndOffset_ReturnsCorrectPage() {
	t := ts.T()

	// Create 3 purposes to ensure we have data
	purposes := []ConsentPurposeCreateRequest{
		{Name: "test_offset_1", Description: "Offset test 1", Type: "string"},
		{Name: "test_offset_2", Description: "Offset test 2", Type: "string"},
		{Name: "test_offset_3", Description: "Offset test 3", Type: "string"},
	}

	resp, body := ts.createPurpose(purposes)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var createResp PurposeCreateResponse
	json.Unmarshal([]byte(body), &createResp)
	for _, p := range createResp.Data {
		ts.trackPurpose(p.ID)
	}

	// Request with limit=1&offset=1 (second item)
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/api/v1/consent-purposes?limit=1&offset=1", baseURL), nil)
	req.Header.Set("org-id", testOrgID)
	req.Header.Set("TPP-client-id", testClientID)

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var listResp PurposeListResponse
	json.NewDecoder(resp.Body).Decode(&listResp)

	require.Equal(t, 1, listResp.Metadata.Limit, "Limit should be 1")
	require.Equal(t, 1, listResp.Metadata.Offset, "Offset should be 1")
	require.LessOrEqual(t, listResp.Metadata.Count, 1, "Count should not exceed 1")
}

// TestListPurposes_FilterByName_ReturnsMatchingPurpose tests name filtering
func (ts *PurposeAPITestSuite) TestListPurposes_FilterByName_ReturnsMatchingPurpose() {
	t := ts.T()

	// Create purposes with distinctive names
	purposes := []ConsentPurposeCreateRequest{
		{Name: "test_filter_exact", Description: "Exact match test", Type: "string"},
		{Name: "test_filter_other", Description: "Other purpose", Type: "string"},
	}

	resp, body := ts.createPurpose(purposes)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var createResp PurposeCreateResponse
	json.Unmarshal([]byte(body), &createResp)
	for _, p := range createResp.Data {
		ts.trackPurpose(p.ID)
	}

	// Filter by exact name
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/api/v1/consent-purposes?name=test_filter_exact", baseURL), nil)
	req.Header.Set("org-id", testOrgID)
	req.Header.Set("TPP-client-id", testClientID)

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var listResp PurposeListResponse
	json.NewDecoder(resp.Body).Decode(&listResp)

	// Should find at least one match
	require.GreaterOrEqual(t, listResp.Metadata.Total, 1, "Should find at least 1 matching purpose")

	// Verify the filtered result contains our purpose
	found := false
	for _, p := range listResp.Data {
		if p.Name == "test_filter_exact" {
			found = true
			break
		}
	}
	require.True(t, found, "Should find purpose with exact name match")
}

// TestListPurposes_EmptyOrg_ReturnsEmptyArray tests listing for org with no purposes
func (ts *PurposeAPITestSuite) TestListPurposes_EmptyOrg_ReturnsEmptyArray() {
	t := ts.T()

	// Use a different org ID that has no purposes
	emptyOrgID := "org-empty-12345678"

	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/api/v1/consent-purposes", baseURL), nil)
	req.Header.Set("org-id", emptyOrgID)
	req.Header.Set("TPP-client-id", testClientID)

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var listResp PurposeListResponse
	json.NewDecoder(resp.Body).Decode(&listResp)

	require.Equal(t, 0, listResp.Metadata.Total, "Total should be 0 for empty org")
	require.Equal(t, 0, listResp.Metadata.Count, "Count should be 0")
	require.Empty(t, listResp.Data, "Data array should be empty")
}

// TestListPurposes_VerifyAllTypes_ReturnsMixedTypes tests that all purpose types are returned
func (ts *PurposeAPITestSuite) TestListPurposes_VerifyAllTypes_ReturnsMixedTypes() {
	t := ts.T()

	// Create one of each type
	purposes := []ConsentPurposeCreateRequest{
		{
			Name:        "test_alltypes_string",
			Description: "String type",
			Type:        "string",
		},
		{
			Name:        "test_alltypes_jsonschema",
			Description: "JSON Schema type",
			Type:        "json-schema",
			Attributes: map[string]string{
				"validationSchema": `{"type":"object"}`,
			},
		},
		{
			Name:        "test_alltypes_attribute",
			Description: "Attribute type",
			Type:        "attribute",
			Attributes: map[string]string{
				"resourcePath": "/test",
				"jsonPath":     "$.test",
			},
		},
	}

	resp, body := ts.createPurpose(purposes)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var createResp PurposeCreateResponse
	json.Unmarshal([]byte(body), &createResp)
	for _, p := range createResp.Data {
		ts.trackPurpose(p.ID)
	}

	// List all purposes
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/api/v1/consent-purposes", baseURL), nil)
	req.Header.Set("org-id", testOrgID)
	req.Header.Set("TPP-client-id", testClientID)

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var listResp PurposeListResponse
	json.NewDecoder(resp.Body).Decode(&listResp)

	// Verify we have at least one of each type in the results
	typesSeen := make(map[string]bool)
	for _, p := range listResp.Data {
		typesSeen[p.Type] = true
	}

	require.True(t, typesSeen["string"], "Should have string type purpose")
	require.True(t, typesSeen["json-schema"], "Should have json-schema type purpose")
	require.True(t, typesSeen["attribute"], "Should have attribute type purpose")
}

// TestListPurposes_ErrorCases tests error scenarios for LIST
func (ts *PurposeAPITestSuite) TestListPurposes_ErrorCases() {
	testCases := []struct {
		name            string
		setOrgHeader    bool
		expectedStatus  int
		expectedCode    string
		messageContains string
	}{
		{
			name:            "MissingOrgID_ReturnsValidationError",
			setOrgHeader:    false,
			expectedStatus:  http.StatusBadRequest,
			expectedCode:    "CSE-4001",
			messageContains: "organization ID is required",
		},
	}

	for _, tc := range testCases {
		ts.T().Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", fmt.Sprintf("%s/api/v1/consent-purposes", baseURL), nil)

			if tc.setOrgHeader {
				req.Header.Set("org-id", testOrgID)
				req.Header.Set("TPP-client-id", testClientID)
			}

			client := &http.Client{}
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			require.Equal(t, tc.expectedStatus, resp.StatusCode)

			var errResp ErrorResponse
			json.NewDecoder(resp.Body).Decode(&errResp)
			require.Equal(t, tc.expectedCode, errResp.Code)
			require.Contains(t, strings.ToLower(errResp.Description), strings.ToLower(tc.messageContains))
		})
	}
}
