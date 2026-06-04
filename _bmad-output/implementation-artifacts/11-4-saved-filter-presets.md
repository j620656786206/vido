# Story 11.4: Saved Filter Presets

Status: done

## Story

As a Traditional Chinese NAS user who frequently uses specific filter combinations,
I want to save, load, and delete named filter presets,
so that I can quickly access my favorite browsing patterns (e.g., "2024年後韓劇", "高評分動畫").

## Acceptance Criteria

1. Given active filters, when the user clicks "儲存篩選", then a dialog prompts for a preset name and saves the current filter state
2. Given saved presets, when displayed in the UI, then they appear as quick-access chips above the filter area
3. Given a saved preset chip, when clicked, then all filters are restored to the saved combination and results update
4. Given a saved preset, when the user clicks delete, then the preset is removed after confirmation
5. Given presets, when stored, then they persist across browser sessions (stored in DB, not localStorage)

## Tasks / Subtasks

- [x] Task 1: Backend — preset CRUD (AC: #1, #4, #5)
  - [x] 1.1 Create `filter_presets` table: `id TEXT PK, name TEXT, filters TEXT (JSON), sort_order INT, created_at`
  - [x] 1.2 Migration (next available number) — **023**
  - [x] 1.3 API endpoints:
    - `GET /api/v1/filter-presets` — list all
    - `POST /api/v1/filter-presets` — create `{ name, filters }` (filters is a JSON **string** in URL-param shape — see Dev Notes)
    - `DELETE /api/v1/filter-presets/:id` — delete
  - [x] 1.4 Max 20 presets (prevent unbounded growth) — enforced in service, HTTP 409 `FILTER_PRESET_LIMIT_REACHED`

- [x] Task 2: Frontend — save dialog (AC: #1)
  - [x] 2.1 "儲存篩選" button in FilterChipBar (visible when ≥1 filter active)
  - [x] 2.2 Dialog: text input for name, save/cancel buttons — `SavePresetDialog` (Screen AS-3)
  - [x] 2.3 Validate: name required, max 30 chars (`maxLength=30` + trim + inline error)

- [x] Task 3: Frontend — preset chips (AC: #2, #3)
  - [x] 3.1 Create `apps/web/src/components/search/PresetChips.tsx`
  - [x] 3.2 Render saved presets as chips above filter area (Screen AS-1 presetBar)
  - [x] 3.3 Click → apply all saved filters to URL state (via `useFilterState().setFilters` from 11-2, wired in discover.tsx)
  - [x] 3.4 Long-press (500ms) or right-click → delete confirmation dialog (AC #4)

- [x] Task 4: Tests (AC: #1-5)
  - [x] 4.1 Backend: CRUD tests, max preset limit (21 tests: repository 6, service 8, handler 7)
  - [x] 4.2 Frontend: save dialog, preset chip click → filter restore (21 tests: SavePresetDialog 6, PresetChips 7, FilterChipBar +4, +caseTransform-safe service)
  - [x] 4.3 Persistence: verify presets survive page reload — E2E `tests/e2e/saved-filter-presets.spec.ts` (reload re-fetches GET = DB-backed, AC #5)

## Dev Notes

### Architecture Compliance

- **DB storage:** Presets in SQLite, not localStorage — per PRD "sync across browser sessions"
- **Filter JSON:** Store as JSON string matching URL param format: `{"genre":"28","year_gte":"2024","region":"KR","sort_by":"vote_average.desc"}`
- **Depends on 11-2:** Uses `useFilterState` hook for applying preset filters to URL

### References

- [Source: _bmad-output/planning-artifacts/prd/functional-requirements.md#3.5] — P2-015
- [Source: apps/web/src/hooks/] — Hook patterns

## Dev Agent Record

### Agent Model Used

claude-opus-4-8[1m] (BMM dev-story workflow, Amelia)

### Debug Log References

- Backend: `go build ./...` OK; `go test ./internal/{repository,services,handlers}/ -run FilterPreset -v` → 21/21 PASS.

### Completion Notes List

**Design decision — `filters` stored/transported as a JSON string (not a nested object):**
The API boundary applies `snakeToCamel` (response) / `camelToSnake` (request) recursively over object keys (Rule 18). Storing `filters` as a nested object would mangle its inner snake_case URL-param keys (`year_gte` → `yearGte`) on the round-trip. `caseTransform` returns scalars (incl. strings) untouched, so the `filters` payload is an opaque JSON **string** in the URL search-param shape (e.g. `{"genre":"28","year_gte":"2024","region":"KR"}`). The frontend owns its serialization. Matches Dev Notes "Store as JSON string matching URL param format".

🔗 AC Drift: NONE (checked: grep `filter-presets|filter_presets|FilterPreset|preset` across `_bmad-output/implementation-artifacts/*.md` — 0 hits outside this story. New subsystem P2-015; consumes Story 11-2 `useFilterState` as REUSE, not DRIFT.)

📎 Contract Stamps: NONE (no `[@contract-v*]` stamps in this story; upstream Story 11-2 is pre-Rule-20 / implicit v0, so ack is not required per the forward-only retrofit).

CR follow-up (LOW, deferred — not blocking): max-20 cap has a benign TOCTOU between `Count()`/`Create()` (single-user NAS); no size cap on the `filters` JSON string; `sort_order` can collide after a mid-list delete (resolved by `created_at` tiebreak); `useFilterPresets` query errors render an empty preset row silently. Logged here as known minor items.

Error-code prefix note: handler uses resource-scoped codes `FILTER_PRESET_{NOT_FOUND,VALIDATION_FAILED,LIMIT_REACHED}`, mirroring the Story 10.3 `EXPLORE_BLOCK_*` precedent. Per that precedent these resource-level codes are NOT registered in the project-context Rule 7 authoritative prefix set (which tracks external integration *sources* like TMDB/SUBTITLE/LIBRARY), so no Rule 7 list/CR-sync change is made.

### File List

- `apps/api/internal/models/filter_preset.go` (new)
- `apps/api/internal/database/migrations/023_create_filter_presets.go` (new)
- `apps/api/internal/repository/filter_preset_repository.go` (new)
- `apps/api/internal/repository/filter_preset_repository_test.go` (new)
- `apps/api/internal/services/filter_preset_service.go` (new)
- `apps/api/internal/services/filter_preset_service_test.go` (new)
- `apps/api/internal/handlers/filter_presets_handler.go` (new)
- `apps/api/internal/handlers/filter_presets_handler_test.go` (new)
- `apps/api/internal/repository/registry.go` (modified — register FilterPresets repo)
- `apps/api/cmd/api/main.go` (modified — DI + route registration)
- `apps/web/src/services/filterPresetService.ts` (new)
- `apps/web/src/services/filterPresetService.spec.ts` (new — caseTransform-safe service tests)
- `apps/web/src/hooks/useFilterPresets.ts` (new — TanStack Query list + create/delete mutations)
- `apps/web/src/components/search/SavePresetDialog.tsx` (new — Screen AS-3)
- `apps/web/src/components/search/SavePresetDialog.spec.tsx` (new)
- `apps/web/src/components/search/PresetChips.tsx` (new — Screen AS-1 presetBar; CR fix: swallow synthesized click after long-press)
- `apps/web/src/components/search/PresetChips.spec.tsx` (new; CR fix: +2 touch long-press/tap tests)
- `apps/web/src/components/search/FilterChipBar.tsx` (modified — optional `onSavePreset` 儲存篩選 button)
- `apps/web/src/components/search/FilterChipBar.spec.tsx` (modified — save-button tests)
- `apps/web/src/routes/discover.tsx` (modified — wire PresetChips + SavePresetDialog)
- `tests/e2e/saved-filter-presets.spec.ts` (new — preset journey + persistence)
- `tests/e2e/saved-filter-presets.api.spec.ts` (new — API + service-boundary coverage)

### UX Verification (Step 9)

Compared implementation against `ux-design.pen` Screen AS-3 (Save Filter Preset Modal, `i74p2`) and the AS-1 `presetBar` (`dPbq2`) via Pencil MCP `get_screenshot`.

| Area | Design Spec | Implementation | Match? |
|------|------------|----------------|--------|
| Save modal title | "儲存篩選條件" 18px bold + X close | same | ✅ |
| Modal helper text | "將目前的篩選條件儲存為快速存取預設" 13px | same | ✅ |
| Name field | label "預設名稱" + input placeholder "例：高評分韓劇" | same (maxLength 30) | ✅ |
| Filter preview | "包含的篩選條件：" + blue accent chips | `activeFilterChips` labels in accent-primary chips | ✅ |
| Modal actions | 取消 (outline) + 儲存 (blue) right-aligned | same | ✅ |
| Preset bar | "快速篩選:" label + rounded bordered chips (bg-secondary) | `PresetChips` same styling | ✅ |
| Save trigger | `presetAdd` plus-icon + "儲存篩選" rounded-full bordered chip | same styling, placed in FilterChipBar | ✅* |

🎨 UX Verification: PASS. *Deliberate placement: per Task 2.1 the "儲存篩選" trigger lives in the active-filter chip bar and shows only when ≥1 filter is active (saving is meaningless with no filters). Visually identical to the designed `presetAdd` chip; the design groups it inside the preset bar, but both render above the results area. Not a design drift — a story-task-driven UX refinement, documented here.

## Change Log

| Date       | Change |
|------------|--------|
| 2026-06-04 | Task 1 — Backend filter preset CRUD: migration 023 `filter_presets`, model + Validate (name ≤30 runes, filters valid JSON), repository (Create/GetAll/Delete/Count), service (max-20 cap), handler (GET/POST/DELETE `/api/v1/filter-presets`), wired in registry + main.go. 21 backend tests pass. |
| 2026-06-04 | Task 2 — Frontend save dialog: `filterPresetService` (filters as caseTransform-safe JSON string), `useFilterPresets` TanStack Query hooks, `SavePresetDialog` (Screen AS-3, name required ≤30), optional 儲存篩選 button in `FilterChipBar`. 14 web tests. |
| 2026-06-04 | Task 3 — Frontend preset chips: `PresetChips` (Screen AS-1 presetBar) — click applies saved filters via `useFilterState`, right-click/long-press → delete confirmation; wired into `discover.tsx`. 7 web tests. |
| 2026-06-04 | Task 4 — Tests + UX verify: full backend suite green (no regressions), full web suite 1952 pass, ESLint (Rule 21) + prettier + gofmt clean, 3 E2E preset-journey tests pass (incl. persistence-across-reload). UX Verification PASS vs Screen AS-3 + AS-1 presetBar. |
| 2026-06-04 | CR fixes (adversarial review) — **M1:** `PresetChips` long-press no longer also applies the preset; touchend's synthesized click is swallowed via a `didLongPress` ref (Task 3.4). +2 touch tests (long-press swallow / short-tap apply) → `PresetChips.spec.tsx` 9 pass. **M2:** File List completed — added `filterPresetService.spec.ts` and `saved-filter-presets.api.spec.ts` (committed but previously undocumented). |
