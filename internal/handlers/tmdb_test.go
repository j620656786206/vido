package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/alexyu/vido/internal/middleware"
	"github.com/alexyu/vido/internal/tmdb"
	"github.com/gin-gonic/gin"
)

// mockTMDbClient is a mock implementation of the TMDb client for testing
type mockTMDbClient struct {
	searchMoviesFunc    func(ctx context.Context, query string, page int) (*tmdb.SearchResultMovies, error)
	getMovieDetailsFunc func(ctx context.Context, movieID int) (*tmdb.MovieDetails, error)
	searchTVShowsFunc   func(ctx context.Context, query string, page int) (*tmdb.SearchResultTVShows, error)
	getTVShowDetailsFunc func(ctx context.Context, tvID int) (*tmdb.TVShowDetails, error)
}

func (m *mockTMDbClient) SearchMovies(ctx context.Context, query string, page int) (*tmdb.SearchResultMovies, error) {
	if m.searchMoviesFunc != nil {
		return m.searchMoviesFunc(ctx, query, page)
	}
	return nil, errors.New("not implemented")
}

func (m *mockTMDbClient) GetMovieDetails(ctx context.Context, movieID int) (*tmdb.MovieDetails, error) {
	if m.getMovieDetailsFunc != nil {
		return m.getMovieDetailsFunc(ctx, movieID)
	}
	return nil, errors.New("not implemented")
}

func (m *mockTMDbClient) SearchTVShows(ctx context.Context, query string, page int) (*tmdb.SearchResultTVShows, error) {
	if m.searchTVShowsFunc != nil {
		return m.searchTVShowsFunc(ctx, query, page)
	}
	return nil, errors.New("not implemented")
}

func (m *mockTMDbClient) GetTVShowDetails(ctx context.Context, tvID int) (*tmdb.TVShowDetails, error) {
	if m.getTVShowDetailsFunc != nil {
		return m.getTVShowDetailsFunc(ctx, tvID)
	}
	return nil, errors.New("not implemented")
}

func TestSearchMoviesHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	posterPath := "/poster.jpg"
	backdropPath := "/backdrop.jpg"

	tests := []struct {
		name             string
		queryParam       string
		pageParam        string
		mockResponse     *tmdb.SearchResultMovies
		mockError        error
		wantStatus       int
		wantErrorCode    string
		wantErrorMessage string
		checkResponse    bool
	}{
		{
			name:       "successful search with results",
			queryParam: "Fight Club",
			pageParam:  "1",
			mockResponse: &tmdb.SearchResultMovies{
				Page: 1,
				Results: []tmdb.Movie{
					{
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
				},
				TotalPages:   1,
				TotalResults: 1,
			},
			mockError:     nil,
			wantStatus:    http.StatusOK,
			checkResponse: true,
		},
		{
			name:       "successful search with no results",
			queryParam: "NonexistentMovie123456",
			pageParam:  "1",
			mockResponse: &tmdb.SearchResultMovies{
				Page:         1,
				Results:      []tmdb.Movie{},
				TotalPages:   0,
				TotalResults: 0,
			},
			mockError:     nil,
			wantStatus:    http.StatusOK,
			checkResponse: true,
		},
		{
			name:             "missing query parameter",
			queryParam:       "",
			pageParam:        "1",
			mockResponse:     nil,
			mockError:        nil,
			wantStatus:       http.StatusBadRequest,
			wantErrorCode:    "VALIDATION_ERROR",
			wantErrorMessage: "query parameter is required",
		},
		{
			name:             "invalid page parameter",
			queryParam:       "Inception",
			pageParam:        "invalid",
			mockResponse:     nil,
			mockError:        nil,
			wantStatus:       http.StatusBadRequest,
			wantErrorCode:    "VALIDATION_ERROR",
			wantErrorMessage: "page must be a positive integer",
		},
		{
			name:             "negative page parameter",
			queryParam:       "Inception",
			pageParam:        "-1",
			mockResponse:     nil,
			mockError:        nil,
			wantStatus:       http.StatusBadRequest,
			wantErrorCode:    "VALIDATION_ERROR",
			wantErrorMessage: "page must be a positive integer",
		},
		{
			name:             "TMDb not found error",
			queryParam:       "NotFound",
			pageParam:        "1",
			mockResponse:     nil,
			mockError:        tmdb.NewNotFoundError("movie"),
			wantStatus:       http.StatusNotFound,
			wantErrorCode:    "TMDB_NOT_FOUND",
			wantErrorMessage: "TMDb resource not found: movie",
		},
		{
			name:             "TMDb rate limit error",
			queryParam:       "TooManyRequests",
			pageParam:        "1",
			mockResponse:     nil,
			mockError:        tmdb.NewRateLimitError(),
			wantStatus:       http.StatusTooManyRequests,
			wantErrorCode:    "TMDB_RATE_LIMIT_EXCEEDED",
			wantErrorMessage: "TMDb API rate limit exceeded. Please try again later.",
		},
		{
			name:             "TMDb unauthorized error",
			queryParam:       "Unauthorized",
			pageParam:        "1",
			mockResponse:     nil,
			mockError:        tmdb.NewUnauthorizedError("Invalid API key"),
			wantStatus:       http.StatusUnauthorized,
			wantErrorCode:    "TMDB_UNAUTHORIZED",
			wantErrorMessage: "Invalid API key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock client
			mockClient := &mockTMDbClient{
				searchMoviesFunc: func(ctx context.Context, query string, page int) (*tmdb.SearchResultMovies, error) {
					if tt.mockError != nil {
						return nil, tt.mockError
					}
					return tt.mockResponse, nil
				},
			}

			// Create handler
			handler := NewTMDbHandler(mockClient)

			// Create test context and request
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			// Build URL with query parameters
			reqURL := "/api/v1/movies/search"
			if tt.queryParam != "" || tt.pageParam != "" {
				params := url.Values{}
				if tt.queryParam != "" {
					params.Add("query", tt.queryParam)
				}
				if tt.pageParam != "" {
					params.Add("page", tt.pageParam)
				}
				reqURL += "?" + params.Encode()
			}
			c.Request = httptest.NewRequest("GET", reqURL, nil)

			// Call handler
			handler.SearchMovies(c)

			// Apply error handler middleware if there are errors
			if len(c.Errors) > 0 {
				// Manually handle errors like the middleware does
				err := c.Errors.Last()
				if appErr, ok := err.Err.(*middleware.AppError); ok {
					w.Code = appErr.StatusCode
					response := middleware.ErrorResponse{
						Error: middleware.ErrorDetail{
							Code:    appErr.Code,
							Message: appErr.Message,
						},
					}
					jsonBytes, _ := json.Marshal(response)
					w.Body.Write(jsonBytes)
				}
			}

			// Check status code
			if w.Code != tt.wantStatus {
				t.Errorf("Status = %v, want %v", w.Code, tt.wantStatus)
			}

			// Check error response if expected
			if tt.wantErrorCode != "" {
				var response middleware.ErrorResponse
				if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
					t.Fatalf("Failed to unmarshal error response: %v", err)
				}

				if response.Error.Code != tt.wantErrorCode {
					t.Errorf("Error code = %v, want %v", response.Error.Code, tt.wantErrorCode)
				}

				if response.Error.Message != tt.wantErrorMessage {
					t.Errorf("Error message = %v, want %v", response.Error.Message, tt.wantErrorMessage)
				}
			}

			// Check successful response if expected
			if tt.checkResponse && w.Code == http.StatusOK {
				var response tmdb.SearchResultMovies
				if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}

				if response.Page != tt.mockResponse.Page {
					t.Errorf("Page = %v, want %v", response.Page, tt.mockResponse.Page)
				}

				if len(response.Results) != len(tt.mockResponse.Results) {
					t.Errorf("Results length = %v, want %v", len(response.Results), len(tt.mockResponse.Results))
				}
			}
		})
	}
}

func TestGetMovieDetailsHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	posterPath := "/poster.jpg"
	backdropPath := "/backdrop.jpg"
	homepage := "https://example.com"

	tests := []struct {
		name             string
		movieID          string
		mockResponse     *tmdb.MovieDetails
		mockError        error
		wantStatus       int
		wantErrorCode    string
		wantErrorMessage string
		checkResponse    bool
	}{
		{
			name:    "successful get movie details",
			movieID: "550",
			mockResponse: &tmdb.MovieDetails{
				Movie: tmdb.Movie{
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
				Tagline: "Mischief. Mayhem. Soap.",
				ImdbID:  "tt0137523",
				Homepage: &homepage,
			},
			mockError:     nil,
			wantStatus:    http.StatusOK,
			checkResponse: true,
		},
		{
			name:             "invalid movie ID - non-numeric",
			movieID:          "invalid",
			mockResponse:     nil,
			mockError:        nil,
			wantStatus:       http.StatusBadRequest,
			wantErrorCode:    "VALIDATION_ERROR",
			wantErrorMessage: "invalid movie ID",
		},
		{
			name:             "invalid movie ID - zero",
			movieID:          "0",
			mockResponse:     nil,
			mockError:        nil,
			wantStatus:       http.StatusBadRequest,
			wantErrorCode:    "VALIDATION_ERROR",
			wantErrorMessage: "invalid movie ID",
		},
		{
			name:             "invalid movie ID - negative",
			movieID:          "-1",
			mockResponse:     nil,
			mockError:        nil,
			wantStatus:       http.StatusBadRequest,
			wantErrorCode:    "VALIDATION_ERROR",
			wantErrorMessage: "invalid movie ID",
		},
		{
			name:             "TMDb not found error",
			movieID:          "999999",
			mockResponse:     nil,
			mockError:        tmdb.NewNotFoundError("movie"),
			wantStatus:       http.StatusNotFound,
			wantErrorCode:    "TMDB_NOT_FOUND",
			wantErrorMessage: "TMDb resource not found: movie",
		},
		{
			name:             "TMDb rate limit error",
			movieID:          "550",
			mockResponse:     nil,
			mockError:        tmdb.NewRateLimitError(),
			wantStatus:       http.StatusTooManyRequests,
			wantErrorCode:    "TMDB_RATE_LIMIT_EXCEEDED",
			wantErrorMessage: "TMDb API rate limit exceeded. Please try again later.",
		},
		{
			name:             "TMDb server error",
			movieID:          "550",
			mockResponse:     nil,
			mockError:        tmdb.NewServerError(errors.New("server error")),
			wantStatus:       http.StatusBadGateway,
			wantErrorCode:    "TMDB_SERVER_ERROR",
			wantErrorMessage: "TMDb API server error occurred",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock client
			mockClient := &mockTMDbClient{
				getMovieDetailsFunc: func(ctx context.Context, movieID int) (*tmdb.MovieDetails, error) {
					if tt.mockError != nil {
						return nil, tt.mockError
					}
					return tt.mockResponse, nil
				},
			}

			// Create handler
			handler := NewTMDbHandler(mockClient)

			// Create test context and request
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", fmt.Sprintf("/api/v1/movies/%s", tt.movieID), nil)
			c.Params = gin.Params{gin.Param{Key: "id", Value: tt.movieID}}

			// Call handler
			handler.GetMovieDetails(c)

			// Apply error handler middleware if there are errors
			if len(c.Errors) > 0 {
				err := c.Errors.Last()
				if appErr, ok := err.Err.(*middleware.AppError); ok {
					w.Code = appErr.StatusCode
					response := middleware.ErrorResponse{
						Error: middleware.ErrorDetail{
							Code:    appErr.Code,
							Message: appErr.Message,
						},
					}
					jsonBytes, _ := json.Marshal(response)
					w.Body.Write(jsonBytes)
				}
			}

			// Check status code
			if w.Code != tt.wantStatus {
				t.Errorf("Status = %v, want %v", w.Code, tt.wantStatus)
			}

			// Check error response if expected
			if tt.wantErrorCode != "" {
				var response middleware.ErrorResponse
				if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
					t.Fatalf("Failed to unmarshal error response: %v", err)
				}

				if response.Error.Code != tt.wantErrorCode {
					t.Errorf("Error code = %v, want %v", response.Error.Code, tt.wantErrorCode)
				}

				if response.Error.Message != tt.wantErrorMessage {
					t.Errorf("Error message = %v, want %v", response.Error.Message, tt.wantErrorMessage)
				}
			}

			// Check successful response if expected
			if tt.checkResponse && w.Code == http.StatusOK {
				var response tmdb.MovieDetails
				if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}

				if response.ID != tt.mockResponse.ID {
					t.Errorf("ID = %v, want %v", response.ID, tt.mockResponse.ID)
				}

				if response.Title != tt.mockResponse.Title {
					t.Errorf("Title = %v, want %v", response.Title, tt.mockResponse.Title)
				}
			}
		})
	}
}

func TestSearchTVShowsHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	posterPath := "/poster.jpg"
	backdropPath := "/backdrop.jpg"

	tests := []struct {
		name             string
		queryParam       string
		pageParam        string
		mockResponse     *tmdb.SearchResultTVShows
		mockError        error
		wantStatus       int
		wantErrorCode    string
		wantErrorMessage string
		checkResponse    bool
	}{
		{
			name:       "successful search with results",
			queryParam: "Breaking Bad",
			pageParam:  "1",
			mockResponse: &tmdb.SearchResultTVShows{
				Page: 1,
				Results: []tmdb.TVShow{
					{
						ID:           1396,
						Name:         "絕命毒師",
						OriginalName: "Breaking Bad",
						Overview:     "一位高中化學老師被診斷出癌症後，開始製造冰毒以維持家計。",
						FirstAirDate: "2008-01-20",
						PosterPath:   &posterPath,
						BackdropPath: &backdropPath,
						VoteAverage:  8.9,
						VoteCount:    12345,
						Popularity:   369.594,
					},
				},
				TotalPages:   1,
				TotalResults: 1,
			},
			mockError:     nil,
			wantStatus:    http.StatusOK,
			checkResponse: true,
		},
		{
			name:       "successful search with no results",
			queryParam: "NonexistentShow123456",
			pageParam:  "1",
			mockResponse: &tmdb.SearchResultTVShows{
				Page:         1,
				Results:      []tmdb.TVShow{},
				TotalPages:   0,
				TotalResults: 0,
			},
			mockError:     nil,
			wantStatus:    http.StatusOK,
			checkResponse: true,
		},
		{
			name:             "missing query parameter",
			queryParam:       "",
			pageParam:        "1",
			mockResponse:     nil,
			mockError:        nil,
			wantStatus:       http.StatusBadRequest,
			wantErrorCode:    "VALIDATION_ERROR",
			wantErrorMessage: "query parameter is required",
		},
		{
			name:             "invalid page parameter",
			queryParam:       "Game of Thrones",
			pageParam:        "invalid",
			mockResponse:     nil,
			mockError:        nil,
			wantStatus:       http.StatusBadRequest,
			wantErrorCode:    "VALIDATION_ERROR",
			wantErrorMessage: "page must be a positive integer",
		},
		{
			name:             "TMDb rate limit error",
			queryParam:       "TooManyRequests",
			pageParam:        "1",
			mockResponse:     nil,
			mockError:        tmdb.NewRateLimitError(),
			wantStatus:       http.StatusTooManyRequests,
			wantErrorCode:    "TMDB_RATE_LIMIT_EXCEEDED",
			wantErrorMessage: "TMDb API rate limit exceeded. Please try again later.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock client
			mockClient := &mockTMDbClient{
				searchTVShowsFunc: func(ctx context.Context, query string, page int) (*tmdb.SearchResultTVShows, error) {
					if tt.mockError != nil {
						return nil, tt.mockError
					}
					return tt.mockResponse, nil
				},
			}

			// Create handler
			handler := NewTMDbHandler(mockClient)

			// Create test context and request
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			// Build URL with query parameters
			reqURL := "/api/v1/tv/search"
			if tt.queryParam != "" || tt.pageParam != "" {
				params := url.Values{}
				if tt.queryParam != "" {
					params.Add("query", tt.queryParam)
				}
				if tt.pageParam != "" {
					params.Add("page", tt.pageParam)
				}
				reqURL += "?" + params.Encode()
			}
			c.Request = httptest.NewRequest("GET", reqURL, nil)

			// Call handler
			handler.SearchTVShows(c)

			// Apply error handler middleware if there are errors
			if len(c.Errors) > 0 {
				err := c.Errors.Last()
				if appErr, ok := err.Err.(*middleware.AppError); ok {
					w.Code = appErr.StatusCode
					response := middleware.ErrorResponse{
						Error: middleware.ErrorDetail{
							Code:    appErr.Code,
							Message: appErr.Message,
						},
					}
					jsonBytes, _ := json.Marshal(response)
					w.Body.Write(jsonBytes)
				}
			}

			// Check status code
			if w.Code != tt.wantStatus {
				t.Errorf("Status = %v, want %v", w.Code, tt.wantStatus)
			}

			// Check error response if expected
			if tt.wantErrorCode != "" {
				var response middleware.ErrorResponse
				if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
					t.Fatalf("Failed to unmarshal error response: %v", err)
				}

				if response.Error.Code != tt.wantErrorCode {
					t.Errorf("Error code = %v, want %v", response.Error.Code, tt.wantErrorCode)
				}

				if response.Error.Message != tt.wantErrorMessage {
					t.Errorf("Error message = %v, want %v", response.Error.Message, tt.wantErrorMessage)
				}
			}

			// Check successful response if expected
			if tt.checkResponse && w.Code == http.StatusOK {
				var response tmdb.SearchResultTVShows
				if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}

				if response.Page != tt.mockResponse.Page {
					t.Errorf("Page = %v, want %v", response.Page, tt.mockResponse.Page)
				}

				if len(response.Results) != len(tt.mockResponse.Results) {
					t.Errorf("Results length = %v, want %v", len(response.Results), len(tt.mockResponse.Results))
				}
			}
		})
	}
}

func TestGetTVShowDetailsHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	posterPath := "/poster.jpg"
	backdropPath := "/backdrop.jpg"
	homepage := "https://example.com"

	tests := []struct {
		name             string
		tvID             string
		mockResponse     *tmdb.TVShowDetails
		mockError        error
		wantStatus       int
		wantErrorCode    string
		wantErrorMessage string
		checkResponse    bool
	}{
		{
			name: "successful get TV show details",
			tvID: "1396",
			mockResponse: &tmdb.TVShowDetails{
				TVShow: tmdb.TVShow{
					ID:           1396,
					Name:         "絕命毒師",
					OriginalName: "Breaking Bad",
					Overview:     "一位高中化學老師被診斷出癌症後，開始製造冰毒以維持家計。",
					FirstAirDate: "2008-01-20",
					PosterPath:   &posterPath,
					BackdropPath: &backdropPath,
					VoteAverage:  8.9,
					VoteCount:    12345,
					Popularity:   369.594,
				},
				NumberOfEpisodes: 62,
				NumberOfSeasons:  5,
				Status:           "Ended",
				Tagline:          "Change the equation.",
				Homepage:         &homepage,
			},
			mockError:     nil,
			wantStatus:    http.StatusOK,
			checkResponse: true,
		},
		{
			name:             "invalid TV show ID - non-numeric",
			tvID:             "invalid",
			mockResponse:     nil,
			mockError:        nil,
			wantStatus:       http.StatusBadRequest,
			wantErrorCode:    "VALIDATION_ERROR",
			wantErrorMessage: "invalid TV show ID",
		},
		{
			name:             "invalid TV show ID - zero",
			tvID:             "0",
			mockResponse:     nil,
			mockError:        nil,
			wantStatus:       http.StatusBadRequest,
			wantErrorCode:    "VALIDATION_ERROR",
			wantErrorMessage: "invalid TV show ID",
		},
		{
			name:             "invalid TV show ID - negative",
			tvID:             "-1",
			mockResponse:     nil,
			mockError:        nil,
			wantStatus:       http.StatusBadRequest,
			wantErrorCode:    "VALIDATION_ERROR",
			wantErrorMessage: "invalid TV show ID",
		},
		{
			name:             "TMDb not found error",
			tvID:             "999999",
			mockResponse:     nil,
			mockError:        tmdb.NewNotFoundError("TV show"),
			wantStatus:       http.StatusNotFound,
			wantErrorCode:    "TMDB_NOT_FOUND",
			wantErrorMessage: "TMDb resource not found: TV show",
		},
		{
			name:             "TMDb rate limit error",
			tvID:             "1396",
			mockResponse:     nil,
			mockError:        tmdb.NewRateLimitError(),
			wantStatus:       http.StatusTooManyRequests,
			wantErrorCode:    "TMDB_RATE_LIMIT_EXCEEDED",
			wantErrorMessage: "TMDb API rate limit exceeded. Please try again later.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock client
			mockClient := &mockTMDbClient{
				getTVShowDetailsFunc: func(ctx context.Context, tvID int) (*tmdb.TVShowDetails, error) {
					if tt.mockError != nil {
						return nil, tt.mockError
					}
					return tt.mockResponse, nil
				},
			}

			// Create handler
			handler := NewTMDbHandler(mockClient)

			// Create test context and request
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", fmt.Sprintf("/api/v1/tv/%s", tt.tvID), nil)
			c.Params = gin.Params{gin.Param{Key: "id", Value: tt.tvID}}

			// Call handler
			handler.GetTVShowDetails(c)

			// Apply error handler middleware if there are errors
			if len(c.Errors) > 0 {
				err := c.Errors.Last()
				if appErr, ok := err.Err.(*middleware.AppError); ok {
					w.Code = appErr.StatusCode
					response := middleware.ErrorResponse{
						Error: middleware.ErrorDetail{
							Code:    appErr.Code,
							Message: appErr.Message,
						},
					}
					jsonBytes, _ := json.Marshal(response)
					w.Body.Write(jsonBytes)
				}
			}

			// Check status code
			if w.Code != tt.wantStatus {
				t.Errorf("Status = %v, want %v", w.Code, tt.wantStatus)
			}

			// Check error response if expected
			if tt.wantErrorCode != "" {
				var response middleware.ErrorResponse
				if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
					t.Fatalf("Failed to unmarshal error response: %v", err)
				}

				if response.Error.Code != tt.wantErrorCode {
					t.Errorf("Error code = %v, want %v", response.Error.Code, tt.wantErrorCode)
				}

				if response.Error.Message != tt.wantErrorMessage {
					t.Errorf("Error message = %v, want %v", response.Error.Message, tt.wantErrorMessage)
				}
			}

			// Check successful response if expected
			if tt.checkResponse && w.Code == http.StatusOK {
				var response tmdb.TVShowDetails
				if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}

				if response.ID != tt.mockResponse.ID {
					t.Errorf("ID = %v, want %v", response.ID, tt.mockResponse.ID)
				}

				if response.Name != tt.mockResponse.Name {
					t.Errorf("Name = %v, want %v", response.Name, tt.mockResponse.Name)
				}
			}
		})
	}
}
