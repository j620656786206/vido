package migrations

import (
	"database/sql"
	"fmt"
)

// CreateConnectionHistoryTable is the migration to create the connection_history table
// for tracking service connection status changes (Story 4.6)
type CreateConnectionHistoryTable struct {
	migrationBase
}

func init() {
	Register(&CreateConnectionHistoryTable{
		migrationBase: NewMigrationBase(14, "create_connection_history_table"),
	})
}

// Up creates the connection_history table
func (m *CreateConnectionHistoryTable) Up(tx *sql.Tx) error {
	query := `
		CREATE TABLE IF NOT EXISTS connection_history (
			id TEXT PRIMARY KEY,
			service TEXT NOT NULL,
			event_type TEXT NOT NULL,
			status TEXT NOT NULL,
			message TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);

		CREATE INDEX IF NOT EXISTS idx_connection_history_service ON connection_history(service);
		CREATE INDEX IF NOT EXISTS idx_connection_history_created_at ON connection_history(created_at);
	`

	if _, err := tx.Exec(query); err != nil {
		return fmt.Errorf("failed to create connection_history table: %w", err)
	}

	return nil
}

// Down drops the connection_history table
func (m *CreateConnectionHistoryTable) Down(tx *sql.Tx) error {
	query := `
		DROP INDEX IF EXISTS idx_connection_history_created_at;
		DROP INDEX IF EXISTS idx_connection_history_service;
		DROP TABLE IF EXISTS connection_history;
	`

	if _, err := tx.Exec(query); err != nil {
		return fmt.Errorf("failed to drop connection_history table: %w", err)
	}

	return nil
}
