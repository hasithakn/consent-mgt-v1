package consentpurpose

import (
	"net/http"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/wso2/consent-management-api/internal/config"
	"github.com/wso2/consent-management-api/internal/dao"
	"github.com/wso2/consent-management-api/internal/database"
	"github.com/wso2/consent-management-api/internal/router"
	"github.com/wso2/consent-management-api/internal/service"
)

// TestConsentPurposeAPIEnvironment holds test dependencies for API tests
type TestConsentPurposeAPIEnvironment struct {
	Router         http.Handler
	PurposeService *service.ConsentPurposeService
	PurposeDAO     *dao.ConsentPurposeDAO
	ConsentDAO     *dao.ConsentDAO
}

// setupConsentPurposeAPITestEnvironment initializes test environment for API tests
func setupConsentPurposeAPITestEnvironment(t *testing.T) *TestConsentPurposeAPIEnvironment {
	// Load configuration from new location
	cfg, err := config.Load("../../../cmd/server/repository/conf/deployment.yaml")
	require.NoError(t, err, "Failed to load config")

	// Initialize logger
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel) // Reduce noise in tests

	// Initialize database
	db, err := database.Initialize(&cfg.Database.Consent, logger)
	require.NoError(t, err, "Failed to initialize database")

	// Initialize DAOs
	consentDAO := dao.NewConsentDAO(db)
	statusAuditDAO := dao.NewStatusAuditDAO(db)
	attributeDAO := dao.NewConsentAttributeDAO(db)
	authResourceDAO := dao.NewAuthResourceDAO(db)
	purposeDAO := dao.NewConsentPurposeDAO(db.DB)
	purposeAttributeDAO := dao.NewConsentPurposeAttributeDAO(db.DB)

	// Initialize services
	consentService := service.NewConsentService(
		consentDAO,
		statusAuditDAO,
		attributeDAO,
		authResourceDAO,
		purposeDAO,
		db,
		logger,
	)

	authResourceService := service.NewAuthResourceService(
		authResourceDAO,
		consentDAO,
		db,
		logger,
	)

	purposeService := service.NewConsentPurposeService(
		purposeDAO,
		purposeAttributeDAO,
		consentDAO,
		db.DB,
		logger,
	)

	// Setup router (pass nil for extension client in tests)
	ginRouter := router.SetupRouter(consentService, authResourceService, purposeService, nil)

	return &TestConsentPurposeAPIEnvironment{
		Router:         ginRouter,
		PurposeService: purposeService,
		PurposeDAO:     purposeDAO,
		ConsentDAO:     consentDAO,
	}
}
