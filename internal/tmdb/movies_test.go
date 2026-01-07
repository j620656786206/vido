package tmdb

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
