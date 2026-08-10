package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"sort"

	"github.com/vido/api/internal/ai"
	"github.com/vido/api/internal/models"
)

// ─── Route prediction (Rule 19 mirror) ─────────────────────────────────────

// RoutePrediction mirrors subtitle.RoutePrediction.
//
// services ↛ subtitle — see project-context.md Rule 19. The values are kept
// byte-identical to the originals and pinned by a cross-package parity test
// (generation_candidates_parity_test.go), the same guard srt_parity_test.go
// gives the SRT mirror.
type RoutePrediction string

const (
	// RouteExtract — a usable embedded text track exists; no speech
	// recognition needed. NOT the same as free: an English track still pays
	// for LLM translation.
	RouteExtract RoutePrediction = "extract"
	// RouteASR — no usable embedded text track; speech recognition is the only
	// route. This is the expensive class the consent screen exists for.
	RouteASR RoutePrediction = "asr"
	// RouteSkipped — embedded text tracks exist but none qualifies (`und` or
	// another non-Chinese, non-eng/en tag). The pipeline declines these
	// outright, so they are not actionable and must never be quoted.
	RouteSkipped RoutePrediction = "skip"
)

// RoutePredictor is the narrow port over the subtitle package's probe-only
// classifier (Rule 11 + Rule 19). cmd/api adapts *subtitle.Router — the same
// bridge shape sub-3-2 used for pipelineASRAdapter.
type RoutePredictor interface {
	// FromTracks classifies an ALREADY-KNOWN track list without touching disk.
	FromTracks(tracks []SubtitleTrack) RoutePrediction
	// Probe classifies a file by probing it. Never extracts.
	Probe(ctx context.Context, mediaPath string) (RoutePrediction, error)
}

// ─── Enumeration ports ─────────────────────────────────────────────────────

// CandidateMovieFinder / CandidateEpisodeFinder are the narrow enumeration
// surfaces (Rule 11). Both are satisfied by the existing repositories, whose
// shared "missing zh-Hant" predicate stays the single source of truth for what
// counts as missing — this service adds no SQL of its own.
type CandidateMovieFinder interface {
	FindMissingZhHantSubtitle(ctx context.Context) ([]models.Movie, error)
}

type CandidateEpisodeFinder interface {
	FindMissingZhHantSubtitle(ctx context.Context) ([]models.Episode, error)
}

// ─── Cost model ────────────────────────────────────────────────────────────

// unknownRuntimeMinutes is what an item with no measurable duration is priced
// at. The UI states this assumption verbatim ("片長未知，以 45 分鐘估算") — an
// estimate the user cannot see the assumption behind is not an estimate.
const unknownRuntimeMinutes = 45.0

// translationUSDPerMinute prices the LLM half.
//
// It cannot be computed exactly before extraction — cost scales with cue count,
// which is unknown until the track is parsed — so this is a per-minute average
// calibrated against the M1 pilot's observed ~$0.03 for a feature-length item.
// It is deliberately a named constant rather than a magic number so a future
// story can recalibrate it from `subtitle_runs` usage data.
const translationUSDPerMinute = 0.0004

// GenerationCandidate is one row of the cost-preview screen.
type GenerationCandidate struct {
	MediaID   string          `json:"media_id"`
	MediaType string          `json:"media_type"`
	Title     string          `json:"title"`
	Route     RoutePrediction `json:"route"`
	// RuntimeMinutes is what the estimate was computed from; RuntimeKnown is
	// false when it fell back to unknownRuntimeMinutes.
	RuntimeMinutes float64 `json:"runtime_minutes"`
	RuntimeKnown   bool    `json:"runtime_known"`
	EstimatedUSD   float64 `json:"estimated_usd"`
}

// GenerationCandidateSummary is the aggregate the footer renders.
type GenerationCandidateSummary struct {
	ExtractCount int `json:"extract_count"`
	ASRCount     int `json:"asr_count"`
	// SkippedCount are items the pipeline would decline outright. They are
	// reported so the numbers add up, but are NOT included in Candidates.
	SkippedCount      int     `json:"skipped_count"`
	EstimatedTotalUSD float64 `json:"estimated_total_usd"`
	// SelfHostedASR tells the client the ASR rate applied was zero because the
	// endpoint is self-hosted, so it can say so instead of showing a
	// suspicious $0.00.
	SelfHostedASR bool `json:"self_hosted_asr"`
}

// GenerationCandidateResult is the whole preview payload.
type GenerationCandidateResult struct {
	Candidates []GenerationCandidate      `json:"candidates"`
	Summary    GenerationCandidateSummary `json:"summary"`
}

// GenerationCandidateService answers "what would generating subtitles cost?"
// for the whole library, WITHOUT spending anything: it probes (cheap, local)
// but never extracts and never transcribes.
//
// Story sub-4-1. It exists because scanning no longer enqueues generation —
// the user picks from this list instead.
type GenerationCandidateService struct {
	movies    CandidateMovieFinder
	episodes  CandidateEpisodeFinder
	predictor RoutePredictor
	logger    *slog.Logger

	// selfHostedASR mirrors "ASR_BASE_URL points somewhere other than the paid
	// API". Wired from config; see ai.EstimatedASRPerMinuteUSD.
	selfHostedASR bool
}

func NewGenerationCandidateService(
	movies CandidateMovieFinder,
	episodes CandidateEpisodeFinder,
	predictor RoutePredictor,
	selfHostedASR bool,
	logger *slog.Logger,
) *GenerationCandidateService {
	if logger == nil {
		logger = slog.Default()
	}
	return &GenerationCandidateService{
		movies:        movies,
		episodes:      episodes,
		predictor:     predictor,
		logger:        logger.With("service", "generation_candidates"),
		selfHostedASR: selfHostedASR,
	}
}

// AnalysisProgress reports how far the probe sweep has got. Emitted per item so
// the UI can show "分析字幕軌 234 / 1,247" — the pass is real work (one ffprobe
// per un-enriched file, throttled by the shared semaphore) and must not be
// presented as instant.
type AnalysisProgress func(done, total int)

// Analyze enumerates every item missing a zh-Hant subtitle, classifies its
// route, and prices it.
//
// Classification takes the cheap path whenever it can: movies carry their
// scan-time probe result in `subtitle_tracks`, so they need no disk access.
// Everything else is probed live — episodes always (their table has no
// tech-info columns at all), plus movies whose enrichment was short-circuited
// by NFO-sourced tech info or whose probe failed.
//
// A per-item probe failure degrades that item to "unknown route" and drops it
// from the quote rather than failing the sweep: one unreadable file must not
// deny the user a price for the other 1,200.
func (s *GenerationCandidateService) Analyze(ctx context.Context, progress AnalysisProgress) (*GenerationCandidateResult, error) {
	rows, err := s.enumerate(ctx)
	if err != nil {
		return nil, err
	}

	asrRate := ai.EstimatedASRPerMinuteUSD(s.selfHostedASR)
	result := &GenerationCandidateResult{
		Candidates: make([]GenerationCandidate, 0, len(rows)),
		Summary:    GenerationCandidateSummary{SelfHostedASR: s.selfHostedASR},
	}

	for i, row := range rows {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		route, ok := s.classify(ctx, row)
		if progress != nil {
			progress(i+1, len(rows))
		}
		if !ok {
			continue // unreadable — already logged, and not quotable
		}

		switch route {
		case RouteSkipped:
			result.Summary.SkippedCount++
			continue
		case RouteASR:
			result.Summary.ASRCount++
		case RouteExtract:
			result.Summary.ExtractCount++
		}

		minutes, known := row.runtimeMinutes()
		c := GenerationCandidate{
			MediaID:        row.id,
			MediaType:      row.mediaType,
			Title:          row.title,
			Route:          route,
			RuntimeMinutes: minutes,
			RuntimeKnown:   known,
			EstimatedUSD:   estimateUSD(route, minutes, asrRate),
		}
		result.Candidates = append(result.Candidates, c)
		result.Summary.EstimatedTotalUSD += c.EstimatedUSD
	}

	// Sub-cent noise in a float sum reads as a bug in a money field.
	result.Summary.EstimatedTotalUSD = roundUSD(result.Summary.EstimatedTotalUSD)

	s.logger.Info("generation candidate analysis complete",
		"candidates", len(result.Candidates),
		"extract", result.Summary.ExtractCount,
		"asr", result.Summary.ASRCount,
		"skipped", result.Summary.SkippedCount,
		"estimated_total_usd", result.Summary.EstimatedTotalUSD,
		"self_hosted_asr", s.selfHostedASR,
	)
	return result, nil
}

// estimateUSD prices one item.
//
// The ASR leg pays for audio minutes AND the translation that follows it; the
// extract leg pays only for translation. Note an extract item is therefore
// rarely exactly $0 — labelling it "free" in the UI is a rounding decision the
// client makes, not a claim this function makes.
func estimateUSD(route RoutePrediction, minutes, asrRate float64) float64 {
	switch route {
	case RouteASR:
		return roundUSD(minutes*asrRate + minutes*translationUSDPerMinute)
	case RouteExtract:
		return roundUSD(minutes * translationUSDPerMinute)
	default:
		return 0
	}
}

func roundUSD(v float64) float64 { return math.Round(v*100) / 100 }

// candidateRow is the media-type-neutral shape the estimator works in, so the
// movie and episode halves converge immediately after enumeration.
type candidateRow struct {
	id        string
	mediaType string
	title     string
	filePath  string
	runtime   models.NullInt64
	// tracksJSON is the persisted scan-time probe result. Empty for episodes,
	// which have no such column.
	tracksJSON string
}

// runtimeMinutes returns the metadata runtime and whether it was known.
func (r candidateRow) runtimeMinutes() (float64, bool) {
	if r.runtime.Valid && r.runtime.Int64 > 0 {
		return float64(r.runtime.Int64), true
	}
	return unknownRuntimeMinutes, false
}

func (s *GenerationCandidateService) enumerate(ctx context.Context) ([]candidateRow, error) {
	var rows []candidateRow

	if s.movies != nil {
		movies, err := s.movies.FindMissingZhHantSubtitle(ctx)
		if err != nil {
			return nil, fmt.Errorf("enumerate movies missing zh-Hant subtitle: %w", err)
		}
		for _, m := range movies {
			if !m.FilePath.Valid || m.FilePath.String == "" {
				continue
			}
			rows = append(rows, candidateRow{
				id:         m.ID,
				mediaType:  models.SubtitleRunMediaMovie,
				title:      m.Title,
				filePath:   m.FilePath.String,
				runtime:    m.Runtime,
				tracksJSON: m.SubtitleTracks.String,
			})
		}
	}

	if s.episodes != nil {
		episodes, err := s.episodes.FindMissingZhHantSubtitle(ctx)
		if err != nil {
			return nil, fmt.Errorf("enumerate episodes missing zh-Hant subtitle: %w", err)
		}
		for _, e := range episodes {
			if !e.FilePath.Valid || e.FilePath.String == "" {
				continue
			}
			rows = append(rows, candidateRow{
				id:        e.ID,
				mediaType: models.SubtitleRunMediaEpisode,
				title:     episodeTitle(e),
				filePath:  e.FilePath.String,
				runtime:   e.Runtime,
			})
		}
	}

	// Stable order so the list does not reshuffle between two previews of the
	// same library.
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].title != rows[j].title {
			return rows[i].title < rows[j].title
		}
		return rows[i].id < rows[j].id
	})
	return rows, nil
}

// episodeTitle renders the SxxEyy label the list shows. The episode's own title
// is optional in the schema, so the season/episode numbers carry the identity.
func episodeTitle(e models.Episode) string {
	label := fmt.Sprintf("S%02dE%02d", e.SeasonNumber, e.EpisodeNumber)
	if e.Title.Valid && e.Title.String != "" {
		return e.Title.String + " " + label
	}
	return label
}

// classify resolves one row's route, preferring persisted tracks over a probe.
func (s *GenerationCandidateService) classify(ctx context.Context, row candidateRow) (RoutePrediction, bool) {
	if s.predictor == nil {
		return "", false
	}

	if tracks, ok := parsePersistedTracks(row.tracksJSON); ok {
		return s.predictor.FromTracks(tracks), true
	}

	route, err := s.predictor.Probe(ctx, row.filePath)
	if err != nil {
		// Rule 13 case 2: logged and skipped for THIS item. Quoting a price for
		// a file we could not read would be worse than omitting it.
		s.logger.Warn("route prediction failed — item omitted from the estimate",
			"media_id", row.id, "media_type", row.mediaType, "file", row.filePath, "error", err)
		return "", false
	}
	return route, true
}

// parsePersistedTracks reads the scan-time `subtitle_tracks` JSON.
//
// ok is false for absent/blank/unparseable/empty JSON — all of which mean "we
// have no scan-time answer", which is a PROBE, not a conclusion. Treating an
// empty array as "no tracks ⇒ ASR" would quote paid speech recognition for
// every movie whose enrichment never ran.
func parsePersistedTracks(raw string) ([]SubtitleTrack, bool) {
	if raw == "" {
		return nil, false
	}
	var tracks []SubtitleTrack
	if err := json.Unmarshal([]byte(raw), &tracks); err != nil {
		return nil, false
	}
	if len(tracks) == 0 {
		return nil, false
	}
	return tracks, true
}
