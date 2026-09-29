package repository

import (
	"context"
	"sort"
	"testing"

	"github.com/vido/api/internal/models"
)

// disc-2026-09-unmatched-filter-vs-parse-status AC #1/#2/#4 [@contract-v1]:
// 「未匹配」means the enrichment gave up AND no source ever supplied data.
// One fixture set drives both the 未匹配 filter and the 未匹配 (N) count, so
// the list a user opens always holds exactly N rows.
type unmatchedCase struct {
	id        string
	status    models.ParseStatus
	source    string // "" leaves metadata_source NULL; "<empty>" writes ''
	tmdbID    int64
	removed   bool
	unmatched bool
}

var unmatchedCases = []unmatchedCase{
	{id: "tmdb-success", status: models.ParseStatusSuccess, source: "tmdb", tmdbID: 100},
	{id: "douban-success", status: models.ParseStatusSuccess, source: "douban"},
	{id: "nfo-success", status: models.ParseStatusSuccess, source: "nfo"},
	{id: "manual-success", status: models.ParseStatusSuccess, source: "manual"},
	// ⚖️ ruling ② (A): a row still being worked on is 整理中, not 未匹配.
	{id: "pending-nosource", status: models.ParseStatusPending},
	// Re-parsed during a TMDb outage: failed, but the old data is still there.
	{id: "failed-keeps-old-data", status: models.ParseStatusFailed, source: "tmdb", tmdbID: 200},
	// Same, for a match that never had a TMDb id — the case the old
	// tmdb_id-only rule got wrong (CR MED-1).
	{id: "failed-keeps-douban-data", status: models.ParseStatusFailed, source: "douban"},
	{id: "failed-keeps-nfo-data", status: models.ParseStatusFailed, source: "nfo"},
	{id: "failed-nosource", status: models.ParseStatusFailed, unmatched: true},
	{id: "failed-emptysource", status: models.ParseStatusFailed, source: "<empty>", unmatched: true},
	{id: "failed-nosource-removed", status: models.ParseStatusFailed, removed: true},
}

func (c unmatchedCase) metadataSource() models.NullString {
	switch c.source {
	case "":
		return models.NullString{}
	case "<empty>":
		return models.NewNullString("")
	default:
		return models.NewNullString(c.source)
	}
}

func (c unmatchedCase) tmdb() models.NullInt64 {
	if c.tmdbID == 0 {
		return models.NullInt64{}
	}
	return models.NewNullInt64(c.tmdbID)
}

func wantUnmatchedIDs() []string {
	var ids []string
	for _, c := range unmatchedCases {
		if c.unmatched {
			ids = append(ids, c.id)
		}
	}
	sort.Strings(ids)
	return ids
}

func sortedIDs(ids []string) []string {
	sort.Strings(ids)
	return ids
}

func TestMovieUnmatched_OnlyFailedWithNoSource(t *testing.T) {
	db := setupTestDBWithFTS(t)
	defer db.Close()
	repo := NewMovieRepository(db)
	ctx := context.Background()

	for _, c := range unmatchedCases {
		m := &models.Movie{
			ID: c.id, Title: "Probe " + c.id, ReleaseDate: "2020-01-01", Genres: []string{},
			ParseStatus: c.status, MetadataSource: c.metadataSource(), TMDbID: c.tmdb(), IsRemoved: c.removed,
		}
		if err := repo.Create(ctx, m); err != nil {
			t.Fatalf("Create %s: %v", c.id, err)
		}
	}
	want := wantUnmatchedIDs()

	params := NewListParams()
	params.Filters["unmatched"] = true
	movies, p, err := repo.List(ctx, params)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	var got []string
	for _, m := range movies {
		got = append(got, m.ID)
	}
	if got = sortedIDs(got); !equalStrings(got, want) || p.TotalResults != len(want) {
		t.Errorf("List unmatched = %v (total %d), want %v", got, p.TotalResults, want)
	}

	// The search path (/library/search) applies the same condition.
	params = NewListParams()
	params.Filters["unmatched"] = true
	found, fp, err := repo.FullTextSearch(ctx, "Probe", params)
	if err != nil {
		t.Fatalf("FullTextSearch: %v", err)
	}
	got = nil
	for _, m := range found {
		got = append(got, m.ID)
	}
	if got = sortedIDs(got); !equalStrings(got, want) || fp.TotalResults != len(want) {
		t.Errorf("FullTextSearch unmatched = %v (total %d), want %v", got, fp.TotalResults, want)
	}

	stats, err := repo.GetStats(ctx)
	if err != nil {
		t.Fatalf("GetStats: %v", err)
	}
	if stats.UnmatchedCount != len(want) {
		t.Errorf("unmatched_count = %d, want %d (must equal the filter's row count)", stats.UnmatchedCount, len(want))
	}
}

func TestSeriesUnmatched_OnlyFailedWithNoSource(t *testing.T) {
	db := setupSeriesTestDBWithFTS(t)
	defer db.Close()
	repo := NewSeriesRepository(db)
	ctx := context.Background()

	for _, c := range unmatchedCases {
		s := &models.Series{
			ID: c.id, Title: "Probe " + c.id, FirstAirDate: "2020-01-01", Genres: []string{},
			ParseStatus: c.status, MetadataSource: c.metadataSource(), TMDbID: c.tmdb(), IsRemoved: c.removed,
		}
		if err := repo.Create(ctx, s); err != nil {
			t.Fatalf("Create %s: %v", c.id, err)
		}
	}
	want := wantUnmatchedIDs()

	params := NewListParams()
	params.Filters["unmatched"] = true
	series, p, err := repo.List(ctx, params)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	var got []string
	for _, s := range series {
		got = append(got, s.ID)
	}
	if got = sortedIDs(got); !equalStrings(got, want) || p.TotalResults != len(want) {
		t.Errorf("List unmatched = %v (total %d), want %v", got, p.TotalResults, want)
	}

	params = NewListParams()
	params.Filters["unmatched"] = true
	found, fp, err := repo.FullTextSearch(ctx, "Probe", params)
	if err != nil {
		t.Fatalf("FullTextSearch: %v", err)
	}
	got = nil
	for _, s := range found {
		got = append(got, s.ID)
	}
	if got = sortedIDs(got); !equalStrings(got, want) || fp.TotalResults != len(want) {
		t.Errorf("FullTextSearch unmatched = %v (total %d), want %v", got, fp.TotalResults, want)
	}

	stats, err := repo.GetStats(ctx)
	if err != nil {
		t.Fatalf("GetStats: %v", err)
	}
	if stats.UnmatchedCount != len(want) {
		t.Errorf("unmatched_count = %d, want %d (must equal the filter's row count)", stats.UnmatchedCount, len(want))
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
