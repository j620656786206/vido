package migrations

import "database/sql"

func init() {
	Register(&retryUndecidedBurnedInChecks{
		migrationBase: NewMigrationBase(50, "retry_undecided_burned_in_checks"),
	})
}

// retryUndecidedBurnedInChecks (disc-2026-11-burned-in-skip-missing-files) is a
// one-time data fix. The first production sweep (2026-11-02) could not judge
// 《仙逆》S1 because it sampled a row whose file had been removed; it stamped
// burned_in_checked_at on that season, so the sweep would not look again for 7
// days. The service now skips missing files — clearing the stamp on seasons
// that were stamped WITHOUT a verdict lets the next round retry them at once.
// Seasons that have a verdict are untouched.
type retryUndecidedBurnedInChecks struct {
	migrationBase
}

func (m *retryUndecidedBurnedInChecks) Up(tx *sql.Tx) error {
	_, err := tx.Exec(`UPDATE episodes SET burned_in_checked_at = NULL
		WHERE burned_in_subtitle IS NULL AND burned_in_checked_at IS NOT NULL`)
	return err
}

// Down is a no-op: the cleared stamps only said "tried, could not tell", and
// the sweep writes them again on its next attempt.
func (m *retryUndecidedBurnedInChecks) Down(tx *sql.Tx) error { return nil }
