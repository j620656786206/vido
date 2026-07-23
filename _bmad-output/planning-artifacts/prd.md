---
stepsCompleted: ['step-01-init', 'step-02-discovery', 'step-03-success', 'step-04-journeys', 'step-05-domain', 'step-06-innovation', 'step-07-project-type', 'step-08-scoping', 'step-09-functional', 'step-10-nonfunctional', 'step-11-polish', 'step-12-complete']
workflowStatus: complete
completedDate: '2026-07-23'
classification:
  projectType: web_app (backend-pipeline / api_backend character; auth N/A per v4 single-user)
  domain: general (self-hosted consumer media)
  complexity: medium
  projectContext: brownfield
inputDocuments:
  - _bmad-output/planning-artifacts/vido-subtitle-pipeline-spec.md
  - project-context.md
  - _bmad-output/planning-artifacts/subtitle-engine-design-brief.md
  - docs/deployment.md
  - docs/sse-event-types.md
documentCounts:
  briefs: 1
  research: 0
  brainstorming: 0
  projectDocs: 3
  techSpec: 1
workflowType: 'prd'
project: 'Vido — Subtitle Pipeline'
---

# Product Requirements Document - Vido Subtitle Pipeline

**Author:** Alexyu
**Date:** 2026-07-23
**Scope:** Automated media → Traditional Chinese subtitle pipeline. Target deployment: mainstream NAS (Synology / QNAP) + self-built NAS (Unraid). Brownfield — extends the existing Vido subtitle engine (`apps/api/internal/subtitle/`).

**Primary source:** `vido-subtitle-pipeline-spec.md` (unified tech spec). Bound by `project-context.md` (the bible).

<!-- PRD content is built append-only across the workflow steps. -->

## Executive Summary

**Vision:** Every foreign-language film and episode in a self-hosted NAS library carries a correctly-timed Traditional-Chinese subtitle that the owner never had to hunt for or hand-fix.

**Differentiator:** Vido leads with **embedded-subtitle extraction + LLM translation to 繁中** — directly attacking Bazarr's known Traditional-Chinese weakness (search-first, thin zh-TW sources) — and layers on **per-show glossary consistency** and **compute-aware hardware tiering**, tuned specifically for Traditional-Chinese self-hosters. Quality is guaranteed by a two-stage design: the LLM handles semantics, and a deterministic OpenCC pass guarantees Traditional script.

**Target users:** Owners of mainstream NAS boxes (Synology / QNAP) and self-built NAS (Unraid) whose libraries are English-first and who want reliable zh-TW subtitles without touching Bazarr or fixing timing.

**Approach:** A brownfield extension of the existing Vido subtitle engine. M1 proves the core assumption — extraction + Claude translation, running within the resource envelope of a standard Synology DS920+.

## Success Criteria

### User Success

A NAS owner with an English-language film/episode opens it in their player (Plex / Jellyfin / Synology Video Station) and a correctly-timed, readable Traditional-Chinese subtitle (`.zh-Hant.srt`) is simply **there** — no Bazarr, no manual download, no timing fixes. The "aha" moment is opening the media and finding zh-Hant already present with zero manual steps.

- **Measurable:** For a media item that has an embedded English text subtitle, time-to-`.zh-Hant.srt` is on the order of tens of seconds (extraction is I/O-bound + one translation call). The produced subtitle is accepted **without hand-editing** in ≥ a target % of cases (a trust proxy — exact % set during pilot on the owner's real library).

### Business Success

Vido is single-user, self-hosted, no revenue — so "business success" = **adoption and replacement of manual workflows**.

- **3-month:** the pipeline runs on the standard target (DS920+) against the owner's real library; the owner **stops manually sourcing** zh-TW subtitles for English content.
- **12-month:** glossary auto-harvest drives per-episode manual term entry toward zero (consistent character/place names across a whole series); the local Tier-2 (no-key) path and additional source languages land.
- **Metrics:** zh-Hant subtitle **coverage %** of the English-content library via the pipeline; **manual-intervention rate** per item.

### Technical Success

- **M1 runs on the standard target — DS920+ (Celeron J4125 / 4 GB)** — using extraction + cloud translation only (no local ASR on the NAS).
- **Soft-subtitle output only, zero re-encode**; extension `.zh-Hant.srt` / `.zh-Hans.srt` per project-context §9b (NOT `.zh-TW.srt`).
- **Provider abstraction**: OpenAI-compatible ASR interface + a pluggable translation provider so cloud / external-worker / local are swappable without touching the pipeline core.
- **Cost control**: Claude token spend per season is bounded and already-translated segments are cached; the UI surfaces an estimate.
- **Reuse, not rebuild**: the existing engine's ffprobe subtitle-track detection, OpenCC (s2twp) conversion, subtitle placer, and SSE task-status are reused; the pipeline prepends extraction and reorders search to last.
- Follows the bible: all new backend code in `apps/api`; slog; `{success,data}` envelope; Rule 7 error codes (new `SUBTITLE_`/pipeline codes registered); **SSE consumers stay lazy** (no connect-on-mount).

### Measurable Outcomes

- **M1 acceptance (from spec §8):** Docker deploy on DS920+ → scan → extract an English embedded sub → Claude translate → `.zh-Hant.srt` appears beside the video → Video Station lists the Traditional-Chinese track.
- **Translation quality bar:** timing preserved (cue offsets match source within tolerance); line length ≤ N chars with the two-line rule; human spot-check of a sample reaches an agreed "usable without edit" threshold.

## Product Scope

Scope is defined authoritatively in **Project Scoping & Phased Development** below (MVP / M1.5 / Growth / Vision with rationale and risk analysis). In brief: **M1** validates the core assumption — extract + translate, env-var key, English → 繁中, on DS920+; **M1.5** adds the key-config UI; **Growth (M2/M3)** adds ASR fallback, glossary auto-harvest, compute-aware routing, and local Tier-2 models; the **Vision** is fully-local, zero-key generation with hands-free per-show consistency.

## User Journeys

Vido is single-user and self-hosted (v4 has no authentication), so there is one persona — the NAS owner — exercised across distinct usage scenarios rather than multiple roles.

**Persona — Wei:** A Taiwanese NAS owner running a Synology DS920+ with a library of English/foreign films and TV. Pain: Bazarr's zh-TW results are poor and he keeps hunting for subtitles and fixing timing by hand. Goal: Traditional-Chinese subtitles on everything, with zero manual work.

### J1 — Happy path (fully automatic)

Wei adds _Dune: Part Two_ (English audio, embedded English subtitle track). The pipeline detects the text sub, extracts it, routes it as English, and Claude translates it to `.zh-Hant.srt` placed beside the file. That evening he opens the movie in Video Station and the Traditional-Chinese track is simply there, correctly timed. **Reveals:** scan/detect, extraction, language routing, translation provider, delivery, SSE status.

### J2 — Manual on-demand + cost awareness

Wei wants a whole season translated now. He opens the item, triggers translation, sees a **per-season token-cost estimate**, confirms, and watches progress over SSE. **Reveals:** manual trigger, batch, cost estimate, progress reporting.

### J3 — First run / no key (capability gating)

Freshly installed, Wei tries to translate before any Claude key is set. The feature is clearly gated with a "translation needs an API key" message rather than a silent failure (M1: key via env-var; M1.5: the 前往設定 link routes to the key-config page — fixing today's dead loop). He sets the key and it works. **Reveals:** capability-honor gating, env-var/secrets key handling, the fixed settings loop.

### J4 — Series name consistency (order-independent glossary, M2)

Wei translates **episode 5 first** (it's what he's watching), then episode 2 later. Character and place names stay consistent across both because the glossary auto-harvested from episode 5 applies to episode 2 — no manual re-entry, regardless of order. **Reveals:** glossary auto-harvest, per-show accumulation, order-independence.

### J5 — Edge: no text source (M1 skip / M2 ASR)

Wei adds a film that has only a PGS (image) subtitle or no subtitle. In M1 it is marked "no text source" and skipped cleanly; in M2 the ASR fallback generates subtitles from the audio. **Reveals:** source detection, fallback priority (extract > ASR > search), graceful "no source" state.

### Journey Requirements Summary

The journeys reveal these capability areas: subtitle-track **detection**, embedded-text **extraction**, **language routing**, **translation provider** (with cost estimate), **delivery + SSE** status, **capability gating / key handling**, **glossary auto-harvest**, and **ASR fallback / source detection**.

## Domain-Specific Requirements

Self-hosted consumer media — there is **no regulatory / compliance regime**. The genuine domain constraints are:

### Privacy (the self-hosting expectation)

Cloud translation (Claude, the default) sends subtitle **dialogue off the NAS**, which self-hosters may object to on principle. **Mitigation:** the local Tier-2 path (Ollama / FunASR) keeps everything on-box — no key, no data egress — offered to privacy-conscious owners with capable hardware. The UI must state this trade-off transparently.

### Cost governance (LLM spend)

Per-season Claude token cost is bounded; already-translated segments are cached; an estimate is surfaced before batch translation.

### Third-party terms

Claude / OpenAI API usage terms are respected; keys are **the user's own (BYO-key)**, stored in the encrypted secrets service and never logged (slog sanitize).

### Risk mitigations

No dialogue egress occurs unless the owner has supplied a cloud key (capability-gated); a fully-local path is available; no PII is handled beyond the subtitle text already inside the media file.

## Innovation & Novel Patterns

Largely an excellent execution of existing concepts (LLM translation is not new), with three genuine differentiators:

### Detected Innovation Areas

- **Extract-first-for-繁中:** leading with embedded-subtitle extraction + LLM translation (not search) directly attacks Bazarr's known Traditional-Chinese weakness. A reordering, not new tech, but the differentiator.
- **Glossary auto-harvest closed loop (spec §6.5):** the translation LLM's own output auto-populates a per-show, order-independent glossary that then feeds back for cross-episode name consistency. The genuinely novel pattern.
- **Compute-aware hardware tiering:** install-and-it-does-the-right-thing — extract-only vs cloud-ASR vs FunASR-local vs external-worker chosen by detected NAS hardware, plus the insight that non-autoregressive FunASR SenseVoice makes local Chinese ASR viable on weak NAS CPUs where Whisper cannot.

### Market Context & Competitive Landscape

Bazarr = search-first, weak zh-TW. Whisper-based auto-sub tools (whisper-subs / Jellyfin) = ASR-first, English-centric, heavy. Vido's differentiator = extract-first + LLM-translate-to-繁中 + glossary consistency, tuned for Traditional-Chinese self-hosters.

### Robustness Pattern — Two-Stage Translation Guarantee

- **LLM handles SEMANTICS** (meaning, tone, Taiwan phrasing) via a structured prompt — translate to Taiwan Traditional Chinese, text-only, **never touch cue numbering / timestamps** (timestamps are re-stitched programmatically) — plus injected metadata and the per-show glossary.
- **OpenCC s2twp runs as a DETERMINISTIC final pass** on the output, guaranteeing Traditional script even if the LLM leaks Simplified characters (idempotent on already-Traditional text; reuses the existing `converter.go`). **LLM = semantics, OpenCC = orthography.**
- **Per-cue quality gate:** detect Simplified leakage (`detector.go`), empty / echoed cues, and cue-count mismatch → retry only the affected cues; assert output timestamps equal the source cue-by-cue.

### Translation Quality Notes (informs functional requirements)

- **Line segmentation:** for the **extract path (M1), PRESERVE the source subtitle's professional segmentation and timing — translate in place, do NOT re-break.** A CJK line-length norm (~14–16 full-width chars, max 2 lines) applies ONLY to the **ASR path (M2)**, where Whisper produces long run-on cues that must be re-segmented. (Correction: an earlier "≤ N chars" constraint was over-applied to M1 — it is a readability convention, not a technical limit, and is unnecessary when inheriting a professionally-segmented source.)
- **Metadata-aware translation (9R-8):** inject TMDb title / original-title / year / genre / overview / cast / production-countries as context. Quality **depends on a correct TMDb match.**
- **Metadata match correction (dependency):** Vido already ships a Plex-like re-match flow — `ManualSearchDialog` → search TMDb → pick the right result → apply (Story 3.7; `POST /api/v1/metadata/manual-search`). Translation MUST use the corrected metadata; if a match is corrected **after** translation, offer a **re-translate-with-corrected-metadata** entry point (same pattern as the glossary re-translate).

### Validation Approach

Validate the glossary loop by measuring cross-episode name consistency and manual-entry reduction over a multi-episode series; validate extract-first by coverage % on a real English library; validate compute-aware by running M1 on the actual DS920+.

### Risk Mitigation

LLM translation quality varies → the two-stage guarantee + per-cue gate + human spot-check bar + model ladder (Sonnet → Opus for hard lines). Glossary auto-harvest could pollute → unconfirmed/suggested state + the existing confirm workflow. Fallback everywhere: extract > ASR > search; cloud > local; if all fail, a clean "no subtitle" state — never a page failure.

## Technical Requirements (backend pipeline within the web app)

### Project-Type Overview

The feature is a backend data pipeline (Go, `apps/api`) with a thin settings UI (M1.5). It extends the existing Vido web app — **no new browser-support / SEO / accessibility scope** beyond the established design system. **Authentication: none** (v4 single-user).

### Technical Architecture Considerations

- **Pipeline state machine:** scan → probe → (extract | asr) → translate → deliver, with media/task status tracked in SQLite and driven by the existing Go **worker pool** (3–5 goroutines, no external queue — project-context §5) with exponential-backoff retry.
- **ffmpeg / ffprobe layer:** extraction via `ffmpeg -map 0:s -c copy` (I/O-bound); subtitle-stream detection reuses `ffprobe_service.go`.
- **Provider abstraction:** `ASRProvider` (OpenAI-compatible `/v1/audio/transcriptions`) + `TranslateProvider` (Claude default; Ollama optional). External calls follow Rule 27's Five Pillars: one rate-limiter per upstream, cache (translated segments), fail-soft degradation (never fail-page), Rule 7 error codes, BYO keys via the secrets service.
- **SSE:** task status over the existing lazy SSE hub; consumers stay lazy (project-context §8 — no connect-on-mount).

### Endpoints & Data Schemas

- All endpoints `/api/v1/*` with the `{success, data}` envelope. SRT cue model (index, start, end, text) — extraction and translation **preserve timestamps**; output `.zh-Hant.srt` / `.zh-Hans.srt`. Metadata manual-search + apply already exist (`POST /api/v1/metadata/manual-search`); new orchestration endpoints for extraction/translation as needed. Provider request/response contracts (OpenAI-compatible ASR; translation payload carrying the metadata + glossary term-map).

### Error Codes (Rule 7)

Extend the existing `SUBTITLE_` prefix (or add a new pipeline prefix) and register it in project-context Rule 7 + `code-review/instructions.xml` Step 3. Reuse `TRANSCRIPTION_` for ASR and `METADATA_` / `TMDB_` for matching; provider errors map to `{SOURCE}_{ERROR_TYPE}`.

### Rate Limits, Versioning, Auth

External LLM/ASR provider rate limits honored (Rule 27 ①) with cost governance (see Domain-Specific Requirements). `/api/v1` versioning. No auth (v4 single-user).

### Implementation Considerations

All new backend code → `apps/api` (Rule 1); slog only (Rule 2); layered handler → service → repository (Rule 4); **reuse existing engine components (ffprobe detection, OpenCC converter, subtitle placer, SSE hub) — do not rebuild.**

## Project Scoping & Phased Development

### MVP Strategy & Philosophy

**MVP Approach:** Problem-solving MVP aimed at validated learning — the fastest path to prove _extraction + AI translation produces subtitles the owner trusts enough not to hand-edit._ M1 runs the full pipeline with the Claude key supplied via env-var (no config UI), avoiding local ASR entirely (cloud translation is HTTP-only, near-zero NAS load).

**Resource Requirements:** Lean / solo (BMAD dev agent). Cross-stack, Go-backend-heavy with minimal frontend. M1 ≈ 5–6 stories.

### MVP Feature Set (Phase 1 — M1)

**Core journeys supported:** J1 (fully-automatic happy path), J3 (no-key capability gating, env-var), J5 (no-text-source skip).

**Must-have capabilities:**

- Embedded **text** subtitle extraction (`ffmpeg -map 0:s -c copy`; SDH filter; multi-track single-read).
- Language routing (繁 → done · 簡 → OpenCC · English → translate · other → skip).
- Claude translation provider (env-var key; structured prompt; text-only; timestamps re-stitched programmatically).
- **Two-stage guarantee** (LLM semantics + OpenCC s2twp orthography) + **per-cue quality gate** (Simplified-leak / empty / echo / cue-count detection → per-cue retry; timestamp equality assertion).
- Deliver `.zh-Hant.srt` beside the video; task status over existing SSE.
- Docker multi-arch (amd64 + arm64) + Container Manager deployment docs (EN + zh-TW). **English → 繁中 only; runs on DS920+.**

**M1.5 fast-follow:** key-configuration settings UI (fix the dead 前往設定 loop).

### Post-MVP Features

**Phase 2 (Growth — M2):** J2 (manual on-demand + cost estimate), J4 (glossary auto-harvest, order-independent), **ASR fallback** (extract > ASR > online-search), compute-aware auto-defaults, re-translate-after-metadata-correction.

**Phase 3 (Expansion — M3 / Vision):** local Tier-2 models (Breeze ASR 25 / FunASR SenseVoice / Ollama) + model-management UI, additional source languages, burn-in (Quick Sync), PGS OCR, fully-local zero-key generation.

### Risk Mitigation Strategy

**Technical Risks:** translation quality/trust → two-stage guarantee + per-cue gate + human spot-check + model ladder (Sonnet → Opus); **timestamp integrity** → translate text only, re-stitch programmatically, assert per-cue equality; DS920+ resource limits → M1 avoids local ASR (the riskiest assumption — local ASR on a NAS — is deferred to M2/Tier-2 and de-risked via non-autoregressive FunASR); metadata-match correctness → existing `ManualSearchDialog` + re-translate.

**Market Risks:** biggest risk is whether the owner trusts AI-translated subtitles enough to stop manual work → M1 validates exactly this against the real DS920+ library.

**Resource Risks:** lean/solo → M1 is already the smallest viable slice (env-var key, English-only, no ASR); if needed it can shrink further (single-file manual trigger before auto-on-add).

## Functional Requirements

_The capability contract. Phase tags: unmarked = M1 (MVP); [M1.5], [P2] = Growth, [Tier-2] = Expansion._

### A. Subtitle Source Detection & Extraction

- FR1: The system can detect the subtitle tracks in a media file (embedded text, embedded image, external sidecar) and their language.
- FR2: The system can extract an embedded text subtitle track without re-encoding the media.
- FR3: The system can extract multiple text subtitle tracks in a single pass.
- FR4: The system can filter SDH (hearing-impaired) annotations from an extracted subtitle.
- FR5: The system can identify a media item as having no usable text source (image-only or none) and mark it.

### B. Language Routing & Conversion

- FR6: The system can determine a subtitle's language from its content (not its filename).
- FR7: The system can pass an already-Traditional-Chinese subtitle through unchanged.
- FR8: The system can convert a Simplified-Chinese subtitle to Traditional (Taiwan) without AI translation.
- FR9: The system can route a subtitle by detected language (Traditional → done, Simplified → convert, English → translate, other → skip).

### C. AI Translation

- FR10: The system can translate an English subtitle to Traditional Chinese via a translation provider.
- FR11: The system can translate a subtitle while preserving its cue numbering and timestamps exactly.
- FR12: The user can trigger subtitle translation for a media item on demand.
- FR13: The system can translate subtitles automatically when new media is added.
- FR14: The user can see an estimated cost before translating a batch/season. [P2]

### D. Translation Quality Assurance

- FR15: The system can guarantee Traditional script by applying a deterministic conversion pass to the translation output.
- FR16: The system can detect and retry only the individual cues that come back empty, untranslated, or in the wrong language.
- FR17: The system can verify that translated cue timestamps match the source.

### E. Glossary & Cross-Episode Consistency [P2]

- FR18: The system can harvest proper-noun/term mappings from a translation and add them to the show's glossary as unconfirmed suggestions.
- FR19: The user can review, confirm, edit, or delete glossary terms.
- FR20: The system can apply a show's glossary as translation context for any episode of that show, regardless of episode order.

### F. Provider & Key Management

- FR21: The user can supply their own API keys for translation and (optional) ASR providers.
- FR22: The system can perform translation with a locally-hosted provider that requires no external key. [Tier-2]
- FR23: The system can gracefully disable translation with a clear message when no provider/key is configured (no silent failure).
- FR24: The user can select which translation/ASR provider and model is active. [Tier-2]
- FR25: The user can configure and edit provider keys from within the app after initial setup. [M1.5]

### G. Metadata Context & Match Correction

- FR26: The system can supply media metadata (title, genre, overview, cast, country) as context to the translation provider.
- FR27: The user can correct an incorrect TMDb match by searching for and selecting the correct entry.
- FR28: The system can re-translate a subtitle using the corrected metadata after a match is fixed.

### H. Source Fallback [P2]

- FR29: The system can generate subtitles from audio via ASR when no text source exists.
- FR30: The system can prioritize sources as extract > ASR > online-search.
- FR31: The system can route ASR to a cloud API, an external worker, or a local model based on available compute.

### I. Pipeline Operation & Status

- FR32: The system can place the produced Traditional-Chinese subtitle beside the media file for player auto-load.
- FR33: The user can see real-time progress status for subtitle processing.
- FR34: The user can run batch subtitle processing over a scope (missing-only / season / library). [P2]

## Non-Functional Requirements

_Only the categories that matter for this product. Scalability is folded into Performance (single-user, single NAS — "scale" = library size); Accessibility is skipped (new UI inherits the existing design-system a11y baseline)._

### Performance

- NFR-P1: On the standard target (DS920+, J4125 / 4 GB), processing one media item (extract + translate an embedded English sub) must not degrade the responsiveness of other NAS services — extraction is I/O-bound (seconds); cloud translation is HTTP-only (the NAS is near-idle during the call).
- NFR-P2: In M1 the NAS performs no heavy AI compute locally (no local ASR); heavy compute is offloaded (cloud / external worker) or gated by compute-aware defaults.
- NFR-P3: Batch/queue concurrency is bounded per hardware tier (ARM / low-RAM → 1) so a large-library batch does not saturate the NAS.

### Security & Privacy

- NFR-S1: Provider API keys are stored encrypted (secrets service) and are never written to logs (slog sanitize).
- NFR-S2: No subtitle dialogue leaves the NAS unless the owner has configured a cloud provider key (capability-gated); a fully-local path is available (Tier-2).

### Reliability

- NFR-R1: Every external-dependency failure is fail-soft — the pipeline degrades per item to a clear state ("no subtitle" / "translation unavailable"), never a page/app failure.
- NFR-R2: The pipeline is idempotent — re-processing an item does not duplicate or corrupt output, and an existing acceptable `.zh-Hant.srt` is not overwritten without intent.
- NFR-R3: Partial failures recover granularly — per-cue retry for translation; batch operations preserve completed results on cancel and are resumable.

### Integration & Compatibility

- NFR-I1: External providers (Claude / OpenAI, TMDb, local runtimes) are integrated per Rule 27's Five Pillars (rate-limit, cache, degrade, error-codes, keys).
- NFR-I2: The pipeline ships as a single multi-arch (amd64 + arm64) Docker image deployable via Synology Container Manager / QNAP Container Station.
- NFR-I3: Output subtitles use the `.zh-Hant.srt` / `.zh-Hans.srt` sidecar convention so Plex / Jellyfin / Synology Video Station auto-detect them.
