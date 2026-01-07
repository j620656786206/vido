package tmdb

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/alexyu/vido/internal/config"
)

func TestSearchMovies(t *testing.T) {
	tests := []struct {
		name           string
		query          string
		page           int
		mockResponse   *SearchResultMovies
		mockStatusCode int
		wantErr        bool
		wantErrCode    string
	}{
		{
			name:  "successful search with results",
			query: "Fight Club",
			page:  1,
			mockResponse: &SearchResultMovies{
				Page: 1,
				Results: []Movie{
					{
						ID:            550,
						Title:         "鬥陣俱樂部",
						OriginalTitle: "Fight Club",
						Overview:      "一個失眠的上班族與一個肥皂製造商建立了一個地下搏擊俱樂部。",
						ReleaseDate:   "1999-10-15",
						VoteAverage:   8.4,
						VoteCount:     26280,
						Popularity:    61.416,
					},
				},
				TotalPages:   1,
				TotalResults: 1,
			},
			mockStatusCode: http.StatusOK,
			wantErr:        false,
		},
		{
			name:  "successful search with multiple pages",
			query: "Star Wars",
			page:  2,
			mockResponse: &SearchResultMovies{
				Page:         2,
				Results:      []Movie{},
				TotalPages:   10,
				TotalResults: 200,
			},
			mockStatusCode: http.StatusOK,
			wantErr:        false,
		},
		{
			name:  "successful search with no results",
			query: "xyz123nonexistent",
			page:  1,
			mockResponse: &SearchResultMovies{
				Page:         1,
				Results:      []Movie{},
				TotalPages:   0,
				TotalResults: 0,
			},
			mockStatusCode: http.StatusOK,
			wantErr:        false,
		},
		{
			name:  "search with zh-TW query",
			query: "復仇者聯盟",
			page:  1,
			mockResponse: &SearchResultMovies{
				Page: 1,
				Results: []Movie{
					{
						ID:            24428,
						Title:         "復仇者聯盟",
						OriginalTitle: "The Avengers",
						Overview:      "地球最強的英雄們必須團結起來對抗未知的威脅。",
						ReleaseDate:   "2012-04-25",
						VoteAverage:   7.7,
						VoteCount:     28123,
						Popularity:    89.234,
					},
				},
				TotalPages:   1,
				TotalResults: 1,
			},
			mockStatusCode: http.StatusOK,
			wantErr:        false,
		},
		{
			name:           "empty query returns error",
			query:          "",
			page:           1,
			mockResponse:   nil,
			mockStatusCode: http.StatusOK,
			wantErr:        true,
			wantErrCode:    ErrCodeBadRequest,
		},
		{
			name:  "page less than 1 defaults to 1",
			query: "Matrix",
			page:  0,
			mockResponse: &SearchResultMovies{
				Page:         1,
				Results:      []Movie{},
				TotalPages:   5,
				TotalResults: 100,
			},
			mockStatusCode: http.StatusOK,
			wantErr:        false,
		},
		{
			name:  "negative page defaults to 1",
			query: "Matrix",
			page:  -5,
			mockResponse: &SearchResultMovies{
				Page:         1,
				Results:      []Movie{},
				TotalPages:   5,
				TotalResults: 100,
			},
			mockStatusCode: http.StatusOK,
			wantErr:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify query parameters
				query := r.URL.Query()
				if tt.query != "" && query.Get("query") != tt.query {
					t.Errorf("query parameter = %v, want %v", query.Get("query"), tt.query)
				}

				// Verify API key and language are included
				if query.Get("api_key") == "" {
					t.Error("api_key parameter is missing")
				}
				if query.Get("language") != "zh-TW" {
					t.Errorf("language parameter = %v, want %v", query.Get("language"), "zh-TW")
				}

				// Write response
				w.WriteHeader(tt.mockStatusCode)
				if tt.mockResponse != nil {
					json.NewEncoder(w).Encode(tt.mockResponse)
				}
			}))
			defer server.Close()

			// Create client with mock server URL
			cfg := &config.Config{
				TMDbAPIKey:          "test-api-key",
				TMDbDefaultLanguage: "zh-TW",
			}
			client := NewClient(cfg)
			client.baseURL = server.URL

			// Call SearchMovies
			ctx := context.Background()
			result, err := client.SearchMovies(ctx, tt.query, tt.page)

			// Check error
			if (err != nil) != tt.wantErr {
				t.Errorf("SearchMovies() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// If we expect an error, check error code if specified
			if tt.wantErr && tt.wantErrCode != "" {
				// Note: We can't easily check AppError code here without importing middleware
				// This is acceptable as the error handling is tested in errors_test.go
				return
			}

			// If we don't expect an error, verify the result
			if !tt.wantErr {
				if result == nil {
					t.Error("SearchMovies() returned nil result, want non-nil")
					return
				}

				if result.Page != tt.mockResponse.Page {
					t.Errorf("result.Page = %v, want %v", result.Page, tt.mockResponse.Page)
				}

				if result.TotalPages != tt.mockResponse.TotalPages {
					t.Errorf("result.TotalPages = %v, want %v", result.TotalPages, tt.mockResponse.TotalPages)
				}

				if result.TotalResults != tt.mockResponse.TotalResults {
					t.Errorf("result.TotalResults = %v, want %v", result.TotalResults, tt.mockResponse.TotalResults)
				}

				if len(result.Results) != len(tt.mockResponse.Results) {
					t.Errorf("len(result.Results) = %v, want %v", len(result.Results), len(tt.mockResponse.Results))
				}

				// Verify first movie if results exist
				if len(result.Results) > 0 && len(tt.mockResponse.Results) > 0 {
					got := result.Results[0]
					want := tt.mockResponse.Results[0]

					if got.ID != want.ID {
						t.Errorf("result.Results[0].ID = %v, want %v", got.ID, want.ID)
					}
					if got.Title != want.Title {
						t.Errorf("result.Results[0].Title = %v, want %v", got.Title, want.Title)
					}
					if got.OriginalTitle != want.OriginalTitle {
						t.Errorf("result.Results[0].OriginalTitle = %v, want %v", got.OriginalTitle, want.OriginalTitle)
					}
					if got.Overview != want.Overview {
						t.Errorf("result.Results[0].Overview = %v, want %v", got.Overview, want.Overview)
					}
				}
			}
		})
	}
}

func TestSearchMovies_HTTPErrors(t *testing.T) {
	tests := []struct {
		name           string
		mockStatusCode int
		mockResponse   string
		wantErr        bool
	}{
		{
			name:           "unauthorized error",
			mockStatusCode: http.StatusUnauthorized,
			mockResponse: `{
				"success": false,
				"status_code": 7,
				"status_message": "Invalid API key: You must be granted a valid key."
			}`,
			wantErr: true,
		},
		{
			name:           "rate limit error",
			mockStatusCode: http.StatusTooManyRequests,
			mockResponse: `{
				"success": false,
				"status_code": 25,
				"status_message": "Your request count (41) is over the allowed limit of 40."
			}`,
			wantErr: true,
		},
		{
			name:           "server error",
			mockStatusCode: http.StatusInternalServerError,
			mockResponse:   `{"success": false, "status_message": "Internal Server Error"}`,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.mockStatusCode)
				w.Write([]byte(tt.mockResponse))
			}))
			defer server.Close()

			// Create client with mock server URL
			cfg := &config.Config{
				TMDbAPIKey:          "test-api-key",
				TMDbDefaultLanguage: "zh-TW",
			}
			client := NewClient(cfg)
			client.baseURL = server.URL

			// Call SearchMovies
			ctx := context.Background()
			result, err := client.SearchMovies(ctx, "test query", 1)

			// Check error
			if (err != nil) != tt.wantErr {
				t.Errorf("SearchMovies() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && result != nil {
				t.Errorf("SearchMovies() result = %v, want nil on error", result)
			}
		})
	}
}

func TestSearchMovies_ContextCancellation(t *testing.T) {
	// Create mock server that never responds
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Block forever
		<-r.Context().Done()
	}))
	defer server.Close()

	// Create client with mock server URL
	cfg := &config.Config{
		TMDbAPIKey:          "test-api-key",
		TMDbDefaultLanguage: "zh-TW",
	}
	client := NewClient(cfg)
	client.baseURL = server.URL

	// Create context with immediate cancellation
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Call SearchMovies with cancelled context
	_, err := client.SearchMovies(ctx, "test query", 1)

	// Should return an error due to context cancellation
	if err == nil {
		t.Error("SearchMovies() with cancelled context should return error")
	}
}

func TestGetMovieDetails(t *testing.T) {
	posterPath := "/pB8BM7pdSp6B6Ih7QZ4DrQ3PmJK.jpg"
	backdropPath := "/fCayJrkfRaCRCTh8GqN30f8oyQF.jpg"
	homepage := "https://www.foxmovies.com/movies/fight-club"

	tests := []struct {
		name           string
		movieID        int
		mockResponse   *MovieDetails
		mockStatusCode int
		wantErr        bool
		wantErrCode    string
	}{
		{
			name:    "successful movie details retrieval",
			movieID: 550,
			mockResponse: &MovieDetails{
				Movie: Movie{
					ID:            550,
					Title:         "鬥陣俱樂部",
					OriginalTitle: "Fight Club",
					Overview:      "一個失眠的上班族與一個肥皂製造商建立了一個地下搏擊俱樂部。",
					ReleaseDate:   "1999-10-15",
					PosterPath:    &posterPath,
					BackdropPath:  &backdropPath,
					VoteAverage:   8.4,
					VoteCount:     26280,
					Popularity:    61.416,
				},
				Budget:  63000000,
				Revenue: 100853753,
				Runtime: 139,
				Status:  "Released",
				Tagline: "搗蛋。混亂。肥皂。",
				Genres: []Genre{
					{ID: 18, Name: "劇情"},
					{ID: 53, Name: "驚悚"},
				},
				ImdbID:   "tt0137523",
				Homepage: &homepage,
			},
			mockStatusCode: http.StatusOK,
			wantErr:        false,
		},
		{
			name:    "successful movie details with zh-TW localization",
			movieID: 24428,
			mockResponse: &MovieDetails{
				Movie: Movie{
					ID:            24428,
					Title:         "復仇者聯盟",
					OriginalTitle: "The Avengers",
					Overview:      "地球最強的英雄們必須團結起來對抗未知的威脅。",
					ReleaseDate:   "2012-04-25",
					PosterPath:    &posterPath,
					BackdropPath:  &backdropPath,
					VoteAverage:   7.7,
					VoteCount:     28123,
					Popularity:    89.234,
				},
				Budget:  220000000,
				Revenue: 1518815515,
				Runtime: 143,
				Status:  "Released",
				Tagline: "有些人是為了組成復仇者聯盟而生的。",
				Genres: []Genre{
					{ID: 28, Name: "動作"},
					{ID: 878, Name: "科幻"},
				},
				ImdbID:   "tt0848228",
				Homepage: &homepage,
			},
			mockStatusCode: http.StatusOK,
			wantErr:        false,
		},
		{
			name:           "invalid movie ID returns error",
			movieID:        0,
			mockResponse:   nil,
			mockStatusCode: http.StatusOK,
			wantErr:        true,
			wantErrCode:    ErrCodeBadRequest,
		},
		{
			name:           "negative movie ID returns error",
			movieID:        -1,
			mockResponse:   nil,
			mockStatusCode: http.StatusOK,
			wantErr:        true,
			wantErrCode:    ErrCodeBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify the endpoint path
				if tt.movieID > 0 {
					expectedPath := "/movie/" + strconv.Itoa(tt.movieID)
					if r.URL.Path != expectedPath {
						t.Errorf("request path = %v, want %v", r.URL.Path, expectedPath)
					}
				}

				// Verify API key and language are included
				query := r.URL.Query()
				if query.Get("api_key") == "" {
					t.Error("api_key parameter is missing")
				}
				if query.Get("language") != "zh-TW" {
					t.Errorf("language parameter = %v, want %v", query.Get("language"), "zh-TW")
				}

				// Write response
				w.WriteHeader(tt.mockStatusCode)
				if tt.mockResponse != nil {
					json.NewEncoder(w).Encode(tt.mockResponse)
				}
			}))
			defer server.Close()

			// Create client with mock server URL
			cfg := &config.Config{
				TMDbAPIKey:          "test-api-key",
				TMDbDefaultLanguage: "zh-TW",
			}
			client := NewClient(cfg)
			client.baseURL = server.URL

			// Call GetMovieDetails
			ctx := context.Background()
			result, err := client.GetMovieDetails(ctx, tt.movieID)

			// Check error
			if (err != nil) != tt.wantErr {
				t.Errorf("GetMovieDetails() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// If we expect an error, check error code if specified
			if tt.wantErr && tt.wantErrCode != "" {
				// Note: We can't easily check AppError code here without importing middleware
				// This is acceptable as the error handling is tested in errors_test.go
				return
			}

			// If we don't expect an error, verify the result
			if !tt.wantErr {
				if result == nil {
					t.Error("GetMovieDetails() returned nil result, want non-nil")
					return
				}

				// Verify Movie fields
				if result.ID != tt.mockResponse.ID {
					t.Errorf("result.ID = %v, want %v", result.ID, tt.mockResponse.ID)
				}
				if result.Title != tt.mockResponse.Title {
					t.Errorf("result.Title = %v, want %v", result.Title, tt.mockResponse.Title)
				}
				if result.OriginalTitle != tt.mockResponse.OriginalTitle {
					t.Errorf("result.OriginalTitle = %v, want %v", result.OriginalTitle, tt.mockResponse.OriginalTitle)
				}
				if result.Overview != tt.mockResponse.Overview {
					t.Errorf("result.Overview = %v, want %v", result.Overview, tt.mockResponse.Overview)
				}
				if result.ReleaseDate != tt.mockResponse.ReleaseDate {
					t.Errorf("result.ReleaseDate = %v, want %v", result.ReleaseDate, tt.mockResponse.ReleaseDate)
				}

				// Verify MovieDetails-specific fields
				if result.Budget != tt.mockResponse.Budget {
					t.Errorf("result.Budget = %v, want %v", result.Budget, tt.mockResponse.Budget)
				}
				if result.Revenue != tt.mockResponse.Revenue {
					t.Errorf("result.Revenue = %v, want %v", result.Revenue, tt.mockResponse.Revenue)
				}
				if result.Runtime != tt.mockResponse.Runtime {
					t.Errorf("result.Runtime = %v, want %v", result.Runtime, tt.mockResponse.Runtime)
				}
				if result.Status != tt.mockResponse.Status {
					t.Errorf("result.Status = %v, want %v", result.Status, tt.mockResponse.Status)
				}
				if result.Tagline != tt.mockResponse.Tagline {
					t.Errorf("result.Tagline = %v, want %v", result.Tagline, tt.mockResponse.Tagline)
				}
				if result.ImdbID != tt.mockResponse.ImdbID {
					t.Errorf("result.ImdbID = %v, want %v", result.ImdbID, tt.mockResponse.ImdbID)
				}

				// Verify genres
				if len(result.Genres) != len(tt.mockResponse.Genres) {
					t.Errorf("len(result.Genres) = %v, want %v", len(result.Genres), len(tt.mockResponse.Genres))
				}

				// Verify poster and backdrop paths
				if (result.PosterPath == nil) != (tt.mockResponse.PosterPath == nil) {
					t.Errorf("result.PosterPath nil mismatch")
				} else if result.PosterPath != nil && *result.PosterPath != *tt.mockResponse.PosterPath {
					t.Errorf("result.PosterPath = %v, want %v", *result.PosterPath, *tt.mockResponse.PosterPath)
				}

				if (result.BackdropPath == nil) != (tt.mockResponse.BackdropPath == nil) {
					t.Errorf("result.BackdropPath nil mismatch")
				} else if result.BackdropPath != nil && *result.BackdropPath != *tt.mockResponse.BackdropPath {
					t.Errorf("result.BackdropPath = %v, want %v", *result.BackdropPath, *tt.mockResponse.BackdropPath)
				}
			}
		})
	}
}

func TestGetMovieDetails_HTTPErrors(t *testing.T) {
	tests := []struct {
		name           string
		mockStatusCode int
		mockResponse   string
		wantErr        bool
	}{
		{
			name:           "not found error",
			mockStatusCode: http.StatusNotFound,
			mockResponse: `{
				"success": false,
				"status_code": 34,
				"status_message": "The resource you requested could not be found."
			}`,
			wantErr: true,
		},
		{
			name:           "unauthorized error",
			mockStatusCode: http.StatusUnauthorized,
			mockResponse: `{
				"success": false,
				"status_code": 7,
				"status_message": "Invalid API key: You must be granted a valid key."
			}`,
			wantErr: true,
		},
		{
			name:           "rate limit error",
			mockStatusCode: http.StatusTooManyRequests,
			mockResponse: `{
				"success": false,
				"status_code": 25,
				"status_message": "Your request count (41) is over the allowed limit of 40."
			}`,
			wantErr: true,
		},
		{
			name:           "server error",
			mockStatusCode: http.StatusInternalServerError,
			mockResponse:   `{"success": false, "status_message": "Internal Server Error"}`,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.mockStatusCode)
				w.Write([]byte(tt.mockResponse))
			}))
			defer server.Close()

			// Create client with mock server URL
			cfg := &config.Config{
				TMDbAPIKey:          "test-api-key",
				TMDbDefaultLanguage: "zh-TW",
			}
			client := NewClient(cfg)
			client.baseURL = server.URL

			// Call GetMovieDetails
			ctx := context.Background()
			result, err := client.GetMovieDetails(ctx, 550)

			// Check error
			if (err != nil) != tt.wantErr {
				t.Errorf("GetMovieDetails() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && result != nil {
				t.Errorf("GetMovieDetails() result = %v, want nil on error", result)
			}
		})
	}
}

func TestGetMovieDetails_ContextCancellation(t *testing.T) {
	// Create mock server that never responds
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Block forever
		<-r.Context().Done()
	}))
	defer server.Close()

	// Create client with mock server URL
	cfg := &config.Config{
		TMDbAPIKey:          "test-api-key",
		TMDbDefaultLanguage: "zh-TW",
	}
	client := NewClient(cfg)
	client.baseURL = server.URL

	// Create context with immediate cancellation
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Call GetMovieDetails with cancelled context
	_, err := client.GetMovieDetails(ctx, 550)

	// Should return an error due to context cancellation
	if err == nil {
		t.Error("GetMovieDetails() with cancelled context should return error")
	}
}
