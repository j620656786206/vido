package services

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
	"github.com/vido/api/internal/sse"
)

// ErrScanNotActive is returned when attempting to cancel a scan that is not running
var ErrScanNotActive = fmt.Errorf("SCANNER_NOT_ACTIVE: no scan is currently active")

// videoExtensions defines the supported video file extensions (lowercase).
// Using a function to prevent mutation of the lookup map.
var videoExtensions = func() map[string]bool {
	return map[string]bool{
		".mkv":  true,
		".mp4":  true,
		".avi":  true,
		".rmvb": true,
	}
}()

// ScanProgress represents the current state of an active scan
type ScanProgress struct {
	FilesFound   int       `json:"filesFound"`
	FilesCreated int       `json:"filesCreated"`
	FilesUpdated int       `json:"filesUpdated"`
	FilesSkipped int       `json:"filesSkipped"`
	FilesRemoved int       `json:"filesRemoved"`
	ErrorCount   int       `json:"errorCount"`
	CurrentFile  string    `json:"currentFile"`
	PercentDone  int       `json:"percentDone"`
	IsActive     bool      `json:"isActive"`
	StartedAt    time.Time `json:"startedAt,omitempty"`
}

// ScanResult contains the outcome of a completed scan operation
type ScanResult struct {
	FilesFound   int       `json:"filesFound"`
	FilesCreated int       `json:"filesCreated"`
	FilesUpdated int       `json:"filesUpdated"`
	FilesSkipped int       `json:"filesSkipped"`
	FilesRemoved int       `json:"filesRemoved"`
	ErrorCount   int       `json:"errorCount"`
	Duration     string    `json:"duration"`
	StartedAt    time.Time `json:"startedAt"`
	CompletedAt  time.Time `json:"completedAt"`
}

// ScannerService handles recursive folder scanning and video file discovery
type ScannerService struct {
	movieRepo  repository.MovieRepositoryInterface
	seriesRepo repository.SeriesRepositoryInterface
	mediaDirs  []string
	sseHub     *sse.Hub
	logger     *slog.Logger

	mu         sync.Mutex
	isScanning bool
	cancelChan chan struct{}
	progress   ScanProgress
}

// NewScannerService creates a new ScannerService
func NewScannerService(
	movieRepo repository.MovieRepositoryInterface,
	seriesRepo repository.SeriesRepositoryInterface,
	mediaDirs []string,
	sseHub *sse.Hub,
	logger *slog.Logger,
) *ScannerService {
	if logger == nil {
		logger = slog.Default()
	}
	return &ScannerService{
		movieRepo:  movieRepo,
		seriesRepo: seriesRepo,
		mediaDirs:  mediaDirs,
		sseHub:     sseHub,
		logger:     logger,
	}
}

// StartScan initiates a recursive scan of all configured media directories.
// Returns SCANNER_ALREADY_RUNNING error if a scan is already in progress.
func (s *ScannerService) StartScan(ctx context.Context) (*ScanResult, error) {
	s.mu.Lock()
	if s.isScanning {
		s.mu.Unlock()
		return nil, fmt.Errorf("SCANNER_ALREADY_RUNNING: a scan is already in progress")
	}
	s.isScanning = true
	s.cancelChan = make(chan struct{})
	s.progress = ScanProgress{
		IsActive:  true,
		StartedAt: time.Now(),
	}
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.isScanning = false
		s.progress.IsActive = false
		s.mu.Unlock()
	}()

	startedAt := time.Now()
	s.logger.Info("scan started", "media_dirs", s.mediaDirs)

	// Track seen resolved paths to deduplicate across directories
	seenPaths := make(map[string]bool)
	var pendingMovies []*models.Movie

	for _, dir := range s.mediaDirs {
		// Check cancellation
		select {
		case <-s.cancelChan:
			s.logger.Info("scan cancelled before walking directory", "dir", dir)
			return s.buildResult(startedAt), nil
		default:
		}

		// Validate directory
		info, err := os.Stat(dir)
		if err != nil {
			if os.IsNotExist(err) {
				s.logger.Warn("SCANNER_PATH_NOT_FOUND: configured path does not exist", "path", dir)
				s.mu.Lock()
				s.progress.ErrorCount++
				s.mu.Unlock()
			} else if os.IsPermission(err) {
				s.logger.Warn("SCANNER_PERMISSION_DENIED: cannot access configured path", "path", dir, "error", err)
				s.mu.Lock()
				s.progress.ErrorCount++
				s.mu.Unlock()
			} else {
				s.logger.Error("failed to stat directory", "path", dir, "error", err)
				s.mu.Lock()
				s.progress.ErrorCount++
				s.mu.Unlock()
			}
			continue
		}
		if !info.IsDir() {
			s.logger.Warn("SCANNER_PATH_NOT_FOUND: configured path is not a directory", "path", dir)
			s.mu.Lock()
			s.progress.ErrorCount++
			s.mu.Unlock()
			continue
		}

		err = s.walkDirectory(ctx, dir, seenPaths, &pendingMovies)
		if err != nil {
			s.logger.Error("SCANNER_PARSE_FAILED: error walking directory", "path", dir, "error", err)
			s.mu.Lock()
			s.progress.ErrorCount++
			s.mu.Unlock()
		}
	}

	// Flush remaining pending movies
	if len(pendingMovies) > 0 {
		if err := s.flushBatch(ctx, &pendingMovies); err != nil {
			s.logger.Error("failed to flush final batch", "error", err)
		}
	}

	// Detect removed files (Story 7-2: incremental scan)
	removedCount, err := s.detectRemovedFiles(ctx)
	if err != nil {
		s.logger.Error("failed to detect removed files", "error", err)
	} else if removedCount > 0 {
		s.mu.Lock()
		s.progress.FilesRemoved = removedCount
		s.mu.Unlock()
		s.logger.Info("detected removed files", "count", removedCount)
	}

	result := s.buildResult(startedAt)

	// Broadcast final completion event
	s.broadcastProgress()

	s.logger.Info("scan completed",
		"files_found", result.FilesFound,
		"files_created", result.FilesCreated,
		"files_updated", result.FilesUpdated,
		"files_skipped", result.FilesSkipped,
		"error_count", result.ErrorCount,
		"duration", result.Duration,
	)

	return result, nil
}

// IsScanActive returns true if a scan is currently in progress (thread-safe)
func (s *ScannerService) IsScanActive() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.isScanning
}

// CancelScan cancels the currently active scan
func (s *ScannerService) CancelScan() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.isScanning {
		return ErrScanNotActive
	}
	close(s.cancelChan)
	s.logger.Info("scan cancellation requested")
	return nil
}

// GetProgress returns a copy of the current scan progress (thread-safe)
func (s *ScannerService) GetProgress() ScanProgress {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.progress
}

// walkDirectory recursively walks a directory and discovers video files
func (s *ScannerService) walkDirectory(ctx context.Context, root string, seenPaths map[string]bool, pendingMovies *[]*models.Movie) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		// Check cancellation
		select {
		case <-s.cancelChan:
			return filepath.SkipAll
		default:
		}

		if err != nil {
			if os.IsPermission(err) {
				s.logger.Warn("SCANNER_PERMISSION_DENIED: cannot access path", "path", path, "error", err)
				s.mu.Lock()
				s.progress.ErrorCount++
				s.mu.Unlock()
				return nil // continue scanning other paths
			}
			s.logger.Error("error accessing path", "path", path, "error", err)
			s.mu.Lock()
			s.progress.ErrorCount++
			s.mu.Unlock()
			return nil
		}

		// Skip directories (but still walk into them)
		if d.IsDir() {
			return nil
		}

		// Check if this is a video file
		if !isVideoFile(path) {
			return nil
		}

		// Resolve symlinks to get the real path
		resolvedPath, err := filepath.EvalSymlinks(path)
		if err != nil {
			s.logger.Warn("failed to resolve symlink", "path", path, "error", err)
			s.mu.Lock()
			s.progress.ErrorCount++
			s.mu.Unlock()
			return nil
		}

		// Convert to absolute path
		resolvedPath, err = filepath.Abs(resolvedPath)
		if err != nil {
			s.logger.Warn("failed to get absolute path", "path", path, "error", err)
			s.mu.Lock()
			s.progress.ErrorCount++
			s.mu.Unlock()
			return nil
		}

		// Deduplicate by resolved path (across directories and symlinks)
		if seenPaths[resolvedPath] {
			s.mu.Lock()
			s.progress.FilesSkipped++
			s.mu.Unlock()
			return nil
		}
		seenPaths[resolvedPath] = true

		s.mu.Lock()
		s.progress.FilesFound++
		s.progress.CurrentFile = resolvedPath
		filesFound := s.progress.FilesFound
		s.mu.Unlock()

		// Broadcast progress every 10 files
		if filesFound%10 == 0 {
			s.broadcastProgress()
		}

		// Process the video file
		err = s.processVideoFile(ctx, resolvedPath, pendingMovies)
		if err != nil {
			s.logger.Error("failed to process video file", "path", resolvedPath, "error", err)
			s.mu.Lock()
			s.progress.ErrorCount++
			s.mu.Unlock()
		}

		return nil
	})
}

// isVideoFile checks if a file has a supported video extension (case-insensitive)
func isVideoFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return videoExtensions[ext]
}

// processVideoFile checks for duplicates and either creates or updates a movie record.
// Note: All scanned files are initially created as Movie records regardless of whether
// they are movies or TV series. The parser service (Stories 2-3, 3-1) is responsible
// for determining the media type and converting to Series records if needed.
func (s *ScannerService) processVideoFile(ctx context.Context, resolvedPath string, pendingMovies *[]*models.Movie) error {
	// Get file info for size
	info, err := os.Stat(resolvedPath)
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	// Check for existing record (duplicate detection)
	existing, err := s.movieRepo.FindByFilePath(ctx, resolvedPath)
	if err != nil {
		return fmt.Errorf("failed to check for existing record: %w", err)
	}

	if existing != nil {
		// File already in DB — check if file size changed or mtime is newer
		sizeChanged := !existing.FileSize.Valid || existing.FileSize.Int64 != info.Size()
		mtimeNewer := info.ModTime().After(existing.UpdatedAt)

		if !sizeChanged && !mtimeNewer {
			// No change, skip
			s.mu.Lock()
			s.progress.FilesSkipped++
			s.mu.Unlock()
			return nil
		}

		// File changed (size or mtime) — update the record and reset parse status
		existing.FileSize = sql.NullInt64{Int64: info.Size(), Valid: true}
		existing.ParseStatus = models.ParseStatusPending
		existing.UpdatedAt = time.Now()
		if err := s.movieRepo.Update(ctx, existing); err != nil {
			return fmt.Errorf("failed to update movie record: %w", err)
		}
		s.mu.Lock()
		s.progress.FilesUpdated++
		s.mu.Unlock()
		return nil
	}

	// New file — create a movie record with pending status
	movie := &models.Movie{
		ID:             uuid.New().String(),
		Title:          filepath.Base(resolvedPath),
		FilePath:       sql.NullString{String: resolvedPath, Valid: true},
		FileSize:       sql.NullInt64{Int64: info.Size(), Valid: true},
		ParseStatus:    models.ParseStatusPending,
		SubtitleStatus: models.SubtitleStatusNotSearched,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	*pendingMovies = append(*pendingMovies, movie)

	// Batch flush every 100 files
	if len(*pendingMovies) >= 100 {
		if err := s.flushBatch(ctx, pendingMovies); err != nil {
			return err
		}
	}

	s.mu.Lock()
	s.progress.FilesCreated++
	s.mu.Unlock()

	return nil
}

// flushBatch inserts pending movies via BulkCreate and resets the slice
func (s *ScannerService) flushBatch(ctx context.Context, pendingMovies *[]*models.Movie) error {
	if len(*pendingMovies) == 0 {
		return nil
	}
	if err := s.movieRepo.BulkCreate(ctx, *pendingMovies); err != nil {
		return fmt.Errorf("failed to bulk create movies: %w", err)
	}
	s.logger.Info("batch inserted movies", "count", len(*pendingMovies))
	*pendingMovies = nil
	return nil
}

// broadcastProgress sends the current scan progress as an SSE event
func (s *ScannerService) broadcastProgress() {
	if s.sseHub == nil {
		return
	}

	s.mu.Lock()
	progress := s.progress
	s.mu.Unlock()

	event := sse.Event{
		ID:   uuid.New().String(),
		Type: sse.EventScanProgress,
		Data: progress,
	}
	s.sseHub.Broadcast(event)
}

// detectRemovedFiles checks all movies with file paths and marks those
// whose files no longer exist on disk as removed (IsRemoved=true).
func (s *ScannerService) detectRemovedFiles(ctx context.Context) (int, error) {
	movies, err := s.movieRepo.FindAllWithFilePath(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to query movies with file paths: %w", err)
	}

	removedCount := 0
	for i := range movies {
		movie := &movies[i]
		if !movie.FilePath.Valid || movie.FilePath.String == "" {
			continue
		}

		_, err := os.Stat(movie.FilePath.String)
		if err == nil {
			// File exists, skip
			continue
		}
		if !os.IsNotExist(err) {
			// Some other error (permissions, etc.) — log but don't mark as removed
			s.logger.Warn("error checking file existence", "path", movie.FilePath.String, "error", err)
			continue
		}

		// File does not exist — mark as removed
		movie.IsRemoved = true
		movie.UpdatedAt = time.Now()
		if err := s.movieRepo.Update(ctx, movie); err != nil {
			s.logger.Error("failed to mark movie as removed", "id", movie.ID, "path", movie.FilePath.String, "error", err)
			continue
		}
		removedCount++
		s.logger.Info("marked movie as removed (file not found)", "id", movie.ID, "path", movie.FilePath.String)
	}

	return removedCount, nil
}

// buildResult creates a ScanResult from the current progress
func (s *ScannerService) buildResult(startedAt time.Time) *ScanResult {
	s.mu.Lock()
	defer s.mu.Unlock()

	completedAt := time.Now()
	return &ScanResult{
		FilesFound:   s.progress.FilesFound,
		FilesCreated: s.progress.FilesCreated,
		FilesUpdated: s.progress.FilesUpdated,
		FilesSkipped: s.progress.FilesSkipped,
		FilesRemoved: s.progress.FilesRemoved,
		ErrorCount:   s.progress.ErrorCount,
		Duration:     completedAt.Sub(startedAt).String(),
		StartedAt:    startedAt,
		CompletedAt:  completedAt,
	}
}
