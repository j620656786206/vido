# Story disc-2026-10-solo-live-spend：單集生成跑到一半，就回報「到目前花了多少」

Status: done

**Source:** sprint-status `disc-2026-10-solo-live-spend`（Sally 2026-10-28 從程式查到）。⚖️ Alexyu 2026-10-28 選 A：要做，後端先做；新的季清單列／停止確認框／活動列畫面等設計稿（不在本單）。相關傘狀：`disc-2026-10-episode-row-live-progress`（列上「轉錄 3/12 段 · US$0.12」）、`disc-2026-10-solo-generation-cancel`（確認框「已花的 US$0.31 不會退」）。本單只負責「後端送出真實數字」，外加：既有 F3「本次用量」若因事件沒帶欄位而卡在 US$0.00，讓它照設計稿即時更新（必要的前端小改一併做）。

## 查到的事（2026-10-28，Bob 逐一開檔確認）

### 正式機設定

- NAS `docker inspect Vido`：`VIDO_SUBTITLE_PIPELINE_MODE=pipeline`；`AI_RUN_BUDGET_USD` **沒設** → 預設 5.0（`apps/api/internal/config/config.go:211`）。所以每次單集生成都有 US$5 上限，F3 的「本次用量」那行會出現（它要 spent 和 budget 兩個數字才畫，見下）。

### Sally 說法 1：「單集生成途中事件不帶 spent_usd」→ **pipeline 模式成立；legacy 模式不成立**

- pipeline 模式的點擊走 `SoloRunner`。每個工作自己有一個 `ai.Budget`（`apps/api/internal/subtitle/solo_runner.go:248`），掛在 ctx 上傳給管線（`:264`）。
- SoloRunner **自己**只送兩個事件：開始（`emitStart`，`:396-403`）和結束（`emitTerminal`，`:405-`）；兩者都經 `jobFields` 帶 `spent_usd`（一律）與 `budget_usd`（有上限才帶）（`:378-393`）。開始時預算剛建立，spent 是 0。
- 中途的事件來自兩個別人：
  1. **管線的 D6 `subtitle_progress`**（翻片內英文字幕那條：probing／extracting／translating 每一段）——由 `NewSSEProgressHook` 送出，payload 是 `map[string]string{media_id, media_type, stage, message}`，**沒有花費**（`apps/api/internal/subtitle/progress_sse.go:61-75`）。
  2. **聽寫支線**（ASR 路線）：管線經 `cmd/api/asr_adapter.go:36` 呼叫 `TranscriptionService.RunTranscription`，它以 `Solo=false` 登記（`apps/api/internal/services/transcription_service.go:619`）。`resolveBudget` 在 ctx 上找到 SoloRunner 的 budget → `ownBudget=false`（`:653-659`），所以**不會** `trackRunBudget`（`:959-962`），`job.budget` 是 nil。`broadcastEvent` 每個 transcription_* 事件都會呼叫 `costFields`（`:2425-2441`），但 `costFields` 只讀 `job.budget`，nil 就什麼都不加（`:677-697`）。→ 聽寫途中的 `transcription_progress`／`translation_progress` 都沒有花費。
- 管線本身在 ctx 沒有 budget 時才自己建一個（`apps/api/internal/subtitle/process_item.go:104-106`）；SoloRunner 的工作一定已經帶了，所以花費全部記在 SoloRunner 那個 budget 上。
- **legacy 模式**（`StartTranscription`，Solo=true）：`resolveBudget` 自己建 budget → `trackRunBudget` 記到 `job.budget` → 每個 transcription_* 事件**都有** `spent_usd`（9R-17 AC #1）。所以 legacy 不是問題。

### Sally 說法 2：「F3『本次用量』整段停在 US$0.00、結束才跳」→ **成立（pipeline 模式、有上限時）**

- F3 管理字幕對話框用 `useGenerationProgress`。每個進度事件（`transcription_extracting`／`transcription_progress`／`translation_progress`，以及 D6 `subtitle_progress` 的 probing／extracting／translating）都走 `PHASE`，`PHASE` 會用 `cost()` 合併花費：事件有 `spent_usd` 就換、沒有就保留上一個（`apps/web/src/hooks/useGenerationProgress.ts:178-183, 201-214, 460-466, 503-510`）。
- 對話框只在 spent 和 budget **都是數字**時才畫「本次用量」（`apps/web/src/components/subtitle/ManageSubtitleDialogV2.tsx:1180-1184`，套在 `:789`；畫的地方 `GenerationProgressV2.tsx:363`）。
- 所以：開始事件給 `spent_usd: 0, budget_usd: 5` → 顯示「本次用量：$0.00 / 上限 $5.00」；中途事件都沒帶花費 → 一直保留 0；結束事件才帶真實數字。和 Sally 描述的一樣。
- **前端不用改就能讀新欄位**：`PHASE` 本來就從任何一個屬於這部片的事件讀 `spentUsd`，D6 那條也一樣（payload 整個傳進 reducer）。後端補上欄位，F3 就會即時跳。
- 生成工作區（`GenerationWorkspaceV2`）的單集列也畫「本次用量」（`GenerationWorkspaceV2.tsx:384-390`，資料來自 `useGenerationJobsFeed` 的 `singleJobs`，`:1014-1015`）。transcription_* 那條會讀 `spentUsd`（`useGenerationJobsFeed.ts:382-393`），但 **D6 那條只把 `{ message }` 傳進 reducer**（`useGenerationJobsFeed.ts:657`），花費會被丟掉；`PipelinePayload` 型別也只有 `mediaId/stage/message`（`:196-200`）。→ 要小改，否則翻英文字幕那條在工作區仍卡在 0。

### 活動 API

- `ActiveJob`（`apps/api/internal/services/activity_service.go:140-158`）沒有花費欄位；`soloJobRows`（`:302-322`）從 `SoloJobSnapshot` 組列，`SoloJobSnapshot`（`apps/api/internal/services/solo_activity.go:12-23`）也沒有花費。
- 兩個來源：`SoloRunner.ActiveSoloJobs`（`solo_runner.go:312-327`，`job.Budget` 就在旁邊）與 `TranscriptionService.ActiveSoloJobs`（只列 Solo=true，`transcription_service.go:826-`，`job.budget` 在 trackRunBudget 後才有）。`cmd/api/solo_job_source.go:24-26` 把兩邊接起來；聽寫支線是 Solo=false，不會被列兩次。
- 前例：最近活動的 `spent_usd` 是 `*float64` + omitempty，「0 照送 0、沒記錄就不送」，原始數字、不先四捨五入（`activity_service.go:233-235`）。

### 不會重複算的地方

- `GET /ai/usage` 的 `ActiveManualUsage`：`TranscriptionService` 只加 `job.budget`（自己建的）（`transcription_service.go:702-715`），`SoloRunner` 加自己的 jobs（`solo_runner.go:346-356`），`composedSoloJobs` 相加（`solo_job_source.go:28-33`）。聽寫支線的 `job.budget` 是 nil → 不算。本單**不能**把 SoloRunner 的 budget 寫進 `job.budget`，否則 `/ai/usage` 會算兩次。

### 契約文件

- SSE：`docs/sse-event-types.md` ／ `docs/sse-event-types.zh-TW.md`，`subtitle_progress` 一節（欄位表 `:157-164`）；transcription_* 沒有在這份文件裡（宣告在 `transcription_service.go`）。
- 活動 API：`_bmad-output/implementation-artifacts/ux3-2-2-activity-api.md`「Real vs. greenfield」表 `active_jobs.transcription` 那列（`:57`），新欄位寫在 story AC 並標 `[@contract-v1]`（前例 `disc-2026-10-activity-solo-jobs-collapsed` AC #7）。
- 先前 `disc-2026-10-single-generate-ignores-embedded-english-a` 寫「不要改 D6 `subtitle_progress` 的 payload（sub-1-3 `[@contract-v1]`）」——那張的意思是不要動 stage 詞彙與既有欄位；本單只**加**兩個選填欄位，既有欄位與 stage 值不變，屬 additive，不升版（Rule 20：additive 不升版的前例 = FreeOnly）。

## 設計（Bob × Winston，architecture-first）

**一個「進行中單集花費表」**，讓「不擁有這個工作、但在幫它送事件」的兩個地方查得到它的 budget：

- 新型別 `services.SoloSpendBoard`（`internal/services/solo_spend_board.go`）：`map[mediaID]*ai.Budget` + mutex。`Track(mediaID, b)`、`Untrack(mediaID, b)`（只在還是同一個 budget 時才刪）、`Lookup(mediaID) *ai.Budget`；nil receiver 安全。放在 services 是因為 subtitle 可以 import services，反過來不行（Rule 19）。
- 共用寫法 `services.AddRunCost(data, budget)`：spent_usd 一律、budget_usd 有上限才加——SoloRunner `jobFields`、`TranscriptionService.costFields`、D6 hook 都用它，三處語意不會漂。
- **SoloRunner**：`WithSoloSpendBoard(board)`；接受點擊時 `Track(ref.ID, job.Budget)`，`ProcessItem` 回來、刪 job 的同一處 `Untrack`（在結束事件與釋放 in-flight 之前）。
- **TranscriptionService**：`SetSoloSpendBoard(board)`（main.go 開機時設，存取在 `s.mu` 下）。`costFields`：自己的 `job.budget` 優先；沒有時才查 board。`ActiveManualUsage` **不變**（只算自己的）→ 不重複。
- **D6 hook**：`NewSSEProgressHook(hub, opts...)` 加 `WithProgressSpend(board)`；有登記的 media 就在 payload 加花費。payload 改成 `map[string]interface{}`（wire JSON 完全一樣；`PhaseTracker.payloadString` 兩種型別都吃，`phase_tracker.go:136-147`）。
- **活動**：`SoloJobSnapshot.SpentUSD *float64`；SoloRunner 從 `job.Budget`、legacy 從 `job.budget`（還沒掛上就 nil）。`ActiveJob.SpentUSD *float64 json:"spent_usd,omitempty"`，只有 `transcription` 列會填。
- 為什麼不用 ctx 標記：D6 的 `emitProgress(ref, stage, message)` 有 19 個呼叫點、多數沒有 ctx，改簽章牽動太大；一張以 media id 為 key 的表兩邊都能查，且單一飛行（`SoloGuard`）保證同一部片同時只有一個工作。
- 成本：每個事件一次 map 查找 + budget 自己的 mutex 讀快照，沒有 DB。

## Story

身為一次替一整季按好幾次「生成字幕」的人，
我希望在字幕還在生成時就看得到這一集到目前真正花了多少錢，
這樣我決定要不要停下來時，是看真實數字，而不是一直停在 US$0.00 的假象。

## Acceptance Criteria

1. **聽寫支線途中事件帶花費：** pipeline 模式下 SoloRunner 工作的聽寫支線（`TranscriptionService.RunTranscription`，Solo=false）送出的 `transcription_extracting`／`transcription_progress`／`translation_progress`／`transcription_complete`／`transcription_failed`，帶該 SoloRunner 工作 budget 的當下快照：`spent_usd`（number，原始值，一律）、`budget_usd`（number，有上限才帶）。花費增加後的下一個事件，`spent_usd` 跟著變大。
2. **D6 途中事件帶花費：** 管線為 SoloRunner 工作的那部片送出的 `subtitle_progress`（經 `NewSSEProgressHook`），多兩個選填欄位 `spent_usd`／`budget_usd`，語意同 AC #1。既有四個欄位與 stage 值不變。
3. **不是單集工作就不帶：** 批次（consent batch）項目、FR12 pool 項目、搜尋引擎送的 `subtitle_progress`，payload 與現在一樣——沒有 `spent_usd`／`budget_usd`（absent ≠ 0）。legacy 單集（Solo=true、自己的 budget）行為不變。
4. **不重複算：** `GET /ai/usage`（`ActiveManualUsage`）在 SoloRunner 工作的聽寫支線進行中，數字 = 該工作 budget 一次，不是兩次；活動頁不會多出一列。
5. **活動 API `[@contract-v1]`（只加不改）：** `active_jobs.jobs[]` 的 `kind: "transcription"` 列新增
   - `spent_usd`: number，選填。有 = 這個工作 budget 的當下讀數（原始值、不先四捨五入；`0` = 到目前還沒花錢，照送 0）；沒有 = 這個工作還沒有計費表（legacy 工作在 budget 掛上之前）。排隊中的列也照送（排隊前可能已花錢）。
   - 其他 kind（scan／subtitle_batch／generation_batch）不加。
6. **F3「本次用量」即時更新：** 前端 `useGenerationProgress` 收到帶 `spent_usd` 的 D6 `subtitle_progress`（translating）時，`spentUsd` 更新——補一個 hook 測試釘住（transcription_* 那條已有測試）。
7. **生成工作區單集列同樣即時：** `useGenerationJobsFeed` 的 D6 frame 把 `spentUsd`／`budgetUsd` 一起傳進 reducer（目前只傳 message）；批次成員那條不受影響（後端本來就不帶）。補測試。
8. **型別與文件：** 前端 `ActiveJob` 型別加 `spentUsd?: number`（註明 absent ≠ 0；畫面本單不用）。`docs/sse-event-types.md` + `.zh-TW.md` 的 `subtitle_progress` 欄位表加兩個選填欄位並說明何時有；`ux3-2-2-activity-api.md` 那列補 `spent_usd`。
9. **成本：** 讀花費不查 DB；只有 map 查找 + `Budget.Snapshot()`。

## Tasks / Subtasks

後端（Go）：
- [x] T1 `services.SoloSpendBoard` + `AddRunCost` + 單元測試（AC #1, #2, #9）
- [x] T2 SoloRunner：`WithSoloSpendBoard`、Track／Untrack、`jobFields` 改用 `AddRunCost`、`ActiveSoloJobs` 填 `SpentUSD`（AC #1, #2, #5）
- [x] T3 TranscriptionService：`SetSoloSpendBoard`、`costFields` 查 board、`ActiveSoloJobs` 填 `SpentUSD`；`ActiveManualUsage` 不變（AC #1, #3, #4, #5）
- [x] T4 D6 hook：`WithProgressSpend`、payload 改 `map[string]interface{}`（AC #2, #3）
- [x] T5 Activity：`ActiveJob.SpentUSD`、`soloJobRows` 帶過去；main.go 接線（建一張 board 給三方）（AC #5）
- [x] T6 Go 測試：途中事件花費遞增（聽寫支線、D6）、批次／搜尋不帶、`/ai/usage` 不重複、活動列有／沒有 `spent_usd`、JSON 形狀（AC #1–#5）

前端：
- [x] T7 `useGenerationJobsFeed` D6 frame 帶花費 + `PipelinePayload` 型別 + 測試（AC #7）
- [x] T8 `useGenerationProgress` D6 花費測試；`ActiveJob.spentUsd` 型別（AC #6, #8）

文件：
- [x] T9 SSE 文件雙語、活動 API 文件（AC #8）

（拆單檢查：後端 6、前端 2 → 前端 ≤3，不用拆。）

## Dev Notes

### 已知陷阱

- **不要**把 SoloRunner 的 budget 寫進 `soloTranscriptionJob.budget`：`ActiveManualUsage` 會把它和 SoloRunner 自己的加總算兩次（`transcription_service.go:702-715` + `solo_job_source.go:28-33`）。用 board，`costFields` 只在自己沒有 budget 時才查。
- `Untrack` 要比對是同一個 budget 才刪；順序放在 SoloRunner 刪 job 那段（`solo_runner.go:281-285`），在 `emitTerminal` 與 `guard.Release` 之前，這樣下一次點擊的 Track 不會被舊工作刪掉。
- `costFields` 目前在 `s.mu` 裡讀 job；board 指標也在 `s.mu` 裡讀，然後**放鎖後**才 `Lookup`／`Snapshot`（不要巢狀鎖）。
- `broadcastEvent` 只對 `carriesRunCost` 的五種事件加花費（`subtitle_run_receipt` 有自己的「本 run 增量」，不可覆蓋）——維持不變。
- `progress_sse_test.go:21` 目前斷言 payload 是 `map[string]string`，改型別要一起改。
- 規則：只送記錄到的數字；不要把 absent 補成 0。

### 不要做的事

- 不畫新的列／確認框／活動列（等設計稿）。
- 不改批次列（`generation_batch` 已在自己的 SSE 帶整批花費；活動列要加得改 primitive 介面，不算 trivial）。
- 不改 `GET …/transcribe/status`（重新打開對話框時花費要等下一個事件才出現——見 Discovery Triage）。

### 測試

- Go：`go test ./internal/services/... ./internal/subtitle/... ./cmd/api/...`、`go test ./...` 一次、`go vet ./...`、`staticcheck ./...`。
- 前端：`pnpm nx test web`（前景）、`pnpm run lint:all`（看 TOTAL 那行）、`pnpm run format:check`。
- 本機 node 要用 `export PATH="$HOME/.nvm/versions/node/v24.13.0/bin:$PATH"`。
- 不跑任何付費實測。

### Time-dependent visual coverage

N/A — no wall-clock-reading components touched.

### References

- [Source: apps/api/internal/subtitle/solo_runner.go:248-300, 312-356, 378-403]
- [Source: apps/api/internal/services/transcription_service.go:619, 653-715, 959-962, 2425-2441]
- [Source: apps/api/internal/subtitle/progress_sse.go:61-75]
- [Source: apps/api/internal/services/activity_service.go:140-158, 233-235, 302-322]
- [Source: apps/web/src/hooks/useGenerationProgress.ts:178-214]
- [Source: apps/web/src/hooks/useGenerationJobsFeed.ts:196-200, 382-393, 645-662]
- [Source: _bmad-output/implementation-artifacts/9R-17-ai-usage-endpoint.md AC #1, #3]

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5 (claude-opus-5-5)

### Debug Log References

- `go test ./... -count=1`（apps/api，CR 修完後重跑）：44 個套件全綠；`go vet ./...` 乾淨；`staticcheck ./...` 無輸出。
- `go test -race ./internal/subtitle/ -run 'SoloRunner|SSEProgressHook'`、`./internal/services/ -run 'SoloSpend|AddRunCost'`：綠。
- `pnpm nx test web --skip-nx-cache`：298 檔／4910 測試全綠；`pnpm run lint:all`：TOTAL `✖ 170 problems (0 errors, 170 warnings)`，本單動到的 4 個前端檔 eslint 0 警告；`pnpm run format:check`：全部通過。測完 `ps aux | grep vitest` = 0。

### Completion Notes List

- Ultimate context engine analysis completed - comprehensive developer guide created
- **後端：** 新增 `services.SoloSpendBoard`（media id → 單集工作的 `ai.Budget`）與共用寫入器 `services.AddRunCost`。SoloRunner 接受點擊時登記、工作結束（刪 job 那段，在結束事件與釋放 in-flight 之前）註銷；`jobFields` 改用 `AddRunCost`。`TranscriptionService.costFields`：自己的 budget 優先、沒有才查 board（聽寫支線 Solo=false 因此帶花費）；`ActiveManualUsage` 不動 → `/ai/usage` 不重複算。D6 hook 加 `WithProgressSpend(board)`，payload 由 `map[string]string` 改 `map[string]interface{}`（非單集項目的 JSON 逐字不變，有測試釘住）。活動 `ActiveJob.SpentUSD *float64`（transcription 列；SoloRunner 工作從接受起就有 → 0 也照送；legacy 工作 budget 掛上前沒有欄位）。main.go 在 pipeline 模式建一張 board 給三方。
- **前端：** F3「本次用量」本身**不用改**——`useGenerationProgress` 的 `PHASE` 本來就從任何屬於這部片的事件合併 `spentUsd`（D6 也是），補了一個測試釘住（AC #6）。生成工作區單集列（`useGenerationJobsFeed`）原本把 D6 frame 的花費丟掉，改成一起傳（AC #7）。`ActiveJob.spentUsd` 型別補上（畫面不讀）。
- **批次列：** 不改（理由見 Dev Notes「不要做的事」）。
- 🔗 AC Drift: FOUND —（1）`progress_sse_test.go` 原斷言「payload must stay map[string]string」：Go 端型別改 `map[string]interface{}`，wire JSON 對非單集項目逐字相同（`TestSSEProgressHook_ItemNotOnTheBoardKeepsItsFourKeys`），前端與 `PhaseTracker.payloadString` 兩種型別都吃。（2）`disc-2026-10-single-generate-ignores-embedded-english-a` Dev Notes「不要改 D6 `subtitle_progress` 的 payload」：本單只加兩個選填欄位、只在單集工作的項目出現，既有四欄與 stage 值不變。（3）9R-17 AC #1「批次項目不帶」：仍成立（board 只登記單集工作）。e2e 掃描：`grep -rn subtitle_progress tests/e2e tests/visual apps/web/src/routes/test` → 只有 `generation-workspace.spec.ts:122`（搜尋引擎的 frame，未受影響）。
- 📎 Contract Stamps: FOUND（本單 AC #5 新 `[@contract-v1]`（活動列 `spent_usd`），additive、不升任何既有版；SSE 兩欄 additive 不升版。上游：confirmed against `[@contract-v1]` sub-1-3 AC #1（12 值 `PipelineStage`，未動）、confirmed against `[@contract-v1]` disc-2026-10-activity-solo-jobs-collapsed AC #7（`kind/detail/media_id/queued` 未動））。
- 🎭 A11y Pre-Flight: PASS（0 個元件變動——只動 hook 與型別；觸及檔案 jsx-a11y 警告 0，本單新增 0）。
- UX 驗證（step 9）：沒有改任何畫面元件；F3 本次用量的樣子照舊，只是數字會在途中更新。
- 付費實測：沒有跑（依指示）。

### Discovery Triage

- ① expand-scope-in-place → AC #8：SSE 文件 `media_type` 少列生成管線的 `"episode"`（CR L4，已修）。
- ③ backlog-with-carry-forward-link → `backlog-transcribe-status-live-spend`：重新打開管理字幕（掛回進行中的工作）時，`GET …/transcribe/status` 不帶花費，`START` 把 `spentUsd` 歸 null，「本次用量」要等下一個帶花費的事件才出現（不是錯的數字，只是晚出現）。非阻擋。

### File List

- apps/api/internal/services/solo_spend_board.go（新）
- apps/api/internal/services/solo_spend_board_test.go（新）
- apps/api/internal/services/solo_live_spend_test.go（新）
- apps/api/internal/services/solo_activity.go
- apps/api/internal/services/transcription_service.go
- apps/api/internal/services/activity_service.go
- apps/api/internal/subtitle/solo_runner.go
- apps/api/internal/subtitle/solo_runner_test.go
- apps/api/internal/subtitle/progress_sse.go
- apps/api/internal/subtitle/progress_sse_test.go
- apps/api/cmd/api/main.go
- apps/web/src/hooks/useGenerationJobsFeed.ts
- apps/web/src/hooks/useGenerationJobsFeed.spec.ts
- apps/web/src/hooks/useGenerationProgress.spec.ts
- apps/web/src/services/activityService.ts
- docs/sse-event-types.md
- docs/sse-event-types.zh-TW.md
- _bmad-output/implementation-artifacts/ux3-2-2-activity-api.md
- _bmad-output/implementation-artifacts/disc-2026-10-solo-live-spend.md
- _bmad-output/implementation-artifacts/sprint-status.yaml
- _bmad-output/implementation-artifacts/disc-2026-10-single-generate-ignores-embedded-english-a.md（AC drift reference — see Completion Notes；未修改）

### Senior Developer Review (AI) — 2026-10-28

**Outcome:** Approve（修完 in-scope 項目後）。Git vs File List 差異 0（`.claude/memory/MEMORY.md` 是開工前就有的未提交修改，不屬本單）。
🔒 Rule 7 Wire Format: PASS（本單沒有新增錯誤碼常數）· 🔒 Rule 20 Contract Bump: N/A（沒有升版；新 `[@contract-v1]` + additive 欄位）· 🔒 Rule 25 Mega-line: N/A（沒動 project-context.md）。

AC 逐條：#1 ✅（`TestRunTranscription_ASRLegEventsCarryTheClicksLiveSpend`、`TestCostFields_ASRLegOfASoloClickReadsTheBoard`）· #2 ✅（`TestSoloRunner_StageEventsCarryTheJobsLiveSpend`、`TestSSEProgressHook_SoloItemCarriesTheJobsLiveSpend`）· #3 ✅（`TestSSEProgressHook_ItemNotOnTheBoardKeepsItsFourKeys`、`TestPipeline_ItemWithoutASoloJobSendsNoStageCost`、既有 `TestCostFields_OnlyARunOwnedBudgetAddsKeys`）· #4 ✅（`TestASRLegIsNotCountedAgain`、`TestSoloRunner_ASRLegSpendsFromTheBoardsBudget`）· #5 ✅（`TestActivity_SoloRowCarriesTheJobsLiveSpend`）· #6 ✅ · #7 ✅ · #8 ✅ · #9 ✅（map 查找 + `Budget.Snapshot`，無 DB）。

發現與處理：
- **[M1] 已修** 測試缺口：AC #3 只有 hook 單元測試，沒有「同一條管線、不經 SoloRunner 跑（pool／批次直接呼叫 ProcessItem）」的整合證明 → 補 `TestPipeline_ItemWithoutASoloJobSendsNoStageCost`。
- **[L1] 已修** `docs/sse-event-types*.md` 的 `subtitle_progress` 發布者只列搜尋路徑，沒列生成管線的 `NewSSEProgressHook`（唯一會帶花費欄位的發布者）→ 雙語補上。
- **[L2] 已修** sprint-status 傘狀 `disc-2026-10-episode-row-live-progress` 仍寫「即時金額要等 solo-live-spend 決定」→ 補「後端已做、欄位見 AC #1/#2/#5」。
- **[L3] 不修（記錄）** `SoloSpendBoard` 只用 media id 當 key、不分 movie／episode：id 是 UUID，加上 SoloGuard 單一飛行，不會撞；與既有 `PhaseTracker`、`costFields` 同一慣例。
- **[L4] 已修（lane ①，併入 AC #8 文件工作）** `docs/sse-event-types*.md` 寫 `subtitle_progress` 的 `media_type` 只有 `"movie"`／`"series"`，生成管線實際送 `"movie"`／`"episode"`（`MediaRef.MediaType` = `models.SubtitleRunMediaEpisode`）——本單之前就存在的文件落差；雙語欄位表補上生成管線的值（搜尋路徑的描述照舊，未查證不改）。
- **[L5] lane ③** 重新打開對話框時「本次用量」晚出現 → `backlog-transcribe-status-live-spend`（見 Discovery Triage）。

驗證（CR 修完後重跑）：見 Change Log 第二列。

## Change Log

| Date | Change |
| ---- | ------ |
| 2026-10-28 | 建單（Bob）；dev 完成：單集工作途中事件（聽寫支線 transcription_*、D6 subtitle_progress）帶 `spent_usd`／`budget_usd`；活動 transcription 列新增 `spent_usd` `[@contract-v1]`（additive）；生成工作區 D6 frame 帶花費；SSE／活動 API 文件更新。 |
| 2026-10-28 | CR：補 AC #3 整合測試（M1）、SSE 文件補生成管線發布者（L1）與 media_type 值（L4）、傘狀條目同步（L2）；L3 記錄不修，L5 → `backlog-transcribe-status-live-spend`。狀態 done。 |
