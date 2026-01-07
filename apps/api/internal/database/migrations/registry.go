package migrations

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"sync"
)

var (
	// globalRegistry holds all registered migrations
	globalRegistry = &Registry{
		migrations: make(map[int64]Migration),
		mu:         sync.RWMutex{},
	}
)

// Registry manages a collection of migrations
type Registry struct {
	migrations map[int64]Migration
	mu         sync.RWMutex
}

// Register adds a migration to the global registry
// This function is safe to call from init() functions
func Register(migration Migration) error {
	return globalRegistry.Register(migration)
}

// Register adds a migration to this registry
func (r *Registry) Register(migration Migration) error {
	if migration == nil {
		return fmt.Errorf("migration cannot be nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	version := migration.Version()
	if _, exists := r.migrations[version]; exists {
		return fmt.Errorf("migration with version %d already registered", version)
	}

	r.migrations[version] = migration
	return nil
}

// GetAll returns all registered migrations sorted by version
func GetAll() []Migration {
	return globalRegistry.GetAll()
}

// GetAll returns all migrations from this registry sorted by version
func (r *Registry) GetAll() []Migration {
	r.mu.RLock()
	defer r.mu.RUnlock()

	migrations := make([]Migration, 0, len(r.migrations))
	for _, m := range r.migrations {
		migrations = append(migrations, m)
	}

	// Sort by version
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version() < migrations[j].Version()
	})

	return migrations
}

// GetByVersion returns a specific migration by version
func GetByVersion(version int64) (Migration, error) {
	return globalRegistry.GetByVersion(version)
}

// GetByVersion returns a specific migration by version from this registry
func (r *Registry) GetByVersion(version int64) (Migration, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	migration, exists := r.migrations[version]
	if !exists {
		return nil, fmt.Errorf("migration version %d not found", version)
	}

	return migration, nil
}

// Count returns the number of registered migrations
func Count() int {
	return globalRegistry.Count()
}

// Count returns the number of migrations in this registry
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.migrations)
}

// GetPending returns migrations that haven't been applied yet
func GetPending(ctx context.Context, db *sql.DB) ([]Migration, error) {
	return globalRegistry.GetPending(ctx, db)
}

// GetPending returns migrations from this registry that haven't been applied yet
func (r *Registry) GetPending(ctx context.Context, db *sql.DB) ([]Migration, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection cannot be nil")
	}

	// Get applied migrations from database
	appliedVersions, err := r.getAppliedVersions(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("failed to get applied migrations: %w", err)
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	// Find pending migrations
	pending := make([]Migration, 0)
	for version, migration := range r.migrations {
		if !appliedVersions[version] {
			pending = append(pending, migration)
		}
	}

	// Sort by version
	sort.Slice(pending, func(i, j int) bool {
		return pending[i].Version() < pending[j].Version()
	})

	return pending, nil
}

// GetStatus returns the status of all migrations
func GetStatus(ctx context.Context, db *sql.DB) ([]MigrationStatus, error) {
	return globalRegistry.GetStatus(ctx, db)
}

// GetStatus returns the status of all migrations in this registry
func (r *Registry) GetStatus(ctx context.Context, db *sql.DB) ([]MigrationStatus, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection cannot be nil")
	}

	// Get applied migration records from database
	appliedRecords, err := r.getAppliedRecords(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("failed to get applied migrations: %w", err)
	}

	// Create a map of applied migrations for quick lookup
	appliedMap := make(map[int64]MigrationRecord)
	for _, record := range appliedRecords {
		appliedMap[record.Version] = record
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	// Build status for all migrations
	statuses := make([]MigrationStatus, 0, len(r.migrations))
	for _, migration := range r.migrations {
		status := MigrationStatus{
			Version: migration.Version(),
			Name:    migration.Name(),
			Applied: false,
		}

		if record, ok := appliedMap[migration.Version()]; ok {
			status.Applied = true
			status.AppliedAt = record.AppliedAt
		}

		statuses = append(statuses, status)
	}

	// Sort by version
	sort.Slice(statuses, func(i, j int) bool {
		return statuses[i].Version < statuses[j].Version
	})

	return statuses, nil
}

// getAppliedVersions returns a set of applied migration versions
func (r *Registry) getAppliedVersions(ctx context.Context, db *sql.DB) (map[int64]bool, error) {
	// Check if schema_migrations table exists
	var tableExists bool
	query := `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='schema_migrations'`
	if err := db.QueryRowContext(ctx, query).Scan(&tableExists); err != nil {
		return nil, fmt.Errorf("failed to check schema_migrations table: %w", err)
	}

	if !tableExists {
		// No migrations have been applied yet
		return make(map[int64]bool), nil
	}

	// Get all applied versions
	rows, err := db.QueryContext(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, fmt.Errorf("failed to query schema_migrations: %w", err)
	}
	defer rows.Close()

	versions := make(map[int64]bool)
	for rows.Next() {
		var version int64
		if err := rows.Scan(&version); err != nil {
			return nil, fmt.Errorf("failed to scan version: %w", err)
		}
		versions[version] = true
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating versions: %w", err)
	}

	return versions, nil
}

// getAppliedRecords returns all applied migration records from the database
func (r *Registry) getAppliedRecords(ctx context.Context, db *sql.DB) ([]MigrationRecord, error) {
	// Check if schema_migrations table exists
	var tableExists bool
	query := `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='schema_migrations'`
	if err := db.QueryRowContext(ctx, query).Scan(&tableExists); err != nil {
		return nil, fmt.Errorf("failed to check schema_migrations table: %w", err)
	}

	if !tableExists {
		// No migrations have been applied yet
		return []MigrationRecord{}, nil
	}

	// Get all applied migration records
	rows, err := db.QueryContext(ctx, `SELECT version, name, applied_at FROM schema_migrations ORDER BY version ASC`)
	if err != nil {
		return nil, fmt.Errorf("failed to query schema_migrations: %w", err)
	}
	defer rows.Close()

	records := make([]MigrationRecord, 0)
	for rows.Next() {
		var record MigrationRecord
		if err := rows.Scan(&record.Version, &record.Name, &record.AppliedAt); err != nil {
			return nil, fmt.Errorf("failed to scan migration record: %w", err)
		}
		records = append(records, record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating migration records: %w", err)
	}

	return records, nil
}

// Clear removes all migrations from the global registry
// This is primarily useful for testing
func Clear() {
	globalRegistry.Clear()
}

// Clear removes all migrations from this registry
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.migrations = make(map[int64]Migration)
}

// NewRegistry creates a new empty registry
// Most users should use the global registry functions instead
func NewRegistry() *Registry {
	return &Registry{
		migrations: make(map[int64]Migration),
		mu:         sync.RWMutex{},
	}
}
