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

func TestClient_SearchMovies(t *testing.T) {
	tests := []struct {
		name         string
		query        string
		page         int
		mockResponse SearchResultMovies
		wantErr      bool
		wantErrCode  string
	}{
		{
			name:  "successful search",
			query: "鬼滅之刃",
			page:  1,
			mockResponse: SearchResultMovies{
				Page: 1,
				Results: []Movie{
					{
						ID:    635302,
						Title: "鬼滅之刃劇場版 無限列車篇",
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
			mockResponse: SearchResultMovies{
				Page:         1,
				Results:      []Movie{},
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

				result, err := client.SearchMovies(context.Background(), tt.query, tt.page)

				require.Error(t, err)
				assert.Nil(t, result)
				return
			}

			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request
				assert.Equal(t, "/search/movie", r.URL.Path)
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

			result, err := client.SearchMovies(context.Background(), tt.query, tt.page)

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

func TestClient_SearchMoviesWithLanguage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify language parameter
		lang := r.URL.Query().Get("language")
		assert.Equal(t, "zh-CN", lang)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(SearchResultMovies{
			Page:    1,
			Results: []Movie{},
		})
	}))
	defer server.Close()

	client := NewClient(ClientConfig{
		APIKey:   "test-key",
		BaseURL:  server.URL,
		Language: "zh-TW", // Default language
	})

	// Call with different language
	result, err := client.SearchMoviesWithLanguage(context.Background(), "test", "zh-CN", 1)

	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestClient_GetMovieDetails(t *testing.T) {
	tests := []struct {
		name         string
		movieID      int
		mockResponse MovieDetails
		wantErr      bool
		wantErrCode  string
	}{
		{
			name:    "successful get details",
			movieID: 550,
			mockResponse: MovieDetails{
				Movie: Movie{
					ID:    550,
					Title: "Fight Club",
				},
				Runtime: 139,
				Budget:  63000000,
			},
		},
		{
			name:        "invalid movie ID",
			movieID:     0,
			wantErr:     true,
			wantErrCode: ErrCodeBadRequest,
		},
		{
			name:        "negative movie ID",
			movieID:     -1,
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

				result, err := client.GetMovieDetails(context.Background(), tt.movieID)

				require.Error(t, err)
				assert.Nil(t, result)
				return
			}

			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request path
				assert.Contains(t, r.URL.Path, "/movie/")

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(tt.mockResponse)
			}))
			defer server.Close()

			client := NewClient(ClientConfig{
				APIKey:  "test-key",
				BaseURL: server.URL,
			})

			result, err := client.GetMovieDetails(context.Background(), tt.movieID)

			require.NoError(t, err)
			assert.Equal(t, tt.mockResponse.ID, result.ID)
			assert.Equal(t, tt.mockResponse.Title, result.Title)
			assert.Equal(t, tt.mockResponse.Runtime, result.Runtime)
		})
	}
}

func TestClient_GetMovieDetailsWithLanguage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify language parameter
		lang := r.URL.Query().Get("language")
		assert.Equal(t, "en", lang)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(MovieDetails{
			Movie: Movie{
				ID:    550,
				Title: "Fight Club",
			},
		})
	}))
	defer server.Close()

	client := NewClient(ClientConfig{
		APIKey:   "test-key",
		BaseURL:  server.URL,
		Language: "zh-TW",
	})

	result, err := client.GetMovieDetailsWithLanguage(context.Background(), 550, "en")

	require.NoError(t, err)
	assert.NotNil(t, result)
}
