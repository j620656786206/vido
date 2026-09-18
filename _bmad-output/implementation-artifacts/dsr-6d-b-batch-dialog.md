# Story DSR.6d-b：批次生成對話框改看後端給的真相——失敗不再顯示成「完成」，關掉再開也接得回去

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As the person who starts a batch from 活動 or 媒體庫 and watches it in the dialog,
I want 每一列寫的就是那部片真正的結果、關掉再打開還接得回去、取消失敗時它會說話,
so that 我知道哪幾部要重試（而且按得到重試），也不會被一個永遠停在「進行中」的畫面騙。

## Context

`dsr-6d`（批次生成＋生成工作區）拆出來的**第二塊**。第一塊 `dsr-6d-a`（後端）已完成（PR #464，commit `b8541d4e`）：批次進度現在帶**每一部的狀態與原因**、狀態查詢帶**最後一次結果 `last`**、開始的回應帶 **`progress`**、分集帶**劇名**、執行中的 SSE 只送 **`changed_item`**、終態才送整份 `items`。**本張把對話框接上去**，第三塊 `dsr-6d-c`（生成工作區）之後再做。

| 單子 | 範圍 |
| --- | --- |
| `dsr-6d-a` ✅ | 後端：每一部的狀態、`last`＋dismiss、202 `progress`、`error` 終態、分集劇名 |
| **`dsr-6d-b`（本張）** | 批次對話框 F8-D-v2 `i9Nun1`（對話框 `Fkiqd`）、F9-D-v2 `JMqPg`（`v45TX`）→ `GenerationBatchDialogV2.tsx`、`useGenerationBatchProgress.ts` |
| `dsr-6d-c` | 生成工作區 F11–F13、`Component/GenQueueRow-v2`、工作區的 bug |

⛔ **同意流程（`components/subtitle/consent/**`、F14–F20）屬 `dsr-6e`**，接縫只有 `onStartBatch`／`preselectedIds`／`forceAnalyze`。本張不進 `consent/` 一步。
⚠️ **驗收基準是 `.pen` 節點值**（文字逐字、字級變數、顏色變數、間距）。PNG 只是參考。

### 🔴 建單時查到的事（main `c181f3fa`；行號皆為現況）

1. **每一列的狀態是「猜」出來的，猜錯就把失敗寫成「完成」。** `deriveRowStates`（`:68-95`）只有計數與索引可用：沒被標記失敗的一律當 `done`（`:76`）、`budget_ceiling` 假設暫停的是最後 `pausedCount` 列、`cancelled`／`error` 用 `currentMediaId` 在 `items` 裡的位置切一刀。真實情境：第 4 部「正在別處處理」被拒（後端不送逐部事件）→ `fail_count` 加一，但第 4 列顯示「完成」，`failedRowIds` 是空的，`onRetryFailed` 因此是 `undefined`（`:821`），**「重試失敗項目」按鈕整顆不會出現**——使用者沒有任何路徑回到那部片。
2. **失敗還會算到錯的那一列（新發現）。** 兩個 effect（`:653-657` 與 `:664-673`）都看 `currentMediaId`：A 失敗被記下後，下一個批次事件把 `currentMediaId` 換成 B，但那一次 render 裡 `perItem.progress.phase` **還是 `failed`**（重設要到下一次 render），於是 **B 也被加進 `failedIds`**。B 明明正在跑卻顯示「失敗」，終態後還會被「重試失敗項目」預選。現有測試抓不到（兩條競態測試都把 `currentMediaId` 固定住）。
3. **中途接上只畫一張卡。** `:417-455` 在 `items` 為空時用 `currentMediaId`／`currentItem` 拼一列。後端現在 status／409／202 都帶完整 `items` 了。
4. **409 的 `data` 可能是 `null`。** `generation_batch_handler.go:165` 用 `GetProgress()`；批次剛好在 Start 回 409 與讀取之間結束就是 `null`。前端型別（`subtitleService.ts:488`）寫成必有，`startBatchTracking(null)` → `seed ?? {}` → 畫面**永遠停在 `0 / 0`、狀態 running**。
5. **「全部取消」失敗會被吞掉。** `:767-774` 的 `catch {}` 是空的，`:514-517` 又在 promise 還沒回來前就把確認列收掉：網路斷線時，畫面看起來取消了，批次其實還在跑，而且沒有任何訊息、沒有「取消中…」。
6. **佇列快取結束後不清。** 檔頭 `:57-58` 寫「Cleared on terminal」，但終態 effect（`:679-685`）只 invalidate 片庫與 preview，`handleClose`（`:776-787`）也沒清。批次結束、關掉對話框後打開工作區，`GenerationWorkspaceV2.tsx:577` 還讀得到那 5 筆舊資料。
7. **進度條在終態仍是泥金**（`:398`，整個區塊 `:377-403` 每個狀態都畫）。DESIGN.md:293 泥金＝正在跑；批次結束了還亮著就是在騙人（先例 dsr-4:78）。
8. **兩顆按鈕穿了硃砂。** 「確定取消」`:519`、「重試失敗項目」`:546` 都用 `error-tint`／`error-text`。兩者都不是不可復原的破壞性動作；重試更是**復原**動作（DESIGN.md:300「狀態色不得被挪用…它一旦出現，就是在對『狀態』做出主張，而那個主張必須為真」——程式碼用的是 `error-tint` 不是實心硃砂，所以要引這一條，不是 §Buttons 的 Destructive 規則；:296 硃砂＝壞了）。
9. **列上的「轉錄中」固定不看階段**（`:148-151`）：抽內嵌字幕的路線根本不轉錄，翻譯階段也還是寫「轉錄中」；而 `:197-201` 把 `failed` 映成 `idle`，所以失敗的那一列會「標籤寫失敗、下面的步驟條還在跑提取音訊」。
10. **`complete` 一律報「批次生成完成」**（`:273-274`），即使有 3 部失敗。畫面上沒有任何總結果的一句話，失敗只能逐列看。
11. **結束後刷新不完整**（`:683-684`）：只 invalidate `libraryKeys.all` 與 preview。缺 `detailKeys`（開著的詳情頁還寫「缺字幕」）、季集查詢、`activityKeys`（活動頁那一列是舊的）、單片估價（`ManageSubtitleDialogV2.tsx:236-240` 的先例）、以及第 6 點的 items 快取。
12. **焦點會掉。** 「全部取消」按下去之後，按鈕被換成確認列（`:487-523`），被聚焦的元素消失、沒有人接手，焦點掉回 `<body>`（與 `disc-2026-09-glossary-panel-a11y-leftovers` ① 同類）。
13. **`deriveRowStates` 不是只有這裡用。** `GenerationWorkspaceV2.tsx:31,334` 會 import，`ActivityHub.spec.tsx:16-23` 的 `vi.mock` 工廠也逐一 re-export（`generationBatchPreviewKey`／`generationBatchItemsKey`／`deriveRowStates`）。**刪掉或改簽名會讓工作區編譯失敗、讓活動頁測試整支紅**。
14. **hook 收到 `changed_item` 也丟掉。** `useGenerationBatchProgress.ts:82-98` 的 `SSE_UPDATE` 逐欄複製 11 個欄位，`items` 與 `changed_item` 都沒接；`items` 目前只有 `START` 那條路能塞（`:80-81`）。
15. **`last` 不能用 `startTracking` 塞。** `START` 的 reducer 把 `status` 寫死成 `running`（`:81`）而且會**開一條 EventSource**（`:188`）：拿 `last` 餵進去會把已結束的批次畫成進行中，還留一條沒人關的連線。
16. **11px 凍結**（`disc-2026-09-11px-micro-label-not-on-type-scale`）：SSE 膠囊 `:475` 的 `text-[11px]` **原樣保留**；只收 6 處 13px。
17. **e2e 與夾具都還是舊形狀**：`tests/e2e/batch-subtitle.spec.ts` 的 202（`:116-123`）沒有 `progress`／`series_title`，409（`:358-383`）沒有 `items`，status（`:209-211`）沒有 `last`；兩個 gallery 夾具（`:4885-4986`）的 `progress` 沒有 `items`、`items[]` 沒有 `seriesTitle`。夾具的 `props` 是 `Record<string, unknown>`，**typecheck 抓不到**，不同步更新就會畫出空佇列。

### 設計稿節點

| 代號 | 節點 | 本張要看的 |
| --- | --- | --- |
| F8-D-v2 | `i9Nun1`（對話框 `Fkiqd` 880、`$border-subtle` 1px） | 標題 `OEYMD`、標題列 `tJKu3`、內容 `bUpff`[20,24]、範圍列 `V9LuY`（`STldN`「範圍：」＋兩顆獨立 chip `vvyEq`／`fxkko`，**不是分段控制**）、已完成 `hK7Or`／數字 `LkNhg`／`dP5Lo`(12 / 38)／軌道 `xnzZn`／填色 `M2wvy8`、列 `L91L5s`（沙丘那一列）、進行中的列 `K3X0m` 與它的集數碼 `f6dnS6`、完成勾 `m4haV`／`qserD`(皆 16)／`e003m`、轉錄中 `RCXv0`、排隊中 `jure8`、步驟縮排 `OurAy`(48)／instance `MJlft`、成本 `gmdHT`／`ICPL8`／`da87j`、SSE `Blkcw`／`u2nle`、footer `g6bYon`／`hb4bs`、註記 `bp5aN` |
| F9-D-v2 | `JMqPg`（`v45TX`） | 標題 `DJkYj`、標題列 `BNMAl`、橫幅 `GNV6A`（`$warning-tint` 底、`Bc8Ps` 圖示 `$warning`、文字 `H5EL7`／`mL8Gw`／`Z7I4h`／`h97TVY` **已是中性 `$text-primary`**）、範圍文字版 `G7sGD`（`RVwv8`／`OVoNY`／後綴 `eag82`）、已暫停 `cptH3`／`HtXeF`(皆 14，正確)／`yhSFD`、完成勾 `H3Kt44`／`gQwFy`／`B5NzaB`（皆 14，**與 F8 的 16 不一致**）、footer `gLRHV`：`ZI9nb` 關閉／`GxWnJ` 重試失敗項目／`oSvfd` 下次繼續（Primary `otvKh` 14/600 px20） |
| 母版 | `Component/Button/Secondary` `YDPhc`（h36→稿上覆寫 44、`$bg-tertiary`、14/500、px20）、`Component/Button/Primary` `otvKh`、`Component/GenerationProgress-v2` `XkGvG`（dsr-6b 已對齊） | 兩顆 footer 按鈕、步驟條 |

字階（`DESIGN.md:390-394`）：Label 12／1.5、Body 14／1.625、BodyLg 16／1.625。

---

## Acceptance Criteria

1. **設計稿先改（Pencil MCP；動手前每個節點再 `Get` 一次）。**
   - **標題統一**：F8 `OEYMD`、F9 `DJkYj` 的標題文字「批次生成字幕」→「**產生字幕**」（同一個對話框的同意畫面 F14 `bsV9h`／F15 `G6vQEv` 與程式碼 `:320` 都是「產生字幕」；三個畫面同一顆對話框不能有兩個名字）。
   - **範圍那一行**：F8 `V9LuY` 底下的**兩顆 chip `vvyEq`（缺字幕的項目）與 `fxkko`（已選項目）**（含它們的子文字 `grLGK`／`FlzYM`／`yf0WI`／`oRWTl`）整組移除，只留 `STldN`「範圍：」並接上 F9 的文字版樣式；兩張稿的文字都改成程式碼的實況「範圍：已選項目（**5** 部）」（`scope=missing` 已在 sub-4-3 移除，批次一律 `scope=selected`；F9 改 `RVwv8`／`OVoNY`／後綴 `eag82`）。
   - **數字要互相對得上**（改成 5 部之後）：F8 的 `dP5Lo`「12 / 38」與進度填色 `M2wvy8`、F9 橫幅的 `Z7I4h`「12」／`h97TVY`「26」（12＋26＝38）、F9 的 `OVoNY`「38」全部改成與 5 一致的一組數字（例如 2 / 5、已完成 2、剩餘 3），否則同一張稿自相矛盾。
   - **F9 拿掉 `GxWnJ`「重試失敗項目」**：預算上限時只有「關閉」＋「下次繼續」（sub-5-3 CR M1，程式碼 `:537-541` 早就這樣；下次繼續已經同時帶失敗與暫停的項目）。
   - **完成圖示尺寸統一**：F9 的**三顆**完成勾 `H3Kt44`／`gQwFy`／`B5NzaB` 14 → **16**（與 F8 的 `m4haV`／`qserD` 一致；程式碼是 `h-4`）。⛔ 暫停圖示 `cptH3`／`HtXeF` 維持 14（程式碼就是 `h-3.5`）。
   - **轉錄中的「45%」只在翻譯階段**：F8 的步驟條 instance `MJlft` 目前是 `{"id":"MJlft","type":"ref","ref":"XkGvG"}`（**沒有 `descendants`**）；`fVBQF`（`gp-st2-pct`，內容「45%」）在母版 `XkGvG` 的 `A8AT5n` 底下。照 dsr-6b 對 F3 的做法加一個 instance 覆寫，形狀逐字如下：
     ```json
     {"id":"NmhL0","type":"ref","ref":"XkGvG","descendants":{"fVBQF":{"enabled":false}}}
     ```
     即 `Update("MJlft", {descendants:{fVBQF:{enabled:false}}})`。⛔ **不准刪母版 `XkGvG` 的節點**（會連動 F3／F8-M／元件庫），⛔ 也不要用 `WSt82`（F4-D-v2）那種整個節點置換的寫法。
   - ⚠️ **手機稿不動**：F8-M `H717g` 的步驟條 instance `RGssw` 同樣沒有覆寫 `fVBQF`，改完之後 `f8-m-v2` 會跟 `f8-d-v2` 不一致——手機歸 `dsr-6f`，建單時已把這件事寫進它的 sprint 條目。
   - **補一段註記**（Flow F 的規格群組是 `JzmvC`；比照 dsr-6b 的做法，在既有的 `c4FIoB`（`note-gen-capability`）裡 **append 一個 text 子節點**，像 `b4A6k` 裡的 `QARsD`，不要新開一張畫面、也不要塞進 F8／F9 的畫面裡）：「列的狀態、失敗原因、劇名都來自後端的 `items[]`（dsr-6d-a）；執行中的 SSE 只送 `changed_item`，整份佇列來自 202／status／409。轉錄百分比只有翻譯階段有。」
   - **F8 的 `bp5aN` 註記**（「本 frame 為 all-in-one 規格參考…實際出貨 UI 為兩個分離狀態」）**保留**——它仍然成立（idle 已由同意流程取代，執行中才是本張）。
   - 新文字節點一律綁變數（`fontSize: $Type/...`、`fill: $...`）；⚠️ 稿上的範圍數字 `OVoNY` 是 Primary 14/400、程式碼是 `font-mono tabular-nums`，`Bc8Ps` 用 `$warning` 而程式碼用 `--warning-text`（`local/no-base-semantic-as-text` 擋基礎語意色當文字）——這兩處**以程式碼為準，稿不用改**，在註記裡寫明即可；每張改完 `ctx.problems` 掃裁切；存檔後確認 ` M ux-design.pen` 並讀磁碟檔確認落盤；匯出後只 stage `flow-f-subtitle-v2/{f8-d-v2,f9-d-v2}.png` ＋ `_bmad-output/pen-tokens.json`（若變），其餘重繪雜訊還原。**若 `f8-m-v2`、`f3-d-v2` 或元件庫也變了，代表動到了母版——回頭檢查。**

2. **每一列的狀態改讀後端（🔴 #1、#2、#13）。**
   - 新增 `apps/web/src/components/subtitle/generationQueueRow.ts`（＋spec）：把列的顯示語彙集中在一處。**檔案第一行必須是** `// Implements: <utility — no .pen counterpart>`（Rule 21；先例 `generationWorkspace.ts:1`、`generateCostView.ts:1`），否則 `local/implements-pen-node-id` 會讓 `lint:all` 紅——
     - `type QueueRowView = { status: GenerationBatchItemStatus; reason: GenerationBatchItemReason }`
     - `queueRowLabel(view, phase?)` → `{ text, className }`：`done`「完成」`--success-text`；`failed` 依 `reason`（見 AC #3）`--error-text`；`running` 依**目前階段**（AC #3）`--accent-text`；`paused`「已暫停 — 下次繼續」`--text-muted`；`cancelled`「已取消」`--text-muted`；`queued`「排隊中」`--text-muted`。
   - `QueueRow`（`:166-209`）改吃 `GenerationBatchItemState`：`status` 決定標籤與是否畫步驟條（只有 `running` 畫），`seriesTitle` 有值時標題顯示「{seriesTitle} {title}」、沒有就只有 `title`（⛔ 不准直接內插——e2e mock 與夾具都可能沒有這個鍵）。
   - **三層來源，優先序寫死**（文中一律區分 **`progress.items`（後端快照佇列）** 與 **`items` prop（202 的 `result.items`，只有 id／title／劇名）**）：
     1. `progress.items` 有內容 → 每一列的 id、標題、劇名、狀態、原因**全部**來自它。
     2. 否則 `items` prop 有內容 → 走既有的 `deriveRowStates`（退回路徑）。
     3. 兩者皆空、但 `progress.currentItem` 有值 → 維持既有的單張降級卡（`:417-455`）。⚠️ 那張卡自己組了一個 `GenerationBatchItem` 字面值（`:425-434`），改吃 `GenerationBatchItemState` 之後要補 `status`／`reason`（由批次狀態推：running→`running`、budget_ceiling→`paused`、complete→`done`、其餘→`cancelled`；`reason` 一律 `''`），否則 typecheck 紅。
   - **`data-state` 改寫後端的 `status`**（`:179`）：`active`→`running`、`stopped`→`cancelled`；spec `:256`、`:343` 斷言的 `'active'` 要一併改成 `'running'`。⛔ 退回路徑的 `RowState` 詞彙（`:66`）**不動**，所以 `:195`／`:206` 的 `'stopped'` 保持原樣。
   - ⛔ **`deriveRowStates`／`failedRowIds`／`remainingIds`／`generationBatchItemsKey`／`generationBatchPreviewKey` 的 export 全部保留**（`GenerationWorkspaceV2.tsx:31,334` 與 `ActivityHub.spec.tsx:16-23` 的 mock 工廠都依賴它們）；`deriveRowStates` 的檔頭補一行「僅供沒有 `items[]` 時退回使用；`dsr-6d-c` 之後評估移除」。
   - **`failedRowIds` 的語意不變**（sub-5-3 AC #3：重試預選＝畫面上看到標記失敗的那些）：有 `progress.items` 時用 `progress.items[].status === 'failed'`，沒有時走舊路（`deriveRowStates` + `items` prop）。`remainingIds` 同理（failed ∪ paused ∪ cancelled）。
   - **拿掉會算錯的那條路**：有 `progress.items` 時不再用逐部 SSE 推 `failedIds`（🔴 #2）。`failedIds` 只在退回路徑使用。

3. **列的文案（SM 定，Sally／Alexyu 可在 review 推翻）。**
   - `running`：依目前階段逐字顯示 **「提取音訊」「轉錄中」「翻譯中」**（與 `GenerationProgressV2.tsx:33-40` 的 `GENERATION_STAGES` 同一套詞——⛔ **不准為了一致去改 `GENERATION_STAGES`**，那是凍結詞彙，改名會打斷夾具與基準線的對應）。`phase` 是 `idle`／`complete`／`failed`／拿不到時（剛接上、還沒有逐部事件）→「處理中」。
   - `failed` 依 `reason`：`skipped` →**「沒有可用的字幕來源」**（⛔ 不要用「已略過」開頭：legacy 沒設定金鑰也走這個原因，說「略過」會讓人以為是系統自己決定不做）；`busy_elsewhere` →「這部正在別處處理」；`error` →「生成失敗」；`''`（舊後端）→「生成失敗」。
   - 步驟條：`status === 'running'` 才畫；失敗時**不畫**（現況會畫成「提取音訊中」的假動作，🔴 #9）。
   - **總結果一句話**（`complete` 時，🔴 #10）：全部成功→「全部完成（N 部）」`--success-text`；有失敗→「完成 N 部、失敗 M 部」**中性**（不綠、不紅——綠色是「有答案而且是好的那個」，有失敗就不成立）。`cancelled`→「已取消：完成 N 部」中性；`error`→沿用既有的錯誤橫幅。
   - **不要重複播報**：把這句**可見的**文字本身設成 `aria-live="polite"`，並讓 `:271-279` 的 `statusAnnouncement` 在 `complete`／`cancelled` 時回 `''`——否則 sr-only 的 `:326-328` 與可見文字會被螢幕報讀唸兩次。

4. **對話框外框與字級（對齊 F8／F9）。**
   - 面板 `DialogContent`（`:301-306`）：`sm:max-w-3xl` → `sm:max-w-[880px]`；加 `sm:border sm:border-[var(--border-subtle)]`（`Fkiqd`）。⚠️ 今天同意畫面與面板**都是 768**，所以這一步會**新產生**一次「同意 → 執行」的寬度變化（兩者是分開的 Dialog root，切換時本來就重新掛載，不是動畫中途跳動）。⚖️ SM 裁定：同意畫面（`consent/**`，768）**本張不動**——稿上 F14 560／F15 960／F8 880 本來就各自不同，同意與執行是兩個分開的 Dialog root（切換時本來就重新掛載），寬度統一與否由 `dsr-6e` 決定。
   - 內容區 `:324` `p-6` → `px-6 py-5`（`bUpff` [20,24]）。
   - 6 處 13px → `text-sm`：`:332`（範圍）、`:347`（上限橫幅）、`:370`（錯誤橫幅）、`:380`（已完成）、`:462`（成本）、`:502`（取消確認問句）。⛔ SSE 膠囊 `:475` 的 `text-[11px]` 不動（🔴 #16）。
   - 數字列 `:379` `items-baseline` → `items-center`（`LkNhg`）；`gap-[3px]`（`:332`／`:347`／`:462`）→ `gap-1`（`$Space/xs`）。
   - 步驟條縮排 `:195` `sm:pl-[52px]` → `sm:pl-12`（`OurAy` 48），而且**要靠左**（稿上步驟條與海報左緣對齊）：`GenerationProgressV2` 目前是 `sm:justify-center`，改法是給它一個對齊用的 prop／className。⛔ **不准改 `GenerationProgressV2` 的母版樣式或預設值**（dsr-6b 才剛對齊，工作區與批次共用它）。
   - footer 兩顆 Secondary（全部取消 `:493`、關閉 `:531`）維持 `bg-tertiary`；`下次繼續` `:557` `px-6 font-medium` → `px-5 font-semibold`（`oSvfd` 是 Primary 14/600、padding [8,20]）。所有按鈕保留 `min-h-[44px]`。
   - 進度條依狀態換色（🔴 #7，先例 `components/downloads/downloadStatus.ts:50-74`「a bar is not text」，dsr-4:78「只有下載中是泥金、完成青碧、錯誤硃砂、其餘中性」）：`running`→`--accent-primary`；`complete` 且沒有失敗→`--success`；`error`→`--error`；其餘終態（`complete` 有失敗、`cancelled`、`budget_ceiling`）→`--text-muted`。⛔ **絕不用 `--bg-tertiary`**——那就是軌道自己的顏色（`:394`），填上去等於進度條消失。
   - 「確定取消」`:519`、「重試失敗項目」`:546` 改中性 Secondary（`bg-[var(--bg-tertiary)] text-[var(--text-primary)]`，🔴 #8）。破壞性實心硃砂留給真正不可復原的動作。

5. **接上後端的四個新來源（🔴 #3、#14、#15）。**
   - **hook（`useGenerationBatchProgress.ts`）**：
     - `SSE_UPDATE` 的 payload 型別補 `changedItem?: GenerationBatchItemState | null`；reducer 更新 `items`：`payload.items`（終態）優先；否則若有 `changedItem` **且** 目前 `items` 不是 null → 依 `mediaId` 取代該筆（找不到就原樣，不要新增）；否則保持原值。`items` 為 null 時收到 `changedItem` **不要**生出一筆假佇列。
     - 新增一個**不連線**的 seed 動作（例如 `attachSnapshot(progress)`）：把整個快照（含 `status`、`items`）寫進 state，**不開 EventSource**、**不把 status 寫死成 running**。`startTracking` 維持原樣給「真的要開始追」用。
   - **狀態查詢（`:627-648`）三條路**：`running && progress` → `startTracking(progress)`（含 `items`）；`!running && last` → `attachSnapshot(last)`，畫面直接顯示終態（F9 的上限畫面、完成、取消、錯誤）；兩者皆無 → 同意畫面。⛔ 不准用 `startTracking` 塞 `last`（🔴 #15）。
   - 🚨 **`last` 只接一次，而且終態一定要有出口**（否則會把對話框鎖死）：`last` 在後端是留到下一次開始、dismiss 或重啟才清（dsr-6d-a AC #3），而本張**不呼叫 dismiss**——若每次開啟都無條件接上 `last`，一個成功結束的批次會讓「產生字幕」永遠停在昨天的完成畫面，`selectedMediaIds` 被忽略、再也開不了新批次（終態面板只有「關閉」與可能沒有的「重試失敗項目」）。做法：
     - 用一個 `seenLastBatchIdRef`（或 state）記住**這一次開啟已經顯示過的 `last.batchId`**；同一個 `batchId` 第二次開啟就直接走同意畫面。
     - 終態面板在 `complete`／`cancelled`／`error` 且**沒有**「重試失敗項目」時，必須有一顆回到同意流程的中性 Secondary「**再產生字幕**」（`data-testid="gen-batch-restart-btn"`）：`resetBatch()` ＋ `setItems([])`（比照 `handleResume` 的機制）⛔ **不呼叫 `startGenerationBatch`**。
     - 測試要蓋住：終態接上 → 關掉 → 再打開 → 看到同意畫面；以及按「再產生字幕」回到同意畫面且沒有任何 API 被呼叫。
   - **202（`:717-727`）**：改用 `outcome.result.progress` seed（`batchId`／`totalItems`／`budgetUsd`／`items`）。`progress` 為 `null` 時（舊後端）**只 seed `{batchId, totalItems}`，佇列交給 `items` prop ＋ `deriveRowStates` 的退回路徑畫**（⚠️ 建單時原本寫「組一份全 `queued` 的佇列」塞進 `progress.items`——那會讓 AC #8 點名的 `:538`／`:559`／`:719` 整列變「排隊中」，**以本行為準**）。🔴 **`progress` 不是 `running` 時要走 `attachSnapshot`，不能 `startTracking`**（極短批次可能在 202 被讀到之前就結束了，`START` 會把 status 寫死成 running 並開一條沒必要的連線——和 🔴 #15 是同一個陷阱）。**`batchId` 為 null（空 scope 的 200，`generation_batch_handler.go:189-195`）→ 完全不 seed、不進執行中畫面**：留在同意畫面並用既有的 `setStartError` 顯示「沒有可以生成的項目」（現在 `?? ''` 會讓畫面卡在 `0 / 0` 的執行中）。
   - **409（`:709-716`）**：`subtitleService.ts` 的衝突分支型別改成 `progress: GenerationBatchProgress | null`（🔴 #4）；`null` 時不要 seed 成執行中，改成重新查一次 status（那一步會拿到 `last` 或 idle）。
   - **斷線重連要重查**：hook 在 **`es.onerror` 的 backoff 重連路徑**（`useGenerationBatchProgress.ts:157-164`）遞增一個 `connectionEpoch`；⛔ `startTracking` 的第一次連線（`:188`）**不算**，否則剛開始批次就會立刻重查一次 status。對話框看到 epoch 變了就重新查 status，`{running:false, last}` 就切到終態。這是「終態事件掉了 → 畫面永遠停在進行中」（9R-16 CR L1）唯一的補救。
   - **`attachSnapshot` 要 reference-stable**：用 `useCallback(…, [])`（`dispatch` 本來就穩定），並加進 `:627-648` 那個 effect 的依賴陣列（現在是 `[open, startBatchTracking]`）——不穩定的函式會讓 effect 無限重跑。
   - ⛔ **對話框不呼叫 `dismissGenerationBatch()`**：清掉後端記住的結果是工作區 F12「關閉」的事（`dsr-6d-c`）；對話框的「關閉」只停止觀看。

6. **取消與快取（🔴 #5、#6、#11）。**
   - 取消：`:767-774` 不再吞錯——送出中「確定取消」顯示忙碌（`aria-disabled`＋文案「取消中…」，比照 dsr-6c 的做法：**不要用 `disabled`**，焦點掉到 `<body>` 會讓 Esc 行為錯亂）；成功才收掉確認列；失敗顯示一行 `role="alert"`：「取消失敗，批次仍在進行。請再試一次。」`data-testid="gen-batch-cancel-error"`。
     - 為此 `onConfirmCancelAll` 的型別要從 `() => void`（`:230`）改成 `() => Promise<void>`，容器的 `:819` 一併改；面板要 `await` 它之後才收確認列（`:514-517` 現在是同步收掉）。既有 spec `:272`（斷言只呼叫一次、確認列消失）與 `:489` 要改成 `await`／`findBy*`，否則會出現 act 警告。
   - 終態時（`:679-685`）除了現有的 `libraryKeys.all`＋`generationBatchPreviewKey`，再 invalidate：`detailKeys.all`（`hooks/useMediaDetails.ts:15-25`，前綴已含季與分集）、`activityKeys.all`（`hooks/useActivity.ts:13-15`）、`transcriptionEstimateKeys.all`（`hooks/useTranscriptionEstimate.ts:15`；⛔ **不是** `.item(mediaType, mediaId)`——批次有 N 部、型別還可能混），並 `removeQueries(generationBatchItemsKey)`（檔頭 `:57-58` 的承諾）。
   - 🚨 **只在「真的看著它結束」時做一次**：加了 AC #5 的 `attachSnapshot(last)` 之後，終態 effect（`:679-685`）在**每次開啟**都會觸發。用一個 ref 記下已處理過的終態 `batchId`，只有第一次看到那個 `batchId` 進入終態才 invalidate／`removeQueries`／`setPostTerminal(true)`；否則每開一次對話框就整批刷新片庫、詳情、活動與估價。
   - ⛔ **關閉對話框時不要清 items 快取**：關閉只是停止觀看、批次還在跑，而 `GenerationWorkspaceV2.tsx:577-582` 靠這個 key 畫佇列（檔頭 `:54-59` 的承諾就是「終態才清」）。⚠️ 交代 `dsr-6d-c`：終態 `removeQueries` 之後，工作區必須改讀 status 的 `last.items`，否則批次結束後工作區會失去佇列（建單時已寫進它的 sprint 條目）。
   - 焦點（🔴 #12）：切到取消確認列時，把焦點移到「繼續生成」（第一顆）；按「繼續生成」收掉確認列 → 焦點回「全部取消」；**取消成功後批次進終態、「全部取消」已經不存在（`:487` 只在 `isRunning` 時畫）→ 焦點移到「關閉」（`gen-batch-close-btn`）**；取消失敗 → 焦點留在「確定取消」，讓 `role="alert"` 播報。用 ref＋`useEffect`，不要靠 `autoFocus` 的競態。

7. **既有的誠實行為不准回歸**（測試都要留著）：
   - 批次狀態勝過逐部事件：`budget_ceiling`／`cancelled` 時，正在跑那一部即使收到逐部失敗，也要顯示「已暫停」「已取消」（9R-16 CR；spec `:184`、`:538`、`:559`）。有 `items` 之後這由後端保證，**但退回路徑仍要守住**。
   - 「已完成」＝成功＋失敗（`:267`，Sally 裁定）。
   - 「重試失敗項目」只在真正終態、**永遠不在 `budget_ceiling`**（spec `:657`、`:669`）。
   - 「下次繼續」「重試失敗項目」一律回到同意流程、**不直接開始批次**（spec `:579`、`:719`）。
   - 關閉只是停止觀看，批次繼續跑（`:776-778`）；Esc／點外面只在執行中被擋（spec `:347`、`:354`）；探測沒回來前不渲染（spec `:397`）；終態後下一次同意強制重新分析（spec `:591`）。
   - SSE 膠囊只在執行中顯示（`:472`）。⚠️ 這條、以及「點外面在執行中不關」（`:295-300`）目前**沒有任何測試**（spec 只有 `:263` 斷言執行中會出現、`:354` 測的是 Esc 不是點外面）——AC #8 要**補**，不是「保留」。

8. **測試（先寫紅測試）。**
   - `generationQueueRow.spec.ts`（新）：六種 status × 三種 reason 的文案與 class、`running` 依 phase、缺 phase 的退路。
   - `GenerationBatchDialogV2.spec.tsx`：
     - **hook 的 `vi.mock` 工廠（`:36-43`）必須補** `attachSnapshot: h.batchAttachSnapshot` 與 `connectionEpoch: h.batchEpoch`（新的 `vi.hoisted` 欄位），否則 15 支容器測試全部 `attachSnapshot is not a function`、CI 紅。
     - hoisted 的 `h.batchState`（`:7-19`）補 `items`，**預設值是 `null`**（保留退回路徑的覆蓋率）；items-first 的新測試各自 override。`progressOf`（`:139-157`）**已經有 `items: null`**（`:152-154`），不用動。服務 mock（`:53-63`）補 `dismissGenerationBatch`（本張不呼叫，但避免之後漏掉）。
     - ⚠️ AC #5 的「202 `progress` 為 null → 用 `result.items` 合成全 `queued` 佇列」**不算 items-first**：四個 mock 站點（`:435`、`:526`、`:709`、`:775`）都是 `progress: null`，若合成的佇列也走 items-first，`:538`／`:559`／`:719` 會整列變成「排隊中」。合成結果只能餵給退回路徑。
     - **保留不動**（AC #2 明訂這些路徑仍在）：`:164`／`:174`／`:195`／`:206`／`:611` 是 `deriveRowStates`／`failedRowIds` 的**純函式**測試（`ITEMS` 沒有 `status` 欄位、`progressOf` 的 `items` 是 null），它們是退回路徑唯一的覆蓋；`:306`／`:325`／`:336` 是 `progress.currentItem` 單張降級卡唯一的覆蓋。
     - **新增**（不是改寫）：一組 items-first 的測試，斷言 `progress.items` 有內容時，列的狀態、失敗原因、劇名全部來自它（含 `budget_ceiling` 的 paused、`cancelled`）。
     - **改寫**：`:423` 的探測測試改成三條路（running／`last`／idle）；`:432` 的 seed 斷言改成 202 `progress`；`:256`／`:343` 的 `data-state` 斷言 `'active'` → `'running'`。
     - **新增**：失敗原因三種文案；「別處處理」的那一列顯示失敗且重試鈕出現（🔴 #1 的回歸測試）；連續兩部中第一部失敗後、第二部不會被誤標（🔴 #2）；409 `data` 為 null 不會卡在 `0 / 0`；重連 epoch 變化會重查 status 並切終態；取消失敗顯示 alert 且確認列不收；終態清掉 items 快取；終態 invalidate 的四組 key；進度條終態不是泥金；兩顆按鈕是中性 Secondary；`seriesTitle` 有值時標題是「劇名 集數標題」、沒有時不出現 `undefined`；焦點在確認列切換時的落點。
     - **保留不動**：AC #7 列的那些。
   - `useGenerationBatchProgress.spec.ts`：`changedItem` 合併（取代該筆、找不到不新增、`items` 為 null 時不生假佇列）、終態帶整份 `items`、`attachSnapshot` 不開連線且保留 `status`、重連時 epoch 遞增。
   - e2e `tests/e2e/batch-subtitle.spec.ts`：202 mock 補 `progress`（含 `items[]`，每筆 `{media_id,title,media_type,series_title,status,reason}`）與 `series_title`；409 補 `items`；status 補 `last: null`。⛔ 不要改那兩處 `toEqual` 的請求主體斷言（`:314`、`:347`）。
     - ⚠️ 409 的 `items[]` **必須包含** `media_id: '9ff0c000-dead-4bee-8f00-000000000999'`、`title: '正在處理的電影'`、`status: 'running'`——那兩條斷言（`:391-397`）本來是降級卡畫出來的，換成 items-first 之後要由 `items[]` 提供，否則測試紅。
   - `pnpm run test:e2e -- --grep @batch-subtitle`（`tests/e2e/batch-subtitle.spec.ts:245` 的 describe 標籤；⛔ 不要用 `@story-9R-16`——那個標籤不存在，grep 會 0 個測試卻回報成功）在本機跑過一次，結果寫進 Completion Notes。

9. **視覺夾具與基準線。**
   - 更新 `generation-batch-dialog-v2/{running,budget_ceiling}`（`-gallery.fixtures.tsx:4885-4986`）：`progress.items` 補齊（狀態與畫面一致）、`items[]` 補 `seriesTitle`（至少一筆是分集，用來驗劇名）、修掉 `budget_ceiling` 夾具裡 `currentItem` 與 `currentMediaId` 對不上的資料。
   - 新增兩個夾具：`generation-batch-dialog-v2/complete-with-failures`（唯一會畫「重試失敗項目」與失敗列的狀態，目前完全沒有基準線）與 `generation-batch-dialog-v2/error`（`gen-batch-error-banner` 從來沒被截過圖）。
   - 基準線流程照 `project_visual_baseline_intentional_change.md`：`--update-snapshots=all` 後只留上列、其餘還原；改過的 `-linux` 一律 `git rm` 交給 CI bootstrap。⛔ 不要本機產 `-linux.png`。
   - ⚠️ 夾具的 `props` 是 `Record<string, unknown>`，typecheck 抓不到漏欄位——改完一定要看圖確認佇列有畫出來。

10. **另立的單子（建單時已寫入 sprint-status，不在本張做）。**
    - `disc-2026-09-batch-sse-items-transform-cost`：兩個 SSE hook 會把終態事件的整份佇列跑一次 `snakeToCamel`（2,400 項約 14,400 個新物件），工作區的事件紀錄 hook 根本不讀它（dsr-6d-a CR M4；本張只處理批次 hook 這一半，紀錄 hook 歸 `dsr-6d-c`）。
    - 既有：`disc-2026-09-activity-batch-row-count-in-flight-index`、`disc-2026-09-dialog-close-target-44px`、`disc-2026-09-dialogframe-shadow-vs-shadow-xl`、`9R-17-ai-usage-endpoint`。

11. **CI 全綠**：`pnpm run lint:all`、`pnpm nx run web:typecheck --skip-nx-cache`、`python3 scripts/check-design-tokens.py`、`pnpm nx test web`、`pnpm nx test api`（本張不改後端，但全套閘門照跑）。⛔ 絕不用 `run_in_background` 跑測試；每次跑完 `pnpm run test:cleanup`。Nx daemon 建圖逾時就用 `NX_DAEMON=false` 重跑並記在 Completion Notes。

## Tasks / Subtasks

- [x] **Task 1 — 設計稿（AC: #1）**
  - [x] F8／F9 標題、範圍行、F9 拿掉重試、完成圖示 16、F8 步驟百分比 instance 覆寫、補註記
  - [x] `ctx.problems`；存檔並 grep 落盤；匯出後只 stage f8-d-v2／f9-d-v2（＋pen-tokens）
- [x] **Task 2 — 列的顯示語彙（AC: #2 前半, #3, #8 第一項）**
  - [x] 先寫紅測試 `generationQueueRow.spec.ts` → `generationQueueRow.ts`
- [x] **Task 3 — hook 接 `changed_item`／`attachSnapshot`／重連 epoch（AC: #5 前半, #8）**
  - [x] 先寫紅測試（合併規則、不開連線、epoch）→ 實作
- [x] **Task 4 — 對話框改讀 `items[]`（AC: #2, #3, #7, #8）**
  - [x] 先寫紅測試：三種失敗原因、別處處理會顯示失敗且有重試、第二部不被誤標、退回路徑仍守住批次狀態優先
  - [x] `QueueRow`／`RowStageLabel` 改吃 `GenerationBatchItemState`；`failedRowIds`／`remainingIds` 雙路徑
- [x] **Task 5 — 探測三條路、202／409、取消、快取與焦點（AC: #5 後半, #6, #8）**
  - [x] 先寫紅測試：`last` 走終態、409 null、重連重查、取消失敗 alert、終態清快取與 invalidate、焦點落點
- [x] **Task 6 — 外框與字級（AC: #4）**
  - [x] 880＋`sm:border`、`px-6 py-5`、6 處 13px、`items-center`、`gap-1`、`sm:pl-12`、下次繼續、進度條終態中性、兩顆按鈕中性
- [x] **Task 7 — 夾具、基準線、e2e（AC: #8 後半, #9）**
- [x] **Task 8 — 收尾（AC: #10, #11）**
  - [x] 全套閘門；dev-story Step 9 截圖比對（`f8-d-v2`、`f9-d-v2`）

## Dev Notes

### 這張的重點

- **真相已經在後端**（dsr-6d-a）：本張的價值是「畫面不再自己猜」。猜錯造成的是使用者看得到的謊言（失敗寫成完成、重試鈕消失、第二部被誤標失敗）。
- **退回路徑要留**：舊版後端、或還沒收到任何快照時（`items` 為 null），`deriveRowStates` 仍是唯一可用的東西；而且工作區與活動頁測試都還依賴那些 export。
- **`last` 是這條線的重點**：關掉再打開、或終態事件掉了，畫面現在有辦法回到正確的結果。塞 `last` 一定要用不連線的 seed。

### 上游契約（Rule 20 ack）

- confirmed against [@contract-v1] (Story dsr-6d-a AC #1) —— `items[]` 每筆 `{media_id,title,media_type,series_title,status,reason}`，六個鍵恆存在；`reason` 只在 `failed` 有值（`skipped`／`busy_elsewhere`／`error`）。
- confirmed against [@contract-v1] (Story dsr-6d-a AC #2) —— `generation_batch_progress` 每次 13 個鍵：執行中 `items: null` ＋ `changed_item`；終態 `items` 完整 ＋ `changed_item: null`。
- confirmed against [@contract-v1] (Story dsr-6d-a AC #3) —— `GET …/status` 是 `{running, progress, last}`；`last` 是終態快照，跑的時候一定是 `null`。
- confirmed against [@contract-v1] (Story dsr-6d-a AC #5) —— 202 的 `data.progress`（可能為 `null`）；空 scope 的 200 不帶 `progress`。
- confirmed against [@contract-v1] (Story dsr-6d-a AC #7) —— `series_title` 一定出現，電影為 `""`。

### 建單裁定（2026-09-18，Sally／Alexyu 可在 review 推翻）

1. 面板寬度依稿 880；同意畫面的寬度留給 `dsr-6e`（稿上三個階段本來就不同寬，且兩者是分開的 Dialog root，切換時本來就重新掛載）。⚠️ 今天兩邊都是 768，所以這是**新產生**的一次寬度變化；建單時已把這個裁定寫進 `dsr-6e` 的 sprint 條目，免得被重新翻案。
2. 列的文案（AC #3）：階段名稱逐字沿用凍結的 `GENERATION_STAGES`（「提取音訊」「轉錄中」「翻譯中」）；失敗依原因分三種說法；`skipped` 一律說「**沒有可用的字幕來源**」，**不要**用「已略過」開頭——legacy 未設定金鑰也走這個原因（dsr-6d-a CR L8），說「略過」會讓人以為是系統自己決定不做。
3. 總結果：有失敗就不給綠色。
4. 進度條在終態改中性（dsr-4:78 的延伸）。
5. 「確定取消」與「重試失敗項目」改中性 Secondary（重試是復原動作，不是破壞）。
6. 對話框不碰 `dismissGenerationBatch()`。

### 不要做的事

- **不要進 `components/subtitle/consent/**`**（F14–F20 是 `dsr-6e`），也不要動 `GenerationConsentView` 的寬度。
- **不要刪 `deriveRowStates` 等 export**（🔴 #13）。
- **不要改 `ui/Dialog.tsx`**（關閉鈕、陰影、動畫——三張既有單子）。
- **不要把 `text-[11px]` 改成 12**。
- **不要做手機版**（F8-M `H717g` 歸 `dsr-6f`），`sm:` 以下維持現況。
- **不要改後端**；`9R-17` 的每片花費、活動頁計數都另有單子。
- **不要改 e2e 那兩處請求主體的 `toEqual`**。

### 已知陷阱

- **四個 mock 站點＋一個正式匯入者**：`ActivityHub.spec.tsx:16-23` 的 `vi.mock` 是整個模組的替身（逐一 re-export `generationBatchPreviewKey`／`generationBatchItemsKey`／`deriveRowStates`）；`LibraryBrowseV2.spec.tsx:45-48` 與 `routes/library.spec.tsx:49-51` 只 re-export `GenerationBatchDialogV2`；`GenerationWorkspaceV2.tsx:31,334` 與**正式程式碼** `scanner/ScanProgress.tsx:14`（用在 `:58`）也 import 這個模組的 export。本張若新增任何被別處 import 的 export，那些 mock 工廠也要補，否則整支測試紅。
- **`startTracking` 會開連線且把 status 寫死 running**（`useGenerationBatchProgress.ts:81, 188`）。
- **`snakeToCamel` 會遞迴進 `items[]`**（`utils/caseTransform.ts:6-17`）——終態事件會轉一整份佇列，成本記在 AC #10 的單子，本張不優化。
- **Radix Dialog 走 Portal**：spec 用 `screen` 找；同意→面板是兩個 Dialog root 的切換（`:622`／`:791`），焦點會重新種。
- **`toHaveTextContent` 是子字串比對**——文案要逐字；class 也要逐字（dsr-6b／6c 的教訓）。
- **視覺 CI 沒有後端**：夾具全靠 props，缺欄位不會被 typecheck 抓到。
- **行號以建單時為準**（2026-09-18，main `c181f3fa`）。

### Source tree

```
apps/web/src/components/subtitle/generationQueueRow.ts(+spec)          ← Task 2（新）
apps/web/src/hooks/useGenerationBatchProgress.ts(+spec)                ← Task 3
apps/web/src/components/subtitle/GenerationBatchDialogV2.tsx(+spec)    ← Task 4, 5, 6
apps/web/src/services/subtitleService.ts(+spec)                        ← Task 5（409 型別）
apps/web/src/routes/test/-gallery.fixtures.tsx                         ← Task 7
tests/e2e/batch-subtitle.spec.ts                                       ← Task 7
tests/visual/components.visual.spec.ts-snapshots/components/generation-batch-dialog-v2/** ← Task 7
apps/web/src/components/subtitle/GenerationWorkspaceV2.tsx             ← 只讀（確認 export 沒被破壞）
ux-design.pen、_bmad-output/pen-tokens.json、_bmad-output/screenshots/flow-f-subtitle-v2/{f8-d-v2,f9-d-v2}.png ← Task 1
```

### Cross-Stack Split Check

後端 task **0 個**，前端／設計 task 8 個。→ **本張不拆**。

### Time-dependent visual coverage

- **N/A — no wall-clock-reading components touched.** `GenerationBatchDialogV2` 與 `useGenerationBatchProgress` 都不讀時鐘（dsr-6d 稽核確認），本張也不加已用時間／ETA（dsr-6b 已裁定不顯示系統給不出的時間）。

### References

- [Source: `apps/web/src/components/subtitle/GenerationBatchDialogV2.tsx:1-32, 52-60, 66-126, 132-209, 245-566, 590-825`]
- [Source: `apps/web/src/hooks/useGenerationBatchProgress.ts:33-53, 70-104, 157-164, 185-191`、`hooks/useGenerationProgress.ts:42-48, 337-348`、`hooks/useMediaDetails.ts:15-25`、`hooks/useActivity.ts:13-15`、`hooks/useTranscriptionEstimate.ts:15`、`components/downloads/downloadStatus.ts:50-74`、`utils/caseTransform.ts:6-17`]
- [Source: `apps/web/src/services/subtitleService.ts:141-176（型別）, 397-409（StartResult）, 426-455, 488, 583-604, 637`]
- [Source: `apps/web/src/components/subtitle/GenerationWorkspaceV2.tsx:31, 334, 577-583`、`components/activity/ActivityHub.spec.tsx:16-23`]
- [Source: `apps/web/src/components/subtitle/GenerationBatchDialogV2.spec.tsx:7-19, 53-63, 131-157, 164-761`、`tests/e2e/batch-subtitle.spec.ts:116-123, 171, 209-211, 245（describe 標籤）, 314, 347, 358-383（409 mock）, 391-397（斷言）`]
- [Source: `apps/web/src/routes/test/-gallery.fixtures.tsx:4885-4986`、`tests/visual/components.visual.spec.ts:249-320`]
- [Source: `ux-design.pen` `i9Nun1`／`Fkiqd`／`JMqPg`／`v45TX`／`YDPhc`／`otvKh`／`XkGvG`／`bp5aN` — 唯讀稽核代理以 Pencil MCP 讀出（2026-09-17）＋建單時抽查（2026-09-18）]
- [Source: `_bmad-output/implementation-artifacts/dsr-6d-a-batch-status-backend.md`（AC #1–#8 的形狀與 CR 交代事項）、`ux3-subtitle-v2-batch.md`（AC 1／2／7、CR L1／L2）、`sub-5-3`（重試與下次繼續的紅線）、`dsr-6b`／`dsr-6c`（字級收斂、aria-disabled、逐字斷言的做法）]
- [Source: `DESIGN.md:291-300`（狀態色表）、`:308-320`（金額）、`:390-394`（字階）、§Buttons；`sprint-status.yaml` → `dsr-6d-b-batch-dialog`／`disc-2026-09-11px-micro-label-not-on-type-scale`]
- [Source: project-context.md#Rule 5 / #Rule 16 / #Rule 20 / #Rule 21 / #Rule 23 / #Rule 24]

## Dev Agent Record

### Agent Model Used

Claude Opus 5 (1M context) — dev-story (Amelia)

### Debug Log References

- Nx daemon 的建圖偶爾逾時（`nx serve web` 直接掛掉、`Calculating the project graph on the Nx Daemon is taking longer than expected`），全部閘門與 dev server 一律加 `NX_DAEMON=false` 重跑。
- 本機跑 e2e 需要 `AI_PROVIDER=claude`（`.env` 預設 `gemini` 但沒有 `GEMINI_API_KEY`，後端起不來）。副作用：同意流程因此拿得到模型清單，會在請求主體多送 `model_id` → 兩條 `toEqual` 斷言在本機紅。**已用乾淨工作樹（`git stash`）驗證同樣紅**，是本機環境造成，不是本張的回歸；CI 沒有模型清單，那兩條會綠。⛔ 依 AC 沒有改那兩處 `toEqual`。

### Completion Notes List

**做了什麼（對使用者的差別）**

1. **列上寫的就是那部片真正的結果。** 佇列改讀後端 `progress.items[]` 的 `status` 與 `reason`。被拒的那一部（已在別處處理、沒有可用的字幕來源）以前顯示「完成」、`failedRowIds` 是空的、「重試失敗項目」整顆不出現；現在它顯示失敗、寫出原因，重試鈕也回來了。
2. **失敗不會算到隔壁那一列。** 有後端佇列時完全不再用逐部 SSE 推 `failedIds`（那條路的 effect 直接 return），🔴 #2 的競態在正式路徑上消失。退回路徑（沒有 `items[]` 的舊後端）仍保留原行為與原測試。
3. **關掉再打開接得回去。** 狀態探測三條路：`running && progress` → `startTracking`；`!running && last` → 新的**不連線** `attachSnapshot`（保留終態、不開 EventSource、不把 status 寫死成 running）；兩者皆無 → 同意畫面。`last` **同一個 `batchId` 只接一次**（`seenLastBatchIdRef`），而且終態面板在沒有「重試失敗項目」時一定有一顆中性的「再產生字幕」（`gen-batch-restart-btn`）——否則後端記著的 `last` 會把對話框永遠鎖在昨天的結果。
4. **斷線重連會重查。** hook 的 backoff 重連路徑（不是第一次連線）遞增 `connectionEpoch`，對話框的探測 effect 把它放進依賴陣列 → 終態事件掉了也救得回來。
5. **取消失敗會說話。** 容器不再吞錯（`handleConfirmCancelAll` 直接讓例外往上丟），面板送出中顯示「取消中…」＋`aria-disabled`（不是 `disabled`），成功才收確認列，失敗顯示 `role="alert"`「取消失敗，批次仍在進行。請再試一次。」焦點也接住了：全部取消→「繼續生成」、繼續生成→「全部取消」、取消成功→「關閉」、取消失敗→留在「確定取消」。
6. **結束後畫面是新的。** 終態（**同一個 batchId 只做一次**）除了片庫與 preview，再 invalidate `detailKeys.all`／`activityKeys.all`／`transcriptionEstimateKeys.all`，並 `removeQueries(generationBatchItemsKey)`。關閉對話框**不**清快取（批次還在跑，工作區還在畫那份佇列）。
7. **不再用顏色說謊。** 進度條 running→泥金、complete 且零失敗→青碧、error→硃砂、其餘終態→中性（⛔ 不用 `--bg-tertiary`，那等於進度條消失）。「確定取消」與「重試失敗項目」改中性 Secondary（重試是復原動作）。有失敗的 `complete` 不給綠色：「完成 N 部、失敗 M 部」中性；全成功才「全部完成（N 部）」青碧。
8. **一句話總結只播報一次。** 可見的總結那一行自己帶 `aria-live="polite"`，`statusAnnouncement` 在 `complete`／`cancelled` 回 `''`，螢幕報讀不會唸兩次。
9. **失敗的列不再演戲。** 步驟條只有 `status === 'running'` 才畫；`running` 的標籤依目前階段逐字用凍結的 `GENERATION_STAGES`（提取音訊／轉錄中／翻譯中），拿不到階段時說「處理中」，不假裝知道。

**設計稿（Task 1）**

- F8 `OEYMD`／F9 `DJkYj` 標題統一「產生字幕」；F8 範圍行的兩顆 chip 移除、改成文字版「範圍：已選項目（5 部）」；F9 拿掉 `GxWnJ`「重試失敗項目」；F9 三顆完成勾 14→16；F8 步驟條 instance `MJlft` 加 `descendants:{fVBQF:{enabled:false}}`（母版 `XkGvG` 未動）；能力邊界註記 `c4FIoB` 內 append `J46QF`。
- ⚠️ **F9 的數字自相矛盾，做 Step 9 截圖比對時才發現並修掉**：稿上有 3 列「完成」＋2 列「已暫停」，但建單時要求的示例數字是「已完成 2／剩餘 3」。以**列為準**改成 `3 / 5`、橫幅「已完成 3 部，剩餘 2 部」、進度填色 333→499（60 %）。視覺夾具 `budget_ceiling` 一併改回 3 done／2 paused。
- `ctx.problems` 掃描：全檔 70 筆，與 `git show HEAD:ux-design.pen` 逐一比對**完全相同**（F8 `gwhXf`、F9 `a47yZ`／`v45TX` 三筆都是既有的），本張沒有新增裁切。
- 匯出後只 stage `flow-f-subtitle-v2/{f8-d-v2,f9-d-v2}.png` ＋ `pen-tokens.json`（只有 `penSha256` 變）；`flow-i-discover-v2` 的四張重繪雜訊已還原。**`f8-m-v2`、`f3-d-v2`、`design-system/component-library.png` 都沒有變**（確認沒動到母版）。

**閘門結果**

| 閘門 | 結果 |
| --- | --- |
| `pnpm run lint:all` | ✅ 0 errors / 128 warnings（全部既有）、prettier all clean |
| `pnpm nx run web:typecheck --skip-nx-cache` | ✅ |
| `python3 scripts/check-design-tokens.py` | ✅ 夜行 33／日巡 33、82 變數、195 畫面、73 母版 |
| `pnpm nx test web` | ✅ 3739/3739（268 個檔案；新增 88 條，含 CR 後補的 10 條回歸測試） |
| `pnpm nx test api` | ✅（本張沒改後端） |
| e2e `--grep @batch-subtitle`（chromium） | 7 條中 5 綠；2 條紅是本機 `AI_PROVIDER` 造成的 `model_id`，乾淨工作樹同樣紅（見 Debug Log） |
| 視覺 | `running`／`budget_ceiling` 重生 darwin、過期 `-linux` 已 `git rm`；新增 `complete-with-failures`、`error` 兩張 |

**對抗式驗證（mutation check）**——把修法拿掉，測試要變紅，逐一驗過：

| 拿掉什麼 | 變紅的測試 |
| --- | --- |
| items-first（`backendQueue` 恆為 null） | 5 條 rows-from-items ＋ 🔴 #2 回歸測試 |
| `last` 只接一次的判斷 | 「同一個 last 只顯示一次」 |
| 容器把取消的例外吞回去 | 「取消失敗會顯示 alert」 |
| 終態 once-per-batchId 的守衛 | 「同一個終態只 invalidate 一次」 |

**🔍 /ship 對抗式 CR（2026-09-18，fresh-context 代理，只讀）**——2 HIGH／6 MEDIUM／8 LOW，**修 12、交代 3、駁回 0**。

| 等級 | 問題 | 處置 |
| --- | --- | --- |
| **H1** | 202 的 `progress` **也可能是終態**（`SnapshotFor` 在 `Start()` 之後才讀；極短批次——例如唯一那部片被立刻拒絕——在讀到回應前就結束了）。走 `startTracking` 會把 status 寫死成 `running`：畫面永遠泥金、`全部取消` 打過去拿到 `{cancelled:false}` 卻被當成成功、終態 effect 永遠不觸發、Esc 與點外面都被擋住，只剩 ✕ 能逃。 | 修：非 `running` 的 202 快照改走 `attachSnapshot`（和 `last` 同一條規則） |
| **H2** | `seenLastBatchIdRef` 只在**探測**那條路寫入。**看著批次跑完**的那次不算，所以下次開啟會把同一批結果再播一次；如果那批有失敗，`重試失敗項目` 會把 `retryIds` 填進去，而 `preselectedIds={retryIds ?? selectedMediaIds}` **默默吃掉使用者剛選的片**。 | 修：終態 effect 也寫 `seenLastBatchIdRef`（跨「重新掛載」那一半另立 `disc-2026-09-batch-dialog-last-attach-per-mount`） |
| **M3** | 取消失敗的 alert 不會被「繼續生成」清掉 → 再按一次「全部取消」，確認列一出現就帶著上一次的紅字，`role="alert"` 會重新播報一個根本沒發生的失敗。 | 修 |
| **M4** | 把總結那一行做成「和內容一起掛載」的 `aria-live` 區域，螢幕報讀**不會唸**（AT 只播報既有區域的變動）。同時 `statusAnnouncement` 被清空 → 批次完成從「唸一次」變成「完全不唸」。 | 修：改由**一直掛著**的 sr-only 區域播報，可見那行改成純文字 |
| **M5** | `setPostTerminal(true)` 被關在 once-per-batch 守衛裡面，而 `handleClose` 會把它清成 `false` → 重新打開接上同一個 `last` 時不再強制重新分析，CR H2 的「過期候選快照禁令」倒退。 | 修：`setPostTerminal` 移到守衛之前 |
| **M6** | AC #5 要求的 409 型別（`GenerationBatchProgress \| null`）根本沒改——測試得寫 `progress: null as never` 才過，就是契約還在說謊的鐵證。 | 修：`subtitleService.ts` 改成可為 null（原本漏掉這個檔案） |
| **M7** | 取消成功後立刻收掉確認列，但批次要幾秒後才真的終止 → 被聚焦的按鈕消失、「關閉」還沒出現，焦點在那段空窗掉到 `<body>`；畫面也退回一顆普通的「全部取消」，看起來像什麼都沒發生。 | 修：**送出成功後維持「取消中…」**，直到批次真的回報終態才收（`!isRunning` effect 負責清） |
| **M8** | 409 空 body → 重查 status，但如果重查也查不到東西（批次結束又被 dismiss／伺服器重啟），使用者按了開始卻**什麼都沒發生**：沒面板、沒錯誤、沒轉圈。 | 修：先設一句「剛才那個批次已經結束了，請再按一次開始」，重查成功接上任何東西就清掉 |
| **L9** | `cancelGenerationBatch` 回 `{cancelled:false, running:false}`（沒東西可取消）被當成成功。 | 修：那種回應改成觸發重查 status |
| **L10** | 新的 `error` 視覺夾具餵了一個 `failed` 列**又**只給 `onRestart` —— 真實容器在那種資料下一定會畫「重試失敗項目」，等於幫一個產品做不出來的畫面立基準線。 | 修：改成 1 done ＋ 4 cancelled、`failCount: 0`（`finish(error)` 的真實形狀） |
| **L11** | 新加的「點外面不關」測試**不可能失敗**——Radix 的 DismissableLayer 在 jsdom 根本不發事件，負向斷言穩過。 | 修：jsdom 那條刪掉並寫明為什麼，改到 e2e（真瀏覽器）驗 |
| **L12** | `failedRows` 的 `useMemo` 註解宣稱避免了每個 SSE tick 重走佇列，但 `batch.progress` 每個 tick 都是新物件。 | 修：註解改成實話（items-first 分支只是一次 filter，便宜） |
| **L13** | AC #5 的 202 退回寫法與實作相反（AC 說合成全 `queued` 佇列，實作交給退回路徑；AC #8 自己又說合成的不能走 items-first）。 | 修：改 AC 文字，以實作為準 |
| **L15** | e2e 的 409 mock `total_items: 38` 但 `items[]` 只有 1 筆——後端不可能送出這種快照。 | 修：改成 3 筆一致的佇列（含一筆 `busy_elsewhere`，順便驗失敗原因） |
| L14 | `phase: 'complete'` 到批次 `changed_item` 之間，列寫「處理中」但步驟條已全綠。 | 交代 `dsr-6d-c`（比舊版寫死「轉錄中」好） |
| L16 | `gen-batch-sse-chip` 用 `radius-sm`，DESIGN.md 規定徽章是藥丸形。 | 交代 `dsr-6f`（緊鄰凍結的 11px） |
| — | `seenLastBatchIdRef`／`handledTerminalRef` 是 ref，**跨元件重新掛載會歸零**（`ActivityHub.tsx:314` vs `:358`）。 | 新立 `disc-2026-09-batch-dialog-last-attach-per-mount`（根治要靠 `dsr-6d-c` 呼叫 `dismiss`） |

CR 後 mutation check：把上面每一條修法拿掉，**13 條測試變紅**（含 H1、H2、M3、M4、M5、M7、M8、L9 各自的回歸測試）。

**已知殘留（刻意，不在本張）**

- 🔴 #2 的競態在**退回路徑**仍在（沒有 `items[]` 的舊後端）。要根治得讓 `useGenerationProgress` 一起吐出事件所屬的 `mediaId`，超出本張範圍；`deriveRowStates` 的檔頭已註明只供退回使用，`dsr-6d-c` 再評估移除。
- 本機兩張與本張無關的視覺基準會漂（`parse-floating-parse-progress-card`、`retry-retry-notifications`）。**乾淨工作樹同樣漂**，diff 的紅點在側欄的片庫計數與儲存空間數字上——那是本機資料庫內容，不是程式問題，CI 用固定資料不受影響。已還原，未提交。

### Discovery Triage

- **Did this story discover any work outside its current scope?** **YES** —— 建單當下處理如下。

- **① expand-scope-in-place**
  - 列狀態用猜的、失敗顯示成完成、重試鈕不出現（🔴 #1）→ **AC #2**
  - 失敗算到錯的那一列（🔴 #2）→ **AC #2**
  - 中途接上只畫一張卡（🔴 #3）→ **AC #5**
  - 409 `data` 為 null 卡在 0 / 0（🔴 #4）→ **AC #5**
  - 取消失敗被吞（🔴 #5）→ **AC #6**
  - items 快取不清、刷新不完整（🔴 #6、#11）→ **AC #6**
  - 進度條終態仍泥金、兩顆按鈕穿硃砂（🔴 #7、#8）→ **AC #4**
  - 列上「轉錄中」不看階段、失敗仍畫步驟條（🔴 #9）→ **AC #3**
  - `complete` 有失敗也報「完成」（🔴 #10）→ **AC #3**
  - 確認列切換時焦點掉到 body（🔴 #12）→ **AC #6**
  - hook 丟掉 `changed_item`、`last` 不能用 `startTracking` 塞（🔴 #14、#15）→ **AC #5**
  - e2e 與夾具還是舊形狀（🔴 #17）→ **AC #8、#9**

- **② spawn-blocking-story**：無（依賴的 `dsr-6d-a` 已完成）。

- **③ backlog-with-carry-forward-link**（建單時立）
  - `disc-2026-09-batch-sse-items-transform-cost` — 終態事件的整份佇列會被無用地轉換（紀錄 hook 那一半歸 `dsr-6d-c`）

- Reference: `project-context.md` Rule 24

### File List

**新增**

- `apps/web/src/components/subtitle/generationQueueRow.ts`（Rule 21 檔頭 `// Implements: <utility — no .pen counterpart>`）
- `apps/web/src/components/subtitle/generationQueueRow.spec.ts`
- `tests/visual/components.visual.spec.ts-snapshots/components/generation-batch-dialog-v2/complete-with-failures/default-visual-darwin.png`
- `tests/visual/components.visual.spec.ts-snapshots/components/generation-batch-dialog-v2/error/default-visual-darwin.png`

**修改**

- `apps/web/src/components/subtitle/GenerationBatchDialogV2.tsx`
- `apps/web/src/components/subtitle/GenerationBatchDialogV2.spec.tsx`
- `apps/web/src/components/subtitle/GenerationProgressV2.tsx`（只加 `align?: 'center' | 'start'`，預設值與既有 DOM 完全相同）
- `apps/web/src/hooks/useGenerationBatchProgress.ts`
- `apps/web/src/hooks/useGenerationBatchProgress.spec.ts`
- `apps/web/src/services/subtitleService.ts`（CR M6：409 的 `progress` 改成可為 `null`）
- `apps/web/src/routes/test/-gallery.fixtures.tsx`
- `tests/e2e/batch-subtitle.spec.ts`
- `tests/visual/.../generation-batch-dialog-v2/{running,budget_ceiling}/default-visual-darwin.png`
- `ux-design.pen`、`_bmad-output/pen-tokens.json`、`_bmad-output/screenshots/flow-f-subtitle-v2/{f8-d-v2,f9-d-v2}.png`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`

**刪除**

- `tests/visual/.../generation-batch-dialog-v2/{running,budget_ceiling}/default-visual-linux.png`（過期基準；讓 CI 的 bootstrap 重新補）

⛔ **沒有碰**：`components/subtitle/consent/**`、`ui/Dialog.tsx`、任何後端檔案、`GenerationWorkspaceV2.tsx`（只讀確認 export 沒被破壞）。

## Change Log

| 日期 | 內容 |
| --- | --- |
| 2026-09-18 | ✅ **DONE** —— PR #466 合併進 main（commit `df4c3a93`）。CI 全綠：Lint、Unit、Go、4 個 E2E shard、3 個 Build、Serve Smoke、4 個視覺 shard。視覺第一輪 4 個 shard 紅,查證是**純粹缺 `-linux` 基準、零像素差異**（四行都是 `A snapshot doesn't exist at …-visual-linux.png`）,手動對分支觸發 Visual Regression workflow → bootstrap PR #467（4 張 PNG ＋ 一行稽核紀錄,無原始碼）→ 合進分支 → 第二輪全綠。本機 e2e 那 2 條 `model_id` 紅在 CI 是綠的,證實只是本機 `AI_PROVIDER` 的環境副作用。 |
| 2026-09-18 | 🔍 **/ship 對抗式 CR**（fresh-context 代理，只讀）：2 HIGH／6 MEDIUM／8 LOW，修 12、交代 3。最重要：① 202 的快照也可能是終態，`startTracking` 會把它畫成永遠在跑（連取消鈕都在說謊）→ 改走 `attachSnapshot`；② 看著批次跑完那次沒有記進 `seenLastBatchIdRef`，下次開啟會重播舊結果並吃掉使用者剛選的片；③ 總結那行的 `aria-live` 和內容一起掛載，螢幕報讀根本不唸 → 改由一直掛著的 sr-only 區域播報；④ 取消成功後立刻收確認列，但批次要幾秒後才終止，焦點掉到 `<body>` → 改成維持「取消中…」到真的終態；⑤ AC #5 要求的 409 可空型別漏改（測試靠 `as never` 才過）。web 3739/3739，13 條 mutation check 全部有牙。 |
| 2026-09-18 | 🚧 **REVIEW**（dev-story, Amelia）。Task 1–8 全數完成。設計稿改完後做 Step 9 截圖比對時發現 F9 的數字與列數自相矛盾（3 列完成但橫幅寫已完成 2），以列為準改成 `3 / 5`／已完成 3／剩餘 2，夾具跟著改。閘門：lint 0 errors、typecheck ✅、design-tokens ✅、web 3729/3729、api ✅；e2e `@batch-subtitle` 本機 5/7（2 條紅是 `AI_PROVIDER` 造成的 `model_id`，乾淨工作樹同樣紅）。四項修法做過 mutation check，拿掉就變紅。 |
| 2026-09-18 | 🔍 **建單後對抗驗證**（fresh-context 驗證代理，只讀；抽查 60＋條，含 Pencil 節點逐一比對）：9 項 CRITICAL、28 項 SHOULD FIX、多項 NIT，**全部併入**。最重要：① 無條件接上 `last` 會把對話框永遠鎖在上一次的終態畫面（`last` 只有 dsr-6d-c 的 dismiss 會清）→ 同一個 `batchId` 只接一次，並在終態補一顆「再產生字幕」回同意流程；② 終態 effect 會因此在每次開啟都觸發 → 加上「同一個 batchId 只處理一次」；③ 原本寫「關閉時也清 items 快取」會害工作區在批次還在跑時失去佇列 → 只在終態清，並交代 dsr-6d-c 改讀 `last.items`；④ 少了 Rule 20 的上游契約 ack（補 5 條）；⑤ hook 的 `vi.mock` 工廠沒補 `attachSnapshot` 會讓 15 支測試全紅；⑥ 新檔少了 Rule 21 檔頭會讓 lint 紅；⑦ e2e 的 grep 標籤寫錯會「0 個測試卻回報成功」；⑧ 原本要改寫的 5＋3 條測試其實是退回路徑唯一的覆蓋 → 改成保留並新增 items-first 測試。設計面：F9 標題／範圍後綴的節點 id 補上、範圍是兩顆 chip 不是分段控制、三顆完成勾都要改 16、改數字時整張稿的數字要一起對齊、`fVBQF` 覆寫的 JSON 形狀逐字寫明、手機稿的同一個問題交給 dsr-6f。 |
| 2026-09-18 | Story 建立（SM Bob, create-story）。承接 dsr-6d-a（已合併）的四個新來源：每一部的狀態與原因、`last`、202 `progress`、劇名。找到 12 個現況問題，其中兩個是新發現（失敗會算到錯的那一列；409 `data` 為 null 會讓畫面永遠停在 0 / 0）。六項 SM 裁定（寬度、列文案、總結果不給綠、終態進度條中性、兩顆按鈕中性、不碰 dismiss）。新立 1 張 disc 單。 |
