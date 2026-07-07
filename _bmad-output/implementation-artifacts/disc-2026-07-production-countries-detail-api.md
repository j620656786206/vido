# Story disc-2026-07-production-countries-detail-api: Persist + expose `production_countries` on the local movie detail API, and light up the §9b CN-policy display (full-stack)

Status: ready-for-dev

<!-- Rule-24 ③ discovery from ux3-subtitle-v2 (DEV Amelia, 2026-07-05). Scope ratified full-stack via Party Mode 2026-07-08. -->

> **Party Mode ruling (2026-07-08, ratified by Alexyu):** FULL-STACK. The discovery labeled this "expose the field", but investigation found `production_countries` is **never persisted** — `converters.go` sets it on the model from TMDb, but `Create`/`Update`/`Upsert` all drop it, and `movieSelectColumns`/`scanMovie` never read it, so the column is effectively always NULL. The honest scope is the full chain: **persist → read → expose → FE-consume**. Wire shape = **array** `[{iso_3166_1, name}]` (matches `models.ProductionCountry`, the `Genres` array-on-wire precedent, and the TMDb path `MediaDetailPanel.tsx`). The FE §9b CN info line ALREADY exists in `ManageSubtitleDialogV2` (dormant `productionCountry` prop) — wiring it is the trivial, un-design-blocked last mile, so full-stack is the smallest COMPLETE increment (BE-only would ship a field nobody consumes → YAGNI). **Series is out of scope** — it has no `production_countries` column or model field at all.

## Story

As a Vido user viewing a 大陸片 (mainland movie) whose subtitle is 簡中,
I want the 管理字幕 dialog to actually show the §9b CN-policy info line explaining why 簡中 is kept (not auto-converted),
so that a correct policy behaviour reads as intentional, not as a bug — which requires the backend to finally persist and expose the film's production countries.

## Acceptance Criteria

1. **Persist `production_countries` (Rule 15 write-sync).** `MovieRepository.Create` (INSERT) and `MovieRepository.Update` (UPDATE) MUST persist `movie.ProductionCountriesJSON` to the `movies.production_countries` column (column + `?` placeholder + arg, matching order). `Upsert` needs no change (it delegates to Create/Update). Verified: today all three silently drop it (`converters.go:87` sets the model field from TMDb but it never reaches the DB). [@contract-v1]

2. **Read `production_countries` (Rule 15 read-sync).** `movieSelectColumns` (`movie_repository.go:617`) MUST include `production_countries`, and `scanMovie` (`~630`) MUST scan it into `&movie.ProductionCountriesJSON` **at the matching column position**. This covers the detail path (`FindByID` → `MovieService.GetByID` → `GET /api/v1/movies/:id`) and the `FindBy*` family. The separate LIST query (`~362`, its own inline column list + scan) is deliberately NOT touched — the grid does not consume production countries (keep the surface minimal).

3. **Expose as an array (Rule 6/18).** Add a computed field to `models.Movie`: `ProductionCountries []ProductionCountry `db:"-" json:"production_countries,omitempty"``, populated in `scanMovie` after the row scan via the existing `GetProductionCountries()` accessor (mirrors the `Genres []string` array-on-wire pattern — one JSON-text column → typed slice → real JSON array). `db:"-"` so it is NOT a scan target. Wire element shape = `{"iso_3166_1": "...", "name": "..."}` (the existing `models.ProductionCountry`, movie.go:92). [@contract-v1]

4. **FE consumes it — light up the §9b info line.** `apps/web/src/types/library.ts` `LibraryMovie` gains `productionCountries?: ProductionCountry[]` (FE `ProductionCountry` = `{ iso31661, name }` from `types/tmdb.ts:91`; wire `production_countries[].iso_3166_1` → `productionCountries[].iso31661` via the `snakeToCamel` boundary in `libraryService.ts:38`, Rule 18). `LocalDetailV2.tsx` computes the comma-joined ISO string for movies only and passes it to `ManageSubtitleDialogV2`'s existing `productionCountry` prop — EXACTLY mirroring `MediaDetailPanel.tsx:99` (`details.productionCountries?.map((c) => c.iso31661).join(',') ?? ''`) + `:294`. Result: `isCNContent` (`ManageSubtitleDialogV2.tsx:158`) becomes true for CN films → the §9b policy copy renders live (today `productionCountry` is `undefined` → always `false`).

5. **Series unchanged (capability honor).** Series has NO `production_countries` column or model field (migration 006 is movies-only; `subtitle/batch.go:464` documents the ConvertAuto default). `LocalDetailV2` passes empty/undefined `productionCountry` for series → the info line stays hidden, consistent with today. Do NOT add a series column/migration (out of scope).

6. **Existing rows need a re-scan (no backfill migration).** Because the column was never written, existing movies have NULL `production_countries` and will show no info line until re-scanned/re-enriched (which, after AC #1, now persists it). This is expected — document it in Completion Notes; do NOT write a data-migration/backfill.

7. **Tests + gates.** (a) **MANDATORY real-DB integration test (Rule 15 / bugfix-20-1 precedent):** in `movie_repository_test.go`, create a movie with `SetProductionCountries([{ISO3166_1:"CN",Name:"China"}, ...])` → persist via `Create`/`Upsert` → `FindByID` → assert `movie.ProductionCountries` round-trips the array. This MUST hit a real (in-memory sqlite) DB, NOT a mocked repo. (b) Handler/serialization assertion that `GET /movies/:id` emits `production_countries` as a JSON array. (c) FE spec: `LocalDetailV2` passes the joined string; `ManageSubtitleDialogV2` renders the §9b info line when a CN code is present. (d) Full regression: `nx test api` + `nx test web` green; `pnpm lint:all` 0 errors; UX screenshot check of the dialog (Sally gate). A11y pre-flight for the touched `apps/web/src/components/**` (info line is static text — low risk, but record the pre-flight result).

## Tasks / Subtasks

- [ ] **Task 1: Persist `production_countries` (BE, Rule 15 write-sync)** (AC: 1)
  - [ ] `movie_repository.go` `Create` (~L44): add `production_countries` to the INSERT column list, a `?` placeholder, and `movie.ProductionCountriesJSON` to the exec args (matching order).
  - [ ] `movie_repository.go` `Update` (~L193): add `production_countries = ?` and `movie.ProductionCountriesJSON` to the exec args (matching order).
  - [ ] Confirm `Upsert` (L1057) needs no change (delegates to Create/Update).
- [ ] **Task 2: Read + expose (BE, Rule 15 read-sync + Rule 6/18)** (AC: 2, 3)
  - [ ] `movieSelectColumns` (L617): add `production_countries` (choose a stable position, e.g. after `hdr_format`).
  - [ ] `scanMovie` (~L630): add `&movie.ProductionCountriesJSON` to the scan targets **at the exact matching position**; after the scan-error check, populate `movie.ProductionCountries` via `GetProductionCountries()` (ignore parse error → empty slice, mirror the accessor's own nil-safety).
  - [ ] `models/movie.go`: add `ProductionCountries []ProductionCountry `db:"-" json:"production_countries,omitempty"`` (near `ProductionCountriesJSON` / `Genres`).
- [ ] **Task 3: Real-DB round-trip test (BE, Rule 15 mandate)** (AC: 7a, 7b)
  - [ ] `movie_repository_test.go`: real sqlite integration test — persist a movie with `SetProductionCountries` → `FindByID` → assert `ProductionCountries` array (incl. a `CN` entry). NOT a mocked repo.
  - [ ] Handler/serialization assertion that `GET /movies/:id` JSON carries `production_countries` as an array (mirror an existing `movie_handler_test.go` detail test).
- [ ] **Task 4: FE consume — §9b info line (FE, Rule 18)** (AC: 4, 5)
  - [ ] `types/library.ts`: add `productionCountries?: ProductionCountry[]` to `LibraryMovie` (import/define FE `ProductionCountry` `{ iso31661, name }` — reuse the `types/tmdb.ts:91` type).
  - [ ] `LocalDetailV2.tsx`: compute `productionCountryStr` for movies (`data.productionCountries?.map((c) => c.iso31661).join(',') ?? ''`, empty for series) and pass `productionCountry={productionCountryStr}` to `ManageSubtitleDialogV2` (mirror `MediaDetailPanel.tsx:99/294`).
  - [ ] Spec: `LocalDetailV2` passes the joined string; `ManageSubtitleDialogV2` shows the §9b info line when a CN code is present.
- [ ] **Task 5: Verification** (AC: 7c, 7d)
  - [ ] `nx test api` + `nx test web` green; `pnpm lint:all` 0 errors; A11y pre-flight on touched `apps/web/src/components/**` (record result).
  - [ ] UX screenshot check of the 管理字幕 dialog CN state vs `flow-f-subtitle-v2/` (Sally gate; the info line is an existing design element).

**Cross-stack split check:** backend tasks = 3 (persist / read+expose / test), frontend tasks = 1 (Task 4) → single story, no a/b split.

## Dev Notes

### The core problem — a broken persistence chain (code-verified 2026-07-08)

`production_countries` is captured but NEVER persisted or read:

- **Captured:** `services/converters.go:87` `movie.SetProductionCountries(countries)` (from `tmdb.MovieDetails.ProductionCountries`).
- **Dropped on write:** `MovieRepository.Create` (movie_repository.go:27, INSERT L44-53) and `Update` (L163, UPDATE L~193) column lists **omit `production_countries`** (they also omit `credits`, `spoken_languages` — see Discovery Triage ③). `Upsert` (L1057) delegates to those.
- **Not read:** `movieSelectColumns` (L617) omits it; `scanMovie` (~L630) never scans it; `MovieService.GetByID` (services/movie_service.go:54) does no enrichment.
- **Not serialized:** `models.Movie.ProductionCountriesJSON` (movie.go:134) is `json:"-"` (raw JSON blob).

So flipping the `json:"-"` tag alone would emit an empty JSON-encoded STRING — wrong on three counts. All four links must be fixed.

### Reuse map (do NOT reinvent)

- **`Genres` is the array-on-wire precedent:** `Genres []string `db:"genres" json:"genres"`` (movie.go:110) — one JSON-text column scanned via a custom path into a typed slice, serialized as a real JSON array. `production_countries` follows the same shape (array of objects instead of strings).
- **`GetProductionCountries()` accessor** (movie.go:247) already parses `ProductionCountriesJSON` → `[]ProductionCountry`; `ProductionCountry{ISO3166_1 string json:"iso_3166_1"; Name string json:"name"}` (movie.go:92). Use it to populate the computed field — do NOT write new JSON parsing.
- **`SuccessResponse(c, movie)`** (movie_handler.go:106) serializes `*models.Movie` **directly, no DTO** — that's why the exposure must be a real JSON-tagged field on the model.
- **FE precedent — the TMDb path already does exactly this join:** `MediaDetailPanel.tsx:99` `const productionCountryStr = details.productionCountries?.map((c) => c.iso31661).join(',') ?? '';` → `:294` `productionCountry={productionCountryStr}`. Copy this verbatim into `LocalDetailV2`. FE `ProductionCountry` type: `types/tmdb.ts:91` (`{ iso31661, name }`).
- **Consumer (already built, just dormant):** `ManageSubtitleDialogV2.tsx` `productionCountry?: string` (L129, JSDoc L128 "ISO 3166-1 codes; contains 'CN' → §9b policy display (no local-detail source today)"), `isCNContent = productionCountry?.includes('CN') ?? false` (L158). This story provides the missing source. The v1 `SubtitleSearchDialog.tsx:31` has the same prop (consistency reference).
- **Detail fetch path:** `LocalDetailV2.tsx` data at L60 (`isMovie ? localMovie.data : localSeries.data`) from `useLocalMovieDetails` (`hooks/useMediaDetails.ts:138`) → `libraryService.getMovieById` → `GET /movies/${id}` (`libraryService.ts:136`); `snakeToCamel` at `libraryService.ts:38` (Rule 18 boundary — `production_countries` → `productionCountries`, `iso_3166_1` → `iso31661`).

### Architecture compliance

- **Rule 15 (the whole point):** write-sync (Create/Update) AND read-sync (movieSelectColumns/scanMovie) MUST move together; a real-DB integration test is MANDATORY (the mocked-repo unit test is the exact trap that hid bugfix-20-1 — a column added but never SELECT/scanned silently returns the zero value). Scan order MUST match the SELECT column order.
- Rule 6: snake_case JSON (`production_countries`, `iso_3166_1`). Rule 18: `snakeToCamel` at the FE boundary. Rule 5: TanStack Query (detail hooks already use it — the new field rides the existing query). Rule 4: Handler → Service → Repository (unchanged layering).
- Rule 20: `[@contract-v1]` on the movie-detail `production_countries` array field (AC #1/#3) — the wire shape the FE consumes; a future FE/BE consumer acks it.
- Rule 7: N/A — no error codes. Rule 23: N/A — no wall-clock reads.
- Strangler: `MediaDetailPanel` (legacy TMDb dialog) already had a live `productionCountry`; this story only adds the LOCAL/v2 path — do not touch the legacy panel's logic.

### Series has no data source (do not invent one)

`models.Series` has no `production_countries` field; no migration adds it to the `series` table (006/018/019/020/021/024 checked). Adding it would need a new column + TMDb-fetch mapping + enrichment — a separate, larger story. Out of scope: series `productionCountry` stays empty → info line hidden → ConvertAuto (today's behaviour).

### Project Structure Notes

- BE touch: `internal/repository/movie_repository.go` (+ `_test.go`), `internal/models/movie.go`. No migration (column exists since 006). No `main.go` change.
- FE touch: `apps/web/src/types/library.ts`, `apps/web/src/components/media/LocalDetailV2.tsx` (+ `.spec.tsx`).

### Time-dependent visual coverage

N/A — no `apps/web/src/components/**` file added/modified reads `Date.now()`/`new Date()`. `LocalDetailV2` change is a pure prop pass-through; `ManageSubtitleDialogV2` is unchanged.

### References

- [Source: apps/api/internal/repository/movie_repository.go — Create L27/L44, Update L163/L193, Upsert L1057, movieSelectColumns L617, scanMovie L~630, FindByID L95]
- [Source: apps/api/internal/models/movie.go — ProductionCountry L92, ProductionCountriesJSON L134 (json:"-"), GetProductionCountries L247, Genres L110 (array-on-wire precedent)]
- [Source: apps/api/internal/services/converters.go:87 — SetProductionCountries (the capture that is currently dropped)]
- [Source: apps/api/internal/handlers/movie_handler.go:106 — SuccessResponse(c, movie), direct model serialization, no DTO]
- [Source: apps/web/src/components/media/MediaDetailPanel.tsx:99,294 — the FE join precedent to copy]
- [Source: apps/web/src/components/subtitle/ManageSubtitleDialogV2.tsx:129,158 — dormant productionCountry prop + isCNContent]
- [Source: apps/web/src/components/media/LocalDetailV2.tsx:60,279-292 — data source + dialog render site]
- [Source: apps/web/src/types/library.ts (LibraryMovie), types/tmdb.ts:91 (ProductionCountry), services/libraryService.ts:38,136 (snakeToCamel boundary + GET /movies/:id)]
- [Source: project-context.md Rule 15 (DB Column Sync + real-DB test), §9b CN policy, Rules 5/6/18/20]
- [Source: sprint-status.yaml disc-2026-07-production-countries-detail-api (this), disc-2026-07-track-convert-endpoint (sibling, shares the FE dialog surface)]
- [Party Mode 2026-07-08: Winston/John/Murat/Sally/Amelia + Alexyu — full-stack scope, array shape, real-DB test, credits/spoken_languages ③ triage]

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

### Discovery Triage

Story-authoring-time discoveries (SM Bob, 2026-07-08 — Party Mode):

- **③ backlog-with-carry-forward-link — `disc-2026-07-credits-spoken-languages-persist`** (BE): `credits` and `spoken_languages` have the EXACT same never-persisted gap as `production_countries` — `converters.go` sets them (`SetCredits`/`SetSpokenLanguages`), but `Create`/`Update`/`Upsert` and `movieSelectColumns`/`scanMovie` all omit them, and they are `json:"-"`. Out of scope here (this story is `production_countries` only), but a real latent gap surfaced by the Party Mode investigation → file it so it is tracked, not silently ignored. File the `sprint-status.yaml` entry at story-authoring time (bidirectional link).
- **Cross-story relationship note (→ `disc-2026-07-track-convert-endpoint`, already done):** the §9b CN INFO LINE (this story's FE consumer, un-design-blocked) and the `轉為繁中` convert BUTTON (the convert endpoint's FE consumer, design-blocked pending the `.pen` affordance) live in the SAME `ManageSubtitleDialogV2` surface but are INDEPENDENTLY shippable. This story wires the info line now; the button stays deferred. A one-line note is added to the convert story's Discovery Triage recording this (no code/behaviour change to the merged convert endpoint — its server-side independence from `production_countries` was correct).
- (Dev: add any further in-flight discoveries here per Rule 24 before marking done.)

### File List
