package services

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
	"github.com/vido/api/internal/sse"
)

// MockMovieRepoScanner implements MovieRepositoryInterface for scanner tests
type MockMovieRepoScanner struct {
	mock.Mock
}

func (m *MockMovieRepoScanner) Create(ctx context.Context, movie *models.Movie) error {
	return m.Called(ctx, movie).Error(0)
}
func (m *MockMovieRepoScanner) FindByID(ctx context.Context, id string) (*models.Movie, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Movie), args.Error(1)
}
func (m *MockMovieRepoScanner) FindByTMDbID(ctx context.Context, tmdbID int64) (*models.Movie, error) {
	return nil, nil
}
func (m *MockMovieRepoScanner) FindByIMDbID(ctx context.Context, imdbID string) (*models.Movie, error) {
	return nil, nil
}
func (m *MockMovieRepoScanner) Update(ctx context.Context, movie *models.Movie) error {
	return m.Called(ctx, movie).Error(0)
}
func (m *MockMovieRepoScanner) Delete(ctx context.Context, id string) error { return nil }
func (m *MockMovieRepoScanner) List(ctx context.Context, params repository.ListParams) ([]models.Movie, *repository.PaginationResult, error) {
	return nil, nil, nil
}
func (m *MockMovieRepoScanner) SearchByTitle(ctx context.Context, title string, params repository.ListParams) ([]models.Movie, *repository.PaginationResult, error) {
	return nil, nil, nil
}
func (m *MockMovieRepoScanner) FullTextSearch(ctx context.Context, query string, params repository.ListParams) ([]models.Movie, *repository.PaginationResult, error) {
	return nil, nil, nil
}
func (m *MockMovieRepoScanner) Upsert(ctx context.Context, movie *models.Movie) error { return nil }
func (m *MockMovieRepoScanner) FindByFilePath(ctx context.Context, filePath string) (*models.Movie, error) {
	args := m.Called(ctx, filePath)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Movie), args.Error(1)
}
func (m *MockMovieRepoScanner) GetDistinctGenres(ctx context.Context) ([]string, error) {
	return nil, nil
}
func (m *MockMovieRepoScanner) GetYearRange(ctx context.Context) (int, int, error) {
	return 0, 0, nil
}
func (m *MockMovieRepoScanner) Count(ctx context.Context) (int, error) { return 0, nil }
func (m *MockMovieRepoScanner) BulkCreate(ctx context.Context, movies []*models.Movie) error {
	return m.Called(ctx, movies).Error(0)
}
func (m *MockMovieRepoScanner) FindByParseStatus(ctx context.Context, status models.ParseStatus) ([]models.Movie, error) {
	return nil, nil
}
func (m *MockMovieRepoScanner) UpdateSubtitleStatus(ctx context.Context, id string, status models.SubtitleStatus, path, language string, score float64) error {
	return nil
}
func (m *MockMovieRepoScanner) FindBySubtitleStatus(ctx context.Context, status models.SubtitleStatus) ([]models.Movie, error) {
	return nil, nil
}
func (m *MockMovieRepoScanner) FindNeedingSubtitleSearch(ctx context.Context, olderThan time.Time) ([]models.Movie, error) {
	return nil, nil
}

// MockSeriesRepoScanner implements SeriesRepositoryInterface for scanner tests
type MockSeriesRepoScanner struct {
	mock.Mock
}

func (m *MockSeriesRepoScanner) Create(ctx context.Context, series *models.Series) error {
	return nil
}
func (m *MockSeriesRepoScanner) FindByID(ctx context.Context, id string) (*models.Series, error) {
	return nil, nil
}
func (m *MockSeriesRepoScanner) FindByTMDbID(ctx context.Context, tmdbID int64) (*models.Series, error) {
	return nil, nil
}
func (m *MockSeriesRepoScanner) FindByIMDbID(ctx context.Context, imdbID string) (*models.Series, error) {
	return nil, nil
}
func (m *MockSeriesRepoScanner) Update(ctx context.Context, series *models.Series) error {
	return nil
}
func (m *MockSeriesRepoScanner) Delete(ctx context.Context, id string) error { return nil }
func (m *MockSeriesRepoScanner) List(ctx context.Context, params repository.ListParams) ([]models.Series, *repository.PaginationResult, error) {
	return nil, nil, nil
}
func (m *MockSeriesRepoScanner) SearchByTitle(ctx context.Context, title string, params repository.ListParams) ([]models.Series, *repository.PaginationResult, error) {
	return nil, nil, nil
}
func (m *MockSeriesRepoScanner) FullTextSearch(ctx context.Context, query string, params repository.ListParams) ([]models.Series, *repository.PaginationResult, error) {
	return nil, nil, nil
}
func (m *MockSeriesRepoScanner) Upsert(ctx context.Context, series *models.Series) error {
	return nil
}
func (m *MockSeriesRepoScanner) GetDistinctGenres(ctx context.Context) ([]string, error) {
	return nil, nil
}
func (m *MockSeriesRepoScanner) GetYearRange(ctx context.Context) (int, int, error) {
	return 0, 0, nil
}
func (m *MockSeriesRepoScanner) Count(ctx context.Context) (int, error) { return 0, nil }
func (m *MockSeriesRepoScanner) BulkCreate(ctx context.Context, seriesList []*models.Series) error {
	return nil
}
func (m *MockSeriesRepoScanner) FindByParseStatus(ctx context.Context, status models.ParseStatus) ([]models.Series, error) {
	return nil, nil
}
func (m *MockSeriesRepoScanner) UpdateSubtitleStatus(ctx context.Context, id string, status models.SubtitleStatus, path, language string, score float64) error {
	return nil
}
func (m *MockSeriesRepoScanner) FindBySubtitleStatus(ctx context.Context, status models.SubtitleStatus) ([]models.Series, error) {
	return nil, nil
}
func (m *MockSeriesRepoScanner) FindNeedingSubtitleSearch(ctx context.Context, olderThan time.Time) ([]models.Series, error) {
	return nil, nil
}

// helper to create a scanner service with mocks
func setupScannerService(t *testing.T, mediaDirs []string) (*ScannerService, *MockMovieRepoScanner, *sse.Hub) {
	t.Helper()
	movieRepo := new(MockMovieRepoScanner)
	seriesRepo := new(MockSeriesRepoScanner)
	hub := sse.NewHub()
	t.Cleanup(func() { hub.Close() })

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	svc := NewScannerService(movieRepo, seriesRepo, mediaDirs, hub, logger)
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
		ID:       "existing-id",
		Title:    "existing.mkv",
		FilePath: sql.NullString{String: resolvedPath, Valid: true},
		FileSize: sql.NullInt64{Int64: int64(len("fake video content")), Valid: true},
	}
	movieRepo.On("FindByFilePath", mock.Anything, resolvedPath).Return(existingMovie, nil)

	ctx := context.Background()
	result, err := svc.StartScan(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 1, result.FilesFound)
	assert.Equal(t, 0, result.FilesCreated)
	assert.Equal(t, 1, result.FilesSkipped) // skipped because same size
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
		FilePath: sql.NullString{String: resolvedPath, Valid: true},
		FileSize: sql.NullInt64{Int64: 999, Valid: true}, // different size
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
	// Create many files so the scan takes a moment
	var names []string
	for i := 0; i < 5; i++ {
		names = append(names, filepath.Join("sub", strings.Repeat("a", 5)+".mkv"))
	}

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
	// Create files that will take time to scan
	var names []string
	for i := 0; i < 200; i++ {
		names = append(names, filepath.Join("dir", "movie"+strings.Repeat("x", 3)+".mkv"))
	}

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

	// Should have received at least one scan_progress event (at 10 files) + final broadcast
	assert.GreaterOrEqual(t, len(events), 1, "should have received at least one SSE event")
	for _, evt := range events {
		assert.Equal(t, sse.EventScanProgress, evt.Type)
	}
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

	movieRepo := new(MockMovieRepoScanner)
	seriesRepo := new(MockSeriesRepoScanner)
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))

	// Create service with nil SSE hub — should not panic
	svc := NewScannerService(movieRepo, seriesRepo, []string{dir}, nil, logger)

	movieRepo.On("FindByFilePath", mock.Anything, mock.AnythingOfType("string")).Return(nil, nil)
	movieRepo.On("BulkCreate", mock.Anything, mock.AnythingOfType("[]*models.Movie")).Return(nil)

	ctx := context.Background()
	result, err := svc.StartScan(ctx)

	assert.NoError(t, err)
	assert.Equal(t, 1, result.FilesCreated)
}
