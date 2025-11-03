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
)

// TestConsentPurposeEnvironment holds test dependencies for consent purpose tests
type TestConsentPurposeEnvironment struct {
	PurposeDAO *dao.ConsentPurposeDAO
	ConsentDAO *dao.ConsentDAO
	DB         *database.DB
}

// setupConsentPurposeTestEnvironment initializes test environment for consent purpose tests
func setupConsentPurposeTestEnvironment(t *testing.T) *TestConsentPurposeEnvironment {
	// Load configuration
	cfg, err := config.Load("../../configs/config.yaml")
	require.NoError(t, err, "Failed to load config")

	// Initialize logger
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// Initialize database
	db, err := database.Initialize(&cfg.Database, logger)
	require.NoError(t, err, "Failed to initialize database")

	// Create DAOs
	purposeDAO := dao.NewConsentPurposeDAO(db.DB)
	consentDAO := dao.NewConsentDAO(db)

	return &TestConsentPurposeEnvironment{
		PurposeDAO: purposeDAO,
		ConsentDAO: consentDAO,
		DB:         db,
	}
}

// cleanupConsentPurposeTestData removes test data
func cleanupConsentPurposeTestData(t *testing.T, env *TestConsentPurposeEnvironment, purposeIDs []string, consentIDs []string) {
	ctx := context.Background()

	// Clean up purposes
	for _, purposeID := range purposeIDs {
		err := env.PurposeDAO.Delete(ctx, purposeID, "TEST_ORG")
		if err != nil {
			t.Logf("Warning: Failed to cleanup purpose %s: %v", purposeID, err)
		}
	}

	// Clean up consents
	for _, consentID := range consentIDs {
		err := env.ConsentDAO.Delete(ctx, consentID, "TEST_ORG")
		if err != nil {
			t.Logf("Warning: Failed to cleanup consent %s: %v", consentID, err)
		}
	}
}

// TestConsentPurposeCreate_Success tests successful purpose creation
func TestConsentPurposeCreate_Success(t *testing.T) {
	env := setupConsentPurposeTestEnvironment(t)
	ctx := context.Background()

	description := "Test purpose for data access"
	purpose := &models.ConsentPurpose{
		ID:          "PURPOSE-test-001",
		Name:        "Data Access",
		Description: &description,
		OrgID:       "TEST_ORG",
	}

	// Create purpose
	err := env.PurposeDAO.Create(ctx, purpose)
	require.NoError(t, err)

	// Verify creation
	retrieved, err := env.PurposeDAO.GetByID(ctx, purpose.ID, "TEST_ORG")
	require.NoError(t, err)
	assert.Equal(t, purpose.ID, retrieved.ID)
	assert.Equal(t, purpose.Name, retrieved.Name)
	assert.Equal(t, *purpose.Description, *retrieved.Description)
	assert.Equal(t, purpose.OrgID, retrieved.OrgID)

	t.Logf("✓ Successfully created and verified purpose: %s", purpose.ID)

	// Cleanup
	cleanupConsentPurposeTestData(t, env, []string{purpose.ID}, nil)
}

// TestConsentPurposeCreate_WithoutDescription tests purpose creation without description
func TestConsentPurposeCreate_WithoutDescription(t *testing.T) {
	env := setupConsentPurposeTestEnvironment(t)
	ctx := context.Background()

	purpose := &models.ConsentPurpose{
		ID:          "PURPOSE-test-002",
		Name:        "Marketing",
		Description: nil,
		OrgID:       "TEST_ORG",
	}

	// Create purpose
	err := env.PurposeDAO.Create(ctx, purpose)
	require.NoError(t, err)

	// Verify creation
	retrieved, err := env.PurposeDAO.GetByID(ctx, purpose.ID, "TEST_ORG")
	require.NoError(t, err)
	assert.Equal(t, purpose.ID, retrieved.ID)
	assert.Equal(t, purpose.Name, retrieved.Name)
	assert.Nil(t, retrieved.Description)

	t.Logf("✓ Successfully created purpose without description: %s", purpose.ID)

	// Cleanup
	cleanupConsentPurposeTestData(t, env, []string{purpose.ID}, nil)
}

// TestConsentPurposeGet_NotFound tests retrieving non-existent purpose
func TestConsentPurposeGet_NotFound(t *testing.T) {
	env := setupConsentPurposeTestEnvironment(t)
	ctx := context.Background()

	// Try to get non-existent purpose
	_, err := env.PurposeDAO.GetByID(ctx, "PURPOSE-nonexistent", "TEST_ORG")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	t.Log("✓ Correctly handled non-existent purpose retrieval")
}

// TestConsentPurposeList_Success tests listing purposes
func TestConsentPurposeList_Success(t *testing.T) {
	env := setupConsentPurposeTestEnvironment(t)
	ctx := context.Background()

	// Create multiple purposes
	description1 := "Purpose for analytics"
	description2 := "Purpose for marketing"
	purposes := []*models.ConsentPurpose{
		{
			ID:          "PURPOSE-test-list-001",
			Name:        "Analytics",
			Description: &description1,
			OrgID:       "TEST_ORG",
		},
		{
			ID:          "PURPOSE-test-list-002",
			Name:        "Marketing",
			Description: &description2,
			OrgID:       "TEST_ORG",
		},
		{
			ID:          "PURPOSE-test-list-003",
			Name:        "Customer Service",
			Description: nil,
			OrgID:       "TEST_ORG",
		},
	}

	for _, purpose := range purposes {
		err := env.PurposeDAO.Create(ctx, purpose)
		require.NoError(t, err)
	}

	// List purposes
	retrieved, total, err := env.PurposeDAO.List(ctx, "TEST_ORG", 10, 0)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(retrieved), 3, "Should have at least 3 purposes")
	assert.GreaterOrEqual(t, total, 3, "Total should be at least 3")

	t.Logf("✓ Successfully listed %d purposes (total: %d)", len(retrieved), total)

	// Cleanup
	cleanupConsentPurposeTestData(t, env, []string{
		"PURPOSE-test-list-001",
		"PURPOSE-test-list-002",
		"PURPOSE-test-list-003",
	}, nil)
}

// TestConsentPurposeList_Pagination tests pagination
func TestConsentPurposeList_Pagination(t *testing.T) {
	env := setupConsentPurposeTestEnvironment(t)
	ctx := context.Background()

	// Create 5 purposes
	purposeIDs := make([]string, 5)
	for i := 0; i < 5; i++ {
		purposeID := "PURPOSE-test-page-" + string(rune('A'+i))
		purposeIDs[i] = purposeID
		description := "Description " + string(rune('A'+i))
		purpose := &models.ConsentPurpose{
			ID:          purposeID,
			Name:        "Purpose " + string(rune('A'+i)),
			Description: &description,
			OrgID:       "TEST_ORG",
		}
		err := env.PurposeDAO.Create(ctx, purpose)
		require.NoError(t, err)
	}

	// Test pagination - first page
	page1, total, err := env.PurposeDAO.List(ctx, "TEST_ORG", 2, 0)
	require.NoError(t, err)
	assert.LessOrEqual(t, len(page1), 2)
	assert.GreaterOrEqual(t, total, 5)

	// Test pagination - second page
	page2, _, err := env.PurposeDAO.List(ctx, "TEST_ORG", 2, 2)
	require.NoError(t, err)
	assert.LessOrEqual(t, len(page2), 2)

	t.Logf("✓ Successfully tested pagination (Page 1: %d items, Page 2: %d items, Total: %d)",
		len(page1), len(page2), total)

	// Cleanup
	cleanupConsentPurposeTestData(t, env, purposeIDs, nil)
}

// TestConsentPurposeUpdate_Success tests successful purpose update
func TestConsentPurposeUpdate_Success(t *testing.T) {
	env := setupConsentPurposeTestEnvironment(t)
	ctx := context.Background()

	description := "Original description"
	purpose := &models.ConsentPurpose{
		ID:          "PURPOSE-test-update-001",
		Name:        "Original Name",
		Description: &description,
		OrgID:       "TEST_ORG",
	}

	// Create purpose
	err := env.PurposeDAO.Create(ctx, purpose)
	require.NoError(t, err)

	// Update purpose
	updatedDescription := "Updated description"
	purpose.Name = "Updated Name"
	purpose.Description = &updatedDescription

	err = env.PurposeDAO.Update(ctx, purpose)
	require.NoError(t, err)

	// Verify update
	retrieved, err := env.PurposeDAO.GetByID(ctx, purpose.ID, "TEST_ORG")
	require.NoError(t, err)
	assert.Equal(t, "Updated Name", retrieved.Name)
	assert.Equal(t, "Updated description", *retrieved.Description)

	t.Logf("✓ Successfully updated purpose: %s", purpose.ID)

	// Cleanup
	cleanupConsentPurposeTestData(t, env, []string{purpose.ID}, nil)
}

// TestConsentPurposeDelete_Success tests successful purpose deletion
func TestConsentPurposeDelete_Success(t *testing.T) {
	env := setupConsentPurposeTestEnvironment(t)
	ctx := context.Background()

	description := "Purpose to be deleted"
	purpose := &models.ConsentPurpose{
		ID:          "PURPOSE-test-delete-001",
		Name:        "Delete Me",
		Description: &description,
		OrgID:       "TEST_ORG",
	}

	// Create purpose
	err := env.PurposeDAO.Create(ctx, purpose)
	require.NoError(t, err)

	// Delete purpose
	err = env.PurposeDAO.Delete(ctx, purpose.ID, "TEST_ORG")
	require.NoError(t, err)

	// Verify deletion
	_, err = env.PurposeDAO.GetByID(ctx, purpose.ID, "TEST_ORG")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	t.Logf("✓ Successfully deleted purpose: %s", purpose.ID)
}

// TestConsentPurposeMapping_LinkAndUnlink tests linking and unlinking purposes to consents
func TestConsentPurposeMapping_LinkAndUnlink(t *testing.T) {
	env := setupConsentPurposeTestEnvironment(t)
	ctx := context.Background()

	// Create a consent first
	validityTime := int64(7776000)
	frequency := 1
	recurringIndicator := false
	receipt, _ := json.Marshal(map[string]interface{}{"test": "data"})

	consent := &models.Consent{
		ConsentID:          "CONSENT-test-mapping-001",
		ClientID:           "TEST_CLIENT",
		OrgID:              "TEST_ORG",
		ConsentType:        "accounts",
		CurrentStatus:      "awaitingAuthorization",
		ValidityTime:       &validityTime,
		RecurringIndicator: &recurringIndicator,
		ConsentFrequency:   &frequency,
		ConsentPurposes:            models.JSON(receipt),
	}

	err := env.ConsentDAO.Create(ctx, consent)
	require.NoError(t, err)

	// Create purposes
	description1 := "Analytics purpose"
	description2 := "Marketing purpose"
	purpose1 := &models.ConsentPurpose{
		ID:          "PURPOSE-test-mapping-001",
		Name:        "Analytics",
		Description: &description1,
		OrgID:       "TEST_ORG",
	}
	purpose2 := &models.ConsentPurpose{
		ID:          "PURPOSE-test-mapping-002",
		Name:        "Marketing",
		Description: &description2,
		OrgID:       "TEST_ORG",
	}

	err = env.PurposeDAO.Create(ctx, purpose1)
	require.NoError(t, err)
	err = env.PurposeDAO.Create(ctx, purpose2)
	require.NoError(t, err)

	// Link purposes to consent
	err = env.PurposeDAO.LinkPurposeToConsent(ctx, consent.ConsentID, purpose1.ID, "TEST_ORG")
	require.NoError(t, err)
	err = env.PurposeDAO.LinkPurposeToConsent(ctx, consent.ConsentID, purpose2.ID, "TEST_ORG")
	require.NoError(t, err)

	// Verify links
	purposes, err := env.PurposeDAO.GetByConsentID(ctx, consent.ConsentID, "TEST_ORG")
	require.NoError(t, err)
	assert.Equal(t, 2, len(purposes))

	t.Logf("✓ Successfully linked 2 purposes to consent: %s", consent.ConsentID)

	// Unlink one purpose
	err = env.PurposeDAO.UnlinkPurposeFromConsent(ctx, consent.ConsentID, purpose1.ID, "TEST_ORG")
	require.NoError(t, err)

	// Verify after unlink
	purposes, err = env.PurposeDAO.GetByConsentID(ctx, consent.ConsentID, "TEST_ORG")
	require.NoError(t, err)
	assert.Equal(t, 1, len(purposes))
	assert.Equal(t, purpose2.ID, purposes[0].ID)

	t.Logf("✓ Successfully unlinked 1 purpose from consent")

	// Cleanup
	cleanupConsentPurposeTestData(t, env,
		[]string{purpose1.ID, purpose2.ID},
		[]string{consent.ConsentID})
}

// TestConsentPurposeMapping_CascadeDelete tests cascade delete behavior
func TestConsentPurposeMapping_CascadeDelete(t *testing.T) {
	env := setupConsentPurposeTestEnvironment(t)
	ctx := context.Background()

	// Create a consent
	validityTime := int64(7776000)
	frequency := 1
	recurringIndicator := false
	receipt, _ := json.Marshal(map[string]interface{}{"test": "cascade"})

	consent := &models.Consent{
		ConsentID:          "CONSENT-test-cascade-001",
		ClientID:           "TEST_CLIENT",
		OrgID:              "TEST_ORG",
		ConsentType:        "accounts",
		CurrentStatus:      "awaitingAuthorization",
		ValidityTime:       &validityTime,
		RecurringIndicator: &recurringIndicator,
		ConsentFrequency:   &frequency,
		ConsentPurposes:            models.JSON(receipt),
	}

	err := env.ConsentDAO.Create(ctx, consent)
	require.NoError(t, err)

	// Create purpose
	description := "Test cascade delete"
	purpose := &models.ConsentPurpose{
		ID:          "PURPOSE-test-cascade-001",
		Name:        "Cascade Test",
		Description: &description,
		OrgID:       "TEST_ORG",
	}

	err = env.PurposeDAO.Create(ctx, purpose)
	require.NoError(t, err)

	// Link purpose to consent
	err = env.PurposeDAO.LinkPurposeToConsent(ctx, consent.ConsentID, purpose.ID, "TEST_ORG")
	require.NoError(t, err)

	// Delete consent (should cascade delete the mapping)
	err = env.ConsentDAO.Delete(ctx, consent.ConsentID, "TEST_ORG")
	require.NoError(t, err)

	// Verify mapping is deleted but purpose still exists
	purposes, err := env.PurposeDAO.GetByConsentID(ctx, consent.ConsentID, "TEST_ORG")
	require.NoError(t, err)
	assert.Equal(t, 0, len(purposes), "Mapping should be deleted")

	// Purpose should still exist
	retrievedPurpose, err := env.PurposeDAO.GetByID(ctx, purpose.ID, "TEST_ORG")
	require.NoError(t, err)
	assert.Equal(t, purpose.ID, retrievedPurpose.ID)

	t.Log("✓ Cascade delete works correctly - mapping deleted, purpose retained")

	// Cleanup
	cleanupConsentPurposeTestData(t, env, []string{purpose.ID}, nil)
}

// TestConsentPurposeOrgScope tests organization scoping
func TestConsentPurposeOrgScope(t *testing.T) {
	env := setupConsentPurposeTestEnvironment(t)
	ctx := context.Background()

	description := "Org scoping test"

	// Create multiple purposes for TEST_ORG
	purpose1 := &models.ConsentPurpose{
		ID:          "PURPOSE-test-scope-001",
		Name:        "Purpose 1",
		Description: &description,
		OrgID:       "TEST_ORG",
	}

	purpose2 := &models.ConsentPurpose{
		ID:          "PURPOSE-test-scope-002",
		Name:        "Purpose 2",
		Description: &description,
		OrgID:       "TEST_ORG",
	}

	err := env.PurposeDAO.Create(ctx, purpose1)
	require.NoError(t, err)
	err = env.PurposeDAO.Create(ctx, purpose2)
	require.NoError(t, err)

	// Retrieve purposes
	retrieved1, err := env.PurposeDAO.GetByID(ctx, purpose1.ID, "TEST_ORG")
	require.NoError(t, err)
	assert.Equal(t, "Purpose 1", retrieved1.Name)
	assert.Equal(t, "TEST_ORG", retrieved1.OrgID)

	retrieved2, err := env.PurposeDAO.GetByID(ctx, purpose2.ID, "TEST_ORG")
	require.NoError(t, err)
	assert.Equal(t, "Purpose 2", retrieved2.Name)
	assert.Equal(t, "TEST_ORG", retrieved2.OrgID)

	// List purposes for the org
	purposes, total, err := env.PurposeDAO.List(ctx, "TEST_ORG", 100, 0)
	require.NoError(t, err)

	// Verify that all returned purposes belong to TEST_ORG
	foundPurpose1 := false
	foundPurpose2 := false
	for _, p := range purposes {
		assert.Equal(t, "TEST_ORG", p.OrgID, "All purposes should belong to TEST_ORG")
		if p.ID == purpose1.ID {
			foundPurpose1 = true
		}
		if p.ID == purpose2.ID {
			foundPurpose2 = true
		}
	}
	assert.True(t, foundPurpose1, "Should find purpose1 in list")
	assert.True(t, foundPurpose2, "Should find purpose2 in list")
	assert.GreaterOrEqual(t, total, 2, "Should have at least 2 purposes for TEST_ORG")

	t.Log("✓ Organization scoping works correctly")

	// Cleanup
	cleanupConsentPurposeTestData(t, env, []string{purpose1.ID, purpose2.ID}, nil)
} // TestConsentPurposeLifecycle_Complete tests complete purpose lifecycle
func TestConsentPurposeLifecycle_Complete(t *testing.T) {
	env := setupConsentPurposeTestEnvironment(t)
	ctx := context.Background()

	t.Log("Step 1: Create purpose")
	description := "Lifecycle test purpose"
	purpose := &models.ConsentPurpose{
		ID:          "PURPOSE-test-lifecycle-001",
		Name:        "Lifecycle Purpose",
		Description: &description,
		OrgID:       "TEST_ORG",
	}
	err := env.PurposeDAO.Create(ctx, purpose)
	require.NoError(t, err)
	t.Logf("Created purpose: %s", purpose.ID)

	t.Log("Step 2: Retrieve purpose")
	retrieved, err := env.PurposeDAO.GetByID(ctx, purpose.ID, "TEST_ORG")
	require.NoError(t, err)
	assert.Equal(t, purpose.Name, retrieved.Name)

	t.Log("Step 3: Create consent")
	validityTime := int64(7776000)
	frequency := 1
	recurringIndicator := false
	receipt, _ := json.Marshal(map[string]interface{}{"test": "lifecycle"})
	consent := &models.Consent{
		ConsentID:          "CONSENT-test-lifecycle-001",
		ClientID:           "TEST_CLIENT",
		OrgID:              "TEST_ORG",
		ConsentType:        "accounts",
		CurrentStatus:      "awaitingAuthorization",
		ValidityTime:       &validityTime,
		RecurringIndicator: &recurringIndicator,
		ConsentFrequency:   &frequency,
		ConsentPurposes:            models.JSON(receipt),
	}
	err = env.ConsentDAO.Create(ctx, consent)
	require.NoError(t, err)

	t.Log("Step 4: Link purpose to consent")
	err = env.PurposeDAO.LinkPurposeToConsent(ctx, consent.ConsentID, purpose.ID, "TEST_ORG")
	require.NoError(t, err)

	t.Log("Step 5: Verify link")
	purposes, err := env.PurposeDAO.GetByConsentID(ctx, consent.ConsentID, "TEST_ORG")
	require.NoError(t, err)
	assert.Equal(t, 1, len(purposes))

	t.Log("Step 6: Update purpose")
	updatedDescription := "Updated lifecycle description"
	purpose.Name = "Updated Lifecycle Purpose"
	purpose.Description = &updatedDescription
	err = env.PurposeDAO.Update(ctx, purpose)
	require.NoError(t, err)

	t.Log("Step 7: Verify update")
	retrieved, err = env.PurposeDAO.GetByID(ctx, purpose.ID, "TEST_ORG")
	require.NoError(t, err)
	assert.Equal(t, "Updated Lifecycle Purpose", retrieved.Name)

	t.Log("Step 8: Unlink purpose")
	err = env.PurposeDAO.UnlinkPurposeFromConsent(ctx, consent.ConsentID, purpose.ID, "TEST_ORG")
	require.NoError(t, err)

	t.Log("Step 9: Delete purpose")
	err = env.PurposeDAO.Delete(ctx, purpose.ID, "TEST_ORG")
	require.NoError(t, err)

	t.Log("Step 10: Verify deletion")
	_, err = env.PurposeDAO.GetByID(ctx, purpose.ID, "TEST_ORG")
	assert.Error(t, err)

	t.Log("✓ Complete lifecycle test passed")

	// Cleanup
	cleanupConsentPurposeTestData(t, env, nil, []string{consent.ConsentID})
}
