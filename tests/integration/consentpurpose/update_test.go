package consentpurpose

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wso2/consent-management-api/tests/integration/testutils"
)

// ========================================
// PUT /consent-purposes/{purposeId} Tests
// ========================================

// TestUpdatePurpose_DescriptionChange_Succeeds tests updating only the description
func (ts *PurposeAPITestSuite) TestUpdatePurpose_DescriptionChange_Succeeds() {
	t := ts.T()

	// Create a purpose
	createPayload := []ConsentPurposeCreateRequest{
		{
			Name:        "test_update_desc",
			Description: "Original description",
			Type:        "string",
		},
	}

	resp, bodyBytes := ts.createPurpose(createPayload)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var createResp PurposeCreateResponse
	json.Unmarshal(bodyBytes, &createResp)
	purposeID := createResp.Data[0].ID
	ts.trackPurpose(purposeID)

	// Update description
	updatePayload := ConsentPurposeUpdateRequest{
		Name:        "test_update_desc",
		Description: "Updated description",
		Type:        "string",
	}

	resp, bodyBytes = ts.updatePurpose(purposeID, updatePayload)
	require.Equal(t, http.StatusOK, resp.StatusCode, "Failed to update purpose: %s", bodyBytes)

	var updateResp PurposeResponse
	json.Unmarshal(bodyBytes, &updateResp)
	require.Equal(t, "Updated description", *updateResp.Description)
}

// TestUpdatePurpose_AttributeChange_JsonPath_Succeeds tests updating jsonPath attribute
func (ts *PurposeAPITestSuite) TestUpdatePurpose_AttributeChange_JsonPath_Succeeds() {
	t := ts.T()

	// Create attribute type purpose
	createPayload := []ConsentPurposeCreateRequest{
		{
			Name:        "test_update_jsonpath",
			Description: "Attribute purpose",
			Type:        "attribute",
			Attributes: map[string]string{
				"resourcePath": "/users",
				"jsonPath":     "$.firstName",
			},
		},
	}

	resp, bodyBytes := ts.createPurpose(createPayload)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var createResp PurposeCreateResponse
	json.Unmarshal(bodyBytes, &createResp)
	purposeID := createResp.Data[0].ID
	ts.trackPurpose(purposeID)

	// Update jsonPath
	updatePayload := ConsentPurposeUpdateRequest{
		Name:        "test_update_jsonpath",
		Description: "Attribute purpose",
		Type:        "attribute",
		Attributes: map[string]string{
			"resourcePath": "/users",
			"jsonPath":     "$.profile.firstName", // Changed
		},
	}

	resp, bodyBytes = ts.updatePurpose(purposeID, updatePayload)
	require.Equal(t, http.StatusOK, resp.StatusCode, "Failed to update purpose: %s", bodyBytes)

	var updateResp PurposeResponse
	json.Unmarshal(bodyBytes, &updateResp)
	require.Equal(t, "$.profile.firstName", updateResp.Attributes["jsonPath"])
}

// TestUpdatePurpose_TypeChange_StringToJsonSchema_Succeeds tests changing purpose type
func (ts *PurposeAPITestSuite) TestUpdatePurpose_TypeChange_StringToJsonSchema_Succeeds() {
	t := ts.T()

	// Create string type purpose
	createPayload := []ConsentPurposeCreateRequest{
		{
			Name:        "test_type_change",
			Description: "Type change test",
			Type:        "string",
		},
	}

	resp, bodyBytes := ts.createPurpose(createPayload)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var createResp PurposeCreateResponse
	json.Unmarshal(bodyBytes, &createResp)
	purposeID := createResp.Data[0].ID
	ts.trackPurpose(purposeID)

	// Change to json-schema type
	updatePayload := ConsentPurposeUpdateRequest{
		Name:        "test_type_change",
		Description: "Type change test",
		Type:        "json-schema",
		Attributes: map[string]string{
			"validationSchema": `{"type":"object","properties":{"name":{"type":"string"}}}`,
		},
	}

	resp, bodyBytes = ts.updatePurpose(purposeID, updatePayload)
	require.Equal(t, http.StatusOK, resp.StatusCode, "Failed to update purpose: %s", bodyBytes)

	var updateResp PurposeResponse
	json.Unmarshal(bodyBytes, &updateResp)
	require.Equal(t, "json-schema", updateResp.Type)
	require.NotEmpty(t, updateResp.Attributes["validationSchema"])
}

// TestUpdatePurpose_AllFieldsAtOnce_Succeeds tests updating all fields simultaneously
func (ts *PurposeAPITestSuite) TestUpdatePurpose_AllFieldsAtOnce_Succeeds() {
	t := ts.T()

	// Create purpose
	createPayload := []ConsentPurposeCreateRequest{
		{
			Name:        "test_full_update",
			Description: "Original",
			Type:        "string",
			Attributes: map[string]string{
				"resourcePath": "/old",
			},
		},
	}

	resp, bodyBytes := ts.createPurpose(createPayload)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var createResp PurposeCreateResponse
	json.Unmarshal(bodyBytes, &createResp)
	purposeID := createResp.Data[0].ID
	ts.trackPurpose(purposeID)

	// Update all fields
	updatePayload := ConsentPurposeUpdateRequest{
		Name:        "test_full_update_new",
		Description: "Completely new description",
		Type:        "attribute",
		Attributes: map[string]string{
			"resourcePath": "/new/path",
			"jsonPath":     "$.newField",
		},
	}

	resp, bodyBytes = ts.updatePurpose(purposeID, updatePayload)
	require.Equal(t, http.StatusOK, resp.StatusCode, "Failed to update purpose: %s", bodyBytes)

	var updateResp PurposeResponse
	json.Unmarshal(bodyBytes, &updateResp)
	require.Equal(t, "test_full_update_new", updateResp.Name)
	require.Equal(t, "Completely new description", *updateResp.Description)
	require.Equal(t, "attribute", updateResp.Type)
	require.Equal(t, "/new/path", updateResp.Attributes["resourcePath"])
	require.Equal(t, "$.newField", updateResp.Attributes["jsonPath"])
}

// TestUpdatePurpose_NonExistent_ReturnsNotFound tests updating non-existent purpose
func (ts *PurposeAPITestSuite) TestUpdatePurpose_NonExistent_ReturnsNotFound() {
	t := ts.T()

	nonExistentID := "00000000-0000-0000-0000-000000000000"
	updatePayload := ConsentPurposeUpdateRequest{
		Name:        "test_nonexistent",
		Description: "Should fail",
		Type:        "string",
	}

	resp, bodyBytes := ts.updatePurpose(nonExistentID, updatePayload)
	require.Equal(t, http.StatusNotFound, resp.StatusCode, "Should return 404 for non-existent purpose")

	var errResp ErrorResponse
	json.Unmarshal(bodyBytes, &errResp)
	require.Equal(t, "CSE-4004", errResp.Code)
	require.Contains(t, strings.ToLower(errResp.Description), "not found")
}

// TestUpdatePurpose_ErrorCases tests error scenarios for UPDATE
func (ts *PurposeAPITestSuite) TestUpdatePurpose_ErrorCases() {
	// Create a valid purpose for error testing
	createPayload := []ConsentPurposeCreateRequest{
		{
			Name:        "test_update_errors",
			Description: "For error testing",
			Type:        "string",
		},
	}
	resp, body := ts.createPurpose(createPayload)
	require.Equal(ts.T(), http.StatusCreated, resp.StatusCode)

	var createResp PurposeCreateResponse
	json.Unmarshal([]byte(body), &createResp)
	purposeID := createResp.Data[0].ID
	ts.trackPurpose(purposeID)

	testCases := []struct {
		name            string
		payload         interface{}
		setHeaders      bool
		expectedStatus  int
		expectedCode    string
		messageContains string
	}{
		{
			name:            "MissingOrgID_ReturnsValidationError",
			payload:         ConsentPurposeUpdateRequest{Name: "test", Type: "string"},
			setHeaders:      false,
			expectedStatus:  http.StatusBadRequest,
			expectedCode:    "CSE-4000",
			messageContains: "organization ID is required",
		},
		{
			name:            "MissingNameField_ReturnsValidationError",
			payload:         map[string]interface{}{"type": "string", "description": "test"},
			setHeaders:      true,
			expectedStatus:  http.StatusBadRequest,
			expectedCode:    "CSE-4001",
			messageContains: "purpose name is required",
		},
		{
			name:            "MissingTypeField_ReturnsValidationError",
			payload:         map[string]interface{}{"name": "test", "description": "test"},
			setHeaders:      true,
			expectedStatus:  http.StatusBadRequest,
			expectedCode:    "CSE-4001",
			messageContains: "purpose type is required",
		},
		{
			name: "NameExceeds255Chars_ReturnsValidationError",
			payload: ConsentPurposeUpdateRequest{
				Name:        strings.Repeat("a", 256),
				Description: "test",
				Type:        "string",
			},
			setHeaders:      true,
			expectedStatus:  http.StatusBadRequest,
			expectedCode:    "CSE-4001",
			messageContains: "must not exceed 255 characters",
		},
		{
			name: "DescriptionExceeds1024Chars_ReturnsValidationError",
			payload: ConsentPurposeUpdateRequest{
				Name:        "test_desc_long",
				Description: strings.Repeat("a", 1025),
				Type:        "string",
			},
			setHeaders:      true,
			expectedStatus:  http.StatusBadRequest,
			expectedCode:    "CSE-4001",
			messageContains: "must not exceed 1024 characters",
		},
		{
			name: "InvalidType_ReturnsValidationError",
			payload: ConsentPurposeUpdateRequest{
				Name:        "test_invalid_type",
				Description: "test",
				Type:        "INVALID_TYPE",
			},
			setHeaders:      true,
			expectedStatus:  http.StatusBadRequest,
			expectedCode:    "CSE-4001",
			messageContains: "invalid purpose type",
		},
		{
			name: "JsonSchemaType_MissingValidationSchema_ReturnsValidationError",
			payload: ConsentPurposeUpdateRequest{
				Name:        "test_jsonschema_missing",
				Description: "test",
				Type:        "json-schema",
			},
			setHeaders:      true,
			expectedStatus:  http.StatusBadRequest,
			expectedCode:    "CSE-4001",
			messageContains: "validationSchema is required for json-schema type",
		},
		{
			name: "AttributeType_MissingResourcePath_ReturnsValidationError",
			payload: ConsentPurposeUpdateRequest{
				Name:        "test_attr_no_resource",
				Description: "test",
				Type:        "attribute",
				Attributes: map[string]string{
					"jsonPath": "$.test",
				},
			},
			setHeaders:      true,
			expectedStatus:  http.StatusBadRequest,
			expectedCode:    "CSE-4001",
			messageContains: "resourcePath is required for attribute type",
		},
		{
			name: "AttributeType_MissingJsonPath_ReturnsValidationError",
			payload: ConsentPurposeUpdateRequest{
				Name:        "test_attr_no_json",
				Description: "test",
				Type:        "attribute",
				Attributes: map[string]string{
					"resourcePath": "/test",
				},
			},
			setHeaders:      true,
			expectedStatus:  http.StatusBadRequest,
			expectedCode:    "CSE-4001",
			messageContains: "jsonPath is required for attribute type",
		},
	}

	for _, tc := range testCases {
		ts.T().Run(tc.name, func(t *testing.T) {
			var reqBody []byte
			var err error

			if str, ok := tc.payload.(string); ok {
				reqBody = []byte(str)
			} else {
				reqBody, err = json.Marshal(tc.payload)
				require.NoError(t, err)
			}

			req, _ := http.NewRequest("PUT",
				fmt.Sprintf("%s/api/v1/consent-purposes/%s", baseURL, purposeID),
				bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")

			if tc.setHeaders {
				req.Header.Set(testutils.HeaderOrgID, testOrgID)
				req.Header.Set(testutils.HeaderClientID, testClientID)
			}

			client := &http.Client{}
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			require.Equal(t, tc.expectedStatus, resp.StatusCode, "Status code mismatch for %s", tc.name)

			body, _ := io.ReadAll(resp.Body)
			var errResp ErrorResponse
			json.Unmarshal(body, &errResp)
			require.Equal(t, tc.expectedCode, errResp.Code, "Error code mismatch for %s", tc.name)
			require.Contains(t, strings.ToLower(errResp.Description), strings.ToLower(tc.messageContains),
				"Error message should contain '%s', got: %s", tc.messageContains, errResp.Description)
		})
	}
}
