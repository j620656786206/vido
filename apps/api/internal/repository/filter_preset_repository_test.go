package repository

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/models"
)

func setupFilterPresetTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)

	_, err = db.Exec(`
		CREATE TABLE filter_presets (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			filters TEXT NOT NULL DEFAULT '{}',
			sort_order INTEGER NOT NULL DEFAULT 0,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX idx_filter_presets_sort_order ON filter_presets(sort_order);
	`)
	require.NoError(t, err)

	t.Cleanup(func() { db.Close() })
	return db
}

func newTestPreset(name string, order int) *models.FilterPreset {
	return &models.FilterPreset{
		Name:      name,
		Filters:   `{"genre":"28","region":"KR"}`,
		SortOrder: order,
	}
}

func TestFilterPresetRepository_Create(t *testing.T) {
	db := setupFilterPresetTestDB(t)
	repo := NewFilterPresetRepository(db)
	ctx := context.Background()

	preset := newTestPreset("2024年後韓劇", 0)
	err := repo.Create(ctx, preset)
	require.NoError(t, err)
	assert.NotEmpty(t, preset.ID, "should auto-generate UUID")
	assert.False(t, preset.CreatedAt.IsZero(), "should stamp created_at")
}

func TestFilterPresetRepository_GetAll_OrderedBySortOrder(t *testing.T) {
	db := setupFilterPresetTestDB(t)
	repo := NewFilterPresetRepository(db)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, newTestPreset("第二", 1)))
	require.NoError(t, repo.Create(ctx, newTestPreset("第一", 0)))

	presets, err := repo.GetAll(ctx)
	require.NoError(t, err)
	require.Len(t, presets, 2)
	assert.Equal(t, "第一", presets[0].Name)
	assert.Equal(t, "第二", presets[1].Name)
	assert.Equal(t, `{"genre":"28","region":"KR"}`, presets[0].Filters)
}

func TestFilterPresetRepository_GetAll_Empty(t *testing.T) {
	db := setupFilterPresetTestDB(t)
	repo := NewFilterPresetRepository(db)

	presets, err := repo.GetAll(context.Background())
	require.NoError(t, err)
	assert.Empty(t, presets)
}

func TestFilterPresetRepository_Delete(t *testing.T) {
	db := setupFilterPresetTestDB(t)
	repo := NewFilterPresetRepository(db)
	ctx := context.Background()

	preset := newTestPreset("待刪除", 0)
	require.NoError(t, repo.Create(ctx, preset))

	require.NoError(t, repo.Delete(ctx, preset.ID))

	presets, err := repo.GetAll(ctx)
	require.NoError(t, err)
	assert.Empty(t, presets)
}

func TestFilterPresetRepository_Delete_NotFound(t *testing.T) {
	db := setupFilterPresetTestDB(t)
	repo := NewFilterPresetRepository(db)

	err := repo.Delete(context.Background(), "nonexistent")
	assert.True(t, errors.Is(err, ErrFilterPresetNotFound))
}

func TestFilterPresetRepository_Count(t *testing.T) {
	db := setupFilterPresetTestDB(t)
	repo := NewFilterPresetRepository(db)
	ctx := context.Background()

	count, err := repo.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, count)

	require.NoError(t, repo.Create(ctx, newTestPreset("一", 0)))
	require.NoError(t, repo.Create(ctx, newTestPreset("二", 1)))

	count, err = repo.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}
