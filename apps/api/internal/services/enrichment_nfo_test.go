package services

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
	"github.com/vido/api/internal/tmdb"
)

// ─── Mock TMDb service for enrichment tests ─────────────────────────────────

type mockTMDbServiceForNFO struct {
	getMovieDetailsResp *tmdb.MovieDetails
	getMovieDetailsErr  error
	findByExtResp       *tmdb.FindByExternalIDResponse
	findByExtErr        error
}

func (m *mockTMDbServiceForNFO) SearchMovies(ctx context.Context, query string, page int) (*tmdb.SearchResultMovies, error) {
	return &tmdb.SearchResultMovies{}, nil
}

func (m *mockTMDbServiceForNFO) SearchTVShows(ctx context.Context, query string, page int) (*tmdb.SearchResultTVShows, error) {
	return &tmdb.SearchResultTVShows{}, nil
}

func (m *mockTMDbServiceForNFO) GetMovieDetails(ctx context.Context, movieID int) (*tmdb.MovieDetails, error) {
	if m.getMovieDetailsErr != nil {
		return nil, m.getMovieDetailsErr
	}
	return m.getMovieDetailsResp, nil
}

func (m *mockTMDbServiceForNFO) GetTVShowDetails(ctx context.Context, tvID int) (*tmdb.TVShowDetails, error) {
	return &tmdb.TVShowDetails{}, nil
}

func (m *mockTMDbServiceForNFO) FindByExternalID(ctx context.Context, externalID string, externalSource string) (*tmdb.FindByExternalIDResponse, error) {
	if m.findByExtErr != nil {
		return nil, m.findByExtErr
	}
	return m.findByExtResp, nil
}

// ─── Mock movie repo for enrichment tests ───────────────────────────────────

type mockMovieRepoForNFO struct {
	mock.Mock
	updatedMovie *models.Movie
}

func (m *mockMovieRepoForNFO) Update(ctx context.Context, movie *models.Movie) error {
	m.updatedMovie = movie
	return nil
}

func (m *mockMovieRepoForNFO) FindByParseStatus(ctx context.Context, status models.ParseStatus) ([]models.Movie, error) {
	return nil, nil
}

// Satisfy the rest of MovieRepositoryInterface with no-ops
func (m *mockMovieRepoForNFO) Create(ctx context.Context, movie *models.Movie) error { return nil }
func (m *mockMovieRepoForNFO) FindByID(ctx context.Context, id string) (*models.Movie, error) {
	return nil, nil
}
func (m *mockMovieRepoForNFO) FindByTMDbID(ctx context.Context, tmdbID int64) (*models.Movie, error) {
	return nil, nil
}
func (m *mockMovieRepoForNFO) FindByIMDbID(ctx context.Context, imdbID string) (*models.Movie, error) {
	return nil, nil
}
func (m *mockMovieRepoForNFO) Delete(ctx context.Context, id string) error { return nil }
func (m *mockMovieRepoForNFO) List(ctx context.Context, params repository.ListParams) ([]models.Movie, *repository.PaginationResult, error) {
	return nil, nil, nil
}
func (m *mockMovieRepoForNFO) SearchByTitle(ctx context.Context, title string, params repository.ListParams) ([]models.Movie, *repository.PaginationResult, error) {
	return nil, nil, nil
}
func (m *mockMovieRepoForNFO) FullTextSearch(ctx context.Context, query string, params repository.ListParams) ([]models.Movie, *repository.PaginationResult, error) {
	return nil, nil, nil
}
func (m *mockMovieRepoForNFO) Upsert(ctx context.Context, movie *models.Movie) error { return nil }
func (m *mockMovieRepoForNFO) FindByFilePath(ctx context.Context, filePath string) (*models.Movie, error) {
	return nil, nil
}
func (m *mockMovieRepoForNFO) GetDistinctGenres(ctx context.Context) ([]string, error) {
	return nil, nil
}
func (m *mockMovieRepoForNFO) GetYearRange(ctx context.Context) (int, int, error) { return 0, 0, nil }
func (m *mockMovieRepoForNFO) Count(ctx context.Context) (int, error)             { return 0, nil }
func (m *mockMovieRepoForNFO) BulkCreate(ctx context.Context, movies []*models.Movie) error {
	return nil
}
func (m *mockMovieRepoForNFO) UpdateSubtitleStatus(ctx context.Context, id string, status models.SubtitleStatus, path, language string, score float64) error {
	return nil
}
func (m *mockMovieRepoForNFO) FindBySubtitleStatus(ctx context.Context, status models.SubtitleStatus) ([]models.Movie, error) {
	return nil, nil
}
func (m *mockMovieRepoForNFO) FindNeedingSubtitleSearch(ctx context.Context, olderThan time.Time) ([]models.Movie, error) {
	return nil, nil
}
func (m *mockMovieRepoForNFO) FindAllWithFilePath(ctx context.Context) ([]models.Movie, error) {
	return nil, nil
}

// ─── Test: NFO enrichment — TMDB direct lookup (AC #2) ─────────────────────

func TestEnrichMovie_NFO_TMDbDirectLookup(t *testing.T) {
	dir := t.TempDir()
	videoPath := filepath.Join(dir, "Kung.Fu.Hustle.2004.mkv")
	nfoPath := filepath.Join(dir, "Kung.Fu.Hustle.2004.nfo")

	nfoContent := `<movie>
  <title>功夫</title>
  <uniqueid type="tmdb">10196</uniqueid>
  <fileinfo>
    <streamdetails>
      <video><codec>h265</codec><width>3840</width><height>2160</height></video>
      <audio><codec>dts</codec><channels>6</channels></audio>
    </streamdetails>
  </fileinfo>
</movie>`
	require.NoError(t, os.WriteFile(nfoPath, []byte(nfoContent), 0o644))

	posterPath := "/poster.jpg"
	mockTMDb := &mockTMDbServiceForNFO{
		getMovieDetailsResp: &tmdb.MovieDetails{
			Movie: tmdb.Movie{
				ID:          10196,
				Title:       "功夫",
				VoteAverage: 7.5,
				PosterPath:  &posterPath,
			},
			ImdbID: "tt0373074",
			Genres: []tmdb.Genre{{ID: 28, Name: "Action"}},
		},
	}

	mockRepo := &mockMovieRepoForNFO{}
	nfoReader := NewNFOReaderService(nil)

	svc := NewEnrichmentService(mockRepo, nil, nil, nfoReader, mockTMDb, nil, nil, nil)

	movie := &models.Movie{
		ID:       "test-id",
		Title:    "Kung.Fu.Hustle.2004.mkv",
		FilePath: models.NewNullString(videoPath),
	}

	err := svc.enrichMovie(context.Background(), movie)
	require.NoError(t, err)

	// Verify NFO data was applied
	assert.Equal(t, string(models.MetadataSourceNFO), mockRepo.updatedMovie.MetadataSource.String)
	assert.Equal(t, models.ParseStatusSuccess, mockRepo.updatedMovie.ParseStatus)
	assert.Equal(t, int64(10196), mockRepo.updatedMovie.TMDbID.Int64)
	assert.Equal(t, "功夫", mockRepo.updatedMovie.Title)
	assert.Equal(t, "tt0373074", mockRepo.updatedMovie.IMDbID.String)

	// Verify tech info from streamdetails (AC #5)
	assert.Equal(t, "h265", mockRepo.updatedMovie.VideoCodec.String)
	assert.Equal(t, "4K", mockRepo.updatedMovie.VideoResolution.String)
	assert.Equal(t, "dts", mockRepo.updatedMovie.AudioCodec.String)
	assert.Equal(t, int64(6), mockRepo.updatedMovie.AudioChannels.Int64)
}

// ─── Test: NFO enrichment — IMDB ID find (AC #3) ──────────────────────────

func TestEnrichMovie_NFO_IMDbLookup(t *testing.T) {
	dir := t.TempDir()
	videoPath := filepath.Join(dir, "Movie.mkv")
	nfoPath := filepath.Join(dir, "Movie.nfo")

	nfoContent := `<movie>
  <title>Test Movie</title>
  <uniqueid type="imdb">tt1234567</uniqueid>
</movie>`
	require.NoError(t, os.WriteFile(nfoPath, []byte(nfoContent), 0o644))

	posterPath := "/poster2.jpg"
	mockTMDb := &mockTMDbServiceForNFO{
		findByExtResp: &tmdb.FindByExternalIDResponse{
			MovieResults: []tmdb.Movie{{ID: 55555, Title: "找到的電影"}},
		},
		getMovieDetailsResp: &tmdb.MovieDetails{
			Movie: tmdb.Movie{
				ID:          55555,
				Title:       "找到的電影",
				VoteAverage: 8.0,
				PosterPath:  &posterPath,
			},
		},
	}

	mockRepo := &mockMovieRepoForNFO{}
	nfoReader := NewNFOReaderService(nil)
	svc := NewEnrichmentService(mockRepo, nil, nil, nfoReader, mockTMDb, nil, nil, nil)

	movie := &models.Movie{
		ID:       "test-id-2",
		Title:    "Movie.mkv",
		FilePath: models.NewNullString(videoPath),
	}

	err := svc.enrichMovie(context.Background(), movie)
	require.NoError(t, err)

	assert.Equal(t, string(models.MetadataSourceNFO), mockRepo.updatedMovie.MetadataSource.String)
	assert.Equal(t, int64(55555), mockRepo.updatedMovie.TMDbID.Int64)
}

// ─── Test: Manual source blocks NFO overwrite (AC #9) ──────────────────────

func TestEnrichMovie_NFO_ManualBlocksOverwrite(t *testing.T) {
	dir := t.TempDir()
	videoPath := filepath.Join(dir, "Movie.mkv")
	nfoPath := filepath.Join(dir, "Movie.nfo")

	nfoContent := `<movie><title>Movie</title><uniqueid type="tmdb">123</uniqueid></movie>`
	require.NoError(t, os.WriteFile(nfoPath, []byte(nfoContent), 0o644))

	mockRepo := &mockMovieRepoForNFO{}
	nfoReader := NewNFOReaderService(nil)
	mockTMDb := &mockTMDbServiceForNFO{}
	svc := NewEnrichmentService(mockRepo, nil, nil, nfoReader, mockTMDb, nil, nil, nil)

	movie := &models.Movie{
		ID:             "test-id-3",
		Title:          "Movie.mkv",
		FilePath:       models.NewNullString(videoPath),
		MetadataSource: models.NewNullString("manual"), // User corrected
	}

	// NFO should not overwrite manual — ShouldOverwrite("manual", "nfo") returns false
	enriched, err := svc.tryNFOEnrichment(context.Background(), movie)
	assert.NoError(t, err)
	assert.False(t, enriched) // NFO was rejected
	assert.Equal(t, "manual", movie.MetadataSource.String) // Unchanged
}

// ─── Test: No NFO file — falls through to AI parse (AC #7) ────────────────

func TestEnrichMovie_NoNFO_FallsThrough(t *testing.T) {
	dir := t.TempDir()
	videoPath := filepath.Join(dir, "Movie.mkv")
	// No .nfo file created

	mockRepo := &mockMovieRepoForNFO{}
	nfoReader := NewNFOReaderService(nil)
	svc := NewEnrichmentService(mockRepo, nil, nil, nfoReader, nil, nil, nil, nil)

	movie := &models.Movie{
		ID:       "test-id-4",
		Title:    "Movie.mkv",
		FilePath: models.NewNullString(videoPath),
	}

	// Without parser service, this will panic or nil-deref — so test just the tryNFOEnrichment
	enriched, err := svc.tryNFOEnrichment(context.Background(), movie)
	assert.NoError(t, err)
	assert.False(t, enriched)
}

// ─── Test: NFO re-scan idempotent (AC #8) ──────────────────────────────────

func TestEnrichMovie_NFO_RescanIdempotent(t *testing.T) {
	dir := t.TempDir()
	videoPath := filepath.Join(dir, "Movie.mkv")
	nfoPath := filepath.Join(dir, "Movie.nfo")

	nfoContent := `<movie><title>Movie</title><uniqueid type="tmdb">123</uniqueid></movie>`
	require.NoError(t, os.WriteFile(nfoPath, []byte(nfoContent), 0o644))

	mockTMDb := &mockTMDbServiceForNFO{
		getMovieDetailsResp: &tmdb.MovieDetails{
			Movie: tmdb.Movie{ID: 123, Title: "Movie"},
		},
	}

	mockRepo := &mockMovieRepoForNFO{}
	nfoReader := NewNFOReaderService(nil)
	svc := NewEnrichmentService(mockRepo, nil, nil, nfoReader, mockTMDb, nil, nil, nil)

	movie := &models.Movie{
		ID:             "test-id-5",
		Title:          "Movie.mkv",
		FilePath:       models.NewNullString(videoPath),
		MetadataSource: models.NewNullString("nfo"), // Already from NFO
	}

	// ShouldOverwrite("nfo", "nfo") returns true — re-scan is idempotent
	enriched, err := svc.tryNFOEnrichment(context.Background(), movie)
	require.NoError(t, err)
	assert.True(t, enriched)
	assert.Equal(t, string(models.MetadataSourceNFO), mockRepo.updatedMovie.MetadataSource.String)
}

// ─── Test: Malformed NFO falls back gracefully (AC #6) ─────────────────────

func TestEnrichMovie_NFO_MalformedFallsBack(t *testing.T) {
	dir := t.TempDir()
	videoPath := filepath.Join(dir, "Movie.mkv")
	nfoPath := filepath.Join(dir, "Movie.nfo")

	// Write malformed NFO
	require.NoError(t, os.WriteFile(nfoPath, []byte("<notvalid>broken</wrong>"), 0o644))

	mockRepo := &mockMovieRepoForNFO{}
	nfoReader := NewNFOReaderService(nil)
	svc := NewEnrichmentService(mockRepo, nil, nil, nfoReader, nil, nil, nil, nil)

	movie := &models.Movie{
		ID:       "test-id-6",
		Title:    "Movie.mkv",
		FilePath: models.NewNullString(videoPath),
	}

	// tryNFOEnrichment should return error but not crash
	enriched, err := svc.tryNFOEnrichment(context.Background(), movie)
	assert.Error(t, err)
	assert.False(t, enriched)
}

// ─── Test: NFO URL format enrichment (AC #4) ──────────────────────────────

func TestEnrichMovie_NFO_URLFormat(t *testing.T) {
	dir := t.TempDir()
	videoPath := filepath.Join(dir, "Movie.mkv")
	nfoPath := filepath.Join(dir, "Movie.nfo")

	require.NoError(t, os.WriteFile(nfoPath, []byte("https://www.themoviedb.org/movie/12345\n"), 0o644))

	mockTMDb := &mockTMDbServiceForNFO{
		getMovieDetailsResp: &tmdb.MovieDetails{
			Movie: tmdb.Movie{ID: 12345, Title: "URL Movie"},
		},
	}

	mockRepo := &mockMovieRepoForNFO{}
	nfoReader := NewNFOReaderService(nil)
	svc := NewEnrichmentService(mockRepo, nil, nil, nfoReader, mockTMDb, nil, nil, nil)

	movie := &models.Movie{
		ID:       "test-id-7",
		Title:    "Movie.mkv",
		FilePath: models.NewNullString(videoPath),
	}

	enriched, err := svc.tryNFOEnrichment(context.Background(), movie)
	require.NoError(t, err)
	assert.True(t, enriched)
	assert.Equal(t, string(models.MetadataSourceNFO), mockRepo.updatedMovie.MetadataSource.String)
	assert.Equal(t, int64(12345), mockRepo.updatedMovie.TMDbID.Int64)
}

// ─── Test: IMDB find returns no movie results (AC #3 edge case) ────────────

func TestEnrichMovie_NFO_IMDbNoMovieResults(t *testing.T) {
	dir := t.TempDir()
	videoPath := filepath.Join(dir, "Movie.mkv")
	nfoPath := filepath.Join(dir, "Movie.nfo")

	nfoContent := `<movie><title>Ghost</title><uniqueid type="imdb">tt0000001</uniqueid></movie>`
	require.NoError(t, os.WriteFile(nfoPath, []byte(nfoContent), 0o644))

	mockTMDb := &mockTMDbServiceForNFO{
		findByExtResp: &tmdb.FindByExternalIDResponse{
			MovieResults: []tmdb.Movie{}, // No results
		},
	}

	mockRepo := &mockMovieRepoForNFO{}
	nfoReader := NewNFOReaderService(nil)
	svc := NewEnrichmentService(mockRepo, nil, nil, nfoReader, mockTMDb, nil, nil, nil)

	movie := &models.Movie{
		ID:       "test-imdb-empty",
		Title:    "Movie.mkv",
		FilePath: models.NewNullString(videoPath),
	}

	// tryNFOEnrichment still succeeds (NFO was parsed), but TMDB lookup logged a warning
	enriched, err := svc.tryNFOEnrichment(context.Background(), movie)
	require.NoError(t, err)
	assert.True(t, enriched) // NFO data still applied (metadata_source=nfo)
	assert.Equal(t, string(models.MetadataSourceNFO), mockRepo.updatedMovie.MetadataSource.String)
}

// ─── Test: Invalid TMDB ID in NFO (non-numeric) ───────────────────────────

func TestEnrichMovie_NFO_InvalidTMDbID(t *testing.T) {
	dir := t.TempDir()
	videoPath := filepath.Join(dir, "Movie.mkv")
	nfoPath := filepath.Join(dir, "Movie.nfo")

	nfoContent := `<movie><title>Bad ID</title><uniqueid type="tmdb">not-a-number</uniqueid></movie>`
	require.NoError(t, os.WriteFile(nfoPath, []byte(nfoContent), 0o644))

	mockTMDb := &mockTMDbServiceForNFO{}
	mockRepo := &mockMovieRepoForNFO{}
	nfoReader := NewNFOReaderService(nil)
	svc := NewEnrichmentService(mockRepo, nil, nil, nfoReader, mockTMDb, nil, nil, nil)

	movie := &models.Movie{
		ID:       "test-bad-tmdb",
		Title:    "Movie.mkv",
		FilePath: models.NewNullString(videoPath),
	}

	// TMDB lookup fails due to invalid ID, but NFO enrichment still succeeds with warning
	enriched, err := svc.tryNFOEnrichment(context.Background(), movie)
	require.NoError(t, err)
	assert.True(t, enriched) // Still succeeds — NFO data applied, TMDB lookup failure is non-fatal
}

// ─── Test: Movie with no FilePath skips NFO ─────────��──────────────────────

func TestEnrichMovie_NFO_NoFilePath(t *testing.T) {
	mockRepo := &mockMovieRepoForNFO{}
	nfoReader := NewNFOReaderService(nil)
	svc := NewEnrichmentService(mockRepo, nil, nil, nfoReader, nil, nil, nil, nil)

	movie := &models.Movie{
		ID:    "test-no-filepath",
		Title: "Movie.mkv",
		// FilePath is not set (zero value NullString)
	}

	// tryNFOEnrichment should not be called when FilePath is empty
	// Test the enrichMovie guard: s.nfoReader != nil && movie.FilePath.Valid
	enriched, err := svc.tryNFOEnrichment(context.Background(), movie)
	assert.NoError(t, err)
	assert.False(t, enriched) // FindNFOSidecar("") returns ""
}

// ─── Test: NFO overwriting tmdb source (nfo priority > tmdb) ───────────────

func TestEnrichMovie_NFO_OverwritesTMDb(t *testing.T) {
	dir := t.TempDir()
	videoPath := filepath.Join(dir, "Movie.mkv")
	nfoPath := filepath.Join(dir, "Movie.nfo")

	nfoContent := `<movie><title>Better Data</title><uniqueid type="tmdb">999</uniqueid></movie>`
	require.NoError(t, os.WriteFile(nfoPath, []byte(nfoContent), 0o644))

	mockTMDb := &mockTMDbServiceForNFO{
		getMovieDetailsResp: &tmdb.MovieDetails{
			Movie: tmdb.Movie{ID: 999, Title: "Better Data (TMDB)"},
		},
	}

	mockRepo := &mockMovieRepoForNFO{}
	nfoReader := NewNFOReaderService(nil)
	svc := NewEnrichmentService(mockRepo, nil, nil, nfoReader, mockTMDb, nil, nil, nil)

	movie := &models.Movie{
		ID:             "test-overwrite-tmdb",
		Title:          "Movie.mkv",
		FilePath:       models.NewNullString(videoPath),
		MetadataSource: models.NewNullString("tmdb"), // Previously from TMDB
	}

	// ShouldOverwrite("tmdb", "nfo") → true (nfo priority 80 > tmdb priority 60)
	enriched, err := svc.tryNFOEnrichment(context.Background(), movie)
	require.NoError(t, err)
	assert.True(t, enriched)
	assert.Equal(t, string(models.MetadataSourceNFO), mockRepo.updatedMovie.MetadataSource.String)
}

// ─── Test: applyNFOTechInfo with partial data ────────────��─────────────────

func TestEnrichMovie_NFO_PartialStreamDetails(t *testing.T) {
	dir := t.TempDir()
	videoPath := filepath.Join(dir, "Movie.mkv")
	nfoPath := filepath.Join(dir, "Movie.nfo")

	// Only video codec, no audio, no resolution
	nfoContent := `<movie>
  <title>Partial</title>
  <uniqueid type="tmdb">111</uniqueid>
  <fileinfo>
    <streamdetails>
      <video><codec>av1</codec></video>
    </streamdetails>
  </fileinfo>
</movie>`
	require.NoError(t, os.WriteFile(nfoPath, []byte(nfoContent), 0o644))

	mockTMDb := &mockTMDbServiceForNFO{
		getMovieDetailsResp: &tmdb.MovieDetails{
			Movie: tmdb.Movie{ID: 111, Title: "Partial"},
		},
	}

	mockRepo := &mockMovieRepoForNFO{}
	nfoReader := NewNFOReaderService(nil)
	svc := NewEnrichmentService(mockRepo, nil, nil, nfoReader, mockTMDb, nil, nil, nil)

	movie := &models.Movie{
		ID:       "test-partial-stream",
		Title:    "Movie.mkv",
		FilePath: models.NewNullString(videoPath),
	}

	enriched, err := svc.tryNFOEnrichment(context.Background(), movie)
	require.NoError(t, err)
	assert.True(t, enriched)

	// Only video codec should be set
	assert.Equal(t, "av1", mockRepo.updatedMovie.VideoCodec.String)
	assert.True(t, mockRepo.updatedMovie.VideoCodec.Valid)
	assert.False(t, mockRepo.updatedMovie.VideoResolution.Valid) // No width/height → no resolution
	assert.False(t, mockRepo.updatedMovie.AudioCodec.Valid)      // No audio element
	assert.False(t, mockRepo.updatedMovie.AudioChannels.Valid)   // No audio channels
}
