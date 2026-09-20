# Story DSR.6f-1：手機上的字幕對話框從底部滑上來，「管理字幕」與生成進度在手機上對齊設計稿

Status: review

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As the person who opens 管理字幕 on a phone,
I want 對話框像一張從底部滑上來的抽屜、關閉鈕按得到、「生成字幕」是一整條好按的按鈕,
so that 我用拇指就能操作，而且畫面告訴我它是從哪裡來的，不是憑空在螢幕底部淡出來。

## Context

`dsr-6f`（Flow F 全部手機稿）拆出來的**第一塊**。依賴的桌機四張（`dsr-6b`／`6c`／`6d`／`6e`）全部 done。

⚖️ **SM 裁定（2026-09-20，拆單）**：八張手機稿、五個元件、一個全新的共用機制（由下往上的進場動畫）、外加「視覺測試今天根本拍不到手機版的對話框」這個前提工程——一張做完會超過 `dsr-6e-1`。依 `feedback_split_oversized_stories`「該拆就拆」，**切分線選版面**，而且照 BMAD 慣例**一次只建一張**（後面幾張等前一張做完再建，才吃得到前一張的教訓）：

| 單子 | 範圍 | 狀態 |
| --- | --- | --- |
| **`dsr-6f-1`（本張）** | **地基**：手機 sheet 共用外殼（把手、由下往上的動畫、44×44 關閉鈕、safe-area）＋視覺測試支援手機 viewport；**管理字幕** F1-M-v2 `JkdfH`、**生成進度** F3-M-v2 `k8sJl4` → `ManageSubtitleDialogV2.tsx`、`GenerationProgressV2.tsx` | 本張 |
| `dsr-6f-2` | 名詞對照表 F6-M-v2 `buepS`（今天在手機上是**置中對話框**不是 sheet；列在 390 寬會溢出；所有列都要有「編輯」） | backlog（稽核結果已寫進 sprint 條目） |
| `dsr-6f-3` | 批次 F8-M-v2 `H717g`＋同意流程 F15-M／F16-M／F19-M（頁尾、safe-area、`confirm-mobile` 夾具） | backlog（同上） |
| `dsr-6f-4` | 生成工作區 F11-M-v2 `PXB0z`（是一頁不是 sheet：返回鍵、總覽卡、事件紀錄在手機要能收合） | backlog（同上） |

⚠️ **驗收基準是 `.pen` 節點值**。PNG 只是參考。
⛔ **本張不進** `GlossaryPanelV2`／`GlossaryRowV2`（6f-2）、`GenerationWorkspaceV2`（6f-4）；`GenerationBatchDialogV2`／`GenerationConsentView`／`ConfirmGenerationDialog` **只換外殼常數**（AC #3），內容排版歸 6f-3。

### 🔴 建單時查到的事（main `6e09d4bc`；行號皆為現況）

**共用外殼**

1. **手機上的「抽屜」是原地淡入放大，不是滑上來。** 四個對話框（管理字幕、批次、同意、確認）在手機上都已經貼底、上緣圓角，但進場動畫跟桌機一樣是 `dialog-enter`（`opacity`＋`scale 0.96→1`，`styles.css:378-397`）——一張貼在螢幕底部的面板原地放大 4%，看起來像畫面抖了一下。`styles.css` 裡沒有任何由下往上的 keyframe。唯一真的會滑上來的是 `ui/Sheet.tsx`（Base UI，不是 Radix），只有媒體庫篩選與「更多」在用。
2. **同一串 class 複製了四份。** 手機外殼字串（`bottom-0 left-0 right-0 top-auto w-full max-w-none translate-x-0 translate-y-0 rounded-b-none rounded-t-[var(--radius-xl)]`）逐字出現在 `ManageSubtitleDialogV2.tsx:466-477`、`GenerationBatchDialogV2.tsx:431-450`、`GenerationConsentView.tsx:519-531`、`ConfirmGenerationDialog.tsx:82-92`。
3. **管理字幕沒有把手。** 另外三個都有（36×4、`--bg-tertiary`、`sm:hidden`），管理字幕漏了。
4. **關閉鈕在手機上只有 16×16。** `ui/Dialog.tsx:66-69` 的 ✕ 是 `absolute right-4 top-4`、`size-4`、沒有 padding。手機稿上沒有頁尾也沒有「關閉」（F1-M），✕ 是唯一的按鈕式出口，稿是 44×44（`XAbaF`）。全 app 的對話框共用這顆，所以既有單 `disc-2026-09-dialog-close-target-44px` 一直沒人敢動。
5. **只有確認框有 safe-area——而且它今天其實沒有作用。** `ConfirmGenerationDialog.tsx:207` 有 `pb-[max(0.875rem,env(safe-area-inset-bottom))]`，管理字幕沒有。但 `apps/web/index.html:8` 的 viewport meta 沒有 `viewport-fit=cover`，所以 iOS 上 `env(safe-area-inset-bottom)` 恆為 0（沒有 `cover` 時 Safari 本來就把頁面留在安全區內）——「home indicator 會壓到按鈕」**未經查證、多半不成立**。本張照樣補上 class（為將來開 `cover` 預留、與確認框一致），但不翻那個 meta（全站的事，另立單子）。
6. **視覺測試拍不到手機版。** `visual` project 把 viewport 釘死在 1280×800（`playwright.config.ts:149-165`），`GalleryFixture` 只有 `width`（包一層定寬盒子），沒有 `viewport`。對話框走 Portal、看的是**視窗**的 `sm:`，所以今天沒有任何 sheet 能在手機寬度留基準線。兩個 `list-mobile*` 夾具能成立，只是因為那一列用的是 container query。

**管理字幕 F1-M-v2**

7. **外框**：標題列 `h-14 border-b pl-6 pr-12`（`:479`）；稿 `CGvIz` 高 44（由 44 的關閉鈕撐開）、**沒有底線**、padding [0,4,0,16]。內容區 `px-6 py-5 gap-6`（`:494`）；稿 `PPLQr` [6,16,16,16] gap 14。
8. **「生成字幕」不是滿版。** `:636-681`（排版 class 在 `:639`）是 `flex items-center gap-4`（按鈕在左、說明在右）；稿 `zLRkb` 是直排置中 gap 6：按鈕 `k0H8wR` **滿版**、padding [12,20]，說明 `qR6hi` 在下方置中。
9. **手機稿沒有頁尾。** 程式碼在所有寬度都有頁尾（`:783`：左邊「搜尋線上字幕（成功率低）」、右邊「關閉」）；稿把「搜尋線上字幕」放在內容最下面置中（`V5LMv`，h44、Label `$text-secondary`），沒有「關閉」。
10. **軌道列**：程式碼一行（藥丸＋來源）；稿兩行（來源／檔名）。檔名在桌機稿就註明是 aspirational（F1-D `peX3f`：「待 production-countries-detail-api 接線」），程式碼沒有檔名可顯示；「轉為繁中（簡轉繁）」`kfQ76` 也沒接（`disc-2026-09-dialog-track-convert-not-wired`）。兩者都不是手機的事。
11. **稿的「已生成」顏色不一致**：F1-M `un8dT` 是 `$accent-text`，F1-D `dYrRl` 是 `$text-secondary`。泥金＝正在跑；「已生成」是來源標籤不是進行中 → 手機稿錯。
12. **Rule 21 標頭**已列 F1-M，只缺 F3-M。

**生成進度 F3-M-v2**

13. **稿上畫著系統給不出來的東西**：`E10kib`「45%」掛在「轉錄中」（轉錄沒有百分比，dsr-6b／6d-b 已裁定百分比只在翻譯階段）；`NPtOO`「本次用量：$0.42 / 上限 $5.00」（單片對話框從來不傳 `costUsedText`，程式碼不畫；桌機 F3-D 也沒有）。
14. **稿少了一句**：桌機 F3-D 在步驟條下面有階段說明 `OQ2vp`「正在轉錄音訊」，程式碼在所有寬度都會畫 `message`；手機稿沒有。
15. **階段標籤 13px**：`GenerationProgressV2.tsx:185` `text-[13px] sm:text-xs`，註解寫「stays 13px — the mobile sheet is dsr-6f」；稿 `IdGB2` 等是 Body 14。百分比 `:195` 是 `text-[11px]`（凍結，不動）。
16. **「關閉」不是滿版、提示位置不對**：程式碼頁尾是「提示在左、關閉在右」；稿 `uHnRS` 是滿版 Secondary、padding [12,20]，提示 `wupMI` 在**按鈕下方置中**。
17. **稿的 SSE 膠囊是 `$radius-sm`**（`Vsxtz`），DESIGN.md 規定徽章是藥丸——但它緊鄰凍結的 11px，桌機也還沒改（dsr-6d-b /ship CR L16 交代）；全 Flow F 一起處理，歸 `dsr-6f-3`。

### 設計稿節點

| 代號 | 節點 | 本張要看的 |
| --- | --- | --- |
| F1-M-v2 | `JkdfH`（sheet `FtarQ` 390、`$bg-secondary`、`$border-subtle` 1px、上緣 `$radius-xl`） | 把手 `uCZyP`（padding [8,0,4,0]）／`j53Qt`（36×4、`$bg-tertiary`、pill）；標題列 `CGvIz`（[0,4,0,16]、**無底線**）：`M9n9fJ` BodyLg 600、集數碼 `CvLdG` Body 600、關閉 `XAbaF` 44×44／`ELx15` 18 `$text-secondary`；內容 `PPLQr` [6,16,16,16] gap 14；`oUbta`「現有字幕」Body 600 `$text-secondary`；軌道列 `wWXOB`／`cvrMd`／`K6IRZe`（[10,12] gap 10 `$bg-tertiary` `$radius-md`；藥丸 [4,10] pill；來源 Label）；生成區 `zLRkb`（直排置中 gap 6、padding-top 4）：`k0H8wR` ButtonCost **滿版** [12,20]、`qR6hi` Label `$text-muted`；名詞表列 `E2Wjv`（h44、[0,4] gap 8）；`V5LMv`（h44 置中）／`WOEbf` Label `$text-secondary` |
| F3-M-v2 | `k8sJl4`（sheet `Me1fR`） | 標題列 `B9gEpS`（**有**底線）：`p3uMy2`「生成字幕 — 怪奇物語」；內容 `tRFbG` [6,16,16,16] gap 14；直排步驟 `fS5is` gap 6：每列 padding 4 gap 10、圓點 22、標籤 Body（進行中 600 `$accent-text`、完成 `$text-secondary`、未到 `$text-muted`）、百分比 Label `$accent-text` 靠右；SSE `Vsxtz`；關閉 `uHnRS` 滿版 Secondary [12,20]；提示 `wupMI` Label `$text-secondary` 置中 |
| 對照 | F1-D-v2 `r1EY9`（`dYrRl`、`peX3f`、`W3o1f9`「手機版行為相同，不另出圖」）、F3-D-v2 `JbXai`（`OQ2vp`、`NmhL0` 的 `fVBQF:{enabled:false}`） | 桌機已裁定的事 |
| 母版 | `Component/BottomSheet` `SG1ln`（把手 `XQMd0` 36×4 **`$text-muted`**、說明「最大高度是螢幕的 80%」）、`Component/ButtonCost/Default` `qAERt`、`Component/Button/Secondary` `YDPhc` | 見建單裁定 3 |

字階（`DESIGN.md:355-358`）：BodyLg 16、Body 14、Label 12。

---

## Acceptance Criteria

1. **設計稿先改（Pencil MCP；動手前每個節點再 `Get` 一次）。**
   - F3-M：`E10kib`（45%）`enabled:false`；`NPtOO`（本次用量）`enabled:false`；在 `fS5is` 之後補上階段說明——`Copy` F3-D 的 `OQ2vp` 進 `tRFbG`、`Move` 到 index 1，`width: fill_container`＋`textGrowth: fixed-width`＋`textAlign: center`。
   - F1-M：`un8dT` `fill` → `$text-secondary`（🔴 #11）。
   - ⛔ **不改母版 `SG1ln`**（建單裁定 3：把手在整份稿裡有三種畫法，不是「只有母版不一樣」——另立單子）。
   - 在規格群組 `JzmvC` 新增一個中性註記（比照 `gwS55`／`bSlYR`：`$Type/Label/*`、`fill: $text-muted`；`bSlYR` 底在 y=35121、它下方 x=17040 沒有東西 → 放在 y=35161、同寬）：「Flow F 手機 sheet：由下往上滑入（reduced-motion 時直接出現），最高 85vh、內容內部捲動；關閉鈕 44×44。F1-M 沒有頁尾——關閉靠 ✕、點遮罩、Esc；『搜尋線上字幕』在內容最下面。失敗（F4）與生成中（F3）在手機保留底部的動作。F1-M 的檔名與『轉為繁中』同桌機，尚未接線。F3-M 的百分比只有翻譯階段有；單片對話框不顯示用量。F2／F4／F5／F10 沒有手機稿，手機沿用 F1-M／F3-M 的外殼。」
   - `ctx.problems`；存檔並 grep 磁碟檔確認落盤；匯出後只 stage `flow-f-subtitle-v2/{f1-m-v2,f3-m-v2}.png`（＋`pen-tokens.json`），其餘還原。**`design-system/component-library.png` 若變了，代表動到了母版——回頭檢查。**

2. **視覺測試支援手機 viewport（🔴 #6；先做，後面的 AC 才有東西可驗）。**
   - 夾具清單是**從 DOM 刮出來的**（spec `:164-174` 用 `evaluateAll` 讀 `li[data-gallery-id]` 與 `data-gallery-clock-time`；`gallery.tsx:273` 畫那個 `<li>`）。所以：
     - `GalleryFixture`（`-gallery.fixtures.tsx:396-466`）加 `viewport?: { width: number; height: number }`；`width` 與 `viewport` **互斥**（同時給就是寫錯，型別上用 union 或在 gallery 裡 throw）。
     - `gallery.tsx` 的 manifest `<li>` 加 `data-gallery-viewport={fx.viewport ? `${fx.viewport.width}x${fx.viewport.height}` : undefined}`。
     - spec：同一個 `evaluateAll` 把它讀出來；`const DEFAULT_VP = { width: 1280, height: 800 }`；在 `clockTime` 那個分支**之前**：`const want = parsed ?? DEFAULT_VP; const cur = page.viewportSize(); if (!cur || cur.width !== want.width || cur.height !== want.height) await page.setViewportSize(want);`——**只在真的不一樣時才呼叫**，既有三百個夾具一次多餘的 `setDeviceMetricsOverride` 都不會有。放在 `page.goto` 之前（版面在導頁之後才算，所以重設不會動到下一個夾具的基準線）。
   - 已查證在 390 寬下不會壞的事：`component-gallery-page` 是一般 block、仍可見；殼層的 `<header>` 還在，所以 `UNSTICK_SHELL_HEADER_CSS` 照樣套用且無害；`MobileTabBar` 是 `z-40`、在對話框的 `z-50` 遮罩之下；截圖路徑是明寫的、`VISUAL_BUCKETS` 分桶不受影響（每個桶有自己的 page，迴圈內重設就夠）。Portal 夾具拍的是**視窗**（`expect(page)`，`:318`），所以圖會是 390×844。
   - 手機夾具一律 `viewport: { width: 390, height: 844 }`。
   - 先用一個**不改任何產品程式碼**的夾具證明機制可用：`generation-consent/confirm-mobile`（`ConfirmGenerationDialog`，props 同 `f16-model-default`）——它本來就是 `dsr-6e-2` 交代要補的；排版對不對是 6f-3 的事，本張只要它**真的拍到 sheet**（貼底、有把手）。
   - **既有 1280 的基準線一張都不能變**——跑完整支 visual，`git status` 只能多出新夾具。

3. **共用外殼（🔴 #1–#5）。**
   - 新增 `apps/web/src/components/ui/mobileSheet.tsx`（第一行 `// Implements: Component/BottomSheet (SG1ln)`）：
     - `MOBILE_SHEET_CONTENT`：今天那串手機外殼 class（逐字）＋ `max-sm:data-[state=open]:animate-sheet-enter max-sm:data-[state=closed]:animate-sheet-exit`。
     - `MOBILE_SHEET_CLOSE`：`max-sm:right-1 max-sm:flex max-sm:h-11 max-sm:w-11 max-sm:items-center max-sm:justify-center max-sm:opacity-100 max-sm:text-[var(--text-secondary)] max-sm:[&_svg]:size-[18px]`。⛔ **不含 `top`**——四個對話框的標題列高度不一樣，一個固定的 `top` 置不了中：把手區高 16（8＋4＋4）；管理字幕的標題列在本張變成 44 → ✕ 要 `max-sm:top-4`（16）；批次、同意、確認三個在手機仍是 `h-14`（56）→ `max-sm:top-[22px]`（到 6f-3 改它們的標題列為止）。各對話框自己 `cn(MOBILE_SHEET_CLOSE, 'max-sm:top-…')`。（那三個今天其實就是歪的：✕ 中心 y=24、標題中心 y=44。）標題列的 `pr-12`（48）剛好等於 `right-1`（4）＋44，零餘裕但不重疊，不用改。
     - `SheetGrabber(props: React.ComponentProps<'div'>)`：`<div {...props} className={cn('flex shrink-0 justify-center pb-1 pt-2 sm:hidden', props.className)}><span aria-hidden="true" className="h-1 w-9 rounded-full bg-[var(--bg-tertiary)]" /></div>`——**一定要把 props 展開到外層**：批次的把手帶著 `data-testid="gen-batch-drag-handle"`（`GenerationBatchDialogV2.tsx:455`），`GenerationBatchDialogV2.spec.tsx:253-255` 在斷言它。
   - `styles.css`：新增 `@keyframes sheet-enter`（`from { translate: 0 100%; } to { translate: 0 0; }`）與 `sheet-exit`（反向），在 `@theme` 區塊（`:452-462`）註冊 `--animate-sheet-enter: sheet-enter var(--motion-move) var(--ease-settle) both;`／`--animate-sheet-exit: sheet-exit var(--motion-leave) var(--ease-leave) both;`。
     - 🚨 **進場與退場的 animation-name 不能相同**（`styles.css:350-356` 的 ⚠️）。
     - 動 **`translate`**（個別屬性），不是 `transform`；而且**只在 `max-sm:`**——`sm:` 以上對話框靠 `translate` 置中。
     - **不淡入**：抽屜是滑進來的實體，遮罩自己會淡入。
     - reduced-motion：這個 keyframe 用的是寫死的 `100%`、不是 `--motion-rise`，所以**距離不會歸零**，保護它的只有「時長收成 1ms」（`styles.css:300-313`、`:525-534`）——結果一樣是直接出現，但不要在註解裡寫成「距離歸零」。
     - 已查證（用 repo 的 `@tailwindcss/node` 4.1.18 實際編譯）：`max-sm:` 是 v4 內建（`@media (width < 40rem)`，與 `sm:` 正好互補；這是全 repo 第一次用）；Tailwind 把 `max-sm:*` 規則排在基底工具與 `data-[state=open]:animate-dialog-enter` 之後、權重相同，所以手機規則會贏；twMerge 3.4.0 只會丟掉「修飾詞完全相同」的衝突 class，所以 `right-4`＋`max-sm:right-1`、`opacity-70`＋`max-sm:opacity-100`、兩個 animate class 都會留著。
   - `ui/Dialog.tsx`：`DialogContent` 加**可選**的 `closeClassName?: string`，`cn()` 併到 ✕（`:66`）。🚨 **要在 `overlayClassName` 旁邊（`:37`）解構出來**——`DialogContent` 會把 `...props` 展開到 `DialogPrimitive.Content`（`:63`），沒解構就會變成 DOM 上的 `closeclassname` 屬性。⛔ 預設 `undefined`：`cn(base, undefined)` 回傳的字串與今天逐位元相同，不傳的對話框一個像素都不變。
   - 四個對話框改用常數與元件：`ManageSubtitleDialogV2`（**補上把手**）、`GenerationBatchDialogV2`、`GenerationConsentView`、`ConfirmGenerationDialog`——手機外殼字串 → `MOBILE_SHEET_CONTENT`、把手 → `<SheetGrabber />`（批次那顆保留它的 testid）、`closeClassName`。⛔ 後三個**只換這三樣**，`sm:` 那半段與所有內容不動。
   - 刪除確認（`GlossaryRowV2.tsx:212`）與名詞表（`GlossaryPanelV2.tsx:317`）**不動**（6f-2）。

4. **管理字幕在手機上對稿（🔴 #7–#10、#12；一律 `max-sm:`，桌機一個像素都不能變）。**
   - 標題列 `:479`：加 `max-sm:h-11 max-sm:pl-4`；底線依狀態——閒置 `max-sm:border-b-0`（F1-M `CGvIz` 無底線），生成中／失敗保留（F3-M `B9gEpS` 有）。集數碼 `:486` 加 `max-sm:text-sm`（稿 `CvLdG` Body 14；桌機 BodyLg）。
   - 內容區 `:494`：`max-sm:px-4 max-sm:pt-1.5 max-sm:gap-3.5`；**底部 padding 只寫一次**：閒置（手機沒有頁尾）`max-sm:pb-[max(1rem,env(safe-area-inset-bottom))]`，其他狀態 `max-sm:pb-4`。
   - 軌道列 `:388`：`max-sm:px-3 max-sm:py-2.5 max-sm:gap-2.5`（稿 [10,12] gap 10）。結構不改（一行：藥丸＋來源；稿的第二行是檔名，程式碼沒有這個資料）。「現有字幕」區塊內部的 `gap-2.5`（`:382`）不動——稿是把標題與各列平鋪在 gap 14 的內容區裡，程式碼多包一層，手機上差 4px，不值得為此改結構（刻意不對齊）。
   - 生成區（`:636-681`，排版 class 在 `:639`）：`tracks.length > 0` 時加 `max-sm:flex-col max-sm:items-stretch max-sm:gap-1.5 max-sm:pt-1`，`ButtonCost` 用它**既有的** `className` prop（`ButtonCost.tsx:44`）傳 `max-sm:w-full max-sm:py-3`（基底已是 `justify-center`，不用再加），說明 `max-sm:text-center`。**沒有軌道（F2）時**那一區本來就是直排置中（`:636-642`）——手機上同樣讓按鈕滿版（F2 沒有手機稿，沿用 F1-M 的 `zLRkb`；寫進 Completion Notes）。
   - **頁尾（`:783`，是同一個 div、子節點依狀態切換）**——令 `inProgressView` ＝ 生成中／完成／失敗的那個分支：
     - `!inProgressView`（閒置、F2 無軌道、F5、載入中 F10、`triggerError`、分集閒置）→ 頁尾 `max-sm:hidden`（建單裁定 4）。這些狀態的頁尾只有「搜尋線上字幕」與「關閉」兩樣，前者手機另有一顆、後者由 44×44 的 ✕ 取代。⚠️ `triggerError`（觸發失敗）若頁尾裡有帶金額的「重試」，它算 `inProgressView` 那一邊——**動手前逐一列出每個狀態頁尾裡有什麼，任何付費按鈕所在的狀態都不准藏**，結果寫進 Completion Notes。
     - 手機的「搜尋線上字幕（成功率低）」：畫在 `open-glossary` 那一列**之後**、`{fetchOpen && …}` 區塊**之前**（面板才會開在它的觸發鈕下面）：`sm:hidden flex min-h-[44px] w-full items-center justify-center text-xs text-[var(--text-secondary)]`，同一個 handler，`data-testid="toggle-fetch-mobile"`。分集今天就不顯示桌機那顆，手機也不顯示。（已查證：既有測試與 TestSprite TC071–073／077 都用 `getByTestId('toggle-fetch')`，多一顆同文字的按鈕不會撞；`display:none` 的那顆不在無障礙樹與 tab 順序裡。）
   - Rule 21：第一行已經是 `// Design ref:` 而且已列 `Screen F1-M-v2 (JkdfH)`，只缺 F3-M——在那一行後面接 ` + Screen F3-M-v2 (k8sJl4)`（文法是 ` + Screen …`，不是 ` · `）。

5. **生成進度在手機上對稿（🔴 #13–#16）。**
   - `GenerationProgressV2.tsx:185`：`text-[13px] sm:text-xs sm:leading-normal` → `text-sm sm:text-xs sm:leading-normal`；拿掉「stays 13px — the mobile sheet is dsr-6f」那句註解。每列 `:179` 加 `max-sm:p-1`（`gap-2.5` 已經在）。⛔ `:195` 的 `text-[11px]`、`GENERATION_STAGES` 都不動（凍結）。
     - ⚠️ **這個元件是共用的**：批次（`GenerationBatchDialogV2.tsx:219`）與工作區（`GenerationWorkspaceV2.tsx:301`）在手機上也會跟著變成 14px＋4px 內距。這是要的（F8-M／F11-M 的方向一致、兩者都沒有手機基準線），已寫進 6f-3／6f-4 的 sprint 條目。
     - `GenerationProgressV2.spec.tsx:147-165` 今天在斷言 `text-[13px]`——那條要**改**，不是只加新的。
   - 生成中／完成的頁尾在手機依稿搬成「內容區的延伸」（稿 `Me1fR` 根本沒有頁尾，`uHnRS`／`wupMI` 在 body 裡、左右 16、沒有上框線）：
     - 頁尾 div：`cn(base, !inProgressView && 'max-sm:hidden', inProgressView && !runFailed && 'max-sm:flex-col-reverse max-sm:items-stretch max-sm:border-t-0 max-sm:px-4 max-sm:pt-0', inProgressView && 'max-sm:pb-[max(0.75rem,env(safe-area-inset-bottom))]')`。
     - 🚨 包著按鈕的那層（`:814`，`ml-auto flex shrink-0 …`）要加 `!runFailed && 'max-sm:ml-0 max-sm:w-full'`——`ml-auto` 會取消 `align-items: stretch`，不拿掉的話 `w-full` 是對著縮起來的外層算，按鈕不會滿版。按鈕本身 `!runFailed && 'max-sm:w-full max-sm:justify-center'`（`min-h-[44px]` 已給高度）。
     - 提示 `<span>`（`:807-811`）在完成時內容是 `''`——直排之後會多出一條 12px 的空隙，**空的時候改回傳 `null`**；有內容時 `max-sm:text-center`。
     - 失敗（F4）：頁尾維持橫排（「稍後再試」＋帶金額的「重試」，`basis-full` 的提示在上），只加 safe-area 那一條。
   - 百分比仍由 `percentage` 決定（只有翻譯階段會有）；不要因為稿關掉了 45% 就把程式碼的百分比拿掉。

6. **既有的行為不准回歸。**
   - 桌機（≥640）：四個對話框的版面與既有 1280 基準線**完全不變**。
   - 關閉的三條路都還在：✕、點遮罩、Esc；生成中關閉只是停止觀看。
   - 巢狀的名詞對照表仍能從管理字幕裡打開、關掉之後焦點回到「名詞對照表」那一列。
   - `ManageSubtitleDialogV2.spec.tsx` 既有測試全留；`toggle-fetch` 不改名。
   - ⛔ 不處理 `dsr-6e-2` 交代的 L5（取消鈕在關閉動畫期間閃回「取消」）——它在 6f-3。

7. **測試。** 下面分「**紅**」（現在會失敗）與「**守**」（現在就會過，用來擋回歸）——不要把守門測試當成紅測試交差。
   - `ui/mobileSheet.spec.tsx`（新，紅）：把 `MOBILE_SHEET_CONTENT` 拆成 class token 比對——含 `max-sm:data-[state=open]:animate-sheet-enter`、`max-sm:data-[state=closed]:animate-sheet-exit`，而且**沒有任何不帶 `max-sm:` 的 sheet 動畫 token**；`MOBILE_SHEET_CLOSE` 沒有任何 `top` token；`SheetGrabber` 外層有 `sm:hidden`、內層 `<span>` 有 `aria-hidden="true"`、傳進去的 `data-testid` 落在外層。
   - `ui/Dialog.spec.tsx`（**不存在，要新增**）：（守）不傳 `closeClassName` 時 ✕ 的 `className` 與今天逐字相同；（紅）傳了就併進去；（紅）content 元素上**沒有** `closeclassname` 屬性。
   - `apps/web/src/styles-motion.spec.ts`（**已存在**——擴充 `:240-243` 那份清單）：（紅）`sheet-enter`／`sheet-exit` 都在 `@theme` 註冊、兩個 keyframe 名稱不同、keyframe 內容動的是 `translate` 不是 `transform`。
   - `ManageSubtitleDialogV2.spec.tsx`：（紅）有把手；content 帶 `max-sm:data-[state=open]:animate-sheet-enter`（⚠️ 外殼的其他 token 今天就有，拿它們當斷言是假紅）；✕ 帶 `max-sm:top-4`；閒置頁尾有 `max-sm:hidden`；`toggle-fetch-mobile` 存在、點了跟 `toggle-fetch` 開同一個區塊、位置在 fetch 區塊之前；分集沒有手機那顆；**失敗狀態的頁尾沒有 `max-sm:hidden`、也沒有 `max-sm:flex-col-reverse`**；生成中頁尾有 `max-sm:flex-col-reverse`、按鈕外層有 `max-sm:ml-0`、完成時提示節點不存在；`ButtonCost` 有 `max-sm:w-full`。
   - 另外三個對話框的 spec：（紅）✕ 帶 `max-sm:top-[22px]`；（守）批次的 `gen-batch-drag-handle` 還在。
   - `GenerationProgressV2.spec.tsx`：（紅，改寫 `:147-165`）階段標籤是 `text-sm`、沒有 `text-[13px]`；（守）`text-[11px]` 的百分比還在。
   - ⚠️ jsdom 不看 media query，上面全是 token 斷言——`feedback_measure_the_breakpoint_you_return_to` 說得很清楚，class 字串對版面是空話。真的「手機長這樣」由 AC #8 守。

8. **真瀏覽器的驗證（兩層）。**
   - **視覺夾具**（`viewport: { width: 390, height: 844 }`）：`subtitle-manage-subtitle-dialog-v2-mobile`（F1-M，props 同既有桌機夾具 `:4378`）、`generation-progress-v2/轉錄中-mobile`、`generation-progress-v2/失敗-mobile`（非 Portal，狀態 div 截圖本來就能用）、`generation-consent/confirm-mobile`（AC #2）。
     - 🚨 **管理字幕的「生成中／失敗」做不出視覺夾具**：既有的兩個夾具（`:4378`、`:4461`）都是閒置；`genView` 是元件內部 state（`ManageSubtitleDialogV2.tsx:205`），只在 POST 成功的 `onSuccess`（`:292`）才切、階段來自 SSE（`:210`），沒有 prop 可以種；視覺 CI 沒有後端；`openTrigger` 也幫不上（spec `:288` 只在狀態 div 非零尺寸時才點，Portal 夾具是零尺寸）。⛔ 不要為了拍照在元件上開測試用的後門。
   - **e2e**（新，`tests/e2e/manage-subtitle-mobile.spec.ts`，`@manage-subtitle-mobile`，只跑 chromium 也會進 CI）：`page.setViewportSize({ width: 390, height: 844 })`（先例 `homepage-layout.spec.ts:399`），用 `page.route` 假掉詳情頁需要的 API（照 `batch-subtitle.spec.ts` 的做法），打開管理字幕後斷言：sheet 的 computed `animation-name` 是 `sheet-enter`；頁尾 `display: none`；`toggle-fetch-mobile` 可見；生成按鈕的寬度 ≈ sheet 寬 − 32；✕ 的 bounding box ≥ 44×44。再把 viewport 設成 640×844 重開一次：`animation-name` 是 `dialog-enter`、頁尾可見（**斷點的兩側都量**）。
     - 生成中／失敗：用 `page.route` 把 `POST **/transcribe` 回 202、SSE 端點回一則 `text/event-stream` 的 `transcribing`／`transcription_failed` 事件，斷言「關閉」的寬度 ≈ sheet 寬 − 32、提示在按鈕**下方**；失敗時「重試」可見。**SSE 假不起來就不要硬做**：把這一半記到 `disc-2026-09-manage-dialog-f3-f4-no-visual-coverage`（建單時已立），Completion Notes 寫明哪幾條只有 token 覆蓋。
   - 只產 darwin 基準線；`-linux` 交給 CI bootstrap（`project_visual_baseline_intentional_change` 四步）。視覺測試只有**一支** test，`--grep` 濾不到夾具，只能整支跑再還原無關的（本機固定會漂的 `parse-floating-parse-progress-card`／`retry-retry-notifications`）。

9. **另立的單子（建單時已寫入 sprint-status）。**
   - 新：`dsr-6f-2-glossary-mobile`、`dsr-6f-3-batch-and-consent-mobile`、`dsr-6f-4-workspace-mobile`；`disc-2026-09-bottomsheet-grabber-three-variants`、`disc-2026-09-viewport-fit-cover-missing`、`disc-2026-09-manage-dialog-f3-f4-no-visual-coverage`、`disc-2026-09-flow-f-mobile-missing-state-screens`。
   - 既有（已補本張的交接註記）：`disc-2026-09-dialog-close-target-44px`；另 `disc-2026-09-dialog-track-convert-not-wired`、`disc-2026-09-11px-micro-label-not-on-type-scale`、`disc-2026-09-dialogframe-shadow-vs-shadow-xl`。

10. **CI 全綠**：`pnpm run lint:all`、`pnpm nx run web:typecheck --skip-nx-cache`、`python3 scripts/check-design-tokens.py`、`pnpm nx test web`、`pnpm nx test api`。⛔ 絕不用 `run_in_background` 跑測試；每次跑完 `pnpm run test:cleanup`；Nx 一律 `NX_DAEMON=false`。
    - 🚨 **合併之後要看 `main` 那一次的 CI，不只看 PR 的**（dsr-6e-2 的教訓）。新的非同步測試一律 `findBy*`／`waitFor`，不要在 `await` 之後接裸的 `expect` 去讀「下一個階段才會出現」的東西。

## Tasks / Subtasks

- [x] **Task 1 — 設計稿（AC: #1）**
- [x] **Task 2 — 視覺測試支援手機 viewport（AC: #2）**
  - [x] `viewport` 欄位＋spec 的 set／reset；`confirm-mobile` 夾具拍得到 sheet；既有基準線零變動
- [x] **Task 3 — 共用外殼（AC: #3, #7）**
  - [x] 先寫紅測試 → `ui/mobileSheet.tsx`、keyframes、`Dialog` 的 `closeClassName`、四個對話框換常數
- [x] **Task 4 — 管理字幕手機版（AC: #4, #7）**
- [x] **Task 5 — 生成進度手機版（AC: #5, #7）**
- [x] **Task 6 — 夾具、基準線、e2e、收尾（AC: #6, #8, #9, #10）**
  - [x] 手機視覺夾具四個；`manage-subtitle-mobile.spec.ts`（390 與 640 兩側）
  - [x] dev-story Step 9：`f1-m-v2`／`f3-m-v2` 對新的手機基準線

## Dev Notes

### 這張的重點

- **它是地基。** 6f-2／3／4 都要用本張的三樣東西：sheet 外殼常數、滑入動畫、手機 viewport 的視覺測試。所以本張寧可把這三樣做扎實，內容只收管理字幕一個對話框。
- **桌機零變動是紅線。** 每一個改動都是 `max-sm:` 或 `sm:` 成對；共用元件（`Dialog`、`ButtonCost`）只加**可選**的 prop。
- **手機稿比桌機舊。** 桌機那幾張做過的裁定（百分比只在翻譯階段、單片不顯示用量、來源標籤不是泥金）手機稿沒跟上——先改稿，再對程式碼，不要反過來把程式碼改回去對舊稿。

### 上游契約（Rule 20 ack）

- 本張不消費也不改任何線上契約（純呈現層）。`useGenerationProgress` 的 `percentage` 語意沿用 dsr-6b。

### 建單裁定（2026-09-20，Sally／Alexyu 可在 review 推翻）

1. **拆成四張、一次建一張**（見 Context）。
2. ⚖️ **sheet 繼續用 Radix Dialog，不搬到 `ui/Sheet.tsx`（Base UI）。** 理由：這四個對話框在桌機是置中對話框、手機才是 sheet，同一個元件兩種外觀；搬到 Base UI 等於每個都要拆成兩個元件、兩套焦點管理，而且管理字幕裡還巢狀著名詞表。代價是滑入動畫要自己寫（兩個 keyframe）。`ui/Sheet.tsx` 留給「只有手機才存在」的面板（篩選、更多）。
3. ⚖️ **Flow F 的把手維持 `$bg-tertiary` 36×4（稿與程式碼一致），母版 `SG1ln` 不動。** 建單初稿以為「只有母版不一樣」，對抗驗證查出整份稿有三種畫法：Flow F 七個把手＋`zaD71`／`wwwiN` 是 `$bg-tertiary`；Flow B（`gzF1O`／`YeOZB`）、C（`ItByt`）、D（`hmhlF`／`ZrhaS`／`fcaAf`）六個是 `$text-muted` 40×4（與母版同）；`E992A` 是 `$border-subtle`；程式碼的 `ui/Sheet.tsx` 是 `w-10`＋`--border-subtle`。改母版只是把一種不一致換成另一種——另立 `disc-2026-09-bottomsheet-grabber-three-variants` 給 Sally 裁定（連同母版說明寫的「80%」與程式碼 `85vh` 的出入、以及 `ui/Sheet.tsx` 的 Rule 21 標頭寫 `<utility>` 而本張的 `mobileSheet.tsx` 認領 `SG1ln`）。
4. ⚖️ **閒置類的狀態在手機沒有頁尾；有動作的狀態保留。** F1 依稿（✕＋遮罩＋Esc 三條出路，✕ 做到 44×44 才成立——所以 AC #3 的關閉鈕是前提不是加分）。F3（生成中／完成）依稿把「關閉」＋提示搬成內容的延伸。F4（失敗）沒有手機稿，而「重試」是帶金額的付費按鈕（`DESIGN.md`「沒有金額，就沒有可按的按鈕」反過來也成立：付費按鈕不能因為沒畫就消失）——保留橫排頁尾。⚠️ 手機上 ✕ 成了閒置狀態唯一的按鈕式出口，而它的無障礙名稱是英文「Close」（`Dialog.tsx:68` 的 sr-only）——全 app 共用，記在 `disc-2026-09-dialog-close-target-44px` 的交接註記裡，本張不改。
5. ⚖️ **檔名與「轉為繁中」不在本張**：兩者桌機也沒接，是資料／功能缺口不是手機排版。
6. ⚖️ **SSE 膠囊的圓角與 11px** 留給 `dsr-6f-3` 連同全 Flow F 一起處理。

### 不要做的事

- 不要改 `GlossaryPanelV2`／`GlossaryRowV2`／`GenerationWorkspaceV2`。
- 不要改批次、同意、確認三個對話框的內容排版（只換外殼三樣）。
- 不要改 `ui/Dialog.tsx` 的預設樣式、陰影、桌機動畫。
- 不要動 `GENERATION_STAGES`、不要動 11px。
- 不要本機產 `-linux.png`。
- 不要用 `transform` 做滑入；不要讓 sheet 動畫在 `sm:` 以上生效。

### 已知陷阱

- **Radix Presence 比對 `animation-name`**：進出同名 → 立刻卸載，退場動畫永遠看不到（`styles.css:350-356`）。
- **Tailwind v4 的 `translate-*` 設的是個別屬性 `translate`**，所以 keyframe 動 `translate` 會跟 `translate-y-0` 疊在同一個屬性上——動畫期間以 keyframe 為準，結束後（`both`）停在 `0 0`，與 `translate-y-0` 相同，沒問題；但**桌機**的 `-translate-x-1/2 -translate-y-1/2` 也是同一個屬性，這就是為什麼一定要 `max-sm:`。
- **同一支 visual test 跑完所有夾具**：viewport 不重設就會污染後面一百多個夾具。
- **兩顆「搜尋線上字幕」同時在 DOM**：`getByTestId` 會撞，手機那顆用 `-mobile` 後綴；`getByText` 的既有測試可能因此變成 multiple matches——改用 testid。
- **`@theme`，不是 `@theme inline`**（註解 `styles.css:444-451`、區塊 `:452-462`）：inline 會把值烤死，失去 reduced-motion 的收合。
- **`ml-auto` 會取消 `align-items: stretch`**（AC #5）：滿版按鈕沒滿版，十之八九是外面還有一層 `ml-auto`。
- **`DialogContent` 把 `...props` 展開到 DOM**：新 prop 沒解構就會漏成 HTML 屬性。
- **`GenerationProgressV2` 是三個地方共用的**：改它的手機 class，批次與工作區的手機畫面也會變。
- **行號以建單時為準**（2026-09-20，main `6e09d4bc`）。

### Source tree

```
apps/web/src/components/ui/mobileSheet.tsx(+spec)                     ← Task 3（新）
apps/web/src/components/ui/Dialog.tsx(+spec)                          ← Task 3（可選 prop）
apps/web/src/styles.css（＋ motion spec）                              ← Task 3
apps/web/src/components/subtitle/ManageSubtitleDialogV2.tsx(+spec)    ← Task 3, 4, 5
apps/web/src/components/subtitle/GenerationProgressV2.tsx(+spec)      ← Task 5
apps/web/src/components/subtitle/GenerationBatchDialogV2.tsx          ← Task 3（只換外殼三樣）
apps/web/src/components/subtitle/consent/GenerationConsentView.tsx    ← Task 3（同上）
apps/web/src/components/subtitle/consent/ConfirmGenerationDialog.tsx  ← Task 3（同上）
apps/web/src/routes/test/-gallery.fixtures.tsx、routes/test/gallery.tsx ← Task 2, 6
tests/visual/components.visual.spec.ts                                ← Task 2
tests/e2e/manage-subtitle-mobile.spec.ts                              ← Task 6（新）
apps/web/src/styles-motion.spec.ts                                    ← Task 3（已存在，擴充）
ux-design.pen、pen-tokens.json、screenshots/flow-f-subtitle-v2/{f1-m-v2,f3-m-v2}.png ← Task 1
```

### Cross-Stack Split Check

後端 task **0 個**，前端／設計／測試工具 task 6 個。→ 不觸發跨棧拆分；規模拆分已在 Context 說明。

### Time-dependent visual coverage

- **N/A — no wall-clock-reading components touched.** `ManageSubtitleDialogV2`、`GenerationProgressV2`、`ui/Dialog`、新的 `ui/mobileSheet` 都不讀時鐘；動畫時長是 CSS token，視覺測試在 reduced-motion 下跑。

### References

- [Source: `apps/web/src/components/ui/Dialog.tsx:27, 60, 66-69`、`ui/Sheet.tsx:30-37`、`ui/ButtonCost.tsx:73`]
- [Source: `apps/web/src/styles.css:300-313, 346-397, 440-462, 525-534`]
- [Source: `apps/web/src/components/subtitle/ManageSubtitleDialogV2.tsx:385-412, 466-528, 643-704, 785-843`、`GenerationProgressV2.tsx:32-39, 49-69, 156-195, 217-251`]
- [Source: `GenerationBatchDialogV2.tsx:431-462`、`consent/GenerationConsentView.tsx:519-539`、`consent/ConfirmGenerationDialog.tsx:82-100, 207`]
- [Source: `apps/web/src/routes/test/-gallery.fixtures.tsx:396-466, 4378, 4461, 4629, 4666`、`tests/visual/components.visual.spec.ts:195-320`、`playwright.config.ts:133-165`]
- [Source: `ux-design.pen` `JkdfH`／`FtarQ`／`k8sJl4`／`Me1fR`／`r1EY9`／`JbXai`／`SG1ln`／`JzmvC` —— SM 建單時以 Pencil MCP 逐節點讀出（2026-09-20）；程式碼現況由唯讀稽核代理查證]
- [Source: `sprint-status.yaml` → `dsr-6f-flow-f-mobile`（七段交接）、`disc-2026-09-dialog-close-target-44px`、`disc-2026-09-dialog-track-convert-not-wired`、`disc-2026-09-11px-micro-label-not-on-type-scale`]
- [Source: `dsr-6b-manage-subtitle-dialog-desktop.md`（桌機的裁定）、`dsr-6e-2-consent-small-surfaces.md`（token 斷言、賽跑測試的教訓）]
- [Source: project-context.md#Rule 16 / #Rule 21 / #Rule 23 / #Rule 24；`.claude/memory/project_visual_baseline_intentional_change.md`、`feedback_css_verify_before_iterate.md`、`feedback_measure_the_breakpoint_you_return_to.md`]

## Dev Agent Record

### Agent Model Used

Claude Fable 5.1 — dev-story (Amelia)

### Debug Log References

- Nx 一律 `NX_DAEMON=false`；視覺測試只有一支 test、`--grep` 濾不到夾具，整支跑 8.8 分鐘；`test:cleanup` 會砍掉 `nx serve web`。
- e2e 第一輪兩條紅，都是測試自己的問題：① e2e 跑的時候動畫是開的，進場中途量到的 sheet 底邊在 y=1046（還在螢幕下面）→ 先 `await` `getAnimations()` 的 `finished` 再量；② 觸發端點的真實 URL 帶 `?translate=true`，`**/transcribe` 這個 glob 對不上、請求打到真後端 → 改 `**/transcribe?*`。
- `webkit-core` project 有自己的測試集合，這支 spec 不在裡面（`No tests found`）——CI 的 e2e 只會在 chromium 跑它。

### Completion Notes List

**做了什麼（對使用者的差別）**

1. **手機上的字幕對話框真的從底部滑上來。** 新的 `sheet-enter`／`sheet-exit`（動個別屬性 `translate`、不淡入），只在 `max-sm:` 生效；桌機仍是置中的 `dialog-enter`。真瀏覽器量過：390 寬算出來的 `animation-name` 是 `sheet-enter`，640 寬是 `dialog-enter`。四個對話框（管理字幕、批次、同意、確認）一起拿到。
2. **關閉鈕在手機是 44×44**，而且對齊各自的標題列（管理字幕 `top-4`、另外三個 `top-[22px]`——那三個以前其實是歪的）。`ui/Dialog` 只多了一個可選的 `closeClassName`；不傳的對話框 ✕ 的 class 逐字不變（有守門測試）。
3. **管理字幕在手機上對稿（F1-M）**：補上把手；標題列 44 高、閒置時沒有底線；內容 [6,16,16,16] gap 14；「生成字幕」滿版、說明置中在下；閒置時**沒有頁尾**，「搜尋線上字幕」搬到內容最下面（面板開在它下方）。
4. **生成中（F3-M）**：「關閉」滿版、提示在按鈕**下方**置中、沒有上框線；階段標籤 13 → 14。**失敗（F4）保留原本的橫排頁尾**——「重試」是帶金額的按鈕。
5. **視覺測試現在拍得到手機。** `GalleryFixture.viewport`；spec 只在 viewport 真的不同時才 `setViewportSize`、下一個夾具自動回到 1280×800。**既有三百多張基準線零變動**（整支跑過，`git status` 只多出四個新夾具）。

**頁尾各狀態盤點（AC #4 要求動手前逐一列出）**

| 狀態 | 頁尾裡有什麼 | 手機 |
| --- | --- | --- |
| 閒置（電影） | 搜尋線上字幕、關閉 | 藏（搜尋搬進內容、關閉＝✕） |
| 閒置（分集）／載入中／F2／F5／`notConfigured` | 關閉 | 藏 |
| `triggerError` | 關閉（**帶金額的「重試」在內容區的錯誤面板裡，不在頁尾**） | 藏——付費按鈕不受影響 |
| 生成中／完成 | 提示、關閉 | 直排：關閉滿版、提示在下 |
| 失敗 | 提示、稍後再試、**重試 $x** | 不變（橫排） |

**設計稿（Task 1）**：F3-M 關掉 `E10kib`（45%）與 `NPtOO`（本次用量）、補階段說明 `lGMjs`（複製自 F3-D `OQ2vp`）；F1-M `un8dT` → `$text-secondary`；中性註記 `PyB9P` 放在 `bSlYR` 下方 40px。母版 `SG1ln` 沒動。存檔後已 grep 磁碟檔確認落盤；匯出後只留 `f1-m-v2`／`f3-m-v2`＋`pen-tokens.json`。（`NPtOO` 被關掉後 Pencil 會警告它的 `fill_container` 不在 flex 裡——停用節點的誤報，與 6e-1 的 `DesXa` 同類。）

**真瀏覽器驗證（AC #8）**

- 視覺夾具四個（390×844）：`subtitle-manage-subtitle-dialog-v2-mobile`、`generation-progress-v2/轉錄中-mobile`、`/失敗-mobile`、`generation-consent/confirm-mobile`。看過圖：sheet 貼底、把手在（`--bg-tertiary` 壓在 `--bg-secondary` 上，很淡——與稿一致，三種把手的裁定在 `disc-2026-09-bottomsheet-grabber-three-variants`）、✕ 與標題同一條水平線、按鈕滿版。
- e2e `tests/e2e/manage-subtitle-mobile.spec.ts`（chromium，3 條，burn-in 3 輪 9/9）：390 寬——`sheet-enter`、貼底、把手、頁尾隱藏、✕ ≥44×44、生成按鈕寬 358（390−32）、搜尋面板開在觸發鈕下方；**640 寬（斷點另一側）**——`dialog-enter`、沒有把手、頁尾回來、按鈕沒被拉寬；390 寬生成中（假掉 POST 與 SSE）——「關閉」寬 358、提示在按鈕下方、階段標籤算出來是 14px。**SSE 假得起來**，所以 `disc-2026-09-manage-dialog-f3-f4-no-visual-coverage` 剩下的缺口只有「失敗狀態沒有真瀏覽器覆蓋」（那一半維持 token 斷言）。

**閘門結果**

| 閘門 | 結果 |
| --- | --- |
| `pnpm run lint:all` | ✅ 0 errors（第一輪 2 個 `no-undef`：e2e 裡裸的 `getComputedStyle` → `window.getComputedStyle`，與既有 spec 同寫法） |
| `pnpm nx run web:typecheck --skip-nx-cache` | ✅ |
| `python3 scripts/check-design-tokens.py` | ✅ |
| `pnpm nx test web` | ✅ 3935/3935（274 檔；新增 3 支 spec） |
| `pnpm nx test api` | ✅（沒改後端） |
| e2e（chromium） | ✅ 3/3 |
| 視覺 | ✅ 1 passed（8.8 分鐘）；既有基準線零變動；新增 4 張 darwin |

- 🔗 AC Drift: NONE（checked: 'toggle-fetch|dialog-close|gen-batch-drag-handle|text-\[13px\]' across `_bmad-output/implementation-artifacts/*.md` — 命中的是 dsr-6b（桌機的管理字幕）與 dsr-6d-b（批次把手）；本張的改動全部在 `max-sm:` 之下或是可選 prop，桌機的 AC 逐條仍成立。`GenerationProgressV2` 的手機標籤 13→14 是 dsr-6b 明文留給 dsr-6f 的。）
- 📎 Contract Stamps: NONE（純呈現層；不消費也不改任何 `[@contract-v*]`）
- 🎭 A11y Pre-Flight: PASS（6 個元件；touched files 上 0 個新的 jsx-a11y warning）。手機上兩顆「搜尋線上字幕」同時在 DOM，但各自有一顆是 `display:none`，不在無障礙樹與 tab 順序裡。Radix 的焦點管理不變。⚠️ ✕ 的無障礙名稱仍是英文「Close」（共用元件，記在 `disc-2026-09-dialog-close-target-44px`）。
- 🎨 UX Verification: PASS —— `f1-m-v2`／`f3-m-v2` 對新的手機基準線與 e2e 量測：

| Area | Design Spec | Implementation | Match? |
| --- | --- | --- | --- |
| 外殼 | 貼底、上緣 radius-xl、把手 36×4 | 同；由下往上滑入 | ✅ |
| 標題列 | 44 高、[0,4,0,16]、F1 無底線／F3 有 | 同 | ✅ |
| 關閉鈕 | 44×44、圖示 18 `$text-secondary` | 同（e2e 量過 ≥44） | ✅ |
| 內容 | [6,16,16,16] gap 14 | 同 | ✅ |
| 軌道列 | [10,12] gap 10；兩行（來源／檔名） | padding 同；**一行**（沒有檔名資料） | ⚠️ 刻意（桌機同） |
| 生成按鈕 | 滿版 [12,20]、說明置中在下 | 同（e2e 量過 358） | ✅ |
| 搜尋線上字幕 | 內容最下面置中 h44 | 同 | ✅ |
| 簡轉繁 `kfQ76` | 有 | 沒接 | ❌ 既有單 `disc-2026-09-dialog-track-convert-not-wired` |
| F3 步驟 | 直排、Body 14、無百分比 | 同 | ✅ |
| F3 關閉／提示 | 滿版、提示在下、無框線 | 同 | ✅ |

**🔍 /ship 對抗式 CR（2026-09-20，fresh-context 代理，只讀；這次特別先查 CI 賽跑風險）**——0 HIGH／2 MEDIUM／8 LOW，**修 8、記 2、駁回 0**。

| 等級 | 問題 | 處置 |
| --- | --- | --- |
| **M1** | 兩張非 Portal 的手機基準線其實不是手機寬：gallery 的外層是會縮起來的 flex 子項，裡面的 `w-full` 只解析成內容寬——「轉錄中」那張只有 **96px** 寬，滿版列、靠右的百分比、置中的說明全都沒拍到。後面三張單子都要用這個機制。 | 修：外層也加 `w-full min-w-0`；兩張重拍（現在 326 寬＝390−頁面 padding）；spec 加守門——有 `viewport` 的在頁內夾具寬度不得小於 viewport−65 |
| **M2** | e2e 的「生成中」測試沒有證明假的 SSE 真的被吃到：所有斷言在一開始的「提取音訊」階段也成立，假資料壞了測試照樣綠。 | 修：先等 `gen-stage-轉錄中` 的 `data-state` 變成 `active`（會自動重試，不引入賽跑） |
| L1 | `evaluate` 把 `Animation` 物件陣列傳回 Node；動畫被取消時 `finished` 會 reject。 | 修：`Promise.allSettled(…).then(() => undefined)` |
| L2 | `expect(box.x).toBe(0)` 遇到 `-0` 會紅。 | 修：`Math.round` |
| L3 | 四處註解變成假話（fixtures 說「gallery 會 throw」、spec 的註解位置錯、`13px`、`is dsr-6f`）。 | 修 |
| L4 | 失敗狀態的頁尾在手機仍是 `px-6`，比 16px 的內容區多縮 8px。 | 修：頁尾基底加 `max-sm:px-4`＋token 斷言 |
| L6 | 「完成」那條單元測試在沒有空 span 時是空轉的。 | 修：直接斷言「沒有任何在手機上看得到的空 span」 |
| L7 | `fulfill({ response, json })` 會保留原本的 `content-length`，而 body 變長了。 | 修：只傳 `status`＋`json` |
| L5 | 對話框開著時旋轉手機（跨過 640）會重播一次進場動畫。純外觀、不會卡住。 | 記：併入 `disc-2026-09-bottomsheet-grabber-three-variants` 同一輪裁定 |
| L8 | 固定在底部的 `MobileTabBar`（z-40）可能蓋到很高的在頁內手機夾具。目前的夾具都夠矮。 | 記：已寫進 6f-2／6f-3／6f-4 的 sprint 條目 |

已查證無誤（摘要）：CI 的 e2e 是 `serve --single` 的正式版前端對真後端，`**/…` glob 兩個 origin 都對得上；glob 是錨定的，`movies/{id}` 不會吃掉 `/transcribe…`；這支 spec 在 CI 只跑 chromium；SSE 假資料結束後每 10 秒才重連一次，不會洗版也不會卡 `networkidle`；精確像素斷言（390／844／358）在 CI 的 headless chromium（DPR 1、隱藏捲軸）是安全的；每個 bucket 測試有自己的 page，viewport 不會外洩。

**已知殘留（刻意）**

- 批次、同意、確認三個對話框的手機**內容排版**沒動（6f-3）；它們現在多了滑入動畫與 44px 的 ✕。
- safe-area 的 class 今天是空轉的（沒有 `viewport-fit=cover`）。

### Discovery Triage

- **Did this story discover any work outside its current scope?** **YES** —— 建單當下處理如下。

- **① expand-scope-in-place**
  - 視覺測試拍不到手機版的對話框（🔴 #6）→ **AC #2**
  - 管理字幕漏了把手、頁尾沒有 safe-area（🔴 #3、#5）→ **AC #3、#5**
  - 手機稿三處與桌機已裁定的事矛盾（45%、用量、來源標籤顏色）、少一句階段說明（🔴 #11、#13、#14）→ **AC #1**
  - sheet 的關閉鈕 16×16（🔴 #4）→ **AC #3**（只限 sheet；全 app 仍在 `disc-2026-09-dialog-close-target-44px`）

- **② spawn-blocking-story**：無。

- **③ backlog-with-carry-forward-link**（建單時立）
  - `dsr-6f-2-glossary-mobile`、`dsr-6f-3-batch-and-consent-mobile`、`dsr-6f-4-workspace-mobile`（本張拆出來的三張，各帶稽核事實）
  - `disc-2026-09-bottomsheet-grabber-three-variants` — 把手在稿與程式碼裡共三種畫法；母版寫 80%、程式碼 85vh；`ui/Sheet.tsx` 的 Rule 21 標頭
  - `disc-2026-09-viewport-fit-cover-missing` — 沒有 `viewport-fit=cover`，全站的 `env(safe-area-inset-*)` 今天都是 0
  - `disc-2026-09-manage-dialog-f3-f4-no-visual-coverage` — 管理字幕的生成中／失敗狀態沒有 prop 可種，拍不了視覺基準線
  - `disc-2026-09-flow-f-mobile-missing-state-screens` — F2／F4／F5／F10 沒有手機稿

- Reference: `project-context.md` Rule 24

### File List

**新增**

- `apps/web/src/components/ui/mobileSheet.tsx`（+spec）
- `apps/web/src/components/ui/Dialog.spec.tsx`
- `apps/web/src/routes/test/gallery-fixture-viewport.spec.ts`
- `tests/e2e/manage-subtitle-mobile.spec.ts`
- `tests/visual/.../{subtitle-manage-subtitle-dialog-v2-mobile,generation-progress-v2/轉錄中-mobile,generation-progress-v2/失敗-mobile,generation-consent/confirm-mobile}/default-visual-darwin.png`
- `_bmad-output/implementation-artifacts/dsr-6f-1-mobile-sheet-and-manage-dialog.md`（建單產物）

**修改**

- `apps/web/src/components/ui/Dialog.tsx`（可選 `closeClassName`）
- `apps/web/src/styles.css`、`apps/web/src/styles-motion.spec.ts`
- `apps/web/src/components/subtitle/ManageSubtitleDialogV2.tsx`（+spec）
- `apps/web/src/components/subtitle/GenerationProgressV2.tsx`（+spec）
- `apps/web/src/components/subtitle/GenerationBatchDialogV2.tsx`（+spec；只換外殼三樣）
- `apps/web/src/components/subtitle/consent/GenerationConsentView.tsx`（+spec；同上）
- `apps/web/src/components/subtitle/consent/ConfirmGenerationDialog.tsx`（+spec；同上）
- `apps/web/src/routes/test/-gallery.fixtures.tsx`、`apps/web/src/routes/test/gallery.tsx`
- `tests/visual/components.visual.spec.ts`
- `ux-design.pen`、`_bmad-output/pen-tokens.json`、`_bmad-output/screenshots/flow-f-subtitle-v2/{f1-m-v2,f3-m-v2}.png`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`

⛔ **沒有碰**：`GlossaryPanelV2`／`GlossaryRowV2`、`GenerationWorkspaceV2`、`ui/ButtonCost.tsx`（用既有的 `className` prop）、`ui/Sheet.tsx`、`index.html`、任何後端檔案。

## Change Log

| 日期 | 內容 |
| --- | --- |
| 2026-09-20 | 🔍 **/ship 對抗式 CR**：0H／2M／8L，修 8、記 2。最重要：兩張手機基準線其實只有 96px 寬（gallery 外層會縮起來）→ 修外層、重拍成 326 寬、spec 加寬度守門；e2e 的生成中測試補上「SSE 假資料真的被吃到」的證明。web 3935/3935、e2e burn-in 9/9、既有基準線零變動。 |
| 2026-09-20 | 🚧 **REVIEW**（dev-story, Amelia）。Task 1–6 完成。手機 sheet 由下往上滑入（真瀏覽器量過：390 是 `sheet-enter`、640 是 `dialog-enter`）、44×44 的 ✕、管理字幕與生成進度對 F1-M／F3-M；視覺測試支援手機 viewport，既有基準線零變動。閘門：lint 0 errors、typecheck ✅、design-tokens ✅、web 3935/3935、api ✅、e2e 3/3（burn-in 9/9）。 |
| 2026-09-20 | 🔍 **建單後對抗驗證**（fresh-context 代理，只讀；逐節點比對 `.pen`、逐行比對程式碼、用 repo 的 Tailwind 4.1.18 實際編譯候選 class）：4 CRITICAL／11 SHOULD FIX／6 NIT，**全部併入**。最重要：① 管理字幕的「生成中／失敗」根本做不出視覺夾具（狀態是元件內部的、來自 POST＋SSE）→ 改成內層元件的手機夾具＋一支 390／640 兩側都量的 e2e；② 一個固定的 `top` 置不了四個對話框的 ✕（標題列高度不同），`14px` 連管理字幕自己都不對 → `top` 由各對話框自己帶；③ 「關閉」外面那層 `ml-auto` 會讓滿版失效，而且稿上 F3-M 根本沒有頁尾 → 改寫成依狀態的 class；④ 「只有母版的把手不一樣」是錯的，整份稿有三種 → 不改母版、另立單子；⑤ `closeClassName` 不解構會漏成 DOM 屬性；⑥ 批次的把手有 testid 在被斷言，`SheetGrabber` 要能轉傳 props；⑦ safe-area 今天是 0（沒有 `viewport-fit=cover`）；⑧ `GenerationProgressV2` 是共用的，手機字級的改動會波及批次與工作區；⑨ Rule 21 標頭其實只缺 F3-M、文法是 ` + Screen`。Tailwind／twMerge 的疊加行為已實際編譯查證，會照單子寫的運作。 |
| 2026-09-20 | Story 建立（SM Bob, create-story）。`dsr-6f` 依版面拆成四張，本張是地基：sheet 共用外殼（把手、由下往上的動畫、44×44 關閉鈕、safe-area）、視覺測試支援手機 viewport、管理字幕 F1-M 與生成進度 F3-M。建單稽核：SM 以 Pencil MCP 讀八張手機稿與桌機對照，唯讀代理查程式碼在手機寬度的現況。找到 17 項；最重要的兩個前提——手機上的「抽屜」其實是原地淡入放大、視覺測試把 viewport 釘在 1280 所以沒有任何 sheet 能留手機基準線。六項建單裁定。 |
