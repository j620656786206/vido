# Story disc-2026-10-episode-row-live-progress-a：季清單每一集的「生成中」轉圈圖示即時出現、即時消失

Status: done

**Source:** sprint-status `disc-2026-10-episode-row-live-progress-a`（傘狀 `disc-2026-10-episode-row-live-progress`）。2026-10-28 Alexyu 回報：特種部隊：母獅 S01 八集一集一集從「管理字幕」按「生成字幕」，季清單每一列都還是 ✓，要點開「管理字幕」才看得到進度。⚖️ Alexyu 2026-10-28 選 A：Pencil 額度用完前不加新畫面，只讓 J11 既有的轉圈圖示即時出現與消失。列上的進度文字、段數、金額、「排隊中」是 `-b`（等設計稿 J13-D）。

## 查到的事（2026-10-29，Bob 逐一開檔確認）

### Winston 的推論「staleTime 1 小時、只在結束時重抓」→ **成立，而且比推論更糟**

- 季清單的集數查詢 `useSeasonEpisodes`：`staleTime: 60 * 60 * 1000`（`apps/web/src/hooks/useMediaDetails.ts:56`）。全 repo grep `season-episodes`／`seasonEpisodes`：會讓它失效的只有 `useLibrary.ts:114`（刪除／重新整理媒體）、`useManualSearch.ts:62`（手動選片）、`GenerationWorkspaceV2.tsx:1304` 與 `GenerationBatchDialogV2.tsx:1178`（**批次**結束時 `detailKeys.all`）。單集生成**開始**時沒有任何人重抓。
- 單集生成**結束**時的重抓只有對話框的 `onGenerationComplete`／`onGenerationFailed` → `refetch()`（`apps/web/src/components/media/SeasonAccordion.tsx:156-157`）。但對話框關掉就卸載（`SeasonAccordion.tsx:144` 只在有 `openedEpisode` 時渲染，`:155` 關閉時設回 null），`useGenerationProgress` 卸載時關掉 SSE（`apps/web/src/hooks/useGenerationProgress.ts:564-571`），之後**結束事件沒人聽**。Alexyu 的用法（按完生成就關掉、換下一集）→ 開始不重抓、結束也不重抓，列停在 1 小時前的狀態。
- 列的轉圈圖示早就做好：`IN_FLIGHT = searching|probing|extracting|translating`（`apps/web/src/components/media/EpisodeList.tsx:176`），有 `chineseSubtitle` 時在飛行中蓋過結論，畫 J11 的「字幕生成中／打開「管理字幕」看進度」＋ `Loader2 animate-spin`（`EpisodeList.tsx:194-202`）。設計稿 `_bmad-output/screenshots/flow-j-specs/j11-d.png`（S01E03 新的血脈那列）就是這個。**本單不改 EpisodeList 的畫法。**

### 後端在 pipeline 模式單集生成時寫進 `episodes.subtitle_status` 的值

- 點擊走 `SoloRunner.Start`：claim、預測路線、寫入探測、`emitStart`（送 `transcription_extracting`），**回 202 之後**才在 goroutine 裡跑 `ProcessItem`（`apps/api/internal/subtitle/solo_runner.go:240-301`）。所以 202 回來的瞬間，DB 可能還沒寫 `extracting`（競態，幾十毫秒）。
- `ProcessItem` 建 run 列、標 running 之後**立刻**把媒體列寫成 `extracting`（`apps/api/internal/subtitle/process_item.go:141-147`）——在進路由／抽軌閘門**之前**。所以：
  - **排隊等抽軌的集數 = `extracting`**（Winston 的疑問）。閘門在 `router.SelectAndRoute` 裡（`process_item.go:160-180`），等待只送 `subtitle_progress`「等待抽軌（前方 N 件）」，不改 DB。聽寫（ASR）路線的音訊抽取排隊也一樣：`transcribeFallback` 不改媒體列（`process_item.go:571-579` 註解：「the media row already reads `extracting`」），SoloRunner 只把排隊記在記憶體的 `job.QueuedAt`（`solo_runner.go:278-280, 345-357`）。
  - **正式機證據（唯讀）**：2026-10-28 21:35:29 起跑的 S01E08，log `audio extraction budget` 21:35:29、`audio extracted` 21:42:34——排了約 6 分鐘；那段時間它的列就是 `extracting`。八集的 `subtitle_runs` 都是 `route=asr`、`triggered_by=manual`、21:33～21:35 開始，21:38～21:47 依序完成（`ssh root@192.168.50.52 'sqlite3 -readonly /mnt/cache/appdata/vido/vido.db …'`）。
  - **進行中**：翻片內英文字幕那條在翻譯前改 `translating`（`process_item.go:425`）；聽寫那條整段維持 `extracting`，直到 `TranscriptionService` 寫結果（`apps/api/internal/services/transcription_service.go:1494-1500`）。
- **結束**：
  - 成功 → `found`（`process_item.go:337`；聽寫路線由 `writeSubtitleStatus` 寫 `found` 或 `untranslated`，`transcription_service.go:1478-1500`）。
  - 部分翻譯失敗或寫入前被拒 → `restoreMediaStatus` 還原成開始前的值（`process_item.go:334`、`:965-980`）。
  - 失敗 → `failItem` 寫 `not_searched`（`process_item.go:935`）。
  - 管線先寫 DB 再送 `subtitle_progress` complete/failed（`process_item.go:337-340`、`:935-941`），SoloRunner 在 `ProcessItem` 回來後才送 `transcription_complete`／`transcription_failed`（`solo_runner.go:288-299`, `:405-466`）——**收到這些事件時 DB 已經是最終值**，那時重抓一定拿到結果。

### 會不會誤導（排隊中用既有圖示）

- 有 `chineseSubtitle` 判定（正式機的後端都有）時，排隊中與跑中都是同一個「字幕生成中／打開「管理字幕」看進度」（`EpisodeList.tsx:194-202`）——排隊也是「生成中」的一部分，對話框裡看得到「等待抽軌（前方 N 件）」。**不誤導，不需要改畫面。**
- 只有在沒有 `chineseSubtitle` 的舊後端，才會退回 `statusMark` 的「抽取內嵌字幕中」（`EpisodeList.tsx:108-113`, `:251-255`），排隊時字面不精確——正式機不會走到，列入報告即可，不在本單改。

### SSE 事件（可以拿來觸發重抓的）

- 一條 `GET /api/v1/events` 廣播給所有連線（`docs/sse-event-types.md` Overview）。
- `subtitle_progress`：`media_id` = **集數 id**、`media_type: "episode"`、`stage`（probing／extracting／translating／complete／failed／skipped）（`apps/api/internal/subtitle/progress_sse.go:81-101`；文件 `docs/sse-event-types.md` subtitle_progress 一節）。
- `transcription_extracting`（開始）／`transcription_complete`／`transcription_failed`：`media_id`、`media_type`、`job_id`（`solo_runner.go:378-466`）。
- 事件只有集數 id，沒有劇集 id → 前端用「這一季已載入的 `episodeId`」比對。
- 既有 hooks 各自開一條 `EventSource`（`useGenerationJobsFeed.ts:614`、`useGenerationProgress.ts:457`、`useEnrichmentRefresh.ts`）。`useEnrichmentRefresh.ts` 的教訓（disc-2026-10-reparse-dialog-stuck-over-tunnel）：透過 Cloudflare 通道 SSE 可能一直沒打開，而且卡住時會拖住頁面其他回應——所以**連線 3 秒沒打開就關掉**，改靠輪詢。

### 重抓的成本

- `GetSeasonEpisodes`：讀劇集列、TMDb 季資料（快取 24h，`apps/api/internal/services/tmdb_service.go:38`）、本地集數、每個資料夾一次目錄讀取（`apps/api/internal/services/series_season.go:133-215`），沒有 ffprobe。15 秒一次在進行中時可以接受。

## Story

身為一次替一整季按好幾次「生成字幕」的人，
我希望關掉「管理字幕」回到季清單時，正在生成的那幾集就在轉圈、做完的那幾集自己變成結果，
這樣我不用一集一集打開對話框才知道哪集還在跑。

## Acceptance Criteria

1. **按下生成後很快出現轉圈：** 從某一集的「管理字幕」按「生成字幕」，後端接受（`started`）或回「已在跑」（409 `inProgress`）後——或打開「管理字幕」時發現這集已經在跑（CR M1 補）——，該季的集數查詢**立刻**重抓；在伺服器確認那一集進入飛行中狀態之前（最多 30 秒），每 2 秒重抓一次，確認後就停止快速重抓。對話框開著時就開始，不等關閉。
2. **跑的時候跟著 SSE 更新：** 只要這一季有任何集數是飛行中（`IN_FLIGHT` 四個值）或剛按下還沒被確認，就開一條 `/events` 連線；收到 `media_id` 屬於這一季已載入集數的 `subtitle_progress`（stage = complete／failed／skipped）、`transcription_complete`、`transcription_failed` → 重抓（同一時間多個事件合併成一次）。剛按下、還沒確認的那集，收到它 `probing` 之後的任何 `subtitle_progress`（extracting／translating）也重抓——那時 DB 已寫 `extracting`。**不**對 `transcription_extracting` 與 `probing` 重抓：SoloRunner 的開始事件與 probing frame 都在 DB 寫入之前送出（`solo_runner.go:396-403`、`process_item.go:140-147`），那時重抓只會拿到舊值（CR M2 修正 AC 文字，2026-10-29）。別季、別部片的事件不重抓。
3. **SSE 不可靠時的保底輪詢：** 有集數飛行中時，集數查詢每 15 秒重抓一次（SSE 漏事件、通道擋 SSE 都還是會更新）。SSE 3 秒內沒打開 → 關掉，只靠輪詢。
4. **沒有在跑就完全安靜：** 這一季沒有任何飛行中或待確認的集數時，不輪詢、不開 SSE；最後一集結束後連線關閉、輪詢停止。收合這一季（查詢 disabled）或離開頁面也一樣。
5. **結束時列自己變成最終狀態：** 一集做完／失敗，不重新整理頁面，該列就從轉圈變成後端寫的最終狀態（`found`／`untranslated`／`not_searched`／還原值，依 `chineseSubtitle` 規則畫）。
6. **離開再回來不是舊資料：** `useSeasonEpisodes` 的 staleTime 從 1 小時降到 30 秒（字幕狀態會變，不是「很少變」），重新展開或回到頁面時會拿新資料。
7. **不加新畫面：** EpisodeList 的圖示、文字、tooltip 不變；不加進度文字、段數、金額、「排隊中」（那是 `-b`）。電影詳細頁的標頭不在範圍內。
8. **測試（Vitest）：** 開始 → 重抓；本季集數的 SSE 終點事件 → 重抓、別人的事件 → 不重抓；只在飛行中輪詢、結束後停止；終點後列變成最終狀態。

## Tasks / Subtasks

- [x] Task 1：`useSeasonEpisodes` staleTime 與輪詢（AC #3, #4, #6）
  - [x] 1.1 staleTime 改 30 秒並寫明原因
  - [x] 1.2 輪詢間隔（飛行中 15s、待確認 2s、否則不輪詢）——**實作改由 hook 自己的計時器做**，不經 `useQuery` 的 `refetchInterval`：間隔取決於查詢自己的資料＋「待確認」狀態，放進 `useSeasonEpisodes` 會變成循環依賴；`useSeasonEpisodes` 簽章因此不變（見 Completion Notes）
- [x] Task 2：新 hook `useEpisodeGenerationLive`（AC #1–#4）
  - [x] 2.1 輸入：這季集數、refetch、enabled；輸出：`markStarted(episodeId)`
  - [x] 2.2 待確認集合（30 秒逾時；伺服器回飛行中或收到終點事件就移除）
  - [x] 2.3 飛行中或待確認時開 SSE（3 秒連線逾時、出錯就關），事件依 `media_id` 過濾，250ms 合併重抓
  - [x] 2.4 不在跑時關 SSE、停輪詢；分頁隱藏時跳過輪詢
- [x] Task 3：接線（AC #1, #5）
  - [x] 3.1 `ManageSubtitleDialogV2` 加選填 `onGenerationStarted`，在 trigger `onSuccess` 的 started／inProgress 呼叫（disabled、失敗不呼叫）
  - [x] 3.2 `SeasonAccordionItem` 用新 hook，把 `markStarted` 接到對話框
- [x] Task 4：測試（AC #8）
  - [x] 4.1 hook 測試（假 EventSource、fake timers）13 個
  - [x] 4.2 對話框：started／inProgress 會呼叫 `onGenerationStarted`，disabled／失敗不會（3 個）
  - [x] 4.3 SeasonAccordion：開始 → 立刻重抓；終點事件 → 重抓 → 列從轉圈變結果（2 個）
- [x] Task 5：驗證：`pnpm nx test web`、`pnpm run lint:all`、Prettier、typecheck、testid grep、對照 J11

## Dev Notes

### 不要做的事

- 不要改 `EpisodeList.tsx` 的畫法（AC #7）。
- 不要動後端：排隊中的值（`extracting`）配既有圖示不誤導（見「會不會誤導」）。
- 不要拿事件的 `stage` 直接改快取裡的 `subtitleStatus`（樂觀更新）：202 回來到 DB 寫 `extracting` 之間有競態，伺服器才是唯一來源；畫面只畫伺服器給的值。
- 不要每季開一條「永遠開著」的 SSE：沒有在跑時不開（AC #4），瀏覽器每個網域 HTTP/1.1 只有 6 條連線（區網是 `http://192.168.50.52:8088`）。
- 不要沿用 `useGenerationProgress` 的 SSE 重連迴圈：這裡有輪詢保底，出錯就關。

### 已知限制（報告用，不在本單修）

- legacy 模式（`VIDO_SUBTITLE_PIPELINE_MODE` 不是 pipeline）的單集生成：⚠️ 未查證它進行中會不會寫飛行中狀態；若不寫，待確認 30 秒到期後就停，結束時靠對話框（開著）或 staleTime 30 秒。正式機是 pipeline，不影響。
- 多季同時展開且都在跑 → 每季一條 SSE。實際上一次只會有一季在跑；若要合併成一條共用連線，是另一張。

### 架構對齊

- 查詢 key 不變：`detailKeys.seasonEpisodes(seriesId, seasonNumber)`（`useMediaDetails.ts:24-25`）。
- SSE URL：`${import.meta.env.VITE_API_BASE_URL || '/api/v1'}/events`（同 `useGenerationJobsFeed.ts:50`）；payload 解析用 `JSON.parse(e.data).data ?? parsed`（同 `useGenerationJobsFeed.ts:600-603`）。
- 檔頭標記：新 hook 是 utility → `// Implements: <utility — no .pen counterpart>`（同 `useEnrichmentRefresh.ts:1`）。

### Project Structure Notes

- 新檔：`apps/web/src/hooks/useEpisodeGenerationLive.ts` + `.spec.ts(x)`。
- 改：`useMediaDetails.ts`、`SeasonAccordion.tsx`、`ManageSubtitleDialogV2.tsx` 及其測試。

### Time-dependent visual coverage

- N/A — 不碰任何讀牆鐘的元件（hook 用 `setTimeout`／`Date.now()` 只算「待確認是否逾時」，不渲染時間；不在 `components/**`）。

### References

- [Source: apps/web/src/hooks/useMediaDetails.ts:52-59]
- [Source: apps/web/src/components/media/SeasonAccordion.tsx:62-66, 144-158]
- [Source: apps/web/src/components/media/EpisodeList.tsx:176, 194-202]
- [Source: apps/web/src/components/subtitle/ManageSubtitleDialogV2.tsx:417-455]
- [Source: apps/api/internal/subtitle/solo_runner.go:240-301, 378-466]
- [Source: apps/api/internal/subtitle/process_item.go:141-180, 334-340, 425, 935]
- [Source: docs/sse-event-types.md#subtitle_progress]
- [Design: _bmad-output/screenshots/flow-j-specs/j11-d.png]

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5 (claude-opus-5-5)

### Debug Log References

- `npx vitest run src/hooks/useEpisodeGenerationLive.spec.ts` → 13 passed（CR 後 14）
- `npx vitest run src/components/subtitle/ManageSubtitleDialogV2.spec.tsx -t "episode mode"` → 13 passed
- `npx vitest run src/components/media/SeasonAccordion.spec.tsx src/components/media/EpisodeList` → 63 passed
- `pnpm nx test web --skip-nx-cache`（前景）→ dev 時 299 files / 4941 tests；CR 修正後 299 files / **4943 tests passed**；`pgrep -fl vitest` 無殘留
- `pnpm run lint:all` → `✖ 170 problems (0 errors, 170 warnings)`（既有警告；本單改的 10 個檔案單獨跑 eslint 0 warnings）；Prettier 全過
- `pnpm nx run web:typecheck` → 成功
- 未動 Go → 未跑 `pnpm nx test api`／staticcheck

### Completion Notes List

- Ultimate context engine analysis completed - comprehensive developer guide created
- **查證結果**：Winston 的 staleTime 推論成立（`useMediaDetails.ts:56`），而且更糟——對話框關掉就卸載、SSE 關閉（`useGenerationProgress.ts:564-571`），結束時的 refetch 也沒人觸發。
- **後端狀態**：排隊等抽軌 = `extracting`（`process_item.go:145` 在進閘門前就寫）；ASR 路線整段 `extracting`；翻英文字幕路線翻譯時 `translating`；結束 `found`／`untranslated`／失敗 `not_searched`／部分失敗還原。正式機 log 證據：S01E08 排隊約 6 分鐘（21:35:29 → 21:42:34）。有 `chineseSubtitle` 時排隊與跑中同畫「字幕生成中／打開「管理字幕」看進度」，不誤導。
- **做法**：新 hook `useEpisodeGenerationLive` 只決定「什麼時候重抓」，從不改快取內容（伺服器是唯一來源）。按下生成 → 立刻重抓＋每 2 秒重抓直到伺服器確認（最多 30 秒，處理 202 早於 DB 寫入的競態）；有集數飛行中 → 開一條 `/events`，本季集數的終點事件 → 250ms 合併後重抓；另有 15 秒保底輪詢；SSE 3 秒沒開就關。沒在跑 → 什麼都不開。
- **Task 1.2 實作偏離**：輪詢放在 hook 內的 `setInterval`，不用 `useQuery.refetchInterval`。原因：間隔取決於查詢資料本身＋hook 的待確認狀態，傳回 `useSeasonEpisodes` 會變成「先有資料才知道間隔、先有間隔才能建查詢」。附帶好處：`useSeasonEpisodes` 簽章不變、既有 mock 不用改。
- 把 `IN_FLIGHT` 集合搬到 `utils/libraryStatus.ts` 的 `isSubtitleInFlight`，EpisodeList 與 hook 共用同一份清單（畫面行為不變）。
- 🔗 AC Drift: FOUND — `12-2-season-episode-list.md` Task 8.4「`staleTime: 60 * 60 * 1000`（1h — episodes change less frequently）」→ 30 秒（本單 AC #6）。那是 task 細節不是 AC；12-2 的 AC 不受影響。另 grep `onGenerationComplete`：9R-10c／dsr-6b 的結束重抓保留不變（REUSE）。
- 📎 Contract Stamps: NONE（本單不定義、不消費新的 wire 契約；只讀既有 SSE 欄位 `media_id`／`stage`，未改 payload）
- 🎭 A11y Pre-Flight: PASS（3 個元件檢查：EpisodeList／SeasonAccordion／ManageSubtitleDialogV2；touched files jsx-a11y 0 warnings；本單沒有新增或改變任何 DOM——轉圈圖示仍是既有 button＋aria-label＋Tooltip）
- 🎨 UX Verification: PASS — 本單不改畫面。對照 `flow-j-specs/j11-d.png` S01E03 列：轉圈＋「字幕生成中／打開「管理字幕」看進度」，由既有 `chineseMark`（`EpisodeList.tsx:191-200`）畫，SeasonAccordion 測試斷言了這兩個 aria-label（轉圈→「有繁中字幕，Vido 生成」）。**沒有實際啟動 app 跑一次真生成**（不做付費執行；本機沒有能跑的劇集資料），即時行為由 Vitest 驗證。
- testid：沒有新增、刪除、改名任何 data-testid → `tests/` 不受影響。

### Discovery Triage

- **③ backlog-with-carry-forward-link** — 季清單開著、什麼都沒在跑時，從別處開始的生成不會被發現；SSE 開啟後出錯不重連；多季同時展開各開一條連線 → `disc-2026-10-episode-row-live-progress-idle-detect`（sprint-status，名字回指本單）。
- **不立案（僅記錄）** — 沒有 `chineseSubtitle` 的舊後端，排隊中的列 tooltip 會寫「抽取內嵌字幕中」：正式機後端都有 `chineseSubtitle`，走不到。legacy 模式進行中狀態未查證：正式機是 pipeline，legacy 不在 -a 範圍（Dev Notes 已知限制）。

### File List

- `apps/web/src/hooks/useEpisodeGenerationLive.ts`（新）
- `apps/web/src/hooks/useEpisodeGenerationLive.spec.ts`（新）
- `apps/web/src/hooks/useMediaDetails.ts`
- `apps/web/src/components/media/SeasonAccordion.tsx`
- `apps/web/src/components/media/SeasonAccordion.spec.tsx`
- `apps/web/src/components/media/EpisodeList.tsx`
- `apps/web/src/components/subtitle/ManageSubtitleDialogV2.tsx`
- `apps/web/src/components/subtitle/ManageSubtitleDialogV2.spec.tsx`
- `apps/web/src/utils/libraryStatus.ts`
- `apps/web/src/utils/libraryStatus.spec.ts`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`
- `_bmad-output/implementation-artifacts/12-2-season-episode-list.md`（AC drift reference — see Completion Notes；未修改）

## Senior Developer Review (AI)

- **Reviewer:** Claude Opus 5.5（adversarial self-review，2026-10-29）
- **Outcome:** Approve after fixes
- 🔒 Rule 7 Wire Format: N/A (no Go error-code files in scope)
- 🔒 Rule 20 Contract Bump: N/A (no stamp bumps in this review)
- 🔒 Rule 25 Mega-line: N/A (no mega-line change behind a rebase/merge)
- Git vs File List：一致（10 個前端檔＋sprint-status；12-2 只作 drift 參考、未修改）。

### Findings

- [x] **[M1] 打開對話框時發現已在跑，季清單不會開始追**（`ManageSubtitleDialogV2.tsx` attach effect）。列若是在這次執行開始前抓的（staleTime 只在掛載／切回視窗時生效），使用者看到對話框在跑、關掉後列仍是 ✓。修：attach 時也呼叫 `onGenerationStarted`，經 ref 讀取（父層每次 render 給新的 arrow，放進 effect 依賴會無限重抓）。測試：「episode — running when opened → tells the parent a run is going, exactly once」（拿掉那行會紅，已驗證）。
- [x] **[M2] AC #2 文字和實作不符**：AC 寫「任何 `subtitle_progress` 或 `transcription_extracting` 都重抓」，實作刻意不對 `transcription_extracting`／`probing` 重抓（兩者都早於 DB 寫入，重抓只會拿到舊值）。實作才是對的 → 修正 AC 文字並附程式位置。
- [x] **[L1] 隱藏分頁跳過輪詢沒有測試** → 補「skips the poll while the tab is hidden and resumes when it is visible」。
- [ ] **[L2] SSE 開啟後出錯就關、不重連**：那次執行剩下時間只靠 15 秒輪詢。設計取捨（通道 SSE 不可靠、避免重連迴圈），延到 `disc-2026-10-episode-row-live-progress-idle-detect`。
- [ ] **[L3] 多季同時展開且都在跑時各開一條 `/events`**：實際一次只有一季在跑；共用連線延到同一張。
- [ ] **[L4] 季清單閒置時不會發現別處開始的生成**：-a 範圍外，延到同一張（Discovery Triage ③）。

## Change Log

| Date | Change |
|------|--------|
| 2026-10-29 | Story created (Bob)：查證 staleTime 推論、後端排隊／進行中／結束的狀態值。 |
| 2026-10-29 | Dev：新 hook `useEpisodeGenerationLive`（開始即重抓＋待確認快速輪詢、飛行中 SSE＋15s 保底、沒在跑完全安靜）；對話框 `onGenerationStarted`；季清單 staleTime 1h → 30s；`isSubtitleInFlight` 共用。測試 +20。 |
| 2026-10-29 | CR：修 M1（attach 時通知季清單）、M2（AC #2 文字對齊實作）、L1（補隱藏分頁測試）；L2–L4 延到 `disc-2026-10-episode-row-live-progress-idle-detect`。Status → done。 |
