package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/qbittorrent"
)

// MockQBServiceForDownload mocks the QBittorrentServiceInterface for download tests.
type MockQBServiceForDownload struct {
	mock.Mock
}

func (m *MockQBServiceForDownload) GetConfig(ctx context.Context) (*qbittorrent.Config, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*qbittorrent.Config), args.Error(1)
}

func (m *MockQBServiceForDownload) SaveConfig(ctx context.Context, config *qbittorrent.Config) error {
	args := m.Called(ctx, config)
	return args.Error(0)
}

func (m *MockQBServiceForDownload) TestConnection(ctx context.Context) (*qbittorrent.VersionInfo, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*qbittorrent.VersionInfo), args.Error(1)
}

func (m *MockQBServiceForDownload) TestConnectionWithConfig(ctx context.Context, config *qbittorrent.Config) (*qbittorrent.VersionInfo, error) {
	args := m.Called(ctx, config)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*qbittorrent.VersionInfo), args.Error(1)
}

func (m *MockQBServiceForDownload) IsConfigured(ctx context.Context) bool {
	args := m.Called(ctx)
	return args.Bool(0)
}

func newTestDownloadService(mockQB *MockQBServiceForDownload) *DownloadService {
	return NewDownloadService(mockQB, slog.Default())
}

func TestNewDownloadService(t *testing.T) {
	mockQB := new(MockQBServiceForDownload)
	service := NewDownloadService(mockQB, slog.Default())
	assert.NotNil(t, service)
}

func TestDownloadService_GetAllDownloads_NotConfigured(t *testing.T) {
	mockQB := new(MockQBServiceForDownload)
	mockQB.On("GetConfig", mock.Anything).Return(&qbittorrent.Config{Host: ""}, nil)

	service := newTestDownloadService(mockQB)
	torrents, err := service.GetAllDownloads(context.Background(), "all", "", "")

	assert.Nil(t, torrents)
	assert.Error(t, err)

	var connErr *qbittorrent.ConnectionError
	require.ErrorAs(t, err, &connErr)
	assert.Equal(t, qbittorrent.ErrCodeNotConfigured, connErr.Code)
	mockQB.AssertExpectations(t)
}

func TestDownloadService_GetAllDownloads_ConfigError(t *testing.T) {
	mockQB := new(MockQBServiceForDownload)
	mockQB.On("GetConfig", mock.Anything).Return(nil, errors.New("database error"))

	service := newTestDownloadService(mockQB)
	torrents, err := service.GetAllDownloads(context.Background(), "all", "", "")

	assert.Nil(t, torrents)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get qBittorrent config")
	mockQB.AssertExpectations(t)
}

func TestDownloadService_MapToQBFilter(t *testing.T) {
	tests := []struct {
		input    string
		expected qbittorrent.TorrentsFilter
	}{
		{"all", qbittorrent.FilterAll},
		{"downloading", qbittorrent.FilterDownloading},
		{"paused", qbittorrent.FilterPaused},
		{"completed", qbittorrent.FilterCompleted},
		{"seeding", qbittorrent.FilterSeeding},
		{"error", qbittorrent.TorrentsFilter("errored")},
		{"invalid", qbittorrent.FilterAll},
		{"", qbittorrent.FilterAll},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := mapToQBFilter(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDownloadService_ValidFilters(t *testing.T) {
	valid := []string{"all", "downloading", "paused", "completed", "seeding", "error"}
	for _, f := range valid {
		assert.True(t, validFilters[f], "expected %q to be valid", f)
	}
	assert.False(t, validFilters["invalid"])
	assert.False(t, validFilters[""])
}

func TestDownloadService_GetDownloadDetails_NotConfigured(t *testing.T) {
	mockQB := new(MockQBServiceForDownload)
	mockQB.On("GetConfig", mock.Anything).Return(&qbittorrent.Config{Host: ""}, nil)

	service := newTestDownloadService(mockQB)
	details, err := service.GetDownloadDetails(context.Background(), "abc123")

	assert.Nil(t, details)
	assert.Error(t, err)

	var connErr *qbittorrent.ConnectionError
	require.ErrorAs(t, err, &connErr)
	assert.Equal(t, qbittorrent.ErrCodeNotConfigured, connErr.Code)
	mockQB.AssertExpectations(t)
}

func TestDownloadService_GetDownloadDetails_ConfigError(t *testing.T) {
	mockQB := new(MockQBServiceForDownload)
	mockQB.On("GetConfig", mock.Anything).Return(nil, errors.New("database error"))

	service := newTestDownloadService(mockQB)
	details, err := service.GetDownloadDetails(context.Background(), "abc123")

	assert.Nil(t, details)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get qBittorrent config")
	mockQB.AssertExpectations(t)
}

func TestDownloadService_GetDownloadCounts_NotConfigured(t *testing.T) {
	mockQB := new(MockQBServiceForDownload)
	mockQB.On("GetConfig", mock.Anything).Return(&qbittorrent.Config{Host: ""}, nil)

	service := newTestDownloadService(mockQB)
	counts, err := service.GetDownloadCounts(context.Background())

	assert.Nil(t, counts)
	assert.Error(t, err)

	var connErr *qbittorrent.ConnectionError
	require.ErrorAs(t, err, &connErr)
	assert.Equal(t, qbittorrent.ErrCodeNotConfigured, connErr.Code)
	mockQB.AssertExpectations(t)
}

func TestDownloadService_ImplementsInterface(t *testing.T) {
	var _ DownloadServiceInterface = (*DownloadService)(nil)
}

// setupMockQBServer creates a mock qBittorrent HTTP server for integration-style service tests.
func setupMockQBServer(t *testing.T, torrentsJSON string) (*httptest.Server, *MockQBServiceForDownload) {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/auth/login", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "SID", Value: "test-session"})
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "Ok.")
	})
	mux.HandleFunc("/api/v2/torrents/info", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, torrentsJSON)
	})

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mockQB := new(MockQBServiceForDownload)
	mockQB.On("GetConfig", mock.Anything).Return(&qbittorrent.Config{
		Host:     server.URL,
		Username: "admin",
		Password: "password",
	}, nil)

	return server, mockQB
}

func TestDownloadService_GetDownloadCounts_AggregatesCorrectly(t *testing.T) {
	// GIVEN: qBittorrent server returns torrents with mixed statuses
	torrentsJSON := `[
		{"hash":"a1","name":"Movie 1","state":"downloading","added_on":1704067200},
		{"hash":"a2","name":"Movie 2","state":"forcedDL","added_on":1704067200},
		{"hash":"a3","name":"Movie 3","state":"pausedDL","added_on":1704067200},
		{"hash":"a4","name":"Movie 4","state":"stalledUP","added_on":1704067200},
		{"hash":"a5","name":"Movie 5","state":"stalledUP","added_on":1704067200},
		{"hash":"a6","name":"Movie 6","state":"stalledUP","added_on":1704067200},
		{"hash":"a7","name":"Movie 7","state":"uploading","added_on":1704067200},
		{"hash":"a8","name":"Movie 8","state":"error","added_on":1704067200}
	]`
	_, mockQB := setupMockQBServer(t, torrentsJSON)
	service := newTestDownloadService(mockQB)

	// WHEN: GetDownloadCounts is called
	counts, err := service.GetDownloadCounts(context.Background())

	// THEN: counts are correctly aggregated by normalized status
	require.NoError(t, err)
	assert.Equal(t, 8, counts.All)
	assert.Equal(t, 2, counts.Downloading) // downloading + forcedDL
	assert.Equal(t, 1, counts.Paused)      // pausedDL
	assert.Equal(t, 3, counts.Completed)   // stalledUP → completed
	assert.Equal(t, 1, counts.Seeding)     // uploading → seeding
	assert.Equal(t, 1, counts.Error)       // error
	mockQB.AssertExpectations(t)
}

func TestDownloadService_GetDownloadCounts_EmptyList(t *testing.T) {
	// GIVEN: qBittorrent server returns empty torrent list
	_, mockQB := setupMockQBServer(t, "[]")
	service := newTestDownloadService(mockQB)

	// WHEN: GetDownloadCounts is called
	counts, err := service.GetDownloadCounts(context.Background())

	// THEN: all counts are zero
	require.NoError(t, err)
	assert.Equal(t, 0, counts.All)
	assert.Equal(t, 0, counts.Downloading)
	assert.Equal(t, 0, counts.Paused)
	assert.Equal(t, 0, counts.Completed)
	assert.Equal(t, 0, counts.Seeding)
	assert.Equal(t, 0, counts.Error)
	mockQB.AssertExpectations(t)
}

func TestDownloadService_GetDownloadCounts_UnmappedStatusesNotCounted(t *testing.T) {
	// GIVEN: torrents with statuses that don't map to the 5 counted categories
	// stalledDL → stalled, queuedDL → queued, checkingDL → checking
	torrentsJSON := `[
		{"hash":"a1","name":"Stalled","state":"stalledDL","added_on":1704067200},
		{"hash":"a2","name":"Queued","state":"queuedDL","added_on":1704067200},
		{"hash":"a3","name":"Checking","state":"checkingDL","added_on":1704067200},
		{"hash":"a4","name":"Downloading","state":"downloading","added_on":1704067200}
	]`
	_, mockQB := setupMockQBServer(t, torrentsJSON)
	service := newTestDownloadService(mockQB)

	// WHEN: GetDownloadCounts is called
	counts, err := service.GetDownloadCounts(context.Background())

	// THEN: all=4 but individual sum < all because stalled/queued/checking aren't counted
	require.NoError(t, err)
	assert.Equal(t, 4, counts.All)
	assert.Equal(t, 1, counts.Downloading)
	assert.Equal(t, 0, counts.Paused)
	assert.Equal(t, 0, counts.Completed)
	assert.Equal(t, 0, counts.Seeding)
	assert.Equal(t, 0, counts.Error)
	// Sum of individual < All (3 unmapped statuses not counted)
	sum := counts.Downloading + counts.Paused + counts.Completed + counts.Seeding + counts.Error
	assert.Less(t, sum, counts.All)
	mockQB.AssertExpectations(t)
}

func TestDownloadService_GetAllDownloads_Success(t *testing.T) {
	// GIVEN: qBittorrent server returns torrents
	torrentsJSON := `[
		{"hash":"a1","name":"Movie A","state":"downloading","added_on":1704067200,"size":1000},
		{"hash":"a2","name":"Movie B","state":"pausedDL","added_on":1704067300,"size":2000}
	]`
	_, mockQB := setupMockQBServer(t, torrentsJSON)
	service := newTestDownloadService(mockQB)

	// WHEN: GetAllDownloads is called with filter
	torrents, err := service.GetAllDownloads(context.Background(), "all", "added_on", "desc")

	// THEN: returns mapped torrents
	require.NoError(t, err)
	require.Len(t, torrents, 2)
	assert.Equal(t, "a1", torrents[0].Hash)
	assert.Equal(t, qbittorrent.StatusDownloading, torrents[0].Status)
	assert.Equal(t, "a2", torrents[1].Hash)
	assert.Equal(t, qbittorrent.StatusPaused, torrents[1].Status)
	mockQB.AssertExpectations(t)
}

func TestDownloadService_GetAllDownloads_InvalidFilterFallsBackToAll(t *testing.T) {
	// GIVEN: qBittorrent server returns torrents
	_, mockQB := setupMockQBServer(t, "[]")
	service := newTestDownloadService(mockQB)

	// WHEN: called with invalid filter
	torrents, err := service.GetAllDownloads(context.Background(), "nonexistent", "added_on", "desc")

	// THEN: does not error (falls back to "all")
	require.NoError(t, err)
	assert.Empty(t, torrents)
	mockQB.AssertExpectations(t)
}

func TestDownloadService_GetAllDownloads_StatusSort(t *testing.T) {
	// GIVEN: torrents with different statuses
	torrentsJSON := `[
		{"hash":"a1","name":"Seeding","state":"uploading","added_on":1704067200},
		{"hash":"a2","name":"Downloading","state":"downloading","added_on":1704067200},
		{"hash":"a3","name":"Paused","state":"pausedDL","added_on":1704067200}
	]`
	_, mockQB := setupMockQBServer(t, torrentsJSON)
	service := newTestDownloadService(mockQB)

	// WHEN: sorted by status ascending
	torrents, err := service.GetAllDownloads(context.Background(), "all", "status", "asc")

	// THEN: sorted alphabetically by status string
	require.NoError(t, err)
	require.Len(t, torrents, 3)
	// "downloading" < "paused" < "seeding" (alphabetical)
	assert.Equal(t, qbittorrent.StatusDownloading, torrents[0].Status)
	assert.Equal(t, qbittorrent.StatusPaused, torrents[1].Status)
	assert.Equal(t, qbittorrent.StatusSeeding, torrents[2].Status)
	mockQB.AssertExpectations(t)
}

func TestDownloadService_GetDownloadCounts_ConfigError(t *testing.T) {
	// GIVEN: GetConfig returns an error
	mockQB := new(MockQBServiceForDownload)
	mockQB.On("GetConfig", mock.Anything).Return(nil, errors.New("database error"))

	service := newTestDownloadService(mockQB)

	// WHEN: GetDownloadCounts is called
	counts, err := service.GetDownloadCounts(context.Background())

	// THEN: returns nil counts and wrapped error
	assert.Nil(t, counts)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get qBittorrent config")
	mockQB.AssertExpectations(t)
}

func TestDownloadService_GetDownloadDetails_Success(t *testing.T) {
	// GIVEN: qBittorrent server returns torrent details
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/auth/login", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "SID", Value: "test-session"})
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "Ok.")
	})
	mux.HandleFunc("/api/v2/torrents/info", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `[{"hash":"abc123","name":"Test Movie","state":"downloading","added_on":1704067200,"size":5000}]`)
	})
	mux.HandleFunc("/api/v2/torrents/properties", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"piece_size":4194304,"comment":"Test comment","creation_date":1704067200,"time_elapsed":3600}`)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	mockQB := new(MockQBServiceForDownload)
	mockQB.On("GetConfig", mock.Anything).Return(&qbittorrent.Config{
		Host:     server.URL,
		Username: "admin",
		Password: "password",
	}, nil)

	service := newTestDownloadService(mockQB)

	// WHEN: GetDownloadDetails is called
	details, err := service.GetDownloadDetails(context.Background(), "abc123")

	// THEN: returns valid details
	require.NoError(t, err)
	require.NotNil(t, details)
	assert.Equal(t, "abc123", details.Hash)
	assert.Equal(t, "Test Movie", details.Name)
	mockQB.AssertExpectations(t)
}

func TestDownloadService_GetDownloadDetails_ClientError(t *testing.T) {
	// GIVEN: qBittorrent server returns error for torrent info
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/auth/login", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "SID", Value: "test-session"})
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "Ok.")
	})
	mux.HandleFunc("/api/v2/torrents/info", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `[]`) // Empty = not found
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	mockQB := new(MockQBServiceForDownload)
	mockQB.On("GetConfig", mock.Anything).Return(&qbittorrent.Config{
		Host:     server.URL,
		Username: "admin",
		Password: "password",
	}, nil)

	service := newTestDownloadService(mockQB)

	// WHEN: GetDownloadDetails is called with non-existent hash
	details, err := service.GetDownloadDetails(context.Background(), "nonexistent")

	// THEN: returns error (torrent not found)
	assert.Nil(t, details)
	assert.Error(t, err)
	mockQB.AssertExpectations(t)
}

func TestDownloadService_GetAllDownloads_StatusSortDesc(t *testing.T) {
	// GIVEN: torrents with different statuses
	torrentsJSON := `[
		{"hash":"a1","name":"Downloading","state":"downloading","added_on":1704067200},
		{"hash":"a2","name":"Seeding","state":"uploading","added_on":1704067200},
		{"hash":"a3","name":"Paused","state":"pausedDL","added_on":1704067200}
	]`
	_, mockQB := setupMockQBServer(t, torrentsJSON)
	service := newTestDownloadService(mockQB)

	// WHEN: sorted by status descending
	torrents, err := service.GetAllDownloads(context.Background(), "all", "status", "desc")

	// THEN: reverse alphabetical order
	require.NoError(t, err)
	require.Len(t, torrents, 3)
	assert.Equal(t, qbittorrent.StatusSeeding, torrents[0].Status)
	assert.Equal(t, qbittorrent.StatusPaused, torrents[1].Status)
	assert.Equal(t, qbittorrent.StatusDownloading, torrents[2].Status)
	mockQB.AssertExpectations(t)
}
