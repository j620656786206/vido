package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"github.com/vido/api/internal/models"
)

// SettingsRepository provides data access operations for application settings
type SettingsRepository struct {
	db *sql.DB
}

// NewSettingsRepository creates a new instance of SettingsRepository
func NewSettingsRepository(db *sql.DB) *SettingsRepository {
	return &SettingsRepository{
		db: db,
	}
}

// Set creates or updates a setting (upsert operation)
func (r *SettingsRepository) Set(ctx context.Context, setting *models.Setting) error {
	if setting == nil {
		return fmt.Errorf("setting cannot be nil")
	}

	if setting.Key == "" {
		return fmt.Errorf("setting key cannot be empty")
	}

	// Set timestamps
	now := time.Now()
	setting.UpdatedAt = now

	// Check if setting exists
	var exists bool
	err := r.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM settings WHERE key = ?)", setting.Key).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check setting existence: %w", err)
	}

	if exists {
		// Update existing setting
		query := `
			UPDATE settings
			SET value = ?, type = ?, updated_at = ?
			WHERE key = ?
		`
		_, err = r.db.ExecContext(ctx, query,
			setting.Value,
			setting.Type,
			setting.UpdatedAt,
			setting.Key,
		)
		if err != nil {
			return fmt.Errorf("failed to update setting: %w", err)
		}
	} else {
		// Insert new setting
		setting.CreatedAt = now
		query := `
			INSERT INTO settings (key, value, type, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?)
		`
		_, err = r.db.ExecContext(ctx, query,
			setting.Key,
			setting.Value,
			setting.Type,
			setting.CreatedAt,
			setting.UpdatedAt,
		)
		if err != nil {
			return fmt.Errorf("failed to insert setting: %w", err)
		}
	}

	return nil
}

// Get retrieves a setting by its key
func (r *SettingsRepository) Get(ctx context.Context, key string) (*models.Setting, error) {
	if key == "" {
		return nil, fmt.Errorf("setting key cannot be empty")
	}

	query := `
		SELECT key, value, type, created_at, updated_at
		FROM settings
		WHERE key = ?
	`

	setting := &models.Setting{}
	err := r.db.QueryRowContext(ctx, query, key).Scan(
		&setting.Key,
		&setting.Value,
		&setting.Type,
		&setting.CreatedAt,
		&setting.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("setting with key %s not found", key)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get setting: %w", err)
	}

	return setting, nil
}

// GetAll retrieves all settings
func (r *SettingsRepository) GetAll(ctx context.Context) ([]models.Setting, error) {
	query := `
		SELECT key, value, type, created_at, updated_at
		FROM settings
		ORDER BY key ASC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get all settings: %w", err)
	}
	defer rows.Close()

	settings := []models.Setting{}
	for rows.Next() {
		setting := models.Setting{}
		err := rows.Scan(
			&setting.Key,
			&setting.Value,
			&setting.Type,
			&setting.CreatedAt,
			&setting.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan setting: %w", err)
		}
		settings = append(settings, setting)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating settings: %w", err)
	}

	return settings, nil
}

// Delete removes a setting from the database by key
func (r *SettingsRepository) Delete(ctx context.Context, key string) error {
	if key == "" {
		return fmt.Errorf("setting key cannot be empty")
	}

	query := `DELETE FROM settings WHERE key = ?`

	result, err := r.db.ExecContext(ctx, query, key)
	if err != nil {
		return fmt.Errorf("failed to delete setting: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("setting with key %s not found", key)
	}

	return nil
}

// GetString retrieves a setting as a string value
// Returns an error if the setting doesn't exist or has a different type
func (r *SettingsRepository) GetString(ctx context.Context, key string) (string, error) {
	setting, err := r.Get(ctx, key)
	if err != nil {
		return "", err
	}

	if setting.Type != string(models.SettingTypeString) {
		return "", fmt.Errorf("setting %s is not a string type (actual: %s)", key, setting.Type)
	}

	return setting.Value, nil
}

// GetInt retrieves a setting as an integer value
// Returns an error if the setting doesn't exist, has a different type, or cannot be parsed as an integer
func (r *SettingsRepository) GetInt(ctx context.Context, key string) (int, error) {
	setting, err := r.Get(ctx, key)
	if err != nil {
		return 0, err
	}

	if setting.Type != string(models.SettingTypeInt) {
		return 0, fmt.Errorf("setting %s is not an int type (actual: %s)", key, setting.Type)
	}

	value, err := strconv.Atoi(setting.Value)
	if err != nil {
		return 0, fmt.Errorf("failed to parse setting %s as int: %w", key, err)
	}

	return value, nil
}

// GetBool retrieves a setting as a boolean value
// Returns an error if the setting doesn't exist, has a different type, or cannot be parsed as a boolean
func (r *SettingsRepository) GetBool(ctx context.Context, key string) (bool, error) {
	setting, err := r.Get(ctx, key)
	if err != nil {
		return false, err
	}

	if setting.Type != string(models.SettingTypeBool) {
		return false, fmt.Errorf("setting %s is not a bool type (actual: %s)", key, setting.Type)
	}

	value, err := strconv.ParseBool(setting.Value)
	if err != nil {
		return false, fmt.Errorf("failed to parse setting %s as bool: %w", key, err)
	}

	return value, nil
}

// SetString is a convenience method to set a string value
func (r *SettingsRepository) SetString(ctx context.Context, key, value string) error {
	return r.Set(ctx, &models.Setting{
		Key:   key,
		Value: value,
		Type:  string(models.SettingTypeString),
	})
}

// SetInt is a convenience method to set an integer value
func (r *SettingsRepository) SetInt(ctx context.Context, key string, value int) error {
	return r.Set(ctx, &models.Setting{
		Key:   key,
		Value: strconv.Itoa(value),
		Type:  string(models.SettingTypeInt),
	})
}

// SetBool is a convenience method to set a boolean value
func (r *SettingsRepository) SetBool(ctx context.Context, key string, value bool) error {
	return r.Set(ctx, &models.Setting{
		Key:   key,
		Value: strconv.FormatBool(value),
		Type:  string(models.SettingTypeBool),
	})
}
