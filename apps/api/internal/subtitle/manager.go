package subtitle

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
)

// Manager coordinates subtitle file placement with database updates.
type Manager struct {
	placer    *Placer
	movieRepo repository.MovieRepositoryInterface
	seriesRepo repository.SeriesRepositoryInterface
}

// NewManager creates a subtitle Manager.
func NewManager(placer *Placer, movieRepo repository.MovieRepositoryInterface, seriesRepo repository.SeriesRepositoryInterface) *Manager {
	return &Manager{
		placer:     placer,
		movieRepo:  movieRepo,
		seriesRepo: seriesRepo,
	}
}

// PlaceAndRecord places a subtitle file on disk and updates the database record.
// mediaType is either "movie" or "series".
func (m *Manager) PlaceAndRecord(ctx context.Context, mediaID, mediaType string, req PlaceRequest) error {
	result, err := m.placer.Place(req)
	if err != nil {
		return fmt.Errorf("manager: place failed: %w", err)
	}

	// Update database — if this fails, clean up the orphaned file on disk
	var dbErr error
	switch mediaType {
	case "movie":
		dbErr = m.movieRepo.UpdateSubtitleStatus(ctx, mediaID,
			models.SubtitleStatusFound,
			result.SubtitlePath,
			result.Language,
			req.Score,
		)
	case "series":
		dbErr = m.seriesRepo.UpdateSubtitleStatus(ctx, mediaID,
			models.SubtitleStatusFound,
			result.SubtitlePath,
			result.Language,
			req.Score,
		)
	default:
		// Clean up the written file before returning
		if cleanErr := Cleanup(result.SubtitlePath); cleanErr != nil {
			slog.Warn("Failed to clean up subtitle after bad media type", "path", result.SubtitlePath, "error", cleanErr)
		}
		return fmt.Errorf("manager: unknown media type %q", mediaType)
	}

	if dbErr != nil {
		// Compensating action: remove the orphaned subtitle file
		if cleanErr := Cleanup(result.SubtitlePath); cleanErr != nil {
			slog.Warn("Failed to clean up subtitle after DB error", "path", result.SubtitlePath, "error", cleanErr)
		}
		return fmt.Errorf("manager: update %s subtitle status: %w", mediaType, dbErr)
	}

	slog.Info("Subtitle placed and recorded",
		"mediaID", mediaID,
		"mediaType", mediaType,
		"path", result.SubtitlePath,
		"language", result.Language,
		"score", req.Score,
	)

	return nil
}

// ClearSubtitleFields resets all subtitle-related DB fields for a media item.
func (m *Manager) ClearSubtitleFields(ctx context.Context, mediaID, mediaType string) error {
	switch mediaType {
	case "movie":
		return m.movieRepo.UpdateSubtitleStatus(ctx, mediaID,
			models.SubtitleStatusNotSearched, "", "", 0)
	case "series":
		return m.seriesRepo.UpdateSubtitleStatus(ctx, mediaID,
			models.SubtitleStatusNotSearched, "", "", 0)
	default:
		return fmt.Errorf("manager: unknown media type %q", mediaType)
	}
}

// CleanupAndClear removes subtitle files from disk and clears DB fields.
func (m *Manager) CleanupAndClear(ctx context.Context, mediaID, mediaType, subtitlePath string) error {
	if err := Cleanup(subtitlePath); err != nil {
		slog.Warn("Subtitle file cleanup failed", "path", subtitlePath, "error", err)
		// Continue to clear DB fields even if file cleanup fails
	}

	if err := m.ClearSubtitleFields(ctx, mediaID, mediaType); err != nil {
		return fmt.Errorf("manager: clear subtitle fields: %w", err)
	}

	slog.Info("Subtitle cleaned up and DB cleared",
		"mediaID", mediaID,
		"mediaType", mediaType,
	)

	return nil
}
