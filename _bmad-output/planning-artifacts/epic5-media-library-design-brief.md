# Vido Media Library — Design Brief for Pencil

**Epic:** 5 - Media Library Management
**Date:** 2026-03-05
**Author:** Sally (UX Designer) + Alexyu
**Reference:** UX Design Specification (ux-design-specification.md), Epic 5 Stories (epics.md)
**Design File:** `ux-design.pen` (Pencil app — read via MCP tools only)
**Screenshots:** `_bmad-output/screenshots/` (regenerate via `python3 scripts/export-pen-screenshots.py`)

---

## 1. Project Context

Vido is a smart media management tool for Traditional Chinese NAS users. Epic 5 covers the **Media Library Management** frontend — the core experience where users browse, search, filter, and manage their media collection.

**Design Philosophy:** "Automatic but visible, intelligent but controllable"

**Core Emotional Goal:** Users feel pride and satisfaction seeing their media collection displayed with perfect Traditional Chinese metadata and beautiful posters — the "Appreciation Loop."

---

## 2. Design Principles (Anti-AI-Slop Rules)

These rules ensure the design looks professionally crafted, not AI-generated:

### Icons Over Emojis
- Use **Lucide icons** (already in the codebase) for all UI elements: navigation, actions, status indicators, badges
- Emojis are ONLY acceptable in:
  - Empty state illustrations (friendly, human touch)
  - Fallback poster placeholders when no image exists (a single film/tv/art emoji as icon, styled subtly)
- NEVER use emojis for: navigation tabs, buttons, badges, status indicators, card overlays
- When in doubt, use a Lucide icon with appropriate color instead

### Color Through Meaning, Not Decoration
- Color should come from **meaningful data**: metadata source badges, rating scores, download progress, genre tags
- NO gratuitous gradient backgrounds, rainbow badges, or decorative color blocks
- Status colors (success/error/warning) only appear when there IS a status to communicate
- Poster images themselves are the primary color source — the UI should be a neutral, dark stage that lets posters shine

### Layout With Purpose
- Every element must earn its screen space — no decorative fillers
- Collapse secondary actions into contextual menus (triple-dot / right-click)
- Avoid repeating the same information across multiple sections
- Tighten spacing where density improves scanability; add breathing room where it aids focus

### Professional Typography Hierarchy
- **Display font** (English titles, logo, large headings): DM Sans or Plus Jakarta Sans — 700-800 weight, modern and distinctive
- **Body font** (Chinese text, descriptions, metadata): Noto Sans TC — optimized for Traditional Chinese readability
- **Monospace** (technical info, file sizes, dates): JetBrains Mono — adds precision feel to data
- Film/show titles in the Detail Panel should be the clear visual anchor — largest size, boldest weight

---

## 3. Design System Foundation

### Color Palette (Midnight Blue Dark Theme)

```
Background:
  --bg-primary:    hsl(222, 47%, 11%)    Main canvas
  --bg-secondary:  hsl(217, 33%, 17%)    Cards, panels
  --bg-tertiary:   hsl(215, 28%, 23%)    Hover states

Accent:
  --accent-primary: hsl(217, 91%, 60%)   Primary actions (vibrant blue)
  --accent-hover:   hsl(217, 91%, 70%)   Hover
  --accent-pressed: hsl(217, 91%, 50%)   Active/pressed

Semantic:
  --success: hsl(142, 76%, 36%)          Completion, connected
  --error:   hsl(0, 84%, 60%)            Errors, failures
  --warning: hsl(38, 92%, 50%)           Pending, warnings
  --info:    hsl(200, 98%, 48%)          Informational

Text:
  --text-primary:   hsl(0, 0%, 95%)      Headings, primary content
  --text-secondary: hsl(0, 0%, 70%)      Descriptions, metadata
  --text-muted:     hsl(0, 0%, 50%)      Timestamps, tertiary info
```

### Background Atmosphere
- Grid area: Subtle radial gradient from center (slightly lighter) to edges (darker), creating a "spotlight on collection" feel
- Very faint noise texture overlay (opacity 0.02-0.03) to add depth without distraction
- Detail Panel top: Blurred, enlarged poster image as backdrop with gradient fade to --bg-secondary

### Shadows & Radius
```
Radius:  sm(4px) md(8px) lg(12px) xl(16px)
Shadows: sm(subtle) md(cards) lg(hover) xl(modals) 2xl(panels)

Poster cards: radius-lg (12px), shadow-md default, shadow-xl on hover
Poster hover: Add micro accent glow — hsl(217, 91%, 60%, 0.12) beneath card
```

### Animation Tokens (for annotation only — Pencil shows static keyframes)
```
--duration-fast:   150ms    Hover, focus
--duration-base:   300ms    Card transitions
--duration-slow:   500ms    Panel slide-in
--ease-out:        cubic-bezier(0, 0, 0.2, 1)
```

### Responsive Breakpoints
```
Desktop:  1440px+     5-6 poster columns, 16px gap
Tablet:   768-1439px  3-4 columns, 12px gap
Mobile:   <768px      2 columns, 12px gap
```

---

## 4. Page Layout Structure

### Global Shell (All Pages)

```
Desktop (1440px+):
+-----------------------------------------------------+
| Top Toolbar (60px, fixed)                           |
| [Vido logo]  [--- Global Search (40%) ---]  [3] [cog]|
+-----------------------------------------------------+
| Tab Navigation (48px)                               |
| [Library]  [Downloads]  [To Parse]  [Settings]      |
+-----------------------------------------------------+
|                                                     |
|              Main Content Area                      |
|              (fills remaining viewport)             |
|                                                     |
|                           [Floating Parse Card -->] |
+-----------------------------------------------------+
```

**Top Toolbar:**
- Logo: Left-aligned, "vido" wordmark (120px width)
- Global Search: Center, 40% width, rounded input with Lucide Search icon
- Pending Parse Badge: Small pill badge with count (e.g., "3"), --warning color, right of search
- Settings: Lucide Settings icon, far right

**Tab Navigation:**
- Active tab: --text-primary (95%) + 2px bottom border in --accent-primary
- Inactive tab: --text-secondary (70%), no border
- Hover: --text-primary + 1px bottom border in --text-muted (fades in)
- All text, no emojis, no icons in tabs

**Floating AI Parse Progress Card (Bottom-right):**
- Width: 360px, --bg-secondary, --shadow-lg, radius-xl
- Shows: Lucide Loader icon + filename + progress bar + detected title
- Collapsible to small icon when minimized

---

## 5. Screen Specifications (10 Screens)

### Screen 1: Library Grid View — Desktop (1440px+)
**Stories:** 5.1, 5.8

```
+-----------------------------------------------------+
| [Top Toolbar + Tabs as defined above]               |
+-----------------------------------------------------+
| Recently Added                          [See All ->]|
| [poster][poster][poster][poster][poster][poster]    |
|  Title    Title    Title   Title   Title   Title    |
+-----------------------------------------------------+
| All Media                [Sort: v] [Filter] [Grid|List]|
+-----------------------------------------------------+
|                                                     |
| [poster] [poster] [poster] [poster] [poster]        |
|  Title    Title    Title    Title    Title           |
|                                                     |
| [poster] [poster] [poster] [poster] [poster]        |
|  Title    Title    Title    Title    Title           |
|                                                     |
| [poster] [poster] [poster] [poster] [poster]        |
|  Title    Title    Title    Title    Title           |
+-----------------------------------------------------+
```

**Recently Added Section:**
- Horizontal scrollable row of 6-8 posters (smaller than main grid)
- Section header: "Recently Added" left + "See All" link right (Lucide ArrowRight icon)
- New items: Small accent-colored dot or subtle "NEW" text badge (not emoji)

**Main Grid:**
- Sub-header row: "All Media" label + Sort dropdown + Filter button + View toggle (Grid/List icons)
- Poster cards: 5-6 columns, 2:3 ratio, 256px wide, 16px gap
- Below each poster: Title (1-2 lines, Noto Sans TC), Year in --text-muted
- Virtual scrolling indication: Show enough rows to fill viewport + hint of next row

**Poster Card (Default State):**
- Image fills card with radius-lg (12px)
- Shadow-md
- Title below card (not overlaid)
- No overlays, no badges in default state — clean and poster-focused

**Settings Gear Dropdown (Top Toolbar):**
- Triggered by: Lucide Settings icon click (far-right of Top Toolbar)
- Pattern: Standard dropdown menu, aligned to right edge
- Width: 240-280px, --bg-secondary, --shadow-lg, radius-md
- Items (Epic 5 scope only — do NOT render Growth items):

| # | Icon | Label | Description |
|---|------|-------|-------------|
| 1 | Lucide LayoutGrid | 海報大小 | Small / Medium / Large segmented control (adjusts grid columns) |
| 2 | Lucide ArrowUpDown | 預設排序 | Dropdown or radio: 新增日期 / 標題 / 年份 / 評分 |
| 3 | Lucide Languages | 標題顯示語言 | Toggle: 中文優先 / 原文優先 |

- Each item row: Icon (--text-muted, 16px) + Label (--text-primary) + Control on right
- Divider lines between items (1px --bg-tertiary)
- Close on: Click outside, Escape key, or selecting an option
- Mobile: Same dropdown, but may shift to bottom sheet if viewport is narrow

---

### Screen 2: Library Grid View — Tablet (768-1439px)
**Stories:** 5.1, 5.8

Same structure as Desktop with:
- 3-4 columns, minmax(160px, 1fr), 12px gap
- Recently Added: 4-5 visible items
- Top Toolbar: Search shrinks to icon that expands on tap
- Tab Navigation: Same horizontal tabs, slightly tighter spacing
- Sort/Filter controls: May stack vertically on narrower tablets

---

### Screen 3: Library Grid View — Mobile (<768px)
**Stories:** 5.1, 5.8

```
+---------------------------+
| [Vido]        [Search] [cog]|
+---------------------------+
| [Library][DL][Parse][Set] |
+---------------------------+
| Recently Added    [All >] |
| [poster][poster][poster]>>|
+---------------------------+
| All Media                 |
| [Sort v] [Filter v]      |
+---------------------------+
| [poster]  [poster]        |
|  Title     Title          |
| [poster]  [poster]        |
|  Title     Title          |
+---------------------------+
```

- 2 columns, fixed, 12px gap
- Tab labels abbreviated or use icons only
- Recently Added: Horizontal scroll strip
- Sort/Filter: Compact dropdowns or bottom sheet triggers
- No hover effects (touch device)

---

### Screen 4: Poster Hover States + Detail Panel — Desktop
**Stories:** 5.1, 5.6

Show **3 keyframe states side by side** (or stacked with labels):

**Keyframe A — Hover State:**
```
+----------------+
| [8.7]    [...] |  <- Top: Rating badge (left), Menu icon (right)
|                |
|   [Play icon]  |  <- Center: Lucide Play circle, white, large
|                |
| [TV] [10 eps]  |  <- Bottom: Type icon + episode count
+----------------+
  Title (1999)
```
- Card: scale(1.05), translateY(-4px), shadow-xl
- Overlay: Dark gradient (bottom-up, opacity 0.5), badges slide in
- Rating badge: Small pill, --bg-tertiary with --text-primary
- `...` icon (Lucide MoreHorizontal): Top-right, 24px, white, --bg-tertiary/60% pill backdrop
- Accent glow beneath card: hsl(217, 91%, 60%, 0.12)

**Keyframe A-1 — PosterCard Context Menu (on `...` click):**
```
+----------------+
| [8.7]    [...] |
|       +-----------------+
|       | View Details     |
|       | Re-parse         |
|       | Export Metadata   |
|       |------------------|
|       | Delete           |  <- --error red text
|       +-----------------+
+----------------+
```
- Triggered by: Click on `...` icon (desktop), long-press on card (mobile)
- Pattern: Standard dropdown, right-aligned to `...` icon
- Width: 200-220px, --bg-secondary, --shadow-lg, radius-md
- Items (Epic 5 scope only — Growth items NOT rendered):

| # | Icon | Label | Notes |
|---|------|-------|-------|
| 1 | Lucide Eye | 查看詳情 | Opens Detail Panel (Story 5.6) |
| 2 | Lucide RefreshCw | 重新解析 | Re-parse metadata |
| 3 | Lucide Download | 匯出中繼資料 | Export metadata |
| — | *(separator)* | | 1px --bg-tertiary divider |
| 4 | Lucide Trash2 | 刪除 | --error color, confirmation dialog required |

- Each row: Icon (16px, --text-muted; Trash2 uses --error) + Label (--text-primary; Delete uses --error)
- Row hover: --bg-tertiary background
- Delete confirmation dialog: "確定要刪除「{title}」嗎？此操作無法復原。" with [取消] + [刪除] (--error) buttons
- Mobile: Long-press triggers bottom sheet with same items, larger touch targets (min 48px row height)

**Keyframe B — Panel Opening (transition midpoint):**
```
+-------------------------------+------------------+
| Grid (dimmed, opacity 0.7)    | Detail Panel     |
| [poster] [poster]             | (sliding in,     |
| [poster] [poster]             |  partially visible)|
+-------------------------------+------------------+
```
- Annotate: "500ms ease-out, translateX(100% -> 0)"
- Grid backdrop dims to 0.7 opacity

**Keyframe C — Detail Panel Loaded:**
```
+-------------------------------+--------------------+
| Grid (dimmed 0.7)             | [X]          [...] |
|                               |                    |
|                               | [Blurred poster    |
|                               |  backdrop, gradient |
|                               |  fade to bg]       |
|                               |                    |
|                               | Title (large, bold)|
|                               | Original Title     |
|                               | [star] 8.7  2h 16m|
|                               | [Drama] [Action]   |
|                               |                    |
|                               | Director: Name     |
|                               | Cast: A, B, C...  |
|                               |                    |
|                               | Synopsis text...   |
|                               | [Show more]        |
|                               |                    |
|                               | [Play] [+ List]   |
|                               |                    |
|                               | -- File Info --    |
|                               | Source: TMDb       |
|                               | Added: 2026-01-15 |
|                               | Size: 4.2 GB      |
|                               |                    |
|                               | -- Related --      |
|                               | [thumb][thumb][th] |
+-------------------------------+--------------------+
```

**Panel Specifications:**
- Width: 420-500px
- Background: --bg-primary with --shadow-2xl
- Top section: Blurred enlarged poster backdrop (40% height), gradient fade to --bg-primary
- Title: Display font, 24-28px, weight 700-800, --text-primary
- Original title: --text-secondary, 14px
- Rating: Lucide Star icon (filled, --warning color) + score + duration
- Genre tags: Small pills, --bg-tertiary, --text-secondary, radius-full
- Metadata source badge: Small indicator "TMDb" or "Manual" in --text-muted
- Action buttons: Primary (Play) + Secondary (Add to List)
- Close: Lucide X icon, top-right
- `...` icon: Lucide MoreHorizontal, top-right (left of X close button), 24px
- Scrollable independently from grid

**Detail Panel Context Menu (on `...` click):**
```
+--------------------+
| [X]          [...] |  <- `...` left of X
|        +-----------------+
|        | Re-parse         |
|        | Export Metadata   |
|        |------------------|
|        | Delete           |  <- --error red text
|        +-----------------+
```
- Triggered by: Click on `...` icon (desktop), tap (mobile shows bottom sheet)
- Pattern: Same dropdown style as PosterCard menu, right-aligned
- Width: 200-220px, --bg-secondary, --shadow-lg, radius-md
- Items (Epic 5 scope only — Growth items NOT rendered):

| # | Icon | Label | Notes |
|---|------|-------|-------|
| 1 | Lucide RefreshCw | 重新解析 | Re-parse metadata |
| 2 | Lucide Download | 匯出中繼資料 | Export JSON/YAML/NFO |
| — | *(separator)* | | 1px --bg-tertiary divider |
| 3 | Lucide Trash2 | 刪除 | --error color, confirmation dialog required |

- Same styling rules as PosterCard Context Menu (icon + label per row, hover states)
- "View Details" is omitted here (user is already in the Detail Panel)
- Delete confirmation: Same dialog pattern as PosterCard
- Mobile: Bottom sheet menu with large touch targets (min 48px row height)

---

### Screen 5: Detail Panel — Mobile (Bottom Sheet)
**Stories:** 5.6

```
+---------------------------+
| Grid (dimmed)             |
|                           |
+===========================+
| --- drag handle ---       |
| [Blurred poster backdrop] |
|                           |
| Title (bold)              |
| [star] 8.7  2h 16m       |
| [Drama] [Action]          |
|                           |
| Director: Name            |
| Cast: A, B, C...         |
|                           |
| Synopsis...               |
|                           |
| [Play]  [+ List]         |
+---------------------------+
```

- Bottom sheet pattern (swipe up from bottom)
- Half-screen initial state, full-screen on swipe up
- Drag handle indicator at top (small rounded bar)
- Close: Swipe down or tap backdrop
- Same content hierarchy as desktop but full-width

---

### Screen 6: List View — Desktop
**Stories:** 5.2

```
+-----------------------------------------------------+
| [Tabs] All Media          [Sort: v] [Filter] [Grid|List]|
+-----------------------------------------------------+
| [Thumb] Title              Year   Genre      Rating  Added     |
+-----------------------------------------------------+
| [img]   The Matrix         1999   Sci-Fi     8.7    Jan 15    |
| [img]   Spirited Away      2001   Animation  8.6    Jan 14    |
| [img]   Parasite           2019   Thriller   8.5    Jan 13    |
| [img]   Your Name          2016   Animation  8.4    Jan 12    |
+-----------------------------------------------------+
```

- Table layout with sortable column headers (Lucide ArrowUpDown icon)
- Poster thumbnail: Small (40x60px), radius-sm
- Active sort column: --accent-primary text + arrow direction indicator
- Row hover: --bg-tertiary background
- Row click: Opens same Detail Panel as grid view
- Zebra striping: Alternate rows with very subtle --bg-secondary (optional)

---

### Screen 7: Search + Sort + Filter — Desktop
**Stories:** 5.3, 5.4, 5.5

```
+-----------------------------------------------------+
| All Media  [Search within library...]               |
|            [Sort: Date Added v]                     |
|            [Filter v]                               |
+-----------------------------------------------------+
| Active Filters: [Drama x] [2020s x]  [Clear All]    |
| Showing 45 of 500 items                             |
+-----------------------------------------------------+
| [filtered poster grid...]                           |
+-----------------------------------------------------+
```

**Filter Panel (expanded, dropdown or side panel):**
```
+-------------------------------+
| Filters                       |
|                               |
| Genre (dynamic from API)      |
| [動作] [冒險] [✓ 動畫] [喜劇] |
| [犯罪] [✓ 劇情] [家庭] [奇幻] |
| [歷史] [恐怖] [音樂] [懸疑]   |
| [愛情] [科幻] [電視電影] [驚悚]|
| [戰爭] [西部] [運動] [真人秀] |
| [脫口秀] [新聞] [肥皂劇] [兒童]|
|                               |
| Year                          |
| [2020s] [2010s] [2000s] [older]|
|                               |
| Type                          |
| [All | Movies | TV Shows]     |
|                               |
| [Apply]  [Reset]              |
+-------------------------------+
```

**Year Filter — Decade Chip Toggles:**
- Same chip style as genre filter (radius-full pills, same visual states)
- Chips: `[2020s]` `[2010s]` `[2000s]` `[1990s]` `[更早]` (zh-TW) / `[Older]` (en)
- Multi-select: tap to toggle, e.g. selecting `[2020s]` + `[2010s]` shows 2010-2029
- Decade labels are static (not API-driven), derived from library content range
- Active state identical to genre chips: `--accent-primary/15` bg + `--accent-primary` border + Check icon

**Genre Filter — Wrapping Chip Toggle Grid:**
- Pattern: All genres rendered as `radius-full` pill chips in a `flex-wrap` container
- All options visible at once — no "Show more", no hidden content
- Chips: `px-3 py-1.5`, `text-sm`, `font-body` (Noto Sans TC for zh-TW, system sans for en)
- Container: `flex flex-wrap gap-2` (8px gap)
- **Default state:** `--bg-tertiary/30` background, `border border-bg-tertiary`, `--text-secondary`
- **Hover:** `--bg-tertiary` background
- **Active (selected):** `--accent-primary/15` background, `border-accent-primary`, `--accent-hover` text, Lucide Check icon (14px) prefix
- **Interaction:** Single tap/click toggles active state (multi-select)
- **Space budget:** CJK genres (2-4 chars) fit ~7-8 per row at 320px; English genres (4-15 chars) fit ~3-4 per row — both acceptable for 25 items
- **Clear all:** Text button below chip container, `--text-muted`, resets all selections
- Genre list is dynamically populated from `GET /api/v1/library/genres` — mockup values are illustrative only

- Search: Real-time filter as user types, matching terms highlighted in --accent-primary
- Active filter chips: --bg-tertiary pills with Lucide X to remove
- Result count: "--text-muted, e.g., "Showing 45 of 500 items"
- Sort dropdown: Date Added / Title / Year / Rating, each with asc/desc

---

### Screen 8: Batch Operations Mode — Desktop
**Stories:** 5.7

```
+-----------------------------------------------------+
| Batch Toolbar (appears when selection mode active)  |
| [x] 5 selected    [Re-parse] [Export] [Delete]  [Cancel]|
+-----------------------------------------------------+
|                                                     |
| [x][poster] [x][poster] [ ][poster] [ ][poster]    |
|                                                     |
| [ ][poster] [x][poster] [ ][poster] [x][poster]    |
+-----------------------------------------------------+
```

- Trigger: Long-press on card OR click "Select" action from toolbar
- Selected cards: 2px --accent-primary border + checkbox overlay (top-left, Lucide Check icon)
- Unselected cards: Slightly dimmed (opacity 0.8) + empty checkbox
- Batch toolbar: Sticky top, --bg-secondary, shows selected count + action buttons
- Delete button: --error color, requires confirmation dialog
- Progress overlay (during batch operation): "Processing 5 of 20..." with progress bar

---

### Screen 9: Empty Library + Loading Skeleton — Desktop
**Stories:** 5.1

**Empty State:**
```
+-----------------------------------------------------+
| [Tabs: Library active]                              |
+-----------------------------------------------------+
|                                                     |
|              [Film reel illustration]               |
|                                                     |
|         Your media library is empty                 |
|                                                     |
|    Start by connecting qBittorrent or adding        |
|    media files to your watch folder.                |
|                                                     |
|    [Connect qBittorrent]  [Learn More]              |
|                                                     |
+-----------------------------------------------------+
```

- Center-aligned, generous whitespace
- Illustration: Simple, monochromatic line art or subtle icon composition (Lucide Film + FolderOpen), NOT emoji art
- Heading: Display font, --text-primary
- Description: --text-secondary, max 2 lines
- CTA: Primary button (accent) + ghost secondary

**Loading Skeleton:**
```
+-----------------------------------------------------+
| Recently Added                                      |
| [skeleton][skeleton][skeleton][skeleton][skeleton]   |
+-----------------------------------------------------+
| All Media                                           |
| [skeleton] [skeleton] [skeleton] [skeleton] [skel]  |
| [skeleton] [skeleton] [skeleton] [skeleton] [skel]  |
+-----------------------------------------------------+
```

- Poster skeleton: --bg-tertiary, 2:3 ratio, radius-lg, pulsing animation
- Title skeleton: Two gray bars below each card (60% and 40% width)
- Pulsing: Subtle opacity oscillation (0.4 -> 0.7 -> 0.4), 1.5s cycle

---

### Screen 10: Filter & Sort — Mobile
**Stories:** 5.3, 5.4, 5.5

```
+---------------------------+
| [Search...]               |
| [Sort v]  [Filter v]     |
| [Drama x] [2020s x]      |
| 45 of 500                 |
+---------------------------+
| [poster]  [poster]        |
| [poster]  [poster]        |
+---------------------------+

Filter Bottom Sheet (on tap):
+===============================+
| --- drag handle ---           |
| Filters                [Done] |
|                               |
| Genre (dynamic from API)      |
| [動作] [冒險] [✓ 動畫] [喜劇] |
| [犯罪] [✓ 劇情] [家庭] [奇幻] |
| [歷史] [恐怖] [音樂] [懸疑]   |
| [愛情] [科幻] [電視電影] [驚悚]|
| [戰爭] [西部] [運動] [真人秀] |
| [脫口秀] [新聞] [肥皂劇] [兒童]|
|                 [清除全部]     |
|                               |
| Year                          |
| [2020s] [2010s] [2000s] [older]|
|                               |
| Type                          |
| [All | Movies | TV Shows]     |
|                               |
| [Apply]  [Reset]              |
+===============================+
```

- Sort and Filter trigger bottom sheets (not dropdowns)
- Filter chips below search, horizontally scrollable if overflow
- Segmented control for media type (instead of radio buttons)
- Genre list is dynamically populated from `GET /api/v1/library/genres` — mockup values are illustrative only
- **Genre filter pattern: Wrapping Chip Toggle Grid** — same spec as Screen 7 desktop filter panel, with these mobile adjustments:
  - Chip touch targets: min 44px height (`py-2` instead of `py-1.5`)
  - Container width: full bottom sheet width (~375px), fits ~8 CJK chips or ~4 English chips per row
  - All 25 genres visible without scrolling in the genre section (~6 rows CJK, ~8 rows English)
- Touch-friendly: All targets min 44px

---

## 6. Content & Copy (All in Traditional Chinese)

| Element | Text |
|---------|------|
| Tab 1 | 媒體庫 |
| Tab 2 | 下載中 |
| Tab 3 | 待解析 |
| Tab 4 | 設定 |
| Recently Added section | 最近新增 |
| See All link | 查看全部 |
| All Media section | 全部媒體 |
| Search placeholder | 搜尋電影或影集... |
| Library search placeholder | 在媒體庫中搜尋... |
| Sort options | 新增日期 / 標題 / 年份 / 評分 |
| Filter button | 篩選 |
| Filter panel title | 篩選條件 |
| Genre label | 類型 |
| Year label | 年份 |
| Media type label | 媒體類別 |
| Movie | 電影 |
| TV Show | 影集 |
| Apply | 套用 |
| Reset | 重置 |
| Clear All | 清除全部 |
| Showing X of Y | 顯示 X / Y 項 |
| Empty library heading | 你的媒體庫還是空的 |
| Empty library description | 連接 qBittorrent 或將媒體檔案加入監控資料夾即可開始 |
| Empty library CTA | 連接 qBittorrent |
| Batch selected count | 已選取 X 項 |
| Re-parse | 重新解析 |
| Export | 匯出 |
| Delete | 刪除 |
| Cancel | 取消 |
| Detail: Synopsis | 劇情簡介 |
| Detail: Director | 導演 |
| Detail: Cast | 主演 |
| Detail: File Info | 檔案資訊 |
| Detail: Source | 來源 |
| Detail: Added | 新增日期 |
| Detail: Related | 相關推薦 |
| Play button | 播放 |
| Add to list | 加入清單 |
| **Context Menu Items** | |
| View Details | 查看詳情 |
| Re-parse Metadata | 重新解析 |
| Export Metadata | 匯出中繼資料 |
| Delete | 刪除 |
| Delete confirm heading | 確認刪除 |
| Delete confirm message | 確定要刪除「{title}」嗎？此操作無法復原。 |
| Delete confirm button | 刪除 |
| **Settings Gear Items** | |
| Poster Size / Density | 海報大小 |
| Size: Small | 小 |
| Size: Medium | 中 |
| Size: Large | 大 |
| Default Sort Preference | 預設排序 |
| Title Display Language | 標題顯示語言 |
| Language: zh-TW priority | 中文優先 |
| Language: Original priority | 原文優先 |

---

## 7. Sample Data for Design

Use these as realistic content in the design mockups:

| Title (zh-TW) | Original | Year | Genre | Rating | Type |
|---------------|----------|------|-------|--------|------|
| 你的名字 | Your Name | 2016 | 動畫、愛情 | 8.4 | Movie |
| 鬼滅之刃 | Demon Slayer | 2019 | 動畫、動作 | 8.7 | TV |
| 寄生上流 | Parasite | 2019 | 劇情、驚悚 | 8.5 | Movie |
| 進擊的巨人 | Attack on Titan | 2013 | 動畫、動作 | 9.0 | TV |
| 駭客任務 | The Matrix | 1999 | 科幻、動作 | 8.7 | Movie |
| 鈴芽之旅 | Suzume | 2022 | 動畫、冒險 | 7.8 | Movie |
| 咒術迴戰 | Jujutsu Kaisen | 2020 | 動畫、動作 | 8.6 | TV |
| 瀑布 | The Falls | 2021 | 劇情 | 7.2 | Movie |
| 周處除三害 | The Pig, the Snake and the Pigeon | 2023 | 動作、犯罪 | 7.8 | Movie |
| 排球少年 | Haikyu!! | 2014 | 動畫、運動 | 8.8 | TV |

---

## 8. Interaction Design Rules (Context Menus & Dropdowns)

These rules apply to ALL dropdown/context menus in the design:

1. **Scope gating:** Only render items scoped to Epic 5 (1.0). Growth/future items do NOT appear in the UI — they are defined in PRD for future reference only.
2. **Delete always last:** Delete actions always appear as the final item, separated by a divider, styled in `--error` (red) color.
3. **Confirmation required:** All destructive actions (Delete) trigger a confirmation dialog before executing.
4. **Lucide icons mandatory:** Every menu item is prefixed with a Lucide icon (16px, --text-muted; Delete uses --error).
5. **Consistent sizing:** Menu width 200-280px, row height min 36px (desktop) / 48px (mobile).
6. **Dismiss behavior:** Click outside, Escape key, or selecting an item closes the menu.
7. **Mobile adaptation:** Desktop dropdowns become bottom sheets on mobile (<768px), triggered by long-press (cards) or tap (explicit icons).
8. **No nested menus:** Keep menus flat — no sub-menus or flyouts in 1.0.

---

## 9. Design File & Screenshot Workflow

**Source of truth:** `ux-design.pen` (Pencil app, root of project)

**Exported screenshots** are committed to `_bmad-output/screenshots/` organized by user flow:

| Folder | Flow | Screens |
|--------|------|---------|
| `flow-a-browse-desktop/` | Desktop browse | Empty → Loading → Grid → List |
| `flow-b-hover-detail-desktop/` | Desktop hover + detail | Hover → Context Menu → Detail (Movie/TV) → Detail Context Menu |
| `flow-c-search-filter-settings-desktop/` | Desktop search/filter/settings | Search+Filter → Batch Ops → Settings |
| `flow-d-browse-mobile/` | Mobile browse | Empty → Loading → Grid → Sort → Filter |
| `flow-e-interaction-mobile/` | Mobile interaction | Context Menu → Detail → Detail Context Menu |
| `flow-f-batch-settings-mobile/` | Mobile batch/settings | Batch Ops → Settings |

**When the .pen file is updated:**
1. Run `python3 scripts/export-pen-screenshots.py` (requires Pencil.app running)
2. If new screens were added, update the `SCREENS` dict in the script
3. Commit updated screenshots alongside .pen file changes

**Implementation reference:** Developers should reference these screenshots when implementing screens. Each screenshot filename maps to the screen number in this Design Brief (e.g., `01-library-grid-desktop.png` → Screen 1).

---

## 10. What NOT to Do

- NO emoji in navigation, tabs, buttons, or status badges
- NO bright gradient backgrounds or decorative color blocks
- NO generic AI patterns: 4-grid KPI cards, gradient avatar circles, rainbow tags
- NO repeating the same data in multiple places
- NO busy card overlays in the default (non-hover) state — let posters breathe
- NO generic icon-font fallbacks — use Lucide consistently
- NO pure black backgrounds — always use the blue-tinted --bg-primary
- NO overcrowded mobile layouts — 2 columns max, generous touch targets
