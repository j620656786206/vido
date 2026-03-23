package subtitle

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/subtitle/providers"
)

// --- Task 8.2: Language Factor ---

func TestScoreLanguage(t *testing.T) {
	tests := []struct {
		lang     string
		expected float64
	}{
		{"zh-Hant", 1.0},
		{"zh-TW", 1.0},
		{"zh-tw", 1.0},  // case-insensitive
		{"CHT", 1.0},
		{"繁體", 1.0},
		{"zh-Hans", 0.6},
		{"zh-CN", 0.6},
		{"zh-cn", 0.6},  // case-insensitive
		{"CHS", 0.6},
		{"簡體", 0.6},
		{"zh", 0.4},     // ambiguous
		{"en", 0.0},
		{"ja", 0.0},
		{"", 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.lang, func(t *testing.T) {
			assert.InDelta(t, tt.expected, scoreLanguage(tt.lang), 0.001)
		})
	}
}

// --- Task 8.3: Resolution Factor ---

func TestScoreResolution(t *testing.T) {
	tests := []struct {
		name      string
		mediaRes  string
		subRes    string
		expected  float64
	}{
		{"exact match 1080p", "1080p", "1080p", 1.0},
		{"exact match normalized", "FHD", "1080p", 1.0},
		{"exact match 4K", "4K", "2160p", 1.0},
		{"untagged subtitle", "1080p", "", 0.5},
		{"unknown media", "", "1080p", 0.5},
		{"both unknown", "", "", 0.5},
		{"mismatch", "1080p", "720p", 0.2},
		{"mismatch 4K vs 1080p", "2160p", "1080p", 0.2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.InDelta(t, tt.expected, scoreResolution(tt.mediaRes, tt.subRes), 0.001)
		})
	}
}

func TestNormalizeResolution(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"1080p", "1080p"},
		{"1080", "1080p"},
		{"FHD", "1080p"},
		{"FullHD", "1080p"},
		{"720p", "720p"},
		{"HD", "720p"},
		{"4K", "2160p"},
		{"UHD", "2160p"},
		{"480p", "480p"},
		{"SD", "480p"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, normalizeResolution(tt.input))
		})
	}
}

// --- Task 8.4: Source Trust Factor ---

func TestScoreSourceTrust(t *testing.T) {
	s := NewScorer(NewDefaultScorerConfig())

	assert.InDelta(t, 0.8, s.scoreSourceTrust("assrt"), 0.001)
	assert.InDelta(t, 0.7, s.scoreSourceTrust("opensubtitles"), 0.001)
	assert.InDelta(t, 0.6, s.scoreSourceTrust("zimuku"), 0.001)
	assert.InDelta(t, 0.5, s.scoreSourceTrust("unknown_provider"), 0.001)
}

// --- Task 8.5: Group Match Factor ---

func TestScoreGroup(t *testing.T) {
	tests := []struct {
		name     string
		group    string
		expected float64
	}{
		{"known group CHD", "CHD", 1.0},
		{"known group YYeTs", "YYeTs", 1.0},
		{"known group 幻櫻字幕組", "幻櫻字幕組", 1.0},
		{"unknown group", "SomeRandomGroup", 0.3},
		{"empty group", "", 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.InDelta(t, tt.expected, scoreGroup(tt.group), 0.001)
		})
	}
}

// --- Task 8.6: Downloads Factor ---

func TestScoreDownloads(t *testing.T) {
	tests := []struct {
		name     string
		count    int
		max      int
		expected float64
	}{
		{"max downloads", 1000, 1000, 1.0},
		{"half downloads", 500, 1000, 0.5},
		{"zero downloads", 0, 1000, 0.0},
		{"single result", 42, 42, 1.0},
		{"zero max", 0, 0, 0.0},
		{"negative downloads clamped", -5, 1000, 0.0},
		{"negative max", 100, -1, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.InDelta(t, tt.expected, scoreDownloads(tt.count, tt.max), 0.001)
		})
	}
}

// --- Task 8.7: Composite Scoring ---

func TestScorer_Score_CompositeScoring(t *testing.T) {
	s := NewScorer(NewDefaultScorerConfig())

	results := []providers.SubtitleResult{
		{
			ID:       "1",
			Source:   "assrt",
			Language: "zh-Hant",
			Resolution: "1080p",
			Group:    "CHD",
			Downloads: 1000,
		},
		{
			ID:       "2",
			Source:   "zimuku",
			Language: "zh-Hans",
			Resolution: "720p",
			Group:    "UnknownGroup",
			Downloads: 200,
		},
		{
			ID:       "3",
			Source:   "opensubtitles",
			Language: "en",
			Resolution: "1080p",
			Group:    "",
			Downloads: 5000,
		},
	}

	scored := s.Score(results, "1080p")
	require.Len(t, scored, 3)

	// Result 1 should be top: zh-Hant + matching res + high trust + known group
	assert.Equal(t, "1", scored[0].ID)
	assert.Greater(t, scored[0].Score, 0.8)
	assert.InDelta(t, 1.0, scored[0].ScoreBreakdown.Language, 0.001)
	assert.InDelta(t, 1.0, scored[0].ScoreBreakdown.Resolution, 0.001)

	// zh-Hant result (#1) should score highest
	// The remaining two may swap depending on downloads vs language weight trade-off
	// Verify that zh-Hant ranks first and scores are populated
	for _, r := range scored {
		assert.Greater(t, r.ScoreBreakdown.Language+r.ScoreBreakdown.Resolution+r.ScoreBreakdown.SourceTrust, 0.0,
			"all results should have non-zero breakdown components for %s", r.ID)
	}

	// English result should have 0 language score
	var enResult *ScoredResult
	for i := range scored {
		if scored[i].ID == "3" {
			enResult = &scored[i]
		}
	}
	require.NotNil(t, enResult)
	assert.InDelta(t, 0.0, enResult.ScoreBreakdown.Language, 0.001)
}

// --- Task 8.8: Sort Order ---

func TestScorer_Score_SortOrder(t *testing.T) {
	s := NewScorer(NewDefaultScorerConfig())

	results := []providers.SubtitleResult{
		{ID: "low", Source: "zimuku", Language: "en", Downloads: 100},
		{ID: "high", Source: "assrt", Language: "zh-Hant", Downloads: 500},
		{ID: "mid", Source: "opensubtitles", Language: "zh-Hans", Downloads: 300},
	}

	scored := s.Score(results, "1080p")
	require.Len(t, scored, 3)

	// Descending by score
	assert.Equal(t, "high", scored[0].ID)
	assert.GreaterOrEqual(t, scored[0].Score, scored[1].Score)
	assert.GreaterOrEqual(t, scored[1].Score, scored[2].Score)
}

func TestScorer_Score_TieBreakByDownloads(t *testing.T) {
	s := NewScorer(NewDefaultScorerConfig())

	// Two results with same language/source/resolution but different downloads
	results := []providers.SubtitleResult{
		{ID: "less-dl", Source: "assrt", Language: "zh-Hant", Downloads: 100},
		{ID: "more-dl", Source: "assrt", Language: "zh-Hant", Downloads: 500},
	}

	scored := s.Score(results, "")
	require.Len(t, scored, 2)

	// Same score components except downloads → higher downloads ranked first
	assert.Equal(t, "more-dl", scored[0].ID)
}

// --- Task 8.9: Empty Input ---

func TestScorer_Score_EmptyInput(t *testing.T) {
	s := NewScorer(NewDefaultScorerConfig())

	scored := s.Score(nil, "1080p")
	assert.Nil(t, scored)

	scored = s.Score([]providers.SubtitleResult{}, "1080p")
	assert.Nil(t, scored)
}

// --- Task 8.10: Custom Config ---

func TestScorer_Score_CustomConfig(t *testing.T) {
	// Create config where only language matters (weight = 1.0)
	config := ScorerConfig{
		WeightLanguage:   1.0,
		WeightResolution: 0.0,
		WeightTrust:      0.0,
		WeightGroup:      0.0,
		WeightDownloads:  0.0,
		ProviderTrust:    map[string]float64{},
	}
	s := NewScorer(config)

	results := []providers.SubtitleResult{
		{ID: "en", Language: "en", Source: "assrt", Downloads: 10000},
		{ID: "zh", Language: "zh-Hant", Source: "zimuku", Downloads: 1},
	}

	scored := s.Score(results, "1080p")
	require.Len(t, scored, 2)

	// With only language weight, zh-Hant should be first despite fewer downloads
	assert.Equal(t, "zh", scored[0].ID)
	assert.InDelta(t, 1.0, scored[0].Score, 0.001)
	assert.Equal(t, "en", scored[1].ID)
	assert.InDelta(t, 0.0, scored[1].Score, 0.001)
}

// --- CR Fix: Single result with zero downloads should score 1.0 (AC #6) ---

func TestScorer_Score_SingleResultZeroDownloads(t *testing.T) {
	s := NewScorer(NewDefaultScorerConfig())

	results := []providers.SubtitleResult{
		{ID: "only", Source: "assrt", Language: "zh-Hant", Downloads: 0},
	}

	scored := s.Score(results, "1080p")
	require.Len(t, scored, 1)

	// AC #6: single result scores 1.0 for downloads
	assert.InDelta(t, 1.0, scored[0].ScoreBreakdown.Downloads, 0.001)
}

// --- CR Fix: Negative downloads should not produce negative scores ---

func TestScorer_Score_NegativeDownloads(t *testing.T) {
	s := NewScorer(NewDefaultScorerConfig())

	results := []providers.SubtitleResult{
		{ID: "normal", Source: "assrt", Language: "zh-Hant", Downloads: 100},
		{ID: "negative", Source: "assrt", Language: "zh-Hant", Downloads: -5},
	}

	scored := s.Score(results, "1080p")
	require.Len(t, scored, 2)

	for _, r := range scored {
		assert.GreaterOrEqual(t, r.ScoreBreakdown.Downloads, 0.0,
			"downloads score should never be negative for %s", r.ID)
		assert.GreaterOrEqual(t, r.Score, 0.0,
			"composite score should never be negative for %s", r.ID)
	}
}

// --- CR Fix: Tertiary tie-breaker by ID for full determinism ---

func TestScorer_Score_TertiaryTieBreakByID(t *testing.T) {
	s := NewScorer(NewDefaultScorerConfig())

	// Identical scores and downloads — should tie-break by ID ascending
	results := []providers.SubtitleResult{
		{ID: "zzz", Source: "assrt", Language: "zh-Hant", Downloads: 100},
		{ID: "aaa", Source: "assrt", Language: "zh-Hant", Downloads: 100},
	}

	scored := s.Score(results, "1080p")
	require.Len(t, scored, 2)

	// With equal scores and downloads, "aaa" < "zzz" so "aaa" comes first
	assert.Equal(t, "aaa", scored[0].ID)
	assert.Equal(t, "zzz", scored[1].ID)
}

// --- CR Fix: Default weights must sum to 1.0 ---

func TestDefaultWeightsSumToOne(t *testing.T) {
	config := NewDefaultScorerConfig()
	sum := config.WeightLanguage + config.WeightResolution + config.WeightTrust +
		config.WeightGroup + config.WeightDownloads
	assert.InDelta(t, 1.0, sum, 0.001, "default weights must sum to 1.0")
}

func TestNewDefaultScorerConfig(t *testing.T) {
	config := NewDefaultScorerConfig()
	assert.InDelta(t, 0.4, config.WeightLanguage, 0.001)
	assert.InDelta(t, 0.2, config.WeightResolution, 0.001)
	assert.InDelta(t, 0.2, config.WeightTrust, 0.001)
	assert.InDelta(t, 0.1, config.WeightGroup, 0.001)
	assert.InDelta(t, 0.1, config.WeightDownloads, 0.001)
	assert.Len(t, config.ProviderTrust, 3)
}
