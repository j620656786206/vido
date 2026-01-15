package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
)

// MockMovieService is a mock implementation of MovieServiceInterface
type MockMovieService struct {
	mock.Mock
}

func (m *MockMovieService) Create(ctx context.Context, movie *models.Movie) error {
	args := m.Called(ctx, movie)
	return args.Error(0)
}

func (m *MockMovieService) GetByID(ctx context.Context, id string) (*models.Movie, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Movie), args.Error(1)
}

func (m *MockMovieService) GetByTMDbID(ctx context.Context, tmdbID int64) (*models.Movie, error) {
	args := m.Called(ctx, tmdbID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Movie), args.Error(1)
}

func (m *MockMovieService) Update(ctx context.Context, movie *models.Movie) error {
	args := m.Called(ctx, movie)
	return args.Error(0)
}

func (m *MockMovieService) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockMovieService) List(ctx context.Context, params repository.ListParams) ([]models.Movie, *repository.PaginationResult, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, nil, args.Error(2)
	}
	return args.Get(0).([]models.Movie), args.Get(1).(*repository.PaginationResult), args.Error(2)
}

func (m *MockMovieService) SearchByTitle(ctx context.Context, title string, params repository.ListParams) ([]models.Movie, *repository.PaginationResult, error) {
	args := m.Called(ctx, title, params)
	if args.Get(0) == nil {
		return nil, nil, args.Error(2)
	}
	return args.Get(0).([]models.Movie), args.Get(1).(*repository.PaginationResult), args.Error(2)
}

// Verify mock implements interface
var _ MovieServiceInterface = (*MockMovieService)(nil)

func setupTestRouter(handler *MovieHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	apiV1 := router.Group("/api/v1")
	handler.RegisterRoutes(apiV1)
	return router
}

func TestMovieHandler_List(t *testing.T) {
	tests := []struct {
		name           string
		setupMock      func(*MockMovieService)
		expectedStatus int
		expectedCount  int
	}{
		{
			name: "success - returns movies",
			setupMock: func(m *MockMovieService) {
				m.On("List", mock.Anything, mock.AnythingOfType("repository.ListParams")).Return(
					[]models.Movie{
						{ID: "1", Title: "Movie 1", ReleaseDate: "2024-01-01"},
						{ID: "2", Title: "Movie 2", ReleaseDate: "2024-02-01"},
					},
					&repository.PaginationResult{Page: 1, PageSize: 20, TotalResults: 2, TotalPages: 1},
					nil,
				)
			},
			expectedStatus: http.StatusOK,
			expectedCount:  2,
		},
		{
			name: "success - empty list",
			setupMock: func(m *MockMovieService) {
				m.On("List", mock.Anything, mock.AnythingOfType("repository.ListParams")).Return(
					[]models.Movie{},
					&repository.PaginationResult{Page: 1, PageSize: 20, TotalResults: 0, TotalPages: 0},
					nil,
				)
			},
			expectedStatus: http.StatusOK,
			expectedCount:  0,
		},
		{
			name: "error - service failure",
			setupMock: func(m *MockMovieService) {
				m.On("List", mock.Anything, mock.AnythingOfType("repository.ListParams")).Return(
					nil, nil, errors.New("database error"),
				)
			},
			expectedStatus: http.StatusInternalServerError,
			expectedCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockMovieService)
			tt.setupMock(mockService)

			handler := NewMovieHandler(mockService)
			router := setupTestRouter(handler)

			req, _ := http.NewRequest(http.MethodGet, "/api/v1/movies", nil)
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			assert.Equal(t, tt.expectedStatus, resp.Code)

			if tt.expectedStatus == http.StatusOK {
				var response APIResponse
				err := json.Unmarshal(resp.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.True(t, response.Success)

				// Parse the data as PaginatedResponse
				dataBytes, _ := json.Marshal(response.Data)
				var paginated PaginatedResponse
				json.Unmarshal(dataBytes, &paginated)

				items, ok := paginated.Items.([]interface{})
				if ok {
					assert.Equal(t, tt.expectedCount, len(items))
				}
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestMovieHandler_GetByID(t *testing.T) {
	tests := []struct {
		name           string
		movieID        string
		setupMock      func(*MockMovieService)
		expectedStatus int
	}{
		{
			name:    "success",
			movieID: "movie-123",
			setupMock: func(m *MockMovieService) {
				m.On("GetByID", mock.Anything, "movie-123").Return(
					&models.Movie{ID: "movie-123", Title: "Test Movie", ReleaseDate: "2024-01-01"},
					nil,
				)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:    "not found",
			movieID: "nonexistent",
			setupMock: func(m *MockMovieService) {
				m.On("GetByID", mock.Anything, "nonexistent").Return(nil, errors.New("not found"))
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockMovieService)
			tt.setupMock(mockService)

			handler := NewMovieHandler(mockService)
			router := setupTestRouter(handler)

			req, _ := http.NewRequest(http.MethodGet, "/api/v1/movies/"+tt.movieID, nil)
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			assert.Equal(t, tt.expectedStatus, resp.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestMovieHandler_Create(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		setupMock      func(*MockMovieService)
		expectedStatus int
	}{
		{
			name: "success",
			requestBody: CreateMovieRequest{
				Title:       "New Movie",
				ReleaseDate: "2024-06-15",
			},
			setupMock: func(m *MockMovieService) {
				m.On("Create", mock.Anything, mock.AnythingOfType("*models.Movie")).Return(nil)
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "validation error - missing title",
			requestBody: map[string]string{
				"releaseDate": "2024-06-15",
			},
			setupMock:      func(m *MockMovieService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "validation error - missing release date",
			requestBody: map[string]string{
				"title": "New Movie",
			},
			setupMock:      func(m *MockMovieService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "service error",
			requestBody: CreateMovieRequest{
				Title:       "New Movie",
				ReleaseDate: "2024-06-15",
			},
			setupMock: func(m *MockMovieService) {
				m.On("Create", mock.Anything, mock.AnythingOfType("*models.Movie")).Return(errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockMovieService)
			tt.setupMock(mockService)

			handler := NewMovieHandler(mockService)
			router := setupTestRouter(handler)

			body, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest(http.MethodPost, "/api/v1/movies", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			assert.Equal(t, tt.expectedStatus, resp.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestMovieHandler_Delete(t *testing.T) {
	tests := []struct {
		name           string
		movieID        string
		setupMock      func(*MockMovieService)
		expectedStatus int
	}{
		{
			name:    "success",
			movieID: "movie-123",
			setupMock: func(m *MockMovieService) {
				m.On("Delete", mock.Anything, "movie-123").Return(nil)
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name:    "service error",
			movieID: "movie-456",
			setupMock: func(m *MockMovieService) {
				m.On("Delete", mock.Anything, "movie-456").Return(errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockMovieService)
			tt.setupMock(mockService)

			handler := NewMovieHandler(mockService)
			router := setupTestRouter(handler)

			req, _ := http.NewRequest(http.MethodDelete, "/api/v1/movies/"+tt.movieID, nil)
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			assert.Equal(t, tt.expectedStatus, resp.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestMovieHandler_Search(t *testing.T) {
	tests := []struct {
		name           string
		query          string
		setupMock      func(*MockMovieService)
		expectedStatus int
	}{
		{
			name:  "success",
			query: "test",
			setupMock: func(m *MockMovieService) {
				m.On("SearchByTitle", mock.Anything, "test", mock.AnythingOfType("repository.ListParams")).Return(
					[]models.Movie{{ID: "1", Title: "Test Movie", ReleaseDate: "2024-01-01"}},
					&repository.PaginationResult{Page: 1, PageSize: 20, TotalResults: 1, TotalPages: 1},
					nil,
				)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "missing query parameter",
			query:          "",
			setupMock:      func(m *MockMovieService) {},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockMovieService)
			tt.setupMock(mockService)

			handler := NewMovieHandler(mockService)
			router := setupTestRouter(handler)

			url := "/api/v1/movies/search"
			if tt.query != "" {
				url += "?q=" + tt.query
			}
			req, _ := http.NewRequest(http.MethodGet, url, nil)
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			assert.Equal(t, tt.expectedStatus, resp.Code)
			mockService.AssertExpectations(t)
		})
	}
}
