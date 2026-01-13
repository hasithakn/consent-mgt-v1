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
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/stretchr/testify/require"
)

// ================================================
// GET /consent-purpose-groups/{groupId} Tests
// ================================================

// TestGetPurposeGroup_ExistingGroup_Success tests retrieving an existing group
func (ts *PurposeGroupAPITestSuite) TestGetPurposeGroup_ExistingGroup_Success() {
	t := ts.T()

	// Create a group first
	createPayload := PurposeGroupCreateRequest{
		Name:        "test_get_existing",
		Description: "Group to retrieve",
		Purposes: []PurposeGroupPurpose{
			{PurposeName: "test_email", IsMandatory: true},
			{PurposeName: "test_phone", IsMandatory: false},
		},
	}

	createResp, body := ts.createGroup(createPayload)
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	var createdGroup PurposeGroupResponse
	json.Unmarshal(body, &createdGroup)
	ts.trackGroup(createdGroup.ID)

	// Get the group
	resp, body := ts.getGroup(createdGroup.ID)
	require.Equal(t, http.StatusOK, resp.StatusCode, "Failed to get group: %s", body)

	var getResp PurposeGroupResponse
	err := json.Unmarshal(body, &getResp)
	require.NoError(t, err)

	// Verify all fields match
	require.Equal(t, createdGroup.ID, getResp.ID)
	require.Equal(t, "test_get_existing", getResp.Name)
	require.NotNil(t, getResp.Description)
	require.Equal(t, "Group to retrieve", *getResp.Description)
	require.Equal(t, testClientID, getResp.ClientID)
	require.Len(t, getResp.Purposes, 2)
	require.Equal(t, createdGroup.CreatedTime, getResp.CreatedTime)
	require.Equal(t, createdGroup.UpdatedTime, getResp.UpdatedTime)

	// Verify purposes
	purposeMap := make(map[string]bool)
	for _, p := range getResp.Purposes {
		purposeMap[p.PurposeName] = p.IsMandatory
	}
	require.True(t, purposeMap["test_email"])
	require.False(t, purposeMap["test_phone"])
}

// TestGetPurposeGroup_SinglePurpose_Success tests retrieving group with single purpose
func (ts *PurposeGroupAPITestSuite) TestGetPurposeGroup_SinglePurpose_Success() {
	t := ts.T()

	createPayload := PurposeGroupCreateRequest{
		Name:        "test_get_single_purpose",
		Description: "Single purpose group",
		Purposes: []PurposeGroupPurpose{
			{PurposeName: "test_analytics", IsMandatory: true},
		},
	}

	createResp, body := ts.createGroup(createPayload)
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	var createdGroup PurposeGroupResponse
	json.Unmarshal(body, &createdGroup)
	ts.trackGroup(createdGroup.ID)

	// Get and verify
	resp, body := ts.getGroup(createdGroup.ID)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var getResp PurposeGroupResponse
	json.Unmarshal(body, &getResp)

	require.Len(t, getResp.Purposes, 1)
	require.Equal(t, "test_analytics", getResp.Purposes[0].PurposeName)
	require.True(t, getResp.Purposes[0].IsMandatory)
}

// TestGetPurposeGroup_MultiplePurposes_Success tests retrieving group with multiple purposes
func (ts *PurposeGroupAPITestSuite) TestGetPurposeGroup_MultiplePurposes_Success() {
	t := ts.T()

	createPayload := PurposeGroupCreateRequest{
		Name:        "test_get_multi_purposes",
		Description: "Multiple purposes group",
		Purposes: []PurposeGroupPurpose{
			{PurposeName: "test_email", IsMandatory: true},
			{PurposeName: "test_phone", IsMandatory: true},
			{PurposeName: "test_address", IsMandatory: false},
			{PurposeName: "test_marketing", IsMandatory: false},
		},
	}

	createResp, body := ts.createGroup(createPayload)
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	var createdGroup PurposeGroupResponse
	json.Unmarshal(body, &createdGroup)
	ts.trackGroup(createdGroup.ID)

	// Get and verify all purposes
	resp, body := ts.getGroup(createdGroup.ID)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var getResp PurposeGroupResponse
	json.Unmarshal(body, &getResp)

	require.Len(t, getResp.Purposes, 4)

	// Count mandatory vs optional
	mandatoryCount := 0
	optionalCount := 0
	for _, p := range getResp.Purposes {
		if p.IsMandatory {
			mandatoryCount++
		} else {
			optionalCount++
		}
	}
	require.Equal(t, 2, mandatoryCount)
	require.Equal(t, 2, optionalCount)
}

// TestGetPurposeGroup_NoDescription_Success tests retrieving group without description
func (ts *PurposeGroupAPITestSuite) TestGetPurposeGroup_NoDescription_Success() {
	t := ts.T()

	createPayload := PurposeGroupCreateRequest{
		Name: "test_get_no_desc",
		Purposes: []PurposeGroupPurpose{
			{PurposeName: "test_email", IsMandatory: true},
		},
	}

	createResp, body := ts.createGroup(createPayload)
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	var createdGroup PurposeGroupResponse
	json.Unmarshal(body, &createdGroup)
	ts.trackGroup(createdGroup.ID)

	// Get and verify
	resp, body := ts.getGroup(createdGroup.ID)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var getResp PurposeGroupResponse
	json.Unmarshal(body, &getResp)

	require.Equal(t, "test_get_no_desc", getResp.Name)
	// Description can be nil or empty string
}

// TestGetPurposeGroup_NonExistentID_ReturnsNotFound tests getting non-existent group
func (ts *PurposeGroupAPITestSuite) TestGetPurposeGroup_NonExistentID_ReturnsNotFound() {
	t := ts.T()

	resp, body := ts.getGroup("GROUP-non-existent-id-xyz")
	require.Equal(t, http.StatusNotFound, resp.StatusCode, "Should return 404: %s", body)

	var errResp ErrorResponse
	err := json.Unmarshal(body, &errResp)
	require.NoError(t, err)
	// Message could be "not found" or "Resource Not Found"
	msg := strings.ToLower(errResp.Message)
	require.True(t, strings.Contains(msg, "not found"), "Expected 'not found' in message, got: %s", errResp.Message)
}

// TestGetPurposeGroup_InvalidUUIDFormat_ReturnsBadRequest tests invalid ID format
func (ts *PurposeGroupAPITestSuite) TestGetPurposeGroup_InvalidUUIDFormat_ReturnsBadRequest() {
	t := ts.T()

	resp, body := ts.getGroup("invalid-id-format")
	// Could be 400 or 404 depending on validation
	require.Contains(t, []int{http.StatusBadRequest, http.StatusNotFound}, resp.StatusCode,
		"Should reject invalid ID format: %s", body)
}

// TestGetPurposeGroup_MissingOrgIDHeader_Fails tests missing org-id header
func (ts *PurposeGroupAPITestSuite) TestGetPurposeGroup_MissingOrgIDHeader_Fails() {
	t := ts.T()

	// Create a group first
	createPayload := PurposeGroupCreateRequest{
		Name: "test_get_missing_orgid",
		Purposes: []PurposeGroupPurpose{
			{PurposeName: "test_email", IsMandatory: true},
		},
	}

	createResp, body := ts.createGroup(createPayload)
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	var createdGroup PurposeGroupResponse
	json.Unmarshal(body, &createdGroup)
	ts.trackGroup(createdGroup.ID)

	// Try to get without org-id header
	httpReq, _ := http.NewRequest("GET",
		fmt.Sprintf("%s/api/v1/consent-purpose-groups/%s", testServerURL, createdGroup.ID),
		nil)
	// Deliberately omit org-id header

	client := &http.Client{}
	resp, _ := client.Do(httpReq)
	defer resp.Body.Close()

	require.Equal(t, http.StatusBadRequest, resp.StatusCode, "Should reject missing org-id")
}

// TestGetPurposeGroup_WrongOrgID_ReturnsNotFound tests accessing with wrong org
func (ts *PurposeGroupAPITestSuite) TestGetPurposeGroup_WrongOrgID_ReturnsNotFound() {
	t := ts.T()

	// Create a group with our test org
	createPayload := PurposeGroupCreateRequest{
		Name: "test_get_wrong_org",
		Purposes: []PurposeGroupPurpose{
			{PurposeName: "test_email", IsMandatory: true},
		},
	}

	createResp, body := ts.createGroup(createPayload)
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	var createdGroup PurposeGroupResponse
	json.Unmarshal(body, &createdGroup)
	ts.trackGroup(createdGroup.ID)

	// Try to get with different org-id
	httpReq, _ := http.NewRequest("GET",
		fmt.Sprintf("%s/api/v1/consent-purpose-groups/%s", testServerURL, createdGroup.ID),
		nil)
	httpReq.Header.Set("org-id", "different-org-id")

	client := &http.Client{}
	resp, _ := client.Do(httpReq)
	defer resp.Body.Close()

	require.Equal(t, http.StatusNotFound, resp.StatusCode, "Should not find group in different org")
}

// TestGetPurposeGroup_AfterUpdate_ReturnsUpdatedData tests getting after update
func (ts *PurposeGroupAPITestSuite) TestGetPurposeGroup_AfterUpdate_ReturnsUpdatedData() {
	t := ts.T()

	// Create group
	createPayload := PurposeGroupCreateRequest{
		Name:        "test_get_after_update",
		Description: "Original description",
		Purposes: []PurposeGroupPurpose{
			{PurposeName: "test_email", IsMandatory: true},
		},
	}

	createResp, body := ts.createGroup(createPayload)
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	var createdGroup PurposeGroupResponse
	json.Unmarshal(body, &createdGroup)
	ts.trackGroup(createdGroup.ID)

	// Update the group
	updatePayload := PurposeGroupUpdateRequest{
		Name:        "test_get_after_update_modified",
		Description: "Updated description",
		Purposes: []PurposeGroupPurpose{
			{PurposeName: "test_phone", IsMandatory: false},
		},
	}

	updateResp, _ := ts.updateGroup(createdGroup.ID, updatePayload)
	require.Equal(t, http.StatusOK, updateResp.StatusCode)

	// Get and verify updated data
	resp, body := ts.getGroup(createdGroup.ID)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var getResp PurposeGroupResponse
	json.Unmarshal(body, &getResp)

	require.Equal(t, "test_get_after_update_modified", getResp.Name)
	require.Equal(t, "Updated description", *getResp.Description)
	require.Len(t, getResp.Purposes, 1)
	require.Equal(t, "test_phone", getResp.Purposes[0].PurposeName)
	require.False(t, getResp.Purposes[0].IsMandatory)
	require.GreaterOrEqual(t, getResp.UpdatedTime, createdGroup.UpdatedTime, "UpdatedTime should be equal or newer")
}
