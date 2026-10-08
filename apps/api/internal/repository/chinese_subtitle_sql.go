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

// worstChineseSubtitleSQLFunc is the SQL aggregate over models.WorstChineseSubtitle:
// the episode verdict that most needs handling
// (disc-2026-10-subtitle-filter-series-phase-2). NULL over zero rows. Same
// caveat as vido_chinese_subtitle — queries only, never an index/view/trigger.
const worstChineseSubtitleSQLFunc = "vido_worst_chinese_subtitle"

func init() {
	if err := sqlite.RegisterFunction(worstChineseSubtitleSQLFunc, &sqlite.FunctionImpl{
		NArgs:         1,
		Deterministic: true,
		MakeAggregate: func(sqlite.FunctionContext) (sqlite.AggregateFunction, error) {
			return &worstChineseSubtitleAgg{}, nil
		},
	}); err != nil {
		panic(err)
	}
}

type worstChineseSubtitleAgg struct {
	verdicts []models.ChineseSubtitle
}

func (a *worstChineseSubtitleAgg) Step(_ *sqlite.FunctionContext, args []driver.Value) error {
	if len(args) == 1 {
		a.verdicts = append(a.verdicts, models.ChineseSubtitle(sqlText(args[0])))
	}
	return nil
}

func (a *worstChineseSubtitleAgg) WindowInverse(*sqlite.FunctionContext, []driver.Value) error {
	return fmt.Errorf("%s cannot be used as a window function", worstChineseSubtitleSQLFunc)
}

func (a *worstChineseSubtitleAgg) WindowValue(*sqlite.FunctionContext) (driver.Value, error) {
	if worst, ok := models.WorstChineseSubtitle(a.verdicts); ok {
		return string(worst), nil
	}
	return nil, nil
}

func (a *worstChineseSubtitleAgg) Final(*sqlite.FunctionContext) {}

// seriesChineseSubtitleSQL is the ONE expression for a series' "has Chinese
// subtitles" verdict (disc-2026-10-subtitle-filter-series-phase-2 AC #1–#3):
// the worst verdict among its episodes that have a file; a series with none of
// those falls back to its own row. `table` is how the outer query names the
// series table ("series", or the FTS join's "s") — it must be explicit, a bare
// `id` inside the subquery would resolve to episodes.id.
func seriesChineseSubtitleSQL(table string) string {
	return fmt.Sprintf(`COALESCE(
		(SELECT %[1]s(%[2]s(e.subtitle_status, e.subtitle_language, e.subtitle_tracks))
		   FROM episodes e
		  WHERE e.series_id = %[3]s.id AND e.file_path IS NOT NULL AND e.file_path != ''),
		%[2]s(%[3]s.subtitle_status, %[3]s.subtitle_language, %[3]s.subtitle_tracks))`,
		worstChineseSubtitleSQLFunc, chineseSubtitleSQLFunc, table)
}

// seriesChineseSubtitleFilterCondition is chineseSubtitleFilterCondition for
// series: the same groups, matched against seriesChineseSubtitleSQL so the
// filter and the field cannot disagree.
func seriesChineseSubtitleFilterCondition(groups []string, table string) (string, []interface{}) {
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
	return fmt.Sprintf("%s IN (%s)", seriesChineseSubtitleSQL(table), placeholders), args
}
