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

- [x] Task 1: Add the 探索 tab to the nav config (AC: #1, #2, #5)
  - [x] 1.1 In `apps/web/src/components/shell/TabNavigation.tsx`, add `{ label: '探索', to: '/discover', matchPaths: ['/discover'] }` to the `TABS` array.
  - [x] 1.2 Place it as the **second** entry — immediately after `媒體庫` and before `下載中` (browse/discover grouping). Do not reorder or alter the other four entries.
  - [x] 1.3 No new component/markup needed — the existing `TABS.map(...)` render loop, `Link`, active-state logic, and `data-testid={`tab-${tab.label}`}` automatically produce a `tab-探索` element. Confirm `to="/discover"` is type-accepted by TanStack Router's typed `Link` (route already registered in `routeTree.gen.ts`).

- [x] Task 2: Tests (AC: #1, #3, #4, #5)
  - [x] 2.1 In `apps/web/src/components/shell/TabNavigation.spec.tsx`, register a `/discover` route in `createTestRouter` (add a `discoverRoute` child alongside the existing `libraryRoute`/`downloadsRoute`/etc. and include it in `addChildren`).
  - [x] 2.2 Update the "renders all … navigation tabs" test to also assert `tab-探索` is in the document (and update the test title/count wording from "four" → "five").
  - [x] 2.3 Add a test: navigating to `/discover` makes `tab-探索` active (`toHaveClass('text-white')` + `toHaveClass('border-blue-400')`) and a non-active tab (e.g. `tab-媒體庫`) inactive — mirror the existing `[P1] pending route` test. (Also added a query-param variant `/discover?genre=16` and an `href="/discover"` assertion.)
  - [x] 2.4 Extend the `[P2] shows no active tab on non-tab route` test to also assert `tab-探索` has `text-[var(--text-muted)]` on `/`.

- [x] Task 3: Verify no AppShell regression (AC: #5)
  - [x] 3.1 Check `apps/web/src/components/shell/AppShell.spec.tsx` still passes unchanged (it asserts the header icon-bar: logo/search/settings — this story does NOT touch the AppShell header, only the `TabNavigation` tab row rendered below it). ✅ AppShell.spec.tsx 12/12 green.

- [~] Task 4 (DISCOVERED — Rule 24 ①, absorbed in this story per Alexyu 2026-06-04): Re-bless the `shell-tab-navigation` visual-regression baseline
  - [x] 4.1 Regenerated `-darwin` baselines via `pnpm run test:visual:update`: `shell-tab-navigation/{default,hover,focus}-visual-darwin.png`. Visually confirmed the new default baseline shows 媒體庫·**探索**·下載中·待解析·設定 (媒體庫 active). The update ALSO re-emitted `media-media-detail-panel/{default,focus}-visual-darwin.png` (unrelated surprise) → REVERTED (`git checkout`), NOT blessed; filed as `disc-flaky-visual-media-detail-panel` backlog (Rule 24 ③). Only the 3 `shell-tab-navigation` darwin PNGs remain changed.
  - [x] 4.1b Audit-doc line appended in `_bmad-output/audit/visual-baseline-19-4.md` (`shell-tab-navigation` row → "darwin re-blessed 2026-06-04 … `-linux` pending CI").
  - [x] 4.2 **UX (Sally) gallery review** — ✅ APPROVED 2026-06-04 (Sally / ux-designer). Reviewed all 3 darwin states (default/hover/focus) against `Component/TabActive (TboA7)` + `TabInactive (j98G4)`: 探索 integrates faithfully — consistent font/weight/color tokens/gap, correct active(白字+藍底線)/inactive(muted) treatments, 2nd-position browse-IA grouping, no layout drift. Recorded on PR #28; `requires-manual-review` label removed.
  - [ ] 4.3 `-linux` re-bless via CI (delete-stale `-linux` → incremental bootstrap path 2, or the visual-regression.yml rebless flow) after the darwin/Sally commit lands.

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

claude-opus-4-8[1m] (Amelia / BMM dev-story workflow)

### Debug Log References

- `npx vitest run src/components/shell/TabNavigation.spec.tsx` — RED first (5 new tests fail on missing `tab-探索`), then GREEN after the `TABS` edit.
- `npx vitest run TabNavigation.spec.tsx AppShell.spec.tsx` — 23/23 pass (TabNavigation 11 + AppShell 12).
- `pnpm nx test web --skip-nx-cache` — 1961/1961 pass (161 files); test-cleanup reported all processes reaped.
- `pnpm exec eslint` + `pnpm exec prettier --check` on both touched files — clean (Rule 21 `local/implements-pen-node-id` satisfied by the unchanged file header — no new `.pen` node).
- `pnpm nx test api` (Go full-regression gate, Epic 9 Retro AI-1) — PASS (exit 0, all packages); zero Go files touched.
- `pnpm run test:visual:update` — regenerated darwin baselines; the single looping visual test passed (1.1m). Auto-started Go API + Vite web; orphaned-process cleanup ran.

### Completion Notes List

**AC Drift / Contract checks (Step 2):**

- 🔗 AC Drift: NONE (checked: `TabNavigation`/`matchPaths`/`探索` across `_bmad-output/implementation-artifacts/*.md`. Hits are the nav-shell origin (5-0), the visual-baseline stories (19-3/19-4/19-4b/19-8), and 10-3 "custom explore blocks" — none specify a prior AC contract that adding a 5th tab changes. The 4 existing tabs' labels/order/active-matching are untouched; the new tab is purely additive.)
- 📎 Contract Stamps: NONE (no `[@contract-v*]` stamps in this story; it defines/consumes no wire contract — pure FE nav config.)

**Architecture / key decisions:**

- One data-driven entry added to `TABS` in `TabNavigation.tsx` (2nd position, after 媒體庫). Render loop, `Link`, active styling, and `data-testid` are unchanged — the `tab-探索` element and its active behavior fall out of the existing map. `matchPaths: ['/discover']` lights up on `/discover` and `/discover?…` (query string isn't part of `pathname`). Shared desktop+mobile nav (no separate mobile component) → one edit covers both.
- Reuse / Rule 21: reuses `Component/TabActive (TboA7)` + `Component/TabInactive (j98G4)` — no new `.pen` node, no `ux-design.pen` edit, so the CLAUDE.md screenshot workflow does not apply.

**🎨 UX Verification (dev Step 9):** the 探索 entry reuses `Component/TabActive (TboA7)` + `Component/TabInactive (j98G4)` (no new `.pen` node). Regenerated darwin baseline visually verified: nav reads 媒體庫·探索·下載中·待解析·設定, active-tab styling intact.

**🎨 UX (Sally) sign-off — APPROVED 2026-06-04:** reviewed the 3 regenerated darwin gallery renders (default/hover/focus). 探索 is faithful to TabActive/TabInactive design intent — identical typography/color-token/spacing to the 4 sibling tabs, correct active (white + blue underline) and inactive (muted) states, no layout break or overflow, "探索" is the correct zh-TW Discover term, 2nd-position placement matches browse-IA grouping. No drift. "First-Sally-approved web rendering" per `tests/visual/README.md` point 4. Approved on PR #28; `requires-manual-review` removed → merge unblocked. `-linux` rebless still delegated to CI (Task 4.3).

**🔍 Discovery Triage (Rule 24):**

- **① expand-scope-in-place — Visual-regression baseline rebless (ABSORBED, Alexyu 2026-06-04):** the 探索 tab changes the `shell-tab-navigation` render → the 6 committed baselines pixel-diff and the visual CI gate (`.github/workflows/visual-regression.yml`, watches `apps/web/src/components/**`) fails until re-blessed. Darwin re-blessed locally (Task 4.1) + audit line added; **Sally review + separate rebless commit + `-linux` via CI still pending** (Task 4.2/4.3). NOT silently committed — Sally gate respected.
- **③ backlog-with-carry-forward-link — `media-media-detail-panel` flaky visual baseline:** `test:visual:update` re-emitted that component's default+focus darwin PNGs with NO code change to it. NOT Rule 23 (fixture `createdAt` is a fixed ISO; `MediaDetailPanel.tsx:253` `new Date(createdAt).toLocaleDateString` is wall-clock-independent). Likely async image-load flakiness (bugfix-10-4 class). Out of scope + non-blocking for this story; baseline reverted (not blessed). Tracked: `disc-flaky-visual-media-detail-panel` (sprint-status.yaml) — filed at discovery time, names this story.

### File List

- `apps/web/src/components/shell/TabNavigation.tsx` (modified — added the 探索 `/discover` entry to `TABS`, 2nd position)
- `apps/web/src/components/shell/TabNavigation.spec.tsx` (modified — registered `/discover` test route; "four"→"five" tab assertion; added active / query-param-active / href / inactive-on-`/` assertions for `tab-探索`)
- `_bmad-output/implementation-artifacts/sprint-status.yaml` (modified — `disc-nav-entry-discover-route` ready-for-dev → in-progress + progress note; added `disc-flaky-visual-media-detail-panel` backlog)
- `tests/visual/components.visual.spec.ts-snapshots/components/shell-tab-navigation/default-visual-darwin.png` (rebaselined — 探索 tab)
- `tests/visual/components.visual.spec.ts-snapshots/components/shell-tab-navigation/hover-visual-darwin.png` (rebaselined — 探索 tab)
- `tests/visual/components.visual.spec.ts-snapshots/components/shell-tab-navigation/focus-visual-darwin.png` (rebaselined — 探索 tab)
- `_bmad-output/audit/visual-baseline-19-4.md` (modified — `shell-tab-navigation` darwin re-bless line, 2026-06-04)

_Pending (Task 4.3):_ `shell-tab-navigation/{default,hover,focus}-visual-linux.png` re-bless via CI.

> **⚠️ Commit discipline (`tests/visual/README.md` point 3):** the 3 `shell-tab-navigation` darwin PNGs + audit-doc line are a **separate `test(visual): rebaseline …` commit**, NOT mixed with the logic change (TabNavigation + spec). And that commit happens only **after** the Sally gallery review.

### Change Log

| Date       | Change                                                                                                                  |
| ---------- | ----------------------------------------------------------------------------------------------------------------------- |
| 2026-06-04 | Task 1: added 探索 (`/discover`) entry to `TabNavigation` `TABS` (2nd position) — top-nav entry for the existing discover route (AC #1, #2, #5) |
| 2026-06-04 | Task 2: `TabNavigation.spec.tsx` — `/discover` test route + 4 new/updated assertions (render, active, query-param active, inactive-on-`/`, href) (AC #1, #3, #4, #5) |
| 2026-06-04 | Task 3: verified `AppShell.spec.tsx` 12/12 green — no header regression (AC #5)                                          |
| 2026-06-04 | Discovery (Rule 24 ①): 探索 tab changes the `shell-tab-navigation` visual baseline → Task 4 filed |
| 2026-06-04 | Task 4.1 (Rule 24 ① absorbed): regenerated `shell-tab-navigation` darwin baselines (3 PNGs) + audit-doc line. Reverted unrelated `media-media-detail-panel` darwin re-emit (flaky, not blessed) → filed `disc-flaky-visual-media-detail-panel` backlog (Rule 24 ③). Sally review + separate rebless commit + `-linux` CI rebless still pending (Task 4.2/4.3) |
| 2026-06-04 | Go full-regression gate (`pnpm nx test api`) PASS — no Go changes, no regressions |
| 2026-06-04 | Task 4.2: UX (Sally) gallery review APPROVED — darwin baselines blessed; `requires-manual-review` removed on PR #28 (merge unblocked); `-linux` rebless still via CI (Task 4.3) |
