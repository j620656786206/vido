# Story 9R-13 — Metadata localization: localize .nfo + additive zh-TW writeback

Status: review

**Epic:** epic-9R-subtitle-route-c (Track 6 — Metadata localization) · **Owner:** dev (Amelia)
**Date:** 2026-07-05 · **Priority:** P1 (differentiator) · **Effort:** L · **Feasibility:** SPIKE-GATED ✅ (S1 passed, #134)
**Depends:** 9R-6 ✅ · 9R-7 ✅ (#137) · S1 ✅ (#134)

## Why

The category-level differentiator no subtitle tool offers: localize a movie's metadata (plot /
title / genres / cast roles) to zh-TW and write it back as an **additive** `.nfo` that Kodi /
Jellyfin / Plex scrape — **without ever overwriting the original**. Reuses the same LLM + glossary
infra as subtitle generation (ADR Decision 6 — one stack, two products).

## S1 re-spec applied (why "additive" means free-slot, not `*.zh-TW.nfo`)

Spike S1 (#134) proved a language-suffixed `Movie.zh-TW.nfo` is invisible to all three players.
The viable additive form is the **free recognized slot**: the two recognized movie-nfo names are
`<basename>.nfo` and `movie.nfo`; writing zh-TW to whichever is free is additive (the player that
prefers the occupied slot keeps showing the original — non-destructive). Jellyfin prefers
`movie.nfo`, Kodi prefers `<basename>.nfo` — opposite, so one layout can't serve both, but the
write is always safe. **Movies-first** (S1: TV names are single-slot → replace-only).

## What shipped

### `NFOLocalizerService` (`internal/services/nfo_localizer_service.go`)
- `LocalizeMovieNFO(ctx, movie)`:
  1. Sources canonical metadata from the DB `models.Movie` (the reader's `NFOData` drops
     genres/cast, so DB is the fuller source) via `movieToNFO`.
  2. Loads the per-show glossary (`GlossaryRepository.LookupByMedia`, fail-soft).
  3. Translates **title + plot + each genre + each cast role** in ONE glossary-aware batch
     (`TranslationService.TranslateRequest`, 9R-7). Preserves originaltitle, person names, year,
     rating, uniqueids. Fail-soft per field (missing translation keeps the original value).
  4. Marshals a zh-TW `MovieNFO` (reuses the 9R-generator structs + `marshalNFO`).
  5. **`writeAdditiveNFO`** — the S1 free-slot strategy:
     - neither slot exists → write `<basename>.nfo` (primary; no original to preserve)
     - original at `<basename>.nfo` → additive write to free `movie.nfo`
     - original at `movie.nfo` → additive write to free `<basename>.nfo`
     - **both occupied** → back up `<basename>.nfo` → `.nfo.orig` (once), then replace. The
       original always survives.
- Returns `{Path, BackupPath, Replaced}`.
- Rule 19-clean: all in `services`, no subtitle import; reuses the same-package NFO generator.

### Endpoint (`internal/handlers/nfo_localizer_handler.go`)
`POST /api/v1/movies/:id/localize-nfo` — 503 when no translation provider, 404 unknown movie,
400 no file path, else `SuccessResponse({path, backup_path, replaced})`. Registered only when the
localizer is live. Rule 7: `NFO_LOCALIZE_DISABLED` / `NFO_LOCALIZE_FAILED` (new endpoint codes).

## Acceptance Criteria

1. ✅ Localize `.nfo` plot / title / genres / cast roles → zh-TW via the shared LLM + glossary
   infra.
2. ✅ Write back as an **additive** zh-TW `.nfo`, **never overwriting** the original; both-occupied
   backs up to `.nfo.orig` first (original preserved).
3. ✅ Kodi/Jellyfin/Plex scrape & display — **verified per S1** (#134: live Jellyfin showed the
   free-slot zh-TW nfo). Real-player re-confirmation → NAS checklist below.
4. ✅ Tests (free-slot matrix + preserved fields + glossary-in-prompt + fail-soft + nil-safe) +
   the manual verification checklist below.

## NAS verification checklist (real players — Alexyu)

- [ ] **Jellyfin (NAS)**: `POST /movies/:id/localize-nfo` on a movie whose original is at
      `<basename>.nfo` → the movie shows the zh-TW title/plot (written to `movie.nfo`); original
      untouched.
- [ ] **Kodi**: same movie with original at `movie.nfo` → zh-TW at `<basename>.nfo` displayed.
- [ ] **Plex** (PMS ≥ 1.43.1, NFO agent): confirm which slot wins; note the watch-state limitation
      (S1 §5).
- [ ] Both-slot case: confirm `<basename>.nfo.orig` backup exists and holds the original.

## Dev Notes

- **TV** localization is a follow-up (single-slot names → replace-only) — filed below.
- The endpoint localizes on demand (manual). Auto-localize on scan/import shares the
  `9R-10b-on-add-autotrigger` policy decision (opt-in vs default).
- Glossary is CONSUMED here (same as 9R-10); population/mining is separate.

### Discovery Triage

- **YES — one follow-up filed (Rule 24 ③):**
  - `9R-13a-tv-nfo-localization` (backlog) — TV `.nfo` (tvshow.nfo / per-episode) localization via
    backup-and-replace (single-slot, no additive option). Needs episode metadata + the replace
    UX/opt-in. Movies shipped here; TV deferred per S1 movies-first re-spec.

### References

- [Source: 9R-S1-nfo-localization-spike.md#§6] — free-slot re-spec + player rules.
- [Source: subtitle-route-c-stories-2026-06.md#9R-13] — ACs.
- [Source: internal/services/nfo_generator.go] — MovieNFO structs + marshalNFO reused.

## Dev Agent Record

### Agent Model Used

claude-fable-5 (dev)

### Completion Notes List

- NFOLocalizerService + POST /movies/:id/localize-nfo shipped: DB-sourced metadata →
  glossary-aware TranslateRequest (title/plot/genres/roles, preserving names/ids/year) → zh-TW
  MovieNFO → S1 free-slot additive write (backup-and-replace only when both slots occupied,
  original always preserved). Movies-first; TV filed as 9R-13a. Full suite + staticcheck green.

### File List

- `apps/api/internal/services/nfo_localizer_service.go` (+ test)
- `apps/api/internal/handlers/nfo_localizer_handler.go`
- `apps/api/cmd/api/main.go` — wire service + handler
- `_bmad-output/implementation-artifacts/sprint-status.yaml`

## Change Log

| Date | Change |
|---|---|
| 2026-07-05 | 9R-13 implemented (dev): NFOLocalizerService (DB metadata → glossary-aware TranslateRequest → zh-TW MovieNFO → S1 free-slot additive write, backup-and-replace fallback, original always preserved) + POST /movies/:id/localize-nfo. Movies-first; TV → 9R-13a. Tests: free-slot matrix, preserved fields, glossary-in-prompt, nil-safe. Full suite + staticcheck green. Status → review. |
