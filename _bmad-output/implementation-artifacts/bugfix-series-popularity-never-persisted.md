# Bugfix: `popularity` never round-trips — series never persist it, and movies write it but never read it back

Status: done

**Origin:** Rule 24 ③ from `bugfix-upsert-zeroes-unloaded-columns` CR **F4** (2026-08-24). The CR saw half of it (series converter sets `Popularity`, series `Create`/`Update` never write it). Authoring this story found the other half is worse: **movies don't round-trip it either** — `movieSelectColumns` omits `popularity`, so the value the wide `Update` and `UpdateEnrichedMetadata` faithfully write is never scanned back, and the next load-then-wide-update **zeroes it**. Backend only.

---

## Story

As the Vido operator,
I want `popularity` to be a real column — written, read back, and surviving edits — for both movies and series,
so that a future "sort by popularity" over the local library has data instead of zeros.

## Root cause (verified 2026-08-24)

| Path | Writes `popularity`? | Reads it back? | Net effect |
|------|---------------------|----------------|------------|
| Movie `Create` | **no** (INSERT column list omits it — verify in Task 1) | — | new rows start NULL |
| Movie wide `Update` (`movie_repository.go:204/:245`) | yes | — | writes whatever the loaded model holds |
| Movie `UpdateEnrichedMetadata` (`enriched_metadata_update.go:83/:98`) | yes | — | enrichment sets it (`enrichment_service.go:660,759`) |
| `movieSelectColumns` (`movie_repository.go:561-572`) + `scanMovie` (`:596`) | — | **NO** | every loaded movie has `Popularity` zero ⇒ any load-then-wide-update (e.g. a user metadata edit) **zeroes the column enrichment just wrote** |
| Series `Create` (`series_repository.go:45-54`) / wide `Update` / `seriesSelectColumns` (`:556-567`) / `scanSeries` (`:588`) | **no** / **no** / — | **no** | converter (`services/converters.go:157,319`) and nothing else touch the model field; the column (migration 006:46) stays NULL forever |

No consumer reads DB `popularity` today (no `ORDER BY popularity` in `apps/api`; frontend popularity usages are TMDb API responses and the explore-block **TMDb** sort key, not DB). So this is a data-integrity fix ahead of use, not a behaviour change.

## Design

| # | Decision | Chosen | Why |
|---|----------|--------|-----|
| D1 | Scope | **Make it round-trip on both entities**: add `popularity` to `movieSelectColumns` + `scanMovie`, series `Create` + wide `Update` + `seriesSelectColumns` + `scanSeries` | The column and the model field exist on both; half-wired is exactly what produced the last three bugfixes. |
| D2 | Series enriched writer | **Leave `UpdateEnrichedMetadata` (series) unchanged** | Series enrichment (`enrichSeries`, `enrichment_service.go:271`) does not compute popularity today; adding the column to a writer whose service never sets it would re-create the "writes a value it never loaded" class. Note it as the follow-up seam if series enrichment ever gains popularity. |
| D3 | Upsert interaction | Nothing to do — converter **produces** popularity, so under #269's ownership contract it is TMDb-owned and correctly NOT preserved; once the wide Update writes it (this story), a re-match refreshes it. Add one line to the movie Upsert contract comment listing popularity explicitly. | Keeps the contract comment exhaustive. |
| D4 | Backfill | **None** | Values refresh on the next enrichment/re-match; a migration to backfill from TMDb would be a network job, not a schema fix. Stated in the PR. |

## Acceptance Criteria

1. **AC #1 — Movie round-trip.** `Create`→`FindByID` and `Update`→`FindByID` both return the stored `Popularity`; and the regression that motivated this: enrichment writes popularity → a wide `Update` from a freshly loaded model **no longer zeroes it**.
2. **AC #2 — Series round-trip.** `Create`→`FindByID` returns `Popularity`; wide `Update` persists it; `Upsert` re-match refreshes it from the incoming TMDb model (D3).
3. **AC #3 — Movie Create parity.** If Task 1 confirms movie `Create` omits the column, it gains it (same INSERT list style); if it already writes it, record that and skip.
4. **AC #4 — Every scan site compiles and passes.** `movieSelectColumns` / `seriesSelectColumns` are used 16 / 14 times — the single `scanMovie` / `scanSeries` helpers gain the field in the correct position; the whole repository package stays green.
5. **AC #5 — No regressions.** `pnpm nx test api` zero FAIL · `nx test web` green · `cost_consent_test.go` diff 0. The #269 Upsert tests stay green (popularity being TMDb-owned means they need no change; if one asserts otherwise, fix the assertion and say why).

## Tasks / Subtasks

- [x] **Task 1 — Movie side** — `movie_repository.go` (+ `enriched_metadata_update.go` comment if its audit list mentions the select-omission)
  - [x] 1.1 Confirm whether movie `Create` writes `popularity`; add if absent (AC #3).
  - [x] 1.2 Add `popularity` to `movieSelectColumns` and `scanMovie` (position must match the column list).
  - [x] 1.3 Regression test: create → enrich-write popularity (use `UpdateEnrichedMetadata` or direct Update) → `FindByID` → wide `Update` of the loaded model → `FindByID` still has the value (the zeroing case, red before this story).
- [x] **Task 2 — Series side** — `series_repository.go`
  - [x] 2.1 Add `popularity` to `Create` INSERT list + binds, wide `Update` SET + binds, `seriesSelectColumns`, `scanSeries`.
  - [x] 2.2 Round-trip test + Upsert-refresh test (AC #2).
- [x] **Task 3 — Contract comment + record** — one line in the movie `Upsert` ownership comment (D3); Dev Agent Record; gates (AC #5).

### Cross-stack split check — Backend 3 / Frontend 0 ⇒ no split.

## Dev Notes

- **Column position discipline**: `scanMovie`/`scanSeries` scan positionally against the `*SelectColumns` strings — add the field in the SAME position in both, and run the whole package, not just the new tests (AC #4 is the guard).
- The movie zeroing regression (Task 1.3) is the same lost-write family as #263–#267 — one test, clearly named, so the class stays visible.
- Do NOT touch series `UpdateEnrichedMetadata` (D2) or add backfill (D4).
- Check the test schemas: `movie_repository_test.go:50` has `popularity REAL` already; verify the series test schema (`series_repository_test.go` setup) has the column — add to the test DDL if missing (migration 006 has it in prod).
- Rule 20: none. Rule 24: bidirectional link (this story ↔ upsert story CR F4) already in sprint-status.

### Time-dependent visual coverage — N/A (backend only).

### References

- [Source: `apps/api/internal/repository/movie_repository.go:204,245,561-572,596`; `series_repository.go:45-54,195-240,556-567,588`]
- [Source: `apps/api/internal/repository/enriched_metadata_update.go:83,98,111`; `services/enrichment_service.go:271,344,660,759`]
- [Source: `apps/api/internal/services/converters.go:157,319`; migration `006_media_entities_enhancement.go:46`]
- [Source: `_bmad-output/implementation-artifacts/bugfix-upsert-zeroes-unloaded-columns.md` CR F4]

## Dev Agent Record

### Agent Model Used

Claude Fable 5 (claude-fable-5) — 2026-08-24.

### Debug Log References

- RED: both round-trip tests failed before wiring (movie `Popularity` came back zero after Create; series after Update).
- Task 1.1 finding: movie `Create` did NOT write popularity — and neither did **BulkCreate** (both entities), which the story's audit table had not listed. Both fixed; the movie test asserts the BulkCreate path too.

### Completion Notes List

- Movie: `popularity` added to `Create` + `BulkCreate` INSERT lists/binds, `movieSelectColumns`, `scanMovie`. Series: `Create` + `BulkCreate` + wide `Update` + `seriesSelectColumns` + `scanSeries` (positionally after `vote_count` everywhere). Series test DDL gained the column (prod migration 006 already has it).
- Upsert contract comment now lists `popularity` as TMDb-owned (D3); a re-match refreshes it — test-pinned.
- D2 held: series `UpdateEnrichedMetadata` untouched (series enrichment computes no popularity).
- Tests: `TestMoviePopularity_RoundTripsAndSurvivesWideUpdate` (Create→read, enrichment-write→load→wide-update does NOT zero — the motivating regression — plus BulkCreate) and `TestSeriesPopularity_RoundTripsAndUpsertRefreshes` (Create→read, Update, Upsert refresh).
- Gates: `pnpm nx test api` zero FAIL · `pnpm nx test web` 237 files green · lint api · format:check · 0 orphaned workers · `cost_consent_test.go` diff 0.
- **Code review (fresh-context agent, same model): PASS, zero findings in the diff** — every INSERT/SET/SELECT/scan walked positionally (33/33/33 movie inserts, 35s series, 42- and 43-column select↔scan walks, 34-bind series Update); the package's `TestEveryReadPathReturnsEveryColumn` guards pass. One pre-existing observation absorbed as **Rule 24 lane ①**: movie `Create`/`BulkCreate` also omitted `vote_count` (converter sets it; first insert lost it until enrichment) — added to both lists + a Create-test assertion, same family, two lines.
- 🔗 AC Drift: N/A (no shipped AC observes DB popularity — no consumer exists yet). 📎 Contract Stamps: NONE. 🎭 A11y: N/A. 🎨 UX: SKIPPED. No backfill (D4) — values refresh on next enrichment/re-match.

### Discovery Triage

- **Did this story discover any work outside its current scope?**
  - ① **vote_count omitted from movie `Create`/`BulkCreate`** (review observation) — absorbed into this story: added to both INSERT lists + Create-test assertion (the added AC is the assertion itself). Same lost-on-first-insert family; no separate entry needed.

### File List

- `apps/api/internal/repository/movie_repository.go`
- `apps/api/internal/repository/series_repository.go`
- `apps/api/internal/repository/series_repository_test.go` (test DDL: `popularity REAL`)
- `apps/api/internal/repository/popularity_roundtrip_test.go` (new)
- `_bmad-output/implementation-artifacts/bugfix-series-popularity-never-persisted.md`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`

## Change Log

| Date | Change |
|------|--------|
| 2026-08-24 | code-review: PASS (all positional walks clean); pre-existing vote_count Create-omission absorbed (lane ①). Status → done. |
| 2026-08-24 | dev-story (Amelia): wired through Create/BulkCreate/Update/SELECT/scan on both entities (BulkCreate was an authoring blind spot, caught in dev); 2 round-trip tests incl. the zeroing regression. Status → review. |
| 2026-08-24 | Story created (create-story). Authoring widened the CR F4 finding: movies also fail to round-trip (`movieSelectColumns` omits popularity ⇒ load-then-wide-update zeroes what enrichment wrote). |
