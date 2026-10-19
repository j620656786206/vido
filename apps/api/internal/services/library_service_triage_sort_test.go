package services

// Story disc-2026-10-library-list-triage-a AC #4: for sort_by=subtitle_status
// the "全部" view must use the same rank and the same tie-breakers as the
// single-type lists, so pages joined end to end equal one big fetch — no row
// repeated, none skipped — and match the per-type order. Since
// disc-2026-10-library-all-view-past-100 the order is one SQL query
// (LibraryRepository.ListKeys); this stays as its regression test.

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
)

func triageSeed(t *testing.T, db *sql.DB, table, id string, status string, language, tracks interface{}, createdAt time.Time) {
	t.Helper()
	_, err := db.Exec(`UPDATE `+table+` SET subtitle_status = ?, subtitle_language = ?, subtitle_tracks = ?, created_at = ? WHERE id = ?`,
		status, language, tracks, createdAt, id)
	require.NoError(t, err)
}

func libraryItemIDs(items []LibraryItem) []string {
	ids := make([]string, len(items))
	for i, it := range items {
		if it.Movie != nil {
			ids[i] = it.Movie.ID
		} else if it.Series != nil {
			ids[i] = it.Series.ID
		}
	}
	return ids
}

func TestLibraryService_ListAll_SortBySubtitleStatus(t *testing.T) {
	db := setupTestDB(t)
	movieRepo := repository.NewMovieRepository(db)
	seriesRepo := repository.NewSeriesRepository(db)
	service := NewLibraryService(movieRepo, seriesRepo, repository.NewEpisodeRepository(db), repository.NewLibraryRepository(db))
	ctx := context.Background()

	const (
		eng     = `[{"language":"eng","format":"subrip","external":false,"stream_index":2}]`
		zhHant  = `[{"language":"zh-Hant","format":"srt","external":true,"stream_index":0}]`
		chiHans = `[{"language":"chi","format":"subrip","external":false,"title":"简体","stream_index":2}]`
	)
	base := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)

	for _, id := range []string{"b-bodycam", "c-old-missing", "m-untrans", "m-hant"} {
		require.NoError(t, movieRepo.Create(ctx, &models.Movie{ID: id, Title: id, ReleaseDate: "2020-01-01"}))
	}
	for _, id := range []string{"a-see", "s-unknown", "s-hans"} {
		require.NoError(t, seriesRepo.Create(ctx, &models.Series{ID: id, Title: id, FirstAirDate: "2020-01-01"}))
	}

	// rank 0 缺中文 — a movie and a series with the SAME created_at: id decides
	// across types (a-see < b-bodycam).
	triageSeed(t, db, "movies", "b-bodycam", "not_found", nil, eng, base)
	// a-see's own row says 繁中, but its one episode is English only → the
	// series rolls up to 缺中文 (AC #3).
	triageSeed(t, db, "series", "a-see", "not_searched", nil, zhHant, base)
	_, err := db.Exec(`INSERT INTO episodes (id, series_id, season_number, episode_number, file_path, subtitle_status, subtitle_tracks, created_at, updated_at)
		VALUES ('a-see-s01e01', 'a-see', 1, 1, '/tv/See/S01E01.mkv', 'not_searched', ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`, eng)
	require.NoError(t, err)
	triageSeed(t, db, "movies", "c-old-missing", "not_found", nil, nil, base.Add(-3*time.Hour))
	// rank 1 未翻譯, rank 2 不知道, rank 3 簡中, rank 5 繁中 — newest last, so a
	// created_at sort would order them the other way round.
	triageSeed(t, db, "movies", "m-untrans", "untranslated", "en", nil, base.Add(1*time.Hour))
	triageSeed(t, db, "series", "s-unknown", "not_searched", nil, nil, base.Add(2*time.Hour))
	triageSeed(t, db, "series", "s-hans", "not_searched", nil, chiHans, base.Add(3*time.Hour))
	triageSeed(t, db, "movies", "m-hant", "not_searched", nil, zhHant, base.Add(4*time.Hour))

	list := func(order string, page, size int) *LibraryListResult {
		t.Helper()
		p := repository.NewListParams()
		p.SortBy = "subtitle_status"
		p.SortOrder = order
		p.Page = page
		p.PageSize = size
		r, err := service.ListLibrary(ctx, p, "all")
		require.NoError(t, err)
		return r
	}

	wantAsc := []string{"a-see", "b-bodycam", "c-old-missing", "m-untrans", "s-unknown", "s-hans", "m-hant"}
	wantDesc := []string{"m-hant", "s-hans", "s-unknown", "m-untrans", "a-see", "b-bodycam", "c-old-missing"}

	for order, want := range map[string][]string{"asc": wantAsc, "desc": wantDesc} {
		t.Run(order, func(t *testing.T) {
			one := list(order, 1, 20)
			assert.Equal(t, want, libraryItemIDs(one.Items))
			assert.Equal(t, 7, one.Pagination.TotalResults)

			// page_size=2 → 4 pages; joined end to end they must equal one fetch.
			var joined []string
			for page := 1; page <= 4; page++ {
				r := list(order, page, 2)
				assert.Equal(t, 4, r.Pagination.TotalPages)
				joined = append(joined, libraryItemIDs(r.Items)...)
			}
			assert.Equal(t, want, joined, "pages joined must equal one fetch — no repeats, no gaps")
		})
	}

	// The merged order agrees with each single-type order.
	t.Run("all agrees with movie-only and tv-only", func(t *testing.T) {
		all := libraryItemIDs(list("asc", 1, 20).Items)
		for _, mt := range []string{"movie", "tv"} {
			p := repository.NewListParams()
			p.SortBy = "subtitle_status"
			p.SortOrder = "asc"
			r, err := service.ListLibrary(ctx, p, mt)
			require.NoError(t, err)
			only := libraryItemIDs(r.Items)
			var sub []string
			for _, id := range all {
				for _, o := range only {
					if id == o {
						sub = append(sub, id)
					}
				}
			}
			assert.Equal(t, only, sub, mt)
		}
	})
}
