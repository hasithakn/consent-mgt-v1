package dao

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/wso2/consent-management-api/internal/database"
	"github.com/wso2/consent-management-api/internal/models"
)

// ConsentAttributeDAO handles database operations for consent attributes
type ConsentAttributeDAO struct {
	db *database.DB
}

// NewConsentAttributeDAO creates a new ConsentAttributeDAO instance
func NewConsentAttributeDAO(db *database.DB) *ConsentAttributeDAO {
	return &ConsentAttributeDAO{db: db}
}

// Create inserts new consent attributes (multiple key-value pairs)
func (dao *ConsentAttributeDAO) Create(ctx context.Context, consentID, orgID string, attributes map[string]string) error {
	if len(attributes) == 0 {
		return nil // Nothing to insert
	}

	query := `
		INSERT INTO CONSENT_ATTRIBUTE (CONSENT_ID, ATT_KEY, ATT_VALUE, ORG_ID)
		VALUES (?, ?, ?, ?)
	`

	for key, value := range attributes {
		_, err := dao.db.ExecContext(ctx, query, consentID, key, value, orgID)
		if err != nil {
			return fmt.Errorf("failed to create consent attribute %s: %w", key, err)
		}
	}

	return nil
}

// CreateWithTx inserts new consent attributes using a transaction
func (dao *ConsentAttributeDAO) CreateWithTx(ctx context.Context, tx *database.Transaction, consentID, orgID string, attributes map[string]string) error {
	if len(attributes) == 0 {
		return nil // Nothing to insert
	}

	query := `
		INSERT INTO CONSENT_ATTRIBUTE (CONSENT_ID, ATT_KEY, ATT_VALUE, ORG_ID)
		VALUES (?, ?, ?, ?)
	`

	for key, value := range attributes {
		_, err := tx.ExecContext(ctx, query, consentID, key, value, orgID)
		if err != nil {
			return fmt.Errorf("failed to create consent attribute %s with transaction: %w", key, err)
		}
	}

	return nil
}

// GetByConsentID retrieves all attributes for a specific consent
func (dao *ConsentAttributeDAO) GetByConsentID(ctx context.Context, consentID, orgID string) (map[string]string, error) {
	query := `
		SELECT ATT_KEY, ATT_VALUE
		FROM CONSENT_ATTRIBUTE
		WHERE CONSENT_ID = ? AND ORG_ID = ?
	`

	var attributes []models.ConsentAttribute
	err := dao.db.SelectContext(ctx, &attributes, query, consentID, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get consent attributes: %w", err)
	}

	// Convert to map
	result := make(map[string]string)
	for _, attr := range attributes {
		result[attr.AttKey] = attr.AttValue
	}

	return result, nil
}

// GetByConsentIDWithTx retrieves all attributes for a specific consent using a transaction
func (dao *ConsentAttributeDAO) GetByConsentIDWithTx(ctx context.Context, tx *database.Transaction, consentID, orgID string) (map[string]string, error) {
	query := `
		SELECT ATT_KEY, ATT_VALUE
		FROM CONSENT_ATTRIBUTE
		WHERE CONSENT_ID = ? AND ORG_ID = ?
	`

	var attributes []models.ConsentAttribute
	err := tx.SelectContext(ctx, &attributes, query, consentID, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get consent attributes: %w", err)
	}

	// Convert to map
	result := make(map[string]string)
	for _, attr := range attributes {
		result[attr.AttKey] = attr.AttValue
	}

	return result, nil
}

// GetByKey retrieves a specific attribute value by key
func (dao *ConsentAttributeDAO) GetByKey(ctx context.Context, consentID, orgID, key string) (string, error) {
	query := `
		SELECT ATT_VALUE
		FROM CONSENT_ATTRIBUTE
		WHERE CONSENT_ID = ? AND ORG_ID = ? AND ATT_KEY = ?
	`

	var value string
	err := dao.db.GetContext(ctx, &value, query, consentID, orgID, key)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("attribute not found: %s", key)
		}
		return "", fmt.Errorf("failed to get consent attribute: %w", err)
	}

	return value, nil
}

// Update updates consent attributes (replaces all existing attributes)
func (dao *ConsentAttributeDAO) Update(ctx context.Context, consentID, orgID string, attributes map[string]string) error {
	// Delete existing attributes first
	if err := dao.DeleteByConsentID(ctx, consentID, orgID); err != nil {
		return fmt.Errorf("failed to delete existing attributes: %w", err)
	}

	// Insert new attributes
	return dao.Create(ctx, consentID, orgID, attributes)
}

// UpdateWithTx updates consent attributes using a transaction
func (dao *ConsentAttributeDAO) UpdateWithTx(ctx context.Context, tx *database.Transaction, consentID, orgID string, attributes map[string]string) error {
	// Delete existing attributes first
	if err := dao.DeleteByConsentIDWithTx(ctx, tx, consentID, orgID); err != nil {
		return fmt.Errorf("failed to delete existing attributes: %w", err)
	}

	// Insert new attributes
	return dao.CreateWithTx(ctx, tx, consentID, orgID, attributes)
}

// UpdateAttribute updates a single attribute
func (dao *ConsentAttributeDAO) UpdateAttribute(ctx context.Context, consentID, orgID, key, value string) error {
	query := `
		INSERT INTO CONSENT_ATTRIBUTE (CONSENT_ID, ATT_KEY, ATT_VALUE, ORG_ID)
		VALUES (?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE ATT_VALUE = ?
	`

	_, err := dao.db.ExecContext(ctx, query, consentID, key, value, orgID, value)
	if err != nil {
		return fmt.Errorf("failed to update consent attribute: %w", err)
	}

	return nil
}

// DeleteByConsentID deletes all attributes for a consent
func (dao *ConsentAttributeDAO) DeleteByConsentID(ctx context.Context, consentID, orgID string) error {
	query := `DELETE FROM CONSENT_ATTRIBUTE WHERE CONSENT_ID = ? AND ORG_ID = ?`

	_, err := dao.db.ExecContext(ctx, query, consentID, orgID)
	if err != nil {
		return fmt.Errorf("failed to delete consent attributes: %w", err)
	}

	return nil
}

// DeleteByConsentIDWithTx deletes all attributes for a consent using a transaction
func (dao *ConsentAttributeDAO) DeleteByConsentIDWithTx(ctx context.Context, tx *database.Transaction, consentID, orgID string) error {
	query := `DELETE FROM CONSENT_ATTRIBUTE WHERE CONSENT_ID = ? AND ORG_ID = ?`

	_, err := tx.ExecContext(ctx, query, consentID, orgID)
	if err != nil {
		return fmt.Errorf("failed to delete consent attributes with transaction: %w", err)
	}

	return nil
}

// DeleteAttribute deletes a specific attribute by key
func (dao *ConsentAttributeDAO) DeleteAttribute(ctx context.Context, consentID, orgID, key string) error {
	query := `DELETE FROM CONSENT_ATTRIBUTE WHERE CONSENT_ID = ? AND ORG_ID = ? AND ATT_KEY = ?`

	result, err := dao.db.ExecContext(ctx, query, consentID, orgID, key)
	if err != nil {
		return fmt.Errorf("failed to delete consent attribute: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("attribute not found: %s", key)
	}

	return nil
}

// Exists checks if attributes exist for a consent
func (dao *ConsentAttributeDAO) Exists(ctx context.Context, consentID, orgID string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM CONSENT_ATTRIBUTE WHERE CONSENT_ID = ? AND ORG_ID = ?)`

	var exists bool
	err := dao.db.GetContext(ctx, &exists, query, consentID, orgID)
	if err != nil {
		return false, fmt.Errorf("failed to check consent attributes existence: %w", err)
	}

	return exists, nil
}

// AttributeExists checks if a specific attribute exists
func (dao *ConsentAttributeDAO) AttributeExists(ctx context.Context, consentID, orgID, key string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM CONSENT_ATTRIBUTE WHERE CONSENT_ID = ? AND ORG_ID = ? AND ATT_KEY = ?)`

	var exists bool
	err := dao.db.GetContext(ctx, &exists, query, consentID, orgID, key)
	if err != nil {
		return false, fmt.Errorf("failed to check attribute existence: %w", err)
	}

	return exists, nil
}

// FindConsentIDsByAttributeKey finds all consent IDs that have a specific attribute key
func (dao *ConsentAttributeDAO) FindConsentIDsByAttributeKey(ctx context.Context, key, orgID string) ([]string, error) {
	query := `
		SELECT DISTINCT CONSENT_ID
		FROM CONSENT_ATTRIBUTE
		WHERE ATT_KEY = ? AND ORG_ID = ?
		ORDER BY CONSENT_ID
	`

	var consentIDs []string
	err := dao.db.SelectContext(ctx, &consentIDs, query, key, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to find consent IDs by attribute key: %w", err)
	}

	return consentIDs, nil
}

// FindConsentIDsByAttribute finds all consent IDs that have a specific attribute key-value pair
func (dao *ConsentAttributeDAO) FindConsentIDsByAttribute(ctx context.Context, key, value, orgID string) ([]string, error) {
	query := `
		SELECT DISTINCT CONSENT_ID
		FROM CONSENT_ATTRIBUTE
		WHERE ATT_KEY = ? AND ATT_VALUE = ? AND ORG_ID = ?
		ORDER BY CONSENT_ID
	`

	var consentIDs []string
	err := dao.db.SelectContext(ctx, &consentIDs, query, key, value, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to find consent IDs by attribute: %w", err)
	}

	return consentIDs, nil
}
