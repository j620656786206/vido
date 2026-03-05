package migrations

import (
	"database/sql"
	"fmt"
)

// CreateSeasonsTable is the migration to create the seasons table
// and add season_id FK to episodes table (Action Item #3: Series/Season/Episode 3-tier)
type CreateSeasonsTable struct {
	migrationBase
}

func init() {
	Register(&CreateSeasonsTable{
		migrationBase: NewMigrationBase(15, "create_seasons_table"),
	})
}

// Up creates the seasons table and adds season_id to episodes
func (m *CreateSeasonsTable) Up(tx *sql.Tx) error {
	query := `
		CREATE TABLE IF NOT EXISTS seasons (
			id TEXT PRIMARY KEY,
			series_id TEXT NOT NULL,
			tmdb_id INTEGER,
			season_number INTEGER NOT NULL,
			name TEXT,
			overview TEXT,
			poster_path TEXT,
			air_date TEXT,
			episode_count INTEGER,
			vote_average REAL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (series_id) REFERENCES series(id) ON DELETE CASCADE,
			UNIQUE(series_id, season_number)
		);

		CREATE INDEX IF NOT EXISTS idx_seasons_series_id ON seasons(series_id);
		CREATE INDEX IF NOT EXISTS idx_seasons_tmdb_id ON seasons(tmdb_id);

		ALTER TABLE episodes ADD COLUMN season_id TEXT REFERENCES seasons(id) ON DELETE SET NULL;
		CREATE INDEX IF NOT EXISTS idx_episodes_season_id ON episodes(season_id);
	`

	if _, err := tx.Exec(query); err != nil {
		return fmt.Errorf("failed to create seasons table: %w", err)
	}

	return nil
}

// Down drops the seasons table and removes season_id from episodes
func (m *CreateSeasonsTable) Down(tx *sql.Tx) error {
	// SQLite 3.35.0+ supports ALTER TABLE DROP COLUMN
	query := `
		DROP INDEX IF EXISTS idx_episodes_season_id;
		ALTER TABLE episodes DROP COLUMN season_id;
		DROP INDEX IF EXISTS idx_seasons_tmdb_id;
		DROP INDEX IF EXISTS idx_seasons_series_id;
		DROP TABLE IF EXISTS seasons;
	`

	if _, err := tx.Exec(query); err != nil {
		return fmt.Errorf("failed to drop seasons table: %w", err)
	}

	return nil
}
