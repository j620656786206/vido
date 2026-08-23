# Bugfix: audit every other caller of the wide `Update` for the stale-copy hazard; narrow the ones that do not own the row

Status: done

**Origin:** Rule 24 ③ from 9R-10b CR-249 finding B (2026-08-21). That fix moved **EnrichmentService** onto the narrow `UpdateEnrichedMetadata`. The wide `MovieRepository.Update` (37 columns, incl. the 5 subtitle-delivery columns, `file_path`, `file_size`, `library_id`, `is_removed`) and `SeriesRepository.Update` (32 columns) still have other production callers, each with its own "load → work → write the whole row back" window. The entry asked for a per-site audit with its own table and regression tests rather than a blind sweep. This story is that audit, plus the fixes the audit justifies. Backend only.

---

## Story

As the Vido NAS operator,
I want background jobs that change one or two columns of a movie/series row to write only those columns,
so that a subtitle delivered (or metadata enriched) while a scan's removed-file pass or size aggregation is running is never silently reverted by a stale in-memory copy.

## Audit (2026-08-24, verified against code — the deliverable the backlog entry asked for)

Column facts: `MovieRepository.Update` (`movie_repository.go:168`) writes 37 columns incl. subtitle×5 / file_path / file_size / library_id / is_removed. `SeriesRepository.Update` (`series_repository.go:173`) writes 32 — **no** subtitle columns, but file_path / file_size / library_id / is_removed / parse_status / credits. `EpisodeRepository.Update` has **zero** production callers (`Upsert` only). Existing narrow writers: `UpdateEnrichedMetadata`, `UpdateSubtitleStatus`, `UpdateSubtitleGenerationStatus`, `UpdateDoubanRating`, `Episode.UpdateEpisodeSubtitleStatus`. None covers `parse_status`, `file_size`, `is_removed`, or `poster_path`.

| # | Site | Trigger | Load → Update window | Fields the caller actually assigns | Verdict | Action |
|---|------|---------|----------------------|-----------------------------------|---------|--------|
| 1 | `scanner_service.go:492` `processVideoFile` | scan (HTTP / scheduler / poller) | µs — `FindByFilePath` `:472`, compare, write | `FileSize`, `ParseStatus=pending`, `UpdatedAt` | **C→narrow** (short window, but a 2-field intent writing 37 columns during the very scan whose callback starts the subtitle lane) | `UpdateScanFileInfo(ctx, id, fileSize, parseStatus)` |
| 2 | `scanner_service.go:660` `detectRemovedFiles` | post-scan step `:282` | **ms → seconds+**: `FindAllWithFilePath` loads ALL movies at `:634`, then per-row `os.Stat` on the NAS `:644`; row N's copy is as old as all stats before it. Runs right after scan-complete — i.e. **concurrently with enrichment and the 9R-10b free lane** | `IsRemoved=true`, `UpdatedAt` | **B** | `MarkRemoved(ctx, id)` (movie) — writes `is_removed, updated_at` only |
| 3 | `scanner_service.go:737` `aggregateSeriesFileSizes` | post-scan step `:293` | **ms → seconds+**: `seriesRepo.List(10000)` at `:697`, then per-series `FindBySeriesID` + `os.Stat` per episode `:723` | `FileSize`, `UpdatedAt` | **B** (series Update has no subtitle columns, but clobbers `is_removed`, `library_id`, `parse_status`, `credits`… from the stale copy — enrichment writes those concurrently) | `UpdateFileSize(ctx, id, size)` (series) |
| 4 | `library_service.go:794` / `:802` `BatchReparse` | HTTP `POST /library/batch/reparse` | µs — `FindByID` immediately before | `ParseStatus="pending"` | **C→narrow** (1-field intent) | `UpdateParseStatus(ctx, id, status)` (movie + series) |
| 5 | `metadata_edit_service.go:153` / `:242` `updateMovieMetadata` / `updateSeriesMetadata` | HTTP `PUT /media/:id/metadata` | µs — `FindByID` `:86`/`:175`, pure assignment | title, original_title, release/first_air date, genres, credits, overview, poster_path, metadata_source=manual | **A** — user edit of many metadata fields; server re-loads fresh, no long window | keep wide; document |
| 6 | `metadata_edit_service.go:317` / `:325` `updatePosterPath` | HTTP poster upload / fetch-from-URL | µs — the slow `http.Get` + image processing happens **before** `FindByID` `:311`/`:319` | `PosterPath`, `UpdatedAt` | **C→narrow** (1-field intent) | `UpdatePosterPath(ctx, id, path)` (movie + series) |
| 7 | `movie_service.go:114` `MovieService.Update` | HTTP `PUT /movies/:id` | µs — handler `GetByID` then JSON bind | title, original_title, release_date, genres, overview, poster_path, rating, runtime, status | **A** | keep wide; document |
| 8 | `series_service.go:138` `SeriesService.Update` | HTTP `PUT /series/:id` | µs | title, …, number_of_seasons/episodes, status, in_production | **A** | keep wide; document |
| 9 | `movie_repository.go:1096` / `series_repository.go:993` `Upsert` | only `LibraryService.SaveMovieFromTMDb` / `SaveSeriesFromTMDb` — **no production callers** (tests + a comment) | n/a | whole row from `ConvertTMDbMovieToModel`; **writes zero-values for subtitle×5, library_id, is_removed, file_size it never loaded** | not a stale-copy hazard; a "zeroes unloaded columns" hazard if ever wired | out of scope → tracked as `bugfix-upsert-zeroes-unloaded-columns` (Rule 24 ③, filed at creation) |

Optimistic concurrency: none of the PUT bodies carries `updated_at` / version / `If-Match` (`metadata_handler.go:275-285`, `movie_handler.go:53`, `series_handler.go:63`). Not this story's problem — the A sites have µs windows — but it is why they stay "A" and not "safe forever".

## Design

- **One narrow writer per intent, not per caller**, in `apps/api/internal/repository/scan_state_update.go` (movie + series side by side, the `enriched_metadata_update.go` shape with the same file-header audit record):
  - `UpdateScanFileInfo(ctx, id string, fileSize int64, parseStatus string)` — movie (site 1)
  - `MarkRemoved(ctx, id string)` — movie (site 2) → `is_removed = 1, updated_at = ?`
  - `UpdateFileSize(ctx, id string, fileSize int64)` — series (site 3)
  - `UpdateParseStatus(ctx, id, status string)` — movie + series (site 4)
  - `UpdatePosterPath(ctx, id, path string)` — movie + series (site 6)
- Each writer: `UPDATE … SET <cols>, updated_at = ? WHERE id = ?`; error on nil/empty id; **`RowsAffected == 0` → `sql.ErrNoRows`-wrapped error** (a narrow write to a vanished row must not be silent — the wide Update's behaviour here is checked at Task 1.1 and mirrored).
- Interfaces (ruled at review, M1): the scanner and library services consume `repository.MovieRepositoryInterface` / `SeriesRepositoryInterface` directly — that IS their consumer-side interface, and `UpdateEnrichedMetadata` (9R-10b) was added there for the same reason. The new writers go on those two interfaces (accepted exception to "don't widen the shared interface": introducing per-service interfaces for the scanner/library would be a cross-cutting refactor, not this fix). The metadata-edit service has its own `MovieMetadataRepository` / `SeriesMetadataRepository` and gets only `UpdatePosterPath`. Cost, stated: three unrelated test fakes grew no-op methods.
- **Characterization test per B site** (the `TestWideUpdate_LosesConcurrentSubtitleWrite` pattern, `enriched_metadata_update_test.go:65`): prove the wide path clobbers, then prove the narrow path does not. Sites 2 and 3 get one each; the C→narrow sites get a plain "writes only these columns" test (assert the other columns are untouched after a narrow write on a row whose other columns were changed underneath).

## Acceptance Criteria

1. **AC #1 — Site 2 (`detectRemovedFiles`)**: Given a movie whose `subtitle_status` is changed by another writer after `FindAllWithFilePath` loaded it, when the removed-file pass marks it removed, then `is_removed = 1` AND the concurrent `subtitle_status` / `subtitle_path` survive. A characterization test shows the previous wide path loses them.
2. **AC #2 — Site 3 (`aggregateSeriesFileSizes`)**: Same shape for series: a concurrent `UpdateEnrichedMetadata` (e.g. `parse_status`, `poster_path`) made after `List` survives the size write.
3. **AC #3 — Sites 1, 4, 6** use the new single-intent writers; a test per writer asserts only the intended columns (+ `updated_at`) change.
4. **AC #4 — Zero-row narrow writes error** (`errors.Is(err, sql.ErrNoRows)`), matching whatever the wide `Update` does today (Task 1.1 records the fact; if wide Update is silent on 0 rows, the narrow writers are still strict — the story chooses strict and says so).
5. **AC #5 — No remaining wide `Update` caller outside the A sites**: `grep -n "movieRepo.Update(\|seriesRepo.Update(\|\.repo.Update(ctx, movie)\|\.repo.Update(ctx, series)" apps/api/internal/services/` returns exactly sites 5, 7, 8 (and `Upsert`'s internal call). Each A site gets a one-line comment naming this story and why it keeps the wide writer.
6. **AC #6 — Behaviour otherwise unchanged**: scanner SSE/progress semantics, `BatchReparse` response, poster upload flow — existing tests green; `pnpm nx test api` zero FAIL; `cost_consent_test.go` untouched.

## Tasks / Subtasks

- [x] **Task 1 — Repository narrow writers** (AC #3, #4) — `apps/api/internal/repository/scan_state_update.go` (+ `_test.go`)
  - [x] 1.1 Record in the file header what the wide `Update` does on 0 rows affected (read `movie_repository.go:168-…`); implement the five writers (movie: `UpdateScanFileInfo`, `MarkRemoved`, `UpdateParseStatus`, `UpdatePosterPath`; series: `UpdateFileSize`, `UpdateParseStatus`, `UpdatePosterPath`) with strict 0-row errors.
  - [x] 1.2 Tests on the real SQLite test DB (follow `enriched_metadata_update_test.go` setup): per writer, seed a row, change an unrelated column underneath (e.g. `UpdateSubtitleStatus`), call the narrow writer, assert the unrelated column survived and only the intended columns changed; plus the 0-row case.
- [x] **Task 2 — Site 2 `detectRemovedFiles`** (AC #1) — `scanner_service.go:634-665`, its repo interface, `scanner_service_test.go`
  - [x] 2.1 Add `MarkRemoved` to the scanner's movie-repo interface; replace `existing.IsRemoved = true; Update(existing)` with `MarkRemoved(ctx, existing.ID)`.
  - [x] 2.2 Characterization test: fake repo that records writes; wide path loses a subtitle write made after load (pin with the old code path in a table row or a comment-referenced prior commit), narrow path keeps it. Prefer the real-SQLite harness if the scanner tests already have one; otherwise a recording fake that asserts the *method* used and the *fields* sent.
- [x] **Task 3 — Site 3 `aggregateSeriesFileSizes`** (AC #2) — `scanner_service.go:697-745`
  - [x] 3.1 Add `UpdateFileSize` to the scanner's series-repo interface; replace the wide write.
  - [x] 3.2 Characterization test as in 2.2, series flavour (`parse_status` / `poster_path` changed underneath survives).
- [x] **Task 4 — Sites 1, 4, 6 → single-intent writers** (AC #3) — `scanner_service.go:480-495`, `library_service.go:785-806`, `metadata_edit_service.go:305-330`; their interfaces and tests
  - [x] 4.1 Site 1: `UpdateScanFileInfo(ctx, existing.ID, info.Size(), "pending")` — keep the "only when size changed" condition exactly as today.
  - [x] 4.2 Site 4: `UpdateParseStatus` for both branches; the handler response is unchanged.
  - [x] 4.3 Site 6: `UpdatePosterPath` for both branches.
  - [x] 4.4 Update the affected service tests' fakes (they will fail to compile — good; that is the list of touched seams).
- [x] **Task 5 — A sites documented + sweep** (AC #5) — one comment each at `metadata_edit_service.go:153/:242`, `movie_service.go:114`, `series_service.go:138`: "wide Update kept on purpose — user edit owns these fields; loaded µs before; no long window (bugfix-wide-update-stale-copy-other-callers §audit #5/#7/#8)". Run the AC #5 grep and paste the output into Completion Notes.
- [x] **Task 6 — Gates** — `pnpm nx test api` zero FAIL, `pnpm nx test web` (unchanged but the full-regression gate requires it), lint api, format:check; Dev Agent Record with the audit table's final state.

### Cross-stack split check — Backend 6 / Frontend 0 ⇒ no split.

## Dev Notes

- **Do not** convert the A sites (5, 7, 8). A user edit that must persist what the user typed across many fields is exactly what the wide writer is for; narrowing it would be a product change (which fields are editable) dressed as a refactor.
- **Do not** touch `Upsert` (site 9) — tracked separately; it has no production callers.
- `enriched_metadata_update.go` is the template: header audit record, nil/empty-id guards, `updated_at` stamped from `time.Now().UTC()` (check which clock the repo uses and match it).
- The scanner's post-scan steps run **after** `onScanComplete` has fired (`scanner_service.go:282-314`: confirm the order — if `detectRemovedFiles` runs before the callback, say so in Completion Notes; the hazard then comes from a *previous* scan's enrichment still running, which is still real on a large library).
- Rule 11: add methods to the consumer-side interfaces in the services, not to one shared interface. Rule 13: a narrow write that errors is logged and the loop continues (site 2/3 are best-effort passes), exactly as the wide write's error is handled today — keep the same branch.
- Rule 20: no stamped ACs involved; the scanner SSE contract is untouched.

### Time-dependent visual coverage — N/A (backend only).

### Discovery Triage (pre-filled at creation)

- ③ `bugfix-upsert-zeroes-unloaded-columns` — filed in `sprint-status.yaml` at creation (site 9): `Upsert` writes zero-values for columns it never loaded; dormant (no prod callers). Bidirectional.

### References

- [Source: `apps/api/internal/repository/enriched_metadata_update.go` — header audit, narrow-writer shape; `_test.go:65` characterization test]
- [Source: `apps/api/internal/repository/movie_repository.go:168`, `series_repository.go:173` — wide Update column lists]
- [Source: `apps/api/internal/services/scanner_service.go:472-495, 634-665, 697-745`]
- [Source: `apps/api/internal/services/library_service.go:785-806`; `metadata_edit_service.go:86-153, 175-242, 305-330`; `movie_service.go:114`; `series_service.go:138`]
- [Source: `_bmad-output/implementation-artifacts/9R-10b-on-add-autotrigger.md` — CR-249 finding B]

## Dev Agent Record

### Agent Model Used

Claude Fable 5 (claude-fable-5) — 2026-08-24.

### Debug Log References

- RED: `scan_state_update_test.go` failed to compile on the seven new writers; series tests then failed on `no such table: series` (wrong DB fixture — `setupTestDB` is movies-only; switched to `setupSeriesTestDB`).
- Wide `Update` on 0 rows: returns `fmt.Errorf("movie with id %s not found")` WITHOUT wrapping `sql.ErrNoRows` (`movie_repository.go:276`). Narrow writers wrap it (AC #4, recorded in the file header).
- gofmt: `scanner_service.go` / `library_service.go` were already non-gofmt-clean on `main` (pre-existing); not reformatted to keep the diff reviewable.

### Completion Notes List

- `apps/api/internal/repository/scan_state_update.go`: `execNarrow` helper + movie `UpdateScanFileInfo` / `MarkRemoved` / `UpdateParseStatus` / `UpdatePosterPath`, series `UpdateFileSize` / `UpdateParseStatus` / `UpdatePosterPath`. Header carries the audit record.
- Sites converted: #1 `processVideoFile` → `UpdateScanFileInfo`; #2 `detectRemovedFiles` → `MarkRemoved`; #3 `aggregateSeriesFileSizes` → `UpdateFileSize`; #4 `BatchReparse` (both branches) → `UpdateParseStatus`; #6 `updatePosterPath` (both branches) → `UpdatePosterPath` (FindByID kept for the not-found error shape). A sites #5/#7/#8 annotated.
- Interfaces: methods added to `repository.MovieRepositoryInterface` / `SeriesRepositoryInterface` (the `UpdateEnrichedMetadata` precedent — the scanner and library services consume that interface) and to the metadata-edit service's own `MovieMetadataRepository` / `SeriesMetadataRepository`. Mocks updated: `testutil.MockMovieRepository`/`MockSeriesRepository`, `mockMovieRepoForNFO`, `mockPQMovieRepo`/`mockPQSeriesRepo`, `mockMovieMetadataRepository`/`mockSeriesMetadataRepository`.
- Tests (corrected at review, L2): repository — 7 column-isolation (4 movie + 3 series), 2 missing-row, 4 characterization replaying the scanner sequences (wide path loses the concurrent subtitle/enrichment write, narrow path keeps it; the movie pair now runs the REAL order: load → subtitle lands → write, review L3); services — 3 scanner tests rewritten to pin `UpdateScanFileInfo` / `MarkRemoved` AND `AssertNotCalled("Update")`, poster-upload test pins `UpdatePosterPath` + no wide `Update`, new `TestLibraryService_BatchReparse_UsesNarrowParseStatusWriter` (review L4). 14 new + 4 rewritten.
- **AC #5 grep output** (`grep -n "movieRepo.Update(\|seriesRepo.Update(\|\.repo.Update(ctx, movie)\|\.repo.Update(ctx, series)" apps/api/internal/services/*.go | grep -v _test`):
  `metadata_edit_service.go:161`, `metadata_edit_service.go:251`, `movie_service.go:117`, `series_service.go:140` — exactly the A sites.
- Post-scan order (Dev Notes check): `detectRemovedFiles` runs inside `StartScan` BEFORE `onScanComplete` fires (`scanner_service.go:282` vs `:314`), so within one scan the overlap is with the PREVIOUS scan's enrichment / free-lane round still running — still real on a large library, and the characterization tests do not depend on the order.
- Site 3 has no service-level test (no episode-repo mock exists in `testutil`; building one is out of proportion) — pinned by the compile-time interface change + the repository characterization + AC #5 grep.
- `UpdatePosterPath` binds `models.NewNullString(path)` so `""` clears to NULL exactly as the wide path did (review nit).
- Code review (fresh-context agent, same model): 1 M / 3 L / 1 nit — all addressed; M1 was the story contradicting itself on Rule 11, resolved by recording the precedent as the accepted exception above.
- Gates: `pnpm nx test api` zero FAIL · `pnpm nx test web` 237 files green · lint api · format:check · 0 orphaned workers.
- 🔗 AC Drift: NONE (checked `detectRemovedFiles|aggregateSeriesFileSizes|BatchReparse|poster` across stories — Story 9c-3 AC #8 "series file_size aggregated from episode files" still holds; the write is narrower, the observable value identical). 📎 Contract Stamps: NONE. 🎭 A11y: N/A. 🎨 UX: SKIPPED.

### Discovery Triage

- ③ `bugfix-upsert-zeroes-unloaded-columns` — filed at creation (site 9). No new discoveries during dev.

### File List

- `apps/api/internal/repository/scan_state_update.go` (new)
- `apps/api/internal/repository/scan_state_update_test.go` (new)
- `apps/api/internal/repository/interfaces.go`
- `apps/api/internal/services/scanner_service.go`
- `apps/api/internal/services/scanner_service_test.go`
- `apps/api/internal/services/library_service.go`
- `apps/api/internal/services/metadata_edit_service.go`
- `apps/api/internal/services/metadata_edit_service_test.go`
- `apps/api/internal/services/movie_service.go`
- `apps/api/internal/services/series_service.go`
- `apps/api/internal/services/enrichment_nfo_test.go`
- `apps/api/internal/services/parse_queue_service_test.go`
- `apps/api/internal/testutil/mocks.go`
- `_bmad-output/implementation-artifacts/bugfix-wide-update-stale-copy-other-callers.md`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`

## Change Log

| Date | Change |
|------|--------|
| 2026-08-24 | code-review: 1M/3L/1nit addressed (interface decision recorded, counts corrected, faithful movie characterization, service pins for sites 4/6, NULL poster). Status → done. |
| 2026-08-24 | dev-story (Amelia): 7 narrow writers, 5 sites converted, 3 A sites annotated, 14 new + 4 rewritten tests; gates green. Status → review. |
| 2026-08-24 | Story created (create-story). Audit of 9 sites: 2×B (narrow + characterization test), 3×C→narrow (single-intent writers), 3×A (kept, documented), 1 dormant (filed separately). |
