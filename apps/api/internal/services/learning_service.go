package services

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/vido/api/internal/learning"
)

// LearningServiceInterface defines the interface for the learning service
type LearningServiceInterface interface {
	LearnFromCorrection(ctx context.Context, req LearnFromCorrectionRequest) (*learning.FilenameMapping, error)
	FindMatchingPattern(ctx context.Context, filename string) (*learning.MatchResult, error)
	GetPatternStats(ctx context.Context) (*PatternStats, error)
	ListPatterns(ctx context.Context) ([]*learning.FilenameMapping, error)
	DeletePattern(ctx context.Context, id string) error
	ApplyPattern(ctx context.Context, id string) error
}

// LearnFromCorrectionRequest represents the request to learn from a user correction
type LearnFromCorrectionRequest struct {
	Filename     string `json:"filename"`
	MetadataID   string `json:"metadataId"`
	MetadataType string `json:"metadataType"` // "movie" or "series"
	TmdbID       int    `json:"tmdbId,omitempty"`
}

// PatternStats contains statistics about learned patterns
type PatternStats struct {
	TotalPatterns    int    `json:"totalPatterns"`
	TotalApplied     int    `json:"totalApplied"`
	MostUsedPattern  string `json:"mostUsedPattern,omitempty"`
	MostUsedCount    int    `json:"mostUsedCount,omitempty"`
}

// LearningService provides business logic for filename pattern learning
type LearningService struct {
	repo      learning.LearningRepositoryInterface
	extractor *learning.PatternExtractor
	matcher   *learning.PatternMatcher
}

// NewLearningService creates a new LearningService
func NewLearningService(repo learning.LearningRepositoryInterface) *LearningService {
	return &LearningService{
		repo:      repo,
		extractor: learning.NewPatternExtractor(),
		matcher:   learning.NewPatternMatcher(repo),
	}
}

// LearnFromCorrection learns a new pattern from a user's manual correction
func (s *LearningService) LearnFromCorrection(ctx context.Context, req LearnFromCorrectionRequest) (*learning.FilenameMapping, error) {
	if req.Filename == "" {
		return nil, fmt.Errorf("filename cannot be empty")
	}

	if req.MetadataID == "" {
		return nil, fmt.Errorf("metadataId cannot be empty")
	}

	if req.MetadataType != "movie" && req.MetadataType != "series" {
		return nil, fmt.Errorf("metadataType must be 'movie' or 'series'")
	}

	slog.Info("Learning from correction",
		"filename", req.Filename,
		"metadataId", req.MetadataID,
		"metadataType", req.MetadataType,
	)

	// Extract pattern from filename
	extracted, err := s.extractor.Extract(req.Filename)
	if err != nil {
		slog.Error("Failed to extract pattern", "error", err, "filename", req.Filename)
		return nil, fmt.Errorf("failed to extract pattern: %w", err)
	}

	// Check if similar pattern already exists
	existingMatch, err := s.matcher.FindMatch(ctx, req.Filename)
	if err != nil {
		slog.Warn("Error checking for existing pattern", "error", err)
		// Continue anyway - we'll try to save
	}

	if existingMatch != nil && existingMatch.Confidence >= 0.95 {
		slog.Info("Similar pattern already exists",
			"existingId", existingMatch.Pattern.ID,
			"confidence", existingMatch.Confidence,
		)
		return existingMatch.Pattern, nil
	}

	// Convert extracted pattern to filename mapping
	mapping := extracted.ToFilenameMapping(req.MetadataID, req.MetadataType, req.TmdbID)

	// Save to repository
	if err := s.repo.Save(ctx, mapping); err != nil {
		slog.Error("Failed to save pattern", "error", err)
		return nil, fmt.Errorf("failed to save pattern: %w", err)
	}

	slog.Info("Pattern learned successfully",
		"patternId", mapping.ID,
		"pattern", mapping.Pattern,
		"patternType", mapping.PatternType,
	)

	return mapping, nil
}

// FindMatchingPattern finds a matching pattern for a given filename
func (s *LearningService) FindMatchingPattern(ctx context.Context, filename string) (*learning.MatchResult, error) {
	if filename == "" {
		return nil, fmt.Errorf("filename cannot be empty")
	}

	result, err := s.matcher.FindMatch(ctx, filename)
	if err != nil {
		slog.Error("Failed to find matching pattern", "error", err, "filename", filename)
		return nil, fmt.Errorf("failed to find matching pattern: %w", err)
	}

	if result != nil {
		slog.Info("Found matching pattern",
			"patternId", result.Pattern.ID,
			"confidence", result.Confidence,
			"matchType", result.MatchType,
		)
	}

	return result, nil
}

// GetPatternStats returns statistics about learned patterns
func (s *LearningService) GetPatternStats(ctx context.Context) (*PatternStats, error) {
	patterns, err := s.repo.ListAll(ctx)
	if err != nil {
		slog.Error("Failed to list patterns for stats", "error", err)
		return nil, fmt.Errorf("failed to get pattern stats: %w", err)
	}

	stats := &PatternStats{
		TotalPatterns: len(patterns),
	}

	var mostUsed *learning.FilenameMapping
	for _, p := range patterns {
		stats.TotalApplied += p.UseCount
		if mostUsed == nil || p.UseCount > mostUsed.UseCount {
			mostUsed = p
		}
	}

	if mostUsed != nil && mostUsed.UseCount > 0 {
		stats.MostUsedPattern = mostUsed.Pattern
		stats.MostUsedCount = mostUsed.UseCount
	}

	return stats, nil
}

// ListPatterns returns all learned patterns
func (s *LearningService) ListPatterns(ctx context.Context) ([]*learning.FilenameMapping, error) {
	patterns, err := s.repo.ListAll(ctx)
	if err != nil {
		slog.Error("Failed to list patterns", "error", err)
		return nil, fmt.Errorf("failed to list patterns: %w", err)
	}

	return patterns, nil
}

// DeletePattern removes a learned pattern
func (s *LearningService) DeletePattern(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("pattern id cannot be empty")
	}

	slog.Info("Deleting pattern", "patternId", id)

	if err := s.repo.Delete(ctx, id); err != nil {
		slog.Error("Failed to delete pattern", "error", err, "patternId", id)
		return fmt.Errorf("failed to delete pattern: %w", err)
	}

	return nil
}

// ApplyPattern marks a pattern as used (increments use count)
func (s *LearningService) ApplyPattern(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("pattern id cannot be empty")
	}

	slog.Info("Applying pattern", "patternId", id)

	if err := s.repo.IncrementUseCount(ctx, id); err != nil {
		slog.Error("Failed to apply pattern", "error", err, "patternId", id)
		return fmt.Errorf("failed to apply pattern: %w", err)
	}

	return nil
}

// GetPatternByID retrieves a pattern by its ID
func (s *LearningService) GetPatternByID(ctx context.Context, id string) (*learning.FilenameMapping, error) {
	if id == "" {
		return nil, fmt.Errorf("pattern id cannot be empty")
	}

	pattern, err := s.repo.FindByID(ctx, id)
	if err != nil {
		slog.Error("Failed to get pattern", "error", err, "patternId", id)
		return nil, fmt.Errorf("failed to get pattern: %w", err)
	}

	return pattern, nil
}
