package tmdb

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name     string
		cfg      ClientConfig
		wantLang string
		wantURL  string
	}{
		{
			name: "default values",
			cfg: ClientConfig{
				APIKey: "test-api-key",
			},
			wantLang: "zh-TW",
			wantURL:  DefaultBaseURL,
		},
		{
			name: "custom language",
			cfg: ClientConfig{
				APIKey:   "test-api-key",
				Language: "en",
			},
			wantLang: "en",
			wantURL:  DefaultBaseURL,
		},
		{
			name: "custom base URL",
			cfg: ClientConfig{
				APIKey:  "test-api-key",
				BaseURL: "https://custom.api.com",
			},
			wantLang: "zh-TW",
			wantURL:  "https://custom.api.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(tt.cfg)

			assert.NotNil(t, client)
			assert.Equal(t, tt.wantLang, client.language)
			assert.Equal(t, tt.wantURL, client.baseURL)
			assert.Equal(t, tt.cfg.APIKey, client.apiKey)
			assert.NotNil(t, client.httpClient)
			assert.NotNil(t, client.limiter)
		})
	}
}

func TestClient_buildURL(t *testing.T) {
	client := NewClient(ClientConfig{
		APIKey:   "test-key",
		Language: "zh-TW",
	})

	tests := []struct {
		name        string
		endpoint    string
		queryParams map[string]string
		wantContains []string
	}{
		{
			name:     "simple endpoint",
			endpoint: "/search/movie",
			wantContains: []string{
				"api_key=test-key",
				"language=zh-TW",
				"/search/movie",
			},
		},
		{
			name:     "endpoint with query params",
			endpoint: "/search/movie",
			queryParams: map[string]string{
				"query": "test movie",
				"page":  "1",
			},
			wantContains: []string{
				"api_key=test-key",
				"language=zh-TW",
				"query=test+movie",
				"page=1",
			},
		},
		{
			name:     "endpoint with custom language",
			endpoint: "/movie/123",
			queryParams: map[string]string{
				"language": "en",
			},
			wantContains: []string{
				"api_key=test-key",
				"language=en", // custom language should override default
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var params map[string][]string
			if tt.queryParams != nil {
				params = make(map[string][]string)
				for k, v := range tt.queryParams {
					params[k] = []string{v}
				}
			}

			url, err := client.buildURL(tt.endpoint, params)
			require.NoError(t, err)

			for _, want := range tt.wantContains {
				assert.Contains(t, url, want)
			}
		})
	}
}

func TestClient_Get_Success(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Accept"))
		assert.Contains(t, r.Header.Get("User-Agent"), "Vido")

		// Return mock response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"page": 1, "results": [], "total_pages": 0, "total_results": 0}`))
	}))
	defer server.Close()

	client := NewClient(ClientConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
	})

	var result SearchResultMovies
	err := client.Get(context.Background(), "/search/movie", nil, &result)

	require.NoError(t, err)
	assert.Equal(t, 1, result.Page)
	assert.Empty(t, result.Results)
}

func TestClient_Get_HTTPError(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		responseBody   string
		wantErrorCode  string
	}{
		{
			name:          "not found",
			statusCode:    http.StatusNotFound,
			responseBody:  `{"status_code": 34, "status_message": "Not found"}`,
			wantErrorCode: ErrCodeNotFound,
		},
		{
			name:          "unauthorized",
			statusCode:    http.StatusUnauthorized,
			responseBody:  `{"status_code": 7, "status_message": "Invalid API key"}`,
			wantErrorCode: ErrCodeUnauthorized,
		},
		{
			name:          "rate limit",
			statusCode:    http.StatusTooManyRequests,
			responseBody:  `{"status_code": 25, "status_message": "Rate limit exceeded"}`,
			wantErrorCode: ErrCodeRateLimitExceeded,
		},
		{
			name:          "server error",
			statusCode:    http.StatusInternalServerError,
			responseBody:  `{"status_code": 1, "status_message": "Internal error"}`,
			wantErrorCode: ErrCodeServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := NewClient(ClientConfig{
				APIKey:  "test-key",
				BaseURL: server.URL,
			})

			var result SearchResultMovies
			err := client.Get(context.Background(), "/search/movie", nil, &result)

			require.Error(t, err)
			tmdbErr, ok := err.(*TMDbError)
			require.True(t, ok, "error should be TMDbError")
			assert.Equal(t, tt.wantErrorCode, tmdbErr.Code)
		})
	}
}

func TestClient_Get_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(ClientConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	var result SearchResultMovies
	err := client.Get(ctx, "/search/movie", nil, &result)

	require.Error(t, err)
}

func TestClient_InterfaceCompliance(t *testing.T) {
	// Verify Client implements ClientInterface
	var _ ClientInterface = (*Client)(nil)
}

func TestClient_RateLimiting(t *testing.T) {
	// This test verifies that the rate limiter correctly throttles requests
	// TMDb rate limit: 40 requests per 10 seconds

	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"page": 1, "results": [], "total_pages": 0, "total_results": 0}`))
	}))
	defer server.Close()

	client := NewClient(ClientConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
	})

	// Send 5 concurrent requests - they should complete but be rate-limited
	numRequests := 5
	done := make(chan error, numRequests)

	start := time.Now()

	for i := 0; i < numRequests; i++ {
		go func() {
			var result SearchResultMovies
			err := client.Get(context.Background(), "/search/movie", nil, &result)
			done <- err
		}()
	}

	// Wait for all requests to complete
	for i := 0; i < numRequests; i++ {
		err := <-done
		assert.NoError(t, err)
	}

	elapsed := time.Since(start)

	// Verify all requests completed
	assert.Equal(t, numRequests, requestCount)

	// Rate limiter should allow burst of up to 40 requests, so 5 requests
	// should complete quickly (within 1 second, allowing for test overhead)
	assert.Less(t, elapsed, 2*time.Second,
		"5 requests should complete within burst allowance")
}

func TestClient_RateLimiting_ExceedsBurst(t *testing.T) {
	// Test that requests beyond burst are throttled

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"page": 1, "results": [], "total_pages": 0, "total_results": 0}`))
	}))
	defer server.Close()

	client := NewClient(ClientConfig{
		APIKey:  "test-key",
		BaseURL: server.URL,
	})

	// Exhaust burst capacity first
	for i := 0; i < 40; i++ {
		var result SearchResultMovies
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_ = client.Get(ctx, "/search/movie", nil, &result)
		cancel()
	}

	// Next request should be delayed by rate limiter
	start := time.Now()
	var result SearchResultMovies
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := client.Get(ctx, "/search/movie", nil, &result)

	elapsed := time.Since(start)

	// Request should succeed but take at least 250ms (10s/40 = 250ms per request)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, elapsed.Milliseconds(), int64(200),
		"Request after burst should be rate-limited (expected ~250ms delay)")
}
