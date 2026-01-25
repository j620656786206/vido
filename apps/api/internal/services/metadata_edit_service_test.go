package services

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/jpeg"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/images"
	"github.com/vido/api/internal/models"
)

// mockMovieMetadataRepository implements MovieMetadataRepository for testing
type mockMovieMetadataRepository struct {
	movies     map[string]*models.Movie
	findByIDFn func(ctx context.Context, id string) (*models.Movie, error)
	updateFn   func(ctx context.Context, movie *models.Movie) error
}

func newMockMovieRepo() *mockMovieMetadataRepository {
	return &mockMovieMetadataRepository{
		movies: make(map[string]*models.Movie),
	}
}

func (m *mockMovieMetadataRepository) FindByID(ctx context.Context, id string) (*models.Movie, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(ctx, id)
	}
	movie, ok := m.movies[id]
	if !ok {
		return nil, ErrUpdateMetadataNotFound
	}
	return movie, nil
}

func (m *mockMovieMetadataRepository) Update(ctx context.Context, movie *models.Movie) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, movie)
	}
	m.movies[movie.ID] = movie
	return nil
}

// mockSeriesMetadataRepository implements SeriesMetadataRepository for testing
type mockSeriesMetadataRepository struct {
	series     map[string]*models.Series
	findByIDFn func(ctx context.Context, id string) (*models.Series, error)
	updateFn   func(ctx context.Context, series *models.Series) error
}

func newMockSeriesRepo() *mockSeriesMetadataRepository {
	return &mockSeriesMetadataRepository{
		series: make(map[string]*models.Series),
	}
}

func (m *mockSeriesMetadataRepository) FindByID(ctx context.Context, id string) (*models.Series, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(ctx, id)
	}
	series, ok := m.series[id]
	if !ok {
		return nil, ErrUpdateMetadataNotFound
	}
	return series, nil
}

func (m *mockSeriesMetadataRepository) Update(ctx context.Context, series *models.Series) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, series)
	}
	m.series[series.ID] = series
	return nil
}

func TestNewMetadataEditService(t *testing.T) {
	movieRepo := newMockMovieRepo()
	seriesRepo := newMockSeriesRepo()

	service := NewMetadataEditService(movieRepo, seriesRepo, nil)

	assert.NotNil(t, service)
}

func TestMetadataEditService_Exists_MovieExists(t *testing.T) {
	movieRepo := newMockMovieRepo()
	movieRepo.movies["movie-1"] = &models.Movie{ID: "movie-1", Title: "Test Movie"}
	seriesRepo := newMockSeriesRepo()

	service := NewMetadataEditService(movieRepo, seriesRepo, nil)

	exists, err := service.Exists(context.Background(), "movie-1")

	require.NoError(t, err)
	assert.True(t, exists)
}

func TestMetadataEditService_Exists_SeriesExists(t *testing.T) {
	movieRepo := newMockMovieRepo()
	seriesRepo := newMockSeriesRepo()
	seriesRepo.series["series-1"] = &models.Series{ID: "series-1", Title: "Test Series"}

	service := NewMetadataEditService(movieRepo, seriesRepo, nil)

	exists, err := service.Exists(context.Background(), "series-1")

	require.NoError(t, err)
	assert.True(t, exists)
}

func TestMetadataEditService_Exists_NotFound(t *testing.T) {
	movieRepo := newMockMovieRepo()
	seriesRepo := newMockSeriesRepo()

	service := NewMetadataEditService(movieRepo, seriesRepo, nil)

	exists, err := service.Exists(context.Background(), "nonexistent")

	require.NoError(t, err)
	assert.False(t, exists)
}

func TestMetadataEditService_UpdateMetadata_MovieSuccess(t *testing.T) {
	movieRepo := newMockMovieRepo()
	movieRepo.movies["movie-1"] = &models.Movie{
		ID:          "movie-1",
		Title:       "Old Title",
		ReleaseDate: "2000-01-01",
		Genres:      []string{"Drama"},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	seriesRepo := newMockSeriesRepo()

	service := NewMetadataEditService(movieRepo, seriesRepo, nil)

	req := &UpdateMetadataRequest{
		ID:           "movie-1",
		MediaType:    "movie",
		Title:        "鬼滅之刃",
		TitleEnglish: "Demon Slayer",
		Year:         2019,
		Genres:       []string{"動作", "奇幻"},
		Overview:     "大正時代的日本...",
	}

	result, err := service.UpdateMetadata(context.Background(), req)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "movie-1", result.ID)
	assert.Equal(t, "鬼滅之刃", result.Title)
	assert.Equal(t, models.MetadataSourceManual, result.MetadataSource)

	// Verify movie was updated
	updatedMovie := movieRepo.movies["movie-1"]
	assert.Equal(t, "鬼滅之刃", updatedMovie.Title)
	assert.Equal(t, "Demon Slayer", updatedMovie.OriginalTitle.String)
	assert.Equal(t, "2019-01-01", updatedMovie.ReleaseDate)
	assert.Contains(t, updatedMovie.Genres, "動作")
	assert.Equal(t, "大正時代的日本...", updatedMovie.Overview.String)
	assert.Equal(t, "manual", updatedMovie.MetadataSource.String)
}

func TestMetadataEditService_UpdateMetadata_SeriesSuccess(t *testing.T) {
	movieRepo := newMockMovieRepo()
	seriesRepo := newMockSeriesRepo()
	seriesRepo.series["series-1"] = &models.Series{
		ID:           "series-1",
		Title:        "Old Title",
		FirstAirDate: "2000-01-01",
		Genres:       []string{"Drama"},
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	service := NewMetadataEditService(movieRepo, seriesRepo, nil)

	req := &UpdateMetadataRequest{
		ID:        "series-1",
		MediaType: "series",
		Title:     "Breaking Bad",
		Year:      2008,
		Genres:    []string{"Drama", "Crime"},
	}

	result, err := service.UpdateMetadata(context.Background(), req)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "series-1", result.ID)
	assert.Equal(t, "Breaking Bad", result.Title)

	// Verify series was updated
	updatedSeries := seriesRepo.series["series-1"]
	assert.Equal(t, "Breaking Bad", updatedSeries.Title)
	assert.Equal(t, "2008-01-01", updatedSeries.FirstAirDate)
}

func TestMetadataEditService_UpdateMetadata_MovieNotFound(t *testing.T) {
	movieRepo := newMockMovieRepo()
	seriesRepo := newMockSeriesRepo()

	service := NewMetadataEditService(movieRepo, seriesRepo, nil)

	req := &UpdateMetadataRequest{
		ID:        "nonexistent",
		MediaType: "movie",
		Title:     "Test",
		Year:      2020,
	}

	result, err := service.UpdateMetadata(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, ErrUpdateMetadataNotFound, err)
}

func TestMetadataEditService_UpdateMetadata_DefaultsToMovie(t *testing.T) {
	movieRepo := newMockMovieRepo()
	movieRepo.movies["media-1"] = &models.Movie{
		ID:          "media-1",
		Title:       "Test",
		ReleaseDate: "2020-01-01",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	seriesRepo := newMockSeriesRepo()

	service := NewMetadataEditService(movieRepo, seriesRepo, nil)

	req := &UpdateMetadataRequest{
		ID:        "media-1",
		MediaType: "", // Empty - should default to movie
		Title:     "Updated Title",
		Year:      2021,
	}

	result, err := service.UpdateMetadata(context.Background(), req)

	require.NoError(t, err)
	assert.Equal(t, "Updated Title", result.Title)
}

func TestMetadataEditService_UploadPoster_Success(t *testing.T) {
	movieRepo := newMockMovieRepo()
	movieRepo.movies["movie-1"] = &models.Movie{
		ID:        "movie-1",
		Title:     "Test Movie",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	seriesRepo := newMockSeriesRepo()

	tmpDir := t.TempDir()
	processor, err := images.NewImageProcessor(tmpDir)
	require.NoError(t, err)

	service := NewMetadataEditService(movieRepo, seriesRepo, processor)

	// Create test image
	jpegData := createTestJPEGData(t, 400, 600)

	req := &UploadPosterRequest{
		MediaID:     "movie-1",
		MediaType:   "movie",
		FileData:    jpegData,
		FileName:    "poster.jpg",
		ContentType: "image/jpeg",
		FileSize:    int64(len(jpegData)),
	}

	result, err := service.UploadPoster(context.Background(), req)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, result.PosterURL, "movie-1")
	assert.Contains(t, result.ThumbnailURL, "movie-1-thumb")
}

func TestMetadataEditService_UploadPoster_NoProcessor(t *testing.T) {
	movieRepo := newMockMovieRepo()
	seriesRepo := newMockSeriesRepo()

	service := NewMetadataEditService(movieRepo, seriesRepo, nil)

	req := &UploadPosterRequest{
		MediaID:     "movie-1",
		MediaType:   "movie",
		FileData:    []byte("test"),
		FileName:    "poster.jpg",
		ContentType: "image/jpeg",
		FileSize:    4,
	}

	result, err := service.UploadPoster(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "not configured")
}

func TestMetadataEditService_UploadPoster_InvalidImage(t *testing.T) {
	movieRepo := newMockMovieRepo()
	movieRepo.movies["movie-1"] = &models.Movie{ID: "movie-1", Title: "Test"}
	seriesRepo := newMockSeriesRepo()

	tmpDir := t.TempDir()
	processor, err := images.NewImageProcessor(tmpDir)
	require.NoError(t, err)

	service := NewMetadataEditService(movieRepo, seriesRepo, processor)

	req := &UploadPosterRequest{
		MediaID:     "movie-1",
		MediaType:   "movie",
		FileData:    []byte("not an image"),
		FileName:    "poster.jpg",
		ContentType: "image/jpeg",
		FileSize:    12,
	}

	result, err := service.UploadPoster(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, result)
}

// createTestJPEGData creates test JPEG image data
func createTestJPEGData(t *testing.T, width, height int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: 255, G: 128, B: 64, A: 255})
		}
	}

	var buf bytes.Buffer
	err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90})
	require.NoError(t, err)
	return buf.Bytes()
}

func TestMetadataEditService_UpdateMetadata_WithPosterURL(t *testing.T) {
	movieRepo := newMockMovieRepo()
	movieRepo.movies["movie-1"] = &models.Movie{
		ID:          "movie-1",
		Title:       "Test",
		ReleaseDate: "2020-01-01",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	seriesRepo := newMockSeriesRepo()

	service := NewMetadataEditService(movieRepo, seriesRepo, nil)

	req := &UpdateMetadataRequest{
		ID:        "movie-1",
		MediaType: "movie",
		Title:     "Test Updated",
		Year:      2021,
		PosterURL: "https://example.com/poster.jpg",
	}

	result, err := service.UpdateMetadata(context.Background(), req)

	require.NoError(t, err)
	assert.NotNil(t, result)

	// Verify poster was updated
	updatedMovie := movieRepo.movies["movie-1"]
	assert.Equal(t, "https://example.com/poster.jpg", updatedMovie.PosterPath.String)
}

func TestBytesReader_Read(t *testing.T) {
	data := []byte("hello world")
	reader := NewBytesReader(data)

	buf := make([]byte, 5)
	n, err := reader.Read(buf)

	require.NoError(t, err)
	assert.Equal(t, 5, n)
	assert.Equal(t, "hello", string(buf))

	// Read more
	n, err = reader.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, 5, n)
	assert.Equal(t, " worl", string(buf))

	// Read remaining
	n, err = reader.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, 1, n)
	assert.Equal(t, "d", string(buf[:n]))

	// Read past end - bytes.Reader returns io.EOF
	n, err = reader.Read(buf)
	assert.Equal(t, 0, n)
	assert.Error(t, err) // io.EOF
}
