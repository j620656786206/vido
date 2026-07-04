package sonarr

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

// staticResolver returns a fixed TVDB id (0 = not on TVDB).
func staticResolver(tvdbID int64, err error) TVDBResolver {
	return TVDBResolverFunc(func(ctx context.Context, tmdbID int64) (int64, error) {
		return tvdbID, err
	})
}

func requireAPIKey(t *testing.T, next http.HandlerFunc) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, testAPIKey, r.Header.Get("X-Api-Key"))
		next(w, r)
	}
}

func TestNewClient(t *testing.T) {
	client := NewClient(testConfig("http://sonarr:8989"), staticResolver(121361, nil))

	require.NotNil(t, client)
	assert.Equal(t, 10*time.Second, client.httpClient.Timeout)
	require.NotNil(t, client.limiter)
	assert.Equal(t, rate.Limit(10), client.limiter.Limit())
	assert.Equal(t, 10, client.limiter.Burst())
	assert.Equal(t, "sonarr", client.Name())
}

func TestClient_TestConnection_V4Pass(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/system/status", requireAPIKey(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"version": "4.0.10.2544"}`)
	}))
	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(testConfig(server.URL), staticResolver(0, nil))
	assert.NoError(t, client.TestConnection(context.Background(), testConfig(server.URL)))
}

func TestClient_TestConnection_V3Rejected(t *testing.T) {
	// AC #2 version gate: Sonarr v3 requires languageProfileId on adds —
	// fail the connection test loudly instead of half-working.
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/system/status", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"version": "3.0.10.1567"}`)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(testConfig(server.URL), staticResolver(0, nil))
	err := client.TestConnection(context.Background(), testConfig(server.URL))

	var pluginErr *plugins.PluginError
	require.ErrorAs(t, err, &pluginErr)
	assert.Equal(t, plugins.ErrCodeTestFailed, pluginErr.Code)
	assert.Contains(t, pluginErr.Message, "需要 Sonarr v4")
}

func TestClient_TestConnection_AuthFailed(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/system/status", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(testConfig(server.URL), staticResolver(0, nil))
	err := client.TestConnection(context.Background(), testConfig(server.URL))

	var pluginErr *plugins.PluginError
	require.ErrorAs(t, err, &pluginErr)
	assert.Equal(t, plugins.ErrCodeAuthFailed, pluginErr.Code)
}

func TestClient_AddMovie_NotSupported(t *testing.T) {
	client := NewClient(testConfig("http://sonarr:8989"), staticResolver(0, nil))
	_, err := client.AddMovie(context.Background(), 550, plugins.AddOptions{})

	var pluginErr *plugins.PluginError
	require.ErrorAs(t, err, &pluginErr)
	assert.Equal(t, plugins.ErrCodeNotSupported, pluginErr.Code)
}

func TestClient_AddSeries_Success(t *testing.T) {
	lookupCalled := false
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/series/lookup", requireAPIKey(t, func(w http.ResponseWriter, r *http.Request) {
		lookupCalled = true
		assert.Equal(t, "tvdb:121361", r.URL.Query().Get("term"))
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[{
			"title": "Game of Thrones",
			"titleSlug": "game-of-thrones",
			"tvdbId": 121361,
			"images": [{"coverType": "poster", "remoteUrl": "http://x/poster.jpg"}],
			"seasons": [
				{"seasonNumber": 0, "monitored": false},
				{"seasonNumber": 1, "monitored": false},
				{"seasonNumber": 2, "monitored": false}
			]
		}]`)
	}))
	mux.HandleFunc("/api/v3/series", requireAPIKey(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)

		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))

		// The POST body is the lookup object enriched with config fields —
		// hand-building a minimal body is the classic Sonarr-400 source.
		assert.Equal(t, "Game of Thrones", body["title"])
		assert.Equal(t, float64(121361), body["tvdbId"])
		assert.Equal(t, float64(4), body["qualityProfileId"])
		assert.Equal(t, "/tv", body["rootFolderPath"])
		assert.Equal(t, true, body["monitored"])

		seasons, ok := body["seasons"].([]any)
		require.True(t, ok, "seasons array from the lookup object must be present")
		require.Len(t, seasons, 3)
		for _, s := range seasons {
			assert.Equal(t, true, s.(map[string]any)["monitored"], "whole-series add monitors every season")
		}

		addOpts, ok := body["addOptions"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "all", addOpts["monitor"])
		assert.Equal(t, true, addOpts["searchForMissingEpisodes"])

		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"id": 7, "title": "Game of Thrones"}`)
	}))
	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(testConfig(server.URL), staticResolver(121361, nil))
	externalID, err := client.AddSeries(context.Background(), 1399, plugins.AddOptions{
		QualityProfileID: 4,
		RootFolderPath:   "/tv",
		SearchNow:        true,
	})

	require.NoError(t, err)
	assert.Equal(t, int64(7), externalID)
	assert.True(t, lookupCalled)
}

func TestClient_AddSeries_NoTVDBEntry(t *testing.T) {
	// AC #1.2 — no TVDB id = Sonarr fundamental limitation, typed terminal error.
	client := NewClient(testConfig("http://sonarr:8989"), staticResolver(0, nil))
	_, err := client.AddSeries(context.Background(), 250551, plugins.AddOptions{QualityProfileID: 1, RootFolderPath: "/tv"})

	var pluginErr *plugins.PluginError
	require.ErrorAs(t, err, &pluginErr)
	assert.Equal(t, plugins.ErrCodeTVDBNotFound, pluginErr.Code)
}

func TestClient_AddSeries_ResolverError(t *testing.T) {
	client := NewClient(testConfig("http://sonarr:8989"), staticResolver(0, errors.New("tmdb down")))
	_, err := client.AddSeries(context.Background(), 1399, plugins.AddOptions{QualityProfileID: 1, RootFolderPath: "/tv"})

	var pluginErr *plugins.PluginError
	require.ErrorAs(t, err, &pluginErr)
	assert.Equal(t, plugins.ErrCodeConnectionFailed, pluginErr.Code)
}

func TestClient_AddSeries_EmptyLookup(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/series/lookup", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[]`)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(testConfig(server.URL), staticResolver(121361, nil))
	_, err := client.AddSeries(context.Background(), 1399, plugins.AddOptions{QualityProfileID: 1, RootFolderPath: "/tv"})

	var pluginErr *plugins.PluginError
	require.ErrorAs(t, err, &pluginErr)
	assert.Equal(t, plugins.ErrCodeAddFailed, pluginErr.Code)
}

func TestClient_AddSeries_AlreadyExists(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/series/lookup", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[{"title": "GoT", "tvdbId": 121361, "seasons": []}]`)
	})
	mux.HandleFunc("/api/v3/series", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `[{"errorMessage": "This series has already been added"}]`)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(testConfig(server.URL), staticResolver(121361, nil))
	_, err := client.AddSeries(context.Background(), 1399, plugins.AddOptions{QualityProfileID: 1, RootFolderPath: "/tv"})

	var pluginErr *plugins.PluginError
	require.ErrorAs(t, err, &pluginErr)
	assert.Equal(t, plugins.ErrCodeAddFailed, pluginErr.Code)
	assert.Contains(t, pluginErr.Message, "already been added")
}

func TestClient_GetQueue_NormalizesSeriesRecords(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/queue", requireAPIKey(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "100", r.URL.Query().Get("pageSize"))
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"page": 1, "pageSize": 100, "totalRecords": 1,
			"records": [
				{"seriesId": 7, "title": "Game of Thrones S01E01", "status": "downloading", "size": 2147483648.0, "sizeleft": 1073741824.0, "downloadId": "HASH01"}
			]
		}`)
	}))
	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(testConfig(server.URL), staticResolver(0, nil))
	items, err := client.GetQueue(context.Background())

	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, plugins.QueueItem{
		ExternalID: 7,
		Title:      "Game of Thrones S01E01",
		Status:     "downloading",
		Size:       2147483648,
		SizeLeft:   1073741824,
		DownloadID: "HASH01",
	}, items[0])
}

func TestClient_GetQueue_Paginates(t *testing.T) {
	// 13-4b CR L1 — radarr-test parity: the pagination loop is live logic.
	pagesServed := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/queue", func(w http.ResponseWriter, r *http.Request) {
		pagesServed++
		page := r.URL.Query().Get("page")
		w.Header().Set("Content-Type", "application/json")
		if page == "" || page == "1" {
			fmt.Fprint(w, `{"page": 1, "pageSize": 100, "totalRecords": 101, "records": [`+repeatQueueRecord(100)+`]}`)
			return
		}
		fmt.Fprint(w, `{"page": 2, "pageSize": 100, "totalRecords": 101, "records": [{"seriesId": 999, "title": "Last", "status": "queued", "size": 1, "sizeleft": 1, "downloadId": "LAST"}]}`)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(testConfig(server.URL), staticResolver(0, nil))
	items, err := client.GetQueue(context.Background())

	require.NoError(t, err)
	assert.Len(t, items, 101)
	assert.Equal(t, 2, pagesServed)
	assert.Equal(t, int64(999), items[100].ExternalID)
}

// repeatQueueRecord builds n identical queue-record JSON objects.
func repeatQueueRecord(n int) string {
	rec := `{"seriesId": 7, "title": "Bulk", "status": "downloading", "size": 10, "sizeleft": 5, "downloadId": "BULK"}`
	out := rec
	for i := 1; i < n; i++ {
		out += "," + rec
	}
	return out
}

func TestClient_GetQualityProfilesAndRootFolders(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/qualityprofile", requireAPIKey(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[{"id": 6, "name": "HD-1080p"}]`)
	}))
	mux.HandleFunc("/api/v3/rootfolder", requireAPIKey(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[{"id": 1, "path": "/tv"}]`)
	}))
	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(testConfig(server.URL), staticResolver(0, nil))

	profiles, err := client.GetQualityProfiles(context.Background())
	require.NoError(t, err)
	assert.Equal(t, []plugins.QualityProfile{{ID: 6, Name: "HD-1080p"}}, profiles)

	folders, err := client.GetRootFolders(context.Background())
	require.NoError(t, err)
	assert.Equal(t, []plugins.RootFolder{{ID: 1, Path: "/tv"}}, folders)
}

func TestClient_ImplementsInterfaces(t *testing.T) {
	var _ plugins.DVRPlugin = (*Client)(nil)
	var _ plugins.ProfileLister = (*Client)(nil)
}
