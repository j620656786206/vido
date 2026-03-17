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

// MockSetupService is a mock implementation of SetupServiceInterface
type MockSetupService struct {
	mock.Mock
}

func (m *MockSetupService) IsFirstRun(ctx context.Context) (bool, error) {
	args := m.Called(ctx)
	return args.Bool(0), args.Error(1)
}

func (m *MockSetupService) CompleteSetup(ctx context.Context, config models.SetupConfig) error {
	args := m.Called(ctx, config)
	return args.Error(0)
}

func (m *MockSetupService) ValidateStep(ctx context.Context, step string, data map[string]interface{}) error {
	args := m.Called(ctx, step, data)
	return args.Error(0)
}

var _ SetupServiceInterface = (*MockSetupService)(nil)

func setupSetupTestRouter(handler *SetupHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	apiV1 := router.Group("/api/v1")
	handler.RegisterRoutes(apiV1)
	return router
}

func TestSetupHandler_GetStatus(t *testing.T) {
	tests := []struct {
		name           string
		setupMock      func(*MockSetupService)
		expectedStatus int
		expectedSetup  bool
	}{
		{
			name: "needs setup - first run",
			setupMock: func(m *MockSetupService) {
				m.On("IsFirstRun", mock.Anything).Return(true, nil)
			},
			expectedStatus: http.StatusOK,
			expectedSetup:  true,
		},
		{
			name: "setup completed",
			setupMock: func(m *MockSetupService) {
				m.On("IsFirstRun", mock.Anything).Return(false, nil)
			},
			expectedStatus: http.StatusOK,
			expectedSetup:  false,
		},
		{
			name: "service error",
			setupMock: func(m *MockSetupService) {
				m.On("IsFirstRun", mock.Anything).Return(false, errors.New("db error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockSetupService)
			tt.setupMock(mockService)

			handler := NewSetupHandler(mockService)
			router := setupSetupTestRouter(handler)

			req, _ := http.NewRequest(http.MethodGet, "/api/v1/setup/status", nil)
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			assert.Equal(t, tt.expectedStatus, resp.Code)

			if tt.expectedStatus == http.StatusOK {
				var response APIResponse
				err := json.Unmarshal(resp.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.True(t, response.Success)

				dataMap, ok := response.Data.(map[string]interface{})
				assert.True(t, ok)
				assert.Equal(t, tt.expectedSetup, dataMap["needsSetup"])
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestSetupHandler_Complete(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		setupMock      func(*MockSetupService)
		expectedStatus int
	}{
		{
			name: "success - full config",
			requestBody: models.SetupConfig{
				Language:        "zh-TW",
				MediaFolderPath: "/media",
			},
			setupMock: func(m *MockSetupService) {
				m.On("CompleteSetup", mock.Anything, mock.AnythingOfType("models.SetupConfig")).Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "error - already completed",
			requestBody: models.SetupConfig{
				Language: "zh-TW",
			},
			setupMock: func(m *MockSetupService) {
				m.On("CompleteSetup", mock.Anything, mock.AnythingOfType("models.SetupConfig")).Return(errors.New("setup already completed"))
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "error - service failure",
			requestBody: models.SetupConfig{
				Language: "zh-TW",
			},
			setupMock: func(m *MockSetupService) {
				m.On("CompleteSetup", mock.Anything, mock.AnythingOfType("models.SetupConfig")).Return(errors.New("db error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "error - invalid JSON",
			requestBody:    "not json",
			setupMock:      func(m *MockSetupService) {},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockSetupService)
			tt.setupMock(mockService)

			handler := NewSetupHandler(mockService)
			router := setupSetupTestRouter(handler)

			body, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest(http.MethodPost, "/api/v1/setup/complete", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			assert.Equal(t, tt.expectedStatus, resp.Code)

			if tt.expectedStatus == http.StatusOK {
				var response APIResponse
				err := json.Unmarshal(resp.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.True(t, response.Success)
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestSetupHandler_ValidateStep(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		setupMock      func(*MockSetupService)
		expectedStatus int
	}{
		{
			name: "success - valid step",
			requestBody: ValidateStepRequest{
				Step: "welcome",
				Data: map[string]interface{}{"language": "zh-TW"},
			},
			setupMock: func(m *MockSetupService) {
				m.On("ValidateStep", mock.Anything, "welcome", mock.AnythingOfType("map[string]interface {}")).Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "error - validation failed",
			requestBody: ValidateStepRequest{
				Step: "welcome",
				Data: map[string]interface{}{},
			},
			setupMock: func(m *MockSetupService) {
				m.On("ValidateStep", mock.Anything, "welcome", mock.AnythingOfType("map[string]interface {}")).Return(errors.New("language is required"))
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "error - missing step field",
			requestBody:    map[string]interface{}{"data": map[string]interface{}{}},
			setupMock:      func(m *MockSetupService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "error - missing data field",
			requestBody:    map[string]interface{}{"step": "welcome"},
			setupMock:      func(m *MockSetupService) {},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockSetupService)
			tt.setupMock(mockService)

			handler := NewSetupHandler(mockService)
			router := setupSetupTestRouter(handler)

			body, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest(http.MethodPost, "/api/v1/setup/validate-step", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			assert.Equal(t, tt.expectedStatus, resp.Code)

			if tt.expectedStatus == http.StatusOK {
				var response APIResponse
				err := json.Unmarshal(resp.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.True(t, response.Success)
			}

			mockService.AssertExpectations(t)
		})
	}
}
