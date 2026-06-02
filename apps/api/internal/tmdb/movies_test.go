package tmdb

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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

// --- Story 10-1 additions ---

func TestClient_GetTrendingMovies(t *testing.T) {
	tests := []struct {
		name         string
		timeWindow   string
		page         int
		wantPath     string
		wantLanguage string
	}{
		{name: "week window", timeWindow: "week", page: 1, wantPath: "/trending/movie/week", wantLanguage: "zh-TW"},
		{name: "day window", timeWindow: "day", page: 2, wantPath: "/trending/movie/day", wantLanguage: "zh-TW"},
		{name: "unknown window defaults to week", timeWindow: "invalid", page: 1, wantPath: "/trending/movie/week", wantLanguage: "zh-TW"},
		{name: "negative page normalized to 1", timeWindow: "week", page: -5, wantPath: "/trending/movie/week", wantLanguage: "zh-TW"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, tt.wantPath, r.URL.Path)
				assert.Equal(t, tt.wantLanguage, r.URL.Query().Get("language"))
				assert.NotEmpty(t, r.URL.Query().Get("page"))
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(SearchResultMovies{Page: 1, Results: []Movie{{ID: 1, Title: "Trending"}}})
			}))
			defer server.Close()

			client := NewClient(ClientConfig{APIKey: "k", BaseURL: server.URL})
			result, err := client.GetTrendingMovies(context.Background(), tt.timeWindow, tt.page)

			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Len(t, result.Results, 1)
		})
	}
}

func TestClient_GetTrendingMoviesWithLanguage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/trending/movie/week", r.URL.Path)
		assert.Equal(t, "en", r.URL.Query().Get("language"))
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(SearchResultMovies{Page: 1})
	}))
	defer server.Close()

	client := NewClient(ClientConfig{APIKey: "k", BaseURL: server.URL, Language: "zh-TW"})
	_, err := client.GetTrendingMoviesWithLanguage(context.Background(), "week", "en", 1)
	require.NoError(t, err)
}

// TestClient_TrendingDiscover_ContextCancellation verifies that the new
// trending/discover endpoints respect ctx.Done(). A slow upstream + cancelled
// context must surface context.Canceled (or DeadlineExceeded) instead of
// hanging until the server eventually responds.
func TestClient_TrendingDiscover_ContextCancellation(t *testing.T) {
	cases := []struct {
		name string
		call func(ctx context.Context, c *Client) error
	}{
		{
			name: "GetTrendingMovies",
			call: func(ctx context.Context, c *Client) error {
				_, err := c.GetTrendingMovies(ctx, "week", 1)
				return err
			},
		},
		{
			name: "DiscoverMovies",
			call: func(ctx context.Context, c *Client) error {
				_, err := c.DiscoverMovies(ctx, DiscoverParams{GenreIDs: []int{28}})
				return err
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Server holds the response until the client's context cancels it.
			released := make(chan struct{})
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				select {
				case <-r.Context().Done():
				case <-released:
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(SearchResultMovies{Page: 1})
			}))
			defer func() {
				close(released)
				server.Close()
			}()

			client := NewClient(ClientConfig{APIKey: "k", BaseURL: server.URL, Language: "zh-TW"})

			ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
			defer cancel()

			start := time.Now()
			err := tc.call(ctx, client)
			elapsed := time.Since(start)

			require.Error(t, err, "cancelled context must surface as error")
			assert.True(t, elapsed < 2*time.Second, "must not block indefinitely after cancel; took %s", elapsed)

			isCancel := errors.Is(err, context.Canceled) ||
				errors.Is(err, context.DeadlineExceeded)
			assert.True(t, isCancel, "error must wrap context.Canceled / DeadlineExceeded; got %v", err)
		})
	}
}

func TestClient_DiscoverMovies(t *testing.T) {
	tests := []struct {
		name   string
		params DiscoverParams
		check  func(t *testing.T, q map[string][]string)
	}{
		{
			name: "with genre + date range + region + sort",
			params: DiscoverParams{
				GenreIDs: []int{28, 12}, YearGte: 2024, YearLte: 2026,
				Region: "TW", SortBy: "popularity.desc", Page: 3,
			},
			check: func(t *testing.T, q map[string][]string) {
				assert.Equal(t, "28,12", q["with_genres"][0])
				assert.Equal(t, "TW", q["region"][0])
				assert.Equal(t, "popularity.desc", q["sort_by"][0])
				assert.Equal(t, "2024-01-01", q["primary_release_date.gte"][0])
				assert.Equal(t, "2026-12-31", q["primary_release_date.lte"][0])
				assert.Equal(t, "3", q["page"][0])
			},
		},
		{
			name:   "empty params omits optional fields but keeps page+language",
			params: DiscoverParams{},
			check: func(t *testing.T, q map[string][]string) {
				assert.Equal(t, "1", q["page"][0])
				assert.NotEmpty(t, q["language"][0])
				_, hasGenre := q["with_genres"]
				_, hasRegion := q["region"]
				_, hasSort := q["sort_by"]
				_, hasDateGte := q["primary_release_date.gte"]
				_, hasVoteGte := q["vote_average.gte"]
				_, hasWatch := q["with_watch_providers"]
				assert.False(t, hasGenre)
				assert.False(t, hasRegion)
				assert.False(t, hasSort)
				assert.False(t, hasDateGte)
				assert.False(t, hasVoteGte)
				assert.False(t, hasWatch)
			},
		},
		{
			name:   "per-call language overrides client default",
			params: DiscoverParams{Language: "ja"},
			check: func(t *testing.T, q map[string][]string) {
				assert.Equal(t, "ja", q["language"][0])
			},
		},
		{
			// AC #1: multiple filters combine (genre + year + region + rating).
			name: "vote average range maps to vote_average.gte/lte",
			params: DiscoverParams{
				GenreIDs: []int{28}, YearGte: 2024,
				VoteAverageGte: 7, VoteAverageLte: 9.5,
			},
			check: func(t *testing.T, q map[string][]string) {
				assert.Equal(t, "28", q["with_genres"][0])
				assert.Equal(t, "2024-01-01", q["primary_release_date.gte"][0])
				assert.Equal(t, "7", q["vote_average.gte"][0]) // whole number: no trailing .0
				assert.Equal(t, "9.5", q["vote_average.lte"][0])
			},
		},
		{
			// AC #2: platform filter via TMDb Watch Providers for a region.
			name: "watch providers map to with_watch_providers + watch_region",
			params: DiscoverParams{
				WatchProviders: []int{8, 337}, WatchRegion: "TW",
			},
			check: func(t *testing.T, q map[string][]string) {
				assert.Equal(t, "8|337", q["with_watch_providers"][0]) // pipe = OR
				assert.Equal(t, "TW", q["watch_region"][0])
			},
		},
		{
			// AC #2: watch_region defaults to region, then "TW", when providers set.
			name: "watch_region defaults to region when WatchRegion blank",
			params: DiscoverParams{
				WatchProviders: []int{8}, Region: "US",
			},
			check: func(t *testing.T, q map[string][]string) {
				assert.Equal(t, "8", q["with_watch_providers"][0])
				assert.Equal(t, "US", q["watch_region"][0])
			},
		},
		{
			name: "watch_region defaults to TW when neither WatchRegion nor Region set",
			params: DiscoverParams{
				WatchProviders: []int{8},
			},
			check: func(t *testing.T, q map[string][]string) {
				assert.Equal(t, "TW", q["watch_region"][0])
			},
		},
		{
			// AC #3 / Task 3.3: date_added is a local-library sort, never sent to TMDb.
			name:   "local date_added sort is NOT forwarded to TMDb",
			params: DiscoverParams{SortBy: SortByDateAdded},
			check: func(t *testing.T, q map[string][]string) {
				_, hasSort := q["sort_by"]
				assert.False(t, hasSort, "date_added is a local sort; must not reach TMDb")
			},
		},
		{
			// AC #3 / Task 3.1+3.2: TMDb-native sort keys still pass through.
			name:   "native sort key (vote_average.desc) is forwarded",
			params: DiscoverParams{SortBy: "vote_average.desc"},
			check: func(t *testing.T, q map[string][]string) {
				assert.Equal(t, "vote_average.desc", q["sort_by"][0])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/discover/movie", r.URL.Path)
				tt.check(t, r.URL.Query())
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(SearchResultMovies{Page: 1})
			}))
			defer server.Close()

			client := NewClient(ClientConfig{APIKey: "k", BaseURL: server.URL, Language: "zh-TW"})
			_, err := client.DiscoverMovies(context.Background(), tt.params)
			require.NoError(t, err)
		})
	}
}

func TestParseIntCSV(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []int
	}{
		{name: "empty string", in: "", want: nil},
		{name: "single id", in: "28", want: []int{28}},
		{name: "multiple ids", in: "28,12,18", want: []int{28, 12, 18}},
		{name: "trims whitespace", in: " 28 , 12 ", want: []int{28, 12}},
		{name: "skips blank tokens", in: "28,,12,", want: []int{28, 12}},
		{name: "skips non-numeric tokens", in: "28,abc,12", want: []int{28, 12}},
		{name: "all-invalid yields nil", in: "abc,def", want: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ParseIntCSV(tt.in))
		})
	}
}

func TestFormatVote(t *testing.T) {
	assert.Equal(t, "7", formatVote(7.0), "whole numbers drop the trailing .0")
	assert.Equal(t, "7.5", formatVote(7.5))
	assert.Equal(t, "9.25", formatVote(9.25))
}

func TestJoinInts(t *testing.T) {
	assert.Equal(t, "", joinInts(nil, ","))
	assert.Equal(t, "28", joinInts([]int{28}, ","))
	assert.Equal(t, "28,12", joinInts([]int{28, 12}, ","))
	assert.Equal(t, "8|337", joinInts([]int{8, 337}, "|"))
}

// TestClient_DiscoverMovies_AllFiltersCombined is the AC #1 integration check:
// when EVERY filter dimension is supplied at once, the outgoing TMDb query must
// carry ALL of them simultaneously (combined AND semantics). Existing subtests
// exercise dimensions in subsets; this asserts the full cross-product in a
// single request so no dimension silently drops when others are present. [P1]
func TestClient_DiscoverMovies_AllFiltersCombined(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/discover/movie", r.URL.Path)
		q := r.URL.Query()
		assert.Equal(t, "28,18", q.Get("with_genres"))
		assert.Equal(t, "2020-01-01", q.Get("primary_release_date.gte"))
		assert.Equal(t, "2025-12-31", q.Get("primary_release_date.lte"))
		assert.Equal(t, "TW", q.Get("region"))
		assert.Equal(t, "7", q.Get("vote_average.gte"))
		assert.Equal(t, "9.5", q.Get("vote_average.lte"))
		assert.Equal(t, "8|337", q.Get("with_watch_providers"))
		assert.Equal(t, "TW", q.Get("watch_region"))
		assert.Equal(t, "popularity.desc", q.Get("sort_by"))
		assert.Equal(t, "2", q.Get("page"))
		assert.Equal(t, "zh-TW", q.Get("language"))
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(SearchResultMovies{Page: 2})
	}))
	defer server.Close()

	client := NewClient(ClientConfig{APIKey: "k", BaseURL: server.URL, Language: "zh-TW"})
	_, err := client.DiscoverMovies(context.Background(), DiscoverParams{
		GenreIDs: []int{28, 18}, YearGte: 2020, YearLte: 2025, Region: "TW",
		VoteAverageGte: 7, VoteAverageLte: 9.5,
		WatchProviders: []int{8, 337}, WatchRegion: "TW",
		SortBy: "popularity.desc", Page: 2,
	})
	require.NoError(t, err)
}
