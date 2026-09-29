package services

import (
	"context"
	"errors"
	"os"
	"path/filepath"
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
		errMsg   string
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
			errMsg:   "check setup status",
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
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
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
				ClaudeApiKey:    "sk-ant-123",
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
				// Keys land under the names KeyResolver reads. Literal on purpose: a
				// secret name is persisted data, so renaming one is a migration (dsr-13).
				sec.On("Store", mock.Anything, "tmdb.api_key", "abc123def456").Return(nil)
				sec.On("Store", mock.Anything, "claude.api_key", "sk-ant-123").Return(nil)
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
			svc.SetKeyWriter(NewKeySettingsService(nil, mockSecrets, true))
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
				require.NoError(t, err)
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
			name: "error - save qbittorrent.host fails",
			config: SetupConfig{
				Language: "zh-TW",
				QBTUrl:   "http://localhost:8080",
			},
			setup: func(repo *MockSettingsRepo, sec *MockSecretsService) {
				repo.On("GetBool", mock.Anything, "setup_completed").Return(false, errors.New("setting with key setup_completed not found"))
				repo.On("SetString", mock.Anything, "language", "zh-TW").Return(nil)
				repo.On("SetString", mock.Anything, "qbt_url", "http://localhost:8080").Return(nil)
				repo.On("SetString", mock.Anything, "qbittorrent.host", "http://localhost:8080").Return(errors.New("db error"))
			},
			errMsg: "save qbittorrent.host",
		},
		{
			name: "error - save qbittorrent.username fails",
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
				repo.On("SetString", mock.Anything, "qbt_username", "admin").Return(nil)
				repo.On("SetString", mock.Anything, "qbittorrent.username", "admin").Return(errors.New("db error"))
			},
			errMsg: "save qbittorrent.username",
		},
		{
			name: "error - save qbittorrent.password fails",
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
				sec.On("Store", mock.Anything, "qbt_password", "secret").Return(nil)
				sec.On("Store", mock.Anything, "qbittorrent.password", "secret").Return(errors.New("encryption error"))
			},
			errMsg: "save qbittorrent.password",
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
			name: "error - save tmdb api key fails",
			config: SetupConfig{
				Language:   "zh-TW",
				TMDbApiKey: "key123",
			},
			setup: func(repo *MockSettingsRepo, sec *MockSecretsService) {
				repo.On("GetBool", mock.Anything, "setup_completed").Return(false, errors.New("setting with key setup_completed not found"))
				repo.On("SetString", mock.Anything, "language", "zh-TW").Return(nil)
				sec.On("Store", mock.Anything, "tmdb.api_key", "key123").Return(errors.New("encryption error"))
			},
			errMsg: "save api keys",
		},
		{
			name: "error - save claude api key fails",
			config: SetupConfig{
				Language:     "zh-TW",
				ClaudeApiKey: "sk-ant-key",
			},
			setup: func(repo *MockSettingsRepo, sec *MockSecretsService) {
				repo.On("GetBool", mock.Anything, "setup_completed").Return(false, errors.New("setting with key setup_completed not found"))
				repo.On("SetString", mock.Anything, "language", "zh-TW").Return(nil)
				sec.On("Store", mock.Anything, "claude.api_key", "sk-ant-key").Return(errors.New("encryption error"))
			},
			errMsg: "save api keys",
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
				svc.SetKeyWriter(NewKeySettingsService(nil, mockSecrets, true))
			}

			err := svc.CompleteSetup(context.Background(), tt.config)

			if tt.errMsg != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
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
			errMsg:  "請選擇語言",
		},
		{
			name: "welcome - non-string language type",
			step: "welcome",
			data: map[string]interface{}{
				"language": 123,
			},
			wantErr: true,
			errMsg:  "請選擇語言",
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
			errMsg:  "是檔案，不是資料夾",
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
			errMsg:  "TMDb 金鑰格式不對",
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
			errMsg:  "qBittorrent 網址看起來不對",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockSettingsRepo)
			svc := NewSetupService(mockRepo, nil)
			svc.SetKeyWriter(NewKeySettingsService(nil, new(MockSecretsService), true))

			err := svc.ValidateStep(context.Background(), tt.step, tt.data)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
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
			errMsg:  "請選擇語言",
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
			errMsg:  "qBittorrent 網址看起來不對",
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
			errMsg:  "還有資料夾沒填路徑",
		},
		{
			name: "media-folder - nonexistent path",
			step: "media-folder",
			data: map[string]interface{}{
				"media_folder_path": "/nonexistent/path/xyz",
			},
			wantErr: true,
			errMsg:  "找不到「/nonexistent/path/xyz」",
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
			errMsg:  "TMDb 金鑰格式不對",
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
			svc.SetKeyWriter(NewKeySettingsService(nil, new(MockSecretsService), true))

			err := svc.ValidateStep(context.Background(), tt.step, tt.data)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestSetupService_KeysNotWritable: without an ENCRYPTION_KEY the settings page
// refuses to store API keys (KeySettingsService.Save → ErrKeysNotWritable). The
// wizard shares that path, so it must refuse too — and before writing anything,
// or a retry would create the libraries twice (dsr-13).
func TestSetupService_KeysNotWritable(t *testing.T) {
	notFound := errors.New("setting with key setup_completed not found")
	notWritable := func(sec *MockSecretsService) SetupKeyWriter {
		return NewKeySettingsService(nil, sec, false)
	}

	t.Run("validate api-keys refuses a key it cannot store", func(t *testing.T) {
		svc := NewSetupService(new(MockSettingsRepo), nil)
		svc.SetKeyWriter(notWritable(new(MockSecretsService)))

		err := svc.ValidateStep(context.Background(), "api-keys", map[string]interface{}{
			"claude_api_key": "sk-ant-1",
		})

		assert.ErrorIs(t, err, ErrKeysNotWritable)
	})

	t.Run("validate api-keys still lets the user go on with no keys", func(t *testing.T) {
		svc := NewSetupService(new(MockSettingsRepo), nil)
		svc.SetKeyWriter(notWritable(new(MockSecretsService)))

		require.NoError(t, svc.ValidateStep(context.Background(), "api-keys", map[string]interface{}{
			"tmdb_api_key":   "",
			"claude_api_key": "   ",
		}))
	})

	t.Run("complete refuses before writing anything", func(t *testing.T) {
		repo := new(MockSettingsRepo)
		sec := new(MockSecretsService)
		repo.On("GetBool", mock.Anything, "setup_completed").Return(false, notFound)
		svc := NewSetupService(repo, sec)
		svc.SetKeyWriter(notWritable(sec))

		err := svc.CompleteSetup(context.Background(), SetupConfig{
			Language:        "zh-TW",
			MediaFolderPath: "/media",
			TMDbApiKey:      "abcdef1234567890",
		})

		assert.ErrorIs(t, err, ErrKeysNotWritable)
		repo.AssertNotCalled(t, "SetString", mock.Anything, mock.Anything, mock.Anything)
		repo.AssertNotCalled(t, "SetBool", mock.Anything, mock.Anything, mock.Anything)
		sec.AssertNotCalled(t, "Store", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("complete with no key writer wired refuses keys instead of dropping them", func(t *testing.T) {
		repo := new(MockSettingsRepo)
		repo.On("GetBool", mock.Anything, "setup_completed").Return(false, notFound)
		svc := NewSetupService(repo, nil)

		err := svc.CompleteSetup(context.Background(), SetupConfig{Language: "zh-TW", ClaudeApiKey: "sk-ant-1"})

		assert.ErrorIs(t, err, ErrKeysNotWritable)
	})

	t.Run("complete without keys is unaffected", func(t *testing.T) {
		repo := new(MockSettingsRepo)
		sec := new(MockSecretsService)
		repo.On("GetBool", mock.Anything, "setup_completed").Return(false, notFound)
		repo.On("SetString", mock.Anything, "language", "en").Return(nil)
		repo.On("SetBool", mock.Anything, "setup_completed", true).Return(nil)
		svc := NewSetupService(repo, sec)
		svc.SetKeyWriter(notWritable(sec))

		require.NoError(t, svc.CompleteSetup(context.Background(), SetupConfig{Language: "en", ClaudeApiKey: "   "}))
		repo.AssertExpectations(t)
	})
}

// disc-setup-wizard-container-path-hint AC #1: a path the container cannot see
// is answered with the folders it CAN see — the media root, then its real
// subfolders sorted, at most four, hidden/Synology system folders left out.
func TestSetupService_MediaFolderNotFound_SuggestsContainerFolders(t *testing.T) {
	root := t.TempDir()
	for _, d := range []string{"tv", "movies", "anime", "docs", "kids", ".hidden", "@eaDir"} {
		require.NoError(t, os.Mkdir(filepath.Join(root, d), 0o755))
	}
	require.NoError(t, os.WriteFile(filepath.Join(root, "readme.txt"), []byte("x"), 0o644))
	svc := NewSetupService(nil, nil)
	svc.SetMediaRoots([]string{root})

	err := svc.ValidateStep(context.Background(), "media-folder", map[string]interface{}{
		"libraries": []interface{}{map[string]interface{}{"path": "/video/Movies", "content_type": "movie"}},
	})
	require.Error(t, err)
	want := "找不到「/video/Movies」。Vido 在 Docker 裡，只看得到掛進容器的資料夾——請填容器裡的路徑，例如 " +
		root + "、" + filepath.Join(root, "anime") + "、" + filepath.Join(root, "docs") + "、" +
		filepath.Join(root, "kids") + "。"
	assert.Equal(t, want, err.Error())
}

func TestSetupService_MediaFolderNotFound_RootWithoutSubfolders(t *testing.T) {
	root := t.TempDir()
	svc := NewSetupService(nil, nil)
	svc.SetMediaRoots([]string{root})

	err := svc.ValidateStep(context.Background(), "media-folder", map[string]interface{}{
		"libraries": []interface{}{map[string]interface{}{"path": "/video/Movies"}},
	})
	require.Error(t, err)
	assert.Equal(t, "找不到「/video/Movies」。Vido 在 Docker 裡，只看得到掛進容器的資料夾——請填容器裡的路徑，例如 "+root+"。", err.Error())
}

// VIDO_MEDIA_DIRS can list the library folders themselves — those come first.
func TestSetupService_MediaFolderNotFound_SeveralRootsComeFirst(t *testing.T) {
	base := t.TempDir()
	movies, tv := filepath.Join(base, "movies"), filepath.Join(base, "tv")
	require.NoError(t, os.MkdirAll(filepath.Join(movies, "Dune (2021)"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(tv, "Arcane"), 0o755))
	svc := NewSetupService(nil, nil)
	svc.SetMediaRoots([]string{movies, tv})

	err := svc.ValidateStep(context.Background(), "media-folder", map[string]interface{}{
		"libraries": []interface{}{map[string]interface{}{"path": "/video/Movies"}},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "例如 "+movies+"、"+tv+"、")
}

// A root nested in another root is listed once, and trailing slashes don't
// make a second spelling.
func TestSetupService_MediaFolderNotFound_NoDuplicateSuggestions(t *testing.T) {
	root := t.TempDir()
	movies := filepath.Join(root, "movies")
	require.NoError(t, os.Mkdir(movies, 0o755))
	svc := NewSetupService(nil, nil)
	svc.SetMediaRoots([]string{root + "/", movies, root})

	err := svc.ValidateStep(context.Background(), "media-folder", map[string]interface{}{
		"libraries": []interface{}{map[string]interface{}{"path": "/video/Movies"}},
	})
	require.Error(t, err)
	assert.Equal(t, "找不到「/video/Movies」。Vido 在 Docker 裡，只看得到掛進容器的資料夾——請填容器裡的路徑，例如 "+
		root+"、"+movies+"。", err.Error())
}

func TestSetupService_MediaFolderUnreadableIsAPermissionProblem(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root reads everything")
	}
	parent := filepath.Join(t.TempDir(), "locked")
	require.NoError(t, os.Mkdir(parent, 0o755))
	target := filepath.Join(parent, "movies")
	require.NoError(t, os.Mkdir(target, 0o755))
	require.NoError(t, os.Chmod(parent, 0o000))
	t.Cleanup(func() { _ = os.Chmod(parent, 0o755) })

	svc := NewSetupService(nil, nil)
	err := svc.ValidateStep(context.Background(), "media-folder", map[string]interface{}{
		"libraries": []interface{}{map[string]interface{}{"path": target}},
	})
	require.Error(t, err)
	assert.Equal(t, "沒有權限讀取「"+target+"」：Vido 所在容器的使用者（PUID／PGID）讀不到這個資料夾。", err.Error())
}

func TestSetupService_MediaFolderNotFound_NothingMounted(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "media")
	svc := NewSetupService(nil, nil)
	svc.SetMediaRoots([]string{missing})

	err := svc.ValidateStep(context.Background(), "media-folder", map[string]interface{}{
		"libraries": []interface{}{map[string]interface{}{"path": "/video/Movies"}},
	})
	require.Error(t, err)
	assert.Equal(t, "找不到「/video/Movies」。Vido 在 Docker 裡，只看得到掛進容器的資料夾，但目前容器裡沒有 "+missing+
		"——請在 docker-compose 的 volumes 把媒體資料夾掛到 "+missing+"。", err.Error())
}

func TestSetupService_MediaFolderMessagesAreChinese(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "a.mkv")
	require.NoError(t, os.WriteFile(file, []byte("x"), 0o644))
	svc := NewSetupService(nil, nil)
	svc.SetMediaRoots([]string{root})
	cases := []struct {
		lib  map[string]interface{}
		want string
	}{
		{map[string]interface{}{"path": ""}, "還有資料夾沒填路徑。"},
		{map[string]interface{}{"path": file}, "「" + file + "」是檔案，不是資料夾。"},
		{map[string]interface{}{"path": root, "content_type": "music"}, "資料夾的類型只能是電影或影集。"},
	}
	for _, tc := range cases {
		err := svc.ValidateStep(context.Background(), "media-folder", map[string]interface{}{
			"libraries": []interface{}{tc.lib},
		})
		require.Error(t, err)
		assert.Equal(t, tc.want, err.Error())
	}
}
