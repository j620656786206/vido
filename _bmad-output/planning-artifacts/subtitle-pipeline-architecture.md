---
stepsCompleted: [1, 2, 3, 4, 5, 6, 7, 8]
workflowStatus: complete
completedDate: '2026-07-26'
lastStep: 8
inputDocuments:
  - _bmad-output/planning-artifacts/prd.md
  - _bmad-output/planning-artifacts/prd-validation-report.md
  - _bmad-output/planning-artifacts/vido-subtitle-pipeline-spec.md
  - _bmad-output/planning-artifacts/subtitle-engine-design-brief.md
  - _bmad-output/planning-artifacts/subtitle-v4-replan-and-feasibility-audit-2026-06.md
  - _bmad-output/planning-artifacts/architecture/adr-subtitle-route-c-generation.md
  - _bmad-output/planning-artifacts/architecture/adr-external-api-integration-standard.md
  - _bmad-output/planning-artifacts/ux-design-specification.md
  - project-context.md
  - docs/sse-event-types.md
  - docs/deployment.md
workflowType: 'architecture'
project_name: 'vido — Subtitle Pipeline'
user_name: 'Alexyu'
date: '2026-07-23'
premise:
  integration: 'Option B (strangler wrapper) + Option D (feature flag as safety valve)'
  decidedBy: 'Alexyu, 2026-07-23'
---

# Architecture Decision Document — Vido Subtitle Pipeline

_This document builds collaboratively through step-by-step discovery. Sections are appended as we work through each architectural decision together._

**Scope:** Brownfield extension of the existing Vido subtitle engine (`apps/api/internal/subtitle/`) to add embedded-subtitle extraction + LLM translation to Traditional Chinese. Governed by `project-context.md` (the bible) and driven by `prd.md` (Vido Subtitle Pipeline).

## Pre-agreed Premise (locked before Step 2)

Decided by Alexyu on 2026-07-23, ahead of the workflow:

- **Option B — Strangler wrapper.** A new `SubtitlePipeline` orchestrator becomes the single entry point for the **automatic** path. It runs extract-first and calls the existing `Engine.Process` as the final search fallback. The single non-test call site (`internal/subtitle/batch.go:244`) is rewired to the new orchestrator.
- **Option D — Feature flag as safety valve.** A config switch (`legacy | pipeline`) allows instant rollback to the current search-first behavior. Orthogonal to B; layered on top of it.
- **Boundaries agreed with the premise:**
  - The manual UI path (`POST /api/v1/subtitles/search` + `/download` + `/preview` + `/convert`) is **untouched in M1** — it is an explicit user action with different semantics from the automatic pipeline.
  - Route C (ASR generation, `generationBatchProcessor` / `TranscriptionService`) is **retained as-is** per spec §9; it moves behind the `ASRProvider` interface in M2.

## Project Context Analysis

### Requirements Overview

**Functional Requirements:** 34 FRs across 9 capability areas — M1 = 23, M1.5 = 1, P2 = 8, Tier-2 = 2.

The defining characteristic is the **reuse ratio**: roughly 11 of the 23 M1 FRs are already shipped (ffprobe track detection, content-based language detection, OpenCC s2twp conversion, secrets service, metadata context from 9R-8, TMDb manual re-match from Story 3.7, subtitle placer, SSE hub, and `TranslationService`'s glossary-aware translate). Roughly 12 are genuinely new (extraction, multi-track single-read, SDH filtering, no-text-source state, language routing, timestamp fidelity, per-cue quality gate, capability gating, re-translate). **This is an orchestration architecture, not a greenfield build.**

**Corrections to the source spec, discovered by reading the code during analysis:**

- `ai.ASRProvider` **already exists** (`internal/ai/asr.go:18`, Story 9R-9) with `Transcribe` / `TranscribeWithLanguage`, a compile-time `var _ ASRProvider = (*WhisperClient)(nil)` assertion, and a doc comment citing ADR `adr-subtitle-route-c-generation` Decision 2 (Speaches / WhisperLive / Subgen). Spec §3 lists the provider abstraction as new work — half of it is built.
- `services.TranslationService` **already exposes** `TranslateWithGlossary(blocks, glossary, progressFn)` and `TranslationRequest{Fields, Glossary}` — the ADR Decision 3 "keystone" has shipped. Spec §6.5 needs only the harvest-back half of the loop.
- `ai/claude.go`'s default model **is already fixed** to `claude-haiku-4-5` (Story 9R-1; the comment records that `claude-3-5-haiku-latest` was retired 2026-02-19). The 2026-06 feasibility audit's 5-bug list must be re-verified item by item, not copied forward.

**Non-Functional Requirements:** NFR-P1..P3 (hardware tiering, bounded per-tier concurrency), NFR-S1/S2 (encrypted secrets, no dialogue egress without a configured key), NFR-R1..R3 (fail-soft, idempotent, granular retry/resume), NFR-I1..I3 (Rule 27 Five Pillars, multi-arch Docker, sidecar naming). Per `prd-validation-report.md` gaps G2/G4, several NFRs lack hard numbers — quantify during this workflow and the M1 pilot.

**Scale & Complexity:** medium feature complexity, **high integration complexity** — three pre-existing parallel subtitle paths plus a compile-time package-cycle constraint.

- Primary domain: Go backend data pipeline inside an existing web app
- Complexity level: medium (integration-heavy)
- Architectural components: ~8 new, ~7 reused

### Technical Constraints & Dependencies

**HARD (compile-time / CI-enforced):**

- **Rule 19: `services ↛ subtitle`.** `subtitle → services` is allowed and already used (`subtitle/engine.go` holds a `services.TerminologyCorrectionServiceInterface`). The new orchestrator therefore **cannot live under `services/`**. Legal homes: inside `subtitle/`, or a new leaf package. `ai` is a verified leaf (safe to import from anywhere). Enforced by `boundaries_test.go` — a violation fails the build.
- **Rule 1:** all new backend code in `apps/api`.
- **Rule 4 + Rule 19:** Handler → Service → Repository; `Handler → Subtitle` is explicitly permitted.
- **Rule 7:** new error codes extend the authoritative 16-prefix registry **and** sync `code-review/instructions.xml` Step 3.
- **Rule 12:** `pnpm lint:all` (go vet, staticcheck@2026.1, eslint, prettier) must stay green.

**STRONG (conventions with teeth):** Rule 14 client lifecycle; Rule 20 `[@contract-vN]` stamping; Rule 27 Five Pillars; Rule 17 bilingual docs; project-context §5 worker pool (3–5 goroutines, no external queue, backoff 1s→2s→4s→8s); §8 SSE lazy-connection pattern + wire format; §9b `.zh-Hant.srt` / `.zh-Hans.srt`.

**EXTERNAL:** ffmpeg/ffprobe **must** be present in the Docker image (the 2026-06 audit found silent degradation when absent); Claude API (BYO key, env-var in M1); TMDb (existing shared client).

### Cross-Cutting Concerns Identified

1. **Output-file ownership — HIGHEST RISK.** Three writers exist today (Engine via batch, `SubtitleHandler` manual flow, Route C generation). Option B adds a fourth entry point; NFR-R2 idempotency requires a single defined owner and an explicit overwrite policy.
2. **AI budget governance.** `ai.Governor` (Story 9R-11) already caps concurrency, QPS, and run budget across both ASR and LLM. New translation calls **must ride it** — a second budget pool would defeat it.
3. **Capability gating.** One gate honored by the automatic, manual, and batch entry points (FR23 / NFR-S2), not per-handler `if` checks.
4. **SSE stage enum is a frontend-consumed wire contract.** Adding extract/translate stages is a Rule 20 contract change.
5. **Timestamp integrity.** Spans extraction, translation, per-cue retry, and assertion (FR11 / FR16 / FR17).
6. **Feature-flag seam (Option D).** Must gate exactly one boundary, not be sprinkled through the pipeline.
7. **Deployment.** Multi-arch image, ffmpeg presence, bilingual docs (Rule 17).

### Open Inconsistencies Carried Into Decision-Making

- `.zh-TW.srt` (spec §1/§3/§8) vs `.zh-Hant.srt` (PRD / project-context §9b) — **PRD wins**.
- Line segmentation: PRD restricts the ≤N-char rule to the ASR path; spec §7 applies it broadly — **PRD wins**.
- Key-config UI: spec §8 places it in M1; PRD defers it to M1.5 (M1 uses an env-var key) — **PRD wins**, and J3's dead 前往設定 loop remains broken through M1 as accepted debt.

### Resolved During Analysis — FR28 Phase Ambiguity

**RULING (Alexyu, 2026-07-23): FR28 is `[P2]`.** The FR list is missing the tag; three signals outweigh it — the Post-MVP prose lists re-translate-after-metadata-correction under Phase 2; the Innovation section calls it "same pattern as the glossary re-translate" and glossary (FR18–20) is `[P2]`; and the M1 must-have list and M1 journeys (J1/J3/J5) both exclude it.

M1 therefore ships **no** staleness detection, **no** re-translate entry point, and **no** change to Story 3.7's metadata manual-search apply contract.

### M1 Pilot Instrumentation (owner will iterate on prompts during M1)

Alexyu confirmed the M1 pilot **will re-run translations repeatedly** to compare quality. This is an **operator capability, not FR28**, and it promotes two design elements from "P2 insurance" to **mandatory in M1** — they are required for M1's own success criterion (measuring translation quality on the real DS920+ library):

1. **The cache key must be fully versioned:**

   ```
   hash(source cue) + metadata version + glossary version
                    + PROMPT VERSION + MODEL ID
   ```

   Omitting prompt/model is a **silent-failure trap**: changing the prompt and re-running would return the cached prior translation, making two variants look identical and yielding a false "the prompt made no difference" conclusion. The pilot's comparison data would be invalid with no error surfacing.

2. **Translation provenance persisted** per produced subtitle: tmdb_id used, metadata snapshot hash, glossary version, prompt version, model id. Required to attribute pilot results after the fact — and, for free, the prerequisite for the P2 re-translate.

**Operator re-run path (M1, ops-flavored — NOT a product feature):**

- A `force` parameter at the orchestrator entry that **both** overwrites an existing sidecar **and** bypasses the segment cache. It is the single named exception to the "never overwrite an existing acceptable `.zh-Hant.srt`" policy (the NFR-R2 "intent").
- **Must ride `ai.Governor`** (`AIRunBudgetUSD`, Story 9R-11) — repeated re-runs are exactly the runaway-spend scenario the governor exists for.
- Requires **no** staleness detection, **no** UI entry point, and **no** Rule 20 contract bump on the shipped Story 3.7 endpoint.
- Reuses `placer.go`'s existing atomic-write + `.bak` backup.

**Pilot variant isolation (NFR-I3 protection):** comparison variants **must not** be written into the media folder — Plex / Jellyfin / Synology Video Station auto-detect *any* matching sidecar and would flood the player's subtitle list. Variants go to a non-media scratch directory; only the chosen output is handed to `placer.go`. The media folder holds exactly one current-best `.zh-Hant.srt`.

**Escape hatch (falls out for free):** because the overwrite policy is "don't write if an acceptable file exists", deleting the sidecar and re-triggering is inherently a valid re-run — no extra code.

## Foundation Evaluation (Step 3 — reframed for brownfield)

The workflow's starter-template step does not apply: Vido's stack is fixed (Nx monorepo, Go/Gin backend in `apps/api`, React/TanStack Router frontend, SQLite, Tailwind). No starter is being selected, and **no project-initialization story is needed** — the first implementation story is the extraction stage, not scaffolding. The equivalent foundational decision is **which Claude client the translation path builds on**.

### Verified facts (code + current Claude API reference)

1. **No `anthropic-sdk-go` in `go.mod`** — `internal/ai/claude.go` is a hand-rolled raw-HTTP client (`anthropic-version: 2023-06-01`). Raw HTTP is a supported access mode, but the official guidance is to use the language's official SDK where one exists; Go has one.
2. `claudeRequest.System` is a plain `string`. **Prompt caching requires `system` to be an array of content blocks carrying `cache_control`** — the current type structurally cannot express it.
3. **Token metering already exists**: `doRequest` calls `BudgetFromContext(ctx).RecordLLM(model, in, out)` (Story 9R-11). What is missing is returning usage to the **caller** — `CompleteText` returns only `(string, error)`, so FR14 cannot surface a per-item/per-season estimate.
4. **Prompt-caching minimum prefix on `claude-haiku-4-5` (the repo default) is 4096 tokens.** Below it, `cache_control` **silently** does nothing (`cache_creation_input_tokens: 0`, no error).
5. Caching is a **prefix match**; render order is `tools` → `system` → `messages`. Stable content must physically precede volatile content.
6. A cache entry is readable only after the first response **begins streaming** — N parallel requests sharing a prefix all pay full price. With the §5 worker pool (3–5 goroutines) a season batch would write the same per-show prefix 3–5 times. **Mitigation:** send the first request for a show, await its first token, then release the rest.
7. Cache write premium: **1.25×** at the 5-minute TTL, **2×** at the 1-hour TTL. A season batch spans tens of minutes → the **1-hour TTL** is correct.
8. `ai.Provider` / `ai.TextCompleter` are **multi-provider** abstractions — Gemini implements them too (`gemini.go:33`), and `factory.go` switches on `ProviderName` with `NewProviderWithFallback` for cross-provider degradation. Adopting the SDK therefore replaces **one implementation** behind those interfaces; the client count does not change.

### Model ladder — re-baselined against current pricing

Spec §9 proposes "Sonnet base + Opus for hard re-translation". Current per-MTok pricing (input/output): **Haiku 4.5 $1/$5** (200K ctx) · **Sonnet 5 $3/$15** (introductory $2/$10 through 2026-08-31, 1M) · **Opus 4.8 $5/$25** (1M).

Subtitle translation is high-volume, cost-sensitive, short-per-cue, and the repo already defaults to `claude-haiku-4-5` (Story 9R-1). **Revised ladder:** `claude-haiku-4-5` base → `claude-sonnet-5` for cues the quality gate rejects → `claude-opus-4-8` only for the hardest lines.

### DECISION: adopt the official Go SDK for the Claude implementation (Option B)

**Ruling (Alexyu, 2026-07-23).** `internal/ai/claude.go` is re-implemented on `github.com/anthropics/anthropic-sdk-go`. Rationale: M1 needs prompt caching and caller-visible usage; P2 needs structured outputs (§6.5 glossary harvest) and likely the Batches API (season-scale cost). Hand-rolling those is four separate re-implementations of the wire protocol; Rule 27 / ADR Decision 3's "defer until the third duplication" trigger is already met by the roadmap.

**Interfaces are the moat — they do NOT change.** `ai.Provider`, `ai.TextCompleter`, `ai.ASRProvider`, `factory.go`, `gemini.go`, and every service consumer (`AIService`, `TranslationService`, `TerminologyCorrectionService`) are untouched. No Rule 20 contract bump.

**What the SDK gives us verbatim** (bindings confirmed against the Go SDK reference; **anything not listed here MUST be verified against the SDK repo before use** — do not infer Go symbols from the cURL or Python shapes):

- `anthropic.NewClient(option.WithAPIKey(...))`
- `client.Messages.New(ctx, anthropic.MessageNewParams{Model, MaxTokens, Messages, System})`
- Prompt caching: `System: []anthropic.TextBlockParam{{Text: ..., CacheControl: anthropic.NewCacheControlEphemeralParam()}}`; 1-hour TTL via `anthropic.CacheControlEphemeralParam{TTL: anthropic.CacheControlEphemeralTTLTTL1h}`
- Usage / cache verification: `resp.Usage.CacheCreationInputTokens`, `resp.Usage.CacheReadInputTokens`
- `option.WithRequestTimeout(time.Duration)`, `option.WithMiddleware(...)`
- Errors: `var apierr *anthropic.Error; errors.As(err, &apierr)` then switch on `apierr.StatusCode`

### Scope and regression surface

| Item | Detail |
|---|---|
| `internal/ai/claude.go` | 345 lines — internals replaced, exported surface preserved |
| `internal/ai/claude_test.go` | 545 lines, 24 tests (+2 Claude-provider tests in `retry_test.go` → a 26-test guard), **all** built on `httptest.NewServer`. The fake servers can be **kept** (the SDK accepts a base-URL override); only client construction and the request-body assertions change |
| `go.mod` | one new dependency |
| Untouched | interfaces, `gemini.go`, `factory.go`, all services, all handlers |

**Regression surface is wider than the subtitle pipeline.** `claude.go` also backs `Provider.Parse` — the **AI filename-parsing path** used by media scanning (`AIService`). A regression here breaks scanning, not just subtitles. The existing 26 Claude-touching tests (24 in `claude_test.go` + 2 in `retry_test.go`) are the guardrail: **all must stay green**, and that is the completion bar for this work.

### Migration hazards (must be handled explicitly)

1. **DOUBLE RETRY.** The SDK retries by default (max 2, on 408/409/429/5xx and connection errors). `doRequest` already wraps `retryTransient`. Naive wrapping yields up to 2×3 = **6 real requests** while the Governor's budget pre-check runs **once** — silently bypassing cost control. Choose ONE: disable the SDK's retries and keep `retryTransient`, or drop `retryTransient` and adopt the SDK's. **Do not run both.**
2. **Preserve the 404 model diagnostic.** The current 404 branch emits a specific, hard-won message ("the configured model id is deprecated or invalid — set `CLAUDE_MODEL` to a current model"), the direct product of the 9R-1 incident. Re-create it from `apierr.StatusCode == 404`.
3. **Preserve Rule 7 error mapping.** `ErrAIQuotaExceeded` (429), `ErrAITimeout`, and `ErrAIProviderError` are consumed elsewhere; re-map them from `*anthropic.Error` rather than letting SDK errors leak upward.
4. **Governor + budget stay.** `governed(...)` and `BudgetFromContext(ctx).RecordLLM(...)` (9R-11) must still wrap/observe every call — either around the SDK call or as `option.WithMiddleware`.
5. **Per-attempt timeout.** Currently `context.WithTimeout` per attempt; map to `option.WithRequestTimeout` and verify it composes with whichever retry strategy hazard 1 settles on.

### Consequences for later steps

- The translation prompt must be **designed around** the 4096-token cacheable minimum, not have caching bolted on afterwards (Steps 5/6).
- FR14 needs usage returned to the caller **and aggregated** — the metering exists, the plumbing does not.
- Per-show first-request serialization is a worker-pool concern (Step 6).

### NFR gap found during this step — carried into Step 4

M1.5's key-configuration UI has the user **type their Claude API key into a browser form**. Vido ships over **plain HTTP** by default (`docs/deployment.md` uses `http://localhost:8080`; the NAS target is `http://192.168.50.52:8088`; the production checklist lists an HTTPS reverse proxy as an **optional manual step**). The key therefore crosses the LAN in cleartext, and any device on that LAN can capture it.

**NFR-S1 does not cover this** — it guarantees encrypted **at rest** and never-logged, but says nothing about **in transit**. Proposed addition (to be ruled on in Step 4): the key-configuration page must require HTTPS, or explicitly warn and require confirmation over a non-HTTPS connection.

**Corollary:** M1's env-var key is not merely *simpler* than the M1.5 UI — it is strictly **safer**, because the key never crosses the network at all.

## Core Architectural Decisions

### Already Decided — do NOT re-litigate

From `project-context.md` (the bible) and earlier steps: Go/Gin backend in `apps/api` · SQLite · **no authentication** (v4 single-user) · tiered cache (AD #4) · worker pool 3–5 goroutines, no external queue (AD #5) · slog + `AppError` (AD #6) · SSE hub with lazy consumers (AD #8) · `/api/v1` + `{success,data}` envelope (Rules 3/10) · layered architecture (Rules 4/19) · **integration strategy B + D** (strangler wrapper + feature flag) · **official Go SDK for the Claude client** (Step 3) · **FR28 → P2 with a fully-versioned cache key and persisted provenance** (Step 2) · **model ladder** Haiku 4.5 → Sonnet 5 → Opus 4.8.

**Verified version:** `github.com/anthropics/anthropic-sdk-go` latest is **v1.59.0** (checked with `go list -m -versions`, not recalled). Pin at install time.

### Decision Priority Analysis

**Critical (block implementation):** D1 orchestrator placement · D2 pipeline-state persistence · D3 output-file ownership.
**Important (shape the architecture):** D4 chunking + cache key · D5 feature-flag seam · D6 SSE stage enum · D7 error-code prefix · D8 retry strategy · D9 key-transit security · D10 per-show request serialization.
**Deferred (P2 / Tier-2):** ASR fallback and provider routing (FR29–31) · glossary auto-harvest (FR18–20) · cost estimate UI (FR14) · re-translate after metadata correction (FR28) · local Tier-2 models (FR22/24).

### Schema finding that forces a wire-contract change

`subtitle_status` exists on **movies, series, and episodes** (migrations 018 + 025) as a TEXT column, typed in Go as `models.SubtitleStatus` with exactly four values: `not_searched | searching | found | not_found` — **all search-flavoured**. There is no vocabulary for extraction, translation, or "no usable text source".

**FR5 (mark an item as having no usable text source) cannot be expressed in the current enum.** The extension is therefore *required by the requirements*, not a stylistic choice. Because `subtitle_status` is serialized to the frontend (`json:"subtitle_status"`) and is used as a URL search param (the Rule 26 precedent cites `subtitleStatus`), extending it is a deliberate **Rule 20 wire-contract change** and must be stamped and acked.

Sibling columns `subtitle_search_score` and `subtitle_last_searched` are meaningless on the extract/translate path — they stay, unset, rather than being repurposed.

**Bonus:** `subtitle_tracks` (migration 021) already persists ffprobe track data on movies and series, so FR1 (detect subtitle tracks) is largely satisfied by existing data.

### D1 — Orchestrator placement · **`internal/subtitle/pipeline.go`**

Rule 19 makes `services → subtitle` a **compile-time** error (`boundaries_test.go`), so the orchestrator cannot live under `services/`. Of the two legal homes — inside `subtitle/`, or a new leaf package — we take **inside `subtitle/`**: the orchestrator needs the SRT parser, converter, and placer, and `subtitle → services` is already an established, legal direction (`engine.go` holds a `services.TerminologyCorrectionServiceInterface`). A new leaf package would add a hop without removing a dependency.

Entry from HTTP goes `Handler → Subtitle`, explicitly permitted by Rule 19.

### D2 — Pipeline-state persistence · **split by concern**

- **`subtitle_status` remains the single answer to "what is this item's subtitle state"**, extended with the new values the pipeline needs (extraction / translation / no-text-source / skipped).
- **A new table (migration 030)** holds only what has no place on the media row: **translation provenance** (tmdb_id used, metadata snapshot hash, glossary version, prompt version, model id) and per-run detail.

Rejected: a pipeline table that *also* answers "does this item have a subtitle". Rule 24's superseded-mechanism corollary is explicit — two storage mechanisms for one concept is a deferred-discovery time-bomb (the `series.seasons` / `seasons` precedent). One concept, one home.

Migrations in this repo are **Go files** under `apps/api/internal/database/migrations/` with a registry; the latest is **029**, so the new one is **030**.

### D3 — Output-file ownership · **`placer.go` is the only writer**

Three writers exist today (Engine via batch, `SubtitleHandler` manual flow, Route C generation); the orchestrator would be a fourth entry point. Ruling:

- **`placer.go` is the single writer** of the sidecar. Every path goes through it.
- **The orchestrator is the single caller on the automatic path.** The manual UI path keeps its current behaviour — it is an explicit user action with different semantics.
- **Overwrite policy: do not write if an acceptable `.zh-Hant.srt` already exists** (NFR-R2). The **only** named exception is the operator `force` re-run defined in Step 2, which also bypasses the segment cache.

### D4 — Chunking and cache key

Cache key (settled in Step 2, mandatory in M1):

```
hash(source cue) + metadata version + glossary version + prompt version + model id
```

**Open sub-problem carried forward:** the stable prompt prefix must exceed **4096 tokens** or `cache_control` is inert on `claude-haiku-4-5` — silently, with no error. Candidate prefix content: full TMDb metadata (title, original title, year, genre, overview, cast, production countries), the whole per-show glossary, the translation style rules, and few-shot examples.

**Ruling: do not pad the prefix with filler to reach the threshold.** If the genuine prefix cannot clear 4096 tokens for a given show, **explicitly disable prompt caching for that show and record it** — a caching design that looks active but never fires is worse than no caching, because it hides the cost.

### D5 — Feature-flag seam · **one call site**

The `legacy | pipeline` switch gates exactly **`internal/subtitle/batch.go:244`** — `legacy` calls `Engine.Process` as today, `pipeline` calls the new orchestrator. One conditional at one boundary. The flag must not appear anywhere inside the pipeline stages.

### D6 — SSE stage enum · extended, stamped, documented

Current stages: `searching | scoring | downloading | converting | correcting | placing | complete | failed` _(corrected 2026-07-28 — Stage 4.5 `correcting` was missing; it has been live on the wire since story 9-1 and is broadcast at `engine.go:194`, :176 before sub-1-3's const-block insert shifted it)_.
Added: `probing | extracting | translating | skipped`.

`subtitle_progress.stage` is a **frontend-consumed wire contract** (`useSubtitleSearch.ts`; documented in `docs/sse-event-types.md`). The extension therefore carries a `[@contract-vN]` stamp per Rule 20, and **both** `docs/sse-event-types.md` and `docs/sse-event-types.zh-TW.md` must be updated in the same change (Rule 17).

### D7 — Error codes · **extend `SUBTITLE_`, no new prefix**

New codes land under the **existing** `SUBTITLE_` prefix — e.g. `SUBTITLE_EXTRACT_FAILED`, `SUBTITLE_NO_TEXT_SOURCE`, `SUBTITLE_TRANSLATE_FAILED`, `SUBTITLE_TIMESTAMP_MISMATCH`. The authoritative prefix count stays at **16**.

This follows ADR `adr-external-api-integration-standard` Decision 4 / Pillar 4: reuse existing prefixes, don't invent new ones. Reuse `TRANSCRIPTION_` for ASR (P2) and `METADATA_` / `TMDB_` for matching.

**Mandatory sync:** the Rule 7 code list in `project-context.md` **and** `_bmad/bmm/workflows/4-implementation/code-review/instructions.xml` Step 3 (both the inline prefix list and the sync date).

### D8 — Retry strategy · **disable the SDK's retries, keep `retryTransient`**

The Go SDK retries by default (max 2, on 408/409/429/5xx and connection errors) and `doRequest` already wraps `retryTransient`. Running both yields up to 6 real requests per logical call.

**Ruling: turn the SDK's retries off and keep `retryTransient`.** The decisive reason is not preference but correctness — the Governor's budget pre-check (`governed(...)`, Story 9R-11) wraps *outside* `retryTransient`. Moving retries into the SDK would put them *below* the budget gate, so a retry storm would bypass cost control entirely. Keeping the existing nesting preserves the existing guarantee.

### D9 — Key transit · **new NFR-S3**

> **NFR-S3:** The provider-key configuration page must require an HTTPS connection, or explicitly warn the user and require confirmation when served over plain HTTP.

Closes the gap found in Step 3: NFR-S1 covers at-rest encryption and log sanitization but is silent on **in transit**, while Vido ships over plain HTTP by default. M1 is unaffected (env-var key, never crosses the network); this binds M1.5.

### D10 — Per-show request serialization · **orchestrator-level gate**

A cache entry is only readable once the first response begins streaming, so 3–5 worker-pool goroutines translating the same show would each write the same prefix. The orchestrator holds a **per-show first-request gate**: the first request for a show runs alone; once it returns, the rest are released.

This is deliberately an orchestrator concern, not a worker-pool one — the pool stays a generic 3–5 goroutine executor (AD #5) with no show-awareness.

### Decision Impact Analysis

**Implementation sequence** (each step unblocks the next):

1. `claude.go` on the Go SDK (D8 retry ruling baked in) — the foundation everything else calls.
2. Migration 030 + `SubtitleStatus` enum extension (D2) — the state model.
3. `SUBTITLE_` error codes + `code-review/instructions.xml` sync (D7).
4. SSE stage extension + bilingual doc update (D6).
5. `internal/subtitle/pipeline.go` orchestrator (D1) with the ownership policy (D3), per-show gate (D10), and prompt/cache design (D4).
6. Feature-flag seam at `batch.go:244` (D5).
7. M1.5: key-configuration page under NFR-S3 (D9).

**Cross-component dependencies:**

- D2 (status enum) and D6 (SSE stages) are **both** frontend wire contracts and should be stamped together so the frontend absorbs one coordinated change, not two.
- D3 (ownership) constrains D5: the flag selects the *caller*, never a second writer.
- D4 (cache key + prefix) depends on the glossary shape, which is P2 — so M1 must version the glossary field even while it is always empty.
- D8 is a precondition for any cost measurement: with double retries, spend figures would be wrong before FR14 is even built.

## Implementation Patterns & Consistency Rules

### Scope of this section

Generic naming, response-format, date-format, test-location, interface-location, and case-transformation patterns are **already binding** via `project-context.md` Rules 3, 6, 8, 9, 11, and 18. They are **not re-decided here** — restating them would create a second source of truth.

This section covers only the **twelve points specific to this pipeline** where two dev agents would plausibly implement differently and no existing Rule decides the matter.

### P0 — Source-language determination · **a real gap in the existing detector**

`internal/subtitle/detector.go` distinguishes **CJK variants only**:

```go
LangTraditional  = "zh-Hant"
LangSimplified   = "zh-Hans"
LangAmbiguous    = "zh"
LangUndetermined = "und"   // "No CJK content detected"
```

English, Japanese, French, and Spanish **all return `und`** — one undifferentiated bucket. **FR9's routing (Traditional → done / Simplified → OpenCC / English → translate / other → skip) cannot be implemented with this detector alone.** Left unspecified, agents diverge three ways: treat `und` as English (which translates Japanese subtitles into nonsense), switch to the ffprobe track tag (seemingly against §9b's "content, not filename"), or pull in a new language-detection dependency.

**Ruling:**

- `detector.Detect` remains authoritative for the **CJK variant** decision only — that is its actual job, and §9b's "detect from content, not filename" rule governs Traditional-vs-Simplified.
- **Source-language identification uses the ffprobe stream language tag** (already persisted in `subtitle_tracks`, migration 021). §9b is not violated: a stream tag is container metadata, not a filename guess.
- **M1 accepts only tracks tagged `eng` / `en`.** Everything else routes to skip.
- **`und` is NEVER treated as English.** No explicit English tag ⇒ skip.

This keeps M1 honest about its stated scope (English → Traditional Chinese only) and fails closed rather than mistranslating.

**Also noted:** `internal/ai/prompts/subtitle_translator.go` already exists (Route C heritage) — the prompts package is the established home; do not create a new one.

### P1 — Cue identity

A cue's identity for caching and retry is **`hash(content)`**, never its index. SDH filtering shifts indices, so index-keyed caches and retries silently mis-target after the first filtered line.

### P2 — Timestamps never leave Go

The LLM receives **numbered text only**. `SubtitleBlock` timestamps stay on the Go side and are re-stitched programmatically after the response returns.

❌ **Anti-pattern:** serializing the whole SRT (timestamps included) into the prompt and asking the model to preserve them. This is the single largest threat to FR11/FR17.

### P3 — Chunk is the transport unit, cue is the retry unit

A chunk batches cues into one request; a failure inside a chunk retries **only the affected cues** (FR16), never the whole chunk.

### P4 — Quality gate runs BEFORE OpenCC

```
LLM output
  → quality gate  (Simplified leakage / empty / echoed / cue-count mismatch)  → per-cue retry
  → OpenCC s2twp  (deterministic final pass)
  → output
```

❌ **Anti-pattern — the easiest mistake in this design:** running OpenCC first. It converts the Simplified characters away, so `detector.go` can never observe the leakage, the per-cue retry never fires, **and the quality gate becomes dead code that nobody notices.** Order is load-bearing, not stylistic.

### P5 — "Acceptable existing subtitle" is a defined predicate

NFR-R2's overwrite guard requires a testable definition: the file **exists**, **parses via `ParseSRT` without error**, and has **cue count > 0**. An existence-only check treats a truncated or zero-byte artifact as valid and permanently blocks regeneration.

### P6 — SDH filtering happens BEFORE translation

Filtering after translation means paying tokens for lines that are then discarded.

### P7 — Filtered cues keep their original numbering

SDH removal does **not** renumber the remaining cues. FR17's timestamp-equality assertion needs a stable common basis between source and output.

### P8 — SSE broadcast is throttled per chunk

Broadcast **once per chunk**, never per cue. The hub's broadcast channel holds 256 events and drops on overflow (AD #8) — per-cue broadcasting over a season would flood and silently lose events. This mirrors `ScannerService`'s established "every 10 files" throttle.

### P9 — Provenance is written AFTER a successful place

Write order is: place the file → confirm success → write provenance. The reverse leaves records claiming a subtitle exists when a crash occurred between the two writes.

### P10 — Go SDK symbol discipline

Bindings confirmed in the architecture record (`System` content blocks, `CacheControl`, `Usage.CacheReadInputTokens` / `CacheCreationInputTokens`, `option.*`, `*anthropic.Error`) may be used directly. **Anything not listed — notably the Go syntax for Batches and structured outputs — MUST be verified against the SDK repository before it is written.** Do not infer Go symbols from cURL or Python shapes. **Pin `github.com/anthropics/anthropic-sdk-go v1.59.0`.**

### P11 — Prompt text and its version constant live together

Prompt text lives in `internal/ai/prompts/`, and the **prompt version constant lives in the same file**. Changing prompt text **requires** bumping the constant in the same edit — otherwise the cache key is unchanged and a re-run silently returns the previous translation (the Step 2 silent-failure trap).

### Enforcement Guidelines

**All implementing agents MUST:**

1. Route source language by the ffprobe track tag, never by treating `und` as English (P0).
2. Keep timestamps out of every prompt (P2).
3. Run the quality gate before OpenCC (P4).
4. Bump the prompt version constant in the same commit as any prompt text change (P11).
5. Verify any Go SDK symbol not listed in P10 against the SDK repo before writing it.

**Verification:** points 1–4 are behavioural and belong in code review; point 3 additionally warrants a unit test that feeds deliberately-Simplified LLM output through the pipeline and asserts the retry fired — without it, the inverted order is undetectable at runtime.

**Pattern updates:** if any of P0–P11 proves wrong during implementation, it is a Rule 24 discovery — triage it into a tracked lane, do not fix it silently in one story.

## Project Structure & Boundaries

This is a **brownfield delta**, not a greenfield tree. Reproducing Vido's full directory listing would bury the ~28 files that actually change. Markers: 🆕 new · ✏️ modified · 🔒 deliberately untouched.

### Delta tree

```
apps/api/
├── cmd/api/main.go                          ✏️ wire orchestrator + feature flag
├── internal/
│   ├── subtitle/
│   │   ├── pipeline.go                      🆕 orchestrator (D1)
│   │   ├── pipeline_test.go                 🆕
│   │   ├── extractor.go                     🆕 ffmpeg -map 0:{index} -c:s srt (FR2/FR3 — 1.4 Finding 2)
│   │   ├── extractor_test.go                🆕
│   │   ├── sdh_filter.go                    🆕 FR4 — runs pre-translation (P6)
│   │   ├── sdh_filter_test.go               🆕
│   │   ├── router.go                        🆕 language routing FR9 + the P0 tag rule
│   │   ├── router_test.go                   🆕
│   │   ├── quality_gate.go                  🆕 FR16/FR17 — runs BEFORE OpenCC (P4)
│   │   ├── quality_gate_test.go             🆕 MUST include a Simplified-leak-triggers-retry test
│   │   ├── engine.go                        ✏️ +4 PipelineStage consts ONLY (1.3 [@contract-v1]); search flow untouched
│   │   ├── batch.go                         ✏️ one line at :244 + flag (D5)
│   │   ├── detector.go                      🔒 scope narrowed to CJK-variant only (P0)
│   │   ├── converter.go / placer.go         🔒 reused; placer is the sole writer (D3)
│   │   ├── srt_parser.go                    🔒 SubtitleBlock is the cue contract
│   │   └── manager.go / scorer.go / providers/   🔒
│   ├── ai/
│   │   ├── claude.go                        ✏️ internals replaced with the official SDK (Step 3)
│   │   ├── claude_test.go                   ✏️ httptest servers kept; construction + body assertions change
│   │   ├── prompts/
│   │   │   └── subtitle_translator.go       ✏️ extended + prompt version constant (P11)
│   │   ├── provider.go / factory.go / gemini.go   🔒 the interfaces are the moat
│   │   ├── governor.go / budget.go / retry.go     🔒 D8 keeps retryTransient
│   │   └── asr.go                           🔒 P2 only
│   ├── services/
│   │   ├── ffprobe_service.go               ✏️ +stream_index (additive, 1.4) — FR1 reuse holds for enumeration; extraction needs the index
│   │   ├── translation_service.go           ✏️ must return usage (FR14 prerequisite)
│   │   ├── scanner_service.go               ✏️ V3 — enqueue scanned items → worker pool (FR13)
│   │   └── terminology_service.go           🔒
│   ├── repository/
│   │   ├── subtitle_run_repository.go       🆕 provenance (D2)
│   │   ├── subtitle_run_repository_test.go  🆕
│   │   └── glossary_repository.go           🔒 P2 only
│   ├── models/
│   │   ├── movie.go                         ✏️ SubtitleStatus enum extension — WIRE CONTRACT
│   │   ├── series.go / episode.go           🔒 share the same enum
│   │   └── subtitle_run.go                  🆕
│   ├── database/migrations/
│   │   ├── 030_create_subtitle_runs_table.go   🆕 provenance table + new status values
│   │   └── 030_create_subtitle_runs_table_test.go   🆕
│   ├── handlers/subtitle_handler.go         🔒 manual path unchanged in M1 (D3)
│   └── sse/hub.go                           🔒 event types unchanged; only stage values extend
docs/
├── sse-event-types.md                       ✏️ new stages (D6)
├── sse-event-types.zh-TW.md                 ✏️ Rule 17 sync
└── deployment.md (+ .zh-TW.md)              ✏️ ffmpeg requirement, multi-arch, NFR-S3
project-context.md                           ✏️ Rule 7 new codes + §9b update
_bmad/bmm/workflows/4-implementation/code-review/instructions.xml   ✏️ Rule 7 prefix-list sync date
apps/web/                                    (frontend delta — stories 1.7a/1.7b, added 2026-07-27 per IR-r2 F6)
├── src/utils/libraryStatus.ts               ✏️ badge derivation for the 5 new statuses (1.7b)
├── src/utils/libraryStatus.spec.ts          ✏️
├── src/components/media/EpisodeList.tsx     ✏️ status-icon map — 9 values (1.7b)
└── src/components/media/EpisodeList.spec.tsx ✏️
ux-design.pen                                ✏️ j2 badge spec screen + F2/F5 copy revisions (1.7a)
scripts/export-pen-screenshots.py            ✏️ SCREENS + ("flow-j-specs","j2-d") (1.7a)
_bmad-output/planning-artifacts/ux-design-specification.md   ✏️ StatusBadge enumeration + pointer (1.7a AC #9)
```

> Badge labels / tints / icons are authoritative in `ux-design.pen` `flow-j-specs` screen j2 (story 1.7a) — deliberately **not** restated here (single source of truth; see § Scope of this section).

**~16 new · ~21 modified · ~13 explicitly untouched** _(recounted 2026-07-27, IR-r2 F6 — +`scanner_service.go` (the V3 amendment, previously claimed but never applied) and +7 frontend/design files for 1.7a/1.7b; re-tallied 2026-07-28 by story 1.4 when `ffprobe_service.go` moved 🔒 → ✏️)_.

### Requirements-to-structure mapping

| FR group | Location |
|---|---|
| **A. Detection & extraction** (FR1–5) | FR1 `ffprobe_service.go` ✏️ (+`stream_index`, 1.4) + existing `subtitle_tracks` column · FR2/3 `extractor.go` 🆕 · FR4 `sdh_filter.go` 🆕 · FR5 new status value in `models/movie.go` ✏️ |
| **B. Language routing** (FR6–9) | `router.go` 🆕 + `detector.go` 🔒 (CJK only) + `converter.go` 🔒 (s2twp) |
| **C. AI translation** (FR10–13) | `pipeline.go` 🆕 → `translation_service.go` ✏️ → `ai/claude.go` ✏️ + `prompts/subtitle_translator.go` ✏️ |
| **D. Quality assurance** (FR15–17) | `quality_gate.go` 🆕 (gate) → `converter.go` 🔒 (OpenCC final pass) |
| **F. Keys** (FR21/FR23) | M1 env-var via `main.go` · M1.5 secrets service under NFR-S3 |
| **G. Metadata** (FR26–27) | existing TMDb service 🔒 + `subtitle_run` provenance 🆕 |
| **I. Delivery & status** (FR32–33) | `placer.go` 🔒 (sole writer) + `sse/hub.go` 🔒 (stage values extend) |
| **P2 groundwork** | `ai/asr.go`, `glossary_repository.go`, `engine.go` (search) — **all already exist**; nothing new to build |

### Architectural boundaries

```
Handler ──────────────┐
   │                  │  (Handler → Subtitle: explicitly permitted by Rule 19)
   ▼                  ▼
Service ──────────► Subtitle ──────► ai (leaf package)
   │                  │                 │
   ▼                  ▼                 ▼
Repository      placer / converter   Claude SDK
   │                  │
   ▼                  ▼
 SQLite          filesystem (single write point)

❌ Service ↛ Subtitle — compile-time prohibition, guarded by boundaries_test.go
```

**External boundaries:** Claude API (Rule 27 Five Pillars · Governor throttle · BYO key) · the `ffmpeg`/`ffprobe` binaries (**must be present in the Docker image** — the 2026-06 audit proved absence degrades silently) · the media folder (mounted read-only; only `placer.go` writes the sidecar beside the video).

### Data flow — M1 happy path

```
scan → ffprobe (subtitle_tracks) → find text track
  → track tag eng/en?  no → status = skipped ──┐
  → extractor (ffmpeg -c:s srt, one pass)      │
  → sdh_filter (before translation)            │
  → detector (CJK variant)                     │
      Traditional → done                       │
      Simplified  → OpenCC → done              │
      English     → chunk → Claude             │
                    (timestamps stay in Go)    │
                  → quality_gate → per-cue retry
                  → OpenCC final pass          │
  → placer (sole writer; never overwrites)     │
  → provenance written AFTER a successful place│
  → SSE stage broadcast (once per chunk) ◄─────┘
```

### Development-workflow integration

- **Tests** stay co-located (`*_test.go` beside the source) per Rule 9 — no separate `tests/` directory.
- **Migrations** are Go files under `internal/database/migrations/` with a registry; `030` follows `029`.
- **Build/deploy**: the multi-arch (amd64 + arm64) image must bundle `ffmpeg`/`ffprobe`; `docs/deployment.md` and its zh-TW counterpart both document it (Rule 17).
- **Lint gate**: `pnpm lint:all` (go vet → staticcheck@2026.1 → eslint → prettier) must pass before push, per Rule 12.

## Architecture Validation Results

Adversarial pass over the 22 M1 FRs and 11 NFRs (count corrected 2026-07-27) — not a rubber stamp. Six gaps found; all resolved below. Two required checking the `.pen` design directly.

### Coherence validation ✅

- **Rule 19**: orchestrator in `subtitle/` uses a legal direction; `boundaries_test.go` enforces it. Consistent.
- **D3 ownership ↔ NFR-R2**: single writer + the P5 "acceptable subtitle" predicate make the overwrite guard testable. Consistent.
- **D8 retry ↔ Governor budget**: keeping `retryTransient` inside `governed(...)` preserves the budget pre-check; inverting it would silently bypass cost control. Consistent.
- **P4 gate order ↔ FR15/16**: quality gate before OpenCC is the only order under which Simplified-leak detection can fire. Consistent.
- **P2 groundwork already exists** (`asr.go`, `glossary_repository.go`, `engine.go`) — no future-proofing scaffolding built prematurely.

### Requirements coverage — gaps found and resolved

**V1 (Critical) — FR6 contradicted our own P0 rule.** FR6 says "determine language *from content, not filename*"; P0 routes source language by the ffprobe track tag, which is container metadata, not content. **Ruling (Alexyu): V1(a)** — revise FR6's scope so "content-based detection" governs the **Traditional-vs-Simplified** decision (the original Bazarr-bug case), while **source-language identification uses the track tag**. Requires a PRD amendment (cross-document action, tracked below).

**V2 (Critical, corrected during validation) — the UI entry already exists.** My initial finding ("M1 has no trigger UI") was wrong. Verified against `ux-design.pen` flow F v2: **F2-D-v2** (`HCjjz`) carries a `btn-generate` "生成字幕" primary button with a demoted "搜尋線上字幕（成功率低）" fetch row (Route-A-dormant, ADR-consistent); **F5-D-v2** (`BeuNH`) is the fail-soft capability gate ("字幕生成尚未設定" + "前往設定"). The real gap is the **backend endpoint contract** those buttons call, which Step 6 did not define. **Resolution:** add **`POST /api/v1/subtitles/pipeline/run`** (single item, optional `force`) as the backend for F2's button — not a UI-less API. **Plus three copy/scope mismatches to fix before M1 ships:**
- F2's subtitle reads "**轉錄**＋AI 翻譯" — transcription is ASR (Route C = **M2**). M1 is **extraction** + translation. Ship the M1 label as "抽取內嵌字幕＋AI 翻譯"; the "轉錄" wording waits for M2, or the user expects a capability M1 doesn't have.
- F5's "前往設定" button is a **known-broken dead loop in M1** (PRD J3; fixed in M1.5). A visible button that does nothing is worse than none. In M1 either hide it (show env-var guidance only) or route it to documentation.
- F5's warning panel frames **FFmpeg as a user-configurable setting**. Step 6 rules FFmpeg is **bundled in the Docker image** — its absence is a deployment error, not a user setting. Reword to mention only the API key.

**V3 (Critical) — FR13 (auto-translate on media add) had no trigger hook.** The feature flag lives on the **batch** path (`batch.go:244`); nothing routes newly-scanned items into the pipeline. **Resolution:** `ScannerService` enqueues completed items into the existing worker pool (AD #5), which calls the orchestrator. Adds `scanner_service.go` ✏️ to the delta tree.

**V4 (Important) — NFR-P3 (per-hardware-tier concurrency) unaddressed in M1.** Compute detection is P2 (FR31). **Resolution:** M1 uses a **fixed concurrency of 2** on the DS920+ target; "bounded per hardware tier" is explicitly labeled P2 groundwork so the NFR is not falsely shown as met.

**V5 (Important) — NFR-R3 resumability had no mechanism.** Per-cue retry exists; resume did not. **Resolution:** the D2 provenance table **explicitly** carries resume responsibility — completed items/cues are recorded and skipped on re-run — rather than leaving it an implicit side effect.

**V6 (Important) — FR23 capability gate had no home.** Cross-cutting concern #3 names one gate for three entry points, but no file owned it. **Resolution:** a single check at the top of `pipeline.go`, shared by the new endpoint (V2), the batch entry, and the scanner enqueue path (V3).

### NFRs otherwise covered

NFR-P1/P2 (I/O-bound extraction, no local heavy compute in M1) ✅ · NFR-S1 (encrypted secrets, sanitized logs) ✅ · **NFR-S3 added** (key transit, D9) · NFR-R1 (fail-soft) ✅ · NFR-R2 (idempotent, D3+P5) ✅ · NFR-I1 (Rule 27 Five Pillars) ✅ · NFR-I2 (multi-arch Docker + ffmpeg) ✅ · NFR-I3 (`.zh-Hant.srt` sidecar) ✅.

### Cross-document actions (not resolvable inside this document)

1. **PRD amendment for FR6** (V1(a)) — scope "content-based" to CJK-variant detection; add track-tag source-language identification. Owner: PM/analyst. **✅ Done 2026-07-26** (PRD amended; verified IR 2026-07-27).
2. **PRD tag fix for FR28** — add the missing `[P2]` tag (Step 2 ruling). **✅ Done 2026-07-26.**
3. **`.pen` copy revisions** (V2 ①②③) — F2 subtitle wording, F5 button behaviour, F5 FFmpeg framing. Owner: UX (Sally), before M1 UI ships. Per CLAUDE.md, any `.pen` edit requires regenerating and committing screenshots. **→ Carried by story `sub-1-7a` AC #7 (2026-07-27).**

### Architecture readiness assessment

**Overall status: READY WITH GAPS RESOLVED.** Every in-document gap (V2–V6) is closed here; the three cross-document actions are tracked and owned. The core design (strangler wrapper, single writer, quality-gate-before-OpenCC, versioned cache key, SDK client) held up under adversarial review without structural change — the gaps clustered at **trigger entry points and one requirement's wording**, not in the core.

**Confidence: medium-high.** The residual risk is entirely in the tracked cross-document actions (chiefly the FR6 PRD amendment); the buildable architecture is coherent and complete.

**Key strengths:** ~48% M1 reuse (orchestration, not greenfield); compile-time boundary enforcement; a single defined writer; and hazard-level detail (double-retry, gate order, cache-key versioning, silent-caching-below-4096) captured before implementation rather than discovered during it.

**Areas for future enhancement:** ASR fallback + provider routing (M2), glossary auto-harvest (P2), compute-aware tiering (P2), Batches API + structured outputs (triggers the P2 re-evaluation already recorded in Step 3).

## Architecture Completion Summary

**Create-Architecture workflow: COMPLETE ✅** — 8 steps, 2026-07-26. Brownfield extension of the Vido subtitle engine for automated media → Traditional-Chinese subtitles (M1 scope; M1.5/P2/Tier-2 groundwork recorded).

### Decision index (quick reference for implementers)

| # | Decision | Ruling |
|---|----------|--------|
| Integration | Strangler wrapper (B) + feature flag (D) | New `SubtitlePipeline` fronts the automatic path; flag at `batch.go:244` |
| Client | Official Go SDK (Step 3) | `anthropic-sdk-go` **v1.59.0**; interfaces unchanged; SDK retries **off** |
| FR28 | → P2 | M1 pays two premiums: fully-versioned cache key + persisted provenance |
| Model ladder | Haiku 4.5 → Sonnet 5 → Opus 4.8 | one tier cheaper than the spec |
| D1 | Orchestrator in `internal/subtitle/pipeline.go` | Rule 19 forbids `services/` |
| D2 | Split state | `subtitle_status` (extended) + migration 030 provenance table |
| D3 | `placer.go` sole writer | overwrite guard + single `force` exception |
| D4 | Versioned cache key | disable caching (recorded) if prefix < 4096 tokens |
| D6 | SSE stages + `probing/extracting/translating/skipped` | wire contract — stamp + bilingual doc |
| D7 | Extend `SUBTITLE_` | no new Rule 7 prefix (stays 16) |
| D8 | Keep `retryTransient`, disable SDK retries | preserves Governor budget gate |
| D9 | New NFR-S3 | key-config page requires/​warns on non-HTTPS |
| D10 | Per-show first-request gate | orchestrator-level, not worker-pool |

### First implementation priority

**Not scaffolding — this is brownfield.** The implementation sequence (Decision Impact Analysis, Step 4) starts with **re-implementing `internal/ai/claude.go` on the Go SDK** (foundation for caching + usage, with the D8 retry ruling baked in), then migration 030 + enum, then the `SUBTITLE_` codes, then SSE stages, then the orchestrator, then the feature-flag seam, then (M1.5) the key page under NFR-S3.

### Open cross-document actions (must be handled outside this document)

1. **PRD amendment — FR6 scope** (V1(a)): content-based detection governs CJK variant; source language uses the track tag. **✅ Done 2026-07-26.**
2. **PRD tag fix — FR28** gets its missing `[P2]` tag. **✅ Done 2026-07-26.**
3. **`.pen` copy revisions** (V2 ①②③): F2 "轉錄"→"抽取" wording, F5 "前往設定" M1 behaviour, F5 FFmpeg framing. Owner UX; regenerate + commit screenshots per CLAUDE.md. **→ Carried by story `sub-1-7a` AC #7 (2026-07-27).**

### Handoff

This document is the single source of truth for the subtitle-pipeline architecture. Downstream BMAD flow: **`create-epics-and-stories`** (PRD + this architecture → implementation-ready stories), with **`check-implementation-readiness`** (the `IR` menu item) as the adversarial gate before dev. The three cross-document actions above should be closed — or explicitly accepted as deferred — before story creation, since FR6's wording drives the language-router story.

**Status: READY FOR IMPLEMENTATION** (with the three tracked cross-document actions).
