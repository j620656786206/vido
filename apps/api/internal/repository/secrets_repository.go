package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/vido/api/internal/models"
)

var (
	// ErrSecretNotFound indicates the requested secret does not exist.
	ErrSecretNotFound = errors.New("secret not found")
)

// SecretsRepository provides data access operations for encrypted secrets
type SecretsRepository struct {
	db *sql.DB
}

// NewSecretsRepository creates a new instance of SecretsRepository
func NewSecretsRepository(db *sql.DB) *SecretsRepository {
	return &SecretsRepository{
		db: db,
	}
}

// Set creates or updates an encrypted secret (upsert by name)
func (r *SecretsRepository) Set(ctx context.Context, name string, encryptedValue string) error {
	if name == "" {
		return fmt.Errorf("secret name cannot be empty")
	}

	if encryptedValue == "" {
		return fmt.Errorf("encrypted value cannot be empty")
	}

	now := time.Now()

	// Check if secret exists
	var existingID string
	err := r.db.QueryRowContext(ctx, "SELECT id FROM secrets WHERE name = ?", name).Scan(&existingID)

	if err == sql.ErrNoRows {
		// Insert new secret
		id := uuid.New().String()
		query := `
			INSERT INTO secrets (id, name, encrypted_value, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?)
		`
		_, err = r.db.ExecContext(ctx, query, id, name, encryptedValue, now, now)
		if err != nil {
			return fmt.Errorf("failed to insert secret: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("failed to check secret existence: %w", err)
	} else {
		// Update existing secret
		query := `
			UPDATE secrets
			SET encrypted_value = ?, updated_at = ?
			WHERE name = ?
		`
		_, err = r.db.ExecContext(ctx, query, encryptedValue, now, name)
		if err != nil {
			return fmt.Errorf("failed to update secret: %w", err)
		}
	}

	return nil
}

// Get retrieves an encrypted secret by name
func (r *SecretsRepository) Get(ctx context.Context, name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("secret name cannot be empty")
	}

	query := `SELECT encrypted_value FROM secrets WHERE name = ?`

	var encryptedValue string
	err := r.db.QueryRowContext(ctx, query, name).Scan(&encryptedValue)

	if err == sql.ErrNoRows {
		return "", ErrSecretNotFound
	}
	if err != nil {
		return "", fmt.Errorf("failed to get secret: %w", err)
	}

	return encryptedValue, nil
}

// GetFull retrieves the full secret record by name
func (r *SecretsRepository) GetFull(ctx context.Context, name string) (*models.Secret, error) {
	if name == "" {
		return nil, fmt.Errorf("secret name cannot be empty")
	}

	query := `
		SELECT id, name, encrypted_value, created_at, updated_at
		FROM secrets
		WHERE name = ?
	`

	secret := &models.Secret{}
	err := r.db.QueryRowContext(ctx, query, name).Scan(
		&secret.ID,
		&secret.Name,
		&secret.EncryptedValue,
		&secret.CreatedAt,
		&secret.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrSecretNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get secret: %w", err)
	}

	return secret, nil
}

// Delete removes a secret by name
func (r *SecretsRepository) Delete(ctx context.Context, name string) error {
	if name == "" {
		return fmt.Errorf("secret name cannot be empty")
	}

	query := `DELETE FROM secrets WHERE name = ?`

	result, err := r.db.ExecContext(ctx, query, name)
	if err != nil {
		return fmt.Errorf("failed to delete secret: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrSecretNotFound
	}

	return nil
}

// Exists checks if a secret with the given name exists
func (r *SecretsRepository) Exists(ctx context.Context, name string) (bool, error) {
	if name == "" {
		return false, fmt.Errorf("secret name cannot be empty")
	}

	query := `SELECT EXISTS(SELECT 1 FROM secrets WHERE name = ?)`

	var exists bool
	err := r.db.QueryRowContext(ctx, query, name).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check secret existence: %w", err)
	}

	return exists, nil
}

// List returns all secret names (not values)
func (r *SecretsRepository) List(ctx context.Context) ([]string, error) {
	query := `SELECT name FROM secrets ORDER BY name ASC`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list secrets: %w", err)
	}
	defer rows.Close()

	names := []string{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("failed to scan secret name: %w", err)
		}
		names = append(names, name)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating secrets: %w", err)
	}

	return names, nil
}

// ListAll returns all secret records (without encrypted values for security)
func (r *SecretsRepository) ListAll(ctx context.Context) ([]models.SecretInfo, error) {
	query := `
		SELECT id, name, created_at, updated_at
		FROM secrets
		ORDER BY name ASC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list secrets: %w", err)
	}
	defer rows.Close()

	secrets := []models.SecretInfo{}
	for rows.Next() {
		info := models.SecretInfo{}
		if err := rows.Scan(&info.ID, &info.Name, &info.CreatedAt, &info.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan secret: %w", err)
		}
		secrets = append(secrets, info)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating secrets: %w", err)
	}

	return secrets, nil
}
