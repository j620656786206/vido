package services

import (
	"context"
	"fmt"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/qbittorrent"
	"github.com/vido/api/internal/repository"
)

// --- Mocks ---

type mockParseJobRepo struct {
	jobs map[string]*models.ParseJob
	err  error
}

func newMockParseJobRepo() *mockParseJobRepo {
	return &mockParseJobRepo{jobs: make(map[string]*models.ParseJob)}
}

func (m *mockParseJobRepo) Create(_ context.Context, job *models.ParseJob) error {
	if m.err != nil {
		return m.err
	}
	m.jobs[job.TorrentHash] = job
	return nil
}

func (m *mockParseJobRepo) GetByID(_ context.Context, id string) (*models.ParseJob, error) {
	if m.err != nil {
		return nil, m.err
	}
	for _, j := range m.jobs {
		if j.ID == id {
			return j, nil
		}
	}
	return nil, fmt.Errorf("not found")
}

func (m *mockParseJobRepo) GetByTorrentHash(_ context.Context, hash string) (*models.ParseJob, error) {
	if m.err != nil {
		return nil, m.err
	}
	if j, ok := m.jobs[hash]; ok {
		return j, nil
	}
	return nil, nil
}

func (m *mockParseJobRepo) GetPending(_ context.Context, _ int) ([]*models.ParseJob, error) {
	return nil, nil
}

func (m *mockParseJobRepo) UpdateStatus(_ context.Context, _ string, _ models.ParseJobStatus, _ string) error {
	return nil
}

func (m *mockParseJobRepo) Update(_ context.Context, _ *models.ParseJob) error {
	return nil
}

func (m *mockParseJobRepo) Delete(_ context.Context, _ string) error {
	return nil
}

func (m *mockParseJobRepo) ListAll(_ context.Context, _ int) ([]*models.ParseJob, error) {
	return nil, nil
}

// Verify mock satisfies interface
var _ repository.ParseJobRepositoryInterface = (*mockParseJobRepo)(nil)

type mockMovieFileLookup struct {
	movies map[string]*models.Movie
}

func newMockMovieFileLookup() *mockMovieFileLookup {
	return &mockMovieFileLookup{movies: make(map[string]*models.Movie)}
}

func (m *mockMovieFileLookup) FindByFilePath(_ context.Context, filePath string) (*models.Movie, error) {
	if movie, ok := m.movies[filePath]; ok {
		return movie, nil
	}
	return nil, nil
}

func makeTorrent(hash, name string, status qbittorrent.TorrentStatus, savePath string) qbittorrent.Torrent {
	return qbittorrent.Torrent{
		Hash:     hash,
		Name:     name,
		Status:   status,
		SavePath: savePath,
	}
}

func newTestDetector(parseJobRepo repository.ParseJobRepositoryInterface, movieRepo MovieFileLookup) *CompletionDetector {
	return NewCompletionDetector(
		parseJobRepo,
		movieRepo,
		slog.Default(),
	)
}

// --- Tests ---

func TestCompletionDetector_DetectNewCompletions_OnlyCompleted(t *testing.T) {
	detector := newTestDetector(newMockParseJobRepo(), newMockMovieFileLookup())

	torrents := []qbittorrent.Torrent{
		makeTorrent("hash1", "downloading.mkv", qbittorrent.StatusDownloading, "/downloads"),
		makeTorrent("hash2", "completed.mkv", qbittorrent.StatusCompleted, "/downloads"),
		makeTorrent("hash3", "seeding.mkv", qbittorrent.StatusSeeding, "/downloads"),
		makeTorrent("hash4", "paused.mkv", qbittorrent.StatusPaused, "/downloads"),
	}

	result := detector.DetectNewCompletions(context.Background(), torrents)

	require.Len(t, result, 1)
	assert.Equal(t, "hash2", result[0].Hash)
}

func TestCompletionDetector_DetectNewCompletions_PreventsRetrigger(t *testing.T) {
	detector := newTestDetector(newMockParseJobRepo(), newMockMovieFileLookup())

	torrents := []qbittorrent.Torrent{
		makeTorrent("hash1", "movie.mkv", qbittorrent.StatusCompleted, "/downloads"),
	}

	// First call should detect
	result1 := detector.DetectNewCompletions(context.Background(), torrents)
	require.Len(t, result1, 1)

	// Second call should NOT re-detect (already seen)
	result2 := detector.DetectNewCompletions(context.Background(), torrents)
	assert.Len(t, result2, 0)
}

func TestCompletionDetector_DetectNewCompletions_SkipsExistingParseJob(t *testing.T) {
	parseJobRepo := newMockParseJobRepo()
	parseJobRepo.jobs["hash1"] = &models.ParseJob{
		ID:          "job-1",
		TorrentHash: "hash1",
		Status:      models.ParseJobCompleted,
	}

	detector := newTestDetector(parseJobRepo, newMockMovieFileLookup())

	torrents := []qbittorrent.Torrent{
		makeTorrent("hash1", "already-parsed.mkv", qbittorrent.StatusCompleted, "/downloads"),
		makeTorrent("hash2", "new-file.mkv", qbittorrent.StatusCompleted, "/downloads"),
	}

	result := detector.DetectNewCompletions(context.Background(), torrents)

	require.Len(t, result, 1)
	assert.Equal(t, "hash2", result[0].Hash)
}

func TestCompletionDetector_DetectNewCompletions_SkipsAlreadyInLibrary(t *testing.T) {
	movieRepo := newMockMovieFileLookup()
	// Full path = SavePath + Name (filepath.Join)
	movieRepo.movies["/downloads/existing.mkv"] = &models.Movie{
		ID:    "movie-1",
		Title: "Existing Movie",
	}

	detector := newTestDetector(newMockParseJobRepo(), movieRepo)

	torrents := []qbittorrent.Torrent{
		makeTorrent("hash1", "existing.mkv", qbittorrent.StatusCompleted, "/downloads"),
		makeTorrent("hash2", "new.mkv", qbittorrent.StatusCompleted, "/other"),
	}

	result := detector.DetectNewCompletions(context.Background(), torrents)

	require.Len(t, result, 1)
	assert.Equal(t, "hash2", result[0].Hash)
}

func TestCompletionDetector_DetectNewCompletions_Empty(t *testing.T) {
	detector := newTestDetector(newMockParseJobRepo(), newMockMovieFileLookup())

	result := detector.DetectNewCompletions(context.Background(), nil)
	assert.Nil(t, result)

	result = detector.DetectNewCompletions(context.Background(), []qbittorrent.Torrent{})
	assert.Nil(t, result)
}

func TestCompletionDetector_DetectNewCompletions_MultipleNewCompletions(t *testing.T) {
	detector := newTestDetector(newMockParseJobRepo(), newMockMovieFileLookup())

	torrents := []qbittorrent.Torrent{
		makeTorrent("hash1", "movie1.mkv", qbittorrent.StatusCompleted, "/downloads/a"),
		makeTorrent("hash2", "movie2.mkv", qbittorrent.StatusCompleted, "/downloads/b"),
		makeTorrent("hash3", "movie3.mkv", qbittorrent.StatusCompleted, "/downloads/c"),
	}

	result := detector.DetectNewCompletions(context.Background(), torrents)

	require.Len(t, result, 3)
	assert.Equal(t, "hash1", result[0].Hash)
	assert.Equal(t, "hash2", result[1].Hash)
	assert.Equal(t, "hash3", result[2].Hash)
}

func TestCompletionDetector_DetectNewCompletions_SeenHashesPersistAcrossCalls(t *testing.T) {
	detector := newTestDetector(newMockParseJobRepo(), newMockMovieFileLookup())

	// Call 1: detect hash1
	torrents1 := []qbittorrent.Torrent{
		makeTorrent("hash1", "movie1.mkv", qbittorrent.StatusCompleted, "/downloads"),
	}
	result1 := detector.DetectNewCompletions(context.Background(), torrents1)
	require.Len(t, result1, 1)

	// Call 2: hash1 already seen, hash2 is new
	torrents2 := []qbittorrent.Torrent{
		makeTorrent("hash1", "movie1.mkv", qbittorrent.StatusCompleted, "/downloads"),
		makeTorrent("hash2", "movie2.mkv", qbittorrent.StatusCompleted, "/other"),
	}
	result2 := detector.DetectNewCompletions(context.Background(), torrents2)
	require.Len(t, result2, 1)
	assert.Equal(t, "hash2", result2[0].Hash)
}

func TestCompletionDetector_DetectNewCompletions_ParseJobRepoErrorSkips(t *testing.T) {
	parseJobRepo := newMockParseJobRepo()
	parseJobRepo.err = fmt.Errorf("db error")

	detector := newTestDetector(parseJobRepo, newMockMovieFileLookup())

	torrents := []qbittorrent.Torrent{
		makeTorrent("hash1", "movie.mkv", qbittorrent.StatusCompleted, "/downloads"),
	}

	// Should skip the torrent on DB error to avoid duplicate processing
	result := detector.DetectNewCompletions(context.Background(), torrents)
	assert.Len(t, result, 0)
}
