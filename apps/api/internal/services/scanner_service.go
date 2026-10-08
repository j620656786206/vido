package services

import (
	"context"
	"encoding/json"
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
	"github.com/vido/api/internal/parser"
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
	FilesFound     int       `json:"files_found"`
	FilesCreated   int       `json:"files_created"`
	FilesUpdated   int       `json:"files_updated"`
	FilesSkipped   int       `json:"files_skipped"`
	FilesRemoved   int       `json:"files_removed"`
	FilesUnmatched int       `json:"files_unmatched"`
	ErrorCount     int       `json:"error_count"`
	CurrentFile    string    `json:"current_file"`
	PercentDone    int       `json:"percent_done"`
	IsActive       bool      `json:"is_active"`
	StartedAt      time.Time `json:"started_at,omitempty"`
}

// ScanResult contains the outcome of a completed scan operation
type ScanResult struct {
	FilesFound     int       `json:"files_found"`
	FilesCreated   int       `json:"files_created"`
	FilesUpdated   int       `json:"files_updated"`
	FilesSkipped   int       `json:"files_skipped"`
	FilesRemoved   int       `json:"files_removed"`
	FilesUnmatched int       `json:"files_unmatched"`
	ErrorCount     int       `json:"error_count"`
	Duration       string    `json:"duration"`
	StartedAt      time.Time `json:"started_at"`
	CompletedAt    time.Time `json:"completed_at"`
}

// LastScanSummary is the most recent COMPLETED scan, as the settings page's
// 「上次掃描」 line shows it (bugfix-last-scan-never-shown AC #1 [@contract-v1]).
// A cancelled scan never replaces it.
type LastScanSummary struct {
	CompletedAt time.Time `json:"completed_at"`
	FilesFound  int       `json:"files_found"`
	DurationMs  int64     `json:"duration_ms"`
}

// settingsKeyScanLastResult persists LastScanSummary (JSON) across restarts.
const settingsKeyScanLastResult = "scan_last_result"

// ScannerService handles recursive folder scanning and video file discovery
type ScannerService struct {
	movieRepo     repository.MovieRepositoryInterface
	seriesRepo    repository.SeriesRepositoryInterface
	episodeRepo   repository.EpisodeRepositoryInterface
	libraryRepo   repository.MediaLibraryRepositoryInterface
	ingestService *MediaIngestService
	parserService ParserServiceInterface
	mediaDirs     []string // Fallback dirs from VIDO_MEDIA_DIRS env var
	sseHub        *sse.Hub
	logger        *slog.Logger

	mu             sync.Mutex
	isScanning     bool
	cancelChan     chan struct{}
	progress       ScanProgress
	onScanComplete func()

	// Per-folder walk evidence for the removed-file pass: videos seen
	// (including ones an earlier folder already claimed) and access errors
	// while walking the CURRENT folder (bugfix-scan-mount-drop-hides-movies).
	rootWalkVideos int
	rootWalkErrors int

	// Last completed scan: loaded from settings on first read, then kept in
	// memory and re-saved on every completion (bugfix-last-scan-never-shown).
	settingsRepo   repository.SettingsRepositoryInterface
	lastScan       *LastScanSummary
	lastScanLoaded bool
}

// SetSettingsRepo wires where the last completed scan is persisted. Without
// it the record lives in memory only (lost on restart).
func (s *ScannerService) SetSettingsRepo(repo repository.SettingsRepositoryInterface) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.settingsRepo = repo
	s.lastScanLoaded = false
}

// GetLastScan returns the most recent completed scan, or nil if none has ever
// finished. Safe for concurrent use. The settings read happens OUTSIDE s.mu
// (a busy SQLite must not stall progress updates or CancelScan), detached from
// the request's cancellation, and a transient read error is retried on the
// next call instead of being remembered as "never scanned".
func (s *ScannerService) GetLastScan(ctx context.Context) *LastScanSummary {
	s.mu.Lock()
	loaded, repo := s.lastScanLoaded, s.settingsRepo
	s.mu.Unlock()

	if !loaded {
		var stored *LastScanSummary
		settled := repo == nil
		if repo != nil {
			raw, err := repo.GetString(context.WithoutCancel(ctx), settingsKeyScanLastResult)
			switch {
			case err == nil:
				settled = true
				var decoded LastScanSummary
				if jerr := json.Unmarshal([]byte(raw), &decoded); jerr != nil || decoded.CompletedAt.IsZero() {
					s.logger.Warn("ignoring unreadable last scan record", "value", raw)
				} else {
					stored = &decoded
				}
			case isSettingNotFound(err):
				settled = true // never scanned
			default:
				s.logger.Warn("could not read last scan record; will retry", "error", err)
			}
		}
		s.mu.Lock()
		// A scan that completed while we were reading wins over the stored copy.
		if !s.lastScanLoaded && settled {
			s.lastScanLoaded = true
			if s.lastScan == nil {
				s.lastScan = stored
			}
		}
		s.mu.Unlock()
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.lastScan == nil {
		return nil
	}
	cp := *s.lastScan
	return &cp
}

// recordLastScan remembers a completed scan and persists it. A failed save is
// logged only — the scan itself succeeded.
func (s *ScannerService) recordLastScan(result *ScanResult) {
	summary := LastScanSummary{
		CompletedAt: result.CompletedAt,
		FilesFound:  result.FilesFound,
		DurationMs:  result.CompletedAt.Sub(result.StartedAt).Milliseconds(),
	}
	s.mu.Lock()
	s.lastScan = &summary
	s.lastScanLoaded = true
	repo := s.settingsRepo
	s.mu.Unlock()

	if repo == nil {
		return
	}
	raw, err := json.Marshal(summary)
	if err != nil {
		s.logger.Error("failed to encode last scan record", "error", err)
		return
	}
	if err := repo.SetString(context.Background(), settingsKeyScanLastResult, string(raw)); err != nil {
		s.logger.Error("failed to persist last scan record", "error", err)
	}
}

// AppendOnScanComplete chains fn AFTER whatever callback is already set, so
// a late-wired consumer (sub-7-5b's official-subtitle miner) does not have to
// know about — or replace — the enrichment and auto-generation hooks that
// were composed earlier in main.go. Order: existing callback first, fn after.
func (s *ScannerService) AppendOnScanComplete(fn func()) {
	if fn == nil {
		return
	}
	prev := s.onScanComplete
	if prev == nil {
		s.SetOnScanComplete(fn)
		return
	}
	s.SetOnScanComplete(func() {
		prev()
		fn()
	})
}

// SetOnScanComplete sets a callback to be invoked after a successful scan.
func (s *ScannerService) SetOnScanComplete(fn func()) {
	s.onScanComplete = fn
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

// SetLibraryRepo sets the media library repository for DB-based directory reading.
// When set, the scanner reads libraries from DB instead of mediaDirs.
func (s *ScannerService) SetLibraryRepo(repo repository.MediaLibraryRepositoryInterface) {
	s.libraryRepo = repo
}

// SetEpisodeRepo sets the episode repository for series file_size aggregation (Story 9c-3)
func (s *ScannerService) SetEpisodeRepo(repo repository.EpisodeRepositoryInterface) {
	s.episodeRepo = repo
}

// SetTVIngest enables TV routing. Without it the scanner keeps its historical behaviour
// of writing every file to `movies`, which is what left series/seasons/episodes empty.
func (s *ScannerService) SetTVIngest(ingest *MediaIngestService, parserService ParserServiceInterface) {
	s.ingestService = ingest
	s.parserService = parserService
}

// resolveMediaType decides whether a scanned file is a movie or a TV episode.
//
// The library's content_type wins when the user has declared one — that is the whole point
// of the manual per-folder type designation. It is empty when the scan is running off the
// VIDO_MEDIA_DIRS fallback (which carries no type), and then the filename decides:
// tv_parser recognises SxxExx / 1x05 / 第N集 / anime episode forms.
func (s *ScannerService) resolveMediaType(contentType, filePath string) (isTV bool, parseResult *parser.ParseResult) {
	switch models.MediaLibraryContentType(contentType) {
	case models.ContentTypeSeries:
		return true, s.parseFilename(filePath)
	case models.ContentTypeMovie:
		return false, nil
	}

	// No declared type → let the filename decide.
	result := s.parseFilename(filePath)
	if result != nil && result.MediaType == parser.MediaTypeTVShow {
		return true, result
	}
	return false, nil
}

func (s *ScannerService) parseFilename(filePath string) *parser.ParseResult {
	if s.parserService == nil {
		return nil
	}
	return s.parserService.ParseFilename(filepath.Base(filePath))
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

	// Resolve scan directories from DB libraries or fallback to env var
	type scanDir struct {
		path        string
		libraryID   string
		contentType string
	}
	var dirs []scanDir

	if s.libraryRepo != nil {
		libraries, err := s.libraryRepo.GetAllWithPathsAndCounts(ctx)
		if err != nil {
			s.logger.Error("failed to read libraries from DB, falling back to env var", "error", err)
		} else {
			for _, lib := range libraries {
				for _, p := range lib.Paths {
					dirs = append(dirs, scanDir{path: p.Path, libraryID: lib.ID, contentType: string(lib.ContentType)})
				}
			}
		}
	}
	if len(dirs) == 0 {
		// Fallback to env var
		for _, d := range s.mediaDirs {
			dirs = append(dirs, scanDir{path: d})
		}
		if len(dirs) > 0 {
			s.logger.Warn("Using VIDO_MEDIA_DIRS fallback — configure libraries in Settings for per-folder content type")
		}
	}

	s.logger.Info("scan started", "dir_count", len(dirs))

	// Check cancellation after library resolution
	select {
	case <-s.cancelChan:
		s.logger.Info("scan cancelled before directory walk")
		result := s.buildResult(startedAt)
		s.broadcastScanCancelled(result)
		return result, nil
	default:
	}

	// Track seen resolved paths to deduplicate across directories
	seenPaths := make(map[string]bool)
	walkedDirs := 0 // folders that existed and were walked (bugfix-last-scan-never-shown)
	// Folders this scan cannot vouch for — unreachable, failed mid-walk, or
	// reachable but with no video in them (Docker's view of an unmounted NAS
	// share is an EMPTY folder). Nothing under them is marked removed
	// (bugfix-scan-mount-drop-hides-movies).
	var untrustedRoots []string
	untrustedLibraries := map[string]bool{}
	var pendingMovies []*models.Movie
	distrust := func(sd scanDir) {
		untrustedRoots = append(untrustedRoots, rootForms(sd.path)...)
		if sd.libraryID != "" {
			untrustedLibraries[sd.libraryID] = true
		}
	}

	for _, sd := range dirs {
		dir := sd.path
		// Check cancellation
		select {
		case <-s.cancelChan:
			s.logger.Info("scan cancelled before walking directory", "dir", dir)
			result := s.buildResult(startedAt)
			s.broadcastScanCancelled(result)
			return result, nil
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
			distrust(sd)
			continue
		}
		if !info.IsDir() {
			s.logger.Warn("SCANNER_PATH_NOT_FOUND: configured path is not a directory", "path", dir)
			s.mu.Lock()
			s.progress.ErrorCount++
			s.mu.Unlock()
			distrust(sd)
			continue
		}

		walkedDirs++
		s.mu.Lock()
		s.rootWalkVideos, s.rootWalkErrors = 0, 0
		s.mu.Unlock()
		err = s.walkDirectory(ctx, dir, sd.libraryID, sd.contentType, seenPaths, &pendingMovies)
		if err != nil {
			s.logger.Error("SCANNER_PARSE_FAILED: error walking directory", "path", dir, "error", err)
			s.mu.Lock()
			s.progress.ErrorCount++
			s.mu.Unlock()
		}
		s.mu.Lock()
		videos, walkErrors := s.rootWalkVideos, s.rootWalkErrors
		s.mu.Unlock()
		// No video at all (Docker's empty mount point) or any part of the
		// folder unreadable (a share dropping mid-walk, a dead symlink into
		// the NAS) — this scan cannot say what is really gone from it.
		if err != nil || videos == 0 || walkErrors > 0 {
			distrust(sd)
		}
	}

	// Flush remaining pending movies (flushBatch accounts its own failure)
	if len(pendingMovies) > 0 {
		_ = s.flushBatch(ctx, &pendingMovies)
	}

	// Detect removed files (Story 7-2: incremental scan)
	removedCount, err := s.detectRemovedFiles(ctx, untrustedRoots, untrustedLibraries)
	if err != nil {
		s.logger.Error("failed to detect removed files", "error", err)
	}
	// Episodes too (disc-2026-10-episode-rows-outlive-deleted-files): a
	// deleted episode file goes back to "known episode, no local file".
	removedEpisodes, err := s.detectRemovedEpisodeFiles(ctx, untrustedRoots, untrustedLibraries)
	if err != nil {
		s.logger.Error("failed to detect removed episode files", "error", err)
	}
	if total := removedCount + removedEpisodes; total > 0 {
		s.mu.Lock()
		s.progress.FilesRemoved = total
		s.mu.Unlock()
		s.logger.Info("detected removed files", "count", total, "movies", removedCount, "episodes", removedEpisodes)
	}

	// Aggregate series file sizes (Story 9c-3 AC #8)
	if err := s.aggregateSeriesFileSizes(ctx); err != nil {
		s.logger.Error("failed to aggregate series file sizes", "error", err)
	}

	result := s.buildResult(startedAt)

	// Broadcast final progress update
	s.broadcastProgress()

	// Broadcast completion or cancellation event
	wasCancelled := false
	select {
	case <-s.cancelChan:
		wasCancelled = true
	default:
	}
	if wasCancelled {
		s.broadcastScanCancelled(result)
	} else {
		// Every configured folder unreachable (e.g. the NAS mount dropped) is
		// not a finished scan of the library — keep the previous record rather
		// than show 「0 檔案」 as if the library were empty.
		if len(dirs) == 0 || walkedDirs > 0 {
			s.recordLastScan(result)
		}
		s.broadcastScanComplete(result)
		// Trigger post-scan enrichment if configured
		if s.onScanComplete != nil && (result.FilesCreated > 0 || result.FilesUpdated > 0) {
			s.onScanComplete()
		}
	}

	s.logger.Info("scan completed",
		"files_found", result.FilesFound,
		"files_created", result.FilesCreated,
		"files_updated", result.FilesUpdated,
		"files_skipped", result.FilesSkipped,
		"files_unmatched", result.FilesUnmatched,
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
func (s *ScannerService) walkDirectory(ctx context.Context, root string, libraryID string, contentType string, seenPaths map[string]bool, pendingMovies *[]*models.Movie) error {
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
				s.rootWalkErrors++
				s.mu.Unlock()
				return nil // continue scanning other paths
			}
			s.logger.Error("error accessing path", "path", path, "error", err)
			s.mu.Lock()
			s.progress.ErrorCount++
			s.rootWalkErrors++
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
			s.rootWalkErrors++ // a dead link into an unreachable share
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
			s.rootWalkVideos++ // present — just claimed by an earlier folder
			s.mu.Unlock()
			return nil
		}
		seenPaths[resolvedPath] = true

		s.mu.Lock()
		s.rootWalkVideos++
		s.progress.FilesFound++
		s.progress.CurrentFile = resolvedPath
		filesFound := s.progress.FilesFound
		s.mu.Unlock()

		// Broadcast progress every 10 files
		if filesFound%10 == 0 {
			s.broadcastProgress()
		}

		// Process the video file
		err = s.processVideoFile(ctx, resolvedPath, root, libraryID, contentType, pendingMovies)
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

// processVideoFile routes one scanned file to the table it belongs in: movies, or
// series/seasons/episodes. Historically it wrote every file as a Movie and left the
// media-type decision to a "parser service ... converting to Series records if needed"
// that was never written — which is why a TV library came out as one movies row per
// episode, each stamped with the whole series' TMDb metadata.
func (s *ScannerService) processVideoFile(ctx context.Context, resolvedPath, scanRoot, libraryID, contentType string, pendingMovies *[]*models.Movie) error {
	// Get file info for size
	info, err := os.Stat(resolvedPath)
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	if s.ingestService != nil {
		if isTV, parseResult := s.resolveMediaType(contentType, resolvedPath); isTV {
			return s.processTVFile(ctx, resolvedPath, scanRoot, libraryID, parseResult)
		}
	}

	// Check for existing record (duplicate detection)
	existing, err := s.movieRepo.FindByFilePath(ctx, resolvedPath)
	if err != nil {
		return fmt.Errorf("failed to check for existing record: %w", err)
	}

	if existing != nil {
		// The file is here, so a "removed" flag from an earlier scan (e.g. the
		// NAS folder was briefly unreachable) no longer holds. Nothing else
		// ever cleared it, so without this the movie stayed hidden for good
		// (bugfix-scan-mount-drop-hides-movies).
		restored := false
		if existing.IsRemoved {
			if err := s.movieRepo.RestoreRemoved(ctx, existing.ID); err != nil {
				return fmt.Errorf("failed to restore removed movie: %w", err)
			}
			restored = true
			s.logger.Info("restored movie whose file is back", "id", existing.ID, "path", resolvedPath)
		}

		// File already in DB — check if file size changed or mtime is newer
		sizeChanged := !existing.FileSize.Valid || existing.FileSize.Int64 != info.Size()
		mtimeNewer := info.ModTime().After(existing.UpdatedAt)

		if restored && !sizeChanged && !mtimeNewer {
			s.mu.Lock()
			s.progress.FilesUpdated++
			s.mu.Unlock()
			return nil
		}
		if !sizeChanged && !mtimeNewer {
			// No change, skip
			s.mu.Lock()
			s.progress.FilesSkipped++
			s.mu.Unlock()
			return nil
		}

		// File changed (size or mtime) — record the new size and reset the
		// parse status. Narrow write (bugfix-wide-update-stale-copy-other-
		// callers §audit #1): two columns are what this pass learned; the wide
		// Update would also write back the five subtitle columns from a copy
		// loaded a moment ago, during the very scan whose completion starts
		// the free subtitle lane.
		if err := s.movieRepo.UpdateScanFileInfo(ctx, existing.ID, info.Size(), models.ParseStatusPending); err != nil {
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
		FilePath:       models.NewNullString(resolvedPath),
		FileSize:       models.NewNullInt64(info.Size()),
		ParseStatus:    models.ParseStatusPending,
		SubtitleStatus: models.SubtitleStatusNotSearched,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	if libraryID != "" {
		movie.LibraryID = models.NewNullString(libraryID)
	}

	*pendingMovies = append(*pendingMovies, movie)

	// Batch flush every 100 files. FilesCreated is counted inside flushBatch on
	// actual insert success — counting at append time reported rows that a
	// failed flush never wrote (bugfix-scanner-counter-reports-phantom-creates).
	// flushBatch fully accounts its own failure (ErrorCount += batch size), so
	// its error is not propagated — propagating would double-count this file.
	if len(*pendingMovies) >= 100 {
		_ = s.flushBatch(ctx, pendingMovies)
	}

	return nil
}

// processTVFile ingests one episode file into series/seasons/episodes.
//
// It also repairs history: if a `movies` row still claims this file_path, it was written
// by the pre-fix scanner, which mis-filed every episode as a movie. That row is deleted —
// the file is provably in the wrong table, the episode row now owns it, and leaving it
// would show the same episode twice (once in the movie grid, once under its series).
// A soft-delete would not do: FindByFilePath does not filter is_removed, so the next scan
// would resurrect it.
func (s *ScannerService) processTVFile(ctx context.Context, resolvedPath, scanRoot, libraryID string, parseResult *parser.ParseResult) error {
	// A TV file whose episode number cannot be determined used to be upserted
	// with episode 0, collapsing EVERY unparseable file of a series into one
	// phantom row while FilesCreated++ claimed success for each — 411 of the
	// NAS scan's "created" rows never existed. Count it honestly instead
	// (bugfix-scanner-counter-reports-phantom-creates); the sibling entry
	// bugfix-scanner-bracket-prefix-filenames-dropped tracks parsing them.
	if parseResult == nil || parseResult.Episode <= 0 {
		s.logger.Warn("SCANNER_UNMATCHED: cannot determine episode number, file not imported",
			"file_path", resolvedPath)
		s.mu.Lock()
		s.progress.FilesUnmatched++
		s.mu.Unlock()
		return nil
	}

	if stale, err := s.movieRepo.FindByFilePath(ctx, resolvedPath); err != nil {
		return fmt.Errorf("failed to check for a mis-filed movie row: %w", err)
	} else if stale != nil {
		if err := s.movieRepo.Delete(ctx, stale.ID); err != nil {
			return fmt.Errorf("failed to delete mis-filed movie row %s: %w", stale.ID, err)
		}
		s.logger.Info("removed mis-filed movie row for a TV episode",
			"movie_id", stale.ID, "file_path", resolvedPath)
	}

	_, created, err := s.ingestService.IngestEpisodeFile(ctx, resolvedPath, scanRoot, libraryID, parseResult)
	if err != nil {
		return fmt.Errorf("failed to ingest episode: %w", err)
	}

	s.mu.Lock()
	if created {
		s.progress.FilesCreated++
	} else {
		s.progress.FilesUpdated++
	}
	s.mu.Unlock()

	return nil
}

// flushBatch inserts pending movies via BulkCreate and resets the slice. It
// owns the progress accounting for the batch: FilesCreated counts rows that
// were actually written, and a failed batch counts every one of its files into
// ErrorCount (they were never created). Callers must NOT add their own counts
// for a flushBatch outcome.
func (s *ScannerService) flushBatch(ctx context.Context, pendingMovies *[]*models.Movie) error {
	n := len(*pendingMovies)
	if n == 0 {
		return nil
	}
	if err := s.movieRepo.BulkCreate(ctx, *pendingMovies); err != nil {
		s.logger.Error("failed to bulk create movies — batch not written",
			"count", n, "error", err)
		s.mu.Lock()
		s.progress.ErrorCount += n
		s.mu.Unlock()
		*pendingMovies = nil
		return fmt.Errorf("failed to bulk create movies: %w", err)
	}
	s.logger.Info("batch inserted movies", "count", n)
	s.mu.Lock()
	s.progress.FilesCreated += n
	s.mu.Unlock()
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

// broadcastScanComplete sends a scan_complete SSE event with final result data.
// NOTE: Uses snake_case keys (files_found, error_count) to match frontend
// useScanProgress.ts expectations at lines 230-233. This differs from
// broadcastProgress() which sends the ScanProgress struct with camelCase JSON tags.
func (s *ScannerService) broadcastScanComplete(result *ScanResult) {
	if s.sseHub == nil {
		return
	}
	s.sseHub.Broadcast(sse.Event{
		ID:   uuid.New().String(),
		Type: sse.EventScanComplete,
		Data: map[string]interface{}{
			"files_found":     result.FilesFound,
			"files_created":   result.FilesCreated,
			"files_updated":   result.FilesUpdated,
			"files_skipped":   result.FilesSkipped,
			"files_removed":   result.FilesRemoved,
			"files_unmatched": result.FilesUnmatched,
			"error_count":     result.ErrorCount,
			"duration":        result.Duration,
		},
	})
}

// broadcastScanCancelled sends a scan_cancelled SSE event.
func (s *ScannerService) broadcastScanCancelled(result *ScanResult) {
	if s.sseHub == nil {
		return
	}
	s.sseHub.Broadcast(sse.Event{
		ID:   uuid.New().String(),
		Type: sse.EventScanCancelled,
		Data: map[string]interface{}{
			"files_found": result.FilesFound,
			"error_count": result.ErrorCount,
		},
	})
}

// detectRemovedFiles checks all movies with file paths and marks those
// whose files no longer exist on disk as removed (IsRemoved=true).
func (s *ScannerService) detectRemovedFiles(ctx context.Context, untrustedRoots []string, untrustedLibraries map[string]bool) (int, error) {
	movies, err := s.movieRepo.FindAllWithFilePath(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to query movies with file paths: %w", err)
	}

	removedCount := 0
	protected := 0
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
		// Its folder was unreachable or came back empty this scan — that is a
		// mount problem, not a deleted film (bugfix-scan-mount-drop-hides-movies).
		// Matched by path AND by library: a folder that is itself a symlink
		// into the NAS stores its movies under the TARGET path, which a dead
		// link can no longer tell us.
		if underAnyRoot(movie.FilePath.String, untrustedRoots) ||
			(movie.LibraryID.Valid && untrustedLibraries[movie.LibraryID.String]) {
			protected++
			continue
		}

		// File does not exist — mark as removed. Narrow write (bugfix-wide-
		// update-stale-copy-other-callers §audit #2): `movies` was loaded in
		// full ABOVE and every row since has cost an os.Stat on the NAS, so
		// this copy can be seconds stale — and this pass runs right after
		// scan-complete, concurrently with enrichment and the free subtitle
		// lane. Writing the whole row back reverted their work.
		if err := s.movieRepo.MarkRemoved(ctx, movie.ID); err != nil {
			s.logger.Error("failed to mark movie as removed", "id", movie.ID, "path", movie.FilePath.String, "error", err)
			continue
		}
		removedCount++
		s.logger.Info("marked movie as removed (file not found)", "id", movie.ID, "path", movie.FilePath.String)
	}

	if protected > 0 {
		s.logger.Warn("SCANNER_ROOT_UNTRUSTED: kept movies whose folder was unreachable or empty this scan — not marking them removed",
			"movies_kept", protected, "folders", untrustedRoots)
	}
	return removedCount, nil
}

// episodeRemovalRepo is the slice of the episode repository the removed-file
// pass needs. *repository.EpisodeRepository satisfies it; the scanner's
// episodeRepo is asserted against it so the wide interface (and every mock of
// it) does not grow.
type episodeRemovalRepo interface {
	FindFilesForRemovalCheck(ctx context.Context) ([]repository.EpisodeFileRef, error)
	ClearMissingFile(ctx context.Context, episodeID, filePath string) (bool, error)
}

// detectRemovedEpisodeFiles is detectRemovedFiles for episodes
// (disc-2026-10-episode-rows-outlive-deleted-files, ⚖️ Alexyu option A):
// an episode whose file no longer exists loses its file_path — the state
// every reader already treats as "no local file" — instead of gaining an
// is_removed flag each of them would have to learn. The next scan that finds
// the file again writes the path back onto the same row (Upsert matches
// series + season + episode), so there is nothing to restore by hand.
//
// The same mount protection as movies: nothing under a folder this scan
// could not vouch for is touched, matched by path and by the series' library.
func (s *ScannerService) detectRemovedEpisodeFiles(ctx context.Context, untrustedRoots []string, untrustedLibraries map[string]bool) (int, error) {
	repo, ok := s.episodeRepo.(episodeRemovalRepo)
	if !ok {
		return 0, nil
	}
	refs, err := repo.FindFilesForRemovalCheck(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to query episode files: %w", err)
	}

	cleared, protected := 0, 0
	for _, ref := range refs {
		if ctx.Err() != nil {
			return cleared, ctx.Err()
		}
		_, err := os.Stat(ref.FilePath)
		if err == nil {
			continue
		}
		if !os.IsNotExist(err) {
			s.logger.Warn("error checking episode file existence", "path", ref.FilePath, "error", err)
			continue
		}
		if underAnyRoot(ref.FilePath, untrustedRoots) || (ref.LibraryID != "" && untrustedLibraries[ref.LibraryID]) {
			protected++
			continue
		}
		done, err := repo.ClearMissingFile(ctx, ref.ID, ref.FilePath)
		if err != nil {
			s.logger.Error("failed to clear missing episode file", "id", ref.ID, "path", ref.FilePath, "error", err)
			continue
		}
		if !done {
			continue // re-ingested at another path while we looked — keep it
		}
		cleared++
		s.logger.Info("episode file gone — now shown as no local file", "id", ref.ID, "path", ref.FilePath)
	}

	if protected > 0 {
		s.logger.Warn("SCANNER_ROOT_UNTRUSTED: kept episodes whose folder was unreachable or empty this scan — not clearing their files",
			"episodes_kept", protected, "folders", untrustedRoots)
	}
	return cleared, nil
}

// rootForms returns the spellings a movie's stored (symlink-resolved,
// absolute) file_path may start with for this configured folder.
func rootForms(dir string) []string {
	forms := []string{filepath.Clean(dir)}
	if abs, err := filepath.Abs(dir); err == nil && abs != forms[0] {
		forms = append(forms, abs)
	}
	// An unreachable folder cannot be resolved itself; resolve its nearest
	// existing parent and re-attach the rest, so a symlinked mount parent
	// still matches the stored paths.
	rest := ""
	for cur := filepath.Clean(dir); ; {
		if resolved, err := filepath.EvalSymlinks(cur); err == nil {
			if abs, err := filepath.Abs(filepath.Join(resolved, rest)); err == nil && abs != forms[0] {
				forms = append(forms, abs)
			}
			break
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			break
		}
		rest = filepath.Join(filepath.Base(cur), rest)
		cur = parent
	}
	return forms
}

// underAnyRoot reports whether path lies inside one of roots.
func underAnyRoot(path string, roots []string) bool {
	p := filepath.Clean(path)
	for _, r := range roots {
		if p == r || strings.HasPrefix(p, strings.TrimSuffix(r, string(filepath.Separator))+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

// buildResult creates a ScanResult from the current progress
func (s *ScannerService) buildResult(startedAt time.Time) *ScanResult {
	s.mu.Lock()
	defer s.mu.Unlock()

	completedAt := time.Now()
	return &ScanResult{
		FilesFound:     s.progress.FilesFound,
		FilesCreated:   s.progress.FilesCreated,
		FilesUpdated:   s.progress.FilesUpdated,
		FilesSkipped:   s.progress.FilesSkipped,
		FilesRemoved:   s.progress.FilesRemoved,
		FilesUnmatched: s.progress.FilesUnmatched,
		ErrorCount:     s.progress.ErrorCount,
		Duration:       completedAt.Sub(startedAt).String(),
		StartedAt:      startedAt,
		CompletedAt:    completedAt,
	}
}

// aggregateSeriesFileSizes recalculates file_size for all series by summing
// their episode file sizes from the filesystem. (Story 9c-3 AC #8)
func (s *ScannerService) aggregateSeriesFileSizes(ctx context.Context) error {
	if s.episodeRepo == nil || s.seriesRepo == nil {
		return nil
	}

	// Get all series
	allSeries, _, err := s.seriesRepo.List(ctx, repository.ListParams{Page: 1, PageSize: 10000})
	if err != nil {
		return fmt.Errorf("list series: %w", err)
	}

	updated := 0
	for i := range allSeries {
		series := &allSeries[i]
		episodes, err := s.episodeRepo.FindBySeriesID(ctx, series.ID)
		if err != nil {
			s.logger.Warn("failed to get episodes for series file_size",
				"series_id", series.ID, "error", err)
			continue
		}

		// Count episodes with file paths first to avoid unnecessary filesystem calls
		var pathCount int
		for _, ep := range episodes {
			if ep.FilePath.Valid && ep.FilePath.String != "" {
				pathCount++
			}
		}
		if pathCount == 0 {
			continue
		}

		var totalSize int64
		for _, ep := range episodes {
			if ep.FilePath.Valid && ep.FilePath.String != "" {
				info, err := os.Stat(ep.FilePath.String)
				if err == nil {
					totalSize += info.Size()
				}
			}
		}

		if totalSize > 0 {
			// Narrow write (bugfix-wide-update-stale-copy-other-callers §audit
			// #3): `allSeries` was loaded once above; by the time row N is
			// reached every earlier series has cost one stat per episode. The
			// wide Update would write parse_status / poster_path / credits back
			// from that stale copy over what enrichment wrote meanwhile.
			if err := s.seriesRepo.UpdateFileSize(ctx, series.ID, totalSize); err != nil {
				s.logger.Warn("failed to update series file_size",
					"series_id", series.ID, "error", err)
			} else {
				updated++
			}
		}
	}

	if updated > 0 {
		s.logger.Info("series file sizes aggregated", "updated", updated)
	}
	return nil
}
