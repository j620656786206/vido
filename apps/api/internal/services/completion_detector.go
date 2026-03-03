package services

import (
	"context"
	"log/slog"
	"path/filepath"
	"sync"

	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/qbittorrent"
	"github.com/vido/api/internal/repository"
)

// maxSeenHashes caps the in-memory seen cache to prevent unbounded growth.
const maxSeenHashes = 10000

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
			d.logger.Warn("Error checking parse job, skipping torrent to avoid duplicates",
				"hash", t.Hash,
				"error", err,
			)
			continue
		}
		if existingJob != nil {
			d.markSeen(t.Hash)
			continue
		}

		// Check if file already in library (duplicate detection)
		// Use full file path (SavePath + Name) instead of just the directory
		fullPath := filepath.Join(t.SavePath, t.Name)
		existingMedia, err := d.movieRepo.FindByFilePath(ctx, fullPath)
		if err != nil {
			d.logger.Warn("Error checking library, skipping torrent to avoid duplicates",
				"hash", t.Hash,
				"error", err,
			)
			continue
		}
		if existingMedia != nil {
			d.logger.Info("File already in library, skipping",
				"hash", t.Hash,
				"path", fullPath,
			)
			d.markSeen(t.Hash)
			continue
		}

		// New completion detected
		newCompletions = append(newCompletions, t)
		d.markSeen(t.Hash)
	}

	return newCompletions
}

// markSeen adds a hash to the seen cache, evicting all entries when the cap is reached.
func (d *CompletionDetector) markSeen(hash string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if len(d.seenHashes) >= maxSeenHashes {
		d.logger.Info("Seen hashes cache full, clearing", "size", len(d.seenHashes))
		d.seenHashes = make(map[string]bool)
	}
	d.seenHashes[hash] = true
}

// Compile-time interface verification
var _ CompletionDetectorInterface = (*CompletionDetector)(nil)
