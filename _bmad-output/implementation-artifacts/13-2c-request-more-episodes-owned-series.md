# Story 13.2c: 已入庫影集也能「想要更多集數」—— 媒體庫影集詳情頁的季／集樹入口

Status: in-progress（Task 0 設計稿待 Alexyu 跑 inline agent）

**Depends on:** `13-2b-partial-request`（季／集樹 `SeasonEpisodeTreeDialog`，PR #656 已合）。**Source:** `disc-2026-10-partial-request-entry-for-owned-series`（P2，13-2b dev 時立案）。

## Story

As a NAS owner who already has season 1 of a show,
I want a way, on that show's library page, to ask for the seasons or episodes I'm missing,
so that partial requests work for the case they exist for — "I have S1, get me S2".

## 為什麼要做（白話）

13-2b 做好了季／集勾選清單，但它只從 TMDb 詳情頁的「想要」打開；只要本機有這部影集的任何一集，那裡就只顯示「已入庫」，而媒體庫的影集頁根本沒有請求按鈕。所以最常見的用法打不開清單。後端（13-2a）早就支援。

## ⚖️ 設計裁定（Sally 2026-10-02；Alexyu 2026-10-02「可以開始畫設計稿」，採推薦方案）

- **入口放在媒體庫影集詳情頁（B4p-D `N2fmG6`）「季與劇集」標題列右側**（`eph` `hMcD2`，水平 space_between，目前只有標題 `jTi5j`——本來就留了右側位置）。不放進 hero 動作列：那列已有 4 顆，而且整頁唯一的實心強調色屬於「管理字幕」。
- **樣式：`Component/Button/Secondary`（`YDPhc`）**＋ `plus` 圖示＋「想要更多集數」，高 44。不用實心金色（一頁一個強調色）。
- **點下去 → 打開 13-2b 的季／集樹（L3-D-v2 `He04g`）**，已入庫的集自動鎖定。
- **狀態沿用 L2（`VH3Tq`）**：這部影集已有進行中的請求 → 換成「已請求 · 處理中」膠囊（$info-tint，不可點）；沒有 TMDb ID（還沒比對成功）→ 不顯示。
- **TMDb 詳情頁的「已入庫」不改**（13-1b 三態語意不動）。
- 手機沒有影集詳情稿；同一個標題列在手機上照樣放得下（標題左、按鈕右），不另畫。

## Acceptance Criteria

1. **設計稿（Task 0）**：B4p-D 的 `hMcD2` 右側多一顆 `btn-request-more`（Secondary、plus、「想要更多集數」、44px）；`problems` 為空；截圖重出、只 stage `flow-b-detail-v2` 裡真的變了的那張。
2. **入口**：`LocalDetailV2` 影集（有 `tmdbId`）的「季與劇集」標題列右側出現該按鈕（`SeasonAccordion` 加一個標題列右側的插槽，三種狀態——載入／錯誤／內容——都顯示）；電影與無 `tmdbId` 的影集不顯示。
3. **行為**：點按鈕 → 13-2b 的樹；確認 → 送部分請求（已入庫的集不會被送出——樹已鎖定、後端 13-2a 兜底）→ 「已加入想要清單」toast；已有進行中請求 → 顯示「已請求 · 處理中」膠囊。全部勾選＋有已入庫集 → 逐集送出，**絕不**送整部（整部會被後端 `REQUEST_ALREADY_IN_LIBRARY` 擋）。
4. **共用元件**：`RequestButton` 加 `variant`（primary／secondary）與 `label`，預設行為與既有呼叫點逐字不變。
5. **13-2b CR 遺留一併處理**（入口打開後這些才會常見）：
   - 確認後焦點回到觸發按鈕所在位置（按鈕變膠囊時，焦點落在膠囊或標題列，不落到 body）。
   - 建立請求失敗時樹不要已經關閉、選取不要遺失：樹在送出中保持開啟並停用確認鈕，成功才關；失敗在樹內顯示後端訊息。
   - 鎖定數只算 TMDb 有列出的集號（絕對集數的動畫不會被誤判成整季已入庫）——coverage 的 owned 集號若超過該季 `episodeCount`，不計入「整季已入庫」判定。
6. **測試＋驗證**：元件 spec（入口顯示條件、狀態、樹內失敗）；gallery fixture＋darwin 基準（linux 走 CI bootstrap）；瀏覽器實測一部已入庫影集補一季。

## Tasks / Subtasks

- [ ] **Task 0 — 設計稿（Alexyu 跑 inline agent）**（AC: #1）——提示詞見下
- [ ] Task 1 — `SeasonAccordion` 標題列插槽＋`LocalDetailV2` 接 `RequestButton`（AC: #2, #3）
- [ ] Task 2 — `RequestButton` variant／label（AC: #4）
- [ ] Task 3 — 13-2b 遺留三項（AC: #5）
- [ ] Task 4 — 測試、fixture、基準、瀏覽器實測（AC: #6）

## Task 0 提示詞（Sally 2026-10-02；貼給 Pencil Inline AI Agent，跑完 ⌘S）

> **B4p-D 加「想要更多集數」入口**
>
> 1. 開啟 `N2fmG6`（B4p-D · 影集詳情）。找到 `sec-eps` 底下的標題列 `eph`（`hMcD2`，水平排列、`justifyContent: space_between`，目前只有一個子節點：標題文字「季與劇集」`jTi5j`）。
> 2. 在 `jTi5j` 的**右邊**（`hMcD2` 的第二個子節點）新增一個 `Component/Button/Secondary`（`YDPhc`）instance，命名 `btn-request-more`：
>    - 高度 `44`（與同頁 hero 動作鈕一致）。
>    - 文字改成「想要更多集數」，字級 `$Type/Body/Size`、字重 `600`、顏色 `$text-primary`。
>    - 文字左邊加一個 icon，圖示 `plus`，`18 × 18`，顏色 `$text-primary`，與文字間距 `$Space/sm`。
> 3. 不要改動 `hMcD2` 以外的任何節點；不要新增 frame。
> 4. 完成後 `placeholder:false`，確認 `problems` 為空，⌘S 存檔。

跑完請：`python3 scripts/export-pen-screenshots.py`，只 stage `_bmad-output/screenshots/flow-b-detail-v2/` 裡 B4p-D 那一張（其他 re-render 雜訊 `git checkout`），連同 `ux-design.pen` 一起 commit。之後我用 MCP 唯讀複審。

## Dev Notes

- 入口元件：`RequestButton`（`pickEpisodes`、新 `variant="secondary"`、`label="想要更多集數"`），`owned={false}`（這裡的語意是「可以再要」，不是「整部已入庫」）、`requested` 取自 `useRequestedMedia`。
- `SeasonAccordion` 現有三個 `<h2>季與劇集</h2>`（loading／error／content）——抽一個 header 共用並加 `action` 插槽。
- 13-2b 的「全部勾選＝整部」紅線只在沒有任何鎖定時成立；已入庫影集一定有鎖定（coverage owned 非空）→ 永遠逐集／逐季送出。coverage 失敗時（undefined）會被當成無鎖定 → 可能送整部 → 後端回 409「此片已在媒體庫中」：在這個入口上要把「coverage 失敗」當成不可送出（樹顯示錯誤、確認停用），不再 fail-soft。

## Dev Agent Record

### Agent Model Used

### Completion Notes List

### File List

## Change Log

- 2026-10-02 建立（Bob＋Sally；合併 `disc-2026-10-partial-request-entry-for-owned-series` 與 13-2b CR 遺留三項）。
