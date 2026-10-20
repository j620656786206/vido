# Disc：沒有字幕來源也沒設語音辨識 A（後端）——即時事件帶新原因代碼 `no_text_source`，不再落成「其他」

Status: review

**Source:** 傘狀單 `disc-2026-10-no-text-source-skip`（2026-10-19 party mode：founder-pm／Winston／Sally／John／Bob／Mary＋#848 的另一個 session；⚖️ Alexyu：由這邊直接做）。單次生成「片內沒有可用字幕、又沒設語音辨識」被記成 `skipped`，即時事件的 `failure_reason` 卻是 `other`，生成畫面因此寫紅色「失敗 · 生成失敗」。

⚖️ **SM 裁定（Bob，2026-10-19）：**
- **拆單**：後端 5 項、前端 5 項（Story Splitting Rule）。本單是後端；前端 `disc-2026-10-no-text-source-skip-b`。**合併順序：本單先**（單獨上線無害，見 AC #1 最後一條）。
- **在寫紀錄的時候就決定**（照 `disc-2026-10-run-timeout-not-cancel` 的前例），加一個新前綴 `no-text-source: `。理由：skipped 紀錄的 `error_message` 是路由器的英文說明句（見「查到的事」），有四五種寫法，而且有的情況（語言標 `und` 的軌被批次略過）**不是**「沒設語音辨識」——事後比對英文字串分不出來，也不可靠。寫紀錄的當下（`transcribeFallback`）手上同時知道「沒有可用來源」和「語音辨識接不起來」兩件事，判斷一定準。
- **分類仍在同一處**：`models.SubtitleRunFailureReason`（`apps/api/internal/models/subtitle_run_failure.go`）多一條「認前綴」的規則。單次生成的 `soloFailureReason` 本來就先讀紀錄（`solo_runner.go:443-446`），**不用改邏輯**就會得到新代碼。
- **不改** `apps/api/internal/services/transcription_service.go`（`disc-2026-10-generate-keeps-user-en-srt-a` 正在改）。查過：不需要——它只負責丟出 `ErrTranscriptionDisabled`，接住它的是 `process_item.go:577-578`。
- **不碰** `PredictFromTracks`、`SelectAndRoute`、片旁字幕判定（#848 `-b1`／`-b2`）。它們之後走到「沒來源又沒聽寫」時，自然經過同一個 `transcribeFallback`，拿到同一個代碼。

設計稿：**無變更**（後端單）。畫面見 `-b`。

## 查到的事（2026-10-19，讀 `origin/main` 49d96125）

### skipped 的紀錄長什麼樣

- `recordSkip`（`apps/api/internal/subtitle/process_item.go:810-838`）：`run.Status = skipped`、**`run.ErrorMessage = decision.Reason`**（`:821`，註解 `:819-820`「for a skipped run it carries the routing reason, not a failure」），媒體列寫 `no_text_source` 或 `skipped`，再送 `subtitle_progress` `StageSkipped`（`:832`）。
- `decision.Reason` 是路由器的英文句（`apps/api/internal/subtitle/router.go`），四種：
  - `:138` `stream N parsed but no cue survived SDH filtering`（`RouteNoTextSource`）
  - `:205` `N embedded text track(s) present but none tagged Chinese or eng/en — M1 never treats und as English`（`RouteSkip`）
  - `:213` `no usable embedded text subtitle track (N image-codec track(s), 0 text)`（`RouteNoTextSource`）
  - `:411-413` `stream N tagged X: detector returned unrecognized variant … — failing closed (P0)`（`RouteSkip`）
  - 另有 `deferPaidItem`（`:771-772`）寫 `deferred-paid: …`（只有自動生成的 `FreeOnly` 會走到，不是本單的情況）。
- 這些句子都不含分類器認得的關鍵字 → 今天一律 `other`。釘住現況的測試：`solo_runner_test.go:352`（`assert.Equal(t, models.SubtitleRunFailureOther, data["failure_reason"], "…a skip's routing reason names no known cause")`）、`:500-501`。

### 什麼時候是「沒來源又沒聽寫」（只有這裡要加前綴）

- `ProcessItem` 路由後的分支（`process_item.go:230-244`）：
  - `RouteNoTextSource` → `transcribeFallback(…, no_text_source)`（`:235`）；
  - `RouteSkip` ＋ `TranscribeWhenSkipped`（**只有詳細頁單次生成**設，`solo_runner.go:251`；說明在 `pipeline.go:189-196`）→ `transcribeFallback(…, skipped)`（`:242`）；
  - `RouteSkip` 沒設該選項（批次、WorkerPool、自動生成）→ **直接** `recordSkip`（`:244`）。這是「片內有字幕軌但語言認不出來，刻意略過」，**沒有試過聽寫**，不能說「沒設語音辨識」→ **不加前綴**。
- `transcribeFallback`（`:542-583`）裡走到 `recordSkip` 的三種：
  1. `p.asr == nil`（語音辨識沒接上：只能翻譯的建構、或部署沒設 ASR）→ `:551-552` → **加前綴**；
  2. `!transcribable`（影集本身不是檔案）→ 同一個 `if`（`:551`）→ **不加**（不是「沒設語音辨識」；實際上單次與批次都不會送影集進來）。dev 要把這個 `if` 拆成兩個；
  3. `p.asr.Transcribe` 回 `services.ErrTranscriptionDisabled`（沒 ASR 金鑰／沒 ffmpeg，或跑到一半金鑰被刪）→ `:577-578` → **加前綴**。
- 這個判斷跟「誰呼叫」無關：單次、批次、WorkerPool 走到上面 1／3 都加前綴。自動生成（`FreeOnly`）在路由後就先 `deferPaidItem`（`:196-197`），走不到這裡。

### 即時事件出口

- **單次生成**：`SoloRunner.emitTerminal`（`solo_runner.go:374-424`）。skipped 的紀錄落到失敗段（`:410-421`），`failure_reason = soloFailureReason(run, runErr)`（`:420`）；`soloFailureReason`（`:443-468`）**先讀** `run.ErrorMessage`（`:444-446`）→ 有前綴就會分類成新代碼，**不用改程式**。`message` 是「這部影片沒有可用的字幕來源，也沒有設定語音辨識」（`failureMessage` `:507-508`），`error` 是紀錄原文（含前綴，對話框的診斷行顯示，可以接受）。
- **批次**：pipeline 模式的 runner `pipelineGenerationRunner.ExecuteGeneration`（`apps/api/cmd/api/generation_batch_runner_adapter.go:74-92`）看到 skipped 紀錄就回 `fmt.Errorf("media %s: %s: %w", mediaID, outcome.Run.ErrorMessage, services.ErrGenerationItemSkipped)`（`:89-91`）。批次（`apps/api/internal/services/generation_batch.go:789-798`）`errors.Is(err, ErrGenerationItemSkipped) || errors.Is(err, ErrTranscriptionDisabled)` → `markItem(…, failed, reason "skipped", failureReason "", …)`（`:798`）。`failureReason` 參數的規則寫在 `:846-850`（「"" unless status is failed and reason is error」），項目型別在 `:144-156`。
  - 包過的錯誤字串前面多了 `media <id>: `，**不能**對它做 `HasPrefix` → 要用型別帶（見 AC #4）。
  - 舊路徑（legacy／Route C）的 runner（`services/generation_batch_runner.go:37-40`）直接回 `RunTranscription` 的錯；沒設 ASR 時是 `ErrTranscriptionDisabled`。舊路徑**根本不看片內字幕**，說「沒有可用的字幕來源」不精確 → 給既有代碼 `asr_unconfigured`（「語音辨識未設定」）。
- **舊路徑單次**（`TranscriptionService.failJob`）：沒有 skipped，不在本單（也不准動那個檔）。
- **活動頁**：`RecentTerminalRuns` 只撈 `status IN (completed, failed)`（`apps/api/internal/repository/subtitle_run_repository.go:484`），skipped 不列 → 活動頁送出的值不變。`activity_service.go:491` 不用改。
- **其他讀 skipped 紀錄的地方**：`AutoGenerator.excludedMediaIDs`（`auto_generation.go:754-781`）只看 `deferred-paid: ` 前綴，新前綴不影響；`auto_generation.go:845` 同。全 repo 沒有 SQL 用 `error_message` 篩 skipped（`git grep -n -i error_message -- apps/api/internal/repository` 只有讀寫欄位）。

### 合約

- 即時事件的 `failure_reason`：`disc-2026-10-live-activity-failure-reasons-a` AC #1 `[@contract-v1]`（該檔 `:41`，沒有 Change Log 段）。值域今天是 `models.SubtitleRunFailure*` 十一個（`subtitle_run_failure.go:14-26`；v2 的 `queue_timeout` 是在活動頁那份合約升的）。批次規則「`failed`＋`error` 才有代碼，其餘 `""`」寫在：`generation_batch.go:147-150`、`:848-849`、`sse/hub.go:41-44`、前端 `subtitleService.ts:176-183`。
- 新增一個值、批次多一種帶代碼的情況 → Rule 20：live-a AC #1 升 `[@contract-v1→v2]`。
- 下游（Rule 20 🔁 掃描結果，建單時）：`disc-2026-10-live-activity-failure-reasons-b`（done → 凍結，不動）；`disc-2026-10-generation-queue-failure-reason`（ready-for-dev，正在另一個 dev agent 手上；它讀 `failureReason` 但只看 `error`，沒有正式 ack——**保守起見，若本單合併時它還沒 done，一樣 stale-mark**）；`disc-2026-10-no-text-source-skip-b`（本單的前端，照 v2 寫）。#848 `-b1`／`-b2`／`-b3` 還沒有 story 檔，在 #848 傘狀檔附註即可（T5）。

### 既有測試會跟著改的

- `solo_runner_test.go:339-355`（`TestSoloRunner_FailedTerminals` 第一個子測試，`:352` 斷言 `other`）、`:497-510`（`TestSoloFailureReason_RowFirstThenError`，`:500-501`）。
- `process_item_test.go:356-368`＋`:425`（表格：`no usable text source` 斷言 `ErrorMessage == decision.Reason`）。
- `process_item_asr_test.go:207-216`（`TestProcessItem_ASRFallback_NilPortRecordsNoTextSourceAsToday`，`:216`）、`:230`（`…UnavailableServiceDegradesLikeNilPort`）、`:250`（`…RouteSkipNeverTouchesASR`）、`:351`（`…SeriesStillDegradesToNoTextSource`）。
- `process_item_solo_options_test.go`：`TestProcessItem_TranscribeWhenSkippedWithoutASRStillRecordsSkipped`、`TestProcessItem_RouteSkipWithoutTheOptionStillNeverTouchesASR`。
- `cmd/api/generation_batch_runner_adapter_test.go:86-97`（skipped → `ErrGenerationItemSkipped`）。
- `models/subtitle_run_failure_test.go:177-191`（`TestSubtitleRunFailureReason_CodesAreTheContract`，十一個代碼、訊息寫 `[@contract-v2] list`）。
- 不受影響但要確認仍綠：`generation_batch_items_test.go:116`、`:173`（`only non-target tracks` 的一般 skipped → `""`）、`generation_batch_test.go:788`。

## Story

身為只有我一個人在用的 NAS 片庫主人，
我想要「片子沒有可用的字幕、我又沒設語音辨識」這種情況，在即時事件裡有自己的原因代碼，
這樣生成畫面可以告訴我「沒有可用的字幕來源，也沒有設定語音辨識」，而不是一句看不出原因的「生成失敗」。

## Acceptance Criteria

1. **合約 `[@contract-v2]`（升級 `disc-2026-10-live-activity-failure-reasons-a` AC #1 `[@contract-v1]`）：**
   - `failure_reason` 值域多一個 `no_text_source`，共十二個。意思：「這次沒做成，但不是壞掉——片內沒有能用的字幕，語音辨識也接不起來」。
   | 事件 | 什麼時候是 `no_text_source` |
   |---|---|
   | `transcription_failed`（單次生成，`SoloRunner`） | 這次的紀錄是 skipped，而且紀錄有 `no-text-source: ` 前綴（AC #3） |
   | `generation_batch_progress` 的 `changed_item`／終局 `items[]` | `status == "failed" && reason == "skipped"` 且 runner 回的是 `ErrGenerationItemNoTextSource`（AC #4） |
   - 批次 `reason == "skipped"` 的 `failure_reason` 規則改成：`no_text_source`（上表）｜`asr_unconfigured`（舊路徑 `ErrTranscriptionDisabled`）｜`""`（其他 skipped，例如 `und` 軌被刻意略過）。`failed`＋`error` 的規則不變；其他狀態仍是 `""`。
   - 其他鍵、事件類型、`phase`（仍是 `failed`）、`message`、`error` **都不變**。後端只送代碼、不送文字。
   - **活動頁不受影響**：skipped 不列（`subtitle_run_repository.go:484`），`disc-2026-10-activity-subtitle-run-time-a` AC #1 維持 `[@contract-v2]`、不升版。
   - **舊前端相容**：單次遇到不認得的代碼顯示「生成失敗」（跟今天一樣）；批次 `skipped` 項目的代碼舊前端本來就丟掉（`useGenerationJobsFeed.ts:524`）。→ 本單可以單獨上線。
2. **分類函式（`models`）：**
   - 新增 `SubtitleRunFailureNoTextSource = "no_text_source"`、`SubtitleRunNoTextSourcePrefix = "no-text-source: "`。
   - `subtitleRunFailureRules` **第一條**：`strings.HasPrefix(m, SubtitleRunNoTextSourcePrefix)` → `no_text_source`。**只認前綴，不比對路由器的英文句**（本單以前的 skipped 紀錄維持 `other`，已知且接受：它們不會出現在任何畫面上）。
   - 排第一的理由寫進註解：skipped 紀錄的原文是路由器說明，可能剛好含後面規則的字（例如未來的句子含 `timed out`），前綴要先贏。
   - 檔頭（`:5-13`）、規則說明（`:28-52`）、`SubtitleRunFailureReason` 說明（`:127-142`）改寫：「這是唯一一個給 **skipped** 紀錄的代碼；其餘都是 failed 紀錄的原因」；「十一」→「十二」。
3. **寫紀錄時加前綴（`process_item.go`）：**
   - 新增 `NoTextSourceRunPrefix = models.SubtitleRunNoTextSourcePrefix`（比照 `CancelledRunPrefix`／`QueueTimeoutRunPrefix` `:711-735` 的寫法與註解）。
   - `recordSkip` 多一個參數（建議 `runMessage string`），寫進 `run.ErrorMessage`；`emitProgress`（`:832`）與 log 仍用 `decision.Reason`（`subtitle_progress` 的訊息不變）。
   - `transcribeFallback` 裡：`p.asr == nil` → `recordSkip(…, NoTextSourceRunPrefix+decision.Reason)`；`!transcribable` → `recordSkip(…, decision.Reason)`（把 `:551` 的 `||` 拆成兩個 `if`）；`ErrTranscriptionDisabled`（`:577-578`）→ 加前綴。
   - 其他呼叫 `recordSkip` 的地方（`:244` 刻意略過）傳 `decision.Reason`，**行為不變**。
   - 前綴加在 1000 字截斷之前不會被切掉（路由器說明句都很短；`recordSkip` 本來就不截斷）。
4. **批次帶代碼：**
   - `services` 新增 `ErrGenerationItemNoTextSource = fmt.Errorf("%w: no text source and speech recognition unavailable", ErrGenerationItemSkipped)`——**包住** `ErrGenerationItemSkipped`，既有的 `errors.Is(err, ErrGenerationItemSkipped)` 照樣成立。
   - `pipelineGenerationRunner.ExecuteGeneration`（`generation_batch_runner_adapter.go:89-91`）：skipped 紀錄且 `models.SubtitleRunFailureReason(outcome.Run.ErrorMessage) == models.SubtitleRunFailureNoTextSource` → 包 `ErrGenerationItemNoTextSource`；其餘 skipped 照舊包 `ErrGenerationItemSkipped`。（用分類函式判斷，不另寫一次 `HasPrefix`。）
   - `generation_batch.go:789-798`：skipped 分支算 `failureReason`：`errors.Is(err, ErrGenerationItemNoTextSource)` → `no_text_source`；`errors.Is(err, ErrTranscriptionDisabled)` → `asr_unconfigured`；其餘 `""`。`reason` 仍是 `skipped`、`fail_count` 照算（sub-4-2 CR H1 不變）。
   - 註解跟著改：`generation_batch.go:43-51`（`ErrGenerationItemSkipped`）、`:89-92`（`GenerationBatchItemReasonSkipped`）、`:144-150`（`GenerationBatchItemState`）、`:846-849`（`markItem`）；`sse/hub.go:41-44`。
5. **單次生成：** `solo_runner.go` **不改邏輯**。改註解：`soloFailureReason` 上方（`:426-442`）補「skipped 紀錄帶 `no-text-source: ` 前綴時是 `no_text_source`」；`failureMessage` 的 skipped 那條（`:507`）補一行說明 `failure_reason` 是什麼。
6. **測試（每條都要能被下面的「故意改壞」打紅）：**
   - **分類函式**（`subtitle_run_failure_test.go`）：
     - 前綴＋路由器原句（三種形狀：`:213`、`:138`、`:205` 的句子）→ `no_text_source`；
     - 前綴＋含 `timed out`／`context canceled` 的字 → 仍是 `no_text_source`；
     - **沒有前綴**的路由器原句（同三種）→ `other`（釘住「不比對英文句」）；
     - `deferred-paid: …` → `other`（不是本單的情況）；
     - `TestSubtitleRunFailureReason_CodesAreTheContract` 加 `no_text_source`，訊息改成寫明兩份合約（活動頁 `[@contract-v2]`、即時事件 `[@contract-v2]`）。
     - 故意改壞①：新規則排到 `timeout` 後面 → 「前綴＋timed out」那條紅。②：新規則改成 `Contains("no usable")` 之類的英文比對 → 「沒有前綴」那條紅。
   - **寫紀錄**（`process_item_test.go`、`process_item_asr_test.go`、`process_item_solo_options_test.go`）：
     - 表格 `no usable text source`（沒接 ASR）→ `ErrorMessage == NoTextSourceRunPrefix + decision.Reason`；`deliberate skip`（`RouteSkip`、沒設選項）→ `ErrorMessage == decision.Reason` 且**沒有**前綴；
     - `…NilPortRecordsNoTextSourceAsToday`、`…UnavailableServiceDegradesLikeNilPort`（`ErrTranscriptionDisabled`）、`…TranscribeWhenSkippedWithoutASRStillRecordsSkipped` → 有前綴，且 `models.SubtitleRunFailureReason(final.ErrorMessage) == "no_text_source"`；媒體列狀態照舊（`no_text_source`／`skipped`）；
     - `…RouteSkipNeverTouchesASR`、`…RouteSkipWithoutTheOptionStillNeverTouchesASR`、`…SeriesStillDegradesToNoTextSource` → **沒有**前綴、分類 `other`；
     - `subtitle_progress` 的 `StageSkipped` 訊息仍是 `decision.Reason`（沒有前綴）。
     - 故意改壞③：`transcribeFallback` 不加前綴 → 三支「有前綴」紅。④：`:244` 也加前綴 → `deliberate skip` 紅。⑤：`!transcribable` 也加前綴 → series 那支紅。
   - **單次生成**（`solo_runner_test.go`）：
     - `TestSoloRunner_FailedTerminals` 第一個子測試：`failure_reason == "no_text_source"`，**並且等於** `models.SubtitleRunFailureReason(s.runs 裡這次那一列的 ErrorMessage)`（即時事件與紀錄一致）；`message` 不變；`phase` 仍是 `failed`；
     - 新增：`RouteSkip`（`und` 軌）＋沒接 ASR → `no_text_source`；接了 ASR 但 `Transcribe` 回 `ErrTranscriptionDisabled`（模擬跑到一半金鑰被刪）→ `no_text_source`；
     - `TestSoloFailureReason_RowFirstThenError`：沒前綴的 skipped 列仍是 `other`（保留 `:500-501`），另加一列有前綴的 → `no_text_source`。
   - **批次**（`generation_batch_runner_adapter_test.go`、`generation_batch_items_test.go`）：
     - adapter：有前綴的 skipped 紀錄 → `errors.Is(err, ErrGenerationItemNoTextSource)` 且 `errors.Is(err, ErrGenerationItemSkipped)`；沒前綴 → 只有後者。
     - 批次：runner 回包了 `ErrGenerationItemNoTextSource` 的錯 → `changed_item` 是 `failed`／`skipped`／`no_text_source`、`fail_count` 加一；回 `ErrTranscriptionDisabled` → `asr_unconfigured`；回一般 `ErrGenerationItemSkipped` → `""`；終局 `items[]` 同一項同一個值。
     - 故意改壞⑥：skipped 分支一律傳 `""` → 前兩條紅。⑦：adapter 一律包 `ErrGenerationItemSkipped` → adapter 測試紅。
7. **合約文件（Rule 20）：** 見 T5。
8. `apps/api` 下 `go test ./...`、`go vet ./...`、`~/go/bin/staticcheck ./...` 全綠（`lint:all` 不含 staticcheck）。

## Tasks / Subtasks

- [x] **T1 models（AC #2、#6 分類）**：常數、前綴、第一條規則、註解；表格測試與合約清單測試。
- [x] **T2 寫紀錄（AC #3、#6 寫紀錄）**：`NoTextSourceRunPrefix`、`recordSkip` 參數、`transcribeFallback` 拆 `if`＋加前綴；改寫／新增 `process_item*_test.go` 的斷言。
- [x] **T3 單次生成（AC #5、#6 單次）**：註解；`solo_runner_test.go` 兩支改寫＋兩個新案例。
- [x] **T4 批次（AC #4、#6 批次）**：`ErrGenerationItemNoTextSource`、adapter、`markItem` 的 skipped 分支、註解（含 `sse/hub.go`）；adapter 與批次測試。
- [x] **T5 合約與文件（AC #1、#7）**：
  - `disc-2026-10-live-activity-failure-reasons-a.md` AC #1 的戳改 `[@contract-v2]`；檔尾新增 `## Change Log` 段與一列：`| 2026-10-?? | [@contract-v1→v2] AC #1: failure_reason 多一個值 no_text_source（沒有可用的字幕來源、也接不起語音辨識的 skipped 紀錄；單次 transcription_failed 從 other 改成它），批次 reason=skipped 的項目改帶 no_text_source／asr_unconfigured／""（原本一律 ""）；下游：前端 RunFailureReason／FAILURE_COPY_BY_REASON 要多一個鍵、useGenerationJobsFeed 要保留 skipped 的代碼，舊前端照舊顯示「生成失敗」／「沒有可用的字幕來源」（disc-2026-10-no-text-source-skip-a） |`。
  - Rule 20 🔁 下游掃描：`grep -rnE 'confirmed against \`?\[@contract-v[0-9]+\]' _bmad-output/implementation-artifacts/ | grep -E 'live-activity-failure-reasons-a.*AC #1'`，再放寬成 `grep -rn "live-activity-failure-reasons-a" …`。預期：`-b`（done → 凍結）；`disc-2026-10-generation-queue-failure-reason` 若當時不是 done → 在它的 Dev Notes 與 sprint-status 條目加 `⚠️ STALE [@contract-v1→v2]: upstream disc-2026-10-live-activity-failure-reasons-a AC #1 bumped 2026-10-??; re-confirm against v2 before dev — see disc-2026-10-no-text-source-skip-a`；結果寫進 Completion Notes。
  - `disc-2026-10-generate-keeps-user-en-srt-b.md` **檔尾附加**（只加不改，另一個 session 也在動這組檔）：`## 下游註記（2026-10-??，disc-2026-10-no-text-source-skip）`＋一句「-b1／-b2／-b3 走到『沒有可用來源、又沒聽寫』時，沿用 `no_text_source` 代碼（寫紀錄時的 `no-text-source: ` 前綴，`process_item.go` `transcribeFallback`）與前端文字，不另外定義」。
  - `sprint-status.yaml` 本單條目改狀態。

## Dev Notes

### 不要做的事

- **不要改 `apps/api/internal/services/transcription_service.go`**（#848 `-a` 正在改 `runPipeline`／resume／`buildCompleteData`）。若實作中發現非改不可，停下來寫進 Completion Notes，並排在 `disc-2026-10-generate-keeps-user-en-srt-a` 合併之後。
- 不要用路由器的英文說明句判斷（AC #2）；不要用中文 `message` 判斷。
- 不要把刻意略過（`RouteSkip` 沒設 `TranscribeWhenSkipped`，`process_item.go:244`）也標成 `no_text_source`——那時沒有試過聽寫，說「沒設語音辨識」是錯的（見傘狀單「需要決定的事」#4）。
- 不要改 `transcription_failed` 的事件類型或 `phase`（前端 hook 與對話框都靠它；要不要另外有 `skipped` 事件不在裁定範圍）。
- 不要改批次的 `reason` 值、`fail_count` 規則（sub-4-2 CR H1、E2E `generation-workspace.spec.ts:76` 釘住「完成 1 部、失敗 2 部」）。
- 不要動活動頁的查詢（skipped 不列，傘狀單「不做」）。
- 不要碰 `PredictFromTracks`、`SelectAndRoute`、`router.go`、片旁字幕（#848）。
- 不要做資料遷移（舊 skipped 紀錄維持 `other`；畫面不列）。

### 已知但不處理

- `ErrTranscriptionDisabled` 也涵蓋「伺服器缺 ffmpeg」；那種情況前端會說「也沒有設定語音辨識」，不精確。跟今天管理字幕視窗的說法（`solo_runner.go:508`）一樣，本單不分。
- 舊路徑批次（Route C）走到 `ErrTranscriptionDisabled` 給 `asr_unconfigured`（「語音辨識未設定」），不是 `no_text_source`：舊路徑不看片內字幕，說「沒有可用的字幕來源」沒有根據。
- 批次遇到 `und` 軌不聽寫（傘狀單 #4）——另開 backlog。

### 測試標準

- Go：testify、表格測試、不打真的 AI；staticcheck 一定要跑。
- 一致性：單次的事件代碼要**等於**紀錄經分類函式得到的代碼（跟 live-a AC #3 同一種寫法）。

### E2E data-testid／視覺基準

- 後端單，**不影響**任何 testid、E2E、視覺基準。E2E 的 SSE stub 沒有送 `transcription_failed`（`git grep -n "transcription_failed" origin/main -- tests` 無命中）；`generation-workspace.spec.ts:62-64` 的 stub 送的是沒有 `failure_reason` 的 `skipped`，跟後端無關。

### Project Structure Notes

- 動到：`apps/api/internal/models/{subtitle_run_failure.go,subtitle_run_failure_test.go}`、`apps/api/internal/subtitle/{process_item.go,process_item_test.go,process_item_asr_test.go,process_item_solo_options_test.go,solo_runner.go,solo_runner_test.go}`、`apps/api/internal/services/{generation_batch.go,generation_batch_items_test.go}`、`apps/api/cmd/api/{generation_batch_runner_adapter.go,generation_batch_runner_adapter_test.go}`、`apps/api/internal/sse/hub.go`、`_bmad-output/implementation-artifacts/{disc-2026-10-live-activity-failure-reasons-a.md,disc-2026-10-generate-keeps-user-en-srt-b.md（檔尾附加）,sprint-status.yaml}`（＋若需 stale-mark：`disc-2026-10-generation-queue-failure-reason.md`）。
- 沒有 migration、沒有新 endpoint、沒有新事件類型。
- 跟 #848 `-a` 的檔案沒有交集（它只動 `transcription_service.go` 與其測試）。

### Time-dependent visual coverage

- N/A。

### References

- [Source: apps/api/internal/models/subtitle_run_failure.go:5-150]
- [Source: apps/api/internal/subtitle/process_item.go:196-197, 230-244, 542-583, 700-735, 806-838]
- [Source: apps/api/internal/subtitle/router.go:26-42, 133-139, 184-215, 407-414；pipeline.go:189-196]
- [Source: apps/api/internal/subtitle/solo_runner.go:204-216, 246-253, 374-468, 487-514]
- [Source: apps/api/cmd/api/generation_batch_runner_adapter.go:48-92]
- [Source: apps/api/internal/services/generation_batch.go:43-51, 82-97, 144-156, 754-818, 846-875；generation_batch_runner.go:33-40]
- [Source: apps/api/internal/sse/hub.go:34-48]
- [Source: apps/api/internal/repository/subtitle_run_repository.go:475-484]
- [Source: apps/api/internal/subtitle/auto_generation.go:748-806, 845]
- [Source: apps/web/src/hooks/useGenerationJobsFeed.ts:505-525]（舊前端相容）
- [Source: _bmad-output/implementation-artifacts/disc-2026-10-live-activity-failure-reasons-a.md AC #1 `[@contract-v1]`]
- [Source: _bmad-output/implementation-artifacts/disc-2026-10-run-timeout-not-cancel.md]（寫紀錄時加前綴、Rule 20 升版的前例）
- [Source: _bmad-output/implementation-artifacts/disc-2026-10-generate-keeps-user-en-srt-b.md]（#848，下游沿用）
- [Source: project-context.md Rule 20]

## Change Log

| 日期 | 變更 |
|---|---|
| 2026-10-19 | 建單（Bob）。party mode 裁定、⚖️ Alexyu 由這邊直接做；拆 -a／-b。本單把 `disc-2026-10-live-activity-failure-reasons-a` AC #1 升 `[@contract-v1→v2]`（`failure_reason` 加 `no_text_source`、批次 skipped 帶代碼），實際改戳與下游掃描在 T5。 |
| 2026-10-20 | dev 完成（Amelia）：T1–T5 全做；live-a AC #1 已改戳 `[@contract-v2]`＋Change Log；gen-queue stale-mark。 |

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（Amelia，dev）

### Debug Log References

- 七種「故意改壞」逐一套上再跑對應測試，全部變紅（①規則排到 timeout 後、②改成比對 `no usable`、③ transcribeFallback 不加前綴、④ `:244` 也加前綴、⑤ 影集也加前綴、⑥ 批次 skipped 一律 `""`、⑦ adapter 一律包 `ErrGenerationItemSkipped`），還原後全綠。

### Completion Notes List

- **沒有改** `apps/api/internal/services/transcription_service.go`（不需要；`ErrTranscriptionDisabled` 由 `process_item.go` 的 `transcribeFallback` 接住）。
- `recordSkip` 多一個 `runMessage` 參數；`transcribeFallback` 的 `p.asr == nil || !transcribable` 拆成兩個 `if`：影集不加前綴、沒接 ASR 加前綴；`ErrTranscriptionDisabled` 加前綴。刻意略過（`RouteSkip` 沒設選項）照舊傳 `decision.Reason`。`subtitle_progress` 的訊息仍是不帶前綴的路由器原句（測試釘住）。
- 分類函式第一條只認前綴；舊的 skipped 紀錄維持 `other`（畫面不列，已知且接受）。
- 批次：`ErrGenerationItemNoTextSource` 包住 `ErrGenerationItemSkipped`；adapter 用分類函式決定包哪個；`markItem` skipped 分支由新的 `skippedFailureReason(err)` 依錯誤型別給 `no_text_source`／`asr_unconfigured`／`""`。`reason` 仍是 `skipped`、`fail_count` 照算（新測試釘住 3 失敗 1 成功）。`assertInvariants` 放寬成 skipped 項目只准帶這三種值。
- 測試 harness 多記一份 progress 訊息（`progressMsg`），用來釘住 `StageSkipped` 訊息不帶前綴。
- **Rule 20 下游掃描**：`grep 'confirmed against [@contract-vN]' … live-activity-failure-reasons-a AC #1` 只命中 `disc-2026-10-no-text-source-skip-b`（已照 v2 寫）。放寬成 `grep -rln live-activity-failure-reasons-a` 命中：`disc-2026-10-live-activity-failure-reasons-b`（done → 凍結不動）、`disc-2026-10-run-timeout-not-cancel`（done，上游單，不受影響）、`disc-2026-10-no-text-source-skip-b`（本單前端，已照 v2 寫）、sprint-status。`disc-2026-10-generation-queue-failure-reason` 雖沒直接引用，仍照 story 保守 stale-mark（ready-for-dev、未 done）：Dev Notes 開頭與 sprint-status 條目都加了 `⚠️ STALE [@contract-v1→v2] … see disc-2026-10-no-text-source-skip-a`；沒有碰它的分支 `feat/generation-queue-failure-reason`。
- `disc-2026-10-generate-keeps-user-en-srt-b.md` 檔尾只附加「下游註記」一段，沒改其他內容。
- 驗證：`go test ./...`（apps/api 全部）、`go vet ./...`、`~/go/bin/staticcheck ./...` 全綠；`pnpm run lint:all` → `✖ 170 problems (0 errors, 170 warnings)`（都是既有警告）；`pnpm run format:check` 通過。
- 舊前端相容已確認：`failureCopyForReason` 不認得的代碼回「生成失敗」；`useGenerationJobsFeed.ts:524` 只保留 `reason === 'error'` 的代碼。

### File List

- apps/api/internal/models/subtitle_run_failure.go
- apps/api/internal/models/subtitle_run_failure_test.go
- apps/api/internal/subtitle/process_item.go
- apps/api/internal/subtitle/process_item_test.go
- apps/api/internal/subtitle/process_item_asr_test.go
- apps/api/internal/subtitle/process_item_solo_options_test.go
- apps/api/internal/subtitle/solo_runner.go
- apps/api/internal/subtitle/solo_runner_test.go
- apps/api/internal/services/generation_batch.go
- apps/api/internal/services/generation_batch_items_test.go
- apps/api/internal/sse/hub.go
- apps/api/cmd/api/generation_batch_runner_adapter.go
- apps/api/cmd/api/generation_batch_runner_adapter_test.go
- _bmad-output/implementation-artifacts/disc-2026-10-no-text-source-skip-a.md
- _bmad-output/implementation-artifacts/disc-2026-10-live-activity-failure-reasons-a.md
- _bmad-output/implementation-artifacts/disc-2026-10-generation-queue-failure-reason.md
- _bmad-output/implementation-artifacts/disc-2026-10-generate-keeps-user-en-srt-b.md
- _bmad-output/implementation-artifacts/sprint-status.yaml
