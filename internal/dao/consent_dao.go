package dao

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/wso2/consent-management-api/internal/database"
	"github.com/wso2/consent-management-api/internal/models"
)

// ConsentDAO handles database operations for consents
type ConsentDAO struct {
	db *database.DB
}

// NewConsentDAO creates a new ConsentDAO instance
func NewConsentDAO(db *database.DB) *ConsentDAO {
	return &ConsentDAO{db: db}
}

// Create inserts a new consent into the database
func (dao *ConsentDAO) Create(ctx context.Context, consent *models.Consent) error {
	query := `
		INSERT INTO FS_CONSENT (
			CONSENT_ID, RECEIPT, CREATED_TIME, UPDATED_TIME, CLIENT_ID,
			CONSENT_TYPE, CURRENT_STATUS, CONSENT_FREQUENCY, VALIDITY_TIME,
			RECURRING_INDICATOR, ORG_ID
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := dao.db.ExecContext(
		ctx,
		query,
		consent.ConsentID,
		consent.Receipt,
		consent.CreatedTime,
		consent.UpdatedTime,
		consent.ClientID,
		consent.ConsentType,
		consent.CurrentStatus,
		consent.ConsentFrequency,
		consent.ValidityTime,
		consent.RecurringIndicator,
		consent.OrgID,
	)

	if err != nil {
		return fmt.Errorf("failed to create consent: %w", err)
	}

	return nil
}

// CreateWithTx inserts a new consent using a transaction
func (dao *ConsentDAO) CreateWithTx(ctx context.Context, tx *database.Transaction, consent *models.Consent) error {
	query := `
		INSERT INTO FS_CONSENT (
			CONSENT_ID, RECEIPT, CREATED_TIME, UPDATED_TIME, CLIENT_ID,
			CONSENT_TYPE, CURRENT_STATUS, CONSENT_FREQUENCY, VALIDITY_TIME,
			RECURRING_INDICATOR, ORG_ID
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := tx.ExecContext(
		ctx,
		query,
		consent.ConsentID,
		consent.Receipt,
		consent.CreatedTime,
		consent.UpdatedTime,
		consent.ClientID,
		consent.ConsentType,
		consent.CurrentStatus,
		consent.ConsentFrequency,
		consent.ValidityTime,
		consent.RecurringIndicator,
		consent.OrgID,
	)

	if err != nil {
		return fmt.Errorf("failed to create consent with transaction: %w", err)
	}

	return nil
}

// GetByID retrieves a consent by ID and organization ID
func (dao *ConsentDAO) GetByID(ctx context.Context, consentID, orgID string) (*models.Consent, error) {
	query := `
		SELECT CONSENT_ID, RECEIPT, CREATED_TIME, UPDATED_TIME, CLIENT_ID,
		       CONSENT_TYPE, CURRENT_STATUS, CONSENT_FREQUENCY, VALIDITY_TIME,
		       RECURRING_INDICATOR, ORG_ID
		FROM FS_CONSENT
		WHERE CONSENT_ID = ? AND ORG_ID = ?
	`

	var consent models.Consent
	err := dao.db.GetContext(ctx, &consent, query, consentID, orgID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("consent not found: %s", consentID)
		}
		return nil, fmt.Errorf("failed to get consent: %w", err)
	}

	return &consent, nil
}

// GetByIDWithTx retrieves a consent by ID using a transaction
func (dao *ConsentDAO) GetByIDWithTx(ctx context.Context, tx *database.Transaction, consentID, orgID string) (*models.Consent, error) {
	query := `
		SELECT CONSENT_ID, RECEIPT, CREATED_TIME, UPDATED_TIME, CLIENT_ID,
		       CONSENT_TYPE, CURRENT_STATUS, CONSENT_FREQUENCY, VALIDITY_TIME,
		       RECURRING_INDICATOR, ORG_ID
		FROM FS_CONSENT
		WHERE CONSENT_ID = ? AND ORG_ID = ?
	`

	var consent models.Consent
	err := tx.GetContext(ctx, &consent, query, consentID, orgID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("consent not found: %s", consentID)
		}
		return nil, fmt.Errorf("failed to get consent: %w", err)
	}

	return &consent, nil
}

// Update updates an existing consent
func (dao *ConsentDAO) Update(ctx context.Context, consent *models.Consent) error {
	query := `
		UPDATE FS_CONSENT
		SET RECEIPT = ?, UPDATED_TIME = ?, CURRENT_STATUS = ?,
		    CONSENT_FREQUENCY = ?, VALIDITY_TIME = ?, RECURRING_INDICATOR = ?
		WHERE CONSENT_ID = ? AND ORG_ID = ?
	`

	result, err := dao.db.ExecContext(
		ctx,
		query,
		consent.Receipt,
		consent.UpdatedTime,
		consent.CurrentStatus,
		consent.ConsentFrequency,
		consent.ValidityTime,
		consent.RecurringIndicator,
		consent.ConsentID,
		consent.OrgID,
	)

	if err != nil {
		return fmt.Errorf("failed to update consent: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("consent not found: %s", consent.ConsentID)
	}

	return nil
}

// UpdateWithTx updates an existing consent using a transaction
func (dao *ConsentDAO) UpdateWithTx(ctx context.Context, tx *database.Transaction, consent *models.Consent) error {
	query := `
		UPDATE FS_CONSENT
		SET RECEIPT = ?, UPDATED_TIME = ?, CURRENT_STATUS = ?,
		    CONSENT_FREQUENCY = ?, VALIDITY_TIME = ?, RECURRING_INDICATOR = ?
		WHERE CONSENT_ID = ? AND ORG_ID = ?
	`

	result, err := tx.ExecContext(
		ctx,
		query,
		consent.Receipt,
		consent.UpdatedTime,
		consent.CurrentStatus,
		consent.ConsentFrequency,
		consent.ValidityTime,
		consent.RecurringIndicator,
		consent.ConsentID,
		consent.OrgID,
	)

	if err != nil {
		return fmt.Errorf("failed to update consent with transaction: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("consent not found: %s", consent.ConsentID)
	}

	return nil
}

// UpdateStatus updates only the status of a consent
func (dao *ConsentDAO) UpdateStatus(ctx context.Context, consentID, orgID, status string, updatedTime int64) error {
	query := `
		UPDATE FS_CONSENT
		SET CURRENT_STATUS = ?, UPDATED_TIME = ?
		WHERE CONSENT_ID = ? AND ORG_ID = ?
	`

	result, err := dao.db.ExecContext(ctx, query, status, updatedTime, consentID, orgID)
	if err != nil {
		return fmt.Errorf("failed to update consent status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("consent not found: %s", consentID)
	}

	return nil
}

// UpdateStatusWithTx updates only the status of a consent using a transaction
func (dao *ConsentDAO) UpdateStatusWithTx(ctx context.Context, tx *database.Transaction, consentID, orgID, status string, updatedTime int64) error {
	query := `
		UPDATE FS_CONSENT
		SET CURRENT_STATUS = ?, UPDATED_TIME = ?
		WHERE CONSENT_ID = ? AND ORG_ID = ?
	`

	result, err := tx.ExecContext(ctx, query, status, updatedTime, consentID, orgID)
	if err != nil {
		return fmt.Errorf("failed to update consent status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("consent not found: %s", consentID)
	}

	return nil
}

// Delete deletes a consent (soft delete by updating status or hard delete)
func (dao *ConsentDAO) Delete(ctx context.Context, consentID, orgID string) error {
	query := `DELETE FROM FS_CONSENT WHERE CONSENT_ID = ? AND ORG_ID = ?`

	result, err := dao.db.ExecContext(ctx, query, consentID, orgID)
	if err != nil {
		return fmt.Errorf("failed to delete consent: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("consent not found: %s", consentID)
	}

	return nil
}

// Search searches for consents based on provided parameters
func (dao *ConsentDAO) Search(ctx context.Context, params *models.ConsentSearchParams) ([]models.Consent, int, error) {
	// Build the WHERE clause dynamically
	var conditions []string
	var args []interface{}

	// Always filter by organization
	conditions = append(conditions, "ORG_ID = ?")
	args = append(args, params.OrgID)

	// Add consent IDs filter
	if len(params.ConsentIDs) > 0 {
		placeholders := strings.Repeat("?,", len(params.ConsentIDs)-1) + "?"
		conditions = append(conditions, fmt.Sprintf("CONSENT_ID IN (%s)", placeholders))
		for _, id := range params.ConsentIDs {
			args = append(args, id)
		}
	}

	// Add client IDs filter
	if len(params.ClientIDs) > 0 {
		placeholders := strings.Repeat("?,", len(params.ClientIDs)-1) + "?"
		conditions = append(conditions, fmt.Sprintf("CLIENT_ID IN (%s)", placeholders))
		for _, id := range params.ClientIDs {
			args = append(args, id)
		}
	}

	// Add consent types filter
	if len(params.ConsentTypes) > 0 {
		placeholders := strings.Repeat("?,", len(params.ConsentTypes)-1) + "?"
		conditions = append(conditions, fmt.Sprintf("CONSENT_TYPE IN (%s)", placeholders))
		for _, t := range params.ConsentTypes {
			args = append(args, t)
		}
	}

	// Add consent statuses filter
	if len(params.ConsentStatuses) > 0 {
		placeholders := strings.Repeat("?,", len(params.ConsentStatuses)-1) + "?"
		conditions = append(conditions, fmt.Sprintf("CURRENT_STATUS IN (%s)", placeholders))
		for _, s := range params.ConsentStatuses {
			args = append(args, s)
		}
	}

	// Add user IDs filter (requires join with auth resource table)
	var joinClause string
	if len(params.UserIDs) > 0 {
		joinClause = " INNER JOIN FS_CONSENT_AUTH_RESOURCE ar ON c.CONSENT_ID = ar.CONSENT_ID AND c.ORG_ID = ar.ORG_ID"
		placeholders := strings.Repeat("?,", len(params.UserIDs)-1) + "?"
		conditions = append(conditions, fmt.Sprintf("ar.USER_ID IN (%s)", placeholders))
		for _, uid := range params.UserIDs {
			args = append(args, uid)
		}
	}

	// Add time range filters
	if params.FromTime != nil {
		conditions = append(conditions, "CREATED_TIME >= ?")
		args = append(args, *params.FromTime)
	}

	if params.ToTime != nil {
		conditions = append(conditions, "CREATED_TIME <= ?")
		args = append(args, *params.ToTime)
	}

	whereClause := strings.Join(conditions, " AND ")

	// Count total matching records
	countQuery := fmt.Sprintf("SELECT COUNT(DISTINCT c.CONSENT_ID) FROM FS_CONSENT c%s WHERE %s", joinClause, whereClause)
	var total int
	err := dao.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count consents: %w", err)
	}

	// Build the main query
	query := fmt.Sprintf(`
		SELECT DISTINCT c.CONSENT_ID, c.RECEIPT, c.CREATED_TIME, c.UPDATED_TIME, c.CLIENT_ID,
		       c.CONSENT_TYPE, c.CURRENT_STATUS, c.CONSENT_FREQUENCY, c.VALIDITY_TIME,
		       c.RECURRING_INDICATOR, c.ORG_ID
		FROM FS_CONSENT c%s
		WHERE %s
		ORDER BY c.CREATED_TIME DESC
		LIMIT ? OFFSET ?
	`, joinClause, whereClause)

	// Set default pagination if not provided
	limit := params.Limit
	if limit <= 0 {
		limit = 20 // Default limit
	}
	offset := params.Offset
	if offset < 0 {
		offset = 0
	}

	args = append(args, limit, offset)

	var consents []models.Consent
	err = dao.db.SelectContext(ctx, &consents, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to search consents: %w", err)
	}

	return consents, total, nil
}

// GetByClientID retrieves all consents for a specific client
func (dao *ConsentDAO) GetByClientID(ctx context.Context, clientID, orgID string) ([]models.Consent, error) {
	query := `
		SELECT CONSENT_ID, RECEIPT, CREATED_TIME, UPDATED_TIME, CLIENT_ID,
		       CONSENT_TYPE, CURRENT_STATUS, CONSENT_FREQUENCY, VALIDITY_TIME,
		       RECURRING_INDICATOR, ORG_ID
		FROM FS_CONSENT
		WHERE CLIENT_ID = ? AND ORG_ID = ?
		ORDER BY CREATED_TIME DESC
	`

	var consents []models.Consent
	err := dao.db.SelectContext(ctx, &consents, query, clientID, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get consents by client ID: %w", err)
	}

	return consents, nil
}

// Exists checks if a consent exists
func (dao *ConsentDAO) Exists(ctx context.Context, consentID, orgID string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM FS_CONSENT WHERE CONSENT_ID = ? AND ORG_ID = ?)`

	var exists bool
	err := dao.db.GetContext(ctx, &exists, query, consentID, orgID)
	if err != nil {
		return false, fmt.Errorf("failed to check consent existence: %w", err)
	}

	return exists, nil
}

// GetQueryBuilder returns a query builder for complex queries (future enhancement)
type QueryBuilder struct {
	baseQuery  string
	conditions []string
	args       []interface{}
}

// NewQueryBuilder creates a new query builder
func (dao *ConsentDAO) NewQueryBuilder() *QueryBuilder {
	return &QueryBuilder{
		baseQuery: `
			SELECT CONSENT_ID, RECEIPT, CREATED_TIME, UPDATED_TIME, CLIENT_ID,
			       CONSENT_TYPE, CURRENT_STATUS, CONSENT_FREQUENCY, VALIDITY_TIME,
			       RECURRING_INDICATOR, ORG_ID
			FROM FS_CONSENT
		`,
		conditions: []string{},
		args:       []interface{}{},
	}
}

// AddCondition adds a WHERE condition
func (qb *QueryBuilder) AddCondition(condition string, args ...interface{}) *QueryBuilder {
	qb.conditions = append(qb.conditions, condition)
	qb.args = append(qb.args, args...)
	return qb
}

// Build builds the final query
func (qb *QueryBuilder) Build() (string, []interface{}) {
	query := qb.baseQuery
	if len(qb.conditions) > 0 {
		query += " WHERE " + strings.Join(qb.conditions, " AND ")
	}
	return query, qb.args
}

// Execute executes the query and returns results
func (dao *ConsentDAO) Execute(ctx context.Context, qb *QueryBuilder) ([]models.Consent, error) {
	query, args := qb.Build()
	var consents []models.Consent
	err := dao.db.SelectContext(ctx, &consents, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	return consents, nil
}
