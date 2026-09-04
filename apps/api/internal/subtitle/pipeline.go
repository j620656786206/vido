package subtitle

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/vido/api/internal/ai"
	"github.com/vido/api/internal/ai/prompts"
	"github.com/vido/api/internal/fsprobe"
	"github.com/vido/api/internal/models"
)

const (
	// maxQualityRetries caps SEMANTIC resends per cue (FR16). Transport-level
	// retries are the ai client's job (retryTransient, D8) — do not add another
	// transport loop here.
	maxQualityRetries = 2

	// stubbornCeilingDenominator expresses the stubborn-cue ceiling as 1/N of
	// the track's cues: 20 → 5%.
	stubbornCeilingDenominator = 20
)

// ChunkTranslator is the narrow port the pipeline needs from the services-side
// TranslationService: ONE chunk in, per-index translations plus token usage out.
//
// Chunking, the quality gate and every semantic retry live on THIS side of the
// boundary (architecture P3): retry granularity is the cue, transport
// granularity is the chunk. If the service batched internally the pipeline
// could not resend only the failed cues without re-sending the clean ones —
// which is the quality gate's whole point.
// The harvested-terms return (sub-5-5) is the chunk's optional trailer yield —
// the proper-noun renderings the model reported for this chunk; nil when none.
// This port is unstamped, so the widening syncs implementations and fakes in
// the same change with no Rule 20 bump.
type ChunkTranslator interface {
	TranslateChunk(ctx context.Context, sys []ai.SystemBlock, contextBlocks, blocks []prompts.SubtitleTranslatorBlock) (map[int]string, map[string]string, ai.CompletionUsage, error)
}

// VariantConverter is the narrow port over *Converter. Injected once and reused
// for the whole track (Rule 14 — never constructed per call).
type VariantConverter interface {
	ConvertS2TWP(content []byte) ([]byte, error)
}

// TranslateContext carries the FR26 show metadata injected into the translation
// prompt as context. Every field is optional; a zero value yields a prompt that
// is byte-identical to the no-metadata path.
//
// [@contract-v1] — consumed by sub-1-5b (delivery + provenance + cache policy)
// and sub-1-6. Changing the signature, the stubborn-cue policy, or the usage
// semantics = Rule 20 bump + downstream stale-mark.
type TranslateContext struct {
	Title, OriginalTitle string
	Year                 int
	Genres               []string
	Overview             string
	Cast                 []string // capped at 10 by the builder
	Countries            []string
	Glossary             []prompts.GlossaryEntry // M1: always empty — field exists NOW (D4 versioning)
}

// TranslateResult is the translate stage's output.
//
// [@contract-v1] — see TranslateContext. HarvestedTerms is ADDITIVE on v1
// (sub-5-5 AC #3, the `default_budget_usd` precedent — no bump): the
// proper-noun renderings harvested from the chunks' trailers, merged across
// chunks with the FIRST occurrence winning (later hindsight does not overturn
// an established rendering). nil when nothing was harvested.
type TranslateResult struct {
	Blocks         []SubtitleBlock    // translated + gated + OpenCC'd; Index/Start/End byte-equal source
	Usage          ai.CompletionUsage // aggregated across all chunks + retries
	StubbornCues   int                // cues delivered with English fallback (see policy)
	HarvestedTerms map[string]string  // trailer yield, first-wins across chunks (sub-5-5)
}

// MediaRef identifies one media row for the item flow. MediaType uses the
// INTERNAL vocabulary — movie | series | episode (sub-1-2 AC #1) — never the
// TMDB movie|tv pair that `requests.media_type` carries.
//
// [@contract-v1] — consumed by sub-1-6 (flag seam, endpoint, scanner enqueue).
type MediaRef struct{ ID, MediaType string }

// ProcessItemOptions carries the operator levers for one item.
//
// NAMING: the story's AC #1 calls this `ProcessOptions`, but that identifier is
// already taken in this package by the SEARCH engine's per-item options
// (engine.go:125, constructed at batch.go:238 — the very call site sub-1-6's
// flag seam wraps). Renaming the incumbent would rewrite a shipped Epic-8
// surface for cosmetic reasons, so the NEW type carries the disambiguating
// name instead. sub-1-6 acks `[@contract-v1] sub-1-5b AC #1` — the shape is
// unchanged, only the identifier.
//
// [@contract-v1] — see MediaRef. FreeOnly is ADDITIVE on v1 (9R-10b AC #3,
// the `HarvestedTerms` / `default_budget_usd` precedent — no bump): its zero
// value is the shipped behaviour, so every existing caller keeps its contract
// byte-for-byte.
type ProcessItemOptions struct {
	// Force is the M1 pilot's re-run switch: it bypasses the P5 pre-flight AND
	// segment-cache READS (writes still happen, so a forced run refreshes the
	// cache rather than orphaning it). It rides ai.Governor like every other
	// call, so repeated re-runs stay inside the budget ceiling.
	Force bool

	// FreeOnly restricts the run to the work that costs nothing: an embedded
	// Traditional-Chinese track delivered as-is, or a Simplified one converted
	// by local OpenCC. The two PAID routes — `translate` (the LLM) and
	// `no_text_source` (speech recognition) — are not attempted; the item is
	// recorded as deferred and left exactly as it was found, so it reappears
	// in the consent list with an estimate instead of quietly vanishing.
	//
	// This is the automation lane opened by the 2026-08-19 ruling
	// 「9R-10b 花錢須同意」, and it sits UNDERNEATH the 2026-08-07 ruling
	// (scanning updates metadata and nothing else) rather than replacing it:
	// nothing on this path can produce a charge. `internal/cost_consent_test.go`
	// still guards the library-wide paid sweep, which remains uncalled.
	FreeOnly bool
}

// ProcessOutcome is what one item flow produced.
//
// Run is nil in exactly one case — the P5 pre-flight early-exited before any
// row was written — and Kind is then the zero RouteKind while SubtitlePath
// points at the acceptable sidecar that already existed. RouteKind deliberately
// gains no "already done" member: it is sub-1-4's `[@contract-v1]` enum and
// extending it would be a Rule 20 bump on a shipped contract.
//
// [@contract-v1] — see MediaRef.
type ProcessOutcome struct {
	Run          *models.SubtitleRun // the recorded run (nil on pre-flight early-exit)
	Kind         RouteKind           // what happened
	SubtitlePath string              // non-empty when a sidecar is in place
}

// RunStore is the narrow port over repository.SubtitleRunRepositoryInterface —
// only the three methods the item flow actually uses, so tests fake three
// methods instead of five.
type RunStore interface {
	Create(ctx context.Context, run *models.SubtitleRun) error
	Update(ctx context.Context, run *models.SubtitleRun) error
	FindCompletedRun(ctx context.Context, mediaID, mediaType string, v models.RunVersion) (*models.SubtitleRun, error)
}

// MediaItem is the pipeline's read-only view of one media row. ProcessItem
// receives a MediaRef, so something must resolve it to a file path and the FR26
// metadata — that resolution is a repository concern and therefore a port.
type MediaItem struct {
	// FilePath is the absolute path of the media file.
	FilePath string
	// TMDbID is recorded on the run for pilot attribution; nil when unmatched.
	TMDbID *int64
	// ShowKey groups items that share a prompt prefix for D10: the SERIES id
	// for a series or episode row, empty for a movie. It cannot be derived from
	// MediaRef — an episode's ref carries the EPISODE id, and keying the gate
	// on that would serialize nothing while defeating the exact season-batch
	// case D10 exists for.
	ShowKey string
	// Context is the FR26 show metadata injected into the translation prompt
	// and hashed into the run version.
	Context TranslateContext
	// SubtitlePath / SubtitleLanguage are the row's values AS LOADED.
	//
	// CR-249 H1: the FreeOnly brake used to hand setMediaStatus empty strings
	// for these, and the shared writer maps "" to NULL unconditionally
	// (repository/subtitle_generation_status.go:44-45) — so "do not stamp a
	// path" silently became "ERASE the path". For an item bugfix-j had left at
	// `untranslated` that path is the ENGLISH SRT a paid ASR run already
	// produced, and it is the only thing that lets the next consented run
	// translate instead of transcribing again. Blanking it makes the user pay
	// for the same speech recognition twice.
	SubtitlePath     string
	SubtitleLanguage string
	// SubtitleStatus is the row's status AS LOADED, before this run touches it.
	//
	// 9R-10b AC #3: the FreeOnly brake has to put the media row back exactly
	// where it found it. It cannot blanket-write `not_searched`, because that
	// would erase bugfix-j's `untranslated` verdict — an item that already has
	// an English sidecar and only needs translating would start claiming
	// nothing had ever been searched. The zero value ("") means "the store did
	// not report one"; the brake falls back to `not_searched` for that case.
	SubtitleStatus models.SubtitleStatus
}

// MediaStore is the narrow port over the three media tables, dispatched on
// MediaRef.MediaType by the implementation (sub-1-6 wires it to the movie /
// series / episode repositories).
//
// CONTRACT NOTE for the implementer: writes here must touch subtitle_status /
// subtitle_path / subtitle_language ONLY. `subtitle_search_score` and
// `subtitle_last_searched` must stay untouched (AC #6.3) — extract-and-translate
// is not a search, and stamping the search columns would make a generated
// sidecar indistinguishable from a provider hit in the library UI.
type MediaStore interface {
	Load(ctx context.Context, ref MediaRef) (*MediaItem, error)
	SetSubtitleStatus(ctx context.Context, ref MediaRef, status models.SubtitleStatus, subtitlePath, language string) error
}

// SubtitlePlacer is the narrow port over *Placer. D3 makes placer.go the SOLE
// writer of the sidecar; the pipeline never touches the media folder itself.
type SubtitlePlacer interface {
	Place(req PlaceRequest) (*PlaceResult, error)
}

// TrackRouter is the narrow port over *Router (sub-1-4 AC #1).
type TrackRouter interface {
	SelectAndRoute(ctx context.Context, mediaPath, tmpDir string) (RouteDecision, error)
}

// SpeechTranscriber is the narrow ASR port behind the sub-3-1 fallback leg —
// named for what the pipeline needs (turn audio into a placed subtitle), not
// after the concrete service. main.go adapts it over
// services.TranscriptionService.RunTranscription, the SYNCHRONOUS seam 9R-16
// added precisely for callers that own the loop (never StartTranscription —
// that path detaches from the caller ctx and reports via SSE only).
//
// Error contract: Transcribe returns an error satisfying
// errors.Is(err, services.ErrTranscriptionDisabled) when the service cannot
// run at all (no ASR key / no FFmpeg and no translate-only resume), and one
// satisfying errors.Is(err, ai.ErrBudgetExceeded) when the shared run budget
// ceiling was hit mid-run. The leg maps the former to today's `no_text_source`
// degrade and the latter to a pause.
//
// Available is the SWEEP-side probe (worker_pool re-enumeration gate). The
// leg itself deliberately does not call it per item: the service's own entry
// gate is richer (resume-aware — CR sub-2-2a M2) and must stay authoritative.
//
// Transcribe takes the full MediaRef (sub-3-2): the adapter needs the media
// TYPE to route the service's writeback and resume-read to the right table
// (WithMediaType dispatch).
type SpeechTranscriber interface {
	Available() bool
	Transcribe(ctx context.Context, ref MediaRef, filePath, mediaDir string) error
}

// Pipeline owns the generation-side stages that must not live in services:
// the quality gate needs detector.Detect, and Rule 19 forbids services →
// subtitle. The thin LLM call stays on the services side behind ChunkTranslator.
//
// One Pipeline is constructed for the process (sub-1-6) and shared by the
// worker pool, which is why the D10 latch and the progress hook are struct
// fields rather than per-call parameters.
type Pipeline struct {
	translator ChunkTranslator
	converter  VariantConverter
	logger     *slog.Logger

	// Item-flow ports. TranslateTrack needs none of them, which is why they are
	// options rather than constructor parameters: sub-1-5a's translate-only
	// construction stays valid and ProcessItem reports precisely which port is
	// missing instead of nil-panicking mid-run.
	runs   RunStore
	cache  SegmentCache
	router TrackRouter
	placer SubtitlePlacer
	media  MediaStore

	// asr is the OPTIONAL sub-3-1 fallback port — nil is a legal, supported
	// state (translate-only construction, ASR-less deployment) and means
	// RouteNoTextSource records `no_text_source` exactly as it did pre-M2.
	// Deliberately NOT validated by requireItemPorts (see the note there).
	asr SpeechTranscriber

	// glossary is the OPTIONAL sub-5-5 feed/harvest port — nil is a legal,
	// supported state (translate-only construction, or a deployment wired
	// before this story) and means exactly today's behavior: empty glossary
	// fed, nothing harvested. Deliberately NOT in requireItemPorts.
	glossary GlossaryStore

	// modelID is the model that produces the translations — a RunVersion field.
	// sub-6-5: it is read through a source func at RunVersion assembly time
	// rather than snapshotted at boot, because the boot-time env override was
	// EMPTY on every default-model run and every run row recorded "" as the
	// model. WithModelID wraps a constant for tests and legacy wiring.
	modelID func() string

	// probeWritable is the sub-6-1 pre-flight over the media file's directory,
	// run after the run row exists and before any ffprobe/extract/LLM call.
	// Defaults to fsprobe.ProbeWritable; tests inject a failing one.
	probeWritable func(dir string) error

	// runBudgetUSD is the per-item AI cost ceiling (sub-5-1 AC #3;
	// AI_RUN_BUDGET_USD, 0 = unlimited) applied when ProcessItem's ctx carries
	// no Budget already. Wiring-supplied like modelID.
	runBudgetUSD float64

	// progress is the nil-safe stage hook. sub-1-6 connects it to the SSE hub;
	// it stays nil here and in every test that does not assert on it, which is
	// what keeps this story free of SSE (AC #8).
	//
	// It carries the MediaRef because ONE Pipeline serves the whole worker pool
	// — without it a concurrent season batch could not attribute an event to an
	// item, and sub-1-6's subtitle_progress payload needs {media_id, media_type}.
	progress func(ref MediaRef, stage PipelineStage, message string)

	// gate is D10's per-show first-request latch, owned by the Pipeline because
	// serialization is an orchestrator concern — sub-1-6's worker pool stays a
	// generic executor with no show-awareness.
	gate *showGate

	// now is the injected clock (D10's warm window is the only wall-clock read).
	now func() time.Time
}

// PipelineOption injects one item-flow port. Variadic on NewPipeline so
// sub-1-5a's three-argument construction — and its nil-port panic guard — keep
// working byte-identically.
type PipelineOption func(*Pipeline)

// WithRunStore injects the subtitle_runs provenance port.
func WithRunStore(runs RunStore) PipelineOption {
	return func(p *Pipeline) { p.runs = runs }
}

// WithSegmentCache injects the cue-grain translation cache (AD #4 tier 2).
func WithSegmentCache(cache SegmentCache) PipelineOption {
	return func(p *Pipeline) { p.cache = cache }
}

// WithRouter injects the front-half verdict port (sub-1-4).
func WithRouter(router TrackRouter) PipelineOption {
	return func(p *Pipeline) { p.router = router }
}

// WithPlacer injects the sole sidecar writer (D3).
func WithPlacer(placer SubtitlePlacer) PipelineOption {
	return func(p *Pipeline) { p.placer = placer }
}

// WithMediaStore injects the media-row resolver and status writer.
func WithMediaStore(media MediaStore) PipelineOption {
	return func(p *Pipeline) { p.media = media }
}

// WithSpeechTranscriber injects the OPTIONAL ASR fallback port (sub-3-1).
// Unlike the five required item-flow ports, absence is not a wiring bug — it
// is the documented ASR-less degradation (AC #4).
func WithSpeechTranscriber(asr SpeechTranscriber) PipelineOption {
	return func(p *Pipeline) { p.asr = asr }
}

// WithGlossaryStore injects the OPTIONAL per-show glossary feed/harvest port
// (sub-5-5). Nil-safe: unwired = empty glossary fed, nothing harvested —
// byte-identical to the pre-sub-5-5 item flow.
func WithGlossaryStore(store GlossaryStore) PipelineOption {
	return func(p *Pipeline) { p.glossary = store }
}

// WithModelID records which model produces the translations. It is part of the
// cache key and of every run row, so leaving it empty across a model change
// would serve the previous model's translations back unnoticed.
func WithModelID(modelID string) PipelineOption {
	return WithModelSource(func() string { return modelID })
}

// WithModelSource records WHERE to ask for the model id when a RunVersion is
// assembled (sub-6-5 AC #2). Production wires the provider holder's
// EffectiveModel so a model override — or a future per-run choice — reaches
// every run row and cache key without a restart or a stale boot snapshot.
func WithModelSource(source func() string) PipelineOption {
	return func(p *Pipeline) { p.modelID = source }
}

// currentModelID is the RunVersion read of the model source. A pipeline built
// without either option reports the ai package default rather than "" — the
// empty string is exactly the value this story exists to keep out of run rows
// and cache keys, so no construction path may produce it (sub-6-5 CR H2).
func (p *Pipeline) currentModelID() string {
	if p.modelID == nil {
		return ai.DefaultClaudeModel
	}
	if id := p.modelID(); id != "" {
		return id
	}
	return ai.DefaultClaudeModel
}

// WithWritableProbe replaces the sub-6-1 target-directory probe. Production
// keeps the default (fsprobe.ProbeWritable); tests inject a failing one to
// prove the item flow spends nothing after a refusal.
func WithWritableProbe(fn func(dir string) error) PipelineOption {
	return func(p *Pipeline) { p.probeWritable = fn }
}

// WithRunBudgetUSD sets the per-item AI cost ceiling ProcessItem attaches when
// its ctx carries no Budget (sub-5-1 AC #3, the WithModelID wiring precedent).
// Without it the FR12/pool path's LLM translation spend was never metered and
// never capped — "AI 花費上限" only held on the consent batch.
func WithRunBudgetUSD(usd float64) PipelineOption {
	return func(p *Pipeline) { p.runBudgetUSD = usd }
}

// WithProgress installs the stage hook sub-1-6 bridges to the SSE hub.
func WithProgress(fn func(ref MediaRef, stage PipelineStage, message string)) PipelineOption {
	return func(p *Pipeline) { p.progress = fn }
}

// WithClock overrides the pipeline clock. Test-only in practice: the D10 warm
// window is the sole wall-clock read, and a deterministic clock is what makes
// "stale entry re-warms" assertable without sleeping.
func WithClock(now func() time.Time) PipelineOption {
	return func(p *Pipeline) {
		if now != nil {
			p.now = now
		}
	}
}

// NewPipeline wires the pipeline to its two mandatory ports plus any item-flow
// options. The two positional ports are wiring-time invariants, so a nil one
// panics at construction rather than degrading at runtime — a nil converter in
// particular would silently void FR15's deterministic Traditional-script
// guarantee on every delivered cue.
func NewPipeline(translator ChunkTranslator, converter VariantConverter, logger *slog.Logger, opts ...PipelineOption) *Pipeline {
	if translator == nil {
		panic("subtitle.NewPipeline: ChunkTranslator must not be nil")
	}
	if converter == nil {
		panic("subtitle.NewPipeline: VariantConverter must not be nil — FR15's OpenCC guarantee would silently disappear")
	}
	if logger == nil {
		logger = slog.Default()
	}
	p := &Pipeline{
		translator:    translator,
		probeWritable: fsprobe.ProbeWritable,
		converter:     converter,
		logger:        logger.With("component", "subtitle_pipeline"),
		now:           time.Now,
	}
	for _, opt := range opts {
		opt(p)
	}
	// Built after the options so WithClock reaches the gate's warm window.
	p.gate = newShowGate(p.now)
	return p
}

// ─── Per-item scope ────────────────────────────────────────────────────────

// processScope is the per-ITEM state the chunk loop inside TranslateTrack needs
// but must not receive as parameters: TranslateTrack's signature is sub-1-5a's
// `[@contract-v1]` surface, and one Pipeline is shared by sub-1-6's worker pool,
// so a struct field could not attribute an observation to an item either.
//
// It therefore rides the context. Absent scope = TranslateTrack called
// directly (sub-1-5a's own callers and tests), and every observation becomes a
// no-op — the stage keeps working exactly as it shipped.
//
// One item runs on one goroutine, so the fields need no synchronization.
type processScope struct {
	ref MediaRef
	// showKey groups items that share a prompt prefix (D10). Empty = no gate.
	showKey string
	// cacheProbed / cacheEnabled record the AC #4.2 verdict, taken from the
	// FIRST chunk's usage and never overwritten by later chunks.
	cacheProbed  bool
	cacheEnabled bool

	// requestsSent counts the chunk requests this item actually issued.
	// cache_enabled is a NON-NULLABLE bool, so a fully segment-cached item —
	// which sends nothing — records `false` and is indistinguishable in the
	// column from an item whose prompt prefix genuinely failed to cache. Until
	// the column can go nullable (tracked as
	// backlog-subtitle-run-cache-enabled-tristate), this counter is what lets
	// the M1 pilot separate the two populations from the completion log.
	requestsSent int

	// fullTrackCues is the FULL routed track's cue count. The segment cache
	// hands TranslateTrack only the miss subset, so FR16's stubborn ceiling must
	// still be measured against the whole DELIVERED track: cache hits are
	// already-accepted translations and belong in the denominator. Without this,
	// a warm cache silently tightens "5% of the episode" into "5% of whatever is
	// left", and one flaky cue kills a 99%-deliverable episode.
	// Zero = no cache split in play; the subset IS the track.
	fullTrackCues int

	// stubbornIndexes are the cues TranslateTrack delivered with their ENGLISH
	// original (NFR-R1 fail-soft). translateWithCache excludes them from the
	// segment-cache write: caching a fail-soft fallback would freeze the failure
	// for the whole 30-day TTL, and because the key is cue content + show
	// metadata it would serve that English line to every other episode of the
	// show. The fail-soft is meant to be transient.
	stubbornIndexes map[int]struct{}

	// harvestedTerms counts the glossary terms this item's harvest actually
	// INSERTED (deduped conflicts excluded) — the AC #6 completion-log figure.
	harvestedTerms int

	// spentUSDAtStart snapshots the ctx Budget's cumulative spend when THIS
	// item began. A consent batch shares ONE Budget across items (sub-4-2), so
	// the per-run spend stamped at the terminal write (ux3-1-6) must be the
	// delta from here, not the batch running total — otherwise every run row
	// would misattribute its siblings' cost.
	spentUSDAtStart float64
}

type processScopeKey struct{}

func withProcessScope(ctx context.Context, scope *processScope) context.Context {
	return context.WithValue(ctx, processScopeKey{}, scope)
}

func processScopeFrom(ctx context.Context) *processScope {
	scope, _ := ctx.Value(processScopeKey{}).(*processScope)
	return scope
}

// ceilingTotal answers "how many cues does the caller actually deliver?" for
// FR16's stubborn ceiling. It is the full routed track when a segment-cache
// split is in play and `subset` otherwise — sub-1-5a's direct callers carry no
// scope, so for them the subset IS the track and the policy is byte-identical
// to how it shipped.
func ceilingTotal(ctx context.Context, subset int) int {
	if scope := processScopeFrom(ctx); scope != nil && scope.fullTrackCues > subset {
		return scope.fullTrackCues
	}
	return subset
}

// noteStubbornCue records a cue that shipped its English original, so the item
// flow can keep it OUT of the segment cache. No-op without a scope.
func noteStubbornCue(ctx context.Context, index int) {
	scope := processScopeFrom(ctx)
	if scope == nil {
		return
	}
	if scope.stubbornIndexes == nil {
		scope.stubbornIndexes = make(map[int]struct{})
	}
	scope.stubbornIndexes[index] = struct{}{}
}

// countRequest tallies one issued chunk request (retries included), which is
// what disambiguates a `cache_enabled=false` caused by an inert prompt prefix
// from one caused by never sending a request at all.
func countRequest(ctx context.Context) {
	if scope := processScopeFrom(ctx); scope != nil {
		scope.requestsSent++
	}
}

// emitProgress invokes the stage hook when one is wired. Nil-safe by design:
// the hook is sub-1-6's SSE bridge, and every call site here must work with it
// absent (AC #8 keeps SSE entirely out of this story).
func (p *Pipeline) emitProgress(ref MediaRef, stage PipelineStage, message string) {
	if p.progress == nil {
		return
	}
	p.progress(ref, stage, message)
}

// observeChunk is the per-chunk hook (P8's throttle grain): exactly one call per
// chunk, after that chunk's first response returns.
//
// AC #4.2 — `cache_enabled` is recorded by DETECTION, not estimation. The
// Messages API reports cache-creation and cache-read tokens; both reading zero
// on the first chunk means the prefix silently failed to cache (below the
// model's minimum, or a provider that does not cache at all — the Gemini
// degradation path returns zero usage by construction). No tokenizer
// heuristics: the API's own accounting is the truth, which is exactly why
// sub-1-1 surfaced the two fields.
func (p *Pipeline) observeChunk(ctx context.Context, chunk, total int, usage ai.CompletionUsage) {
	scope := processScopeFrom(ctx)
	if scope == nil {
		return
	}

	if !scope.cacheProbed {
		scope.cacheProbed = true
		scope.cacheEnabled = usage.CacheCreationInputTokens > 0 || usage.CacheReadInputTokens > 0
		if !scope.cacheEnabled {
			p.logger.Info("prompt cache is inert for this item — recording cache_enabled=false",
				"media_id", scope.ref.ID,
				"media_type", scope.ref.MediaType,
				"input_tokens", usage.InputTokens,
			)
		}
	}

	// The format is a shared const so the SSE bridge can parse the counters back
	// out (progress_sse.go) without the two literals silently drifting apart.
	p.emitProgress(scope.ref, StageTranslating, fmt.Sprintf(translateChunkProgressFormat, chunk, total))
}

// deliveredLanguage / deliveredFormat are what M1 always writes. They are
// constants rather than parameters on purpose: D3 gives placer.go sole
// ownership of the sidecar, and every route in this pipeline converges on one
// zh-Hant .srt beside the media file.
const (
	deliveredLanguage = "zh-Hant"
	deliveredFormat   = "srt"
)

// ExpectedSidecarPath resolves the sidecar this pipeline would write for
// mediaPath. It runs the SAME normalize + filename builders placer.Place uses,
// so the P5 pre-flight can never check a different path from the one delivery
// writes.
func ExpectedSidecarPath(mediaPath string) string {
	return BuildSubtitleFilename(mediaPath, NormalizeLanguageTag(deliveredLanguage), deliveredFormat)
}

// acceptableSidecar is P5's predicate, spelled out: the file EXISTS, parses via
// ParseSRT without error, and carries at least one cue.
//
// The cue-count clause is the whole point. An existence-only check treats a
// zero-byte or truncated artifact as valid and permanently blocks regeneration
// — the named anti-pattern. A file that fails any clause is not "acceptable",
// so the pipeline proceeds and overwrites it (placer keeps a .bak).
func acceptableSidecar(path string) (bool, string) {
	raw, err := os.ReadFile(path) //nolint:gosec // path is derived from the media file we were handed
	if err != nil {
		if os.IsNotExist(err) {
			return false, "no existing zh-Hant sidecar"
		}
		return false, fmt.Sprintf("existing zh-Hant sidecar is unreadable (%v)", err)
	}

	blocks, err := ParseSRT(string(raw))
	if err != nil {
		return false, fmt.Sprintf("existing zh-Hant sidecar does not parse (%v)", err)
	}
	if len(blocks) == 0 {
		return false, fmt.Sprintf("existing zh-Hant sidecar parsed to 0 cues (%d bytes) — truncated, not acceptable", len(raw))
	}
	return true, fmt.Sprintf("existing zh-Hant sidecar parses with %d cue(s)", len(blocks))
}

// preflightSkip answers "may we exit before spending a single ffprobe or LLM
// call?" — AC #2. The sidecar predicate is the GATE; the completed-run lookup
// only REFINES the reason we log, which is what makes "delete the sidecar and
// re-trigger" an inherently valid re-run even when a completed run row exists.
//
// force bypasses the gate entirely: the operator asked for a re-run, and
// placer.Place's .bak backup is the safety net for the overwrite.
func (p *Pipeline) preflightSkip(ctx context.Context, ref MediaRef, mediaPath string, version models.RunVersion, opts ProcessItemOptions) (bool, string) {
	if opts.Force {
		return false, "force: pre-flight and segment-cache reads bypassed"
	}

	ok, reason := acceptableSidecar(ExpectedSidecarPath(mediaPath))
	if !ok {
		return false, reason
	}

	// One lookup per early-exit: it is a DB round trip, and the log line and the
	// returned reason must describe the same observation.
	resume := p.resumeReason(ctx, ref, version)
	p.logger.Info("subtitle pre-flight early-exit",
		"media_id", ref.ID,
		"media_type", ref.MediaType,
		"reason", reason,
		"resume", resume,
	)
	return true, reason + " — " + resume
}

// resumeReason classifies an early-exit for the operator: a version-matched
// completed run means "we made this file and nothing has changed since", while
// no match means the sidecar came from somewhere else (the search engine, the
// manual flow, a pre-pipeline library, or a run under a bumped prompt/model).
//
// Deliberate Rule 13 case-3 discard: this is a LOG refinement, not the gate. A
// repository hiccup must not turn a satisfied P5 predicate into a full re-run
// — that would be the runaway-spend path the predicate exists to prevent.
func (p *Pipeline) resumeReason(ctx context.Context, ref MediaRef, version models.RunVersion) string {
	if p.runs == nil {
		return "no run store wired; resume state unknown"
	}

	run, err := p.runs.FindCompletedRun(ctx, ref.ID, ref.MediaType, version)
	switch {
	case err != nil:
		p.logger.Warn("subtitle resume lookup failed — early-exit stands on the sidecar predicate alone",
			"media_id", ref.ID, "media_type", ref.MediaType, "error", err)
		return "resume state unknown (lookup failed)"
	case run == nil:
		return "foreign or pre-pipeline sidecar (no version-matched completed run)"
	default:
		return "version-matched resume-skip of run " + run.ID
	}
}

// TranslateTrack translates one routed English track to Traditional Chinese.
//
// [@contract-v1] — see TranslateContext.
func (p *Pipeline) TranslateTrack(ctx context.Context, track *ExtractedTrack, tctx TranslateContext) (*TranslateResult, error) {
	if track == nil || len(track.Blocks) == 0 {
		return nil, fmt.Errorf("%w: translate stage received no cues", ErrSubtitleTranslateFailed)
	}

	source := track.Blocks
	sys := buildSystemBlocks(tctx)

	// Snapshot cue identity BEFORE any work so the FR17 check compares against
	// the track as it was routed — not against whatever state it is in when the
	// stage finishes. SubtitleBlock is all value fields, so a shallow copy is a
	// full one.
	routed := append([]SubtitleBlock(nil), source...)

	// final holds the accepted translation per cue Index. Cue identity is the
	// Index carried from the source track, never the position: SDH filtering
	// leaves gaps in the numbering (P1/P7, sub-1-4 AC #1).
	final := make(map[int]string, len(source))
	var usage ai.CompletionUsage
	var harvested map[string]string

	stubborn := 0

	// Chunk numbering is derived once so the per-chunk observation can report
	// "N of M" without the caller having to recompute the batching rule.
	totalChunks := (len(source) + prompts.SubtitleTranslatorBatchSize - 1) / prompts.SubtitleTranslatorBatchSize

	for start := 0; start < len(source); start += prompts.SubtitleTranslatorBatchSize {
		end := min(start+prompts.SubtitleTranslatorBatchSize, len(source))
		chunk := source[start:end]
		chunkNumber := start/prompts.SubtitleTranslatorBatchSize + 1

		// A whole track is tens of sequential chunks; check cancellation at the
		// boundary so a shutdown stops promptly. Both sentinels are chained so
		// the orchestrator can classify the run AND still tell a shutdown apart
		// from a genuine translate failure with errors.Is(context.Canceled) —
		// the sub-1-4 CR M1/M2 precedent.
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("%w: cancelled at cue %d: %w", ErrSubtitleTranslateFailed, chunk[0].Index, err)
		}

		// Read-only context is fixed for the whole chunk: it looks BACKWARDS at
		// already-translated cues, so a retry inside this chunk cannot change it.
		contextBlocks := p.contextWindow(source, start, final)

		// pending starts as the whole chunk and narrows to just the cues the
		// gate rejected. Quality retries are SEMANTIC only, capped per cue —
		// transport retries already live inside the ai client (D8); a second
		// loop here would multiply them.
		pending := chunk
		var verdict GateVerdict

		for attempt := 0; ; attempt++ {
			// D10: the item's very first LLM request warms the show's prompt
			// prefix, and the entry only becomes readable once that response
			// begins. Everything after it — later chunks, semantic retries —
			// rides the warm prefix and is never serialized.
			release, err := p.enterShowGate(ctx, start == 0 && attempt == 0)
			if err != nil {
				return nil, fmt.Errorf("%w: cancelled waiting for the show's first request: %w",
					ErrSubtitleTranslateFailed, err)
			}

			got, chunkTerms, chunkUsage, err := p.translator.TranslateChunk(ctx, sys, contextBlocks, promptBlocksOf(pending))
			// The gate is released as WARM only when the request actually
			// returned. A failed first request never wrote the provider-side
			// prefix, so marking the show warm would open the whole 50-minute
			// window with nothing cached — every remaining episode would then
			// skip the gate and pay the 2x prefix write premium individually,
			// which is precisely the cost D10 exists to avoid. On failure the
			// next waiter legitimately becomes the leader and re-attempts the
			// warm-up, still serialized.
			release(err == nil)
			countRequest(ctx)
			usage = addUsage(usage, chunkUsage)
			if err != nil {
				return nil, fmt.Errorf("%w: cues %d-%d: %w",
					ErrSubtitleTranslateFailed, pending[0].Index, pending[len(pending)-1].Index, err)
			}

			// Exactly once per chunk, on the response that first came back: a
			// semantic retry re-sends the same prefix, so observing it again
			// would double-count progress and could flip the cache verdict on
			// the second read of an entry the first read created.
			if attempt == 0 {
				p.observeChunk(ctx, chunkNumber, totalChunks, chunkUsage)
			}

			verdict = checkChunk(pending, got, p.logger)
			for _, b := range pending {
				if !verdict.Failed(b.Index) {
					final[b.Index] = got[b.Index]
				}
			}

			// Trailer yield merges FIRST-wins across chunks AND retries (sub-5-5
			// AC #3): the model's later hindsight does not overturn a rendering
			// an earlier chunk already established. Gated on the verdict (CR
			// sub-5-5 M1): an attempt the gate rejected WHOLESALE contributes
			// nothing — its renderings are exactly what the retry exists to
			// replace, and first-wins would otherwise let them squat over the
			// corrected ones. A partially-accepted attempt still harvests: its
			// accepted cues ship, so their renderings are established.
			if len(verdict.FailedIndexes) < len(pending) {
				harvested = mergeChunkTerms(harvested, chunkTerms)
			}

			if verdict.Passed() || attempt == maxQualityRetries {
				break
			}
			pending = subsetByIndex(pending, verdict.FailedIndexes)
		}

		for _, index := range verdict.FailedIndexes {
			stubborn++
			noteStubbornCue(ctx, index)
			p.logger.Warn("subtitle cue kept its English original after quality retries",
				"cue_index", index,
				"reason", verdict.Reasons[index],
				"retries", maxQualityRetries,
				"stream_index", track.StreamIndex,
			)
		}
	}

	// Stubborn-cue policy (NFR-R1 fail-soft, bounded): one flaky cue must not
	// kill a 1000-cue episode, but a broken track must not ship half-English.
	// stubborn*20 > total is the integer form of stubborn/total > 5%.
	//
	// The denominator is the DELIVERED track, which is not always `source`: with
	// a warm segment cache, sub-1-5b hands this stage only the miss subset while
	// the cache hits are already-accepted translations that still ship. Counting
	// only the subset would make the ceiling tighten as the cache warms — a
	// 20-cue episode with one flaky cue passes cold (1/20 = 5%) and fails at 18
	// hits (1/2 = 50%). See ceilingTotal's derivation on processScope.
	delivered := ceilingTotal(ctx, len(source))
	if stubborn*stubbornCeilingDenominator > delivered {
		return nil, fmt.Errorf("%w: %d of %d cues still failed the quality gate after %d retries — over the %d%% ceiling",
			ErrSubtitleTranslateFailed, stubborn, delivered, maxQualityRetries, 100/stubbornCeilingDenominator)
	}

	blocks := p.convertAndStitch(source, final)
	if err := checkTimestampInvariant(routed, blocks); err != nil {
		return nil, err
	}

	return &TranslateResult{Blocks: blocks, Usage: usage, StubbornCues: stubborn, HarvestedTerms: harvested}, nil
}

// mergeChunkTerms folds one chunk's trailer terms into the accumulated map,
// FIRST occurrence winning. Returns acc unchanged (possibly nil) when the
// chunk harvested nothing.
func mergeChunkTerms(acc, chunk map[string]string) map[string]string {
	if len(chunk) == 0 {
		return acc
	}
	if acc == nil {
		acc = make(map[string]string, len(chunk))
	}
	for src, zh := range chunk {
		if _, seen := acc[src]; !seen {
			acc[src] = zh
		}
	}
	return acc
}

// subsetByIndex narrows a chunk to the cues the gate rejected, preserving
// source order so the resent prompt keeps reading top-to-bottom.
func subsetByIndex(blocks []SubtitleBlock, indexes []int) []SubtitleBlock {
	wanted := make(map[int]struct{}, len(indexes))
	for _, index := range indexes {
		wanted[index] = struct{}{}
	}

	out := make([]SubtitleBlock, 0, len(indexes))
	for _, b := range blocks {
		if _, ok := wanted[b.Index]; ok {
			out = append(out, b)
		}
	}
	return out
}

// contextWindow returns the previous SubtitleTranslatorContextWindow cues as
// read-only context, carrying their TRANSLATED text so the model stays
// terminologically consistent across chunk boundaries. A cue with no accepted
// translation yet contributes its source text.
func (p *Pipeline) contextWindow(source []SubtitleBlock, start int, final map[int]string) []prompts.SubtitleTranslatorBlock {
	if start == 0 {
		return nil
	}
	from := max(start-prompts.SubtitleTranslatorContextWindow, 0)

	out := make([]prompts.SubtitleTranslatorBlock, 0, start-from)
	for _, b := range source[from:start] {
		out = append(out, prompts.SubtitleTranslatorBlock{Index: b.Index, Text: textOf(b, final)})
	}
	return out
}

// convertAndStitch runs the OpenCC final pass over every cue and writes the
// result back onto a COPY of the source cue, so Index/Start/End are preserved
// by construction (FR11/FR17). It runs only once every chunk has cleared the
// quality gate — converting earlier would repair a Simplified leak the gate
// exists to catch (P4).
func (p *Pipeline) convertAndStitch(source []SubtitleBlock, final map[int]string) []SubtitleBlock {
	out := make([]SubtitleBlock, len(source))
	copy(out, source)

	failures := 0
	var firstErr error

	for i := range out {
		text := textOf(source[i], final)
		converted, err := p.converter.ConvertS2TWP([]byte(text))
		if err != nil {
			// Deliberate non-fatal discard (Rule 13 case 3): the gate has
			// already guaranteed this cue carries no simplified-only
			// character, so s2twp here is phrase-level polish rather than
			// the correctness barrier. Delivering the gated text beats
			// failing the item (NFR-R1) — the same call the Converter's own
			// contract makes ("unconverted subtitle is better than no
			// subtitle"). Surfaced in one aggregate warning below.
			failures++
			if firstErr == nil {
				firstErr = err
			}
		} else {
			text = string(converted)
		}
		out[i].Text = text
	}

	if failures > 0 {
		p.logger.Warn("OpenCC final pass failed for some cues — delivering the gated text unconverted",
			"failed_cues", failures,
			"total_cues", len(out),
			"first_error", firstErr,
		)
	}
	return out
}

// textOf returns the accepted translation for a cue, falling back to its
// original source text.
func textOf(b SubtitleBlock, final map[int]string) string {
	if text, ok := final[b.Index]; ok {
		return text
	}
	return b.Text
}

// promptBlocksOf projects source cues onto the prompt's numbered-text shape.
// Start/End never cross this boundary — timestamps stay in Go (P2/FR11).
func promptBlocksOf(blocks []SubtitleBlock) []prompts.SubtitleTranslatorBlock {
	out := make([]prompts.SubtitleTranslatorBlock, 0, len(blocks))
	for _, b := range blocks {
		out = append(out, prompts.SubtitleTranslatorBlock{Index: b.Index, Text: b.Text})
	}
	return out
}

// buildSystemBlocks composes the system prompt in stable-first order, the shape
// prompt caching depends on: [0] the invariant translator prompt (most stable),
// [1] the per-show metadata + glossary sections. Block [1] is omitted entirely
// when there is nothing per-show to say, so the no-metadata path is
// byte-identical to Route C's.
//
// CACHE POLICY (AC #4.1, sub-1-5b): the LAST block carries CacheTTL1h and every
// earlier block carries none. Caching is a PREFIX match, so a single breakpoint
// on the last stable block caches everything before it too — an inner
// breakpoint would split the prefix for no gain. 1h rather than 5m because a
// season batch spans tens of minutes and the ephemeral entry would expire
// mid-run (architecture fact 7; the 2× write premium is paid once per show).
//
// Whether the breakpoint actually fires is MEASURED, never assumed: below the
// model's minimum cacheable prefix (4096 tokens on claude-haiku-4-5)
// cache_control is silently inert, so observeChunk reads the API's own usage
// and records the verdict in subtitle_runs.cache_enabled. D4 explicitly bans
// padding the prefix with filler to clear the threshold.
func buildSystemBlocks(tctx TranslateContext) []ai.SystemBlock {
	blocks := []ai.SystemBlock{
		{Text: prompts.SubtitleTranslatorSystemPrompt, CacheTTL: ai.CacheTTLNone},
	}

	perShow := prompts.BuildMetadataSection(metadataOf(tctx)) + prompts.BuildGlossarySection(tctx.Glossary)
	if perShow != "" {
		blocks = append(blocks, ai.SystemBlock{Text: perShow, CacheTTL: ai.CacheTTLNone})
	}

	blocks[len(blocks)-1].CacheTTL = ai.CacheTTL1h
	return blocks
}

// metadataOf maps the stage's context onto the prompts package's mirror type.
// prompts cannot import subtitle (that would cycle), so the mapping lives here.
func metadataOf(tctx TranslateContext) prompts.MediaMetadata {
	return prompts.MediaMetadata{
		Title:         tctx.Title,
		OriginalTitle: tctx.OriginalTitle,
		Year:          tctx.Year,
		Genres:        tctx.Genres,
		Overview:      tctx.Overview,
		Cast:          tctx.Cast,
		Countries:     tctx.Countries,
	}
}

// addUsage accumulates token usage across chunks and retries.
func addUsage(acc, u ai.CompletionUsage) ai.CompletionUsage {
	acc.InputTokens += u.InputTokens
	acc.OutputTokens += u.OutputTokens
	acc.CacheCreationInputTokens += u.CacheCreationInputTokens
	acc.CacheReadInputTokens += u.CacheReadInputTokens
	return acc
}

// checkTimestampInvariant is the FR17 structural guard: the translated track
// must carry exactly the routed cue count, and every cue must keep its routed
// Index/Start/End byte-for-byte. It passes by construction today — that is the
// point. It exists so a future refactor that starts reordering, dropping or
// re-numbering cues fails loudly instead of shipping desynced subtitles.
//
// source is the snapshot taken at stage entry, which also makes the check
// meaningful when the track is mutated underneath the stage.
func checkTimestampInvariant(source, translated []SubtitleBlock) error {
	if len(source) != len(translated) {
		return fmt.Errorf("%w: cue count %d != source %d",
			ErrSubtitleTimestampMismatch, len(translated), len(source))
	}
	for i := range source {
		s, t := source[i], translated[i]
		if s.Index != t.Index || s.Start != t.Start || s.End != t.End {
			return fmt.Errorf("%w: cue %d: got [%d] %s --> %s, want [%d] %s --> %s",
				ErrSubtitleTimestampMismatch, i, t.Index, t.Start, t.End, s.Index, s.Start, s.End)
		}
	}
	return nil
}
