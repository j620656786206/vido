package migrations

import (
	"database/sql"
	"fmt"
)

// CreateSettingsTable is the migration to create the settings table
type CreateSettingsTable struct {
	migrationBase
}

func init() {
	// Register this migration with the global registry
	Register(&CreateSettingsTable{
		migrationBase: NewMigrationBase(3, "create_settings_table"),
	})
}

// Up creates the settings table
func (m *CreateSettingsTable) Up(tx *sql.Tx) error {
	query := `
		CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			type TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
	`

	if _, err := tx.Exec(query); err != nil {
		return fmt.Errorf("failed to create settings table: %w", err)
	}

	return nil
}

// Down drops the settings table
func (m *CreateSettingsTable) Down(tx *sql.Tx) error {
	query := `
		DROP TABLE IF EXISTS settings;
	`

	if _, err := tx.Exec(query); err != nil {
		return fmt.Errorf("failed to drop settings table: %w", err)
	}

	return nil
}
