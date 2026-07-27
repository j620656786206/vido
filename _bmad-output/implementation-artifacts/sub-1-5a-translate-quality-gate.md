# Story sub-1.5a: Translate core + quality gate (test-first)

Status: ready-for-dev

**Epic:** `epic-subtitle-pipeline-m1` (M1) · **Risk: 🔴 HIGH · TEST-FIRST (NAIL 3 lives here)** · **BACKEND-ONLY**
**Source:** `epics-subtitle-pipeline.md` § Story 1.5a (size-split 2026-07-27, IR-r2 M1) · architecture **P2/P3/P4** + **P11** + D-flow
**Depends on (all merged):** **sub-1-1** (`CachingCompleter` — acked below) · **sub-1-3** (`ErrSubtitleTranslateFailed`/`ErrSubtitleTimestampMismatch`) · **sub-1-4** (`ExtractedTrack` — acked below). **sub-1-2 NOT required** (RunVersion is 1.5b's).
**Blocks:** sub-1-5b (consumes `TranslateTrack` + its usage return; flips the cache TTLs this story deliberately leaves off), sub-1-6.
**Cross-stack split check:** backend tasks = 5, frontend tasks = **0** → single story.

---

## Story

As a NAS owner,
I want English subtitles translated to Traditional Chinese with guaranteed script and preserved timing,
so that the translated cues are trustworthy without hand-editing.

---

## 🔎 Codebase findings (verified 2026-07-27): the translate mechanics mostly EXIST — reuse them

Route C (9R-7) already ships, in `internal/ai/prompts/subtitle_translator.go` + `internal/services/translation_service.go`:

| Exists | Where | Reuse verdict |
|---|---|---|
| System prompt with Taiwan-usage rules + **`[N]` indexed output format** (multi-line rule included) | `SubtitleTranslatorSystemPrompt` | **Reuse verbatim as system block [0]** |
| Batch size **10** + context window **5** (read-only previous blocks for consistency) | `SubtitleTranslatorBatchSize` / `SubtitleTranslatorContextWindow` | **Reuse both constants** — chunk = 10 is the proven transport unit (P3); do not invent a new size |
| `BuildSubtitleTranslatorPrompt(contextBlocks, blocks)` + `BuildGlossarySection` | prompts pkg | Reuse |
| **`parseTranslationResponse(response, indices) map[int]string`** | `translation_service.go:302` | **Reuse** — the parse-by-index machinery is done |
| `TranslationMaxTokens = 4096` (sized for 10 blocks) | `translation_service.go:27` | Reuse |

**What is genuinely missing = exactly this story:** ① a P11 prompt-**version constant** (none exists), ② FR26 **metadata injection** (glossary section exists, metadata doesn't), ③ **usage returned to the caller** (the delta tree's `translation_service.go ✏️ "must return usage"`), ④ the **quality gate + per-cue retry** (`quality_gate.go` 🆕). Do not rebuild what the table above already provides.

**Layering constraint that decides where code lives:** the gate needs `detector.Detect`, which is in `internal/subtitle` — and **Rule 19 forbids `services → subtitle`** (compile-time, `boundaries_test.go`). Therefore: gate + retry loop + stitching = **subtitle side** (`pipeline.go`/`quality_gate.go`); the thin LLM call = **services side** (new single-chunk method on `TranslationService`). The architecture chain `pipeline.go → translation_service.go → ai` is exactly this.

---

## Acceptance Criteria

### AC #1 — [NAIL 3, TEST-FIRST] Quality gate runs BEFORE OpenCC; only affected cues retry

**Given** LLM output for a chunk, **when** any cue is **missing / empty / echoed / Simplified-leaked**, **then** only those cues are resent (FR16), **and** OpenCC runs only after the gate passes (P4).

`internal/subtitle/quality_gate.go`:

```go
type GateVerdict struct {
    FailedIndexes []int          // cue Index values needing retry
    Reasons       map[int]string // per-cue failure class: "missing"|"empty"|"echoed"|"simplified_leak"
}
// CheckChunk inspects RAW (pre-OpenCC) translations for one chunk.
func CheckChunk(source []SubtitleBlock, got map[int]string) GateVerdict
```

Per-cue rules (each one a test case):

| Class | Rule |
|---|---|
| `missing` | cue Index absent from `got` (covers cue-count mismatch; unexpected extra indexes are logged and ignored) |
| `empty` | translated text is empty/whitespace |
| `echoed` | normalized-equal to the source **and** contains ≥ 3 consecutive Latin letters (so `OK!`, numerals, names don't false-positive) |
| `simplified_leak` | `detector.Detect(text).SimplifiedCount > 0` — **any** simplified-only character fails the cue. Strict by design; ratio thresholds are for variant *classification*, not leak *detection* |

**The test-first obligation (NAIL 3, non-negotiable order):**
1. **First commit red:** `TestTranslateTrack_SimplifiedLeakTriggersPerCueRetry` — fake completer returns one deliberately-Simplified cue (e.g. `软件很好用`) among clean ones; assert the **second** LLM call contains **only** that cue's index, and the final output for it is Traditional.
2. **The order proof:** a recording fake converter + fake completer; assert the converter is invoked **only after** the last gate pass, and that the gate observed the leak (the retry fired). If OpenCC ran first, the leak would be converted away, the retry could never fire, and the gate would be dead code — the exact anti-pattern P4 names. The test fails against that implementation.

Quality retries are **semantic only** — max **2** per cue. Transport-level retries already live inside the `ai` client (`retryTransient`, D8); do **not** add another transport retry loop here.

### AC #2 — `[@contract-v1]` `TranslateTrack` — the stage contract (consumer: sub-1-5b)

**Given** a routed English track, **then** `internal/subtitle/pipeline.go` (created here; extended by 1.5b/1.6) exposes:

```go
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

type TranslateResult struct {
    Blocks       []SubtitleBlock      // translated + gated + OpenCC'd; Index/Start/End byte-equal source
    Usage        ai.CompletionUsage   // aggregated across all chunks + retries
    StubbornCues int                  // cues delivered with English fallback (see policy)
}

func (p *Pipeline) TranslateTrack(ctx context.Context, track *ExtractedTrack, tctx TranslateContext) (*TranslateResult, error)
```

- Chunks of **10** (`SubtitleTranslatorBatchSize`) with the **5**-block context window, sequential per track.
- Failure mapping: unrecoverable LLM/transport errors wrap **`ErrSubtitleTranslateFailed`**; the AC #4 invariant violation wraps **`ErrSubtitleTimestampMismatch`** (both sub-1-3 sentinels — their first consumers).
- **Stubborn-cue policy (ruled — flagged to Alexyu below):** a cue still failing after 2 quality retries keeps its **original English text** (fail-soft, NFR-R1) and increments `StubbornCues`; if stubborn cues exceed **5% of the track's cues**, the whole item fails with `ErrSubtitleTranslateFailed`. One flaky cue must not kill a 1000-cue episode; a broken track must not ship half-English.

### AC #3 — Timestamps never leave Go (P2, FR11) + the FR17 invariant

**Given** a chunk, **then** the LLM receives **numbered text only** — `BuildSubtitleTranslatorPrompt`'s existing shape; never `Start`/`End`, never serialized SRT (P2's named anti-pattern, "the single largest threat to FR11/FR17").

**Given** re-stitching, **then** each translated text is written back onto the source cue's `SubtitleBlock` (Index/Start/End untouched), and a final **invariant check** asserts: equal cue count, and per-cue `Index`/`Start`/`End` equality against the source track. Violation → `ErrSubtitleTimestampMismatch`. This is a cheap structural guard against future refactors — by construction it passes today, and that is the point.

### AC #4 — OpenCC final pass per cue (FR15) + metadata as context (FR26)

- After the gate passes, `converter.ConvertS2TWP` runs on **each cue's text** (idempotent on already-Traditional text). Reuse the existing `Converter` (🔒) — injected, never constructed per call (Rule 14).
- `prompts.BuildMetadataSection(tctx)` (new, sibling of `BuildGlossarySection`) renders FR26 context: title / original title / year / genres / overview / top-10 cast / production countries. Returns `""` on a zero-value context (byte-identical prompts for the no-metadata path — the `BuildGlossarySection` precedent).
- **System blocks, stable-first order** (the shape 1.5b's caching depends on): `[0]` = `SubtitleTranslatorSystemPrompt` (most stable) · `[1]` = metadata + glossary sections (per-show stable). **Both `CacheTTLNone` in this story** — cache policy (TTL flip, versioned key, <4096 disable-and-record) is **deliberately 1.5b's**, and turning it on here would ship caching without its policy.

### AC #5 — `TranslationService` gains a single-chunk, usage-returning method; Route C untouched

**Given** the pipeline controls chunking and retry, **then** the new service method is **single-chunk and thin**:

```go
// TranslateChunk sends ONE chunk (numbered cues + context window) and returns
// the parsed per-index map plus token usage. No internal batching, no retry —
// the pipeline (subtitle side) owns both.
func (s *TranslationService) TranslateChunk(ctx context.Context, sys []ai.SystemBlock,
    contextBlocks, blocks []prompts.SubtitleTranslatorBlock) (map[int]string, ai.CompletionUsage, error)
```

- Internally: type-assert `s.provider.(ai.CachingCompleter)` → `CompleteTextWithUsage` (usage flows); **fallback** to plain `CompleteText` with zero `CompletionUsage` + one `slog.Info` when the provider doesn't implement it (Gemini — the established degradation shape). This is the delta tree's `translation_service.go ✏️ must return usage`.
- Reuses `BuildSubtitleTranslatorPrompt` + `parseTranslationResponse` — **no new prompt-building or parsing code paths**.
- **Regression guard:** `Translate`/`TranslateWithGlossary`/`TranslateRequest` and every existing `translation_service_test.go` test are untouched and green.
- The pipeline consumes this through a narrow subtitle-side interface (`ChunkTranslator`) so `pipeline_test.go` runs on fakes.

### AC #6 — P11: prompt version constant lives with the prompt text

**Given** `internal/ai/prompts/subtitle_translator.go`, **then** it gains:

```go
// SubtitleTranslatorPromptVersion identifies the prompt revision for cache-key
// and provenance purposes (P11). ANY text change to the prompts or section
// builders in this file REQUIRES bumping this constant IN THE SAME EDIT —
// otherwise a re-run silently returns the previous translation and the pilot's
// A/B comparison is invalid (the architecture's named silent-failure trap).
const SubtitleTranslatorPromptVersion = "m1-v1"
```

Consumed by 1.5b (`RunVersion.PromptVersion` — per sub-1-2's split note, this story is the *definer*, 1.5b the *consumer*). If this story's `BuildMetadataSection` addition changes any existing prompt byte, the constant is already carrying it ("m1-v1" is the first versioned state).

### AC #7 — Tests (Rule 9/16) beyond the NAIL 3 pair

1. `quality_gate_test.go` — table over all four failure classes + the echo false-positive guards (`OK!`, numerals, names) + multi-failure chunks returning the exact failed-index set.
2. `pipeline_test.go` — retry-resends-only-failed-cues; stubborn ≤5% delivers with English fallback + correct `StubbornCues`; stubborn >5% fails with `errors.Is(ErrSubtitleTranslateFailed)`; usage aggregation sums across chunks **and** retries; FR17 invariant (a mutated-fake path asserting `ErrSubtitleTimestampMismatch`); context-window contents are previous **translated** blocks.
3. `translation_service_test.go` (extend) — `TranslateChunk` happy path; Gemini-shaped fallback (non-caching completer → zero usage, no error); existing tests untouched.
4. `go test ./...` + `pnpm lint:all` green.

### AC #8 — Scope fence

- ❌ No caching policy: no `cache_control` TTLs, no versioned key, no <4096 rule, no `cache_enabled` writes — **1.5b**.
- ❌ No delivery: no `placer.go` call, no provenance write, no `FindCompletedRun` resume, no DB writes of any kind — **1.5b**.
- ❌ No D10 per-show serialization gate — **1.5b**.
- ❌ No SSE broadcasts, no stage writes, no triggering/endpoint/flag — **1.6**.
- ❌ No model-ladder escalation (Haiku→Sonnet→Opus for stubborn cues) — the ladder is pilot-informed; M1 ships single-model. Note it in Completion Notes if stubborn rates suggest it early.
- ❌ No changes to `SubtitleTranslatorSystemPrompt`'s text beyond what `BuildMetadataSection` requires — and if any byte changes, P11's constant already covers it.

---

## Tasks / Subtasks

- [ ] **Task 1 — RED FIRST (AC #1):** write `TestTranslateTrack_SimplifiedLeakTriggersPerCueRetry` + the converter-order proof against stubs; confirm both fail.
- [ ] **Task 2 — Gate (AC #1):** implement `CheckChunk` + full class table tests; NAIL 3 tests go green.
- [ ] **Task 3 — Prompts (AC #4, #6):** `BuildMetadataSection` (+ zero-value byte-identity test) + `SubtitleTranslatorPromptVersion`.
- [ ] **Task 4 — Service (AC #5):** `TranslateChunk` + CachingCompleter type-assert/fallback + tests; verify Route C tests untouched-green.
- [ ] **Task 5 — Pipeline stage (AC #2, #3):** `pipeline.go` with `TranslateTrack` (chunk loop → gate → retry → English-fallback policy → per-cue OpenCC → stitch + invariant) + `pipeline_test.go`; record the two Rule 20 acks in Dev Notes; full gates (`go test ./...`, `pnpm lint:all`).

---

## Dev Notes

- **Rule 20 acks (record verbatim at implementation):** `confirmed against [@contract-v1] sub-1-1 AC #5` (`CachingCompleter`/`SystemBlock`/`CompletionUsage`) · `confirmed against [@contract-v1] sub-1-4 AC #1` (`ExtractedTrack` — SDH-filtered blocks with original numbering, which is why chunk indexes may have gaps; P1: cue identity is content-hash, never position).
- **File-overlap check:** `pipeline.go`/`quality_gate.go` new · `prompts/subtitle_translator.go` + `translation_service.go` edited — zero overlap with 1-1 (`claude.go`/`provider.go`), 1-2, 1-3 (`engine.go`/`errors.go`), 1-4 (`extractor`/`sdh_filter`/`router`/`ffprobe_service`). Parallel-draftable, **merge-ordered** after its three dependencies.
- **Why `TranslateChunk` doesn't batch:** retry granularity is the cue, transport is the chunk (P3). If the service batched internally, the pipeline couldn't resend only-failed cues without re-sending clean ones — the gate's whole point.
- **`ai.Governor`/budget ride along for free** — every call goes through the sub-1-1 client, whose `governed → retryTransient` nesting is NAIL 2's guarantee. Do not add throttling here.
- **Rule 13:** every chunk/gate error wrapped `%w`; stubborn-cue fallbacks logged at `slog.Warn` with index + reason (pilot data).

### Time-dependent visual coverage

**N/A — no wall-clock-reading components touched.** Backend-only Go. Rule 23 does not apply.

### References

- [Source: `epics-subtitle-pipeline.md`#Story 1.5a] · [architecture #P2/#P3/#P4 (:347-366) — the anti-patterns are verbatim test targets] · [#P11 (:392-394)] · [#M1 Pilot Instrumentation (:101-121) — why the version constant exists]
- [Source: `apps/api/internal/ai/prompts/subtitle_translator.go`:1-80 — prompt, `[N]` format, batch/context constants, `BuildGlossarySection`'s zero-value precedent]
- [Source: `apps/api/internal/services/translation_service.go`:27,111,119,259,302 — `TranslationMaxTokens`, the three untouchable Route C methods, `parseTranslationResponse`]
- [Source: `sub-1-1-claude-sdk-migration.md`#AC #5 · `sub-1-4-extract-filter-route.md`#AC #1 · `sub-1-3`#AC #2]
- [Source: `project-context.md`#Rule 13/14/19/20]

---

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

### Discovery Triage

- **Did this story discover any work outside its current scope?**
  - If **NO** beyond the pre-recorded item: state `N/A — no further out-of-scope work discovered`.
  - **① expand-scope-in-place → AC #5's reuse ruling.** The epic implied new translate machinery; discovery showed Route C already ships the prompt/format/parse/batch mechanics — the story was reshaped to *extend* (`TranslateChunk`, metadata section, version constant) rather than rebuild. Tracked by AC #5's regression guard on the three Route C methods.
- Reference: `project-context.md` Rule 24.

### File List

---

## Open Questions for Alexyu (non-blocking — dev proceeds with the stated rulings)

1. **Stubborn-cue policy (AC #2):** ≤5% of cues still failing after 2 quality retries ship with their **English original** (counted + logged); >5% fails the item. Alternative is strict (any stubborn cue fails the item — better for the trust bar, worse for coverage). Confirm or flip before 1.5b consumes the contract.
2. **Prompt version seed value:** `"m1-v1"` as the first versioned state (Route C's unversioned era is implicitly "pre"). Any preference for a different scheme, say now — 1.5b bakes it into the cache key.
