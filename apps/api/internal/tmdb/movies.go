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
