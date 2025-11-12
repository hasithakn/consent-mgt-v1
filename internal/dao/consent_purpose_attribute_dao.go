package dao

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// ConsentPurposeAttributeDAO handles database operations for consent purpose attributes
type ConsentPurposeAttributeDAO struct {
	db *sqlx.DB
}

// NewConsentPurposeAttributeDAO creates a new ConsentPurposeAttributeDAO
func NewConsentPurposeAttributeDAO(db *sqlx.DB) *ConsentPurposeAttributeDAO {
	return &ConsentPurposeAttributeDAO{db: db}
}

// GetAttributes retrieves all attributes for a purpose
func (dao *ConsentPurposeAttributeDAO) GetAttributes(ctx context.Context, purposeID, orgID string) (map[string]string, error) {
	query := `
		SELECT ATT_KEY, ATT_VALUE
		FROM CONSENT_PURPOSE_ATTRIBUTE
		WHERE PURPOSE_ID = ? AND ORG_ID = ?
	`

	rows, err := dao.db.QueryContext(ctx, query, purposeID, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get attributes: %w", err)
	}
	defer rows.Close()

	attributes := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, fmt.Errorf("failed to scan attribute: %w", err)
		}
		attributes[key] = value
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating attributes: %w", err)
	}

	return attributes, nil
}

// SaveAttributesWithTx saves or updates attributes for a purpose within a transaction
func (dao *ConsentPurposeAttributeDAO) SaveAttributesWithTx(ctx context.Context, tx *sqlx.Tx, purposeID, orgID string, attributes map[string]string) error {
	// Always clear existing attributes first (even if new attributes are empty)
	deleteQuery := `DELETE FROM CONSENT_PURPOSE_ATTRIBUTE WHERE PURPOSE_ID = ? AND ORG_ID = ?`
	_, err := tx.ExecContext(ctx, deleteQuery, purposeID, orgID)
	if err != nil {
		return fmt.Errorf("failed to clear existing attributes: %w", err)
	}

	// If no new attributes, we're done (old ones are deleted)
	if len(attributes) == 0 {
		return nil
	}

	// Insert new attributes
	insertQuery := `
		INSERT INTO CONSENT_PURPOSE_ATTRIBUTE (PURPOSE_ID, ATT_KEY, ATT_VALUE, ORG_ID)
		VALUES (?, ?, ?, ?)
	`

	for key, value := range attributes {
		_, err := tx.ExecContext(ctx, insertQuery, purposeID, key, value, orgID)
		if err != nil {
			return fmt.Errorf("failed to save attribute %s: %w", key, err)
		}
	}

	return nil
}

// DeleteAttributesWithTx deletes all attributes for a purpose within a transaction
func (dao *ConsentPurposeAttributeDAO) DeleteAttributesWithTx(ctx context.Context, tx *sqlx.Tx, purposeID, orgID string) error {
	query := `DELETE FROM CONSENT_PURPOSE_ATTRIBUTE WHERE PURPOSE_ID = ? AND ORG_ID = ?`

	_, err := tx.ExecContext(ctx, query, purposeID, orgID)
	if err != nil {
		return fmt.Errorf("failed to delete attributes: %w", err)
	}

	return nil
}
