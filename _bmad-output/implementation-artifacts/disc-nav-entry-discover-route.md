# Story disc-nav-entry-discover-route: Discover Route Top-Nav Entry (探索)

Status: ready-for-dev

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

> **Origin:** Discovered by Story 11-2 (Rule 24 ③ — backlog-with-carry-forward-link). The `/discover`
> route shipped in 11-2 is fully functional and reachable by URL, but has **no entry in the global
> navigation shell** (the `TabNavigation` tab bar lists 媒體庫 / 下載中 / 待解析 / 設定 only). This story
> closes that carry-forward link by adding the 探索 (Discover) tab now that the nav IA decision is "add it
> to the existing tab row." Non-blocking, frontend-only.

## Story

As a Traditional Chinese NAS user,
I want a **探索** (Discover) entry in the top navigation bar,
so that I can reach the discover/browse page without typing the URL and can see when I'm currently on it.

## Acceptance Criteria

1. Given the app shell renders, when I look at the primary tab navigation, then a **探索** tab is visible and links to `/discover`.
2. Given I click the **探索** tab, when navigation completes, then I land on the `/discover` route (the existing discover page from Story 11-2).
3. Given I am on `/discover` (including with filter query params, e.g. `/discover?genre=16`), when the tab bar renders, then the **探索** tab shows the active styling (`border-blue-400 text-white`) and renders the screen-reader active indicator, exactly like the other tabs.
4. Given I am NOT on `/discover`, when the tab bar renders, then the **探索** tab shows the inactive styling (`text-[var(--text-muted)]`) and no other tab's active behavior is changed.
5. Given the existing tabs (媒體庫 / 下載中 / 待解析 / 設定), when 探索 is added, then their order, labels, active-matching, and styling remain unchanged (no regressions).

## Tasks / Subtasks

- [ ] Task 1: Add the 探索 tab to the nav config (AC: #1, #2, #5)
  - [ ] 1.1 In `apps/web/src/components/shell/TabNavigation.tsx`, add `{ label: '探索', to: '/discover', matchPaths: ['/discover'] }` to the `TABS` array.
  - [ ] 1.2 Place it as the **second** entry — immediately after `媒體庫` and before `下載中` (browse/discover grouping). Do not reorder or alter the other four entries.
  - [ ] 1.3 No new component/markup needed — the existing `TABS.map(...)` render loop, `Link`, active-state logic, and `data-testid={`tab-${tab.label}`}` automatically produce a `tab-探索` element. Confirm `to="/discover"` is type-accepted by TanStack Router's typed `Link` (route already registered in `routeTree.gen.ts`).

- [ ] Task 2: Tests (AC: #1, #3, #4, #5)
  - [ ] 2.1 In `apps/web/src/components/shell/TabNavigation.spec.tsx`, register a `/discover` route in `createTestRouter` (add a `discoverRoute` child alongside the existing `libraryRoute`/`downloadsRoute`/etc. and include it in `addChildren`).
  - [ ] 2.2 Update the "renders all … navigation tabs" test to also assert `tab-探索` is in the document (and update the test title/count wording from "four" → "five").
  - [ ] 2.3 Add a test: navigating to `/discover` makes `tab-探索` active (`toHaveClass('text-white')` + `toHaveClass('border-blue-400')`) and a non-active tab (e.g. `tab-媒體庫`) inactive — mirror the existing `[P1] pending route` test.
  - [ ] 2.4 Extend the `[P2] shows no active tab on non-tab route` test to also assert `tab-探索` has `text-[var(--text-muted)]` on `/`.

- [ ] Task 3: Verify no AppShell regression (AC: #5)
  - [ ] 3.1 Check `apps/web/src/components/shell/AppShell.spec.tsx` still passes unchanged (it asserts the header icon-bar: logo/search/settings — this story does NOT touch the AppShell header, only the `TabNavigation` tab row rendered below it).

## Dev Notes

### What this story is (and is NOT)

- **IS:** one line added to the `TABS` array in `TabNavigation.tsx` + spec updates. The `/discover` route, page, filters, and E2E already exist and are `done` (Story 11-2). This story ONLY wires the nav entry.
- **IS NOT:** any change to the discover page, filter engine, AppShell header icon-bar, routing, or backend. There are **zero backend tasks** — this is a single-file frontend change (Cross-Stack Split check: PASS, frontend-only).

### Architecture Compliance

- **Nav source of truth:** `TabNavigation.tsx` (`apps/web/src/components/shell/TabNavigation.tsx:12-17`) holds the `TABS: NavTab[]` config — `{ label, to, matchPaths }`. This is the ONLY place to add the entry; the render loop, `Link`, active styling, and `data-testid` are all data-driven off this array. Do **not** hand-roll a new `<Link>`.
- **Active matching:** active state is `tab.matchPaths.some((path) => currentPath.startsWith(path))` (`TabNavigation.tsx:30`). `matchPaths: ['/discover']` correctly lights up on `/discover` and any `/discover?...` (query string is not part of `pathname`) and any future `/discover/...` subpath. `startsWith('/discover')` does NOT collide with any existing route.
- **Shared desktop + mobile:** there is **no separate mobile nav component** — `TabNavigation` is the single nav for all viewports (horizontal-scroll tab bar, `overflow-x-auto`). Adding one `TABS` entry covers both desktop and mobile automatically. No bottom-nav/hamburger to also edit.
- **Reuse / Rule 21 (pen-node-id):** `TabNavigation.tsx` already carries the header `// Implements: Component/TabActive (TboA7) + Component/TabInactive (j98G4)`. The 探索 tab reuses those exact design nodes — **no new `.pen` node, no `ux-design.pen` edit**, so the CLAUDE.md screenshot-regeneration workflow does NOT apply. The `local/implements-pen-node-id` ESLint rule is already satisfied by the unchanged file header.
- **Icons:** the tab bar is **text-label only** (no icons per tab) — do NOT add a `Compass`/icon to the tab; that would diverge from the `TabActive`/`TabInactive` design. (The header icon-bar separately uses lucide `Search`/`Settings`, but that's the AppShell header, out of scope here.)

### Placement rationale (IA)

Order becomes: **媒體庫 · 探索 · 下載中 · 待解析 · 設定**. 探索 is a browse/discover surface, so it sits next to 媒體庫 (the other browse surface) rather than among the acquisition/ops tabs (下載中/待解析) or 設定. If Alexyu prefers a different slot, it's a one-line reorder.

### Project Structure Notes

- Touch only: `apps/web/src/components/shell/TabNavigation.tsx` (1 line) + `apps/web/src/components/shell/TabNavigation.spec.tsx` (test additions).
- Route already registered: `apps/web/src/routes/discover.tsx` → `createFileRoute('/discover')`; present in `apps/web/src/routeTree.gen.ts`. No routing changes.
- Naming/label convention: zh-TW labels matching existing tabs (媒體庫/下載中/待解析/設定). 探索 = "Explore/Discover".

### Time-dependent visual coverage

- **Does this story add/modify any `apps/web/src/components/**/*.{ts,tsx}` that reads `Date.now()` / `new Date()` / `Date.UTC()` / `Date.parse()`?**
  - **NO** — `N/A — no wall-clock-reading components touched.` `TabNavigation.tsx` reads only `useRouterState()` (current pathname); it has no clock dependency. (Rule 23 not applicable.)
- Reference: `project-context.md` Rule 23.

### References

- [Source: apps/web/src/components/shell/TabNavigation.tsx#L12-L17] — `TABS` config to extend
- [Source: apps/web/src/components/shell/TabNavigation.tsx#L29-L52] — data-driven render loop, active styling, `data-testid`
- [Source: apps/web/src/components/shell/TabNavigation.spec.tsx] — test harness pattern (`createTestRouter` + per-route children) to mirror for `/discover`
- [Source: apps/web/src/routes/discover.tsx] — existing `/discover` route (Story 11-2 deliverable)
- [Source: apps/web/src/components/shell/AppShell.tsx] — header icon-bar (logo/search/settings) — NOT modified by this story
- [Source: _bmad-output/implementation-artifacts/11-2-persistent-filter-chip-ui.md#Completion Notes — Discovery Triage] — origin of this carry-forward item
- [Source: _bmad-output/implementation-artifacts/sprint-status.yaml — `disc-nav-entry-discover-route`] — backlog tracking entry

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

### Discovery Triage

- **Did this story discover any work outside its current scope?**
  - To be completed by dev. Expected: `N/A — no out-of-scope work discovered` (this story IS the resolution of a prior Rule 24 ③ carry-forward; it should not itself spawn new out-of-scope work).
- Reference: `project-context.md` Rule 24.

### File List
</content>
</invoke>
