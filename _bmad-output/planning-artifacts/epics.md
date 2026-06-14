---
stepsCompleted: ["step-01-validate-prerequisites", "step-02-design-epics", "step-03-create-stories"]
inputDocuments:
  - "_bmad-output/planning-artifacts/ux-redesign/03-phase3-destination-epic-map.md"
  - "_bmad-output/planning-artifacts/ux-redesign/01-nav-ia-decision-adr.md"
  - "_bmad-output/planning-artifacts/ux-redesign/01-design-language-v2.md"
  - "_bmad-output/planning-artifacts/ux-redesign/02-pilot-validation.md"
  - "_bmad-output/planning-artifacts/ux-redesign/00-redesign-brief.md"
  - "_bmad-output/planning-artifacts/prd/functional-requirements.md"
  - "_bmad-output/planning-artifacts/prd/non-functional-requirements.md"
  - "_bmad-output/planning-artifacts/epics/epic-list.md"
  - "_bmad-output/implementation-artifacts/sprint-status.yaml"
---

# vido — Epic Breakdown (UX Redesign Phase 3 — per-flow strangler cascade)

## Overview

This document is the requirements inventory for **UX Redesign Phase 3**: the
flow-by-flow strangler migration of Vido's remaining navigation destinations to the
v2 design language + 7-destination IA (ADR D1–D4), behind `new_shell_enabled`.

**Scope framing (critical):** Phase 3 is a **re-skin + re-chassis of already-shipped
features**, not new-product FR decomposition (`00-redesign-brief.md` §2). The
requirement source is therefore the **ADR + Design-Language-v2 + the destination→epic
map (`03-...`)** plus a curated set of **existing PRD FRs the redesign re-homes or
completes**, plus a small set of **genuinely-new backend FRs** the pilot surfaced
(`02-pilot-validation.md` §4). Media Library (Browse A + Detail B) is **already
migrated** in the Phase-2 pilot and is out of scope except for cleanup.

## Requirements Inventory

### Functional Requirements

**A. New backend/shell FRs (redesign-surfaced — not in prior PRD):**

- **PH3-F1 — N1 item-level lifecycle field.** Movies/series expose an item-level
  **durable** lifecycle/subtitle field so a poster badge can truthfully render the
  **load-time-queryable** states (`整理中`/`已入庫`/`失敗` + `繁中`/`缺字幕`). **Transient
  process states (`簡轉繁`/`AI 校正中`) relocate to the Activity hub's live SSE** (Epic 2+),
  and `下載中·%` defers to Epic 13/14 — so N1's poster-badge promise converges to durable
  state **by capability** (pilot §4.2: rich process states are not derivable on a steady
  library grid). Backend field + migration + API exposure + poster-badge wiring.
  *(Serves N1; gates Home + Activity.)*
- **PH3-F2 — Status-summary aggregate.** `GET /api/v1/status/summary` returns disk
  headroom + active-scan + queue-count + service-health as a **fail-soft per-section**
  aggregate, feeding the sidebar-footer status strip (D4-2). *(pilot §4.3; partially
  delivers Epic 18 P4-020/P4-021.)*
- **PH3-F3 — Clean-route split.** `/library/movies` + `/library/tv` route-file views
  with `beforeLoad` redirects from `/library?type=`; deep links preserved. *(D2; pilot
  §4.4 — low-risk cleanup.)*
- **PH3-F4 — Activity aggregate + hub.** `GET /api/v1/activity` composition over
  parse / subtitle-batch / scan / AI-jobs + a downloads-summary row, explain-why rows;
  the `/activity` destination renders it (D4-1). *(Delivers Epic 18 P4-022 activity-log
  slice; surfaces P8 invisible journeys.)*
- **PH3-F5 — Global omnisearch.** Header search becomes one box over local-library +
  TMDB (+ future subtitles), sectioned results; `/search` retired with a `q`-preserving
  redirect (D4-5). Extends `search_service`.

**B. Migration FRs (re-skin/re-chassis of EXISTING shipped features → v2; behavior
preserved, each adopts the N4 four-state standard + v2 tokens/type):**

- **PH3-M1 — Home v2** (re-chassis Epic 10 Hero/ExploreBlocks, P2-001–006) with
  **own-above-external ordering (D3)**; dashboard remnants relocated off home.
- **PH3-M2 — Discover v2** (re-chassis Epic 11 chips/presets/instant-search,
  P2-010–015) as the active power-filter tool; grows no dashboard (D3 boundary).
- **PH3-M3 — Downloads v2** (Epic 14; P3-010 done, P3-012 SSE) — v2 deep page + card
  actions; linked from Activity.
- **PH3-M4 — System/Settings split** (Epic 6 settings children) → `/system` (ops:
  status/logs/cache/performance/backup/export) vs `/settings` (prefs:
  連線/掃描/首頁/qBT), \*arr model (D4-3); route-level redirects 1 release.
- **PH3-M5 — Subtitle UI v2** (Epic 8 manual + batch, P1-016–019) — v2; detail
  `管理字幕` entry + Activity batch surface.
- **PH3-M6 — Detail Epic-12 design backfill** (P2-020–025: dual ratings / seasons /
  recs / streaming / trailer / douban) — v2 design coverage (shipped in code, Tier-4
  no-design); recs tile → `PosterCardV2`.

**C. Greenfield FE for an existing backend FR (P8 — never built):**

- **PH3-G1 — AI subtitle UI** (Epic 9: P1-020 terminology correction + P1-021
  MKV/Whisper→翻譯). Backend DONE, **frontend UI never existed** — build it in Activity
  (AI jobs) + detail. **Honor backend capability** (no Pause/Resume if backend lacks
  it — Rule 24; `disc-2026-06-batch-subtitle-pause`).

**D. Reserved / blocked (slot designed in Phase 3; data delivered by a backlog epic):**

- **PH3-R1 — Continue-watching** (Epic 17 P4-011, Plex/Jellyfin watch progress). Home
  v2 **reserves the slot**; live data **blocked on Epic 17 (backlog)** — Vido has no
  playback path (§4.1). Renders fail-soft-empty/hidden until a media server connects.
- **PH3-R2 — Requests landing** (Epic 13 P3-001–005). Discover reserves the request
  entry point; the full request flow is Epic 13.
- **PH3-R3 — Stats / Health landings** (Epic 16 P4-001–004 / Epic 18 P4-020–022).
  System reserves the landing; the full dashboards are Epics 16/18.

### NonFunctional Requirements

**Redesign principles (enforced via gates — `00-redesign-brief.md` §5 / DL-v2):**

- **N1** One truthful state machine (status-tint tokens + four-state standard +
  status strip render one lifecycle everywhere).
- **N2** zh-TW typography is a design material (Noto Sans TC for all CJK, JetBrains
  Mono numeric, 2-line CJK title grid, AA baked into the scale).
- **N3** Content first, tasks adjacent (sidebar groups 內容 above 任務).
- **N4** Four states or it doesn't ship (empty/loading/error/no-result up front;
  per-section fail-soft).
- **N5** Density is the user's choice (grid/list + size + customizable rows; 44px floor
  never traded away).
- **N6** Enforced, not aspirational (token-lint, contrast/touch checks, visual-
  regression CI, Rule 21 traceability).

**v2 accessibility baseline (DL-v2 §8):** WCAG AA contrast (body ≥4.5:1); 44×44px
touch targets all breakpoints; visible 2px focus-ring; Base UI keyboard/focus
management; `prefers-reduced-motion`; color never the sole state carrier.

**PRD performance carried (must hold under v2):** NFR-P1 FCP <1.5s · NFR-P2 LCP <2.5s
· NFR-P3 TTI <3.5s · NFR-P4 CLS <0.1 (**Home v2 D3 must hold these under real
content**); NFR-P6 library listing <300ms p95 (PH3-F1 field must not degrade);
NFR-P11 route transitions <200ms; NFR-P12 image lazy-load <300ms; NFR-U4 user-action
feedback <200ms.

**PRD reliability:** NFR-R2 graceful external-API failure (no crash); NFR-R4 degrade +
manual fallback — realized as the **bidirectional fail-soft (ADR F3)** for every new
aggregate endpoint (PH3-F2/F4/F5).

### Additional Requirements

**From Architecture (ADR `01-nav-ia-decision-adr.md`):**

- D1–D4 contracts: 7-destination IA; collapsible sidebar (240↔64 rail) + mobile
  bottom-4 (首頁·媒體庫·活動·下載·More) + status strip footer; Base UI primitives;
  D2 sidebar group 媒體庫▸電影/影集 + pinnable saved views; D3 hot homepage + 3 binding
  guardrails (own-above-external ordering law / home-discover boundary / dashboard-off-
  home); D4 Activity hub + status strip + Settings|System split + omnisearch.
- Routing (A1): nested routes; **route-level `beforeLoad` redirects, never 404 /
  never component-level redirect**; deep-link preservation mandatory.
- **Strangler/flag discipline:** `new_shell_enabled` single chokepoint (`__root.tsx`);
  a shell-migration story **MUST NOT modify legacy content** (P3); `LegacyContent-
  Container` pixel-fidelity (untouched routes' existing baselines must pass under v2
  shell); flag retirement = delete one read site + legacy shell + `LegacyContent-
  Container` + remove `tab-{label}` testids **in the same story** (not aliased).
- **Base UI:** wrap-once in `components/ui/`; ESLint F2 `no-restricted-imports` ban
  outside that dir; package is `@base-ui/react@1.5.0` (NOT the stale
  `@base-ui-components/react` rc.0 the ADR text named — correct ADR/project-context
  when next touched).
- Saved-view pin (C3): reuse `filter_presets` backend + add `pinned BOOLEAN` (Rule 15
  migration); general mechanism (anime = first instance, not a special case).
- Aggregate-endpoint fail-soft (B1/F3): per-section status objects; **reuse existing
  Rule 7 error prefixes — no new prefix**; frontend treats a non-`ok` section as data.
- `ScanProgress` global-overlay ownership moves explicitly INTO the v2 shell.

**From UX / Design Language v2 + pilot test-policy:**

- Every flow's `.pen` redrawn to v2 (tokens §2, type §3, four-state §7, shell §6) →
  `export-pen-screenshots.py`, update `SCREENS`, **commit only genuinely-changed PNGs**.
- Test contracts: `nav-{key}` testids; TestSprite 50-case runs flag-ON; **E2E reuse
  `tests/support/helpers/seed-helpers.ts` real seeding — NO data-dependent
  `test.skip` self-skips** (Epic 20 lesson); visual baselines per Rule 22/23, `-linux`
  via CI bootstrap PR (never generate locally — darwin machine).
- Cross-stack story-splitting: >3 tasks per side → split (BE/FE).

### FR Coverage Map

Every Phase-3 FR maps to exactly one epic (15/15, no gaps):

```
PH3-F1 N1 item-level lifecycle field   → Epic 0  ux3-foundation
PH3-F2 GET /api/v1/status/summary       → Epic 0  ux3-foundation
PH3-F3 /library/{movies,tv} clean-route → Epic 0  ux3-foundation
PH3-F4 /activity aggregate + hub        → Epic 2  ux3-activity-hub   (+ Epic 18 P4-022 slice)
PH3-F5 global omnisearch                → Epic 5  ux3-system-settings
PH3-M1 Home re-chassis                  → Epic 1  ux3-home-v2        (Epic 10 P2-001..006)
PH3-M2 Discover re-chassis              → Epic 3  ux3-discover-v2    (Epic 11 P2-010..015)
PH3-M3 Downloads v2                     → Epic 4  ux3-downloads-v2   (Epic 14 P3-010/012)
PH3-M4 System/Settings split            → Epic 5  ux3-system-settings(Epic 6 settings)
PH3-M5 Subtitle UI v2                   → Epic 6  ux3-subtitle-v2    (Epic 8 P1-016..019)
PH3-M6 Detail Epic-12 design backfill   → Epic 8  ux3-detail-backfill(Epic 12 P2-020..025)
PH3-G1 AI subtitle UI (greenfield FE)   → Epic 7  ux3-ai-subtitle    (Epic 9 P1-020/021)
PH3-R1 Continue-watching slot           → Epic 1  ux3-home-v2        (data=Epic 17 P4-011, blocked)
PH3-R2 Requests landing                 → Epic 3  ux3-discover-v2    (flow=Epic 13 P3-001..005)
PH3-R3 Stats/Health landing             → Epic 5  ux3-system-settings(dashboards=Epic 16/18)
```

NFRs (N1–N6 + v2 a11y baseline + carried PRD performance + bidirectional fail-soft F3)
are cross-cutting and apply to every epic. **Home v2 (Epic 1) is the dual headline
gate:** D3 own-above-external AND homepage performance (NFR-P1..P4) under real content.

## Epic List

> Sequenced cascade (high-risk / high-value first, dependency-aware). Each epic is
> **standalone**: it consumes only PAST epics (e.g. foundation), never requires a
> FUTURE epic to function. Activity's links to not-yet-migrated subtitle/AI surfaces
> are soft + fail-soft. Stories are authored just-in-time per the strangler model;
> this run details Epic 0 + Epic 1 only (the rest are epic skeletons until their turn).

### Epic 0: ux3-foundation — Truthful status everywhere
Every poster tells the truth (durable lifecycle badge: 整理中 / 已入庫 / 失敗 / 繁中 /
缺字幕; transient 簡轉繁·AI校正中 live in Activity, 下載中% in Epic 13/14) and the NAS
pulse is always visible (sidebar-footer status strip: disk / active-scan / queue /
service-health), with clean library URLs. Delivers the N1 data layer the rest of the
cascade renders against.
**FRs covered:** PH3-F1, PH3-F2, PH3-F3

### Epic 1: ux3-home-v2 — Open the app to your own library first (⚠ highest risk)
The homepage leads with the user's OWN content (recently-added / task-status) ABOVE the
Hero + explore blocks (D3 ordering law); the continue-watching slot is reserved (live
data deferred to Epic 17); dashboard remnants move off home. The dual headline gate:
D3 own-above-external + homepage performance under real content.
**FRs covered:** PH3-M1, PH3-R1 · consumes Epic 0 (F1)

### Epic 2: ux3-activity-hub — One place for "what is Vido doing right now" (BUILD, net-new)
A new `/activity` destination unifies parse / subtitle-batch / scan / AI jobs + a
downloads-summary row with explain-why rows; unblocks the mobile bottom-4. Standalone on
existing backend data; the v2 surfaces of subtitle/AI migrate later.
**FRs covered:** PH3-F4 · consumes Epic 0 (F1+F2)

### Epic 3: ux3-discover-v2 — The active power-filter tool
Discover's chips / presets / instant-search migrate to v2 as the active discovery tool
(grows no dashboard — D3 boundary); reserves the Requests entry point (full flow = Epic 13).
**FRs covered:** PH3-M2, PH3-R2

### Epic 4: ux3-downloads-v2 — The downloads deep page, seen and controlled
The retained downloads deep page migrates to v2 with card actions; entered from Activity.
**FRs covered:** PH3-M3

### Epic 5: ux3-system-settings — Preferences vs operations, and one search box
Split `/system` (ops: status/logs/cache/performance/backup/export) from `/settings`
(preferences) on the \*arr model; global omnisearch replaces the three search surfaces and
retires `/search`; reserves the Stats/Health landing (dashboards = Epics 16/18).
**FRs covered:** PH3-M4, PH3-F5, PH3-R3

### Epic 6: ux3-subtitle-v2 — Subtitle management in v2
Epic 8's manual + batch subtitle UI migrates to v2; the detail `管理字幕` entry + the
Activity batch surface. Soft-depends on Activity (batch summary).
**FRs covered:** PH3-M5

### Epic 7: ux3-ai-subtitle — AI subtitles finally have a UI (P8 greenfield)
Epic 9's terminology correction + Whisper translation (backend already shipped) get their
first frontend: AI-jobs in Activity + a detail trigger. Honors backend capability (no
Pause/Resume control unless the backend supports it — Rule 24).
**FRs covered:** PH3-G1

### Epic 8: ux3-detail-backfill — v2 design coverage for the detail blocks
Epic 12's dual-ratings / seasons / recs / streaming / trailer / douban blocks (shipped in
code, Tier-4 no-design) get v2 design coverage; recs tile → `PosterCardV2`.
**FRs covered:** PH3-M6

### Milestone (post-cascade): Flag retirement
Core flows migrated → `new_shell_enabled` flips default-ON → batch-delete the legacy shell
(`TabNavigation`, `AppShell`, `LegacyContentContainer`) + remove `tab-{label}` testids in
the same story (ADR D1-c — not aliased).

---

> **Story authoring scope (strangler discipline):** implementation-ready stories are
> authored just-in-time. This run details **Epic 0 + Epic 1** (the next two in the
> cascade). **Epics 2–8 carry their epic skeleton (goal + FRs + deps above) only** —
> their stories are authored by `sm create-story` when each reaches its turn, so they
> don't go stale. Each detailed story is single-dev-agent-sized and consumes only
> PAST stories.

## Epic 0: ux3-foundation — Truthful status everywhere

Every poster tells the truth and the NAS pulse is always visible, with clean library
URLs. Delivers the N1 data layer the cascade renders against. **FRs:** PH3-F1/F2/F3.

### Story 0.1: Item-level media lifecycle/subtitle field (backend)

As a NAS owner,
I want each movie/series to carry one truthful item-level lifecycle+subtitle state,
So that posters and detail pages can show the same lifecycle consistently (N1).

**Acceptance Criteria:**

**Given** a Rule-24 capability audit of what lifecycle states are derivable for a
*library item* today,
**When** the field is designed,
**Then** it surfaces only **persisted, load-time-queryable** states — `整理中`/`已入庫`/`失敗`
(from `parseStatus`) + `繁中`/`缺字幕` (from `subtitleTracks`),
**And** the **transient processing states** `簡轉繁`/`AI 校正中` are NOT stored on this field —
they live in the subtitle pipeline's ephemeral queue/SSE and are surfaced by the **Activity
hub's live feed (Epic 2+)**, never the poster badge (a steady library grid cannot truthfully
show a live process),
**And** `下載中·%` is **NOT derivable** for a library item (a library row exists only
post-download; download% belongs to requested/in-flight items → deferred to Epic 13/14),
**So** N1's poster-badge promise converges to **durable lifecycle by capability, not
omission** (pilot §4.2).

**Given** the new field,
**When** the API returns a movie/series,
**Then** the field is exposed at the boundary in camelCase (Rule 18),
**And** the repo's SELECT column list **AND** row scan are synced (Rule 15 — the exact
gap that caused bugfix-20-1).

**Given** a migration is needed,
**When** it lands,
**Then** it follows the versioned-migration convention,
**And** a repository/integration test against a **real sqlite DB** asserts the field for
seeded rows (not a mocked-repo false-green — Epic 20 lesson).

**Given** an item with no subtitle/parse data,
**When** the field is computed,
**Then** it returns a well-defined default and never errors.

### Story 0.2: Poster badge renders the lifecycle field (frontend)

As a user browsing the library,
I want the poster badge to reflect the real item lifecycle,
So that I can see each title's state at a glance.

**Acceptance Criteria:**

**Given** Story 0.1's field is exposed,
**When** `PosterCardV2` renders,
**Then** the N1 status badge derives from the new field (superseding the pilot's
`parseStatus`+`subtitleTracks` derivation) via the DL-v2 §2.5 status→token map.

**Given** an item in the steady-state (`已入庫` + `繁中`),
**When** the badge renders,
**Then** it is **suppressed** (no badge) — the badge is an **exception signal** for
attention states only (`整理中` / `失敗` / `缺字幕`), avoiding the always-on info-noise the
PosterCard density lessons warned about (Epic 10/19).

**Given** an attention state,
**When** it renders,
**Then** it uses the correct `*-tint`/`*-text` token **and** pairs the hue with a zh-TW
label (color never the sole carrier — a11y).

**Given** the strangler flag is OFF,
**When** the legacy library renders,
**Then** the legacy `PosterCard` is unaffected (the new field is additive; legacy ignores
it; legacy specs/baselines unchanged).

**Given** the PosterCardV2 unit specs,
**When** updated,
**Then** they cover each rendered state, including the suppressed steady-state.

### Story 0.3: `GET /api/v1/status/summary` aggregate endpoint (backend)

As a NAS owner,
I want one endpoint summarizing disk/scan/queue/service-health,
So that the status strip shows a real NAS pulse (D4-2; partially delivers Epic 18).

**Acceptance Criteria:**

**Given** the four ambient concerns,
**When** `GET /api/v1/status/summary` is called,
**Then** it returns disk headroom (used/total) + active-scan state + download queue count
+ per-service health as **per-section status objects** (B1/F3).

**Given** Epic 7b multi-library folders may span volumes,
**When** disk headroom is computed,
**Then** the endpoint defines exactly which volume(s) it measures (the media-library
volume(s); multiple → aggregate or the primary) so the number is never misleading,
**And** Epic 18 (P4-020/021) MUST reuse this endpoint — not build a parallel disk/health
aggregate.

**Given** a downstream source is unavailable,
**When** the endpoint composes,
**Then** that section returns `{status:"unavailable", error:{code,…}}` carrying the
downstream's ORIGINAL Rule-7 prefix,
**And** the endpoint still returns `success:true` (never fail-page; Epic 18 P4-020/021).

**Given** Rule-4 boundaries,
**When** implemented,
**Then** the handler READS existing services (disk stat / `ScannerService` / download
service / `ServiceStatusService`) and never reaches repositories directly or duplicates
subsystem logic,
**And** the response is camelCase at the boundary (Rule 18).

### Story 0.4: Status strip goes live (frontend)

As a user,
I want the sidebar-footer status strip to show real data,
So that disk/scan/queue/health are always visible.

**Acceptance Criteria:**

**Given** Story 0.3's endpoint,
**When** `SidebarFooter` renders,
**Then** it consumes `/status/summary` via TanStack Query (Rule 5), replacing the
pilot-degraded health-only state.

**Given** a non-`ok` section,
**When** the strip renders,
**Then** that section shows empty/stale (treated as data, never throws — F3 frontend
half) and the rest still render.

**Given** the collapsed 64px rail,
**When** the strip renders,
**Then** it collapses to dots only (DL-v2 §6.4); on mobile it sits at the top of the More
sheet,
**And** the active-scan pulse respects `prefers-reduced-motion`.

### Story 0.5: `/library/{movies,tv}` clean-route split (frontend)

As a user,
I want clean type URLs,
So that library links are clean and bookmarkable (D2).

**Acceptance Criteria:**

**Given** the library route,
**When** split,
**Then** `library.tsx` becomes a LAYOUT (shared toolbar/filter/sort + `<Outlet/>` + ONE
shared grid + shared filter/scroll state F5) with `library/index` (merged),
`library/movies`, `library/tv` children sharing ONE grid (no forked grid logic — P7).

**Given** old deep links,
**When** `/library?type=movie` (or `type=series`) loads,
**Then** a route-level `beforeLoad` throws `redirect()` to `/library/movies` (`/library/tv`),
`type` coerced per Rule 26, never a component redirect, never 404 (F1/P2).

**Given** the strangler flag,
**When** OFF,
**Then** the legacy library renders byte-unchanged (the split lives on the v2 path only).

**Given** the v2 grid shipped in ux2-2,
**When** the three views render,
**Then** they reuse that SAME v2 grid (no re-flow / regression),
**And** the redirects cannot loop (loading `/library/movies` never re-triggers a redirect).

**Given** E2E,
**When** the routes are tested,
**Then** they reuse `seed-helpers.ts` real seeding (no self-skip).

## Epic 1: ux3-home-v2 — Open the app to your own library first

The homepage leads with the user's own content ABOVE Hero+ExploreBlocks (D3); the
dual headline gate. **FRs:** PH3-M1, PH3-R1 · consumes Epic 0 (F1).

### Story 1.1: Home v2 design (`.pen` flow-h → v2)

As the design system,
I want the homepage redrawn to v2,
So that dev builds against a spec, not an assumption (per-flow recipe step 1).

**Acceptance Criteria:**

**Given** DL-v2 (tokens/type/four-state/shell),
**When** the home flow is redrawn in `ux-design.pen` (flow-h-homepage → v2),
**Then** it specifies own-content (recently-added + task-status) structurally ABOVE
Hero+ExploreBlocks (D3 ordering law), a reserved continue-watching slot with its
hidden/empty state (data=Epic 17), all four states, and desktop+mobile (390/768/1440).

**Given** a user whose library is stable (sparse recently-added / no active tasks),
**When** the own-content section is designed,
**Then** it specifies a graceful **sparse/empty treatment** — it collapses without leaving
a top gap, external blocks rise but stay conceptually below the own-content zone, and the
D3 ordering law still holds (own-content gets its OWN empty state, not just the page).

**Given** the regen convention,
**When** screenshots are exported,
**Then** `export-pen-screenshots.py` runs, `SCREENS` is updated if new screens exist,
and ONLY genuinely-changed PNGs are committed.

**Given** a D3 ordering-rationale annotation is needed,
**When** added,
**Then** it stands alone as its own spec frame, not crammed into a mockup.

### Story 1.2: Home v2 shell migration — own content above external

As a user,
I want to open Vido to my own library first,
So that personal content is never below promotional/discovery content (D3).

**Acceptance Criteria:**

**Given** the home route,
**When** migrated,
**Then** it opts into the v2 layout (shell-version gated via `staticData.shell:'v2'`, no
2nd flag read — F4), rendering Hero+ExploreBlocks in the v2 shell.

**Given** the D3 ordering law,
**When** home renders,
**Then** own-content (recently-added + task-status, reusing `RecentMediaPanel`) is
structurally ABOVE Hero+ExploreBlocks,
**And** a test asserts this ordering holds under real content (the ADR's primary
watch-item).

**Given** F1's lifecycle field,
**When** own-content tiles render,
**Then** their badges use the new field via `PosterCardV2`.

**Given** the project gates on browser-pixel review + visual-regression (not CI perf
budgets — P10 / N6: enforce only what we can),
**When** Home v2 is verified,
**Then** the perf gate is a **manual 390/768/1440 walk with real content** (no visible
layout-shift/jank; FCP/LCP/TTI/CLS NFR-P1..P4 as targets, not CI-enforced numeric ACs),
**And** visual-regression covers pixel stability (skeleton loading, lazy images).

**Given** flag OFF,
**When** home loads,
**Then** the legacy home is byte-unchanged.

### Story 1.3: Continue-watching reserved slot

As a user,
I want a continue-watching slot ready on home,
So that the layout is prepared for media-server sync without showing a broken block now.

**Acceptance Criteria:**

**Given** Vido has no playback path and continue-watching data = Epic 17 (P4-011),
**When** home renders today,
**Then** the slot is hidden (or shows a quiet "連接 Plex / Jellyfin 後顯示" affordance) —
never a broken/empty tile.

**Given** the slot is later populated (Epic 17),
**When** data arrives,
**Then** it sits within own-content, ABOVE external (D3 ordering preserved).

**Given** no media server,
**When** home loads,
**Then** there is no error/console noise from the absent slot (fail-soft).

### Story 1.4: Dashboard remnants leave home

As a user,
I want home to be curation-first, not a task dashboard,
So that it follows the D3 home/discover boundary.

**Acceptance Criteria:**

**Given** D3 guardrail #3,
**When** home is migrated,
**Then** `DownloadPanel` / `QBStatusIndicator` / `ConnectionHistoryPanel` are REMOVED
from home.

**Given** their data must stay reachable (no future-epic dependency),
**When** removed,
**Then** the ambient QB/connection info is carried by the Epic-0 status strip (already
live) **and** in-flight downloads remain reachable at the existing `/downloads` page
(DownloadPanel's eventual Activity home is Epic 2 — not required for 1.4 to function).

**Given** Activity (Epic 2) does not exist yet,
**When** DownloadPanel is removed from home,
**Then** the **temporary loss of the at-a-glance home download view is explicitly
acknowledged** (not a silent gap — Rule 24) — mitigated by the status-strip queue count +
`/downloads`, and closed when Epic 2 ships (sequenced immediately after Epic 1).

**Given** the strangler discipline (P3),
**When** 1.4 edits home,
**Then** it does NOT modify legacy route content elsewhere ("while I'm here" edits banned).

**Given** flag OFF,
**When** home loads,
**Then** the legacy home (with its dashboard elements) is unchanged.

## Epics 2–8: skeletons (stories authored at cascade turn)

Goal + FR coverage + dependencies are in the **Epic List** above. Detailed stories for
`ux3-activity-hub`, `ux3-discover-v2`, `ux3-downloads-v2`, `ux3-system-settings`,
`ux3-subtitle-v2`, `ux3-ai-subtitle`, `ux3-detail-backfill` are authored by
`sm create-story` when each reaches its turn (strangler — avoids stale artifacts).
