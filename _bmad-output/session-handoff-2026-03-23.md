# Session Handoff — 2026-03-23

## What Was Done This Session

### PRD v4 Migration (Complete)
- Rewrote all planning documents from v3 (3 phases, 14 epics, 94 FRs) to v4 (4 phases, 12 epics 7-18, ~60 feature IDs)
- 16 commits total for documentation migration
- Key changes: auth removed, subtitle engine promoted to Phase 1, new plugin/SSE/request/dashboard concepts

### Phase 0 Prerequisites (Complete)
- Migration 018: subtitle fields on movies/series tables
- Migration 019: is_removed soft-delete field
- Repository extensions: BulkCreate, FindByParseStatus, subtitle query methods, FindAllWithFilePath
- SSE Hub: apps/api/internal/sse/ package with hub + handler + tests
- All wired into main.go

### Epic 7 Stories Created (All 4 ready-for-dev)
- 7-1: Recursive Folder Scanner ✅ DONE
- 7-2: Scheduled Scan Service ✅ DONE
- 7-3: Manual Scan Trigger UI — ready-for-dev (frontend, needs UX design first)
- 7-4: Scan Progress Tracking — ready-for-dev (frontend, needs UX design first)

### Story 7-1 Implementation (Done)
- ScannerService: recursive walk, symlink following, dedup, video format filter, BulkCreate batch, SSE progress
- ScannerHandler: POST /scan, GET /status, POST /cancel
- 18 service tests + 6 handler tests
- CR fixes: context.Background() for goroutine, sentinel error, TOCTOU race fix

### Story 7-2 Implementation (Done)
- ScanScheduler: time.Ticker with manual/hourly/daily, Start/Stop/Reconfigure
- Incremental scan: mtime comparison, IsRemoved soft-delete, detectRemovedFiles
- Schedule API: GET/PUT /api/v1/scanner/schedule
- 7 scheduler tests + 2 incremental tests + 5 handler tests
- CR fixes: DB columns in Update/BulkCreate/FindByFilePath, ParseStatus reset, Reconfigure race

## What Needs to Be Done Next

### Immediate: Flow H UX Design in Pencil (BLOCKING Story 7-3/7-4)

Design these screens in `ux-design.pen` using Pencil MCP tools:

**Desktop (1440×900):**
1. H1: Settings — 媒體庫掃描設定 section (media folder paths, scan schedule selector, "Scan Now" button, last scan info)
2. H2: Scan Progress floating card (bottom-right corner, progress bar, files found/current file/ETA, minimize/cancel buttons)
3. H3: Scan Results summary toast (auto-dismiss 10s, X found/Y new/Z errors)

**Mobile (390×844):**
4. H4: Settings — 媒體庫掃描設定 (mobile version)
5. H5: Scan Progress bottom sheet (peek bar + expandable)

**Design constraints:**
- Dark theme: bg-primary=#1B2336, bg-secondary=#24304A, accent=#3B82F6
- Follow existing Settings page layout (sidebar nav desktop, tab nav mobile)
- Use existing components: ButtonPrimary, Input, Select, Progress bar
- Lucide icons, Inter font
- Reference: `scanner-ui-design-brief.md` for detailed specs

### After UX Design Approval

1. Export screenshots: `python3 scripts/export-pen-screenshots.py`
2. Commit screenshots + .pen file together
3. Update Story 7-3 and 7-4 Dev Notes with screenshot references
4. Dev Story 7-3 (Manual Scan Trigger UI) → TA → CR → Fix → Done
5. Dev Story 7-4 (Scan Progress Tracking) → TA → CR → Fix → Done
6. Epic 7 Retrospective

### After Epic 7 Complete

1. Start Epic 8 (Subtitle Engine) — follow same BMAD flow:
   - Party Mode gate review for Epic 8
   - Sprint Planning (add story-level entries)
   - Create all 10 stories
   - Dev Story → TA → CR → Fix → Done for each
2. Epic 9 (AI Subtitle Enhancement)
3. Then Phase 2 epics (10-12)

## Current Sprint Status

```yaml
epic-7: in-progress
7-1: done
7-2: done
7-3: ready-for-dev  # BLOCKED on UX design
7-4: ready-for-dev  # BLOCKED on UX design
epic-8 through epic-18: backlog
```

## Key Files

- Sprint status: `_bmad-output/implementation-artifacts/sprint-status.yaml`
- Epic 7: `_bmad-output/planning-artifacts/epics/epic-7-media-library-scanner.md`
- Story 7-3: `_bmad-output/implementation-artifacts/7-3-manual-scan-trigger-ui.md`
- Story 7-4: `_bmad-output/implementation-artifacts/7-4-scan-progress-tracking.md`
- Scanner design brief: `_bmad-output/planning-artifacts/scanner-ui-design-brief.md`
- Subtitle design brief: `_bmad-output/planning-artifacts/subtitle-engine-design-brief.md`
- UX gap analysis: `_bmad-output/planning-artifacts/ux-design-gap-analysis-v4.md`
- UX design file: `ux-design.pen`
- Project context: `project-context.md`

## Design System Variables (for Pencil)
- bg-primary: #1B2336
- bg-secondary: #24304A
- bg-tertiary: #2E3B56
- accent-primary: #3B82F6
- text-primary: #F2F2F2
- text-secondary: #B3B3B3
- text-muted: #808080
- border-subtle: #374461
- success: #22C55E
- warning: #F59E0B
- error: #EF4444
- radius-md: 8, radius-lg: 12
- gap-sm: 8, gap-md: 12, gap-lg: 16, gap-xl: 24
