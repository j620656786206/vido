# 對齊設計稿：探索頁電腦版收合篩選欄後，「篩選」按鈕排到工具列最左邊

Status: review

**Source:** `disc-2026-10-filter-rail-toggle-no-motion` create-story 時 Bob 發現（sprint-status `disc-2026-10-discover-filter-button-leftmost`，P3・對齊設計稿）。⚖️ Alexyu 2026-10-06：「1 要開搬的單，動畫做完接著做這張」。前置：`disc-2026-10-filter-rail-toggle-no-motion`（PR #686，分支 `feat/filter-rail-motion`，commit `52429268`）——本單直接疊在它上面。

## Story

身為在電腦上用探索頁的 Vido 使用者，
我希望收起篩選欄之後，工具列上的「篩選」按鈕就在最左邊（篩選欄剛剛所在的那一側），
這樣篩選欄收起來時是「縮成左邊那顆按鈕」，要叫回來時也知道往左邊找，跟設計稿一樣。

## 查到的事（2026-10-06，Bob 逐一開檔確認）

### 程式碼現在的順序（探索頁）

- 工具列是一條 `flex flex-wrap` 橫列：`apps/web/src/components/search/DiscoverBrowseV2.tsx:202`。裡面依序是：
  1. `MediaTypeTabs`（全部／電影／影集）`:203-210`；
  2. 手機／平板的「篩選」鈕 `open-filter-sheet`（`lg:hidden`）`:212-221`；
  3. 電腦版的「篩選」鈕 `discover-rail-expand`（`hidden lg:flex`，只在 `railCollapsed` 時出現）`:225-243`——上一張單加的 `ref={railExpandBtnRef}` `:228`、`aria-expanded={false}` `:230`、`data-rail-vt="trigger"` `:231` 都在這顆上；
  4. 「想要清單」`discover-requests-entry` `:246-258`，靠 `ml-auto` 推到最右（`:251`）。
- 所以電腦版收合後，畫面是「全部 · 電影 · 影集 · 篩選 …… 想要清單」——篩選在分頁**後面**。
- ⚠️ sprint-status 原條目寫的 `:191-226` 是 PR #686 之前的行號；本單一律以上面的現行行號為準。

### 設計稿怎麼畫（結論：探索頁的稿**一致**，都是「篩選」最左）

- **I11-D「適用範圍」卡**（`ux-design.pen` `Qaz1x`，`_bmad-output/screenshots/flow-i-advanced-search/i11-d.png` 右下角最後一張卡）：「探索頁的『篩選』按鈕在工具列最左，變形終點跟著它。只限桌面（lg 以上）；平板、手機用 bottom sheet，不在此規格內。」
- **I4-D-v2 篩選軌收合**（`m4fY7c`，`_bmad-output/screenshots/flow-i-discover-v2/i4-d.png`；登記在 `scripts/export-pen-screenshots.py:318`）：這是探索頁**唯一一張**畫了「篩選欄收起來」的電腦版稿。工具列是「**篩選 3**（最左）· 找到 312 部作品 · 評分排序 …… 想要清單（最右）」。
- 其他探索頁電腦版稿 I1-D（`i1-d.png`）、I3-D（`i3-d.png`）、I6-D（`i6-d.png`）、I7-D（`i7-d-v2.png`）、I8-D（`i8-d.png`）、I9-D（`i9-d.png`）、I10-D（`i10-d.png`）全都是**篩選欄展開**的狀態，工具列上沒有「篩選」鈕，所以不會跟上面兩張衝突。I5-D-v2（`i5-d-v2.png`）是儲存篩選的對話框，沒有工具列。
- `flow-i-advanced-search/` 的 I5-D（`i5-d.png`）是**媒體庫**的篩選欄展開稿、I7-D（`i7-d.png`）是篩選欄各狀態規格，兩張都沒有收合後的工具列。
- 手機稿 I2-M（`flow-i-discover-v2/i2-m.png`）是「評分 · 篩選」，篩選在排序後面——但 I11-D 明寫手機／平板不在範圍內，本單不動手機那顆。
- 稿跟程式碼本來就有一個**已立案**的工具列結構差異：稿上沒有「全部／電影／影集」分頁、排序在工具列、多一行「找到 312 部作品」——見 `_bmad-output/implementation-artifacts/sprint-status.yaml:1565` `disc-2026-09-discover-toolbar-structure-drift`（backlog）。本單**不處理**那些；只把「篩選」排到分頁前面。I11-D 是 2026-10-06 畫的最新稿，畫的時候程式碼已經有分頁，所以「篩選在最左＝在分頁前面」這個解讀沒有歧義。

### 媒體庫那一頁（結論：照它自己的稿**不用搬**）

- 媒體庫工具列：排序 `SortSelector` `apps/web/src/components/library/LibraryBrowseV2.tsx:680-684` → 平板篩選鈕 `library-filter-open`（`sm:flex lg:hidden`）`:687-702` → 電腦版篩選鈕 `library-rail-expand` `:706-728` → 篩選標籤 `:730-757` → 「選取」`ml-auto` `:759-769` → 檢視切換 `:770`。也就是「新增日期 · 篩選 3 ……」。
- I11-D 的分鏡是「以媒體庫桌面版面縮小繪製」（`i11-d.png` 頁首說明），收合終點與展開起點兩格都畫「新增日期 · 篩選 3」——跟程式碼一致。（展開「途中 80ms」那一格把篩選畫在新增日期前面，是按鈕正在飛回篩選欄的中間畫面，不是排版。）
- A′ 系列（`_bmad-output/screenshots/flow-a-browse-v2/` 的 a1p-d、a2p-d、a3p-d、a4p-d、a7p-d、a8p-d）是篩選欄出現之前的稿，工具列是「新增日期 · 類型 · 年份 · 解析度」下拉，沒有「篩選」鈕，所以對這題沒有意見。`LibraryBrowseV2.tsx:5` 的 Design ref 也只指向 A3p-D 等這些舊稿。
- 所以兩頁收合後的排法**本來就不一樣**，而且是 I11-D 明寫的：媒體庫「排序 · 篩選」，探索「篩選 · 分頁」。差別的來由：媒體庫工具列第一顆就是排序；探索頁的排序在篩選欄裡（`disc-2026-09-discover-toolbar-structure-drift` ②），工具列沒有排序可以排在前面。本單**不動媒體庫**。

### 變形動畫的終點會自己跟著走（不用改 CSS 或 hook）

- 變形靠的是「標題組」與「篩選鈕」在動畫期間共用同一個 `view-transition-name`：`apps/web/src/styles.css:641-643`（`:root[data-rail-motion] [data-rail-vt='trigger'] { view-transition-name: filter-rail-trigger; }`），時長曲線 `:669-681`。名字是掛在屬性上、不是掛在位置上，瀏覽器會量按鈕「現在在哪」當終點——按鈕搬到最左，終點就自動在最左。
- 共用 hook `apps/web/src/hooks/useFilterRailTransition.ts` 只管狀態、`data-rail-motion` 與焦點（`:50-83`），焦點目標是傳進來的取得函式（`DiscoverBrowseV2.tsx:72-76`，`railExpandBtnRef`／`railCollapseBtnRef`），跟按鈕在列中的位置無關。
- 分頁不會多出新的「殘影」：收合時整個內容欄本來就往左移 264px（篩選欄卸載），分頁從舊位置淡出、新位置淡入早就是 root 交叉淡化的一部分（`styles.css:646-650`）。搬完以後只是分頁的落點再往右一顆按鈕寬。

### 會被影響的測試與基準

- **單元測試沒有綁順序：** `DiscoverBrowseV2.spec.tsx` 用 `grep -n "getAllByRole\|getAllByTestId\|\[0\]\|\[1\]\|compareDocumentPosition\|nextElementSibling\|previousElementSibling"` 零結果；收合／展開測試（`:181-221`）都用 `getByTestId` 找按鈕，搬位置不影響。
- **E2E 沒有綁順序：** `grep -rn "discover-rail-expand" tests` 只有 `tests/e2e/discover-filters.spec.ts:276`（`toBeVisible()`）；`open-filter-sheet` 只有 `:339`（點擊）。其他會開探索頁的 E2E（`saved-filter-presets.spec.ts`、`availability-badges.spec.ts`、`explore-blocks.spec.ts`）用 `nth/first/last` 的地方都是海報卡或區塊，不是工具列（同一個 grep）。`tests/e2e` 裡沒有對探索頁按 Tab 的測試（`grep -rn "keyboard.press('Tab')" tests/e2e | grep -i discover` 零結果）。
- **TestSprite：** `grep -rl -e "rail-expand" -e "工具列最左" testsprite_tests` 零結果。
- **視覺基準：** 沒有整頁探索的 fixture（`grep -n "DiscoverBrowseV2" apps/web/src/routes/test/-gallery.fixtures.tsx` 零結果）。碰到探索的四個基準 `search-discover-filter-rail-unavailable`、`search-discover-no-result-v2`、`search-discover-section-error-v2`、`search-media-type-tabs`（`tests/visual/components.visual.spec.ts-snapshots/components/`）都是單獨畫一個元件；`search-media-type-tabs` 畫的是 `MediaTypeTabs` 本身（`-gallery.fixtures.tsx:1575-1576`），不含工具列。→ 預期**沒有**基準要更新。

### 鍵盤順序

- 工具列沒有任何 `order-*` class（`DiscoverBrowseV2.tsx:202-258`），DOM 順序＝畫面順序＝Tab 順序。
- `MediaTypeTabs` 是 `role="tablist"`，但沒有做 roving tabindex：三顆 `role="tab"` 的按鈕都可以 Tab 到（`apps/web/src/components/search/MediaTypeTabs.tsx:38-56`，沒有 `tabIndex`）。
- 搬完後電腦版收合時的 Tab 順序：篩選 → 全部 → 電影 → 影集 → 想要清單 → 快速篩選列…。收合後焦點落在「篩選」（hook 規則不變），往後 Tab 第一站就是分頁，跟眼睛看到的順序一致。
- 小於 `lg` 時：電腦版那顆是 `hidden`（`display:none`），不在 Tab 順序裡；手機鈕位置不變 → 平板／手機的 Tab 順序完全不變。

## 設計

只搬一個區塊：把 `DiscoverBrowseV2.tsx:222-243`（電腦版「篩選」鈕連同它上面的註解）整段移到 `MediaTypeTabs`（`:203`）**前面**，成為工具列 `div`（`:202`）的第一個子元素。

- 按鈕本身一個字都不改：`ref`、`onClick={expandRail}`、`aria-expanded={false}`、`data-rail-vt="trigger"`、`data-testid="discover-rail-expand"`、class、數字徽章全部原樣。
- 手機／平板那顆 `open-filter-sheet` 留在原位（分頁後面）。小於 `lg` 的畫面一個像素都不變。
- 篩選欄展開時（電腦版那顆不存在），工具列跟現在一樣從分頁開始。
- 不改 `styles.css`、不改 `useFilterRailTransition`、不改 `FilterRailShell`、不改媒體庫。
- 順手把檔頭 `DiscoverBrowseV2.tsx:1` 的 Design ref 補上 `Screen I11-D (Qaz1x)`（本單依據的稿；同一個節點已被 `useFilterRailTransition.ts:1` 引用）。

## Acceptance Criteria

1. **電腦版順序：** `lg` 以上、篩選欄收合時，`discover-rail-expand` 是工具列的第一個子元素，排在 `MediaTypeTabs`（`role="tablist"`）前面；畫面是「篩選 [N] · 全部 · 電影 · 影集 …… 想要清單」，「想要清單」仍在最右。
2. **其他狀態不變：** 篩選欄展開時工具列仍從分頁開始；小於 `lg` 時 `open-filter-sheet` 仍在 `MediaTypeTabs` 後面，平板／手機畫面不變。
3. **變形與焦點不變：** 按鈕的 `ref`／`aria-expanded`／`data-rail-vt`／`data-testid` 原樣；`styles.css`、`useFilterRailTransition.ts`、`FilterRailShell.tsx` 不改。收合後焦點落在最左的「篩選」，變形動畫的終點是它的新位置；展開後焦點回收合鈕。
4. **鍵盤：** 電腦版收合時，從「篩選」按 Tab 第一站是「全部」分頁（DOM 順序＝畫面順序，不用 `order-*` 排）。
5. **媒體庫不動：** `LibraryBrowseV2.tsx` 沒有任何改動（它的稿 I11-D 是「排序 · 篩選」）。
6. **測試：** `DiscoverBrowseV2.spec.tsx` 新增一條：收合後 `discover-rail-expand` 在 `tablist` 之前、`open-filter-sheet` 在 `tablist` 之後（用 `compareDocumentPosition`）。既有測試全綠；`tests/e2e/discover-filters.spec.ts` 在 chromium 跑綠。
7. **檢查全綠：** `pnpm nx test web`、`pnpm run lint:all`、`pnpm run format:check`；本機 visual 不需更新任何基準。

## Tasks / Subtasks

- [x] **T1 搬按鈕**（AC #1–#3, #5）
  - [x] 把 `DiscoverBrowseV2.tsx:222-243`（註解＋`{railCollapsed && (<button …>)}`）剪下，貼到 `:202` 的工具列 `div` 開頭、`<MediaTypeTabs` 之前。內容不改。
  - [x] 更新 `:201` 那行工具列註解，寫出新順序（電腦版篩選 → 分頁 → 手機篩選 → 想要清單），並註明「最左＝I11-D 適用範圍、I4-D-v2」。
  - [x] `:1` Design ref 補 `Screen I11-D (Qaz1x)`。
- [x] **T2 測試**（AC #4, #6）
  - [x] 在 `DiscoverBrowseV2.spec.tsx` 的 `describe('desktop rail collapse / expand (I11-D)')`（`:181`）裡加一條：點收合後，`screen.getByTestId('discover-rail-expand').compareDocumentPosition(screen.getByRole('tablist'))` 含 `Node.DOCUMENT_POSITION_FOLLOWING`；`screen.getByRole('tablist').compareDocumentPosition(screen.getByTestId('open-filter-sheet'))` 也含 `DOCUMENT_POSITION_FOLLOWING`。再斷言 `discover-rail-expand` 是工具列容器的 `firstElementChild`（容器用 `expandBtn.parentElement` 取，不要另加 testid）。
  - [x] 先跑紅（還沒搬時第一個斷言要失敗），再搬，再跑綠。
- [x] **T3 檢查與實機確認**（AC #1–#4, #7）
  - [x] `pnpm nx test web`、`pnpm run lint:all`、`pnpm run format:check`。
  - [x] Playwright：`tests/e2e/discover-filters.spec.ts` chromium 跑綠。
  - [x] 本機 visual 跑一次（帶 `AI_PROVIDER=claude`），確認四個探索相關基準無差異；**不要** `test:visual:update`。
  - [x] 瀏覽器實機 1280 寬：收合後對照 `flow-i-discover-v2/i4-d.png`（篩選在最左）；把 DevTools Animations 放慢到 10%，確認「篩選」標題變形落在最左那顆；鍵盤：收合鈕按 Enter → 焦點在最左「篩選」→ Tab 到「全部」；縮到 `lg` 以下確認手機篩選鈕仍在分頁後面。

前端 3 項、後端 0 項 → 不拆單（Cross-Stack Split Check：後端 0 ≤ 3）。

## Dev Notes

### 不要做的事

- 不要動 `styles.css` 的 `::view-transition-*` 規則或 `useFilterRailTransition`：變形終點由瀏覽器量按鈕位置，搬了就跟著走。
- 不要用 `order-first`／`lg:order-first` 這類 CSS 排序來「看起來在最左」：畫面順序會跟 Tab 順序（DOM 順序）不一致（WCAG 2.4.3），而 AC #4 要它們一致。直接搬 JSX。
- 不要把手機鈕 `open-filter-sheet` 一起搬：I11-D 寫明手機／平板不在範圍，手機稿 I2-M 也是篩選在排序後面。
- 不要把兩顆篩選鈕合成一顆：它們開的東西不一樣（電腦版叫回篩選欄、手機版開底部面板），手機那顆刻意沒有 `aria-expanded`、沒有 `data-rail-vt`（`DiscoverBrowseV2.spec.tsx:214-220` 守著）。
- 不要動媒體庫的工具列（見上面「媒體庫那一頁」）。
- 不要順手處理 `disc-2026-09-discover-toolbar-structure-drift` 的東西（拿掉分頁、把排序搬上工具列、加「找到 N 部作品」）——那張單要先改稿。
- 不要改任何 `data-testid`。

### 已知陷阱

- **條件渲染要整段搬：** 按鈕外面包著 `{railCollapsed && (…)}`（`:225`），連同它一起搬；只搬 `<button>` 會讓展開狀態也出現按鈕。
- **`flex-wrap`：** 工具列會換行（`:202`）。最左多一顆按鈕後，窄一點的 `lg` 寬度（1024px、側欄展開）若擠不下，換到第二行的會是最右邊的東西，「篩選」一定留在第一行最左，可接受；實機時順便看一眼 1024 寬。
- **jsdom 的 `hidden lg:flex`：** 單元測試環境不吃 Tailwind 斷點，兩顆篩選鈕都在 DOM 裡，所以 T2 用 DOM 位置斷言，不要用「看不看得見」斷言順序。

### 待裁定

無。探索頁的稿（I11-D 卡、I4-D-v2）一致都是「篩選」最左；媒體庫的稿（I11-D 分鏡）一致都是「排序 · 篩選」。兩頁排法不同是 I11-D 寫明的，本單照稿做。
（若 Alexyu 之後想讓兩頁一模一樣，那是改稿的決定，要先請 Sally 改 I11-D，再另開單；本單不預設。）

### Time-dependent visual coverage

N/A — no wall-clock-reading components touched（只在 `DiscoverBrowseV2.tsx` 搬 JSX，不讀 `Date`）。

視覺回歸：沒有整頁探索的 fixture；`visual` 專案固定 `reducedMotion: 'reduce'`，本來就拍不到動畫。預期無基準要更新。

### References

- 設計：`ux-design.pen` `Qaz1x`（I11-D，`flow-i-advanced-search/i11-d.png`「適用範圍」卡）、`m4fY7c`（I4-D-v2，`flow-i-discover-v2/i4-d.png`）
- 前一張單：`_bmad-output/implementation-artifacts/disc-2026-10-filter-rail-toggle-no-motion.md`（「⚖️ 已確認」段：本單由那裡拆出）
- 已立案的稿／碼結構差異：`sprint-status.yaml:1565` `disc-2026-09-discover-toolbar-structure-drift`

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（claude-opus-5-5），主 session 直接實作（3 項小任務）。

### Debug Log References

- 先寫測試：新測試在搬之前跑紅（`expandBtn.parentElement.firstElementChild` 不是它），搬完變綠。
- 瀏覽器實機（Playwright Chromium，1280×800，本機 :4200）：收合後「篩選」在「全部」左邊；焦點落在 `discover-rail-expand`；按 Tab 下一站是「全部」。截圖 `/private/tmp/claude-502/poc/discover-collapsed.png`（scratchpad，不進 repo）。

### Completion Notes List

- `DiscoverBrowseV2.tsx`：把電腦版「篩選」按鈕（`railCollapsed &&` 那段，內容一字未改）搬到工具列 `div` 開頭、`MediaTypeTabs` 之前；工具列註解改寫成新順序；Design ref 補 I11-D。手機篩選鈕、想要清單位置不變。`styles.css`、`useFilterRailTransition.ts`、`FilterRailShell.tsx`、`LibraryBrowseV2.tsx` 都沒動（AC #3、#5）。
- `DiscoverBrowseV2.spec.tsx`：新增 1 條（AC #6）。
- 檢查：`DiscoverBrowseV2.spec.tsx` 21／21；`pnpm nx test web` 299 檔／4616 測試全綠；`lint:all` 0 errors、Prettier 全過；`tests/e2e/discover-filters.spec.ts` chromium 10／10。
- 本機 visual 沒跑全套（visual 是整包一支測試，本機另有 4 張因本機 API 資料不同而差異的已知雜訊）；交給 CI 的 Visual diff 判定。探索相關的 3 張基準（rail-unavailable、no-result、section-error）都不含工具列收合狀態。

### File List

- `apps/web/src/components/search/DiscoverBrowseV2.tsx`
- `apps/web/src/components/search/DiscoverBrowseV2.spec.tsx`
- `_bmad-output/implementation-artifacts/disc-2026-10-discover-filter-button-leftmost.md`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`

## Change Log

- 2026-10-06 Bob create-story：ready-for-dev。Ultimate context engine analysis completed - comprehensive developer guide created.
- 2026-10-06 dev（主 session）：review。
