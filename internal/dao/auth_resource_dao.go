package dao

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/wso2/consent-management-api/internal/database"
	"github.com/wso2/consent-management-api/internal/models"
)

// AuthResourceDAO handles database operations for authorization resources
type AuthResourceDAO struct {
	db *database.DB
}

// NewAuthResourceDAO creates a new AuthResourceDAO instance
func NewAuthResourceDAO(db *database.DB) *AuthResourceDAO {
	return &AuthResourceDAO{db: db}
}

// Create inserts a new authorization resource into the database
func (dao *AuthResourceDAO) Create(ctx context.Context, authResource *models.ConsentAuthResource) error {
	query := `
		INSERT INTO FS_CONSENT_AUTH_RESOURCE (
			AUTH_ID, CONSENT_ID, AUTH_TYPE, USER_ID, AUTH_STATUS,
			UPDATED_TIME, APPROVED_PURPOSE_DETAILS, ORG_ID
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := dao.db.ExecContext(
		ctx,
		query,
		authResource.AuthID,
		authResource.ConsentID,
		authResource.AuthType,
		authResource.UserID,
		authResource.AuthStatus,
		authResource.UpdatedTime,
		authResource.ApprovedPurposeDetails,
		authResource.OrgID,
	)

	if err != nil {
		return fmt.Errorf("failed to create auth resource: %w", err)
	}

	return nil
}

// CreateWithTx inserts a new authorization resource using a transaction
func (dao *AuthResourceDAO) CreateWithTx(ctx context.Context, tx *database.Transaction, authResource *models.ConsentAuthResource) error {
	query := `
		INSERT INTO FS_CONSENT_AUTH_RESOURCE (
			AUTH_ID, CONSENT_ID, AUTH_TYPE, USER_ID, AUTH_STATUS,
			UPDATED_TIME, APPROVED_PURPOSE_DETAILS, ORG_ID
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := tx.ExecContext(
		ctx,
		query,
		authResource.AuthID,
		authResource.ConsentID,
		authResource.AuthType,
		authResource.UserID,
		authResource.AuthStatus,
		authResource.UpdatedTime,
		authResource.ApprovedPurposeDetails,
		authResource.OrgID,
	)

	if err != nil {
		return fmt.Errorf("failed to create auth resource with transaction: %w", err)
	}

	return nil
}

// GetByID retrieves an authorization resource by ID
func (dao *AuthResourceDAO) GetByID(ctx context.Context, authID, orgID string) (*models.ConsentAuthResource, error) {
	query := `
		SELECT AUTH_ID, CONSENT_ID, AUTH_TYPE, USER_ID, AUTH_STATUS,
		       UPDATED_TIME, APPROVED_PURPOSE_DETAILS, ORG_ID
		FROM FS_CONSENT_AUTH_RESOURCE
		WHERE AUTH_ID = ? AND ORG_ID = ?
	`

	var authResource models.ConsentAuthResource
	err := dao.db.GetContext(ctx, &authResource, query, authID, orgID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("auth resource not found: %s", authID)
		}
		return nil, fmt.Errorf("failed to get auth resource: %w", err)
	}

	return &authResource, nil
}

// GetByIDWithTx retrieves an authorization resource by ID using a transaction
func (dao *AuthResourceDAO) GetByIDWithTx(ctx context.Context, tx *database.Transaction, authID, orgID string) (*models.ConsentAuthResource, error) {
	query := `
		SELECT AUTH_ID, CONSENT_ID, AUTH_TYPE, USER_ID, AUTH_STATUS,
		       UPDATED_TIME, APPROVED_PURPOSE_DETAILS, ORG_ID
		FROM FS_CONSENT_AUTH_RESOURCE
		WHERE AUTH_ID = ? AND ORG_ID = ?
	`

	var authResource models.ConsentAuthResource
	err := tx.GetContext(ctx, &authResource, query, authID, orgID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("auth resource not found: %s", authID)
		}
		return nil, fmt.Errorf("failed to get auth resource: %w", err)
	}

	return &authResource, nil
}

// GetByConsentID retrieves all authorization resources for a specific consent
func (dao *AuthResourceDAO) GetByConsentID(ctx context.Context, consentID, orgID string) ([]models.ConsentAuthResource, error) {
	query := `
		SELECT AUTH_ID, CONSENT_ID, AUTH_TYPE, USER_ID, AUTH_STATUS,
		       UPDATED_TIME, APPROVED_PURPOSE_DETAILS, ORG_ID
		FROM FS_CONSENT_AUTH_RESOURCE
		WHERE CONSENT_ID = ? AND ORG_ID = ?
		ORDER BY UPDATED_TIME DESC
	`

	var authResources []models.ConsentAuthResource
	err := dao.db.SelectContext(ctx, &authResources, query, consentID, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get auth resources by consent ID: %w", err)
	}

	return authResources, nil
}

// GetByConsentIDWithTx retrieves all authorization resources for a specific consent using a transaction
func (dao *AuthResourceDAO) GetByConsentIDWithTx(ctx context.Context, tx *database.Transaction, consentID, orgID string) ([]models.ConsentAuthResource, error) {
	query := `
		SELECT AUTH_ID, CONSENT_ID, AUTH_TYPE, USER_ID, AUTH_STATUS,
		       UPDATED_TIME, APPROVED_PURPOSE_DETAILS, ORG_ID
		FROM FS_CONSENT_AUTH_RESOURCE
		WHERE CONSENT_ID = ? AND ORG_ID = ?
		ORDER BY UPDATED_TIME DESC
	`

	var authResources []models.ConsentAuthResource
	err := tx.SelectContext(ctx, &authResources, query, consentID, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get auth resources by consent ID: %w", err)
	}

	return authResources, nil
}

// Update updates an existing authorization resource
func (dao *AuthResourceDAO) Update(ctx context.Context, authResource *models.ConsentAuthResource) error {
	query := `
		UPDATE FS_CONSENT_AUTH_RESOURCE
		SET AUTH_STATUS = ?, USER_ID = ?, APPROVED_PURPOSE_DETAILS = ?, UPDATED_TIME = ?
		WHERE AUTH_ID = ? AND ORG_ID = ?
	`

	result, err := dao.db.ExecContext(
		ctx,
		query,
		authResource.AuthStatus,
		authResource.UserID,
		authResource.ApprovedPurposeDetails,
		authResource.UpdatedTime,
		authResource.AuthID,
		authResource.OrgID,
	)

	if err != nil {
		return fmt.Errorf("failed to update auth resource: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("auth resource not found: %s", authResource.AuthID)
	}

	return nil
}

// UpdateWithTx updates an existing authorization resource using a transaction
func (dao *AuthResourceDAO) UpdateWithTx(ctx context.Context, tx *database.Transaction, authResource *models.ConsentAuthResource) error {
	query := `
		UPDATE FS_CONSENT_AUTH_RESOURCE
		SET AUTH_STATUS = ?, USER_ID = ?, APPROVED_PURPOSE_DETAILS = ?, UPDATED_TIME = ?
		WHERE AUTH_ID = ? AND ORG_ID = ?
	`

	result, err := tx.ExecContext(
		ctx,
		query,
		authResource.AuthStatus,
		authResource.UserID,
		authResource.ApprovedPurposeDetails,
		authResource.UpdatedTime,
		authResource.AuthID,
		authResource.OrgID,
	)

	if err != nil {
		return fmt.Errorf("failed to update auth resource with transaction: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("auth resource not found: %s", authResource.AuthID)
	}

	return nil
}

// UpdateStatus updates only the status of an authorization resource
func (dao *AuthResourceDAO) UpdateStatus(ctx context.Context, authID, orgID, status string, updatedTime int64) error {
	query := `
		UPDATE FS_CONSENT_AUTH_RESOURCE
		SET AUTH_STATUS = ?, UPDATED_TIME = ?
		WHERE AUTH_ID = ? AND ORG_ID = ?
	`

	result, err := dao.db.ExecContext(ctx, query, status, updatedTime, authID, orgID)
	if err != nil {
		return fmt.Errorf("failed to update auth resource status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("auth resource not found: %s", authID)
	}

	return nil
}

// Delete deletes an authorization resource
func (dao *AuthResourceDAO) Delete(ctx context.Context, authID, orgID string) error {
	query := `DELETE FROM FS_CONSENT_AUTH_RESOURCE WHERE AUTH_ID = ? AND ORG_ID = ?`

	result, err := dao.db.ExecContext(ctx, query, authID, orgID)
	if err != nil {
		return fmt.Errorf("failed to delete auth resource: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("auth resource not found: %s", authID)
	}

	return nil
}

// DeleteWithTx deletes an authorization resource using a transaction
func (dao *AuthResourceDAO) DeleteWithTx(ctx context.Context, tx *database.Transaction, authID, orgID string) error {
	query := `DELETE FROM FS_CONSENT_AUTH_RESOURCE WHERE AUTH_ID = ? AND ORG_ID = ?`

	result, err := tx.ExecContext(ctx, query, authID, orgID)
	if err != nil {
		return fmt.Errorf("failed to delete auth resource with transaction: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("auth resource not found: %s", authID)
	}

	return nil
}

// DeleteByConsentID deletes all authorization resources for a consent
func (dao *AuthResourceDAO) DeleteByConsentID(ctx context.Context, consentID, orgID string) error {
	query := `DELETE FROM FS_CONSENT_AUTH_RESOURCE WHERE CONSENT_ID = ? AND ORG_ID = ?`

	_, err := dao.db.ExecContext(ctx, query, consentID, orgID)
	if err != nil {
		return fmt.Errorf("failed to delete auth resources by consent ID: %w", err)
	}

	return nil
}

// DeleteByConsentIDWithTx deletes all authorization resources for a consent using a transaction
func (dao *AuthResourceDAO) DeleteByConsentIDWithTx(ctx context.Context, tx *database.Transaction, consentID, orgID string) error {
	query := `DELETE FROM FS_CONSENT_AUTH_RESOURCE WHERE CONSENT_ID = ? AND ORG_ID = ?`

	_, err := tx.ExecContext(ctx, query, consentID, orgID)
	if err != nil {
		return fmt.Errorf("failed to delete auth resources by consent ID with transaction: %w", err)
	}

	return nil
}

// Exists checks if an authorization resource exists
func (dao *AuthResourceDAO) Exists(ctx context.Context, authID, orgID string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM FS_CONSENT_AUTH_RESOURCE WHERE AUTH_ID = ? AND ORG_ID = ?)`

	var exists bool
	err := dao.db.GetContext(ctx, &exists, query, authID, orgID)
	if err != nil {
		return false, fmt.Errorf("failed to check auth resource existence: %w", err)
	}

	return exists, nil
}

// GetByUserID retrieves all authorization resources for a specific user
func (dao *AuthResourceDAO) GetByUserID(ctx context.Context, userID, orgID string) ([]models.ConsentAuthResource, error) {
	query := `
		SELECT AUTH_ID, CONSENT_ID, AUTH_TYPE, USER_ID, AUTH_STATUS,
		       UPDATED_TIME, RESOURCE, ORG_ID
		FROM FS_CONSENT_AUTH_RESOURCE
		WHERE USER_ID = ? AND ORG_ID = ?
		ORDER BY UPDATED_TIME DESC
	`

	var authResources []models.ConsentAuthResource
	err := dao.db.SelectContext(ctx, &authResources, query, userID, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get auth resources by user ID: %w", err)
	}

	return authResources, nil
}
