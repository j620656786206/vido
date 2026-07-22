package migrations

import "database/sql"

func init() {
	Register(&dropNewShellEnabledSetting{
		migrationBase: NewMigrationBase(29, "drop_new_shell_enabled_setting"),
	})
}

type dropNewShellEnabledSetting struct {
	migrationBase
}

func (m *dropNewShellEnabledSetting) Up(tx *sql.Tx) error {
	// ux3-cutover-4 — flag retirement (ADR D1-c). The legacy shell is deleted and
	// nothing reads `new_shell_enabled` anymore (the `__root.tsx` chokepoint and
	// the cutover-1 startup force-ON are both gone), so the stored row is dead
	// data on every environment that ever seeded it (NAS + local DBs).
	_, err := tx.Exec(`DELETE FROM settings WHERE key = 'new_shell_enabled'`)
	return err
}

func (m *dropNewShellEnabledSetting) Down(tx *sql.Tx) error {
	// Restore the post-cutover-1 state: the flag existed and was forced ON.
	_, err := tx.Exec(`
		INSERT INTO settings (key, value, type, created_at, updated_at)
		VALUES ('new_shell_enabled', 'true', 'bool', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT(key) DO UPDATE SET value = 'true', updated_at = CURRENT_TIMESTAMP`)
	return err
}
