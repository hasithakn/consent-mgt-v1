package integration

import (
	"context"
	"testing"

	"github.com/wso2/consent-management-api/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testAuthResourceClientID = "test-client-auth-res"
	testAuthResourceOrgID    = "test-org-auth-res"
)

// TestAuthResourceCreate_Success tests creating an auth resource successfully
func TestAuthResourceCreate_Success(t *testing.T) {
	env := setupTestEnvironment(t)
	ctx := context.Background()

	// Step 1: Create a consent first
	t.Log("Step 1: Creating consent...")
	consentRequest := createTestConsentRequest()
	consent, err := env.ConsentService.CreateConsent(ctx, consentRequest, testAuthResourceClientID, testAuthResourceOrgID)
	require.NoError(t, err, "Should create consent successfully")
	defer cleanupTestData(t, env, consent.ConsentID)

	// Step 2: Create auth resource
	t.Log("Step 2: Creating auth resource...")
	authResourceRequest := &models.ConsentAuthResourceCreateRequest{
		AuthType:   "user",
		UserID:     stringPtr("user-123"),
		AuthStatus: "pending",
		ApprovedPurposeDetails: &models.ApprovedPurposeDetails{
			ApprovedPurposesNames:       []string{"utility_read", "taxes_read"},
			ApprovedAdditionalResources: []interface{}{},
		},
	}

	authResource, err := env.AuthResourceService.CreateAuthResource(ctx, consent.ConsentID, testAuthResourceOrgID, authResourceRequest)
	require.NoError(t, err, "Should create auth resource successfully")
	require.NotNil(t, authResource, "Auth resource should not be nil")

	// Step 3: Verify auth resource fields
	t.Log("Step 3: Verifying auth resource fields...")
	assert.NotEmpty(t, authResource.AuthID, "Auth ID should not be empty")
	assert.Contains(t, authResource.AuthID, "AUTH-", "Auth ID should have correct prefix")
	assert.Equal(t, consent.ConsentID, authResource.ConsentID, "Consent ID should match")
	assert.Equal(t, "user", authResource.AuthType, "Auth type should match")
	assert.Equal(t, "user-123", *authResource.UserID, "User ID should match")
	assert.Equal(t, "pending", authResource.AuthStatus, "Auth status should match")
	assert.Equal(t, testAuthResourceOrgID, authResource.OrgID, "Org ID should match")
	assert.NotZero(t, authResource.UpdatedTime, "Updated time should be set")

	// Verify approved purpose details
	assert.NotNil(t, authResource.ApprovedPurposeDetails, "ApprovedPurposeDetails should not be nil")
	assert.Len(t, authResource.ApprovedPurposeDetails.ApprovedPurposesNames, 2, "Should have 2 approved purposes")
	assert.Contains(t, authResource.ApprovedPurposeDetails.ApprovedPurposesNames, "utility_read", "Should contain utility_read purpose")
	assert.Contains(t, authResource.ApprovedPurposeDetails.ApprovedPurposesNames, "taxes_read", "Should contain taxes_read purpose")

	t.Logf("✓ Successfully created auth resource: %s", authResource.AuthID)
}

// TestAuthResourceCreate_InvalidConsentID tests creating auth resource with invalid consent ID
func TestAuthResourceCreate_InvalidConsentID(t *testing.T) {
	env := setupTestEnvironment(t)
	ctx := context.Background()

	authResourceRequest := &models.ConsentAuthResourceCreateRequest{
		AuthType:   "user",
		AuthStatus: "pending",
	}

	_, err := env.AuthResourceService.CreateAuthResource(ctx, "CONSENT-nonexistent", testAuthResourceOrgID, authResourceRequest)
	assert.Error(t, err, "Should fail with non-existent consent ID")
	assert.Contains(t, err.Error(), "consent not found", "Error should mention consent not found")

	t.Log("✓ Correctly rejected invalid consent ID")
}

// TestAuthResourceGet_Success tests retrieving an auth resource by ID
func TestAuthResourceGet_Success(t *testing.T) {
	env := setupTestEnvironment(t)
	ctx := context.Background()

	// Setup: Create consent and auth resource
	consentRequest := createTestConsentRequest()
	consent, err := env.ConsentService.CreateConsent(ctx, consentRequest, testAuthResourceClientID, testAuthResourceOrgID)
	require.NoError(t, err)
	defer cleanupTestData(t, env, consent.ConsentID)

	authResourceRequest := &models.ConsentAuthResourceCreateRequest{
		AuthType:   "device",
		UserID:     stringPtr("user-456"),
		AuthStatus: "active",
		ApprovedPurposeDetails: &models.ApprovedPurposeDetails{
			ApprovedPurposesNames:       []string{"profile_read"},
			ApprovedAdditionalResources: []interface{}{},
		},
	}

	created, err := env.AuthResourceService.CreateAuthResource(ctx, consent.ConsentID, testAuthResourceOrgID, authResourceRequest)
	require.NoError(t, err)

	// Test: Retrieve the auth resource
	t.Log("Retrieving auth resource...")
	retrieved, err := env.AuthResourceService.GetAuthResource(ctx, created.AuthID, testAuthResourceOrgID)
	require.NoError(t, err, "Should retrieve auth resource successfully")
	require.NotNil(t, retrieved, "Retrieved auth resource should not be nil")

	// Verify fields match
	assert.Equal(t, created.AuthID, retrieved.AuthID, "Auth ID should match")
	assert.Equal(t, created.ConsentID, retrieved.ConsentID, "Consent ID should match")
	assert.Equal(t, created.AuthType, retrieved.AuthType, "Auth type should match")
	assert.Equal(t, created.UserID, retrieved.UserID, "User ID should match")
	assert.Equal(t, created.AuthStatus, retrieved.AuthStatus, "Auth status should match")
	assert.NotNil(t, retrieved.ApprovedPurposeDetails, "ApprovedPurposeDetails should not be nil")
	assert.Contains(t, retrieved.ApprovedPurposeDetails.ApprovedPurposesNames, "profile_read", "Should contain profile_read purpose")

	t.Logf("✓ Successfully retrieved auth resource: %s", retrieved.AuthID)
}

// TestAuthResourceGet_NotFound tests retrieving a non-existent auth resource
func TestAuthResourceGet_NotFound(t *testing.T) {
	env := setupTestEnvironment(t)
	ctx := context.Background()

	_, err := env.AuthResourceService.GetAuthResource(ctx, "AUTH-nonexistent", testAuthResourceOrgID)
	assert.Error(t, err, "Should fail to retrieve non-existent auth resource")
	assert.Contains(t, err.Error(), "auth resource not found", "Error should mention auth resource not found")

	t.Log("✓ Correctly handled non-existent auth resource")
}

// TestAuthResourceGetByConsentID_Success tests retrieving all auth resources for a consent
func TestAuthResourceGetByConsentID_Success(t *testing.T) {
	env := setupTestEnvironment(t)
	ctx := context.Background()

	// Setup: Create consent
	consentRequest := createTestConsentRequest()
	consent, err := env.ConsentService.CreateConsent(ctx, consentRequest, testAuthResourceClientID, testAuthResourceOrgID)
	require.NoError(t, err)
	defer cleanupTestData(t, env, consent.ConsentID)

	// Create multiple auth resources
	t.Log("Creating multiple auth resources...")

	authResource1 := &models.ConsentAuthResourceCreateRequest{
		AuthType:   "user",
		UserID:     stringPtr("user-1"),
		AuthStatus: "pending",
		ApprovedPurposeDetails: &models.ApprovedPurposeDetails{
			ApprovedPurposesNames:       []string{"utility_read"},
			ApprovedAdditionalResources: []interface{}{},
		},
	}

	authResource2 := &models.ConsentAuthResourceCreateRequest{
		AuthType:   "device",
		UserID:     stringPtr("user-1"),
		AuthStatus: "active",
		ApprovedPurposeDetails: &models.ApprovedPurposeDetails{
			ApprovedPurposesNames:       []string{"profile_read"},
			ApprovedAdditionalResources: []interface{}{},
		},
	}

	authResource3 := &models.ConsentAuthResourceCreateRequest{
		AuthType:   "account",
		UserID:     stringPtr("user-2"),
		AuthStatus: "authorized",
		ApprovedPurposeDetails: &models.ApprovedPurposeDetails{
			ApprovedPurposesNames:       []string{"taxes_read"},
			ApprovedAdditionalResources: []interface{}{},
		},
	}

	_, err = env.AuthResourceService.CreateAuthResource(ctx, consent.ConsentID, testAuthResourceOrgID, authResource1)
	require.NoError(t, err)

	_, err = env.AuthResourceService.CreateAuthResource(ctx, consent.ConsentID, testAuthResourceOrgID, authResource2)
	require.NoError(t, err)

	_, err = env.AuthResourceService.CreateAuthResource(ctx, consent.ConsentID, testAuthResourceOrgID, authResource3)
	require.NoError(t, err)

	// Test: Retrieve all auth resources for the consent
	t.Log("Retrieving all auth resources for consent...")
	result, err := env.AuthResourceService.GetAuthResourcesByConsentID(ctx, consent.ConsentID, testAuthResourceOrgID)
	require.NoError(t, err, "Should retrieve auth resources successfully")
	require.NotNil(t, result, "Result should not be nil")

	// Verify we got all 3 auth resources
	assert.Len(t, result.Data, 3, "Should have 3 auth resources")

	// Verify auth types are present
	authTypes := make(map[string]bool)
	for _, ar := range result.Data {
		authTypes[ar.AuthType] = true
		assert.Equal(t, consent.ConsentID, ar.ConsentID, "All auth resources should belong to same consent")
	}

	assert.True(t, authTypes["user"], "Should have user auth type")
	assert.True(t, authTypes["device"], "Should have device auth type")
	assert.True(t, authTypes["account"], "Should have account auth type")

	t.Logf("✓ Successfully retrieved %d auth resources for consent", len(result.Data))
}

// TestAuthResourceUpdate_Success tests updating an auth resource
func TestAuthResourceUpdate_Success(t *testing.T) {
	env := setupTestEnvironment(t)
	ctx := context.Background()

	// Setup: Create consent and auth resource
	consentRequest := createTestConsentRequest()
	consent, err := env.ConsentService.CreateConsent(ctx, consentRequest, testAuthResourceClientID, testAuthResourceOrgID)
	require.NoError(t, err)
	defer cleanupTestData(t, env, consent.ConsentID)

	authResourceRequest := &models.ConsentAuthResourceCreateRequest{
		AuthType:   "user",
		UserID:     stringPtr("user-original"),
		AuthStatus: "pending",
		ApprovedPurposeDetails: &models.ApprovedPurposeDetails{
			ApprovedPurposesNames:       []string{"utility_read"},
			ApprovedAdditionalResources: []interface{}{},
		},
	}

	created, err := env.AuthResourceService.CreateAuthResource(ctx, consent.ConsentID, testAuthResourceOrgID, authResourceRequest)
	require.NoError(t, err)

	// Test: Update the auth resource
	t.Log("Updating auth resource...")
	updateRequest := &models.ConsentAuthResourceUpdateRequest{
		AuthStatus: "authorized",
		UserID:     stringPtr("user-updated"),
		ApprovedPurposeDetails: &models.ApprovedPurposeDetails{
			ApprovedPurposesNames:       []string{"utility_read", "taxes_read", "profile_read"},
			ApprovedAdditionalResources: []interface{}{},
		},
	}

	updated, err := env.AuthResourceService.UpdateAuthResource(ctx, created.AuthID, testAuthResourceOrgID, updateRequest)
	require.NoError(t, err, "Should update auth resource successfully")
	require.NotNil(t, updated, "Updated auth resource should not be nil")

	// Verify updated fields
	assert.Equal(t, created.AuthID, updated.AuthID, "Auth ID should remain same")
	assert.Equal(t, "authorized", updated.AuthStatus, "Auth status should be updated")
	assert.Equal(t, "user-updated", *updated.UserID, "User ID should be updated")
	assert.Greater(t, updated.UpdatedTime, created.UpdatedTime, "Updated time should be greater")

	// Verify approved purpose details were updated
	assert.NotNil(t, updated.ApprovedPurposeDetails, "ApprovedPurposeDetails should not be nil")
	assert.Len(t, updated.ApprovedPurposeDetails.ApprovedPurposesNames, 3, "Should have 3 approved purposes")
	assert.Contains(t, updated.ApprovedPurposeDetails.ApprovedPurposesNames, "utility_read", "Should contain utility_read purpose")
	assert.Contains(t, updated.ApprovedPurposeDetails.ApprovedPurposesNames, "taxes_read", "Should contain taxes_read purpose")
	assert.Contains(t, updated.ApprovedPurposeDetails.ApprovedPurposesNames, "profile_read", "Should contain profile_read purpose")

	// Verify persistence by retrieving again
	t.Log("Verifying persistence...")
	retrieved, err := env.AuthResourceService.GetAuthResource(ctx, created.AuthID, testAuthResourceOrgID)
	require.NoError(t, err)

	assert.Equal(t, "authorized", retrieved.AuthStatus, "Persisted status should be updated")
	assert.Equal(t, "user-updated", *retrieved.UserID, "Persisted user ID should be updated")
	assert.NotNil(t, retrieved.ApprovedPurposeDetails, "Persisted ApprovedPurposeDetails should not be nil")
	assert.Len(t, retrieved.ApprovedPurposeDetails.ApprovedPurposesNames, 3, "Persisted purposes should have 3 items")

	t.Logf("✓ Successfully updated auth resource: %s", updated.AuthID)
}

// TestAuthResourceUpdate_PartialUpdate tests partial update of auth resource
func TestAuthResourceUpdate_PartialUpdate(t *testing.T) {
	env := setupTestEnvironment(t)
	ctx := context.Background()

	// Setup
	consentRequest := createTestConsentRequest()
	consent, err := env.ConsentService.CreateConsent(ctx, consentRequest, testAuthResourceClientID, testAuthResourceOrgID)
	require.NoError(t, err)
	defer cleanupTestData(t, env, consent.ConsentID)

	authResourceRequest := &models.ConsentAuthResourceCreateRequest{
		AuthType:   "user",
		UserID:     stringPtr("user-123"),
		AuthStatus: "pending",
		ApprovedPurposeDetails: &models.ApprovedPurposeDetails{
			ApprovedPurposesNames:       []string{"utility_read"},
			ApprovedAdditionalResources: []interface{}{},
		},
	}

	created, err := env.AuthResourceService.CreateAuthResource(ctx, consent.ConsentID, testAuthResourceOrgID, authResourceRequest)
	require.NoError(t, err)

	// Test: Update only status
	t.Log("Updating only auth status...")
	updateRequest := &models.ConsentAuthResourceUpdateRequest{
		AuthStatus: "active",
	}

	updated, err := env.AuthResourceService.UpdateAuthResource(ctx, created.AuthID, testAuthResourceOrgID, updateRequest)
	require.NoError(t, err)

	// Verify only status changed
	assert.Equal(t, "active", updated.AuthStatus, "Status should be updated")
	assert.Equal(t, created.UserID, updated.UserID, "User ID should remain unchanged")
	assert.NotNil(t, updated.ApprovedPurposeDetails, "ApprovedPurposeDetails should not be nil")
	assert.Equal(t, created.ApprovedPurposeDetails.ApprovedPurposesNames, updated.ApprovedPurposeDetails.ApprovedPurposesNames, "Purposes should remain unchanged")

	t.Log("✓ Partial update completed successfully")
}

// TestAuthResourceDelete_Success tests deleting an auth resource
func TestAuthResourceDelete_Success(t *testing.T) {
	env := setupTestEnvironment(t)
	ctx := context.Background()

	// Setup: Create consent and auth resource
	consentRequest := createTestConsentRequest()
	consent, err := env.ConsentService.CreateConsent(ctx, consentRequest, testAuthResourceClientID, testAuthResourceOrgID)
	require.NoError(t, err)
	defer cleanupTestData(t, env, consent.ConsentID)

	authResourceRequest := &models.ConsentAuthResourceCreateRequest{
		AuthType:   "user",
		UserID:     stringPtr("user-123"),
		AuthStatus: "pending",
	}

	created, err := env.AuthResourceService.CreateAuthResource(ctx, consent.ConsentID, testAuthResourceOrgID, authResourceRequest)
	require.NoError(t, err)

	// Test: Delete the auth resource
	t.Log("Deleting auth resource...")
	err = env.AuthResourceService.DeleteAuthResource(ctx, created.AuthID, testAuthResourceOrgID)
	require.NoError(t, err, "Should delete auth resource successfully")

	// Verify it's deleted
	t.Log("Verifying deletion...")
	_, err = env.AuthResourceService.GetAuthResource(ctx, created.AuthID, testAuthResourceOrgID)
	assert.Error(t, err, "Should fail to retrieve deleted auth resource")
	assert.Contains(t, err.Error(), "auth resource not found", "Error should mention auth resource not found")

	t.Logf("✓ Successfully deleted auth resource: %s", created.AuthID)
}

// TestAuthResourceDelete_NotFound tests deleting a non-existent auth resource
func TestAuthResourceDelete_NotFound(t *testing.T) {
	env := setupTestEnvironment(t)
	ctx := context.Background()

	err := env.AuthResourceService.DeleteAuthResource(ctx, "AUTH-nonexistent", testAuthResourceOrgID)
	assert.Error(t, err, "Should fail to delete non-existent auth resource")
	assert.Contains(t, err.Error(), "auth resource not found", "Error should mention auth resource not found")

	t.Log("✓ Correctly handled non-existent auth resource deletion")
}

// TestAuthResourceLifecycle_Complete tests the complete lifecycle of an auth resource
func TestAuthResourceLifecycle_Complete(t *testing.T) {
	env := setupTestEnvironment(t)
	ctx := context.Background()

	// Step 1: Create consent
	t.Log("Step 1: Creating consent...")
	consentRequest := createTestConsentRequest()
	consent, err := env.ConsentService.CreateConsent(ctx, consentRequest, testAuthResourceClientID, testAuthResourceOrgID)
	require.NoError(t, err)
	defer cleanupTestData(t, env, consent.ConsentID)

	// Step 2: Create auth resource
	t.Log("Step 2: Creating auth resource...")
	createRequest := &models.ConsentAuthResourceCreateRequest{
		AuthType:   "user",
		UserID:     stringPtr("user-lifecycle"),
		AuthStatus: "pending",
		ApprovedPurposeDetails: &models.ApprovedPurposeDetails{
			ApprovedPurposesNames:       []string{"utility_read"},
			ApprovedAdditionalResources: []interface{}{},
		},
	}

	created, err := env.AuthResourceService.CreateAuthResource(ctx, consent.ConsentID, testAuthResourceOrgID, createRequest)
	require.NoError(t, err)
	assert.Equal(t, "pending", created.AuthStatus)
	t.Logf("Created auth resource: %s", created.AuthID)

	// Step 3: Retrieve auth resource
	t.Log("Step 3: Retrieving auth resource...")
	retrieved, err := env.AuthResourceService.GetAuthResource(ctx, created.AuthID, testAuthResourceOrgID)
	require.NoError(t, err)
	assert.Equal(t, created.AuthID, retrieved.AuthID)

	// Step 4: Update auth resource
	t.Log("Step 4: Updating auth resource to authorized...")
	updateRequest := &models.ConsentAuthResourceUpdateRequest{
		AuthStatus: "authorized",
		ApprovedPurposeDetails: &models.ApprovedPurposeDetails{
			ApprovedPurposesNames:       []string{"utility_read", "taxes_read"},
			ApprovedAdditionalResources: []interface{}{},
		},
	}

	updated, err := env.AuthResourceService.UpdateAuthResource(ctx, created.AuthID, testAuthResourceOrgID, updateRequest)
	require.NoError(t, err)
	assert.Equal(t, "authorized", updated.AuthStatus)
	assert.NotNil(t, updated.ApprovedPurposeDetails, "ApprovedPurposeDetails should not be nil")
	assert.Len(t, updated.ApprovedPurposeDetails.ApprovedPurposesNames, 2, "Should have 2 approved purposes")
	assert.Contains(t, updated.ApprovedPurposeDetails.ApprovedPurposesNames, "taxes_read", "Should contain taxes_read purpose")

	// Step 5: Get all auth resources for consent
	t.Log("Step 5: Retrieving all auth resources for consent...")
	allAuthResources, err := env.AuthResourceService.GetAuthResourcesByConsentID(ctx, consent.ConsentID, testAuthResourceOrgID)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(allAuthResources.Data), 1, "Should have at least 1 auth resource")

	// Step 6: Delete auth resource
	t.Log("Step 6: Deleting auth resource...")
	err = env.AuthResourceService.DeleteAuthResource(ctx, created.AuthID, testAuthResourceOrgID)
	require.NoError(t, err)

	// Step 7: Verify deletion
	t.Log("Step 7: Verifying deletion...")
	_, err = env.AuthResourceService.GetAuthResource(ctx, created.AuthID, testAuthResourceOrgID)
	assert.Error(t, err)

	t.Log("✓ Complete auth resource lifecycle test passed")
}
