package services

import (
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/tmdb"
)

// ConvertTMDbMovieToModel converts a TMDb MovieDetails to a models.Movie
// filePath is optional and represents the local file path if available
func ConvertTMDbMovieToModel(tmdbMovie *tmdb.MovieDetails, filePath string) *models.Movie {
	if tmdbMovie == nil {
		return nil
	}

	movie := &models.Movie{
		Title:       tmdbMovie.Title,
		ReleaseDate: tmdbMovie.ReleaseDate,
		ParseStatus: models.ParseStatusSuccess,
	}

	// Set TMDb ID
	movie.TMDbID = models.NewNullInt64(int64(tmdbMovie.ID))

	// Set original title
	if tmdbMovie.OriginalTitle != "" {
		movie.OriginalTitle = models.NewNullString(tmdbMovie.OriginalTitle)
	}

	// Set overview
	if tmdbMovie.Overview != "" {
		movie.Overview = models.NewNullString(tmdbMovie.Overview)
	}

	// Set poster path
	if tmdbMovie.PosterPath != nil && *tmdbMovie.PosterPath != "" {
		movie.PosterPath = models.NewNullString(*tmdbMovie.PosterPath)
	}

	// Set backdrop path
	if tmdbMovie.BackdropPath != nil && *tmdbMovie.BackdropPath != "" {
		movie.BackdropPath = models.NewNullString(*tmdbMovie.BackdropPath)
	}

	// Set rating fields
	movie.VoteAverage = models.NewNullFloat64(tmdbMovie.VoteAverage)
	movie.VoteCount = models.NewNullInt64(int64(tmdbMovie.VoteCount))
	movie.Popularity = models.NewNullFloat64(tmdbMovie.Popularity)

	// Also set the legacy rating field for backward compatibility
	movie.Rating = models.NewNullFloat64(tmdbMovie.VoteAverage)

	// Set runtime
	if tmdbMovie.Runtime > 0 {
		movie.Runtime = models.NewNullInt64(int64(tmdbMovie.Runtime))
	}

	// Set original language
	if tmdbMovie.OriginalLanguage != "" {
		movie.OriginalLanguage = models.NewNullString(tmdbMovie.OriginalLanguage)
	}

	// Set status
	if tmdbMovie.Status != "" {
		movie.Status = models.NewNullString(tmdbMovie.Status)
	}

	// Set IMDb ID
	if tmdbMovie.ImdbID != "" {
		movie.IMDbID = models.NewNullString(tmdbMovie.ImdbID)
	}

	// Convert genres (from []Genre to []string of names)
	movie.Genres = make([]string, 0, len(tmdbMovie.Genres))
	for _, genre := range tmdbMovie.Genres {
		movie.Genres = append(movie.Genres, genre.Name)
	}

	// Convert production countries
	if len(tmdbMovie.ProductionCountries) > 0 {
		countries := make([]models.ProductionCountry, 0, len(tmdbMovie.ProductionCountries))
		for _, c := range tmdbMovie.ProductionCountries {
			countries = append(countries, models.ProductionCountry{
				ISO3166_1: c.ISO31661,
				Name:      c.Name,
			})
		}
		_ = movie.SetProductionCountries(countries)
	}

	// Convert spoken languages
	if len(tmdbMovie.SpokenLanguages) > 0 {
		languages := make([]models.SpokenLanguage, 0, len(tmdbMovie.SpokenLanguages))
		for _, l := range tmdbMovie.SpokenLanguages {
			languages = append(languages, models.SpokenLanguage{
				ISO639_1: l.ISO6391,
				Name:     l.Name,
			})
		}
		_ = movie.SetSpokenLanguages(languages)
	}

	// Set file path if provided
	if filePath != "" {
		movie.FilePath = models.NewNullString(filePath)
	}

	// Set metadata source
	movie.MetadataSource = models.NewNullString(string(models.MetadataSourceTMDb))

	return movie
}

// ConvertTMDbSeriesToModel converts a TMDb TVShowDetails to a models.Series
// filePath is optional and represents the local file/folder path if available
func ConvertTMDbSeriesToModel(tmdbSeries *tmdb.TVShowDetails, filePath string) *models.Series {
	if tmdbSeries == nil {
		return nil
	}

	series := &models.Series{
		Title:        tmdbSeries.Name,
		FirstAirDate: tmdbSeries.FirstAirDate,
		ParseStatus:  models.ParseStatusSuccess,
	}

	// Set TMDb ID
	series.TMDbID = models.NewNullInt64(int64(tmdbSeries.ID))

	// Set original title
	if tmdbSeries.OriginalName != "" {
		series.OriginalTitle = models.NewNullString(tmdbSeries.OriginalName)
	}

	// Set last air date
	if tmdbSeries.LastAirDate != "" {
		series.LastAirDate = models.NewNullString(tmdbSeries.LastAirDate)
	}

	// Set overview
	if tmdbSeries.Overview != "" {
		series.Overview = models.NewNullString(tmdbSeries.Overview)
	}

	// Set poster path
	if tmdbSeries.PosterPath != nil && *tmdbSeries.PosterPath != "" {
		series.PosterPath = models.NewNullString(*tmdbSeries.PosterPath)
	}

	// Set backdrop path
	if tmdbSeries.BackdropPath != nil && *tmdbSeries.BackdropPath != "" {
		series.BackdropPath = models.NewNullString(*tmdbSeries.BackdropPath)
	}

	// Set rating fields
	series.VoteAverage = models.NewNullFloat64(tmdbSeries.VoteAverage)
	series.VoteCount = models.NewNullInt64(int64(tmdbSeries.VoteCount))
	series.Popularity = models.NewNullFloat64(tmdbSeries.Popularity)

	// Also set the legacy rating field for backward compatibility
	series.Rating = models.NewNullFloat64(tmdbSeries.VoteAverage)

	// Set number of seasons and episodes
	series.NumberOfSeasons = models.NewNullInt64(int64(tmdbSeries.NumberOfSeasons))
	series.NumberOfEpisodes = models.NewNullInt64(int64(tmdbSeries.NumberOfEpisodes))

	// Set original language
	if tmdbSeries.OriginalLanguage != "" {
		series.OriginalLanguage = models.NewNullString(tmdbSeries.OriginalLanguage)
	}

	// Set status
	if tmdbSeries.Status != "" {
		series.Status = models.NewNullString(tmdbSeries.Status)
	}

	// Set in production flag
	series.InProduction = models.NewNullBool(tmdbSeries.InProduction)

	// Convert genres (from []Genre to []string of names)
	series.Genres = make([]string, 0, len(tmdbSeries.Genres))
	for _, genre := range tmdbSeries.Genres {
		series.Genres = append(series.Genres, genre.Name)
	}

	// Convert seasons to SeasonSummary
	if len(tmdbSeries.Seasons) > 0 {
		seasons := make([]models.SeasonSummary, 0, len(tmdbSeries.Seasons))
		for _, s := range tmdbSeries.Seasons {
			season := models.SeasonSummary{
				ID:           s.ID,
				SeasonNumber: s.SeasonNumber,
				Name:         s.Name,
				Overview:     s.Overview,
				EpisodeCount: s.EpisodeCount,
			}
			if s.PosterPath != nil {
				season.PosterPath = *s.PosterPath
			}
			if s.AirDate != nil {
				season.AirDate = *s.AirDate
			}
			seasons = append(seasons, season)
		}
		_ = series.SetSeasons(seasons)
	}

	// Set file path if provided
	if filePath != "" {
		series.FilePath = models.NewNullString(filePath)
	}

	// Set metadata source
	series.MetadataSource = models.NewNullString(string(models.MetadataSourceTMDb))

	return series
}

// ConvertTMDbMovieSearchResultToModel converts a basic TMDb Movie (from search results) to a models.Movie
// This is useful when we only have search results without full details
func ConvertTMDbMovieSearchResultToModel(tmdbMovie *tmdb.Movie, filePath string) *models.Movie {
	if tmdbMovie == nil {
		return nil
	}

	movie := &models.Movie{
		Title:       tmdbMovie.Title,
		ReleaseDate: tmdbMovie.ReleaseDate,
		ParseStatus: models.ParseStatusSuccess,
	}

	// Set TMDb ID
	movie.TMDbID = models.NewNullInt64(int64(tmdbMovie.ID))

	// Set original title
	if tmdbMovie.OriginalTitle != "" {
		movie.OriginalTitle = models.NewNullString(tmdbMovie.OriginalTitle)
	}

	// Set overview
	if tmdbMovie.Overview != "" {
		movie.Overview = models.NewNullString(tmdbMovie.Overview)
	}

	// Set poster path
	if tmdbMovie.PosterPath != nil && *tmdbMovie.PosterPath != "" {
		movie.PosterPath = models.NewNullString(*tmdbMovie.PosterPath)
	}

	// Set backdrop path
	if tmdbMovie.BackdropPath != nil && *tmdbMovie.BackdropPath != "" {
		movie.BackdropPath = models.NewNullString(*tmdbMovie.BackdropPath)
	}

	// Set rating fields
	movie.VoteAverage = models.NewNullFloat64(tmdbMovie.VoteAverage)
	movie.VoteCount = models.NewNullInt64(int64(tmdbMovie.VoteCount))
	movie.Popularity = models.NewNullFloat64(tmdbMovie.Popularity)
	movie.Rating = models.NewNullFloat64(tmdbMovie.VoteAverage)

	// Set original language
	if tmdbMovie.OriginalLanguage != "" {
		movie.OriginalLanguage = models.NewNullString(tmdbMovie.OriginalLanguage)
	}

	// Note: Search results only have genre IDs, not names
	// We store empty genres here; full details would need a separate API call
	movie.Genres = []string{}

	// Set file path if provided
	if filePath != "" {
		movie.FilePath = models.NewNullString(filePath)
	}

	// Set metadata source
	movie.MetadataSource = models.NewNullString(string(models.MetadataSourceTMDb))

	return movie
}

// ConvertTMDbTVShowSearchResultToModel converts a basic TMDb TVShow (from search results) to a models.Series
// This is useful when we only have search results without full details
func ConvertTMDbTVShowSearchResultToModel(tmdbSeries *tmdb.TVShow, filePath string) *models.Series {
	if tmdbSeries == nil {
		return nil
	}

	series := &models.Series{
		Title:        tmdbSeries.Name,
		FirstAirDate: tmdbSeries.FirstAirDate,
		ParseStatus:  models.ParseStatusSuccess,
	}

	// Set TMDb ID
	series.TMDbID = models.NewNullInt64(int64(tmdbSeries.ID))

	// Set original title
	if tmdbSeries.OriginalName != "" {
		series.OriginalTitle = models.NewNullString(tmdbSeries.OriginalName)
	}

	// Set overview
	if tmdbSeries.Overview != "" {
		series.Overview = models.NewNullString(tmdbSeries.Overview)
	}

	// Set poster path
	if tmdbSeries.PosterPath != nil && *tmdbSeries.PosterPath != "" {
		series.PosterPath = models.NewNullString(*tmdbSeries.PosterPath)
	}

	// Set backdrop path
	if tmdbSeries.BackdropPath != nil && *tmdbSeries.BackdropPath != "" {
		series.BackdropPath = models.NewNullString(*tmdbSeries.BackdropPath)
	}

	// Set rating fields
	series.VoteAverage = models.NewNullFloat64(tmdbSeries.VoteAverage)
	series.VoteCount = models.NewNullInt64(int64(tmdbSeries.VoteCount))
	series.Popularity = models.NewNullFloat64(tmdbSeries.Popularity)
	series.Rating = models.NewNullFloat64(tmdbSeries.VoteAverage)

	// Set original language
	if tmdbSeries.OriginalLanguage != "" {
		series.OriginalLanguage = models.NewNullString(tmdbSeries.OriginalLanguage)
	}

	// Note: Search results only have genre IDs, not names
	// We store empty genres here; full details would need a separate API call
	series.Genres = []string{}

	// Set file path if provided
	if filePath != "" {
		series.FilePath = models.NewNullString(filePath)
	}

	// Set metadata source
	series.MetadataSource = models.NewNullString(string(models.MetadataSourceTMDb))

	return series
}
