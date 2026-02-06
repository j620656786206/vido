package migrations

import (
	"database/sql"
	"fmt"
)

// CreateRetryQueueTable is the migration to create the retry_queue table
// for storing tasks that need to be retried after temporary failures
type CreateRetryQueueTable struct {
	migrationBase
}

func init() {
	Register(&CreateRetryQueueTable{
		migrationBase: NewMigrationBase(11, "create_retry_queue_table"),
	})
}

// Up creates the retry_queue table
func (m *CreateRetryQueueTable) Up(tx *sql.Tx) error {
	query := `
		CREATE TABLE IF NOT EXISTS retry_queue (
			id TEXT PRIMARY KEY,
			task_id TEXT NOT NULL,
			task_type TEXT NOT NULL,
			payload TEXT NOT NULL,
			attempt_count INTEGER DEFAULT 0,
			max_attempts INTEGER DEFAULT 4,
			last_error TEXT,
			next_attempt_at TIMESTAMP NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			CONSTRAINT unique_task_id UNIQUE (task_id)
		);

		CREATE INDEX IF NOT EXISTS idx_retry_queue_next_attempt ON retry_queue(next_attempt_at);
		CREATE INDEX IF NOT EXISTS idx_retry_queue_task_type ON retry_queue(task_type);
	`

	if _, err := tx.Exec(query); err != nil {
		return fmt.Errorf("failed to create retry_queue table: %w", err)
	}

	return nil
}

// Down drops the retry_queue table
func (m *CreateRetryQueueTable) Down(tx *sql.Tx) error {
	query := `
		DROP INDEX IF EXISTS idx_retry_queue_task_type;
		DROP INDEX IF EXISTS idx_retry_queue_next_attempt;
		DROP TABLE IF EXISTS retry_queue;
	`

	if _, err := tx.Exec(query); err != nil {
		return fmt.Errorf("failed to drop retry_queue table: %w", err)
	}

	return nil
}
