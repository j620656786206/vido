package migrations

import (
	"database/sql"
	"fmt"
)

// CreateSeriesTable is the migration to create the series table
type CreateSeriesTable struct {
	migrationBase
}

func init() {
	// Register this migration with the global registry
	Register(&CreateSeriesTable{
		migrationBase: NewMigrationBase(2, "create_series_table"),
	})
}

// Up creates the series table
func (m *CreateSeriesTable) Up(tx *sql.Tx) error {
	query := `
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

		CREATE INDEX IF NOT EXISTS idx_series_title ON series(title);
		CREATE INDEX IF NOT EXISTS idx_series_tmdb_id ON series(tmdb_id);
		CREATE INDEX IF NOT EXISTS idx_series_imdb_id ON series(imdb_id);
		CREATE INDEX IF NOT EXISTS idx_series_first_air_date ON series(first_air_date);
	`

	if _, err := tx.Exec(query); err != nil {
		return fmt.Errorf("failed to create series table: %w", err)
	}

	return nil
}

// Down drops the series table
func (m *CreateSeriesTable) Down(tx *sql.Tx) error {
	query := `
		DROP INDEX IF EXISTS idx_series_first_air_date;
		DROP INDEX IF EXISTS idx_series_imdb_id;
		DROP INDEX IF EXISTS idx_series_tmdb_id;
		DROP INDEX IF EXISTS idx_series_title;
		DROP TABLE IF EXISTS series;
	`

	if _, err := tx.Exec(query); err != nil {
		return fmt.Errorf("failed to drop series table: %w", err)
	}

	return nil
}
