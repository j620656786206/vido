package qbittorrent

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMapQBState(t *testing.T) {
	tests := []struct {
		state    string
		expected TorrentStatus
	}{
		// Downloading: actively transferring or allocating
		{"downloading", StatusDownloading},
		{"forcedDL", StatusDownloading},
		{"metaDL", StatusDownloading},
		{"allocating", StatusDownloading},
		// Paused: download incomplete, user stopped (qBT 4.x + 5.0+)
		{"pausedDL", StatusPaused},
		{"stoppedDL", StatusPaused},
		// Completed: download finished, not actively seeding (qBT 4.x + 5.0+)
		{"pausedUP", StatusCompleted},
		{"stoppedUP", StatusCompleted},
		{"stalledUP", StatusCompleted},
		// Seeding: actively uploading to peers
		{"uploading", StatusSeeding},
		{"forcedUP", StatusSeeding},
		// Stalled: download in progress but no peers
		{"stalledDL", StatusStalled},
		// Queued
		{"queuedDL", StatusQueued},
		{"queuedUP", StatusQueued},
		// Checking
		{"checkingDL", StatusChecking},
		{"checkingUP", StatusChecking},
		{"checkingResumeData", StatusChecking},
		// Moving (qBT 5.0+)
		{"moving", StatusChecking},
		// Error
		{"error", StatusError},
		{"missingFiles", StatusError},
		// Unknown defaults to downloading
		{"unknownState", StatusDownloading},
		{"", StatusDownloading},
	}

	for _, tt := range tests {
		t.Run(tt.state, func(t *testing.T) {
			result := MapQBState(tt.state)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTorrentStatus_Constants(t *testing.T) {
	assert.Equal(t, TorrentStatus("downloading"), StatusDownloading)
	assert.Equal(t, TorrentStatus("paused"), StatusPaused)
	assert.Equal(t, TorrentStatus("seeding"), StatusSeeding)
	assert.Equal(t, TorrentStatus("completed"), StatusCompleted)
	assert.Equal(t, TorrentStatus("stalled"), StatusStalled)
	assert.Equal(t, TorrentStatus("error"), StatusError)
	assert.Equal(t, TorrentStatus("queued"), StatusQueued)
	assert.Equal(t, TorrentStatus("checking"), StatusChecking)
}

func TestTorrentsSort_Constants(t *testing.T) {
	assert.Equal(t, TorrentsSort("added_on"), SortAddedOn)
	assert.Equal(t, TorrentsSort("name"), SortName)
	assert.Equal(t, TorrentsSort("progress"), SortProgress)
	assert.Equal(t, TorrentsSort("size"), SortSize)
	assert.Equal(t, TorrentsSort("status"), SortStatus)
}

func TestTorrent_JSONSerialization(t *testing.T) {
	addedOn := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	completedOn := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)

	torrent := Torrent{
		Hash:          "abc123",
		Name:          "Test Movie [1080p]",
		Size:          4294967296,
		Progress:      0.85,
		DownloadSpeed: 10485760,
		UploadSpeed:   524288,
		ETA:           600,
		Status:        StatusDownloading,
		AddedOn:       addedOn,
		CompletedOn:   &completedOn,
		Seeds:         10,
		Peers:         5,
		Downloaded:    3650722201,
		Uploaded:      104857600,
		Ratio:         0.03,
		SavePath:      "/downloads/movies",
	}

	data, err := json.Marshal(torrent)
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)

	assert.Equal(t, "abc123", result["hash"])
	assert.Equal(t, "Test Movie [1080p]", result["name"])
	assert.Equal(t, "downloading", result["status"])
	assert.Equal(t, float64(10485760), result["download_speed"])
	assert.Equal(t, "/downloads/movies", result["save_path"])
	assert.Contains(t, result, "completed_on")
}

func TestTorrent_JSONSerialization_NilCompletedOn(t *testing.T) {
	torrent := Torrent{
		Hash:    "abc123",
		Name:    "Test Movie",
		Status:  StatusDownloading,
		AddedOn: time.Now().UTC(),
	}

	data, err := json.Marshal(torrent)
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)

	assert.NotContains(t, result, "completed_on", "completedOn should be omitted when nil")
}

func TestTorrentDetails_JSONSerialization(t *testing.T) {
	addedOn := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)

	details := TorrentDetails{
		Torrent: Torrent{
			Hash:    "abc123",
			Name:    "Test Movie",
			Status:  StatusCompleted,
			AddedOn: addedOn,
		},
		PieceSize:    4194304,
		Comment:      "Test comment",
		CreatedBy:    "qBittorrent",
		CreationDate: addedOn,
		TotalWasted:  0,
		TimeElapsed:  3600,
		SeedingTime:  1800,
		AvgDownSpeed: 8388608,
		AvgUpSpeed:   262144,
	}

	data, err := json.Marshal(details)
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)

	assert.Equal(t, "abc123", result["hash"])
	assert.Equal(t, float64(4194304), result["piece_size"])
	assert.Equal(t, "Test comment", result["comment"])
	assert.Equal(t, float64(3600), result["time_elapsed"])
}

func TestMapQBTorrentInfo(t *testing.T) {
	qbt := qbTorrentInfo{
		Hash:         "8c212779b4abde7c6bc608571a69eb3a9ec4c28c",
		Name:         "[SubGroup] Movie Name (2024) [1080p]",
		Size:         4294967296,
		Progress:     0.85,
		DLSpeed:      10485760,
		UPSpeed:      524288,
		ETA:          600,
		State:        "downloading",
		AddedOn:      1704067200,
		CompletionOn: 0,
		NumSeeds:     10,
		NumLeechs:    5,
		SavePath:     "/downloads/movies",
		Downloaded:   3650722201,
		Uploaded:     104857600,
		Ratio:        0.03,
	}

	torrent := mapQBTorrentInfo(qbt)

	assert.Equal(t, qbt.Hash, torrent.Hash)
	assert.Equal(t, qbt.Name, torrent.Name)
	assert.Equal(t, qbt.Size, torrent.Size)
	assert.Equal(t, qbt.Progress, torrent.Progress)
	assert.Equal(t, qbt.DLSpeed, torrent.DownloadSpeed)
	assert.Equal(t, qbt.UPSpeed, torrent.UploadSpeed)
	assert.Equal(t, qbt.ETA, torrent.ETA)
	assert.Equal(t, StatusDownloading, torrent.Status)
	assert.Equal(t, time.Unix(1704067200, 0).UTC(), torrent.AddedOn)
	assert.Nil(t, torrent.CompletedOn, "completedOn should be nil when completion_on is 0")
	assert.Equal(t, qbt.NumSeeds, torrent.Seeds)
	assert.Equal(t, qbt.NumLeechs, torrent.Peers)
	assert.Equal(t, qbt.Downloaded, torrent.Downloaded)
	assert.Equal(t, qbt.Uploaded, torrent.Uploaded)
	assert.Equal(t, qbt.Ratio, torrent.Ratio)
	assert.Equal(t, qbt.SavePath, torrent.SavePath)
}

func TestMapQBTorrentInfo_WithCompletionOn(t *testing.T) {
	qbt := qbTorrentInfo{
		Hash:         "abc123",
		Name:         "Completed Movie",
		State:        "stalledUP",
		CompletionOn: 1704153600,
	}

	torrent := mapQBTorrentInfo(qbt)

	assert.Equal(t, StatusCompleted, torrent.Status)
	require.NotNil(t, torrent.CompletedOn)
	assert.Equal(t, time.Unix(1704153600, 0).UTC(), *torrent.CompletedOn)
}

func TestMapTorrentDetails(t *testing.T) {
	torrent := &Torrent{
		Hash:   "abc123",
		Name:   "Test Movie",
		Status: StatusCompleted,
	}

	props := qbTorrentProperties{
		PieceSize:    4194304,
		Comment:      "Test comment",
		CreatedBy:    "qBittorrent",
		CreationDate: 1704067200,
		TotalWasted:  1024,
		TimeElapsed:  3600,
		SeedingTime:  1800,
		DLSpeedAvg:   8388608,
		UPSpeedAvg:   262144,
	}

	details := mapTorrentDetails(torrent, props)

	assert.Equal(t, "abc123", details.Hash)
	assert.Equal(t, "Test Movie", details.Name)
	assert.Equal(t, StatusCompleted, details.Status)
	assert.Equal(t, int64(4194304), details.PieceSize)
	assert.Equal(t, "Test comment", details.Comment)
	assert.Equal(t, "qBittorrent", details.CreatedBy)
	assert.Equal(t, time.Unix(1704067200, 0).UTC(), details.CreationDate)
	assert.Equal(t, int64(1024), details.TotalWasted)
	assert.Equal(t, int64(3600), details.TimeElapsed)
	assert.Equal(t, int64(1800), details.SeedingTime)
	assert.Equal(t, int64(8388608), details.AvgDownSpeed)
	assert.Equal(t, int64(262144), details.AvgUpSpeed)
}

func TestErrCodeTorrentNotFound(t *testing.T) {
	assert.Equal(t, "QBITTORRENT_TORRENT_NOT_FOUND", ErrCodeTorrentNotFound)
}

// --- Client torrent method tests ---

// mockTorrentInfoJSON returns a JSON array of mock torrent info objects.
func mockTorrentInfoJSON() string {
	return `[
		{
			"hash": "abc123def456",
			"name": "[SubGroup] Movie Name (2024) [1080p]",
			"size": 4294967296,
			"progress": 0.85,
			"dlspeed": 10485760,
			"upspeed": 524288,
			"eta": 600,
			"state": "downloading",
			"added_on": 1704067200,
			"completion_on": 0,
			"num_seeds": 10,
			"num_leechs": 5,
			"save_path": "/downloads/movies",
			"downloaded": 3650722201,
			"uploaded": 104857600,
			"ratio": 0.03
		},
		{
			"hash": "xyz789ghi012",
			"name": "Completed Series S01",
			"size": 8589934592,
			"progress": 1.0,
			"dlspeed": 0,
			"upspeed": 262144,
			"eta": 8640000,
			"state": "stalledUP",
			"added_on": 1704000000,
			"completion_on": 1704060000,
			"num_seeds": 20,
			"num_leechs": 3,
			"save_path": "/downloads/series",
			"downloaded": 8589934592,
			"uploaded": 1073741824,
			"ratio": 0.125
		}
	]`
}

func mockTorrentPropertiesJSON() string {
	return `{
		"save_path": "/downloads/movies",
		"creation_date": 1704067200,
		"piece_size": 4194304,
		"comment": "Downloaded via torrent",
		"total_wasted": 1024,
		"total_uploaded": 104857600,
		"total_downloaded": 3650722201,
		"peers": 5,
		"seeds": 10,
		"addition_date": 1704067200,
		"completion_date": -1,
		"created_by": "qBittorrent v4.5.2",
		"dl_speed_avg": 8388608,
		"up_speed_avg": 262144,
		"time_elapsed": 3600,
		"seeding_time": 0
	}`
}

func setupTorrentTestServer(t *testing.T) *http.ServeMux {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/auth/login", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "SID", Value: "test-session"})
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "Ok.")
	})
	return mux
}

func TestClient_GetTorrents_Success(t *testing.T) {
	mux := setupTorrentTestServer(t)
	mux.HandleFunc("/api/v2/torrents/info", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, mockTorrentInfoJSON())
	})

	server := newTestServer(t, mux)
	defer server.Close()

	client := NewClient(&Config{
		Host:     server.URL,
		Username: "admin",
		Password: "password",
	})

	torrents, err := client.GetTorrents(context.Background(), nil)
	require.NoError(t, err)
	require.Len(t, torrents, 2)

	// First torrent - downloading
	assert.Equal(t, "abc123def456", torrents[0].Hash)
	assert.Equal(t, "[SubGroup] Movie Name (2024) [1080p]", torrents[0].Name)
	assert.Equal(t, int64(4294967296), torrents[0].Size)
	assert.Equal(t, 0.85, torrents[0].Progress)
	assert.Equal(t, int64(10485760), torrents[0].DownloadSpeed)
	assert.Equal(t, StatusDownloading, torrents[0].Status)
	assert.Nil(t, torrents[0].CompletedOn)

	// Second torrent - completed
	assert.Equal(t, "xyz789ghi012", torrents[1].Hash)
	assert.Equal(t, StatusCompleted, torrents[1].Status)
	require.NotNil(t, torrents[1].CompletedOn)
}

func TestClient_GetTorrents_WithSortOptions(t *testing.T) {
	mux := setupTorrentTestServer(t)
	mux.HandleFunc("/api/v2/torrents/info", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "name", r.URL.Query().Get("sort"))
		assert.Equal(t, "true", r.URL.Query().Get("reverse"))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "[]")
	})

	server := newTestServer(t, mux)
	defer server.Close()

	client := NewClient(&Config{
		Host:     server.URL,
		Username: "admin",
		Password: "password",
	})

	opts := &ListTorrentsOptions{
		Sort:    SortName,
		Reverse: true,
	}
	torrents, err := client.GetTorrents(context.Background(), opts)
	require.NoError(t, err)
	assert.Empty(t, torrents)
}

func TestClient_GetTorrents_EmptyList(t *testing.T) {
	mux := setupTorrentTestServer(t)
	mux.HandleFunc("/api/v2/torrents/info", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "[]")
	})

	server := newTestServer(t, mux)
	defer server.Close()

	client := NewClient(&Config{
		Host:     server.URL,
		Username: "admin",
		Password: "password",
	})

	torrents, err := client.GetTorrents(context.Background(), nil)
	require.NoError(t, err)
	assert.Empty(t, torrents)
}

func TestClient_GetTorrents_AuthFailure(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/auth/login", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "Fails.")
	})

	server := newTestServer(t, mux)
	defer server.Close()

	client := NewClient(&Config{
		Host:     server.URL,
		Username: "admin",
		Password: "wrong",
	})

	torrents, err := client.GetTorrents(context.Background(), nil)
	assert.Nil(t, torrents)
	assert.Error(t, err)

	var connErr *ConnectionError
	assert.ErrorAs(t, err, &connErr)
	assert.Equal(t, ErrCodeAuthFailed, connErr.Code)
}

func TestClient_GetTorrents_ServerError(t *testing.T) {
	mux := setupTorrentTestServer(t)
	mux.HandleFunc("/api/v2/torrents/info", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	server := newTestServer(t, mux)
	defer server.Close()

	client := NewClient(&Config{
		Host:     server.URL,
		Username: "admin",
		Password: "password",
	})

	torrents, err := client.GetTorrents(context.Background(), nil)
	assert.Nil(t, torrents)
	assert.Error(t, err)

	var connErr *ConnectionError
	assert.ErrorAs(t, err, &connErr)
	assert.Equal(t, ErrCodeConnectionFailed, connErr.Code)
}

func TestClient_GetTorrentDetails_Success(t *testing.T) {
	mux := setupTorrentTestServer(t)
	mux.HandleFunc("/api/v2/torrents/info", func(w http.ResponseWriter, r *http.Request) {
		// Verify hashes filter is used for detail requests
		hashes := r.URL.Query().Get("hashes")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if hashes == "abc123def456" {
			// Return only the matching torrent
			fmt.Fprint(w, `[`+`{
				"hash": "abc123def456",
				"name": "[SubGroup] Movie Name (2024) [1080p]",
				"size": 4294967296,
				"progress": 0.85,
				"dlspeed": 10485760,
				"upspeed": 524288,
				"eta": 600,
				"state": "downloading",
				"added_on": 1704067200,
				"completion_on": 0,
				"num_seeds": 10,
				"num_leechs": 5,
				"save_path": "/downloads/movies",
				"downloaded": 3650722201,
				"uploaded": 104857600,
				"ratio": 0.03
			}`+`]`)
		} else {
			fmt.Fprint(w, mockTorrentInfoJSON())
		}
	})
	mux.HandleFunc("/api/v2/torrents/properties", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "abc123def456", r.URL.Query().Get("hash"))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, mockTorrentPropertiesJSON())
	})

	server := newTestServer(t, mux)
	defer server.Close()

	client := NewClient(&Config{
		Host:     server.URL,
		Username: "admin",
		Password: "password",
	})

	details, err := client.GetTorrentDetails(context.Background(), "abc123def456")
	require.NoError(t, err)
	require.NotNil(t, details)

	assert.Equal(t, "abc123def456", details.Hash)
	assert.Equal(t, "[SubGroup] Movie Name (2024) [1080p]", details.Name)
	assert.Equal(t, StatusDownloading, details.Status)
	assert.Equal(t, int64(4194304), details.PieceSize)
	assert.Equal(t, "Downloaded via torrent", details.Comment)
	assert.Equal(t, "qBittorrent v4.5.2", details.CreatedBy)
	assert.Equal(t, int64(3600), details.TimeElapsed)
	assert.Equal(t, int64(8388608), details.AvgDownSpeed)
}

func TestClient_GetTorrentDetails_NotFound(t *testing.T) {
	mux := setupTorrentTestServer(t)
	mux.HandleFunc("/api/v2/torrents/info", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// hashes filter returns empty when no match
		fmt.Fprint(w, "[]")
	})

	server := newTestServer(t, mux)
	defer server.Close()

	client := NewClient(&Config{
		Host:     server.URL,
		Username: "admin",
		Password: "password",
	})

	details, err := client.GetTorrentDetails(context.Background(), "nonexistent_hash")
	assert.Nil(t, details)
	assert.Error(t, err)

	var connErr *ConnectionError
	assert.ErrorAs(t, err, &connErr)
	assert.Equal(t, ErrCodeTorrentNotFound, connErr.Code)
}

func TestClient_GetTorrentDetails_PropertiesError(t *testing.T) {
	mux := setupTorrentTestServer(t)
	mux.HandleFunc("/api/v2/torrents/info", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// Return a single matching torrent
		fmt.Fprint(w, `[{"hash":"abc123def456","name":"Test","state":"downloading","added_on":1704067200}]`)
	})
	mux.HandleFunc("/api/v2/torrents/properties", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	server := newTestServer(t, mux)
	defer server.Close()

	client := NewClient(&Config{
		Host:     server.URL,
		Username: "admin",
		Password: "password",
	})

	details, err := client.GetTorrentDetails(context.Background(), "abc123def456")
	assert.Nil(t, details)
	assert.Error(t, err)
}
