# Story DSR.1b-c：手機媒體庫的四張畫面（空白／骨架／網格／未匹配）與設計稿雙向對齊——稿不再畫產品沒有的東西，骨架不再跟網格對不齊

Status: ready-for-dev

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As the person who opens 媒體庫 on a phone,
I want 載入中的骨架長得跟載完的網格一樣、空片庫說的話跟桌機一樣、未匹配篩選狀態就是網格加一顆膠囊，而且設計稿畫的每一顆按鈕都是真的按得到的,
so that 畫面不會在載完的那一刻跳一下，我也不會被稿上根本不存在的「4K」「未看完」快速篩選誤導。

## Context

`dsr-1b`（Flow A 手機）拆出來的**第三塊：四張畫面的版面**。**在 `dsr-1b-b` 之後做**（兩張都碰 `LibraryBrowseV2` 標題列：`-b` 加篩選鈕，本張對標題字級與計數；先後順序避免衝突）。不依賴 `-a`。

**這張的方向以「碼→稿」為主。** 四張手機稿是 2026-09-10 補內容時畫的，當時桌機稿 A1p-D 已在 dsr-1 依程式碼重畫過、手機稿沒有跟上：A1p-M 的空白文案是 dsr-1 已作廢的「第四種寫法」；A3p-M 畫了一列產品沒有的「快速篩選」（全部／動畫／4K／2020s／未看完）；E4-M 還是舊的瀏覽版面（「掃描結果」標題、頁內排序／篩選鈕）。程式碼側真正要修的是**骨架欄數與網格不同**（每次載入都會 reflow）與**手機標題字級**。

| 稿 | 節點 | 稿畫了什麼 | 程式碼現況 | 方向 |
| --- | --- | --- | --- | --- |
| A1p-M · 空白資料庫 | `BfGVZ` | 「片庫還是空的」＋「設定媒體資料夾後，掃描會自動辨識並整理你的電影與影集。」＋CTA「設定媒體資料夾」＋「或先了解掃描設定 →」；**上面還有一列篩選 chip** | `EmptyNoFolder`／`EmptyNoQBT`／`EmptyReadyForScan` 三態分類器（dsr-1 已把 A1p-D 重畫成 `EmptyNoFolder`）；空片庫沒有 chip 列 | **碼→稿** |
| A2p-M · 載入骨架 | `qBWQC` | 2 欄 × 3 列、卡 171×274（海報 236＋兩條線 12／10） | `LibraryGridSkeletonV2`（`LibraryStatesV2.tsx:30`）12 格、`grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-6`——**與真網格的欄數不同**（`LibraryBrowseV2.tsx:493-495` 是 `grid-cols-2 sm:grid-cols-3 lg:grid-cols-3 xl:grid-cols-4 2xl:grid-cols-5`） | 欄數：**碼**（自己對自己）；格子尺寸：**稿→碼**（骨架＝真卡比例） |
| A3p-M · 內容網格 | `h1v1U6` | 頂列「電影」H4 18＋`search`＋`sliders-horizontal` 兩顆 44×44；**篩選 chip 列（全部／動畫／4K／2020s／未看完）**；2 欄 `PosterCard-v2` 171×294、gap 16、padding [12,16,16,16]；**沒有計數** | `AppShellV2` 頂列（56，含 `mobile-search-toggle`）＋頁面 `<h1>`「電影」＋計數「N 部」（dsr-1 裁定放頁首）；`FilterChips` 只在有生效篩選時出現；`grid-cols-2`、頁面 `px-4 py-6` | chip 列：**碼→稿**；計數：**碼→稿**；標題字級：**稿→碼**（DESIGN.md 手機降一階） |
| E4-M · 未比對篩選 | `n7jVF` | 「掃描結果」標題、「狀態： 未比對」chip＋「清除」、「42 項未比對」、頁內 `新增日期▾`＋`篩選` 兩顆鈕、4 張 `PosterCard-v2/Unmatched` 173×296 gap 12 | 同一個 `LibraryBrowseV2` 的 `?unmatched=true`：`<h1>`＋計數、膠囊「未匹配 ×」、`PosterCardV2` unmatched 變體（dsr-5 已對母版）、入口在頂列（`-b`） | **碼→稿**（重畫成「A3p-M＋一顆膠囊」） |

⚠️ **驗收基準是 `.pen` 節點值**。PNG 只是參考。
⛔ **本張不進**：抽屜與篩選鈕（`-b`）、後端、桌機 ≥640 的任何版面、`PosterCardV2` 母版形狀（`disc-2026-09-postercard-v2-badge-shape-drift`／`-cannot-right-anchor`）、把頂列標題搬進 shell（`disc-2026-09-library-header-not-in-shell-bar`）、「未比對／未匹配」二字之爭（`disc-2026-09-unmatched-two-words`）、空片庫三態各給一張稿（`disc-2026-09-library-empty-three-states-one-frame`，Sally 的版面決定）。

### 🔴 建單時查到的事（main `49cc74a3`；行號皆為現況）

1. **骨架欄數 ≠ 網格欄數**（`LibraryStatesV2.tsx:36` vs `LibraryBrowseV2.tsx:493-495`）。在 `lg` 骨架 4 欄、網格 3 欄（軌展開）；`xl` 骨架 6 欄、網格 4 欄。手機兩邊都是 2 欄，所以**手機看不到這個 bug，但它是同一個元件**——修在同一處、桌機一起好。⚠️ 修法是**共用一個常數**（網格的兩組 class：軌展開／收合），不是把骨架也寫一份。
2. `LibraryGridSkeletonV2`（`:30`）：`data-testid="library-grid-skeleton"`、`aria-busy`、`aria-label="載入中"`、12 格。格子的形狀要看現況（dev 讀）；目標是與 `PosterCardV2` 同比例（海報 2:3＋標題列＋中繼列）。
3. 空白三態在 `LibraryStatesV2.tsx`／`LibraryBrowseV2.tsx:662-686`（`classifyEmptyState()`）；`EmptyNoFolder` 的文案與 CTA 是 A1p-D 現在畫的（dsr-1 AC #6）。**A1p-M 抄 A1p-D 的字，不要抄稿上舊的**。CTA 按鈕在手機要 ≥44 高（DESIGN.md 觸控目標；桌機尺寸不動）。
4. 標題列 `LibraryBrowseV2.tsx:513-528`：`<h1 data-testid="library-page-title">`（`TYPE_TITLE`：媒體庫／電影／影集）＋`<span data-testid="library-result-count">`「N 部」（`isLoading`／`isError` 時隱藏）。字級 class 要 dev 讀現況；DESIGN.md `:604-611`「手機標題降一階」（Title 24→20）。
5. 三張手機稿的頂列（`電影`＋兩顆圖示鈕）是 **shell 的 TopAppBar**，在真實 app 裡對應 `AppShellV2` 的 56 高頂列（`search` ＝既有的 `mobile-search-toggle` `:92-97`）＋頁面自己的標題列。這與 dsr-4b-1 🔴 #12 是同一件事；「把頁面標題釘進 shell」是 `disc-2026-09-library-header-not-in-shell-bar`，本張不動——**稿改成畫真實的兩層**（shell 頂列＋頁面標題列＋計數），與 Flow D 的 D1-M-v2 同一種畫法（看 `uMDjw` 怎麼畫）。
6. A3p-M 的 chip 列 `全部／動畫／4K／2020s／未看完`：**「4K」「未看完」在產品裡不存在**（`disc-2026-09-library-has-filters-the-product-lacks`），「全部」是媒體類型不是篩選。程式碼的 `FilterChips` 是**生效篩選**的膠囊（每顆帶 ×）。稿改成畫生效篩選（例：「動畫 ×」「缺字幕 ×」），空片庫那張不畫。
7. A2p-M 的骨架 171×274 vs A3p-M 真卡 171×294——**稿自己就對不上**；改稿讓骨架＝真卡尺寸。`skr2`／`r2`／`mGridRow2` 三處 partially clipped 是刻意的「露出半列表示可捲」，**不算 problems 增加**。
8. A3p-M `y2MDfn` 節點名「c-駭客任務」內容「瀑布」——命名殘留，順手改名。
9. E4-M（`n7jVF`）：card 173×296 gap 12 ≠ A3p-M 171×294 gap 16；tabbar active 標籤 600／`$accent-text` ≠ Flow A 三張的 700／`$accent-primary`；文案「42 項未比對」≠ 桌機 E4-D「顯示 42 項未比對媒體」≠ 程式碼「N 部」。重畫成 A3p-M 的結構後這些自然消失。**膠囊的字**：程式碼是「未匹配」、稿是「未比對」——本張**不裁**；重畫時膠囊文字**維持稿的「未比對」**並在規格註記標明待 `disc-2026-09-unmatched-two-words` 裁定（文末問題 1）。
10. 視覺夾具：`library-empty-no-folder`／`-no-qbt`／`-ready-for-scan`／`library-library-grid` 都是 `width` 框寬夾具（沒有 390 viewport 版）；`library-grid-skeleton` **沒有夾具**。抽屜以外的手機夾具可以用 `width: 390`（不是 Portal，不需要 `viewport`）——先例看 `-gallery.fixtures.tsx` 裡其他 `width: 390` 的條目。
11. e2e：`tests/e2e/library-mobile.spec.ts` 由 `-b` 建立；本張**追加**條目，不另開檔。`empty-library.spec.ts` 既有（桌機）——不改。
12. Rule 21 檔頭現況：`LibraryStatesV2.tsx` 寫 `Design ref: … A2p-D／A7p-D／A8p-D`（dsr-1 改的）；`LibraryBrowseV2.tsx:1-2` 寫 `A3p-D (LcHBs) · A4p-D (b1H71g)`（`-b` 會補 `A6p-M`）。本張補手機節點。

### 設計稿節點

| 代號 | 節點 | 位置 | 說明 |
| --- | --- | --- | --- |
| A1p-M | `BfGVZ` | 17040,10993 | `empty-m`（`R5pmG`）：72 圓＋`clapperboard` 32、H4 標題、Body 內文 300 寬、CTA 162×44 `$accent-primary`、次要連結；`uqgW0` filterbar **要刪** |
| A2p-M | `qBWQC` | 17530,10993 | `sk-grid-m` 3 列 × 2 欄，卡 171×274（`p` 236／`l1` 171×12／`l2` 60×10）**要改成 240／294 比例** |
| A3p-M | `h1v1U6` | 18020,10993 | topbar 52（`電影` H4＋`search`＋`sliders-horizontal`）、filterbar 44（**要改成生效篩選膠囊**）、`grid`（`wW2oF`）2 欄 × 3 列 `PosterCard-v2`（`hD7Tw`）171×294 gap 16 padding [12,16,16,16]；`y2MDfn` **改名** |
| E4-M | `n7jVF` | 18510,28389（Flow E 群組） | **整張重畫**成 A3p-M 結構＋膠囊「未比對 ×」＋計數；4 張 `PosterCard-v2/Unmatched`（`n6Crb`）改成 171×294 gap 16；刪 `掃描結果`／`mControls` |
| 對照 | `uMDjw`（D1-M-v2） | 17040,24325 | shell 頂列＋頁面標題列的畫法範本 |
| 群組 | `p6EGC` → `JUkwx`（手機）；Flow E 群組見 `n7jVF` 的父層 | | 規格註記放 `JUkwx` 下方（與 `spec-note-dsr-1b-b` 排開） |

共用殼：tabbar 84 高 `$bg-secondary`、5 個 `Component/MobileTabItem`（`S86VM`），active `$accent-primary` 700。字階：H4 18、BodyLg 16、Body 14、Label 12。

---

## Acceptance Criteria

1. **設計稿先改（Pencil MCP；每個節點動手前再 `Get` 一次）。**
   - **A1p-M** `BfGVZ`：刪 `uqgW0` filterbar；`empty-m` 文案、CTA 文字、次要連結**逐字改成 A1p-D 現在畫的**（dev 用 `Get` 讀 `vZpT8` 的文字節點抄，不要抄程式碼以外的來源；兩邊必須與 `EmptyNoFolder` 元件的字相同）；CTA 高度 ≥44。加一則小註記（比照 A1p-D 上 dsr-1 加的那則）說明 `EmptyNoQBT`／`EmptyReadyForScan` 何時出現。
   - **A2p-M** `qBWQC`：6 張骨架卡改成 171×294（海報 240、標題線 171×12、中繼線 60×10，間距與 `PosterCard-v2` 的 `textcol` gap 相同）；第三列仍露半列。
   - **A3p-M** `h1v1U6`：頂列改畫「shell 頂列＋頁面標題列」兩層（照 `uMDjw`）：頁面標題列左「電影」＋計數「1,284 部」（Body `$text-secondary`，與 A3p-D 頁首同一種畫法）、右側留一顆 44×44 篩選鈕的位置（**鈕本身由 `-b` 畫**——若 `-b` 已合併就是已經有了，不要畫第二顆）；filterbar 改成**生效篩選膠囊**：兩顆帶 × 的膠囊「動畫」「缺字幕」＋「清除全部」（形狀照 `FilterChips`：pill、`$bg-tertiary`、Label 12、× 11）；`y2MDfn` 改名 `c-瀑布`。
   - **E4-M** `n7jVF`：重畫成 A3p-M 的結構（同一個頂列／標題列／膠囊列／2 欄網格 gap 16），差別只有：膠囊一顆「未比對 ×」＋「清除全部」、計數「42 部」、4 張 `PosterCard-v2/Unmatched` instance 171×294。刪「掃描結果」標題與 `mControls`。tabbar active 標籤改成與 Flow A 一致（700／`$accent-primary`）。**膠囊文字維持「未比對」**（🔴 #9）。
   - 規格註記 `spec-note-dsr-1b-c`（`JUkwx` 下方、與 `spec-note-dsr-1b-b` 錯開；樣式同 4b-1）：
     > 「A1p／A2p／A3p／E4（手機）：頂列是 shell 的（搜尋鈕在那裡），頁面自己再有一列標題＋計數（「N 部」，與桌機同一個位置）。載入骨架與網格用同一組欄數（手機 2 欄），格子＝海報卡的比例，載完不會跳。膠囊列只畫生效中的篩選（每顆可移除），空片庫沒有膠囊列；沒有「4K」「未看完」這類快速篩選。E4 是同一個網格加一顆「未比對」膠囊，沒有自己的標題與頁內排序鈕（入口在頂列的篩選鈕）。「未比對／未匹配」用哪個字待裁定（disc-2026-09-unmatched-two-words）。」
   - 收尾：`problems` **不得增加**（三處刻意露半列維持）；存檔走選單 Save（osascript 同 4b-1）、`git status --porcelain ux-design.pen` 出現 ` M`、grep 磁碟檔確認 `spec-note-dsr-1b-c` 與 `c-瀑布`；**存檔後**才匯出；**只 stage** `flow-a-browse-v2/a1p-m.png`、`a2p-m.png`、`a3p-m.png`、`flow-e-scanner/e4-m.png`（確認 `SCREENS` 的 key 名稱）與 `pen-tokens.json`，其餘 `git checkout --`。
   - 📎 Pencil：`execute` 失敗會 rollback；`Replace` 重設沒寫的屬性；`Copy` 一般節點 `descendants` 名稱 key 靜默忽略；新 `Insert` 的節點存檔前截圖不出來（先存再驗）。

2. **骨架與網格共用欄數（🔴 #1）。**
   - 在 `components/library/` 抽一個常數模組（例：`libraryGrid.ts` export `LIBRARY_GRID_COLS_RAIL_OPEN`／`LIBRARY_GRID_COLS_RAIL_COLLAPSED`，內容＝`LibraryBrowseV2.tsx:493-495` 現在的兩串 class **逐字**），`LibraryBrowseV2` 與 `LibraryGridSkeletonV2` 都 import。骨架多接一個 `railCollapsed?: boolean` prop（預設 false）讓 `LibraryBrowseV2` 傳入當下的軌狀態。
   - 骨架格子改成與 `PosterCardV2` 同結構：海報 `aspect-[2/3]` 圓角 `$radius-lg` `$bg-tertiary`、標題線 `h-3 w-full` `$bg-tertiary`、中繼線 `h-2.5 w-[60px]` `$bg-secondary`，列距與真卡 `textcol` 相同。12 格不變。`data-testid`／`aria-*` 不變。
   - **桌機基準線會變**（骨架在 `lg`／`xl` 從 4／6 欄變成 3／4 欄）——這是修 bug 不是回歸；`library-grid-skeleton` 今天沒有夾具，**本張補一個**（AC #5），所以沒有既有基準線被改，只有新的。

3. **手機標題字級與 CTA 觸控（🔴 #3, #4）。**
   - `<h1 data-testid="library-page-title">` 在 <640 降一階：依 DESIGN.md `:604-611`（Title 24→20），寫成 `max-sm:` 變體；≥640 一個像素不變。計數位置與文字不動（dsr-1 裁定）。
   - `EmptyNoFolder`／`EmptyNoQBT`／`EmptyReadyForScan` 的 CTA 在 <640 命中區 ≥44 高（`max-sm:min-h-11`）；桌機不變。
   - 網格在 <640 的 gap 與稿一致（稿 16 ＝ `gap-4`；dev 讀現況，若已是 16 就不動並在 Completion Notes 寫「已相符」）。
   - Rule 21：`LibraryStatesV2.tsx` 檔頭補 ` · A1p-M (BfGVZ) · A2p-M (qBWQC)`；`LibraryBrowseV2.tsx:1-2` 補 ` · A3p-M (h1v1U6) · E4-M (n7jVF)`（照該檔既有寫法；`-b` 已補的 `A6p-M` 保留）。

4. **既有的行為不准回歸。**
   - ≥640：`library-empty-*`、`library-library-grid`、`poster-card` 既有基準線**零變動**；`LibraryStatesV2.spec.tsx`、`LibraryBrowseV2.spec.tsx` 既有斷言一條不改。
   - `classifyEmptyState()` 的三態分類**不動**（⛔ 不要為了對稿合併成一態——dsr-1 AC #6 的禁令）。
   - ⛔ 不改 `PosterCardV2.tsx`、`FilterChips.tsx`、`LibraryFilterSheetV2.tsx`、後端、`AppShellV2`、`MobileTabBar`。

5. **測試。** 紅／守（Rule 16）。
   - `LibraryStatesV2.spec.tsx`：（紅）骨架容器的 class 字串 **等於** `LIBRARY_GRID_COLS_RAIL_OPEN`（`railCollapsed` 時等於另一串）——兩者 import 同一個常數比對，拿掉共用就紅；每格有 `aspect-[2/3]` 的海報佔位＋兩條線；CTA 帶 `max-sm:min-h-11`。（守）`aria-busy`、`aria-label="載入中"`、`library-grid-skeleton`、三態文案既有斷言。
   - `LibraryBrowseV2.spec.tsx`：（紅）網格容器 class 等於同一個常數（軌展開／收合各一條）；`h1` 帶 `max-sm:` 字級 class；骨架收到與網格相同的 `railCollapsed`。（守）既有三個 describe。
   - **視覺夾具**（AC #2 說的新基準線）：`library-mobile-screens/skeleton`（`LibraryGridSkeletonV2`、`width: 390`、`penNode: 'qBWQC'`）、`library-mobile-screens/empty-no-folder`（`EmptyNoFolder`、`width: 390`、`penNode: 'BfGVZ'`）；再加 `library-grid-skeleton` 桌機一張（`width: 1280`、`penNode: 'EsoIv'`——A2p-D）。只產 darwin；`-linux` 走 bootstrap PR。
   - **e2e**（追加到 `tests/e2e/library-mobile.spec.ts`）：
     - **390・骨架＝網格**：以延遲的 stub 讓骨架先出現——量骨架第一列兩格的 x／寬，再等網格出現量第一列兩張卡的 x／寬，**兩組差 ≤1px**（這一條就是「載完不會跳」的驗收）。
     - **390・空片庫**：stub 空清單＋無資料夾 → `EmptyNoFolder` 的 CTA 高 ≥44、文字與 `.pen` `BfGVZ` 標題逐字相同（字串寫死在測試裡、註明來源節點）。
     - **390・E4**：開 `/library?unmatched=true` → 膠囊列有「未匹配」（程式碼現況的字）＋計數可見、**沒有**「掃描結果」文字、卡片 `data-variant="unmatched"`（或該元件既有的辨識屬性）。
     - **640**：`h1` computed `font-size` 是桌機值（與 1280 相同）。
   - **每一項修法做 mutation check**（拿掉 → 必須紅），結果寫進 Completion Notes。

6. **另立的單子（建單時已寫入 sprint-status）**：無新立。↪ 補記：`disc-2026-09-unmatched-two-words`（E4-M 重畫後只剩這一個字未決）、`disc-2026-09-library-header-not-in-shell-bar`（手機稿改成兩層之後，這張單子涵蓋桌機＋手機）、`disc-2026-09-library-has-filters-the-product-lacks`（A3p-M 的「4K／未看完」快速篩選列由本張刪除）。

7. **CI 全綠**：`pnpm run format:check`、`pnpm run lint:all`、`pnpm nx run web:typecheck --skip-nx-cache`、`python3 scripts/check-design-tokens.py`、`pnpm nx test web`。⛔ 絕不 `run_in_background` 跑測試；Nx 一律 `NX_DAEMON=false`。🚨 合併之後要看 `main` 那一次的 Tests／Docker／Visual Regression 三條。gh 操作一律帶 `GH_TOKEN=$(gh auth token --user j620656786206)`。

## Tasks / Subtasks

- [ ] **Task 1 — 設計稿：A1p-M 文案＋刪 chip 列、A2p-M 骨架尺寸、A3p-M 兩層頂列＋生效膠囊＋改名、E4-M 重畫、規格註記（AC: #1）**
- [ ] **Task 2 — 骨架與網格共用欄數常數、骨架格子形狀（AC: #2, #5）**
- [ ] **Task 3 — 手機 h1 字級、CTA 觸控高、網格 gap 核對、Rule 21 檔頭（AC: #3, #5）**
- [ ] **Task 4 — 三個視覺夾具、e2e 四條、mutation check、收尾（AC: #4, #5, #6, #7）**
  - [ ] dev-story Step 9：`a1p-m`／`a2p-m`／`a3p-m`／`e4-m`

## Dev Notes

### 這張的重點

- **大部分工作在稿上。** 程式碼真正要修的只有兩件：骨架欄數（bug）與手機標題字級。不要為了「對稿」去改稿上本來就錯的東西（4K、未看完、掃描結果、片名）。
- **共用常數，不是複製 class。** 骨架與網格的欄數只能有一個來源，測試用「等於同一個 import」釘住。
- **A1p-M 抄 A1p-D，A1p-D 抄程式碼。** 空片庫文案的正典在 `EmptyNoFolder`。
- **E4-M 是「A3p-M＋一顆膠囊」**，不是一張獨立的頁。

### 上游契約（Rule 20）

- 本張不定義也不消費任何 `[@contract-v*]`。`?unmatched=true` 是既有的 implicit v0。

### 建單裁定（2026-09-22，SM；Sally／Alexyu 可在 review 推翻）

1. ⚖️ **拆成三張**（見 `-a` 裁定 1）；本張排在 `-b` 之後。
2. ⚖️ **快速篩選列不做**（碼→稿）：稿上的「全部／動畫／4K／2020s／未看完」有兩個是產品沒有的、一個是媒體類型；程式碼的生效篩選膠囊是已出貨的行為。要做「常駐快速篩選」是新功能，得先立案。
3. ⚖️ **計數留在頁面標題列**（dsr-1 裁定的延伸；手機稿補上）。
4. ⚖️ **頂列畫成真實的兩層**（shell＋頁面），與 Flow D 一致；「釘進 shell」另案。
5. ⚖️ **骨架欄數修在共用常數**，順手修掉桌機 `lg`／`xl` 的 reflow（同一個元件、同一行）。
6. ⚖️ **「未比對」二字本張不裁**——稿維持「未比對」、碼維持「未匹配」，e2e 用碼的字。

### 不要做的事

- 不要合併空片庫三態；不要改 `classifyEmptyState()`。
- 不要動 `PosterCardV2`、`FilterChips`、抽屜、`AppShellV2`、`MobileTabBar`。
- 不要在稿上留下「4K」「未看完」「掃描結果」「片名」。
- 不要把「未比對」改成「未匹配」或反過來。
- 不要本機產 `-linux.png`；不要改舊 spec 的斷言。

### 已知陷阱

- **骨架夾具是新的**——第一次拍 12 格 `animate-pulse`（若有）的基準線：視覺 project 有 `reducedMotion: 'reduce'`；仍然連跑三次確認穩定。
- **jsdom 看不到斷點**：`max-sm:` class 斷言只證明 token 在；字級由 e2e 640 那條守。
- **E4-M 在 Flow E 群組不在 Flow A**（`n7jVF` 位於 18510,28389）——匯出的 key 在 `flow-e-scanner/`，不要去 `flow-a-browse-v2/` 找。
- **A1p-M 的 `uqgW0` 是用絕對座標放的**（x=0／y=52），刪掉後 `empty-m` 不會位移（它從 y=52 起算是自己的 padding）；刪完 `ctx.bounds` 核一次。
- **Pencil**：同 `-b`。
- **gh 帳號會被別的 session 切走**：所有 `gh` 指令帶 `GH_TOKEN=…`。
- **行號以建單時為準**（2026-09-22，main `49cc74a3`；`-b` 合併後 `LibraryBrowseV2.tsx` 行號會位移，以符號找）。

### Source tree

```
ux-design.pen、_bmad-output/pen-tokens.json、screenshots/flow-a-browse-v2/{a1p,a2p,a3p}-m.png、flow-e-scanner/e4-m.png   ← Task 1
apps/web/src/components/library/libraryGrid.ts（新，欄數常數）                                                  ← Task 2
apps/web/src/components/library/LibraryStatesV2.tsx（+spec）                                                    ← Task 2/3
apps/web/src/components/library/LibraryBrowseV2.tsx（+spec）                                                    ← Task 2/3
apps/web/src/routes/test/-gallery.fixtures.tsx                                                                  ← Task 4
tests/e2e/library-mobile.spec.ts（追加）                                                                        ← Task 4
tests/visual/…/library-mobile-screens/{skeleton,empty-no-folder}、…/library-grid-skeleton                       ← Task 4
```

### Cross-Stack Split Check

後端 task **0**、前端／設計／測試 task 4 → 不觸發跨棧拆分。規模：一個新常數模組＋兩個既有元件小改、四張稿（其中一張重畫）、三個夾具、四條 e2e——比 `dsr-4b-1` 小；設計稿工作量是這張的大頭。

### Time-dependent visual coverage

- **N/A — no wall-clock-reading components touched.** 骨架與空狀態不讀時間；`RecentlyAdded` 不在範圍。

### References

- [Source: `apps/web/src/components/library/LibraryStatesV2.tsx:23-36, 58, 93`；`LibraryBrowseV2.tsx:1-2, 68-72, 493-533, 513-528, 662-686`；`components/shell/AppShellV2.tsx:84-123`；`components/shell/MobileTabBar.tsx:27`]
- [Source: `apps/web/src/routes/test/-gallery.fixtures.tsx`（`library-empty-*`、`library-library-grid` 條目；`width: 390` 先例）；`tests/e2e/empty-library.spec.ts`；`tests/e2e/downloads-mobile.spec.ts`（範本）]
- [Source: `ux-design.pen` `BfGVZ`／`R5pmG`／`uqgW0`／`qBWQC`／`h1v1U6`／`wW2oF`／`y2MDfn`／`n7jVF`／`vZpT8`（A1p-D）／`EsoIv`（A2p-D）／`uMDjw`（D1-M-v2）／`S86VM`／`hD7Tw`／`n6Crb`／`JUkwx`／`p6EGC` —— SM 建單時以 Pencil MCP 逐節點讀出（2026-09-22）]
- [Source: `DESIGN.md:604-636`（手機規則：標題降一階、44×44、間距一套）]
- [Source: `sprint-status.yaml` → `dsr-1b-flow-a-mobile`（`:1302`）、`dsr-5-flow-e-scanner`（`:1351`，E4-M 耦合）、`disc-2026-09-library-empty-three-states-one-frame`（`:1305`）、`disc-2026-09-library-has-filters-the-product-lacks`（`:1307`）、`disc-2026-09-library-header-not-in-shell-bar`（`:1312`）、`disc-2026-09-unmatched-two-words`（`:1352`）、`disc-2026-09-postercard-v2-badge-shape-drift`（`:1434`）]
- [Source: `dsr-1-flow-a-browse-v2.md` AC #6（A1p-D 重畫、三態禁令）；`dsr-4b-1` 🔴 #12（頁面標題列 vs shell TopAppBar）]
- [Source: project-context.md#Rule 16／#Rule 21／#Rule 24；`.claude/memory/feedback_split_oversized_stories.md`、`feedback_measure_the_breakpoint_you_return_to.md`、`feedback_verify_pen_saved_before_commit.md`、`project_pen_schema_gotchas.md`]

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

- Ultimate context engine analysis completed - comprehensive developer guide created（SM Bob，2026-09-22）

### Discovery Triage

- **建單時的發現（SM Bob 2026-09-22）：**
  - ① 骨架欄數與網格不同（桌機 `lg`／`xl` 每次載入 reflow）→ AC #2 吸收（手機看不到但同一個元件）。
  - ③ A3p-M 的「4K／未看完」快速篩選列 → 由本張刪除，↪ 補記 `disc-2026-09-library-has-filters-the-product-lacks`。
  - ③ 「未比對／未匹配」→ 仍在 `disc-2026-09-unmatched-two-words`，本張不裁。
  - ③ 手機頂列要不要釘進 shell → 仍在 `disc-2026-09-library-header-not-in-shell-bar`（稿改成兩層後該單同時涵蓋桌機與手機）。
- **dev-story 期間的發現：** （dev 填寫）

### File List

## Change Log

- 2026-09-22 — 建單（SM Bob，create-story；main `49cc74a3`）。由 `dsr-1b-flow-a-mobile` 拆出的第三塊；排在 `-b` 之後。四張手機稿與 E4-M 以 Pencil MCP 逐節點讀出（含三處刻意的 partially clipped），`LibraryStatesV2`／`LibraryBrowseV2` 現況由唯讀稽核代理查證（骨架欄數 ≠ 網格欄數為建單時發現）。
