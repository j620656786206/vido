package repository

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func setupSecretsTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)

	// Create secrets table
	_, err = db.Exec(`
		CREATE TABLE secrets (
			id TEXT PRIMARY KEY,
			name TEXT UNIQUE NOT NULL,
			encrypted_value TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX idx_secrets_name ON secrets(name);
	`)
	require.NoError(t, err)

	return db
}

func TestSecretsRepository_Set_NewSecret(t *testing.T) {
	db := setupSecretsTestDB(t)
	defer db.Close()

	repo := NewSecretsRepository(db)
	ctx := context.Background()

	err := repo.Set(ctx, "tmdb_api_key", "encrypted-value-base64")
	assert.NoError(t, err)

	// Verify it was stored
	value, err := repo.Get(ctx, "tmdb_api_key")
	assert.NoError(t, err)
	assert.Equal(t, "encrypted-value-base64", value)
}

func TestSecretsRepository_Set_UpdateExisting(t *testing.T) {
	db := setupSecretsTestDB(t)
	defer db.Close()

	repo := NewSecretsRepository(db)
	ctx := context.Background()

	// Set initial value
	err := repo.Set(ctx, "api_key", "initial-value")
	require.NoError(t, err)

	// Update value
	err = repo.Set(ctx, "api_key", "updated-value")
	assert.NoError(t, err)

	// Verify it was updated
	value, err := repo.Get(ctx, "api_key")
	assert.NoError(t, err)
	assert.Equal(t, "updated-value", value)
}

func TestSecretsRepository_Set_EmptyName(t *testing.T) {
	db := setupSecretsTestDB(t)
	defer db.Close()

	repo := NewSecretsRepository(db)
	ctx := context.Background()

	err := repo.Set(ctx, "", "value")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "name cannot be empty")
}

func TestSecretsRepository_Set_EmptyValue(t *testing.T) {
	db := setupSecretsTestDB(t)
	defer db.Close()

	repo := NewSecretsRepository(db)
	ctx := context.Background()

	err := repo.Set(ctx, "key", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "encrypted value cannot be empty")
}

func TestSecretsRepository_Get_NotFound(t *testing.T) {
	db := setupSecretsTestDB(t)
	defer db.Close()

	repo := NewSecretsRepository(db)
	ctx := context.Background()

	_, err := repo.Get(ctx, "nonexistent")
	assert.ErrorIs(t, err, ErrSecretNotFound)
}

func TestSecretsRepository_Get_EmptyName(t *testing.T) {
	db := setupSecretsTestDB(t)
	defer db.Close()

	repo := NewSecretsRepository(db)
	ctx := context.Background()

	_, err := repo.Get(ctx, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "name cannot be empty")
}

func TestSecretsRepository_Delete(t *testing.T) {
	db := setupSecretsTestDB(t)
	defer db.Close()

	repo := NewSecretsRepository(db)
	ctx := context.Background()

	// Create a secret
	err := repo.Set(ctx, "to_delete", "value")
	require.NoError(t, err)

	// Delete it
	err = repo.Delete(ctx, "to_delete")
	assert.NoError(t, err)

	// Verify it's gone
	_, err = repo.Get(ctx, "to_delete")
	assert.ErrorIs(t, err, ErrSecretNotFound)
}

func TestSecretsRepository_Delete_NotFound(t *testing.T) {
	db := setupSecretsTestDB(t)
	defer db.Close()

	repo := NewSecretsRepository(db)
	ctx := context.Background()

	err := repo.Delete(ctx, "nonexistent")
	assert.ErrorIs(t, err, ErrSecretNotFound)
}

func TestSecretsRepository_Exists(t *testing.T) {
	db := setupSecretsTestDB(t)
	defer db.Close()

	repo := NewSecretsRepository(db)
	ctx := context.Background()

	// Create a secret
	err := repo.Set(ctx, "existing", "value")
	require.NoError(t, err)

	// Check exists
	exists, err := repo.Exists(ctx, "existing")
	assert.NoError(t, err)
	assert.True(t, exists)

	// Check not exists
	exists, err = repo.Exists(ctx, "nonexistent")
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestSecretsRepository_List(t *testing.T) {
	db := setupSecretsTestDB(t)
	defer db.Close()

	repo := NewSecretsRepository(db)
	ctx := context.Background()

	// Create some secrets
	err := repo.Set(ctx, "beta_key", "value1")
	require.NoError(t, err)
	err = repo.Set(ctx, "alpha_key", "value2")
	require.NoError(t, err)
	err = repo.Set(ctx, "gamma_key", "value3")
	require.NoError(t, err)

	// List should be sorted alphabetically
	names, err := repo.List(ctx)
	assert.NoError(t, err)
	assert.Equal(t, []string{"alpha_key", "beta_key", "gamma_key"}, names)
}

func TestSecretsRepository_List_Empty(t *testing.T) {
	db := setupSecretsTestDB(t)
	defer db.Close()

	repo := NewSecretsRepository(db)
	ctx := context.Background()

	names, err := repo.List(ctx)
	assert.NoError(t, err)
	assert.Empty(t, names)
}

func TestSecretsRepository_GetFull(t *testing.T) {
	db := setupSecretsTestDB(t)
	defer db.Close()

	repo := NewSecretsRepository(db)
	ctx := context.Background()

	// Create a secret
	err := repo.Set(ctx, "full_test", "encrypted-data")
	require.NoError(t, err)

	// Get full record
	secret, err := repo.GetFull(ctx, "full_test")
	assert.NoError(t, err)
	assert.NotEmpty(t, secret.ID)
	assert.Equal(t, "full_test", secret.Name)
	assert.Equal(t, "encrypted-data", secret.EncryptedValue)
	assert.False(t, secret.CreatedAt.IsZero())
	assert.False(t, secret.UpdatedAt.IsZero())
}

func TestSecretsRepository_GetFull_NotFound(t *testing.T) {
	db := setupSecretsTestDB(t)
	defer db.Close()

	repo := NewSecretsRepository(db)
	ctx := context.Background()

	_, err := repo.GetFull(ctx, "nonexistent")
	assert.ErrorIs(t, err, ErrSecretNotFound)
}

func TestSecretsRepository_ListAll(t *testing.T) {
	db := setupSecretsTestDB(t)
	defer db.Close()

	repo := NewSecretsRepository(db)
	ctx := context.Background()

	// Create some secrets
	err := repo.Set(ctx, "key1", "value1")
	require.NoError(t, err)
	err = repo.Set(ctx, "key2", "value2")
	require.NoError(t, err)

	// ListAll should return metadata without encrypted values
	secrets, err := repo.ListAll(ctx)
	assert.NoError(t, err)
	assert.Len(t, secrets, 2)

	// Verify info contains metadata but NOT encrypted values
	for _, info := range secrets {
		assert.NotEmpty(t, info.ID)
		assert.NotEmpty(t, info.Name)
		assert.False(t, info.CreatedAt.IsZero())
	}
}
