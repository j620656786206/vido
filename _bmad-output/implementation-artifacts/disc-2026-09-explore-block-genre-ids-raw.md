# Bugfix: 自訂首頁的「類型」改成點中文名稱，不再手打 TMDb 代碼

Status: in-progress（Task 0 設計稿待 Alexyu 跑 inline agent）

**Source:** `disc-2026-09-explore-block-genre-ids-raw`（P2，dsr-3 立案）。

## Story

As the person setting up the homepage,
I want to pick a block's genres by name and see them by name,
so that I don't have to look up that 16 means 動畫.

## 病情（2026-10-03 查證）

- 編輯框（`ExploreBlockEditModal`）的欄位是文字輸入「類型 ID（逗號分隔 TMDb genre IDs，可留空）」，placeholder `28,12`。
- 區塊列表（`ExploreBlocksSettings`）的描述印「類型 16」。
- **設計稿 H3（`Paqlk`）本來就畫成可點的類型標籤**（`genreField` `JtzjF`：「類型篩選」＋一排 chip，選中／未選兩種樣式）——是程式沒照稿。
- C10-D（`wnmGh`）／C10-M（`ZjsVs`）的描述被 dsr-3d 改成配合程式印 ID（「類型 16」「類型 878」）——這次改回名稱。
- 名稱來源早就有：`lib/genres.ts` 的 `GENRE_MAP`＋電影 19／影集 16 個 TMDb 類型 ID（修改資訊的類型選單在用）。

## ⚖️ 設計裁定（Sally 2026-10-03）

- 編輯框照 H3：顯示「目前內容類型」的**全部**類型為可切換的 chip（電影 19、影集 16），點一下選／再點取消；可複選；全不選＝不篩類型。
- chip 外觀用 v2 的篩選 chip 語言（探索頁 I1-D-v2 `FilterPanel` 同一套：選中＝金色描邊＋淡金底＋✓、金色字；未選＝`$bg-tertiary`、次要字）。H3 的 chip 配色早於 v2 token（選中是 `$text-primary` 壓 `$accent-primary`，日巡下對比不足），不照抄；不另改 H3。
- 切換「內容類型」電影↔影集：保留兩邊都有的類型（動畫、喜劇、犯罪、紀錄、劇情、家庭、懸疑、西部），其餘清掉——影集沒有「動作」(28)，送了 TMDb 也不認。
- 舊資料裡不在清單上的 ID（以前手打的）：照樣顯示成選中的 chip「ID 123」，不會被悄悄刪掉；點掉才移除。
- 列表描述：「類型 動畫」；多個用「、」：「類型 動作、科幻」；不認得的 ID 印「ID 123」。

## Acceptance Criteria

1. **設計稿（Task 0）**：C10-D 兩列、C10-M 一列的描述改回名稱（見提示詞）；其他不動；截圖只 stage 這兩張。
2. **編輯框**：類型欄改成 chip 組（`role="group"`，每顆 `aria-pressed`），標籤「類型篩選」；存檔仍送原本的逗號分隔 ID 字串（後端格式不變）。
3. **內容類型切換**的保留／清除規則如上；未知 ID 保留。
4. **列表描述**顯示名稱。
5. **測試**：元件 spec（選取／取消、切換內容類型、未知 ID、存檔送出的字串、描述）；gallery 基準重產（darwin 本機、linux 走 CI bootstrap）。

## Tasks / Subtasks

- [ ] **Task 0 — 設計稿（Alexyu 跑 inline agent）**（AC: #1）
- [ ] Task 1 — `lib/genres.ts` 加「依內容類型列 ID」與「ID → 顯示名」（AC: #2–#4）
- [ ] Task 2 — 編輯框 chip 組＋切換規則（AC: #2, #3）
- [ ] Task 3 — 列表描述（AC: #4）
- [ ] Task 4 — 測試＋基準（AC: #5）

## Task 0 提示詞（Sally 2026-10-03；貼給 Pencil Inline AI Agent，跑完 ⌘S）

> **C10 區塊描述改回類型名稱**（只改文字內容，不改任何樣式或結構）
>
> 1. `HKzWT`（C10-D `wnmGh` 裡「高分動畫」那列的描述）：內容改成「電影 · 評分（高→低） · 15 部 · 類型 動畫」。
> 2. `a1Zc8L`（C10-D 裡「經典科幻」那列的描述）：內容改成「電影 · 評分（高→低） · 20 部 · 類型 科幻」。
> 3. `cLjHN`（C10-M `ZjsVs` 裡「高分動畫」那列的描述）：內容改成「電影 · 評分（高→低） · 15 部 · 類型 動畫」。
> 4. 不要動其他節點。`problems` 為空，⌘S 存檔。

跑完請：`python3 scripts/export-pen-screenshots.py`，只 stage `_bmad-output/screenshots/flow-c-search-settings/c10-d.png`、`c10-m.png`＋`ux-design.pen`（＋`pen-tokens.json` 若有變），其他 re-render 雜訊 `git checkout`。

## Dev Agent Record

### Agent Model Used

### Completion Notes List

### File List

## Change Log

- 2026-10-03 建立（Bob＋Sally）。
