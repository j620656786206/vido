# Story DSR.4b-1：手機上的「下載」頁排序改成從底部滑上來的抽屜，膠囊列變成一行可以橫向捲

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As the person who checks downloads from a phone,
I want 排序是一個用拇指按得到的抽屜（不是系統的小下拉），而且狀態膠囊排成一行可以左右滑,
so that 我不用去戳一個為滑鼠設計的下拉選單，膠囊列也不會佔掉兩行螢幕。

## Context

`dsr-4b`（Flow D 手機抽屜）拆出來的**第一塊：地基＋排序＋膠囊列**。第二塊 `dsr-4b-2-mobile-actions-and-detail-sheets`（卡片動作＋詳細資訊）**依賴本張先合併**（它要用本張做的 `ui/Sheet` 新 prop 與 `useIsPhone`）。

🚨 **進度表原本的轉述是錯的。** 原條目（與 `dsr-4` 故事檔 `:15`）寫「D8-M 篩選／D9-M 排序／D10-M 卡片動作」。稿上實際是：D8-M＝卡片動作、D9-M＝詳細資訊、**D10-M＝排序**；稿上**沒有「篩選抽屜」**——手機的篩選就是那一排可以橫向捲的膠囊。

| 稿 | 節點 | 是什麼 | 程式碼現況 |
| --- | --- | --- | --- |
| D10-M-v2 · 手機排序 sheet | `JxMWL` → `t6GBBy` | 排序抽屜 | 原生 `<select aria-label="排序方式">`（`DownloadsBrowseV2.tsx:331-353`） |
| D1-M-v2 · 手機 | `uMDjw` | 頂列右側一顆 44×44 排序鈕（`arrow-down-up`）；膠囊列**單行橫向捲動** | 沒有排序鈕；膠囊列 `flex flex-wrap`（`:276`），390 寬時換成兩行 |

⚠️ **驗收基準是 `.pen` 節點值**。PNG 只是參考。
⛔ **本張不進**：卡片的 ⋯、動作／詳細資訊抽屜（4b-2）、桌機的任何版面（≥640 一個像素都不能變）、表格、批次選取列、後端、D1-M 頁面其餘的對齊（另立單，AC #8）。

### 🔴 建單時查到的事（main `ae204761`；行號皆為現況）

**現成的抽屜零件——有兩套，選哪一套已經有書面裁定**

1. `ui/mobileSheet.tsx` 的檔頭寫明：`ui/Dialog`＋`MOBILE_SHEET_*` 給「**一個元件、兩種長相**」（桌機置中對話框、手機抽屜）；`ui/Sheet`（Base UI）給「**只存在於手機的面板**」。排序抽屜桌機不存在（桌機是 select）→ **用 `ui/Sheet`**。現有兩個使用者：`shell/MobileMoreSheet.tsx:20`、`library/LibraryFilterSheetV2.tsx:42`。
2. `ui/Sheet.tsx`（50 行）今天的 props 只有 `open`／`onOpenChange`／`title`／`ariaLabel`／`children`：
   - `data-testid="bottom-sheet"` **寫死**（`:31`）；
   - Popup 的 `p-4`＋`pb-[max(1rem,env(safe-area-inset-bottom))]` **寫死**（`:30`）——稿的列是貼邊 8px 的整列按鈕（`Vs994` padding [4,8]），用 `p-4` 會縮成 16；
   - 標題 `Dialog.Title` 的 class 寫死 `mb-3 text-base font-semibold`（`:39`）——Popup 改成 `p-0` 之後標題沒有左右內距，也沒辦法 `line-clamp`／換字級（4b-2 需要）；`Dialog.Title` 不能在 `ui/` 以外 import；
   - Popup 自己就是捲動容器（`overflow-y-auto`）——4b-2 的詳細資訊要「內容捲、底部動作列固定」，得靠 `className` 把它改成 `flex flex-col overflow-hidden`；
   - 沒有轉傳 `finalFocus`／`onOpenChangeComplete`；className 是純字串（沒過 `cn()`）；沒有 spec 檔。
   - 動畫是 Base UI 的 `data-[starting-style]`／`data-[ending-style]`＋`transition-transform`，**不是** `sheet-enter` keyframes（那是 Radix 那一套的）。
   - `@base-ui/react` 在 `components/ui/**` 以外 import 會被 ESLint 擋（`eslint.config.mjs:361-384`）→ 要新能力就**擴充 `ui/Sheet.tsx`**。
3. Base UI 版本 1.5.0（`package.json:96`）已查證：`Dialog.Popup` 的 `finalFocus?: boolean | React.RefObject<HTMLElement | null> | ((closeType) => boolean | HTMLElement | null | void)`（`DialogPopup.d.ts:32`；預設 `true`＝回到觸發元素或先前有焦點的元素）；`initialFocus` 預設「第一個可 tab 的元素」；`Dialog.Root` 有 `onOpenChangeComplete`（`DialogRoot.d.ts:44`）、`modal` 預設 true（鎖焦點、鎖頁面捲動）、點外面預設會關；`Dialog.Title` 會設 `aria-labelledby`。
4. 把手：Flow D 稿是 40×4 `$text-muted`，`ui/Sheet` 畫的是 `w-10 h-1` `--border-subtle`——三種畫法的問題已立 `disc-2026-09-bottomsheet-grabber-three-variants`，**本張不動**。
5. z-index：`ui/Sheet` 是 `z-[70]`／`z-[71]`，`MobileTabBar` 是 `z-40` → 抽屜會蓋住底部分頁列——這是對的（有遮罩、可關閉的 modal；`MobileMoreSheet` 先例；`ScanProgress.tsx:101-111` 的「要讓開分頁列」只適用於非 modal 的常駐面板）。

**排序（D10-M）**

6. 排序狀態是 **`useState`，不在 URL**（`DownloadsBrowseV2.tsx:119-120`；filter／page／pageSize 才在 URL）。本張不改這件事。
7. 程式碼有 **8 個**排序選項（`SORT_OPTIONS` `:62-71`，value 格式 `field:order`）：加入時間（新到舊）／加入時間（舊到新）／名稱（A–Z）／名稱（Z–A）／進度（多到少）／進度（少到多）／狀態／狀態（反向）。`handleSortOption(value: string)` 是既有的處理函式。
8. **稿畫了 4 個，其中兩個系統做不到**：「加入時間（新→舊）」「下載進度」「**下載速度**」「**檔案大小**」。後端支援 `size`／`eta`（`download_service.go:186-195`）但**沒有速度**；前端 `SortField` 沒有 `size`（`downloadService.ts:19`）。→ 改稿：抽屜列出與桌機 select **同一份 8 個選項、同一組文字**（裁定 2）。
9. 稿的未選列標籤在 x=12、已選列在 x=44（已選列有 `check` 圖示、未選列把圖示 `enabled:false`）→ **選了哪一個，那一列的字就往右跳 32px**。稿的疏失：每一列都保留 20px 的圖示位（裁定 2）。另有一顆殘留的停用節點 `P77zR`「· 62.4%」是全檔 69 個 `problems` 之一，刪掉。
10. 五條既有 unit 斷言靠 `getByRole('combobox', { name: '排序方式' })`＋`selectOptions`（`DownloadsBrowseV2.spec.tsx:92, 97, 157, 178-179, 202`）→ **`<select>` 要留在 DOM**，手機只用 CSS 藏（`max-sm:hidden`）。既有 spec 沒有任何按鈕叫「排序」。

**膠囊列與排序鈕的位置（D1-M）**

11. 膠囊列 `:276` `flex flex-wrap gap-2`＋`role="tablist"`；`error` 膠囊在計數為 0 且未選取時不渲染（`:281`）；膠囊 `h-11`（44）——稿是 40，**維持 44**（裁定 3）。
12. 頁面自己的標題列在 `:259-273`：`<header className="flex items-start justify-between gap-4">`，左邊 `<h1>下載</h1>`＋副標，右邊只有選取模式時的「取消」。外面還有 `AppShellV2` 的 56 高頂列——稿的 `TopAppBar` 在真實 app 裡對應的是**頁面標題列**，排序鈕放在它的右邊。`showToolbar`（`:250`＝qBT 可用且不在清單選取模式）與「取消」互斥，是排序鈕正確的條件。
13. 全域焦點框是 `outline: 2px` ＋ `offset 2px`（`styles.css:530-533`）＝膠囊外 4px。`overflow-x: auto` 會連帶讓 `overflow-y` 變 `auto` → **上下的焦點框會被裁掉**，要留 4px。`styles.css` 沒有隱藏捲軸的 utility；先例是 `SettingsLayout.tsx:185` 的 `[scrollbar-width:none] [&::-webkit-scrollbar]:hidden`。

**斷點與 hook**

14. 全 repo 沒有共用的 `useIsPhone`。`DownloadsBrowseV2.tsx:88-103` 有檔內私有的 `useIsDesktop`（1024）；`scanner/ScanProgress.tsx:20` 有另一個（768）。
15. 🚨 **`apps/web/src/test-setup.ts:80-90` 有一個全域的 `matchMedia` stub，對任何 query 都回 `matches: false`。** 所以 hook 必須寫成「直接問 `(max-width: 639.98px)`、直接取 `matches`」——jsdom 預設＝**不是手機**，全部既有 spec 留在桌機路徑。（若寫成 `(min-width: 640px)` 取反，每一支 jsdom 測試都會被丟進手機路徑。）
16. `DownloadsBrowseV2.spec.tsx:226-235` 那一條自己 stub 了一個**對任何 query 都回 true** 的 `matchMedia`（為了表格檢視）。本張的排序鈕與膠囊列是純 CSS、不讀 hook，**所以本張不受影響**；但 4b-2 會——屆時在那支 spec 檔頭 mock hook（寫在 4b-2）。

**測試現況**

17. e2e `tests/e2e/downloads-v2.spec.ts`（8 條）沒有設 viewport；CI 只跑 `--project=chromium --project=webkit-core`（`.github/workflows/test.yml:461`；後者的 `testMatch` 不含下載）。`:304` 表格那一條在本機手機 project **本來就是紅的**（切換鈕 `hidden lg:flex`）。
18. 視覺夾具：`downloads-*` 全是 `width` 框寬。抽屜是 **Portal**（Base UI `Dialog.Portal`）→ `viewport: { width: 390, height: 844 }` 夾具：state div 高度為 0 → 視覺 spec 改拍整個視窗（`components.visual.spec.ts:273-276`），是誠實的 390。`GalleryFixture.penNode` 是**必填**（`-gallery.fixtures.tsx:413`）。⚠️ 既有兩個 `ui/Sheet` 使用者都沒有夾具——**這會是第一次拍 Base UI 的抽屜**。
19. Rule 21：lint 只掃 `components/**`（`eslint.config.mjs:252`）；新元件檔頭一行 `// Design ref: ux-design.pen Screen X (id)` 就滿足；`hooks/` 不在範圍。

### 設計稿節點

| 代號 | 節點 | 本張要看的 |
| --- | --- | --- |
| D10-M-v2 | `JxMWL`（18510,24325，390×844）→ `t6GBBy` actionSheet（y=504、390×340、padding [8,0,20,0]、`$bg-secondary`、頂角 `$radius-xl`、`$border-subtle`） | 把手 `G2M3X`／`fcaAf`；`NPwYa` sheetHeader（[4,20,12,20]、gap 4）：`lyZpl`「排序」BodyLg 600、`vh7za`「選擇下載清單的排序方式」Body `$text-secondary`、`P77zR` **殘留要刪**；`hReu0` 分隔線；`Vs994` actions（[4,8]、gap 2）：每列 h52、padding [0,12]、gap 12、`$radius-md`；已選 `H0hCBY`＝`$accent-subtle` 底＋`check` 20 `$accent-text`＋BodyLg 600 `$accent-text`；未選＝透明底、BodyLg 500 `$text-primary`、圖示 `enabled:false`（**要改**） |
| D1-M-v2 | `uMDjw`（17040,24325） | `Q1EEk` 排序鈕 44×44 `$bg-tertiary` `$radius-md`、圖示 `h13W7n` `arrow-down-up` 20 `$text-secondary`；`eqXdV` ChipRow 單行（[12,16]、gap 8；第五顆 `LprU8` 出血＝橫向捲動的示意，是 69 個 `problems` 裡刻意的那種）；`lZZd6` swipeHint **要改** |
| Flow D | `H3Nxng`；說明 `ggTb3`（17040,22467）；手機群組 `bblkD`（17040,24325，1860×844） | 規格註記放 `bblkD` 下方 |

字階：BodyLg 16、Body 14、Label 12。間距：`2xs`2・`xs`4・`sm`8・`md`12・`lg`16・`lg-plus`20。全檔 `problems` 現況 **69**。

---

## Acceptance Criteria

1. **設計稿先改（Pencil MCP；每個節點動手前再 `Get` 一次）。**
   - **D10-M** `JxMWL`：`Vs994` 改成**八列**，文字逐字取自 `SORT_OPTIONS`（🔴 #7）；第一列維持已選樣式。**每一列都保留 20px 圖示位**：未選列的圖示 `enabled:true`、`icon:"check"`、`fill:"#00000000"`（八列的標籤左緣一致）。列的節點名稱改成 `opt-*`（現在還叫 `act-暫停` 之類）。刪掉殘留的 `P77zR`。⛔ 不要寫死抽屜高度：改完用 `ctx.bounds` 讀 `t6GBBy` 的實高（或設成包住內容），再 `y = 844 − 高度` 重新貼底；高度必須 ≤ 675（844×0.8）。📎 `Copy` 一般節點時 `descendants` 用名稱當 key 會被**靜默忽略**——`Copy` 後 `Get` 出新 id 再逐一 `Update`。
   - **D1-M** `uMDjw`：`lZZd6` →「點 ⋯ 開啟動作」（拿掉「左滑可快速暫停/繼續」——沒有這個手勢，稿不能答應做不到的事；裁定 4）。
   - 規格註記：在 `bblkD` 下方放一則 text（樣式比照 Flow F 的 `oVRQd`：Label、`$text-muted`、`textGrowth:"fixed-width"`、寬 300；**位置先用 `FindEmptySpace`／`ctx.bounds` 確認不壓到下一個 Flow 的標題**），名稱 `spec-note-dsr-4b-1`：
     > 「D10／D1（手機）：排序是只存在於手機的底部抽屜（<640），桌機維持排序下拉。入口是標題列右邊的 44×44 排序鈕；選取模式與 qBittorrent 連不上時沒有排序鈕。抽屜與桌機下拉是同一份八個選項、同一組文字，按了就套用並關閉；每一列都留圖示位，選了哪一列字都不會跳。抽屜蓋住底部分頁列（有遮罩，點遮罩或 Esc 關閉），關掉後焦點回到排序鈕。膠囊列在手機單行橫向捲動，膠囊維持 44 高（稿畫 40）。」
   - 收尾：全檔 `problems` **不得超過 69，理想 68**（`P77zR` 消失）；存檔走選單 Save（`osascript -e 'tell application "Pen" to activate' -e 'delay 1' -e 'tell application "System Events" to tell process "Pen" to click menu item "Save" of menu "File" of menu bar 1'`），`git status --porcelain ux-design.pen` 必須出現 ` M`，並 grep 磁碟檔確認 `spec-note-dsr-4b-1` 真的在裡面；**存檔後**才跑 `python3 scripts/export-pen-screenshots.py`；**只 stage** `flow-d-downloads-v2/d1-m-v2.png`、`d10-m-v2.png` 與 `_bmad-output/pen-tokens.json`，其餘 `git checkout --` 還原。`SCREENS` 已有這兩張。
   - 📎 Pencil `execute` 失敗會 rollback 同一次呼叫的編輯；`Replace` 會重設沒寫到的屬性；讀值用 `Print(...)`。

2. **地基：`ui/Sheet` 補五個 prop、一個共用的 `useIsPhone`。** 兩者 4b-2 都要用——API 在本張定案。
   - `ui/Sheet.tsx` 新增 props，**預設值讓既有兩個使用者逐位元不變**（已查證：`cn(base, undefined) === base`；沒有任何 spec／視覺基準斷言 Popup 的 class 字串）：
     - `testId?: string`（預設 `'bottom-sheet'`）；
     - `className?: string`——`cn(基底, className)` 併到 Popup。📎 已用 twMerge 實測：基底＋`p-0` 會**同時**丟掉 `p-4` 與 `pb-[max(1rem,env(safe-area-inset-bottom))]`（同一個衝突群組）——所以歸零內距的呼叫端必須自己把 safe-area 加回去（AC #3 的 className）；`overflow-hidden` 會取代 `overflow-y-auto`；
     - `titleClassName?: string`——`cn('mb-3 text-base font-semibold text-[var(--text-primary)]', titleClassName)`；
     - `finalFocus?: React.ComponentProps<typeof Dialog.Popup>['finalFocus']`（ref、函式、boolean 三種都要能傳——4b-2 用函式版）；
     - `onOpenChangeComplete?: (open: boolean) => void` → 轉傳給 `Dialog.Root`。
   - 新檔 `ui/Sheet.spec.tsx`（今天沒有）：（紅）五個新 prop 各一條——`testId` 換掉 testid、`className` 併入且 `p-0` 真的蓋掉 `p-4`（斷言 class 字串裡**沒有** `p-4`）、`titleClassName` 併入、`finalFocus` 給 ref 時關閉後焦點落在該元素、`onOpenChangeComplete` 被呼叫；（守）不給新 prop 時 testid 仍是 `bottom-sheet`、沒有 `title` 時有 sr-only 標題「選單」。
   - 新檔 `apps/web/src/hooks/useIsPhone.ts`：`useSyncExternalStore`；查詢字串 **`(max-width: 639.98px)`，直接取 `matches`（不取反）**（🔴 #15）；`typeof window.matchMedia !== 'function'` → `false`；server snapshot `false`。`hooks/` 不在 Rule 21 的 lint 範圍——檔頭寫一般 JSDoc，說明「640＝`sm`，與 `MobileTabBar` 的 `sm:hidden` 同一個數字」。附 `useIsPhone.spec.ts`：`matches:true`→true、`matches:false`→false、沒有 `matchMedia`→false、`change` 事件會更新、卸載會 `removeEventListener`。
   - 📎 本張自己**不呼叫** `useIsPhone`（排序鈕與膠囊列都是純 CSS）；它在這裡是因為地基要一次定案、一次測好。
   - ⛔ 不要順手把 `DownloadsBrowseV2` 的 `useIsDesktop` 或 `ScanProgress` 的那一個合併進來（另一個斷點、另一張單）。

3. **排序抽屜（D10-M）。**
   - 新檔 `downloads/DownloadSortSheet.tsx`（檔頭 `// Design ref: ux-design.pen Screen D10-M-v2 (JxMWL)`）。Props：`{ open, onOpenChange, finalFocus, options, value, onChange }`——`options` 就是 `SORT_OPTIONS`（由 `DownloadsBrowseV2` 傳入，⛔ 不准在抽屜裡再列一份）；`value` 是 `field:order` 字串。
   - `ui/Sheet`：`testId="download-sort-sheet"`、`title="排序"`、`titleClassName="mb-1 px-5"`、`className="p-0 pt-2 pb-[max(1.25rem,env(safe-area-inset-bottom))]"`（📎 已實測 twMerge 後就是這一串）。標題下：副標「選擇下載清單的排序方式」（`px-5 pb-3 text-sm text-[var(--text-secondary)]`）、分隔線、選項區（`px-2 py-1`、列距 2px）。
   - 八列。語意用 **`role="radiogroup"`（`aria-label="排序方式"`）＋每列 `role="radio"`＋`aria-checked`**，每列是一顆 `<button type="button">`：`min-h-[52px]`、`px-3`、`gap-3`、`rounded-[var(--radius-md)]`、`text-base`；已選列 `bg-[var(--accent-subtle)] font-semibold text-[var(--accent-text)]`＋`Check`（`size-5`、`aria-hidden`）；未選列 `font-medium text-[var(--text-primary)]`＋同一個 20px 位置放 `aria-hidden` 的空 `<span className="size-5 shrink-0">`。
   - 鍵盤：roving tabindex（已選的那一列 `tabIndex=0`，其餘 `-1`——Base UI 的 `initialFocus` 預設落在第一個可 tab 的元素＝已選列，正好）；上下鍵在列之間移動焦點（頭尾循環）、Enter／Space 選取。
   - 選取 → `onChange(value)`＋`onOpenChange(false)`。選到目前已選的那一列也一樣關閉。

4. **排序鈕＋原生 select 在手機藏起來＋膠囊列單行捲動（🔴 #10–#13）。**
   - 標題列 `:259-273`：在 `<header>` 的右側加排序鈕——`<button type="button" ref={sortBtnRef} aria-label="排序" aria-haspopup="dialog" data-testid="downloads-sort-btn" className="flex size-11 shrink-0 items-center justify-center rounded-[var(--radius-md)] bg-[var(--bg-tertiary)] text-[var(--text-secondary)] sm:hidden">`＋`ArrowDownUp`（`size-5`、`aria-hidden`）。**只在 `showToolbar` 為真時畫**（與「取消」互斥）。
   - `const [sortSheetOpen, setSortSheetOpen] = useState(false)`；渲染 `<DownloadSortSheet finalFocus={sortBtnRef} options={SORT_OPTIONS} value={`${sortField}:${sortOrder}`} onChange={handleSortOption} …>`——`handleSortOption` 是**既有的同一個函式**。`showToolbar` 變 false 時把 `sortSheetOpen` 歸 false。
   - 原生 `<select>` 的外層 `<label>`（`:331`）加 `max-sm:hidden`——**留在 DOM**。
   - 膠囊列 `:276`：加 `max-sm:-mx-4 max-sm:flex-nowrap max-sm:overflow-x-auto max-sm:px-4 max-sm:-my-1 max-sm:py-1 [scrollbar-width:none] [&::-webkit-scrollbar]:hidden`（`-my-1 py-1`＝替焦點框留 4px，🔴 #13；隱藏捲軸兩個 class 不帶 `max-sm:` 也無害——桌機那一列不捲）；每顆膠囊加 `max-sm:shrink-0`。`role="tablist"`、`aria-controls`、`error` 膠囊的顯示條件**不動**。
   - Rule 21：`DownloadsBrowseV2.tsx:1-2` 的標頭補 ` · D1-M-v2 (uMDjw)`（照該檔既有的寫法）。
   - ⛔ 不把排序放進 URL；不新增「檔案大小」「下載速度」排序。

5. **既有的行為不准回歸。**
   - 桌機（≥640）：`downloads-*` 所有既有視覺基準線**零變動**；`DownloadsBrowseV2.spec.tsx`（12 條）既有斷言**一條都不改寫**（`combobox`、`selectOptions`、`tab` 全部照舊）。
   - `tests/e2e/downloads-v2.spec.ts` 在 `chromium` 全綠、斷言不改。只准：替 `:304`（表格）加 `test.skip((page.viewportSize()?.width ?? 1280) < 1024, 'table toggle is lg-only')`（寫在那一條**裡面**；⛔ 不要寫成 chromium-only，會誤傷 firefox）。先在 main 的版本上用 `mobile-chrome` 跑一次確認它本來就紅（Rule 24 ①）。
   - `MobileMoreSheet`、`LibraryFilterSheetV2` 的行為與長相不變（`ui/Sheet` 的新 prop 都有預設值）；這兩個元件的既有 spec 全綠。
   - ⛔ 不改 `DownloadRowActions.tsx`、`DownloadCardV2.tsx`、`DownloadsTableV2.tsx`、後端、`routes/downloads.tsx`、`ui/Dialog.tsx`、`ui/mobileSheet.tsx`。

6. **測試。** 分「**紅**」與「**守**」，誠實標示（Rule 16）。jsdom 不看 media query——class 斷言只證明 token 在，版面由 AC #7 守。
   - `useIsPhone.spec.ts`、`ui/Sheet.spec.tsx`（AC #2）。
   - `DownloadSortSheet.spec.tsx`：（紅）`radiogroup`（名稱「排序方式」）＋8 個 `radio`、`aria-checked="true"` 恰好一個且是傳入的 `value`；點「名稱（A–Z）」→ `onChange('name:asc')`＋`onOpenChange(false)`；點已選列 → 也關閉；ArrowDown／ArrowUp 移動焦點且頭尾循環、Enter 選取；未選列有 `size-5` 的佔位、已選列有圖示（兩者都不是同一個元素——量的是「每一列都有一個 20px 位」）。
   - `DownloadsBrowseV2.spec.tsx`：（紅）`downloads-sort-btn` 存在、名稱「排序」、帶 `sm:hidden`；選取模式時沒有它；qBT 錯誤時沒有它；`<select>` 的外層帶 `max-sm:hidden` 且 `combobox` 仍在；按排序鈕 → `download-sort-sheet` 出現；在抽屜選「名稱（A–Z）」→ `useDownloads` 以 `('all','name','asc',1,100)` 被呼叫（與 `:178-179` 同一個斷言形狀）且抽屜關閉；膠囊列帶 `max-sm:overflow-x-auto`、膠囊帶 `max-sm:shrink-0`。（守）既有 12 條。
   - **每一項修法做 mutation check**（拿掉 → 必須有測試變紅），結果寫進 Completion Notes。
   - ⚠️ 不要寫不可能失敗的斷言。膠囊列是 `nowrap`＋`overflow-x-auto`：要量的是「所有膠囊同一條水平線」與「容器 `scrollWidth > clientWidth`」，不是「沒有橫向溢出」。

7. **真瀏覽器的驗證。**
   - **視覺夾具**：`downloads-mobile-sheets/sort`——`component: DownloadSortSheet`、`open: true`、固定的 `options`／`value`、`viewport: { width: 390, height: 844 }`（不可同時給 `width`）、`penNode: 'JxMWL'`（必填）、`statesOnly: ['default']`。只產 darwin；`-linux` 由 `gh workflow run "Visual Regression" --ref <branch>` 開 bootstrap PR 補。
   - **e2e**（新，`tests/e2e/downloads-mobile.spec.ts`，`@e2e @downloads-mobile`）：stub 照 `downloads-v2.spec.ts` 既有的寫法（可把共用的搬到 `tests/support/helpers/downloads-stubs.ts`，**不准改舊 spec 的行為與斷言**）；`import { test, expect } from '../support/fixtures'`，型別從 `@playwright/test`；`beforeEach` 非 `chromium` 就 skip（像素斷言）；**每一條自己 `page.setViewportSize(...)`**（`chromium` project 預設是 1280）；box 一律 `Math.round`；量之前等轉場結束（`Promise.allSettled(el.getAnimations({ subtree: true }).map(a => a.finished))`——Base UI 的抽屜是 transition，`getAnimations()` 一樣抓得到）；≥640 有側欄，水平位置量相對於 `downloads-browse-v2` 根節點。
     - **390・膠囊列**：所有膠囊垂直中心同一條線（差 <2px）；每顆高 ≥44；容器 `scrollWidth > clientWidth`（資料要讓 `error` 膠囊也出現，六顆才會超過 358）；整頁 `document.documentElement.scrollWidth <= 390`；Tab 到第一顆膠囊時焦點框沒被裁（量容器的 `clientHeight ≥ 膠囊高 + 8`）。
     - **390・排序**：`downloads-sort-btn` 可見、≥44×44、右緣 ≈ 根節點右緣 −16；原生 select **不可見**；按下 → `download-sort-sheet` 貼底（相對視窗 `x`＝0、寬 390、底邊＝844）、高度 ≤ 844×0.85；八個 `radio`，八個標籤文字的**左緣相等**（差 ≤1px——🔴 #9 的那個跳動；量文字節點不是量按鈕）；每列高 ≥52；`mobile-tab-bar` 被蓋住（`elementFromPoint` 在分頁列的位置拿到的不是分頁列）；選「名稱（A–Z）」→ 抽屜關閉、發出的清單請求 URL 同時 `includes('sort=name')` 與 `includes('order=asc')`、焦點回到排序鈕；再開一次 → 已選的是「名稱（A–Z）」；Esc 關閉 → 焦點回排序鈕。
     - **640（斷點另一側）**：`downloads-sort-btn` 不可見、原生 select 可見且能用；膠囊列 computed `flex-wrap: wrap`。
     - 本機跑要 `AI_PROVIDER=claude npx playwright test tests/e2e/downloads-mobile.spec.ts --project=chromium`；過了再 `--repeat-each=3`。
   - dev-story Step 9：`d10-m-v2`／`d1-m-v2` 對新基準線與 e2e 量測逐項核（整頁截圖只在 `DSR_SHOTS=1` 時產生、寫到 `testInfo.outputPath()`——6f-4 CR M2）。

8. **另立的單子（建單時已寫入 sprint-status）**：`disc-2026-09-d1-m-downloads-page-not-aligned`、`disc-2026-09-downloads-sort-by-size-and-eta`；補記 `disc-2026-09-bottomsheet-grabber-three-variants`。📎 Base UI 建議 modal popup 裡放一顆 `<Dialog.Close>` 給觸控螢幕報讀（`DialogRoot.d.ts:32`），`ui/Sheet` 沒有 → `disc-2026-09-ui-sheet-no-close-button`（影響全部 `ui/Sheet` 使用者，不在本張修）。

9. **CI 全綠**：`pnpm run format:check`、`pnpm run lint:all`、`pnpm nx run web:typecheck --skip-nx-cache`、`python3 scripts/check-design-tokens.py`、`pnpm nx test web`、`pnpm nx test api`。⛔ 絕不 `run_in_background` 跑測試；Nx 一律 `NX_DAEMON=false`；跑完 api 測試把 `apps/api/coverage/` 刪掉。🚨 合併之後要看 `main` 那一次的 Tests／Docker／Visual Regression 三條。gh 操作一律帶 `GH_TOKEN=$(gh auth token --user j620656786206)`（2026-09-21 同一個 session 內 active 帳號被切走兩次，背景的自動合併因此靜默失敗）。

## Tasks / Subtasks

- [x] **Task 1 — 設計稿：D10-M 八個選項＋圖示位＋刪殘留節點、D1-M 提示、規格註記（AC: #1）**
- [x] **Task 2 — 地基：`ui/Sheet` 五個 prop＋spec、`useIsPhone`＋spec（AC: #2, #6）**
  - [x] 先跑 `MobileMoreSheet`／`LibraryFilterSheetV2` 既有 spec 確認綠 → 改 `ui/Sheet` → 仍綠
- [x] **Task 3 — `DownloadSortSheet`（AC: #3, #6）**
- [x] **Task 4 — 排序鈕、select 在手機藏起來、膠囊列單行捲動（AC: #4, #6）**
- [x] **Task 5 — 視覺夾具、手機 e2e、舊 e2e 表格那一條的 viewport skip、mutation check、收尾（AC: #5, #7, #8, #9）**
  - [x] dev-story Step 9：`d10-m-v2`／`d1-m-v2`

### Review Follow-ups (AI)

<!-- /ship 對抗式 CR（2026-09-22，獨立 context 的審查代理）— Rule 24 ① 吸收的項目，每項都有測試＋mutation。 -->

- [x] [AI-Review][MEDIUM] 抽屜開著時轉橫向（跨過 `sm`）不會關，關掉後焦點落在隱藏的排序鈕＝`<body>` → render 階段偵測 `useIsPhone` 由 true 變 false 就關；`finalFocus` 改函式：排序鈕 → select → `<h1>`
- [x] [AI-Review][MEDIUM] e2e 量抽屜位置有競態（`data-starting-style` 下一幀才拿掉）→ 先等屬性消失再等動畫
- [x] [AI-Review][MEDIUM] `useIsPhone` 用 px，Tailwind `sm` 是 `40rem` → 改 `(width < 40rem)`（與 `max-sm:` 同一句）
- [x] [AI-Review][MEDIUM] 膠囊列 e2e 擋不住 `overflow-hidden`／拿掉出血／下緣焦點空間 → 改量 `overflow-x: auto`＋`scrollLeft` 真的會動、出血 x、Tab 到第一顆後四邊 ≥4px
- [x] [AI-Review][LOW] `Math.round` 不會消掉 -0 → `Math.abs(x) < 0.5`
- [x] [AI-Review][LOW] `value` 不在選項裡時八列都是 `tabIndex=-1` → 第一列補位
- [x] [AI-Review][LOW] qBT 斷線自動關抽屜時焦點可能落到舊元素 → 改在 render 階段關、後備到 `<h1 tabIndex={-1}>`
- [x] [AI-Review][LOW] 副標不是 dialog 的描述、排序鈕沒有 `aria-expanded` → `ui/Sheet` 加 `description`／`descriptionClassName`（`Dialog.Description`）、排序鈕補 `aria-expanded`
- [x] [AI-Review][LOW] 夾具自己抄一份選項 → `SORT_OPTIONS` 加 `export`（內容不變），夾具 import；另加「抽屜與 select 的選項逐字相同」的測試
- [x] [AI-Review][LOW] stub 共用檔註解錯（兩個 route 其實不競爭）→ 改正、補回 `queued` 契約註解
- [x] [AI-Review][LOW] 「高度 ≤ 85%」不可能紅 → 改 ≤ 80%（DESIGN.md）
- [x] [AI-Review][LOW] e2e 分不出 `finalFocus` 有沒有接、點遮罩沒測 → 補「合成點擊（不移焦點，模擬 Safari）」與點遮罩兩條
- [x] [AI-Review][LOW] 既有使用者「逐位元不變」只驗 4 個 token → 改成比對完整 class 字串
- [x] [AI-Review][MEDIUM，推測→確認] 單行膠囊列讓 `?filter=seeding` 的選中膠囊一開始在畫面外（本張造成的退化）→ 選中膠囊捲進列內；另加 `scroll-px-4`、`overscroll-x-contain`

## Dev Notes

### 這張的重點

- **桌機一個像素、一個 role 都不變。** `<select>` 留在 DOM、只用 `max-sm:hidden` 藏；排序鈕 `sm:hidden`。本張不需要任何 JS 斷點判斷。
- **`ui/Sheet`，不是 `ui/mobileSheet`。** `mobileSheet.tsx` 的檔頭就是裁定書。不要把 Flow F 那一套（`MOBILE_SHEET_CONTENT`、`sheet-enter`）搬過來。
- **`ui/Sheet` 的 API 在這張定案，4b-2 直接用。** 五個 prop 一個都不能少（`titleClassName`、函式版 `finalFocus`、`onOpenChangeComplete` 是 4b-2 要的）。
- **稿有兩處是錯的**（做不到的兩個排序、選了會跳的圖示位）。動手前看節點值與程式碼。

### 上游契約（Rule 20 ack）

- 本張不定義也不改任何線上契約。消費的是既有的 `GET /downloads?sort=&order=`（`dsr-4`／`bugfix-f` 的未 stamp AC＝implicit v0）。e2e 的假資料沿用 `downloads-v2.spec.ts` 既有的形狀。

### 建單裁定（2026-09-22，Sally／Alexyu 可在 review 推翻）

1. ⚖️ **拆成兩張**（SM，依 `feedback_split_oversized_stories`：規模超過上一張同類單子就主動拆、切分線選版面）。原本一張的規模＝4 個新元件＋hook＋`ui/Sheet` 擴充＋一個跨元件的狀態機＋一次確認框重構，明顯大於 `dsr-6f-3`。切分線：**標題列／工具列**（本張：排序＋膠囊列＋地基）vs **卡片**（4b-2：動作＋詳細資訊）——本張完全不碰 `DownloadRowActions`，兩張的檔案幾乎不重疊，各自都能獨立出貨。⚖️ Alexyu 2026-09-22 裁定的「詳細資訊一起做」落在 4b-2。
2. ⚖️ **排序抽屜＝桌機下拉的同一份八個選項、同一組文字**（改稿）；每一列保留圖示位（改稿）。稿的「下載速度」後端沒有、「檔案大小」前端沒有 → `disc-2026-09-downloads-sort-by-size-and-eta`。
3. ⚖️ **膠囊維持 44 高**（稿 40）。DESIGN.md 觸控目標 44；程式碼已經做到的不為了稿退回去。
4. ⚖️ **不做「左滑暫停／繼續」**，稿上的提示改成只講 ⋯。手勢是另一個功能，併入 `disc-2026-09-d1-m-downloads-page-not-aligned`。
5. ⚖️ **排序不進 URL**（維持現況）。
6. ⚖️ **用 `radiogroup`，不是 `menu`／`listbox`。** 八選一、選了就生效——單選的語意最貼近；鍵盤行為也最不意外。

### 不要做的事

- 不要用 `MOBILE_SHEET_*`／`ui/Dialog` 做抽屜；不要在 `components/ui/` 以外 import `@base-ui/react`。
- 不要讓 `<select>` 離開 DOM；不要改 `SORT_OPTIONS`、`handleSortOption`；不要在抽屜裡再列一份選項。
- 不要碰 `DownloadRowActions`、卡片、表格、批次選取列（4b-2 或不在範圍）。
- 不要動把手的畫法、`ui/mobileSheet.tsx`、`ui/Dialog.tsx`、後端、`validateSearch`。
- 不要在元件測試裡 stub `matchMedia`。
- 不要本機產 `-linux.png`；不要改舊 e2e／spec 的斷言（舊 e2e 只准加一個 viewport skip）。

### 已知陷阱

- **第一次替 Base UI 抽屜拍基準線**：進場是 `transition-transform`＋`data-[starting-style]`。視覺 project 有 `reducedMotion: 'reduce'`，而 `styles.css` 頂端把 reduced-motion 的時長收成 ~1ms——理論上瞬間到位；**仍然要連跑三次確認基準線穩定**（半途的抽屜＝每次差幾個像素）。不穩就看 `components.visual.spec.ts` 怎麼等 Radix 對話框的，照樣等。
- **`p-0` 會連 safe-area 的 `pb-[…]` 一起丟掉**（同一個 twMerge 群組）——AC #3 的 className 已經把它加回去；4b-2 照抄。
- **`overflow-x: auto` 會連帶裁掉上下的焦點框**——`-my-1 py-1`。
- **`test-setup.ts` 的全域 `matchMedia` stub 回 false**——hook 的寫法由它決定（🔴 #15）。
- **jsdom 看不到斷點**：class 斷言不代表版面；行為由 e2e 守。
- **e2e**：`chromium` project 預設 1280——每條自己設 viewport；`expect(box.x).toBe(0)` 遇到 `-0` 會紅 → `Math.round`；本機要 `AI_PROVIDER=claude`。
- **Pencil**：失敗會 rollback；`Copy` 的 `descendants` 名稱 key 對一般節點無效；存檔走選單 Save＋驗磁碟內容；匯出全量 re-render 近 200 張，只 stage 兩張。
- **gh 帳號會被別的 session 切走**：所有 `gh` 指令帶 `GH_TOKEN=…`。
- **行號以建單時為準**（2026-09-22，main `ae204761`）。

### Source tree

```
ux-design.pen、_bmad-output/pen-tokens.json、screenshots/flow-d-downloads-v2/{d1,d10}-m-v2.png   ← Task 1
apps/web/src/components/ui/Sheet.tsx（+ 新 Sheet.spec.tsx）                                    ← Task 2
apps/web/src/hooks/useIsPhone.ts（新，+spec）                                                   ← Task 2
apps/web/src/components/downloads/DownloadSortSheet.tsx（新，+spec）                             ← Task 3
apps/web/src/components/downloads/DownloadsBrowseV2.tsx(+spec)                                  ← Task 4
apps/web/src/routes/test/-gallery.fixtures.tsx                                                  ← Task 5
tests/e2e/downloads-mobile.spec.ts（新）、tests/support/helpers/downloads-stubs.ts（若抽共用 stub）、tests/e2e/downloads-v2.spec.ts（只准一個 skip）  ← Task 5
tests/visual/…/downloads-mobile-sheets/sort                                                     ← Task 5
```

### Cross-Stack Split Check

後端 task **0 個**，前端／設計／測試 task 5 個 → 不觸發跨棧拆分。規模：1 個新元件＋1 個 hook＋`ui/Sheet` 擴充＋1 個既有元件、一張稿兩個畫面、一個夾具、一支 e2e——與 `dsr-6f-4` 同級。

### Time-dependent visual coverage

- **N/A — no wall-clock-reading components touched.** `components/downloads/` grep `Date.now|new Date|performance.now` 零命中；本張新增的元件不碰時間。

### References

- [Source: `apps/web/src/components/downloads/DownloadsBrowseV2.tsx:1-2, 46-76, 88-124, 244-273, 275-302, 304-353`；`DownloadsBrowseV2.spec.tsx:92-97, 121, 157, 178-179, 184, 202, 220-246`]
- [Source: `apps/web/src/components/ui/Sheet.tsx`（全檔）；`ui/mobileSheet.tsx:1-16`（兩套零件的分工）；`shell/MobileMoreSheet.tsx:9-20`、`library/LibraryFilterSheetV2.tsx:1-42`；`shell/MobileTabBar.tsx:27`；`scanner/ScanProgress.tsx:101-114`；`settings/SettingsLayout.tsx:185`（隱藏捲軸的先例）；`eslint.config.mjs:252, 361-384`；`apps/web/src/test-setup.ts:80-90`；`apps/web/src/styles.css:530-533`]
- [Source: `node_modules/@base-ui/react`（1.5.0）`dialog/popup/DialogPopup.d.ts:27-32`、`dialog/root/DialogRoot.d.ts:32-44`]
- [Source: `apps/web/src/services/downloadService.ts:19, 172`；`apps/api/internal/services/download_service.go:186-195`；`tests/e2e/downloads-v2.spec.ts:304-352`；`.github/workflows/test.yml:461`；`playwright.config.ts:105-182`；`apps/web/src/routes/test/-gallery.fixtures.tsx:396-445`；`tests/visual/components.visual.spec.ts:273-276`；`routes/test/gallery-fixture-viewport.spec.ts`]
- [Source: `ux-design.pen` `JxMWL`／`t6GBBy`／`NPwYa`／`P77zR`／`Vs994`／`H0hCBY`／`uMDjw`／`Q1EEk`／`eqXdV`／`lZZd6`／`ggTb3`／`bblkD` —— SM 建單時以 Pencil MCP 逐節點讀出（2026-09-22）；程式碼現況由唯讀稽核代理查證，建單後再經對抗驗證]
- [Source: `DESIGN.md:351-360`（字階）、`:604-636`（手機規則、44×44、Sheet 80%）、`:836`（`problems`；現況 69）]
- [Source: `dsr-4-flow-d-downloads-v2.md`（`:15` 的錯誤轉述）；`dsr-6f-1`…`dsr-6f-4`（手機 e2e 的量法、`viewport` 夾具、`Math.round`、project skip、≥640 側欄、`DSR_SHOTS`）]
- [Source: project-context.md#Rule 16／#Rule 20／#Rule 21／#Rule 23／#Rule 24；`.claude/memory/feedback_split_oversized_stories.md`、`feedback_measure_the_breakpoint_you_return_to.md`、`feedback_verify_pen_saved_before_commit.md`、`project_pen_schema_gotchas.md`、`feedback_gh_token_explicit_for_pr_ops.md`]

## Dev Agent Record

### Agent Model Used

Claude Opus 5 (1M context) — `claude-opus-5[1m]`（dev-story, Amelia，2026-09-22）

### Debug Log References

- Pencil：`problems` 69 → **68**（`P77zR` 刪除）；`t6GBBy` 改 `fit_content` 後實高 **560**（≤675），`y = 844 − 560 = 284`；八列標籤 x 全部 = 44。存檔走選單 Save，`git status` ` M ux-design.pen`，磁碟檔 grep 到 `spec-note-dsr-4b-1`、`opt-進度（多到少）`、`點 ⋯ 開啟動作"`，`P77zR` 0 次；`pen-tokens.json` 的 `penSha256` = 磁碟檔 `shasum -a 256`。
- 匯出 196/196；16 個 PNG 變動裡只留 `d1-m-v2.png`／`d10-m-v2.png`，其餘 `git checkout --`。
- 本機 e2e 要 `AI_PROVIDER=claude`（`preexisting-fail-e2e-local-ai-provider`）。

### Completion Notes List

- Ultimate context engine analysis completed - comprehensive developer guide created（SM Bob，2026-09-22）
- **做了什麼（dev-story，2026-09-22）**
  - **稿（Task 1）**：D10-M `Vs994` 八列（文字逐字取自 `SORT_OPTIONS`，節點名 `opt-*`；前三列沿用、後四列 `Copy` 後逐一 `Update`），未選列圖示 `enabled:true`／`check`／`#00000000`；刪 `P77zR`；抽屜 `fit_content` 後重新貼底。D1-M `lZZd6` →「點 ⋯ 開啟動作」。規格註記 `spec-note-dsr-4b-1`（`WRKrD`，Flow D 群組、`bblkD` 下方 60px、300×144，下一個 Flow 標題在 27169，不重疊）。
  - **地基（Task 2）**：`ui/Sheet` 五個新 prop（CR 後再加 `description`／`descriptionClassName`，共七個；`testId` 預設 `bottom-sheet`、`className`／`titleClassName` 走 `cn()`、`finalFocus`（型別直接取 `Dialog.Popup` 的）、`onOpenChangeComplete`）。已查證 Base UI 1.5.0 `FloatingFocusManager` 以 `returnFocus = true` 解構預設，`finalFocus={undefined}` 等於沒傳——既有兩個使用者逐位元不變。`hooks/useIsPhone.ts`：`useSyncExternalStore`＋`(max-width: 639.98px)` 直接取 `matches`。
  - **排序抽屜（Task 3）**：`DownloadSortSheet`——`radiogroup`＋八個 `role="radio"` 按鈕、roving tabindex、上下鍵循環、Enter／Space 是按鈕本身的 click；選了就 `onChange`＋關閉。⚠️ 與故事字面的一處差異：上下鍵處理掛在**每一列**而不是 `radiogroup` 容器上——掛在容器上 `jsx-a11y/interactive-supports-focus` 報 error（有互動 role 的容器必須可聚焦），行為完全相同。
  - **頁面（Task 4）**：標題列右側 `downloads-sort-btn`（只在 `showToolbar` 時畫、`sm:hidden`）；`DownloadSortSheet` 用既有的 `handleSortOption`；`showToolbar` 變 false 時把抽屜關掉（CR 後改成 render 階段，並加上「轉橫向跨過 `sm` 也關」）；原生 select 的 `<label>` 加 `max-sm:hidden`（留在 DOM）；膠囊列加 `max-sm:` 單行捲動＋`-my-1 py-1`，膠囊 `max-sm:shrink-0`；檔頭補 ` · D1-M-v2 (uMDjw)`。
  - **驗證（Task 5）**：夾具 `downloads-mobile-sheets/sort`（390×844、`penNode: 'JxMWL'`）；新 e2e `tests/e2e/downloads-mobile.spec.ts`（3 條）＋共用 stub `tests/support/helpers/downloads-stubs.ts`（只給新 spec 用，舊 spec 一行都沒搬）；舊 e2e 表格那一條加 viewport<1024 skip（寫在該條裡面）。
- **測試（Rule 16：紅／守）**
  - 紅：`ui/Sheet.spec.tsx` 5 條（新 prop 各一）；`useIsPhone.spec.ts` 7 條（新檔，實作前 import 失敗）；`DownloadSortSheet.spec.tsx` 9 條（新檔）；`DownloadsBrowseV2.spec.tsx` 新增 8 條——其中 4 條實作前真的紅，「選取模式沒有排序鈕」「qBT 錯誤沒有排序鈕」兩條實作前是**空轉綠**（按鈕還不存在），實作後由 mutation「拿掉 `showToolbar` 條件」證明會紅；另兩條（Safari 式點擊後焦點回排序鈕、qBT 斷線時抽屜跟著關）是寫完 mutation 清單時補的，由 mutation 證明。
  - 守：`ui/Sheet.spec.tsx` 2 條（不給新 prop 時 testid／內距／標題不變、沒 title 時 sr-only「選單」）；`DownloadsBrowseV2.spec.tsx` 既有 12 條一字未改；`MobileTabBar.spec.tsx`（會打開「更多」抽屜）等四支 Sheet 使用者的 spec 改前改後都綠（46 → 46）。
- **Mutation check（每一項拿掉 → 必須紅）：unit 24／24 紅、e2e 5／5 紅**
  - `ui/Sheet`：testId 被忽略、className 不走 `cn()`、titleClassName 丟掉、finalFocus 沒轉傳、onOpenChangeComplete 沒轉傳、預設 testId 改掉 → 全紅。
  - `useIsPhone`：寫成 `!(min-width: 640px)`（5 條紅）、拿掉沒有 `matchMedia` 的防護、不 `removeEventListener` → 全紅。
  - `DownloadSortSheet`：未選列沒有佔位、每列都可 tab、上下鍵不循環、選了不關、已選列沒有強調色 → 全紅。
  - `DownloadsBrowseV2`：排序鈕不帶 `sm:hidden`、不看 `showToolbar`（3 條紅）、select 不藏、抽屜的 onChange 沒接、拿掉 `finalFocus={sortBtnRef}`、拿掉「toolbar 消失就關抽屜」、膠囊列拿掉 `overflow-x-auto`／`-my-1`／`flex-nowrap`、膠囊拿掉 `shrink-0` → 全紅。
  - e2e（真瀏覽器量版面）：拿掉 `-my-1 py-1`（焦點框空間）、拿掉 `flex-nowrap`、未選列沒佔位（標籤左緣跳動）、select 在手機可見、排序鈕在 640 可見 → 全紅。
- **閘門**：`format:check` ✅；`lint:all` 0 errors／129 warnings（既有批次，與 dsr-6f-4 同數；本張碰的檔案 0 warnings）；`web:typecheck --skip-nx-cache` ✅；`check-design-tokens.py` ✅；`nx test web` **4007／4007**（277 檔）；`nx test api` ✅（跑完已刪 `apps/api/coverage/`）；e2e `chromium` `downloads-mobile`＋`downloads-v2` `--repeat-each=3` **33／33**；`mobile-chrome` 7 passed／4 skipped（新 spec 3 條是 chromium-only、表格那條被新 skip 擋下）；visual 整支跑三次：新基準線三次都一致，既有 `downloads-*` 基準線**零變動**，唯一紅的是本機固定會漂的 `retry-retry-notifications`（已立 `preexisting-fail-visual-darwin-three-stale-baselines`，CI 不受影響）。每次測試後 `test:cleanup` 皆無殘留。
- 🔗 AC Drift: NONE (checked: `'排序方式\|bottom-sheet\|flex-wrap\|排序'` across _bmad-output/implementation-artifacts/*.md — 相關命中 3 處：`dsr-4` AC #6「排序一個下拉同時決定欄位與方向」、`ux3-4-4` AC #3「欄頭排序共用同一組 sortField/sortOrder」、`dsr-6f-1` 裁定 2「只存在於手機的面板用 `ui/Sheet`」；全部是 REUSE：`dsr-4` 的範圍明寫是桌機 D1–D7，桌機仍是那一個下拉；手機的抽屜也是一個選項同時決定欄位與方向、寫進同一組 state；本張正是照 6f-1 的分工用 `ui/Sheet`)
- 📎 Contract Stamps: NONE (本張不定義也不消費任何 `[@contract-v*]`；上游 `GET /downloads?sort=&order=`（`dsr-4`／`bugfix-f`）未 stamp＝implicit v0)
- 🎭 A11y Pre-Flight: PASS (3 components checked — `ui/Sheet`、`DownloadSortSheet`、`DownloadsBrowseV2`；0 jsx-a11y warnings on touched files, 0 introduced by this story；一個 jsx-a11y **error** 在開發中出現並修掉（見上：鍵盤處理移到列上）。四類：① 圖片 N/A；② modal 焦點——Base UI 鎖焦點、開啟時落在已選列、關閉回排序鈕（unit 以不移動焦點的 `fireEvent.click` 模擬 Safari、e2e 以真點擊＋Esc 各證一次）；③ aria-live N/A（沒有非同步出現的狀態）；④ 自訂元件——`radiogroup`／`radio`＋`aria-checked`、roving tabindex、上下鍵、抽屜 Esc 關閉＋開啟時取得焦點。已知缺口：抽屜沒有關閉鈕給觸控螢幕報讀 → 建單時已立 `disc-2026-09-ui-sheet-no-close-button`)
- 🎨 UX Verification: PASS（對照表見下；落差全部是本張範圍外、已有單子，或是建單裁定）
- Pre-existing fix: N/A（`nx test web`／`nx test api` 全綠；唯一的既有紅燈是本機 visual 漂移，已有追蹤條目）

- **/ship 對抗式 CR（2026-09-22，獨立 context 的審查代理；0 HIGH／5 MEDIUM＋1 推測 MEDIUM／12 LOW）**：吸收 14 項（見上方 Review Follow-ups，Rule 24 ①），每項都補了測試並做 mutation：unit 8／8 紅、e2e 8／8 紅（其中一條第一次寫錯變成語法錯誤、全部紅，改成乾淨的 mutation 重跑，只紅該紅的那一條）。**沒有照做的**：
  - #4 radiogroup 的鍵盤不符合 APG——AC #3 明文規定「上下鍵移焦點、Enter／Space 選取」，是建單裁定 6 → ③ `disc-2026-09-sort-sheet-radiogroup-keyboard-apg`（Sally 裁定）。
  - #11 舊 spec 與共用 stub 兩份——本張禁止動舊 spec → ③ `disc-2026-09-downloads-e2e-stub-duplication`。
  - #15 把手暗示可拖但拖不動——範圍外 → 併入 ③ `disc-2026-09-bottomsheet-grabber-three-variants`。
  - #12 的 85vh 與 DESIGN.md 80% 落差——早已記在同一張把手單子裡。
  - #18 基準線上的初始焦點框——已在 Step 9 表格說明（初始焦點落在已選列是 AC #3 要的）。
  - 與故事字面的差異（CR 後）：`useIsPhone` 的查詢字串從 AC 寫的 `(max-width: 639.98px)` 改成 `(width < 40rem)`——AC 的 px 寫法會在瀏覽器預設字級不是 16px 時與 CSS 版面矛盾；🔴 #15 的原則（直接取 `matches`、不取反）不變。`ui/Sheet` 多了 AC 沒列的 `description`／`descriptionClassName`。`<h1>` 加了 `tabIndex={-1}`（4b-2 原本就要加，提前到本張，已寫進 4b-2 的交接）。
  - CR 後重跑：`DownloadsBrowseV2` 等受影響 spec 206／206；手機 e2e 6 條；visual 整支：新基準線仍一致、既有零變動（`description` 換成 `Dialog.Description` 仍是同一個 `<p>`、同一組 class）。

#### 🎨 UX 對照（dev-story Step 9；量測＝390×844 chromium 夜行，基準＝`.pen` 節點值）

| 區域 | 稿（節點） | 實作（量測） | 相符？ | 要修？ |
| --- | --- | --- | --- | --- |
| 抽屜位置 | `t6GBBy` x0、寬 390、貼底 | x 0、寬 390、底邊 844（e2e 斷言） | ✅ | — |
| 抽屜外觀 | `$bg-secondary`、頂角 `$radius-xl`＝16、上框 `$border-subtle`、padding 上 8／下 20 | bg-secondary、16px、1px border-subtle、pt 8／pb 20 | ✅ | — |
| 抽屜高度 | 560（`fit_content`） | 544 | ≈ | 不修：差的 16px 全在標題以上——把手區（稿 24 高、把手置中；`ui/Sheet` 是 4px＋mb-3）與字的行高（稿 1.625、全站 `text-base`／`text-sm` 是 24／20）。把手已立 `disc-2026-09-bottomsheet-grabber-three-variants`；行高是全站慣例 |
| 把手 | `fcaAf` 40×4 `$text-muted` | 40×4 `--border-subtle` | ❌ | 不在本張（同上單子，建單明文不動） |
| 標題 | `lyZpl`「排序」BodyLg 16／600 `$text-primary`、左右 20 | 16px／600／text-primary、`px-5` | ✅ | — |
| 副標 | `vh7za` Body 14 `$text-secondary`、標題與副標 gap 4、下距 12 | 14px／text-secondary、`mb-1`＝4、`pb-3`＝12 | ✅ | — |
| 分隔線 | `hReu0` 1px `$border-subtle` 滿寬 | 1px border-subtle、寬 390 | ✅ | — |
| 選項區 | `Vs994` padding [4,8]、列距 2 | `px-2 py-1`、`gap-0.5` | ✅ | — |
| 列 | h52、padding [0,12]、gap 12、`$radius-md`＝8 | 52、pl 12、圖示 x 20→標籤 x 52、8px | ✅ | — |
| 已選列 | `$accent-subtle` 底、`check` 20 `$accent-text`、BodyLg 600 `$accent-text` | accent-subtle、Check 20、600、accent-text | ✅ | — |
| 未選列 | 透明、BodyLg 500 `$text-primary`、20px 圖示位（本張改稿） | 透明、500、text-primary、空的 20px 佔位 | ✅ | — |
| 八列標籤左緣 | 全部 x=44（本張改稿） | e2e：八個左緣差 ≤1px | ✅ | — |
| 選項文字 | 八個，逐字＝`SORT_OPTIONS`（本張改稿） | 同一份 `SORT_OPTIONS` 傳入 | ✅ | — |
| 排序鈕 | `Q1EEk` 44×44 `$bg-tertiary` `$radius-md`、`arrow-down-up` 20 `$text-secondary` | 44×44、bg-tertiary、8px、ArrowDownUp 20、text-secondary、右緣＝根節點右緣 −16 | ✅ | — |
| 排序鈕垂直位置 | 與 `TopAppBar` 的 H4「下載」置中 | 對齊頁面標題列頂端（`items-start`；標題 `text-2xl`＋副標） | ≈ | 不在本張：標題列本身與稿不同（`disc-2026-09-d1-m-downloads-page-not-aligned` ③） |
| 膠囊列 | `eqXdV` 單行、左右 16、gap 8、上下 12 | 單行橫向捲動、pl 16、gap 8、上下 4（替焦點框留的 `py-1`，頁面 `gap-5` 管外距） | ≈ | 垂直間距屬頁面其餘對齊（同上單子）；單行／左右／gap 相符 |
| 膠囊高 | 40 | 44 | ❌（刻意） | 建單裁定 3：維持 44（觸控目標） |
| 提示文字 | `lZZd6`「點 ⋯ 開啟動作」（本張改稿） | 程式碼沒有這一行提示 | ❌ | 不在本張：D1-M 其餘對齊（同上單子） |
| 開啟時的焦點框 | 稿沒有畫焦點狀態 | 夾具基準線上已選列有黃色焦點框：夾具一掛載就是開的，Base UI 把初始焦點放到已選列，Chromium 視為鍵盤焦點；**真的點排序鈕打開時不會出現**（e2e 截圖確認） | — | 不修：這是鍵盤使用者該看到的，初始焦點落在已選列是 AC #3 要的 |

### Discovery Triage

<!-- Rule 24 — project-context.md. Any out-of-scope finding MUST land in exactly one lane with its
     sprint-status.yaml entry ID (② / ③) or absorbed AC # (①) BEFORE this story is marked done. -->

- **建單時的發現（SM Bob 2026-09-22）：**
  - ③ D1-M 頁面其餘的對齊＋「左滑暫停」手勢 → `disc-2026-09-d1-m-downloads-page-not-aligned`
  - ③ 後端支援 `size`／`eta` 排序、前端沒有 → `disc-2026-09-downloads-sort-by-size-and-eta`
  - ③ Flow D 的新抽屜把手是第三種畫法 → 補記 `disc-2026-09-bottomsheet-grabber-three-variants`
  - ③ `ui/Sheet` 沒有 `<Dialog.Close>`（Base UI 對觸控螢幕報讀的建議）→ `disc-2026-09-ui-sheet-no-close-button`
  - ① 舊 e2e `downloads-v2.spec.ts:304`（表格）在手機 project 本來就紅 → AC #5／Task 5（加 viewport<1024 skip；dev 先確認）
- **dev-story 期間的發現：** N/A — no out-of-scope work discovered。（本機 visual 的 `retry-retry-notifications` 紅燈是既有的，早已立 `preexisting-fail-visual-darwin-three-stale-baselines`；建單時的 ① 已確認並吸收：舊 e2e 表格那一條在 main 版本以 `mobile-chrome` 跑確實紅——`表格檢視` 按鈕 15 秒內找不到——已加 skip。）

### File List

- `ux-design.pen` — D10-M 八列＋圖示位＋刪 `P77zR`＋抽屜貼底、D1-M 提示、`spec-note-dsr-4b-1`
- `_bmad-output/pen-tokens.json` — `clippingWarnings` 69→68、`penSha256`
- `_bmad-output/screenshots/flow-d-downloads-v2/d1-m-v2.png`
- `_bmad-output/screenshots/flow-d-downloads-v2/d10-m-v2.png`
- `apps/web/src/components/ui/Sheet.tsx` — 五個新 prop
- `apps/web/src/components/ui/Sheet.spec.tsx`（新）
- `apps/web/src/hooks/useIsPhone.ts`（新）
- `apps/web/src/hooks/useIsPhone.spec.ts`（新）
- `apps/web/src/components/downloads/DownloadSortSheet.tsx`（新）
- `apps/web/src/components/downloads/DownloadSortSheet.spec.tsx`（新）
- `apps/web/src/components/downloads/DownloadsBrowseV2.tsx` — 排序鈕、抽屜、select 手機藏、膠囊列單行捲動、檔頭
- `apps/web/src/components/downloads/DownloadsBrowseV2.spec.tsx` — 新增 10 條（既有 12 條未改；檔頭 mock `useIsPhone`，預設 false＝與全域 stub 相同）
- `eslint.config.mjs` — DOM 型別白名單加 `HTMLHeadingElement`（`useRef<HTMLHeadingElement>`；與同一串 `HTMLSelectElement` 等並列）
- `apps/web/src/routes/test/-gallery.fixtures.tsx` — 夾具 `downloads-mobile-sheets/sort`
- `tests/visual/components.visual.spec.ts-snapshots/components/downloads-mobile-sheets/sort/default-visual-darwin.png`（新；`-linux` 由 CI bootstrap PR 補）
- `tests/e2e/downloads-mobile.spec.ts`（新）
- `tests/support/helpers/downloads-stubs.ts`（新）
- `tests/e2e/downloads-v2.spec.ts` — 表格那一條加 viewport<1024 skip（只有這一處）
- `_bmad-output/implementation-artifacts/sprint-status.yaml` — 狀態
- `_bmad-output/implementation-artifacts/dsr-4b-1-mobile-sort-sheet-and-chip-row.md` — 本檔

## Change Log

- 2026-09-22 — 建單（SM Bob，create-story；main `ae204761`）。原本是一張 `dsr-4b`，建單後對抗驗證（2 CRITICAL＋12 SHOULD FIX＋9 NIT）指出規模過大與兩個嚴重問題，**全部併入**並依版面拆成 `-1`（本張）與 `-2`。併入本張的重點：① `useIsPhone` 原本寫成 `(min-width:640px)` 取反——`test-setup.ts` 的全域 stub 回 false，那樣會把**每一支** jsdom 測試丟進手機路徑 → 改成直接問 `(max-width: 639.98px)`；② `ui/Sheet` 要五個 prop 不是三個（`titleClassName`、函式版 `finalFocus`、`onOpenChangeComplete`）；③ `p-0` 會連 safe-area 的 padding 一起丟掉 → className 明寫；④ 橫向捲動會裁掉焦點框 → `-my-1 py-1`；⑤ 夾具的 `penNode` 是必填；⑥ e2e 要自己設 viewport；⑦ 不寫死稿的抽屜高度。
- 2026-09-22 — Task 1（dev-story, Amelia）：D10-M 八個真的排序選項＋每列圖示位＋刪 `P77zR`（`problems` 69→68）＋抽屜 560 高貼底；D1-M 提示拿掉「左滑」；規格註記 `spec-note-dsr-4b-1`；存檔驗證、只 stage 兩張截圖＋`pen-tokens.json`。
- 2026-09-22 — Task 2：`ui/Sheet` 五個新 prop（預設值讓既有兩個使用者不變）＋`Sheet.spec.tsx`；`useIsPhone`＋spec。
- 2026-09-22 — Task 3：`DownloadSortSheet`（radiogroup、roving tabindex、上下鍵循環、選了就關）＋spec。
- 2026-09-22 — Task 4：`DownloadsBrowseV2` 標題列排序鈕（`sm:hidden`、跟著 `showToolbar`）、抽屜走既有 `handleSortOption`、select 手機藏（留在 DOM）、膠囊列手機單行捲動。
- 2026-09-22 — Task 5：夾具 `downloads-mobile-sheets/sort`＋darwin 基準線（連跑三次一致）、手機 e2e 3 條（`--repeat-each=3` 綠）、舊 e2e 表格那一條加 viewport skip（先在 main 版本確認 `mobile-chrome` 本來就紅）、mutation check unit 24／24＋e2e 5／5 全紅、Step 9 對照表；Status → review。
- 2026-09-22 — /ship 對抗式 CR：吸收 14 項（轉橫向關抽屜＋焦點後備、`useIsPhone` 改 rem、`ui/Sheet` 加 `description`、排序鈕 `aria-expanded`、第一列補 tab 位、選中膠囊捲進列內、e2e 競態與五條弱斷言），另立 `disc-2026-09-sort-sheet-radiogroup-keyboard-apg`、`disc-2026-09-downloads-e2e-stub-duplication`，補記把手單。
- 2026-09-22 — 合併：PR #502（squash `52444ef8`），`-linux` 基準線由 bootstrap PR #503 補上；PR 上 CI 全綠（Tests／E2E 4 片／Visual diff 4 片／Docker／Lint）。Status → done。
