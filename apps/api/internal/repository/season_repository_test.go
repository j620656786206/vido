package repository

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/models"
	_ "modernc.org/sqlite"
)

// setupSeasonTestDB creates an in-memory database with series and seasons tables
func setupSeasonTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Create series table (for foreign key reference)
	_, err = db.Exec(`
		CREATE TABLE series (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			original_title TEXT,
			first_air_date TEXT,
			last_air_date TEXT,
			genres TEXT NOT NULL DEFAULT '[]',
			rating REAL,
			overview TEXT,
			poster_path TEXT,
			backdrop_path TEXT,
			number_of_seasons INTEGER,
			number_of_episodes INTEGER,
			status TEXT,
			original_language TEXT,
			imdb_id TEXT,
			tmdb_id INTEGER,
			in_production INTEGER,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create series table: %v", err)
	}

	// Create seasons table
	_, err = db.Exec(`
		CREATE TABLE seasons (
			id TEXT PRIMARY KEY,
			series_id TEXT NOT NULL,
			tmdb_id INTEGER,
			season_number INTEGER NOT NULL,
			name TEXT,
			overview TEXT,
			poster_path TEXT,
			air_date TEXT,
			episode_count INTEGER,
			vote_average REAL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (series_id) REFERENCES series(id) ON DELETE CASCADE,
			UNIQUE(series_id, season_number)
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create seasons table: %v", err)
	}

	// Create indexes
	_, err = db.Exec(`CREATE INDEX idx_seasons_series_id ON seasons(series_id)`)
	if err != nil {
		t.Fatalf("Failed to create series_id index: %v", err)
	}

	_, err = db.Exec(`CREATE INDEX idx_seasons_tmdb_id ON seasons(tmdb_id)`)
	if err != nil {
		t.Fatalf("Failed to create tmdb_id index: %v", err)
	}

	return db
}

// createTestSeriesForSeason creates a series for season tests
func createTestSeriesForSeason(t *testing.T, db *sql.DB, id string) {
	_, err := db.Exec(`
		INSERT INTO series (id, title, first_air_date, genres, created_at, updated_at)
		VALUES (?, ?, ?, '[]', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, id, "Test Series", "2020-01-01")
	if err != nil {
		t.Fatalf("Failed to create test series: %v", err)
	}
}

// --- CRUD Tests ---

func TestSeasonCreate(t *testing.T) {
	db := setupSeasonTestDB(t)
	defer db.Close()

	createTestSeriesForSeason(t, db, "series-1")

	repo := NewSeasonRepository(db)
	ctx := context.Background()

	season := &models.Season{
		ID:           "season-1",
		SeriesID:     "series-1",
		TMDbID:       sql.NullInt64{Int64: 12345, Valid: true},
		SeasonNumber: 1,
		Name:         sql.NullString{String: "Season 1", Valid: true},
		Overview:     sql.NullString{String: "The first season", Valid: true},
		PosterPath:   sql.NullString{String: "/posters/s1.jpg", Valid: true},
		AirDate:      sql.NullString{String: "2020-01-15", Valid: true},
		EpisodeCount: sql.NullInt64{Int64: 10, Valid: true},
		VoteAverage:  sql.NullFloat64{Float64: 8.5, Valid: true},
	}

	err := repo.Create(ctx, season)
	require.NoError(t, err)

	// Verify timestamps were set
	assert.False(t, season.CreatedAt.IsZero())
	assert.False(t, season.UpdatedAt.IsZero())

	// Verify season was inserted
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM seasons WHERE id = ?", "season-1").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestSeasonCreateNil(t *testing.T) {
	db := setupSeasonTestDB(t)
	defer db.Close()

	repo := NewSeasonRepository(db)
	err := repo.Create(context.Background(), nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "season cannot be nil")
}

func TestSeasonFindByID(t *testing.T) {
	db := setupSeasonTestDB(t)
	defer db.Close()

	createTestSeriesForSeason(t, db, "series-1")

	repo := NewSeasonRepository(db)
	ctx := context.Background()

	original := &models.Season{
		ID:           "season-1",
		SeriesID:     "series-1",
		TMDbID:       sql.NullInt64{Int64: 12345, Valid: true},
		SeasonNumber: 1,
		Name:         sql.NullString{String: "Season 1", Valid: true},
		Overview:     sql.NullString{String: "The first season", Valid: true},
		PosterPath:   sql.NullString{String: "/posters/s1.jpg", Valid: true},
		AirDate:      sql.NullString{String: "2020-01-15", Valid: true},
		EpisodeCount: sql.NullInt64{Int64: 10, Valid: true},
		VoteAverage:  sql.NullFloat64{Float64: 8.5, Valid: true},
	}

	err := repo.Create(ctx, original)
	require.NoError(t, err)

	found, err := repo.FindByID(ctx, "season-1")
	require.NoError(t, err)
	require.NotNil(t, found)

	assert.Equal(t, "season-1", found.ID)
	assert.Equal(t, "series-1", found.SeriesID)
	assert.Equal(t, int64(12345), found.TMDbID.Int64)
	assert.Equal(t, 1, found.SeasonNumber)
	assert.Equal(t, "Season 1", found.Name.String)
	assert.Equal(t, "The first season", found.Overview.String)
	assert.Equal(t, "/posters/s1.jpg", found.PosterPath.String)
	assert.Equal(t, "2020-01-15", found.AirDate.String)
	assert.Equal(t, int64(10), found.EpisodeCount.Int64)
	assert.Equal(t, 8.5, found.VoteAverage.Float64)
}

func TestSeasonFindByIDNotFound(t *testing.T) {
	db := setupSeasonTestDB(t)
	defer db.Close()

	repo := NewSeasonRepository(db)
	_, err := repo.FindByID(context.Background(), "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestSeasonFindBySeriesID(t *testing.T) {
	db := setupSeasonTestDB(t)
	defer db.Close()

	createTestSeriesForSeason(t, db, "series-1")

	repo := NewSeasonRepository(db)
	ctx := context.Background()

	// Create 3 seasons out of order
	for _, s := range []struct {
		id     string
		num    int
		name   string
	}{
		{"season-3", 3, "Season 3"},
		{"season-1", 1, "Season 1"},
		{"season-2", 2, "Season 2"},
	} {
		err := repo.Create(ctx, &models.Season{
			ID:           s.id,
			SeriesID:     "series-1",
			SeasonNumber: s.num,
			Name:         sql.NullString{String: s.name, Valid: true},
		})
		require.NoError(t, err)
	}

	seasons, err := repo.FindBySeriesID(ctx, "series-1")
	require.NoError(t, err)
	assert.Len(t, seasons, 3)

	// Verify ordering by season_number
	assert.Equal(t, 1, seasons[0].SeasonNumber)
	assert.Equal(t, 2, seasons[1].SeasonNumber)
	assert.Equal(t, 3, seasons[2].SeasonNumber)
}

func TestSeasonFindBySeriesIDEmpty(t *testing.T) {
	db := setupSeasonTestDB(t)
	defer db.Close()

	repo := NewSeasonRepository(db)
	seasons, err := repo.FindBySeriesID(context.Background(), "nonexistent-series")
	require.NoError(t, err)
	assert.Empty(t, seasons)
}

func TestSeasonFindBySeriesAndNumber(t *testing.T) {
	db := setupSeasonTestDB(t)
	defer db.Close()

	createTestSeriesForSeason(t, db, "series-1")

	repo := NewSeasonRepository(db)
	ctx := context.Background()

	err := repo.Create(ctx, &models.Season{
		ID:           "season-1",
		SeriesID:     "series-1",
		SeasonNumber: 1,
		Name:         sql.NullString{String: "Season 1", Valid: true},
	})
	require.NoError(t, err)

	err = repo.Create(ctx, &models.Season{
		ID:           "season-2",
		SeriesID:     "series-1",
		SeasonNumber: 2,
		Name:         sql.NullString{String: "Season 2", Valid: true},
	})
	require.NoError(t, err)

	found, err := repo.FindBySeriesAndNumber(ctx, "series-1", 2)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, "season-2", found.ID)
	assert.Equal(t, 2, found.SeasonNumber)
	assert.Equal(t, "Season 2", found.Name.String)
}

func TestSeasonFindBySeriesAndNumberNotFound(t *testing.T) {
	db := setupSeasonTestDB(t)
	defer db.Close()

	createTestSeriesForSeason(t, db, "series-1")

	repo := NewSeasonRepository(db)
	_, err := repo.FindBySeriesAndNumber(context.Background(), "series-1", 99)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestSeasonUpdate(t *testing.T) {
	db := setupSeasonTestDB(t)
	defer db.Close()

	createTestSeriesForSeason(t, db, "series-1")

	repo := NewSeasonRepository(db)
	ctx := context.Background()

	season := &models.Season{
		ID:           "season-1",
		SeriesID:     "series-1",
		SeasonNumber: 1,
		Name:         sql.NullString{String: "Season 1", Valid: true},
	}

	err := repo.Create(ctx, season)
	require.NoError(t, err)

	originalUpdatedAt := season.UpdatedAt

	// Update the season
	season.Name = sql.NullString{String: "Season 1 - Updated", Valid: true}
	season.Overview = sql.NullString{String: "Updated overview", Valid: true}
	season.EpisodeCount = sql.NullInt64{Int64: 12, Valid: true}

	err = repo.Update(ctx, season)
	require.NoError(t, err)

	// Verify update
	found, err := repo.FindByID(ctx, "season-1")
	require.NoError(t, err)
	assert.Equal(t, "Season 1 - Updated", found.Name.String)
	assert.Equal(t, "Updated overview", found.Overview.String)
	assert.Equal(t, int64(12), found.EpisodeCount.Int64)
	assert.True(t, found.UpdatedAt.After(originalUpdatedAt) || found.UpdatedAt.Equal(originalUpdatedAt))
}

func TestSeasonUpdateNil(t *testing.T) {
	db := setupSeasonTestDB(t)
	defer db.Close()

	repo := NewSeasonRepository(db)
	err := repo.Update(context.Background(), nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "season cannot be nil")
}

func TestSeasonUpdateNotFound(t *testing.T) {
	db := setupSeasonTestDB(t)
	defer db.Close()

	repo := NewSeasonRepository(db)
	err := repo.Update(context.Background(), &models.Season{
		ID:           "nonexistent",
		SeriesID:     "series-1",
		SeasonNumber: 1,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestSeasonDelete(t *testing.T) {
	db := setupSeasonTestDB(t)
	defer db.Close()

	createTestSeriesForSeason(t, db, "series-1")

	repo := NewSeasonRepository(db)
	ctx := context.Background()

	err := repo.Create(ctx, &models.Season{
		ID:           "season-1",
		SeriesID:     "series-1",
		SeasonNumber: 1,
	})
	require.NoError(t, err)

	err = repo.Delete(ctx, "season-1")
	require.NoError(t, err)

	_, err = repo.FindByID(ctx, "season-1")
	assert.Error(t, err)
}

func TestSeasonDeleteNotFound(t *testing.T) {
	db := setupSeasonTestDB(t)
	defer db.Close()

	repo := NewSeasonRepository(db)
	err := repo.Delete(context.Background(), "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// --- Upsert Tests ---

func TestSeasonUpsertCreate(t *testing.T) {
	db := setupSeasonTestDB(t)
	defer db.Close()

	createTestSeriesForSeason(t, db, "series-1")

	repo := NewSeasonRepository(db)
	ctx := context.Background()

	season := &models.Season{
		ID:           "season-new",
		SeriesID:     "series-1",
		SeasonNumber: 1,
		Name:         sql.NullString{String: "Season 1", Valid: true},
	}

	err := repo.Upsert(ctx, season)
	require.NoError(t, err)

	// Verify it was created
	found, err := repo.FindBySeriesAndNumber(ctx, "series-1", 1)
	require.NoError(t, err)
	assert.Equal(t, "season-new", found.ID)
	assert.Equal(t, "Season 1", found.Name.String)
}

func TestSeasonUpsertUpdate(t *testing.T) {
	db := setupSeasonTestDB(t)
	defer db.Close()

	createTestSeriesForSeason(t, db, "series-1")

	repo := NewSeasonRepository(db)
	ctx := context.Background()

	// Create original
	original := &models.Season{
		ID:           "season-original",
		SeriesID:     "series-1",
		SeasonNumber: 1,
		Name:         sql.NullString{String: "Season 1", Valid: true},
	}
	err := repo.Create(ctx, original)
	require.NoError(t, err)

	// Upsert with same series+number but different ID
	updated := &models.Season{
		ID:           "season-should-be-overridden",
		SeriesID:     "series-1",
		SeasonNumber: 1,
		Name:         sql.NullString{String: "Season 1 - Updated via Upsert", Valid: true},
		Overview:     sql.NullString{String: "New overview", Valid: true},
	}
	err = repo.Upsert(ctx, updated)
	require.NoError(t, err)

	// Verify original ID was preserved
	assert.Equal(t, "season-original", updated.ID)

	// Verify data was updated
	found, err := repo.FindByID(ctx, "season-original")
	require.NoError(t, err)
	assert.Equal(t, "Season 1 - Updated via Upsert", found.Name.String)
	assert.Equal(t, "New overview", found.Overview.String)
}

func TestSeasonUpsertNil(t *testing.T) {
	db := setupSeasonTestDB(t)
	defer db.Close()

	repo := NewSeasonRepository(db)
	err := repo.Upsert(context.Background(), nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "season cannot be nil")
}

// --- Advanced Tests ---

func TestSeasonAllFieldsPersistence(t *testing.T) {
	db := setupSeasonTestDB(t)
	defer db.Close()

	createTestSeriesForSeason(t, db, "series-1")

	repo := NewSeasonRepository(db)
	ctx := context.Background()

	season := &models.Season{
		ID:           "season-full",
		SeriesID:     "series-1",
		TMDbID:       sql.NullInt64{Int64: 99999, Valid: true},
		SeasonNumber: 3,
		Name:         sql.NullString{String: "第三季", Valid: true},
		Overview:     sql.NullString{String: "精彩的第三季", Valid: true},
		PosterPath:   sql.NullString{String: "/posters/s3.jpg", Valid: true},
		AirDate:      sql.NullString{String: "2023-09-01", Valid: true},
		EpisodeCount: sql.NullInt64{Int64: 8, Valid: true},
		VoteAverage:  sql.NullFloat64{Float64: 9.1, Valid: true},
	}

	err := repo.Create(ctx, season)
	require.NoError(t, err)

	found, err := repo.FindByID(ctx, "season-full")
	require.NoError(t, err)

	assert.Equal(t, "season-full", found.ID)
	assert.Equal(t, "series-1", found.SeriesID)
	assert.Equal(t, int64(99999), found.TMDbID.Int64)
	assert.True(t, found.TMDbID.Valid)
	assert.Equal(t, 3, found.SeasonNumber)
	assert.Equal(t, "第三季", found.Name.String)
	assert.Equal(t, "精彩的第三季", found.Overview.String)
	assert.Equal(t, "/posters/s3.jpg", found.PosterPath.String)
	assert.Equal(t, "2023-09-01", found.AirDate.String)
	assert.Equal(t, int64(8), found.EpisodeCount.Int64)
	assert.Equal(t, 9.1, found.VoteAverage.Float64)
}

func TestSeasonNullFields(t *testing.T) {
	db := setupSeasonTestDB(t)
	defer db.Close()

	createTestSeriesForSeason(t, db, "series-1")

	repo := NewSeasonRepository(db)
	ctx := context.Background()

	season := &models.Season{
		ID:           "season-minimal",
		SeriesID:     "series-1",
		SeasonNumber: 0,
	}

	err := repo.Create(ctx, season)
	require.NoError(t, err)

	found, err := repo.FindByID(ctx, "season-minimal")
	require.NoError(t, err)

	assert.Equal(t, "season-minimal", found.ID)
	assert.Equal(t, 0, found.SeasonNumber)
	assert.False(t, found.TMDbID.Valid)
	assert.False(t, found.Name.Valid)
	assert.False(t, found.Overview.Valid)
	assert.False(t, found.PosterPath.Valid)
	assert.False(t, found.AirDate.Valid)
	assert.False(t, found.EpisodeCount.Valid)
	assert.False(t, found.VoteAverage.Valid)
}

func TestSeasonUniqueConstraint(t *testing.T) {
	db := setupSeasonTestDB(t)
	defer db.Close()

	createTestSeriesForSeason(t, db, "series-1")

	repo := NewSeasonRepository(db)
	ctx := context.Background()

	// Create first season
	err := repo.Create(ctx, &models.Season{
		ID:           "season-1",
		SeriesID:     "series-1",
		SeasonNumber: 1,
	})
	require.NoError(t, err)

	// Attempt duplicate (same series_id + season_number)
	err = repo.Create(ctx, &models.Season{
		ID:           "season-1-dup",
		SeriesID:     "series-1",
		SeasonNumber: 1,
	})
	assert.Error(t, err)
}

func TestSeasonFindBySeriesIDOrdering(t *testing.T) {
	db := setupSeasonTestDB(t)
	defer db.Close()

	createTestSeriesForSeason(t, db, "series-1")

	repo := NewSeasonRepository(db)
	ctx := context.Background()

	// Create seasons including season 0 (specials)
	for _, num := range []int{3, 0, 2, 1} {
		err := repo.Create(ctx, &models.Season{
			ID:           "season-" + string(rune('0'+num)),
			SeriesID:     "series-1",
			SeasonNumber: num,
		})
		require.NoError(t, err)
	}

	seasons, err := repo.FindBySeriesID(ctx, "series-1")
	require.NoError(t, err)
	assert.Len(t, seasons, 4)

	// Should be ordered 0, 1, 2, 3
	assert.Equal(t, 0, seasons[0].SeasonNumber)
	assert.Equal(t, 1, seasons[1].SeasonNumber)
	assert.Equal(t, 2, seasons[2].SeasonNumber)
	assert.Equal(t, 3, seasons[3].SeasonNumber)
}

func TestSeasonMultipleSeries(t *testing.T) {
	db := setupSeasonTestDB(t)
	defer db.Close()

	createTestSeriesForSeason(t, db, "series-A")
	createTestSeriesForSeason(t, db, "series-B")

	repo := NewSeasonRepository(db)
	ctx := context.Background()

	// Create seasons for two different series
	err := repo.Create(ctx, &models.Season{
		ID: "sA-s1", SeriesID: "series-A", SeasonNumber: 1,
	})
	require.NoError(t, err)

	err = repo.Create(ctx, &models.Season{
		ID: "sA-s2", SeriesID: "series-A", SeasonNumber: 2,
	})
	require.NoError(t, err)

	err = repo.Create(ctx, &models.Season{
		ID: "sB-s1", SeriesID: "series-B", SeasonNumber: 1,
	})
	require.NoError(t, err)

	// Verify isolation
	seasonsA, err := repo.FindBySeriesID(ctx, "series-A")
	require.NoError(t, err)
	assert.Len(t, seasonsA, 2)

	seasonsB, err := repo.FindBySeriesID(ctx, "series-B")
	require.NoError(t, err)
	assert.Len(t, seasonsB, 1)

	// Same season_number allowed across different series
	foundA, err := repo.FindBySeriesAndNumber(ctx, "series-A", 1)
	require.NoError(t, err)
	assert.Equal(t, "sA-s1", foundA.ID)

	foundB, err := repo.FindBySeriesAndNumber(ctx, "series-B", 1)
	require.NoError(t, err)
	assert.Equal(t, "sB-s1", foundB.ID)
}
