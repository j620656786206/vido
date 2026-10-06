# Bugfix: 電腦版收合／展開篩選欄時，畫面會滑動過去，不再「突然少一塊、突然多一塊」

Status: review

**Source:** Alexyu 2026-10-06 回報（sprint-status `disc-2026-10-filter-rail-toggle-no-motion`，P3・動態）。Sally 2026-10-06 提出做法（View Transitions），⚖️ **Alexyu 2026-10-06：「同意方案、探索頁一起做」**（兩頁抽同一個共用 hook）。設計稿：`ux-design.pen` 節點 `Qaz1x`「I11-D · 篩選 Rail 收合／展開動態 spec（桌面）」，截圖 `_bmad-output/screenshots/flow-i-advanced-search/i11-d.png`（`scripts/export-pen-screenshots.py:227` 已登記）。

## Story

身為在電腦上用媒體庫和探索頁的 Vido 使用者，
我希望按下「收合篩選」或工具列的「篩選」時，篩選欄是滑出去、滑回來，
這樣我看得出剛剛發生了什麼事、篩選欄去了哪裡、要從哪裡叫回來，而不是畫面突然跳一下。

## 查到的事（2026-10-06，Bob 逐一開檔確認）

### 兩頁的收合狀態

- **媒體庫：** 存在 `localStorage` 的 `vido:library:rail-collapsed`（`apps/web/src/components/library/LibraryBrowseV2.tsx:66`）。開頁時由 `getStoredRailCollapsed()`（`:91-97`）在 `useState` 初始值讀出（`:174`），所以重新整理時一開始就是終點狀態、沒有「從展開變收合」的過程。`setRailCollapsed`（`:175-182`）同時寫 state 與 localStorage。
- **探索頁：** 同一套寫法，key 是 `vido:discover:rail-collapsed`（`apps/web/src/components/search/DiscoverBrowseV2.tsx:37`），讀取 `:39-45`、state `:57`、setter `:58-65`。
- **篩選欄是直接掛載／卸載：** 媒體庫 `LibraryBrowseV2.tsx:611` `{!railCollapsed && (…)}`，外層 `<div className="hidden lg:block">`（`:612`），`onCollapse={() => setRailCollapsed(true)}`（`:621`）。探索頁 `DiscoverBrowseV2.tsx:170-186`（外層同樣 `hidden lg:block` `:171`、`onCollapse` `:183`）。
- **海報格子：** 媒體庫的欄數 class 跟著 `railCollapsed` 瞬間切換（`LibraryBrowseV2.tsx:537-539`），表在 `apps/web/src/components/library/libraryGridCols.ts:14-18`（展開 `lg:grid-cols-3 xl:grid-cols-4 2xl:grid-cols-5`、收合 `lg:grid-cols-4 xl:grid-cols-6`），用在 `LibraryBrowseV2.tsx:783-786`；載入骨架也吃同一個值（`:759`、`:776`）。探索頁的 `MediaGrid` 是 `auto-fill`，欄數由寬度決定（`apps/web/src/components/media/MediaGrid.tsx:44`、`:112`）。

### 共用的篩選欄外殼

- `apps/web/src/components/ui/FilterRailShell.tsx:47-51`：`<aside>`，`sticky top-14`、`w-[264px]`。
- 標題列 `:53`；「篩選」標題＋數字徽章是一組 `<div className="flex items-center gap-2">`（`:54-71`，`<h2>` 在 `:57-59`、徽章在 `:60-70`，只有 `activeCount > 0` 才出現）。
- 收合鈕 `:72-80`：`aria-label="收合篩選"`（`:76`），**沒有 `aria-expanded`，也沒有 ref**。
- 兩頁各包一層：`LibraryFilterRail.tsx:41-46`、`DiscoverFilterRail.tsx:56-61`，只轉傳 `testId`／`collapseTestId`／`onCollapse`，沒有轉傳 ref。

### 工具列的「篩選」鈕（展開鈕）

- **媒體庫電腦版展開鈕：** `LibraryBrowseV2.tsx:690-709`，`data-testid="library-rail-expand"`（`:694`）、`hidden … lg:flex`（`:695`），裡面有數字（`:703-707`）。**沒有 ref、沒有 `aria-expanded`。** 位置在排序選單（`:666-670`）之後，跟 I11-D 畫的「新增日期 → 篩選 3」一致。
- ⚠️ **`toolbarFilterBtnRef` 不是電腦版那顆。** `:165`／`:675` 那顆是平板（`sm:flex lg:hidden`，`:680`）打開底部面板的鈕，它的 `aria-expanded={filterSheetOpen}`（`:678`）講的是面板，不是篩選欄。不要動它，也不要拿它當焦點目標。手機的 `phoneFilterBtnRef`（`:164`、`:584-589`）同理。
- **媒體庫選取模式：** 選取時整條工具列換成 `SelectionToolbar`（`:628-663`），展開鈕不存在；但篩選欄本身不受選取模式影響（`:611` 只看 `railCollapsed`）。
- **探索頁電腦版展開鈕：** `DiscoverBrowseV2.tsx:211-226`，`data-testid="discover-rail-expand"`（`:215`）、`hidden lg:flex`（`:216`），同樣沒有 ref、沒有 `aria-expanded`。位置在 `MediaTypeTabs`（`:191-198`）和手機篩選鈕（`:200-209`，`lg:hidden`）之後。探索頁沒有選取模式。

### 焦點

- 兩頁都**沒有**處理收合／展開後的焦點：`grep -n "focus"` 掃 `LibraryBrowseV2.tsx`、`DiscoverBrowseV2.tsx`、`FilterRailShell.tsx`，只有 `LibraryBrowseV2.tsx:559` 與 `:715-716` 的註解，沒有任何 `.focus()`。收合鈕在篩選欄裡，篩選欄一卸載，焦點就掉到 `<body>`。
- 媒體庫已有「程式聚焦用」的頁面標題：`<h1 ref={headingRef} tabIndex={-1}>`（`:561-567`），底部面板關閉時拿它當最後的退路（`filterSheetFinalFocus`，`:168-172`）。探索頁的 `<h1>`（`DiscoverBrowseV2.tsx:166`）沒有 ref、沒有 `tabIndex`。

### 動態 token 與減少動態

- Token：`apps/web/src/styles.css:143-149` — `--motion-state: 200ms`、`--motion-move: 320ms`、`--motion-leave: 240ms`、`--ease-settle: cubic-bezier(0.16, 1, 0.3, 1)`、`--ease-leave: cubic-bezier(0.4, 0, 0.9, 0.4)`。規則「離場比進場快」寫在 `:140-141`。
- 減少動態第 1 段（token 歸零）：`styles.css:300-315`，放在 `@layer base` 外面（`:290-293` 說明原因），把所有時間 token 壓到 `1ms`。
- 減少動態第 2 段（保險網）：`styles.css:582-591`，選擇器是 `*, *::before, *::after`（`:583-585`）。`::view-transition-*` 偽元素不是 `*`／`::before`／`::after`，**這張網抓不到** — 所以減少動態時要在 JS 端直接不呼叫 `startViewTransition`（Sally 方案）。
- JS 端已有判斷函式：`apps/web/src/lib/motion.ts:21-24` `prefersReducedMotion()`，每次即時讀 `matchMedia`、沒有 `matchMedia` 時回 false。**重用它，不要另寫。** 測試寫法前例在 `lib/motion.spec.ts`（`vi.stubGlobal('matchMedia', …)`）。

### 工具鏈

- **`flushSync` 目前沒人用：** `grep -rn "flushSync\|startViewTransition\|view-transition\|viewTransition" apps/web/src tests` 零結果。這張單是第一個用 View Transitions 的地方。
- **型別：** TypeScript 5.9.3（`node_modules/typescript/package.json`；`package.json` 宣告 `~5.9.2`）。`tsconfig.base.json:13` `lib: ["es2020", "dom"]`，`apps/web/tsconfig*.json` 沒有覆寫 `lib`。TS 內建的 `lib.dom.d.ts:10378` 已宣告 `startViewTransition(callbackOptions?: ViewTransitionUpdateCallback | StartViewTransitionOptions): ViewTransition` —— **而且是「一定存在」的型別**，所以功能偵測要寫 `typeof document.startViewTransition === 'function'`，不要寫 `if (document.startViewTransition)`（TS 會判定恆為真）。
- **單元測試環境：** vitest `environment: 'jsdom'`（`apps/web/vite.config.mts:58`）、setup `./src/test-setup.ts`（`:60`）。jsdom 版本 22.1.0，`grep -rl startViewTransition node_modules/jsdom/lib` 零結果 → jsdom **沒有**這個 API，測試要自己裝替身。`test-setup.ts:80-91` 只在缺 `matchMedia` 時裝一個 `matches: false` 的替身。
- **視覺回歸：** Playwright `visual` 專案 1280×800、`contextOptions: { reducedMotion: 'reduce' }`（`playwright.config.ts:148-182`，`:180`）。也就是說，視覺測試永遠走「不動畫」那條路，拍不到動畫本身。

### 會被影響的測試

- **E2E（只有一支碰到篩選欄收合）：** `tests/e2e/discover-filters.spec.ts:263-277` — 按 `discover-rail-collapse` 後斷言 `discover-filter-rail` 數量為 0、`discover-rail-expand` 可見。跑在 `chromium`／`firefox` 桌面專案（`playwright.config.ts:106-116`），**沒有**減少動態 → 會真的走 View Transition。斷言都是會自動重試的 `expect`，預期仍綠，但要實跑確認。`grep -rn "rail-collapse\|rail-expand\|rail-collapsed" tests` 只有這兩行。
- **單元測試：** `LibraryBrowseV2.spec.tsx:188-209`（預設有篩選欄、收合後消失＋欄數變 4、再展開）、`:515-527`（localStorage 預設收合時骨架欄數）；`LibraryFilterRail.spec.tsx:55`、`DiscoverFilterRail.spec.tsx:78`（點收合呼叫 `onCollapse`）；`FilterRailShell.spec.tsx`；`routes/library.spec.tsx:176-209`；`LibraryStatesV2.spec.tsx:73-79`。jsdom 沒有 API → 走「直接切換」那條路，這些測試應不受影響（`userEvent.click` 後同步斷言，所以退路必須是同步的）。
- **視覺基準：** `-gallery.fixtures.tsx:3037-3061` 的 `search-discover-filter-rail-unavailable` 單獨畫 `DiscoverFilterRail`（寬 300），基準在 `tests/visual/components.visual.spec.ts-snapshots/components/search-discover-filter-rail-unavailable/`。`grep -n "LibraryBrowseV2\|DiscoverBrowseV2" -gallery.fixtures.tsx` 零結果 → 沒有整頁的 fixture。收合鈕加 `aria-expanded`、外殼加 `data-*` 屬性不改像素，預期不會有基準要更新。
- **動態守門測試：** `apps/web/src/styles-motion.spec.ts:210-290` 只掃 `.ts/.tsx` 裡的 `animate-*` utility 是否有註冊；在 `styles.css` 寫給 `::view-transition-*` 用的 `@keyframes` 不會被它擋。ESLint `local/no-hardcoded-duration`（`apps/web/src/eslint-rules/no-hardcoded-duration.js:17-25`）只管 TS 字串裡的 Tailwind `duration-*`，CSS 裡的時間由 `styles.css` 自己管 —— 但仍然**只准用 token**。

## 設計（I11-D，`Qaz1x`）

**收合（篩選區退場）** — 按篩選欄右上的收合鈕：
- 篩選欄 `translateX 0 → -100%`＋淡出，`--motion-leave` 240ms、`--ease-leave`。一起步就決定要走，180ms 時才走 43%，最後 60ms 一口氣離開，不回頭。
- 「篩選 3」標題脫離篩選欄，往右變形成工具列的「篩選 3」按鈕。
- 海報格不逐張飛：舊的 3 欄淡出、新的 4 欄淡入，`--motion-state` 200ms 內完成。
- 終點：內容欄拿回 264px；焦點移到工具列「篩選」鈕（`aria-expanded="false"`）。

**展開（篩選區落定）** — 按工具列「篩選」：
- 篩選欄 `translateX -100% → 0`＋淡入，`--motion-move` 320ms、`--ease-settle`（起步快、收尾慢，80ms 就走了 83%，不回彈）。
- 「篩選」按鈕縮回篩選欄頂端，長回標題。
- 海報格交叉淡化：4 欄淡出、3 欄淡入，200ms。
- 終點：焦點移到篩選欄的收合鈕（`aria-expanded="true"`），再按一次 Enter 就能收回。

**共通規則：**
- 減少動態（`prefers-reduced-motion: reduce`）：不做任何過場，篩選欄、按鈕、海報格直接切到終點畫面；焦點規則照舊。
- 載入／重新整理：直接畫出上次記住的狀態，從不播放收合或展開。動畫只回應使用者這一次的點擊。
- 內容欄寬度一次到位，不做寬度動畫；版面重排的視覺過渡只交給海報格的 200ms 交叉淡化。
- 焦點與無障礙：收合後 → 工具列「篩選」鈕；展開後 → 篩選欄收合鈕。兩顆鈕都帶 `aria-expanded`。
- 適用範圍：媒體庫（`LibraryBrowseV2`）與探索（`DiscoverBrowseV2`）共用 `FilterRailShell`，兩頁同一套動態。只限桌面（`lg` 以上）；平板、手機的底部面板不在範圍內。

**實作做法（Sally 方案，Alexyu 已同意）：**
- 一個共用 hook（建議 `apps/web/src/hooks/useFilterRailTransition.ts`），兩頁都用它取代直接呼叫 `setRailCollapsed`。
- 點擊時：若 `typeof document.startViewTransition === 'function'` 且 `!prefersReducedMotion()` → 在 `<html>` 設 `data-rail-motion="collapse"|"expand"`，呼叫 `document.startViewTransition(() => { flushSync(() => setRailCollapsed(next)); 聚焦目標 })`，在 `transition.finished` 後（不論成功或被跳過）移除 `data-rail-motion`。否則 → `flushSync(() => setRailCollapsed(next))` 後直接聚焦。兩條路都同步更新 DOM 再聚焦。
- `view-transition-name` **只在 `:root[data-rail-motion]` 底下才生效**（CSS 用屬性選擇器套上，平常不掛名字）：篩選欄 `<aside>` 一個名字（例：`filter-rail`）；標題組（`FilterRailShell.tsx:54-71` 那個 div）與兩頁的展開鈕共用另一個名字（例：`filter-rail-trigger`），讓瀏覽器把兩者當成同一個東西變形。兩者永遠不會同時存在，名字不會撞。
- 動畫寫在 `styles.css`，全部用 token：`::view-transition-old(filter-rail)` 跑退場（`--motion-leave`／`--ease-leave`，translateX 到 -100%＋淡出）；`::view-transition-new(filter-rail)` 跑進場（`--motion-move`／`--ease-settle`，從 -100% 進來＋淡入）；`::view-transition-group(filter-rail-trigger)` 的時長與曲線依 `data-rail-motion` 是 collapse 還是 expand 取對應那一組；`::view-transition-old(root)`／`::view-transition-new(root)` 的交叉淡化設成 `--motion-state`（海報格就是靠這個淡入淡出）。這些規則全部掛在 `:root[data-rail-motion]` 底下，不影響全站。
- 不用 `transition-[width]`、不用 FLIP、不給海報格或內容欄取名字（取了名字，瀏覽器會把整欄當圖片拉伸）。

## Acceptance Criteria

1. **收合有動態：** 桌面（`lg`+）、沒有減少動態、瀏覽器支援 View Transitions 時，按收合鈕 → 篩選欄往左滑出並淡出（translateX 0 → −100%），時長 `--motion-leave`、曲線 `--ease-leave`；「篩選」標題組變形到工具列「篩選」鈕的位置。
2. **展開有動態：** 同條件下按工具列「篩選」→ 篩選欄從左滑入並淡入（−100% → 0），時長 `--motion-move`、曲線 `--ease-settle`；工具列「篩選」鈕變形回篩選欄頂端的標題組。
3. **海報格只淡入淡出：** 收合與展開時，海報格只有 `--motion-state` 的整片交叉淡化，沒有單張卡片位移；內容欄寬度不做動畫（沒有 `transition-[width]`、沒有 FLIP、海報格和內容欄不掛 `view-transition-name`）。
4. **減少動態：** `prefers-reduced-motion: reduce` 時不呼叫 `document.startViewTransition`，直接切到終點畫面；焦點規則（AC #6）照舊。
5. **不支援的瀏覽器／載入時：** `document.startViewTransition` 不存在時直接切換（行為與現在相同，再加上 AC #6 的焦點）；頁面載入、重新整理、切換路由時，依 localStorage 直接畫出記住的狀態，**絕不**呼叫 `startViewTransition`（只有點擊會觸發）。
6. **焦點：** 收合後焦點落在工具列「篩選」鈕（`library-rail-expand`／`discover-rail-expand`）；展開後焦點落在篩選欄收合鈕（`library-rail-collapse`／`discover-rail-collapse`）。媒體庫在選取模式中收合（此時工具列沒有展開鈕），焦點退到頁面標題 `library-page-title`（已有 `tabIndex={-1}`）。
7. **`aria-expanded`：** 收合鈕帶 `aria-expanded="true"`；兩頁的電腦版展開鈕帶 `aria-expanded="false"`。不加 `aria-controls`（篩選欄收合時已卸載，指向不存在的 id 反而不合規）。平板／手機那幾顆開底部面板的鈕的 `aria-expanded` 不變。
8. **兩頁同一套：** 媒體庫與探索頁都用同一個共用 hook；`FilterRailShell` 只改一次。
9. **測試：** hook 的單元測試（AC #1–#6 的分支）、兩頁的焦點落點測試、`aria-expanded` 斷言（詳見 Tasks T5）。既有測試全綠，`tests/e2e/discover-filters.spec.ts:263-277` 在 chromium 實跑綠。
10. **檢查全綠：** `pnpm nx test web`、`pnpm run lint:all`、`pnpm run format:check`；視覺回歸本機跑 `search-discover-filter-rail-unavailable` 無差異。

## Tasks / Subtasks

- [x] **T1 共用 hook ＋ 動畫 CSS**（AC #1–#5, #8）
  - [x] 新增 `apps/web/src/hooks/useFilterRailTransition.ts`：輸入 `setRailCollapsed`、兩個焦點目標的取得函式（收合後要聚焦誰、展開後要聚焦誰）；回傳 `collapse()`／`expand()`。內部用 `prefersReducedMotion()`（`lib/motion.ts:21`）、`typeof document.startViewTransition === 'function'`、`flushSync`（`react-dom`）。
  - [x] 動畫期間在 `document.documentElement` 設 `data-rail-motion`，`transition.finished.finally(...)` 移除；同一時間又點一次（前一個被瀏覽器跳過）也要能正確清掉。
  - [x] `styles.css`：新增 `@keyframes`（例：`rail-leave`、`rail-enter`）與 `:root[data-rail-motion]` 底下的 `view-transition-name` 與 `::view-transition-*` 規則，時長／曲線只用 token。放在 `@layer base` 之外或之內都可以，但要有一段註解說明「為什麼 JS 端擋減少動態」（`:582` 的保險網抓不到偽元素）。
- [x] **T2 `FilterRailShell` 加無障礙與掛名點**（AC #1, #2, #6, #7）
  - [x] 收合鈕加 `aria-expanded="true"`，並接受一個 ref（`collapseButtonRef` prop 或 `forwardRef` 皆可；React 19 可直接把 `ref` 當 prop）。
  - [x] `<aside>` 與標題組 div 加上給 CSS 選的屬性（例：`data-rail-vt="rail"`／`data-rail-vt="trigger"`），名字本身只在 `:root[data-rail-motion]` 下由 CSS 套上。
  - [x] `LibraryFilterRail`、`DiscoverFilterRail` 轉傳 ref。
- [x] **T3 媒體庫接上**（AC #1–#7）
  - [x] `LibraryBrowseV2`：展開鈕（`:690-709`）加 ref、`aria-expanded={false}`、與標題組相同的 `data-rail-vt="trigger"`；`onCollapse`（`:621`）與展開鈕 `onClick`（`:693`）改走 hook。
  - [x] 收合後焦點：展開鈕存在就聚焦它；不存在（選取模式）就聚焦 `headingRef`（`:561-563`）。
- [x] **T4 探索頁接上**（AC #1–#8）
  - [x] `DiscoverBrowseV2`：展開鈕（`:211-226`）同 T3 處理；`onCollapse`（`:183`）與展開鈕 `onClick`（`:214`）改走 hook。探索頁沒有選取模式，展開鈕在收合後一定存在。
- [x] **T5 測試**（AC #9）
  - [x] `hooks/useFilterRailTransition.spec.ts(x)`：
    - 支援 API、沒有減少動態 → 點擊時呼叫 `startViewTransition` 一次，callback 內狀態已更新、焦點已移動；`data-rail-motion` 在動畫中為 `collapse`／`expand`，`finished` 後移除。
    - 減少動態（stub `matchMedia` 回 `matches: true`）→ **不**呼叫 `startViewTransition`，狀態與焦點照樣更新。
    - API 不存在 → 不拋錯，狀態與焦點同步更新。
    - 掛載（初次 render）時**從不**呼叫 `startViewTransition`，即使 localStorage 是收合。
    - 替身寫法：`Object.defineProperty(document, 'startViewTransition', { value: vi.fn((cb) => { cb(); return { finished: Promise.resolve(), ready: Promise.resolve(), updateCallbackDone: Promise.resolve(), skipTransition: vi.fn() }; }), configurable: true })`，`afterEach` 刪掉；`matchMedia` 照 `lib/motion.spec.ts` 用 `vi.stubGlobal` ＋ `vi.unstubAllGlobals()`。
  - [x] `LibraryBrowseV2.spec.tsx`：收合後 `document.activeElement` 是 `library-rail-expand`；展開後是 `library-rail-collapse`；選取模式中收合 → `library-page-title`；兩顆鈕的 `aria-expanded`。
  - [x] `DiscoverBrowseV2.spec.tsx`：收合／展開的焦點落點（`discover-rail-expand`／`discover-rail-collapse`）與 `aria-expanded`（此檔目前沒有收合測試，要新增）。
  - [x] `FilterRailShell.spec.tsx`：收合鈕 `aria-expanded="true"`、ref 指到收合鈕。
- [x] **T6 檢查與實機確認**（AC #9, #10）
  - [x] `pnpm nx test web`、`pnpm run lint:all`、`pnpm run format:check`。
  - [x] Playwright：`tests/e2e/discover-filters.spec.ts` 在 chromium 跑綠。
  - [x] 本機 visual 跑一次，確認 `search-discover-filter-rail-unavailable` 無差異（不要 `test:visual:update`）。
  - [x] 瀏覽器實機：1280 寬、系統未開減少動態，用 DevTools Animations 面板放慢到 10%，對照 `i11-d.png` 兩列分鏡（收合 240ms、展開 320ms、海報格只淡化、標題↔按鈕變形）；開 DevTools「Emulate prefers-reduced-motion: reduce」確認直接切換；重新整理確認不播動畫；鍵盤 Tab＋Enter 確認焦點落點。

前端 6 項、後端 0 項 → 不拆單（Cross-Stack Split Check：後端 0 ≤ 3）。

## Dev Notes

### 不要做的事

- 不要用 `transition-[width]` 或任何寬度動畫，也不要做 FLIP 逐張飛卡片（Sally 方案已排除：每格重排上百張卡，最後仍會跳欄）。
- 不要給海報格、內容欄掛 `view-transition-name`：被命名的元素會被當成一張圖片在新舊尺寸間縮放，海報會被拉伸。海報格的過渡就是 root 的交叉淡化。
- 不要在 `useEffect`／掛載時呼叫 `startViewTransition`，也不要把它包在 `setRailCollapsed` 裡（`setRailCollapsed` 只是 setter；動畫只屬於點擊）。localStorage 初始值的讀法（`LibraryBrowseV2.tsx:174`、`DiscoverBrowseV2.tsx:57`）不要動。
- 不要另寫 `matchMedia('(prefers-reduced-motion: reduce)')`，用 `lib/motion.ts:21` 的 `prefersReducedMotion()`。
- 不要在 CSS 寫死毫秒或 cubic-bezier，一律用 `--motion-*`／`--ease-*` token（減少動態第 1 段會把 token 壓成 1ms，這是第二道保險）。
- 不要動 `toolbarFilterBtnRef`／`phoneFilterBtnRef` 那兩顆鈕（它們開的是平板／手機的底部面板，`aria-expanded` 講的是面板）。
- 不要改 `data-testid`（`library-rail-collapse`／`library-rail-expand`／`discover-rail-collapse`／`discover-rail-expand`／`*-filter-rail`）——E2E 與單元測試都靠它們。
- 不要把 `view-transition-name` 永久掛在元素上：它會讓元素自成一個 stacking context，可能影響 sticky 篩選欄與其他疊層；只在 `:root[data-rail-motion]` 期間生效。
- 不要改平板／手機的底部面板（不在 I11-D 範圍）。

### 已知陷阱

- **TS 型別認定 API 一定存在**（`lib.dom.d.ts:10378`）：偵測用 `typeof document.startViewTransition === 'function'`。
- **jsdom 沒有這個 API**：既有單元測試會走直接切換路徑，所以退路必須同步（`flushSync`）——`LibraryBrowseV2.spec.tsx:197-200` 點擊後立刻斷言 DOM。
- **`flushSync` 在 View Transition callback 裡是必要的**：瀏覽器在 callback 回傳（或回傳的 Promise 結束）後才拍「新畫面」，React 18+ 的批次更新若沒 flush，會拍到舊畫面，動畫等於沒做。這也是全 repo 第一次用 `flushSync`（grep 零結果），在 hook 裡寫一行註解說明原因。
- **聚焦要在 `flushSync` 之後**、在同一個 callback 裡：此時展開鈕／收合鈕才剛掛上 DOM。
- **重複點擊**：動畫中再呼叫一次 `startViewTransition`，瀏覽器會跳過前一個（`finished` 仍會結束）。`data-rail-motion` 的移除要只在「最後一次」的 `finished` 後做，或用計數，避免下一個動畫半路被拔掉屬性。⚠️ 未查證（dev 開工時先確認）：Chrome 是否在動畫中把點擊導到 `<html>`，讓第二次點擊根本點不到按鈕——不影響 AC，只影響這個陷阱是否會發生。
- **媒體庫選取模式**：工具列換成 `SelectionToolbar`（`LibraryBrowseV2.tsx:628-663`），收合後沒有展開鈕可聚焦，要退到 `headingRef`。此時變形（標題組 → 按鈕）沒有終點，瀏覽器會讓標題組單獨淡出，可接受。
- **E2E 在桌面 Chrome／Firefox 沒有減少動態**：`discover-filters.spec.ts:263-277` 會真的跑動畫；它的斷言會自動重試，預期仍綠，但 T6 要實跑。⚠️ 未查證（dev 開工時先確認）：Playwright 內建 Firefox 版本是否支援 `startViewTransition`——不支援就走退路，兩種結果都應該綠。
- **探索頁的 `<h1>` 沒有 ref**（`DiscoverBrowseV2.tsx:166`）：探索頁沒有選取模式，展開鈕收合後一定在，所以不需要標題退路；不要順手改 `<h1>`。

### ⚖️ 已確認（Alexyu 2026-10-06：「都照推薦」）

- 探索頁「篩選」按鈕**本單不搬**，變形終點跟著它現在的位置（`MediaTypeTabs` 之後，`DiscoverBrowseV2.tsx:191-226`）。搬到工具列最左另開單 `disc-2026-10-discover-filter-button-leftmost`，Alexyu 指定**本單做完接著做**。
- AC #6「選取模式中收合 → 焦點退到頁面標題」照 Bob 的提議（`filterSheetFinalFocus` 前例，`LibraryBrowseV2.tsx:168-172`）。

### Time-dependent visual coverage

N/A — no wall-clock-reading components touched（hook 與兩頁的改動都不讀 `Date`）。

視覺回歸：`visual` 專案固定 `reducedMotion: 'reduce'`（`playwright.config.ts:180`），只會拍到「直接切換」後的終點畫面，拍不到動畫；動畫本身靠 T5 單元測試＋T6 實機對照 `i11-d.png`。唯一碰到篩選欄的基準是 `search-discover-filter-rail-unavailable`（單獨畫篩選欄），本單只加屬性，預期不需更新。

### References

- 設計：`ux-design.pen` `Qaz1x`（I11-D）、`_bmad-output/screenshots/flow-i-advanced-search/i11-d.png`
- 動態規範：`apps/web/src/styles.css:123-149`（token 與「動的東西＝正在發生的事」）、`:288-315`、`:569-591`
- JS 動態合約：`apps/web/src/lib/motion.ts`
- 前例（篩選欄外殼抽出）：`FilterRailShell.tsx:1-6` 註解（ux3-3-2 AC #11）

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5 (Amelia, dev-story, non-interactive)

### Debug Log References

- Red first: `useFilterRailTransition.spec.tsx` failed on the missing module; `LibraryBrowseV2.spec.tsx` 2 new focus tests red before wiring (plus a 3rd red from my own selection-mode test leaking `vido:library:rail-collapsed='1'` into the dsr-1b-c suite — fixed with `onTestFinished` cleanup); `DiscoverBrowseV2.spec.tsx` new focus test red before wiring.
- Mutation check (each restored afterwards; restored file `cmp`-identical): (a) drop `|| prefersReducedMotion()` → 1 failed (AC #4 test); (b) drop the `owner === motionOwner` guard → 1 failed (second-click test); (c) `flushSync(() => set…)` → plain `set…` → 4 failed (AC #1/#2/#4/#5).
- Browser verification (Playwright headless Chromium 1217 against the dev server, 1440×900, `colorScheme: 'dark'`; slowed frames via CDP `Animation.setPlaybackRate(0.1)`; `startViewTransition` call counter injected by init script). Frames in `/private/tmp/claude-502/-Users-alexyu-projects-personal-vido/07840650-15ce-4e6a-9507-6230e9bd5abf/scratchpad/motion/`:
  - `library-collapse-0ms.png` → `library-collapse-100ms-slowed.png` → `library-collapse-180ms-slowed.png` → `library-collapse-end.png`: rail slides left and fades, clipped at its own left edge (does not cross the sidebar nav); 「篩選」 title travels right/up toward the toolbar; grid is a whole-page cross-fade (3-col ghost under 6-col) with no per-card travel; end = toolbar 「新增日期 · 篩選」, grid reflowed.
  - `library-expand-80ms-slowed.png` → `library-expand-200ms-slowed.png` → `library-expand-end.png`: rail enters from the left, already ~82% of the way at 80ms (collapse button 437px vs 485px final — matches 「80ms 就走了 83%」).
  - Real speed: `library-collapse-realtime-100ms.png` / `-500ms.png`, `library-expand-realtime-100ms.png` / `-500ms.png`; same set for `discover-*` (TMDb is 401 locally so the discover grid area shows the fail-soft banner, the rail/trigger motion is still visible).
  - Read back from `document.getAnimations({subtree:true})`: `::view-transition-old(filter-rail)` = `rail-leave` 240ms `cubic-bezier(0.4, 0, 0.9, 0.4)`; `::view-transition-new(filter-rail)` = `rail-enter` 320ms `cubic-bezier(0.16, 1, 0.3, 1)`; `filter-rail-trigger` group/old/new 240ms ease-leave on collapse, 320ms ease-settle on expand; `root` group/old/new 200ms. `data-rail-motion` = `collapse`/`expand` during, `null` after.
  - Focus (both pages, mouse and keyboard Enter): collapse → `*-rail-expand`, expand → `*-rail-collapse`.
  - `reducedMotion: 'reduce'`: `startViewTransition` calls = 0 for collapse + expand, rail gone at 30ms (`library-reduced-collapse-30ms.png`, `discover-reduced-collapse-30ms.png`), focus rules unchanged, no attribute set.
  - Reload with `rail-collapsed='1'`: calls = 0, no attribute, rail not mounted (`library-reload-collapsed.png`, `discover-reload-collapsed.png`).
  - Console: no page errors / no unhandled `AbortError` from skipped transitions. Only pre-existing env noise (`/health` 404 on :4200, TMDb 401 with the dummy key).

### Completion Notes List

- **T1** `apps/web/src/hooks/useFilterRailTransition.ts`: `collapse()`/`expand()`; path = `typeof document.startViewTransition === 'function' && !prefersReducedMotion()` → set `<html data-rail-motion>`, `startViewTransition(() => { flushSync(set); focus() })`, remove the attribute on `finished` (`.catch().finally()`, so a rejected/skipped transition also clears) only if this click is still the latest owner (module-scope counter — the attribute is document-global). Otherwise `flushSync` + focus synchronously. `styles.css` (end of file, outside `@layer`): `rail-leave`/`rail-enter` keyframes, names only under `:root[data-rail-motion]`, all timings via `--motion-*`/`--ease-*`, comment explaining why the JS guards reduced motion. Addition beyond the Sally sketch: `::view-transition-group(filter-rail) { overflow: clip }` — without it the named rail paints above the root snapshot and slides across the sidebar nav for the whole exit; clipped, it slides away behind its own left edge (I11-D 「從側欄底下退場」).
- **T2** `FilterRailShell`: collapse button `aria-expanded="true"`, `collapseButtonRef` prop (prop, not forwardRef); `data-rail-vt="rail"` on `<aside>`, `data-rail-vt="trigger"` on the 「篩選 + badge」 group. `LibraryFilterRail`/`DiscoverFilterRail` forward `collapseButtonRef`. No `aria-controls`.
- **T3** `LibraryBrowseV2`: expand button gets `ref`, `aria-expanded={false}`, `data-rail-vt="trigger"`; `onCollapse`/`onClick` go through the hook; collapse focus = expand button ?? `headingRef` (selection mode). The `?? heading` check uses ref presence, not `getClientRects()` (jsdom returns no rects, and the button only renders when collapsed and not selecting). `toolbarFilterBtnRef`/`phoneFilterBtnRef` untouched; localStorage initialisers untouched.
- **T4** `DiscoverBrowseV2`: same wiring; no heading fallback (no selection mode); `<h1>` untouched; button position untouched (move is `disc-2026-10-discover-filter-button-leftmost`).
- **T5** New `useFilterRailTransition.spec.tsx` (9 tests: AC #1, #2, attribute lifetime, second click mid-transition, rejected `finished`, AC #4 reduced motion, AC #5 no API, AC #5 mount with remembered collapsed, missing focus target). `LibraryBrowseV2.spec.tsx` +3 (focus both ways + `aria-expanded` + trigger hook; tablet sheet button keeps its own `aria-expanded`; selection-mode collapse → `library-page-title`). `DiscoverBrowseV2.spec.tsx` +3 (focus both ways + `aria-expanded`; remembered collapsed renders collapsed and takes no focus; phone sheet button untouched). `FilterRailShell.spec.tsx` +3 (`aria-expanded="true"` / no `aria-controls`; ref reaches the button; `data-rail-vt` hooks, badge inside the trigger group).
- **T6** `pnpm nx test web` 299 files / **4615 passed**; `pnpm nx test api` green (42 packages ok); `pnpm run lint:all` 0 errors (172 pre-existing warnings, none in touched files); `format:check` clean; `python3 scripts/check-design-tokens.py` consistent; `tsc -p apps/web/tsconfig.app.json` — no errors in touched files (only the pre-existing jest-dom matcher typing errors in unrelated specs). E2E `tests/e2e/discover-filters.spec.ts` chromium **10/10**; firefox collapse test 3/3 green (it runs the real View Transition path — see below). Visual `--project=visual` local run **1 passed** (the single gallery test that covers `search-discover-filter-rail-unavailable`) — no baseline changed, nothing updated.
- **⚠️ 未查證 #1 resolved — rapid double click:** in Chromium, during the transition `document.elementFromPoint()` at the expand button's centre returns `<html>`; a second mouse click there does nothing (`startViewTransition` calls stayed at 1, rail stayed collapsed). So the mouse cannot double-toggle. The keyboard CAN: focus is moved to 「篩選」 inside the update callback, so a second Enter mid-animation starts a second transition (calls = 2, attribute = `expand` mid-way, `null` after, rail mounted, focus on `*-rail-collapse`). The latest-owner guard is what keeps that case clean, and it is pinned by the second-click unit test + mutation (b).
- **⚠️ 未查證 #2 resolved — Playwright Firefox:** Playwright 1.58 ships Firefox **146.0.1**; `typeof document.startViewTransition === 'function'` is **true** there, so Firefox runs the animated path; `discover-filters.spec.ts` collapse test green 3/3 on firefox. Also found: CI's E2E job runs `--project=chromium --project=webkit-core` only (`.github/workflows/test.yml:461`), and `webkit-core` does not match `discover-filters.spec.ts` — Firefox is never run in CI.
- 🔗 AC Drift: NONE (checked: `rail-collapse|rail-expand|railCollapsed` across `_bmad-output/implementation-artifacts/*.md` — 3 prior stories (ux3-0-7, ux3-3-2, dsr-1b-c), all REUSE: end states, testids, grid tables and persistence unchanged; this story only adds the in-between motion, focus and `aria-expanded`. `tests/e2e`/`tests/visual` hits: `discover-filters.spec.ts:272/276` only — unchanged and green).
- 📎 Contract Stamps: NONE (no `[@contract-v*]` stamps in this story or the upstream stories it cites).
- 🎭 A11y Pre-Flight: PASS (5 components checked — FilterRailShell, LibraryFilterRail, DiscoverFilterRail, LibraryBrowseV2, DiscoverBrowseV2; 0 jsx-a11y warnings on touched files, 0 introduced). Focus management verified in jsdom and in Chromium; `aria-expanded` on both buttons; no dangling `aria-controls`; tablet/phone sheet buttons' `aria-expanded` unchanged.
- 🎨 UX Verification vs `flow-i-advanced-search/i11-d.png`:

| Area | Design Spec (I11-D) | Implementation | Match? | Fix Needed |
|------|------|------|------|------|
| Collapse rail | translateX 0→−100% + fade, 240ms ease-leave | `rail-leave` 240ms `cubic-bezier(0.4,0,0.9,0.4)` (read back from the live animation) | ✅ | — |
| Expand rail | −100%→0 + fade-in, 320ms ease-settle, ~83% at 80ms | `rail-enter` 320ms `cubic-bezier(0.16,1,0.3,1)`; ~82% at 80ms in frame | ✅ | — |
| 篩選 morph | title ⇄ toolbar 篩選 N button | shared `filter-rail-trigger` name; frames show the title travelling to/from the button | ✅ | — |
| Grid | whole cross-fade 200ms, no per-card flight | root cross-fade 200ms, nothing in the grid named | ✅ | — |
| Content width | lands in one step, no width animation | no `transition-[width]`, no FLIP | ✅ | — |
| Reduced motion | instant, focus rules kept | 0 `startViewTransition` calls, rail gone at 30ms, focus correct | ✅ | — |
| Load / reload | draws remembered state, never animates | 0 calls on reload with collapsed persisted | ✅ | — |
| Focus / ARIA | collapse → 篩選 (`aria-expanded=false`); expand → 收合鈕 (`aria-expanded=true`) | as specified, both pages, mouse + keyboard | ✅ | — |
| Rail exit path | 「從側欄底下退場」 | slides behind its own left edge (24px right of the nav) via `overflow: clip` | ✅ | — |

🎨 UX Verification: PASS — implementation matches design screenshots.

### Discovery Triage

1. **Firefox: `discover-filters.spec.ts:169` 「[P0] browser back skips intermediate filter toggles」 fails 2/2 on HEAD code (my 9 changed files swapped back to `HEAD` for the run, then restored) and 3/3 with this story's change** — pre-existing, unrelated to the rail (back-navigation/replace semantics). Lane ③. ⚠️ **sprint-status entry NOT filed by me:** this run was restricted to editing only this story's sprint-status line. Proposed entry for whoever files it: `preexisting-fail-discover-back-nav-firefox: backlog  # tests/e2e/discover-filters.spec.ts:169 — firefox only; URL keeps rating_gte after goBack (filed by disc-2026-10-filter-rail-toggle-no-motion)`. Non-blocking: CI never runs Firefox.
2. **CI does not run Firefox E2E** (story Dev Notes/AC #9 assumed chromium/firefox). Information only — no work implied; recorded so the next story's assumptions are right. No entry needed.
3. Local dev servers on :8080 / :4200 were found **not listening** when the E2E step started (they served the browser verification minutes earlier; I did not stop them and found no process of mine that would). Playwright then started and tore down its own servers for the E2E/visual runs. Not restarted (seeded config unknown). Information only.

### Code Review（2026-10-06，主 session 對抗式審查，與 dev 不同 context）

- 讀過 `hooks/useFilterRailTransition.ts` 與 `styles.css` 新增段落；看過實機分格截圖（collapse 100ms／180ms 放慢版、end）。
- 確認：所有 `view-transition-name` 只在 `:root[data-rail-motion]` 期間存在（不會常駐造成新的堆疊層）；減少動態在 JS 擋（`*` 安全網碰不到 `::view-transition-*`）；只有點擊會觸發；連點靠 `motionOwner` 只讓最後一次拿掉屬性。
- 觀察（不修）：放慢看時，交叉淡化期間新舊兩版海報格會疊在一起（3 欄淡出、4 欄淡入），正常速度 200ms 內完成，是 I11-D 指定的做法。
- 結論：無阻擋項。

### File List

- apps/web/src/hooks/useFilterRailTransition.ts (new)
- apps/web/src/hooks/useFilterRailTransition.spec.tsx (new)
- apps/web/src/styles.css
- apps/web/src/components/ui/FilterRailShell.tsx
- apps/web/src/components/ui/FilterRailShell.spec.tsx
- apps/web/src/components/library/LibraryFilterRail.tsx
- apps/web/src/components/library/LibraryBrowseV2.tsx
- apps/web/src/components/library/LibraryBrowseV2.spec.tsx
- apps/web/src/components/search/DiscoverFilterRail.tsx
- apps/web/src/components/search/DiscoverBrowseV2.tsx
- apps/web/src/components/search/DiscoverBrowseV2.spec.tsx
- _bmad-output/implementation-artifacts/disc-2026-10-filter-rail-toggle-no-motion.md
- _bmad-output/implementation-artifacts/sprint-status.yaml

## Change Log

- 2026-10-06 Bob create-story：ready-for-dev。Ultimate context engine analysis completed - comprehensive developer guide created.
- 2026-10-06 Amelia dev-story：T1–T6 完成。共用 hook `useFilterRailTransition`（View Transitions＋flushSync，減少動態／不支援時直接切換）＋ `styles.css` 動畫規則；`FilterRailShell` 加 `aria-expanded`／ref／`data-rail-vt`；媒體庫、探索接上 hook 與焦點規則。單元 +18（web 4615 全綠）、E2E chromium 10/10、visual 無變化、Chromium 實機逐格確認。兩個 ⚠️ 未查證已查清（見 Completion Notes）。Status → review。
