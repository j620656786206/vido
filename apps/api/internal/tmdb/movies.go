package tmdb

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// SearchMovies searches for movies by title and returns paginated results
// The results will be in the language specified by the client (e.g., zh-TW)
func (c *Client) SearchMovies(ctx context.Context, query string, page int) (*SearchResultMovies, error) {
	return c.SearchMoviesWithLanguage(ctx, query, c.language, page)
}

// SearchMoviesWithLanguage searches for movies with a specific language
// This is used by the language fallback chain
func (c *Client) SearchMoviesWithLanguage(ctx context.Context, query string, language string, page int) (*SearchResultMovies, error) {
	// Validate input
	if query == "" {
		return nil, NewBadRequestError("search query cannot be empty")
	}

	if page < 1 {
		page = 1
	}

	// Build query parameters
	queryParams := url.Values{
		"query":    []string{query},
		"page":     []string{strconv.Itoa(page)},
		"language": []string{language},
	}

	// Make API request
	var result SearchResultMovies
	if err := c.Get(ctx, "/search/movie", queryParams, &result); err != nil {
		return nil, fmt.Errorf("failed to search movies: %w", err)
	}

	return &result, nil
}

// GetMovieDetails retrieves complete movie information for a specific movie ID
// The results will be in the language specified by the client (e.g., zh-TW)
func (c *Client) GetMovieDetails(ctx context.Context, movieID int) (*MovieDetails, error) {
	return c.GetMovieDetailsWithLanguage(ctx, movieID, c.language)
}

// GetMovieDetailsWithLanguage retrieves movie details with a specific language
// This is used by the language fallback chain
func (c *Client) GetMovieDetailsWithLanguage(ctx context.Context, movieID int, language string) (*MovieDetails, error) {
	// Validate input
	if movieID <= 0 {
		return nil, NewBadRequestError("movie ID must be greater than 0")
	}

	// Build endpoint path
	endpoint := fmt.Sprintf("/movie/%d", movieID)

	// Build query parameters with language
	queryParams := url.Values{
		"language": []string{language},
	}

	// Make API request
	var result MovieDetails
	if err := c.Get(ctx, endpoint, queryParams, &result); err != nil {
		return nil, fmt.Errorf("failed to get movie details: %w", err)
	}

	return &result, nil
}

// FindByExternalID finds movies/TV shows using an external ID (e.g., IMDB ID).
// externalSource should be "imdb_id", "tvdb_id", etc.
func (c *Client) FindByExternalID(ctx context.Context, externalID string, externalSource string) (*FindByExternalIDResponse, error) {
	if externalID == "" {
		return nil, NewBadRequestError("external ID cannot be empty")
	}
	if externalSource == "" {
		return nil, NewBadRequestError("external source cannot be empty")
	}

	endpoint := fmt.Sprintf("/find/%s", url.QueryEscape(externalID))
	queryParams := url.Values{
		"external_source": []string{externalSource},
		"language":        []string{c.language},
	}

	var result FindByExternalIDResponse
	if err := c.Get(ctx, endpoint, queryParams, &result); err != nil {
		return nil, fmt.Errorf("failed to find by external ID: %w", err)
	}

	return &result, nil
}

// validTrendingWindows lists the time windows TMDb accepts for trending endpoints.
var validTrendingWindows = map[string]struct{}{"day": {}, "week": {}}

// normalizeTrendingWindow returns a valid TMDb trending window, defaulting
// to "week" when the caller passes an unknown or empty value.
func normalizeTrendingWindow(window string) string {
	if _, ok := validTrendingWindows[window]; ok {
		return window
	}
	return "week"
}

// GetTrendingMovies fetches the trending movies for a time window ("day" or "week").
// The results will be in the language configured on the client (e.g., zh-TW).
func (c *Client) GetTrendingMovies(ctx context.Context, timeWindow string, page int) (*SearchResultMovies, error) {
	return c.GetTrendingMoviesWithLanguage(ctx, timeWindow, c.language, page)
}

// GetTrendingMoviesWithLanguage fetches trending movies with a specific language,
// used by the language fallback chain.
func (c *Client) GetTrendingMoviesWithLanguage(ctx context.Context, timeWindow string, language string, page int) (*SearchResultMovies, error) {
	if page < 1 {
		page = 1
	}

	endpoint := fmt.Sprintf("/trending/movie/%s", normalizeTrendingWindow(timeWindow))
	queryParams := url.Values{
		"page":     []string{strconv.Itoa(page)},
		"language": []string{language},
	}

	var result SearchResultMovies
	if err := c.Get(ctx, endpoint, queryParams, &result); err != nil {
		return nil, fmt.Errorf("failed to get trending movies: %w", err)
	}
	return &result, nil
}

// DiscoverMovies queries /discover/movie with the given filter params. When
// params.Language is empty, the client's default language is used.
func (c *Client) DiscoverMovies(ctx context.Context, params DiscoverParams) (*SearchResultMovies, error) {
	queryParams := discoverQueryParams(params, true /*movie*/, c.language)

	var result SearchResultMovies
	if err := c.Get(ctx, "/discover/movie", queryParams, &result); err != nil {
		return nil, fmt.Errorf("failed to discover movies: %w", err)
	}
	return &result, nil
}

// discoverQueryParams builds a TMDb query string from DiscoverParams.
// forMovies controls whether YearGte/YearLte map to primary_release_date
// (movies) or first_air_date (TV). Zero-valued fields are omitted so TMDb
// applies its own defaults.
func discoverQueryParams(p DiscoverParams, forMovies bool, defaultLanguage string) url.Values {
	qp := url.Values{}

	page := p.Page
	if page < 1 {
		page = 1
	}
	qp.Set("page", strconv.Itoa(page))

	language := p.Language
	if language == "" {
		language = defaultLanguage
	}
	qp.Set("language", language)

	if p.Genre != "" {
		qp.Set("with_genres", p.Genre)
	}
	if p.Region != "" {
		qp.Set("region", p.Region)
	}
	if p.SortBy != "" {
		qp.Set("sort_by", p.SortBy)
	}

	dateKeyGte, dateKeyLte := "primary_release_date.gte", "primary_release_date.lte"
	if !forMovies {
		dateKeyGte, dateKeyLte = "first_air_date.gte", "first_air_date.lte"
	}
	if p.YearGte > 0 {
		qp.Set(dateKeyGte, fmt.Sprintf("%04d-01-01", p.YearGte))
	}
	if p.YearLte > 0 {
		qp.Set(dateKeyLte, fmt.Sprintf("%04d-12-31", p.YearLte))
	}

	return qp
}

// GetMovieVideos retrieves videos (trailers, teasers, etc.) for a movie
func (c *Client) GetMovieVideos(ctx context.Context, movieID int) (*VideosResponse, error) {
	if movieID <= 0 {
		return nil, NewBadRequestError("movie ID must be greater than 0")
	}

	endpoint := fmt.Sprintf("/movie/%d/videos", movieID)
	queryParams := url.Values{
		"language": []string{c.language},
	}

	var result VideosResponse
	if err := c.Get(ctx, endpoint, queryParams, &result); err != nil {
		return nil, fmt.Errorf("failed to get movie videos: %w", err)
	}

	return &result, nil
}
