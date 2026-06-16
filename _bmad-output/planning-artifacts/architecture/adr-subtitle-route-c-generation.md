# ADR: Subtitle Route C Generation — Pluggable ASR + Glossary-Centric Localization (Epic 8/9 revision)

> **Status:** ACCEPTED
> **Date:** 2026-06-16
> **Deciders:** Alexyu (product owner), Winston (architect), John (PM)
> **Origin:** `subtitle-v4-replan-and-feasibility-audit-2026-06.md` — live POC + feasibility audit (party-mode planning session)
> **Related PRD:** P1-010…P1-021 (subtitle), new Section E (metadata localization)
> **Related epics:** `epic-8-subtitle-engine.md`, `epic-9-ai-subtitle-enhancement.md`, Phase-3 `epics.md` Epic 6/7 (ux3-subtitle-v2 / ux3-ai-subtitle)
> **Supersedes:** the "fetch-first" premise of Epic 8 (for 繁中)
> **Builds on:** `adr-external-api-integration-standard.md` (Five Pillars), `adr-media-info-nfo-pipeline.md` (metadata model)

---

## Context

Epic 8 shipped a **fetch-first** subtitle engine: search Assrt/Zimuku/OpenSubtitles → score → download → OpenCC convert → place. Its premise was that 繁中 human subtitles are abundantly fetchable, with AI generation (Epic 9, P3) as an optional fallback.

A 2026-06-16 **live POC** (real network, vido's own provider code, `apps/api/cmd/route-c-poc/`) disproved that premise for 繁中:

- **Zimuku** — hardcoded `zimuku.org` sits behind a Yunsuo anti-bot WAF → every query returns `ErrCaptchaDetected`. Effectively dead.
- **Assrt** — needs an API token; the site is a semi-abandoned 射手(伪) SPA mirror. **Owner verified the token is unobtainable (cannot register).**
- **OpenSubtitles** — works, but Asian fansubbers don't upload there → 繁中 coverage is thin (owner domain call).

**→ For 繁中, all three fetch sources are non-viable.** Route A's failure mode is *external and uncontrollable* (WAF arms-race, abandoned mirrors, registration walls). Meanwhile the same POC validated the **generation** pipeline end-to-end on a real 4K episode — ffmpeg → Whisper → Claude → OpenCC → write — producing natural Taiwan-繁中, and beating the user's existing (mislabeled-simplified) "human" subtitle on the one dimension that matters: it's actually 繁中.

The fragility is **localized**: a 12-dependency scan showed every *uncontrollable/blocked* dependency is a 華語 community subtitle/metadata source; the commercial/Western deps (TMDB, Wikipedia, LLM APIs, ffmpeg) are healthy or controllable.

---

## Decision 1 — Route C generation is the sole 繁中 path; Route A is de-scoped

繁中 subtitles come from **generation (transcribe → LLM-translate)**, not fetching. Route A code (Assrt/OpenSubtitles providers) stays **dormant** — kept for the day a credential appears, but **not a planning dependency and never surfaced in UI as a reliable path**. **Zimuku provider is removed** (WAF-dead). Consequence: the entire fetch-side concern set (version-matching scorer, time-sync/offset/drift, ffsubsync/alass, manual sync UI) is **de-scoped for 繁中** — generated subtitles are inherently time-aligned to the audio they're transcribed from.

## Decision 2 — ASR is a pluggable engine behind an `ASRProvider` interface (anti-lock-in)

The Whisper *model* is MIT-open (cannot be un-released); only the OpenAI *API* carries lock-in/cost risk. Therefore transcription is treated as a **replaceable commodity**:

- Define `ASRProvider` (`Transcribe(audio) → SRT`); `ai/whisper.go` becomes one implementation.
- Make provider + base URL configurable (cloud OpenAI / self-hosted), mirroring the existing `AI_PROVIDER` pattern.
- vido's Whisper client already targets the OpenAI-standard `/v1/audio/transcriptions` and exposes `WithWhisperBaseURL` → swapping to a self-hosted **OpenAI-compatible** server is largely a base-URL change.
- Candidate engines (LICENSE-verified 2026-06-16, all **MIT**, commercial-OK): **Speaches** (faster-whisper-server), **WhisperLive** (Collabora — has an **OpenVINO** backend for Intel CPU/iGPU, relevant to NAS hardware), **Subgen** (also exposes OpenAI-compatible `/v1/audio/{transcriptions,translations}` + Plex/Jellyfin/Emby integration → usable as a drop-in backend). Avoid copyleft (Bazarr GPL-3.0, LibreTranslate AGPL-3.0) for bundling; pyannote diarization models are HF-gated.
- **Do NOT integrate ArcSub-the-app** (React+Express monolith, no API, different stack) — study it, don't embed it.

## Decision 3 — Glossary-centric translation (the keystone)

A **per-show glossary** is the differentiator no OSS provides and the fix for proper-noun drift (POC showed the same title translated differently across runs: 隱形戰士/隱形特務, 深海之潮/洶湧狂潮, 透視人/透明人; and "The Deep" rendered as 深海怪物 because the model lacked the character roster).

- New `show_glossary` table (media-keyed term ↔ zh-TW, source, confirmed flag).
- Generalize `services.TranslationService` from subtitle-blocks-only to `TranslationRequest{Fields, Glossary}`.
- Feed show metadata (title, plot, cast/character table) into the translation prompt as context.
- **This same infra serves BOTH subtitle translation AND .nfo metadata localization** — one stack, not two.

## Decision 4 — Translation stays cloud LLM; Whisper cloud-default + local opt-in

"Controllable" ≠ "local" — a paid commercial API is controllable (not at the mercy of a WAF). On a GPU-less NAS, local LLM 繁中 quality/speed is inadequate, so **translation stays cloud** (Claude/Gemini). Whisper **defaults cloud, with local faster-whisper/OpenVINO as an opt-in** (privacy/offline; user accepts slowness). **Gated on spike S2** (benchmark on the real NAS; NAS iGPU does not accelerate via faster-whisper but may via OpenVINO).

## Decision 5 — Fix the 4 production bugs + add VAD hallucination filter (POC-surfaced, mock-invisible)

1. `ai/claude.go:18` default model `claude-3-5-haiku-latest` → 404 (deprecated) — update default.
2. `ai/whisper.go` no `language` param → mis-detection (English mis-transcribed as Chinese) — `WithWhisperLanguage` added; wire the audio-track language (eng→en) through.
3. `ai/whisper.go` chunking: `NeedsChunking` (size) vs `SplitAudioChunks`/`getWAVDuration` (duration, mis-parses ffmpeg WAV header) disagree → oversized file → HTTP 413 — fix (segment muxer / header-robust duration / size headroom for multipart overhead).
4. `ai/whisper.go` no retry → single transient timeout kills the run — add retry/backoff (folds into the AI cost/quota-control work).
5. Whisper hallucination on silence/credits (POC produced a fake "like & subscribe" outro) → VAD / tail-detection post-filter.

## Decision 6 — Metadata localization (Section E) reuses the same infra

Localize `.nfo` plot/episode-titles/cast → zh-TW via the **same** LLM+glossary stack; write back as an **additive parallel zh-TW `.nfo`** (preserve original, never overwrite) for Kodi/Jellyfin/Plex scraping. **Gated on spike S1.** This is the category-level differentiator over standalone subtitle tools.

---

## Consequences

- **Smaller, more focused scope:** fetch-side stories (version-match, time-sync, sync UI, fetch orchestration) drop out for 繁中.
- **Anti-lock-in by construction:** transcription engine is swappable; only the LLM-translation step is a deliberate cloud dependency, itself provider-pluggable.
- **One infra serves two products** (subtitle + metadata localization) via the glossary keystone.
- **Prerequisites:** AI cost/quota controls (backoff/retry/throttle/token metering) before any batch run; spikes S1 (.nfo localization), S2 (NAS Whisper benchmark + OpenVINO), S3 (Douban fallback) gate their dependent specs.
- **Codify as** project-context.md rule update (subtitle generation standard) — follow-up.
