package migrations

import (
	"database/sql"
	"strings"
	"time"
)

func init() {
	Register(&addSubtitleRunSpendAndCompletedAtIndexes{
		migrationBase: NewMigrationBase(32, "add_subtitle_run_spend_and_completed_at_indexes"),
	})
}

// addSubtitleRunSpendAndCompletedAtIndexes persists per-run AI spend, indexes
// the terminal timestamps, and normalizes parse_jobs' historical local-offset
// timestamps to UTC (Story ux3-1-6, tech-spec D2/D3).
//
// spent_usd/budget_usd are NULLABLE on purpose: NULL means "this run predates
// spend recording (or ran on the legacy path)" — absent is not $0, and the
// home-summary endpoint treats them differently. The values come from the
// per-item ai.Budget delta at every terminal transition (completed, failed,
// skipped — cost was incurred regardless of outcome).
//
// The timestamp normalization is the load-bearing half: the modernc driver
// stores time.Time as Go's String() text ("2026-08-26 04:21:10.590294 +0000
// UTC"), which SQLite's datetime() cannot parse — so the ONLY correct
// comparison on these columns is lexicographic, and that is only correct when
// every row carries the SAME offset. subtitle_runs always wrote UTC
// (subtitleRunValues normalizes); parse_jobs wrote time.Now() in server-local
// time, so its historical rows are rewritten to UTC here and its writers are
// UTC from this story on. The plain indexes then serve both the >= window
// filter and ORDER BY.
type addSubtitleRunSpendAndCompletedAtIndexes struct {
	migrationBase
}

func (m *addSubtitleRunSpendAndCompletedAtIndexes) Up(tx *sql.Tx) error {
	if !columnExists(tx, "subtitle_runs", "spent_usd") {
		if _, err := tx.Exec("ALTER TABLE subtitle_runs ADD COLUMN spent_usd REAL"); err != nil {
			return err
		}
	}
	if !columnExists(tx, "subtitle_runs", "budget_usd") {
		if _, err := tx.Exec("ALTER TABLE subtitle_runs ADD COLUMN budget_usd REAL"); err != nil {
			return err
		}
	}
	if err := normalizeParseJobTimestampsToUTC(tx); err != nil {
		return err
	}
	if _, err := tx.Exec("CREATE INDEX IF NOT EXISTS idx_subtitle_runs_completed_at ON subtitle_runs (completed_at)"); err != nil {
		return err
	}
	if _, err := tx.Exec("CREATE INDEX IF NOT EXISTS idx_parse_jobs_completed_at ON parse_jobs (completed_at)"); err != nil {
		return err
	}
	return nil
}

func (m *addSubtitleRunSpendAndCompletedAtIndexes) Down(tx *sql.Tx) error {
	// The columns default to NULL and are harmless if left in place; SQLite
	// DROP COLUMN support is version-dependent (mirrors migrations 024/026/031).
	// The UTC normalization is not reverted — UTC text is strictly more correct.
	if _, err := tx.Exec("DROP INDEX IF EXISTS idx_subtitle_runs_completed_at"); err != nil {
		return err
	}
	if _, err := tx.Exec("DROP INDEX IF EXISTS idx_parse_jobs_completed_at"); err != nil {
		return err
	}
	return nil
}

// goTimeTextLayouts are the text shapes a time.Time column can hold in this
// database: the modernc driver's Go String() form, and RFC3339 as a defensive
// extra. Unparseable text is left untouched (better a stale row than a lost
// timestamp).
var goTimeTextLayouts = []string{
	"2006-01-02 15:04:05.999999999 -0700 MST",
	"2006-01-02 15:04:05.999999999 -0700",
	time.RFC3339Nano,
}

func normalizeParseJobTimestampsToUTC(tx *sql.Tx) error {
	rows, err := tx.Query(`SELECT id,
		CAST(created_at AS TEXT), CAST(updated_at AS TEXT), CAST(IFNULL(completed_at, '') AS TEXT)
		FROM parse_jobs`)
	if err != nil {
		return err
	}
	defer rows.Close()

	type fix struct {
		id                                   string
		createdAt, updatedAt                 *time.Time
		completedAt                          *time.Time
		hasCreated, hasUpdated, hasCompleted bool
	}
	var fixes []fix
	for rows.Next() {
		var id, created, updated, completed string
		if err := rows.Scan(&id, &created, &updated, &completed); err != nil {
			return err
		}
		f := fix{id: id}
		f.createdAt, f.hasCreated = parseNonUTCTimeText(created)
		f.updatedAt, f.hasUpdated = parseNonUTCTimeText(updated)
		f.completedAt, f.hasCompleted = parseNonUTCTimeText(completed)
		if f.hasCreated || f.hasUpdated || f.hasCompleted {
			fixes = append(fixes, f)
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for _, f := range fixes {
		if f.hasCreated {
			if _, err := tx.Exec(`UPDATE parse_jobs SET created_at = ? WHERE id = ?`, f.createdAt.UTC(), f.id); err != nil {
				return err
			}
		}
		if f.hasUpdated {
			if _, err := tx.Exec(`UPDATE parse_jobs SET updated_at = ? WHERE id = ?`, f.updatedAt.UTC(), f.id); err != nil {
				return err
			}
		}
		if f.hasCompleted {
			if _, err := tx.Exec(`UPDATE parse_jobs SET completed_at = ? WHERE id = ?`, f.completedAt.UTC(), f.id); err != nil {
				return err
			}
		}
	}
	return nil
}

// parseNonUTCTimeText parses a stored time text and reports whether it needs a
// UTC rewrite (parseable AND not already at a +0000 offset).
func parseNonUTCTimeText(text string) (*time.Time, bool) {
	if text == "" {
		return nil, false
	}
	if strings.Contains(text, "+0000") || strings.HasSuffix(text, "Z") {
		return nil, false // already UTC text — rewriting would only churn rows
	}
	for _, layout := range goTimeTextLayouts {
		if ts, err := time.Parse(layout, text); err == nil {
			return &ts, true
		}
	}
	return nil, false
}
