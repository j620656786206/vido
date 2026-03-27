package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
	"github.com/vido/api/internal/testutil"
)

func setupExportMocks() (*testutil.MockMovieRepository, *testutil.MockSeriesRepository) {
	movieRepo := new(testutil.MockMovieRepository)
	seriesRepo := new(testutil.MockSeriesRepository)

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
		movieRepo := new(testutil.MockMovieRepository)
		seriesRepo := new(testutil.MockSeriesRepository)
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

		// Verify NFO file was created (strips .mkv extension per Kodi convention)
		nfoPath := filepath.Join(tmpDir, "The.Matrix.1999.nfo")
		_, err = os.Stat(nfoPath)
		assert.NoError(t, err)
	})

	t.Run("skips media without file path", func(t *testing.T) {
		movieRepo := new(testutil.MockMovieRepository)
		seriesRepo := new(testutil.MockSeriesRepository)
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

func TestExportService_ExportJSON_RepoError(t *testing.T) {
	ctx := context.Background()

	t.Run("movie repo error returns failed status", func(t *testing.T) {
		movieRepo := new(testutil.MockMovieRepository)
		seriesRepo := new(testutil.MockSeriesRepository)
		svc := NewExportService(movieRepo, seriesRepo, t.TempDir())

		movieRepo.On("List", mock.Anything, mock.AnythingOfType("repository.ListParams")).Return(
			[]models.Movie{}, (*repository.PaginationResult)(nil), assert.AnError,
		)

		result, err := svc.ExportJSON(ctx)
		assert.NoError(t, err) // Returns result, not error
		assert.Equal(t, ExportStatusFailed, result.Status)
		assert.Contains(t, result.Error, "EXPORT_FAILED")
	})
}

func TestExportService_ExportJSON_EmptyLibrary(t *testing.T) {
	ctx := context.Background()

	t.Run("empty library exports 0 items", func(t *testing.T) {
		movieRepo := new(testutil.MockMovieRepository)
		seriesRepo := new(testutil.MockSeriesRepository)
		svc := NewExportService(movieRepo, seriesRepo, t.TempDir())

		pagination := &repository.PaginationResult{Page: 1, PageSize: 100, TotalResults: 0, TotalPages: 1}
		movieRepo.On("List", mock.Anything, mock.AnythingOfType("repository.ListParams")).Return([]models.Movie{}, pagination, nil)
		seriesRepo.On("List", mock.Anything, mock.AnythingOfType("repository.ListParams")).Return([]models.Series{}, pagination, nil)

		result, err := svc.ExportJSON(ctx)
		assert.NoError(t, err)
		assert.Equal(t, ExportStatusCompleted, result.Status)
		assert.Equal(t, 0, result.ItemCount)
	})
}

func TestExportService_ExportJSON_FieldVerification(t *testing.T) {
	ctx := context.Background()

	t.Run("exported JSON includes all optional fields", func(t *testing.T) {
		movieRepo, seriesRepo := setupExportMocks()
		svc := NewExportService(movieRepo, seriesRepo, t.TempDir())

		result, err := svc.ExportJSON(ctx)
		assert.NoError(t, err)

		data, err := os.ReadFile(result.FilePath)
		assert.NoError(t, err)

		var doc ExportDocument
		assert.NoError(t, json.Unmarshal(data, &doc))

		// Verify movie fields
		movie := doc.Media[0]
		assert.Equal(t, "The Matrix", movie.OriginalTitle)
		assert.Equal(t, int64(603), movie.TMDbID)
		assert.Equal(t, 8.7, movie.Rating)
		assert.Equal(t, "一個年輕的電腦駭客...", movie.Overview)
		assert.Equal(t, []string{"科幻", "動作"}, movie.Genres)

		// Verify series fields
		series := doc.Media[1]
		assert.Equal(t, int64(12345), series.TMDbID)
		assert.Equal(t, "tv", series.MediaType)
	})
}

func TestExportService_ExportNFO_SeriesWithFilePath(t *testing.T) {
	ctx := context.Background()

	t.Run("creates NFO file for series with file path", func(t *testing.T) {
		movieRepo := new(testutil.MockMovieRepository)
		seriesRepo := new(testutil.MockSeriesRepository)
		tmpDir := t.TempDir()
		svc := NewExportService(movieRepo, seriesRepo, tmpDir)

		seriesFile := filepath.Join(tmpDir, "Ping.Pong.S01")
		os.MkdirAll(seriesFile, 0o755)

		pagination := &repository.PaginationResult{Page: 1, PageSize: 100, TotalResults: 0, TotalPages: 1}
		movieRepo.On("List", mock.Anything, mock.AnythingOfType("repository.ListParams")).Return([]models.Movie{}, pagination, nil)
		seriesRepo.On("List", mock.Anything, mock.AnythingOfType("repository.ListParams")).Return([]models.Series{
			{
				ID: "s1", Title: "乒乓", FirstAirDate: "2014",
				FilePath:  sql.NullString{String: seriesFile, Valid: true},
				Genres:    []string{"動畫"},
				CreatedAt: time.Now(),
			},
		}, pagination, nil)

		result, err := svc.ExportNFO(ctx)
		assert.NoError(t, err)
		assert.Equal(t, ExportStatusCompleted, result.Status)
		assert.Equal(t, 1, result.ItemCount)

		nfoPath := seriesFile + ".nfo"
		data, err := os.ReadFile(nfoPath)
		assert.NoError(t, err)
		assert.Contains(t, string(data), "<tvshow>")
		assert.Contains(t, string(data), "<title>乒乓</title>")
	})
}

func TestExportService_ExportNFO_MovieRepoError(t *testing.T) {
	ctx := context.Background()

	t.Run("movie repo error returns failed status", func(t *testing.T) {
		movieRepo := new(testutil.MockMovieRepository)
		seriesRepo := new(testutil.MockSeriesRepository)
		svc := NewExportService(movieRepo, seriesRepo, t.TempDir())

		movieRepo.On("List", mock.Anything, mock.AnythingOfType("repository.ListParams")).Return(
			[]models.Movie{}, (*repository.PaginationResult)(nil), assert.AnError,
		)

		result, err := svc.ExportNFO(ctx)
		assert.NoError(t, err)
		assert.Equal(t, ExportStatusFailed, result.Status)
		assert.Contains(t, result.Error, "fetch movies")
	})
}

func TestExportService_GetExportFilePath_MismatchedID(t *testing.T) {
	ctx := context.Background()

	t.Run("wrong ID returns error", func(t *testing.T) {
		svc := NewExportService(nil, nil, "")
		svc.lastResult = &ExportResult{ExportID: "e1", FilePath: "/tmp/export.json"}

		_, err := svc.GetExportFilePath(ctx, "wrong-id")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "export not found")
	})
}

func TestNFOGenerator_GenerateMovieNFO_MinimalData(t *testing.T) {
	gen := &NFOGenerator{}

	t.Run("minimal movie with no optional fields", func(t *testing.T) {
		movie := models.Movie{
			Title:       "Test Movie",
			ReleaseDate: "2026",
		}

		data := gen.GenerateMovieNFO(movie)
		xml := string(data)

		assert.Contains(t, xml, "<movie>")
		assert.Contains(t, xml, "<title>Test Movie</title>")
		assert.Contains(t, xml, "<year>2026</year>")
		assert.NotContains(t, xml, "<originaltitle>")
		assert.NotContains(t, xml, "<director>")
		assert.NotContains(t, xml, "<actor>")
	})
}

func TestNFOGenerator_GenerateMovieNFO_ActorLimit(t *testing.T) {
	gen := &NFOGenerator{}

	t.Run("limits actors to 10", func(t *testing.T) {
		// Build credits with 15 cast members
		var cast []CreditsPerson
		for i := 0; i < 15; i++ {
			cast = append(cast, CreditsPerson{Name: fmt.Sprintf("Actor %d", i), Character: fmt.Sprintf("Role %d", i)})
		}
		creditsJSON, _ := json.Marshal(CreditsData{Cast: cast})

		movie := models.Movie{
			Title:       "Big Cast",
			ReleaseDate: "2026",
			CreditsJSON: sql.NullString{String: string(creditsJSON), Valid: true},
		}

		data := gen.GenerateMovieNFO(movie)
		xml := string(data)

		// Count <name> occurrences — should be exactly 10
		count := 0
		for i := 0; i < len(xml)-6; i++ {
			if xml[i:i+6] == "<name>" {
				count++
			}
		}
		assert.Equal(t, 10, count)
	})
}

func TestNFOGenerator_GenerateMovieNFO_InvalidCreditsJSON(t *testing.T) {
	gen := &NFOGenerator{}

	t.Run("invalid credits JSON doesn't crash", func(t *testing.T) {
		movie := models.Movie{
			Title:       "Bad Credits",
			ReleaseDate: "2026",
			CreditsJSON: sql.NullString{String: "not valid json{{{", Valid: true},
		}

		data := gen.GenerateMovieNFO(movie)
		xml := string(data)

		// Should still generate valid XML, just without actors/directors
		assert.Contains(t, xml, "<movie>")
		assert.Contains(t, xml, "<title>Bad Credits</title>")
		assert.NotContains(t, xml, "<actor>")
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
