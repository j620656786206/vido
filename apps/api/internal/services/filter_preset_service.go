// Package services FilterPresetService — Story 11.4 (P2-015).
//
// CRUD for saved discover filter presets. Presets persist in SQLite so they
// sync across browser sessions (AC #5). The filters payload is an opaque JSON
// string in the URL search-param shape — the service never inspects it beyond
// the model's json.Valid check.
package services

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
)

// ErrFilterPresetLimitReached is returned when creating a preset would exceed
// the FilterPresetMaxCount cap (AC/Task 1.4).
var ErrFilterPresetLimitReached = fmt.Errorf("filter preset limit reached: max %d presets", models.FilterPresetMaxCount)

// FilterPresetServiceInterface defines the contract for filter preset operations.
type FilterPresetServiceInterface interface {
	GetAllPresets(ctx context.Context) ([]models.FilterPreset, error)
	CreatePreset(ctx context.Context, req CreateFilterPresetRequest) (*models.FilterPreset, error)
	DeletePreset(ctx context.Context, id string) error
}

// CreateFilterPresetRequest is the input for creating a preset. Filters is a
// raw JSON string (URL-param shape); see models.FilterPreset.
type CreateFilterPresetRequest struct {
	Name    string `json:"name"`
	Filters string `json:"filters"`
}

// FilterPresetService implements FilterPresetServiceInterface.
type FilterPresetService struct {
	repo repository.FilterPresetRepositoryInterface
}

// Compile-time verification.
var _ FilterPresetServiceInterface = (*FilterPresetService)(nil)

// NewFilterPresetService builds a new FilterPresetService.
func NewFilterPresetService(repo repository.FilterPresetRepositoryInterface) *FilterPresetService {
	return &FilterPresetService{repo: repo}
}

func (s *FilterPresetService) GetAllPresets(ctx context.Context) ([]models.FilterPreset, error) {
	presets, err := s.repo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("get all presets: %w", err)
	}
	return presets, nil
}

func (s *FilterPresetService) CreatePreset(ctx context.Context, req CreateFilterPresetRequest) (*models.FilterPreset, error) {
	// Enforce the max-preset cap before doing any work (Task 1.4).
	count, err := s.repo.Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("count presets: %w", err)
	}
	if count >= models.FilterPresetMaxCount {
		return nil, ErrFilterPresetLimitReached
	}

	preset := &models.FilterPreset{
		Name:      strings.TrimSpace(req.Name),
		Filters:   strings.TrimSpace(req.Filters),
		SortOrder: count, // append at the bottom of the current list
	}

	if err := preset.Validate(); err != nil {
		return nil, fmt.Errorf("validation: %w", err)
	}

	if err := s.repo.Create(ctx, preset); err != nil {
		return nil, fmt.Errorf("create preset: %w", err)
	}

	slog.Info("Filter preset created", "id", preset.ID, "name", preset.Name)
	return preset, nil
}

func (s *FilterPresetService) DeletePreset(ctx context.Context, id string) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete preset: %w", err)
	}
	slog.Info("Filter preset deleted", "id", id)
	return nil
}
