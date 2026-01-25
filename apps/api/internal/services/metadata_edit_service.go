// Package services provides business logic for the API.
package services

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/vido/api/internal/images"
	"github.com/vido/api/internal/models"
)

// MovieMetadataRepository defines the interface for movie metadata operations
type MovieMetadataRepository interface {
	FindByID(ctx context.Context, id string) (*models.Movie, error)
	Update(ctx context.Context, movie *models.Movie) error
}

// SeriesMetadataRepository defines the interface for series metadata operations
type SeriesMetadataRepository interface {
	FindByID(ctx context.Context, id string) (*models.Series, error)
	Update(ctx context.Context, series *models.Series) error
}

// MetadataEditService handles metadata editing operations for movies and series
type MetadataEditService struct {
	movieRepo      MovieMetadataRepository
	seriesRepo     SeriesMetadataRepository
	imageProcessor *images.ImageProcessor
	logger         *slog.Logger
}

// NewMetadataEditService creates a new MetadataEditService
func NewMetadataEditService(
	movieRepo MovieMetadataRepository,
	seriesRepo SeriesMetadataRepository,
	imageProcessor *images.ImageProcessor,
) *MetadataEditService {
	return &MetadataEditService{
		movieRepo:      movieRepo,
		seriesRepo:     seriesRepo,
		imageProcessor: imageProcessor,
		logger:         slog.Default(),
	}
}

// Compile-time interface verification
var _ MetadataEditor = (*MetadataEditService)(nil)
var _ PosterUploader = (*MetadataEditService)(nil)

// Exists checks if a media item exists
func (s *MetadataEditService) Exists(ctx context.Context, id string) (bool, error) {
	// Try movie first
	_, err := s.movieRepo.FindByID(ctx, id)
	if err == nil {
		return true, nil
	}

	// Try series
	_, err = s.seriesRepo.FindByID(ctx, id)
	if err == nil {
		return true, nil
	}

	return false, nil
}

// UpdateMetadata updates metadata for a movie or series
func (s *MetadataEditService) UpdateMetadata(ctx context.Context, req *UpdateMetadataRequest) (*UpdateMetadataResponse, error) {
	switch req.MediaType {
	case "movie":
		return s.updateMovieMetadata(ctx, req)
	case "series":
		return s.updateSeriesMetadata(ctx, req)
	default:
		// Default to movie
		return s.updateMovieMetadata(ctx, req)
	}
}

// updateMovieMetadata updates metadata for a movie
func (s *MetadataEditService) updateMovieMetadata(ctx context.Context, req *UpdateMetadataRequest) (*UpdateMetadataResponse, error) {
	movie, err := s.movieRepo.FindByID(ctx, req.ID)
	if err != nil {
		s.logger.Debug("Movie not found for metadata update",
			"id", req.ID,
			"error", err,
		)
		return nil, ErrUpdateMetadataNotFound
	}

	// Update fields
	movie.Title = req.Title
	if req.TitleEnglish != "" {
		movie.OriginalTitle = sql.NullString{String: req.TitleEnglish, Valid: true}
	}

	// Update year in release date
	if req.Year > 0 {
		movie.ReleaseDate = fmt.Sprintf("%d-01-01", req.Year)
	}

	// Update genres
	if len(req.Genres) > 0 {
		movie.Genres = req.Genres
	}

	// Update overview
	if req.Overview != "" {
		movie.Overview = sql.NullString{String: req.Overview, Valid: true}
	}

	// Update poster URL if provided
	if req.PosterURL != "" {
		movie.PosterPath = sql.NullString{String: req.PosterURL, Valid: true}
	}

	// Set metadata source to manual
	movie.MetadataSource = sql.NullString{String: string(models.MetadataSourceManual), Valid: true}

	// Update timestamp
	now := time.Now()
	movie.UpdatedAt = now

	// Save to database
	if err := s.movieRepo.Update(ctx, movie); err != nil {
		s.logger.Error("Failed to update movie metadata",
			"id", req.ID,
			"error", err,
		)
		return nil, fmt.Errorf("failed to update movie: %w", err)
	}

	s.logger.Info("Updated movie metadata",
		"id", req.ID,
		"title", req.Title,
	)

	return &UpdateMetadataResponse{
		ID:             movie.ID,
		Title:          movie.Title,
		MetadataSource: models.MetadataSourceManual,
		UpdatedAt:      now.Format(time.RFC3339),
	}, nil
}

// updateSeriesMetadata updates metadata for a series
func (s *MetadataEditService) updateSeriesMetadata(ctx context.Context, req *UpdateMetadataRequest) (*UpdateMetadataResponse, error) {
	series, err := s.seriesRepo.FindByID(ctx, req.ID)
	if err != nil {
		s.logger.Debug("Series not found for metadata update",
			"id", req.ID,
			"error", err,
		)
		return nil, ErrUpdateMetadataNotFound
	}

	// Update fields
	series.Title = req.Title
	if req.TitleEnglish != "" {
		series.OriginalTitle = sql.NullString{String: req.TitleEnglish, Valid: true}
	}

	// Update year in first air date
	if req.Year > 0 {
		series.FirstAirDate = fmt.Sprintf("%d-01-01", req.Year)
	}

	// Update genres
	if len(req.Genres) > 0 {
		series.Genres = req.Genres
	}

	// Update overview
	if req.Overview != "" {
		series.Overview = sql.NullString{String: req.Overview, Valid: true}
	}

	// Update poster URL if provided
	if req.PosterURL != "" {
		series.PosterPath = sql.NullString{String: req.PosterURL, Valid: true}
	}

	// Set metadata source to manual
	series.MetadataSource = sql.NullString{String: string(models.MetadataSourceManual), Valid: true}

	// Update timestamp
	now := time.Now()
	series.UpdatedAt = now

	// Save to database
	if err := s.seriesRepo.Update(ctx, series); err != nil {
		s.logger.Error("Failed to update series metadata",
			"id", req.ID,
			"error", err,
		)
		return nil, fmt.Errorf("failed to update series: %w", err)
	}

	s.logger.Info("Updated series metadata",
		"id", req.ID,
		"title", req.Title,
	)

	return &UpdateMetadataResponse{
		ID:             series.ID,
		Title:          series.Title,
		MetadataSource: models.MetadataSourceManual,
		UpdatedAt:      now.Format(time.RFC3339),
	}, nil
}

// UploadPoster uploads and processes a poster image
func (s *MetadataEditService) UploadPoster(ctx context.Context, req *UploadPosterRequest) (*UploadPosterResponse, error) {
	if s.imageProcessor == nil {
		return nil, fmt.Errorf("image processor not configured")
	}

	// Process the image
	result, err := s.imageProcessor.ProcessPoster(
		NewBytesReader(req.FileData),
		req.MediaID,
	)
	if err != nil {
		s.logger.Error("Failed to process poster image",
			"media_id", req.MediaID,
			"error", err,
		)
		return nil, fmt.Errorf("failed to process image: %w", err)
	}

	// Update the media item with the new poster path
	posterURL := s.imageProcessor.GetPosterURL(req.MediaID)
	thumbnailURL := s.imageProcessor.GetThumbnailURL(req.MediaID)

	// Try to update the poster path
	if err := s.updatePosterPath(ctx, req.MediaID, req.MediaType, posterURL); err != nil {
		s.logger.Warn("Failed to update poster path in database",
			"media_id", req.MediaID,
			"error", err,
		)
		// Don't fail - the image is still saved locally
	}

	s.logger.Info("Poster uploaded successfully",
		"media_id", req.MediaID,
		"original_size", result.OriginalSize,
		"processed_size", result.ProcessedSize,
	)

	return &UploadPosterResponse{
		PosterURL:    posterURL,
		ThumbnailURL: thumbnailURL,
	}, nil
}

// updatePosterPath updates only the poster path in the database
func (s *MetadataEditService) updatePosterPath(ctx context.Context, id, mediaType, posterURL string) error {
	switch mediaType {
	case "series":
		series, err := s.seriesRepo.FindByID(ctx, id)
		if err != nil {
			return err
		}
		series.PosterPath = sql.NullString{String: posterURL, Valid: true}
		series.UpdatedAt = time.Now()
		return s.seriesRepo.Update(ctx, series)
	default:
		movie, err := s.movieRepo.FindByID(ctx, id)
		if err != nil {
			return err
		}
		movie.PosterPath = sql.NullString{String: posterURL, Valid: true}
		movie.UpdatedAt = time.Now()
		return s.movieRepo.Update(ctx, movie)
	}
}

// FetchPosterFromURL fetches a poster from a URL and processes it
func (s *MetadataEditService) FetchPosterFromURL(ctx context.Context, mediaID, mediaType, url string) (*UploadPosterResponse, error) {
	// Fetch the image
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch image from URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch image: HTTP %d", resp.StatusCode)
	}

	// Check content type
	contentType := resp.Header.Get("Content-Type")
	validTypes := map[string]bool{
		"image/jpeg": true,
		"image/png":  true,
		"image/webp": true,
	}
	if !validTypes[contentType] {
		return nil, ErrPosterInvalidFormat
	}

	// Check content length
	if resp.ContentLength > 5*1024*1024 {
		return nil, ErrPosterTooLarge
	}

	// Read the body
	data := make([]byte, resp.ContentLength)
	_, err = resp.Body.Read(data)
	if err != nil {
		return nil, fmt.Errorf("failed to read image data: %w", err)
	}

	// Process as upload
	return s.UploadPoster(ctx, &UploadPosterRequest{
		MediaID:     mediaID,
		MediaType:   mediaType,
		FileData:    data,
		FileName:    "poster",
		ContentType: contentType,
		FileSize:    resp.ContentLength,
	})
}

// BytesReader wraps a byte slice for io.Reader interface
type BytesReader struct {
	*bytes.Reader
}

// NewBytesReader creates a new BytesReader
func NewBytesReader(data []byte) *BytesReader {
	return &BytesReader{bytes.NewReader(data)}
}
