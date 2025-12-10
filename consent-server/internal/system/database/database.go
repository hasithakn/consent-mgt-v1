/*
 * Copyright (c) 2025, WSO2 LLC. (https://www.wso2.com).
 *
 * WSO2 LLC. licenses this file to you under the Apache License,
 * Version 2.0 (the "License"); you may not use this file except
 * in compliance with the License at
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

// Package database provides database connection management.
package database

import (
	"context"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/wso2/consent-management-api/internal/system/config"
	"github.com/wso2/consent-management-api/internal/system/log"
)

// DB holds the database connection.
type DB struct {
	*sqlx.DB
}

// Initialize creates and initializes the database connection.
func Initialize(cfg *config.DatabaseConfig) (*DB, error) {
	logger := log.GetLogger().With(log.String(log.LoggerKeyComponentName, "Database"))
	dsn := cfg.GetDSN()

	logger.Info("Connecting to database...",
		log.String("hostname", cfg.Hostname),
		log.Int("port", cfg.Port),
		log.String("database", cfg.Database))

	// Open database connection
	db, err := sqlx.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Info("Successfully connected to database")

	return &DB{DB: db}, nil
}

// Close closes the database connection.
func (db *DB) Close() error {
	if db.DB != nil {
		logger := log.GetLogger().With(log.String(log.LoggerKeyComponentName, "Database"))
		logger.Info("Closing database connection...")
		return db.DB.Close()
	}
	return nil
}

// HealthCheck checks if the database is healthy.
func (db *DB) HealthCheck(ctx context.Context) error {
	if db.DB == nil {
		return fmt.Errorf("database connection is not initialized")
	}

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	return nil
}

// Tx wraps sqlx.Tx to provide transaction management.
type Tx struct {
	*sqlx.Tx
}

// BeginTx starts a new transaction.
func (db *DB) BeginTx(ctx context.Context) (*Tx, error) {
	tx, err := db.DB.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	return &Tx{Tx: tx}, nil
}

// Commit commits the transaction.
func (tx *Tx) Commit() error {
	if err := tx.Tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

// Rollback rolls back the transaction.
func (tx *Tx) Rollback() error {
	if err := tx.Tx.Rollback(); err != nil {
		return fmt.Errorf("failed to rollback transaction: %w", err)
	}
	return nil
}
