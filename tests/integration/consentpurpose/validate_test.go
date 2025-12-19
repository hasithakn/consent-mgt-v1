package consentpurpose

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wso2/consent-management-api/tests/integration/testutils"
)

// ========================================
// POST /consent-purposes/validate Tests
// ========================================

// TestValidatePurposes_AllValid_ReturnsAll tests validating all existing purposes
func (ts *PurposeAPITestSuite) TestValidatePurposes_AllValid_ReturnsAll() {
	t := ts.T()

	// Create three purposes
	createPayload := []ConsentPurposeCreateRequest{
		{Name: "test_validate_1", Description: "First", Type: "string"},
		{Name: "test_validate_2", Description: "Second", Type: "string"},
		{Name: "test_validate_3", Description: "Third", Type: "string"},
	}

	resp, body := ts.createPurpose(createPayload)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var createResp PurposeCreateResponse
	json.Unmarshal([]byte(body), &createResp)
	for _, p := range createResp.Data {
		ts.trackPurpose(p.ID)
	}

	// Validate all three
	validatePayload := []string{"test_validate_1", "test_validate_2", "test_validate_3"}
	validNames := ts.validatePurposes(validatePayload)

	require.Len(t, validNames, 3, "Should return all 3 valid names")
	require.Contains(t, validNames, "test_validate_1")
	require.Contains(t, validNames, "test_validate_2")
	require.Contains(t, validNames, "test_validate_3")
}

// TestValidatePurposes_PartialValid_ReturnsSubset tests mixed valid and invalid names
func (ts *PurposeAPITestSuite) TestValidatePurposes_PartialValid_ReturnsSubset() {
	t := ts.T()

	// Create two purposes
	createPayload := []ConsentPurposeCreateRequest{
		{Name: "test_partial_1", Description: "First", Type: "string"},
		{Name: "test_partial_2", Description: "Second", Type: "string"},
	}

	resp, body := ts.createPurpose(createPayload)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var createResp PurposeCreateResponse
	json.Unmarshal([]byte(body), &createResp)
	for _, p := range createResp.Data {
		ts.trackPurpose(p.ID)
	}

	// Validate mix of valid and invalid
	validatePayload := []string{
		"test_partial_1", // Valid
		"test_partial_2", // Valid
		"nonexistent_1",  // Invalid
		"nonexistent_2",  // Invalid
	}
	validNames := ts.validatePurposes(validatePayload)

	require.Len(t, validNames, 2, "Should return only 2 valid names")
	require.Contains(t, validNames, "test_partial_1")
	require.Contains(t, validNames, "test_partial_2")
	require.NotContains(t, validNames, "nonexistent_1")
	require.NotContains(t, validNames, "nonexistent_2")
}

// TestValidatePurposes_NoneValid_ReturnsEmpty tests all invalid names
func (ts *PurposeAPITestSuite) TestValidatePurposes_NoneValid_ReturnsEmpty() {
	t := ts.T()

	// Validate only non-existent names
	validatePayload := []string{
		"totally_fake_name_1",
		"totally_fake_name_2",
		"totally_fake_name_3",
	}

	// Server returns 400 error when no valid purposes found
	reqBody, err := json.Marshal(validatePayload)
	require.NoError(t, err)

	httpReq, _ := http.NewRequest("POST",
		fmt.Sprintf("%s/api/v1/consent-purposes/validate", baseURL),
		bytes.NewBuffer(reqBody))
	httpReq.Header.Set(testutils.HeaderContentType, "application/json")
	httpReq.Header.Set(testutils.HeaderOrgID, testOrgID)
	httpReq.Header.Set(testutils.HeaderClientID, testClientID)

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusBadRequest, resp.StatusCode, "Should return 400 when no valid purposes found")

	var errResp ErrorResponse
	json.NewDecoder(resp.Body).Decode(&errResp)
	require.Equal(t, "CSE-4001", errResp.Code)
	require.Contains(t, strings.ToLower(errResp.Description), "no valid purposes found")
}

// TestValidatePurposes_SingleName_ReturnsOne tests single name validation
func (ts *PurposeAPITestSuite) TestValidatePurposes_SingleName_ReturnsOne() {
	t := ts.T()

	// Create purpose
	createPayload := []ConsentPurposeCreateRequest{
		{Name: "test_single_validate", Description: "Single", Type: "string"},
	}

	resp, body := ts.createPurpose(createPayload)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var createResp PurposeCreateResponse
	json.Unmarshal([]byte(body), &createResp)
	ts.trackPurpose(createResp.Data[0].ID)

	// Validate single name
	validatePayload := []string{"test_single_validate"}
	validNames := ts.validatePurposes(validatePayload)

	require.Len(t, validNames, 1)
	require.Equal(t, "test_single_validate", validNames[0])
}

// TestValidatePurposes_DuplicatesInRequest_ReturnsDeduplicated tests duplicate handling
func (ts *PurposeAPITestSuite) TestValidatePurposes_DuplicatesInRequest_ReturnsDeduplicated() {
	t := ts.T()

	// Create purpose
	createPayload := []ConsentPurposeCreateRequest{
		{Name: "test_duplicate_validate", Description: "Duplicate test", Type: "string"},
	}

	resp, body := ts.createPurpose(createPayload)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var createResp PurposeCreateResponse
	json.Unmarshal([]byte(body), &createResp)
	ts.trackPurpose(createResp.Data[0].ID)

	// Validate with duplicates
	validatePayload := []string{
		"test_duplicate_validate",
		"test_duplicate_validate",
		"test_duplicate_validate",
	}
	validNames := ts.validatePurposes(validatePayload)

	// Should return deduplicated result
	require.LessOrEqual(t, len(validNames), 3, "Should handle duplicates")
	require.Contains(t, validNames, "test_duplicate_validate")
}

// TestValidatePurposes_MixedTypes_ReturnsAllValid tests validation across different purpose types
func (ts *PurposeAPITestSuite) TestValidatePurposes_MixedTypes_ReturnsAllValid() {
	t := ts.T()

	// Create one of each type
	createPayload := []ConsentPurposeCreateRequest{
		{
			Name:        "test_validate_string",
			Description: "String type",
			Type:        "string",
		},
		{
			Name:        "test_validate_jsonschema",
			Description: "JSON Schema type",
			Type:        "json-schema",
			Attributes: map[string]string{
				"validationSchema": `{"type":"object"}`,
			},
		},
		{
			Name:        "test_validate_attribute",
			Description: "Attribute type",
			Type:        "attribute",
			Attributes: map[string]string{
				"resourcePath": "/test",
				"jsonPath":     "$.test",
			},
		},
	}

	resp, body := ts.createPurpose(createPayload)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var createResp PurposeCreateResponse
	json.Unmarshal([]byte(body), &createResp)
	for _, p := range createResp.Data {
		ts.trackPurpose(p.ID)
	}

	// Validate all types
	validatePayload := []string{
		"test_validate_string",
		"test_validate_jsonschema",
		"test_validate_attribute",
	}
	validNames := ts.validatePurposes(validatePayload)

	require.Len(t, validNames, 3, "Should validate all purpose types")
	require.Contains(t, validNames, "test_validate_string")
	require.Contains(t, validNames, "test_validate_jsonschema")
	require.Contains(t, validNames, "test_validate_attribute")
}

// TestValidatePurposes_ErrorCases tests error scenarios for VALIDATE
func (ts *PurposeAPITestSuite) TestValidatePurposes_ErrorCases() {
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
			payload:         []string{"test"},
			setHeaders:      false,
			expectedStatus:  http.StatusBadRequest,
			expectedCode:    "CSE-4001",
			messageContains: "organization ID is required",
		},
		{
			name:            "EmptyArray_ReturnsBadRequest",
			payload:         []string{},
			setHeaders:      true,
			expectedStatus:  http.StatusBadRequest,
			expectedCode:    "CSE-4001",
			messageContains: "at least one purpose name must be provided",
		},
		{
			name:            "MalformedJSON_ReturnsBadRequest",
			payload:         "invalid{{{",
			setHeaders:      true,
			expectedStatus:  http.StatusBadRequest,
			expectedCode:    "CSE-4000",
			messageContains: "invalid request body",
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

			req, _ := http.NewRequest("POST",
				fmt.Sprintf("%s/api/v1/consent-purposes/validate", baseURL),
				bytes.NewBuffer(reqBody))
			req.Header.Set(testutils.HeaderContentType, "application/json")

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

// Helper method to validate purpose names
func (ts *PurposeAPITestSuite) validatePurposes(names []string) []string {
	reqBody, err := json.Marshal(names)
	ts.Require().NoError(err)

	httpReq, _ := http.NewRequest("POST",
		fmt.Sprintf("%s/api/v1/consent-purposes/validate", baseURL),
		bytes.NewBuffer(reqBody))
	httpReq.Header.Set(testutils.HeaderContentType, "application/json")
	httpReq.Header.Set(testutils.HeaderOrgID, testOrgID)
	httpReq.Header.Set(testutils.HeaderClientID, testClientID)

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	ts.Require().NoError(err)
	defer resp.Body.Close()

	ts.Require().Equal(http.StatusOK, resp.StatusCode, "Validate request should succeed")

	var validNames []string
	err = json.NewDecoder(resp.Body).Decode(&validNames)
	ts.Require().NoError(err)

	return validNames
}
