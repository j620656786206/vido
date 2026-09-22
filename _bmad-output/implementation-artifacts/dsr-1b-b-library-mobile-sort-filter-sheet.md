# Story DSR.1b-b：手機媒體庫的「排序與篩選」抽屜對齊設計稿——有字幕篩選、看得到會篩出幾部、套用鈕固定在底部

Status: ready-for-dev

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As the person who browses 媒體庫 on a phone,
I want 一個抽屜就能排序、挑類型、挑年份、挑字幕狀態，按下去之前就知道會篩出幾部，而且「查看未找到項目」那條連結真的只列缺字幕的片,
so that 我不用在小螢幕上跟一個為滑鼠設計的下拉選單搏鬥，也不用在幾百部片裡自己找哪些還缺字幕。

## Context

`dsr-1b`（Flow A 手機）拆出來的**第二塊：抽屜＋字幕篩選的前端接線**。**依賴 `dsr-1b-a` 先合併**（後端 `GET /library?subtitle_status=` `[@contract-v1]`）。第三塊 `dsr-1b-c`（四張手機畫面的版面）在本張之後做——兩張都會動 `LibraryBrowseV2` 的標題列，排好序免衝突。

同時收掉 `disc-2026-06-library-subtitle-status-filter` 的**前端半**，並把 `disc-2026-09-library-has-filters-the-product-lacks` 裡 A6p-M 那兩整組「稿上有、產品沒有」的篩選收成一組（字幕：做；解析度：從稿上拿掉）。

| 稿 | 節點 | 是什麼 | 程式碼現況 |
| --- | --- | --- | --- |
| A6p-M · 排序＋篩選 Sheet（手機） | `Bz0YN` → `VebkY` | 一個抽屜：標題「排序與篩選」＋「重設」；四個分區（排序 4／類型 7／解析度 3／字幕 3）；底部「套用篩選 · 128 部」 | `LibraryFilterSheetV2.tsx`（69 行）：`ui/Sheet` 裡塞 `SortSelector`（**桌機下拉，popup 套 popup**）＋ `FilterPanel` 批次模式（自己的「套用／重置」按鈕**跟著內容捲動**，沒有計數） |
| A3p-M · 內容網格（手機） | `h1v1U6` | 頂列右側 44×44 `sliders-horizontal` 圖示鈕是抽屜唯一入口 | 工具列裡一顆帶字的「篩選」按鈕（`lg:hidden`，`:593-605`），沒有 `aria-haspopup`、沒有計數徽章；膠囊列在手機 `flex-wrap` 換行（`:629`） |

⚠️ **驗收基準是 `.pen` 節點值**。PNG 只是參考。
⛔ **本張不進**：後端（`-a`）、A1p-M／A2p-M／A3p-M 頂列文字與計數、骨架、網格、E4-M（全部 `-c`）、桌機 ≥1024 的任何版面（篩選軌、`SortSelector` 在工具列的長相一個像素都不能變）、解析度篩選（沒有資料，稿上拿掉）、把手畫法（`disc-2026-09-bottomsheet-grabber-three-variants`）、關閉鈕（`disc-2026-09-ui-sheet-no-close-button`）。

### 🔴 建單時查到的事（main `49cc74a3`；行號皆為現況）

**抽屜本體**

1. `LibraryFilterSheetV2.tsx`（69 行）：props `open`／`onOpenChange`／`sortBy`／`sortOrder`／`onSortChange`／`filters`／`mediaType`／`unmatchedCount?`／`onApply`／`onClear`／`onTypeChange`；render 是 `<Sheet title="排序與篩選">` 裡兩個 `<section>`——「排序」放 `SortSelector`（`:46`）、「篩選」放 `FilterPanel` 批次模式（`onApply` 後關閉）。**沒有 spec、沒有視覺夾具**——是 `library/` v2 元件裡唯一零覆蓋的一個。檔頭 `// Implements: Component/MergedSortFilterSheet (Bz0YN)` **形式錯**：`Bz0YN` 是畫面 A6p-M，`.pen` 裡沒有叫 `MergedSortFilterSheet` 的母版（dsr-1 CR 第 2 項同類；當時明文留給 dsr-1b）。
2. `ui/Sheet.tsx`（101 行，dsr-4b-1 之後）：`open`／`onOpenChange`／`title`／`ariaLabel`／`testId`（預設 `bottom-sheet`）／`className`／`titleClassName`／`description`／`descriptionClassName`／`finalFocus`／`onOpenChangeComplete`。Popup 基底 `fixed inset-x-0 bottom-0 z-[71] max-h-[85vh] overflow-y-auto … pb-[max(1rem,env(safe-area-inset-bottom))]`。**沒有 footer 插槽、沒有關閉鈕、沒有 snap**。📎 `p-0` 會連 safe-area 的 `pb-[…]` 一起丟掉（同一個 twMerge 群組，`:32-36` 有註解）——`DownloadSortSheet.tsx:64` 用 `className="p-0 pt-2 pb-[max(1.25rem,env(safe-area-inset-bottom))]"` 補回。**「內容捲、底部固定」要靠 `className` 把 Popup 改成 `flex flex-col overflow-hidden`**（dsr-4b-2 詳細資訊抽屜的做法，看 `downloads/DownloadDetailSheet.tsx` 怎麼排 header／捲動區／footer）。
3. `SortSelector.tsx:15-18` 的 `SORT_OPTIONS`：`created_at`「新增日期」desc／`title`「標題」asc／`release_date`「年份」desc／`rating`「評分」desc（各有 `defaultOrder`）。稿的第二顆寫「**片名**」——碼贏（「標題」是桌機下拉正在用的字）。`SORT_OPTIONS` 目前是不是 `export` 要看檔頭；4b-1 的先例是加 `export`、內容不變、抽屜 import 同一份。
4. `FilterPanel.tsx`（356 行）有**兩種模式**：`instant`（桌機篩選軌，改了就 `emitInstant` `:128-139`，**沒有按鈕**）與批次（本地 draft＋「套用」`filter-apply`／「重置」`filter-reset`，`:300-320`）。任何新欄位要同時過：本地 state＋`useEffect` 同步（`:110-118`）、`selected*` 推導（`:121-126`）、`emitInstant`、`handleApply`（`:166-174`）、`handleClear`（`:176-181`）——漏一處桌機軌就靜默丟掉那個篩選。既有分區：媒體類型 chips（全部／電影／影集）、類型（genres，來自 `useLibraryGenres`）、年代（`DECADE_OPTIONS`；`normalizeDecadeSelection` `:77-88` 會自動補中間的年代，因為後端只吃 min／max）、未匹配 toggle（`filter-unmatched`，`:287-297`，帶 `unmatchedCount`）。標題層級在抽屜裡是 `h2 → h4 → h3`（`:183-186` 註解；`disc-2026-08-rails-a11y-landmarks-and-unreachable-disabled`）。
5. `FilterValues`（`FilterPanel.tsx:7-12`）：`genres: string[]`／`yearMin?`／`yearMax?`／`unmatched?`。消費者：`FilterChips.tsx:4`、`LibraryFilterSheetV2.tsx:10`、`LibraryFilterRail.tsx:15`、`LibraryBrowseV2.tsx:45`（組 `filters` `:166-174`、`hasActiveFilters` `:175-179`、`activeFilterCount` `:182-185`、`activeFilterLabels` `:192-198`）、夾具 `-gallery.fixtures.tsx:250`。

**字幕篩選的接線（今天斷在三個地方）**

6. `routes/library.tsx:17-22, 49`：`subtitleStatus?: string` 有解析、有保留、**沒人讀**；`:17-22` 的註解寫「not yet wired」——接上之後**這段註解要刪**，否則變成反方向的謊言。字串 enum 守門在 Rule 26 是安全的（值永遠不會是純數字）。
7. `services/libraryService.ts:65-78` `listLibrary` 送 `page`／`page_size`／`type`／`sort_by`／`sort_order`／`genres`／`year_min`／`year_max`／`unmatched`——**沒有 `subtitle_status`**。`LibraryListParams` 在 `types/library.ts:246-256`。`hooks/useLibraryInfinite.ts` 的 key 是手寫的 `['library','infinite',params]`（`:18`），**不走 `libraryKeys`**——夾具的 `seedQueries` 要用同一把 key 才打得到（`-gallery.fixtures.tsx:3073-3079` 的註解就在講這件事）。
8. `BatchSubtitleDialog.tsx:361` 是唯一的產生端：`navigate({ to: '/library', search: (prev) => ({ ...prev, subtitleStatus: 'not_found' }) })`（`BatchSubtitleDialog.spec.tsx:189` 蓋著）。
9. 🚨 **「繁中」徽章不是 `subtitle_status` 算出來的。** `utils/libraryStatus.ts:170-216` 的 `deriveSubtitleStatus` 是一條梯子：`no_text_source`／`skipped`／`untranslated` 先判；三個進行中的值回 `null`；**`not_searched`／`searching`／`found` 都掉到「看內嵌音軌」**（`deriveFromTracks`）；只有「引擎搜過沒找到、又沒有內嵌軌」才是「缺字幕」。所以**後端按 `subtitle_status` 欄位篩，篩出來的集合與海報徽章的「繁中／簡中」不會一對一**——稿上的「繁中／簡轉繁」是徽章的語言，不是狀態。抽屜的字幕 chip 只能用**狀態**的話說（裁定 3）。
10. 後端合法值 10 個（`models/movie.go:118-131`）；`-a` 接受 CSV。`found` 的意思是「引擎找過、找到了」，不是「有繁中」。

**入口、膠囊列、生命週期**

11. 篩選入口 `LibraryBrowseV2.tsx:593-605`：`lg:hidden` 按鈕 `library-filter-open`（`SlidersHorizontal`＋「篩選」），沒有 `aria-haspopup="dialog"`／`aria-expanded`；計數徽章只在桌機的軌展開鈕上（`:620-624`），`activeFilterCount` 早就算好、只是沒傳給手機那顆。
12. **兩個斷點**：軌／抽屜切在 `lg`（1024，`:533, 597`）；`useIsPhone` 是 `sm`（640）。640–1024 之間「不是手機」但用抽屜。本張不換斷點：**抽屜仍是 <1024 的東西**；手機頂列的圖示鈕只是 <640 的另一個入口。
13. 生命週期：沒有 `finalFocus`（關掉焦點落到 `<body>`）、視窗放大跨過 `lg` 時抽屜不會關。`DownloadsBrowseV2.tsx:355-373` 是照抄範本：render 階段偵測到條件不成立就 `setOpen(false)`、`finalFocus` 函式版依序找「還顯示的觸發鈕 → `<h1 tabIndex={-1}>`」。
14. 膠囊列 `:629` 在手機是 `order-last w-full` 換到自己一列再 `flex-wrap`。dsr-4b-1 的單行捲動寫法在 `DownloadsBrowseV2.tsx:478-500`（`max-sm:-mx-4 max-sm:-my-1 max-sm:flex-nowrap max-sm:overflow-x-auto max-sm:overscroll-x-contain max-sm:scroll-px-4 max-sm:px-4 max-sm:py-1 [scrollbar-width:none] [&::-webkit-scrollbar]:hidden`，膠囊 `max-sm:shrink-0`）＋ `:164-175` 把選中膠囊捲進列內的 effect。**`-my-1 py-1` 是替 4px 焦點框留的**（`overflow-x:auto` 會連帶裁掉上下）。
15. `unmatchedCount`（`:225-226`）是**全域**的（movie＋series stats 相加、不看 `currentType` 與其他篩選）——抽屜裡「未匹配 (12)」可能大於當下的結果數。本張不修它、也**不要**替字幕 chip 加同一種全域計數；footer 的數字走裁定 4。

**測試現況**

16. Vitest：`LibraryBrowseV2.spec.tsx`（三個 describe，測試名帶 `[P0]`／`[P1]`）、`FilterPanel.spec.tsx`、`FilterChips.spec.tsx`、`LibraryFilterRail.spec.tsx`、`SortSelector.spec.tsx`、`LibraryStatesV2.spec.tsx`；**沒有 `LibraryFilterSheetV2.spec.tsx`**。抽屜的先例：`DownloadSortSheet.spec.tsx`、`DownloadActionsSheet.spec.tsx`。spec 檔頭 mock hook 的寫法：`vi.mock('../../hooks/useIsPhone', () => ({ useIsPhone: () => h.isPhone }))`。`test-setup.ts` 的全域 `matchMedia` 回 `false`＝桌機路徑。
17. e2e：**沒有任何 `library-*.spec.ts`**（最近的是 `empty-library.spec.ts`、`poster-card-hover.spec.ts`）。手機 e2e 範本 `tests/e2e/downloads-mobile.spec.ts`（`PHONE` 常數、每條自己 `setViewportSize`、`[P0] 390 — …`／`[P1] 640 — …` 命名、`shot()` 只在 `DSR_SHOTS=1`）。CI 只跑 `chromium`＋`webkit-core`。
18. 視覺夾具：`library-*` 現有 19 個（`library-filter-panel` `:3058-3080` 示範 `seedQueries: [{ queryKey: libraryKeys.genres(), data: [...] }]`），**沒有抽屜的**。抽屜夾具範本 `downloads-mobile-sheets/sort`（`:1781-1798`：`viewport: { width: 390, height: 844 }`、`penNode` 必填、`statesOnly: ['default']`）。Base UI 抽屜是 Portal → 夾具 state div 高度 0 → 視覺 spec 拍整個視窗。

### 設計稿節點（A6p-M `Bz0YN`，390×844，位於 17040 群組 `JUkwx` 內第四張）

| 節點 | 內容 |
| --- | --- |
| `lJGih` backdrop | 390×844 `$overlay-scrim` |
| `VebkY` sheet | y=200、390×644（76%）、`$bg-secondary`、頂角 `$radius-xl`、vertical、無 padding、**不是 `Component/BottomSheet`（`SG1ln`）的 instance**（手畫） |
| `grab` → `h` | 40×4 `$border-subtle`（＝`ui/Sheet` 今天畫的那一種；三種把手的裁定另案） |
| `hd` | 「排序與篩選」H4 18／700 `$text-primary` @ (16,4)；右側「重設」Body 14／600 `$accent-text`（純文字，**沒有 44 觸控區**） |
| `div` | 1px `$border-subtle` |
| `R1alM3` body | 390×492、padding 16、分區 gap `$Space/lg-plus` 20、分區內標題→列 gap 8、列距 8；**不是捲動容器**（內容剛好放完） |
| 分區標題 | Label 12／600 `$text-secondary` |
| 排序 `sortrow` | 4 顆 **方角** chip（`$radius-md`、高 44、padding [0,12]、gap 6）：「新增日期」已選（`$accent-subtle`／600／`$accent-text`＋前置 `arrow-down` 14）、「片名」「年份」「評分」未選（`$bg-tertiary`／400／`$text-secondary`） |
| 類型 | 7 顆 **pill** chip（`$radius-pill`、高 38、padding [8,14]、gap 8）：動畫（選）／動作／劇情／科幻 ‖ 愛情／懸疑／紀錄片 |
| 解析度 | 4K（選）／1080p／720p —— **要拿掉** |
| 字幕 | 繁中／簡轉繁／缺字幕（無選中）—— **文字要改** |
| `FQRcn` footer | y=560、390×84、`$bg-secondary`、上框 `$border-subtle`、padding [12,16,24,16] → `apply` 358×48 `$accent-primary` `$radius-md`「套用篩選 · 128 部」BodyLg 16／600 `$text-on-accent` |
| Flow A 群組 | `p6EGC`；手機子群組 `JUkwx`（17040,10993，1860×889）；規格註記放 `JUkwx` 下方 |

字階：H4 18、BodyLg 16、Body 14、Label 12。間距：`xs`4・`xs-plus`6・`sm`8・`md`12・`lg`16・`lg-plus`20・`xl`24。`ctx.problems` 全檔現況以 dev 動手時 `Get` 為準（4b-2 結案時 68）。

---

## Acceptance Criteria

1. **設計稿先改（Pencil MCP；每個節點動手前再 `Get` 一次）。** 目標：A6p-M 畫的是程式碼**真的會做**的抽屜。
   - `sortrow` 第二顆「片名」→「**標題**」（🔴 #3）。四顆維持方角 44 高；已選那顆保留 `arrow-down` 圖示位。
   - 「解析度」整個分區**刪掉**（沒有資料來源，`disc-2026-09-library-has-filters-the-product-lacks`）。
   - 「字幕」分區三顆改成 **「有字幕」「缺字幕」「還沒搜尋」**（裁定 3），第二顆畫成已選（呼應 `?subtitleStatus=not_found` 深連結）。
   - 新增「**年份**」分區（碼→稿）：pill chip，文字逐字取自 `FilterPanel.tsx` 的 `DECADE_OPTIONS`（dev 從程式碼讀，不要猜）。
   - 新增「**狀態**」分區（碼→稿）：一顆 pill chip「未匹配」（文字沿用程式碼；「未比對／未匹配」二字之爭在 `disc-2026-09-unmatched-two-words`，本張不裁）。
   - 「媒體類型」（全部／電影／影集）**不加進稿**：手機上媒體類型由頁面決定（頂列標題「電影」），抽屜裡的那組 chips 在手機**隱藏**（AC #3）。
   - 「重設」加上 44×44 的觸控區（視覺文字不變、外框透明）。
   - 分區順序：排序／類型／年份／字幕／狀態。⛔ 不要寫死抽屜高度：改完 `VebkY` 設成包住內容（或用 `ctx.bounds` 讀實高）再 `y = 844 − 高度` 貼底；**高度 ≤ 675（80%）**——超過就把 `R1alM3` 改成固定高＋標示可捲動，footer 仍貼底。
   - 規格註記：`JUkwx` 下方放一則 text（樣式比照 4b-1 的 `spec-note-dsr-4b-1`：Label、`$text-muted`、`textGrowth:"fixed-width"`、寬 300；**位置先用 `FindEmptySpace`／`ctx.bounds` 確認不壓到下一個 Flow**），名稱 `spec-note-dsr-1b-b`：
     > 「A6p（手機）：排序與篩選是同一個底部抽屜（<1024 都用它；桌機 ≥1024 是篩選軌）。手機 <640 的入口是頂列右側 44×44 的篩選鈕（帶生效篩選數）；640–1024 是工具列的「篩選」鈕。排序四顆＝桌機下拉同一份四個選項、同一組字；點已選的那顆會翻轉升／降冪（箭頭跟著翻）。字幕篩選按的是後端的 subtitle_status 狀態（有字幕＝found、缺字幕＝not_found、還沒搜尋＝not_searched），不是海報徽章的語言；沒有解析度篩選（沒有資料）。底部「套用篩選 · N 部」的 N 是套用前先查一次的結果數，查不到時只顯示「套用篩選」。抽屜蓋住底部分頁列（有遮罩，點遮罩或 Esc 關閉），關掉後焦點回到觸發鈕。」
   - 收尾：全檔 `problems` **不得增加**（刪解析度分區可能減少）；存檔走選單 Save（`osascript -e 'tell application "Pen" to activate' -e 'delay 1' -e 'tell application "System Events" to tell process "Pen" to click menu item "Save" of menu "File" of menu bar 1'`），`git status --porcelain ux-design.pen` 必須出現 ` M`，grep 磁碟檔確認 `spec-note-dsr-1b-b` 與「還沒搜尋」在裡面；**存檔後**才跑 `python3 scripts/export-pen-screenshots.py`；**只 stage** `flow-a-browse-v2/a6p-m.png` 與 `_bmad-output/pen-tokens.json`，其餘 `git checkout --` 還原。
   - 📎 Pencil `execute` 失敗會 rollback 同一次呼叫的編輯；`Replace` 會重設沒寫到的屬性；`Copy` 一般節點時 `descendants` 名稱 key 會被靜默忽略（Copy 後 `Get` 新 id 再 `Update`）；讀值用 `Print(...)`。

2. **字幕篩選前端接線（`confirmed against [@contract-v1] (Story dsr-1b-a AC #1)`）。**
   - `FilterValues` 加 `subtitleStatus?: string[]`（值＝後端狀態字串；本張 UI 只出 `found`／`not_found`／`not_searched` 三個，型別不限制——契約是 CSV 任意合法值）。
   - `types/library.ts` `LibraryListParams` 加 `subtitleStatus?: string[]`；`services/libraryService.ts` `listLibrary` 有值且非空時送 `subtitle_status=<逗號連接>`（Rule 18：URL 參數手動 snake，照 `year_min` 那幾行的寫法）。⛔ `searchLibrary`（`:88-99`）不動（它本來就不送 genres／year，另一件事）。
   - ⚠️ **搜尋模式（dsr-1b-a /ship CR HIGH #1，2026-09-22）**：`useLibrary.ts:51` 有 `q` 就改走 `GET /library/search`，而該端點**忽略所有 `Filters`**（`FullTextSearch` 只有 `MATCH`＋`notRemoved`；今天 `unmatched=true` 就已經被靜默丟掉）→ `disc-2026-09-library-search-ignores-filters`。接上字幕篩選後，「搜尋框有字＋字幕 pill 亮著」會讓網址與 pill 說有篩、結果卻沒篩——正是 `-a` 要消滅的那種謊。⚖️ **開工前要裁定**：(a) 後端 `SearchLibrary` handler 也解析 `subtitle_status`（與其他篩選）並傳進 `FullTextSearch`（跨棧，可能要再拆一張小後端單）；或 (b) 本張前端在搜尋模式下**停用篩選 pill、從網址移除 `subtitleStatus`／`unmatched`**，並在無結果／結果列標示「搜尋不套用篩選」。SM 傾向 (b) 先行、(a) 立案——但這是產品行為，請 Alexyu 拍板。
   - `routes/library.tsx`：`subtitleStatus` 維持 **CSV 字串**（深連結 `?subtitleStatus=not_found` 不變、`BatchSubtitleDialog.tsx:361` 一個字不改）；**刪掉 `:17-22` 那段「not yet wired」註解**，改成一句指向本張與 `-a` 的契約。
   - `LibraryBrowseV2.tsx`：`filters` 組裝（`:166-174`）把 `search.subtitleStatus` 切成陣列放進 `subtitleStatus`；`useLibraryInfinite` 的參數（`:210-218`）帶上它；`hasActiveFilters`／`activeFilterCount`／`activeFilterLabels`（`:175-198`）都算它；套用／清除（`:260-279`）寫回 URL 時 `subtitleStatus` 用逗號連接、空陣列＝`undefined`（從網址消失）。
   - **一份標籤對照表**：新檔 `components/library/subtitleStatusFilter.ts`（或放 `FilterPanel.tsx` 內 export）：`SUBTITLE_STATUS_FILTER_OPTIONS = [{ value: 'found', label: '有字幕' }, { value: 'not_found', label: '缺字幕' }, { value: 'not_searched', label: '還沒搜尋' }]`——抽屜、`FilterChips`、無結果句子、桌機軌**全部 import 這一份**；⛔ 不要在四個地方各抄一份字串（`disc-2026-09-media-type-label-four-maps` 的教訓）。
   - `FilterPanel.tsx`：新增「字幕」分區（pill chip 多選，形狀與「類型」相同），走完 🔴 #4 列的**每一個**同步點（本地 state／`useEffect`／`selected*`／`emitInstant`／`handleApply`／`handleClear`）。**桌機篩選軌因此也多一個「字幕」分區**——這是接線的自然結果、不是本張的版面目標；軌的既有基準線（`library-filter-panel` 夾具）**會變**，屬預期（Rule 24 ①，本 AC 即是吸收點）。
   - `FilterChips.tsx`：每個選中的狀態一顆 chip，文字取自對照表，`aria-label="移除{label}篩選"`（照「未匹配」那顆 `:107-118` 的形狀）。

3. **抽屜重寫（`LibraryFilterSheetV2.tsx`）。**
   - 檔頭改 `// Design ref: ux-design.pen Screen A6p-M (Bz0YN)`（🔴 #1）。
   - `ui/Sheet`：`testId="library-sort-filter-sheet"`、`title="排序與篩選"`、`titleClassName`／`className` 把 Popup 改成 `flex flex-col overflow-hidden p-0 pb-[max(…,env(safe-area-inset-bottom))]`（🔴 #2；照 `DownloadDetailSheet` 的三段式：標題列固定、中段 `overflow-y-auto`、footer 固定）、`finalFocus` 由父層傳入（AC #4）。**標題列右側「重設」**：44×44 觸控區的 `<button type="button">`、文字 Body 600 `$accent-text`，`data-testid="library-filter-reset"`——把 draft 全部清空（排序回 `created_at desc`）、**不關閉**。
   - **排序分區**：`role="radiogroup" aria-label="排序方式"`＋四顆 `role="radio"` 方角 chip（`min-h-11`、`rounded-[var(--radius-md)]`），選項與文字 **import `SORT_OPTIONS`**（沒 export 就加 `export`，內容不變）。已選顆：`$accent-subtle` 底／600／`$accent-text`＋`ArrowDown`／`ArrowUp` 14（依 order；`aria-hidden`）；未選顆：`$bg-tertiary`／`$text-secondary`。點未選 → 該欄位＋它的 `defaultOrder`；**點已選 → 翻轉 order**（裁定 2）。鍵盤照 `DownloadSortSheet` 的 roving tabindex＋上下鍵循環（同一份 helper 可以抽到 `library/` 內自用；⛔ 不要跨資料夾 import downloads 的元件）。
   - **篩選分區**：`FilterPanel` 以 **`instant` 模式**掛在抽屜裡（改了就回報 draft；🔴 #4），`hideTypeChips`（新 prop，預設 `false`）在抽屜內設 `true`（AC #1 裁定：手機媒體類型由頁面決定；桌機軌不受影響）。抽屜自己持有 draft（`FilterValues`＋sort），`open` 由 false→true 時從 props 重設 draft。
   - **footer**：固定在底部（不隨內容捲）、上框 `$border-subtle`、padding [12,16,24,16]、一顆 358 寬 48 高的主按鈕 `data-testid="library-filter-apply"`。文字：有數字時「套用篩選 · {N.toLocaleString()} 部」，查詢中或失敗時「套用篩選」。N 來自 **既有的 `useLibraryList`**（`hooks/useLibrary.ts:22-28`，key `libraryKeys.list(params)`）以 draft 為參數、`pageSize: 1`、`enabled: open`，回應的 `totalItems`（裁定 4）。⛔ 不新增端點、不加 debounce 以外的複雜度（TanStack 的 key 去重已足夠；若 CR 認為要 debounce 用 200ms）。
   - 按「套用」→ `onApply(draft)`＋`onSortChange(...)`＋`onOpenChange(false)`。Esc／點遮罩關閉＝放棄 draft。
   - 「未匹配」chip 的 `unmatchedCount` 維持今天的傳法（🔴 #15 不修）。

4. **入口、生命週期、膠囊列（`LibraryBrowseV2.tsx`）。**
   - 標題列 `:513-528` 右側加 **手機篩選鈕**：`<button type="button" ref={filterBtnRef} aria-label="篩選" aria-haspopup="dialog" aria-expanded={sheetOpen} data-testid="library-filter-open-phone" className="relative flex size-11 shrink-0 items-center justify-center rounded-[var(--radius-md)] bg-[var(--bg-tertiary)] text-[var(--text-secondary)] sm:hidden">`＋`SlidersHorizontal size-5 aria-hidden`；`activeFilterCount > 0` 時右上角一顆計數徽章（形狀照桌機軌展開鈕 `:620-624` 那顆）。
   - 工具列既有的「篩選」鈕（`:593-605`）改 `hidden sm:flex lg:hidden`（**留在 DOM**，既有 spec 照舊找得到），補 `aria-haspopup="dialog"`／`aria-expanded`。兩顆鈕開同一個抽屜。
   - `finalFocus` 函式版：`[filterBtnRef.current, toolbarFilterBtnRef.current].find(isShown) ?? headingRef.current ?? true`；`<h1>` 加 `tabIndex={-1}`（`DownloadsBrowseV2.tsx:372-394` 的形狀）。
   - 視窗跨過 `lg`（≥1024）時抽屜要關：用 **`useSyncExternalStore`＋`(width >= 64rem)`**，寫成 `hooks/useIsDesktopLg.ts`？——⛔ **不要**。本張不新增 hook：在 render 階段用既有的 `lg:hidden` 判斷做不到，所以**改為**：`onOpenChangeComplete` 不動、接受這個缺口並立案 `disc-2026-09-library-sheet-stays-open-across-lg`（建單時已寫入 sprint-status）。
   - 膠囊列 `:629`：手機改成單行橫向捲動——照 🔴 #14 那一串 class（含 `-my-1 py-1`）＋每顆膠囊 `max-sm:shrink-0`＋選中膠囊捲進列內的 effect。`FilterChips` 元件本身的 role／aria 不動。
   - Rule 21：`LibraryBrowseV2.tsx:1-2` 檔頭補 ` · A6p-M (Bz0YN)`（照該檔既有的多畫面寫法）。

5. **既有的行為不准回歸。**
   - 桌機 ≥1024：篩選軌、`SortSelector`、工具列、網格**零像素變動**，**除了**軌多出「字幕」分區（AC #2 明寫的預期變動；`library-filter-panel` 基準線更新屬正常）。
   - `LibraryBrowseV2.spec.tsx`、`FilterPanel.spec.tsx`、`FilterChips.spec.tsx`、`LibraryFilterRail.spec.tsx`、`SortSelector.spec.tsx`、`BatchSubtitleDialog.spec.tsx` 既有斷言**一條都不改寫**。
   - `MobileMoreSheet`、`DownloadSortSheet`、`DownloadActionsSheet`、`DownloadDetailSheet` 不受影響（本張不改 `ui/Sheet`）。
   - ⛔ 不改後端、`ui/Sheet.tsx`、`ui/mobileSheet.tsx`、`SortSelector` 的桌機長相（只准加 `export`）、`LibraryStatesV2.tsx`（`-c`）、`PosterCardV2`。

6. **測試。** 分「紅」與「守」（Rule 16）。jsdom 不看 media query——class 斷言只證明 token 在，版面由 AC #7 守。
   - 新檔 `LibraryFilterSheetV2.spec.tsx`：（紅）`radiogroup`「排序方式」＋4 個 `radio`、文字逐字＝`SORT_OPTIONS`；點「標題」→ draft 變 `title asc`（箭頭朝上）；再點「標題」→ `title desc`；「字幕」分區三顆 chip 文字＝對照表；點「缺字幕」→ footer 以 `subtitleStatus: ['not_found']` 查 `useLibraryList`（mock）→ 顯示「套用篩選 · 12 部」；查詢 pending 時顯示「套用篩選」；「重設」清空 draft 且 `onOpenChange` **沒**被呼叫；「套用」→ `onApply` 收到 draft、`onSortChange` 收到 `('title','asc')`、`onOpenChange(false)`；`open` 重新變 true 時 draft 回到 props；抽屜內**沒有**媒體類型 chips（`hideTypeChips`）；footer 元素在捲動區之外（斷言 DOM 結構：footer 不是捲動容器的子孫）。
   - `FilterPanel.spec.tsx`：（紅）instant 模式點「缺字幕」→ `onApply` 收到 `subtitleStatus: ['not_found']`；批次模式套用／重置都帶／清 `subtitleStatus`；`hideTypeChips` 為 true 時沒有「全部／電影／影集」。（守）既有全部。
   - `FilterChips.spec.tsx`：（紅）`subtitleStatus: ['not_found']` → 一顆「缺字幕」、`aria-label="移除缺字幕篩選"`、點了回呼收到移除後的值。
   - `libraryService.spec.ts`：（紅）`subtitleStatus: ['not_found','not_searched']` → URL 含 `subtitle_status=not_found%2Cnot_searched`（或未編碼逗號——以既有 `genres` 的斷言寫法為準）；空陣列不送。
   - `LibraryBrowseV2.spec.tsx`：（紅）route `subtitleStatus: 'not_found'` → `useLibraryInfinite` 被以含 `subtitleStatus: ['not_found']` 的參數呼叫、膠囊列有「缺字幕」、計數徽章顯示 1；`library-filter-open-phone` 存在、`aria-haspopup="dialog"`、帶 `sm:hidden`；工具列那顆帶 `hidden sm:flex lg:hidden` 且仍在；按手機鈕 → `library-sort-filter-sheet` 出現；抽屜套用 → `navigate` 的 search 含 `subtitleStatus: 'not_found'`；清空 → `subtitleStatus: undefined`；膠囊列容器帶 `max-sm:overflow-x-auto`、膠囊帶 `max-sm:shrink-0`。（守）既有三個 describe 全部。
   - `routes/library` 的 `validateSearch`：（守）`?subtitleStatus=not_found` 仍是字串、`?subtitleStatus=` 變 `undefined`。
   - **每一項修法做 mutation check**（拿掉 → 必須紅），結果寫進 Completion Notes。⚠️ 不要寫不可能失敗的斷言：footer「固定」在 jsdom 量不到，用 DOM 結構斷言＋AC #7 的真瀏覽器量測。

7. **真瀏覽器的驗證。**
   - **視覺夾具** `library-mobile-sheets/sort-filter`：`component: LibraryFilterSheetV2`、`open: true`、`filters: { genres: ['動畫'], subtitleStatus: ['not_found'] }`、`sortBy: 'created_at'`／`desc`、`viewport: { width: 390, height: 844 }`（不可同時給 `width`）、`penNode: 'Bz0YN'`（必填）、`statesOnly: ['default']`、`seedQueries`：`libraryKeys.genres()` 給固定七個類型、**以及** footer 那次查詢的 key（`libraryKeys.list({...draft, pageSize: 1})`——key 要與元件內組出的**逐位元相同**，不然會打到真後端讓高度漂；先在元件內把組 key 的邏輯抽成可 import 的純函式）。只產 darwin；`-linux` 由 `gh workflow run "Visual Regression" --ref <branch>` 開 bootstrap PR 補。`library-filter-panel` 既有夾具會因「字幕」分區變高——**更新該基準線並在 PR 說明寫明**。
   - **e2e**（新，`tests/e2e/library-mobile.spec.ts`，`@e2e @library-mobile`）：stub `GET **/api/v1/library**`／`/library/genres`／`/library/stats`（可抽 `tests/support/helpers/library-stubs.ts`）；`import { test, expect } from '../support/fixtures'`；`beforeEach` 非 `chromium` 就 skip；每條自己 `setViewportSize`；box 一律 `Math.round`；量之前等 `data-starting-style` 消失再等動畫結束（4b-1 CR 的競態）。
     - **390・入口與抽屜**：`library-filter-open-phone` 可見、≥44×44、右緣 ≈ 根節點右緣 −16；工具列「篩選」**不可見**；按下 → `library-sort-filter-sheet` 貼底（x 0、寬 390、底邊 844）、高度 ≤ 844×0.8；footer 的「套用」按鈕底邊在抽屜內、**捲動中段後 footer 位置不變**（先 `scrollTop` 中段 200px 再量）；「重設」命中區 ≥44×44；`mobile-tab-bar` 被蓋住（`elementFromPoint`）。
     - **390・字幕篩選走完全程**：點「缺字幕」→ footer 文字符合 `/套用篩選 · \d[\d,]* 部/`（stub 回 `total_items: 12` → 「12 部」）→ 按套用 → 抽屜關、焦點回 `library-filter-open-phone`、發出的清單請求 URL `includes('subtitle_status=not_found')`、網址 `includes('subtitleStatus=not_found')`、膠囊列出現「缺字幕」、計數徽章「1」；點膠囊的 × → 請求不再含 `subtitle_status`、網址不再含 `subtitleStatus`。
     - **390・深連結**：直接開 `/library?subtitleStatus=not_found` → 第一次清單請求就含 `subtitle_status=not_found`、膠囊列有「缺字幕」（**這一條就是 8-11 AC #6 那條連結的驗收**）。
     - **390・排序**：抽屜點「標題」→ 套用 → 請求含 `sort_by=title`＋`sort_order=asc`；再開、再點「標題」→ 套用 → `sort_order=desc`。
     - **390・膠囊列**：所有膠囊垂直中心同一條線（差 <2px）、容器 `overflow-x: auto` 且 `scrollLeft` 能動（資料要讓膠囊 ≥5 顆）、整頁 `scrollWidth <= 390`、Tab 到第一顆時四邊焦點框 ≥4px 沒被裁。
     - **640（斷點另一側）**：`library-filter-open-phone` 不可見、工具列「篩選」可見且能開同一個抽屜；膠囊列 computed `flex-wrap: wrap`。
     - **1024**：兩顆入口都不可見、篩選軌可見、軌上有「字幕」分區三顆 chip。
     - 本機 `AI_PROVIDER=claude npx playwright test tests/e2e/library-mobile.spec.ts --project=chromium`；過了再 `--repeat-each=3`。
   - dev-story Step 9：`a6p-m` 對新基準線與 e2e 量測逐項核（整頁截圖只在 `DSR_SHOTS=1` 時產生）。

8. **另立的單子（建單時已寫入 sprint-status）**：`disc-2026-09-library-sheet-stays-open-across-lg`（AC #4）；↪ 補記 `disc-2026-09-library-has-filters-the-product-lacks`（字幕半收於本張、解析度半以「稿上拿掉」收；A3p-D 工具列與 A7p-D 例句的解析度仍待該單）、`disc-2026-06-library-subtitle-status-filter`（前端半收於本張）、`disc-2026-08-rails-a11y-landmarks-and-unreachable-disabled`（抽屜重寫時標題層級順手排成 `h2 → h3`，若做到就在該單補記；做不到就不動）。

9. **CI 全綠**：`pnpm run format:check`、`pnpm run lint:all`、`pnpm nx run web:typecheck --skip-nx-cache`、`python3 scripts/check-design-tokens.py`、`pnpm nx test web`。⛔ 絕不 `run_in_background` 跑測試；Nx 一律 `NX_DAEMON=false`。🚨 合併之後要看 `main` 那一次的 Tests／Docker／Visual Regression 三條。gh 操作一律帶 `GH_TOKEN=$(gh auth token --user j620656786206)`。

## Tasks / Subtasks

- [ ] **Task 1 — 設計稿：A6p-M 分區重排（刪解析度、改字幕、加年份與狀態、片名→標題、重設觸控區）＋規格註記（AC: #1）**
- [ ] **Task 2 — 字幕篩選接線：`FilterValues`／型別／service／route／`LibraryBrowseV2` 參數／對照表／`FilterPanel` 新分區／`FilterChips`（AC: #2, #6）**
  - [ ] 先跑 `FilterPanel`／`FilterChips`／`LibraryFilterRail`／`LibraryBrowseV2` 既有 spec 確認綠 → 改 → 仍綠
- [ ] **Task 3 — 抽屜重寫：三段式、排序 chips、instant `FilterPanel`＋`hideTypeChips`、重設、footer 計數（AC: #3, #6）**
- [ ] **Task 4 — 入口鈕、`finalFocus`、膠囊列單行捲動、檔頭（AC: #4, #6）**
- [ ] **Task 5 — 視覺夾具（新＋更新 `library-filter-panel`）、手機 e2e、mutation check、收尾（AC: #5, #7, #8, #9）**
  - [ ] dev-story Step 9：`a6p-m`

## Dev Notes

### 這張的重點

- **先讓深連結說真話。** Task 2 做完、抽屜還沒動之前，`/library?subtitleStatus=not_found` 就應該已經真的在篩——這是本張最有價值的一半，抽屜是另一半。
- **`ui/Sheet` 不改。** 4b-1 已把 prop 補齊；「內容捲、footer 固定」靠 `className` 三段式（看 `DownloadDetailSheet`）。
- **`FilterPanel` 走 instant 模式進抽屜，draft 由抽屜持有。** 不要為了 footer 再發明第三種模式；也不要把「套用／重置」按鈕留在 `FilterPanel` 裡讓它跟著捲。
- **稿有三處是錯的**（片名、解析度、字幕文字）；動手前看節點值與程式碼。
- **標籤只能有一份**（`SUBTITLE_STATUS_FILTER_OPTIONS`、`SORT_OPTIONS`、`DECADE_OPTIONS`）。

### 上游契約（Rule 20）

- `confirmed against [@contract-v1] (Story dsr-1b-a AC #1)`：`GET /library?subtitle_status=<csv>`，不合法值 400。⚠️ 若 `-a` 在 dev 時 bump 了（例如改成重複參數而不是 CSV），本張的 service 那一行要跟著改——開工前 grep `-a` 的 Change Log。
- 本張**不**定義新契約；`?subtitleStatus=` 的網址形狀是 8-11 AC #6 定的（pre-Rule-20，implicit v0），本張只是讓它生效。

### 建單裁定（2026-09-22，SM；Sally／Alexyu 可在 review 推翻）

1. ⚖️ **拆成三張**（見 `-a` 裁定 1）。本張＝抽屜＋接線；`-c`＝四張畫面。兩張都碰 `LibraryBrowseV2` 標題列（本張加鈕、`-c` 對文字與計數）→ **本張先**。
2. ⚖️ **排序＝四顆方角 chip、點已選翻轉升降冪。** 稿畫了 `arrow-down`，代表「新增日期」是降冪；四個欄位各有 `defaultOrder`，翻轉是把桌機下拉「再點一次同欄位翻方向」（`SortSelector.tsx:57`）搬到手機。不做第五顆「方向」控制項。
3. ⚖️ **字幕 chip 用狀態的話說**：「有字幕」＝`found`、「缺字幕」＝`not_found`、「還沒搜尋」＝`not_searched`。稿上的「繁中／簡轉繁」是徽章的**語言**（來自內嵌軌與 CN 政策），後端的 `subtitle_status` 欄位裡沒有這個資訊（🔴 #9）；寫「繁中」會篩出一堆徽章不是繁中的片。`skipped`／`untranslated`／`no_text_source` 這三個終態**先不出 chip**——契約已支援，之後要加只改對照表。⚖️ **請 Sally 確認三個字**（文末問題 1）。
4. ⚖️ **footer 的數字＝套用前用 draft 先查一次 `total_items`**（`useLibraryList`、`pageSize: 1`），查不到就只寫「套用篩選」。不做 facet 端點；不用全域的 `unmatchedCount` 那種會大於結果數的數字。
5. ⚖️ **手機抽屜隱藏媒體類型 chips**（`hideTypeChips`）。手機頂列標題就是媒體類型（「電影」），要換類型走底部分頁列／頁面，不在抽屜裡再放一組。桌機軌不受影響。
6. ⚖️ **抽屜仍是 <1024 的東西；手機圖示鈕是 <640 的另一個入口。** 不把 `lg:hidden` 換成 `useIsPhone`（🔴 #12）。
7. ⚖️ **跨過 `lg` 不自動關抽屜**——不為了它新增 hook；立案。

### 不要做的事

- 不要改 `ui/Sheet.tsx`、`ui/mobileSheet.tsx`、`ui/Dialog.tsx`、後端。
- 不要跨資料夾 import `downloads/` 的元件（`DownloadSortSheet` 是範本不是零件；要共用 roving-tabindex helper 就抽到 `hooks/` 或 `utils/`）。
- 不要在 `components/ui/` 以外 import `@base-ui/react`。
- 不要動 `BatchSubtitleDialog.tsx`、`SortSelector` 的桌機長相、`searchLibrary`。
- 不要新增計數端點；不要把 `unmatchedCount` 的全域算法套到字幕。
- 不要在元件測試裡 stub `matchMedia`。
- 不要本機產 `-linux.png`；不要改舊 spec 的斷言。
- 不要在稿上加媒體類型分區；不要保留解析度。

### 已知陷阱

- **`FilterPanel` 六個同步點漏一個＝桌機軌靜默丟篩選**（🔴 #4）——mutation check 每一個點都要有一刀。
- **`useLibraryInfinite` 的 key 不走 `libraryKeys`**（🔴 #7）——夾具 `seedQueries` 要用元件實際組出的 key；footer 查詢用 `useLibraryList`（走 `libraryKeys.list`）反而好種。
- **`p-0` 會連 safe-area 一起丟**——className 明寫 `pb-[max(…)]`。
- **`overflow-x: auto` 會裁掉上下焦點框**——`-my-1 py-1`。
- **`test-setup.ts` 的全域 `matchMedia` 回 false**＝桌機路徑；本張的手機鈕是純 CSS，不受影響。
- **Rule 26**：`subtitleStatus` 是字串 enum 守門、值不會是純數字——安全；但如果之後有人把它改成數字 id 就會踩。
- **第二次替 Base UI 抽屜拍基準線**：進場是 transition；連跑三次確認穩定。
- **`library-filter-panel` 基準線會變**（多一個分區）——PR 說明要寫，不要讓 reviewer 以為是回歸。
- **Pencil**：失敗會 rollback；`Copy` 的 `descendants` 名稱 key 對一般節點無效；存檔走選單 Save＋驗磁碟內容；匯出全量 re-render，只 stage 一張。
- **gh 帳號會被別的 session 切走**：所有 `gh` 指令帶 `GH_TOKEN=…`。
- **行號以建單時為準**（2026-09-22，main `49cc74a3`）。

### Source tree

```
ux-design.pen、_bmad-output/pen-tokens.json、screenshots/flow-a-browse-v2/a6p-m.png            ← Task 1
apps/web/src/components/library/FilterPanel.tsx（+spec）— FilterValues、字幕分區、hideTypeChips  ← Task 2/3
apps/web/src/components/library/subtitleStatusFilter.ts（新）                                      ← Task 2
apps/web/src/components/library/FilterChips.tsx（+spec）                                          ← Task 2
apps/web/src/types/library.ts、services/libraryService.ts（+spec）、routes/library.tsx            ← Task 2
apps/web/src/components/library/LibraryFilterSheetV2.tsx（重寫，+ 新 spec）                        ← Task 3
apps/web/src/components/library/SortSelector.tsx（只准加 export）                                  ← Task 3
apps/web/src/components/library/LibraryBrowseV2.tsx（+spec）                                      ← Task 2/4
apps/web/src/routes/test/-gallery.fixtures.tsx                                                    ← Task 5
tests/e2e/library-mobile.spec.ts（新）、tests/support/helpers/library-stubs.ts（若抽）             ← Task 5
tests/visual/…/library-mobile-sheets/sort-filter、…/library-filter-panel（更新）                  ← Task 5
```

### Cross-Stack Split Check

後端 task **0**（後端在 `-a`）、前端／設計／測試 task 5 → 不觸發跨棧拆分。規模：1 個元件重寫＋1 個新對照表＋4 個既有元件小改＋service／route／type 各一處、一張稿、兩個夾具、一支 e2e——與 `dsr-4b-1` 同級、略大；再拆會讓 `LibraryBrowseV2` 被三張單子各動一次，不划算。

### Time-dependent visual coverage

- **N/A — no wall-clock-reading components touched.** `components/library/` 的抽屜與篩選元件不讀時間；`RecentlyAdded` 的 `recent`／`stale` 夾具不在本張範圍。

### References

- [Source: `apps/web/src/components/library/LibraryFilterSheetV2.tsx`（全檔）；`FilterPanel.tsx:7-12, 77-88, 110-139, 166-186, 287-320`；`FilterChips.tsx:4, 107-118`；`SortSelector.tsx:10-31, 57, 70-102`；`LibraryBrowseV2.tsx:1-2, 45, 68-72, 166-198, 210-226, 260-279, 493-533, 586-658`；`LibraryFilterRail.tsx:15`]
- [Source: `apps/web/src/components/ui/Sheet.tsx:23-52, 32-36`；`downloads/DownloadSortSheet.tsx:1-12, 23, 42-53, 64, 68-98`；`downloads/DownloadDetailSheet.tsx`（三段式）；`downloads/DownloadsBrowseV2.tsx:164-175, 355-373, 390-416, 478-500`；`hooks/useIsPhone.ts`]
- [Source: `apps/web/src/routes/library.tsx:17-22, 49`；`services/libraryService.ts:65-99`；`types/library.ts:41, 106, 179, 246-256`；`hooks/useLibraryInfinite.ts:18`；`hooks/useLibrary.ts:6-28`；`utils/libraryStatus.ts:170-216`；`components/subtitle/BatchSubtitleDialog.tsx:242, 361`（+spec `:189`）]
- [Source: `apps/web/src/routes/test/-gallery.fixtures.tsx:250, 1781-1798, 3058-3080`；`tests/visual/components.visual.spec.ts`；`tests/e2e/downloads-mobile.spec.ts`（範本）；`apps/web/src/test-setup.ts`]
- [Source: `apps/api/internal/models/movie.go:68-157`（10 個狀態）；`dsr-1b-a-library-subtitle-status-filter-backend.md` AC #1 `[@contract-v1]`]
- [Source: `ux-design.pen` `Bz0YN`／`lJGih`／`VebkY`／`R1alM3`／`FQRcn`／`JUkwx`／`p6EGC`／`h1v1U6` —— SM 建單時以 Pencil MCP 逐節點讀出（2026-09-22）；程式碼現況由唯讀稽核代理查證]
- [Source: `DESIGN.md:604-636`（手機規則、44×44、Sheet 80%）；`sprint-status.yaml` → `dsr-1b-flow-a-mobile`（`:1302`）、`disc-2026-06-library-subtitle-status-filter`（`:250`）、`disc-2026-09-library-has-filters-the-product-lacks`（`:1307`）、`disc-2026-09-unmatched-two-words`（`:1352`）、`disc-2026-09-bottomsheet-grabber-three-variants`（`:1375`）、`disc-2026-09-ui-sheet-no-close-button`（`:1378`）]
- [Source: `dsr-1-flow-a-browse-v2.md`（拆單紀錄、CR 第 2 項的檔頭形式）；`dsr-4b-1`／`dsr-4b-2`（抽屜、入口鈕、膠囊列、e2e 量法）；`8-11-batch-subtitle-ui.md:49, 183, 216`]
- [Source: project-context.md#Rule 16／#Rule 18／#Rule 20／#Rule 21／#Rule 24／#Rule 26；`.claude/memory/feedback_split_oversized_stories.md`、`feedback_measure_the_breakpoint_you_return_to.md`、`feedback_verify_pen_saved_before_commit.md`、`project_pen_schema_gotchas.md`、`feedback_gh_token_explicit_for_pr_ops.md`]

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

- Ultimate context engine analysis completed - comprehensive developer guide created（SM Bob，2026-09-22）

### Discovery Triage

- **建單時的發現（SM Bob 2026-09-22）：**
  - ① `disc-2026-06-library-subtitle-status-filter` 前端半 → AC #2 吸收。
  - ① `disc-2026-09-library-has-filters-the-product-lacks` 的 A6p-M 兩組 → AC #1（解析度刪、字幕做）；A3p-D／A7p-D 的解析度殘留仍在該單。
  - ① 桌機篩選軌因接線多出「字幕」分區 → AC #2 明寫、AC #5／#7 覆蓋基準線更新。
  - ③ 抽屜開著跨過 `lg` 不會關 → `disc-2026-09-library-sheet-stays-open-across-lg`。
  - ③ 「繁中／簡轉繁」在稿上代表語言、系統沒有可篩的語言欄位 → 併入 `disc-2026-09-library-has-filters-the-product-lacks` 的 ↪ 補記（若之後要做「依字幕語言篩」是新功能，另立）。
- **dev-story 期間的發現：** （dev 填寫）

### File List

## Change Log

- 2026-09-22 — 建單（SM Bob，create-story；main `49cc74a3`）。由 `dsr-1b-flow-a-mobile` 拆出的第二塊；依賴 `dsr-1b-a`。設計稿 A6p-M 以 Pencil MCP 逐節點讀出（含 `ctx.problems` 零 clipping）、`components/library/` 與接線路徑由唯讀稽核代理查證（23 條陷阱清單全數併入 🔴 與 AC）。
