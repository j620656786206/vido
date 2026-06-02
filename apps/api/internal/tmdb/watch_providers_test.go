package tmdb

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_GetWatchProviders(t *testing.T) {
	mock := WatchProvidersResponse{
		ID: 550,
		Results: map[string]WatchProviderRegion{
			"TW": {
				Link:     "https://www.themoviedb.org/movie/550/watch?locale=TW",
				Flatrate: []WatchProvider{{ProviderID: 8, ProviderName: "Netflix"}},
			},
			"US": {
				Link:     "https://www.themoviedb.org/movie/550/watch?locale=US",
				Flatrate: []WatchProvider{{ProviderID: 337, ProviderName: "Disney Plus"}},
			},
		},
	}

	t.Run("movie path + region filter keeps only requested region", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/movie/550/watch/providers", r.URL.Path)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(mock)
		}))
		defer server.Close()

		client := NewClient(ClientConfig{APIKey: "k", BaseURL: server.URL})
		res, err := client.GetWatchProviders(context.Background(), "movie", 550, "TW")
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Equal(t, 550, res.ID)
		assert.Len(t, res.Results, 1, "only the requested region must remain")
		require.Contains(t, res.Results, "TW")
		require.Len(t, res.Results["TW"].Flatrate, 1)
		assert.Equal(t, 8, res.Results["TW"].Flatrate[0].ProviderID)
	})

	t.Run("tv path", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/tv/1396/watch/providers", r.URL.Path)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(mock)
		}))
		defer server.Close()

		client := NewClient(ClientConfig{APIKey: "k", BaseURL: server.URL})
		_, err := client.GetWatchProviders(context.Background(), "tv", 1396, "")
		require.NoError(t, err)
	})

	t.Run("empty region returns all regions", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(mock)
		}))
		defer server.Close()

		client := NewClient(ClientConfig{APIKey: "k", BaseURL: server.URL})
		res, err := client.GetWatchProviders(context.Background(), "movie", 550, "")
		require.NoError(t, err)
		assert.Len(t, res.Results, 2)
	})

	t.Run("region with no providers yields empty results map", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(mock)
		}))
		defer server.Close()

		client := NewClient(ClientConfig{APIKey: "k", BaseURL: server.URL})
		res, err := client.GetWatchProviders(context.Background(), "movie", 550, "JP")
		require.NoError(t, err)
		assert.Empty(t, res.Results, "unknown region filters down to nothing")
	})

	t.Run("invalid media type rejected", func(t *testing.T) {
		client := NewClient(ClientConfig{APIKey: "k"})
		_, err := client.GetWatchProviders(context.Background(), "person", 1, "TW")
		require.Error(t, err)
	})

	t.Run("non-positive id rejected", func(t *testing.T) {
		client := NewClient(ClientConfig{APIKey: "k"})
		_, err := client.GetWatchProviders(context.Background(), "movie", 0, "TW")
		require.Error(t, err)
	})
}

func TestTWWatchProviderIDs(t *testing.T) {
	// Story-named, confident IDs (AC #2, Task 2.3).
	assert.Equal(t, 8, TWWatchProviderIDs["netflix"])
	assert.Equal(t, 337, TWWatchProviderIDs["disney"])
	// Every shortcut must map to a positive provider ID.
	for name, id := range TWWatchProviderIDs {
		assert.Greater(t, id, 0, "provider %q must have a positive TMDb ID", name)
	}
}
