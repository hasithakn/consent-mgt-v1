package integration

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wso2/consent-management-api/internal/config"
	"github.com/wso2/consent-management-api/internal/dao"
	"github.com/wso2/consent-management-api/internal/database"
	"github.com/wso2/consent-management-api/internal/models"
	"github.com/wso2/consent-management-api/internal/service"
	"github.com/wso2/consent-management-api/pkg/utils"
)

// Test constants
const (
	testOrgID    = "TEST_ORG"
	testClientID = "test-client-001"
)

// TestEnvironment holds the test setup
type TestEnvironment struct {
	DB                  *database.DB
	ConsentDAO          *dao.ConsentDAO
	AuditDAO            *dao.StatusAuditDAO
	AttributeDAO        *dao.ConsentAttributeDAO
	AuthResourceDAO     *dao.AuthResourceDAO
	ConsentService      *service.ConsentService
	AuthResourceService *service.AuthResourceService
	Logger              *logrus.Logger
	Config              *config.Config
}

// setupTestEnvironment initializes the test environment with real database
func setupTestEnvironment(t *testing.T) *TestEnvironment {
	// Load configuration
	cfg, err := config.Load("../../configs/config.yaml")
	require.NoError(t, err, "Failed to load config")

	// Initialize logger
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// Initialize database
	db, err := database.Initialize(&cfg.Database, logger)
	require.NoError(t, err, "Failed to initialize database")

	// Verify database connection
	err = db.HealthCheck(context.Background())
	require.NoError(t, err, "Database health check failed")

	// Initialize DAOs
	consentDAO := dao.NewConsentDAO(db)
	auditDAO := dao.NewStatusAuditDAO(db)
	attributeDAO := dao.NewConsentAttributeDAO(db)
	authResourceDAO := dao.NewAuthResourceDAO(db)

	// Initialize services
	consentService := service.NewConsentService(
		consentDAO,
		auditDAO,
		attributeDAO,
		authResourceDAO,
		db,
		logger,
	)

	authResourceService := service.NewAuthResourceService(
		authResourceDAO,
		consentDAO,
		db,
		logger,
	)

	return &TestEnvironment{
		DB:                  db,
		ConsentDAO:          consentDAO,
		AuditDAO:            auditDAO,
		AttributeDAO:        attributeDAO,
		AuthResourceDAO:     authResourceDAO,
		ConsentService:      consentService,
		AuthResourceService: authResourceService,
		Logger:              logger,
		Config:              cfg,
	}
}

// cleanupTestData removes test data from database
func cleanupTestData(t *testing.T, env *TestEnvironment, consentIDs ...string) {
	ctx := context.Background()
	for _, consentID := range consentIDs {
		// Delete will cascade to related tables (attributes, audit, auth resources, files)
		err := env.ConsentDAO.Delete(ctx, consentID, testOrgID)
		if err != nil {
			t.Logf("Warning: Failed to cleanup consent %s: %v", consentID, err)
		}
	}
}

// createTestConsentRequest creates a sample consent request for testing
func createTestConsentRequest() *models.ConsentCreateRequest {
	receiptData := map[string]interface{}{
		"version":    "1.0",
		"purpose":    "Account Information Access",
		"expiryDate": "2025-12-31",
		"data": map[string]interface{}{
			"accountType": "savings",
			"permissions": []string{"read_balance", "read_transactions"},
		},
	}

	validityTime := utils.DaysFromNow(90) // 90 days from now
	frequency := 1
	recurring := false

	return &models.ConsentCreateRequest{
		Receipt:            receiptData,
		ConsentType:        "accounts",
		CurrentStatus:      "awaitingAuthorization",
		ValidityTime:       &validityTime,
		ConsentFrequency:   &frequency,
		RecurringIndicator: &recurring,
		Attributes: map[string]string{
			"source":      "integration-test",
			"environment": "test",
		},
	}
}

// TestConsentCreate_Success tests successful consent creation
func TestConsentCreate_Success(t *testing.T) {
	env := setupTestEnvironment(t)
	defer cleanupTestData(t, env)

	ctx := context.Background()
	request := createTestConsentRequest()

	// Create consent
	response, err := env.ConsentService.CreateConsent(ctx, request, testClientID, testOrgID)

	// Assertions
	require.NoError(t, err, "Failed to create consent")
	require.NotNil(t, response, "Response should not be nil")

	assert.NotEmpty(t, response.ConsentID, "Consent ID should not be empty")
	assert.Contains(t, response.ConsentID, "CONSENT-", "Consent ID should have correct prefix")
	assert.Equal(t, testClientID, response.ClientID, "Client ID should match")
	assert.Equal(t, request.ConsentType, response.ConsentType, "Consent type should match")
	assert.Equal(t, testOrgID, response.OrgID, "Org ID should match")
	assert.Equal(t, "awaitingAuthorization", response.CurrentStatus, "Initial status should be awaitingAuthorization")
	assert.NotZero(t, response.CreatedTime, "Created time should be set")
	assert.NotZero(t, response.UpdatedTime, "Updated time should be set")
	assert.NotNil(t, response.Receipt, "Receipt should not be nil")
	assert.Equal(t, request.Attributes, response.Attributes, "Attributes should match")

	t.Logf("Successfully created consent: %s", response.ConsentID)
}

// TestConsentCreate_WithMinimalData tests consent creation with minimal required data
func TestConsentCreate_WithMinimalData(t *testing.T) {
	env := setupTestEnvironment(t)

	ctx := context.Background()

	request := &models.ConsentCreateRequest{
		Receipt: map[string]interface{}{
			"data": "minimal consent",
		},
		ConsentType:   "payments",
		CurrentStatus: "AUTHORIZED",
	}

	// Create consent
	response, err := env.ConsentService.CreateConsent(ctx, request, "test-client-minimal", testOrgID)

	// Assertions
	require.NoError(t, err, "Failed to create minimal consent")
	require.NotNil(t, response, "Response should not be nil")
	assert.NotEmpty(t, response.ConsentID, "Consent ID should be generated")
	assert.Equal(t, "AUTHORIZED", response.CurrentStatus, "Status should be AUTHORIZED")

	// Cleanup
	cleanupTestData(t, env, response.ConsentID)

	t.Logf("Successfully created minimal consent: %s", response.ConsentID)
}

// TestConsentCreate_InvalidStatus tests that invalid status values are rejected
func TestConsentCreate_InvalidStatus(t *testing.T) {
	env := setupTestEnvironment(t)

	ctx := context.Background()

	request := &models.ConsentCreateRequest{
		Receipt: map[string]interface{}{
			"data": "test consent",
		},
		ConsentType:   "accounts",
		CurrentStatus: "INVALID_STATUS", // Invalid status
	}

	// Attempt to create consent with invalid status
	response, err := env.ConsentService.CreateConsent(ctx, request, testClientID, testOrgID)

	// Assertions
	require.Error(t, err, "Should fail with invalid status")
	require.Nil(t, response, "Response should be nil on validation error")
	assert.Contains(t, err.Error(), "invalid status", "Error should mention invalid status")

	t.Log("Correctly rejected consent with invalid status")
}

// TestConsentCreate_WithAuthResources tests consent creation with authorization resources
func TestConsentCreate_WithAuthResources(t *testing.T) {
	env := setupTestEnvironment(t)

	ctx := context.Background()
	userID := "user-123"

	request := createTestConsentRequest()
	request.AuthResources = []models.ConsentAuthResourceCreateRequest{
		{
			AuthType:   "account",
			UserID:     &userID,
			AuthStatus: "active",
			Resource: map[string]interface{}{
				"accountId":   "ACC-001",
				"accountType": "savings",
				"permissions": []string{"read", "write"},
			},
		},
		{
			AuthType:   "transaction",
			UserID:     &userID,
			AuthStatus: "active",
			Resource: map[string]interface{}{
				"transactionType": "debit",
				"limit":           10000,
			},
		},
	}

	// Create consent with auth resources
	response, err := env.ConsentService.CreateConsent(ctx, request, testClientID, testOrgID)

	// Assertions
	require.NoError(t, err, "Failed to create consent with auth resources")
	require.NotNil(t, response, "Response should not be nil")
	assert.NotEmpty(t, response.ConsentID, "Consent ID should not be empty")
	assert.Equal(t, "awaitingAuthorization", response.CurrentStatus, "Status should be awaitingAuthorization")

	// Verify auth resources were created
	require.NotNil(t, response.AuthResources, "Auth resources should not be nil")
	assert.Len(t, response.AuthResources, 2, "Should have 2 auth resources")

	// Verify auth resources (order may vary)
	authTypeMap := make(map[string]models.ConsentAuthResource)
	for _, authRes := range response.AuthResources {
		authTypeMap[authRes.AuthType] = authRes
	}

	// Verify account auth resource
	accountAuth, exists := authTypeMap["account"]
	require.True(t, exists, "Account auth resource should exist")
	assert.Equal(t, "active", accountAuth.AuthStatus, "Account auth status should be active")
	assert.Equal(t, &userID, accountAuth.UserID, "User ID should match")

	// Verify transaction auth resource
	transactionAuth, exists := authTypeMap["transaction"]
	require.True(t, exists, "Transaction auth resource should exist")
	assert.Equal(t, "active", transactionAuth.AuthStatus, "Transaction auth status should be active")
	assert.Equal(t, &userID, transactionAuth.UserID, "User ID should match")

	// Cleanup
	cleanupTestData(t, env, response.ConsentID)

	t.Logf("Successfully created consent with %d auth resources: %s", len(response.AuthResources), response.ConsentID)
}

// TestConsentGet_Success tests successful consent retrieval
func TestConsentGet_Success(t *testing.T) {
	env := setupTestEnvironment(t)

	ctx := context.Background()
	userID := "user-123"
	request := createTestConsentRequest()
	request.AuthResources = []models.ConsentAuthResourceCreateRequest{
		{
			AuthType:   "account",
			UserID:     &userID,
			AuthStatus: "active",
			Resource: map[string]interface{}{
				"accountId":   "ACC-001",
				"accountType": "savings",
				"permissions": []string{"read", "write"},
			},
		},
		{
			AuthType:   "transaction",
			UserID:     &userID,
			AuthStatus: "active",
			Resource: map[string]interface{}{
				"transactionType": "debit",
				"limit":           10000,
			},
		},
	}

	// First, create a consent
	createResponse, err := env.ConsentService.CreateConsent(ctx, request, testClientID, testOrgID)
	require.NoError(t, err, "Failed to create consent for get test")
	require.NotNil(t, createResponse, "Create response should not be nil")

	defer cleanupTestData(t, env, createResponse.ConsentID)

	// Now retrieve it
	getResponse, err := env.ConsentService.GetConsent(ctx, createResponse.ConsentID, testOrgID)

	// Assertions
	require.NoError(t, err, "Failed to get consent")
	require.NotNil(t, getResponse, "Get response should not be nil")

	assert.Equal(t, createResponse.ConsentID, getResponse.ConsentID, "Consent IDs should match")
	assert.Equal(t, createResponse.ClientID, getResponse.ClientID, "Client IDs should match")
	assert.Equal(t, createResponse.ConsentType, getResponse.ConsentType, "Consent types should match")
	assert.Equal(t, createResponse.CurrentStatus, getResponse.CurrentStatus, "Statuses should match")
	assert.Equal(t, createResponse.OrgID, getResponse.OrgID, "Org IDs should match")

	t.Logf("Successfully retrieved consent: %s", getResponse.ConsentID)
}

// TestConsentGet_NotFound tests retrieving non-existent consent
func TestConsentGet_NotFound(t *testing.T) {
	env := setupTestEnvironment(t)

	ctx := context.Background()
	orgID := "TEST_ORG"
	nonExistentID := "CONSENT-nonexistent-12345"

	// Try to get non-existent consent
	response, err := env.ConsentService.GetConsent(ctx, nonExistentID, orgID)

	// Assertions
	assert.Error(t, err, "Should return error for non-existent consent")
	assert.Nil(t, response, "Response should be nil for non-existent consent")

	t.Logf("Correctly handled non-existent consent retrieval")
}

// TestConsentRevoke_Success tests successful consent revocation
func TestConsentRevoke_Success(t *testing.T) {
	env := setupTestEnvironment(t)

	ctx := context.Background()
	orgID := "TEST_ORG"
	request := createTestConsentRequest()

	// First, create a consent
	createResponse, err := env.ConsentService.CreateConsent(ctx, request, testClientID, testOrgID)
	require.NoError(t, err, "Failed to create consent for revoke test")
	require.NotNil(t, createResponse, "Create response should not be nil")

	defer cleanupTestData(t, env, createResponse.ConsentID)

	// Now revoke it
	reason := "Integration test revocation"
	actionBy := "test-user"
	err = env.ConsentService.RevokeConsent(ctx, createResponse.ConsentID, orgID, reason, actionBy)

	// Assertions
	require.NoError(t, err, "Failed to revoke consent")

	// Verify the consent is revoked by retrieving it
	revokedConsent, err := env.ConsentService.GetConsent(ctx, createResponse.ConsentID, orgID)
	require.NoError(t, err, "Failed to get revoked consent")

	assert.Equal(t, "REVOKED", revokedConsent.CurrentStatus, "Status should be REVOKED")

	// Verify audit trail was created
	audits, err := env.AuditDAO.GetByConsentID(ctx, createResponse.ConsentID, orgID)
	require.NoError(t, err, "Failed to get audit records")
	assert.GreaterOrEqual(t, len(audits), 1, "Should have at least one audit record")

	// Check the latest audit record
	hasRevokeAudit := false
	for _, audit := range audits {
		if audit.CurrentStatus == "REVOKED" {
			hasRevokeAudit = true
			if audit.Reason != nil {
				assert.Equal(t, reason, *audit.Reason, "Revoke reason should match")
			}
			if audit.ActionBy != nil {
				assert.Equal(t, actionBy, *audit.ActionBy, "ActionBy should match")
			}
		}
	}
	assert.True(t, hasRevokeAudit, "Should have revoke audit record")

	t.Logf("Successfully revoked consent: %s", createResponse.ConsentID)
}

// TestConsentDelete_Success tests successful consent deletion
func TestConsentDelete_Success(t *testing.T) {
	env := setupTestEnvironment(t)

	ctx := context.Background()
	orgID := "TEST_ORG"
	request := createTestConsentRequest()

	// First, create a consent
	createResponse, err := env.ConsentService.CreateConsent(ctx, request, testClientID, testOrgID)
	require.NoError(t, err, "Failed to create consent for delete test")
	require.NotNil(t, createResponse, "Create response should not be nil")

	consentID := createResponse.ConsentID

	// Delete the consent
	err = env.ConsentDAO.Delete(ctx, consentID, orgID)
	require.NoError(t, err, "Failed to delete consent")

	// Verify it's deleted by trying to retrieve it
	_, err = env.ConsentDAO.GetByID(ctx, consentID, orgID)
	assert.Error(t, err, "Should return error when getting deleted consent")

	t.Logf("Successfully deleted consent: %s", consentID)
}

// TestConsentSearch_Success tests consent search functionality
func TestConsentSearch_Success(t *testing.T) {
	env := setupTestEnvironment(t)

	ctx := context.Background()
	clientID := "test-client-search"

	// Create multiple consents
	var createdIDs []string
	for i := 0; i < 3; i++ {
		request := createTestConsentRequest()

		response, err := env.ConsentService.CreateConsent(ctx, request, clientID, testOrgID)
		require.NoError(t, err, "Failed to create consent for search test")
		createdIDs = append(createdIDs, response.ConsentID)
	}

	defer cleanupTestData(t, env, createdIDs...)

	// Search for consents
	searchParams := &models.ConsentSearchParams{
		ClientIDs: []string{clientID},
		OrgID:     testOrgID,
		Limit:     10,
		Offset:    0,
	}

	results, pagination, err := env.ConsentService.SearchConsents(ctx, searchParams)

	// Assertions
	require.NoError(t, err, "Failed to search consents")
	require.NotNil(t, results, "Results should not be nil")
	assert.GreaterOrEqual(t, len(results), 3, "Should find at least 3 consents")
	assert.NotNil(t, pagination, "Pagination should not be nil")
	assert.GreaterOrEqual(t, pagination.Total, 3, "Total records should be at least 3")

	// Verify all created consents are in results
	foundIDs := make(map[string]bool)
	for _, result := range results {
		foundIDs[result.ConsentID] = true
	}

	for _, createdID := range createdIDs {
		assert.True(t, foundIDs[createdID], "Created consent %s should be in search results", createdID)
	}

	t.Logf("Successfully searched and found %d consents", len(results))
}

// TestConsentWithAttributes_Success tests consent with attributes
func TestConsentWithAttributes_Success(t *testing.T) {
	env := setupTestEnvironment(t)

	ctx := context.Background()
	orgID := "TEST_ORG"
	request := createTestConsentRequest()

	// Add custom attributes
	request.Attributes = map[string]string{
		"customField1": "value1",
		"customField2": "value2",
		"testFlag":     "true",
	}

	// Create consent
	createResponse, err := env.ConsentService.CreateConsent(ctx, request, testClientID, testOrgID)
	require.NoError(t, err, "Failed to create consent with attributes")

	defer cleanupTestData(t, env, createResponse.ConsentID)

	// Verify attributes were stored
	assert.NotNil(t, createResponse.Attributes, "Attributes should not be nil")
	assert.Equal(t, "value1", createResponse.Attributes["customField1"], "Attribute value should match")
	assert.Equal(t, "value2", createResponse.Attributes["customField2"], "Attribute value should match")
	assert.Equal(t, "true", createResponse.Attributes["testFlag"], "Attribute value should match")

	// Retrieve and verify attributes persisted
	getResponse, err := env.ConsentService.GetConsent(ctx, createResponse.ConsentID, orgID)
	require.NoError(t, err, "Failed to get consent")

	assert.Equal(t, request.Attributes, getResponse.Attributes, "Retrieved attributes should match")

	t.Logf("Successfully created and retrieved consent with attributes: %s", createResponse.ConsentID)
}

// TestConsentLifecycle_Complete tests full consent lifecycle
func TestConsentLifecycle_Complete(t *testing.T) {
	env := setupTestEnvironment(t)

	ctx := context.Background()

	// Step 1: Create consent
	request := createTestConsentRequest()
	createResponse, err := env.ConsentService.CreateConsent(ctx, request, testClientID, testOrgID)
	require.NoError(t, err, "Step 1: Failed to create consent")
	assert.Equal(t, "awaitingAuthorization", createResponse.CurrentStatus, "Initial status should be awaitingAuthorization")
	t.Logf("Step 1: Created consent %s with status %s", createResponse.ConsentID, createResponse.CurrentStatus)

	defer cleanupTestData(t, env, createResponse.ConsentID)

	// Step 2: Retrieve consent
	getResponse, err := env.ConsentService.GetConsent(ctx, createResponse.ConsentID, testOrgID)
	require.NoError(t, err, "Step 2: Failed to get consent")
	assert.Equal(t, createResponse.ConsentID, getResponse.ConsentID, "Retrieved consent ID should match")
	t.Logf("Step 2: Retrieved consent %s", getResponse.ConsentID)

	// Step 3: Search for consent
	searchParams := &models.ConsentSearchParams{
		ClientIDs: []string{testClientID},
		OrgID:     testOrgID,
		Limit:     10,
		Offset:    0,
	}
	searchResults, _, err := env.ConsentService.SearchConsents(ctx, searchParams)
	require.NoError(t, err, "Step 3: Failed to search consents")
	assert.Greater(t, len(searchResults), 0, "Should find at least one consent")
	t.Logf("Step 3: Found %d consent(s) in search", len(searchResults))

	// Step 4: Revoke consent
	err = env.ConsentService.RevokeConsent(ctx, createResponse.ConsentID, testOrgID, "Lifecycle test", "test-admin")
	require.NoError(t, err, "Step 4: Failed to revoke consent")
	t.Logf("Step 4: Revoked consent %s", createResponse.ConsentID)

	// Step 5: Verify revocation
	revokedConsent, err := env.ConsentService.GetConsent(ctx, createResponse.ConsentID, testOrgID)
	require.NoError(t, err, "Step 5: Failed to get revoked consent")
	assert.Equal(t, "REVOKED", revokedConsent.CurrentStatus, "Status should be REVOKED")
	t.Logf("Step 5: Verified consent status is %s", revokedConsent.CurrentStatus)

	// Step 6: Check audit trail
	audits, err := env.AuditDAO.GetByConsentID(ctx, createResponse.ConsentID, testOrgID)
	require.NoError(t, err, "Step 6: Failed to get audit trail")
	assert.GreaterOrEqual(t, len(audits), 2, "Should have at least 2 audit records (creation + revocation)")
	t.Logf("Step 6: Found %d audit record(s)", len(audits))

	// Step 7: Delete consent
	err = env.ConsentDAO.Delete(ctx, createResponse.ConsentID, testOrgID)
	require.NoError(t, err, "Step 7: Failed to delete consent")
	t.Logf("Step 7: Deleted consent %s", createResponse.ConsentID)

	// Step 8: Verify deletion
	_, err = env.ConsentDAO.GetByID(ctx, createResponse.ConsentID, testOrgID)
	assert.Error(t, err, "Step 8: Should not find deleted consent")
	t.Logf("Step 8: Verified consent is deleted")

	t.Log("✓ Complete consent lifecycle test passed")
}

// TestConsentJSON_Serialization tests JSON receipt serialization
func TestConsentJSON_Serialization(t *testing.T) {
	env := setupTestEnvironment(t)

	ctx := context.Background()

	// Create complex receipt structure
	receiptData := map[string]interface{}{
		"version": "2.0",
		"metadata": map[string]interface{}{
			"created": "2025-01-01T00:00:00Z",
			"source":  "mobile-app",
		},
		"permissions": []interface{}{
			map[string]interface{}{
				"resource": "account",
				"actions":  []interface{}{"read", "write"},
			},
			map[string]interface{}{
				"resource": "transaction",
				"actions":  []interface{}{"read"},
			},
		},
		"constraints": map[string]interface{}{
			"maxAmount": 1000.50,
			"currency":  "USD",
		},
	}

	request := &models.ConsentCreateRequest{
		Receipt:       receiptData,
		ConsentType:   "financial",
		CurrentStatus: "AUTHORIZED",
	}

	// Create consent
	response, err := env.ConsentService.CreateConsent(ctx, request, "test-client-json", testOrgID)
	require.NoError(t, err, "Failed to create consent with complex JSON")

	defer cleanupTestData(t, env, response.ConsentID) // Verify JSON structure is preserved
	assert.NotNil(t, response.Receipt, "Receipt should not be nil")

	receiptJSON, err := json.Marshal(response.Receipt)
	require.NoError(t, err, "Failed to marshal receipt")

	var deserializedReceipt map[string]interface{}
	err = json.Unmarshal(receiptJSON, &deserializedReceipt)
	require.NoError(t, err, "Failed to unmarshal receipt")

	assert.Equal(t, "2.0", deserializedReceipt["version"], "Version should be preserved")
	assert.NotNil(t, deserializedReceipt["metadata"], "Metadata should be preserved")
	assert.NotNil(t, deserializedReceipt["permissions"], "Permissions should be preserved")

	t.Logf("Successfully verified JSON serialization for consent: %s", response.ConsentID)
}

// TestConsentUpdate_FullPayload tests updating consent with all fields
func TestConsentUpdate_FullPayload(t *testing.T) {
	env := setupTestEnvironment(t)

	ctx := context.Background()

	// Step 1: Create initial consent
	t.Log("Step 1: Creating initial consent...")
	createRequest := createTestConsentRequest()
	createRequest.Attributes = map[string]string{
		"purpose":    "account_access",
		"dataAccess": "read_only",
	}
	createRequest.AuthResources = []models.ConsentAuthResourceCreateRequest{
		{
			AuthType:   "account",
			UserID:     stringPtr("user-123"),
			AuthStatus: "AUTHORIZED",
			Resource: map[string]interface{}{
				"accountId": "ACC-001",
				"type":      "savings",
			},
		},
	}

	created, err := env.ConsentService.CreateConsent(ctx, createRequest, testClientID, testOrgID)
	require.NoError(t, err, "Step 1: Should create consent successfully")
	assert.Equal(t, "awaitingAuthorization", created.CurrentStatus, "Initial status should be awaitingAuthorization")
	assert.Equal(t, "account_access", created.Attributes["purpose"], "Initial attribute should match")
	assert.Len(t, created.AuthResources, 1, "Should have 1 auth resource")
	t.Logf("Step 1: Created consent %s", created.ConsentID)

	defer cleanupTestData(t, env, created.ConsentID)

	// Step 2: Update consent with full payload
	t.Log("Step 2: Updating consent with full payload...")

	newReceipt := map[string]interface{}{
		"version":    "2.0",
		"purpose":    "Enhanced Account Access",
		"expiryDate": "2026-12-31",
		"data": map[string]interface{}{
			"accountType": "checking",
			"permissions": []string{"read_balance", "read_transactions", "initiate_payment"},
		},
	}

	newValidityTime := utils.DaysFromNow(180) // Extend to 180 days
	newFrequency := 2
	newRecurring := true

	updateRequest := &models.ConsentUpdateRequest{
		Receipt:            newReceipt,
		CurrentStatus:      "ACTIVE",
		ConsentFrequency:   &newFrequency,
		ValidityTime:       &newValidityTime,
		RecurringIndicator: &newRecurring,
		Attributes: map[string]string{
			"purpose":     "enhanced_access",
			"dataAccess":  "read_write",
			"newField":    "newValue",
			"permissions": "full_access",
		},
		AuthResources: []models.ConsentAuthResourceCreateRequest{
			{
				AuthType:   "account",
				UserID:     stringPtr("user-456"),
				AuthStatus: "ACTIVE",
				Resource: map[string]interface{}{
					"accountId": "ACC-002",
					"type":      "checking",
				},
			},
			{
				AuthType:   "device",
				UserID:     stringPtr("user-456"),
				AuthStatus: "ACTIVE",
				Resource: map[string]interface{}{
					"deviceId":   "DEV-001",
					"deviceType": "mobile",
				},
			},
		},
	}

	updated, err := env.ConsentService.UpdateConsent(ctx, created.ConsentID, testOrgID, updateRequest)
	require.NoError(t, err, "Step 2: Should update consent successfully")
	t.Logf("Step 2: Updated consent %s", updated.ConsentID)

	// Step 3: Verify all updated fields
	t.Log("Step 3: Verifying updated fields...")
	assert.Equal(t, created.ConsentID, updated.ConsentID, "Consent ID should remain same")
	assert.Equal(t, "ACTIVE", updated.CurrentStatus, "Status should be updated to ACTIVE")
	assert.Equal(t, newFrequency, *updated.ConsentFrequency, "Frequency should be updated")
	assert.Equal(t, newValidityTime, *updated.ValidityTime, "Validity time should be updated")
	assert.Equal(t, newRecurring, *updated.RecurringIndicator, "Recurring indicator should be updated")
	assert.Greater(t, updated.UpdatedTime, created.UpdatedTime, "Updated time should be greater")

	// Verify receipt
	assert.NotNil(t, updated.Receipt, "Receipt should not be nil")
	assert.Equal(t, "2.0", updated.Receipt["version"], "Receipt version should be updated")
	assert.Equal(t, "Enhanced Account Access", updated.Receipt["purpose"], "Receipt purpose should be updated")

	// Verify attributes
	assert.Equal(t, "enhanced_access", updated.Attributes["purpose"], "Purpose attribute should be updated")
	assert.Equal(t, "read_write", updated.Attributes["dataAccess"], "DataAccess attribute should be updated")
	assert.Equal(t, "newValue", updated.Attributes["newField"], "New attribute should be added")
	assert.Equal(t, "full_access", updated.Attributes["permissions"], "Permissions attribute should be added")

	// Verify auth resources were replaced with new ones
	assert.Len(t, updated.AuthResources, 2, "Should have 2 auth resources after update")

	// Find auth resources by type
	var accountAuth, deviceAuth *models.ConsentAuthResource
	for i := range updated.AuthResources {
		if updated.AuthResources[i].AuthType == "account" {
			accountAuth = &updated.AuthResources[i]
		} else if updated.AuthResources[i].AuthType == "device" {
			deviceAuth = &updated.AuthResources[i]
		}
	}

	require.NotNil(t, accountAuth, "Should have account auth resource")
	require.NotNil(t, deviceAuth, "Should have device auth resource")

	assert.Equal(t, "ACTIVE", accountAuth.AuthStatus, "Account auth status should be ACTIVE")
	assert.Equal(t, "user-456", *accountAuth.UserID, "Account user ID should be updated")

	assert.Equal(t, "ACTIVE", deviceAuth.AuthStatus, "Device auth status should be ACTIVE")
	assert.Equal(t, "user-456", *deviceAuth.UserID, "Device user ID should be updated")

	t.Logf("Step 3: All fields verified successfully")

	// Step 4: Verify status audit was created
	t.Log("Step 4: Verifying status audit trail...")
	auditRecords, err := env.AuditDAO.GetByConsentID(ctx, created.ConsentID, testOrgID)
	require.NoError(t, err, "Should retrieve audit records")
	assert.GreaterOrEqual(t, len(auditRecords), 2, "Should have at least 2 audit records (create + update)")

	// Find the update audit record
	var updateAudit *models.ConsentStatusAudit
	for i := range auditRecords {
		if auditRecords[i].CurrentStatus == "ACTIVE" {
			updateAudit = &auditRecords[i]
			break
		}
	}
	require.NotNil(t, updateAudit, "Should have audit record for status update")
	assert.Equal(t, "awaitingAuthorization", *updateAudit.PreviousStatus, "Previous status should be awaitingAuthorization")
	assert.Equal(t, "ACTIVE", updateAudit.CurrentStatus, "Current status should be ACTIVE")
	t.Logf("Step 4: Status audit verified - %d records found", len(auditRecords))

	// Step 5: Retrieve consent and verify persistence
	t.Log("Step 5: Retrieving consent to verify persistence...")
	retrieved, err := env.ConsentService.GetConsent(ctx, created.ConsentID, testOrgID)
	require.NoError(t, err, "Should retrieve updated consent")

	assert.Equal(t, updated.ConsentID, retrieved.ConsentID, "Retrieved consent ID should match")
	assert.Equal(t, "ACTIVE", retrieved.CurrentStatus, "Retrieved status should be ACTIVE")
	assert.Equal(t, "enhanced_access", retrieved.Attributes["purpose"], "Retrieved attributes should persist")
	assert.Len(t, retrieved.AuthResources, 2, "Retrieved consent should have 2 auth resources")
	assert.NotNil(t, retrieved.Receipt, "Retrieved receipt should not be nil")
	if retrieved.Receipt != nil {
		assert.Equal(t, "2.0", retrieved.Receipt["version"], "Retrieved receipt version should persist")
	}
	t.Logf("Step 5: Retrieved consent verified")

	t.Log("✓ Full payload update test completed successfully")
}

// TestConsentUpdate_PartialUpdate tests updating consent with only some fields
func TestConsentUpdate_PartialUpdate(t *testing.T) {
	env := setupTestEnvironment(t)

	ctx := context.Background()

	// Create initial consent
	createRequest := createTestConsentRequest()
	createRequest.Attributes = map[string]string{
		"purpose": "original_purpose",
		"scope":   "read_only",
	}

	created, err := env.ConsentService.CreateConsent(ctx, createRequest, testClientID, testOrgID)
	require.NoError(t, err, "Should create consent")

	defer cleanupTestData(t, env, created.ConsentID)

	// Update only status
	updateRequest := &models.ConsentUpdateRequest{
		CurrentStatus: "ACTIVE",
	}

	updated, err := env.ConsentService.UpdateConsent(ctx, created.ConsentID, testOrgID, updateRequest)
	require.NoError(t, err, "Should update consent with partial payload")

	// Verify only status changed
	assert.Equal(t, "ACTIVE", updated.CurrentStatus, "Status should be updated")
	assert.Equal(t, created.ConsentType, updated.ConsentType, "Consent type should remain same")
	assert.Equal(t, created.Attributes["purpose"], updated.Attributes["purpose"], "Attributes should remain same")

	// Verify receipt remains unchanged
	originalReceipt, _ := json.Marshal(created.Receipt)
	updatedReceipt, _ := json.Marshal(updated.Receipt)
	assert.JSONEq(t, string(originalReceipt), string(updatedReceipt), "Receipt should remain unchanged")

	t.Log("✓ Partial update test completed successfully")
}

// TestConsentUpdate_AttributesOnly tests updating only attributes
func TestConsentUpdate_AttributesOnly(t *testing.T) {
	env := setupTestEnvironment(t)

	ctx := context.Background()

	// Create initial consent
	createRequest := createTestConsentRequest()
	createRequest.Attributes = map[string]string{
		"key1": "value1",
		"key2": "value2",
	}

	created, err := env.ConsentService.CreateConsent(ctx, createRequest, testClientID, testOrgID)
	require.NoError(t, err, "Should create consent")

	defer cleanupTestData(t, env, created.ConsentID)

	// Update only attributes
	updateRequest := &models.ConsentUpdateRequest{
		Attributes: map[string]string{
			"key1":    "updated_value1",
			"key3":    "value3",
			"newAttr": "newValue",
		},
	}

	updated, err := env.ConsentService.UpdateConsent(ctx, created.ConsentID, testOrgID, updateRequest)
	require.NoError(t, err, "Should update attributes")

	// Verify attributes are replaced (not merged)
	assert.Equal(t, "updated_value1", updated.Attributes["key1"], "key1 should be updated")
	assert.Equal(t, "value3", updated.Attributes["key3"], "key3 should be added")
	assert.Equal(t, "newValue", updated.Attributes["newAttr"], "newAttr should be added")
	assert.Empty(t, updated.Attributes["key2"], "key2 should be removed (attributes replaced)")

	// Verify status remains unchanged (no new audit record should be created for status)
	assert.Equal(t, created.CurrentStatus, updated.CurrentStatus, "Status should remain same")

	t.Log("✓ Attributes-only update test completed successfully")
}

// TestConsentUpdate_InvalidStatus tests update with invalid status
func TestConsentUpdate_InvalidStatus(t *testing.T) {
	env := setupTestEnvironment(t)

	ctx := context.Background()

	// Create consent
	createRequest := createTestConsentRequest()
	created, err := env.ConsentService.CreateConsent(ctx, createRequest, testClientID, testOrgID)
	require.NoError(t, err, "Should create consent")

	defer cleanupTestData(t, env, created.ConsentID)

	// Try to update with invalid status
	updateRequest := &models.ConsentUpdateRequest{
		CurrentStatus: "INVALID_STATUS",
	}

	_, err = env.ConsentService.UpdateConsent(ctx, created.ConsentID, testOrgID, updateRequest)
	assert.Error(t, err, "Should reject invalid status")
	assert.Contains(t, err.Error(), "invalid status", "Error should mention invalid status")

	t.Log("✓ Invalid status rejection test completed successfully")
}

// TestConsentUpdate_NonExistent tests updating non-existent consent
func TestConsentUpdate_NonExistent(t *testing.T) {
	env := setupTestEnvironment(t)

	ctx := context.Background()

	updateRequest := &models.ConsentUpdateRequest{
		CurrentStatus: "ACTIVE",
	}

	_, err := env.ConsentService.UpdateConsent(ctx, "CONSENT-nonexistent", testOrgID, updateRequest)
	assert.Error(t, err, "Should fail to update non-existent consent")
	assert.Contains(t, err.Error(), "consent not found", "Error should mention consent not found")

	t.Log("✓ Non-existent consent test completed successfully")
}

// Helper function to create string pointer
func stringPtr(s string) *string {
	return &s
}
