package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"golang.org/x/time/rate"

	"github.com/vido/api/internal/secrets"
)

// mockSecretsService implements secrets.SecretsServiceInterface for testing.
type mockSecretsService struct {
	secrets map[string]string
}

func newMockSecrets(kv map[string]string) *mockSecretsService {
	return &mockSecretsService{secrets: kv}
}

func (m *mockSecretsService) Retrieve(_ context.Context, name string) (string, error) {
	v, ok := m.secrets[name]
	if !ok {
		return "", secrets.ErrSecretNotFound
	}
	return v, nil
}

func (m *mockSecretsService) Store(context.Context, string, string) error { return nil }
func (m *mockSecretsService) Delete(context.Context, string) error       { return nil }
func (m *mockSecretsService) Exists(_ context.Context, name string) (bool, error) {
	_, ok := m.secrets[name]
	return ok, nil
}
func (m *mockSecretsService) List(context.Context) ([]string, error) { return nil, nil }

// --- Test helpers ---

func searchResponse(subs []assrtSearchItem) assrtSearchResponse {
	return assrtSearchResponse{
		Status: 0,
		Sub:    &assrtSearchSub{Subs: subs},
	}
}

func detailResponse(subs []assrtDetailItem) assrtDetailResponse {
	return assrtDetailResponse{
		Status: 0,
		Sub:    &assrtDetailSub{Subs: subs},
	}
}

// --- Tests ---

func TestAssrtProvider_Name(t *testing.T) {
	p := NewAssrtProvider(context.Background(), newMockSecrets(map[string]string{
		assrtSecretKey: "test-key",
	}))
	assert.Equal(t, "assrt", p.Name())
}

func TestAssrtProvider_SearchSuccess_NativeName(t *testing.T) {
	// P1-011 fix: verify native_name is used, not name
	items := []assrtSearchItem{
		{
			ID:         12345,
			NativeName: "進擊的巨人 第01話",
			VideoName:  "Attack on Titan S01E01",
			Lang:       "Chn",
			Upload:     "2024-01-15 10:30:00",
		},
		{
			ID:         12346,
			NativeName: "進擊的巨人 第02話",
			VideoName:  "Attack on Titan S01E02",
			Lang:       "Chn",
			Upload:     "2024-01-16 11:00:00",
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/sub/search", r.URL.Path)
		assert.Equal(t, "test-key", r.URL.Query().Get("token"))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(searchResponse(items))
	}))
	defer server.Close()

	p := newTestAssrtProvider("test-key", server.URL)

	results, err := p.Search(context.Background(), SubtitleQuery{Title: "進擊的巨人"})
	require.NoError(t, err)
	require.Len(t, results, 2)

	// Verify P1-011: Filename uses native_name
	assert.Equal(t, "進擊的巨人 第01話", results[0].Filename)
	assert.Equal(t, "12345", results[0].ID)
	assert.Equal(t, "assrt", results[0].Source)
	assert.Equal(t, "Chn", results[0].Language)

	assert.Equal(t, "進擊的巨人 第02話", results[1].Filename)
}

func TestAssrtProvider_SearchDisabled(t *testing.T) {
	// When API key is not configured, provider should return empty, not error
	p := NewAssrtProvider(context.Background(), newMockSecrets(map[string]string{}))

	assert.True(t, p.disabled)

	results, err := p.Search(context.Background(), SubtitleQuery{Title: "test"})
	assert.NoError(t, err)
	assert.Nil(t, results)
}

func TestAssrtProvider_SearchWithYear(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("q")
		assert.Contains(t, q, "2024")

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(searchResponse(nil))
	}))
	defer server.Close()

	p := newTestAssrtProvider("test-key", server.URL)

	results, err := p.Search(context.Background(), SubtitleQuery{Title: "Dune", Year: 2024})
	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestAssrtProvider_SearchHTTPError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
	}{
		{"400 Bad Request", http.StatusBadRequest},
		{"404 Not Found", http.StatusNotFound},
		{"500 Internal Server Error", http.StatusInternalServerError},
		{"503 Service Unavailable", http.StatusServiceUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			p := newTestAssrtProvider("test-key", server.URL)

			results, err := p.Search(context.Background(), SubtitleQuery{Title: "test"})
			assert.Error(t, err)
			assert.Nil(t, results)
			assert.Contains(t, err.Error(), fmt.Sprintf("HTTP %d", tt.statusCode))
		})
	}
}

func TestAssrtProvider_SearchMalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{invalid json`))
	}))
	defer server.Close()

	p := newTestAssrtProvider("test-key", server.URL)

	results, err := p.Search(context.Background(), SubtitleQuery{Title: "test"})
	assert.Error(t, err)
	assert.Nil(t, results)
	assert.Contains(t, err.Error(), "failed to parse")
}

func TestAssrtProvider_SearchEmptySubField(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(assrtSearchResponse{Status: 0, Sub: nil})
	}))
	defer server.Close()

	p := newTestAssrtProvider("test-key", server.URL)

	results, err := p.Search(context.Background(), SubtitleQuery{Title: "nonexistent"})
	assert.NoError(t, err)
	assert.Nil(t, results)
}

func TestAssrtProvider_DownloadSuccess(t *testing.T) {
	subtitleContent := []byte("1\n00:00:01,000 --> 00:00:03,000\n你好世界\n")

	var requestCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&requestCount, 1)

		if count == 1 {
			// Detail request
			assert.Equal(t, "/v1/sub/detail", r.URL.Path)
			assert.Equal(t, "42", r.URL.Query().Get("id"))

			resp := detailResponse([]assrtDetailItem{
				{
					ID: 42,
					Filelist: []assrtDetailFile{
						{URL: fmt.Sprintf("http://%s/download/42.srt", r.Host), Filename: "test.srt", Size: int64(len(subtitleContent))},
					},
				},
			})

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		} else {
			// Download request
			assert.Equal(t, "/download/42.srt", r.URL.Path)
			w.Write(subtitleContent)
		}
	}))
	defer server.Close()

	p := newTestAssrtProvider("test-key", server.URL)

	data, err := p.Download(context.Background(), "42")
	require.NoError(t, err)
	assert.Equal(t, subtitleContent, data)
}

func TestAssrtProvider_DownloadDisabled(t *testing.T) {
	p := NewAssrtProvider(context.Background(), newMockSecrets(map[string]string{}))

	data, err := p.Download(context.Background(), "42")
	assert.Error(t, err)
	assert.Nil(t, data)
	assert.Contains(t, err.Error(), "disabled")
}

func TestAssrtProvider_DownloadNoURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := detailResponse([]assrtDetailItem{
			{ID: 42, URL: "", Filelist: nil},
		})
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := newTestAssrtProvider("test-key", server.URL)

	data, err := p.Download(context.Background(), "42")
	assert.Error(t, err)
	assert.Nil(t, data)
	assert.Contains(t, err.Error(), "no download URL")
}

func TestAssrtProvider_DownloadDetailHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	p := newTestAssrtProvider("test-key", server.URL)

	data, err := p.Download(context.Background(), "42")
	assert.Error(t, err)
	assert.Nil(t, data)
	assert.Contains(t, err.Error(), "HTTP 500")
}

func TestAssrtProvider_DownloadEmptyFile(t *testing.T) {
	var requestCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&requestCount, 1)
		if count == 1 {
			resp := detailResponse([]assrtDetailItem{
				{ID: 42, URL: fmt.Sprintf("http://%s/download/42.srt", r.Host)},
			})
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		} else {
			// Return empty body
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	p := newTestAssrtProvider("test-key", server.URL)

	data, err := p.Download(context.Background(), "42")
	assert.Error(t, err)
	assert.Nil(t, data)
	assert.Contains(t, err.Error(), "empty subtitle file")
}

func TestAssrtProvider_RateLimiter(t *testing.T) {
	var requestCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(searchResponse(nil))
	}))
	defer server.Close()

	p := newTestAssrtProvider("test-key", server.URL)

	// Make 3 rapid requests — should be throttled
	start := time.Now()
	for i := 0; i < 3; i++ {
		_, _ = p.Search(context.Background(), SubtitleQuery{Title: "test"})
	}
	elapsed := time.Since(start)

	// With 2 req/s limit, 3 requests should take at least ~1 second
	// (first goes through immediately, 2nd waits ~500ms, 3rd waits ~500ms)
	assert.GreaterOrEqual(t, elapsed, 800*time.Millisecond, "rate limiter should throttle requests")
	assert.Equal(t, int32(3), atomic.LoadInt32(&requestCount))
}

func TestAssrtProvider_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second) // Simulate slow response
	}))
	defer server.Close()

	p := newTestAssrtProvider("test-key", server.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := p.Search(ctx, SubtitleQuery{Title: "test"})
	assert.Error(t, err)
}

// --- Helper ---

// newTestAssrtProvider creates an AssrtProvider pointing at a test server.
func newTestAssrtProvider(apiKey, baseURL string) *AssrtProvider {
	p := &AssrtProvider{
		apiKey:      apiKey,
		disabled:    false,
		httpClient:  &http.Client{Timeout: 10 * time.Second},
		rateLimiter: rate.NewLimiter(rate.Limit(assrtRateLimit), 1),
	}
	// Override base URL for testing — we patch the const via closure
	origBaseURL := assrtBaseURL
	_ = origBaseURL // suppress unused warning

	// We need to override the URL used in Search/Download. Since it's a const,
	// we'll use a different approach: set a testBaseURL field.
	p.testBaseURL = baseURL + "/v1"
	return p
}
