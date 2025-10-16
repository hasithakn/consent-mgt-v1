package dao

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/wso2/consent-management-api/internal/database"
	"github.com/wso2/consent-management-api/internal/models"
)

// ConsentFileDAO handles database operations for consent files
type ConsentFileDAO struct {
	db *database.DB
}

// NewConsentFileDAO creates a new ConsentFileDAO instance
func NewConsentFileDAO(db *database.DB) *ConsentFileDAO {
	return &ConsentFileDAO{db: db}
}

// Upload inserts a new consent file (BLOB data)
func (dao *ConsentFileDAO) Upload(ctx context.Context, file *models.ConsentFile) error {
	query := `
		INSERT INTO FS_CONSENT_FILE (CONSENT_ID, CONSENT_FILE, ORG_ID)
		VALUES (?, ?, ?)
	`

	_, err := dao.db.ExecContext(
		ctx,
		query,
		file.ConsentID,
		file.ConsentFile,
		file.OrgID,
	)

	if err != nil {
		return fmt.Errorf("failed to upload consent file: %w", err)
	}

	return nil
}

// UploadWithTx inserts a new consent file using a transaction
func (dao *ConsentFileDAO) UploadWithTx(ctx context.Context, tx *database.Transaction, file *models.ConsentFile) error {
	query := `
		INSERT INTO FS_CONSENT_FILE (CONSENT_ID, CONSENT_FILE, ORG_ID)
		VALUES (?, ?, ?)
	`

	_, err := tx.ExecContext(
		ctx,
		query,
		file.ConsentID,
		file.ConsentFile,
		file.OrgID,
	)

	if err != nil {
		return fmt.Errorf("failed to upload consent file with transaction: %w", err)
	}

	return nil
}

// Get retrieves a consent file by consent ID
func (dao *ConsentFileDAO) Get(ctx context.Context, consentID, orgID string) (*models.ConsentFile, error) {
	query := `
		SELECT CONSENT_ID, CONSENT_FILE, ORG_ID
		FROM FS_CONSENT_FILE
		WHERE CONSENT_ID = ? AND ORG_ID = ?
	`

	var file models.ConsentFile
	err := dao.db.GetContext(ctx, &file, query, consentID, orgID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("consent file not found: %s", consentID)
		}
		return nil, fmt.Errorf("failed to get consent file: %w", err)
	}

	return &file, nil
}

// GetWithTx retrieves a consent file by consent ID using a transaction
func (dao *ConsentFileDAO) GetWithTx(ctx context.Context, tx *database.Transaction, consentID, orgID string) (*models.ConsentFile, error) {
	query := `
		SELECT CONSENT_ID, CONSENT_FILE, ORG_ID
		FROM FS_CONSENT_FILE
		WHERE CONSENT_ID = ? AND ORG_ID = ?
	`

	var file models.ConsentFile
	err := tx.GetContext(ctx, &file, query, consentID, orgID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("consent file not found: %s", consentID)
		}
		return nil, fmt.Errorf("failed to get consent file: %w", err)
	}

	return &file, nil
}

// GetMetadata retrieves file metadata without the BLOB data (for performance)
func (dao *ConsentFileDAO) GetMetadata(ctx context.Context, consentID, orgID string) (*models.ConsentFileResponse, error) {
	query := `
		SELECT CONSENT_ID, LENGTH(CONSENT_FILE) as file_size, ORG_ID
		FROM FS_CONSENT_FILE
		WHERE CONSENT_ID = ? AND ORG_ID = ?
	`

	type FileMetadata struct {
		ConsentID string `db:"CONSENT_ID"`
		FileSize  int    `db:"file_size"`
		OrgID     string `db:"ORG_ID"`
	}

	var metadata FileMetadata
	err := dao.db.GetContext(ctx, &metadata, query, consentID, orgID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("consent file not found: %s", consentID)
		}
		return nil, fmt.Errorf("failed to get consent file metadata: %w", err)
	}

	return &models.ConsentFileResponse{
		ConsentID: metadata.ConsentID,
		FileSize:  metadata.FileSize,
		OrgID:     metadata.OrgID,
		Message:   "File metadata retrieved successfully",
	}, nil
}

// Update updates an existing consent file
func (dao *ConsentFileDAO) Update(ctx context.Context, file *models.ConsentFile) error {
	query := `
		UPDATE FS_CONSENT_FILE
		SET CONSENT_FILE = ?
		WHERE CONSENT_ID = ? AND ORG_ID = ?
	`

	result, err := dao.db.ExecContext(
		ctx,
		query,
		file.ConsentFile,
		file.ConsentID,
		file.OrgID,
	)

	if err != nil {
		return fmt.Errorf("failed to update consent file: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("consent file not found: %s", file.ConsentID)
	}

	return nil
}

// UpdateWithTx updates an existing consent file using a transaction
func (dao *ConsentFileDAO) UpdateWithTx(ctx context.Context, tx *database.Transaction, file *models.ConsentFile) error {
	query := `
		UPDATE FS_CONSENT_FILE
		SET CONSENT_FILE = ?
		WHERE CONSENT_ID = ? AND ORG_ID = ?
	`

	result, err := tx.ExecContext(
		ctx,
		query,
		file.ConsentFile,
		file.ConsentID,
		file.OrgID,
	)

	if err != nil {
		return fmt.Errorf("failed to update consent file with transaction: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("consent file not found: %s", file.ConsentID)
	}

	return nil
}

// Delete deletes a consent file
func (dao *ConsentFileDAO) Delete(ctx context.Context, consentID, orgID string) error {
	query := `DELETE FROM FS_CONSENT_FILE WHERE CONSENT_ID = ? AND ORG_ID = ?`

	result, err := dao.db.ExecContext(ctx, query, consentID, orgID)
	if err != nil {
		return fmt.Errorf("failed to delete consent file: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("consent file not found: %s", consentID)
	}

	return nil
}

// DeleteWithTx deletes a consent file using a transaction
func (dao *ConsentFileDAO) DeleteWithTx(ctx context.Context, tx *database.Transaction, consentID, orgID string) error {
	query := `DELETE FROM FS_CONSENT_FILE WHERE CONSENT_ID = ? AND ORG_ID = ?`

	result, err := tx.ExecContext(ctx, query, consentID, orgID)
	if err != nil {
		return fmt.Errorf("failed to delete consent file with transaction: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("consent file not found: %s", consentID)
	}

	return nil
}

// Exists checks if a consent file exists
func (dao *ConsentFileDAO) Exists(ctx context.Context, consentID, orgID string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM FS_CONSENT_FILE WHERE CONSENT_ID = ? AND ORG_ID = ?)`

	var exists bool
	err := dao.db.GetContext(ctx, &exists, query, consentID, orgID)
	if err != nil {
		return false, fmt.Errorf("failed to check consent file existence: %w", err)
	}

	return exists, nil
}

// GetFileSize retrieves only the size of the file without fetching the BLOB
func (dao *ConsentFileDAO) GetFileSize(ctx context.Context, consentID, orgID string) (int, error) {
	query := `
		SELECT LENGTH(CONSENT_FILE) as file_size
		FROM FS_CONSENT_FILE
		WHERE CONSENT_ID = ? AND ORG_ID = ?
	`

	var fileSize int
	err := dao.db.GetContext(ctx, &fileSize, query, consentID, orgID)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, fmt.Errorf("consent file not found: %s", consentID)
		}
		return 0, fmt.Errorf("failed to get file size: %w", err)
	}

	return fileSize, nil
}

// UpsertFile inserts or updates a consent file (insert if not exists, update if exists)
func (dao *ConsentFileDAO) UpsertFile(ctx context.Context, file *models.ConsentFile) error {
	query := `
		INSERT INTO FS_CONSENT_FILE (CONSENT_ID, CONSENT_FILE, ORG_ID)
		VALUES (?, ?, ?)
		ON DUPLICATE KEY UPDATE CONSENT_FILE = VALUES(CONSENT_FILE)
	`

	_, err := dao.db.ExecContext(
		ctx,
		query,
		file.ConsentID,
		file.ConsentFile,
		file.OrgID,
	)

	if err != nil {
		return fmt.Errorf("failed to upsert consent file: %w", err)
	}

	return nil
}

// UpsertFileWithTx inserts or updates a consent file using a transaction
func (dao *ConsentFileDAO) UpsertFileWithTx(ctx context.Context, tx *database.Transaction, file *models.ConsentFile) error {
	query := `
		INSERT INTO FS_CONSENT_FILE (CONSENT_ID, CONSENT_FILE, ORG_ID)
		VALUES (?, ?, ?)
		ON DUPLICATE KEY UPDATE CONSENT_FILE = VALUES(CONSENT_FILE)
	`

	_, err := tx.ExecContext(
		ctx,
		query,
		file.ConsentID,
		file.ConsentFile,
		file.OrgID,
	)

	if err != nil {
		return fmt.Errorf("failed to upsert consent file with transaction: %w", err)
	}

	return nil
}

// ListByOrgID lists all file metadata for an organization (without BLOB data)
func (dao *ConsentFileDAO) ListByOrgID(ctx context.Context, orgID string, limit, offset int) ([]models.ConsentFileResponse, error) {
	query := `
		SELECT CONSENT_ID, LENGTH(CONSENT_FILE) as file_size, ORG_ID
		FROM FS_CONSENT_FILE
		WHERE ORG_ID = ?
		ORDER BY CONSENT_ID
		LIMIT ? OFFSET ?
	`

	type FileMetadata struct {
		ConsentID string `db:"CONSENT_ID"`
		FileSize  int    `db:"file_size"`
		OrgID     string `db:"ORG_ID"`
	}

	var metadataList []FileMetadata
	err := dao.db.SelectContext(ctx, &metadataList, query, orgID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list consent files: %w", err)
	}

	// Convert to response format
	var responses []models.ConsentFileResponse
	for _, metadata := range metadataList {
		responses = append(responses, models.ConsentFileResponse{
			ConsentID: metadata.ConsentID,
			FileSize:  metadata.FileSize,
			OrgID:     metadata.OrgID,
		})
	}

	return responses, nil
}
