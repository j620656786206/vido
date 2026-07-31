# ADR: Subtitle Route C Generation — Pluggable ASR + Glossary-Centric Localization (Epic 8/9 revision)

> **Status:** ACCEPTED · **AMENDED 2026-07-31** (Decision 1 partially revised — see [Amendment](#amendment-2026-07-31--route-a-is-no-longer-source-less))
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
- **Assrt** — needs an API token; the site is a semi-abandoned 射手(伪) SPA mirror. ~~**Owner verified the token is unobtainable (cannot register).**~~ → **RETRACTED 2026-07-31: the token IS obtainable.** The 2026-06-16 attempt hit a broken registration SPA, not a policy wall. Re-test succeeded. The real constraint is a **quota wall: 5 req/min** (the official docs' 20 req/min is wrong — trust the account panel). See Amendment below.
- **OpenSubtitles** — works, but Asian fansubbers don't upload there → 繁中 coverage is thin (owner domain call).

~~**→ For 繁中, all three fetch sources are non-viable.**~~ → **amended 2026-07-31: not all three.** Route A's failure mode is *external and uncontrollable* (WAF arms-race, abandoned mirrors, ~~registration walls~~ quota walls). Meanwhile the same POC validated the **generation** pipeline end-to-end on a real 4K episode — ffmpeg → Whisper → Claude → OpenCC → write — producing natural Taiwan-繁中, and beating the user's existing (mislabeled-simplified) "human" subtitle on the one dimension that matters: it's actually 繁中.

The fragility is **localized**: a 12-dependency scan showed every *uncontrollable/blocked* dependency is a 華語 community subtitle/metadata source; the commercial/Western deps (TMDB, Wikipedia, LLM APIs, ffmpeg) are healthy or controllable.

---

## Decision 1 — Route C generation is the sole 繁中 path; Route A is de-scoped

> ⚠️ **Partially amended 2026-07-31** — the "sole path" framing survives, the "no sources exist" premise does not. See [Amendment](#amendment-2026-07-31--route-a-is-no-longer-source-less).

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

---

## Amendment 2026-07-31 — Route A is no longer source-less

**Trigger:** three source-feasibility updates recorded in `subtitle-v4-replan-and-feasibility-audit-2026-06.md` PART A.

### What changed

| | 2026-06-16 | 2026-07-31 |
|---|---|---|
| **Assrt** | token unobtainable (cannot register) | ✅ **token obtained** — the earlier blocker was a broken registration SPA, not a policy wall. Real constraint: **5 req/min** (official docs say 20 — wrong, trust the panel) |
| **SubHD** | not evaluated | ⚠️ no official API; community uses HTML scraping (Bazarr+ provider, `subhd.py`, subfinder) — no login/key, no cloudscraper needed. Chinese-first but **zh-Hans/zh-Hant not distinguished**; multiple live domains → scraper maintenance is permanent |
| **shooter.cn** (original 射手網) | not evaluated — the audit only looked at the `assrt.net` mirror | ⚠️ officially shut down 2014, but `api/subapi.php` **still answers HTTP 200** (Cloudflare, `application/octet-stream`, 1-byte `0xFF` body). Hash-based, **no token**. Whether it still serves real subtitles is **unverified** — needs a real-filehash spike (S5) |

### What did NOT change — and why Decision 1 still stands

Decision 1 rested on **two independent legs**, and only one was kicked out:

1. ~~**No usable 繁中 fetch source exists.**~~ → **overturned.**
2. **A fetched subtitle doesn't match the copy you're holding.** → **untouched.** Version matching, global offset, progressive drift, ffsubsync/alass, manual sync UI — that entire concern set is exactly as unsolved as it was on 2026-06-16. Generated subtitles remain inherently time-aligned to the audio they were transcribed from; fetched ones are not.

**Therefore Route C remains the 繁中 mainline.** Restoring fetch to primary status would drag the whole de-scoped sync problem back into scope (see Consequences above) — that trade did not get cheaper just because a token appeared.

### Revised position for Assrt

**Assrt becomes one layer, placed before ASR, on the single-title on-demand path only.** Rationale: 5 req/min is fine for "user clicks 找字幕 on one episode" and structurally incapable of a library-wide batch sweep. The dormancy clause in Decision 1 ("kept for the day a credential appears") has self-triggered — but into a **narrow, quota-bounded role**, not a restoration of fetch-first.

### Immediate code consequence (not optional) — ✅ FIXED 2026-07-31

`internal/subtitle/providers/assrt.go:22` set `assrtRateLimit = 2 // requests per second` = **120 req/min, 24× the real 5 req/min quota**. While the token was unobtainable this was moot; with a token it was a live footgun — the first search would blow the quota. **Fixed same day:** `rate.Every(13 * time.Second)` with burst 1 (worst case in any 60s window = 5 calls), plus `TestAssrtProvider_ProductionRateLimitMatchesQuota` pinning the config so a regression to the old value fails CI. Note the flow cost: one search→detail→download cycle = 3 API calls ≈ 26s of limiter wait — acceptable for on-demand, confirming batch is out. **Spike S4** (ceiling behaviour: hard 429? ban? refill window?) remains open; the `/v1/user/quota` endpoint is live-verified and machine-readable, so the provider could pre-check remaining quota cheaply.

### Live verification 2026-07-31 (both sources)

- **Assrt full chain PASSED with the real token**: `/user/quota` (returns remaining count; 5→4 after one call — direct evidence of the 5/min quota), `/sub/search` (real results incl. `langcht` 繁中 entries), `/sub/detail` (download URL + 9-file filelist incl. `繁体&英文.ass`, release-tagged FLUX/AMZN/WEB-DL).
- **SubHD full chain PASSED with plain curl + UA** (no cloudscraper): SSR search → `/d/` work page → `/a/<sid>` subtitle page → `POST /api/sub/prepare-download` → `/down/<sid>` interstitial → `POST /api/sub/down` → real ZIP from `dl.subhd.me`. Traps found: same-domain session required (cookies don't cross `.tv`/`.me`; cross-domain `/down` → 403 expired), temp download page expires, Cloudflare passive JSD + hidden honeypot link in `<body>`, zh-script only inferable from filenames (简体/繁体).
- **Timeline-alignment hypothesis EMPIRICALLY CONFIRMED (2026-07-31)** — downloaded SubHD sub (cut for `1080p.ATVP.WEB-DL...NTb`) vs the embedded track in the owner's actual NAS file (`2160p.ATVP.WEB-DL...HDR...NTb`, same source + group, different resolution): **399/402 cues matched within 500ms; mean drift 4.6ms, max 9ms** (sub-frame). The 3 unmatched = 1 merged cue + 2 uploader-credit cues. Release-matched fetch needs NO time-sync work — the version-match gate is the whole story.
- **Content caveat from the same comparison:** the file's embedded `Chinese (Traditional)` track is Apple's official zh-Hant (妳/full-width punctuation, native TW phrasing); the SubHD "繁体" variant is the uploader's s2t **conversion** of the simplified track, with simplified leaks (「你在做什么」). Two consequences: (1) fetched "繁中" must still pass OpenCC s2twp (vido's fetch path already does); (2) **when an embedded zh track exists, extraction beats fetching on quality** — supporting extract-中文 as layer 1, fetch as layer 2 for embedded-English-only files.

### ⚠️ Open conflict for the owner to settle

`vido-subtitle-pipeline-spec.md` §優先序 locks **抽內嵌 > ASR > 線上搜尋** as an Alexyu decision, justified by "線上搜尋命中率低". This amendment moves **availability**, not **hit rate** — different axes. Putting Assrt ahead of ASR contradicts that locked order, so **this amendment does not touch the spec**; the ordering call is escalated, not assumed.
