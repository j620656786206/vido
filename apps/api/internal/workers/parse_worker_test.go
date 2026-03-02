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
