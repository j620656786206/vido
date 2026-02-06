package migrations

import (
	"database/sql"
	"fmt"
)

// CreateRetryStatsTable is the migration to create the retry_stats table
// for tracking historical retry success/failure counts (Story 3.11 - AC2, AC3)
type CreateRetryStatsTable struct {
	migrationBase
}

func init() {
	Register(&CreateRetryStatsTable{
		migrationBase: NewMigrationBase(12, "create_retry_stats_table"),
	})
}

// Up creates the retry_stats table
func (m *CreateRetryStatsTable) Up(tx *sql.Tx) error {
	query := `
		CREATE TABLE IF NOT EXISTS retry_stats (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			task_type TEXT NOT NULL,
			date TEXT NOT NULL,
			total_queued INTEGER DEFAULT 0,
			total_succeeded INTEGER DEFAULT 0,
			total_failed INTEGER DEFAULT 0,
			total_exhausted INTEGER DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			CONSTRAINT unique_type_date UNIQUE (task_type, date)
		);

		CREATE INDEX IF NOT EXISTS idx_retry_stats_date ON retry_stats(date);
		CREATE INDEX IF NOT EXISTS idx_retry_stats_task_type ON retry_stats(task_type);
	`

	if _, err := tx.Exec(query); err != nil {
		return fmt.Errorf("failed to create retry_stats table: %w", err)
	}

	return nil
}

// Down drops the retry_stats table
func (m *CreateRetryStatsTable) Down(tx *sql.Tx) error {
	query := `
		DROP INDEX IF EXISTS idx_retry_stats_task_type;
		DROP INDEX IF EXISTS idx_retry_stats_date;
		DROP TABLE IF EXISTS retry_stats;
	`

	if _, err := tx.Exec(query); err != nil {
		return fmt.Errorf("failed to drop retry_stats table: %w", err)
	}

	return nil
}
