package dao

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/wso2/consent-management-api/internal/models"
)

// ConsentPurposeDAO handles database operations for consent purposes
type ConsentPurposeDAO struct {
	db *sqlx.DB
}

// NewConsentPurposeDAO creates a new ConsentPurposeDAO
func NewConsentPurposeDAO(db *sqlx.DB) *ConsentPurposeDAO {
	return &ConsentPurposeDAO{db: db}
}

// Create inserts a new consent purpose into the database
func (dao *ConsentPurposeDAO) Create(ctx context.Context, purpose *models.ConsentPurpose) error {
	query := `
		INSERT INTO CONSENT_PURPOSE (ID, NAME, DESCRIPTION, TYPE, ORG_ID)
		VALUES (?, ?, ?, ?, ?)
	`

	_, err := dao.db.ExecContext(ctx, query,
		purpose.ID,
		purpose.Name,
		purpose.Description,
		purpose.Type,
		purpose.OrgID,
	)

	if err != nil {
		return fmt.Errorf("failed to create consent purpose: %w", err)
	}

	return nil
}

// CreateWithTx inserts a new consent purpose within a transaction
func (dao *ConsentPurposeDAO) CreateWithTx(ctx context.Context, tx *sqlx.Tx, purpose *models.ConsentPurpose) error {
	query := `
		INSERT INTO CONSENT_PURPOSE (ID, NAME, DESCRIPTION, TYPE, ORG_ID)
		VALUES (?, ?, ?, ?, ?)
	`

	_, err := tx.ExecContext(ctx, query,
		purpose.ID,
		purpose.Name,
		purpose.Description,
		purpose.Type,
		purpose.OrgID,
	)

	if err != nil {
		return fmt.Errorf("failed to create consent purpose: %w", err)
	}

	return nil
}

// ExistsByNameWithTx checks if a purpose with the given name exists within a transaction
func (dao *ConsentPurposeDAO) ExistsByNameWithTx(ctx context.Context, tx *sqlx.Tx, name, orgID string) (bool, error) {
	query := `SELECT COUNT(*) FROM CONSENT_PURPOSE WHERE NAME = ? AND ORG_ID = ?`

	var count int
	err := tx.GetContext(ctx, &count, query, name, orgID)
	if err != nil {
		return false, fmt.Errorf("failed to check purpose name existence: %w", err)
	}

	return count > 0, nil
}

// GetByID retrieves a consent purpose by ID and organization ID
func (dao *ConsentPurposeDAO) GetByID(ctx context.Context, purposeID, orgID string) (*models.ConsentPurpose, error) {
	query := `
		SELECT ID, NAME, DESCRIPTION, TYPE, ORG_ID
		FROM CONSENT_PURPOSE
		WHERE ID = ? AND ORG_ID = ?
	`

	var purpose models.ConsentPurpose
	err := dao.db.GetContext(ctx, &purpose, query, purposeID, orgID)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("consent purpose not found: %s", purposeID)
		}
		return nil, fmt.Errorf("failed to get consent purpose: %w", err)
	}

	return &purpose, nil
}

// GetByName retrieves a consent purpose by name and organization ID
func (dao *ConsentPurposeDAO) GetByName(ctx context.Context, name, orgID string) (*models.ConsentPurpose, error) {
	query := `
		SELECT ID, NAME, DESCRIPTION, TYPE, ORG_ID
		FROM CONSENT_PURPOSE
		WHERE NAME = ? AND ORG_ID = ?
	`

	var purpose models.ConsentPurpose
	err := dao.db.GetContext(ctx, &purpose, query, name, orgID)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("consent purpose not found: %s", name)
		}
		return nil, fmt.Errorf("failed to get consent purpose: %w", err)
	}

	return &purpose, nil
}

// List retrieves all consent purposes for an organization
func (dao *ConsentPurposeDAO) List(ctx context.Context, orgID string, limit, offset int) ([]models.ConsentPurpose, int, error) {
	// Get total count
	countQuery := `SELECT COUNT(*) FROM CONSENT_PURPOSE WHERE ORG_ID = ?`
	var total int
	err := dao.db.GetContext(ctx, &total, countQuery, orgID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get consent purpose count: %w", err)
	}

	// Get purposes with pagination
	query := `
		SELECT ID, NAME, DESCRIPTION, TYPE, ORG_ID
		FROM CONSENT_PURPOSE
		WHERE ORG_ID = ?
		ORDER BY NAME ASC
		LIMIT ? OFFSET ?
	`

	var purposes []models.ConsentPurpose
	err = dao.db.SelectContext(ctx, &purposes, query, orgID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list consent purposes: %w", err)
	}

	return purposes, total, nil
}

// Update updates an existing consent purpose
func (dao *ConsentPurposeDAO) Update(ctx context.Context, purpose *models.ConsentPurpose) error {
	query := `
		UPDATE CONSENT_PURPOSE
		SET NAME = ?, DESCRIPTION = ?, TYPE = ?
		WHERE ID = ? AND ORG_ID = ?
	`

	_, err := dao.db.ExecContext(ctx, query,
		purpose.Name,
		purpose.Description,
		purpose.Type,
		purpose.ID,
		purpose.OrgID,
	)

	if err != nil {
		return fmt.Errorf("failed to update consent purpose: %w", err)
	}

	// Note: MySQL returns 0 rows affected when values don't change
	// We don't check rowsAffected here because:
	// 1. If ID doesn't exist, we should have caught it earlier (via GetByID in service)
	// 2. If values are the same, UPDATE returns 0 but it's not an error

	return nil
}

// UpdateWithTx updates an existing consent purpose within a transaction
func (dao *ConsentPurposeDAO) UpdateWithTx(ctx context.Context, tx *sqlx.Tx, purpose *models.ConsentPurpose) error {
	query := `
		UPDATE CONSENT_PURPOSE
		SET NAME = ?, DESCRIPTION = ?, TYPE = ?
		WHERE ID = ? AND ORG_ID = ?
	`

	_, err := tx.ExecContext(ctx, query,
		purpose.Name,
		purpose.Description,
		purpose.Type,
		purpose.ID,
		purpose.OrgID,
	)

	if err != nil {
		return fmt.Errorf("failed to update consent purpose: %w", err)
	}

	return nil
}

// Delete removes a consent purpose from the database
func (dao *ConsentPurposeDAO) Delete(ctx context.Context, purposeID, orgID string) error {
	query := `DELETE FROM CONSENT_PURPOSE WHERE ID = ? AND ORG_ID = ?`

	result, err := dao.db.ExecContext(ctx, query, purposeID, orgID)
	if err != nil {
		return fmt.Errorf("failed to delete consent purpose: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("consent purpose not found: %s", purposeID)
	}

	return nil
}

// DeleteWithTx removes a consent purpose from the database within a transaction
func (dao *ConsentPurposeDAO) DeleteWithTx(ctx context.Context, tx *sqlx.Tx, purposeID, orgID string) error {
	query := `DELETE FROM CONSENT_PURPOSE WHERE ID = ? AND ORG_ID = ?`

	result, err := tx.ExecContext(ctx, query, purposeID, orgID)
	if err != nil {
		return fmt.Errorf("failed to delete consent purpose: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("consent purpose not found: %s", purposeID)
	}

	return nil
}

// GetByConsentID retrieves all purposes associated with a consent
func (dao *ConsentPurposeDAO) GetByConsentID(ctx context.Context, consentID, orgID string) ([]models.ConsentPurpose, error) {
	query := `
		SELECT cp.ID, cp.NAME, cp.DESCRIPTION, cp.TYPE, cp.ORG_ID
		FROM CONSENT_PURPOSE cp
		INNER JOIN CONSENT_PURPOSE_MAPPING cpm ON cp.ID = cpm.PURPOSE_ID AND cp.ORG_ID = cpm.ORG_ID
		WHERE cpm.CONSENT_ID = ? AND cpm.ORG_ID = ?
		ORDER BY cp.NAME ASC
	`

	var purposes []models.ConsentPurpose
	err := dao.db.SelectContext(ctx, &purposes, query, consentID, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get purposes for consent: %w", err)
	}

	return purposes, nil
}

// GetMappingsByConsentID retrieves all purpose mappings with value, isUserApproved, and isMandatory for a consent
func (dao *ConsentPurposeDAO) GetMappingsByConsentID(ctx context.Context, consentID, orgID string) ([]models.ConsentPurposeMapping, error) {
	query := `
		SELECT cpm.CONSENT_ID, cpm.ORG_ID, cpm.PURPOSE_ID, cpm.VALUE, cpm.IS_USER_APPROVED, cpm.IS_MANDATORY,
		       cp.NAME
		FROM CONSENT_PURPOSE_MAPPING cpm
		INNER JOIN CONSENT_PURPOSE cp ON cpm.PURPOSE_ID = cp.ID AND cpm.ORG_ID = cp.ORG_ID
		WHERE cpm.CONSENT_ID = ? AND cpm.ORG_ID = ?
		ORDER BY cp.NAME ASC
	`

	type MappingRow struct {
		ConsentID      string  `db:"CONSENT_ID"`
		OrgID          string  `db:"ORG_ID"`
		PurposeID      string  `db:"PURPOSE_ID"`
		Value          *string `db:"VALUE"`
		IsUserApproved bool    `db:"IS_USER_APPROVED"`
		IsMandatory    bool    `db:"IS_MANDATORY"`
		Name           string  `db:"NAME"`
	}

	var rows []MappingRow
	err := dao.db.SelectContext(ctx, &rows, query, consentID, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get purpose mappings for consent: %w", err)
	}

	mappings := make([]models.ConsentPurposeMapping, len(rows))
	for i, row := range rows {
		mappings[i] = models.ConsentPurposeMapping{
			ConsentID:      row.ConsentID,
			OrgID:          row.OrgID,
			PurposeID:      row.PurposeID,
			Value:          nil,
			IsUserApproved: row.IsUserApproved,
			IsMandatory:    row.IsMandatory,
			Name:           row.Name, // For building the response
		}

		// Unmarshal value JSON if present
		if row.Value != nil && *row.Value != "" {
			var valueObj interface{}
			if err := json.Unmarshal([]byte(*row.Value), &valueObj); err == nil {
				mappings[i].Value = valueObj
			}
		}
	}

	return mappings, nil
}

// LinkPurposeToConsent creates a mapping between a consent and a purpose
func (dao *ConsentPurposeDAO) LinkPurposeToConsent(ctx context.Context, consentID, purposeID, orgID string, value *string, isUserApproved, isMandatory bool) error {
	query := `
		INSERT INTO CONSENT_PURPOSE_MAPPING (CONSENT_ID, ORG_ID, PURPOSE_ID, VALUE, IS_USER_APPROVED, IS_MANDATORY)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err := dao.db.ExecContext(ctx, query, consentID, orgID, purposeID, value, isUserApproved, isMandatory)
	if err != nil {
		return fmt.Errorf("failed to link purpose to consent: %w", err)
	}

	return nil
}

// LinkPurposeToConsentWithTx creates a mapping between a consent and a purpose within a transaction
func (dao *ConsentPurposeDAO) LinkPurposeToConsentWithTx(ctx context.Context, tx *sqlx.Tx, consentID, purposeID, orgID string, value *string, isUserApproved, isMandatory bool) error {
	query := `
		INSERT INTO CONSENT_PURPOSE_MAPPING (CONSENT_ID, ORG_ID, PURPOSE_ID, VALUE, IS_USER_APPROVED, IS_MANDATORY)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err := tx.ExecContext(ctx, query, consentID, orgID, purposeID, value, isUserApproved, isMandatory)
	if err != nil {
		return fmt.Errorf("failed to link purpose to consent: %w", err)
	}

	return nil
}

// UnlinkPurposeFromConsent removes the mapping between a consent and a purpose
func (dao *ConsentPurposeDAO) UnlinkPurposeFromConsent(ctx context.Context, consentID, purposeID, orgID string) error {
	query := `
		DELETE FROM CONSENT_PURPOSE_MAPPING
		WHERE CONSENT_ID = ? AND PURPOSE_ID = ? AND ORG_ID = ?
	`

	result, err := dao.db.ExecContext(ctx, query, consentID, purposeID, orgID)
	if err != nil {
		return fmt.Errorf("failed to unlink purpose from consent: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("purpose mapping not found for consent: %s and purpose: %s", consentID, purposeID)
	}

	return nil
}

// GetConsentsByPurpose retrieves all consent IDs associated with a purpose
func (dao *ConsentPurposeDAO) GetConsentsByPurpose(ctx context.Context, purposeID, orgID string, limit, offset int) ([]string, int, error) {
	// Get total count
	countQuery := `
		SELECT COUNT(*) FROM CONSENT_PURPOSE_MAPPING
		WHERE PURPOSE_ID = ? AND ORG_ID = ?
	`
	var total int
	err := dao.db.GetContext(ctx, &total, countQuery, purposeID, orgID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get consent count for purpose: %w", err)
	}

	// Get consent IDs with pagination
	query := `
		SELECT CONSENT_ID FROM CONSENT_PURPOSE_MAPPING
		WHERE PURPOSE_ID = ? AND ORG_ID = ?
		LIMIT ? OFFSET ?
	`

	var consentIDs []string
	err = dao.db.SelectContext(ctx, &consentIDs, query, purposeID, orgID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get consents for purpose: %w", err)
	}

	return consentIDs, total, nil
}

// ClearConsentPurposes removes all purpose mappings for a consent
func (dao *ConsentPurposeDAO) ClearConsentPurposes(ctx context.Context, consentID, orgID string) error {
	query := `DELETE FROM CONSENT_PURPOSE_MAPPING WHERE CONSENT_ID = ? AND ORG_ID = ?`

	_, err := dao.db.ExecContext(ctx, query, consentID, orgID)
	if err != nil {
		return fmt.Errorf("failed to clear consent purposes: %w", err)
	}

	return nil
}

// ClearConsentPurposesWithTx removes all purpose mappings for a consent within a transaction
func (dao *ConsentPurposeDAO) ClearConsentPurposesWithTx(ctx context.Context, tx *sqlx.Tx, consentID, orgID string) error {
	query := `DELETE FROM CONSENT_PURPOSE_MAPPING WHERE CONSENT_ID = ? AND ORG_ID = ?`

	_, err := tx.ExecContext(ctx, query, consentID, orgID)
	if err != nil {
		return fmt.Errorf("failed to clear consent purposes: %w", err)
	}

	return nil
}

// ExistsByName checks if a purpose with the given name already exists for the organization
func (dao *ConsentPurposeDAO) ExistsByName(ctx context.Context, name, orgID string) (bool, error) {
	query := `
		SELECT COUNT(*) FROM CONSENT_PURPOSE
		WHERE NAME = ? AND ORG_ID = ?
	`

	var count int
	err := dao.db.GetContext(ctx, &count, query, name, orgID)
	if err != nil {
		return false, fmt.Errorf("failed to check purpose name existence: %w", err)
	}

	return count > 0, nil
}

// GetIDsByNames retrieves purpose IDs for given purpose names
func (dao *ConsentPurposeDAO) GetIDsByNames(ctx context.Context, names []string, orgID string) (map[string]string, error) {
	if len(names) == 0 {
		return make(map[string]string), nil
	}

	// Build query with IN clause
	query := `
		SELECT NAME, ID FROM CONSENT_PURPOSE
		WHERE ORG_ID = ? AND NAME IN (?)
	`

	query, args, err := sqlx.In(query, orgID, names)
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}
	query = dao.db.Rebind(query)

	rows, err := dao.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get purpose IDs by names: %w", err)
	}
	defer rows.Close()

	result := make(map[string]string)
	for rows.Next() {
		var name, id string
		if err := rows.Scan(&name, &id); err != nil {
			return nil, fmt.Errorf("failed to scan purpose: %w", err)
		}
		result[name] = id
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return result, nil
}

// ValidatePurposeNames checks which purpose names exist in the database for an organization
// Returns a slice of valid purpose names that exist in the database
func (dao *ConsentPurposeDAO) ValidatePurposeNames(ctx context.Context, names []string, orgID string) ([]string, error) {
	if len(names) == 0 {
		return []string{}, nil
	}

	// Build query with IN clause
	query := `
		SELECT NAME FROM CONSENT_PURPOSE
		WHERE ORG_ID = ? AND NAME IN (?)
		ORDER BY NAME ASC
	`

	query, args, err := sqlx.In(query, orgID, names)
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}
	query = dao.db.Rebind(query)

	var validNames []string
	err = dao.db.SelectContext(ctx, &validNames, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to validate purpose names: %w", err)
	}

	return validNames, nil
}

// HasConsentBindings checks if a consent purpose has any existing consent bindings
// Returns true if the purpose is referenced in any consent, false otherwise
func (dao *ConsentPurposeDAO) HasConsentBindings(ctx context.Context, purposeID, orgID string) (bool, error) {
	query := `
		SELECT COUNT(*) 
		FROM CONSENT_PURPOSE_MAPPING 
		WHERE PURPOSE_ID = ? AND ORG_ID = ?
	`

	var count int
	err := dao.db.GetContext(ctx, &count, query, purposeID, orgID)
	if err != nil {
		return false, fmt.Errorf("failed to check consent bindings: %w", err)
	}

	return count > 0, nil
}

// CountConsentBindings returns the number of consents that reference a given purpose
func (dao *ConsentPurposeDAO) CountConsentBindings(ctx context.Context, purposeID, orgID string) (int, error) {
	query := `
		SELECT COUNT(*) 
		FROM CONSENT_PURPOSE_MAPPING 
		WHERE PURPOSE_ID = ? AND ORG_ID = ?
	`

	var count int
	err := dao.db.GetContext(ctx, &count, query, purposeID, orgID)
	if err != nil {
		return 0, fmt.Errorf("failed to count consent bindings: %w", err)
	}

	return count, nil
}
