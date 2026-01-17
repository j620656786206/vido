package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/vido/api/internal/models"
	_ "modernc.org/sqlite"
)

// setupTestDB creates an in-memory database with movies table
func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Create movies table
	_, err = db.Exec(`
		CREATE TABLE movies (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			original_title TEXT,
			release_date TEXT,
			genres TEXT NOT NULL DEFAULT '[]',
			rating REAL,
			overview TEXT,
			poster_path TEXT,
			backdrop_path TEXT,
			runtime INTEGER,
			original_language TEXT,
			status TEXT,
			imdb_id TEXT,
			tmdb_id INTEGER,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create movies table: %v", err)
	}

	// Create indexes
	_, err = db.Exec(`CREATE INDEX idx_movies_title ON movies(title)`)
	if err != nil {
		t.Fatalf("Failed to create title index: %v", err)
	}

	_, err = db.Exec(`CREATE INDEX idx_movies_tmdb_id ON movies(tmdb_id)`)
	if err != nil {
		t.Fatalf("Failed to create tmdb_id index: %v", err)
	}

	_, err = db.Exec(`CREATE INDEX idx_movies_imdb_id ON movies(imdb_id)`)
	if err != nil {
		t.Fatalf("Failed to create imdb_id index: %v", err)
	}

	return db
}

// TestMovieCreate verifies movie creation
func TestMovieCreate(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewMovieRepository(db)
	ctx := context.Background()

	movie := &models.Movie{
		ID:               "movie-1",
		Title:            "The Matrix",
		OriginalTitle:    sql.NullString{String: "The Matrix", Valid: true},
		ReleaseDate:      "1999-03-31",
		Genres:           []string{"Action", "Science Fiction"},
		Rating:           sql.NullFloat64{Float64: 8.7, Valid: true},
		Overview:         sql.NullString{String: "A computer hacker learns about the true nature of reality.", Valid: true},
		Runtime:          sql.NullInt64{Int64: 136, Valid: true},
		OriginalLanguage: sql.NullString{String: "en", Valid: true},
		Status:           sql.NullString{String: "Released", Valid: true},
		IMDbID:           sql.NullString{String: "tt0133093", Valid: true},
		TMDbID:           sql.NullInt64{Int64: 603, Valid: true},
	}

	err := repo.Create(ctx, movie)
	if err != nil {
		t.Fatalf("Failed to create movie: %v", err)
	}

	// Verify timestamps were set
	if movie.CreatedAt.IsZero() {
		t.Error("Expected CreatedAt to be set")
	}
	if movie.UpdatedAt.IsZero() {
		t.Error("Expected UpdatedAt to be set")
	}

	// Verify movie was inserted
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM movies WHERE id = ?", "movie-1").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count movies: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 movie, got %d", count)
	}
}

// TestMovieCreateNil verifies nil movie rejection
func TestMovieCreateNil(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewMovieRepository(db)
	ctx := context.Background()

	err := repo.Create(ctx, nil)
	if err == nil {
		t.Fatal("Expected error for nil movie, got nil")
	}
}

// TestMovieFindByID verifies finding movie by ID
func TestMovieFindByID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewMovieRepository(db)
	ctx := context.Background()

	// Create movie
	movie := &models.Movie{
		ID:          "movie-1",
		Title:       "Inception",
		ReleaseDate: "2010-07-16",
		Genres:      []string{"Action", "Thriller"},
		Rating:      sql.NullFloat64{Float64: 8.8, Valid: true},
	}

	err := repo.Create(ctx, movie)
	if err != nil {
		t.Fatalf("Failed to create movie: %v", err)
	}

	// Find movie
	found, err := repo.FindByID(ctx, "movie-1")
	if err != nil {
		t.Fatalf("Failed to find movie: %v", err)
	}

	if found.ID != movie.ID {
		t.Errorf("Expected ID %s, got %s", movie.ID, found.ID)
	}
	if found.Title != movie.Title {
		t.Errorf("Expected title %s, got %s", movie.Title, found.Title)
	}
	if len(found.Genres) != len(movie.Genres) {
		t.Errorf("Expected %d genres, got %d", len(movie.Genres), len(found.Genres))
	}
}

// TestMovieFindByIDNotFound verifies error for non-existent movie
func TestMovieFindByIDNotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewMovieRepository(db)
	ctx := context.Background()

	_, err := repo.FindByID(ctx, "non-existent")
	if err == nil {
		t.Fatal("Expected error for non-existent movie, got nil")
	}
}

// TestMovieFindByTMDbID verifies finding movie by TMDb ID
func TestMovieFindByTMDbID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewMovieRepository(db)
	ctx := context.Background()

	// Create movie
	movie := &models.Movie{
		ID:          "movie-1",
		Title:       "Interstellar",
		ReleaseDate: "2014-11-07",
		Genres:      []string{"Science Fiction", "Drama"},
		TMDbID:      sql.NullInt64{Int64: 157336, Valid: true},
	}

	err := repo.Create(ctx, movie)
	if err != nil {
		t.Fatalf("Failed to create movie: %v", err)
	}

	// Find by TMDb ID
	found, err := repo.FindByTMDbID(ctx, 157336)
	if err != nil {
		t.Fatalf("Failed to find movie by TMDb ID: %v", err)
	}

	if found.ID != movie.ID {
		t.Errorf("Expected ID %s, got %s", movie.ID, found.ID)
	}
	if found.TMDbID.Int64 != movie.TMDbID.Int64 {
		t.Errorf("Expected TMDb ID %d, got %d", movie.TMDbID.Int64, found.TMDbID.Int64)
	}
}

// TestMovieFindByIMDbID verifies finding movie by IMDb ID
func TestMovieFindByIMDbID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewMovieRepository(db)
	ctx := context.Background()

	// Create movie
	movie := &models.Movie{
		ID:          "movie-1",
		Title:       "The Dark Knight",
		ReleaseDate: "2008-07-18",
		Genres:      []string{"Action", "Crime", "Drama"},
		IMDbID:      sql.NullString{String: "tt0468569", Valid: true},
	}

	err := repo.Create(ctx, movie)
	if err != nil {
		t.Fatalf("Failed to create movie: %v", err)
	}

	// Find by IMDb ID
	found, err := repo.FindByIMDbID(ctx, "tt0468569")
	if err != nil {
		t.Fatalf("Failed to find movie by IMDb ID: %v", err)
	}

	if found.ID != movie.ID {
		t.Errorf("Expected ID %s, got %s", movie.ID, found.ID)
	}
	if found.IMDbID.String != movie.IMDbID.String {
		t.Errorf("Expected IMDb ID %s, got %s", movie.IMDbID.String, found.IMDbID.String)
	}
}

// TestMovieUpdate verifies movie update
func TestMovieUpdate(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewMovieRepository(db)
	ctx := context.Background()

	// Create movie
	movie := &models.Movie{
		ID:          "movie-1",
		Title:       "Original Title",
		ReleaseDate: "2020-01-01",
		Genres:      []string{"Drama"},
	}

	err := repo.Create(ctx, movie)
	if err != nil {
		t.Fatalf("Failed to create movie: %v", err)
	}

	// Wait a bit to ensure updated_at changes
	time.Sleep(10 * time.Millisecond)

	// Update movie
	movie.Title = "Updated Title"
	movie.Genres = []string{"Drama", "Thriller"}
	originalUpdatedAt := movie.UpdatedAt

	err = repo.Update(ctx, movie)
	if err != nil {
		t.Fatalf("Failed to update movie: %v", err)
	}

	// Verify updated_at changed
	if !movie.UpdatedAt.After(originalUpdatedAt) {
		t.Error("Expected UpdatedAt to be updated")
	}

	// Find and verify update
	found, err := repo.FindByID(ctx, "movie-1")
	if err != nil {
		t.Fatalf("Failed to find movie: %v", err)
	}

	if found.Title != "Updated Title" {
		t.Errorf("Expected title 'Updated Title', got '%s'", found.Title)
	}
	if len(found.Genres) != 2 {
		t.Errorf("Expected 2 genres, got %d", len(found.Genres))
	}
}

// TestMovieUpdateNotFound verifies error for non-existent movie
func TestMovieUpdateNotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewMovieRepository(db)
	ctx := context.Background()

	movie := &models.Movie{
		ID:          "non-existent",
		Title:       "Test",
		ReleaseDate: "2020-01-01",
		Genres:      []string{},
	}

	err := repo.Update(ctx, movie)
	if err == nil {
		t.Fatal("Expected error for non-existent movie, got nil")
	}
}

// TestMovieDelete verifies movie deletion
func TestMovieDelete(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewMovieRepository(db)
	ctx := context.Background()

	// Create movie
	movie := &models.Movie{
		ID:          "movie-1",
		Title:       "To Be Deleted",
		ReleaseDate: "2020-01-01",
		Genres:      []string{"Drama"},
	}

	err := repo.Create(ctx, movie)
	if err != nil {
		t.Fatalf("Failed to create movie: %v", err)
	}

	// Delete movie
	err = repo.Delete(ctx, "movie-1")
	if err != nil {
		t.Fatalf("Failed to delete movie: %v", err)
	}

	// Verify movie was deleted
	_, err = repo.FindByID(ctx, "movie-1")
	if err == nil {
		t.Fatal("Expected error for deleted movie, got nil")
	}
}

// TestMovieDeleteNotFound verifies error for non-existent movie
func TestMovieDeleteNotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewMovieRepository(db)
	ctx := context.Background()

	err := repo.Delete(ctx, "non-existent")
	if err == nil {
		t.Fatal("Expected error for non-existent movie, got nil")
	}
}

// TestMovieList verifies movie listing with pagination
func TestMovieList(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewMovieRepository(db)
	ctx := context.Background()

	// Create multiple movies
	for i := 1; i <= 5; i++ {
		movie := &models.Movie{
			ID:          sql.NullString{String: string(rune('0' + i)), Valid: true}.String,
			Title:       sql.NullString{String: "Movie " + string(rune('0'+i)), Valid: true}.String,
			ReleaseDate: "2020-01-01",
			Genres:      []string{"Drama"},
		}
		movie.ID = "movie-" + string(rune('0'+i))
		movie.Title = "Movie " + string(rune('0'+i))

		err := repo.Create(ctx, movie)
		if err != nil {
			t.Fatalf("Failed to create movie %d: %v", i, err)
		}
	}

	// List movies
	params := NewListParams()
	params.Page = 1
	params.PageSize = 3

	movies, pagination, err := repo.List(ctx, params)
	if err != nil {
		t.Fatalf("Failed to list movies: %v", err)
	}

	if len(movies) != 3 {
		t.Errorf("Expected 3 movies, got %d", len(movies))
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

// TestMovieListEmpty verifies empty list handling
func TestMovieListEmpty(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewMovieRepository(db)
	ctx := context.Background()

	params := NewListParams()
	movies, pagination, err := repo.List(ctx, params)
	if err != nil {
		t.Fatalf("Failed to list movies: %v", err)
	}

	if len(movies) != 0 {
		t.Errorf("Expected 0 movies, got %d", len(movies))
	}
	if pagination.TotalResults != 0 {
		t.Errorf("Expected total results 0, got %d", pagination.TotalResults)
	}
}

// TestMovieListWithSorting verifies list sorting
func TestMovieListWithSorting(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewMovieRepository(db)
	ctx := context.Background()

	// Create movies with different ratings
	movies := []*models.Movie{
		{
			ID:          "movie-1",
			Title:       "Movie A",
			ReleaseDate: "2020-01-01",
			Genres:      []string{},
			Rating:      sql.NullFloat64{Float64: 7.5, Valid: true},
		},
		{
			ID:          "movie-2",
			Title:       "Movie B",
			ReleaseDate: "2020-01-01",
			Genres:      []string{},
			Rating:      sql.NullFloat64{Float64: 9.0, Valid: true},
		},
		{
			ID:          "movie-3",
			Title:       "Movie C",
			ReleaseDate: "2020-01-01",
			Genres:      []string{},
			Rating:      sql.NullFloat64{Float64: 8.0, Valid: true},
		},
	}

	for _, movie := range movies {
		err := repo.Create(ctx, movie)
		if err != nil {
			t.Fatalf("Failed to create movie: %v", err)
		}
	}

	// List with rating sort descending
	params := NewListParams()
	params.SortBy = "rating"
	params.SortOrder = "desc"

	result, _, err := repo.List(ctx, params)
	if err != nil {
		t.Fatalf("Failed to list movies: %v", err)
	}

	if len(result) != 3 {
		t.Fatalf("Expected 3 movies, got %d", len(result))
	}

	// Verify sort order
	if result[0].Rating.Float64 != 9.0 {
		t.Errorf("Expected first movie rating 9.0, got %f", result[0].Rating.Float64)
	}
}

// TestMovieSearchByTitle verifies search functionality
func TestMovieSearchByTitle(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewMovieRepository(db)
	ctx := context.Background()

	// Create movies
	movies := []*models.Movie{
		{
			ID:          "movie-1",
			Title:       "The Matrix",
			ReleaseDate: "1999-03-31",
			Genres:      []string{},
		},
		{
			ID:          "movie-2",
			Title:       "The Matrix Reloaded",
			ReleaseDate: "2003-05-15",
			Genres:      []string{},
		},
		{
			ID:          "movie-3",
			Title:       "Inception",
			ReleaseDate: "2010-07-16",
			Genres:      []string{},
		},
	}

	for _, movie := range movies {
		err := repo.Create(ctx, movie)
		if err != nil {
			t.Fatalf("Failed to create movie: %v", err)
		}
	}

	// Search for "Matrix"
	params := NewListParams()
	result, pagination, err := repo.SearchByTitle(ctx, "Matrix", params)
	if err != nil {
		t.Fatalf("Failed to search movies: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("Expected 2 movies, got %d", len(result))
	}
	if pagination.TotalResults != 2 {
		t.Errorf("Expected total results 2, got %d", pagination.TotalResults)
	}

	// Verify both results contain "Matrix"
	for _, movie := range result {
		if movie.Title != "The Matrix" && movie.Title != "The Matrix Reloaded" {
			t.Errorf("Unexpected movie in results: %s", movie.Title)
		}
	}
}

// TestMovieGenresSerialization verifies genres JSON handling
func TestMovieGenresSerialization(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewMovieRepository(db)
	ctx := context.Background()

	// Create movie with multiple genres
	movie := &models.Movie{
		ID:          "movie-1",
		Title:       "Test Movie",
		ReleaseDate: "2020-01-01",
		Genres:      []string{"Action", "Adventure", "Science Fiction"},
	}

	err := repo.Create(ctx, movie)
	if err != nil {
		t.Fatalf("Failed to create movie: %v", err)
	}

	// Retrieve and verify genres
	found, err := repo.FindByID(ctx, "movie-1")
	if err != nil {
		t.Fatalf("Failed to find movie: %v", err)
	}

	if len(found.Genres) != 3 {
		t.Errorf("Expected 3 genres, got %d", len(found.Genres))
	}

	expectedGenres := map[string]bool{
		"Action":            true,
		"Adventure":         true,
		"Science Fiction":   true,
	}

	for _, genre := range found.Genres {
		if !expectedGenres[genre] {
			t.Errorf("Unexpected genre: %s", genre)
		}
	}
}

// TestMovieEmptyGenres verifies empty genres handling
func TestMovieEmptyGenres(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewMovieRepository(db)
	ctx := context.Background()

	// Create movie with no genres
	movie := &models.Movie{
		ID:          "movie-1",
		Title:       "Test Movie",
		ReleaseDate: "2020-01-01",
		Genres:      []string{},
	}

	err := repo.Create(ctx, movie)
	if err != nil {
		t.Fatalf("Failed to create movie: %v", err)
	}

	// Retrieve and verify genres
	found, err := repo.FindByID(ctx, "movie-1")
	if err != nil {
		t.Fatalf("Failed to find movie: %v", err)
	}

	if found.Genres == nil {
		t.Error("Expected empty slice for genres, got nil")
	}
	if len(found.Genres) != 0 {
		t.Errorf("Expected 0 genres, got %d", len(found.Genres))
	}
}

// TestMovieUpsertCreate verifies upsert creates new movie when TMDb ID not found
func TestMovieUpsertCreate(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewMovieRepository(db)
	ctx := context.Background()

	movie := &models.Movie{
		ID:          "movie-1",
		Title:       "New Movie",
		ReleaseDate: "2023-01-01",
		Genres:      []string{"Action"},
		TMDbID:      sql.NullInt64{Int64: 99999, Valid: true},
	}

	err := repo.Upsert(ctx, movie)
	if err != nil {
		t.Fatalf("Failed to upsert movie: %v", err)
	}

	// Verify movie was created
	found, err := repo.FindByTMDbID(ctx, 99999)
	if err != nil {
		t.Fatalf("Failed to find movie: %v", err)
	}

	if found.Title != "New Movie" {
		t.Errorf("Expected title 'New Movie', got '%s'", found.Title)
	}
}

// TestMovieUpsertUpdate verifies upsert updates existing movie by TMDb ID
func TestMovieUpsertUpdate(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewMovieRepository(db)
	ctx := context.Background()

	// Create initial movie
	movie := &models.Movie{
		ID:          "movie-1",
		Title:       "Original Title",
		ReleaseDate: "2023-01-01",
		Genres:      []string{"Action"},
		TMDbID:      sql.NullInt64{Int64: 12345, Valid: true},
	}

	err := repo.Create(ctx, movie)
	if err != nil {
		t.Fatalf("Failed to create movie: %v", err)
	}

	// Upsert with same TMDb ID but different data
	updatedMovie := &models.Movie{
		ID:          "movie-new",
		Title:       "Updated Title",
		ReleaseDate: "2023-06-01",
		Genres:      []string{"Action", "Thriller"},
		TMDbID:      sql.NullInt64{Int64: 12345, Valid: true},
	}

	err = repo.Upsert(ctx, updatedMovie)
	if err != nil {
		t.Fatalf("Failed to upsert movie: %v", err)
	}

	// Verify movie was updated with original ID
	found, err := repo.FindByTMDbID(ctx, 12345)
	if err != nil {
		t.Fatalf("Failed to find movie: %v", err)
	}

	if found.ID != "movie-1" {
		t.Errorf("Expected ID 'movie-1', got '%s'", found.ID)
	}
	if found.Title != "Updated Title" {
		t.Errorf("Expected title 'Updated Title', got '%s'", found.Title)
	}
}

// TestMovieUpsertNoTMDbID verifies upsert creates when no TMDb ID is provided
func TestMovieUpsertNoTMDbID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewMovieRepository(db)
	ctx := context.Background()

	movie := &models.Movie{
		ID:          "movie-1",
		Title:       "Movie Without TMDb",
		ReleaseDate: "2023-01-01",
		Genres:      []string{"Drama"},
		// No TMDb ID set
	}

	err := repo.Upsert(ctx, movie)
	if err != nil {
		t.Fatalf("Failed to upsert movie: %v", err)
	}

	// Verify movie was created
	found, err := repo.FindByID(ctx, "movie-1")
	if err != nil {
		t.Fatalf("Failed to find movie: %v", err)
	}

	if found.Title != "Movie Without TMDb" {
		t.Errorf("Expected title 'Movie Without TMDb', got '%s'", found.Title)
	}
}

// TestMovieUpsertNil verifies nil movie rejection
func TestMovieUpsertNil(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewMovieRepository(db)
	ctx := context.Background()

	err := repo.Upsert(ctx, nil)
	if err == nil {
		t.Fatal("Expected error for nil movie, got nil")
	}
}

// TestMovieFindByTMDbIDNotFound verifies error for non-existent TMDb ID
func TestMovieFindByTMDbIDNotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewMovieRepository(db)
	ctx := context.Background()

	_, err := repo.FindByTMDbID(ctx, 99999)
	if err == nil {
		t.Fatal("Expected error for non-existent TMDb ID, got nil")
	}
}

// TestMovieFindByIMDbIDNotFound verifies error for non-existent IMDb ID
func TestMovieFindByIMDbIDNotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewMovieRepository(db)
	ctx := context.Background()

	_, err := repo.FindByIMDbID(ctx, "tt9999999")
	if err == nil {
		t.Fatal("Expected error for non-existent IMDb ID, got nil")
	}
}

// setupTestDBWithFTS creates an in-memory database with movies table and FTS5 virtual table
func setupTestDBWithFTS(t *testing.T) *sql.DB {
	db := setupTestDB(t)

	// Create FTS5 virtual table for full-text search
	_, err := db.Exec(`
		CREATE VIRTUAL TABLE movies_fts USING fts5(
			title,
			original_title,
			overview,
			content='movies',
			content_rowid='rowid'
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create FTS5 table: %v", err)
	}

	// Create triggers to keep FTS in sync
	_, err = db.Exec(`
		CREATE TRIGGER movies_ai AFTER INSERT ON movies BEGIN
			INSERT INTO movies_fts(rowid, title, original_title, overview)
			VALUES (new.rowid, new.title, new.original_title, new.overview);
		END
	`)
	if err != nil {
		t.Fatalf("Failed to create FTS insert trigger: %v", err)
	}

	_, err = db.Exec(`
		CREATE TRIGGER movies_ad AFTER DELETE ON movies BEGIN
			INSERT INTO movies_fts(movies_fts, rowid, title, original_title, overview)
			VALUES ('delete', old.rowid, old.title, old.original_title, old.overview);
		END
	`)
	if err != nil {
		t.Fatalf("Failed to create FTS delete trigger: %v", err)
	}

	_, err = db.Exec(`
		CREATE TRIGGER movies_au AFTER UPDATE ON movies BEGIN
			INSERT INTO movies_fts(movies_fts, rowid, title, original_title, overview)
			VALUES ('delete', old.rowid, old.title, old.original_title, old.overview);
			INSERT INTO movies_fts(rowid, title, original_title, overview)
			VALUES (new.rowid, new.title, new.original_title, new.overview);
		END
	`)
	if err != nil {
		t.Fatalf("Failed to create FTS update trigger: %v", err)
	}

	return db
}

// TestMovieFullTextSearchEmptyQuery verifies empty query falls back to List
func TestMovieFullTextSearchEmptyQuery(t *testing.T) {
	db := setupTestDBWithFTS(t)
	defer db.Close()

	repo := NewMovieRepository(db)
	ctx := context.Background()

	// Create some movies
	movies := []*models.Movie{
		{ID: "movie-1", Title: "The Matrix", ReleaseDate: "1999-03-31", Genres: []string{"Action"}},
		{ID: "movie-2", Title: "Inception", ReleaseDate: "2010-07-16", Genres: []string{"Sci-Fi"}},
	}

	for _, m := range movies {
		err := repo.Create(ctx, m)
		if err != nil {
			t.Fatalf("Failed to create movie: %v", err)
		}
	}

	// Empty query should fall back to List
	params := NewListParams()
	result, pagination, err := repo.FullTextSearch(ctx, "", params)
	if err != nil {
		t.Fatalf("Failed to search movies: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("Expected 2 movies, got %d", len(result))
	}
	if pagination.TotalResults != 2 {
		t.Errorf("Expected total results 2, got %d", pagination.TotalResults)
	}
}

// TestMovieFullTextSearchByTitle verifies FTS search by title
func TestMovieFullTextSearchByTitle(t *testing.T) {
	db := setupTestDBWithFTS(t)
	defer db.Close()

	repo := NewMovieRepository(db)
	ctx := context.Background()

	// Create movies
	movies := []*models.Movie{
		{
			ID:          "movie-1",
			Title:       "The Matrix",
			ReleaseDate: "1999-03-31",
			Genres:      []string{"Action"},
			Overview:    sql.NullString{String: "A computer hacker discovers reality is a simulation.", Valid: true},
		},
		{
			ID:          "movie-2",
			Title:       "Matrix Reloaded",
			ReleaseDate: "2003-05-15",
			Genres:      []string{"Action"},
			Overview:    sql.NullString{String: "Neo continues his fight against the machines.", Valid: true},
		},
		{
			ID:          "movie-3",
			Title:       "Inception",
			ReleaseDate: "2010-07-16",
			Genres:      []string{"Sci-Fi"},
			Overview:    sql.NullString{String: "A thief who steals corporate secrets through dream-sharing.", Valid: true},
		},
	}

	for _, m := range movies {
		err := repo.Create(ctx, m)
		if err != nil {
			t.Fatalf("Failed to create movie: %v", err)
		}
	}

	// Search for "Matrix"
	params := NewListParams()
	result, pagination, err := repo.FullTextSearch(ctx, "Matrix", params)
	if err != nil {
		t.Fatalf("Failed to search movies: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("Expected 2 movies matching 'Matrix', got %d", len(result))
	}
	if pagination.TotalResults != 2 {
		t.Errorf("Expected total results 2, got %d", pagination.TotalResults)
	}
}

// TestMovieFullTextSearchByOverview verifies FTS search by overview content
func TestMovieFullTextSearchByOverview(t *testing.T) {
	db := setupTestDBWithFTS(t)
	defer db.Close()

	repo := NewMovieRepository(db)
	ctx := context.Background()

	// Create movies
	movies := []*models.Movie{
		{
			ID:          "movie-1",
			Title:       "The Matrix",
			ReleaseDate: "1999-03-31",
			Genres:      []string{"Action"},
			Overview:    sql.NullString{String: "A computer simulation discovers reality.", Valid: true},
		},
		{
			ID:          "movie-2",
			Title:       "Dark City",
			ReleaseDate: "1998-02-27",
			Genres:      []string{"Thriller"},
			Overview:    sql.NullString{String: "A man wakes up in a strange simulation world.", Valid: true},
		},
		{
			ID:          "movie-3",
			Title:       "Inception",
			ReleaseDate: "2010-07-16",
			Genres:      []string{"Sci-Fi"},
			Overview:    sql.NullString{String: "A thief who steals corporate secrets through dreams.", Valid: true},
		},
	}

	for _, m := range movies {
		err := repo.Create(ctx, m)
		if err != nil {
			t.Fatalf("Failed to create movie: %v", err)
		}
	}

	// Search for "simulation" - should match overview content
	params := NewListParams()
	result, _, err := repo.FullTextSearch(ctx, "simulation", params)
	if err != nil {
		t.Fatalf("Failed to search movies: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("Expected 2 movies matching 'simulation' in overview, got %d", len(result))
	}
}

// TestMovieFullTextSearchWithPagination verifies FTS pagination
func TestMovieFullTextSearchWithPagination(t *testing.T) {
	db := setupTestDBWithFTS(t)
	defer db.Close()

	repo := NewMovieRepository(db)
	ctx := context.Background()

	// Create 5 movies with "Action" in title
	for i := 1; i <= 5; i++ {
		movie := &models.Movie{
			ID:          "movie-" + string(rune('0'+i)),
			Title:       "Action Movie " + string(rune('0'+i)),
			ReleaseDate: "2020-01-01",
			Genres:      []string{"Action"},
		}
		err := repo.Create(ctx, movie)
		if err != nil {
			t.Fatalf("Failed to create movie: %v", err)
		}
	}

	// Search with pagination
	params := NewListParams()
	params.Page = 1
	params.PageSize = 2

	result, pagination, err := repo.FullTextSearch(ctx, "Action", params)
	if err != nil {
		t.Fatalf("Failed to search movies: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("Expected 2 movies on page 1, got %d", len(result))
	}
	if pagination.TotalResults != 5 {
		t.Errorf("Expected total results 5, got %d", pagination.TotalResults)
	}
	if pagination.TotalPages != 3 {
		t.Errorf("Expected 3 total pages, got %d", pagination.TotalPages)
	}
}

// TestMovieFullTextSearchNoResults verifies empty results handling
func TestMovieFullTextSearchNoResults(t *testing.T) {
	db := setupTestDBWithFTS(t)
	defer db.Close()

	repo := NewMovieRepository(db)
	ctx := context.Background()

	// Create a movie
	movie := &models.Movie{
		ID:          "movie-1",
		Title:       "The Matrix",
		ReleaseDate: "1999-03-31",
		Genres:      []string{"Action"},
	}
	err := repo.Create(ctx, movie)
	if err != nil {
		t.Fatalf("Failed to create movie: %v", err)
	}

	// Search for non-existent term
	params := NewListParams()
	result, pagination, err := repo.FullTextSearch(ctx, "NonExistentTerm", params)
	if err != nil {
		t.Fatalf("Failed to search movies: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("Expected 0 movies, got %d", len(result))
	}
	if pagination.TotalResults != 0 {
		t.Errorf("Expected total results 0, got %d", pagination.TotalResults)
	}
}
