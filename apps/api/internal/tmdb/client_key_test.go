package tmdb

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// sub-7-7a — the client asks its KeyProvider on EVERY request, so a key saved
// in the settings page is what the next call sends. Before this the key was
// frozen into the struct at boot (backlog-tmdb-runtime-key-resolution).

const okBody = `{"page": 1, "results": [], "total_pages": 0, "total_results": 0}`

func newRecordingServer(t *testing.T, status int, body string) (*httptest.Server, *[]string) {
	t.Helper()
	var seenKeys []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenKeys = append(seenKeys, r.URL.Query().Get("api_key"))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	return server, &seenKeys
}

func TestClient_KeyProvider_IsAskedPerRequest(t *testing.T) {
	server, seen := newRecordingServer(t, http.StatusOK, okBody)

	var current atomic.Value
	current.Store("key-at-boot")
	client := NewClient(ClientConfig{
		BaseURL: server.URL,
		KeyProvider: func(context.Context) (string, error) {
			return current.Load().(string), nil
		},
	})

	var result SearchResultMovies
	require.NoError(t, client.Get(context.Background(), "/search/movie", nil, &result))

	// The "user saved a key in the settings page" moment — no rebuild, no restart.
	current.Store("key-saved-later")
	require.NoError(t, client.Get(context.Background(), "/search/movie", nil, &result))

	assert.Equal(t, []string{"key-at-boot", "key-saved-later"}, *seen)
}

func TestClient_KeyProvider_StaticKeyStillWorksWithoutProvider(t *testing.T) {
	server, seen := newRecordingServer(t, http.StatusOK, okBody)
	client := NewClient(ClientConfig{APIKey: "static-key", BaseURL: server.URL})

	var result SearchResultMovies
	require.NoError(t, client.Get(context.Background(), "/search/movie", nil, &result))

	assert.Equal(t, []string{"static-key"}, *seen)
}

// No key anywhere → fail locally as "not configured". The status page keys on
// that wording (models.isUnconfiguredError) to show 未設定 instead of an outage,
// and no request is sent, so no rate-limit token is spent.
func TestClient_KeyProvider_EmptyKeyFailsBeforeTheNetwork(t *testing.T) {
	server, seen := newRecordingServer(t, http.StatusOK, okBody)
	var observed error
	client := NewClient(ClientConfig{
		BaseURL:     server.URL,
		KeyProvider: func(context.Context) (string, error) { return "   ", nil },
		Observer:    func(_ string, err error) { observed = err },
	})

	var result SearchResultMovies
	err := client.Get(context.Background(), "/search/movie", nil, &result)

	var tmdbErr *TMDbError
	require.ErrorAs(t, err, &tmdbErr)
	assert.Equal(t, ErrCodeUnauthorized, tmdbErr.Code)
	assert.Contains(t, strings.ToLower(err.Error()), "not configured")
	assert.Empty(t, *seen, "a keyless install must not hit TMDb at all")
	assert.Same(t, tmdbErr, observed, "the observer is told, so the status page can say 未設定")
}

func TestClient_KeyProvider_ErrorIsSurfaced(t *testing.T) {
	server, seen := newRecordingServer(t, http.StatusOK, okBody)
	boom := errors.New("secrets store is on fire")
	client := NewClient(ClientConfig{
		BaseURL:     server.URL,
		KeyProvider: func(context.Context) (string, error) { return "", boom },
	})

	var result SearchResultMovies
	err := client.Get(context.Background(), "/search/movie", nil, &result)

	require.ErrorIs(t, err, boom)
	assert.Empty(t, *seen)
}

// The transport error text carries the request URL, api_key and all. With a
// bundled key that would print the owner's key into every user's log, so the
// key is scrubbed while the error chain (context / net errors) is preserved.
func TestClient_TransportError_DoesNotLeakTheKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	deadURL := server.URL
	server.Close() // connection refused from here on

	const secret = "s3cr3t-bundled-key"
	client := NewClient(ClientConfig{APIKey: secret, BaseURL: deadURL})

	var result SearchResultMovies
	err := client.Get(context.Background(), "/search/movie", nil, &result)

	require.Error(t, err)
	assert.NotContains(t, err.Error(), secret)
	assert.Contains(t, err.Error(), "REDACTED")
	assert.Contains(t, err.Error(), "HTTP request failed")
}

func TestClient_TransportError_KeepsTheErrorChain(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	deadURL := server.URL
	server.Close()

	const secret = "chain-key"
	client := NewClient(ClientConfig{APIKey: secret, BaseURL: deadURL})

	var result SearchResultMovies
	err := client.Get(context.Background(), "/search/movie", nil, &result)

	// Redaction rewrites the URL INSIDE the *url.Error; it must not flatten
	// the chain into a string, or callers lose errors.As/Is on net errors.
	require.Error(t, err)
	var ue *url.Error
	require.True(t, errors.As(err, &ue), err.Error())
	assert.Equal(t, "Get", ue.Op)
	assert.Contains(t, ue.URL, "api_key=REDACTED")
	assert.NotContains(t, ue.URL, secret)
}

// The observer is the health monitor's window onto real traffic (AC #4): nil
// on success, the *TMDbError on an API error — a 429 in particular.
func TestClient_Observer_SeesSuccessAndRateLimit(t *testing.T) {
	var status atomic.Int32
	status.Store(http.StatusOK)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		code := int(status.Load())
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		if code == http.StatusOK {
			_, _ = w.Write([]byte(okBody))
			return
		}
		_, _ = w.Write([]byte(`{"status_code":25,"status_message":"Your request count is over the allowed limit."}`))
	}))
	t.Cleanup(server.Close)

	var (
		endpoints []string
		outcomes  []error
	)
	client := NewClient(ClientConfig{APIKey: "k", BaseURL: server.URL})
	client.SetObserver(func(endpoint string, err error) {
		endpoints = append(endpoints, endpoint)
		outcomes = append(outcomes, err)
	})

	var result SearchResultMovies
	require.NoError(t, client.Get(context.Background(), "/search/movie", nil, &result))

	status.Store(http.StatusTooManyRequests)
	err := client.Get(context.Background(), "/search/movie", nil, &result)
	require.Error(t, err)

	require.Len(t, outcomes, 2)
	assert.Equal(t, []string{"/search/movie", "/search/movie"}, endpoints)
	assert.NoError(t, outcomes[0])
	var tmdbErr *TMDbError
	require.ErrorAs(t, outcomes[1], &tmdbErr)
	assert.Equal(t, ErrCodeRateLimitExceeded, tmdbErr.Code)
}

func TestClient_Observer_NilIsSafe(t *testing.T) {
	server, _ := newRecordingServer(t, http.StatusOK, okBody)
	client := NewClient(ClientConfig{APIKey: "k", BaseURL: server.URL})
	client.SetObserver(nil)

	var result SearchResultMovies
	assert.NoError(t, client.Get(context.Background(), "/search/movie", nil, &result))
}
