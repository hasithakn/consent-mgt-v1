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

// Package provider provides functionality for managing database connections and clients.
package provider

import (
	"fmt"
	"sync"

	"github.com/wso2/consent-management-api/internal/system/database"
	"github.com/wso2/consent-management-api/internal/system/log"
)

// DBProviderInterface defines the interface for getting database clients.
type DBProviderInterface interface {
	GetConsentDBClient() (DBClientInterface, error)
}

// DBProviderCloser is a separate interface for closing the provider.
// Only the lifecycle manager should use this interface.
type DBProviderCloser interface {
	Close() error
}

// dbProvider is the implementation of DBProviderInterface.
type dbProvider struct {
	consentClient DBClientInterface
	consentMutex  sync.RWMutex
	db            *database.DB
}

var (
	instance *dbProvider
	once     sync.Once
)

// InitDBProvider initializes the singleton instance of DBProvider with the database connection.
func InitDBProvider(db *database.DB) {
	once.Do(func() {
		instance = &dbProvider{
			db: db,
		}
		instance.initializeClient()
	})
}

// GetDBProvider returns the instance of DBProvider.
func GetDBProvider() DBProviderInterface {
	if instance == nil {
		panic("DBProvider not initialized. Call InitDBProvider first.")
	}
	return instance
}

// GetDBProviderCloser returns the DBProvider with closing capability.
// This should only be called from the main lifecycle manager.
func GetDBProviderCloser() DBProviderCloser {
	if instance == nil {
		panic("DBProvider not initialized. Call InitDBProvider first.")
	}
	return instance
}

// GetConsentDBClient returns a database client for consent datasource.
// Not required to close the returned client manually since it manages its own connection pool.
func (d *dbProvider) GetConsentDBClient() (DBClientInterface, error) {
	d.consentMutex.RLock()
	if d.consentClient != nil {
		defer d.consentMutex.RUnlock()
		return d.consentClient, nil
	}
	d.consentMutex.RUnlock()

	// Initialize client if not already done
	d.consentMutex.Lock()
	defer d.consentMutex.Unlock()

	// Double-check after acquiring write lock
	if d.consentClient != nil {
		return d.consentClient, nil
	}

	return d.consentClient, nil
}

// initializeClient initializes the database client.
func (d *dbProvider) initializeClient() {
	d.consentMutex.Lock()
	defer d.consentMutex.Unlock()

	logger := log.GetLogger().With(log.String(log.LoggerKeyComponentName, "DBProvider"))

	if d.db == nil {
		logger.Fatal("Database connection is nil")
		return
	}

	d.consentClient = NewDBClient(d.db.DB, "mysql")
	logger.Debug("Consent DB client initialized")
}

// Close closes the database connections. This should only be called by the lifecycle manager during shutdown.
func (d *dbProvider) Close() error {
	logger := log.GetLogger().With(log.String(log.LoggerKeyComponentName, "DBProvider"))
	logger.Debug("Closing database connections")

	return d.closeClient(&d.consentClient, &d.consentMutex, "consent")
}

// closeClient is a helper to close a DB client with locking.
func (d *dbProvider) closeClient(clientPtr *DBClientInterface, mutex *sync.RWMutex, clientName string) error {
	mutex.Lock()
	defer mutex.Unlock()

	logger := log.GetLogger().With(log.String(log.LoggerKeyComponentName, "DBProvider"))

	if *clientPtr != nil {
		// For now, we just set the client to nil since the underlying DB connection
		// is managed by the database.DB instance which has its own Close() method.
		// In the future, if DBClient needs specific cleanup, implement a close() method.
		*clientPtr = nil
		logger.Debug("DB client closed", log.String("client", clientName))
	}
	return nil
}

// close is a helper method to close the underlying database connection.
// This delegates to the database.DB Close() method.
func (d *dbProvider) closeDB() error {
	if d.db != nil {
		if err := d.db.Close(); err != nil {
			return fmt.Errorf("failed to close database: %w", err)
		}
	}
	return nil
}
