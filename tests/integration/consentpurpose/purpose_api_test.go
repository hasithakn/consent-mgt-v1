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
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/wso2/consent-management-api/tests/integration/testutils"
)

const (
	testServerURL = testutils.TestServerURL
	baseURL       = testutils.TestServerURL
	testOrgID     = "test-org-purpose"
	testClientID  = "test-client-purpose"
)

type PurposeAPITestSuite struct {
	suite.Suite
	createdPurposeIDs []string // Track created purposes for cleanup
}

// SetupSuite runs once before all tests
func (ts *PurposeAPITestSuite) SetupSuite() {
	ts.createdPurposeIDs = make([]string, 0)
	ts.T().Logf("=== ConsentPurpose Test Suite Starting ===")
	// Note: Pre-cleanup has been removed as it runs before server is ready
	// If you need to clean leftover data, run tests twice or manually delete test_* purposes
}

// TearDownSuite runs once after all tests to cleanup
func (ts *PurposeAPITestSuite) TearDownSuite() {
	if len(ts.createdPurposeIDs) == 0 {
		ts.T().Logf("=== No purposes to clean up ===")
		return
	}

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

	ts.T().Logf("=== Cleanup complete: %d deleted, %d failed ===", successCount, failCount)
	ts.T().Logf("=== ConsentPurpose Test Suite Complete ===")
}

// TearDownTest runs after each test to ensure cleanup
func (ts *PurposeAPITestSuite) TearDownTest() {
	// Additional per-test cleanup if needed
}

func TestPurposeAPITestSuite(t *testing.T) {
	suite.Run(t, new(PurposeAPITestSuite))
}

// Helper functions

// createPurpose creates purpose(s) and returns the response and body for flexible assertions
func (ts *PurposeAPITestSuite) createPurpose(payload interface{}) (*http.Response, []byte) {
	var reqBody []byte
	var err error

	// Handle both []ConsentPurposeCreateRequest and string (for malformed JSON tests)
	if str, ok := payload.(string); ok {
		reqBody = []byte(str)
	} else {
		reqBody, err = json.Marshal(payload)
		ts.Require().NoError(err)
	}

	httpReq, _ := http.NewRequest("POST", testServerURL+"/api/v1/consent-purposes",
		bytes.NewBuffer(reqBody))
	httpReq.Header.Set(testutils.HeaderContentType, "application/json")
	httpReq.Header.Set(testutils.HeaderOrgID, testOrgID)
	httpReq.Header.Set(testutils.HeaderClientID, testutils.TestClientID)

	client := testutils.GetHTTPClient()
	resp, err := client.Do(httpReq)
	ts.Require().NoError(err)

	body, err := io.ReadAll(resp.Body)
	ts.Require().NoError(err)

	return resp, body
}

// getPurpose retrieves a purpose by ID and returns response and body
func (ts *PurposeAPITestSuite) getPurpose(purposeID string) (*http.Response, []byte) {
	url := fmt.Sprintf("%s/api/v1/consent-purposes/%s", testServerURL, purposeID)
	httpReq, _ := http.NewRequest("GET", url, nil)
	httpReq.Header.Set(testutils.HeaderOrgID, testOrgID)
	httpReq.Header.Set(testutils.HeaderClientID, testutils.TestClientID)

	client := testutils.GetHTTPClient()
	resp, err := client.Do(httpReq)
	ts.Require().NoError(err)

	body, err := io.ReadAll(resp.Body)
	ts.Require().NoError(err)

	return resp, body
}

// updatePurpose updates a purpose by ID and returns response and body
func (ts *PurposeAPITestSuite) updatePurpose(purposeID string, payload interface{}) (*http.Response, []byte) {
	var reqBody []byte
	var err error

	if str, ok := payload.(string); ok {
		reqBody = []byte(str)
	} else {
		reqBody, err = json.Marshal(payload)
		ts.Require().NoError(err)
	}

	url := fmt.Sprintf("%s/api/v1/consent-purposes/%s", testServerURL, purposeID)
	httpReq, _ := http.NewRequest("PUT", url, bytes.NewBuffer(reqBody))
	httpReq.Header.Set(testutils.HeaderContentType, "application/json")
	httpReq.Header.Set(testutils.HeaderOrgID, testOrgID)
	httpReq.Header.Set(testutils.HeaderClientID, testutils.TestClientID)

	client := testutils.GetHTTPClient()
	resp, err := client.Do(httpReq)
	ts.Require().NoError(err)

	body, err := io.ReadAll(resp.Body)
	ts.Require().NoError(err)

	return resp, body
}

// deletePurpose deletes a purpose by ID
func (ts *PurposeAPITestSuite) deletePurpose(purposeID string) {
	httpReq, _ := http.NewRequest("DELETE",
		fmt.Sprintf("%s/api/v1/consent-purposes/%s", testServerURL, purposeID),
		nil)
	httpReq.Header.Set(testutils.HeaderOrgID, testOrgID)
	httpReq.Header.Set(testutils.HeaderClientID, testutils.TestClientID)

	client := testutils.GetHTTPClient()
	resp, err := client.Do(httpReq)
	if err != nil {
		ts.T().Logf("Warning: failed to delete purpose %s: %v", purposeID, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		ts.T().Logf("Warning: failed to delete purpose %s: %d - %s", purposeID, resp.StatusCode, string(body))
	}
}

// trackPurpose registers a purpose ID for cleanup in TearDownSuite
func (ts *PurposeAPITestSuite) trackPurpose(purposeID string) {
	ts.createdPurposeIDs = append(ts.createdPurposeIDs, purposeID)
}

// deletePurposeWithCheck deletes a purpose and returns success status
// Returns true only for successful deletion (204/200), false for 404 or other errors
func (ts *PurposeAPITestSuite) deletePurposeWithCheck(purposeID string) bool {
	httpReq, _ := http.NewRequest("DELETE",
		fmt.Sprintf("%s/api/v1/consent-purposes/%s", testServerURL, purposeID),
		nil)
	httpReq.Header.Set(testutils.HeaderOrgID, testOrgID)
	httpReq.Header.Set(testutils.HeaderClientID, testutils.TestClientID)

	client := testutils.GetHTTPClient()
	resp, err := client.Do(httpReq)
	if err != nil {
		ts.T().Logf("Warning: failed to delete purpose %s: %v", purposeID, err)
		return false
	}
	defer resp.Body.Close()

	// Return true only for successful deletion (204 or 200)
	// Return false for 404 (not found) or any other status
	if resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusOK {
		return true
	}

	return false
}
