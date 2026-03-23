package subtitle

import (
	"math"
	"sort"
	"strings"

	"github.com/vido/api/internal/subtitle/providers"
)

// scoreEpsilon is the threshold for treating two scores as equal.
const scoreEpsilon = 1e-9

// Scoring weight defaults (Gate 2A decision).
const (
	DefaultWeightLanguage   = 0.40
	DefaultWeightResolution = 0.20
	DefaultWeightTrust      = 0.20
	DefaultWeightGroup      = 0.10
	DefaultWeightDownloads  = 0.10
)

// ScorerConfig holds configurable weights and provider trust values.
type ScorerConfig struct {
	WeightLanguage   float64
	WeightResolution float64
	WeightTrust      float64
	WeightGroup      float64
	WeightDownloads  float64

	// ProviderTrust maps provider name → trust score (0.0–1.0).
	ProviderTrust map[string]float64
}

// NewDefaultScorerConfig returns the Gate 2A default weights and trust values.
func NewDefaultScorerConfig() ScorerConfig {
	return ScorerConfig{
		WeightLanguage:   DefaultWeightLanguage,
		WeightResolution: DefaultWeightResolution,
		WeightTrust:      DefaultWeightTrust,
		WeightGroup:      DefaultWeightGroup,
		WeightDownloads:  DefaultWeightDownloads,
		ProviderTrust: map[string]float64{
			"assrt":          0.8,
			"opensubtitles":  0.7,
			"zimuku":         0.6,
		},
	}
}

// ScoreBreakdown shows the individual factor scores for debugging/UI display.
type ScoreBreakdown struct {
	Language   float64 `json:"language"`
	Resolution float64 `json:"resolution"`
	SourceTrust float64 `json:"sourceTrust"`
	Group      float64 `json:"group"`
	Downloads  float64 `json:"downloads"`
}

// ScoredResult wraps a SubtitleResult with its composite score and breakdown.
type ScoredResult struct {
	providers.SubtitleResult
	Score          float64        `json:"score"`
	ScoreBreakdown ScoreBreakdown `json:"scoreBreakdown"`
}

// Scorer calculates and ranks subtitle search results.
type Scorer struct {
	config ScorerConfig
}

// NewScorer creates a Scorer with the given config.
func NewScorer(config ScorerConfig) *Scorer {
	return &Scorer{config: config}
}

// Score calculates composite scores for all results and returns them sorted
// descending by score (ties broken by download count descending).
func (s *Scorer) Score(results []providers.SubtitleResult, mediaResolution string) []ScoredResult {
	if len(results) == 0 {
		return nil
	}

	// Find max download count for normalization.
	// If only one result exists, it gets 1.0 for downloads per AC #6.
	maxDownloads := 0
	for _, r := range results {
		if r.Downloads > maxDownloads {
			maxDownloads = r.Downloads
		}
	}
	singleResult := len(results) == 1

	scored := make([]ScoredResult, len(results))
	for i, r := range results {
		var dlScore float64
		if singleResult {
			dlScore = 1.0 // AC #6: single result always scores 1.0 for downloads
		} else {
			dlScore = scoreDownloads(r.Downloads, maxDownloads)
		}

		bd := ScoreBreakdown{
			Language:    scoreLanguage(r.Language),
			Resolution:  scoreResolution(mediaResolution, r.Resolution),
			SourceTrust: s.scoreSourceTrust(r.Source),
			Group:       scoreGroup(r.Group),
			Downloads:   dlScore,
		}

		composite := bd.Language*s.config.WeightLanguage +
			bd.Resolution*s.config.WeightResolution +
			bd.SourceTrust*s.config.WeightTrust +
			bd.Group*s.config.WeightGroup +
			bd.Downloads*s.config.WeightDownloads

		scored[i] = ScoredResult{
			SubtitleResult: r,
			Score:          composite,
			ScoreBreakdown: bd,
		}
	}

	// Sort descending by score, then by downloads for ties, then by ID for determinism
	sort.SliceStable(scored, func(i, j int) bool {
		if math.Abs(scored[i].Score-scored[j].Score) > scoreEpsilon {
			return scored[i].Score > scored[j].Score
		}
		if scored[i].Downloads != scored[j].Downloads {
			return scored[i].Downloads > scored[j].Downloads
		}
		return scored[i].ID < scored[j].ID
	})

	return scored
}

// scoreLanguage returns the language factor (0.0–1.0).
// zh-Hant/zh-TW = 1.0, zh-Hans/zh-CN = 0.6 (convertible via OpenCC), other = 0.0.
func scoreLanguage(lang string) float64 {
	lower := strings.ToLower(lang)
	switch lower {
	case "zh-hant", "zh-tw", "cht", "繁體", "繁体":
		return 1.0
	case "zh-hans", "zh-cn", "chs", "簡體", "简体":
		return 0.6
	case "zh":
		return 0.4 // ambiguous Chinese — might be either variant
	default:
		return 0.0
	}
}

// scoreResolution returns the resolution factor (0.0–1.0).
// Exact match = 1.0, untagged = 0.5, mismatch = 0.2.
func scoreResolution(mediaRes, subtitleRes string) float64 {
	normMedia := normalizeResolution(mediaRes)
	normSub := normalizeResolution(subtitleRes)

	if normSub == "" {
		return 0.5 // untagged
	}
	if normMedia == "" {
		return 0.5 // unknown media resolution
	}
	if normMedia == normSub {
		return 1.0 // exact match
	}
	return 0.2 // mismatch
}

// normalizeResolution standardizes resolution strings.
func normalizeResolution(res string) string {
	lower := strings.ToLower(strings.TrimSpace(res))
	switch lower {
	case "2160p", "2160", "4k", "uhd":
		return "2160p"
	case "1080p", "1080", "fhd", "fullhd", "full hd":
		return "1080p"
	case "720p", "720", "hd":
		return "720p"
	case "480p", "480", "sd":
		return "480p"
	case "":
		return ""
	default:
		return lower
	}
}

// scoreSourceTrust returns the provider trust factor from config.
func (s *Scorer) scoreSourceTrust(providerName string) float64 {
	if trust, ok := s.config.ProviderTrust[providerName]; ok {
		return trust
	}
	return 0.5 // unknown provider default
}

// knownFansubGroups is an initial set of known Chinese fansub groups.
var knownFansubGroups = map[string]struct{}{
	"CHD":    {}, "CMCT":     {}, "MySiLU":  {}, "FLTth":   {}, "HDChina": {},
	"TTG":    {}, "TLF":      {}, "Wiki":    {}, "FRDS":    {}, "BMDru":   {},
	"beAst":  {}, "HDHome":   {}, "PTer":    {}, "CHDWEB":  {}, "ADWeb":   {},
	"YYeTs":  {}, "SubHD":    {}, "R3SUB":   {}, "SOFCJ":   {},
	"幻櫻字幕組": {}, "天使動漫":    {}, "極影字幕社":  {}, "動漫國字幕組": {},
	"Leopard-Raws": {}, "NC-Raws": {},
}

// scoreGroup returns the group match factor.
// Known group = 1.0, unknown non-empty = 0.3, empty = 0.0.
func scoreGroup(groupName string) float64 {
	if groupName == "" {
		return 0.0
	}
	if _, ok := knownFansubGroups[groupName]; ok {
		return 1.0
	}
	return 0.3
}

// scoreDownloads returns the downloads factor normalized to 0.0–1.0.
// Negative counts are clamped to 0.
func scoreDownloads(count int, maxCount int) float64 {
	if maxCount <= 0 {
		return 0.0
	}
	if count < 0 {
		count = 0
	}
	return float64(count) / float64(maxCount)
}
