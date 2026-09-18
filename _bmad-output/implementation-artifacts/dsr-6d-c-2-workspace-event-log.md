# Story DSR.6d-c-2：生成工作區右欄「即時活動」說真話——每一部片都會出現、看得到是哪一部、失敗講中文、翻譯進度不再一個百分比一列

Status: review

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As the person who opens 活動 › 生成字幕 to watch a batch on a full page,
I want 右邊「即時活動」的每一列都寫清楚「是哪一部片、走到哪一步、成功還是失敗、為什麼」，而且一部片的翻譯進度只佔一列,
so that 我瞄一眼就知道現在在忙什麼、剛剛哪一部出了事，而不是看到一整排「翻譯中 12%、翻譯中 13%……」卻不知道是哪一部，或是一句看不懂的英文錯誤。

## Context

`dsr-6d`（批次生成＋生成工作區）的**最後一塊**。前面都已合併：

| 單子 | 範圍 | 狀態 |
| --- | --- | --- |
| `dsr-6d-a` | 後端：每一部的 `status`／`reason`、`last`＋`dismiss`、SSE `changed_item` | ✅ done（PR #464） |
| `dsr-6d-b` | 批次對話框 F8／F9、共用列詞彙 `generationQueueRow.ts` | ✅ done（PR #466） |
| `dsr-6d-c-1` | 工作區**左欄佇列＋頁首＋底部列** | ✅ done（PR #469） |
| **`dsr-6d-c-2`（本張）** | 工作區**右欄「即時活動」**：`useGenerationJobsFeed.ts` ＋ `EventLogPane` ＋ F11／F12 紀錄區的設計對齊 ＋ 一個小後端加欄位 | 本張 |

⛔ **本張不碰左欄佇列**（`QueueRow`、`OverallStrip`、`BudgetBanner`、底部列、探測三條路），唯一例外見 AC #4：single 模式那一列的**標題來源**（`GenerationWorkspaceV2.tsx:658`）改讀後端新給的 `title`——那是 `disc-2026-09-single-job-title-missing` 的根治，與右欄同一個根因。
⛔ **不碰對話框**（`GenerationBatchDialogV2.tsx`）、**不碰** `useGenerationBatchProgress.ts`／`useGenerationProgress.ts`。
⚠️ **驗收基準是 `.pen` 節點值**（文字逐字、字級變數、顏色變數、間距變數）。PNG 只是參考。

### 🔴 建單時查到的事（main `ff16deef`；行號皆為現況）

建單用了兩個唯讀稽核代理（設計稿節點、後端 SSE 事件），程式碼由 SM 自己讀。後端的關鍵說法（`failJob` 英文訊息、預算停止會先送 `transcription_failed`、ASR 退回路線的 `extracting`、5 分鐘上限）都已由 SM 逐行複核。

**A. 紀錄漏掉東西、或講錯東西**

1. **抽內嵌字幕的那一部完全不出現。** `TRANSCRIPTION_EVENTS`（`useGenerationJobsFeed.ts:90-94`）沒有 `subtitle_progress`。⚠️ 只在伺服器設 `VIDO_SUBTITLE_PIPELINE_MODE=pipeline` 時發生（預設 `legacy`，`config.go:217`；legacy 下每一部都走語音辨識，送 `transcription_*`）。pipeline 模式下有內嵌字幕的那一部**只**送 `subtitle_progress`＋`generation_batch_progress`（`adapter:85`、`process_item.go`）。
2. 🚨 **但 `subtitle_progress` 不能無條件聽。** 它有**三個**發送者，兩個不是生成：字幕搜尋引擎（`subtitle/engine.go:390-404`，**英文**訊息 "Searching subtitle providers..."）、手動下載字幕（`subtitle_handler.go:236-301, 485-498`，英文），以及生成 pipeline（`progress_sse.go:60-75`，中文）。掃描完之後的自動生成還會對每一部要付費的片送一次 `skipped`（`process_item.go:746`）——一次掃描可能上千列。**先例**：`useGenerationProgress.ts:200-205, 277-310` 用「這一部我先看過 pipeline 階段才相信終態」擋掉搜尋引擎的終態（sub-4-3 CR M7）。
3. **批次裡被預算擋下的那一部，紀錄寫「失敗」。** 預算用完時後端**先**送 `transcription_failed`（`error` 帶 `AI_BUDGET_EXCEEDED`，`transcription_service.go:669` 經 `failJob`），**然後**批次才把那一部標成 `paused` 並以 `budget_ceiling` 結束（`generation_batch.go:665-671`）。紀錄因此對一部**只是暫停**的片寫「失敗」，而左欄同一部寫「已暫停」——**兩欄對同一部講相反的話**，就是 dsr-6d-c-1 花一整張單子修掉的那一類。
4. **批次被取消時，正在跑的那一部也寫「失敗」。** 取消會讓那一部的 run 收到 `context canceled` 而走 `failJob`；批次隨後以 `cancelled` 結束（`generation_batch.go:661-664`）。
5. **pipeline 模式下走語音辨識退回路線的那一部，會出現兩次終態。** 它先送 `subtitle_progress` `extracting`（`process_item.go:529`），中間跑一整套 `transcription_*`，最後再送 `subtitle_progress` `complete`／`failed`（`process_item.go:571, 597`）。兩個家族都聽 → 同一部兩列「完成」。
6. **有些結果根本不會有 `transcription_*` 事件**：`busy_elsewhere`／`skipped`（`ErrTranscriptionInProgress`／`ErrTranscriptionDisabled` 在送事件前就回傳）、以及已經有合格字幕而提早結束的那一部（`process_item.go:71-72`）。它們**只**出現在批次的 `changed_item`（`generation_batch.go:674-694`）——而紀錄目前**完全不看 `changed_item`**（`BATCH` reducer `:190-208` 只看 `status`）。

**B. 洗版與播報**

7. **每一個翻譯百分比都新增一列。** `PHASE` reducer（`:127-155`）每個事件都 `appendFeed`。`translation_progress` 每 **10 句**送一次、沒有節流（`translation_service.go:297, 366-368`；批量 10，`prompts/subtitle_translator.go:34`）——一部 1,500 句的電影就是 **150 列**「翻譯中 N%」。`FEED_CAP = 200`（`:32`）於是**不到兩部片就把前面所有紀錄擠掉**。pipeline 模式的 `subtitle_progress translating` 也是每段一次（`pipeline.go:669`）。
8. **而且每一列都被螢幕報讀唸出來。** `<ol aria-live="polite">`（`GenerationWorkspaceV2.tsx:321-325`）。
9. **最新的在最下面，但不會自己捲下去。** `overflow-y-auto`（`:324`）沒有任何自動捲動；十幾列之後最新的事件在可視範圍外。設計稿那個節點名字就叫 `Autoscroll Label`（`rAWiU`，名字已過時但意圖在）。

**C. 看不出是哪一部**

10. **沒有片名。** `transcription_*` 與 `subtitle_progress` **都不帶片名**（後端稽核 §6）。紀錄的 `message`（`:142`）是後端的階段句子（「正在轉錄音訊」），不是片名。批次的片名只在 `changed_item.title`／`series_title`（`generation_batch.go:121-137`）與 `items[]`。
11. **批次事件那一列寫的是「最後一部的片名」。** `BATCH` 的 `message: action.payload.currentItem`（`:205`）——「已達本次預算上限 · 奧本海默」讀起來像奧本海默出了事，其實是整批。設計稿 `eEbpz` 寫的是「**本批次**」。
12. **單部任務（詳情頁觸發）根本沒有片名可用**——`disc-2026-09-single-job-title-missing`。但**後端其實已經算好了**：solo 任務在 `acquireJob` 之前就用 `resolveActivityTitle` 解出「片名」或「劇名 S01E02」（`transcription_service.go:455-486`），存在 `soloTranscriptionJob.Title`（`:431-435`），只是只給了活動頁、**沒有放進 SSE**。⚠️ 它解不到時會退回**原始 mediaID**（`:466, :470, :474, :481, :484`）——前端絕不能把它當片名顯示。

**D. 失敗講英文**

13. `failJob`（`transcription_service.go:1565-1578`）送 `error` 原字串、`message` = `"Transcription failed: " + error`；紀錄顯示 `payload.error ?? payload.message`（`:185`）。實際會出現的字串（後端稽核 §4）：`select audio track: no audio track found in media file`、`extract audio: ffmpeg not available`、`transcribe: … whisper: API error: status 500 — <body>`、`… context deadline exceeded`、`translate: … AI_BUDGET_EXCEEDED: …`、`save SRT: <os err>`（**會帶伺服器檔案路徑**）。**沒有機器可讀的原因碼**——只有字串裡的大寫前綴 `AI_BUDGET_EXCEEDED:`／`AI_TIMEOUT:`／`AI_UNAUTHORIZED:`。批次的 `changed_item.reason`（`skipped`／`busy_elsewhere`／`error`）才是結構化的，而且 `generationQueueRow.ts:48-54` 已有中文。

**E. 誠實性**

14. **SSE 膠囊不知道連線到底開沒開。** 右欄那顆（`:356`）無條件畫；jobs hook 沒有任何「已連線」狀態（`onopen` 沒接、`onerror` 只排 10 秒後重連，`:275-282`）。重連空窗那 10 秒裡膠囊照樣說「即時更新」，而那 10 秒的事件**永遠拿不回來**（hub 沒有 replay、不寫 `id:`，`sse/handler.go:70`；buffer 滿了靜默丟，`sse/hub.go:121-128, 164-175`）。⚠️ 與左邊統計列那顆**不同**：批次終態後這條連線**確實還開著**（jobs hook 從不因批次終態關閉），所以終態顯示它是對的——要判斷的是「連著沒有」，不是「批次結束沒有」。
15. **終態之後，紀錄裡「進行中」的那一列還在轉。** 沒有任何機制讓一列「不再是現在」。預算上限或取消之後，那一部最後一列仍是泥金＋（若有）百分比——DESIGN.md「動的東西＝正在發生的事」（Motion 段）。
16. **批次的每一部都被塞進 `singleJobs`。** `PHASE` reducer 不分批次或單部，每個 `transcription_*` 都寫進 `singleJobs`（`:145-153`）。批次在跑時沒事（模式由批次決定），但 **dsr-6d-c-1 CR M6 已證實**：按「關閉」之後批次回 idle，若 map 裡還留著某部（終態事件被丟、或沒送），`deriveWorkspaceMode` 就回 `'single'` → 畫面出現**假的**「進行中任務」。根因交代給本張。

**F. 效能（`disc-2026-09-batch-sse-items-transform-cost` 的工作區那一半）**

17. `parse`（`:234-241`）對**整份** payload 做 `snakeToCamel`（`:237`）。終態的 `generation_batch_progress` 帶完整 `items`——全選 2,400 部時是 2,400 個物件、約 14,400 次鍵重寫，而 `BatchPayload`（`:82-86`）只用三個欄位，**轉完立刻變垃圾**。工作區同時掛批次 hook 與 jobs hook 兩條 EventSource（`GenerationWorkspaceV2.tsx:837-838`）→ 同一份佇列被轉兩次。批次 hook 那一次是它**真的要用** `items`，不算浪費；jobs hook 這一次是本張要拿掉的。

**G. 設計對齊（右欄）**

18. **列的外觀與稿不符**（節點 → 程式碼）：稿上每列是 **16×16 圖示**＋階段字 **Body 600**＋`·`（`$text-muted`）＋項目 **Body `$text-secondary`**＋撐開＋尾端 **Mono Body**；程式碼**沒有圖示、沒有 `·`**，三段字都是 `text-xs`(12)（`:332`／`:336`／`:343`），列有 `rounded` 與 `px-2 py-1.5`（`:330`）、清單有 `px-1.5 space-y-0.5`（`:324`）；稿上列是 `[8,14]` gap 8、無圓角、清單 `[6,0]` 無 gap。
19. **欄頭少了 `activity` 圖示**（稿 `rDZQj`／`nNXwM` 14×14 `$text-secondary`），高度稿上是 44（程式碼靠 `py-2.5` 撐）。
20. **F12 底部第二行沒實作**：`v4Mp3s`（靠右）＋`F0IlN` `circle-pause` 13×13 `$warning-text`＋`Oqlsx`「已停止（達預算上限）」Label **500** `$warning-text`。
21. **紀錄的詞彙與設計稿三方不一致**：稿上有六種列（完成／提取音訊完成／本次用量／排入佇列／轉錄中 45%／已達預算上限），**其中三種系統給不出來**：
    - 「**排入佇列**」——SSE **沒有**排隊事件；排隊只在 HTTP 的 `items[].status=queued`（後端稽核 §5）。
    - 「**提取音訊完成**」——沒有專屬事件，只能從「下一階段的事件到了」推論。
    - 「**本次用量**」——只有批次有（`spent_usd`，每個事件都帶，但**只在每部結束時更新**），而左欄統計列**已經**在顯示同一個數字。
    - 而「**轉錄中 · 45%**」與稿上自己的註記 `J46QF`「轉錄百分比只有翻譯階段有」矛盾，也與左欄 F11 `OkdGK`（dsr-6d-c-1 已把 45% 關掉、步驟條停在「轉錄中」）同一部片講不同的話。
    - 反過來，程式碼**必須**產生的「翻譯中」「失敗」「已取消」「批次完成」「批次發生錯誤」在稿上**沒有畫**。
22. **稿上有兩處基礎語意色當圖示色**：`$success`（`O35JV`／`YtcfP`／`f0yNSe`／`ZYy6B`，F11／F12 evt-1、evt-2 的勾）、`$info`（`kxXPK`／`nb8eu`，兩顆 radio）。程式碼的 `local/no-base-semantic-as-text` 擋 `text-[var(--success)]`；dsr-6d-c-1 已把 `$warning` 當圖示色的四處改成 `$warning-text`（`Zeu0P`／`F0IlN`／`ShmdC`／`gavvw`）——本張照同一個先例。

**H. 測試與夾具**

23. **夾具在畫系統做不出來的列。** `running` 夾具有「本次用量 · $0.42」（`-gallery.fixtures.tsx:5128`），`budget_ceiling` 夾具有「已達預算上限 · $5.00」（`:5159`）——但 hook 實際產生的是「**已達本次預算上限**」＋**最後一部片名**、**沒有尾端金額**（`useGenerationJobsFeed.ts:109, 205`）。兩張基準線畫的都是產品不會畫出來的東西。`props` 是 `Record<string, unknown>`，typecheck 放行。
24. **既有測試把錯的行為釘死**：`useGenerationJobsFeed.spec.ts:61-84`（每個事件都新增一列）、`:110-132`（失敗顯示原字串）、`:134-149`（「已達本次預算上限」）、`GenerationWorkspaceV2.spec.tsx:182-194`（`aria-live` 在清單上）。
25. **e2e 對右欄零覆蓋**，而且 repo 裡**沒有任何** UI e2e mock 過 `/api/v1/events`（只有 `parse-progress.api.spec.ts` 驗 content-type）。

### 設計稿節點（本張要看的；唯讀稽核代理以 Pencil MCP 實測，2026-09-18）

| 代號 | 節點 | 內容 |
| --- | --- | --- |
| F11 紀錄欄 | `DUvwI`（在 `body` `dAHkG` 內，左邊 `gen-queue` 寬 712、gap 16） | 400 寬、高 fill、`clip`、`$bg-primary`、`$radius-md`、`$border-subtle` 1 inner |
| F11 欄頭 | `h8wLFJ`（h **44**、`[0,14]`、gap **10**、只有底線） | `rDZQj` lucide **`activity`** 14×14 `$text-secondary`（節點名「Terminal Icon」過時）、`hdjVa`「即時活動」Body **600** `$text-primary`、`zm4MC` 撐開、`rAWiU`「自開啟本頁起累積」Label **normal** `$text-muted`（節點名「Autoscroll Label」過時） |
| F11 清單 | `dPQ9b`（`[6,0]`，無 gap） | evt-1 `VxB7x`「完成 · 沙丘：第二部」、evt-2 `jque4`「完成 · 芭比」、evt-3 `a7UpAm`「提取音訊完成 · 奧本海默」(`audio-lines` `$accent-text`)、evt-4 `u4njFq`「本次用量 · 本批次 · $0.42」(`circle-dollar-sign`)、evt-5 `E7Me1`「排入佇列 · 花月殺手」(`hourglass`)、evt-6 `YtpPW`「轉錄中 · 奧本海默 · 45%」(`loader-circle` `$accent-text`) |
| 列（每一列同構） | `evt-N`（`[8,14]`、gap **8**、置中、實測高 39、無底色無圓角） | `evt-ic` **16×16**、`evt-stage` Body **600**、`evt-dot`「·」Body `$text-muted`、`evt-item` Body `$text-secondary`、`evt-sp` 撐開、`evt-trail`（可選）Mono Body |
| F11 底部 | `VVU55`（`[10,14]`、gap 8、只有上線、實測高 50） | `IgSaf` 膠囊（`$info-tint`、`$radius-sm`、`[6,8]`、gap 6）＋`kxXPK` radio 12 `$info`＋`GxPp2`「即時更新（SSE）」Label normal `$info-text`、`YOo9e` 撐開、`I17jS`「僅狀態事件，不含逐字內容」Label normal `$text-muted` |
| F12 紀錄欄 | `LTW74` | 欄頭 `Nf4qq`（`nNXwM`／`qA71L`／`hQyB4`／`phX01`）、清單 `IoJN4`：evt-1 `I9Dva6`、evt-2 `Nm2Lr`、evt-3 `BrmMZ`、evt-4 `CjZRX`（本次用量）、evt-5 `T0Mk6`（排入佇列）、evt-6 `PdlIv`（轉錄中 45%）、**evt-7 `eEbpz`**「已達預算上限 · 本批次 · $5.00」（`ShmdC` `circle-alert` `$warning-text`、階段字 `$warning-text`、尾端 Mono **`$text-primary`**） |
| F12 底部 | `pwfDs`（**垂直**、gap 8、實測高 76） | 第一行 `jFEQ9` 同 F11（`uuFxf`／`nb8eu`／`Z5Jhmz`／`R4cVNv`／`wDi9t`）；第二行 `v4Mp3s`（gap 6、靠右）：`sIE7E` 撐開＋`F0IlN` `circle-pause` **13×13** `$warning-text`＋`Oqlsx`「已停止（達預算上限）」Label **500** `$warning-text` |
| F13 | `F7ohe` | **沒有紀錄欄**（`modeShowsFeed` 對 idle 回 false，一致） |
| 註記 | `c4FIoB`→`J46QF`、`NtMLG`→`fblIx`、`c4FIoB`→`m9nBEI` | J46QF：「轉錄百分比只有翻譯階段有」；fblIx：「complete / cancelled / error 三個終態沿用此版面，只換 token 與文案；不另畫 frame」；m9nBEI：「右欄是事件日誌，非逐字稿」 |
| 手機 | `PXB0z` → `rMD73` | 收合區塊，沒畫事件列 → **歸 `dsr-6f`**，本張不碰 |

- **紀錄欄裡沒有任何元件實例**（全是一般 frame／text），全檔也沒有「事件列」母版 → 本張**不需要** Rule 21 的 `Implements:` 新標記。
- **`ctx.problems` 基線**：`l8FsB`／`iH98f`／`F7ohe`／`DUvwI`／`LTW74` **全部 0**。改完必須仍是 0。
- 字階（`DESIGN.md:390-394`）：Label 12／1.5、Body 14／1.625。Space：`xs-plus`=6、`sm`=8、`sm-plus`=10、`md-plus`=14。

---

## Acceptance Criteria

1. **設計稿先改（Pencil MCP；動手前每個節點再 `Get` 一次）。**
   - **F11 清單**（`dPQ9b`）改成系統真的會產生的樣子，並與左欄 `OkdGK`（轉錄中、無百分比）一致：
     - evt-1 `VxB7x`、evt-2 `jque4` 不動（只改圖示色，見下）。
     - evt-3 `a7UpAm`「提取音訊完成」→ **「已過的階段」列**：階段字改「**提取音訊**」`$text-secondary`、圖示改 **`check` `$text-muted`**（原 `audio-lines` `$accent-text`）。
     - **刪** evt-4 `u4njFq`（本次用量——左欄統計列已顯示同一個數字，而且它只在每部結束時才更新）與 evt-5 `E7Me1`（排入佇列——SSE 沒有這個事件）。
     - evt-6 `YtpPW`「轉錄中 · 奧本海默 · 45%」→ **拿掉尾端 `45%`**（`evt-trail` `enabled:false` 或刪）——轉錄沒有百分比（`J46QF`），而且左欄同一部已經停在轉錄中、沒有 %。其餘不動（仍是進行中：`loader-circle` `$accent-text`）。
   - **F12 清單**（`IoJN4`）：evt-3 `BrmMZ` 同 F11 處理；刪 evt-4 `CjZRX`、evt-5 `T0Mk6`；evt-6 `PdlIv`「轉錄中 · 奧本海默 · 45%」→ **「已過的階段」列**（批次已經停了，它不再是現在）：`check` `$text-muted`、階段字 `$text-secondary`、**拿掉 45%**。evt-7 `eEbpz` **不動**（已是正確的批次層提示：赭色標籤與圖示、金額 `$text-primary`）。
   - **基礎語意色當圖示色 → `-text`**（dsr-6d-c-1 對 `$warning` 的同一個先例）：`O35JV`／`YtcfP`／`f0yNSe`／`ZYy6B` 由 `$success` → **`$success-text`**；`kxXPK`／`nb8eu` 由 `$info` → **`$info-text`**。
   - **節點改名**（只改 `name`，不影響渲染）：`rDZQj`／`nNXwM`「Terminal Icon」→「Activity Icon」；`rAWiU`／`phX01`「Autoscroll Label」→「Scope Label」。
   - **新的獨立規格畫面 `F11-SPEC-LOG`「即時活動 · 事件種類」**（照 `feedback_pencil_spec_standalone_screen`：規格註記自己一張，不塞進既有畫面；照 `project_pen_flow_layout_convention` 放在 Flow F 區塊、標題在 frame 上方）：把 **AC #2 的詞彙表**逐列畫成真的事件列（用 F11 列的同一套結構與變數），每列旁邊一行小字寫「來源事件」。⚖️ 詞彙表是 SM 草案，Sally 在 review 可改；**改了就同步改程式碼與本 AC #2**，三方一致才算完成。
   - **註記 `fblIx`（`NtMLG`）補一句**：「右欄紀錄底部的第二行「已停止（達預算上限）」只在預算上限出現；其餘終態由紀錄最後一列（批次完成／批次已取消／批次發生錯誤）說明。」
   - 新文字節點一律綁變數；每張改完 `ctx.problems`（**基線 0，改完仍 0**，新規格畫面也要 0）；存檔後確認 ` M ux-design.pen` 並讀磁碟檔確認落盤（`feedback_verify_pen_saved_before_commit`）。
   - **`scripts/export-pen-screenshots.py` 的 `SCREENS` 補一筆**：新規格畫面的 node id → `("flow-f-subtitle-v2", "f11-spec-log")`（加之前先跑重複 key 檢查，見 `feedback_pen_inline_agent_workflow` 的指令）。匯出後**只 stage** `flow-f-subtitle-v2/{f11-d-v2,f12-d-v2,f11-spec-log}.png` ＋ `_bmad-output/pen-tokens.json`（若變），其餘重繪雜訊 `git checkout`。**若 `f11-m-v2`、`f13-d-v2`、`f8-d-v2` 或 `design-system/` 也變了，代表動錯了節點——回頭檢查。**

2. **紀錄的詞彙表（⚖️ SM 草案，Sally／Alexyu 可在 review 推翻；推翻就三方一起改）。**

   每一列＝「圖示 16 · 階段字（Body 600）· `·` · 項目（Body `--text-secondary`）· 撐開 · 尾端（Mono Body，可選）」。

   | # | 列 | 什麼時候出現（來源） | 圖示 | 階段字 | 項目 | 尾端 |
   | --- | --- | --- | --- | --- | --- | --- |
   | 1 | **階段・進行中** | 某一部進入一個新階段（見下「階段字對照」） | `loader-circle` `--accent-text`，`animate-spin motion-reduce:animate-none` | 階段字，`--accent-text` | 片名 | **只有翻譯中**才有 `N%`（`--accent-text`）；同一階段後續的事件**原地更新**這個數字，不新增列 |
   | 2 | **階段・已過** | 同一部進了下一階段、有了終態、批次終態、或連線重開（見 AC #3） | `check` `--text-muted`（靜止） | 同一個階段字，`--text-secondary` | 片名 | **無**（% 拿掉——它不再是現在的數字） |
   | 3 | **完成** | 批次成員：`changed_item.status = "done"`；單部：`transcription_complete` | `check` `--success-text` | 完成，`--success-text` | 片名 | — |
   | 4 | **失敗** | 批次成員：`changed_item.status = "failed"`；單部：`transcription_failed`（非預算） | `triangle-alert` `--error-text` | 失敗，`--error-text` | 片名 · 原因（見 AC #5） | — |
   | 5 | **已停止（單部預算）** | 單部 `transcription_failed` 的 `error` 含 `AI_BUDGET_EXCEEDED` | `circle-alert` `--warning-text` | 已停止，`--warning-text` | 片名 · 已達預算上限 | — |
   | 6 | **批次完成** | 批次終態 `complete` 且 `fail_count = 0` | `check` `--success-text` | 批次完成，`--success-text` | 本批次 | — |
   | 7 | **批次完成（有失敗）** | `complete` 且 `fail_count > 0` | `check` `--text-muted` | 批次完成，`--text-secondary` | 本批次 · 完成 N 部、失敗 M 部 | — |
   | 8 | **批次已取消** | `cancelled` | `circle-pause` `--text-muted` | 批次已取消，`--text-secondary` | 本批次 · 完成 N 部 | — |
   | 9 | **批次發生錯誤** | `error` | `circle-alert` `--error-text` | 批次發生錯誤，`--error-text` | 本批次 | — |
   | 10 | **已達預算上限** | `budget_ceiling` | `circle-alert` `--warning-text` | 已達預算上限，`--warning-text` | 本批次 | `usd(budget_usd)`，**`--text-primary`**（金錢是事實，DESIGN.md「金錢是事實」） |

   **階段字對照**（事件 → 階段字）：`transcription_extracting` → **提取音訊**；`transcription_progress` → **轉錄中**；`translation_progress` → **翻譯中**；`subtitle_progress` `extracting` → **抽取字幕**；`subtitle_progress` `translating` → **翻譯中**（**無 %**——D6 只給「第 N/M 段」的句子，不解析句子）；`subtitle_progress` `probing`／`placing`／`converting` → **不出列**（極短、且後面必接下一個階段）。
   - 片名一律走 `queueRowTitle({title, seriesTitle})`（`generationQueueRow.ts:150-153`），⛔ 不准再寫一份。拿不到片名 → 「**處理中的項目**」；⛔ **絕不顯示 UUID**（包含後端 `title` 恰好等於 `media_id` 的退回值）。
   - 7／8 的「完成 N 部、失敗 M 部」「完成 N 部」與左欄 `terminalVerdict`（`GenerationWorkspaceV2.tsx:404-429`）**逐字一致**，數字取終態事件的 `success_count`／`fail_count`。
   - 顏色全部是 `-text` 那一階或 `--text-*`；⛔ 不准出現 `text-[var(--success)]` 這類基礎色（lint 會擋）。
   - 「排入佇列」「本次用量」「提取音訊完成」**不產生**（見 🔴 #21；設計稿依 AC #1 刪除／改寫）。

3. **事件模型改寫（`useGenerationJobsFeed.ts`；🔴 #1–#7、#10–#11、#14–#17）。**
   - **批次成員**：hook 記一份「目前這個批次有哪些片」＋片名。來源兩個：①`generation_batch_progress` 的 `changed_item`（`status: "running"` 那一則在該部**開始處理之前**就送出——`generation_batch.go:651` 的 `markItem(running)` 早於 `:653` 的 `ExecuteGeneration`，所以它的 `transcription_*` 到的時候片名已經知道了）；②新增 **`seedBatch(batchId, items)`**，由容器在探測到**執行中**批次時呼叫（見 AC #4），讓「開到一半才打開頁面」也有成員與片名。批次終態列寫完後清空成員（之後同一部片若從詳情頁單獨生成，要算單部）。
   - **一部片的終態只有一個權威來源**（🔴 #3–#6）：
     - **批次成員**：終態列**只**來自 `changed_item`（`done` → #3、`failed` → #4）。它的 `transcription_complete`／`transcription_failed`、`subtitle_progress` `complete`／`failed`／`skipped` **一律不出終態列**（只把那一部的進行中列降成「已過」）。`paused`／`cancelled` 不出逐部列——批次終態那一列（#8／#10）已經說了。
     - **單部**（不是成員）：終態列來自 `transcription_complete`／`transcription_failed`。
   - **`subtitle_progress` 只收批次成員**（🔴 #2）：`media_id` 不在成員裡的一律丟掉（這同時擋掉搜尋引擎、手動下載、掃描後自動生成的事件）。成員的階段照 AC #2 對照表；終態照上一條忽略。⚠️ 不需要 `useGenerationProgress` 那個 `d6PipelineSeenRef`——成員過濾已經比它嚴格。
   - **原地更新，不洗版**（🔴 #7）：一部片目前的進行中列若與新事件是**同一個階段字**，只更新它的尾端（%）與內部狀態，**不新增列、`seq` 不變**。階段字變了 → 把舊的那列降成「已過」、新增一列。
   - **降級規則**（🔴 #15）：以下情況把列從「進行中」降成「已過」——同一部進入新階段；同一部有了終態（任一來源）；批次終態（該批所有成員的進行中列）；**連線重新打開**（重連前的進行中列全部降級——空窗內的事件拿不回來，不能再宣稱它們還在跑）。
   - **`connected`**（🔴 #14）：hook 回傳 `connected: boolean`。`onopen` → `true`；`onerror`、`stop()`、卸載 → `false`。⚠️ jsdom 的 `EventSource` stub 要補 `onopen`（`useGenerationJobsFeed.spec.ts:7-28` 目前沒有）。
   - **`singleJobs` 只放單部**（🔴 #16）：成員的 `transcription_*` **不寫進** `singleJobs`。`SingleJobState` 加 `title: string`（來自 AC #8 的後端欄位；空字串＝不知道）。
   - **拿掉 `items` 再轉換**（🔴 #17）：`generation_batch_progress` 在 `snakeToCamel` 之前把 `data.items` 刪掉（不要轉、不要讀）。要讀的欄位：`batch_id`、`status`、`changed_item`、`success_count`、`fail_count`、`budget_usd`。⛔ 不要動批次 hook 的轉換——那邊是真的在用 `items`。
   - **批次層列只寫一次**：以 `(batch_id, status)` 去重，取代現在的 `lastBatchStatus`（`:193-197`）；`running` 不出列。
   - **失敗原因要能事後補**：成員的 `transcription_failed` 雖然不出列，但要**記住**它的 `error`，等 `changed_item.status="failed"` 且 `reason="error"` 時，用 AC #5 對照出的中文取代籠統的「生成失敗」。
   - `FEED_CAP = 200` 保留（原地更新之後，200 列約是 50 部片）。
   - ⛔ **不讀時鐘**（Rule 23）：`seq` 仍是單調遞增整數，不准用 `Date.now()`。
   - **檔頭註解改成實話**：`:16-18` 說 `active_jobs` 沒有 transcription kind（`disc-2026-07-transcription-active-jobs` backlog）——那張 2026-08-24 已 done；`:8-10` 說「每個 `transcription_*` 事件都新增一列」——改完之後不是了。

4. **容器接線（`GenerationWorkspace`，`GenerationWorkspaceV2.tsx:832-992`）。**
   - 探測走 running 那條路時（`:899-905`，`startBatchTracking(probeData.progress)` 旁邊），同時 `jobs.seedBatch(progress.batchId, progress.items ?? [])`。⛔ 不要把它塞進別的 effect 或改那條 effect 的依賴——dsr-6d-c-1 CR H3 的「晚到的探測」守衛（`:903`）要一起守住它（被忽略的探測也不准 seed）。
   - 傳 `feedConnected={jobs.connected}` 給畫面。
   - **single 模式那一列的標題**（`:658`）：`title: job.message || '處理中的項目'` → **`title: job.title || '處理中的項目'`**。`job.message` 是階段句子（「正在轉錄音訊」），不是片名。這一行是本張對左欄唯一的改動，理由見 Context。

5. **失敗原因講中文（🔴 #13）。**
   - **新檔 `apps/web/src/components/subtitle/generationEventCopy.ts`**（純函式＋spec，Rule 21 檔頭 `// Implements: <utility — no .pen counterpart>`）：`failureCopy(error: string): { text: string; budget: boolean }`，子字串比對、**第一個命中的算數**：

     | `error` 含 | 句子 |
     | --- | --- |
     | `AI_BUDGET_EXCEEDED` | 已達預算上限（`budget: true` → 走 AC #2 的 #5 列，不是失敗） |
     | `no audio track found` | 找不到可用的音軌 |
     | `ffmpeg not available` | 伺服器缺少 ffmpeg |
     | `transcription unavailable` | 語音辨識未設定 |
     | `AI_UNAUTHORIZED` | API 金鑰無效 |
     | `AI_TIMEOUT`、`timed out`、`deadline exceeded` | 處理逾時 |
     | （其他，含空字串） | 生成失敗 |

   - **批次成員**的原因：`reason` 不是 `error` → 用 `queueRowLabel({status:'failed', reason}).text`（`generationQueueRow.ts:73-78`，「沒有可用的字幕來源」「這部正在別處處理」）；`reason === 'error'` → 先前記住的 `transcription_failed` 對照結果，沒有就「生成失敗」。
   - ⛔ **畫面上絕不出現後端原字串**（含 `title=` 屬性、`aria-label`）——`save SRT: <os err>` 會帶伺服器檔案路徑。原字串已在伺服器日誌（`transcription_service.go:1566-1570`）。

6. **右欄外觀（`EventLogPane`，`GenerationWorkspaceV2.tsx:310-363`；🔴 #8、#9、#18–#20）。**
   - **欄頭**（`h8wLFJ`）：`h-11`（44）`px-3.5` `gap-2.5`、底線；最前面加 lucide `Activity` `h-3.5 w-3.5 text-[var(--text-secondary)]` `aria-hidden`；「即時活動」`text-sm font-semibold`（已對）；「自開啟本頁起累積」`text-xs text-[var(--text-muted)]`（已對）。
   - **清單**（`dPQ9b`）：`py-1.5`，**不要** `px-1.5`、**不要** `space-y-0.5`。
   - **列**（`evt-N`）：`flex items-center gap-2 px-3.5 py-2`，**無圓角無底色**；圖示 `h-4 w-4 shrink-0`；階段字 `text-sm font-semibold`；`·` `text-sm text-[var(--text-muted)]` `aria-hidden`；項目 `min-w-0 flex-1 truncate text-sm text-[var(--text-secondary)]`；尾端 `ml-auto shrink-0 font-mono text-sm tabular-nums`。
   - **SSE 膠囊只在 `feedConnected` 時顯示**（`:356`）。`GenerationWorkspaceV2Props` 新增 `feedConnected?: boolean`，**預設 `false`**（不知道＝不宣稱連著；夾具與 spec 要明寫 `true`）。⛔ **不要**依批次是否終態來決定——那條連線在終態確實開著（🔴 #14）。⛔ 左邊統計列那顆（`:140`）不動。
   - **F12 底部第二行**（`v4Mp3s`）：只在 `mode === 'budget_ceiling'`；底部改成垂直兩行（`flex-col gap-2`），第二行 `flex items-center justify-end gap-1.5`：`CirclePause` `h-[13px] w-[13px] text-[var(--warning-text)]` `aria-hidden` ＋「已停止（達預算上限）」`text-xs font-medium text-[var(--warning-text)]`。其餘終態**不畫**第二行（`fblIx` 補記的裁定）。
   - ⛔ **11px 凍結**（`disc-2026-09-11px-micro-label-not-on-type-scale`）：`:149`（膠囊字）與 `:357`（「僅狀態事件，不含逐字內容」）的 `text-[11px]` **原樣保留**，即使稿上是 Label 12。
   - **播報**（🔴 #8）：`<ol>` **拿掉** `aria-live`（保留 `aria-label="生成事件日誌"`）。改由**一個一直掛著的** sr-only 區（`aria-live="polite"`，放在 `EventLogPane` 裡、不隨內容掛載）只唸 **終態列與批次層列**（AC #2 的 #3–#10），內容＝那一列的「階段字 片名／本批次 原因」一句。**進行中／已過的階段列、% 更新一律不唸**。⚠️ dsr-6d-b CR M4 的教訓：與內容一起掛載的 live region 報讀不唸——要確定這個 sr-only 區在 `running → complete` 模式切換時**不會被重新掛載**（`{showFeed && <EventLogPane/>}` 同一位置，應該不會；寫測試釘住）。⛔ 全頁**不准再多**任何 live region。
   - **自動捲到底**（🔴 #9）：新列加入時，若使用者**原本就在底部**（距底 ≤ 一列高），把清單捲到底；使用者往上捲了就不打擾。只改 `scrollTop`，⛔ 不要用 `scrollIntoView`（會捲動整頁）。⛔ 不要 smooth（不是「正在發生的事」，Motion 規則）。
   - `FEED_TONE_CLASS`（`:71-76`）依新的列型別改寫或換成 AC #2 的表驅動；⛔ 不准出現基礎語意色當文字。

7. **既有的誠實行為不准回歸（測試都要留著）。**
   - 右欄**只在** `modeShowsFeed` 的模式出現（`GenerationWorkspaceV2.spec.tsx:85`）；idle／loading 沒有右欄。
   - 「僅狀態事件，不含逐字內容」與「自開啟本頁起累積」逐字保留；**沒有時間戳**。
   - §8 lazy-SSE：掛載時**不連線**，`startTracking()` 才連（`useGenerationJobsFeed.spec.ts:41-51`）；`stop()` 關閉。
   - 超過 `FEED_CAP` 丟最舊、`seq` 唯一且單調（`:151-164`）。
   - dsr-6d-c-1 的左欄行為全部不動：`GenerationWorkspaceContainer.spec.tsx` 現有的 15 條＋CR 補的回歸測試**一條都不准改斷言**（只准補 mock 欄位）。

8. **後端：`transcription_*` 事件帶片名（唯一的後端 task；🔴 #12；根治 `disc-2026-09-single-job-title-missing`）。**
   - 每一個 `transcription_*` 事件（`transcription_extracting` `:607-612`、`transcription_progress` `:642-647`、`translation_progress` `:886-892` 與 `:1322-1329`、`transcription_complete` `:695-725`、`failJob` `:1571-1577`）的 payload **加一個鍵 `title`**（字串，**永遠存在**）：
     - **solo 任務**（詳情頁觸發，`acquireJob(…, solo=true)`）＝已經解好的 `soloTranscriptionJob.Title`；
     - **但** `Title == mediaID`（`resolveActivityTitle` 的退回值）→ 送 **`""`**。⛔ 絕不把 UUID 當片名送出去。
     - **批次／pipeline 路徑**（`solo=false`，本來就不查片名，`:437-441`）→ **`""`**。⛔ 不准為了這個欄位讓批次路徑開始查 DB——批次成員的片名前端已經從 `changed_item` 拿到。
   - 取值方式由 dev 決定（在 `runPipeline` 內把 title 一路帶下去，或讀 `inProgress[mediaID].Title`），⚠️ 但要確認**每一個**送出點當下那個 entry 還在（`runPipeline` 的 deferred cleanup 會釋放；`transcription_complete`／`failJob` 若晚於釋放就拿不到）。
   - **Rule 20**：`transcription_*` 目前是 `[@contract-v2]`（`transcription_service.go:88-95`，9R-18 首次蓋章、sub-3-2 升 v2）。本張是**純加寬**（多一個可忽略的鍵，既有欄位語意不變）→ **不升版**，比照 sub-5-1 AC #7 對 9R-16 AC #3、sub-6-8a 對 sub-4-2 的加寬先例。在 Change Log 記一行，並**更正那段常數註解的消費者清單**——它寫「兩個消費者都用自己的 media id 嚴格過濾」，但 ux3-ai-2 的 `useGenerationJobsFeed` 是**不過濾**的第三個消費者。
   - 測試（`transcription_service_test.go` 或既有的事件測試檔）：①solo 任務的每一種事件都帶解好的 `title`；②解不到（退回 mediaID）時 `title` 是 `""`；③批次路徑 `title` 是 `""` 而且**沒有**多查任何 reader（用會在被呼叫時 fail 的 fake 釘住）。

9. **測試（先寫紅測試）。**
   - **`useGenerationJobsFeed.spec.ts`**：
     - **改寫**：`:61-84`（同一階段第二個事件原地更新、不新增列）、`:110-132`（顯示中文、不顯示原字串）、`:134-149`（`已達預算上限`＋`本批次`＋尾端金額，同一個 `(batch_id,status)` 只一列）。**保留不動**：`:41-59`（lazy／stop）、`:151-164`（cap／seq）。
     - **新增**（每一條對應上面的 🔴）：成員的 `subtitle_progress` 出列、非成員的（例：英文 `searching`）**不出列**（#1、#2）；預算：成員先收到 `transcription_failed`（`AI_BUDGET_EXCEEDED`）再收到批次 `budget_ceiling` → **沒有**「失敗」列、有「已達預算上限」列、那一部的進行中列已降級（#3、#15）；取消同理（#4）；pipeline 退回路線同一部兩個家族的終態只出一列（#5）；`busy_elsewhere` 只有 `changed_item` 也出「失敗 · 這部正在別處處理」（#6）；150 個 `translation_progress` 只有一列（#7）；片名來自 `changed_item`、`seedBatch`、單部的 `title`，`title` 等於 `media_id` 時顯示「處理中的項目」（#10、#12）；批次層列的項目是「本批次」不是片名（#11）；`connected` 隨 `onopen`／`onerror` 變化、重開後舊的進行中列全部降級（#14）；成員不進 `singleJobs`（#16）；終態事件的 `items` 沒被轉換（用 spy 包 `snakeToCamel` 或斷言轉換的輸入不含 `items`）（#17）；`reason="error"` 時採用先前記住的中文原因。
   - **`generationEventCopy.spec.ts`**（新）：表中每一列一條＋未知字串回「生成失敗」＋回傳值不含任何英文原字串。
   - **`GenerationWorkspaceV2.spec.tsx`**：`:48-51` 的 `feed` 夾具換成新的列型別；`:182-194` 改寫（清單**沒有** `aria-live`、sr-only 區有、只唸終態列）；新增：欄頭有 `activity` 圖示、每列有 `·`、`feedConnected=false` 時右欄**沒有**膠囊、`budget_ceiling` 有第二行而 `complete` 沒有、`running → complete` 重新 render 時 sr-only 區是**同一個 DOM 節點**。`:327-338` 的膠囊測試依 `feedConnected` 改寫（統計列那顆的斷言不動）。
   - **`GenerationWorkspaceContainer.spec.tsx`**：jobs mock 工廠（`:69-76`）補 **`connected`、`seedBatch`**（🚨 dsr-6d-b AC #8 的教訓：漏一個就整支紅）；新增：探測 running 時呼叫 `seedBatch(batchId, items)`、被 CR H3 守衛忽略的晚到探測**不** seed、single 列標題用 `job.title`。
   - **e2e**（`tests/e2e/generation-workspace.spec.ts` 加一條，沿用 `@ui @generation-workspace`）：用 `page.route` 對 `**/api/v1/events` 回 `content-type: text/event-stream` 的一次性 body（frame 格式＝`event: <type>\ndata: {"type":"<type>","data":{…}}\n\n`，見 `sse/handler.go:63-70`），依序送：批次 `changed_item`(running, 奧本海默) → `transcription_extracting` → `translation_progress` ×3（10／20／30）→ 一則**非成員**的英文 `subtitle_progress` `searching`。斷言：只有一列「翻譯中」而且是 `30%`、看得到「奧本海默」、頁面上**沒有**英文 "Searching"。⚠️ **repo 沒有這種 mock 的先例**——先確認 Chromium 會把 `EventSource` 請求交給 `page.route`；body 結束後 EventSource 會觸發 `onerror` 並在 10 秒後重連，**第二次以後的請求回空 body**，否則重播會讓列重複。若 `page.route` 攔不到 EventSource，把這條改成記錄在 Completion Notes 並靠 hook／畫面層測試，⛔ 不要硬寫一條會 flaky 的測試。⛔ 不要用 `--grep @story-*`（那種標籤不存在）。
   - **後端**：見 AC #8。
   - **對抗式 mutation check**：把下面每一條修法拿掉，至少一條測試要變紅，並在 Completion Notes 列出——成員過濾、終態單一權威、原地更新、降級、`connected`、`items` 剝除、`title === media_id` 守衛、中文對照、sr-only 只唸終態。

10. **視覺夾具與基準線。**
    - 三個夾具（`-gallery.fixtures.tsx:5092`／`:5139`／`:5172`）的 `feed` **改成 hook 真的會產生的列**（🔴 #23），並加 `feedConnected: true`：
      - `running`：完成 · 沙丘：第二部 ／ 提取音訊（已過）· 奧本海默 ／ 轉錄中（進行中、無 %）· 奧本海默——與左欄 `activeItemProgress.phase: 'transcribing'` 一致。
      - `budget_ceiling`：完成 · 沙丘：第二部 ／ 提取音訊（已過）· 奧本海默 ／ 轉錄中（已過）· 奧本海默 ／ 已達預算上限 · 本批次 · $5.00。底部要看得到第二行「已停止（達預算上限）」。
      - `complete-with-failures`：完成 · 沙丘：第二部 ／ 失敗 · 奧本海默 · 這部正在別處處理 ／ 失敗 · 怪奇物語 S04E07 第七章 · 沒有可用的字幕來源 ／ 批次完成（有失敗）· 本批次 · 完成 1 部、失敗 2 部——**與左欄三列的 reason 一致**（`wsFxItem(1,'failed','busy_elsewhere')`、`wsFxItem(2,'failed','skipped')`）；第三部是分集，順便驗劇名前綴。
      - `idle` 沒有右欄，**不動**。
    - ⚠️ 夾具的 `props` 是 `Record<string, unknown>`，typecheck 抓不到漏欄位——**改完一定要看圖**確認每一列都有圖示、`·`、片名。
    - 基準線流程照 `project_visual_baseline_intentional_change.md`：`--update-snapshots` 後只留上列三張（`generation-workspace-v2/{running,budget_ceiling,complete-with-failures}`）、其餘還原（⚠️ `parse-floating-parse-progress-card` 與 `retry-retry-notifications` 在本機一定會漂，乾淨工作樹同樣漂，還原即可）；改過的 `-linux` 一律 `git rm` 交給 CI bootstrap。⛔ 不要本機產 `-linux.png`。
    - 夾具底部是否被 1280×800 視窗裁掉：`budget_ceiling` 的底部第二行要在圖裡（見 `project_visual_fixture_viewport_ceiling.md`）。

11. **另立／交代的單子（建單時已寫入 sprint-status，不在本張做）。**
    - **`disc-2026-09-transcription-run-5min-hard-timeout`**（**新立，P2 待實測**）：`TranscriptionService.timeout` 寫死 **5 分鐘**（`transcription_service.go:177`，沒有 setter），而且套在**整個 run**（提取＋語音辨識＋翻譯）上——async 路徑 `:367`、同步／批次路徑 `:401`（`context.WithTimeout`）。一部兩小時電影的分段語音辨識加上約 150 次翻譯請求，很可能超過 5 分鐘 → `context deadline exceeded` → 失敗。**實際頻率未驗證**（disc-2026-07-transcription-active-jobs 在 NAS 上實測一集約 200 秒）。本張的中文對照會把它顯示成「處理逾時」——如果這個對照在實際使用中變成最常見的一句，就是這張單子。
    - **`disc-2026-09-pipeline-asr-fallback-extracting-label`**（**新立，P3・後端**）：pipeline 模式走語音辨識退回路線時送 `subtitle_progress` `extracting`（`process_item.go:529`），而 `zhTWStageMessage`（`progress_sse.go:85-90`）把它翻成「**抽取內嵌字幕中…**」——這一部根本沒有內嵌字幕。本張的紀錄用中性的「抽取字幕」規避了一半，但 D6 的訊息本身與 `useGenerationProgress` 的消費端仍會說錯。修法要碰 D6 的 stamped stage set（`[@contract-v1]` sub-1-3 AC #1）或只改訊息組字，另案評估。
    - **`disc-2026-09-workspace-single-job-lost-terminal`**（**新立，P3・生成工作區**）：單部任務的終態事件若被 hub 丟掉（`sse/hub.go:121-128, 164-175`，沒有 replay），它會永遠留在 `singleJobs`，工作區永遠顯示「進行中任務」。dsr-6d-c-1 CR M6 的**批次那一半**由本張 AC #3（成員不進 `singleJobs`）根治；**單部那一半**需要一個對帳來源（例如 `/api/v1/activity` 的 `transcription` 任務數，但要處理「活動資料比 SSE 舊」的競態），另案。
    - **`disc-2026-09-batch-sse-items-transform-cost`**（既有）：工作區那一半由本張 AC #3 完成 → 本張 done 時一併改 done。
    - **`disc-2026-09-single-job-title-missing`**（既有）：由本張 AC #4＋AC #8 根治 → 本張 done 時一併改 done。
    - 既有、不在本張：`disc-2026-09-batch-dialog-last-attach-per-mount`（對話框／工作區順序問題）、`disc-2026-09-11px-micro-label-not-on-type-scale`（11px 凍結）、手機 F11-M `rMD73`（`dsr-6f`）。

12. **CI 全綠**：`pnpm run lint:all`、`pnpm nx run web:typecheck --skip-nx-cache`、`python3 scripts/check-design-tokens.py`、`pnpm nx test web`、`pnpm nx test api`、`pnpm run format:check`。⛔ 絕不用 `run_in_background` 跑測試；每次跑完 `pnpm run test:cleanup`。Nx daemon 建圖逾時就用 `NX_DAEMON=false` 重跑並記在 Completion Notes。⚠️ 本機跑 e2e 需要 `AI_PROVIDER=claude`（`.env` 預設 gemini 但沒金鑰，後端起不來）；firefox／webkit 本機沒裝瀏覽器會紅，CI 上是綠的（dsr-6d-c-1 已證實）。

## Tasks / Subtasks

- [x] **Task 1 — 設計稿（AC: #1, #2）**
  - [x] F11／F12 清單改寫（刪 evt-4／evt-5、evt-3 改「已過」、F11 evt-6 拿掉 45%、F12 evt-6 改「已過」）
  - [x] 圖示色 `$success`／`$info` → `-text`；兩組節點改名；`fblIx` 補一句
  - [x] 新規格畫面 `F11-SPEC-LOG`（AC #2 詞彙表 10 列＋來源小字）；`SCREENS` 補一筆
  - [x] `ctx.problems` 基線 0 → 改完仍 0；存檔並確認落盤；只 stage f11／f12／f11-spec-log（＋pen-tokens）
- [x] **Task 2 — 後端 `title`（AC: #8）** ← 唯一的後端 task
  - [x] 先寫紅測試：solo 帶片名、退回 mediaID 時送 `""`、批次路徑 `""` 且不查 reader
  - [x] 六個送出點加 `title`；更正 `[@contract-v2]` 常數註解的消費者清單
- [x] **Task 3 — 中文失敗原因（AC: #5）**
  - [x] 先寫紅測試：`generationEventCopy.spec.ts`
  - [x] `generationEventCopy.ts`（Rule 21 utility 檔頭）
- [x] **Task 4 — 事件模型（AC: #3）**
  - [x] 先寫紅測試：AC #9 的 hook 那一串（成員過濾、終態單一權威、原地更新、降級、`connected`、`items` 剝除、片名）
  - [x] 改寫 reducer／listener；`seedBatch`、`connected`、`SingleJobState.title`；檔頭註解改實話
- [x] **Task 5 — 容器接線（AC: #4）**
  - [x] 先寫紅測試：`seedBatch` 呼叫時機（含 H3 守衛不 seed）、single 標題用 `job.title`
  - [x] 接 `seedBatch`／`feedConnected`；改 `:658`
- [x] **Task 6 — 右欄外觀與播報（AC: #2, #6, #7）**
  - [x] 先寫紅測試：清單無 `aria-live`、sr-only 區只唸終態且不重掛、膠囊跟 `connected`、F12 第二行、欄頭圖示、`·`
  - [x] 列結構＋十種列型、欄頭、清單間距、F12 第二行、自動捲到底
- [x] **Task 7 — 夾具、基準線、e2e（AC: #9 後半, #10）**
- [x] **Task 8 — 收尾（AC: #11, #12）**
  - [x] 全套閘門；mutation check；dev-story Step 9 截圖比對（`f11-d-v2`、`f12-d-v2`、`f11-spec-log`）
  - [x] sprint-status：兩張既有 disc 改 done

## Dev Notes

### 這張的重點

- **最有感的是 🔴 #7＋#10**：現在的紀錄是一整排「翻譯中 12%、翻譯中 13%……」，而且**沒有一列說得出是哪一部**。改完之後一部片最多三、四列，每一列都有片名。
- **最容易做錯的是 🔴 #3**：預算用完時後端**先**送單部失敗、**後**送批次暫停。只要紀錄還相信 `transcription_failed`，左右兩欄就會對同一部講相反的話。**批次成員的終態只看 `changed_item`**——這是整張單子的中心規則。
- **`subtitle_progress` 是陷阱**（🔴 #2）：同一個事件名有三個發送者，兩個是英文、不是生成。不做成員過濾就接上它，會把紀錄從「漏東西」變成「塞垃圾」。
- **後端只加一個欄位**，而且那個值後端早就算好了（`resolveActivityTitle`），只是沒送出來。

### 上游契約（Rule 20 ack）

- confirmed against [@contract-v2] (Story 9R-18 AC #3) —— `transcription_*` 的 `media_id` 是媒體列 UUID 字串（電影或分集，sub-3-2 升 v2）；本張**加寬**一個 `title` 鍵、不升版（AC #8）。
- confirmed against [@contract-v1] (Story dsr-6d-a AC #1) —— `items[]`／`changed_item` 每筆 `{media_id,title,media_type,series_title,status,reason}`，六個鍵恆存在；`reason` 只在 `failed` 有值（`skipped`／`busy_elsewhere`／`error`）。
- confirmed against [@contract-v1] (Story dsr-6d-a AC #2) —— `generation_batch_progress` 每次 13 個鍵：執行中 `items: null` ＋ `changed_item`；終態 `items` 完整、`changed_item: null`；開始時**不送**事件；`paused`／`cancelled` 只出現在終態。
- confirmed against [@contract-v1] (Story sub-1-3 AC #1) —— D6 `PipelineStage` 12 個值，**唯讀消費**；本張只用 `extracting`／`translating`，其餘不出列。`subtitle_progress` payload `{media_id, media_type, stage, message}`（全部字串，無百分比、無片名、無 job／batch id）。

### 建單裁定（2026-09-18，Sally／Alexyu 可在 review 推翻）

1. **批次成員的終態只看 `changed_item`**，單部任務才看 `transcription_complete`／`failed`。一部片一個終態權威。
2. **`subtitle_progress` 只收批次成員**，其餘丟掉。
3. **同一部片同一階段只佔一列**，百分比原地更新；階段換了舊列降成「已過」（靜止、中性）。
4. **設計稿刪掉「排入佇列」「本次用量」**：前者系統沒有事件，後者與左欄統計列重複且只在每部結束時才更新。「提取音訊完成」改成「已過」的「提取音訊」列。
5. **失敗原因一律中文，原字串不上畫面**（含屬性）；對照表是 SM 草案。
6. **播報只唸終態與批次層**，階段與百分比不唸。
7. **F12 底部第二行只在預算上限**；其他終態由紀錄最後一列說明。
8. **後端 `title` 加寬不升版**，而且批次路徑不准為它多查 DB。

### 不要做的事

- **不要碰左欄**（`QueueRow`、`OverallStrip`、`BudgetBanner`、底部列、探測 effect 的條件與依賴），唯一例外是 AC #4 的 `:658` 與在既有 running 分支裡多呼叫一次 `seedBatch`。
- **不要合併兩條 EventSource**（批次 hook 與 jobs hook 各一條）——那是架構調整，不是本張；本張只讓 jobs hook 不再白轉 `items`。
- **不要動** `useGenerationProgress.ts`、`useGenerationBatchProgress.ts`、`GenerationBatchDialogV2.tsx`、`GenerationProgressV2.tsx`、`generationQueueRow.ts` 的既有匯出（只准 import）。
- **不要改 D6 的 stage set 或 `zhTWStageMessage`**（另立 `disc-2026-09-pipeline-asr-fallback-extracting-label`）。
- **不要動 5 分鐘上限**（另立 `disc-2026-09-transcription-run-5min-hard-timeout`）。
- **不要把 `text-[11px]` 改成 12**、**不要做手機版**（`rMD73` 歸 `dsr-6f`）、**不要在本機產 `-linux.png`**。

### 已知陷阱

- **兩個 hook 兩條 EventSource**（`GenerationWorkspaceV2.tsx:837-838`），對話框開著再兩條。改連線時序時要意識到這件事；本張的 `connected` 只描述 **jobs hook 那一條**。
- **`EventSource` 的 jsdom stub 沒有 `onopen`**（`useGenerationJobsFeed.spec.ts:7-28`）——補一個 `open()` 方法觸發它。
- **`changed_item` 是 flatten 的**（`generation_batch.go:121-137`），`snakeToCamel` 之後是 `changedItem: {mediaId, title, mediaType, seriesTitle, status, reason}`。
- **`duration` 是字串**（Go duration，例如 `"3m12s"`，`transcription_service.go:695-725`），`useGenerationProgress.ts` 的型別寫成 number——本張不讀它，⛔ 也不要順手改那邊的型別（不在範圍）。
- **resume 的翻譯續跑會跳過 `transcription_extracting` 與 `transcription_progress`**（`transcription_service.go:594-596`）——第一列可能直接是「翻譯中」，正常。
- **「轉錄完成」也可能是翻譯失敗（非預算）時送的**（`:915-925`）：只有 `zh_srt_path`／`partial` 分得出來。本張的「完成」列**不宣稱繁中**（與 `queueRowSubStatus` 的 `done` 同一個低宣稱原則），⛔ 不要寫「已生成繁中字幕」。
- **重連之後同一部同一階段會再出一列**（舊的已依 AC #3 降成「已過」，下一個事件又開一列「進行中」）。這是**正確的**——它如實標出中間有一段收不到事件的空窗；⛔ 不要為了好看把兩列合併。
- **`translation_progress` 的第一個 frame 是 0%**（`:886-892`）——「翻譯中 0%」是真話，不要濾掉。
- **`toHaveTextContent` 是子字串比對**——「完成」會命中「批次完成」「完成 1 部」。文案斷言要用精確選擇器。
- **視覺 CI 沒有後端**：夾具全靠 props，缺欄位 typecheck 不會抓。
- **行號以建單時為準**（2026-09-18，main `ff16deef`）。

### Source tree

```
ux-design.pen（F11 `DUvwI`／F12 `LTW74` 清單與圖示色、`fblIx` 註記、新畫面 F11-SPEC-LOG）  ← Task 1
scripts/export-pen-screenshots.py（SCREENS +1）                                      ← Task 1
apps/api/internal/services/transcription_service.go(+test)                          ← Task 2
apps/web/src/components/subtitle/generationEventCopy.ts(+spec)（新）                  ← Task 3
apps/web/src/hooks/useGenerationJobsFeed.ts(+spec)                                  ← Task 4
apps/web/src/components/subtitle/GenerationWorkspaceV2.tsx(+spec)                   ← Task 5–6
apps/web/src/components/subtitle/GenerationWorkspaceContainer.spec.tsx              ← Task 5
apps/web/src/routes/test/-gallery.fixtures.tsx                                      ← Task 7
tests/e2e/generation-workspace.spec.ts                                              ← Task 7
tests/visual/components.visual.spec.ts-snapshots/components/generation-workspace-v2/{running,budget_ceiling,complete-with-failures}/** ← Task 7
_bmad-output/pen-tokens.json、_bmad-output/screenshots/flow-f-subtitle-v2/{f11-d-v2,f12-d-v2,f11-spec-log}.png ← Task 1
apps/web/src/components/subtitle/generationQueueRow.ts                              ← 只讀（import queueRowTitle／queueRowLabel）
```

### Cross-Stack Split Check

後端 task **1 個**（Task 2），前端／設計 task 7 個。→ 跨棧門檻（兩邊都 >3）**不觸發**。規模與 `dsr-6d-c-1`（8 個 task）相當，不再拆（`feedback_split_oversized_stories`：沒有超過上一張同類單子）。

### Time-dependent visual coverage

- **N/A — no wall-clock-reading components touched.** `EventLogPane`、`useGenerationJobsFeed.ts`、`generationEventCopy.ts` 都不讀時鐘；紀錄只靠 `seq` 排序（Rule 23-clean，ux3-ai-2 起就是這樣）。本張明文禁止加時間戳或 `Date.now()`（AC #3）。

### References

- [Source: `apps/web/src/hooks/useGenerationJobsFeed.ts:1-24, 32, 37-56, 82-110, 127-155, 173-208, 234-241, 252-282, 298-300`]
- [Source: `apps/web/src/components/subtitle/GenerationWorkspaceV2.tsx:71-76, 140, 145-155, 310-363, 404-429, 467, 645-680, 683, 832-992`]
- [Source: `apps/web/src/components/subtitle/generationQueueRow.ts:48-54, 69-92, 150-153`、`generationWorkspace.ts:36-56`]
- [Source: `apps/web/src/hooks/useGenerationProgress.ts:1-33, 166-205, 277-310`（D6 雙家族先例、CR M7）]
- [Source: `apps/web/src/components/subtitle/GenerationWorkspaceV2.spec.tsx:48-51, 85, 182-194, 327-338`、`GenerationWorkspaceContainer.spec.tsx:20-90`、`useGenerationJobsFeed.spec.ts:7-165`]
- [Source: `apps/web/src/routes/test/-gallery.fixtures.tsx:274-284, 5092-5205`、`tests/e2e/generation-workspace.spec.ts`]
- [Source: `apps/api/internal/services/transcription_service.go:88-102, 177, 367, 401, 431-486, 594-596, 607-612, 642-647, 669, 684-725, 886-892, 915-925, 1322-1329, 1565-1578`]
- [Source: `apps/api/internal/services/generation_batch.go:121-137, 651-694, 823, 845-859`]
- [Source: `apps/api/internal/subtitle/progress_sse.go:60-124`、`process_item.go:71-72, 529, 571, 597, 746`、`engine.go:390-404`、`handlers/subtitle_handler.go:236-301`]
- [Source: `apps/api/internal/sse/hub.go:87, 121-128, 151, 164-175`、`sse/handler.go:45, 63-70`、`config/config.go:217`]
- [Source: `ux-design.pen` `l8FsB`／`iH98f`／`F7ohe`／`DUvwI`／`LTW74`／`J46QF`／`fblIx`／`m9nBEI` — 唯讀稽核代理以 Pencil MCP 逐節點讀出（2026-09-18）]
- [Source: `DESIGN.md` 固定詞彙規則、金錢是事實、Motion「動的東西＝正在發生的事」、字階 `:390-394`]
- [Source: `_bmad-output/implementation-artifacts/dsr-6d-c-1-workspace-queue.md`（CR M6 交代、CR H3 晚到探測守衛）、`dsr-6d-b-batch-dialog.md`（CR M4 live region 教訓、AC #8 mock 工廠教訓）、`dsr-6d-a-batch-status-backend.md`]
- [Source: project-context.md#Rule 13 / #Rule 16 / #Rule 18 / #Rule 20 / #Rule 21 / #Rule 23 / #Rule 24]

## Dev Agent Record

### Agent Model Used

Claude Opus 5 (1M context) — dev-story (Amelia)，2026-09-18，branch `feat/dsr-6d-c-2-workspace-event-log`。

### Debug Log References

- Pencil：`osascript` 點 File ▸ Save 之後 Pencil 把文件關回 Dashboard（MCP 回 `Failed to access file`），用 `open -a Pen ux-design.pen` 重開即恢復；存檔內容已落盤（grep `F11-SPEC-LOG` 命中），重開後讀回的節點值與存檔前一致。
- `pnpm nx test web -- <file>` 不會只跑單檔（`{args._}` 位置參數），單檔迭代改在 `apps/web` 直接 `npx vitest run <file>` 並每次跑 `scripts/cleanup-test-processes.sh --all`；每個 task 結束仍以 `pnpm nx test web` 跑全套。
- e2e：Playwright 的 `route.fulfill` 對 `EventSource` **有效**（Chromium），但 fulfill 的 body 一結束串流就斷 → hook 走 `onerror` → 容器的 effect 重跑 `startTracking()` → 第二條連線又拿到同一批 frame（真實 SSE 不會結束、也不會重播）。改成「只有第一條連線給 frame，其後一律 `route.abort()`」。
- lint：`lint:all` 的總數在兩次執行間是 127／129 浮動；以 `eslint . --format json` 對 stash 前後逐條比對，**兩邊完全相同（135 條，0 條新增）**，浮動與本張無關。

### Completion Notes List

- Ultimate context engine analysis completed - comprehensive developer guide created（SM Bob，2026-09-18；兩個唯讀稽核代理：設計稿節點、後端 SSE 事件；程式碼由 SM 自讀，後端關鍵說法逐行複核）。
- 🔗 **AC Drift: FOUND** —— 本張改變了 `ux3-ai-2-workspace-frontend.md` 三條 AC 的可觀察行為：**AC #4**（「every `transcription_*` event appends a feed row」）→ 同一部同一階段原地更新、批次成員的終態只看 `changed_item`、新增 `subtitle_progress`（只收成員）；**AC #5**（「listens … UNFILTERED, maintains a per-media map of in-flight single jobs」）→ 批次成員不再進 `singleJobs`；**AC #7**（「status transitions aria-live=polite」）→ 清單不再是 live region，改由一個 sr-only 區只唸結果與批次列。grep：`useGenerationJobsFeed|workspace-feed-row|EventLogPane|即時活動` 掃 `_bmad-output/implementation-artifacts/*.md` 命中 5 個檔，`dsr-6d-c-1`／`ux3-ai-1-workspace-design-prompt`／`disc-2026-07-transcription-active-jobs` 都是 REUSE。`ux3-ai-2` 的 AC 未 stamp（implicit v0），無 ack 義務，漂移依 retro-10-AI2 記錄於此。
- 📎 **Contract Stamps: FOUND（本張 0 個自有 stamp、4 條上游 ack，版本全部對得上）** —— `[@contract-v2]` 9R-18 AC #3（sub-3-2 升 v2，Change Log 在 `sub-3-2-episode-asr-fallback.md:176`）；`[@contract-v1]` dsr-6d-a AC #1、#2（AC #10 宣告 #1–#7 皆 v1）；`[@contract-v1]` sub-1-3 AC #1（D6 12 值）。本張對 `transcription_*` **加寬**一個 `title` 鍵（永遠存在、既有欄位語意不變）→ 依 sub-5-1／sub-6-8a 的加寬先例**不升版**，常數註解已更正並寫明（`transcription_service.go:88-95`）；無 bump → 無下游 stale-mark 義務。
- **Task 1（設計稿）**：F11／F12 紀錄刪「本次用量」`u4njFq`／`CjZRX` 與「排入佇列」`E7Me1`／`T0Mk6`；「提取音訊完成」`a7UpAm`／`BrmMZ` 改成已過的「提取音訊」（`check` `$text-muted`、`$text-secondary`）；F11 `YtpPW` 拿掉 45%（`ldYQz`），F12 `PdlIv` 改已過並拿掉 45%（`jiXW1`）；四顆勾 `$success`→`$success-text`、兩顆 radio `$info`→`$info-text`；兩組節點改名；`fblIx` 補一句；**新規格畫面 `F11-SPEC-LOG` `tT3JX`**（十種列＋來源、不出列的、播報規則），`SCREENS` 補 `tT3JX → f11-spec-log`（重複 key 檢查：只有 `name` 這個誤報）。`ctx.problems`：`l8FsB`／`iH98f`／`F7ohe`／`tT3JX` 各 0；全檔 70 → 68（只減不增）。**以 `.pen` 的 JSON 差異逐一核對：146 個變動節點全部落在兩個紀錄欄、`NtMLG` 與新規格畫面內，沒有碰任何母版**；所以匯出後 `design-system/`、`f1–f5`、`b12p`、`flow-i`、`j9` 的差異全是重繪雜訊，已還原。
  - ⚠️ **對 AC #2 草案的一處補充**：批次層列的播報句（例：「已達預算上限，批次已停止」）AC 只寫了「那一列的一句」，實作在 `feedRowView` 各自定義，規格畫面有寫播報範圍。
- **Task 2（後端）**：`eventTitle(mediaID)` 讀 `inProgress`（map、在鎖內，不查 DB），六個送出點＋`failJob` 都帶 `title`；solo＝解好的片名，退回值（`== mediaID`）、批次／pipeline 路徑、不在執行中 → `""`。新測試 5 條（`transcription_event_title_test.go`），mutation：拿掉 UUID 守衛 → 2 條紅。
- **Task 3–6（前端）**：
  - ⚖️ **設計決策**：hook 只記「發生了什麼」（`FeedRow` 是有型別的聯集：`stage`／`done`／`failed`／`batch`，原始 `reason`／`error` 照存），文字與顏色集中在新檔 `generationEventCopy.ts` 的 `feedRowView()`（十種列一個函式、表驅動）——hook 因此不必 import 任何元件模組，而對話框與工作區共用的 `queueRowTitle`／`queueRowLabel`／`GENERATION_STAGES` 只在 view 層用一次。
  - `seedBatch(batchId, items)` 照 AC 的簽名，但 reducer 不需要 `batchId`（成員在批次終態清空，靠 `(batch_id,status)` 去重的是批次列）；參數保留給呼叫端語意，已在 JSDoc 註明。
  - 自動捲到底用 `useLayoutEffect` 依最新 `seq` 觸發、`onScroll` 記錄是否停在底部（距底 ≤ 40px）；只改 `scrollTop`，沒有 smooth。
  - 既有測試的改動：`GenerationWorkspaceV2.spec.tsx` 的 attach 測試把 `getByText('奧本海默')` **限縮到 attach 面板**（紀錄現在也會寫同一部片名，查詢變成多筆）——斷言語意不變；single 模式兩個夾具補 `title`。`GenerationWorkspaceContainer.spec.tsx` 原有 17 條**一條斷言都沒改**，只在 mock 工廠補 `connected`／`seedBatch`。
- **對抗式 mutation check（11 項修法逐一拿掉，全部有牙）**：成員過濾 → 1 紅；終態單一權威 → 5 紅；原地更新 → 2 紅（＋e2e 紅）；降級 → 6 紅；`connected` → 2 紅；`items` 剝除 → 1 紅；UUID 守衛（前端）→ 1 紅；中文對照 → 1 紅；sr-only 只唸終態 → 2 紅；後端 UUID 守衛 → 2 紅。
- **Task 7**：三個夾具改成 hook 真的會產生的列（`wsFxFeed` helper），加 `feedConnected: true`；`--update-snapshots=all` 後只留 `generation-workspace-v2/{running,budget_ceiling,complete-with-failures}` 三張 darwin，其餘 168 張還原；三張 `-linux` 已 `git rm` 交給 CI bootstrap。e2e 新增一條（真瀏覽器 mock `/api/v1/events`），chromium 4/4 綠。
- 🎭 **A11y Pre-Flight: PASS**（3 個元件：`EventLogPane`、`FeedRowItem`、single 列；jsx-a11y 對本張檔案 0 warning，全 repo lint 條目與 stash 前逐條相同）。清單拿掉 `aria-live`（保留 `aria-label`）；**全頁只有一個** `aria-live`（sr-only、一直掛著、`running → complete` 時是同一個 DOM 節點，測試釘住）；只唸結果與批次列；所有裝飾圖示與 `·` 都 `aria-hidden`；旋轉圖示帶 `motion-reduce:animate-none`。本張沒有 modal、沒有 combobox、沒有 TMDb `<img>`。
- 🎨 **UX Verification: PASS**（對 `f11-d-v2`、`f12-d-v2`、`f11-spec-log`）：

  | 區域 | 設計 | 實作 | 相符 |
  | --- | --- | --- | --- |
  | 欄頭 | h44、`[0,14]`、gap 10、`activity` 14 `$text-secondary`、Body 600／Label muted | `h-11 px-3.5 gap-2.5`、`Activity h-3.5` secondary、`text-sm font-semibold`／`text-xs` muted | ✅ |
  | 清單 | `[6,0]` 無 gap | `py-1.5` | ✅ |
  | 列 | `[8,14]` gap 8、16 圖示、Body 600、`·` muted、項目 Body secondary、Mono 尾端 | `px-3.5 py-2 gap-2`、`h-4 w-4`、`text-sm font-semibold`、`·` muted、`text-sm` secondary、`font-mono text-sm` | ✅ |
  | 已過／進行中 | 勾 `$text-muted`＋secondary／`loader-circle` accent | 同 | ✅ |
  | 預算列 | `circle-alert` `$warning-text`、「本批次」、`$5.00` `$text-primary` | 同 | ✅ |
  | 底部 F11／F12 | `[10,14]` gap 8；F12 第二行靠右 `circle-pause` 13＋Label 500 `$warning-text` | `px-3.5 py-2.5`、`flex-col gap-2`；`justify-end gap-1.5`、`h-[13px]`、`text-xs font-medium` warning-text | ✅ |
  | 11px | 稿上 Label 12 | 膠囊字與「僅狀態事件…」維持 `text-[11px]`（凍結，`disc-2026-09-11px-micro-label-not-on-type-scale`） | ⚠️ 刻意 |
- **Pre-existing failures**：無。`pnpm nx test web` 270 檔／3843 條全綠；`pnpm nx test api` 全綠。

### Discovery Triage

- **Did this story discover any work outside its current scope?** **YES** —— 建單當下處理如下。

- **① expand-scope-in-place**
  - 抽內嵌字幕的項目不出現、`subtitle_progress` 三個發送者（🔴 #1、#2）→ **AC #3**
  - 預算／取消把暫停寫成失敗、雙終態、只有 `changed_item` 的結果（🔴 #3–#6）→ **AC #3**
  - 百分比洗版與逐條播報、不會捲到底（🔴 #7–#9）→ **AC #3、#6**
  - 沒有片名、批次列寫最後一部片名（🔴 #10、#11）→ **AC #2、#3**
  - 單部沒有片名（🔴 #12；既有 `disc-2026-09-single-job-title-missing`）→ **AC #4、#8**
  - 失敗講英文（🔴 #13）→ **AC #5**
  - 膠囊不知道連線狀態、終態後仍在轉、批次項目污染 `singleJobs`（🔴 #14–#16；dsr-6d-c-1 CR M6 交代）→ **AC #3、#6**
  - `items` 白轉（🔴 #17；既有 `disc-2026-09-batch-sse-items-transform-cost` 工作區那一半）→ **AC #3**
  - 設計對齊與詞彙（🔴 #18–#22）→ **AC #1、#2、#6**
  - 夾具與測試釘死錯的行為、e2e 零覆蓋（🔴 #23–#25）→ **AC #9、#10**

- **② spawn-blocking-story**：無（依賴的 `dsr-6d-a`／`dsr-6d-b`／`dsr-6d-c-1` 都已合併）。

- **③ backlog-with-carry-forward-link**（建單時立）
  - `disc-2026-09-transcription-run-5min-hard-timeout` — 整個 run 寫死 5 分鐘（P2 待實測）
  - `disc-2026-09-pipeline-asr-fallback-extracting-label` — 退回語音辨識時 D6 說「抽取內嵌字幕中…」
  - `disc-2026-09-workspace-single-job-lost-terminal` — 單部任務終態被丟就永遠「進行中」（CR M6 的單部那一半）

- Reference: `project-context.md` Rule 24

### File List

**新增**

- `apps/web/src/components/subtitle/generationEventCopy.ts`（＋`.spec.ts`，20 條）
- `apps/api/internal/services/transcription_event_title_test.go`（5 條）
- `_bmad-output/screenshots/flow-f-subtitle-v2/f11-spec-log.png`

**修改**

- `apps/web/src/hooks/useGenerationJobsFeed.ts`（重寫）、`useGenerationJobsFeed.spec.ts`（重寫，25 條）
- `apps/web/src/components/subtitle/GenerationWorkspaceV2.tsx`（`EventLogPane` 重寫、`FeedRowItem`、容器 `seedBatch`／`feedConnected`、single 列標題）
- `apps/web/src/components/subtitle/GenerationWorkspaceV2.spec.tsx`、`GenerationWorkspaceContainer.spec.tsx`
- `apps/web/src/routes/test/-gallery.fixtures.tsx`
- `apps/api/internal/services/transcription_service.go`（`eventTitle`＋六個送出點＋`[@contract-v2]` 註解）
- `tests/e2e/generation-workspace.spec.ts`
- `tests/visual/components.visual.spec.ts-snapshots/components/generation-workspace-v2/{running,budget_ceiling,complete-with-failures}/default-visual-darwin.png`
- `ux-design.pen`、`_bmad-output/pen-tokens.json`、`_bmad-output/screenshots/flow-f-subtitle-v2/{f11-d-v2,f12-d-v2}.png`、`scripts/export-pen-screenshots.py`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`
- `_bmad-output/implementation-artifacts/ux3-ai-2-workspace-frontend.md`（AC drift reference — see Completion Notes；本張不改該檔內容）

**刪除**

- `tests/visual/components.visual.spec.ts-snapshots/components/generation-workspace-v2/{running,budget_ceiling,complete-with-failures}/default-visual-linux.png`（過期基準；交給 CI bootstrap）

## Change Log

| 日期 | 內容 |
| --- | --- |
| 2026-09-18 | 🚧 **REVIEW**（dev-story, Amelia）。Task 1–8 全數完成。閘門：lint 0 errors（warning 條目與改動前逐條相同）、typecheck、design-tokens、format、**web 3843/3843**、api 全綠、e2e `@generation-workspace` chromium 4/4。11 項修法 mutation check 全部有牙。`transcription_*` 加 `title`（`[@contract-v2]` 加寬不升版）。設計稿改 F11／F12 紀錄＋新規格畫面 `F11-SPEC-LOG`。視覺三張重生、三張 `-linux` 已 `git rm`。 |
| 2026-09-18 | Story 建立（SM Bob, create-story；main `ff16deef`）。`dsr-6d-c` 的第二半、`dsr-6d` 的最後一塊。兩個唯讀稽核代理（設計稿節點／後端 SSE 事件）＋ SM 自讀程式碼，找到 **25 條**：最有感的是**翻譯每 10 句就新增一列、而且沒有一列說得出是哪一部**；最容易做錯的是**預算用完時後端先送單部失敗、後送批次暫停**，紀錄只要還相信前者，左右兩欄就對同一部講相反的話——中心規則定為「批次成員的終態只看 `changed_item`」。`subtitle_progress` 有三個發送者、兩個是英文且不是生成，接上前必須先做成員過濾。設計稿有三種列系統給不出來（排入佇列、提取音訊完成、本次用量），而系統必須產生的五種列稿上沒畫 → 新規格畫面 `F11-SPEC-LOG`。唯一的後端改動是讓 `transcription_*` 帶上後端早就算好的片名（`[@contract-v2]` 加寬不升版），同時根治 `disc-2026-09-single-job-title-missing`。八項 SM 裁定；新立三張 disc（含一個可能很常見的 **5 分鐘整體逾時**）。 |
