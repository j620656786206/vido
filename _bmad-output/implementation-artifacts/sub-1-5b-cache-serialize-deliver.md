# Story sub-1.5b: Cache, per-show serialization, and delivery

Status: review

**Epic:** `epic-subtitle-pipeline-m1` (M1) · **Risk: 🔴 HIGH** · **BACKEND-ONLY**
**Source:** `epics-subtitle-pipeline.md` § Story 1.5b (size-split 2026-07-27, IR-r2 M1) · architecture **D2/D3/D4/D10** + **P1/P5/P9** + § M1 Pilot Instrumentation
**Depends on (all merged):** **sub-1-5a** (`TranslateTrack` — acked below) · **sub-1-2** (`RunVersion` + `SubtitleRunRepository` + status enum — acked below) · **sub-1-4** (`RouteDecision` — acked below) · sub-1-1/1-3 transitively.
**Blocks:** sub-1-6 (wires `ProcessItem` to the flag/endpoint/scanner and connects the progress hook to SSE).
**Cross-stack split check:** backend tasks = 5, frontend tasks = **0** → single story.

---

## Story

As a NAS owner,
I want translations cached, serialized per show, and delivered atomically with provenance,
so that re-runs are cheap, cache writes are not duplicated, and the `.zh-Hant.srt` appears beside the file exactly once.

---

## ⚠️ Two different caches — do not conflate them (D4 contains both)

| Cache | What it stores | Key | Recorded where |
|---|---|---|---|
| **Segment cache** (AD #4 tier — `cache_entries` via `CacheRepositoryInterface`) | the **final, post-OpenCC translated text per cue** | `subseg:v1:` + sha256(cueText + RunVersion fields) | cache hit/miss counts in logs |
| **Prompt cache** (Anthropic `cache_control` on the stable prefix, via sub-1-1's `SystemBlock.CacheTTL`) | the provider-side prefix (rules + metadata + glossary) | n/a — prefix match | **`subtitle_runs.cache_enabled`** — the D4 "disable and record" ruling applies to THIS one |

A dev who conflates them will either record segment-cache hits into `cache_enabled` (wrong) or try to token-count the segment cache (meaningless). The ACs below keep them separate.

---

## Acceptance Criteria

### AC #1 — `[@contract-v1]` `ProcessItem` — the complete item flow (consumer: sub-1-6)

**Given** a media item, **then** `pipeline.go` (extending 1.5a's file) exposes:

```go
// [@contract-v1] — consumed by sub-1-6 (flag seam, endpoint, scanner enqueue).
type MediaRef struct{ ID, MediaType string } // internal vocabulary: movie|series|episode (sub-1-2 AC #1)
type ProcessOptions struct{ Force bool }
type ProcessOutcome struct {
    Run          *models.SubtitleRun // the recorded run (nil on pre-flight early-exit)
    Kind         RouteKind           // what happened
    SubtitlePath string              // non-empty when a sidecar was placed
}
func (p *Pipeline) ProcessItem(ctx context.Context, ref MediaRef, opts ProcessOptions) (*ProcessOutcome, error)
```

Flow (each numbered step is a required test seam):

1. **Pre-flight (AC #2)** — may exit before any ffprobe/LLM spend.
2. Create `subtitle_runs` row (`pending`→`running`); media row → `extracting` (the probe is milliseconds; extraction is the long half — `probing` stays an SSE-only granularity for 1.6's hook).
3. `SelectAndRoute` (1.4). Verdict handling:
   - `RouteNoTextSource` → media `no_text_source`, run `skipped` + reason. `RouteSkip` → media `skipped`, run `skipped` + reason.
   - `RouteDeliverDirect` → serialize (`SerializeSRT`) → **deliver (AC #6)** as `zh-Hant`.
   - `RouteConvertThenDeliver` → `ConvertS2TWP` on the serialized text → deliver as `zh-Hant`.
   - `RouteTranslate` → media `translating` → segment-cache split (AC #3) → D10 gate (AC #5) → `TranslateTrack` (1.5a) on the miss-subset → merge by `Index` → full-set FR17 invariant → serialize → deliver.
4. **Deliver → provenance → terminal status, in that order (P9).**
5. **Failure policy (ruled — flagged to Alexyu):** any step failing → run `failed` + `error_message` (zh-TW not required here — Rule 3 messages are the handler's, 1.6); media row **reverts to `not_searched`** (retryable — NFR-R3's granular-recovery reading; the run row is the audit trail, and enumeration is scan/manual-triggered so this cannot hot-loop).
6. Progress: a **nil-safe `Pipeline.progress func(stage PipelineStage, msg string)` struct field**, invoked at stage transitions and **once per chunk** inside the translate loop (P8's throttle grain). **No Rule 20 bump** — 1.5a's stamped `TranslateTrack` signature is untouched; the hook is struct-internal. 1.6 connects it to SSE.

### AC #2 — Pre-flight: P5 predicate + `force` + resume refinement

**Given** an item, **before any extraction or LLM call**:

1. **P5 check (the gate):** the expected sidecar path (placer naming, `zh-Hant`) **exists + `ParseSRT` succeeds + cue count > 0** → and `!opts.Force` → **early-exit, no new run row**, `slog.Info` with reason. An existence-only check is the named P5 anti-pattern (a truncated file must not block regeneration — it fails `ParseSRT` and we proceed).
2. **Resume refinement (NFR-R3):** when the P5 check exits early, consult `FindCompletedRun(ref, RunVersion)` purely to **log** whether this is a version-matched resume-skip or a foreign/pre-pipeline sidecar. The sidecar predicate is the gate; the run lookup refines the reason.
3. **The escape hatch is behaviour, not code:** sidecar deleted by the operator → step 1 passes → full re-run proceeds *even if* a `completed` run exists (the architecture's "delete and re-trigger is inherently a valid re-run").
4. **`force`** bypasses the P5 exit **and** segment-cache **reads** (writes still happen — refresh), overwrites via placer (its `.bak` backup is the safety net), and rides the Governor like every call (automatic — nothing to build).

### AC #3 — Segment cache: `hash(cue) + RunVersion` (P1), SQLite tier, 30d

1. **`MetadataHash` is defined HERE** (per sub-1-2's split note): sha256 over a **canonical, field-ordered** serialization of `TranslateContext`'s metadata fields (title|original|year|genres-sorted|overview|cast|countries-sorted — a fixed `\x1f`-joined order, documented in code). Non-determinism in this hash silently splits the cache and invalidates the pilot's A/B — it gets its own test.
2. `RunVersion` = `{MetadataHash, GlossaryVersion: "" (M1), PromptVersion: prompts.SubtitleTranslatorPromptVersion, ModelID: p.modelID}` (`modelID` is a `Pipeline` field, wired from config by 1.6).
3. Key = `subseg:v1:` + sha256(cueText + `\x00` + the four RunVersion fields joined). **Content-hash, never cue index** — SDH filtering shifts positions (P1's exact trap).
4. Value = the **final post-OpenCC** text. `cacheType = "subtitle_segment"`, TTL **30d** (the AI-parsing precedent; version-bumped keys orphan stale entries anyway).
5. Before translating: split cues into hits (filled from cache, excluded from chunks) and misses (→ `TranslateTrack` on a reduced `ExtractedTrack` — legal, the stamped signature takes the track; **the 1.5a contract is not touched**). After: write misses' final text back.
6. Storage via a narrow `SegmentCache` interface over `CacheRepositoryInterface.Get/Set` — fakes in tests.

### AC #4 — Prompt-cache policy: 1h TTL + detection-based `cache_enabled` (D4)

1. The **last** stable system block (`[1]` metadata+glossary) gets `CacheTTL1h` — one breakpoint caches `[0]+[1]` together (prefix semantics; 1h because a season batch spans tens of minutes — architecture fact 7).
2. **`cache_enabled` is recorded by detection, not estimation:** after the item's **first** chunk, inspect `CompletionUsage` — `CacheCreationInputTokens == 0 && CacheReadInputTokens == 0` ⇒ the prefix silently failed to cache (the <4096 haiku minimum, or a non-caching provider) ⇒ `subtitle_runs.cache_enabled = false`; any non-zero ⇒ `true`. No tokenizer heuristics — the API's own usage is the truth (this is exactly why sub-1-1 surfaced the two cache fields).
3. Gemini-shaped fallback (zero usage from 1.5a's degradation path) ⇒ `false`. **No padding the prefix to reach the minimum** — D4's explicit ban.

### AC #5 — D10: per-show first-request gate

1. Scope: **series only** — gate key = the series' `MediaRef.ID` (episodes of one show share the prefix). Movies bypass (nothing shares their prefix).
2. Semantics: the **first chunk request** for a show runs alone; when it returns, waiters release (the cache entry is readable only after the first response begins — architecture fact 6). Subsequent items within the **warm window (50 min < the 1h TTL)** skip the gate; a stale entry makes the next requester "first" again (re-warm).
3. Implementation: `Pipeline`-owned keyed latch — `mutex + map[showKey]{state, warmedAt}`; stale entries pruned on access (**Rule 14 bounded-map**), `ctx` cancellation honoured while waiting. This is deliberately an orchestrator concern — the worker pool (1.6) stays show-unaware.

### AC #6 — Delivery + provenance, in P9 order (FR32)

1. Sole writer: `placer.Place(PlaceRequest{MediaFilePath, SubtitleData, Language: "zh-Hant", Format: "srt", Score: 0})` (`placer.go:43-77`) — atomic write + `.bak`. The pipeline never touches the media folder itself.
2. **Order is load-bearing:** place → confirm success → `subtitle_runs` update (`completed`, `output_path` = `PlaceResult.SubtitlePath`, `cue_count`, `source_language`, the `RunVersion` tuple, `cache_enabled`) → media row terminal (`found` + language `zh-Hant` via the existing `UpdateSubtitleStatus` writers). The reverse leaves provenance claiming a file that a crash never wrote (P9).
3. `subtitle_search_score` / `subtitle_last_searched` stay unset — extract/translate is not a search (the architecture's explicit non-repurposing note).

### AC #7 — Tests (Rule 9/16)

1. `pipeline_test.go` (extend) — the numbered ProcessItem seams: every verdict branch's status+run writes; failure → run `failed` + media `not_searched`; P9 order via a recording fake placer/repo (provenance write observed strictly after place); pre-flight early-exit does zero probe/LLM calls (spy prober asserts 0 hits); `force` bypasses P5 + cache reads but still cache-writes.
2. Segment cache — hit/miss split (hits never reach the fake translator); content-hash key stability test (same cue+version ⇒ same key; any single `RunVersion` field change ⇒ different key — four sub-cases, the sub-1-2 pattern); **MetadataHash canonicalization** (field order shuffle in construction ⇒ identical hash; value change ⇒ different).
3. Prompt-cache recording — fake usage with creation>0 ⇒ `cache_enabled=true`; both-zero ⇒ `false`; zero-usage fallback ⇒ `false`.
4. D10 — two concurrent episodes, one show: second's first chunk blocks until first returns (channel-instrumented fake translator); warm window skips; stale re-warms; movie bypass; cancellation while waiting returns `ctx.Err()`.
5. Integration (real `:memory:` SQLite — the Rule 15/bugfix-20-1 lesson): full `ProcessItem` happy path over fakes for ffmpeg/LLM but **real** `SubtitleRunRepository` + cache repo — run row's 16 columns land correctly.
6. `go test ./...` + `pnpm lint:all` green.

### AC #8 — Scope fence

- ❌ No SSE (`progress` hook stays nil in tests; 1.6 connects it), no `subtitle_progress` broadcasts, no P8 wiring beyond the hook call sites.
- ❌ No feature flag, no `batch.go:244`, no endpoint, no scanner enqueue, no worker pool, no capability gate — **1.6**.
- ❌ No zh-TW user-facing messages (handler concern, 1.6) — `error_message` here is diagnostic English + the sentinel code.
- ❌ No model-ladder escalation, no glossary population (`GlossaryVersion` stays `""`), no cost-estimate UI (FR14, P2).
- ❌ No new migrations — `cache_entries` and `subtitle_runs` both exist (sub-1-2).
- ❌ No frontend, no `.pen`.

---

## Tasks / Subtasks

- [x] **Task 1 — Pre-flight + P5 (AC #2):** predicate helper (reusing `ParseSRT` + placer naming), force/resume semantics + tests (incl. the truncated-file case).
- [x] **Task 2 — Segment cache + RunVersion assembly (AC #3):** canonical `MetadataHash`, key builder, `SegmentCache` iface, hit/miss split + merge; tests.
- [x] **Task 3 — Prompt-cache policy (AC #4):** TTL flip on block `[1]`, first-chunk detection, `cache_enabled` recording; tests.
- [x] **Task 4 — D10 gate (AC #5):** keyed latch + warm window + pruning + cancellation; concurrency tests.
- [x] **Task 5 — ProcessItem + delivery (AC #1, #6):** verdict handling, status/run lifecycle, P9-ordered delivery, progress hook, failure policy; the AC #7 seam tests + integration test; record the four Rule 20 acks in Dev Notes; full gates.
  - [x] 5.1 (absorbed, Rule 24 lane ① — see Discovery Triage ②) progress hook carries `MediaRef`; sub-1-6's ack line + sprint-status entry corrected in the same change.

---

## Dev Notes

- **Rule 20 acks (record verbatim at implementation):** `confirmed against [@contract-v1] sub-1-5a AC #2` (`TranslateTrack`/`TranslateResult`) · `confirmed against [@contract-v1] sub-1-2 AC #2` (status values written here) · `confirmed against [@contract-v1] sub-1-2 AC #4` (`RunVersion` — this story is its cache-key consumer) · `confirmed against [@contract-v1] sub-1-4 AC #1` (`RouteDecision` verdict handling) · plus sub-1-1 AC #5's `CacheTTL1h` (the TTL flip this story performs).
- **Status writers:** the existing `UpdateSubtitleStatus` repo methods (movie `:816` / series `:799` / episode `:273`) behind a narrow subtitle-side writer interface keyed by `MediaType` — wired by 1.6, faked here. `IsValid()` (sub-1-2) guards writes on the paths this story owns.
- **Why failure → `not_searched`:** the enum has no `failed`; leaving `translating` strands a phantom in-flight state; `not_found` lies (nothing was searched). `not_searched` + the `failed` run row = retryable with an audit trail. Flagged below.
- **Rule 13/14:** every step wrapped `%w`; one `Pipeline` with injected deps constructed once (1.6), latch map bounded.

### Time-dependent visual coverage

**N/A — no wall-clock-reading components touched.** Backend-only Go. (The D10 warm window reads the clock in *backend* code — Rule 23 governs frontend visual fixtures only; the D10 tests inject a `now func() time.Time` field for determinism, which is just good hygiene.)

### References

- [architecture #D3/#D4/#D10 (:235-290) · #P1/#P5/#P9 (:343-386) · #M1 Pilot Instrumentation (:101-125, force + variant isolation) · fact 6/7 (:138-139, first-request + TTL economics)]
- [`apps/api/internal/subtitle/placer.go`:43-77 · `repository/interfaces.go`:286-300 (`CacheRepositoryInterface`) · `sub-1-2` AC #1/#3/#4/#5 · `sub-1-5a` AC #2/#4]
- [`project-context.md`#Rule 13/14/15/20 · AD #4 tiered cache TTLs]

---

## Dev Agent Record

### Agent Model Used

Amelia (Developer Agent) · Claude Opus 5 (1M context), effort xhigh · 2026-07-28

### Debug Log References

RED verified before every task (build failure naming exactly the new symbols, or a failing assertion):

| Task | RED signal |
|---|---|
| 1 | `vet: pipeline_test.go:688: undefined: ExpectedSidecarPath` |
| 2 | `vet: segment_cache_test.go:88: undefined: MetadataHash` |
| 3 | `vet: pipeline_test.go:860: undefined: processScope` |
| 4 | `vet: show_gate_test.go:40: undefined: newShowGate` |
| 5 | build failure on the five item-flow ports |

Falsification checks (deliberately break the implementation, confirm the test catches it, restore):

| Guard | Falsification | Result |
|---|---|---|
| D10 wiring | `enterShowGate` forced to always no-op | `TestTranslateTrack_SerializesTheShowsFirstRequest` FAILS — "the second episode's first request must wait" |
| Failure policy | media revert changed `not_searched` → `not_found` | 8 FAIL lines across `TestProcessItem_FailurePolicy` |
| Merged-track FR17 guard | `checkTimestampInvariant` removed from `translateWithCache` | **still green** — recorded honestly below, the guard is a construction guard (same class as sub-1-5a's), covered directly by `TestMergeCues_DesyncIsCaughtByTheInvariant` |

### Completion Notes List

- 🔗 **AC Drift: NONE** (checked `CacheTTLNone|TTL flip`, `NewPipeline`, `ProcessItem` across `_bmad-output/implementation-artifacts/*.md` — 12 hits, all REUSE not DRIFT). The two candidates and why they are REUSE:
  - The **CacheTTL flip on `buildSystemBlocks`** touches a file in sub-1-5a's File List, but sub-1-5a's own story text states it twice (`:112`, `:220`) as a deliberate deferral — *"Both `CacheTTLNone` in this story — cache policy … is deliberately 1.5b's"*. It is a planned handoff, not drift. Its lock-in test moved WITH the policy rather than being deleted: `TestTranslateTrack_SystemBlocksAreStableFirstAndUncached` → `…AndCacheBreakpointed`, now asserting `[0]=None, [1]=1h`.
  - **`NewPipeline` gained a variadic `...PipelineOption`.** Backward-compatible: sub-1-5a's three-argument call sites and its CR-M2 `TestNewPipeline_NilPortsPanic` are untouched and green. `NewPipeline` carries no `[@contract-vN]` stamp (sub-1-5a stamped `TranslateTrack`/`TranslateResult`/`TranslateContext` only).
- 📎 **Contract Stamps: FOUND** (5 stamped ACs across 5 files). Produced: **AC #1 `[@contract-v1]`** (`MediaRef` / `ProcessItemOptions` / `ProcessOutcome` / `ProcessItem`, stamped in `pipeline.go` + `process_item.go`; consumer sub-1-6) — a NEW v1, so **no bump and no stale-mark obligation**. Consumed, all greped at v1 and all reconciling:
  - `confirmed against [@contract-v1] sub-1-5a AC #2` — `TranslateTrack` / `TranslateResult` / `TranslateContext` consumed **unchanged**; the miss-subset is passed as a reduced `ExtractedTrack`, which the stamped signature already accepts.
  - `confirmed against [@contract-v1] sub-1-2 AC #2` — every media status written here (`extracting`, `translating`, `found`, `no_text_source`, `skipped`, `not_searched`) is a member of the 9-value set; `IsValid()` guards every write.
  - `confirmed against [@contract-v1] sub-1-2 AC #4` — `RunVersion`; this story is its cache-key consumer and defines the deferred `MetadataHash` half.
  - `confirmed against [@contract-v1] sub-1-4 AC #1` — `RouteDecision` / `RouteKind` / `ExtractedTrack`; all five verdicts handled, `default:` fails closed. **`RouteKind` was deliberately NOT extended** with an "already done" member for the pre-flight early-exit (that would be a bump on a shipped contract) — `ProcessOutcome.Run == nil` is the discriminator instead.
  - `confirmed against [@contract-v1] sub-1-1 AC #5` — `CachingCompleter` / `SystemBlock.CacheTTL` / `CompletionUsage`; this story performs the `CacheTTL1h` flip and reads the two cache-token fields the interface exists to surface.
- 🎭 **A11y Pre-Flight: N/A** (100% backend — no `apps/web/` files touched).
- 🎨 **UX Verification: SKIPPED** — no UI changes in this story.
- **Two caches stayed separate throughout** (the story's headline risk): `subtitle_runs.cache_enabled` is written ONLY from `observeChunk`'s reading of `CompletionUsage` (prompt cache); segment hit/miss counts go to `slog` only. `TestProcessItem_FullCacheHitSkipsTheLLMEntirely` pins the distinction — a 100%-segment-hit item records `cache_enabled=false` because no request was ever sent, and the assertion says so in words.
- **`observeChunk` fires on `attempt == 0` only.** A semantic retry re-sends the same prefix, so observing it again would double-count progress and could flip the verdict on the second read of an entry the first read created.
- **Cleanup writes use `context.WithoutCancel`.** Without it, a shutdown-cancelled item would fail its own `failed`-run and `not_searched` writes and strand the row at `translating` forever — the exact phantom in-flight state the failure policy exists to avoid. Covered by `TestProcessItem_CancellationStillRecordsTheFailure`.
- **`error_message` is bounded to 1000 bytes on a rune boundary.** A wrapped ffmpeg stderr tail (sub-1-4 keeps one) can run long, and provenance is for diagnosis, not archival.
- **Open Questions answered as ruled** (both were "dev proceeds with the stated ruling"): (1) failure → run `failed` + media `not_searched` — **implemented as ruled**, with the reasoning in `failItem`'s doc comment; (2) segment-cache TTL **30d** — implemented, pinned by `TestStoreCachedCues_WritesFinalTextAtTheThirtyDayTTL` so a change is a deliberate edit.
- **Gates:** `go test ./...` exit 0 (3 consecutive runs) · `go test ./internal/subtitle/ -race` clean · `pnpm nx test web` 2457 tests / 225 files · `pnpm nx run api:lint` (vet + staticcheck) clean · `pnpm lint:all` 0 errors (120 pre-existing warnings) · `gofmt` clean on all touched files · `prettier --check` clean · no orphaned test workers.
- **Test count:** +46 new test functions (18 `process_item_test.go`, 12 `segment_cache_test.go`, 9 `show_gate_test.go`, 7 appended to `pipeline_test.go`); package total 242, 0 FAIL.

### Discovery Triage

- **Did this story discover any work outside its current scope?** Yes — one pre-recorded item plus four found at implementation. Each is triaged below (Rule 24: exactly one of ①/②/③, no prose-only findings).
  - **① expand-scope-in-place → the two-caches clarification.** D4's single section carries both the segment cache and the prompt cache; the epic AC inherited the ambiguity. Absorbed as this story's header table + the AC #3/#4 split (segment = content-keyed storage; prompt = provider prefix + `cache_enabled` recording).
  - **① expand-scope-in-place → `ProcessOptions` name collision.** AC #1 names the options type `ProcessOptions`, but that identifier is **already taken in this package** by the search engine's per-item options (`engine.go:125`, constructed at `batch.go:238` — the very call site sub-1-6's flag seam wraps, and `engine_test.go:402`). Renaming the incumbent would rewrite a shipped Epic-8 surface for cosmetic reasons, so the NEW type is **`ProcessItemOptions`** (shape identical: one `Force bool`). Absorbed here; `sub-1-6:113`'s Rule 20 ack line and its sprint-status entry were corrected in the same change so its author does not ack a type that never existed.
  - **① expand-scope-in-place → the progress hook needs a `MediaRef` (sub-task 5.1).** AC #1.6 specifies `progress func(stage PipelineStage, msg string)`, but sub-1-6 AC #6 must broadcast `{media_id, media_type, stage, message}` from **one** Pipeline shared by a 2-worker pool (its Dev Notes: *"one `Pipeline` … constructed once"*). With the literal signature, a concurrent season batch could not attribute an event to an item, making sub-1-6 AC #6 unimplementable without per-item Pipelines. Shipped as `func(ref MediaRef, stage PipelineStage, message string)`. Still a NEW `[@contract-v1]`, so **no Rule 20 bump is owed**; sub-1-6's story + sprint-status entry updated in the same change.
  - **① expand-scope-in-place → `MediaItem.ShowKey` for the D10 key.** AC #5.1 says the gate key is "the series' `MediaRef.ID`", but for an **episode** the ref carries the EPISODE id — keying on it would gate nothing while defeating the exact season-batch case D10 exists for. The `MediaStore` port therefore supplies `ShowKey` (series id for series/episode rows, empty for movies, empty ⇒ bypass). Covered by `TestProcessItem_MoviesBypassTheShowGate` + `TestTranslateTrack_SerializesTheShowsFirstRequest`.
  - **③ backlog-with-carry-forward-link → `backlog-subtitle-status-writer-search-columns`.** AC #6.3 requires `subtitle_search_score` **and** `subtitle_last_searched` to stay unset. Score is satisfied for free (`Score: 0` → `sql.NullFloat64{Valid: score > 0}` → NULL), but **`MovieRepository.UpdateSubtitleStatus:817` and `SeriesRepository.UpdateSubtitleStatus:800` unconditionally stamp `subtitle_last_searched = now`** — so wiring `MediaStore` to them as-is would violate AC #6.3. `EpisodeRepository.UpdateEpisodeSubtitleStatus:273` is already clean (it writes only status/path/language). The adapter is explicitly sub-1-6's scope ("wired by 1.6, faked here"), so the constraint is recorded in the `MediaStore` port's doc comment AND filed as a tracked entry, bidirectionally linked to sub-1-6.
- Reference: `project-context.md` Rule 24.

### File List

| File | Change |
|---|---|
| `apps/api/internal/subtitle/pipeline.go` | **modified** — `[@contract-v1]` `MediaRef` / `ProcessItemOptions` / `ProcessOutcome`; the five narrow ports (`RunStore`, `MediaStore`+`MediaItem`, `SubtitlePlacer`, `TrackRouter`, and `SegmentCache` from `segment_cache.go`); `PipelineOption` + 8 `With*` options on a variadic `NewPipeline`; `processScope` (context-carried) + `observeChunk` + `emitProgress`; `ExpectedSidecarPath` / `acceptableSidecar` / `preflightSkip` / `resumeReason`; `buildSystemBlocks` TTL flip; chunk loop now numbers chunks, enters the D10 gate on its first request, and observes once per chunk |
| `apps/api/internal/subtitle/process_item.go` | **new** — `[@contract-v1]` `ProcessItem` (pre-flight → run row → route → deliverable → P9-ordered place/provenance/status), `deliverable`, `translateWithCache`, `recordSkip`, `failItem`, `truncateErrorMessage`, `setMediaStatus`, `sourceLanguageOf`, `requireItemPorts` |
| `apps/api/internal/subtitle/segment_cache.go` | **new** — `SegmentCache` port + `segmentCacheRepository` adapter over `CacheRepositoryInterface`; `MetadataHash` (canonical `\x1f`-joined, set fields sorted on a copy), `segmentKey` (`subseg:v1:` + sha256(cue + version tuple)), `Pipeline.runVersion`, `splitCachedCues` / `storeCachedCues` / `mergeCues`; `segmentCacheType` / `segmentCacheTTL` (30d) |
| `apps/api/internal/subtitle/show_gate.go` | **new** — D10 `showGate` keyed latch (leader/waiter/warm states, 50-min warm window, `showEntryTTL` pruning per Rule 14, ctx-cancellable, idempotent release) + `Pipeline.enterShowGate` |
| `apps/api/internal/subtitle/pipeline_test.go` | **modified** — +7 tests (pre-flight P5 table + resume refinement, system-block TTL policy, first-chunk cache verdict table, no-scope compatibility); `fakeRunStore` / `testVersion` / `newMediaFile` / `oneCueSRT` helpers; `…AndUncached` renamed to `…AndCacheBreakpointed` and re-pointed at the new policy |
| `apps/api/internal/subtitle/process_item_test.go` | **new** — 18 tests: 5-row verdict matrix, version-tuple provenance, P9 order proof, pre-flight-spends-nothing, force semantics, cache split/full-hit, 3-row failure policy, bounded error message, cancellation, progress cadence, wiring guards, and the real-`:memory:`-SQLite integration test |
| `apps/api/internal/subtitle/segment_cache_test.go` | **new** — 12 tests: `MetadataHash` canonicalization (8 sub-cases incl. non-mutation + boundary forging), segment key versioning (4 RunVersion sub-cases + boundary forging), split/merge/store behaviour, repository adapter round-trip, merge-desync guard; `newMigratedTestDB` helper |
| `apps/api/internal/subtitle/show_gate_test.go` | **new** — 9 tests: first-runs-alone, warm-window skip, stale re-warm, movie bypass, cancellation, Rule 14 pruning, idempotent release, cross-show independence, and the TranslateTrack wiring proof |
| `_bmad-output/implementation-artifacts/sub-1-6-wire-triggering-gating.md` | **modified** — Rule 24 lane ① carry-forward: Dev Notes ack line corrected to `ProcessItemOptions`, progress-hook signature and the `MediaStore` search-column constraint recorded |
| `_bmad-output/implementation-artifacts/sprint-status.yaml` | **modified** — this story → `review`; `backlog-subtitle-status-writer-search-columns` filed; sub-1-6 entry annotated |

### Change Log

| Date | Change |
|---|---|
| 2026-07-28 | **Task 1 (AC #2) — RED first** (`undefined: ExpectedSidecarPath`). P5 spelled out as a three-clause predicate — exists AND `ParseSRT` succeeds AND cue count > 0 — behind `ExpectedSidecarPath`, which runs the SAME `NormalizeLanguageTag` + `BuildSubtitleFilename` pair `placer.Place` uses, so the pre-flight can never check a different path from the one delivery writes. `preflightSkip` treats the sidecar as the GATE and `FindCompletedRun` as a LOG refinement only, which is what makes "delete the sidecar and re-trigger" an inherently valid re-run and what keeps a repository hiccup from forcing a full re-translation (Rule 13 case-3, justified in place). 6-row predicate table incl. the zero-byte and unparseable rows the existence-only anti-pattern would have accepted. |
| 2026-07-28 | **Task 2 (AC #3) — RED first** (`undefined: MetadataHash`). `MetadataHash` = sha256 over a fixed 7-field `\x1f`-joined serialization; genres and countries sorted on a COPY (TMDb ordering jitter must not split the cache; mutating the caller's slice would silently reorder what the prompt renders), cast left in billing order, glossary excluded because it owns its own `RunVersion` field. `segmentKey` = `subseg:v1:` + sha256(cue text + `\x00` + the four version fields) — content-hashed, never index-keyed (P1: SDH filtering leaves gaps, so the same line can carry different indexes across runs). `SegmentCache` is a 2-method port; `segmentCacheRepository` adapts `CacheRepositoryInterface` with `cacheType="subtitle_segment"` and a 30d TTL. Read failures degrade to a miss, write failures are logged and swallowed — a cache costs tokens, never correctness. |
| 2026-07-28 | **Task 3 (AC #4) — RED first** (`undefined: processScope`). The TTL flip sub-1-5a deferred: the LAST stable system block now carries `CacheTTL1h` (one breakpoint caches `[0]+[1]` together — prefix semantics; 1h because a season batch outlives the 5m entry). `cache_enabled` is recorded by DETECTION from the FIRST chunk's `CompletionUsage` — both cache fields zero ⇒ the prefix silently failed to cache ⇒ `false`; no tokenizer heuristics, no prefix padding (D4's ban). The per-item state rides the **context** (`processScope`) rather than a parameter, because `TranslateTrack`'s signature is sub-1-5a's stamped surface and one Pipeline serves the whole worker pool; absent scope = no-op, so sub-1-5a's direct callers are unchanged. sub-1-5a's `…AndUncached` lock-in test moved with the policy instead of being deleted. |
| 2026-07-28 | **Task 4 (AC #5) — RED first** (`undefined: newShowGate`). `showGate`: mutex + `map[showKey]{inFlight, warmedAt, touchedAt}`. First requester for a show leads and holds the latch across its first LLM request; waiters park on the channel and **re-evaluate** on release (so a window that went stale mid-wait promotes the next waiter to leader rather than letting everyone through cold). Warm window 50 min < the 1h TTL; stale entries pruned on access at `2×` the window (Rule 14); `ctx` cancellation unparks a waiter with `ctx.Err()`; release is `sync.Once`-idempotent. Empty key = movie = full bypass. Wired at `start == 0 && attempt == 0` only. Falsification-verified: forcing `enterShowGate` to no-op fails the wiring test. |
| 2026-07-28 | **Task 5 (AC #1, #6)** — `ProcessItem` `[@contract-v1]`: pre-flight (before any row is written, so a re-scan appends no provenance) → `pending`→`running` as two writes (a killed process leaves an honest `running`) → media `extracting` → `SelectAndRoute` → per-verdict deliverable → **place → provenance → media terminal, in that order (P9)**, proven by a shared order log rather than by inspection. Translate path splits against the segment cache, sends only the misses as a reduced `ExtractedTrack`, writes back the FINAL post-OpenCC text, merges by cue `Index`, and re-checks the FR17 invariant over the FULL set (`TranslateTrack` only ever sees the subset). `RouteConvertThenDeliver` fails the item on a converter error — unlike the translate path, conversion IS the deliverable there and shipping Simplified text under a `.zh-Hant.srt` name would be a lie. `failItem` writes the run `failed` + media `not_searched` through a `context.WithoutCancel` so a shutdown still records why. |
| 2026-07-28 | **Rule 24 carry-forward** — `sub-1-6-wire-triggering-gating.md` Dev Notes corrected in place (`ProcessOptions` → `ProcessItemOptions`, progress-hook signature, `MediaStore` search-column constraint); `backlog-subtitle-status-writer-search-columns` filed in `sprint-status.yaml` with bidirectional links. |

---

## Open Questions for Alexyu (non-blocking — dev proceeds with the stated rulings)

1. **Failure policy (AC #1.5):** run `failed` + media row back to **`not_searched`** (retryable; run row is the audit trail). Alternative: a dedicated `failed` media status — but that's a `[@contract-v1→v2]` bump on sub-1-2's enum plus FE badge work. Confirm the revert-to-`not_searched` reading.
   - **IMPLEMENTED AS RULED (2026-07-28).** Still cheap to reverse before sub-1-6 ships; after that the FE badge work makes it a real change.
2. **Segment-cache TTL 30d** (AI-parsing precedent). Cheap to change; say now if you want permanent.
   - **IMPLEMENTED AS RULED (2026-07-28)** — one constant (`segmentCacheTTL`), pinned by a test so a change is deliberate.
3. **NEW — raised at implementation:** `MovieRepository`/`SeriesRepository.UpdateSubtitleStatus` unconditionally stamp `subtitle_last_searched`, which AC #6.3 says must stay unset for a generated subtitle. Filed as `backlog-subtitle-status-writer-search-columns` for sub-1-6 (which owns the adapter). The cheap fix is a narrow `UpdateSubtitleGenerationStatus` repo method that writes only status/path/language — the shape `EpisodeRepository` already has.
