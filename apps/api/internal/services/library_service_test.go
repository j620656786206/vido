package services

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/database/migrations"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
	"github.com/vido/api/internal/tmdb"

	_ "modernc.org/sqlite"
)

// setupTestDB creates a temporary SQLite database with schema from production migrations
func setupTestDB(t *testing.T) *sql.DB {
	tmpFile, err := os.CreateTemp("", "test_library_*.db")
	require.NoError(t, err)
	tmpFile.Close()

	t.Cleanup(func() {
		os.Remove(tmpFile.Name())
	})

	db, err := sql.Open("sqlite", tmpFile.Name()+"?_pragma=foreign_keys(1)")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	runner, err := migrations.NewRunner(db)
	require.NoError(t, err)
	err = runner.RegisterAll(migrations.GetAll())
	require.NoError(t, err)
	err = runner.Up(context.Background())
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
		}, "all")
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
		}, "all")
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
		}, "all")
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
		}, "all")
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
		}, "all")
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

func TestLibraryService_ListLibrary(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	movieRepo := repository.NewMovieRepository(db)
	seriesRepo := repository.NewSeriesRepository(db)
	episodeRepo := repository.NewEpisodeRepository(db)

	service := NewLibraryService(movieRepo, seriesRepo, episodeRepo)

	ctx := context.Background()

	// Insert test data
	posterPath := "/poster.jpg"
	for i := 1; i <= 3; i++ {
		_, err := service.SaveMovieFromTMDb(ctx, &tmdb.MovieDetails{
			Movie: tmdb.Movie{
				ID:          i,
				Title:       fmt.Sprintf("Movie %d", i),
				ReleaseDate: "2023-01-01",
				PosterPath:  &posterPath,
			},
		}, "")
		require.NoError(t, err)
	}
	for i := 100; i <= 102; i++ {
		_, err := service.SaveSeriesFromTMDb(ctx, &tmdb.TVShowDetails{
			TVShow: tmdb.TVShow{
				ID:           i,
				Name:         fmt.Sprintf("Series %d", i-99),
				FirstAirDate: "2023-01-01",
				PosterPath:   &posterPath,
			},
		}, "")
		require.NoError(t, err)
	}

	t.Run("list all media types", func(t *testing.T) {
		result, err := service.ListLibrary(ctx, repository.NewListParams(), "all")
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 6, len(result.Items))
		assert.Equal(t, 6, result.Pagination.TotalResults)
	})

	t.Run("list movies only", func(t *testing.T) {
		result, err := service.ListLibrary(ctx, repository.NewListParams(), "movie")
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 3, len(result.Items))
		for _, item := range result.Items {
			assert.Equal(t, "movie", item.Type)
			assert.NotNil(t, item.Movie)
			assert.Nil(t, item.Series)
		}
	})

	t.Run("list series only", func(t *testing.T) {
		result, err := service.ListLibrary(ctx, repository.NewListParams(), "tv")
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 3, len(result.Items))
		for _, item := range result.Items {
			assert.Equal(t, "series", item.Type)
			assert.NotNil(t, item.Series)
			assert.Nil(t, item.Movie)
		}
	})

	t.Run("pagination works", func(t *testing.T) {
		params := repository.NewListParams()
		params.PageSize = 2
		params.Page = 1
		result, err := service.ListLibrary(ctx, params, "movie")
		require.NoError(t, err)
		assert.Equal(t, 2, len(result.Items))
		assert.Equal(t, 3, result.Pagination.TotalResults)
		assert.Equal(t, 2, result.Pagination.TotalPages)
	})

	t.Run("default sort is created_at DESC", func(t *testing.T) {
		result, err := service.ListLibrary(ctx, repository.NewListParams(), "movie")
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(result.Items), 2)
		// Last inserted should be first (DESC order)
		assert.Equal(t, "Movie 3", result.Items[0].Movie.Title)
	})

	t.Run("sort by title ASC", func(t *testing.T) {
		params := repository.NewListParams()
		params.SortBy = "title"
		params.SortOrder = "asc"
		result, err := service.ListLibrary(ctx, params, "movie")
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(result.Items), 3)
		assert.Equal(t, "Movie 1", result.Items[0].Movie.Title)
		assert.Equal(t, "Movie 2", result.Items[1].Movie.Title)
		assert.Equal(t, "Movie 3", result.Items[2].Movie.Title)
	})

	t.Run("sort by title DESC", func(t *testing.T) {
		params := repository.NewListParams()
		params.SortBy = "title"
		params.SortOrder = "desc"
		result, err := service.ListLibrary(ctx, params, "movie")
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(result.Items), 3)
		assert.Equal(t, "Movie 3", result.Items[0].Movie.Title)
		assert.Equal(t, "Movie 2", result.Items[1].Movie.Title)
		assert.Equal(t, "Movie 1", result.Items[2].Movie.Title)
	})

	t.Run("sort by vote_average supported", func(t *testing.T) {
		params := repository.NewListParams()
		params.SortBy = "vote_average"
		params.SortOrder = "desc"
		result, err := service.ListLibrary(ctx, params, "movie")
		require.NoError(t, err)
		assert.NotNil(t, result)
	})
}

func TestLibraryService_DeleteMovie(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	movieRepo := repository.NewMovieRepository(db)
	seriesRepo := repository.NewSeriesRepository(db)
	episodeRepo := repository.NewEpisodeRepository(db)

	service := NewLibraryService(movieRepo, seriesRepo, episodeRepo)

	ctx := context.Background()

	posterPath := "/poster.jpg"
	movie, err := service.SaveMovieFromTMDb(ctx, &tmdb.MovieDetails{
		Movie: tmdb.Movie{
			ID:         500,
			Title:      "To Delete",
			PosterPath: &posterPath,
		},
	}, "")
	require.NoError(t, err)

	t.Run("delete existing movie", func(t *testing.T) {
		err := service.DeleteMovie(ctx, movie.ID)
		require.NoError(t, err)
		_, err = service.GetMovieByID(ctx, movie.ID)
		assert.Error(t, err)
	})

	t.Run("empty ID returns error", func(t *testing.T) {
		err := service.DeleteMovie(ctx, "")
		assert.Error(t, err)
	})
}

func TestLibraryService_DeleteSeries(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	movieRepo := repository.NewMovieRepository(db)
	seriesRepo := repository.NewSeriesRepository(db)
	episodeRepo := repository.NewEpisodeRepository(db)

	service := NewLibraryService(movieRepo, seriesRepo, episodeRepo)

	ctx := context.Background()

	posterPath := "/poster.jpg"
	series, err := service.SaveSeriesFromTMDb(ctx, &tmdb.TVShowDetails{
		TVShow: tmdb.TVShow{
			ID:         700,
			Name:       "To Delete Series",
			PosterPath: &posterPath,
		},
	}, "")
	require.NoError(t, err)

	t.Run("delete existing series", func(t *testing.T) {
		err := service.DeleteSeries(ctx, series.ID)
		require.NoError(t, err)
		_, err = service.GetSeriesByID(ctx, series.ID)
		assert.Error(t, err)
	})

	t.Run("empty ID returns error", func(t *testing.T) {
		err := service.DeleteSeries(ctx, "")
		assert.Error(t, err)
	})
}

func TestLibraryService_GetRecentlyAdded(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	movieRepo := repository.NewMovieRepository(db)
	seriesRepo := repository.NewSeriesRepository(db)
	episodeRepo := repository.NewEpisodeRepository(db)

	service := NewLibraryService(movieRepo, seriesRepo, episodeRepo)

	ctx := context.Background()

	// Insert movies with different created_at timestamps
	posterPath := "/poster.jpg"
	for i := 1; i <= 3; i++ {
		_, err := service.SaveMovieFromTMDb(ctx, &tmdb.MovieDetails{
			Movie: tmdb.Movie{
				ID:          2000 + i,
				Title:       fmt.Sprintf("Recent Movie %d", i),
				ReleaseDate: "2026-01-01",
				PosterPath:  &posterPath,
			},
		}, "")
		require.NoError(t, err)
	}

	// Insert series
	for i := 1; i <= 2; i++ {
		_, err := service.SaveSeriesFromTMDb(ctx, &tmdb.TVShowDetails{
			TVShow: tmdb.TVShow{
				ID:           3000 + i,
				Name:         fmt.Sprintf("Recent Series %d", i),
				FirstAirDate: "2026-01-01",
				PosterPath:   &posterPath,
			},
		}, "")
		require.NoError(t, err)
	}

	t.Run("returns all items with mixed types", func(t *testing.T) {
		result, err := service.GetRecentlyAdded(ctx, 20)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 5, len(result.Items))

		// Verify items contain both movies and series
		movieCount := 0
		seriesCount := 0
		for _, item := range result.Items {
			if item.Type == "movie" {
				movieCount++
				assert.NotNil(t, item.Movie)
			} else if item.Type == "series" {
				seriesCount++
				assert.NotNil(t, item.Series)
			}
		}
		assert.Equal(t, 3, movieCount)
		assert.Equal(t, 2, seriesCount)
	})

	t.Run("respects limit parameter", func(t *testing.T) {
		result, err := service.GetRecentlyAdded(ctx, 3)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 3, len(result.Items))
	})

	t.Run("defaults to 20 when limit is zero", func(t *testing.T) {
		result, err := service.GetRecentlyAdded(ctx, 0)
		require.NoError(t, err)
		require.NotNil(t, result)
		// Should return all 5 items (less than default 20)
		assert.Equal(t, 5, len(result.Items))
	})

	t.Run("defaults to 20 when limit is negative", func(t *testing.T) {
		result, err := service.GetRecentlyAdded(ctx, -1)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, 5, len(result.Items))
	})
}

func TestLibraryService_FilterByGenre(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	movieRepo := repository.NewMovieRepository(db)
	seriesRepo := repository.NewSeriesRepository(db)
	episodeRepo := repository.NewEpisodeRepository(db)

	service := NewLibraryService(movieRepo, seriesRepo, episodeRepo)

	ctx := context.Background()
	posterPath := "/poster.jpg"

	// Insert movies with different genres
	_, err := service.SaveMovieFromTMDb(ctx, &tmdb.MovieDetails{
		Movie:  tmdb.Movie{ID: 10, Title: "Sci-Fi Action Movie", PosterPath: &posterPath, ReleaseDate: "2020-01-01"},
		Genres: []tmdb.Genre{{Name: "Science Fiction"}, {Name: "Action"}},
	}, "")
	require.NoError(t, err)

	_, err = service.SaveMovieFromTMDb(ctx, &tmdb.MovieDetails{
		Movie:  tmdb.Movie{ID: 11, Title: "Drama Movie", PosterPath: &posterPath, ReleaseDate: "2015-06-01"},
		Genres: []tmdb.Genre{{Name: "Drama"}},
	}, "")
	require.NoError(t, err)

	_, err = service.SaveMovieFromTMDb(ctx, &tmdb.MovieDetails{
		Movie:  tmdb.Movie{ID: 12, Title: "Action Drama Movie", PosterPath: &posterPath, ReleaseDate: "2022-03-15"},
		Genres: []tmdb.Genre{{Name: "Action"}, {Name: "Drama"}},
	}, "")
	require.NoError(t, err)

	t.Run("filter by single genre returns matching movies", func(t *testing.T) {
		params := repository.NewListParams()
		params.Filters["genres"] = []string{"Action"}
		result, err := service.ListLibrary(ctx, params, "movie")
		require.NoError(t, err)
		assert.Equal(t, 2, len(result.Items)) // Sci-Fi Action + Action Drama
	})

	t.Run("filter by multiple genres uses AND logic", func(t *testing.T) {
		params := repository.NewListParams()
		params.Filters["genres"] = []string{"Action", "Drama"}
		result, err := service.ListLibrary(ctx, params, "movie")
		require.NoError(t, err)
		assert.Equal(t, 1, len(result.Items)) // Only Action Drama
		assert.Equal(t, "Action Drama Movie", result.Items[0].Movie.Title)
	})

	t.Run("filter by non-existing genre returns empty", func(t *testing.T) {
		params := repository.NewListParams()
		params.Filters["genres"] = []string{"Horror"}
		result, err := service.ListLibrary(ctx, params, "movie")
		require.NoError(t, err)
		assert.Equal(t, 0, len(result.Items))
	})
}

func TestLibraryService_FilterByYearRange(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	movieRepo := repository.NewMovieRepository(db)
	seriesRepo := repository.NewSeriesRepository(db)
	episodeRepo := repository.NewEpisodeRepository(db)

	service := NewLibraryService(movieRepo, seriesRepo, episodeRepo)

	ctx := context.Background()
	posterPath := "/poster.jpg"

	// Insert movies with different years
	for _, m := range []struct {
		id    int
		title string
		year  string
	}{
		{20, "Movie 1999", "1999-05-15"},
		{21, "Movie 2010", "2010-08-20"},
		{22, "Movie 2020", "2020-12-01"},
		{23, "Movie 2023", "2023-03-10"},
	} {
		_, err := service.SaveMovieFromTMDb(ctx, &tmdb.MovieDetails{
			Movie: tmdb.Movie{ID: m.id, Title: m.title, PosterPath: &posterPath, ReleaseDate: m.year},
		}, "")
		require.NoError(t, err)
	}

	t.Run("filter by year_min", func(t *testing.T) {
		params := repository.NewListParams()
		params.Filters["year_min"] = "2010"
		result, err := service.ListLibrary(ctx, params, "movie")
		require.NoError(t, err)
		assert.Equal(t, 3, len(result.Items)) // 2010, 2020, 2023
	})

	t.Run("filter by year_max", func(t *testing.T) {
		params := repository.NewListParams()
		params.Filters["year_max"] = "2010"
		result, err := service.ListLibrary(ctx, params, "movie")
		require.NoError(t, err)
		assert.Equal(t, 2, len(result.Items)) // 1999, 2010
	})

	t.Run("filter by year range", func(t *testing.T) {
		params := repository.NewListParams()
		params.Filters["year_min"] = "2010"
		params.Filters["year_max"] = "2020"
		result, err := service.ListLibrary(ctx, params, "movie")
		require.NoError(t, err)
		assert.Equal(t, 2, len(result.Items)) // 2010, 2020
	})
}

func TestLibraryService_GetDistinctGenres(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	movieRepo := repository.NewMovieRepository(db)
	seriesRepo := repository.NewSeriesRepository(db)
	episodeRepo := repository.NewEpisodeRepository(db)

	service := NewLibraryService(movieRepo, seriesRepo, episodeRepo)

	ctx := context.Background()
	posterPath := "/poster.jpg"

	// Insert movies and series with various genres
	_, err := service.SaveMovieFromTMDb(ctx, &tmdb.MovieDetails{
		Movie:  tmdb.Movie{ID: 30, Title: "Movie A", PosterPath: &posterPath},
		Genres: []tmdb.Genre{{Name: "Action"}, {Name: "Drama"}},
	}, "")
	require.NoError(t, err)

	_, err = service.SaveMovieFromTMDb(ctx, &tmdb.MovieDetails{
		Movie:  tmdb.Movie{ID: 31, Title: "Movie B", PosterPath: &posterPath},
		Genres: []tmdb.Genre{{Name: "Comedy"}, {Name: "Drama"}},
	}, "")
	require.NoError(t, err)

	_, err = service.SaveSeriesFromTMDb(ctx, &tmdb.TVShowDetails{
		TVShow: tmdb.TVShow{ID: 200, Name: "Series A", PosterPath: &posterPath},
		Genres: []tmdb.Genre{{Name: "Thriller"}, {Name: "Action"}},
	}, "")
	require.NoError(t, err)

	t.Run("returns deduplicated sorted genres", func(t *testing.T) {
		genres, err := service.GetDistinctGenres(ctx)
		require.NoError(t, err)
		assert.Equal(t, []string{"Action", "Comedy", "Drama", "Thriller"}, genres)
	})
}

func TestLibraryService_GetLibraryStats(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	movieRepo := repository.NewMovieRepository(db)
	seriesRepo := repository.NewSeriesRepository(db)
	episodeRepo := repository.NewEpisodeRepository(db)

	service := NewLibraryService(movieRepo, seriesRepo, episodeRepo)

	ctx := context.Background()
	posterPath := "/poster.jpg"

	// Insert movies
	_, err := service.SaveMovieFromTMDb(ctx, &tmdb.MovieDetails{
		Movie: tmdb.Movie{ID: 40, Title: "Old Movie", PosterPath: &posterPath, ReleaseDate: "1999-05-01"},
	}, "")
	require.NoError(t, err)

	_, err = service.SaveMovieFromTMDb(ctx, &tmdb.MovieDetails{
		Movie: tmdb.Movie{ID: 41, Title: "New Movie", PosterPath: &posterPath, ReleaseDate: "2024-01-01"},
	}, "")
	require.NoError(t, err)

	// Insert series
	_, err = service.SaveSeriesFromTMDb(ctx, &tmdb.TVShowDetails{
		TVShow: tmdb.TVShow{ID: 300, Name: "Old Series", PosterPath: &posterPath, FirstAirDate: "2005-06-01"},
	}, "")
	require.NoError(t, err)

	t.Run("returns correct stats", func(t *testing.T) {
		stats, err := service.GetLibraryStats(ctx)
		require.NoError(t, err)
		assert.Equal(t, 1999, stats.YearMin)
		assert.Equal(t, 2024, stats.YearMax)
		assert.Equal(t, 2, stats.MovieCount)
		assert.Equal(t, 1, stats.TvCount)
		assert.Equal(t, 3, stats.TotalCount)
	})
}

func TestLibraryService_CombinedFilters(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	movieRepo := repository.NewMovieRepository(db)
	seriesRepo := repository.NewSeriesRepository(db)
	episodeRepo := repository.NewEpisodeRepository(db)

	service := NewLibraryService(movieRepo, seriesRepo, episodeRepo)

	ctx := context.Background()
	posterPath := "/poster.jpg"

	// Insert diverse test data
	_, err := service.SaveMovieFromTMDb(ctx, &tmdb.MovieDetails{
		Movie:  tmdb.Movie{ID: 50, Title: "Old Action", PosterPath: &posterPath, ReleaseDate: "2000-01-01"},
		Genres: []tmdb.Genre{{Name: "Action"}},
	}, "")
	require.NoError(t, err)

	_, err = service.SaveMovieFromTMDb(ctx, &tmdb.MovieDetails{
		Movie:  tmdb.Movie{ID: 51, Title: "New Action", PosterPath: &posterPath, ReleaseDate: "2022-06-01"},
		Genres: []tmdb.Genre{{Name: "Action"}},
	}, "")
	require.NoError(t, err)

	_, err = service.SaveMovieFromTMDb(ctx, &tmdb.MovieDetails{
		Movie:  tmdb.Movie{ID: 52, Title: "New Drama", PosterPath: &posterPath, ReleaseDate: "2023-01-01"},
		Genres: []tmdb.Genre{{Name: "Drama"}},
	}, "")
	require.NoError(t, err)

	t.Run("genre + year range combined filter", func(t *testing.T) {
		params := repository.NewListParams()
		params.Filters["genres"] = []string{"Action"}
		params.Filters["year_min"] = "2020"
		result, err := service.ListLibrary(ctx, params, "movie")
		require.NoError(t, err)
		assert.Equal(t, 1, len(result.Items))
		assert.Equal(t, "New Action", result.Items[0].Movie.Title)
	})

	t.Run("genre + type combined filter", func(t *testing.T) {
		params := repository.NewListParams()
		params.Filters["genres"] = []string{"Action"}
		result, err := service.ListLibrary(ctx, params, "movie")
		require.NoError(t, err)
		assert.Equal(t, 2, len(result.Items))
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
