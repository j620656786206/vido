package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/douban"
	"github.com/vido/api/internal/services"
)

func TestDoubanReviewSummaryHandler_MovieSuccess(t *testing.T) {
	svc := new(MockDoubanRatingService)
	svc.On("EnrichDoubanReviewSummary", mock.Anything, "m1", "movie").
		Return(&douban.ReviewSummaryResult{
			ID:            "1292052",
			TotalComments: 152340,
			TopComments:   []douban.ReviewComment{{Author: "甲", Rating: 5, Text: "這部電影太棒了"}},
		}, nil)

	router := setupDoubanRatingRouter(NewDoubanRatingHandler(svc))
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/movies/m1/douban-review-summary", nil)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Success bool `json:"success"`
		Data    *struct {
			ID            string `json:"id"`
			TotalComments int    `json:"total_comments"`
			TopComments   []struct {
				Author string `json:"author"`
				Rating int    `json:"rating"`
				Text   string `json:"text"`
			} `json:"top_comments"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.True(t, resp.Success)
	require.NotNil(t, resp.Data)
	assert.Equal(t, "1292052", resp.Data.ID)
	assert.Equal(t, 152340, resp.Data.TotalComments)
	require.Len(t, resp.Data.TopComments, 1)
	assert.Equal(t, "甲", resp.Data.TopComments[0].Author)
	assert.Equal(t, 5, resp.Data.TopComments[0].Rating)
	assert.Equal(t, "這部電影太棒了", resp.Data.TopComments[0].Text)
	svc.AssertExpectations(t)
}

func TestDoubanReviewSummaryHandler_SeriesSuccess(t *testing.T) {
	svc := new(MockDoubanRatingService)
	svc.On("EnrichDoubanReviewSummary", mock.Anything, "s1", "series").
		Return(&douban.ReviewSummaryResult{ID: "26794435", TotalComments: 10}, nil)

	router := setupDoubanRatingRouter(NewDoubanRatingHandler(svc))
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/series/s1/douban-review-summary", nil)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	svc.AssertExpectations(t)
}

func TestDoubanReviewSummaryHandler_NullOnDegradation(t *testing.T) {
	svc := new(MockDoubanRatingService)
	// nil result, nil error = graceful degradation (no review summary).
	svc.On("EnrichDoubanReviewSummary", mock.Anything, "m1", "movie").Return(nil, nil)

	router := setupDoubanRatingRouter(NewDoubanRatingHandler(svc))
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/movies/m1/douban-review-summary", nil)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Success bool            `json:"success"`
		Data    json.RawMessage `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.True(t, resp.Success)
	assert.Equal(t, "null", string(resp.Data), "missing review summary serializes as data: null")
}

func TestDoubanReviewSummaryHandler_NotFound(t *testing.T) {
	svc := new(MockDoubanRatingService)
	svc.On("EnrichDoubanReviewSummary", mock.Anything, "missing", "movie").Return(nil, services.ErrMediaNotFound)

	router := setupDoubanRatingRouter(NewDoubanRatingHandler(svc))
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/movies/missing/douban-review-summary", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
