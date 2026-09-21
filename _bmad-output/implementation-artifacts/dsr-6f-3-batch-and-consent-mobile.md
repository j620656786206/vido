# Story DSR.6f-3：手機上的「產生字幕」三個抽屜（挑片、確認金額、批次進度）對齊設計稿，用量永遠看得到

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As the person who starts a paid subtitle batch from a phone,
I want 挑片、確認金額、看進度這三個抽屜的標題列、內距、按鈕都照手機稿排好，而且「本次用量」不會被捲走,
so that 我在手機上花錢的每一步都看得清楚、按得到，而不是桌機版面硬縮進 390 寬。

## Context

`dsr-6f`（Flow F 全部手機稿）拆出來的**第三塊**。**Depends on: `dsr-6f-1`（done，PR #488）、`dsr-6f-2`（done，PR #491）。**

6f-1 已經替這三個對話框換好**外殼**（貼底、滑入動畫、把手、44×44 ✕），但明文只換外殼三樣、**內容排版留給本張**。所以今天在手機上它們是「抽屜的殼＋桌機的肚子」：標題列 56 高、左右內距 24、批次的用量列會跟著清單捲走。

| 畫面 | 稿 | 程式碼 |
| --- | --- | --- |
| 批次進度 | F8-M-v2 `H717g`（sheet `bCZ9p`） | `GenerationBatchDialogV2.tsx`（`GenerationBatchPanelV2`） |
| 挑片清單 | F15-M-v2 `fdu4y`（sheet `tbF3W`） | `consent/GenerationConsentView.tsx`＋`consent/CandidateListPanel.tsx` |
| 確認金額 | F16-M-v2 `x45wBO`（`pYy2o`）／超過上限 F19-M-v2 `IMQO6`（`jd4CX`） | `consent/ConfirmGenerationDialog.tsx` |
| 沒有手機稿 | 分析中 F14／空狀態 F17・F20／錯誤 F18 | `consent/AnalysisProgressPanel.tsx`、`ConsentEmptyState.tsx`、consent view 的 error 區 |

⚠️ **驗收基準是 `.pen` 節點值**。PNG 只是參考。
⛔ **本張不進** `GenerationWorkspaceV2`（6f-4）、`ManageSubtitleDialogV2`／`GlossaryPanelV2`（已 done）、`GenerationProgressV2`（共用步驟條，6f-1 已做完手機版；本張**一行都不改**）。

### ⚖️ 裁定（Alexyu 2026-09-21，建單當下）

**批次卡片裡的六步進度條，手機用直排**（跟管理字幕 F3-M 同一個長相、14px、程式碼零改動）；**改的是稿**（F8-M 現在畫的是橫排縮窄版）。6f-4（生成工作區）之後照這個走——全 app 手機只有一種步驟條。代價已說明並接受：正在跑的那張卡片約 250 高，一次看到的片子少一兩部。

### 🔴 建單時查到的事（main `c5b0c2be`；行號皆為現況）

**三個抽屜共通**

1. **標題列是桌機的。** 批次 `GenerationBatchDialogV2.tsx:459` 與挑片 `GenerationConsentView.tsx:537` 逐字相同：`flex h-14 shrink-0 items-center justify-between border-b … pl-6 pr-12`（56 高、左 24），沒有任何 `sm:`。確認框 `ConfirmGenerationDialog.tsx:98` 是 `h-14 … pl-4 pr-12 sm:pl-6`（左邊已對、高度沒對）。四張手機稿的 `sheet-header`（`H1CYPa`／`hc6dc`／`F6AOwG`／`j8BUqm`）全部是：高 44（由 44 的 `close-hit` 撐開）、padding [0,4,0,16]、**有底線**、標題 BodyLg 600。
2. **✕ 的 `top` 跟著標題列高度走。** 三處都是 `max-sm:top-[22px]`（＝把手 16＋56/2−22）。標題列改 44 之後要變 `max-sm:top-4`（＝16＋44/2−22），與管理字幕、名詞表相同。🚨 **三支 spec 都在斷言 `max-sm:top-[22px]`**，而且註解寫明「改標題列高度這個 token 就要跟著改」：`GenerationBatchDialogV2.spec.tsx:258-268`、`ConfirmGenerationDialog.spec.tsx:76-85`、`GenerationConsentView.spec.tsx:1058-1069`——三條都要**改寫**。
3. **只有確認框的頁尾有 safe-area**（`ConfirmGenerationDialog.tsx:207`）。批次頁尾（`:631`）、挑片頁尾（`CandidateListPanel.tsx:1123`）、分析中頁尾（`AnalysisProgressPanel.tsx:67`）、空狀態頁尾（`GenerationConsentView.tsx:597`）都沒有。（今天 `env(safe-area-inset-bottom)` 恆為 0——`disc-2026-09-viewport-fit-cover-missing`——照寫，但不要宣稱它生效。）
4. **把手只有批次那顆有 testid**（`gen-batch-drag-handle`）；另外兩顆 `<SheetGrabber />` 沒有，e2e 與 spec 取不到。

**批次 F8-M**

5. **稿整張過時。** 標題 `Qm8Bc` 還是「批次生成字幕」（桌機 `i9Nun1` 與程式碼都是「產生字幕」）；`scope-line` `g9aR7j` 還畫著 sub-4-3 就移除的切換鈕「缺字幕的項目 38／已選項目 5」（`RRXVB`／`C2mqH`；桌機是純文字「範圍：已選項目（5 部）」）；計數 `mW8JY` 是 12 / 38（桌機 2 / 5）；步驟條 instance `RGssw` 沒有覆寫 `fVBQF`，所以還顯示「轉錄中 45%」（轉錄沒有百分比，dsr-6b 已裁定）；`RGssw` 把六個步驟各壓成 52 寬、五條連接線（`ITuZl`／`fpZoq`／`qheA5`／`GZtws`／`Wztud`，母版 `XkGvG` 裡 `gp-cnN-ln` 線段）壓成 8 寬，才勉強塞進 334。清單 `p6x759` 有**四**張卡：`GmqGd`（沙丘・完成）、`fBMht`（奧本海默・完成）、`pE5IE`（進行中）、`L676DC`（星際效應・排隊中）——最後一張今天就是 `partially clipped`（全檔 70 個 problems 之一）。
6. **用量列會被捲走。** 稿把「本次用量 $0.42 / 上限 $5.00」＋SSE 膠囊放在**固定頁尾**（`GSnOg`：直排 gap 10、padding [12,16]，上面是 `M5Cp2f` cost-row、下面是滿版 h44 的「全部取消」`E0wGis`）；程式碼把它放在**捲動區的最後一個子項**（`:603-627`）——清單一長，錢花了多少就被捲到看不見。桌機稿 `i9Nun1` 的 cost-row 在 body 裡（桌機不動）。
7. **內距是桌機的。** body `:464` `px-6 py-5 gap-4`；稿 `mxssT` [6,16,12,16] gap 14、`p6x759` [0,16,8,16] gap 8。頁尾 `:631` `px-6 py-3.5`，按鈕靠右、不滿版；稿滿版。
8. **頁尾有四種狀態，稿只畫了一種**（執行中）：執行中「全部取消」／取消確認（一句話＋「繼續生成」＋「確定取消」，`:647-694`，外層 `flex flex-wrap`）／結束（「關閉」＋「重試失敗項目」或「再產生字幕」）／達上限（「關閉」＋「下次繼續」）。
9. **卡片裡的步驟條**：`QueueRow` `:218-231` 用 `<div className="sm:pl-12">` 包 `GenerationProgressV2`，手機上已是直排 14px（6f-1）。⚖️ 裁定用直排 → **程式碼不動，改稿**。

**挑片 F15-M**

10. **列本身已經對過了**（`bugfix-f15-row-mobile-identity-collapse`，container query：`@xl:`／`@max-xl:`），工具列的手機短文案也已經有（`:943-944`「全部可抽取」、`:952-953`「清除」）。**本張不重做列。**
11. **外圍內距是桌機的。** 控制區 `:794` `px-6 pb-3 pt-6 gap-3`（稿 `fusXx` [6,16,10,16] gap 10）；清單 `:963-971` `px-6`（稿 `DXXEC` [0,16,8,16]）；提示條 `:1043`、超過上限橫幅 `:1050-1054`、啟動錯誤 `:1112-1115`（`mx-6`）、頁尾 `:1123` 全是 `px-6`。
12. **頁尾已經直排、「開始產生」也已經滿版**（手機沒有 `items-*` → stretch，按鈕本身 `justify-center`）；**沒對的只有預算輸入框固定 `w-24`**（`:1150` 是帶框的 `<span class="flex h-9 w-24 …">`，真正的 `<input>` 在 `:1157-1167`、已是 `min-w-0 flex-1`，testid `consent-budget-input` 掛在 input 上）與內距。 稿 `XNgsZ`：直排 gap 10、padding [12,16]；`Ro1aU` 預算列＝標籤＋**撐滿**的輸入框（`LfwJL` `fill_container` h36）；`g1oWtd`「開始產生」**滿版** h44。
13. **挑片畫面做不出視覺夾具。** `GenerationConsentView` 的 phase／candidates 全是元件內部 state，直接呼叫 `subtitleService`（`:224`、`:245`），沒有 query 可以 seed；既有的 `generation-consent/list-mobile`／`list-mobile-groups` 是 `width: 390` 的**框寬**夾具——頁面仍是 1280，所以 `sm:` 走的是**桌機分支**，它們拍到的不是手機版面。⛔ 不要為了拍照開後門（6f-1 同一條裁定）。手機版面由 e2e 守。

**確認 F16-M／F19-M**

14. **幾乎已經對了。** 內容區 `:102` `p-4 sm:px-6 sm:py-5`（稿 `UXiBN` [20,16] gap 16 → 只差上下 16 vs 20）；頁尾 `:207` `px-4 py-3.5 gap-3 justify-end`＋safe-area（稿 `V7TDNe` [14,16] gap 12 **`justifyContent:end`**）。
15. ❗ **sprint 條目寫的「確認框頁尾兩顆按鈕不滿版」不是缺口**——稿 `V7TDNe`／`Maz74` 就是靠右、不滿版（F16-M 的 `e5Vn1`／`dqOIH`、F19-M 的 `vODtl`／`cSqGe` 都沒有 `fill_container`）。程式碼與稿一致，**不要改成滿版**。
16. `generation-consent/confirm-mobile` 夾具已存在（6f-1 立的，`-gallery.fixtures.tsx:5081`，註解寫明「mobile LAYOUT is dsr-6f-3's job」）。本張改標題列高度之後**它的基準線會變**——這是刻意的改動，不是回歸。

**交接進來的舊帳**

17. **dsr-6e-2 CR L5**：`GenerationConsentView.tsx:478-490` 的 `handleCancelAnalysis` 在 `finally` 裡 `setCancelling(false)`，而 `onClose()` 在它之前——**若**父層的關閉有動畫，按鈕會從「取消中…」閃回「取消」（`AnalysisProgressPanel.tsx:90`）。⚠️ **今天在 production 看不到**：唯一的父層 `GenerationBatchDialogV2.tsx`（約 `:1102`）是 `if (!open) return null;`，`onOpenChange(false)` 之後整棵樹立刻卸載。這同時代表 6f-1 的 `sheet-exit` 在批次／挑片這兩個抽屜**永遠播不到**（另立 `disc-2026-09-batch-consent-sheet-exit-never-plays`）。本張把 L5 當元件層的防呆修掉——哪天退場動畫接上，它不會跟著冒出來。`cancelling` 只在 `:126` 宣告、`:479` 設 true、`:488` 設 false、`:558` 傳給 panel，**沒有 on-open 重設點**。
18. **dsr-6d-b CR L16**（SSE 膠囊是 `radius-sm`，DESIGN.md 要藥丸）：全 Flow F 共三處（`GenerationBatchDialogV2.tsx:618`、`GenerationWorkspaceV2.tsx:152`、`ManageSubtitleDialogV2.tsx:547`），改了會動到**桌機**基準線與 6f-4 的範圍 → 本張不做，另立單（裁定 5）。
19. **dsr-6e-2 CR L1**（補 `confirm-mobile` **系列**夾具並對 F16-M／F19-M 字階）：6f-1 只立了 `confirm-mobile` 一張；F19-M（超過上限）沒有手機夾具。
20. **Rule 21**：`GenerationBatchDialogV2.tsx:1` 只列 `Screen F8-D-v2 (i9Nun1)`，沒有 F8-M；其餘檔案已列手機稿（`AnalysisProgressPanel`／`ConsentEmptyState` 沒有手機稿，不必動）。
21. **兩處稿與碼的小差異**：F8-M 頁尾用量列整排是 **Label 12**（`Ffh4p`／`kbBNl`／`tZ5z4`／`bOP9s`），程式碼 `text-sm`；卡片稿 padding 12／gap 12，`QueueRow` 是 `px-4 py-3.5 gap-3`。

### 設計稿節點

| 代號 | 節點 | 本張要看的 |
| --- | --- | --- |
| F8-M-v2 | `H717g` → sheet `bCZ9p`（y=214、390×630、`clip:true`） | 把手 `zkKkC`／`k46gFw`（36×4 pill `$bg-tertiary`）；標題列 `H1CYPa`（[0,4,0,16]、底線）：`Qm8Bc`、✕ `eVZFG` 44×44／`PCE7G` 18；`mxssT` body-top [6,16,12,16] gap 14：`g9aR7j` scope-line（**過時**）、`zczAR` overall（`tUXAf` Label、`mW8JY` H4 Mono、軌道 `IPPLS` h6）；`p6x759` item-list [0,16,8,16] gap 8：卡片 `GmqGd`／`fBMht`／`pE5IE`（padding 12、gap 12、`$radius-lg`、thumb 38×54）；`pE5IE` 內 `wiqIo` → `RGssw`（橫排步驟條，**要換**）；頁尾 `GSnOg`（直排 gap 10、[12,16]、上框線）：`M5Cp2f` cost-row（`Ffh4p`／`kbBNl`／`tZ5z4`／`bOP9s` 全 Label）、`bQoHT` sse-chip、`E0wGis`「全部取消」滿版 h44 |
| F8-D-v2（對照） | `i9Nun1` → `Fkiqd` | 標題「產生字幕」、scope「範圍：已選項目（5 部）」純文字、2 / 5、步驟條 `fVBQF:{enabled:false}`、cost-row 在 **body**、頁尾只有「全部取消」 |
| F3-M-v2（步驟條來源） | `k8sJl4` → `fS5is` | 直排步驟：gap 6、每列 padding 4 gap 10、圓點 22、標籤 Body（進行中 600 `$accent-text`／完成 `$text-secondary`／未到 `$text-muted`）；6f-1 已把 45% `E10kib` 關掉 |
| F15-M-v2 | `fdu4y` → sheet `tbF3W`（y=98、390×746） | 標題列 `hc6dc`；`fusXx` body-top [6,16,10,16] gap 10（摘要 Label、搜尋＋排序 h44、chips、工具列「全部可抽取」「清除」）；`DXXEC` item-list [0,16,8,16] gap 4；頁尾 `XNgsZ`（直排 gap 10、[12,16]）：`hEtIY` 已選／明細、`Ro1aU` 預算列（`LfwJL` `fill_container` h36）、`FP0aj` 提示、`g1oWtd`「開始產生」**滿版** h44（`otvKh` Primary） |
| F16-M-v2／F19-M-v2 | `x45wBO` → `pYy2o`；`IMQO6` → `jd4CX`（y=199、高 645） | 標題列 `F6AOwG`／`j8BUqm`；body `UXiBN`／`Rn42x` [20,16] gap 16；頁尾 `V7TDNe`／`Maz74` [14,16] gap 12 **`justifyContent:end`**、按鈕 h44 **不滿版** |
| 規格註記 | 群組 `JzmvC`；6f-2 的 `x1MC3W` 在 x=17040、y=35348、寬 300、實測高 217 | 新註記放 `x1MC3W` 下方 40px → y=**35605**（動手前再 `Get` 量一次確認） |

字階（`DESIGN.md:351-360`）：H4 18、BodyLg 16、Body 14、Label 12。間距：`xs`4・`xs-plus`6・`sm`8・`sm-plus`10・`md`12・`md-plus`14・`lg`16・`lg-plus`20・`xl`24。

---

## Acceptance Criteria

1. **設計稿先改（Pencil MCP；每個節點動手前再 `Get` 一次）。只改 F8-M，另外三張手機稿不動。**
   - `Qm8Bc` content →「產生字幕」。
   - `g9aR7j` scope-line：刪掉 `RRXVB`、`C2mqH`，把 `s3Rju` 改成「範圍：已選項目（5 部）」——桌機 `i9Nun1` 的 scope-line 是三段（`STldN` 前綴／`S4Tnh` 數字 Mono／`hMRfV` 後綴），`Copy` 那三個節點進來最省事（手機用 Label 字階，與原 `s3Rju` 一致）。畫面 frame 名稱「F8-M-v2 · 批次生成（手機）」不改（桌機也還叫「批次生成」，改名會動到匯出腳本的對照）。
   - `mW8JY` →「2 / 5」；`ibeyL` 寬度改成軌道寬 × 2/5（先 `Get` 量 `IPPLS` 的實寬）。
   - **步驟條改直排（⚖️ 裁定）**：刪掉 `RGssw`，`Copy` F3-M 的 `fS5is`（在 `k8sJl4 > Me1fR > tRFbG` 底下，實高 216＝6 列×31＋5×gap 6）進 `wiqIo`（`width: fill_container`）。`fS5is` **已經是目標狀態**（提取音訊＝完成、轉錄中＝600 `$accent-text`、其餘 muted），與卡片右上的 `vSxCC`「轉錄中」一致，不用再改狀態；它帶著一顆 6f-1 停用的 `E10kib`「45%」（`enabled:false`）——**讓它維持停用**，不要打開也不必刪。`wiqIo` 的 `justifyContent:"center"` 改掉（直排要靠左）。
   - **清單只留兩張卡**：刪掉 `GmqGd`（沙丘）**與** `L676DC`（星際效應），留下 `fBMht`（奧本海默・完成）＋`pE5IE`（進行中）。高度預算已算過：換成直排之後進行中的卡＝12+54+12+216+12＝306；兩張卡時 sheet ≈ 16+44+96+(78+8+306+8)+104 ≈ **660 ≤ 675**（螢幕 80%，`DESIGN.md:630`），三張卡約 746 會爆。「2 / 5」不改——另一張完成的與兩張排隊中的在捲動區外，規格註記會寫明清單在抽屜內捲動。改完 `bCZ9p` 的高度與 `y = 844 − 高度` 重新貼底（`bCZ9p` 現在是固定高 630、`p6x759` 是 `fill_container`，先 `Get` 量再設）。
   - 規格註記（`JzmvC`，樣式比照 `x1MC3W`）：
     > 「F8／F15／F16／F19（手機）：三個抽屜的標題列都是 44 高、有底線、✕ 44×44。批次的『本次用量』＋SSE 膠囊在**固定頁尾**（不隨清單捲動），其下的動作鈕滿版；執行中以外的頁尾狀態（取消確認／結束／達上限）沒有手機稿：用量列同樣在頁尾最上面，兩顆鈕等寬並排。卡片裡的六步進度條在手機一律**直排**（同 F3-M，⚖️ 2026-09-21），桌機橫排；清單在抽屜內捲動，本稿只畫得下兩張卡。挑片頁尾的『開始產生』滿版、預算輸入框撐滿；確認框（F16／F19）頁尾兩顆鈕**靠右不滿版**。分析中／空狀態／錯誤（F14／F17／F18／F20）沒有手機稿：沿用桌機元件，左右內距 16、頁尾按鈕靠右。」
   - ⛔ 不改母版（`XkGvG`、`YDPhc`、`otvKh`、`SG1ln`）、不改桌機稿、不改 F15-M／F16-M／F19-M／F3-M。
   - 收尾：全檔 `problems` 數**不得超過現況 70，理想是 69**（`L676DC` 那個 clip 會消失；`DESIGN.md:836` 寫的 74 是舊值，不要拿它當基準）；對 Pen.app 送 Cmd+S（`osascript -e 'tell application "Pen" to activate' …`）後用 `stat` 確認 mtime／size 變了；匯出後**只 stage** `flow-f-subtitle-v2/f8-m-v2.png` 與 `_bmad-output/pen-tokens.json`（`penSha256` 必變，漏了 `check-design-tokens.py` 會紅），其餘 `git checkout` 還原。
   - 📎 Pencil `execute` 沒有 `console`／`Save`；讀值用 `throw new Error(JSON.stringify(Get(...)))`，而 **throw 會把同一次呼叫裡的編輯一起 rollback**——量測與寫入分兩次呼叫（6f-2 Debug Log）。

2. **三個抽屜的標題列在手機是 44 高（🔴 #1、#2、#4）。** 一律 `max-sm:`，桌機一個像素都不能變。
   - 批次 `:459`、挑片 `GenerationConsentView.tsx:537`：加 `max-sm:h-11 max-sm:pl-4`（底線保留）。確認框 `:98`：加 `max-sm:h-11`（`pl-4` 已有）。
   - 三處 `closeClassName` 的 `max-sm:top-[22px]` → `max-sm:top-4`。
   - 三個標題列各給 testid：`gen-batch-title-bar`／`consent-title-bar`／`consent-confirm-title-bar`；另外兩顆把手補 testid：`consent-sheet-grabber`／`consent-confirm-sheet-grabber`（批次那顆 `gen-batch-drag-handle` **不准改名**，`GenerationBatchDialogV2.spec.tsx:255` 在斷言它）。
   - 🚨 三支 spec 的 `max-sm:top-[22px]` 斷言要**改寫**成 `max-sm:top-4`，連同那段「16＋28−22＝22」的註解（🔴 #2）。

3. **批次：用量列固定在頁尾，動作鈕滿版（🔴 #6–#8）。**
   - 把 `:603-627` 的用量列抽成檔內的小元件（例如 `CostRow`，props 夠它畫出用量與 `isRunning` 的 SSE 膠囊即可），**畫兩次**：
     - 原位（body 最後）那份外層加 `max-sm:hidden`，testid **不變**（`gen-batch-cost-line`／`gen-batch-sse-chip`——既有測試與 `tests/e2e/batch-subtitle.spec.ts` 都用它們）。
     - 頁尾最上面那份外層 `sm:hidden`，testid 加 `-mobile` 後綴（`gen-batch-cost-line-mobile`／`gen-batch-sse-chip-mobile`）；字級依稿是 **Label 12**（`text-xs`；原位那份維持 `text-sm`，桌機不動）。先例：6f-1 的 `toggle-fetch-mobile`。`display:none` 的那份不在無障礙樹裡，不會被念兩次。
     - 📎 已查證：既有 unit 只用 testid 取這兩個元素（spec `:283-286`、`:1653-1660`），`tests/` 底下的 e2e 完全沒用到它們——畫兩次**不會**讓任何現有斷言 multiple-match。新測試也請一律用 testid，不要 `getByText('本次用量')`。
   - body（`:464`）與頁尾（`:631`）今天都沒有 testid——補 `gen-batch-body`／`gen-batch-footer`（AC #8、#9 要量它們）。
   - 頁尾 `:631`：加 `max-sm:flex-col max-sm:items-stretch max-sm:gap-2.5 max-sm:px-4 max-sm:py-3 max-sm:pb-[max(0.75rem,env(safe-area-inset-bottom))]`。
   - 四種狀態的按鈕在手機：
     - 執行中：「全部取消」只要 `max-sm:justify-center`（稿 `E0wGis`）——頁尾變成 column＋`items-stretch` 之後它**自己就會滿版**，`w-full` 是多餘的。
     - 取消確認（`:647-694`，外層 `flex flex-wrap items-center gap-3`）：確認句約 294px 會自己獨佔一行，但**錯誤句「取消失敗…」約 252px，會跟「繼續生成」擠在同一行**（252+12+88=352 < 358）——兩個 `<span>` 都要加 `max-sm:w-full`；兩顆鈕 `max-sm:flex-1 max-sm:justify-center`。只用 `max-sm:` 就做得到。
     - 結束／達上限：這些按鈕今天是頁尾的**直接子項、四個條件式兄弟**（close／retry／restart／resume），所以這是一次小的 JSX 重構，不是「包一層」而已——把它們收進一個 `<div className="flex gap-3 max-sm:w-full sm:contents">`，各 `max-sm:flex-1 max-sm:justify-center`。（`sm:contents` → 桌機盒模型不變，6f-2 已實測。）
     - 📎 已查證：頁尾今天沒有 `ml-auto`；`justify-end` 在 column 之下無害。仍然用瀏覽器量（AC #9）。
   - body `:464`：加 `max-sm:gap-3.5 max-sm:px-4 max-sm:pt-1.5 max-sm:pb-2`。
   - `QueueRow` 的卡片加 `max-sm:p-3`（稿 padding 12；`gap-3` 已是 12）；`sm:pl-12` 與 `GenerationProgressV2` **不動**（⚖️ 直排）。
   - Rule 21：`GenerationBatchDialogV2.tsx:1` 的 `// Design ref:` 接 ` + Screen F8-M-v2 (H717g)`（🔴 #20）。
   - 橫幅（`gen-batch-budget-banner`／`gen-batch-error-banner`）不動。

4. **挑片：內距 16、預算輸入框撐滿、「開始產生」滿版（🔴 #11、#12）。**
   - `CandidateListPanel.tsx`：控制區 `:794` 加 `max-sm:gap-2.5 max-sm:px-4 max-sm:pt-1.5 max-sm:pb-2.5`；清單 `:963-971`、提示條 `:1043`、超過上限橫幅 `:1050-1054` 各加 `max-sm:px-4`；啟動錯誤 `:1112-1115` 加 `max-sm:mx-4`。
     - 🚨 `CandidateListPanel.spec.tsx:1503-1510` 斷言橫幅 `toContain('px-6')`、`:1063` 斷言 `min-h-[10rem]`、`:1512-1521` 斷言開始鈕 `px-5`——**只加 `max-sm:`，不要替換既有 token**。
   - 頁尾 `:1123`：加 `max-sm:gap-2.5 max-sm:px-4 max-sm:py-3 max-sm:pb-[max(0.75rem,env(safe-area-inset-bottom))]`；預算那一列在手機撐滿：class 加在 **`:1150` 那個帶框的 `<span>`** 上（`max-sm:flex-1` 就夠，`flex: 1 1 0%` 會蓋掉 `w-24`），並替它補 testid `consent-budget-box`——⚠️ `consent-budget-input` 掛在裡面的 `<input>` 上，斷言打到 input 會是綠的但版面沒變。`consent-start-btn` **不用加任何東西**（今天就滿版，🔴 #12）。
   - ⛔ 列（`:338-399` 的 container query）、chips、搜尋排序列、工具列文案**不動**（🔴 #10）。
   - `GenerationConsentView.tsx`：空狀態頁尾 `:597` 與錯誤區 `:611` 加 `max-sm:px-4`；空狀態頁尾另加 safe-area。
   - `AnalysisProgressPanel.tsx`：內容 `:37` `px-10` → 加 `max-sm:px-4`；頁尾 `:67-69` 加 `max-sm:px-4`＋safe-area。🚨 `AnalysisProgressPanel.spec.tsx:22-28` 斷言頁尾有 `justify-end`、`:33-47` 斷言取消鈕**沒有任何 `disabled:` 開頭的 token**——按鈕維持靠右（與 F16-M 的頁尾同一套），不要改成滿版。
   - `ConsentEmptyState.tsx` `px-8` → 加 `max-sm:px-4`。

5. **確認框：只差兩處（🔴 #14–#16）。**
   - 標題列高度（AC #2）；內容區 `:102` 加 `max-sm:py-5`（稿 [20,16]）。📎 這個 className 沒有過 `cn()`，`p-4`＋`max-sm:py-5` 是靠 CSS 產生順序生效（`max-sm:` 排在基底工具之後），可行。
   - 補 F19-M 的手機夾具 `generation-consent/confirm-over-budget-mobile`（props 同 `:5033`、`viewport: 390×844`），並在 dev-story Step 9 對 F16-M／F19-M 逐項核字階（標題 BodyLg 600、lead BodyLg 600、模型名 Body、金額 Body 700、合計 BodyLg 700、提示 Body）——這兩件做完即**結掉 dsr-6e-2 CR L1**。
   - ⛔ 頁尾兩顆鈕**維持靠右不滿版**（稿 `V7TDNe` 就是 `justifyContent:end`；🔴 #15）。
   - `generation-consent/confirm-mobile` 的基準線**會變**（刻意）——走 `project_visual_baseline_intentional_change` 四步：本機重產 darwin → `git rm` 舊的 `-linux` → 先併 main → 對分支 `gh workflow run "Visual Regression" --ref <branch>`，合掉它開回分支的 bootstrap PR。⚠️ `--update-snapshots` 會順手重生所有本機漂移的 darwin 基準——**只提交真的因本張而變的那張**，其餘 `git checkout --` 還原。

6. **取消分析時按鈕不再閃回「取消」（🔴 #17）。**
   - `handleCancelAnalysis`：拿掉 `finally` 的 `setCancelling(false)`；成功與失敗兩條路都是 `onClose()` 之後**維持** `cancelling === true`，改在 `bootstrap` 開頭歸零（`:221` 旁、與重設 phase 同一個時機——今天**沒有**任何 on-open 重設點，要新增；「重試」也會走到這裡，無害）。
   - 📎 `AnalysisProgressPanel` 有 `if (!cancelling) onCancel()`：若父層在 `onClose()` 之後**不**關，取消鈕會永久失效。現行唯一的父層一定會關（而且是直接卸載），可接受——寫進元件註解。
   - 紅測試：點「取消」→ 等 `cancelCandidateAnalysis` resolve → `onClose` 已被呼叫**之後**，按鈕文字仍是「取消中…」；重新 `open` 之後回到「取消」。失敗路徑（reject）同樣。

7. **既有的行為不准回歸。**
   - 桌機（≥640）：所有既有 1280 基準線**零變動**（`generation-batch-dialog-v2/*` 四張、`generation-consent/*` 除 `confirm-mobile` 以外全部）。
   - 執行中按 Esc／點遮罩不會關（`:436-445`）；取消確認的 `aria-disabled` 行為、`gen-batch-status-live`／`consent-phase-live` 的播報、409 復原、lazy-SSE 全部照舊。
   - `tests/e2e/batch-subtitle.spec.ts` 七條全綠，**不改它**。
   - ⛔ 不改 `GenerationProgressV2.tsx`、`ModelPicker.tsx`、`consentSelection.ts`／`consentRows.ts`、`ui/*`、任何後端檔案。

8. **測試。** 分「**紅**」與「**守**」，誠實標示（Rule 16；6f-2 CR L10）。jsdom 不看 media query，unit 只能驗 token——版面由 AC #9 守。
   - 三支 spec（改寫）：✕ 是 `max-sm:top-4`；（紅）標題列 testid 存在且帶 `max-sm:h-11`；（紅）另外兩顆把手的 testid。
   - `GenerationBatchDialogV2.spec.tsx`：（紅）頁尾有 `gen-batch-cost-line-mobile`、其外層 `sm:hidden`；原位那份外層 `max-sm:hidden`；執行中才有 `gen-batch-sse-chip-mobile`；`gen-batch-footer` 帶 `max-sm:flex-col`；四種狀態的按鈕各帶對的 token（取消確認的兩個 `<span>` 帶 `max-sm:w-full`）；（守）`gen-batch-drag-handle`、Esc 守衛、既有 86 條。
   - `CandidateListPanel.spec.tsx`：（紅）`consent-budget-box` 帶 `max-sm:flex-1`；（紅）控制區／清單／頁尾帶 `max-sm:px-4`；（守）`px-6`／`px-5`／`min-h-[10rem]` 還在。⚠️ 「開始產生滿版」**不是紅測試**——今天就滿版。
   - `AnalysisProgressPanel.spec.tsx`：（紅）頁尾帶 `max-sm:px-4`；（守）`justify-end`、零 `disabled:`。
   - `GenerationConsentView.spec.tsx`：AC #6 的兩條紅測試。
   - ⚠️ **不要寫「沒有橫向溢出＝`scrollWidth <= clientWidth`」這種斷言去守會換行的容器**——它不可能失敗（6f-2 CR H1）。要守就量高度上界或「同一條水平線」。

9. **真瀏覽器的驗證。**
   - **視覺夾具**（Portal，`viewport: { width: 390, height: 844 }`，不可同時給 `width`）：`generation-batch-dialog-v2/running-mobile`（props 同 `:5158`）、`generation-batch-dialog-v2/budget_ceiling-mobile`（props 同 `:5204`——兩顆鈕的頁尾）、`generation-consent/confirm-over-budget-mobile`（AC #5）。`confirm-mobile` 重拍（AC #5）。
     - ⛔ **不要**加「在頁內」的 390 夾具：`/test/gallery` 的 `p-8` 讓它只有 326 寬，拍到的是沒有任何畫面會長成那樣的版面（6f-2 CR M7）。既有的 `list-mobile`／`list-mobile-groups` 不動。
   - **e2e**（新，`tests/e2e/batch-consent-mobile.spec.ts`，`@e2e @batch-consent-mobile`）：stub 照抄 `tests/e2e/batch-subtitle.spec.ts` 的 `stubPopulatedLibrary`／`readyCandidates`／`startedBatch`（可以把共用的搬到 `tests/support/`，**但不准改 `batch-subtitle.spec.ts` 的行為**）；viewport 與動畫等待照抄 `tests/e2e/glossary-mobile.spec.ts`（`Promise.allSettled(getAnimations().map(a => a.finished))` 再量）。
     - 390：走 `/library` → `enter-selection-btn` → `batch-subtitle-btn`。
       - 挑片：`animation-name` 是 `sheet-enter`、貼底 (0,390,844)、標題列高 44、✕ ≥44×44 且**垂直中心與標題列中心差 <2px**（這就是 `top-4` 對不對的量法）、**`consent-budget-box` 的右緣 ≈ 374**（量 span，不是 input——input 右緣約 361）；「開始產生」寬 ≈ 358 是**守**（改動前就會過）。
       - 確認：標題列高 44；「取消」「確認並開始／仍要開始」**同一條水平線且都靠右**（右緣 ≈ 374，寬 < 200）。
       - 批次：`gen-batch-cost-line-mobile` 可見、原位那份不可見；**`gen-batch-cost-line-mobile` 的 `y` ≥ `gen-batch-body` 的底邊**（固定在頁尾）；「全部取消」寬 ≈ 358；執行中的卡片裡步驟條是直排（用現成的 `gen-stage-<stage>` testid，`GenerationProgressV2.tsx:177`：第二步的 `y` > 第一步的 `y`）。📎 已查證不需要改 stub：既有 stub `:205` abort `/events`，而 `startedBatch` 的 item 0 就是 `running`，步驟條會以 idle 畫出六步；390 寬時 `enter-selection-btn`／`batch-subtitle-btn` 都可見（只有文字收成 `hidden sm:inline`）。
     - 640（斷點另一側）：`dialog-enter`、沒有把手、`gen-batch-cost-line` 可見而 `-mobile` 不可見、「全部取消」沒被拉寬。
     - 本機跑要 `AI_PROVIDER=claude npx playwright test …`（`preexisting-fail-e2e-local-ai-provider`）。
   - 只產 darwin；新夾具缺的 `-linux` 由 `gh workflow run "Visual Regression" --ref <branch>` 開 bootstrap PR 補（6f-1 #489、6f-2 #492 都是手動觸發才開的）。

10. **另立的單子（建單時已寫入 sprint-status）**：`disc-2026-09-sse-chip-pill-radius`、`disc-2026-09-batch-consent-sheet-exit-never-plays`；補記 `disc-2026-09-flow-f-mobile-missing-state-screens`（F14／F17／F18／F20 與批次三種頁尾狀態沒有手機稿）、`dsr-6f-4-workspace-mobile`（⚖️ 步驟條直排）。

11. **CI 全綠**：`pnpm run lint:all`、`pnpm nx run web:typecheck --skip-nx-cache`、`python3 scripts/check-design-tokens.py`、`pnpm nx test web`、`pnpm nx test api`。⛔ 絕不 `run_in_background` 跑測試；Nx 一律 `NX_DAEMON=false`；跑完 api 測試把 `apps/api/coverage/` 刪掉（`disc-2026-09-api-coverage-not-gitignored`）。🚨 合併之後要看 `main` 那一次的 CI。

## Tasks / Subtasks

- [x] **Task 1 — 設計稿：F8-M 追上桌機、步驟條改直排（AC: #1）**
- [x] **Task 2 — 三個標題列 44 高＋✕ `top-4`＋testid（AC: #2, #8）**
  - [x] 先改寫三支 spec 的 `top-[22px]` 斷言（會紅）→ 實作
- [x] **Task 3 — 批次：用量列進頁尾、四種頁尾狀態、內距（AC: #3, #8）**
- [x] **Task 4 — 挑片、分析中、空狀態的內距與頁尾；確認框內距（AC: #4, #5, #8）**
- [x] **Task 5 — 取消分析不閃回（AC: #6）**
- [x] **Task 6 — 夾具、`confirm-mobile` 重拍、e2e、收尾（AC: #7, #9, #10, #11）**
  - [x] dev-story Step 9：`f8-m-v2`／`f15-m-v2`／`f16-m-v2`／`f19-m-v2` 對新的手機基準線與 e2e 量測

## Dev Notes

### 這張的重點

- **全部是 `max-sm:` 加法。** 沒有新機制；唯一的結構變動是批次用量列「畫兩次」。
- **最容易踩的雷是 `top-[22px]`**：三支 spec、三個元件、一個數字，漏一個 ✕ 就歪 6px，而 jsdom 看不出來——e2e 用「✕ 中心對標題列中心」量。
- **稿不是都對的。** F8-M 整張過時（先改稿）；但 F16-M 的「按鈕不滿版」是**對的**，是 sprint 條目寫錯。動手前看節點值，不要照條目的轉述做。
- **挑片畫面沒有視覺夾具可拍**（內部 state），它的手機版面只有 e2e 守——e2e 的斷言要量得到東西（見 AC #8 最後一條）。

### 上游契約（Rule 20 ack）

- 本張不消費也不改任何線上契約（純呈現層）。e2e 的假資料沿用 `batch-subtitle.spec.ts` 既有的形狀，不自創欄位。

### 建單裁定（2026-09-21，Sally／Alexyu 可在 review 推翻）

1. ⚖️ **步驟條在手機直排**（**Alexyu 2026-09-21 當場裁定**）。改稿不改碼。
2. ⚖️ **批次用量列在手機「畫兩次」而不是用 CSS 搬。** body 是捲動容器、頁尾是它的兄弟，`order` 跨不過容器；而「錢花了多少」固定在看得到的地方，正是稿把它放頁尾的理由。代價是一組 `-mobile` testid（6f-1 `toggle-fetch-mobile` 先例）。
3. ⚖️ **沒有手機稿的頁尾狀態**（批次的取消確認／結束／達上限）：用量列同樣在頁尾最上面，兩顆鈕等寬並排。**沒有手機稿的畫面**（分析中／空／錯誤）：只調內距與 safe-area，按鈕維持靠右——與 F16-M 的頁尾同一套，也不必動 `AnalysisProgressPanel.spec.tsx` 的 `justify-end` 守門。
4. ⚖️ **確認框頁尾不滿版**——依稿（`justifyContent:end`）。同一個流程裡 F8-M／F15-M 的主要鈕滿版、F16-M 不滿版，是稿自己的不一致；本張照稿，不替設計師統一，已補記到 `disc-2026-09-flow-f-mobile-missing-state-screens` 請 Sally 一併看。
5. ⚖️ **SSE 膠囊的圓角（L16）不在本張。** 三處程式碼＋多張稿，會動桌機基準線、也跨進 6f-4 的檔案——另立 `disc-2026-09-sse-chip-pill-radius`，一次改完三處。11px 仍凍結。
6. ⚖️ **挑片畫面不為了拍照開後門。** 手機版面靠 e2e。

### 不要做的事

- 不要改 `GenerationProgressV2.tsx`（三處共用；6f-1 已完成手機版）。
- 不要把確認框的按鈕改成滿版；不要把分析中／空狀態的按鈕改成滿版。
- 不要替換既有的 `px-6`／`px-5`／`justify-end`／`min-h-[10rem]` token——只加 `max-sm:`。
- 不要改 `batch-subtitle.spec.ts` 的斷言；不要改既有 `list-mobile*` 夾具。
- 不要加在頁內的 390 夾具；不要本機產 `-linux.png`。
- 不要動 SSE 膠囊的圓角與 11px。

### 已知陷阱

- **`sm:contents` 包裝**：桌機上包裝層消失、盒模型不變（6f-2 已用 repo 的 Tailwind 4.1.18 實編＋真瀏覽器量過：產生順序 base → `max-sm:` → `sm:`，639／640 沒有死角）。
- **`ml-auto`／`justify-end` 會讓 `w-full` 失效**：滿版沒滿版，先看外面那一層。
- **兩份用量列同時在 DOM**：`getByText` 會 multiple matches。
- **`confirm-mobile` 是「差異」不是「缺少」**：bootstrap 只補缺少的——要先 `git rm` 舊的 `-linux`（四步流程）。
- **Pencil 的 throw 會 rollback 同一次呼叫的編輯**；存檔要 Cmd+S＋`stat`。
- **本機 e2e 要 `AI_PROVIDER=claude`**；GitHub API 當天斷過三次，`gh` 失敗先重試再判斷。
- **行號以建單時為準**（2026-09-21，main `c5b0c2be`）。

### Source tree

```
ux-design.pen、_bmad-output/pen-tokens.json、screenshots/flow-f-subtitle-v2/f8-m-v2.png   ← Task 1
apps/web/src/components/subtitle/GenerationBatchDialogV2.tsx(+spec)                        ← Task 2, 3
apps/web/src/components/subtitle/consent/GenerationConsentView.tsx(+spec)                  ← Task 2, 4, 5
apps/web/src/components/subtitle/consent/ConfirmGenerationDialog.tsx(+spec)                ← Task 2, 4
apps/web/src/components/subtitle/consent/CandidateListPanel.tsx(+spec)                     ← Task 4
apps/web/src/components/subtitle/consent/AnalysisProgressPanel.tsx(+spec)                  ← Task 4
apps/web/src/components/subtitle/consent/ConsentEmptyState.tsx(+spec)                      ← Task 4
apps/web/src/routes/test/-gallery.fixtures.tsx                                             ← Task 6
tests/e2e/batch-consent-mobile.spec.ts（新）、tests/support/*（若抽共用 stub）              ← Task 6
tests/visual/…/generation-batch-dialog-v2/{running,budget_ceiling}-mobile、generation-consent/confirm-mobile ← Task 6
```

### Cross-Stack Split Check

後端 task **0 個**，前端／設計／測試 task 6 個 → 不觸發跨棧拆分。規模：6 個元件但全是 `max-sm:` 加法＋一處結構變動、一張稿、兩個新夾具、一支 e2e——與 `dsr-6f-1`（7 個元件＋`styles.css`＋兩張稿）相當，不再拆。

### Time-dependent visual coverage

- **N/A — no wall-clock-reading components touched.** `GenerationBatchDialogV2.tsx:30` 與 `GenerationConsentView.tsx:24` 都明文宣告零時鐘讀取（進度／金額／數量全來自 SSE payload）；`consent/*` grep `Date.now|new Date|performance.now` 零命中。

### References

- [Source: `GenerationBatchDialogV2.tsx:218-231, 432-461, 464, 603-627, 631-694`、`consent/GenerationConsentView.tsx:111-127, 221-252, 478-490, 520-539, 597-625`、`consent/CandidateListPanel.tsx:793-794, 943-953, 963-975, 1043-1054, 1112-1204`、`consent/ConfirmGenerationDialog.tsx:83-102, 207-230`、`consent/AnalysisProgressPanel.tsx:37-39, 67-90`、`consent/ConsentEmptyState.tsx`]
- [Source: `GenerationBatchDialogV2.spec.tsx:255-268`、`ConfirmGenerationDialog.spec.tsx:76-85, 133-143`、`GenerationConsentView.spec.tsx:1058-1069`、`CandidateListPanel.spec.tsx:1063, 1503-1521`、`AnalysisProgressPanel.spec.tsx:22-47, 71-76`]
- [Source: `apps/web/src/routes/test/-gallery.fixtures.tsx:4808-4879, 5081-5096, 5158-5240`；`tests/e2e/batch-subtitle.spec.ts:132-195, 264-274`；`tests/e2e/glossary-mobile.spec.ts`、`manage-subtitle-mobile.spec.ts`（手機 e2e 的寫法）]
- [Source: `ux-design.pen` `H717g`／`bCZ9p`／`H1CYPa`／`mxssT`／`p6x759`／`pE5IE`／`RGssw`／`GSnOg`／`i9Nun1`／`fS5is`／`fdu4y`／`tbF3W`／`fusXx`／`DXXEC`／`XNgsZ`／`x45wBO`／`pYy2o`／`UXiBN`／`V7TDNe`／`IMQO6`／`jd4CX`／`Maz74`／`XkGvG`／`JzmvC`／`x1MC3W` —— SM 建單時以 Pencil MCP 逐節點讀出（2026-09-21）；程式碼現況由唯讀稽核代理查證]
- [Source: `DESIGN.md:351-360`（字階）、`:604-636`（手機規則、BottomSheet 80%）、`:618-622`（44×44）、`:836`（`problems` 基準；現況 70）]
- [Source: `dsr-6f-1-…md`（外殼、`top` 的算法、`ml-auto` 教訓）、`dsr-6f-2-glossary-mobile.md`（`sm:contents` 實測、CR H1 假斷言、M7 326 寬夾具、Pencil／e2e 的 Debug Log）、`dsr-6e-2-consent-small-surfaces.md`（CR L1／L5）、`dsr-6d-b-batch-dialog.md`（CR L16）]
- [Source: project-context.md#Rule 16／#Rule 20／#Rule 21／#Rule 23／#Rule 24；`.claude/memory/project_visual_baseline_intentional_change.md`、`feedback_measure_the_breakpoint_you_return_to.md`]

## Dev Agent Record

### Agent Model Used

Claude Fable 5.1 — dev-story (Amelia)

### Debug Log References

- Pencil `execute` 有 `Copy(src, parent)`（回傳新 id）；讀值仍用 throw（會 rollback，同次呼叫不要混寫入）。匯出第一次 176/196 失敗一個 chunk，腳本自己會擋（「do NOT commit」）——還原後重跑即 196/196。
- 本機 e2e：`AI_PROVIDER=claude`。**`batch-subtitle.spec.ts` 有兩條在本機紅**（POST body 多一個 `model_id: "claude-sonnet-5"`）：該 spec 沒有 stub `/settings/models`，本機後端有 Claude 金鑰所以模型目錄有回應、`ModelPicker` 預選後就帶上 `model_id`；CI 的後端沒有金鑰所以是綠的。與本張無關（元件改動不可能多送欄位），已立 `preexisting-fail-e2e-batch-subtitle-model-id-local`。
- `--update-snapshots=missing` 只補缺的，不會重生漂移的 darwin——`confirm-mobile` 要先刪掉舊圖才會重拍。

### Completion Notes List

**做了什麼（對使用者的差別）**

1. **三個抽屜的標題列在手機變成 44 高**（原本是桌機的 56），✕ 仍然正對標題列中心——e2e 量過三個抽屜「✕ 的垂直中心與標題列中心差 <2px」，這就是 `top-[22px]` → `top-4` 對不對的量法。
2. **批次的「本次用量」固定在頁尾了。** 以前它是捲動清單的最後一項，片子一多就被捲到看不見；現在手機上它在固定頁尾的最上面（e2e 量過：它的 y ≥ 捲動區底邊），下面是滿版的「全部取消」。桌機不變（仍在清單下方）。做法是同一個 `CostRow` 畫兩次、各自在另一個斷點 `display:none`。
3. **批次頁尾另外三種狀態**（取消確認／結束／達上限）在手機：句子各佔一行、兩顆鈕等寬並排。
4. **挑片**：左右內距 24 → 16（清單 x=16，e2e 量過）；預算輸入框從固定 96 寬變成撐滿到右邊（右緣 374）；「開始產生」本來就滿版（守門測試）。
5. **確認框**：只改了標題列高度與內容上下 20；**頁尾兩顆鈕維持靠右不滿版**（e2e 守：同一水平線、右緣 374、各自 <200 寬）——這是稿的意思，不是缺口。
6. **取消分析**：`cancelling` 在 `onClose()` 之後不再歸零，改在下次 bootstrap 歸零（元件層防呆；今天 production 因為父層關閉即卸載所以本來就看不到閃回）。
7. **設計稿 F8-M 追上現況**：標題「產生字幕」、範圍改純文字、2 / 5、步驟條換成 F3-M 的直排（⚖️ 2026-09-21）、清單留兩張卡、sheet 630→655 重新貼底（≤675）。`problems` 70 → **69**（`L676DC` 的 clip 消失；CR 後刪掉複製來的停用 45% 節點）。

**桌機零變動**：整支 visual 跑過，既有基準線除了刻意改的 `generation-consent/confirm-mobile` 之外**一張都沒變**（另一張紅的是本機固定會漂的 `retry-retry-notifications`）。e2e 在 640 那一側量過：`dialog-enter`、標題列 56、預算框 96、用量列在 body、步驟條橫排。

**閘門結果**

| 閘門 | 結果 |
| --- | --- |
| `format:check`／`lint:all` | ✅ 0 errors（129 warnings 與 main 相同；本張碰到的檔案上那 2 個是既有的 `attachSnapshot` deps 與 `useVirtualizer`，0 個新增） |
| `web:typecheck`／`check-design-tokens.py` | ✅ |
| `pnpm nx test web` | ✅ 3963/3963（新增 17 條，改寫 3 條） |
| `pnpm nx test api` | ✅ |
| e2e（chromium） | ✅ 新 spec 4/4，burn-in 12/12；`batch-subtitle.spec.ts` 7 條中 5 綠、**2 條本機紅是既有的環境相依**（見 Debug Log，CI 綠） |
| 視覺 | ✅ 新增 3 張 darwin（`running-mobile`、`budget_ceiling-mobile`、`confirm-over-budget-mobile`）、`confirm-mobile` 刻意重拍（舊 `-linux` 已 `git rm`，待分支上觸發 bootstrap） |

**🔍 /ship 對抗式 CR（2026-09-21，fresh-context 代理，只讀；實編 Tailwind 4.1.18 查 CSS 產生順序、Pencil 逐節點讀 F8-M、跑 406 條相關單元測試）**——0 HIGH／2 MEDIUM／7 LOW，**修 9、駁回 0**。

| 等級 | 問題 | 處置 |
| --- | --- | --- |
| **M1** | AC #6 只測了一半：沒有任何測試涵蓋「重新打開之後按鈕回到『取消』」——把歸零那一行刪掉，46 條照樣綠。而且「維持取消中…」那條包在 `waitFor` 裡，只要**曾經**是取消中就會過。 | 修：補一條 rerender `open` false→true 的測試；「維持」改成 `onClose` 之後的裸 `expect`。 |
| **M2** | 「四種頁尾狀態的按鈕各帶對的 token」只測了三種——「重試失敗項目」「再產生字幕」從沒被檢查過。 | 修：`it.each` 補兩種結束狀態。 |
| L1 | **真瑕疵**：歸零寫在 `bootstrap()` 裡，而 `bootstrap` 的 identity 會跟著 `preselectedIds`（新陣列）變——取消進行到一半若父層重傳選取，按鈕會被重新武裝、可以按第二次。 | 修：先寫紅測試（確實紅）→ 歸零改到只看 `open` 的 effect＋「重試」的 handler。 |
| L2 | JSX 重組後留下一個與外層重複的條件。 | 修。 |
| L3 | `.pen` 的 problems「仍是 70」其實是一進一出：`L676DC` 的 clip 沒了，但複製過來的那顆停用「45%」（`JgXZh`）變成新的 partially clipped。 | 修：`Rlam8` 是一般 frame 不是 instance，直接刪掉 `JgXZh` → **69**。 |
| L4 | 確認框 spec 的標題還寫「56px title row」；`ui/mobileSheet.tsx` 的註解還說標題列有 44 與 56 兩種。 | 修（`mobileSheet.tsx` **只改註解**）。 |
| L5 | e2e 把「開始產生寬 358」標成守門，但 main 上頁尾是 `px-6`、寬是 342——那條在改動前是紅的。 | 修：註解寫清楚「撐滿是守、358 是新的」。 |
| L6 | 640 那一側只量了挑片，批次的標題列與 ✕ 沒量。 | 修：補批次在 640 的標題列 56、把手隱藏、✕ <44。 |
| L7 | 新 spec 的像素斷言沒在 Pixel 5／iPhone 13 這些裝置 project 上試過（CI 只跑 chromium，但本機 `test:e2e` 會跑）。 | 修：非 `chromium` project 直接 skip。 |

代理同時實測確認：CSS 產生順序讓 `max-sm:hidden`／`sm:hidden`／`sm:contents` 都贏過 `flex`、`max-sm:flex-1` 贏過 `w-24`；`CostRow` 沒有 live region／id／`aria-describedby`，桌機 DOM 只多一個 class；三個 ref 都還接著、`sm:contents` 之下按鈕順序不變、沒有空的 flex 子項；`expectTitleBar` 真的分得出 `top-4` 與 `top-[22px]`（差 6px）；stub 搬移逐字相同；`.pen` 各節點與 `penSha256` 正確。

- 🔗 AC Drift: NONE（checked: `top-\[22px\]|gen-batch-cost-line|consent-start-btn|justify-end` across `_bmad-output/implementation-artifacts/*.md`——命中 dsr-6f-1（`top-[22px]` 是它為 56px 標題列算的，並明文寫「改了標題列高度要一起改」→ 本張就是那次改動）、dsr-6d-b／dsr-6e-1／6e-2（桌機版面，全部仍成立：本張只加 `max-sm:`）。用量列在桌機的位置與 testid 不變。）
- 📎 Contract Stamps: FOUND（上游 dsr-6d-b／dsr-6e-2 對 `[@contract-v1]`（dsr-6d-a AC #1–#7、sub-4-1 AC #8、sub-6-8a AC #2）的 ack 仍成立——本張純呈現層，不改任何消費；本張自己不定義契約）
- 🎭 A11y Pre-Flight: PASS（6 個元件；碰到的檔案 0 個新增 jsx-a11y warning）。用量列兩份各在另一個斷點 `display:none`，不會被念兩次；`gen-batch-status-live`／`consent-phase-live` 未動；Esc 守衛、`aria-disabled` 行為未動；新 wrapper 是純 `<div>`＋`sm:contents`。
- 🎨 UX Verification: PASS —— 對 `f8-m-v2`（改稿後）／`f15-m-v2`／`f16-m-v2`／`f19-m-v2`：

| Area | Design Spec | Implementation | Match? |
| --- | --- | --- | --- |
| 三個標題列 | 44 高、[0,4,0,16]、底線、✕ 44×44 | 同（e2e：高 44、✕ 置中） | ✅ |
| F8-M 用量列＋SSE | 固定頁尾、Label 12 | 同（`text-xs`，e2e：在 body 底邊之下） | ✅ |
| F8-M 全部取消 | 滿版 h44 | 同（358） | ✅ |
| F8-M 步驟條 | 直排（⚖️ 改稿） | 直排 14px | ✅ |
| F8-M 卡片 | padding 12 gap 12 | `max-sm:p-3`＋`gap-3` | ✅ |
| F15-M 內距／預算框／開始產生 | 16／撐滿／滿版 | 同 | ✅ |
| F15-M 列 | — | 沿用 `bugfix-f15-row-mobile-identity-collapse` | ✅ 未動 |
| F16-M／F19-M 內容 | [20,16] gap 16 | `p-4 max-sm:py-5 gap-4` | ✅ |
| F16-M／F19-M 頁尾 | 靠右、不滿版 | 同（e2e 守） | ✅ |
| F16-M／F19-M 字階 | 標題／lead BodyLg 600、模型名 Body、金額 Body 700、合計 BodyLg 700 | 對照 `confirm-mobile`／`confirm-over-budget-mobile` 基準線一致 → **dsr-6e-2 CR L1 結案** | ✅ |
| SSE 膠囊圓角 | `radius-sm`（稿）vs DESIGN.md 藥丸 | 未動 | ⚠️ `disc-2026-09-sse-chip-pill-radius` |

### Discovery Triage

- **Did this story discover any work outside its current scope?** **YES** —— 建單當下處理如下。

- **① expand-scope-in-place**
  - F8-M 整張過時（標題、範圍切換、12/38、45%、死覆寫）→ **AC #1**
  - 三支 spec 釘死 `top-[22px]` → **AC #2**
  - 批次用量列會被捲走 → **AC #3**
  - dsr-6e-2 CR L5（取消鈕閃回；元件層防呆）→ **AC #6**
  - dsr-6e-2 CR L1（`confirm-mobile` 系列夾具＋F16-M／F19-M 字階）→ **AC #5**
  - Rule 21 標頭缺 F8-M → **AC #3**

- **② spawn-blocking-story**：無。

- **③ backlog-with-carry-forward-link**（建單時立／補記／實作時新增）
  - `preexisting-fail-e2e-batch-subtitle-model-id-local`（**實作時新立**）— `batch-subtitle.spec.ts` 沒 stub `/settings/models`，本機有 Claude 金鑰時 POST 會多帶 `model_id`，兩條斷言在本機紅、CI 綠
  - `disc-2026-09-batch-consent-sheet-exit-never-plays`（新）— 批次／挑片的父層 `if (!open) return null`，關閉即卸載，6f-1 的 `sheet-exit` 在這兩個抽屜永遠播不到
  - `disc-2026-09-sse-chip-pill-radius`（新）— Flow F 三處 SSE 膠囊 `radius-sm` → 藥丸（dsr-6d-b CR L16；會動桌機基準線與 6f-4 的檔案）
  - `disc-2026-09-flow-f-mobile-missing-state-screens`（補記）— F14／F17／F18／F20 與批次三種頁尾狀態沒有手機稿；F8-M／F15-M 主要鈕滿版而 F16-M 不滿版的不一致（owner: Sally）
  - `dsr-6f-4-workspace-mobile`（補記）— ⚖️ 步驟條手機一律直排

### File List

**新增**

- `tests/e2e/batch-consent-mobile.spec.ts`
- `tests/support/helpers/batch-subtitle-stubs.ts`（自 `batch-subtitle.spec.ts` 原樣搬出，兩支 spec 共用）
- `tests/visual/…/generation-batch-dialog-v2/{running-mobile,budget_ceiling-mobile}/default-visual-darwin.png`
- `tests/visual/…/generation-consent/confirm-over-budget-mobile/default-visual-darwin.png`
- `_bmad-output/implementation-artifacts/dsr-6f-3-batch-and-consent-mobile.md`

**修改**

- `apps/web/src/components/subtitle/GenerationBatchDialogV2.tsx`（+spec）
- `apps/web/src/components/subtitle/consent/{GenerationConsentView,CandidateListPanel,ConfirmGenerationDialog,AnalysisProgressPanel}.tsx`（+spec）、`ConsentEmptyState.tsx`
- `apps/web/src/routes/test/-gallery.fixtures.tsx`（三個 390×844 Portal 夾具）
- `apps/web/src/components/ui/mobileSheet.tsx`（**只改註解**）
- `tests/e2e/batch-subtitle.spec.ts`（只改 import：stub 搬到共用檔，斷言零改動）
- `tests/visual/…/generation-consent/confirm-mobile/default-visual-darwin.png`（刻意重拍）
- `ux-design.pen`、`_bmad-output/pen-tokens.json`、`_bmad-output/screenshots/flow-f-subtitle-v2/f8-m-v2.png`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`

**刪除**

- `tests/visual/…/generation-consent/confirm-mobile/default-visual-linux.png`（過期；由 CI bootstrap 重產）

## Change Log

| 日期 | 內容 |
| --- | --- |
| 2026-09-21 | ✅ **DONE** —— PR #494 合併進 main（commit `3a9f64b8`）。PR 側 **17 pass／0 fail**；`-linux` 基準（3 張新夾具＋刻意重拍的 `confirm-mobile`）由手動觸發 `Visual Regression` 開出的 bootstrap PR #495 補齊（4 張 PNG＋1 行稽核紀錄，零原始碼改動）。**合併後 `main` 那個 commit 的 Visual Regression／Docker／Tests 三條都確認是 success。** |
| 2026-09-21 | 🔍 **/ship 對抗式 CR**：0H／2M／7L，修 9、駁回 0。最實在的一條：取消鈕的歸零原本寫在 `bootstrap()` 裡，父層重傳選取陣列就會在取消途中把按鈕重新武裝（紅測試證實）→ 改到只看 `open` 的 effect。補齊重新打開、兩種結束狀態、640 批次側的測試；`.pen` problems 70→69；web 3963/3963、e2e burn-in 12/12。 |
| 2026-09-21 | 🚧 **REVIEW**（dev-story, Amelia）。Task 1–6 完成。三個抽屜標題列手機 44 高（✕ `top-4`，三支 spec 改寫）；批次「本次用量」固定到頁尾（`CostRow` 畫兩次）＋四種頁尾狀態；挑片內距 16、預算框撐滿；確認框頁尾依稿維持靠右；取消分析不再於關閉途中歸零；F8-M 改稿（直排步驟條、2/5、兩張卡、655 高貼底）。閘門：lint 0 errors、typecheck ✅、tokens ✅、web 3959/3959、api ✅、新 e2e 4/4（burn-in 12/12）、視覺新增 3 張＋`confirm-mobile` 刻意重拍，其餘基準線零變動。 |
| 2026-09-21 | 🔍 **建單後對抗驗證**（fresh-context 代理，只讀；逐節點比對 `.pen`、逐行比對程式碼與 spec）：1 CRITICAL／9 SHOULD FIX／7 NIT，**全部併入**。最重要：① F8-M 的清單其實有**四**張卡，初稿的高度預算漏算——換成直排步驟條後三張卡約 746 會爆 675，改成明確只留兩張（≈660）；② 「『開始產生』不滿版」是錯的，它今天就滿版，初稿為它寫的「紅測試」不會紅——改標成守；預算輸入框的目標其實是外面那個 `<span>`，打到 `<input>` 測試會綠但版面不變；③ L5 的閃回在 production **看不到**（父層關閉即卸載），順帶發現 6f-1 的退場動畫在這兩個抽屜永遠播不到 → 另立單；④ 補上漏掉的 Rule 21 標頭、dsr-6e-2 CR L1（F19-M 手機夾具）、用量列字級與卡片內距。代理同時確認：所有行號、三支 spec 的 `top-[22px]`、✕ 改 `top-4` 的算術（三個對話框同一個 offsetParent）、F16-M／F19-M 頁尾確實靠右不滿版、e2e 路徑在 390 寬可走且不必改 stub、挑片畫面確實沒有任何可種 state 的 seam。 |
| 2026-09-21 | Story 建立（SM Bob, create-story）。SM 以 Pencil MCP 逐節點讀 F8-M／F15-M／F16-M／F19-M 與桌機對照，唯讀代理查六個元件、九支 spec、21 個夾具與既有 e2e 的現況。找到 18 項；最重要的——① 6f-1 只換了外殼，三個抽屜的標題列仍是桌機的 56 高，而 ✕ 的 `top-[22px]` 被三支 spec 釘死；② 批次的「本次用量」在捲動區裡會被捲走，稿把它放在固定頁尾；③ F8-M 整張過時；④ sprint 條目說的「確認框按鈕不滿版」**不是缺口**，稿本來就靠右。⚖️ Alexyu 當日裁定：步驟條手機直排（改稿不改碼）。另五項建單裁定，新立 1 張 disc 單、補記 2 條。 |
