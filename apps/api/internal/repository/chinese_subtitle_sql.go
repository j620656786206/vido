package repository

import (
	"database/sql/driver"
	"fmt"
	"strings"

	"github.com/vido/api/internal/models"
	"modernc.org/sqlite"
)

// chineseSubtitleSQLFunc is the SQL name of models.ChineseSubtitleVerdict.
//
// Story disc-2026-10-subtitle-filter-disagrees-with-badges AC #3: the library's
// chinese_subtitle filter must use the SAME rule the badge uses. Registering the
// Go function into SQLite (rather than re-writing it with json_each) keeps one
// implementation and always reads live columns — no stored column to go stale,
// no migration, no backfill.
//
// ⚠️ Use it ONLY inside queries. Never in an index, view, trigger or generated
// column: external tools (the sqlite3 CLI, backup_service's read-only restore
// connection) do not have the function and could not open such an object.
const chineseSubtitleSQLFunc = "vido_chinese_subtitle"

// init registers the function with the modernc driver. modernc makes a custom
// function available to every connection opened AFTER registration
// (sqlite.go RegisterDeterministicScalarFunction), and package init runs before
// main or any test opens a database — both the app (whose sqlite-utc driver
// wraps the same registered driver, internal/database/utc_driver.go) and this
// package's own tests (sql.Open("sqlite", …)) see it. Connections opened with
// the mattn sqlite3 driver do NOT; none of those touch the movie/series lists.
func init() {
	sqlite.MustRegisterDeterministicScalarFunction(chineseSubtitleSQLFunc, 3, chineseSubtitleSQL)
}

// chineseSubtitleSQL adapts the SQLite call vido_chinese_subtitle(status,
// language, tracks). It never returns an error: a broken row must not turn the
// whole library list into a 500 (AC #4) — the verdict degrades to "unknown".
func chineseSubtitleSQL(_ *sqlite.FunctionContext, args []driver.Value) (driver.Value, error) {
	if len(args) != 3 {
		return string(models.ChineseSubtitleUnknown), nil
	}
	return string(models.ChineseSubtitleVerdict(sqlText(args[0]), sqlText(args[1]), sqlText(args[2]))), nil
}

// sqlText turns a SQLite argument into the verdict's string form; NULL is "".
func sqlText(v driver.Value) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case []byte:
		return string(x)
	default:
		return fmt.Sprint(x)
	}
}

// chineseSubtitleFilterCondition builds the WHERE term for the library's
// chinese_subtitle filter (AC #2 [@contract-v1]). `col` qualifies a column for
// the FTS join alias. Groups are validated by the handler; an unknown group
// maps to no verdicts and is skipped. Returns "" when nothing applies.
func chineseSubtitleFilterCondition(groups []string, col func(string) string) (string, []interface{}) {
	var args []interface{}
	for _, g := range groups {
		for _, v := range models.ChineseSubtitleFilterVerdicts(g) {
			args = append(args, string(v))
		}
	}
	if len(args) == 0 {
		return "", nil
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?, ", len(args)), ", ")
	return fmt.Sprintf("%s(%s, %s, %s) IN (%s)", chineseSubtitleSQLFunc,
		col("subtitle_status"), col("subtitle_language"), col("subtitle_tracks"), placeholders), args
}
