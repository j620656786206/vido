# Bugfix: `Upsert` writes zero-values for columns it never loaded — make the re-match preserve them

Status: done

**Origin:** Rule 24 ③ from `bugfix-wide-update-stale-copy-other-callers` audit site 9 (2026-08-24). `MovieRepository.Upsert` (`movie_repository.go:1063-1096`) / `SeriesRepository.Upsert` (`series_repository.go:965-993`) take a **freshly converted** TMDb model, copy only `ID` + `CreatedAt` (+ `CreditsJSON` when the incoming one is empty) from the existing row, then call the wide `Update` — so every column the TMDb converter never produces is written back as its zero value. Not a stale-copy race — a **"overwrite what you never loaded"** defect. Currently **dormant**: the only callers are `LibraryService.SaveMovieFromTMDb` / `SaveSeriesFromTMDb`, which no handler wires (tests only — `library_service_test.go` ×20, `library_handler_test.go`, `route_c_uuid_integration_test.go`). Backend only.

---

## Story

As a future developer who wires TMDb re-matching into a real flow,
I want `Upsert` to preserve every column the TMDb payload does not carry,
so that re-matching a movie cannot silently erase its subtitle delivery state, library membership, removed flag, or file information.

## Facts (verified 2026-08-24)

- Wide `Update` writes 37 movie / 32 series columns (see `bugfix-wide-update-stale-copy-other-callers` §audit). The converter (`ConvertTMDbMovieToModel` / `ConvertTMDbSeriesToModel`, `tmdb/converters.go`) produces TMDb metadata + `FilePath` (from the caller's argument) and never sets: **subtitle_status / subtitle_path / subtitle_language / subtitle_last_searched / subtitle_search_score** (movie), **library_id**, **is_removed**, **file_size** — those become `''`/NULL/0/false on every re-match of an existing row.
- The `CreditsJSON` preservation already inside `Upsert` (both repos) is the precedent: the same treatment is owed to the other never-produced columns; the existing comment even explains the principle ("Without this, Upsert would overwrite the persisted manual cast with NULL").
- `FindByTMDbID`'s not-found detection matches on the **error string** (`movie_repository.go:1077`, `series_repository.go:975`) — brittle, but out of scope; do not change it here (note it in Completion Notes if it bites).
- The narrow writers from `bugfix-wide-update-stale-copy-other-callers` don't apply: Upsert legitimately writes the whole TMDb-owned surface; its defect is only the never-owned remainder.

## Design

| # | Decision | Chosen | Why |
|---|----------|--------|-----|
| D1 | Fix or delete? | **Fix** (preserve-on-rematch) | The three test files use `SaveMovieFromTMDb` as a convenient seeding helper; deleting the path is churn without removing the class — and the entry exists precisely so the helper is safe to wire later. |
| D2 | Preservation mechanism | **Explicit field copies from `existing`, next to the `CreditsJSON` precedent** — movie: subtitle×5, `LibraryID`, `IsRemoved`, `FileSize`, and `FilePath` only when the incoming one is empty/invalid; series: `LibraryID`, `IsRemoved`, `FileSize`, `ParseStatus`?—NO: check the converter first (Task 1.1) and preserve exactly the set it never assigns | A load-then-patch on all 37 columns would be a second wide-copy mechanism; explicit copies keep the "who owns what" audit readable in one place. |
| D3 | Contract statement | A doc comment on both `Upsert`s: *"TMDb owns the metadata surface; everything else on the row belongs to other writers and is preserved on re-match. If the converter grows a field, add it here or it will be preserved-stale."* | The failure mode of D2 is a future converter field silently preserved; the comment names the trade. |

## Acceptance Criteria

1. **AC #1 — Movie re-match preserves delivery/ownership state.** Given an existing movie (matched by TMDb id) with `subtitle_status=found` + `subtitle_path`, a `library_id`, `is_removed=true`, a `file_size`, and a `file_path`, when `Upsert` runs with a fresh TMDb model (empty those fields; empty `FilePath`), then all of them survive and the TMDb metadata (title, overview, vote_average…) is refreshed.
2. **AC #2 — Incoming `FilePath` wins when provided.** When the fresh model carries a non-empty `FilePath`, it replaces the stored one (re-match after a file move is a legitimate path update).
3. **AC #3 — Series re-match** preserves its never-produced set (per Task 1.1's converter audit — at minimum `library_id`, `is_removed`, `file_size`; series wide Update carries no subtitle columns).
4. **AC #4 — Create path untouched.** A TMDb id with no existing row still `Create`s exactly as today (both the no-TMDb-id and not-found branches).
5. **AC #5 — Existing behaviour pinned, not weakened.** `CreditsJSON` preservation and the `spoken_languages` refresh-on-rematch decision (documented in the existing comment) still hold; all existing `library_service_test.go` / handler / integration tests stay green unmodified (or with strictly added assertions).

## Tasks / Subtasks

- [x] **Task 1 — Converter audit + preservation** — `apps/api/internal/repository/movie_repository.go`, `series_repository.go`
  - [x] 1.1 Enumerate every field `ConvertTMDbMovieToModel` / `ConvertTMDbSeriesToModel` assigns (read `tmdb/converters.go`); the preservation set = wide-Update columns minus that set minus `ID`/`CreatedAt`/`UpdatedAt`. Record both lists in the Upsert doc comment (D3).
  - [x] 1.2 Implement the copies in both `Upsert`s beside the `CreditsJSON` block; `FilePath` incoming-wins-when-non-empty (AC #2).
- [x] **Task 2 — Tests** — `apps/api/internal/repository/scan_state_update_test.go` sibling file or a new `upsert_preserve_test.go` (real SQLite, same harness)
  - [x] 2.1 AC #1: seed via `Create` + `UpdateSubtitleGenerationStatus` + set library/removed/size, then `Upsert` a fresh TMDb model → assert preserved set + refreshed metadata.
  - [x] 2.2 AC #2, #3, #4 cases; a characterization comment (not a test of the old behaviour — the old behaviour was never shipped to a caller).
- [x] **Task 3 — Gates** — `pnpm nx test api` zero FAIL · `nx test web` · lint api · format:check; Dev Agent Record; AC #5 confirmed by the untouched existing tests.

### Cross-stack split check — Backend 3 / Frontend 0 ⇒ no split.

## Dev Notes

- **Do not** wire `SaveMovieFromTMDb` anywhere — the story makes it safe, not live.
- **Do not** touch the string-matched not-found detection or convert `Upsert` to the narrow writers (D2's why).
- The dormant status means no characterization test of the broken behaviour is owed (nothing shipped depends on it) — fix + pin the correct behaviour only.
- Rule 20: nothing stamped touches this path. Rule 24: bidirectional links already in place (sprint-status ↔ wide-update story §audit #9).

### Time-dependent visual coverage — N/A (backend only).

### References

- [Source: `apps/api/internal/repository/movie_repository.go:1063-1096`, `series_repository.go:965-993` — the two Upserts + CreditsJSON precedent]
- [Source: `apps/api/internal/services/library_service.go:151-200` — the only callers; `tmdb/converters.go` — the produced-field set]
- [Source: `_bmad-output/implementation-artifacts/bugfix-wide-update-stale-copy-other-callers.md` §audit #9]

## Dev Agent Record

### Agent Model Used

Claude Fable 5 (claude-fable-5) — 2026-08-24.

### Debug Log References

- Fault injection: commenting out one preservation line (`movie.LibraryID = existing.LibraryID`) turns `TestMovieUpsert_RematchPreservesColumnsTMDbDoesNotOwn` red — the tests bite.
- Repo-level create test needed explicit IDs (`Create` does not uuid; `SaveMovieFromTMDb` does at the service layer) — noted in the test comment.
- A long-lived `playwright test` process (PID 12883, ~3 min) surfaced in `test:cleanup` — parented by ANOTHER Claude session on this machine (its shell shows `git checkout feat/pg-13517-visual-gate`, the work-repo naming), not by this session's vitest runs; left alone. Zero vitest workers.

### Completion Notes List

- **Task 1.1 converter audit** (`services/converters.go` — note: the story guessed `tmdb/converters.go`; actual path corrected): movie converter produces title, release_date, parse_status=success, tmdb_id, original_title, overview, poster/backdrop, vote_average/count, popularity, rating, runtime, original_language, status, imdb_id, genres, production_countries, spoken_languages, file_path (when arg non-empty), metadata_source. **Never produces**: subtitle×5, library_id, is_removed, file_size, video_codec, video_resolution, audio_codec, audio_channels, subtitle_tracks, hdr_format — the ffprobe tech columns were NOT in the original entry's list; the audit added them. Series never-produces the same minus subtitle×5 (its wide Update has no subtitle columns).
- **Task 1.2**: explicit copies in both `Upsert`s beside the `CreditsJSON` precedent; `FilePath` incoming-wins-when-non-empty; movie `Upsert` carries the full ownership-contract doc comment (D3), series refers to it.
- **Tests (6)**: movie re-match preserves all non-TMDb columns + refreshes metadata; incoming file path wins; both create branches unchanged; CreditsJSON rule still holds (crew-based — `models.Credits` has no Director field, story example corrected); series re-match flavour. Real SQLite.
- Gates: `pnpm nx test api` zero FAIL · `pnpm nx test web` 237 files green · lint api · format:check · 0 vitest workers.
- **Code review (fresh-context agent, same model): 1 H / 1 M / 2 L — all addressed.**
  - **F1 (HIGH, real catch)**: series `imdb_id` is wide-Update-written but the TV converter can never produce it (TMDb's TV payload has no imdb field — it lives behind the external-ids endpoint), while the series create handler sets it and `FindByIMDbID` depends on it ⇒ a re-match NULLed it. Preserved now + test-pinned; both ownership comments corrected (the movie contract's "imdb ids are TMDb-owned" is movie-only).
  - **F2 (MED)**: fixtures now seed and assert EVERY preserved column (movie seeds via `UpdateSubtitleStatus` so `last_searched`/`search_score` are non-zero; ffprobe×6 seeded both sides). Fault injection re-run on 4 sampled lines incl. `series.IMDbID` — each single deletion goes red.
  - **F4 (LOW)**: series `popularity` is converter-produced but neither series `Create` nor `Update` persists it — out-of-scope discovery, filed as `bugfix-series-popularity-never-persisted` (Rule 24 ③, bidirectional).
  - **F5 (LOW)**: doc comment now states the corollary — a path can never be CLEARED through `Upsert` (removal is the scanner's `is_removed`).
- 🔗 AC Drift: N/A (dormant path; no shipped AC observes Upsert). 📎 Contract Stamps: NONE. 🎭 A11y: N/A (backend). 🎨 UX: SKIPPED.

### Discovery Triage

- **Did this story discover any work outside its current scope?**
  - ③ `bugfix-series-popularity-never-persisted` — filed at CR F4 (converter→DB drop, not a zeroing bug; series Create/Update never write `popularity`). Bidirectional.

### File List

- `apps/api/internal/repository/movie_repository.go`
- `apps/api/internal/repository/series_repository.go`
- `apps/api/internal/repository/upsert_preserve_test.go` (new)
- `_bmad-output/implementation-artifacts/bugfix-upsert-zeroes-unloaded-columns.md`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`

## Change Log

| Date | Change |
|------|--------|
| 2026-08-24 | code-review: F1 series imdb_id preserved (the story's own defect class, one column missed), F2 full-fixture pinning + 4-line fault injection, F4 filed, F5 doc. Status → done. |
| 2026-08-24 | dev-story (Amelia): converter audit widened the preserve set (+6 ffprobe tech columns); explicit copies + ownership contract; 6 tests, fault-injection verified. Status → review. |
| 2026-08-24 | Story created (create-story). D1 fix-not-delete; preservation = explicit copies beside the CreditsJSON precedent, driven by a converter-field audit. |
