# Session Handoff — 2026-03-23 (Session 2)

## What Was Done This Session

### Flow H UX Design (Complete — 8 screens)
- Designed all 8 Flow H screens in `ux-design.pen` via Pencil MCP tools
- Desktop: H1 (Settings Scanner), H2 (Progress Card), H3 (Complete Toast), H7 (Filtered Library Unmatched)
- Mobile: H4 (Settings Scanner), H5 (Progress Bottom Sheet), H6 (Complete Toast), H8 (Filtered Library Unmatched)
- Added screen labels, arrows with trigger descriptions between screens
- Updated export script (`scripts/export-pen-screenshots.py`) with 8 new screen entries
- Exported 33/33 screenshots successfully
- Committed: `03addf3` — feat: add Flow H Scanner UI design

### Story 7-3: Manual Scan Trigger UI (Complete — done)
- Created `scannerService.ts` with fetchApi pattern, ScannerApiError class, 5 API methods
- Created `useScanner.ts` with 5 TanStack Query hooks (key factory pattern)
- Created `ScannerSettings.tsx` component with folder list, schedule selector, last scan info, scan button
- Added scanner nav item to `SettingsLayout.tsx` (now 8 categories)
- Created route `routes/settings/scanner.tsx`
- Committed: `aa45d0e` — feat: implement Story 7-3

### Story 7-3 TA + CR (Complete)
- TA Score: 82/100 (A - Good)
- CR found 5 issues (2 High, 3 Medium), all fixed:
  1. Fixed broken test assertion (scannerService.spec.ts:44)
  2. Corrected task 3.5/3.6 descriptions (SSE deferred to 7-4)
  3. Added 3 missing hook tests (useTriggerScan, useCancelScan, useUpdateScanSchedule)
  4. Fixed setTimeout memory leak with useRef cleanup
  5. Added routeTree.gen.ts to File List + committed it
- Committed: `2728d52` — fix: resolve Story 7-3 code review issues
- Committed: `3057256` — chore: regenerate routeTree

## What Needs to Be Done Next

### Story 7-4: Scan Progress Tracking (ready-for-dev)

This is the largest story in Epic 7 with 5 tasks. Follow BMAD `dev-story` workflow.

**Task 1: ScanProgressCard (desktop)**
- Floating card, fixed bottom-right, z-50, 400px width
- Progress bar + stats (found/parsed/matched/errors) + current file + ETA
- Minimize to pill ("掃描中 62%"), expand back
- Completion summary with action links ("查看未比對項目", "查看錯誤")
- Auto-dismiss 10s with shrinking progress bar
- Design ref: `_bmad-output/screenshots/flow-h-scanner-desktop/h2-scan-progress-desktop.png`

**Task 2: useScanProgress hook (SSE)**
- This is the FIRST SSE integration in the frontend (no existing pattern)
- Connect EventSource to `/api/v1/events`, filter `scan_progress` type
- useReducer for state: isScanning, progress, currentFile, filesFound, etc.
- Reconnection fallback: poll GET /scanner/status on SSE error/timeout (5s)
- Backend SSE hub: `apps/api/internal/sse/hub.go`

**Task 3: Cancel scan flow**
- Cancel button on progress card
- Confirmation dialog with Traditional Chinese copy
- POST /api/v1/scanner/cancel

**Task 4: Mobile bottom sheet**
- Peek state (64px, full width): loader + % + file count
- Expanded state (half screen): progress bar, stats (2 rows), ETA, cancel
- Drag handle, swipe-to-collapse
- Design ref: `_bmad-output/screenshots/flow-h-scanner-mobile/h5-scan-progress-mobile.png`

**Task 5: Responsive wrapper + app integration**
- ScanProgress wrapper: ≥768px → Card, <768px → Sheet
- Add to app root layout (visible on all pages during scan)
- Stack with AI Parse Progress Card if both active (12px gap)

### After Story 7-4
1. TA + CR for Story 7-4
2. Epic 7 Retrospective
3. Start Epic 8 (Subtitle Engine)

## Key Technical Decisions Made This Session

1. **SSE deferred from Story 7-3 to 7-4**: Story 7-3 uses TanStack Query polling (3s while scanning, 30s idle) for scan status. Story 7-4 will implement the actual SSE EventSource hook.

2. **Folder paths are read-only**: Media folder paths are managed via `VIDO_MEDIA_DIRS` env var, not editable in UI. The settings page shows an informational message about this.

3. **Notification pattern**: No global toast library in the project. ScannerSettings uses inline notification with useRef-based auto-dismiss timer (5s). Story 7-4's completion toast will use a different pattern (top-center absolute positioned card).

4. **Design system tokens**: bg-primary=#1B2336, bg-secondary=#24304A, bg-tertiary=#2E3B56, accent-primary=#3B82F6, text-primary=#F2F2F2, text-secondary=#B3B3B3, text-muted=#808080, border-subtle=#374461.

## Current Sprint Status

```yaml
epic-7: in-progress
7-1: done
7-2: done
7-3: done  # 2026-03-23 — TA 82/100, CR fixes applied
7-4: ready-for-dev  # NEXT
epic-8 through epic-18: backlog
```

## Key Files for Story 7-4

### Existing (created in 7-3, extend in 7-4)
- `apps/web/src/services/scannerService.ts` — has cancelScan(), getScanStatus(), getSSEUrl()
- `apps/web/src/hooks/useScanner.ts` — has useScanStatus (polling), useCancelScan

### To Create in 7-4
- `apps/web/src/components/scanner/ScanProgressCard.tsx` — desktop floating card
- `apps/web/src/components/scanner/ScanProgressSheet.tsx` — mobile bottom sheet
- `apps/web/src/components/scanner/ScanProgress.tsx` — responsive wrapper
- `apps/web/src/hooks/useScanProgress.ts` — SSE consumption hook

### Backend (already exists, no changes needed)
- `apps/api/internal/sse/hub.go` — SSE hub
- `apps/api/internal/handlers/scanner_handler.go` — GET /status, POST /cancel
- SSE endpoint: GET /api/v1/events

### Design References
- Desktop progress: `_bmad-output/screenshots/flow-h-scanner-desktop/h2-scan-progress-desktop.png`
- Desktop toast: `_bmad-output/screenshots/flow-h-scanner-desktop/h3-scan-complete-toast-desktop.png`
- Mobile progress: `_bmad-output/screenshots/flow-h-scanner-mobile/h5-scan-progress-mobile.png`
- Mobile toast: `_bmad-output/screenshots/flow-h-scanner-mobile/h6-scan-complete-toast-mobile.png`
- Filtered library: `_bmad-output/screenshots/flow-h-scanner-desktop/h7-filtered-library-unmatched-desktop.png`
- Full spec: `_bmad-output/planning-artifacts/scanner-ui-design-brief.md`
- Story spec: `_bmad-output/implementation-artifacts/7-4-scan-progress-tracking.md`

## Test Suite State
- 118 test files, 1459 tests, all passing
- No orphaned test processes
- Build clean (no type errors)
