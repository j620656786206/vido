# Epic 9R — Subtitle Route C (繁中 generation core) — Story Breakdown

**Created:** 2026-06-16 (Bob / SM, party-mode)
**Source of truth:** `../planning-artifacts/architecture/adr-subtitle-route-c-generation.md` + `../planning-artifacts/subtitle-v4-replan-and-feasibility-audit-2026-06.md`
**Premise:** Route A fetch confirmed non-viable for 繁中 (Assrt token unobtainable, Zimuku WAF-dead, OpenSubtitles 繁中-thin). Route C generation is the sole 繁中 path, validated end-to-end by a live POC (`../../apps/api/cmd/route-c-poc/`).

**Feasibility-gate rule:** SPIKE-gated stories may NOT be marked `ready-for-dev` until their spike passes. POC-PROVEN stories are immediately actionable.

---

## Spikes (feasibility gates — run before dependent stories)

### S1 — .nfo localization feasibility spike
- **Gates:** 9R-13 (metadata localization)
- **Question:** Can we LLM-localize a `.nfo` and write an additive zh-TW `.nfo` that Kodi/Jellyfin/Plex actually scrape & display?
- **Pass:** a real player shows the translated plot/cast from the zh-TW `.nfo`; original `.nfo` untouched.
- **Effort:** S

### S2 — NAS Whisper benchmark + OpenVINO eval
- **Gates:** D2 default (cloud vs local Whisper), 9R-9 local-engine option
- **Question:** faster-whisper `base`/`small` throughput on the real target NAS CPU; does WhisperLive-OpenVINO usefully use the Intel iGPU?
- **Pass:** measured min/episode for each; a go/no-go on local default.
- **Effort:** S

### S3 — Douban localization fallback spike
- **Gates:** any 在地化 path that leans on Douban metadata
- **Question:** does vido's Douban scraper return real zh metadata live (it JS-renders)?
- **Pass:** end-to-end parse of a real result, else drop Douban from the chain.
- **Effort:** S

---

## Track 1 — Production bug fixes (POC-PROVEN, immediately actionable)

### 9R-1 — Fix stale default Claude model
- **Priority:** P0 · **Effort:** XS · **Feasibility:** PROVEN
- **Context:** POC hit `404 model: claude-3-5-haiku-latest` — Anthropic deprecated it.
- **AC:**
  1. `ai/claude.go` `DefaultClaudeModel` updated to a current, supported model id.
  2. Model id is config-overridable (env), not only a constant.
  3. A test asserts the default is non-empty and the request body carries it; add a guard/log on 404 `not_found_error` that names the bad model.
- **Files:** `apps/api/internal/ai/claude.go` (+ config)

### 9R-2 — Pin Whisper source language from the audio track
- **Priority:** P0 · **Effort:** S · **Feasibility:** PROVEN (option already added)
- **Context:** No `language` param → Whisper mis-detected English-as-Chinese, producing garbage. `WithWhisperLanguage` already added to `whisper.go`.
- **AC:**
  1. The transcription path derives the language from the selected audio track (`eng`→`en`, ISO-639-1) and passes it via `WithWhisperLanguage`.
  2. Fallback to empty (auto-detect) only when the track language is `und`.
  3. Unit test: given a track lang, the multipart body includes `language=<iso2>`.
- **Files:** `apps/api/internal/services/transcription_service.go`, `internal/ai/whisper.go`

### 9R-3 — Fix Whisper chunking 413 (size/duration inconsistency)
- **Priority:** P0 · **Effort:** S · **Feasibility:** PROVEN (POC bypassed via segment muxer)
- **Context:** `NeedsChunking` (size) vs `SplitAudioChunks`/`getWAVDuration` (duration; mis-parses ffmpeg WAV header with extra chunks) disagree → whole oversized file sent → HTTP 413 (limit + multipart overhead).
- **AC:**
  1. Chunking guarantees every chunk POST body (file + multipart overhead) stays **under** 25 MiB (leave headroom).
  2. Duration source made header-robust (or switch to ffmpeg segment muxer); `NeedsChunking` and the splitter agree.
  3. Chunk timestamps offset correctly on merge; test with a >25 MiB WAV asserts N chunks, each < limit, contiguous timestamps.
- **Files:** `apps/api/internal/ai/whisper.go`

### 9R-4 — Retry/backoff on transient ASR/LLM failures
- **Priority:** P1 · **Effort:** S · **Feasibility:** PROVEN (POC added 3× retry; saved a run)
- **Context:** `WhisperClient` has no retry — one transient timeout killed a full transcription.
- **AC:**
  1. Transcription and translation calls retry transient failures (timeout, 5xx, 429) with bounded exponential backoff.
  2. Retries are capped and logged; permanent errors (4xx non-429) do not retry.
  3. Test simulates a transient failure then success.
- **Files:** `apps/api/internal/ai/whisper.go`, `internal/ai/claude.go` (or shared helper) — folds into 9R-11.

### 9R-5 — VAD / hallucination tail filter
- **Priority:** P1 · **Effort:** M · **Feasibility:** PROVEN (POC produced a fake "like & subscribe" outro on silent credits)
- **Context:** Whisper hallucinates text on silence/music (esp. credits).
- **AC:**
  1. Detect & drop hallucinated blocks (repeated boilerplate, blocks over long silence via VAD, or end-of-file outro patterns).
  2. Conservative — must not drop real dialogue (precision over recall); configurable.
  3. Test fixture with a known hallucinated tail asserts removal.
- **Files:** `apps/api/internal/subtitle/` (new post-filter), `internal/ai/whisper.go` (optional VAD)

### 9R-14 — Remove Zimuku provider
- **Priority:** P2 · **Effort:** XS · **Feasibility:** PROVEN (WAF-dead, owner decision D3)
- **AC:**
  1. `providers/zimuku.go` + its registration in the engine removed; tests updated; no dangling refs.
  2. UI no longer lists Zimuku as a source.
- **Files:** `apps/api/internal/subtitle/providers/zimuku*.go`, engine wiring, web source list

---

## Track 2 — Keystone (differentiation — the glossary)

### 9R-6 — `show_glossary` schema + migration
- **Priority:** P0 (keystone) · **Effort:** M · **Feasibility:** PROVEN
- **Context:** Proper-noun drift across runs (隱形戰士/隱形特務; "The Deep"→深海怪物). A per-show glossary is the fix and the differentiator.
- **AC:**
  1. Migration adds `show_glossary` (media-keyed: `term_src`, `term_zh`, `language`, `source` [subtitle|metadata|manual], `confirmed`, timestamps; unique on media+term+lang).
  2. Repository CRUD + lookup-by-media.
  3. Tests for migration + repo.
- **Files:** new migration, `apps/api/internal/models|repository`

### 9R-7 — Generalize TranslationService → `TranslationRequest{Fields, Glossary}`
- **Priority:** P0 (keystone) · **Effort:** M · **Feasibility:** PROVEN · **Depends:** 9R-6
- **Context:** Service is hardwired to subtitle blocks; localization needs the same engine for arbitrary fields + a glossary.
- **AC:**
  1. New generic request type carrying fields + a glossary map; subtitle translation refactored onto it (no behavior regression — existing tests green).
  2. Glossary terms injected into the prompt (do-not-retranslate + use-this-rendering).
  3. Tests: glossary term is honored in output.
- **Files:** `apps/api/internal/services/translation_service.go`, `internal/ai/prompts/`

### 9R-8 — Metadata-aware translation context
- **Priority:** P1 · **Effort:** S · **Feasibility:** PROVEN · **Depends:** 9R-7
- **Context:** Translation ignores show metadata; feeding it lifts quality cheaply.
- **AC:**
  1. Subtitle/metadata translation prompt receives show title + plot + cast/character table when available.
  2. Character names resolved against the glossary (9R-6).
  3. Test: a known character name renders consistently.
- **Files:** `internal/ai/prompts/`, `internal/services/translation_service.go`

---

## Track 3 — Architecture (pluggable engines, anti-lock-in)

### 9R-9 — `ASRProvider` interface + configurable engine/base-URL
- **Priority:** P1 · **Effort:** M · **Feasibility:** PROVEN (vido already OpenAI-API-shaped, has `WithWhisperBaseURL`)
- **Context:** Decouple transcription from the OpenAI API; enable self-hosted Speaches/WhisperLive via base-URL swap.
- **AC:**
  1. `ASRProvider` interface (`Transcribe(audio) → SRT`); `whisper.go` is one impl.
  2. Provider + base URL configurable (cloud OpenAI / self-hosted OpenAI-compatible), mirroring `AI_PROVIDER`.
  3. Doc note: verified MIT engines (Speaches, WhisperLive-OpenVINO, Subgen).
  4. Smoke test against a configurable base URL (mock OpenAI-compatible server).
- **Files:** `apps/api/internal/ai/` (interface + whisper impl), config

---

## Track 4 — Robustness prerequisite

### 9R-11 — AI cost/quota controls
- **Priority:** P0 (prereq for batch) · **Effort:** M · **Feasibility:** PROVEN
- **Context:** `ai/` layer has 429 detection only — no backoff/retry/throttle/token metering. Batch translate/transcribe over a library would runaway-cost and rate-limit-fail.
- **AC:**
  1. Concurrency cap + token-bucket throttle across ASR & LLM calls.
  2. Token-usage + cost metering logged per job; a configurable per-run budget ceiling.
  3. Backoff/retry shared with 9R-4.
  4. Tests for throttle + budget cutoff.
- **Files:** `apps/api/internal/ai/` (shared client middleware)

---

## Track 5 — Orchestration

### 9R-10 — Wire the Route C generation pipeline (single service flow)
- **Priority:** P1 · **Effort:** M · **Feasibility:** PROVEN (POC proved the data flow) · **Depends:** 9R-1..4, 9R-7, 9R-11
- **Context:** Components exist but aren't wired as one automated flow.
- **AC:**
  1. One service: extract audio → transcribe (ASRProvider) → glossary-aware translate → OpenCC → place, with SSE progress.
  2. Trigger conditions defined (manual + on-add; Route A dormant, not a precondition).
  3. Partial-failure resilience (per 9R-4); end-to-end integration test on a fixture media file (mock ASR/LLM).
- **Files:** `apps/api/internal/services/` (orchestrator), handlers, SSE

---

## Track 6 — Metadata localization (Section E — SPIKE-gated)

### 9R-12 — (= Spike S1) .nfo localization spike
- See Spikes / S1. **Effort:** S

### 9R-13 — Metadata localization: localize .nfo + additive zh-TW writeback
- **Priority:** P1 (differentiator) · **Effort:** L · **Feasibility:** SPIKE-GATED (S1) · **Depends:** 9R-6, 9R-7, S1
- **Context:** Category-level differentiator no subtitle tool offers.
- **AC:**
  1. Localize `.nfo` plot/outline, per-episode titles, cast → zh-TW via the shared LLM+glossary infra.
  2. Write back as an **additive parallel zh-TW `.nfo`** (or zh-TW fields), **never overwriting** the original; original backed up/preserved.
  3. Kodi/Jellyfin/Plex scrape & display the zh-TW metadata (verified per S1).
  4. Tests + a manual verification checklist.
- **Files:** `apps/api/internal/services/` (nfo localizer), models, export path

---

## Sequencing (suggested)

1. **Spikes** S1, S2, S3 (parallel).
2. **Bug fixes** 9R-1..4 + 9R-14 (fast, unblock correct generation) + **9R-11** (cost controls, batch prereq).
3. **Keystone** 9R-6 → 9R-7 → 9R-8.
4. **Architecture** 9R-9.
5. **Orchestration** 9R-10.
6. **Localization** 9R-13 (after S1 + keystone).
