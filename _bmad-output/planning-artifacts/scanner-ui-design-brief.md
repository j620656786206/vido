# Scanner UI Design Brief (Flow H)

**Epic:** 7 — Media Library Scanner
**Date:** 2026-03-23
**Author:** Sally (UX Designer)
**Reference:** Epic 7 Stories (7-1 through 7-4), UX Design Specification, Epic 5 Design Brief
**Design File:** `ux-design.pen` (Pencil app — to be updated when screens are built)

---

## 1. Context

Epic 7 delivers the "point at your folders and go" experience — users configure media library paths, trigger recursive scanning, and see real-time progress as files are discovered, parsed, and matched. The scanner UI extends the existing Settings page and adds non-blocking progress indicators so users can continue browsing while scans run.

**Design Philosophy:** Scanning should feel effortless and transparent. The system does heavy work in the background; the UI shows just enough to build confidence without demanding attention.

---

## 2. Screens Needed (4 Screens)

### H1: Scan Trigger & Settings (Desktop)

**Where:** New "媒體庫掃描" section within the existing Settings page (below current settings items)

```
+-----------------------------------------------------+
| Settings                                            |
+-----------------------------------------------------+
| [existing settings sections...]                     |
+-----------------------------------------------------+
| 媒體庫掃描                                          |
|                                                     |
| 媒體資料夾                                          |
| +-----------------------------------------------+  |
| | /volume1/media/movies        [Edit] [Remove]  |  |
| | /volume1/media/tv-shows      [Edit] [Remove]  |  |
| +-----------------------------------------------+  |
| [+ 新增資料夾]                                      |
|                                                     |
| 掃描排程                                            |
| [每小時 v]  (每小時 / 每天 / 僅手動)                  |
|                                                     |
| 上次掃描                                            |
| 2026-03-22 14:30 · 1,247 檔案 · 耗時 3 分 12 秒      |
|                                                     |
| [     掃描媒體庫     ]  ← Primary button, accent    |
+-----------------------------------------------------+
```

**Element Specifications:**

- **Media folder list:** Each row shows path (monospace, JetBrains Mono), with Lucide Pencil (edit) and Lucide Trash2 (remove, --error) icon buttons right-aligned
- **Add folder button:** Ghost button with Lucide FolderPlus icon, --accent-primary text
- **Path input (on add/edit):** Text input with validation — shows Lucide CheckCircle (--success) if path exists, Lucide AlertCircle (--error) if not accessible
- **Schedule selector:** Standard dropdown, --bg-tertiary, radius-md
- **Last scan info:** --text-muted, monospace for numbers
- **Scan Now button:** Full-width primary button, --accent-primary background, Lucide ScanLine icon prefix
- **If scan is in progress:** Button changes to disabled state showing "掃描進行中..." with Lucide Loader spinning icon

---

### H2: Scan Progress (Desktop)

**Triggered by:** Clicking "掃描媒體庫" button or automatic schedule trigger

**Pattern:** Floating card, bottom-right corner (same position as existing AI Parse Progress Card from Epic 5 design brief), non-modal — user can navigate freely.

```
+--------------------------------------+
| 媒體庫掃描中                    [—] [X] |
|                                      |
| ████████████░░░░░░░░  62%            |
|                                      |
| 找到 847 · 解析 524 · 比對 498 · 錯誤 3 |
|                                      |
| 正在處理:                             |
| [Leopard-Raws] Demon Slayer S03...   |
|                                      |
| 預估剩餘: 1 分 42 秒                   |
|                                      |
|              [取消掃描]                |
+--------------------------------------+
```

**Element Specifications:**

- **Card dimensions:** 400px width, --bg-secondary, --shadow-lg, radius-xl (matches existing floating card pattern)
- **Header:** "媒體庫掃描中" in --text-primary, 14px semibold
- **Minimize button:** Lucide Minus icon — collapses to small pill showing "掃描中 62%" with Lucide Loader spinning
- **Close/Cancel button:** Lucide X icon — triggers cancel confirmation
- **Progress bar:** Full width, 6px height, radius-full, --accent-primary fill on --bg-tertiary track
- **Stats row:** Four counters with Lucide icons (Lucide File for found, Lucide FileCheck for parsed, Lucide Link for matched, Lucide AlertTriangle in --error for errors), --text-secondary, monospace numbers
- **Current file:** Single line, truncated with ellipsis, --text-muted, monospace font
- **ETA:** --text-muted, updates every 5 seconds
- **Cancel button:** Ghost button, --text-secondary, requires confirmation: "確定要取消掃描嗎？已處理的結果會保留。" with [繼續掃描] + [取消掃描] (--error)
- **SSE-powered:** All counters and progress update in real-time via SSE stream

**Minimized State:**
```
+-------------------------+
| ⟳ 掃描中 62%  [expand]  |
+-------------------------+
```
- Small pill, bottom-right, --bg-secondary, --shadow-md
- Lucide Loader spinning icon + percentage
- Click to expand back to full card

---

### H3: Scan Results Summary

**Shown after:** Scan completes (success or cancelled)

**Pattern:** Toast notification, top-center, auto-dismiss after 10 seconds

```
+------------------------------------------------+
| ✓ 掃描完成                                [X]   |
|                                                |
| 找到 1,247 檔案 · 比對成功 1,198 · 未比對 42 · 錯誤 7 |
|                                                |
| [查看未比對項目]    [查看錯誤]                     |
+------------------------------------------------+
```

**Element Specifications:**

- **Card:** 480px max-width, centered, --bg-secondary, --shadow-xl, radius-lg
- **Success icon:** Lucide CheckCircle, --success color
- **Error variant:** If errors > 0, icon changes to Lucide AlertTriangle, --warning color
- **Stats:** --text-primary for numbers (monospace), --text-secondary for labels
- **Action links:** Text buttons, --accent-primary, underline on hover
  - "查看未比對項目" → navigates to Library with filter `status=unmatched`
  - "查看錯誤" → navigates to Library with filter `status=error`
- **Auto-dismiss:** 10 seconds with subtle progress indicator (thin line at bottom that shrinks)
- **Dismiss:** Click X, click anywhere outside, or click an action link
- **Cancelled variant:** Header shows "掃描已取消" with Lucide XCircle icon in --text-muted

---

### H4: Scan Progress (Mobile)

**Pattern:** Bottom sheet, peek height 64px, expandable on tap

**Peek State (always visible during scan):**
```
+---------------------------+
| ⟳ 掃描中 62%  847 檔案    |
+---------------------------+
```
- Bottom edge of screen, full width, --bg-secondary, --shadow-lg
- Lucide Loader spinning + percentage + file count
- Tap to expand

**Expanded State:**
```
+===========================+
| --- drag handle ---       |
| 媒體庫掃描中               |
|                           |
| ████████████░░░  62%      |
|                           |
| 找到 847 · 解析 524        |
| 比對 498 · 錯誤 3          |
|                           |
| 預估剩餘: 1 分 42 秒       |
|                           |
|       [取消掃描]           |
+===========================+
```
- Half-screen bottom sheet, drag handle at top
- Simplified: no current file name (save space)
- Stats split into two rows for narrow viewport
- Swipe down to collapse back to peek state

---

## 3. Content & Copy (Traditional Chinese)

| Element | Text |
|---------|------|
| Settings section title | 媒體庫掃描 |
| Folder list label | 媒體資料夾 |
| Add folder button | 新增資料夾 |
| Schedule label | 掃描排程 |
| Schedule: Hourly | 每小時 |
| Schedule: Daily | 每天 |
| Schedule: Manual only | 僅手動 |
| Last scan label | 上次掃描 |
| Scan button | 掃描媒體庫 |
| Scan in progress (button) | 掃描進行中... |
| Progress header | 媒體庫掃描中 |
| Found count | 找到 |
| Parsed count | 解析 |
| Matched count | 比對 |
| Errors count | 錯誤 |
| Current file label | 正在處理 |
| ETA label | 預估剩餘 |
| Cancel button | 取消掃描 |
| Cancel confirm message | 確定要取消掃描嗎？已處理的結果會保留。 |
| Cancel confirm: continue | 繼續掃描 |
| Cancel confirm: cancel | 取消掃描 |
| Scan complete header | 掃描完成 |
| Scan cancelled header | 掃描已取消 |
| Files label | 檔案 |
| Matched label | 比對成功 |
| Unmatched label | 未比對 |
| View unmatched link | 查看未比對項目 |
| View errors link | 查看錯誤 |
| Path not accessible error | 無法存取此路徑 |
| Scan already running toast | 掃描已在進行中 |

---

## 4. Design Constraints

- **Reuse existing patterns:** Settings page layout, floating card (AI Parse Progress), toast notifications
- **Dark theme:** All colors from Epic 5 design system (--bg-primary, --bg-secondary, etc.)
- **Lucide icons only:** ScanLine, FolderPlus, Pencil, Trash2, Loader, File, FileCheck, Link, AlertTriangle, CheckCircle, XCircle, Minus, X
- **SSE-powered progress:** Real-time updates, not polling
- **Non-blocking:** Scan progress must not prevent navigation or library browsing
- **Tailwind CSS:** All styling via Tailwind utility classes

---

## 5. Interaction Rules

1. **Duplicate scan prevention:** If user clicks "掃描媒體庫" while a scan is active, show toast "掃描已在進行中" (--warning), do not start a second scan
2. **Settings auto-save:** Folder path and schedule changes save immediately on change — no separate save button needed
3. **Folder path validation:** Validate on blur; show inline error with Lucide AlertCircle if path is not accessible
4. **Scan results persistence:** Scan results toast links work even after auto-dismiss (results accessible via library filters)
5. **Cancel is safe:** Cancelling preserves all results processed so far — communicate this clearly in the confirmation dialog
6. **Progress card position:** Same bottom-right position as existing AI Parse Progress Card; if both are active simultaneously, stack vertically with 12px gap

---

## 6. What NOT to Do

- NO modal dialog for scan progress — it must be non-blocking
- NO emoji in status indicators or buttons — use Lucide icons
- NO separate "Scan Settings" page — integrate into existing Settings
- NO manual file-by-file scan — always scan entire configured folders
- NO progress polling — use SSE exclusively
