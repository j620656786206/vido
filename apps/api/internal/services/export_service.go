package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
	"gopkg.in/yaml.v3"
)

// ExportFormat represents supported export formats
type ExportFormat string

const (
	ExportFormatJSON ExportFormat = "json"
	ExportFormatYAML ExportFormat = "yaml"
	ExportFormatNFO  ExportFormat = "nfo"
)

// ExportStatus represents the current state of an export operation
type ExportStatus string

const (
	ExportStatusInProgress ExportStatus = "in_progress"
	ExportStatusCompleted  ExportStatus = "completed"
	ExportStatusFailed     ExportStatus = "failed"
)

// ExportResult contains the outcome of an export operation
type ExportResult struct {
	ExportID  string       `json:"exportId"`
	Format    ExportFormat `json:"format"`
	Status    ExportStatus `json:"status"`
	FilePath  string       `json:"-"` // internal only, not exposed to API
	ItemCount int          `json:"itemCount"`
	Message   string       `json:"message,omitempty"`
	Error     string       `json:"error,omitempty"`
}

// ExportMediaItem represents a single media item in the export
type ExportMediaItem struct {
	Title         string   `json:"title" yaml:"title"`
	OriginalTitle string   `json:"originalTitle,omitempty" yaml:"originalTitle,omitempty"`
	Year          string   `json:"year" yaml:"year"`
	MediaType     string   `json:"mediaType" yaml:"mediaType"`
	TMDbID        int64    `json:"tmdbId,omitempty" yaml:"tmdbId,omitempty"`
	IMDbID        string   `json:"imdbId,omitempty" yaml:"imdbId,omitempty"`
	Genres        []string `json:"genres" yaml:"genres"`
	Overview      string   `json:"overview,omitempty" yaml:"overview,omitempty"`
	PosterURL     string   `json:"posterUrl,omitempty" yaml:"posterUrl,omitempty"`
	FilePath      string   `json:"filePath,omitempty" yaml:"filePath,omitempty"`
	Rating        float64  `json:"rating,omitempty" yaml:"rating,omitempty"`
	AddedAt       string   `json:"addedAt" yaml:"addedAt"`
}

// ExportDocument is the top-level export structure
type ExportDocument struct {
	ExportVersion string            `json:"exportVersion" yaml:"exportVersion"`
	ExportedAt    string            `json:"exportedAt" yaml:"exportedAt"`
	ItemCount     int               `json:"itemCount" yaml:"itemCount"`
	Media         []ExportMediaItem `json:"media" yaml:"media"`
}

// ExportServiceInterface defines the contract for export operations
type ExportServiceInterface interface {
	ExportJSON(ctx context.Context) (*ExportResult, error)
	ExportYAML(ctx context.Context) (*ExportResult, error)
	ExportNFO(ctx context.Context) (*ExportResult, error)
	GetExportStatus(ctx context.Context) (*ExportResult, error)
	GetExportFilePath(ctx context.Context, id string) (string, error)
}

// ExportService manages metadata export operations
type ExportService struct {
	movieRepo  repository.MovieRepositoryInterface
	seriesRepo repository.SeriesRepositoryInterface
	exportDir  string
	mu         sync.Mutex
	exporting  bool
	lastResult *ExportResult
}

// Compile-time interface verification
var _ ExportServiceInterface = (*ExportService)(nil)

// NewExportService creates a new ExportService
func NewExportService(
	movieRepo repository.MovieRepositoryInterface,
	seriesRepo repository.SeriesRepositoryInterface,
	exportDir string,
) *ExportService {
	return &ExportService{
		movieRepo:  movieRepo,
		seriesRepo: seriesRepo,
		exportDir:  exportDir,
	}
}

// ExportJSON exports all media metadata to a JSON file
func (s *ExportService) ExportJSON(ctx context.Context) (*ExportResult, error) {
	s.mu.Lock()
	if s.exporting {
		s.mu.Unlock()
		return nil, fmt.Errorf("EXPORT_FAILED: another export is already in progress")
	}
	s.exporting = true
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		s.exporting = false
		s.mu.Unlock()
	}()

	result := &ExportResult{
		ExportID: uuid.New().String(),
		Format:   ExportFormatJSON,
		Status:   ExportStatusInProgress,
	}
	s.setResult(result)

	doc, err := s.buildExportDocument(ctx)
	if err != nil {
		result.Status = ExportStatusFailed
		result.Error = err.Error()
		s.setResult(result)
		return result, nil
	}

	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		result.Status = ExportStatusFailed
		result.Error = fmt.Sprintf("EXPORT_FAILED: marshal JSON: %v", err)
		s.setResult(result)
		return result, nil
	}

	filename := fmt.Sprintf("vido-export-%s.json", time.Now().Format("20060102-150405"))
	filePath, err := s.writeExportFile(filename, data)
	if err != nil {
		result.Status = ExportStatusFailed
		result.Error = fmt.Sprintf("EXPORT_FAILED: %v", err)
		s.setResult(result)
		return result, nil
	}

	result.Status = ExportStatusCompleted
	result.FilePath = filePath
	result.ItemCount = doc.ItemCount
	result.Message = fmt.Sprintf("匯出完成，共 %d 個項目", doc.ItemCount)
	s.setResult(result)

	slog.Info("JSON export completed", "export_id", result.ExportID, "items", doc.ItemCount)
	return result, nil
}

// ExportYAML exports all media metadata to a YAML file
func (s *ExportService) ExportYAML(ctx context.Context) (*ExportResult, error) {
	s.mu.Lock()
	if s.exporting {
		s.mu.Unlock()
		return nil, fmt.Errorf("EXPORT_FAILED: another export is already in progress")
	}
	s.exporting = true
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		s.exporting = false
		s.mu.Unlock()
	}()

	result := &ExportResult{
		ExportID: uuid.New().String(),
		Format:   ExportFormatYAML,
		Status:   ExportStatusInProgress,
	}
	s.setResult(result)

	doc, err := s.buildExportDocument(ctx)
	if err != nil {
		result.Status = ExportStatusFailed
		result.Error = err.Error()
		s.setResult(result)
		return result, nil
	}

	data, err := yaml.Marshal(doc)
	if err != nil {
		result.Status = ExportStatusFailed
		result.Error = fmt.Sprintf("EXPORT_FAILED: marshal YAML: %v", err)
		s.setResult(result)
		return result, nil
	}

	filename := fmt.Sprintf("vido-export-%s.yaml", time.Now().Format("20060102-150405"))
	filePath, err := s.writeExportFile(filename, data)
	if err != nil {
		result.Status = ExportStatusFailed
		result.Error = fmt.Sprintf("EXPORT_FAILED: %v", err)
		s.setResult(result)
		return result, nil
	}

	result.Status = ExportStatusCompleted
	result.FilePath = filePath
	result.ItemCount = doc.ItemCount
	result.Message = fmt.Sprintf("匯出完成，共 %d 個項目", doc.ItemCount)
	s.setResult(result)

	slog.Info("YAML export completed", "export_id", result.ExportID, "items", doc.ItemCount)
	return result, nil
}

// ExportNFO exports media metadata as Kodi-compatible .nfo files
func (s *ExportService) ExportNFO(ctx context.Context) (*ExportResult, error) {
	s.mu.Lock()
	if s.exporting {
		s.mu.Unlock()
		return nil, fmt.Errorf("EXPORT_FAILED: another export is already in progress")
	}
	s.exporting = true
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		s.exporting = false
		s.mu.Unlock()
	}()

	result := &ExportResult{
		ExportID: uuid.New().String(),
		Format:   ExportFormatNFO,
		Status:   ExportStatusInProgress,
	}
	s.setResult(result)

	nfoGen := &NFOGenerator{}
	count := 0

	// Export movies
	movies, err := s.fetchAllMovies(ctx)
	if err != nil {
		result.Status = ExportStatusFailed
		result.Error = fmt.Sprintf("EXPORT_FAILED: fetch movies: %v", err)
		s.setResult(result)
		return result, nil
	}

	for _, m := range movies {
		if !m.FilePath.Valid || m.FilePath.String == "" {
			continue
		}
		nfoData := nfoGen.GenerateMovieNFO(m)
		nfoPath := nfoFilePath(m.FilePath.String)
		if err := writeFile(nfoPath, nfoData); err != nil {
			slog.Warn("Failed to write movie NFO", "path", nfoPath, "error", err)
			continue
		}
		count++
	}

	// Export series
	series, err := s.fetchAllSeries(ctx)
	if err != nil {
		result.Status = ExportStatusFailed
		result.Error = fmt.Sprintf("EXPORT_FAILED: fetch series: %v", err)
		s.setResult(result)
		return result, nil
	}

	for _, sv := range series {
		if !sv.FilePath.Valid || sv.FilePath.String == "" {
			continue
		}
		nfoData := nfoGen.GenerateSeriesNFO(sv)
		nfoPath := nfoFilePath(sv.FilePath.String)
		if err := writeFile(nfoPath, nfoData); err != nil {
			slog.Warn("Failed to write series NFO", "path", nfoPath, "error", err)
			continue
		}
		count++
	}

	result.Status = ExportStatusCompleted
	result.ItemCount = count
	result.Message = fmt.Sprintf("NFO 匯出完成，共產生 %d 個 .nfo 檔案", count)
	s.setResult(result)

	slog.Info("NFO export completed", "export_id", result.ExportID, "nfo_files", count)
	return result, nil
}

// GetExportStatus returns the current/last export operation status
func (s *ExportService) GetExportStatus(ctx context.Context) (*ExportResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.lastResult == nil {
		return &ExportResult{
			Status:  ExportStatusCompleted,
			Message: "沒有進行中的匯出作業",
		}, nil
	}
	return s.lastResult, nil
}

// GetExportFilePath returns the file path for downloading an export
func (s *ExportService) GetExportFilePath(ctx context.Context, id string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.lastResult == nil || s.lastResult.ExportID != id {
		return "", fmt.Errorf("export not found: %s", id)
	}
	if s.lastResult.FilePath == "" {
		return "", fmt.Errorf("export has no downloadable file")
	}
	return s.lastResult.FilePath, nil
}

func (s *ExportService) setResult(result *ExportResult) {
	s.mu.Lock()
	s.lastResult = result
	s.mu.Unlock()
}

func (s *ExportService) buildExportDocument(ctx context.Context) (*ExportDocument, error) {
	items := make([]ExportMediaItem, 0)

	movies, err := s.fetchAllMovies(ctx)
	if err != nil {
		return nil, fmt.Errorf("EXPORT_FAILED: fetch movies: %v", err)
	}

	for _, m := range movies {
		item := ExportMediaItem{
			Title:     m.Title,
			Year:      m.ReleaseDate,
			MediaType: "movie",
			Genres:    m.Genres,
			AddedAt:   m.CreatedAt.Format(time.RFC3339),
		}
		if m.OriginalTitle.Valid {
			item.OriginalTitle = m.OriginalTitle.String
		}
		if m.TMDbID.Valid {
			item.TMDbID = m.TMDbID.Int64
		}
		if m.IMDbID.Valid {
			item.IMDbID = m.IMDbID.String
		}
		if m.Overview.Valid {
			item.Overview = m.Overview.String
		}
		if m.PosterPath.Valid {
			item.PosterURL = m.PosterPath.String
		}
		if m.FilePath.Valid {
			item.FilePath = m.FilePath.String
		}
		if m.VoteAverage.Valid {
			item.Rating = m.VoteAverage.Float64
		}
		if item.Genres == nil {
			item.Genres = []string{}
		}
		items = append(items, item)
	}

	series, err := s.fetchAllSeries(ctx)
	if err != nil {
		return nil, fmt.Errorf("EXPORT_FAILED: fetch series: %v", err)
	}

	for _, sv := range series {
		item := ExportMediaItem{
			Title:     sv.Title,
			Year:      sv.FirstAirDate,
			MediaType: "tv",
			Genres:    sv.Genres,
			AddedAt:   sv.CreatedAt.Format(time.RFC3339),
		}
		if sv.OriginalTitle.Valid {
			item.OriginalTitle = sv.OriginalTitle.String
		}
		if sv.TMDbID.Valid {
			item.TMDbID = sv.TMDbID.Int64
		}
		if sv.IMDbID.Valid {
			item.IMDbID = sv.IMDbID.String
		}
		if sv.Overview.Valid {
			item.Overview = sv.Overview.String
		}
		if sv.PosterPath.Valid {
			item.PosterURL = sv.PosterPath.String
		}
		if sv.FilePath.Valid {
			item.FilePath = sv.FilePath.String
		}
		if sv.VoteAverage.Valid {
			item.Rating = sv.VoteAverage.Float64
		}
		if item.Genres == nil {
			item.Genres = []string{}
		}
		items = append(items, item)
	}

	return &ExportDocument{
		ExportVersion: "1.0",
		ExportedAt:    time.Now().Format(time.RFC3339),
		ItemCount:     len(items),
		Media:         items,
	}, nil
}

func (s *ExportService) fetchAllMovies(ctx context.Context) ([]models.Movie, error) {
	params := repository.NewListParams()
	params.PageSize = repository.MaxPageSize
	params.SortBy = "created_at"
	params.SortOrder = "desc"

	var allMovies []models.Movie
	for {
		movies, pagination, err := s.movieRepo.List(ctx, params)
		if err != nil {
			return nil, err
		}
		allMovies = append(allMovies, movies...)
		if pagination == nil || params.Page >= pagination.TotalPages {
			break
		}
		params.Page++
	}
	return allMovies, nil
}

func (s *ExportService) fetchAllSeries(ctx context.Context) ([]models.Series, error) {
	params := repository.NewListParams()
	params.PageSize = repository.MaxPageSize
	params.SortBy = "created_at"
	params.SortOrder = "desc"

	var allSeries []models.Series
	for {
		series, pagination, err := s.seriesRepo.List(ctx, params)
		if err != nil {
			return nil, err
		}
		allSeries = append(allSeries, series...)
		if pagination == nil || params.Page >= pagination.TotalPages {
			break
		}
		params.Page++
	}
	return allSeries, nil
}

// nfoFilePath generates the NFO path for a media file.
// For files with known media extensions (.mkv, .mp4, etc.), strips the extension: movie.mkv → movie.nfo
// For directories or unknown extensions, appends .nfo: ShowDir → ShowDir.nfo
func nfoFilePath(mediaPath string) string {
	ext := strings.ToLower(filepath.Ext(mediaPath))
	mediaExts := map[string]bool{
		".mkv": true, ".mp4": true, ".avi": true, ".mov": true,
		".wmv": true, ".flv": true, ".ts": true, ".m4v": true,
	}
	if mediaExts[ext] {
		return strings.TrimSuffix(mediaPath, filepath.Ext(mediaPath)) + ".nfo"
	}
	return mediaPath + ".nfo"
}

func (s *ExportService) writeExportFile(filename string, data []byte) (string, error) {
	if err := os.MkdirAll(s.exportDir, 0o755); err != nil {
		return "", fmt.Errorf("create export dir: %w", err)
	}
	filePath := filepath.Join(s.exportDir, filename)
	if err := os.WriteFile(filePath, data, 0o644); err != nil {
		return "", fmt.Errorf("write export file: %w", err)
	}
	return filePath, nil
}
