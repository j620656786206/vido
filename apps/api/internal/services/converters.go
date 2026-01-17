package services

import (
	"database/sql"

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
	movie.TMDbID = sql.NullInt64{Int64: int64(tmdbMovie.ID), Valid: true}

	// Set original title
	if tmdbMovie.OriginalTitle != "" {
		movie.OriginalTitle = sql.NullString{String: tmdbMovie.OriginalTitle, Valid: true}
	}

	// Set overview
	if tmdbMovie.Overview != "" {
		movie.Overview = sql.NullString{String: tmdbMovie.Overview, Valid: true}
	}

	// Set poster path
	if tmdbMovie.PosterPath != nil && *tmdbMovie.PosterPath != "" {
		movie.PosterPath = sql.NullString{String: *tmdbMovie.PosterPath, Valid: true}
	}

	// Set backdrop path
	if tmdbMovie.BackdropPath != nil && *tmdbMovie.BackdropPath != "" {
		movie.BackdropPath = sql.NullString{String: *tmdbMovie.BackdropPath, Valid: true}
	}

	// Set rating fields
	movie.VoteAverage = sql.NullFloat64{Float64: tmdbMovie.VoteAverage, Valid: true}
	movie.VoteCount = sql.NullInt64{Int64: int64(tmdbMovie.VoteCount), Valid: true}
	movie.Popularity = sql.NullFloat64{Float64: tmdbMovie.Popularity, Valid: true}

	// Also set the legacy rating field for backward compatibility
	movie.Rating = sql.NullFloat64{Float64: tmdbMovie.VoteAverage, Valid: true}

	// Set runtime
	if tmdbMovie.Runtime > 0 {
		movie.Runtime = sql.NullInt64{Int64: int64(tmdbMovie.Runtime), Valid: true}
	}

	// Set original language
	if tmdbMovie.OriginalLanguage != "" {
		movie.OriginalLanguage = sql.NullString{String: tmdbMovie.OriginalLanguage, Valid: true}
	}

	// Set status
	if tmdbMovie.Status != "" {
		movie.Status = sql.NullString{String: tmdbMovie.Status, Valid: true}
	}

	// Set IMDb ID
	if tmdbMovie.ImdbID != "" {
		movie.IMDbID = sql.NullString{String: tmdbMovie.ImdbID, Valid: true}
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
		movie.FilePath = sql.NullString{String: filePath, Valid: true}
	}

	// Set metadata source
	movie.MetadataSource = sql.NullString{String: string(models.MetadataSourceTMDb), Valid: true}

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
	series.TMDbID = sql.NullInt64{Int64: int64(tmdbSeries.ID), Valid: true}

	// Set original title
	if tmdbSeries.OriginalName != "" {
		series.OriginalTitle = sql.NullString{String: tmdbSeries.OriginalName, Valid: true}
	}

	// Set last air date
	if tmdbSeries.LastAirDate != "" {
		series.LastAirDate = sql.NullString{String: tmdbSeries.LastAirDate, Valid: true}
	}

	// Set overview
	if tmdbSeries.Overview != "" {
		series.Overview = sql.NullString{String: tmdbSeries.Overview, Valid: true}
	}

	// Set poster path
	if tmdbSeries.PosterPath != nil && *tmdbSeries.PosterPath != "" {
		series.PosterPath = sql.NullString{String: *tmdbSeries.PosterPath, Valid: true}
	}

	// Set backdrop path
	if tmdbSeries.BackdropPath != nil && *tmdbSeries.BackdropPath != "" {
		series.BackdropPath = sql.NullString{String: *tmdbSeries.BackdropPath, Valid: true}
	}

	// Set rating fields
	series.VoteAverage = sql.NullFloat64{Float64: tmdbSeries.VoteAverage, Valid: true}
	series.VoteCount = sql.NullInt64{Int64: int64(tmdbSeries.VoteCount), Valid: true}
	series.Popularity = sql.NullFloat64{Float64: tmdbSeries.Popularity, Valid: true}

	// Also set the legacy rating field for backward compatibility
	series.Rating = sql.NullFloat64{Float64: tmdbSeries.VoteAverage, Valid: true}

	// Set number of seasons and episodes
	series.NumberOfSeasons = sql.NullInt64{Int64: int64(tmdbSeries.NumberOfSeasons), Valid: true}
	series.NumberOfEpisodes = sql.NullInt64{Int64: int64(tmdbSeries.NumberOfEpisodes), Valid: true}

	// Set original language
	if tmdbSeries.OriginalLanguage != "" {
		series.OriginalLanguage = sql.NullString{String: tmdbSeries.OriginalLanguage, Valid: true}
	}

	// Set status
	if tmdbSeries.Status != "" {
		series.Status = sql.NullString{String: tmdbSeries.Status, Valid: true}
	}

	// Set in production flag
	series.InProduction = sql.NullBool{Bool: tmdbSeries.InProduction, Valid: true}

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
		series.FilePath = sql.NullString{String: filePath, Valid: true}
	}

	// Set metadata source
	series.MetadataSource = sql.NullString{String: string(models.MetadataSourceTMDb), Valid: true}

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
	movie.TMDbID = sql.NullInt64{Int64: int64(tmdbMovie.ID), Valid: true}

	// Set original title
	if tmdbMovie.OriginalTitle != "" {
		movie.OriginalTitle = sql.NullString{String: tmdbMovie.OriginalTitle, Valid: true}
	}

	// Set overview
	if tmdbMovie.Overview != "" {
		movie.Overview = sql.NullString{String: tmdbMovie.Overview, Valid: true}
	}

	// Set poster path
	if tmdbMovie.PosterPath != nil && *tmdbMovie.PosterPath != "" {
		movie.PosterPath = sql.NullString{String: *tmdbMovie.PosterPath, Valid: true}
	}

	// Set backdrop path
	if tmdbMovie.BackdropPath != nil && *tmdbMovie.BackdropPath != "" {
		movie.BackdropPath = sql.NullString{String: *tmdbMovie.BackdropPath, Valid: true}
	}

	// Set rating fields
	movie.VoteAverage = sql.NullFloat64{Float64: tmdbMovie.VoteAverage, Valid: true}
	movie.VoteCount = sql.NullInt64{Int64: int64(tmdbMovie.VoteCount), Valid: true}
	movie.Popularity = sql.NullFloat64{Float64: tmdbMovie.Popularity, Valid: true}
	movie.Rating = sql.NullFloat64{Float64: tmdbMovie.VoteAverage, Valid: true}

	// Set original language
	if tmdbMovie.OriginalLanguage != "" {
		movie.OriginalLanguage = sql.NullString{String: tmdbMovie.OriginalLanguage, Valid: true}
	}

	// Note: Search results only have genre IDs, not names
	// We store empty genres here; full details would need a separate API call
	movie.Genres = []string{}

	// Set file path if provided
	if filePath != "" {
		movie.FilePath = sql.NullString{String: filePath, Valid: true}
	}

	// Set metadata source
	movie.MetadataSource = sql.NullString{String: string(models.MetadataSourceTMDb), Valid: true}

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
	series.TMDbID = sql.NullInt64{Int64: int64(tmdbSeries.ID), Valid: true}

	// Set original title
	if tmdbSeries.OriginalName != "" {
		series.OriginalTitle = sql.NullString{String: tmdbSeries.OriginalName, Valid: true}
	}

	// Set overview
	if tmdbSeries.Overview != "" {
		series.Overview = sql.NullString{String: tmdbSeries.Overview, Valid: true}
	}

	// Set poster path
	if tmdbSeries.PosterPath != nil && *tmdbSeries.PosterPath != "" {
		series.PosterPath = sql.NullString{String: *tmdbSeries.PosterPath, Valid: true}
	}

	// Set backdrop path
	if tmdbSeries.BackdropPath != nil && *tmdbSeries.BackdropPath != "" {
		series.BackdropPath = sql.NullString{String: *tmdbSeries.BackdropPath, Valid: true}
	}

	// Set rating fields
	series.VoteAverage = sql.NullFloat64{Float64: tmdbSeries.VoteAverage, Valid: true}
	series.VoteCount = sql.NullInt64{Int64: int64(tmdbSeries.VoteCount), Valid: true}
	series.Popularity = sql.NullFloat64{Float64: tmdbSeries.Popularity, Valid: true}
	series.Rating = sql.NullFloat64{Float64: tmdbSeries.VoteAverage, Valid: true}

	// Set original language
	if tmdbSeries.OriginalLanguage != "" {
		series.OriginalLanguage = sql.NullString{String: tmdbSeries.OriginalLanguage, Valid: true}
	}

	// Note: Search results only have genre IDs, not names
	// We store empty genres here; full details would need a separate API call
	series.Genres = []string{}

	// Set file path if provided
	if filePath != "" {
		series.FilePath = sql.NullString{String: filePath, Valid: true}
	}

	// Set metadata source
	series.MetadataSource = sql.NullString{String: string(models.MetadataSourceTMDb), Valid: true}

	return series
}
