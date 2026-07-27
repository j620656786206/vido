# Story sub-1.5b: Cache, per-show serialization, and delivery

Status: ready-for-dev

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

- [ ] **Task 1 — Pre-flight + P5 (AC #2):** predicate helper (reusing `ParseSRT` + placer naming), force/resume semantics + tests (incl. the truncated-file case).
- [ ] **Task 2 — Segment cache + RunVersion assembly (AC #3):** canonical `MetadataHash`, key builder, `SegmentCache` iface, hit/miss split + merge; tests.
- [ ] **Task 3 — Prompt-cache policy (AC #4):** TTL flip on block `[1]`, first-chunk detection, `cache_enabled` recording; tests.
- [ ] **Task 4 — D10 gate (AC #5):** keyed latch + warm window + pruning + cancellation; concurrency tests.
- [ ] **Task 5 — ProcessItem + delivery (AC #1, #6):** verdict handling, status/run lifecycle, P9-ordered delivery, progress hook, failure policy; the AC #7 seam tests + integration test; record the four Rule 20 acks in Dev Notes; full gates.

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

### Debug Log References

### Completion Notes List

### Discovery Triage

- **Did this story discover any work outside its current scope?**
  - If **NO** beyond the pre-recorded item: state `N/A — no further out-of-scope work discovered`.
  - **① expand-scope-in-place → the two-caches clarification.** D4's single section carries both the segment cache and the prompt cache; the epic AC inherited the ambiguity. Absorbed as this story's header table + the AC #3/#4 split (segment = content-keyed storage; prompt = provider prefix + `cache_enabled` recording).
- Reference: `project-context.md` Rule 24.

### File List

---

## Open Questions for Alexyu (non-blocking — dev proceeds with the stated rulings)

1. **Failure policy (AC #1.5):** run `failed` + media row back to **`not_searched`** (retryable; run row is the audit trail). Alternative: a dedicated `failed` media status — but that's a `[@contract-v1→v2]` bump on sub-1-2's enum plus FE badge work. Confirm the revert-to-`not_searched` reading.
2. **Segment-cache TTL 30d** (AI-parsing precedent). Cheap to change; say now if you want permanent.
