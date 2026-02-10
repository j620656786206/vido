package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
)

func setupRecentMediaTestRouter(handler *RecentMediaHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	apiV1 := router.Group("/api/v1")
	handler.RegisterRoutes(apiV1)
	return router
}

func TestRecentMediaHandler_GetRecentMedia(t *testing.T) {
	now := time.Now()
	recentTime := now.Add(-2 * time.Minute) // within 5 min = justAdded
	olderTime := now.Add(-10 * time.Minute) // beyond 5 min

	tests := []struct {
		name           string
		queryLimit     string
		setupMocks     func(*MockMovieService, *MockSeriesService)
		expectedStatus int
		expectedCount  int
		checkJustAdded bool
	}{
		{
			name:       "success - returns combined movies and series",
			queryLimit: "",
			setupMocks: func(ms *MockMovieService, ss *MockSeriesService) {
				ms.On("List", mock.Anything, mock.AnythingOfType("repository.ListParams")).Return(
					[]models.Movie{
						{
							ID:         "movie-1",
							Title:      "Test Movie",
							ReleaseDate: "2024-06-15",
							PosterPath: sql.NullString{String: "/poster1.jpg", Valid: true},
							CreatedAt:  recentTime,
						},
					},
					&repository.PaginationResult{Page: 1, PageSize: 10, TotalResults: 1, TotalPages: 1},
					nil,
				)
				ss.On("List", mock.Anything, mock.AnythingOfType("repository.ListParams")).Return(
					[]models.Series{
						{
							ID:           "series-1",
							Title:        "Test Series",
							FirstAirDate: "2023-01-10",
							PosterPath:   sql.NullString{String: "/poster2.jpg", Valid: true},
							CreatedAt:    olderTime,
						},
					},
					&repository.PaginationResult{Page: 1, PageSize: 10, TotalResults: 1, TotalPages: 1},
					nil,
				)
			},
			expectedStatus: http.StatusOK,
			expectedCount:  2,
			checkJustAdded: true,
		},
		{
			name:       "success - custom limit",
			queryLimit: "5",
			setupMocks: func(ms *MockMovieService, ss *MockSeriesService) {
				ms.On("List", mock.Anything, mock.AnythingOfType("repository.ListParams")).Return(
					[]models.Movie{},
					&repository.PaginationResult{Page: 1, PageSize: 5, TotalResults: 0, TotalPages: 0},
					nil,
				)
				ss.On("List", mock.Anything, mock.AnythingOfType("repository.ListParams")).Return(
					[]models.Series{},
					&repository.PaginationResult{Page: 1, PageSize: 5, TotalResults: 0, TotalPages: 0},
					nil,
				)
			},
			expectedStatus: http.StatusOK,
			expectedCount:  0,
		},
		{
			name:       "error - movie service fails",
			queryLimit: "",
			setupMocks: func(ms *MockMovieService, ss *MockSeriesService) {
				ms.On("List", mock.Anything, mock.AnythingOfType("repository.ListParams")).Return(
					nil, nil, errors.New("db error"),
				)
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:       "error - series service fails",
			queryLimit: "",
			setupMocks: func(ms *MockMovieService, ss *MockSeriesService) {
				ms.On("List", mock.Anything, mock.AnythingOfType("repository.ListParams")).Return(
					[]models.Movie{},
					&repository.PaginationResult{Page: 1, PageSize: 10, TotalResults: 0, TotalPages: 0},
					nil,
				)
				ss.On("List", mock.Anything, mock.AnythingOfType("repository.ListParams")).Return(
					nil, nil, errors.New("db error"),
				)
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:       "success - invalid limit defaults to 10",
			queryLimit: "abc",
			setupMocks: func(ms *MockMovieService, ss *MockSeriesService) {
				ms.On("List", mock.Anything, mock.AnythingOfType("repository.ListParams")).Return(
					[]models.Movie{},
					&repository.PaginationResult{Page: 1, PageSize: 10, TotalResults: 0, TotalPages: 0},
					nil,
				)
				ss.On("List", mock.Anything, mock.AnythingOfType("repository.ListParams")).Return(
					[]models.Series{},
					&repository.PaginationResult{Page: 1, PageSize: 10, TotalResults: 0, TotalPages: 0},
					nil,
				)
			},
			expectedStatus: http.StatusOK,
			expectedCount:  0,
		},
		{
			name:       "success - limit exceeding 50 defaults to 10",
			queryLimit: "100",
			setupMocks: func(ms *MockMovieService, ss *MockSeriesService) {
				ms.On("List", mock.Anything, mock.AnythingOfType("repository.ListParams")).Return(
					[]models.Movie{},
					&repository.PaginationResult{Page: 1, PageSize: 10, TotalResults: 0, TotalPages: 0},
					nil,
				)
				ss.On("List", mock.Anything, mock.AnythingOfType("repository.ListParams")).Return(
					[]models.Series{},
					&repository.PaginationResult{Page: 1, PageSize: 10, TotalResults: 0, TotalPages: 0},
					nil,
				)
			},
			expectedStatus: http.StatusOK,
			expectedCount:  0,
		},
		{
			name:       "success - zero limit defaults to 10",
			queryLimit: "0",
			setupMocks: func(ms *MockMovieService, ss *MockSeriesService) {
				ms.On("List", mock.Anything, mock.AnythingOfType("repository.ListParams")).Return(
					[]models.Movie{},
					&repository.PaginationResult{Page: 1, PageSize: 10, TotalResults: 0, TotalPages: 0},
					nil,
				)
				ss.On("List", mock.Anything, mock.AnythingOfType("repository.ListParams")).Return(
					[]models.Series{},
					&repository.PaginationResult{Page: 1, PageSize: 10, TotalResults: 0, TotalPages: 0},
					nil,
				)
			},
			expectedStatus: http.StatusOK,
			expectedCount:  0,
		},
		{
			name:       "success - negative limit defaults to 10",
			queryLimit: "-5",
			setupMocks: func(ms *MockMovieService, ss *MockSeriesService) {
				ms.On("List", mock.Anything, mock.AnythingOfType("repository.ListParams")).Return(
					[]models.Movie{},
					&repository.PaginationResult{Page: 1, PageSize: 10, TotalResults: 0, TotalPages: 0},
					nil,
				)
				ss.On("List", mock.Anything, mock.AnythingOfType("repository.ListParams")).Return(
					[]models.Series{},
					&repository.PaginationResult{Page: 1, PageSize: 10, TotalResults: 0, TotalPages: 0},
					nil,
				)
			},
			expectedStatus: http.StatusOK,
			expectedCount:  0,
		},
		{
			name:       "success - results sorted by addedAt descending",
			queryLimit: "",
			setupMocks: func(ms *MockMovieService, ss *MockSeriesService) {
				ms.On("List", mock.Anything, mock.AnythingOfType("repository.ListParams")).Return(
					[]models.Movie{
						{
							ID:          "movie-old",
							Title:       "Old Movie",
							ReleaseDate: "2020-01-01",
							PosterPath:  sql.NullString{String: "", Valid: false},
							CreatedAt:   olderTime,
						},
					},
					&repository.PaginationResult{Page: 1, PageSize: 10, TotalResults: 1, TotalPages: 1},
					nil,
				)
				ss.On("List", mock.Anything, mock.AnythingOfType("repository.ListParams")).Return(
					[]models.Series{
						{
							ID:           "series-new",
							Title:        "New Series",
							FirstAirDate: "2025-06-01",
							PosterPath:   sql.NullString{String: "/new.jpg", Valid: true},
							CreatedAt:    recentTime,
						},
					},
					&repository.PaginationResult{Page: 1, PageSize: 10, TotalResults: 1, TotalPages: 1},
					nil,
				)
			},
			expectedStatus: http.StatusOK,
			expectedCount:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockMovieSvc := &MockMovieService{}
			mockSeriesSvc := &MockSeriesService{}
			tt.setupMocks(mockMovieSvc, mockSeriesSvc)

			handler := NewRecentMediaHandler(mockMovieSvc, mockSeriesSvc)
			router := setupRecentMediaTestRouter(handler)

			url := "/api/v1/media/recent"
			if tt.queryLimit != "" {
				url += "?limit=" + tt.queryLimit
			}

			req, _ := http.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				var resp APIResponse
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				assert.NoError(t, err)
				assert.True(t, resp.Success)

				items, ok := resp.Data.([]interface{})
				assert.True(t, ok)
				assert.Len(t, items, tt.expectedCount)

				if tt.checkJustAdded && len(items) == 2 {
					// First item should be the movie (more recent)
					firstItem := items[0].(map[string]interface{})
					assert.Equal(t, "movie-1", firstItem["id"])
					assert.Equal(t, "movie", firstItem["mediaType"])
					assert.Equal(t, true, firstItem["justAdded"])
					assert.Equal(t, float64(2024), firstItem["year"])

					// Second item should be the series (older)
					secondItem := items[1].(map[string]interface{})
					assert.Equal(t, "series-1", secondItem["id"])
					assert.Equal(t, "tv", secondItem["mediaType"])
					assert.Equal(t, false, secondItem["justAdded"])
				}

				// Verify sort order for the sort test case
				if tt.name == "success - results sorted by addedAt descending" && len(items) == 2 {
					firstItem := items[0].(map[string]interface{})
					secondItem := items[1].(map[string]interface{})
					// series-new (recentTime) should come before movie-old (olderTime)
					assert.Equal(t, "series-new", firstItem["id"])
					assert.Equal(t, "movie-old", secondItem["id"])
				}
			}

			mockMovieSvc.AssertExpectations(t)
			mockSeriesSvc.AssertExpectations(t)
		})
	}
}
