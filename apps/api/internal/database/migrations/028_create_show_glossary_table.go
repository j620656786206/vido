package migrations

import "database/sql"

func init() {
	Register(&createShowGlossaryTable{
		migrationBase: NewMigrationBase(28, "create_show_glossary_table"),
	})
}

type createShowGlossaryTable struct {
	migrationBase
}

func (m *createShowGlossaryTable) Up(tx *sql.Tx) error {
	// Story 9R-6 — Epic 9R (Subtitle Route C) keystone: the per-show glossary
	// that fixes proper-noun drift across generation runs (隱形戰士/隱形特務;
	// "The Deep"→深海怪物 when the model lacks the character roster). One
	// infra serves BOTH subtitle translation (9R-7) AND .nfo metadata
	// localization (9R-13) — ADR adr-subtitle-route-c-generation Decision 3/6.
	//
	// media_id is the local movie/series id (string; movie int ids stringified)
	// — the identifier the generation pipeline (9R-10) and the detail 管理字幕
	// surface + the 9R-15 REST routes (/{movies|series}/:id/glossary) hold.
	// source records provenance so the review UI (F6) can show where a term
	// came from; confirmed gates whether an auto-mined term is trusted.
	if _, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS show_glossary (
			id TEXT PRIMARY KEY,
			media_id TEXT NOT NULL,
			term_src TEXT NOT NULL,
			term_zh TEXT NOT NULL,
			language TEXT NOT NULL DEFAULT 'zh-Hant',
			source TEXT NOT NULL DEFAULT 'manual'
				CHECK(source IN ('subtitle','metadata','manual')),
			confirmed INTEGER NOT NULL DEFAULT 0,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`); err != nil {
		return err
	}

	// Uniqueness on (media, source term, language): a given English term maps
	// to exactly one zh rendering per show+language. An auto-mined term that
	// re-appears UPSERTs rather than duplicating (repository ON CONFLICT).
	if _, err := tx.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_show_glossary_unique
		ON show_glossary(media_id, term_src, language)`); err != nil {
		return err
	}
	// Lookup-by-media is the hot path (translation injection + F6 list).
	if _, err := tx.Exec(`
		CREATE INDEX IF NOT EXISTS idx_show_glossary_media
		ON show_glossary(media_id)`); err != nil {
		return err
	}
	return nil
}

func (m *createShowGlossaryTable) Down(tx *sql.Tx) error {
	_, err := tx.Exec(`DROP TABLE IF EXISTS show_glossary`)
	return err
}
