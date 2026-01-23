package douban

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	t.Run("creates client with default config", func(t *testing.T) {
		config := DefaultConfig()
		client := NewClient(config, nil)

		assert.NotNil(t, client)
		assert.NotNil(t, client.httpClient)
		assert.NotNil(t, client.rateLimiter)
		assert.True(t, client.IsEnabled())
	})

	t.Run("applies default values for zero config", func(t *testing.T) {
		client := NewClient(ClientConfig{}, nil)

		assert.NotNil(t, client)
		// Should have default rate limit of 0.5 req/s
		assert.Equal(t, 0.5, client.config.RequestsPerSecond)
	})

	t.Run("uses custom logger", func(t *testing.T) {
		logger := slog.Default()
		client := NewClient(DefaultConfig(), logger)

		assert.Equal(t, logger, client.logger)
	})
}

func TestClient_GetNextUserAgent(t *testing.T) {
	client := NewClient(DefaultConfig(), nil)

	// Should rotate through all user agents
	seenAgents := make(map[string]bool)
	for i := 0; i < len(defaultUserAgents)*2; i++ {
		ua := client.getNextUserAgent()
		seenAgents[ua] = true
	}

	// Should have seen all user agents
	assert.Equal(t, len(defaultUserAgents), len(seenAgents))
}

func TestClient_AddJitter(t *testing.T) {
	config := DefaultConfig()
	config.JitterMin = 100 * time.Millisecond
	config.JitterMax = 500 * time.Millisecond
	client := NewClient(config, nil)

	baseDuration := 1 * time.Second

	// Run multiple times to check jitter is within bounds
	for i := 0; i < 100; i++ {
		result := client.addJitter(baseDuration)
		minExpected := baseDuration + config.JitterMin
		maxExpected := baseDuration + config.JitterMax

		assert.GreaterOrEqual(t, result, minExpected, "jitter should be >= min")
		assert.LessOrEqual(t, result, maxExpected, "jitter should be <= max")
	}
}

func TestClient_Get_Success(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("<html><body>Test</body></html>"))
	}))
	defer server.Close()

	config := DefaultConfig()
	config.RequestsPerSecond = 100 // High rate for testing
	client := NewClient(config, nil)

	ctx := context.Background()
	resp, err := client.Get(ctx, server.URL)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	// Check metrics
	metrics := client.GetMetrics()
	assert.Equal(t, int64(1), metrics.TotalRequests)
	assert.Equal(t, int64(1), metrics.SuccessfulRequests)
}

func TestClient_Get_Blocked403(t *testing.T) {
	requestCount := int32(0)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("Forbidden"))
	}))
	defer server.Close()

	config := DefaultConfig()
	config.RequestsPerSecond = 1000 // High rate for testing
	config.MaxRetries = 2
	config.InitialBackoff = 10 * time.Millisecond
	config.MaxBackoff = 50 * time.Millisecond
	config.JitterMin = 0
	config.JitterMax = 1 * time.Millisecond
	client := NewClient(config, nil)

	ctx := context.Background()
	_, err := client.Get(ctx, server.URL)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "retries failed")

	// Should have made multiple attempts
	assert.GreaterOrEqual(t, atomic.LoadInt32(&requestCount), int32(2))

	// Check metrics
	metrics := client.GetMetrics()
	assert.Greater(t, metrics.BlockedRequests, int64(0))
}

func TestClient_Get_RateLimited429(t *testing.T) {
	requestCount := int32(0)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&requestCount, 1)
		if count <= 2 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte("Too Many Requests"))
			return
		}
		// Success on third attempt
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("<html></html>"))
	}))
	defer server.Close()

	config := DefaultConfig()
	config.RequestsPerSecond = 1000
	config.MaxRetries = 3
	config.InitialBackoff = 10 * time.Millisecond
	config.MaxBackoff = 50 * time.Millisecond
	config.JitterMin = 0
	config.JitterMax = 1 * time.Millisecond
	client := NewClient(config, nil)

	ctx := context.Background()
	resp, err := client.Get(ctx, server.URL)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	// Should have retried
	assert.Equal(t, int32(3), atomic.LoadInt32(&requestCount))
}

func TestClient_Get_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Sleep longer than timeout
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := ClientConfig{
		Timeout:           50 * time.Millisecond,
		MaxRetries:        0, // No retries for this test
		RequestsPerSecond: 1000,
		InitialBackoff:    10 * time.Millisecond,
		MaxBackoff:        50 * time.Millisecond,
		JitterMin:         0,
		JitterMax:         1 * time.Millisecond,
	}
	client := NewClient(config, nil)

	ctx := context.Background()
	_, err := client.Get(ctx, server.URL)

	require.Error(t, err)
}

func TestClient_GetBody(t *testing.T) {
	expectedBody := "<html><body>Hello Douban</body></html>"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(expectedBody))
	}))
	defer server.Close()

	config := DefaultConfig()
	config.RequestsPerSecond = 1000
	client := NewClient(config, nil)

	ctx := context.Background()
	body, err := client.GetBody(ctx, server.URL)

	require.NoError(t, err)
	assert.Equal(t, expectedBody, body)
}

func TestClient_RateLimiting(t *testing.T) {
	requestCount := int32(0)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := DefaultConfig()
	config.RequestsPerSecond = 10 // 10 requests per second = 1 per 100ms
	client := NewClient(config, nil)

	ctx := context.Background()
	start := time.Now()

	// Make 3 requests
	for i := 0; i < 3; i++ {
		resp, err := client.Get(ctx, server.URL)
		require.NoError(t, err)
		resp.Body.Close()
	}

	elapsed := time.Since(start)

	// Should take at least 200ms for 3 requests at 10/s (first instant, then 2 waits)
	assert.GreaterOrEqual(t, elapsed, 150*time.Millisecond, "rate limiting should slow requests")
}

func TestClient_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(1 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := ClientConfig{
		Timeout:           500 * time.Millisecond,
		MaxRetries:        0, // No retries for this test
		RequestsPerSecond: 1000,
		InitialBackoff:    10 * time.Millisecond,
		MaxBackoff:        50 * time.Millisecond,
		JitterMin:         0,
		JitterMax:         1 * time.Millisecond,
	}
	client := NewClient(config, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := client.Get(ctx, server.URL)

	require.Error(t, err)
}

func TestClient_UserAgentRotation(t *testing.T) {
	seenAgents := make(map[string]int)
	var mu = &struct{ sync.Mutex }{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ua := r.Header.Get("User-Agent")
		mu.Lock()
		seenAgents[ua]++
		mu.Unlock()

		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := DefaultConfig()
	config.RequestsPerSecond = 1000
	client := NewClient(config, nil)

	ctx := context.Background()

	// Make enough requests to see rotation
	for i := 0; i < len(defaultUserAgents)*2; i++ {
		resp, err := client.Get(ctx, server.URL)
		require.NoError(t, err)
		resp.Body.Close()
	}

	// Should have seen multiple user agents
	assert.Greater(t, len(seenAgents), 1, "should rotate user agents")
}

func TestSearchURL(t *testing.T) {
	tests := []struct {
		name      string
		query     string
		mediaType MediaType
		wantContains []string
	}{
		{
			name:      "basic search",
			query:     "寄生上流",
			mediaType: MediaTypeMovie,
			wantContains: []string{
				"https://search.douban.com/movie/subject_search",
				"search_text=",
				"cat=1002",
			},
		},
		{
			name:      "search with spaces",
			query:     "The Matrix",
			mediaType: MediaTypeMovie,
			wantContains: []string{
				"search_text=The+Matrix",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := SearchURL(tt.query, tt.mediaType)
			for _, substr := range tt.wantContains {
				assert.True(t, strings.Contains(url, substr),
					"URL should contain %q, got %q", substr, url)
			}
		})
	}
}

func TestDetailURL(t *testing.T) {
	tests := []struct {
		id   string
		want string
	}{
		{"1292052", "https://movie.douban.com/subject/1292052/"},
		{"27010768", "https://movie.douban.com/subject/27010768/"},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			got := DetailURL(tt.id)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestClient_SetEnabled(t *testing.T) {
	client := NewClient(DefaultConfig(), nil)

	assert.True(t, client.IsEnabled())

	client.SetEnabled(false)
	assert.False(t, client.IsEnabled())

	client.SetEnabled(true)
	assert.True(t, client.IsEnabled())
}

func TestClientMetrics(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := DefaultConfig()
	config.RequestsPerSecond = 1000
	client := NewClient(config, nil)

	ctx := context.Background()

	// Initial metrics
	metrics := client.GetMetrics()
	assert.Equal(t, int64(0), metrics.TotalRequests)

	// Make a request
	resp, err := client.Get(ctx, server.URL)
	require.NoError(t, err)
	resp.Body.Close()

	// Check updated metrics
	metrics = client.GetMetrics()
	assert.Equal(t, int64(1), metrics.TotalRequests)
	assert.Equal(t, int64(1), metrics.SuccessfulRequests)
	assert.False(t, metrics.LastRequestTime.IsZero())
}

func TestBlockedError(t *testing.T) {
	err := &BlockedError{
		StatusCode: 403,
		Reason:     "forbidden",
	}

	assert.Contains(t, err.Error(), "blocked")
	assert.Contains(t, err.Error(), "forbidden")
	assert.True(t, err.IsBlocked())
}

func TestParseError(t *testing.T) {
	err := &ParseError{
		Field:  "title",
		Reason: "element not found",
	}

	assert.Contains(t, err.Error(), "title")
	assert.Contains(t, err.Error(), "element not found")
}

func TestNotFoundError(t *testing.T) {
	t.Run("with query", func(t *testing.T) {
		err := &NotFoundError{Query: "test movie"}
		assert.Contains(t, err.Error(), "test movie")
	})

	t.Run("with ID", func(t *testing.T) {
		err := &NotFoundError{ID: "12345"}
		assert.Contains(t, err.Error(), "12345")
	})
}
