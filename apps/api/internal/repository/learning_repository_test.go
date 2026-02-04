package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/learning"
	_ "modernc.org/sqlite"
)

func setupLearningTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)

	// Create the filename_mappings table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS filename_mappings (
			id TEXT PRIMARY KEY,
			pattern TEXT NOT NULL UNIQUE,
			pattern_type TEXT NOT NULL,
			pattern_regex TEXT,
			fansub_group TEXT,
			title_pattern TEXT,
			metadata_type TEXT NOT NULL,
			metadata_id TEXT NOT NULL,
			tmdb_id INTEGER,
			confidence REAL NOT NULL DEFAULT 1.0,
			use_count INTEGER NOT NULL DEFAULT 0,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			last_used_at TIMESTAMP
		);

		CREATE INDEX IF NOT EXISTS idx_filename_mappings_pattern ON filename_mappings(pattern);
		CREATE INDEX IF NOT EXISTS idx_filename_mappings_fansub_group ON filename_mappings(fansub_group);
		CREATE INDEX IF NOT EXISTS idx_filename_mappings_title_pattern ON filename_mappings(title_pattern);
		CREATE INDEX IF NOT EXISTS idx_filename_mappings_metadata ON filename_mappings(metadata_type, metadata_id);
	`)
	require.NoError(t, err)

	return db
}

func TestLearningRepository_Save(t *testing.T) {
	db := setupLearningTestDB(t)
	defer db.Close()

	repo := NewLearningRepository(db)
	ctx := context.Background()

	mapping := &learning.FilenameMapping{
		ID:           "test-1",
		Pattern:      "[Leopard-Raws] Kimetsu no Yaiba",
		PatternType:  "fansub",
		PatternRegex: `(?i)[\[【]Leopard-Raws[\]】]\s*Kimetsu[.\s_-]+no[.\s_-]+Yaiba.*`,
		FansubGroup:  "Leopard-Raws",
		TitlePattern: "Kimetsu no Yaiba",
		MetadataType: "series",
		MetadataID:   "series-123",
		TmdbID:       85937,
		Confidence:   1.0,
		UseCount:     0,
		CreatedAt:    time.Now(),
	}

	err := repo.Save(ctx, mapping)
	require.NoError(t, err)

	// Verify it was saved
	saved, err := repo.FindByID(ctx, "test-1")
	require.NoError(t, err)
	require.NotNil(t, saved)

	assert.Equal(t, mapping.Pattern, saved.Pattern)
	assert.Equal(t, mapping.PatternType, saved.PatternType)
	assert.Equal(t, mapping.FansubGroup, saved.FansubGroup)
	assert.Equal(t, mapping.TitlePattern, saved.TitlePattern)
	assert.Equal(t, mapping.MetadataType, saved.MetadataType)
	assert.Equal(t, mapping.MetadataID, saved.MetadataID)
	assert.Equal(t, mapping.TmdbID, saved.TmdbID)
}

func TestLearningRepository_FindByID(t *testing.T) {
	db := setupLearningTestDB(t)
	defer db.Close()

	repo := NewLearningRepository(db)
	ctx := context.Background()

	// Insert test data
	_, err := db.Exec(`
		INSERT INTO filename_mappings (id, pattern, pattern_type, metadata_type, metadata_id, tmdb_id)
		VALUES ('find-test', 'Test Pattern', 'exact', 'movie', 'movie-123', 12345)
	`)
	require.NoError(t, err)

	// Find by ID
	found, err := repo.FindByID(ctx, "find-test")
	require.NoError(t, err)
	require.NotNil(t, found)

	assert.Equal(t, "find-test", found.ID)
	assert.Equal(t, "Test Pattern", found.Pattern)

	// Find non-existent
	notFound, err := repo.FindByID(ctx, "non-existent")
	require.NoError(t, err)
	assert.Nil(t, notFound)
}

func TestLearningRepository_FindByExactPattern(t *testing.T) {
	db := setupLearningTestDB(t)
	defer db.Close()

	repo := NewLearningRepository(db)
	ctx := context.Background()

	// Insert test data
	_, err := db.Exec(`
		INSERT INTO filename_mappings (id, pattern, pattern_type, metadata_type, metadata_id)
		VALUES ('exact-test', '[Group] Anime Title - 01.mkv', 'exact', 'series', 'series-123')
	`)
	require.NoError(t, err)

	// Find exact pattern
	found, err := repo.FindByExactPattern(ctx, "[Group] Anime Title - 01.mkv")
	require.NoError(t, err)
	require.NotNil(t, found)

	assert.Equal(t, "exact-test", found.ID)

	// Non-matching pattern
	notFound, err := repo.FindByExactPattern(ctx, "Different Pattern")
	require.NoError(t, err)
	assert.Nil(t, notFound)
}

func TestLearningRepository_FindByFansubAndTitle(t *testing.T) {
	db := setupLearningTestDB(t)
	defer db.Close()

	repo := NewLearningRepository(db)
	ctx := context.Background()

	// Insert test data
	_, err := db.Exec(`
		INSERT INTO filename_mappings (id, pattern, pattern_type, fansub_group, title_pattern, metadata_type, metadata_id)
		VALUES
			('fansub-1', '[SubsPlease] Frieren', 'fansub', 'SubsPlease', 'Frieren', 'series', 'series-1'),
			('fansub-2', '[SubsPlease] Other', 'fansub', 'SubsPlease', 'Other', 'series', 'series-2'),
			('fansub-3', '[OtherGroup] Frieren', 'fansub', 'OtherGroup', 'Frieren', 'series', 'series-3')
	`)
	require.NoError(t, err)

	// Find by fansub and title
	results, err := repo.FindByFansubAndTitle(ctx, "SubsPlease", "Frieren")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "fansub-1", results[0].ID)

	// No match
	noResults, err := repo.FindByFansubAndTitle(ctx, "NonExistent", "Frieren")
	require.NoError(t, err)
	assert.Len(t, noResults, 0)
}

func TestLearningRepository_ListWithRegex(t *testing.T) {
	db := setupLearningTestDB(t)
	defer db.Close()

	repo := NewLearningRepository(db)
	ctx := context.Background()

	// Insert test data with and without regex
	_, err := db.Exec(`
		INSERT INTO filename_mappings (id, pattern, pattern_type, pattern_regex, metadata_type, metadata_id)
		VALUES
			('with-regex', 'Pattern 1', 'fansub', '(?i)Pattern.*', 'series', 'series-1'),
			('no-regex', 'Pattern 2', 'exact', NULL, 'movie', 'movie-1'),
			('with-regex-2', 'Pattern 3', 'standard', '(?i)Standard.*', 'series', 'series-2')
	`)
	require.NoError(t, err)

	// List only those with regex
	results, err := repo.ListWithRegex(ctx)
	require.NoError(t, err)
	require.Len(t, results, 2)

	// Both should have regex
	for _, r := range results {
		assert.NotEmpty(t, r.PatternRegex)
	}
}

func TestLearningRepository_ListAll(t *testing.T) {
	db := setupLearningTestDB(t)
	defer db.Close()

	repo := NewLearningRepository(db)
	ctx := context.Background()

	// Insert test data
	_, err := db.Exec(`
		INSERT INTO filename_mappings (id, pattern, pattern_type, metadata_type, metadata_id)
		VALUES
			('all-1', 'Pattern 1', 'fansub', 'series', 'series-1'),
			('all-2', 'Pattern 2', 'exact', 'movie', 'movie-1'),
			('all-3', 'Pattern 3', 'standard', 'series', 'series-2')
	`)
	require.NoError(t, err)

	// List all
	results, err := repo.ListAll(ctx)
	require.NoError(t, err)
	require.Len(t, results, 3)
}

func TestLearningRepository_Delete(t *testing.T) {
	db := setupLearningTestDB(t)
	defer db.Close()

	repo := NewLearningRepository(db)
	ctx := context.Background()

	// Insert test data
	_, err := db.Exec(`
		INSERT INTO filename_mappings (id, pattern, pattern_type, metadata_type, metadata_id)
		VALUES ('delete-test', 'To Delete', 'exact', 'movie', 'movie-1')
	`)
	require.NoError(t, err)

	// Verify it exists
	found, err := repo.FindByID(ctx, "delete-test")
	require.NoError(t, err)
	require.NotNil(t, found)

	// Delete it
	err = repo.Delete(ctx, "delete-test")
	require.NoError(t, err)

	// Verify it's gone
	notFound, err := repo.FindByID(ctx, "delete-test")
	require.NoError(t, err)
	assert.Nil(t, notFound)
}

func TestLearningRepository_IncrementUseCount(t *testing.T) {
	db := setupLearningTestDB(t)
	defer db.Close()

	repo := NewLearningRepository(db)
	ctx := context.Background()

	// Insert test data
	_, err := db.Exec(`
		INSERT INTO filename_mappings (id, pattern, pattern_type, metadata_type, metadata_id, use_count)
		VALUES ('count-test', 'Count Pattern', 'exact', 'movie', 'movie-1', 5)
	`)
	require.NoError(t, err)

	// Increment use count
	err = repo.IncrementUseCount(ctx, "count-test")
	require.NoError(t, err)

	// Verify it was incremented
	found, err := repo.FindByID(ctx, "count-test")
	require.NoError(t, err)
	require.NotNil(t, found)

	assert.Equal(t, 6, found.UseCount)
	assert.NotNil(t, found.LastUsedAt)
}

func TestLearningRepository_Count(t *testing.T) {
	db := setupLearningTestDB(t)
	defer db.Close()

	repo := NewLearningRepository(db)
	ctx := context.Background()

	// Initially empty
	count, err := repo.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, count)

	// Insert test data
	_, err = db.Exec(`
		INSERT INTO filename_mappings (id, pattern, pattern_type, metadata_type, metadata_id)
		VALUES
			('count-1', 'Pattern 1', 'fansub', 'series', 'series-1'),
			('count-2', 'Pattern 2', 'exact', 'movie', 'movie-1')
	`)
	require.NoError(t, err)

	// Now should be 2
	count, err = repo.Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestLearningRepository_Update(t *testing.T) {
	db := setupLearningTestDB(t)
	defer db.Close()

	repo := NewLearningRepository(db)
	ctx := context.Background()

	// Insert test data
	_, err := db.Exec(`
		INSERT INTO filename_mappings (id, pattern, pattern_type, metadata_type, metadata_id, confidence)
		VALUES ('update-test', 'Original Pattern', 'exact', 'movie', 'movie-1', 0.5)
	`)
	require.NoError(t, err)

	// Get and update
	mapping, err := repo.FindByID(ctx, "update-test")
	require.NoError(t, err)
	require.NotNil(t, mapping)

	mapping.Confidence = 0.9
	mapping.TitlePattern = "Updated Title"

	err = repo.Update(ctx, mapping)
	require.NoError(t, err)

	// Verify update
	updated, err := repo.FindByID(ctx, "update-test")
	require.NoError(t, err)
	require.NotNil(t, updated)

	assert.Equal(t, 0.9, updated.Confidence)
	assert.Equal(t, "Updated Title", updated.TitlePattern)
}
