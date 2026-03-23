# Story 7.4: Scan Progress Tracking

Status: ready-for-dev

## Story

As a **NAS media collector**,
I want **to see real-time scan progress with file counts and current file being processed**,
so that **I know the scan is working and how long it will take**.

## Acceptance Criteria

1. **Given** a scan is in progress, **When** the user is on any page, **Then** a floating progress card appears in the bottom-right corner (desktop) showing: progress bar, files found count, current file name, and estimated time
2. **Given** the progress card is visible, **When** the user clicks minimize, **Then** it collapses to a small pill showing only the progress percentage with a spinning loader icon
3. **Given** a scan completes, **When** results are ready, **Then** the progress card shows a completion summary (found, matched, unmatched, errors) and auto-dismisses after 10 seconds
4. **Given** a scan is in progress, **When** the user clicks Cancel on the progress card, **Then** POST /api/v1/scanner/cancel is called and the scan stops
5. **Given** the user is on mobile, **When** a scan is in progress, **Then** a bottom sheet with peek state shows progress bar and file count, expandable for details
6. **Given** SSE connection drops, **When** the client reconnects, **Then** the progress card resumes showing current state (via GET /api/v1/scanner/status fallback)

## Tasks / Subtasks

- [ ] Task 1: Create ScanProgressCard component (desktop) (AC: 1, 2, 3)
  - [ ] 1.1: Create `apps/web/src/components/scanner/ScanProgressCard.tsx` with floating card layout (fixed bottom-right, z-50, dark theme, 400px width)
  - [ ] 1.2: Implement progress bar (percentage), stats row (found/parsed/matched/errors with Lucide icons), current file (truncated, monospace), and ETA display
  - [ ] 1.3: Implement minimize/expand toggle — minimize collapses to small pill showing "掃描中 62%" with Lucide Loader spinning icon
  - [ ] 1.4: Implement completion state — show summary (found, matched, unmatched, errors) with action links ("查看未比對項目", "查看錯誤")
  - [ ] 1.5: Implement auto-dismiss on completion (setTimeout 10s, clearTimeout if user interacts, thin shrinking progress indicator at bottom)
  - [ ] 1.6: Write component tests (≥70% coverage)

- [ ] Task 2: Create useScanProgress custom hook (AC: 1, 6)
  - [ ] 2.1: Create `apps/web/src/hooks/useScanProgress.ts` custom hook
  - [ ] 2.2: Connect to EventSource at /api/v1/events, filter for `scan_progress` event type
  - [ ] 2.3: Maintain local state via useReducer: isScanning, progress (percentDone), currentFile, filesFound, filesParsed, filesMatched, errorCount, estimatedTime
  - [ ] 2.4: Implement reconnection fallback: on SSE error/timeout (5s), poll GET /api/v1/scanner/status until SSE reconnects
  - [ ] 2.5: Expose scanStatus, cancelScan, and isMinimized state from the hook
  - [ ] 2.6: Write hook tests (≥70% coverage)

- [ ] Task 3: Implement cancel scan flow (AC: 4)
  - [ ] 3.1: Add Cancel button ("取消掃描") to progress card (ghost button, --text-secondary)
  - [ ] 3.2: Implement cancel confirmation dialog: "確定要取消掃描嗎？已處理的結果會保留。" with [繼續掃描] + [取消掃描] (--error)
  - [ ] 3.3: Call POST /api/v1/scanner/cancel on confirm, show "取消中..." state until confirmed via SSE
  - [ ] 3.4: Write tests for cancel flow (confirmation, API call, state transitions)

- [ ] Task 4: Create mobile bottom sheet variant (AC: 5)
  - [ ] 4.1: Create `apps/web/src/components/scanner/ScanProgressSheet.tsx` with bottom sheet pattern
  - [ ] 4.2: Implement peek state (64px height, full width, bottom edge): spinning Lucide Loader + percentage + file count
  - [ ] 4.3: Implement expanded state (half-screen): progress bar, stats (two rows for narrow viewport), ETA, cancel button — no current file name (save space)
  - [ ] 4.4: Implement drag handle and swipe-down-to-collapse gesture
  - [ ] 4.5: Write component tests (≥70% coverage)

- [ ] Task 5: Responsive wrapper and app integration (AC: all)
  - [ ] 5.1: Create `apps/web/src/components/scanner/ScanProgress.tsx` responsive wrapper — renders ScanProgressCard on ≥768px, ScanProgressSheet on <768px
  - [ ] 5.2: Add ScanProgress to app root layout (visible on all pages during active scan)
  - [ ] 5.3: Handle stacking with AI Parse Progress Card if both active — 12px vertical gap
  - [ ] 5.4: Run frontend tests (`npx nx test web`)
  - [ ] 5.5: Manual verification: trigger scan, watch progress, test cancel, test completion auto-dismiss
  - [ ] 5.6: UX verification against scanner-ui-design-brief.md (screens H2, H3, H4)

## Dev Notes

### Design Reference
- **Screen H2:** Scan Progress (Desktop) — floating card, bottom-right, 400px, --bg-secondary, --shadow-lg, radius-xl
  - Screenshot: `_bmad-output/screenshots/flow-h-scanner-desktop/h2-scan-progress-desktop.png`
- **Screen H3:** Scan Results Summary — toast notification, top-center, 480px max-width, auto-dismiss 10s
  - Screenshot: `_bmad-output/screenshots/flow-h-scanner-desktop/h3-scan-complete-toast-desktop.png`
- **Screen H5:** Scan Progress (Mobile) — bottom sheet, peek height 64px, expandable on tap
  - Screenshot: `_bmad-output/screenshots/flow-h-scanner-mobile/h5-scan-progress-mobile.png`
- **Screen H6:** Scan Complete Toast (Mobile)
  - Screenshot: `_bmad-output/screenshots/flow-h-scanner-mobile/h6-scan-complete-toast-mobile.png`
- **Screen H7/H8:** Filtered library grid (unmatched) — destination for "查看未比對項目" action link
  - Desktop: `_bmad-output/screenshots/flow-h-scanner-desktop/h7-filtered-library-unmatched-desktop.png`
  - Mobile: `_bmad-output/screenshots/flow-h-scanner-mobile/h8-filtered-library-unmatched-mobile.png`
- Full specs: `_bmad-output/planning-artifacts/scanner-ui-design-brief.md`

### Backend Endpoints (all exist from Story 7-1, no backend changes needed)
- **SSE stream:** GET /api/v1/events — EventSource connection, filter for `scan_progress` event type
- **Status fallback:** GET /api/v1/scanner/status — returns current scan progress or last result
- **Cancel:** POST /api/v1/scanner/cancel — cancel active scan

### SSE Event Shape
```typescript
// Event type: "scan_progress"
interface ScanProgressEvent {
  filesFound: number;
  currentFile: string;
  percentDone: number;
  errorCount: number;
  estimatedTime: string; // e.g., "1 分 42 秒"
}
```

### SSE Hub Implementation
- Hub location: `apps/api/internal/sse/hub.go`
- Event type constant: `EventScanProgress = "scan_progress"`
- Client channels buffered at 100, broadcast channel buffered at 256
- Browser EventSource auto-reconnects, but add manual fallback after 5s timeout

### Component & Hook Locations
- `apps/web/src/components/scanner/ScanProgressCard.tsx` — desktop floating card
- `apps/web/src/components/scanner/ScanProgressSheet.tsx` — mobile bottom sheet
- `apps/web/src/components/scanner/ScanProgress.tsx` — responsive wrapper
- `apps/web/src/hooks/useScanProgress.ts` — SSE consumption hook

### UI Implementation Rules
- **State management:** useState/useReducer for SSE-driven state (NOT TanStack Query — SSE is push-based)
- **Floating card positioning:** fixed bottom-right, z-index 50 (above content, below modals)
- **Stacking:** If AI Parse Progress Card is also visible, stack vertically with 12px gap
- **Mobile breakpoint:** <768px uses bottom sheet, ≥768px uses floating card
- **Auto-dismiss:** setTimeout 10s after completion, clearTimeout if user interacts
- **Icons:** Lucide only — Loader (spinning), File, FileCheck, Link, AlertTriangle, CheckCircle, XCircle, Minus, X
- **Styling:** Tailwind CSS utility classes, dark theme colors (--bg-primary, --bg-secondary, --text-primary, --text-muted, --accent-primary)
- **Font:** Monospace (JetBrains Mono) for file paths and numeric counters
- **Copy language:** Traditional Chinese (zh-TW) — see scanner-ui-design-brief.md Section 3 for all copy strings

### Scope Boundaries — DO NOT Implement
- ❌ Backend endpoints (all exist from Story 7-1)
- ❌ Settings page scan trigger (Story 7-3)
- ❌ Scheduled scan service (Story 7-2)
- ❌ Filename parsing or TMDB matching (separate epics)
- ❌ Scan history or log persistence

### References

- [Source: scanner-ui-design-brief.md] — Screens H2, H3, H4 with full element specifications
- [Source: project-context.md] — Mandatory rules (Tailwind, TanStack Query for server state, naming conventions)
- [Source: architecture/core-architectural-decisions.md#Decision-8] — SSE Hub architecture
- [Source: epic-7-media-library-scanner.md] — Epic scope, success criteria
- [Source: 7-1-recursive-folder-scanner.md] — SSE event broadcasting, scanner endpoints

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

### File List
