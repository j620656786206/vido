package migrations

import (
	"database/sql"
	"fmt"
)

// CreateSecretsTable is the migration to create the secrets table
type CreateSecretsTable struct {
	migrationBase
}

func init() {
	// Register this migration with the global registry
	Register(&CreateSecretsTable{
		migrationBase: NewMigrationBase(5, "create_secrets_table"),
	})
}

// Up creates the secrets table for encrypted secrets storage
func (m *CreateSecretsTable) Up(tx *sql.Tx) error {
	query := `
		CREATE TABLE IF NOT EXISTS secrets (
			id TEXT PRIMARY KEY,
			name TEXT UNIQUE NOT NULL,
			encrypted_value TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);

		CREATE INDEX IF NOT EXISTS idx_secrets_name ON secrets(name);
	`

	if _, err := tx.Exec(query); err != nil {
		return fmt.Errorf("failed to create secrets table: %w", err)
	}

	return nil
}

// Down drops the secrets table
func (m *CreateSecretsTable) Down(tx *sql.Tx) error {
	query := `
		DROP INDEX IF EXISTS idx_secrets_name;
		DROP TABLE IF EXISTS secrets;
	`

	if _, err := tx.Exec(query); err != nil {
		return fmt.Errorf("failed to drop secrets table: %w", err)
	}

	return nil
}
