package tmdb

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
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

	if len(p.GenreIDs) > 0 {
		qp.Set("with_genres", joinInts(p.GenreIDs, ",")) // comma = AND
	}
	if p.Region != "" {
		qp.Set("region", p.Region)
	}
	// Only TMDb-recognized sort keys are forwarded. The local-library
	// "date added" sort (SortByDateAdded) has no TMDb equivalent — ordering by
	// when a title was added to the user's library is applied in the
	// application/library layer after fetch (Task 3.3; library sorting lives in
	// Story 5-4), never sent to TMDb (which would 400 on an unknown sort_by).
	if p.SortBy != "" && !isLocalSortKey(p.SortBy) {
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

	if p.VoteAverageGte > 0 {
		qp.Set("vote_average.gte", formatVote(p.VoteAverageGte))
	}
	if p.VoteAverageLte > 0 {
		qp.Set("vote_average.lte", formatVote(p.VoteAverageLte))
	}

	if len(p.WatchProviders) > 0 {
		qp.Set("with_watch_providers", joinInts(p.WatchProviders, "|")) // pipe = OR
		// TMDb only honors with_watch_providers alongside a watch_region.
		watchRegion := p.WatchRegion
		if watchRegion == "" {
			watchRegion = p.Region
		}
		if watchRegion == "" {
			watchRegion = "TW"
		}
		qp.Set("watch_region", watchRegion)
	}

	return qp
}

// SortByDateAdded is the compound-sort key for ordering by when a title was
// added to the local library. It is a local-only sort applied in the
// application layer after fetch — TMDb's /discover has no equivalent, so it is
// never forwarded as sort_by. (Story 11-1 AC #3, Task 3.3)
const SortByDateAdded = "date_added"

// isLocalSortKey reports whether a sort key is handled in the application layer
// rather than by TMDb (currently only the date-added family).
func isLocalSortKey(sortBy string) bool {
	switch sortBy {
	case SortByDateAdded, SortByDateAdded + ".asc", SortByDateAdded + ".desc":
		return true
	default:
		return false
	}
}

// joinInts renders a slice of ints as a sep-delimited string (e.g. {28,12} →
// "28,12"). Used to build with_genres (sep ",") and with_watch_providers
// (sep "|") query values.
func joinInts(ids []int, sep string) string {
	strs := make([]string, len(ids))
	for i, id := range ids {
		strs[i] = strconv.Itoa(id)
	}
	return strings.Join(strs, sep)
}

// formatVote renders a TMDb rating bound without a trailing ".0" for whole
// numbers (7.0 → "7", 7.5 → "7.5"), keeping query strings tidy.
func formatVote(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}

// ParseIntCSV parses a comma-separated list of integer IDs (e.g. "28,12") into
// a []int, silently skipping blank or non-numeric tokens. Returns nil for an
// empty/all-invalid input. Used by the HTTP handler and the explore-block
// service to map the `genre`/`watch_providers` wire params (and stored CSV)
// onto DiscoverParams.GenreIDs / DiscoverParams.WatchProviders.
func ParseIntCSV(csv string) []int {
	if csv == "" {
		return nil
	}
	parts := strings.Split(csv, ",")
	ids := make([]int, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if n, err := strconv.Atoi(p); err == nil {
			ids = append(ids, n)
		}
	}
	if len(ids) == 0 {
		return nil
	}
	return ids
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
