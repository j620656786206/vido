package migrations

import (
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"unicode"
)

func init() {
	Register(&fixReversedHarvestedGlossaryTerms{
		migrationBase: NewMigrationBase(45, "fix_reversed_harvested_glossary_terms"),
	})
}

// fixReversedHarvestedGlossaryTerms (disc-2026-10-mine-accented-names-truncated
// AC #4) repairs glossary rows the translation trailer wrote backwards: the
// model answered `馬言者=>The Neigh-sayer`, the parser stored 馬言者 as the
// English term and the English as the "rendering". The NAS held two such
// rows on 2026-10-08. The parser now turns those pairs around
// (services.orientHarvestPair); this fixes what it already wrote.
//
// Only harvested (source='subtitle'), unconfirmed rows whose term_src has
// Chinese and whose term_zh has none. Turned around in place — unless the
// right-way-round term already exists in the same scope and language, in
// which case the backwards row is the duplicate and is deleted. A confirmed
// row is the user's own decision and is never touched.
type fixReversedHarvestedGlossaryTerms struct {
	migrationBase
}

func (m *fixReversedHarvestedGlossaryTerms) Up(tx *sql.Tx) error {
	rows, err := tx.Query(`SELECT id, scope, term_src, term_zh, language FROM show_glossary
		WHERE source = 'subtitle' AND confirmed = 0`)
	if err != nil {
		return fmt.Errorf("list harvested glossary rows: %w", err)
	}
	type row struct{ id, scope, src, zh, lang string }
	var reversed []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.scope, &r.src, &r.zh, &r.lang); err != nil {
			rows.Close()
			return fmt.Errorf("scan glossary row: %w", err)
		}
		// An empty rendering would become an empty term once turned around
		// (CR M5) — not "reversed", just broken; left for the review panel.
		if containsHan(r.src) && !containsHan(r.zh) && strings.TrimSpace(r.zh) != "" {
			reversed = append(reversed, r)
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	// The duplicate check reads the live table inside this transaction, so two
	// backwards rows converging on one English term see each other: the first
	// is turned around, the second finds it and is deleted (CR H2).
	swapped, deleted := 0, 0
	for _, r := range reversed {
		var exists int
		if err := tx.QueryRow(`SELECT COUNT(*) FROM show_glossary
			WHERE scope = ? AND term_src = ? COLLATE NOCASE AND language = ? AND id != ?`,
			r.scope, r.zh, r.lang, r.id).Scan(&exists); err != nil {
			return fmt.Errorf("check glossary duplicate: %w", err)
		}
		if exists > 0 {
			if _, err := tx.Exec(`DELETE FROM show_glossary WHERE id = ?`, r.id); err != nil {
				return fmt.Errorf("delete reversed glossary row: %w", err)
			}
			deleted++
			continue
		}
		if _, err := tx.Exec(`UPDATE show_glossary SET term_src = ?, term_zh = ? WHERE id = ?`, r.zh, r.src, r.id); err != nil {
			return fmt.Errorf("turn glossary row around: %w", err)
		}
		swapped++
	}
	if swapped+deleted > 0 {
		slog.Info("migration 045: reversed harvested glossary terms repaired", "turned_around", swapped, "deleted_duplicates", deleted)
	}
	return nil
}

func (m *fixReversedHarvestedGlossaryTerms) Down(tx *sql.Tx) error {
	// A data repair; the backwards rows are not worth restoring.
	return nil
}

func containsHan(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}
