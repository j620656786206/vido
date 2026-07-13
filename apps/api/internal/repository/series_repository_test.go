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
			vote_count INTEGER,
			is_removed INTEGER NOT NULL DEFAULT 0,
			library_id TEXT,
			video_codec TEXT,
			video_resolution TEXT,
			audio_codec TEXT,
			audio_channels INTEGER,
			subtitle_tracks TEXT,
			hdr_format TEXT,
			credits TEXT,
			douban_id TEXT,
			douban_rating REAL,
			douban_vote_count INTEGER,
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

// TestSeriesListReturnsVoteAverage guards the Rule-15 List SELECT/scan desync for
// the rating column: the scanner stores the TMDb rating in vote_average, so a List
// query that omits it silently drops the rating from every list view and breaks
// sort_by=rating. List must surface vote_average the same way detail (FindByID) does.
func TestSeriesListReturnsVoteAverage(t *testing.T) {
	db := setupSeriesTestDB(t)
	defer db.Close()

	repo := NewSeriesRepository(db)
	ctx := context.Background()

	series := &models.Series{
		ID:           "series-vote",
		Title:        "Vote Series",
		FirstAirDate: "2020-01-01",
		Genres:       []string{"Drama"},
	}
	if err := repo.Create(ctx, series); err != nil {
		t.Fatalf("Failed to create series: %v", err)
	}

	if _, err := db.ExecContext(ctx,
		`UPDATE series SET vote_average = ? WHERE id = ?`, 8.9, "series-vote",
	); err != nil {
		t.Fatalf("Failed to set vote_average: %v", err)
	}

	params := NewListParams()
	params.Page = 1
	params.PageSize = 10

	seriesList, _, err := repo.List(ctx, params)
	if err != nil {
		t.Fatalf("Failed to list series: %v", err)
	}
	if len(seriesList) != 1 {
		t.Fatalf("Expected 1 series, got %d", len(seriesList))
	}

	got := seriesList[0]
	if !got.VoteAverage.Valid || got.VoteAverage.Float64 != 8.9 {
		t.Errorf("List() VoteAverage = %v, want 8.9 (Rule-15 List SELECT/scan desync — rating dropped from list views)", got.VoteAverage)
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
		"Drama":           true,
		"Science Fiction": true,
		"Thriller":        true,
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

// TestSeriesCreditsRoundTrip verifies credits persist through Create AND Update, are read
// back via seriesSelectColumns/scanSeries, and populate the wire-exposed Credits field.
// Same data-loss class as movies — the series Metadata Editor's SetCredits was silently
// dropped by Update before this story (Rule 15 real-DB test, not a mocked repo).
func TestSeriesCreditsRoundTrip(t *testing.T) {
	db := setupSeriesTestDB(t)
	defer db.Close()

	repo := NewSeriesRepository(db)
	ctx := context.Background()

	s := &models.Series{ID: "series-credits", Title: "Edited Series"}
	if err := s.SetCredits(&models.Credits{
		Cast: []models.CastMember{{Name: "Actor One", Order: 0}},
		Crew: []models.CrewMember{{Name: "Showrunner", Job: "Executive Producer", Department: "Production"}},
	}); err != nil {
		t.Fatalf("SetCredits: %v", err)
	}

	if err := repo.Create(ctx, s); err != nil {
		t.Fatalf("Create: %v", err)
	}
	found, err := repo.FindByID(ctx, "series-credits")
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if found.Credits == nil || len(found.Credits.Cast) != 1 || found.Credits.Cast[0].Name != "Actor One" {
		t.Fatalf("expected [Actor One] cast, got %+v", found.Credits)
	}
	if len(found.Credits.Crew) != 1 || found.Credits.Crew[0].Job != "Executive Producer" {
		t.Errorf("expected crew round-trip, got %+v", found.Credits)
	}

	// Update path must also persist credits (the data-loss origin — SetCredits + Update).
	if err := found.SetCredits(&models.Credits{Cast: []models.CastMember{{Name: "Replaced", Order: 0}}}); err != nil {
		t.Fatalf("SetCredits (update): %v", err)
	}
	if err := repo.Update(ctx, found); err != nil {
		t.Fatalf("Update: %v", err)
	}
	refound, err := repo.FindByID(ctx, "series-credits")
	if err != nil {
		t.Fatalf("FindByID after update: %v", err)
	}
	if refound.Credits == nil || refound.Credits.Cast[0].Name != "Replaced" {
		t.Errorf("expected [Replaced] after update, got %+v", refound.Credits)
	}
}

// TestSeriesBulkCreateCredits verifies BulkCreate also persists credits, and that a series
// with no credits reads back nil Credits (omitempty → absent on the wire).
func TestSeriesBulkCreateCredits(t *testing.T) {
	db := setupSeriesTestDB(t)
	defer db.Close()

	repo := NewSeriesRepository(db)
	ctx := context.Background()

	s := &models.Series{ID: "bulk-series-credits", Title: "Bulk Series"}
	if err := s.SetCredits(&models.Credits{Cast: []models.CastMember{{Name: "BulkActor", Order: 0}}}); err != nil {
		t.Fatalf("SetCredits: %v", err)
	}
	if err := repo.BulkCreate(ctx, []*models.Series{s}); err != nil {
		t.Fatalf("BulkCreate: %v", err)
	}
	found, err := repo.FindByID(ctx, "bulk-series-credits")
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if found.Credits == nil || found.Credits.Cast[0].Name != "BulkActor" {
		t.Errorf("expected [BulkActor] via BulkCreate, got %+v", found.Credits)
	}

	plain := &models.Series{ID: "plain-series", Title: "Plain"}
	if err := repo.Create(ctx, plain); err != nil {
		t.Fatalf("Create plain: %v", err)
	}
	pf, err := repo.FindByID(ctx, "plain-series")
	if err != nil {
		t.Fatalf("FindByID plain: %v", err)
	}
	if pf.Credits != nil {
		t.Errorf("expected nil Credits for series without credits, got %+v", pf.Credits)
	}
}

// TestSeriesUpsertPreservesManualCredits guards the same credits data-loss regression for
// series: a re-match (SaveSeriesFromTMDb → Upsert, fresh model with no credits) must NOT
// overwrite a manually-edited cast.
func TestSeriesUpsertPreservesManualCredits(t *testing.T) {
	db := setupSeriesTestDB(t)
	defer db.Close()

	repo := NewSeriesRepository(db)
	ctx := context.Background()

	s := &models.Series{ID: "sup-1", Title: "Manual Series", TMDbID: models.NewNullInt64(777)}
	if err := s.SetCredits(&models.Credits{Cast: []models.CastMember{{Name: "ManualSeriesCast", Order: 0}}}); err != nil {
		t.Fatalf("SetCredits: %v", err)
	}
	if err := repo.Create(ctx, s); err != nil {
		t.Fatalf("Create: %v", err)
	}

	fresh := &models.Series{Title: "Manual Series (re-scan)", TMDbID: models.NewNullInt64(777)}
	if err := repo.Upsert(ctx, fresh); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	found, err := repo.FindByID(ctx, "sup-1")
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if found.Credits == nil || len(found.Credits.Cast) != 1 || found.Credits.Cast[0].Name != "ManualSeriesCast" {
		t.Errorf("manual series credits must survive re-scan, got %+v", found.Credits)
	}
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

// TestSeriesListReturnsLifecycleFields guards the Rule-15 List() SELECT/scan sync
// (ux3-0-1): parse_status / subtitle_status / subtitle_language must reach the
// library-list read path, not only single-row FindByID. Mirrors the movie guard;
// the v2 poster badge (N1) derives from these fields.
func TestSeriesListReturnsLifecycleFields(t *testing.T) {
	db := setupSeriesTestDB(t)
	defer db.Close()

	repo := NewSeriesRepository(db)
	ctx := context.Background()

	s := &models.Series{
		ID:           "series-lifecycle",
		Title:        "Lifecycle Series",
		FirstAirDate: "2021-01-01",
		Genres:       []string{"Drama"},
	}
	if err := repo.Create(ctx, s); err != nil {
		t.Fatalf("Failed to create series: %v", err)
	}

	if _, err := db.ExecContext(ctx,
		`UPDATE series SET parse_status = ?, subtitle_status = ?, subtitle_language = ? WHERE id = ?`,
		string(models.ParseStatusSuccess), string(models.SubtitleStatusNotFound), "zh-Hant", "series-lifecycle",
	); err != nil {
		t.Fatalf("Failed to set lifecycle fields: %v", err)
	}

	params := NewListParams()
	params.Page = 1
	params.PageSize = 10

	list, _, err := repo.List(ctx, params)
	if err != nil {
		t.Fatalf("Failed to list series: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("Expected 1 series, got %d", len(list))
	}

	got := list[0]
	if got.ParseStatus != models.ParseStatusSuccess {
		t.Errorf("List() ParseStatus = %q, want %q (Rule-15 List SELECT/scan desync)", got.ParseStatus, models.ParseStatusSuccess)
	}
	if got.SubtitleStatus != models.SubtitleStatusNotFound {
		t.Errorf("List() SubtitleStatus = %q, want %q", got.SubtitleStatus, models.SubtitleStatusNotFound)
	}
	if !got.SubtitleLanguage.Valid || got.SubtitleLanguage.String != "zh-Hant" {
		t.Errorf("List() SubtitleLanguage = %v, want zh-Hant", got.SubtitleLanguage)
	}
}

// fullyPopulatedSeries returns a series with every persisted column set to a non-zero
// value, so a read path that forgets a column fails loudly. Mirrors the movie-side helper.
func fullyPopulatedSeries(id string, tmdbID int64, filePath string) *models.Series {
	return &models.Series{
		ID:               id,
		Title:            "Blades of the Guardians",
		OriginalTitle:    models.NewNullString("鏢人"),
		FirstAirDate:     "2026-02-17",
		LastAirDate:      models.NewNullString("2026-05-17"),
		Genres:           []string{"Action", "Animation"},
		Rating:           models.NewNullFloat64(8.2),
		Overview:         models.NewNullString("An overview."),
		PosterPath:       models.NewNullString("/poster.jpg"),
		BackdropPath:     models.NewNullString("/backdrop.jpg"),
		NumberOfSeasons:  models.NewNullInt64(2),
		NumberOfEpisodes: models.NewNullInt64(24),
		Status:           models.NewNullString("Returning Series"),
		OriginalLanguage: models.NewNullString("zh"),
		IMDbID:           models.NewNullString("tt7654321"),
		TMDbID:           models.NewNullInt64(tmdbID),
		InProduction:     models.NewNullBool(true),
		FilePath:         models.NewNullString(filePath),
		FileSize:         models.NewNullInt64(4_294_967_296),
		ParseStatus:      models.ParseStatusSuccess,
		MetadataSource:   models.NewNullString("tmdb"),
		LibraryID:        models.NewNullString("lib-tv"),
		SubtitleStatus:   models.SubtitleStatusFound,
		SubtitlePath:     models.NewNullString("/media/sub.srt"),
		SubtitleLanguage: models.NewNullString("zh-TW"),
		VoteAverage:      models.NewNullFloat64(8.4),
		VoteCount:        models.NewNullInt64(3100),
		VideoCodec:       models.NewNullString("x265"),
		VideoResolution:  models.NewNullString("1080p"),
		AudioCodec:       models.NewNullString("AAC"),
		AudioChannels:    models.NewNullInt64(6),
		SubtitleTracks:   models.NewNullString(`["zh-TW"]`),
		HDRFormat:        models.NewNullString("HDR10"),
		CreditsJSON:      models.NewNullString(`{"cast":[{"name":"Somebody"}],"crew":[]}`),
	}
}

func assertSeriesFullyPopulated(t *testing.T, readPath string, s *models.Series) {
	t.Helper()

	if !s.FilePath.Valid || s.FilePath.String == "" {
		t.Errorf("%s: file_path is empty — the read path dropped the column", readPath)
	}
	if !s.FileSize.Valid || s.FileSize.Int64 == 0 {
		t.Errorf("%s: file_size is empty — the read path dropped the column", readPath)
	}
	if !s.LibraryID.Valid || s.LibraryID.String == "" {
		t.Errorf("%s: library_id is empty — the read path dropped the column", readPath)
	}
	if !s.MetadataSource.Valid || s.MetadataSource.String == "" {
		t.Errorf("%s: metadata_source is empty — the read path dropped the column", readPath)
	}
	if !s.SubtitlePath.Valid || s.SubtitlePath.String == "" {
		t.Errorf("%s: subtitle_path is empty — the read path dropped the column", readPath)
	}
	if !s.VoteCount.Valid || s.VoteCount.Int64 == 0 {
		t.Errorf("%s: vote_count is empty — the read path dropped the column", readPath)
	}
	if s.Credits == nil || len(s.Credits.Cast) == 0 {
		t.Errorf("%s: credits is empty — the read path dropped the column", readPath)
	}
}

// TestEverySeriesReadPathReturnsEveryColumn is the series twin of the movie guard.
// series_repository shipped the identical defect (List and FullTextSearch each spelled
// their own SELECT list); it was invisible only because the series table is empty in
// production — every TV episode is currently ingested into the movies table instead.
func TestEverySeriesReadPathReturnsEveryColumn(t *testing.T) {
	db := setupSeriesTestDBWithFTS(t)
	defer db.Close()

	repo := NewSeriesRepository(db)
	ctx := context.Background()

	// Walk the real write lifecycle: Create persists what the scanner knows, and the
	// subtitle columns have a dedicated writer (UpdateSubtitleStatus), so a row is only
	// fully populated on disk after both. Anything still empty on read is a read-path bug.
	s := fullyPopulatedSeries("series-full", 1399, "/media/tv/blades")
	if err := repo.Create(ctx, s); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if err := repo.UpdateSubtitleStatus(ctx, s.ID, models.SubtitleStatusFound, "/media/sub.srt", "zh-TW", 0.91); err != nil {
		t.Fatalf("UpdateSubtitleStatus failed: %v", err)
	}

	t.Run("FindByID", func(t *testing.T) {
		got, err := repo.FindByID(ctx, "series-full")
		if err != nil {
			t.Fatalf("FindByID failed: %v", err)
		}
		assertSeriesFullyPopulated(t, "FindByID", got)
	})

	t.Run("List", func(t *testing.T) {
		list, _, err := repo.List(ctx, ListParams{Page: 1, PageSize: 10})
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}
		if len(list) != 1 {
			t.Fatalf("Expected 1 series, got %d", len(list))
		}
		assertSeriesFullyPopulated(t, "List", &list[0])
	})

	t.Run("FullTextSearch", func(t *testing.T) {
		list, _, err := repo.FullTextSearch(ctx, "Blades", ListParams{Page: 1, PageSize: 10})
		if err != nil {
			t.Fatalf("FullTextSearch failed: %v", err)
		}
		if len(list) != 1 {
			t.Fatalf("Expected 1 FTS hit, got %d", len(list))
		}
		assertSeriesFullyPopulated(t, "FullTextSearch", &list[0])
	})
}

// TestSeriesListAndCountExcludeRemoved — the series twin of the soft-delete leak that
// made /api/v1/movies (5922) and /api/v1/movies/stats (5163) disagree in production.
func TestSeriesListAndCountExcludeRemoved(t *testing.T) {
	db := setupSeriesTestDBWithFTS(t)
	defer db.Close()

	repo := NewSeriesRepository(db)
	ctx := context.Background()

	live := fullyPopulatedSeries("series-live", 1, "/media/tv/live")
	if err := repo.Create(ctx, live); err != nil {
		t.Fatalf("Create live failed: %v", err)
	}
	removed := fullyPopulatedSeries("series-removed", 2, "/media/tv/removed")
	removed.IsRemoved = true
	if err := repo.Create(ctx, removed); err != nil {
		t.Fatalf("Create removed failed: %v", err)
	}

	list, pagination, err := repo.List(ctx, ListParams{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(list) != 1 || list[0].ID != "series-live" {
		t.Fatalf("Expected only series-live, got %d rows", len(list))
	}
	if pagination.TotalResults != 1 {
		t.Errorf("Expected TotalResults 1, got %d", pagination.TotalResults)
	}

	fts, _, err := repo.FullTextSearch(ctx, "Blades", ListParams{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("FullTextSearch failed: %v", err)
	}
	if len(fts) != 1 || fts[0].ID != "series-live" {
		t.Errorf("Expected only series-live from FTS, got %d rows", len(fts))
	}

	// Count feeds /api/v1/library/stats. It must agree with List and GetStats, or the
	// library header reports a different total than the grid below it.
	count, err := repo.Count(ctx)
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}
	if count != 1 {
		t.Errorf("Count = %d, want 1 — Count does not exclude soft-deleted series", count)
	}

	stats, err := repo.GetStats(ctx)
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}
	if stats.Total != count {
		t.Errorf("GetStats.Total (%d) != Count (%d) — the two count paths disagree", stats.Total, count)
	}
}

// TestSeriesCreatePersistsLibraryID — the series table has had library_id since migration
// 020, but models.Series never carried the field, so no write path could populate it.
func TestSeriesCreatePersistsLibraryID(t *testing.T) {
	db := setupSeriesTestDB(t)
	defer db.Close()

	repo := NewSeriesRepository(db)
	ctx := context.Background()

	t.Run("Create", func(t *testing.T) {
		s := fullyPopulatedSeries("series-create", 1, "/media/tv/create")
		if err := repo.Create(ctx, s); err != nil {
			t.Fatalf("Create failed: %v", err)
		}

		var libraryID, filePath sql.NullString
		if err := db.QueryRow(
			"SELECT library_id, file_path FROM series WHERE id = ?", "series-create",
		).Scan(&libraryID, &filePath); err != nil {
			t.Fatalf("Failed to read row: %v", err)
		}
		if !libraryID.Valid || libraryID.String != "lib-tv" {
			t.Errorf("library_id = %v, want 'lib-tv' — INSERT dropped the column", libraryID)
		}
		// file_path was missing from the INSERT too, even though Update always wrote it.
		if !filePath.Valid || filePath.String != "/media/tv/create" {
			t.Errorf("file_path = %v, want '/media/tv/create' — INSERT dropped the column", filePath)
		}
	})

	t.Run("BulkCreate", func(t *testing.T) {
		list := []*models.Series{
			fullyPopulatedSeries("series-bulk-1", 11, "/media/tv/bulk1"),
			fullyPopulatedSeries("series-bulk-2", 12, "/media/tv/bulk2"),
		}
		if err := repo.BulkCreate(ctx, list); err != nil {
			t.Fatalf("BulkCreate failed: %v", err)
		}

		var count int
		if err := db.QueryRow(
			`SELECT COUNT(*) FROM series WHERE id LIKE 'series-bulk-%' AND library_id = ? AND file_path IS NOT NULL`,
			"lib-tv",
		).Scan(&count); err != nil {
			t.Fatalf("Failed to count: %v", err)
		}
		if count != 2 {
			t.Errorf("Expected 2 bulk-created series with library_id + file_path, got %d", count)
		}
	})
}
