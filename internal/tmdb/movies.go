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
	var result SearchResultMovies
	if err := c.Get(ctx, "/search/movie", queryParams, &result); err != nil {
		return nil, fmt.Errorf("failed to search movies: %w", err)
	}

	return &result, nil
}

// GetMovieDetails retrieves complete movie information for a specific movie ID
// The results will be in the language specified by the client (e.g., zh-TW)
func (c *Client) GetMovieDetails(ctx context.Context, movieID int) (*MovieDetails, error) {
	// Validate input
	if movieID <= 0 {
		return nil, NewBadRequestError("movie ID must be greater than 0")
	}

	// Build endpoint path
	endpoint := fmt.Sprintf("/movie/%d", movieID)

	// Make API request (no additional query parameters needed)
	var result MovieDetails
	if err := c.Get(ctx, endpoint, nil, &result); err != nil {
		return nil, fmt.Errorf("failed to get movie details: %w", err)
	}

	return &result, nil
}
