package radarr

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"

	"github.com/vido/api/internal/plugins"
)

const testAPIKey = "test-api-key"

func testConfig(url string) plugins.PluginConfig {
	return plugins.PluginConfig{URL: url, APIKey: testAPIKey}
}

// requireAPIKey wraps a handler with the X-Api-Key auth assertion every
// Radarr endpoint must carry.
func requireAPIKey(t *testing.T, next http.HandlerFunc) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, testAPIKey, r.Header.Get("X-Api-Key"))
		next(w, r)
	}
}

func TestNewClient(t *testing.T) {
	client := NewClient(testConfig("http://radarr:7878"))

	require.NotNil(t, client)
	assert.Equal(t, 10*time.Second, client.httpClient.Timeout)
	require.NotNil(t, client.limiter)
	assert.Equal(t, rate.Limit(10), client.limiter.Limit())
	assert.Equal(t, 10, client.limiter.Burst())
	assert.Equal(t, "radarr", client.Name())
}

func TestClient_BuildURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		path     string
		expected string
	}{
		{"basic", "http://192.168.1.100:7878", "/system/status", "http://192.168.1.100:7878/api/v3/system/status"},
		{"trailing slash", "http://radarr:7878/", "/movie", "http://radarr:7878/api/v3/movie"},
		{"https", "https://nas.example.com", "/queue", "https://nas.example.com/api/v3/queue"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(testConfig(tt.url))
			assert.Equal(t, tt.expected, client.buildURL(tt.path))
		})
	}
}

func TestClient_TestConnection_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/system/status", requireAPIKey(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"version": "5.14.0.9383"}`)
	}))
	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(testConfig(server.URL))
	err := client.TestConnection(context.Background(), testConfig(server.URL))
	assert.NoError(t, err)
}

func TestClient_TestConnection_AuthFailed(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/system/status", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(testConfig(server.URL))
	err := client.TestConnection(context.Background(), testConfig(server.URL))

	var pluginErr *plugins.PluginError
	require.ErrorAs(t, err, &pluginErr)
	assert.Equal(t, plugins.ErrCodeAuthFailed, pluginErr.Code)
}

func TestClient_TestConnection_ConnectionFailed(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	serverURL := server.URL
	server.Close() // dead endpoint

	client := NewClient(testConfig(serverURL))
	err := client.TestConnection(context.Background(), testConfig(serverURL))

	var pluginErr *plugins.PluginError
	require.ErrorAs(t, err, &pluginErr)
	assert.Equal(t, plugins.ErrCodeConnectionFailed, pluginErr.Code)
}

func TestClient_TestConnection_UnparseableVersion(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/system/status", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"version": ""}`)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(testConfig(server.URL))
	err := client.TestConnection(context.Background(), testConfig(server.URL))

	var pluginErr *plugins.PluginError
	require.ErrorAs(t, err, &pluginErr)
	assert.Equal(t, plugins.ErrCodeConnectionFailed, pluginErr.Code)
}

func TestClient_AddMovie_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/movie", requireAPIKey(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)

		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, float64(550), body["tmdbId"])
		assert.Equal(t, float64(1), body["qualityProfileId"])
		assert.Equal(t, "/movies", body["rootFolderPath"])
		assert.Equal(t, true, body["monitored"])
		addOpts, ok := body["addOptions"].(map[string]any)
		require.True(t, ok, "addOptions must be an object")
		assert.Equal(t, true, addOpts["searchForMovie"])

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"id": 42, "title": "Fight Club"}`)
	}))
	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(testConfig(server.URL))
	externalID, err := client.AddMovie(context.Background(), 550, plugins.AddOptions{
		QualityProfileID: 1,
		RootFolderPath:   "/movies",
		SearchNow:        true,
	})

	require.NoError(t, err)
	assert.Equal(t, int64(42), externalID)
}

func TestClient_AddMovie_AlreadyExists(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/movie", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `[{"propertyName":"TmdbId","errorMessage":"This movie has already been added"}]`)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(testConfig(server.URL))
	_, err := client.AddMovie(context.Background(), 550, plugins.AddOptions{QualityProfileID: 1, RootFolderPath: "/movies"})

	var pluginErr *plugins.PluginError
	require.ErrorAs(t, err, &pluginErr)
	assert.Equal(t, plugins.ErrCodeAddFailed, pluginErr.Code)
	assert.Contains(t, pluginErr.Message, "This movie has already been added")
}

func TestClient_AddMovie_AuthFailed(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/movie", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(testConfig(server.URL))
	_, err := client.AddMovie(context.Background(), 550, plugins.AddOptions{QualityProfileID: 1, RootFolderPath: "/movies"})

	var pluginErr *plugins.PluginError
	require.ErrorAs(t, err, &pluginErr)
	assert.Equal(t, plugins.ErrCodeAuthFailed, pluginErr.Code)
}

func TestClient_AddSeries_NotSupported(t *testing.T) {
	client := NewClient(testConfig("http://radarr:7878"))
	_, err := client.AddSeries(context.Background(), 1399, plugins.AddOptions{})

	var pluginErr *plugins.PluginError
	require.ErrorAs(t, err, &pluginErr)
	assert.Equal(t, plugins.ErrCodeNotSupported, pluginErr.Code)
}

func TestClient_GetQueue_NormalizesRecords(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/queue", requireAPIKey(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "100", r.URL.Query().Get("pageSize"))
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"page": 1, "pageSize": 100, "totalRecords": 2,
			"records": [
				{"movieId": 42, "title": "Fight Club", "status": "downloading", "size": 8589934592.0, "sizeleft": 4294967296.0, "downloadId": "ABCDEF0123456789"},
				{"movieId": 43, "title": "Se7en", "status": "queued", "size": 4000000000, "sizeleft": 4000000000, "downloadId": "FEDCBA9876543210"}
			]
		}`)
	}))
	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(testConfig(server.URL))
	items, err := client.GetQueue(context.Background())

	require.NoError(t, err)
	require.Len(t, items, 2)
	assert.Equal(t, plugins.QueueItem{
		ExternalID: 42,
		Title:      "Fight Club",
		Status:     "downloading",
		Size:       8589934592,
		SizeLeft:   4294967296,
		DownloadID: "ABCDEF0123456789",
	}, items[0])
	assert.Equal(t, int64(43), items[1].ExternalID)
}

func TestClient_GetQueue_Paginates(t *testing.T) {
	pagesServed := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/queue", func(w http.ResponseWriter, r *http.Request) {
		pagesServed++
		page := r.URL.Query().Get("page")
		w.Header().Set("Content-Type", "application/json")
		if page == "" || page == "1" {
			fmt.Fprint(w, `{"page": 1, "pageSize": 100, "totalRecords": 101, "records": [`+repeatRecord(100)+`]}`)
			return
		}
		fmt.Fprint(w, `{"page": 2, "pageSize": 100, "totalRecords": 101, "records": [{"movieId": 999, "title": "Last", "status": "queued", "size": 1, "sizeleft": 1, "downloadId": "LAST"}]}`)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(testConfig(server.URL))
	items, err := client.GetQueue(context.Background())

	require.NoError(t, err)
	assert.Len(t, items, 101)
	assert.Equal(t, 2, pagesServed)
	assert.Equal(t, int64(999), items[100].ExternalID)
}

// repeatRecord builds n identical queue-record JSON objects.
func repeatRecord(n int) string {
	rec := `{"movieId": 1, "title": "Bulk", "status": "downloading", "size": 10, "sizeleft": 5, "downloadId": "BULK"}`
	out := rec
	for i := 1; i < n; i++ {
		out += "," + rec
	}
	return out
}

func TestClient_GetQueue_AuthFailed(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/queue", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(testConfig(server.URL))
	_, err := client.GetQueue(context.Background())

	var pluginErr *plugins.PluginError
	require.ErrorAs(t, err, &pluginErr)
	assert.Equal(t, plugins.ErrCodeAuthFailed, pluginErr.Code)
}

func TestClient_GetQualityProfiles(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/qualityprofile", requireAPIKey(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[{"id": 1, "name": "HD-1080p"}, {"id": 4, "name": "Ultra-HD"}]`)
	}))
	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(testConfig(server.URL))
	profiles, err := client.GetQualityProfiles(context.Background())

	require.NoError(t, err)
	require.Len(t, profiles, 2)
	assert.Equal(t, plugins.QualityProfile{ID: 1, Name: "HD-1080p"}, profiles[0])
	assert.Equal(t, plugins.QualityProfile{ID: 4, Name: "Ultra-HD"}, profiles[1])
}

func TestClient_GetRootFolders(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/rootfolder", requireAPIKey(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[{"id": 1, "path": "/movies", "freeSpace": 282500063232}]`)
	}))
	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(testConfig(server.URL))
	folders, err := client.GetRootFolders(context.Background())

	require.NoError(t, err)
	require.Len(t, folders, 1)
	assert.Equal(t, plugins.RootFolder{ID: 1, Path: "/movies"}, folders[0])
}

func TestClient_Timeout(t *testing.T) {
	blocked := make(chan struct{})
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/system/status", func(w http.ResponseWriter, r *http.Request) {
		<-blocked // hold the request past the client deadline
	})
	server := httptest.NewServer(mux)
	defer server.Close()
	defer close(blocked)

	client := NewClient(testConfig(server.URL))
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := client.TestConnection(ctx, testConfig(server.URL))

	var pluginErr *plugins.PluginError
	require.ErrorAs(t, err, &pluginErr)
	assert.Equal(t, plugins.ErrCodeTimeout, pluginErr.Code)
	assert.True(t, errors.Is(err, context.DeadlineExceeded) || pluginErr.Cause != nil)
}

func TestClient_ImplementsDVRPlugin(t *testing.T) {
	var _ plugins.DVRPlugin = (*Client)(nil)
}
