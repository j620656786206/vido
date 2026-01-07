package migrations

import (
	"database/sql"
	"time"
)

// Migration represents a database migration with up and down operations
type Migration interface {
	// Version returns the unique version identifier for this migration
	// Versions should be sequential (e.g., 1, 2, 3) or timestamped (e.g., 20240101120000)
	Version() int64

	// Name returns a human-readable name for this migration
	Name() string

	// Up applies the migration
	Up(tx *sql.Tx) error

	// Down reverts the migration
	Down(tx *sql.Tx) error
}

// MigrationRecord represents a migration record in the schema_migrations table
type MigrationRecord struct {
	Version   int64
	Name      string
	AppliedAt time.Time
}

// migrationBase provides a base implementation for common migration functionality
type migrationBase struct {
	version int64
	name    string
}

// Version returns the migration version
func (m *migrationBase) Version() int64 {
	return m.version
}

// Name returns the migration name
func (m *migrationBase) Name() string {
	return m.name
}

// NewMigrationBase creates a new migration base with version and name
func NewMigrationBase(version int64, name string) migrationBase {
	return migrationBase{
		version: version,
		name:    name,
	}
}
