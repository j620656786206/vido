# ux3-3-1 — Discover v2 design prompt (for the Pencil In-App AI agent)

> Paste the block below into Pencil.app's in-app AI agent to draw the Discover v2 screens.
> Then ux-designer reviews the result via Pencil MCP (read/screenshot/targeted-fix) — token-light
> vs. drawing the whole flow over MCP. Same path as ux3-1-1 (home v2), documented as token-efficient.
> Story spec: `ux3-3-1-discover-design.md`.

---

You are redesigning the **探索 / Discover** flow of the Vido NAS media app to the **v2 design
language** (UX Redesign Phase 3). Work in the current `ux-design.pen`. Create NEW frames in a
**new flow group** — these are Discover's power-filter screens, NOT the existing advanced-search
mockups and NOT the library filter rail. Name the new frames:
`I1-D-v2 · 探索 v2（桌面）`, `I2-M-v2 · 探索 v2（手機）`, `I3-D-v2 · 即時搜尋建議`,
`I4-D-v2 · 篩選面板`, `I4-M-v2 · 篩選 sheet（手機）`, `I5-D-v2 · 儲存篩選`,
`I6-D-v2 · 載入骨架`, `I7-D-v2 · 無結果`, `I8-D-v2 · 區段 fail-soft`.
Place them as siblings **clear of and below/right of** the existing flow-i frames
(`i1-d` `NWxok` · `i2` `TMaw5` · `i3` `i74p2` · `i4-m` `pjKVZ` · `i5-d` `vpDLh` · `i6-d` `VwTvy`
· `i7-d` `SgncH`) — read their positions and anchor a new row that does not overlap any of them.
Set `clip: true` and `placeholder: true` on each new screen while building; remove `placeholder`
when the screen is done.

## ⚠️ Do NOT reuse these (they are different surfaces)

- `i1-d` `NWxok` / `i2` / `i3` / `i4-m` — the **pre-v2** advanced-search mockups. Stale; `i1-d`
  invented unbacked dimensions (地區 / 最低評分). Redraw from scratch to v2 — do not copy.
- `i5-d` `vpDLh` / `i6-d` `VwTvy` / `i7-d` `SgncH` — the **媒體庫 own-collection filter rail**
  (a different feature: it filters what you already have). Discover power-filters across
  **TMDB + library to find things**. Discover v2 must read as visibly distinct from that rail.

## What Discover is (the composition)

Render inside the **v2 sidebar shell** (reference the assembled `Navigation Shell v2` frame
`CLo58`; sidebar active item flipped to **探索 / compass icon**; NOT the legacy top tabs). The
main content column, **top → bottom**:

1. **Search input** — instant, debounced (this is a power-filter discovery box, not the header
   omnisearch). Noto Sans TC placeholder, 44px height, focus-ring on focus.
2. **Persistent filter-chip row** — active filters as **removable** chips (active =
   `$accent-subtle` bg + `$accent-text` label) + a `清除全部` affordance. Chips persist across
   browsing.
3. **Saved-preset chip row** — named presets as chips (applied = active state) + a
   `+ 儲存目前篩選` affordance. (Epic 11 "saved filter presets".)
4. **Toolbar** — a **sort** control (sort lives in the toolbar, not a side panel) + a
   `篩選` button that opens the multi-dimension filter panel (#I4) + a **reserved, inert**
   `想要清單 / 請求` entry (Epic 13 lands later — draw it quiet and "coming soon"-feeling, like
   a disabled affordance; NEVER a broken control).
5. **Results grid** — `Component/PosterCard-v2` (`hD7Tw`) cards. **The rating / 評分 value is
   visible on each card** (this is where rating-sort/filter lives — the value must show).

**D3 boundary (a hard RULE):** Discover grows **NO dashboard** — no download panel, no qBittorrent
status, no connection history. It is the discovery surface only.

## The dimension set (READ THIS — Rule 24 capability-honor)

The multi-dimension filter panel (#I4) exposes Epic 11's dimensions: **類型 genre · 年份 year ·
地區 region · 評分 rating · 串流平台 streaming platform**. BUT only draw a dimension as **active**
if the Epic 11 filter backend can actually query it. If 地區 / 評分 / 串流平台 are not backed by a
real query parameter, draw them **explicitly disabled / "即將推出"** rather than as live controls —
do NOT re-introduce `i1-d`'s unbacked invented fields as if they work. (If you are unsure, draw
類型 + 年份 as definitely-live and mark the other three as reserved, with a note frame.)

## Design language v2 (apply everywhere)

- **Type (the #1 fix):** ALL CJK and mixed CJK+Latin text = **Noto Sans TC**. DM Sans ONLY for the
  `vido` logo / pure-English display. **JetBrains Mono** for numerics (評分 values, years, counts).
- **Color = tokens only, no hex literals.** Use document variables: `$bg-primary`, `$bg-secondary`,
  `$bg-tertiary`, `$text-primary`, `$text-secondary`, `$text-muted` (#A0AABE, AA-safe),
  `$text-disabled` (never load-bearing text), `$accent-primary`, `$accent-text`, `$accent-subtle`,
  `$accent-tint`, `$success`/`$success-tint`, `$warning`/`$warning-tint`, `$error`/`$error-text`/
  `$error-tint`, `$info`/`$info-tint`, `$border-subtle`, `$overlay-scrim`, `$radius-*`, `$shadow-*`,
  `$gap-*`. Reference the `Design Language v2` frame `V2Kez` for the scale.
- **Colored body text uses the `*-text` AA variants** (`$accent-text` / `$error-text`), not the base
  fill hues.
- **2-line CJK title grid** on poster cards; ellipsis on overflow, never clip mid-glyph.
- **Touch targets ≥ 44×44px** on every interactive element (chips, buttons, sort, filter rows).
- **Spacing:** section rhythm 24–48; intra-component 8–12. Flat chrome (`$border-subtle`, not shadow).

## States to produce (N4 — four states or it doesn't ship)

- **`I1-D-v2` Default / populated** — the composition above with real-looking content.
- **`I3-D-v2` Instant-search suggestions** — debounced suggestions in a `$shadow-lg` popover,
  **sectioned 媒體庫 (local) / TMDB**, zh-TW results boosted to the top. Rows Noto Sans TC; years Mono.
- **`I4-D-v2` / `I4-M-v2` Filter panel/sheet** — the dimension set above. Desktop = inline panel or
  popover; **mobile = bottom sheet** (`$radius-xl` top corners, `$overlay-scrim` backdrop) — reuse
  the existing Epic-11 mobile filter sheet look (`i4-m` `pjKVZ` as structural reference), restyled v2.
- **`I5-D-v2` Save-preset** — the `儲存目前篩選` interaction (name + confirm).
- **`I6-D-v2` Loading skeleton** — grid-shaped skeleton blocks + chip-row skeleton
  (`$bg-secondary`/`$bg-tertiary`); shimmer respects reduced-motion.
- **`I7-D-v2` No-result** — filter/search returned nothing. **Distinct from empty**: acknowledge
  the active filter, offer `清除篩選` / `調整搜尋`. Never a bare blank.
- **`I8-D-v2` Per-section fail-soft** — TMDB suggestions/availability source down → that section
  shows inline `無法載入，請稍後再試` + `重試` (`$error-tint` panel, `$error-text`); the rest of the
  page (local results, chips) still renders. Never a full-page error.

## Mobile (`I2-M-v2`, 390px)

**Discover is NOT a bottom-tab on mobile — it is reached via the More sheet** (探索 is a desktop
power-filter idiom). Draw: top app bar (探索 title + search entry) → condensed chip row → a `篩選`
trigger that opens the bottom sheet (#I4-M) → results grid. The bottom tab bar shows the **bottom-4
首頁 · 媒體庫 · 活動 · 下載 + More** (use `Component/MobileTabItem` `S86VM`) — **no 探索 tab**.
44px targets; horizontal rows scroll; no clipped CJK.

## Components to REUSE (don't reinvent)

- `Component/PosterCard-v2` (`hD7Tw`) — every result poster.
- Shell: `Component/SidebarNavItem` (`W5KQr`), `SidebarGroupParent` (`imFBW`), `SidebarGroupLabel`
  (`v5Io8`), `SidebarFooterStatus` (`PrmQG`), `MobileTabItem` (`S86VM`); or the assembled
  `Component/HomeSidebar-v2` instanced with 探索 active.
- **Filter chips / search input / sort** — instance the existing `FilterChip` / `SearchInput` /
  `SortDropdown` atoms and apply the v2 corrections (Noto Sans TC labels, token fills,
  `$accent-subtle` active chip, 44px, focus-ring). If a **preset chip** needs a different anatomy
  (name + applied-state + remove), make it a `FilterChip` instance override before forking a new
  component.
- Reference (copy/adapt, restyle to v2): the existing `i4-m` `pjKVZ` mobile sheet structure.

## Done checklist

- New frames only, in a new flow group; existing `i1-d…i4-m` and `i5-d/i6-d/i7-d` UNTOUCHED.
- Discover reads as visibly distinct from the library filter rail.
- All CJK is Noto Sans TC; numerics JetBrains Mono; no hex literals (tokens only); AA contrast.
- Four states present incl. **no-result distinct from empty** and **per-section fail-soft**.
- No dashboard elements (D3 guardrail #2).
- Epic 13 Requests entry present but inert/"coming later" — never a broken control.
- Only dimensions the backend supports are drawn live; unbacked ones explicitly reserved/disabled.
- 評分/vote_average visible on result cards.
- Mobile reaches Discover via More (bottom-4 unchanged); 44px targets; nothing clipped.
- `placeholder` removed on every finished screen.

---

### After the in-app agent finishes (ux-designer, via MCP — token-light)

1. `get_screenshot` each new `I*-D-v2` / `I*-M-v2` frame; check against this checklist + DL-v2.
2. Targeted `batch_design` fixes only (don't redraw): token literals, font slips (CJK in DM Sans),
   missing states, dimension-set capability, the inert Requests entry, 44px floors.
3. **Rule-24 capability audit** of the Epic 11 filter engine — confirm which of 地區/評分/串流平台
   are live; file a backlog entry for any unbacked dimension (sprint-status), per the story's
   Discovery Triage.
4. Update `scripts/export-pen-screenshots.py` `SCREENS` (new node IDs → codes) → run
   `python3 scripts/export-pen-screenshots.py` → commit `.pen` + **only genuinely-changed PNGs**
   (regen is non-deterministic) as `feat(ux3-3-1): Discover v2 design (.pen flow-i-discover-v2)`.
5. Set `ux3-3-1-discover-design` → `done` in sprint-status; fill the story's Close-out (node IDs).
