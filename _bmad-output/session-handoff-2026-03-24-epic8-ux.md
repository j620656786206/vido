# Session Handoff: Epic 8 Subtitle Engine — 2026-03-24 (Post UX Design)

## What Was Done This Session

### UX Design (Sally — Pencil MCP)
- Created **Flow I** in `ux-design.pen` — 6 screens for Story 8-8 and 8-9:
  - **I1**: Subtitle Search Dialog Desktop (table with score color coding, HI badges, format column)
  - **I2**: Preview Popover + Download States Desktop (success/loading/error/retry)
  - **I3**: Mobile Search Dialog (card-based layout, provider chips)
  - **I4**: Batch Progress Desktop (progress card overlay)
  - **I5**: Batch Progress Mobile (bottom sheet with stats dashboard)
  - **I6**: Mobile Subtitle Preview (SRT content display + encoding badge)
- Updated **existing screens** with "搜尋字幕" entry points:
  - B4, B4b Detail Panels — added button
  - B4a, B4c Context Menus — added menu item
  - E-04a, E-04c, E-05 Mobile — added button/menu item
- Reorganized canvas: closed 2000px gap, unified label colors (#222222/#666666/#888888)
- Exported 39/39 screenshots to `_bmad-output/screenshots/`
- Updated `scripts/export-pen-screenshots.py` with Flow I mappings

### Sprint Change Proposal (Correct Course)
- **CN Content Subtitle Conversion Policy** — approved by Alexyu
- Mainland China content (production_countries contains "CN") keeps Simplified Chinese subtitles
- User can override via "繁體轉換" toggle in search dialog
- Sprint Change Proposal: `_bmad-output/planning-artifacts/sprint-change-proposal-2026-03-24.md`
- Updated `project-context.md` Rule 9 with CN conversion policy
- Updated Story 8-8: new ACs #9-#11, new Task 5 (CN Conversion Policy Backend)
- Updated Story 8-9: new AC #9 (batch CN awareness)
- Tasks reordered: Backend (1-4) → CN Policy (5) → Frontend (6-9) → Integration (10) → Tests (11-12)

### Commits Made
1. `2350776` feat: add Flow I UX designs for Story 8-8/8-9
2. `58c5278` docs: sprint change proposal — CN content subtitle conversion policy
3. `5e33e8b` docs: reorder Story 8-8 tasks — CN conversion policy as Task 5

## What Needs to Happen Next

### Priority 1: Story 8-8 Code Review + Implementation
1. Run `/bmad:bmm:workflows:code-review` for Story 8-8
   - Story status: `review` (code complete for original scope)
   - **NEW tasks need implementation**: Task 5 (CN conversion policy backend), Task 8.4 (繁體轉換 toggle)
   - UX verification against Flow I screenshots required
2. Implement remaining tasks (Task 5 CN policy + Task 8.4 toggle + Task 10.2 productionCountry prop)
3. Run tests, commit, mark done

### Priority 2: Story 8-9 Development
1. Run `/bmad:bmm:workflows:dev-story` for Story 8-9 (Batch Subtitle Processing)
2. AC #9 (CN batch awareness) is new — ensure batch processor passes productionCountry to engine
3. Code review after implementation

### Priority 3: Epic 8 Wrap-up
1. Full regression: `go test ./...`
2. Update sprint-status: epic-8 → done
3. Run `/bmad:bmm:workflows:retrospective` for Epic 8

## Sprint Status
```
epic-8: in-progress
8-1 through 8-7, 8-10: done
8-8: review (code complete, needs CR + CN policy implementation + UX verification)
8-9: ready-for-dev
```

## Key Design Decisions to Remember
- **Table columns**: 來源, 語言, 字幕名稱 (with HI badge + ⓘ in header), 格式 (SRT/ASS), 評分, 下載數, 操作
- **Column gap**: 24px between all columns
- **Score badges**: green >70%, yellow >40%, red ≤40% with border + 40% alpha fill
- **HI badge**: inline after subtitle name, orange (#F59E0B), with ⓘ info icon in column header
- **Long names**: truncated with "...", HI badge always visible after truncation
- **Mobile**: card-based layout (not table), bottom sheet navigation (I3 → I6 slide within same sheet)
- **Entry points**: Desktop = Detail Panel button + hover context menu; Mobile = Detail Panel button + context menu
- **CN Policy**: production_countries "CN" → skip OpenCC, 繁體轉換 toggle for user override

## Loop Schedule
- Was set to every 2h `/bmad:bmm:workflows:dev-story` but cancelled for session handoff
- Re-create in new session: `/loop 2h /bmad:bmm:workflows:dev-story`

## Key Files Modified This Session
- `ux-design.pen` — Flow I screens + entry point updates to existing screens
- `scripts/export-pen-screenshots.py` — Flow I screen mappings
- `_bmad-output/screenshots/flow-i-subtitle-desktop/` — 3 new screenshots
- `_bmad-output/screenshots/flow-i-subtitle-mobile/` — 3 new screenshots
- `project-context.md` — Rule 9 CN conversion policy
- `_bmad-output/implementation-artifacts/8-8-manual-subtitle-search-ui.md` — ACs #9-11, Task 5, reordered
- `_bmad-output/implementation-artifacts/8-9-batch-subtitle-processing.md` — AC #9
- `_bmad-output/planning-artifacts/sprint-change-proposal-2026-03-24.md` — new
