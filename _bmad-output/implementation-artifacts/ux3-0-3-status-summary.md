# Story ux3-0-3 — `GET /api/v1/status/summary` aggregate endpoint (backend)

**Epic:** ux3-foundation (UX Redesign Phase 3) · **Status:** review (impl done, tests green)
**Owner:** Dev (Amelia) · **Type:** backend · **FRs:** PH3-F2 · **Feeds:** ux3-0-4 (status strip)

## Story

As a NAS owner,
I want one endpoint summarizing disk / active-scan / download-queue / service-health,
So that the v2 sidebar-footer status strip can show a real NAS pulse (D4-2).

## Acceptance Criteria

**Given** the four ambient concerns,
**When** `GET /api/v1/status/summary` is called,
**Then** it returns per-section status objects (`disk_headroom` / `active_scan` /
`download_queue` / `service_health`), each `{status:"ok"|"unavailable", …}` — snake_case
(the web `snakeToCamel` fetchApi boundary camelCases it, Rule 18).

**Given** a downstream source fails,
**When** the endpoint composes,
**Then** ONLY that section is `"unavailable"` (+ an `error` string) and the endpoint still
returns `success:true` — never a fail page (ADR B1/F3, N1/N4). Verified per section.

**Given** Rule 4 boundaries,
**When** implemented,
**Then** `StatusSummaryService` READS existing services via narrow interfaces
(`ServiceStatusService.GetAllStatuses`, `ScannerService.IsScanActive`/`GetProgress`,
`DownloadService.GetDownloadCounts`, `MediaLibraryService.GetAllLibraries`) — no
repository reach-through, no duplicated subsystem logic.

**Given** Epic 7b multi-library folders may span volumes,
**When** disk headroom is computed,
**Then** it `syscall.Statfs` each library path and **de-dups by device id** (subdirs on one
volume counted once); an unreadable path degrades only itself; zero readable volumes →
section `"unavailable"`. Partially delivers Epic 18 P4-020/P4-021.

## Tasks

1. [x] `services/status_summary_service.go` — composer + 4 narrow source interfaces + 4
   per-section builders (fail-soft) + `syscall.Statfs` disk helper with device-dedup.
2. [x] `handlers/status_summary_handler.go` — thin handler; `GET /status/summary` →
   `SuccessResponse(svc.GetSummary(ctx))` (service is fail-soft, so no error envelope).
3. [x] `cmd/api/main.go` — construct `NewStatusSummaryService(serviceStatusService,
   scannerService, downloadService, mediaLibraryService)` + register the route.
4. [x] Tests — `status_summary_service_test.go`: all-ok; health/download per-section
   fail-soft isolation; disk unreadable-path → unavailable; disk same-volume dedup → 1
   volume; nil sources degrade (no panic). Services+handlers pkgs green; `go build/vet
   ./...` clean; gofmt clean.

## Dev notes

- Response is snake_case (codebase convention); FE consumes camelCase via `snakeToCamel`
  at `fetchApi` (confirmed in `types/library.ts` + `utils/caseTransform.ts`).
- No new Rule-7 error prefix — section `"unavailable"` status is data, not an error code
  (ADR B1). Epic 18 MUST reuse this endpoint, not rebuild a parallel disk/health aggregate.
- ux3-0-4 (FE) wires `SidebarFooter` to this endpoint (currently pilot-degraded health-only).
