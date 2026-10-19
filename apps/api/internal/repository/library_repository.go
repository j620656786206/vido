package repository

// Story disc-2026-10-library-all-view-past-100: the library list has ONE
// ordering rule, written once in SQL. The movie list, the series list and the
// "全部" (movies + series) list all build their ORDER BY here, and the "全部"
// list is ordered by a single UNION ALL query instead of being merged in Go —
// a second, Go-side copy of the rule is what let 「評分」 sort by one column in
// SQL and another in Go, and what dropped titles past the 100th.

import (
	"context"
	"database/sql"
	"fmt"
)

// Library item kinds, as returned in LibraryKey.Type (and LibraryItem.Type).
const (
	LibraryKindMovie  = "movie"
	LibraryKindSeries = "series"
)

// librarySortKey resolves a sort_by value into the SQL expression it orders
// by on one table (`kind` is LibraryKindMovie or LibraryKindSeries; the
// expressions use bare column names on the un-aliased `movies` / `series`
// table). Unknown values fall back to created_at.
//
//   - rating and vote_average both order by vote_average — the score the list
//     shows. The legacy `rating` column is no longer an ordering key: the
//     scan / match paths never write it, so on real libraries it is NULL for
//     every title (AC #3).
//   - release_date and first_air_date are the same key: a movie's
//     release_date, a series' first_air_date (so the "全部" list never
//     compares a date with something else).
//   - subtitle_status orders by the subtitle badge rank
//     (models.SubtitleTriageRank, disc-2026-10-library-list-triage-a); a
//     series' verdict is its episode rollup.
func librarySortKey(kind, sortBy string) string {
	switch sortBy {
	case "id", "title", "updated_at":
		return sortBy
	case "release_date", "first_air_date":
		if kind == LibraryKindSeries {
			return "first_air_date"
		}
		return "release_date"
	case "rating", "vote_average":
		return "vote_average"
	case "subtitle_status":
		verdict := fmt.Sprintf("%s(subtitle_status, subtitle_language, subtitle_tracks)", chineseSubtitleSQLFunc)
		if kind == LibraryKindSeries {
			verdict = seriesChineseSubtitleSQL("series")
		}
		return subtitleRankKey(verdict)
	}
	return "created_at"
}

// subtitleRankKey is the subtitle badge rank of a row whose verdict is the
// SQL expression `verdict`. SeriesRepository.List passes its already-selected
// rollup column, so the episode rollup is not evaluated a second time for
// the ORDER BY.
func subtitleRankKey(verdict string) string {
	return fmt.Sprintf("%s(%s, subtitle_status)", subtitleTriageRankSQLFunc, verdict)
}

// libraryOrderBy is the ORDER BY body for `key` (a librarySortKey result, or
// the "全部" query's sort_key column) in `order` ("asc"/"desc"), with the
// tie-breakers every library sort shares: created_at newest first, then id
// ascending. The tie-breakers never flip with `order`
// (disc-2026-10-library-list-triage-a AC #1 [@contract-v1]). Without them, rows
// with equal keys come back in an unspecified order that can differ between
// two LIMIT/OFFSET queries, so paging repeats some rows and skips others.
func libraryOrderBy(key, order string) string {
	if order != "asc" {
		order = "desc"
	}
	if key == "created_at" {
		return "created_at " + order + ", id ASC"
	}
	return key + " " + order + ", created_at DESC, id ASC"
}

// LibraryKey identifies one library item: its kind and its id.
type LibraryKey struct {
	Type string // LibraryKindMovie or LibraryKindSeries
	ID   string
}

// LibraryRepository reads the movies + series library as one list.
type LibraryRepository struct {
	db *sql.DB
}

// NewLibraryRepository creates a new LibraryRepository.
func NewLibraryRepository(db *sql.DB) *LibraryRepository {
	return &LibraryRepository{db: db}
}

// ListKeys returns one page of the combined movies + series list as
// (kind, id) keys, in display order, and the combined pagination. Both halves
// apply exactly the WHERE that MovieRepository.List / SeriesRepository.List
// apply (movieListWhere / seriesListWhere), and the order is libraryOrderBy
// plus kind as the last tie-breaker — so the combined list contains the same
// rows as the two single-type lists, in an order both of them agree with.
// The NUMBER of queries per page is fixed (two counts + one ordered query),
// whatever the page number; each ordered query still sorts every matching
// row of both tables (no index serves these keys) — fine at library sizes.
// The three reads are separate statements, as they were before: a scan
// running in between can make total_items differ by a row from what pages
// return.
func (r *LibraryRepository) ListKeys(ctx context.Context, params ListParams) ([]LibraryKey, *PaginationResult, error) {
	params.Validate()

	movieWhere, movieArgs := movieListWhere(params)
	seriesWhere, seriesArgs := seriesListWhere(params)

	var movieCount, seriesCount int
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM movies "+movieWhere, movieArgs...).Scan(&movieCount); err != nil {
		return nil, nil, fmt.Errorf("failed to count movies: %w", err)
	}
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM series "+seriesWhere, seriesArgs...).Scan(&seriesCount); err != nil {
		return nil, nil, fmt.Errorf("failed to count series: %w", err)
	}

	// A compound SELECT's ORDER BY may only name result columns, hence the
	// wrapping sub-query. The series branch must stay un-aliased: its subtitle
	// rollup refers to the table as `series`. For sort_by=created_at,
	// sort_key IS created_at, so "sort_key <dir>, created_at DESC, id ASC" is
	// exactly the single-type "created_at <dir>, id ASC".
	query := fmt.Sprintf(`
		SELECT kind, id FROM (
			SELECT '%s' AS kind, id, %s AS sort_key, created_at FROM movies %s
			UNION ALL
			SELECT '%s' AS kind, id, %s AS sort_key, created_at FROM series %s
		)
		ORDER BY %s, kind ASC
		LIMIT ? OFFSET ?
	`,
		LibraryKindMovie, librarySortKey(LibraryKindMovie, params.SortBy), movieWhere,
		LibraryKindSeries, librarySortKey(LibraryKindSeries, params.SortBy), seriesWhere,
		libraryOrderBy("sort_key", params.SortOrder))

	// Bound args follow placeholder order: movie WHERE, series WHERE, then
	// LIMIT/OFFSET. That holds because the projected columns (kind, id,
	// sort_key, created_at) carry no placeholders — a sort key that ever needs
	// a bound value must be threaded in here, in position.
	args := make([]interface{}, 0, len(movieArgs)+len(seriesArgs)+2)
	args = append(args, movieArgs...)
	args = append(args, seriesArgs...)
	args = append(args, params.Limit(), params.Offset())

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list library: %w", err)
	}
	defer rows.Close()

	keys := []LibraryKey{}
	for rows.Next() {
		var k LibraryKey
		if err := rows.Scan(&k.Type, &k.ID); err != nil {
			return nil, nil, fmt.Errorf("failed to scan library key: %w", err)
		}
		keys = append(keys, k)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("error iterating library keys: %w", err)
	}

	return keys, NewPaginationResult(params, movieCount+seriesCount), nil
}
