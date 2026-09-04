package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/vido/api/internal/ai"
	"github.com/vido/api/internal/fsprobe"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/sse"
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

// CandidateModelCatalog is the selectable-model source the quote prices
// against (sub-6-8a AC #3). Narrow on purpose (Rule 11);
// *ModelCatalogService satisfies it. nil degrades to "the ai package default
// only" — a deployment whose keys cannot be read still gets one honest
// number rather than none.
type CandidateModelCatalog interface {
	Available(ctx context.Context) []ai.ModelInfo
	DefaultModel(ctx context.Context) string
}

// CandidateSeriesTitleResolver supplies the series title the F15 group headers
// render (sub-5-3 AC #1). Narrow on purpose (Rule 11);
// *repository.SeriesRepository satisfies it via FindByID; main.go injects it.
// nil degrades to empty titles — grouping still works on series_id.
type CandidateSeriesTitleResolver interface {
	FindByID(ctx context.Context, id string) (*models.Series, error)
}

// ─── Cost model ────────────────────────────────────────────────────────────

// unknownRuntimeMinutes is what an item with no measurable duration is priced
// at. The UI states this assumption verbatim ("片長未知，以 45 分鐘估算") — an
// estimate the user cannot see the assumption behind is not an estimate.
const unknownRuntimeMinutes = 45.0

// translationUSDPerMinuteByModel prices the LLM half, per model, per minute of
// RUNTIME (not of processing).
//
// Cost cannot be computed exactly before extraction — it scales with cue
// count, which is unknown until the track is parsed — so these are averages.
// They are MEASURED, not derived: eval-1 translated 12h20m of this library
// (9 titles, 10,304 cues) twice and billed $2.229 on Haiku and $5.951 on
// Sonnet, i.e. $0.18 and $0.48 per hour of runtime.
//
// ⚠️ This REPLACES a single 0.0004 constant that the M1 pilot calibrated from
// one feature-length item at ~$0.03. Against eval-1's nine-title measurement
// that figure under-quoted by roughly 7×: it would have promised $0.04 for a
// 90-minute film that actually bills $0.27 on Haiku, or $0.72 on Sonnet. An
// estimate the invoice contradicts by that margin is worse than no estimate,
// and the whole point of the consent screen is that the number on it is the
// number you pay. The old constant's own comment invited exactly this
// recalibration once real usage data existed.
//
// tech-money-decimal-arithmetic: decimal STRINGS, not float64 literals. These
// rates are multiplied by runtime minutes for up to ~1,200 titles and the
// results summed; starting from a value that is already ~1e-19 off the figure
// eval-1 actually measured is a needless way to make the quote and the invoice
// disagree.
var translationUSDPerMinuteByModel = map[string]decimal.Decimal{
	"claude-haiku-4-5": decimal.RequireFromString("0.00301"),
	"claude-sonnet-5":  decimal.RequireFromString("0.00804"),
}

// translationCalibrationModel is the anchor an UNMEASURED model is priced
// from: its measured rate scaled by that model's real price ratio. Sonnet,
// deliberately — an unmeasured model is more likely to be a premium one, and
// a quote that surprises upward is the failure mode this screen exists to
// prevent.
//
// Note the scaling is PROPORTIONAL, so a genuinely cheaper model quotes
// cheaper — that is correct, and it is the whole point of offering one. The
// floor applies only to a model with NO price row at all (see
// translationRatePerMinute), where the alternative would be scaling by a
// fallback price that is a fiction.
const translationCalibrationModel = "claude-sonnet-5"

// translationRatePerMinute is the per-runtime-minute LLM cost of a model.
// Measured models use their measured rate; anything else is the anchor scaled
// by the blended (input+output) price ratio — the two prices move together
// across every row of the pricing table, so a single blended ratio is as good
// as a token-split one and needs no assumption about the input/output mix.
func translationRatePerMinute(model string) decimal.Decimal {
	if rate, ok := translationUSDPerMinuteByModel[model]; ok {
		return rate
	}
	anchorRate := translationUSDPerMinuteByModel[translationCalibrationModel]
	if !ai.HasPricing(model) {
		// No real price row: PricingFor would hand back the cheapest-tier
		// fallback, and scaling by it would quote an unknown model BELOW the
		// anchor. Quote the anchor instead — the estimate may be generous, but
		// it can never surprise the user upward.
		return anchorRate
	}
	anchor := blendedPer1M(ai.PricingFor(translationCalibrationModel))
	priced := blendedPer1M(ai.PricingFor(model))
	if !anchor.IsPositive() || !priced.IsPositive() {
		return anchorRate
	}
	// The only division in the estimator. `Div` rounds at
	// decimal.DivisionPrecision (16 significant digits) — eleven orders of
	// magnitude below the cent this figure is eventually rounded to.
	return anchorRate.Mul(priced).Div(anchor)
}

func blendedPer1M(p ai.ModelPricing) decimal.Decimal { return p.InputPer1M.Add(p.OutputPer1M) }

// Processing time as a fraction of runtime, measured in the same eval-1 run
// (two workers in parallel): Haiku finished 12h20m of video in 1h24m, Sonnet
// in 2h03m. An unmeasured model is quoted at the slower anchor, for the same
// reason its price is.
var translationTimeShareByModel = map[string]float64{
	"claude-haiku-4-5": 0.11,
	"claude-sonnet-5":  0.17,
}

func translationTimeShare(model string) float64 {
	if share, ok := translationTimeShareByModel[model]; ok {
		return share
	}
	return translationTimeShareByModel[translationCalibrationModel]
}

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

	// Writable / Blocker (sub-6-1 AC #4) — additive on the sub-4-1 AC #7
	// [@contract-v1] envelope (existing keys unchanged — no bump, the
	// default_budget_usd precedent). Writable is false when a real write probe
	// of the media file's directory failed; Blocker carries the reason. The FE
	// must not pre-select such a row and must keep it out of the total — the
	// pipeline would refuse it before spending anyway (ProcessItem pre-flight),
	// but the user should see that BEFORE consenting, not after.
	Writable bool `json:"writable"`
	// Blocker is the Rule 7 CODE (SUBTITLE_TARGET_NOT_WRITABLE) and BlockerDir
	// the folder's base name — the zh-TW sentence is composed by the client,
	// and an absolute NAS path never leaves the server (CR M6).
	Blocker    string `json:"blocker,omitempty"`
	BlockerDir string `json:"blocker_dir,omitempty"`

	// Series identity (sub-5-3 AC #1) — episodes only; movies leave all four
	// at their zero value. Additive on the sub-4-1 AC #7 [@contract-v1]
	// envelope (existing keys unchanged — no bump, the default_budget_usd
	// precedent). The FE groups by series_id != "", NEVER by season_number:
	// S00 specials are a legal season ZERO, which is why season_number is
	// deliberately NOT omitempty while the two strings are.
	SeriesID     string `json:"series_id,omitempty"`
	SeriesTitle  string `json:"series_title,omitempty"`
	SeasonNumber int    `json:"season_number"`
	// EpisodeNumber gives the FE a deterministic within-season display order —
	// the candidate list's global sort is (title, id), which shuffles episodes
	// whose titles differ. Parsing the SxxEyy substring back out of the title
	// would be the fragile alternative.
	EpisodeNumber int `json:"episode_number"`
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
	// UnwritableCount (sub-6-1 AC #4, additive) — candidates listed with
	// writable=false. Their estimate is NOT in EstimatedTotalUSD.
	UnwritableCount int `json:"unwritable_count"`
}

// ModelEstimate is what one model would cost for this sweep (sub-6-8a AC #3).
// PerCandidate lets the client re-price every row when the user switches
// model without re-running the sweep or re-implementing the cost model.
type ModelEstimate struct {
	TotalUSD float64 `json:"total_usd"`
	// PerCandidate is keyed by media_id and covers the WRITABLE candidates —
	// the same set TotalUSD sums, so a client can add the visible rows and
	// land on the footer figure.
	PerCandidate map[string]float64 `json:"per_candidate,omitempty"`
}

// GenerationCandidateResult is the whole preview payload.
type GenerationCandidateResult struct {
	Candidates []GenerationCandidate      `json:"candidates"`
	Summary    GenerationCandidateSummary `json:"summary"`

	// EstimatesByModel prices this sweep under every model the deployment can
	// actually run (sub-6-8a AC #3), keyed by model id. Additive on the
	// sub-4-1 AC #7 [@contract-v1] envelope — existing keys unchanged, so no
	// bump (the default_budget_usd precedent).
	//
	// It lives on the RESULT rather than beside it on AnalysisSnapshot
	// (which is where the story sketched it) because it IS the quote: a
	// cancelled or failed sweep clears `result`, and these numbers must
	// vanish with it. Kept at snapshot level they would be one more piece of
	// state to invalidate by hand, and a stale price is the one thing this
	// screen must never show.
	EstimatesByModel map[string]ModelEstimate `json:"estimates_by_model,omitempty"`
	// EstimatedMinutesByModel is the wall-clock cost of the same choice —
	// eval-1 measured Sonnet taking half again as long as Haiku, which is a
	// real part of "which model do I want" on a NAS that is also serving media.
	EstimatedMinutesByModel map[string]float64 `json:"estimated_minutes_by_model,omitempty"`
}

// Analysis lifecycle states (story sub-4-1 AC #8).
const (
	// AnalysisIdle — never run in this process.
	AnalysisIdle = "idle"
	// AnalysisRunning — the probe sweep is in flight.
	AnalysisRunning = "analyzing"
	// AnalysisReady — a result is available.
	AnalysisReady = "ready"
	// AnalysisCancelled — the user stopped it; any partial result is discarded.
	AnalysisCancelled = "cancelled"
	// AnalysisFailed — enumeration failed; there is nothing trustworthy to show.
	AnalysisFailed = "error"
)

// ErrAnalysisRunning is returned when a second sweep is requested while one is
// already in flight.
var ErrAnalysisRunning = errors.New("generation candidate analysis already running")

// AnalysisSnapshot is the whole observable state of the preview: where the
// sweep is, and the result if there is one.
type AnalysisSnapshot struct {
	Status   string `json:"status"`
	Analyzed int    `json:"analyzed"`
	Total    int    `json:"total"`
	// Result is non-nil only in the ready state.
	Result *GenerationCandidateResult `json:"result,omitempty"`
	// AnalyzedAt lets the client say how stale the numbers are — the library
	// changes underneath them.
	AnalyzedAt *time.Time `json:"analyzed_at,omitempty"`
	Error      string     `json:"error,omitempty"`
	// DefaultBudgetUSD is the operator-configured AI_RUN_BUDGET_USD value
	// (sub-5-1 AC #5) — the F15 budget-input PREFILL source, so an operator's
	// env change follows through to the consent screen. Additive on the
	// sub-4-1 AC #7 [@contract-v1] envelope (no bump — existing keys
	// unchanged); WYSIWYG consent semantics untouched: the value SENT is
	// always the value on screen.
	DefaultBudgetUSD float64 `json:"default_budget_usd"`
}

// GenerationCandidateService answers "what would generating subtitles cost?"
// for the whole library, WITHOUT spending anything: it probes (cheap, local)
// but never extracts and never transcribes.
//
// Story sub-4-1. It exists because scanning no longer enqueues generation —
// the user picks from this list instead.
//
// The sweep runs as a single-flight background job (the generation-batch
// precedent) because it is real work: one ffprobe per file that carries no
// scan-time track data, and a TV library is mostly such files. A synchronous
// request would hold an HTTP connection open for minutes and tell the user
// nothing while it did.
type GenerationCandidateService struct {
	movies    CandidateMovieFinder
	episodes  CandidateEpisodeFinder
	series    CandidateSeriesTitleResolver
	predictor RoutePredictor
	models    CandidateModelCatalog

	// probeWritable is the sub-6-1 target-directory probe (fsprobe.ProbeWritable
	// by default). Per-analysis results are memoised by directory, so a
	// 24-episode season costs one probe, not 24.
	probeWritable func(ctx context.Context, dir string) error
	sseHub        *sse.Hub
	logger        *slog.Logger
	now           func() time.Time

	// selfHostedASR mirrors "ASR_BASE_URL points somewhere other than the paid
	// API". Wired from config; see ai.EstimatedASRPerMinuteUSD.
	selfHostedASR bool

	// defaultBudgetUSD is the configured AI_RUN_BUDGET_USD, exposed on every
	// snapshot (sub-5-1 AC #5). Pure config pass-through — never mutated.
	defaultBudgetUSD float64

	// routeCache remembers probe verdicts across sweeps (sub-5-4). nil is a
	// supported state: the sweep then behaves exactly as it did before, probing
	// every un-persisted row.
	routeCache RouteCache
	// fileIdentity supplies the size+mtime the cache key is built from. Always
	// osFileIdentity in production; tests replace it so invalidation can be
	// asserted without a real file.
	fileIdentity fileIdentityFunc

	mu         sync.Mutex
	status     string
	analyzed   int
	total      int
	result     *GenerationCandidateResult
	analyzedAt *time.Time
	lastErr    string
	cancel     context.CancelFunc
	// job is a generation token (CR sub-4-1 H1). Cancel-then-restart is a
	// normal user gesture, and without the token the OLD goroutine's
	// wind-down could run AFTER the new sweep started — marking the fresh
	// run cancelled, nil-ing its cancel func (making it unstoppable), and
	// letting stale progress callbacks fight the live ones. Every deferred
	// write is guarded by "is my token still current?".
	job uint64
}

func NewGenerationCandidateService(
	movies CandidateMovieFinder,
	episodes CandidateEpisodeFinder,
	series CandidateSeriesTitleResolver,
	predictor RoutePredictor,
	selfHostedASR bool,
	defaultBudgetUSD float64,
	logger *slog.Logger,
) *GenerationCandidateService {
	if logger == nil {
		logger = slog.Default()
	}
	return &GenerationCandidateService{
		movies:           movies,
		episodes:         episodes,
		series:           series,
		predictor:        predictor,
		logger:           logger.With("service", "generation_candidates"),
		selfHostedASR:    selfHostedASR,
		defaultBudgetUSD: defaultBudgetUSD,
		now:              time.Now,
		fileIdentity:     osFileIdentity,
		probeWritable:    defaultWritableProbe,
		status:           AnalysisIdle,
	}
}

// writableProbeTimeout bounds one directory probe during a consent sweep.
const writableProbeTimeout = 3 * time.Second

// defaultWritableProbe is what NewGenerationCandidateService installs; the
// package tests replace it with a permissive probe in TestMain because their
// fixture paths do not exist on disk.
var defaultWritableProbe = fsprobe.ProbeWritableContext

// SetSSEHub wires live progress. Nil-safe: without a hub the sweep still runs
// and the snapshot still updates, callers just have to poll for it.
func (s *GenerationCandidateService) SetSSEHub(hub *sse.Hub) { s.sseHub = hub }

// SetModelCatalog wires the per-model quote source (sub-6-8a). A setter rather
// than a constructor parameter for the same reason SetSSEHub is one: the
// catalog is optional, and every existing caller keeps compiling.
func (s *GenerationCandidateService) SetModelCatalog(c CandidateModelCatalog) { s.models = c }

// quoteModels answers "which models should this sweep price?" — the ones the
// deployment can actually run, with the effective default first. An
// unconfigured or unwired catalog yields just the ai default, so the existing
// single-number contract never regresses to nothing.
func (s *GenerationCandidateService) quoteModels(ctx context.Context) (defaultModel string, models []string) {
	defaultModel = ai.DefaultClaudeModel
	if s.models == nil {
		return defaultModel, []string{defaultModel}
	}
	if id := s.models.DefaultModel(ctx); id != "" {
		defaultModel = id
	}
	seen := map[string]struct{}{defaultModel: {}}
	models = []string{defaultModel}
	for _, m := range s.models.Available(ctx) {
		if _, dup := seen[m.ID]; dup {
			continue
		}
		seen[m.ID] = struct{}{}
		models = append(models, m.ID)
	}
	return defaultModel, models
}

// SetRouteCache wires the sub-5-4 route cache. A setter rather than a
// constructor parameter for the same reason SetSSEHub is one: the cache is
// optional infrastructure, and without it every existing caller keeps the
// behaviour it already has.
func (s *GenerationCandidateService) SetRouteCache(cache RouteCache) { s.routeCache = cache }

// StartAnalysis kicks off the sweep and returns immediately.
//
// The job ctx is detached from the request (the generation-batch precedent):
// a preview that dies because the user navigated away would have to re-probe
// the whole library on the next visit.
func (s *GenerationCandidateService) StartAnalysis() error {
	s.mu.Lock()
	if s.status == AnalysisRunning {
		s.mu.Unlock()
		return ErrAnalysisRunning
	}
	jobCtx, cancel := context.WithCancel(context.Background())
	s.job++
	job := s.job
	s.status = AnalysisRunning
	s.analyzed, s.total = 0, 0
	s.result, s.analyzedAt = nil, nil
	s.lastErr = ""
	s.cancel = cancel
	s.mu.Unlock()

	s.broadcast()
	go s.runAnalysis(jobCtx, job)
	return nil
}

// CancelAnalysis stops an in-flight sweep. No-op when nothing is running.
//
// The state flips to cancelled SYNCHRONOUSLY (CR sub-4-1 H1): the old
// goroutine may sit inside an ffprobe for up to its 10s timeout, and making
// the user stare at "analyzing" for that long after pressing 取消 reads as a
// broken button. Bumping the job token turns the still-running goroutine into
// a zombie whose every write is dropped — it exits quietly on its own.
func (s *GenerationCandidateService) CancelAnalysis() {
	s.mu.Lock()
	cancel := s.cancel
	if cancel == nil {
		s.mu.Unlock()
		return
	}
	s.cancel = nil
	s.job++ // supersede the running goroutine — its wind-down becomes a no-op
	s.status = AnalysisCancelled
	// A partial classification is not a quote.
	s.result = nil
	s.analyzedAt = nil
	s.mu.Unlock()

	cancel()
	s.broadcast()
}

// Snapshot returns the current state — safe to call at any time.
func (s *GenerationCandidateService) Snapshot() AnalysisSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	return AnalysisSnapshot{
		Status:           s.status,
		Analyzed:         s.analyzed,
		Total:            s.total,
		Result:           s.result,
		AnalyzedAt:       s.analyzedAt,
		Error:            s.lastErr,
		DefaultBudgetUSD: s.defaultBudgetUSD,
	}
}

// runAnalysis is the background half of StartAnalysis. job is its generation
// token — every write back into shared state is dropped once a newer
// StartAnalysis has superseded this run (CR sub-4-1 H1).
func (s *GenerationCandidateService) runAnalysis(ctx context.Context, job uint64) {
	// Progress is throttled by TIME, not by item count: a library of 12 and a
	// library of 12,000 both produce a readable stream, and the hub's bounded
	// broadcast channel never becomes the thing that drops the final event.
	var lastEmit time.Time
	result, err := s.Analyze(ctx, func(done, total int) {
		s.mu.Lock()
		if s.job != job {
			s.mu.Unlock()
			return // a newer sweep owns the counters now
		}
		s.analyzed, s.total = done, total
		s.mu.Unlock()

		now := s.now()
		if done == total || now.Sub(lastEmit) >= progressEmitInterval {
			lastEmit = now
			s.broadcast()
		}
	})

	s.mu.Lock()
	if s.job != job {
		// Superseded: a newer StartAnalysis reset the state after our cancel.
		// Touching anything here would mislabel the LIVE run — the exact
		// cancel-then-restart clobber this token exists to prevent.
		s.mu.Unlock()
		return
	}
	s.cancel = nil
	switch {
	case errors.Is(err, context.Canceled):
		// A partial classification is not a quote — discard it rather than let
		// the UI price a fraction of the library as if it were all of it.
		s.status = AnalysisCancelled
		s.result = nil
	case err != nil:
		s.status = AnalysisFailed
		s.lastErr = err.Error()
		s.result = nil
	default:
		at := s.now()
		s.status = AnalysisReady
		s.result = result
		s.analyzedAt = &at
	}
	s.mu.Unlock()

	s.broadcast()
}

// progressEmitInterval throttles the SSE counter.
const progressEmitInterval = 250 * time.Millisecond

func (s *GenerationCandidateService) broadcast() {
	if s.sseHub == nil {
		return
	}
	snap := s.Snapshot()
	s.sseHub.Broadcast(sse.Event{
		ID:   uuid.New().String(),
		Type: sse.EventGenerationCandidatesProgress,
		Data: map[string]interface{}{
			"status":   snap.Status,
			"analyzed": snap.Analyzed,
			"total":    snap.Total,
			"error":    snap.Error,
		},
	})
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

	asrRate := ai.EstimatedASRPerMinute(s.selfHostedASR)
	// sub-6-8a AC #3: the row-level EstimatedUSD stays the DEFAULT model's
	// price (so a client that never learned about model choice is unchanged),
	// and every selectable model gets its own total beside it.
	defaultModel, quoteModels := s.quoteModels(ctx)
	result := &GenerationCandidateResult{
		Candidates:              make([]GenerationCandidate, 0, len(rows)),
		Summary:                 GenerationCandidateSummary{SelfHostedASR: s.selfHostedASR},
		EstimatesByModel:        make(map[string]ModelEstimate, len(quoteModels)),
		EstimatedMinutesByModel: make(map[string]float64, len(quoteModels)),
	}
	// Running totals stay DECIMAL for the whole sweep and narrow to the wire's
	// float64 exactly once, at the end. A float64 accumulator over ~1,200
	// addends is the classic place where a screen's parts stop adding up to
	// its own total.
	summaryTotal := decimal.Zero
	modelTotals := make(map[string]decimal.Decimal, len(quoteModels))
	for _, m := range quoteModels {
		result.EstimatesByModel[m] = ModelEstimate{PerCandidate: map[string]float64{}}
	}

	// sub-5-4: one batched read for the whole sweep, decided BEFORE the loop.
	plans := s.planRouteCache(ctx, rows)
	cached := s.readRouteCache(ctx, plans)
	var cacheHits, probes int

	// sub-6-1: one probe per distinct directory per sweep, each bounded by
	// writableProbeTimeout so a hung mount cannot stall the whole preview
	// (CR M7). The value is the blocker CODE; the FE composes the sentence.
	writableCache := make(map[string]string)
	writableOf := func(dir string) (writable bool, code, base string) {
		if s.probeWritable == nil {
			return true, "", ""
		}
		code, seen := writableCache[dir]
		if !seen {
			probeCtx, cancel := context.WithTimeout(ctx, writableProbeTimeout)
			err := s.probeWritable(probeCtx, dir)
			cancel()
			if err != nil {
				code = "SUBTITLE_TARGET_NOT_WRITABLE"
				s.logger.Warn("consent list: target directory is not writable",
					"dir", dir, "error", err)
			}
			writableCache[dir] = code
		}
		if code == "" {
			return true, "", ""
		}
		return false, code, filepath.Base(dir)
	}
	for i, row := range rows {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		plan := plans[i]
		var route RoutePrediction
		var ok bool
		if plan.key != "" {
			if hit, found := cached[plan.key]; found {
				route, ok = hit, true
				cacheHits++
			}
		}
		if !ok {
			route, ok = s.classify(ctx, row)
			if plan.probeBound {
				probes++
			}
			if ok {
				// AC #3 red line: only a trustworthy verdict is stored. A probe
				// failure returns ok == false and must stay OUT of the cache —
				// freezing one transient I/O error would drop that file from the
				// candidate list for the whole TTL, silently.
				s.storeRoute(ctx, plan.key, route)
			}
		}
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
		// CR L6: computed ONCE — the summary total below reuses this instead of
		// re-deriving the same decimal for every writable candidate.
		defaultUSD := estimateUSD(route, minutes, asrRate, defaultModel)
		c := GenerationCandidate{
			MediaID:        row.id,
			MediaType:      row.mediaType,
			Title:          row.title,
			Route:          route,
			RuntimeMinutes: minutes,
			RuntimeKnown:   known,
			EstimatedUSD:   defaultUSD.InexactFloat64(),
			SeriesID:       row.seriesID,
			SeriesTitle:    row.seriesTitle,
			SeasonNumber:   row.seasonNumber,
			EpisodeNumber:  row.episodeNumber,
		}
		// sub-6-1 AC #4: the same write probe the pipeline runs, done here so the
		// consent list shows the blocker before a cent is committed. Memoised
		// per directory for this sweep (Rule 14: bounded by distinct dirs).
		c.Writable, c.Blocker, c.BlockerDir = writableOf(filepath.Dir(row.filePath))
		result.Candidates = append(result.Candidates, c)
		if c.Writable {
			summaryTotal = summaryTotal.Add(defaultUSD)
			// An unwritable row is excluded here for the same reason it is
			// excluded from the headline total (sub-6-1): the pipeline would
			// refuse it before spending, so quoting it would inflate every
			// model's number by work that cannot happen.
			for _, m := range quoteModels {
				est := result.EstimatesByModel[m]
				usd := estimateUSD(route, minutes, asrRate, m)
				modelTotals[m] = modelTotals[m].Add(usd)
				est.PerCandidate[c.MediaID] = usd.InexactFloat64()
				result.EstimatesByModel[m] = est
				result.EstimatedMinutesByModel[m] += estimateMinutes(route, minutes, m)
			}
		} else {
			result.Summary.UnwritableCount++
		}
	}

	// Every addend was already rounded to whole cents by estimateUSD, so these
	// sums are exact and need no second rounding — the rows add up to the
	// footer by construction, not by tolerance.
	result.Summary.EstimatedTotalUSD = summaryTotal.InexactFloat64()
	for m, est := range result.EstimatesByModel {
		est.TotalUSD = modelTotals[m].InexactFloat64()
		result.EstimatesByModel[m] = est
		result.EstimatedMinutesByModel[m] = math.Round(result.EstimatedMinutesByModel[m])
	}

	s.logger.Info("generation candidate analysis complete",
		"candidates", len(result.Candidates),
		"extract", result.Summary.ExtractCount,
		"asr", result.Summary.ASRCount,
		"skipped", result.Summary.SkippedCount,
		"unwritable", result.Summary.UnwritableCount,
		"estimated_total_usd", result.Summary.EstimatedTotalUSD,
		"quote_model", defaultModel,
		"quoted_models", len(quoteModels),
		"self_hosted_asr", s.selfHostedASR,
		// sub-5-4 AC #4: without these two numbers, "is the incremental path
		// working?" can only be answered with a stopwatch — and they are the
		// only evidence a future routeVersion or TTL change has to reason from.
		"route_cache_hits", cacheHits,
		"route_probes", probes,
	)
	return result, nil
}

// ─── Route cache (sub-5-4) ─────────────────────────────────────────────────

// routeCachePlan is the per-row decision taken once, before the classification
// loop runs.
type routeCachePlan struct {
	// probeBound is true when classify() has no persisted tracks to read and
	// will therefore reach for predictor.Probe. Those rows are the only ones
	// the cache is worth anything for — and the only ones route_probes counts.
	probeBound bool
	// key is "" when this row does not participate in the cache: it is served
	// by persisted tracks, no cache is wired, or its file could not be stat'ed.
	key string
}

// planRouteCache decides, per row, whether the route cache applies.
//
// It runs before the loop so the whole batch can be read in ONE query: reading
// per row would reintroduce the N+1 the segment cache was corrected for
// (sub-1-5b CR M3), which on a 1,200-episode library means 1,200 round trips
// before the first list can render.
func (s *GenerationCandidateService) planRouteCache(ctx context.Context, rows []candidateRow) []routeCachePlan {
	plans := make([]routeCachePlan, len(rows))
	for i, row := range rows {
		// Honour cancellation mid-plan (CR sub-5-4 L1): os.Stat takes no ctx, so
		// without this check a cancel pressed during the stat pass would wait
		// out the whole library. Rows left unplanned are harmless — the Analyze
		// loop re-checks ctx before touching its first row.
		if ctx.Err() != nil {
			return plans
		}
		// The persisted-tracks fast path touches no disk at all, so it outranks
		// the cache and those rows never reach it (AC #2). Re-parsing here is a
		// JSON unmarshal of a handful of bytes — cheaper than threading the
		// parsed result through classify's signature.
		//
		// PARITY CONTRACT (CR sub-5-4 L3): "does this row probe?" is answered
		// HERE and inside classify, by the same parsePersistedTracks call. If
		// classify ever gains another probe-free source (e.g. episode tech-info
		// parity — `backlog-episode-tech-info-parity`), this predicate must
		// learn it too, or probeBound (and the route_probes log field) silently
		// drifts. classify carries the mirror of this note.
		if _, ok := parsePersistedTracks(row.tracksJSON); ok {
			continue
		}
		plans[i].probeBound = true

		if s.routeCache == nil || s.fileIdentity == nil {
			continue
		}
		size, mtime, err := s.fileIdentity(row.filePath)
		if err != nil {
			// Rule 13 case 3: a file we cannot stat is one the probe is about to
			// fail on anyway. Degrade to the uncached path and let classify
			// report the real error — building a key from a zero-valued stat
			// would make every unreadable file share ONE cache entry.
			s.logger.Debug("route cache skipped — file identity unavailable",
				"media_id", row.id, "file", row.filePath, "error", err)
			continue
		}
		plans[i].key = routeCacheKey(routeVersion, row.id, size, mtime)
	}
	return plans
}

// readRouteCache fetches every planned key in one round trip.
//
// A read failure is logged and swallowed (Rule 13 case 3): the sweep simply
// probes everything, which is exactly what it did before this cache existed.
// Failing the analysis over a cache miss-to-be would turn a transient SQLite
// hiccup into "no price for your library".
func (s *GenerationCandidateService) readRouteCache(ctx context.Context, plans []routeCachePlan) map[string]RoutePrediction {
	if s.routeCache == nil {
		return nil
	}
	keys := make([]string, 0, len(plans))
	for _, p := range plans {
		if p.key != "" {
			keys = append(keys, p.key)
		}
	}
	if len(keys) == 0 {
		return nil
	}

	values, err := s.routeCache.GetMany(ctx, keys)
	if err != nil {
		s.logger.Warn("route cache read failed — every row falls back to a probe",
			"keys", len(keys), "error", err)
		return nil
	}

	out := make(map[string]RoutePrediction, len(values))
	for key, value := range values {
		route := RoutePrediction(value)
		if !isKnownRoute(route) {
			// Not a verdict this build understands; probing is always safe.
			s.logger.Debug("route cache entry ignored — unrecognised verdict",
				"key", key, "value", value)
			continue
		}
		out[key] = route
	}
	return out
}

// storeRoute writes one trustworthy verdict back.
//
// A write failure is logged at Debug and swallowed (Rule 13 case 3, AC #3): the
// classification already succeeded, and the only consequence of a lost write is
// that the next sweep probes this file again — strictly better than failing the
// item over it.
func (s *GenerationCandidateService) storeRoute(ctx context.Context, key string, route RoutePrediction) {
	if s.routeCache == nil || key == "" {
		return
	}
	if err := s.routeCache.Set(ctx, key, string(route), routeCacheTTL); err != nil {
		s.logger.Debug("route cache write failed — the next analysis probes this file again",
			"key", key, "error", err)
	}
}

// estimateUSD prices one item.
//
// The ASR leg pays for audio minutes AND the translation that follows it; the
// extract leg pays only for translation. Note an extract item is therefore
// rarely exactly $0 — labelling it "free" in the UI is a rounding decision the
// client makes, not a claim this function makes.
func estimateUSD(route RoutePrediction, minutes float64, asrRate decimal.Decimal, model string) decimal.Decimal {
	mins := decimal.NewFromFloat(minutes)
	switch route {
	case RouteASR:
		// Only the translation half moves with the model — speech recognition
		// is billed per audio minute by a different provider entirely.
		return roundUSD(mins.Mul(asrRate).Add(mins.Mul(translationRatePerMinute(model))))
	case RouteExtract:
		return roundUSD(mins.Mul(translationRatePerMinute(model)))
	default:
		return decimal.Zero
	}
}

// estimateMinutes is how long a route is expected to TAKE on a model, in
// wall-clock minutes. A skipped item takes none; an ASR item pays the
// transcription pass on top of translation, which eval-1 did not separate —
// so the ASR leg is quoted at the same share, which is the honest floor
// rather than a number nobody measured.
func estimateMinutes(route RoutePrediction, minutes float64, model string) float64 {
	switch route {
	case RouteASR, RouteExtract:
		return minutes * translationTimeShare(model)
	default:
		return 0
	}
}

// roundUSD rounds a decimal amount to whole cents, half away from zero — the
// rule a person doing it by hand uses, and the one every invoice states. This
// is a PRESENTATION decision applied to per-item prices; running totals are
// accumulated from the already-rounded items, never re-rounded.
func roundUSD(v decimal.Decimal) decimal.Decimal { return v.Round(2) }

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
	// Series identity (sub-5-3) — zero-valued for movies. seriesTitle is
	// resolved once per series per sweep (memo in enumerate) and degrades to
	// "" on a lookup failure (Rule 13 case 3 — one missing series row must not
	// deny the whole library its quote; grouping still works on the id).
	seriesID      string
	seriesTitle   string
	seasonNumber  int
	episodeNumber int
}

// runtimeMinutes returns the metadata runtime and whether it was known.
func (r candidateRow) runtimeMinutes() (float64, bool) {
	if r.runtime.Valid && r.runtime.Int64 > 0 {
		return float64(r.runtime.Int64), true
	}
	return unknownRuntimeMinutes, false
}

// resolveSeriesTitle answers the group-header label for one series. Rule 13
// case 3 — deliberately degraded after logging: a nil resolver or a failed
// lookup yields "" (the FE renders 未知影集 and still groups on the id), never
// a failed sweep.
func (s *GenerationCandidateService) resolveSeriesTitle(ctx context.Context, seriesID string) string {
	if s.series == nil || seriesID == "" {
		return ""
	}
	series, err := s.series.FindByID(ctx, seriesID)
	if err != nil || series == nil {
		s.logger.Debug("series title lookup failed — group header degrades to the id",
			"series_id", seriesID, "error", err)
		return ""
	}
	return series.Title
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
		// One title lookup per DISTINCT series per sweep (sub-5-3 AC #1) — a
		// 20-season show must cost one read, not one per episode. The memo also
		// caches failures ("" entry) so a broken series row is logged once.
		seriesTitles := make(map[string]string)
		for _, e := range episodes {
			if !e.FilePath.Valid || e.FilePath.String == "" {
				continue
			}
			title, seen := seriesTitles[e.SeriesID]
			if !seen {
				title = s.resolveSeriesTitle(ctx, e.SeriesID)
				seriesTitles[e.SeriesID] = title
			}
			rows = append(rows, candidateRow{
				id:            e.ID,
				mediaType:     models.SubtitleRunMediaEpisode,
				title:         episodeTitle(e),
				filePath:      e.FilePath.String,
				runtime:       e.Runtime,
				seriesID:      e.SeriesID,
				seriesTitle:   title,
				seasonNumber:  e.SeasonNumber,
				episodeNumber: e.EpisodeNumber,
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
//
// PARITY CONTRACT (CR sub-5-4 L3): planRouteCache predicts this function's
// probe-vs-not decision by running the same parsePersistedTracks check. If a
// new probe-free source is added here, update planRouteCache in the same
// change — see the mirror note there.
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
