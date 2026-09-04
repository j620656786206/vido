# Story 6.10b: 候選列身分 —— 封面、片長並列、未匹配標記（前端）

Status: ready-for-dev

**Depends on:** sub-6-10a（`poster_path`、`runtime_source`、`tmdb_matched`、`display_title`）。

## Story

As a BYOK NAS owner scrolling the consent list,
I want each row to show its poster, its title and its route AND runtime side by side,
so that「片長未知」never erases the one line that told me why this row costs money.

## Acceptance Criteria

1. **封面。** `CandidateRow` 的佔位 span（`CandidateListPanel.tsx:86-89`）改為 `<img>`：`getImageUrl(posterPath,'w92')`（`lib/image.ts:19`）、`loading="lazy"`、`alt=""`（裝飾，標題已在旁）；無封面 → 維持既有灰底但**加一個 12px 片名首字**（不是空方塊）。

2. **副標不再互斥。** `routeSubtitle`（`:58-61`）改為兩段並列：「內嵌英文字幕 → 翻譯 · 1h 52m」／「無文字字幕軌 → 語音辨識 + 翻譯 · ≈ 45 分（片長未知）」。`runtime_source=fallback` 時金額前保留 `≈`；`ffprobe`／`tmdb` 時不加。

3. **未匹配標記。** `tmdb_matched=false` → 標題用 `display_title`，右側加中性灰標「未匹配」（tooltip：「TMDb 沒有比對到，片名由檔名解析」）；標題 hover 顯示原始檔名。

4. **設計同步。** F15-D／F15-M（`pwMzT`／`fdu4y`）的列規格更新：封面實圖、兩段副標、未匹配標；依 `CLAUDE.md` 流程重出 `flow-f-subtitle-v2/f15-*` 截圖同 commit。

5. **測試。** specs：三種 `runtime_source` 的副標與 `≈`；封面有／無；未匹配標與 tooltip；gallery fixtures 更新（`-darwin` 本機、`-linux` 等 CI）。

## Tasks / Subtasks

- [ ] **Task 1 — 型別（`subtitleService.ts` additive 欄位，optional 兼容舊伺服器）（AC: #1-#3）**
- [ ] **Task 2 — `CandidateRow` 改造（AC: #1, #2, #3）**
- [ ] **Task 3 — 設計更新 + 截圖（AC: #4）**
- [ ] **Task 4 — 測試 + fixtures（AC: #5）**

## Dev Notes

- **Inherited from sub-6-1:** F15 rows now have an「資料夾無法寫入」state (disabled checkbox + error badge with the blocker as tooltip). The `.pen` F15-D/M update in AC #4 should include this state; the shipped idiom is the route badge in error tint.

- 舊伺服器（無新欄位）→ 行為與今日相同（fallback 分支），不得壞。
- Rule 21 header 已是 F15-D/M/F18 併列，不變。
- 12px 底線（DESIGN.md）：首字佔位與「未匹配」標都 ≥ 12px。

### Time-dependent visual coverage

- N/A — no wall-clock-reading components touched。

### References

- critique P1「列身分崩塌」；sub-6-10a AC #2/#3/#4（ack）

## Dev Agent Record

### Agent Model Used

### Completion Notes List

### Discovery Triage

- （dev 填）

### File List
