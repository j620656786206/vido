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

// MockSeriesService is a mock implementation of SeriesServiceInterface
type MockSeriesService struct {
	mock.Mock
}

func (m *MockSeriesService) Create(ctx context.Context, series *models.Series) error {
	args := m.Called(ctx, series)
	return args.Error(0)
}

func (m *MockSeriesService) GetByID(ctx context.Context, id string) (*models.Series, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Series), args.Error(1)
}

func (m *MockSeriesService) GetByTMDbID(ctx context.Context, tmdbID int64) (*models.Series, error) {
	args := m.Called(ctx, tmdbID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Series), args.Error(1)
}

func (m *MockSeriesService) Update(ctx context.Context, series *models.Series) error {
	args := m.Called(ctx, series)
	return args.Error(0)
}

func (m *MockSeriesService) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockSeriesService) List(ctx context.Context, params repository.ListParams) ([]models.Series, *repository.PaginationResult, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, nil, args.Error(2)
	}
	return args.Get(0).([]models.Series), args.Get(1).(*repository.PaginationResult), args.Error(2)
}

func (m *MockSeriesService) SearchByTitle(ctx context.Context, title string, params repository.ListParams) ([]models.Series, *repository.PaginationResult, error) {
	args := m.Called(ctx, title, params)
	if args.Get(0) == nil {
		return nil, nil, args.Error(2)
	}
	return args.Get(0).([]models.Series), args.Get(1).(*repository.PaginationResult), args.Error(2)
}

// Verify mock implements interface
var _ SeriesServiceInterface = (*MockSeriesService)(nil)

func setupSeriesTestRouter(handler *SeriesHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	apiV1 := router.Group("/api/v1")
	handler.RegisterRoutes(apiV1)
	return router
}

func TestSeriesHandler_List(t *testing.T) {
	tests := []struct {
		name           string
		setupMock      func(*MockSeriesService)
		expectedStatus int
		expectedCount  int
	}{
		{
			name: "success - returns series",
			setupMock: func(m *MockSeriesService) {
				m.On("List", mock.Anything, mock.AnythingOfType("repository.ListParams")).Return(
					[]models.Series{
						{ID: "1", Title: "Series 1", FirstAirDate: "2024-01-01"},
						{ID: "2", Title: "Series 2", FirstAirDate: "2024-02-01"},
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
			setupMock: func(m *MockSeriesService) {
				m.On("List", mock.Anything, mock.AnythingOfType("repository.ListParams")).Return(
					[]models.Series{},
					&repository.PaginationResult{Page: 1, PageSize: 20, TotalResults: 0, TotalPages: 0},
					nil,
				)
			},
			expectedStatus: http.StatusOK,
			expectedCount:  0,
		},
		{
			name: "error - service failure",
			setupMock: func(m *MockSeriesService) {
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
			mockService := new(MockSeriesService)
			tt.setupMock(mockService)

			handler := NewSeriesHandler(mockService)
			router := setupSeriesTestRouter(handler)

			req, _ := http.NewRequest(http.MethodGet, "/api/v1/series", nil)
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			assert.Equal(t, tt.expectedStatus, resp.Code)

			if tt.expectedStatus == http.StatusOK {
				var response APIResponse
				err := json.Unmarshal(resp.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.True(t, response.Success)

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

func TestSeriesHandler_GetByID(t *testing.T) {
	tests := []struct {
		name           string
		seriesID       string
		setupMock      func(*MockSeriesService)
		expectedStatus int
	}{
		{
			name:     "success",
			seriesID: "series-123",
			setupMock: func(m *MockSeriesService) {
				m.On("GetByID", mock.Anything, "series-123").Return(
					&models.Series{ID: "series-123", Title: "Test Series", FirstAirDate: "2024-01-01"},
					nil,
				)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:     "not found",
			seriesID: "nonexistent",
			setupMock: func(m *MockSeriesService) {
				m.On("GetByID", mock.Anything, "nonexistent").Return(nil, errors.New("not found"))
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockSeriesService)
			tt.setupMock(mockService)

			handler := NewSeriesHandler(mockService)
			router := setupSeriesTestRouter(handler)

			req, _ := http.NewRequest(http.MethodGet, "/api/v1/series/"+tt.seriesID, nil)
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			assert.Equal(t, tt.expectedStatus, resp.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestSeriesHandler_Create(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		setupMock      func(*MockSeriesService)
		expectedStatus int
	}{
		{
			name: "success",
			requestBody: CreateSeriesRequest{
				Title:        "New Series",
				FirstAirDate: "2024-06-15",
			},
			setupMock: func(m *MockSeriesService) {
				m.On("Create", mock.Anything, mock.AnythingOfType("*models.Series")).Return(nil)
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "validation error - missing title",
			requestBody: map[string]string{
				"firstAirDate": "2024-06-15",
			},
			setupMock:      func(m *MockSeriesService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "validation error - missing first air date",
			requestBody: map[string]string{
				"title": "New Series",
			},
			setupMock:      func(m *MockSeriesService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "service error",
			requestBody: CreateSeriesRequest{
				Title:        "New Series",
				FirstAirDate: "2024-06-15",
			},
			setupMock: func(m *MockSeriesService) {
				m.On("Create", mock.Anything, mock.AnythingOfType("*models.Series")).Return(errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockSeriesService)
			tt.setupMock(mockService)

			handler := NewSeriesHandler(mockService)
			router := setupSeriesTestRouter(handler)

			body, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest(http.MethodPost, "/api/v1/series", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			assert.Equal(t, tt.expectedStatus, resp.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestSeriesHandler_Update(t *testing.T) {
	tests := []struct {
		name           string
		seriesID       string
		requestBody    interface{}
		setupMock      func(*MockSeriesService)
		expectedStatus int
	}{
		{
			name:     "success",
			seriesID: "series-123",
			requestBody: UpdateSeriesRequest{
				Title: "Updated Series",
			},
			setupMock: func(m *MockSeriesService) {
				m.On("GetByID", mock.Anything, "series-123").Return(
					&models.Series{ID: "series-123", Title: "Original Series", FirstAirDate: "2024-01-01"},
					nil,
				)
				m.On("Update", mock.Anything, mock.AnythingOfType("*models.Series")).Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:     "not found",
			seriesID: "nonexistent",
			requestBody: UpdateSeriesRequest{
				Title: "Updated Series",
			},
			setupMock: func(m *MockSeriesService) {
				m.On("GetByID", mock.Anything, "nonexistent").Return(nil, errors.New("not found"))
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:     "service error on update",
			seriesID: "series-456",
			requestBody: UpdateSeriesRequest{
				Title: "Updated Series",
			},
			setupMock: func(m *MockSeriesService) {
				m.On("GetByID", mock.Anything, "series-456").Return(
					&models.Series{ID: "series-456", Title: "Original Series", FirstAirDate: "2024-01-01"},
					nil,
				)
				m.On("Update", mock.Anything, mock.AnythingOfType("*models.Series")).Return(errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockSeriesService)
			tt.setupMock(mockService)

			handler := NewSeriesHandler(mockService)
			router := setupSeriesTestRouter(handler)

			body, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest(http.MethodPut, "/api/v1/series/"+tt.seriesID, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			assert.Equal(t, tt.expectedStatus, resp.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestSeriesHandler_Delete(t *testing.T) {
	tests := []struct {
		name           string
		seriesID       string
		setupMock      func(*MockSeriesService)
		expectedStatus int
	}{
		{
			name:     "success",
			seriesID: "series-123",
			setupMock: func(m *MockSeriesService) {
				m.On("Delete", mock.Anything, "series-123").Return(nil)
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name:     "service error",
			seriesID: "series-456",
			setupMock: func(m *MockSeriesService) {
				m.On("Delete", mock.Anything, "series-456").Return(errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockSeriesService)
			tt.setupMock(mockService)

			handler := NewSeriesHandler(mockService)
			router := setupSeriesTestRouter(handler)

			req, _ := http.NewRequest(http.MethodDelete, "/api/v1/series/"+tt.seriesID, nil)
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			assert.Equal(t, tt.expectedStatus, resp.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestSeriesHandler_Search(t *testing.T) {
	tests := []struct {
		name           string
		query          string
		setupMock      func(*MockSeriesService)
		expectedStatus int
	}{
		{
			name:  "success",
			query: "test",
			setupMock: func(m *MockSeriesService) {
				m.On("SearchByTitle", mock.Anything, "test", mock.AnythingOfType("repository.ListParams")).Return(
					[]models.Series{{ID: "1", Title: "Test Series", FirstAirDate: "2024-01-01"}},
					&repository.PaginationResult{Page: 1, PageSize: 20, TotalResults: 1, TotalPages: 1},
					nil,
				)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "missing query parameter",
			query:          "",
			setupMock:      func(m *MockSeriesService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:  "service error",
			query: "error",
			setupMock: func(m *MockSeriesService) {
				m.On("SearchByTitle", mock.Anything, "error", mock.AnythingOfType("repository.ListParams")).Return(
					nil, nil, errors.New("database error"),
				)
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockSeriesService)
			tt.setupMock(mockService)

			handler := NewSeriesHandler(mockService)
			router := setupSeriesTestRouter(handler)

			url := "/api/v1/series/search"
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

func TestSeriesHandler_CreateWithAllOptionalFields(t *testing.T) {
	mockService := new(MockSeriesService)
	mockService.On("Create", mock.Anything, mock.AnythingOfType("*models.Series")).Return(nil)

	handler := NewSeriesHandler(mockService)
	router := setupSeriesTestRouter(handler)

	// Create request with all optional fields
	requestBody := CreateSeriesRequest{
		Title:            "Full Series",
		OriginalTitle:    "Original Full Series",
		FirstAirDate:     "2024-01-15",
		Genres:           []string{"Drama", "Thriller"},
		Overview:         "A complete series description",
		PosterPath:       "/posters/series.jpg",
		NumberOfSeasons:  3,
		NumberOfEpisodes: 30,
		TMDbID:           12345,
		IMDbID:           "tt1234567",
	}

	body, _ := json.Marshal(requestBody)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/series", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusCreated, resp.Code)
	mockService.AssertExpectations(t)
}

func TestSeriesHandler_UpdateWithAllOptionalFields(t *testing.T) {
	mockService := new(MockSeriesService)
	mockService.On("GetByID", mock.Anything, "series-123").Return(
		&models.Series{ID: "series-123", Title: "Original Series", FirstAirDate: "2024-01-01"},
		nil,
	)
	mockService.On("Update", mock.Anything, mock.AnythingOfType("*models.Series")).Return(nil)

	handler := NewSeriesHandler(mockService)
	router := setupSeriesTestRouter(handler)

	// Update request with all optional fields
	inProd := true
	requestBody := UpdateSeriesRequest{
		Title:            "Updated Series",
		OriginalTitle:    "Original Updated Series",
		FirstAirDate:     "2024-02-20",
		LastAirDate:      "2024-12-20",
		Genres:           []string{"Comedy", "Romance"},
		Overview:         "Updated series description",
		PosterPath:       "/posters/updated.jpg",
		Rating:           8.5,
		NumberOfSeasons:  5,
		NumberOfEpisodes: 50,
		Status:           "Returning Series",
		InProduction:     &inProd,
	}

	body, _ := json.Marshal(requestBody)
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/series/series-123", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	mockService.AssertExpectations(t)
}

func TestSeriesHandler_UpdateInvalidJSON(t *testing.T) {
	mockService := new(MockSeriesService)
	mockService.On("GetByID", mock.Anything, "series-123").Return(
		&models.Series{ID: "series-123", Title: "Original Series", FirstAirDate: "2024-01-01"},
		nil,
	)

	handler := NewSeriesHandler(mockService)
	router := setupSeriesTestRouter(handler)

	// Invalid JSON body
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/series/series-123", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusBadRequest, resp.Code)
}
