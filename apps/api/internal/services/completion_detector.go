package services

import (
	"context"
	"log/slog"
	"sync"

	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/qbittorrent"
	"github.com/vido/api/internal/repository"
)

// CompletionDetectorInterface defines the contract for detecting newly completed downloads.
type CompletionDetectorInterface interface {
	DetectNewCompletions(ctx context.Context, torrents []qbittorrent.Torrent) []qbittorrent.Torrent
}

// MovieFileLookup checks if a movie exists by file path (for library duplicate detection).
// Satisfied by repository.MovieRepositoryInterface.
type MovieFileLookup interface {
	FindByFilePath(ctx context.Context, filePath string) (*models.Movie, error)
}

// CompletionDetector detects newly completed downloads and filters out already-processed ones.
type CompletionDetector struct {
	parseJobRepo repository.ParseJobRepositoryInterface
	movieRepo    MovieFileLookup
	logger       *slog.Logger
	seenHashes   map[string]bool
	mu           sync.RWMutex
}

// NewCompletionDetector creates a new CompletionDetector.
func NewCompletionDetector(
	parseJobRepo repository.ParseJobRepositoryInterface,
	movieRepo MovieFileLookup,
	logger *slog.Logger,
) *CompletionDetector {
	return &CompletionDetector{
		parseJobRepo: parseJobRepo,
		movieRepo:    movieRepo,
		logger:       logger,
		seenHashes:   make(map[string]bool),
	}
}

// DetectNewCompletions scans the given torrents and returns only those that are
// newly completed: status is "completed", not previously seen, no existing parse job,
// and not already in the media library.
func (d *CompletionDetector) DetectNewCompletions(ctx context.Context, torrents []qbittorrent.Torrent) []qbittorrent.Torrent {
	var newCompletions []qbittorrent.Torrent

	for _, t := range torrents {
		if t.Status != qbittorrent.StatusCompleted {
			continue
		}

		// Check in-memory seen cache
		d.mu.RLock()
		seen := d.seenHashes[t.Hash]
		d.mu.RUnlock()

		if seen {
			continue
		}

		// Check if already has a parse job
		existingJob, err := d.parseJobRepo.GetByTorrentHash(ctx, t.Hash)
		if err != nil {
			d.logger.Debug("Error checking parse job, treating as new",
				"hash", t.Hash,
				"error", err,
			)
		}
		if existingJob != nil {
			d.mu.Lock()
			d.seenHashes[t.Hash] = true
			d.mu.Unlock()
			continue
		}

		// Check if file already in library (duplicate detection)
		existingMedia, err := d.movieRepo.FindByFilePath(ctx, t.SavePath)
		if err != nil {
			d.logger.Debug("Error checking library, treating as new",
				"hash", t.Hash,
				"error", err,
			)
		}
		if existingMedia != nil {
			d.logger.Info("File already in library, skipping",
				"hash", t.Hash,
				"path", t.SavePath,
			)
			d.mu.Lock()
			d.seenHashes[t.Hash] = true
			d.mu.Unlock()
			continue
		}

		// New completion detected
		newCompletions = append(newCompletions, t)

		d.mu.Lock()
		d.seenHashes[t.Hash] = true
		d.mu.Unlock()
	}

	return newCompletions
}

// Compile-time interface verification
var _ CompletionDetectorInterface = (*CompletionDetector)(nil)
