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
	// TranscriptionPlanSidecar — no usable embedded track, but the English
	// subtitle beside the file matches its timing, so the click translates
	// that file instead of paying for speech recognition
	// (disc-2026-10-generate-keeps-user-en-srt-b2).
	TranscriptionPlanSidecar = "sidecar"
)

// RouteSidecarPrediction is the estimate's `route` for a matching English
// sidecar. It is not a RoutePrediction: the probe alone cannot know it.
const RouteSidecarPrediction = "sidecar"

// Sidecar check states reported in TranscriptionEstimate.SidecarCheck
// (-b2, field names confirmed by Alexyu 2026-10-20).
const (
	SidecarStatusNone     = "none"     // no English subtitle beside the file
	SidecarStatusChecking = "checking" // the timing check is running in the background
	SidecarStatusAligned  = "aligned"  // matches: plan "sidecar"
	SidecarStatusOffset   = "offset"   // the whole file is early or late by OffsetSeconds
	SidecarStatusMismatch = "mismatch" // no single shift fits (frame rate, cut, another film)
	SidecarStatusUnusable = "unusable" // unreadable, too short or not English
	SidecarStatusUnknown  = "unknown"  // the check could not run
)

// SidecarAlignment is what the estimate knows about the English subtitle
// beside the file. FileName is the file the status is about.
type SidecarAlignment struct {
	Status        string  `json:"status"`
	OffsetSeconds float64 `json:"offset_seconds"`
	FileName      string  `json:"file_name,omitempty"`
}

// SidecarAlignmentSource answers from stored checks and starts a check in the
// background when none is stored (sidecarcheck.Finder in production). It
// never makes the caller wait for ffmpeg.
type SidecarAlignmentSource interface {
	Peek(ctx context.Context, mediaPath string) SidecarAlignment
}

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
	// ASRProvider names the engine the speech-recognition half is priced at
	// (asr-scribe-b, additive): "openai", "elevenlabs", "self_hosted", or ""
	// when none is configured.
	ASRProvider string `json:"asr_provider,omitempty"`
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
	// ResumedASRMinutes and CachedTranslationShare are the paid-for work an
	// earlier, unfinished run left behind, already taken off EstimatedUSD
	// (disc-2026-09-generation-resume-c, a [@contract-v2] widening: both
	// omitted when there is none; RuntimeMinutes still means the whole film).
	// ResumedASRMinutes: speech-recognition minutes stored (plan=full).
	ResumedASRMinutes float64 `json:"resumed_asr_minutes,omitempty"`
	// CachedTranslationShare: 0–1 of the English SRT's cues whose translation
	// is cached (plan=translate_only).
	CachedTranslationShare float64 `json:"cached_translation_share,omitempty"`
	// SidecarCheck is the English-sidecar timing check (-b2). Absent when no
	// sidecar source is wired or an embedded track already decides the route.
	SidecarCheck *SidecarAlignment `json:"sidecar_check,omitempty"`
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
	// The resume credit (disc-2026-09-generation-resume-c): work an earlier
	// run paid for that this one will be served for free.
	resumedASRMinutes(ctx context.Context, mediaID, filePath string, runtimeMinutes float64) float64
	cachedTranslationShare(ctx context.Context, mediaType, mediaID string) float64
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
	// asrRate, when set, replaces selfHostedASR as the rate source
	// (asr-scribe-b): the holder's per-call answer.
	asrRate func(ctx context.Context) (decimal.Decimal, string)
	// prober measures a file that has no stored duration. nil = no live probe;
	// the ladder continues to TMDb and then the stated assumption.
	prober           RouteDurationPredictor
	episodeDurations CandidateEpisodeDurationWriter
	// routes classifies the file for the pipeline-mode quote. nil = legacy
	// mode: no route is reported and the speech-recognition quote stands.
	routes RoutePredictor
	// sidecars reports the English-sidecar check; nil = not consulted.
	sidecars SidecarAlignmentSource
	logger   *slog.Logger
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

// SetASRRateSource installs the per-call speech-recognition rate reader
// (asr-scribe-b) — ASRProviderHolder.ASRRate in production. nil keeps the
// boot-time self-hosted bool.
func (s *TranscriptionEstimateService) SetASRRateSource(fn func(ctx context.Context) (decimal.Decimal, string)) {
	s.asrRate = fn
}

func (s *TranscriptionEstimateService) asrRateFor(ctx context.Context) (decimal.Decimal, string) {
	if s.asrRate != nil {
		return s.asrRate(ctx)
	}
	if s.selfHostedASR {
		return decimal.Zero, ASRProviderSelfHosted
	}
	return ai.EstimatedASRPerMinute(false), ASRProviderOpenAI
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

// SetSidecarSource turns on the English-sidecar quote (-b2). main.go wires it
// with the same finder the pipeline's router uses, so the quote and the run
// read the same stored checks.
func (s *TranscriptionEstimateService) SetSidecarSource(src SidecarAlignmentSource) {
	s.sidecars = src
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

	// No usable embedded track: the router tries the English subtitle beside
	// the file before speech recognition, so the quote asks the same finder.
	// Only a stored "aligned" changes the price; anything else — including a
	// check still running — quotes speech recognition, the dearer lane, so the
	// user never pays more than they were shown.
	var sidecar *SidecarAlignment
	if s.sidecars != nil && route != RouteExtract && target.FilePath != "" && s.routes != nil {
		sa := s.sidecars.Peek(ctx, target.FilePath)
		sidecar = &sa
	}

	plan := TranscriptionPlanFull
	switch {
	case sidecar != nil && sidecar.Status == SidecarStatusAligned:
		plan = TranscriptionPlanSidecar
	case route == RouteExtract:
		// The embedded track is translated (or delivered); the `untranslated`
		// row's English SRT is not what this lane reads, so no resume.
		plan = TranscriptionPlanExtract
	case s.plan.canResumeTranslateOnly(ctx, target.MediaType, target.MediaID):
		plan = TranscriptionPlanTranslateOnly
	}

	var credit resumeCredit
	switch {
	case plan == TranscriptionPlanFull && s.plan.IsAvailable():
		credit.asrMinutes = s.plan.resumedASRMinutes(ctx, target.MediaID, target.FilePath, minutes)
	case plan == TranscriptionPlanTranslateOnly && translate:
		// The cue keys carry the model id: look them up under the model this
		// quote prices, which is the one the run will bill.
		credit.translatedShare = s.plan.cachedTranslationShare(ai.WithModelID(ctx, model), target.MediaType, target.MediaID)
	}

	asrRate, asrProvider := s.asrRateFor(ctx)
	usd := s.priceUSD(plan, translate, minutes, model, credit, asrRate)

	routeOut := string(route)
	if plan == TranscriptionPlanSidecar {
		routeOut = RouteSidecarPrediction
	}

	return TranscriptionEstimate{
		MediaID:                target.MediaID,
		MediaType:              target.MediaType,
		Plan:                   plan,
		Route:                  routeOut,
		ASRAvailable:           s.plan.IsAvailable(),
		SelfHostedASR:          asrProvider == ASRProviderSelfHosted,
		ASRProvider:            asrProvider,
		TranslationConfigured:  translate,
		ModelID:                model,
		RuntimeMinutes:         minutes,
		RuntimeKnown:           known,
		RuntimeSource:          source,
		EstimatedUSD:           usd.InexactFloat64(),
		ResumedASRMinutes:      credit.asrMinutes,
		CachedTranslationShare: credit.translatedShare,
		SidecarCheck:           sidecar,
	}
}

// resumeCredit is what an earlier, unfinished run already paid for. The zero
// value is "nothing", which prices the whole film — the pre-resume quote.
type resumeCredit struct {
	asrMinutes      float64 // 0..minutes
	translatedShare float64 // 0..1
}

// priceUSD composes the quote from the candidate list's own pricing functions —
// no rate or formula is restated here (story dsr-6a AC #3).
//
// credit takes off what an earlier run already paid for: stored ASR minutes
// come off the speech-recognition half only (the translation of a film whose
// transcription is unfinished has not started), and the cached share comes off
// a translate-only resume.
func (s *TranscriptionEstimateService) priceUSD(plan string, translate bool, minutes float64, model string, credit resumeCredit, asrRate decimal.Decimal) decimal.Decimal {
	mins := decimal.NewFromFloat(minutes)
	asrMins := decimal.NewFromFloat(max(minutes-credit.asrMinutes, 0))
	switch {
	case plan == TranscriptionPlanSidecar && translate:
		// The sidecar runs the pipeline's translate lane — the same lane and
		// price as an embedded English track.
		return estimateUSD(RouteExtract, minutes, asrRate, model)
	case plan == TranscriptionPlanSidecar:
		// No key: the translate verdict fails before any paid call.
		return decimal.Zero
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
		if credit.asrMinutes <= 0 {
			return estimateUSD(RouteASR, minutes, asrRate, model)
		}
		// estimateUSD's ASR formula with the stored minutes taken off its
		// first term.
		return roundUSD(asrMins.Mul(asrRate).Add(mins.Mul(asrLegTranslationRatePerMinute(model))))
	case plan == TranscriptionPlanFull:
		// English-only run: the translate leg is skipped, so only audio minutes bill.
		return roundUSD(asrMins.Mul(asrRate))
	case translate:
		// Translate-only resume still translates through the ASR leg's
		// translateSRT, so it pays that leg's rate — NOT RouteExtract's, which
		// is a cheaper, cached code path this run never takes. Cues already
		// in the segment cache are not sent again.
		share := decimal.NewFromFloat(min(max(credit.translatedShare, 0), 1))
		return roundUSD(mins.Mul(asrLegTranslationRatePerMinute(model)).Mul(decimal.NewFromInt(1).Sub(share)))
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
