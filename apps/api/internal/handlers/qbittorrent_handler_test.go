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
	"github.com/vido/api/internal/qbittorrent"
)

// --- Mock QBittorrent Service ---

type MockQBittorrentService struct {
	mock.Mock
}

func (m *MockQBittorrentService) GetConfig(ctx context.Context) (*qbittorrent.Config, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*qbittorrent.Config), args.Error(1)
}

func (m *MockQBittorrentService) SaveConfig(ctx context.Context, config *qbittorrent.Config) error {
	args := m.Called(ctx, config)
	return args.Error(0)
}

func (m *MockQBittorrentService) TestConnection(ctx context.Context) (*qbittorrent.VersionInfo, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*qbittorrent.VersionInfo), args.Error(1)
}

func (m *MockQBittorrentService) TestConnectionWithConfig(ctx context.Context, config *qbittorrent.Config) (*qbittorrent.VersionInfo, error) {
	args := m.Called(ctx, config)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*qbittorrent.VersionInfo), args.Error(1)
}

func (m *MockQBittorrentService) IsConfigured(ctx context.Context) bool {
	args := m.Called(ctx)
	return args.Bool(0)
}

var _ QBittorrentServiceInterface = (*MockQBittorrentService)(nil)

func setupQBRouter(handler *QBittorrentHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler.RegisterRoutes(router.Group("/api/v1"))
	return router
}

func TestQBittorrentHandler_GetConfig(t *testing.T) {
	tests := []struct {
		name           string
		setupMock      func(*MockQBittorrentService)
		expectedStatus int
		checkResponse  func(*testing.T, map[string]interface{})
	}{
		{
			name: "returns config without password",
			setupMock: func(m *MockQBittorrentService) {
				m.On("GetConfig", mock.Anything).Return(&qbittorrent.Config{
					Host:     "http://192.168.1.100:8080",
					Username: "admin",
					Password: "should-not-appear",
					BasePath: "/qbt",
				}, nil)
				m.On("IsConfigured", mock.Anything).Return(true)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, data map[string]interface{}) {
				assert.Equal(t, "http://192.168.1.100:8080", data["host"])
				assert.Equal(t, "admin", data["username"])
				assert.Equal(t, "/qbt", data["base_path"])
				assert.Equal(t, true, data["configured"])
				assert.NotContains(t, data, "password")
			},
		},
		{
			name: "returns empty config when not configured",
			setupMock: func(m *MockQBittorrentService) {
				m.On("GetConfig", mock.Anything).Return(&qbittorrent.Config{}, nil)
				m.On("IsConfigured", mock.Anything).Return(false)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, data map[string]interface{}) {
				assert.Equal(t, "", data["host"])
				assert.Equal(t, false, data["configured"])
			},
		},
		{
			name: "returns 500 when service fails",
			setupMock: func(m *MockQBittorrentService) {
				m.On("GetConfig", mock.Anything).Return(nil, errors.New("service error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockQBittorrentService)
			tt.setupMock(mockService)

			handler := NewQBittorrentHandler(mockService)
			router := setupQBRouter(handler)

			req, _ := http.NewRequest(http.MethodGet, "/api/v1/settings/qbittorrent", nil)
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			assert.Equal(t, tt.expectedStatus, resp.Code)

			if tt.checkResponse != nil {
				var apiResp APIResponse
				err := json.Unmarshal(resp.Body.Bytes(), &apiResp)
				assert.NoError(t, err)
				assert.True(t, apiResp.Success)

				data, ok := apiResp.Data.(map[string]interface{})
				assert.True(t, ok)
				tt.checkResponse(t, data)
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestQBittorrentHandler_SaveConfig(t *testing.T) {
	tests := []struct {
		name           string
		body           interface{}
		setupMock      func(*MockQBittorrentService)
		expectedStatus int
	}{
		{
			name: "saves valid config",
			body: SaveQBConfigRequest{
				Host:     "http://192.168.1.100:8080",
				Username: "admin",
				Password: "secret",
				BasePath: "/qbt",
			},
			setupMock: func(m *MockQBittorrentService) {
				m.On("SaveConfig", mock.Anything, mock.MatchedBy(func(c *qbittorrent.Config) bool {
					return c.Host == "http://192.168.1.100:8080" &&
						c.Username == "admin" &&
						c.Password == "secret" &&
						c.BasePath == "/qbt"
				})).Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "returns 400 for missing host",
			body: map[string]string{
				"username": "admin",
				"password": "secret",
			},
			setupMock:      func(m *MockQBittorrentService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "returns 400 for missing password",
			body: map[string]string{
				"host":     "http://host:8080",
				"username": "admin",
			},
			setupMock:      func(m *MockQBittorrentService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "returns 500 when service fails",
			body: SaveQBConfigRequest{
				Host:     "http://host:8080",
				Username: "admin",
				Password: "secret",
			},
			setupMock: func(m *MockQBittorrentService) {
				m.On("SaveConfig", mock.Anything, mock.Anything).Return(errors.New("save error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockQBittorrentService)
			tt.setupMock(mockService)

			handler := NewQBittorrentHandler(mockService)
			router := setupQBRouter(handler)

			bodyBytes, _ := json.Marshal(tt.body)
			req, _ := http.NewRequest(http.MethodPut, "/api/v1/settings/qbittorrent", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			assert.Equal(t, tt.expectedStatus, resp.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestQBittorrentHandler_TestConnection(t *testing.T) {
	tests := []struct {
		name           string
		setupMock      func(*MockQBittorrentService)
		expectedStatus int
		checkResponse  func(*testing.T, *APIResponse)
	}{
		{
			name: "returns version info on success",
			setupMock: func(m *MockQBittorrentService) {
				m.On("TestConnection", mock.Anything).Return(&qbittorrent.VersionInfo{
					AppVersion: "v4.5.2",
					APIVersion: "2.9.3",
				}, nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp *APIResponse) {
				assert.True(t, resp.Success)
				data := resp.Data.(map[string]interface{})
				assert.Equal(t, "v4.5.2", data["app_version"])
				assert.Equal(t, "2.9.3", data["api_version"])
			},
		},
		{
			name: "returns error on connection failure with specific error code",
			setupMock: func(m *MockQBittorrentService) {
				m.On("TestConnection", mock.Anything).Return(nil, &qbittorrent.ConnectionError{
					Code:    qbittorrent.ErrCodeAuthFailed,
					Message: "authentication failed: invalid credentials",
				})
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, resp *APIResponse) {
				assert.False(t, resp.Success)
				assert.Equal(t, "QB_AUTH_FAILED", resp.Error.Code)
				assert.Equal(t, "無法連線到 qBittorrent", resp.Error.Message)
			},
		},
		{
			name: "returns error when not configured with specific code",
			setupMock: func(m *MockQBittorrentService) {
				m.On("TestConnection", mock.Anything).Return(nil, &qbittorrent.ConnectionError{
					Code:    qbittorrent.ErrCodeNotConfigured,
					Message: "qBittorrent not configured",
				})
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, resp *APIResponse) {
				assert.False(t, resp.Success)
				assert.NotNil(t, resp.Error)
				assert.Equal(t, "QB_NOT_CONFIGURED", resp.Error.Code)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockQBittorrentService)
			tt.setupMock(mockService)

			handler := NewQBittorrentHandler(mockService)
			router := setupQBRouter(handler)

			req, _ := http.NewRequest(http.MethodPost, "/api/v1/settings/qbittorrent/test", nil)
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			assert.Equal(t, tt.expectedStatus, resp.Code)

			if tt.checkResponse != nil {
				var apiResp APIResponse
				err := json.Unmarshal(resp.Body.Bytes(), &apiResp)
				assert.NoError(t, err)
				tt.checkResponse(t, &apiResp)
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestQBittorrentHandler_TestConnection_WithBody(t *testing.T) {
	mockService := new(MockQBittorrentService)
	mockService.On("TestConnectionWithConfig", mock.Anything, mock.MatchedBy(func(c *qbittorrent.Config) bool {
		return c.Host == "http://192.168.1.100:8080" &&
			c.Username == "admin" &&
			c.Password == "secret" &&
			c.BasePath == "/qbt"
	})).Return(&qbittorrent.VersionInfo{
		AppVersion: "v4.5.2",
		APIVersion: "2.9.3",
	}, nil)

	handler := NewQBittorrentHandler(mockService)
	router := setupQBRouter(handler)

	body := TestQBConnectionRequest{
		Host:     "http://192.168.1.100:8080",
		Username: "admin",
		Password: "secret",
		BasePath: "/qbt",
	}
	bodyBytes, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/settings/qbittorrent/test", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)

	var apiResp APIResponse
	err := json.Unmarshal(resp.Body.Bytes(), &apiResp)
	assert.NoError(t, err)
	assert.True(t, apiResp.Success)

	data := apiResp.Data.(map[string]interface{})
	assert.Equal(t, "v4.5.2", data["app_version"])

	mockService.AssertExpectations(t)
}
