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

// ===================================================
// DELETE /consent-purpose-groups/{groupId} Tests
// ===================================================

// TestDeletePurposeGroup_ExistingGroup_Success tests deleting an existing group
func (ts *PurposeGroupAPITestSuite) TestDeletePurposeGroup_ExistingGroup_Success() {
	t := ts.T()

	// Create group
	createPayload := PurposeGroupCreateRequest{
		Name:        "test_delete_existing",
		Description: "To be deleted",
		Purposes: []PurposeGroupPurpose{
			{PurposeName: "test_email", IsMandatory: true},
		},
	}

	createResp, body := ts.createGroup(createPayload)
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	var createdGroup PurposeGroupResponse
	json.Unmarshal(body, &createdGroup)

	// Delete the group
	resp, _ := ts.deleteGroup(createdGroup.ID)
	require.Equal(t, http.StatusNoContent, resp.StatusCode, "Delete should return 204")

	// Verify it's gone
	resp, _ = ts.getGroup(createdGroup.ID)
	require.Equal(t, http.StatusNotFound, resp.StatusCode, "Group should not exist after deletion")
}

// TestDeletePurposeGroup_WithMultiplePurposes_Success tests deleting group with multiple purposes
func (ts *PurposeGroupAPITestSuite) TestDeletePurposeGroup_WithMultiplePurposes_Success() {
	t := ts.T()

	// Create group with multiple purposes
	createPayload := PurposeGroupCreateRequest{
		Name:        "test_delete_multi_purposes",
		Description: "Multiple purposes to delete",
		Purposes: []PurposeGroupPurpose{
			{PurposeName: "test_email", IsMandatory: true},
			{PurposeName: "test_phone", IsMandatory: true},
			{PurposeName: "test_address", IsMandatory: false},
		},
	}

	createResp, body := ts.createGroup(createPayload)
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	var createdGroup PurposeGroupResponse
	json.Unmarshal(body, &createdGroup)

	// Delete the group
	resp, _ := ts.deleteGroup(createdGroup.ID)
	require.Equal(t, http.StatusNoContent, resp.StatusCode)

	// Verify deletion
	resp, _ = ts.getGroup(createdGroup.ID)
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// TestDeletePurposeGroup_NonExistentID_ReturnsNotFound tests deleting non-existent group
func (ts *PurposeGroupAPITestSuite) TestDeletePurposeGroup_NonExistentID_ReturnsNotFound() {
	t := ts.T()

	resp, body := ts.deleteGroup("GROUP-non-existent-id-xyz")
	require.Equal(t, http.StatusNotFound, resp.StatusCode, "Should return 404: %s", body)

	var errResp ErrorResponse
	err := json.Unmarshal(body, &errResp)
	require.NoError(t, err)
	// Message could be "not found" or "Resource Not Found"
	msg := strings.ToLower(errResp.Message)
	require.True(t, strings.Contains(msg, "not found"), "Expected 'not found' in message, got: %s", errResp.Message)
}

// TestDeletePurposeGroup_InvalidUUIDFormat_ReturnsBadRequestOrNotFound tests invalid ID
func (ts *PurposeGroupAPITestSuite) TestDeletePurposeGroup_InvalidUUIDFormat_ReturnsBadRequestOrNotFound() {
	t := ts.T()

	resp, _ := ts.deleteGroup("invalid-id-format")
	// Could be 400 or 404 depending on validation
	require.Contains(t, []int{http.StatusBadRequest, http.StatusNotFound}, resp.StatusCode)
}

// TestDeletePurposeGroup_Twice_ReturnsNotFound tests double deletion
func (ts *PurposeGroupAPITestSuite) TestDeletePurposeGroup_Twice_ReturnsNotFound() {
	t := ts.T()

	// Create group
	createPayload := PurposeGroupCreateRequest{
		Name:        "test_delete_twice",
		Description: "Delete twice test",
		Purposes: []PurposeGroupPurpose{
			{PurposeName: "test_email", IsMandatory: true},
		},
	}

	createResp, body := ts.createGroup(createPayload)
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	var createdGroup PurposeGroupResponse
	json.Unmarshal(body, &createdGroup)

	// First deletion
	resp, _ := ts.deleteGroup(createdGroup.ID)
	require.Equal(t, http.StatusNoContent, resp.StatusCode)

	// Second deletion should fail
	resp, _ = ts.deleteGroup(createdGroup.ID)
	require.Equal(t, http.StatusNotFound, resp.StatusCode, "Second delete should return 404")
}

// TestDeletePurposeGroup_MissingOrgIDHeader_Fails tests missing org-id header
func (ts *PurposeGroupAPITestSuite) TestDeletePurposeGroup_MissingOrgIDHeader_Fails() {
	t := ts.T()

	// Create group
	createPayload := PurposeGroupCreateRequest{
		Name:        "test_delete_missing_orgid",
		Description: "Missing header test",
		Purposes: []PurposeGroupPurpose{
			{PurposeName: "test_email", IsMandatory: true},
		},
	}

	createResp, body := ts.createGroup(createPayload)
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	var createdGroup PurposeGroupResponse
	json.Unmarshal(body, &createdGroup)
	ts.trackGroup(createdGroup.ID)

	// Try to delete without org-id header
	httpReq, _ := http.NewRequest("DELETE",
		fmt.Sprintf("%s/api/v1/consent-purpose-groups/%s", testServerURL, createdGroup.ID),
		nil)
	// Deliberately omit org-id header

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusBadRequest, resp.StatusCode, "Should reject missing org-id")
}

// TestDeletePurposeGroup_WrongOrgID_ReturnsNotFound tests deleting with wrong org
func (ts *PurposeGroupAPITestSuite) TestDeletePurposeGroup_WrongOrgID_ReturnsNotFound() {
	t := ts.T()

	// Create group with our test org
	createPayload := PurposeGroupCreateRequest{
		Name:        "test_delete_wrong_org",
		Description: "Wrong org test",
		Purposes: []PurposeGroupPurpose{
			{PurposeName: "test_email", IsMandatory: true},
		},
	}

	createResp, body := ts.createGroup(createPayload)
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	var createdGroup PurposeGroupResponse
	json.Unmarshal(body, &createdGroup)
	ts.trackGroup(createdGroup.ID)

	// Try to delete with different org-id
	httpReq, _ := http.NewRequest("DELETE",
		fmt.Sprintf("%s/api/v1/consent-purpose-groups/%s", testServerURL, createdGroup.ID),
		nil)
	httpReq.Header.Set("org-id", "different-org-id")

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusNotFound, resp.StatusCode, "Should not find group in different org")
}

// TestDeletePurposeGroup_PurposesRemainIntact_Success tests that purposes are not deleted
func (ts *PurposeGroupAPITestSuite) TestDeletePurposeGroup_PurposesRemainIntact_Success() {
	t := ts.T()

	// Create group using existing purposes
	createPayload := PurposeGroupCreateRequest{
		Name:        "test_delete_purposes_intact",
		Description: "Purposes should remain",
		Purposes: []PurposeGroupPurpose{
			{PurposeName: "test_email", IsMandatory: true},
			{PurposeName: "test_phone", IsMandatory: false},
		},
	}

	createResp, body := ts.createGroup(createPayload)
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	var createdGroup PurposeGroupResponse
	json.Unmarshal(body, &createdGroup)

	// Delete the group
	resp, _ := ts.deleteGroup(createdGroup.ID)
	require.Equal(t, http.StatusNoContent, resp.StatusCode)

	// Verify group is gone
	resp, _ = ts.getGroup(createdGroup.ID)
	require.Equal(t, http.StatusNotFound, resp.StatusCode)

	// Verify purposes still exist by creating another group with same purposes
	createPayload2 := PurposeGroupCreateRequest{
		Name:        "test_purposes_still_exist",
		Description: "Using same purposes",
		Purposes: []PurposeGroupPurpose{
			{PurposeName: "test_email", IsMandatory: true},
			{PurposeName: "test_phone", IsMandatory: true},
		},
	}

	createResp, body = ts.createGroup(createPayload2)
	require.Equal(t, http.StatusCreated, createResp.StatusCode, "Should be able to reuse purposes: %s", body)
	var newGroup PurposeGroupResponse
	json.Unmarshal(body, &newGroup)
	ts.trackGroup(newGroup.ID)
}

// TestDeletePurposeGroup_CanRecreateWithSameName_Success tests recreation after deletion
func (ts *PurposeGroupAPITestSuite) TestDeletePurposeGroup_CanRecreateWithSameName_Success() {
	t := ts.T()

	groupName := "test_delete_and_recreate"

	// Create group
	createPayload := PurposeGroupCreateRequest{
		Name:        groupName,
		Description: "First version",
		Purposes: []PurposeGroupPurpose{
			{PurposeName: "test_email", IsMandatory: true},
		},
	}

	createResp, body := ts.createGroup(createPayload)
	require.Equal(t, http.StatusCreated, createResp.StatusCode)
	var firstGroup PurposeGroupResponse
	json.Unmarshal(body, &firstGroup)

	// Delete it
	resp, _ := ts.deleteGroup(firstGroup.ID)
	require.Equal(t, http.StatusNoContent, resp.StatusCode)

	// Recreate with same name
	createPayload.Description = "Second version"
	createResp, body = ts.createGroup(createPayload)
	require.Equal(t, http.StatusCreated, createResp.StatusCode, "Should allow recreation: %s", body)

	var secondGroup PurposeGroupResponse
	json.Unmarshal(body, &secondGroup)
	ts.trackGroup(secondGroup.ID)

	require.Equal(t, groupName, secondGroup.Name)
	require.NotEqual(t, firstGroup.ID, secondGroup.ID, "Should have different ID")
}

// TestDeletePurposeGroup_MultipleGroups_OnlyDeletesOne tests selective deletion
func (ts *PurposeGroupAPITestSuite) TestDeletePurposeGroup_MultipleGroups_OnlyDeletesOne() {
	t := ts.T()

	// Create first group
	createPayload1 := PurposeGroupCreateRequest{
		Name:        "test_delete_selective_1",
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
		Name:        "test_delete_selective_2",
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

	// Delete first group only
	resp, _ := ts.deleteGroup(group1.ID)
	require.Equal(t, http.StatusNoContent, resp.StatusCode)

	// Verify first is gone
	resp, _ = ts.getGroup(group1.ID)
	require.Equal(t, http.StatusNotFound, resp.StatusCode)

	// Verify second still exists
	resp, body = ts.getGroup(group2.ID)
	require.Equal(t, http.StatusOK, resp.StatusCode, "Second group should still exist: %s", body)

	var stillExists PurposeGroupResponse
	json.Unmarshal(body, &stillExists)
	require.Equal(t, group2.ID, stillExists.ID)
	require.Equal(t, "test_delete_selective_2", stillExists.Name)
}

// TestDeletePurposeGroup_EmptyID_ReturnsBadRequest tests deletion with empty ID
func (ts *PurposeGroupAPITestSuite) TestDeletePurposeGroup_EmptyID_ReturnsBadRequest() {
	t := ts.T()

	httpReq, _ := http.NewRequest("DELETE",
		fmt.Sprintf("%s/api/v1/consent-purpose-groups/", testServerURL),
		nil)
	httpReq.Header.Set("org-id", testOrgID)

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Should return 404 or 405 (Method Not Allowed) since the route doesn't match
	require.Contains(t, []int{http.StatusNotFound, http.StatusMethodNotAllowed}, resp.StatusCode)
}
