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
	// GetTrendingMoviesWithFallback gets trending movies using the language fallback chain
	GetTrendingMoviesWithFallback(ctx context.Context, timeWindow string, page int) (*SearchResultMovies, string, error)
	// GetTrendingTVShowsWithFallback gets trending TV shows using the language fallback chain
	GetTrendingTVShowsWithFallback(ctx context.Context, timeWindow string, page int) (*SearchResultTVShows, string, error)
	// DiscoverMoviesWithFallback queries /discover/movie across the language fallback chain
	DiscoverMoviesWithFallback(ctx context.Context, params DiscoverParams) (*SearchResultMovies, string, error)
	// DiscoverTVShowsWithFallback queries /discover/tv across the language fallback chain
	DiscoverTVShowsWithFallback(ctx context.Context, params DiscoverParams) (*SearchResultTVShows, string, error)
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

// GetTrendingMoviesWithFallback gets trending movies, trying each language in the fallback chain.
// Trending lists themselves don't depend on language (same global popularity list), but result
// titles/overviews are language-specific — so we fall back if the first language returns items
// without localized content, matching the existing search/detail behavior.
func (c *LanguageFallbackClient) GetTrendingMoviesWithFallback(ctx context.Context, timeWindow string, page int) (*SearchResultMovies, string, error) {
	var lastResult *SearchResultMovies
	var lastLang string
	var lastErr error

	for _, lang := range c.languages {
		result, err := c.client.GetTrendingMoviesWithLanguage(ctx, timeWindow, lang, page)
		if err != nil {
			slog.Debug("Language fallback: trending movies failed",
				"language", lang,
				"time_window", timeWindow,
				"error", err,
			)
			lastErr = err
			continue
		}

		lastResult = result
		lastLang = lang
		lastErr = nil

		if len(result.Results) > 0 && hasLocalizedMovieContent(result.Results) {
			return result, lang, nil
		}
	}

	if lastErr != nil {
		return nil, "", lastErr
	}
	if lastResult == nil {
		return &SearchResultMovies{Page: page, Results: []Movie{}}, c.languages[len(c.languages)-1], nil
	}
	return lastResult, lastLang, nil
}

// GetTrendingTVShowsWithFallback gets trending TV shows using the language fallback chain.
func (c *LanguageFallbackClient) GetTrendingTVShowsWithFallback(ctx context.Context, timeWindow string, page int) (*SearchResultTVShows, string, error) {
	var lastResult *SearchResultTVShows
	var lastLang string
	var lastErr error

	for _, lang := range c.languages {
		result, err := c.client.GetTrendingTVShowsWithLanguage(ctx, timeWindow, lang, page)
		if err != nil {
			slog.Debug("Language fallback: trending TV shows failed",
				"language", lang,
				"time_window", timeWindow,
				"error", err,
			)
			lastErr = err
			continue
		}

		lastResult = result
		lastLang = lang
		lastErr = nil

		if len(result.Results) > 0 && hasLocalizedTVShowContent(result.Results) {
			return result, lang, nil
		}
	}

	if lastErr != nil {
		return nil, "", lastErr
	}
	if lastResult == nil {
		return &SearchResultTVShows{Page: page, Results: []TVShow{}}, c.languages[len(c.languages)-1], nil
	}
	return lastResult, lastLang, nil
}

// DiscoverMoviesWithFallback runs /discover/movie across the language fallback chain.
// When params.Language is set explicitly by the caller, it is honored on the first attempt
// and the chain is only consulted if subsequent localization checks fail — but because
// discover results are already language-filtered by the caller's intent, we treat a
// caller-provided language as authoritative and skip the chain in that case.
func (c *LanguageFallbackClient) DiscoverMoviesWithFallback(ctx context.Context, params DiscoverParams) (*SearchResultMovies, string, error) {
	if params.Language != "" {
		result, err := c.client.DiscoverMovies(ctx, params)
		if err != nil {
			return nil, "", err
		}
		return result, params.Language, nil
	}

	var lastResult *SearchResultMovies
	var lastLang string
	var lastErr error

	for _, lang := range c.languages {
		p := params
		p.Language = lang
		result, err := c.client.DiscoverMovies(ctx, p)
		if err != nil {
			slog.Debug("Language fallback: discover movies failed",
				"language", lang,
				"error", err,
			)
			lastErr = err
			continue
		}

		lastResult = result
		lastLang = lang
		lastErr = nil

		if len(result.Results) > 0 && hasLocalizedMovieContent(result.Results) {
			return result, lang, nil
		}
	}

	if lastErr != nil {
		return nil, "", lastErr
	}
	if lastResult == nil {
		return &SearchResultMovies{Page: 1, Results: []Movie{}}, c.languages[len(c.languages)-1], nil
	}
	return lastResult, lastLang, nil
}

// DiscoverTVShowsWithFallback runs /discover/tv across the language fallback chain
// (see DiscoverMoviesWithFallback for semantics).
func (c *LanguageFallbackClient) DiscoverTVShowsWithFallback(ctx context.Context, params DiscoverParams) (*SearchResultTVShows, string, error) {
	if params.Language != "" {
		result, err := c.client.DiscoverTVShows(ctx, params)
		if err != nil {
			return nil, "", err
		}
		return result, params.Language, nil
	}

	var lastResult *SearchResultTVShows
	var lastLang string
	var lastErr error

	for _, lang := range c.languages {
		p := params
		p.Language = lang
		result, err := c.client.DiscoverTVShows(ctx, p)
		if err != nil {
			slog.Debug("Language fallback: discover TV shows failed",
				"language", lang,
				"error", err,
			)
			lastErr = err
			continue
		}

		lastResult = result
		lastLang = lang
		lastErr = nil

		if len(result.Results) > 0 && hasLocalizedTVShowContent(result.Results) {
			return result, lang, nil
		}
	}

	if lastErr != nil {
		return nil, "", lastErr
	}
	if lastResult == nil {
		return &SearchResultTVShows{Page: 1, Results: []TVShow{}}, c.languages[len(c.languages)-1], nil
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
