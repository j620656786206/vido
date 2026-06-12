package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/douban"
	"github.com/vido/api/internal/services"
)

// MockDoubanRatingService satisfies DoubanRatingServiceInterface for handler tests.
type MockDoubanRatingService struct {
	mock.Mock
}

func (m *MockDoubanRatingService) EnrichDoubanRating(ctx context.Context, mediaID, mediaType string) (*services.DoubanRatingResult, error) {
	args := m.Called(ctx, mediaID, mediaType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.DoubanRatingResult), args.Error(1)
}

func (m *MockDoubanRatingService) EnrichDoubanReviewSummary(ctx context.Context, mediaID, mediaType string) (*douban.ReviewSummaryResult, error) {
	args := m.Called(ctx, mediaID, mediaType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*douban.ReviewSummaryResult), args.Error(1)
}

var _ DoubanRatingServiceInterface = (*MockDoubanRatingService)(nil)

func setupDoubanRatingRouter(handler *DoubanRatingHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	apiV1 := router.Group("/api/v1")
	handler.RegisterRoutes(apiV1)
	return router
}

func TestDoubanRatingHandler_MovieSuccess(t *testing.T) {
	svc := new(MockDoubanRatingService)
	svc.On("EnrichDoubanRating", mock.Anything, "m1", "movie").
		Return(&services.DoubanRatingResult{DoubanID: "1292052", DoubanRating: 9.7, DoubanVoteCount: 2130000}, nil)

	router := setupDoubanRatingRouter(NewDoubanRatingHandler(svc))
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/movies/m1/douban-rating", nil)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Success bool                        `json:"success"`
		Data    *services.DoubanRatingResult `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.True(t, resp.Success)
	require.NotNil(t, resp.Data)
	assert.Equal(t, "1292052", resp.Data.DoubanID)
	assert.Equal(t, 9.7, resp.Data.DoubanRating)
	assert.Equal(t, 2130000, resp.Data.DoubanVoteCount)
	svc.AssertExpectations(t)
}

func TestDoubanRatingHandler_SeriesSuccess(t *testing.T) {
	svc := new(MockDoubanRatingService)
	svc.On("EnrichDoubanRating", mock.Anything, "s1", "series").
		Return(&services.DoubanRatingResult{DoubanID: "26794435", DoubanRating: 9.4, DoubanVoteCount: 880000}, nil)

	router := setupDoubanRatingRouter(NewDoubanRatingHandler(svc))
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/series/s1/douban-rating", nil)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	svc.AssertExpectations(t)
}

func TestDoubanRatingHandler_NullDataOnGracefulDegradation(t *testing.T) {
	svc := new(MockDoubanRatingService)
	// nil result, nil error = graceful degradation (no Douban data).
	svc.On("EnrichDoubanRating", mock.Anything, "m1", "movie").Return(nil, nil)

	router := setupDoubanRatingRouter(NewDoubanRatingHandler(svc))
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/movies/m1/douban-rating", nil)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Success bool            `json:"success"`
		Data    json.RawMessage `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.True(t, resp.Success)
	assert.Equal(t, "null", string(resp.Data), "missing Douban data serializes as data: null")
}

func TestDoubanRatingHandler_NotFoundOnMissingRecord(t *testing.T) {
	// M2: ErrMediaNotFound → 404.
	svc := new(MockDoubanRatingService)
	svc.On("EnrichDoubanRating", mock.Anything, "missing", "movie").Return(nil, services.ErrMediaNotFound)

	router := setupDoubanRatingRouter(NewDoubanRatingHandler(svc))
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/movies/missing/douban-rating", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDoubanRatingHandler_InternalErrorOnInfraFailure(t *testing.T) {
	// M2: a non-not-found error must surface as 500, not be masked as 404.
	svc := new(MockDoubanRatingService)
	svc.On("EnrichDoubanRating", mock.Anything, "m1", "movie").
		Return(nil, errors.New("failed to find movie: db connection reset"))

	router := setupDoubanRatingRouter(NewDoubanRatingHandler(svc))
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/movies/m1/douban-rating", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
