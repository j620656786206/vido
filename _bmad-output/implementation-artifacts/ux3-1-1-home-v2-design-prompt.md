# ux3-1-1 — Home v2 design prompt (for the Pencil In-App AI agent)

> Paste the block below into Pencil.app's in-app AI agent to draw the Home v2 screens.
> Then I'll review the result via Pencil MCP (read/screenshot) — token-light vs me drawing.

---

You are redesigning the **Home / 首頁** flow of the Vido NAS media app to the **v2 design
language** (UX Redesign Phase 3). Work in the current `ux-design.pen`. Create NEW frames
(don't overwrite the existing H1–H5); place them as siblings just below/right of the
existing flow-h frames (anchor near H1-D `sAaCR` at x:17040, y:18320). Name them
`H1-D-v2 · 首頁 v2（桌面）`, `H2-M-v2 · 首頁 v2（手機）`, `H4-D-v2 · 首頁載入骨架 v2`,
plus state frames named below. Set `clip: true` and `placeholder: true` on each new screen
frame while building; remove `placeholder` when the screen is done.

## The composition (this is a fixed design RULE — do not reorder)

Render the home content inside the **v2 sidebar shell** (reference the assembled
`Navigation Shell v2` frame `CLo58` — collapsible left sidebar + sidebar-footer status
strip; NOT the legacy top tabs). The main content column, **top → bottom**:

1. **Own-content zone (ALWAYS above external — D3 ordering law):**
   - **繼續觀看** horizontal row — RESERVED slot. Its data is blocked on a future
     media-server integration, so design its **empty/placeholder state**: a row header
     「繼續觀看」 + a muted inline hint 「連接 Plex / Jellyfin 後顯示」 (text-muted), NOT a
     broken empty box. (In the populated variant you may show 2–3 poster cards with a thin
     progress bar along the bottom edge.)
   - **最近新增** horizontal row — a scrollable row of `Component/PosterCard-v2` (`hD7Tw`)
     instances (6–8 visible), each ~150–160px wide, 2-line CJK title.
   - Do NOT put a full download dashboard here (it moved to the Activity hub). At most a
     single compact one-line 「進行中 · N」 chip linking to Activity.
2. **Hero banner** (keep Epic 10's hero in full — the emotional centerpiece). Reuse the
   look of `hero-banner` `VSORG` (backdrop, title, rating, actions, carousel dots) BUT
   restyle per v2 (see Type rule — the current hero wrongly uses DM Sans for the CJK
   title; v2 requires Noto Sans TC).
3. **ExploreBlocks** (keep in full) — reuse the look of `explore-blocks` `hosiq` (block
   title + horizontal poster row), restyled to v2.

## Design language v2 (apply everywhere)

- **Type (the #1 fix):** ALL CJK and mixed CJK+Latin text = **Noto Sans TC**. Use DM Sans
  ONLY for the `vido` logo and pure-English display strings. JetBrains Mono for numeric/
  tech values (counts, years, durations, "3.2 / 8.0 TB"). The existing hero uses DM Sans
  for 你的名字 — that is WRONG; switch CJK to Noto Sans TC.
- **Color = tokens only, no hex literals.** Use the document variables: `$bg-primary`,
  `$bg-secondary`, `$bg-tertiary`, `$text-primary`, `$text-secondary`, `$text-muted`
  (= #A0AABE, AA-safe), `$accent-primary`, `$accent-text`, `$accent-subtle`,
  `$accent-tint`, `$success`, `$success-tint`, `$warning`, `$warning-tint`, `$error`,
  `$error-text`, `$error-tint`, `$border-subtle`, `$overlay-scrim`, `$radius-*`,
  `$shadow-*`, `$gap-*`. Reference `Design Language v2` frame `V2Kez` for the scale.
- **Status badges (N1 — one truthful state machine):** poster lifecycle/subtitle badge
  uses the §2.5 map — 已入庫 = `$success-tint`/`$success`, 整理中 = `$warning-tint`/
  `$warning`, 失敗 = `$error-tint`/`$error-text`, 繁中 = success tint, 缺字幕 =
  `$bg-tertiary`/muted. The badge is an EXCEPTION signal — the steady state (已入庫 + 繁中)
  shows NO badge.
- **2-line CJK title grid:** poster titles reserve two full lines, ellipsis on overflow,
  never clip mid-glyph; year/meta sits below.
- **Touch targets ≥ 44×44px** on every interactive element (mobile especially).
- **Spacing:** section rhythm 24–48; intra-component 8–12. Flat chrome (sidebar/header use
  `$border-subtle`, not shadow).

## States to produce (N4 — four states or it doesn't ship)

Each as its own frame (desktop unless noted):
- **Default / populated** — the composition above with real-looking content.
- **Loading skeleton** — evolve `H4-D` `g6p38` to v2 (skeleton blocks matching the new
  own-content rows + hero + explore shape; `$bg-secondary`/`$bg-tertiary` blocks).
- **Own-content sparse/empty** — when the library is stable (no recent adds, no active
  tasks): the own-content zone collapses gracefully WITHOUT leaving a top gap; the Hero
  rises but stays conceptually below the (empty) own-content zone; show a quiet
  「尚無最近新增」 hint, not a blank. (This is the key edge case — design it explicitly.)
- **Error (per-section fail-soft)** — if a section's data fails, that section shows a
  compact inline 「無法載入，請稍後再試」 + 重試 button (`$error-tint` panel, `$error-text`),
  the rest of the page still renders. Never a full-page error.

## Mobile (H2-M-v2, 390px)

Same composition (own-content rows above Hero+Explore) in the mobile v2 shell: bottom tab
bar (`Component/MobileTabItem` `S86VM`: 首頁·媒體庫·活動·下載·More) + the status strip at
the top of the More sheet. Horizontal rows scroll; 44px targets; no clipped CJK.

## Components to REUSE (don't reinvent)

- `Component/PosterCard-v2` (`hD7Tw`) — every poster.
- `Component/SidebarNavItem` (`W5KQr`), `SidebarGroupParent` (`imFBW`), `SidebarGroupLabel`
  (`v5Io8`), `SidebarFooterStatus` (`PrmQG`), `MobileTabItem` (`S86VM`) — the shell.
- Reference (copy/adapt, restyle to v2): `hero-banner` `VSORG`, `explore-blocks` `hosiq`,
  mobile `m-hero-banner` `j5EVJ` / `mExplore` `y3dMj`.

## Done checklist

- Own-content is structurally ABOVE the Hero+Explore in every variant (D3).
- All CJK is Noto Sans TC (no DM Sans on Chinese text).
- No hex literals — tokens only.
- Four states present incl. the sparse own-content state.
- Nothing clipped; layout not collapsed; 44px touch targets; AA contrast.
- New frames only; existing H1–H5 untouched; `placeholder` removed when done.
