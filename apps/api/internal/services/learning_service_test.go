package services

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/models"
)

// MockLearningRepository implements learning.LearningRepositoryInterface for testing
type MockLearningRepository struct {
	mappings []*models.FilenameMapping
}

func (m *MockLearningRepository) Save(ctx context.Context, mapping *models.FilenameMapping) error {
	m.mappings = append(m.mappings, mapping)
	return nil
}

func (m *MockLearningRepository) FindByID(ctx context.Context, id string) (*models.FilenameMapping, error) {
	for _, mapping := range m.mappings {
		if mapping.ID == id {
			return mapping, nil
		}
	}
	return nil, nil
}

func (m *MockLearningRepository) FindByExactPattern(ctx context.Context, pattern string) (*models.FilenameMapping, error) {
	for _, mapping := range m.mappings {
		if mapping.Pattern == pattern {
			return mapping, nil
		}
	}
	return nil, nil
}

func (m *MockLearningRepository) FindByFansubAndTitle(ctx context.Context, fansubGroup, titlePattern string) ([]*models.FilenameMapping, error) {
	var results []*models.FilenameMapping
	for _, mapping := range m.mappings {
		if mapping.FansubGroup == fansubGroup && mapping.TitlePattern == titlePattern {
			results = append(results, mapping)
		}
	}
	return results, nil
}

func (m *MockLearningRepository) ListWithRegex(ctx context.Context) ([]*models.FilenameMapping, error) {
	var results []*models.FilenameMapping
	for _, mapping := range m.mappings {
		if mapping.PatternRegex != "" {
			results = append(results, mapping)
		}
	}
	return results, nil
}

func (m *MockLearningRepository) ListAll(ctx context.Context) ([]*models.FilenameMapping, error) {
	return m.mappings, nil
}

func (m *MockLearningRepository) Delete(ctx context.Context, id string) error {
	for i, mapping := range m.mappings {
		if mapping.ID == id {
			m.mappings = append(m.mappings[:i], m.mappings[i+1:]...)
			return nil
		}
	}
	return nil
}

func (m *MockLearningRepository) IncrementUseCount(ctx context.Context, id string) error {
	for _, mapping := range m.mappings {
		if mapping.ID == id {
			mapping.UseCount++
			now := time.Now()
			mapping.LastUsedAt = &now
			return nil
		}
	}
	return nil
}

func (m *MockLearningRepository) Count(ctx context.Context) (int, error) {
	return len(m.mappings), nil
}

func (m *MockLearningRepository) Update(ctx context.Context, mapping *models.FilenameMapping) error {
	for i, existing := range m.mappings {
		if existing.ID == mapping.ID {
			m.mappings[i] = mapping
			return nil
		}
	}
	return nil
}

func TestLearningService_LearnFromCorrection(t *testing.T) {
	repo := &MockLearningRepository{}
	service := NewLearningService(repo)
	ctx := context.Background()

	// Learn from a fansub filename correction
	result, err := service.LearnFromCorrection(ctx, LearnFromCorrectionRequest{
		Filename:     "[Leopard-Raws] Kimetsu no Yaiba - 26 (BD 1920x1080 x264 FLAC).mkv",
		MetadataID:   "series-123",
		MetadataType: "series",
		TmdbID:       85937,
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	assert.NotEmpty(t, result.ID)
	assert.Equal(t, "fansub", result.PatternType)
	assert.Equal(t, "Leopard-Raws", result.FansubGroup)
	assert.Equal(t, "Kimetsu no Yaiba", result.TitlePattern)
	assert.Equal(t, "series", result.MetadataType)
	assert.Equal(t, "series-123", result.MetadataID)
	assert.Equal(t, 85937, result.TmdbID)

	// Verify it was saved to repo
	assert.Len(t, repo.mappings, 1)
}

func TestLearningService_FindMatchingPattern(t *testing.T) {
	repo := &MockLearningRepository{
		mappings: []*models.FilenameMapping{
			{
				ID:           "1",
				Pattern:      "[Leopard-Raws] Kimetsu no Yaiba",
				PatternType:  "fansub",
				PatternRegex: `(?i)[\[【]Leopard-Raws[\]】]\s*Kimetsu[.\s_-]+no[.\s_-]+Yaiba.*`,
				FansubGroup:  "Leopard-Raws",
				TitlePattern: "Kimetsu no Yaiba",
				MetadataType: "series",
				MetadataID:   "series-123",
				TmdbID:       85937,
			},
		},
	}

	service := NewLearningService(repo)
	ctx := context.Background()

	// Find matching pattern for similar filename
	result, err := service.FindMatchingPattern(ctx, "[Leopard-Raws] Kimetsu no Yaiba - 27 [1080p].mkv")
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "1", result.Pattern.ID)
	assert.GreaterOrEqual(t, result.Confidence, 0.9)
}

func TestLearningService_GetPatternStats(t *testing.T) {
	repo := &MockLearningRepository{
		mappings: []*models.FilenameMapping{
			{ID: "1", Pattern: "Pattern 1", UseCount: 10},
			{ID: "2", Pattern: "Pattern 2", UseCount: 5},
			{ID: "3", Pattern: "Pattern 3", UseCount: 20},
		},
	}

	service := NewLearningService(repo)
	ctx := context.Background()

	stats, err := service.GetPatternStats(ctx)
	require.NoError(t, err)
	require.NotNil(t, stats)

	assert.Equal(t, 3, stats.TotalPatterns)
	assert.Equal(t, 35, stats.TotalApplied)
}

func TestLearningService_ListPatterns(t *testing.T) {
	repo := &MockLearningRepository{
		mappings: []*models.FilenameMapping{
			{ID: "1", Pattern: "Pattern 1"},
			{ID: "2", Pattern: "Pattern 2"},
		},
	}

	service := NewLearningService(repo)
	ctx := context.Background()

	patterns, err := service.ListPatterns(ctx)
	require.NoError(t, err)
	assert.Len(t, patterns, 2)
}

func TestLearningService_DeletePattern(t *testing.T) {
	repo := &MockLearningRepository{
		mappings: []*models.FilenameMapping{
			{ID: "1", Pattern: "Pattern 1"},
		},
	}

	service := NewLearningService(repo)
	ctx := context.Background()

	// Delete pattern
	err := service.DeletePattern(ctx, "1")
	require.NoError(t, err)

	// Verify it was deleted
	assert.Len(t, repo.mappings, 0)
}

func TestLearningService_ApplyPattern(t *testing.T) {
	repo := &MockLearningRepository{
		mappings: []*models.FilenameMapping{
			{
				ID:           "1",
				Pattern:      "[Leopard-Raws] Kimetsu no Yaiba",
				MetadataType: "series",
				MetadataID:   "series-123",
				TmdbID:       85937,
				UseCount:     5,
			},
		},
	}

	service := NewLearningService(repo)
	ctx := context.Background()

	// Apply pattern (increment use count)
	err := service.ApplyPattern(ctx, "1")
	require.NoError(t, err)

	// Verify use count was incremented
	assert.Equal(t, 6, repo.mappings[0].UseCount)
}

func TestLearningService_LearnFromCorrection_DuplicatePrevented(t *testing.T) {
	repo := &MockLearningRepository{
		mappings: []*models.FilenameMapping{
			{
				ID:           "existing",
				Pattern:      "[Leopard-Raws] Kimetsu no Yaiba",
				PatternType:  "fansub",
				FansubGroup:  "Leopard-Raws",
				TitlePattern: "Kimetsu no Yaiba",
			},
		},
	}

	service := NewLearningService(repo)
	ctx := context.Background()

	// Try to learn the same pattern again
	result, err := service.LearnFromCorrection(ctx, LearnFromCorrectionRequest{
		Filename:     "[Leopard-Raws] Kimetsu no Yaiba - 26.mkv",
		MetadataID:   "series-456",
		MetadataType: "series",
		TmdbID:       85937,
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	// Should return existing pattern, not create new one
	assert.Equal(t, "existing", result.ID)
	assert.Len(t, repo.mappings, 1) // No new pattern added
}
