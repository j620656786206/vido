package tmdb

import (
	"context"
	"log/slog"
)

// DefaultFallbackLanguages defines the default language fallback chain
// zh-TW (Traditional Chinese) → zh-CN (Simplified Chinese) → en (English)
var DefaultFallbackLanguages = []string{"zh-TW", "zh-CN", "en"}

// LanguageFallbackClient wraps a TMDb client and provides automatic language fallback
// When searching or getting details, it tries each language in the fallback chain
// until it finds results with localized content
type LanguageFallbackClient struct {
	client    ClientInterface
	languages []string
}

// LanguageFallbackClientInterface defines the contract for language fallback operations
type LanguageFallbackClientInterface interface {
	// SearchMoviesWithFallback searches for movies, trying each language in the fallback chain
	SearchMoviesWithFallback(ctx context.Context, query string, page int) (*SearchResultMovies, string, error)
	// SearchTVShowsWithFallback searches for TV shows, trying each language in the fallback chain
	SearchTVShowsWithFallback(ctx context.Context, query string, page int) (*SearchResultTVShows, string, error)
	// GetMovieDetailsWithFallback gets movie details, trying each language in the fallback chain
	GetMovieDetailsWithFallback(ctx context.Context, movieID int) (*MovieDetails, string, error)
	// GetTVShowDetailsWithFallback gets TV show details, trying each language in the fallback chain
	GetTVShowDetailsWithFallback(ctx context.Context, tvID int) (*TVShowDetails, string, error)
}

// Compile-time interface verification
var _ LanguageFallbackClientInterface = (*LanguageFallbackClient)(nil)

// NewLanguageFallbackClient creates a new LanguageFallbackClient with the given client and languages
// If languages is nil or empty, DefaultFallbackLanguages is used
func NewLanguageFallbackClient(client ClientInterface, languages []string) *LanguageFallbackClient {
	if len(languages) == 0 {
		languages = DefaultFallbackLanguages
	}
	return &LanguageFallbackClient{
		client:    client,
		languages: languages,
	}
}

// SearchMoviesWithFallback searches for movies, trying each language in the fallback chain
// Returns the results, the language used, and any error
// If all languages return empty results, returns results from the last language tried
func (c *LanguageFallbackClient) SearchMoviesWithFallback(ctx context.Context, query string, page int) (*SearchResultMovies, string, error) {
	var lastResult *SearchResultMovies
	var lastLang string
	var lastErr error

	for _, lang := range c.languages {
		result, err := c.client.SearchMoviesWithLanguage(ctx, query, lang, page)
		if err != nil {
			slog.Debug("Language fallback: search movies failed",
				"language", lang,
				"query", query,
				"error", err,
			)
			lastErr = err
			continue
		}

		lastResult = result
		lastLang = lang
		lastErr = nil

		// Check if we have results with localized content
		if len(result.Results) > 0 && hasLocalizedMovieContent(result.Results) {
			slog.Debug("Language fallback: found localized movie content",
				"language", lang,
				"query", query,
				"results", len(result.Results),
			)
			return result, lang, nil
		}

		slog.Debug("Language fallback: no localized movie content",
			"language", lang,
			"query", query,
			"results", len(result.Results),
		)
	}

	// Return whatever we got from the last attempt
	if lastErr != nil {
		return nil, "", lastErr
	}

	if lastResult == nil {
		// All attempts failed, return empty result
		return &SearchResultMovies{
			Page:         page,
			Results:      []Movie{},
			TotalPages:   0,
			TotalResults: 0,
		}, c.languages[len(c.languages)-1], nil
	}

	return lastResult, lastLang, nil
}

// SearchTVShowsWithFallback searches for TV shows, trying each language in the fallback chain
// Returns the results, the language used, and any error
func (c *LanguageFallbackClient) SearchTVShowsWithFallback(ctx context.Context, query string, page int) (*SearchResultTVShows, string, error) {
	var lastResult *SearchResultTVShows
	var lastLang string
	var lastErr error

	for _, lang := range c.languages {
		result, err := c.client.SearchTVShowsWithLanguage(ctx, query, lang, page)
		if err != nil {
			slog.Debug("Language fallback: search TV shows failed",
				"language", lang,
				"query", query,
				"error", err,
			)
			lastErr = err
			continue
		}

		lastResult = result
		lastLang = lang
		lastErr = nil

		// Check if we have results with localized content
		if len(result.Results) > 0 && hasLocalizedTVShowContent(result.Results) {
			slog.Debug("Language fallback: found localized TV show content",
				"language", lang,
				"query", query,
				"results", len(result.Results),
			)
			return result, lang, nil
		}

		slog.Debug("Language fallback: no localized TV show content",
			"language", lang,
			"query", query,
			"results", len(result.Results),
		)
	}

	if lastErr != nil {
		return nil, "", lastErr
	}

	if lastResult == nil {
		return &SearchResultTVShows{
			Page:         page,
			Results:      []TVShow{},
			TotalPages:   0,
			TotalResults: 0,
		}, c.languages[len(c.languages)-1], nil
	}

	return lastResult, lastLang, nil
}

// GetMovieDetailsWithFallback gets movie details, trying each language in the fallback chain
// Returns the details, the language used, and any error
func (c *LanguageFallbackClient) GetMovieDetailsWithFallback(ctx context.Context, movieID int) (*MovieDetails, string, error) {
	var lastResult *MovieDetails
	var lastLang string
	var lastErr error

	for _, lang := range c.languages {
		result, err := c.client.GetMovieDetailsWithLanguage(ctx, movieID, lang)
		if err != nil {
			slog.Debug("Language fallback: get movie details failed",
				"language", lang,
				"movie_id", movieID,
				"error", err,
			)
			lastErr = err
			continue
		}

		lastResult = result
		lastLang = lang
		lastErr = nil

		// Check if we have localized content (non-empty title and overview)
		if hasLocalizedMovieDetails(result) {
			slog.Debug("Language fallback: found localized movie details",
				"language", lang,
				"movie_id", movieID,
			)
			return result, lang, nil
		}

		slog.Debug("Language fallback: no localized movie details",
			"language", lang,
			"movie_id", movieID,
		)
	}

	if lastErr != nil {
		return nil, "", lastErr
	}

	return lastResult, lastLang, nil
}

// GetTVShowDetailsWithFallback gets TV show details, trying each language in the fallback chain
// Returns the details, the language used, and any error
func (c *LanguageFallbackClient) GetTVShowDetailsWithFallback(ctx context.Context, tvID int) (*TVShowDetails, string, error) {
	var lastResult *TVShowDetails
	var lastLang string
	var lastErr error

	for _, lang := range c.languages {
		result, err := c.client.GetTVShowDetailsWithLanguage(ctx, tvID, lang)
		if err != nil {
			slog.Debug("Language fallback: get TV show details failed",
				"language", lang,
				"tv_id", tvID,
				"error", err,
			)
			lastErr = err
			continue
		}

		lastResult = result
		lastLang = lang
		lastErr = nil

		// Check if we have localized content
		if hasLocalizedTVShowDetails(result) {
			slog.Debug("Language fallback: found localized TV show details",
				"language", lang,
				"tv_id", tvID,
			)
			return result, lang, nil
		}

		slog.Debug("Language fallback: no localized TV show details",
			"language", lang,
			"tv_id", tvID,
		)
	}

	if lastErr != nil {
		return nil, "", lastErr
	}

	return lastResult, lastLang, nil
}

// hasLocalizedMovieContent checks if any movie in the results has localized content
// Content is considered localized if it has a non-empty title and overview
func hasLocalizedMovieContent(movies []Movie) bool {
	for _, m := range movies {
		if m.Title != "" && m.Overview != "" {
			return true
		}
	}
	return false
}

// hasLocalizedTVShowContent checks if any TV show in the results has localized content
func hasLocalizedTVShowContent(shows []TVShow) bool {
	for _, s := range shows {
		if s.Name != "" && s.Overview != "" {
			return true
		}
	}
	return false
}

// hasLocalizedMovieDetails checks if movie details have localized content
func hasLocalizedMovieDetails(m *MovieDetails) bool {
	return m != nil && m.Title != "" && m.Overview != ""
}

// hasLocalizedTVShowDetails checks if TV show details have localized content
func hasLocalizedTVShowDetails(s *TVShowDetails) bool {
	return s != nil && s.Name != "" && s.Overview != ""
}
