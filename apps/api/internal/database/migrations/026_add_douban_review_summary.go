package migrations

import "database/sql"

func init() {
	Register(&addDoubanReviewSummary{
		migrationBase: NewMigrationBase(26, "add_douban_review_summary"),
	})
}

// addDoubanReviewSummary adds the review_summary_json column to douban_cache so the
// Douban short-comment summary (Story 12-6) is cached alongside the detail row,
// keyed by the same douban_id under the existing 7-day TTL. The ALTER is idempotent
// (guarded by columnExists) and self-registers via init() — no registry.go edit
// (per the Story 12-2 migration-registration correction).
type addDoubanReviewSummary struct {
	migrationBase
}

func (m *addDoubanReviewSummary) Up(tx *sql.Tx) error {
	if columnExists(tx, "douban_cache", "review_summary_json") {
		return nil
	}
	if _, err := tx.Exec("ALTER TABLE douban_cache ADD COLUMN review_summary_json TEXT"); err != nil {
		return err
	}
	return nil
}

func (m *addDoubanReviewSummary) Down(tx *sql.Tx) error {
	// The column is nullable and harmless if left in place; SQLite DROP COLUMN
	// support is version-dependent (mirrors migration 024's Down).
	return nil
}
