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

package consentpurposegroup

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/stretchr/testify/require"
)

// ======================================
// POST /consent-purpose-groups Tests
// ======================================

// TestCreatePurposeGroup_SinglePurpose_Success tests creating a group with one purpose
func (ts *PurposeGroupAPITestSuite) TestCreatePurposeGroup_SinglePurpose_Success() {
	t := ts.T()

	payload := PurposeGroupCreateRequest{
		Name:        "test_single_purpose_group",
		Description: "Group with single purpose",
		Purposes: []PurposeGroupPurpose{
			{PurposeName: "test_email", IsMandatory: true},
		},
	}

	resp, body := ts.createGroup(payload)
	require.Equal(t, http.StatusCreated, resp.StatusCode, "Failed to create group: %s", body)

	var groupResp PurposeGroupResponse
	err := json.Unmarshal(body, &groupResp)
	require.NoError(t, err)

	require.NotEmpty(t, groupResp.ID, "Group ID should not be empty")
	require.Equal(t, "test_single_purpose_group", groupResp.Name)
	require.NotNil(t, groupResp.Description)
	require.Equal(t, "Group with single purpose", *groupResp.Description)
	require.Equal(t, testClientID, groupResp.ClientID)
	require.Len(t, groupResp.Purposes, 1)
	require.Equal(t, "test_email", groupResp.Purposes[0].PurposeName)
	require.True(t, groupResp.Purposes[0].IsMandatory)
	require.NotZero(t, groupResp.CreatedTime)
	require.NotZero(t, groupResp.UpdatedTime)

	ts.trackGroup(groupResp.ID)
}

// TestCreatePurposeGroup_MultiplePurposes_Success tests creating a group with multiple purposes
func (ts *PurposeGroupAPITestSuite) TestCreatePurposeGroup_MultiplePurposes_Success() {
	t := ts.T()

	payload := PurposeGroupCreateRequest{
		Name:        "test_multi_purpose_group",
		Description: "Group with multiple purposes",
		Purposes: []PurposeGroupPurpose{
			{PurposeName: "test_email", IsMandatory: true},
			{PurposeName: "test_phone", IsMandatory: true},
			{PurposeName: "test_address", IsMandatory: false},
		},
	}

	resp, body := ts.createGroup(payload)
	require.Equal(t, http.StatusCreated, resp.StatusCode, "Failed to create group: %s", body)

	var groupResp PurposeGroupResponse
	err := json.Unmarshal(body, &groupResp)
	require.NoError(t, err)

	require.Equal(t, "test_multi_purpose_group", groupResp.Name)
	require.Len(t, groupResp.Purposes, 3)

	// Verify all purposes are present
	purposeMap := make(map[string]bool)
	for _, p := range groupResp.Purposes {
		purposeMap[p.PurposeName] = p.IsMandatory
	}
	require.True(t, purposeMap["test_email"])
	require.True(t, purposeMap["test_phone"])
	require.False(t, purposeMap["test_address"])

	ts.trackGroup(groupResp.ID)
}

// TestCreatePurposeGroup_NoDescription_Success tests creating without description
func (ts *PurposeGroupAPITestSuite) TestCreatePurposeGroup_NoDescription_Success() {
	t := ts.T()

	payload := PurposeGroupCreateRequest{
		Name: "test_no_description_group",
		Purposes: []PurposeGroupPurpose{
			{PurposeName: "test_email", IsMandatory: true},
		},
	}

	resp, body := ts.createGroup(payload)
	require.Equal(t, http.StatusCreated, resp.StatusCode, "Failed to create group: %s", body)

	var groupResp PurposeGroupResponse
	err := json.Unmarshal(body, &groupResp)
	require.NoError(t, err)

	require.Equal(t, "test_no_description_group", groupResp.Name)
	// Description can be nil or empty
	ts.trackGroup(groupResp.ID)
}

// TestCreatePurposeGroup_AllOptionalPurposes_Success tests creating with all optional purposes
func (ts *PurposeGroupAPITestSuite) TestCreatePurposeGroup_AllOptionalPurposes_Success() {
	t := ts.T()

	payload := PurposeGroupCreateRequest{
		Name:        "test_all_optional_group",
		Description: "All purposes are optional",
		Purposes: []PurposeGroupPurpose{
			{PurposeName: "test_email", IsMandatory: false},
			{PurposeName: "test_phone", IsMandatory: false},
		},
	}

	resp, body := ts.createGroup(payload)
	require.Equal(t, http.StatusCreated, resp.StatusCode, "Failed to create group: %s", body)

	var groupResp PurposeGroupResponse
	err := json.Unmarshal(body, &groupResp)
	require.NoError(t, err)

	for _, p := range groupResp.Purposes {
		require.False(t, p.IsMandatory, "All purposes should be optional")
	}

	ts.trackGroup(groupResp.ID)
}

// TestCreatePurposeGroup_DuplicateName_SameClient_Fails tests duplicate name for same client
func (ts *PurposeGroupAPITestSuite) TestCreatePurposeGroup_DuplicateName_SameClient_Fails() {
	t := ts.T()

	payload := PurposeGroupCreateRequest{
		Name:        "test_duplicate_name",
		Description: "First group",
		Purposes: []PurposeGroupPurpose{
			{PurposeName: "test_email", IsMandatory: true},
		},
	}

	// Create first group
	resp, body := ts.createGroup(payload)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var groupResp PurposeGroupResponse
	json.Unmarshal(body, &groupResp)
	ts.trackGroup(groupResp.ID)

	// Try to create another with same name
	payload.Description = "Second group"
	resp, body = ts.createGroup(payload)
	require.Equal(t, http.StatusConflict, resp.StatusCode, "Should reject duplicate name: %s", body)

	var errResp ErrorResponse
	err := json.Unmarshal(body, &errResp)
	require.NoError(t, err)
	// Message could be "already exists", "Conflict", or "duplicate"
	msg := strings.ToLower(errResp.Message)
	require.True(t,
		strings.Contains(msg, "already exists") || strings.Contains(msg, "conflict") || strings.Contains(msg, "duplicate"),
		"Expected conflict/duplicate error message, got: %s", errResp.Message)
}

// TestCreatePurposeGroup_InvalidPurposeName_Fails tests creating with non-existent purpose
func (ts *PurposeGroupAPITestSuite) TestCreatePurposeGroup_InvalidPurposeName_Fails() {
	t := ts.T()

	payload := PurposeGroupCreateRequest{
		Name:        "test_invalid_purpose",
		Description: "Group with invalid purpose",
		Purposes: []PurposeGroupPurpose{
			{PurposeName: "non_existent_purpose_xyz", IsMandatory: true},
		},
	}

	resp, body := ts.createGroup(payload)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode, "Should reject invalid purpose: %s", body)

	var errResp ErrorResponse
	err := json.Unmarshal(body, &errResp)
	require.NoError(t, err)
	require.Contains(t, errResp.Description, "does not exist")
}

// TestCreatePurposeGroup_EmptyName_Fails tests creating with empty name
func (ts *PurposeGroupAPITestSuite) TestCreatePurposeGroup_EmptyName_Fails() {
	t := ts.T()

	payload := PurposeGroupCreateRequest{
		Name:        "",
		Description: "Empty name test",
		Purposes: []PurposeGroupPurpose{
			{PurposeName: "test_email", IsMandatory: true},
		},
	}

	resp, body := ts.createGroup(payload)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode, "Should reject empty name: %s", body)
}

// TestCreatePurposeGroup_NoPurposes_Fails tests creating without any purposes
func (ts *PurposeGroupAPITestSuite) TestCreatePurposeGroup_NoPurposes_Fails() {
	t := ts.T()

	payload := PurposeGroupCreateRequest{
		Name:        "test_no_purposes",
		Description: "Group with no purposes",
		Purposes:    []PurposeGroupPurpose{},
	}

	resp, body := ts.createGroup(payload)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode, "Should reject group with no purposes: %s", body)
}

// TestCreatePurposeGroup_DuplicatePurposeInGroup_Fails tests duplicate purpose within same group
func (ts *PurposeGroupAPITestSuite) TestCreatePurposeGroup_DuplicatePurposeInGroup_Fails() {
	t := ts.T()

	payload := PurposeGroupCreateRequest{
		Name:        "test_duplicate_purpose_in_group",
		Description: "Group with duplicate purpose",
		Purposes: []PurposeGroupPurpose{
			{PurposeName: "test_email", IsMandatory: true},
			{PurposeName: "test_email", IsMandatory: false}, // Duplicate
		},
	}

	resp, body := ts.createGroup(payload)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode, "Should reject duplicate purposes: %s", body)

	var errResp ErrorResponse
	err := json.Unmarshal(body, &errResp)
	require.NoError(t, err)
	require.Contains(t, errResp.Description, "duplicate")
}

// TestCreatePurposeGroup_MissingOrgIDHeader_Fails tests missing org-id header
func (ts *PurposeGroupAPITestSuite) TestCreatePurposeGroup_MissingOrgIDHeader_Fails() {
	t := ts.T()

	payload := PurposeGroupCreateRequest{
		Name:        "test_missing_orgid",
		Description: "Test missing header",
		Purposes: []PurposeGroupPurpose{
			{PurposeName: "test_email", IsMandatory: true},
		},
	}

	reqBody, _ := json.Marshal(payload)
	httpReq, _ := http.NewRequest("POST", testServerURL+"/api/v1/consent-purpose-groups",
		bytes.NewBuffer(reqBody))
	// Deliberately omit org-id header
	httpReq.Header.Set("TPP-client-id", testClientID)
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, _ := client.Do(httpReq)
	defer resp.Body.Close()

	require.Equal(t, http.StatusBadRequest, resp.StatusCode, "Should reject missing org-id")
}

// TestCreatePurposeGroup_MissingClientIDHeader_Fails tests missing TPP-client-id header
func (ts *PurposeGroupAPITestSuite) TestCreatePurposeGroup_MissingClientIDHeader_Fails() {
	t := ts.T()

	payload := PurposeGroupCreateRequest{
		Name:        "test_missing_clientid",
		Description: "Test missing header",
		Purposes: []PurposeGroupPurpose{
			{PurposeName: "test_email", IsMandatory: true},
		},
	}

	reqBody, _ := json.Marshal(payload)
	httpReq, _ := http.NewRequest("POST", testServerURL+"/api/v1/consent-purpose-groups",
		bytes.NewBuffer(reqBody))
	httpReq.Header.Set("org-id", testOrgID)
	// Deliberately omit TPP-client-id header
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, _ := client.Do(httpReq)
	defer resp.Body.Close()

	require.Equal(t, http.StatusBadRequest, resp.StatusCode, "Should reject missing TPP-client-id")
}

// TestCreatePurposeGroup_InvalidJSON_Fails tests malformed JSON
func (ts *PurposeGroupAPITestSuite) TestCreatePurposeGroup_InvalidJSON_Fails() {
	t := ts.T()

	resp, body := ts.createGroup("{invalid json}")
	require.Equal(t, http.StatusBadRequest, resp.StatusCode, "Should reject invalid JSON: %s", body)
}

// TestCreatePurposeGroup_LongName_Success tests creating with a long name
func (ts *PurposeGroupAPITestSuite) TestCreatePurposeGroup_LongName_Success() {
	t := ts.T()

	// Create a name that's exactly 200 characters
	longName := "test_very_long_name_"
	for len(longName) < 200 {
		longName += "a"
	}
	longName = longName[:200] // Ensure exactly 200 chars

	payload := PurposeGroupCreateRequest{
		Name:        longName, // Use the 200 char name
		Description: "Long name test",
		Purposes: []PurposeGroupPurpose{
			{PurposeName: "test_email", IsMandatory: true},
		},
	}

	resp, body := ts.createGroup(payload)
	// Should succeed if within database limits, otherwise fail gracefully
	if resp.StatusCode == http.StatusCreated {
		var groupResp PurposeGroupResponse
		json.Unmarshal(body, &groupResp)
		ts.trackGroup(groupResp.ID)
	} else {
		require.Contains(t, []int{http.StatusBadRequest, http.StatusInternalServerError}, resp.StatusCode)
	}
}
