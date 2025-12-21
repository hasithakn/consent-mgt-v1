package stores

import (
	dbmodel "github.com/wso2/consent-management-api/internal/system/database/model"
	"github.com/wso2/consent-management-api/internal/system/database/provider"
	"github.com/wso2/consent-management-api/internal/system/log"
	"github.com/wso2/consent-management-api/internal/system/stores/interfaces"
)

// StoreRegistry holds references to all stores in the application
type StoreRegistry struct {
	dbClient provider.DBClientInterface

	// Store instances with typed interfaces
	Consent        interfaces.ConsentStore
	AuthResource   interfaces.AuthResourceStore
	ConsentPurpose interfaces.ConsentPurposeStore
}

// NewStoreRegistry creates a new store registry with all initialized stores
func NewStoreRegistry(
	dbClient provider.DBClientInterface,
	consentStore interfaces.ConsentStore,
	authResourceStore interfaces.AuthResourceStore,
	consentPurposeStore interfaces.ConsentPurposeStore,
) *StoreRegistry {
	return &StoreRegistry{
		dbClient:       dbClient,
		Consent:        consentStore,
		AuthResource:   authResourceStore,
		ConsentPurpose: consentPurposeStore,
	}
}

// ExecuteTransaction executes multiple store operations in a single transaction
// This follows Thunder's functional composition pattern
func (r *StoreRegistry) ExecuteTransaction(queries []func(tx dbmodel.TxInterface) error) error {
	logger := log.GetLogger()
	logger.Debug("Starting transaction", log.Int("query_count", len(queries)))

	tx, err := r.dbClient.BeginTx()
	if err != nil {
		logger.Error("Failed to begin transaction", log.Error(err))
		return err
	}

	for i, query := range queries {
		if err := query(tx); err != nil {
			logger.Warn("Transaction query failed, rolling back",
				log.Error(err),
				log.Int("failed_query_index", i),
			)
			tx.Rollback()
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		logger.Error("Failed to commit transaction", log.Error(err))
		return err
	}

	logger.Debug("Transaction committed successfully", log.Int("query_count", len(queries)))
	return nil
}
