package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/vido/api/internal/models"
)

// ErrTableMissing is returned when a required database table does not exist.
var ErrTableMissing = errors.New("table missing")

// BackupRepository provides data access operations for backup records.
type BackupRepository struct {
	db *sql.DB
}

// NewBackupRepository creates a new instance of BackupRepository.
func NewBackupRepository(db *sql.DB) *BackupRepository {
	return &BackupRepository{db: db}
}

// Create inserts a new backup record into the database.
func (r *BackupRepository) Create(ctx context.Context, backup *models.Backup) error {
	query := `INSERT INTO backups (id, filename, size_bytes, schema_version, checksum, status, error_message, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := r.db.ExecContext(ctx, query,
		backup.ID, backup.Filename, backup.SizeBytes, backup.SchemaVersion,
		backup.Checksum, string(backup.Status), backup.ErrorMessage, backup.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert backup: %w", err)
	}
	return nil
}

// isTableMissing checks if a SQLite error indicates a missing table.
// This relies on the error message format from modernc.org/sqlite (production driver).
// Validated by TestIsTableMissing and TestBackupRepository_List_MissingTable integration tests.
func isTableMissing(err error) bool {
	return err != nil && strings.Contains(err.Error(), "no such table")
}

// List retrieves all backups ordered by creation time descending.
func (r *BackupRepository) List(ctx context.Context) ([]models.Backup, error) {
	query := `SELECT id, filename, size_bytes, schema_version, checksum, status, error_message, created_at
		FROM backups ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		if isTableMissing(err) {
			return nil, fmt.Errorf("query backups: %w: %w", ErrTableMissing, err)
		}
		return nil, fmt.Errorf("query backups: %w", err)
	}
	defer rows.Close()

	var backups []models.Backup
	for rows.Next() {
		var b models.Backup
		if err := rows.Scan(&b.ID, &b.Filename, &b.SizeBytes, &b.SchemaVersion,
			&b.Checksum, &b.Status, &b.ErrorMessage, &b.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan backup: %w", err)
		}
		backups = append(backups, b)
	}
	return backups, rows.Err()
}

// GetByID retrieves a backup by its ID.
func (r *BackupRepository) GetByID(ctx context.Context, id string) (*models.Backup, error) {
	query := `SELECT id, filename, size_bytes, schema_version, checksum, status, error_message, created_at
		FROM backups WHERE id = ?`

	var b models.Backup
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&b.ID, &b.Filename, &b.SizeBytes, &b.SchemaVersion,
		&b.Checksum, &b.Status, &b.ErrorMessage, &b.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get backup by id: %w", err)
	}
	return &b, nil
}

// Update modifies an existing backup record.
func (r *BackupRepository) Update(ctx context.Context, backup *models.Backup) error {
	query := `UPDATE backups SET filename = ?, size_bytes = ?, checksum = ?, status = ?, error_message = ?
		WHERE id = ?`

	_, err := r.db.ExecContext(ctx, query,
		backup.Filename, backup.SizeBytes, backup.Checksum,
		string(backup.Status), backup.ErrorMessage, backup.ID,
	)
	if err != nil {
		return fmt.Errorf("update backup: %w", err)
	}
	return nil
}

// Delete removes a backup record by ID.
func (r *BackupRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM backups WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete backup: %w", err)
	}
	return nil
}

// TotalSizeBytes returns the sum of all backup sizes.
func (r *BackupRepository) TotalSizeBytes(ctx context.Context) (int64, error) {
	var total sql.NullInt64
	err := r.db.QueryRowContext(ctx, `SELECT SUM(size_bytes) FROM backups WHERE status = 'completed'`).Scan(&total)
	if err != nil {
		if isTableMissing(err) {
			return 0, fmt.Errorf("sum backup sizes: %w: %w", ErrTableMissing, err)
		}
		return 0, fmt.Errorf("sum backup sizes: %w", err)
	}
	if total.Valid {
		return total.Int64, nil
	}
	return 0, nil
}
