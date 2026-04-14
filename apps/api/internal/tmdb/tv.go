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
