package tmdb

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/alexyu/vido/internal/config"
	"github.com/alexyu/vido/internal/middleware"
	"golang.org/x/time/rate"
)

func TestNewClient(t *testing.T) {
	cfg := &config.Config{
		TMDbAPIKey:           "test-api-key",
		TMDbDefaultLanguage: "zh-TW",
	}

	client := NewClient(cfg)

	if client.baseURL != DefaultBaseURL {
		t.Errorf("baseURL = %v, want %v", client.baseURL, DefaultBaseURL)
	}

	if client.apiKey != "test-api-key" {
		t.Errorf("apiKey = %v, want %v", client.apiKey, "test-api-key")
	}

	if client.language != "zh-TW" {
		t.Errorf("language = %v, want %v", client.language, "zh-TW")
	}

	if client.httpClient == nil {
		t.Error("httpClient is nil, want non-nil")
	}

	if client.limiter == nil {
		t.Error("limiter is nil, want non-nil")
	}

	if client.httpClient.Timeout != 30*time.Second {
		t.Errorf("httpClient.Timeout = %v, want %v", client.httpClient.Timeout, 30*time.Second)
	}
}

func TestBuildURL(t *testing.T) {
	client := &Client{
		baseURL:  DefaultBaseURL,
		apiKey:   "test-api-key",
		language: "zh-TW",
	}

	tests := []struct {
		name         string
		endpoint     string
		queryParams  url.Values
		wantContains []string
	}{
		{
			name:     "basic endpoint with no params",
			endpoint: "/movie/550",
			queryParams: nil,
			wantContains: []string{
				"api_key=test-api-key",
				"language=zh-TW",
			},
		},
		{
			name:     "endpoint with additional params",
			endpoint: "/search/movie",
			queryParams: url.Values{
				"query": []string{"Fight Club"},
				"page":  []string{"1"},
			},
			wantContains: []string{
				"api_key=test-api-key",
				"language=zh-TW",
				"query=Fight+Club",
				"page=1",
			},
		},
		{
			name:     "endpoint with special characters",
			endpoint: "/search/movie",
			queryParams: url.Values{
				"query": []string{"搏擊俱樂部"},
			},
			wantContains: []string{
				"api_key=test-api-key",
				"language=zh-TW",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotURL, err := client.buildURL(tt.endpoint, tt.queryParams)
			if err != nil {
				t.Fatalf("buildURL() error = %v", err)
			}

			for _, want := range tt.wantContains {
				if !contains(gotURL, want) {
					t.Errorf("buildURL() = %v, should contain %v", gotURL, want)
				}
			}

			// Verify base URL is included
			if !contains(gotURL, DefaultBaseURL) {
				t.Errorf("buildURL() = %v, should contain %v", gotURL, DefaultBaseURL)
			}

			// Verify endpoint is included
			if !contains(gotURL, tt.endpoint) {
				t.Errorf("buildURL() = %v, should contain %v", gotURL, tt.endpoint)
			}
		})
	}
}

func TestDoRequest_Success(t *testing.T) {
	// Create a test server that returns a successful response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify headers
		if r.Header.Get("Accept") != "application/json" {
			t.Errorf("Accept header = %v, want application/json", r.Header.Get("Accept"))
		}
		if r.Header.Get("User-Agent") != "Vido/1.0" {
			t.Errorf("User-Agent header = %v, want Vido/1.0", r.Header.Get("User-Agent"))
		}

		// Verify query parameters
		query := r.URL.Query()
		if query.Get("api_key") != "test-key" {
			t.Errorf("api_key param = %v, want test-key", query.Get("api_key"))
		}
		if query.Get("language") != "zh-TW" {
			t.Errorf("language param = %v, want zh-TW", query.Get("language"))
		}

		// Return success response
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	}))
	defer server.Close()

	client := &Client{
		baseURL:  server.URL,
		apiKey:   "test-key",
		language: "zh-TW",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		limiter: newTestRateLimiter(),
	}

	ctx := context.Background()
	body, err := client.doRequest(ctx, http.MethodGet, "/test", nil)
	if err != nil {
		t.Fatalf("doRequest() error = %v, want nil", err)
	}

	expected := `{"status": "ok"}`
	if string(body) != expected {
		t.Errorf("doRequest() body = %v, want %v", string(body), expected)
	}
}

func TestDoRequest_HTTPErrors(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		responseBody   string
		wantErrorCode  string
		wantStatusCode int
	}{
		{
			name:           "401 unauthorized - invalid API key",
			statusCode:     http.StatusUnauthorized,
			responseBody:   `{"status_code": 7, "status_message": "Invalid API key", "success": false}`,
			wantErrorCode:  ErrCodeUnauthorized,
			wantStatusCode: http.StatusUnauthorized,
		},
		{
			name:           "404 not found - resource not found",
			statusCode:     http.StatusNotFound,
			responseBody:   `{"status_code": 34, "status_message": "The resource you requested could not be found.", "success": false}`,
			wantErrorCode:  ErrCodeNotFound,
			wantStatusCode: http.StatusNotFound,
		},
		{
			name:           "429 rate limit exceeded",
			statusCode:     http.StatusTooManyRequests,
			responseBody:   `{"status_code": 25, "status_message": "Your request count is over the allowed limit.", "success": false}`,
			wantErrorCode:  ErrCodeRateLimitExceeded,
			wantStatusCode: http.StatusTooManyRequests,
		},
		{
			name:           "400 bad request - invalid parameters",
			statusCode:     http.StatusBadRequest,
			responseBody:   `{"status_code": 22, "status_message": "Invalid parameters", "success": false}`,
			wantErrorCode:  ErrCodeBadRequest,
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "500 server error",
			statusCode:     http.StatusInternalServerError,
			responseBody:   `{"status_code": 11, "status_message": "Internal error", "success": false}`,
			wantErrorCode:  ErrCodeServerError,
			wantStatusCode: http.StatusBadGateway,
		},
		{
			name:           "503 service unavailable",
			statusCode:     http.StatusServiceUnavailable,
			responseBody:   `{"status_code": 9, "status_message": "Service offline", "success": false}`,
			wantErrorCode:  ErrCodeServerError,
			wantStatusCode: http.StatusBadGateway,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := &Client{
				baseURL:  server.URL,
				apiKey:   "test-key",
				language: "zh-TW",
				httpClient: &http.Client{
					Timeout: 30 * time.Second,
				},
				limiter: newTestRateLimiter(),
			}

			ctx := context.Background()
			_, err := client.doRequest(ctx, http.MethodGet, "/test", nil)
			if err == nil {
				t.Fatal("doRequest() error = nil, want non-nil")
			}

			// Check if error is AppError
			appErr, ok := err.(*middleware.AppError)
			if !ok {
				t.Fatalf("error type = %T, want *middleware.AppError", err)
			}

			if appErr.Code != tt.wantErrorCode {
				t.Errorf("error code = %v, want %v", appErr.Code, tt.wantErrorCode)
			}

			if appErr.StatusCode != tt.wantStatusCode {
				t.Errorf("error status code = %v, want %v", appErr.StatusCode, tt.wantStatusCode)
			}
		})
	}
}

func TestDoRequest_ContextCancellation(t *testing.T) {
	// Create a server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	}))
	defer server.Close()

	client := &Client{
		baseURL:  server.URL,
		apiKey:   "test-key",
		language: "zh-TW",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		limiter: newTestRateLimiter(),
	}

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.doRequest(ctx, http.MethodGet, "/test", nil)
	if err == nil {
		t.Fatal("doRequest() error = nil, want context cancellation error")
	}
}

func TestGet_Success(t *testing.T) {
	type TestResponse struct {
		ID    int    `json:"id"`
		Title string `json:"title"`
	}

	expectedResponse := TestResponse{
		ID:    550,
		Title: "Fight Club",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(expectedResponse)
	}))
	defer server.Close()

	client := &Client{
		baseURL:  server.URL,
		apiKey:   "test-key",
		language: "zh-TW",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		limiter: newTestRateLimiter(),
	}

	ctx := context.Background()
	var result TestResponse
	err := client.Get(ctx, "/movie/550", nil, &result)
	if err != nil {
		t.Fatalf("Get() error = %v, want nil", err)
	}

	if result.ID != expectedResponse.ID {
		t.Errorf("result.ID = %v, want %v", result.ID, expectedResponse.ID)
	}

	if result.Title != expectedResponse.Title {
		t.Errorf("result.Title = %v, want %v", result.Title, expectedResponse.Title)
	}
}

func TestGet_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{invalid json}`))
	}))
	defer server.Close()

	client := &Client{
		baseURL:  server.URL,
		apiKey:   "test-key",
		language: "zh-TW",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		limiter: newTestRateLimiter(),
	}

	ctx := context.Background()
	var result map[string]interface{}
	err := client.Get(ctx, "/test", nil, &result)
	if err == nil {
		t.Fatal("Get() error = nil, want JSON unmarshal error")
	}

	if !contains(err.Error(), "failed to unmarshal response") {
		t.Errorf("error message = %v, should contain 'failed to unmarshal response'", err.Error())
	}
}

func TestRateLimiter(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	}))
	defer server.Close()

	client := &Client{
		baseURL:  server.URL,
		apiKey:   "test-key",
		language: "zh-TW",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		limiter: newTestRateLimiter(),
	}

	ctx := context.Background()

	// Make multiple rapid requests
	start := time.Now()
	for i := 0; i < 5; i++ {
		_, err := client.doRequest(ctx, http.MethodGet, "/test", nil)
		if err != nil {
			t.Fatalf("doRequest() error = %v, want nil", err)
		}
	}
	elapsed := time.Since(start)

	// Verify all requests completed
	if requestCount != 5 {
		t.Errorf("requestCount = %v, want 5", requestCount)
	}

	// Rate limiter should not add significant delay for small number of requests
	// with our test limiter (which has a high burst)
	if elapsed > 1*time.Second {
		t.Errorf("elapsed time = %v, want < 1s (rate limiter may be too strict)", elapsed)
	}
}

func TestDoRequest_QueryParameters(t *testing.T) {
	var receivedQuery url.Values

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedQuery = r.URL.Query()
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	}))
	defer server.Close()

	client := &Client{
		baseURL:  server.URL,
		apiKey:   "test-key",
		language: "zh-TW",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		limiter: newTestRateLimiter(),
	}

	ctx := context.Background()
	queryParams := url.Values{
		"query": []string{"搏擊俱樂部"},
		"page":  []string{"2"},
	}

	_, err := client.doRequest(ctx, http.MethodGet, "/search/movie", queryParams)
	if err != nil {
		t.Fatalf("doRequest() error = %v, want nil", err)
	}

	// Verify API key and language were added
	if receivedQuery.Get("api_key") != "test-key" {
		t.Errorf("api_key = %v, want test-key", receivedQuery.Get("api_key"))
	}

	if receivedQuery.Get("language") != "zh-TW" {
		t.Errorf("language = %v, want zh-TW", receivedQuery.Get("language"))
	}

	// Verify custom query params were preserved
	if receivedQuery.Get("query") != "搏擊俱樂部" {
		t.Errorf("query = %v, want 搏擊俱樂部", receivedQuery.Get("query"))
	}

	if receivedQuery.Get("page") != "2" {
		t.Errorf("page = %v, want 2", receivedQuery.Get("page"))
	}
}

func TestClient_DifferentLanguages(t *testing.T) {
	tests := []struct {
		name         string
		language     string
		wantLanguage string
	}{
		{
			name:         "Traditional Chinese",
			language:     "zh-TW",
			wantLanguage: "zh-TW",
		},
		{
			name:         "Simplified Chinese",
			language:     "zh-CN",
			wantLanguage: "zh-CN",
		},
		{
			name:         "English",
			language:     "en-US",
			wantLanguage: "en-US",
		},
		{
			name:         "Japanese",
			language:     "ja-JP",
			wantLanguage: "ja-JP",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var receivedLanguage string

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedLanguage = r.URL.Query().Get("language")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"status": "ok"}`))
			}))
			defer server.Close()

			client := &Client{
				baseURL:  server.URL,
				apiKey:   "test-key",
				language: tt.language,
				httpClient: &http.Client{
					Timeout: 30 * time.Second,
				},
				limiter: newTestRateLimiter(),
			}

			ctx := context.Background()
			_, err := client.doRequest(ctx, http.MethodGet, "/test", nil)
			if err != nil {
				t.Fatalf("doRequest() error = %v, want nil", err)
			}

			if receivedLanguage != tt.wantLanguage {
				t.Errorf("received language = %v, want %v", receivedLanguage, tt.wantLanguage)
			}
		})
	}
}

// Helper functions

// newTestRateLimiter creates a rate limiter for testing that allows high burst
func newTestRateLimiter() *rate.Limiter {
	// Use a more permissive limiter for tests to avoid flakiness
	// Allow 100 requests per second with burst of 100
	return rate.NewLimiter(100, 100)
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || (len(s) > 0 && len(substr) > 0 && stringContains(s, substr)))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
