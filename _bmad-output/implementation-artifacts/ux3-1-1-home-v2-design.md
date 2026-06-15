# Story ux3-1-1 — Home v2 design (`.pen` flow-h → v2)

**Epic:** ux3-home-v2 (UX Redesign Phase 3) · **Status:** done (design landed + reviewed)
**Owner:** ux-designer (Pencil in-app AI agents, MCP-reviewed) · **Type:** design · **FRs:** PH3-M1, PH3-R1

## Story

As the design system,
I want the Home flow redrawn to v2,
So that dev builds Home v2 against a spec (per-flow recipe step 1), validating D3.

## What landed (in `ux-design.pen`, flow-h-homepage-v2)

Drawn by the Pencil **in-app AI agent** (token-efficient: it draws, I review via MCP),
then MCP-reviewed + corrected. New frames + a new component:

- **`Component/HomeSidebar-v2`** (`BDeUS`) — the v2 sidebar shell assembled from the
  shipped shell components (SidebarNavItem/GroupParent/GroupLabel/SidebarFooterStatus):
  首頁(active=`$accent-tint`) · 媒體庫 group · 探索 · 活動 · 下載(badge) · 系統 · 設定 +
  footer status strip. Noto Sans TC, token-only.
- **`H1-D-v2`** (`yixu1`) — desktop default. Main column top→bottom (D3 ordering law):
  **own-content** (繼續觀看 reserved slot + 最近新增 PosterCardV2 row) **ABOVE**
  **hero-banner-v2** **ABOVE** **explore-blocks-v2**.
- **`H2-M-v2`** (`uCfjb`) — mobile: top app bar + scroll content (same own-above-external
  order) + bottom tab bar (首頁·媒體庫·活動·下載·More).
- **`H4-D-v2`** (`nnGs6`) — loading skeleton (v2 shell + skeleton blocks; no hero).
- **`H5-D-v2`** (`Z7OJB`) — own-content **sparse/empty** state (the pre-mortem edge case:
  graceful collapse, 「尚無最近新增」 hint, no top gap, Hero stays below the zone).
- **`H6-D-v2`** (`xCQA7`) — per-section **fail-soft** error (a failed section shows inline
  「無法載入，請稍後再試」 + 重試; the page still renders).

## Review (MCP) — PASS

- **D3 ordering law** verified: own-content structurally above Hero+Explore in every
  variant (the headline watch-item — its felt-experience gate is the post-build P10
  browser-verify, not design).
- **繼續觀看 reserved slot**: empty-state hint 「連接 Plex / Jellyfin 後顯示」 (data blocked
  on Epic 17) — not a broken/fake-populated tile (correct, PH3-R1).
- **Type**: all CJK = Noto Sans TC (the legacy hero's DM-Sans-on-CJK R1 violation is
  fixed), numeric/tech = JetBrains Mono. A residual mobile `mHeroSub` DM-Sans miss was
  fixed via MCP. Remaining DM Sans = the legacy VSORG source hero only (out of scope).
- token-only colors, v2 components reused (no fork), 5 states present.

## Close-out

- Screenshots regenerated via `scripts/export-pen-screenshots.py` (`SCREENS` extended with
  the 5 `flow-h-homepage-v2` frames); only the genuinely-new PNGs committed (re-render was
  byte-stable this run — 0 noise).
- Design prompt archived: `ux3-1-1-home-v2-design-prompt.md`.
- **Next (ux3-home-v2):** sm create-story → dev builds Home v2 in code (own-content blocks
  above Hero/ExploreBlocks, shell-version gated, four states) → tea → ship → P10
  browser-verify the D3 own-above-external felt experience under real content.
