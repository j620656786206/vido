package repository

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/database/migrations"
	"github.com/vido/api/internal/models"
)

// Story disc-2026-10-subtitle-filter-disagrees-with-badges AC #5 (Rule 28 real
// shape): a database built by the REAL migration chain, file-backed with
// foreign keys on like the app DSN, seeded with every AC #4 shape. For each
// filter group, List and FullTextSearch must return EXACTLY the rows whose
// chinese_subtitle field (filled by scanMovie / scanSeries) falls in that group
// — the badge and the filter are the same rule. The three groups partition the
// library.

func newMigratedLibraryDB(t *testing.T) *sql.DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "library.db")
	db, err := sql.Open("sqlite", path+"?_pragma=foreign_keys(1)")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	runner, err := migrations.NewRunner(db)
	require.NoError(t, err)
	require.NoError(t, runner.RegisterAll(migrations.GetAll()))
	require.NoError(t, runner.Up(context.Background()))
	return db
}

// subtitleShape is one row's raw subtitle columns; nil = NULL.
type subtitleShape struct {
	name     string
	status   string
	language *string
	tracks   *string
	want     models.ChineseSubtitle
}

func strp(s string) *string { return &s }

const (
	trChiEng    = `[{"language":"chi","format":"subrip","external":false,"stream_index":2},{"language":"eng","format":"subrip","external":false,"stream_index":3}]`
	trEng       = `[{"language":"eng","format":"subrip","external":false,"stream_index":2}]`
	trSidecarTW = `[{"language":"zh-TW","format":"srt","external":true,"stream_index":0}]`
)

// ac4Shapes mirrors the AC #4 table in models/chinese_subtitle_test.go.
var ac4Shapes = []subtitleShape{
	{"chi+eng", "not_searched", nil, strp(trChiEng), models.ChineseSubtitleZh},
	{"eng only", "not_searched", nil, strp(trEng), models.ChineseSubtitleNone},
	{"empty array", "not_searched", nil, strp(`[]`), models.ChineseSubtitleNone},
	{"NULL not_searched", "not_searched", nil, nil, models.ChineseSubtitleUnknown},
	{"NULL not_found", "not_found", nil, nil, models.ChineseSubtitleNone},
	{"NULL untranslated en", "untranslated", strp("en"), nil, models.ChineseSubtitleNone},
	{"found zh-Hant", "found", strp("zh-Hant"), nil, models.ChineseSubtitleZhHant},
	{"found zh", "found", strp("zh"), nil, models.ChineseSubtitleZh},
	{"NFO chi,eng", "not_searched", nil, strp("chi,eng"), models.ChineseSubtitleZh},
	{"sidecar zh-TW", "not_searched", nil, strp(trSidecarTW), models.ChineseSubtitleZhHant},
	{"sidecar chi.forced", "not_searched", nil, strp(`[{"language":"chi.forced","format":"srt","external":true,"stream_index":0}]`), models.ChineseSubtitleZh},
	{"und only", "not_searched", nil, strp(`[{"language":"und","format":"srt","external":true,"stream_index":0}]`), models.ChineseSubtitleUnknown},
	{"eng+und", "not_searched", nil, strp(`[{"language":"eng","format":"subrip","external":false,"stream_index":2},{"language":"und","format":"srt","external":true,"stream_index":0}]`), models.ChineseSubtitleUnknown},
	{"chi 繁體中文", "not_searched", nil, strp(`[{"language":"chi","format":"subrip","external":false,"title":"繁體中文","stream_index":2}]`), models.ChineseSubtitleZhHant},
	{"chi 简体", "not_searched", nil, strp(`[{"language":"chi","format":"subrip","external":false,"title":"简体","stream_index":2}]`), models.ChineseSubtitleZhHans},
	{"chi 粵語 only", "not_searched", nil, strp(`[{"language":"chi","format":"subrip","external":false,"title":"粵語","stream_index":2}]`), models.ChineseSubtitleNone},
	{"yue only", "not_searched", nil, strp(`[{"language":"yue","format":"subrip","external":false,"stream_index":2}]`), models.ChineseSubtitleNone},
	{"yue+chi", "not_searched", nil, strp(`[{"language":"yue","format":"subrip","external":false,"stream_index":2},{"language":"chi","format":"subrip","external":false,"stream_index":3}]`), models.ChineseSubtitleZh},
	{"pgs chi no_text_source", "no_text_source", nil, strp(`[{"language":"chi","format":"hdmv_pgs_subtitle","external":false,"stream_index":4}]`), models.ChineseSubtitleZh},
	{"found en NULL", "found", strp("en"), nil, models.ChineseSubtitleUnknown},
	{"garbage", "not_searched", nil, strp(`{oops this is not json`), models.ChineseSubtitleUnknown},
}

// seedMovie inserts a movie through the real Create path, then forces the raw
// subtitle columns (NULL where the shape says so) — the way the scanner and
// the subtitle pipeline leave them in production.
func seedMovie(t *testing.T, db *sql.DB, repo *MovieRepository, id, title string, sh subtitleShape) {
	t.Helper()
	ctx := context.Background()
	require.NoError(t, repo.Create(ctx, &models.Movie{ID: id, Title: title, ReleaseDate: "2020-01-01"}))
	_, err := db.ExecContext(ctx,
		`UPDATE movies SET subtitle_status = ?, subtitle_language = ?, subtitle_tracks = ? WHERE id = ?`,
		sh.status, nullable(sh.language), nullable(sh.tracks), id)
	require.NoError(t, err)
}

func seedSeries(t *testing.T, db *sql.DB, repo *SeriesRepository, id, title string, sh subtitleShape) {
	t.Helper()
	ctx := context.Background()
	require.NoError(t, repo.Create(ctx, &models.Series{ID: id, Title: title, FirstAirDate: "2020-01-01"}))
	_, err := db.ExecContext(ctx,
		`UPDATE series SET subtitle_status = ?, subtitle_language = ?, subtitle_tracks = ? WHERE id = ?`,
		sh.status, nullable(sh.language), nullable(sh.tracks), id)
	require.NoError(t, err)
}

func nullable(s *string) interface{} {
	if s == nil {
		return nil
	}
	return *s
}

func groupOf(v models.ChineseSubtitle) string {
	for _, g := range models.AllChineseSubtitleFilters() {
		for _, gv := range models.ChineseSubtitleFilterVerdicts(g) {
			if gv == v {
				return g
			}
		}
	}
	return ""
}

func filterParams(groups ...string) ListParams {
	p := NewListParams()
	p.PageSize = MaxPageSize
	if len(groups) > 0 {
		p.Filters["chinese_subtitle"] = groups
	}
	return p
}

func csSortedIDs[T any](items []T, id func(T) string) []string {
	out := make([]string, 0, len(items))
	for _, it := range items {
		out = append(out, id(it))
	}
	sort.Strings(out)
	return out
}

func TestChineseSubtitleFilter_RealShape_FilterMatchesField(t *testing.T) {
	db := newMigratedLibraryDB(t)
	movieRepo := NewMovieRepository(db)
	seriesRepo := NewSeriesRepository(db)
	ctx := context.Background()

	for i, sh := range ac4Shapes {
		// Every title shares the word "Probe" so FullTextSearch sees them all.
		seedMovie(t, db, movieRepo, fmt.Sprintf("m-%02d", i), "Probe movie "+sh.name, sh)
	}
	seedSeries(t, db, seriesRepo, "s-unknown", "Probe series unknown", subtitleShape{status: "not_searched"})
	seedSeries(t, db, seriesRepo, "s-found", "Probe series found", subtitleShape{status: "found", language: strp("zh-Hant")})

	// 1. The field (scanMovie / scanSeries) is the AC #4 verdict for every shape.
	all, _, err := movieRepo.List(ctx, filterParams())
	require.NoError(t, err)
	require.Len(t, all, len(ac4Shapes))
	fieldOf := map[string]models.ChineseSubtitle{}
	for _, m := range all {
		fieldOf[m.ID] = m.ChineseSubtitle
	}
	for i, sh := range ac4Shapes {
		assert.Equal(t, sh.want, fieldOf[fmt.Sprintf("m-%02d", i)], "field for %q", sh.name)
	}
	allSeries, _, err := seriesRepo.List(ctx, filterParams())
	require.NoError(t, err)
	seriesField := map[string]models.ChineseSubtitle{}
	for _, s := range allSeries {
		seriesField[s.ID] = s.ChineseSubtitle
	}
	assert.Equal(t, models.ChineseSubtitleUnknown, seriesField["s-unknown"])
	assert.Equal(t, models.ChineseSubtitleZhHant, seriesField["s-found"])

	// The detail path (FindByID) carries the same field.
	one, err := movieRepo.FindByID(ctx, "m-00")
	require.NoError(t, err)
	assert.Equal(t, models.ChineseSubtitleZh, one.ChineseSubtitle)

	// 2. Each group, on both read paths, returns exactly the rows whose field is
	// in that group; the groups partition the library.
	movieUnion := map[string]int{}
	seriesUnion := map[string]int{}
	for _, g := range models.AllChineseSubtitleFilters() {
		wantMovies, wantSeries := []string{}, []string{}
		for id, v := range fieldOf {
			if groupOf(v) == g {
				wantMovies = append(wantMovies, id)
			}
		}
		for id, v := range seriesField {
			if groupOf(v) == g {
				wantSeries = append(wantSeries, id)
			}
		}
		sort.Strings(wantMovies)
		sort.Strings(wantSeries)
		movieID := func(m models.Movie) string { return m.ID }
		seriesID := func(s models.Series) string { return s.ID }

		listed, p, err := movieRepo.List(ctx, filterParams(g))
		require.NoError(t, err)
		assert.Equal(t, wantMovies, csSortedIDs(listed, movieID), "movie List %s", g)
		assert.Equal(t, len(wantMovies), p.TotalResults, "movie List %s total", g)
		for _, id := range csSortedIDs(listed, movieID) {
			movieUnion[id]++
		}

		searched, sp, err := movieRepo.FullTextSearch(ctx, "Probe", filterParams(g))
		require.NoError(t, err)
		assert.Equal(t, wantMovies, csSortedIDs(searched, movieID), "movie FullTextSearch %s", g)
		assert.Equal(t, len(wantMovies), sp.TotalResults, "movie FullTextSearch %s total", g)

		sList, _, err := seriesRepo.List(ctx, filterParams(g))
		require.NoError(t, err)
		assert.Equal(t, wantSeries, csSortedIDs(sList, seriesID), "series List %s", g)
		for _, id := range csSortedIDs(sList, seriesID) {
			seriesUnion[id]++
		}

		sSearch, _, err := seriesRepo.FullTextSearch(ctx, "Probe", filterParams(g))
		require.NoError(t, err)
		assert.Equal(t, wantSeries, csSortedIDs(sSearch, seriesID), "series FullTextSearch %s", g)
	}
	assert.Len(t, movieUnion, len(ac4Shapes), "the three groups cover every movie")
	assert.Len(t, seriesUnion, 2, "the three groups cover every series")
	for id, n := range movieUnion {
		assert.Equal(t, 1, n, "movie %s must sit in exactly one group", id)
	}
	for id, n := range seriesUnion {
		assert.Equal(t, 1, n, "series %s must sit in exactly one group", id)
	}

	// 3. Two groups OR together; chinese_subtitle ANDs with subtitle_status.
	hasOrMissing, _, err := movieRepo.List(ctx, filterParams("has", "missing"))
	require.NoError(t, err)
	assert.Len(t, hasOrMissing, len(ac4Shapes)-countGroup(fieldOf, "unknown"))

	p := filterParams("missing")
	p.Filters["subtitle_status"] = []string{"not_found"}
	andRows, _, err := movieRepo.List(ctx, p)
	require.NoError(t, err)
	require.Len(t, andRows, 1)
	assert.Equal(t, models.ChineseSubtitleNone, andRows[0].ChineseSubtitle)
	assert.Equal(t, models.SubtitleStatusNotFound, andRows[0].SubtitleStatus)

	// A non-[]string value is ignored, not a crash (the year_min trap).
	bad := NewListParams()
	bad.PageSize = MaxPageSize
	bad.Filters["chinese_subtitle"] = "missing"
	rows, _, err := movieRepo.List(ctx, bad)
	require.NoError(t, err)
	assert.Len(t, rows, len(ac4Shapes))
}

func countGroup(field map[string]models.ChineseSubtitle, g string) int {
	n := 0
	for _, v := range field {
		if groupOf(v) == g {
			n++
		}
	}
	return n
}

// AC #9 (backend): the NAS distribution from the story — 37 movies with
// embedded chi+eng, 3 English-only, 12 NULL+not_searched, 3 found+zh-Hant,
// plus 2 series NULL+not_searched → has 40 / missing 3 / unknown 14.
func TestChineseSubtitleFilter_NASShape(t *testing.T) {
	db := newMigratedLibraryDB(t)
	movieRepo := NewMovieRepository(db)
	seriesRepo := NewSeriesRepository(db)
	ctx := context.Background()

	n := 0
	add := func(count int, sh subtitleShape) {
		for i := 0; i < count; i++ {
			seedMovie(t, db, movieRepo, fmt.Sprintf("nas-%02d", n), fmt.Sprintf("NAS movie %d", n), sh)
			n++
		}
	}
	add(37, subtitleShape{status: "not_searched", tracks: strp(trChiEng)})
	add(3, subtitleShape{status: "not_searched", tracks: strp(trEng)})
	add(12, subtitleShape{status: "not_searched"})
	add(3, subtitleShape{status: "found", language: strp("zh-Hant")})
	seedSeries(t, db, seriesRepo, "nas-s1", "NAS series 1", subtitleShape{status: "not_searched"})
	seedSeries(t, db, seriesRepo, "nas-s2", "NAS series 2", subtitleShape{status: "not_searched"})

	count := func(g string) int {
		_, mp, err := movieRepo.List(ctx, filterParams(g))
		require.NoError(t, err)
		_, sp, err := seriesRepo.List(ctx, filterParams(g))
		require.NoError(t, err)
		return mp.TotalResults + sp.TotalResults
	}
	assert.Equal(t, 40, count("has"))
	assert.Equal(t, 3, count("missing"))
	assert.Equal(t, 14, count("unknown"))
}

// The SQL function must never fail a query, whatever the column holds.
func TestChineseSubtitleSQLFunction_BadInputNeverErrors(t *testing.T) {
	db := newMigratedLibraryDB(t)
	for _, q := range []string{
		`SELECT vido_chinese_subtitle(NULL, NULL, NULL)`,
		`SELECT vido_chinese_subtitle('found', 'zh-Hant', X'00FF')`,
		`SELECT vido_chinese_subtitle(1, 2.5, '[')`,
	} {
		var v string
		require.NoError(t, db.QueryRow(q).Scan(&v), q)
		assert.True(t, models.ChineseSubtitle(v).IsValid(), "%s → %q", q, v)
	}
	var v string
	require.NoError(t, db.QueryRow(`SELECT vido_chinese_subtitle('found', 'zh-Hant', NULL)`).Scan(&v))
	assert.Equal(t, "zh_hant", v)
}
