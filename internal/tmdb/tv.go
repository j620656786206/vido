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
	// Validate input
	if query == "" {
		return nil, NewBadRequestError("search query cannot be empty")
	}

	if page < 1 {
		page = 1
	}

	// Build query parameters
	queryParams := url.Values{
		"query": []string{query},
		"page":  []string{strconv.Itoa(page)},
	}

	// Make API request
	var result SearchResultTVShows
	if err := c.Get(ctx, "/search/tv", queryParams, &result); err != nil {
		return nil, fmt.Errorf("failed to search TV shows: %w", err)
	}

	return &result, nil
}
