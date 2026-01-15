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
