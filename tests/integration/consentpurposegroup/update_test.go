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
	"fmt"
	"net/http"

	"github.com/stretchr/testify/require"
)

// ================================================
// PUT /consent-purpose-groups/{groupId} Tests
// ================================================

// TestUpdatePurposeGroup_ChangeName_Success tests updating the group name
func (ts *PurposeGroupAPITestSuite) TestUpdatePurposeGroup_ChangeName_Success() {
	t := ts.T()

	// Create group
	createPayload := PurposeGroupCreateRequest{
		Name:        "test_update_name_original",
		Description: "Original name",
		Purposes: []PurposeGroupPurpose{
			{PurposeName: "test_email", IsMandatory: true},
		},
	}

	createResp, body := ts.createGroup(createPayload)
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	var createdGroup PurposeGroupResponse
	json.Unmarshal(body, &createdGroup)
	ts.trackGroup(createdGroup.ID)

	// Update name
	updatePayload := PurposeGroupUpdateRequest{
		Name:        "test_update_name_modified",
		Description: "Original name",
		Purposes: []PurposeGroupPurpose{
			{PurposeName: "test_email", IsMandatory: true},
		},
	}

	resp, body := ts.updateGroup(createdGroup.ID, updatePayload)
	require.Equal(t, http.StatusOK, resp.StatusCode, "Failed to update group: %s", body)

	var updateResp PurposeGroupResponse
	err := json.Unmarshal(body, &updateResp)
	require.NoError(t, err)

	require.Equal(t, "test_update_name_modified", updateResp.Name)
	require.GreaterOrEqual(t, updateResp.UpdatedTime, createdGroup.UpdatedTime, "UpdatedTime should be equal or newer")
}

// TestUpdatePurposeGroup_ChangeDescription_Success tests updating description
func (ts *PurposeGroupAPITestSuite) TestUpdatePurposeGroup_ChangeDescription_Success() {
	t := ts.T()

	// Create group
	createPayload := PurposeGroupCreateRequest{
		Name:        "test_update_description",
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

	// Update description
	updatePayload := PurposeGroupUpdateRequest{
		Name:        "test_update_description",
		Description: "This is the new updated description",
		Purposes: []PurposeGroupPurpose{
			{PurposeName: "test_email", IsMandatory: true},
		},
	}

	resp, body := ts.updateGroup(createdGroup.ID, updatePayload)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var updateResp PurposeGroupResponse
	json.Unmarshal(body, &updateResp)

	require.Equal(t, "This is the new updated description", *updateResp.Description)
}

// TestUpdatePurposeGroup_AddPurpose_Success tests adding a purpose to the group
func (ts *PurposeGroupAPITestSuite) TestUpdatePurposeGroup_AddPurpose_Success() {
	t := ts.T()

	// Create group with single purpose
	createPayload := PurposeGroupCreateRequest{
		Name:        "test_add_purpose",
		Description: "Will add more purposes",
		Purposes: []PurposeGroupPurpose{
			{PurposeName: "test_email", IsMandatory: true},
		},
	}

	createResp, body := ts.createGroup(createPayload)
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	var createdGroup PurposeGroupResponse
	json.Unmarshal(body, &createdGroup)
	ts.trackGroup(createdGroup.ID)

	// Update to add more purposes
	updatePayload := PurposeGroupUpdateRequest{
		Name:        "test_add_purpose",
		Description: "Will add more purposes",
		Purposes: []PurposeGroupPurpose{
			{PurposeName: "test_email", IsMandatory: true},
			{PurposeName: "test_phone", IsMandatory: false},
			{PurposeName: "test_address", IsMandatory: false},
		},
	}

	resp, body := ts.updateGroup(createdGroup.ID, updatePayload)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var updateResp PurposeGroupResponse
	json.Unmarshal(body, &updateResp)

	require.Len(t, updateResp.Purposes, 3)
}

// TestUpdatePurposeGroup_RemovePurpose_Success tests removing a purpose from the group
func (ts *PurposeGroupAPITestSuite) TestUpdatePurposeGroup_RemovePurpose_Success() {
	t := ts.T()

	// Create group with multiple purposes
	createPayload := PurposeGroupCreateRequest{
		Name:        "test_remove_purpose",
		Description: "Will remove purposes",
		Purposes: []PurposeGroupPurpose{
			{PurposeName: "test_email", IsMandatory: true},
			{PurposeName: "test_phone", IsMandatory: false},
			{PurposeName: "test_address", IsMandatory: false},
		},
	}

	createResp, body := ts.createGroup(createPayload)
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	var createdGroup PurposeGroupResponse
	json.Unmarshal(body, &createdGroup)
	ts.trackGroup(createdGroup.ID)

	// Update to remove purposes
	updatePayload := PurposeGroupUpdateRequest{
		Name:        "test_remove_purpose",
		Description: "Will remove purposes",
		Purposes: []PurposeGroupPurpose{
			{PurposeName: "test_email", IsMandatory: true},
		},
	}

	resp, body := ts.updateGroup(createdGroup.ID, updatePayload)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var updateResp PurposeGroupResponse
	json.Unmarshal(body, &updateResp)

	require.Len(t, updateResp.Purposes, 1)
	require.Equal(t, "test_email", updateResp.Purposes[0].PurposeName)
}

// TestUpdatePurposeGroup_ChangeMandatoryFlag_Success tests changing isMandatory flag
func (ts *PurposeGroupAPITestSuite) TestUpdatePurposeGroup_ChangeMandatoryFlag_Success() {
	t := ts.T()

	// Create group
	createPayload := PurposeGroupCreateRequest{
		Name:        "test_change_mandatory",
		Description: "Change mandatory flag",
		Purposes: []PurposeGroupPurpose{
			{PurposeName: "test_email", IsMandatory: true},
		},
	}

	createResp, body := ts.createGroup(createPayload)
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	var createdGroup PurposeGroupResponse
	json.Unmarshal(body, &createdGroup)
	ts.trackGroup(createdGroup.ID)

	// Update to make it optional
	updatePayload := PurposeGroupUpdateRequest{
		Name:        "test_change_mandatory",
		Description: "Change mandatory flag",
		Purposes: []PurposeGroupPurpose{
			{PurposeName: "test_email", IsMandatory: false},
		},
	}

	resp, body := ts.updateGroup(createdGroup.ID, updatePayload)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var updateResp PurposeGroupResponse
	json.Unmarshal(body, &updateResp)

	require.False(t, updateResp.Purposes[0].IsMandatory)
}

// TestUpdatePurposeGroup_ReplaceAllPurposes_Success tests completely replacing purposes
func (ts *PurposeGroupAPITestSuite) TestUpdatePurposeGroup_ReplaceAllPurposes_Success() {
	t := ts.T()

	// Create group
	createPayload := PurposeGroupCreateRequest{
		Name:        "test_replace_purposes",
		Description: "Replace all purposes",
		Purposes: []PurposeGroupPurpose{
			{PurposeName: "test_email", IsMandatory: true},
			{PurposeName: "test_phone", IsMandatory: true},
		},
	}

	createResp, body := ts.createGroup(createPayload)
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	var createdGroup PurposeGroupResponse
	json.Unmarshal(body, &createdGroup)
	ts.trackGroup(createdGroup.ID)

	// Update with completely different purposes
	updatePayload := PurposeGroupUpdateRequest{
		Name:        "test_replace_purposes",
		Description: "Replace all purposes",
		Purposes: []PurposeGroupPurpose{
			{PurposeName: "test_marketing", IsMandatory: false},
			{PurposeName: "test_analytics", IsMandatory: false},
		},
	}

	resp, body := ts.updateGroup(createdGroup.ID, updatePayload)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var updateResp PurposeGroupResponse
	json.Unmarshal(body, &updateResp)

	require.Len(t, updateResp.Purposes, 2)

	// Verify old purposes are gone, new ones present
	purposeNames := make([]string, 0)
	for _, p := range updateResp.Purposes {
		purposeNames = append(purposeNames, p.PurposeName)
	}
	require.Contains(t, purposeNames, "test_marketing")
	require.Contains(t, purposeNames, "test_analytics")
	require.NotContains(t, purposeNames, "test_email")
	require.NotContains(t, purposeNames, "test_phone")
}

// TestUpdatePurposeGroup_UpdateAll_Success tests updating all fields at once
func (ts *PurposeGroupAPITestSuite) TestUpdatePurposeGroup_UpdateAll_Success() {
	t := ts.T()

	// Create group
	createPayload := PurposeGroupCreateRequest{
		Name:        "test_update_all_original",
		Description: "Original",
		Purposes: []PurposeGroupPurpose{
			{PurposeName: "test_email", IsMandatory: true},
		},
	}

	createResp, body := ts.createGroup(createPayload)
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	var createdGroup PurposeGroupResponse
	json.Unmarshal(body, &createdGroup)
	ts.trackGroup(createdGroup.ID)

	// Update everything
	updatePayload := PurposeGroupUpdateRequest{
		Name:        "test_update_all_modified",
		Description: "Completely changed",
		Purposes: []PurposeGroupPurpose{
			{PurposeName: "test_phone", IsMandatory: false},
			{PurposeName: "test_address", IsMandatory: true},
		},
	}

	resp, body := ts.updateGroup(createdGroup.ID, updatePayload)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var updateResp PurposeGroupResponse
	json.Unmarshal(body, &updateResp)

	require.Equal(t, "test_update_all_modified", updateResp.Name)
	require.Equal(t, "Completely changed", *updateResp.Description)
	require.Len(t, updateResp.Purposes, 2)
}

// TestUpdatePurposeGroup_NonExistentID_ReturnsNotFound tests updating non-existent group
func (ts *PurposeGroupAPITestSuite) TestUpdatePurposeGroup_NonExistentID_ReturnsNotFound() {
	t := ts.T()

	updatePayload := PurposeGroupUpdateRequest{
		Name:        "test_nonexistent",
		Description: "Does not exist",
		Purposes: []PurposeGroupPurpose{
			{PurposeName: "test_email", IsMandatory: true},
		},
	}

	resp, body := ts.updateGroup("GROUP-non-existent-id", updatePayload)
	require.Equal(t, http.StatusNotFound, resp.StatusCode, "Should return 404: %s", body)
}

// TestUpdatePurposeGroup_DuplicateName_DifferentGroup_Fails tests duplicate name with another group
func (ts *PurposeGroupAPITestSuite) TestUpdatePurposeGroup_DuplicateName_DifferentGroup_Fails() {
	t := ts.T()

	// Create first group
	createPayload1 := PurposeGroupCreateRequest{
		Name:        "test_update_dup_group1",
		Description: "First group",
		Purposes: []PurposeGroupPurpose{
			{PurposeName: "test_email", IsMandatory: true},
		},
	}

	createResp, body := ts.createGroup(createPayload1)
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	var group1 PurposeGroupResponse
	json.Unmarshal(body, &group1)
	ts.trackGroup(group1.ID)

	// Create second group
	createPayload2 := PurposeGroupCreateRequest{
		Name:        "test_update_dup_group2",
		Description: "Second group",
		Purposes: []PurposeGroupPurpose{
			{PurposeName: "test_phone", IsMandatory: true},
		},
	}

	createResp, body = ts.createGroup(createPayload2)
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	var group2 PurposeGroupResponse
	json.Unmarshal(body, &group2)
	ts.trackGroup(group2.ID)

	// Try to update group2 with group1's name
	updatePayload := PurposeGroupUpdateRequest{
		Name:        "test_update_dup_group1", // Same as group1
		Description: "Trying to duplicate",
		Purposes: []PurposeGroupPurpose{
			{PurposeName: "test_phone", IsMandatory: true},
		},
	}

	resp, body := ts.updateGroup(group2.ID, updatePayload)
	require.Equal(t, http.StatusConflict, resp.StatusCode, "Should reject duplicate name: %s", body)
}

// TestUpdatePurposeGroup_SameName_Success tests updating with same name (allowed)
func (ts *PurposeGroupAPITestSuite) TestUpdatePurposeGroup_SameName_Success() {
	t := ts.T()

	// Create group
	createPayload := PurposeGroupCreateRequest{
		Name:        "test_update_same_name",
		Description: "Original",
		Purposes: []PurposeGroupPurpose{
			{PurposeName: "test_email", IsMandatory: true},
		},
	}

	createResp, body := ts.createGroup(createPayload)
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	var createdGroup PurposeGroupResponse
	json.Unmarshal(body, &createdGroup)
	ts.trackGroup(createdGroup.ID)

	// Update with same name but different description
	updatePayload := PurposeGroupUpdateRequest{
		Name:        "test_update_same_name", // Same name
		Description: "Updated description",
		Purposes: []PurposeGroupPurpose{
			{PurposeName: "test_email", IsMandatory: true},
		},
	}

	resp, body := ts.updateGroup(createdGroup.ID, updatePayload)
	require.Equal(t, http.StatusOK, resp.StatusCode, "Should allow update with same name: %s", body)
}

// TestUpdatePurposeGroup_InvalidPurposeName_Fails tests updating with non-existent purpose
func (ts *PurposeGroupAPITestSuite) TestUpdatePurposeGroup_InvalidPurposeName_Fails() {
	t := ts.T()

	// Create group
	createPayload := PurposeGroupCreateRequest{
		Name:        "test_update_invalid_purpose",
		Description: "Valid purposes",
		Purposes: []PurposeGroupPurpose{
			{PurposeName: "test_email", IsMandatory: true},
		},
	}

	createResp, body := ts.createGroup(createPayload)
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	var createdGroup PurposeGroupResponse
	json.Unmarshal(body, &createdGroup)
	ts.trackGroup(createdGroup.ID)

	// Try to update with invalid purpose
	updatePayload := PurposeGroupUpdateRequest{
		Name:        "test_update_invalid_purpose",
		Description: "Invalid purposes",
		Purposes: []PurposeGroupPurpose{
			{PurposeName: "invalid_purpose_xyz", IsMandatory: true},
		},
	}

	resp, body := ts.updateGroup(createdGroup.ID, updatePayload)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode, "Should reject invalid purpose: %s", body)
}

// TestUpdatePurposeGroup_EmptyPurposes_Fails tests updating with no purposes
func (ts *PurposeGroupAPITestSuite) TestUpdatePurposeGroup_EmptyPurposes_Fails() {
	t := ts.T()

	// Create group
	createPayload := PurposeGroupCreateRequest{
		Name:        "test_update_empty_purposes",
		Description: "Has purposes",
		Purposes: []PurposeGroupPurpose{
			{PurposeName: "test_email", IsMandatory: true},
		},
	}

	createResp, body := ts.createGroup(createPayload)
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	var createdGroup PurposeGroupResponse
	json.Unmarshal(body, &createdGroup)
	ts.trackGroup(createdGroup.ID)

	// Try to update with empty purposes
	updatePayload := PurposeGroupUpdateRequest{
		Name:        "test_update_empty_purposes",
		Description: "No purposes",
		Purposes:    []PurposeGroupPurpose{},
	}

	resp, body := ts.updateGroup(createdGroup.ID, updatePayload)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode, "Should reject empty purposes: %s", body)
}

// TestUpdatePurposeGroup_DuplicatePurposeInGroup_Fails tests duplicate purpose in update
func (ts *PurposeGroupAPITestSuite) TestUpdatePurposeGroup_DuplicatePurposeInGroup_Fails() {
	t := ts.T()

	// Create group
	createPayload := PurposeGroupCreateRequest{
		Name:        "test_update_dup_purpose",
		Description: "Valid",
		Purposes: []PurposeGroupPurpose{
			{PurposeName: "test_email", IsMandatory: true},
		},
	}

	createResp, body := ts.createGroup(createPayload)
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	var createdGroup PurposeGroupResponse
	json.Unmarshal(body, &createdGroup)
	ts.trackGroup(createdGroup.ID)

	// Try to update with duplicate purposes
	updatePayload := PurposeGroupUpdateRequest{
		Name:        "test_update_dup_purpose",
		Description: "Duplicate purposes",
		Purposes: []PurposeGroupPurpose{
			{PurposeName: "test_email", IsMandatory: true},
			{PurposeName: "test_email", IsMandatory: false}, // Duplicate
		},
	}

	resp, body := ts.updateGroup(createdGroup.ID, updatePayload)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode, "Should reject duplicate purposes: %s", body)
}

// TestUpdatePurposeGroup_MissingRequiredHeaders_Fails tests missing headers
func (ts *PurposeGroupAPITestSuite) TestUpdatePurposeGroup_MissingRequiredHeaders_Fails() {
	t := ts.T()

	// Create group
	createPayload := PurposeGroupCreateRequest{
		Name: "test_update_missing_headers",
		Purposes: []PurposeGroupPurpose{
			{PurposeName: "test_email", IsMandatory: true},
		},
	}

	createResp, body := ts.createGroup(createPayload)
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	var createdGroup PurposeGroupResponse
	json.Unmarshal(body, &createdGroup)
	ts.trackGroup(createdGroup.ID)

	updatePayload := PurposeGroupUpdateRequest{
		Name: "test_update_missing_headers_mod",
		Purposes: []PurposeGroupPurpose{
			{PurposeName: "test_phone", IsMandatory: true},
		},
	}

	reqBody, _ := json.Marshal(updatePayload)

	// Try without org-id
	httpReq, _ := http.NewRequest("PUT",
		fmt.Sprintf("%s/api/v1/consent-purpose-groups/%s", testServerURL, createdGroup.ID),
		bytes.NewBuffer(reqBody))
	httpReq.Header.Set("TPP-client-id", testClientID)
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, _ := client.Do(httpReq)
	resp.Body.Close()

	require.Equal(t, http.StatusBadRequest, resp.StatusCode, "Should reject missing org-id")

	// Try without TPP-client-id
	httpReq, _ = http.NewRequest("PUT",
		fmt.Sprintf("%s/api/v1/consent-purpose-groups/%s", testServerURL, createdGroup.ID),
		bytes.NewBuffer(reqBody))
	httpReq.Header.Set("org-id", testOrgID)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, _ = client.Do(httpReq)
	resp.Body.Close()

	require.Equal(t, http.StatusBadRequest, resp.StatusCode, "Should reject missing TPP-client-id")
}
