# Story DSR.4b-2：手機上按下載卡片的 ⋯ 會滑出大按鈕的動作抽屜，還能打開「詳細資訊」看 Hash 與儲存路徑

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As the person who manages downloads from a phone,
I want 按卡片的 ⋯ 會從底部滑出一個每一列都很大顆的動作抽屜，並且能打開一張「詳細資訊」,
so that 我用拇指就能暫停、移除，也終於看得到 Hash 與檔案存在哪裡——這些今天在任何裝置上都看不到。

## Context

`dsr-4b`（Flow D 手機抽屜）拆出來的**第二塊：卡片動作＋詳細資訊**。

🔒 **Depends on: `dsr-4b-1-mobile-sort-sheet-and-chip-row` 先合併。** 本張直接使用 4b-1 做好的 `ui/Sheet` 新 prop（`testId`／`className`／`titleClassName`／函式版 `finalFocus`／`onOpenChangeComplete`）與 `hooks/useIsPhone.ts`。⚠️ 4b-1 會改 `DownloadsBrowseV2.tsx`（標題列、膠囊列、排序抽屜），本張引用的該檔行號在 4b-1 合併後會位移——**以符號為準，不要照行號**。

| 稿 | 節點 | 是什麼 | 程式碼現況 |
| --- | --- | --- | --- |
| D8-M-v2 · 手機動作 sheet | `jDgxJ` → `aJFk0` | **卡片動作**抽屜：詳細資訊／暫停／移除（保留檔案）／——／移除（連同檔案刪除） | `DownloadRowActions.tsx` 的 Radix DropdownMenu（全 app 唯一一處），手機也是它 |
| D9-M-v2 · 手機詳細 sheet | `DrYXb` → `g4A3Qk` | **詳細資訊**抽屜：標題、狀態、進度、八格資料、底部「暫停＋⋯」 | **完全沒有**——桌機也沒有任何詳細資訊（`dsr-4` 刪掉了 v1 的 `DownloadDetails`） |

⚖️ **Alexyu 2026-09-22（建單當下）裁定：詳細資訊抽屜要做。** 資料清單 API 已經都有，不用改後端。

⚠️ **驗收基準是 `.pen` 節點值**。PNG 只是參考。
⛔ **本張不進**：桌機的任何版面與 DOM（≥640 一個像素、一個 role 都不能變）、表格檢視、批次選取列、排序與膠囊列（4b-1 已做）、後端。

### 🔴 建單時查到的事（main `ae204761`；行號皆為現況）

**卡片動作（D8-M）**

1. `DownloadRowActions.tsx`（224 行），卡片（`DownloadCardV2.tsx:140-145`）與表格（`DownloadsTableV2.tsx:212-218`，`variant="table"`）共用：狀態鈕（`stateActionFor()` `:50-67`：paused→繼續、error→重試、completed→沒有、其餘→暫停）＋ ⋯ DropdownMenu（`:114-168`，`modal={false}`，理由寫在 `:115-116`）＋刪除確認 Dialog（`:170-221`；`onCloseAutoFocus` 把焦點送回 `moreRef`）。**只有「連同檔案刪除」要確認**（`:72-77` 的註解就是規則）。整個元件**零 testid**，測試全用無障礙名稱。沒有 busy／disabled 狀態。
2. 選單項目的圖示：程式碼與桌機稿 D3-D（`lCFq2`）都是 `FolderMinus`／`folder-minus`；**手機稿 D8-M 畫的是 `circle-minus`**（`i6jEmH`）→ 稿錯，改稿。
3. 稿的第一項「詳細資訊」（`KTyMD`，`info`）桌機選單沒有——它是手機獨有的入口（裁定 2）。
4. 🚨 **`role="menuitem"` 的爆炸半徑**：`DownloadRowActions.spec.tsx` 有 9 處 `findByRole('menuitem', …)`、`DownloadsTableV2.spec.tsx:135-137` 1 處、e2e `downloads-v2.spec.ts:215` 1 處。`ui/Sheet` 是 Dialog，裡面是普通 `<button>`。→ 手機路徑**不准**改到桌機路徑的 DOM。
5. **移除是樂觀更新**：`useDownloadActions.ts:49-52` 在 `onMutate` 就把項目從快取濾掉 → 按下「保留檔案」或確認框的「刪除檔案」，**那張卡片和它的 ⋯ 立刻卸載**——存下來的觸發元素變成游離節點，焦點不能送回去（會掉到 `<body>`）。卡片以 hash 為 key（`DownloadsBrowseV2.tsx:500`），所以**輪詢本身不會**讓元素失效；只有移除、換頁、換篩選會。
6. 暫停／繼續也是樂觀更新：`useDownloadActions.ts:53-54` 先改快取裡的狀態，`onSettled` 再 invalidate（＝立刻重抓）。不是「等下一次輪詢」。
7. z-index：`ui/Sheet` `z-[70]`／`z-[71]` > `ui/Dialog` `z-50` → **刪除確認框不能在抽屜還開著的時候開**（會被蓋在下面，兩個焦點鎖也會打架）。

**詳細資訊（D9-M）**

8. 稿的八格：下載速度／上傳速度／進度（`5.1 GB / 8.1 GB`）／剩餘時間／來源（`qBittorrent`）／加入時間（`2026-06-30 21:14`）／Hash／儲存路徑。`Download`（`downloadService.ts:44-66`）全部給得出來；「來源」今天是常數（卡片也是寫死的 `qBittorrent` 膠囊）。`addedOn` 是 RFC3339 的 UTC 字串（`torrent.go:105, 203`）。
9. **稿上兩樣東西沒有資料來源**：`KB5wY` 檔名（`Download` 只有種子名稱 `name`）、`dC4Cg` 技術徽章列 2160p／HDR10／TrueHD／BluRay（卡片程式碼不解析這些）→ 改稿拿掉（裁定 4）。
10. 🚨 **卡片的數字不是直接讀欄位。** `formatDownloadMeta(d)`（`formatters.ts:76-96`）回 `{ down: '↓ …', up: '↑ …', eta, size }`——`down`／`up` **帶箭頭**；`size` 是用 **`progress × size`** 算的，不是 `download.downloaded`（`DownloadsBrowseV2.spec.tsx:60` 的夾具就是 `downloaded: 0`＋`progress: 0.5`）。詳細資訊若讀 `downloaded` 會顯示「0 B / …」、與同一張卡片互相矛盾。其他 export：`formatSpeed(n)`、`formatSize(n)`、`formatETA(n)`、`formatProgress(progress)`（→「62.4%」）。
11. 狀態的文字與顏色：`getDownloadStatus(status)`（`downloadStatus.ts:20-27`）的 `className` 是**膠囊的底色＋字色**，不能拿來當一行文字的顏色。文字用 `.label`；顏色與進度條用 `getDownloadTone(download)`（`downloadStatus.ts:63`）的 `.text`／`.fill`——卡片 `DownloadCardV2.tsx:54, 109, 118` 就是這樣用。`DownloadStatusPill` **已經** export（`DownloadCardV2.tsx:23`）。
12. 時間：`new Date(iso)`（帶參數）**不觸發** Rule 23 的 lint（`time-dependent-fixture-stability.js:14, 113`）；但「2026-06-30 21:14」是**時區相依**的字串——本機 +8、CI 是 UTC。全 repo 沒有絕對時間的格式化函式。
13. `getDownloadDetails`（`downloadService.ts:176`）今天沒有任何呼叫端，本張也不用它。

**hook 與既有測試**

14. `useIsPhone`（4b-1）問的是 `(max-width: 639.98px)`；`test-setup.ts:80-90` 的全域 stub 回 false → jsdom 預設＝不是手機。🚨 但 `DownloadsBrowseV2.spec.tsx:226-235` 那一條自己 stub 了**對任何 query 都回 true** 的 `matchMedia` → 本張讓 `DownloadRowActions`／`DownloadsBrowseV2` 讀 hook 之後，那一條會掉進手機路徑。→ 在 `DownloadsBrowseV2.spec.tsx` 檔頭 mock hook（AC #6）。加 mock 不算改寫斷言。
15. e2e：`downloads-v2.spec.ts:183`「⋯ menu → 連同檔案刪除」用 `getByRole('menuitem')`；本機 `mobile-chrome`／`mobile-safari` project（393／390 寬）在本張之後會走抽屜 → 找不到 `menuitem`。CI 只跑 `chromium`＋`webkit-core`，不受影響。
16. `GalleryFixture.penNode` 必填；Portal 夾具的 state div 高度為 0 → 視覺 spec 拍整個視窗（4b-1 已替 `downloads-mobile-sheets/sort` 走過一次這條路——照抄）。

### 設計稿節點

| 代號 | 節點 | 本張要看的 |
| --- | --- | --- |
| D8-M-v2 | `jDgxJ`（17530,24325，390×844）→ `aJFk0` actionSheet（y=504、390×340、padding [8,0,20,0]、`$bg-secondary`、頂角 `$radius-xl`） | 把手 `KzeuZ`／`hmhlF`；`J7QMBQ` sheetHeader（[4,20,12,20]、gap 4）：`pQtcH` BodyLg 600 片名、`Gd6gf` metaRow（`rqbiJ` Body 500＋`W2oZXn`「· 62.4%」，兩者 `$accent-text`）；`payIm` 分隔線；`C31QIb` actions（[4,8]、gap 2）：每列 h52、padding [0,12]、gap 12、`$radius-md`、圖示 20、標籤 BodyLg 500；列：`KTyMD` info／`TAE3d` pause／`KQi9B`（圖示 `i6jEmH` **要改**）／`Guoqw` 分隔線／`TVVAF` trash-2，圖示與字 `$error-text` |
| D9-M-v2 | `DrYXb`（18020,24325）→ `g4A3Qk` detailSheet（y=150、高 694、padding-top 8） | `bhoYP` content（[4,20,16,20]、gap 16）：`CwAjz`（`Vvdar` H4 700、`dC4Cg` 技術徽章列 **要拿掉**、`ensI9` 狀態膠囊）、`OIX0v` 進度列（軌道 h8＋`pIPlp` BodyLg 600）、`KB5wY` 檔名 **要拿掉**、`HxVqo`「詳細資訊」Label 600、`Mpq5S` 格子（gap 12；三列兩欄＋Hash 整列＋儲存路徑整列；標籤 Label `$text-secondary`、值 Body `$text-primary`）；`DrHEM` actionBarWrap（[12,20,24,20]、上框線、**固定在底**）→ `Wydrn` 主要鈕 h48 `$accent-primary` 滿版＋`xshlR` ⋯ 48×48 `$bg-tertiary` |
| D3-D-v2（對照） | `lCFq2` | 桌機選單：pause／**folder-minus**／trash-2；確認框文案 |
| Flow D 說明 | `ggTb3`（17040,22467） | 「深層下載操作頁為 design-ahead 規格稿（後端尚未實作）」——本張之後不再成立，改掉這半句 |

字階：H4 18、BodyLg 16、Body 14、Label 12。全檔 `problems`：以 4b-1 合併後的數字為基準（預期 68），**只減不增**。

---

## Acceptance Criteria

1. **設計稿先改（Pencil MCP；每個節點動手前再 `Get` 一次）。**
   - **D8-M** `jDgxJ`：`i6jEmH` 的圖示 `circle-minus` → `folder-minus`。
   - **D9-M** `DrYXb`：拿掉 `dC4Cg`（技術徽章列）與 `KB5wY`（檔名）；`Vvdar` 的範例字改成種子名稱的樣子（與卡片母版 `TFYEw` 同一句「沙丘：第二部 Dune: Part Two (2024) 2160p UHD BluRay」，會換行）。`g4A3Qk` 是固定高 694——拿掉兩塊之後改成剛好包住內容並重新貼底（以 `ctx.bounds` 讀實高再 `y = 844 − 高度`；不得出現新的 `problems`）。
   - `ggTb3`：句尾改成「…→ 動作 sheet · 排序 sheet · 詳細資訊 sheet。」（拿掉 design-ahead 那半句）。
   - 規格註記：放在 4b-1 的 `spec-note-dsr-4b-1` 下方 40px（先 `Get` 量它的實高），名稱 `spec-note-dsr-4b-2`：
     > 「D8／D9（手機）：兩個都是只存在於手機的底部抽屜（<640）；桌機維持下拉選單，桌機沒有詳細資訊。一次只開一個覆蓋層：動作 → 詳細、詳細 → 動作、動作 → 刪除確認，全部是先關再開。**動作抽屜**：標題是種子名稱（最多兩行）＋狀態・百分比；『連同檔案刪除』先關抽屜再跳確認框，其餘動作按了就做、抽屜關閉；已完成的項目沒有暫停／繼續那一列。**詳細資訊抽屜**：資料全部來自清單（不另外打 API），數字與卡片用同一套算法；開著的時候跟著清單更新；項目從清單消失就自動關閉。沒有檔名與技術徽章（系統沒有這兩樣資料）。內容可捲、底部動作列固定：主要鈕＝該項目的狀態動作（暫停／繼續／重試），按了**不關**抽屜；⋯ 換到動作抽屜；已完成的項目只有一顆『更多動作』。關掉後焦點回到那張卡片的 ⋯；卡片已經不在了（被移除）就回到頁面標題。」
   - 收尾：`problems` 只減不增；存檔走選單 Save，`git status --porcelain ux-design.pen` 必須出現 ` M`，並 grep 磁碟檔確認 `spec-note-dsr-4b-2` 在裡面；**存檔後**才匯出；**只 stage** `flow-d-downloads-v2/d8-m-v2.png`、`d9-m-v2.png` 與 `_bmad-output/pen-tokens.json`。

2. **誰擁有什麼（先讀這一條，再讀 #3–#5）。** 手機的兩個抽屜與它們的確認框都由 **`DownloadsBrowseV2`** 擁有；`DownloadRowActions` 在手機只負責「把 ⋯ 被按了這件事往上報」。
   - `DownloadsBrowseV2` 的 state：
     - `const [sheet, setSheet] = useState<{ kind: 'actions' | 'detail'; hash: string } | null>(null)`——**聯集型別＝「一次只開一個」由結構保證**；
     - `const [confirmTarget, setConfirmTarget] = useState<Download | null>(null)`——存**物件快照**不是 hash：確認框要顯示名稱與大小，而「刪除檔案」按下去之後項目已經從清單消失（🔴 #5），用 hash 回頭查會讓確認框在退場動畫期間變空；
     - `const triggerRef = useRef<HTMLElement | null>(null)`（那張卡片的 ⋯）、`const handoffRef = useRef(false)`（這次關閉是不是「換手」）、`const headingRef`（`<h1 tabIndex={-1}>`，焦點的後備落點）。
   - `const sheetDownload = sheet ? items.find((d) => d.hash === sheet.hash) : undefined`——抽屜讀的永遠是**最新的清單項目**。`useEffect`：`sheet` 有值但 `sheetDownload` 找不到（被移除、換頁、換篩選）→ `setSheet(null)`；`isPhone` 變 false → `setSheet(null)`。
   - **焦點**：兩個抽屜的 `finalFocus` 都傳同一個**函式**：`() => handoffRef.current ? false : (triggerRef.current?.isConnected ? triggerRef.current : headingRef.current)`。
     - 換手（動作→詳細、詳細→動作、動作→確認框）時先 `handoffRef.current = true` 再改 state，下一個 effect 重設成 false。回 `false`＝Base UI 這次不搬焦點——讓下一個覆蓋層自己的 `initialFocus`／Radix 的 `onOpenAutoFocus` 說了算，**不會有兩套焦點管理搶同一個 tick**。
     - 真的關閉（Esc、點遮罩、做完動作）→ 回到 ⋯；⋯ 已經卸載（項目被移除）→ 回到 `<h1>`。焦點**永遠不准**落在 `<body>`。
   - **頁面層級的確認框**：`<DeleteWithFilesDialog download={confirmTarget} open={confirmTarget !== null} …>`；`onCloseAutoFocus`：`preventDefault()` 後聚焦同一個解析結果（⋯ 還在就 ⋯，否則 `<h1>`）。

3. **`DownloadRowActions`：手機路徑只多一個 prop；桌機路徑一個字都不變（🔴 #1–#4）。**
   - 唯一的新 prop：`onOpenActions?: (hash: string, trigger: HTMLElement) => void`。⛔ 沒有 `onShowDetails`、沒有內建抽屜、沒有 `sheetOpen` state。
   - `const isPhone = useIsPhone(); const useSheet = isPhone && variant === 'card' && !!onOpenActions;`
     - `useSheet === true`：⋯ 是一顆普通 `<button type="button">`——同一個 `aria-label={`更多動作：${download.name}`}`、同一組 class、同一個 `moreRef`，外加 `aria-haspopup="dialog"`；`onClick={(e) => onOpenActions(download.hash, e.currentTarget)}`。不渲染 `DropdownMenu`。
     - `useSheet === false`：**現有 JSX 原封不動**。手機上沒傳 `onOpenActions` 的卡片（gallery、單獨渲染的測試）照樣得到下拉選單——它在任何寬度都能用，所以不需要「內建抽屜」這個後備。
     - `variant === 'table'` 的判斷要留著：表格 `lg` 以上才有，這一行是防呆（也保護 `DownloadsTableV2.spec` 與 `DownloadsBrowseV2.spec.tsx:226`）。
   - 把 `:170-221` 的確認框抽成 `downloads/DeleteWithFilesDialog.tsx`（檔頭 `// Design ref: ux-design.pen Screen D3-D-v2 (lCFq2)`）：props `{ download: Pick<Download,'hash'|'name'|'size'> | null; open; onOpenChange; onConfirm: (hash) => void; onCloseAutoFocus? }`；標題、說明、檔案列（`size === 0` 不顯示大小）、「取消」「刪除檔案」**逐字照搬**。`DownloadRowActions` 的桌機路徑改用它（行為與 DOM 不變）。📎 這是本張唯一的結構性重構：**先**跑 `DownloadRowActions.spec.tsx` 確認 16 條綠 → 抽出 → **仍然** 16 條綠（含 `findByRole('dialog', { name: '移除並刪除檔案？' })` 與焦點回 ⋯）→ 才開始加手機路徑。做不到原樣綠就**停下來回報**，不要改舊測試。
   - `stateActionFor` 從 `DownloadRowActions.tsx` **export** 出來給兩個抽屜共用（⛔ 不准複製一份）。
   - `DownloadCardV2` 多收一個 `onOpenActions` 並透傳。Rule 21：`DownloadRowActions.tsx:1` 行尾接 ` · D8-M-v2 (jDgxJ)`。

4. **動作抽屜（D8-M）——純呈現元件。**
   - 新檔 `downloads/DownloadActionsSheet.tsx`（`// Design ref: ux-design.pen Screen D8-M-v2 (jDgxJ)`）。Props：`{ download, open, onOpenChange, finalFocus, onPause?, onResume?, onRemove?, onShowDetails?, onRequestDeleteWithFiles }`。
   - `ui/Sheet`：`testId="download-actions-sheet"`、`title={download.name}`、`titleClassName="mb-1 px-5 line-clamp-2"`（種子名稱很長；⛔ 不要再另外給 `ariaLabel`——`title` 就是 `Dialog.Title`）、`className="p-0 pt-2 pb-[max(1.25rem,env(safe-area-inset-bottom))]"`（📎 `p-0` 會連 safe-area 的 padding 一起丟掉，所以要自己加回去——4b-1 已實測）。
   - 標題下一行（`px-5 pb-3 text-sm`）：`getDownloadStatus(download.status).label`＋「 · 」＋`formatProgress(download.progress)`，顏色 `getDownloadTone(download).text`（🔴 #11）。分隔線。動作區 `px-2 py-1`、列距 2px。
   - 動作列（每列一顆 `<button>`：`min-h-[52px]`、`px-3`、`gap-3`、`rounded-[var(--radius-md)]`、圖示 `size-5`、標籤 `text-base font-medium`）：
     1. 「詳細資訊」`Info`——只有傳了 `onShowDetails` 才畫；
     2. 狀態動作（`stateActionFor()` 的 label／icon／run）——completed 沒有這一列；
     3. 「移除（保留檔案）」`FolderMinus` → `onRemove(hash, false)`，不確認；
     4. 分隔線；
     5. 「移除（連同檔案刪除）」`Trash2`，`text-[var(--error-text)]` → `onRequestDeleteWithFiles()`。
   - 2、3 按了之後 `onOpenChange(false)`。1 與 5 是**換手**，關閉由擁有者（AC #2）處理——元件只呼叫 callback。

5. **詳細資訊抽屜（D9-M）——純呈現元件，資料全部來自清單（🔴 #8–#13）。**
   - 新檔 `downloads/DownloadDetailSheet.tsx`（`// Design ref: ux-design.pen Screen D9-M-v2 (DrYXb)`）。Props：`{ download, open, onOpenChange, finalFocus, onPause?, onResume?, onOpenActions }`。
   - **內容捲、動作列固定**：`ui/Sheet` `testId="download-detail-sheet"`、`title={download.name}`、`titleClassName="mb-2 px-5 text-lg font-bold break-words"`、`className="flex flex-col overflow-hidden p-0 pt-2"`；children＝`<div className="min-h-0 flex-1 overflow-y-auto px-5 pb-4">…</div>`＋`<div className="shrink-0 border-t border-[var(--border-subtle)] px-5 pt-3 pb-[max(1.5rem,env(safe-area-inset-bottom))]">…</div>`。把手與標題在捲動區外（固定）。📎 Popup 本身是 `overflow-y-auto`——不這樣改，底部動作列會跟著內容捲走。
   - 捲動區：`DownloadStatusPill`（已 export 的那一顆）；進度列（軌道 h8＋`formatProgress` 的百分比 `text-base font-semibold`；`role="progressbar"`＋`aria-valuenow`；填色 `getDownloadTone(download).fill`、百分比字色 `.text`）；小標「詳細資訊」（`text-xs font-semibold text-[var(--text-secondary)]`）；`<dl>` 格子（兩欄 `grid-cols-2 gap-3`，Hash 與儲存路徑 `col-span-2`；`<dt>` `text-xs text-[var(--text-secondary)]`、`<dd>` `text-sm text-[var(--text-primary)]`）：
     | `<dt>` | `<dd>` |
     | --- | --- |
     | 下載速度 | `meta.downSpeed` |
     | 上傳速度 | `meta.upSpeed` |
     | 進度 | `meta.size`（**與卡片同一個算法**；⛔ 不讀 `download.downloaded`，🔴 #10） |
     | 剩餘時間 | `meta.eta` |
     | 來源 | `qBittorrent` |
     | 加入時間 | `formatAddedOn(download.addedOn)` |
     | Hash | `download.hash`（`font-mono text-xs break-all`） |
     | 儲存路徑 | `download.savePath`（`break-all`） |
     數字一律中性（`--text-primary`）——速度不是狀態（`disc-2026-08-download-speed-wears-completed-green` 的裁定）。
   - `formatters.ts`：
     - `formatDownloadMeta` 新增**不帶箭頭**的 `downSpeed`／`upSpeed`；既有的 `down`／`up` 改由它們組成（`↓ ${downSpeed}`）——**卡片的輸出逐字不變**（既有 `formatters.spec.ts` 守著）。
     - 新增純函式 `formatAddedOn(iso: string, timeZone?: string): string` → `YYYY-MM-DD HH:mm`：`Intl.DateTimeFormat('en-CA', { …, hourCycle: 'h23', timeZone })`＋`formatToParts`（⛔ 不要用 `hour12: false`——某些引擎午夜會給 `24:xx`）；`timeZone` 省略＝瀏覽器時區；無法解析回 `—`。元件不傳 `timeZone`；**unit 一律明傳 `'Asia/Taipei'`**（CI 是 UTC）。帶參數的 `new Date(iso)` 不觸發 Rule 23 的 lint，不用加標記。
   - 底部動作列：主要鈕＝`stateActionFor()` 的動作（`h-12`、`flex-1`、`bg-[var(--accent-primary)] text-[var(--text-on-accent)]`、圖示 `size-[18px]`＋`text-base font-semibold`；`error` 的「重試」也是同一個樣式——紅色留給狀態膠囊）；旁邊 ⋯（`size-12`、`bg-[var(--bg-tertiary)]`、`aria-label={`更多動作：${download.name}`}`）→ `onOpenActions()`（換手）。**completed**：只有一顆滿版的 Secondary「更多動作」→ `onOpenActions()`。按主要鈕**不關抽屜**——樂觀更新會讓膠囊立刻變成「已暫停」、主要鈕變「繼續」（🔴 #6）。

6. **接線（`DownloadsBrowseV2`）與既有測試的保護。**
   - 卡片：`onOpenActions={(hash, el) => { triggerRef.current = el; setSheet({ kind: 'actions', hash }); }}`（只在 `isPhone` 時傳也可以——`DownloadRowActions` 自己也會看 `isPhone`）。
   - 動作抽屜：`onShowDetails` → 換手＋`setSheet({ kind: 'detail', hash })`；`onRequestDeleteWithFiles` → 換手＋`setConfirmTarget(sheetDownload)`＋`setSheet(null)`；`onPause`／`onResume`／`onRemove` 就是傳給卡片的那三個既有 handler。
   - 詳細資訊：`onOpenActions` → 換手＋`setSheet({ kind: 'actions', hash })`。
   - 確認框 `onConfirm` → 既有的 `onRemove(hash, true)`。
   - `<h1>` 加 `ref={headingRef} tabIndex={-1}`（與 `outline-none`——它只會被程式聚焦）。Rule 21：檔頭補 D8-M／D9-M。
   - `DownloadsBrowseV2.spec.tsx` 檔頭加 `vi.mock('../../hooks/useIsPhone', () => ({ useIsPhone: () => h.isPhone }))`（`h.isPhone` 預設 false、`beforeEach` 重設）——保護 `:226-235` 那條全 true 的 `matchMedia` stub（🔴 #14）。`DownloadRowActions.spec.tsx`、`DownloadCardV2.spec.tsx` 同樣用 mock 進手機路徑；⛔ 不要在元件測試裡 stub `matchMedia`。

7. **既有的行為不准回歸。**
   - 桌機（≥640）：`downloads-*` 所有既有視覺基準線**零變動**；`DownloadRowActions.spec.tsx`（16）、`DownloadsBrowseV2.spec.tsx`、`DownloadsTableV2.spec.tsx`（8）、`DownloadCardV2.spec.tsx`（13）、`formatters.spec.ts` 既有斷言**一條都不改寫**。
   - `tests/e2e/downloads-v2.spec.ts` 在 `chromium` 全綠、斷言不改。只准：替 `:183`（⋯ menu）加 `test.skip((page.viewportSize()?.width ?? 1280) < 640, 'phones use the actions sheet — see downloads-mobile.spec.ts')`（寫在那一條裡面；⛔ 不要寫成 chromium-only）。這一條是**本張自己造成**的手機 project 失敗，不是既有的紅。
   - ⛔ 不改後端、`routes/downloads.tsx`、批次選取列、表格、`ui/Dialog.tsx`、`ui/mobileSheet.tsx`、`ui/Sheet.tsx`（4b-1 已定案；真的缺東西就停下來回報）、`useDownloadActions.ts`。

8. **測試。** 分「**紅**」與「**守**」，誠實標示（Rule 16）。
   - `DownloadActionsSheet.spec.tsx`：（紅）`stateActionFor` 的四個分支（paused→繼續、error→重試、completed→沒有狀態列、其餘以 `stalled` 或 `queued` 代表→暫停）；沒有 `onShowDetails` 就沒有「詳細資訊」；「保留檔案」→ `onRemove(hash,false)`＋`onOpenChange(false)`；「連同檔案刪除」→ **不**呼叫 `onRemove`、呼叫 `onRequestDeleteWithFiles`；狀態行的文字是 `label · 百分比`；標題是 `download.name`。
   - `DownloadDetailSheet.spec.tsx`：（紅）八個 `<dt>` 的文字與順序；速度**沒有箭頭**；「進度」在 `downloaded: 0, progress: 0.5` 的夾具下**不是**「0 B」（🔴 #10 的那個坑）；Hash／路徑逐字；completed 只有「更多動作」、沒有主要鈕；按主要鈕 → `onPause(hash)` 且 `onOpenChange` **沒有**被呼叫；按 ⋯ → `onOpenActions`。
   - `formatters.spec.ts`：（紅）`formatAddedOn('2026-06-30T13:14:00Z','Asia/Taipei')` → `2026-06-30 21:14`；`…T16:05:00Z` → `2026-07-01 00:05`（午夜不是 24）；亂字串 → `—`；`formatDownloadMeta` 的 `downSpeed`／`upSpeed` 沒有箭頭。（守）`down`／`up`／`eta`／`size` 既有輸出。
   - `DownloadRowActions.spec.tsx`：（紅，`isPhone` mock true）傳了 `onOpenActions` → ⋯ 是普通 button（有 `aria-haspopup="dialog"`）、按下以 `(hash, 那顆 button 元素)` 呼叫、**沒有** `menu`；沒傳 → 手機也走下拉（`menuitem` 在）；`variant="table"` 即使傳了也走下拉。（守）`isPhone` false 時既有 16 條原樣。
   - `DownloadsBrowseV2.spec.tsx`（`isPhone` mock true、渲染真的卡片）：（紅）按 ⋯ → `download-actions-sheet`；按「詳細資訊」→ 動作抽屜不見、`download-detail-sheet` 出現（同時最多一個）；詳細的 ⋯ → 換回動作抽屜；「連同檔案刪除」→ 抽屜不見、確認框出現且顯示該項目的名稱；「刪除檔案」→ `remove` 以 `(hash, true)` 被呼叫；清單重抓後那一筆的狀態變了 → 抽屜裡的膠囊跟著變；那一筆從清單消失 → 抽屜關閉；`isPhone` 由 true 變 false → 抽屜關閉。
   - **每一項修法做 mutation check**，結果寫進 Completion Notes。
   - ⚠️ `toHaveTextContent` 是子字串比對（「暫停」會命中「已暫停」）；用 role＋name。

9. **真瀏覽器的驗證。**
   - **視覺夾具**（照抄 4b-1 的 `downloads-mobile-sheets/sort`）：`downloads-mobile-sheets/actions`（`penNode: 'jDgxJ'`）、`downloads-mobile-sheets/detail`（`penNode: 'DrYXb'`）——`open: true`、固定的 `Download` 假資料、`viewport: { width: 390, height: 844 }`、`statesOnly: ['default']`。`detail` 的「加入時間」會經過瀏覽器時區——darwin 與 linux 基準線本來就分開、各自穩定；⛔ 不要為了讓兩邊一樣而在元件裡寫死時區。只產 darwin。
   - **e2e**：加到 4b-1 的 `tests/e2e/downloads-mobile.spec.ts`（同一套 helper、每條自己 `setViewportSize`、非 `chromium` skip、`Math.round`、等轉場結束）。
     - **390・動作**：按卡片 ⋯ → `download-actions-sheet` 貼底；頁面上**沒有** `role="menu"`；每一列高 ≥52、寬 ≈374（390−16）；按「移除（保留檔案）」→ 直接發 DELETE（URL **不含** `deleteFiles=true`）、沒有確認框、抽屜關閉、**焦點不在 `<body>`**（卡片已被樂觀移除 → 落在 `<h1>`）。
     - **390・連同檔案刪除**：按下 → 抽屜**先消失**、確認框出現且 `document.activeElement` 在確認框**裡面**；按「取消」→ 焦點回到那張卡片的 ⋯；再走一次按「刪除檔案」→ DELETE 帶 `deleteFiles=true`、焦點落在 `<h1>`。
     - **390・詳細資訊**：動作抽屜 →「詳細資訊」→ 任一時刻可見的 `[data-testid$="-sheet"]` 最多一個；Hash 與儲存路徑逐字、`<dd>` 沒有橫向溢出（`el.scrollWidth <= el.clientWidth`——📎 有牙：`break-all` 拿掉，40 個字的 hash 就會撐寬）；底部動作列**固定**：把捲動區捲到底，動作列的 `y` 不變；按主要鈕「暫停」→ 發 pause 請求、抽屜**還在**、膠囊立刻變「已暫停」、主要鈕變「繼續」，**之後不變回來**——📎 stub 要在**收到 pause POST 之後**才把清單裡那一筆改成 paused（樂觀更新之後馬上會重抓；用輪詢次數切換回應會讓畫面彈回「下載中」）；按 ⋯ → 換成動作抽屜；Esc → 焦點回卡片的 ⋯。
     - **640（斷點另一側）**：按 ⋯ → 出現的是 `role="menu"`，沒有任何 `-sheet`；選單裡**沒有**「詳細資訊」。
     - 本機跑要 `AI_PROVIDER=claude …`；過了再 `--repeat-each=3`。
   - dev-story Step 9：`d8-m-v2`／`d9-m-v2` 對新基準線與 e2e 量測逐項核。

10. **另立的單子**：沿用 4b-1 已立的三張；本張若發現新的，照 Rule 24 當場立。

11. **CI 全綠**：同 4b-1 AC #9（六個閘門；不在背景跑測試；`NX_DAEMON=false`；清 `apps/api/coverage/`；合併後看 `main` 的三條；`gh` 一律帶 `GH_TOKEN=$(gh auth token --user j620656786206)`）。

## Tasks / Subtasks

- [x] **Task 0 — 確認 `dsr-4b-1` 已合併；`git pull`；重新對一次本張引用的符號位置**
- [x] **Task 1 — 設計稿：D8-M 圖示、D9-M 拿掉兩塊沒資料的、Flow 說明、規格註記（AC: #1）**
- [x] **Task 2 — 確認框抽成 `DeleteWithFilesDialog`；`stateActionFor` export（AC: #3, #8）**
  - [x] 先 16 條綠 → 抽出 → 仍 16 條綠
- [x] **Task 3 — `formatDownloadMeta` 加 `downSpeed`／`upSpeed`、`formatAddedOn`（AC: #5, #8）**
- [x] **Task 4 — `DownloadActionsSheet`、`DownloadDetailSheet`（純呈現，各自的 spec）（AC: #4, #5, #8）**
- [x] **Task 5 — `DownloadRowActions` 的 `onOpenActions`；`DownloadsBrowseV2` 的狀態機、焦點、確認框；三支 spec 的 hook mock（AC: #2, #3, #6, #8）**
- [x] **Task 6 — 兩個視覺夾具、手機 e2e、舊 e2e 那一條的 viewport skip、mutation check、收尾（AC: #7, #9, #11）**
  - [x] dev-story Step 9：`d8-m-v2`／`d9-m-v2`

### Review Follow-ups (AI)

<!-- /ship 對抗式 CR（2026-09-22，獨立 context 的審查代理）— Rule 24 ① 吸收的項目，每項都有測試＋mutation。 -->

- [x] [AI-Review][MEDIUM] 「連同檔案刪除」的確認框（z-50）跟正在退場的抽屜（z-71＋遮罩）同一個 commit 出現——320ms 內確認框在抽屜**下面**滑進來 → 改成真的「先關再開」：`pendingRef` 記住下一個要開的東西，抽屜 `onOpenChangeComplete(false)` 才開它（動作↔詳細的換手也一樣，不再交叉淡入）
- [x] [AI-Review][MEDIUM] 桌機確認框開著時視窗縮到 640 以下，`confirmOpen` 留著、再放大會自己跳出「移除並刪除檔案？」→ `useSheet` 變 true 時重設；確認框改成手機也掛著（關著不畫東西）
- [x] [AI-Review][MEDIUM] e2e 的「重抓後仍是已暫停」靠 `waitForTimeout(700)`、換手後的焦點檢查靠 `waitForTimeout(500)` → 改成數清單請求次數、等舊抽屜 `toHaveCount(0)`
- [x] [AI-Review][LOW] 已完成項目的「更多動作」無障礙名稱跟其他狀態不一樣 → 統一 `更多動作：{name}`
- [x] [AI-Review][LOW] `handoffRef` 只靠 `finalFocus` 被呼叫才清 → 下一個抽屜 `onOpenChangeComplete(true)` 也清（防中途取消退場時卡住；沒有可觀察的失敗情境，mutation 綠，誠實記錄）
- [x] [AI-Review][LOW] `formatAddedOn` 每次 render 都 new 一個 `Intl.DateTimeFormat` → 依時區快取
- [x] [AI-Review][LOW，推測] 先關再卸載的修法靠 `--motion-move` 不是 0 → `styles.css` 的 reduced-motion 區塊加註解，並補一條 `reducedMotion: 'reduce'` 的 e2e（保留檔案 → 焦點回 `<h1>`）

## Dev Notes

### 這張的重點

- **`DownloadsBrowseV2` 擁有一切；卡片只往上報。** 一個聯集型別的 `sheet` state 讓「一次只開一個」不可能被寫錯。不要在 `DownloadRowActions` 裡放抽屜。
- **換手時 `finalFocus` 回 `false`。** 這是避免 Base UI 與 Radix 兩套焦點管理互搶的正解——不是「等一個 frame」。
- **移除是樂觀的。** 按下去卡片就沒了——焦點後備到 `<h1>`，確認框存物件快照。
- **數字跟卡片用同一個函式。** `meta.size` 是 `progress × size`；讀 `downloaded` 會跟卡片打架。
- **桌機一個 role 都不變。** `menuitem` 的 11 處斷言是護欄。

### 上游契約（Rule 20 ack）

- 本張不定義也不改任何線上契約。消費既有的 `GET /downloads`（清單項目欄位）、`POST …/pause|resume`、`DELETE …?deleteFiles=`——`dsr-4`／`bugfix-e`／`bugfix-f` 的未 stamp AC＝implicit v0。e2e 的假資料沿用 `downloads-v2.spec.ts` 既有的形狀。
- 對 `dsr-4b-1` 的依賴是**元件 API**（`ui/Sheet` 的五個 prop、`useIsPhone`），不是線上契約——若 4b-1 出貨的 API 與本張寫的不一樣，以 4b-1 的程式碼為準並在 Debug Log 記一筆。

### 建單裁定（2026-09-22，Sally／Alexyu 可在 review 推翻）

1. ⚖️ **詳細資訊抽屜要做**（**Alexyu 2026-09-22 當場裁定**）。
2. ⚖️ **「詳細資訊」只有手機有入口。** 桌機的下拉選單不加這一項（桌機沒有稿、會動桌機基準線與 `menuitem` 測試）；桌機要不要有詳細資訊是另一個產品問題。
3. ⚖️ **手機的抽屜與確認框由頁面擁有**；沒有傳 `onOpenActions` 的卡片在手機上仍是下拉選單（不做內建抽屜的後備——下拉在任何寬度都能用）。
4. ⚖️ **詳細資訊沒有檔名與技術徽章**（改稿）——系統沒有這兩樣資料，不為了稿去解析種子名稱。
5. ⚖️ **詳細資訊的主要鈕按了不關抽屜**；動作抽屜的每一項按了都關。前者是「看著它變」，後者是「做完就走」。
6. ⚖️ **焦點的後備落點是頁面的 `<h1>`。** 卡片被移除之後沒有「原本的位置」可以回；`<h1>` 是螢幕報讀最有意義的重新定位點。

### 不要做的事

- 不要在 `DownloadRowActions` 裡放抽屜或 `sheetOpen` state；不要給它 `onShowDetails`。
- 不要改桌機路徑的 JSX（`DropdownMenu`、`role="menuitem"`、`modal={false}`）；不要在桌機選單加「詳細資訊」。
- 不要複製 `stateActionFor`、`DownloadStatusPill`、確認框的文案、`formatters`；不要讀 `download.downloaded` 當進度。
- 不要用 hash 去查確認框要顯示的項目（用快照）；不要讓焦點落在 `<body>`。
- 不要在元件裡寫死時區；不要在元件測試裡 stub `matchMedia`（mock hook）。
- 不要改 `ui/Sheet.tsx`（4b-1 已定案）、`ui/Dialog.tsx`、`ui/mobileSheet.tsx`、`useDownloadActions.ts`、後端。
- 不要本機產 `-linux.png`；不要改舊 e2e／spec 的斷言（舊 e2e 只准加一個 viewport skip；三支 spec 只准加 hook mock）。

### 已知陷阱

- **`p-0` 會連 safe-area 的 padding 一起丟掉**——兩個抽屜的 className 都已經把它加回去（動作：在 Popup；詳細：在固定的動作列）。
- **Popup 自己是捲動容器**——詳細資訊要 `flex flex-col overflow-hidden`＋內層捲動區，動作列才會固定。
- **e2e 的輪詢 stub**：樂觀更新之後會**立刻**重抓——stub 以「有沒有收到 pause POST」決定回什麼，不要用次數。
- **`DownloadsBrowseV2.spec.tsx:226-235` 的全 true `matchMedia` stub**——靠檔頭的 hook mock 隔離；別「修」那個 stub。
- **`items` 每次重抓都是新陣列**——`sheet` 存 hash、每次 render `find`；找不到就關。
- **jsdom 看不到斷點**；行為由 e2e 守。`expect(box.x).toBe(0)` 遇到 `-0` → `Math.round`。
- **Pencil**：失敗會 rollback；存檔走選單 Save＋驗磁碟內容；只 stage 兩張圖。
- **gh 帳號會被別的 session 切走**：所有 `gh` 指令帶 `GH_TOKEN=…`。
- **行號以建單時為準**（2026-09-22，main `ae204761`）；`DownloadsBrowseV2.tsx` 的行號在 4b-1 合併後會位移。

### Source tree

```
ux-design.pen、_bmad-output/pen-tokens.json、screenshots/flow-d-downloads-v2/{d8,d9}-m-v2.png        ← Task 1
apps/web/src/components/downloads/DeleteWithFilesDialog.tsx（新；自 DownloadRowActions 抽出）          ← Task 2
apps/web/src/components/downloads/DownloadRowActions.tsx(+spec)                                      ← Task 2, 5
apps/web/src/components/downloads/formatters.ts(+spec)                                               ← Task 3
apps/web/src/components/downloads/DownloadActionsSheet.tsx（新，+spec）                               ← Task 4
apps/web/src/components/downloads/DownloadDetailSheet.tsx（新，+spec）                                ← Task 4
apps/web/src/components/downloads/DownloadCardV2.tsx(+spec)、DownloadsBrowseV2.tsx(+spec)             ← Task 5
apps/web/src/routes/test/-gallery.fixtures.tsx                                                       ← Task 6
tests/e2e/downloads-mobile.spec.ts（4b-1 建的，本張加測試）、tests/e2e/downloads-v2.spec.ts（只准一個 skip）  ← Task 6
tests/visual/…/downloads-mobile-sheets/{actions,detail}                                              ← Task 6
```

### Cross-Stack Split Check

後端 task **0 個**，前端／設計／測試 task 6 個（＋Task 0 確認依賴）→ 不觸發跨棧拆分。規模：3 個新元件＋3 個既有元件＋`formatters`、一張稿兩個畫面、兩個夾具、e2e 加四組——與 `dsr-6f-3` 同級。動作與詳細資訊共用同一個狀態機與焦點規則（AC #2），再拆會讓第二張一開工就改第一張剛寫好的 `DownloadsBrowseV2`——不再拆。

### Time-dependent visual coverage

- **N/A — no wall-clock-reading components touched.** `components/downloads/` grep `Date.now|new Date|performance.now` 零命中。本張新增的 `formatAddedOn` 用的是**帶參數**的 `new Date(iso)`——格式化一個固定的時間戳，不讀當下時間；`time-dependent-fixture-stability` 的 lint 明文不管它（rule 檔 `:14`）。夾具的 `addedOn` 是固定字串，基準線不隨日期變動（只隨**時區**，而 darwin／linux 基準線本來就分開）。

### References

- [Source: `apps/web/src/components/downloads/DownloadRowActions.tsx:1-3, 50-77, 87-221`；`DownloadCardV2.tsx:1, 23-26, 54, 59, 79, 84-100, 109, 118, 120-149`；`DownloadsBrowseV2.tsx:1-2, 88-124, 211-214, 244-273, 487-505`；`DownloadsTableV2.tsx:212-218`；`formatters.ts:76-96`；`downloadStatus.ts:20-27, 63`]
- [Source: `apps/web/src/hooks/useDownloadActions.ts:49-54`（樂觀更新）；`apps/web/src/services/downloadService.ts:44-66, 172-195`；`apps/api/internal/models/torrent.go:105, 203`（`addedOn` 是 UTC）；`apps/web/src/eslint-rules/time-dependent-fixture-stability.js:12-14, 113`]
- [Source: `apps/web/src/components/ui/Sheet.tsx`（4b-1 之後的版本）；`node_modules/@base-ui/react`（1.5.0）`dialog/popup/DialogPopup.d.ts:27-32`（`finalFocus` 的函式形式）、`dialog/root/DialogRoot.d.ts:32-44`；`ui/Dialog.tsx:27, 41-83`；`apps/web/src/test-setup.ts:80-90`]
- [Source: `DownloadRowActions.spec.tsx:35-131`；`DownloadsBrowseV2.spec.tsx:60, 226-246`；`DownloadsTableV2.spec.tsx:135-137`；`DownloadCardV2.spec.tsx:114-115`；`tests/e2e/downloads-v2.spec.ts:154-221`；`apps/web/src/routes/test/-gallery.fixtures.tsx:396-445, 1595-1776`]
- [Source: `ux-design.pen` `jDgxJ`／`aJFk0`／`J7QMBQ`／`C31QIb`／`KTyMD`／`TAE3d`／`KQi9B`／`i6jEmH`／`TVVAF`／`DrYXb`／`g4A3Qk`／`bhoYP`／`CwAjz`／`dC4Cg`／`KB5wY`／`Mpq5S`／`DrHEM`／`Wydrn`／`xshlR`／`lCFq2`／`ggTb3`／`Mz428` —— SM 建單時以 Pencil MCP 逐節點讀出（2026-09-22）；程式碼現況由唯讀稽核代理查證，建單後再經對抗驗證]
- [Source: `DESIGN.md:296-310`（狀態色、泥金只填滿要被按的控制項、數字中性）、`:351-360`（字階）、`:604-636`（手機規則、44×44）]
- [Source: `dsr-4b-1-mobile-sort-sheet-and-chip-row.md`（地基、夾具與 e2e 的寫法）；`dsr-4-flow-d-downloads-v2.md`（桌機的動作裁定）；`dsr-6f-1`…`dsr-6f-4`（手機 e2e 的量法）]
- [Source: project-context.md#Rule 16／#Rule 20／#Rule 21／#Rule 23／#Rule 24；`.claude/memory/feedback_split_oversized_stories.md`、`feedback_verify_pen_saved_before_commit.md`、`feedback_gh_token_explicit_for_pr_ops.md`]

## Dev Agent Record

### Agent Model Used

Claude Fable 5.1 — `claude-fable-5-1`（dev-story, Amelia，2026-09-22）

### Debug Log References

- Task 0：main `e8da8955`（4b-1 已合併，#502／#504）。4b-1 出貨的 API 與本張所寫的差異：`useIsPhone` 問的是 `(width < 40rem)` 不是 `639.98px`；`ui/Sheet` 另有 `description`／`descriptionClassName`；`<h1 ref={headingRef} tabIndex={-1}>` 4b-1 已加好，本張直接用；`DownloadsBrowseV2.spec.tsx` 檔頭的 `useIsPhone` mock 4b-1 已加好（`h.isPhone`）。
- Pencil：`problems` 68 → **68**（只減不增；`DrYXb` 唯一的 problem 是既有刻意的膠囊出血 `rZ3AX`）。`g4A3Qk` 改 `fit_content`（連 `bhoYP` 一起，否則 fill_container 循環）後實高 **606**，`y = 844 − 606 = 238`。存檔走選單 Save；磁碟檔 grep 到 `spec-note-dsr-4b-2`、`排序 sheet · 詳細資訊 sheet`；`KB5wY`／`dC4Cg` 0 次；`penSha256` 對得上。匯出 196/196，只留 `d8-m-v2.png`／`d9-m-v2.png`。
- 🔴 **焦點的坑（兩個，都在真瀏覽器才看得到）**：
  1. Base UI 的 return-focus 在 popup **卸載**那個 commit 的 layout-cleanup 同步解析目標、再用 microtask 去 focus。按「保留檔案」時 `onOpenChange(false)`＋樂觀移除是兩個 commit：抽屜先卸載（那一刻卡片還在 → 解析到 ⋯）、卡片再卸載 → microtask 打在游離的按鈕上 → 焦點掉到 `<body>`。unit（jsdom）看不出來，e2e 抓到。**解法**：抽屜永遠**先關再卸載**——`sheet` state 多一個 `open`，`onOpenChange(false)` 只把它翻 false，`onOpenChangeComplete(false)` 才清空 slot（有「換手時已被新的接管」的防護）；退場動畫跑完時卡片早已不在 → `isConnected` false → `<h1>`。兩個抽屜因此多一個 AC 沒列的 `onOpenChangeComplete` prop（純轉傳）。
  2. 中途試過用 ref 在 render 期間記錄 `items`（`itemsRef.current = items`）——`react-hooks` lint 直接擋（Cannot access refs during render），改用上面的做法後不需要。
- e2e 的「內容捲、動作列固定」一開始是**空轉綠**：390×844 下八格內容根本不到 85vh 的上限，什麼都不會捲。改成該條測試切到 390×667（iPhone SE 高度）＋長路徑，並斷言 `scrollHeight > clientHeight`、`scrollTop > 0`、抽屜貼底且 ≤ 85vh；mutation 才變紅。「Hash／路徑沒有橫向溢出」同理——40 字的 hash 在 350px 內本來就塞得下，改用長路徑才守得住 `break-all`。
- 本機 e2e 要 `AI_PROVIDER=claude`（`preexisting-fail-e2e-local-ai-provider`）。

### Completion Notes List

- Ultimate context engine analysis completed - comprehensive developer guide created（SM Bob，2026-09-22）
- **做了什麼（dev-story，2026-09-22）**
  - **稿（Task 1）**：D8-M `i6jEmH` → `folder-minus`；D9-M 刪 `dC4Cg`（技術徽章）與 `KB5wY`（檔名）、`Vvdar` 換成種子名稱樣式、`g4A3Qk` 改成包住內容並重新貼底；`ggTb3` 拿掉 design-ahead 那半句；規格註記 `spec-note-dsr-4b-2`（`i9Qn2`，4b-1 註記下方 40px，300×270）。
  - **確認框抽出（Task 2）**：`DeleteWithFilesDialog.tsx`（文案、版面、按鈕逐字照搬；`download` 是快照物件）；`DownloadRowActions` 桌機路徑改用它，16 條原樣綠；`stateActionFor`／`StateAction` export。
  - **formatters（Task 3）**：`formatDownloadMeta` 多 `downSpeed`／`upSpeed`（不帶箭頭；`down`／`up` 由它們組成，既有輸出逐字不變）；`formatAddedOn(iso, timeZone?)`（`en-CA`＋`hourCycle:'h23'`＋`formatToParts`，壞字串回 `—`）。
  - **兩個抽屜（Task 4）**：`DownloadActionsSheet`（標題＝種子名稱兩行、`status.label · 百分比` 用 `getDownloadTone().text`、四／三列、分隔線；2、3 按了就關，1、5 是換手只呼叫 callback）；`DownloadDetailSheet`（`flex flex-col overflow-hidden`＋內層捲動區、固定動作列；`<dl>` 八格、Hash mono＋`break-all`；主要鈕按了不關；completed 只有滿版「更多動作」）。
  - **接線（Task 5）**：`DownloadRowActions` 只多 `onOpenActions`——`isPhone && variant==='card' && !!onOpenActions` 時 ⋯ 是普通 button（`aria-haspopup="dialog"`），否則現有 JSX 原封不動；`DownloadCardV2` 透傳；`DownloadsBrowseV2` 擁有 `sheet`（聯集型別＋`snapshot`＋`open`）、`confirmTarget`（快照）、`triggerRef`、`handoffRef`；`finalFocus` 函式（換手回 `false`；否則 ⋯ 還在就 ⋯、不在就 `<h1>`；函式讀完就清 `handoffRef`——Base UI 在 popup 卸載時同步讀一次，用 effect 重設會太早）；`isPhone` 變 false 與項目消失都是「關閉」不是「卸載」。
  - **驗證（Task 6）**：夾具 `downloads-mobile-sheets/actions`（`jDgxJ`）／`detail`（`DrYXb`），選項用 `downloadFixture`；e2e 四條加進 `downloads-mobile.spec.ts`（stub 以「收到 pause POST」切換清單，不用次數）；舊 e2e `downloads-v2.spec.ts` ⋯ menu 那條加 viewport<640 skip（本張造成的手機 project 紅）。
- **測試（Rule 16：紅／守）**
  - 紅：`formatters.spec.ts` 5 條（新函式與新欄位，實作前紅）；`DownloadActionsSheet.spec.tsx` 11 條、`DownloadDetailSheet.spec.tsx` 9 條（新檔，實作前 import 失敗）；`DownloadRowActions.spec.tsx` 新增 3 條（`isPhone` mock true）；`DownloadCardV2.spec.tsx` 新增 1 條；`DownloadsBrowseV2.spec.tsx` 新增 9 條（`isPhone` mock true、渲染真的卡片）——其中「項目消失 → 焦點回 `<h1>`」實作第一版真的紅（見 Debug Log 坑 1）。
  - 守：`DownloadRowActions.spec.tsx` 16 條（抽出確認框前後皆綠、一字未改）、`DownloadsTableV2.spec.tsx` 8、`DownloadCardV2.spec.tsx` 13、`DownloadsBrowseV2.spec.tsx` 既有 22、`formatters.spec.ts` 既有 33——全部未改。三支 spec 只加了 hook mock（`DownloadsBrowseV2.spec.tsx` 的 4b-1 已加）。
- **Mutation check（拿掉 → 必須紅）：unit 23／23 紅、e2e 7／7 紅**（兩條 e2e 第一次是空轉綠，改測試後才紅——見 Debug Log）
  - formatters：`downSpeed` 帶箭頭、拿掉 NaN 防護 → 紅。**`hourCycle:'h23'` 換成 `hour12:false` → 綠**：Node 的 ICU 兩者午夜都給 `00`，「24:xx」是舊引擎的行為，jsdom 測不出來；照 AC 用 `h23`，誠實記錄。
  - RowActions：手機路徑無視 `variant`、沒傳 `onOpenActions` 也走抽屜、拿掉 `aria-haspopup` → 紅。Card：不透傳 → 紅。
  - ActionsSheet：狀態列不關、「連同檔案刪除」直接 `onRemove`、永遠畫「詳細資訊」、狀態行沒百分比 → 紅。
  - DetailSheet：「進度」讀 `downloaded`、速度帶箭頭、主要鈕關抽屜、拿掉 `break-all`、completed 仍有主要鈕 → 紅。
  - Browse：兩個抽屜同時開（detail 不看 kind）、刪除請求不先關抽屜、項目消失不關、`isPhone` false 不關、沒有 `<h1>` 後備、`finalFocus` 不用函式、抽屜讀快照不讀清單 → 紅。
  - e2e：換手時 `finalFocus` 不回 `false`（焦點被拉回卡片的 ⋯）、詳細資訊沒有內層捲動（動作列跟著捲走）、Hash／路徑沒 `break-all`、刪除請求不先關抽屜、關閉時直接卸載（焦點落 `<body>`）、沒有 `<h1>` 後備、手機仍是下拉 → 紅。
- **閘門**：`format:check` ✅；`lint:all` 0 errors／129 warnings（既有批次；本張碰的檔案 0 warnings）；`web:typecheck --skip-nx-cache` ✅；`check-design-tokens.py` ✅；`nx test web` **4049／4049**（279 檔）；`nx test api` ✅（已刪 `apps/api/coverage/`）；e2e `chromium` `downloads-mobile`（10 條）＋`downloads-v2`（8 條）`--repeat-each=3` **54／54**；`mobile-chrome` 6 passed／12 skipped（新 spec chromium-only、舊 spec 表格與 ⋯ menu 兩條被 viewport skip 擋下）；visual 整支跑三次：兩張新基準線第二、三次零差異，既有 `downloads-*` 零變動，唯一紅的是本機固定會漂的 `retry-retry-notifications`（`preexisting-fail-visual-darwin-three-stale-baselines`，CI 不受影響）。每次測試後 `test:cleanup` 無殘留。
- **/ship 對抗式 CR（2026-09-22，獨立 context 的審查代理；0 HIGH／3 MEDIUM／9 LOW＋2 推測）**：吸收 7 項（見上方 Review Follow-ups，Rule 24 ①），mutation：先關再開（把確認框改回同 commit 開 → 紅）、完成項目的標籤（→ 紅）；`handoffRef` 在開啟時清除是防禦性的、沒有可觀察的失敗情境（mutation 綠）。**沒有照做的**：
  - #6 `useIsPhone` 每次 render 都 `matchMedia()`、現在每張卡片都訂閱 → hook 是 4b-1 的，③ `disc-2026-09-useisphone-cache-mql`。
  - #7 `detail` 視覺基準線隨機器時區 → ③ `disc-2026-09-visual-project-pin-timezone`（改 `playwright.config.ts` 的 visual project，不是元件；本張不寫死時區）。
  - #10 class-token 斷言（`min-h-[52px]`、`break-all`）只守字串——它們是「守」，幾何由 e2e 量（`≥52`、`scrollWidth ≤ clientWidth`）；上方測試段落已如此標示。
  - #12 舊引擎忽略 `hourCycle` → 不在支援矩陣，記錄不做。
  - 範圍外：樂觀更新失敗沒有任何提示（卡片默默回來、焦點已在 `<h1>`）、對別的客戶端已刪掉的項目按「刪除檔案」→ 靜默 API 錯誤 → ③ `disc-2026-09-download-optimistic-failure-silent`（既有行為）。
  - CR 後重跑：unit 168／168、e2e `chromium` ×3 **57／57**（新增 reduced-motion 一條）、`mobile-chrome` 6 passed／13 skipped、visual 兩次：兩張新基準線零差異、既有零變動；六個閘門全綠（web 4049／4049）。
- 🔗 AC Drift: NONE (checked: `'menuitem\|連同檔案刪除\|onCloseAutoFocus\|移除（保留檔案）'` across _bmad-output/implementation-artifacts/*.md — 相關命中 3 處：`dsr-4` AC「一顆狀態鈕＋⋯ 選單、只有『連同檔案刪除』要確認」、`ux3-4-4` AC #5／#7「卡片與表格共用 `DownloadRowActions`」、`ux3-4-3b` AC3 卡片動作；全部 REUSE：桌機路徑的 JSX、`role="menuitem"`、確認框文案與 `onCloseAutoFocus` 逐字不變（16 條既有斷言原樣綠；確認框只是抽成獨立元件、DOM 相同）；手機多的是入口，不是改契約)
- 📎 Contract Stamps: NONE (本張不定義也不消費任何 `[@contract-v*]`；上游 `GET /downloads`、`POST …/pause|resume`、`DELETE …?deleteFiles=` 皆未 stamp＝implicit v0；對 4b-1 的依賴是元件 API 不是線上契約，差異已記在 Debug Log)
- 🎭 A11y Pre-Flight: PASS (5 components checked — `DownloadRowActions`、`DeleteWithFilesDialog`、`DownloadActionsSheet`、`DownloadDetailSheet`、`DownloadsBrowseV2`；0 jsx-a11y warnings on touched files, 0 introduced by this story。四類：① 圖片 N/A；② modal 焦點——兩個抽屜由 Base UI 鎖焦點、開啟時落在第一個可 tab 的列、關閉回卡片的 ⋯、卡片被移除就回 `<h1>`（unit 以 `fireEvent.click` 模擬 Safari、e2e 真點擊＋Esc＋刪除路徑各證一次）；換手時 `finalFocus` 回 `false`、下一個覆蓋層自己取焦點（e2e 斷言換手後焦點在新抽屜裡）；確認框走 Radix `onCloseAutoFocus` 同一套解析；③ aria-live N/A（狀態膠囊沿用卡片同一顆，本張沒有新的非同步揭露）；④ 自訂元件——⋯ 在手機帶 `aria-haspopup="dialog"`、進度條 `role="progressbar"`＋`aria-valuenow`、八格用 `<dl>`；已知缺口沿用 4b-1 已立的 `disc-2026-09-ui-sheet-no-close-button`)
- 🎨 UX Verification: PASS（對照表見下；落差都是建單裁定或已立單）

#### 🎨 UX 對照（dev-story Step 9；量測＝390×844 chromium 夜行，基準＝`.pen` 節點值）

| 區域 | 稿（節點） | 實作（量測） | 相符？ | 要修？ |
| --- | --- | --- | --- | --- |
| D8-M 抽屜 | `aJFk0` x0、寬 390、貼底、340 高、`$bg-secondary`、頂角 16、padding 上 8 下 20 | x 0、寬 390、底邊 844、**339** 高、bg-secondary、16px、pt 8／pb 20 | ✅ | — |
| D8-M 標題 | `pQtcH` BodyLg 16／600 `$text-primary`、左右 20 | 16px／600／text-primary、`px-5` | ✅ | — |
| D8-M 狀態行 | `rqbiJ`「下載中」Body 14／500＋`W2oZXn`「· 62.4%」`$accent-text` | 「下載中 · 62.4%」14px／500、accent-text（`getDownloadTone().text`） | ✅ | — |
| D8-M 分隔線／動作區 | `payIm` 1px；`C31QIb` padding [4,8]、列距 2 | 1px；`px-2 py-1`、`gap-0.5` | ✅ | — |
| D8-M 列 | h52、padding [0,12]、gap 12、`$radius-md`、圖示 20、BodyLg 500 | 52、pl 12、圖示 x 20 w 20、8px、16px／500、寬 374 | ✅ | — |
| D8-M 圖示 | `info`／`pause`／**`folder-minus`**（本張改稿）／`trash-2`；刪除列 `$error-text` | `Info`／`Pause`／`FolderMinus`／`Trash2`；刪除列 error-text | ✅ | — |
| D9-M 抽屜 | `g4A3Qk` 606 高（改稿後）、貼底、padding-top 8 | **538.5** 高、貼底、pt 8 | ≈ | 不修：差在標題行高（稿 H4 line 1.625、程式 `text-lg` 28px）與格子字的行高；結構、間距、字級全對 |
| D9-M 標題 | `Vvdar` H4 18／700 `$text-primary`、換行 | 18px／700／text-primary、`break-words` | ✅ | — |
| D9-M 狀態膠囊 | `ensI9` `$accent-tint` 藥丸 | `DownloadStatusPill`（同一顆） | ✅ | — |
| D9-M 進度列 | 軌道 h8 `$bg-tertiary`、填 `$accent-primary`、`pIPlp` BodyLg 600 mono `$accent-text` | 軌道 8、accent-primary、16px／600 mono、accent-text | ✅ | — |
| D9-M 小標 | `HxVqo` Label 12／600 `$text-secondary` | 12px／600／text-secondary | ✅ | — |
| D9-M 格子 | `Mpq5S` gap 12；標籤 Label `$text-secondary`、值 Body `$text-primary`；Hash／路徑整列 mono | gap 12；12px text-secondary；14px text-primary；Hash mono 12px 整列、路徑整列 | ≈ | 不修：路徑稿是 mono，實作用一般字＋`break-all`（AC #5 的規格就這樣寫；Hash 才 mono） |
| D9-M 拿掉的 | `dC4Cg` 技術徽章、`KB5wY` 檔名（本張改稿） | 沒有 | ✅ | — |
| D9-M 動作列 | `DrHEM` [12,20,24,20]、上框線 `$border-subtle`、固定在底；`Wydrn` h48 `$accent-primary` 滿版＋`xshlR` 48×48 `$bg-tertiary` | pt 12／pb 24／pl 20、1px border-subtle、固定（e2e 捲到底 y 不變）；主要鈕 48 高 290 寬 accent-primary＋⋯ 48×48 bg-tertiary | ✅ | — |
| 加入時間 | 稿 `2026-06-30 21:14` | `YYYY-MM-DD HH:mm`（瀏覽器時區） | ✅ | — |
| 把手 | `hmhlF`／`ZrhaS` 40×4 `$text-muted` | 40×4 `--border-subtle` | ❌ | 不在本張（`disc-2026-09-bottomsheet-grabber-three-variants`） |
| 桌機 | 不變 | 既有 `downloads-*` 基準線零變動、`menuitem` 11 處斷言原樣 | ✅ | — |

### Discovery Triage

<!-- Rule 24 — project-context.md. Any out-of-scope finding MUST land in exactly one lane with its
     sprint-status.yaml entry ID (② / ③) or absorbed AC # (①) BEFORE this story is marked done. -->

- **建單時的發現（SM Bob 2026-09-22）：** 與 `dsr-4b-1` 共用（已立三張 disc＋一則補記，見該檔）。本張自己的：`downloads-v2.spec.ts:183` 在手機 project 會因本張而紅 → ①（AC #7，加 viewport<640 skip）。
- **dev-story 期間的發現：** N/A — no out-of-scope work discovered。（Base UI return-focus 在「卸載」與「關閉」兩條路的時序差異，本張自己吸收——見 Debug Log；把手、關閉鈕、radiogroup 鍵盤三張單 4b-1 已立。）

### File List

- `ux-design.pen` — D8-M 圖示、D9-M 刪兩塊＋標題＋貼底、`ggTb3`、`spec-note-dsr-4b-2`
- `_bmad-output/pen-tokens.json` — `penSha256`
- `_bmad-output/screenshots/flow-d-downloads-v2/d8-m-v2.png`
- `_bmad-output/screenshots/flow-d-downloads-v2/d9-m-v2.png`
- `apps/web/src/components/downloads/DeleteWithFilesDialog.tsx`（新；自 `DownloadRowActions` 抽出）
- `apps/web/src/components/downloads/DownloadRowActions.tsx` — 用抽出的確認框、`stateActionFor` export、`onOpenActions` 手機路徑、檔頭
- `apps/web/src/components/downloads/DownloadRowActions.spec.tsx` — 檔頭 mock hook、新增 3 條
- `apps/web/src/components/downloads/formatters.ts` — `downSpeed`／`upSpeed`、`formatAddedOn`
- `apps/web/src/components/downloads/formatters.spec.ts` — 新增 5 條
- `apps/web/src/components/downloads/DownloadActionsSheet.tsx`（新）＋ `.spec.tsx`（新）
- `apps/web/src/components/downloads/DownloadDetailSheet.tsx`（新）＋ `.spec.tsx`（新）
- `apps/web/src/components/downloads/DownloadCardV2.tsx` — `onOpenActions` 透傳
- `apps/web/src/components/downloads/DownloadCardV2.spec.tsx` — 檔頭 mock hook、新增 1 條
- `apps/web/src/components/downloads/DownloadsBrowseV2.tsx` — 兩個抽屜＋確認框的狀態機、焦點、檔頭
- `apps/web/src/components/downloads/DownloadsBrowseV2.spec.tsx` — 新增 9 條
- `apps/web/src/routes/test/-gallery.fixtures.tsx` — 夾具 `downloads-mobile-sheets/actions`、`/detail`
- `tests/visual/components.visual.spec.ts-snapshots/components/downloads-mobile-sheets/{actions,detail}/default-visual-darwin.png`（新；`-linux` 由 CI bootstrap PR 補）
- `tests/e2e/downloads-mobile.spec.ts` — 新增第二個 describe（4 條）
- `tests/e2e/downloads-v2.spec.ts` — ⋯ menu 那一條加 viewport<640 skip（只有這一處）
- `_bmad-output/implementation-artifacts/sprint-status.yaml` — 狀態
- `_bmad-output/implementation-artifacts/dsr-4b-2-mobile-actions-and-detail-sheets.md` — 本檔

## Change Log

- 2026-09-22 — 建單（SM Bob，create-story；main `ae204761`）。⚖️ Alexyu 當場裁定詳細資訊抽屜要做。原本與 `-1` 是同一張 `dsr-4b`；建單後對抗驗證（2 CRITICAL＋12 SHOULD FIX＋9 NIT）**全部併入**並拆單。併入本張的重點：① 原稿的 AC #3 與 AC #5 對「誰擁有抽屜」互相矛盾，照字面做會生出兩個抽屜、兩條確認路徑 → 改成頁面擁有一個聯集型別的 `sheet` state，`DownloadRowActions` 只多一個 `onOpenActions`；② 移除是樂觀更新，卡片當下就卸載 → `finalFocus` 用函式、後備到 `<h1>`、確認框存物件快照；換手時回 `false` 取代「等一個 frame」；③ `formatDownloadMeta` 的速度帶箭頭、`size` 是 `progress×size`（讀 `downloaded` 會跟卡片矛盾）→ 明確指名函式並加不帶箭頭的欄位；④ 狀態行的顏色要用 `getDownloadTone`，不是 descriptor 的膠囊 class；⑤ 暫停是樂觀更新不是「等下一次輪詢」→ e2e 的 stub 以「收到 POST」切換；⑥ 詳細資訊要 `flex flex-col overflow-hidden`＋內層捲動，動作列才固定；⑦ `hourCycle: 'h23'`；⑧ 夾具 `penNode` 必填。
- 2026-09-22 — Task 0／1（dev-story, Amelia）：確認 4b-1 已合併（main `e8da8955`）；D8-M `folder-minus`、D9-M 刪檔名與技術徽章＋貼底、Flow 說明、`spec-note-dsr-4b-2`；只 stage 兩張圖＋`pen-tokens.json`。
- 2026-09-22 — Task 2／3：確認框抽成 `DeleteWithFilesDialog`（16 條原樣綠）、`stateActionFor` export；`formatDownloadMeta` 加 `downSpeed`／`upSpeed`、新 `formatAddedOn`。
- 2026-09-22 — Task 4：`DownloadActionsSheet`、`DownloadDetailSheet`＋各自 spec（20 條）。
- 2026-09-22 — Task 5：`DownloadRowActions.onOpenActions`（桌機 JSX 不變）、`DownloadCardV2` 透傳、`DownloadsBrowseV2` 狀態機（`sheet`＋`snapshot`＋`open`、`confirmTarget` 快照、函式版 `finalFocus`、換手）；三支 spec 的 hook mock。真瀏覽器抓到「卸載時回焦點打在游離按鈕」→ 改成先關再卸載（`onOpenChangeComplete` 才清 slot）。
- 2026-09-22 — Task 6：兩個夾具＋darwin 基準線（三次一致）、手機 e2e 四條（`--repeat-each=3` 54/54）、舊 e2e ⋯ menu 那條 viewport skip、mutation unit 23/23＋e2e 7/7 紅（兩條 e2e 先修成真的會紅）、Step 9 對照表；Status → review。
- 2026-09-22 — /ship 對抗式 CR：吸收 7 項（確認框與換手改成真的先關再開、桌機確認框跨斷點不殘留、e2e 拿掉兩個 `waitForTimeout`、完成項目的「更多動作」標籤、`handoffRef` 開啟時清除、`formatAddedOn` 快取、reduced-motion e2e＋註解），另立 `disc-2026-09-useisphone-cache-mql`、`disc-2026-09-visual-project-pin-timezone`、`disc-2026-09-download-optimistic-failure-silent`。
- 2026-09-22 — 合併：PR #505（squash `b803fd89`），`-linux` 基準線由 bootstrap PR #506 補上；PR 上 CI 全綠。Status → done。整個 `dsr-4b`（Flow D 手機抽屜）兩張子單都合併，傘狀條目結案。
