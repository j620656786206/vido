package services

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/models"
)

// Alias for cleaner test code
type SetupConfig = models.SetupConfig

// MockSettingsRepo is a mock implementation of SettingsRepositoryInterface for setup tests
type MockSettingsRepo struct {
	mock.Mock
}

func (m *MockSettingsRepo) Set(ctx context.Context, setting *models.Setting) error {
	args := m.Called(ctx, setting)
	return args.Error(0)
}

func (m *MockSettingsRepo) Get(ctx context.Context, key string) (*models.Setting, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Setting), args.Error(1)
}

func (m *MockSettingsRepo) GetAll(ctx context.Context) ([]models.Setting, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Setting), args.Error(1)
}

func (m *MockSettingsRepo) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *MockSettingsRepo) GetString(ctx context.Context, key string) (string, error) {
	args := m.Called(ctx, key)
	return args.String(0), args.Error(1)
}

func (m *MockSettingsRepo) GetInt(ctx context.Context, key string) (int, error) {
	args := m.Called(ctx, key)
	return args.Int(0), args.Error(1)
}

func (m *MockSettingsRepo) GetBool(ctx context.Context, key string) (bool, error) {
	args := m.Called(ctx, key)
	return args.Bool(0), args.Error(1)
}

func (m *MockSettingsRepo) SetString(ctx context.Context, key, value string) error {
	args := m.Called(ctx, key, value)
	return args.Error(0)
}

func (m *MockSettingsRepo) SetInt(ctx context.Context, key string, value int) error {
	args := m.Called(ctx, key, value)
	return args.Error(0)
}

func (m *MockSettingsRepo) SetBool(ctx context.Context, key string, value bool) error {
	args := m.Called(ctx, key, value)
	return args.Error(0)
}

// MockSecretsService is a mock for secrets.SecretsServiceInterface
type MockSecretsService struct {
	mock.Mock
}

func (m *MockSecretsService) Store(ctx context.Context, name, value string) error {
	args := m.Called(ctx, name, value)
	return args.Error(0)
}

func (m *MockSecretsService) Retrieve(ctx context.Context, name string) (string, error) {
	args := m.Called(ctx, name)
	return args.String(0), args.Error(1)
}

func (m *MockSecretsService) Delete(ctx context.Context, name string) error {
	args := m.Called(ctx, name)
	return args.Error(0)
}

func (m *MockSecretsService) Exists(ctx context.Context, name string) (bool, error) {
	args := m.Called(ctx, name)
	return args.Bool(0), args.Error(1)
}

func (m *MockSecretsService) List(ctx context.Context) ([]string, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

// TestSetupService_IsFirstRun tests the IsFirstRun method
func TestSetupService_IsFirstRun(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*MockSettingsRepo)
		expected bool
		wantErr  bool
	}{
		{
			name: "first run - key not found",
			setup: func(m *MockSettingsRepo) {
				m.On("GetBool", mock.Anything, "setup_completed").Return(false, errors.New("setting with key setup_completed not found"))
			},
			expected: true,
			wantErr:  false,
		},
		{
			name: "first run - flag is false",
			setup: func(m *MockSettingsRepo) {
				m.On("GetBool", mock.Anything, "setup_completed").Return(false, nil)
			},
			expected: true,
			wantErr:  false,
		},
		{
			name: "not first run - setup completed",
			setup: func(m *MockSettingsRepo) {
				m.On("GetBool", mock.Anything, "setup_completed").Return(true, nil)
			},
			expected: false,
			wantErr:  false,
		},
		{
			name: "error - database connection failure propagated",
			setup: func(m *MockSettingsRepo) {
				m.On("GetBool", mock.Anything, "setup_completed").Return(false, errors.New("database connection refused"))
			},
			expected: false,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockSettingsRepo)
			tt.setup(mockRepo)

			svc := NewSetupService(mockRepo, nil)
			result, err := svc.IsFirstRun(context.Background())

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

// TestSetupService_CompleteSetup tests the CompleteSetup method
func TestSetupService_CompleteSetup(t *testing.T) {
	tests := []struct {
		name        string
		config      SetupConfig
		setup       func(*MockSettingsRepo, *MockSecretsService)
		wantErr     bool
		errMsg      string
		sentinelErr error
	}{
		{
			name: "success - full config",
			config: SetupConfig{
				Language:        "zh-TW",
				QBTUrl:          "http://localhost:8080",
				QBTUsername:     "admin",
				QBTPassword:     "secret",
				MediaFolderPath: "/media/videos",
				TMDbApiKey:      "abc123def456",
				AIProvider:      "gemini",
				AIApiKey:        "ai-key-123",
			},
			setup: func(repo *MockSettingsRepo, sec *MockSecretsService) {
				// IsFirstRun check
				repo.On("GetBool", mock.Anything, "setup_completed").Return(false, errors.New("setting with key setup_completed not found"))
				// Save settings
				repo.On("SetString", mock.Anything, "language", "zh-TW").Return(nil)
				repo.On("SetString", mock.Anything, "qbt_url", "http://localhost:8080").Return(nil)
				repo.On("SetString", mock.Anything, "qbittorrent.host", "http://localhost:8080").Return(nil)
				repo.On("SetString", mock.Anything, "qbt_username", "admin").Return(nil)
				repo.On("SetString", mock.Anything, "qbittorrent.username", "admin").Return(nil)
				sec.On("Store", mock.Anything, "qbt_password", "secret").Return(nil)
				sec.On("Store", mock.Anything, "qbittorrent.password", "secret").Return(nil)
				repo.On("SetString", mock.Anything, "media_folder_path", "/media/videos").Return(nil)
				sec.On("Store", mock.Anything, "tmdb_api_key", "abc123def456").Return(nil)
				repo.On("SetString", mock.Anything, "ai_provider", "gemini").Return(nil)
				sec.On("Store", mock.Anything, "ai_api_key", "ai-key-123").Return(nil)
				repo.On("SetBool", mock.Anything, "setup_completed", true).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "success - minimal config (only required fields)",
			config: SetupConfig{
				Language:        "en",
				MediaFolderPath: "/media",
			},
			setup: func(repo *MockSettingsRepo, sec *MockSecretsService) {
				repo.On("GetBool", mock.Anything, "setup_completed").Return(false, errors.New("setting with key setup_completed not found"))
				repo.On("SetString", mock.Anything, "language", "en").Return(nil)
				repo.On("SetString", mock.Anything, "media_folder_path", "/media").Return(nil)
				repo.On("SetBool", mock.Anything, "setup_completed", true).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "error - setup already completed",
			config: SetupConfig{
				Language: "zh-TW",
			},
			setup: func(repo *MockSettingsRepo, sec *MockSecretsService) {
				repo.On("GetBool", mock.Anything, "setup_completed").Return(true, nil)
			},
			wantErr:     true,
			errMsg:      "setup already completed",
			sentinelErr: ErrSetupAlreadyCompleted,
		},
		{
			name: "error - save language fails",
			config: SetupConfig{
				Language: "zh-TW",
			},
			setup: func(repo *MockSettingsRepo, sec *MockSecretsService) {
				repo.On("GetBool", mock.Anything, "setup_completed").Return(false, errors.New("setting with key setup_completed not found"))
				repo.On("SetString", mock.Anything, "language", "zh-TW").Return(errors.New("db error"))
			},
			wantErr: true,
			errMsg:  "save language",
		},
		{
			name: "error - mark completed fails",
			config: SetupConfig{
				Language:        "zh-TW",
				MediaFolderPath: "/media",
			},
			setup: func(repo *MockSettingsRepo, sec *MockSecretsService) {
				repo.On("GetBool", mock.Anything, "setup_completed").Return(false, errors.New("setting with key setup_completed not found"))
				repo.On("SetString", mock.Anything, "language", "zh-TW").Return(nil)
				repo.On("SetString", mock.Anything, "media_folder_path", "/media").Return(nil)
				repo.On("SetBool", mock.Anything, "setup_completed", true).Return(errors.New("db error"))
			},
			wantErr: true,
			errMsg:  "mark setup completed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockSettingsRepo)
			mockSecrets := new(MockSecretsService)
			tt.setup(mockRepo, mockSecrets)

			svc := NewSetupService(mockRepo, mockSecrets)
			err := svc.CompleteSetup(context.Background(), tt.config)

			if tt.wantErr {
				require.Error(t, err)
				if tt.sentinelErr != nil {
					assert.ErrorIs(t, err, tt.sentinelErr)
				}
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
			mockRepo.AssertExpectations(t)
			mockSecrets.AssertExpectations(t)
		})
	}
}

// TestSetupService_CompleteSetup_PartialFailures tests individual field save failures
func TestSetupService_CompleteSetup_PartialFailures(t *testing.T) {
	tests := []struct {
		name          string
		config        SetupConfig
		setup         func(*MockSettingsRepo, *MockSecretsService)
		errMsg        string
		useNilSecrets bool
	}{
		{
			name: "error - save qbt_url fails",
			config: SetupConfig{
				Language: "zh-TW",
				QBTUrl:   "http://localhost:8080",
			},
			setup: func(repo *MockSettingsRepo, sec *MockSecretsService) {
				repo.On("GetBool", mock.Anything, "setup_completed").Return(false, errors.New("setting with key setup_completed not found"))
				repo.On("SetString", mock.Anything, "language", "zh-TW").Return(nil)
				repo.On("SetString", mock.Anything, "qbt_url", "http://localhost:8080").Return(errors.New("db error"))
			},
			errMsg: "save qbt_url",
		},
		{
			name: "error - save qbt_username fails",
			config: SetupConfig{
				Language:    "zh-TW",
				QBTUrl:      "http://localhost:8080",
				QBTUsername: "admin",
			},
			setup: func(repo *MockSettingsRepo, sec *MockSecretsService) {
				repo.On("GetBool", mock.Anything, "setup_completed").Return(false, errors.New("setting with key setup_completed not found"))
				repo.On("SetString", mock.Anything, "language", "zh-TW").Return(nil)
				repo.On("SetString", mock.Anything, "qbt_url", "http://localhost:8080").Return(nil)
				repo.On("SetString", mock.Anything, "qbittorrent.host", "http://localhost:8080").Return(nil)
				repo.On("SetString", mock.Anything, "qbt_username", "admin").Return(errors.New("db error"))
			},
			errMsg: "save qbt_username",
		},
		{
			name: "error - save qbt_password fails",
			config: SetupConfig{
				Language:    "zh-TW",
				QBTUrl:      "http://localhost:8080",
				QBTPassword: "secret",
			},
			setup: func(repo *MockSettingsRepo, sec *MockSecretsService) {
				repo.On("GetBool", mock.Anything, "setup_completed").Return(false, errors.New("setting with key setup_completed not found"))
				repo.On("SetString", mock.Anything, "language", "zh-TW").Return(nil)
				repo.On("SetString", mock.Anything, "qbt_url", "http://localhost:8080").Return(nil)
				repo.On("SetString", mock.Anything, "qbittorrent.host", "http://localhost:8080").Return(nil)
				sec.On("Store", mock.Anything, "qbt_password", "secret").Return(errors.New("encryption error"))
			},
			errMsg: "save qbt_password",
		},
		{
			name: "error - save media_folder_path fails",
			config: SetupConfig{
				Language:        "zh-TW",
				MediaFolderPath: "/media",
			},
			setup: func(repo *MockSettingsRepo, sec *MockSecretsService) {
				repo.On("GetBool", mock.Anything, "setup_completed").Return(false, errors.New("setting with key setup_completed not found"))
				repo.On("SetString", mock.Anything, "language", "zh-TW").Return(nil)
				repo.On("SetString", mock.Anything, "media_folder_path", "/media").Return(errors.New("db error"))
			},
			errMsg: "save media_folder_path",
		},
		{
			name: "error - save tmdb_api_key fails",
			config: SetupConfig{
				Language:   "zh-TW",
				TMDbApiKey: "key123",
			},
			setup: func(repo *MockSettingsRepo, sec *MockSecretsService) {
				repo.On("GetBool", mock.Anything, "setup_completed").Return(false, errors.New("setting with key setup_completed not found"))
				repo.On("SetString", mock.Anything, "language", "zh-TW").Return(nil)
				sec.On("Store", mock.Anything, "tmdb_api_key", "key123").Return(errors.New("encryption error"))
			},
			errMsg: "save tmdb_api_key",
		},
		{
			name: "error - save ai_provider fails",
			config: SetupConfig{
				Language:   "zh-TW",
				AIProvider: "gemini",
			},
			setup: func(repo *MockSettingsRepo, sec *MockSecretsService) {
				repo.On("GetBool", mock.Anything, "setup_completed").Return(false, errors.New("setting with key setup_completed not found"))
				repo.On("SetString", mock.Anything, "language", "zh-TW").Return(nil)
				repo.On("SetString", mock.Anything, "ai_provider", "gemini").Return(errors.New("db error"))
			},
			errMsg: "save ai_provider",
		},
		{
			name: "error - save ai_api_key fails",
			config: SetupConfig{
				Language:   "zh-TW",
				AIProvider: "gemini",
				AIApiKey:   "ai-key",
			},
			setup: func(repo *MockSettingsRepo, sec *MockSecretsService) {
				repo.On("GetBool", mock.Anything, "setup_completed").Return(false, errors.New("setting with key setup_completed not found"))
				repo.On("SetString", mock.Anything, "language", "zh-TW").Return(nil)
				repo.On("SetString", mock.Anything, "ai_provider", "gemini").Return(nil)
				sec.On("Store", mock.Anything, "ai_api_key", "ai-key").Return(errors.New("encryption error"))
			},
			errMsg: "save ai_api_key",
		},
		{
			name: "success - qbt password skipped when no secrets service",
			config: SetupConfig{
				Language:    "zh-TW",
				QBTUrl:      "http://localhost:8080",
				QBTPassword: "secret",
			},
			setup: func(repo *MockSettingsRepo, sec *MockSecretsService) {
				repo.On("GetBool", mock.Anything, "setup_completed").Return(false, errors.New("setting with key setup_completed not found"))
				repo.On("SetString", mock.Anything, "language", "zh-TW").Return(nil)
				repo.On("SetString", mock.Anything, "qbt_url", "http://localhost:8080").Return(nil)
				repo.On("SetString", mock.Anything, "qbittorrent.host", "http://localhost:8080").Return(nil)
				// No secrets call expected — nil secrets service
				repo.On("SetBool", mock.Anything, "setup_completed", true).Return(nil)
			},
			errMsg:        "", // no error — password silently skipped when no secrets service
			useNilSecrets: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockSettingsRepo)
			mockSecrets := new(MockSecretsService)
			tt.setup(mockRepo, mockSecrets)

			var svc *SetupService
			if tt.useNilSecrets {
				svc = NewSetupService(mockRepo, nil)
			} else {
				svc = NewSetupService(mockRepo, mockSecrets)
			}

			err := svc.CompleteSetup(context.Background(), tt.config)

			if tt.errMsg != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
			mockRepo.AssertExpectations(t)
			if !tt.useNilSecrets {
				mockSecrets.AssertExpectations(t)
			}
		})
	}
}

// TestSetupService_ValidateStep_EdgeCases tests edge cases for validation
func TestSetupService_ValidateStep_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		step    string
		data    map[string]interface{}
		wantErr bool
		errMsg  string
	}{
		{
			name: "welcome - empty string language",
			step: "welcome",
			data: map[string]interface{}{
				"language": "",
			},
			wantErr: true,
			errMsg:  "language is required",
		},
		{
			name: "welcome - non-string language type",
			step: "welcome",
			data: map[string]interface{}{
				"language": 123,
			},
			wantErr: true,
			errMsg:  "language is required",
		},
		{
			name: "media-folder - path is a file not directory",
			step: "media-folder",
			data: func() map[string]interface{} {
				// Create a temp file (not directory)
				f, err := os.CreateTemp("", "testfile")
				if err != nil {
					t.Fatalf("failed to create temp file: %v", err)
				}
				f.Close()
				t.Cleanup(func() { os.Remove(f.Name()) })
				return map[string]interface{}{
					"media_folder_path": f.Name(),
				}
			}(),
			wantErr: true,
			errMsg:  "not a directory",
		},
		{
			name: "api-keys - TMDb key exactly 16 chars (valid)",
			step: "api-keys",
			data: map[string]interface{}{
				"tmdb_api_key": "1234567890123456",
			},
			wantErr: false,
		},
		{
			name: "api-keys - TMDb key 15 chars (invalid)",
			step: "api-keys",
			data: map[string]interface{}{
				"tmdb_api_key": "123456789012345",
			},
			wantErr: true,
			errMsg:  "invalid TMDb API key format",
		},
		{
			name: "qbittorrent - URL exactly 7 chars passes length check",
			step: "qbittorrent",
			data: map[string]interface{}{
				"qbt_url": "http://",
			},
			wantErr: false,
		},
		{
			name: "qbittorrent - URL 6 chars (invalid)",
			step: "qbittorrent",
			data: map[string]interface{}{
				"qbt_url": "ftp://",
			},
			wantErr: true,
			errMsg:  "invalid qBittorrent URL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockSettingsRepo)
			svc := NewSetupService(mockRepo, nil)

			err := svc.ValidateStep(context.Background(), tt.step, tt.data)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestSetupService_ValidateStep tests the ValidateStep method
func TestSetupService_ValidateStep(t *testing.T) {
	tests := []struct {
		name    string
		step    string
		data    map[string]interface{}
		wantErr bool
		errMsg  string
	}{
		{
			name: "welcome - valid language",
			step: "welcome",
			data: map[string]interface{}{
				"language": "zh-TW",
			},
			wantErr: false,
		},
		{
			name:    "welcome - missing language",
			step:    "welcome",
			data:    map[string]interface{}{},
			wantErr: true,
			errMsg:  "language is required",
		},
		{
			name: "qbittorrent - valid URL",
			step: "qbittorrent",
			data: map[string]interface{}{
				"qbt_url": "http://localhost:8080",
			},
			wantErr: false,
		},
		{
			name:    "qbittorrent - skip (empty URL)",
			step:    "qbittorrent",
			data:    map[string]interface{}{},
			wantErr: false,
		},
		{
			name: "qbittorrent - invalid URL",
			step: "qbittorrent",
			data: map[string]interface{}{
				"qbt_url": "bad",
			},
			wantErr: true,
			errMsg:  "invalid qBittorrent URL",
		},
		{
			name: "media-folder - valid path",
			step: "media-folder",
			data: map[string]interface{}{
				"media_folder_path": os.TempDir(),
			},
			wantErr: false,
		},
		{
			name:    "media-folder - missing path",
			step:    "media-folder",
			data:    map[string]interface{}{},
			wantErr: true,
			errMsg:  "media folder path is required",
		},
		{
			name: "media-folder - nonexistent path",
			step: "media-folder",
			data: map[string]interface{}{
				"media_folder_path": "/nonexistent/path/xyz",
			},
			wantErr: true,
			errMsg:  "does not exist",
		},
		{
			name: "api-keys - valid TMDb key",
			step: "api-keys",
			data: map[string]interface{}{
				"tmdb_api_key": "abcdef1234567890abcdef1234567890",
			},
			wantErr: false,
		},
		{
			name:    "api-keys - skip (empty)",
			step:    "api-keys",
			data:    map[string]interface{}{},
			wantErr: false,
		},
		{
			name: "api-keys - invalid TMDb key (too short)",
			step: "api-keys",
			data: map[string]interface{}{
				"tmdb_api_key": "short",
			},
			wantErr: true,
			errMsg:  "invalid TMDb API key format",
		},
		{
			name:    "complete - always valid",
			step:    "complete",
			data:    map[string]interface{}{},
			wantErr: false,
		},
		{
			name:    "unknown step",
			step:    "nonexistent",
			data:    map[string]interface{}{},
			wantErr: true,
			errMsg:  "unknown step",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockSettingsRepo)
			svc := NewSetupService(mockRepo, nil)

			err := svc.ValidateStep(context.Background(), tt.step, tt.data)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
