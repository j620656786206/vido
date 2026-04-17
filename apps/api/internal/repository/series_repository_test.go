package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/vido/api/internal/models"
	_ "modernc.org/sqlite"
)

// setupSeriesTestDB creates an in-memory database with series table
func setupSeriesTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Create series table
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
			file_path TEXT,
			file_size INTEGER,
			parse_status TEXT NOT NULL DEFAULT 'pending',
			metadata_source TEXT,
			subtitle_status TEXT NOT NULL DEFAULT 'not_searched',
			subtitle_path TEXT,
			subtitle_language TEXT,
			subtitle_last_searched TIMESTAMP,
			subtitle_search_score REAL,
			vote_average REAL,
			is_removed INTEGER NOT NULL DEFAULT 0,
			video_codec TEXT,
			video_resolution TEXT,
			audio_codec TEXT,
			audio_channels INTEGER,
			subtitle_tracks TEXT,
			hdr_format TEXT,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create series table: %v", err)
	}

	// Create indexes
	_, err = db.Exec(`CREATE INDEX idx_series_title ON series(title)`)
	if err != nil {
		t.Fatalf("Failed to create title index: %v", err)
	}

	_, err = db.Exec(`CREATE INDEX idx_series_tmdb_id ON series(tmdb_id)`)
	if err != nil {
		t.Fatalf("Failed to create tmdb_id index: %v", err)
	}

	_, err = db.Exec(`CREATE INDEX idx_series_imdb_id ON series(imdb_id)`)
	if err != nil {
		t.Fatalf("Failed to create imdb_id index: %v", err)
	}

	return db
}

// TestSeriesCreate verifies series creation
func TestSeriesCreate(t *testing.T) {
	db := setupSeriesTestDB(t)
	defer db.Close()

	repo := NewSeriesRepository(db)
	ctx := context.Background()

	series := &models.Series{
		ID:               "series-1",
		Title:            "Breaking Bad",
		OriginalTitle:    models.NewNullString("Breaking Bad"),
		FirstAirDate:     "2008-01-20",
		LastAirDate:      models.NewNullString("2013-09-29"),
		Genres:           []string{"Drama", "Crime", "Thriller"},
		Rating:           models.NewNullFloat64(9.5),
		Overview:         models.NewNullString("A high school chemistry teacher turned meth cook."),
		NumberOfSeasons:  models.NewNullInt64(5),
		NumberOfEpisodes: models.NewNullInt64(62),
		Status:           models.NewNullString("Ended"),
		OriginalLanguage: models.NewNullString("en"),
		IMDbID:           models.NewNullString("tt0903747"),
		TMDbID:           models.NewNullInt64(1396),
		InProduction:     models.NewNullBool(false),
	}

	err := repo.Create(ctx, series)
	if err != nil {
		t.Fatalf("Failed to create series: %v", err)
	}

	// Verify timestamps were set
	if series.CreatedAt.IsZero() {
		t.Error("Expected CreatedAt to be set")
	}
	if series.UpdatedAt.IsZero() {
		t.Error("Expected UpdatedAt to be set")
	}

	// Verify series was inserted
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM series WHERE id = ?", "series-1").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count series: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 series, got %d", count)
	}
}

// TestSeriesCreateNil verifies nil series rejection
func TestSeriesCreateNil(t *testing.T) {
	db := setupSeriesTestDB(t)
	defer db.Close()

	repo := NewSeriesRepository(db)
	ctx := context.Background()

	err := repo.Create(ctx, nil)
	if err == nil {
		t.Fatal("Expected error for nil series, got nil")
	}
}

// TestSeriesFindByID verifies finding series by ID
func TestSeriesFindByID(t *testing.T) {
	db := setupSeriesTestDB(t)
	defer db.Close()

	repo := NewSeriesRepository(db)
	ctx := context.Background()

	// Create series
	series := &models.Series{
		ID:               "series-1",
		Title:            "Game of Thrones",
		FirstAirDate:     "2011-04-17",
		Genres:           []string{"Drama", "Fantasy"},
		Rating:           models.NewNullFloat64(9.3),
		NumberOfSeasons:  models.NewNullInt64(8),
		NumberOfEpisodes: models.NewNullInt64(73),
	}

	err := repo.Create(ctx, series)
	if err != nil {
		t.Fatalf("Failed to create series: %v", err)
	}

	// Find series
	found, err := repo.FindByID(ctx, "series-1")
	if err != nil {
		t.Fatalf("Failed to find series: %v", err)
	}

	if found.ID != series.ID {
		t.Errorf("Expected ID %s, got %s", series.ID, found.ID)
	}
	if found.Title != series.Title {
		t.Errorf("Expected title %s, got %s", series.Title, found.Title)
	}
	if len(found.Genres) != len(series.Genres) {
		t.Errorf("Expected %d genres, got %d", len(series.Genres), len(found.Genres))
	}
}

// TestSeriesFindByIDNotFound verifies error for non-existent series
func TestSeriesFindByIDNotFound(t *testing.T) {
	db := setupSeriesTestDB(t)
	defer db.Close()

	repo := NewSeriesRepository(db)
	ctx := context.Background()

	_, err := repo.FindByID(ctx, "non-existent")
	if err == nil {
		t.Fatal("Expected error for non-existent series, got nil")
	}
}

// TestSeriesFindByTMDbID verifies finding series by TMDb ID
func TestSeriesFindByTMDbID(t *testing.T) {
	db := setupSeriesTestDB(t)
	defer db.Close()

	repo := NewSeriesRepository(db)
	ctx := context.Background()

	// Create series
	series := &models.Series{
		ID:           "series-1",
		Title:        "The Mandalorian",
		FirstAirDate: "2019-11-12",
		Genres:       []string{"Science Fiction", "Action"},
		TMDbID:       models.NewNullInt64(82856),
	}

	err := repo.Create(ctx, series)
	if err != nil {
		t.Fatalf("Failed to create series: %v", err)
	}

	// Find by TMDb ID
	found, err := repo.FindByTMDbID(ctx, 82856)
	if err != nil {
		t.Fatalf("Failed to find series by TMDb ID: %v", err)
	}

	if found.ID != series.ID {
		t.Errorf("Expected ID %s, got %s", series.ID, found.ID)
	}
	if found.TMDbID.Int64 != series.TMDbID.Int64 {
		t.Errorf("Expected TMDb ID %d, got %d", series.TMDbID.Int64, found.TMDbID.Int64)
	}
}

// TestSeriesFindByIMDbID verifies finding series by IMDb ID
func TestSeriesFindByIMDbID(t *testing.T) {
	db := setupSeriesTestDB(t)
	defer db.Close()

	repo := NewSeriesRepository(db)
	ctx := context.Background()

	// Create series
	series := &models.Series{
		ID:           "series-1",
		Title:        "Stranger Things",
		FirstAirDate: "2016-07-15",
		Genres:       []string{"Science Fiction", "Horror", "Drama"},
		IMDbID:       models.NewNullString("tt4574334"),
	}

	err := repo.Create(ctx, series)
	if err != nil {
		t.Fatalf("Failed to create series: %v", err)
	}

	// Find by IMDb ID
	found, err := repo.FindByIMDbID(ctx, "tt4574334")
	if err != nil {
		t.Fatalf("Failed to find series by IMDb ID: %v", err)
	}

	if found.ID != series.ID {
		t.Errorf("Expected ID %s, got %s", series.ID, found.ID)
	}
	if found.IMDbID.String != series.IMDbID.String {
		t.Errorf("Expected IMDb ID %s, got %s", series.IMDbID.String, found.IMDbID.String)
	}
}

// TestSeriesUpdate verifies series update
func TestSeriesUpdate(t *testing.T) {
	db := setupSeriesTestDB(t)
	defer db.Close()

	repo := NewSeriesRepository(db)
	ctx := context.Background()

	// Create series
	series := &models.Series{
		ID:               "series-1",
		Title:            "Original Title",
		FirstAirDate:     "2020-01-01",
		Genres:           []string{"Drama"},
		NumberOfSeasons:  models.NewNullInt64(1),
		NumberOfEpisodes: models.NewNullInt64(10),
	}

	err := repo.Create(ctx, series)
	if err != nil {
		t.Fatalf("Failed to create series: %v", err)
	}

	// Wait a bit to ensure updated_at changes
	time.Sleep(10 * time.Millisecond)

	// Update series
	series.Title = "Updated Title"
	series.Genres = []string{"Drama", "Thriller"}
	series.NumberOfSeasons = models.NewNullInt64(2)
	series.NumberOfEpisodes = models.NewNullInt64(20)
	originalUpdatedAt := series.UpdatedAt

	err = repo.Update(ctx, series)
	if err != nil {
		t.Fatalf("Failed to update series: %v", err)
	}

	// Verify updated_at changed
	if !series.UpdatedAt.After(originalUpdatedAt) {
		t.Error("Expected UpdatedAt to be updated")
	}

	// Find and verify update
	found, err := repo.FindByID(ctx, "series-1")
	if err != nil {
		t.Fatalf("Failed to find series: %v", err)
	}

	if found.Title != "Updated Title" {
		t.Errorf("Expected title 'Updated Title', got '%s'", found.Title)
	}
	if len(found.Genres) != 2 {
		t.Errorf("Expected 2 genres, got %d", len(found.Genres))
	}
	if found.NumberOfSeasons.Int64 != 2 {
		t.Errorf("Expected 2 seasons, got %d", found.NumberOfSeasons.Int64)
	}
}

// TestSeriesUpdateNotFound verifies error for non-existent series
func TestSeriesUpdateNotFound(t *testing.T) {
	db := setupSeriesTestDB(t)
	defer db.Close()

	repo := NewSeriesRepository(db)
	ctx := context.Background()

	series := &models.Series{
		ID:           "non-existent",
		Title:        "Test",
		FirstAirDate: "2020-01-01",
		Genres:       []string{},
	}

	err := repo.Update(ctx, series)
	if err == nil {
		t.Fatal("Expected error for non-existent series, got nil")
	}
}

// TestSeriesDelete verifies series deletion
func TestSeriesDelete(t *testing.T) {
	db := setupSeriesTestDB(t)
	defer db.Close()

	repo := NewSeriesRepository(db)
	ctx := context.Background()

	// Create series
	series := &models.Series{
		ID:           "series-1",
		Title:        "To Be Deleted",
		FirstAirDate: "2020-01-01",
		Genres:       []string{"Drama"},
	}

	err := repo.Create(ctx, series)
	if err != nil {
		t.Fatalf("Failed to create series: %v", err)
	}

	// Delete series
	err = repo.Delete(ctx, "series-1")
	if err != nil {
		t.Fatalf("Failed to delete series: %v", err)
	}

	// Verify series was deleted
	_, err = repo.FindByID(ctx, "series-1")
	if err == nil {
		t.Fatal("Expected error for deleted series, got nil")
	}
}

// TestSeriesDeleteNotFound verifies error for non-existent series
func TestSeriesDeleteNotFound(t *testing.T) {
	db := setupSeriesTestDB(t)
	defer db.Close()

	repo := NewSeriesRepository(db)
	ctx := context.Background()

	err := repo.Delete(ctx, "non-existent")
	if err == nil {
		t.Fatal("Expected error for non-existent series, got nil")
	}
}

// TestSeriesList verifies series listing with pagination
func TestSeriesList(t *testing.T) {
	db := setupSeriesTestDB(t)
	defer db.Close()

	repo := NewSeriesRepository(db)
	ctx := context.Background()

	// Create multiple series
	for i := 1; i <= 5; i++ {
		series := &models.Series{
			ID:           "series-" + string(rune('0'+i)),
			Title:        "Series " + string(rune('0'+i)),
			FirstAirDate: "2020-01-01",
			Genres:       []string{"Drama"},
		}

		err := repo.Create(ctx, series)
		if err != nil {
			t.Fatalf("Failed to create series %d: %v", i, err)
		}
	}

	// List series
	params := NewListParams()
	params.Page = 1
	params.PageSize = 3

	seriesList, pagination, err := repo.List(ctx, params)
	if err != nil {
		t.Fatalf("Failed to list series: %v", err)
	}

	if len(seriesList) != 3 {
		t.Errorf("Expected 3 series, got %d", len(seriesList))
	}

	if pagination.Page != 1 {
		t.Errorf("Expected page 1, got %d", pagination.Page)
	}
	if pagination.PageSize != 3 {
		t.Errorf("Expected page size 3, got %d", pagination.PageSize)
	}
	if pagination.TotalResults != 5 {
		t.Errorf("Expected total results 5, got %d", pagination.TotalResults)
	}
	if pagination.TotalPages != 2 {
		t.Errorf("Expected total pages 2, got %d", pagination.TotalPages)
	}
}

// TestSeriesListEmpty verifies empty list handling
func TestSeriesListEmpty(t *testing.T) {
	db := setupSeriesTestDB(t)
	defer db.Close()

	repo := NewSeriesRepository(db)
	ctx := context.Background()

	params := NewListParams()
	seriesList, pagination, err := repo.List(ctx, params)
	if err != nil {
		t.Fatalf("Failed to list series: %v", err)
	}

	if len(seriesList) != 0 {
		t.Errorf("Expected 0 series, got %d", len(seriesList))
	}
	if pagination.TotalResults != 0 {
		t.Errorf("Expected total results 0, got %d", pagination.TotalResults)
	}
}

// TestSeriesListWithSorting verifies list sorting
func TestSeriesListWithSorting(t *testing.T) {
	db := setupSeriesTestDB(t)
	defer db.Close()

	repo := NewSeriesRepository(db)
	ctx := context.Background()

	// Create series with different ratings
	seriesList := []*models.Series{
		{
			ID:           "series-1",
			Title:        "Series A",
			FirstAirDate: "2020-01-01",
			Genres:       []string{},
			Rating:       models.NewNullFloat64(7.5),
		},
		{
			ID:           "series-2",
			Title:        "Series B",
			FirstAirDate: "2020-01-01",
			Genres:       []string{},
			Rating:       models.NewNullFloat64(9.0),
		},
		{
			ID:           "series-3",
			Title:        "Series C",
			FirstAirDate: "2020-01-01",
			Genres:       []string{},
			Rating:       models.NewNullFloat64(8.0),
		},
	}

	for _, series := range seriesList {
		err := repo.Create(ctx, series)
		if err != nil {
			t.Fatalf("Failed to create series: %v", err)
		}
	}

	// List with rating sort descending
	params := NewListParams()
	params.SortBy = "rating"
	params.SortOrder = "desc"

	result, _, err := repo.List(ctx, params)
	if err != nil {
		t.Fatalf("Failed to list series: %v", err)
	}

	if len(result) != 3 {
		t.Fatalf("Expected 3 series, got %d", len(result))
	}

	// Verify sort order
	if result[0].Rating.Float64 != 9.0 {
		t.Errorf("Expected first series rating 9.0, got %f", result[0].Rating.Float64)
	}
}

// TestSeriesSearchByTitle verifies search functionality
func TestSeriesSearchByTitle(t *testing.T) {
	db := setupSeriesTestDB(t)
	defer db.Close()

	repo := NewSeriesRepository(db)
	ctx := context.Background()

	// Create series
	seriesList := []*models.Series{
		{
			ID:           "series-1",
			Title:        "The Walking Dead",
			FirstAirDate: "2010-10-31",
			Genres:       []string{},
		},
		{
			ID:           "series-2",
			Title:        "The Walking Dead: World Beyond",
			FirstAirDate: "2020-10-04",
			Genres:       []string{},
		},
		{
			ID:           "series-3",
			Title:        "Breaking Bad",
			FirstAirDate: "2008-01-20",
			Genres:       []string{},
		},
	}

	for _, series := range seriesList {
		err := repo.Create(ctx, series)
		if err != nil {
			t.Fatalf("Failed to create series: %v", err)
		}
	}

	// Search for "Walking Dead"
	params := NewListParams()
	result, pagination, err := repo.SearchByTitle(ctx, "Walking Dead", params)
	if err != nil {
		t.Fatalf("Failed to search series: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("Expected 2 series, got %d", len(result))
	}
	if pagination.TotalResults != 2 {
		t.Errorf("Expected total results 2, got %d", pagination.TotalResults)
	}

	// Verify both results contain "Walking Dead"
	for _, series := range result {
		if series.Title != "The Walking Dead" && series.Title != "The Walking Dead: World Beyond" {
			t.Errorf("Unexpected series in results: %s", series.Title)
		}
	}
}

// TestSeriesGenresSerialization verifies genres JSON handling
func TestSeriesGenresSerialization(t *testing.T) {
	db := setupSeriesTestDB(t)
	defer db.Close()

	repo := NewSeriesRepository(db)
	ctx := context.Background()

	// Create series with multiple genres
	series := &models.Series{
		ID:           "series-1",
		Title:        "Test Series",
		FirstAirDate: "2020-01-01",
		Genres:       []string{"Drama", "Science Fiction", "Thriller"},
	}

	err := repo.Create(ctx, series)
	if err != nil {
		t.Fatalf("Failed to create series: %v", err)
	}

	// Retrieve and verify genres
	found, err := repo.FindByID(ctx, "series-1")
	if err != nil {
		t.Fatalf("Failed to find series: %v", err)
	}

	if len(found.Genres) != 3 {
		t.Errorf("Expected 3 genres, got %d", len(found.Genres))
	}

	expectedGenres := map[string]bool{
		"Drama":            true,
		"Science Fiction":  true,
		"Thriller":         true,
	}

	for _, genre := range found.Genres {
		if !expectedGenres[genre] {
			t.Errorf("Unexpected genre: %s", genre)
		}
	}
}

// TestSeriesInProduction verifies in_production field handling
func TestSeriesInProduction(t *testing.T) {
	db := setupSeriesTestDB(t)
	defer db.Close()

	repo := NewSeriesRepository(db)
	ctx := context.Background()

	// Create series in production
	series := &models.Series{
		ID:           "series-1",
		Title:        "Test Series",
		FirstAirDate: "2020-01-01",
		Genres:       []string{"Drama"},
		InProduction: models.NewNullBool(true),
	}

	err := repo.Create(ctx, series)
	if err != nil {
		t.Fatalf("Failed to create series: %v", err)
	}

	// Retrieve and verify
	found, err := repo.FindByID(ctx, "series-1")
	if err != nil {
		t.Fatalf("Failed to find series: %v", err)
	}

	if !found.InProduction.Valid {
		t.Error("Expected InProduction to be valid")
	}
	if !found.InProduction.Bool {
		t.Error("Expected InProduction to be true")
	}
}

// TestSeriesUpsertCreate verifies upsert creates new series when TMDb ID not found
func TestSeriesUpsertCreate(t *testing.T) {
	db := setupSeriesTestDB(t)
	defer db.Close()

	repo := NewSeriesRepository(db)
	ctx := context.Background()

	series := &models.Series{
		ID:           "series-1",
		Title:        "New Series",
		FirstAirDate: "2023-01-01",
		Genres:       []string{"Drama"},
		TMDbID:       models.NewNullInt64(99999),
	}

	err := repo.Upsert(ctx, series)
	if err != nil {
		t.Fatalf("Failed to upsert series: %v", err)
	}

	// Verify series was created
	found, err := repo.FindByTMDbID(ctx, 99999)
	if err != nil {
		t.Fatalf("Failed to find series: %v", err)
	}

	if found.Title != "New Series" {
		t.Errorf("Expected title 'New Series', got '%s'", found.Title)
	}
}

// TestSeriesUpsertUpdate verifies upsert updates existing series by TMDb ID
func TestSeriesUpsertUpdate(t *testing.T) {
	db := setupSeriesTestDB(t)
	defer db.Close()

	repo := NewSeriesRepository(db)
	ctx := context.Background()

	// Create initial series
	series := &models.Series{
		ID:               "series-1",
		Title:            "Original Title",
		FirstAirDate:     "2023-01-01",
		Genres:           []string{"Drama"},
		TMDbID:           models.NewNullInt64(12345),
		NumberOfSeasons:  models.NewNullInt64(2),
		NumberOfEpisodes: models.NewNullInt64(20),
	}

	err := repo.Create(ctx, series)
	if err != nil {
		t.Fatalf("Failed to create series: %v", err)
	}

	// Upsert with same TMDb ID but different data
	updatedSeries := &models.Series{
		ID:               "series-new",
		Title:            "Updated Title",
		FirstAirDate:     "2023-01-01",
		Genres:           []string{"Drama", "Thriller"},
		TMDbID:           models.NewNullInt64(12345),
		NumberOfSeasons:  models.NewNullInt64(3),
		NumberOfEpisodes: models.NewNullInt64(30),
	}

	err = repo.Upsert(ctx, updatedSeries)
	if err != nil {
		t.Fatalf("Failed to upsert series: %v", err)
	}

	// Verify series was updated with original ID
	found, err := repo.FindByTMDbID(ctx, 12345)
	if err != nil {
		t.Fatalf("Failed to find series: %v", err)
	}

	if found.ID != "series-1" {
		t.Errorf("Expected ID 'series-1', got '%s'", found.ID)
	}
	if found.Title != "Updated Title" {
		t.Errorf("Expected title 'Updated Title', got '%s'", found.Title)
	}
	if found.NumberOfSeasons.Int64 != 3 {
		t.Errorf("Expected 3 seasons, got %d", found.NumberOfSeasons.Int64)
	}
}

// TestSeriesUpsertNoTMDbID verifies upsert creates when no TMDb ID is provided
func TestSeriesUpsertNoTMDbID(t *testing.T) {
	db := setupSeriesTestDB(t)
	defer db.Close()

	repo := NewSeriesRepository(db)
	ctx := context.Background()

	series := &models.Series{
		ID:           "series-1",
		Title:        "Series Without TMDb",
		FirstAirDate: "2023-01-01",
		Genres:       []string{"Drama"},
		// No TMDb ID set
	}

	err := repo.Upsert(ctx, series)
	if err != nil {
		t.Fatalf("Failed to upsert series: %v", err)
	}

	// Verify series was created
	found, err := repo.FindByID(ctx, "series-1")
	if err != nil {
		t.Fatalf("Failed to find series: %v", err)
	}

	if found.Title != "Series Without TMDb" {
		t.Errorf("Expected title 'Series Without TMDb', got '%s'", found.Title)
	}
}

// TestSeriesUpsertNil verifies nil series rejection
func TestSeriesUpsertNil(t *testing.T) {
	db := setupSeriesTestDB(t)
	defer db.Close()

	repo := NewSeriesRepository(db)
	ctx := context.Background()

	err := repo.Upsert(ctx, nil)
	if err == nil {
		t.Fatal("Expected error for nil series, got nil")
	}
}

// TestSeriesFindByTMDbIDNotFound verifies error for non-existent TMDb ID
func TestSeriesFindByTMDbIDNotFound(t *testing.T) {
	db := setupSeriesTestDB(t)
	defer db.Close()

	repo := NewSeriesRepository(db)
	ctx := context.Background()

	_, err := repo.FindByTMDbID(ctx, 99999)
	if err == nil {
		t.Fatal("Expected error for non-existent TMDb ID, got nil")
	}
}

// TestSeriesFindByIMDbIDNotFound verifies error for non-existent IMDb ID
func TestSeriesFindByIMDbIDNotFound(t *testing.T) {
	db := setupSeriesTestDB(t)
	defer db.Close()

	repo := NewSeriesRepository(db)
	ctx := context.Background()

	_, err := repo.FindByIMDbID(ctx, "tt9999999")
	if err == nil {
		t.Fatal("Expected error for non-existent IMDb ID, got nil")
	}
}

// TestSeriesFindOwnedTMDbIDs verifies batch ownership lookup semantics (Story 10-4).
func TestSeriesFindOwnedTMDbIDs(t *testing.T) {
	db := setupSeriesTestDB(t)
	defer db.Close()

	repo := NewSeriesRepository(db)
	ctx := context.Background()

	seedSeries := []*models.Series{
		{
			ID:           "s-1396",
			Title:        "Breaking Bad",
			FirstAirDate: "2008-01-20",
			Genres:       []string{"Drama"},
			TMDbID:       models.NewNullInt64(1396),
			IsRemoved:    false,
		},
		{
			ID:           "s-66732",
			Title:        "Stranger Things",
			FirstAirDate: "2016-07-15",
			Genres:       []string{"Drama"},
			TMDbID:       models.NewNullInt64(66732),
			IsRemoved:    false,
		},
		{
			ID:           "s-removed",
			Title:        "Lost (soft-deleted)",
			FirstAirDate: "2004-09-22",
			Genres:       []string{"Drama"},
			TMDbID:       models.NewNullInt64(4607),
			IsRemoved:    true,
		},
	}
	for _, s := range seedSeries {
		if err := repo.Create(ctx, s); err != nil {
			t.Fatalf("seed failed for %s: %v", s.Title, err)
		}
	}
	// SeriesRepository.Create does not write is_removed (relies on DB default 0).
	// Flip the soft-deleted row manually to mirror production soft-delete flow.
	if _, err := db.ExecContext(ctx, `UPDATE series SET is_removed = 1 WHERE id = 's-removed'`); err != nil {
		t.Fatalf("seed soft-delete update failed: %v", err)
	}

	t.Run("returns subset of owned non-removed ids", func(t *testing.T) {
		owned, err := repo.FindOwnedTMDbIDs(ctx, []int64{1396, 66732, 4607, 999999})
		if err != nil {
			t.Fatalf("FindOwnedTMDbIDs error: %v", err)
		}
		got := map[int64]bool{}
		for _, id := range owned {
			got[id] = true
		}
		if !got[1396] || !got[66732] {
			t.Errorf("expected 1396 and 66732 owned, got %v", owned)
		}
		if got[4607] {
			t.Error("expected soft-deleted 4607 to be excluded")
		}
	})

	t.Run("empty input returns empty slice", func(t *testing.T) {
		owned, err := repo.FindOwnedTMDbIDs(ctx, []int64{})
		if err != nil {
			t.Fatalf("FindOwnedTMDbIDs error: %v", err)
		}
		if owned == nil || len(owned) != 0 {
			t.Errorf("expected empty slice, got %v", owned)
		}
	})
}

// TestSeriesEmptyGenres verifies empty genres handling
func TestSeriesEmptyGenres(t *testing.T) {
	db := setupSeriesTestDB(t)
	defer db.Close()

	repo := NewSeriesRepository(db)
	ctx := context.Background()

	// Create series with no genres
	series := &models.Series{
		ID:           "series-1",
		Title:        "Test Series",
		FirstAirDate: "2020-01-01",
		Genres:       []string{},
	}

	err := repo.Create(ctx, series)
	if err != nil {
		t.Fatalf("Failed to create series: %v", err)
	}

	// Retrieve and verify genres
	found, err := repo.FindByID(ctx, "series-1")
	if err != nil {
		t.Fatalf("Failed to find series: %v", err)
	}

	if found.Genres == nil {
		t.Error("Expected empty slice for genres, got nil")
	}
	if len(found.Genres) != 0 {
		t.Errorf("Expected 0 genres, got %d", len(found.Genres))
	}
}

// setupSeriesTestDBWithFTS creates an in-memory database with series table and FTS5 virtual table
func setupSeriesTestDBWithFTS(t *testing.T) *sql.DB {
	db := setupSeriesTestDB(t)

	// Create FTS5 virtual table for full-text search
	_, err := db.Exec(`
		CREATE VIRTUAL TABLE series_fts USING fts5(
			title,
			original_title,
			overview,
			content='series',
			content_rowid='rowid'
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create FTS5 table: %v", err)
	}

	// Create triggers to keep FTS in sync
	_, err = db.Exec(`
		CREATE TRIGGER series_ai AFTER INSERT ON series BEGIN
			INSERT INTO series_fts(rowid, title, original_title, overview)
			VALUES (new.rowid, new.title, new.original_title, new.overview);
		END
	`)
	if err != nil {
		t.Fatalf("Failed to create FTS insert trigger: %v", err)
	}

	_, err = db.Exec(`
		CREATE TRIGGER series_ad AFTER DELETE ON series BEGIN
			INSERT INTO series_fts(series_fts, rowid, title, original_title, overview)
			VALUES ('delete', old.rowid, old.title, old.original_title, old.overview);
		END
	`)
	if err != nil {
		t.Fatalf("Failed to create FTS delete trigger: %v", err)
	}

	_, err = db.Exec(`
		CREATE TRIGGER series_au AFTER UPDATE ON series BEGIN
			INSERT INTO series_fts(series_fts, rowid, title, original_title, overview)
			VALUES ('delete', old.rowid, old.title, old.original_title, old.overview);
			INSERT INTO series_fts(rowid, title, original_title, overview)
			VALUES (new.rowid, new.title, new.original_title, new.overview);
		END
	`)
	if err != nil {
		t.Fatalf("Failed to create FTS update trigger: %v", err)
	}

	return db
}

// TestSeriesGetStats verifies GetStats returns correct total and unmatched counts
func TestSeriesGetStats(t *testing.T) {
	db := setupSeriesTestDB(t)
	defer db.Close()

	repo := NewSeriesRepository(db)
	ctx := context.Background()

	series := []*models.Series{
		{ID: "s1", Title: "Matched", FirstAirDate: "2020-01-01", Genres: []string{}, TMDbID: models.NewNullInt64(100)},
		{ID: "s2", Title: "Unmatched", FirstAirDate: "2020-01-01", Genres: []string{}},
	}

	for _, s := range series {
		if err := repo.Create(ctx, s); err != nil {
			t.Fatalf("Failed to create series: %v", err)
		}
	}

	stats, err := repo.GetStats(ctx)
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	if stats.Total != 2 {
		t.Errorf("Expected total 2, got %d", stats.Total)
	}
	if stats.UnmatchedCount != 1 {
		t.Errorf("Expected unmatched 1, got %d", stats.UnmatchedCount)
	}
}

// TestSeriesGetStatsEmpty verifies GetStats works with empty table
func TestSeriesGetStatsEmpty(t *testing.T) {
	db := setupSeriesTestDB(t)
	defer db.Close()

	repo := NewSeriesRepository(db)
	ctx := context.Background()

	stats, err := repo.GetStats(ctx)
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	if stats.Total != 0 {
		t.Errorf("Expected total 0, got %d", stats.Total)
	}
	if stats.UnmatchedCount != 0 {
		t.Errorf("Expected unmatched 0, got %d", stats.UnmatchedCount)
	}
}

// TestSeriesListUnmatchedFilter verifies unmatched filter returns only series without TMDb ID
func TestSeriesListUnmatchedFilter(t *testing.T) {
	db := setupSeriesTestDB(t)
	defer db.Close()

	repo := NewSeriesRepository(db)
	ctx := context.Background()

	// Create matched series (have tmdb_id)
	matched := &models.Series{
		ID:           "series-matched",
		Title:        "Matched Series",
		FirstAirDate: "2020-01-01",
		Genres:       []string{"Drama"},
		TMDbID:       models.NewNullInt64(55555),
	}

	// Create unmatched series (no tmdb_id or tmdb_id=0)
	unmatchedNull := &models.Series{
		ID:           "series-unmatched-null",
		Title:        "Unmatched Null",
		FirstAirDate: "2020-01-01",
		Genres:       []string{"Horror"},
	}
	unmatchedZero := &models.Series{
		ID:           "series-unmatched-zero",
		Title:        "Unmatched Zero",
		FirstAirDate: "2020-01-01",
		Genres:       []string{"Comedy"},
		TMDbID:       models.NewNullInt64(0),
	}

	for _, s := range []*models.Series{matched, unmatchedNull, unmatchedZero} {
		if err := repo.Create(ctx, s); err != nil {
			t.Fatalf("Failed to create series: %v", err)
		}
	}

	// List with unmatched filter
	params := NewListParams()
	params.Filters["unmatched"] = true

	series, pagination, err := repo.List(ctx, params)
	if err != nil {
		t.Fatalf("Failed to list unmatched series: %v", err)
	}

	if len(series) != 2 {
		t.Errorf("Expected 2 unmatched series, got %d", len(series))
	}
	if pagination.TotalResults != 2 {
		t.Errorf("Expected total results 2, got %d", pagination.TotalResults)
	}

	for _, s := range series {
		if s.ID != "series-unmatched-null" && s.ID != "series-unmatched-zero" {
			t.Errorf("Unexpected matched series in results: %s", s.ID)
		}
	}
}

// TestSeriesFullTextSearchEmptyQuery verifies empty query falls back to List
func TestSeriesFullTextSearchEmptyQuery(t *testing.T) {
	db := setupSeriesTestDBWithFTS(t)
	defer db.Close()

	repo := NewSeriesRepository(db)
	ctx := context.Background()

	// Create some series
	seriesList := []*models.Series{
		{ID: "series-1", Title: "Breaking Bad", FirstAirDate: "2008-01-20", Genres: []string{"Drama"}},
		{ID: "series-2", Title: "Game of Thrones", FirstAirDate: "2011-04-17", Genres: []string{"Fantasy"}},
	}

	for _, s := range seriesList {
		err := repo.Create(ctx, s)
		if err != nil {
			t.Fatalf("Failed to create series: %v", err)
		}
	}

	// Empty query should fall back to List
	params := NewListParams()
	result, pagination, err := repo.FullTextSearch(ctx, "", params)
	if err != nil {
		t.Fatalf("Failed to search series: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("Expected 2 series, got %d", len(result))
	}
	if pagination.TotalResults != 2 {
		t.Errorf("Expected total results 2, got %d", pagination.TotalResults)
	}
}

// TestSeriesFullTextSearchByTitle verifies FTS search by title
func TestSeriesFullTextSearchByTitle(t *testing.T) {
	db := setupSeriesTestDBWithFTS(t)
	defer db.Close()

	repo := NewSeriesRepository(db)
	ctx := context.Background()

	// Create series
	seriesList := []*models.Series{
		{
			ID:           "series-1",
			Title:        "The Walking Dead",
			FirstAirDate: "2010-10-31",
			Genres:       []string{"Horror"},
			Overview:     models.NewNullString("Sheriff's deputy awakens from a coma to find a zombie apocalypse."),
		},
		{
			ID:           "series-2",
			Title:        "Fear the Walking Dead",
			FirstAirDate: "2015-08-23",
			Genres:       []string{"Horror"},
			Overview:     models.NewNullString("A prequel to The Walking Dead."),
		},
		{
			ID:           "series-3",
			Title:        "Breaking Bad",
			FirstAirDate: "2008-01-20",
			Genres:       []string{"Drama"},
			Overview:     models.NewNullString("A chemistry teacher turns to cooking meth."),
		},
	}

	for _, s := range seriesList {
		err := repo.Create(ctx, s)
		if err != nil {
			t.Fatalf("Failed to create series: %v", err)
		}
	}

	// Search for "Walking Dead"
	params := NewListParams()
	result, pagination, err := repo.FullTextSearch(ctx, "Walking Dead", params)
	if err != nil {
		t.Fatalf("Failed to search series: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("Expected 2 series matching 'Walking Dead', got %d", len(result))
	}
	if pagination.TotalResults != 2 {
		t.Errorf("Expected total results 2, got %d", pagination.TotalResults)
	}
}

// TestSeriesFullTextSearchWithPagination verifies FTS pagination
func TestSeriesFullTextSearchWithPagination(t *testing.T) {
	db := setupSeriesTestDBWithFTS(t)
	defer db.Close()

	repo := NewSeriesRepository(db)
	ctx := context.Background()

	// Create 5 series with "Drama" in title
	for i := 1; i <= 5; i++ {
		series := &models.Series{
			ID:           "series-" + string(rune('0'+i)),
			Title:        "Drama Series " + string(rune('0'+i)),
			FirstAirDate: "2020-01-01",
			Genres:       []string{"Drama"},
		}
		err := repo.Create(ctx, series)
		if err != nil {
			t.Fatalf("Failed to create series: %v", err)
		}
	}

	// Search with pagination
	params := NewListParams()
	params.Page = 1
	params.PageSize = 2

	result, pagination, err := repo.FullTextSearch(ctx, "Drama", params)
	if err != nil {
		t.Fatalf("Failed to search series: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("Expected 2 series on page 1, got %d", len(result))
	}
	if pagination.TotalResults != 5 {
		t.Errorf("Expected total results 5, got %d", pagination.TotalResults)
	}
	if pagination.TotalPages != 3 {
		t.Errorf("Expected 3 total pages, got %d", pagination.TotalPages)
	}
}

// TestSeriesFullTextSearchNoResults verifies empty results handling
func TestSeriesFullTextSearchNoResults(t *testing.T) {
	db := setupSeriesTestDBWithFTS(t)
	defer db.Close()

	repo := NewSeriesRepository(db)
	ctx := context.Background()

	// Create a series
	series := &models.Series{
		ID:           "series-1",
		Title:        "Breaking Bad",
		FirstAirDate: "2008-01-20",
		Genres:       []string{"Drama"},
	}
	err := repo.Create(ctx, series)
	if err != nil {
		t.Fatalf("Failed to create series: %v", err)
	}

	// Search for non-existent term
	params := NewListParams()
	result, pagination, err := repo.FullTextSearch(ctx, "NonExistentTerm", params)
	if err != nil {
		t.Fatalf("Failed to search series: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("Expected 0 series, got %d", len(result))
	}
	if pagination.TotalResults != 0 {
		t.Errorf("Expected total results 0, got %d", pagination.TotalResults)
	}
}
