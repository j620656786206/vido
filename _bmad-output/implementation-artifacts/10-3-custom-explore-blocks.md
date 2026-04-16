# Story 10.3: Custom Explore Blocks CRUD

Status: review

## Story

As a Traditional Chinese NAS user,
I want to create and customize themed content blocks on the homepage (e.g., "近期台灣院線", "熱門韓劇"),
so that the homepage shows discovery content relevant to my interests.

## Acceptance Criteria

1. Given the homepage, when explore blocks are configured, then each block shows a horizontal scrollable row of poster cards with a section title
2. Given the settings/homepage section, when a user creates a new block, then they can specify: name, content type (movie/tv), genre filter, language/region filter, sort order, and max items
3. Given multiple explore blocks, when the user reorders them, then the homepage reflects the new order immediately
4. Given an explore block, when the user edits or deletes it, then the homepage updates without page reload
5. Given a fresh install, when no blocks are configured, then 3 default blocks are created: "熱門電影", "熱門影集", "近期新片"
6. Given explore block data, when fetched from TMDB, then results use the discover API with the block's filter parameters

## Tasks / Subtasks

- [x] Task 1: Backend — explore blocks CRUD (AC: #2, #3, #4)
  - [x] 1.1 Create `explore_blocks` table: `id TEXT PK, name TEXT, content_type TEXT, genre_ids TEXT, language TEXT, region TEXT, sort_by TEXT, max_items INT, sort_order INT, created_at, updated_at`
  - [x] 1.2 Create migration #022 (or next available)
  - [x] 1.3 Create `ExploreBlockRepository` interface + SQLite implementation
  - [x] 1.4 API endpoints:
    - `GET /api/v1/explore-blocks` — list all blocks (ordered)
    - `POST /api/v1/explore-blocks` — create block
    - `PUT /api/v1/explore-blocks/:id` — update block
    - `DELETE /api/v1/explore-blocks/:id` — delete block
    - `PUT /api/v1/explore-blocks/reorder` — batch reorder
  - [x] 1.5 Seed default blocks on first run (AC: #5)

- [x] Task 2: Backend — explore block content endpoint (AC: #6)
  - [x] 2.1 `GET /api/v1/explore-blocks/:id/content` — fetch TMDB discover results using block's saved filters
  - [x] 2.2 Use Story 10-1's `DiscoverMovies`/`DiscoverTVShows` with block params
  - [x] 2.3 Apply content filters (far-future, low-quality) from Story 10-1's `ContentFilterService`
  - [x] 2.4 Cache results per block (1-hour TTL)

- [x] Task 3: Frontend — ExploreBlock component (AC: #1)
  - [x] 3.1 Create `apps/web/src/components/homepage/ExploreBlock.tsx`
  - [x] 3.2 Section title + horizontal scrollable row of `PosterCard`
  - [x] 3.3 "查看更多" link at end of row
  - [x] 3.4 Loading skeleton while fetching

- [x] Task 4: Frontend — block management UI (AC: #2, #3, #4)
  - [x] 4.1 Add "自訂首頁" section in Settings page
  - [x] 4.2 Create/Edit modal: block name, content type selector, genre multi-select, language/region, sort, max items
  - [x] 4.3 Drag-to-reorder list (or up/down arrows)
  - [x] 4.4 Delete with confirmation

- [x] Task 5: Frontend — homepage integration (AC: #1, #5)
  - [x] 5.1 Create `apps/web/src/hooks/useExploreBlocks.ts` — fetch blocks + their content
  - [x] 5.2 Render ExploreBlock for each configured block below HeroBanner
  - [x] 5.3 TanStack Query with staleTime: 5 minutes

- [x] Task 6: Tests (AC: #1-6)
  - [x] 6.1 Backend: repository CRUD tests (13), handler tests (17), service tests (17), seed logic covered
  - [x] 6.2 Frontend: ExploreBlock render (7), ExploreBlocksList (4), ExploreBlocksSettings (7), ExploreBlockEditModal (4)
  - [x] 6.3 Integration: 9 Playwright E2E — homepage render, settings CRUD, reorder, delete confirm, API contract

## Dev Notes

### Architecture Compliance

- **DB pattern:** Follow existing repository pattern (`MovieRepositoryInterface` etc.)
- **Migration:** Next available number after Epic 9c's #021
- **API format:** `ApiResponse<T>` wrapper, snake_case (Rule 18)
- **Frontend state:** TanStack Query for server state, no local state management needed
- **Drag reorder:** Use `@dnd-kit/sortable` or simple up/down buttons (simpler for single-user)

### Cross-Stack Split Check

⚠️ This story has 3 backend tasks and 3 frontend tasks. At the boundary but acceptable — the backend CRUD is straightforward and the frontend is a natural consumer. No split recommended.

### References

- [Source: apps/api/internal/services/tmdb_service.go] — TMDB service interface
- [Source: apps/web/src/components/media/PosterCard.tsx] — Card component to reuse
- [Source: _bmad-output/planning-artifacts/prd/functional-requirements.md#3.4] — P2-002 spec

## Dev Agent Record

### Agent Model Used

- Amelia (dev agent) on `claude-opus-4-6[1m]` — Story 10.3 implementation 2026-04-15/16.

### Debug Log References

- Backend regression gate (`pnpm nx test api`): 31 Go packages PASS (including new `explore_block_repository_test.go`, `explore_block_service_test.go`, `explore_blocks_handler_test.go`). Migration registry 22 entries.
- Frontend regression gate (`pnpm nx test web`): 139 spec files / 1696 tests PASS (+ 22 new: ExploreBlock 7, ExploreBlocksList 4, ExploreBlocksSettings 7, ExploreBlockEditModal 4; SettingsLayout `SETTINGS_CATEGORIES` count assertion updated 8 → 9).
- Full lint gate (`pnpm lint:all`): 0 errors across go vet / staticcheck@2026.1 / eslint . / prettier --check. After `pnpm nx reset` the phantom `@vido/source:lint` target (pre-existing issue, matches `preexisting-fail-root-lint-target` sprint entry) is not inferred — 3 lint targets run (shared-types, api, web).
- E2E (`npx playwright test tests/e2e/explore-blocks.spec.ts --project=chromium`): 9/9 PASS in 13.2s.

### Completion Notes List

- **AC #1** Horizontal scrollable rows + "查看更多" link — `ExploreBlock.tsx` renders `PosterCard` in a `snap-x` scroller with desktop-only chevron buttons and a responsive skeleton set (6 placeholders) while loading. `ExploreBlocksList.tsx` maps configured blocks to sections below `HeroBanner` in `routes/index.tsx`.
- **AC #2** Create flow — `/settings/homepage` (new route) hosts `ExploreBlocksSettings` → `ExploreBlockEditModal` with fields: name, content type, genre IDs, language, region, sort, max items (1–40). Labels updated to match design brief `hp3-block-crud-modal.png` ("新增探索區塊" / "儲存區塊").
- **AC #3** Reorder — up/down arrow buttons on each row swap adjacent blocks and call `PUT /explore-blocks/reorder` with the new id array. Repository wraps the update in a single tx; unknown IDs roll the whole batch back (test `TestExploreBlockRepository_Reorder_UnknownIDRollsBack`).
- **AC #4** Edit/Delete without page reload — mutations invalidate the `explore-blocks` query key so the list updates in place; delete goes through a confirm dialog before the `DELETE` fires (tested).
- **AC #5** Fresh-install seeding — `ExploreBlockService.SeedDefaultsIfEmpty` runs once in `main.go` after service wiring; inserts 熱門電影 / 熱門影集 / 近期新片 with incrementing sort_order. Idempotent on subsequent boots (count-check first).
- **AC #6** TMDb discover integration — `ExploreBlockService.GetBlockContent` builds `tmdb.DiscoverParams` from the block's stored filters, delegates to `TMDbServiceInterface.DiscoverMovies`/`DiscoverTVShows` (Story 10-1), then applies `ContentFilterService.FilterFarFuture*` + `FilterLowQuality*` (Epic 10 content policy), caps at `max_items`, and caches the payload under `explore_block:<id>` with 1h TTL via `CacheRepository`. Update/Delete invalidate that cache so config changes surface immediately.

**Architectural decisions:**

- Content type domain: used TMDb's `tv` (not `series`) to mirror `/discover/tv` endpoint semantics — tests assert CHECK constraint `content_type IN ('movie', 'tv')`.
- Error codes: `EXPLORE_BLOCK_NOT_FOUND` / `EXPLORE_BLOCK_VALIDATION_FAILED` per Rule 7's `{SOURCE}_{ERROR_TYPE}` convention; 404 and 400 responses keep the `ApiResponse<T>` envelope (Rule 3).
- Route ordering: `PUT /explore-blocks/reorder` registered BEFORE `PUT /explore-blocks/:id` so Gin's literal-path priority wins — regression test `TestExploreBlocksHandler_ReorderRouteDoesNotCollideWithID` locks this in.
- Cache invalidation fan-out: `Update` and `Delete` both call `invalidateContentCache(id)` — tested via `TestExploreBlockService_UpdateBlock_InvalidatesCache` (asserts TMDb is re-hit after edit).
- Cross-stack split check: Story has 2 backend + 3 frontend tasks (shared Task 6 for tests). Kept unified per retro-9c-AI1 threshold (`BE>3 AND FE>3`).

🎨 **UX Verification: PASS with scoped follow-ups**

Compared against `flow-g-homepage-desktop/hp1-homepage-desktop.png`, `flow-g-homepage-mobile/hp2-homepage-mobile.png`, and `flow-g-homepage-desktop/hp3-block-crud-modal.png`:

| Area | Design Spec | Implementation | Match? | Fix Needed |
|------|------------|----------------|--------|------------|
| Modal title | 新增探索區塊 | 新增探索區塊 | ✅ | — (fixed 2026-04-16) |
| Save button | 儲存區塊 | 儲存區塊 | ✅ | — (fixed 2026-04-16) |
| Block row layout | horizontal scroll, poster cards | horizontal scroll + PosterCard | ✅ | — |
| Section title | top-left, section-header weight | text-lg/xl semibold | ✅ | — |
| See-more link | top-right with chevron | 查看更多 + ChevronRight | ✅ | — |
| Hero + blocks order | hero → 3 rows stacked | `<HeroBanner/>` → `<ExploreBlocksList/>` | ✅ | — |
| Loading skeleton | 6 placeholder cards | Array(6) × `PosterCardSkeleton` | ✅ | — |
| Mobile layout | full-width scrollers, hero compressed | `max-w-7xl` + `w-[140px]` cards on `sm` | ✅ | — |
| **Modal: genre chips** | multi-select pill chips (動作/科幻/...) | free-form `genre_ids` text input | ⚠️ Partial | **Follow-up:** proper multi-select UI needs a TMDb genre list endpoint (not in story AC). Current text input accepts the same comma-separated IDs the backend already stores and is functional but not design-parity. |
| **Modal: language/region** | combined dropdown "zh-TW 台灣" | two side-by-side text inputs | ⚠️ Partial | **Follow-up:** unifying into a single typed dropdown needs a curated locale list. Current split maps 1:1 to the backend fields (`language`, `region`) and is functional. |

The two partial-parity items (genre chip picker, combined locale dropdown) are UI polish that does not touch the data model or API contract — they can ship later as a pure-frontend follow-up without a migration or API change. All six ACs are satisfied end-to-end with the current implementation.

### File List

**Backend (new)**

- `apps/api/internal/models/explore_block.go`
- `apps/api/internal/database/migrations/022_create_explore_blocks.go`
- `apps/api/internal/repository/explore_block_repository.go`
- `apps/api/internal/repository/explore_block_repository_test.go`
- `apps/api/internal/services/explore_block_service.go`
- `apps/api/internal/services/explore_block_service_test.go`
- `apps/api/internal/handlers/explore_blocks_handler.go`
- `apps/api/internal/handlers/explore_blocks_handler_test.go`

**Backend (modified)**

- `apps/api/internal/repository/registry.go` — `ExploreBlocks` field + `NewExploreBlockRepository` wiring in both `NewRepositories` and `NewRepositoriesWithCache`.
- `apps/api/cmd/api/main.go` — initialise `exploreBlockService`, call `SeedDefaultsIfEmpty` at boot, wire `exploreBlocksHandler.RegisterRoutes(apiV1)`.

**Frontend (new)**

- `apps/web/src/services/exploreBlockService.ts`
- `apps/web/src/hooks/useExploreBlocks.ts`
- `apps/web/src/components/homepage/ExploreBlock.tsx`
- `apps/web/src/components/homepage/ExploreBlock.spec.tsx`
- `apps/web/src/components/homepage/ExploreBlocksList.tsx`
- `apps/web/src/components/homepage/ExploreBlocksList.spec.tsx`
- `apps/web/src/components/settings/ExploreBlocksSettings.tsx`
- `apps/web/src/components/settings/ExploreBlocksSettings.spec.tsx`
- `apps/web/src/components/settings/ExploreBlockEditModal.tsx`
- `apps/web/src/components/settings/ExploreBlockEditModal.spec.tsx`
- `apps/web/src/routes/settings/homepage.tsx`

**Frontend (modified)**

- `apps/web/src/components/settings/SettingsLayout.tsx` — import `LayoutGrid` icon, insert the `homepage` entry between `scanner` and `cache`.
- `apps/web/src/components/settings/SettingsLayout.spec.tsx` — `SETTINGS_CATEGORIES` length assertion 8 → 9.
- `apps/web/src/routes/index.tsx` — render `<ExploreBlocksList/>` below `<HeroBanner/>`.
- `apps/web/src/routeTree.gen.ts` — auto-regenerated to register `/settings/homepage`.

**Tests (new)**

- `tests/e2e/explore-blocks.spec.ts` — 9 Playwright scenarios tagged `@ui @explore-blocks @story-10-3` + `@api` for contract checks.

## Change Log

- **2026-04-16** — Story 10.3 implementation complete. Backend: migration 022 `explore_blocks`, `ExploreBlockRepository` + `ExploreBlockService` + `ExploreBlocksHandler` (6 endpoints, 1h per-block cache, 3 seeded defaults). Frontend: `ExploreBlock` / `ExploreBlocksList` homepage components + `ExploreBlocksSettings` at `/settings/homepage` + `ExploreBlockEditModal`. All 6 ACs satisfied. Regression: `nx test api` 31 packages PASS, `nx test web` 1696/1696 PASS (+22 new), `pnpm lint:all` 0 errors, E2E 9/9 PASS. UX verification vs `flow-g-homepage-*/hp*` screenshots: structural parity for homepage and settings; 2 modal-UI polish items (genre chip picker, combined locale dropdown) deferred as pure-frontend follow-ups — no backend or contract change needed.
