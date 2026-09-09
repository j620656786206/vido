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
- [x] **Task 4 — 手機版與設計更新（AC: #5, #6）**
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

**AC #6 `.pen` —— 完成（兩輪）。** 第一輪由 Alexyu 跑 Pencil Inline AI Agent
（提示詞 A/B 在 `sub-6-11-f15-operability-pen-prompt.md`），Claude 以 MCP 複審抓到兩個
**提示詞本身的錯**：(1) 漏寫「影集區塊要搬到最後」，稿面把影集夾在兩段電影中間 ——
而那個段落順序**就是送出順序**，等於在稿子上推翻「已匹配在前」的裁定；
(2) `全面啟動` 被放進未匹配段，但它沒有「未匹配」徽章，它的問題是「資料夾無法寫入」。
第二輪（提示詞 C）Alexyu 裁定由 Claude 直接以 MCP 執行（純 `Move` ×5 ＋ `Update` ×3，
無設計判斷），**破例一次，`.pen` 分工原則不變**。

兩張稿最終順序：桌機 `已匹配 → 沙丘 → 奧本海默 → 全面啟動 → 未匹配 → 星際效應 → 怪奇物語（收合）`；
手機 `沙丘 → 全面啟動 → 星際效應 → 怪奇物語（收合）`（手機不畫電影分段標頭）。

Alexyu 的兩處 deviation 已追認：排序 icon 因 Pencil 不准插進元件實例，改成
label 外包一層框裝 icon＋文字（視覺相同）；未匹配標頭的 checkbox 從複製來的半選
改成空選（`已選 0/1` 配半選會自相矛盾）。

**手機群組標頭換行 —— 裁定維持兩行**（Alexyu 選、Claude 同意）：390px 塞不下
checkbox＋三角＋標題＋徽章＋`已選 3/9 · $0.78`，標題擠成兩行。**不拿掉金額** ——
收合起來那一行是使用者對該劇唯一看得到的東西，「要花多少錢」正是 AC #4 加這一行的理由。

**存檔驗證的坑（寫進 prompt 文件）：** 第二輪改完檔案大小**完全沒變**
（三處改字等長、搬動只是重排），所以 `feedback_verify_pen_saved_before_commit`
教的「看 size」在這種改動上是無效訊號。有效的驗證是跑 `export-pen-screenshots.py`
再看圖 —— 那支腳本讀磁碟檔。全量重出會動到 ~153 張 PNG，只 stage 真的改到的兩張。

**視覺基準（實際走法與原計畫不同）。** darwin 基準本機重生；`-linux` 原本 `git rm` 掉
想讓 CI bootstrap 補，**但 bootstrap 沒跑成**：它的門檻是「純缺少、無像素差異」，而同一輪
裡有一個**與本 story 無關**的 fixture（`media-media-detail-panel/focus`，背景漸層帶 7% 差異，
既有的 `disc-flaky-visual-media-detail-panel` 抖動）讓 `bootstrap_needed=false`，八張缺的
一張都沒補。改走 `infra_visual_regression_genuine_diff_baseline` 的手動路徑：
從 CI run 的 artifact 下載 `-actual.png`、逐張看過、存成 `-visual-linux.png` 提交
（commit `b7abc42d`）。**沒有碰那個抖動的 fixture。**

**Fixture 高度上限（新發現，已寫成 memory）。** `visual` project 的視窗是 1280x800；
fixture 一旦高過 800，gallery 頁面自己會捲動、app shell 的 sticky header 壓在 fixture 頂端，
落點在兩次 render 之間不重現 → CI 隨機 1% 差異。加了 44px 搜尋列之後
`over-budget` 764→820、`f15-sorted-cost` 867、`list-mobile` 848→920 全部破線。
修法：over-budget 5→4 列、sorted-cost 6→5 列、`list-mobile` 拆成
`list-mobile`（列版式）+ `list-mobile-groups`（群組標頭），八張回到 479–779。
見 `project_visual_fixture_viewport_ceiling`。

**對抗式 CR（`/code-review high`，換模型）——六項，四項當場修：**

1. **（MEDIUM，真 bug）搜尋中點收合會反向。** `isExpanded` 原本先看 `searching` 就回 true，
   但收合鈕仍可點，而點擊寫的是 `!(override ?? default)` = `!false` = **true**。
   使用者點「收起來」→ 畫面沒反應、狀態卻記成「展開」，搜尋清掉後那部劇是**開的**。
   修法兩處：`isExpanded` 改成 override 優先（搜尋的自動展開降級為 default），
   `toggleSection` 改成反轉**畫面上看得到的** `row.expanded`。
2. **（LOW/MED）`measureElement` 讀到被 transform 縮放的高度。** 手寫的
   `getBoundingClientRect().height` 讀的是視覺框，而 host dialog 開場動畫是
   `scale(0.96→1)`；首次量測會把每一列記矮約 4%，而 transform 不改 layout box 所以
   ResizeObserver 不會再觸發。改用套件預設（優先 `borderBoxSize`，退回 `offsetHeight`）。
3. **（MEDIUM）矮視窗下清單會被壓成 0。** 固定控制區在 85vh 的短視窗（~450px 高的筆電視窗、
   橫向手機）會吃掉全部空間。清單加 `min-h-[10rem]` 地板、body 加 `overflow-y-auto` 兜底。
4. **（MEDIUM，不修，立案）群組勾選作用於搜尋隱藏的列。** 搜尋「S04E07」只看到一集，
   勾標頭卻同意整部 9 集（$2.79）。**sub-6-12 明訂擁有全選/群組語意**，在這裡改會
   越權且抵觸 sub-5-3 已測試的語意。已在 `onToggleGroup` 的註解寫明危害與立案歸屬。
5. **（LOW）標頭兩個數字的分母不同。** 路線徽章數可選取的列、`已選 x/n` 的 n 數全部列 ——
   9 集有 2 集資料夾不可寫時，同一行會並排「語音辨識 7」與「已選 0/9」。分母改成可選取數，
   與徽章和標頭 checkbox 的 all 判定同源。
6. **（LOW）電影段落的 aria-label 是「選取整部 已匹配」。** 段落是一堆電影不是一部，
   改成「選取所有已匹配的電影」／「選取所有未匹配的電影」，影集與季維持原文案。

另接受一項效能建議：`getItemKey` 包 `useCallback`（原本每次 render 都是新 closure，
2,400 列時每次按鍵都重算 2,400 個 offset）。四項修正各自補了迴歸測試，測試數 +7。

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
- `ux-design.pen`（F15-D-v2 `pwMzT` / F15-M-v2 `fdu4y`）
- `_bmad-output/screenshots/flow-f-subtitle-v2/f15-d-v2.png`、`f15-m-v2.png`
