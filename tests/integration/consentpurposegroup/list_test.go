package consentpurposegroup

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/stretchr/testify/require"
)

// =========================================
// GET /consent-purpose-groups Tests (List)
// =========================================

// TestListPurposeGroups_NoFilters_ReturnsAllGroups tests listing all purpose groups without filters
func (ts *PurposeGroupAPITestSuite) TestListPurposeGroups_NoFilters_ReturnsAllGroups() {
	t := ts.T()

	// Create multiple groups
	group1Payload := PurposeGroupCreateRequest{
		Name:        "test_marketing_group",
		Description: "Marketing communications group",
		Purposes: []PurposeGroupPurpose{
			{PurposeName: "test_email", IsMandatory: true},
			{PurposeName: "test_phone", IsMandatory: false},
		},
	}

	group2Payload := PurposeGroupCreateRequest{
		Name:        "test_analytics_group",
		Description: "Analytics group",
		Purposes: []PurposeGroupPurpose{
			{PurposeName: "test_analytics", IsMandatory: true},
		},
	}

	// Create first group
	resp, body := ts.createGroup(group1Payload)
	require.Equal(t, http.StatusCreated, resp.StatusCode, "Failed to create group1: %s", body)
	var group1Resp PurposeGroupResponse
	err := json.Unmarshal(body, &group1Resp)
	require.NoError(t, err)
	ts.trackGroup(group1Resp.ID)

	// Create second group
	resp, body = ts.createGroup(group2Payload)
	require.Equal(t, http.StatusCreated, resp.StatusCode, "Failed to create group2: %s", body)
	var group2Resp PurposeGroupResponse
	err = json.Unmarshal(body, &group2Resp)
	require.NoError(t, err)
	ts.trackGroup(group2Resp.ID)

	// List all groups
	resp, body = ts.listGroups("", nil, nil, 100, 0)
	require.Equal(t, http.StatusOK, resp.StatusCode, "Failed to list groups: %s", body)

	var listResp PurposeGroupListResponse
	err = json.Unmarshal(body, &listResp)
	require.NoError(t, err)

	// Should have at least our 2 groups
	require.GreaterOrEqual(t, len(listResp.Data), 2, "Expected at least 2 groups")
	require.GreaterOrEqual(t, listResp.Metadata.Total, 2, "Total should be at least 2")

	// Find our groups in the list
	foundGroup1 := false
	foundGroup2 := false
	for _, group := range listResp.Data {
		if group.ID == group1Resp.ID {
			foundGroup1 = true
			require.Equal(t, "test_marketing_group", group.Name)
			require.Len(t, group.Purposes, 2)
		}
		if group.ID == group2Resp.ID {
			foundGroup2 = true
			require.Equal(t, "test_analytics_group", group.Name)
			require.Len(t, group.Purposes, 1)
		}
	}
	require.True(t, foundGroup1, "Group 1 not found in list")
	require.True(t, foundGroup2, "Group 2 not found in list")
}

// TestListPurposeGroups_FilterByName_ReturnsMatchingGroup tests filtering by exact name
func (ts *PurposeGroupAPITestSuite) TestListPurposeGroups_FilterByName_ReturnsMatchingGroup() {
	t := ts.T()

	// Create groups with unique names
	group1Payload := PurposeGroupCreateRequest{
		Name:        "test_filter_by_name_1",
		Description: "First group",
		Purposes: []PurposeGroupPurpose{
			{PurposeName: "test_email", IsMandatory: true},
		},
	}

	group2Payload := PurposeGroupCreateRequest{
		Name:        "test_filter_by_name_2",
		Description: "Second group",
		Purposes: []PurposeGroupPurpose{
			{PurposeName: "test_phone", IsMandatory: true},
		},
	}

	// Create groups
	resp, body := ts.createGroup(group1Payload)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var group1Resp PurposeGroupResponse
	json.Unmarshal(body, &group1Resp)
	ts.trackGroup(group1Resp.ID)

	resp, body = ts.createGroup(group2Payload)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var group2Resp PurposeGroupResponse
	json.Unmarshal(body, &group2Resp)
	ts.trackGroup(group2Resp.ID)

	// Filter by first group name
	resp, body = ts.listGroups("test_filter_by_name_1", nil, nil, 100, 0)
	require.Equal(t, http.StatusOK, resp.StatusCode, "Failed to list groups: %s", body)

	var listResp PurposeGroupListResponse
	err := json.Unmarshal(body, &listResp)
	require.NoError(t, err)

	// Should only return the matching group
	require.GreaterOrEqual(t, len(listResp.Data), 1, "Expected at least 1 group")

	// Find our group in the results
	found := false
	for _, g := range listResp.Data {
		if g.ID == group1Resp.ID && g.Name == "test_filter_by_name_1" {
			found = true
			break
		}
	}
	require.True(t, found, "Should find the created group with matching name")
}

// TestListPurposeGroups_FilterByClientID_ReturnsMatchingGroups tests filtering by clientID
func (ts *PurposeGroupAPITestSuite) TestListPurposeGroups_FilterByClientID_ReturnsMatchingGroups() {
	t := ts.T()

	// Create group with default client
	group1Payload := PurposeGroupCreateRequest{
		Name:        "test_client_filter_default",
		Description: "Group for default client",
		Purposes: []PurposeGroupPurpose{
			{PurposeName: "test_email", IsMandatory: true},
		},
	}

	resp, body := ts.createGroup(group1Payload)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var group1Resp PurposeGroupResponse
	json.Unmarshal(body, &group1Resp)
	ts.trackGroup(group1Resp.ID)

	// List groups filtered by our test client ID
	resp, body = ts.listGroups("", []string{testClientID}, nil, 100, 0)
	require.Equal(t, http.StatusOK, resp.StatusCode, "Failed to list groups: %s", body)

	var listResp PurposeGroupListResponse
	err := json.Unmarshal(body, &listResp)
	require.NoError(t, err)

	// Should have at least our group
	require.GreaterOrEqual(t, len(listResp.Data), 1, "Expected at least 1 group")

	// All returned groups should have our client ID
	foundOurGroup := false
	for _, group := range listResp.Data {
		require.Equal(t, testClientID, group.ClientID, "All groups should have the filtered client ID")
		if group.ID == group1Resp.ID {
			foundOurGroup = true
		}
	}
	require.True(t, foundOurGroup, "Our group not found in filtered results")
}

// TestListPurposeGroups_FilterBySinglePurposeName_ReturnsGroupsContainingPurpose tests filtering by single purpose
func (ts *PurposeGroupAPITestSuite) TestListPurposeGroups_FilterBySinglePurposeName_ReturnsGroupsContainingPurpose() {
	t := ts.T()

	// Create groups with different purposes
	group1Payload := PurposeGroupCreateRequest{
		Name:        "test_single_purpose_filter_1",
		Description: "Group with email",
		Purposes: []PurposeGroupPurpose{
			{PurposeName: "test_email", IsMandatory: true},
			{PurposeName: "test_phone", IsMandatory: false},
		},
	}

	group2Payload := PurposeGroupCreateRequest{
		Name:        "test_single_purpose_filter_2",
		Description: "Group with email and address",
		Purposes: []PurposeGroupPurpose{
			{PurposeName: "test_email", IsMandatory: true},
			{PurposeName: "test_address", IsMandatory: false},
		},
	}

	group3Payload := PurposeGroupCreateRequest{
		Name:        "test_single_purpose_filter_3",
		Description: "Group without email",
		Purposes: []PurposeGroupPurpose{
			{PurposeName: "test_analytics", IsMandatory: true},
		},
	}

	// Create groups
	resp, body := ts.createGroup(group1Payload)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var group1Resp PurposeGroupResponse
	json.Unmarshal(body, &group1Resp)
	ts.trackGroup(group1Resp.ID)

	resp, body = ts.createGroup(group2Payload)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var group2Resp PurposeGroupResponse
	json.Unmarshal(body, &group2Resp)
	ts.trackGroup(group2Resp.ID)

	resp, body = ts.createGroup(group3Payload)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var group3Resp PurposeGroupResponse
	json.Unmarshal(body, &group3Resp)
	ts.trackGroup(group3Resp.ID)

	// Filter by "test_email" - should return group1 and group2 only
	resp, body = ts.listGroups("", nil, []string{"test_email"}, 100, 0)
	require.Equal(t, http.StatusOK, resp.StatusCode, "Failed to list groups: %s", body)

	var listResp PurposeGroupListResponse
	err := json.Unmarshal(body, &listResp)
	require.NoError(t, err)

	// Should have at least 2 groups (group1 and group2)
	require.GreaterOrEqual(t, len(listResp.Data), 2, "Expected at least 2 groups with test_email")

	// Verify our groups are in the list and group3 is not
	foundGroup1 := false
	foundGroup2 := false
	foundGroup3 := false
	for _, group := range listResp.Data {
		if group.ID == group1Resp.ID {
			foundGroup1 = true
		}
		if group.ID == group2Resp.ID {
			foundGroup2 = true
		}
		if group.ID == group3Resp.ID {
			foundGroup3 = true
		}
	}
	require.True(t, foundGroup1, "Group 1 should be in results")
	require.True(t, foundGroup2, "Group 2 should be in results")
	require.False(t, foundGroup3, "Group 3 should NOT be in results")
}

// TestListPurposeGroups_FilterByMultiplePurposeNames_ANDLogic_ReturnsOnlyGroupsWithAllPurposes tests AND logic
func (ts *PurposeGroupAPITestSuite) TestListPurposeGroups_FilterByMultiplePurposeNames_ANDLogic_ReturnsOnlyGroupsWithAllPurposes() {
	t := ts.T()

	// Create groups with different purpose combinations
	group1Payload := PurposeGroupCreateRequest{
		Name:        "test_and_logic_both",
		Description: "Group with BOTH email AND phone",
		Purposes: []PurposeGroupPurpose{
			{PurposeName: "test_email", IsMandatory: true},
			{PurposeName: "test_phone", IsMandatory: true},
			{PurposeName: "test_address", IsMandatory: false},
		},
	}

	group2Payload := PurposeGroupCreateRequest{
		Name:        "test_and_logic_email_only",
		Description: "Group with ONLY email",
		Purposes: []PurposeGroupPurpose{
			{PurposeName: "test_email", IsMandatory: true},
			{PurposeName: "test_marketing", IsMandatory: false},
		},
	}

	group3Payload := PurposeGroupCreateRequest{
		Name:        "test_and_logic_phone_only",
		Description: "Group with ONLY phone",
		Purposes: []PurposeGroupPurpose{
			{PurposeName: "test_phone", IsMandatory: true},
		},
	}

	group4Payload := PurposeGroupCreateRequest{
		Name:        "test_and_logic_neither",
		Description: "Group with NEITHER email NOR phone",
		Purposes: []PurposeGroupPurpose{
			{PurposeName: "test_analytics", IsMandatory: true},
		},
	}

	// Create groups
	resp, body := ts.createGroup(group1Payload)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var group1Resp PurposeGroupResponse
	json.Unmarshal(body, &group1Resp)
	ts.trackGroup(group1Resp.ID)

	resp, body = ts.createGroup(group2Payload)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var group2Resp PurposeGroupResponse
	json.Unmarshal(body, &group2Resp)
	ts.trackGroup(group2Resp.ID)

	resp, body = ts.createGroup(group3Payload)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var group3Resp PurposeGroupResponse
	json.Unmarshal(body, &group3Resp)
	ts.trackGroup(group3Resp.ID)

	resp, body = ts.createGroup(group4Payload)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var group4Resp PurposeGroupResponse
	json.Unmarshal(body, &group4Resp)
	ts.trackGroup(group4Resp.ID)

	// Filter by BOTH "test_email" AND "test_phone" - should return ONLY group1
	resp, body = ts.listGroups("", nil, []string{"test_email", "test_phone"}, 100, 0)
	require.Equal(t, http.StatusOK, resp.StatusCode, "Failed to list groups: %s", body)

	var listResp PurposeGroupListResponse
	err := json.Unmarshal(body, &listResp)
	require.NoError(t, err)

	// Should have at least 1 group (group1)
	require.GreaterOrEqual(t, len(listResp.Data), 1, "Expected at least 1 group with both purposes")

	// Verify only group1 is in the list (has both purposes)
	foundGroup1 := false
	foundGroup2 := false
	foundGroup3 := false
	foundGroup4 := false
	for _, group := range listResp.Data {
		if group.ID == group1Resp.ID {
			foundGroup1 = true
			// Verify it has both purposes
			hasEmail := false
			hasPhone := false
			for _, p := range group.Purposes {
				if p.PurposeName == "test_email" {
					hasEmail = true
				}
				if p.PurposeName == "test_phone" {
					hasPhone = true
				}
			}
			require.True(t, hasEmail, "Group should have test_email")
			require.True(t, hasPhone, "Group should have test_phone")
		}
		if group.ID == group2Resp.ID {
			foundGroup2 = true
		}
		if group.ID == group3Resp.ID {
			foundGroup3 = true
		}
		if group.ID == group4Resp.ID {
			foundGroup4 = true
		}
	}
	require.True(t, foundGroup1, "Group 1 (has both) should be in results")
	require.False(t, foundGroup2, "Group 2 (only email) should NOT be in results")
	require.False(t, foundGroup3, "Group 3 (only phone) should NOT be in results")
	require.False(t, foundGroup4, "Group 4 (neither) should NOT be in results")
}

// TestListPurposeGroups_CombineAllFilters_ReturnsCorrectlyFilteredGroups tests all 3 filters together
func (ts *PurposeGroupAPITestSuite) TestListPurposeGroups_CombineAllFilters_ReturnsCorrectlyFilteredGroups() {
	t := ts.T()

	// Create a group that matches all filters
	matchingGroupPayload := PurposeGroupCreateRequest{
		Name:        "test_all_filters_match",
		Description: "Group that matches all filters",
		Purposes: []PurposeGroupPurpose{
			{PurposeName: "test_email", IsMandatory: true},
			{PurposeName: "test_phone", IsMandatory: true},
		},
	}

	// Create a group with same purposes but different name
	differentNamePayload := PurposeGroupCreateRequest{
		Name:        "test_all_filters_different_name",
		Description: "Different name",
		Purposes: []PurposeGroupPurpose{
			{PurposeName: "test_email", IsMandatory: true},
			{PurposeName: "test_phone", IsMandatory: true},
		},
	}

	// Create matching group
	resp, body := ts.createGroup(matchingGroupPayload)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var matchingResp PurposeGroupResponse
	json.Unmarshal(body, &matchingResp)
	ts.trackGroup(matchingResp.ID)

	// Create non-matching group
	resp, body = ts.createGroup(differentNamePayload)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var differentNameResp PurposeGroupResponse
	json.Unmarshal(body, &differentNameResp)
	ts.trackGroup(differentNameResp.ID)

	// Filter by name + clientID + purposeNames (all 3 filters)
	resp, body = ts.listGroups(
		"test_all_filters_match",
		[]string{testClientID},
		[]string{"test_email", "test_phone"},
		100, 0)
	require.Equal(t, http.StatusOK, resp.StatusCode, "Failed to list groups: %s", body)

	var listResp PurposeGroupListResponse
	err := json.Unmarshal(body, &listResp)
	require.NoError(t, err)

	// Should return exactly 1 group - the matching one
	require.Equal(t, 1, listResp.Metadata.Total, "Expected exactly 1 matching group")
	require.Len(t, listResp.Data, 1, "Expected exactly 1 group in data")
	require.Equal(t, matchingResp.ID, listResp.Data[0].ID)
	require.Equal(t, "test_all_filters_match", listResp.Data[0].Name)
	require.Equal(t, testClientID, listResp.Data[0].ClientID)
}

// TestListPurposeGroups_Pagination_ReturnsCorrectSubset tests pagination parameters
func (ts *PurposeGroupAPITestSuite) TestListPurposeGroups_Pagination_ReturnsCorrectSubset() {
	t := ts.T()

	// Create multiple groups for pagination testing
	for i := 1; i <= 5; i++ {
		payload := PurposeGroupCreateRequest{
			Name:        fmt.Sprintf("test_pagination_group_%d", i),
			Description: fmt.Sprintf("Pagination test group %d", i),
			Purposes: []PurposeGroupPurpose{
				{PurposeName: "test_email", IsMandatory: true},
			},
		}

		resp, body := ts.createGroup(payload)
		require.Equal(t, http.StatusCreated, resp.StatusCode)
		var groupResp PurposeGroupResponse
		json.Unmarshal(body, &groupResp)
		ts.trackGroup(groupResp.ID)
	}

	// Test first page (limit 2, offset 0)
	resp, body := ts.listGroups("", nil, nil, 2, 0)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var listResp PurposeGroupListResponse
	err := json.Unmarshal(body, &listResp)
	require.NoError(t, err)

	require.Len(t, listResp.Data, 2, "First page should have 2 groups")
	require.Equal(t, 0, listResp.Metadata.Offset)
	require.Equal(t, 2, listResp.Metadata.Limit)

	// Test second page (limit 2, offset 2)
	resp, body = ts.listGroups("", nil, nil, 2, 2)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	err = json.Unmarshal(body, &listResp)
	require.NoError(t, err)

	require.LessOrEqual(t, len(listResp.Data), 2, "Second page should have at most 2 groups")
	require.Equal(t, 2, listResp.Metadata.Offset)
	require.Equal(t, 2, listResp.Metadata.Limit)
}

// TestListPurposeGroups_EmptyResult_ReturnsEmptyList tests when no groups match filters
func (ts *PurposeGroupAPITestSuite) TestListPurposeGroups_EmptyResult_ReturnsEmptyList() {
	t := ts.T()

	// Filter by non-existent name
	resp, body := ts.listGroups("non_existent_group_name_xyz", nil, nil, 100, 0)
	require.Equal(t, http.StatusOK, resp.StatusCode, "Should return OK even with no results: %s", body)

	var listResp PurposeGroupListResponse
	err := json.Unmarshal(body, &listResp)
	require.NoError(t, err)

	require.Equal(t, 0, listResp.Metadata.Total, "Total should be 0")
	require.Len(t, listResp.Data, 0, "Data should be empty")
}
