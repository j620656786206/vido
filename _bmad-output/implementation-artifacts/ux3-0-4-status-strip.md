# Story ux3-0-4 — Status strip goes live (frontend)

**Epic:** ux3-foundation (UX Redesign Phase 3) · **Status:** review (impl done, tests green)
**Owner:** Dev (Amelia) · **Type:** frontend · **FRs:** PH3-F2 (consumer) · **Depends on:** ux3-0-3 (merged #80)

## Story

As a user,
I want the sidebar-footer status strip to show real disk / scan / queue / health data,
So that the NAS pulse is always visible (D4-2), not a pilot placeholder.

## Acceptance Criteria

**Given** ux3-0-3's `/api/v1/status/summary`,
**When** `SidebarFooter` renders,
**Then** it consumes it via a TanStack Query hook (`useStatusSummary`, visibility-gated
poll, Rule 5/8) — replacing the pilot's health-only `useServiceStatuses` — showing the
real disk bar + `X.X / Y.Y TB`, `● 掃描中` (when active), `佇列 N` (when >0), and the
service-health dots from `serviceHealth.services`.

**Given** a section's status is not `"ok"` (or the query is loading/errored),
**When** the strip renders,
**Then** that section shows an empty/placeholder treatment (`—` disk, no scan/queue row,
muted dots) and **never throws** (ADR F3 frontend half).

**Given** the collapsed 64px rail,
**When** rendered,
**Then** it shows the health dots only (DL-v2 §6.4); the active-scan pulse uses
`motion-safe:animate-pulse` (respects `prefers-reduced-motion`).

**Given** the snake→camel boundary,
**When** the response is consumed,
**Then** `statusSummaryService` camelCases via `snakeToCamel` (Rule 18), mirroring
`serviceStatusService`.

## Tasks

1. [x] `services/statusSummaryService.ts` — `StatusSummary` types (camelCased sections) +
   `getSummary()` (fetch + snakeToCamel, mirrors serviceStatusService).
2. [x] `hooks/useStatusSummary.ts` — visibility-gated `useQuery` (30s poll while visible).
3. [x] `components/shell/SidebarFooter.tsx` — consume `useStatusSummary`; render disk bar
   (fill color accent→warning→error by usage), `掃描中`, `佇列 N`, dots from
   `serviceHealth`; per-section fail-soft; collapsed = dots only; reduced-motion pulse.
4. [x] Tests — new `SidebarFooter.spec.tsx` (all-ok render; per-section fail-soft; scan/
   queue hidden when idle; collapsed = dots; no-data no-throw). Updated the
   `useServiceStatus`→`useStatusSummary` mock in `AppSidebar.spec` / `AppShellV2.spec` /
   `MobileTabBar.spec` (they render the footer; the data hook is mocked, not provider-wrapped).

## Dev notes

- Verified: web unit suite 2171/2172 green (the 1 failure is the documented
  `preexisting-fail-instant-search-debounce-flake` — confirmed 8/8 green in isolation, not
  a regression from this story). `nx build web` green; `nx lint web` 0 errors.
- `useServiceStatus` is still used by the settings ServiceStatusDashboard — left intact.
- Disk display is decimal TB (NAS-vendor convention).
