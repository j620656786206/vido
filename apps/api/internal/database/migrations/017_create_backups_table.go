package migrations

import "database/sql"

func init() {
	Register(&createBackupsTable{
		migrationBase: NewMigrationBase(17, "create_backups_table"),
	})
}

type createBackupsTable struct {
	migrationBase
}

func (m *createBackupsTable) Up(tx *sql.Tx) error {
	_, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS backups (
			id TEXT PRIMARY KEY,
			filename TEXT NOT NULL,
			size_bytes INTEGER NOT NULL DEFAULT 0,
			schema_version INTEGER NOT NULL,
			checksum TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'pending',
			error_message TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);

		CREATE INDEX IF NOT EXISTS idx_backups_created_at ON backups(created_at DESC);
		CREATE INDEX IF NOT EXISTS idx_backups_status ON backups(status);
	`)
	return err
}

func (m *createBackupsTable) Down(tx *sql.Tx) error {
	_, err := tx.Exec(`
		DROP INDEX IF EXISTS idx_backups_status;
		DROP INDEX IF EXISTS idx_backups_created_at;
		DROP TABLE IF EXISTS backups;
	`)
	return err
}
