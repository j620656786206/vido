package migrations

import (
	"database/sql"
	"fmt"
)

// CreateSystemLogsTable is the migration to create the system_logs table
// for storing structured application logs (Story 6.3)
type CreateSystemLogsTable struct {
	migrationBase
}

func init() {
	Register(&CreateSystemLogsTable{
		migrationBase: NewMigrationBase(16, "create_system_logs_table"),
	})
}

// Up creates the system_logs table with indexes for efficient querying
func (m *CreateSystemLogsTable) Up(tx *sql.Tx) error {
	query := `
		CREATE TABLE IF NOT EXISTS system_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			level TEXT NOT NULL,
			message TEXT NOT NULL,
			source TEXT,
			context_json TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);

		CREATE INDEX IF NOT EXISTS idx_system_logs_level ON system_logs(level);
		CREATE INDEX IF NOT EXISTS idx_system_logs_created_at ON system_logs(created_at DESC);
	`

	if _, err := tx.Exec(query); err != nil {
		return fmt.Errorf("failed to create system_logs table: %w", err)
	}

	return nil
}

// Down drops the system_logs table
func (m *CreateSystemLogsTable) Down(tx *sql.Tx) error {
	query := `
		DROP INDEX IF EXISTS idx_system_logs_created_at;
		DROP INDEX IF EXISTS idx_system_logs_level;
		DROP TABLE IF EXISTS system_logs;
	`

	if _, err := tx.Exec(query); err != nil {
		return fmt.Errorf("failed to drop system_logs table: %w", err)
	}

	return nil
}
