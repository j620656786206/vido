# Story 6.11: 同意清單可操作 —— 搜尋、排序、虛擬化、群組摺疊（前端）

Status: ready-for-dev

## Story

As a NAS owner with 2,400 candidates,
I want to find, sort and collapse the consent list instead of scrolling it,
so that choosing what to generate takes seconds on desktop and is still possible on a phone.

## Context

critique P1「2399 列不可操作」。現況：三個路線 chip 是唯一篩選（`CandidateListPanel.tsx:300-305`）；無搜尋、無排序；`<ul>` 一次渲染全部列（`:351-399`）；電影平鋪、影集群組永遠展開。`@tanstack/react-virtual` **已在 `package.json:106`**。PRODUCT.md：手機必須能完成任務。

## Acceptance Criteria

1. **搜尋。** 清單上方 sticky 搜尋框（`h-11`，placeholder「搜尋片名或檔名」），200ms debounce，比對 `display_title`／`title`／原始檔名（大小寫不分、去空白）。命中 0 → 清單區顯示「沒有符合的候選」＋「清除搜尋」。搜尋是**檢視篩選**，與 chip 相乘；全選語意見 sub-6-12。

2. **排序。** 排序選單（`select`，44px）：預設「群組（現況）」；另有「金額高→低」「金額低→高」「片名 A→Z」「未匹配優先」。排序**只改顯示**，提交順序與 F18 可完成數的累計走訪仍用 `groupOrder`（三序同源紅線）——但 F18 的「約 N 部」要對照**顯示**順序時，顯示分隔線（sub-6-12）依提交順序畫，文案註明「依提交順序」。

3. **虛擬化。** 用 `@tanstack/react-virtual` 渲染列與群組標頭（動態高度 `measureElement`）；捲動位置在篩選／排序變更時回頂；2,400 列首次繪製 < 100ms（spec 用 fake timers + 計數斷言 DOM 節點 < 100）。

4. **群組摺疊。** 影集群組標頭加展開／收合；**預設收合**，標頭顯示「已選 x/n · $subtotal」與路線組成（永遠顯示，不是勾了才出現——`:191-199` 改）。電影分兩段「未匹配」「已匹配」可收合。搜尋命中的群組自動展開。

5. **手機。** F15-M：搜尋框與排序放同一列（排序縮成 icon 按鈕開 sheet）；虛擬化在 85vh 抽屜內正常；群組收合狀態記在 dialog 生命週期內。

6. **設計 + 測試。** `.pen` F15-D/M 加搜尋列、排序、收合態；重出截圖。specs：搜尋 debounce 與空結果、排序不改提交順序（斷言 `handleConfirm` payload 順序不變）、虛擬化節點數、群組預設收合／搜尋展開；gallery fixtures：f15-search-hit／f15-collapsed／f15-sorted-cost。

## Tasks / Subtasks

- [ ] **Task 1 — 搜尋與排序 state（container）＋ selector 擴充（AC: #1, #2）**
- [ ] **Task 2 — 虛擬化清單（AC: #3）**
- [ ] **Task 3 — 群組摺疊與永顯路線組成（AC: #4）**
- [ ] **Task 4 — 手機版與設計更新（AC: #5, #6）**
- [ ] **Task 5 — 測試與 fixtures（AC: #6）**

（全前端；後端不動。）

## Dev Notes

- 三序同源紅線（`consentSelection.ts` `groupOrder` 註解）不得破：顯示排序是「投影」，state 順序不動。
- `visibleIds` memo（`:239`）延伸為 `visibleIds = applyRouteFilter ∘ applySearch`；sub-6-12 的全選語意吃這個集合。
- Rule 23：無時鐘讀取。

### Time-dependent visual coverage

- N/A — no wall-clock-reading components touched。

### References

- critique P1「2399 列不可操作」與 Casey／Alex 紅旗；`package.json:106`

## Dev Agent Record

### Agent Model Used

### Completion Notes List

### Discovery Triage

- （dev 填）

### File List
