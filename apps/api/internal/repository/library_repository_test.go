package repository

// Story disc-2026-10-library-all-view-past-100 AC #1–#2 (Rule 28 real shape):
// 130 movies + 120 series on the REAL migration chain, shaped like the
// production library — legacy `rating` NULL everywhere, few distinct
// vote_average / dates / created_at values (lots of ties), duplicate titles,
// every subtitle-badge rank, three soft-deleted rows. For every sort key ×
// direction × type, paging to the end must return every live row exactly
// once, in the documented order, whatever the page size.

import (
	"context"
	"database/sql"
	"fmt"
	"hash/fnv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/database/dbtime"
	"github.com/vido/api/internal/models"
)

// libRow is the oracle's view of one seeded row.
type libRow struct {
	kind      string
	id        string
	title     string
	date      string          // release_date / first_air_date
	vote      sql.NullFloat64 // vote_average
	createdAt string          // dbtime-formatted, fixed width: text order = time order
	rank      int             // models.SubtitleTriageRank of what the list shows
	removed   bool
}

func (r libRow) key() LibraryKey { return LibraryKey{Type: r.kind, ID: r.id} }

const (
	libMovies = 130
	libSeries = 120
)

var (
	libVotes = []float64{6.0, 7.3, 7.7, 8.5, 9.5} // index 5 → NULL
	libDates = []string{"2019-05-01", "2020-01-01", "2021-12-31", "2023-07-15", "1999-09-09", ""}
	// Six subtitle shapes, one per badge rank (0 缺中文 … 5 繁中).
	libSubtitleShapes = []struct {
		status           string
		language, tracks interface{}
	}{
		{"not_found", nil, trEng},   // 缺中文
		{"untranslated", "en", nil}, // 未翻譯
		{"not_searched", nil, nil},  // 不知道
		{"not_searched", nil, `[{"language":"chi","format":"subrip","external":false,"title":"简体","stream_index":2}]`}, // 簡中
		{"not_searched", nil, trChiOnly},   // 中文
		{"not_searched", nil, trZhHantSRT}, // 繁中
	}
)

// libID is a stable, uuid-like id whose order is unrelated to the seed index
// and interleaves movies with series, so the id tie-breaker really matters.
func libID(kind string, i int) string {
	h := fnv.New32a()
	fmt.Fprintf(h, "%s-%d", kind, i)
	return fmt.Sprintf("%08x-%s-%03d", h.Sum32(), kind[:1], i)
}

// seedLibrary writes the AC #1 library and returns the oracle rows.
func seedLibrary(t *testing.T, db *sql.DB) []libRow {
	t.Helper()
	ctx := context.Background()
	movieRepo := NewMovieRepository(db)
	seriesRepo := NewSeriesRepository(db)
	base := time.Date(2026, 9, 1, 8, 0, 0, 0, time.UTC)

	var rows []libRow
	add := func(kind string, i int) {
		r := libRow{kind: kind, id: libID(kind, i)}
		// Duplicate titles within and across kinds; some CJK.
		if i%7 == 0 {
			r.title = fmt.Sprintf("鏢人 %d", i%3)
		} else {
			r.title = fmt.Sprintf("Title %02d", i%37)
		}
		r.date = libDates[(i/2)%len(libDates)]
		if v := (i * 5) % 6; v < len(libVotes) {
			r.vote = sql.NullFloat64{Float64: libVotes[v], Valid: true}
		}
		r.createdAt = dbtime.Format(base.Add(time.Duration((i*7)%10) * time.Hour))
		shape := libSubtitleShapes[(i/3)%len(libSubtitleShapes)]

		table := "movies"
		if kind == LibraryKindMovie {
			require.NoError(t, movieRepo.Create(ctx, &models.Movie{ID: r.id, Title: r.title, ReleaseDate: r.date, Genres: []string{}}))
		} else {
			table = "series"
			require.NoError(t, seriesRepo.Create(ctx, &models.Series{ID: r.id, Title: r.title, FirstAirDate: r.date, Genres: []string{}}))
		}
		_, err := db.Exec(`UPDATE `+table+` SET rating = NULL, vote_average = ?, created_at = ?,
			subtitle_status = ?, subtitle_language = ?, subtitle_tracks = ? WHERE id = ?`,
			r.vote, r.createdAt, shape.status, shape.language, shape.tracks, r.id)
		require.NoError(t, err)
		rows = append(rows, r)
	}
	for i := 0; i < libMovies; i++ {
		add(LibraryKindMovie, i)
	}
	for i := 0; i < libSeries; i++ {
		add(LibraryKindSeries, i)
	}

	// Every 10th series has one English-only episode: its badge rolls up to
	// 缺中文 whatever the series row says (the rank must follow the rollup).
	for i := 0; i < libSeries; i += 10 {
		id := libID(LibraryKindSeries, i)
		_, err := db.Exec(`INSERT INTO episodes (id, series_id, season_number, episode_number, file_path, subtitle_status, subtitle_tracks, created_at, updated_at)
			VALUES (?, ?, 1, 1, ?, 'not_searched', ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
			id+"-e1", id, "/tv/"+id+"/S01E01.mkv", trEng)
		require.NoError(t, err)
	}

	// Soft-delete 2 movies and 1 series.
	removed := map[string]bool{
		libID(LibraryKindMovie, 3): true, libID(LibraryKindMovie, 64): true, libID(LibraryKindSeries, 17): true,
	}
	for k := range removed {
		table := "movies"
		if strings.Contains(k, "-s-") {
			table = "series"
		}
		_, err := db.Exec(`UPDATE `+table+` SET is_removed = 1 WHERE id = ?`, k)
		require.NoError(t, err)
	}

	// The badge rank comes from what the read path shows (models.SubtitleTriageRank
	// over the verdict the list returns — for a series, the episode rollup).
	for i := range rows {
		rows[i].removed = removed[rows[i].id]
		if rows[i].kind == LibraryKindMovie {
			m, err := movieRepo.FindByID(ctx, rows[i].id)
			require.NoError(t, err)
			rows[i].rank = models.SubtitleTriageRank(m.ChineseSubtitle, m.SubtitleStatus)
		} else {
			s, err := seriesRepo.FindByID(ctx, rows[i].id)
			require.NoError(t, err)
			rows[i].rank = models.SubtitleTriageRank(s.ChineseSubtitle, s.SubtitleStatus)
		}
	}
	return rows
}

// cmpNullFloat orders like SQLite: NULL before every number.
func cmpNullFloat(a, b sql.NullFloat64) int {
	switch {
	case !a.Valid && !b.Valid:
		return 0
	case !a.Valid:
		return -1
	case !b.Valid:
		return 1
	case a.Float64 < b.Float64:
		return -1
	case a.Float64 > b.Float64:
		return 1
	}
	return 0
}

func cmpInt(a, b int) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}

// libLess is the documented order, written independently of the SQL: the key
// in the requested direction, then created_at newest first, then id, then
// kind. For sort_by=created_at the key IS created_at, so the second step is
// moot.
func libLess(sortBy, order string, a, b libRow) bool {
	var c int
	switch sortBy {
	case "title":
		c = strings.Compare(a.title, b.title)
	case "release_date", "first_air_date":
		c = strings.Compare(a.date, b.date)
	case "rating", "vote_average":
		c = cmpNullFloat(a.vote, b.vote)
	case "subtitle_status":
		c = cmpInt(a.rank, b.rank)
	default: // created_at
		c = strings.Compare(a.createdAt, b.createdAt)
	}
	if order == "desc" {
		c = -c
	}
	if c != 0 {
		return c < 0
	}
	if c := strings.Compare(a.createdAt, b.createdAt); c != 0 {
		return c > 0 // newest first, whatever the direction
	}
	if c := strings.Compare(a.id, b.id); c != 0 {
		return c < 0
	}
	return a.kind < b.kind
}

var (
	// The five keys the UI sends, plus the two aliases the API also accepts.
	libSortKeys   = []string{"created_at", "title", "release_date", "rating", "subtitle_status", "vote_average", "first_air_date"}
	libSortOrders = []string{"asc", "desc"}
	libTypes      = []string{"all", "movie", "tv"}
)

// pageThrough pages one list to the end and returns the keys page by page.
func pageThrough(t *testing.T, db *sql.DB, mediaType, sortBy, order string, pageSize int) (pages [][]LibraryKey, total, totalPages int) {
	t.Helper()
	ctx := context.Background()
	for page := 1; ; page++ {
		p := NewListParams()
		p.SortBy, p.SortOrder, p.Page, p.PageSize = sortBy, order, page, pageSize
		var keys []LibraryKey
		var pg *PaginationResult
		switch mediaType {
		case "movie":
			ms, mpg, err := NewMovieRepository(db).List(ctx, p)
			require.NoError(t, err)
			for _, m := range ms {
				keys = append(keys, LibraryKey{Type: LibraryKindMovie, ID: m.ID})
			}
			pg = mpg
		case "tv":
			ss, spg, err := NewSeriesRepository(db).List(ctx, p)
			require.NoError(t, err)
			for _, s := range ss {
				keys = append(keys, LibraryKey{Type: LibraryKindSeries, ID: s.ID})
			}
			pg = spg
		default:
			ks, kpg, err := NewLibraryRepository(db).ListKeys(ctx, p)
			require.NoError(t, err)
			keys, pg = ks, kpg
		}
		if page == 1 {
			total, totalPages = pg.TotalResults, pg.TotalPages
		}
		require.Equal(t, total, pg.TotalResults, "total_items changed between pages (page %d)", page)
		if page > totalPages {
			require.Empty(t, keys, "page %d is past total_pages=%d but returned rows", page, totalPages)
			return pages, total, totalPages
		}
		require.NotEmpty(t, keys, "page %d of %d came back empty", page, totalPages)
		pages = append(pages, keys)
	}
}

func TestLibraryList_PagingIsCompleteAndOrdered(t *testing.T) {
	db := newMigratedLibraryDB(t)
	rows := seedLibrary(t, db)

	byKey := map[LibraryKey]libRow{}
	live := map[string]map[LibraryKey]bool{"all": {}, "movie": {}, "tv": {}}
	for _, r := range rows {
		byKey[r.key()] = r
		if r.removed {
			continue
		}
		live["all"][r.key()] = true
		if r.kind == LibraryKindMovie {
			live["movie"][r.key()] = true
		} else {
			live["tv"][r.key()] = true
		}
	}
	require.Len(t, live["movie"], 128)
	require.Len(t, live["tv"], 119)
	require.Len(t, live["all"], 247)

	for _, sortBy := range libSortKeys {
		for _, order := range libSortOrders {
			for _, mediaType := range libTypes {
				name := fmt.Sprintf("sort_by=%s/%s/type=%s", sortBy, order, mediaType)
				t.Run(name, func(t *testing.T) {
					var reference []LibraryKey
					for _, size := range []int{36, 7, 100} {
						pages, total, totalPages := pageThrough(t, db, mediaType, sortBy, order, size)
						want := len(live[mediaType])
						require.Equal(t, want, total, "total_items (page_size %d)", size)
						require.Equal(t, (want+size-1)/size, totalPages, "total_pages (page_size %d)", size)

						var seq []LibraryKey
						seen := map[LibraryKey]int{}
						var prev *libRow
						for pi, page := range pages {
							for _, k := range page {
								if firstPage, dup := seen[k]; dup {
									t.Fatalf("%s page_size=%d: %s %s on page %d repeats (first on page %d)", name, size, k.Type, k.ID, pi+1, firstPage)
								}
								seen[k] = pi + 1
								require.True(t, live[mediaType][k], "%s page_size=%d page %d: %s %s is not a live %s row", name, size, pi+1, k.Type, k.ID, mediaType)
								cur := byKey[k]
								if prev != nil && !libLess(sortBy, order, *prev, cur) {
									t.Fatalf("%s page_size=%d page %d: out of order — %+v came after %+v", name, size, pi+1, cur, *prev)
								}
								prev = &cur
								seq = append(seq, k)
							}
						}
						require.Len(t, seq, want, "%s page_size=%d: rows across all pages", name, size)

						// AC #2: the joined sequence does not depend on the page size.
						if reference == nil {
							reference = seq
						} else {
							require.Equal(t, reference, seq, "%s: page_size %d gives a different sequence than 36", name, size)
						}
					}
				})
			}
		}
	}
}

// The "全部" list contains each single-type list as a subsequence, in the
// same order: the two halves are filtered and ordered by the same rule.
func TestLibraryListKeys_AgreesWithSingleTypeLists(t *testing.T) {
	db := newMigratedLibraryDB(t)
	seedLibrary(t, db)

	for _, sortBy := range libSortKeys {
		for _, order := range libSortOrders {
			all, _, _ := pageThrough(t, db, "all", sortBy, order, 100)
			movies, _, _ := pageThrough(t, db, "movie", sortBy, order, 100)
			series, _, _ := pageThrough(t, db, "tv", sortBy, order, 100)
			var gotMovies, gotSeries, wantMovies, wantSeries []LibraryKey
			for _, p := range all {
				for _, k := range p {
					if k.Type == LibraryKindMovie {
						gotMovies = append(gotMovies, k)
					} else {
						gotSeries = append(gotSeries, k)
					}
				}
			}
			for _, p := range movies {
				wantMovies = append(wantMovies, p...)
			}
			for _, p := range series {
				wantSeries = append(wantSeries, p...)
			}
			require.Equal(t, wantMovies, gotMovies, "%s %s: movies inside 全部", sortBy, order)
			require.Equal(t, wantSeries, gotSeries, "%s %s: series inside 全部", sortBy, order)
		}
	}
}

// Filters and search apply to both halves of the "全部" query exactly as they
// do to the single-type lists, and the bound args line up.
func TestLibraryListKeys_FiltersMatchSingleTypeLists(t *testing.T) {
	db := newMigratedLibraryDB(t)
	seedLibrary(t, db)
	ctx := context.Background()

	cases := map[string]map[string]interface{}{
		"search":           {"search": "鏢人"},
		"year range":       {"year_min": "2020", "year_max": "2021"},
		"chinese_subtitle": {"chinese_subtitle": []string{"missing"}},
		"subtitle_status":  {"subtitle_status": []string{"untranslated", "not_found"}},
		"search+year+zh":   {"search": "Title", "year_min": "2019", "chinese_subtitle": []string{"has"}},
	}
	for name, filters := range cases {
		t.Run(name, func(t *testing.T) {
			p := NewListParams()
			p.SortBy, p.SortOrder, p.PageSize = "rating", "desc", 100
			p.Filters = filters
			ms, mpg, err := NewMovieRepository(db).List(ctx, p)
			require.NoError(t, err)
			ss, spg, err := NewSeriesRepository(db).List(ctx, p)
			require.NoError(t, err)
			keys, kpg, err := NewLibraryRepository(db).ListKeys(ctx, p)
			require.NoError(t, err)

			require.Positive(t, mpg.TotalResults+spg.TotalResults, "case must match something")
			require.Equal(t, mpg.TotalResults+spg.TotalResults, kpg.TotalResults)
			want := map[LibraryKey]bool{}
			for _, m := range ms {
				want[LibraryKey{Type: LibraryKindMovie, ID: m.ID}] = true
			}
			for _, s := range ss {
				want[LibraryKey{Type: LibraryKindSeries, ID: s.ID}] = true
			}
			got := map[LibraryKey]bool{}
			for _, k := range keys {
				got[k] = true
			}
			require.Equal(t, want, got)
		})
	}
}

func TestFindByIDs(t *testing.T) {
	db := newMigratedLibraryDB(t)
	rows := seedLibrary(t, db)
	ctx := context.Background()

	t.Run("empty list returns empty without error", func(t *testing.T) {
		ms, err := NewMovieRepository(db).FindByIDs(ctx, nil)
		require.NoError(t, err)
		require.Empty(t, ms)
		ss, err := NewSeriesRepository(db).FindByIDs(ctx, []string{})
		require.NoError(t, err)
		require.Empty(t, ss)
	})

	t.Run("returns the asked rows and skips missing ids", func(t *testing.T) {
		var movieIDs, seriesIDs []string
		for _, r := range rows[5:10] { // movie 3 is soft-deleted
			movieIDs = append(movieIDs, r.id)
		}
		for _, r := range rows[libMovies : libMovies+5] {
			seriesIDs = append(seriesIDs, r.id)
		}
		ms, err := NewMovieRepository(db).FindByIDs(ctx, append(movieIDs, "no-such-movie"))
		require.NoError(t, err)
		var got []string
		for _, m := range ms {
			got = append(got, m.ID)
			require.NotEmpty(t, m.Title)
		}
		require.ElementsMatch(t, movieIDs, got)

		ss, err := NewSeriesRepository(db).FindByIDs(ctx, append(seriesIDs, "no-such-series"))
		require.NoError(t, err)
		got = nil
		for _, s := range ss {
			got = append(got, s.ID)
			require.NotEmpty(t, s.ChineseSubtitle, "series rollup column must be read")
		}
		require.ElementsMatch(t, seriesIDs, got)
	})

	t.Run("soft-deleted rows are skipped", func(t *testing.T) {
		gone := libID(LibraryKindMovie, 3) // is_removed=1 in seedLibrary
		ms, err := NewMovieRepository(db).FindByIDs(ctx, []string{gone, rows[0].id})
		require.NoError(t, err)
		require.Len(t, ms, 1)
		require.Equal(t, rows[0].id, ms[0].ID)
		ss, err := NewSeriesRepository(db).FindByIDs(ctx, []string{libID(LibraryKindSeries, 17)})
		require.NoError(t, err)
		require.Empty(t, ss)
	})
}

func TestLibrarySortKey(t *testing.T) {
	// rating is the score the list shows (vote_average), never the legacy column.
	require.Equal(t, "vote_average", librarySortKey(LibraryKindMovie, "rating"))
	require.Equal(t, "vote_average", librarySortKey(LibraryKindSeries, "rating"))
	require.Equal(t, "first_air_date", librarySortKey(LibraryKindSeries, "release_date"))
	require.Equal(t, "release_date", librarySortKey(LibraryKindMovie, "release_date"))
	// first_air_date is the same key as release_date for both kinds, so "全部"
	// never compares a movie's created_at against a series' air date.
	require.Equal(t, "release_date", librarySortKey(LibraryKindMovie, "first_air_date"))
	require.Equal(t, "first_air_date", librarySortKey(LibraryKindSeries, "first_air_date"))
	require.Equal(t, "vido_subtitle_triage_rank(chinese_subtitle_rollup, subtitle_status)", seriesListSortKey("subtitle_status"))
	require.Equal(t, librarySortKey(LibraryKindSeries, "rating"), seriesListSortKey("rating"))
	require.Equal(t, "created_at", librarySortKey(LibraryKindMovie, "rating; DROP TABLE movies"))
	require.Equal(t, "created_at", librarySortKey(LibraryKindSeries, ""))

	require.Equal(t, "created_at asc, id ASC", libraryOrderBy("created_at", "asc"))
	require.Equal(t, "vote_average desc, created_at DESC, id ASC", libraryOrderBy("vote_average", "desc"))
	require.Equal(t, "title desc, created_at DESC, id ASC", libraryOrderBy("title", "DROP"))
}
