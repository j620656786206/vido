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

func setupExploreBlockTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)

	_, err = db.Exec(`
		CREATE TABLE explore_blocks (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			content_type TEXT NOT NULL CHECK(content_type IN ('movie', 'tv')),
			genre_ids TEXT NOT NULL DEFAULT '',
			language TEXT NOT NULL DEFAULT '',
			region TEXT NOT NULL DEFAULT '',
			sort_by TEXT NOT NULL DEFAULT '',
			max_items INTEGER NOT NULL DEFAULT 20,
			sort_order INTEGER NOT NULL DEFAULT 0,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX idx_explore_blocks_sort_order ON explore_blocks(sort_order);
	`)
	require.NoError(t, err)

	t.Cleanup(func() { db.Close() })
	return db
}

func newTestBlock(name string, ct models.ExploreBlockContentType, order int) *models.ExploreBlock {
	return &models.ExploreBlock{
		Name:        name,
		ContentType: ct,
		MaxItems:    20,
		SortOrder:   order,
		SortBy:      "popularity.desc",
	}
}

func TestExploreBlockRepository_Create(t *testing.T) {
	db := setupExploreBlockTestDB(t)
	repo := NewExploreBlockRepository(db)
	ctx := context.Background()

	block := newTestBlock("熱門電影", models.ExploreBlockContentMovie, 0)
	err := repo.Create(ctx, block)
	require.NoError(t, err)
	assert.NotEmpty(t, block.ID, "should auto-generate UUID")
	assert.False(t, block.CreatedAt.IsZero())
	assert.False(t, block.UpdatedAt.IsZero())
}

func TestExploreBlockRepository_Create_Nil(t *testing.T) {
	db := setupExploreBlockTestDB(t)
	repo := NewExploreBlockRepository(db)
	err := repo.Create(context.Background(), nil)
	assert.Error(t, err)
}

func TestExploreBlockRepository_GetByID(t *testing.T) {
	db := setupExploreBlockTestDB(t)
	repo := NewExploreBlockRepository(db)
	ctx := context.Background()

	block := newTestBlock("熱門韓劇", models.ExploreBlockContentTV, 1)
	block.Region = "KR"
	block.Language = "ko"
	block.GenreIDs = "18,10765"
	require.NoError(t, repo.Create(ctx, block))

	got, err := repo.GetByID(ctx, block.ID)
	require.NoError(t, err)
	assert.Equal(t, block.Name, got.Name)
	assert.Equal(t, models.ExploreBlockContentTV, got.ContentType)
	assert.Equal(t, "KR", got.Region)
	assert.Equal(t, "ko", got.Language)
	assert.Equal(t, "18,10765", got.GenreIDs)
	assert.Equal(t, 1, got.SortOrder)
}

func TestExploreBlockRepository_GetByID_NotFound(t *testing.T) {
	db := setupExploreBlockTestDB(t)
	repo := NewExploreBlockRepository(db)
	_, err := repo.GetByID(context.Background(), "nonexistent")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrExploreBlockNotFound))
}

func TestExploreBlockRepository_GetAll_OrderedBySortOrder(t *testing.T) {
	db := setupExploreBlockTestDB(t)
	repo := NewExploreBlockRepository(db)
	ctx := context.Background()

	b2 := newTestBlock("第二", models.ExploreBlockContentMovie, 2)
	b0 := newTestBlock("第零", models.ExploreBlockContentMovie, 0)
	b1 := newTestBlock("第一", models.ExploreBlockContentMovie, 1)
	require.NoError(t, repo.Create(ctx, b2))
	require.NoError(t, repo.Create(ctx, b0))
	require.NoError(t, repo.Create(ctx, b1))

	blocks, err := repo.GetAll(ctx)
	require.NoError(t, err)
	require.Len(t, blocks, 3)
	assert.Equal(t, "第零", blocks[0].Name)
	assert.Equal(t, "第一", blocks[1].Name)
	assert.Equal(t, "第二", blocks[2].Name)
}

func TestExploreBlockRepository_Update(t *testing.T) {
	db := setupExploreBlockTestDB(t)
	repo := NewExploreBlockRepository(db)
	ctx := context.Background()

	block := newTestBlock("原始", models.ExploreBlockContentMovie, 0)
	require.NoError(t, repo.Create(ctx, block))

	block.Name = "已更新"
	block.MaxItems = 30
	block.GenreIDs = "28,12"
	err := repo.Update(ctx, block)
	require.NoError(t, err)

	got, err := repo.GetByID(ctx, block.ID)
	require.NoError(t, err)
	assert.Equal(t, "已更新", got.Name)
	assert.Equal(t, 30, got.MaxItems)
	assert.Equal(t, "28,12", got.GenreIDs)
}

func TestExploreBlockRepository_Update_NotFound(t *testing.T) {
	db := setupExploreBlockTestDB(t)
	repo := NewExploreBlockRepository(db)

	ghost := newTestBlock("幽靈", models.ExploreBlockContentMovie, 0)
	ghost.ID = "not-there"
	err := repo.Update(context.Background(), ghost)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrExploreBlockNotFound))
}

func TestExploreBlockRepository_Delete(t *testing.T) {
	db := setupExploreBlockTestDB(t)
	repo := NewExploreBlockRepository(db)
	ctx := context.Background()

	block := newTestBlock("將被刪除", models.ExploreBlockContentTV, 0)
	require.NoError(t, repo.Create(ctx, block))

	require.NoError(t, repo.Delete(ctx, block.ID))

	_, err := repo.GetByID(ctx, block.ID)
	assert.True(t, errors.Is(err, ErrExploreBlockNotFound))
}

func TestExploreBlockRepository_Delete_NotFound(t *testing.T) {
	db := setupExploreBlockTestDB(t)
	repo := NewExploreBlockRepository(db)
	err := repo.Delete(context.Background(), "does-not-exist")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrExploreBlockNotFound))
}

func TestExploreBlockRepository_Reorder(t *testing.T) {
	db := setupExploreBlockTestDB(t)
	repo := NewExploreBlockRepository(db)
	ctx := context.Background()

	a := newTestBlock("A", models.ExploreBlockContentMovie, 0)
	b := newTestBlock("B", models.ExploreBlockContentMovie, 1)
	c := newTestBlock("C", models.ExploreBlockContentMovie, 2)
	require.NoError(t, repo.Create(ctx, a))
	require.NoError(t, repo.Create(ctx, b))
	require.NoError(t, repo.Create(ctx, c))

	// Reverse the order
	err := repo.Reorder(ctx, []string{c.ID, b.ID, a.ID})
	require.NoError(t, err)

	blocks, err := repo.GetAll(ctx)
	require.NoError(t, err)
	require.Len(t, blocks, 3)
	assert.Equal(t, "C", blocks[0].Name)
	assert.Equal(t, "B", blocks[1].Name)
	assert.Equal(t, "A", blocks[2].Name)
}

func TestExploreBlockRepository_Reorder_UnknownIDRollsBack(t *testing.T) {
	db := setupExploreBlockTestDB(t)
	repo := NewExploreBlockRepository(db)
	ctx := context.Background()

	a := newTestBlock("A", models.ExploreBlockContentMovie, 0)
	b := newTestBlock("B", models.ExploreBlockContentMovie, 1)
	require.NoError(t, repo.Create(ctx, a))
	require.NoError(t, repo.Create(ctx, b))

	err := repo.Reorder(ctx, []string{b.ID, "unknown", a.ID})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrExploreBlockNotFound))

	// Original order must be preserved
	blocks, err := repo.GetAll(ctx)
	require.NoError(t, err)
	require.Len(t, blocks, 2)
	assert.Equal(t, "A", blocks[0].Name)
	assert.Equal(t, "B", blocks[1].Name)
}

func TestExploreBlockRepository_Reorder_Empty(t *testing.T) {
	db := setupExploreBlockTestDB(t)
	repo := NewExploreBlockRepository(db)
	err := repo.Reorder(context.Background(), []string{})
	assert.NoError(t, err, "empty reorder is a no-op")
}

func TestExploreBlockRepository_Count(t *testing.T) {
	db := setupExploreBlockTestDB(t)
	repo := NewExploreBlockRepository(db)
	ctx := context.Background()

	count, err := repo.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, count)

	require.NoError(t, repo.Create(ctx, newTestBlock("X", models.ExploreBlockContentMovie, 0)))
	require.NoError(t, repo.Create(ctx, newTestBlock("Y", models.ExploreBlockContentTV, 1)))

	count, err = repo.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}
