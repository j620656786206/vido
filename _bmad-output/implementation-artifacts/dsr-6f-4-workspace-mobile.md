# Story DSR.6f-4：手機上的「生成工作區」照手機稿排好——有返回鍵、總覽卡直排、即時活動預設收起來

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As the person who checks on a long paid subtitle batch from a phone,
I want 生成工作區這一頁在手機上有返回鍵、總覽卡直排、左右留 16、即時活動預設收起來（要看再展開）,
so that 我一打開就看到「跑到第幾部、花了多少錢、哪一部失敗」，而不是桌機版面硬塞進 390 寬、事件紀錄把整頁拉得很長。

## Context

`dsr-6f`（Flow F 全部手機稿）拆出來的**第四塊，也是最後一塊**。**Depends on: `dsr-6f-1`（done，PR #488）。** 做完這張，`dsr-6f-flow-f-mobile` 傘狀條目即可結案。

| 畫面 | 稿 | 程式碼 |
| --- | --- | --- |
| 生成工作區（執行中） | F11-M-v2 `PXB0z`（390×1500） | `apps/web/src/components/subtitle/GenerationWorkspaceV2.tsx`（路徑 `/activity?view=generation`，由 `activity/ActivityHub.tsx:308-317` 掛載） |
| 沒有手機稿 | 達上限／結束（F12）、閒置／中途接入／單部（F13）、載入中、資料錯誤、取消確認、**即時活動展開後** | 同一個檔案 |

🚨 **這張跟前三張根本不同：它是一「頁」，不是抽屜。**

- ⛔ 不用 `ui/mobileSheet.tsx` 的 `MOBILE_SHEET_CONTENT`／`MOBILE_SHEET_CLOSE`（那是對話框的外殼）；沒有把手、沒有 ✕、沒有 80% 高度上限、沒有 `sheet-enter`。
- 出口是**返回鍵**，不是 ✕。
- 頁面外面是 `AppShellV2`：56 高的 sticky 頂列＋手機底部 84 高的 `MobileTabBar`（`fixed … z-40 … sm:hidden`）。`<main>` 已經留了 `pb-[calc(84px+env(safe-area-inset-bottom))] sm:pb-0`（`AppShellV2.tsx:104`），所以**頁內流動的內容不會被分頁列蓋住**；shell **不給**任何左右內距——手機左右的 24 是工作區自己的 `px-6`。

⚠️ **驗收基準是 `.pen` 節點值**。PNG 只是參考。

### 🔴 建單時查到的事（main `aee223d0`；行號皆為現況）

**頁首（稿 `e6nOTG` topbar）**

1. **沒有返回鍵。** 稿 `d86Ux` 是 24 的 `chevron-left`（`$text-secondary`）放在標題左邊（`KUJ6E` gap 10）。程式碼 `:591-617` 完全沒有；手機上離開工作區只能按底部「活動」分頁。全 app 沒有共用的「手機頂列＋返回」元件（先例只有 `DetailHeroV2.tsx:73-82` 的浮動圓鈕與 `AppShellV2.tsx:113-129` 搜尋覆蓋層的 44×44 `返回`）。
2. **順序相反。** 稿：第一行「返回＋標題……狀態藥丸（靠右，`CarVq` `space_between`）」，第二行麵包屑（`J79fp`，Label 12、`$text-muted`）。程式碼：麵包屑在上（`text-sm`、`$text-secondary`）、標題與藥丸在下且藥丸緊貼標題（`gap-2`）。
3. **標題字級。** 稿 `sSwG5` 是 H4 18；程式碼 `:603` `text-xl`（20）。`DESIGN.md:604-616`「標題字級降一階」：H3 20 → 手機 18。
4. **內距。** 稿 topbar padding [12,16]、兩行之間 gap 2、整頁各區塊之間 gap 14（`PXB0z` `gap:$Space/md-plus`）。程式碼頭區 `:591` `px-6 pb-4 pt-6 gap-3.5`、身體 `:620` `px-6 pb-6 gap-4`、底部列 `:800` `px-6`——**手機左右 24，全 app 其他手機畫面都是 16**（`tests/e2e/batch-consent-mobile.spec.ts:37` `const GUTTER = 16`）。
5. 狀態藥丸的圓點：稿 `AA7pb` 6×6；程式碼 `h-2 w-2`（8，與桌機稿一致）。

**總覽卡（稿 `QNVCS`）**

6. **稿是直排三行**：①「已完成 12 / 38 部」＋SSE 膠囊靠右（`v6RDo` `space_between`）②滿寬進度條（`AX0nC` `fill_container` h6）③「本次用量 $0.42 ／上限 $5.00」**同一行**（`dQVOX`）。padding [12,14]、gap 8。**沒有「整批進度」這個小標**。
7. **程式碼是桌機的橫排**：`:106` `flex flex-wrap items-center gap-x-8 gap-y-3 … px-4 py-3`，兩個直排小欄（各帶一個 `text-[11px]` 小標）＋膠囊；**進度條寫死 `w-[180px]`**（`:122`）。390 寬時三塊會各自換行，進度條只佔一半。
8. 字級：稿 `YQrWx` H4 18 Mono／`tAobl`・`C3lFU`・`jC6W2` Body 14／`qmVqX` Body 14 600 Mono／`o6T6j` Body 14 Mono；程式碼 `text-xl`／`text-base`／`text-xs`（部）／`text-base`（桌機稿是 H3／BodyLg，桌機不動）。

**佇列**

9. **列本身幾乎已經對了。** `QueueRow` `:263` `px-4 py-3.5 gap-2.5`＝稿 [14,16] gap 10；海報 40×60 ✓；標題 `text-base font-semibold`＝BodyLg 600 ✓；徽章 ✓；列距 `space-y-2.5`＝10 ✓；步驟條 `sm:pl-[54px]` 在手機本來就是 0 ✓，`GenerationProgressV2` 手機已是直排 14px（6f-1）。**本張不重做列。**
10. **稿的進行中那一列過時，而且牴觸裁定。** `n79vZ` 把 slot `dxgox` 換成一顆**迷你步驟條** `G5v4u`（直排、圓點 8、字 12、gap 6、高 132），還畫著「45%」兩處（`OMD2y` enabled、`UmgMX`）。⚖️ Alexyu 2026-09-21 已裁定「六步進度條在手機一律直排、同 F3-M，全 app 手機只有一種步驟條，`GenerationProgressV2` 不需要 compact 模式」→ **改稿不改碼**：換成 F3-M 的 `fS5is`（圓點 22、Body 14、每列 31 高、總高 216；裡面那顆 `E10kib`「45%」維持停用）。轉錄沒有百分比（dsr-6b 裁定），`OMD2y` 要關。
11. **稿上三句話是系統說不出來的。** `Qp7Cm`「已寫入繁中字幕」（程式碼刻意不說「繁中」：`generationQueueRow.ts` `queueRowSubStatus` →「已完成，字幕已寫入檔案」）；`n79vZ`「轉錄音訊中…」（程式碼「轉錄中…」）；`Ah8fN`「轉錄失敗，已略過（計入失敗數）」（程式碼只有四句：沒有可用的字幕來源／這部正在別處處理／生成失敗）。桌機稿 `KZmNG` 有同樣的舊句子 → 另立 `disc-2026-09-f11-pen-row-copy-and-breadcrumb-stale`，本張只改手機稿。
12. 稿的失敗列副標是**兩行**（`lUZol` `fixed-width`，46 高）；程式碼 `:276` `truncate`。📎 以現在的四句話（最長 9 個字 ≈126px，手機可用寬 ≈193px）**今天不會被截**——這是對齊稿的防呆，不是現行 bug，測試要誠實標「守」。
13. **取消確認在手機會擠。** `:719` 標題列是 `flex items-center justify-between gap-3`，「生成佇列」（≈64px）旁邊直接放 `CancelAll`；確認態 `:881` 是 `flex flex-wrap items-center gap-3`：一句 ≈294px 的話＋（失敗時）≈252px 的錯誤句＋兩顆鈕，全部擠在 358−64−12≈282px 裡。稿只畫了「全部取消」單顆（`jbYHw` h44）。
14. **底部列（達上限／結束）沒有手機稿。** `:800` `flex-wrap justify-end gap-3 px-6`；「關閉」「下次繼續」靠右、不等寬。它在頁面流裡（不是 fixed），所以不會被分頁列蓋住。

**即時活動（稿 `rMD73`）**

15. **稿：手機預設收合。** 只有一條標題列 `AfUUS`（`$bg-secondary`、`$radius-md`、**無框線**、padding [12,14]）：`activity` 16＋「即時活動」Body 600＋「· 自開啟本頁起累積」Label muted **緊跟在標題後面**……`chevron-down` 20（`fdUFU`）靠右；下面 `mZsfW`：一行 Label muted「展開查看即時事件（不含逐字內容、無時間戳）」＋SSE 膠囊。**稿沒有畫任何一列事件，也沒有畫展開後的樣子。**
16. **程式碼：`lg` 以下永遠展開、沒有高度上限。** `:408` 的 `lg:sticky lg:top-[4.5rem] lg:max-h-[calc(100vh-6rem)] lg:w-[400px]` 只在 ≥1024 生效；`:620` `lg:flex-row`。手機上事件一多，整頁被拉得很長，而且 **dsr-6d-c-2 CR H3 修好的「清單自己捲、跟著最新一列」在手機上根本不會發生**（沒有上限 → 清單不捲、捲的是整頁）。
17. 🚨 **全頁唯一的 `aria-live` 就在這個區塊裡**：`workspace-log-announcer`（`:433`，sr-only、一直掛著）。`GenerationWorkspaceV2.spec.tsx:815-817` 釘住「跨 mode 切換是**同一個 DOM 節點**」且 `document.querySelectorAll('[aria-live]')` **恰好 1 個**。**收合不准用卸載**（`{open && …}`）去做——region 跟內容一起掛載，螢幕報讀不會唸（dsr-6d-b M4、dsr-6d-c-1 M4 各踩過一次）。
18. `GenerationWorkspaceV2.spec.tsx:355/363` 釘住終態時 `workspace-sse-chip` **恰好 1 顆**（紀錄頁尾那顆）→ ⛔ **不准**像 6f-3 那樣「畫兩次」多生一顆 `-mobile` 膠囊。
19. 沒有 `@radix-ui/react-collapsible`／accordion，全 repo 也沒有 `<details>`；收合都是手寫 `useState`＋`aria-expanded`（`dashboard/CollapsibleSection.tsx`、`consent/CandidateListPanel.tsx:538-566`）。沒有共用的 `useMediaQuery`／`useIsMobile`——**本張也不需要**（全部用 `max-sm:` 類別做，見裁定 2）。

**測試現況**

20. 視覺夾具 `generation-workspace-v2/{running,budget_ceiling,complete-with-failures,idle}` 全是 `width: 1200` 的框寬夾具（頁面 1280）→ 它們守的是桌機。**工作區做不出誠實的手機視覺夾具**：它不是 Portal，「在頁內」的 `viewport: 390×844` 夾具只有 **326 寬**（`/test/gallery` 的 `p-8` 吃掉 64），而且比 844 高的元素截圖會把 fixed 的 `MobileTabBar` 一起拍進去（6f-1 CR L8；`components.visual.spec.ts` 只把 header 改 static，沒處理分頁列）。6f-2 CR M7 已裁定：**不把 326 寬的假版面收成基準線**。→ 手機版面由 e2e 守（裁定 4）。
21. `tests/e2e/generation-workspace.spec.ts`（5 條）沒有設 viewport、沒有手機斷言；CI 只跑 `chromium`（1280×720）。它的 stub 全是檔內手寫（`ITEMS`／`snapshot`／`stubCommon`／`sseFrame`），沒有用 `tests/support/`。⚠️ 第 5 條「long log scrolls inside its pane」靠的是 `lg:max-h-…`——本機 `test:e2e` 會用 Pixel 5／iPhone 13 跑它（`playwright.config.ts:132-141` 沒有 `testMatch`）：那兩個 project 寬度 <1024，清單沒有上限 → **推定今天就已經是紅的**（`:267` `scrolls:false`），不是本張造成；本張之後清單在手機還會預設藏起來。→ 在**那一條裡面**加 `test.skip(({ viewport }) => (viewport?.width ?? 1280) < 1024, 'pane cap is lg-only')`。⛔ 不要寫成 chromium-only——那會連 `firefox`（1280 寬，這條測試在那裡是有效的守門）一起跳掉。
22. 最容易被排版改動打到的三條既有斷言：`GenerationWorkspaceContainer.spec.tsx:522`（`row.querySelector('.font-semibold.text-base')` → **列標題的 class 不准動**）、`GenerationWorkspaceV2.spec.tsx:110-111`（佇列 `<ul>` 裡**不准有任何 button**）、`:815-817`（🔴 #17）。**沒有任何測試**斷言 `w-[180px]`、`lg:`、`flex-wrap`、`sm:px-8`、麵包屑文字——這些可以放心加 `max-sm:`。
23. **Rule 21**：`GenerationWorkspaceV2.tsx:2` 的 `// Design ref:` 只列 F11-D／F12-D／F13-D，沒有 F11-M。

### 設計稿節點

| 代號 | 節點 | 本張要看的 |
| --- | --- | --- |
| F11-M-v2 | `PXB0z`（x=19000、y=32693、390×1500、`layout:vertical`、gap 14、`clip:true`） | topbar `e6nOTG`（[12,16]、gap 2）：`CarVq` `space_between`→`KUJ6E`（gap 10：`d86Ux` chevron-left 24 `$text-secondary`、`sSwG5` H4「批次生成字幕」）＋`htLc2` 藥丸（[4,10]、`AA7pb` 6×6、`qVFne` Label）；`J79fp` 麵包屑 Label muted |
| 總覽卡 | `nQHJX`（[0,16]）→ `QNVCS`（[12,14]、gap 8、`$bg-secondary`、`$radius-md`、`$border-subtle`） | `v6RDo`（`leB0o` 計數：`XwSrb` Body／`YQrWx` H4 Mono／`tAobl`・`C3lFU` Body Mono／`jC6W2` Body；`avhZw` SSE 膠囊）；`AX0nC` 軌道 330×6；`dQVOX` 用量一行（`wgBFX` Label muted／`qmVqX` Body 600 Mono／`uRhZf` Label／`o6T6j` Body Mono） |
| 佇列 | `qDBhI`（[4,16]、`space_between`：`Unc2c` BodyLg 600、`jbYHw` Secondary h44）；`Ruyx7`（[0,16]、gap 10） | 四列都是 `aw4Qr` 的 instance：`Qp7Cm` 完成（88 高）／`n79vZ` 進行中（**要改**）／`QIlJE` 排隊／`Ah8fN` 失敗（104 高，副標兩行） |
| 即時活動 | `rMD73`（[0,16]、gap 8） | `AfUUS` 標題列（[12,14]、`$bg-secondary`、`$radius-md`、無框線）：`KG1ac` activity 16、`S0hNLz` Body 600、`tpoAF` Label muted「· 自開啟本頁起累積」、`fdUFU` chevron-down 20 `$text-muted`；`mZsfW`（gap 8）：`H044hI` 提示 Label muted、`A55Y5` SSE 膠囊 |
| 其他 | `XQ0Hl` spacer（`fill_container`）、`NFeQz` tab-bar h80 | 不動 |
| F11-D-v2（對照） | `l8FsB` | 頁首 `jxF9O`[24,32,16,32]；麵包屑 `O6OQNE`（`mqvB6` chevron-right）；標題 `j7IMjB` H3；統計列 `whlu3` 橫排；進行中列 `OkdGK`：`OMD2y:{enabled:false}`、步驟條 `gkGTR`→`XkGvG`＋`fVBQF:{enabled:false}` |
| F3-M-v2（步驟條來源） | `k8sJl4` → `fS5is`（`m-stepper`，358×216，六列 `x0G3j`／`t6U0O0`／`XPXpx`／`YKDDs`／`GOutw`／`e4Ar7Q`，`E10kib` 停用） | 已是目標狀態（提取音訊完成、轉錄中進行、其餘 muted） |
| 規格註記 | 群組 `JzmvC`；6f-3 的 `oVRQd` 在 x=17040、y=35605、寬 300、實測高 202 | 新註記放 `oVRQd` 下方 40px → y=**35847**（動手前再 `Get` 量一次） |

字階（`DESIGN.md:351-360`）：H3 20（手機 18）、H4 18、BodyLg 16、Body 14、Label 12。間距：`2xs`2・`xs`4・`xs-plus`6・`sm`8・`sm-plus`10・`md`12・`md-plus`14・`lg`16。全檔 `problems` 現況 **69**（`DESIGN.md:836` 寫的 74 是舊值）；`PXB0z` 自己 0 個。

---

## Acceptance Criteria

1. **設計稿先改（Pencil MCP；每個節點動手前再 `Get` 一次）。只改 F11-M `PXB0z` 與一則規格註記。**
   - `n79vZ`（進行中列）：`OMD2y` → `enabled:false`；`lUZol` content →「轉錄中…」；**迷你步驟條 `G5v4u` 換成 F3-M 的直排步驟條**（內容與 `fS5is` 相同、`width:"fill_container"`；`E10kib` 維持 `enabled:false`）。📎 `G5v4u` 是 slot `dxgox` 被覆寫後的**實體節點**——直接對 `G5v4u` 這個 id 下 `Replace`／`Delete`＋`Copy`，不要走 `Update(n79vZ,{descendants:{dxgox:…}})`（slot 已被覆寫時會 `Node not found for override path`）。`Replace` 會把沒寫的屬性重設（`enabled` 會變 false）→ 明確補 `enabled:true`，並確認 `E10kib` 的副本仍是 `enabled:false`。改完這一列約 314 高（14+60+10+216+14），`XQ0Hl` spacer 是 `fill_container` 會自己吸收，`PXB0z` 維持 1500 高、`NFeQz` 仍貼底——改完用 `ctx.bounds` 驗。
   - `Qp7Cm` `lUZol` →「已完成，字幕已寫入檔案」；`Ah8fN` `lUZol` →「這部正在別處處理」（會變回一行，列高 88）。
   - `J79fp` →「活動 ／ 生成字幕」（與程式碼的分隔符一致；桌機稿的 chevron 分隔符另見 AC #9 的 disc）。
   - ⛔ 不改：12 / 38 與進度條寬（31.6%＝12/38，自洽；清單只畫得下四列是長頁面的正常情況）、標題「批次生成字幕」（**不是過時**——執行中模式程式碼就是這四個字，sprint 條目的轉述有誤）、兩顆 SSE 膠囊的 `$radius-sm`（`disc-2026-09-sse-chip-pill-radius`）、母版 `aw4Qr`／`XkGvG`／`YDPhc`／`S86VM`、桌機稿、F3-M。
   - 規格註記（`JzmvC`，樣式比照 `oVRQd`，名稱 `spec-note-dsr-6f-4`）：
     > 「F11（手機）：這是一頁不是抽屜——沒有把手與 ✕，出口是標題左邊的返回鍵（44×44 命中區，回『活動』）。左右內距 16。總覽卡直排三行：計數＋SSE 膠囊／滿寬進度條／本次用量一行。進行中那一列的六步進度條是**直排、與 F3-M 同一顆**（⚖️ 2026-09-21：全 app 手機只有一種步驟條），轉錄沒有百分比。**即時活動在手機預設收合**：整條標題列可按，收合時底下顯示提示＋SSE 膠囊（達上限時『已停止』那一行也留著）；展開後清單最高 320、自己捲、跟著最新一列，提示換回『僅狀態事件，不含逐字內容』。沒有手機稿的狀態（達上限／結束的底部列、閒置、中途接入、單部、載入中、錯誤、取消確認、即時活動展開）：沿用桌機元件、左右 16；底部列與取消確認的兩顆鈕等寬並排、句子獨佔一行。640 以上完全是桌機版面。」
   - 收尾：全檔 `problems` **不得超過 69**；存檔用 `osascript -e 'tell application "Pen" to activate' -e 'delay 1' -e 'tell application "System Events" to tell process "Pen" to click menu item "Save" of menu "File" of menu bar 1'`，再 `git status --porcelain ux-design.pen` 必須出現 ` M`，並 grep 磁碟檔確認新字串（例如 `spec-note-dsr-6f-4`）真的在裡面；**存檔後**才跑 `python3 scripts/export-pen-screenshots.py`；**只 stage** `_bmad-output/screenshots/flow-f-subtitle-v2/f11-m-v2.png` 與 `_bmad-output/pen-tokens.json`（`penSha256` 必變），其餘 `git checkout --` 還原。`SCREENS` 已有 `PXB0z`（`export-pen-screenshots.py:355`），不用加。
   - 📎 Pencil `execute` 失敗（含 throw）會把同一次呼叫裡的編輯整個 rollback——量測與寫入分兩次呼叫；讀值用 `Print(...)`。

2. **手機頁首：返回鍵＋標題＋藥丸一行，麵包屑在下（🔴 #1–#5）。** 全部 `max-sm:`／`sm:hidden`，桌機一個像素都不能變。
   - `GenerationWorkspaceV2Props` 新增 `onBack?: () => void`；`GenerationWorkspaceProps`（容器）同名透傳；`ActivityHub.tsx:308-317` 傳入「回到 `/activity`（沒有 `view`）」的導頁。📎 該檔今天只 import `Link, getRouteApi`（`:15`），沒有任何 `useNavigate`——新增 `const navigate = routeApi.useNavigate();`（`routeApi` 在 `:293`），`onBack={() => void navigate({ to: '/activity', search: {} })}`；`search: {}` 符合 `routes/activity.tsx:7-17` 的 `{ view?: 'generation' }`。用 push，不要 replace。沒有 `onBack` 就不畫返回鍵（gallery 夾具與既有 spec 不受影響）。
   - 返回鍵：`<button type="button" aria-label="返回活動" data-testid="workspace-back" className="… sm:hidden">`，`ChevronLeft` 24（`h-6 w-6`，`aria-hidden`）、`text-[var(--text-secondary)]`；**命中區 44×44**（`h-11 w-11`），**四周多出來的 10px 用負 margin 吃回去：`-my-2.5 -ml-2.5`**——左邊讓圖示的左緣仍貼齊 16 的內距（稿 `d86Ux` x=16），上下讓標題列不被 44 撐高（稿這一列約 27 高）。圖示到標題的視覺距離依稿 10（按鈕右側的 10px 留白已經提供，`<h1>` 不必再加 margin；以 e2e 量「圖示右緣到標題左緣 ≈10」為準）。
   - 標題列（`:602`）在手機：返回鍵＋`<h1>` 在左、`StatusPill` **靠右**——`:602` 加 `max-sm:justify-between`，返回鍵（第一個孩子）與 `<h1>` 包進一個 `flex items-center sm:contents` 的內層（桌機上這一層消失，`:602` 的孩子仍是 h1＋藥丸、`gap-2` 照舊；返回鍵本身 `sm:hidden`）；`<h1>` 加 `max-sm:text-lg`；藥丸圓點加 `max-sm:h-1.5 max-sm:w-1.5`。
   - 麵包屑在手機排到標題列**下面**。🚨 頭區 `:591` 是**同一個** `flex-col`，孩子有四個：麵包屑、標題列、`BudgetBanner`、`OverallStrip`（`:592-616`）——替麵包屑加正的 `order` 會把它丟到**最後面**（總覽卡之後）。正確做法：**標題列 `:602` 加 `max-sm:order-first`，麵包屑不給 order**（留在 order 0、DOM 順序仍在橫幅與總覽卡之前）。兩行之間稿是 2px：麵包屑加 `max-sm:-mt-3`（14−12＝2）；⛔ 不要為了這 2px 把兩者包進一個 div（會改到桌機的 DOM 與 gap）。麵包屑另加 `max-sm:text-xs max-sm:text-[var(--text-muted)]`。⚠️ 第二段 `<span>`（`:600`）帶著 `text-[var(--text-primary)]`——手機依稿整條 muted，要一起蓋掉。分隔符「／」不動。⛔ 本張**不把**「活動」改成連結（會動桌機；有返回鍵就夠了）。
   - 內距：頭區 `:591` 加 `max-sm:px-4 max-sm:pt-3 max-sm:pb-3.5`（`gap-3.5` 本來就是 14，不用再加；`pb-3.5` 讓頭區與身體之間是稿的 14）；身體 `:620` 加 `max-sm:px-4 max-sm:gap-3.5`；底部列 `:800` 加 `max-sm:px-4`。
   - 橫幅 `BudgetBanner`、`workspace-data-error`、`workspace-idle`、`workspace-attach`、`workspace-single`、skeleton：**只吃到左右 16 的改變**，本身的 class 不動。

3. **總覽卡在手機直排三行（🔴 #6–#8）。** 只准 `max-sm:`；⛔ 不准把卡片畫兩次（🔴 #18 的同一個理由：`workspace-overall`、`role="progressbar"` 都被既有測試以「唯一」取用）。
   - 目標版面：①「已完成 N / M 部」在左、SSE 膠囊在右，**同一條水平線** ②進度條**滿寬**（卡片內寬） ③「本次用量 $x ／上限 $y」**一行**（小標與金額同一條基線）。「整批進度」小標在手機不顯示。
   - 建議做法（6f-2 已實測 `max-sm:contents`＋`order` 在 Tailwind 4.1.18 可行、639／640 沒有死角）：第一個小欄 `:108` 加 `max-sm:contents`，它的三個孩子變成根的 flex item——小標 `max-sm:hidden`、計數列 `max-sm:order-1`、進度條 `max-sm:order-3 max-sm:w-full`；膠囊 `max-sm:order-2 max-sm:ml-auto`；用量小欄 `:135` 加 `max-sm:order-4 max-sm:w-full max-sm:flex-row max-sm:items-baseline max-sm:gap-1`。根 `:106` 加 `max-sm:gap-x-2 max-sm:gap-y-2 max-sm:px-3.5`。`SseChip` 被兩處共用——**只准**替 `SseChip` 加一個 `className?: string`（用 `cn()` 併入），在總覽卡那一處傳 `max-sm:order-2 max-sm:ml-auto`；⛔ **不要包一層**（包裝層會變成一個 block 化的 flex item，行內基線的下緣空間可能改變桌機那一列的高度 → `running` 1200 基準線會變）。**`ml-auto` 一定要帶 `max-sm:`**：桌機上膠囊是那一列最後一個，不帶前綴會被推到最右邊。📎 `:106`／`:108`／`:122`／`:135` 都是純字串（沒過 `cn()`），base 與 `max-sm:` 兩者都會進樣式表，`max-sm:` 排在 base 之後 → `max-sm:w-full` 贏過 `w-[180px]`、`max-sm:gap-1` 贏過 `gap-1.5`。
   - 字級（只在手機）：完成數 `max-sm:text-lg`、`/` 與總數 `max-sm:text-sm`、「部」`max-sm:text-sm`、兩個金額 `max-sm:text-sm`。兩個 `text-[11px]` 小標**凍結不動**（`disc-2026-09-11px-micro-label-not-on-type-scale`）。
   - `role="progressbar"`、`aria-label="整批生成進度"`、`aria-value*` 原封不動。

4. **佇列：取消確認與底部列在手機排得下；副標可以換行（🔴 #12–#14）。**
   - 標題列 `:719`：加 `max-sm:flex-wrap`。`CancelAll` 確認態 `:881`：加 `max-sm:w-full`；兩個 `<span>`（確認句、錯誤句）各加 `max-sm:w-full`；「繼續生成」「確定取消」各加 `max-sm:flex-1 max-sm:justify-center`。單顆「全部取消」不動（稿 `jbYHw` 就是靠右 h44）。⚠️ 焦點搬移（`pendingFocusRef`）、`aria-disabled` 忙碌態、`取消中…` 文案一律不動。
   - 底部列 `:800`：`workspace-dismiss-error` 加 `max-sm:w-full`（它有 `mr-auto`）；「關閉」「下次繼續」各加 `max-sm:flex-1 max-sm:justify-center`。只有「關閉」一顆時它滿版（`flex-1`）——可接受（6f-3 裁定 3 的同一套）。
   - `QueueRow` 副標 `:276`：加 `max-sm:whitespace-normal`（`truncate` 的其餘兩條無害）。⛔ 標題那個 `<span>`（`:271`）的 class **一個都不准動**（🔴 #22）；`sm:pl-[54px]`、`GenerationProgressV2`、徽章、海報不動。

5. **即時活動在手機預設收合；展開後清單自己捲（🔴 #15–#19）。**
   - `EventLogPane` 加 `const [open, setOpen] = useState(false)`。**這個 state 只透過 `max-sm:` 類別起作用**，所以 ≥640 的畫面與它無關（不需要 `matchMedia`）。
   - **開關**：在標題列 `:410` 裡加一顆 `<button type="button" data-testid="workspace-log-toggle" aria-expanded={open} aria-controls={清單的 id} aria-label="即時活動事件清單" className="… sm:hidden">`（**名稱固定**，狀態交給 `aria-expanded`——名稱跟著翻成「收合…」會被唸成「收合即時活動，已展開」；先例 `dashboard/CollapsibleSection.tsx:47-48`），`ChevronDown` 20（`h-5 w-5`、`text-[var(--text-muted)]`、`aria-hidden`），展開時 `rotate-180`，轉場 `transition-transform duration-[var(--motion-state)] motion-reduce:transition-none`。按鈕自己 44×44、靠右（`ml-auto`），並用 `after:absolute after:inset-0`（標題列加 `relative`）讓**整條標題列**都是它的命中區（稿的整條 `AfUUS` 看起來就是可按的）。🚨 **按鈕自己不准是 `relative`**（`relative` 只加在標題列上），否則 `::after` 會縮回 44×44。展開時 `onClick` 裡先把 `pinnedRef.current = true` 再 `setOpen(true)`——讀者若先前往上捲過（`:425` 會把它設成 false），而 `display:none` 可能把 `scrollTop` 歸零又不發 scroll 事件，重新展開就會停在最上面、也不再跟著最新一列。⛔ 展開時**不要**自動捲動整頁（不加 `scrollIntoView`）。⛔ 不要把標題列畫兩次——`GenerationWorkspaceV2.spec.tsx:560` 用 `getByTestId('workspace-log-header-icon')` 取圖示，兩份會 multiple-match。（`:213-215` 是 `toHaveTextContent`，不會 multiple-match；沒有任何測試在全域數按鈕、斷言標題列孩子的順序或 `<ol>` 的 className。）
   - 標題列在手機：`max-sm:h-auto max-sm:min-h-[47px] max-sm:rounded-[var(--radius-md)] max-sm:border-b-0 max-sm:bg-[var(--bg-secondary)]`；圖示 `max-sm:h-4 max-sm:w-4`；「自開啟本頁起累積」在手機緊跟標題（`max-sm:ml-0`，前面補一個 `aria-hidden` 的「·」，只在手機顯示）。外層 `<aside>` `:408` 在手機**沒有外框與底色**（`max-sm:border-0 max-sm:bg-transparent max-sm:overflow-visible max-sm:gap-2`）；展開後的清單自己帶框（`max-sm:rounded-[var(--radius-md)] max-sm:border max-sm:border-[var(--border-subtle)]`）。
   - **清單 `<ol>` `:419`**：補 `id`；收合時 `cn(!open && 'max-sm:hidden')`——**`display:none`，不是卸載**；展開時 `max-sm:max-h-80`（320）＋既有的 `overflow-y-auto`，所以清單**自己捲**。自動跟到底的 `useLayoutEffect`（`:397-400`）的依賴加上 `open`：展開的那一刻要落在最新一列（收合期間 `scrollHeight` 是 0，`pinnedRef` 仍是 true）。`onScroll`、「距底 ≤ 40px」的判定不動。
   - **頁尾 `:438-456`**：手機加一行提示 `<p data-testid="workspace-log-hint" className="text-xs text-[var(--text-muted)] sm:hidden">展開查看即時事件（不含逐字內容、無時間戳）</p>`，**只在收合時**可見——用 **HTML 屬性** `hidden={open}`（jsdom 沒有 Tailwind 樣式表，unit 只能斷言屬性與 class token，⛔ 不要在 unit 用 `toBeVisible`）；既有的「僅狀態事件，不含逐字內容」在手機**收合時**藏起來（`cn(!open && 'max-sm:hidden')`；提示已經講了同一件事）、展開時照舊。頁尾第一列（`:439`，膠囊＋那句話）在「收合且 `connected === false`」時是空的卻還佔著 `gap-2`——那個情況替它加 `max-sm:hidden`。SSE 膠囊與「已停止（達預算上限）」那一行**收合與展開都看得到**（批次層的事實不能被收進去）。頁尾在手機：`max-sm:border-t-0 max-sm:px-0 max-sm:py-0`，提示在上、膠囊在下（稿 `mZsfW` gap 8）。⚠️ `workspace-sse-chip` 的數量不准變（🔴 #18）。
   - 🚨 **`workspace-log-announcer` 永遠掛著、永遠不在被藏起來的容器裡**（它是 `<aside>` 的直接子項，與 `<ol>` 是兄弟——維持這樣）。收合時結果照樣要被唸出來。
   - `showFeed` 的條件、`feedRowView()`、`useGenerationJobsFeed`、兩條 EventSource **一律不動**。

6. **Rule 21**：`GenerationWorkspaceV2.tsx:2` 的 `// Design ref:` 行尾接 ` · F11-M-v2 (PXB0z)`（沿用該行既有的分隔寫法）。

7. **既有的行為不准回歸。**
   - 桌機（≥640）：`generation-workspace-v2/*` 四張 1200 寬基準線**零變動**；`tests/e2e/generation-workspace.spec.ts` 五條在 `chromium` 全綠，**斷言不改**（只准替第 5 條加「viewport 寬度 <1024 就 skip」，見 🔴 #21）。
   - `GenerationWorkspaceV2.spec.tsx`（45 條）、`GenerationWorkspaceContainer.spec.tsx`（26 條）、`generationWorkspace.spec.ts`（11 條）、`ActivityHub.spec.tsx` 全綠，既有斷言一條都不改寫。
   - ⛔ 不改 `GenerationProgressV2.tsx`、`generationQueueRow.ts`、`generationEventCopy.ts`、`generationWorkspace.ts`、任何 hook、`ui/*`、`shell/*`、任何後端檔案；不動 11px 與 SSE 膠囊圓角。

8. **測試。** 分「**紅**」與「**守**」，誠實標示（Rule 16）。jsdom 不看 media query——unit 只能驗 token 與 state，版面由真瀏覽器守。
   - `GenerationWorkspaceV2.spec.tsx`：
     - （紅）有 `onBack` 才有 `workspace-back`，無障礙名稱「返回活動」，點了呼叫一次；沒有 `onBack` 就沒有這顆。
     - （紅）`workspace-log-toggle` 預設 `aria-expanded="false"`、`aria-controls` 指到的 id 就是那個 `role=list`（名稱「生成事件日誌」）；點一下變 `"true"`、清單 class 不再含 `max-sm:hidden`；再點一下回來。
     - （紅）收合與展開兩種狀態下，`workspace-log-announcer` 是**同一個 DOM 節點**，且 `document.querySelectorAll('[aria-live]')` 仍恰好 1 個；收合時新進來一列終態事件，announcer 的文字照樣更新。
     - （紅）收合時 `workspace-log-hint` **沒有** `hidden` 屬性、展開時 `toHaveAttribute('hidden')`；「僅狀態事件…」那個元素收合時帶 `max-sm:hidden`、展開時不帶（`:215` 的 `toHaveTextContent` 照樣綠）；達上限時「已停止（達預算上限）」兩種狀態都在。
     - （紅）展開的那一刻 `scrollTop` 被設到 `scrollHeight`（沿用 `:830-846` 塞假 `scrollHeight` 的做法）；（紅）往上捲 → 收合 → 再展開，`scrollTop` 仍回到 `scrollHeight`（`pinnedRef` 被重設）。
     - （紅）token：`<h1>` 帶 `max-sm:text-lg`；進度條帶 `max-sm:w-full`；取消確認的兩個 `<span>` 帶 `max-sm:w-full`；底部列兩顆鈕帶 `max-sm:flex-1`。
     - （守）`:110-111` 佇列 `<ul>` 零按鈕、`:355/363` 膠囊數量、`:815-817`、列標題 class。
   - `ActivityHub.spec.tsx`（真 router）：🚨 這支 spec 把工作區**整個 mock 掉了**（`:31-35`：`GenerationWorkspace: ({ active }) => <div data-testid="generation-workspace-stub" …/>`），裡面沒有 `workspace-back`。做法：**只擴充那個 stub**（既有斷言不動）——`({ active, onBack }) => <div data-testid="generation-workspace-stub" …>{onBack && <button data-testid="workspace-back-stub" onClick={onBack} />}</div>`；（紅）`renderHub('/activity?view=generation')` → 點 `workspace-back-stub` → `generation-workspace-stub` 不見、活動列表（`activity-empty` 或該檔既有的列表 testid）出現。
   - `GenerationWorkspaceContainer.spec.tsx`（它**沒有** mock V2，`vi.mock` `:43-91` 只蓋 hook 與 service）：（紅）容器把 `onBack` 透傳到 `workspace-back`。`renderWorkspace()`（`:144`）今天不收參數——替它加一個可選的 `props`，既有呼叫端不用改。
   - **每一項修法做 mutation check**（拿掉 → 必須有測試變紅；6d-c-1／6d-c-2 的慣例），結果寫進 Completion Notes。
   - ⚠️ 不要寫不可能失敗的斷言：會換行的容器量「沒有橫向溢出」永遠成立（6f-2 CR H1）——量**同一條水平線**、**寬度**、**高度上界**。

9. **真瀏覽器的驗證＝新的 e2e `tests/e2e/generation-workspace-mobile.spec.ts`**（`@e2e @generation-workspace-mobile`）。
   - import 照手機 e2e 的先例：`import { test, expect } from '../support/fixtures'`（`batch-consent-mobile.spec.ts:26`）；舊 spec 維持 `@playwright/test` 的 import，除非抽共用 stub。
   - stub：把 `generation-workspace.spec.ts` 的 `ITEMS`／`snapshot`／`stubCommon`／`sseFrame`／`jsonOk` 搬到 `tests/support/helpers/generation-workspace-stubs.ts` 兩邊共用（**不准改既有五條的行為與斷言**），或照抄；SSE 照既有寫法「第一條連線給 frame、其後 `route.abort()`」。
   - `beforeEach`：非 `chromium` project 就 `test.skip`（像素斷言；6f-3 CR L7）。`const PHONE = { width: 390, height: 844 }`、`const AT_BREAKPOINT = { width: 640, height: 844 }`、`const GUTTER = 16`；box 一律 `Math.round`（`-0`）。🚨 **所有水平位置都量「相對於 `generation-workspace` 根節點」**（`el.x − root.x`、`root.width − el.width`），不要量絕對 x——≥640 時 `AppShellV2.tsx:64` 的側欄會出現（`AppSidebar` 預設 `w-60`），640 寬時內容欄只剩 400、總覽卡的絕對 x 是 272 不是 32。展開的轉場量之前先 `Promise.allSettled(el.getAnimations().map(a => a.finished))`。
   - **390・執行中**：
     - `workspace-overall`：`x − root.x`＝16、寬＝358（**紅**：今天 24／342）。
     - `workspace-back` 可見、≥44×44、圖示左緣 ≈16；它與 `<h1>` 的垂直中心差 <2px；`<h1>` 的 `font-size` 是 `18px`（紅）；`workspace-status-pill` 與 `<h1>` 同一條水平線且右緣 ≈374（紅）；麵包屑的 `y` > 標題列的底邊（紅）。點返回 → URL 不再有 `view=generation`、`generation-workspace` 消失。
     - 總覽卡：`role=progressbar` 的寬 ≈**328**（358−2 邊框−28 內距，`box-sizing:border-box`；容差 ±1）（紅：今天 180）；SSE 膠囊與「已完成」計數同一條水平線、膠囊右緣 ≈359（16+358−1−14）；用量那一行在進度條**下面**，且「本次用量」與金額同一條水平線（差 <4px）。
     - 即時活動：`workspace-log-toggle` 可見、`aria-expanded="false"`、自己的 box ≥44×44（📎 `boundingBox()` **不含** `::after`，所以量不到「整條 358」）；整條標題列可按改用行為驗：`log.getByText('即時活動').click()`（點在最左邊的文字上，落點是按鈕的 `::after`）→ `aria-expanded` 變 `"true"`；可再加 `document.elementFromPoint(標題列左緣+8, 中心y)` 就是那顆按鈕；`role=list` **不可見**、`workspace-log-hint` 可見。送 40 列事件 → 點開 → 清單可見、**高度 ≤ 320**、`{ scrolls: true, atBottom: true }`（與既有第 5 條同一個量法）、`ol.scrollWidth <= ol.clientWidth`（📎 這條是有牙的：事件列是不換行的 `flex`，溢出就真的會撐寬）；整頁 `document.documentElement.scrollWidth <= 390`。再點一下 → 清單不可見。
   - **390・取消確認**：點「全部取消」→ 確認句獨佔一行；「繼續生成」「確定取消」同一條水平線、寬度相等（差 ≤1px）、兩顆都在 16–374 之間。
   - **390・達上限**：沒有現成的 e2e stub 可抄（`tests/e2e/` 裡只有 `batch-subtitle.spec.ts:29` 的註解提到）——用 status 回 `{ running:false, progress:null, last: snapshot({ status:'budget_ceiling', paused_count:1, current_media_id:'', items:[…其中一筆 'paused'] }) }`（`attachSnapshot` 會原樣吃下，`useGenerationBatchProgress.ts:269-271`）。`workspace-close`／`workspace-resume` 同一條水平線、等寬（≈173）；**先 `await page.evaluate(() => window.scrollTo(0, document.documentElement.scrollHeight))` 再量**：底部列的底邊 ≤ `mobile-tab-bar` 的頂邊（760）。⛔ 不要用 `scrollIntoViewIfNeeded`——它會把底部列停在視窗最底（y≈844），剛好在 fixed 的分頁列底下，正確的頁面也會被量成紅的；「已停止（達預算上限）」在收合狀態下可見。
   - **640（斷點另一側）**：`workspace-back`、`workspace-log-toggle`、`workspace-log-hint` 都不可見；`role=list` 可見（即使從沒按過展開）；`workspace-overall` 的 `x − root.x`＝32 且 `root.width − overall.width`＝64；進度條寬 180；`<h1>` `20px`；麵包屑在標題**上面**。
   - 本機跑要 `AI_PROVIDER=claude npx playwright test tests/e2e/generation-workspace-mobile.spec.ts --project=chromium`（`preexisting-fail-e2e-local-ai-provider`）；過了再 `--repeat-each=3`。
   - dev-story Step 9（UX 比對）：用 e2e 裡的 `page.screenshot()`（390、執行中／展開／達上限三張，**不是基準線**，附在 PR 說明）對 `f11-m-v2.png` 逐項核。⛔ 不新增 `generation-workspace-v2/*-mobile` 視覺夾具（🔴 #20、裁定 4）。

10. **另立的單子（建單時已寫入 sprint-status）**：`disc-2026-09-workspace-log-uncapped-640-1023`（640–1023 單欄時即時活動沒有高度上限、不會自己捲）、`disc-2026-09-f11-pen-row-copy-and-breadcrumb-stale`（桌機稿 `KZmNG` 的列文案與麵包屑分隔符與程式碼不一致）；補記 `disc-2026-09-flow-f-mobile-missing-state-screens`（F11-M 缺：達上限／結束、閒置、中途接入、單部、取消確認、即時活動展開）。

11. **CI 全綠**：`pnpm run format:check`、`pnpm run lint:all`、`pnpm nx run web:typecheck --skip-nx-cache`、`python3 scripts/check-design-tokens.py`、`pnpm nx test web`、`pnpm nx test api`。⛔ 絕不 `run_in_background` 跑測試；Nx 一律 `NX_DAEMON=false`；跑完 api 測試把 `apps/api/coverage/` 刪掉（`disc-2026-09-api-coverage-not-gitignored`）。🚨 合併之後要看 `main` 那一次的 Tests／Docker／Visual Regression 三條。本張**不應該**產生任何新的或變動的視覺基準線——視覺 shard 若紅，先當成回歸查，不要去 bootstrap。

## Tasks / Subtasks

- [x] **Task 1 — 設計稿：F11-M 的進行中列換成 F3-M 步驟條、三句文案、麵包屑、規格註記（AC: #1）**
- [x] **Task 2 — 手機頁首：`onBack`＋返回鍵、標題列／麵包屑順序、16 內距（AC: #2, #6, #8）**
  - [x] 先寫 `workspace-back` 與 `ActivityHub` 的紅測試 → 實作
- [x] **Task 3 — 總覽卡直排三行（AC: #3, #8）**
- [x] **Task 4 — 取消確認、底部列、副標換行（AC: #4, #8）**
- [x] **Task 5 — 即時活動收合（AC: #5, #8）**
  - [x] 先寫 announcer「同一個節點＋恰好一個 aria-live」在收合／展開下的紅測試 → 實作
- [x] **Task 6 — 手機 e2e、既有 e2e 第 5 條的 viewport<1024 skip、mutation check、收尾（AC: #7, #9, #10, #11）**
  - [x] dev-story Step 9：`f11-m-v2.png` 對 e2e 截圖與量測

## Dev Notes

### 這張的重點

- **全部是 `max-sm:`／`sm:hidden` 加法。** 唯二的新東西是一個 `onBack` prop 與 `EventLogPane` 裡的一個 `open` state；兩者在 ≥640 都看不到。
- **最容易踩的雷是 `aria-live`**：收合要用 `display:none` 藏**清單**，announcer 留在外面。用 `{open && <ol>}` 或把整個 `<aside>` 條件渲染，無障礙就真的壞了，而且 `:815-817` 會紅。
- **不准「畫兩次」。** 6f-3 的批次對話框可以畫兩份用量列，這裡不行：`workspace-overall`、`progressbar`、`workspace-sse-chip` 的數量、「即時活動」的文字，都被既有測試當成唯一。用 `max-sm:contents`＋`order` 在原地重排。
- **稿不是都對的，sprint 條目的轉述也不是都對的。** 標題「批次生成字幕」沒有過時；稿的迷你步驟條牴觸 9/21 的裁定（改稿）；失敗副標「會被截斷」以今天的文案不成立（防呆，標「守」）。動手前看節點值與程式碼，不要照轉述做。
- **這張沒有視覺基準線會變。** 手機版面完全靠 e2e；桌機四張基準線零變動是回歸的證明。

### 上游契約（Rule 20 ack）

- 本張不消費也不改任何線上契約（純呈現層）。e2e 的假資料沿用 `generation-workspace.spec.ts` 既有的形狀（`progress.items[]`／`last`，dsr-6d-a 的未 stamp AC＝implicit v0），不自創欄位。

### 建單裁定（2026-09-21，Sally／Alexyu 可在 review 推翻）

1. ⚖️ **六步進度條在手機直排、同 F3-M**（Alexyu 2026-09-21，dsr-6f-3 建單當下）→ F11-M 的迷你步驟條**改稿不改碼**。
2. ⚖️ **斷點用 `sm`（640），不是 `lg`（1024）。** 稿是「手機」，前三張也都以 640 為界；用 `max-sm:` 才能保證 640 以上零變動、四張桌機基準線不動。代價：640–1023（單欄但不是手機）的即時活動仍然沒有高度上限——那是今天就存在、沒有稿的問題，另立 `disc-2026-09-workspace-log-uncapped-640-1023`，不夾帶。
3. ⚖️ **收合＝藏清單（`display:none`），不是卸載；展開後清單最高 320、自己捲、跟著最新一列。** 稿沒畫展開後的樣子；320 約是 8 列，與桌機「清單自己捲」同一個行為（dsr-6d-c-2 CR H3），且不會把佇列推到看不見。用規格註記交代（6f-2 先例），不補狀態稿；缺稿已補記到 `disc-2026-09-flow-f-mobile-missing-state-screens`。
4. ⚖️ **不新增手機視覺夾具。** 工作區不是 Portal，頁內 390 夾具實寬 326 且會拍到分頁列（🔴 #20）；6f-2 CR M7 已裁定不把假版面收成基準線。手機版面由 e2e 守（與 6f-3 挑片畫面同一條裁定）。
5. ⚖️ **返回鍵用 `onBack` prop，不在呈現元件裡直接用 router 的 `<Link>`。** `GenerationWorkspaceV2` 今天不依賴 router，45 條 spec 與 4 個 gallery 夾具都是裸 render；導頁留在已經在 router 裡的 `ActivityHub`。
6. ⚖️ **麵包屑「活動」不改成連結、分隔符不動。** 會動桌機；手機已有返回鍵，桌機有側欄。稿與碼的分隔符差異另立 disc。
7. ⚖️ **沒有手機稿的狀態**：只調左右 16；兩顆鈕的地方等寬並排、句子獨佔一行（6f-3 裁定 3 的同一套）。
8. ⚖️ **SSE 膠囊圓角、11px 仍不在本張**（`disc-2026-09-sse-chip-pill-radius`、`disc-2026-09-11px-micro-label-not-on-type-scale`）。

### 不要做的事

- 不要用 `MOBILE_SHEET_*`；不要加把手、✕、`sheet-enter`。
- 不要用 `matchMedia`／新的 `useIsMobile` hook——`max-sm:` 類別就夠，而且 jsdom 的 `matchMedia` 是假的（`test-setup.ts:75-90`）。
- 不要卸載 `EventLogPane`、`<ol>` 或 announcer；不要多生第二個 `aria-live`；不要多生第二顆 `workspace-sse-chip`。
- 不要動列標題 `<span>` 的 class；不要在佇列 `<ul>` 裡放任何 button。
- 不要改 `GenerationProgressV2.tsx`、`generationQueueRow.ts`、`generationEventCopy.ts`、任何 hook。
- 不要新增 `*-mobile` 視覺夾具；不要本機產 `-linux.png`；不要為了視覺 shard 紅就去 bootstrap（本張不該有基準線變動）。
- 不要改既有 e2e／spec 的斷言（第 5 條 e2e 只准加 viewport<1024 的 skip；`ActivityHub.spec.tsx` 只准擴充 stub）。
- 不要加任何 `fixed`／`sticky` 的手機底部元素（會跟 84 高的分頁列打架）。

### 已知陷阱

- **`max-sm:contents`＋`order`**：父層沒有 `contents`，孫層的 `order` 是空話；6f-2 已用 repo 的 Tailwind 4.1.18＋tailwind-merge 3.4.0 實編確認產生順序 `base → max-sm: → sm:`，639／640 沒有死角。
- **`ml-auto`／`flex-1`／`justify-end` 會讓滿版失效**：沒滿版先看外面那一層（6f-1／2／3 各踩一次）。`workspace-dismiss-error` 的 `mr-auto`、用量小欄、膠囊的 `ml-auto` 都在這一類。
- **`SseChip` 是共用的**：`order`／`ml-auto` 只加在總覽卡那一處，用 `className` prop、不要包一層、`ml-auto` 一定帶 `max-sm:`。
- **頭區是一個四個孩子的 `flex-col`**：要讓麵包屑到標題下面，是把**標題列** `order-first`，不是替麵包屑加 order。
- **`::after` 命中區**：按鈕不能 `relative`；`boundingBox()` 量不到 `::after`。
- **≥640 有側欄**：e2e 的水平位置一律量相對於工作區根節點。
- **fixed 的分頁列**：量底部列之前用 `window.scrollTo` 捲到頁尾，不要用 `scrollIntoViewIfNeeded`。
- **`cn()` 會用 twMerge**：`truncate` 與 `max-sm:whitespace-normal` 不衝突；但同一組屬性不帶前綴的舊值可能被丟掉——加完看真瀏覽器的 computed style。
- **`toHaveTextContent` 是子字串比對**（「完成」會命中「批次完成」）；新測試一律用 testid／role＋name。
- **jsdom 完全看不到斷點**：class 斷言不論功能在不在都可能綠 → 行為由 e2e 守；unit 只驗 state、ARIA、token 是否存在。
- **e2e**：`route.fulfill` 對 `EventSource` 的 body 一結束串流就斷 → 只有第一條連線給 frame，其後 `route.abort()`；`fulfill({ json })` 不要同時傳 `response`；`expect(box.x).toBe(0)` 遇到 `-0` 會紅 → `Math.round`；本機要 `AI_PROVIDER=claude`；`webkit-core` 有自己的測試集合，新 spec 不在裡面。
- **Pencil**：失敗會 rollback 同一次呼叫的編輯；`Replace` 會重設沒寫到的屬性（`enabled`）；slot 已被覆寫時對巢狀節點的 id 直接下手；存檔走選單 Save＋驗磁碟內容（⌘S 不可靠；`get_screenshot` 讀的是記憶體不是磁碟）；匯出全量會 re-render 近 200 張，只 stage 一張。
- **`test:cleanup` 會砍掉 `nx serve web`**；紅綠迴圈用 `cd apps/web && npx vitest run src/components/subtitle/GenerationWorkspaceV2.spec.tsx`，跑完 `scripts/cleanup-test-processes.sh --all`。
- **行號以建單時為準**（2026-09-21，main `aee223d0`）。

### Source tree

```
ux-design.pen、_bmad-output/pen-tokens.json、_bmad-output/screenshots/flow-f-subtitle-v2/f11-m-v2.png  ← Task 1
apps/web/src/components/subtitle/GenerationWorkspaceV2.tsx(+spec)                                      ← Task 2–5
apps/web/src/components/subtitle/GenerationWorkspaceContainer.spec.tsx（onBack 透傳）                   ← Task 2
apps/web/src/components/activity/ActivityHub.tsx(+spec)                                                ← Task 2
tests/e2e/generation-workspace-mobile.spec.ts（新）                                                     ← Task 6
tests/support/helpers/generation-workspace-stubs.ts（新，若抽共用 stub）                                 ← Task 6
tests/e2e/generation-workspace.spec.ts（只准：改 import、替第 5 條加 viewport<1024 的 skip）                      ← Task 6
_bmad-output/implementation-artifacts/sprint-status.yaml                                               ← 收尾
```

### Cross-Stack Split Check

後端 task **0 個**，前端／設計／測試 task 6 個 → 不觸發跨棧拆分。規模：一個元件檔（＋`ActivityHub` 三行）、一張稿、一支 e2e、零視覺基準線——**比 `dsr-6f-3`（6 個元件、兩個新夾具、一張重拍）小**，不再拆。

### Time-dependent visual coverage

- **N/A — no wall-clock-reading components touched.** `GenerationWorkspaceV2.tsx`、`generationEventCopy.ts` grep `Date.now|new Date|performance.now` 零命中；`useGenerationJobsFeed.ts:31` 明文「Row keys use a monotonic `seq`, never `Date.now()`」；事件紀錄刻意沒有時間戳（Rule 23-clean）。本張也不新增任何夾具。

### References

- [Source: `apps/web/src/components/subtitle/GenerationWorkspaceV2.tsx:1-3, 88-162, 169-193, 237-315, 373-459, 588-617, 620, 718-731, 796-832, 842-931, 942-1116`]
- [Source: `apps/web/src/components/activity/ActivityHub.tsx:145, 308-317`；`apps/web/src/routes/activity.tsx:12-19`；`apps/web/src/components/shell/AppShellV2.tsx:62-110`；`shell/MobileTabBar.tsx:24-38`]
- [Source: `GenerationWorkspaceV2.spec.tsx:110-111, 213-219, 300-303, 345-363, 815-817, 830-846`；`GenerationWorkspaceContainer.spec.tsx:522`；`ActivityHub.spec.tsx:80-82`（真 router 的 render helper）]
- [Source: `tests/e2e/generation-workspace.spec.ts:17-86, 196-205, 218-278`；`tests/e2e/batch-consent-mobile.spec.ts:35-87`、`glossary-mobile.spec.ts`、`manage-subtitle-mobile.spec.ts`（手機 e2e 的寫法）；`playwright.config.ts:105-181`]
- [Source: `apps/web/src/routes/test/-gallery.fixtures.tsx:396-478, 5432-5632`；`routes/test/gallery.tsx:298, 370, 396-397`；`tests/visual/components.visual.spec.ts:119-121, 274-285, 343-352`；`routes/test/gallery-fixture-viewport.spec.ts:12-25`]
- [Source: `apps/web/src/components/dashboard/CollapsibleSection.tsx`、`apps/web/src/components/subtitle/consent/CandidateListPanel.tsx:538-566`、`subtitle/ManageSubtitleDialogV2.tsx:741-753`（手寫收合的先例）；`generationQueueRow.ts`（列文案）]
- [Source: `ux-design.pen` `PXB0z`／`e6nOTG`／`CarVq`／`d86Ux`／`sSwG5`／`htLc2`／`J79fp`／`QNVCS`／`v6RDo`／`AX0nC`／`dQVOX`／`qDBhI`／`Ruyx7`／`Qp7Cm`／`n79vZ`／`G5v4u`／`OMD2y`／`UmgMX`／`QIlJE`／`Ah8fN`／`rMD73`／`AfUUS`／`fdUFU`／`mZsfW`／`H044hI`／`A55Y5`／`l8FsB`／`KZmNG`／`OkdGK`／`fS5is`／`E10kib`／`aw4Qr`／`JzmvC`／`oVRQd` —— SM 建單時以 Pencil MCP 逐節點讀出（2026-09-21）；程式碼現況由兩個唯讀稽核代理查證]
- [Source: `DESIGN.md:296-310`（狀態色、金錢是事實）、`:351-360`（字階）、`:514-516`（動＝正在發生）、`:604-622`（手機規則、44×44）、`:650-654`（徽章）、`:836`（`problems`；現況 69）]
- [Source: `dsr-6f-1-…md`（`viewport` 夾具、CR L8 分頁列、CR M1 326 寬）、`dsr-6f-2-glossary-mobile.md`（`max-sm:contents` 實測、CR H1 假斷言、CR M7 刪夾具）、`dsr-6f-3-batch-and-consent-mobile.md`（⚖️ 步驟條直排、裁定 3／5、CR L7 project skip）、`dsr-6d-c-1-workspace-queue.md`、`dsr-6d-c-2-workspace-event-log.md`（CR H3 清單自己捲、唯一 aria-live、M4）]
- [Source: project-context.md#Rule 16／#Rule 20／#Rule 21／#Rule 23／#Rule 24；`.claude/memory/feedback_measure_the_breakpoint_you_return_to.md`、`feedback_verify_pen_saved_before_commit.md`、`project_pen_schema_gotchas.md`]

## Dev Agent Record

### Agent Model Used

Claude Fable 5.1（claude-fable-5-1）— dev-story, Amelia

### Debug Log References

- **e2e：`elementFromPoint` 吃的是視窗座標。** 佇列一長，即時活動的標題列落在摺線下、剛好在 fixed 的分頁列底下，量到的是 `nav-home`。先 `scrollIntoView({ block: 'center' })` 再量。
- **e2e：`locator.click()` 點標題文字會失敗**——Playwright 正確地回報「按鈕的 `::after` 攔截了點擊」，而那正是設計。改用 `page.mouse.click(x, y)`。
- **e2e：stub 的 SSE 串流送完 frame 就結束**，所以紀錄區誠實地顯示「未連線」、沒有 SSE 膠囊——原本寫的「收合時膠囊可見」斷言拿掉，改由 unit 覆蓋。
- **Pencil**：`Replace("G5v4u", …)` 之後回報 `Node 'G5v4u' has 'fill_container' sizing but is not inside a flexbox layout`——那是被換掉的舊節點，誤報（已知類型）；新節點 `p8No2j` 326×216 正常。`execute` 現在有 `Print()`，不必再用 throw 讀值。
- ESLint 在 e2e 檔不認 `getComputedStyle` 全域 → 寫 `window.getComputedStyle`；`async ({}, testInfo)` 觸發 `no-empty-pattern` → `{ page: _page }`。

### Completion Notes List

- Ultimate context engine analysis completed - comprehensive developer guide created（SM Bob，2026-09-21）
- 🔗 AC Drift: NONE (checked: 'workspace-event-log'、'lg:max-h'、'aria-live' across `_bmad-output/implementation-artifacts/*.md` — 命中都在 dsr-6d-c-1／-2；本張只加 `max-sm:`／`sm:hidden`，≥640 的行為與那兩張的 AC 完全相同＝REUSE。CR H3「清單自己捲」在 ≥1024 原樣保留，e2e 第 5 條 chromium 仍綠）
- 📎 Contract Stamps: NONE (no [@contract-v*] stamps in this story or upstream refs — 純呈現層，不定義也不消費線上契約)
- 🎭 A11y Pre-Flight: PASS (2 components checked — `GenerationWorkspaceV2.tsx`、`ActivityHub.tsx`；0 jsx-a11y warnings on touched files, 0 introduced by this story)。手動四類：圖片 N/A；對話框焦點 N/A（這是一頁）；**aria-live**：全頁仍恰好一個 live region（`workspace-log-announcer`），收合是 `display:none` 藏清單、announcer 不在被藏的容器裡，unit 釘住「收合／展開同一個 DOM 節點、收合時仍更新」；**自訂元件的鍵盤與 ARIA**：收合鈕是真的 `<button>`＋固定名稱「即時活動事件清單」＋`aria-expanded`＋`aria-controls`（指到 `role=list`），返回鍵 `aria-label="返回活動"`，兩顆都 44×44。
- **Task 1（稿）**：`n79vZ` 的迷你步驟條 `G5v4u` 換成 F3-M 步驟條的副本（新節點 `p8No2j`，326×216，`mst-2-pct` 維持停用）；`OMD2y` 關掉；三句列文案改成程式碼的說法；`J79fp` →「活動 ／ 生成字幕」；新規格註記 `ooxDh`（`spec-note-dsr-6f-4`，x=17040、y=35847、300×256）。全檔 `problems` 69（不變）；`PXB0z` 仍 1500 高、tab-bar 貼底。存檔走選單 Save，磁碟檔 grep 得到 `spec-note-dsr-6f-4`；匯出 196 張只留 `f11-m-v2.png`，其餘還原。
- **Task 2–5（程式碼）**：全部是 `max-sm:`／`sm:hidden`。新增 `onBack` prop（V2＋容器＋`ActivityHub` 用 `routeApi.useNavigate()` push 回 `/activity`）、`EventLogPane` 的 `open` state＋`useId`、`SseChip` 的 `className`。標題列 `order-first`、麵包屑不給 order；總覽卡第一欄 `max-sm:contents`＋`order` 原地重排；清單收合用 `max-sm:hidden`（不卸載）、展開 `max-sm:max-h-80`、展開時重設 `pinnedRef`、`useLayoutEffect` 依賴加 `open`；提示用 `hidden` 屬性。
- **測試**：`GenerationWorkspaceV2.spec.tsx` +9（54）、`GenerationWorkspaceContainer.spec.tsx` +1（27）、`ActivityHub.spec.tsx` +1（只擴充 stub）。新 e2e `generation-workspace-mobile.spec.ts` 5 條（390×4＋640×1）；共用 stub 抽到 `tests/support/helpers/generation-workspace-stubs.ts`（舊 spec 只改 import＋第 5 條加 viewport<1024 skip，斷言零改動）。
- **Mutation check 14 項 → 14 紅**（拿掉返回鍵／`order-first`／`max-sm:w-full`／`ml-auto` 去前綴／清單不藏／不重設 `pinnedRef`／依賴少 `open`／提示永遠顯示／`aria-expanded` 凍結／底部列不等寬／確認句不獨佔一行／容器不透傳／hub 不接線／副標不換行——每一項都恰有測試變紅，改完還原後 98/98 綠）。
- **閘門**：`format:check` ✅、`lint:all` 0 errors（129 warnings 為既有批次）、`web:typecheck` ✅、`check-design-tokens.py` ✅、`nx test web` **3974/3974**（274 檔）、`nx test api` ✅；e2e `chromium` 兩支 spec `--repeat-each=3` **30/30**；`mobile-chrome` 4 passed／6 skipped。跑完已清 `apps/api/coverage/` 與測試行程。
- ⚠️ **沒有在本機跑整支視覺套件**（約 9 分鐘、無法只濾工作區夾具）。桌機不變的證據是：所有新類別都是 `max-sm:`／`sm:hidden`；640 的 e2e 量到桌機版面（內距 32、進度條 180、標題 20px、藥丸貼著標題 8px、清單不必展開就可見）；CI 的 Visual Regression 是 required check——**若它紅了要當回歸查，不要 bootstrap**。
- **Pre-existing（已確認）**：把程式碼 stash 回 main 的版本後，`generation-workspace.spec.ts` 第 5 條在 `mobile-chrome` 是紅的（`scrolls: false`）——不是本張造成；走 Rule 24 ①（本張加 viewport<1024 skip）。

#### 🎨 UX Verification（Step 9）— 對 `f11-m-v2.png`／`PXB0z` 節點值

| Area | Design Spec | Implementation（390 e2e 量測） | Match? | Fix Needed |
| --- | --- | --- | --- | --- |
| 左右內距 | 16 | 總覽卡 x−root＝16、寬 358 | ✅ | — |
| 返回鍵 | chevron-left 24、貼齊 16、到標題 10 | 圖示左緣 16、到 `<h1>` 10、命中區 44×44 | ✅ | — |
| 標題 | H4 18 | `18px` | ✅ | — |
| 狀態藥丸 | 靠右、與標題同一行 | 右緣 374、中心差 <2px | ✅ | — |
| 麵包屑 | 標題下方、Label 12、muted、「／」 | 在標題下、在總覽卡上、`12px` | ✅ | — |
| 總覽卡 | 三行：計數＋膠囊／滿寬軌道／用量一行 | 軌道寬 328（358−2−28）、膠囊右緣對齊卡片內緣、用量在軌道下且同一基線 | ✅（稿 330 是因為稿的 stroke 不佔寬） | — |
| 佇列標題列 | BodyLg 600＋Secondary h44 | 不變 | ✅ | — |
| 進行中列步驟條 | 直排、圓點 22、Body 14（改稿後） | `GenerationProgressV2` 手機直排 | ✅ | — |
| 即時活動（收合） | 標題卡＋提示＋膠囊、chevron 20 靠右 | 清單隱藏、提示可見、整條標題列可按 | ✅ | — |
| 即時活動（展開） | 無稿（規格註記） | 清單 ≤320、自己捲、停在最新、無橫向溢出 | ✅（依裁定 3） | — |
| 底部列（達上限） | 無稿（裁定 7） | 兩顆等寬 173、不被分頁列蓋住 | ✅ | — |
| SSE 膠囊圓角 | `$radius-sm` | `radius-sm` | ✅（藥丸化另見 `disc-2026-09-sse-chip-pill-radius`） | — |

🎨 UX Verification: PASS — implementation matches design screenshots。e2e 截圖三張在 `test-results/dsr-6f-4/`（`390-running`／`390-log-open`／`390-budget-ceiling`；📎 `fullPage` 截圖會把 fixed 的頂列與分頁列疊在頁面中段，那是截圖方式造成的，不是版面問題）。

### Discovery Triage

<!-- Rule 24 — project-context.md. Any out-of-scope finding MUST land in exactly one lane with its
     sprint-status.yaml entry ID (② / ③) or absorbed AC # (①) BEFORE this story is marked done. -->

- **建單時的發現（SM Bob 2026-09-21）：**
  - ③ 640–1023 單欄時即時活動沒有高度上限、不會自己捲 → `disc-2026-09-workspace-log-uncapped-640-1023`
  - ③ 桌機稿 `KZmNG` 的列文案（「已生成繁中字幕，已寫入檔案」「轉錄音訊中…」「轉錄失敗，已略過並繼續（計入失敗數）」）與麵包屑分隔符（chevron vs「／」）與程式碼不一致 → `disc-2026-09-f11-pen-row-copy-and-breadcrumb-stale`
  - ③ F11-M 缺六種狀態的手機稿 → 補記 `disc-2026-09-flow-f-mobile-missing-state-screens`
  - ① 既有 e2e 第 5 條在本機手機 project **推定今天就是紅的**（<1024 沒有上限 → `scrolls:false`）→ AC #7／Task 6（加 viewport<1024 的 skip）。dev 先在 `mobile-chrome` 跑一次確認是既有的紅，再記成 ①
- **dev-story 期間的發現：** `N/A — no out-of-scope work discovered`（第 5 條 e2e 的既有紅已在建單時列為 ①，dev 確認屬實）。

### File List

- `ux-design.pen`（F11-M `PXB0z`：`n79vZ`／`p8No2j`／`Qp7Cm`／`Ah8fN`／`J79fp`；新註記 `ooxDh`）
- `_bmad-output/pen-tokens.json`
- `_bmad-output/screenshots/flow-f-subtitle-v2/f11-m-v2.png`
- `apps/web/src/components/subtitle/GenerationWorkspaceV2.tsx`
- `apps/web/src/components/subtitle/GenerationWorkspaceV2.spec.tsx`
- `apps/web/src/components/subtitle/GenerationWorkspaceContainer.spec.tsx`
- `apps/web/src/components/activity/ActivityHub.tsx`
- `apps/web/src/components/activity/ActivityHub.spec.tsx`
- `tests/e2e/generation-workspace-mobile.spec.ts`（新）
- `tests/support/helpers/generation-workspace-stubs.ts`（新）
- `tests/e2e/generation-workspace.spec.ts`（只改 import＋第 5 條 viewport<1024 skip）
- `_bmad-output/implementation-artifacts/sprint-status.yaml`
- `_bmad-output/implementation-artifacts/dsr-6f-4-workspace-mobile.md`

## Change Log

- 2026-09-21 — 建單（SM Bob，create-story；main `aee223d0`）。
- 2026-09-21 — 建單後對抗驗證（fresh-context 唯讀代理）：5 CRITICAL＋7 SHOULD FIX＋6 NIT，**全部併入**。最重要的五條：① `ActivityHub.spec.tsx` 把工作區 mock 掉了，原本寫的返回鍵測試寫不出來 → 改成擴充 stub＋容器 spec 驗透傳；② 替麵包屑加 `order` 會把它丟到總覽卡後面 → 改成標題列 `order-first`；③ 640 寬時側欄會出現，`x=32` 是錯的 → 一律量相對於根節點；④ `boundingBox()` 量不到 `::after`，「命中區寬 358」不可能過 → 改用點擊文字驗；⑤ `scrollIntoViewIfNeeded` 會把底部列停在分頁列底下 → 改用 `window.scrollTo` 到頁尾。另外：進度條寬是 328 不是 330（邊框）；`SseChip` 不准包一層；重新展開要重設 `pinnedRef`；既有 e2e 第 5 條的 skip 改成看 viewport 寬度（不誤傷 firefox）；`ActivityHub` 今天沒有 `useNavigate`，給出確切寫法。
- 2026-09-21 — dev-story（Amelia）：Task 1–6 全部完成。稿改 F11-M；程式碼全 `max-sm:`／`sm:hidden`＋`onBack`＋`open`；unit +11、e2e +5、mutation 14/14 紅；閘門全綠（web 3974/3974）。Status → review。
- 2026-09-21 — /ship 對抗式 CR（fresh-context 唯讀代理）：**0 HIGH／2 MEDIUM／7 LOW，修 8、記 1**。M1「收合且沒連線時頁尾那一列讓出間距」沒有任何測試守（14 項 mutation 清單漏了它）→ 補 testid `workspace-log-footer-row`＋unit，mutation 確認會紅；M2 e2e 每次都往 `test-results/` 寫三張整頁截圖（CI 每次都會上傳，本機下次跑就被清掉）→ 改成 `DSR_SHOTS=1` 才拍、寫到 `testInfo.outputPath()`；L3 還沒有事件就展開會畫一個空框 → `max-sm:empty:hidden`；L4 收合鈕的名稱「即時活動事件清單」與它控制的清單「生成事件日誌」用詞不同 → 改成畫面上看得到的「即時活動」；L5 「恰好一個 aria-live」只在收合時量 → 展開時也量；L6 標題與麵包屑之間的 2px、頭區到身體的 14px 沒量 → e2e 補上（拿掉 `-mt-3` 會紅，已確認）；L7 沒有 SSE 膠囊時（終態）總覽卡沒量 → 達上限那條 e2e 補兩行；L8 `type Route` 從不匯出它的模組 import → 改從 `@playwright/test`。**不修（記錄）**：L9 平板在收合狀態下從 ≥640 轉到 <640 再轉回來會丟掉清單的捲動位置，要等下一列事件才回到底——批次已結束就不會回來；屬於 640 兩側來回的邊角，併入 `disc-2026-09-workspace-log-uncapped-640-1023` 一起看。CR 後：unit 56/56（該檔）、e2e 兩支 chromium ×2 20/20。⚠️ 承上，Completion Notes 寫的 `test-results/dsr-6f-4/` 三張截圖已改為 `DSR_SHOTS=1` 時才產生。
- 2026-09-21 — **DONE**。PR #498 已合併進 main（commit `71d91c9b`）。PR 側 17 pass／0 fail，四個視覺 shard 全綠——證實本張沒有動到任何視覺基準線（本機沒跑的那一塊由 CI 補上），不需要 bootstrap PR。合併後 main 的 Docker／Tests／Visual Regression 三條都確認 success。`dsr-6f-flow-f-mobile` 傘狀條目一併結案。
