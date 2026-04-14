package services

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/vido/api/internal/tmdb"
)

// fixedClock returns a closure that always yields the given time — used to
// make FarFuture horizon math deterministic across test runs.
func fixedClock(t time.Time) func() time.Time {
	return func() time.Time { return t }
}

func TestContentFilterService_FilterFarFutureMovies(t *testing.T) {
	now := time.Date(2026, 4, 14, 0, 0, 0, 0, time.UTC)
	// horizon := now + 6 months = 2026-10-14
	svc := NewContentFilterServiceWithClock(fixedClock(now))

	movies := []tmdb.Movie{
		{ID: 1, Title: "past", ReleaseDate: "2020-01-01"},
		{ID: 2, Title: "released today", ReleaseDate: "2026-04-14"},
		{ID: 3, Title: "just inside horizon", ReleaseDate: "2026-10-13"},
		{ID: 4, Title: "exactly on horizon", ReleaseDate: "2026-10-14"},
		{ID: 5, Title: "one day past horizon", ReleaseDate: "2026-10-15"},
		{ID: 6, Title: "far future", ReleaseDate: "2028-01-01"},
		{ID: 7, Title: "no release date", ReleaseDate: ""},
		{ID: 8, Title: "unparseable", ReleaseDate: "tbd"},
	}

	got := svc.FilterFarFutureMovies(movies)

	// Expected: keep IDs 1, 2, 3, 4, 7, 8 (past/near-future/unknown)
	// Drop IDs 5, 6 (strictly after horizon)
	var gotIDs []int
	for _, m := range got {
		gotIDs = append(gotIDs, m.ID)
	}
	assert.Equal(t, []int{1, 2, 3, 4, 7, 8}, gotIDs)
}

func TestContentFilterService_FilterFarFutureTVShows(t *testing.T) {
	now := time.Date(2026, 4, 14, 0, 0, 0, 0, time.UTC)
	svc := NewContentFilterServiceWithClock(fixedClock(now))

	shows := []tmdb.TVShow{
		{ID: 1, Name: "old", FirstAirDate: "2015-06-01"},
		{ID: 2, Name: "within", FirstAirDate: "2026-09-01"},
		{ID: 3, Name: "far future", FirstAirDate: "2027-06-01"},
		{ID: 4, Name: "empty date", FirstAirDate: ""},
	}

	got := svc.FilterFarFutureTVShows(shows)

	var gotIDs []int
	for _, s := range got {
		gotIDs = append(gotIDs, s.ID)
	}
	assert.Equal(t, []int{1, 2, 4}, gotIDs)
}

func TestContentFilterService_FilterLowQualityMovies(t *testing.T) {
	svc := NewContentFilterService()

	movies := []tmdb.Movie{
		// Good quality
		{ID: 1, Title: "high rating many votes", VoteAverage: 8.5, VoteCount: 1000},
		// Low rating but enough votes → keep (signal, not noise)
		{ID: 2, Title: "low rating many votes", VoteAverage: 2.5, VoteCount: 500},
		// High rating few votes → keep (niche but positive)
		{ID: 3, Title: "high rating few votes", VoteAverage: 8.0, VoteCount: 20},
		// BOTH low → drop
		{ID: 4, Title: "bad obscure", VoteAverage: 2.0, VoteCount: 10},
		// Boundary: rating exactly 3.0 → NOT < 3.0, so keep
		{ID: 5, Title: "rating on boundary", VoteAverage: 3.0, VoteCount: 10},
		// Boundary: votes exactly 50 → NOT < 50, so keep
		{ID: 6, Title: "votes on boundary", VoteAverage: 1.0, VoteCount: 50},
		// Just below both thresholds → drop
		{ID: 7, Title: "just below", VoteAverage: 2.9, VoteCount: 49},
	}

	got := svc.FilterLowQualityMovies(movies)

	var gotIDs []int
	for _, m := range got {
		gotIDs = append(gotIDs, m.ID)
	}
	// Dropped: 4, 7
	assert.Equal(t, []int{1, 2, 3, 5, 6}, gotIDs)
}

func TestContentFilterService_FilterLowQualityTVShows(t *testing.T) {
	svc := NewContentFilterService()

	shows := []tmdb.TVShow{
		{ID: 1, Name: "keep", VoteAverage: 7.0, VoteCount: 200},
		{ID: 2, Name: "drop", VoteAverage: 2.5, VoteCount: 10},
		{ID: 3, Name: "keep-low-rating-many-votes", VoteAverage: 2.0, VoteCount: 300},
	}

	got := svc.FilterLowQualityTVShows(shows)

	var gotIDs []int
	for _, s := range got {
		gotIDs = append(gotIDs, s.ID)
	}
	assert.Equal(t, []int{1, 3}, gotIDs)
}

func TestContentFilterService_EmptyInputPassThrough(t *testing.T) {
	svc := NewContentFilterService()

	assert.Empty(t, svc.FilterFarFutureMovies(nil))
	assert.Empty(t, svc.FilterFarFutureTVShows(nil))
	assert.Empty(t, svc.FilterLowQualityMovies(nil))
	assert.Empty(t, svc.FilterLowQualityTVShows(nil))

	assert.Empty(t, svc.FilterFarFutureMovies([]tmdb.Movie{}))
	assert.Empty(t, svc.FilterFarFutureTVShows([]tmdb.TVShow{}))
}

func TestContentFilterService_ConstantsAreSensible(t *testing.T) {
	// Guard against accidental threshold loosening — AC #3/#4 specify these.
	assert.Equal(t, 6, FarFutureHorizonMonths)
	assert.Equal(t, 3.0, LowQualityRatingThreshold)
	assert.Equal(t, 50, LowQualityVoteCountThreshold)
}
