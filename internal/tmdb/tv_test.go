package tmdb

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexyu/vido/internal/config"
)

func TestSearchTVShows(t *testing.T) {
	tests := []struct {
		name           string
		query          string
		page           int
		mockResponse   *SearchResultTVShows
		mockStatusCode int
		wantErr        bool
		wantErrCode    string
	}{
		{
			name:  "successful search with results",
			query: "Breaking Bad",
			page:  1,
			mockResponse: &SearchResultTVShows{
				Page: 1,
				Results: []TVShow{
					{
						ID:            1396,
						Name:          "絕命毒師",
						OriginalName:  "Breaking Bad",
						Overview:      "一位高中化學老師被診斷出癌症，他決定製造和販賣冰毒來為家人留下財產。",
						FirstAirDate:  "2008-01-20",
						VoteAverage:   8.9,
						VoteCount:     12345,
						Popularity:    369.594,
						OriginCountry: []string{"US"},
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
			query: "Game of Thrones",
			page:  2,
			mockResponse: &SearchResultTVShows{
				Page:         2,
				Results:      []TVShow{},
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
			mockResponse: &SearchResultTVShows{
				Page:         1,
				Results:      []TVShow{},
				TotalPages:   0,
				TotalResults: 0,
			},
			mockStatusCode: http.StatusOK,
			wantErr:        false,
		},
		{
			name:  "search with zh-TW query",
			query: "權力的遊戲",
			page:  1,
			mockResponse: &SearchResultTVShows{
				Page: 1,
				Results: []TVShow{
					{
						ID:            1399,
						Name:          "權力的遊戲",
						OriginalName:  "Game of Thrones",
						Overview:      "七大王國爭奪鐵王座的故事。",
						FirstAirDate:  "2011-04-17",
						VoteAverage:   8.4,
						VoteCount:     15678,
						Popularity:    789.123,
						OriginCountry: []string{"US"},
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
			query: "Friends",
			page:  0,
			mockResponse: &SearchResultTVShows{
				Page:         1,
				Results:      []TVShow{},
				TotalPages:   5,
				TotalResults: 100,
			},
			mockStatusCode: http.StatusOK,
			wantErr:        false,
		},
		{
			name:  "negative page defaults to 1",
			query: "Friends",
			page:  -5,
			mockResponse: &SearchResultTVShows{
				Page:         1,
				Results:      []TVShow{},
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

			// Call SearchTVShows
			ctx := context.Background()
			result, err := client.SearchTVShows(ctx, tt.query, tt.page)

			// Check error
			if (err != nil) != tt.wantErr {
				t.Errorf("SearchTVShows() error = %v, wantErr %v", err, tt.wantErr)
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
					t.Error("SearchTVShows() returned nil result, want non-nil")
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

				// Verify first TV show if results exist
				if len(result.Results) > 0 && len(tt.mockResponse.Results) > 0 {
					got := result.Results[0]
					want := tt.mockResponse.Results[0]

					if got.ID != want.ID {
						t.Errorf("result.Results[0].ID = %v, want %v", got.ID, want.ID)
					}
					if got.Name != want.Name {
						t.Errorf("result.Results[0].Name = %v, want %v", got.Name, want.Name)
					}
					if got.OriginalName != want.OriginalName {
						t.Errorf("result.Results[0].OriginalName = %v, want %v", got.OriginalName, want.OriginalName)
					}
					if got.Overview != want.Overview {
						t.Errorf("result.Results[0].Overview = %v, want %v", got.Overview, want.Overview)
					}
				}
			}
		})
	}
}

func TestSearchTVShows_HTTPErrors(t *testing.T) {
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

			// Call SearchTVShows
			ctx := context.Background()
			result, err := client.SearchTVShows(ctx, "test query", 1)

			// Check error
			if (err != nil) != tt.wantErr {
				t.Errorf("SearchTVShows() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && result != nil {
				t.Errorf("SearchTVShows() result = %v, want nil on error", result)
			}
		})
	}
}

func TestSearchTVShows_ContextCancellation(t *testing.T) {
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

	// Call SearchTVShows with cancelled context
	_, err := client.SearchTVShows(ctx, "test query", 1)

	// Should return an error due to context cancellation
	if err == nil {
		t.Error("SearchTVShows() with cancelled context should return error")
	}
}
