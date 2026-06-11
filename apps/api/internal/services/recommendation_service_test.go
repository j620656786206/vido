package services

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/tmdb"
)

// recsTMDbMock embeds the full-interface mockTMDbServiceForExplore and overrides
// only the recommendations/similar methods Story 12-3 exercises.
type recsTMDbMock struct {
	mockTMDbServiceForExplore

	movieRecs, movieSim *tmdb.SearchResultMovies
	tvRecs, tvSim       *tmdb.SearchResultTVShows

	movieRecsErr, movieSimErr error
	tvRecsErr, tvSimErr       error

	movieSimCalled, tvSimCalled int
}

func (m *recsTMDbMock) GetMovieRecommendations(_ context.Context, _ int) (*tmdb.SearchResultMovies, error) {
	if m.movieRecsErr != nil {
		return nil, m.movieRecsErr
	}
	if m.movieRecs != nil {
		return m.movieRecs, nil
	}
	return &tmdb.SearchResultMovies{}, nil
}

func (m *recsTMDbMock) GetMovieSimilar(_ context.Context, _ int) (*tmdb.SearchResultMovies, error) {
	m.movieSimCalled++
	if m.movieSimErr != nil {
		return nil, m.movieSimErr
	}
	if m.movieSim != nil {
		return m.movieSim, nil
	}
	return &tmdb.SearchResultMovies{}, nil
}

func (m *recsTMDbMock) GetTVRecommendations(_ context.Context, _ int) (*tmdb.SearchResultTVShows, error) {
	if m.tvRecsErr != nil {
		return nil, m.tvRecsErr
	}
	if m.tvRecs != nil {
		return m.tvRecs, nil
	}
	return &tmdb.SearchResultTVShows{}, nil
}

func (m *recsTMDbMock) GetTVSimilar(_ context.Context, _ int) (*tmdb.SearchResultTVShows, error) {
	m.tvSimCalled++
	if m.tvSimErr != nil {
		return nil, m.tvSimErr
	}
	if m.tvSim != nil {
		return m.tvSim, nil
	}
	return &tmdb.SearchResultTVShows{}, nil
}

// recsMovieRepo / recsSeriesRepo embed the full-interface PQ repo mocks and
// override the single ownership-lookup method.
type recsMovieRepo struct {
	mockPQMovieRepo
	owned []int64
	err   error
}

func (r *recsMovieRepo) FindOwnedTMDbIDs(_ context.Context, _ []int64) ([]int64, error) {
	return r.owned, r.err
}

type recsSeriesRepo struct {
	mockPQSeriesRepo
	owned []int64
	err   error
}

func (r *recsSeriesRepo) FindOwnedTMDbIDs(_ context.Context, _ []int64) ([]int64, error) {
	return r.owned, r.err
}

func strPtr(s string) *string { return &s }

func TestRecommendationService_MovieRecommendations_Source(t *testing.T) {
	tmdbMock := &recsTMDbMock{
		movieRecs: &tmdb.SearchResultMovies{Results: []tmdb.Movie{
			{ID: 603, Title: "The Matrix", ReleaseDate: "1999-03-31", VoteAverage: 8.2, PosterPath: strPtr("/m.jpg")},
			{ID: 604, Title: "Matrix Reloaded"},
		}},
	}
	svc := NewRecommendationService(tmdbMock, &recsMovieRepo{owned: []int64{603}}, &recsSeriesRepo{})

	res, err := svc.GetMovieRecommendations(context.Background(), 1)

	require.NoError(t, err)
	assert.Equal(t, "recommendations", res.Source)
	require.Len(t, res.Items, 2)
	assert.Equal(t, "movie", res.Items[0].MediaType)
	assert.Equal(t, "The Matrix", res.Items[0].Title)
	assert.True(t, res.Items[0].IsOwned, "603 is owned")
	assert.False(t, res.Items[1].IsOwned, "604 not owned")
	assert.Equal(t, 0, tmdbMock.movieSimCalled, "similar not called when recommendations non-empty")
}

func TestRecommendationService_MovieFallsBackToSimilar(t *testing.T) {
	tmdbMock := &recsTMDbMock{
		movieRecs: &tmdb.SearchResultMovies{Results: []tmdb.Movie{}}, // empty → fall back
		movieSim:  &tmdb.SearchResultMovies{Results: []tmdb.Movie{{ID: 700, Title: "Similar"}}},
	}
	svc := NewRecommendationService(tmdbMock, &recsMovieRepo{}, &recsSeriesRepo{})

	res, err := svc.GetMovieRecommendations(context.Background(), 1)

	require.NoError(t, err)
	assert.Equal(t, "similar", res.Source)
	require.Len(t, res.Items, 1)
	assert.Equal(t, 700, res.Items[0].ID)
	assert.Equal(t, 1, tmdbMock.movieSimCalled)
}

func TestRecommendationService_BothEmpty_SourceEmpty(t *testing.T) {
	tmdbMock := &recsTMDbMock{
		movieRecs: &tmdb.SearchResultMovies{Results: []tmdb.Movie{}},
		movieSim:  &tmdb.SearchResultMovies{Results: []tmdb.Movie{}},
	}
	svc := NewRecommendationService(tmdbMock, &recsMovieRepo{}, &recsSeriesRepo{})

	res, err := svc.GetMovieRecommendations(context.Background(), 1)

	require.NoError(t, err)
	assert.Equal(t, "", res.Source)
	assert.Empty(t, res.Items)
}

func TestRecommendationService_TMDBError_Propagates(t *testing.T) {
	tmdbMock := &recsTMDbMock{movieRecsErr: tmdb.NewTimeoutError(errors.New("upstream timeout"))}
	svc := NewRecommendationService(tmdbMock, &recsMovieRepo{}, &recsSeriesRepo{})

	_, err := svc.GetMovieRecommendations(context.Background(), 1)

	require.Error(t, err)
	var tmdbErr *tmdb.TMDbError
	require.True(t, errors.As(err, &tmdbErr))
	assert.Equal(t, tmdb.ErrCodeTimeout, tmdbErr.Code)
}

func TestRecommendationService_OwnershipError_DegradesNotFails(t *testing.T) {
	tmdbMock := &recsTMDbMock{
		movieRecs: &tmdb.SearchResultMovies{Results: []tmdb.Movie{{ID: 603, Title: "Matrix"}}},
	}
	svc := NewRecommendationService(tmdbMock,
		&recsMovieRepo{err: errors.New("db down")}, &recsSeriesRepo{})

	res, err := svc.GetMovieRecommendations(context.Background(), 1)

	require.NoError(t, err, "ownership-lookup error must NOT fail the call (Rule 27 Pillar 3)")
	require.Len(t, res.Items, 1)
	assert.False(t, res.Items[0].IsOwned, "degrades to not-owned on lookup error")
}

func TestRecommendationService_CapsAt18(t *testing.T) {
	results := make([]tmdb.Movie, 30)
	for i := range results {
		results[i] = tmdb.Movie{ID: i + 1, Title: "M"}
	}
	tmdbMock := &recsTMDbMock{movieRecs: &tmdb.SearchResultMovies{Results: results}}
	svc := NewRecommendationService(tmdbMock, &recsMovieRepo{}, &recsSeriesRepo{})

	res, err := svc.GetMovieRecommendations(context.Background(), 1)

	require.NoError(t, err)
	assert.Len(t, res.Items, maxRecommendationItems)
}

func TestRecommendationService_TVRecommendations_OwnershipAndShape(t *testing.T) {
	tmdbMock := &recsTMDbMock{
		tvRecs: &tmdb.SearchResultTVShows{Results: []tmdb.TVShow{
			{ID: 1396, Name: "Breaking Bad", FirstAirDate: "2008-01-20", VoteAverage: 8.9},
		}},
	}
	svc := NewRecommendationService(tmdbMock, &recsMovieRepo{}, &recsSeriesRepo{owned: []int64{1396}})

	res, err := svc.GetTVRecommendations(context.Background(), 1)

	require.NoError(t, err)
	assert.Equal(t, "recommendations", res.Source)
	require.Len(t, res.Items, 1)
	assert.Equal(t, "tv", res.Items[0].MediaType)
	assert.Equal(t, "Breaking Bad", res.Items[0].Title)
	assert.Equal(t, "2008-01-20", res.Items[0].ReleaseDate)
	assert.True(t, res.Items[0].IsOwned)
}

func TestRecommendationService_RejectsNonPositiveID(t *testing.T) {
	svc := NewRecommendationService(&recsTMDbMock{}, &recsMovieRepo{}, &recsSeriesRepo{})
	_, err := svc.GetMovieRecommendations(context.Background(), 0)
	require.Error(t, err)
	_, err = svc.GetTVRecommendations(context.Background(), -5)
	require.Error(t, err)
}
