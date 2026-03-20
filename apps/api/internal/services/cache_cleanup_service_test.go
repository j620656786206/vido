package services

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCacheCleanupService_ClearCacheByType_InvalidType(t *testing.T) {
	db := setupCacheTestDB(t)
	svc := NewCacheCleanupService(db, "")

	result, err := svc.ClearCacheByType(context.Background(), "bogus")
	assert.ErrorIs(t, err, ErrInvalidCacheType)
	assert.Nil(t, result)
}

func TestCacheCleanupService_ClearCacheByType_Metadata(t *testing.T) {
	db := setupCacheTestDB(t)
	svc := NewCacheCleanupService(db, "")

	// Insert test data
	_, err := db.Exec(`INSERT INTO cache_entries (key, value, type, expires_at) VALUES ('k1', 'v1', 'tmdb', '2099-01-01')`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO cache_entries (key, value, type, expires_at) VALUES ('k2', 'v2', 'tmdb', '2099-01-01')`)
	require.NoError(t, err)

	result, err := svc.ClearCacheByType(context.Background(), "metadata")
	require.NoError(t, err)
	assert.Equal(t, "metadata", result.Type)
	assert.Equal(t, int64(2), result.EntriesRemoved)

	// Verify table is empty
	var count int
	db.QueryRow("SELECT COUNT(*) FROM cache_entries").Scan(&count)
	assert.Equal(t, 0, count)
}

func TestCacheCleanupService_ClearCacheByType_AI(t *testing.T) {
	db := setupCacheTestDB(t)
	svc := NewCacheCleanupService(db, "")

	_, err := db.Exec(`INSERT INTO ai_cache (id, filename_hash, provider, request_prompt, response_json, expires_at) VALUES ('a1', 'h1', 'openai', 'prompt', '{}', '2099-01-01')`)
	require.NoError(t, err)

	result, err := svc.ClearCacheByType(context.Background(), "ai")
	require.NoError(t, err)
	assert.Equal(t, int64(1), result.EntriesRemoved)
}

func TestCacheCleanupService_ClearCacheByType_Douban(t *testing.T) {
	db := setupCacheTestDB(t)
	svc := NewCacheCleanupService(db, "")

	_, err := db.Exec(`INSERT INTO douban_cache (id, douban_id, title, expires_at) VALUES ('d1', '123', 'Test', '2099-01-01')`)
	require.NoError(t, err)

	result, err := svc.ClearCacheByType(context.Background(), "douban")
	require.NoError(t, err)
	assert.Equal(t, int64(1), result.EntriesRemoved)
}

func TestCacheCleanupService_ClearCacheByType_Wikipedia(t *testing.T) {
	db := setupCacheTestDB(t)
	svc := NewCacheCleanupService(db, "")

	_, err := db.Exec(`INSERT INTO wikipedia_cache (id, query, page_title, title, expires_at) VALUES ('w1', 'q', 'p', 'Test', '2099-01-01')`)
	require.NoError(t, err)

	result, err := svc.ClearCacheByType(context.Background(), "wikipedia")
	require.NoError(t, err)
	assert.Equal(t, int64(1), result.EntriesRemoved)
}

func TestCacheCleanupService_ClearCacheByType_Image(t *testing.T) {
	tmpDir := t.TempDir()
	db := setupCacheTestDB(t)

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "img1.jpg"), make([]byte, 1024), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "img2.png"), make([]byte, 2048), 0644))

	svc := NewCacheCleanupService(db, tmpDir)

	result, err := svc.ClearCacheByType(context.Background(), "image")
	require.NoError(t, err)
	assert.Equal(t, "image", result.Type)
	assert.Equal(t, int64(2), result.EntriesRemoved)
	assert.Equal(t, int64(3072), result.BytesReclaimed)

	// Verify files removed
	entries, _ := os.ReadDir(tmpDir)
	assert.Empty(t, entries)
}

func TestCacheCleanupService_ClearCacheByAge_InvalidDays(t *testing.T) {
	db := setupCacheTestDB(t)
	svc := NewCacheCleanupService(db, "")

	result, err := svc.ClearCacheByAge(context.Background(), 0)
	assert.Error(t, err)
	assert.Nil(t, result)

	result, err = svc.ClearCacheByAge(context.Background(), -1)
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestCacheCleanupService_ClearCacheByAge_RemovesOldEntries(t *testing.T) {
	db := setupCacheTestDB(t)
	svc := NewCacheCleanupService(db, "")

	oldTime := time.Now().AddDate(0, 0, -60).Format("2006-01-02 15:04:05")
	newTime := time.Now().Format("2006-01-02 15:04:05")

	// Old entry — should be removed
	_, err := db.Exec(`INSERT INTO cache_entries (key, value, type, expires_at, created_at) VALUES ('old', 'v', 'tmdb', '2099-01-01', ?)`, oldTime)
	require.NoError(t, err)
	// New entry — should remain
	_, err = db.Exec(`INSERT INTO cache_entries (key, value, type, expires_at, created_at) VALUES ('new', 'v', 'tmdb', '2099-01-01', ?)`, newTime)
	require.NoError(t, err)

	result, err := svc.ClearCacheByAge(context.Background(), 30)
	require.NoError(t, err)
	assert.Greater(t, result.EntriesRemoved, int64(0))

	// Verify new entry remains
	var count int
	db.QueryRow("SELECT COUNT(*) FROM cache_entries").Scan(&count)
	assert.Equal(t, 1, count)
}

func TestCacheCleanupService_ClearCacheByAge_OldImages(t *testing.T) {
	tmpDir := t.TempDir()
	db := setupCacheTestDB(t)

	// Create file and set mod time to 60 days ago
	imgPath := filepath.Join(tmpDir, "old.jpg")
	require.NoError(t, os.WriteFile(imgPath, make([]byte, 500), 0644))
	oldTime := time.Now().AddDate(0, 0, -60)
	require.NoError(t, os.Chtimes(imgPath, oldTime, oldTime))

	// Create recent file
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "new.jpg"), make([]byte, 500), 0644))

	svc := NewCacheCleanupService(db, tmpDir)

	result, err := svc.ClearCacheByAge(context.Background(), 30)
	require.NoError(t, err)
	assert.Greater(t, result.EntriesRemoved, int64(0))
	assert.Equal(t, int64(500), result.BytesReclaimed)

	// Verify only new file remains
	entries, _ := os.ReadDir(tmpDir)
	assert.Len(t, entries, 1)
	assert.Equal(t, "new.jpg", entries[0].Name())
}

func TestCacheCleanupService_ClearCacheByType_EmptyImageDir(t *testing.T) {
	db := setupCacheTestDB(t)
	svc := NewCacheCleanupService(db, "")

	result, err := svc.ClearCacheByType(context.Background(), "image")
	require.NoError(t, err)
	assert.Equal(t, int64(0), result.EntriesRemoved)
}

func TestCacheCleanupService_ValidCacheTypes(t *testing.T) {
	expected := []string{"image", "ai", "metadata", "douban", "wikipedia"}
	assert.Equal(t, expected, ValidCacheTypes)
}

func TestCacheCleanupService_InterfaceCompliance(t *testing.T) {
	var _ CacheCleanupServiceInterface = (*CacheCleanupService)(nil)
}
