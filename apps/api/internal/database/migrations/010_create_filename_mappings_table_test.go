package migrations

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestCreateFilenameMappingsTable_Up(t *testing.T) {
	// Create in-memory SQLite database
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	// Enable foreign keys
	_, err = db.Exec("PRAGMA foreign_keys = ON")
	require.NoError(t, err)

	// Create prerequisite movies table (migration 001)
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS movies (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			original_title TEXT,
			release_date TEXT NOT NULL,
			genres TEXT NOT NULL DEFAULT '[]',
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
	`)
	require.NoError(t, err)

	// Create prerequisite series table (migration 002)
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS series (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			original_title TEXT,
			first_air_date TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
	`)
	require.NoError(t, err)

	// Insert test data for foreign key tests
	_, err = db.Exec(`
		INSERT INTO movies (id, title, release_date)
		VALUES ('movie-123', 'Demon Slayer: Mugen Train', '2020-10-16');

		INSERT INTO series (id, title, first_air_date)
		VALUES ('series-456', 'Demon Slayer', '2019-04-06');
	`)
	require.NoError(t, err)

	// Start transaction for migration
	tx, err := db.Begin()
	require.NoError(t, err)

	// Apply migration
	m := &CreateFilenameMappingsTable{
		migrationBase: NewMigrationBase(10, "create_filename_mappings_table"),
	}

	err = m.Up(tx)
	require.NoError(t, err)

	err = tx.Commit()
	require.NoError(t, err)

	// Verify table was created with correct columns
	t.Run("table_exists_with_columns", func(t *testing.T) {
		// Insert a test pattern
		_, err := db.Exec(`
			INSERT INTO filename_mappings (
				id, pattern, pattern_type, pattern_regex, fansub_group,
				title_pattern, metadata_type, metadata_id, tmdb_id, confidence
			) VALUES (
				'pattern-1', '[Leopard-Raws] Kimetsu no Yaiba', 'fansub',
				'\\[Leopard-Raws\\]\\s*Kimetsu no Yaiba.*', 'Leopard-Raws',
				'Kimetsu no Yaiba', 'series', 'series-456', 85937, 1.0
			)
		`)
		require.NoError(t, err)

		// Query the inserted pattern
		var id, pattern, patternType, fansubGroup, titlePattern, metadataType, metadataID string
		var tmdbID int
		var confidence float64
		var patternRegex sql.NullString
		var useCount int
		var createdAt, lastUsedAt sql.NullString

		err = db.QueryRow(`
			SELECT id, pattern, pattern_type, pattern_regex, fansub_group,
				   title_pattern, metadata_type, metadata_id, tmdb_id,
				   confidence, use_count, created_at, last_used_at
			FROM filename_mappings WHERE id = 'pattern-1'
		`).Scan(&id, &pattern, &patternType, &patternRegex, &fansubGroup,
			&titlePattern, &metadataType, &metadataID, &tmdbID,
			&confidence, &useCount, &createdAt, &lastUsedAt)
		require.NoError(t, err)

		assert.Equal(t, "pattern-1", id)
		assert.Equal(t, "[Leopard-Raws] Kimetsu no Yaiba", pattern)
		assert.Equal(t, "fansub", patternType)
		assert.True(t, patternRegex.Valid)
		assert.Equal(t, "Leopard-Raws", fansubGroup)
		assert.Equal(t, "Kimetsu no Yaiba", titlePattern)
		assert.Equal(t, "series", metadataType)
		assert.Equal(t, "series-456", metadataID)
		assert.Equal(t, 85937, tmdbID)
		assert.Equal(t, 1.0, confidence)
		assert.Equal(t, 0, useCount) // Default value
		assert.True(t, createdAt.Valid)
		assert.False(t, lastUsedAt.Valid) // Should be NULL initially
	})

	// Verify indexes exist
	t.Run("indexes_exist", func(t *testing.T) {
		// Check for pattern index
		var count int
		err := db.QueryRow(`
			SELECT COUNT(*) FROM sqlite_master
			WHERE type='index' AND name='idx_filename_mappings_pattern'
		`).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 1, count, "idx_filename_mappings_pattern index should exist")

		// Check for fansub_group index
		err = db.QueryRow(`
			SELECT COUNT(*) FROM sqlite_master
			WHERE type='index' AND name='idx_filename_mappings_fansub_group'
		`).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 1, count, "idx_filename_mappings_fansub_group index should exist")

		// Check for title_pattern index
		err = db.QueryRow(`
			SELECT COUNT(*) FROM sqlite_master
			WHERE type='index' AND name='idx_filename_mappings_title_pattern'
		`).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 1, count, "idx_filename_mappings_title_pattern index should exist")

		// Check for metadata lookup index
		err = db.QueryRow(`
			SELECT COUNT(*) FROM sqlite_master
			WHERE type='index' AND name='idx_filename_mappings_metadata'
		`).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 1, count, "idx_filename_mappings_metadata index should exist")
	})

	// Verify default values work
	t.Run("default_values", func(t *testing.T) {
		// Insert minimal pattern (only required fields)
		_, err := db.Exec(`
			INSERT INTO filename_mappings (
				id, pattern, pattern_type, metadata_type, metadata_id
			) VALUES (
				'pattern-2', 'Test Pattern', 'exact', 'movie', 'movie-123'
			)
		`)
		require.NoError(t, err)

		// Check defaults were applied
		var confidence float64
		var useCount int
		err = db.QueryRow(`
			SELECT confidence, use_count FROM filename_mappings WHERE id = 'pattern-2'
		`).Scan(&confidence, &useCount)
		require.NoError(t, err)

		assert.Equal(t, 1.0, confidence, "default confidence should be 1.0")
		assert.Equal(t, 0, useCount, "default use_count should be 0")
	})

	// Verify pattern type constraint (exact, regex, fuzzy)
	t.Run("pattern_type_values", func(t *testing.T) {
		// Valid pattern types
		validTypes := []string{"exact", "regex", "fuzzy"}
		for i, pt := range validTypes {
			_, err := db.Exec(`
				INSERT INTO filename_mappings (id, pattern, pattern_type, metadata_type, metadata_id)
				VALUES (?, ?, ?, 'movie', 'movie-123')
			`, "pattern-type-"+pt, "Test "+pt, pt)
			assert.NoError(t, err, "pattern_type '%s' should be valid", pt)
			_ = i
		}
	})

	// Verify use_count can be incremented
	t.Run("use_count_increment", func(t *testing.T) {
		_, err := db.Exec(`
			UPDATE filename_mappings
			SET use_count = use_count + 1, last_used_at = CURRENT_TIMESTAMP
			WHERE id = 'pattern-1'
		`)
		require.NoError(t, err)

		var useCount int
		var lastUsedAt sql.NullString
		err = db.QueryRow(`
			SELECT use_count, last_used_at FROM filename_mappings WHERE id = 'pattern-1'
		`).Scan(&useCount, &lastUsedAt)
		require.NoError(t, err)

		assert.Equal(t, 1, useCount)
		assert.True(t, lastUsedAt.Valid)
	})

	// Verify unique pattern constraint
	t.Run("unique_pattern", func(t *testing.T) {
		// First insert should succeed
		_, err := db.Exec(`
			INSERT INTO filename_mappings (id, pattern, pattern_type, metadata_type, metadata_id)
			VALUES ('unique-1', 'Unique Pattern Test', 'exact', 'movie', 'movie-123')
		`)
		require.NoError(t, err)

		// Duplicate pattern should fail
		_, err = db.Exec(`
			INSERT INTO filename_mappings (id, pattern, pattern_type, metadata_type, metadata_id)
			VALUES ('unique-2', 'Unique Pattern Test', 'exact', 'movie', 'movie-123')
		`)
		assert.Error(t, err, "duplicate pattern should fail unique constraint")
	})
}

func TestCreateFilenameMappingsTable_Down(t *testing.T) {
	// Create in-memory SQLite database
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	// Create prerequisite tables
	_, err = db.Exec(`
		CREATE TABLE movies (id TEXT PRIMARY KEY);
		CREATE TABLE series (id TEXT PRIMARY KEY);
	`)
	require.NoError(t, err)

	m := &CreateFilenameMappingsTable{
		migrationBase: NewMigrationBase(10, "create_filename_mappings_table"),
	}

	// Apply Up migration
	tx, err := db.Begin()
	require.NoError(t, err)
	err = m.Up(tx)
	require.NoError(t, err)
	err = tx.Commit()
	require.NoError(t, err)

	// Verify table exists before Down
	var count int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM sqlite_master
		WHERE type='table' AND name='filename_mappings'
	`).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count, "filename_mappings table should exist before Down")

	// Apply Down migration
	tx, err = db.Begin()
	require.NoError(t, err)
	err = m.Down(tx)
	require.NoError(t, err)
	err = tx.Commit()
	require.NoError(t, err)

	// Verify table is dropped
	t.Run("table_dropped", func(t *testing.T) {
		var count int
		err := db.QueryRow(`
			SELECT COUNT(*) FROM sqlite_master
			WHERE type='table' AND name='filename_mappings'
		`).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 0, count, "filename_mappings table should be dropped")
	})

	// Verify indexes are dropped
	t.Run("indexes_dropped", func(t *testing.T) {
		var count int
		err := db.QueryRow(`
			SELECT COUNT(*) FROM sqlite_master
			WHERE type='index' AND name LIKE 'idx_filename_mappings_%'
		`).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 0, count, "all filename_mappings indexes should be dropped")
	})
}

func TestCreateFilenameMappingsTable_Version(t *testing.T) {
	m := &CreateFilenameMappingsTable{
		migrationBase: NewMigrationBase(10, "create_filename_mappings_table"),
	}

	assert.Equal(t, int64(10), m.Version())
	assert.Equal(t, "create_filename_mappings_table", m.Name())
}
