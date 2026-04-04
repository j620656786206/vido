# Story 11.4: Saved Filter Presets

Status: ready-for-dev

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

- [ ] Task 1: Backend — preset CRUD (AC: #1, #4, #5)
  - [ ] 1.1 Create `filter_presets` table: `id TEXT PK, name TEXT, filters TEXT (JSON), sort_order INT, created_at`
  - [ ] 1.2 Migration (next available number)
  - [ ] 1.3 API endpoints:
    - `GET /api/v1/filter-presets` — list all
    - `POST /api/v1/filter-presets` — create `{ name, filters: { genre, year_gte, ... } }`
    - `DELETE /api/v1/filter-presets/:id` — delete
  - [ ] 1.4 Max 20 presets (prevent unbounded growth)

- [ ] Task 2: Frontend — save dialog (AC: #1)
  - [ ] 2.1 "儲存篩選" button in FilterChipBar (visible when ≥1 filter active)
  - [ ] 2.2 Dialog: text input for name, save/cancel buttons
  - [ ] 2.3 Validate: name required, max 30 chars

- [ ] Task 3: Frontend — preset chips (AC: #2, #3)
  - [ ] 3.1 Create `apps/web/src/components/search/PresetChips.tsx`
  - [ ] 3.2 Render saved presets as chips above filter area
  - [ ] 3.3 Click → apply all saved filters to URL state (via `useFilterState` from 11-2)
  - [ ] 3.4 Long-press or right-click → delete option

- [ ] Task 4: Tests (AC: #1-5)
  - [ ] 4.1 Backend: CRUD tests, max preset limit
  - [ ] 4.2 Frontend: save dialog, preset chip click → filter restore
  - [ ] 4.3 Persistence: verify presets survive page reload

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

### Debug Log References

### Completion Notes List

### File List
