package services

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/douban"
	"github.com/vido/api/internal/metadata"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/testutil"
)

// fakeReviewScraper is a controllable DoubanReviewScraper for service tests.
type fakeReviewScraper struct {
	result *douban.ReviewSummaryResult
	err    error
	called bool
	lastID string
}

func (f *fakeReviewScraper) ScrapeReviewSummary(_ context.Context, id string) (*douban.ReviewSummaryResult, error) {
	f.called = true
	f.lastID = id
	return f.result, f.err
}

func sampleSummary(id string) *douban.ReviewSummaryResult {
	return &douban.ReviewSummaryResult{
		ID:            id,
		TotalComments: 152340,
		TopComments:   []douban.ReviewComment{{Author: "甲", Rating: 5, Text: "這部電影太棒了"}},
	}
}

func TestEnrichDoubanReviewSummary_StoredID(t *testing.T) {
	movieRepo := new(testutil.MockMovieRepository)
	seriesRepo := new(testutil.MockSeriesRepository)
	scraper := &fakeReviewScraper{result: sampleSummary("1292052")}

	movie := &models.Movie{ID: "m1", Title: "肖申克的救贖", DoubanID: models.NewNullString("1292052")}
	movieRepo.On("FindByID", mock.Anything, "m1").Return(movie, nil)

	svc := NewDoubanRatingService(&fakeDoubanSearcher{available: true}, scraper, movieRepo, seriesRepo)
	result, err := svc.EnrichDoubanReviewSummary(context.Background(), "m1", "movie")

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "1292052", result.ID)
	assert.Equal(t, 152340, result.TotalComments)
	require.Len(t, result.TopComments, 1)
	assert.Equal(t, "甲", result.TopComments[0].Author)
	assert.True(t, scraper.called)
	assert.Equal(t, "1292052", scraper.lastID, "must scrape by the stored douban_id")
}

func TestEnrichDoubanReviewSummary_ResolvesViaLookup(t *testing.T) {
	// No stored douban_id → resolve via the rating lookup, persist, then scrape.
	movieRepo := new(testutil.MockMovieRepository)
	seriesRepo := new(testutil.MockSeriesRepository)
	searcher := &fakeDoubanSearcher{
		available: true,
		result:    &metadata.SearchResult{Items: []metadata.MetadataItem{{ID: "1292052", Year: 1994, Rating: 9.7, VoteCount: 2130000}}},
	}
	scraper := &fakeReviewScraper{result: sampleSummary("1292052")}

	movie := &models.Movie{ID: "m1", Title: "肖申克的救贖", ReleaseDate: "1994-09-23"}
	movieRepo.On("FindByID", mock.Anything, "m1").Return(movie, nil)
	movieRepo.On("UpdateDoubanRating", mock.Anything, "m1", "1292052", 9.7, 2130000).Return(nil)

	svc := NewDoubanRatingService(searcher, scraper, movieRepo, seriesRepo)
	result, err := svc.EnrichDoubanReviewSummary(context.Background(), "m1", "movie")

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "1292052", scraper.lastID, "scrapes by the freshly resolved id")
	movieRepo.AssertExpectations(t)
}

func TestEnrichDoubanReviewSummary_Unresolved(t *testing.T) {
	// No stored id and the searcher is unavailable → no id, no scrape, nil result.
	movieRepo := new(testutil.MockMovieRepository)
	seriesRepo := new(testutil.MockSeriesRepository)
	scraper := &fakeReviewScraper{result: sampleSummary("x")}

	movie := &models.Movie{ID: "m1", Title: "未匹配"}
	movieRepo.On("FindByID", mock.Anything, "m1").Return(movie, nil)

	svc := NewDoubanRatingService(&fakeDoubanSearcher{available: false}, scraper, movieRepo, seriesRepo)
	result, err := svc.EnrichDoubanReviewSummary(context.Background(), "m1", "movie")

	require.NoError(t, err)
	assert.Nil(t, result)
	assert.False(t, scraper.called, "no scrape when the id is unresolved (AC #4)")
}

func TestEnrichDoubanReviewSummary_ScrapeErrorDegrades(t *testing.T) {
	movieRepo := new(testutil.MockMovieRepository)
	seriesRepo := new(testutil.MockSeriesRepository)
	scraper := &fakeReviewScraper{err: errors.New("douban blocked")}

	movie := &models.Movie{ID: "m1", Title: "x", DoubanID: models.NewNullString("1292052")}
	movieRepo.On("FindByID", mock.Anything, "m1").Return(movie, nil)

	svc := NewDoubanRatingService(&fakeDoubanSearcher{available: true}, scraper, movieRepo, seriesRepo)
	result, err := svc.EnrichDoubanReviewSummary(context.Background(), "m1", "movie")

	require.NoError(t, err, "a scrape failure degrades to nil, not an error (AC #5)")
	assert.Nil(t, result)
}

func TestEnrichDoubanReviewSummary_NilScraper(t *testing.T) {
	movieRepo := new(testutil.MockMovieRepository)
	seriesRepo := new(testutil.MockSeriesRepository)

	svc := NewDoubanRatingService(&fakeDoubanSearcher{available: true}, nil, movieRepo, seriesRepo)
	result, err := svc.EnrichDoubanReviewSummary(context.Background(), "m1", "movie")

	require.NoError(t, err)
	assert.Nil(t, result, "no review scraper wired → omit section")
}

func TestEnrichDoubanReviewSummary_MediaNotFound(t *testing.T) {
	movieRepo := new(testutil.MockMovieRepository)
	seriesRepo := new(testutil.MockSeriesRepository)
	scraper := &fakeReviewScraper{}
	movieRepo.On("FindByID", mock.Anything, "missing").Return((*models.Movie)(nil), sql.ErrNoRows)

	svc := NewDoubanRatingService(&fakeDoubanSearcher{available: true}, scraper, movieRepo, seriesRepo)
	result, err := svc.EnrichDoubanReviewSummary(context.Background(), "missing", "movie")

	require.ErrorIs(t, err, ErrMediaNotFound, "a missing record maps to 404, not a degraded nil")
	assert.Nil(t, result)
	assert.False(t, scraper.called)
}

func TestEnrichDoubanReviewSummary_SeriesStoredID(t *testing.T) {
	movieRepo := new(testutil.MockMovieRepository)
	seriesRepo := new(testutil.MockSeriesRepository)
	scraper := &fakeReviewScraper{result: sampleSummary("3016187")}

	series := &models.Series{ID: "s1", Title: "權力的遊戲", DoubanID: models.NewNullString("3016187")}
	seriesRepo.On("FindByID", mock.Anything, "s1").Return(series, nil)

	svc := NewDoubanRatingService(&fakeDoubanSearcher{available: true}, scraper, movieRepo, seriesRepo)
	result, err := svc.EnrichDoubanReviewSummary(context.Background(), "s1", "series")

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "3016187", scraper.lastID)
}
