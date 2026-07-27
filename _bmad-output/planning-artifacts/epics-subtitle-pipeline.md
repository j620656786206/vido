---
stepsCompleted: [1, 2, 3, 4]
workflowStatus: complete
completedDate: '2026-07-26'
inputDocuments:
  - _bmad-output/planning-artifacts/prd.md
  - _bmad-output/planning-artifacts/subtitle-pipeline-architecture.md
  - ux-design.pen (flow F v2, generation-centric)
scope: 'M1 subtitle pipeline (extraction + LLM translation to Traditional Chinese). Stories are drafted M1-first; Epic 2 (M1.5) is defined here and unblocks via Story 1.7a. P2/Tier-2 requirements are inventoried for traceability only. (Reworded 2026-07-27, IR-r2 F10.)'
ceremony: 'Risk-tiered (party-mode consensus 2026-07-26): high-risk stories get full ACs; low-risk mechanical stories get lean quick-spec ACs; the three "nails" below are citable ACs regardless of format.'
---

# Vido Subtitle Pipeline — Epic Breakdown

## Overview

This document decomposes the **M1 subtitle pipeline** (embedded-subtitle extraction + LLM translation to Traditional Chinese) from `prd.md` and `subtitle-pipeline-architecture.md` into implementable stories. Brownfield extension of `apps/api/internal/subtitle/`. Bound by `project-context.md` (the bible).

**Ceremony model (risk-tiered):** high-risk stories (`claude.go` SDK migration; orchestrator + quality gate) carry full acceptance criteria; low-risk mechanical stories (error codes, SSE stage, docs) carry lean ACs. Three hazards MUST appear as citable ACs regardless of format:

1. All **26** existing Claude-touching tests stay green — 24 in `claude_test.go` + 2 in `retry_test.go` (the D8 retry guards; count verified via `go test -list` 2026-07-27) — the media-scanning regression guard.
2. SDK retries disabled — `retryTransient` + Governor budget gate preserved (D8).
3. Quality gate runs **before** OpenCC (P4) — verified by a test-first Simplified-leak-triggers-retry unit test.

## Requirements Inventory

### Functional Requirements

_Phase tags from the PRD: unmarked = M1; [M1.5], [P2] = Growth; [Tier-2] = Expansion._

**A. Subtitle Source Detection & Extraction**
- FR1: Detect subtitle tracks (embedded text, embedded image, external sidecar) and their language. _(M1 — reuses `ffprobe_service.go` + `subtitle_tracks`)_
- FR2: Extract an embedded text subtitle track without re-encoding. _(M1)_
- FR3: Extract multiple text subtitle tracks in a single pass. _(M1)_
- FR4: Filter SDH annotations from an extracted subtitle. _(M1)_
- FR5: Identify a media item as having no usable text source and mark it. _(M1 — new `subtitle_status` value)_

**B. Language Routing & Conversion**
- FR6: Determine whether a Chinese subtitle is Traditional or Simplified **from content** (not filename or track tag); **source language** from the track tag per FR1. _(M1 — revised 2026-07-26)_
- FR7: Pass an already-Traditional-Chinese subtitle through unchanged. _(M1)_
- FR8: Convert Simplified → Traditional (Taiwan) via OpenCC, no AI. _(M1)_
- FR9: Route by detected language (Traditional → done, Simplified → convert, English → translate, other → skip). _(M1)_

**C. AI Translation**
- FR10: Translate an English subtitle to Traditional Chinese via a translation provider. _(M1)_
- FR11: Translate while preserving cue numbering and timestamps exactly. _(M1)_
- FR12: User can trigger subtitle translation on demand. _(M1 — F2 生成字幕 button → new `POST /api/v1/subtitles/pipeline/run`)_
- FR13: Translate automatically when new media is added. _(M1 — `ScannerService` enqueue)_
- FR14: Show an estimated cost before translating a batch/season. _[P2]_

**D. Translation Quality Assurance**
- FR15: Guarantee Traditional script via a deterministic conversion pass (OpenCC). _(M1)_
- FR16: Detect and retry only the cues that come back empty, untranslated, or wrong-language. _(M1)_
- FR17: Verify translated cue timestamps match the source. _(M1)_

**E. Glossary & Cross-Episode Consistency** _[P2]_
- FR18: Harvest term mappings into the show glossary as unconfirmed suggestions. _[P2]_
- FR19: Review, confirm, edit, delete glossary terms. _[P2]_
- FR20: Apply a show's glossary as context for any episode, order-independent. _[P2]_

**F. Provider & Key Management**
- FR21: User supplies their own API keys for translation and (optional) ASR. _(M1 — env-var)_
- FR22: Translate with a locally-hosted no-key provider. _[Tier-2]_
- FR23: Gracefully disable translation with a clear message when no key is configured. _(M1 — F5 capability gate)_
- FR24: Select active translation/ASR provider and model. _[Tier-2]_
- FR25: Configure/edit provider keys in-app after setup. _[M1.5]_

**G. Metadata Context & Match Correction**
- FR26: Supply media metadata as translation context. _(M1 — existing TMDb service)_
- FR27: Correct an incorrect TMDb match. _(M1 — existing `ManualSearchDialog`)_
- FR28: Re-translate using corrected metadata after a match is fixed. _[P2]_

**H. Source Fallback** _[P2]_
- FR29: Generate subtitles from audio via ASR when no text source exists. _[P2]_
- FR30: Prioritize sources extract > ASR > online-search. _[P2]_
- FR31: Route ASR to cloud / external worker / local by compute. _[P2]_

**I. Pipeline Operation & Status**
- FR32: Place the produced Traditional-Chinese subtitle beside the media file. _(M1 — `placer.go`)_
- FR33: Show real-time progress status. _(M1 — SSE)_
- FR34: Run batch processing over a scope (missing/season/library). _[P2]_

**M1 functional scope: ~22 FRs** (FR1–13, 15–17, 21, 23, 26, 27, 32, 33).

### NonFunctional Requirements

- NFR-P1: On DS920+ (J4125/4 GB), one item (extract + translate) must not degrade other NAS services. _(threshold TBD at pilot — G2)_
- NFR-P2: M1 does no heavy local AI compute (no local ASR).
- NFR-P3: Concurrency bounded per hardware tier (ARM/low-RAM → 1). _(M1: fixed concurrency 2 on DS920+; tiering → P2)_
- NFR-S1: Provider keys encrypted at rest; never logged (slog sanitize).
- NFR-S2: No dialogue egress without a configured cloud key; local path available (Tier-2).
- NFR-S3: Key-config page requires HTTPS, or warns + requires confirmation over plain HTTP. _(added by architecture; binds M1.5)_
- NFR-R1: Every external failure fail-soft — degrade per item, never fail-page.
- NFR-R2: Idempotent — no duplicate/corrupt output; existing acceptable `.zh-Hant.srt` not overwritten without intent.
- NFR-R3: Granular recovery — per-cue retry; batch preserves completed results on cancel and is resumable (via provenance).
- NFR-I1: External providers integrated per Rule 27 Five Pillars.
- NFR-I2: Ships as a single multi-arch (amd64 + arm64) Docker image with ffmpeg bundled.
- NFR-I3: Output uses `.zh-Hant.srt` / `.zh-Hans.srt` sidecar convention.

### Additional Requirements

**From Architecture (technical constraints that shape stories):**
- **No scaffolding story** — brownfield. First story is the `claude.go` SDK migration, not project init.
- **`anthropic-sdk-go` pinned at v1.59.0**; SDK retries **disabled** (D8) — Governor budget + `retryTransient` preserved.
- **Regression guard:** all **26** existing Claude-touching tests green (24 `claude_test.go` + 2 `retry_test.go`) — `claude.go` also backs AI filename parsing used by **media scanning**, so the blast radius exceeds subtitles.
- **Rule 19:** orchestrator lives in `internal/subtitle/pipeline.go` (never `services/`; compile-time enforced by `boundaries_test.go`).
- **Migration 030:** provenance table + `subtitle_status` enum extension (`probing`/`extracting`/`translating`/`no_text_source`/`skipped`). The enum is a **wire contract** → Rule 20 `[@contract-vN]` stamp.
- **Error codes:** extend `SUBTITLE_` (no new Rule 7 prefix); sync `code-review/instructions.xml` Step 3.
- **SSE stages:** add `probing`/`extracting`/`translating`/`skipped`; wire contract → stamp + update `docs/sse-event-types.md` (+ zh-TW, Rule 17).
- **`placer.go` is the sole writer**; overwrite guard = existing acceptable `.zh-Hant.srt` (parses, cue count > 0); single `force` exception.
- **Quality gate before OpenCC** (P4) — test-first.
- **Versioned cache key** (metadata + glossary + prompt + model); disable caching + record when the prefix < 4096 tokens.
- **Per-show first-request serialization** at the orchestrator (avoids 3–5× cache writes).
- **Feature flag** (`legacy | pipeline`) gates exactly `batch.go:244`.
- **FR12 endpoint** `POST /api/v1/subtitles/pipeline/run`; **FR13** via `ScannerService` enqueue; **FR23** capability gate at the top of `pipeline.go`, shared by both entry points.
- **ffmpeg/ffprobe must be bundled in the Docker image** (silent degradation if absent).
- **Bilingual docs** (Rule 17) for any user-facing doc changes.

**From UX (flow F v2, generation-centric):**
- F2-D-v2 `btn-generate` (生成字幕) is the FR12 trigger surface.
- F5-D-v2 is the FR23 capability gate (尚未設定 fail-soft).
- F3-D-v2 is the FR33 generation-progress surface (SSE-driven).
- **Three `.pen` copy revisions block only the M1.5 UI story**, not M1 backend: F2 "轉錄"→"抽取內嵌字幕", F5 "前往設定" M1 behaviour, F5 FFmpeg framing (deployment concern, not a user setting).

### FR Coverage Map

| FR | Epic | Note |
|----|------|------|
| FR1 | Epic 1 | Detect subtitle tracks + language (reuse `ffprobe_service.go`) |
| FR2 | Epic 1 | Extract embedded text track, no re-encode |
| FR3 | Epic 1 | Multi-track single-pass extraction |
| FR4 | Epic 1 | SDH filtering (pre-translation) |
| FR5 | Epic 1 | No-text-source state — enum (1.2) · written by the router (1.4) · **user-visible mark: badge/icon (1.7a spec → 1.7b)** |
| FR6 | Epic 1 | Content-based CJK-variant detection |
| FR7 | Epic 1 | Traditional pass-through |
| FR8 | Epic 1 | Simplified → OpenCC s2twp |
| FR9 | Epic 1 | Language routing (1.4) · **`skipped` user-visible mark (1.7a spec → 1.7b)** |
| FR10 | Epic 1 | English → Traditional via provider |
| FR11 | Epic 1 | Preserve cue numbering + timestamps |
| FR12 | Epic 1 | Manual trigger (`POST /subtitles/pipeline/run`) |
| FR13 | Epic 1 | Auto on media-add (Scanner enqueue) |
| FR15 | Epic 1 | Deterministic Traditional guarantee (OpenCC final pass) |
| FR16 | Epic 1 | Per-cue retry |
| FR17 | Epic 1 | Timestamp-equality assertion |
| FR21 | Epic 1 | BYO key (env-var in M1) — **already shipped**: `CLAUDE_API_KEY` at `apps/api/internal/config/config.go:122`; zero new M1 work |
| FR23 | Epic 1 | Capability gate, no silent failure |
| FR26 | Epic 1 | Metadata context to provider |
| FR27 | Epic 1 | TMDb match correction — **already shipped**: Story 3.7 `ManualSearchDialog` + `POST /api/v1/metadata/manual-search`; zero new M1 work |
| FR32 | Epic 1 | Place sidecar beside media |
| FR33 | Epic 1 | Real-time progress (SSE) |
| FR25 | Epic 2 | In-app key config (+ NFR-S3) |
| FR14 | — | Deferred [P2] — cost estimate UI |
| FR18–20 | — | Deferred [P2] — glossary auto-harvest |
| FR22, FR24 | — | Deferred [Tier-2] — local no-key provider / model selection |
| FR28 | — | Deferred [P2] — re-translate after match correction |
| FR29–31 | — | Deferred [P2] — ASR fallback + provider routing |
| FR34 | — | Deferred [P2] — batch scope processing |

All 34 FRs accounted for: 22 → Epic 1, 1 → Epic 2, 11 → deferred (P2/Tier-2).

## Epic List

### Epic 1: Automatic Traditional-Chinese subtitles for English media (M1)

A NAS owner drops English-content media into their library and a correctly-timed `.zh-Hant.srt` appears beside the file — extracted from the embedded English track, translated by Claude, guaranteed Traditional by OpenCC — with real-time progress and an on-demand trigger, all within the DS920+ resource envelope. Standalone and shippable using an env-var API key; does not require Epic 2.

**FRs covered:** FR1–13, FR15–17, FR21, FR23, FR26, FR27, FR32, FR33

### Epic 2: In-app provider-key configuration (M1.5)

The owner configures and edits TMDB / Claude / (optional) ASR keys from a settings page instead of env-vars, and the `ManageSubtitleDialogV2` "前往設定" dead loop is fixed. Builds on Epic 1 (replaces the env-var key with a UI); Epic 1 does not depend on it.

**FRs covered:** FR25 (+ NFR-S3)
**Blocked-by:** three `.pen` copy revisions (F2 轉錄→抽取, F5 前往設定 M1 behaviour, F5 FFmpeg framing) — not addressed by PR #177. **Resolved by Story 1.7a** (2026-07-27 ruling — batched into its Pencil session).

<!-- Detailed per-epic stories are appended in Step 3. -->

## Epic 1: Automatic Traditional-Chinese subtitles for English media (M1)

A NAS owner drops English-content media into their library and a correctly-timed `.zh-Hant.srt` appears beside the file, automatically and on-demand, within the DS920+ envelope.

Stories follow the architecture's 7-step implementation sequence with risk-tiered acceptance criteria. Dependency order is strictly backward (no story depends on a later one).

### Story 1.1: Migrate the Claude client to the official Go SDK 🔴

As a maintainer,
I want `internal/ai/claude.go` re-implemented on `anthropic-sdk-go`,
So that the translation path gains prompt caching and caller-visible usage without hand-rolling the wire protocol.

**Acceptance Criteria:**

**Given** the re-implemented client, **When** any existing consumer (`Provider.Parse`, `TextCompleter.CompleteText`) calls it, **Then** behavior is unchanged **And** all **26** existing Claude-touching tests pass (24 in `claude_test.go` + 2 in `retry_test.go` — the D8 retry guards) — the media-scanning regression guard (NAIL 1).

**Given** the SDK's built-in retries, **When** the client is constructed, **Then** SDK retries are disabled **And** the existing `retryTransient` + Governor budget gate (`governed(...)`, 9R-11) remain the sole retry/throttle path (NAIL 2).

**Given** an upstream error, **When** it surfaces, **Then** the 404 "set `CLAUDE_MODEL` to a current model" diagnostic is preserved **And** 429→`ErrAIQuotaExceeded`, timeout→`ErrAITimeout`, other→`ErrAIProviderError` mapping is preserved **And** `BudgetFromContext().RecordLLM(...)` still fires.

**Given** a translation-oriented call, **When** it runs, **Then** `system` content blocks with `cache_control` are supported **And** usage (input/output/cache tokens) is returned to the caller. `anthropic-sdk-go` pinned at **v1.59.0**.

### Story 1.2: Pipeline state model — provenance table + status enum 🟡

As the pipeline,
I want a provenance table and extended `subtitle_status` values,
So that runs are tracked, resumable, and a no-text-source item is representable.

**Acceptance Criteria:**

**Given** migration **030** (Go migration, after 029), **When** it runs, **Then** a `subtitle_runs` provenance table exists (plural per Rule 6 — sub-1-2 AC #1) (tmdb_id, metadata-snapshot hash, glossary version, prompt version, model id, per-run status) **And** `subtitle_status` accepts `probing` / `extracting` / `translating` / `no_text_source` / `skipped`.

**Given** `subtitle_status` is frontend-consumed (`json:"subtitle_status"`, a URL search param), **When** the enum is extended, **Then** it carries a `[@contract-vN]` stamp (Rule 20).

**Given** a re-run, **When** a completed item/cue has a provenance record, **Then** it can be skipped (NFR-R3 resume basis).

### Story 1.3: Error codes, SSE stages, and bilingual docs 🟢

As a developer and a frontend consumer,
I want new `SUBTITLE_` error codes and SSE stages registered,
So that pipeline failures and progress are observable through the established contracts.

**Acceptance Criteria:**

**Given** the `SUBTITLE_` prefix, **When** `SUBTITLE_EXTRACT_FAILED` / `SUBTITLE_NO_TEXT_SOURCE` / `SUBTITLE_TRANSLATE_FAILED` / `SUBTITLE_TIMESTAMP_MISMATCH` are added, **Then** the Rule 7 list in `project-context.md` **And** `code-review/instructions.xml` Step 3 are synced (prefix count stays 16).

**Given** `subtitle_progress.stage`, **When** `probing` / `extracting` / `translating` / `skipped` are added, **Then** `docs/sse-event-types.md` and `docs/sse-event-types.zh-TW.md` are both updated (Rule 17) **And** the stage set carries a Rule 20 stamp.

### Story 1.4: Extract, SDH-filter, and route embedded subtitles 🔴

As a NAS owner,
I want embedded text subtitles detected, extracted, SDH-filtered, and routed by language,
So that Traditional/Simplified embedded subs are delivered directly and English is queued for translation.

**Acceptance Criteria:**

**Given** a media file, **When** probed, **Then** tracks + language tags are read from `subtitle_tracks` (FR1) **And** embedded text tracks are extracted via `ffmpeg -map 0:s -c copy` with no re-encode, multiple tracks in one pass (FR2/3).

**Given** an extracted subtitle, **When** SDH annotations are present, **Then** they are filtered **before** translation **And** remaining cues keep their original numbering (P6/P7, FR4).

**Given** routing, **When** the track language tag is `eng`/`en`, **Then** route to translate; Traditional content → done (FR7); Simplified → OpenCC s2twp → done (FR8); `und` or a non-English tag → `skipped` / `no_text_source` (FR5/9). `und` is **never** treated as English (P0).

**Given** CJK-variant determination, **When** deciding Traditional vs Simplified, **Then** it is decided from **content** via `detector.go`, not the track tag (FR6).

### Story 1.5a: Translate core + quality gate 🔴 (test-first)

_Size-split 2026-07-27 (IR-r2 finding M1, Alexyu ruling) per the 13-4a/13-4b precedent — 1.5 was BACKEND-ONLY but LARGE (five concerns in one 🔴 story). 1.5a = the translate stage itself; 1.5b = cache/serialization/delivery. **NAIL 3 lives here.**_

As a NAS owner,
I want English subtitles translated to Traditional Chinese with guaranteed script and preserved timing,
So that the translated cues are trustworthy without hand-editing.

**Acceptance Criteria:**

**Given** the quality gate, **When** LLM output has Simplified leakage / empty / echoed / cue-count mismatch, **Then** only the affected cues retry (FR16) — proven by a **test-first** unit test that feeds deliberately-Simplified output and asserts the retry fired **And** the gate runs **before** OpenCC (P4, NAIL 3).

**Given** a translation request, **When** cues are sent, **Then** the LLM receives numbered text only; timestamps stay in Go and are re-stitched; output timestamps assert-equal the source cue-by-cue (FR11/17, P2).

**Given** the final pass, **When** translation completes, **Then** OpenCC s2twp guarantees Traditional script (FR15) **And** metadata (FR26) is injected as prompt context **And** the prompt text + its version constant live together in `internal/ai/prompts/` (P11). Consumes sub-1-1 `[@contract-v1]` `CachingCompleter`.

### Story 1.5b: Cache, per-show serialization, and delivery 🔴

_Second half of the 1.5 split. Depends on: 1.5a (gate-passed translated cues) + 1.2 (`RunVersion` + provenance repo)._

As a NAS owner,
I want translations cached, serialized per show, and delivered atomically with provenance,
So that re-runs are cheap, cache writes are not duplicated, and the `.zh-Hant.srt` appears beside the file exactly once.

**Acceptance Criteria:**

**Given** caching, **When** translating, **Then** the cache key = `hash(cue) + RunVersion` (metadata + glossary + prompt + model versions — 1.2's `[@contract-v1]`) **And** if the stable prefix < 4096 tokens, caching is disabled **and recorded** (`subtitle_runs.cache_enabled`, D4).

**Given** per-show concurrency, **When** several items of one show translate, **Then** the first request runs alone before the rest are released (D10).

**Given** delivery, **When** an item completes, **Then** the sidecar is written by `placer.go` only, not overwriting an acceptable existing `.zh-Hant.srt` (parses + cue count > 0) except with `force` (D3/P5, FR32) **And** provenance is written AFTER the successful place (P9) **And** a completed run with a matching `RunVersion` is skipped on re-run (NFR-R3 resume via `FindCompletedRun`).

### Story 1.6: Wire triggering, gating, and progress 🟡

As a NAS owner,
I want the pipeline triggered automatically on media-add and on demand, gated when unconfigured, with live progress,
So that subtitles appear without manual steps and failures are always visible.

**Acceptance Criteria:**

**Given** the feature flag, **When** set to `pipeline`, **Then** `batch.go:244` calls the orchestrator; `legacy` calls `Engine.Process` — one seam only (D5).

**Given** new media, **When** a scan completes, **Then** `ScannerService` enqueues eligible items into the worker pool (fixed concurrency **2** on M1) which calls the orchestrator (FR13, NFR-P3).

**Given** a user action, **When** `POST /api/v1/subtitles/pipeline/run` is called (single item, optional `force`), **Then** the orchestrator processes that item (FR12).

**Given** no configured key, **When** the pipeline is invoked from either entry point, **Then** the capability gate at the top of `pipeline.go` returns a clear message with no silent failure (FR23).

**Given** a run, **When** stages progress, **Then** SSE broadcasts once per chunk (FR33/P8) **And** each item degrades fail-soft, never fail-page (NFR-R1).

**Given** the M1 pilot's deferred G2/G4 thresholds (IR 2026-07-27 F4 — Alexyu ruling: they live HERE), **When** this story is drafted, **Then** it fixes the measurable bars as citable ACs: (a) an NFR-P1 resource bound while one item processes on DS920+, (b) a concrete time-to-`.zh-Hant.srt` number for an embedded-English-sub item (the PRD's "tens of seconds", made testable), (c) the "usable without hand-editing ≥ X %" trust bar — X set with Alexyu at drafting time and recorded in the story.

**Given** Rule 17 (IR-r2 F9), **Then** `docs/deployment.md` + `docs/deployment.zh-TW.md` document the bundled ffmpeg/ffprobe requirement and the multi-arch image (infrastructure already shipped — `Dockerfile:47`, `docker.yml:80`; only the docs half was unowned). The NFR-S3 half belongs to Story 2.1.

### Story 1.7a: Subtitle-status badge — design spec + Epic 2 `.pen` copy unblock 🟢 (UX)

_Added 2026-07-27 by Alexyu ruling ("前端 badge 讓 M1 出貨時就帶 badge") — promotes the lane-③ entry `backlog-subtitle-status-fe-rendering` into M1 scope. Discovered during sub-1-2: extending `subtitle_status` 4→9 values leaves the frontend rendering none of them (fail-soft — the badge is simply absent — but a UX gap once the pipeline is live). **Not** covered by Story 1.6, which owns the SSE *progress* surface (F3), not library/detail badge rendering._

As Sally (UX),
I want a standalone spec screen defining the badge label, tint, and icon for the 5 new `subtitle_status` values,
So that the implementation has something concrete to verify against instead of a developer guessing zh-TW copy and colour tokens.

**Acceptance Criteria:**

**Given** the Pencil spec-screen convention (design decisions get their **own** standalone screen), **When** the design lands, **Then** `ux-design.pen` carries a new desktop screen exported to `flow-j-specs/j2-d.png` (sibling to the PosterCard info-density spec `j1-d.png`) **And** `scripts/export-pen-screenshots.py`'s `SCREENS` dict is extended **And** only that one PNG is staged (a full regen is non-deterministic).

**Given** the governing principle in `libraryStatus.ts` — transient process states belong to the Activity hub, **not** the badge, because they have "no persisted per-item field" — **When** Story 1.2 makes `probing`/`extracting`/`translating` persisted column values, **Then** the spec explicitly rules whether the poster/list badge now shows in-flight states or stays terminal-only, with rationale.

**Given** two rendering surfaces, **When** the spec is authored, **Then** it specifies per new value both the poster/list badge (label + tint from the existing six tokens — **no new colour token**) and the `EpisodeList` row icon (icon + colour + spin), respecting the standing ruling that **accent is reserved for in-progress states**.

**Given** the existing badge vocabulary, **Then** the spec resolves two copy collisions: `no_text_source` vs the existing `缺字幕` (different recoveries — P2 ASR vs re-search), and making `skipped` read as *deliberate* rather than broken (it is the correct P0 outcome for an `und` or non-English track tag).

**Given** this story already opens `ux-design.pen`, **When** the same Pencil session runs, **Then** the **three copy revisions that block Epic 2** are closed here too (Alexyu ruling 2026-07-27, party-mode — one `.pen` edit, one screenshot regeneration, one commit; splitting them pays the non-deterministic re-render risk twice on the same file): F2-D-v2 轉錄 → 抽取內嵌字幕 (M1 extracts an existing track; ASR is P2) · F5-D-v2 前往設定 must state M1 behaviour (no settings page exists until FR25 — the link is a dead loop today) · F5-D-v2 FFmpeg reframed as a **deployment** concern, not a user setting. Copy/framing only — no layout rework, no new screens. **This makes Story 1.7a the resolver of Epic 2's blocked-by.**

### Story 1.7b: Subtitle-status badge — frontend 🟡

As a NAS owner,
I want the library, list, and episode rows to show what the pipeline actually did to each file,
So that an item the pipeline permanently declined is visibly distinguishable from one it simply hasn't reached yet.

**Blocked-by:** 1.7a (the spec screen is the verification target) and 1.2 (the `[@contract-v1]` 9-value enum).

**Acceptance Criteria:**

**Given** `deriveSubtitleStatus`, **When** it receives a terminal pipeline value (`no_text_source` / `skipped`), **Then** it returns the spec'd badge **And** that branch is evaluated **before** embedded-track inference — otherwise a file carrying image-only or `und` tracks infers 有字幕 and the pipeline's authoritative verdict is silently overridden. The three transient values return `null` without reaching track inference either.

**Given** `EpisodeList`'s status-icon map, **When** it is extended, **Then** all 9 values have an entry with a long-form `aria-label` (the icon carries no visible text, so the accessible name is where the full explanation lives).

**Given** Rule 16, **Then** tests cover all 9 values on both surfaces **And** include the two ordering-regression cases (`no_text_source` / `skipped` **with** non-empty `subtitleTracks`) that fail against a naive append-at-the-bottom implementation.

**Given** the mandatory UX gate, **Then** each new state is rendered and compared side by side with `flow-j-specs/j2-d.png` before completion.

## Epic 2: In-app provider-key configuration (M1.5)

The owner configures keys from a settings page instead of env-vars, and the `ManageSubtitleDialogV2` "前往設定" dead loop is fixed.

> ⚠️ **Blocked-by:** three `.pen` copy revisions (F2 轉錄→抽取, F5 前往設定 M1 behaviour, F5 FFmpeg framing) — not addressed by PR #177.
> ✅ **Resolution path (2026-07-27):** all three are carried by **Story 1.7a**, batched into the same Pencil session as the subtitle-status badge spec. Epic 2 unblocks when 1.7a lands; no separate design story is needed.

### Story 2.1: Provider-key settings page + dead-loop fix 🟡

As a NAS owner,
I want to configure and edit TMDB / Claude / (optional) ASR keys in a settings page,
So that I don't need env-vars and the "前往設定" link actually goes somewhere.

**Acceptance Criteria:**

**Given** the key-configuration page, **When** I enter or edit a key, **Then** it persists to the existing encrypted secrets service and is never logged (NFR-S1) **And** `ManageSubtitleDialogV2`'s "前往設定" routes here, fixing the dead loop (FR25).

**Given** a non-HTTPS connection, **When** the page loads, **Then** it requires HTTPS, or warns and requires explicit confirmation before accepting a key (NFR-S3).

**Given** the F2/F5 UI, **When** this story starts, **Then** the three `.pen` copy revisions are already merged (this story is blocked until then).

**Given** Rule 17 (IR-r2 F9), **Then** the NFR-S3 HTTPS-warning behaviour is documented in `docs/deployment.md` + `docs/deployment.zh-TW.md` (the ffmpeg/multi-arch half landed with Story 1.6).
