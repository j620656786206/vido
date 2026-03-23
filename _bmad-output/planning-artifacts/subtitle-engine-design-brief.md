# Subtitle Engine UI Design Brief (Flow G)

**Epic:** 8 — Subtitle Engine
**Date:** 2026-03-23
**Author:** Sally (UX Designer)
**Reference:** Epic 8 Stories (8-1 through 8-10), UX Design Specification, Epic 5 Design Brief
**Design File:** `ux-design.pen` (Pencil app — to be updated when screens are built)

---

## 1. Context

Epic 8 is the core v4 MVP differentiator — solving the #1 pain point for Traditional Chinese NAS users: finding and managing zh-TW subtitles. The UI needs to surface subtitle status at a glance, provide a powerful manual search interface, and support batch processing for library-wide subtitle acquisition. All subtitle sources (Assrt, Zimuku, OpenSubtitles) are searched in parallel with results scored and ranked.

**Design Philosophy:** Subtitles should feel like a first-class citizen in the media library, not an afterthought. Status is always visible; actions are always one click away.

---

## 2. Screens Needed (6 Screens)

### G1: Subtitle Status Indicators

**Where:** PosterCard (grid view) + Detail Panel — extends existing Epic 5 components

**PosterCard Badge (Grid View):**
```
+----------------+
|                |
|   [poster]     |
|                |
|          [sub] |  ← Bottom-right corner badge, 20x20px
+----------------+
  Title (2024)
```

**Badge States:**

| State | Icon | Color | Meaning |
|-------|------|-------|---------|
| Found (zh-Hant) | Lucide Subtitles | --success | Traditional Chinese subtitle available |
| Found (zh-Hans) | Lucide Subtitles | --warning | Simplified Chinese only (conversion available) |
| Not found | Lucide SubtitlesOff | --error | No subtitle found after search |
| Searching | Lucide Loader (spinning) | --info | Search in progress |
| Not searched | *(no badge)* | — | Default state, no visual indicator |

**Badge Specifications:**
- Position: Bottom-right of poster image, 4px inset from edges
- Size: 20x20px icon inside 28x28px circle backdrop
- Backdrop: --bg-primary at 80% opacity, for contrast against any poster image
- Only visible on hover (desktop) alongside other hover overlays, OR always visible (configurable via settings in future)
- Mobile: Always visible (no hover state)

**Detail Panel Subtitle Section:**
```
+--------------------------------------------+
| [existing detail content...]               |
|                                            |
| -- 字幕 --                                 |
| 狀態: ✓ 繁體中文字幕已就緒                    |
| 語言: zh-Hant (繁體中文)                     |
| 來源: Assrt                                |
| 檔案: Movie.Name.2024.zh-Hant.srt          |
|                                            |
| [搜尋字幕]                                  |
+--------------------------------------------+
```

**Status Variants in Detail Panel:**

| State | Icon | Status Text | Action Button |
|-------|------|-------------|---------------|
| Found (zh-Hant) | Lucide CheckCircle --success | 繁體中文字幕已就緒 | [重新搜尋] (ghost) |
| Found (zh-Hans) | Lucide AlertTriangle --warning | 簡體中文字幕 (已轉換為繁體) | [重新搜尋] (ghost) |
| Not found | Lucide XCircle --error | 未找到字幕 | [搜尋字幕] (primary) |
| Searching | Lucide Loader --info (spinning) | 正在搜尋字幕... | [取消] (ghost) |
| Not searched | Lucide Minus --text-muted | 尚未搜尋 | [搜尋字幕] (primary) |

**Conversion Indicator:**
- When subtitle was converted from simplified, show: "已從簡體中文轉換 (s2twp)" in --text-muted below the language line
- Lucide ArrowRight icon between "簡體" → "繁體" labels

---

### G2: Manual Subtitle Search (Desktop)

**Triggered from:** Detail Panel "搜尋字幕" button, or PosterCard Context Menu "搜尋字幕"

**Pattern:** Side panel overlay (same pattern as Detail Panel — slides in from right, 500px width)

```
+-------------------------------+--------------------+
| Grid (dimmed 0.7)             | 字幕搜尋            |
|                               |                    |
|                               | 搜尋: [你的名字  ]  |
|                               |                    |
|                               | 來源:              |
|                               | [✓ Assrt] [✓ Zimuku] [✓ OpenSub] |
|                               |                    |
|                               | [自動選擇最佳字幕]   |
|                               |                    |
|                               | --- 搜尋結果 (12) --- |
|                               |                    |
|                               | ▸ 你的名字.zh-Hant.srt        |
|                               |   zh-Hant · Assrt · ████░ 92  |
|                               |   1080p ✓           [下載]    |
|                               |                    |
|                               | ▸ Your.Name.2016.TC.srt       |
|                               |   zh-Hant · Zimuku · ███░░ 78 |
|                               |   1080p ✓           [下載]    |
|                               |                    |
|                               | ▸ Kimi.no.Na.wa.CHS.srt      |
|                               |   zh-Hans · OpenSub · ██░░░ 65|
|                               |   720p ✗           [下載]    |
|                               |                    |
|                               | [載入更多...]        |
+-------------------------------+--------------------+
```

**Element Specifications:**

- **Search input:** Pre-filled with media title (zh-TW), editable, Lucide Search icon prefix, --bg-tertiary
- **Source filter:** Chip toggles (same pattern as genre filter chips from Epic 5), all enabled by default. Disabled source shows tooltip "未設定 API 金鑰" if API key not configured
- **Auto-select button:** Secondary button, Lucide Zap icon — picks highest-scored result and downloads immediately
- **Results list:** Scrollable, sorted by score descending

**Result Row Specifications:**

| Element | Style |
|---------|-------|
| Filename | --text-primary, 14px, truncate with ellipsis |
| Language tag | Pill badge: zh-Hant = --success bg/text, zh-Hans = --warning bg/text |
| Source | --text-muted, 12px |
| Score bar | 60px width, 4px height, --accent-primary fill proportional to score (0-100) |
| Score number | --text-secondary, monospace, 12px |
| Resolution match | Lucide Check (--success) if matches media resolution, Lucide X (--error) if not |
| Download button | Ghost button, Lucide Download icon, --accent-primary |

**Loading State:**
- While searching: Skeleton rows with pulsing animation (same pattern as library loading)
- Header shows "正在搜尋..." with Lucide Loader spinning
- SSE-powered: results appear incrementally as each source responds

**Empty State:**
- "未找到符合的字幕" with Lucide SearchX icon
- Suggestion: "試試修改搜尋關鍵字或啟用更多來源"

---

### G3: Subtitle Download Progress

**Pattern:** Inline progress within the result row (G2) or Detail Panel subtitle section (G1)

**In Search Results (G2):**
```
| ▸ 你的名字.zh-Hant.srt        |
|   zh-Hant · Assrt · ████░ 92  |
|   [████████████░░░] 下載中...  |  ← Download button replaced by inline progress
```

**Download Steps (shown sequentially):**
1. "下載中..." — Lucide Download icon spinning, progress bar
2. "轉換中 簡→繁..." — Lucide ArrowRightLeft icon spinning (only if zh-Hans source, OpenCC conversion)
3. "完成" — Lucide CheckCircle --success, row highlights briefly with --success/10% background

**Element Specifications:**
- Progress replaces the download button inline — no modal or separate view
- Progress bar: Same 60px area where score bar was, --accent-primary fill
- Step label: --text-secondary, 12px
- On completion: Download button changes to Lucide CheckCircle (--success, static)
- On failure: Lucide AlertTriangle (--error) + "重試" text button

**Detail Panel Update:**
- After successful download, the subtitle section in G1 updates in real-time (SSE)
- PosterCard badge also updates without page refresh

---

### G4: Batch Subtitle Processing (Desktop)

**Triggered from:** Library batch toolbar (extend existing batch actions from Epic 5) or Settings page

**Pattern:** Side panel (same as Detail Panel, 500px width) OR modal dialog (480px width, centered)

```
+----------------------------------------------+
| 批次字幕搜尋                              [X] |
|                                              |
| 範圍:                                        |
| (●) 缺少字幕的項目 (42 項)                    |
| ( ) 整季: 鬼滅之刃 第三季 (11 集)             |
| ( ) 整個媒體庫 (1,247 項)                     |
|                                              |
| [     開始批次搜尋     ]                      |
+----------------------------------------------+

Processing state:
+----------------------------------------------+
| 批次字幕搜尋                              [X] |
|                                              |
| ████████████████░░░░  38 / 42                |
|                                              |
| 正在搜尋: 寄生上流 (2019)                      |
|                                              |
| 已找到: 28   未找到: 6   錯誤: 2   剩餘: 6    |
|                                              |
| 最近結果:                                     |
| ✓ 你的名字 — zh-Hant (Assrt)                 |
| ✓ 駭客任務 — zh-Hans→繁 (Zimuku)             |
| ✗ 瀑布 — 未找到                              |
| ✓ 鈴芽之旅 — zh-Hant (OpenSub)               |
|                                              |
| [暫停]    [取消]                              |
+----------------------------------------------+

Completed state:
+----------------------------------------------+
| 批次字幕搜尋完成                          [X] |
|                                              |
| 找到 28 · 未找到 8 · 轉換 4 · 錯誤 2          |
|                                              |
| [查看未找到項目]    [關閉]                     |
+----------------------------------------------+
```

**Element Specifications:**

- **Scope selector:** Radio buttons, --text-primary labels, count in --text-muted parentheses
  - "Missing subtitles only" pre-selected by default
  - "This season" only appears when triggered from a TV show context
- **Start button:** Primary, full-width, --accent-primary, Lucide Search icon
- **Progress bar:** Full width, 6px, --accent-primary on --bg-tertiary
- **Item counter:** "38 / 42" in monospace, --text-primary
- **Current item:** --text-secondary, truncated
- **Stats row:** Four counters with semantic colors (found = --success, not found = --error, converted = --warning, errors = --error)
- **Recent results list:** Scrollable, max 8 visible rows
  - Found: Lucide Check (--success) + title + language + source
  - Converted: Lucide Check (--success) + title + "簡→繁" tag in --warning
  - Not found: Lucide X (--error) + title + "未找到"
  - Error: Lucide AlertTriangle (--error) + title + error reason
- **Pause button:** Ghost, Lucide Pause icon — pauses queue, button changes to [繼續] with Lucide Play icon
- **Cancel button:** Ghost, --text-secondary — confirmation: "確定要取消嗎？已處理的結果會保留。"
- **Completion:** Stats summary + "查看未找到項目" link (filters library to subtitle_status=not_found)

---

### G5: Manual Subtitle Search (Mobile)

**Pattern:** Full-height bottom sheet (swipe up from bottom, same as Detail Panel mobile)

```
+===========================+
| --- drag handle ---       |
| 字幕搜尋              [X] |
|                           |
| [你的名字            🔍]  |
|                           |
| [✓ Assrt] [✓ Zimuku] [✓ OpenSub] |
|                           |
| [自動選擇最佳]            |
|                           |
| --- 結果 (5) ---          |
|                           |
| 你的名字.zh-Hant.srt      |
| zh-Hant · Assrt · 92 分   |
| 1080p ✓          [下載]  |
|                           |
| Your.Name.TC.srt          |
| zh-Hant · Zimuku · 78 分  |
| 1080p ✓          [下載]  |
|                           |
| [查看更多結果...]          |
+===========================+
```

**Mobile Adjustments:**
- Bottom sheet, half-screen initial, full-screen on swipe up
- Show top 5 results by default, "查看更多結果" to load remaining
- Score shown as number only (no bar — save horizontal space)
- Download button: Full-row tap target (min 48px height)
- Source filter chips: Horizontally scrollable if overflow
- Drag handle + X button for dismiss

---

### G6: Batch Processing (Mobile)

**Pattern:** Bottom sheet with progress, peek height 72px

**Peek State:**
```
+---------------------------+
| ⟳ 字幕搜尋 38/42  找到 28 |
+---------------------------+
```

**Expanded State:**
```
+===========================+
| --- drag handle ---       |
| 批次字幕搜尋               |
|                           |
| ████████████░░░  38 / 42  |
|                           |
| 找到 28 · 未找到 6 · 錯誤 2|
|                           |
| 正在: 寄生上流             |
|                           |
| [暫停]    [取消]           |
+===========================+
```

- Simplified: no recent results list (save space)
- Stats on single row
- Tap to expand, swipe down to collapse
- Completion: Peek bar changes to "字幕搜尋完成 — 找到 28 / 42"

---

## 3. Context Menu Extensions

The existing PosterCard and Detail Panel context menus (defined in Epic 5 Design Brief) need new subtitle items:

**PosterCard Context Menu — Add:**

| # | Icon | Label | Notes |
|---|------|-------|-------|
| 2.5 | Lucide Subtitles | 搜尋字幕 | Opens G2 (manual search), inserted after "重新解析" |

**Detail Panel Context Menu — Add:**

| # | Icon | Label | Notes |
|---|------|-------|-------|
| 1.5 | Lucide Subtitles | 搜尋字幕 | Opens G2, inserted after existing items, before separator |

---

## 4. Content & Copy (Traditional Chinese)

| Element | Text |
|---------|------|
| Subtitle section title | 字幕 |
| Status: found zh-Hant | 繁體中文字幕已就緒 |
| Status: found zh-Hans (converted) | 簡體中文字幕 (已轉換為繁體) |
| Status: not found | 未找到字幕 |
| Status: searching | 正在搜尋字幕... |
| Status: not searched | 尚未搜尋 |
| Language label | 語言 |
| Source label | 來源 |
| File label | 檔案 |
| Search button | 搜尋字幕 |
| Re-search button | 重新搜尋 |
| Search panel title | 字幕搜尋 |
| Search input placeholder | 輸入影片名稱搜尋... |
| Source filter label | 來源 |
| Auto-select button | 自動選擇最佳字幕 |
| Results header | 搜尋結果 |
| No results | 未找到符合的字幕 |
| No results suggestion | 試試修改搜尋關鍵字或啟用更多來源 |
| Download button | 下載 |
| Downloading | 下載中... |
| Converting | 轉換中 簡→繁... |
| Download complete | 完成 |
| Download failed | 下載失敗 |
| Retry | 重試 |
| Converted indicator | 已從簡體中文轉換 (s2twp) |
| Batch title | 批次字幕搜尋 |
| Scope: missing only | 缺少字幕的項目 |
| Scope: this season | 整季 |
| Scope: entire library | 整個媒體庫 |
| Start batch button | 開始批次搜尋 |
| Found count | 已找到 |
| Not found count | 未找到 |
| Converted count | 轉換 |
| Errors count | 錯誤 |
| Remaining count | 剩餘 |
| Currently searching | 正在搜尋 |
| Pause button | 暫停 |
| Resume button | 繼續 |
| Cancel button | 取消 |
| Cancel confirm | 確定要取消嗎？已處理的結果會保留。 |
| Batch complete header | 批次字幕搜尋完成 |
| View not found link | 查看未找到項目 |
| Score label | 分 |
| Resolution match | 解析度符合 |
| Load more results | 查看更多結果... |
| API key not configured tooltip | 未設定 API 金鑰 |
| Context menu: search subtitles | 搜尋字幕 |

---

## 5. Design Constraints

- **Subtitle badge size:** 28x28px maximum on PosterCard — must not obscure poster artwork
- **Score visualization:** Horizontal bar (60px width, 4px height) on desktop; numeric score only on mobile
- **Search results:** Horizontal scroll NOT needed — each row is single-column with stacked metadata
- **Dark theme:** All colors from Epic 5 design system
- **Lucide icons only:** Subtitles, SubtitlesOff, Loader, CheckCircle, XCircle, AlertTriangle, Search, SearchX, Download, ArrowRightLeft, Zap, Pause, Play, Check, X, Minus
- **Tailwind CSS:** All styling via Tailwind utility classes
- **SSE-powered:** Search results stream incrementally; batch progress updates in real-time; subtitle status badges update on completion

---

## 6. Interaction Rules

1. **Auto-download (P1-017):** Happens silently in background when new media is added — no UI trigger needed, only status badge update via SSE
2. **Manual search (P1-018):** User-initiated, shows results BEFORE download — user sees language (簡/繁) before committing
3. **Batch processing (P1-019):** Interruptible (pause/cancel), preserves completed results on cancel
4. **Language shown before download:** zh-Hans results clearly marked with --warning color so user knows it will need conversion
5. **Conversion is automatic:** When user downloads a zh-Hans subtitle, OpenCC conversion (s2twp) runs automatically — show "轉換中 簡→繁..." step in progress
6. **Converted indicator:** After conversion, Detail Panel shows "已從簡體中文轉換" — user knows the origin
7. **Failed search:** Show in UI with Lucide AlertTriangle + "重試" button — never fail silently
8. **Concurrent search prevention:** If search is already running for this media item, disable the search button and show "正在搜尋字幕..."
9. **Source unavailability:** If a source API key is not configured, its chip is disabled with tooltip — search proceeds with available sources only

---

## 7. Key UX Decisions

1. **Badge visibility:** On desktop, subtitle badge appears on hover (consistent with other hover overlays). On mobile, badge is always visible. This keeps the default grid clean while ensuring discoverability.
2. **No raw score numbers in badges:** Score is only shown in search results — the PosterCard badge is binary (has subtitle / doesn't have subtitle).
3. **Conversion transparency:** Users see "簡→繁" conversion happened but don't need to approve it. The s2twp profile is the default; advanced users can change it in Settings (future scope).
4. **Search results before download:** Unlike auto-download which is fire-and-forget, manual search always shows results first. This gives users control over which subtitle to pick when the auto-selected one doesn't work.
5. **Batch default scope:** "Missing subtitles only" is pre-selected to avoid re-processing items that already have subtitles.

---

## 8. What NOT to Do

- NO emoji for subtitle status — use Lucide Subtitles/SubtitlesOff icons with semantic colors
- NO star ratings for subtitle scores — use simple horizontal bar or numeric score
- NO nested modals — search panel replaces (not overlays) the detail panel
- NO silent failures — every failed search/download shows in UI with retry option
- NO automatic download of zh-Hans without conversion — always convert to zh-Hant via OpenCC
- NO polling for progress — use SSE exclusively
- NO separate "Subtitle Manager" page — subtitle features are integrated into existing library views
