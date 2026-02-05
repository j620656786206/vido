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
	"github.com/vido/api/internal/learning"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/services"
)

// MockLearningService implements services.LearningServiceInterface for testing
type MockLearningService struct {
	learnResult      *models.FilenameMapping
	learnErr         error
	findMatchResult  *learning.MatchResult
	findMatchErr     error
	statsResult      *services.PatternStats
	statsErr         error
	listResult       []*models.FilenameMapping
	listErr          error
	deleteErr        error
	applyErr         error

	// Track calls
	learnCalled   bool
	learnReq      services.LearnFromCorrectionRequest
	findCalled    bool
	findFilename  string
	deleteCalled  bool
	deleteID      string
	applyCalled   bool
	applyID       string
}

func (m *MockLearningService) LearnFromCorrection(ctx context.Context, req services.LearnFromCorrectionRequest) (*models.FilenameMapping, error) {
	m.learnCalled = true
	m.learnReq = req
	return m.learnResult, m.learnErr
}

func (m *MockLearningService) FindMatchingPattern(ctx context.Context, filename string) (*learning.MatchResult, error) {
	m.findCalled = true
	m.findFilename = filename
	return m.findMatchResult, m.findMatchErr
}

func (m *MockLearningService) GetPatternStats(ctx context.Context) (*services.PatternStats, error) {
	return m.statsResult, m.statsErr
}

func (m *MockLearningService) ListPatterns(ctx context.Context) ([]*models.FilenameMapping, error) {
	return m.listResult, m.listErr
}

func (m *MockLearningService) DeletePattern(ctx context.Context, id string) error {
	m.deleteCalled = true
	m.deleteID = id
	return m.deleteErr
}

func (m *MockLearningService) ApplyPattern(ctx context.Context, id string) error {
	m.applyCalled = true
	m.applyID = id
	return m.applyErr
}

func setupLearningTestRouter(service *MockLearningService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	handler := NewLearningHandler(service)
	api := router.Group("/api/v1")
	handler.RegisterRoutes(api)

	return router
}

func TestLearningHandler_Create(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		mockResult     *models.FilenameMapping
		mockErr        error
		expectedStatus int
		expectedError  bool
		checkRequest   func(t *testing.T, req services.LearnFromCorrectionRequest)
	}{
		{
			name: "successful pattern creation",
			requestBody: map[string]interface{}{
				"filename":     "[Leopard-Raws] Kimetsu no Yaiba - 26.mkv",
				"metadataId":   "series-123",
				"metadataType": "series",
				"tmdbId":       85937,
			},
			mockResult: &models.FilenameMapping{
				ID:           "pattern-456",
				Pattern:      "[Leopard-Raws] Kimetsu no Yaiba",
				FansubGroup:  "Leopard-Raws",
				TitlePattern: "Kimetsu no Yaiba",
				PatternType:  "fansub",
				MetadataType: "series",
				MetadataID:   "series-123",
				TmdbID:       85937,
			},
			expectedStatus: http.StatusCreated,
			checkRequest: func(t *testing.T, req services.LearnFromCorrectionRequest) {
				assert.Equal(t, "[Leopard-Raws] Kimetsu no Yaiba - 26.mkv", req.Filename)
				assert.Equal(t, "series-123", req.MetadataID)
				assert.Equal(t, "series", req.MetadataType)
				assert.Equal(t, 85937, req.TmdbID)
			},
		},
		{
			name: "missing filename",
			requestBody: map[string]interface{}{
				"metadataId":   "series-123",
				"metadataType": "series",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  true,
		},
		{
			name: "missing metadataId",
			requestBody: map[string]interface{}{
				"filename":     "test.mkv",
				"metadataType": "series",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  true,
		},
		{
			name: "missing metadataType",
			requestBody: map[string]interface{}{
				"filename":   "test.mkv",
				"metadataId": "series-123",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  true,
		},
		{
			name: "invalid metadataType",
			requestBody: map[string]interface{}{
				"filename":     "test.mkv",
				"metadataId":   "series-123",
				"metadataType": "invalid",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  true,
		},
		{
			name: "service error",
			requestBody: map[string]interface{}{
				"filename":     "test.mkv",
				"metadataId":   "series-123",
				"metadataType": "series",
			},
			mockErr:        errors.New("service error"),
			expectedStatus: http.StatusInternalServerError,
			expectedError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockLearningService{
				learnResult: tt.mockResult,
				learnErr:    tt.mockErr,
			}

			router := setupLearningTestRouter(mockService)

			body, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest("POST", "/api/v1/learning/patterns", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response APIResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			if tt.expectedError {
				assert.False(t, response.Success)
				assert.NotNil(t, response.Error)
			} else {
				assert.True(t, response.Success)
				assert.Nil(t, response.Error)
			}

			if tt.checkRequest != nil && mockService.learnCalled {
				tt.checkRequest(t, mockService.learnReq)
			}
		})
	}
}

func TestLearningHandler_List(t *testing.T) {
	tests := []struct {
		name           string
		mockPatterns   []*models.FilenameMapping
		mockStats      *services.PatternStats
		mockErr        error
		statsErr       error
		expectedStatus int
		expectedError  bool
	}{
		{
			name: "successful list with patterns",
			mockPatterns: []*models.FilenameMapping{
				{
					ID:           "pattern-1",
					Pattern:      "[Leopard-Raws] Kimetsu no Yaiba",
					MetadataType: "series",
					UseCount:     12,
				},
				{
					ID:           "pattern-2",
					Pattern:      "Breaking Bad",
					MetadataType: "series",
					UseCount:     5,
				},
			},
			mockStats: &services.PatternStats{
				TotalPatterns:   2,
				TotalApplied:    17,
				MostUsedPattern: "[Leopard-Raws] Kimetsu no Yaiba",
				MostUsedCount:   12,
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:         "empty patterns list",
			mockPatterns: []*models.FilenameMapping{},
			mockStats: &services.PatternStats{
				TotalPatterns: 0,
				TotalApplied:  0,
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "service error on list",
			mockErr:        errors.New("database error"),
			expectedStatus: http.StatusInternalServerError,
			expectedError:  true,
		},
		{
			name: "stats error falls back gracefully",
			mockPatterns: []*models.FilenameMapping{
				{ID: "pattern-1", Pattern: "Test"},
			},
			statsErr:       errors.New("stats error"),
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockLearningService{
				listResult:  tt.mockPatterns,
				listErr:     tt.mockErr,
				statsResult: tt.mockStats,
				statsErr:    tt.statsErr,
			}

			router := setupLearningTestRouter(mockService)

			req, _ := http.NewRequest("GET", "/api/v1/learning/patterns", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response APIResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			if tt.expectedError {
				assert.False(t, response.Success)
			} else {
				assert.True(t, response.Success)
			}
		})
	}
}

func TestLearningHandler_Delete(t *testing.T) {
	tests := []struct {
		name           string
		patternID      string
		mockErr        error
		expectedStatus int
		expectedError  bool
	}{
		{
			name:           "successful delete",
			patternID:      "pattern-123",
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "pattern not found",
			patternID:      "non-existent",
			mockErr:        errors.New("not found"),
			expectedStatus: http.StatusInternalServerError,
			expectedError:  true,
		},
		{
			name:           "empty pattern ID",
			patternID:      "",
			expectedStatus: http.StatusNotFound, // Gin router won't match empty param
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockLearningService{
				deleteErr: tt.mockErr,
			}

			router := setupLearningTestRouter(mockService)

			req, _ := http.NewRequest("DELETE", "/api/v1/learning/patterns/"+tt.patternID, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.patternID != "" {
				assert.True(t, mockService.deleteCalled)
				assert.Equal(t, tt.patternID, mockService.deleteID)
			}
		})
	}
}

func TestLearningHandler_GetStats(t *testing.T) {
	tests := []struct {
		name           string
		mockStats      *services.PatternStats
		mockErr        error
		expectedStatus int
		expectedError  bool
	}{
		{
			name: "successful stats",
			mockStats: &services.PatternStats{
				TotalPatterns:   15,
				TotalApplied:    48,
				MostUsedPattern: "[Leopard-Raws]",
				MostUsedCount:   12,
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "stats error",
			mockErr:        errors.New("database error"),
			expectedStatus: http.StatusInternalServerError,
			expectedError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockLearningService{
				statsResult: tt.mockStats,
				statsErr:    tt.mockErr,
			}

			router := setupLearningTestRouter(mockService)

			req, _ := http.NewRequest("GET", "/api/v1/learning/stats", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response APIResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			if tt.expectedError {
				assert.False(t, response.Success)
			} else {
				assert.True(t, response.Success)
				// Verify stats structure
				if tt.mockStats != nil {
					data, ok := response.Data.(map[string]interface{})
					assert.True(t, ok)
					assert.Equal(t, float64(tt.mockStats.TotalPatterns), data["totalPatterns"])
					assert.Equal(t, float64(tt.mockStats.TotalApplied), data["totalApplied"])
				}
			}
		})
	}
}
