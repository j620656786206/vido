package services

import (
	"context"
	"log/slog"

	"github.com/shopspring/decimal"

	"github.com/vido/api/internal/ai"
	"github.com/vido/api/internal/models"
)

// Plans a single-item run can take (story dsr-6a AC #2 [@contract-v2]).
const (
	// TranscriptionPlanFull — extract audio, speech recognition, then translate.
	TranscriptionPlanFull = "full"
	// TranscriptionPlanTranslateOnly — the row is `untranslated` and its English
	// SRT is still on disk, so the run skips extract + ASR (sub-2-2a AC #3).
	TranscriptionPlanTranslateOnly = "translate_only"
	// TranscriptionPlanExtract — the file carries a usable embedded text track,
	// so the click routes through the subtitle pipeline without speech
	// recognition: a Chinese track is delivered (free), an English one is
	// translated (disc-2026-10-single-generate-ignores-embedded-english-a).
	// Only reported in pipeline mode.
	TranscriptionPlanExtract = "extract"
)

// TranscriptionEstimate is the price the 管理字幕 dialog shows on its paid
// buttons (story dsr-6a AC #2 [@contract-v2]).
//
// It prices what POST /movies/:id/transcribe?translate=true and
// POST /episodes/:id/transcribe will ACTUALLY do. In pipeline mode (a route
// predictor is wired) the click routes like the batch, so the quote names the
// predicted lane in Route and prices it: extract → the candidate list's
// extract price (LLM translation; a Chinese track turns out free, so this is
// the honest UPPER bound), asr / skip → speech recognition + translation, or
// translate-only on resume. In legacy mode Route is absent and the quote is
// the speech-recognition one — which is what that mode's click still does.
//
// v1→v2: `plan` gained the value `extract`, `route` was added. ⚖️ 2026-08-06
// ruling A ("the single-item button never reads embedded tracks") was reversed
// by Alexyu on 2026-10-06 after the See S01E02 eval.
type TranscriptionEstimate struct {
	MediaID   string `json:"media_id"`
	MediaType string `json:"media_type"`
	// Plan is TranscriptionPlanFull, TranscriptionPlanTranslateOnly or
	// TranscriptionPlanExtract.
	Plan string `json:"plan"`
	// Route is the probe-only prediction of the lane the click will take —
	// "extract" | "asr" | "skip" (RoutePrediction) — absent in legacy mode and
	// when the file could not be classified. Not a RouteKind: deliver /
	// convert / translate cannot be told apart before extraction.
	Route string `json:"route,omitempty"`
	// ASRAvailable mirrors the trigger's availability gate. false with
	// Plan=full means the click would 503 — the dialog disables the button
	// before it gets that far (J9-D ⑥).
	ASRAvailable  bool `json:"asr_available"`
	SelfHostedASR bool `json:"self_hosted_asr"`
	// TranslationConfigured is the run's own translate check; false means the
	// run stops at the English SRT and no translation is billed.
	TranslationConfigured bool `json:"translation_configured"`
	// ModelID is the model the translation leg will bill — the holder's
	// effective model, not the catalog's pre-selection. "" when no translation
	// leg runs.
	ModelID string `json:"model_id"`
	// Runtime fields reuse the candidate list's vocabulary
	// (confirmed against [@contract-v1] (Story sub-4-1 AC #7)).
	RuntimeMinutes float64 `json:"runtime_minutes"`
	RuntimeKnown   bool    `json:"runtime_known"`
	RuntimeSource  string  `json:"runtime_source"`
	EstimatedUSD   float64 `json:"estimated_usd"`
}

// TranscriptionEstimateTarget is the media row the handler already resolved and
// validated (id, file path on disk). The estimator never looks it up again.
type TranscriptionEstimateTarget struct {
	MediaID         string
	MediaType       string // models.SubtitleRunMediaMovie | models.SubtitleRunMediaEpisode
	FilePath        string
	DurationSeconds models.NullInt64
	Runtime         models.NullInt64
	// SubtitleTracksJSON is the movie row's persisted scan-time probe
	// (`subtitle_tracks`); empty for episodes and unenriched movies. When it
	// parses, the route is read from it without touching the file.
	SubtitleTracksJSON string
}

// transcriptionPlanSource is the part of TranscriptionService the estimate must
// share with the run, so the two can never disagree about what a click does.
// *TranscriptionService is the only production implementation; the unexported
// methods keep it package-private (Rule 11 — narrow, and not a public seam).
type transcriptionPlanSource interface {
	IsAvailable() bool
	translationEnabled() bool
	canResumeTranslateOnly(ctx context.Context, mediaType, mediaID string) bool
}

// TranscriptionEstimateService prices one single-item generation run.
//
// It spends nothing and starts nothing: no single-flight slot, no SSE, and the
// only write is an episode's freshly probed duration (the same write-back the
// candidate sweep does, sub-6-10a AC #1).
type TranscriptionEstimateService struct {
	plan           transcriptionPlanSource
	effectiveModel func() string
	selfHostedASR  bool
	// prober measures a file that has no stored duration. nil = no live probe;
	// the ladder continues to TMDb and then the stated assumption.
	prober           RouteDurationPredictor
	episodeDurations CandidateEpisodeDurationWriter
	// routes classifies the file for the pipeline-mode quote. nil = legacy
	// mode: no route is reported and the speech-recognition quote stands.
	routes RoutePredictor
	logger *slog.Logger
}

// NewTranscriptionEstimateService wires the estimator. plan is the
// TranscriptionService whose run is being priced; effectiveModel is the Claude
// holder's EffectiveModel; selfHostedASR comes from the same
// ai.IsSelfHostedASRBaseURL answer the candidate sweep uses.
func NewTranscriptionEstimateService(plan transcriptionPlanSource, effectiveModel func() string, selfHostedASR bool, logger *slog.Logger) *TranscriptionEstimateService {
	if logger == nil {
		logger = slog.Default()
	}
	return &TranscriptionEstimateService{
		plan:           plan,
		effectiveModel: effectiveModel,
		selfHostedASR:  selfHostedASR,
		logger:         logger.With("service", "transcription_estimate"),
	}
}

// SetDurationProber wires the live ffprobe used when no duration is stored.
// Production passes the same adapter the candidate sweep uses, so both share
// FFprobeService's concurrency limit.
func (s *TranscriptionEstimateService) SetDurationProber(p RouteDurationPredictor) {
	s.prober = p
}

// SetEpisodeDurationWriter wires the episode duration write-back.
func (s *TranscriptionEstimateService) SetEpisodeDurationWriter(w CandidateEpisodeDurationWriter) {
	s.episodeDurations = w
}

// SetRoutePredictor turns on the pipeline-mode quote: the click routes
// through the subtitle pipeline, so the estimate predicts the lane and prices
// it. main.go wires it only when the pipeline is enabled; production passes
// the same adapter SetDurationProber received, so a file is probed once.
func (s *TranscriptionEstimateService) SetRoutePredictor(p RoutePredictor) {
	s.routes = p
}

// Estimate prices the run a click on 生成字幕 would start for target.
//
// The dialog's movie trigger always sends ?translate=true and the episode
// route always translates, so the estimate assumes translation is requested;
// whether it RUNS is the run's own translationEnabled check.
func (s *TranscriptionEstimateService) Estimate(ctx context.Context, target TranscriptionEstimateTarget) TranscriptionEstimate {
	translate := s.plan.translationEnabled()

	model := ""
	if translate && s.effectiveModel != nil {
		model = s.effectiveModel()
	}

	// Route first from what is already known (the persisted track list), then
	// from the ladder's own probe when it runs, and only then with a probe of
	// our own — one ffprobe at most.
	route := s.routeFromTracks(target)
	minutes, known, source, probed, ladderProbed := s.runtimeMinutes(ctx, target)
	if route == "" {
		route = probed
	}
	// Our own probe only when the ladder did not touch the file at all: a
	// ladder probe that FAILED must not be repeated — the dialog would wait
	// out two timeouts on a sleeping disk (CR L2).
	if route == "" && !ladderProbed && s.routes != nil && target.FilePath != "" {
		route = s.probeRoute(ctx, target)
	}

	plan := TranscriptionPlanFull
	switch {
	case route == RouteExtract:
		// The embedded track is translated (or delivered); the `untranslated`
		// row's English SRT is not what this lane reads, so no resume.
		plan = TranscriptionPlanExtract
	case s.plan.canResumeTranslateOnly(ctx, target.MediaType, target.MediaID):
		plan = TranscriptionPlanTranslateOnly
	}

	usd := s.priceUSD(plan, translate, minutes, model)

	return TranscriptionEstimate{
		MediaID:               target.MediaID,
		MediaType:             target.MediaType,
		Plan:                  plan,
		Route:                 string(route),
		ASRAvailable:          s.plan.IsAvailable(),
		SelfHostedASR:         s.selfHostedASR,
		TranslationConfigured: translate,
		ModelID:               model,
		RuntimeMinutes:        minutes,
		RuntimeKnown:          known,
		RuntimeSource:         source,
		EstimatedUSD:          usd.InexactFloat64(),
	}
}

// priceUSD composes the quote from the candidate list's own pricing functions —
// no rate or formula is restated here (story dsr-6a AC #3).
func (s *TranscriptionEstimateService) priceUSD(plan string, translate bool, minutes float64, model string) decimal.Decimal {
	asrRate := ai.EstimatedASRPerMinute(s.selfHostedASR)
	switch {
	case plan == TranscriptionPlanExtract && translate:
		// The batch consent list's extract price — LLM translation of the
		// track, no speech recognition. A Chinese track ends up free; quoting
		// the translation is the honest upper bound ("寧可往上給").
		return estimateUSD(RouteExtract, minutes, asrRate, model)
	case plan == TranscriptionPlanExtract:
		// Deliver / convert are free, and a translate verdict fails before any
		// paid call without a key — nothing is billed either way.
		return decimal.Zero
	case plan == TranscriptionPlanFull && translate:
		return estimateUSD(RouteASR, minutes, asrRate, model)
	case plan == TranscriptionPlanFull:
		// English-only run: the translate leg is skipped, so only audio minutes bill.
		return roundUSD(decimal.NewFromFloat(minutes).Mul(asrRate))
	case translate:
		// Translate-only resume still translates through the ASR leg's
		// translateSRT, so it pays that leg's rate — NOT RouteExtract's, which
		// is a cheaper, cached code path this run never takes.
		return roundUSD(decimal.NewFromFloat(minutes).Mul(asrLegTranslationRatePerMinute(model)))
	default:
		// Resume without a translation key does nothing billable.
		return decimal.Zero
	}
}

// runtimeMinutes walks the sweep's ladder (stored measurement → TMDb → 45 min)
// with one extra rung before TMDb: a live probe when nothing is stored, so an
// episode — which has no TMDb runtime writer — is not quoted at the assumption.
//
// The probe also classifies the file; that route is returned alongside (""
// when no probe ran or it failed) so Estimate never probes twice, and
// ladderProbed says whether the file was touched at all.
func (s *TranscriptionEstimateService) runtimeMinutes(ctx context.Context, target TranscriptionEstimateTarget) (minutes float64, known bool, source string, probedRoute RoutePrediction, ladderProbed bool) {
	row := candidateRow{
		id:              target.MediaID,
		mediaType:       target.MediaType,
		filePath:        target.FilePath,
		runtime:         target.Runtime,
		durationSeconds: target.DurationSeconds,
	}
	stored := row.durationSeconds.Valid && row.durationSeconds.Int64 > 0
	if !stored && s.prober != nil && target.FilePath != "" {
		ladderProbed = true
		route, seconds, err := s.prober.ProbeWithDuration(ctx, target.FilePath)
		if err == nil && s.routes != nil {
			probedRoute = route
		}
		switch {
		case err != nil && ctx.Err() != nil:
			// The caller went away (the dialog closed mid-probe) — routine, not a
			// failure worth a Warn on every close.
			s.logger.Debug("duration probe cancelled with its request",
				"media_id", target.MediaID, "error", err)
		case err != nil:
			// Rule 13 case 3: the quote degrades to the next rung, it never fails.
			s.logger.Warn("duration probe failed — estimating from the next source",
				"media_id", target.MediaID, "media_type", target.MediaType, "error", err)
		case seconds > 0:
			measured := int64(seconds)
			row.durationSeconds = models.NewNullInt64(measured)
			s.rememberEpisodeDuration(ctx, target, measured)
		}
	}
	minutes, known, source = row.runtimeMinutes()
	return minutes, known, source, probedRoute, ladderProbed
}

// routeFromTracks classifies the persisted track list, when there is one and
// routing is on. "" = not decidable from the row alone.
func (s *TranscriptionEstimateService) routeFromTracks(target TranscriptionEstimateTarget) RoutePrediction {
	if s.routes == nil {
		return ""
	}
	tracks, ok := parsePersistedTracks(target.SubtitleTracksJSON)
	if !ok {
		return ""
	}
	return s.routes.FromTracks(tracks)
}

// probeRoute is the one extra ffprobe a routed quote may need: a stored
// duration kept the ladder from probing, and the row had no track list. A
// failure leaves the route unknown — the quote then stays the speech-
// recognition one, which is the safe (higher) direction.
func (s *TranscriptionEstimateService) probeRoute(ctx context.Context, target TranscriptionEstimateTarget) RoutePrediction {
	route, err := s.routes.Probe(ctx, target.FilePath)
	if err != nil {
		if ctx.Err() == nil {
			s.logger.Warn("route probe failed — quoting the speech-recognition lane",
				"media_id", target.MediaID, "media_type", target.MediaType, "error", err)
		}
		return ""
	}
	return route
}

// rememberEpisodeDuration mirrors GenerationCandidateService's rule: episodes
// only (a movie's duration belongs to enrichment), and a failed write is
// dropped — the quote already used the measurement.
func (s *TranscriptionEstimateService) rememberEpisodeDuration(ctx context.Context, target TranscriptionEstimateTarget, seconds int64) {
	if s.episodeDurations == nil || target.MediaType != models.SubtitleRunMediaEpisode || seconds <= 0 {
		return
	}
	if err := s.episodeDurations.UpdateDurationSeconds(ctx, target.MediaID, seconds); err != nil {
		s.logger.Debug("episode duration write-back failed — the quote still used the measurement",
			"media_id", target.MediaID, "seconds", seconds, "error", err)
	}
}
