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
	// GetSeasonDetailsWithFallback gets season details, trying each language in the fallback chain
	GetSeasonDetailsWithFallback(ctx context.Context, tvID int, seasonNumber int) (*SeasonDetails, string, error)
	// GetTrendingMoviesWithFallback gets trending movies using the language fallback chain
	GetTrendingMoviesWithFallback(ctx context.Context, timeWindow string, page int) (*SearchResultMovies, string, error)
	// GetTrendingTVShowsWithFallback gets trending TV shows using the language fallback chain
	GetTrendingTVShowsWithFallback(ctx context.Context, timeWindow string, page int) (*SearchResultTVShows, string, error)
	// DiscoverMoviesWithFallback queries /discover/movie across the language fallback chain
	DiscoverMoviesWithFallback(ctx context.Context, params DiscoverParams) (*SearchResultMovies, string, error)
	// DiscoverTVShowsWithFallback queries /discover/tv across the language fallback chain
	DiscoverTVShowsWithFallback(ctx context.Context, params DiscoverParams) (*SearchResultTVShows, string, error)
	// GetMovieRecommendationsWithFallback gets recommended movies using the language fallback chain
	GetMovieRecommendationsWithFallback(ctx context.Context, movieID int) (*SearchResultMovies, string, error)
	// GetMovieSimilarWithFallback gets similar movies using the language fallback chain
	GetMovieSimilarWithFallback(ctx context.Context, movieID int) (*SearchResultMovies, string, error)
	// GetTVRecommendationsWithFallback gets recommended TV shows using the language fallback chain
	GetTVRecommendationsWithFallback(ctx context.Context, tvID int) (*SearchResultTVShows, string, error)
	// GetTVSimilarWithFallback gets similar TV shows using the language fallback chain
	GetTVSimilarWithFallback(ctx context.Context, tvID int) (*SearchResultTVShows, string, error)
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
	var merged *MovieDetails
	var baseLang string
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
		lastErr = nil

		if merged == nil {
			// The FIRST language that answers is the base: its values win every
			// field it actually has (bugfix-d — the chain leads with zh-TW).
			// CR M1: copy rather than alias — the merge writes into `merged`, and
			// mutating the client's response would make this function impure with
			// respect to its inputs (harmless against the live client, which
			// decodes a fresh struct per call, but it silently corrupts any
			// caller that hands back a shared value, e.g. a test fixture).
			base := *result
			merged = &base
			baseLang = lang
		} else {
			fillEmptyMovieFields(merged, result)
		}

		if hasLocalizedMovieDetails(merged) {
			slog.Debug("Language fallback: localized movie details complete",
				"base_language", baseLang,
				"completed_at_language", lang,
				"movie_id", movieID,
			)
			return merged, baseLang, nil
		}

		slog.Debug("Language fallback: movie details still incomplete",
			"language", lang,
			"movie_id", movieID,
		)
	}

	// CR M2: surface the error ONLY when nothing was collected. A later language
	// failing must not discard an earlier one's usable payload — throwing away a
	// good zh-TW answer because `en` timed out is the very waste this story
	// exists to stop.
	if merged == nil && lastErr != nil {
		return nil, "", lastErr
	}

	return merged, baseLang, nil
}

// fillEmptyMovieFields copies localized fields from src into dst ONLY where dst
// is empty (bugfix-d D1/D4).
//
// Before bugfix-d the fallback was all-or-nothing: hasLocalizedMovieDetails
// requires BOTH title and overview, so a movie with a perfectly good zh-TW title
// but an empty zh-TW overview — common for niche titles — failed the check and
// the WHOLE zh-CN (or en) payload was returned instead. That is why the library
// showed 简体 genres (动作/冒险) and romanized titles ("Mag Mag" over 禍禍女):
// one missing field dragged every other field down the chain with it.
//
// Merging per field keeps zh-TW for everything zh-TW actually has, and borrows
// only the genuinely missing pieces. Genres come back populated in every
// language, so zh-TW genre names now always win.
func fillEmptyMovieFields(dst, src *MovieDetails) {
	if dst == nil || src == nil {
		return
	}
	if dst.Title == "" {
		dst.Title = src.Title
	}
	if dst.Overview == "" {
		dst.Overview = src.Overview
	}
	if dst.Tagline == "" {
		dst.Tagline = src.Tagline
	}
	if len(dst.Genres) == 0 {
		dst.Genres = src.Genres
	}
	// TMDb serves per-language artwork; a locale with no poster of its own
	// should still show the movie rather than a blank card.
	if isBlankPath(dst.PosterPath) {
		dst.PosterPath = src.PosterPath
	}
	if isBlankPath(dst.BackdropPath) {
		dst.BackdropPath = src.BackdropPath
	}
}

// isBlankPath treats a nil pointer and a pointer to "" as the same absent value —
// TMDb uses both for "no image in this locale".
func isBlankPath(p *string) bool { return p == nil || *p == "" }

// GetTVShowDetailsWithFallback gets TV show details, trying each language in the fallback chain
// Returns the details, the language used, and any error
func (c *LanguageFallbackClient) GetTVShowDetailsWithFallback(ctx context.Context, tvID int) (*TVShowDetails, string, error) {
	var merged *TVShowDetails
	var baseLang string
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
		lastErr = nil

		if merged == nil {
			// CR M1 — copy, don't alias. See GetMovieDetailsWithFallback.
			base := *result
			merged = &base
			baseLang = lang
		} else {
			fillEmptyTVShowFields(merged, result)
		}

		if hasLocalizedTVShowDetails(merged) {
			slog.Debug("Language fallback: localized TV show details complete",
				"base_language", baseLang,
				"completed_at_language", lang,
				"tv_id", tvID,
			)
			return merged, baseLang, nil
		}

		slog.Debug("Language fallback: TV show details still incomplete",
			"language", lang,
			"tv_id", tvID,
		)
	}

	// CR M2 — see GetMovieDetailsWithFallback.
	if merged == nil && lastErr != nil {
		return nil, "", lastErr
	}

	return merged, baseLang, nil
}

// fillEmptyTVShowFields is fillEmptyMovieFields for series — same bugfix-d
// rationale: a series with a zh-TW name but no zh-TW overview must not have its
// genres and name dragged to 简体 along with the borrowed overview.
func fillEmptyTVShowFields(dst, src *TVShowDetails) {
	if dst == nil || src == nil {
		return
	}
	if dst.Name == "" {
		dst.Name = src.Name
	}
	if dst.Overview == "" {
		dst.Overview = src.Overview
	}
	if dst.Tagline == "" {
		dst.Tagline = src.Tagline
	}
	if len(dst.Genres) == 0 {
		dst.Genres = src.Genres
	}
	if isBlankPath(dst.PosterPath) {
		dst.PosterPath = src.PosterPath
	}
	if isBlankPath(dst.BackdropPath) {
		dst.BackdropPath = src.BackdropPath
	}
}

// GetSeasonDetailsWithFallback gets season details, trying each language in the fallback chain.
// A season is considered localized when at least one episode has a non-empty name, mirroring
// the title/overview localization checks used for TV-show details.
func (c *LanguageFallbackClient) GetSeasonDetailsWithFallback(ctx context.Context, tvID int, seasonNumber int) (*SeasonDetails, string, error) {
	var merged *SeasonDetails
	var baseLang string
	var lastErr error

	for _, lang := range c.languages {
		result, err := c.client.GetSeasonDetailsWithLanguage(ctx, tvID, seasonNumber, lang)
		if err != nil {
			slog.Debug("Language fallback: get season details failed",
				"language", lang,
				"tv_id", tvID,
				"season_number", seasonNumber,
				"error", err,
			)
			lastErr = err
			continue
		}
		lastErr = nil

		if merged == nil {
			// CR M3 — seasons ride the same all-or-nothing defect: a season whose
			// zh-TW response names NO episode used to hand the whole season to
			// zh-CN, taking the season name, overview and artwork down with it.
			// Merging keeps whatever zh-TW did supply and borrows only the rest.
			//
			// SCOPE LIMIT (filed, not fixed here): hasLocalizedSeasonDetails is
			// satisfied by a SINGLE named episode, so a season where zh-TW names
			// some episodes but not others still returns early and the unnamed
			// ones stay blank. Tightening that check to "every episode named"
			// would walk the whole chain for any season TMDb has not fully
			// localized in ANY language — a per-season call increase this story
			// is not authorized to make (AC #5 red-lines the rate-limit layer).
			//
			// Copy the base (CR M1); the episode slice needs its own copy too,
			// since the per-episode merge writes through it.
			base := *result
			base.Episodes = append([]EpisodeInfo(nil), result.Episodes...)
			merged = &base
			baseLang = lang
		} else {
			fillEmptySeasonFields(merged, result)
		}

		if hasLocalizedSeasonDetails(merged) {
			slog.Debug("Language fallback: localized season details complete",
				"base_language", baseLang,
				"completed_at_language", lang,
				"tv_id", tvID,
				"season_number", seasonNumber,
			)
			return merged, baseLang, nil
		}

		slog.Debug("Language fallback: season details still incomplete",
			"language", lang,
			"tv_id", tvID,
			"season_number", seasonNumber,
		)
	}

	// CR M2 — see GetMovieDetailsWithFallback.
	if merged == nil && lastErr != nil {
		return nil, "", lastErr
	}

	return merged, baseLang, nil
}

// fillEmptySeasonFields fills the season's own empty fields and then each
// episode's, matching episodes by EpisodeNumber (stable across locales — the
// slice order and the episode ID are not guaranteed to be). bugfix-d CR M3.
func fillEmptySeasonFields(dst, src *SeasonDetails) {
	if dst == nil || src == nil {
		return
	}
	if dst.Name == "" {
		dst.Name = src.Name
	}
	if dst.Overview == "" {
		dst.Overview = src.Overview
	}
	if isBlankPath(dst.PosterPath) {
		dst.PosterPath = src.PosterPath
	}

	byNumber := make(map[int]EpisodeInfo, len(src.Episodes))
	for _, ep := range src.Episodes {
		byNumber[ep.EpisodeNumber] = ep
	}
	for i := range dst.Episodes {
		fallback, ok := byNumber[dst.Episodes[i].EpisodeNumber]
		if !ok {
			continue
		}
		if dst.Episodes[i].Name == "" {
			dst.Episodes[i].Name = fallback.Name
		}
		if dst.Episodes[i].Overview == "" {
			dst.Episodes[i].Overview = fallback.Overview
		}
		if isBlankPath(dst.Episodes[i].StillPath) {
			dst.Episodes[i].StillPath = fallback.StillPath
		}
	}
}

// hasLocalizedSeasonDetails reports whether a season-details response carries
// localized content (at least one episode with a non-empty name).
func hasLocalizedSeasonDetails(s *SeasonDetails) bool {
	if s == nil {
		return false
	}
	for _, ep := range s.Episodes {
		if ep.Name != "" {
			return true
		}
	}
	return false
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
//
// If a caller-provided language yields zero results (e.g. an unsupported locale
// like "zz" or a locale with no catalog coverage), we log a warning so operators
// can detect mistyped language codes without the request silently degrading.
func (c *LanguageFallbackClient) DiscoverMoviesWithFallback(ctx context.Context, params DiscoverParams) (*SearchResultMovies, string, error) {
	if params.Language != "" {
		result, err := c.client.DiscoverMovies(ctx, params)
		if err != nil {
			return nil, "", err
		}
		if result != nil && len(result.Results) == 0 {
			slog.Warn("Discover movies: caller-provided language returned empty results (fallback chain skipped)",
				"language", params.Language,
				"genre_ids", params.GenreIDs,
				"year_gte", params.YearGte,
				"year_lte", params.YearLte,
				"region", params.Region,
			)
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
// (see DiscoverMoviesWithFallback for semantics, including the empty-result warning
// when a caller-provided language yields nothing).
func (c *LanguageFallbackClient) DiscoverTVShowsWithFallback(ctx context.Context, params DiscoverParams) (*SearchResultTVShows, string, error) {
	if params.Language != "" {
		result, err := c.client.DiscoverTVShows(ctx, params)
		if err != nil {
			return nil, "", err
		}
		if result != nil && len(result.Results) == 0 {
			slog.Warn("Discover TV shows: caller-provided language returned empty results (fallback chain skipped)",
				"language", params.Language,
				"genre_ids", params.GenreIDs,
				"year_gte", params.YearGte,
				"year_lte", params.YearLte,
				"region", params.Region,
			)
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

// GetMovieRecommendationsWithFallback gets recommended movies, trying each language
// in the fallback chain (titles/overviews are language-specific, so we fall back if
// the first language returns items without localized content — mirrors trending).
func (c *LanguageFallbackClient) GetMovieRecommendationsWithFallback(ctx context.Context, movieID int) (*SearchResultMovies, string, error) {
	var lastResult *SearchResultMovies
	var lastLang string
	var lastErr error

	for _, lang := range c.languages {
		result, err := c.client.GetMovieRecommendationsWithLanguage(ctx, movieID, lang)
		if err != nil {
			slog.Debug("Language fallback: movie recommendations failed",
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

		if len(result.Results) > 0 && hasLocalizedMovieContent(result.Results) {
			return result, lang, nil
		}
	}

	if lastErr != nil {
		return nil, "", lastErr
	}
	if lastResult == nil {
		return &SearchResultMovies{Results: []Movie{}}, c.languages[len(c.languages)-1], nil
	}
	return lastResult, lastLang, nil
}

// GetMovieSimilarWithFallback gets similar movies using the language fallback chain.
func (c *LanguageFallbackClient) GetMovieSimilarWithFallback(ctx context.Context, movieID int) (*SearchResultMovies, string, error) {
	var lastResult *SearchResultMovies
	var lastLang string
	var lastErr error

	for _, lang := range c.languages {
		result, err := c.client.GetMovieSimilarWithLanguage(ctx, movieID, lang)
		if err != nil {
			slog.Debug("Language fallback: similar movies failed",
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

		if len(result.Results) > 0 && hasLocalizedMovieContent(result.Results) {
			return result, lang, nil
		}
	}

	if lastErr != nil {
		return nil, "", lastErr
	}
	if lastResult == nil {
		return &SearchResultMovies{Results: []Movie{}}, c.languages[len(c.languages)-1], nil
	}
	return lastResult, lastLang, nil
}

// GetTVRecommendationsWithFallback gets recommended TV shows using the language fallback chain.
func (c *LanguageFallbackClient) GetTVRecommendationsWithFallback(ctx context.Context, tvID int) (*SearchResultTVShows, string, error) {
	var lastResult *SearchResultTVShows
	var lastLang string
	var lastErr error

	for _, lang := range c.languages {
		result, err := c.client.GetTVRecommendationsWithLanguage(ctx, tvID, lang)
		if err != nil {
			slog.Debug("Language fallback: TV recommendations failed",
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

		if len(result.Results) > 0 && hasLocalizedTVShowContent(result.Results) {
			return result, lang, nil
		}
	}

	if lastErr != nil {
		return nil, "", lastErr
	}
	if lastResult == nil {
		return &SearchResultTVShows{Results: []TVShow{}}, c.languages[len(c.languages)-1], nil
	}
	return lastResult, lastLang, nil
}

// GetTVSimilarWithFallback gets similar TV shows using the language fallback chain.
func (c *LanguageFallbackClient) GetTVSimilarWithFallback(ctx context.Context, tvID int) (*SearchResultTVShows, string, error) {
	var lastResult *SearchResultTVShows
	var lastLang string
	var lastErr error

	for _, lang := range c.languages {
		result, err := c.client.GetTVSimilarWithLanguage(ctx, tvID, lang)
		if err != nil {
			slog.Debug("Language fallback: similar TV shows failed",
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

		if len(result.Results) > 0 && hasLocalizedTVShowContent(result.Results) {
			return result, lang, nil
		}
	}

	if lastErr != nil {
		return nil, "", lastErr
	}
	if lastResult == nil {
		return &SearchResultTVShows{Results: []TVShow{}}, c.languages[len(c.languages)-1], nil
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
