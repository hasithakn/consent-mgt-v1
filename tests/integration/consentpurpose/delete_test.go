package consentpurpose

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wso2/consent-management-api/tests/integration/testutils"
)

// ========================================
// DELETE /consent-purposes/{purposeId} Tests
// ========================================

// TestDeletePurpose_StringType_Success tests deleting a string type purpose
func (ts *PurposeAPITestSuite) TestDeletePurpose_StringType_Success() {
	t := ts.T()

	// Create purpose
	createPayload := []ConsentPurposeCreateRequest{
		{
			Name:        "test_delete_string",
			Description: "To be deleted",
			Type:        "string",
		},
	}

	resp, body := ts.createPurpose(createPayload)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var createResp PurposeCreateResponse
	json.Unmarshal([]byte(body), &createResp)
	purposeID := createResp.Data[0].ID

	// Delete it
	deleted := ts.deletePurposeWithCheck(purposeID)
	require.True(t, deleted, "Failed to delete purpose")

	// Verify it's gone with GET
	resp, _ = ts.getPurpose(purposeID)
	require.Equal(t, http.StatusNotFound, resp.StatusCode, "Purpose should not exist after deletion")
}

// TestDeletePurpose_JsonSchemaType_Success tests deleting a json-schema type purpose
func (ts *PurposeAPITestSuite) TestDeletePurpose_JsonSchemaType_Success() {
	t := ts.T()

	// Create json-schema purpose
	createPayload := []ConsentPurposeCreateRequest{
		{
			Name:        "test_delete_jsonschema",
			Description: "To be deleted",
			Type:        "json-schema",
			Attributes: map[string]string{
				"validationSchema": `{"type":"object"}`,
			},
		},
	}

	resp, body := ts.createPurpose(createPayload)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var createResp PurposeCreateResponse
	json.Unmarshal([]byte(body), &createResp)
	purposeID := createResp.Data[0].ID

	// Delete it
	deleted := ts.deletePurposeWithCheck(purposeID)
	require.True(t, deleted, "Failed to delete purpose")

	// Verify deletion
	resp, _ = ts.getPurpose(purposeID)
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// TestDeletePurpose_AttributeType_Success tests deleting an attribute type purpose
func (ts *PurposeAPITestSuite) TestDeletePurpose_AttributeType_Success() {
	t := ts.T()

	// Create attribute purpose
	createPayload := []ConsentPurposeCreateRequest{
		{
			Name:        "test_delete_attribute",
			Description: "To be deleted",
			Type:        "attribute",
			Attributes: map[string]string{
				"resourcePath": "/users",
				"jsonPath":     "$.email",
			},
		},
	}

	resp, body := ts.createPurpose(createPayload)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var createResp PurposeCreateResponse
	json.Unmarshal([]byte(body), &createResp)
	purposeID := createResp.Data[0].ID

	// Delete it
	deleted := ts.deletePurposeWithCheck(purposeID)
	require.True(t, deleted, "Failed to delete purpose")

	// Verify deletion
	resp, _ = ts.getPurpose(purposeID)
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// TestDeletePurpose_AlreadyDeleted_ReturnsNotFound tests idempotent deletion
func (ts *PurposeAPITestSuite) TestDeletePurpose_AlreadyDeleted_ReturnsNotFound() {
	t := ts.T()

	// Create purpose
	createPayload := []ConsentPurposeCreateRequest{
		{
			Name:        "test_delete_twice",
			Description: "Delete twice test",
			Type:        "string",
		},
	}

	resp, body := ts.createPurpose(createPayload)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var createResp PurposeCreateResponse
	json.Unmarshal([]byte(body), &createResp)
	purposeID := createResp.Data[0].ID

	// First delete - should succeed
	deleted := ts.deletePurposeWithCheck(purposeID)
	require.True(t, deleted, "First deletion should succeed")

	// Second delete - should return 404
	deleted = ts.deletePurposeWithCheck(purposeID)
	require.False(t, deleted, "Second deletion should fail (return false)")
}

// TestDeletePurpose_ThenRecreateWithSameName_Succeeds tests name reusability after deletion
func (ts *PurposeAPITestSuite) TestDeletePurpose_ThenRecreateWithSameName_Succeeds() {
	t := ts.T()

	purposeName := "test_delete_recreate"

	// Create purpose
	createPayload := []ConsentPurposeCreateRequest{
		{
			Name:        purposeName,
			Description: "First version",
			Type:        "string",
		},
	}

	resp, body := ts.createPurpose(createPayload)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var createResp PurposeCreateResponse
	json.Unmarshal([]byte(body), &createResp)
	purposeID := createResp.Data[0].ID

	// Delete it
	deleted := ts.deletePurposeWithCheck(purposeID)
	require.True(t, deleted, "Failed to delete purpose")

	// Recreate with same name
	createPayload[0].Description = "Second version"
	resp, body = ts.createPurpose(createPayload)
	require.Equal(t, http.StatusCreated, resp.StatusCode, "Should be able to recreate with same name after deletion")

	var recreateResp PurposeCreateResponse
	json.Unmarshal([]byte(body), &recreateResp)
	newPurposeID := recreateResp.Data[0].ID
	ts.trackPurpose(newPurposeID)

	// Verify it's a different purpose with different ID
	require.NotEqual(t, purposeID, newPurposeID, "New purpose should have different ID")
	require.Equal(t, purposeName, recreateResp.Data[0].Name)
}

// TestDeletePurpose_NonExistent_ReturnsNotFound tests deleting non-existent purpose
func (ts *PurposeAPITestSuite) TestDeletePurpose_NonExistent_ReturnsNotFound() {
	t := ts.T()

	nonExistentID := "00000000-0000-0000-0000-000000000000"

	deleted := ts.deletePurposeWithCheck(nonExistentID)
	require.False(t, deleted, "Deleting non-existent purpose should return false")
}

// TestDeletePurpose_ErrorCases tests error scenarios for DELETE
func (ts *PurposeAPITestSuite) TestDeletePurpose_ErrorCases() {
	testCases := []struct {
		name            string
		purposeID       string
		setHeaders      bool
		expectedStatus  int
		expectedCode    string
		messageContains string
	}{
		{
			name:            "MissingOrgID_ReturnsValidationError",
			purposeID:       "00000000-0000-0000-0000-000000000000",
			setHeaders:      false,
			expectedStatus:  http.StatusBadRequest,
			expectedCode:    "CSE-4001",
			messageContains: "organization ID is required",
		},
		{
			name:            "InvalidUUIDFormat_ReturnsNotFound",
			purposeID:       "invalid-uuid-format",
			setHeaders:      true,
			expectedStatus:  http.StatusNotFound,
			expectedCode:    "CSE-4004",
			messageContains: "not found",
		},
	}

	for _, tc := range testCases {
		ts.T().Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest("DELETE",
				fmt.Sprintf("%s/api/v1/consent-purposes/%s", baseURL, tc.purposeID), nil)

			if tc.setHeaders {
				req.Header.Set(testutils.HeaderOrgID, testOrgID)
				req.Header.Set(testutils.HeaderClientID, testClientID)
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
