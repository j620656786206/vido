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

func TestClient_SearchTVShows(t *testing.T) {
	tests := []struct {
		name         string
		query        string
		page         int
		mockResponse SearchResultTVShows
		wantErr      bool
		wantErrCode  string
	}{
		{
			name:  "successful search",
			query: "Breaking Bad",
			page:  1,
			mockResponse: SearchResultTVShows{
				Page: 1,
				Results: []TVShow{
					{
						ID:   1396,
						Name: "Breaking Bad",
					},
				},
				TotalPages:   1,
				TotalResults: 1,
			},
		},
		{
			name:  "negative page defaults to 1",
			query: "test",
			page:  -1,
			mockResponse: SearchResultTVShows{
				Page:         1,
				Results:      []TVShow{},
				TotalPages:   0,
				TotalResults: 0,
			},
		},
		{
			name:        "empty query returns error",
			query:       "",
			page:        1,
			wantErr:     true,
			wantErrCode: ErrCodeBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantErr && tt.wantErrCode == ErrCodeBadRequest {
				// Test validation error without server
				client := NewClient(ClientConfig{
					APIKey: "test-key",
				})

				result, err := client.SearchTVShows(context.Background(), tt.query, tt.page)

				require.Error(t, err)
				assert.Nil(t, result)
				return
			}

			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request
				assert.Equal(t, "/search/tv", r.URL.Path)
				assert.NotEmpty(t, r.URL.Query().Get("query"))
				assert.NotEmpty(t, r.URL.Query().Get("language"))
				assert.NotEmpty(t, r.URL.Query().Get("api_key"))

				// Return mock response
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(tt.mockResponse)
			}))
			defer server.Close()

			client := NewClient(ClientConfig{
				APIKey:  "test-key",
				BaseURL: server.URL,
			})

			result, err := client.SearchTVShows(context.Background(), tt.query, tt.page)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.mockResponse.Page, result.Page)
			assert.Equal(t, tt.mockResponse.TotalResults, result.TotalResults)
			assert.Len(t, result.Results, len(tt.mockResponse.Results))
		})
	}
}

func TestClient_SearchTVShowsWithLanguage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify language parameter
		lang := r.URL.Query().Get("language")
		assert.Equal(t, "zh-CN", lang)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(SearchResultTVShows{
			Page:    1,
			Results: []TVShow{},
		})
	}))
	defer server.Close()

	client := NewClient(ClientConfig{
		APIKey:   "test-key",
		BaseURL:  server.URL,
		Language: "zh-TW",
	})

	result, err := client.SearchTVShowsWithLanguage(context.Background(), "test", "zh-CN", 1)

	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestClient_GetTVShowDetails(t *testing.T) {
	tests := []struct {
		name         string
		tvID         int
		mockResponse TVShowDetails
		wantErr      bool
		wantErrCode  string
	}{
		{
			name: "successful get details",
			tvID: 1396,
			mockResponse: TVShowDetails{
				TVShow: TVShow{
					ID:   1396,
					Name: "Breaking Bad",
				},
				NumberOfSeasons:  5,
				NumberOfEpisodes: 62,
			},
		},
		{
			name:        "invalid TV ID",
			tvID:        0,
			wantErr:     true,
			wantErrCode: ErrCodeBadRequest,
		},
		{
			name:        "negative TV ID",
			tvID:        -1,
			wantErr:     true,
			wantErrCode: ErrCodeBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantErr {
				// Test validation error without server
				client := NewClient(ClientConfig{
					APIKey: "test-key",
				})

				result, err := client.GetTVShowDetails(context.Background(), tt.tvID)

				require.Error(t, err)
				assert.Nil(t, result)
				return
			}

			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request path
				assert.Contains(t, r.URL.Path, "/tv/")

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(tt.mockResponse)
			}))
			defer server.Close()

			client := NewClient(ClientConfig{
				APIKey:  "test-key",
				BaseURL: server.URL,
			})

			result, err := client.GetTVShowDetails(context.Background(), tt.tvID)

			require.NoError(t, err)
			assert.Equal(t, tt.mockResponse.ID, result.ID)
			assert.Equal(t, tt.mockResponse.Name, result.Name)
			assert.Equal(t, tt.mockResponse.NumberOfSeasons, result.NumberOfSeasons)
		})
	}
}

func TestClient_GetTVShowDetailsWithLanguage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify language parameter
		lang := r.URL.Query().Get("language")
		assert.Equal(t, "en", lang)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(TVShowDetails{
			TVShow: TVShow{
				ID:   1396,
				Name: "Breaking Bad",
			},
		})
	}))
	defer server.Close()

	client := NewClient(ClientConfig{
		APIKey:   "test-key",
		BaseURL:  server.URL,
		Language: "zh-TW",
	})

	result, err := client.GetTVShowDetailsWithLanguage(context.Background(), 1396, "en")

	require.NoError(t, err)
	assert.NotNil(t, result)
}
