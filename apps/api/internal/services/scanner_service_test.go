package services

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/sse"
	"github.com/vido/api/internal/testutil"
)

// helper to create a scanner service with mocks
func setupScannerService(t *testing.T, mediaDirs []string) (*ScannerService, *testutil.MockMovieRepository, *sse.Hub) {
	t.Helper()
	movieRepo := new(testutil.MockMovieRepository)
	seriesRepo := new(testutil.MockSeriesRepository)

	hub := sse.NewHub()
	t.Cleanup(func() { hub.Close() })

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	svc := NewScannerService(movieRepo, seriesRepo, mediaDirs, hub, logger)

	// Default mock for FindAllWithFilePath (called by detectRemovedFiles in StartScan)
	// Tests can override this by adding their own expectation before calling StartScan
	movieRepo.On("FindAllWithFilePath", mock.Anything).Maybe().Return([]models.Movie{}, nil)

	return svc, movieRepo, hub
}

// createVideoFiles creates fake video files in the given directory and returns their paths
func createVideoFiles(t *testing.T, dir string, names []string) []string {
	t.Helper()
	var paths []string
	for _, name := range names {
		p := filepath.Join(dir, name)
		// Create any needed subdirectories
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("fake video content"), 0644); err != nil {
			t.Fatal(err)
		}
		paths = append(paths, p)
	}
	return paths
}

func TestScannerService_StartScan_Success(t *testing.T) {
	dir := t.TempDir()
	createVideoFiles(t, dir, []string{
		"movie1.mkv",
		"movie2.mp4",
		"subdir/movie3.avi",
	})

	svc, movieRepo, _ := setupScannerService(t, []string{dir})

	// FindByFilePath returns nil for all (no duplicates)
	movieRepo.On("FindByFilePath", mock.Anything, mock.AnythingOfType("string")).Return(nil, nil)
	// BulkCreate succeeds
	movieRepo.On("BulkCreate", mock.Anything, mock.AnythingOfType("[]*models.Movie")).Return(nil)

	ctx := context.Background()
	result, err := svc.StartScan(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 3, result.FilesFound)
	assert.Equal(t, 3, result.FilesCreated)
	assert.Equal(t, 0, result.FilesSkipped)
	assert.Equal(t, 0, result.ErrorCount)
	assert.NotEmpty(t, result.Duration)

	movieRepo.AssertCalled(t, "BulkCreate", mock.Anything, mock.AnythingOfType("[]*models.Movie"))
}

func TestScannerService_StartScan_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	// dir is empty — no files

	svc, movieRepo, _ := setupScannerService(t, []string{dir})

	ctx := context.Background()
	result, err := svc.StartScan(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 0, result.FilesFound)
	assert.Equal(t, 0, result.FilesCreated)
	assert.Equal(t, 0, result.ErrorCount)

	movieRepo.AssertNotCalled(t, "BulkCreate", mock.Anything, mock.Anything)
}

func TestScannerService_StartScan_InvalidPath(t *testing.T) {
	nonExistent := filepath.Join(t.TempDir(), "does_not_exist")

	svc, _, _ := setupScannerService(t, []string{nonExistent})

	ctx := context.Background()
	result, err := svc.StartScan(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 0, result.FilesFound)
	assert.Equal(t, 1, result.ErrorCount) // one error for the invalid path
}

func TestScannerService_StartScan_PermissionDenied(t *testing.T) {
	dir := t.TempDir()
	restrictedDir := filepath.Join(dir, "restricted")
	err := os.Mkdir(restrictedDir, 0000)
	if err != nil {
		t.Skip("cannot create restricted directory")
	}
	t.Cleanup(func() { os.Chmod(restrictedDir, 0755) })

	// Put a video file in the parent that we CAN read
	createVideoFiles(t, dir, []string{"readable.mkv"})

	svc, movieRepo, _ := setupScannerService(t, []string{dir})

	movieRepo.On("FindByFilePath", mock.Anything, mock.AnythingOfType("string")).Return(nil, nil)
	movieRepo.On("BulkCreate", mock.Anything, mock.AnythingOfType("[]*models.Movie")).Return(nil)

	ctx := context.Background()
	result, err := svc.StartScan(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	// Should have found the readable file and logged errors for restricted dir
	assert.Equal(t, 1, result.FilesCreated)
	assert.GreaterOrEqual(t, result.ErrorCount, 1)
}

func TestScannerService_StartScan_DuplicateDetection(t *testing.T) {
	dir := t.TempDir()
	paths := createVideoFiles(t, dir, []string{"existing.mkv"})

	svc, movieRepo, _ := setupScannerService(t, []string{dir})

	// Simulate existing record with same file size
	// Use EvalSymlinks to match what the scanner does (macOS /var -> /private/var)
	resolvedPath, _ := filepath.EvalSymlinks(paths[0])
	resolvedPath, _ = filepath.Abs(resolvedPath)
	existingMovie := &models.Movie{
		ID:        "existing-id",
		Title:     "existing.mkv",
		FilePath:  models.NewNullString(resolvedPath),
		FileSize:  models.NewNullInt64(int64(len("fake video content"))),
		UpdatedAt: time.Now().Add(1 * time.Hour), // future time so mtime is not newer
	}
	movieRepo.On("FindByFilePath", mock.Anything, resolvedPath).Return(existingMovie, nil)

	ctx := context.Background()
	result, err := svc.StartScan(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 1, result.FilesFound)
	assert.Equal(t, 0, result.FilesCreated)
	assert.Equal(t, 1, result.FilesSkipped) // skipped because same size and mtime not newer
	assert.Equal(t, 0, result.FilesUpdated)

	movieRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
	movieRepo.AssertNotCalled(t, "BulkCreate", mock.Anything, mock.Anything)
}

func TestScannerService_StartScan_DuplicateDetection_SizeChanged(t *testing.T) {
	dir := t.TempDir()
	paths := createVideoFiles(t, dir, []string{"changed.mkv"})

	svc, movieRepo, _ := setupScannerService(t, []string{dir})

	resolvedPath, _ := filepath.EvalSymlinks(paths[0])
	resolvedPath, _ = filepath.Abs(resolvedPath)
	existingMovie := &models.Movie{
		ID:       "existing-id",
		Title:    "changed.mkv",
		FilePath: models.NewNullString(resolvedPath),
		FileSize: models.NewNullInt64(999), // different size
	}
	movieRepo.On("FindByFilePath", mock.Anything, resolvedPath).Return(existingMovie, nil)
	movieRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.Movie")).Return(nil)

	ctx := context.Background()
	result, err := svc.StartScan(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 1, result.FilesFound)
	assert.Equal(t, 0, result.FilesCreated)
	assert.Equal(t, 1, result.FilesUpdated)
	assert.Equal(t, 0, result.FilesSkipped)

	movieRepo.AssertCalled(t, "Update", mock.Anything, mock.AnythingOfType("*models.Movie"))
}

func TestScannerService_StartScan_VideoFormatFiltering(t *testing.T) {
	dir := t.TempDir()
	createVideoFiles(t, dir, []string{
		"movie.mkv",
		"movie.mp4",
		"movie.avi",
		"movie.rmvb",
		"uppercase_movie.MKV", // uppercase extension
		"readme.txt",          // not video
		"image.jpg",           // not video
		"subtitle.srt",        // not video
		"document.pdf",        // not video
		"movie.mkv.part",      // not video (wrong extension)
	})

	svc, movieRepo, _ := setupScannerService(t, []string{dir})
	movieRepo.On("FindByFilePath", mock.Anything, mock.AnythingOfType("string")).Return(nil, nil)
	movieRepo.On("BulkCreate", mock.Anything, mock.AnythingOfType("[]*models.Movie")).Return(nil)

	ctx := context.Background()
	result, err := svc.StartScan(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	// .mkv, .mp4, .avi, .rmvb, .MKV = 5 video files
	assert.Equal(t, 5, result.FilesFound)
	assert.Equal(t, 5, result.FilesCreated)
}

func TestScannerService_StartScan_SymlinkFollowing(t *testing.T) {
	dir := t.TempDir()
	// Create a real file
	realDir := filepath.Join(dir, "real")
	os.MkdirAll(realDir, 0755)
	realFile := filepath.Join(realDir, "movie.mkv")
	os.WriteFile(realFile, []byte("fake video content"), 0644)

	// Create a symlink to the real file
	linkDir := filepath.Join(dir, "links")
	os.MkdirAll(linkDir, 0755)
	linkFile := filepath.Join(linkDir, "movie_link.mkv")
	err := os.Symlink(realFile, linkFile)
	if err != nil {
		t.Skip("symlinks not supported on this platform")
	}

	svc, movieRepo, _ := setupScannerService(t, []string{dir})
	movieRepo.On("FindByFilePath", mock.Anything, mock.AnythingOfType("string")).Return(nil, nil)
	movieRepo.On("BulkCreate", mock.Anything, mock.AnythingOfType("[]*models.Movie")).Return(nil)

	ctx := context.Background()
	result, err := svc.StartScan(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	// Both the real file and symlink resolve to the same path — only 1 should be created
	assert.Equal(t, 1, result.FilesFound, "symlink and real file should resolve to same path, counting only once as found")
	assert.Equal(t, 1, result.FilesCreated)
	// The second encounter should be skipped by dedup
	assert.Equal(t, 1, result.FilesSkipped)

	// Verify BulkCreate was called with the resolved path
	movieRepo.AssertCalled(t, "BulkCreate", mock.Anything, mock.MatchedBy(func(movies []*models.Movie) bool {
		if len(movies) != 1 {
			return false
		}
		resolvedReal, _ := filepath.EvalSymlinks(realFile)
		resolvedReal, _ = filepath.Abs(resolvedReal)
		return movies[0].FilePath.String == resolvedReal
	}))
}

func TestScannerService_IsScanActive(t *testing.T) {
	dir := t.TempDir()

	svc, movieRepo, _ := setupScannerService(t, []string{dir})
	movieRepo.On("FindByFilePath", mock.Anything, mock.AnythingOfType("string")).Return(nil, nil)
	movieRepo.On("BulkCreate", mock.Anything, mock.AnythingOfType("[]*models.Movie")).Return(nil)

	// Before scan
	assert.False(t, svc.IsScanActive())

	// After scan completes
	ctx := context.Background()
	_, _ = svc.StartScan(ctx)
	assert.False(t, svc.IsScanActive())
}

func TestScannerService_ConcurrentScanPrevention(t *testing.T) {
	dir := t.TempDir()

	svc, movieRepo, _ := setupScannerService(t, []string{dir})
	movieRepo.On("FindByFilePath", mock.Anything, mock.AnythingOfType("string")).Return(nil, nil)
	movieRepo.On("BulkCreate", mock.Anything, mock.AnythingOfType("[]*models.Movie")).Return(nil)

	// Manually set isScanning to simulate an active scan
	svc.mu.Lock()
	svc.isScanning = true
	svc.mu.Unlock()

	ctx := context.Background()
	result, err := svc.StartScan(ctx)

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "SCANNER_ALREADY_RUNNING")

	// Clean up
	svc.mu.Lock()
	svc.isScanning = false
	svc.mu.Unlock()
}

func TestScannerService_CancelScan(t *testing.T) {
	dir := t.TempDir()
	// Create many files so scan doesn't finish instantly
	for i := 0; i < 50; i++ {
		subdir := filepath.Join(dir, "sub"+strings.Repeat("d", 3))
		os.MkdirAll(subdir, 0755)
		os.WriteFile(filepath.Join(subdir, "movie"+strings.Repeat("x", 3)+".mkv"), []byte("fake"), 0644)
	}

	svc, movieRepo, _ := setupScannerService(t, []string{dir})
	movieRepo.On("FindByFilePath", mock.Anything, mock.AnythingOfType("string")).Return(nil, nil)
	movieRepo.On("BulkCreate", mock.Anything, mock.AnythingOfType("[]*models.Movie")).Return(nil)

	// Cancel before scan should return error
	err := svc.CancelScan()
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrScanNotActive)

	// Simulate a running scan and cancel it
	svc.mu.Lock()
	svc.isScanning = true
	svc.cancelChan = make(chan struct{})
	svc.mu.Unlock()

	err = svc.CancelScan()
	assert.NoError(t, err)

	// Verify the cancel chan is closed
	select {
	case <-svc.cancelChan:
		// expected — channel is closed
	default:
		t.Error("cancelChan should be closed after CancelScan")
	}

	// Clean up
	svc.mu.Lock()
	svc.isScanning = false
	svc.mu.Unlock()
}

func TestScannerService_SSEBroadcast(t *testing.T) {
	dir := t.TempDir()
	// Create 15 files to trigger at least one broadcast (every 10 files)
	for i := 0; i < 15; i++ {
		name := filepath.Join(dir, "movie"+strings.Repeat("x", 3)+"_"+string(rune('a'+i))+".mkv")
		os.WriteFile(name, []byte("fake video"), 0644)
	}

	svc, movieRepo, hub := setupScannerService(t, []string{dir})
	movieRepo.On("FindByFilePath", mock.Anything, mock.AnythingOfType("string")).Return(nil, nil)
	movieRepo.On("BulkCreate", mock.Anything, mock.AnythingOfType("[]*models.Movie")).Return(nil)

	// Register a client to receive events
	client := hub.Register()

	// Give the hub goroutine a moment to register the client
	time.Sleep(10 * time.Millisecond)

	ctx := context.Background()
	result, err := svc.StartScan(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Wait for events to propagate through hub goroutine
	time.Sleep(20 * time.Millisecond)

	// Collect events (non-blocking drain)
	var events []sse.Event
	for {
		select {
		case evt := <-client.Events:
			events = append(events, evt)
		default:
			goto done
		}
	}
done:

	// Should have received at least one scan_progress event + one scan_complete event
	assert.GreaterOrEqual(t, len(events), 2, "should have received at least two SSE events (progress + complete)")

	// All events except the last should be scan_progress
	for _, evt := range events[:len(events)-1] {
		assert.Equal(t, sse.EventScanProgress, evt.Type)
	}

	// Last event should be scan_complete
	lastEvent := events[len(events)-1]
	assert.Equal(t, sse.EventScanComplete, lastEvent.Type)

	// Verify scan_complete payload has expected fields
	data, ok := lastEvent.Data.(map[string]interface{})
	assert.True(t, ok, "scan_complete data should be map[string]interface{}")
	assert.Contains(t, data, "files_found")
	assert.Contains(t, data, "error_count")
}

func TestScannerService_SSEBroadcast_ScanCancelled(t *testing.T) {
	dir := t.TempDir()
	// Create enough files so scan takes time
	for i := 0; i < 20; i++ {
		name := filepath.Join(dir, fmt.Sprintf("movie_%02d.mkv", i))
		os.WriteFile(name, []byte("fake video"), 0644)
	}

	svc, movieRepo, hub := setupScannerService(t, []string{dir})
	movieRepo.On("FindByFilePath", mock.Anything, mock.AnythingOfType("string")).Return(nil, nil)
	movieRepo.On("BulkCreate", mock.Anything, mock.AnythingOfType("[]*models.Movie")).Return(nil)
	movieRepo.On("FindAllWithFilePath", mock.Anything).Return(nil, nil)

	client := hub.Register()
	time.Sleep(10 * time.Millisecond)

	// Start scan and cancel quickly — use 1ms to catch the early cancel check
	// added in Story 7b-5 (before directory walk)
	ctx := context.Background()
	go func() {
		time.Sleep(1 * time.Millisecond)
		svc.CancelScan()
	}()

	result, err := svc.StartScan(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Wait a bit for events to propagate
	time.Sleep(20 * time.Millisecond)

	// Drain events
	var events []sse.Event
	for {
		select {
		case evt := <-client.Events:
			events = append(events, evt)
		default:
			goto cancelled_done
		}
	}
cancelled_done:

	// Should have at least one event, and the last should be scan_cancelled
	assert.GreaterOrEqual(t, len(events), 1, "should have received at least one SSE event")

	lastEvent := events[len(events)-1]
	assert.Equal(t, sse.EventScanCancelled, lastEvent.Type)
}

func TestScannerService_GetProgress(t *testing.T) {
	svc, _, _ := setupScannerService(t, []string{})

	// Default progress should not be active
	progress := svc.GetProgress()
	assert.False(t, progress.IsActive)
	assert.Equal(t, 0, progress.FilesFound)
}

func TestScannerService_StartScan_MultipleDirs(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()
	createVideoFiles(t, dir1, []string{"movie1.mkv"})
	createVideoFiles(t, dir2, []string{"movie2.mp4"})

	svc, movieRepo, _ := setupScannerService(t, []string{dir1, dir2})
	movieRepo.On("FindByFilePath", mock.Anything, mock.AnythingOfType("string")).Return(nil, nil)
	movieRepo.On("BulkCreate", mock.Anything, mock.AnythingOfType("[]*models.Movie")).Return(nil)

	ctx := context.Background()
	result, err := svc.StartScan(ctx)

	assert.NoError(t, err)
	assert.Equal(t, 2, result.FilesFound)
	assert.Equal(t, 2, result.FilesCreated)
}

func TestScannerService_StartScan_MixedValidInvalidPaths(t *testing.T) {
	dir := t.TempDir()
	createVideoFiles(t, dir, []string{"movie.mkv"})
	nonExistent := filepath.Join(t.TempDir(), "nope")

	svc, movieRepo, _ := setupScannerService(t, []string{nonExistent, dir})
	movieRepo.On("FindByFilePath", mock.Anything, mock.AnythingOfType("string")).Return(nil, nil)
	movieRepo.On("BulkCreate", mock.Anything, mock.AnythingOfType("[]*models.Movie")).Return(nil)

	ctx := context.Background()
	result, err := svc.StartScan(ctx)

	assert.NoError(t, err)
	assert.Equal(t, 1, result.FilesFound)
	assert.Equal(t, 1, result.FilesCreated)
	assert.Equal(t, 1, result.ErrorCount) // invalid path counted as error
}

func TestScannerService_NewMovieFields(t *testing.T) {
	dir := t.TempDir()
	createVideoFiles(t, dir, []string{"test_movie.mkv"})

	svc, movieRepo, _ := setupScannerService(t, []string{dir})
	movieRepo.On("FindByFilePath", mock.Anything, mock.AnythingOfType("string")).Return(nil, nil)

	var capturedMovies []*models.Movie
	movieRepo.On("BulkCreate", mock.Anything, mock.AnythingOfType("[]*models.Movie")).
		Run(func(args mock.Arguments) {
			capturedMovies = args.Get(1).([]*models.Movie)
		}).
		Return(nil)

	ctx := context.Background()
	result, err := svc.StartScan(ctx)

	assert.NoError(t, err)
	assert.Equal(t, 1, result.FilesCreated)
	assert.Len(t, capturedMovies, 1)

	movie := capturedMovies[0]
	assert.NotEmpty(t, movie.ID, "ID should be a UUID")
	assert.Equal(t, "test_movie.mkv", movie.Title, "Title should be the filename")
	assert.True(t, movie.FilePath.Valid)
	assert.True(t, movie.FileSize.Valid)
	assert.Equal(t, models.ParseStatusPending, movie.ParseStatus)
	assert.Equal(t, models.SubtitleStatusNotSearched, movie.SubtitleStatus)
	assert.False(t, movie.CreatedAt.IsZero())
	assert.False(t, movie.UpdatedAt.IsZero())
}

func TestIsVideoFile(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"movie.mkv", true},
		{"movie.mp4", true},
		{"movie.avi", true},
		{"movie.rmvb", true},
		{"MOVIE.MKV", true},
		{"Movie.Mp4", true},
		{"readme.txt", false},
		{"image.jpg", false},
		{"subtitle.srt", false},
		{"movie.mkv.part", false},
		{".mkv", true}, // just extension is technically valid
		{"no_extension", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			assert.Equal(t, tt.expected, isVideoFile(tt.path))
		})
	}
}

func TestScannerService_NilSSEHub(t *testing.T) {
	dir := t.TempDir()
	createVideoFiles(t, dir, []string{"movie.mkv"})

	movieRepo := new(testutil.MockMovieRepository)
	seriesRepo := new(testutil.MockSeriesRepository)
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))

	// Create service with nil SSE hub — should not panic
	svc := NewScannerService(movieRepo, seriesRepo, []string{dir}, nil, logger)

	movieRepo.On("FindByFilePath", mock.Anything, mock.AnythingOfType("string")).Return(nil, nil)
	movieRepo.On("BulkCreate", mock.Anything, mock.AnythingOfType("[]*models.Movie")).Return(nil)
	movieRepo.On("FindAllWithFilePath", mock.Anything).Return([]models.Movie{}, nil)

	ctx := context.Background()
	result, err := svc.StartScan(ctx)

	assert.NoError(t, err)
	assert.Equal(t, 1, result.FilesCreated)
}

func TestScannerService_IncrementalScan_DetectRemovedFiles(t *testing.T) {
	dir := t.TempDir()

	svc, movieRepo, _ := setupScannerService(t, []string{dir})

	// Return movies with file paths that don't exist
	nonExistentPath := "/tmp/nonexistent_movie_" + t.Name() + ".mkv"
	existingMovies := []models.Movie{
		{
			ID:       "movie-1",
			Title:    "removed.mkv",
			FilePath: models.NewNullString(nonExistentPath),
		},
	}

	// Override the default Maybe() mock with a specific one
	movieRepo.ExpectedCalls = filterCalls(movieRepo.ExpectedCalls, "FindAllWithFilePath")
	movieRepo.On("FindAllWithFilePath", mock.Anything).Return(existingMovies, nil)
	movieRepo.On("Update", mock.Anything, mock.MatchedBy(func(m *models.Movie) bool {
		return m.ID == "movie-1" && m.IsRemoved == true
	})).Return(nil)

	ctx := context.Background()
	result, err := svc.StartScan(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 1, result.FilesRemoved)

	movieRepo.AssertCalled(t, "Update", mock.Anything, mock.MatchedBy(func(m *models.Movie) bool {
		return m.ID == "movie-1" && m.IsRemoved == true
	}))
}

func TestScannerService_IncrementalScan_MtimeChange(t *testing.T) {
	dir := t.TempDir()
	paths := createVideoFiles(t, dir, []string{"mtime_test.mkv"})

	svc, movieRepo, _ := setupScannerService(t, []string{dir})

	resolvedPath, _ := filepath.EvalSymlinks(paths[0])
	resolvedPath, _ = filepath.Abs(resolvedPath)

	// Existing movie with same size but old UpdatedAt (mtime is newer)
	existingMovie := &models.Movie{
		ID:        "existing-id",
		Title:     "mtime_test.mkv",
		FilePath:  models.NewNullString(resolvedPath),
		FileSize:  models.NewNullInt64(int64(len("fake video content"))),
		UpdatedAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), // old time
	}
	movieRepo.On("FindByFilePath", mock.Anything, resolvedPath).Return(existingMovie, nil)
	movieRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.Movie")).Return(nil)

	ctx := context.Background()
	result, err := svc.StartScan(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 1, result.FilesFound)
	assert.Equal(t, 1, result.FilesUpdated) // updated because mtime is newer
	assert.Equal(t, 0, result.FilesSkipped)

	movieRepo.AssertCalled(t, "Update", mock.Anything, mock.AnythingOfType("*models.Movie"))
}

// filterCalls removes mock calls matching the given method name
func filterCalls(calls []*mock.Call, method string) []*mock.Call {
	var filtered []*mock.Call
	for _, c := range calls {
		if c.Method != method {
			filtered = append(filtered, c)
		}
	}
	return filtered
}

// --- bugfix-b: TV routing -----------------------------------------------------------
//
// Before this, the scanner wrote EVERY scanned file into `movies` — the media-type
// decision was computed (scanDir.contentType) and never threaded past StartScan — while
// the enrichment pass correctly recognised each file as TV and then stamped the whole
// series' TMDb metadata onto the movie row. A real TV library came out as one `movies`
// row per episode, all wearing the same poster, with series/seasons/episodes empty.

// setupTVScanner builds a scanner with TV routing enabled over the given dirs.
func setupTVScanner(t *testing.T, mediaDirs []string) (*ScannerService, *testutil.MockMovieRepository, *mockPQSeriesRepo, *mockPQSeasonRepo, *mockPQEpisodeRepo) {
	t.Helper()

	movieRepo := new(testutil.MockMovieRepository)
	seriesRepo := newMockPQSeriesRepo()
	seasonRepo := newMockPQSeasonRepo()
	episodeRepo := newMockPQEpisodeRepo()

	hub := sse.NewHub()
	t.Cleanup(func() { hub.Close() })

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	svc := NewScannerService(movieRepo, seriesRepo, mediaDirs, hub, logger)

	ingest := NewMediaIngestService(seriesRepo, seasonRepo, episodeRepo, logger)
	svc.SetTVIngest(ingest, NewParserService())

	movieRepo.On("FindAllWithFilePath", mock.Anything).Maybe().Return([]models.Movie{}, nil)

	return svc, movieRepo, seriesRepo, seasonRepo, episodeRepo
}

func TestScannerService_TVEpisodesRouteToSeriesNotMovies(t *testing.T) {
	dir := t.TempDir()
	seasonDir := filepath.Join(dir, "鵲刀門傳奇", "Season02")
	if err := os.MkdirAll(seasonDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	createVideoFiles(t, seasonDir, []string{
		"Legend.of.the.Undercover.Chef.2025.S02E05.2160p.WEB-DL.mkv",
		"Legend.of.the.Undercover.Chef.2025.S02E06.2160p.WEB-DL.mkv",
	})

	svc, movieRepo, seriesRepo, seasonRepo, episodeRepo := setupTVScanner(t, []string{dir})

	// The scanner checks for a mis-filed movie row per TV file; there is none here.
	movieRepo.On("FindByFilePath", mock.Anything, mock.AnythingOfType("string")).Return(nil, nil)

	result, err := svc.StartScan(context.Background())
	if err != nil {
		t.Fatalf("StartScan: %v", err)
	}
	if result.FilesFound != 2 {
		t.Fatalf("FilesFound = %d, want 2", result.FilesFound)
	}

	// The whole point: nothing lands in `movies`.
	movieRepo.AssertNotCalled(t, "BulkCreate", mock.Anything, mock.Anything)
	movieRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)

	if len(seriesRepo.series) != 1 {
		t.Fatalf("series count = %d, want 1 (both episodes belong to one show)", len(seriesRepo.series))
	}
	if len(seasonRepo.seasons) != 1 {
		t.Errorf("season count = %d, want 1", len(seasonRepo.seasons))
	}
	if len(episodeRepo.episodes) != 2 {
		t.Errorf("episode count = %d, want 2", len(episodeRepo.episodes))
	}

	// The scanner resolves symlinks, and on macOS t.TempDir() lives under /var → /private/var.
	resolvedDir, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatalf("EvalSymlinks: %v", err)
	}

	for _, s := range seriesRepo.series {
		// Identity is the series folder — climbing past Season02 — so a second season
		// would attach to this same series rather than creating another one.
		wantDir := filepath.Join(resolvedDir, "鵲刀門傳奇")
		if !s.FilePath.Valid || s.FilePath.String != wantDir {
			t.Errorf("series file_path = %q, want %q", s.FilePath.String, wantDir)
		}
		if s.ParseStatus != models.ParseStatusPending {
			t.Errorf("series parse_status = %q, want pending (enrichment fills it in)", s.ParseStatus)
		}
	}

	seasons := map[int]bool{}
	for _, ep := range episodeRepo.episodes {
		seasons[ep.SeasonNumber] = true
		if ep.SeasonNumber != 2 {
			t.Errorf("episode season = %d, want 2", ep.SeasonNumber)
		}
		if ep.EpisodeNumber != 5 && ep.EpisodeNumber != 6 {
			t.Errorf("episode number = %d, want 5 or 6", ep.EpisodeNumber)
		}
		if !ep.FilePath.Valid || !strings.HasSuffix(ep.FilePath.String, ".mkv") {
			t.Errorf("episode file_path = %q, want the .mkv path", ep.FilePath.String)
		}
	}
	if len(seasons) != 1 {
		t.Errorf("distinct seasons = %d, want 1", len(seasons))
	}
}

// TestScannerService_DeletesMisFiledMovieRowForTVEpisode covers the repair path: the
// 5000-odd episode-as-movie rows a real deployment already accumulated must disappear on
// the next scan, or the same episode shows up twice — once in the movie grid, once under
// its series. A soft-delete would not do: FindByFilePath does not filter is_removed, so
// the next scan would resurrect it.
func TestScannerService_DeletesMisFiledMovieRowForTVEpisode(t *testing.T) {
	dir := t.TempDir()
	seasonDir := filepath.Join(dir, "Show", "Season01")
	if err := os.MkdirAll(seasonDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	paths := createVideoFiles(t, seasonDir, []string{"Show.S01E01.1080p.mkv"})

	svc, movieRepo, _, _, episodeRepo := setupTVScanner(t, []string{dir})

	stale := &models.Movie{ID: "stale-movie-row", Title: "Show", FilePath: models.NewNullString(paths[0])}
	movieRepo.On("FindByFilePath", mock.Anything, mock.AnythingOfType("string")).Return(stale, nil)
	movieRepo.On("Delete", mock.Anything, "stale-movie-row").Return(nil)

	if _, err := svc.StartScan(context.Background()); err != nil {
		t.Fatalf("StartScan: %v", err)
	}

	movieRepo.AssertCalled(t, "Delete", mock.Anything, "stale-movie-row")
	if len(episodeRepo.episodes) != 1 {
		t.Errorf("episode count = %d, want 1", len(episodeRepo.episodes))
	}
}

// TestScannerService_MoviesStillRouteToMovies pins the other half: a film is still a film.
func TestScannerService_MoviesStillRouteToMovies(t *testing.T) {
	dir := t.TempDir()
	createVideoFiles(t, dir, []string{"The.Matrix.1999.1080p.BluRay.x264.mkv"})

	svc, movieRepo, seriesRepo, _, episodeRepo := setupTVScanner(t, []string{dir})

	movieRepo.On("FindByFilePath", mock.Anything, mock.AnythingOfType("string")).Return(nil, nil)
	movieRepo.On("BulkCreate", mock.Anything, mock.AnythingOfType("[]*models.Movie")).Return(nil)

	if _, err := svc.StartScan(context.Background()); err != nil {
		t.Fatalf("StartScan: %v", err)
	}

	movieRepo.AssertCalled(t, "BulkCreate", mock.Anything, mock.Anything)
	if len(seriesRepo.series) != 0 {
		t.Errorf("series count = %d, want 0 — a movie must not create a series", len(seriesRepo.series))
	}
	if len(episodeRepo.episodes) != 0 {
		t.Errorf("episode count = %d, want 0", len(episodeRepo.episodes))
	}
}
