# Story UX2-1: FOUNDATION — v2 Navigation Shell behind `new_shell_enabled`

Status: ready-for-dev

> UX Redesign **Phase 2** pilot · Story 1 of 3 (FOUNDATION — **blocking**, must land before UX2-2 / UX2-3).
> Plans against: `_bmad-output/planning-artifacts/ux-redesign/01-nav-ia-decision-adr.md` (D1–D4, patterns) + `01-design-language-v2.md` (§5.3, §6 shell) + `ux-design.pen` `Navigation Shell v2` (node `CLo58`).

## Story

As the solo power user of Vido,
I want the v2 navigation shell (collapsible sidebar on desktop, bottom-tab + More on mobile) to live behind a runtime feature flag,
so that the app can switch chassis for the Browse + Detail pilot **without disturbing any existing screen**, and roll back instantly if the pilot fails its go/no-go gate.

## Acceptance Criteria

1. **Runtime flag (F4 single chokepoint).** A runtime settings flag `new_shell_enabled` (row in the existing key/value settings table — **NO new table**) gates the shell. Read in **exactly one place** — `apps/web/src/routes/__root.tsx` — selecting new `AppShell` vs the legacy shell. No route/component branches on it. Default **OFF**. Backend read via the existing `SettingsRepository.GetBool` (`apps/api/internal/repository/settings_repository.go:214`); exposed through the existing settings handler; frontend reads it via a TanStack Query settings hook (Rule 5).
2. **Flag ON → v2 shell.** Desktop renders the collapsible **AppSidebar** (240px expanded ↔ 64px icon-rail, user-toggled, state persisted); mobile renders **MobileTabBar** (bottom-4 `首頁 · 媒體庫 · 活動 · 下載` + `更多`) opening **MobileMoreSheet**. **Flag OFF → legacy** `AppShell`/`TabNavigation` render pixel-unchanged.
3. **New shell components** under `apps/web/src/components/shell/`, implementing the `.pen` `Navigation Shell v2` (`CLo58`) and its reusable components: `AppSidebar`, `SidebarNav`, `SidebarGroup`, `SidebarGroupLabel`, `SidebarNavItem`, `SidebarGroupParent`, `SidebarFooter` (ambient status strip), `MobileTabBar`, `MobileMoreSheet`, `LegacyContentContainer`. Each carries a Rule 21 `// Implements:` header → the matching `.pen` node id (see Design Refs).
4. **Destinations earn slots (pilot scope).** Sidebar `內容` group: `首頁`(`/`) · `媒體庫`▾(`/library` ▸ `電影` `/library/movies` · `影集` `/library/tv`) · `探索`(`/discover`). `任務` group: `下載`(`/downloads`) · `設定`(`/settings`). **`活動`(`/activity`) and `系統`(`/system`) are deferred** (their routes do not exist yet — no slot for an unbuilt route; they enter in Phase 3). Active state via TanStack Router matching (`Link` activeProps / `useRouterState`) — **NOT** hand-rolled `startsWith`; a child route (`/library/movies`) marks **both** its `SidebarNavItem` (leaf) and its `媒體庫` `SidebarGroupParent` (active-ancestor, subtler treatment) active.
5. **Legacy fidelity (Murat acceptance).** `LegacyContentContainer` reproduces today's centered `max-w-7xl` canvas. Every **non-migrated** route (`/`, `/discover`, `/downloads`, `/settings/*`, `/pending`, `/search`, …) renders **pixel-unchanged** inside the new shell — the existing committed visual baselines for those routes MUST still PASS under flag-ON (they are the acceptance test for the container, not a re-shoot). Routes opt into the new layout via an explicit route-level marker; default = legacy container.
6. **Global-overlay ownership.** `ScanProgress` (today mounted **outside** `AppShell` in `__root.tsx:42`) moves explicitly **into** the new shell so it is owned, not an orphaned corner.
7. **Base UI primitives (D1-d, first runtime UI dep).** Add `@base-ui-components/react`. Wrap **once per primitive** in `apps/web/src/components/ui/` (`Tooltip` — required by the 64px rail — `Popover`, `Sheet`, `Dialog`). Wrappers apply **token classes only**, zero hardcoded values. An ESLint `no-restricted-imports` rule (F2) **bans** importing `@base-ui-components/react` anywhere except `apps/web/src/components/ui/`.
8. **Tokens to code (the deferred sweep, behind the visual-regression net).** Add the 12 v2 R3 semantic tokens to `apps/web/src/styles.css` (`accent-subtle`, `accent-tint`, `accent-text`, `success-tint`, `error-tint`, `error-text`, `warning-tint`, `info-tint`, `text-on-accent`, `text-disabled`, `overlay-scrim`, `focus-ring` — values per `01-design-language-v2.md` §2.4) **and** reassign `--text-muted` `#808080 → #A0AABE` (R5 fix, §2.2). This re-renders shared components → a **deliberate visual-regression baseline re-shoot** is expected (chore-visual `-linux` bootstrap PR per Rule 22/23). Status→token map (§2.5) available for UX2-2/UX2-3.
9. **Ambient status strip (pilot-degraded).** `SidebarFooter` renders disk headroom · active-scan · queue-count · service-health dots, sourced from the **existing** service-health/status endpoints; any unavailable section degrades to empty/stale (F3 fail-soft), never errors. A dedicated aggregate `/api/v1/activity` + status-summary endpoint is **Phase-3** (the strip wires to what already exists for the pilot).
10. **Header search (pilot scope).** The header keeps the existing `InstantSearchBar` behavior, restyled to v2 (`SearchInput`, 44px, `focus-ring`, Noto placeholder). **Full omnisearch consolidation (D4-5, retiring `/search`) is Phase-3** — not in this story.
11. **testid cutover (D1-c, phased).** New sidebar items expose `nav-{key}` (`nav-home`, `nav-library`, `nav-movies`, …). Legacy `tab-{label}` testids stay during the dual-shell window; `tab-{label}` is removed only at flag retirement (separate story), not here. Tests target whichever shell the flag selects.
12. **a11y baseline.** 44px touch targets on every interactive shell element (rail icons via hit-frame padding); 2px `focus-ring` on `:focus-visible`; rail icons each carry a Base UI `Tooltip` (label + count); status dots + nav items carry `aria-label`; `prefers-reduced-motion` disables the status-strip pulse. `jsx-a11y` passes.

## Tasks / Subtasks

### Backend (minimal — flag only)
- [ ] **T1: `new_shell_enabled` flag** (AC #1)
  - [ ] 1.1 Seed/read `new_shell_enabled` (bool, default false) via existing settings table + `SettingsRepository.GetBool`; expose through the existing settings handler/endpoint (no new table, no new error prefix). Verify the GET settings response includes it (Rule 18 case-transform at boundary).
  - [ ] 1.2 If a settings write path is needed to toggle it from the UI, reuse the existing settings update handler; otherwise toggle via DB/seed for the pilot (document which).

### Frontend — shell chassis
- [ ] **T2: Base UI wrappers + ESLint ban** (AC #7) — add dep; `components/ui/{Tooltip,Popover,Sheet,Dialog}.tsx` (token classes only); `no-restricted-imports` rule in `eslint.config.mjs` scoped to ban `@base-ui-components/react` outside `components/ui/`; unit-test the ban fires.
- [ ] **T3: Tokens to `styles.css`** (AC #8) — add 12 R3 tokens; reassign `--text-muted`; run `pnpm lint:all`; expect + handle the visual-regression baseline re-shoot (do NOT commit `-linux` locally — let the chore-visual workflow bootstrap; see CLAUDE.md).
- [ ] **T4: Sidebar components** (AC #3, #4, #12) — `SidebarNavItem` (`W5KQr`), `SidebarGroupLabel` (`v5Io8`), `SidebarGroupParent` (`imFBW`), `SidebarFooter` (`PrmQG`), `SidebarNav`/`SidebarGroup`, `AppSidebar` (expanded `b7CqJ0` + 64px rail `H7eXAK`), collapse toggle + persisted state. Active matching via TanStack Router (AC #4).
- [ ] **T5: Mobile shell** (AC #2, #12) — `MobileTabBar` (`u91vZI`, `MobileTabItem` `S86VM`) + `MobileMoreSheet` (`mfDKV`, reuse Epic-11 bottom-sheet via Base UI `Sheet`).
- [ ] **T6: `LegacyContentContainer` + route opt-in** (AC #5) — reproduce `max-w-7xl`; default-legacy, explicit opt-in marker for migrated routes.
- [ ] **T7: `AppShell` flag swap in `__root.tsx`** (AC #1, #2, #6) — single flag read; select new vs legacy shell; move `ScanProgress` ownership into the new shell; setup page still bypasses the shell.
- [ ] **T8: testids + a11y + specs** (AC #11, #12) — `nav-{key}` testids; jsx-a11y; component specs (Rule 9 co-located, Rule 16 matchers); legacy-fidelity baselines still pass (AC #5).

## Dev Notes

### Architecture Compliance
- **F4 single chokepoint / D1-c flag:** flag read ONLY in `__root.tsx`; runtime settings flag (not env). Retirement = delete one read site + legacy shell + `LegacyContentContainer`.
- **F2 wrap-once (N6):** Base UI imported only inside `components/ui/`; ESLint-enforced.
- **Rule 4/7/18:** flag read rides existing settings handler→service→repo; no new prefix; case-transform at boundary.
- **Rule 5:** flag + settings via TanStack Query; shell/collapse state is local.
- **Rule 21:** every new shell component headers its `.pen` node id (Design Refs).
- **Rule 22/23 (visual-regression):** the §8 token sweep is the one deliberate baseline re-shoot; `-linux` via chore-visual bootstrap, never committed locally (darwin machine).
- **Strangler discipline (P3):** this story MUST NOT modify legacy route CONTENT — only the chassis. Browse/Detail content changes live in UX2-2/UX2-3.

### Cross-Stack Split Check (Agreement 5)
Backend tasks: **1** (flag). Frontend tasks: **7**. Backend ≤3 → **NO split**; single foundation story (frontend-heavy, by design — the backend is just the flag).

### Tailwind note (project-context.md drift)
Repo is **Tailwind v4** (`tailwindcss@^4.1.18`, `@tailwindcss/postcss`) — `project-context.md` §"Core Architectural Decisions 1" still says "v3.x" and is **stale**; implement against v4. (Flag for a project-context correction in retro; do not fix in this story.)

### Project Structure
- CREATE: `components/shell/{AppSidebar,SidebarNav,SidebarGroup,SidebarGroupLabel,SidebarNavItem,SidebarGroupParent,SidebarFooter,MobileTabBar,MobileMoreSheet,LegacyContentContainer}.tsx` (+ specs); `components/ui/{Tooltip,Popover,Sheet,Dialog}.tsx` (+ specs).
- MODIFY: `routes/__root.tsx` (flag swap + ScanProgress ownership); `styles.css` (R3 tokens + text-muted); `eslint.config.mjs` (no-restricted-imports); `apps/web/package.json` (+`@base-ui-components/react`); settings handler/hook (flag exposure).
- DELETE: none (legacy `AppShell`/`TabNavigation` retire at flag removal, not now).

### Design Refs (`.pen` node ids — Rule 21)
- Frame `Navigation Shell v2` = `CLo58`. Assembled: `Sidebar-Expanded` `b7CqJ0` · `Sidebar-Rail` `H7eXAK` · `MobileTabBar` `u91vZI` · `MobileMoreSheet` `mfDKV`.
- Components: `SidebarNavItem` `W5KQr` · `SidebarGroupParent` `imFBW` · `SidebarGroupLabel` `v5Io8` · `SidebarFooterStatus` `PrmQG` · `MobileTabItem` `S86VM`.
- Screenshots: `_bmad-output/screenshots/design-system/navigation-shell-v2.png`.

### References
- [Source: planning-artifacts/ux-redesign/01-nav-ia-decision-adr.md — D1-a/b/c/d, Routing/Shell/BaseUI/Flag patterns, Impact Assessment]
- [Source: planning-artifacts/ux-redesign/01-design-language-v2.md — §2.4 tokens, §2.2 text-muted, §5.3 shell components, §6 shell spec]
- [Source: project-context.md — Rules 4,5,7,18,21,22,23; N6]
- [Source: apps/web/src/routes/__root.tsx:39,42 — AppShell + ScanProgress mount]
- [Source: apps/api/internal/repository/settings_repository.go:214 — GetBool]

## Dev Agent Record
_(to be filled by dev-story)_
