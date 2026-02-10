package services

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/qbittorrent"
	"github.com/vido/api/internal/repository"
)

// --- Mock Settings Repository (for qBittorrent service tests) ---

type MockQBSettingsRepo struct {
	mock.Mock
}

func (m *MockQBSettingsRepo) Set(ctx context.Context, setting *models.Setting) error {
	args := m.Called(ctx, setting)
	return args.Error(0)
}

func (m *MockQBSettingsRepo) Get(ctx context.Context, key string) (*models.Setting, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Setting), args.Error(1)
}

func (m *MockQBSettingsRepo) GetAll(ctx context.Context) ([]models.Setting, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Setting), args.Error(1)
}

func (m *MockQBSettingsRepo) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *MockQBSettingsRepo) GetString(ctx context.Context, key string) (string, error) {
	args := m.Called(ctx, key)
	return args.String(0), args.Error(1)
}

func (m *MockQBSettingsRepo) GetInt(ctx context.Context, key string) (int, error) {
	args := m.Called(ctx, key)
	return args.Int(0), args.Error(1)
}

func (m *MockQBSettingsRepo) GetBool(ctx context.Context, key string) (bool, error) {
	args := m.Called(ctx, key)
	return args.Bool(0), args.Error(1)
}

func (m *MockQBSettingsRepo) SetString(ctx context.Context, key, value string) error {
	args := m.Called(ctx, key, value)
	return args.Error(0)
}

func (m *MockQBSettingsRepo) SetInt(ctx context.Context, key string, value int) error {
	args := m.Called(ctx, key, value)
	return args.Error(0)
}

func (m *MockQBSettingsRepo) SetBool(ctx context.Context, key string, value bool) error {
	args := m.Called(ctx, key, value)
	return args.Error(0)
}

var _ repository.SettingsRepositoryInterface = (*MockQBSettingsRepo)(nil)

// --- Mock Secrets Service ---

type MockQBSecretsService struct {
	mock.Mock
}

func (m *MockQBSecretsService) Store(ctx context.Context, name string, value string) error {
	args := m.Called(ctx, name, value)
	return args.Error(0)
}

func (m *MockQBSecretsService) Retrieve(ctx context.Context, name string) (string, error) {
	args := m.Called(ctx, name)
	return args.String(0), args.Error(1)
}

func (m *MockQBSecretsService) Delete(ctx context.Context, name string) error {
	args := m.Called(ctx, name)
	return args.Error(0)
}

func (m *MockQBSecretsService) Exists(ctx context.Context, name string) (bool, error) {
	args := m.Called(ctx, name)
	return args.Bool(0), args.Error(1)
}

func (m *MockQBSecretsService) List(ctx context.Context) ([]string, error) {
	args := m.Called(ctx)
	return args.Get(0).([]string), args.Error(1)
}

func TestQBittorrentService_GetConfig(t *testing.T) {
	tests := []struct {
		name      string
		setupMock func(*MockQBSettingsRepo, *MockQBSecretsService)
		wantHost  string
		wantUser  string
		wantPass  string
		wantBase  string
		wantErr   bool
	}{
		{
			name: "returns config with decrypted password",
			setupMock: func(repo *MockQBSettingsRepo, sec *MockQBSecretsService) {
				repo.On("GetString", mock.Anything, SettingQBHost).Return("http://192.168.1.100:8080", nil)
				repo.On("GetString", mock.Anything, SettingQBUsername).Return("admin", nil)
				repo.On("GetString", mock.Anything, SettingQBBasePath).Return("", nil)
				sec.On("Exists", mock.Anything, SettingQBPassword).Return(true, nil)
				sec.On("Retrieve", mock.Anything, SettingQBPassword).Return("decrypted-pass", nil)
			},
			wantHost: "http://192.168.1.100:8080",
			wantUser: "admin",
			wantPass: "decrypted-pass",
			wantBase: "",
			wantErr:  false,
		},
		{
			name: "returns config without password when not set",
			setupMock: func(repo *MockQBSettingsRepo, sec *MockQBSecretsService) {
				repo.On("GetString", mock.Anything, SettingQBHost).Return("http://localhost:8080", nil)
				repo.On("GetString", mock.Anything, SettingQBUsername).Return("user", nil)
				repo.On("GetString", mock.Anything, SettingQBBasePath).Return("/qbt", nil)
				sec.On("Exists", mock.Anything, SettingQBPassword).Return(false, nil)
			},
			wantHost: "http://localhost:8080",
			wantUser: "user",
			wantPass: "",
			wantBase: "/qbt",
			wantErr:  false,
		},
		{
			name: "returns error when decrypt fails",
			setupMock: func(repo *MockQBSettingsRepo, sec *MockQBSecretsService) {
				repo.On("GetString", mock.Anything, SettingQBHost).Return("http://host:8080", nil)
				repo.On("GetString", mock.Anything, SettingQBUsername).Return("admin", nil)
				repo.On("GetString", mock.Anything, SettingQBBasePath).Return("", nil)
				sec.On("Exists", mock.Anything, SettingQBPassword).Return(true, nil)
				sec.On("Retrieve", mock.Anything, SettingQBPassword).Return("", errors.New("decrypt error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockQBSettingsRepo)
			mockSecrets := new(MockQBSecretsService)
			tt.setupMock(mockRepo, mockSecrets)

			service := NewQBittorrentService(mockRepo, mockSecrets)
			config, err := service.GetConfig(context.Background())

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.wantHost, config.Host)
			assert.Equal(t, tt.wantUser, config.Username)
			assert.Equal(t, tt.wantPass, config.Password)
			assert.Equal(t, tt.wantBase, config.BasePath)

			mockRepo.AssertExpectations(t)
			mockSecrets.AssertExpectations(t)
		})
	}
}

func TestQBittorrentService_SaveConfig(t *testing.T) {
	tests := []struct {
		name      string
		config    *qbittorrent.Config
		setupMock func(*MockQBSettingsRepo, *MockQBSecretsService)
		wantErr   bool
		errMsg    string
	}{
		{
			name: "saves config with encrypted password",
			config: &qbittorrent.Config{
				Host:     "http://192.168.1.100:8080",
				Username: "admin",
				Password: "secret123",
				BasePath: "/qbt",
			},
			setupMock: func(repo *MockQBSettingsRepo, sec *MockQBSecretsService) {
				repo.On("SetString", mock.Anything, SettingQBHost, "http://192.168.1.100:8080").Return(nil)
				repo.On("SetString", mock.Anything, SettingQBUsername, "admin").Return(nil)
				repo.On("SetString", mock.Anything, SettingQBBasePath, "/qbt").Return(nil)
				sec.On("Store", mock.Anything, SettingQBPassword, "secret123").Return(nil)
			},
			wantErr: false,
		},
		{
			name: "saves config without password when empty",
			config: &qbittorrent.Config{
				Host:     "http://192.168.1.100:8080",
				Username: "admin",
			},
			setupMock: func(repo *MockQBSettingsRepo, sec *MockQBSecretsService) {
				repo.On("SetString", mock.Anything, SettingQBHost, "http://192.168.1.100:8080").Return(nil)
				repo.On("SetString", mock.Anything, SettingQBUsername, "admin").Return(nil)
				repo.On("SetString", mock.Anything, SettingQBBasePath, "").Return(nil)
				// No Store call for empty password
			},
			wantErr: false,
		},
		{
			name:   "returns error when host is empty",
			config: &qbittorrent.Config{Host: ""},
			setupMock: func(repo *MockQBSettingsRepo, sec *MockQBSecretsService) {
				// No calls expected
			},
			wantErr: true,
			errMsg:  "host is required",
		},
		{
			name: "returns error when saving host fails",
			config: &qbittorrent.Config{
				Host:     "http://192.168.1.100:8080",
				Username: "admin",
				Password: "secret",
			},
			setupMock: func(repo *MockQBSettingsRepo, sec *MockQBSecretsService) {
				repo.On("SetString", mock.Anything, SettingQBHost, "http://192.168.1.100:8080").Return(errors.New("db write error"))
			},
			wantErr: true,
			errMsg:  "save host",
		},
		{
			name: "returns error when saving username fails",
			config: &qbittorrent.Config{
				Host:     "http://192.168.1.100:8080",
				Username: "admin",
				Password: "secret",
			},
			setupMock: func(repo *MockQBSettingsRepo, sec *MockQBSecretsService) {
				repo.On("SetString", mock.Anything, SettingQBHost, "http://192.168.1.100:8080").Return(nil)
				repo.On("SetString", mock.Anything, SettingQBUsername, "admin").Return(errors.New("db write error"))
			},
			wantErr: true,
			errMsg:  "save username",
		},
		{
			name: "returns error when saving base path fails",
			config: &qbittorrent.Config{
				Host:     "http://192.168.1.100:8080",
				Username: "admin",
				Password: "secret",
				BasePath: "/qbt",
			},
			setupMock: func(repo *MockQBSettingsRepo, sec *MockQBSecretsService) {
				repo.On("SetString", mock.Anything, SettingQBHost, "http://192.168.1.100:8080").Return(nil)
				repo.On("SetString", mock.Anything, SettingQBUsername, "admin").Return(nil)
				repo.On("SetString", mock.Anything, SettingQBBasePath, "/qbt").Return(errors.New("db write error"))
			},
			wantErr: true,
			errMsg:  "save base path",
		},
		{
			name: "returns error when encryption fails",
			config: &qbittorrent.Config{
				Host:     "http://host:8080",
				Username: "admin",
				Password: "secret",
			},
			setupMock: func(repo *MockQBSettingsRepo, sec *MockQBSecretsService) {
				repo.On("SetString", mock.Anything, SettingQBHost, "http://host:8080").Return(nil)
				repo.On("SetString", mock.Anything, SettingQBUsername, "admin").Return(nil)
				repo.On("SetString", mock.Anything, SettingQBBasePath, "").Return(nil)
				sec.On("Store", mock.Anything, SettingQBPassword, "secret").Return(errors.New("encryption failed"))
			},
			wantErr: true,
			errMsg:  "encrypt password",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockQBSettingsRepo)
			mockSecrets := new(MockQBSecretsService)
			tt.setupMock(mockRepo, mockSecrets)

			service := NewQBittorrentService(mockRepo, mockSecrets)
			err := service.SaveConfig(context.Background(), tt.config)

			if tt.wantErr {
				assert.Error(t, err)
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

func TestQBittorrentService_TestConnection_NotConfigured(t *testing.T) {
	mockRepo := new(MockQBSettingsRepo)
	mockSecrets := new(MockQBSecretsService)

	mockRepo.On("GetString", mock.Anything, SettingQBHost).Return("", errors.New("not found"))
	mockRepo.On("GetString", mock.Anything, SettingQBUsername).Return("", errors.New("not found"))
	mockRepo.On("GetString", mock.Anything, SettingQBBasePath).Return("", errors.New("not found"))
	mockSecrets.On("Exists", mock.Anything, SettingQBPassword).Return(false, nil)

	service := NewQBittorrentService(mockRepo, mockSecrets)
	info, err := service.TestConnection(context.Background())

	assert.Nil(t, info)
	assert.Error(t, err)

	var connErr *qbittorrent.ConnectionError
	assert.ErrorAs(t, err, &connErr)
	assert.Equal(t, qbittorrent.ErrCodeNotConfigured, connErr.Code)
}

func TestQBittorrentService_TestConnectionWithConfig_EmptyHost(t *testing.T) {
	service := NewQBittorrentService(new(MockQBSettingsRepo), new(MockQBSecretsService))
	info, err := service.TestConnectionWithConfig(context.Background(), &qbittorrent.Config{Host: ""})

	assert.Nil(t, info)
	assert.Error(t, err)

	var connErr *qbittorrent.ConnectionError
	assert.ErrorAs(t, err, &connErr)
	assert.Equal(t, qbittorrent.ErrCodeNotConfigured, connErr.Code)
}

func TestQBittorrentService_IsConfigured(t *testing.T) {
	tests := []struct {
		name      string
		setupMock func(*MockQBSettingsRepo)
		want      bool
	}{
		{
			name: "returns true when host is set",
			setupMock: func(repo *MockQBSettingsRepo) {
				repo.On("GetString", mock.Anything, SettingQBHost).Return("http://host:8080", nil)
			},
			want: true,
		},
		{
			name: "returns false when host is empty",
			setupMock: func(repo *MockQBSettingsRepo) {
				repo.On("GetString", mock.Anything, SettingQBHost).Return("", nil)
			},
			want: false,
		},
		{
			name: "returns false when host not found",
			setupMock: func(repo *MockQBSettingsRepo) {
				repo.On("GetString", mock.Anything, SettingQBHost).Return("", errors.New("not found"))
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockQBSettingsRepo)
			tt.setupMock(mockRepo)

			service := NewQBittorrentService(mockRepo, new(MockQBSecretsService))
			result := service.IsConfigured(context.Background())

			assert.Equal(t, tt.want, result)
			mockRepo.AssertExpectations(t)
		})
	}
}
