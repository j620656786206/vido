package services

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/testutil"
)

func TestAvailabilityService_CheckOwned(t *testing.T) {
	ctx := context.Background()

	t.Run("empty input short-circuits — neither repo is called (AC #4)", func(t *testing.T) {
		movieRepo := new(testutil.MockMovieRepository)
		seriesRepo := new(testutil.MockSeriesRepository)

		svc := NewAvailabilityService(movieRepo, seriesRepo)
		owned, err := svc.CheckOwned(ctx, nil)

		assert.NoError(t, err)
		assert.NotNil(t, owned)
		assert.Len(t, owned, 0)
		movieRepo.AssertNotCalled(t, "FindOwnedTMDbIDs", mock.Anything, mock.Anything)
		seriesRepo.AssertNotCalled(t, "FindOwnedTMDbIDs", mock.Anything, mock.Anything)
	})

	t.Run("merges movie and series hits (AC #1)", func(t *testing.T) {
		movieRepo := new(testutil.MockMovieRepository)
		seriesRepo := new(testutil.MockSeriesRepository)

		ids := []int64{603, 157336, 1396, 999999}
		movieRepo.On("FindOwnedTMDbIDs", ctx, ids).Return([]int64{603, 157336}, nil)
		seriesRepo.On("FindOwnedTMDbIDs", ctx, ids).Return([]int64{1396}, nil)

		svc := NewAvailabilityService(movieRepo, seriesRepo)
		owned, err := svc.CheckOwned(ctx, ids)

		assert.NoError(t, err)
		assert.ElementsMatch(t, []int64{603, 157336, 1396}, owned)
		movieRepo.AssertExpectations(t)
		seriesRepo.AssertExpectations(t)
	})

	t.Run("deduplicates colliding ids across tables", func(t *testing.T) {
		movieRepo := new(testutil.MockMovieRepository)
		seriesRepo := new(testutil.MockSeriesRepository)

		ids := []int64{603}
		movieRepo.On("FindOwnedTMDbIDs", ctx, ids).Return([]int64{603}, nil)
		seriesRepo.On("FindOwnedTMDbIDs", ctx, ids).Return([]int64{603}, nil)

		svc := NewAvailabilityService(movieRepo, seriesRepo)
		owned, err := svc.CheckOwned(ctx, ids)

		assert.NoError(t, err)
		assert.Equal(t, []int64{603}, owned)
	})

	t.Run("movie repo error propagates", func(t *testing.T) {
		movieRepo := new(testutil.MockMovieRepository)
		seriesRepo := new(testutil.MockSeriesRepository)

		ids := []int64{603}
		movieRepo.On("FindOwnedTMDbIDs", ctx, ids).Return(nil, errors.New("db down"))
		// Series repo must NOT be called when movie repo fails first.
		svc := NewAvailabilityService(movieRepo, seriesRepo)

		owned, err := svc.CheckOwned(ctx, ids)

		assert.Error(t, err)
		assert.Nil(t, owned)
		seriesRepo.AssertNotCalled(t, "FindOwnedTMDbIDs", mock.Anything, mock.Anything)
	})

	t.Run("series repo error propagates", func(t *testing.T) {
		movieRepo := new(testutil.MockMovieRepository)
		seriesRepo := new(testutil.MockSeriesRepository)

		ids := []int64{603}
		movieRepo.On("FindOwnedTMDbIDs", ctx, ids).Return([]int64{603}, nil)
		seriesRepo.On("FindOwnedTMDbIDs", ctx, ids).Return(nil, errors.New("series repo down"))

		svc := NewAvailabilityService(movieRepo, seriesRepo)
		owned, err := svc.CheckOwned(ctx, ids)

		assert.Error(t, err)
		assert.Nil(t, owned)
	})
}

func TestAvailabilityService_CheckOwnedByType(t *testing.T) {
	// Story 13-3a CR M1 — the type-aware sibling: a movie query must never
	// match an owned SERIES sharing the same TMDb id (and vice versa).
	ctx := context.Background()

	t.Run("movie query consults only the movies table", func(t *testing.T) {
		movieRepo := new(testutil.MockMovieRepository)
		seriesRepo := new(testutil.MockSeriesRepository)
		movieRepo.On("FindOwnedTMDbIDs", ctx, []int64{550}).Return([]int64{550}, nil)

		svc := NewAvailabilityService(movieRepo, seriesRepo)
		owned, err := svc.CheckOwnedByType(ctx, models.RequestMediaTypeMovie, []int64{550})

		require.NoError(t, err)
		assert.Equal(t, []int64{550}, owned)
		seriesRepo.AssertNotCalled(t, "FindOwnedTMDbIDs", mock.Anything, mock.Anything)
	})

	t.Run("tv query consults only the series table", func(t *testing.T) {
		movieRepo := new(testutil.MockMovieRepository)
		seriesRepo := new(testutil.MockSeriesRepository)
		seriesRepo.On("FindOwnedTMDbIDs", ctx, []int64{1399}).Return([]int64{1399}, nil)

		svc := NewAvailabilityService(movieRepo, seriesRepo)
		owned, err := svc.CheckOwnedByType(ctx, models.RequestMediaTypeTV, []int64{1399})

		require.NoError(t, err)
		assert.Equal(t, []int64{1399}, owned)
		movieRepo.AssertNotCalled(t, "FindOwnedTMDbIDs", mock.Anything, mock.Anything)
	})

	t.Run("empty input returns empty without repo calls", func(t *testing.T) {
		movieRepo := new(testutil.MockMovieRepository)
		seriesRepo := new(testutil.MockSeriesRepository)

		svc := NewAvailabilityService(movieRepo, seriesRepo)
		owned, err := svc.CheckOwnedByType(ctx, models.RequestMediaTypeMovie, nil)

		require.NoError(t, err)
		assert.Empty(t, owned)
		movieRepo.AssertNotCalled(t, "FindOwnedTMDbIDs", mock.Anything, mock.Anything)
	})

	t.Run("repo error propagates", func(t *testing.T) {
		movieRepo := new(testutil.MockMovieRepository)
		seriesRepo := new(testutil.MockSeriesRepository)
		movieRepo.On("FindOwnedTMDbIDs", ctx, []int64{603}).Return(nil, errors.New("db down"))

		svc := NewAvailabilityService(movieRepo, seriesRepo)
		_, err := svc.CheckOwnedByType(ctx, models.RequestMediaTypeMovie, []int64{603})
		assert.Error(t, err)
	})
}

// stubLibraryIDRepo adds the id lookup to a repository double.
type stubLibraryIDRepo struct {
	ids map[int64]string
	err error
}

func (s stubLibraryIDRepo) FindLibraryIDByTMDbID(_ context.Context, tmdbID int64) (string, error) {
	return s.ids[tmdbID], s.err
}

func TestAvailabilityService_LibraryID_RoutesByTypeAndToleratesUnwiredRepos(t *testing.T) {
	svc := &AvailabilityService{
		movieIDs:  stubLibraryIDRepo{ids: map[int64]string{1: "movie-1"}},
		seriesIDs: stubLibraryIDRepo{ids: map[int64]string{1: "series-1"}},
	}
	ctx := context.Background()

	// Same numeric TMDb id, two titles: the type picks the table.
	got, err := svc.LibraryID(ctx, "tv", 1)
	if err != nil || got != "series-1" {
		t.Fatalf("tv: got %q, %v", got, err)
	}
	got, err = svc.LibraryID(ctx, "movie", 1)
	if err != nil || got != "movie-1" {
		t.Fatalf("movie: got %q, %v", got, err)
	}
	if got, _ := svc.LibraryID(ctx, "tv", 0); got != "" {
		t.Fatalf("tmdb 0: got %q", got)
	}

	// Repositories without the lookup (test doubles) answer "not in the library".
	if got, err := (&AvailabilityService{}).LibraryID(ctx, "tv", 1); got != "" || err != nil {
		t.Fatalf("unwired: got %q, %v", got, err)
	}
}
