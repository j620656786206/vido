package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
)

// MockMovieRepo implements MovieRepositoryInterface for testing
type MockMovieRepoExport struct {
	mock.Mock
}

func (m *MockMovieRepoExport) List(ctx context.Context, params repository.ListParams) ([]models.Movie, *repository.PaginationResult, error) {
	args := m.Called(ctx, params)
	return args.Get(0).([]models.Movie), args.Get(1).(*repository.PaginationResult), args.Error(2)
}
func (m *MockMovieRepoExport) Create(ctx context.Context, movie *models.Movie) error {
	return m.Called(ctx, movie).Error(0)
}
func (m *MockMovieRepoExport) FindByID(ctx context.Context, id string) (*models.Movie, error) {
	return nil, nil
}
func (m *MockMovieRepoExport) FindByTMDbID(ctx context.Context, tmdbID int64) (*models.Movie, error) {
	return nil, nil
}
func (m *MockMovieRepoExport) FindByIMDbID(ctx context.Context, imdbID string) (*models.Movie, error) {
	return nil, nil
}
func (m *MockMovieRepoExport) Update(ctx context.Context, movie *models.Movie) error {
	return nil
}
func (m *MockMovieRepoExport) Delete(ctx context.Context, id string) error { return nil }
func (m *MockMovieRepoExport) SearchByTitle(ctx context.Context, title string, params repository.ListParams) ([]models.Movie, *repository.PaginationResult, error) {
	return nil, nil, nil
}
func (m *MockMovieRepoExport) FullTextSearch(ctx context.Context, query string, params repository.ListParams) ([]models.Movie, *repository.PaginationResult, error) {
	return nil, nil, nil
}
func (m *MockMovieRepoExport) Upsert(ctx context.Context, movie *models.Movie) error { return nil }
func (m *MockMovieRepoExport) FindByFilePath(ctx context.Context, filePath string) (*models.Movie, error) {
	return nil, nil
}
func (m *MockMovieRepoExport) GetDistinctGenres(ctx context.Context) ([]string, error) {
	return nil, nil
}
func (m *MockMovieRepoExport) GetYearRange(ctx context.Context) (int, int, error) { return 0, 0, nil }
func (m *MockMovieRepoExport) Count(ctx context.Context) (int, error)             { return 0, nil }

// MockSeriesRepoExport implements SeriesRepositoryInterface for testing
type MockSeriesRepoExport struct {
	mock.Mock
}

func (m *MockSeriesRepoExport) List(ctx context.Context, params repository.ListParams) ([]models.Series, *repository.PaginationResult, error) {
	args := m.Called(ctx, params)
	return args.Get(0).([]models.Series), args.Get(1).(*repository.PaginationResult), args.Error(2)
}
func (m *MockSeriesRepoExport) Create(ctx context.Context, series *models.Series) error {
	return nil
}
func (m *MockSeriesRepoExport) FindByID(ctx context.Context, id string) (*models.Series, error) {
	return nil, nil
}
func (m *MockSeriesRepoExport) FindByTMDbID(ctx context.Context, tmdbID int64) (*models.Series, error) {
	return nil, nil
}
func (m *MockSeriesRepoExport) FindByIMDbID(ctx context.Context, imdbID string) (*models.Series, error) {
	return nil, nil
}
func (m *MockSeriesRepoExport) Update(ctx context.Context, series *models.Series) error { return nil }
func (m *MockSeriesRepoExport) Delete(ctx context.Context, id string) error             { return nil }
func (m *MockSeriesRepoExport) SearchByTitle(ctx context.Context, title string, params repository.ListParams) ([]models.Series, *repository.PaginationResult, error) {
	return nil, nil, nil
}
func (m *MockSeriesRepoExport) FullTextSearch(ctx context.Context, query string, params repository.ListParams) ([]models.Series, *repository.PaginationResult, error) {
	return nil, nil, nil
}
func (m *MockSeriesRepoExport) Upsert(ctx context.Context, series *models.Series) error { return nil }
func (m *MockSeriesRepoExport) GetDistinctGenres(ctx context.Context) ([]string, error) {
	return nil, nil
}
func (m *MockSeriesRepoExport) GetYearRange(ctx context.Context) (int, int, error) {
	return 0, 0, nil
}
func (m *MockSeriesRepoExport) Count(ctx context.Context) (int, error) { return 0, nil }

func setupExportMocks() (*MockMovieRepoExport, *MockSeriesRepoExport) {
	movieRepo := new(MockMovieRepoExport)
	seriesRepo := new(MockSeriesRepoExport)

	movies := []models.Movie{
		{
			ID: "m1", Title: "駭客任務", ReleaseDate: "1999",
			OriginalTitle: sql.NullString{String: "The Matrix", Valid: true},
			TMDbID:        sql.NullInt64{Int64: 603, Valid: true},
			Genres:        []string{"科幻", "動作"},
			Overview:      sql.NullString{String: "一個年輕的電腦駭客...", Valid: true},
			VoteAverage:   sql.NullFloat64{Float64: 8.7, Valid: true},
			CreatedAt:     time.Now(),
		},
	}
	series := []models.Series{
		{
			ID: "s1", Title: "乒乓", FirstAirDate: "2014",
			TMDbID:    sql.NullInt64{Int64: 12345, Valid: true},
			Genres:    []string{"動畫"},
			CreatedAt: time.Now(),
		},
	}

	pagination := &repository.PaginationResult{Page: 1, PageSize: 100, TotalResults: 1, TotalPages: 1}
	movieRepo.On("List", mock.Anything, mock.AnythingOfType("repository.ListParams")).Return(movies, pagination, nil)
	seriesRepo.On("List", mock.Anything, mock.AnythingOfType("repository.ListParams")).Return(series, pagination, nil)

	return movieRepo, seriesRepo
}

func TestExportService_ExportJSON(t *testing.T) {
	ctx := context.Background()

	t.Run("successful JSON export", func(t *testing.T) {
		movieRepo, seriesRepo := setupExportMocks()
		exportDir := t.TempDir()
		svc := NewExportService(movieRepo, seriesRepo, exportDir)

		result, err := svc.ExportJSON(ctx)
		assert.NoError(t, err)
		assert.Equal(t, ExportStatusCompleted, result.Status)
		assert.Equal(t, 2, result.ItemCount)
		assert.NotEmpty(t, result.FilePath)

		// Verify file content
		data, err := os.ReadFile(result.FilePath)
		assert.NoError(t, err)

		var doc ExportDocument
		assert.NoError(t, json.Unmarshal(data, &doc))
		assert.Equal(t, "1.0", doc.ExportVersion)
		assert.Equal(t, 2, doc.ItemCount)
		assert.Equal(t, "駭客任務", doc.Media[0].Title)
		assert.Equal(t, "movie", doc.Media[0].MediaType)
		assert.Equal(t, "乒乓", doc.Media[1].Title)
		assert.Equal(t, "tv", doc.Media[1].MediaType)
	})

	t.Run("concurrent export rejected", func(t *testing.T) {
		movieRepo, seriesRepo := setupExportMocks()
		svc := NewExportService(movieRepo, seriesRepo, t.TempDir())
		svc.mu.Lock()
		svc.exporting = true
		svc.mu.Unlock()

		_, err := svc.ExportJSON(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already in progress")

		svc.mu.Lock()
		svc.exporting = false
		svc.mu.Unlock()
	})
}

func TestExportService_ExportYAML(t *testing.T) {
	ctx := context.Background()

	t.Run("successful YAML export", func(t *testing.T) {
		movieRepo, seriesRepo := setupExportMocks()
		exportDir := t.TempDir()
		svc := NewExportService(movieRepo, seriesRepo, exportDir)

		result, err := svc.ExportYAML(ctx)
		assert.NoError(t, err)
		assert.Equal(t, ExportStatusCompleted, result.Status)
		assert.Equal(t, 2, result.ItemCount)
		assert.Contains(t, result.FilePath, ".yaml")
	})
}

func TestExportService_ExportNFO(t *testing.T) {
	ctx := context.Background()

	t.Run("successful NFO export with file paths", func(t *testing.T) {
		movieRepo := new(MockMovieRepoExport)
		seriesRepo := new(MockSeriesRepoExport)
		tmpDir := t.TempDir()
		svc := NewExportService(movieRepo, seriesRepo, tmpDir)

		movieFile := filepath.Join(tmpDir, "The.Matrix.1999.mkv")
		os.WriteFile(movieFile, []byte("fake"), 0o644)

		movies := []models.Movie{
			{
				ID: "m1", Title: "駭客任務", ReleaseDate: "1999",
				FilePath:  sql.NullString{String: movieFile, Valid: true},
				Genres:    []string{"科幻"},
				CreatedAt: time.Now(),
			},
		}
		pagination := &repository.PaginationResult{Page: 1, PageSize: 100, TotalResults: 1, TotalPages: 1}
		movieRepo.On("List", mock.Anything, mock.AnythingOfType("repository.ListParams")).Return(movies, pagination, nil)
		seriesRepo.On("List", mock.Anything, mock.AnythingOfType("repository.ListParams")).Return([]models.Series{}, pagination, nil)

		result, err := svc.ExportNFO(ctx)
		assert.NoError(t, err)
		assert.Equal(t, ExportStatusCompleted, result.Status)
		assert.Equal(t, 1, result.ItemCount)

		// Verify NFO file was created
		nfoPath := movieFile + ".nfo"
		_, err = os.Stat(nfoPath)
		assert.NoError(t, err)
	})

	t.Run("skips media without file path", func(t *testing.T) {
		movieRepo := new(MockMovieRepoExport)
		seriesRepo := new(MockSeriesRepoExport)
		svc := NewExportService(movieRepo, seriesRepo, t.TempDir())

		movies := []models.Movie{
			{ID: "m1", Title: "No File", Genres: []string{}, CreatedAt: time.Now()},
		}
		pagination := &repository.PaginationResult{Page: 1, PageSize: 100, TotalResults: 1, TotalPages: 1}
		movieRepo.On("List", mock.Anything, mock.AnythingOfType("repository.ListParams")).Return(movies, pagination, nil)
		seriesRepo.On("List", mock.Anything, mock.AnythingOfType("repository.ListParams")).Return([]models.Series{}, pagination, nil)

		result, err := svc.ExportNFO(ctx)
		assert.NoError(t, err)
		assert.Equal(t, 0, result.ItemCount)
	})
}

func TestExportService_GetExportStatus(t *testing.T) {
	ctx := context.Background()

	t.Run("no export in progress", func(t *testing.T) {
		svc := NewExportService(nil, nil, "")
		result, err := svc.GetExportStatus(ctx)
		assert.NoError(t, err)
		assert.Equal(t, ExportStatusCompleted, result.Status)
	})
}

func TestExportService_GetExportFilePath(t *testing.T) {
	ctx := context.Background()

	t.Run("returns error when no export", func(t *testing.T) {
		svc := NewExportService(nil, nil, "")
		_, err := svc.GetExportFilePath(ctx, "nonexistent")
		assert.Error(t, err)
	})

	t.Run("returns path for matching export", func(t *testing.T) {
		svc := NewExportService(nil, nil, "")
		svc.lastResult = &ExportResult{ExportID: "e1", FilePath: "/tmp/export.json"}
		path, err := svc.GetExportFilePath(ctx, "e1")
		assert.NoError(t, err)
		assert.Equal(t, "/tmp/export.json", path)
	})
}

func TestNFOGenerator_GenerateMovieNFO(t *testing.T) {
	gen := &NFOGenerator{}

	t.Run("generates valid movie NFO XML", func(t *testing.T) {
		movie := models.Movie{
			Title:         "駭客任務",
			OriginalTitle: sql.NullString{String: "The Matrix", Valid: true},
			ReleaseDate:   "1999",
			Genres:        []string{"科幻", "動作"},
			Overview:      sql.NullString{String: "一個年輕的電腦駭客", Valid: true},
			VoteAverage:   sql.NullFloat64{Float64: 8.7, Valid: true},
			TMDbID:        sql.NullInt64{Int64: 603, Valid: true},
			IMDbID:        sql.NullString{String: "tt0133093", Valid: true},
			CreditsJSON:   sql.NullString{String: `{"cast":[{"name":"Keanu Reeves","character":"Neo"}],"crew":[{"name":"Lana Wachowski","job":"Director"}]}`, Valid: true},
		}

		data := gen.GenerateMovieNFO(movie)
		xml := string(data)

		assert.Contains(t, xml, "<movie>")
		assert.Contains(t, xml, "<title>駭客任務</title>")
		assert.Contains(t, xml, "<originaltitle>The Matrix</originaltitle>")
		assert.Contains(t, xml, "<year>1999</year>")
		assert.Contains(t, xml, "<genre>科幻</genre>")
		assert.Contains(t, xml, "<director>Lana Wachowski</director>")
		assert.Contains(t, xml, "<name>Keanu Reeves</name>")
		assert.Contains(t, xml, "<role>Neo</role>")
		assert.Contains(t, xml, `<uniqueid type="tmdb">603</uniqueid>`)
		assert.Contains(t, xml, `<uniqueid type="imdb">tt0133093</uniqueid>`)
	})
}

func TestNFOGenerator_GenerateSeriesNFO(t *testing.T) {
	gen := &NFOGenerator{}

	t.Run("generates valid series NFO XML", func(t *testing.T) {
		series := models.Series{
			Title:        "乒乓",
			FirstAirDate: "2014",
			Genres:       []string{"動畫"},
			TMDbID:       sql.NullInt64{Int64: 12345, Valid: true},
			Status:       sql.NullString{String: "Ended", Valid: true},
		}

		data := gen.GenerateSeriesNFO(series)
		xml := string(data)

		assert.Contains(t, xml, "<tvshow>")
		assert.Contains(t, xml, "<title>乒乓</title>")
		assert.Contains(t, xml, "<year>2014</year>")
		assert.Contains(t, xml, "<genre>動畫</genre>")
		assert.Contains(t, xml, "<status>Ended</status>")
	})
}
