package migrations

import (
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
)

func init() {
	Register(&rebuildShowGlossaryWithScope{
		migrationBase: NewMigrationBase(36, "rebuild_show_glossary_with_scope"),
	})
}

// rebuildShowGlossaryWithScope re-keys the per-show glossary from this
// machine's row id to a world-wide id (sub-7-1 AC #1, following eval-1's
// 架構評估 verbatim).
//
// Why a rebuild and not ALTER TABLE ADD COLUMN: three things have to change
// at once and SQLite can do none of them in place —
//
//  1. `scope TEXT NOT NULL` becomes the read key (`tmdb:tv:<id>` /
//     `tmdb:movie:<id>` / `local:<media_id>`), with the unique index moved
//     onto `(scope, term_src COLLATE NOCASE, language)`. NOCASE because
//     `Demogorgon` and `demogorgon` were two rows; on one machine that is a
//     smudge, merged across machines (sub-8) it is a wall of duplicates.
//  2. the `source` CHECK is dropped — the enum lives in
//     models.GlossaryTerm.Validate now, so `official_subtitle` (sub-7-5) and
//     `community` (sub-8-1) are a Go constant each, not another rebuild.
//  3. `term_src` is trimmed on the way over.
//
// `media_id` stays for ONE migration as an audit trail of which local row
// wrote each term; the next schema story removes it (Rule 24 superseded-
// mechanism corollary — do not build anything new on that column).
//
// Backfill: a row whose media id is a series/movie with a TMDb id lands in
// that `tmdb:*` scope; everything else lands in `local:<media_id>`, which is
// exactly the pre-036 behaviour under a new name. When the NOCASE/trim rules
// collapse two old rows into one key the EARLIEST row wins and the loser is
// logged by id — a deliberate, visible loss, never a silent overwrite.
type rebuildShowGlossaryWithScope struct {
	migrationBase
}

func (m *rebuildShowGlossaryWithScope) Up(tx *sql.Tx) error {
	if columnExists(tx, "show_glossary", "scope") {
		return nil // already rebuilt — a second Up must be a no-op
	}

	if _, err := tx.Exec(`
		CREATE TABLE show_glossary_v2 (
			id TEXT PRIMARY KEY,
			media_id TEXT NOT NULL,
			scope TEXT NOT NULL,
			term_src TEXT NOT NULL,
			term_zh TEXT NOT NULL,
			language TEXT NOT NULL DEFAULT 'zh-Hant',
			source TEXT NOT NULL DEFAULT 'manual',
			confirmed INTEGER NOT NULL DEFAULT 0,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`); err != nil {
		return fmt.Errorf("create show_glossary_v2: %w", err)
	}
	// The unique index goes on BEFORE the copy so the copy is validated by it.
	if _, err := tx.Exec(`
		CREATE UNIQUE INDEX idx_show_glossary_scope_unique
		ON show_glossary_v2(scope, term_src COLLATE NOCASE, language)`); err != nil {
		return fmt.Errorf("create scope unique index: %w", err)
	}

	// Resolve every old row's scope in ONE pass (LEFT JOINs, no per-row
	// queries). Order = the tie-break for NOCASE collisions: first created,
	// then rowid — "the earlier entry is the one the user has been living with".
	rows, err := tx.Query(`
		SELECT g.id, g.media_id, g.term_src, g.term_zh, g.language, g.source, g.confirmed,
		       g.created_at, g.updated_at,
		       CASE
		         WHEN s.tmdb_id IS NOT NULL THEN 'tmdb:tv:' || s.tmdb_id
		         WHEN mv.tmdb_id IS NOT NULL THEN 'tmdb:movie:' || mv.tmdb_id
		         ELSE 'local:' || g.media_id
		       END AS scope
		FROM show_glossary g
		LEFT JOIN series s ON s.id = g.media_id
		LEFT JOIN movies mv ON mv.id = g.media_id
		ORDER BY g.created_at ASC, g.rowid ASC`)
	if err != nil {
		return fmt.Errorf("read show_glossary for backfill: %w", err)
	}
	type oldRow struct {
		id, mediaID, termSrc, termZh, language, source string
		confirmed                                      int
		createdAt, updatedAt                           any
		scope                                          string
	}
	var old []oldRow
	for rows.Next() {
		var r oldRow
		if err := rows.Scan(&r.id, &r.mediaID, &r.termSrc, &r.termZh, &r.language, &r.source,
			&r.confirmed, &r.createdAt, &r.updatedAt, &r.scope); err != nil {
			rows.Close()
			return fmt.Errorf("scan show_glossary row: %w", err)
		}
		old = append(old, r)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("iterate show_glossary: %w", err)
	}
	rows.Close()

	insert, err := tx.Prepare(`
		INSERT INTO show_glossary_v2
			(id, media_id, scope, term_src, term_zh, language, source, confirmed, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("prepare backfill insert: %w", err)
	}
	defer insert.Close()

	type key struct{ scope, term, language string }
	seen := make(map[key]string, len(old)) // key → surviving id
	var moved, shared, dropped int
	for _, r := range old {
		term := strings.TrimSpace(r.termSrc)
		k := key{r.scope, strings.ToLower(term), r.language}
		if winner, dup := seen[k]; dup {
			dropped++
			slog.Warn("migration 036: glossary term collapsed by the NOCASE/trim rule — earlier row kept",
				"kept_id", winner, "dropped_id", r.id, "scope", r.scope, "term_src", r.termSrc, "language", r.language)
			continue
		}
		seen[k] = r.id
		if _, err := insert.Exec(r.id, r.mediaID, r.scope, term, r.termZh, r.language, r.source,
			r.confirmed, r.createdAt, r.updatedAt); err != nil {
			return fmt.Errorf("backfill glossary term %s: %w", r.id, err)
		}
		moved++
		if strings.HasPrefix(r.scope, "tmdb:") {
			shared++
		}
	}

	for _, stmt := range []string{
		`DROP INDEX IF EXISTS idx_show_glossary_unique`,
		`DROP INDEX IF EXISTS idx_show_glossary_media`,
		`DROP TABLE show_glossary`,
		`ALTER TABLE show_glossary_v2 RENAME TO show_glossary`,
		// Lookup-by-scope is the hot path (translation feed + F6 list); the
		// media_id index only serves the audit column while it still exists.
		`CREATE INDEX IF NOT EXISTS idx_show_glossary_scope ON show_glossary(scope)`,
		`CREATE INDEX IF NOT EXISTS idx_show_glossary_media ON show_glossary(media_id)`,
	} {
		if _, err := tx.Exec(stmt); err != nil {
			return fmt.Errorf("%s: %w", stmt, err)
		}
	}
	slog.Info("migration 036: show_glossary re-keyed by scope",
		"rows", moved, "shared_tmdb_rows", shared, "local_rows", moved-shared, "collapsed_duplicates", dropped)
	return nil
}

// Down restores the 028 shape (media_id key, three-value source CHECK). Rows
// whose scope, source or case-folded term cannot be expressed in the old
// schema are folded or dropped — this is a schema rollback, not an undo.
func (m *rebuildShowGlossaryWithScope) Down(tx *sql.Tx) error {
	if !columnExists(tx, "show_glossary", "scope") {
		return nil
	}
	stmts := []string{
		`CREATE TABLE show_glossary_v1 (
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
		)`,
		`CREATE UNIQUE INDEX idx_show_glossary_unique ON show_glossary_v1(media_id, term_src, language)`,
		`INSERT OR IGNORE INTO show_glossary_v1
			(id, media_id, term_src, term_zh, language, source, confirmed, created_at, updated_at)
		 SELECT id, media_id, term_src, term_zh, language,
		        CASE WHEN source IN ('subtitle','metadata','manual') THEN source ELSE 'manual' END,
		        confirmed, created_at, updated_at
		 FROM show_glossary`,
		`DROP INDEX IF EXISTS idx_show_glossary_scope_unique`,
		`DROP INDEX IF EXISTS idx_show_glossary_scope`,
		`DROP INDEX IF EXISTS idx_show_glossary_media`,
		`DROP TABLE show_glossary`,
		`ALTER TABLE show_glossary_v1 RENAME TO show_glossary`,
		`CREATE INDEX IF NOT EXISTS idx_show_glossary_media ON show_glossary(media_id)`,
	}
	for _, stmt := range stmts {
		if _, err := tx.Exec(stmt); err != nil {
			return fmt.Errorf("036 down: %w", err)
		}
	}
	return nil
}
