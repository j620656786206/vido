// Package services ContentFilterService — Story 10-1.
//
// Applies post-fetch noise filters to TMDb trending/discover results before
// returning them to the frontend (per AC #3, #4):
//
//   - FarFuture: excludes items whose release / first-air date is more than
//     FarFutureHorizonMonths in the future. Items with an empty or
//     unparseable date are kept (we don't silently drop data we can't assess).
//   - LowQuality: excludes items with vote_average < LowQualityRatingThreshold
//     AND vote_count < LowQualityVoteCountThreshold (AND, not OR — a
//     low-rated item with many votes is still signal; a low-count item with
//     a high rating is kept).
//
// `now` is injected to keep FarFuture tests deterministic; NewContentFilterService
// wires it to time.Now.
package services

import (
	"time"

	"github.com/vido/api/internal/tmdb"
)

const (
	// FarFutureHorizonMonths — items releasing further than this many months
	// out are excluded from trending/discover results (P2-004).
	FarFutureHorizonMonths = 6

	// LowQualityRatingThreshold — TMDb vote_average below this counts as low quality.
	LowQualityRatingThreshold = 3.0

	// LowQualityVoteCountThreshold — TMDb vote_count below this is considered
	// statistically unreliable. Only items failing BOTH thresholds are dropped.
	LowQualityVoteCountThreshold = 50

	// tmdbDateLayout matches the YYYY-MM-DD format TMDb returns.
	tmdbDateLayout = "2006-01-02"
)

// ContentFilterService filters TMDb results post-fetch.
type ContentFilterService struct {
	now func() time.Time
}

// NewContentFilterService constructs a ContentFilterService using the real clock.
func NewContentFilterService() *ContentFilterService {
	return &ContentFilterService{now: time.Now}
}

// NewContentFilterServiceWithClock is the test constructor — allows a
// deterministic `now` for FarFuture assertions.
func NewContentFilterServiceWithClock(now func() time.Time) *ContentFilterService {
	return &ContentFilterService{now: now}
}

// FilterFarFutureMovies returns movies whose ReleaseDate is not more than
// FarFutureHorizonMonths from now. Items with an empty or unparseable date
// are retained (conservative stance — see package comment).
func (s *ContentFilterService) FilterFarFutureMovies(movies []tmdb.Movie) []tmdb.Movie {
	if len(movies) == 0 {
		return movies
	}
	horizon := s.now().UTC().AddDate(0, FarFutureHorizonMonths, 0)
	out := make([]tmdb.Movie, 0, len(movies))
	for _, m := range movies {
		if isWithinHorizon(m.ReleaseDate, horizon) {
			out = append(out, m)
		}
	}
	return out
}

// FilterFarFutureTVShows returns TV shows whose FirstAirDate is within
// FarFutureHorizonMonths from now.
func (s *ContentFilterService) FilterFarFutureTVShows(shows []tmdb.TVShow) []tmdb.TVShow {
	if len(shows) == 0 {
		return shows
	}
	horizon := s.now().UTC().AddDate(0, FarFutureHorizonMonths, 0)
	out := make([]tmdb.TVShow, 0, len(shows))
	for _, show := range shows {
		if isWithinHorizon(show.FirstAirDate, horizon) {
			out = append(out, show)
		}
	}
	return out
}

// FilterLowQualityMovies removes movies with both low rating AND low vote count.
func (s *ContentFilterService) FilterLowQualityMovies(movies []tmdb.Movie) []tmdb.Movie {
	if len(movies) == 0 {
		return movies
	}
	out := make([]tmdb.Movie, 0, len(movies))
	for _, m := range movies {
		if !isLowQuality(m.VoteAverage, m.VoteCount) {
			out = append(out, m)
		}
	}
	return out
}

// FilterLowQualityTVShows removes TV shows with both low rating AND low vote count.
func (s *ContentFilterService) FilterLowQualityTVShows(shows []tmdb.TVShow) []tmdb.TVShow {
	if len(shows) == 0 {
		return shows
	}
	out := make([]tmdb.TVShow, 0, len(shows))
	for _, show := range shows {
		if !isLowQuality(show.VoteAverage, show.VoteCount) {
			out = append(out, show)
		}
	}
	return out
}

// isWithinHorizon returns true when dateStr is empty, unparseable, or on/before
// the horizon. Parseable dates strictly AFTER the horizon are excluded. Both
// parsed date and horizon must be in UTC so the comparison has no timezone drift
// on servers running in non-UTC local time (e.g. Asia/Taipei).
func isWithinHorizon(dateStr string, horizon time.Time) bool {
	if dateStr == "" {
		return true
	}
	t, err := time.Parse(tmdbDateLayout, dateStr)
	if err != nil {
		return true
	}
	return !t.After(horizon)
}

// isLowQuality returns true when BOTH rating and vote count are below their
// respective thresholds (AND semantics per AC #4).
func isLowQuality(voteAverage float64, voteCount int) bool {
	return voteAverage < LowQualityRatingThreshold && voteCount < LowQualityVoteCountThreshold
}
