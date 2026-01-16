package secrets

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockSecretsRepository is a mock implementation of SecretsRepositoryInterface
type MockSecretsRepository struct {
	mock.Mock
}

func (m *MockSecretsRepository) Set(ctx context.Context, name string, encryptedValue string) error {
	args := m.Called(ctx, name, encryptedValue)
	return args.Error(0)
}

func (m *MockSecretsRepository) Get(ctx context.Context, name string) (string, error) {
	args := m.Called(ctx, name)
	return args.String(0), args.Error(1)
}

func (m *MockSecretsRepository) Delete(ctx context.Context, name string) error {
	args := m.Called(ctx, name)
	return args.Error(0)
}

func (m *MockSecretsRepository) Exists(ctx context.Context, name string) (bool, error) {
	args := m.Called(ctx, name)
	return args.Bool(0), args.Error(1)
}

func (m *MockSecretsRepository) List(ctx context.Context) ([]string, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

// Helper to create a test key (exactly 32 bytes for AES-256)
func createTestKey() []byte {
	// Exactly 32 bytes for AES-256
	return []byte("12345678901234567890123456789012")
}

func TestNewSecretsService(t *testing.T) {
	mockRepo := new(MockSecretsRepository)
	key := createTestKey()

	svc, err := NewSecretsService(mockRepo, key)
	require.NoError(t, err)
	assert.NotNil(t, svc)
}

func TestNewSecretsService_InvalidKeySize(t *testing.T) {
	mockRepo := new(MockSecretsRepository)

	// Key too short
	_, err := NewSecretsService(mockRepo, []byte("short-key"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid key size")
}

func TestSecretsService_Store(t *testing.T) {
	mockRepo := new(MockSecretsRepository)
	key := createTestKey()

	svc, err := NewSecretsService(mockRepo, key)
	require.NoError(t, err)

	ctx := context.Background()
	secretName := "tmdb_api_key"
	secretValue := "sk-test-api-key-12345"

	// Set expectation: any encrypted value is accepted
	mockRepo.On("Set", ctx, secretName, mock.AnythingOfType("string")).Return(nil)

	err = svc.Store(ctx, secretName, secretValue)
	assert.NoError(t, err)

	mockRepo.AssertExpectations(t)
}

func TestSecretsService_Store_RepositoryError(t *testing.T) {
	mockRepo := new(MockSecretsRepository)
	key := createTestKey()

	svc, err := NewSecretsService(mockRepo, key)
	require.NoError(t, err)

	ctx := context.Background()

	mockRepo.On("Set", ctx, "test", mock.AnythingOfType("string")).Return(errors.New("db error"))

	err = svc.Store(ctx, "test", "value")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "db error")
}

func TestSecretsService_Retrieve(t *testing.T) {
	mockRepo := new(MockSecretsRepository)
	key := createTestKey()

	svc, err := NewSecretsService(mockRepo, key)
	require.NoError(t, err)

	ctx := context.Background()
	secretName := "tmdb_api_key"
	secretValue := "sk-test-api-key-12345"

	// First, store the secret to get a valid encrypted value
	var storedEncryptedValue string
	mockRepo.On("Set", ctx, secretName, mock.AnythingOfType("string")).Run(func(args mock.Arguments) {
		storedEncryptedValue = args.String(2)
	}).Return(nil)

	err = svc.Store(ctx, secretName, secretValue)
	require.NoError(t, err)

	// Now set up Get to return the stored encrypted value
	mockRepo.On("Get", ctx, secretName).Return(storedEncryptedValue, nil)

	// Retrieve and verify
	retrieved, err := svc.Retrieve(ctx, secretName)
	assert.NoError(t, err)
	assert.Equal(t, secretValue, retrieved)

	mockRepo.AssertExpectations(t)
}

func TestSecretsService_Retrieve_NotFound(t *testing.T) {
	mockRepo := new(MockSecretsRepository)
	key := createTestKey()

	svc, err := NewSecretsService(mockRepo, key)
	require.NoError(t, err)

	ctx := context.Background()

	mockRepo.On("Get", ctx, "nonexistent").Return("", ErrSecretNotFound)

	_, err = svc.Retrieve(ctx, "nonexistent")
	assert.ErrorIs(t, err, ErrSecretNotFound)
}

func TestSecretsService_Delete(t *testing.T) {
	mockRepo := new(MockSecretsRepository)
	key := createTestKey()

	svc, err := NewSecretsService(mockRepo, key)
	require.NoError(t, err)

	ctx := context.Background()

	mockRepo.On("Delete", ctx, "test_secret").Return(nil)

	err = svc.Delete(ctx, "test_secret")
	assert.NoError(t, err)

	mockRepo.AssertExpectations(t)
}

func TestSecretsService_Exists(t *testing.T) {
	mockRepo := new(MockSecretsRepository)
	key := createTestKey()

	svc, err := NewSecretsService(mockRepo, key)
	require.NoError(t, err)

	ctx := context.Background()

	mockRepo.On("Exists", ctx, "existing").Return(true, nil)
	mockRepo.On("Exists", ctx, "nonexistent").Return(false, nil)

	exists, err := svc.Exists(ctx, "existing")
	assert.NoError(t, err)
	assert.True(t, exists)

	exists, err = svc.Exists(ctx, "nonexistent")
	assert.NoError(t, err)
	assert.False(t, exists)

	mockRepo.AssertExpectations(t)
}

func TestSecretsService_List(t *testing.T) {
	mockRepo := new(MockSecretsRepository)
	key := createTestKey()

	svc, err := NewSecretsService(mockRepo, key)
	require.NoError(t, err)

	ctx := context.Background()
	expectedNames := []string{"secret1", "secret2", "secret3"}

	mockRepo.On("List", ctx).Return(expectedNames, nil)

	names, err := svc.List(ctx)
	assert.NoError(t, err)
	assert.Equal(t, expectedNames, names)

	mockRepo.AssertExpectations(t)
}

func TestSecretsService_StoreAndRetrieve_Roundtrip(t *testing.T) {
	mockRepo := new(MockSecretsRepository)
	key := createTestKey()

	svc, err := NewSecretsService(mockRepo, key)
	require.NoError(t, err)

	ctx := context.Background()

	testCases := []struct {
		name  string
		value string
	}{
		{"short_key", "abc"},
		{"api_key", "sk-1234567890abcdef"},
		{"unicode_secret", "密碼測試123"},
		{"special_chars", "!@#$%^&*()_+-="},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var storedValue string
			mockRepo.On("Set", ctx, tc.name, mock.AnythingOfType("string")).Run(func(args mock.Arguments) {
				storedValue = args.String(2)
			}).Return(nil).Once()

			err := svc.Store(ctx, tc.name, tc.value)
			require.NoError(t, err)

			mockRepo.On("Get", ctx, tc.name).Return(storedValue, nil).Once()

			retrieved, err := svc.Retrieve(ctx, tc.name)
			require.NoError(t, err)
			assert.Equal(t, tc.value, retrieved)
		})
	}
}
