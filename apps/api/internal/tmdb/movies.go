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
