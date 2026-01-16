package services

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
)

// MockSettingsRepository is a mock implementation of SettingsRepositoryInterface
type MockSettingsRepository struct {
	mock.Mock
}

func (m *MockSettingsRepository) Set(ctx context.Context, setting *models.Setting) error {
	args := m.Called(ctx, setting)
	return args.Error(0)
}

func (m *MockSettingsRepository) Get(ctx context.Context, key string) (*models.Setting, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Setting), args.Error(1)
}

func (m *MockSettingsRepository) GetAll(ctx context.Context) ([]models.Setting, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.Setting), args.Error(1)
}

func (m *MockSettingsRepository) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *MockSettingsRepository) GetString(ctx context.Context, key string) (string, error) {
	args := m.Called(ctx, key)
	return args.String(0), args.Error(1)
}

func (m *MockSettingsRepository) GetInt(ctx context.Context, key string) (int, error) {
	args := m.Called(ctx, key)
	return args.Int(0), args.Error(1)
}

func (m *MockSettingsRepository) GetBool(ctx context.Context, key string) (bool, error) {
	args := m.Called(ctx, key)
	return args.Bool(0), args.Error(1)
}

func (m *MockSettingsRepository) SetString(ctx context.Context, key, value string) error {
	args := m.Called(ctx, key, value)
	return args.Error(0)
}

func (m *MockSettingsRepository) SetInt(ctx context.Context, key string, value int) error {
	args := m.Called(ctx, key, value)
	return args.Error(0)
}

func (m *MockSettingsRepository) SetBool(ctx context.Context, key string, value bool) error {
	args := m.Called(ctx, key, value)
	return args.Error(0)
}

// Verify mock implements interface
var _ repository.SettingsRepositoryInterface = (*MockSettingsRepository)(nil)

func TestSettingsService_Get(t *testing.T) {
	tests := []struct {
		name      string
		key       string
		setupMock func(*MockSettingsRepository)
		wantErr   bool
		errMsg    string
	}{
		{
			name: "success",
			key:  "app.name",
			setupMock: func(m *MockSettingsRepository) {
				m.On("Get", mock.Anything, "app.name").Return(&models.Setting{
					Key:   "app.name",
					Value: "Vido",
					Type:  "string",
				}, nil)
			},
			wantErr: false,
		},
		{
			name: "empty key returns error",
			key:  "",
			setupMock: func(m *MockSettingsRepository) {
				// No mock setup needed - validation should fail first
			},
			wantErr: true,
			errMsg:  "setting key cannot be empty",
		},
		{
			name: "repository error",
			key:  "missing.key",
			setupMock: func(m *MockSettingsRepository) {
				m.On("Get", mock.Anything, "missing.key").Return(nil, errors.New("not found"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockSettingsRepository)
			tt.setupMock(mockRepo)

			service := NewSettingsService(mockRepo)
			setting, err := service.Get(context.Background(), tt.key)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, setting)
				assert.Equal(t, tt.key, setting.Key)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestSettingsService_Set(t *testing.T) {
	tests := []struct {
		name      string
		setting   *models.Setting
		setupMock func(*MockSettingsRepository)
		wantErr   bool
		errMsg    string
	}{
		{
			name: "success",
			setting: &models.Setting{
				Key:   "app.name",
				Value: "Vido",
				Type:  "string",
			},
			setupMock: func(m *MockSettingsRepository) {
				m.On("Set", mock.Anything, mock.AnythingOfType("*models.Setting")).Return(nil)
			},
			wantErr: false,
		},
		{
			name:    "nil setting returns error",
			setting: nil,
			setupMock: func(m *MockSettingsRepository) {
				// No mock setup needed - validation should fail first
			},
			wantErr: true,
			errMsg:  "setting cannot be nil",
		},
		{
			name: "empty key returns error",
			setting: &models.Setting{
				Key:   "",
				Value: "test",
				Type:  "string",
			},
			setupMock: func(m *MockSettingsRepository) {
				// No mock setup needed - validation should fail first
			},
			wantErr: true,
			errMsg:  "setting key cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockSettingsRepository)
			tt.setupMock(mockRepo)

			service := NewSettingsService(mockRepo)
			err := service.Set(context.Background(), tt.setting)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestSettingsService_GetAll(t *testing.T) {
	mockRepo := new(MockSettingsRepository)
	mockRepo.On("GetAll", mock.Anything).Return(
		[]models.Setting{
			{Key: "app.name", Value: "Vido", Type: "string"},
			{Key: "app.debug", Value: "true", Type: "bool"},
		},
		nil,
	)

	service := NewSettingsService(mockRepo)
	settings, err := service.GetAll(context.Background())

	assert.NoError(t, err)
	assert.Len(t, settings, 2)
	mockRepo.AssertExpectations(t)
}

func TestSettingsService_GetStringWithDefault(t *testing.T) {
	t.Run("returns value when found", func(t *testing.T) {
		mockRepo := new(MockSettingsRepository)
		mockRepo.On("GetString", mock.Anything, "existing.key").Return("found value", nil)

		service := NewSettingsService(mockRepo)
		value := service.GetStringWithDefault(context.Background(), "existing.key", "default")

		assert.Equal(t, "found value", value)
	})

	t.Run("returns default when not found", func(t *testing.T) {
		mockRepo := new(MockSettingsRepository)
		mockRepo.On("GetString", mock.Anything, "missing.key").Return("", errors.New("not found"))

		service := NewSettingsService(mockRepo)
		value := service.GetStringWithDefault(context.Background(), "missing.key", "default")

		assert.Equal(t, "default", value)
	})
}

func TestSettingsService_Delete(t *testing.T) {
	tests := []struct {
		name      string
		key       string
		setupMock func(*MockSettingsRepository)
		wantErr   bool
		errMsg    string
	}{
		{
			name: "success",
			key:  "app.name",
			setupMock: func(m *MockSettingsRepository) {
				m.On("Delete", mock.Anything, "app.name").Return(nil)
			},
			wantErr: false,
		},
		{
			name: "empty key returns error",
			key:  "",
			setupMock: func(m *MockSettingsRepository) {
				// No mock setup needed - validation should fail first
			},
			wantErr: true,
			errMsg:  "setting key cannot be empty",
		},
		{
			name: "repository error",
			key:  "app.name",
			setupMock: func(m *MockSettingsRepository) {
				m.On("Delete", mock.Anything, "app.name").Return(errors.New("delete failed"))
			},
			wantErr: true,
			errMsg:  "failed to delete setting",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockSettingsRepository)
			tt.setupMock(mockRepo)

			service := NewSettingsService(mockRepo)
			err := service.Delete(context.Background(), tt.key)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestSettingsService_GetString(t *testing.T) {
	tests := []struct {
		name      string
		key       string
		setupMock func(*MockSettingsRepository)
		want      string
		wantErr   bool
		errMsg    string
	}{
		{
			name: "success",
			key:  "app.name",
			setupMock: func(m *MockSettingsRepository) {
				m.On("GetString", mock.Anything, "app.name").Return("Vido", nil)
			},
			want:    "Vido",
			wantErr: false,
		},
		{
			name: "empty key returns error",
			key:  "",
			setupMock: func(m *MockSettingsRepository) {
				// No mock setup needed - validation should fail first
			},
			wantErr: true,
			errMsg:  "setting key cannot be empty",
		},
		{
			name: "repository error",
			key:  "missing.key",
			setupMock: func(m *MockSettingsRepository) {
				m.On("GetString", mock.Anything, "missing.key").Return("", errors.New("not found"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockSettingsRepository)
			tt.setupMock(mockRepo)

			service := NewSettingsService(mockRepo)
			value, err := service.GetString(context.Background(), tt.key)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, value)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestSettingsService_GetInt(t *testing.T) {
	tests := []struct {
		name      string
		key       string
		setupMock func(*MockSettingsRepository)
		want      int
		wantErr   bool
		errMsg    string
	}{
		{
			name: "success",
			key:  "app.port",
			setupMock: func(m *MockSettingsRepository) {
				m.On("GetInt", mock.Anything, "app.port").Return(8080, nil)
			},
			want:    8080,
			wantErr: false,
		},
		{
			name: "empty key returns error",
			key:  "",
			setupMock: func(m *MockSettingsRepository) {
				// No mock setup needed - validation should fail first
			},
			wantErr: true,
			errMsg:  "setting key cannot be empty",
		},
		{
			name: "repository error",
			key:  "missing.key",
			setupMock: func(m *MockSettingsRepository) {
				m.On("GetInt", mock.Anything, "missing.key").Return(0, errors.New("not found"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockSettingsRepository)
			tt.setupMock(mockRepo)

			service := NewSettingsService(mockRepo)
			value, err := service.GetInt(context.Background(), tt.key)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, value)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestSettingsService_GetBool(t *testing.T) {
	tests := []struct {
		name      string
		key       string
		setupMock func(*MockSettingsRepository)
		want      bool
		wantErr   bool
		errMsg    string
	}{
		{
			name: "success true",
			key:  "app.debug",
			setupMock: func(m *MockSettingsRepository) {
				m.On("GetBool", mock.Anything, "app.debug").Return(true, nil)
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "success false",
			key:  "app.production",
			setupMock: func(m *MockSettingsRepository) {
				m.On("GetBool", mock.Anything, "app.production").Return(false, nil)
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "empty key returns error",
			key:  "",
			setupMock: func(m *MockSettingsRepository) {
				// No mock setup needed - validation should fail first
			},
			wantErr: true,
			errMsg:  "setting key cannot be empty",
		},
		{
			name: "repository error",
			key:  "missing.key",
			setupMock: func(m *MockSettingsRepository) {
				m.On("GetBool", mock.Anything, "missing.key").Return(false, errors.New("not found"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockSettingsRepository)
			tt.setupMock(mockRepo)

			service := NewSettingsService(mockRepo)
			value, err := service.GetBool(context.Background(), tt.key)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, value)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestSettingsService_SetString(t *testing.T) {
	tests := []struct {
		name      string
		key       string
		value     string
		setupMock func(*MockSettingsRepository)
		wantErr   bool
		errMsg    string
	}{
		{
			name:  "success",
			key:   "app.name",
			value: "Vido",
			setupMock: func(m *MockSettingsRepository) {
				m.On("SetString", mock.Anything, "app.name", "Vido").Return(nil)
			},
			wantErr: false,
		},
		{
			name:  "empty key returns error",
			key:   "",
			value: "test",
			setupMock: func(m *MockSettingsRepository) {
				// No mock setup needed - validation should fail first
			},
			wantErr: true,
			errMsg:  "setting key cannot be empty",
		},
		{
			name:  "repository error",
			key:   "app.name",
			value: "Vido",
			setupMock: func(m *MockSettingsRepository) {
				m.On("SetString", mock.Anything, "app.name", "Vido").Return(errors.New("set failed"))
			},
			wantErr: true,
			errMsg:  "failed to set string setting",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockSettingsRepository)
			tt.setupMock(mockRepo)

			service := NewSettingsService(mockRepo)
			err := service.SetString(context.Background(), tt.key, tt.value)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestSettingsService_SetInt(t *testing.T) {
	tests := []struct {
		name      string
		key       string
		value     int
		setupMock func(*MockSettingsRepository)
		wantErr   bool
		errMsg    string
	}{
		{
			name:  "success",
			key:   "app.port",
			value: 8080,
			setupMock: func(m *MockSettingsRepository) {
				m.On("SetInt", mock.Anything, "app.port", 8080).Return(nil)
			},
			wantErr: false,
		},
		{
			name:  "empty key returns error",
			key:   "",
			value: 8080,
			setupMock: func(m *MockSettingsRepository) {
				// No mock setup needed - validation should fail first
			},
			wantErr: true,
			errMsg:  "setting key cannot be empty",
		},
		{
			name:  "repository error",
			key:   "app.port",
			value: 8080,
			setupMock: func(m *MockSettingsRepository) {
				m.On("SetInt", mock.Anything, "app.port", 8080).Return(errors.New("set failed"))
			},
			wantErr: true,
			errMsg:  "failed to set int setting",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockSettingsRepository)
			tt.setupMock(mockRepo)

			service := NewSettingsService(mockRepo)
			err := service.SetInt(context.Background(), tt.key, tt.value)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestSettingsService_SetBool(t *testing.T) {
	tests := []struct {
		name      string
		key       string
		value     bool
		setupMock func(*MockSettingsRepository)
		wantErr   bool
		errMsg    string
	}{
		{
			name:  "success true",
			key:   "app.debug",
			value: true,
			setupMock: func(m *MockSettingsRepository) {
				m.On("SetBool", mock.Anything, "app.debug", true).Return(nil)
			},
			wantErr: false,
		},
		{
			name:  "success false",
			key:   "app.production",
			value: false,
			setupMock: func(m *MockSettingsRepository) {
				m.On("SetBool", mock.Anything, "app.production", false).Return(nil)
			},
			wantErr: false,
		},
		{
			name:  "empty key returns error",
			key:   "",
			value: true,
			setupMock: func(m *MockSettingsRepository) {
				// No mock setup needed - validation should fail first
			},
			wantErr: true,
			errMsg:  "setting key cannot be empty",
		},
		{
			name:  "repository error",
			key:   "app.debug",
			value: true,
			setupMock: func(m *MockSettingsRepository) {
				m.On("SetBool", mock.Anything, "app.debug", true).Return(errors.New("set failed"))
			},
			wantErr: true,
			errMsg:  "failed to set bool setting",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockSettingsRepository)
			tt.setupMock(mockRepo)

			service := NewSettingsService(mockRepo)
			err := service.SetBool(context.Background(), tt.key, tt.value)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestSettingsService_GetIntWithDefault(t *testing.T) {
	t.Run("returns value when found", func(t *testing.T) {
		mockRepo := new(MockSettingsRepository)
		mockRepo.On("GetInt", mock.Anything, "existing.key").Return(8080, nil)

		service := NewSettingsService(mockRepo)
		value := service.GetIntWithDefault(context.Background(), "existing.key", 3000)

		assert.Equal(t, 8080, value)
	})

	t.Run("returns default when not found", func(t *testing.T) {
		mockRepo := new(MockSettingsRepository)
		mockRepo.On("GetInt", mock.Anything, "missing.key").Return(0, errors.New("not found"))

		service := NewSettingsService(mockRepo)
		value := service.GetIntWithDefault(context.Background(), "missing.key", 3000)

		assert.Equal(t, 3000, value)
	})
}

func TestSettingsService_GetBoolWithDefault(t *testing.T) {
	t.Run("returns value when found", func(t *testing.T) {
		mockRepo := new(MockSettingsRepository)
		mockRepo.On("GetBool", mock.Anything, "existing.key").Return(true, nil)

		service := NewSettingsService(mockRepo)
		value := service.GetBoolWithDefault(context.Background(), "existing.key", false)

		assert.True(t, value)
	})

	t.Run("returns default when not found", func(t *testing.T) {
		mockRepo := new(MockSettingsRepository)
		mockRepo.On("GetBool", mock.Anything, "missing.key").Return(false, errors.New("not found"))

		service := NewSettingsService(mockRepo)
		value := service.GetBoolWithDefault(context.Background(), "missing.key", true)

		assert.True(t, value)
	})
}

func TestSettingsService_GetAll_Error(t *testing.T) {
	mockRepo := new(MockSettingsRepository)
	mockRepo.On("GetAll", mock.Anything).Return(nil, errors.New("database error"))

	service := NewSettingsService(mockRepo)
	settings, err := service.GetAll(context.Background())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get all settings")
	assert.Nil(t, settings)
	mockRepo.AssertExpectations(t)
}

func TestSettingsService_Set_RepositoryError(t *testing.T) {
	mockRepo := new(MockSettingsRepository)
	mockRepo.On("Set", mock.Anything, mock.AnythingOfType("*models.Setting")).Return(errors.New("set failed"))

	service := NewSettingsService(mockRepo)
	err := service.Set(context.Background(), &models.Setting{
		Key:   "app.name",
		Value: "Vido",
		Type:  "string",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to set setting")
	mockRepo.AssertExpectations(t)
}
