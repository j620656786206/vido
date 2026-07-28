# Story sub-1.5a: Translate core + quality gate (test-first)

Status: done

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
| `BuildSubtitleTranslatorPrompt(blocks, contextBlocks)` + `BuildGlossarySection` | prompts pkg | Reuse _(arg order corrected at CR — blocks first)_ |
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

- [x] **Task 1 — RED FIRST (AC #1):** write `TestTranslateTrack_SimplifiedLeakTriggersPerCueRetry` + the converter-order proof against stubs; confirm both fail.
- [x] **Task 2 — Gate (AC #1):** implement `CheckChunk` + full class table tests; NAIL 3 tests go green.
- [x] **Task 3 — Prompts (AC #4, #6):** `BuildMetadataSection` (+ zero-value byte-identity test) + `SubtitleTranslatorPromptVersion`.
- [x] **Task 4 — Service (AC #5):** `TranslateChunk` + CachingCompleter type-assert/fallback + tests; verify Route C tests untouched-green.
- [x] **Task 5 — Pipeline stage (AC #2, #3):** `pipeline.go` with `TranslateTrack` (chunk loop → gate → retry → English-fallback policy → per-cue OpenCC → stitch + invariant) + `pipeline_test.go`; record the two Rule 20 acks in Dev Notes; full gates (`go test ./...`, `pnpm lint:all`).

---

## Dev Notes

- **Rule 20 acks — RECORDED 2026-07-28 at implementation:** confirmed against [@contract-v1] (Story sub-1-1 AC #5) — `ai.CachingCompleter` / `ai.SystemBlock` / `ai.CompletionUsage` consumed unchanged by `TranslationService.TranslateChunk`. confirmed against [@contract-v1] (Story sub-1-4 AC #1) — `subtitle.ExtractedTrack` consumed unchanged by `Pipeline.TranslateTrack`; its SDH-filtered original numbering is why `CheckChunk` and the retry subset key on cue `Index` and never on position (covered by `TestCheckChunk_HonoursNonContiguousIndexes` + `TestTranslateTrack_PreservesCueIdentity`). Original drafting note: `confirmed against [@contract-v1] sub-1-1 AC #5` (`CachingCompleter`/`SystemBlock`/`CompletionUsage`) · `confirmed against [@contract-v1] sub-1-4 AC #1` (`ExtractedTrack` — SDH-filtered blocks with original numbering, which is why chunk indexes may have gaps; P1: cue identity is content-hash, never position).
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

claude-opus-5[1m] (dev — Amelia)

### Debug Log References

- **RED proof (Task 1)** — `go test ./internal/subtitle/ -run 'TestTranslateTrack_(SimplifiedLeakTriggersPerCueRetry|ConverterRunsOnlyAfterGatePasses)'` against the gate-less pipeline:
  - `TestTranslateTrack_SimplifiedLeakTriggersPerCueRetry`: `…should have 2 item(s), but has 1` — no retry fired.
  - `TestTranslateTrack_ConverterRunsOnlyAfterGatePasses`: `converter received pre-gate text "这个软件很好用" — OpenCC would have masked the leak` — the P4 anti-pattern reproduced, then eliminated by Task 2.
- **Gates (final)** — `go test ./... -count=1` exit 0 (34 pkgs) · `pnpm nx test api` green on re-run · `pnpm nx test web` 225 files / 2457 tests green · `pnpm nx run api:lint` (vet + staticcheck) clean · `pnpm lint:all` 0 errors / 120 pre-existing FE warnings · `prettier --check .` clean · `gofmt -l` clean on all six touched Go files · `pnpm run test:cleanup` → no orphaned processes.

### Completion Notes List

- 🔗 **AC Drift: NONE** (checked `BuildSubtitleTranslatorPrompt|parseTranslationResponse|SubtitleTranslatorSystemPrompt` across `_bmad-output/implementation-artifacts/*.md` — 3 hits: `9-2b-ai-subtitle-translation.md`, `9R-6-7-glossary-keystone.md`, this story; all REUSE not DRIFT. 9R-7's "a nil/empty glossary yields a **byte-identical** prompt" and 9-2b's AC #1–#6 both hold: `BuildMetadataSection` is a NEW builder no existing caller invokes, the version constant is additive, and `TranslateChunk` is a new method — `Translate`/`TranslateWithGlossary`/`TranslateRequest` and every pre-existing prompt byte are untouched.)
- 📎 **Contract Stamps: FOUND** (3 stamped ACs across 3 files — this story defines AC #2 `[@contract-v1]` (`TranslateContext`/`TranslateResult`/`TranslateTrack`, stamped in `pipeline.go`, consumers sub-1-5b/sub-1-6, first version so no bump). Upstream: sub-1-1 AC #5 `[@contract-v1]` and sub-1-4 AC #1 `[@contract-v1]` — both greped at v1, both acked verbatim in Dev Notes, versions reconcile, no bump anywhere.)
- 🎭 **A11y Pre-Flight: N/A** (100% backend — no `apps/web/` files touched).
- 🎨 **UX Verification: SKIPPED** — no UI changes in this story.
- **Pre-existing failure — already filed (Epic 9c retro AI-2, option 2):** `TestScannerService_SSEBroadcast_ScanCancelled` failed on 1 of 4 full-package `go test ./internal/services/` runs. Confirmed pre-existing and unrelated: with this story's changes **stashed** (`git stash push -u -- apps/api`) the clean tree ran 3/3 green, and with the changes applied it ran 3/4 green — i.e. intermittent on both trees, exactly as the tracked entry describes. No new tracking entry created; the existing `preexisting-fail-scanner-sse-scan-cancelled-flake` backlog entry was enriched with this recurrence datapoint instead. Full `go test ./...` and the re-run `pnpm nx test api` both exit 0.
- **Gate correction to the story's worked example (AC #1):** the story suggests `软件很好用` as the deliberately-Simplified test cue, but neither `软` nor `件` is in `detector.simplifiedOnlySet`, so that string yields `SimplifiedCount == 0` and would NOT trip the gate — the NAIL-3 test would have been green-by-accident. The tests use `这个软件很好用` instead (`这`/`个` are both simplified-only). The gate rule itself is unchanged; only the fixture had to be one the detector can actually see. Worth carrying into 1.5b's pilot fixtures.
- **Echo-guard trade-off (documented in `isEchoed`):** the AC's rule is *normalized-equal AND ≥3 consecutive Latin letters*, implemented verbatim. A cue that is ONLY a long proper noun ("John Smith") therefore false-positives and burns up to two retries before the stubborn policy delivers the very text that was already correct. Cost is bounded and never corrupts output; the guards that DO hold (`OK!`, `1999`, `J.T.`, `Mr. Ed`) are each a test case.
- **P11 enforcement is mechanical, not aspirational:** `TestSubtitleTranslatorPromptVersion_PinsPromptText` sha256s every prompt surface in `subtitle_translator.go`, so editing the system prompt or either section builder fails the suite until `SubtitleTranslatorPromptVersion` and the digest are updated in the same edit.
- **Stubborn rates and the model ladder (AC #8 note):** no pilot data exists yet — every stubborn-cue path in this story is fake-driven. The `slog.Warn` at each fallback carries `cue_index` + `reason` + `stream_index`, which is the data 1.5b/1.6's pilot needs to decide whether the Haiku→Sonnet→Opus ladder is warranted.
- **Cache TTLs are deliberately off:** both system blocks ship `CacheTTLNone` and a test asserts it. Turning them on is 1.5b's, together with the versioned key and the <4096-token disable-and-record rule.

### Discovery Triage

- **Did this story discover any work outside its current scope?**
  - **① expand-scope-in-place → AC #5's reuse ruling** (pre-recorded at drafting). The epic implied new translate machinery; discovery showed Route C already ships the prompt/format/parse/batch mechanics — the story was reshaped to *extend* (`TranslateChunk`, metadata section, version constant) rather than rebuild. Tracked by AC #5's regression guard on the three Route C methods; verified green (all 14 pre-existing `translation_service_test.go` tests untouched and passing).
  - **No further out-of-scope work discovered.** The one finding that surfaced during implementation — the `软件很好用` fixture not tripping the detector — was absorbed in place (it is a test-fixture correction inside AC #1's own scope, not new work), and the `detector.simplifiedOnlySet` coverage question it raises is already the *documented* design of that set ("~200 representative characters… a full production set would contain ~2000") and is precisely why the OpenCC pass remains as the backstop after the gate. No lane ②/③ entry owed.
- Reference: `project-context.md` Rule 24.

### File List

| File | Change |
| --- | --- |
| `apps/api/internal/subtitle/pipeline.go` | **new** — `[@contract-v1]` `TranslateContext` / `TranslateResult` / `Pipeline` / `TranslateTrack`; `ChunkTranslator` + `VariantConverter` ports; chunk loop → gate → per-cue retry → English fallback → 5% ceiling → per-cue OpenCC → stitch → FR17 invariant; `buildSystemBlocks` / `metadataOf` / `contextWindow` / `convertAndStitch` / `promptBlocksOf` / `subsetByIndex` / `addUsage` / `checkTimestampInvariant` |
| `apps/api/internal/subtitle/pipeline_test.go` | **new** — 14 tests incl. the two NAIL-3 proofs, chunking + translated context window, stubborn-cue policy either side of the ceiling, usage aggregation, FR17 (mutated-track path + invariant table), system-block shape, converter degradation, cancellation, port conformance assertions |
| `apps/api/internal/subtitle/quality_gate.go` | **new** — `GateVerdict` + `CheckChunk` over `missing`/`empty`/`echoed`/`simplified_leak`; `isEchoed` + `latinRunPattern` guard; unexpected-index warning |
| `apps/api/internal/subtitle/quality_gate_test.go` | **new** — 20-case class/guard table + multi-failure index set + non-contiguous indexes + unexpected indexes + empty chunk |
| `apps/api/internal/ai/prompts/subtitle_translator.go` | **modified** — added `SubtitleTranslatorPromptVersion = "m1-v1"` (P11), `MediaMetadata`, `BuildMetadataSection`, `MetadataCastLimit`, `joinNonEmpty`, `collapseLines`. Existing prompt text + builders untouched |
| `apps/api/internal/ai/prompts/subtitle_translator_test.go` | **modified** — added zero-value byte-identity, full-field rendering, partial metadata, cast cap, and the sha256 prompt-version pin |
| `apps/api/internal/services/translation_service.go` | **modified** — added `TranslateChunk` (CachingCompleter assert → `CompleteTextWithUsage`; Gemini-shaped fallback to `CompleteText` + zero usage) and `flattenSystemBlocks`. `Translate` / `TranslateWithGlossary` / `TranslateRequest` / `parseTranslationResponse` untouched |
| `apps/api/internal/services/translation_service_test.go` | **modified** — added `cachingTranslationMock` + 4 `TranslateChunk` tests (happy path, non-caching degradation, error chaining, empty no-op). Existing 14 tests untouched |
| `_bmad-output/implementation-artifacts/sprint-status.yaml` | **modified** — `sub-1-5a-translate-quality-gate` → `review`; scanner-SSE flake entry enriched with this story's recurrence + stash-comparison datapoint |

---

### Code Review Record (2026-07-28, adversarial /code-review, Fable 5)

**0 HIGH / 3 MEDIUM / 2 LOW — all fixed in-review.** Git vs File List: 0 discrepancies. 🔒 Rule 7 Wire Format: PASS (0 new error codes in scope — the 4 `SUBTITLE_` sentinels shipped in sub-1-3; this story only consumes them). 🔒 Rule 20 Contract Bump: N/A (new `[@contract-v1]` stamps only, no bump tokens in story or diff; both upstream acks verified greppable). 🔒 Rule 25 Mega-line: N/A (project-context.md untouched).

- **M1 (fixed):** the P11 sha256 pin used a 3-name cast, so a `MetadataCastLimit` change altered real prompts (casts > 10) without failing the pin — the exact silent-failure trap P11 guards. The pin fixture now carries `MetadataCastLimit+2` cast entries so the cap is inside the fingerprint; digest updated `12f3ce76…` → `4ca01f92…` (fixture change only — every production prompt byte is unchanged, so no version bump owed).
- **M2 (fixed):** `NewPipeline` silently accepted a nil `VariantConverter` (`convertAndStitch` nil-guarded it), which would void FR15's Traditional-script guarantee with zero signal if sub-1-6's wiring ever forgot the port. Both ports now panic at construction (`TestNewPipeline_NilPortsPanic`); the runtime nil-guard is removed.
- **M3 (fixed):** `normalizeForEcho` ignored punctuation, so a model echoing the source minus its final period escaped the gate and shipped an untranslated English cue uncounted. Normalization now maps `unicode.IsPunct` runes to spaces before field-collapsing; the Latin-run guard still protects `OK!`/`J.T.`/`Mr. Ed` (2 new class-table cases).
- **L1 (fixed):** the story's reuse table wrote `BuildSubtitleTranslatorPrompt(contextBlocks, blocks)` — the real signature is `(blocks, contextBlocks)`. Corrected so 1.5b doesn't copy the inversion.
- **L2 (fixed):** `logUnexpectedIndexes` used the global `slog` instead of the pipeline's component-tagged logger. `CheckChunk` keeps its AC #1 signature via an exported wrapper; the pipeline calls the logger-injected `checkChunk`.
- **Observation (not a story defect):** the tracked `preexisting-fail-scanner-sse-scan-cancelled-flake` reproduced during review even on **isolated** single-test runs (2/3 failed under session load) — worse than the backlog entry's "10× isolated green" characterization. Datapoint added to the backlog entry; zero scanner code touched by this story.

Gates re-run post-fix: `go build` + `go test ./internal/{subtitle,ai/prompts,services(-run Translation),}/` all green · `go vet` clean · `gofmt -l` clean on all touched files · `prettier --check` clean on the story file.

## Change Log

| Date | Change |
| --- | --- |
| 2026-07-28 | **Adversarial CR (0H/3M/2L, all fixed in-review):** P11 pin fixture now fingerprints `MetadataCastLimit` (digest → `4ca01f92…`, no version bump — no production prompt byte changed); `NewPipeline` panics on nil ports (FR15 wiring guard) + nil-guard removed from `convertAndStitch`; `normalizeForEcho` strips punctuation (dropped-period echoes now caught, 2 new gate cases); story reuse-table arg order corrected; gate warnings ride the component-tagged logger via unexported `checkChunk` (AC #1 public signature untouched). +3 tests (panic guard + 2 echo cases). |
| 2026-07-28 | **Task 5 (AC #2, #3, #4).** `pipeline.go` completed: `buildSystemBlocks` emits `[0]` stable prompt + `[1]` metadata‖glossary (block `[1]` omitted entirely when empty), both `CacheTTLNone`; a `routed` cue-identity snapshot is taken at stage entry so `checkTimestampInvariant` compares against the track as routed; `convertAndStitch` runs the per-cue OpenCC pass only after the last gate pass and aggregates converter failures into one `slog.Warn` (Rule 13 case 3, justified — the gate already guarantees no simplified leak); chunk-boundary `ctx.Err()` check chains BOTH `ErrSubtitleTranslateFailed` and `context.Canceled` (sub-1-4 CR M1/M2 precedent). `pipeline_test.go` completed to 14 tests: chunking-at-10 + translated context window, stubborn ≤5% English fallback (1-in-20 ships), stubborn >5% fails (1-in-10 errors), usage aggregation across chunks AND retries, FR17 via a mutated-track path + a direct 5-case invariant table, system-block shape, converter-failure degradation, cancellation, empty-track rejection, plus compile-time port assertions (`*services.TranslationService` → `ChunkTranslator`, `*Converter` → `VariantConverter`). |
| 2026-07-28 | **Task 4 (AC #5).** `translation_service.go` gains `TranslateChunk` — single-chunk, no batching, no retry — type-asserting `ai.CachingCompleter` for `CompleteTextWithUsage` (usage flows) and degrading to `CompleteText` + zero `CompletionUsage` + one `slog.Info` for a non-caching provider (Gemini), with `flattenSystemBlocks` collapsing the ordered blocks for the single-string API. Reuses `BuildSubtitleTranslatorPrompt` + `parseTranslationResponse` verbatim. `Translate`/`TranslateWithGlossary`/`TranslateRequest` and all 14 pre-existing tests untouched and green. |
| 2026-07-28 | **Task 3 (AC #4, #6).** `prompts/subtitle_translator.go`: `SubtitleTranslatorPromptVersion = "m1-v1"` (P11) + `MediaMetadata` + `BuildMetadataSection` (title / original title / year / genres / production countries / top-10 cast / collapsed overview; `""` on a zero value) + `MetadataCastLimit = 10`. Existing prompt text and builders untouched — the no-metadata path stays byte-identical. `TestSubtitleTranslatorPromptVersion_PinsPromptText` fingerprints every prompt surface in the file (sha256 `12f3ce76…`) so a text edit that forgets the version bump fails the suite. |
| 2026-07-28 | **Task 2 (AC #1) — GREEN.** `quality_gate.go`: `GateVerdict` + `CheckChunk` over the four classes (`missing`/`empty`/`echoed`/`simplified_leak`, first-match wins, strict `SimplifiedCount > 0`), the `[A-Za-z]{3,}` echo guard, and a `slog.Warn` for model-invented indexes. `pipeline.go` chunk loop gains gate → per-cue retry (`maxQualityRetries = 2`, semantic only) → English-fallback + `StubbornCues` → the 5% ceiling. Both NAIL-3 tests green; `quality_gate_test.go` adds 20 class/guard cases + multi-failure index-set + non-contiguous-index (P1/P7) + unexpected-index cases. |
| 2026-07-28 | **Task 1 (AC #1) — RED FIRST.** `pipeline.go` created with the `[@contract-v1]` type surface (`ChunkTranslator`/`VariantConverter` ports, `TranslateContext`, `TranslateResult`, `Pipeline`, `TranslateTrack`) and a deliberately GATE-LESS body (translate → OpenCC → stitch). `pipeline_test.go` adds the two NAIL-3 tests. Both confirmed RED: the leak test failed on `should have 2 item(s), but has 1` (no retry fired) and the P4 order proof failed on `converter received pre-gate text "这个软件很好用"` — the exact anti-pattern the story names. |

---

## Open Questions — RULED by Alexyu at CR (2026-07-28), both confirmed as implemented

1. **Stubborn-cue policy (AC #2): ✅ CONFIRMED — keep the 5% fail-soft ceiling.** ≤5% of cues still failing after 2 quality retries ship with their **English original** (counted + logged); >5% fails the item with `ErrSubtitleTranslateFailed`. The strict alternative was declined. The `[@contract-v1]` stubborn semantics 1.5b consumes are final.
2. **Prompt version seed value: ✅ CONFIRMED — `"m1-v1"`.** Route C's unversioned era is implicitly "pre"; prompt edits bump to `m1-v2`, the M2 milestone starts `m2-v1`. 1.5b bakes this into the cache key as-is.
