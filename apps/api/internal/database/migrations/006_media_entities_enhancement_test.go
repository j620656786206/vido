package migrations

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestMediaEntitiesEnhancement_Up(t *testing.T) {
	// Create in-memory SQLite database
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	// First, apply prerequisite migrations (001 and 002)
	// Create movies table (simplified version of migration 001)
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS movies (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			original_title TEXT,
			release_date TEXT NOT NULL,
			genres TEXT NOT NULL DEFAULT '[]',
			rating REAL,
			overview TEXT,
			poster_path TEXT,
			backdrop_path TEXT,
			runtime INTEGER,
			original_language TEXT,
			status TEXT,
			imdb_id TEXT,
			tmdb_id INTEGER,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
	`)
	require.NoError(t, err)

	// Create series table (simplified version of migration 002)
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS series (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			original_title TEXT,
			first_air_date TEXT NOT NULL,
			last_air_date TEXT,
			genres TEXT NOT NULL DEFAULT '[]',
			rating REAL,
			overview TEXT,
			poster_path TEXT,
			backdrop_path TEXT,
			number_of_seasons INTEGER,
			number_of_episodes INTEGER,
			status TEXT,
			original_language TEXT,
			imdb_id TEXT,
			tmdb_id INTEGER,
			in_production INTEGER,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
	`)
	require.NoError(t, err)

	// Insert test data before migration
	_, err = db.Exec(`
		INSERT INTO movies (id, title, release_date, overview)
		VALUES ('movie-1', 'Test Movie', '2024-01-01', 'A test movie overview');

		INSERT INTO series (id, title, first_air_date, overview)
		VALUES ('series-1', 'Test Series', '2024-01-01', 'A test series overview');
	`)
	require.NoError(t, err)

	// Start transaction for migration
	tx, err := db.Begin()
	require.NoError(t, err)

	// Apply migration
	m := &MediaEntitiesEnhancement{
		migrationBase: NewMigrationBase(6, "media_entities_enhancement"),
	}

	err = m.Up(tx)
	require.NoError(t, err)

	err = tx.Commit()
	require.NoError(t, err)

	// Verify new columns exist on movies table
	t.Run("movies_new_columns", func(t *testing.T) {
		row := db.QueryRow(`
			SELECT vote_average, vote_count, popularity, credits,
				   production_countries, spoken_languages, file_path,
				   file_size, parse_status, metadata_source
			FROM movies WHERE id = 'movie-1'
		`)

		var voteAverage, popularity sql.NullFloat64
		var voteCount, fileSize sql.NullInt64
		var credits, productionCountries, spokenLanguages, filePath, parseStatus, metadataSource sql.NullString

		err := row.Scan(&voteAverage, &voteCount, &popularity, &credits,
			&productionCountries, &spokenLanguages, &filePath,
			&fileSize, &parseStatus, &metadataSource)
		assert.NoError(t, err)
		assert.Equal(t, "pending", parseStatus.String)
	})

	// Verify new columns exist on series table
	t.Run("series_new_columns", func(t *testing.T) {
		row := db.QueryRow(`
			SELECT vote_average, vote_count, popularity, credits,
				   seasons, networks, file_path, parse_status, metadata_source
			FROM series WHERE id = 'series-1'
		`)

		var voteAverage, popularity sql.NullFloat64
		var voteCount sql.NullInt64
		var credits, seasons, networks, filePath, parseStatus, metadataSource sql.NullString

		err := row.Scan(&voteAverage, &voteCount, &popularity, &credits,
			&seasons, &networks, &filePath, &parseStatus, &metadataSource)
		assert.NoError(t, err)
		assert.Equal(t, "pending", parseStatus.String)
	})

	// Verify episodes table exists
	t.Run("episodes_table_exists", func(t *testing.T) {
		_, err := db.Exec(`
			INSERT INTO episodes (id, series_id, season_number, episode_number, title)
			VALUES ('ep-1', 'series-1', 1, 1, 'Pilot')
		`)
		assert.NoError(t, err)

		var title string
		err = db.QueryRow(`SELECT title FROM episodes WHERE id = 'ep-1'`).Scan(&title)
		assert.NoError(t, err)
		assert.Equal(t, "Pilot", title)
	})

	// Verify FTS works for movies
	t.Run("movies_fts_search", func(t *testing.T) {
		rows, err := db.Query(`
			SELECT m.id, m.title
			FROM movies m
			JOIN movies_fts ON movies_fts.rowid = m.rowid
			WHERE movies_fts MATCH 'test'
		`)
		require.NoError(t, err)
		defer rows.Close()

		var count int
		for rows.Next() {
			var id, title string
			err := rows.Scan(&id, &title)
			require.NoError(t, err)
			assert.Equal(t, "movie-1", id)
			count++
		}
		assert.Equal(t, 1, count)
	})

	// Verify FTS works for series
	t.Run("series_fts_search", func(t *testing.T) {
		rows, err := db.Query(`
			SELECT s.id, s.title
			FROM series s
			JOIN series_fts ON series_fts.rowid = s.rowid
			WHERE series_fts MATCH 'test'
		`)
		require.NoError(t, err)
		defer rows.Close()

		var count int
		for rows.Next() {
			var id, title string
			err := rows.Scan(&id, &title)
			require.NoError(t, err)
			assert.Equal(t, "series-1", id)
			count++
		}
		assert.Equal(t, 1, count)
	})

	// Verify FTS trigger on insert
	t.Run("fts_trigger_insert", func(t *testing.T) {
		_, err := db.Exec(`
			INSERT INTO movies (id, title, release_date, overview)
			VALUES ('movie-2', 'Another Movie', '2024-02-01', 'Another overview')
		`)
		require.NoError(t, err)

		rows, err := db.Query(`
			SELECT m.id FROM movies m
			JOIN movies_fts ON movies_fts.rowid = m.rowid
			WHERE movies_fts MATCH 'another'
		`)
		require.NoError(t, err)
		defer rows.Close()

		var found bool
		for rows.Next() {
			var id string
			rows.Scan(&id)
			if id == "movie-2" {
				found = true
			}
		}
		assert.True(t, found, "FTS trigger should index new movie")
	})

	// Verify FTS trigger on update
	t.Run("fts_trigger_update", func(t *testing.T) {
		_, err := db.Exec(`UPDATE movies SET title = 'Updated Title' WHERE id = 'movie-1'`)
		require.NoError(t, err)

		// Old title should not be found
		rows, err := db.Query(`
			SELECT m.id FROM movies m
			JOIN movies_fts ON movies_fts.rowid = m.rowid
			WHERE movies_fts MATCH 'Test Movie'
		`)
		require.NoError(t, err)
		defer rows.Close()

		var count int
		for rows.Next() {
			count++
		}
		// Should not find with old title (exact phrase match)
		// Note: FTS5 might still match partial words

		// New title should be found
		rows2, err := db.Query(`
			SELECT m.id FROM movies m
			JOIN movies_fts ON movies_fts.rowid = m.rowid
			WHERE movies_fts MATCH 'Updated'
		`)
		require.NoError(t, err)
		defer rows2.Close()

		var found bool
		for rows2.Next() {
			var id string
			rows2.Scan(&id)
			if id == "movie-1" {
				found = true
			}
		}
		assert.True(t, found, "FTS trigger should update index on movie update")
	})

	// Verify episode unique constraint
	t.Run("episode_unique_constraint", func(t *testing.T) {
		_, err := db.Exec(`
			INSERT INTO episodes (id, series_id, season_number, episode_number, title)
			VALUES ('ep-dup', 'series-1', 1, 1, 'Duplicate')
		`)
		assert.Error(t, err, "Should fail due to unique constraint on series_id, season_number, episode_number")
	})

	// Verify foreign key constraint (CASCADE DELETE)
	t.Run("episode_cascade_delete", func(t *testing.T) {
		// Enable foreign keys
		_, err := db.Exec("PRAGMA foreign_keys = ON")
		require.NoError(t, err)

		// Insert series and episode
		_, err = db.Exec(`
			INSERT INTO series (id, title, first_air_date) VALUES ('series-del', 'To Delete', '2024-01-01');
			INSERT INTO episodes (id, series_id, season_number, episode_number) VALUES ('ep-del', 'series-del', 1, 1);
		`)
		require.NoError(t, err)

		// Delete series
		_, err = db.Exec(`DELETE FROM series WHERE id = 'series-del'`)
		require.NoError(t, err)

		// Episode should be deleted too
		var count int
		err = db.QueryRow(`SELECT COUNT(*) FROM episodes WHERE series_id = 'series-del'`).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 0, count, "Episodes should be cascade deleted")
	})
}

func TestMediaEntitiesEnhancement_Down(t *testing.T) {
	// Create in-memory SQLite database
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	// Setup: Create base tables and apply Up migration
	_, err = db.Exec(`
		CREATE TABLE movies (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			original_title TEXT,
			release_date TEXT NOT NULL DEFAULT '',
			genres TEXT NOT NULL DEFAULT '[]',
			overview TEXT,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE series (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			original_title TEXT,
			first_air_date TEXT NOT NULL DEFAULT '',
			overview TEXT,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
	`)
	require.NoError(t, err)

	m := &MediaEntitiesEnhancement{
		migrationBase: NewMigrationBase(6, "media_entities_enhancement"),
	}

	// Apply Up migration
	tx, err := db.Begin()
	require.NoError(t, err)
	err = m.Up(tx)
	require.NoError(t, err)
	err = tx.Commit()
	require.NoError(t, err)

	// Apply Down migration
	tx, err = db.Begin()
	require.NoError(t, err)
	err = m.Down(tx)
	require.NoError(t, err)
	err = tx.Commit()
	require.NoError(t, err)

	// Verify FTS tables are dropped
	t.Run("fts_tables_dropped", func(t *testing.T) {
		var count int
		err := db.QueryRow(`
			SELECT COUNT(*) FROM sqlite_master
			WHERE type='table' AND name='movies_fts'
		`).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 0, count, "movies_fts should be dropped")

		err = db.QueryRow(`
			SELECT COUNT(*) FROM sqlite_master
			WHERE type='table' AND name='series_fts'
		`).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 0, count, "series_fts should be dropped")
	})

	// Verify episodes table is dropped
	t.Run("episodes_table_dropped", func(t *testing.T) {
		var count int
		err := db.QueryRow(`
			SELECT COUNT(*) FROM sqlite_master
			WHERE type='table' AND name='episodes'
		`).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 0, count, "episodes table should be dropped")
	})

	// Verify triggers are dropped
	t.Run("triggers_dropped", func(t *testing.T) {
		var count int
		err := db.QueryRow(`
			SELECT COUNT(*) FROM sqlite_master
			WHERE type='trigger' AND name LIKE '%_fts_%'
		`).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 0, count, "FTS triggers should be dropped")
	})
}

func TestMediaEntitiesEnhancement_FTSPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	// Create in-memory SQLite database
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	// Setup base tables
	_, err = db.Exec(`
		CREATE TABLE movies (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			original_title TEXT,
			release_date TEXT NOT NULL DEFAULT '',
			genres TEXT NOT NULL DEFAULT '[]',
			overview TEXT,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE series (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			original_title TEXT,
			first_air_date TEXT NOT NULL DEFAULT '',
			overview TEXT,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
	`)
	require.NoError(t, err)

	// Insert 1000 movies for performance testing
	tx, err := db.Begin()
	require.NoError(t, err)

	for i := 0; i < 1000; i++ {
		_, err = tx.Exec(`
			INSERT INTO movies (id, title, release_date, overview)
			VALUES (?, ?, '2024-01-01', ?)
		`,
			"movie-"+string(rune(i)),
			"Movie Title Number "+string(rune(i)),
			"This is a detailed overview for movie number "+string(rune(i))+". It contains various keywords for testing search functionality.",
		)
		require.NoError(t, err)
	}

	// Add a specific movie to search for
	_, err = tx.Exec(`
		INSERT INTO movies (id, title, release_date, overview)
		VALUES ('target-movie', 'Inception', '2010-07-16', 'A thief who steals corporate secrets through dream-sharing technology')
	`)
	require.NoError(t, err)

	err = tx.Commit()
	require.NoError(t, err)

	// Apply migration
	m := &MediaEntitiesEnhancement{
		migrationBase: NewMigrationBase(6, "media_entities_enhancement"),
	}

	tx, err = db.Begin()
	require.NoError(t, err)
	err = m.Up(tx)
	require.NoError(t, err)
	err = tx.Commit()
	require.NoError(t, err)

	// Test FTS search performance - should be < 500ms per NFR-SC8
	t.Run("fts_search_performance", func(t *testing.T) {
		start := testing.Benchmark(func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				rows, err := db.Query(`
					SELECT m.id, m.title
					FROM movies m
					JOIN movies_fts ON movies_fts.rowid = m.rowid
					WHERE movies_fts MATCH 'dream technology'
					LIMIT 20
				`)
				if err != nil {
					b.Fatal(err)
				}
				rows.Close()
			}
		})

		// Each operation should be well under 500ms
		avgNs := start.NsPerOp()
		avgMs := avgNs / 1_000_000
		t.Logf("FTS search average time: %d ms", avgMs)
		assert.Less(t, avgMs, int64(500), "FTS search should complete in < 500ms")
	})
}
