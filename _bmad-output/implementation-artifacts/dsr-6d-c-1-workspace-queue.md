# Story DSR.6d-c-1：生成工作區的佇列改看後端給的真相——批次一結束不再整區空白，切走分頁再回來也不會卡在「進行中」

Status: review

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As the person who opens 活動 › 生成字幕 to watch a batch on a full page,
I want 佇列上每一列寫的就是那部片真正的結果、批次結束後佇列還在、切走分頁再回來畫面不會騙我,
so that 我看得到哪幾部失敗、為什麼失敗，而且不用重開瀏覽器才能脫離一個永遠停在「進行中」的畫面。

## Context

`dsr-6d`（批次生成＋生成工作區）的**第三塊**。前兩塊已合併：

| 單子 | 範圍 | 狀態 |
| --- | --- | --- |
| `dsr-6d-a` | 後端：每一部的 `status`／`reason`、`last`＋`dismiss`、202 `progress`、`error` 終態、分集劇名、SSE `changed_item` | ✅ done（PR #464） |
| `dsr-6d-b` | 批次對話框 F8／F9 → `GenerationBatchDialogV2.tsx`、`useGenerationBatchProgress.ts`、新檔 `generationQueueRow.ts` | ✅ done（PR #466） |
| **`dsr-6d-c-1`（本張）** | 工作區**主欄＋頁首＋底部列**：F11-D-v2 `l8FsB`、F12-D-v2 `iH98f`、F13-D-v2 `F7ohe` 的左側 ＋ 母版 `Component/GenQueueRow-v2` `aw4Qr` → `GenerationWorkspaceV2.tsx`、`generationWorkspace.ts` | 本張 |
| `dsr-6d-c-2` | 工作區**右欄「即時活動」**：`useGenerationJobsFeed.ts` ＋ `EventLogPane` ＋ 事件紀錄的設計對齊 | 下一張 |

⚖️ **SM 裁定（2026-09-18）：`dsr-6d-c` 拆成兩張。** 建單稽核（三個唯讀代理）在工作區找到 **20 個現況問題**、加上一個程式碼完全沒實作的母版、加上三張稿的字級／間距／顏色對齊、加上三個夾具與六張基準線——單張的規模會超過 `dsr-6d-b`（8 個 task、2,755 行）約四成。切分線選在**版面**而不是「先行為後外觀」，因為右欄有自己的 hook（`useGenerationJobsFeed.ts`）、自己的 SSE 連線、自己的一整串 bug（原 sprint 條目的 (g)），而且左右兩欄的檔案幾乎不重疊。**每一半都能獨立出貨**：本張讓左邊說真話，`dsr-6d-c-2` 讓右邊說真話。

⛔ **本張不碰 `EventLogPane`（`GenerationWorkspaceV2.tsx:205-251`）與 `useGenerationJobsFeed.ts`**，唯一例外見 AC #7（那條 SSE 連線在終態仍開著，膠囊誠實性要一起看）。
⛔ **不碰對話框**（`GenerationBatchDialogV2.tsx`），唯一例外是 AC #4 要它 invalidate 一個 key。
⚠️ **驗收基準是 `.pen` 節點值**（文字逐字、字級變數、顏色變數、間距變數）。PNG 只是參考。

### 🔴 建單時查到的事（main `3a520961`；行號皆為現況）

**A. dsr-6d-b 剛造成的破壞（最該先修）**

1. **批次一結束，工作區主欄整區空白。** 對話框在終態會 `queryClient.removeQueries({queryKey: generationBatchItemsKey})`（`GenerationBatchDialogV2.tsx:926`，檔頭 `:62-67` 的承諾）。工作區讀同一個 key（`GenerationWorkspaceV2.tsx:577-582`）而且是 `enabled: false` ＋ `queryFn: () => []` 的**純快取訂閱**——query 被移除後 observer 掛上一筆 `data === undefined` 的新 entry，`queryFn` 永遠不會跑，**那份資料再也回不來**。於是 `items.length === 0`，而佇列整塊（`:448`）被 `items.length > 0` 守著 → `<h2>生成佇列</h2>`、終態標籤、「下次繼續」、整個 `<ul>` **全部不渲染**。批次剛結束、使用者最想看「哪幾部成功哪幾部失敗」的那一刻，畫面中間是空的。

**B. 列在說謊**

2. **列狀態是猜的。** `:334` 用 `deriveRowStates(items, progress, failedIds)`——那個函式在 dsr-6d-b 已經被標成 **FALLBACK ONLY**（`GenerationBatchDialogV2.tsx:78-83` 檔頭逐字寫「used when no `progress.items[]` snapshot is available」）。真實情境：10 部的批次第 4 部被拒（`busy_elsewhere`），後端 `items[3].status='failed'`，工作區顯示**完成**；同一時間對話框顯示「這部正在別處處理」。**兩個畫面對同一部片講相反的話。**
3. **失敗永遠不顯示。** 容器（`:610-633`）根本沒傳 `failedIds`，取預設 `EMPTY_FAILED`（`:317`）；而且工作區**沒有任何地方會產生 `failedIds`**（對話框有 `:880-892` 那個 effect，工作區沒有對應物）。`deriveRowStates` 的 `complete` 分支（`GenerationBatchDialogV2.tsx:91`）於是一律回 `done`。
4. **列文案自己寫了一份。** `:126-158` 的 `QueueRowLabel` 與 `generationQueueRow.ts:69` 的 `queueRowLabel()` 平行存在：`'active'` 寫死**「轉錄中」**（`:144`）不看 `phase`（抽內嵌字幕的路線根本不轉錄）、`'failed'` 一律「失敗」**沒有 reason**（所以「沒有可用的字幕來源」在工作區永遠說不出來）。`:189` 又把 `phase === 'failed'` 映成 `'idle'`，失敗那列的步驟條會退回「提取音訊」重跑一次。這三條都是 dsr-6d-b 🔴 #9 在對話框修掉的同一批。
5. **`complete` 不看 `failCount`。** `TERMINAL_COPY.complete`（`:290`）用 `--success-text` 寫「全部完成」，即使 3 部失敗。對話框已改成「完成 N 部、失敗 M 部」中性。
6. **attach／single 硬編 `state="active"`**（`:427-433`、`:491-513`）；single 的列標題是 `job.message || job.mediaId`（`:494`）——拿不到訊息就顯示 **UUID**。
7. **`remaining` 用減法猜**（`:259` `pausedCount || Math.max(0, totalItems - done)`）。後端 AC #1 已保證 `paused_count` 等於 paused 的數量，不需要猜。

**C. 畫面會卡死／不同步**

8. **進行中那一列不跟著換片。** `:584-599` 的 effect 依賴陣列是 `[live, statusQuery.data]`（`:599`），**不含 `batch.progress.currentMediaId`**。而 `useGenerationProgress` 是有 mediaId 過濾的（`useGenerationProgress.ts:229`）。第 1 部跑完換第 2 部之後，工作區仍訂閱第 1 部，第 2 部的事件全被丟掉 → 那一列的步驟條**卡在「完成」不動**直到整批結束。對照：對話框有專屬 effect `GenerationBatchDialogV2.tsx:867-872`，依賴 `[batch.status, currentMediaId, startItemTracking]`。
9. **同一段還會洗掉狀態**：`statusQuery.data` 每換一次身分就重呼 `batch.startTracking(probe.progress)`，而 `START` reducer（`useGenerationBatchProgress.ts:117`）是 `{...initialState, ...payload, status:'running'}`——**把累積的 SSE 狀態整個洗掉並強制回 `running`**。按「重試」（`:629-632` 的 `refetch()`）就會觸發。
10. **狀態查詢的 key 沒有匯出。** `:566` 是 inline 字面值 `['subtitles','generation-batch','status']`，全 repo 沒有第二個地方提到它，**所以沒有任何人能讓它失效**；它也沒設 `staleTime` → 吃 `queryClient.ts:8` 的預設 **5 分鐘**。後果：在工作區上面用對話框啟動批次，對話框只 `setQueryData(generationBatchItemsKey, …)`（`GenerationBatchDialogV2.tsx:979`），從不 invalidate status → 工作區的 `batch.status` 永遠是 `'idle'`。但 `jobs.startTracking()`（`:591`）有連上而且**不過濾**（`useGenerationJobsFeed.ts:252-258`），於是 `singleJobCount > 0` → `deriveWorkspaceMode` 回 **`'single'`**。**實際畫面**：標題變「生成字幕」、`OverallStrip` 顯示「已完成 **0 / 0** 部」「**$0.00**／上限 **$0.00**」、下面一列「進行中任務」標題是 UUID，而剛剛快取進去的 5 筆佇列**完全不畫**。批次明明在跑。
11. **切走分頁再回來卡死。** `live = active && usePageVisibility()`（`:548-549`）。切回來時 `statusQuery.data` 還在 5 分鐘 staleTime 內（`refetchOnWindowFocus` 只對 stale 生效）→ **不打網路**，effect 因 `live` 變化重跑，讀到快取的 `{running:true}` → 強制 `status:'running'` 並**重開一條 EventSource**。若批次在那 30 秒內跑完，畫面重建成「進行中」、進度條停在結束前的數字、SSE 膠囊還在說「即時更新」，而 SSE 上再也不會有事件。**5 分鐘內按什麼都救不回來**，`onRetryData` 是唯一出口但畫面沒有任何提示（`dataError` 為 false，`:608`）。
12. **終態標籤與「下次繼續」被鎖在 `items.length > 0` 裡面**（`:448`）。冷接上的工作區碰到預算上限時，橫幅會畫但**「下次繼續」整顆不存在**——那 26 部暫停的片在工作區上沒有任何出路。

**D. 控制項與誠實性**

13. **「全部取消」沒有確認、rejection 沒接。** `:621-623` 是 `void subtitleService.cancelGenerationBatch();`——①沒有確認步驟（對話框有兩段式確認）；②rejection 變成 unhandled promise rejection，網路失敗時畫面**完全沒反應**；③忽略 `{cancelled:false, running:false}`（對話框 `:1058-1066` 已處理成重查）。型別也分岔了：工作區 `() => void`（`:312`）vs 對話框 `() => Promise<void>`。
14. **終態沒有「關閉」。** 後端 `POST …/generation-batch/dismiss`（`generation_batch_handler.go:66`）從 dsr-6d-a 就存在，`subtitleService.dismissGenerationBatch()`（`subtitleService.ts:641-648`）也寫好了，但**全 repo 零呼叫端**（只有 spec 與 mock 提到）。F12 底部列的「關閉」`O3RFR` 就是它唯一的使用者。
15. **SSE 膠囊在終態仍亮。** `SseChip`（`:112`，在 `OverallStrip` 內）無條件畫，但批次 hook 在終態已 `closeSSE()`（`useGenerationBatchProgress.ts:216`）。膠囊在說謊。
16. **attach 佔位 `animate-pulse` 永遠在動**（`:435-441`），沒有終止條件。DESIGN.md:514-516：**會動的元件就是在宣稱「現在有工作在跑」**。
17. **缺字幕數是錯的欄位。** `:618` 讀 `previewQuery.data?.totalItems`，那個欄位的語意是 **movies-only 且 FROZEN**（`subtitleService.ts:470-471`）；同意清單實際會列的是 `totalItemsIncludingEpisodes`（`:477`，`ScanProgress.tsx:68` 已經改讀它）。閒置畫面寫的數字跟按下去看到的**對不上**。
18. **批次只在工作區看時，結束後不刷新片庫徽章。** 終態的 invalidate 全部寫在對話框的 effect（`GenerationBatchDialogV2.tsx:899-927`）；對話框沒開時沒有人做。

**E. 設計對齊（左側）**

19. **母版 `Component/GenQueueRow-v2` `aw4Qr` 程式碼完全沒實作。** 稿上一列是：40×60 海報、**BodyLg 600** 片名、**片名下一行的狀態說明** `lUZol`（Body `$text-secondary`，逐狀態覆寫）、右側**藥丸狀態徽章** `IcPTo`（`$radius-pill`、tint 底、14px 圖示、Label 標籤）、以及只在翻譯階段出現的 `OMD2y` 百分比。程式碼 `QueueRow`（`:171-199`）只有海報＋片名＋一段純文字標籤：**沒有底色、沒有圓角、沒有 padding、沒有狀態說明那一行、沒有百分比**，海報是 38×54（稿上 40×60），失敗圖示用 `CircleAlert`（稿上 `triangle-alert`）、暫停用 `CirclePause`（稿上 `pause`）、排隊中**完全沒有圖示**（稿上 `hourglass`）。
20. **金額穿了赭色。** `:272` `usd(progress.budgetUsd)` 用 `--warning-text`。DESIGN.md:308「**金額不穿狀態色**…金額一律 `--text-primary`」。稿上 `yfxtS`／`o0o6yH`／`iGrEe` 三處金額**都是 `$text-primary`**。
21. **頁首圖示塗泥金。** `:354` 的 `Captions` 圖示是 `--accent-text`，而且三張稿的標題（`j7IMjB`／`f2TQx7`／`G0fna`）**都是純文字沒有圖示**。泥金＝正在跑（DESIGN.md:293），這顆圖示連 `idle` 模式都在畫——無條件宣稱有工作在跑。
22. **狀態膠囊整個沒實作。** F11 `X22E1A`（`$accent-tint` 底＋8×8 `$accent-text` 圓點＋Label「進行中」）、F12 `FKDCf`（`$warning-tint` 底＋`$warning` 圓點＋`$warning-text`「已達上限」）在程式碼裡不存在。
23. **F12 底部列不存在。** `v9WuzZ`（`$bg-secondary` 底、上邊框、`justifyContent:"end"`）＋`O3RFR`「關閉」＋`BXq7X`「下次繼續」。程式碼把「下次繼續」塞在佇列標題列（`:453-462`），「關閉」沒有。
24. **F13 標題錯。** 稿上 `G0fna` 是「**生成工作區**」，程式碼 `:355` 只會輸出「生成字幕」或「批次生成字幕」——字串「生成工作區」**全檔不存在**。
25. **字級與間距**（節點 → 程式碼）：`:345` 麵包屑 `text-[13px]` → Body 14（`text-sm`）；`:451`「生成佇列」`text-[15px]` → BodyLg 16（`text-base`）；`:488`「進行中任務」`text-[15px]`（稿上無此節點，見 AC #2 裁定）；`nqJeZ`／`qdBfc`／`XulIH` 是 BodyLg 16 而程式碼 `:80-83`／`:107-109` 是 `text-sm`；`bSNmA` 是 BodyLg 600 而 `:182` 是 `text-sm` 常規；`MmcmP` 是 Body 500 而 `:374` 是常規。間距：ov-track 180 vs `w-44`(176)（`:87`）、sse-chip padding 6/8 vs `px-2 py-1`（`:119`）、gen-queue gap 12 vs `gap-2.5`(10)（`:449`）、`aw4Qr` gap 10 vs `gap-3`(12)（`:175`）、海報 40×60 vs 38×54（`:180`）、按鈕 padding 20 vs `px-4`(16)（`:379`／`:407`／`:458`／`:531`）、p1-card padding 24 vs `py-14`(56)（`:390`）。⛔ **11px 凍結**（`disc-2026-09-11px-micro-label-not-on-type-scale`）：`:74`／`:101`／`:119`／`:247` 四處 `text-[11px]` **原樣保留**。
26. **`--radius-pill` 這個 CSS 變數不存在**（`styles.css` 只有 `--radius-sm/md/lg`，DESIGN.md:560 已記）。藥丸一律用 `rounded-full`。

**F. 測試與夾具**

27. **容器零測試。** `GenerationWorkspaceV2.spec.tsx` 只 import presentational 的那個（`:4`），7 條測試全部直接餵 props；`ActivityHub.spec.tsx:27-31` 把整個模組 stub 掉。上面 8–18 **每一條都沒有測試**。**e2e 零覆蓋**——`tests/` 底下搜 `activity`／`view=generation`／`workspace-queue-row` 都是 0 筆。
28. **`budget_ceiling` 的視覺基準是一張謊言。** 夾具（`-gallery.fixtures.tsx:5124-5156`）寫 `totalItems: 38 / pausedCount: 26` 但只餵 3 列；`deriveRowStates` 的 budget 分支要 `i >= totalItems - pausedCount` = `i >= 12` 才算 paused → **三列全部畫成「完成」**，而正上方橫幅寫著「已完成 12、剩餘 26 下次繼續」。這張自相矛盾的圖已經 commit 進基準了。
29. 三個夾具的 `progress` **全部沒有 `items` 鍵**（`:5081`／`:5129`／`:5163`），`items[]` 只有 `{mediaId, title}`（**沒有 `mediaType`、沒有 `seriesTitle`**），`activeItemProgress` 缺 `partial`／`englishKeptBlocks`。`props` 型別是 `Record<string, unknown>`，**typecheck 一律放行**。

### 設計稿節點（本張要看的）

| 代號 | 節點 | 本張要對齊的 |
| --- | --- | --- |
| F11-D-v2 | `l8FsB` | 頁首 `jxF9O`[24,32,16,32] gap 14、麵包屑 `O6OQNE`（`mqvB6` chevron-right 12）、標題 `j7IMjB` H3/700、狀態膠囊 `X22E1A`（`$accent-tint`＋`CpkuY` 8×8 `$accent-text`＋`LiskM` Label `$accent-text`「進行中」）、統計列 `whlu3`[12,16] gap 32 `$radius-md`、`Rd1QX` 軌道 180×6＋`rA11p` 填色 `$accent-primary`、`GqKQj` H3 Mono／`nqJeZ` BodyLg Mono `$text-muted`／`qdBfc` BodyLg Mono、`c0Nss` BodyLg Mono 600 `$text-primary`／`XulIH` BodyLg Mono `$text-secondary`、`PUymh` sse-chip[6,8] `$radius-sm` `$info-tint`、`CTzRp`「生成佇列」BodyLg 600、`tk1Rj`「已完成項目已收合 · 捲動可查看完整佇列」、四列 instance `oxc7d`／`OkdGK`／`uMKsh`／`f14sNh` |
| F12-D-v2 | `iH98f` | 狀態膠囊 `FKDCf`（`$warning-tint`＋`gavvw` `$warning`＋`QwyBf` `$warning-text`「已達上限」）、橫幅 `zn1DI`[12,16] `$warning-tint` 無邊框＋`Zeu0P` circle-alert 18＋`AFy6j`/`mSAQn`/`zEsvc`/`WUJH2` Body 500 `$text-primary`＋`yfxtS` `$5.00` Mono 600 **`$text-primary`**＋`m2pTFN`/`bOyrF` 數字 Mono 600 `$warning-text`、底部列 `v9WuzZ`[14,32] gap 12 `$bg-secondary` 上邊框 justify-end＋`O3RFR`「關閉」(YDPhc h44)＋`BXq7X`「下次繼續」(otvKh h44)、四列 instance `bbDtK`／`FU91l`／`F5VI9N`／`jryaP` |
| F13-D-v2 | `F7ohe` | 標題 `G0fna`「**生成工作區**」H3/700、**無狀態膠囊**、body `XZyp2`[8,32,24,32] gap 20、①`i7iIe`[24] gap 12 `$radius-lg`＋`Z0cob` sparkles 40＋`aA2oo`「目前沒有進行中的生成」＋`eqSWB`（`wvyhY`「缺繁中字幕：」／`B2jrTK` H4 Mono／`D9H6j`「部」）＋`gqUxl`「批次生成字幕」(otvKh h44)＋`U72BV` 註記、②`vZjuA`[24] gap 16＋`CMihI`「已接上進行中的批次」BodyLg 600＋`Nu1jB` 總覽（`iGrEe` `$0.12` **`$text-primary`**）＋`ky6bX` 活列＋`T8Cvpd`/`s4ByS` 56 高 `$bg-tertiary` opacity .5＋`X8KfD` 琥珀註記、③`t9FVqB`[24] gap 16＋`V4G4xA`[14,16] `$error-tint` 無邊框＋`V22Ub` circle-alert 20＋`MmcmP` Body **500**＋`KJmTs`「重試」(YDPhc h44)＋`ZiBAb` 註記 |
| 母版 | `Component/GenQueueRow-v2` `aw4Qr`（660 寬、[14,16]、gap 10、`$radius-lg`、`$bg-secondary`、`$border-subtle` 1） | `lJuzY` 海報 **40×60** `$radius-sm` `$bg-tertiary`、`QYvU3` 文字欄 gap 4、`bSNmA` 片名 **BodyLg 600** `$text-primary`、`lUZol` 狀態說明 **Body `$text-secondary`**、`s5N8u` 右側 gap 8、`OMD2y` 百分比 Body **Mono** `$accent-text`（預設 `enabled:false`）、`IcPTo` 徽章[6,10] gap 6 **`$radius-pill`** tint 底、`CQhw2` 圖示 **14×14**、`OLg1J` 標籤 Label、`dxgox` 步驟條槽（預設 `enabled:false`） |
| 母版 | `Component/Button/Secondary` `YDPhc`（h36→稿上覆寫 44、[8,20]、`$bg-tertiary`、Body **500** `$text-primary`）、`Component/Button/Primary` `otvKh`（同幾何、`$accent-primary`、Body **600** `$text-on-accent`） | 關閉／重試／全部取消 vs 下次繼續／批次生成字幕 |

**`ctx.problems` 基線：`l8FsB`／`iH98f`／`F7ohe`／`aw4Qr` 四個節點底下目前是 0 筆裁切**（建單時以 Pencil MCP 實測）。改完必須仍是 0。

字階（`DESIGN.md:390-394`）：Label 12／1.5、Body 14／1.625、BodyLg 16／1.625、H4 18／1.5、H3 20／1.375。
Space token：`2xs`=2、`xs`=4、`xs-plus`=6、`sm`=8、`sm-plus`=10、`md`=12、`md-plus`=14、`lg`=16、`lg-plus`=20、`xl`=24、`2xl`=32。

---

## Acceptance Criteria

1. **設計稿先改（Pencil MCP；動手前每個節點再 `Get` 一次）。**
   - **F12 的列與數字改中性**（⚖️ Alexyu 2026-09-17 Q1：「批次層提示（橫幅、狀態膠囊、紀錄那一行）用赭色，每一列『已暫停—下次繼續』與金額、數字一律中性」）：三個 instance `FU91l`／`F5VI9N`／`jryaP` 的 `descendants` 覆寫裡，`lUZol`／`OLg1J` 的 `fill` 由 `$warning` → **`$text-muted`**、`CQhw2` 的 `fill` 由 `$warning` → **`$text-muted`**、`IcPTo` 的 `fill` 由 `$warning-tint` → **`$bg-tertiary`**；橫幅裡的 `m2pTFN`（「12」）與 `bOyrF`（「26」）由 `$warning-text` → **`$text-primary`**。⛔ **橫幅底 `zn1DI` 的 `$warning-tint`、圖示 `Zeu0P`、膠囊 `FKDCf`／`gavvw`／`QwyBf` 全部維持赭色**——那是批次層的提示。
   - **`$warning` 當文字色一律改 `$warning-text`**：`Zeu0P`（banner-ic）、`F0IlN`（stopped-ic）、`ShmdC`（evt-ic）、`gavvw`（pill-dot）。程式碼的 `local/no-base-semantic-as-text` 擋基礎語意色當文字，稿要跟上（先例：dsr-6d-b 在 `J46QF` 註記寫明的那條）。
   - **F12 的示例數字要自洽**：橫幅寫「已完成 12、剩餘 26」而 `GqKQj`／`qdBfc` 是「12 / 38」——12＋26＝38 ✅ 已一致，**不要動**。但 `dFMhI` 填色寬度是 57，而 12/38＝31.6%，軌道 `dEfRe` 是 180 → 應為 **57**（180×0.316＝56.9）✅ 已一致，**也不要動**。⚠️ 只確認、不改。
   - **拿掉 F12 的「全部取消」`ZMYbU`**（若仍存在；建單稽核在 `iH98f` 底下沒有列到這個節點，**動手前先 `Get("ZMYbU")` 確認**——存在就刪，不存在就在 Completion Notes 記「已不存在」）。預算上限是終態，取消沒有意義。
   - **轉錄百分比只在翻譯階段**：F11 `OkdGK` 的 `OMD2y` 覆寫是 `{enabled:true,content:"45%"}`——**保留**（那一列的 `lUZol` 是「轉錄音訊中…」，但 `dxgox` 掛的是完整步驟條，45% 在稿上是「翻譯中」那一格的數字）。⚠️ 改的是 **`lUZol` 的文案**：「轉錄音訊中…」→「**翻譯中…**」，讓同一列的三個說法（狀態說明、徽章 `OLg1J`、百分比）一致；`OLg1J` 由「轉錄中」→「**翻譯中**」。F13 `ky6bX` 同樣處理。
   - **過期註記**（三則，都在 `JzmvC`／畫面內）：
     - `rhhQ0`→`K5gn5g`（「F11 = 探索性變體…dev 在 Epic 6 以 F1–F10 dialog 流為準」）：**改成**「F11／F12／F13 已是出貨規格（`GenerationWorkspaceV2.tsx` 實作；dsr-6d-c-1）。」——它現在跟程式碼檔頭直接矛盾。
     - `DP53I`→`YyiUQ`（單部任務 BE 缺口）：**整個 `DP53I` frame 移除**。`c4FIoB`→`m9nBEI` 最後一條已寫「單部任務的可見性已由 active_jobs 提供（已合併）」。
     - `X8KfD`→`B5D2a5`（「status 探測無 items[]」）：**改成**「批次的完整佇列來自 status 的 `progress.items`／`last.items`（dsr-6d-a AC #1/#3）；此卡片只在**單部任務**或舊版後端時降級。」
   - **兩個設計註記用了赭色當底**（`DP53I`、`X8KfD` 的 `$warning-tint` 底＋`$warning-tint` 邊框）：`DP53I` 直接刪；`X8KfD` **保留赭色**——它仍是「能力邊界」類註記，與 `c4FIoB` 同一族（⚖️ 2026-09-11 的「不准赭色」裁定針對的是**畫面內容**，不是規格註記；在 Completion Notes 記下這個判讀）。
   - 新文字節點一律綁變數（`fontSize: $Type/...`、`fill: $...`）；每張改完 `ctx.problems` 掃裁切（**基線 0，改完必須仍是 0**）；存檔後確認 ` M ux-design.pen` 並讀磁碟檔確認落盤；匯出後只 stage `flow-f-subtitle-v2/{f11-d-v2,f12-d-v2,f13-d-v2}.png` ＋ `_bmad-output/pen-tokens.json`（若變），其餘重繪雜訊還原。**若 `f11-m-v2`、`f3-d-v2`、`f8-d-v2` 或元件庫也變了，代表動到了母版——回頭檢查**（⚠️ `aw4Qr` 是**新母版**，本張只讀不改；`XkGvG` 步驟條母版 dsr-6b 已對齊，⛔ 不准動）。

2. **列改讀後端的 `items[]`（🔴 #1、#2、#3、#4、#6）。**
   - **列的顯示語彙一律用 `generationQueueRow.ts`**（dsr-6d-b 已建）：`queueRowLabel({status, reason}, phase?)` 與 `queueRowTitle({title, seriesTitle?})`。⛔ **刪掉工作區自己那份 `QueueRowLabel`（`:126-158`）**，不准再寫第二套文案。⚠️ 若需要新的狀態文案（例如 `complete` 階段顯示「收尾中」，見 AC #10 的交代），**改 `generationQueueRow.ts` 並補它的 spec**，讓對話框一起受惠。
   - **三層來源，優先序寫死**（與對話框同一套）：
     1. `batch.progress.items` 有內容 → 每一列的 id、標題、劇名、狀態、原因**全部**來自它；
     2. 否則 `singleJobs` 有內容 → single 模式的機會列（見下）；
     3. 兩者皆空 → idle／attach 面板。
   - **`generationBatchItemsKey` 這個 query 整個移除**（`:577-582`），連同 `:33` 的 import。它的擁有者是對話框，清除時機由 dsr-6d-b 的檔頭承諾固定在終態，工作區不能要求它改。**批次結束後的佇列改讀 `GET …/status` 的 `last.items`**（AC #3）。
   - **`deriveRowStates` 的 import（`:31`）與呼叫（`:334`）移除**；`failedIds` prop 與 `EMPTY_FAILED`（`:317`）一併移除。⛔ **`GenerationBatchDialogV2.tsx` 的 export 本身不准刪**——`ActivityHub.spec.tsx:16-23` 的 mock 工廠與對話框的退回路徑仍依賴它們；本張只是不再使用。⚠️ 移除 import 後 `ActivityHub.spec.tsx:19` 那行註解（「keys imported by the workspace」）要改，但**工廠的 key 保留**（多餘無害，刪了反而可能讓別的 import 爆）。
   - **`data-state` 改寫後端的 `status`**（`:174`）：`active`→`running`、`stopped`→`cancelled`。spec `:67-69`、`:125` 的斷言一併改。
   - **single 模式的列標題改用真的片名**：`:494` 的 `job.message || job.mediaId` → 讀活動 API 的 transcription job 標題（`useActivity()` 的 `activeJobs`，`hooks/useActivity.ts`）；**拿不到就顯示「處理中的項目」，⛔ 絕不顯示 UUID**。
   - `remaining`（`:259`）直接用 `progress.pausedCount`，不要減法。

3. **狀態探測三條路 ＋ 終態出口（🔴 #1、#10、#11、#12、#14）。**
   - **探測改成與對話框對稱的三條路**（`GenerationBatchDialogV2.tsx:826-863` 是範本）：`running && progress` → `batch.startTracking(progress)`；`!running && last` → **`batch.attachSnapshot(last)`**（不連線、不把 status 寫死成 running）；兩者皆無 → idle。⚠️ `attachSnapshot` 與 `connectionEpoch` 目前工作區**完全沒有取用**（`:551` 只解構 `startTracking`／`reset`）。
   - 🚨 **`last` 只接一次**：與對話框同一個陷阱（dsr-6d-b CR H2）——`last` 在後端留到下次開始／dismiss／重啟才清。用 `seenLastBatchIdRef` 記住這一次掛載已經顯示過的 `last.batchId`，同一個 `batchId` 第二次不再接。
   - **終態底部列（F12 `v9WuzZ`）**：`關閉`（Secondary）＋`下次繼續`（Primary，僅 `budget_ceiling`）。**「關閉」呼叫 `subtitleService.dismissGenerationBatch()`**（`subtitleService.ts:644`，本張是它的第一個呼叫端）→ 成功後 `batch.reset()` 回 idle。這同時是 `disc-2026-09-batch-dialog-last-attach-per-mount` 的根治法：後端忘掉 `last` 之後，對話框重新掛載也不會再重播。
     - dismiss 失敗要說話：一行 `role="alert"`「關閉失敗，請再試一次。」`data-testid="workspace-dismiss-error"`，⛔ 不要吞掉（Rule 13）。
     - ⚠️ `dismiss` 在批次還在跑時是 no-op（後端回 `{dismissed:false, running:true}`）——那種回應要當成「還在跑，不能關」處理，顯示同一行 alert。
   - **終態標籤與「下次繼續」不准再被 `items.length > 0` 守著**（`:448`）：改成由 `mode` 決定。
   - **查詢 key 匯出並讓對話框 invalidate**（🔴 #10）：把 `['subtitles','generation-batch','status']` 提成 `export const generationBatchStatusKey`（放在 `GenerationBatchDialogV2.tsx` 兩個既有 key 旁邊，⚠️ 同時要補進 `ActivityHub.spec.tsx:16-23` 的 mock 工廠，否則 ActivityHub 整支紅）。對話框在**啟動成功後**與**終態 effect**各 invalidate 一次。工作區的 `statusQuery` 加 `staleTime: 0`。
   - **切走再回來要重查**（🔴 #11）：`refetchOnWindowFocus: 'always'`（或在 `live` 由 false→true 時 `refetch()`）。並把 `connectionEpoch` 加進探測的依賴，比照對話框。
   - **effect 依賴要跟著換片**（🔴 #8）：逐部訂閱獨立成一條 effect，依賴 `[batch.status, batch.progress.currentMediaId, startItemTracking]`；⛔ **不要再用 `[live, statusQuery.data]` 同時管兩件事**（🔴 #9 的洗狀態就是這樣來的）。

4. **控制項誠實（🔴 #13、#15、#16、#17、#18）。**
   - **「全部取消」加兩段式確認**（比照對話框 `:627-690`）：`onConfirmCancelAll` 型別 `() => void` → **`() => Promise<void>`**；送出中 `aria-disabled`＋「取消中…」（⛔ 不要用 `disabled`）；失敗顯示 `role="alert"`「取消失敗，批次仍在進行。請再試一次。」`data-testid="workspace-cancel-error"`；`{cancelled:false, running:false}` → 觸發重查 status。
   - **SSE 膠囊只在執行中顯示**（`:112`／`OverallStrip`）。⛔ 右欄 `EventLogPane` 那顆（`:246`）**本張不動**——那條連線在終態確實還開著，歸 `dsr-6d-c-2`。
   - **attach 佔位不准永遠動**：`animate-pulse`（`:435-441`）只在 `mode === 'attach'` **且**批次仍在跑時；終態一律靜止（DESIGN.md:514-516）。
   - **缺字幕數改讀 `totalItemsIncludingEpisodes ?? totalItems`**（`:618`，先例 `ScanProgress.tsx:68`）。
   - **終態要刷新片庫**（🔴 #18）：工作區自己在終態（**同一個 `batchId` 只做一次**，比照 dsr-6d-b 的 `handledTerminalRef`）invalidate `libraryKeys.all`／`generationBatchPreviewKey`／`detailKeys.all`／`activityKeys.all`／`transcriptionEstimateKeys.all`。⚠️ 對話框也開著時會重複——**這是可接受的**（invalidate 是冪等的），⛔ 不要為了去重而把對話框的那段搬走。

5. **總結文案（SM 定，Sally／Alexyu 可在 review 推翻）。**
   - `complete` 且零失敗 → 「全部完成（N 部）」`--success-text`；**有失敗 → 「完成 N 部、失敗 M 部」中性**（🔴 #5；與對話框 `GenerationBatchDialogV2.tsx:371-387` 逐字一致）。`cancelled` → 「已取消：完成 N 部」中性。`error` → 沿用 `--error-text`「批次發生錯誤」。
   - `TERMINAL_COPY`（`:289-293`）這張表因此要改成函式或加上 `failCount` 分支；順手補 `budget_ceiling` 鍵（現在沒有，只因為 `isTerminal` 不含它才沒爆，`:466`）。
   - **不要重複播報**：可見的總結那一句自己帶 `aria-live="polite"`；⛔ **不要新增第二個 live region**（dsr-6d-b CR M4 的教訓：`aria-live` 區域與內容一起掛載，螢幕報讀根本不唸——要嘛用一直掛著的 sr-only 區，要嘛確定那一句所在的容器在模式切換時不會整個重新掛載）。

6. **外框、字級與母版（AC: 🔴 #19–#26）。**
   - **`QueueRow` 實作 `Component/GenQueueRow-v2` `aw4Qr`**：海報 **40×60**、片名 **BodyLg 600**（`text-base font-semibold`）、**片名下一行的狀態說明**（Body `--text-secondary`，內容＝`queueRowLabel` 的句子）、右側**藥丸徽章**（`rounded-full`＋tint 底＋**14px** 圖示＋Label 標籤）、只有 `running` 且階段是翻譯時顯示百分比（Mono `--accent-text`）。圖示逐一對齊：done `check`／failed **`triangle-alert`**／running `loader-circle`／queued **`hourglass`**／paused **`pause`**／cancelled 無圖示。
     - ⚠️ 檔頭 Rule 21 標記：`GenerationWorkspaceV2.tsx:1` 目前是 `// Design ref:` 的 screen 形式。實作了一個真母版之後改成 `// Implements: Component/GenQueueRow-v2 (aw4Qr)` ＋ 保留 `// Design ref:` 那行（Rule 21 允許多行；⛔ 動手前先看 `apps/web/src/eslint-rules/implements-pen-node-id.js` 接受哪幾種形狀，寫錯 `lint:all` 會紅）。
   - **狀態膠囊**（F11 泥金「進行中」／F12 赭「已達上限」）：`rounded-full`、8×8 圓點、Label 12。
   - **F12 底部列** `v9WuzZ`：`$bg-secondary` 底、上邊框、`justify-end`、padding [14,32]、gap 12；「關閉」用 Secondary（`bg-tertiary`＋`text-primary`＋Body 500＋**px-5**），「下次繼續」用 Primary（`accent-primary`＋`text-on-accent`＋Body 600＋**px-5**）。所有按鈕 `min-h-[44px]`。
   - **F13 標題「生成工作區」**（`:355`）。
   - **字級**：`:345` 13px→`text-sm`、`:451` 15px→`text-base`、`:488` 15px→`text-base`（「進行中任務」稿上無節點；⚖️ SM 裁定沿用 F11 的「生成佇列」規格 BodyLg 600）、`:80-83`／`:107-109` →`text-base`、`:182` →`text-base font-semibold`、`:374` →`font-medium`。⛔ **`:74`／`:101`／`:119`／`:247` 四處 `text-[11px]` 原樣保留**。
   - **顏色**：`:272` 金額 `--warning-text` → **`--text-primary`**（DESIGN.md:308）；~~`:276`／`:280` 的兩個**計數**維持 `--warning-text`~~ → ⚠️ **這句與 AC #1 矛盾，以 AC #1 為準**：⚖️ Q1 說「每一列『已暫停—下次繼續』與金額、**數字**一律中性」，所以橫幅裡的兩個計數也轉 `--text-primary`（`.pen` 的 `m2pTFN`／`bOyrF` 同步改了）；赭色留給橫幅底、圖示與狀態膠囊。`:354` 的 `Captions` 圖示**整顆移除**（稿上沒有）。
   - **間距**：ov-track `w-44`→`w-[180px]`（`:87`）、sse-chip `py-1`→`py-1.5`（`:119`）、gen-queue `gap-2.5`→`gap-3`（`:449`）、列 `gap-3`→`gap-2.5`（`:175`）、按鈕 `px-4`→`px-5`（`:379`／`:407`／`:458`／`:531`）、p1-card `py-14`→`py-6`（`:390`）。
   - **`CancelAll` 補上母版底色**（`:531` 目前是 `border` 無底）：`bg-[var(--bg-tertiary)]` ＋ `text-[var(--text-primary)]`（`YDPhc`）。`:379` 的重試按鈕移除 `RotateCcw` 圖示（母版沒有）。

7. **既有的誠實行為不准回歸**（測試都要留著）：
   - **逐列沒有暫停／重試按鈕**——後端只有五條路由（`generation_batch_handler.go:61-69`），沒有任何 per-item 端點。`GenerationWorkspaceV2.spec.tsx:76` 的 `queryByRole('button', {name: /暫停|重試/})` **必須保留**。⚠️ 加了 dismiss／cancel 的確認列之後，那條 regex 可能誤中新的「重試」按鈕——**改成更精確的選擇器，不要刪掉這條測試**。
   - 「下次繼續」回同意流程、**不直接開始批次**（`:628` `onResume={onLaunch}`，sub-4-3 CR H1 的紅線）。
   - 工作區資料載入失敗是 inline fail-soft，**整頁絕不 hard-fail**（`:365-384`、稿上 `ZiBAb`）。
   - 右欄「僅狀態事件，不含逐字內容」（`:247`）與它的 11px 都不動。

8. **測試（先寫紅測試）。**
   - **`GenerationWorkspaceV2.spec.tsx` 既有 7 條**：
     - **改寫**：`:64`（`data-state` `'active'`→`'running'`、改用 `progress.items`）、`:112`（single `'active'`→`'running'`）、`:104`（attach 的三條斷言——`items[]` 現在一定有，降級卡只剩單部／舊後端兩種情況）。
     - **保留不動**：`:53`（idle）、`:79`（budget_ceiling 橫幅逐字）、`:128`（fail-soft）、`:138`（事件紀錄——那是 `dsr-6d-c-2` 的地盤）。
     - `:76` 的能力守門**保留但收緊選擇器**（見 AC #7）。
   - **`generationWorkspace.spec.ts`**：`:25`／`:38`／`:46` 三條的前提會變（`hasItems` 改看 `progress.items`）——**改寫而不是刪**，`deriveWorkspaceMode` 的 `attach` 語意要重新定義並在檔頭寫清楚。
   - **新增：容器測試**（`GenerationWorkspace`，`:547-635`，目前**零覆蓋**）。mock 需求：
     - `vi.mock('../../hooks/useDownloads')` 要 stub **`usePageVisibility`** —— ⚠️ 抄 `DownloadsBrowseV2.spec.tsx:25-29`（有），**不要抄 `DownloadPanel.spec.tsx:14-16`（沒有，抄了會炸）**。
     - `subtitleService` 工廠要有 `getGenerationBatchStatus`／`previewGenerationBatch`／`cancelGenerationBatch`／**`dismissGenerationBatch`**。
     - hook 工廠要有 `attachSnapshot`／`connectionEpoch`（🚨 dsr-6d-b AC #8 踩過這一次：漏掉就 `attachSnapshot is not a function`，15 支測試全紅）。
     - 要蓋的情境（每一條都對應上面的 🔴）：批次終態後 `removeQueries` 不再讓佇列消失（#1）；被拒的那一部顯示失敗＋原因（#2、#3）；逐部訂閱跟著 `currentMediaId` 換（#8）；`refetch` 不會把 SSE 狀態洗回 running（#9）；對話框啟動後工作區看得到（#10，用 invalidate 模擬）；分頁切回會重查而不是吃快取 running（#11）；冷接上碰到上限時「下次繼續」存在（#12）；取消失敗顯示 alert 且不吞（#13）；「關閉」呼叫 dismiss、失敗顯示 alert、批次仍在跑時不讓關（#14）；SSE 膠囊終態消失（#15）；缺字幕數用含分集的欄位（#17）；終態 invalidate 五組 key 且同一個 batchId 只做一次（#18）。
   - **e2e 新增**（目前工作區**零 e2e**）：一支 `tests/e2e/generation-workspace.spec.ts`，describe 標籤 `@ui @generation-workspace`，最小覆蓋＝`/activity?view=generation` 進得去 ＋ status mock 回 `{running:true, progress:{…items}}` 時佇列列畫得出來 ＋ 回 `{running:false, last:{…}}` 時終態與「關閉」畫得出來。⛔ **不要用 `--grep @story-*`**（那種標籤不存在，grep 會 0 個測試卻回報成功——dsr-6d-b 踩過）。
   - **`ActivityHub.spec.tsx`**：`:16-23` 的工廠補 `generationBatchStatusKey`（AC #3 新增的 export），`:19` 的註解改成實話。

9. **視覺夾具與基準線。**
   - 三個既有夾具（`-gallery.fixtures.tsx:5076`／`:5124`／`:5158`）：`progress.items` 補齊（**狀態要與計數一致**——🔴 #28 的自相矛盾一併修掉：`budget_ceiling` 改成 `successCount: 1 / pausedCount: 2 / totalItems: 3`，三列＝done／paused／paused，不要再寫 38）、`items[]` 補 `mediaType`＋`seriesTitle`（**至少一筆是分集**，用來驗劇名）、`activeItemProgress` 補 `partial`。
   - **新增夾具**：`generation-workspace-v2/complete-with-failures`（唯一會畫失敗列與失敗原因的狀態，且底部列的「關閉」也在這張裡）。⚠️ 建單時另外點名的 `terminal-close` **沒有單獨建**——`budget_ceiling` 夾具加上 `onDismiss` 之後已經涵蓋「關閉＋下次繼續」的像素，再開一張只是重複。
   - 基準線流程照 `project_visual_baseline_intentional_change.md`：`--update-snapshots` 後只留上列、其餘還原（⚠️ `parse-floating-parse-progress-card` 與 `retry-retry-notifications` 在**本機**一定會漂，那是本機資料庫的側欄數字，**乾淨工作樹同樣漂**，還原即可）；改過的 `-linux` 一律 `git rm` 交給 CI bootstrap。⛔ 不要本機產 `-linux.png`。
   - ⚠️ 夾具的 `props` 是 `Record<string, unknown>`，typecheck 抓不到漏欄位——改完一定要看圖確認佇列有畫出來。

10. **另立／交代的單子（建單時寫入 sprint-status，不在本張做）。**
    - `dsr-6d-c-2-workspace-event-log`（**本張同時建立**）：右欄「即時活動」的全部問題——不聽 `subtitle_progress`（抽內嵌字幕的項目完全不出現，`useGenerationJobsFeed.ts:90-94` 的 `TRANSCRIPTION_EVENTS` 沒有它）、每個翻譯百分比都新增一列且 `aria-live` 逐條唸、沒有項目標題、失敗顯示後端英文錯誤、事件圖示與 `·` 分隔點沒實作、Pane Header 的 `activity` 圖示沒實作、F12 的 log-footer「已停止（達預算上限）」沒實作、以及 `disc-2026-09-batch-sse-items-transform-cost` 的工作區那一半。
    - `disc-2026-09-batch-sse-items-transform-cost`（既有）：**確切位置已查到**——`useGenerationJobsFeed.ts:237` 的 `snakeToCamel<BatchPayload>`，而 `BatchPayload`（`:82-86`）只有三個欄位。終態事件帶整份 `items` 時，全選 2,400 部會配置 2,400 個新物件、14,400 次鍵重寫，然後立刻變垃圾。⚠️ **工作區同時掛兩個 hook**（`:551`／`:552`），兩條 EventSource 都聽 `generation_batch_progress` → **同一個事件被轉換兩次**；對話框同時開著再多兩條（共 4 條）。歸 `dsr-6d-c-2`。
    - `disc-2026-09-batch-dialog-last-attach-per-mount`（既有，dsr-6d-b 立）：本張的 F12「關閉→dismiss」是它的根治法。⚠️ 但**順序相反時仍有洞**：對話框先掛載接上 `last`，工作區才 dismiss → 對話框顯示一份伺服器已經忘掉的快照。本張不處理，在那張單子上補記。
    - 既有：`disc-2026-09-activity-batch-row-count-in-flight-index`、`disc-2026-09-dialog-close-target-44px`、`disc-2026-09-dialogframe-shadow-vs-shadow-xl`、`9R-17-ai-usage-endpoint`。

11. **CI 全綠**：`pnpm run lint:all`、`pnpm nx run web:typecheck --skip-nx-cache`、`python3 scripts/check-design-tokens.py`、`pnpm nx test web`、`pnpm nx test api`（本張不改後端，但全套閘門照跑）。⛔ 絕不用 `run_in_background` 跑測試；每次跑完 `pnpm run test:cleanup`。Nx daemon 建圖逾時就用 `NX_DAEMON=false` 重跑並記在 Completion Notes。⚠️ 本機跑 e2e 需要 `AI_PROVIDER=claude`（`.env` 預設 gemini 但沒金鑰，後端起不來）。

## Tasks / Subtasks

- [x] **Task 1 — 設計稿（AC: #1）**
  - [x] F12 列與數字中性、`$warning`→`$warning-text`、確認示例數字、`ZMYbU` 存在才刪
  - [x] 三則過期註記（`K5gn5g` 改寫、`DP53I` 刪、`B5D2a5` 改寫）、翻譯階段文案一致
  - [x] `ctx.problems` 基線 0 → 改完仍 0；存檔並 grep 落盤；匯出後只 stage f11／f12／f13（＋pen-tokens）
- [x] **Task 2 — 列改讀 `items[]`（AC: #2, #8 前半）**
  - [x] 先寫紅測試：被拒的那一部顯示失敗＋原因、劇名、`data-state` 用後端詞彙
  - [x] 刪 `QueueRowLabel`／`deriveRowStates`／`failedIds`／`generationBatchItemsKey` query，改用 `generationQueueRow.ts`
- [x] **Task 3 — 探測三條路、`last`、dismiss、key 匯出（AC: #3, #8）**
  - [x] 先寫紅測試：終態後佇列還在、`last` 只接一次、關閉呼叫 dismiss、批次仍在跑時不讓關
  - [x] `generationBatchStatusKey` 匯出＋對話框 invalidate＋`ActivityHub.spec` 工廠補 key
- [x] **Task 4 — 訂閱跟著換片、切分頁、洗狀態（AC: #3 後半, #8）**
  - [x] 先寫紅測試：`currentMediaId` 換了會重新訂閱、`refetch` 不會洗回 running、切回來會重查
- [x] **Task 5 — 控制項誠實（AC: #4, #5, #7, #8）**
  - [x] 先寫紅測試：取消確認＋失敗 alert、SSE 膠囊終態消失、缺字幕數含分集、終態 invalidate 五組 key
  - [x] 總結文案有失敗不給綠、`TERMINAL_COPY` 補 `budget_ceiling`
- [x] **Task 6 — 母版、外框與字級（AC: #6）**
  - [x] `GenQueueRow-v2`（海報 40×60、BodyLg 600、狀態說明、藥丸徽章、六種圖示）、狀態膠囊、F12 底部列、F13 標題、字級與間距、金額轉中性、拿掉 `Captions`
- [x] **Task 7 — 夾具、基準線、e2e（AC: #8 後半, #9）**
- [x] **Task 8 — 收尾（AC: #10, #11）**
  - [x] 全套閘門；dev-story Step 9 截圖比對（`f11-d-v2`、`f12-d-v2`、`f13-d-v2`）

## Dev Notes

### 這張的重點

- **最急的是 🔴 #1。** 批次一結束工作區主欄就整區空白，這是 dsr-6d-b 合併之後**現在就活在 main 上**的行為。其餘都是既有的舊帳。
- **真相已經在後端**（dsr-6d-a），對話框已經接完（dsr-6d-b）。本張的價值是「工作區不再自己猜」，而且**兩個畫面不再對同一部片講相反的話**。
- **`last` 是這條線的重點**，而工作區是唯一有資格呼叫 `dismiss` 的畫面——F12 的「關閉」就是「我看過了，忘掉它」。

### 上游契約（Rule 20 ack）

- confirmed against [@contract-v1] (Story dsr-6d-a AC #1) —— `items[]` 每筆 `{media_id,title,media_type,series_title,status,reason}`，六個鍵恆存在；`reason` 只在 `failed` 有值。
- confirmed against [@contract-v1] (Story dsr-6d-a AC #2) —— `generation_batch_progress` 每次 13 個鍵：執行中 `items: null` ＋ `changed_item`；終態 `items` 完整。
- confirmed against [@contract-v1] (Story dsr-6d-a AC #3) —— `GET …/status` 是 `{running, progress, last}`；`last` 是終態快照，跑的時候一定是 `null`。
- confirmed against [@contract-v1] (Story dsr-6d-a AC #6) —— `POST …/dismiss` 回 `{dismissed, running}`；批次執行中是 no-op 且 `running:true`。
- confirmed against [@contract-v1] (Story dsr-6d-a AC #7) —— `series_title` 一定出現，電影為 `""`。
- confirmed against [@contract-v2] (Story 9R-16 AC #3) —— `preview` 的 `total_items_including_episodes` 是 sub-5-1 AC #7 加上去的**可選**加鍵（純加寬、`total_items` 語意凍結為 movies-only，依 retro-19-P3 的 widening-no-bump 不升版），舊伺服器沒有 → 退回 `total_items`。⚠️ **建單時這條寫成「[@contract-v3] (Story sub-5-1 AC #7)」是錯的**——sub-5-1 AC #7 本身沒有 stamp，它 ack 的是 9R-16 AC #3 的 `[@contract-v2]`；dev-story Step 2 的 Contract Stamp Check 抓到並更正。

### 建單裁定（2026-09-18，Sally／Alexyu 可在 review 推翻）

1. **`dsr-6d-c` 拆成兩張**（版面切分，見 Context）。
2. 列的文案**一律走 `generationQueueRow.ts`**；需要新詞就改那個檔案並補它的 spec，讓對話框一起受惠——⛔ 不准在工作區再寫第二套。
3. F12 的赭色範圍照 ⚖️ Alexyu Q1：**批次層提示（橫幅、狀態膠囊、紀錄那一行）赭色；每一列、金額、數字中性**。程式碼 `:272` 的金額違規要修，`:276`／`:280` 的計數維持赭色。
4. 「進行中任務」（`:488`，稿上無節點）沿用 F11「生成佇列」的規格：BodyLg 600。
5. `X8KfD` 這類**規格註記**可以維持赭底（2026-09-11 的「不准赭色」針對畫面內容，不是註記）。
6. 終態的片庫刷新工作區自己也做一次，**允許與對話框重複**（invalidate 冪等）。

### 不要做的事

- **不要碰右欄 `EventLogPane`／`useGenerationJobsFeed.ts`**（`dsr-6d-c-2`），唯一例外是 AC #4 的膠囊誠實性只動 `OverallStrip` 那顆。
- **不要刪 `GenerationBatchDialogV2.tsx` 的任何 export**（`ActivityHub.spec.tsx` 的工廠與對話框退回路徑仍依賴）。
- **不要改 `Component/GenerationProgress-v2` `XkGvG` 母版**（dsr-6b 才對齊過，對話框與工作區共用）。
- **不要改 `ui/Dialog.tsx`**、**不要把 `text-[11px]` 改成 12**、**不要做手機版**（F11-M `PXB0z` 歸 `dsr-6f`）、**不要改後端**。
- **不要在本機產 `-linux.png`**。

### 已知陷阱

- **`removeQueries` 對 `enabled:false` observer 的精確行為**沒有實測（稽核代理標為 SUSPECTED）：讀碼結論是 query 被移除、observer 掛上 `data: undefined` 的新 entry、`queryFn` 永不執行所以資料回不來。**開工第一件事就是寫一條紅測試把它釘死**，不要憑推論改。
- **兩個 hook 兩條 EventSource**（`:551`／`:552`），對話框再兩條。改動連線時序時要意識到這件事。
- **`useGenerationProgress` 有 mediaId 過濾**（`useGenerationProgress.ts:229`），`startTracking(mediaId)` 才寫 `mediaIdRef`（`:337-342`）。
- **`START` reducer 會洗掉狀態並強制 running**（`useGenerationBatchProgress.ts:117`）——這就是為什麼 `last` 一定要用 `attachSnapshot`。
- **`ScanProgress.spec.tsx:20-24` 很脆**：它用單一方法 stub 整個 `subtitleService`。⛔ 不要讓 `ScanProgress` 的模組圖多拉進任何 `subtitleService` 成員。
- **`ActivityHub` 在工作區底下仍每 15 秒打 `/api/v1/activity`**（`useActivity()` 在 `:296`，早於 `view === 'generation'` 的 early return），結果被丟掉。本張不處理，但 AC #2 的 single 標題會用到同一份資料——**直接用那個既有查詢，不要再開一個**。
- **`toHaveTextContent` 是子字串比對**——文案要逐字；class 也要逐字。
- **視覺 CI 沒有後端**：夾具全靠 props，缺欄位不會被 typecheck 抓到。
- **行號以建單時為準**（2026-09-18，main `3a520961`）。

### Source tree

```
ux-design.pen（F12 `iH98f` 顏色/註記、F11 `l8FsB` 註記、F13 `F7ohe` 註記）      ← Task 1
apps/web/src/components/subtitle/GenerationWorkspaceV2.tsx(+spec)              ← Task 2–6
apps/web/src/components/subtitle/generationWorkspace.ts(+spec)                 ← Task 2
apps/web/src/components/subtitle/generationQueueRow.ts(+spec)                  ← Task 2（若需新文案）
apps/web/src/components/subtitle/GenerationBatchDialogV2.tsx                   ← Task 3（只加 key 匯出＋兩處 invalidate）
apps/web/src/components/activity/ActivityHub.spec.tsx                          ← Task 3（mock 工廠補 key）
apps/web/src/routes/test/-gallery.fixtures.tsx                                 ← Task 7
tests/e2e/generation-workspace.spec.ts（新）                                    ← Task 7
tests/visual/.../components/generation-workspace-v2/**                          ← Task 7
_bmad-output/pen-tokens.json、_bmad-output/screenshots/flow-f-subtitle-v2/{f11-d-v2,f12-d-v2,f13-d-v2}.png ← Task 1
apps/web/src/hooks/useGenerationJobsFeed.ts                                     ← 只讀（歸 dsr-6d-c-2）
```

### Cross-Stack Split Check

後端 task **0 個**，前端／設計 task 8 個。→ 跨棧門檻（兩邊都 >3）**不觸發**。⚠️ 但規模門檻觸發了：`dsr-6d-c` 已依 SM 裁定拆成 `-1`／`-2`（見 Context）。

### Time-dependent visual coverage

- **N/A — no wall-clock-reading components touched.** `GenerationWorkspaceV2` 與 `generationWorkspace.ts` 都不讀時鐘（建單稽核確認：全檔沒有 `Date.now()`／`new Date()`），本張也不加已用時間／ETA（dsr-6b 已裁定不顯示系統給不出的時間）。

### References

- [Source: `apps/web/src/components/subtitle/GenerationWorkspaceV2.tsx:1, 30-52, 71-120, 126-199, 205-251, 259-293, 317-345, 427-470, 488-513, 525-536, 547-635`]
- [Source: `apps/web/src/components/subtitle/generationWorkspace.ts:46-55`、`generationQueueRow.ts:69-103`、`GenerationBatchDialogV2.tsx:60-68, 78-107, 251, 371-387, 627-690, 826-863, 867-872, 880-892, 899-927, 957, 979, 1058-1066`]
- [Source: `apps/web/src/hooks/useGenerationBatchProgress.ts:117, 206, 214, 216, 243`、`useGenerationProgress.ts:229, 337-342`、`useGenerationJobsFeed.ts:82-86, 90-94, 237, 249, 252-258`、`useActivity.ts:31-37`、`useDownloads.ts:35-37`、`queryClient.ts:5-12`]
- [Source: `apps/web/src/services/subtitleService.ts:452-456, 470-477, 641-648`、`components/scanner/ScanProgress.tsx:14, 57-68`]
- [Source: `apps/web/src/components/activity/ActivityHub.tsx:29-30, 140-152, 296, 308-317, 331-339, 357-358`、`routes/activity.tsx:12-19`]
- [Source: `apps/web/src/components/subtitle/GenerationWorkspaceV2.spec.tsx:4, 53, 64-76, 104-109, 112-125, 128, 138`、`generationWorkspace.spec.ts:25, 38, 46`、`ActivityHub.spec.tsx:16-23, 27-31`、`DownloadsBrowseV2.spec.tsx:25-29`、`ScanProgress.spec.tsx:20-24`]
- [Source: `apps/web/src/routes/test/-gallery.fixtures.tsx:204-205, 5076-5187`、`tests/visual/components.visual.spec.ts-snapshots/components/generation-workspace-v2/**`]
- [Source: `apps/api/internal/handlers/generation_batch_handler.go:61-69, 189-207, 217-224, 233-247, 257-262, 277-301`、`internal/services/generation_batch.go:65-91, 100-113, 133-137, 295-302, 321-333, 354`]
- [Source: `ux-design.pen` `l8FsB`／`iH98f`／`F7ohe`／`aw4Qr`／`XkGvG`／`YDPhc`／`otvKh`／`NtMLG`／`c4FIoB`／`JzmvC` — 唯讀稽核代理以 Pencil MCP 逐節點讀出（2026-09-18）]
- [Source: `DESIGN.md:289-300`（固定詞彙）、`:308-320`（金錢是事實）、`:390-394`（字階）、`:510-516`（Motion）、`:560`（無 `--radius-pill`）、§Badges and Pills]
- [Source: `_bmad-output/implementation-artifacts/dsr-6d-a-batch-status-backend.md`、`dsr-6d-b-batch-dialog.md`（列語彙、`attachSnapshot`、`last` 只接一次、CR H1/H2/M4 的教訓）]
- [Source: project-context.md#Rule 5 / #Rule 13 / #Rule 16 / #Rule 20 / #Rule 21 / #Rule 23 / #Rule 24]

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

- 🔗 **AC Drift: FOUND** —— 本張改變了 `ux3-ai-2-workspace-frontend.md` 三條 AC 的可觀察行為：**AC #2**（「one row-state authority」＝`deriveRowStates`）→ 權威改成後端 `progress.items[]`；**AC #3**（attach-degraded 卡片）→ `items[]` 現在恆存在，降級路徑縮到「單部任務或舊版後端」；**AC #6**（閒置的缺字幕數）→ 改讀 `total_items_including_episodes`。grep：`GenerationWorkspaceV2|deriveWorkspaceMode|workspace-queue-row` 掃 `_bmad-output/implementation-artifacts/*.md`，8 個檔案命中，其餘 5 個（`sub-4-3`／`sub-5-1`／`sub-5-3`／`dsr-6b`／`dsr-6d-b`／`bugfix-e`）都是 REUSE 不是 DRIFT。⚠️ **`ux3-ai-2` 的 AC 全部未 stamp**（該檔 5 個 `[@contract-v*]` 全在 Dev Notes 的 ack，不在 AC 上）→ 依 Rule 20 forward-only retrofit 視為 implicit v0，無 ack 義務，但漂移已依 retro-10-AI2 記錄於此。
- 📎 **Contract Stamps: FOUND（本張 0 個自有 stamp、6 條上游 ack）** —— 5 條對 `dsr-6d-a` `[@contract-v1]`（AC #1/#2/#3/#6/#7），1 條對 `9R-16` `[@contract-v2]`（AC #3）。⚠️ **建單時第 6 條寫成「`[@contract-v3]` (Story sub-5-1 AC #7)」是錯的**：`sub-5-1` AC #7 本身沒有 stamp，它 ack 的是 `9R-16` AC #3 的 `[@contract-v2]`（純加寬、`total_items` 語意凍結、依 retro-19-P3 widening-no-bump 不升版）。Step 2 的 Contract Stamp Check 抓到並已更正 Dev Notes。本張沒有產生任何 bump → 無下游 stale-mark 義務。
- **Task 1（設計稿）**：F12 三列（`FU91l`／`F5VI9N`／`jryaP`）的 `lUZol`／`OLg1J`／`CQhw2` 改 `$text-muted`、`IcPTo` 改 `$bg-tertiary`；橫幅兩個計數 `m2pTFN`／`bOyrF` 改 `$text-primary`（⚖️ Alexyu Q1：批次層提示赭色，列與數字中性）。`$warning` 當文字／圖示色的四處（`Zeu0P`／`F0IlN`／`ShmdC`／`gavvw`）改 `$warning-text`。`ZMYbU`「全部取消」確認存在後刪除。過期註記：`K5gn5g` 改寫（不再說 F11 是探索性變體）、`B5D2a5` 改寫（`items[]` 已上線）、`DP53I` 整個 frame 刪除。
  - ⚠️ **對 AC #1 的一處刻意偏離**：AC 寫「改 `lUZol` 的文案為『翻譯中…』」。實作時先 `Get` 了母版 `XkGvG`，確認它的**預設 active 步驟就是「轉錄中」**（`QtmaR` 是 `$accent-text`，`gKW5B`「翻譯中」是 `$text-muted`），所以那一列本來就在轉錄階段——說謊的是 **45%**，不是文案。改用 **dsr-6d-b 對 F8 的同一個做法**：把 `OMD2y` 與巢狀步驟條的 `fVBQF` 都 `enabled:false`。滿足 AC 的意圖（「轉錄百分比只在翻譯階段」）而且改動更小、與 F8 一致。
  - ⚠️ **Pencil API 眉角**：`Update(instance,{descendants:{dxgox:…}})` 會失敗（`Node not found for override path: dxgox`）——當一個 slot 已經被 override 成巢狀 instance 時，要**直接對那個巢狀 instance 的 id 下 `Update`**（`gkGTR`／`Drmyd`），不能再走母版的 slot 名。
  - `ctx.problems`：全檔 70（與基線相同），`l8FsB`／`iH98f`／`F7ohe`／`aw4Qr` 四個節點底下**各 0 筆**。存檔後 sha256 由 `9a7b8800` → `0b34baff`，`git status` 顯示 ` M ux-design.pen`。匯出後只 stage `f11-d-v2`／`f12-d-v2`／`f13-d-v2`，`flow-i-discover-v2` 四張重繪雜訊已還原；**`f11-m-v2`、`f3-d-v2`、`f8-d-v2`、`design-system/` 都沒有變**（確認沒動到母版）。

- **Task 2–6（程式碼）**：佇列改讀 `progress.items[]`；工作區自寫的 `QueueRowLabel` 刪除，文案改用 `generationQueueRow.ts`；探測三條路＋`seenLastBatchIdRef`；F12 底部列「關閉」→ `dismissGenerationBatch()`（全 repo 第一個呼叫端，拒絕時丟例外讓面板顯示 alert）；`generationBatchStatusKey` 從對話框匯出＋兩處 invalidate；`staleTime: 0` ＋ `refetchOnWindowFocus: 'always'` ＋ `connectionEpoch` 重查；逐部訂閱獨立 effect；取消兩段式確認；終態 invalidate 五組 key（once per batchId）；母版 `GenQueueRow-v2` 實作；狀態膠囊；F13 標題；SSE 膠囊與 attach 佔位的誠實性；缺字幕數含分集；金額轉中性；拿掉 `Captions`；字級與間距。
  - ⚠️ **對 AC #2 的一處偏離（已立案）**：AC 寫「single 模式的列標題改用 activity API 的 transcription job 標題」。實作前查了型別：`SingleJobState` 只有 `{mediaId, phase, message, percentage}`，`ActiveJob` 只有 `{kind, percentDone, detail, current, total}`——**沒有 `mediaId` 可以 join**，那個來源不存在。退到 AC 自己寫的底線：顯示後端訊息，拿不到就「處理中的項目」，**絕不顯示 UUID**。缺口立案 `disc-2026-09-single-job-title-missing`。
  - ⚠️ **圖示不進共用模組**：`queueRowLabel` 回傳的是**文字與顏色**（共用真相），glyph 留給各自的畫面——F8/F9 與 `aw4Qr` 對同一個狀態畫的是不同圖示（`circle-alert` vs `triangle-alert`、排隊中有無 `hourglass`），硬塞成一套會逼對話框改設計。
  - **對抗式 mutation check（8 項修法逐一拿掉 → 22 條測試變紅）**：`last` 不接、逐部 effect 依賴退回、`staleTime` 退回、dismiss 結果忽略、缺字幕數退回 movies-only、金額塗赭色、SSE 膠囊無條件、items-first 關掉。

**🔍 /ship 對抗式 CR（2026-09-18，fresh-context 代理，只讀）**——3 HIGH／11 MEDIUM／9 LOW，**修 14、交代 5**。

| 等級 | 問題 | 處置 |
| --- | --- | --- |
| **H1** | **切走分頁再回來，終態結果永久消失。** `!live` 會 `batch.reset()`；回來時 `seenLastBatchIdRef` 已經等於 `last.batchId`，於是不再 `attachSnapshot`，畫面變成「目前沒有進行中的生成」——使用者看不到哪幾部失敗，**而且「關閉」也不見了，`dismiss` 永遠叫不到**，後端的 `last` 就留在那裡讓對話框下次重播。這正是 🔴 #1 從另一扇門走回來，還跟單子標題直接矛盾。 | 修：那個 ref 改成只記「**使用者按過關閉**」（`dismissedLastBatchIdRef`），探測路徑不再寫它 |
| **H2** | **釘住「`last` 只接一次」的那條測試不可能失敗。** 它只 `rerender()`，而 effect 的依賴是 `[live, probeData]`，兩者都沒變，所以 effect 根本不會重跑——把守衛整個拿掉，15 條測試照樣全過。 | 修：改成兩條真的會失敗的（切分頁回來要重接／按過關閉就不再接） |
| **H3** | **終態之後才回來的探測會把批次重新畫成執行中。** `refetchOnWindowFocus:'always'` 在 T0 發出探測，T0+50ms 終態 SSE 到了（hook 變 `complete`、`closeSSE()`），T0+80ms 探測才回 `{running:true}` → `START` reducer 強制 `status:'running'` 並**重開一條再也不會有事件的連線**。🔴 #9 只修了一半。 | 修：記住已經終態的 batchId，晚到的探測直接忽略 |
| **M1** | **工作區又寫了第二套列文案**（正是建單裁定 #2 禁止的），而且其中一句是假的：`cancelled` → 「已取消，**未處理**」，但 `finish()` 連**正在跑**的那一部也標 `cancelled`，那一部可能已經付過錢了；`done` → 「已生成**繁中**字幕」對部分翻譯或簡體政策也不成立。 | 修：`queueRowBadge`／`queueRowSubStatus` 搬進 `generationQueueRow.ts`＋補 spec；兩句改成不過度宣稱 |
| **M2** | 終態那句話仍被 `rows.length > 0` 鎖住（「下次繼續」搬到底部列修好了，句子沒有）——冷接上的終態只有計數與「關閉」，沒有任何一句說批次是怎麼結束的。 | 修：標題列改由 `mode` 決定 |
| **M3** | 能力守門測試被「收緊」成兩條**不可能失敗**的斷言（`/^暫停$/`、`/重試失敗\|單項重試/`，畫面上根本沒有這些名字）。 | 修：改成「佇列 `<ul>` 裡一顆 button 都沒有」 |
| **M4** | 新加的 `aria-live` 是**第三個** live region，而且與內容一起掛載＝根本不會被唸（AC #5 明文禁止，dsr-6d-b CR M4 的教訓）。 | 修：拿掉 |
| **M5** | `CancelAll` 沒有抄對話框的焦點管理——按下「全部取消」後被聚焦的按鈕消失，焦點掉 `<body>`。 | 修：`cancelAllRef`／`keepGoingRef`／`pendingFocusRef` |
| **M7** | single 模式仍畫批次統計列：「已完成 **0 / 0** 部」「**$0.00**／上限 $0.00」外加一條 0% 進度條——憑空發明一個不存在的批次。 | 修：single 不畫統計列 |
| **M8** | attach 模式（批次在跑但還沒拿到佇列）**沒有任何取消入口**，而錢正在花。 | 修：同 M2，改由 `mode` 決定 |
| **M10** | 失敗那一列的步驟條仍會倒回「提取音訊」（`phase==='failed'` 被映成 `'idle'`，`failedPhase` 沒傳）。 | 修：直接傳 `phase`／`failedPhase`／`error` |
| **M11** | `live` prop 的註解宣稱「SSE 連線真的開著」，但它其實只看 `mode`——斷線重連的 10 秒空窗裡膠囊照亮。 | 修：註解改成實話 |
| L1 | 「關閉失敗，請再試一次。」但批次還在跑時再試也不會成功。 | 修：改成「批次還在進行中，現在無法關閉。」 |
| L5／L9／L4 | `generationWorkspace.spec.ts` 的測試名稱還在描述舊的 202 快取語意；AC #6 有一條與 AC #1 矛盾的顏色指示；AC #9 點名的 `terminal-close` 夾具其實被 `budget_ceiling`＋`onDismiss` 涵蓋了。 | 修：測試改名、AC 加刪節線註明以 AC #1 為準、夾具說明更正 |
| M6 | 關閉之後若 `singleJobs` 還留著過期項目，畫面會掉進假的「進行中任務」。根因是 feed hook 從不清 map（`jobs.stop()` 只關連線）。 | 交代 `dsr-6d-c-2`（M7 的修法已經拿掉那個畫面最誤導的 0/0 統計列） |
| M9 | 容器測試對 🔴 #1 只斷言 `attachSnapshot` 被呼叫，沒有真的渲染出列。 | 已被新增的 e2e（真瀏覽器）涵蓋，保留現狀並在此記錄 |
| L2／L3／L6／L7／L8 | 列 gap 14 vs 稿上 10（與 `sm:pl-[54px]` 自洽）；`busy` 靠子樹卸載清除；`dataError` 時仍顯示「目前沒有進行中的生成」；新 e2e 沒有限制 project；`terminalVerdict` 沒有 `budget_ceiling` 分支（橫幅已經在說了）。 | 不修，記錄在此 |

CR 後 mutation check：把上面每一條修法拿掉，**9 條測試變紅**（H1／H3／M1×2／M2／M4／M5／M7／M8 各有對應的回歸測試）。

- 🎭 **A11y Pre-Flight: PASS**（觸碰 3 個元件；jsx-a11y 對本張新增／修改的檔案 0 warning；既有 127 個 warning 屬 retro-11-AI1b，未擴大範圍）。逐項確認：取消／關閉的忙碌態一律 `aria-disabled` **不是** `disabled`（焦點不會掉到 `<body>`）；失敗訊息都是 `role="alert"`；終態總結那一句帶 `aria-live="polite"` 且**只有一個** live region（dsr-6d-b CR M4 的教訓）；進度條保留 `role="progressbar"`＋`aria-label`；所有按鈕 `min-h-[44px]`。本張沒有 modal、沒有 combobox、沒有 TMDb `<img>`。
- **Pre-existing failures**：無。跑滿全套前後 `web` 都是全綠。

### Discovery Triage

- **Did this story discover any work outside its current scope?** **YES** —— 建單當下處理如下。

- **① expand-scope-in-place**
  - 終態後佇列整區消失（🔴 #1）→ **AC #2、#3**
  - 列狀態用猜的／不傳 `failedIds`／自寫文案（🔴 #2–#4、#6）→ **AC #2**
  - `complete` 不看失敗數（🔴 #5）→ **AC #5**
  - `remaining` 用減法（🔴 #7）→ **AC #2**
  - 訂閱不跟著換片／洗狀態（🔴 #8、#9）→ **AC #3**
  - status key 沒匯出＋切分頁卡死（🔴 #10、#11）→ **AC #3**
  - 終態出口被 `items.length` 鎖住（🔴 #12）→ **AC #3**
  - 取消沒確認／dismiss 零呼叫端（🔴 #13、#14）→ **AC #3、#4**
  - SSE 膠囊與 `animate-pulse` 說謊（🔴 #15、#16）→ **AC #4**
  - 缺字幕數欄位錯（🔴 #17）→ **AC #4**
  - 終態不刷新片庫（🔴 #18）→ **AC #4**
  - 母版沒實作／金額穿赭色／頁首圖示泥金／膠囊與底部列缺／F13 標題錯／字級間距（🔴 #19–#26）→ **AC #1、#6**
  - 容器零測試、e2e 零覆蓋、夾具與基準線（🔴 #27–#29）→ **AC #8、#9**

- **② spawn-blocking-story**：無（依賴的 `dsr-6d-a`／`dsr-6d-b` 都已合併）。

- **③ backlog-with-carry-forward-link**（建單時立／補記）
  - `dsr-6d-c-2-workspace-event-log` — 右欄「即時活動」的全部問題（**本張建單時同時建立**）
  - `disc-2026-09-batch-sse-items-transform-cost` — 確切位置查到了（`useGenerationJobsFeed.ts:237`），且**同一事件在工作區被轉換兩次**（兩個 hook 各一條連線）→ 歸 `dsr-6d-c-2`
  - `disc-2026-09-batch-dialog-last-attach-per-mount` — 本張的 dismiss 是根治法，但**對話框先接上、工作區後 dismiss** 的順序仍有洞 → 在該條目補記

- Reference: `project-context.md` Rule 24

### File List

**新增**

- `apps/web/src/components/subtitle/GenerationWorkspaceContainer.spec.tsx`（容器層 15 條，之前**零覆蓋**）
- `tests/e2e/generation-workspace.spec.ts`（`@ui @generation-workspace`，之前**零 e2e**）
- `tests/visual/.../components/generation-workspace-v2/complete-with-failures/default-visual-darwin.png`

**修改**

- `apps/web/src/components/subtitle/GenerationWorkspaceV2.tsx`（重寫）
- `apps/web/src/components/subtitle/GenerationWorkspaceV2.spec.tsx`
- `apps/web/src/components/subtitle/generationWorkspace.ts`（只改 JSDoc：`hasItems` 現在指後端佇列）
- `apps/web/src/components/subtitle/GenerationBatchDialogV2.tsx`（＋`generationBatchStatusKey` 匯出、啟動後與終態各一次 invalidate）
- `apps/web/src/components/activity/ActivityHub.spec.tsx`（mock 工廠補 `generationBatchStatusKey`）
- `apps/web/src/routes/test/-gallery.fixtures.tsx`
- `tests/visual/.../generation-workspace-v2/{running,budget_ceiling,idle}/default-visual-darwin.png`
- `ux-design.pen`、`_bmad-output/pen-tokens.json`、`_bmad-output/screenshots/flow-f-subtitle-v2/{f11-d-v2,f12-d-v2,f13-d-v2}.png`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`
- `_bmad-output/implementation-artifacts/ux3-ai-2-workspace-frontend.md`（AC drift reference — see Completion Notes）

**刪除**

- `tests/visual/.../generation-workspace-v2/{running,budget_ceiling,idle}/default-visual-linux.png`（過期基準；交給 CI bootstrap）

⛔ **沒有碰**：`useGenerationJobsFeed.ts`、`EventLogPane` 的內容（`dsr-6d-c-2`）、任何後端檔案、`XkGvG` 母版、手機稿。

## Change Log

| 日期 | 內容 |
| --- | --- |
| 2026-09-18 | 🚧 **REVIEW**。Task 1–8 全數完成。閘門：lint 0 errors、typecheck、design-tokens、**web 3771/3771（+32）**、api 全綠、**新增 e2e `@generation-workspace` 真瀏覽器 3/3 綠**。8 項修法做過 mutation check（拿掉 → 22 條紅）。視覺：三張重生、新增 `complete-with-failures`、過期 `-linux` 已 `git rm`。新立 `disc-2026-09-single-job-title-missing`。 |
| 2026-09-18 | Task 2–7（AC #2–#9）：佇列 items-first、狀態探測三條路＋`last`、`dismiss` 第一個呼叫端、status key 匯出＋對話框 invalidate、逐部訂閱跟著換片、取消兩段式確認、終態五組 invalidate、母版 `GenQueueRow-v2` 實作、金額轉中性、SSE 膠囊誠實、缺字幕數含分集、夾具與基準線。 |
| 2026-09-18 | Task 1（AC #1）設計稿：F12 列與數字轉中性（Q1 裁定）、四處 `$warning`→`$warning-text`、刪 `ZMYbU`「全部取消」與過期註記 `DP53I`、改寫 `K5gn5g`／`B5D2a5`、45% 依 dsr-6d-b F8 先例關掉（而非改文案）。裁切 0／0，母版未動。 |
| 2026-09-18 | Story 建立（SM Bob, create-story；main `3a520961`）。⚖️ **`dsr-6d-c` 拆成兩張**（`-1` 主欄／`-2` 右欄事件紀錄），切分線在版面而非「先行為後外觀」——右欄有自己的 hook、自己的 SSE 連線、自己的一整串 bug，且兩邊檔案幾乎不重疊。三個唯讀稽核代理（程式碼／設計稿／接線）在工作區找到 **29 條**：最急的是 dsr-6d-b 的 `removeQueries` 讓批次一結束主欄整區空白；其次是「兩個畫面對同一部片講相反的話」、status query key 沒匯出所以沒人能讓它失效、切走分頁再回來卡在假的「進行中」五分鐘、以及一個程式碼完全沒實作的母版 `Component/GenQueueRow-v2`。另外抓到兩條設計紅線違規：金額穿赭色（`:272`，DESIGN.md:308）與頁首圖示塗泥金（`:354`，連 idle 都在宣稱有工作在跑）。六項 SM 裁定。 |
