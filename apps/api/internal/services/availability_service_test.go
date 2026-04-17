package services

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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
