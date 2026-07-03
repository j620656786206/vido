---
status: 'complete'
phase: 'UX Redesign Phase 1b — Design Language v2'
author: 'Sally (BMAD UX Designer) with Alexyu'
date: '2026-06-13'
pairs_with: '01-nav-ia-decision-adr.md (Phase 1a) — via the `→ DL-v2` seam protocol'
inputDocuments:
  - _bmad-output/planning-artifacts/ux-redesign/00-redesign-brief.md
  - _bmad-output/planning-artifacts/ux-redesign/01-nav-ia-decision-adr.md
  - _bmad-output/design-context-pack.md
  - _bmad-output/planning-artifacts/ux-design-specification.md
  - apps/web/src/styles.css
  - ux-design.pen (Design System Reference, Component Library, A′ Browse Redesign)
landed_in_pen: 'design-system flow — Design Language v2 + Navigation Shell v2 reference frames + shell components'
---

# Vido Design Language v2 — Phase 1b

> **What this document is.** The reusable visual foundation ("畫面之母") for the
> phased Vido redesign. It owns **visual tokens, typography, components, state
> patterns, and accessibility** — the half of Phase 1 that the Nav/IA ADR
> (`01-nav-ia-decision-adr.md`) defers to via every `→ DL-v2` marker. The ADR owns
> **behavior/structure** (routes, flag, test-contract cutover); this document owns
> **how it looks and what passes**. Together they are the two Phase-1 foundations
> that Phase 2 (Browse + Detail pilot, behind `new_shell_enabled`) applies directly.
>
> **Ground truth.** Token values are authored in `apps/web/src/styles.css` and
> mirrored as `.pen` variables. This document is the **specification**; where it
> states a *corrected* or *new* value, that value is the target the code adopts —
> see §10 for what lands now vs. what is a tracked Phase-2 migration.

---

## 1. Principles this language enforces

Design language v2 is the mechanical expression of the brief's N1–N6 principles
(`00-redesign-brief.md` §5). Each token, type rule, and state standard below traces
to one:

| Principle | How v2 enforces it |
|---|---|
| **N1** One truthful state machine | Status-tint token family + the four-state standard (§7) + status strip (§6.4) render one lifecycle everywhere |
| **N2** zh-TW typography is a design material | The type system (§3): Noto Sans TC for all CJK, AA contrast baked into the text scale (§2.2), 2-line CJK title grid |
| **N3** Content first, tasks adjacent | The navigation shell (§6) separates content destinations from the task/ops group; visual weight follows |
| **N4** Four states or it doesn't ship | The state-pattern standard (§7) — empty / loading / error / no-result are token-defined components, not afterthoughts |
| **N5** Density is the user's choice | Spacing scale + density rules (§4); 44px touch floor never trades away view-mode/size options |
| **N6** Enforced, not aspirational | Token-only color (no literals), the enforcement gates (§9), and the `.pen` ↔ CSS single source of truth |

---

## 2. Token System v2

### 2.1 What changes and why

The 40-screen pen review found three token-level root causes (`00-redesign-brief.md`
§3.1): **R3** hardcoded semantic colors duplicated 30+ places, and **R5**
`--text-muted` failing WCAG AA. v2 fixes both by (a) **adding the missing semantic
tokens** so literals have a home, and (b) **correcting the text scale** so every
text token passes AA.

**No forking.** `apps/web/src/styles.css` stays the single token file (ADR Foundation
Assessment). v2 *adds* tokens and *corrects* values; it never forks a second palette.

### 2.2 Text color scale (R5 fix)

Measured WCAG contrast ratios (text-on-surface), three dark surfaces:

| Token | Value | bg-primary `#1B2336` | bg-secondary `#24304A` | bg-tertiary `#2E3B56` | Verdict |
|---|---|---|---|---|---|
| `text-primary` | `#F2F2F2` | 14.00 | 11.75 | 10.00 | AA+++ all sizes |
| `text-secondary` | `#B3B3B3` | 7.47 | 6.27 | 5.34 | **AA all sizes** |
| `text-muted` **(was `#808080`)** | **→ `#A0AABE`** | 6.71 | 5.63 | 4.79 | **AA all sizes** (was 3.97 / 3.33 / 2.83 — failed) |
| `text-disabled` **(new)** | `#6E7891` | 3.55 | 2.98 | 2.54 | **Intentionally sub-AA** — disabled/decorative only, never load-bearing text |

**Decision (Alexyu, 2026-06-13): "both" remediation for R5 —**

1. **Correct the value.** `--text-muted` becomes `#A0AABE` (passes AA ≥4.5:1 on all
   three surfaces at any size). This is the single value change in v2.
2. **Add a usage rule + a relief token.** `text-muted` is for **secondary metadata
   at ≥14px**. Essential small text (captions, counts, helper text **<14px** that the
   user must read) uses `text-secondary`. Genuinely de-emphasized, non-essential, or
   disabled content uses the **new `text-disabled`** — which is *deliberately* below
   AA and therefore must never carry information the user needs.

> **Rule TC-1 (token-lint, N6):** no UI text uses `text-disabled` to render
> information the user must read. `text-disabled` is for disabled-control labels,
> placeholder-of-placeholder, and decorative-only strings.

### 2.3 Colored text on dark (the second AA trap)

Semantic hues are tuned for **fills, icons, and large/bold text** — several **fail AA
as body-size text**. Measured:

| Hue | As fill / large text | As body text (<18px) | Body-text-safe token |
|---|---|---|---|
| `accent-primary` `#3B82F6` | 4.26 / 3.58 / 3.04 — fills + ≥large only | **fails** | `accent-text` `#60A5FA` (6.16 / 5.17 / 4.40) |
| `error` `#EF4444` | 4.16 / 3.50 / 2.97 — fills + ≥large only | **fails** | `error-text` `#F87171` (5.66 / 4.75 / 4.05) |
| `success` `#22C55E` | 6.88 / 5.77 / 4.91 | AA | `success` (reuse) |
| `warning` `#F59E0B` | 7.30 / 6.12 / 5.21 | AA | `warning` (reuse) |
| `info` `#06B6D4` | 6.45 / 5.42 / 4.61 | AA | `info` (reuse) |

> **Rule TC-2:** accent-colored or error-colored **body text** uses `accent-text` /
> `error-text`. The base `accent-primary` / `error` hues are for fills, borders,
> icons, badges, and ≥18px / bold headings. On `bg-tertiary`, colored body text is
> marginal (≈4.0–4.4) — prefer it on `bg-primary` / `bg-secondary`, or bump weight.

### 2.4 New semantic tokens (R3 fix)

These give the 30+ hardcoded literals a single home. Added as `.pen` variables and to
`styles.css`:

| Token | Value | Purpose | Replaces literal |
|---|---|---|---|
| `accent-subtle` | `#3B82F626` (~15%) | Active nav-item wash, selected-row tint | sidebar `#3B82F624` |
| `accent-tint` | `#3B82F61F` (~12%) | Accent badge / chip background | TechBadge `#3B82F618` |
| `accent-text` | `#60A5FA` | Accent body text / active label (TC-2) | ad-hoc `#60A5FA` |
| `success-tint` | `#22C55E1F` | Success badge bg, "已入庫" pill | `#22C55E18` |
| `error-tint` | `#EF44441F` | Error badge bg, failed-state pill | `#EF444418` |
| `error-text` | `#F87171` | Error body text (TC-2) | ad-hoc reds |
| `warning-tint` | `#F59E0B1F` | Warning badge bg | `#F59E0B18` |
| `info-tint` | `#06B6D41F` | Info badge bg | `#06B6D418` |
| `text-on-accent` | `#FFFFFF` | Label/icon on accent fills (buttons, pills) | `#FFFFFF` literals |
| `text-disabled` | `#6E7891` | Disabled/decorative text (§2.2) | misused `text-muted` |
| `overlay-scrim` | `#000000B3` (70%) | Modal / bottom-sheet backdrop | `#000000B3`, `#000000AA` |
| `focus-ring` | `#3B82F6` | Keyboard focus outline (= accent-primary, aliased for independent tuning) | `:focus-visible` literal |

**Unchanged tokens** (validated, carried forward): all `bg-*`, `border-subtle`,
`accent-primary/hover/pressed`, `success/error/warning/info` (as fills), `text-primary`,
`text-secondary`, `text-inverse`, all `radius-*`, all `shadow-*`, all `gap-*`.

### 2.5 Status → token mapping (N1 one truthful state machine)

The lifecycle the brief mandates (`想要 → 下載中 x% → 整理中 → 已入庫 → 字幕狀態`)
renders from one token mapping, identically on poster badge, detail page, and the
Activity hub:

| Lifecycle state | Surface (badge bg) | Text/icon | zh-TW label |
|---|---|---|---|
| 想要 / Requested | `info-tint` | `info` | 想要 |
| 下載中 / Downloading | `accent-tint` | `accent-text` | 下載中 · {pct}% |
| 整理中 / Organizing | `warning-tint` | `warning` | 整理中 |
| 已入庫 / In library | `success-tint` | `success` | 已入庫 |
| 失敗 / Failed | `error-tint` | `error-text` | 失敗 |
| 搜尋中 / Searching *(added 13-0, Epic 13 requests)* | `warning-tint` | `warning` | 搜尋中 |
| 字幕：有繁中 / 簡轉繁 / AI 校正中 / 缺字幕 | `success-tint` / `accent-tint` / `accent-tint` / `bg-tertiary` | matching | 繁中 / 簡轉繁 / AI 校正中 / 缺字幕 |

> **Request pipeline mapping (13-0):** the `requests.status` enum renders through this
> same table — `pending`→想要, `searching`→搜尋中 (13-0 addition: a transient
> "system working" state, warning family like 整理中), `downloading`→下載中 · {pct}%,
> `completed`→已入庫, `failed`→失敗. One state machine; no bespoke request palette.
> FE consumption is GATE-B on the 13-3/13-4 backend.

---

## 3. Type System

### 3.1 The three families and their exclusive jobs

The #1 systemic defect (R1) was CJK text set in DM Sans / Inter — fonts with **no CJK
glyphs**, producing uncontrolled OS fallback. v2 makes font choice mechanical:

| Family | Used for — **only** | Never used for |
|---|---|---|
| **Noto Sans TC** | **All CJK text, and all mixed CJK+Latin UI text** — body, headings, labels, buttons, nav, captions. This is the default UI font. | — |
| **DM Sans** | The **`vido` logo wordmark** and **pure-English display/marketing** strings (e.g. a standalone "Design System Reference" doc title). | Anything containing a CJK character; body UI text |
| **JetBrains Mono** | **Technical / numeric values**: codecs, resolution, bitrate, file paths, file sizes, counts, years, durations, hashes. | Prose; CJK |

> **Rule TY-1 (the R1 fix):** if a string can ever contain a CJK character, it is
> Noto Sans TC. Buttons, chips, menu items, empty-state titles, tab labels — all
> Noto Sans TC. (In the pre-v2 component library these were Inter/DM Sans — corrected
> in v2; see §5.1.)
>
> **Rule TY-2:** DM Sans appears in exactly two places in product UI: the logo, and
> genuinely English-only display headings. Mixed "中文 — English" strings are Noto
> Sans TC (the CJK side forces it). *(Canvas annotation titles on the Pencil light
> canvas are a separate, non-product concern and keep their existing DM Sans per the
> A–J layout convention.)*
>
> **Rule TY-3 (number + CJK unit, added 13-0):** a numeric value with a CJK unit —
> `107 分`, `12 集`, `4 季 · 87 集`, `412 部` — is never one string in one font. The
> number is its own JetBrains Mono node; the unit its own Noto Sans TC node (3–4px
> gap). Established by the 13-0 season/episode tree and detail-meta fix.

### 3.2 Type scale

Sizes in px; weights map to available Noto Sans TC / DM Sans weights. Line-height is
generous for Traditional Chinese (no descender crowding, comfortable for 2-line
titles).

| Role | Size / Weight | Line-height | Family | Notes |
|---|---|---|---|---|
| Display | 32 / 700 | 1.25 | DM Sans (EN) · Noto Sans TC (CJK) | Page-level hero title |
| H1 | 28 / 700 | 1.3 | Noto Sans TC | Route title |
| H2 | 24 / 700 | 1.3 | Noto Sans TC | Section heading |
| H3 | 20 / 600 | 1.4 | Noto Sans TC | Subsection |
| Body-L | 16 / 400 | 1.6 | Noto Sans TC | Default reading text |
| Body | 14 / 400 | 1.6 | Noto Sans TC | Dense UI text, list rows |
| Label | 14 / 600 | 1.4 | Noto Sans TC | Buttons, nav items, field labels |
| Caption | 12 / 400–600 | 1.5 | Noto Sans TC | Metadata, timestamps (≥AA via text-secondary/muted) |
| Group label | 11 / 600, +1.5 tracking | 1.4 | Noto Sans TC | Sidebar group headers (瀏覽 / 管理) |
| Mono | 11–13 / 400–500 | 1.4 | JetBrains Mono | Tech badges, counts, file data |

### 3.3 CJK title grid rule

Poster titles and card titles are designed for **two full lines of Traditional
Chinese** without clipping (R2 was content clipped to invisibility). Card title region
reserves `2 × lineHeight` (≈ 2 × 14 × 1.6 ≈ 45px) and truncates with ellipsis on the
3rd line, never mid-glyph. Year/metadata sits **below** the reserved title block, never
overlapping it (PosterCard spec §5.2).

---

## 4. Spacing, Radius, Elevation

### 4.1 Spacing — 4px base, 8pt rhythm (carried forward, unchanged)

`gap-xs 4 · gap-sm 8 · gap-md 12 · gap-lg 16 · gap-xl 24 · gap-2xl 32`. Section
rhythm = 24–48; intra-component = 8–12; touch-row padding ≥ the 44px floor (§8).

### 4.2 Radius (unchanged)

`radius-sm 4` (chips/badges inner) · `radius-md 8` (buttons, inputs, nav items,
cards-small) · `radius-lg 12` (poster cards, panels) · `radius-xl 16` (modals,
bottom-sheets top corners). Pills (chips, status) use `radius-full` (100).

### 4.3 Elevation (unchanged)

`shadow-sm` buttons · `shadow-md` cards default · `shadow-lg` hover / dropdowns /
popovers · `shadow-xl` modals / dialogs / bottom-sheets. Dark-theme opacities (0.3→0.6)
carried forward. Sidebar and top bar are **flat** (no shadow) — separated by
`border-subtle`, not elevation, to keep the chrome calm.

---

## 5. Component Specifications

> **Migration note (Alexyu, 2026-06-13 — "additive + defer"):** the *existing* shared
> components (`ButtonPrimary/Secondary`, `FilterChip`, `GenreTag`, `SortDropdown`,
> `SearchInput`, `TechBadge-*`, `PosterCard`, `EmptyLibrary-*`, `TabActive/Inactive`)
> are **not mutated in this Phase-1b commit** — that would re-render every A–J mockup
> and is a per-flow Phase-2 migration. The specs below are the **target** each flow
> adopts when it migrates; the *new* shell components (§5.3) and the v2 reference
> frames are what actually land in the `.pen` now (additive). See §10.

### 5.1 Existing atoms — v2 corrections (applied per-flow in Phase 2)

| Component | v2 correction | Root cause |
|---|---|---|
| `ButtonPrimary` / `ButtonSecondary` | label font **Inter → Noto Sans TC**; on-accent label uses `text-on-accent`; min-height 44 (touch) | R1, R4 |
| `FilterChip` | label **Inter → Noto Sans TC**; bg `bg-tertiary`, active = `accent-subtle` + `accent-text` label | R1, R3 |
| `TechBadge-*` | tint literals → `accent-tint` / `success-tint` / `warning-tint` / `info-tint`; mono label keeps JetBrains Mono | R3 |
| `EmptyLibrary-*` | titles **Inter → Noto Sans TC**; `#0F172A`→`bg-secondary`, `#FFFFFF`→`text-primary`, `#94A3B8`→`text-secondary`; CTA buttons ≥44px | R1, R3, R4 |
| `PosterCard` / `PosterCardHover` | scrim `#000000AA` → `overlay-scrim`; title 2-line CJK grid (§3.3); hover label `#FFFFFFAA` → `text-on-accent` w/ scrim | R2, R3 |
| `SortDropdown` / `SearchInput` | already Noto Sans TC; add 44px min-height, focus-ring on focus | R4 |

### 5.2 Component anatomy invariants (all components, v2)

- **Color is token-only.** No hex literal in any component fill/stroke/text (N6,
  token-lint). Tints come from the new `*-tint` tokens.
- **Touch floor.** Any interactive component's hit area ≥ 44×44px (§8), even when the
  visual is smaller (use padding / an invisible hit-frame).
- **Focus.** Interactive components show a 2px `focus-ring` outline, 2px offset, on
  `:focus-visible` (carried from `styles.css`).
- **States.** Interactive components define rest / hover / active / disabled / focus.
  Disabled uses `text-disabled` + reduced-opacity fill.

### 5.3 New shell components (land now — §6 assembles them)

| Component | Anatomy | Variants (via instance override) |
|---|---|---|
| `SidebarNavItem` | row: `[icon 18 · label(Noto 14)]` left, `[count(Mono 11) / badge-pill]` right; padding `[9,10]`; radius 8; 44px min hit | **rest** (`text-secondary` icon+label, transparent bg) · **active** (`accent-subtle` bg, `text-primary`/600 label, `accent-hover` icon) · **with-count** · **with-badge** (accent pill, `text-on-accent` Mono count) |
| `SidebarGroupLabel` | uppercase-style group header, Noto 11/600 +1.5 tracking, `text-muted` | — |
| `SidebarGroupParent` | `SidebarNavItem` + a disclosure chevron; **the label is itself a link** to the merged view, the chevron toggles children (ADR D2 affordance) | collapsed / expanded |
| `SidebarFooterStatus` | the ambient status strip (§6.4): disk bar + active-scan + queue count + service-health dots | full (expanded) / dots-only (rail) |
| `MobileTabItem` | vertical `[icon 24 · label(Noto 11)]`, centered, fills equal width | active (`accent-primary`/700) · inactive (`text-muted`/500) · with-badge dot |

---

## 6. Navigation Shell Specification

Implements the ADR's resolved IA (`01-nav-ia-decision-adr.md` D1–D4). This is the
first time navigation is a *designed* surface (the ADR notes P2: nav was never owned
in code **or** design). The shell components (§5.3) compose into four chassis views.

**Resulting top-level IA (7 destinations):**
`首頁 Home · 媒體庫 Library ▸ 電影 · 影集 (+ pinned saved views) · 探索 Discover · 活動 Activity · 下載 Downloads · 系統 System · 設定 Settings`
+ sidebar-footer status strip + header omnisearch.

### 6.1 Desktop sidebar — expanded (240px) (ADR D1-a)

```
┌─ 240px ───────────────────────┐
│  vido        ◀ (collapse)     │  logo: DM Sans 22/700 accent-primary
│  NAS 媒體庫                    │  sub:  Noto 11 text-secondary  (was text-muted — TC-1 fix)
│  ── 內容 ──────────────────    │  SidebarGroupLabel
│  ⌂  首頁                       │  SidebarNavItem
│  ▤  媒體庫              ▾      │  SidebarGroupParent (label→merged view, ▾→children)
│      🎬 電影          1,284   │    child SidebarNavItem (+count, Mono)
│      📺 影集             86    │    child
│      ★ 動畫 (已釘選)          │    pinned saved view (D2 — general mechanism)
│  🧭 探索                       │
│  ── 任務 ──────────────────    │  SidebarGroupLabel (N3: tasks adjacent, below content)
│  ◷  活動                 3    │  SidebarNavItem (+badge pill, in-flight jobs)
│  ⤓  下載                 2    │
│  ⚙  系統                      │
│  ⚙  設定                      │
│  ───────────────── (spacer) ──│
│  儲存空間  ▓▓▓▓▓▓░░  3.2/8 TB │  SidebarFooterStatus (§6.4)
│  ● 掃描中  · 佇列 5 · ●●●     │
└───────────────────────────────┘
```

- Surface `bg-secondary`, right `border-subtle` 1px, flat (no shadow).
- **Group order encodes N3:** 內容 (Home/Library/Discover) above 任務 (Activity/
  Downloads/System/Settings). Personal content is never below tasks.
- **Active state:** `accent-subtle` row wash, `text-primary`/600 label, `accent-hover`
  icon, persisted. Active matching uses TanStack router matches (ADR) — a child
  (`/library/movies`) marks **both** its item and its `媒體庫` parent as
  active-ancestor (parent gets a subtler active treatment than the leaf).
- **Library group (D2):** `媒體庫` label-clicks to the merged cross-type view; the ▾
  chevron expands `電影` / `影集` / pinned views. Counts in JetBrains Mono.
- Lucide icons: `house · library · film · tv · star(pinned) · compass · activity ·
  download · server(系統) · settings`.

### 6.2 Desktop sidebar — collapsed icon-rail (64px) (ADR D1-a, gap #3)

- Icon-only, 64px wide; user-toggled, state persisted. Converts the old `max-w-7xl`
  idle side-margin on the 27" display into content width.
- **Each icon has a Base UI `Tooltip`** (label + count) — required because labels are
  hidden (ADR D1-d; this is *why* Base UI's Tooltip drove the primitive choice).
- **Information-budget rule (resolves gap #3):** the rail shows the **7 destination
  icons + footer status dots only**. The `媒體庫` group collapses to a single
  `library` icon (its children + pins are reachable via a hover/click flyout, not
  stacked on the rail). **Pinned-view overflow:** max **3** pinned views show as
  rail icons; beyond 3, pins live only in the expanded sidebar (no rail overflow
  menu — avoids the A3 icon-soup failure mode). Active = `accent-subtle` square +
  `accent-hover` icon. Counts collapse to a dot; badge counts collapse to a dot.

### 6.3 Mobile — bottom tab bar + "More" sheet (ADR D1-b)

**Bottom-4 (Alexyu's decision, 2026-06-13): `首頁 · 媒體庫 · 活動 · 下載`** + a 5th
`More` slot.

- Rationale: mobile is a *monitoring* context (UX-spec principle 5: 手機 = 搜尋 ·
  參閱 · **遠端下載**). Both task-monitoring surfaces (活動, 下載) earn thumb-reach
  slots; `探索` is a desktop power-filter idiom (D3 boundary) and lives in More.
- Bar: 84px tall, `bg-secondary`, top `border-subtle` 1px, 4 `MobileTabItem` +
  `More`. Active = `accent-primary` icon+label /700; inactive = `text-muted` /500.
  In-flight badge = a small dot on 活動/下載.
- **44px floor:** each tab's hit area is full-height × equal-width ≥ 44px.
- **More sheet** (Base UI `Sheet`, bottom, `overlay-scrim` backdrop, `radius-xl` top):
  the status strip at the **top of the sheet**, then `探索 · 系統 · 設定`, then the
  collapse/theme/etc. (Reuses the Epic-11 bottom-sheet component — don't rebuild.)

### 6.4 Ambient status strip (ADR D4-2)

Lives in the sidebar **footer** (zero vertical-space cost — the dividend of the
collapsible-sidebar choice). Renders four ambient NAS concerns:

- **Disk headroom** — a `radius-sm` track (`bg-tertiary`) + fill (`accent-primary`,
  → `warning`/`error` as it fills); `3.2 TB / 8.0 TB` in JetBrains Mono.
- **Active scan** — `● 掃描中` dot (`accent-primary`, pulses; respects reduced-motion).
- **Queue count** — `佇列 N` (Mono).
- **Service-health dots** — `●●●` (`success` / `warning` / `error`), one per service;
  each a Base UI `Tooltip` (qBittorrent / TMDB / Scanner).
- **Collapsed rail:** strip collapses to the dots only. **Mobile:** strip sits at the
  top of the More sheet.
- Data is a fail-soft aggregate (ADR B1/F3): a downstream-unavailable section renders
  empty/stale, never fails the strip.

### 6.5 Header omnisearch (ADR D4-5)

One search box (replaces the three-surface confusion of P9): searches **local library
+ TMDB + (future) subtitles**, sectioned results in a `shadow-lg` popover (Base UI
`Popover`). `SearchInput` styling (§5.1), 44px height, `focus-ring` on focus, Noto Sans
TC placeholder. `/search` route retires (route-level redirect preserving `q`).

---

## 7. State Patterns (N4 — four states or it doesn't ship)

Every data-bearing surface designs all four states up front. A screen design missing a
state is an **incomplete deliverable** in Phase 1+.

| State | Standard | Tokens |
|---|---|---|
| **Empty** (no data yet, expected) | Centered icon + Noto title + Noto subtitle + a *next-step* CTA (N: "always show next step"). Onboarding-aware: distinguishes no-qBittorrent / no-folder / ready-for-scan (the `EmptyLibrary-*` 3-state classifier). | `bg-secondary` card, `text-primary` title, `text-secondary` subtitle, `ButtonPrimary` CTA |
| **Loading** | **Skeleton** matching the real layout's shape (poster blocks, text bars), not a spinner; shimmer respects `prefers-reduced-motion`. Progressive: render shells immediately, hydrate per-section. | `bg-secondary`/`bg-tertiary` skeleton blocks, `radius-lg` |
| **Error** | Per-section, **fail-soft** (Rule 27 / ADR F3): the failed section shows a compact inline error + retry; the **page never hard-fails**. Error text uses `error-text` (TC-2). | `error-tint` panel, `error-text` message, `ButtonSecondary` 重試 |
| **No-result** (query/filter returned nothing) | Distinct from Empty — acknowledges the filter, offers "清除篩選" / "調整搜尋". Never a bare blank. | `text-secondary` message, `FilterChip` clear affordance |

> **Fail-soft is bidirectional (ADR F3):** the backend returns a per-section status
> object; the frontend treats a non-`ok` section as **data** (renders empty/stale) and
> must not throw. External-data blocks (TMDB / Douban / trailers / providers) degrade
> per-section — one unavailable source never blanks the page (design-context-pack §4.4).

---

## 8. Accessibility Baseline

Hard gates, not guidance (P4: a11y shipped broken until a gate forced it). v2 bakes
these into tokens and components so they pass by construction.

- **Contrast ≥ WCAG AA.** Body text ≥ 4.5:1, large/bold (≥18px / ≥14px bold) ≥ 3:1.
  The §2.2 text scale and §2.3 colored-text rules guarantee this; `text-disabled` is
  the *only* sub-AA token and is barred from load-bearing text (TC-1).
- **Touch targets ≥ 44×44px** (R4) on every interactive element, all breakpoints —
  enforced via min-height / hit-frame padding even where the visual is smaller.
- **Visible focus.** 2px `focus-ring`, 2px offset, on `:focus-visible` — never removed.
- **Keyboard + focus management.** Dialogs / sheets / popovers / menus use Base UI
  primitives (ADR D1-d) so focus trap, restore, Escape, and arrow-key navigation are
  correct by default — this is the structural fix for the hand-rolled-a11y failures of
  Epics 10-2 / 11-2.
- **Reduced motion.** All animation respects `prefers-reduced-motion` (carried from
  `styles.css`); the status-strip pulse and skeleton shimmer disable under it.
- **Text alternatives.** Icons, star ratings, and status dots carry `aria-label`;
  color is never the *sole* carrier of state (status pills pair a hue with a label).

---

## 9. Enforcement (N6 — ride the existing gates)

v2 ships as tokens + automated checks, reusing the Epic-19 infrastructure rather than
rebuilding it:

- **Token-lint** — no hardcoded hex in components/styles; semantic colors must resolve
  to a token (the R3 sweep target). Catches new literals at PR time.
- **Contrast check** — TC-1 / TC-2 encoded: `text-disabled` not used for essential
  text; colored body text uses `*-text` variants.
- **Touch-target & jsx-a11y** — `jsx-a11y` already enabled (retro-11-AI1b); extend with
  a 44px min check on interactive components.
- **Visual-regression CI** (Rule 22/23) — the net that makes the §10 token sweep safe;
  `-linux` baselines bootstrap via the existing chore-visual workflow.
- **Rule 21 component↔design traceability** — new shell components reference their
  `.pen` node IDs (the Navigation Shell v2 reference frame, §11) instead of carrying the
  design-coverage-gap placeholder.

---

## 10. What lands now vs. Phase-2 migration

Honest scoping, per Alexyu's "additive + defer" decision and the brief's §7 warning
that a token sweep consumed by all flows "needs the visual-regression net before any
sweep":

**Lands in this Phase-1b `.pen` commit (additive — only design-system screenshots change):**
- The 12 **new** semantic-token variables (§2.4) — net-new, referenced by nothing
  existing, so zero propagation.
- The **Design Language v2** reference frame (token swatches, v2 text scale, type
  system corrected to Noto Sans TC, four-state patterns, a11y baseline).
- The **Navigation Shell v2** reference frame + the new shell components (§5.3, §6).

**Deferred to Phase 2 (tracked stories, behind the visual-regression net):**
- Reassigning `--text-muted` `#808080 → #A0AABE` in `styles.css` + `.pen` variable
  (the global value change that re-renders every flow — the token *sweep*).
- Migrating the existing shared atoms (§5.1) to v2 fonts/tokens — done **per flow** as
  each flow migrates (strangler), starting with Browse + Detail in the Phase-2 pilot.
- Wiring the shell to code (the ADR FOUNDATION story: `AppShell` swap behind
  `new_shell_enabled`, Base UI wrappers, `LegacyContentContainer`).

This keeps Phase 1b strictly "language + IA foundation," and hands Phase 2 a fully
specified, directly-applicable token + shell system.

---

## 11. `.pen` Landing Manifest

Landed in the `design-system` flow of `ux-design.pen` (node IDs filled in on commit):

- **Variables added** (`set_variables`, merged): `accent-subtle`, `accent-tint`,
  `accent-text`, `success-tint`, `error-tint`, `error-text`, `warning-tint`,
  `info-tint`, `text-on-accent`, `text-disabled`, `overlay-scrim`, `focus-ring`.
- **Frame: `Design Language v2`** — sections: new-token swatches · v2 text scale (AA
  annotated) · type system (Noto/DM/Mono, R1-corrected CJK samples) · four-state
  patterns · accessibility baseline.
- **Frame: `Navigation Shell v2`** — sidebar expanded (240) · collapsed rail (64) ·
  mobile tab bar (首頁·媒體庫·活動·下載·More) · More sheet · status strip detail.
- **Components added**: `Component/SidebarNavItem`, `Component/SidebarGroupLabel`,
  `Component/SidebarGroupParent`, `Component/SidebarFooterStatus`,
  `Component/MobileTabItem`.
- Screenshots: both new frames registered in `scripts/export-pen-screenshots.py`
  `SCREENS` under `design-system`; regenerated; only genuinely-new PNGs committed.

---

## 12. Definition of Done & Handoff

- ✅ Token system v2 — R3 semantic tokens defined; R5 `text-muted` corrected (value +
  usage rule + `text-disabled`); colored-text AA traps (TC-2) resolved; status→token
  map for N1.
- ✅ Type system — Noto Sans TC for all CJK (R1 fix, TY-1), DM Sans logo/EN only
  (TY-2), JetBrains Mono numeric; type scale + CJK 2-line title grid.
- ✅ Spacing / radius / elevation — carried forward with density + flat-chrome rules.
- ✅ Component specs — existing-atom v2 corrections (deferred per-flow) + new shell
  components (landed).
- ✅ Navigation shell — sidebar (expanded + 64px rail), mobile bottom-4 + More, status
  strip, omnisearch; all ADR D1–D4 decisions visually resolved, gaps #1 (bottom-4) and
  #3 (rail density + pin overflow) closed.
- ✅ Four-state standard (N4) + accessibility baseline (AA, 44px, focus, Base UI a11y).
- ✅ Enforcement mapped to existing N6 gates.
- ✅ Landed in `ux-design.pen` design-system flow (§11).

**Handoff to Phase 2 (Browse + Detail pilot):** the FOUNDATION story builds the shell
in code from §5.3/§6 behind `new_shell_enabled`; Browse + Detail migrate to v2 tokens
(§2) and type (§3), adopting the four-state standard (§7). The `--text-muted` reassign
and per-atom migration (§10) ride the visual-regression net. Seam with the ADR holds:
ADR owns route/flag/test-contract behavior, this document owns every visual token,
type rule, component, state, and a11y minimum.
