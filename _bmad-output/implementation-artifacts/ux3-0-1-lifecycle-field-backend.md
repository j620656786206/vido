# Story ux3-0-1 — Item-level lifecycle/subtitle field (backend)

**Epic:** ux3-foundation (UX Redesign Phase 3) · **Status:** review (impl done, tests green)
**Owner:** Dev (Amelia) · **Type:** backend · **FRs:** PH3-F1
**Plan:** `_bmad-output/planning-artifacts/epics.md` (Story 0.1) + `ux-redesign/03-phase3-destination-epic-map.md` (F1)

## Story

As a NAS owner,
I want each movie/series to expose one truthful item-level lifecycle + subtitle state at
list-load,
So that the v2 poster badge (and detail) can show the same durable lifecycle consistently
(N1 — one truthful state machine).

## Decision gate outcome (Task 1 — Rule-24 capability audit, DONE 2026-06-15)

Audit of `apps/api/internal/` (evidence file:line below) settled the **stored-column vs
service-computed** question:

- **SERVICE-COMPUTED, NO new column, NO migration.** All durable badge states are
  derivable from **existing** columns: `parse_status` (`models/parse_status.go:11-16`:
  pending/parsing/success/needs_ai/failed), `subtitle_status` (`:51-59`:
  not_searched/searching/found/not_found), `subtitle_language`, `subtitle_tracks`
  (`models/movie.go:142,145-157`, `models/series.go:73,83,86-91`).
- **The real gap is a Rule-15 SELECT/scan desync** (same class as bugfix-20-1): the
  library-grid `List()` queries select only technical-info + poster columns and use a
  LOCAL scan, omitting `parse_status`/`subtitle_status`/`subtitle_language`:
  - `repository/movie_repository.go` `List()` SELECT ~`:362-373` + local Scan ~`:389-412`
    (vs the complete `movieSelectColumns` ~`:610-622`).
  - `repository/series_repository.go` `List()` SELECT ~`:333-344` + local Scan ~`:360-386`
    (vs `seriesSelectColumns` ~`:589-600`).
  This is why the pilot badge was partial (it only had `subtitle_tracks`).
- **Excluded by capability (NOT omission):**
  - `簡轉繁` / `AI 校正中` are **ephemeral** — `subtitle/engine.go` PipelineStage
    (`:26-36`, `StageConverting`/`StageCorrecting`) broadcasts via SSE only, no DB table
    (`subtitle/batch.go` BatchProgress is in-memory; `018_add_subtitle_fields.go` adds no
    job table). → surfaced by the Activity hub's live SSE (Epic 2), never the load-time
    poster badge.
  - `下載中·%` is **not derivable** for a library item — `models/parse_job.go` `MediaID`
    is nil until parse succeeds; a movies/series row exists only post-parse. → Epic 13/14.

## Acceptance Criteria

**Given** the durable badge states (整理中 ← parse_status, 已入庫 ← parse_status=success,
失敗 ← parse_status=failed, 繁中 ← subtitle_status=found + zh-Hant language, 缺字幕 ←
subtitle_status=not_found),
**When** a movie/series is returned by the **library-list** API,
**Then** `parse_status`, `subtitle_status`, `subtitle_language` are present (currently
absent from `List()`), exposed camelCase at the boundary (Rule 18).

**Given** the Rule-15 SELECT/scan sync requirement,
**When** the three columns are added to each `List()`,
**Then** they are added to BOTH the SELECT column list AND the local row Scan in
`movie_repository.go` and `series_repository.go` (the exact desync that caused bugfix-20-1),
**And** a repository/integration test against a **real sqlite DB** seeds rows with known
parse_status/subtitle_status and asserts `List()` returns them (not a mocked-repo
false-green — Epic 20 lesson).

**Given** the audit showed the durable fields already exist and the response boundary
transforms snake→camelCase (Rule 18),
**When** a movie/series is listed,
**Then** the raw durable fields (parseStatus / subtitleStatus / subtitleLanguage) reach the
client via the List fix — **no backend computed descriptor, no stored column, no migration**
(descoped, see Task 4). The canonical status→token badge mapping is a shared FE util built
in ux3-0-2 (DL-v2 §2.5).

**Given** the ephemeral/underivable states,
**When** the field is computed,
**Then** `簡轉繁`/`AI 校正中` and `下載中·%` are explicitly OUT of scope (documented above)
— the field does not fabricate them.

## Tasks

1. [x] **Task 1 — Rule-24 capability audit + decision gate** (DONE; outcome above).
2. [x] **Movie list Rule-15 fix** — added `parse_status`, `subtitle_status`,
   `subtitle_language` to `movie_repository.go` `List()` SELECT + local Scan. (Single-row
   `movieSelectColumns`/`scanMovie` already had them; only `List()` was missing them.)
3. [x] **Series list Rule-15 fix** — same in `series_repository.go` `List()` SELECT + Scan.
4. [x] **Service-computed descriptor — DESCOPED (decided).** Durable fields now reach the
   client camelCase via the Rule-18 response boundary transform; no backend descriptor, no
   migration. The status→token badge mapping is a shared FE util in ux3-0-2 (DL-v2 §2.5) —
   building it backend-side would duplicate that mapping for zero N1 gain (truthful DATA
   already comes from one backend).
5. [x] **Tests** — real-sqlite integration guards `TestMovieListReturnsLifecycleFields` +
   `TestSeriesListReturnsLifecycleFields` (seed known states → assert `List()` returns them;
   fail pre-fix). Repository pkg green; `go build/vet ./...` clean.

> **Split note:** this is the backend half. The frontend badge consuming the field is
> **ux3-0-2** (separate FE story; consumes this, no forward dependency).

## Dev notes

- Rule 4 (handler→service→repo→DB) holds; Rule 18 case-transform at boundary; Rule 15
  SELECT/scan sync is the crux. No new error prefix (Rule 7).
- N1 convergence (Alexyu 2026-06-14): poster badge = durable states only; process states →
  Activity SSE; download% → Epic 13/14.
- Verify the existing single-row path (`movieSelectColumns`/`scanMovie`) already has these
  columns — only `List()` is missing them; do not duplicate.
