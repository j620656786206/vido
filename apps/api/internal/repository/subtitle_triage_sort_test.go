package repository

// Story disc-2026-10-library-list-triage-a: GET /library?sort_by=subtitle_status
// ranks the subtitle badge the list shows — 缺中文 first, 繁中 last — with
// created_at DESC, id ASC as the tie-breakers. Real migration chain (Rule 28).

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/models"
)

const (
	trUndOnly   = `[{"language":"und","format":"subrip","external":false,"stream_index":2}]`
	trChiOnly   = `[{"language":"chi","format":"subrip","external":false,"stream_index":2}]`
	trZhHantSRT = `[{"language":"zh-Hant","format":"srt","external":true,"stream_index":0}]`
)

// triageBase is the created_at every seeded row is offset from.
var triageBase = time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)

func setCreatedAt(t *testing.T, db *sql.DB, table, id string, at time.Time) {
	t.Helper()
	_, err := db.Exec(`UPDATE `+table+` SET created_at = ? WHERE id = ?`, at, id)
	require.NoError(t, err)
}

// seedTriageMovies writes the design mock's five titles (as the NAS holds them)
// plus one untranslated title. Statuses are chosen so that alphabetical order
// of subtitle_status (not_found < not_searched < untranslated) is NOT the rank
// order — a plain `ORDER BY subtitle_status` would put 未翻譯 last and fail.
func seedTriageMovies(t *testing.T, db *sql.DB, repo *MovieRepository) {
	t.Helper()
	rows := []struct {
		id, title string
		sh        subtitleShape
		age       time.Duration // older = larger
	}{
		{"m-bodycam", "執法記錄", subtitleShape{status: "not_found", tracks: strp(trEng)}, 6 * time.Hour},
		{"m-untrans", "未翻譯的那部", subtitleShape{status: "untranslated", language: strp("en")}, 5 * time.Hour},
		{"m-biaoren", "鏢人", subtitleShape{status: "not_searched", tracks: strp(trUndOnly)}, 1 * time.Hour},
		{"m-chainsaw", "Chainsaw Man", subtitleShape{status: "not_searched"}, 3 * time.Hour},
		{"m-priest", "牧師的最後誘惑", subtitleShape{status: "not_searched", tracks: strp(trChiOnly)}, 2 * time.Hour},
		{"m-avatar", "阿凡達：火與燼", subtitleShape{status: "not_searched", tracks: strp(trZhHantSRT)}, 4 * time.Hour},
	}
	for _, r := range rows {
		seedMovie(t, db, repo, r.id, r.title, r.sh)
		setCreatedAt(t, db, "movies", r.id, triageBase.Add(-r.age))
	}
	_, err := db.Exec(`UPDATE movies SET parse_status = 'failed' WHERE id = 'm-chainsaw'`)
	require.NoError(t, err)
}

func triageParams(order string, page, size int) ListParams {
	p := NewListParams()
	p.SortBy = "subtitle_status"
	p.SortOrder = order
	p.Page = page
	p.PageSize = size
	return p
}

func TestMovieList_SortBySubtitleStatus(t *testing.T) {
	db := newMigratedLibraryDB(t)
	repo := NewMovieRepository(db)
	ctx := context.Background()
	seedTriageMovies(t, db, repo)

	t.Run("asc: 缺中文 → 未翻譯 → 不知道 (newer first) → 中文 → 繁中", func(t *testing.T) {
		got, pg, err := repo.List(ctx, triageParams("asc", 1, 20))
		require.NoError(t, err)
		assert.Equal(t, []string{"m-bodycam", "m-untrans", "m-biaoren", "m-chainsaw", "m-priest", "m-avatar"}, movieIDs(got))
		assert.Equal(t, 6, pg.TotalResults)

		// The order is the badge the list shows.
		ranks := make([]int, len(got))
		for i, m := range got {
			ranks[i] = models.SubtitleTriageRank(m.ChineseSubtitle, m.SubtitleStatus)
		}
		assert.Equal(t, []int{0, 1, 2, 2, 4, 5}, ranks)
	})

	t.Run("desc reverses the rank only, not the tie-breakers", func(t *testing.T) {
		got, _, err := repo.List(ctx, triageParams("desc", 1, 20))
		require.NoError(t, err)
		assert.Equal(t, []string{"m-avatar", "m-priest", "m-biaoren", "m-chainsaw", "m-untrans", "m-bodycam"}, movieIDs(got))
	})

	t.Run("page_size=2: three pages joined equal one fetch — no repeats, no gaps", func(t *testing.T) {
		for _, order := range []string{"asc", "desc"} {
			all, _, err := repo.List(ctx, triageParams(order, 1, 20))
			require.NoError(t, err)
			var joined []string
			for page := 1; page <= 3; page++ {
				got, pg, err := repo.List(ctx, triageParams(order, page, 2))
				require.NoError(t, err)
				assert.Equal(t, 3, pg.TotalPages)
				joined = append(joined, movieIDs(got)...)
			}
			assert.Equal(t, movieIDs(all), joined, order)
		}
	})

	t.Run("same rank and same created_at: id ascending", func(t *testing.T) {
		setCreatedAt(t, db, "movies", "m-chainsaw", triageBase.Add(-1*time.Hour))
		got, _, err := repo.List(ctx, triageParams("asc", 1, 20))
		require.NoError(t, err)
		assert.Equal(t, []string{"m-bodycam", "m-untrans", "m-biaoren", "m-chainsaw", "m-priest", "m-avatar"}, movieIDs(got),
			"m-biaoren < m-chainsaw by id")
		got, _, err = repo.List(ctx, triageParams("desc", 1, 20))
		require.NoError(t, err)
		assert.Equal(t, []string{"m-avatar", "m-priest", "m-biaoren", "m-chainsaw", "m-untrans", "m-bodycam"}, movieIDs(got),
			"id stays ascending under desc")
	})
}

func TestSeriesList_SortBySubtitleStatus_UsesEpisodeRollup(t *testing.T) {
	db := newMigratedLibraryDB(t)
	repo := NewSeriesRepository(db)
	ctx := context.Background()

	// 《See》: its own row says nothing; one episode English only, the rest
	// Traditional → the series is 缺中文 (rank 0), not 繁中.
	seedSeries(t, db, repo, "see", "See 末日光明", subtitleShape{status: "not_searched"})
	seedSee(t, db, "see")
	setCreatedAt(t, db, "series", "see", triageBase.Add(-5*time.Hour))

	// Every episode Traditional → 繁中 (rank 5), newest of all.
	seedSeries(t, db, repo, "hant", "全繁中影集", subtitleShape{status: "not_searched"})
	seedEpisode(t, db, "hant", 1, 1, "/tv/H/S01E01.mkv", trZhTW)
	setCreatedAt(t, db, "series", "hant", triageBase)

	// No episode files; the series row itself is skipped → 已略過 (rank 1).
	seedSeries(t, db, repo, "skip", "被略過的影集", subtitleShape{status: "skipped", tracks: strp(trUndOnly)})
	setCreatedAt(t, db, "series", "skip", triageBase.Add(-1*time.Hour))

	// Series status translating while episodes say nothing → 不知道 (rank 2).
	seedSeries(t, db, repo, "busy", "翻譯中的影集", subtitleShape{status: "translating"})
	seedEpisode(t, db, "busy", 1, 1, "/tv/B/S01E01.mkv", "")
	setCreatedAt(t, db, "series", "busy", triageBase.Add(-2*time.Hour))

	got, _, err := repo.List(ctx, triageParams("asc", 1, 20))
	require.NoError(t, err)
	ids := make([]string, len(got))
	for i, s := range got {
		ids[i] = s.ID
	}
	assert.Equal(t, []string{"see", "skip", "busy", "hant"}, ids)
	require.Equal(t, models.ChineseSubtitleNone, got[0].ChineseSubtitle, "See's field and its rank agree")

	got, _, err = repo.List(ctx, triageParams("desc", 1, 20))
	require.NoError(t, err)
	for i, s := range got {
		ids[i] = s.ID
	}
	assert.Equal(t, []string{"hant", "busy", "skip", "see"}, ids)
}

// AC #6: the existing sort keys and the unknown-key fallback are untouched.
func TestMovieList_OtherSortsUnchanged(t *testing.T) {
	db := newMigratedLibraryDB(t)
	repo := NewMovieRepository(db)
	ctx := context.Background()
	seedTriageMovies(t, db, repo)

	p := NewListParams()
	p.SortBy = "no_such_column"
	p.SortOrder = "desc"
	got, _, err := repo.List(ctx, p)
	require.NoError(t, err)
	assert.Equal(t, []string{"m-biaoren", "m-priest", "m-chainsaw", "m-avatar", "m-untrans", "m-bodycam"}, movieIDs(got),
		"unknown sort_by falls back to created_at")
}

// The SQL function must never fail a list: bad arity or garbage degrades to
// the unknown rank.
func TestSubtitleTriageRankSQL_NeverFails(t *testing.T) {
	db := newMigratedLibraryDB(t)
	var rank int
	require.NoError(t, db.QueryRow(`SELECT vido_subtitle_triage_rank('garbage', NULL)`).Scan(&rank))
	assert.Equal(t, models.SubtitleRankUnknown, rank)
	require.NoError(t, db.QueryRow(`SELECT vido_subtitle_triage_rank(NULL, 'untranslated')`).Scan(&rank))
	assert.Equal(t, models.SubtitleRankNoSourceOrStopped, rank)
	require.NoError(t, db.QueryRow(`SELECT vido_subtitle_triage_rank('none', 'not_found')`).Scan(&rank))
	assert.Equal(t, models.SubtitleRankMissing, rank)
}
