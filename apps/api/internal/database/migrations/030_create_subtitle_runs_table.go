package migrations

import "database/sql"

func init() {
	Register(&createSubtitleRunsTable{
		migrationBase: NewMigrationBase(30, "create_subtitle_runs_table"),
	})
}

type createSubtitleRunsTable struct {
	migrationBase
}

func (m *createSubtitleRunsTable) Up(tx *sql.Tx) error {
	// Story sub-1-2 — subtitle pipeline (M1) state model, architecture D2.
	//
	// D2 splits pipeline state by concern: `subtitle_status` on the media row
	// stays the single answer to "what is this item's subtitle state", and this
	// table holds only what has no place there — TRANSLATION PROVENANCE (which
	// inputs produced a given subtitle) plus per-run detail. Cue-grain identity
	// is NOT stored here; it lives in the tiered cache keyed by hash(cue) +
	// RunVersion (D4, sub-1-5). Two storage mechanisms for one concept is the
	// exact anti-pattern D2 rejects.
	//
	// The four version columns are the RunVersion tuple. Every one is mandatory
	// even when empty: omitting prompt/model would let a re-run silently reuse a
	// prior translation, making two prompt variants look identical during the M1
	// pilot with no error surfacing. glossary_version is always '' in M1 (the
	// glossary is P2) and is versioned NOW per D4.
	//
	// media_type uses the INTERNAL table vocabulary ('movie'|'series'|'episode'),
	// deliberately NOT requests.media_type's TMDB 'movie'|'tv' (migration 027) —
	// provenance is keyed to our own rows, and `subtitle_status` lives on all
	// three tables (migrations 018 + 025), so all three grains are representable.
	//
	// output_path is nullable because P9 mandates provenance is written only
	// AFTER a successful place; cache_enabled is RESERVED for sub-1-5 (D4's
	// "disable caching for this show and record it" ruling) and stays 0 here.
	//
	// NOTE: movies/series/episodes are deliberately NOT touched. Their
	// subtitle_status columns are plain `TEXT DEFAULT 'not_searched'` with NO
	// CHECK constraint (migrations 018:18,25 and 025:26), so the extended enum
	// needs zero DDL — it is a Go + frontend contract. SQLite has no
	// ALTER TABLE ADD CONSTRAINT and emulating one means a full table rebuild:
	// enormous risk for zero requirement.
	if _, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS subtitle_runs (
			id TEXT PRIMARY KEY,
			media_id TEXT NOT NULL,
			media_type TEXT NOT NULL CHECK(media_type IN ('movie','series','episode')),
			tmdb_id INTEGER,
			metadata_hash TEXT NOT NULL DEFAULT '',
			glossary_version TEXT NOT NULL DEFAULT '',
			prompt_version TEXT NOT NULL DEFAULT '',
			model_id TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'pending'
				CHECK(status IN ('pending','running','completed','failed','skipped')),
			source_language TEXT,
			output_path TEXT,
			cue_count INTEGER,
			cache_enabled INTEGER NOT NULL DEFAULT 0,
			error_message TEXT,
			started_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			completed_at TIMESTAMP
		)`); err != nil {
		return err
	}

	// Resume lookup (NFR-R3) is the hot path: "is there a completed run for this
	// media whose version tuple matches?"
	if _, err := tx.Exec(`
		CREATE INDEX IF NOT EXISTS idx_subtitle_runs_media
		ON subtitle_runs(media_id, media_type)`); err != nil {
		return err
	}
	// Batch enumeration by run state (sub-1-6).
	if _, err := tx.Exec(`
		CREATE INDEX IF NOT EXISTS idx_subtitle_runs_status
		ON subtitle_runs(status)`); err != nil {
		return err
	}
	return nil
}

func (m *createSubtitleRunsTable) Down(tx *sql.Tx) error {
	// Nothing was added to an existing table, so dropping this one is the whole
	// revert (mirrors 027). Indexes go with it.
	_, err := tx.Exec(`DROP TABLE IF EXISTS subtitle_runs`)
	return err
}
