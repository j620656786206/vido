package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/metadata"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/testutil"
)

// fakeDoubanSearcher is a controllable DoubanSearcher for service tests.
type fakeDoubanSearcher struct {
	available bool
	result    *metadata.SearchResult
	err       error
	called    bool
	lastReq   *metadata.SearchRequest
}

func (f *fakeDoubanSearcher) IsAvailable() bool { return f.available }

func (f *fakeDoubanSearcher) Search(_ context.Context, req *metadata.SearchRequest) (*metadata.SearchResult, error) {
	f.called = true
	f.lastReq = req
	return f.result, f.err
}

func searchResultWith(id string, rating float64, voteCount int) *metadata.SearchResult {
	return &metadata.SearchResult{
		Items: []metadata.MetadataItem{{ID: id, Rating: rating, VoteCount: voteCount}},
	}
}

func TestEnrichDoubanRating_CacheHit(t *testing.T) {
	movieRepo := new(testutil.MockMovieRepository)
	seriesRepo := new(testutil.MockSeriesRepository)
	searcher := &fakeDoubanSearcher{available: true}

	cached := &models.Movie{
		ID:              "m1",
		Title:           "霸王別姬",
		DoubanID:        models.NewNullString("1291546"),
		DoubanRating:    models.NewNullFloat64(9.6),
		DoubanVoteCount: models.NewNullInt64(1500000),
	}
	movieRepo.On("FindByID", mock.Anything, "m1").Return(cached, nil)

	svc := NewDoubanRatingService(searcher, movieRepo, seriesRepo)
	result, err := svc.EnrichDoubanRating(context.Background(), "m1", "movie")

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "1291546", result.DoubanID)
	assert.Equal(t, 9.6, result.DoubanRating)
	assert.Equal(t, 1500000, result.DoubanVoteCount)
	assert.False(t, searcher.called, "cache hit must NOT trigger a Douban search")
	movieRepo.AssertNotCalled(t, "UpdateDoubanRating", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestEnrichDoubanRating_CacheMissPersists(t *testing.T) {
	movieRepo := new(testutil.MockMovieRepository)
	seriesRepo := new(testutil.MockSeriesRepository)
	searcher := &fakeDoubanSearcher{available: true, result: searchResultWith("1292052", 9.7, 2130000)}

	uncached := &models.Movie{ID: "m1", Title: "肖申克的救贖", ReleaseDate: "1994-09-23"}
	movieRepo.On("FindByID", mock.Anything, "m1").Return(uncached, nil)
	movieRepo.On("UpdateDoubanRating", mock.Anything, "m1", "1292052", 9.7, 2130000).Return(nil)

	svc := NewDoubanRatingService(searcher, movieRepo, seriesRepo)
	result, err := svc.EnrichDoubanRating(context.Background(), "m1", "movie")

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "1292052", result.DoubanID)
	assert.Equal(t, 9.7, result.DoubanRating)
	assert.True(t, searcher.called)
	assert.Equal(t, 1994, searcher.lastReq.Year, "year parsed from release_date")
	assert.Equal(t, metadata.MediaTypeMovie, searcher.lastReq.MediaType)
	movieRepo.AssertExpectations(t)
}

func TestEnrichDoubanRating_SeriesCacheMissPersists(t *testing.T) {
	movieRepo := new(testutil.MockMovieRepository)
	seriesRepo := new(testutil.MockSeriesRepository)
	searcher := &fakeDoubanSearcher{available: true, result: searchResultWith("26794435", 9.4, 880000)}

	uncached := &models.Series{ID: "s1", Title: "權力的遊戲", FirstAirDate: "2011-04-17"}
	seriesRepo.On("FindByID", mock.Anything, "s1").Return(uncached, nil)
	seriesRepo.On("UpdateDoubanRating", mock.Anything, "s1", "26794435", 9.4, 880000).Return(nil)

	svc := NewDoubanRatingService(searcher, movieRepo, seriesRepo)
	result, err := svc.EnrichDoubanRating(context.Background(), "s1", "series")

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 9.4, result.DoubanRating)
	assert.Equal(t, metadata.MediaTypeTV, searcher.lastReq.MediaType)
	seriesRepo.AssertExpectations(t)
}

func TestEnrichDoubanRating_SearchErrorDegrades(t *testing.T) {
	movieRepo := new(testutil.MockMovieRepository)
	seriesRepo := new(testutil.MockSeriesRepository)
	searcher := &fakeDoubanSearcher{available: true, err: errors.New("DOUBAN_BLOCKED")}

	movieRepo.On("FindByID", mock.Anything, "m1").Return(&models.Movie{ID: "m1", Title: "x"}, nil)

	svc := NewDoubanRatingService(searcher, movieRepo, seriesRepo)
	result, err := svc.EnrichDoubanRating(context.Background(), "m1", "movie")

	require.NoError(t, err, "Douban failure must degrade gracefully, not error")
	assert.Nil(t, result)
	movieRepo.AssertNotCalled(t, "UpdateDoubanRating", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestEnrichDoubanRating_NoResultsDegrades(t *testing.T) {
	movieRepo := new(testutil.MockMovieRepository)
	seriesRepo := new(testutil.MockSeriesRepository)
	searcher := &fakeDoubanSearcher{available: true, result: &metadata.SearchResult{Items: nil}}

	movieRepo.On("FindByID", mock.Anything, "m1").Return(&models.Movie{ID: "m1", Title: "x"}, nil)

	svc := NewDoubanRatingService(searcher, movieRepo, seriesRepo)
	result, err := svc.EnrichDoubanRating(context.Background(), "m1", "movie")

	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestEnrichDoubanRating_ZeroRatingDegrades(t *testing.T) {
	movieRepo := new(testutil.MockMovieRepository)
	seriesRepo := new(testutil.MockSeriesRepository)
	searcher := &fakeDoubanSearcher{available: true, result: searchResultWith("123", 0, 0)}

	movieRepo.On("FindByID", mock.Anything, "m1").Return(&models.Movie{ID: "m1", Title: "x"}, nil)

	svc := NewDoubanRatingService(searcher, movieRepo, seriesRepo)
	result, err := svc.EnrichDoubanRating(context.Background(), "m1", "movie")

	require.NoError(t, err)
	assert.Nil(t, result, "a match with no usable rating is treated as no rating")
}

func TestEnrichDoubanRating_NilSearcherDegrades(t *testing.T) {
	movieRepo := new(testutil.MockMovieRepository)
	seriesRepo := new(testutil.MockSeriesRepository)

	movieRepo.On("FindByID", mock.Anything, "m1").Return(&models.Movie{ID: "m1", Title: "x"}, nil)

	svc := NewDoubanRatingService(nil, movieRepo, seriesRepo)
	result, err := svc.EnrichDoubanRating(context.Background(), "m1", "movie")

	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestEnrichDoubanRating_SearcherUnavailableDegrades(t *testing.T) {
	movieRepo := new(testutil.MockMovieRepository)
	seriesRepo := new(testutil.MockSeriesRepository)
	searcher := &fakeDoubanSearcher{available: false}

	movieRepo.On("FindByID", mock.Anything, "m1").Return(&models.Movie{ID: "m1", Title: "x"}, nil)

	svc := NewDoubanRatingService(searcher, movieRepo, seriesRepo)
	result, err := svc.EnrichDoubanRating(context.Background(), "m1", "movie")

	require.NoError(t, err)
	assert.Nil(t, result)
	assert.False(t, searcher.called, "unavailable searcher must not be queried")
}

func TestEnrichDoubanRating_InvalidMediaType(t *testing.T) {
	svc := NewDoubanRatingService(&fakeDoubanSearcher{available: true}, new(testutil.MockMovieRepository), new(testutil.MockSeriesRepository))
	result, err := svc.EnrichDoubanRating(context.Background(), "x1", "person")
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestEnrichDoubanRating_RepoErrorPropagates(t *testing.T) {
	movieRepo := new(testutil.MockMovieRepository)
	seriesRepo := new(testutil.MockSeriesRepository)
	movieRepo.On("FindByID", mock.Anything, "missing").Return((*models.Movie)(nil), errors.New("movie not found"))

	svc := NewDoubanRatingService(&fakeDoubanSearcher{available: true}, movieRepo, seriesRepo)
	result, err := svc.EnrichDoubanRating(context.Background(), "missing", "movie")

	require.Error(t, err, "a missing media record is a hard error (→ 404)")
	assert.Nil(t, result)
}

func TestEnrichDoubanRating_PicksYearMatch(t *testing.T) {
	// H1: Douban returns several same-title subjects; the service must pick the
	// one whose year matches the record, not Douban's first hit.
	movieRepo := new(testutil.MockMovieRepository)
	seriesRepo := new(testutil.MockSeriesRepository)
	searcher := &fakeDoubanSearcher{
		available: true,
		result: &metadata.SearchResult{Items: []metadata.MetadataItem{
			{ID: "wrong", Year: 2007, Rating: 6.1, VoteCount: 12000}, // first hit, wrong year
			{ID: "right", Year: 1994, Rating: 9.7, VoteCount: 2130000},
		}},
	}

	uncached := &models.Movie{ID: "m1", Title: "肖申克的救贖", ReleaseDate: "1994-09-23"}
	movieRepo.On("FindByID", mock.Anything, "m1").Return(uncached, nil)
	movieRepo.On("UpdateDoubanRating", mock.Anything, "m1", "right", 9.7, 2130000).Return(nil)

	svc := NewDoubanRatingService(searcher, movieRepo, seriesRepo)
	result, err := svc.EnrichDoubanRating(context.Background(), "m1", "movie")

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "right", result.DoubanID, "exact year match must win over Douban's first hit")
	assert.Equal(t, 9.7, result.DoubanRating)
	movieRepo.AssertExpectations(t)
}

func TestEnrichDoubanRating_NoYearFallsBackToFirstRated(t *testing.T) {
	// H1: with no usable record year, the first subject carrying a rating wins
	// (skipping any zero-rating subjects ahead of it).
	movieRepo := new(testutil.MockMovieRepository)
	seriesRepo := new(testutil.MockSeriesRepository)
	searcher := &fakeDoubanSearcher{
		available: true,
		result: &metadata.SearchResult{Items: []metadata.MetadataItem{
			{ID: "norating", Year: 2001, Rating: 0, VoteCount: 0}, // skipped
			{ID: "first", Year: 2002, Rating: 8.2, VoteCount: 5000},
		}},
	}

	uncached := &models.Movie{ID: "m1", Title: "x"} // no ReleaseDate → year 0
	movieRepo.On("FindByID", mock.Anything, "m1").Return(uncached, nil)
	movieRepo.On("UpdateDoubanRating", mock.Anything, "m1", "first", 8.2, 5000).Return(nil)

	svc := NewDoubanRatingService(searcher, movieRepo, seriesRepo)
	result, err := svc.EnrichDoubanRating(context.Background(), "m1", "movie")

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "first", result.DoubanID)
	movieRepo.AssertExpectations(t)
}

func TestEnrichDoubanRating_SeriesCacheHit(t *testing.T) {
	// L2: series cache-hit returns the persisted rating without a search.
	movieRepo := new(testutil.MockMovieRepository)
	seriesRepo := new(testutil.MockSeriesRepository)
	searcher := &fakeDoubanSearcher{available: true}

	cached := &models.Series{
		ID:              "s1",
		Title:           "權力的遊戲",
		DoubanID:        models.NewNullString("3016187"),
		DoubanRating:    models.NewNullFloat64(9.4),
		DoubanVoteCount: models.NewNullInt64(880000),
	}
	seriesRepo.On("FindByID", mock.Anything, "s1").Return(cached, nil)

	svc := NewDoubanRatingService(searcher, movieRepo, seriesRepo)
	result, err := svc.EnrichDoubanRating(context.Background(), "s1", "series")

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "3016187", result.DoubanID)
	assert.Equal(t, 9.4, result.DoubanRating)
	assert.Equal(t, 880000, result.DoubanVoteCount)
	assert.False(t, searcher.called, "cache hit must NOT trigger a Douban search")
	seriesRepo.AssertNotCalled(t, "UpdateDoubanRating", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestEnrichDoubanRating_NotFoundMapsToSentinel(t *testing.T) {
	// M2: a sql.ErrNoRows from the repo surfaces as ErrMediaNotFound (→ 404);
	// any other repo error propagates unchanged (→ 500).
	movieRepo := new(testutil.MockMovieRepository)
	seriesRepo := new(testutil.MockSeriesRepository)
	movieRepo.On("FindByID", mock.Anything, "missing").
		Return((*models.Movie)(nil), fmt.Errorf("movie with id missing not found: %w", sql.ErrNoRows))

	svc := NewDoubanRatingService(&fakeDoubanSearcher{available: true}, movieRepo, seriesRepo)
	result, err := svc.EnrichDoubanRating(context.Background(), "missing", "movie")

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrMediaNotFound)
	assert.Nil(t, result)
}

func TestEnrichDoubanRating_InfraErrorNotSentinel(t *testing.T) {
	// M2: a non-not-found repo error must NOT be reported as ErrMediaNotFound.
	movieRepo := new(testutil.MockMovieRepository)
	seriesRepo := new(testutil.MockSeriesRepository)
	movieRepo.On("FindByID", mock.Anything, "m1").
		Return((*models.Movie)(nil), errors.New("failed to find movie: db connection reset"))

	svc := NewDoubanRatingService(&fakeDoubanSearcher{available: true}, movieRepo, seriesRepo)
	result, err := svc.EnrichDoubanRating(context.Background(), "m1", "movie")

	require.Error(t, err)
	assert.NotErrorIs(t, err, ErrMediaNotFound)
	assert.Nil(t, result)
}

func TestYearFromDate(t *testing.T) {
	assert.Equal(t, 1993, yearFromDate("1993-01-01"))
	assert.Equal(t, 2011, yearFromDate("2011"))
	assert.Equal(t, 0, yearFromDate(""))
	assert.Equal(t, 0, yearFromDate("abc"))
	assert.Equal(t, 0, yearFromDate("12"))
}
