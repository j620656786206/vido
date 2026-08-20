package migrations

import "database/sql"

func init() {
	Register(&addMediaLibraryAutoSubtitle{
		migrationBase: NewMigrationBase(31, "add_media_library_auto_subtitle"),
	})
}

// addMediaLibraryAutoSubtitle adds the per-library opt-in for free subtitle
// auto-generation (Story 9R-10b AC #2).
//
// DEFAULT 0 is the whole point, not a formality: the 2026-08-07 cost incident
// came from automation that nobody switched on. Every existing library, and
// every library created before someone reads the setting, stays OFF.
//
// A column on media_libraries rather than a settings JSON key so that saving the
// library edit form is ONE write — a two-resource save (library + settings) can
// half-fail and leave the UI claiming a state the backend does not hold. It
// mirrors the shipped `auto_detect` boolean exactly (migration 020).
//
// The ALTER is idempotent (guarded by columnExists) and self-registers via
// init() — no registry.go edit (Story 12-2 migration-registration correction).
type addMediaLibraryAutoSubtitle struct {
	migrationBase
}

func (m *addMediaLibraryAutoSubtitle) Up(tx *sql.Tx) error {
	if columnExists(tx, "media_libraries", "auto_subtitle") {
		return nil
	}
	if _, err := tx.Exec("ALTER TABLE media_libraries ADD COLUMN auto_subtitle INTEGER NOT NULL DEFAULT 0"); err != nil {
		return err
	}
	return nil
}

func (m *addMediaLibraryAutoSubtitle) Down(tx *sql.Tx) error {
	// The column defaults to 0 and is harmless if left in place; SQLite
	// DROP COLUMN support is version-dependent (mirrors migrations 024/026).
	return nil
}
