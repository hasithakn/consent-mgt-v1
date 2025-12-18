package consent

import (
	"context"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/wso2/consent-management-api/internal/config"
	"github.com/wso2/consent-management-api/internal/dao"
	"github.com/wso2/consent-management-api/internal/database"
	"github.com/wso2/consent-management-api/internal/models"
	"github.com/wso2/consent-management-api/internal/router"
	"github.com/wso2/consent-management-api/internal/service"
)

// TestEnvironment sets up test environment for Consent API integration tests
type TestEnvironment struct {
	Router                *gin.Engine
	ConsentService        *service.ConsentService
	AuthResourceService   *service.AuthResourceService
	ConsentPurposeService *service.ConsentPurposeService
	ConsentDAO            *dao.ConsentDAO
	StatusAuditDAO        *dao.StatusAuditDAO
	AttributeDAO          *dao.ConsentAttributeDAO
	FileDAO               *dao.ConsentFileDAO
	AuthResourceDAO       *dao.AuthResourceDAO
	ConsentPurposeDAO     *dao.ConsentPurposeDAO
	ConsentPurposeAttrDAO *dao.ConsentPurposeAttributeDAO
}

// SetupTestEnvironment initializes the test environment with all necessary dependencies
func SetupTestEnvironment(t *testing.T) *TestEnvironment {
	// Load configuration from new location
	cfg, err := config.Load("../../../../consent-server/cmd/server/repository/conf/deployment.yaml")
	require.NoError(t, err, "Failed to load config")

	// Initialize logger
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// Initialize database
	db, err := database.Initialize(&cfg.Database.Consent, logger)
	require.NoError(t, err, "Failed to initialize database")

	// Create DAOs
	consentDAO := dao.NewConsentDAO(db)
	statusAuditDAO := dao.NewStatusAuditDAO(db)
	attributeDAO := dao.NewConsentAttributeDAO(db)
	fileDAO := dao.NewConsentFileDAO(db)
	authResourceDAO := dao.NewAuthResourceDAO(db)
	consentPurposeDAO := dao.NewConsentPurposeDAO(db.DB)
	consentPurposeAttrDAO := dao.NewConsentPurposeAttributeDAO(db.DB)

	// Create services
	consentService := service.NewConsentService(consentDAO, statusAuditDAO, attributeDAO, authResourceDAO, consentPurposeDAO, db, logger)
	authResourceService := service.NewAuthResourceService(authResourceDAO, consentDAO, db, logger)
	consentPurposeService := service.NewConsentPurposeService(consentPurposeDAO, consentPurposeAttrDAO, consentDAO, db.DB, logger)

	// Create router by reusing the application's router setup. Pass nil for extension client in tests.
	gin.SetMode(gin.TestMode)
	testRouter := router.SetupRouter(consentService, authResourceService, consentPurposeService, nil)

	return &TestEnvironment{
		Router:                testRouter,
		ConsentService:        consentService,
		AuthResourceService:   authResourceService,
		ConsentPurposeService: consentPurposeService,
		ConsentDAO:            consentDAO,
		StatusAuditDAO:        statusAuditDAO,
		AttributeDAO:          attributeDAO,
		FileDAO:               fileDAO,
		AuthResourceDAO:       authResourceDAO,
		ConsentPurposeDAO:     consentPurposeDAO,
		ConsentPurposeAttrDAO: consentPurposeAttrDAO,
	}
}

// CleanupTestData removes test consent data from the database
func CleanupTestData(t *testing.T, env *TestEnvironment, consentIDs ...string) {
	ctx := context.Background()
	for _, consentID := range consentIDs {
		err := env.ConsentDAO.Delete(ctx, consentID, "TEST_ORG")
		if err != nil {
			t.Logf("Warning: Failed to cleanup consent %s: %v", consentID, err)
		}
	}
}

// CreateTestPurpose creates a test consent purpose and returns it
func CreateTestPurpose(t *testing.T, env *TestEnvironment, name, description string) *models.ConsentPurpose {
	ctx := context.Background()

	purpose := &models.ConsentPurpose{
		ID:          "PURPOSE-test-" + name,
		Name:        name,
		Description: &description,
		Type:        "string", // Default type for test purposes
		OrgID:       "TEST_ORG",
	}

	err := env.ConsentPurposeDAO.Create(ctx, purpose)
	require.NoError(t, err, "Failed to create test purpose: %s", name)

	return purpose
}

// CleanupTestPurpose removes a test consent purpose from the database
func CleanupTestPurpose(t *testing.T, env *TestEnvironment, purposeID string) {
	ctx := context.Background()
	err := env.ConsentPurposeDAO.Delete(ctx, purposeID, "TEST_ORG")
	if err != nil {
		t.Logf("Warning: Failed to cleanup purpose %s: %v", purposeID, err)
	}
}

// CreateTestPurposes creates multiple test purposes at once
func CreateTestPurposes(t *testing.T, env *TestEnvironment, purposes map[string]string) map[string]*models.ConsentPurpose {
	result := make(map[string]*models.ConsentPurpose)
	for name, description := range purposes {
		result[name] = CreateTestPurpose(t, env, name, description)
	}
	return result
}

// CleanupTestPurposes removes multiple test purposes from the database
func CleanupTestPurposes(t *testing.T, env *TestEnvironment, purposes map[string]*models.ConsentPurpose) {
	for _, purpose := range purposes {
		CleanupTestPurpose(t, env, purpose.ID)
	}
}

// Int64Ptr returns a pointer to an int64 value
func Int64Ptr(v int64) *int64 {
	return &v
}

// IntPtr returns a pointer to an int value
func IntPtr(v int) *int {
	return &v
}

// BoolPtr returns a pointer to a bool value
func BoolPtr(v bool) *bool {
	return &v
}

// StringPtr returns a pointer to a string value
func StringPtr(v string) *string {
	return &v
}
