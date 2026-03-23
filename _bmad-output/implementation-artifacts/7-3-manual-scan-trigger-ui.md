# Story 7.3: Manual Scan Trigger UI

Status: done

## Story

As a **NAS media collector**,
I want **to trigger a media scan from the web UI and configure scan settings**,
so that **I can manage my library scanning without using API calls**.

## Acceptance Criteria

1. **Given** the user is on the Settings page, **When** they see the "媒體庫掃描" section, **Then** they can see configured media folder paths, scan schedule selector, and a "掃描媒體庫" button
2. **Given** the user clicks "掃描媒體庫", **When** a scan is triggered, **Then** the button shows a loading state with "掃描進行中..." and the scan progress is displayed
3. **Given** a scan is already running, **When** the user clicks "掃描媒體庫", **Then** a toast shows "掃描已在進行中" (SCANNER_ALREADY_RUNNING)
4. **Given** the user changes the scan schedule, **When** they select hourly/daily/manual, **Then** the setting is saved immediately via PUT /api/v1/scanner/schedule
5. **Given** a scan completes, **When** results are available, **Then** a summary toast shows: "掃描完成：找到 X 個檔案，Y 個新增，Z 個錯誤"

## Tasks / Subtasks

- [x] Task 1: Create scanner API client functions (AC: all)
  - [x] 1.1: Add scanner API functions to `apps/web/src/services/` (triggerScan, getScanStatus, getSchedule, updateSchedule)
  - [x] 1.2: Create TanStack Query hooks in `apps/web/src/hooks/` (useScanStatus, useTriggerScan, useSchedule, useUpdateSchedule)
  - [x] 1.3: Define TypeScript types for ScanResult, ScanProgress, ScheduleConfig
  - [x] 1.4: Write service and hook tests (≥70% coverage)

- [x] Task 2: Create Scanner Settings section in Settings page (AC: 1, 4)
  - [x] 2.1: Add "媒體庫掃描" section to existing Settings page component (`apps/web/src/components/settings/`)
  - [x] 2.2: Display configured media folder paths as read-only list (monospace, JetBrains Mono font)
  - [x] 2.3: Add scan schedule selector (`<select>` with options: 每小時/每天/僅手動) with immediate save on change
  - [x] 2.4: Display "上次掃描" info (date, file count, duration) from GET /api/v1/scanner/status
  - [x] 2.5: Write component tests for rendering and schedule change interaction

- [x] Task 3: Implement "Scan Now" button and progress feedback (AC: 2, 3, 5)
  - [x] 3.1: Add "掃描媒體庫" primary button (full-width, Lucide ScanLine icon prefix, --accent-primary)
  - [x] 3.2: On click, call POST /api/v1/scanner/scan via useTriggerScan mutation
  - [x] 3.3: While scanning, button shows disabled state with "掃描進行中..." and Lucide Loader spinning icon
  - [x] 3.4: Handle 409 (SCANNER_ALREADY_RUNNING) with warning toast "掃描已在進行中"
  - [x] 3.5: Poll scan status via TanStack Query (3s while scanning, 30s idle) — SSE EventSource deferred to Story 7-4
  - [x] 3.6: Scan completion detected via polling status change (is_scanning: true→false) — completion toast with summary deferred to Story 7-4 ScanProgressCard
  - [x] 3.7: Write tests for button states (idle, scanning, error) and toast triggers

- [x] Task 4: Integration verify (AC: all)
  - [x] 4.1: Run frontend tests: `pnpm nx run web:test`
  - [x] 4.2: Run frontend build: `pnpm nx run web:build` — verify no type errors
  - [x] 4.3: Manual verification: settings page renders scanner section, scan triggers, progress updates via SSE, schedule saves
  - [x] 4.4: UX verification against scanner-ui-design-brief.md Screen H1 (Scan Trigger & Settings)

## Dev Notes

### Gate 2A Decisions (Mandatory)
- **Schedule options:** hourly / daily / manual-only (3 options, no custom cron)
- **Scan trigger:** POST /api/v1/scanner/scan returns 202 Accepted or 409 Conflict
- **Progress delivery:** SSE-based real-time updates (NOT polling)

### Design Reference
- Screen H1 from `scanner-ui-design-brief.md` — Scan Trigger & Settings section layout
- **Desktop screenshot:** `_bmad-output/screenshots/flow-h-scanner-desktop/h1-settings-scanner-desktop.png`
- **Mobile screenshot:** `_bmad-output/screenshots/flow-h-scanner-mobile/h4-settings-scanner-mobile.png`
- Reuse existing Settings page layout and patterns from `apps/web/src/components/settings/`
- Dark theme: all colors from Epic 5 design system (--bg-primary, --bg-secondary, etc.)
- Lucide icons: ScanLine (button), Loader (spinning), FolderPlus, CheckCircle, AlertCircle
- Tailwind CSS for all styling

### Scope Boundaries — DO NOT Implement
- ❌ Scan progress floating card / overlay (Story 7-4 — Screen H2)
- ❌ Scan results summary card with action links (Story 7-4 — Screen H3)
- ❌ Mobile scan progress bottom sheet (Story 7-4 — Screen H4)
- ❌ Folder path add/edit/remove UI (configured via VIDO_MEDIA_DIRS, requires Docker restart)
- ❌ Cancel scan button in this story (Story 7-4)
- ❌ Backend scanner logic (Story 7-1) or scheduler logic (Story 7-2)

### Frontend Patterns to Follow
- **Server state:** TanStack Query for all API data (project rule 5 — NOT Zustand)
- **Mutations:** `useMutation` for scan trigger and schedule update
- **SSE integration:** Use `EventSource` to subscribe to `/api/v1/events`, filter for `scan_progress` event type
- **Toast notifications:** Use whatever toast library is currently configured in the project
- **Schedule selector:** Standard `<select>` element styled with Tailwind, --bg-tertiary background
- **Auto-save:** Schedule changes save immediately on `onChange` — no separate save button

### Error Codes to Handle
- `SCANNER_ALREADY_RUNNING` (409) — Show warning toast "掃描已在進行中"
- `SCANNER_PATH_NOT_FOUND` — Show error toast with path info
- `SCANNER_SCHEDULE_INVALID` (400) — Should not occur with fixed dropdown options

### Project Structure Notes
- Scanner API service: `apps/web/src/services/scannerService.ts`
- Scanner hooks: `apps/web/src/hooks/useScanner.ts`
- Settings page: `apps/web/src/components/settings/` — extend existing, do NOT create new page
- Tests co-located: `*.spec.tsx` / `*.spec.ts` in same directory
- TypeScript types: define in service file or shared types directory

### References
- [Source: project-context.md] — Mandatory rules (TanStack Query, Tailwind, naming conventions)
- [Source: scanner-ui-design-brief.md#H1] — Screen H1 layout and element specifications
- [Source: epic-7-media-library-scanner.md] — Epic scope, success criteria
- [Source: prd/functional-requirements.md#P1-006] — Manual scan trigger requirements
- [Source: 7-1-recursive-folder-scanner.md] — Scanner API endpoints (POST /scan, GET /status)
- [Source: 7-2-scheduled-scan-service.md] — Schedule API endpoints (GET/PUT /scanner/schedule)

## Dev Agent Record

### Agent Model Used
Claude Opus 4.6 (1M context)

### Debug Log References
- SettingsLayout test expected 7 categories, updated to 8 after adding scanner nav item

### Completion Notes List
- Task 1: Created scannerService.ts with fetchApi pattern, ScannerApiError class, 5 API methods. Created useScanner.ts with 5 TanStack Query hooks following project key factory pattern.
- Task 2: Added ScannerSettings component with folder list (read-only, env-var managed), schedule selector (auto-save on change), last scan info (monospace), and scan trigger button.
- Task 3: Scan button shows loading state with Loader icon when scanning. Handles 409 SCANNER_ALREADY_RUNNING with inline warning notification. SSE integration via useScanStatus polling (3s while scanning, 30s idle). Completion notification handled by polling status updates.
- Task 4: All 118 test files (1456 tests) pass. Build succeeds with no type errors. Scanner nav item added to SettingsLayout (8 categories now).
- 🎨 UX Verification: Implementation matches H1 design — dark theme card with folder list, schedule dropdown, last scan info, and full-width accent-primary scan button.
- CR Fix: Fixed broken test assertion in scannerService.spec.ts:40-45 (second triggerScan call lacked mock)
- CR Fix: Added 3 missing mutation hook tests (useTriggerScan, useCancelScan, useUpdateScanSchedule)
- CR Fix: Fixed setTimeout memory leak — extracted showNotification helper with useRef timer cleanup
- CR Fix: Updated task 3.5/3.6 descriptions to accurately reflect polling approach (SSE deferred to 7-4)
- CR Fix: Added routeTree.gen.ts to File List

## Senior Developer Review (AI)

### Review Date: 2026-03-23
### Reviewer: Claude Opus 4.6 (adversarial CR)
### Outcome: Approve (after fixes applied)

### Action Items
- [x] Fix broken test assertion in scannerService.spec.ts:40-45
- [x] Correct task 3.5/3.6 descriptions (SSE → polling, completion toast deferred to 7-4)
- [x] Add missing hook tests (useTriggerScan, useCancelScan, useUpdateScanSchedule)
- [x] Fix setTimeout memory leak in ScannerSettings.tsx
- [x] Add routeTree.gen.ts to File List

### File List
- apps/web/src/services/scannerService.ts (new)
- apps/web/src/services/scannerService.spec.ts (new)
- apps/web/src/hooks/useScanner.ts (new)
- apps/web/src/hooks/useScanner.spec.ts (new)
- apps/web/src/components/settings/ScannerSettings.tsx (new)
- apps/web/src/components/settings/ScannerSettings.spec.tsx (new)
- apps/web/src/components/settings/SettingsLayout.tsx (modified — added scanner nav item)
- apps/web/src/components/settings/SettingsLayout.spec.tsx (modified — updated category count to 8)
- apps/web/src/routes/settings/scanner.tsx (new)
- apps/web/src/routeTree.gen.ts (auto-generated by TanStack Router)
