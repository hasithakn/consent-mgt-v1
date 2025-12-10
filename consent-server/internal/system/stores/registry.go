package stores

import (
	dbmodel "github.com/wso2/consent-management-api/internal/system/database/model"
	"github.com/wso2/consent-management-api/internal/system/database/provider"
)

// StoreRegistry holds references to all stores in the application
// Each store is held as interface{} to avoid circular dependencies
// Services type-assert to their needed store interfaces
type StoreRegistry struct {
	dbClient provider.DBClientInterface

	// Store instances - services will type-assert these to their specific interfaces
	Consent        interface{} // consent.consentStore
	AuthResource   interface{} // authresource.authResourceStore
	ConsentPurpose interface{} // consentpurpose.consentPurposeStore
}

// NewStoreRegistry creates a new store registry with all initialized stores
func NewStoreRegistry(
	dbClient provider.DBClientInterface,
	consentStore interface{},
	authResourceStore interface{},
	consentPurposeStore interface{},
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
	tx, err := r.dbClient.BeginTx()
	if err != nil {
		return err
	}

	for _, query := range queries {
		if err := query(tx); err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}
