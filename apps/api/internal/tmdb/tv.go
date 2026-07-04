package tmdb

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// SearchTVShows searches for TV shows by name and returns paginated results
// The results will be in the language specified by the client (e.g., zh-TW)
func (c *Client) SearchTVShows(ctx context.Context, query string, page int) (*SearchResultTVShows, error) {
	return c.SearchTVShowsWithLanguage(ctx, query, c.language, page)
}

// SearchTVShowsWithLanguage searches for TV shows with a specific language
// This is used by the language fallback chain
func (c *Client) SearchTVShowsWithLanguage(ctx context.Context, query string, language string, page int) (*SearchResultTVShows, error) {
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
	var result SearchResultTVShows
	if err := c.Get(ctx, "/search/tv", queryParams, &result); err != nil {
		return nil, fmt.Errorf("failed to search TV shows: %w", err)
	}

	return &result, nil
}

// GetTVShowDetails retrieves complete TV show information for a specific TV show ID
// The results will be in the language specified by the client (e.g., zh-TW)
func (c *Client) GetTVShowDetails(ctx context.Context, tvID int) (*TVShowDetails, error) {
	return c.GetTVShowDetailsWithLanguage(ctx, tvID, c.language)
}

// GetTVShowDetailsWithLanguage retrieves TV show details with a specific language
// This is used by the language fallback chain
func (c *Client) GetTVShowDetailsWithLanguage(ctx context.Context, tvID int, language string) (*TVShowDetails, error) {
	// Validate input
	if tvID <= 0 {
		return nil, NewBadRequestError("TV show ID must be greater than 0")
	}

	// Build endpoint path
	endpoint := fmt.Sprintf("/tv/%d", tvID)

	// Build query parameters with language
	queryParams := url.Values{
		"language": []string{language},
	}

	// Make API request
	var result TVShowDetails
	if err := c.Get(ctx, endpoint, queryParams, &result); err != nil {
		return nil, fmt.Errorf("failed to get TV show details: %w", err)
	}

	return &result, nil
}

// TVExternalIDs holds the external-service ids for a TV show
// (GET /tv/{id}/external_ids). Only the ids the DVR pipeline consumes are
// mapped; tvdb_id null resolves to 0 (title absent from TVDB — the Sonarr
// fundamental limitation, Story 13-4b AC #1).
type TVExternalIDs struct {
	ID     int64  `json:"id"`
	IMDbID string `json:"imdb_id"`
	TVDbID int64  `json:"tvdb_id"`
}

// GetTVExternalIDs retrieves a TV show's external ids. Language-neutral —
// no language parameter, no fallback chain (the watch-providers class).
func (c *Client) GetTVExternalIDs(ctx context.Context, tvID int) (*TVExternalIDs, error) {
	if tvID <= 0 {
		return nil, NewBadRequestError("TV show ID must be greater than 0")
	}

	endpoint := fmt.Sprintf("/tv/%d/external_ids", tvID)

	var result TVExternalIDs
	if err := c.Get(ctx, endpoint, url.Values{}, &result); err != nil {
		return nil, fmt.Errorf("failed to get TV external ids: %w", err)
	}

	return &result, nil
}

// GetSeasonDetails retrieves the full episode list for a specific season of a TV
// show (GET /tv/{id}/season/{n}). Results are in the client's default language.
func (c *Client) GetSeasonDetails(ctx context.Context, tvID int, seasonNumber int) (*SeasonDetails, error) {
	return c.GetSeasonDetailsWithLanguage(ctx, tvID, seasonNumber, c.language)
}

// GetSeasonDetailsWithLanguage retrieves season details with a specific language.
// This is used by the language fallback chain.
func (c *Client) GetSeasonDetailsWithLanguage(ctx context.Context, tvID int, seasonNumber int, language string) (*SeasonDetails, error) {
	// Validate input
	if tvID <= 0 {
		return nil, NewBadRequestError("TV show ID must be greater than 0")
	}
	if seasonNumber < 0 {
		return nil, NewBadRequestError("season number must be non-negative")
	}

	// Build endpoint path
	endpoint := fmt.Sprintf("/tv/%d/season/%d", tvID, seasonNumber)

	// Build query parameters with language
	queryParams := url.Values{
		"language": []string{language},
	}

	// Make API request
	var result SeasonDetails
	if err := c.Get(ctx, endpoint, queryParams, &result); err != nil {
		return nil, fmt.Errorf("failed to get season details: %w", err)
	}

	return &result, nil
}

// GetTrendingTVShows fetches the trending TV shows for a time window ("day" or "week").
func (c *Client) GetTrendingTVShows(ctx context.Context, timeWindow string, page int) (*SearchResultTVShows, error) {
	return c.GetTrendingTVShowsWithLanguage(ctx, timeWindow, c.language, page)
}

// GetTrendingTVShowsWithLanguage fetches trending TV shows with a specific language,
// used by the language fallback chain.
func (c *Client) GetTrendingTVShowsWithLanguage(ctx context.Context, timeWindow string, language string, page int) (*SearchResultTVShows, error) {
	if page < 1 {
		page = 1
	}

	endpoint := fmt.Sprintf("/trending/tv/%s", normalizeTrendingWindow(timeWindow))
	queryParams := url.Values{
		"page":     []string{strconv.Itoa(page)},
		"language": []string{language},
	}

	var result SearchResultTVShows
	if err := c.Get(ctx, endpoint, queryParams, &result); err != nil {
		return nil, fmt.Errorf("failed to get trending TV shows: %w", err)
	}
	return &result, nil
}

// DiscoverTVShows queries /discover/tv with the given filter params. When
// params.Language is empty, the client's default language is used.
func (c *Client) DiscoverTVShows(ctx context.Context, params DiscoverParams) (*SearchResultTVShows, error) {
	queryParams := discoverQueryParams(params, false /*movie=false → TV*/, c.language)

	var result SearchResultTVShows
	if err := c.Get(ctx, "/discover/tv", queryParams, &result); err != nil {
		return nil, fmt.Errorf("failed to discover TV shows: %w", err)
	}
	return &result, nil
}

// GetTVShowVideos retrieves videos (trailers, teasers, etc.) for a TV show
func (c *Client) GetTVShowVideos(ctx context.Context, tvID int) (*VideosResponse, error) {
	if tvID <= 0 {
		return nil, NewBadRequestError("TV show ID must be greater than 0")
	}

	endpoint := fmt.Sprintf("/tv/%d/videos", tvID)
	queryParams := url.Values{
		"language": []string{c.language},
	}

	var result VideosResponse
	if err := c.Get(ctx, endpoint, queryParams, &result); err != nil {
		return nil, fmt.Errorf("failed to get TV show videos: %w", err)
	}

	return &result, nil
}

// GetTVRecommendations retrieves TMDb's behavior-aggregate recommendations for a
// TV show (GET /tv/{id}/recommendations). Results are in the client's default
// language. Story 12-3 (F-3) — related-content section.
func (c *Client) GetTVRecommendations(ctx context.Context, tvID int) (*SearchResultTVShows, error) {
	return c.GetTVRecommendationsWithLanguage(ctx, tvID, c.language)
}

// GetTVRecommendationsWithLanguage retrieves TV recommendations with a specific
// language. Used by the language fallback chain.
func (c *Client) GetTVRecommendationsWithLanguage(ctx context.Context, tvID int, language string) (*SearchResultTVShows, error) {
	if tvID <= 0 {
		return nil, NewBadRequestError("TV show ID must be greater than 0")
	}

	endpoint := fmt.Sprintf("/tv/%d/recommendations", tvID)
	queryParams := url.Values{
		"language": []string{language},
	}

	var result SearchResultTVShows
	if err := c.Get(ctx, endpoint, queryParams, &result); err != nil {
		return nil, fmt.Errorf("failed to get TV recommendations: %w", err)
	}

	return &result, nil
}

// GetTVSimilar retrieves TMDb's genre/keyword-based similar TV shows
// (GET /tv/{id}/similar). Used as the fallback when /recommendations is empty
// (Story 12-3 AC #4). Results are in the client's default language.
func (c *Client) GetTVSimilar(ctx context.Context, tvID int) (*SearchResultTVShows, error) {
	return c.GetTVSimilarWithLanguage(ctx, tvID, c.language)
}

// GetTVSimilarWithLanguage retrieves similar TV shows with a specific language.
// Used by the language fallback chain.
func (c *Client) GetTVSimilarWithLanguage(ctx context.Context, tvID int, language string) (*SearchResultTVShows, error) {
	if tvID <= 0 {
		return nil, NewBadRequestError("TV show ID must be greater than 0")
	}

	endpoint := fmt.Sprintf("/tv/%d/similar", tvID)
	queryParams := url.Values{
		"language": []string{language},
	}

	var result SearchResultTVShows
	if err := c.Get(ctx, endpoint, queryParams, &result); err != nil {
		return nil, fmt.Errorf("failed to get similar TV shows: %w", err)
	}

	return &result, nil
}
