package workers

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/vido/api/internal/qbittorrent"
	"github.com/vido/api/internal/services"
)

// ParseWorker orchestrates completion detection and parse job processing.
// It polls for completed downloads, queues parse jobs, and processes them
// using a configurable number of concurrent workers (ARCH-5: 3-5 goroutines).
type ParseWorker struct {
	downloadService    services.DownloadServiceInterface
	completionDetector services.CompletionDetectorInterface
	parseQueueService  services.ParseQueueServiceInterface
	logger             *slog.Logger
	workerCount        int
	pollInterval       time.Duration
	wg                 sync.WaitGroup
	stop               chan struct{}
}

// NewParseWorker creates a new ParseWorker with default settings.
func NewParseWorker(
	downloadService services.DownloadServiceInterface,
	completionDetector services.CompletionDetectorInterface,
	parseQueueService services.ParseQueueServiceInterface,
	logger *slog.Logger,
) *ParseWorker {
	return &ParseWorker{
		downloadService:    downloadService,
		completionDetector: completionDetector,
		parseQueueService:  parseQueueService,
		logger:             logger,
		workerCount:        3,
		pollInterval:       5 * time.Second,
		stop:               make(chan struct{}),
	}
}

// Start launches the completion detector goroutine and parse worker goroutines.
func (w *ParseWorker) Start(ctx context.Context) {
	w.wg.Add(1)
	go w.runCompletionDetector(ctx)

	for i := 0; i < w.workerCount; i++ {
		w.wg.Add(1)
		go w.runParseWorker(ctx, i)
	}

	w.logger.Info("Parse workers started",
		"worker_count", w.workerCount,
		"poll_interval", w.pollInterval,
	)
}

// Stop signals all goroutines to stop and waits for them to finish.
func (w *ParseWorker) Stop() {
	close(w.stop)
	w.wg.Wait()
	w.logger.Info("Parse workers stopped")
}

func (w *ParseWorker) runCompletionDetector(ctx context.Context) {
	defer w.wg.Done()

	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.stop:
			return
		case <-ticker.C:
			w.checkForCompletions(ctx)
		}
	}
}

func (w *ParseWorker) checkForCompletions(ctx context.Context) {
	downloads, err := w.downloadService.GetAllDownloads(ctx, "completed", "", "")
	if err != nil {
		w.logger.Debug("Failed to get downloads for completion check", "error", err)
		return
	}

	newCompletions := w.completionDetector.DetectNewCompletions(ctx, downloads)

	for i := range newCompletions {
		t := &qbittorrent.Torrent{
			Hash:     newCompletions[i].Hash,
			Name:     newCompletions[i].Name,
			SavePath: newCompletions[i].SavePath,
		}
		if _, err := w.parseQueueService.QueueParseJob(ctx, t); err != nil {
			w.logger.Error("Failed to queue parse job",
				"hash", newCompletions[i].Hash,
				"error", err,
			)
		}
	}
}

func (w *ParseWorker) runParseWorker(ctx context.Context, workerID int) {
	defer w.wg.Done()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.stop:
			return
		case <-ticker.C:
			if err := w.parseQueueService.ProcessNextJob(ctx); err != nil {
				w.logger.Error("Parse worker error",
					"worker_id", workerID,
					"error", err,
				)
			}
		}
	}
}
