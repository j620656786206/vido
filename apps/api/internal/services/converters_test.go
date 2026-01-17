package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/tmdb"
)

func TestConvertTMDbMovieToModel(t *testing.T) {
	posterPath := "/poster.jpg"
	backdropPath := "/backdrop.jpg"

	tests := []struct {
		name     string
		input    *tmdb.MovieDetails
		filePath string
		validate func(t *testing.T, movie *models.Movie)
	}{
		{
			name:  "nil input returns nil",
			input: nil,
			validate: func(t *testing.T, movie *models.Movie) {
				assert.Nil(t, movie)
			},
		},
		{
			name: "full movie details conversion",
			input: &tmdb.MovieDetails{
				Movie: tmdb.Movie{
					ID:               550,
					Title:            "Fight Club",
					OriginalTitle:    "Fight Club",
					Overview:         "A ticking-time-bomb insomniac...",
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
				ProductionCountries: []tmdb.Country{
					{ISO31661: "US", Name: "United States of America"},
					{ISO31661: "DE", Name: "Germany"},
				},
				SpokenLanguages: []tmdb.Language{
					{ISO6391: "en", Name: "English"},
				},
			},
			filePath: "/movies/fight_club.mkv",
			validate: func(t *testing.T, movie *models.Movie) {
				require.NotNil(t, movie)

				// Core fields
				assert.Equal(t, "Fight Club", movie.Title)
				assert.Equal(t, "1999-10-15", movie.ReleaseDate)
				assert.Equal(t, models.ParseStatusSuccess, movie.ParseStatus)

				// TMDb ID
				assert.True(t, movie.TMDbID.Valid)
				assert.Equal(t, int64(550), movie.TMDbID.Int64)

				// Original title
				assert.True(t, movie.OriginalTitle.Valid)
				assert.Equal(t, "Fight Club", movie.OriginalTitle.String)

				// Overview
				assert.True(t, movie.Overview.Valid)
				assert.Equal(t, "A ticking-time-bomb insomniac...", movie.Overview.String)

				// Poster and backdrop
				assert.True(t, movie.PosterPath.Valid)
				assert.Equal(t, "/poster.jpg", movie.PosterPath.String)
				assert.True(t, movie.BackdropPath.Valid)
				assert.Equal(t, "/backdrop.jpg", movie.BackdropPath.String)

				// Rating fields
				assert.True(t, movie.VoteAverage.Valid)
				assert.Equal(t, 8.4, movie.VoteAverage.Float64)
				assert.True(t, movie.VoteCount.Valid)
				assert.Equal(t, int64(26280), movie.VoteCount.Int64)
				assert.True(t, movie.Popularity.Valid)
				assert.Equal(t, 61.416, movie.Popularity.Float64)
				assert.True(t, movie.Rating.Valid)
				assert.Equal(t, 8.4, movie.Rating.Float64)

				// Runtime
				assert.True(t, movie.Runtime.Valid)
				assert.Equal(t, int64(139), movie.Runtime.Int64)

				// Status and IMDb
				assert.True(t, movie.Status.Valid)
				assert.Equal(t, "Released", movie.Status.String)
				assert.True(t, movie.IMDbID.Valid)
				assert.Equal(t, "tt0137523", movie.IMDbID.String)

				// Original language
				assert.True(t, movie.OriginalLanguage.Valid)
				assert.Equal(t, "en", movie.OriginalLanguage.String)

				// Genres
				assert.Equal(t, []string{"Drama", "Thriller"}, movie.Genres)

				// Production countries
				countries, err := movie.GetProductionCountries()
				require.NoError(t, err)
				require.Len(t, countries, 2)
				assert.Equal(t, "US", countries[0].ISO3166_1)
				assert.Equal(t, "United States of America", countries[0].Name)

				// Spoken languages
				languages, err := movie.GetSpokenLanguages()
				require.NoError(t, err)
				require.Len(t, languages, 1)
				assert.Equal(t, "en", languages[0].ISO639_1)

				// File path
				assert.True(t, movie.FilePath.Valid)
				assert.Equal(t, "/movies/fight_club.mkv", movie.FilePath.String)

				// Metadata source
				assert.True(t, movie.MetadataSource.Valid)
				assert.Equal(t, "tmdb", movie.MetadataSource.String)
			},
		},
		{
			name: "minimal movie details",
			input: &tmdb.MovieDetails{
				Movie: tmdb.Movie{
					ID:          123,
					Title:       "Test Movie",
					ReleaseDate: "2023-01-01",
				},
			},
			filePath: "",
			validate: func(t *testing.T, movie *models.Movie) {
				require.NotNil(t, movie)
				assert.Equal(t, "Test Movie", movie.Title)
				assert.True(t, movie.TMDbID.Valid)
				assert.Equal(t, int64(123), movie.TMDbID.Int64)
				assert.False(t, movie.FilePath.Valid)
				assert.Equal(t, []string{}, movie.Genres)
			},
		},
		{
			name: "movie with nil optional fields",
			input: &tmdb.MovieDetails{
				Movie: tmdb.Movie{
					ID:           456,
					Title:        "No Posters",
					ReleaseDate:  "2020-05-05",
					PosterPath:   nil,
					BackdropPath: nil,
				},
			},
			validate: func(t *testing.T, movie *models.Movie) {
				require.NotNil(t, movie)
				assert.False(t, movie.PosterPath.Valid)
				assert.False(t, movie.BackdropPath.Valid)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertTMDbMovieToModel(tt.input, tt.filePath)
			tt.validate(t, result)
		})
	}
}

func TestConvertTMDbSeriesToModel(t *testing.T) {
	posterPath := "/series_poster.jpg"
	backdropPath := "/series_backdrop.jpg"
	seasonPoster := "/season1_poster.jpg"
	airDate := "2008-01-20"

	tests := []struct {
		name     string
		input    *tmdb.TVShowDetails
		filePath string
		validate func(t *testing.T, series *models.Series)
	}{
		{
			name:  "nil input returns nil",
			input: nil,
			validate: func(t *testing.T, series *models.Series) {
				assert.Nil(t, series)
			},
		},
		{
			name: "full series details conversion",
			input: &tmdb.TVShowDetails{
				TVShow: tmdb.TVShow{
					ID:               1396,
					Name:             "Breaking Bad",
					OriginalName:     "Breaking Bad",
					Overview:         "When Walter White is diagnosed...",
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
				Seasons: []tmdb.Season{
					{
						ID:           3572,
						SeasonNumber: 1,
						Name:         "Season 1",
						Overview:     "High school chemistry teacher...",
						EpisodeCount: 7,
						PosterPath:   &seasonPoster,
						AirDate:      &airDate,
					},
				},
			},
			filePath: "/series/breaking_bad",
			validate: func(t *testing.T, series *models.Series) {
				require.NotNil(t, series)

				// Core fields
				assert.Equal(t, "Breaking Bad", series.Title)
				assert.Equal(t, "2008-01-20", series.FirstAirDate)
				assert.Equal(t, models.ParseStatusSuccess, series.ParseStatus)

				// TMDb ID
				assert.True(t, series.TMDbID.Valid)
				assert.Equal(t, int64(1396), series.TMDbID.Int64)

				// Original title
				assert.True(t, series.OriginalTitle.Valid)
				assert.Equal(t, "Breaking Bad", series.OriginalTitle.String)

				// Last air date
				assert.True(t, series.LastAirDate.Valid)
				assert.Equal(t, "2013-09-29", series.LastAirDate.String)

				// Overview
				assert.True(t, series.Overview.Valid)
				assert.Equal(t, "When Walter White is diagnosed...", series.Overview.String)

				// Poster and backdrop
				assert.True(t, series.PosterPath.Valid)
				assert.Equal(t, "/series_poster.jpg", series.PosterPath.String)
				assert.True(t, series.BackdropPath.Valid)
				assert.Equal(t, "/series_backdrop.jpg", series.BackdropPath.String)

				// Rating fields
				assert.True(t, series.VoteAverage.Valid)
				assert.Equal(t, 8.9, series.VoteAverage.Float64)
				assert.True(t, series.VoteCount.Valid)
				assert.Equal(t, int64(12345), series.VoteCount.Int64)
				assert.True(t, series.Popularity.Valid)
				assert.Equal(t, 369.594, series.Popularity.Float64)

				// Season/episode counts
				assert.True(t, series.NumberOfSeasons.Valid)
				assert.Equal(t, int64(5), series.NumberOfSeasons.Int64)
				assert.True(t, series.NumberOfEpisodes.Valid)
				assert.Equal(t, int64(62), series.NumberOfEpisodes.Int64)

				// Status and in production
				assert.True(t, series.Status.Valid)
				assert.Equal(t, "Ended", series.Status.String)
				assert.True(t, series.InProduction.Valid)
				assert.False(t, series.InProduction.Bool)

				// Genres
				assert.Equal(t, []string{"Drama", "Crime"}, series.Genres)

				// Seasons
				seasons, err := series.GetSeasons()
				require.NoError(t, err)
				require.Len(t, seasons, 1)
				assert.Equal(t, 3572, seasons[0].ID)
				assert.Equal(t, 1, seasons[0].SeasonNumber)
				assert.Equal(t, "Season 1", seasons[0].Name)
				assert.Equal(t, 7, seasons[0].EpisodeCount)
				assert.Equal(t, "/season1_poster.jpg", seasons[0].PosterPath)
				assert.Equal(t, "2008-01-20", seasons[0].AirDate)

				// File path
				assert.True(t, series.FilePath.Valid)
				assert.Equal(t, "/series/breaking_bad", series.FilePath.String)

				// Metadata source
				assert.True(t, series.MetadataSource.Valid)
				assert.Equal(t, "tmdb", series.MetadataSource.String)
			},
		},
		{
			name: "minimal series details",
			input: &tmdb.TVShowDetails{
				TVShow: tmdb.TVShow{
					ID:           789,
					Name:         "Test Series",
					FirstAirDate: "2022-06-15",
				},
			},
			filePath: "",
			validate: func(t *testing.T, series *models.Series) {
				require.NotNil(t, series)
				assert.Equal(t, "Test Series", series.Title)
				assert.True(t, series.TMDbID.Valid)
				assert.Equal(t, int64(789), series.TMDbID.Int64)
				assert.False(t, series.FilePath.Valid)
				assert.Equal(t, []string{}, series.Genres)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertTMDbSeriesToModel(tt.input, tt.filePath)
			tt.validate(t, result)
		})
	}
}

func TestConvertTMDbMovieSearchResultToModel(t *testing.T) {
	posterPath := "/search_poster.jpg"

	tests := []struct {
		name     string
		input    *tmdb.Movie
		filePath string
		validate func(t *testing.T, movie *models.Movie)
	}{
		{
			name:  "nil input returns nil",
			input: nil,
			validate: func(t *testing.T, movie *models.Movie) {
				assert.Nil(t, movie)
			},
		},
		{
			name: "search result conversion",
			input: &tmdb.Movie{
				ID:               550,
				Title:            "Fight Club",
				OriginalTitle:    "Fight Club",
				Overview:         "A ticking-time-bomb insomniac...",
				ReleaseDate:      "1999-10-15",
				PosterPath:       &posterPath,
				VoteAverage:      8.4,
				VoteCount:        26280,
				Popularity:       61.416,
				OriginalLanguage: "en",
				GenreIDs:         []int{18, 53}, // These are IDs, not names
			},
			filePath: "/movies/fight_club.mkv",
			validate: func(t *testing.T, movie *models.Movie) {
				require.NotNil(t, movie)
				assert.Equal(t, "Fight Club", movie.Title)
				assert.True(t, movie.TMDbID.Valid)
				assert.Equal(t, int64(550), movie.TMDbID.Int64)
				assert.True(t, movie.PosterPath.Valid)
				assert.Equal(t, "/search_poster.jpg", movie.PosterPath.String)
				// Search results don't include genre names
				assert.Equal(t, []string{}, movie.Genres)
				assert.True(t, movie.FilePath.Valid)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertTMDbMovieSearchResultToModel(tt.input, tt.filePath)
			tt.validate(t, result)
		})
	}
}

func TestConvertTMDbTVShowSearchResultToModel(t *testing.T) {
	posterPath := "/tv_search_poster.jpg"

	tests := []struct {
		name     string
		input    *tmdb.TVShow
		filePath string
		validate func(t *testing.T, series *models.Series)
	}{
		{
			name:  "nil input returns nil",
			input: nil,
			validate: func(t *testing.T, series *models.Series) {
				assert.Nil(t, series)
			},
		},
		{
			name: "search result conversion",
			input: &tmdb.TVShow{
				ID:               1396,
				Name:             "Breaking Bad",
				OriginalName:     "Breaking Bad",
				Overview:         "When Walter White is diagnosed...",
				FirstAirDate:     "2008-01-20",
				PosterPath:       &posterPath,
				VoteAverage:      8.9,
				VoteCount:        12345,
				Popularity:       369.594,
				OriginalLanguage: "en",
				GenreIDs:         []int{18, 80},
			},
			filePath: "/series/breaking_bad",
			validate: func(t *testing.T, series *models.Series) {
				require.NotNil(t, series)
				assert.Equal(t, "Breaking Bad", series.Title)
				assert.True(t, series.TMDbID.Valid)
				assert.Equal(t, int64(1396), series.TMDbID.Int64)
				assert.True(t, series.PosterPath.Valid)
				assert.Equal(t, "/tv_search_poster.jpg", series.PosterPath.String)
				// Search results don't include genre names
				assert.Equal(t, []string{}, series.Genres)
				assert.True(t, series.FilePath.Valid)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertTMDbTVShowSearchResultToModel(tt.input, tt.filePath)
			tt.validate(t, result)
		})
	}
}
