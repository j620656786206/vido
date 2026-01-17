package services

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
	"github.com/vido/api/internal/tmdb"

	_ "modernc.org/sqlite"
)

// setupTestDB creates a temporary SQLite database with the required schema
func setupTestDB(t *testing.T) *sql.DB {
	// Use a temp file instead of :memory: for better FTS5 external content support
	tmpFile, err := os.CreateTemp("", "test_library_*.db")
	require.NoError(t, err)
	tmpFile.Close()

	// Register cleanup
	t.Cleanup(func() {
		os.Remove(tmpFile.Name())
	})

	db, err := sql.Open("sqlite", tmpFile.Name()+"?_pragma=foreign_keys(1)")
	require.NoError(t, err)

	// Create movies table with FTS support
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS movies (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			original_title TEXT,
			release_date TEXT,
			genres TEXT DEFAULT '[]',
			rating REAL,
			vote_average REAL,
			vote_count INTEGER,
			popularity REAL,
			overview TEXT,
			poster_path TEXT,
			backdrop_path TEXT,
			runtime INTEGER,
			original_language TEXT,
			status TEXT,
			imdb_id TEXT,
			tmdb_id INTEGER UNIQUE,
			credits TEXT,
			production_countries TEXT,
			spoken_languages TEXT,
			file_path TEXT,
			file_size INTEGER,
			parse_status TEXT DEFAULT 'pending',
			metadata_source TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	require.NoError(t, err)

	// Create movies FTS virtual table with external content
	_, err = db.Exec(`
		CREATE VIRTUAL TABLE IF NOT EXISTS movies_fts USING fts5(
			title,
			original_title,
			overview,
			content='movies',
			content_rowid='rowid'
		)
	`)
	require.NoError(t, err)

	// Create triggers to keep FTS in sync
	_, err = db.Exec(`
		CREATE TRIGGER movies_fts_ai AFTER INSERT ON movies BEGIN
			INSERT INTO movies_fts(rowid, title, original_title, overview)
			VALUES (NEW.rowid, NEW.title, NEW.original_title, NEW.overview);
		END
	`)
	require.NoError(t, err)

	_, err = db.Exec(`
		CREATE TRIGGER movies_fts_ad AFTER DELETE ON movies BEGIN
			INSERT INTO movies_fts(movies_fts, rowid, title, original_title, overview)
			VALUES ('delete', OLD.rowid, OLD.title, OLD.original_title, OLD.overview);
		END
	`)
	require.NoError(t, err)

	_, err = db.Exec(`
		CREATE TRIGGER movies_fts_au AFTER UPDATE ON movies BEGIN
			INSERT INTO movies_fts(movies_fts, rowid, title, original_title, overview)
			VALUES ('delete', OLD.rowid, OLD.title, OLD.original_title, OLD.overview);
			INSERT INTO movies_fts(rowid, title, original_title, overview)
			VALUES (NEW.rowid, NEW.title, NEW.original_title, NEW.overview);
		END
	`)
	require.NoError(t, err)

	// Create series table with FTS support
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS series (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			original_title TEXT,
			first_air_date TEXT,
			last_air_date TEXT,
			genres TEXT DEFAULT '[]',
			rating REAL,
			vote_average REAL,
			vote_count INTEGER,
			popularity REAL,
			overview TEXT,
			poster_path TEXT,
			backdrop_path TEXT,
			number_of_seasons INTEGER,
			number_of_episodes INTEGER,
			status TEXT,
			original_language TEXT,
			imdb_id TEXT,
			tmdb_id INTEGER UNIQUE,
			in_production INTEGER,
			credits TEXT,
			seasons TEXT,
			networks TEXT,
			file_path TEXT,
			parse_status TEXT DEFAULT 'pending',
			metadata_source TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	require.NoError(t, err)

	// Create series FTS virtual table with external content
	_, err = db.Exec(`
		CREATE VIRTUAL TABLE IF NOT EXISTS series_fts USING fts5(
			title,
			original_title,
			overview,
			content='series',
			content_rowid='rowid'
		)
	`)
	require.NoError(t, err)

	// Create triggers for series FTS
	_, err = db.Exec(`
		CREATE TRIGGER series_fts_ai AFTER INSERT ON series BEGIN
			INSERT INTO series_fts(rowid, title, original_title, overview)
			VALUES (NEW.rowid, NEW.title, NEW.original_title, NEW.overview);
		END
	`)
	require.NoError(t, err)

	_, err = db.Exec(`
		CREATE TRIGGER series_fts_ad AFTER DELETE ON series BEGIN
			INSERT INTO series_fts(series_fts, rowid, title, original_title, overview)
			VALUES ('delete', OLD.rowid, OLD.title, OLD.original_title, OLD.overview);
		END
	`)
	require.NoError(t, err)

	_, err = db.Exec(`
		CREATE TRIGGER series_fts_au AFTER UPDATE ON series BEGIN
			INSERT INTO series_fts(series_fts, rowid, title, original_title, overview)
			VALUES ('delete', OLD.rowid, OLD.title, OLD.original_title, OLD.overview);
			INSERT INTO series_fts(rowid, title, original_title, overview)
			VALUES (NEW.rowid, NEW.title, NEW.original_title, NEW.overview);
		END
	`)
	require.NoError(t, err)

	// Create episodes table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS episodes (
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
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(series_id, season_number, episode_number),
			FOREIGN KEY (series_id) REFERENCES series(id) ON DELETE CASCADE
		)
	`)
	require.NoError(t, err)

	return db
}

func TestLibraryService_SaveMovieFromTMDb(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	movieRepo := repository.NewMovieRepository(db)
	seriesRepo := repository.NewSeriesRepository(db)
	episodeRepo := repository.NewEpisodeRepository(db)

	service := NewLibraryService(movieRepo, seriesRepo, episodeRepo)

	posterPath := "/pB8BM7pdSp6B6Ih7QZ4DrQ3PmJK.jpg"
	backdropPath := "/fCayJrkfRaCRCTh8GqN30f8oyQF.jpg"

	tmdbMovie := &tmdb.MovieDetails{
		Movie: tmdb.Movie{
			ID:               550,
			Title:            "Fight Club",
			OriginalTitle:    "Fight Club",
			Overview:         "A ticking-time-bomb insomniac and a slippery soap salesman channel primal male aggression into a shocking new form of therapy.",
			ReleaseDate:      "1999-10-15",
			PosterPath:       &posterPath,
			BackdropPath:     &backdropPath,
			VoteAverage:      8.4,
			VoteCount:        26280,
			Popularity:       61.416,
			OriginalLanguage: "en",
		},
		Runtime: 139,
		Status:  "Released",
		ImdbID:  "tt0137523",
		Genres: []tmdb.Genre{
			{ID: 18, Name: "Drama"},
			{ID: 53, Name: "Thriller"},
		},
	}

	ctx := context.Background()

	t.Run("save new movie", func(t *testing.T) {
		movie, err := service.SaveMovieFromTMDb(ctx, tmdbMovie, "/movies/fight_club.mkv")
		require.NoError(t, err)
		require.NotNil(t, movie)

		assert.NotEmpty(t, movie.ID)
		assert.Equal(t, "Fight Club", movie.Title)
		assert.Equal(t, int64(550), movie.TMDbID.Int64)
		assert.Equal(t, "tt0137523", movie.IMDbID.String)
		assert.Equal(t, "/movies/fight_club.mkv", movie.FilePath.String)
		assert.Equal(t, []string{"Drama", "Thriller"}, movie.Genres)
		assert.Equal(t, models.ParseStatusSuccess, movie.ParseStatus)

		// Verify it was saved to database
		retrieved, err := service.GetMovieByTMDbID(ctx, 550)
		require.NoError(t, err)
		assert.Equal(t, movie.ID, retrieved.ID)
		assert.Equal(t, "Fight Club", retrieved.Title)
	})

	t.Run("upsert existing movie", func(t *testing.T) {
		// Update the TMDb movie data
		tmdbMovie.Movie.Title = "Fight Club (Updated)"
		tmdbMovie.Runtime = 140

		movie, err := service.SaveMovieFromTMDb(ctx, tmdbMovie, "/movies/fight_club_updated.mkv")
		require.NoError(t, err)
		require.NotNil(t, movie)

		// Should have same TMDb ID
		assert.Equal(t, int64(550), movie.TMDbID.Int64)

		// Verify there's still only one movie with this TMDb ID
		retrieved, err := service.GetMovieByTMDbID(ctx, 550)
		require.NoError(t, err)
		assert.Equal(t, "Fight Club (Updated)", retrieved.Title)
	})

	t.Run("nil tmdb movie returns error", func(t *testing.T) {
		movie, err := service.SaveMovieFromTMDb(ctx, nil, "")
		assert.Error(t, err)
		assert.Nil(t, movie)
	})
}

func TestLibraryService_SaveSeriesFromTMDb(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	movieRepo := repository.NewMovieRepository(db)
	seriesRepo := repository.NewSeriesRepository(db)
	episodeRepo := repository.NewEpisodeRepository(db)

	service := NewLibraryService(movieRepo, seriesRepo, episodeRepo)

	posterPath := "/ggFHVNu6YYI5L9pCfOacjizRGt.jpg"
	backdropPath := "/tsRy63Mu5cu8etL1X7ZLyf7UP1M.jpg"

	tmdbSeries := &tmdb.TVShowDetails{
		TVShow: tmdb.TVShow{
			ID:               1396,
			Name:             "Breaking Bad",
			OriginalName:     "Breaking Bad",
			Overview:         "When Walter White, a New Mexico chemistry teacher, is diagnosed with Stage III cancer.",
			FirstAirDate:     "2008-01-20",
			PosterPath:       &posterPath,
			BackdropPath:     &backdropPath,
			VoteAverage:      8.9,
			VoteCount:        12345,
			Popularity:       369.594,
			OriginalLanguage: "en",
		},
		LastAirDate:      "2013-09-29",
		NumberOfEpisodes: 62,
		NumberOfSeasons:  5,
		Status:           "Ended",
		InProduction:     false,
		Genres: []tmdb.Genre{
			{ID: 18, Name: "Drama"},
			{ID: 80, Name: "Crime"},
		},
	}

	ctx := context.Background()

	t.Run("save new series", func(t *testing.T) {
		series, err := service.SaveSeriesFromTMDb(ctx, tmdbSeries, "/series/breaking_bad")
		require.NoError(t, err)
		require.NotNil(t, series)

		assert.NotEmpty(t, series.ID)
		assert.Equal(t, "Breaking Bad", series.Title)
		assert.Equal(t, int64(1396), series.TMDbID.Int64)
		assert.Equal(t, "/series/breaking_bad", series.FilePath.String)
		assert.Equal(t, []string{"Drama", "Crime"}, series.Genres)
		assert.Equal(t, models.ParseStatusSuccess, series.ParseStatus)

		// Verify it was saved to database
		retrieved, err := service.GetSeriesByTMDbID(ctx, 1396)
		require.NoError(t, err)
		assert.Equal(t, series.ID, retrieved.ID)
		assert.Equal(t, "Breaking Bad", retrieved.Title)
	})

	t.Run("upsert existing series", func(t *testing.T) {
		// Update the TMDb series data
		tmdbSeries.TVShow.Name = "Breaking Bad (Updated)"

		series, err := service.SaveSeriesFromTMDb(ctx, tmdbSeries, "/series/breaking_bad_updated")
		require.NoError(t, err)
		require.NotNil(t, series)

		// Should have same TMDb ID
		assert.Equal(t, int64(1396), series.TMDbID.Int64)

		// Verify there's still only one series with this TMDb ID
		retrieved, err := service.GetSeriesByTMDbID(ctx, 1396)
		require.NoError(t, err)
		assert.Equal(t, "Breaking Bad (Updated)", retrieved.Title)
	})

	t.Run("nil tmdb series returns error", func(t *testing.T) {
		series, err := service.SaveSeriesFromTMDb(ctx, nil, "")
		assert.Error(t, err)
		assert.Nil(t, series)
	})
}

func TestLibraryService_SearchLibrary(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	movieRepo := repository.NewMovieRepository(db)
	seriesRepo := repository.NewSeriesRepository(db)
	episodeRepo := repository.NewEpisodeRepository(db)

	service := NewLibraryService(movieRepo, seriesRepo, episodeRepo)

	ctx := context.Background()

	// Insert test movies
	posterPath := "/poster.jpg"
	movies := []*tmdb.MovieDetails{
		{
			Movie: tmdb.Movie{
				ID:          1,
				Title:       "The Dark Knight",
				Overview:    "Batman raises the stakes in his war on crime.",
				ReleaseDate: "2008-07-18",
				PosterPath:  &posterPath,
				VoteAverage: 9.0,
			},
			Genres: []tmdb.Genre{{ID: 28, Name: "Action"}},
		},
		{
			Movie: tmdb.Movie{
				ID:          2,
				Title:       "Dark Waters",
				Overview:    "A corporate defense attorney takes on an environmental lawsuit.",
				ReleaseDate: "2019-11-22",
				PosterPath:  &posterPath,
				VoteAverage: 7.5,
			},
			Genres: []tmdb.Genre{{ID: 18, Name: "Drama"}},
		},
		{
			Movie: tmdb.Movie{
				ID:          3,
				Title:       "Inception",
				Overview:    "A thief who steals corporate secrets through dream-sharing technology.",
				ReleaseDate: "2010-07-16",
				PosterPath:  &posterPath,
				VoteAverage: 8.8,
			},
			Genres: []tmdb.Genre{{ID: 28, Name: "Action"}},
		},
	}

	for _, m := range movies {
		_, err := service.SaveMovieFromTMDb(ctx, m, "")
		require.NoError(t, err)
	}

	// Insert test series
	seriesList := []*tmdb.TVShowDetails{
		{
			TVShow: tmdb.TVShow{
				ID:           100,
				Name:         "Dark",
				Overview:     "A family saga with a supernatural twist.",
				FirstAirDate: "2017-12-01",
				PosterPath:   &posterPath,
				VoteAverage:  8.8,
			},
			Genres: []tmdb.Genre{{ID: 18, Name: "Drama"}},
		},
		{
			TVShow: tmdb.TVShow{
				ID:           101,
				Name:         "Breaking Bad",
				Overview:     "A high school chemistry teacher turned methamphetamine manufacturer.",
				FirstAirDate: "2008-01-20",
				PosterPath:   &posterPath,
				VoteAverage:  9.5,
			},
			Genres: []tmdb.Genre{{ID: 18, Name: "Drama"}},
		},
	}

	for _, s := range seriesList {
		_, err := service.SaveSeriesFromTMDb(ctx, s, "")
		require.NoError(t, err)
	}

	t.Run("search for 'dark' returns movies and series", func(t *testing.T) {
		results, err := service.SearchLibrary(ctx, "dark", repository.ListParams{
			PageSize: 10,
			Page:     1,
		})
		require.NoError(t, err)
		require.NotNil(t, results)

		// Should find "The Dark Knight", "Dark Waters" (movies) and "Dark" (series)
		assert.GreaterOrEqual(t, len(results.Results), 3)

		// Count movies and series
		movieCount := 0
		seriesCount := 0
		for _, r := range results.Results {
			if r.Type == "movie" {
				movieCount++
			} else if r.Type == "series" {
				seriesCount++
			}
		}

		assert.GreaterOrEqual(t, movieCount, 2, "Should find at least 2 dark movies")
		assert.GreaterOrEqual(t, seriesCount, 1, "Should find at least 1 dark series")
	})

	t.Run("search for 'batman' returns The Dark Knight", func(t *testing.T) {
		results, err := service.SearchLibrary(ctx, "batman", repository.ListParams{
			PageSize: 10,
			Page:     1,
		})
		require.NoError(t, err)

		found := false
		for _, r := range results.Results {
			if r.Type == "movie" && r.Movie != nil && r.Movie.Title == "The Dark Knight" {
				found = true
				break
			}
		}
		assert.True(t, found, "Should find The Dark Knight when searching for 'batman'")
	})

	t.Run("search for 'chemistry' returns Breaking Bad", func(t *testing.T) {
		results, err := service.SearchLibrary(ctx, "chemistry", repository.ListParams{
			PageSize: 10,
			Page:     1,
		})
		require.NoError(t, err)

		found := false
		for _, r := range results.Results {
			if r.Type == "series" && r.Series != nil && r.Series.Title == "Breaking Bad" {
				found = true
				break
			}
		}
		assert.True(t, found, "Should find Breaking Bad when searching for 'chemistry'")
	})

	t.Run("search with no results", func(t *testing.T) {
		results, err := service.SearchLibrary(ctx, "zzzznonexistent", repository.ListParams{
			PageSize: 10,
			Page:     1,
		})
		require.NoError(t, err)
		require.NotNil(t, results)
		assert.Equal(t, 0, len(results.Results))
		assert.Equal(t, 0, results.TotalCount)
	})

	t.Run("search performance under 500ms", func(t *testing.T) {
		// NFR-SC8: Search should complete within 500ms
		start := time.Now()
		_, err := service.SearchLibrary(ctx, "dark", repository.ListParams{
			PageSize: 20,
			Page:     1,
		})
		elapsed := time.Since(start)

		require.NoError(t, err)
		assert.Less(t, elapsed.Milliseconds(), int64(500), "Search should complete within 500ms")
	})
}

func TestLibraryService_GetMovieByID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	movieRepo := repository.NewMovieRepository(db)
	seriesRepo := repository.NewSeriesRepository(db)
	episodeRepo := repository.NewEpisodeRepository(db)

	service := NewLibraryService(movieRepo, seriesRepo, episodeRepo)

	ctx := context.Background()

	// Save a movie first
	posterPath := "/poster.jpg"
	tmdbMovie := &tmdb.MovieDetails{
		Movie: tmdb.Movie{
			ID:          999,
			Title:       "Test Movie",
			ReleaseDate: "2023-01-01",
			PosterPath:  &posterPath,
		},
	}

	saved, err := service.SaveMovieFromTMDb(ctx, tmdbMovie, "")
	require.NoError(t, err)

	t.Run("get existing movie by ID", func(t *testing.T) {
		movie, err := service.GetMovieByID(ctx, saved.ID)
		require.NoError(t, err)
		require.NotNil(t, movie)
		assert.Equal(t, saved.ID, movie.ID)
		assert.Equal(t, "Test Movie", movie.Title)
	})

	t.Run("get non-existent movie returns error", func(t *testing.T) {
		movie, err := service.GetMovieByID(ctx, "non-existent-id")
		assert.Error(t, err)
		assert.Nil(t, movie)
	})

	t.Run("empty ID returns error", func(t *testing.T) {
		movie, err := service.GetMovieByID(ctx, "")
		assert.Error(t, err)
		assert.Nil(t, movie)
	})
}

func TestLibraryService_GetSeriesByID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	movieRepo := repository.NewMovieRepository(db)
	seriesRepo := repository.NewSeriesRepository(db)
	episodeRepo := repository.NewEpisodeRepository(db)

	service := NewLibraryService(movieRepo, seriesRepo, episodeRepo)

	ctx := context.Background()

	// Save a series first
	posterPath := "/poster.jpg"
	tmdbSeries := &tmdb.TVShowDetails{
		TVShow: tmdb.TVShow{
			ID:           888,
			Name:         "Test Series",
			FirstAirDate: "2023-01-01",
			PosterPath:   &posterPath,
		},
	}

	saved, err := service.SaveSeriesFromTMDb(ctx, tmdbSeries, "")
	require.NoError(t, err)

	t.Run("get existing series by ID", func(t *testing.T) {
		series, err := service.GetSeriesByID(ctx, saved.ID)
		require.NoError(t, err)
		require.NotNil(t, series)
		assert.Equal(t, saved.ID, series.ID)
		assert.Equal(t, "Test Series", series.Title)
	})

	t.Run("get non-existent series returns error", func(t *testing.T) {
		series, err := service.GetSeriesByID(ctx, "non-existent-id")
		assert.Error(t, err)
		assert.Nil(t, series)
	})

	t.Run("empty ID returns error", func(t *testing.T) {
		series, err := service.GetSeriesByID(ctx, "")
		assert.Error(t, err)
		assert.Nil(t, series)
	})
}
