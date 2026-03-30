package migrations

import "database/sql"

func init() {
	Register(&createMediaLibraries{
		migrationBase: NewMigrationBase(20, "create_media_libraries"),
	})
}

type createMediaLibraries struct {
	migrationBase
}

func (m *createMediaLibraries) Up(tx *sql.Tx) error {
	// Create media_libraries table (Story 7b-1)
	if _, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS media_libraries (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			content_type TEXT NOT NULL CHECK(content_type IN ('movie', 'series')),
			auto_detect INTEGER NOT NULL DEFAULT 0,
			sort_order INTEGER NOT NULL DEFAULT 0,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`); err != nil {
		return err
	}

	if _, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS media_library_paths (
			id TEXT PRIMARY KEY,
			library_id TEXT NOT NULL,
			path TEXT NOT NULL UNIQUE,
			status TEXT NOT NULL DEFAULT 'unknown' CHECK(status IN ('unknown', 'accessible', 'not_found', 'not_readable', 'not_directory')),
			last_checked_at TIMESTAMP,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (library_id) REFERENCES media_libraries(id) ON DELETE CASCADE
		)`); err != nil {
		return err
	}

	if _, err := tx.Exec(`CREATE INDEX IF NOT EXISTS idx_media_library_paths_library_id ON media_library_paths(library_id)`); err != nil {
		return err
	}
	if _, err := tx.Exec(`CREATE INDEX IF NOT EXISTS idx_media_library_paths_path ON media_library_paths(path)`); err != nil {
		return err
	}

	// Add columns to movies and series — use helper to skip if column already exists
	alterColumns := []struct {
		table  string
		column string
		ddl    string
	}{
		{"movies", "library_id", "ALTER TABLE movies ADD COLUMN library_id TEXT REFERENCES media_libraries(id)"},
		{"series", "library_id", "ALTER TABLE series ADD COLUMN library_id TEXT REFERENCES media_libraries(id)"},
		{"movies", "detected_type", "ALTER TABLE movies ADD COLUMN detected_type TEXT"},
		{"movies", "override_type", "ALTER TABLE movies ADD COLUMN override_type TEXT"},
		{"series", "detected_type", "ALTER TABLE series ADD COLUMN detected_type TEXT"},
		{"series", "override_type", "ALTER TABLE series ADD COLUMN override_type TEXT"},
	}
	for _, ac := range alterColumns {
		if !columnExists(tx, ac.table, ac.column) {
			if _, err := tx.Exec(ac.ddl); err != nil {
				return err
			}
		}
	}

	if _, err := tx.Exec(`CREATE INDEX IF NOT EXISTS idx_movies_library_id ON movies(library_id)`); err != nil {
		return err
	}
	if _, err := tx.Exec(`CREATE INDEX IF NOT EXISTS idx_series_library_id ON series(library_id)`); err != nil {
		return err
	}

	return nil
}

// columnExists checks if a column exists in a SQLite table.
func columnExists(tx *sql.Tx, table, column string) bool {
	rows, err := tx.Query("PRAGMA table_info(" + table + ")")
	if err != nil {
		return false
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull int
		var dfltValue sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk); err != nil {
			return false
		}
		if name == column {
			return true
		}
	}
	return false
}

func (m *createMediaLibraries) Down(tx *sql.Tx) error {
	// SQLite does not support DROP COLUMN before 3.35.0
	// Drop new tables only
	_, err := tx.Exec(`
		DROP TABLE IF EXISTS media_library_paths;
		DROP TABLE IF EXISTS media_libraries;
	`)
	return err
}
