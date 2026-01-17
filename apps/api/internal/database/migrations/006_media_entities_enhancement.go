package migrations

import (
	"database/sql"
	"fmt"
)

// MediaEntitiesEnhancement adds FTS5 support and new fields to movies/series,
// creates episodes table with foreign key to series
type MediaEntitiesEnhancement struct {
	migrationBase
}

func init() {
	Register(&MediaEntitiesEnhancement{
		migrationBase: NewMigrationBase(6, "media_entities_enhancement"),
	})
}

// Up applies the media entities enhancement migration
func (m *MediaEntitiesEnhancement) Up(tx *sql.Tx) error {
	// Step 1: Add new columns to movies table
	moviesAlterQueries := []string{
		`ALTER TABLE movies ADD COLUMN vote_average REAL`,
		`ALTER TABLE movies ADD COLUMN vote_count INTEGER`,
		`ALTER TABLE movies ADD COLUMN popularity REAL`,
		`ALTER TABLE movies ADD COLUMN credits TEXT`,
		`ALTER TABLE movies ADD COLUMN production_countries TEXT`,
		`ALTER TABLE movies ADD COLUMN spoken_languages TEXT`,
		`ALTER TABLE movies ADD COLUMN file_path TEXT`,
		`ALTER TABLE movies ADD COLUMN file_size INTEGER`,
		`ALTER TABLE movies ADD COLUMN parse_status TEXT DEFAULT 'pending'`,
		`ALTER TABLE movies ADD COLUMN metadata_source TEXT`,
	}

	for _, query := range moviesAlterQueries {
		if _, err := tx.Exec(query); err != nil {
			return fmt.Errorf("failed to alter movies table: %w", err)
		}
	}

	// Step 2: Add new columns to series table
	seriesAlterQueries := []string{
		`ALTER TABLE series ADD COLUMN vote_average REAL`,
		`ALTER TABLE series ADD COLUMN vote_count INTEGER`,
		`ALTER TABLE series ADD COLUMN popularity REAL`,
		`ALTER TABLE series ADD COLUMN credits TEXT`,
		`ALTER TABLE series ADD COLUMN seasons TEXT`,
		`ALTER TABLE series ADD COLUMN networks TEXT`,
		`ALTER TABLE series ADD COLUMN file_path TEXT`,
		`ALTER TABLE series ADD COLUMN parse_status TEXT DEFAULT 'pending'`,
		`ALTER TABLE series ADD COLUMN metadata_source TEXT`,
	}

	for _, query := range seriesAlterQueries {
		if _, err := tx.Exec(query); err != nil {
			return fmt.Errorf("failed to alter series table: %w", err)
		}
	}

	// Step 3: Create episodes table
	episodesQuery := `
		CREATE TABLE IF NOT EXISTS episodes (
			id TEXT PRIMARY KEY,
			series_id TEXT NOT NULL,
			tmdb_id INTEGER,
			season_number INTEGER NOT NULL,
			episode_number INTEGER NOT NULL,
			title TEXT,
			overview TEXT,
			air_date TEXT,
			runtime INTEGER,
			still_path TEXT,
			vote_average REAL,
			file_path TEXT,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (series_id) REFERENCES series(id) ON DELETE CASCADE,
			UNIQUE(series_id, season_number, episode_number)
		);

		CREATE INDEX IF NOT EXISTS idx_episodes_series_id ON episodes(series_id);
		CREATE INDEX IF NOT EXISTS idx_episodes_tmdb_id ON episodes(tmdb_id);
		CREATE INDEX IF NOT EXISTS idx_episodes_season ON episodes(series_id, season_number);
	`

	if _, err := tx.Exec(episodesQuery); err != nil {
		return fmt.Errorf("failed to create episodes table: %w", err)
	}

	// Step 4: Create indexes for new columns
	indexQueries := []string{
		`CREATE INDEX IF NOT EXISTS idx_movies_parse_status ON movies(parse_status)`,
		`CREATE INDEX IF NOT EXISTS idx_movies_file_path ON movies(file_path)`,
		`CREATE INDEX IF NOT EXISTS idx_series_parse_status ON series(parse_status)`,
		`CREATE INDEX IF NOT EXISTS idx_series_file_path ON series(file_path)`,
	}

	for _, query := range indexQueries {
		if _, err := tx.Exec(query); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	// Step 5: Create FTS5 virtual table for movies
	// Note: We need to drop the existing title index to avoid conflicts
	moviesFTSQuery := `
		CREATE VIRTUAL TABLE IF NOT EXISTS movies_fts USING fts5(
			title,
			original_title,
			overview,
			content='movies',
			content_rowid='rowid'
		);

		-- Populate FTS table with existing data
		INSERT INTO movies_fts(rowid, title, original_title, overview)
		SELECT rowid, title, original_title, overview FROM movies;

		-- Trigger: Insert into FTS when movie is inserted
		CREATE TRIGGER IF NOT EXISTS movies_fts_ai AFTER INSERT ON movies BEGIN
			INSERT INTO movies_fts(rowid, title, original_title, overview)
			VALUES (NEW.rowid, NEW.title, NEW.original_title, NEW.overview);
		END;

		-- Trigger: Delete from FTS when movie is deleted
		CREATE TRIGGER IF NOT EXISTS movies_fts_ad AFTER DELETE ON movies BEGIN
			INSERT INTO movies_fts(movies_fts, rowid, title, original_title, overview)
			VALUES ('delete', OLD.rowid, OLD.title, OLD.original_title, OLD.overview);
		END;

		-- Trigger: Update FTS when movie is updated
		CREATE TRIGGER IF NOT EXISTS movies_fts_au AFTER UPDATE ON movies BEGIN
			INSERT INTO movies_fts(movies_fts, rowid, title, original_title, overview)
			VALUES ('delete', OLD.rowid, OLD.title, OLD.original_title, OLD.overview);
			INSERT INTO movies_fts(rowid, title, original_title, overview)
			VALUES (NEW.rowid, NEW.title, NEW.original_title, NEW.overview);
		END;
	`

	if _, err := tx.Exec(moviesFTSQuery); err != nil {
		return fmt.Errorf("failed to create movies FTS5: %w", err)
	}

	// Step 6: Create FTS5 virtual table for series
	seriesFTSQuery := `
		CREATE VIRTUAL TABLE IF NOT EXISTS series_fts USING fts5(
			title,
			original_title,
			overview,
			content='series',
			content_rowid='rowid'
		);

		-- Populate FTS table with existing data
		INSERT INTO series_fts(rowid, title, original_title, overview)
		SELECT rowid, title, original_title, overview FROM series;

		-- Trigger: Insert into FTS when series is inserted
		CREATE TRIGGER IF NOT EXISTS series_fts_ai AFTER INSERT ON series BEGIN
			INSERT INTO series_fts(rowid, title, original_title, overview)
			VALUES (NEW.rowid, NEW.title, NEW.original_title, NEW.overview);
		END;

		-- Trigger: Delete from FTS when series is deleted
		CREATE TRIGGER IF NOT EXISTS series_fts_ad AFTER DELETE ON series BEGIN
			INSERT INTO series_fts(series_fts, rowid, title, original_title, overview)
			VALUES ('delete', OLD.rowid, OLD.title, OLD.original_title, OLD.overview);
		END;

		-- Trigger: Update FTS when series is updated
		CREATE TRIGGER IF NOT EXISTS series_fts_au AFTER UPDATE ON series BEGIN
			INSERT INTO series_fts(series_fts, rowid, title, original_title, overview)
			VALUES ('delete', OLD.rowid, OLD.title, OLD.original_title, OLD.overview);
			INSERT INTO series_fts(rowid, title, original_title, overview)
			VALUES (NEW.rowid, NEW.title, NEW.original_title, NEW.overview);
		END;
	`

	if _, err := tx.Exec(seriesFTSQuery); err != nil {
		return fmt.Errorf("failed to create series FTS5: %w", err)
	}

	// Step 7: Make tmdb_id unique (if not already)
	// SQLite doesn't support ALTER TABLE ADD CONSTRAINT, so we skip this
	// The initial migration should have UNIQUE constraint

	return nil
}

// Down reverts the media entities enhancement migration
func (m *MediaEntitiesEnhancement) Down(tx *sql.Tx) error {
	// Drop FTS triggers
	dropTriggersQueries := []string{
		`DROP TRIGGER IF EXISTS movies_fts_ai`,
		`DROP TRIGGER IF EXISTS movies_fts_ad`,
		`DROP TRIGGER IF EXISTS movies_fts_au`,
		`DROP TRIGGER IF EXISTS series_fts_ai`,
		`DROP TRIGGER IF EXISTS series_fts_ad`,
		`DROP TRIGGER IF EXISTS series_fts_au`,
	}

	for _, query := range dropTriggersQueries {
		if _, err := tx.Exec(query); err != nil {
			return fmt.Errorf("failed to drop trigger: %w", err)
		}
	}

	// Drop FTS tables
	dropFTSQueries := []string{
		`DROP TABLE IF EXISTS movies_fts`,
		`DROP TABLE IF EXISTS series_fts`,
	}

	for _, query := range dropFTSQueries {
		if _, err := tx.Exec(query); err != nil {
			return fmt.Errorf("failed to drop FTS table: %w", err)
		}
	}

	// Drop episodes table
	if _, err := tx.Exec(`DROP TABLE IF EXISTS episodes`); err != nil {
		return fmt.Errorf("failed to drop episodes table: %w", err)
	}

	// Drop indexes for new columns
	dropIndexQueries := []string{
		`DROP INDEX IF EXISTS idx_movies_parse_status`,
		`DROP INDEX IF EXISTS idx_movies_file_path`,
		`DROP INDEX IF EXISTS idx_series_parse_status`,
		`DROP INDEX IF EXISTS idx_series_file_path`,
		`DROP INDEX IF EXISTS idx_episodes_series_id`,
		`DROP INDEX IF EXISTS idx_episodes_tmdb_id`,
		`DROP INDEX IF EXISTS idx_episodes_season`,
	}

	for _, query := range dropIndexQueries {
		if _, err := tx.Exec(query); err != nil {
			return fmt.Errorf("failed to drop index: %w", err)
		}
	}

	// Note: SQLite doesn't support DROP COLUMN, so we can't remove the added columns
	// In a real migration scenario, you would recreate the table without those columns

	return nil
}
