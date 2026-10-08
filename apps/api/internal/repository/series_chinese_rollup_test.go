package repository

// disc-2026-10-subtitle-filter-series-phase-2: a series' "has Chinese
// subtitles" verdict is rolled up from its episodes (the one that most needs
// handling), on every read path, and the library filter matches the SAME
// value. Real migration chain (Rule 28).

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/models"
)

const (
	trEngOnly = `[{"language":"eng","format":"subrip","external":false,"stream_index":8}]`
	trZhTW    = `[{"language":"eng","format":"subrip","external":false,"stream_index":8},{"language":"zh-Hant","format":"srt","external":true,"stream_index":0,"file_name":"x.zh-TW.srt"}]`
	trChiHant = `[{"language":"chi","format":"subrip","external":false,"title":"繁體中文","stream_index":3}]`
)

// seedEpisode writes one episode row; tracks "" = NULL (never read).
func seedEpisode(t *testing.T, db *sql.DB, seriesID string, season, ep int, file, tracks string) {
	t.Helper()
	var tr interface{}
	if tracks != "" {
		tr = tracks
	}
	var fp interface{}
	if file != "" {
		fp = file
	}
	_, err := db.Exec(`INSERT INTO episodes (id, series_id, season_number, episode_number, file_path, subtitle_status, subtitle_tracks, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, 'not_searched', ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		fmt.Sprintf("%s-s%02de%02d", seriesID, season, ep), seriesID, season, ep, fp, tr)
	require.NoError(t, err)
}

// seedSee writes 《末日光明》as the NAS holds it: S1E1 English only, S1E2–E8 an
// official zh-TW file beside them, S2 Traditional inside the file.
func seedSee(t *testing.T, db *sql.DB, seriesID string) {
	t.Helper()
	seedEpisode(t, db, seriesID, 1, 1, "/tv/See/S1/See.S01E01.mkv", trEngOnly)
	for ep := 2; ep <= 8; ep++ {
		seedEpisode(t, db, seriesID, 1, ep, fmt.Sprintf("/tv/See/S1/See.S01E%02d.mkv", ep), trZhTW)
	}
	for ep := 1; ep <= 8; ep++ {
		seedEpisode(t, db, seriesID, 2, ep, fmt.Sprintf("/tv/See/S2/See.S02E%02d.mkv", ep), trChiHant)
	}
}

func TestSeriesChineseRollup_SeeShape(t *testing.T) {
	db := newMigratedLibraryDB(t)
	repo := NewSeriesRepository(db)
	ctx := context.Background()

	seedSeries(t, db, repo, "see", "See 末日光明", subtitleShape{status: "not_searched"})
	seedSee(t, db, "see")

	got, err := repo.FindByID(ctx, "see")
	require.NoError(t, err)
	assert.Equal(t, models.ChineseSubtitleNone, got.ChineseSubtitle, "S1E1 has no Chinese → the series is missing Chinese")

	_, err = db.Exec(`UPDATE episodes SET subtitle_tracks = ? WHERE id = 'see-s01e01'`, trZhTW)
	require.NoError(t, err)
	got, err = repo.FindByID(ctx, "see")
	require.NoError(t, err)
	assert.Equal(t, models.ChineseSubtitleZhHant, got.ChineseSubtitle, "every episode has Traditional")

	_, err = db.Exec(`UPDATE episodes SET subtitle_tracks = NULL WHERE id = 'see-s02e05'`)
	require.NoError(t, err)
	got, err = repo.FindByID(ctx, "see")
	require.NoError(t, err)
	assert.Equal(t, models.ChineseSubtitleUnknown, got.ChineseSubtitle, "one unread episode → cannot claim 'has'")
}

func TestSeriesChineseRollup_EpisodesWithoutFilesDoNotCount(t *testing.T) {
	db := newMigratedLibraryDB(t)
	repo := NewSeriesRepository(db)
	ctx := context.Background()

	// The series row itself says found zh-Hant; its only episode rows are
	// TMDb placeholders with no file — they must not drag it to "unknown".
	seedSeries(t, db, repo, "row-only", "Row only", subtitleShape{status: "found", language: strp("zh-Hant")})
	seedEpisode(t, db, "row-only", 1, 1, "", "")
	got, err := repo.FindByID(ctx, "row-only")
	require.NoError(t, err)
	assert.Equal(t, models.ChineseSubtitleZhHant, got.ChineseSubtitle, "no episode with a file → the series row's own verdict")

	seedSeries(t, db, repo, "bare", "Bare", subtitleShape{status: "not_searched"})
	got, err = repo.FindByID(ctx, "bare")
	require.NoError(t, err)
	assert.Equal(t, models.ChineseSubtitleUnknown, got.ChineseSubtitle)
}

// AC #2: the filter and the field agree, on List and FullTextSearch, and the
// three groups partition the series.
func TestSeriesChineseRollup_FilterMatchesField(t *testing.T) {
	db := newMigratedLibraryDB(t)
	repo := NewSeriesRepository(db)
	ctx := context.Background()

	seedSeries(t, db, repo, "r-missing", "Rollup See", subtitleShape{status: "not_searched"})
	seedSee(t, db, "r-missing")
	seedSeries(t, db, repo, "r-has", "Rollup has", subtitleShape{status: "not_searched"})
	seedEpisode(t, db, "r-has", 1, 1, "/tv/h/1.mkv", trChiHant)
	seedEpisode(t, db, "r-has", 1, 2, "/tv/h/2.mkv", trZhTW)
	seedSeries(t, db, repo, "r-unknown", "Rollup unknown", subtitleShape{status: "not_searched"})
	seedEpisode(t, db, "r-unknown", 1, 1, "/tv/u/1.mkv", trZhTW)
	seedEpisode(t, db, "r-unknown", 1, 2, "/tv/u/2.mkv", "")
	seedSeries(t, db, repo, "r-row", "Rollup row only", subtitleShape{status: "not_found"})

	want := map[string]string{
		"r-missing": models.ChineseSubtitleFilterMissing,
		"r-has":     models.ChineseSubtitleFilterHas,
		"r-unknown": models.ChineseSubtitleFilterUnknown,
		"r-row":     models.ChineseSubtitleFilterMissing, // not_found on the row, no episodes
	}

	all, _, err := repo.List(ctx, filterParams())
	require.NoError(t, err)
	field := map[string]models.ChineseSubtitle{}
	for _, s := range all {
		field[s.ID] = s.ChineseSubtitle
	}
	for id, g := range want {
		assert.Equal(t, g, groupOf(field[id]), "field group of %s", id)
	}

	seriesID := func(s models.Series) string { return s.ID }
	for _, g := range models.AllChineseSubtitleFilters() {
		var ids []string
		for id, wg := range want {
			if wg == g {
				ids = append(ids, id)
			}
		}
		wantIDs := csSortedIDs(ids, func(s string) string { return s })

		listed, _, err := repo.List(ctx, filterParams(g))
		require.NoError(t, err)
		assert.Equal(t, wantIDs, csSortedIDs(listed, seriesID), "List %s", g)

		searched, _, err := repo.FullTextSearch(ctx, "Rollup", filterParams(g))
		require.NoError(t, err)
		assert.Equal(t, wantIDs, csSortedIDs(searched, seriesID), "FullTextSearch %s", g)
	}
}

// AC #1: every read path carries the rolled-up verdict, not the row's.
func TestSeriesChineseRollup_EveryReadPath(t *testing.T) {
	db := newMigratedLibraryDB(t)
	repo := NewSeriesRepository(db)
	ctx := context.Background()

	seedSeries(t, db, repo, "paths", "Paths 末日光明", subtitleShape{status: "not_searched"})
	_, err := db.Exec(`UPDATE series SET tmdb_id = 85421, imdb_id = 'tt7949218', file_path = '/tv/See', parse_status = 'success' WHERE id = 'paths'`)
	require.NoError(t, err)
	seedSee(t, db, "paths")

	check := func(name string, s *models.Series, err error) {
		t.Helper()
		require.NoError(t, err, name)
		require.NotNil(t, s, name)
		assert.Equal(t, models.ChineseSubtitleNone, s.ChineseSubtitle, name)
	}
	s, err := repo.FindByID(ctx, "paths")
	check("FindByID", s, err)
	s, err = repo.FindByTMDbID(ctx, 85421)
	check("FindByTMDbID", s, err)
	s, err = repo.FindActiveByTMDbID(ctx, 85421)
	check("FindActiveByTMDbID", s, err)
	s, err = repo.FindByIMDbID(ctx, "tt7949218")
	check("FindByIMDbID", s, err)
	s, err = repo.FindByFilePath(ctx, "/tv/See")
	check("FindByFilePath", s, err)

	list, _, err := repo.List(ctx, filterParams())
	require.NoError(t, err)
	require.Len(t, list, 1)
	check("List", &list[0], nil)

	found, _, err := repo.FullTextSearch(ctx, "Paths", filterParams())
	require.NoError(t, err)
	require.Len(t, found, 1)
	check("FullTextSearch", &found[0], nil)

	byParse, err := repo.FindByParseStatus(ctx, models.ParseStatusSuccess)
	require.NoError(t, err)
	require.Len(t, byParse, 1)
	check("FindByParseStatus", &byParse[0], nil)
}

// AC #5: garbage in episodes never turns the series list into an error.
func TestSeriesChineseRollup_BadEpisodeDataIsUnknownNotAnError(t *testing.T) {
	db := newMigratedLibraryDB(t)
	repo := NewSeriesRepository(db)
	ctx := context.Background()

	seedSeries(t, db, repo, "bad", "Bad data", subtitleShape{status: "not_searched"})
	seedEpisode(t, db, "bad", 1, 1, "/tv/b/1.mkv", `{not json`)
	got, err := repo.FindByID(ctx, "bad")
	require.NoError(t, err)
	assert.Equal(t, models.ChineseSubtitleUnknown, got.ChineseSubtitle)
}

// CR L5: an empty-string file_path is "no file", like NULL; and once ANY
// episode has a file, the series row's own record no longer decides (SM
// ruling 1 — a series-level "found" names no episode).
func TestSeriesChineseRollup_EmptyPathIsNoFileAndEpisodesOutrankTheRow(t *testing.T) {
	db := newMigratedLibraryDB(t)
	repo := NewSeriesRepository(db)
	ctx := context.Background()

	seedSeries(t, db, repo, "mix", "Mix", subtitleShape{status: "found", language: strp("zh-Hant")})
	_, err := db.Exec(`INSERT INTO episodes (id, series_id, season_number, episode_number, file_path, subtitle_status, created_at, updated_at)
		VALUES ('mix-empty', 'mix', 1, 1, '', 'not_searched', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)
	require.NoError(t, err)
	got, err := repo.FindByID(ctx, "mix")
	require.NoError(t, err)
	assert.Equal(t, models.ChineseSubtitleZhHant, got.ChineseSubtitle, "'' is no file — the row still decides")

	seedEpisode(t, db, "mix", 1, 2, "/tv/m/2.mkv", trEngOnly)
	got, err = repo.FindByID(ctx, "mix")
	require.NoError(t, err)
	assert.Equal(t, models.ChineseSubtitleNone, got.ChineseSubtitle, "an episode with a file and no Chinese outranks the row")
}
