# Story disc-2026-10-activity-solo-jobs-collapsed：一集一集按「生成字幕」，活動頁就一集一列，看得出哪幾集在跑、哪幾集在排隊

Status: done

**Source:** Alexyu 2026-10-28 回報（截圖）。backlog 條目 `disc-2026-10-activity-solo-jobs-collapsed`（P2・顯示，Rule 24 lane ③）。⚖️ Alexyu 2026-10-28 選「現在修」；每集一列或合併一列由 Sally 決定 → Sally 決定**每集一列**（見 AC #1–#6），不改 `ux-design.pen`。

## 查到的事（2026-10-28，Bob 逐一開檔確認）

### 正式機證據（NAS 2026-10-28 21:35）

- 在《特種部隊：母獅》S01E01～E08 的詳細頁各按一次「生成字幕」：8 次 `POST /api/v1/episodes/:id/transcribe` 都回 202，`subtitle_runs` 8 筆 `running`。
- log：E01 翻譯中、E02／E03 聽寫中，其他 5 集在等抽取（音訊／字幕軌）的位子。
- 活動頁「進行中」只有一列「字幕生成中 · 特種部隊：母獅 S01E01」，計數徽章寫 **1**；側欄的活動徽章也是 1。看起來像只有一集在跑。

### 為什麼只有一列

- 正式機是 `VIDO_SUBTITLE_PIPELINE_MODE=pipeline`，詳細頁的點擊走 `SoloRunner`（`internal/subtitle/solo_runner.go`）。`SoloRunner.ActivityProgress` 把所有工作加成一個數字、片名取 map 裡隨便一個；`ActiveMediaID` 多於一個就回 `ok=false`，所以連步驟都沒有。
- 舊模式的 `TranscriptionService.ActivityProgress`（`internal/services/transcription_service.go:798`）同樣只算 Solo 工作、同樣塞成一列（註解寫「picking is not worth ordering machinery」）。
- `cmd/api/solo_job_source.go` 的 `composedSoloJobs` 把兩邊加總（`cmd/api/main.go:1426-1433` 接上）。
- `ActivityService.activeJobsSection`（`internal/services/activity_service.go`）只送一筆 `kind: "transcription"`；`stepOf` 只在剛好一個工作時有用。
- 前端 `ActivityHub.tsx` 的 `ActiveSection` 照 `jobs` 一筆畫一列，徽章 = `jobs.length`；nav 徽章 `useInflightJobCount` 也是 `jobs.length`（同一條計數路徑）。所以只要後端一個工作送一筆，前端列數與兩個徽章就自然變成 8。

### 「排隊中」要怎麼誠實判斷（查過的程式）

- 抽取一次只准一個：`subtitle.ExtractGate`（`extract_gate.go`，容量 1，全程序共用）。
  - 字幕軌抽取：`Extractor.Extract` 在 `gate.Acquire(ctx, onWait)` 等位子；`onWait(ahead>0)` = 開始等、`onWait(0)` = 拿到位子；ctx 結束時回 `ErrSubtitleExtractWaitAborted`。
  - 音訊抽取（聽寫用）：`AudioExtractorService.ExtractAudio` 先等自己的 semaphore（`main.go` 設 `maxConcurrent=1`），再經 `extractSlotAdapter` 等同一個 `ExtractGate`。聽寫路線的 5 集主要卡在這裡。
- `PhaseTracker`（`subtitle.NewPhaseTracker`、`Step(mediaID)`）分不出排隊：等位子時管線送的是 `subtitle_progress` stage=`extracting`（訊息「waiting for the extraction slot (N ahead)」），音訊那條根本不送事件，兩者在 tracker 裡都只是 `extracting`（活動頁會寫「提取音訊」）。所以**不能**拿 PhaseTracker 判斷排隊。
- ⚖️ Bob 裁定：**「排隊中」= 這個工作此刻正卡在抽取位子上（字幕軌抽取的 gate、音訊抽取的 semaphore 或 gate）**。用一個掛在 ctx 上的觀察者回報「開始等／不等了」，SoloRunner 和舊模式的 `TranscriptionService.StartTranscription` 在自己的工作 ctx 掛上它。拿到位子就回到「執行中」。一個工作可能先等字幕軌、跑一下、再等音訊——它就會「排隊中 → 執行中 → 排隊中 → 執行中」，每次都是當下真的狀態。
  - 管線的聽寫支線（`asr_adapter.go` → `RunTranscription(ctx)` → `runPipeline` → `ExtractAudio(ctx)`）沿用同一個 ctx，觀察者會一路帶到音訊抽取。
  - AI 呼叫的併發節流（`ai.Governor`）是每次呼叫輪流、不是整個工作卡住，不算排隊。

### 時間戳

- `SoloRunner` 的 `soloJob.StartedAt` 已有（接受點擊的時間，`s.now()`）。
- 舊模式 `soloTranscriptionJob` 沒有時間 → 要加 `StartedAt`。
- 兩邊都要加「開始排隊的時間」`QueuedAt`（零值 = 沒在等）。

### API 契約慣例

- `/api/v1/activity` 的契約在 `ux3-2-2-activity-api.md`（「Real vs. greenfield」表），新欄位寫在 story 的 AC、標 `[@contract-v1]`（前例 `disc-2026-10-activity-subtitle-run-time-a` AC #1、`disc-activity-downloads-unconfigured-copy`）。只加不改。

## Story

身為一次替一整季按好幾次「生成字幕」的人，
我希望活動頁每一集都有自己的一列，寫出它在提取、轉錄、翻譯還是在排隊，
這樣我按了 8 集就看到 8 列，不會以為只有一集在跑而再去按一次。

## Acceptance Criteria

1. **一個單集／單部生成工作一列**（Sally 規則 1）：沿用現有「字幕生成中」那列，不改樣子——Sparkles 圖示、標題「字幕生成中」、下面那行是那個工作的片名（現有格式，例如「特種部隊：母獅 S01E01」）。
2. **執行中的那列**（規則 2）：右邊照現在寫它**自己的**步驟（「提取音訊」「轉錄 3/22 段」「翻譯中」…），還沒有步驟就寫「進行中」；有段數／句數時畫進度條，`percent_done` = current×100÷total。
3. **排隊中的那列**（規則 3）：右邊只寫「排隊中」，**不畫進度條**，也不顯示步驟（即使 PhaseTracker 記了 `extracting`）。
4. **順序**（規則 4）：先全部執行中的，再全部排隊中的。執行中依開始時間（接受點擊的時間）舊的在前；排隊中依開始排隊的時間舊的在前（= 拿到位子的順序 ≈ 點擊順序）。不照集數排。同一時間再依 media id 排，確保每次輪詢順序穩定。
5. **「批次生成」照舊一個批次一列**（規則 5），排在單集那些列前面（k1 的順序：掃描 → 批次字幕搜尋 → 批次生成 → 字幕生成中…）。
6. **「進行中」徽章 = 顯示的列數**（規則 6）：NAS 那個情況是 8。側欄／分頁列的活動徽章用同一個 `jobs.length`，也會是 8。
7. **API 契約 `[@contract-v1]`（只加不改）：** `active_jobs.jobs[]` 裡 `kind: "transcription"` 改成一個工作一筆，每筆：
   - `kind`: `"transcription"`（不變）
   - `detail`: 那個工作的片名（不變的意義，從「隨便一個」變成「這一個」）
   - `media_id`: string，**新增**——這個工作在生成的電影／單集 id（前端拿來當列的 key）
   - `queued`: bool，**新增**，`omitempty`——`true` = 正在等抽取位子；不送 = 執行中
   - `step`: 執行中且 PhaseTracker 有紀錄時才送（不變的形狀）；`queued: true` 時一律不送
   - `percent_done`: 執行中且 step 有 total 時 = current×100÷total，其他 0
   - `current`／`total`：transcription 不再送（原本的 `current` 是「幾個工作在跑」，現在等於列數；前端從來沒顯示它，`NO_PERCENT_KINDS`）。
   - 其他 kind（`scan`、`subtitle_batch`、`generation_batch`）一個位元組都不變。
   - 範例：`{"kind":"transcription","percent_done":13,"detail":"特種部隊：母獅 S01E02","media_id":"…","step":{"phase":"transcribing","current":3,"total":22}}`、`{"kind":"transcription","percent_done":0,"detail":"特種部隊：母獅 S01E05","media_id":"…","queued":true}`
8. **兩種模式都要對：** 管線模式（`SoloRunner`）與舊模式（`TranscriptionService` 的 Solo 工作）都一個工作一筆；`composedSoloJobs` 把兩邊串起來（不重複——管線的聽寫支線在 `TranscriptionService` 記成 Solo=false，不算）。
9. **首頁不壞：** `home_summary_service.go:244` 只用 `TranscriptionService.ActivityProgress` 的 `active`，保留這個方法與行為。
10. **檢查：** Go 測試（SoloRunner、TranscriptionService、composedSoloJobs、ActivityService、等位子的回報）、Vitest（8 個工作 → 8 列、徽章 8、排隊那列寫「排隊中」且沒有 progressbar、順序照後端），`go test ./...`、`staticcheck-2026.1 ./...`、`pnpm nx test web`、`lint:all`（看 `✖ N problems` 那行）、`format:check` 全綠；`grep -rn activity-job tests/` 確認 E2E 沒被影響；`ux3-2-2-activity-api.md` 補一列契約。

## Tasks / Subtasks

後端 3 個、前端 2 個 → 未達「兩邊各 >3」的拆單門檻，維持一張。

- [x] T1 等位子的回報（AC #3、#4）
  - [x] `services` 新增 ctx 觀察者：`WithExtractWaitObserver(ctx, func(waiting bool))`、`ReportExtractWait(ctx, waiting)`
  - [x] `AudioExtractorService.ExtractAudio`：semaphore 要等時回報 true；`extractSlotAdapter` 在 gate 的 `onWait` 回報；拿到兩個位子（或放棄）後回報 false
  - [x] `subtitle.Extractor.Extract`：gate 的 `onWait` 回報；`Acquire` 回來後回報 false
- [x] T2 每個工作的快照（AC #1、#4、#8）
  - [x] `services.SoloJobSnapshot{MediaID, Title, StartedAt, Queued, QueuedAt}`
  - [x] `SoloRunner.ActiveSoloJobs()`，工作 ctx 掛觀察者；拿掉不再用的 `ActivityProgress`／`ActiveMediaID`
  - [x] `TranscriptionService`：`soloTranscriptionJob` 加 `StartedAt`／`QueuedAt`，`StartTranscription` 掛觀察者，`ActiveSoloJobs()`；保留 `ActivityProgress`（首頁），拿掉不再用的 `ActiveMediaID`
  - [x] `composedSoloJobs.ActiveSoloJobs()` 串兩邊
- [x] T3 活動 API（AC #2–#7）
  - [x] `ActiveJob` 加 `MediaID`、`Queued`；`activeJobsSection` 一個工作一筆、照 AC #4 排序、排隊的不帶 step
  - [x] `ux3-2-2-activity-api.md` 補契約列
- [x] T4 活動頁（AC #1–#3、#6）
  - [x] `activityService.ts` 型別加 `mediaId?`、`queued?`
  - [x] `ActiveSection`：排隊 → 右邊「排隊中」、不畫進度條；列 key 用 `mediaId`
- [x] T5 檢查全綠（AC #10）

## Dev Notes

- **設計：** 不改 `.pen`。k1-d／k1-m 的「字幕生成中」列原樣重複 N 次；「排隊中」沿用右側讀數的字級與顏色（`text-xs text-[var(--text-secondary)]`，跟「進行中」「提取音訊」同一個樣子）。
- **Rule 19：** `services` 不能 import `subtitle`；快照型別與 ctx 觀察者放 `services`，`subtitle` 本來就 import `services`。
- **鎖：** 觀察者在抽取的 goroutine 上呼叫，只做一次上鎖的欄位寫入；`ActiveSoloJobs` 回複本，不把指標交出去。
- **活動頁 15 秒輪詢一次**，排隊→執行的切換最多晚 15 秒，可以接受。
- 首頁「進行中」格的計數是另一條路（見 Discovery Triage），這張不動它。

### Project Structure Notes

- 後端：`internal/services/{solo_activity.go(新), audio_extractor_service.go, transcription_service.go, activity_service.go}`、`internal/subtitle/{solo_runner.go, extractor.go}`、`cmd/api/{solo_job_source.go, main.go}`
- 前端：`apps/web/src/services/activityService.ts`、`apps/web/src/components/activity/ActivityHub.tsx`（+ spec）

### Time-dependent visual coverage

N/A — no wall-clock-reading components touched（前端只照後端給的順序畫；時間只在後端排序用）。

### References

- [Source: internal/subtitle/extract_gate.go — ExtractGate.Acquire 的 onWait 契約]
- [Source: internal/services/audio_extractor_service.go#ExtractAudio — semaphore + slot]
- [Source: _bmad-output/implementation-artifacts/disc-2026-10-activity-generation-progress-detail.md — step 的來源與 STEP_WORDS]
- [Source: _bmad-output/implementation-artifacts/ux3-2-2-activity-api.md — 活動 API 契約表]
- [Source: _bmad-output/screenshots/flow-k-activity-v2/k1-d.png、k1-m.png]

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（claude-opus-5-5），2026-10-28

### Debug Log References

- 本機 `node_modules` 是用 x64 的 node 裝的（只有 `@rollup/rollup-darwin-x64`），預設的 arm64 node v24.22.1 跑 vitest 會找不到 rollup 原生模組；改用 `~/.nvm/versions/node/v24.13.0`（x64）跑前端測試與 lint。沒有重裝套件。

### Completion Notes List

- **排隊怎麼判斷：** 新增 `services.WithExtractWaitObserver(ctx, func(waiting bool))`／`ReportExtractWait`／`ExtractWaitReporter`（`internal/services/solo_activity.go`）。三個會等抽取位子的地方回報：`subtitle.Extractor.Extract`（gate 的 `onWait`，`Acquire` 回來後一律回報 false）、`AudioExtractorService.ExtractAudio`（自己的 semaphore 要等時回報 true；semaphore＋gate 都拿到、或放棄時回報 false）、`cmd/api` 的 `extractSlotAdapter`（把觀察者轉成 gate 的 `onWait`）。`SoloRunner` 在工作 ctx 掛觀察者（管線的聽寫支線沿用同一個 ctx，所以音訊抽取的等待也收得到）；舊模式 `StartTranscription` 也掛（只改自己那個 job id 的狀態）。PhaseTracker 不用來判斷排隊（等位子時它只記得 `extracting`）。
- **每個工作的快照：** `services.SoloJobSnapshot{MediaID, Title, StartedAt, Queued, QueuedAt}`。`SoloRunner.ActiveSoloJobs()`、`TranscriptionService.ActiveSoloJobs()`（只列 Solo；`soloTranscriptionJob` 加 `StartedAt`／`QueuedAt`，服務加可注入的 `now`）、`composedSoloJobs.ActiveSoloJobs()` 串兩邊。重複回報「在等」不會重設 `QueuedAt`。
- **拿掉不再用的：** `SoloRunner.ActivityProgress`／`ActiveMediaID`、`TranscriptionService.ActiveMediaID`、`composedSoloJobs.ActivityProgress`／`ActiveMediaID`。`TranscriptionService.ActivityProgress` 保留（首頁 `home_summary_service.go:244` 用它的 `active`，AC #9）。`GenerationBatchProcessor.ActiveMediaID` 保留（批次列的 step）。
- **活動 API：** `ActiveJob` 加 `media_id`、`queued`（`omitempty`）。`activeJobsSection` 改呼叫 `soloJobRows`：一個工作一筆，排序 = 執行中（`StartedAt` 舊→新）→ 排隊中（`QueuedAt` 舊→新，同時再比 `StartedAt`），最後比 media id。排隊的那筆不帶 step、`percent_done` 0；執行中那筆用 `stepFor(mediaID)` 拿自己的 step。`NewActivityService` 的第四個參數型別改為 `soloJobLister`。`ux3-2-2-activity-api.md` 補了 `active_jobs.transcription` 一列。
- **活動頁：** `activityService.ts` 加 `mediaId?`、`queued?`；`ActiveSection` 對 `queued` 的 transcription 列右邊畫 `QueuedReadout`（「排隊中」，`data-testid="activity-job-queued"`，跟「進行中」同字級顏色），不畫進度條（就算帶了舊的 step／percent）；列的 key 改用 `kind-mediaId`。其他列不變。徽章與 nav 徽章本來就是 `jobs.length`，自然變 8。
- 🔗 AC Drift: FOUND —
  - `disc-2026-10-activity-generation-progress-detail` AC #4「`transcription`：只有一部在跑時帶 step；兩部以上不帶 step」→ 現在每一列帶自己那部的 step（排隊的不帶）。
  - `disc-2026-07-transcription-active-jobs`「所有 Solo 工作一列、`current` = 工作數、片名取任一個」→ 一個工作一列、不送 `current`。
  - E2E／gallery grep（`activity-job`、`active_jobs`、`activeJobs` 於 `tests/`）：只有兩處送空的 `jobs: []`，不受影響。
- 📎 Contract Stamps: FOUND（本 story AC #7 `[@contract-v1]` ×1；上游 `disc-2026-10-activity-subtitle-run-time-a` 的 `[@contract-v1]` 是 `recent.events`，本 story 沒動到）。
- 🎭 A11y Pre-Flight: 只換右側讀數文字；「排隊中」是純文字 span，沒有新互動元素；進度條在排隊列不渲染（不是 `aria-valuenow=0`）。
- **驗證：**
  - Go：`go test ./... -count=1` 44 個套件、8652 個測試全過（CR 修正後重跑）；`go vet ./...` 乾淨；`~/go/bin/staticcheck-2026.1 ./...` 零輸出。新測試：`internal/services/solo_jobs_activity_test.go`（8 個工作 → 8 列且順序 E01…E08、執行中各自 step、排隊列無 step／percent、排序含同時間的情況、批次列在前、`[@contract-v1]` JSON 鍵、`TranscriptionService.ActiveSoloJobs`／`setSoloQueued`、觀察者本身、音訊 semaphore 等待與放棄）、`internal/subtitle/extractor_gate_test.go`（gate 排隊回報 true→false、等到放棄也回報 false）、`internal/subtitle/solo_runner_test.go`（快照、經由工作 ctx 的等待 → 排隊 → 執行）、`cmd/api/solo_job_source_test.go`（兩邊串接、`extractSlotAdapter` 回報）。
  - 突變：排序把 `!a.Queued` 改成 `a.Queued` → 2 條紅；拿掉 `row.Queued = true` → 紅；前端拿掉 queued 分支 → 2 條紅。
  - 前端：`pnpm nx test web` 298 檔、4908 測試全綠；`tsc --noEmit` 綠；`pnpm run lint:all` → `✖ 170 problems (0 errors, 170 warnings)`、Prettier 全過。
- 畫面目視比對 k1-d／k1-m：沒有跑（要在本機造出 8 個同時進行中的生成工作）。列的元件與樣式沒改，只換右側文字；由 Vitest 鎖住文字與有無進度條。

### Senior Developer Review (AI) — 2026-10-28

逐檔對照 AC #1–#10 與 git diff（File List 與 `git status` 一致；`.claude/memory/MEMORY.md` 的改動是本 story 之前就在工作區的，不屬於本單）。

- 🔒 Rule 7 Wire Format: N/A（沒有新增 Go 錯誤碼）
- 🔒 Rule 20 Contract Bump: N/A（只新增 `[@contract-v1]`，沒有 bump）
- 🔒 Rule 25 Mega-line: N/A（沒動 project-context.md）

**已修：**

- **[M1] 舊模式的觀察者接線沒有測試**：`StartTranscription` 在 goroutine 裡直接掛觀察者，測試只測了 `setSoloQueued` 本身。→ 抽成 `withSoloQueueObserver(ctx, mediaID, jobID)`，新增 `TestTranscriptionService_RunCtxObserverFlipsTheSoloJob`（在那個 ctx 上回報等待 → 該 job 變排隊中 → 回報結束 → 回到執行中）。
- **[L1] `soloJobRows` 會就地排序呼叫者給的 slice**：兩個來源每次都回新的複本，所以安全；在註解寫明。
- **[L2] 「排隊順序 = 拿到位子的順序」寫得太滿**：音訊 semaphore 與共用 gate 是兩個各自 FIFO 的隊伍，跨隊伍不保證；註解改成「大致」。依 Sally 規則 4 仍以開始排隊時間排序。

**查過、不是問題：**

- 兩個來源會不會列出同一部片（重複的列／React key）：handler 在管線模式一律走 `SoloRunner`（`transcription_handler.go:186`），舊模式才走 `StartTranscription`；管線的聽寫支線在舊服務記成 Solo=false。兩邊互斥，不會重複。
- 等位子的地方是否都有回報：`ExtractGate.Acquire` 只有 `Extractor.Extract` 與 `extractSlotAdapter` 兩處呼叫（`ScriptPeeker` 走 `Extract`）；音訊抽取另有自己的 semaphore，已回報。
- 鎖：觀察者只拿各自服務的 `mu`，抽取路徑上沒有任何人握著這兩把鎖去等位子。新測試在 `-race` 下全過。

**延後（範圍外）：** 首頁計數 → `disc-2026-10-home-inflight-misses-pipeline-solo-jobs`（見 Discovery Triage）。

### Discovery Triage

- ③ `disc-2026-10-home-inflight-misses-pipeline-solo-jobs`（backlog，2026-10-28 建）：首頁「進行中」格在 `main.go` 只接 `transcriptionService`、沒接 `SoloRunner`，管線模式下詳細頁的生成在首頁完全不算；就算算到也是「有就 1」。要先決定首頁格要不要跟活動頁列數一致。

### File List

- `apps/api/internal/services/solo_activity.go`（新）
- `apps/api/internal/services/solo_jobs_activity_test.go`（新）
- `apps/api/internal/services/activity_service.go`
- `apps/api/internal/services/activity_service_test.go`
- `apps/api/internal/services/activity_recent_test.go`
- `apps/api/internal/services/generation_progress_detail_test.go`
- `apps/api/internal/services/audio_extractor_service.go`
- `apps/api/internal/services/transcription_service.go`
- `apps/api/internal/subtitle/solo_runner.go`
- `apps/api/internal/subtitle/solo_runner_test.go`
- `apps/api/internal/subtitle/extractor.go`
- `apps/api/internal/subtitle/extractor_gate_test.go`
- `apps/api/cmd/api/solo_job_source.go`
- `apps/api/cmd/api/solo_job_source_test.go`
- `apps/api/cmd/api/main.go`
- `apps/web/src/services/activityService.ts`
- `apps/web/src/components/activity/ActivityHub.tsx`
- `apps/web/src/components/activity/ActivityHub.spec.tsx`
- `_bmad-output/implementation-artifacts/ux3-2-2-activity-api.md`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`
- `_bmad-output/implementation-artifacts/disc-2026-10-activity-generation-progress-detail.md`（AC drift reference — see Completion Notes；檔案未修改）

## Change Log

| Date | Change |
|---|---|
| 2026-10-28 | 建檔（Bob）；Sally 規則 1–6 寫入 AC |
| 2026-10-28 | CR：修 M1（舊模式觀察者接線補測試）、L1／L2（註解）；狀態 done |
| 2026-10-28 | T1–T5 實作完成：一個單集生成工作一列、排隊中判斷（等抽取位子）、`media_id`／`queued` `[@contract-v1]`；新檔：首頁計數 backlog |
