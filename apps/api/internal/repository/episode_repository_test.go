package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/vido/api/internal/models"
	_ "modernc.org/sqlite"
)

// setupEpisodeTestDB creates an in-memory database with episodes and series tables
func setupEpisodeTestDB(t *testing.T) *sql.DB {
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

	// Create episodes table
	_, err = db.Exec(`
		CREATE TABLE episodes (
			id TEXT PRIMARY KEY,
			series_id TEXT NOT NULL,
			tmdb_id INTEGER,
			season_number INTEGER NOT NULL,
			episode_number INTEGER NOT NULL,
			title TEXT,
			overview TEXT,
			air_date TEXT,
			runtime INTEGER,
			still_path TEXT,
			vote_average REAL,
			file_path TEXT,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create episodes table: %v", err)
	}

	// Create indexes
	_, err = db.Exec(`CREATE INDEX idx_episodes_series_id ON episodes(series_id)`)
	if err != nil {
		t.Fatalf("Failed to create series_id index: %v", err)
	}

	_, err = db.Exec(`CREATE INDEX idx_episodes_season ON episodes(series_id, season_number)`)
	if err != nil {
		t.Fatalf("Failed to create season index: %v", err)
	}

	_, err = db.Exec(`CREATE UNIQUE INDEX idx_episodes_unique ON episodes(series_id, season_number, episode_number)`)
	if err != nil {
		t.Fatalf("Failed to create unique index: %v", err)
	}

	return db
}

// createTestSeries creates a series for episode tests
func createTestSeries(t *testing.T, db *sql.DB, id string) {
	_, err := db.Exec(`
		INSERT INTO series (id, title, first_air_date, genres, created_at, updated_at)
		VALUES (?, ?, ?, '[]', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, id, "Test Series", "2020-01-01")
	if err != nil {
		t.Fatalf("Failed to create test series: %v", err)
	}
}

// TestEpisodeCreate verifies episode creation
func TestEpisodeCreate(t *testing.T) {
	db := setupEpisodeTestDB(t)
	defer db.Close()

	createTestSeries(t, db, "series-1")

	repo := NewEpisodeRepository(db)
	ctx := context.Background()

	episode := &models.Episode{
		ID:            "episode-1",
		SeriesID:      "series-1",
		TMDbID:        sql.NullInt64{Int64: 62085, Valid: true},
		SeasonNumber:  1,
		EpisodeNumber: 1,
		Title:         sql.NullString{String: "Pilot", Valid: true},
		Overview:      sql.NullString{String: "The first episode of the series.", Valid: true},
		AirDate:       sql.NullString{String: "2020-01-15", Valid: true},
		Runtime:       sql.NullInt64{Int64: 45, Valid: true},
		StillPath:     sql.NullString{String: "/path/to/still.jpg", Valid: true},
		VoteAverage:   sql.NullFloat64{Float64: 8.5, Valid: true},
		FilePath:      sql.NullString{String: "/media/series/s01e01.mkv", Valid: true},
	}

	err := repo.Create(ctx, episode)
	if err != nil {
		t.Fatalf("Failed to create episode: %v", err)
	}

	// Verify timestamps were set
	if episode.CreatedAt.IsZero() {
		t.Error("Expected CreatedAt to be set")
	}
	if episode.UpdatedAt.IsZero() {
		t.Error("Expected UpdatedAt to be set")
	}

	// Verify episode was inserted
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM episodes WHERE id = ?", "episode-1").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count episodes: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 episode, got %d", count)
	}
}

// TestEpisodeCreateNil verifies nil episode rejection
func TestEpisodeCreateNil(t *testing.T) {
	db := setupEpisodeTestDB(t)
	defer db.Close()

	repo := NewEpisodeRepository(db)
	ctx := context.Background()

	err := repo.Create(ctx, nil)
	if err == nil {
		t.Fatal("Expected error for nil episode, got nil")
	}
}

// TestEpisodeFindByID verifies finding episode by ID
func TestEpisodeFindByID(t *testing.T) {
	db := setupEpisodeTestDB(t)
	defer db.Close()

	createTestSeries(t, db, "series-1")

	repo := NewEpisodeRepository(db)
	ctx := context.Background()

	// Create episode
	episode := &models.Episode{
		ID:            "episode-1",
		SeriesID:      "series-1",
		SeasonNumber:  1,
		EpisodeNumber: 1,
		Title:         sql.NullString{String: "Test Episode", Valid: true},
	}

	err := repo.Create(ctx, episode)
	if err != nil {
		t.Fatalf("Failed to create episode: %v", err)
	}

	// Find episode
	found, err := repo.FindByID(ctx, "episode-1")
	if err != nil {
		t.Fatalf("Failed to find episode: %v", err)
	}

	if found.ID != episode.ID {
		t.Errorf("Expected ID %s, got %s", episode.ID, found.ID)
	}
	if found.SeriesID != episode.SeriesID {
		t.Errorf("Expected SeriesID %s, got %s", episode.SeriesID, found.SeriesID)
	}
	if found.Title.String != episode.Title.String {
		t.Errorf("Expected title %s, got %s", episode.Title.String, found.Title.String)
	}
}

// TestEpisodeFindByIDNotFound verifies error for non-existent episode
func TestEpisodeFindByIDNotFound(t *testing.T) {
	db := setupEpisodeTestDB(t)
	defer db.Close()

	repo := NewEpisodeRepository(db)
	ctx := context.Background()

	_, err := repo.FindByID(ctx, "non-existent")
	if err == nil {
		t.Fatal("Expected error for non-existent episode, got nil")
	}
}

// TestEpisodeFindBySeriesID verifies finding all episodes for a series
func TestEpisodeFindBySeriesID(t *testing.T) {
	db := setupEpisodeTestDB(t)
	defer db.Close()

	createTestSeries(t, db, "series-1")
	createTestSeries(t, db, "series-2")

	repo := NewEpisodeRepository(db)
	ctx := context.Background()

	// Create episodes for series-1
	for i := 1; i <= 3; i++ {
		episode := &models.Episode{
			ID:            "episode-1-" + string(rune('0'+i)),
			SeriesID:      "series-1",
			SeasonNumber:  1,
			EpisodeNumber: i,
			Title:         sql.NullString{String: "Episode " + string(rune('0'+i)), Valid: true},
		}
		if err := repo.Create(ctx, episode); err != nil {
			t.Fatalf("Failed to create episode: %v", err)
		}
	}

	// Create episode for series-2
	episode := &models.Episode{
		ID:            "episode-2-1",
		SeriesID:      "series-2",
		SeasonNumber:  1,
		EpisodeNumber: 1,
		Title:         sql.NullString{String: "Other Series Episode", Valid: true},
	}
	if err := repo.Create(ctx, episode); err != nil {
		t.Fatalf("Failed to create episode: %v", err)
	}

	// Find episodes for series-1
	episodes, err := repo.FindBySeriesID(ctx, "series-1")
	if err != nil {
		t.Fatalf("Failed to find episodes: %v", err)
	}

	if len(episodes) != 3 {
		t.Errorf("Expected 3 episodes, got %d", len(episodes))
	}

	// Verify all episodes belong to series-1
	for _, ep := range episodes {
		if ep.SeriesID != "series-1" {
			t.Errorf("Expected SeriesID series-1, got %s", ep.SeriesID)
		}
	}
}

// TestEpisodeFindBySeriesIDEmpty verifies empty result for series without episodes
func TestEpisodeFindBySeriesIDEmpty(t *testing.T) {
	db := setupEpisodeTestDB(t)
	defer db.Close()

	createTestSeries(t, db, "series-1")

	repo := NewEpisodeRepository(db)
	ctx := context.Background()

	episodes, err := repo.FindBySeriesID(ctx, "series-1")
	if err != nil {
		t.Fatalf("Failed to find episodes: %v", err)
	}

	if len(episodes) != 0 {
		t.Errorf("Expected 0 episodes, got %d", len(episodes))
	}
}

// TestEpisodeFindBySeasonNumber verifies finding episodes by season
func TestEpisodeFindBySeasonNumber(t *testing.T) {
	db := setupEpisodeTestDB(t)
	defer db.Close()

	createTestSeries(t, db, "series-1")

	repo := NewEpisodeRepository(db)
	ctx := context.Background()

	// Create episodes for different seasons
	seasonsData := []struct {
		season  int
		episode int
	}{
		{1, 1}, {1, 2}, {1, 3}, // Season 1: 3 episodes
		{2, 1}, {2, 2},         // Season 2: 2 episodes
	}

	for _, sd := range seasonsData {
		episode := &models.Episode{
			ID:            "episode-s" + string(rune('0'+sd.season)) + "e" + string(rune('0'+sd.episode)),
			SeriesID:      "series-1",
			SeasonNumber:  sd.season,
			EpisodeNumber: sd.episode,
		}
		if err := repo.Create(ctx, episode); err != nil {
			t.Fatalf("Failed to create episode: %v", err)
		}
	}

	// Find season 1 episodes
	s1Episodes, err := repo.FindBySeasonNumber(ctx, "series-1", 1)
	if err != nil {
		t.Fatalf("Failed to find season 1 episodes: %v", err)
	}

	if len(s1Episodes) != 3 {
		t.Errorf("Expected 3 episodes for season 1, got %d", len(s1Episodes))
	}

	// Find season 2 episodes
	s2Episodes, err := repo.FindBySeasonNumber(ctx, "series-1", 2)
	if err != nil {
		t.Fatalf("Failed to find season 2 episodes: %v", err)
	}

	if len(s2Episodes) != 2 {
		t.Errorf("Expected 2 episodes for season 2, got %d", len(s2Episodes))
	}
}

// TestEpisodeFindBySeasonNumberOrdering verifies episodes are ordered by episode number
func TestEpisodeFindBySeasonNumberOrdering(t *testing.T) {
	db := setupEpisodeTestDB(t)
	defer db.Close()

	createTestSeries(t, db, "series-1")

	repo := NewEpisodeRepository(db)
	ctx := context.Background()

	// Create episodes out of order
	for _, epNum := range []int{3, 1, 2} {
		episode := &models.Episode{
			ID:            "episode-" + string(rune('0'+epNum)),
			SeriesID:      "series-1",
			SeasonNumber:  1,
			EpisodeNumber: epNum,
		}
		if err := repo.Create(ctx, episode); err != nil {
			t.Fatalf("Failed to create episode: %v", err)
		}
	}

	// Find and verify ordering
	episodes, err := repo.FindBySeasonNumber(ctx, "series-1", 1)
	if err != nil {
		t.Fatalf("Failed to find episodes: %v", err)
	}

	// Verify ordering
	for i, ep := range episodes {
		expectedEpNum := i + 1
		if ep.EpisodeNumber != expectedEpNum {
			t.Errorf("Expected episode %d at position %d, got episode %d", expectedEpNum, i, ep.EpisodeNumber)
		}
	}
}

// TestEpisodeFindBySeriesSeasonEpisode verifies finding specific episode
func TestEpisodeFindBySeriesSeasonEpisode(t *testing.T) {
	db := setupEpisodeTestDB(t)
	defer db.Close()

	createTestSeries(t, db, "series-1")

	repo := NewEpisodeRepository(db)
	ctx := context.Background()

	// Create episode
	episode := &models.Episode{
		ID:            "episode-1",
		SeriesID:      "series-1",
		SeasonNumber:  2,
		EpisodeNumber: 5,
		Title:         sql.NullString{String: "S02E05", Valid: true},
	}

	err := repo.Create(ctx, episode)
	if err != nil {
		t.Fatalf("Failed to create episode: %v", err)
	}

	// Find specific episode
	found, err := repo.FindBySeriesSeasonEpisode(ctx, "series-1", 2, 5)
	if err != nil {
		t.Fatalf("Failed to find episode: %v", err)
	}

	if found.ID != episode.ID {
		t.Errorf("Expected ID %s, got %s", episode.ID, found.ID)
	}
	if found.SeasonNumber != 2 {
		t.Errorf("Expected season 2, got %d", found.SeasonNumber)
	}
	if found.EpisodeNumber != 5 {
		t.Errorf("Expected episode 5, got %d", found.EpisodeNumber)
	}
}

// TestEpisodeFindBySeriesSeasonEpisodeNotFound verifies error for non-existent episode
func TestEpisodeFindBySeriesSeasonEpisodeNotFound(t *testing.T) {
	db := setupEpisodeTestDB(t)
	defer db.Close()

	createTestSeries(t, db, "series-1")

	repo := NewEpisodeRepository(db)
	ctx := context.Background()

	_, err := repo.FindBySeriesSeasonEpisode(ctx, "series-1", 1, 1)
	if err == nil {
		t.Fatal("Expected error for non-existent episode, got nil")
	}
}

// TestEpisodeUpdate verifies episode update
func TestEpisodeUpdate(t *testing.T) {
	db := setupEpisodeTestDB(t)
	defer db.Close()

	createTestSeries(t, db, "series-1")

	repo := NewEpisodeRepository(db)
	ctx := context.Background()

	// Create episode
	episode := &models.Episode{
		ID:            "episode-1",
		SeriesID:      "series-1",
		SeasonNumber:  1,
		EpisodeNumber: 1,
		Title:         sql.NullString{String: "Original Title", Valid: true},
		Runtime:       sql.NullInt64{Int64: 30, Valid: true},
	}

	err := repo.Create(ctx, episode)
	if err != nil {
		t.Fatalf("Failed to create episode: %v", err)
	}

	// Wait a bit to ensure updated_at changes
	time.Sleep(10 * time.Millisecond)

	// Update episode
	episode.Title = sql.NullString{String: "Updated Title", Valid: true}
	episode.Runtime = sql.NullInt64{Int64: 45, Valid: true}
	originalUpdatedAt := episode.UpdatedAt

	err = repo.Update(ctx, episode)
	if err != nil {
		t.Fatalf("Failed to update episode: %v", err)
	}

	// Verify updated_at changed
	if !episode.UpdatedAt.After(originalUpdatedAt) {
		t.Error("Expected UpdatedAt to be updated")
	}

	// Find and verify update
	found, err := repo.FindByID(ctx, "episode-1")
	if err != nil {
		t.Fatalf("Failed to find episode: %v", err)
	}

	if found.Title.String != "Updated Title" {
		t.Errorf("Expected title 'Updated Title', got '%s'", found.Title.String)
	}
	if found.Runtime.Int64 != 45 {
		t.Errorf("Expected runtime 45, got %d", found.Runtime.Int64)
	}
}

// TestEpisodeUpdateNil verifies nil episode rejection
func TestEpisodeUpdateNil(t *testing.T) {
	db := setupEpisodeTestDB(t)
	defer db.Close()

	repo := NewEpisodeRepository(db)
	ctx := context.Background()

	err := repo.Update(ctx, nil)
	if err == nil {
		t.Fatal("Expected error for nil episode, got nil")
	}
}

// TestEpisodeUpdateNotFound verifies error for non-existent episode
func TestEpisodeUpdateNotFound(t *testing.T) {
	db := setupEpisodeTestDB(t)
	defer db.Close()

	repo := NewEpisodeRepository(db)
	ctx := context.Background()

	episode := &models.Episode{
		ID:            "non-existent",
		SeriesID:      "series-1",
		SeasonNumber:  1,
		EpisodeNumber: 1,
	}

	err := repo.Update(ctx, episode)
	if err == nil {
		t.Fatal("Expected error for non-existent episode, got nil")
	}
}

// TestEpisodeDelete verifies episode deletion
func TestEpisodeDelete(t *testing.T) {
	db := setupEpisodeTestDB(t)
	defer db.Close()

	createTestSeries(t, db, "series-1")

	repo := NewEpisodeRepository(db)
	ctx := context.Background()

	// Create episode
	episode := &models.Episode{
		ID:            "episode-1",
		SeriesID:      "series-1",
		SeasonNumber:  1,
		EpisodeNumber: 1,
	}

	err := repo.Create(ctx, episode)
	if err != nil {
		t.Fatalf("Failed to create episode: %v", err)
	}

	// Delete episode
	err = repo.Delete(ctx, "episode-1")
	if err != nil {
		t.Fatalf("Failed to delete episode: %v", err)
	}

	// Verify episode was deleted
	_, err = repo.FindByID(ctx, "episode-1")
	if err == nil {
		t.Fatal("Expected error for deleted episode, got nil")
	}
}

// TestEpisodeDeleteNotFound verifies error for non-existent episode
func TestEpisodeDeleteNotFound(t *testing.T) {
	db := setupEpisodeTestDB(t)
	defer db.Close()

	repo := NewEpisodeRepository(db)
	ctx := context.Background()

	err := repo.Delete(ctx, "non-existent")
	if err == nil {
		t.Fatal("Expected error for non-existent episode, got nil")
	}
}

// TestEpisodeUpsertCreate verifies upsert creates new episode
func TestEpisodeUpsertCreate(t *testing.T) {
	db := setupEpisodeTestDB(t)
	defer db.Close()

	createTestSeries(t, db, "series-1")

	repo := NewEpisodeRepository(db)
	ctx := context.Background()

	episode := &models.Episode{
		ID:            "episode-1",
		SeriesID:      "series-1",
		SeasonNumber:  1,
		EpisodeNumber: 1,
		Title:         sql.NullString{String: "New Episode", Valid: true},
	}

	err := repo.Upsert(ctx, episode)
	if err != nil {
		t.Fatalf("Failed to upsert episode: %v", err)
	}

	// Verify episode was created
	found, err := repo.FindByID(ctx, "episode-1")
	if err != nil {
		t.Fatalf("Failed to find episode: %v", err)
	}

	if found.Title.String != "New Episode" {
		t.Errorf("Expected title 'New Episode', got '%s'", found.Title.String)
	}
}

// TestEpisodeUpsertUpdate verifies upsert updates existing episode
func TestEpisodeUpsertUpdate(t *testing.T) {
	db := setupEpisodeTestDB(t)
	defer db.Close()

	createTestSeries(t, db, "series-1")

	repo := NewEpisodeRepository(db)
	ctx := context.Background()

	// Create initial episode
	episode := &models.Episode{
		ID:            "episode-1",
		SeriesID:      "series-1",
		SeasonNumber:  1,
		EpisodeNumber: 1,
		Title:         sql.NullString{String: "Original Title", Valid: true},
	}

	err := repo.Create(ctx, episode)
	if err != nil {
		t.Fatalf("Failed to create episode: %v", err)
	}

	// Upsert with same series/season/episode but different ID
	updatedEpisode := &models.Episode{
		ID:            "episode-new",
		SeriesID:      "series-1",
		SeasonNumber:  1,
		EpisodeNumber: 1,
		Title:         sql.NullString{String: "Updated Title", Valid: true},
	}

	err = repo.Upsert(ctx, updatedEpisode)
	if err != nil {
		t.Fatalf("Failed to upsert episode: %v", err)
	}

	// Verify episode was updated with original ID
	found, err := repo.FindBySeriesSeasonEpisode(ctx, "series-1", 1, 1)
	if err != nil {
		t.Fatalf("Failed to find episode: %v", err)
	}

	if found.ID != "episode-1" {
		t.Errorf("Expected ID 'episode-1', got '%s'", found.ID)
	}
	if found.Title.String != "Updated Title" {
		t.Errorf("Expected title 'Updated Title', got '%s'", found.Title.String)
	}
}

// TestEpisodeUpsertNil verifies nil episode rejection
func TestEpisodeUpsertNil(t *testing.T) {
	db := setupEpisodeTestDB(t)
	defer db.Close()

	repo := NewEpisodeRepository(db)
	ctx := context.Background()

	err := repo.Upsert(ctx, nil)
	if err == nil {
		t.Fatal("Expected error for nil episode, got nil")
	}
}

// TestEpisodeFindBySeriesIDOrdering verifies episodes are ordered by season and episode number
func TestEpisodeFindBySeriesIDOrdering(t *testing.T) {
	db := setupEpisodeTestDB(t)
	defer db.Close()

	createTestSeries(t, db, "series-1")

	repo := NewEpisodeRepository(db)
	ctx := context.Background()

	// Create episodes out of order
	episodesData := []struct {
		season  int
		episode int
	}{
		{2, 1}, {1, 2}, {1, 1}, {2, 2},
	}

	for _, ed := range episodesData {
		episode := &models.Episode{
			ID:            "episode-s" + string(rune('0'+ed.season)) + "e" + string(rune('0'+ed.episode)),
			SeriesID:      "series-1",
			SeasonNumber:  ed.season,
			EpisodeNumber: ed.episode,
		}
		if err := repo.Create(ctx, episode); err != nil {
			t.Fatalf("Failed to create episode: %v", err)
		}
	}

	// Find all episodes
	episodes, err := repo.FindBySeriesID(ctx, "series-1")
	if err != nil {
		t.Fatalf("Failed to find episodes: %v", err)
	}

	// Expected order: S1E1, S1E2, S2E1, S2E2
	expectedOrder := []struct {
		season  int
		episode int
	}{
		{1, 1}, {1, 2}, {2, 1}, {2, 2},
	}

	for i, ep := range episodes {
		if ep.SeasonNumber != expectedOrder[i].season || ep.EpisodeNumber != expectedOrder[i].episode {
			t.Errorf("Expected S%dE%d at position %d, got S%dE%d",
				expectedOrder[i].season, expectedOrder[i].episode, i,
				ep.SeasonNumber, ep.EpisodeNumber)
		}
	}
}

// TestEpisodeAllFieldsPersistence verifies all fields are correctly persisted
func TestEpisodeAllFieldsPersistence(t *testing.T) {
	db := setupEpisodeTestDB(t)
	defer db.Close()

	createTestSeries(t, db, "series-1")

	repo := NewEpisodeRepository(db)
	ctx := context.Background()

	// Create episode with all fields
	episode := &models.Episode{
		ID:            "episode-1",
		SeriesID:      "series-1",
		TMDbID:        sql.NullInt64{Int64: 12345, Valid: true},
		SeasonNumber:  3,
		EpisodeNumber: 7,
		Title:         sql.NullString{String: "Full Episode", Valid: true},
		Overview:      sql.NullString{String: "A detailed overview of the episode.", Valid: true},
		AirDate:       sql.NullString{String: "2023-06-15", Valid: true},
		Runtime:       sql.NullInt64{Int64: 52, Valid: true},
		StillPath:     sql.NullString{String: "/stills/ep307.jpg", Valid: true},
		VoteAverage:   sql.NullFloat64{Float64: 8.7, Valid: true},
		FilePath:      sql.NullString{String: "/media/series/S03E07.mkv", Valid: true},
	}

	err := repo.Create(ctx, episode)
	if err != nil {
		t.Fatalf("Failed to create episode: %v", err)
	}

	// Retrieve and verify all fields
	found, err := repo.FindByID(ctx, "episode-1")
	if err != nil {
		t.Fatalf("Failed to find episode: %v", err)
	}

	if found.TMDbID.Int64 != 12345 {
		t.Errorf("Expected TMDbID 12345, got %d", found.TMDbID.Int64)
	}
	if found.SeasonNumber != 3 {
		t.Errorf("Expected season 3, got %d", found.SeasonNumber)
	}
	if found.EpisodeNumber != 7 {
		t.Errorf("Expected episode 7, got %d", found.EpisodeNumber)
	}
	if found.Title.String != "Full Episode" {
		t.Errorf("Expected title 'Full Episode', got '%s'", found.Title.String)
	}
	if found.Overview.String != "A detailed overview of the episode." {
		t.Errorf("Expected overview to match, got '%s'", found.Overview.String)
	}
	if found.AirDate.String != "2023-06-15" {
		t.Errorf("Expected air date '2023-06-15', got '%s'", found.AirDate.String)
	}
	if found.Runtime.Int64 != 52 {
		t.Errorf("Expected runtime 52, got %d", found.Runtime.Int64)
	}
	if found.StillPath.String != "/stills/ep307.jpg" {
		t.Errorf("Expected still path '/stills/ep307.jpg', got '%s'", found.StillPath.String)
	}
	if found.VoteAverage.Float64 != 8.7 {
		t.Errorf("Expected vote average 8.7, got %f", found.VoteAverage.Float64)
	}
	if found.FilePath.String != "/media/series/S03E07.mkv" {
		t.Errorf("Expected file path '/media/series/S03E07.mkv', got '%s'", found.FilePath.String)
	}
}

// TestEpisodeNullFields verifies handling of null/empty fields
func TestEpisodeNullFields(t *testing.T) {
	db := setupEpisodeTestDB(t)
	defer db.Close()

	createTestSeries(t, db, "series-1")

	repo := NewEpisodeRepository(db)
	ctx := context.Background()

	// Create episode with minimal fields (others null)
	episode := &models.Episode{
		ID:            "episode-1",
		SeriesID:      "series-1",
		SeasonNumber:  1,
		EpisodeNumber: 1,
		// All other fields are default (null)
	}

	err := repo.Create(ctx, episode)
	if err != nil {
		t.Fatalf("Failed to create episode: %v", err)
	}

	// Retrieve and verify null fields
	found, err := repo.FindByID(ctx, "episode-1")
	if err != nil {
		t.Fatalf("Failed to find episode: %v", err)
	}

	if found.TMDbID.Valid {
		t.Error("Expected TMDbID to be invalid (null)")
	}
	if found.Title.Valid {
		t.Error("Expected Title to be invalid (null)")
	}
	if found.Overview.Valid {
		t.Error("Expected Overview to be invalid (null)")
	}
	if found.Runtime.Valid {
		t.Error("Expected Runtime to be invalid (null)")
	}
}
