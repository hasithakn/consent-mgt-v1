/*
 * Copyright (c) 2025, WSO2 LLC. (https://www.wso2.com).
 *
 * WSO2 LLC. licenses this file to you under the Apache License,
 * Version 2.0 (the "License"); you may not use this file except
 * in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

package consentpurpose

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/wso2/consent-management-api/tests/integration/testutils"
)

// TestCreatePurpose_StringType_WithResourcePath creates a string type purpose with resourcePath attribute
func (ts *PurposeAPITestSuite) TestCreatePurpose_StringType_WithResourcePath() {
	purpose := ConsentPurposeCreateRequest{
		Name:        "test_license_read",
		Description: "Allows accessing driving license API",
		Type:        "string",
		Attributes: map[string]string{
			"resourcePath": "/license/{nic}",
		},
	}

	// Create purpose
	resp, body := ts.createPurpose([]ConsentPurposeCreateRequest{purpose})
	defer resp.Body.Close()

	ts.Require().Equal(http.StatusCreated, resp.StatusCode)

	var createResp PurposeCreateResponse
	err := json.Unmarshal(body, &createResp)
	ts.Require().NoError(err)
	ts.Require().Len(createResp.Data, 1)

	created := createResp.Data[0]
	ts.Require().Equal(purpose.Name, created.Name)
	ts.Require().Equal(purpose.Type, created.Type)
	ts.Require().Equal(purpose.Attributes["validationSchema"], created.Attributes["validationSchema"])
	ts.Require().NotEmpty(created.ID)

	// Track for suite cleanup
	ts.trackPurpose(created.ID)
}

// TestCreatePurpose_StringType_NoAttributes creates a string type purpose with no attributes
func (ts *PurposeAPITestSuite) TestCreatePurpose_StringType_NoAttributes() {
	purpose := ConsentPurposeCreateRequest{
		Name:        "test_basic_string",
		Description: "String type with no attributes",
		Type:        "string",
		Attributes:  map[string]string{},
	}

	resp, body := ts.createPurpose([]ConsentPurposeCreateRequest{purpose})
	defer resp.Body.Close()

	ts.Require().Equal(http.StatusCreated, resp.StatusCode)

	var createResp PurposeCreateResponse
	err := json.Unmarshal(body, &createResp)
	ts.Require().NoError(err)
	ts.Require().Len(createResp.Data, 1)
	ts.Require().Equal(purpose.Name, createResp.Data[0].Name)
	ts.trackPurpose(createResp.Data[0].ID) // Track for suite cleanup
}

// TestCreatePurpose_JsonSchemaType_WithValidationSchema creates a json-schema type purpose
func (ts *PurposeAPITestSuite) TestCreatePurpose_JsonSchemaType_WithValidationSchema() {
	purpose := ConsentPurposeCreateRequest{
		Name:        "test_account_schema",
		Description: "Account access schema validation",
		Type:        "json-schema",
		Attributes: map[string]string{
			"validationSchema": "{}",
		},
	}

	resp, body := ts.createPurpose([]ConsentPurposeCreateRequest{purpose})
	defer resp.Body.Close()

	ts.Require().Equal(http.StatusCreated, resp.StatusCode)

	var createResp PurposeCreateResponse
	err := json.Unmarshal(body, &createResp)
	ts.Require().NoError(err)
	ts.Require().Len(createResp.Data, 1)

	created := createResp.Data[0]
	ts.Require().Equal(purpose.Name, created.Name)
	ts.Require().Equal("json-schema", created.Type)
	ts.Require().NotEmpty(created.Attributes["validationSchema"])
	ts.trackPurpose(created.ID) // Track for suite cleanup
}

// TestCreatePurpose_AttributeType_FirstName creates an attribute type purpose with jsonPath and resourcePath
func (ts *PurposeAPITestSuite) TestCreatePurpose_AttributeType_FirstName() {
	purpose := ConsentPurposeCreateRequest{
		Name:        "test_first_name",
		Description: "Allows access to the user's first name",
		Type:        "attribute",
		Attributes: map[string]string{
			"jsonPath":     "$.personal.firstName",
			"resourcePath": "/user/{nic}",
		},
	}

	resp, body := ts.createPurpose([]ConsentPurposeCreateRequest{purpose})
	defer resp.Body.Close()

	ts.Require().Equal(http.StatusCreated, resp.StatusCode)

	var createResp PurposeCreateResponse
	err := json.Unmarshal(body, &createResp)
	ts.Require().NoError(err)
	ts.Require().Len(createResp.Data, 1)

	created := createResp.Data[0]
	ts.Require().Equal(purpose.Name, created.Name)
	ts.Require().Equal("attribute", created.Type)
	ts.Require().Equal("$.personal.firstName", created.Attributes["jsonPath"])
	ts.Require().Equal("/user/{nic}", created.Attributes["resourcePath"])
	ts.trackPurpose(created.ID) // Track for suite cleanup
}

// TestCreatePurpose_Batch_ThreeAttributeTypePurposes creates 3 attribute type purposes in one request
func (ts *PurposeAPITestSuite) TestCreatePurpose_Batch_ThreeAttributeTypePurposes() {
	purposes := []ConsentPurposeCreateRequest{
		{
			Name:        "test_batch_first_name",
			Description: "Allows access to the user's first name",
			Type:        "attribute",
			Attributes: map[string]string{
				"jsonPath":     "$.personal.firstName",
				"resourcePath": "/user/{nic}",
			},
		},
		{
			Name:        "test_batch_last_name",
			Description: "Allows access to the user's last name",
			Type:        "attribute",
			Attributes: map[string]string{
				"jsonPath":     "$.personal.lastName",
				"resourcePath": "/user/{nic}",
			},
		},
		{
			Name:        "test_batch_full_name",
			Description: "Allows access to the user's full name",
			Type:        "attribute",
			Attributes: map[string]string{
				"jsonPath":     "$.personal.fullName",
				"resourcePath": "/user/{nic}",
			},
		},
	}

	resp, body := ts.createPurpose(purposes)
	defer resp.Body.Close()

	ts.Require().Equal(http.StatusCreated, resp.StatusCode)

	var createResp PurposeCreateResponse
	err := json.Unmarshal(body, &createResp)
	ts.Require().NoError(err)
	ts.Require().Len(createResp.Data, 3)

	// Verify all three were created
	for i, purpose := range purposes {
		ts.Require().Equal(purpose.Name, createResp.Data[i].Name)
		ts.Require().Equal("attribute", createResp.Data[i].Type)
		ts.Require().NotEmpty(createResp.Data[i].ID)
	}

	// Track all for cleanup
	for _, p := range createResp.Data {
		ts.trackPurpose(p.ID)
	}
}

// TestCreatePurpose_Batch_MixedTypes creates purposes with different types in one batch
func (ts *PurposeAPITestSuite) TestCreatePurpose_Batch_MixedTypes() {
	purposes := []ConsentPurposeCreateRequest{
		{
			Name:        "test_mixed_string",
			Description: "String type",
			Type:        "string",
			Attributes: map[string]string{
				"resourcePath": "/api/resource",
			},
		},
		{
			Name:        "test_mixed_json_schema",
			Description: "JSON Schema type",
			Type:        "json-schema",
			Attributes: map[string]string{
				"validationSchema": "{}",
			},
		},
		{
			Name:        "test_mixed_attribute",
			Description: "Attribute type",
			Type:        "attribute",
			Attributes: map[string]string{
				"jsonPath":     "$.data.field",
				"resourcePath": "/resource/{id}",
			},
		},
	}

	resp, body := ts.createPurpose(purposes)
	defer resp.Body.Close()

	ts.Require().Equal(http.StatusCreated, resp.StatusCode)

	var createResp PurposeCreateResponse
	err := json.Unmarshal(body, &createResp)
	ts.Require().NoError(err)
	ts.Require().Len(createResp.Data, 3)

	// Verify types
	ts.Require().Equal("string", createResp.Data[0].Type)
	ts.Require().Equal("json-schema", createResp.Data[1].Type)
	ts.Require().Equal("attribute", createResp.Data[2].Type)

	// Track all for cleanup
	for _, p := range createResp.Data {
		ts.trackPurpose(p.ID)
	}
}

// TestCreatePurpose_RetrieveAndVerifyAllFields creates a purpose and verifies all fields via GET
func (ts *PurposeAPITestSuite) TestCreatePurpose_RetrieveAndVerifyAllFields() {
	purpose := ConsentPurposeCreateRequest{
		Name:        "test_verify_all_fields",
		Description: "Test purpose for field verification",
		Type:        "attribute",
		Attributes: map[string]string{
			"jsonPath":     "$.user.email",
			"resourcePath": "/user/{id}",
		},
	}

	// Create
	createHttpResp, createBody := ts.createPurpose([]ConsentPurposeCreateRequest{purpose})
	defer createHttpResp.Body.Close()
	ts.Require().Equal(http.StatusCreated, createHttpResp.StatusCode)

	var createResp PurposeCreateResponse
	err := json.Unmarshal(createBody, &createResp)
	ts.Require().NoError(err)
	ts.Require().Len(createResp.Data, 1)

	purposeID := createResp.Data[0].ID
	ts.trackPurpose(purposeID) // Track for suite cleanup

	// Retrieve
	getResp, getBody := ts.getPurpose(purposeID)
	defer getResp.Body.Close()
	ts.Require().Equal(http.StatusOK, getResp.StatusCode)

	var retrieved PurposeResponse
	err = json.Unmarshal(getBody, &retrieved)
	ts.Require().NoError(err)

	// Verify all fields match
	ts.Require().Equal(purposeID, retrieved.ID)
	ts.Require().Equal(purpose.Name, retrieved.Name)
	ts.Require().NotNil(retrieved.Description)
	ts.Require().Equal(purpose.Description, *retrieved.Description)
	ts.Require().Equal(purpose.Type, retrieved.Type)
	ts.Require().Equal(purpose.Attributes["jsonPath"], retrieved.Attributes["jsonPath"])
	ts.Require().Equal(purpose.Attributes["resourcePath"], retrieved.Attributes["resourcePath"])
}

// TestCreatePurpose_ErrorCases tests various error scenarios
func (ts *PurposeAPITestSuite) TestCreatePurpose_ErrorCases() {
	validPurpose := ConsentPurposeCreateRequest{
		Name:        "test_error_valid",
		Description: "Valid purpose",
		Type:        "string",
		Attributes:  map[string]string{},
	}

	testCases := []struct {
		name            string
		payload         interface{}
		setHeaders      bool
		expectedStatus  int
		expectedCode    string
		messageContains string
	}{
		// Header validation
		{
			name:            "MissingOrgID_ReturnsValidationError",
			payload:         []ConsentPurposeCreateRequest{validPurpose},
			setHeaders:      false,
			expectedStatus:  http.StatusNotFound, // Missing route/org returns 404
			expectedCode:    "",                  // No structured error for 404
			messageContains: "",
		},

		// Request body validation
		{
			name:            "MalformedJSON_ReturnsBadRequest",
			payload:         "invalid{{{json",
			setHeaders:      true,
			expectedStatus:  http.StatusBadRequest,
			expectedCode:    "CSE-4000",
			messageContains: "invalid request body",
		},
		{
			name:            "EmptyArray_RequiresAtLeastOnePurpose",
			payload:         []ConsentPurposeCreateRequest{},
			setHeaders:      true,
			expectedStatus:  http.StatusBadRequest,
			expectedCode:    "CSE-4000",
			messageContains: "at least one purpose must be provided",
		},

		// Name validation
		{
			name: "MissingNameField_ReturnsValidationError",
			payload: []ConsentPurposeCreateRequest{
				{
					Description: "Missing name",
					Type:        "string",
					Attributes:  map[string]string{},
				},
			},
			setHeaders:      true,
			expectedStatus:  http.StatusBadRequest,
			expectedCode:    "CSE-4001",
			messageContains: "purpose name is required",
		},
		{
			name: "NameExceeds255Chars_ReturnsValidationError",
			payload: []ConsentPurposeCreateRequest{
				{
					Name:        strings.Repeat("a", 256),
					Description: "Name too long",
					Type:        "string",
					Attributes:  map[string]string{},
				},
			},
			setHeaders:      true,
			expectedStatus:  http.StatusBadRequest,
			expectedCode:    "CSE-4001",
			messageContains: "must not exceed 255 characters",
		},
		{
			name: "DuplicateNameInBatch_ReturnsConflict",
			payload: []ConsentPurposeCreateRequest{
				{
					Name:        "test_error_duplicate",
					Description: "First",
					Type:        "string",
					Attributes:  map[string]string{},
				},
				{
					Name:        "test_error_duplicate",
					Description: "Duplicate",
					Type:        "string",
					Attributes:  map[string]string{},
				},
			},
			setHeaders:      true,
			expectedStatus:  http.StatusBadRequest,
			expectedCode:    "CSE-4001",
			messageContains: "duplicate purpose name 'test_error_duplicate' in request batch",
		},

		// Type validation
		{
			name: "MissingTypeField_ReturnsValidationError",
			payload: []ConsentPurposeCreateRequest{
				{
					Name:        "test_error_no_type",
					Description: "Missing type",
					Attributes:  map[string]string{},
				},
			},
			setHeaders:      true,
			expectedStatus:  http.StatusBadRequest,
			expectedCode:    "CSE-4001",
			messageContains: "purpose type is required",
		},
		{
			name: "InvalidTypeValue_STANDARD_ReturnsValidationError",
			payload: []ConsentPurposeCreateRequest{
				{
					Name:        "test_error_invalid_type",
					Description: "Invalid type",
					Type:        "STANDARD",
					Attributes:  map[string]string{},
				},
			},
			setHeaders:      true,
			expectedStatus:  http.StatusBadRequest,
			expectedCode:    "CSE-4001",
			messageContains: "invalid purpose type",
		},

		// Type-specific attribute validation
		{
			name: "JsonSchemaType_MissingValidationSchema_ReturnsValidationError",
			payload: []ConsentPurposeCreateRequest{
				{
					Name:        "test_error_json_no_schema",
					Description: "JSON schema without validationSchema",
					Type:        "json-schema",
					Attributes:  map[string]string{},
				},
			},
			setHeaders:      true,
			expectedStatus:  http.StatusBadRequest,
			expectedCode:    "CSE-4001",
			messageContains: "validationSchema is required for json-schema type",
		},
		{
			name: "AttributeType_MissingResourcePath_ReturnsValidationError",
			payload: []ConsentPurposeCreateRequest{
				{
					Name:        "test_error_attr_no_resource",
					Description: "Attribute without resourcePath",
					Type:        "attribute",
					Attributes: map[string]string{
						"jsonPath": "$.data",
					},
				},
			},
			setHeaders:      true,
			expectedStatus:  http.StatusBadRequest,
			expectedCode:    "CSE-4001",
			messageContains: "resourcePath is required for attribute type",
		},
		{
			name: "AttributeType_MissingJsonPath_ReturnsValidationError",
			payload: []ConsentPurposeCreateRequest{
				{
					Name:        "test_error_attr_no_json",
					Description: "Attribute without jsonPath",
					Type:        "attribute",
					Attributes: map[string]string{
						"resourcePath": "/resource/{id}",
					},
				},
			},
			setHeaders:      true,
			expectedStatus:  http.StatusBadRequest,
			expectedCode:    "CSE-4001",
			messageContains: "jsonPath is required for attribute type",
		},
		{
			name: "AttributeType_MissingBothPaths_ReturnsValidationError",
			payload: []ConsentPurposeCreateRequest{
				{
					Name:        "test_error_attr_no_paths",
					Description: "Attribute without any paths",
					Type:        "attribute",
					Attributes:  map[string]string{},
				},
			},
			setHeaders:      true,
			expectedStatus:  http.StatusBadRequest,
			expectedCode:    "CSE-4001",
			messageContains: "resourcePath is required for attribute type",
		},

		// Description validation
		{
			name: "DescriptionExceeds1024Chars_ReturnsValidationError",
			payload: []ConsentPurposeCreateRequest{
				{
					Name:        "test_error_desc_1025",
					Description: strings.Repeat("x", 1025), // Exceeds 1024 char limit
					Type:        "string",
					Attributes:  map[string]string{},
				},
			},
			setHeaders:      true,
			expectedStatus:  http.StatusBadRequest,
			expectedCode:    "CSE-4001",
			messageContains: "must not exceed 1024 characters",
		},
	}

	for _, tc := range testCases {
		ts.Run(tc.name, func() {
			var resp *http.Response
			var body []byte

			if tc.setHeaders {
				resp, body = ts.createPurpose(tc.payload)
			} else {
				// Create request without headers
				var jsonData []byte
				var err error

				if str, ok := tc.payload.(string); ok {
					jsonData = []byte(str)
				} else {
					jsonData, err = json.Marshal(tc.payload)
					ts.Require().NoError(err)
				}

				req, err := http.NewRequest(http.MethodPost, testServerURL+"/consent-purposes", bytes.NewBuffer(jsonData))
				ts.Require().NoError(err)
				req.Header.Set(testutils.HeaderContentType, "application/json")
				// Deliberately not setting org-id and client-id headers

				client := &http.Client{}
				resp, err = client.Do(req)
				ts.Require().NoError(err)
				body, err = io.ReadAll(resp.Body)
				ts.Require().NoError(err)
			}
			defer resp.Body.Close()

			// Verify status code
			ts.Require().Equal(tc.expectedStatus, resp.StatusCode, "Test case: %s", tc.name)

			// Skip error response validation for 404 (no structured error)
			if tc.expectedStatus == http.StatusNotFound {
				return
			}

			// Parse error response
			var errResp ErrorResponse
			err := json.Unmarshal(body, &errResp)
			ts.Require().NoError(err, "Test case: %s", tc.name)

			// Verify error code and message
			ts.Require().Equal(tc.expectedCode, errResp.Code, "Test case: %s", tc.name)
			if tc.messageContains != "" {
				ts.Require().Contains(strings.ToLower(errResp.Description), strings.ToLower(tc.messageContains), "Test case: %s - Description: %s", tc.name, errResp.Description)
			}
			ts.Require().NotEmpty(errResp.TraceID, "Test case: %s", tc.name)
		})
	}
}
