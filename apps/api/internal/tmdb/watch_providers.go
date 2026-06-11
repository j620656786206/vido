package tmdb

import (
	"context"
	"fmt"
	"net/url"
)

// WatchProvider is a single streaming/rental/purchase provider for a title in a
// given region, as returned by TMDb's /{media_type}/{id}/watch/providers endpoint.
type WatchProvider struct {
	ProviderID      int     `json:"provider_id" example:"8"`
	ProviderName    string  `json:"provider_name" example:"Netflix"`
	LogoPath        *string `json:"logo_path" example:"/t2yyOv40HZeVlLjYsCsPHnWLk4W.jpg"`
	DisplayPriority int     `json:"display_priority" example:"0"`
}

// WatchProviderRegion groups the providers available for a title within one
// region, split by monetization type (flatrate = subscription, rent, buy).
type WatchProviderRegion struct {
	Link     string          `json:"link" example:"https://www.themoviedb.org/movie/550/watch?locale=TW"`
	Flatrate []WatchProvider `json:"flatrate,omitempty"`
	Rent     []WatchProvider `json:"rent,omitempty"`
	Buy      []WatchProvider `json:"buy,omitempty"`
}

// WatchProvidersResponse is the response from /{media_type}/{id}/watch/providers.
// Results is keyed by ISO 3166-1 region code (e.g. "TW", "US").
type WatchProvidersResponse struct {
	ID      int                            `json:"id" example:"550"`
	Results map[string]WatchProviderRegion `json:"results"`
}

// GetWatchProviders returns the watch/providers info for a movie or TV show.
// mediaType must be "movie" or "tv". When region is non-empty, the response's
// Results map is filtered down to that single region (TMDb otherwise returns
// every region). (Story 12-4 AC #2; restored from commit fdbf249^ — the
// re-introduction is ADR-sanctioned now that F-4 is a real consumer.)
func (c *Client) GetWatchProviders(ctx context.Context, mediaType string, id int, region string) (*WatchProvidersResponse, error) {
	if mediaType != "movie" && mediaType != "tv" {
		return nil, NewBadRequestError("media type must be 'movie' or 'tv'")
	}
	if id <= 0 {
		return nil, NewBadRequestError("id must be greater than 0")
	}

	endpoint := fmt.Sprintf("/%s/%d/watch/providers", mediaType, id)

	var result WatchProvidersResponse
	if err := c.Get(ctx, endpoint, url.Values{}, &result); err != nil {
		return nil, fmt.Errorf("failed to get watch providers: %w", err)
	}

	if region != "" && result.Results != nil {
		filtered := make(map[string]WatchProviderRegion, 1)
		if r, ok := result.Results[region]; ok {
			filtered[region] = r
		}
		result.Results = filtered
	}

	return &result, nil
}
