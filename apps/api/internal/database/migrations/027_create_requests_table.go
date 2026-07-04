package migrations

import "database/sql"

func init() {
	Register(&createRequestsTable{
		migrationBase: NewMigrationBase(27, "create_requests_table"),
	})
}

type createRequestsTable struct {
	migrationBase
}

func (m *createRequestsTable) Up(tx *sql.Tx) error {
	// Story 13-1a — Epic 13 (Request System) data foundation (G-1/P3-001).
	// The 5-value status CHECK is the single source of truth the request
	// pipeline (13-3a) and the frontend render against. seasons/episodes stay
	// NULL until 13-2a; fulfilment_source/external_id are written by 13-4.
	// media_type uses the TMDB/frontend vocabulary ('movie'|'tv'), deliberately
	// NOT media_libraries.content_type's 'series' (that classifies folders).
	if _, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS requests (
			id TEXT PRIMARY KEY,
			tmdb_id INTEGER NOT NULL,
			media_type TEXT NOT NULL CHECK(media_type IN ('movie','tv')),
			title TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending'
				CHECK(status IN ('pending','searching','downloading','completed','failed')),
			fulfilment_source TEXT CHECK(fulfilment_source IN ('arr','builtin')),
			external_id TEXT,
			seasons TEXT,
			episodes TEXT,
			error_message TEXT,
			requested_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`); err != nil {
		return err
	}

	if _, err := tx.Exec(`CREATE INDEX IF NOT EXISTS idx_requests_status ON requests(status)`); err != nil {
		return err
	}
	if _, err := tx.Exec(`CREATE INDEX IF NOT EXISTS idx_requests_tmdb_id ON requests(tmdb_id)`); err != nil {
		return err
	}
	// Duplicate-request guard at the DB level (race safety behind the service
	// pre-check). Partial index: only ACTIVE requests block a re-request;
	// completed/failed rows never do — a failed request must be retryable.
	if _, err := tx.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_requests_active_unique
		ON requests(tmdb_id, media_type)
		WHERE status IN ('pending','searching','downloading')`); err != nil {
		return err
	}
	return nil
}

func (m *createRequestsTable) Down(tx *sql.Tx) error {
	_, err := tx.Exec(`DROP TABLE IF EXISTS requests`)
	return err
}
