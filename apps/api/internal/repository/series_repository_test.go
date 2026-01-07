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
		OriginalTitle:    sql.NullString{String: "Breaking Bad", Valid: true},
		FirstAirDate:     "2008-01-20",
		LastAirDate:      sql.NullString{String: "2013-09-29", Valid: true},
		Genres:           []string{"Drama", "Crime", "Thriller"},
		Rating:           sql.NullFloat64{Float64: 9.5, Valid: true},
		Overview:         sql.NullString{String: "A high school chemistry teacher turned meth cook.", Valid: true},
		NumberOfSeasons:  sql.NullInt64{Int64: 5, Valid: true},
		NumberOfEpisodes: sql.NullInt64{Int64: 62, Valid: true},
		Status:           sql.NullString{String: "Ended", Valid: true},
		OriginalLanguage: sql.NullString{String: "en", Valid: true},
		IMDbID:           sql.NullString{String: "tt0903747", Valid: true},
		TMDbID:           sql.NullInt64{Int64: 1396, Valid: true},
		InProduction:     sql.NullBool{Bool: false, Valid: true},
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
		Rating:           sql.NullFloat64{Float64: 9.3, Valid: true},
		NumberOfSeasons:  sql.NullInt64{Int64: 8, Valid: true},
		NumberOfEpisodes: sql.NullInt64{Int64: 73, Valid: true},
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
		TMDbID:       sql.NullInt64{Int64: 82856, Valid: true},
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
		IMDbID:       sql.NullString{String: "tt4574334", Valid: true},
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
		NumberOfSeasons:  sql.NullInt64{Int64: 1, Valid: true},
		NumberOfEpisodes: sql.NullInt64{Int64: 10, Valid: true},
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
	series.NumberOfSeasons = sql.NullInt64{Int64: 2, Valid: true}
	series.NumberOfEpisodes = sql.NullInt64{Int64: 20, Valid: true}
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
			Rating:       sql.NullFloat64{Float64: 7.5, Valid: true},
		},
		{
			ID:           "series-2",
			Title:        "Series B",
			FirstAirDate: "2020-01-01",
			Genres:       []string{},
			Rating:       sql.NullFloat64{Float64: 9.0, Valid: true},
		},
		{
			ID:           "series-3",
			Title:        "Series C",
			FirstAirDate: "2020-01-01",
			Genres:       []string{},
			Rating:       sql.NullFloat64{Float64: 8.0, Valid: true},
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
		InProduction: sql.NullBool{Bool: true, Valid: true},
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
