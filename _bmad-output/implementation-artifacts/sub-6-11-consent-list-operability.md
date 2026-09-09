# Story 6.11: 同意清單可操作 —— 搜尋、排序、虛擬化、群組摺疊（前端）

Status: review

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

- [x] **Task 1 — 搜尋與排序 state（container）＋ selector 擴充（AC: #1, #2）**
- [x] **Task 2 — 虛擬化清單（AC: #3）**
- [x] **Task 3 — 群組摺疊與永顯路線組成（AC: #4）**
- [x] **Task 4 — 手機版（AC: #5）**；設計更新（AC: #6 `.pen`）提示詞已產出，**待 Alexyu 執行**
- [x] **Task 5 — 測試與 fixtures（AC: #6）**

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

Claude Opus 5 (1M context)

### Completion Notes List

**AC #1 搜尋。** 搜尋 state 在 container（`GenerationConsentView`），200ms debounce，
`searchText`（輸入框顯示）與 `searchQuery`（清單過濾）分開。比對欄位是
`displayTitle` ＋ `title` ＋ `seriesTitle`，大小寫與空白皆不敏感。

> **偏離 AC 字面 —— 沒有「原始檔名」可比對。** `GenerationCandidate` 的 wire schema
> **沒有 `file_path` 或任何檔名欄位**（`subtitleService.ts:233-293` 逐欄確認）。未匹配列的
> `title` 就是使用者在檔案總管看到的那串原始字串，所以搜尋 `title` 已是最接近檔名的作法；
> 加真正的檔名欄位是後端變更，本 story 明訂「後端不動」。placeholder 因此寫
> **「搜尋片名或影集」**而非「片名或檔名」——寫得到的才寫。
> `seriesTitle` 是額外加的：集數的 `title` 是 `S04E07` 或 `集名 S04E07`，**永遠不含劇名**
> （`generation_candidates.go:1257-1265`），不比對 `seriesTitle` 的話，在以影集為主的片庫裡
> 打劇名會零命中。

**AC #2 排序。** 五種順序在 `consentRows.ts`。**顯示排序是投影，state 順序不動**：
`groupOrder` 仍是送出順序與 F18 可完成數的走訪順序，並有 container spec
斷言 `onStartBatch` 的 id 順序在 `cost-desc` 之下不變。非 `group` 排序會**攤平**清單
（金額排序本質上就會把影集與電影交錯，留著標頭沒有意義）。排序生效時，F18 banner
會多一句 **「（依提交順序計算，不是目前的排序）」**。

**AC #3 虛擬化。** `@tanstack/react-virtual`，`measureElement` 量真高度、`gap: 8`
讓 offset 與 flexbox 畫出來的一致。**80 列以下不虛擬化**（`VIRTUALIZE_FROM_ROWS`）：
虛擬化會拿走瀏覽器的 Ctrl-F、列印與捲動錨定，百列以下買不到任何東西。兩條路徑
render 完全相同的 row 元件、吃同一份 `ConsentRow` 陣列，差別只在切片。
2,400 列 spec 斷言 DOM 節點 < 100（jsdom 不做版面，spec 內 stub 了 `offsetHeight`
與 `getBoundingClientRect`，否則測試會因為「視窗高度 0」而假通過）。
篩選／搜尋／排序變更時捲動回頂。

**版面重構。** body 從「整片捲動」改成「控制列固定 + 只有清單捲動」。原本 2,400 列時
搜尋框、chips 與全選在使用者閱讀位置的 40 個畫面之上；這同時也是虛擬化需要的
可量測捲動容器。代價是四張既有視覺基準改變（見下）。

**AC #4 收合。** 影集**預設收合**、電影段落與季**預設展開**；`expandedOverride`
只記使用者動過的段落，活在 panel 生命週期內（AC #5）。搜尋進行中所有仍在畫面上的
段落強制展開 —— 沒有命中的段落整段不 render，所以「命中的段落」與「畫面上的段落」
是同一個集合。**路線組成改成整段的組成且永遠顯示**（原本要勾了才出現），仍走
`computeTotals`（CR H1 的「不得第二套加總」不破）—— 對整段的可選取 id 各算一次。

**電影分兩段：`已匹配` 在前。** Alexyu 2026-09-09 裁定（我提的兩案）。理由：這段順序
也是送出順序，超出預算時錢先花在片名已確定的片上；未匹配沉底但有自己的標頭。
只有「兩邊都非空」時才分段 —— 全部匹配、全部未匹配、以及**任何 pre-sub-6-10a 伺服器**
（根本不送 `tmdb_matched`）都維持既有的無標頭平鋪。

**AC #5 手機。** 搜尋框與排序同一列。排序用**一個原生 `<select>`**：手機上平台自己會
把它開成挑選器 sheet，那就是 AC 要的 sheet，而且比手刻的好。

**AC #6 測試與 fixtures。** 新 spec `consentRows.spec.ts`（19）＋ 搜尋（8）＋
面板 UI（16）＋ container（4）。gallery fixtures 新增
`f15-collapsed` / `f15-search-hit` / `f15-sorted-cost`，darwin 視覺基準已產生。

**AC #6 `.pen` —— 未完成，待 Alexyu。** 依 `feedback_pen_inline_agent_workflow`，
`.pen` 由 Alexyu 跑 Pencil Inline AI Agent 執行、Claude 以 MCP 複審。節點錨定的提示詞
（含 F15-D-v2 / F15-M-v2 兩段、確切 node ID、定稿字串、複審清單）已寫在
`sub-6-11-f15-operability-pen-prompt.md`。截圖重出也在那份文件的最後一節。

**視覺基準。** 四張既有 darwin 基準（`list` / `list-mobile` / `grouped` / `over-budget`）
因版面重構改變，已本機重生；對應的 `-linux` 已 `git rm`，讓 CI 的 bootstrap 走「缺少」
那條路（`project_visual_baseline_intentional_change`）。三張新 fixture 的 `-linux`
同樣缺少，一併由 bootstrap 產生。

### Discovery Triage

- **F18「約 N 部」與排序的語意衝突**（本 story 內修）：可以依金額排序之後，
  「預計可完成約 N 部」數的是提交順序、使用者看的是另一個順序。已在 banner 加註
  「依提交順序計算」。sub-6-12 的砍線分隔列要沿用同一句話。
- **沒有檔名欄位可搜尋**（立案候選）：`GET /subtitles/generation-candidates` 不送
  `file_path`。使用者想用檔名找片（例如記得 `1080p.BluRay` 但不記得中文片名）時，
  只有未匹配列做得到。若要做，是 additive 的後端欄位 + FE 加進 `candidateSearchText`。

### File List

- `apps/web/src/components/subtitle/consent/consentRows.ts`（新）
- `apps/web/src/components/subtitle/consent/consentRows.spec.ts`（新）
- `apps/web/src/components/subtitle/consent/consentSelection.ts`
- `apps/web/src/components/subtitle/consent/consentSelection.spec.ts`
- `apps/web/src/components/subtitle/consent/CandidateListPanel.tsx`
- `apps/web/src/components/subtitle/consent/CandidateListPanel.spec.tsx`
- `apps/web/src/components/subtitle/consent/GenerationConsentView.tsx`
- `apps/web/src/components/subtitle/consent/GenerationConsentView.spec.tsx`
- `apps/web/src/routes/test/-gallery.fixtures.tsx`
- `tests/visual/components.visual.spec.ts-snapshots/components/generation-consent/*`
- `_bmad-output/implementation-artifacts/sub-6-11-f15-operability-pen-prompt.md`（新）
