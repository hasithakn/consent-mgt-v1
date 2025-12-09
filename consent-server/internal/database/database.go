package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
	"github.com/wso2/consent-management-api/internal/config"
)

// DB holds the database connection
type DB struct {
	*sqlx.DB
	logger *logrus.Logger
}

var dbInstance *DB

// Initialize creates and initializes the database connection
func Initialize(cfg *config.DatabaseConfig, logger *logrus.Logger) (*DB, error) {
	dsn := cfg.GetDSN()

	logger.WithFields(logrus.Fields{
		"hostname": cfg.Hostname,
		"port":     cfg.Port,
		"database": cfg.Database,
	}).Info("Connecting to database...")

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

	dbInstance = &DB{
		DB:     db,
		logger: logger,
	}

	return dbInstance, nil
}

// Get returns the global database instance
func Get() *DB {
	return dbInstance
}

// Close closes the database connection
func (db *DB) Close() error {
	if db.DB != nil {
		db.logger.Info("Closing database connection...")
		return db.DB.Close()
	}
	return nil
}

// HealthCheck checks if the database is healthy
func (db *DB) HealthCheck(ctx context.Context) error {
	if db.DB == nil {
		return fmt.Errorf("database connection is not initialized")
	}

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	return nil
}

// Transaction represents a database transaction
type Transaction struct {
	*sqlx.Tx
	logger *logrus.Logger
}

// BeginTx starts a new transaction
func (db *DB) BeginTx(ctx context.Context) (*Transaction, error) {
	tx, err := db.DB.BeginTxx(ctx, &sql.TxOptions{
		Isolation: sql.LevelReadCommitted,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	db.logger.Debug("Transaction started")

	return &Transaction{
		Tx:     tx,
		logger: db.logger,
	}, nil
}

// Commit commits the transaction
func (tx *Transaction) Commit() error {
	if err := tx.Tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	tx.logger.Debug("Transaction committed")
	return nil
}

// Rollback rolls back the transaction
func (tx *Transaction) Rollback() error {
	if err := tx.Tx.Rollback(); err != nil {
		if err == sql.ErrTxDone {
			// Transaction already completed
			return nil
		}
		return fmt.Errorf("failed to rollback transaction: %w", err)
	}

	tx.logger.Debug("Transaction rolled back")
	return nil
}

// WithTransaction executes a function within a transaction
// If the function returns an error, the transaction is rolled back
// Otherwise, it is committed
func (db *DB) WithTransaction(ctx context.Context, fn func(*Transaction) error) error {
	tx, err := db.BeginTx(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			// Rollback on panic
			_ = tx.Rollback()
			panic(p)
		}
	}()

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			db.logger.WithError(rbErr).Error("Failed to rollback transaction")
		}
		return err
	}

	return tx.Commit()
}

// Stats returns database statistics
func (db *DB) Stats() sql.DBStats {
	return db.DB.Stats()
}

// LogStats logs current database connection pool statistics
func (db *DB) LogStats() {
	stats := db.Stats()
	db.logger.WithFields(logrus.Fields{
		"open_connections":    stats.OpenConnections,
		"in_use":              stats.InUse,
		"idle":                stats.Idle,
		"wait_count":          stats.WaitCount,
		"wait_duration":       stats.WaitDuration,
		"max_idle_closed":     stats.MaxIdleClosed,
		"max_lifetime_closed": stats.MaxLifetimeClosed,
	}).Debug("Database connection pool stats")
}
