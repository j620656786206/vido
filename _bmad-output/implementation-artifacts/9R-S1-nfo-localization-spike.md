# Spike 9R-S1 — `.nfo` Localization Feasibility

Status: done

**Epic:** epic-9R-subtitle-route-c (Spike S1) · **Gates:** 9R-13 (metadata localization write-back)
**Owner:** dev (Amelia) · **Date:** 2026-07-05 · **Effort:** S

## Question (from `subtitle-route-c-stories-2026-06.md` §Spikes/S1)

> Can we LLM-localize a `.nfo` and write an **additive zh-TW `.nfo`** that Kodi / Jellyfin /
> Plex actually scrape & display?
> **Pass:** a real player shows the translated plot/cast from the zh-TW `.nfo`; original
> `.nfo` untouched.

## TL;DR verdict — **PASS, with a re-spec of "additive"**

- ❌ **Literal additive (`Movie.zh-TW.nfo`) is DEAD** — no player has language-variant nfo
  naming; Jellyfin live-test T1 confirmed the file is completely ignored.
- ✅ **Free-slot additive PASSES, live-verified on Jellyfin 10.11.11**: with the original EN
  nfo untouched at `<filename>.nfo`, writing the zh-TW nfo to the free `movie.nfo` slot made
  Jellyfin display **全面啟動** + zh-TW plot + zh-TW genres (科幻/動作) + zh-TW cast
  (李奧納多·狄卡皮歐 as 唐姆·柯布) — translated plot AND cast from OUR file, original intact
  (T3/T4 + integrity check).
- ⚠️ **Player priorities are OPPOSITE** — Jellyfin: `movie.nfo` beats `<filename>.nfo`
  (empirical, undocumented; orientation-independent per T2 vs T3). Kodi: `<VideoFileName>.nfo`
  beats `movie.nfo` (documented). Consequence: one layout cannot show zh-TW on BOTH players
  simultaneously — but free-slot writing is **always non-destructive** (the player that
  prefers the original's slot simply keeps showing the original language).
- Plex (native NFO Agent, PMS ≥ 1.43.1) reads the same two names; its tiebreak is untested →
  NAS checklist §5.

## 1. What vido already has (code recon, 2026-07-05)

- **Read path (shipped, ADR 2026-04-03):** `apps/api/internal/services/nfo_reader_service.go`
  parses Kodi-format XML (`<movie>`/`<tvshow>`/`<episodedetails>`) + bare-URL nfo; discovery is
  `findNFOSidecar()` = `<video-basename>.nfo` only. Priority chain: `manual(100) > nfo(80) >
  tmdb(60) > …` (`adr-media-info-nfo-pipeline.md` Decision 5).
- **Write path (shipped):** `apps/api/internal/services/nfo_generator.go` already emits
  Kodi-compatible `MovieNFO`/`SeriesNFO` XML (title/originaltitle/year/plot/genre/director/
  actor/uniqueid/rating) via the export service — **9R-13's writer does not start from zero**;
  it reuses this generator with translated field values.

## 2. Player `.nfo` discovery rules (research, 2026-07-05)

| Player | Recognized names (movies) | TV | Language-variant naming | Two-file tiebreak |
|---|---|---|---|---|
| **Kodi** | `<VideoFileName>.nfo`, `movie.nfo` (movie-per-folder) | `tvshow.nfo` + `<episode>.nfo` | ❌ | **`<VideoFileName>.nfo` wins** (documented, kodi.wiki/view/NFO_files/Movies) |
| **Jellyfin** | `movie.nfo`, `VIDEO_TS.nfo`, `<filename>.nfo` | `tvshow.nfo`, `season.nfo`, `<episode>.nfo` | ❌ | **`movie.nfo` wins** (UNDOCUMENTED — established empirically by this spike, T2+T3, 10.11.11) |
| **Plex** | native NFO Agent (PMS ≥ 1.43.1): `movie.nfo` or exact-filename `.nfo` | `tvshow.nfo` + per-episode matching filename | ❌ | untested (NAS checklist) |

Plex caveat: NFO-agent libraries lose Sync Watch State & Ratings (documented limitation) —
a user-facing trade-off 9R-13 must surface, not a blocker.

## 3. The LLM-localization half (already proven)

Producing the translated field values is the Route C POC's proven capability (ADR Decision 3/6:
same `TranslationService` + glossary infra). This spike hand-crafted the zh-TW fixture
(title/plot/genres/director/actors localized; `uniqueid`/`year`/`rating` preserved) — the open
question was ONLY player scraping behavior, and it is now answered.

## 4. Live Jellyfin verification (Docker, jellyfin/jellyfin:latest = 10.11.11)

Method: local container; `/media/movies/Inception (2010)/` with a real 2s mp4 (ffmpeg testsrc)
+ EN `movie.nfo`; movies library with `LocalMetadataReaderOrder:["Nfo"]` and per-type
`MetadataFetchers:[]` (internet OFF — verified: People contained ONLY our nfo names); scrape
state read via `GET /Items` API; each test state = fresh library on a clean DB (item-level
`Refresh` does NOT re-read nfo for unchanged files, and a contaminated DB leaks stale People —
two methodology traps documented for the NAS re-run).

| # | Files present | Result (10.11.11) |
|---|---|---|
| T0 | `movie.nfo` (EN) | `Inception` + EN plot/genres/cast — nfo read ✓ |
| T1 | + `Inception (2010).zh-TW.nfo` | **ignored** — still EN (language-suffixed name unrecognized) |
| T2 | + `Inception (2010).nfo` (zh-TW), `movie.nfo` (EN) kept | still EN → **`movie.nfo` wins** |
| T3 | EN moved to `<filename>.nfo` (byte-identical), zh-TW at `movie.nfo` | **全面啟動 + zh-TW plot/genres/cast** → movie.nfo wins regardless of orientation; **= the PASS case** (original untouched at a recognized name) |
| T4 | zh-TW `movie.nfo` only | 全面啟動 ✓ (swap-scenario sanity) |
| — | Integrity | original EN nfo INTACT throughout (grep + md5) |

## 5. NAS verification checklist (real players — Alexyu)

- [ ] **Jellyfin (NAS)**: repeat T3 layout on the real server → 全面啟動 shown. (Also confirms
      the NAS version matches 10.11 behavior.)
- [ ] **Kodi**: mirror case — original at `movie.nfo`, zh-TW at `<filename>.nfo` → zh-TW shown
      (documented rule; quick confirm).
- [ ] **Plex (PMS ≥ 1.43.1)**: NFO-agent library; test which name wins when both exist; note
      the watch-state limitation.

## 6. Implications for 9R-13 (the gated story) — spike-informed re-spec

1. **Write target = the FREE recognized slot**, not a language-suffixed file:
   - Original at `<filename>.nfo` (vido's own `findNFOSidecar` convention → the common case
     for vido-managed libraries) → write zh-TW to `movie.nfo`. **Jellyfin shows zh-TW
     (proven); Kodi keeps EN (non-destructive).**
   - Original at `movie.nfo` → write zh-TW to `<filename>.nfo`. **Kodi shows zh-TW
     (documented); Jellyfin keeps EN (proven).**
   - Both slots occupied → no free slot: fall back to **opt-in backup-and-replace**
     (`*.nfo` → `*.nfo.orig` + write zh-TW) — preserves the original file, not in-place.
2. **Surface per-player visibility honestly** in UI/docs: free-slot writing is always safe,
   but WHICH player shows zh-TW depends on slot layout (opposite priorities). Full
   both-player coverage requires the opt-in replace mode.
3. **TV shows:** `tvshow.nfo` + `<episode>.nfo` are single-slot names (no movie.nfo-style
   alternate) → additive slot play does NOT exist for TV; TV localization is
   backup-and-replace only (or Jellyfin-only via… n/a). 9R-13 must scope movies-first.
4. **vido re-scan interplay:** vido's reader only discovers `<basename>.nfo`; if zh-TW lands
   there (case 1-original-at-movie.nfo), vido re-scans read the zh-TW file as
   `metadata_source=nfo` (priority 80). Acceptable (it IS the preferred display language) but
   must be a conscious 9R-13 decision + test.
5. **Writer reuse:** extend `nfo_generator.go` structs (plot/genre/actor already exist);
   translated values come from the 9R-7 `TranslationRequest{Fields, Glossary}` path.

## Artifacts

- Fixtures + automated test script: session scratchpad `nfo-spike/` (`run-jf-test.sh`,
  EN/zh-TW nfo pair) — reproducible on the NAS by pointing the same layout at a test library.
- Sources: kodi.wiki NFO_files/Movies · jellyfin.org docs metadata/nfo · support.plex.tv
  "Using NFO Metadata Files with Plex" (native NFO Agent, PMS ≥ 1.43.1).

## Change Log

| Date | Change |
|---|---|
| 2026-07-05 | Spike started (Amelia): code recon + three-player discovery-rule research; zh-TW fixture built. |
| 2026-07-05 | Live Jellyfin 10.11.11 Docker verification complete (T0–T4): literal `*.zh-TW.nfo` dead; free-slot additive PASSES (全面啟動 + zh-TW plot/cast displayed, original intact); Jellyfin tiebreak = movie.nfo wins (opposite of Kodi). Verdict PASS with re-spec; 9R-13 unblocked with revised write strategy. Status → done. |
