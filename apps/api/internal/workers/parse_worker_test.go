package workers

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/qbittorrent"
)

// --- Mocks ---

type mockDownloadService struct {
	torrents []qbittorrent.Torrent
	err      error
}

func (m *mockDownloadService) GetAllDownloads(_ context.Context, _, _, _ string) ([]qbittorrent.Torrent, error) {
	return m.torrents, m.err
}

func (m *mockDownloadService) GetDownloadDetails(_ context.Context, _ string) (*qbittorrent.TorrentDetails, error) {
	return nil, nil
}

func (m *mockDownloadService) GetDownloadCounts(_ context.Context) (*qbittorrent.DownloadCounts, error) {
	return nil, nil
}

type mockCompletionDetector struct {
	completions []qbittorrent.Torrent
}

func (m *mockCompletionDetector) DetectNewCompletions(_ context.Context, torrents []qbittorrent.Torrent) []qbittorrent.Torrent {
	return m.completions
}

type mockParseQueueService struct {
	mu          sync.Mutex
	queuedJobs  []*qbittorrent.Torrent
	processErr  error
	processCalls int
}

func (m *mockParseQueueService) QueueParseJob(_ context.Context, torrent *qbittorrent.Torrent) (*models.ParseJob, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.queuedJobs = append(m.queuedJobs, torrent)
	return &models.ParseJob{ID: "job-1", TorrentHash: torrent.Hash}, nil
}

func (m *mockParseQueueService) ProcessNextJob(_ context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.processCalls++
	return m.processErr
}

func (m *mockParseQueueService) GetJobStatus(_ context.Context, _ string) (*models.ParseJob, error) {
	return nil, nil
}

func (m *mockParseQueueService) RetryJob(_ context.Context, _ string) error {
	return nil
}

func (m *mockParseQueueService) ListJobs(_ context.Context, _ int) ([]*models.ParseJob, error) {
	return nil, nil
}

// --- Tests ---

func TestParseWorker_NewParseWorker(t *testing.T) {
	worker := NewParseWorker(
		&mockDownloadService{},
		&mockCompletionDetector{},
		&mockParseQueueService{},
		slog.Default(),
	)

	assert.NotNil(t, worker)
	assert.Equal(t, 3, worker.workerCount)
	assert.Equal(t, 5*time.Second, worker.pollInterval)
}

func TestParseWorker_StartAndStop(t *testing.T) {
	worker := NewParseWorker(
		&mockDownloadService{},
		&mockCompletionDetector{},
		&mockParseQueueService{},
		slog.Default(),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	worker.Start(ctx)
	// Give it a moment to start goroutines
	time.Sleep(50 * time.Millisecond)
	worker.Stop()
	// If we get here without hanging, the test passes
}

func TestParseWorker_CheckForCompletions(t *testing.T) {
	completedTorrent := qbittorrent.Torrent{
		Hash:     "hash1",
		Name:     "movie.mkv",
		SavePath: "/downloads",
		Status:   qbittorrent.StatusCompleted,
	}

	dlService := &mockDownloadService{
		torrents: []qbittorrent.Torrent{completedTorrent},
	}

	detector := &mockCompletionDetector{
		completions: []qbittorrent.Torrent{completedTorrent},
	}

	pqService := &mockParseQueueService{}

	worker := NewParseWorker(dlService, detector, pqService, slog.Default())

	worker.checkForCompletions(context.Background())

	pqService.mu.Lock()
	defer pqService.mu.Unlock()
	require.Len(t, pqService.queuedJobs, 1)
	assert.Equal(t, "hash1", pqService.queuedJobs[0].Hash)
}

func TestParseWorker_CheckForCompletions_DownloadError(t *testing.T) {
	dlService := &mockDownloadService{
		err: fmt.Errorf("connection refused"),
	}

	pqService := &mockParseQueueService{}
	worker := NewParseWorker(dlService, &mockCompletionDetector{}, pqService, slog.Default())

	// Should not panic
	worker.checkForCompletions(context.Background())

	pqService.mu.Lock()
	defer pqService.mu.Unlock()
	assert.Len(t, pqService.queuedJobs, 0)
}

func TestParseWorker_CheckForCompletions_NoNewCompletions(t *testing.T) {
	dlService := &mockDownloadService{
		torrents: []qbittorrent.Torrent{
			{Hash: "hash1", Status: qbittorrent.StatusCompleted},
		},
	}

	detector := &mockCompletionDetector{
		completions: nil, // Already seen
	}

	pqService := &mockParseQueueService{}
	worker := NewParseWorker(dlService, detector, pqService, slog.Default())

	worker.checkForCompletions(context.Background())

	pqService.mu.Lock()
	defer pqService.mu.Unlock()
	assert.Len(t, pqService.queuedJobs, 0)
}

func TestParseWorker_ContextCancellation(t *testing.T) {
	worker := NewParseWorker(
		&mockDownloadService{},
		&mockCompletionDetector{},
		&mockParseQueueService{},
		slog.Default(),
	)

	ctx, cancel := context.WithCancel(context.Background())
	worker.Start(ctx)

	// Cancel context should stop all workers
	cancel()
	done := make(chan struct{})
	go func() {
		worker.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success - workers stopped
	case <-time.After(2 * time.Second):
		t.Fatal("workers did not stop after context cancellation")
	}
}

// --- Queue error mock ---

type mockParseQueueServiceQueueFails struct {
	mockParseQueueService
	queueErr   error
	queueCalls int
}

func (m *mockParseQueueServiceQueueFails) QueueParseJob(_ context.Context, _ *qbittorrent.Torrent) (*models.ParseJob, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.queueCalls++
	return nil, m.queueErr
}

// --- Expanded worker tests ---

func TestParseWorker_CheckForCompletions_QueueError(t *testing.T) {
	torrents := []qbittorrent.Torrent{
		{Hash: "hash1", Name: "m1.mkv", SavePath: "/dl", Status: qbittorrent.StatusCompleted},
		{Hash: "hash2", Name: "m2.mkv", SavePath: "/dl", Status: qbittorrent.StatusCompleted},
	}

	dlService := &mockDownloadService{torrents: torrents}
	detector := &mockCompletionDetector{completions: torrents}
	pqService := &mockParseQueueServiceQueueFails{queueErr: fmt.Errorf("db full")}

	worker := NewParseWorker(dlService, detector, pqService, slog.Default())

	// GIVEN: QueueParseJob will fail for all completions
	// WHEN: Checking for completions
	worker.checkForCompletions(context.Background())

	// THEN: Worker should not panic, and should attempt to queue both
	pqService.mu.Lock()
	defer pqService.mu.Unlock()
	assert.Equal(t, 2, pqService.queueCalls, "should attempt to queue all completions even when errors occur")
}

func TestParseWorker_CheckForCompletions_MultipleCompletions(t *testing.T) {
	torrents := []qbittorrent.Torrent{
		{Hash: "hash1", Name: "m1.mkv", SavePath: "/dl/a", Status: qbittorrent.StatusCompleted},
		{Hash: "hash2", Name: "m2.mkv", SavePath: "/dl/b", Status: qbittorrent.StatusCompleted},
		{Hash: "hash3", Name: "m3.mkv", SavePath: "/dl/c", Status: qbittorrent.StatusCompleted},
	}

	dlService := &mockDownloadService{torrents: torrents}
	detector := &mockCompletionDetector{completions: torrents}
	pqService := &mockParseQueueService{}

	worker := NewParseWorker(dlService, detector, pqService, slog.Default())

	// GIVEN: 3 newly completed torrents
	// WHEN: Checking for completions
	worker.checkForCompletions(context.Background())

	// THEN: All 3 should be queued for parsing
	pqService.mu.Lock()
	defer pqService.mu.Unlock()
	require.Len(t, pqService.queuedJobs, 3)
	assert.Equal(t, "hash1", pqService.queuedJobs[0].Hash)
	assert.Equal(t, "hash2", pqService.queuedJobs[1].Hash)
	assert.Equal(t, "hash3", pqService.queuedJobs[2].Hash)
}
