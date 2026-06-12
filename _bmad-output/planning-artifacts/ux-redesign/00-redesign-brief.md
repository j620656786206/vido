# Vido UX Redesign Brief — Phase 0: Discovery & North Star

**Date:** 2026-06-12
**Author:** Mary (BMAD Analyst) with Alexyu
**Status:** Phase 0 output — evidence base + design principles + open IA decisions. **No screens were drawn, no code or `.pen` changed.**
**Next phase:** Phase 1 (Design Language v2 + Nav-IA decision ADR) — see `README.md` in this folder.

---

## 1. Purpose & Method

After ~3 years of epic-by-epic delivery, Vido's UI accreted layer-by-layer: each epic was locally sensible, but the app lacks one coherent design language, and the navigation chassis built in Epic 5 is straining under surfaces it was never designed to hold. This brief is the **north star for a phased, strangler-fig redesign** (not a big-bang rewrite). It synthesizes four evidence streams:

| Stream | Source | Method |
|---|---|---|
| Design audit | `pen-review-2026-06-12.md` (this folder) — 40-screen `.pen` review by 4 parallel agents | Cross-validated systemic root causes, Tier 1–4 triage |
| History mining | All 12 epic retros, ~19 retro action items, `sprint-status.yaml` (~184 tracked entries), bugfix artifacts | Pain-point clustering with file-level citations |
| Surface mapping | All epic files (incl. 8 archived v3 epics), `apps/web/src/routes/` ground truth | Full app inventory + IA stress points |
| Competitive scan | Plex / Jellyfin / Emby / Sonarr·Radarr / Overseerr·Jellyseerr, 2025–2026 web research | Convergent patterns, divergent decisions, differentiation gaps |

**Scale facts** (verified, not estimated): 6 completed v3 epics + 12 v4 epics + 5 amendment/infra epics (0, 7b, 9c, 9-T, 19) + 8 archived v3 epics; ~110 numbered stories plus ~74 discovery/bugfix/retro entries in `sprint-status.yaml`; 22 route files rendering UI across 8 top-level routes, 1 detail route, 11 settings children, 2 dev routes. Epics 13, 15, 16, 17, 18 are still backlog and **will demand new top-level IA slots**.

---

## 2. Product Context (what must not be lost in the redesign)

From `ux-design-specification.md` (2026-01-11), still valid and re-confirmed by this audit:

- **User:** Alex, 32, software engineer, Synology NAS, 500+ movies / 200+ shows, 70% Asian content, daily use, desktop-primary (27"), mobile for monitoring. Single power user — v4 explicitly dropped auth/multi-user (`project-context.md` §3).
- **Core philosophy:** 「自動化但可見、智能但可控」 (automated but visible, smart but controllable).
- **Dual loops:** Discovery (search → download → auto-parse → library) feeds Appreciation (browse → perfect zh-TW posters → watch). The most frequent action is opening the library; the most satisfying moment is seeing perfectly parsed Traditional Chinese metadata.
- **Differentiators:** AI fansub-filename parsing; zh-TW-first metadata (TMDB + Douban + Wikipedia fallback); subtitle engine with 簡繁 detection + OpenCC conversion + AI correction/translation.

The redesign must amplify these — it is a re-skinning *and* re-chassising of a product whose value proposition is already proven, not a product pivot.

---

## 3. Evidence-Backed Pain Point Inventory

### 3.1 Systemic design-language root causes (from the 40-screen `.pen` review)

Six root causes, each independently confirmed by ≥2 of 4 review agents (`pen-review-2026-06-12.md` Part 1):

| # | Root cause | Blast radius |
|---|---|---|
| R1 | **CJK text set in DM Sans / Inter** — no CJK glyphs, uncontrolled fallback. Component-level (`ButtonPrimary`/`ButtonSecondary`/`FilterChip`) so it leaks into every flow | All flows A–J |
| R2 | **Content clipped to invisibility** — collapsed sheets, cut buttons, overflowing panels (C3-M, A6-M, B4-D, B8-D/M, E2-M…) | A/B/C/E, HIGH severity |
| R3 | **Hardcoded semantic colors / token drift** — status tints, accent tints, fallback gradients duplicated as literals 30+ places | B/D/E/F/G |
| R4 | **Touch targets < 44px** across all mobile screens | All `-M` screens |
| R5 | **`--text-muted` (#808080) fails WCAG AA** at 11–12px (≈4.0:1 on bg-primary, 3.3:1 on bg-secondary) | App-wide, incl. shipped CSS |
| R6 | **Halfwidth punctuation / mainland-CN wording** in a zh-TW-first product (backlog `chore-zhtw-error-string-punctuation-sweep`) | App-wide |

Plus the two structural findings the review escalated beyond "fix": **navigation mixes content and tasks in one layer**, and **states (empty/loading/error/no-result) are pervasively missing** with Epic 12 detail features having no design coverage at all (Tier 4).

### 3.2 Recurring UX failure modes (from 12 epic retros + sprint history)

Clusters ranked by severity × recurrence; every claim cites its artifact (paths relative to `_bmad-output/implementation-artifacts/` unless noted).

**P1 — Design–implementation drift (HIGH; Epics 2/5/7/10/19).** Stories 5-2..5-7 all deviated from design; 2 remediation stories = ~18% of epic effort (`epic-5-retro-2026-03-15.md`). Sally's element review: 42 match / 23 minor / 8 mismatch (`ux-design-review-epic5.md`). Detail page shipped **without its primary CTAs** (播放/加入清單 buttons absent; `5-10-ux-design-compliance-phase5.md`). `HoverPreviewCard` was independently invented and diverged from `.pen` `Component/PosterCardHover` undetected for months (`bugfix-10-4-hover-comparison.md`, `epic-19-retro-2026-05-29.md`). Epic 19's 131-component sweep then proved drift was *not* systemic (0 material / 2 minor) — but it took a 13-story epic plus ESLint + visual-regression CI to know. **Lesson: the guardrail infrastructure now exists (Rules 21/22/23) — the redesign must ride it, not bypass it.**

**P2 — Navigation pieces forgotten (HIGH; 4 distinct incidents).** Epic 5 originally had *no app-shell story at all* — "no header, no tabs, no dark theme" discovered only when Alexyu ran the dev server; 5-0 was retrofitted mid-sprint (`epic-5-retro-2026-03-15.md`). Nav tab shipped pointing at a nonexistent `/pending` route (`bugfix-2-pending-page-not-found.md`). `/discover` shipped fully functional with **no nav entry** for a week+ (`sprint-status.yaml` `disc-nav-entry-discover-route`). Settings menu rendered dead, unlabeled items for unbuilt pages (`bugfix-6-search-enhancement-page-labels.md`). **Lesson: navigation has never been owned as a designed system — it accretes.**

**P3 — Empty/loading/error/fallback states are afterthoughts (HIGH; 6+ incidents).** Misleading "請連接 qBittorrent" empty state when the real problem was no media folder; the replacement 3-state classifier itself shipped broken (`bugfix-10-5-empty-library-onboarding.md`). Bare "尚未設定媒體資料夾" with zero guidance for first-time Docker users (`bugfix-7-media-directory-setup-guide.md`). Metadata-less media rendered "a blank or overly technical page" until a dedicated fallback story (`5-11-fallback-ui-enhancement.md`). Backup page surfaced raw JSON errors/500s (`bugfix-3`, `bugfix-8`). Matches `.pen` review Tier 4: states are missing in flows A/D/E/F/G too. **The pattern: happy-path-first design, states retrofitted via bugfix sprints.**

**P4 — Accessibility ships broken until a gate forces it (HIGH; Epics 10/11).** Modal with `aria-modal` but no focus management (10-2); bottom sheet `role="dialog"` with no keyboard affordances (11-2) — "a repeat of the exact class Epic 10's AI-1 was meant to prevent" (`epic-11-retro-2026-06-09.md`, its **#1 action item**). Enabling `jsx-a11y` surfaced a 146-warning batch, 56 real violations (`sprint-status.yaml` retro-11-AI1b). Mouse-only dismiss backdrops now duplicated across 7 dialogs (backlog `disc-2026-06-shared-dialog-dismiss-layer`). **The project's own conclusion: "enforcement mechanism predicts stickiness" — passive guidance fails here.**

**P5 — Mobile is consistently second-class (MED-HIGH; Epics 5/10/11).** Mobile sort/filter bottom sheets designed but never built (`ux-design-review-epic5.md` XS-4); 3–5MB original-size backdrops shipped to mobile (10-2 H1); "mobile pixel-verify remains deferred to NAS" recurs verbatim across bugfix-10-2/10-5/10-6/10-7 closeouts and **has never been closed** (retro-10-CP1). Plus `.pen` R4: touch targets <44px on every mobile screen.

**P6 — zh-TW quality erosion in a zh-TW-first product (MED; Epics 5/10/12).** Simplified 「元數據」 in shipped UI; mixed-language menu items; same concept labeled 「加入日期」 vs 「新增日期」 (`5-10-ux-design-compliance-phase5.md`, `ux-design-review-epic5.md`); halfwidth commas in detail-section error strings (backlog `chore-zhtw-error-string-punctuation-sweep`); emoji icons replaced after user feedback 「AI感太重」 (bugfix-10-6/10-7). Matches `.pen` R1/R6. **The core differentiator has no systemic typography/terminology enforcement.**

**P7 — The Library/Detail/PosterCard surface is the remediation black hole (MED-HIGH; Epics 2→5→10→19).** One component family absorbed four epics of rework: card-click 404s (`bugfix-10-1`, `bugfix-1`), hover rebuild against `.pen` incl. Chromium GPU workaround (`bugfix-10-4`), info-density + layout-shift fixes (`bugfix-10-7`), grid stuck at 2 columns from a stray Tailwind `container` class (`bugfix-4-library-grid-layout.md`). This is also the **highest-traffic surface** (the daily entry point) — which is why Phase 2 pilots Browse + Detail.

**P8 — Whole journeys shipped backend-only (MED-HIGH; Epics 8/9).** Batch-subtitle backend done 2026-03-25; **zero web consumer existed until 2026-06-09** — found by TestSprite coverage analysis, not planning (`disc-2026-06-batch-subtitle-frontend-ui`). AI subtitle enhancement (terminology correction, Whisper transcription, AI translation): backend done, 6 design screens done, **no frontend implementation story exists anywhere — the features are unreachable from the UI today** (`sprint-status.yaml` Epic 9; `9-UX-ai-subtitle-enhancement-design.md`). Design G4 promises Pause/Resume the backend doesn't support (`disc-2026-06-batch-subtitle-pause`). **The redesign's flow inventory must treat "reachable from the UI" as the bar, not "merged".**

**P9 — Search IA confusion (MED; Epics 5/11).** Two search bars (global header → `/search` TMDB search; library-local → library filter) confused scope expectations; placeholder text was the only mitigation (`ux-design-review-epic5.md` 5-3-01, `bugfix-6`). Deep links silently dropped lone-numeric filters until Rule 26 (`epic-11-retro-2026-06-09.md` Pattern #2). `/search`, `/discover`, and library-local search now coexist as three search-ish surfaces.

**P10 — "Tests green ≠ feature works" (HIGH; process).** A single post-deploy browser audit surfaced 11 issues / 7 stories (Bug Fix Sprint 2026-03-28); Epic 10 shipped with "1738/1738 PASS" but had never been smoke-tested locally (`epic-10-retro-2026-04-20.md`). **Redesign implication: Phase 2's go/no-go gate must include browser-pixel verification at 390/768/1440, not test-suite green.**

### 3.3 Screen/flow hotspot ranking (where redesign effort pays off most)

| Rank | Surface | Evidence density | Notes |
|---|---|---|---|
| 1 | Library browse (grid/list/sort/filter/search) | 31 review findings + 5 bugfixes + empty-state failures | Daily entry point; pilot candidate ✓ |
| 2 | Media detail page/panel | Missing CTAs, no fallback, Epic-12 features (trailers/providers/recs/Douban) have **no design**; 460px panel IA needs reflow (`pen-review` Part 3) | Pilot candidate ✓ |
| 3 | PosterCard + hover | 4 epics of churn | The atom of surfaces 1, 2, and homepage |
| 4 | Homepage TV wall | Skeleton flicker, mobile payloads, modal a11y, identity ambiguity (see §5 D3) | |
| 5 | Subtitle dialogs (manual/batch/AI) | 2.5-month UI gap, AI UI never built, design-backend contract drift | Differentiator surface, underdesigned |
| 6 | Settings/Scanner/Setup | Dead menu items, bare empty states, toast drift | Also where epics 16/18 want to land |
| 7 | Downloads | Console bursts, polling storms, no card actions in design (`pen-review` Part 4 D1) | Epic 14 v2 pending |
| 8 | Dialogs/modals app-wide | 146-warning a11y batch; 7× duplicated dismiss layer | Cross-cutting component debt |

---

## 4. Competitive Insights (2025–2026 scan)

### 4.1 Table stakes (everyone converges — Vido must simply meet these)

Dark theme; poster-grid library with header toolbar for sort/filter; backdrop-hero detail page with metadata block → action row → cast → related; home as horizontal rows distinct from exhaustive library; status badges on poster cards; admin/ops segregated from consumption; trailers + watch providers + recommendations on detail (Vido already ships these via Epic 12).

### 4.2 Divergent choices = the real decisions (full citations in research stream)

1. **Sidebar vs top nav.** Classic Plex, Jellyfin, Emby, *arr, and seerr all use a left sidebar. Plex's 2025–26 move to horizontal top nav triggered the loudest UX backlash in the category (core actions went "from ~2 clicks to ~6"; Fire TV redesign called "unusable"). Evidence favors persistent sidebar for power-user tools.
2. **Home = personal library vs storefront.** Jellyfin/Emby derive home purely from the user's own data; new Plex injects promotional content above user libraries and users revolted ("scroll past ad-supported channels… to find a movie you already ripped"). For a single power user, home must answer *what's mine, what's new, what's in flight* — curation is welcome only when sourced from the user's own data.
3. **Movies/TV first-class vs library-instances.** *arr hard-splits into two apps; Plex/Jellyfin/Emby use N user-defined libraries (an abstraction serving multi-user generality Vido doesn't need); seerr unifies discovery. Vido's fixed two-type domain (+ Epic 7b multi-library folders) permits first-class 電影/影集 destinations.
4. **Pagination vs continuous scroll.** Jellyfin paginates and users file requests to remove it; continuous/virtualized scroll is the expected behavior.
5. **Technical info foreground vs buried.** *arr makes file/quality data the whole page; Plex/Jellyfin bury it in a modal. **Nobody does the hybrid well** — a consumption-grade page that still gives codec/file-path/subtitle-track truth at a glance. Vido's tech badges (Epic 9c) are already ahead here.
6. **Where acquisition state lives.** *arr: Activity/Queue; seerr: card badges + request list; Plex/Jellyfin: nowhere. **Nobody shows library + downloads + subtitle/scan tasks in one coherent activity surface.**

### 4.3 Vido differentiation opportunities

1. **Own the whole state machine.** Overseerr's most-reported failures (requests stuck in Processing, phantom Available, missing download progress) are structural: it mirrors three other systems' state. Vido owns request → qBittorrent → scanner → library → subtitles in one backend and can render a single truthful lifecycle on every poster and detail page (想要 → 下載中 x% → 整理中 → 已入庫 → 字幕狀態). **No competitor can structurally match this.**
2. **zh-TW-first is unserved.** All five competitors are English-first; CJK typography, mixed-script titles, zh-TW punctuation are retrofits at best. Designing the grid/type system *for* 2-line CJK titles is a moat, not a chore.
3. **Subtitle-centric UX is greenfield.** None of the five surfaces subtitle state at all (Bazarr is a bare admin table). Subtitle status as a first-class dimension — on poster badges, detail pages, activity feed (有繁中 / 簡轉繁 / AI 校正中 / 缺字幕) — defines the category for this user.
4. **Collapse the five-sidebar stack.** A solo NAS user otherwise runs Jellyfin + Sonarr + Radarr + Jellyseerr + Bazarr. Vido can merge consumption, acquisition, and operations into one IA: content destinations on top, task destinations below — *arr's task sidebar married to Jellyfin's content sidebar.
5. **NAS-aware ambient status.** Storage stats only reached Jellyfin's dashboard in late 2025; *arr buries them in System. Disk headroom, active scan, download throughput, service health dots are ambient concerns for a NAS owner — a compact always-visible status strip would out-serve every competitor. Copy *arr's "health warning links to remediation" pattern.
6. **The Plex backlash as hard constraints:** personal library never below promotional content; never remove list view or density options; core actions ≤2 clicks; labeled icons over ambiguous glyphs; user-customizable home rows (pin/hide/reorder).
7. **Single best patterns to steal:** seerr's on-card availability badges (extend to subtitle state); *arr's calendar (即將播出) and explain-why queue rows; Emby's one-stop metadata manager (for zh-TW title/poster fixes); classic Plex's per-library Recommended tab built solely from one's own collection.

---

## 5. Design Principles — North Star v2

Six principles, each grounded in evidence above. These extend (not replace) the seven experience principles in `ux-design-specification.md`; where they overlap, these are the redesign-era sharpening.

**N1 · One truthful state machine（一個真實的狀態機）**
Every media item displays its full lifecycle — discovery, request, download %, organizing, in-library, subtitle state — from one owned backend, consistently on poster badge, detail page, and activity surfaces. *Evidence: §4.3-1 (competitors structurally can't); P8 (journeys invisible to users); seerr's sync wounds.*

**N2 · zh-TW typography is a design material, not a localization pass（繁中排版是設計材料）**
Noto Sans TC everywhere CJK appears; JetBrains Mono for technical values; fullwidth punctuation; line lengths and grid cells designed for 2-line CJK titles; AA contrast minimums baked into the token scale. *Evidence: R1/R5/R6, P6; §4.3-2 (unserved moat).*

**N3 · Content first, tasks adjacent（內容歸內容，任務歸任務）**
Navigation separates appreciation (browse/detail/discover) from operations (downloads, parsing, subtitles, scanning, health). The personal library is never demoted below promotional or discovery content. *Evidence: `.pen` review structural finding; P2; D-decisions below; Plex backlash.*

**N4 · Four states or it doesn't ship（四態齊備才算完成）**
Every screen designs empty / loading / error / no-result up front; external-data sections fail soft per-section (Rule 27 already mandates the backend half). A screen design without its states is an incomplete deliverable in Phase 1+. *Evidence: P3; `.pen` Tier 4; design-context-pack §4-3/4.*

**N5 · Density is the user's choice（密度由使用者決定）**
Grid/list view modes, poster size options, customizable home rows — never remove them. Desktop depth (27" information density), mobile speed (monitoring + quick decisions), 44px touch floor. *Evidence: P5, R4; Plex backlash constraint; UX-spec principle 5 carried forward.*

**N6 · Enforced, not aspirational（用機制守住，不靠自覺）**
Design language v2 ships as tokens + automated gates: token-lint against hardcoded colors, contrast checks, touch-target checks, the existing visual-regression CI and Rule 21/22/23 traceability. The project's own retros prove passive guidance fails. *Evidence: P1/P4; epic-11 retro "enforcement mechanism predicts stickiness"; Epic 19 infra already built — reuse it.*

---

## 6. Key IA Decisions for Phase 1 (framed, NOT decided)

These four decisions shape everything downstream. Phase 0 lays out options + trade-off evidence; **Phase 1 owns the ADR** (`01-nav-ia-decision-adr.md`).

### D1 — Navigation chassis: sidebar vs top tabs (vs hybrid)

**Current state:** top header (logo + global search + settings gear) + 5 horizontal tabs: 媒體庫/探索/下載中/待解析/設定 (`apps/web/src/components/shell/TabNavigation.tsx:12-18`). Home (`/`) is reachable **only via the logo**; Settings has two entry points (tab + gear); `/search` is reachable only by submitting the global search bar (`InstantSearchBar.tsx:83`), never as a destination.

| Option | For | Against |
|---|---|---|
| **A. Persistent left sidebar** (collapsible; drawer/bottom-bar on mobile) | Industry consensus for power tools (all 5 competitors' classic UIs); scales to backlog epics 13/16/18 without crowding; natural place for §4.3-5 ambient status strip + content/task grouping (N3); Plex's top-nav move backfired publicly | Costs ~200–240px horizontal space vs current full-width content; mobile needs a separate pattern (drawer or bottom tabs); biggest single layout migration from today |
| **B. Keep top tabs, fix the layer-mixing** (group content left / tasks right, add Home) | Smallest migration; preserves full-width grids | Hard ceiling: with Requests/Stats/Health/Calendar incoming, 8–10 tabs won't fit; no home for ambient status; horizontal nav scales worst exactly where Vido is growing |
| **C. Hybrid:** top bar for content destinations + a unified "活動/Activity" hub for all tasks (downloads, parsing, subtitle jobs, scans) | Keeps top-nav familiarity; consolidates 4 task surfaces into 1 slot; matches *arr's Activity concept | Activity hub hides in-flight state behind a click (tension with N1's ambient visibility unless paired with a status strip) |

**Evidence to weigh:** P2 (nav was never owned); agent finding that `/pending` — a task queue — sits beside content at top level; backlog epics need ≥3 new slots; mobile bottom-tab conventions.

### D2 — Are 電影/影集 first-class destinations, or views within 媒體庫?

**Current state:** single `/library` with type filtering inside; Epic 7b created N user-defined libraries each typed movie|series.

| Option | For | Against |
|---|---|---|
| **A. Unified 媒體庫 + type tabs/filters** (status quo, refined) | One grid, one mental model; cross-type browse ("everything new this week"); fewer nav slots | Type filter is the single most-used pivot and stays buried one level down; mixed-type sort/filter semantics (seasons vs runtime) complicate every toolbar |
| **B. 電影 / 影集 as top-level destinations** | Matches the fixed two-type domain (§4.2-3); each gets type-appropriate toolbars, columns, calendar hooks (影集 ties to *arr-style 即將播出); cleaner deep links | +1 nav slot; "all media" view needs a third home or disappears; anime-as-a-pseudo-type question resurfaces (user is 70% Asian content — 動畫 tab?) |
| **C. Library-instance model** (Epic 7b libraries become destinations, à la Plex/Jellyfin) | Mirrors the actual data model; multi-folder users see their own organization | The abstraction exists to serve generality Vido doesn't need (single user); competitors' library-instance UX adds the "More" overflow problem; Phase 2 reserve fields (auto_detect) suggest the model may evolve |

**Evidence to weigh:** Epic 7b data model; §4.2-3; UX-spec deep-linking patterns (actor → filter library) work best against a unified index.

### D3 — Homepage (curation) vs 媒體庫 (exhaustive) vs 探索 (external): who owns what?

**Current state:** the homepage's identity has flipped twice — Epic 4 defined it as a *task dashboard* (download list + recently added), Epic 10 rebuilt it as a *discovery platform* ("transforms Vido from a management tool into a discovery platform"). `/discover` then shipped as a second discovery surface (Epic 11), overlapping home's explore blocks. Three surfaces now compete for "find something".

| Option | For | Against |
|---|---|---|
| **A. Home = my library's pulse** (continue watching*, recently added, in-flight tasks, subtitle queue) **; 探索 = external discovery** (TMDB trending, requests); 媒體庫 = exhaustive grid | Cleanest answer to "personal first" (Plex-backlash constraint); home becomes the N1 lifecycle dashboard; 探索 absorbs Epic 13 requests naturally | Demotes Epic 10's hero/explore-block investment from home; "continue watching" awaits Epic 17 |
| **B. Home = curated mix of own + external** (status quo Epic 10 TV wall, with availability badges) | Preserves Epic 10 work; one landing page for both loops (discovery + appreciation) | The exact pattern users punished Plex for *when external content wins*; blurs with `/discover`; harder to keep N3 clean |
| **C. Merge 探索 into Home** (one discovery surface, library stays exhaustive) | Removes the three-way overlap (P9); fewer nav slots | Loses the dedicated filter-chip power surface Epic 11 built; long home page |

**Evidence to weigh:** homepage identity flip-flop (agent finding); P9 search-surface confusion; Epic 13 (requests) and Epic 17 (繼續觀看) both want a discovery/home anchor; §4.2-2.

### D4 — Where do operations live? (the task half of N3)

Sub-decisions Phase 1 must also settle, with lighter stakes:

- **Unified Activity hub** (下載中 + 待解析 + 字幕批次 + 掃描進度 + AI jobs as one queue with explain-why rows, *arr-style) vs today's separate top-level 下載中/待解析 + buried scan/subtitle progress. Evidence: P8 (invisible journeys), §4.2-6 (nobody unifies this — opportunity), N1.
- **Ambient status strip** (disk headroom, active scan, queue count, service health dots) — always-visible vs dashboard-only. Evidence: §4.3-5; Epic 18 (health) and Epic 16 (stats) both partially land here instead of demanding new pages.
- **Settings split:** user preferences vs ops dashboards (status/logs/cache/backup currently 11 children under `/settings`). Epics 16/18 will push more ops surface in; *arr separates Settings from System. Evidence: agent surface map; `bugfix-6` dead menu items.
- **`/pending` (待解析) demotion** from top-level tab into the Activity hub or a library filter chip. Evidence: agent finding (task queue promoted beside content).
- **Search consolidation:** global search vs `/search` vs `/discover` instant search (P9) — likely one omnisearch with scoped results, but Phase 1 decides.

---

## 7. Constraints & Enablers for Phases 1–3

**Constraints**
- **Strangler migration only** — flows convert one at a time behind flags; Phase 2 pilots Browse (A) + Detail (B) with a go/no-go gate (`README.md` this folder).
- Dark theme only; Tailwind CSS v4; TanStack Router; desktop-primary responsive web (`design-context-pack.md` §2).
- Design tokens' ground truth is `apps/web/src/styles.css`; `.pen` ↔ CSS must stay in sync (R3 remediation adds new semantic tokens — `accent-subtle`, `*-tint`, `text-on-accent`, `overlay-scrim` etc., `pen-review` Tier 2A).
- `.pen` canvas follows the A–J merged-block convention; screenshot regen is non-deterministic — commit only genuinely-changed PNGs (`CLAUDE.md`).
- Solo developer + AI agents; BMAD story pipeline; every retro action item becomes a sprint-status entry (no prose-only debts — Rule 24).

**Enablers already in place (reuse, don't rebuild)**
- Rule 21 component↔design traceability + ESLint enforcement; Rule 22 retro drift audits; Rule 23 time-stable visual fixtures; visual-regression CI with `-linux` baseline bootstrap (Epic 19).
- Rule 27 Five Pillars for external integrations (rate-limit/cache/degrade/error-codes/keys) — the backend contract N4's fail-soft states render against.
- `jsx-a11y` lint now enabled (retro-11-AI1b cleared the warning batch) — extend, don't re-litigate.
- TestSprite journey harness (58 plans) for Phase 2 validation; `/test/gallery` visual fixture route.

**Risks to manage**
- **Scope gravity:** the hotspot list invites fixing everything; Phases hold the line — Phase 1 is *language + IA only*, Phase 2 is *two flows only*.
- **Token migration breakage:** R3's token additions touch `styles.css` consumed by all shipped flows; needs the visual-regression net before any sweep.
- **Design-backend contract drift** (P8's G4 pause case): every Phase 2+ screen must verify backend capability per Rule 20/24 before promising controls.
- **Mobile verification debt** (P5): the perpetually-deferred 390px sweep must be a Phase 2 gate criterion, or the redesign re-ships the same debt.

---

## 8. Definition of Done — handoff to Phase 1

This brief delivers the four Phase-0 commitments:

1. ✅ **Pain-point inventory with evidence links** — §3 (6 systemic root causes, 10 recurring failure modes, hotspot ranking).
2. ✅ **Competitive insights** — §4 (table stakes, 6 divergent decisions, 7 differentiation opportunities, Plex-backlash constraints).
3. ✅ **Design principles** — §5 (N1–N6, each evidence-grounded).
4. ✅ **Open IA decision framework** — §6 (D1–D4 with options and trade-off evidence, deliberately undecided).

**Phase 1 must produce:** `01-design-language-v2.md` (token scale incl. new semantic tokens, CJK type system, state-coverage standard, enforcement gates) and `01-nav-ia-decision-adr.md` (D1–D4 resolved with rationale), then hand to Phase 2's Browse+Detail pilot.

**Open questions Phase 1 should put to Alexyu** (genuine product calls, not derivable from evidence):
- D2's anime question: is 動畫 a first-class destination for a 70%-Asian-content library, or a genre filter?
- D3: how much of Epic 10's hero/explore-block investment is emotionally non-negotiable vs movable into 探索?
- Appetite for the ambient status strip (D4) vs keeping chrome minimal?
- Should archived ambitions (multi-user v5, watch history via Epic 17) reserve IA slots now, or be ignored until real?

---

*Sources: `pen-review-2026-06-12.md` · 12 epic retros + 19 retro-AI artifacts + `sprint-status.yaml` (mined 2026-06-12) · `_bmad-output/planning-artifacts/epics/*` incl. archive · `apps/web/src/routes/` + `components/shell/` ground truth · `ux-design-specification.md` (2026-01-11) · `ux-design-gap-analysis-v4.md` (2026-03-23) · `design-context-pack.md` · 2025–2026 web research on Plex/Jellyfin/Emby/Sonarr/Radarr/Overseerr (citations inline in §4 research stream report).*
