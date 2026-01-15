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
)

// MockSettingsService is a mock implementation of SettingsServiceInterface
type MockSettingsService struct {
	mock.Mock
}

func (m *MockSettingsService) Set(ctx context.Context, setting *models.Setting) error {
	args := m.Called(ctx, setting)
	return args.Error(0)
}

func (m *MockSettingsService) Get(ctx context.Context, key string) (*models.Setting, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Setting), args.Error(1)
}

func (m *MockSettingsService) GetAll(ctx context.Context) ([]models.Setting, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Setting), args.Error(1)
}

func (m *MockSettingsService) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *MockSettingsService) GetString(ctx context.Context, key string) (string, error) {
	args := m.Called(ctx, key)
	return args.String(0), args.Error(1)
}

func (m *MockSettingsService) GetInt(ctx context.Context, key string) (int, error) {
	args := m.Called(ctx, key)
	return args.Int(0), args.Error(1)
}

func (m *MockSettingsService) GetBool(ctx context.Context, key string) (bool, error) {
	args := m.Called(ctx, key)
	return args.Bool(0), args.Error(1)
}

func (m *MockSettingsService) SetString(ctx context.Context, key, value string) error {
	args := m.Called(ctx, key, value)
	return args.Error(0)
}

func (m *MockSettingsService) SetInt(ctx context.Context, key string, value int) error {
	args := m.Called(ctx, key, value)
	return args.Error(0)
}

func (m *MockSettingsService) SetBool(ctx context.Context, key string, value bool) error {
	args := m.Called(ctx, key, value)
	return args.Error(0)
}

// Verify mock implements interface
var _ SettingsServiceInterface = (*MockSettingsService)(nil)

func setupSettingsTestRouter(handler *SettingsHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	apiV1 := router.Group("/api/v1")
	handler.RegisterRoutes(apiV1)
	return router
}

func TestSettingsHandler_List(t *testing.T) {
	tests := []struct {
		name           string
		setupMock      func(*MockSettingsService)
		expectedStatus int
		expectedCount  int
	}{
		{
			name: "success - returns settings",
			setupMock: func(m *MockSettingsService) {
				m.On("GetAll", mock.Anything).Return(
					[]models.Setting{
						{Key: "app.name", Value: "Vido", Type: "string"},
						{Key: "app.version", Value: "1.0.0", Type: "string"},
					},
					nil,
				)
			},
			expectedStatus: http.StatusOK,
			expectedCount:  2,
		},
		{
			name: "success - empty list",
			setupMock: func(m *MockSettingsService) {
				m.On("GetAll", mock.Anything).Return(
					[]models.Setting{},
					nil,
				)
			},
			expectedStatus: http.StatusOK,
			expectedCount:  0,
		},
		{
			name: "error - service failure",
			setupMock: func(m *MockSettingsService) {
				m.On("GetAll", mock.Anything).Return(nil, errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockSettingsService)
			tt.setupMock(mockService)

			handler := NewSettingsHandler(mockService)
			router := setupSettingsTestRouter(handler)

			req, _ := http.NewRequest(http.MethodGet, "/api/v1/settings", nil)
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			assert.Equal(t, tt.expectedStatus, resp.Code)

			if tt.expectedStatus == http.StatusOK {
				var response APIResponse
				err := json.Unmarshal(resp.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.True(t, response.Success)

				if tt.expectedCount > 0 {
					dataBytes, _ := json.Marshal(response.Data)
					var settings []models.Setting
					json.Unmarshal(dataBytes, &settings)
					assert.Equal(t, tt.expectedCount, len(settings))
				}
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestSettingsHandler_Get(t *testing.T) {
	tests := []struct {
		name           string
		settingKey     string
		setupMock      func(*MockSettingsService)
		expectedStatus int
	}{
		{
			name:       "success",
			settingKey: "app.name",
			setupMock: func(m *MockSettingsService) {
				m.On("Get", mock.Anything, "app.name").Return(
					&models.Setting{Key: "app.name", Value: "Vido", Type: "string"},
					nil,
				)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:       "not found",
			settingKey: "nonexistent.key",
			setupMock: func(m *MockSettingsService) {
				m.On("Get", mock.Anything, "nonexistent.key").Return(nil, errors.New("not found"))
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockSettingsService)
			tt.setupMock(mockService)

			handler := NewSettingsHandler(mockService)
			router := setupSettingsTestRouter(handler)

			req, _ := http.NewRequest(http.MethodGet, "/api/v1/settings/"+tt.settingKey, nil)
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			assert.Equal(t, tt.expectedStatus, resp.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestSettingsHandler_Set(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		setupMock      func(*MockSettingsService)
		expectedStatus int
	}{
		{
			name: "success - set string",
			requestBody: SetSettingRequest{
				Key:   "app.name",
				Value: "Vido",
				Type:  "string",
			},
			setupMock: func(m *MockSettingsService) {
				m.On("SetString", mock.Anything, "app.name", "Vido").Return(nil)
				m.On("Get", mock.Anything, "app.name").Return(
					&models.Setting{Key: "app.name", Value: "Vido", Type: "string"},
					nil,
				)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "success - set int",
			requestBody: SetSettingRequest{
				Key:   "app.port",
				Value: float64(8080), // JSON numbers come as float64
				Type:  "int",
			},
			setupMock: func(m *MockSettingsService) {
				m.On("SetInt", mock.Anything, "app.port", 8080).Return(nil)
				m.On("Get", mock.Anything, "app.port").Return(
					&models.Setting{Key: "app.port", Value: "8080", Type: "int"},
					nil,
				)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "success - set bool",
			requestBody: SetSettingRequest{
				Key:   "app.debug",
				Value: true,
				Type:  "bool",
			},
			setupMock: func(m *MockSettingsService) {
				m.On("SetBool", mock.Anything, "app.debug", true).Return(nil)
				m.On("Get", mock.Anything, "app.debug").Return(
					&models.Setting{Key: "app.debug", Value: "true", Type: "bool"},
					nil,
				)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "validation error - missing key",
			requestBody: map[string]interface{}{
				"value": "test",
				"type":  "string",
			},
			setupMock:      func(m *MockSettingsService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "validation error - missing value",
			requestBody: map[string]interface{}{
				"key":  "app.name",
				"type": "string",
			},
			setupMock:      func(m *MockSettingsService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "validation error - missing type",
			requestBody: map[string]interface{}{
				"key":   "app.name",
				"value": "test",
			},
			setupMock:      func(m *MockSettingsService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "validation error - invalid type",
			requestBody: map[string]interface{}{
				"key":   "app.name",
				"value": "test",
				"type":  "invalid",
			},
			setupMock:      func(m *MockSettingsService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "validation error - wrong value type for string",
			requestBody: SetSettingRequest{
				Key:   "app.name",
				Value: 123, // Should be string
				Type:  "string",
			},
			setupMock:      func(m *MockSettingsService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "validation error - wrong value type for int",
			requestBody: SetSettingRequest{
				Key:   "app.port",
				Value: "not a number",
				Type:  "int",
			},
			setupMock:      func(m *MockSettingsService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "validation error - wrong value type for bool",
			requestBody: SetSettingRequest{
				Key:   "app.debug",
				Value: "not a bool",
				Type:  "bool",
			},
			setupMock:      func(m *MockSettingsService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "service error",
			requestBody: SetSettingRequest{
				Key:   "app.name",
				Value: "Vido",
				Type:  "string",
			},
			setupMock: func(m *MockSettingsService) {
				m.On("SetString", mock.Anything, "app.name", "Vido").Return(errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockSettingsService)
			tt.setupMock(mockService)

			handler := NewSettingsHandler(mockService)
			router := setupSettingsTestRouter(handler)

			body, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest(http.MethodPost, "/api/v1/settings", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			assert.Equal(t, tt.expectedStatus, resp.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestSettingsHandler_Delete(t *testing.T) {
	tests := []struct {
		name           string
		settingKey     string
		setupMock      func(*MockSettingsService)
		expectedStatus int
	}{
		{
			name:       "success",
			settingKey: "app.name",
			setupMock: func(m *MockSettingsService) {
				m.On("Delete", mock.Anything, "app.name").Return(nil)
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name:       "service error",
			settingKey: "app.name",
			setupMock: func(m *MockSettingsService) {
				m.On("Delete", mock.Anything, "app.name").Return(errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockSettingsService)
			tt.setupMock(mockService)

			handler := NewSettingsHandler(mockService)
			router := setupSettingsTestRouter(handler)

			req, _ := http.NewRequest(http.MethodDelete, "/api/v1/settings/"+tt.settingKey, nil)
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			assert.Equal(t, tt.expectedStatus, resp.Code)
			mockService.AssertExpectations(t)
		})
	}
}
