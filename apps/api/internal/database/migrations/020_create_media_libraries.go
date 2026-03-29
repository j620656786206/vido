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
	_, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS media_libraries (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			content_type TEXT NOT NULL CHECK(content_type IN ('movie', 'series')),
			auto_detect INTEGER NOT NULL DEFAULT 0,
			sort_order INTEGER NOT NULL DEFAULT 0,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS media_library_paths (
			id TEXT PRIMARY KEY,
			library_id TEXT NOT NULL,
			path TEXT NOT NULL UNIQUE,
			status TEXT NOT NULL DEFAULT 'unknown' CHECK(status IN ('unknown', 'accessible', 'not_found', 'not_readable', 'not_directory')),
			last_checked_at TIMESTAMP,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (library_id) REFERENCES media_libraries(id) ON DELETE CASCADE
		);

		CREATE INDEX IF NOT EXISTS idx_media_library_paths_library_id ON media_library_paths(library_id);
		CREATE INDEX IF NOT EXISTS idx_media_library_paths_path ON media_library_paths(path);

		-- Add library_id to movies (Story 7b-1)
		ALTER TABLE movies ADD COLUMN library_id TEXT REFERENCES media_libraries(id);

		-- Add library_id to series (Story 7b-1)
		ALTER TABLE series ADD COLUMN library_id TEXT REFERENCES media_libraries(id);

		-- Route 2 reserve fields: auto-detection support (Phase 2)
		ALTER TABLE movies ADD COLUMN detected_type TEXT;
		ALTER TABLE movies ADD COLUMN override_type TEXT;
		ALTER TABLE series ADD COLUMN detected_type TEXT;
		ALTER TABLE series ADD COLUMN override_type TEXT;

		CREATE INDEX IF NOT EXISTS idx_movies_library_id ON movies(library_id);
		CREATE INDEX IF NOT EXISTS idx_series_library_id ON series(library_id);
	`)
	return err
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
