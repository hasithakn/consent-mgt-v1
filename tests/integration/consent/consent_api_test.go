/*
 * Copyright (c) 2024, WSO2 Inc. (http://www.wso2.org) All Rights Reserved.
 *
 * WSO2 Inc. licenses this file to you under the Apache License,
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

package consent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/wso2/consent-management-api/tests/integration/testutils"
)

const (
	testServerURL = testutils.TestServerURL
	baseURL       = testutils.TestServerURL
	testOrgID     = "test-org-consent"
	testClientID  = "test-client-consent"
)

type ConsentAPITestSuite struct {
	suite.Suite
	createdConsentIDs []string // Track created consents for cleanup
	testPurposeIDs    []string // Track test purposes for cleanup
}

// SetupSuite runs once before all tests
func (ts *ConsentAPITestSuite) SetupSuite() {
	ts.createdConsentIDs = make([]string, 0)
	ts.testPurposeIDs = make([]string, 0)
	ts.T().Logf("=== Consent Test Suite Starting ===")

	// Create test purposes needed for consent tests
	ts.createTestPurposes()
}

// TearDownSuite runs once after all tests
func (ts *ConsentAPITestSuite) TearDownSuite() {
	ts.T().Logf("=== Consent Test Suite Complete ===")
	deleted := 0
	failed := 0

	ts.T().Logf("Cleaning up %d consents...", len(ts.createdConsentIDs))
	for _, consentID := range ts.createdConsentIDs {
		if ts.deleteConsent(consentID) {
			deleted++
		} else {
			failed++
		}
	}
	ts.T().Logf("=== Cleanup complete: %d deleted, %d failed ===", deleted, failed)

	// Cleanup test purposes
	ts.cleanupTestPurposes()
}

// createConsent is a helper to create a consent and returns response and body
func (ts *ConsentAPITestSuite) createConsent(payload interface{}) (*http.Response, []byte) {
	var reqBody []byte
	var err error

	if str, ok := payload.(string); ok {
		reqBody = []byte(str)
	} else {
		reqBody, err = json.Marshal(payload)
		ts.Require().NoError(err)
	}

	httpReq, _ := http.NewRequest("POST", testServerURL+"/api/v1/consents",
		bytes.NewBuffer(reqBody))
	httpReq.Header.Set(testutils.HeaderContentType, "application/json")
	httpReq.Header.Set(testutils.HeaderOrgID, testOrgID)
	httpReq.Header.Set(testutils.HeaderClientID, testClientID)

	client := testutils.GetHTTPClient()
	resp, err := client.Do(httpReq)
	ts.Require().NoError(err)

	body, err := io.ReadAll(resp.Body)
	ts.Require().NoError(err)

	return resp, body
}

// getConsent retrieves a consent by ID and returns response and body
func (ts *ConsentAPITestSuite) getConsent(consentID string) (*http.Response, []byte) {
	url := fmt.Sprintf("%s/api/v1/consents/%s", testServerURL, consentID)

	httpReq, _ := http.NewRequest("GET", url, nil)
	httpReq.Header.Set(testutils.HeaderOrgID, testOrgID)
	httpReq.Header.Set(testutils.HeaderClientID, testClientID)

	client := testutils.GetHTTPClient()
	resp, err := client.Do(httpReq)
	ts.Require().NoError(err)

	body, err := io.ReadAll(resp.Body)
	ts.Require().NoError(err)

	return resp, body
}

// getConsentWithHeaders retrieves a consent with custom headers (for testing header validation)
func (ts *ConsentAPITestSuite) getConsentWithHeaders(consentID, orgID, clientID string) (*http.Response, []byte) {
	url := fmt.Sprintf("%s/api/v1/consents/%s", testServerURL, consentID)

	httpReq, _ := http.NewRequest("GET", url, nil)
	if orgID != "" {
		httpReq.Header.Set(testutils.HeaderOrgID, orgID)
	}
	if clientID != "" {
		httpReq.Header.Set(testutils.HeaderClientID, clientID)
	}

	client := testutils.GetHTTPClient()
	resp, err := client.Do(httpReq)
	ts.Require().NoError(err)

	body, err := io.ReadAll(resp.Body)
	ts.Require().NoError(err)

	return resp, body
}

// listConsents retrieves a list of consents with query parameters
func (ts *ConsentAPITestSuite) listConsents(queryParams map[string]string) (*http.Response, []byte) {
	url := fmt.Sprintf("%s/api/v1/consents", testServerURL)

	// Add query parameters
	if len(queryParams) > 0 {
		query := make([]string, 0, len(queryParams))
		for key, value := range queryParams {
			query = append(query, fmt.Sprintf("%s=%s", key, value))
		}
		url = fmt.Sprintf("%s?%s", url, strings.Join(query, "&"))
	}

	httpReq, _ := http.NewRequest("GET", url, nil)
	httpReq.Header.Set(testutils.HeaderOrgID, testOrgID)
	httpReq.Header.Set(testutils.HeaderClientID, testClientID)

	client := testutils.GetHTTPClient()
	resp, err := client.Do(httpReq)
	ts.Require().NoError(err)

	body, err := io.ReadAll(resp.Body)
	ts.Require().NoError(err)

	return resp, body
}

// listConsentsWithHeaders retrieves a list of consents with custom headers
func (ts *ConsentAPITestSuite) listConsentsWithHeaders(queryParams map[string]string, orgID, clientID string) (*http.Response, []byte) {
	url := fmt.Sprintf("%s/api/v1/consents", testServerURL)

	// Add query parameters
	if len(queryParams) > 0 {
		query := make([]string, 0, len(queryParams))
		for key, value := range queryParams {
			query = append(query, fmt.Sprintf("%s=%s", key, value))
		}
		url = fmt.Sprintf("%s?%s", url, strings.Join(query, "&"))
	}

	httpReq, _ := http.NewRequest("GET", url, nil)
	if orgID != "" {
		httpReq.Header.Set(testutils.HeaderOrgID, orgID)
	}
	if clientID != "" {
		httpReq.Header.Set(testutils.HeaderClientID, clientID)
	}

	client := testutils.GetHTTPClient()
	resp, err := client.Do(httpReq)
	ts.Require().NoError(err)

	body, err := io.ReadAll(resp.Body)
	ts.Require().NoError(err)

	return resp, body
}

// updateConsent updates a consent by ID and returns response and body
func (ts *ConsentAPITestSuite) updateConsent(consentID string, payload interface{}) (*http.Response, []byte) {
	var reqBody []byte
	var err error

	if str, ok := payload.(string); ok {
		reqBody = []byte(str)
	} else {
		reqBody, err = json.Marshal(payload)
		ts.Require().NoError(err)
	}

	url := fmt.Sprintf("%s/api/v1/consents/%s", testServerURL, consentID)
	httpReq, _ := http.NewRequest("PUT", url, bytes.NewBuffer(reqBody))
	httpReq.Header.Set(testutils.HeaderContentType, "application/json")
	httpReq.Header.Set(testutils.HeaderOrgID, testOrgID)
	httpReq.Header.Set(testutils.HeaderClientID, testClientID)

	client := testutils.GetHTTPClient()
	resp, err := client.Do(httpReq)
	ts.Require().NoError(err)

	body, err := io.ReadAll(resp.Body)
	ts.Require().NoError(err)

	return resp, body
}

// updateConsentWithHeaders updates a consent with custom headers (for testing header validation)
func (ts *ConsentAPITestSuite) updateConsentWithHeaders(consentID string, payload interface{}, orgID, clientID string) (*http.Response, []byte) {
	var reqBody []byte
	var err error

	if str, ok := payload.(string); ok {
		reqBody = []byte(str)
	} else {
		reqBody, err = json.Marshal(payload)
		ts.Require().NoError(err)
	}

	url := fmt.Sprintf("%s/api/v1/consents/%s", testServerURL, consentID)
	httpReq, _ := http.NewRequest("PUT", url, bytes.NewBuffer(reqBody))
	httpReq.Header.Set(testutils.HeaderContentType, "application/json")
	if orgID != "" {
		httpReq.Header.Set(testutils.HeaderOrgID, orgID)
	}
	if clientID != "" {
		httpReq.Header.Set(testutils.HeaderClientID, clientID)
	}

	client := testutils.GetHTTPClient()
	resp, err := client.Do(httpReq)
	ts.Require().NoError(err)

	body, err := io.ReadAll(resp.Body)
	ts.Require().NoError(err)

	return resp, body
}

// revokeConsent revokes a consent and returns response and body
func (ts *ConsentAPITestSuite) revokeConsent(consentID string, reason string) (*http.Response, []byte) {
	payload := ConsentRevokeRequest{
		Reason:   reason,
		ActionBy: "test-user",
	}
	reqBody, err := json.Marshal(payload)
	ts.Require().NoError(err)

	url := fmt.Sprintf("%s/api/v1/consents/%s/revoke", testServerURL, consentID)
	httpReq, _ := http.NewRequest("PUT", url, bytes.NewBuffer(reqBody))
	httpReq.Header.Set(testutils.HeaderContentType, "application/json")
	httpReq.Header.Set(testutils.HeaderOrgID, testOrgID)
	httpReq.Header.Set(testutils.HeaderClientID, testClientID)

	client := testutils.GetHTTPClient()
	resp, err := client.Do(httpReq)
	ts.Require().NoError(err)

	body, err := io.ReadAll(resp.Body)
	ts.Require().NoError(err)

	return resp, body
}

// revokeConsentWithHeaders revokes a consent with custom headers (for testing header validation)
func (ts *ConsentAPITestSuite) revokeConsentWithHeaders(consentID string, reason string, orgID, clientID string) (*http.Response, []byte) {
	payload := ConsentRevokeRequest{
		Reason:   reason,
		ActionBy: "test-user",
	}
	reqBody, err := json.Marshal(payload)
	ts.Require().NoError(err)

	url := fmt.Sprintf("%s/api/v1/consents/%s/revoke", testServerURL, consentID)
	httpReq, _ := http.NewRequest("PUT", url, bytes.NewBuffer(reqBody))
	httpReq.Header.Set(testutils.HeaderContentType, "application/json")
	if orgID != "" {
		httpReq.Header.Set(testutils.HeaderOrgID, orgID)
	}
	if clientID != "" {
		httpReq.Header.Set(testutils.HeaderClientID, clientID)
	}

	client := testutils.GetHTTPClient()
	resp, err := client.Do(httpReq)
	ts.Require().NoError(err)

	body, err := io.ReadAll(resp.Body)
	ts.Require().NoError(err)

	return resp, body
}

// validateConsent validates a consent and returns response and body
func (ts *ConsentAPITestSuite) validateConsent(payload interface{}) (*http.Response, []byte) {
	var reqBody []byte
	var err error

	if str, ok := payload.(string); ok {
		reqBody = []byte(str)
	} else {
		reqBody, err = json.Marshal(payload)
		ts.Require().NoError(err)
	}

	url := fmt.Sprintf("%s/api/v1/consents/validate", testServerURL)
	httpReq, _ := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	httpReq.Header.Set(testutils.HeaderContentType, "application/json")
	httpReq.Header.Set(testutils.HeaderOrgID, testOrgID)
	httpReq.Header.Set(testutils.HeaderClientID, testClientID)

	client := testutils.GetHTTPClient()
	resp, err := client.Do(httpReq)
	ts.Require().NoError(err)

	body, err := io.ReadAll(resp.Body)
	ts.Require().NoError(err)

	return resp, body
}

// validateConsentWithHeaders validates a consent with custom headers (for testing header validation)
func (ts *ConsentAPITestSuite) validateConsentWithHeaders(payload interface{}, orgID, clientID string) (*http.Response, []byte) {
	var reqBody []byte
	var err error

	if str, ok := payload.(string); ok {
		reqBody = []byte(str)
	} else {
		reqBody, err = json.Marshal(payload)
		ts.Require().NoError(err)
	}

	url := fmt.Sprintf("%s/api/v1/consents/validate", testServerURL)
	httpReq, _ := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	httpReq.Header.Set(testutils.HeaderContentType, "application/json")
	if orgID != "" {
		httpReq.Header.Set(testutils.HeaderOrgID, orgID)
	}
	if clientID != "" {
		httpReq.Header.Set(testutils.HeaderClientID, clientID)
	}

	client := testutils.GetHTTPClient()
	resp, err := client.Do(httpReq)
	ts.Require().NoError(err)

	body, err := io.ReadAll(resp.Body)
	ts.Require().NoError(err)

	return resp, body
}

// deleteConsent deletes a consent by ID (for cleanup)
func (ts *ConsentAPITestSuite) deleteConsent(consentID string) bool {
	// Note: DELETE /consents/{id} endpoint may not exist
	// This is a placeholder - adjust based on actual API

	url := fmt.Sprintf("%s/api/v1/consents/%s", testServerURL, consentID)
	httpReq, _ := http.NewRequest("DELETE", url, nil)
	httpReq.Header.Set(testutils.HeaderOrgID, testOrgID)
	httpReq.Header.Set(testutils.HeaderClientID, testClientID)

	client := testutils.GetHTTPClient()
	resp, err := client.Do(httpReq)
	if err != nil {
		ts.T().Logf("Warning: failed to delete consent %s: %v", consentID, err)
		return false
	}
	defer resp.Body.Close()

	// Accept 204, 200, or 404 (already deleted) as success
	if resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusOK {
		return true
	}

	body, _ := io.ReadAll(resp.Body)
	ts.T().Logf("Warning: failed to delete consent %s: %d - %s", consentID, resp.StatusCode, string(body))
	return false
}

// trackConsent registers a consent ID for cleanup in TearDownSuite
func (ts *ConsentAPITestSuite) trackConsent(consentID string) {
	ts.createdConsentIDs = append(ts.createdConsentIDs, consentID)
}

// createTestPurposes creates consent purposes needed for testing
func (ts *ConsentAPITestSuite) createTestPurposes() {
	ts.T().Logf("Setting up test purposes...")

	purposes := []map[string]interface{}{
		{
			"name":        "marketing-purpose",
			"description": "Marketing consent purpose",
			"type":        "string",
			"attributes":  map[string]string{},
		},
		{
			"name":        "analytics-purpose",
			"description": "Analytics consent purpose",
			"type":        "string",
			"attributes":  map[string]string{},
		},
		{
			"name":        "terms-purpose",
			"description": "Terms and conditions purpose",
			"type":        "string",
			"attributes":  map[string]string{},
		},
	}

	reqBody, err := json.Marshal(purposes)
	if err != nil {
		ts.T().Logf("Warning: failed to marshal purposes: %v", err)
		return
	}

	httpReq, _ := http.NewRequest("POST", testServerURL+"/api/v1/consent-purposes",
		bytes.NewBuffer(reqBody))
	httpReq.Header.Set(testutils.HeaderContentType, "application/json")
	httpReq.Header.Set(testutils.HeaderOrgID, testOrgID)
	httpReq.Header.Set(testutils.HeaderClientID, testClientID)

	client := testutils.GetHTTPClient()
	resp, err := client.Do(httpReq)
	if err != nil {
		ts.T().Logf("Warning: failed to create purposes: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusCreated {
		var result map[string]interface{}
		body, _ := io.ReadAll(resp.Body)
		if json.Unmarshal(body, &result) == nil {
			if data, ok := result["data"].([]interface{}); ok {
				for _, item := range data {
					if purposeMap, ok := item.(map[string]interface{}); ok {
						if id, ok := purposeMap["purposeId"].(string); ok {
							ts.testPurposeIDs = append(ts.testPurposeIDs, id)
						}
					}
				}
			}
		}
		ts.T().Logf("Created %d test purposes", len(ts.testPurposeIDs))
	} else {
		body, _ := io.ReadAll(resp.Body)
		ts.T().Logf("Warning: failed to create purposes: %d - %s", resp.StatusCode, string(body))
	}
}

// cleanupTestPurposes removes test purposes created in SetupSuite
func (ts *ConsentAPITestSuite) cleanupTestPurposes() {
	ts.T().Logf("Cleaning up test purposes...")
	for _, purposeID := range ts.testPurposeIDs {
		url := fmt.Sprintf("%s/api/v1/consent-purposes/%s", testServerURL, purposeID)
		httpReq, _ := http.NewRequest("DELETE", url, nil)
		httpReq.Header.Set(testutils.HeaderOrgID, testOrgID)
		httpReq.Header.Set(testutils.HeaderClientID, testClientID)

		client := testutils.GetHTTPClient()
		resp, err := client.Do(httpReq)
		if err != nil {
			ts.T().Logf("Warning: failed to delete purpose %s: %v", purposeID, err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
			body, _ := io.ReadAll(resp.Body)
			ts.T().Logf("Warning: failed to delete purpose %s: %d - %s", purposeID, resp.StatusCode, string(body))
		}
	}
}

// TestConsentAPITestSuite runs the test suite
func TestConsentAPITestSuite(t *testing.T) {
	suite.Run(t, new(ConsentAPITestSuite))
}
