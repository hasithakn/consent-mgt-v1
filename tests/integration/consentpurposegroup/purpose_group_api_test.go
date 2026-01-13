package consentpurposegroup

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/wso2/consent-management-api/tests/integration/testutils"
)

var (
	testServerURL = testutils.GetTestServerURL()
)

const (
	testOrgID    = "test-org-purposegroup"
	testClientID = "test-client-purposegroup"
)

type PurposeGroupAPITestSuite struct {
	suite.Suite
	createdGroupIDs   []string
	createdPurposeIDs []string
}

// SetupSuite runs once before all tests
func (ts *PurposeGroupAPITestSuite) SetupSuite() {
	ts.createdGroupIDs = make([]string, 0)
	ts.createdPurposeIDs = make([]string, 0)
	ts.T().Logf("=== ConsentPurposeGroup Test Suite Starting ===")
	ts.setupTestPurposes()
}

// TearDownSuite runs once after all tests to cleanup
func (ts *PurposeGroupAPITestSuite) TearDownSuite() {
	if len(ts.createdGroupIDs) > 0 {
		ts.T().Logf("=== Cleaning up %d created purpose groups ===", len(ts.createdGroupIDs))
		successCount := 0
		failCount := 0
		for _, id := range ts.createdGroupIDs {
			if ts.deleteGroupWithCheck(id) {
				successCount++
			} else {
				failCount++
			}
		}
		ts.T().Logf("=== Group cleanup complete: %d deleted, %d failed ===", successCount, failCount)
	}

	if len(ts.createdPurposeIDs) > 0 {
		ts.T().Logf("=== Cleaning up %d created purposes ===", len(ts.createdPurposeIDs))
		successCount := 0
		failCount := 0
		for _, id := range ts.createdPurposeIDs {
			if ts.deletePurposeWithCheck(id) {
				successCount++
			} else {
				failCount++
			}
		}
		ts.T().Logf("=== Purpose cleanup complete: %d deleted, %d failed ===", successCount, failCount)
	}

	ts.T().Logf("=== ConsentPurposeGroup Test Suite Complete ===")
}

func TestPurposeGroupAPITestSuite(t *testing.T) {
	suite.Run(t, new(PurposeGroupAPITestSuite))
}

// setupTestPurposes creates purposes needed for tests
func (ts *PurposeGroupAPITestSuite) setupTestPurposes() {
	purposeNames := []string{
		"test_email",
		"test_phone",
		"test_address",
		"test_marketing",
		"test_analytics",
	}

	for _, name := range purposeNames {
		payload := []map[string]interface{}{
			{
				"name":        name,
				"description": fmt.Sprintf("Test purpose for %s", name),
				"type":        "string",
				"attributes": map[string]string{
					"resourcePath": fmt.Sprintf("/user/%s", name),
				},
			},
		}

		reqBody, _ := json.Marshal(payload)
		httpReq, _ := http.NewRequest("POST", testServerURL+"/api/v1/consent-purposes",
			bytes.NewBuffer(reqBody))
		httpReq.Header.Set("org-id", testOrgID)
		httpReq.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(httpReq)
		if err != nil {
			ts.T().Logf("ERROR creating test purpose %s: %v", name, err)
			continue
		}

		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode == http.StatusCreated {
			var createResp struct {
				Data []struct {
					ID string `json:"id"`
				} `json:"data"`
			}
			if json.Unmarshal(bodyBytes, &createResp) == nil && len(createResp.Data) > 0 {
				ts.createdPurposeIDs = append(ts.createdPurposeIDs, createResp.Data[0].ID)
				ts.T().Logf("✓ Created test purpose: %s (ID: %s)", name, createResp.Data[0].ID)
			} else {
				ts.T().Logf("ERROR: Failed to parse purpose response for %s", name)
			}
		} else {
			ts.T().Logf("ERROR: Failed to create purpose %s - Status: %d, Body: %s", name, resp.StatusCode, string(bodyBytes))
		}
	}

	if len(ts.createdPurposeIDs) == 0 {
		ts.T().Fatal("Failed to create any test purposes - tests cannot continue")
	}
	ts.T().Logf("Successfully created %d/%d test purposes", len(ts.createdPurposeIDs), len(purposeNames))
}

// createGroup creates a purpose group and returns the response
func (ts *PurposeGroupAPITestSuite) createGroup(payload interface{}) (*http.Response, []byte) {
	var reqBody []byte
	var err error

	if str, ok := payload.(string); ok {
		reqBody = []byte(str)
	} else {
		reqBody, err = json.Marshal(payload)
		ts.Require().NoError(err)
	}

	httpReq, _ := http.NewRequest("POST", testServerURL+"/api/v1/consent-purpose-groups",
		bytes.NewBuffer(reqBody))
	httpReq.Header.Set("org-id", testOrgID)
	httpReq.Header.Set("TPP-client-id", testClientID)
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	ts.Require().NoError(err)

	bodyBytes, err := io.ReadAll(resp.Body)
	ts.Require().NoError(err)
	resp.Body.Close()

	return resp, bodyBytes
}

// getGroup retrieves a purpose group by ID
func (ts *PurposeGroupAPITestSuite) getGroup(groupID string) (*http.Response, []byte) {
	httpReq, _ := http.NewRequest("GET",
		fmt.Sprintf("%s/api/v1/consent-purpose-groups/%s", testServerURL, groupID),
		nil)
	httpReq.Header.Set("org-id", testOrgID)

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	ts.Require().NoError(err)

	bodyBytes, err := io.ReadAll(resp.Body)
	ts.Require().NoError(err)
	resp.Body.Close()

	return resp, bodyBytes
}

// listGroups lists purpose groups with optional filters
func (ts *PurposeGroupAPITestSuite) listGroups(name string, clientIDs []string, purposeNames []string, limit, offset int) (*http.Response, []byte) {
	url := fmt.Sprintf("%s/api/v1/consent-purpose-groups?limit=%d&offset=%d", testServerURL, limit, offset)

	if name != "" {
		url += fmt.Sprintf("&name=%s", name)
	}

	if len(clientIDs) > 0 {
		url += fmt.Sprintf("&clientIds=%s", joinStrings(clientIDs, ","))
	}

	if len(purposeNames) > 0 {
		url += fmt.Sprintf("&purposeNames=%s", joinStrings(purposeNames, ","))
	}

	httpReq, _ := http.NewRequest("GET", url, nil)
	httpReq.Header.Set("org-id", testOrgID)

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	ts.Require().NoError(err)

	bodyBytes, err := io.ReadAll(resp.Body)
	ts.Require().NoError(err)
	resp.Body.Close()

	return resp, bodyBytes
}

// updateGroup updates a purpose group
func (ts *PurposeGroupAPITestSuite) updateGroup(groupID string, payload interface{}) (*http.Response, []byte) {
	var reqBody []byte
	var err error

	if str, ok := payload.(string); ok {
		reqBody = []byte(str)
	} else {
		reqBody, err = json.Marshal(payload)
		ts.Require().NoError(err)
	}

	httpReq, _ := http.NewRequest("PUT",
		fmt.Sprintf("%s/api/v1/consent-purpose-groups/%s", testServerURL, groupID),
		bytes.NewBuffer(reqBody))
	httpReq.Header.Set("org-id", testOrgID)
	httpReq.Header.Set("TPP-client-id", testClientID)
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	ts.Require().NoError(err)

	bodyBytes, err := io.ReadAll(resp.Body)
	ts.Require().NoError(err)
	resp.Body.Close()

	return resp, bodyBytes
}

// deleteGroup deletes a purpose group
func (ts *PurposeGroupAPITestSuite) deleteGroup(groupID string) (*http.Response, []byte) {
	httpReq, _ := http.NewRequest("DELETE",
		fmt.Sprintf("%s/api/v1/consent-purpose-groups/%s", testServerURL, groupID),
		nil)
	httpReq.Header.Set("org-id", testOrgID)

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	ts.Require().NoError(err)

	bodyBytes, err := io.ReadAll(resp.Body)
	ts.Require().NoError(err)
	resp.Body.Close()

	return resp, bodyBytes
}

// trackGroup tracks a created group for cleanup
func (ts *PurposeGroupAPITestSuite) trackGroup(groupID string) {
	ts.createdGroupIDs = append(ts.createdGroupIDs, groupID)
}

// deleteGroupWithCheck attempts to delete a group and returns success status
func (ts *PurposeGroupAPITestSuite) deleteGroupWithCheck(groupID string) bool {
	resp, body := ts.deleteGroup(groupID)
	if resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusNotFound {
		ts.T().Logf("Deleted group: %s", groupID)
		return true
	}
	ts.T().Logf("Failed to delete group %s: %d - %s", groupID, resp.StatusCode, body)
	return false
}

// deletePurposeWithCheck attempts to delete a purpose and returns success status
func (ts *PurposeGroupAPITestSuite) deletePurposeWithCheck(purposeID string) bool {
	httpReq, _ := http.NewRequest("DELETE",
		fmt.Sprintf("%s/api/v1/consent-purposes/%s", testServerURL, purposeID),
		nil)
	httpReq.Header.Set("org-id", testOrgID)

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		ts.T().Logf("Failed to delete purpose %s: %v", purposeID, err)
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusNotFound {
		ts.T().Logf("Deleted purpose: %s", purposeID)
		return true
	}
	ts.T().Logf("Failed to delete purpose %s: %d", purposeID, resp.StatusCode)
	return false
}

// joinStrings joins string slice with delimiter
func joinStrings(strs []string, delim string) string {
	result := ""
	for i, s := range strs {
		if i > 0 {
			result += delim
		}
		result += s
	}
	return result
}
