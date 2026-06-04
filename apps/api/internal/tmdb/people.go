package tmdb

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// SearchPeople searches for people by name and returns paginated results.
// The results use the language configured on the client (e.g. zh-TW).
//
// NOTE: SearchPeople is intentionally NOT part of ClientInterface. Adding it
// there would force every existing ClientInterface mock (tmdb.MockClient in
// fallback_test.go, etc.) to grow a method. Consumers that need person search
// (services.SearchService) depend on a narrow interface satisfied by *Client.
func (c *Client) SearchPeople(ctx context.Context, query string, page int) (*SearchResultPeople, error) {
	return c.SearchPeopleWithLanguage(ctx, query, c.language, page)
}

// SearchPeopleWithLanguage searches for people with a specific language.
func (c *Client) SearchPeopleWithLanguage(ctx context.Context, query string, language string, page int) (*SearchResultPeople, error) {
	if query == "" {
		return nil, NewBadRequestError("search query cannot be empty")
	}

	if page < 1 {
		page = 1
	}

	queryParams := url.Values{
		"query":    []string{query},
		"page":     []string{strconv.Itoa(page)},
		"language": []string{language},
	}

	var result SearchResultPeople
	if err := c.Get(ctx, "/search/person", queryParams, &result); err != nil {
		return nil, fmt.Errorf("failed to search people: %w", err)
	}

	return &result, nil
}
