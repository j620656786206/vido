# Story 6.10b: 候選列身分 —— 封面、片長並列、未匹配標記（前端）

Status: review — 程式與測試交付（Task 1／2／4 的 spec 部分）；**Task 3（`.pen` + 截圖）與視覺基準線重烤需要你的機器**，見 Completion Notes

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

- [x] **Task 1 — 型別（AC: #1-#3）** — 四個 optional 欄位已於 sub-6-10a 的 PR #386 隨契約 ack 一併加入 `subtitleService.ts`，本 story 直接消費。
- [x] **Task 2 — `CandidateRow` 改造（AC: #1, #2, #3）** — 封面 `<img>` + 首字 fallback、副標兩段並列、未匹配灰標 + tooltip + 標題 hover 顯示原始檔名。
- [ ] **Task 3 — 設計更新 + 截圖（AC: #4）** — ⛔ **需要 Pencil.app**，不在這個沙盒能做。提示詞已備妥（見下）。
- [x] **Task 4 — 測試（AC: #5 的 spec 部分）** — 9 個新測試 + 1 個既有測試改寫到新契約。⛔ **gallery fixture 的視覺基準線重烤需要你的機器與 CI**（見下）。

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

Claude Code on the web（2026-09-05）

### Completion Notes List

**交付**：同意清單的每一列現在有封面（或片名首字，不再是空灰塊）、路線與片長**並列**、
未匹配的列標成「未匹配」且標題改用後端解析出的乾淨片名。

**⚠️ 三點裁量**

1. **片長用 `formatRuntime`（`2 小時 46 分`），不是 AC 例子寫的 `1h 52m`。**
   AC 的例子是英文速記，但 `lib/formatMedia.ts` 的 `formatRuntime` 已經是 PosterCard 與
   詳情頁在用的 zh-TW 形式。同一個產品裡有兩套片長寫法，就是兩個畫面對同一部片說不同話的開始。

2. **`≈` 的判定改讀 `runtime_source`，舊伺服器才退回 `runtimeKnown`。**
   `runtimeKnown` 分不出「檔案實測」與「TMDb 的編輯數字」；`runtime_source` 可以。沒有該欄位的
   伺服器行為與本 story 之前完全一致（有測試釘住）。

3. **未匹配標用 `=== false` 判定。** 舊伺服器不送這個欄位，而「伺服器沒告訴我們」不等於
   「TMDb 找不到」—— 後者才該標。

**⛔ 兩件需要你的機器**

- **AC #4 `.pen` + 截圖**：這個沙盒沒有 Pencil.app。提示詞見下方「.pen inline-agent 提示詞」。
- **AC #5 的視覺基準線**：`generation-consent/list`／`grouped`／`over-budget` 三個 fixture
  渲染候選列，本 story 改了列的長相 ⇒ **六張基準線（3 fixture × darwin/linux）會出現真實視覺差異**。
  其餘 consent fixture（`analyzing`／`empty`／`confirm`／`confirm-over-budget`／`f16-*`／`f19-*`）
  渲染的是別的元件，不受影響 —— 已逐一核對過 fixture 的 `component`。

  **我刻意不先刪掉舊基準線**：那會讓 CI 從「真實差異」降級成「缺基準線」，等於把該由人眼看的
  變更藏起來。CI 會產出 before/after 的 diff artifact，那正是你確認新版列長相的材料 ——
  也正好回答「要不要讓 Sally 進 `.pen`」。

  重烤步驟（確認圖沒問題之後）：
  1. `-darwin`：你的 Mac 上 `pnpm run test:visual:update`，只 stage 那三張。
  2. `-linux`：刪掉那三張 `-linux.png` → Actions 上 dispatch `Visual Regression`（分支選本分支）
     → CI 自動開 bootstrap PR（`--update-snapshots=missing` 只補缺失，所以必須先刪）→ 合併。

### .pen inline-agent 提示詞（AC #4，交給你在 Pencil 跑）

> 在 F15-D-v2（`pwMzT`）與 F15-M-v2（`fdu4y`）的候選列規格上，更新為：
> （a）左側 38×54 的佔位方塊改為**實際海報縮圖**；無海報時仍是同尺寸灰底，但置中放**片名首字**（12px，`--text-muted`）。
> （b）列的副標從單行改為**兩段並列**，中間以 `·` 分隔：前段是路線（`內嵌英文字幕 → 翻譯` 或 `無文字字幕軌 → 語音辨識 + 翻譯`），後段是片長（`2 小時 46 分`；片長未知時為 `≈ 45 分（片長未知）`）。
> （c）標題右側、路線徽章左側，新增一個**中性灰標「未匹配」**（`--bg-tertiary` 底、`--text-muted` 字、12px），只在 TMDb 沒比對到時出現。
> （d）一併補上 sub-6-1 已上線但設計稿沒有的「**資料夾無法寫入**」狀態：checkbox disabled、列 70% 透明度、右側 error tint 徽章。
> 不新增任何設計 token，全部沿用現有列型。

### Discovery Triage

- ⚖️ **既有測試 `[P0] unknown-runtime rows…` 的斷言改寫**：它原本斷言 `片長未知，以 45 分鐘估算`
  這串**單獨成行**的文案 —— 那正是 AC #2 要消滅的東西（它會蓋掉路線那句）。改為斷言兩段都在，
  不是放寬而是換到新契約。
- 查證後**不成立**：`analyzing`／`empty`／`confirm` 等 fixture 不渲染 `CandidateListPanel`，
  不受本 story 影響（逐一核對 fixture 的 `component` 欄位）。

### File List

- `apps/web/src/components/subtitle/consent/CandidateListPanel.tsx`
- `apps/web/src/components/subtitle/consent/CandidateListPanel.spec.tsx`
- ⏳ `ux-design.pen` + `_bmad-output/screenshots/flow-f-subtitle-v2/f15-*`（待你在 Pencil 執行）
- ⏳ 六張視覺基準線（待確認 diff 後重烤）
