# Story 11.3: Instant Search with zh-TW Priority

Status: done

## Story

As a Traditional Chinese NAS user,
I want instant search with debounced suggestions that prioritizes Traditional Chinese results,
so that I find content quickly using Chinese or original titles.

## Acceptance Criteria

1. Given the search bar, when the user types ≥2 characters, then debounced suggestions appear within 300ms showing movies, TV shows, and people as separate categories
2. Given a search query in Chinese, when results are returned, then items with zh-TW title matches are ranked above original-title-only matches
3. Given a search query, when TMDB returns results, then the system searches both zh-TW and original titles simultaneously and merges/deduplicates
4. Given the search dropdown, when a user clicks a suggestion, then they navigate to the media detail page
5. Given the search bar on mobile, when activated, then it expands to full-width with a dedicated search view
6. Given an existing Epic 2 search implementation, when enhanced, then backward compatibility with existing search routes is maintained

## Tasks / Subtasks

- [x] Task 1: Backend — dual-language search + ranking (AC: #2, #3)
  - [x] 1.1 Extend `SearchMovies` and `SearchTVShows` in TMDB service to query both `zh-TW` and `en` languages
  - [x] 1.2 Merge results by TMDB ID, prefer zh-TW metadata when available
  - [x] 1.3 Boost score for items where zh-TW title matches the query (move to top)
  - [x] 1.4 New endpoint: `GET /api/v1/search?q={query}&page=1` — unified search across movies + TV + people

- [x] Task 2: Backend — people search (AC: #1)
  - [x] 2.1 Add `SearchPeople(ctx, query, page)` to TMDB client → `GET /search/person`
  - [x] 2.2 Include in unified search response as separate category

- [x] Task 3: Frontend — search suggestions dropdown (AC: #1, #4)
  - [x] 3.1 Create `apps/web/src/components/search/SearchSuggestions.tsx`
  - [x] 3.2 Debounce input (300ms) using `useDebouncedValue` or `useDebounce`
  - [x] 3.3 Three sections in dropdown: 電影, 影集, 人物
  - [x] 3.4 Each item: poster thumbnail + title + year
  - [x] 3.5 Click navigates to `/media/movie.{id}` or `/media/tv.{id}` (impl: actual route is `/media/$type/$id` → `/media/movie/{id}`)
  - [x] 3.6 Keyboard navigation: arrow keys + Enter

- [x] Task 4: Frontend — enhanced search bar (AC: #5)
  - [x] 4.1 Replace or enhance existing search bar in toolbar (AppShell now uses `InstantSearchBar`)
  - [x] 4.2 Desktop: inline suggestions dropdown below search bar
  - [x] 4.3 Mobile: full-screen search overlay (opened via toolbar search toggle)
  - [x] 4.4 Clear button, Escape to close

- [x] Task 5: Tests (AC: #1-6)
  - [x] 5.1 Backend: dual-language merge, zh-TW boost ranking
  - [x] 5.2 Frontend: debounce timing, keyboard navigation, click navigation
  - [x] 5.3 Backward compatibility: existing `/search` route + per-type endpoints still work

## Dev Notes

### Architecture Compliance

- **Debounce:** 300ms client-side debounce. Do NOT debounce on server
- **Existing search:** Epic 2 already has `SearchMovies`/`SearchTVShows` with language fallback. Extend, don't replace
- **Unified endpoint:** New `/api/v1/search` aggregates movies + TV + people. Existing per-type endpoints remain
- **zh-TW boost:** Application-layer ranking. If query matches zh-TW `title` field (substring), boost that result's position

### References

- [Source: apps/api/internal/tmdb/movies.go] — SearchMovies, SearchMoviesWithLanguage
- [Source: apps/api/internal/services/tmdb_service.go] — TMDbServiceInterface
- [Source: apps/web/src/routes/search.tsx] — Existing search page
- [Source: _bmad-output/planning-artifacts/prd/functional-requirements.md#3.5] — P2-013, P2-014

## Dev Agent Record

### Agent Model Used

claude-opus-4-8[1m] (Amelia — Dev Agent, dev-story workflow)

### Debug Log References

- Full Go suite: `cd apps/api && go test ./...` — all packages PASS
- Full web suite: `npx vitest run` — 158 files, 1935 tests PASS
- Lint gate: `pnpm lint:all` — EXIT 0 (0 errors; 121 pre-existing warnings only)

### Completion Notes List

- 🔗 AC Drift: NONE (checked: `/api/v1/search`, `instant search|SearchPeople|unified search` across `_bmad-output/implementation-artifacts/*.md` — the unified `/api/v1/search` endpoint is brand new; existing search routes `/tmdb/search/*`, `/movies/search`, `/library/search`, `/metadata/search` are untouched per AC #6 — all hits REUSE not DRIFT)
- 📎 Contract Stamps: NONE (no `[@contract-v*]` stamps in this story or upstream refs — normal for a story that defines/consumes no wire contracts)
- **Dual-language + zh-TW boost (AC #2, #3):** `SearchService` (apps/api/internal/services/search_service.go) queries zh-TW + en concurrently for movies and TV, dedups by TMDb ID (zh-TW metadata wins), and boosts items whose zh-TW localized title substring-matches the query above original-title-only matches.
- **People search (AC #1):** added `SearchPeople`/`SearchPeopleWithLanguage` on `*tmdb.Client`. Intentionally NOT added to `tmdb.ClientInterface` (would force every existing ClientInterface mock to grow a method); consumed via a narrow `services.SearchTMDbClient` interface that `*tmdb.Client` satisfies, exposed through `TMDbService.SearchClient()` (mirrors `VideosProvider()`).
- **Scope decision — unified categories:** AC #1 + Dev Notes specify movies + TV + people; Task 1.4's "local library" wording is superseded (local library FTS search already exists at `/api/v1/library/search`). Implemented movies + TV + people.
- **Route deviation (Task 3.5):** story wrote `/media/movie.{id}`; the actual TanStack route is `/media/$type/$id`, so navigation targets `/media/movie/{id}` and `/media/tv/{id}`. People rows are displayed but non-navigable (no person detail page exists).
- **Graceful degradation:** a per-category TMDb failure logs a warning and degrades that category to an empty list; only an all-categories failure returns an error.
- 🎨 UX Verification: comparison table below; one discrepancy found and fixed.

#### UX Verification (Step 9) — Design ref: `flow-g-search-desktop/as2-search-suggestions-dropdown.png` (Screen AS-2, pen node `TMaw5`)

| Area | Design Spec | Implementation | Match? | Fix Needed |
|------|------------|----------------|--------|------------|
| Container | rounded, `$bg-secondary`, subtle border, shadow | `rounded-xl bg-[var(--bg-secondary)] border shadow-xl` | ✅ | — |
| Sections | 電影 / 影集 / 人物, plain muted text labels | three sections, muted `text-xs` labels | ✅ | Removed leading section icons to match (initially added Film/Tv/User icons) |
| Movie/TV row | poster thumb + title + "OriginalTitle (Year) · ★ rating" | poster (w92) + title + meta + `★ {rating}` | ✅ | — |
| Person row | circular avatar + name + "導演 · Makoto Shinkai" | rounded-full avatar + name + `{zh dept} · {originalName}` | ✅ | — |
| Footer | centered accent "按 Enter 查看所有結果 →" | accent-colored centered button, same text | ✅ | — |
| Active highlight | first row lighter bg | `bg-[var(--bg-tertiary)]` on active/hover | ✅ | — |

- 🎨 UX Fix: removed the section-label icons (Film/Tv/User) from `SearchSuggestions` so the 電影/影集/人物 labels are plain muted text, matching Screen AS-2.

### File List

**Backend (Go)**
- `apps/api/internal/tmdb/types.go` (M) — added `Person`, `SearchResultPeople` types
- `apps/api/internal/tmdb/people.go` (A) — `SearchPeople` / `SearchPeopleWithLanguage` on `*Client`
- `apps/api/internal/tmdb/people_test.go` (A) — client people-search tests
- `apps/api/internal/services/search_service.go` (A) — `SearchService`, dual-language merge + zh-TW boost, `UnifiedSearchResult`, narrow `SearchTMDbClient` interface
- `apps/api/internal/services/search_service_test.go` (A) — merge/boost/dedup/degradation tests
- `apps/api/internal/services/tmdb_service.go` (M) — `SearchClient()` accessor
- `apps/api/internal/handlers/search_handler.go` (A) — `SearchHandler`, `GET /api/v1/search`
- `apps/api/internal/handlers/search_handler_test.go` (A) — handler tests
- `apps/api/cmd/api/main.go` (M) — wire `searchService` + `searchHandler`, register route

**Frontend (React/TS)**
- `apps/web/src/types/tmdb.ts` (M) — `Person`, `UnifiedSearchResult` types
- `apps/web/src/services/tmdb.ts` (M) — `unifiedSearch(query, page)`
- `apps/web/src/services/tmdb.spec.ts` (M) — `unifiedSearch` + backward-compat tests
- `apps/web/src/hooks/useSearchMedia.ts` (M) — `useInstantSearch` hook + `tmdbKeys.instant`
- `apps/web/src/components/search/SearchSuggestions.tsx` (A) — dropdown (電影/影集/人物), `buildNavigableItems`
- `apps/web/src/components/search/SearchSuggestions.spec.tsx` (A)
- `apps/web/src/components/search/InstantSearchBar.tsx` (A) — input + debounce + keyboard nav + mobile variant
- `apps/web/src/components/search/InstantSearchBar.spec.tsx` (A)
- `apps/web/src/components/shell/AppShell.tsx` (M) — use `InstantSearchBar` desktop + mobile full-screen overlay
- `apps/web/src/components/shell/AppShell.spec.tsx` (M) — updated for Enter-submit + mobile overlay

**E2E (Playwright)**
- `tests/e2e/instant-search.spec.ts` (A) — desktop dropdown journey (3 sections, click-nav, Enter→/search, arrow+Enter) + mobile full-screen view

**CR fixes (2026-06-04, Senior Developer Review — AI)**
- `apps/api/internal/services/search_service.go` (M) — people search now uses `SearchPeopleWithLanguage(zh-TW)` for zh-TW priority parity with movies/TV (narrow `SearchTMDbClient` interface updated)
- `apps/api/internal/services/search_service_test.go` (M) — stub updated to `SearchPeopleWithLanguage`
- `apps/api/cmd/api/main.go` (M) — fail-fast at startup if `SearchClient()` is nil (avoids per-request nil-panic)
- `apps/web/src/hooks/useSearchMedia.ts` (M) — `placeholderData: keepPreviousData` on `useInstantSearch` (no dropdown flicker between queries)
- `apps/web/src/components/search/SearchSuggestions.tsx` (M) — combobox a11y: options carry `searchOptionId` ids inside a listbox holding only `option`/`group` nodes; people render outside the listbox; person row `cursor-default`
- `apps/web/src/components/search/InstantSearchBar.tsx` (M) — `aria-activedescendant` points the combobox at the highlighted option
- `apps/web/src/components/search/SearchSuggestions.spec.tsx` (M) — option-id + listbox-containment assertions
- `apps/web/src/components/search/InstantSearchBar.spec.tsx` (M) — `aria-activedescendant` assertions on arrow-key nav

### Change Log

| Date | Change |
|------|--------|
| 2026-06-04 | Task 1–2 (backend): dual-language (zh-TW+en) unified search with TMDb-ID dedup + zh-TW title boost; people search; `GET /api/v1/search` endpoint wired in main.go. Tests added (service, handler, client). |
| 2026-06-04 | Task 3–4 (frontend): `SearchSuggestions` dropdown (電影/影集/人物, poster+title+year·rating, people w/ department), `InstantSearchBar` (300ms debounce, arrow+Enter keyboard nav, clear/Escape, desktop inline + mobile full-screen overlay); integrated into AppShell. `useInstantSearch` hook + `unifiedSearch` service. |
| 2026-06-04 | Task 5 (tests): backend merge/boost/dedup/degradation + handler + client tests; frontend debounce/keyboard/click-nav + service + backward-compat tests. Full regression green (Go all packages; web 1935 tests); `pnpm lint:all` EXIT 0. |
| 2026-06-04 | UX fix: removed section-label icons from `SearchSuggestions` to match Screen AS-2 design. |
| 2026-06-04 | E2E: added `tests/e2e/instant-search.spec.ts` — hermetic (mocked `/api/v1/search`) desktop dropdown + mobile full-screen coverage. |
| 2026-06-04 | CR fixes (AI review): people search zh-TW priority; startup fail-fast for nil search client; `keepPreviousData` (no flicker); combobox a11y (`aria-activedescendant`, listbox-only-options structure, `searchOptionId`); person-row affordance. Backend + 99 web tests green; gofmt/prettier clean. |
