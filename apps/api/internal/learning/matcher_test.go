package learning

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/models"
)

// MockLearningRepository implements LearningRepositoryInterface for testing
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
			return nil
		}
	}
	return nil
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

func (m *MockLearningRepository) Count(ctx context.Context) (int, error) {
	return len(m.mappings), nil
}

func TestPatternMatcher_FindMatch_ExactMatch(t *testing.T) {
	repo := &MockLearningRepository{
		mappings: []*models.FilenameMapping{
			{
				ID:           "1",
				Pattern:      "[Leopard-Raws] Kimetsu no Yaiba - 26 (BD 1920x1080 x264 FLAC).mkv",
				PatternType:  "exact",
				FansubGroup:  "Leopard-Raws",
				TitlePattern: "Kimetsu no Yaiba",
				MetadataType: "series",
				MetadataID:   "series-123",
				TmdbID:       85937,
			},
		},
	}

	matcher := NewPatternMatcher(repo)

	result, err := matcher.FindMatch(context.Background(), "[Leopard-Raws] Kimetsu no Yaiba - 26 (BD 1920x1080 x264 FLAC).mkv")
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "1", result.Pattern.ID)
	assert.Equal(t, 1.0, result.Confidence)
	assert.Equal(t, "exact", result.MatchType)
}

func TestPatternMatcher_FindMatch_FansubTitleMatch(t *testing.T) {
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

	matcher := NewPatternMatcher(repo)

	// Should match same fansub group + title but different episode
	result, err := matcher.FindMatch(context.Background(), "[Leopard-Raws] Kimetsu no Yaiba - 27 [1080p].mkv")
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "1", result.Pattern.ID)
	assert.GreaterOrEqual(t, result.Confidence, 0.9)
	assert.Equal(t, "pattern", result.MatchType)
}

func TestPatternMatcher_FindMatch_RegexMatch(t *testing.T) {
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

	matcher := NewPatternMatcher(repo)

	// Match via regex
	result, err := matcher.FindMatch(context.Background(), "[Leopard-Raws] Kimetsu no Yaiba - 100 [720p].mkv")
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "1", result.Pattern.ID)
	assert.GreaterOrEqual(t, result.Confidence, 0.9)
}

func TestPatternMatcher_FindMatch_FuzzyMatch(t *testing.T) {
	repo := &MockLearningRepository{
		mappings: []*models.FilenameMapping{
			{
				ID:           "1",
				Pattern:      "Breaking Bad",
				PatternType:  "standard",
				TitlePattern: "Breaking Bad",
				MetadataType: "series",
				MetadataID:   "series-456",
				TmdbID:       1396,
			},
		},
	}

	matcher := NewPatternMatcher(repo)

	// Fuzzy match with slightly different title
	result, err := matcher.FindMatch(context.Background(), "Braking.Bad.S01E01.mkv")
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "1", result.Pattern.ID)
	assert.Equal(t, "fuzzy", result.MatchType)
	assert.GreaterOrEqual(t, result.Confidence, 0.8)
}

func TestPatternMatcher_FindMatch_NoMatch(t *testing.T) {
	repo := &MockLearningRepository{
		mappings: []*models.FilenameMapping{
			{
				ID:           "1",
				Pattern:      "[Leopard-Raws] Kimetsu no Yaiba",
				PatternType:  "fansub",
				FansubGroup:  "Leopard-Raws",
				TitlePattern: "Kimetsu no Yaiba",
			},
		},
	}

	matcher := NewPatternMatcher(repo)

	// Completely different file
	result, err := matcher.FindMatch(context.Background(), "[Other-Group] Different Anime - 01.mkv")
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestPatternMatcher_FindMatch_PrioritizesExactOverFuzzy(t *testing.T) {
	repo := &MockLearningRepository{
		mappings: []*models.FilenameMapping{
			{
				ID:           "1",
				Pattern:      "[SubsPlease] Frieren - Beyond Journey's End - 01.mkv",
				PatternType:  "exact",
				FansubGroup:  "SubsPlease",
				TitlePattern: "Frieren - Beyond Journey's End",
				MetadataType: "series",
				MetadataID:   "series-123",
			},
			{
				ID:           "2",
				Pattern:      "Frieren",
				PatternType:  "fuzzy",
				TitlePattern: "Frieren",
				MetadataType: "series",
				MetadataID:   "series-456",
			},
		},
	}

	matcher := NewPatternMatcher(repo)

	// Should return exact match
	result, err := matcher.FindMatch(context.Background(), "[SubsPlease] Frieren - Beyond Journey's End - 01.mkv")
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "1", result.Pattern.ID)
	assert.Equal(t, "exact", result.MatchType)
}

func TestFuzzyMatch(t *testing.T) {
	tests := []struct {
		s1           string
		s2           string
		minSimilarity float64
	}{
		{"Breaking Bad", "Breaking Bad", 1.0},
		{"Breaking Bad", "Braking Bad", 0.8},
		{"Kimetsu no Yaiba", "kimetsu no yaiba", 1.0}, // Case insensitive
		{"Demon Slayer", "Demon Slayers", 0.9},
		{"Completely Different", "Another Title", 0.0}, // Very different
	}

	for _, tt := range tests {
		t.Run(tt.s1+"_vs_"+tt.s2, func(t *testing.T) {
			similarity := fuzzyMatch(tt.s1, tt.s2)
			assert.GreaterOrEqual(t, similarity, tt.minSimilarity,
				"Expected similarity >= %f for %q vs %q, got %f",
				tt.minSimilarity, tt.s1, tt.s2, similarity)
		})
	}
}

func TestMatchResult_String(t *testing.T) {
	result := &MatchResult{
		Pattern: &models.FilenameMapping{
			ID:           "1",
			TitlePattern: "Test Title",
		},
		Confidence: 0.95,
		MatchType:  "regex",
	}

	str := result.String()
	assert.Contains(t, str, "Test Title")
	assert.Contains(t, str, "0.95")
	assert.Contains(t, str, "regex")
}
