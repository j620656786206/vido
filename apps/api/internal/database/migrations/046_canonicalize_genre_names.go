package migrations

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/vido/api/internal/models"
)

func init() {
	Register(&canonicalizeGenreNames{
		migrationBase: NewMigrationBase(46, "canonicalize_genre_names"),
	})
}

// canonicalizeGenreNames (disc-2026-10-genre-names-simplified AC #3) rewrites
// stored genre names in the Taiwan wording. TMDb's zh-TW genre names are
// Simplified (剧情, 动作冒险) and two TV genres are English only (Sci-Fi &
// Fantasy), so the boot backfill of 2026-10-12 filled the NAS library with
// them. New writes are canonical (models.Movie.GenresJSON); this fixes what
// is already stored, with the same function, so the two cannot disagree.
//
// Every row, the user's own included: the names only change spelling (剧情 →
// 劇情), and the Metadata Editor offers the Taiwan names anyway. updated_at is
// left alone — not an edit of the title. A row whose JSON does not parse is
// skipped and logged; it never fails the migration.
type canonicalizeGenreNames struct {
	migrationBase
}

func (m *canonicalizeGenreNames) Up(tx *sql.Tx) error {
	total := 0
	for _, table := range []string{"movies", "series"} {
		n, err := canonicalizeGenreColumn(tx, table)
		if err != nil {
			return err
		}
		total += n
	}
	if total > 0 {
		slog.Info("migration 046: genre names rewritten in the Taiwan wording", "rows", total)
	}
	return nil
}

func canonicalizeGenreColumn(tx *sql.Tx, table string) (int, error) {
	rows, err := tx.Query(`SELECT id, genres FROM ` + table + ` WHERE genres NOT IN ('[]', '')`)
	if err != nil {
		return 0, fmt.Errorf("list %s genres: %w", table, err)
	}
	type change struct{ id, genres string }
	var changes []change
	for rows.Next() {
		var id, stored string
		if err := rows.Scan(&id, &stored); err != nil {
			rows.Close()
			return 0, fmt.Errorf("scan %s genres: %w", table, err)
		}
		var names []string
		if err := json.Unmarshal([]byte(stored), &names); err != nil {
			slog.Warn("migration 046: skipping malformed genres", "table", table, "id", id, "genres", stored, "error", err)
			continue
		}
		canonical, err := json.Marshal(models.CanonicalGenres(names))
		if err != nil {
			rows.Close()
			return 0, fmt.Errorf("encode %s genres: %w", table, err)
		}
		if string(canonical) != stored {
			changes = append(changes, change{id, string(canonical)})
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}

	for _, c := range changes {
		if _, err := tx.Exec(`UPDATE `+table+` SET genres = ? WHERE id = ?`, c.genres, c.id); err != nil {
			return 0, fmt.Errorf("rewrite %s genres: %w", table, err)
		}
	}
	return len(changes), nil
}

func (m *canonicalizeGenreNames) Down(tx *sql.Tx) error {
	// A spelling fix; the Simplified names are not worth restoring.
	return nil
}
