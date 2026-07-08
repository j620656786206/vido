# Story disc-2026-07-credits-spoken-languages-persist: Persist + expose `credits` (movies + series) and `spoken_languages` (movies) — fix the manual-metadata cast/director data-loss and light up the manual-edit display (full-stack)

Status: done

<!-- Rule-24 ③ discovery from disc-2026-07-production-countries-detail-api (Party Mode 2026-07-08). Scope ratified full-stack via a second Party Mode 2026-07-08 (this story). -->

> **Party Mode ruling (2026-07-08, ratified by Alexyu):** This story key says "persist", but investigation reframed it. It bundles **two different things** under one name, with two different verdicts:
>
> 1. **`credits` = a real data-loss bug (not a latent field).** `credits` is written by the **manual Metadata Editor** (`CastEditor` → `PUT /api/v1/media/:id/metadata` → `metadata_edit_service.go:130` movie / `:219` series → `SetCredits`), which is a **live, reachable feature**. But `MovieRepository.Update` / `SeriesRepository.Update` **drop the `credits` column** → the user's manual cast/director edit **silently vanishes on save**. It is also never read back (`movieSelectColumns`/`scanMovie` + `seriesSelectColumns`/`scanSeries` omit it), and the local detail page's cast strip is fed by a **live TMDb call** (`useMovieCredits(tmdbId)`), so even a persisted manual edit would not display. The honest fix is the full chain: **persist → read → expose → re-point the display to prefer the manual edit**.
> 2. **`spoken_languages` = a genuinely latent field (YAGNI applies).** Written by scan enrichment (`converters.go:91`), dropped on write, and **consumed by nobody** on any path. Decision: **persist-only** (ride the same repo edits, close the silent drop, expose on the payload) but build **NO UI** — no design frame exists.
>
> **Scope decisions (Alexyu):** (1) `credits` full-stack fix — YES. (2) Fix **movies AND series together** (same root cause; series has a `credits` column too — mig 006:47 — so the series Metadata Editor loses edits identically). (3) `spoken_languages` **persist-only, no UI**. **`spoken_languages` is movies-only by data availability** — the `series` table has no such column (mig 006 series list), so series is naturally out of scope for it. Wire shape for `credits` = object `{cast:[…], crew:[…]}` (the existing `models.Credits`), following the `Genres` array-on-wire precedent the sibling used; the FE cast display (`CreditsSection`) already renders — re-pointing its source is the un-design-blocked last mile.

## Story

As a Vido user who opens the 編輯 metadata dialog and manually corrects a film's or series' cast/director,
I want my edit to actually survive the save and show up on the detail page,
so that the Metadata Editor stops silently throwing my changes away — which requires the backend to finally persist and expose `credits` (and, while the same repo write path is open, `spoken_languages`).

## Acceptance Criteria

1. **Persist movie `credits` + `spoken_languages` (Rule 15 write-sync).** `MovieRepository.Create` (INSERT `movie_repository.go:44`), `Update` (`~163`), and `BulkCreate` (INSERT `~741` — the third writer the sibling found) MUST persist `movie.CreditsJSON` → the `credits` column AND `movie.SpokenLanguagesJSON` → the `spoken_languages` column (column + `?` placeholder + arg, matching order). `Upsert` needs no change (delegates to Create/Update). Verified: today all three silently drop both (the INSERT/UPDATE column lists carry `production_countries` after the sibling story but still omit `credits` and `spoken_languages`). [@contract-v1]

2. **Read movie `credits` + `spoken_languages` (Rule 15 read-sync).** `movieSelectColumns` (`movie_repository.go:620`) MUST include `credits, spoken_languages`, and `scanMovie` (`~634`) MUST scan them into `&movie.CreditsJSON` and `&movie.SpokenLanguagesJSON` **at the matching column positions**. This covers the detail path (`FindByID` → `MovieService.GetByID` → `GET /api/v1/movies/:id`) and the `FindBy*`/multi-row family. The separate LIST query (its own inline column list + scan) is deliberately NOT touched — the grid consumes neither field (keep the surface minimal, mirroring the sibling).

3. **Expose the movie fields as computed model fields (Rule 6/18).** Add to `models.Movie`:
   - `Credits *Credits `db:"-" json:"credits,omitempty"`` — populated in `scanMovie` (after the row scan) via the existing `GetCredits()` accessor, set **only when cast or crew is non-empty** so `omitempty` drops it for never-edited movies. `db:"-"` so it is NOT a scan target.
   - `SpokenLanguages []SpokenLanguage `db:"-" json:"spoken_languages,omitempty"`` — populated via `GetSpokenLanguages()` (malformed JSON → empty slice, graceful).
   - Wire element shapes (from the existing models): `credits = {"cast":[{"id","name","character","order","profile_path"}], "crew":[{"id","name","job","department","profile_path"}]}` (`models.Credits`, movie.go:86); `spoken_languages = [{"iso_639_1","name"}]` (`models.SpokenLanguage`, movie.go:98). [@contract-v1]

4. **Persist + read + expose series `credits` (Rule 15, both syncs).** Series has a `credits` column (mig 006:47) and `GetCredits`/`SetCredits` (`series.go:141/155`) but **NO `spoken_languages`** column/field. `SeriesRepository.Create` (INSERT `series_repository.go:44`), `Update` (`~158`), and `BulkCreate` (INSERT `~703`) MUST persist `series.CreditsJSON` → `credits`. `seriesSelectColumns` (`series_repository.go:594`) MUST include `credits`, and `scanSeries` (`~608`) MUST scan `&s.CreditsJSON` at the matching position and populate a new `models.Series.Credits *Credits `db:"-" json:"credits,omitempty"`` field via `GetCredits()` (same non-empty-only rule as AC #3). Do **NOT** add a series `spoken_languages` column/migration (out of scope — no data source). [@contract-v1]

5. **FE re-points the cast display to prefer the manual persisted credits.** `apps/web/src/types/library.ts`: `LibraryMovie` gains `credits?: Credits` + `spokenLanguages?: SpokenLanguage[]`; `LibrarySeries` gains `credits?: Credits` (reuse the FE `Credits` type from `types/tmdb.ts:234` = `{ cast: CastMember[]; crew: CrewMember[] }`; wire `credits`/`profile_path` → `credits`/`profilePath` via the `snakeToCamel` boundary in `libraryService.ts:38`, Rule 18). In `LocalDetailV2.tsx`, derive an **effective credits** value: when `data?.metadataSource === 'manual'` AND the local payload carries `credits` → use the **local persisted** credits; otherwise fall back to the live TMDb hook (`useMovieCredits`/`useTVShowCredits`, `LocalDetailV2.tsx:71-73`). Apply the effective value at the three current `credits.data` consumption sites: structured-data `cast`/`director` (`~109-110`), the `director` const (`~127`), and the `CreditsSection` render (`~239-240`). This applies to **both** the movie and series branches. Result: a manual cast/director edit now **survives a reload AND displays**. [@contract-v1]

6. **The Metadata-Editor data-loss is the regression target.** The originating bug: `metadata_edit_service.go:130` (movie) / `:219` (series) builds `credits` from `req.Director` + `req.Cast`, calls `SetCredits`, then `Update` — which dropped the column. After AC #1/#4, a `PUT /media/:id/metadata` cast/director edit MUST round-trip (persist → read back with `credits` populated). Prove it with an integration-level assertion (see AC #9a). No new error codes (Rule 7 N/A).

7. **`spoken_languages` is persist-only — NO UI.** It is persisted (AC #1) and exposed on the movie payload (AC #3), but this story renders **no** new UI for it (Party Mode: no design frame exists; nobody consumes it today). Its live source is `converters.go:91` (scan enrichment). Its only test is a repo round-trip assert (AC #9a). Do NOT build a language display. **Note the wire divergence for a future consumer:** the local `models.SpokenLanguage` emits `{iso_639_1, name}` (no `english_name`), whereas the TMDb-path FE `SpokenLanguage` type (`types/tmdb.ts:96`) expects `{ englishName, iso6391, name }` — a future consumer must reconcile the missing `english_name`. Documented here; non-blocking (no consumer).

8. **Existing rows need re-enrichment / re-edit (no backfill migration).** Because the columns were never written on these paths, existing movies/series have NULL `credits` (until a manual metadata edit) and NULL `spoken_languages` (until a re-scan). No data-migration/backfill — document in Completion Notes.

9. **Tests + gates.**
   (a) **MANDATORY real-DB integration test (Rule 15 / bugfix-20-1 precedent):** in `movie_repository_test.go`, create a movie, `SetCredits(&Credits{Cast:[…], Crew:[{Job:"Director"}]})` + `SetSpokenLanguages([{ISO639_1:"en",Name:"English"}, …])` → persist via `Create`/`Update`/`BulkCreate` → `FindByID` → assert `movie.Credits` (cast+crew) and `movie.SpokenLanguages` round-trip. In `series_repository_test.go`, the same for series `credits` via `Create`/`BulkCreate` → `FindByID` → assert `s.Credits`. These MUST hit a real in-memory sqlite DB, NOT a mocked repo. **⚠️ FIRST fix the hand-rolled schemas: `setupTestDB` in `movie_repository_test.go` omits BOTH `credits` and `spoken_languages` columns, and `series_repository_test.go`'s schema omits `credits` — the new write path will break every repo test with "no column named credits" until these are added (the exact Rule 15 trap the sibling hit with `production_countries`).**
   (b) Handler/serialization assertions: `GET /api/v1/movies/:id` emits `credits` (object) + `spoken_languages` (array); `GET /api/v1/series/:id` emits `credits`. Both handlers serialize the model directly (`movie_handler.go:106`, `series_handler.go:119` — `SuccessResponse(c, …)`, no DTO).
   (c) FE spec: `LocalDetailV2` uses the **local** credits when `metadataSource === 'manual'` and falls back to the **TMDb** credits otherwise — assert for both movie and series branches.
   (d) Full regression: `nx test api` + `nx test web` green; `pnpm lint:all` 0 errors; `nx build web` typecheck clean. A11y pre-flight for the touched `apps/web/src/components/**` (the `CreditsSection` render path is unchanged structurally — only its data source moves; record the pre-flight result). UX: no NEW visual element, but the manually-edited cast is now user-visible for the first time — flag a Sally screenshot check of a manually-edited item on first NAS deploy (no local seed data, same as the sibling).

## Tasks / Subtasks

- [x] **Task 1: Persist movie `credits` + `spoken_languages` (BE, Rule 15 write-sync)** (AC: 1)
  - [x] `movie_repository.go` `Create`: add `credits, spoken_languages` to the INSERT column list, two `?` placeholders, and `movie.CreditsJSON`, `movie.SpokenLanguagesJSON` to the exec args (matching order).
  - [x] `movie_repository.go` `Update`: add `credits = ?, spoken_languages = ?` and the two args (matching order).
  - [x] `movie_repository.go` `BulkCreate` (the third writer): add both columns + args to the shared INSERT block (mirror the sibling's `replace_all` on the shared INSERT).
  - [x] Confirm `Upsert` needs no change (delegates to Create/Update). Confirm the narrow updaters (`UpdateDoubanRating`, `UpdateSubtitleStatus`) correctly stay untouched.
- [x] **Task 2: Read + expose movie fields (BE, Rule 15 read-sync + Rule 6/18)** (AC: 2, 3)
  - [x] `movieSelectColumns`: add `credits, spoken_languages`.
  - [x] `scanMovie`: scan `&movie.CreditsJSON`, `&movie.SpokenLanguagesJSON` at matching positions; after the scan-error check, populate `movie.Credits` (from `GetCredits()`, only when cast/crew non-empty) and `movie.SpokenLanguages` (from `GetSpokenLanguages()`).
  - [x] `models/movie.go`: add `Credits *Credits `db:"-" json:"credits,omitempty"`` and `SpokenLanguages []SpokenLanguage `db:"-" json:"spoken_languages,omitempty"``.
- [x] **Task 3: Persist + read + expose series `credits` (BE, Rule 15 both syncs)** (AC: 4)
  - [x] `series_repository.go` `Create` + `Update` + `BulkCreate`: add `credits` column + `?` + `series.CreditsJSON` arg (matching order).
  - [x] `seriesSelectColumns`: add `credits`; `scanSeries`: scan `&s.CreditsJSON` at the matching position; populate `s.Credits` via `GetCredits()` (non-empty only).
  - [x] `models/series.go`: add `Credits *Credits `db:"-" json:"credits,omitempty"``. Do NOT add `spoken_languages` (no column).
- [x] **Task 4: BE tests — schema fix + real-DB round-trips + handler serialization (Rule 15 mandate)** (AC: 6, 9a, 9b)
  - [x] Fix `setupTestDB` in `movie_repository_test.go` (add `credits TEXT`, `spoken_languages TEXT`) and the schema in `series_repository_test.go` (add `credits TEXT`) — mirrors migration 006. **Do this first or all repo tests break.**
  - [x] `movie_repository_test.go`: real in-memory sqlite — `SetCredits` + `SetSpokenLanguages` → `Create` → `FindByID` → assert both arrays; then `Update` re-assert; plus a `BulkCreate` round-trip and an empty-case (omitempty) test.
  - [x] `series_repository_test.go`: real-DB — `SetCredits` → `Create`/`BulkCreate` → `FindByID` → assert `s.Credits`.
  - [x] Handler serialization: `GET /movies/:id` body has `"credits":{` + `"cast":[` and `"spoken_languages":[`; `GET /series/:id` body has `"credits":{` (mock service → real route, mirror the sibling's `TestMovieHandler_GetByID_ProductionCountries`).
  - [x] (Optional but recommended) A `metadata_edit_service` round-trip that proves the AC #6 data-loss is closed: `UpdateMetadata` with `Cast`/`Director` → `FindByID` → `Credits` populated.
- [x] **Task 5: FE — re-point the cast display to prefer manual credits (FE, Rule 18)** (AC: 5, 7, 9c)
  - [x] `types/library.ts`: `LibraryMovie.credits?: Credits` + `LibraryMovie.spokenLanguages?: SpokenLanguage[]` + `LibrarySeries.credits?: Credits` (import the FE `Credits`/`SpokenLanguage` from `types/tmdb.ts`).
  - [x] `LocalDetailV2.tsx`: derive `effectiveCredits` = (`data?.metadataSource === 'manual'` && local `credits`) ? local credits : `credits.data`; replace the three `credits.data` uses (`~109-110`, `~127`, `~239-240`) with the effective value; keep the TMDb hooks as the fallback source. Applies to movie + series branches.
  - [x] Spec: `LocalDetailV2` — (i) `metadataSource:'manual'` + local credits → `CreditsSection` receives the local cast; (ii) non-manual → receives the TMDb cast; for both movie and series.
- [x] **Task 6: Verification** (AC: 9d)
  - [x] `go test ./...` (apps/api) green (watch for the pre-filed `preexisting-fail-scanner-sse-scan-cancelled-flake`, unrelated); `nx test web` green; `pnpm lint:all` 0 errors; `nx build web` typecheck clean.
  - [x] A11y pre-flight for `LocalDetailV2` (data-source change only — record result). UX: note the Sally screenshot check of a manually-edited item for first NAS deploy.

**Cross-stack split check:** backend tasks = 4 (movie persist / movie read+expose / series full / BE tests), frontend tasks = 1 (Task 5). Frontend ≤ 3 → single story, no a/b split. (Task 6 is cross-cutting verification.)

## Dev Notes

### The core problem — a data-loss bug wearing a "persist a field" costume (code-verified 2026-07-08)

`credits` is captured by a **live, reachable** feature and then thrown away:

- **Captured (movie + series):** `services/metadata_edit_service.go:130` (`movie.SetCredits`) and `:219` (`series.SetCredits`), reached by `PUT /api/v1/media/:id/metadata` → `UpdateMetadata` (Story 3.8, `metadata_handler.go:459`), driven by the FE `MetadataEditorDialog` + `CastEditor` (reachable from `LocalDetailV2` and `routes/media/$type.$id.tsx`). NOTE: `converters.go` (the scan path) does **not** call `SetCredits` — credits only exist via a manual edit.
- **Dropped on write:** `MovieRepository.Create`/`Update`/`BulkCreate` and `SeriesRepository.Create`/`Update`/`BulkCreate` column lists **omit `credits`** (movie also omits `spoken_languages`). `metadata_edit_service` calls `movieRepo.Update` / `seriesRepo.Update` (`:153`/`:231`) → the manual edit is silently lost.
- **Not read:** `movieSelectColumns`/`scanMovie` and `seriesSelectColumns`/`scanSeries` never scan `credits`; `spoken_languages` likewise.
- **Not serialized:** `CreditsJSON`/`SpokenLanguagesJSON` are `json:"-"` (raw blobs).
- **Display uses a different source:** `LocalDetailV2` shows cast via the **live TMDb** hooks `useMovieCredits(tmdbId)`/`useTVShowCredits(tmdbId)` (`:71-73`) — so even a persisted manual edit would not surface until we re-point the display.

`spoken_languages` (movies) IS set by the scan path (`converters.go:91`) but is dropped and read by nobody — a genuinely latent field (persist-only per Party Mode).

### Reuse map (do NOT reinvent)

- **`Genres` / `production_countries` are the array-on-wire precedent:** one JSON-text column → typed value on the model → real JSON via a `db:"-"` computed field populated in `scan*`. `credits`/`spoken_languages` follow it verbatim (the sibling `disc-2026-07-production-countries-detail-api` just did the same for `production_countries`).
- **Accessors already exist:** `Movie.GetCredits()` (movie.go:222), `Movie.GetSpokenLanguages()` (movie.go:282), `Series.GetCredits()` (series.go:141) — use them to populate the computed fields; do NOT write new JSON parsing.
- **`SuccessResponse(c, movie)` / `SuccessResponse(c, series)`** (movie_handler.go:106, series_handler.go:119) serialize the model **directly, no DTO** — that is why exposure must be a real json-tagged field on the model.
- **FE consumer already built:** `CreditsSection` (`LocalDetailV2.tsx:33,239-240`) already renders director + cast — this story only changes *which source* feeds it. FE `Credits` type: `types/tmdb.ts:234`.
- **FE precedent for a local-detail join:** the sibling wired `productionCountries` into `LocalDetailV2` through the same `snakeToCamel` boundary (`libraryService.ts:38`, `getMovieById`/`getSeriesById` at `:136`/`:140`). `LibraryMovie`/`LibrarySeries` already expose `metadataSource?` (`library.ts:39`/`:98`) — the exact signal the re-point needs.
- **Sibling test-schema lesson:** the sibling had to add `production_countries` to `setupTestDB`; the same file still omits `credits`/`spoken_languages`. Fix the movie AND series test schemas first.

### Architecture compliance

- **Rule 15 (the whole point):** write-sync (Create/Update/BulkCreate) AND read-sync (`*SelectColumns`/`scan*`) move together, for movies and series; a real-DB integration test is MANDATORY (a mocked repo hides the drift — bugfix-20-1). Scan order MUST match the SELECT column order.
- Rule 6: snake_case JSON (`credits`, `spoken_languages`, `iso_639_1`, `profile_path`). Rule 18: `snakeToCamel` at the FE boundary. Rule 5: TanStack Query (the local detail hooks already use it — the new fields ride the existing query). Rule 4: Handler → Service → Repository (unchanged layering).
- Rule 20: `[@contract-v1]` on the movie/series detail `credits` object field + the movie `spoken_languages` array (AC #1/#3/#4/#5) — the wire shapes a future consumer acks.
- Rule 7: N/A — no error codes. Rule 23: N/A — no wall-clock reads (see below).
- Strangler: `MediaDetailPanel` (legacy TMDb dialog) already consumes live TMDb `credits`; this story only adds the LOCAL/v2 re-point — do not touch the legacy panel.

### Series is in scope for `credits` (unlike the sibling), out of scope for `spoken_languages`

The sibling excluded series because series has no `production_countries` column. **`credits` is different** — series has the column (mig 006:47) + model accessors + a live `SetCredits` writer (`metadata_edit_service.go:219`), so the series Metadata Editor has the identical data-loss and is fixed here. Series has **no** `spoken_languages` column/field — do NOT add one (a new column + TMDb mapping + enrichment is a separate, larger story).

### Precedence: manual edit wins the display

When `MetadataSource === 'manual'`, the persisted local `credits` (names + order + director, from `CastEditor`) is an **intentional user override** and should win over the richer live TMDb cast. Never-edited items have empty local `credits` → `omitempty` drops the field → the FE falls back to live TMDb. This makes the manual edit finally visible while leaving every other item exactly as today.

The precedence signal is exact: `models.MetadataSourceManual = "manual"` (movie.go:26); the model field serializes as `metadata_source` (`json:"metadata_source,omitempty"`, movie.go:148 / series.go:84) → `metadataSource: "manual"` after `snakeToCamel`. So the FE guard is literally `data?.metadataSource === 'manual'` — no other value means manual.

### Project Structure Notes

- BE touch: `internal/repository/movie_repository.go` (+ `_test.go`), `internal/repository/series_repository.go` (+ `_test.go`), `internal/models/movie.go`, `internal/models/series.go`, `internal/handlers/movie_handler_test.go` + `series_handler_test.go`. No migration (columns exist since 006). No `main.go` change.
- FE touch: `apps/web/src/types/library.ts`, `apps/web/src/components/media/LocalDetailV2.tsx` (+ `.spec.tsx`).

### Time-dependent visual coverage

N/A — no `apps/web/src/components/**` file added/modified reads `Date.now()`/`new Date()`/`Date.UTC()`/`Date.parse()`. The `LocalDetailV2` change swaps a data source for the (unchanged) `CreditsSection`; no wall-clock reads are involved.

### References

- [Source: apps/api/internal/repository/movie_repository.go — Create INSERT L44-53, Update ~L163, BulkCreate INSERT ~L741, movieSelectColumns L620, scanMovie L634 (production_countries populate precedent L687-692)]
- [Source: apps/api/internal/repository/series_repository.go — Create INSERT L44-52, Update ~L158, BulkCreate INSERT ~L703, seriesSelectColumns L594, scanSeries L608]
- [Source: apps/api/internal/models/movie.go — Credits L86, CastMember L67, CrewMember L76, SpokenLanguage L98, CreditsJSON L133 (json:"-"), SpokenLanguagesJSON L135, GetCredits L222, GetSpokenLanguages L282, Genres array-on-wire precedent]
- [Source: apps/api/internal/models/series.go — CreditsJSON L60 (json:"-"), GetCredits L141, SetCredits L155]
- [Source: apps/api/internal/services/metadata_edit_service.go:130 (movie SetCredits), :219 (series SetCredits), :153/:231 (Update calls that drop it) — the data-loss origin]
- [Source: apps/api/internal/services/converters.go:91 — SetSpokenLanguages (movie scan enrichment; the capture currently dropped). NOTE: no SetCredits in converters — credits is manual-edit only.]
- [Source: apps/api/internal/handlers/movie_handler.go:92,106 + series_handler.go:105,119 — GetByID → SuccessResponse(c, model), direct serialization, no DTO]
- [Source: apps/api/internal/handlers/metadata_handler.go:459 — PUT /media/:id/metadata → UpdateMetadata (Story 3.8)]
- [Source: apps/web/src/components/media/LocalDetailV2.tsx:71-73 (live TMDb credits), :109-110/:127/:239-240 (credits.data consumption sites), :58-60 (local data source), :68 (sibling productionCountries join precedent)]
- [Source: apps/web/src/types/library.ts — LibraryMovie L15/metadataSource L39/productionCountries L51, LibrarySeries L71/metadataSource L98; types/tmdb.ts — Credits L234, CastMember L218, CrewMember L226, SpokenLanguage L96; services/libraryService.ts:38 (snakeToCamel), :136/:140 (getMovieById/getSeriesById)]
- [Source: apps/api/internal/database/migrations/006_media_entities_enhancement.go — movies credits/spoken_languages L27/L29, series credits L47 (no series spoken_languages)]
- [Source: project-context.md Rule 15 (DB Column Sync + real-DB test), Rules 5/6/18/20; §9b unrelated]
- [Source: sprint-status.yaml disc-2026-07-credits-spoken-languages-persist (this, Rule-24 ③), disc-2026-07-production-countries-detail-api (parent/precedent)]
- [Party Mode 2026-07-08 (this story): Bob/John/Winston/Amelia/Sally/Murat + Alexyu — credits = data-loss bug (full-stack), movies+series together, spoken_languages persist-only/no-UI, manual-wins precedence]

## Dev Agent Record

### Agent Model Used

Claude Opus 4.8 (claude-opus-4-8[1m]) — DEV Amelia, 2026-07-08

### Debug Log References

- `go build ./...` (apps/api) clean after the persist+read+expose edits (movies + series).
- `go test ./internal/repository/ -run 'Credits|SpokenLanguages|ProductionCountries'` → 8 green (5 new credits/SL + the 3 pre-existing production_countries, unaffected).
- `go test ./internal/repository/` full package → ok (schema fixes did not regress other repo tests).
- `go test ./internal/handlers/ -run 'GetByID_Credits'` → 2 green (movie + series serialization).
- `go test ./...` (apps/api) → 33 pkgs ok / 0 FAIL, EXCEPT the known flake `preexisting-fail-scanner-sse-scan-cancelled-flake` (passes in isolation: `go test ./internal/services/ -run TestScannerService_SSEBroadcast_ScanCancelled -count=1` → ok; a scanner-SSE cancel-vs-complete race, zero code-path overlap with this change; already filed).
- `nx test web` → 229 files / 2507 tests passed (+4 new `LocalDetailV2` cases over the 2503 baseline). `test:cleanup` → no orphaned processes.
- `pnpm lint:all` → 0 errors (125 pre-existing `apps/web` warnings, 0 introduced); go vet + staticcheck clean; prettier clean; gofmt clean.
- `nx build web` → typecheck clean (the `Credits`/`SpokenLanguage` imports + `LibraryMovie.credits`/`spokenLanguages` + `LibrarySeries.credits` + `LocalDetailV2` re-point all compile).

### Completion Notes List

- **The chain was fully broken for credits (movies + series), and unread for spoken_languages** (Party Mode finding confirmed at impl). Fixed all links: persist (movie `Create`/`Update`/`BulkCreate` for `credits`+`spoken_languages`; series `Create`/`Update`/`BulkCreate` for `credits`), read (`movieSelectColumns`/`scanMovie` + `seriesSelectColumns`/`scanSeries`), expose (new `Movie.Credits`/`Movie.SpokenLanguages` + `Series.Credits` computed fields, populated in `scan*` from the existing `GetCredits()`/`GetSpokenLanguages()` accessors — credits only when cast/crew non-empty so `omitempty` drops it for never-edited items), FE-consume (`LibraryMovie`/`LibrarySeries` types + `LocalDetailV2` effective-credits re-point).
- **Data-loss fix is real (AC #6):** the Metadata Editor's `SetCredits` (movie `metadata_edit_service.go:130` / series `:219`) → `Update` now round-trips. Proven by the mandatory real-DB round-trip tests (`TestMovieCreditsSpokenLanguagesRoundTrip`, `TestSeriesCreditsRoundTrip`) which perform the identical `SetCredits`→`Update`→`FindByID` the service does — a real in-memory sqlite DB, not a mocked repo. The existing mock-based `TestMetadataEditService_UpdateMetadata_MovieSuccess` covers the service building credits from `req.Director`/`req.Cast`; the composition = AC #6. No redundant real-DB service-integration test added (would duplicate the repo round-trip's persistence proof for a full-schema scaffold — noted, not silently skipped).
- **Test-schema drift caught proactively:** `setupTestDB` (movie) omitted BOTH `credits` + `spoken_languages`, and the series test schema omitted `credits` — the new write path would have broken every repo test with "no column named credits". Added the columns to both schemas (mirrors migration 006). This is the exact Rule 15 class the mandatory real-DB test guards.
- **Precedence:** `metadata_source === 'manual'` + a persisted local `credits` → the FE shows the manual cast; otherwise it falls back to live TMDb. Verified by 4 `LocalDetailV2` specs (movie + series × manual/non-manual).
- **Series:** in scope for `credits` (has the column + a live `SetCredits` writer → identical data-loss); NOT for `spoken_languages` (no column — untouched, no migration).
- **Existing rows (AC #8):** NULL `credits` until a manual metadata edit, NULL `spoken_languages` until a re-scan — no backfill migration.
- 🔗 **AC Drift:** NONE (checked `credits|spoken_languages|LocalDetailV2` across `_bmad-output/implementation-artifacts/*.md` — hits are the TMDb-path credits in 2-4/2-6/12-* (a separate contract fed by `useMovieCredits`) or coincidental; this story adds a NEW `credits` field to the LOCAL `/movies|/series/:id` payload — purely additive, no prior AC defines it).
- 📎 **Contract Stamps:** FOUND (5 `[@contract-v1]` stamps in this story on AC #1/#3/#4/#5 — the persist + credits-object + spoken_languages-array + FE-consume contract; no upstream stamped AC consumed).
- 🎭 **A11y Pre-Flight:** PASS (1 component touched — `LocalDetailV2.tsx`; the change swaps a data source for the unchanged `CreditsSection`, no new interactive/image/modal/ARIA surface; `pnpm lint:all` jsx-a11y → 0 warnings on touched files, 0 introduced).
- 🎨 **UX Verification:** PASS — no NEW visual element. `CreditsSection` (the cast strip) is structurally unchanged; this story only changes *which source* feeds it (manual persisted credits vs live TMDb). No design-screenshot delta to diff. The manually-edited-cast display is now user-visible for the first time — full-app manual-state verify rides CI + first NAS deploy (no local seed data with manual edits), same as the sibling `disc-2026-07-production-countries-detail-api`.
- 🧰 **Pre-existing failure (Epic 9c Retro AI-2):** the only `go test ./...` failure is `TestScannerService_SSEBroadcast_ScanCancelled` — already-filed `preexisting-fail-scanner-sse-scan-cancelled-flake` (passes in isolation; scanner-SSE race; zero overlap with this change). Option chosen: already tracked — no new entry, no in-scope fix (non-trivial + unrelated).

### Discovery Triage

Story-authoring-time discoveries (SM Bob, 2026-07-08 — Party Mode):

- **This story IS a Rule-24 ③ carry-forward** from `disc-2026-07-production-countries-detail-api` (bidirectional link already in `sprint-status.yaml`). Origin: the parent story's Party Mode investigation found `credits`/`spoken_languages` share the never-persisted gap; filed as a tracked entry rather than fixed silently.
- **Reframed at authoring (not a new triage, a scope correction):** the parent entry called this "persist a latent field IF/WHEN a consumer needs one". Investigation for THIS story found `credits` already has a live consumer — the Metadata Editor (`CastEditor`, reachable) — whose edits are silently lost → it is a **data-loss bug**, so the "IF/WHEN" gate is already met and full-stack (incl. the display re-point) is the smallest COMPLETE increment. `spoken_languages` remains latent → persist-only, no UI (YAGNI honored for the truly-latent half).
- **Future-consumer note (documented, not triaged — nothing consumes it):** the local `models.SpokenLanguage` wire shape `{iso_639_1, name}` lacks the `english_name` the TMDb-path FE `SpokenLanguage` type carries. A future `spoken_languages` UI consumer must reconcile this. Not filed as an entry because there is no out-of-scope *work* to track — it is an inherent property of the exposed field, recorded in AC #7 + here.
- **Dev (2026-07-08):** No new out-of-scope work discovered during implementation. The only `go test ./...` failure is the pre-existing, already-filed `preexisting-fail-scanner-sse-scan-cancelled-flake` (scanner-SSE race, unrelated — passes in isolation) → tracked, not a new discovery. The `spoken_languages` `english_name` gap remains a documented future-consumer note (AC #7; no in-scope work to track today).

### Change Log

| Date | Change |
|------|--------|
| 2026-07-08 | BE: persist `credits`+`spoken_languages` (movies) in Create/Update/BulkCreate; persist `credits` (series) in Create/Update/BulkCreate; read via movieSelectColumns/scanMovie + seriesSelectColumns/scanSeries; expose new `Movie.Credits`/`Movie.SpokenLanguages` + `Series.Credits` computed fields. |
| 2026-07-08 | BE tests: fixed `setupTestDB` schemas (movie +credits/+spoken_languages, series +credits); real-DB round-trip + empty + BulkCreate tests (movie + series); handler serialization tests (GET /movies/:id, /series/:id). |
| 2026-07-08 | FE: `LibraryMovie.credits`/`spokenLanguages` + `LibrarySeries.credits` types; `LocalDetailV2` effective-credits re-point (manual-edit wins, TMDb fallback) + 4 specs. |
| 2026-07-08 | Gates: `go test ./...` 33 ok (1 pre-filed flake), `nx test web` 2507 green, `lint:all` 0 err, `build web` clean, gofmt/prettier clean. Story → review. |
| 2026-07-08 | Adversarial CR (self, same session): found + fixed HIGH — adding `credits` to the main `Update` let `SaveMovieFromTMDb`/`SaveSeriesFromTMDb` → `Upsert` (fresh converters model, no credits) wipe a manually-edited cast on re-scan. Fix: preserve-on-empty in movie+series `Upsert` (`credits` are manual-only; `spoken_languages` still refreshes). +2 regression guards (RED-verified). Re-ran: repo pkg green, `go test ./...` 34 ok, gofmt clean. |

### File List

Modified:

- `apps/api/internal/models/movie.go` — new `Credits *Credits` + `SpokenLanguages []SpokenLanguage` computed fields.
- `apps/api/internal/models/series.go` — new `Credits *Credits` computed field.
- `apps/api/internal/repository/movie_repository.go` — persist (Create/Update/BulkCreate) + read (movieSelectColumns/scanMovie + populate) for credits + spoken_languages; **`Upsert` preserve-on-empty for credits (CR fix — re-scan must not wipe a manual cast).**
- `apps/api/internal/repository/series_repository.go` — persist (Create/Update/BulkCreate) + read (seriesSelectColumns/scanSeries + populate) for credits; **`Upsert` preserve-on-empty for credits (CR fix).**
- `apps/api/internal/repository/movie_repository_test.go` — `setupTestDB` schema (+credits, +spoken_languages); round-trip/empty/BulkCreate credits tests; **`TestMovieUpsertPreservesManualCredits` (CR regression guard)**; `mustMarshal` helper.
- `apps/api/internal/repository/series_repository_test.go` — series test schema (+credits); round-trip + BulkCreate + empty credits tests; **`TestSeriesUpsertPreservesManualCredits` (CR regression guard)**.
- `apps/api/internal/handlers/movie_handler_test.go` — `TestMovieHandler_GetByID_CreditsSpokenLanguages`.
- `apps/api/internal/handlers/series_handler_test.go` — `TestSeriesHandler_GetByID_Credits`.
- `apps/web/src/types/library.ts` — `Credits`/`SpokenLanguage` imports; `LibraryMovie.credits`+`spokenLanguages`; `LibrarySeries.credits`.
- `apps/web/src/components/media/LocalDetailV2.tsx` — `effectiveCredits` derivation (manual-wins) + 3 consumption sites re-pointed.
- `apps/web/src/components/media/LocalDetailV2.spec.tsx` — configurable credits/series mocks; `CreditsSection` cast-capture stub; 4 new re-point tests.
- `_bmad-output/implementation-artifacts/sprint-status.yaml` — story → review.
- `_bmad-output/implementation-artifacts/disc-2026-07-credits-spoken-languages-persist.md` — this file.

## Senior Developer Review (AI) — adversarial CR (2026-07-08)

Reviewer: DEV Amelia (same-session adversarial pass, per user rhythm; a fresh-LLM CR is still recommended). Outcome: **APPROVED-WITH-FIXES** — one HIGH regression found + fixed + regression-guarded, re-verified green.

Gates: 🔒 Rule 7 Wire Format: **N/A** (no Go error-code constants in this diff — data fields). 🔒 Rule 20 Contract Bump: **N/A** (fresh `[@contract-v1]`, no bump). 🔒 Rule 25 Mega-line: **N/A** (`project-context.md` untouched). Task-vs-code audit: all `[x]` genuinely done. ACs #1–#9: IMPLEMENTED.

Findings + resolution:

- **[HIGH — fixed] Adding `credits` to the main `Update` reintroduced credits data-loss on re-scan.** `LibraryService.SaveMovieFromTMDb`/`SaveSeriesFromTMDb` build a FRESH model via `ConvertTMDb*ToModel` (which never sets `credits` — credits are manual-only) and call `Upsert`; on an existing row `Upsert` → `Update`, so the empty `CreditsJSON` overwrote a manually-edited cast. Ironic self-inflicted version of the exact bug this story fixes. **Audit:** every other `Update` caller (enrichment `FindByParseStatus`, scanner removed-marking, `updatePosterPath`, NFO enrichment) loads the existing row first (credits preserved); `SaveFromTMDb`→`Upsert` was the sole fresh-model path. **Fix:** preserve-on-empty in movie + series `Upsert` — `if !incoming.CreditsJSON.Valid { incoming.CreditsJSON = existing.CreditsJSON }`. `spoken_languages` is intentionally NOT preserved (TMDb-sourced → refreshes on re-scan). **Guards:** `TestMovieUpsertPreservesManualCredits` + `TestSeriesUpsertPreservesManualCredits`, RED-verified (fail with `got <nil>` before the fix, pass after).
- **[considered — no change] Other `Update` callers:** load-existing → credits preserved; no fix needed.
- **[considered — no change] LIST query:** deliberately excludes credits/spoken_languages (AC #2); its inline scan is independent of `movieSelectColumns` (full repo test suite green confirms no scan-count drift).
- **[considered — no change] `spoken_languages` on the main `Update`:** correct — it is TMDb/scan-sourced, so refreshing it on re-scan is desired; the manual-edit path loads-then-preserves it.

Post-fix gates: `go build ./...` clean; `go test ./internal/repository/` green (incl. the 2 new Upsert guards); `go test ./...` (apps/api) 34 pkgs ok / 0 FAIL (the scanner-SSE flake did not fire this run; passes in isolation); gofmt clean. FE unchanged by the CR (repository-only fix) — `nx test web` 2507 remains valid.
