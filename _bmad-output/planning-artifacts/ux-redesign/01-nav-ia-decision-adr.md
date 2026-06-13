---
stepsCompleted: [1, 2, 3, 4, 5, 6, 7, 8]
lastStep: 8
status: 'complete'
completedAt: '2026-06-13'
inputDocuments:
  - _bmad-output/planning-artifacts/ux-redesign/00-redesign-brief.md
  - _bmad-output/planning-artifacts/ux-redesign/pen-review-2026-06-12.md
  - project-context.md
  - _bmad-output/design-context-pack.md
  - apps/web/src/routes/__root.tsx
  - apps/web/src/routes/index.tsx
  - apps/web/src/routes/library.tsx
  - apps/web/src/routes/discover.tsx
  - apps/web/src/routes/search.tsx
  - apps/web/src/routes/downloads.tsx
  - apps/web/src/routes/pending.tsx
  - apps/web/src/routes/media/$type.$id.tsx
  - apps/web/src/routes/settings.tsx
  - _bmad-output/planning-artifacts/architecture/core-architectural-decisions.md
  - _bmad-output/planning-artifacts/architecture/project-structure-boundaries.md
  - _bmad-output/planning-artifacts/prd/user-journeys.md
  - _bmad-output/planning-artifacts/prd/web-application-specific-requirements.md
  - _bmad-output/planning-artifacts/multi-library-ux-spec.md
workflowType: 'architecture'
scope: 'Navigation / IA decision ADR (UX redesign Phase 1a) — D1–D4 from 00-redesign-brief.md'
project_name: 'vido'
user_name: 'Alexyu'
date: '2026-06-12'
---

# Navigation & IA Decision ADR — Vido UX Redesign Phase 1a

_This document builds collaboratively through step-by-step discovery. Sections are appended as we work through each architectural decision together._

## Project Context Analysis

### Scope of This ADR

This is a focused ADR resolving the four open IA decisions (D1–D4) framed in
`00-redesign-brief.md` §6, plus their sub-decisions. It is NOT a full product
architecture — that exists at `planning-artifacts/architecture/`. Output feeds
Phase 1's design language work and Phase 2's Browse+Detail pilot.

### Current IA Ground Truth (verified in code, 2026-06-12)

**Shell:** `AppShell.tsx` — sticky header (logo → `/`, desktop InstantSearchBar,
mobile full-screen search overlay, settings gear → `/settings`) + `TabNavigation.tsx`
with 5 horizontal tabs: 媒體庫 `/library` · 探索 `/discover` · 下載中 `/downloads` ·
待解析 `/pending` · 設定 `/settings`. Max-width container `max-w-7xl`.

**Structural facts that sharpen the D-decisions:**

1. **The homepage is already two identities in one render tree** (`index.tsx`):
   Epic 10's discovery layer (HeroBanner + ExploreBlocksList) stacked with Epic 4's
   dashboard layer (DownloadPanel, RecentMediaPanel, QBStatusIndicator,
   ConnectionHistoryPanel, NewMediaNotifications). D3 is not "choose an identity"
   but "separate two identities that already cohabit".
2. **`/pending` is a static shell** — a hardcoded empty-state with no data wiring,
   yet holds a top-level nav slot. Demoting it (D4) has near-zero migration cost.
3. **Task surfaces are fragmented across ≥4 places:** `/downloads` (polling page),
   `/pending` (static stub), `ScanProgress` (root-level overlay outside AppShell),
   subtitle batch progress (inside dialogs), AI subtitle jobs (backend-only, no UI
   at all — P8). No single place answers "what is Vido doing right now?" (N1 gap).
4. **Settings mixes three concern classes across 11 children:** user preferences
   (連線設定, 媒體庫掃描, 自訂首頁, qBittorrent), ops dashboards (服務狀態, 系統日誌,
   效能監控, 快取管理), data safety (備份與還原, 匯出/匯入). Backlog Epics 16
   (stats) and 18 (health) currently have nowhere to land except more children.
5. **Search is three surfaces** (P9): header InstantSearchBar → `/search` (TMDB),
   `/discover` instant filtering, library-local search. `/search` has no nav
   presence — reachable only by submitting the search bar.
6. **PRD Journey 3 (系統管理) has no home:** the P4 dashboard journey (library
   stats, service health, disk projection, activity log) is scattered across
   settings children with no top-level anchor.
7. **Navigation has no flow of its own in the `.pen` canvas:** the A–J flows cover
   browse/detail/settings/etc., but the shell itself maps to only two component-level
   nodes (TabActive/TabInactive). The design side mirrors P2 — nobody owns navigation
   in code OR in design. Phase 1 must add a dedicated shell/nav flow block to the
   canvas so navigation becomes a designed surface for the first time.

### Requirements That Drive Nav IA

**Functional:** dual loops (Discovery feeds Appreciation — UX spec); J2 ends in
one-click request (Epic 13 wants a discovery-adjacent anchor); J3 is an entire
admin journey (Epics 16/18); fixed two-type domain (movie|series) + Epic 7b
multi-library folders; subtitle state as first-class dimension (N1, §4.3-3).

**Non-functional:** desktop-primary (27" density) + mobile-for-monitoring —
nav must serve both a deep desktop idiom and a thumb-reach mobile idiom; 44px
touch floor (R4); WCAG AA contrast (R5); dark theme only; zh-TW-first labels;
core actions ≤2 clicks (Plex-backlash constraint). The single daily user has
**muscle-memory switching costs** — sharper for a solo product than a multi-user
one, since no incoming new users amortize the relearning curve; migration UX
(bookmark and habit continuity) is a first-class requirement.

**Growth pressure (backlog epics needing IA slots):** 13 Requests · 14 Downloads
v2 · 16 Stats · 17 Continue-watching · 18 Health. Epic 15's scope is currently
undefined — **no IA slot is reserved for undefined requirements** (slots are
earned by defined scope, not placeholders). Option B's "keep top tabs" ceiling
(~8–10 slots) is real and near.

### Technical Constraints & Dependencies

- TanStack Router file-based routes — route moves are cheap, but deep-link
  stability is a hard constraint (E2E-guarded search params per Rule 26;
  existing URLs in user bookmarks). Route-level redirects keep old deep links
  (e.g. `/library?type=movie`) alive if destinations move.
- **Strangler migration — the shell swap is a single-point operation:**
  `AppShell` mounts only in `__root.tsx`; all routes render into its
  `<main>{children}</main>`. A feature flag around the shell component alone
  enables old/new chassis coexistence without per-screen rewrites. This lowers
  D1's migration cost below what the Phase 0 brief implied.
- **But two hidden costs:** (a) `ScanProgress` mounts OUTSIDE AppShell in
  `__root.tsx` — global overlay ownership must be explicitly assigned in the new
  shell or it stays an unowned corner; (b) the app currently lives in a centered
  `max-w-7xl` container — a sidebar changes the width budget, forcing recalculation
  of `LibraryGrid` column breakpoints and poster sizes (grid blast radius must be
  in the impact assessment). Counterweight: on the primary 27" display the
  centered container wastes large side margins — a sidebar converts idle
  whitespace into function, so its "-240px" cost is overstated on desktop.
- Backend impact is limited but non-zero: a unified Activity hub and/or ambient
  status strip needs aggregate endpoints (task queue summary, service health,
  disk stats) — new `/api/v1/*` surface following Rule 3/7/10; SSE hub already
  broadcasts scan/subtitle progress (reuse, don't re-invent).
- Enforcement riders (N6): Rule 21 headers for any new shell components,
  visual-regression baselines per Rule 22/23, jsx-a11y gates.

### Cross-Cutting Concerns

1. **N1 lifecycle visibility** — wherever tasks land (D4), every poster/detail
   surface must reflect the same state machine; nav choice determines whether
   in-flight state is ambient (status strip) or destination-only.
2. **Search consolidation** (P9 → D4-sub) interacts with D1: a sidebar gives
   omnisearch a persistent slot; top-tab layouts keep it in the header.
3. **Mobile nav idiom** is a separate decision surface from desktop (bottom
   tabs vs drawer) — must be decided WITH D1, not deferred (P5 debt pattern).
4. **Homepage identity split** (D3) gates Epic 17 (continue watching) and the
   Epic 10 investment's destination.
5. **Return-context preservation:** `/media/$type/$id` is entered from ≥4
   surfaces (library, search, discover, home). Preserving scroll position and
   filter state on back-navigation is a chassis responsibility, not a
   per-screen one.
6. **Test-contract blast radius:** `TabNavigation` carries testid contracts
   (`tab-媒體庫` etc.) assumed by Playwright E2E and the 50-case TestSprite
   journey catalog. The shell swap IS a test-contract bump and must be declared
   as such up front — affected testids, TestSprite plan regeneration, and a
   full visual-regression baseline re-shoot belong in the impact assessment,
   or Phase 2 will misread test failures as regressions.

### Decision Reversibility Grading

| Decision | Door type | Implication |
|---|---|---|
| D1 nav chassis | **One-way** (near) | Highest validation effort; pilot behind flag before commit |
| D2 movie/TV first-class | Two-way (redirects keep links alive) | Medium validation; URL scheme is the sticky part |
| D3 homepage split | Two-way (content reshuffling) | Decide fast, tune in Phase 2 |
| D4 sub-decisions | Two-way (mostly additive) | Decide fast; `/pending` demotion is near-zero cost |

### Scale & Complexity

- Primary domain: responsive web SPA (frontend-led; backend = aggregate endpoints only)
- Complexity: medium — single user, no multi-tenancy/compliance; complexity is
  concentrated in migration sequencing (strangler) and state-surface unification
- Component blast radius: 1 shell (AppShell/TabNavigation replacement), 5→N
  top-level routes, settings re-grouping, grid re-flow under new width budget,
  test-contract migration, ~2 aggregate API endpoints (estimate)

## Foundation Assessment (Starter Evaluation — N/A, Brownfield)

This ADR operates on a shipped production codebase. No starter template is
evaluated — the stack is locked by `project-context.md` and the existing
implementation (React + TanStack Router file-based routes; Tailwind CSS v4;
Nx/pnpm; Vitest/Playwright/TestSprite; Go-Gin backend in `apps/api` with an
existing SSE hub. Enforcement gates as listed under Context Analysis →
Technical Constraints).

**Foundation commitments for the nav chassis work:**

1. **One token ground truth, no forking.** `apps/web/src/styles.css` stays the
   single token file. The constraint is NO FORKING — not "additive-only":
   design language v2 is expected to CORRECT existing token values (e.g.
   `--text-muted` AA failure, R5) and add semantic tokens (R3). The shell
   consumes whatever v2 defines.
2. **Sequencing: v2 tokens are a BLOCKING input to the Phase-2 shell build**
   (one-way dependency on the parallel `01-design-language-v2.md` session),
   not a forever-parallel track.
3. **Legacy content container.** During the Phase-2/3 cascade the new shell
   hosts both redesigned and untouched screens. The new chassis MUST provide
   a legacy container that reproduces today's centered `max-w-7xl` canvas, so
   untouched flows render pixel-unchanged until their own migration story.
   Without this, the strangler degenerates into a forced big-bang reflow.
4. **Strangler flag with a kill clause (D1-c).** The flag is born with its
   retirement condition tied to Phase 2's go/no-go gate: pass → flag-removal
   story scheduled immediately; fail → new shell rolls back, flag removed.
   Dual-shell coexistence is capped at pilot scope (flows A+B). D1-c also owns
   the TEST-CONTRACT CUTOVER: which shell's testids/baselines gate CI during
   the dual-shell window, and when the switch flips.

**Explicitly NOT introduced by this redesign:**

- **No visual component library** (shadcn/MUI/DaisyUI/etc.) — design language
  v2 tokens + hand-rolled visual components remain the single visual authority.
  ⚠️ Carve-out requiring explicit decision (D1-d): HEADLESS a11y primitives
  (e.g. Radix UI / Base UI) for the shell's interactive primitives (drawer,
  popover menu, dialog focus management) are evaluated as a formal
  sub-decision — NOT banned outright. Rationale: hand-rolled dialog a11y
  failed in two consecutive epics (10-2, 11-2 — P4) and left a 7×-duplicated
  dismiss layer; re-hand-rolling a shell's worth of focus/keyboard behavior
  repeats a proven failure class.
- No state-management addition — TanStack Query (server) + local state remains.
- No routing paradigm change (no SSR/RSC migration).

## Core Architectural Decisions (D1–D4 Resolved)

All four open IA decisions from `00-redesign-brief.md` §6 are resolved below,
each with the chosen option, rejected alternatives, rationale, and the binding
guardrails attached during facilitation. These are AI-agent contracts: every
downstream Phase 1+ story implements against these, no re-litigation.

### D1 — Navigation Chassis

**D1-a Desktop: COLLAPSIBLE SIDEBAR (option A2).** Full-width (~240px,
icon+label) ↔ icon-rail (~64px), user-toggled, state persisted.

- Rejected: full-width-only sidebar (A1, fails N5 density-choice), icon-rail-only
  (A3, violates Plex-backlash "labels over glyphs"), top-tabs variants (B/C/D,
  hit the ~8–10 slot ceiling against backlog Epics 13/16/17/18), command-palette
  chassis (E, not viable as sole chassis — no mobile, monitoring needs _visible_).
- Rationale: sidebar evidence converges three ways (competitor consensus, Plex
  top-nav backlash, growth pressure); A2 over A1 buys N5 (density is the user's
  choice) for one extra collapsed-state design; on the 27" primary display the
  collapse converts the existing `max-w-7xl` idle side-margin into function.

**D1-b Mobile: BOTTOM TAB BAR + "MORE" SHEET (option b-3).** 4 highest-frequency
destinations in the thumb-reach bar; 5th slot opens a bottom sheet for the rest.

- Rejected: pure bottom-tabs (b-1, ≤5-slot hard cap starves destinations),
  hamburger drawer (b-2, discoverability + top-left thumb-reach cost), status quo
  scrolling top-tabs (b-4, is itself a current defect — R4 <44px, top is thumb
  dead-zone).
- Rationale: mobile is a monitoring context; high-freq direct + low-freq in sheet;
  reuses the existing Epic 11 bottom-sheet component (don't rebuild).

**D1-c Strangler flag with kill clause.** (Carried from Foundation Assessment.)
Flag born with retirement tied to Phase 2 go/no-go: pass → flag-removal story
scheduled; fail → rollback + flag removed. Dual-shell capped at pilot scope
(flows A+B). Owns the test-contract cutover (which shell's testids/baselines
gate CI during the dual-shell window, and when the switch flips).

**D1-d Interaction a11y primitives: BASE UI (option 3).** `@base-ui-components/
react` for the shell's interactive primitives (tooltip — required by the A2
collapsed rail — popover menu, drawer/sheet, dialog focus management).

- Rejected: hand-rolled (option 1, P4 proved this fails — a11y broke in Epics
  10-2 & 11-2, left a 7×-duplicated dismiss layer), Radix (option 2, maintenance
  slowed post-WorkOS acquisition per 2026 web check), React Aria (option 4,
  steepest hooks learning curve for solo+AI pipeline), Headless UI (option 5,
  no Tooltip — A2 needs it), Ark UI (option 6, low React-ecosystem penetration).
- Rationale: outsource focus/keyboard behavior to the most actively-maintained
  team in 2026 (Base UI v1.0 shipped 2025-12, 35 components, MUI long-term
  maintenance commitment, ex-Radix/Floating-UI core members). Web-verified.
- NOTE: this introduces the FIRST runtime UI dependency. Visual authority stays
  100% with design-language-v2 tokens (Base UI ships unstyled). Existing 7
  hand-rolled dialogs are NOT force-migrated (separate backlog item); new shell
  primitives use Base UI going forward.

### D2 — Movies / TV Placement

**SIDEBAR GROUP (option 5): "媒體庫" group-label (→ merged view) with nested
"電影" / "影集" children (→ type views).** Expanded: all three rows direct-click;
collapsed rail: one icon.

- Rejected: unified library + in-page type filter (option 1, buries the
  highest-frequency pivot one level down), movies/TV as flat top-level without
  "all" (option 2, kills cross-type browse + eats 2 mobile slots), three flat
  destinations (option 3, mobile 4-slot bankruptcy + tri-config grid drift),
  library-instance model (option 4, serves multi-user generality Vido doesn't
  need; Epic 7b `auto_detect` reserve suggests model still evolving).
- Rationale: only option that gets type-direct-access AND merged view AND mobile
  1-slot; leaves a structural escape hatch for Epic 7b (library instances could
  later nest under the children). First two-level sidebar structure — group-label
  click affordance must be designed explicitly (Phase 1 design-language task).

**Anime sub-decision: 甲 — PINNABLE SAVED VIEW.** Anime is a content dimension
(genre/keyword) spanning BOTH types (劇場版 = movie, 番劇 = series), not a third
type — making it a type would fight the movie|series data model. Implemented as
a saved filter preset (reuses Epic 11 preset mechanism, zero backend cost),
pinnable to the sidebar under the 媒體庫 group as a shortcut.

- **GUARANTEED FEATURE (user-mandated):** "pin a saved view to the sidebar" is a
  GENERAL mechanism, not an anime-specific hack. ANY filter preset (高分日劇,
  未看 4K, etc.) can be pinned. This flexibility is a binding requirement of D2,
  not an optional nicety. Preserves saved-view flexibility per Alexyu's explicit
  condition (2026-06-13).

### D3 — Homepage / Library / Discover Responsibility

**MIXED OWN+EXTERNAL CURATION ON HOMEPAGE (option 2), HOT treatment.** Epic 10's
HeroBanner + ExploreBlocksList stay on the homepage in full. Homepage feeds both
loops (discovery + appreciation) from one landing page.

- Rejected: home=library-pulse-only / discover=external (option 1, demotes the
  Epic 10 investment + loses "open-the-app beauty"), discover-merged-into-home
  (option 3, super-long mixed page), home-eliminated (option 4, no home for the
  N1 lifecycle dashboard).
- Rationale: user's explicit product call — the Epic 10 hero/explore investment
  carries non-negotiable emotional weight; "open the app to something beautiful"
  is a deliberate product value.
- **BINDING GUARDRAILS (attached to avoid the Plex trap that option 2 risks):**
  1. **Ordering law:** own-content (recently-added / continue-watching /
     task-status) is ALWAYS structurally ABOVE external curation. A design RULE,
     not a default; Phase 2 acceptance must verify it.
  2. **Home/Discover boundary, explicit:** Home = passive curation storefront
     (user-defined blocks via Epic 10 block-CRUD). Discover = active power-filter
     tool (chips/presets). Same external content, but **home grows no filters,
     discover grows no dashboard** — the un-ambiguous dividing line (resolves P9).
  3. **Dashboard elements move OFF home:** Epic 4 remnants (DownloadPanel,
     QBStatusIndicator, ConnectionHistoryPanel) relocate to the D4 activity/
     status surfaces. The two-identities-cohabiting problem is solved by moving
     the dashboard half out; the curation half stays in full.

### D4 — Operations Surface

**D4-1 Activity hub: HYBRID (option 丙).** One sidebar "活動/Activity" destination
unifies parse + subtitle-batch + scan + AI-jobs with explain-why rows; downloads
KEEP a dedicated deep page (Epic 14 v2's pagination/batch-ops needs don't fit a
hub), surfaced as a summary row in the hub that links into the deep page.

- Rationale: solves P8 (invisible journeys incl. the never-built AI-subtitle UI)
  without cramming the high-volume downloads surface.

**D4-2 Ambient status strip: SIDEBAR-FOOTER (option 甲).** Disk bar / active-scan
/ queue-count / service-health dots live in the A2 sidebar footer; collapse to
dots on the icon-rail. Mobile home: top of the "More" sheet.

- Rationale: the dividend of choosing A2 — N1 ambient visibility with zero
  vertical-space cost (the top-tabs+strip variant D's drawback vanishes); partly
  absorbs Epics 16/18 needs without new pages.

**D4-3 Settings split: SETTINGS + SYSTEM (option 甲, \*arr model).** Settings =
preferences (連線/掃描/首頁/qBT). System = ops dashboards (狀態/日誌/快取/效能/
備份/匯出). Backlog Epics 16 (stats) / 18 (health) land in System.

- Rationale: gives the 11 children a home; ops belongs to the N3 task side; the
  \*arr Settings-vs-System boundary is battle-tested. Costs +1 sidebar slot.

**D4-4 `/pending` 待解析: BOTH (option 丙).** Activity hub shows an "N pending
parse" row; clicking jumps to the library's unmatched-filter view. Near-zero
migration cost (currently a static stub).

- Rationale: both semantics are correct — visible as activity, processed in
  library context.

**D4-5 Search: GLOBAL OMNISEARCH (option 甲).** Header search upgrades to one box
searching local-library + TMDB + (future) subtitles with sectioned results;
`/search` page retired.

- Rejected: scoped-search (乙, the scope toggle is itself a P9 confusion source),
  placeholder-only patch (丙, status-quo non-fix).
- Rationale: omnisearch is the only root-cause fix for P9's three search surfaces;
  the aggregate endpoint batches with the D4-1/D4-2 activity+status endpoints.

### Resulting Top-Level IA (destinations)

Sidebar (desktop) / bottom-tabs+more (mobile):

1. **首頁 Home** — mixed curation (own-above-external) + Hero/explore blocks
2. **媒體庫 Library** ▸ 電影 Movies · 影集 TV (+ pinned saved views, e.g. 動畫)
3. **探索 Discover** — active power-filter tool (chips/presets) + Epic 13 requests
4. **活動 Activity** — parse/subtitle/scan/AI jobs + downloads-summary row
5. **下載 Downloads** — dedicated deep page (linked from Activity)
6. **系統 System** — stats/health/logs/cache/backup/export (Epics 16/18)
7. **設定 Settings** — preferences

- Sidebar footer: ambient status strip · Header: omnisearch

Mobile bottom-4 (proposal, refined in Phase 1 design): 首頁 · 媒體庫 · 探索 ·
活動 — with 下載/系統/設定 + status in "More". (Final bottom-4 selection is a
Phase 1 design-language task → `01-design-language-v2.md`.)

## Implementation Patterns & Consistency Rules

This redesign DEFERS to the existing 27 rules in `project-context.md` for all
naming / response-format / error-code / test-colocation / state-management /
case-transform conventions. The patterns below cover ONLY the new conflict
points the nav-IA redesign introduces — where AI agents would otherwise diverge.
Hardened through Party Mode (Amelia/Murat/Bob) + Failure-Mode + Pre-mortem
elicitation.

> **Two-document seam (P1 hardening):** every item below marked `→ DL-v2` defers
> its VISUAL definition to the parallel `01-design-language-v2.md` session. This
> ADR owns BEHAVIOR/STRUCTURE; design-language-v2 owns visual tokens/states. The
> seam between the two parallel Phase-1 sessions is the highest divergence risk —
> nothing visual is invented here, nothing structural is invented there.

### Routing Patterns (NEW)

**Nested route scheme mirrors the sidebar group (fork A → A1):**

| Destination | Route | Notes |
|---|---|---|
| Home | `/` | unchanged |
| Library (merged) | `/library` | group-label landing = cross-type view |
| Movies | `/library/movies` | type view |
| TV | `/library/tv` | type view |
| Discover | `/discover` | unchanged (active power-filter tool) |
| Activity | `/activity` | NEW — D4-1 hub |
| Downloads | `/downloads` | unchanged (deep page, linked from Activity) |
| System | `/system` | NEW — D4-3 ops dashboards (was settings children) |
| Settings | `/settings` | preferences only |

- **Library route restructure (Amelia):** the current single-file `library.tsx`
  splits into a LAYOUT route + three child views: `library.tsx` (layout — shared
  toolbar/filter/sort + `<Outlet/>`), `library/index.tsx` (merged view),
  `library/movies.tsx`, `library/tv.tsx`. The three views SHARE ONE grid
  component — never duplicate grid logic (avoids the P7 PosterCard black hole).
- **Shared state lives in the layout route (F5):** filter + scroll state is held
  by the `library.tsx` layout and consumed by children; switching movies↔tv
  preserves compatible filters and scroll position (Context-Analysis
  cross-cutting #5 return-context).
- **Deep-link preservation is mandatory, route-level (F1).** Old links redirect,
  never 404 (P2 lesson). Redirects use TanStack `beforeLoad` throwing
  `redirect()` — NEVER a component-level `useEffect`/`<Navigate>` (a component
  redirect flashes wrong content and breaks on hard load of a bookmark):
  - `/library?type=movie` → `/library/movies` (`type=series` → `/library/tv`);
    `type` coerced per Rule 26 before redirect.
  - `/search?q=…` → omnisearch surface, preserving `q` (fork D4-5).
  - `/pending` → library unmatched view (`/library/movies?unmatched=1` style,
    fork D4-4); an Activity row is the alternate entry.
  - `/settings/{status,logs,cache,performance,backup,export}` →
    `/system/*` (System move); old paths keep a redirect for one release.

### Shell Component Patterns (NEW)

- New shell components live in `apps/web/src/components/shell/`. Naming:
  `AppSidebar`, `SidebarNav`, `SidebarGroup`, `SidebarNavItem`, `SidebarFooter`
  (status strip), `MobileTabBar`, `MobileMoreSheet`, `LegacyContentContainer`.
  `AppShell` is replaced/wrapped, not forked. Visual states → DL-v2.
- **`AppShell` remains the single mount point** (`__root.tsx`); `ScanProgress`
  global-overlay ownership moves explicitly INTO the new shell (Context-Analysis
  hidden-cost (a)).
- **Active-state via TanStack built-in matching (Amelia):** use Router active
  matching (`Link` activeProps / `useRouterState` matches) — do NOT hand-roll
  `startsWith` like today's `TabNavigation`. A child route (`/library/movies`)
  marks BOTH its `SidebarNavItem` active AND its `SidebarGroup` as
  active-ancestor (distinct visual state → DL-v2). Group-label is itself a Link
  to the merged view; click affordance must be unambiguous (→ DL-v2).
- **`LegacyContentContainer` opt-in mechanism (Amelia):** routes default to the
  legacy container (reproduces today's centered `max-w-7xl` canvas, pixel-
  unchanged); a MIGRATED route explicitly opts into the new layout via a
  route-level marker. This gives a single source of truth for "is this screen
  migrated yet" during the strangler cascade.
- **testid contract cutover (D1-c / Murat — phased, not instant):**
  - New sidebar items expose `nav-{key}` (`nav-library`, `nav-activity`, …).
  - During the dual-shell window BOTH testid sets coexist (legacy shell keeps
    `tab-{label}`, new shell has `nav-{key}`); tests target whichever shell the
    `new_shell_enabled` flag selects. TestSprite 50-case catalog runs against the
    FLAG-ON (new shell) deployment.
  - `tab-{label}` testids are removed at FLAG RETIREMENT (not at cutover start),
    in the same story that deletes the legacy shell. Not aliased.
- **Rule 21 headers:** new shell components carry the design-coverage-gap variant
  `// Design ref: ux-design.pen — no current screen frame (Phase 1 nav shell)`
  until the Phase 1 `.pen` shell flow exists (→ DL-v2 owns that flow).

### Base UI Usage Patterns (NEW — first runtime UI dependency)

- Base UI primitives are wrapped ONCE per primitive in `apps/web/src/components/
  ui/` (`Tooltip.tsx`, `Popover.tsx`, `Sheet.tsx`, `Dialog.tsx`) — routes/shell
  import the wrapper, never `@base-ui-components/react` directly.
- **Enforced, not aspirational (F2 / N6):** an ESLint `no-restricted-imports`
  rule BANS `@base-ui-components/react` outside `apps/web/src/components/ui/`.
  This makes "wrap once" mechanical, and a future primitive swap a single-dir
  change.
- Wrappers apply ZERO visual opinion beyond design-language-v2 tokens (Base UI
  ships unstyled — token classes only, no hardcoded values, R3 → DL-v2).
- Existing 7 hand-rolled dialogs are NOT force-migrated (separate backlog item);
  new shell/overlay surfaces use the Base UI wrappers going forward.

### Saved-View Pin Patterns (fork C → C3)

- Reuses the existing `/api/v1/filter-presets` backend (Story 11-4 — already
  server-side: `name`, `filters` JSON, `sort_order`, `created_at`). Pin = a new
  `pinned BOOLEAN` column on the existing `filter_presets` table (migration per
  Rule 15 DB-column-sync); pin ordering reuses the existing `sort_order`.
- No new table, no new service. Sidebar reads pinned presets via the existing
  `useFilterPresets` hook (Rule 5 TanStack Query); pinned views render as
  `SidebarNavItem`s under the 媒體庫 group.
- **Pin is purely ADDITIVE (P2):** pinning a preset ALSO surfaces it in the
  sidebar — it never removes the preset from its Discover preset-chips home.
- "Pin to sidebar" is the GENERAL mechanism (D2 guarantee) — anime is the first
  instance, not a special case.

### Aggregate-Endpoint Error & Fail-Soft Patterns (fork B → B1)

- New aggregate endpoints (`/activity` summary, status strip, omnisearch) are
  COMPOSITIONS, not sources. Their own errors map to EXISTING Rule 7 prefixes:
  `VALIDATION_*` (bad input — empty query, invalid cursor) and `DB_*`
  (persistence). No new prefix is introduced.
- **Fail-soft is BIDIRECTIONAL (F3):**
  - Backend: a downstream failure returns `success: true` with a per-section
    status object (`{ section, status: "ok"|"unavailable"|"stale",
    error?: {code,…} }`); the failed section carries its downstream's ORIGINAL
    prefix (`QBITTORRENT_*` etc.), never collapsed into an aggregator code,
    never fail-page (Rule 27 Pillar 3 / N1 / N4).
  - Frontend: the TanStack Query treats a non-`ok` section as DATA, rendering an
    empty/stale section — it MUST NOT throw or fail the page. (Backend fail-soft
    is worthless if the frontend hard-fails on it.)
- **Upgrade trigger (documented YAGNI):** a new prefix is added ONLY when an
  aggregate endpoint grows its own domain logic with failure modes outside
  validation/DB/downstream (e.g. a fallible ranking engine in omnisearch). Then
  follow the full Rule 7 procedure (extend the prefix table AND sync
  `code-review/instructions.xml` Rule-7 check). Mirrors the project's
  externalapi-extraction YAGNI (Rule 27 ADR, ADR-3).

### Feature-Flag Patterns (fork D → D2)

- The strangler shell flag is a RUNTIME settings flag (settings table, toggled
  from Settings UI), NOT a build-time env var — D1-c requires fast rollback +
  test-contract cutover without a rebuild.
- Flag naming: `new_shell_enabled` (settings key), read via the existing
  settings hook.
- **Single chokepoint (F4):** the flag is read in EXACTLY ONE place —
  `__root.tsx` selecting new `AppShell` vs the legacy shell. No route or
  component branches on it. Flag retirement = delete one read site + the legacy
  shell + `LegacyContentContainer`.
- **Strangler discipline (P3 — rigid):** a shell-migration story MUST NOT modify
  legacy route CONTENT. Legacy screens change ONLY in their own flow-migration
  story. "While I'm here" edits to legacy pages during shell work are banned
  (the P7 scope-creep class).
- Retirement (D1-c): Phase 2 go → flag-removal story; no-go → flag flips back,
  new shell dormant behind it. Dual-shell window capped at pilot scope (flows
  A+B).

### Story-Sequencing Dependencies (Bob — carried to Step 6 impact assessment)

- **Foundation story FIRST:** new shell + `new_shell_enabled` flag +
  `LegacyContentContainer` land together, before ANY flow migration.
- Independently-tracked units (Rule 24 triage, NOT bundled into the shell story):
  `/system` move (6 settings children + redirects), `pinned` migration, the
  route-redirect set. Each is its own sprint-status entry.

### Enforcement (defers to existing N6 gates + one new ESLint rule)

New components ride EXISTING gates: Rule 21 ESLint `implements-pen-node-id`,
Rule 22/23 visual-regression CI, `jsx-a11y` (Base UI helps satisfy this),
Rule 12 `pnpm lint:all`, Rule 15 self-verification (`pinned` migration + flag
wiring). NEW: the F2 `no-restricted-imports` rule for Base UI. **Legacy-fidelity
acceptance (Murat):** untouched routes' EXISTING visual baselines must still
PASS under the new shell — they are the acceptance test for
`LegacyContentContainer` pixel-fidelity, not a re-shoot.

## Project Structure & Impact Assessment

Brownfield impact map: every D1–D4 decision traced to concrete files. Verb tags:
🆕 new · ✏️ modified · ➡️ moved · 🗑️ removed/redirected · 🔁 reused-as-is.

### Frontend — Route Impact (`apps/web/src/routes/`)

| Route file | Change | Driver |
|---|---|---|
| `__root.tsx` | ✏️ shell selection behind `new_shell_enabled` flag (single chokepoint, F4); move `ScanProgress` ownership into new shell | D1-c, D4 |
| `index.tsx` | ✏️ keep Hero+Explore (D3 hot); REMOVE dashboard elements (DownloadPanel/QBStatusIndicator/ConnectionHistoryPanel → Activity/status); enforce own-above-external ordering | D3 |
| `library.tsx` | ✏️→ becomes LAYOUT route (shared toolbar/filter/sort + `<Outlet/>`, shared grid component, shared filter/scroll state F5) | D2, A1 |
| `library/index.tsx` | 🆕 merged cross-type view | D2 |
| `library/movies.tsx` | 🆕 movie type view | D2 |
| `library/tv.tsx` | 🆕 series type view | D2 |
| `discover.tsx` | ✏️ stays the active power-filter tool; grows NO dashboard (D3 guardrail #2); host Epic 13 requests later | D3 |
| `activity.tsx` | 🆕 D4-1 hub (parse+subtitle+scan+AI jobs + downloads-summary row) | D4-1 |
| `downloads.tsx` | 🔁 stays a deep page; gains "linked from Activity" entry | D4-1 |
| `pending.tsx` | 🗑️ → redirect to library unmatched view + Activity row | D4-4 |
| `search.tsx` | 🗑️ → omnisearch surface (route-level `beforeLoad` redirect preserving `q`) | D4-5 |
| `settings.tsx` | ✏️ preferences only (連線/掃描/首頁/qBT) | D4-3 |
| `system.tsx` (+children) | 🆕 ops dashboards; ➡️ move `settings/{status,logs,cache,performance,backup,export}.tsx` here; old paths redirect 1 release | D4-3 |
| `settings/homepage.tsx`, `connection.tsx`, `scanner.tsx`, `qbittorrent.tsx` | 🔁 stay under Settings | D4-3 |

All redirects: route-level `beforeLoad` throwing `redirect()` (F1), never 404 (P2).

### Frontend — Component Impact (`apps/web/src/components/`)

- `shell/` — 🗑️ `TabNavigation.tsx`; 🆕 `AppSidebar`, `SidebarNav`, `SidebarGroup`,
  `SidebarNavItem`, `SidebarFooter` (status strip), `MobileTabBar`,
  `MobileMoreSheet`, `LegacyContentContainer`; ✏️ `AppShell` (wrap + flag-aware,
  owns `ScanProgress`). Visual states → DL-v2.
- `ui/` — 🆕 Base UI wrappers: `Tooltip`, `Popover`, `Sheet`, `Dialog`
  (the only place importing `@base-ui-components/react`; ESLint-enforced F2).
- `dashboard/` — ➡️ `DownloadPanel`, `RecentMediaPanel` relocate from home to
  Activity/Library; `health/` `QBStatusIndicator`/`ConnectionHistoryPanel` →
  status strip / System.
- `library/` — ✏️ extract a single shared `LibraryGrid` consumed by the 3 views;
  grid breakpoints recompute under the sidebar width budget (Context hidden-cost
  (b) → DL-v2 owns final columns).
- `homepage/` (`HeroBanner`, `ExploreBlocksList`) — 🔁 stay on home (D3 hot).
- `search/` — ✏️ `InstantSearchBar` → omnisearch (local+TMDB sectioned results);
  `PresetChips`/`SavePresetDialog` gain a "pin to sidebar" affordance (C3).

### Backend — Impact (`apps/api/internal/`)

- **Feature flag — NO new table.** `new_shell_enabled` is a row in the existing
  key/value settings table; read via `SettingsService.GetBool` (verified to
  exist). Frontend reads via the settings hook. (D1-c / D2-flag)
- **Saved-view pin (C3):** ✏️ `filter_presets` table gains `pinned BOOLEAN`
  (migration per Rule 15); ✏️ `filter_presets_handler.go` + repo INSERT/UPDATE/
  list include `pinned` + order by existing `sort_order`. NO new table/service.
- **Activity hub (D4-1):** 🆕 `activity_handler.go` — a COMPOSITION reading
  existing `download_service`, `parse_job`/`parse_progress`, `scanner_service`,
  subtitle services; returns per-section status objects (fail-soft F3, prefix
  reuse B1). `GET /api/v1/activity`. No new error prefix.
- **Status strip (D4-2):** ✏️ EXTEND existing `status_handler.go`
  (`GET /api/v1/settings/services` already aggregates service health) with disk
  stats + active-scan + queue-count; or 🆕 `GET /api/v1/status/summary` if the
  shape diverges. Reuses `ServiceStatusService`. (Decide exact endpoint in impl.)
- **Omnisearch (D4-5):** ✏️ EXTEND `search_service.go` (today TMDB-only) to
  ALSO query the local library, returning sectioned results (local / TMDB /
  future subtitles); errors are `VALIDATION_*` (empty query) — no new prefix.
  `GET /api/v1/search` extended or 🆕 `/api/v1/search/omni`.
- **SSE reuse:** 🔁 the existing SSE hub already broadcasts scan/subtitle
  progress — the Activity hub SUBSCRIBES, does not re-poll (Rule 8 lazy SSE).

### Architectural Boundaries (unchanged contracts)

- Handler → Service → Repository → DB (Rule 4) holds for all new endpoints.
- Aggregate handlers READ existing services; they do NOT reach into repositories
  directly (Rule 4) and do NOT duplicate subsystem logic.
- Case-transform at the API boundary (Rule 18) for all new endpoints.
- Frontend server-state via TanStack Query only (Rule 5); flag/shell UI state is
  local. Per-section status is DATA, never a thrown error (F3).

### Requirements → Structure Mapping (backlog readiness)

- Epic 13 (Requests) → lands in `discover.tsx` (D3 boundary: discovery side).
- Epic 16 (Stats) / Epic 18 (Health) → land under `system.tsx` (D4-3).
- Epic 17 (Continue-watching) → a home own-content block, ABOVE external (D3 #1).
- Epic 14 v2 (Downloads) → the retained `downloads.tsx` deep page (D4-1).
- Epic 7b (multi-library) → future nesting under `library/` children (D2 escape).

### Story-Sequencing (Bob — Rule 24 triage, for sprint-planning)

1. **FOUNDATION (first, blocking):** new shell + `new_shell_enabled` flag +
   `LegacyContentContainer` + Base UI wrappers + ESLint ban (F2). All legacy
   routes render unchanged inside the new shell (legacy-fidelity baselines pass).
2. Independent units (own sprint-status entries, NOT bundled): `/system` move +
   redirects · `pinned` migration + handler · `/library` nested restructure ·
   omnisearch service extension · activity handler · status-strip endpoint.
3. Pilot flows A (Browse) + B (Detail) migrate behind the flag → Phase 2 gate.

### Backend Change Magnitude

Limited: 1 column migration, 1 new aggregate handler, 2 extended handlers/
services, 0 new tables, 0 new error prefixes, 0 new external integrations. The
redesign is frontend-led; the backend mostly composes what already exists.

## Architecture Validation Results

### Coherence Validation ✅

- **No contradictory decisions.** D1 sidebar enables D2 group-nesting, D4-2
  status-strip, and pinned saved-views (C3) — each depends on the sidebar chosen
  in D1-a. The set is mutually reinforcing, not just compatible.
- **One deliberate principle-tension, resolved + flagged.** D3 (hot homepage,
  Hero+external prominent) softens N3 "content first, tasks adjacent" and brushes
  the Plex-backlash constraint. Resolved by the 3 binding guardrails (own-above-
  external ordering law, home/discover boundary, dashboard-moved-off-home). This
  is the single place a user product-call overrode a principle — it is the
  PRIMARY thing Phase 2's go/no-go gate must watch.
- **Patterns support decisions:** route scheme (A1) ↔ sidebar group (D2); fail-
  soft bidirectional (F3) ↔ N1/N4; flag chokepoint (F4) ↔ strangler retirement.

### Requirements Coverage Validation ✅

- **Brief's 4 open questions — ALL answered:**
  1. Anime first-class? → NO; pinnable saved view (D2 甲), a general mechanism.
  2. Epic 10 hero/explore emotional weight? → Non-negotiable; D3 hot, kept whole.
  3. Ambient status strip appetite? → YES; D4-2 甲 (sidebar footer).
  4. Reserve slots for archived ambitions? → Epic 17 (continue-watching) IS
     defined → home own-content block. Multi-user v5 → NO reserved slot
     (undefined scope earns no slot).
- **Backlog epics 13/14v2/16/17/18 all have a mapped home** (Requirements→
  Structure Mapping). No backlog epic is left homeless by this IA.
- **NFRs:** desktop density (A2 expand) + mobile thumb-reach (b-3) + 44px (R4) +
  AA (R5→DL-v2) + zh-TW labels + ≤2-click core actions — all addressed; deep-
  link stability via route-level redirects (F1).

### Implementation Readiness Validation ✅

- Decisions documented with rejected alternatives + rationale (AI-agent contract
  form). Patterns hardened through 3 elicitation passes. Impact assessment traces
  every decision to concrete files. Backend magnitude bounded (0 new tables/
  prefixes/integrations).

### Gap Analysis — 3 MEDIUM items deferred to Phase 1 design (none blocking)

1. **Mobile bottom-4 composition unproven.** Proposal = 首頁·媒體庫·探索·活動.
   Is 活動 higher-frequency than 下載 for THIS single user? Decide empirically in
   Phase 1 design (→ DL-v2), not now.
2. **Status-strip & omnisearch endpoint shape = "extend vs new"** left to impl
   (extend `status_handler`/`search_service` vs new `/status/summary`,
   `/search/omni`). Acceptable for an ADR; flag as a deferred impl sub-decision.
3. **Collapsed-rail icon density.** 7 destinations + status dots + pinned views
   on a 64px rail risks crowding + icon ambiguity (the A3 failure mode in
   miniature). DL-v2 must validate the collapsed-rail information budget and the
   pinned-view overflow rule (how many pins before "more").

### Architecture Completeness Checklist

- [x] Context analyzed · scale assessed · constraints + cross-cutting mapped
- [x] D1–D4 decided with rationale + reversibility grading
- [x] Patterns: routing/shell/Base UI/pin/aggregate-error/flag + enforcement
- [x] Impact assessment: routes + components + backend + sequencing
- [x] Brief's open questions answered · backlog epics mapped

### Readiness Assessment

**Overall:** READY FOR PHASE 2 (Browse+Detail pilot).
**Confidence:** HIGH — decisions cohere, requirements covered, backend bounded.
**Key strengths:** sidebar choice compounds (D2/D4-2/C3 all ride it); backend is
mostly composition; strangler discipline + flag-retirement prevent the dual-shell
trap; every deferral has an explicit `→ DL-v2` handoff.
**Watch in Phase 2:** the D3 hot-homepage principle-tension (guardrails must hold
under real content); mobile bottom-4; collapsed-rail density.

### Handoff

- This ADR + `01-design-language-v2.md` are the two Phase-1 foundations. Seam
  protocol: ADR owns behavior/structure, DL-v2 owns visual tokens/states; every
  `→ DL-v2` marker is a handoff point.
- First implementation = the FOUNDATION story (shell + flag + LegacyContent-
  Container + Base UI wrappers + ESLint ban), before any flow migration.

## Architecture Completion Summary

**Nav/IA Decision Workflow:** COMPLETED ✅ · 8 steps · 2026-06-13
**Document:** `_bmad-output/planning-artifacts/ux-redesign/01-nav-ia-decision-adr.md`

### Decisions Made (the AI-agent contract)

| ID | Decision |
|---|---|
| D1-a | Collapsible sidebar (desktop) |
| D1-b | Bottom tab bar + "More" sheet (mobile) |
| D1-c | Strangler feature flag with kill clause + test-contract cutover |
| D1-d | Base UI headless primitives (first runtime UI dep; unstyled) |
| D2 | Sidebar group: 媒體庫 ▸ 電影/影集 + pinnable saved views (anime = saved view) |
| D3 | Hot mixed-curation homepage + 3 binding guardrails |
| D4-1 | Hybrid Activity hub (downloads keep deep page) |
| D4-2 | Ambient status strip in sidebar footer |
| D4-3 | Settings / System split (*arr model) |
| D4-4 | `/pending` → both Activity row + library unmatched view |
| D4-5 | Global omnisearch (`/search` retired) |
| Forks | A1 nested routes · B1 reuse error prefixes · C3 reuse preset backend + `pinned` · D2 runtime settings flag |

### Resulting IA

7 destinations (首頁 · 媒體庫▸電影/影集 · 探索 · 活動 · 下載 · 系統 · 設定) +
sidebar-footer status strip + header omnisearch.

### Status

**READY FOR PHASE 2** (Browse + Detail pilot, behind `new_shell_enabled`).
Pairs with `01-design-language-v2.md` (parallel Phase-1 session) via the `→ DL-v2`
seam protocol. First story = the FOUNDATION story.

### Document Maintenance

This ADR records PLANNED conventions. The shipped conventions enter
`project-context.md` only when the FOUNDATION story actually lands (project-context
records what IS, not what's planned) — defer that update to implementation.
