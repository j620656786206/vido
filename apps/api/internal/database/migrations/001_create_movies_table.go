package migrations

import (
	"database/sql"
	"fmt"
)

// CreateMoviesTable is the migration to create the movies table
type CreateMoviesTable struct {
	migrationBase
}

func init() {
	// Register this migration with the global registry
	Register(&CreateMoviesTable{
		migrationBase: NewMigrationBase(1, "create_movies_table"),
	})
}

// Up creates the movies table
func (m *CreateMoviesTable) Up(tx *sql.Tx) error {
	query := `
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

		CREATE INDEX IF NOT EXISTS idx_movies_title ON movies(title);
		CREATE INDEX IF NOT EXISTS idx_movies_tmdb_id ON movies(tmdb_id);
		CREATE INDEX IF NOT EXISTS idx_movies_imdb_id ON movies(imdb_id);
		CREATE INDEX IF NOT EXISTS idx_movies_release_date ON movies(release_date);
	`

	if _, err := tx.Exec(query); err != nil {
		return fmt.Errorf("failed to create movies table: %w", err)
	}

	return nil
}

// Down drops the movies table
func (m *CreateMoviesTable) Down(tx *sql.Tx) error {
	query := `
		DROP INDEX IF EXISTS idx_movies_release_date;
		DROP INDEX IF EXISTS idx_movies_imdb_id;
		DROP INDEX IF EXISTS idx_movies_tmdb_id;
		DROP INDEX IF EXISTS idx_movies_title;
		DROP TABLE IF EXISTS movies;
	`

	if _, err := tx.Exec(query); err != nil {
		return fmt.Errorf("failed to drop movies table: %w", err)
	}

	return nil
}
