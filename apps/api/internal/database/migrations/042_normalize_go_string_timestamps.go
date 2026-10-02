package migrations

import (
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/vido/api/internal/database/dbtime"
)

func init() {
	Register(&normalizeGoStringTimestamps{
		migrationBase: NewMigrationBase(42, "normalize_go_string_timestamps"),
	})
}

// normalizeGoStringTimestamps (bugfix-h) rewrites every timestamp the modernc
// driver stored with Go's time.Time.String() —
//
//	2026-08-24 10:56:26.303854294 +0800 CST m=+26.660278123
//
// — into the text the app now writes (dbtime.Format):
//
//	2026-08-24 02:56:26.303854294Z
//
// SQLite's date functions returned NULL for 100% of the old values, and the
// mixed +0800/+0000 offsets made text comparisons wrong by hours.
//
// Scope: every column declared TIMESTAMP / DATETIME / DATE in an ordinary
// table. Only values with the String() shape that actually parse are touched;
// NULL, empty text, CURRENT_TIMESTAMP text, RFC3339 and anything unexpected are left
// alone (a stale format beats a lost timestamp). Re-running finds nothing to do.
type normalizeGoStringTimestamps struct {
	migrationBase
}

// goStringTimeGlob matches "YYYY-MM-DD HH:MM:SS[.frac] ±hhmm …" — the String()
// shape, and nothing SQLite or RFC3339 would produce.
const goStringTimeGlob = `[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9] [0-9][0-9]:[0-9][0-9]:[0-9][0-9]* [+-][0-9][0-9][0-9][0-9]*`

const normalizeBatch = 5000

func (m *normalizeGoStringTimestamps) Up(tx *sql.Tx) error {
	cols, err := timestampColumns(tx)
	if err != nil {
		return err
	}
	total := 0
	for _, c := range cols {
		n, err := normalizeColumn(tx, c.table, c.column)
		if err != nil {
			return fmt.Errorf("normalize %s.%s: %w", c.table, c.column, err)
		}
		if n > 0 {
			slog.Info("Normalized stored timestamps to UTC", "table", c.table, "column", c.column, "rows", n)
		}
		total += n
	}
	slog.Info("Timestamp normalization complete", "columns", len(cols), "rows", total)
	return nil
}

func (m *normalizeGoStringTimestamps) Down(tx *sql.Tx) error {
	// One-way data repair: the old text carried a monotonic-clock reading and
	// a zone name nothing reads. The new text is the same instant.
	return nil
}

type tableColumn struct{ table, column string }

func timestampColumns(tx *sql.Tx) ([]tableColumn, error) {
	rows, err := tx.Query(`SELECT name FROM sqlite_master
		WHERE type = 'table' AND name NOT LIKE 'sqlite_%'
		  AND sql NOT LIKE 'CREATE VIRTUAL TABLE%' AND sql NOT LIKE '%WITHOUT ROWID%'
		ORDER BY name`)
	if err != nil {
		return nil, err
	}
	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			rows.Close()
			return nil, err
		}
		tables = append(tables, name)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var out []tableColumn
	for _, t := range tables {
		crows, err := tx.Query(`SELECT name, type FROM pragma_table_info(?)`, t)
		if err != nil {
			return nil, err
		}
		for crows.Next() {
			var name, typ string
			if err := crows.Scan(&name, &typ); err != nil {
				crows.Close()
				return nil, err
			}
			switch strings.ToUpper(strings.TrimSpace(typ)) {
			case "TIMESTAMP", "DATETIME", "DATE":
				out = append(out, tableColumn{t, name})
			}
		}
		crows.Close()
		if err := crows.Err(); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func quoteIdent(s string) string { return `"` + strings.ReplaceAll(s, `"`, `""`) + `"` }

// normalizeColumn walks the matching rows in rowid batches (so an unparseable
// match can never loop) and rewrites each parseable one.
func normalizeColumn(tx *sql.Tx, table, column string) (int, error) {
	t, c := quoteIdent(table), quoteIdent(column)
	sel := fmt.Sprintf(`SELECT rowid, CAST(%s AS TEXT) FROM %s
		WHERE rowid > ? AND typeof(%s) = 'text' AND %s GLOB '%s'
		ORDER BY rowid LIMIT %d`, c, t, c, c, goStringTimeGlob, normalizeBatch)
	upd, err := tx.Prepare(fmt.Sprintf(`UPDATE %s SET %s = ? WHERE rowid = ?`, t, c))
	if err != nil {
		return 0, err
	}
	defer upd.Close()

	type pending struct {
		rowid int64
		text  string
	}
	var last int64 = -1 << 62
	changed := 0
	for {
		rows, err := tx.Query(sel, last)
		if err != nil {
			return changed, err
		}
		var batch []pending
		for rows.Next() {
			var p pending
			if err := rows.Scan(&p.rowid, &p.text); err != nil {
				rows.Close()
				return changed, err
			}
			batch = append(batch, p)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return changed, err
		}
		if len(batch) == 0 {
			return changed, nil
		}
		for _, p := range batch {
			last = p.rowid
			ts, ok := parseGoStringTime(p.text)
			if !ok {
				continue
			}
			if _, err := upd.Exec(dbtime.Format(ts), p.rowid); err != nil {
				return changed, err
			}
			changed++
		}
	}
}

// parseGoStringTime reads "2006-01-02 15:04:05.999999999 -0700 MST m=+1.2"
// (zone name and monotonic part optional) by keeping only date, clock and
// numeric offset — the zone abbreviation adds nothing and Go cannot always
// parse it ("+0800 +0800" for unnamed zones).
func parseGoStringTime(s string) (time.Time, bool) {
	f := strings.Fields(s)
	if len(f) < 3 {
		return time.Time{}, false
	}
	ts, err := time.Parse("2006-01-02 15:04:05.999999999 -0700", f[0]+" "+f[1]+" "+f[2])
	if err != nil {
		return time.Time{}, false
	}
	return ts, true
}
