package dao

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/wso2/consent-management-api/internal/database"
	"github.com/wso2/consent-management-api/internal/models"
)

// StatusAuditDAO handles database operations for consent status audit
type StatusAuditDAO struct {
	db *database.DB
}

// NewStatusAuditDAO creates a new StatusAuditDAO instance
func NewStatusAuditDAO(db *database.DB) *StatusAuditDAO {
	return &StatusAuditDAO{db: db}
}

// Create inserts a new status audit record
func (dao *StatusAuditDAO) Create(ctx context.Context, audit *models.ConsentStatusAudit) error {
	query := `
		INSERT INTO CONSENT_STATUS_AUDIT (
			STATUS_AUDIT_ID, CONSENT_ID, CURRENT_STATUS, ACTION_TIME,
			REASON, ACTION_BY, PREVIOUS_STATUS, ORG_ID
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := dao.db.ExecContext(
		ctx,
		query,
		audit.StatusAuditID,
		audit.ConsentID,
		audit.CurrentStatus,
		audit.ActionTime,
		audit.Reason,
		audit.ActionBy,
		audit.PreviousStatus,
		audit.OrgID,
	)

	if err != nil {
		return fmt.Errorf("failed to create status audit: %w", err)
	}

	return nil
}

// CreateWithTx inserts a new status audit record using a transaction
func (dao *StatusAuditDAO) CreateWithTx(ctx context.Context, tx *database.Transaction, audit *models.ConsentStatusAudit) error {
	query := `
		INSERT INTO CONSENT_STATUS_AUDIT (
			STATUS_AUDIT_ID, CONSENT_ID, CURRENT_STATUS, ACTION_TIME,
			REASON, ACTION_BY, PREVIOUS_STATUS, ORG_ID
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := tx.ExecContext(
		ctx,
		query,
		audit.StatusAuditID,
		audit.ConsentID,
		audit.CurrentStatus,
		audit.ActionTime,
		audit.Reason,
		audit.ActionBy,
		audit.PreviousStatus,
		audit.OrgID,
	)

	if err != nil {
		return fmt.Errorf("failed to create status audit with transaction: %w", err)
	}

	return nil
}

// GetByID retrieves a status audit record by ID
func (dao *StatusAuditDAO) GetByID(ctx context.Context, statusAuditID, orgID string) (*models.ConsentStatusAudit, error) {
	query := `
		SELECT STATUS_AUDIT_ID, CONSENT_ID, CURRENT_STATUS, ACTION_TIME,
		       REASON, ACTION_BY, PREVIOUS_STATUS, ORG_ID
		FROM CONSENT_STATUS_AUDIT
		WHERE STATUS_AUDIT_ID = ? AND ORG_ID = ?
	`

	var audit models.ConsentStatusAudit
	err := dao.db.GetContext(ctx, &audit, query, statusAuditID, orgID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("status audit not found: %s", statusAuditID)
		}
		return nil, fmt.Errorf("failed to get status audit: %w", err)
	}

	return &audit, nil
}

// GetByConsentID retrieves all status audit records for a specific consent
func (dao *StatusAuditDAO) GetByConsentID(ctx context.Context, consentID, orgID string) ([]models.ConsentStatusAudit, error) {
	query := `
		SELECT STATUS_AUDIT_ID, CONSENT_ID, CURRENT_STATUS, ACTION_TIME,
		       REASON, ACTION_BY, PREVIOUS_STATUS, ORG_ID
		FROM CONSENT_STATUS_AUDIT
		WHERE CONSENT_ID = ? AND ORG_ID = ?
		ORDER BY ACTION_TIME DESC
	`

	var audits []models.ConsentStatusAudit
	err := dao.db.SelectContext(ctx, &audits, query, consentID, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get status audits by consent ID: %w", err)
	}

	return audits, nil
}

// GetByConsentIDWithTx retrieves all status audit records for a specific consent using a transaction
func (dao *StatusAuditDAO) GetByConsentIDWithTx(ctx context.Context, tx *database.Transaction, consentID, orgID string) ([]models.ConsentStatusAudit, error) {
	query := `
		SELECT STATUS_AUDIT_ID, CONSENT_ID, CURRENT_STATUS, ACTION_TIME,
		       REASON, ACTION_BY, PREVIOUS_STATUS, ORG_ID
		FROM CONSENT_STATUS_AUDIT
		WHERE CONSENT_ID = ? AND ORG_ID = ?
		ORDER BY ACTION_TIME DESC
	`

	var audits []models.ConsentStatusAudit
	err := tx.SelectContext(ctx, &audits, query, consentID, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get status audits by consent ID: %w", err)
	}

	return audits, nil
}

// GetLatestByConsentID retrieves the most recent status audit for a consent
func (dao *StatusAuditDAO) GetLatestByConsentID(ctx context.Context, consentID, orgID string) (*models.ConsentStatusAudit, error) {
	query := `
		SELECT STATUS_AUDIT_ID, CONSENT_ID, CURRENT_STATUS, ACTION_TIME,
		       REASON, ACTION_BY, PREVIOUS_STATUS, ORG_ID
		FROM CONSENT_STATUS_AUDIT
		WHERE CONSENT_ID = ? AND ORG_ID = ?
		ORDER BY ACTION_TIME DESC
		LIMIT 1
	`

	var audit models.ConsentStatusAudit
	err := dao.db.GetContext(ctx, &audit, query, consentID, orgID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no status audit found for consent: %s", consentID)
		}
		return nil, fmt.Errorf("failed to get latest status audit: %w", err)
	}

	return &audit, nil
}

// Delete deletes a status audit record
func (dao *StatusAuditDAO) Delete(ctx context.Context, statusAuditID, orgID string) error {
	query := `DELETE FROM CONSENT_STATUS_AUDIT WHERE STATUS_AUDIT_ID = ? AND ORG_ID = ?`

	result, err := dao.db.ExecContext(ctx, query, statusAuditID, orgID)
	if err != nil {
		return fmt.Errorf("failed to delete status audit: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("status audit not found: %s", statusAuditID)
	}

	return nil
}

// DeleteByConsentID deletes all status audit records for a consent
func (dao *StatusAuditDAO) DeleteByConsentID(ctx context.Context, consentID, orgID string) error {
	query := `DELETE FROM CONSENT_STATUS_AUDIT WHERE CONSENT_ID = ? AND ORG_ID = ?`

	_, err := dao.db.ExecContext(ctx, query, consentID, orgID)
	if err != nil {
		return fmt.Errorf("failed to delete status audits by consent ID: %w", err)
	}

	return nil
}

// GetStatusHistory retrieves status change history within a time range
func (dao *StatusAuditDAO) GetStatusHistory(ctx context.Context, consentID, orgID string, fromTime, toTime int64) ([]models.ConsentStatusAudit, error) {
	query := `
		SELECT STATUS_AUDIT_ID, CONSENT_ID, CURRENT_STATUS, ACTION_TIME,
		       REASON, ACTION_BY, PREVIOUS_STATUS, ORG_ID
		FROM CONSENT_STATUS_AUDIT
		WHERE CONSENT_ID = ? AND ORG_ID = ? AND ACTION_TIME BETWEEN ? AND ?
		ORDER BY ACTION_TIME DESC
	`

	var audits []models.ConsentStatusAudit
	err := dao.db.SelectContext(ctx, &audits, query, consentID, orgID, fromTime, toTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get status history: %w", err)
	}

	return audits, nil
}
