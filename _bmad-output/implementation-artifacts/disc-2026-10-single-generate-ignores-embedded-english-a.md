# Disc（後端）：單集／電影的「生成字幕」按鈕改走字幕管線——先看片內字幕，沒有才聽聲音

Status: done

**Source:** Alexyu 2026-10-06 party mode（See S01E02 實測，清單第 1 項，8 張裡排第一）。同日 Alexyu 裁定**選項 1（長解）**：按鈕改走批次用的 `Pipeline.ProcessItem`，不在舊路上補一步判斷；**電影一起修**。本單是後端半張，前端半張見 `-b`（依賴本單）。實測紀錄：`eval-see-s01e02-asr-vs-official.md`「討論會」一節。

> ⚖️ 本單推翻 2026-08-06 裁定 A（「單集按鈕永遠不讀片內字幕」，`transcription_estimate.go:26-30`）。推翻理由：S01E02 的時間不準、人名亂掉、重複句被刪、sun→son，根源都是「明明有官方英文字幕卻去聽聲音」，而且還多付語音辨識 $0.34。

## Story

身為 Vido 的使用者，
我希望在單集或電影上按「生成字幕」時，Vido 跟批次生成一樣先看片子裡有沒有字幕——有中文直接用（免費）、有英文就翻英文、什麼都沒有才聽聲音，
這樣字幕時間準、人名對、不漏句，而且不用多付一筆語音辨識的錢。

## 查到的事（2026-10-06，Bob 逐一開檔確認；只列本單會動到或依賴的）

### 單集／電影按鈕現在走哪裡

- **路由**（`transcription_handler.go:90-103`）：`POST /movies/:id/transcribe`、`POST /episodes/:id/transcribe`、兩個 `GET …/transcribe/status`、兩個 `GET …/transcribe/estimate`（估價路由只在 `SetEstimator` 被呼叫時掛上）。
- **Handler 只認識 `TranscriptionServiceInterface`**（`:33-46`：`IsAvailable`／`CanResumeTranslateOnly`／`CanResumeEpisodeTranslateOnly`／`IsInProgress`／`StartTranscription`）。
- **流程**：空 id → 400；`!IsAvailable() && !CanResume…` → 503 `TRANSCRIPTION_DISABLED`「語音辨識尚未設定」（`:122-133`）；`lookupMovieFile`（`:284-304`：404／沒路徑或檔案不在 → 400）；`IsInProgress` → 409（`:141`）；電影帶 `?translate=true` 才翻（`:150`）、單集一律翻＋`WithMediaType(episode)`（`:236-242`）；`StartTranscription` 回 `job_id`，202（`:174-180`）。`ErrTranscriptionTargetNotWritable` → 409 `SUBTITLE_TARGET_NOT_WRITABLE`（`:164`、`:274-278`）。
- **沒有讀任何 request body**：model、預算、同意都不是客戶端給的。
- **`GET …/transcribe/status`**（`:359-364`）只回 `{in_progress}`，來源是 `TranscriptionService.IsInProgress`（`transcription_service.go:470-475`，一張記憶體 map）。
- **`StartTranscription`**（`transcription_service.go:481-518`）：閘門 → `checkTargetWritable` → `resolveActivityTitle` → `acquireJob(solo=true)`（`:678-687`，產生 job UUID）→ 用 `context.Background()` 起 goroutine 跑 `runPipeline`。
- **`runPipeline`**（`:824-`）：自己開 `ai.Budget`（`resolveBudget`；只有「自己開的」才 `trackRunBudget`，SSE 才帶 `spent_usd`／`budget_usd`，`costFields :607-627`）；開帳本列 **`Route` 永遠寫 `asr`**（`transcription_ledger.go:73-83`，連續跑只翻譯也寫 asr）；`tryTranslateOnlyResume`；抽音訊 → 語音辨識 → 寫 `.en.srt` → `translateSRT` → `placer`；最後 `broadcastEvent(transcription_complete, buildCompleteData…)`（`:1007`、`:1017-1048`：`job_id`、`media_id`、`srt_path`、`zh_srt_path`、`partial`、`english_kept_blocks`、`message`「轉錄完成」）。
- **事件**：`transcription_extracting`／`transcription_progress`／`translation_progress`／`transcription_complete`／`transcription_failed`（`:106-110`）。`title` 只有 solo 才帶（`eventTitle :757-765`）；`costFields` 只在該 run 自己開預算時加 cost key（`:607-627`、`carriesRunCost :2179-2186`）。`ActivityProgress` 只數 solo（`:733-744`）。

### 批次／自動生成走哪裡（我們要搬過去的那條）

- **`Pipeline.ProcessItem(ctx, MediaRef, ProcessItemOptions) (*ProcessOutcome, error)`**（`process_item.go:34`）。`ProcessItemOptions{Force, FreeOnly, ModelID, Automatic}`（`pipeline.go:141-180`，`[@contract-v1]`，FreeOnly 是「additive 不升版」的前例）。`ProcessOutcome{Run, Kind, SubtitlePath}`（`:191-195`）。
- 流程：`Load` → 名詞表／在地化 → `ai.WithModelID(ctx, opts.ModelID)`（`process_item.go:65`）→ **`preflightSkip`**（`pipeline.go:771-815`：有可用的 `.zh-Hant.srt` 就直接結束，除非 `Force`、或指定了 model 且找不到同版本完成紀錄、或上次是部分交付）→ ctx 沒預算就掛 `ai.NewBudget(p.runBudgetUSD)`（`process_item.go:104-106`）→ run row → 媒體列 `extracting` → **`SelectAndRoute`**（`:176`）→ `run.Route = decision.Kind`（`:184`）→ FreeOnly 煞車 → **寫入探測** `probeWritable`（`:216-226`，在路由之後）→ `switch decision.Kind`（`:230-245`）：`RouteNoTextSource` → `transcribeFallback`；**`RouteSkip` → `recordSkip`（不會聽聲音）**；三個抽字幕路線 → `deliverable` → `placer.Place` → 媒體列 `found`。
- **`transcribeFallback`**（`:529-597`）：`p.asr == nil` 或不是 movie/episode → `recordSkip(no_text_source)`；否則 `p.asr.Transcribe` → `pipelineASRAdapter.Transcribe`（`cmd/api/asr_adapter.go:30-39`）→ `RunTranscription(WithTranslation, WithMediaType, WithRunRecordedByCaller)`（同步、`solo=false`、不開第二張帳本列）。`ErrTranscriptionDisabled` → skip(no_text_source)；`ErrBudgetExceeded` → `pauseASRItem`；其他 → `failItem`。成功後 `run.Route = asr`、記 sidecar（`:571-590`），`emitProgress(StageComplete)`。
- **`failItem`**（`:819-864`）：run `failed`、媒體列退回 `not_searched`、`emitProgress(StageFailed)`。
- **D6 事件**：`subtitle_progress`，payload 只有 `{media_id, media_type, stage, message}`（`progress_sse.go:62-75`）；**沒有** `job_id`／`zh_srt_path`／cost／`title`。每個 run 終點另發 `subtitle_run_receipt`（`NewSSEReceiptHook :134-141`，payload＝`SubtitleRun.ReceiptPayload()`：`run_id、media_id、media_type、status、model_id、cue_count`＋選填 `route、batch_id、cache_hit_cues、spent_usd、budget_usd、completed_at`，`models/subtitle_run.go:195-223`）。
- **同一時間只處理一次**：`WorkerPool.inFlight`；`TryReserve`／`Release`（`worker_pool.go:293-308`）是給「不走佇列、直接呼叫 `ProcessItem`」的外部呼叫者用的（批次就是這樣用，`cmd/api/generation_batch_runner_adapter.go:74-93`）。`EnqueueItem`（`:252-290`）是佇列用。目前**沒有** `IsInFlight(ref)` 這種查詢方法。
- **只在 `pipeline` 模式有**：`VIDO_SUBTITLE_PIPELINE_MODE` 預設 `legacy`（`config.go:235`）；`subtitlePipeline`／`subtitlePipelinePool` 只在 `cfg.SubtitlePipelineEnabled()` 內建立（`main.go:798`、`:839`）；批次在 legacy 用 `RouteCGenerationRunner`（純語音辨識，`:1089-1097`）。Claude 金鑰閘門 `subtitleCapabilityGate`（`:790`）。
- **Rule 19**（`project-context.md:532-560`）：`handlers → subtitle → services` 合法；`services ↛ subtitle`。前例：`subtitle_pipeline_handler.go` 直接用 `subtitle.MediaRef`、`subtitle.ProcessItemOptions`（`:138-143`）；`cmd/api/*_adapter.go` 是橋接層。

### 估價現在怎麼算

- `TranscriptionEstimate`（`transcription_estimate.go:31-54`，dsr-6a AC #2 `[@contract-v1]`）：`media_id、media_type、plan（full｜translate_only，`:14-19`）、asr_available、self_hosted_asr、translation_configured、model_id、runtime_minutes、runtime_known、runtime_source、estimated_usd`。
- `Estimate`（`:125-153`）：plan 只看 `canResumeTranslateOnly`；`priceUSD`（`:157-174`）：full＋翻譯 → `estimateUSD(RouteASR,…)`；只翻譯 → `asrLegTranslationRatePerMinute`。
- **已經在探測、卻把路線丟掉**：沒有存片長時 `_, seconds, err := s.prober.ProbeWithDuration(ctx, target.FilePath)`（`:189`）。prober 是 `routePredictorAdapter`（`cmd/api/route_predictor_adapter.go:44-50`）→ `Router.PredictRouteWithDuration`（`predict_route.go:60-71`）→ `PredictFromTracks`（`:82-98`）只回 `extract｜asr｜skip`，抽字幕三種路線在探測階段分不出來（`:10-15` 寫明）。`estimateUSD(RouteExtract)` 以「LLM 翻譯」計價（`generation_candidates.go:1052-1064`）——中文軌其實免費，所以是**往上報**（符合 `translationCalibrationModel` 的「寧可往上給」原則）。
- 電影有 `subtitle_tracks` 欄位（persisted probe，`PredictFromTracks` 的註解 `predict_route.go:73-77`）；單集沒有。

### 前端會受影響（給 -b 用，本單不改前端）

- 說明文字 `DEFAULT_LINE`「語音辨識＋AI 翻譯，約需數分鐘」等（`generateCostView.ts:48-58`）；`plan !== 'translate_only' && !asrAvailable` 就封鎖按鈕（`:113-115`）。
- 進度條凍結的六格「提取音訊 → 轉錄中 → 翻譯中 → 簡轉繁 → AI校正 → 完成」（`GenerationProgressV2.tsx:32-39`）；D6 `probing/extracting` 對到「提取音訊」（`useGenerationProgress.ts:197-201`）；D6 `complete` 被當終點、但沒有 `zh_srt_path` → 對話框會寫「已生成英文字幕；尚未翻譯」（`ManageSubtitleDialogV2.tsx:751-755`）；`startTracking(mediaId)` 只吃 media id（`useGenerationProgress.ts:358-369`）；`costTexts` 要 `spent_usd`＋`budget_usd` 都有才顯示（`ManageSubtitleDialogV2.tsx:1108-1110`）。
- 工作區的單一任務只從帶 `title` 的 `transcription_*` 事件來（`useGenerationJobsFeed.ts:22-35`、`:316-321`）。

### 其他查到的事

- `MediaItem.Context` 是 `TranslateContext{Title, OriginalTitle, Year, Genres, Overview…}`（`pipeline.go:83-87`）→ solo run 的 `title` 直接用 `item.Context.Title`（單集會是劇名；要「劇名 S01E02」的話另接 `titleResolver`，見 §5）。SoloRunner 要先 `pipeline.media.Load(ctx, ref)` 拿 `FilePath` 與 `Context`。
- `TrackRouter` 介面只有 `SelectAndRoute`（`pipeline.go:267-269`），**沒有** `PredictRoute` → SoloRunner 的能力閘門要另外注入 `predict func(ctx, path) (RoutePrediction, error)`（main.go 給 `routePredictor` 那個 adapter 包的 `*subtitle.Router`，或直接給 `subtitleRouter.PredictRoute`）。
- `probeWritable` 是 `Pipeline` 的**欄位**（`pipeline.go:338-342`，預設 `fsprobe.ProbeWritableContext :517`，`WithWritableProbe :459-460` 可換），`process_item.go:217` 呼叫 `p.probeWritable(ctx, dir)`。同包的 SoloRunner 直接用 `pipeline.probeWritable`，永遠非 nil。
- `services.RoutePredictor`（`generation_candidates.go:53-58`：`FromTracks`／`Probe`）與 `RouteDurationPredictor`（`:70-72`）是兩個介面；`routePredictorAdapter` 兩個都滿足（`route_predictor_adapter.go:26-50`）。估價現在只收 `SetDurationProber`；加 `FromTracks` 時可用型別斷言或再加一個 setter 收同一個 adapter。

### ⚠️ 未查證（dev 開工時先確認）

- `ai.Budget` 在 ctx 上時，`transcribeFallback` → `RunTranscription` 的 `transcription_*` 事件**不帶 cost key**（`resolveBudget` 回 `ownBudget=false`）。本單用 solo 自己的終點事件補 cost，不改 `costFields`。

## 設計（Winston 2026-10-06，依 Alexyu 裁定選項 1）

**一句話：** 新增 `subtitle.SoloRunner`——「一次點擊」的工作層包裝，底下呼叫批次同一個 `Pipeline.ProcessItem`。Handler 多一個可選的 port；pipeline 模式接 SoloRunner，legacy 模式維持舊路。

### 1. `subtitle.SoloRunner`（新檔 `internal/subtitle/solo_runner.go`）

放在 `subtitle` 套件：它需要 `Pipeline`（含未匯出的 `probeWritable`）、`MediaRef`、`ProcessItemOptions`、`WorkerPool` 的 `TryReserve/Release`、`ProgressBroadcaster`——全在同一包；handlers 可以 import subtitle（Rule 19）。

```go
type SoloRunner struct { pipeline *Pipeline; guard interface{TryReserve(MediaRef) bool; Release(MediaRef)}; hub ProgressBroadcaster; asrAvailable func() bool; canResume func(ctx, ref) bool; title func(ctx, ref) string; runBudgetUSD float64; now func() time.Time; mu; jobs map[MediaRef]*soloJob }
type soloJob struct { ID, Title string; Budget *ai.Budget; StartedAt time.Time }

func (s *SoloRunner) Start(ctx context.Context, ref MediaRef, modelID string) (jobID string, err error)
func (s *SoloRunner) Status(ref MediaRef) (inProgress bool, jobID string)
```

**`Start` 同步做（在回 202 之前）：**
1. `guard.TryReserve(ref)`；false → 回 `services.ErrTranscriptionInProgress`（handler 原本就把它對到 409 `TRANSCRIPTION_IN_PROGRESS`，不用改碼）。之後任何錯誤都要 `Release`。
2. **能力閘門（取代今天的 503 條件）**：用 `pipeline.router.PredictRoute(ctx, filePath)` 探一次（只 ffprobe、不抽軌；估價已經在做同一件事）。預測是 `asr` 或 `skip`、而且 `!asrAvailable()`、而且 `!canResume(ctx, ref)` → 回 `services.ErrTranscriptionDisabled`（handler 照舊 503）。預測是 `extract` → 放行（路由本身免費；英文軌沒翻譯金鑰時由管線回 `failed`，錢沒花）。探測失敗 → 不擋（Rule 13 case 3：放行讓管線自己判，不要因為探不到就 503 說「沒設定語音辨識」）。
3. **寫入探測**：`pipeline.probeWritable(ctx, filepath.Dir(filePath))`；失敗回 `ErrSubtitleTargetNotWritable`（handler 要把 `subtitle.ErrSubtitleTargetNotWritable` 也對到 409 `SUBTITLE_TARGET_NOT_WRITABLE`，見 §3）。今天是在付費前擋、同步回 409，要保留；管線內那次探測會再跑一次，沒關係。
4. 建 job：`uuid`、`title(ctx, ref)`、`ai.NewBudget(runBudgetUSD)`；寫進 `jobs`。
5. **立刻發一個工作層起點事件** `transcription_extracting`：`{job_id, media_id, media_type, title, phase:"extracting", message:"正在檢查片內字幕…", predicted_route, spent_usd:0, budget_usd}`。這讓對話框和工作區在管線的 D6 事件（沒有 title／job_id）到來前就知道「這個 job 開始了」。
6. 起 goroutine（`context.Background()` ＋ cancel，跟 `StartTranscription` 一樣脫離 request）：`ctx = ai.WithBudget(ctx, job.Budget)`，`outcome, err := pipeline.ProcessItem(ctx, ref, ProcessItemOptions{Regenerate: true, TranscribeWhenSkipped: true, ModelID: modelID})`。回來後 `Release`、刪 `jobs[ref]`、發終點事件（§1b）。

**1b. 終點事件（solo 的 job 層，權威）**

- 成功（`err == nil` 且 `outcome.Run.Status == completed`）→ `transcription_complete`：`{job_id, media_id, media_type, title, phase:"complete", route: outcome.Run.Route, zh_srt_path: outcome.SubtitlePath（空就不放 key）, cue_count, spent_usd, budget_usd, message}`，`message` 依路線：`deliver_direct`「字幕已生成（直接使用片內中文字幕，沒有花錢）」、`convert_then_deliver`「字幕已生成（片內簡體字幕已轉成繁體，沒有花錢）」、`translate`「翻譯完成（翻譯片內英文字幕）」、`asr`「轉錄完成」。部分交付（`outcome.Run.TransientCount != nil && > 0`）加 `partial:true, english_kept_blocks`。
- 失敗（`err != nil`，或 run 是 `skipped`／`failed`）→ `transcription_failed`：`{job_id, media_id, media_type, title, phase:"failed", route（有就放）, error, message, spent_usd, budget_usd}`。`ErrBudgetExceeded` 的 message 要說「達到單次費用上限 $X，已停在半路；已翻好的部分有保留」；`skipped`＋`no_text_source` 說「這部影片沒有可用的字幕來源，也沒有設定語音辨識」。
- **為什麼不另發新事件型別：** 對話框與工作區已經聽 `transcription_complete/failed`。語音辨識那條路時，`TranscriptionService` 會先發它自己的 `transcription_complete`（不同 `job_id`、沒 cost），管線再發 D6 `complete`，最後才是 solo 的這一發——**前端（-b）以「`job_id` 等於 POST 回的那個」判定哪一發是終點**。本單的契約：**solo job 的終點事件一定帶該 `job_id`，而且是該 job 最後一個事件。**
- 不改 `TranscriptionService.costFields`（它刻意只在自己開預算時帶 cost，批次共享預算會誤報成單片花費）。

### 2. `ProcessItemOptions` 兩個新欄位（additive，v1 不升版——`FreeOnly` 前例）

- `Regenerate bool`：**只**跳過 `preflightSkip`，**不**跳過 segment-cache 讀取。`Force` 兩者都跳；使用者按「生成字幕」是「我要重做」，但已翻過的句子沒理由再付一次。`preflightSkip` 開頭加 `if opts.Regenerate { return false, "regenerate: pre-flight bypassed, cache reads kept" }`。
- `TranscribeWhenSkipped bool`：`RouteSkip`（有文字軌但沒標 zh／eng，例如 `und`）時改走 `transcribeFallback`，不 `recordSkip`。批次／自動生成預設 false、行為不變；solo 給 true，因為今天按鈕遇到這種片會聽聲音，不能變成「已略過」。估價那邊 `skip` 比照 `asr` 計價（§4）。

### 3. Handler：多一個可選 port

```go
type SoloGenerator interface {
    Start(ctx context.Context, mediaType, mediaID, filePath string, modelID string) (jobID string, err error)
    Status(mediaType, mediaID string) (inProgress bool, jobID string)
}
func (h *TranscriptionHandler) SetSoloGenerator(g SoloGenerator)
```

- `NewTranscriptionHandler` 簽章不動（既有測試與 fake 全部有效）。`SetSoloGenerator` 跟 `SetEstimator` 同款：setter，在 `RegisterRoutes` 之前呼叫。
- `TranscribeMovie`／`TranscribeEpisode`：有 solo → 跳過 `IsAvailable`／`CanResume` 閘門（SoloRunner 自己判）、仍做 `lookup*File`、呼叫 `solo.Start`；錯誤對應：`services.ErrTranscriptionInProgress` → 409（既有）、`services.ErrTranscriptionDisabled` → 503（既有文案）、`services.ErrTranscriptionTargetNotWritable` **或** `subtitle.ErrSubtitleTargetNotWritable` → 409 `targetNotWritableError`。回應 `{job_id, message}` 不變。沒 solo → 一字不改走舊路。
- 電影的 `?translate=true`：走 solo 時忽略（管線一律產中文；今天對話框永遠帶 `translate=true`，`transcriptionService.ts:100`）。handler 註解寫明。
- `TranscriptionStatus`：有 solo → `in_progress, job_id = solo.Status(...)`；沒 solo → 舊的 `IsInProgress`。`TranscriptionStatusResponse` 加 `JobID string \`json:"job_id,omitempty"\``（additive）。
- `SoloRunner.Status`：`jobs` 有 → `(true, id)`；否則 `guard.IsInFlight(ref)`（**新增** `WorkerPool.IsInFlight(ref) bool`，讀 `inFlight`）→ `(true, "")`（批次或佇列正在跑它，沒有 solo job id）；都沒有 → `(false, "")`。

### 4. 估價：知道路線、價錢分路線

- `TranscriptionEstimate` 加 `Route string \`json:"route,omitempty"\``（`extract｜asr｜skip`，不知道就不放）；`plan` 多一個值 `TranscriptionPlanExtract = "extract"`。
- `TranscriptionEstimateTarget` 加 `SubtitleTracks []SubtitleTrack`（電影從 `movie.SubtitleTracks` JSON 解出來給它；單集沒有就空）。
- `Estimate`：`routingEnabled`（新 setter `SetRoutingEnabled(bool)`，main.go 只在 pipeline 模式設 true；legacy 保持今天一模一樣）為 true 時：有 tracks → `PredictFromTracks`（透過既有 `RoutePredictor.FromTracks` port）；沒有 → `ProbeWithDuration` 的第一個回傳值**不要再丟掉**（`:189`）。路線決定 plan：`extract` → `plan=extract`；否則既有邏輯（`translate_only`／`full`）。
- `priceUSD`：`plan=extract` → `estimateUSD(RouteExtract, minutes, _, model)`（跟批次候選清單同一個函式、同一個往上報的立場）；`skip` 比照 `asr`。`asr_available` 照舊回報；**封不封鎖按鈕由前端依 `route` 決定**（-b）。
- **Rule 20**：dsr-6a AC #2 `[@contract-v1→v2]`——`plan` 多一個值、多 `route` 欄位、`GET …/transcribe/status` 多 `job_id`。Change Log 要有「改了什麼／下游壞什麼」兩段。下游：`-b`（`transcriptionService.ts:74-87` 的 `plan` union 要加 `'extract'`、要 ack v2）。grep 下游 ack：`grep -rnE 'confirmed against .?\[@contract-v' _bmad-output/implementation-artifacts/ | grep dsr-6a` → 只有 `transcriptionService.ts` 的註解與 dsr-6a 自己；沒有 not-done 的 story 需要 stale-mark，但 -b 開單時要寫 ack v2。

### 5. 接線（`cmd/api/main.go`）

- pipeline 模式（`subtitlePipeline != nil`）：`subtitle.NewSoloRunner(subtitlePipeline, subtitlePipelinePool, sseHub, subtitleRouter.PredictRoute, pipelineASR.Available, canResume, cfg.AIRunBudgetUSD)`（`titleResolver` 可選，預設用 `item.Context.Title`），`transcriptionHandler.SetSoloGenerator(solo)`；`transcriptionEstimateService.SetRoutingEnabled(true)`。
  - `canResume`：包 `transcriptionService.CanResumeTranslateOnly`／`CanResumeEpisodeTranslateOnly`（依 `ref.MediaType`）。
  - `titleResolver`：見 ⚠️ 未查證；可用 `movieService.GetByID`／`repos.Episodes.FindByID` 組「劇名 S01E02 集名」，失敗回 ""。
- legacy 模式：什麼都不接；按鈕行為與今天完全相同。`docs/deployment.md` 的 `VIDO_SUBTITLE_PIPELINE_MODE` 那一列補一句：「`legacy` 時，單集／電影的「生成字幕」只會聽聲音，不會用片內字幕」。

### 6. 範圍外（故意不做）

- `TranscriptionService.costFields` 讀 ctx 預算——不做（會把批次總額誤報成單片）。
- 前端全部（-b）。
- `RouteCache`（sub-5-4）接進估價——可做可不做；估價一次只探一部片、而且已經在探，收益小。
- 語音辨識那條路的音樂符號（`♪♪`）句——另立 `disc-2026-10-asr-music-only-cues`（Alexyu 2026-10-06 提出；見 Discovery Triage）。

## Acceptance Criteria

1. **pipeline 模式下，單集與電影的 POST 都走管線：** 用 fake pipeline 驗證 `ProcessItem` 被呼叫，`ref.MediaType` 分別是 `episode`／`movie`，`opts.Regenerate == true`、`opts.TranscribeWhenSkipped == true`、`opts.ModelID` 為空；回 202 `{job_id, message}`（形狀不變）。
2. **legacy 模式一字不改：** 不呼叫 `SetSoloGenerator` 時，`transcription_handler_test.go`、`transcription_episode_handler_test.go`、`transcription_status_handler_test.go`、`transcription_estimate_handler_test.go` 現有測試全部不改期望值照常通過。
3. **單飛／409：** 同一部片第二次 POST（solo 自己在跑、或 `WorkerPool.IsInFlight` 為 true）→ 409 `TRANSCRIPTION_IN_PROGRESS`。跑完後 `Release`，第三次可再開。
4. **能力閘門：** 預測 `asr`／`skip` ＋ 沒語音辨識 ＋ 不能續跑 → 503 `TRANSCRIPTION_DISABLED`（文案同今天）；預測 `extract` ＋ 沒語音辨識 → **放行** 202；探測失敗 → 放行。
5. **寫入探測在付費前：** 資料夾不可寫 → 同步 409 `SUBTITLE_TARGET_NOT_WRITABLE`，`ProcessItem` 沒被呼叫，`jobs` 沒殘留、guard 已釋放。
6. **事件契約：** 起點 `transcription_extracting` 帶 `job_id/media_id/media_type/title/phase/message/predicted_route/spent_usd/budget_usd`；終點 `transcription_complete`／`transcription_failed` 帶 `job_id` 且是該 job 最後一發；四種成功路線的 `message` 與 `route` 各一個測試；`zh_srt_path` 只在 `outcome.SubtitlePath` 非空時出現；部分交付帶 `partial`＋`english_kept_blocks`；`ErrBudgetExceeded` 與 `skipped/no_text_source` 的 failed 文案各一個測試。
7. **`Regenerate`：** 有可用 `.zh-Hant.srt` 時 `ProcessItem` 不早退（`preflightSkip` 回 false），**且** segment cache 的讀取仍發生（用既有的 fake cache 驗證 `GetMany` 被呼叫）；`Force` 行為不變；兩者都 false 時 `preflightSkip` 行為不變（既有測試保護）。
8. **`TranscribeWhenSkipped`：** `RouteSkip` ＋ true → 走 `transcribeFallback`（fake asr 被呼叫、`run.Route == asr`）；false → 既有 `recordSkip`（既有 `process_item_test.go` 不改）。
9. **`Status`：** solo 在跑 → `{in_progress:true, job_id:"…"}`；只有 guard 在跑 → `{in_progress:true}` 且沒有 `job_id` key；都沒有 → `{in_progress:false}`。`tests/e2e/transcription-status.api.spec.ts` 仍過（它只斷言 `in_progress:false` 的形狀）。
10. **估價分路線：** `routingEnabled=true`：有 tracks（含 eng 文字軌）→ `plan=extract, route=extract`，`estimated_usd == estimateUSD(RouteExtract…)`；沒 tracks、探測回 `asr` → 既有 `full` 價；探測回 `skip` → `route=skip`、`plan=full`、`asr` 價；`untranslated` 可續跑 ＋ 路線 `asr` → `translate_only`（既有）；**路線 `extract` 時不回 `translate_only`**。`routingEnabled=false`：`transcription_estimate_test.go` 現有測試不改期望值照常通過，回應裡**沒有** `route` key。
11. **Rule 20：** `transcription_estimate.go` 的 `[@contract-v1]` 改 `[@contract-v2]`，本單 Change Log 有 `[@contract-v1→v2] AC #2` 列（改了什麼＋下游壞什麼）；`-b` 的 story 寫 ack v2。
12. **文件：** `docs/deployment.md` 補 legacy 的限制一句（EN＋zh-TW 兩份都要，bilingual rule）。
13. **檢查全綠：** `apps/api` 跑 `go build ./... && go vet ./... && go test ./...`、`~/go/bin/staticcheck ./...`、`pnpm run lint:all`、`pnpm nx test web`（前端不動，但要跑完整回歸）。
14. **NAS 實測（部署後補，不擋 PR）：** 對 See S01E02 按單集「生成字幕」：log 的 `subtitle route decided` 要選到第 8 軌、`forced subtitle cues merged…` 併 1 句、帳本 `route=translate`、花費沒有語音辨識那 $0.34。

## Tasks / Subtasks

- [x] T1 `ProcessItemOptions.Regenerate`／`TranscribeWhenSkipped`＋`WorkerPool.IsInFlight`（AC #7、#8、#9 的 guard 半）
- [x] T2 `subtitle.SoloRunner`：Start（閘門、寫入探測、job、起點事件、goroutine）、Status、終點事件（AC #1、#3、#4、#5、#6、#9）
- [x] T3 Handler：`SoloGenerator` port＋`SetSoloGenerator`、兩個 Transcribe 與 Status 的分流、`subtitle.ErrSubtitleTargetNotWritable` 對應（AC #1、#2、#3、#4、#5、#9）
- [x] T4 估價：`route`／`plan=extract`／`SetRoutingEnabled`／`SubtitleTracks`、`priceUSD` 分路線、不再丟掉探測結果（AC #10、#11）
- [x] T5 `main.go` 接線（pipeline 模式才接）＋`titleResolver`／`canResume`；`docs/deployment.md`（AC #12）
- [x] T6 測試：handler 新 fake、SoloRunner 單元測試（fake pipeline／guard／hub）、estimate 新案例、process_item 新案例（AC #1–#11）
- [x] T7 檢查＋sprint-status（AC #13）

後端 7 項、前端 0 項 → 本單不再拆；前端另見 `-b`。

## Dev Notes

### 不要做的事

- **不要改 `TranscriptionService` 的 `StartTranscription`／`runPipeline`／`costFields`。** legacy 模式與批次（legacy）還靠它；本單只是不再從按鈕呼叫它（pipeline 模式）。
- **不要讓 `services` import `subtitle`**（Rule 19 會循環）。SoloRunner 在 `subtitle`，handler 直接用；需要 `TranscriptionService` 的能力（續跑判斷）用函式 port 從 main.go 注入。
- **不要用 `Force`** 代替 `Regenerate`：`Force` 會跳過 segment-cache 讀取，重做一次要再付全額翻譯費。
- **不要把 `RouteSkip` 一律改成聽聲音**：只在 `TranscribeWhenSkipped` 為 true；批次／自動生成的「已略過」語意（`worker_pool.go:312-333` 的 `terminalPipelineVerdict`）不能變。
- **不要發新的 SSE 事件型別**，也不要改 D6 `subtitle_progress` 的 payload（sub-1-3 `[@contract-v1]`）或 `subtitle_run_receipt`。終點走既有 `transcription_complete/failed`，靠 `job_id` 區分。
- **不要在 legacy 模式掛 solo**：`subtitlePipeline == nil` 時 handler 保持舊路（AC #2）。

### 已知陷阱

- **三發 complete：** 語音辨識路線時順序是 `TranscriptionService.transcription_complete`（另一個 job_id、無 cost）→ D6 `subtitle_progress{complete}` → solo `transcription_complete`（帶 job_id、cost）。solo 的那發必須在 `Release` 之後、刪 `jobs` 之前還是之後都可以，但一定是最後一發；測試用 fake hub 記錄順序。
- **goroutine 用 `context.Background()`**：不要用 request ctx（會隨 HTTP 結束被取消）。cancel 要 defer。
- **guard 釋放：** `Start` 的每個錯誤分支都要 `Release`；goroutine 結束也要。寫一個 `defer` 加旗標，或測試每條分支。
- **`predicted_route` 探測共用 ffprobe 併發限制**（`routePredictorAdapter` 註解）：`TrackRouter` 沒有 `PredictRoute`，SoloRunner 另收一個 `predict` 函式（見「其他查到的事」）；不要為此擴大 `TrackRouter` 介面（它是 `[@contract-v1]` 的一部分嗎？不是——但擴大會逼每個 fake 多實作一個方法，Rule 11）。
- **estimate 的 `FromTracks` 走 `services.RoutePredictor` port**（`route_predictor_adapter.go:26-28`），`SetDurationProber` 已經收了同一個 adapter——檢查它是否同時滿足 `RoutePredictor`，避免再注入第二個。
- **電影 `subtitle_tracks` 可能是舊資料**（掃描時的）；探測才是當下真相，但估價只是報價，用舊資料可接受；註解寫明。

### Time-dependent visual coverage

N/A — no wall-clock-reading components touched（純後端）。

### References

- `_bmad-output/implementation-artifacts/eval-see-s01e02-asr-vs-official.md`「討論會」
- `_bmad-output/implementation-artifacts/disc-2026-10-english-track-selection.md`（挑英文軌，已合併 #689；本單讓按鈕用到它）
- `_bmad-output/implementation-artifacts/dsr-6a-cost-on-paid-generate-buttons.md`（估價契約 v1）
- `_bmad-output/implementation-artifacts/sub-3-1-asr-fallback-leg.md`、`sub-4-2-*`（批次走管線的前例）
- `project-context.md` Rule 13、19、20、24、28

## Dev Agent Record

### Agent Model Used

Claude Fable 5.1（claude-fable-5-1）

### Debug Log References

- `apps/api`：`go build ./...`、`go vet ./...`、`go test ./...` 全綠（含 17 個新的 subtitle 測試、8 個新的 handler 測試、9 個新的 estimate 測試）；`~/go/bin/staticcheck ./...` 乾淨。
- 根目錄：`pnpm run lint:all` 0 errors（既有 warning 數不變）；`pnpm nx test web` 299 檔 4616 測試全綠（前端沒動，純回歸）；`test:cleanup` 沒有殘留程序。
- 兩個 red 先確認：`unknown field Regenerate`（T1）、handler 測試在 `SetSoloGenerator` 不存在時編不過（T3）。

### Completion Notes List

- Ultimate context engine analysis completed - comprehensive developer guide created
- **T1：** `ProcessItemOptions.Regenerate`（只跳 `preflightSkip`；`splitCachedCues` 仍只看 `Force`，所以 cache 讀取保留）與 `TranscribeWhenSkipped`（`RouteSkip` 改走 `transcribeFallback`；`transcribeFallback` 多一個 `skipStatus` 參數，ASR 不可用時 RouteSkip 來源仍記 `skipped`、RouteNoTextSource 來源記 `no_text_source`）。`WorkerPool.IsInFlight(ref)`。
- **T2：** 新檔 `subtitle/solo_runner.go`：`NewSoloRunner(pipeline, guard, hub, logger, opts…)`，選項 `WithSoloRoutePredictor`／`WithSoloASRAvailability`／`WithSoloResumeCheck`／`WithSoloTitle`／`WithSoloRunBudgetUSD`／`WithSoloClock`。`Start`：Load → `TryReserve`（失敗回 `services.ErrTranscriptionInProgress`，不釋放別人的 claim）→ 能力閘門（預測非 extract ＋ 無 ASR ＋ 不可續跑 → `services.ErrTranscriptionDisabled`；探測失敗放行）→ 同步 `pipeline.probeWritable`（失敗回 `ErrSubtitleTargetNotWritable`）→ job（uuid、title、`ai.NewBudget`）→ 起點 `transcription_extracting`（`job_id/media_id/media_type/title/phase/message/predicted_route/spent_usd/budget_usd`）→ goroutine 用 `context.Background()` ＋ `ai.WithBudget` 跑 `ProcessItem{Regenerate, TranscribeWhenSkipped, ModelID}` → `Release`、刪 job、終點事件。終點：四種路線各自的 zh-TW `message`、`route`、`zh_srt_path`（有才放）、`cue_count`、`partial`＋`english_kept_blocks`（`TransientCount`）、cost；失敗分 `ErrBudgetExceeded`（含上限金額）、`skipped`、`failed`、其他。`Status`：自己的 job → `(true, id)`；guard 在跑 → `(true, "")`。
- **T3：** `handlers.SoloGenerator` port（用 `subtitle.MediaRef`，handlers 可 import subtitle）＋`SetSoloGenerator`。兩個 Transcribe 在 `lookup*File` 之後分流到 `startSolo`；legacy 閘門（`IsAvailable`／`CanResume…`）在 solo 模式跳過（runner 自己判）。錯誤對應：in-progress → 409、disabled → 503（文案「語音辨識尚未設定」不變，suggestion 多說「這部影片沒有可用的片內字幕」）、`services.ErrTranscriptionTargetNotWritable` 或 `subtitle.ErrSubtitleTargetNotWritable` → 409 `SUBTITLE_TARGET_NOT_WRITABLE`、其他 → 500。`?translate=true` 在 solo 模式忽略（註解寫明）。狀態路由改用 `transcriptionStatusFor(mediaType)` 綁定媒體類型（不用猜路徑），回 `{in_progress, job_id?}`；`TranscriptionStatus` 保留為 movie 版本的薄包裝。**沒有 `SetSoloGenerator` 時全部舊測試不改、照常通過（AC #2）。**
- **T4：** 估價：`TranscriptionPlanExtract`、`Route`（`omitempty`）、`TranscriptionEstimateTarget.SubtitleTracksJSON`（handler 把 `movie.SubtitleTracks.String` 傳進來，服務端用既有 `parsePersistedTracks` 解）、`SetRoutePredictor`（非 nil ＝ pipeline 模式）。路線來源順序：persisted tracks → 片長階梯那次探測順便帶回來（`runtimeMinutes` 多回傳 `probedRoute`，不再把 `_` 丟掉）→ 都沒有才自己 `Probe` 一次。plan：extract 優先於 translate_only。`priceUSD`：extract＋有翻譯金鑰 → `estimateUSD(RouteExtract…)`；extract 無金鑰 → $0；skip 比照 asr。
- **T5：** `main.go`：pipeline 區塊內建 `subtitleSoloRunner`（guard＝`subtitlePipelinePool`、`subtitleRouter.PredictRoute`、`pipelineASR.Available`、`transcriptionService.CanResume*`、title 用 `repos.Episodes`＋`repos.Series` 組「劇名 S01E02」、電影用 `item.Context.Title`、`cfg.AIRunBudgetUSD`）；handler 建好後 `if subtitleSoloRunner != nil { SetSoloGenerator; SetRoutePredictor(routePredictor) }`。`docs/deployment.md` 的 `VIDO_SUBTITLE_PIPELINE_MODE` 列補上兩種模式下按鈕的差別（prettier 重排了表格欄寬，其他列只有空白變動）。**zh-TW 版：`docs/` 沒有 `deployment.md` 的 zh-TW 對應檔（只有安裝指南有），所以沒有第二份可改；AC #12 的雙語在這裡不適用，記在此。**
- **AC #14（NAS 實測）：** 待部署後補；Alexyu 目前不在家裡網路。
- 🔗 AC Drift: FOUND — (1) dsr-6a AC #2「plan ∈ {full, translate_only}、估價永遠不看片內字幕」→ 本單 plan 多 `extract`、多 `route`（只在 pipeline 模式）；legacy 模式不變。(2) 9R-10a AC #2 `POST /episodes/:id/transcribe [@contract-v1]`：回應 `{job_id, message}` 與 202／400／404／409／503 碼都不變，但 @Description 寫的「Runs the Route C pipeline (extract → speech recognition → …)」在 pipeline 模式不再成立（先看片內字幕）；wire 形狀沒變，不升版，@Description 之後由 swagger 重生時更新。(3) sub-2-2a AC #3「不能續跑才 503」在 solo 模式由 runner 的 `canResume` 保留。(4) ⚖️ 2026-08-06 裁定 A 被推翻（`transcription_estimate.go` 註解已改寫）。grep：`single-item button|never reads embedded|ruling A|裁定 A` 於 `_bmad-output/implementation-artifacts/*.md` → dsr-6a、sub-7-6a、sub-7-6、sub-7-7a 等命中皆為引用，非 drift。
- 📎 Contract Stamps: FOUND — dsr-6a AC #2 `[@contract-v1→v2]`（`transcription_estimate.go` 的兩處註解已改 v2；本單 Change Log 有列）。下游 ack grep：`grep -rnE 'confirmed against .?\[@contract-v' _bmad-output/implementation-artifacts/ | grep dsr-6a` → 只有 `apps/web/src/services/transcriptionService.ts:69-73` 的註解（v1）——那是 `-b` 的工作（-b story 已寫明要 ack v2）。沒有 not-done 的 story 需要 stale-mark。另外本單只讀不改 sub-1-4 `[@contract-v2]`（`RouteKind`／`ProcessOutcome`）與 sub-1-5b `[@contract-v1]`（`ProcessItemOptions`：新欄位 additive，FreeOnly 前例，不升版）。
- **對抗式 CR（另開一個全新的 agent，2026-10-06）：2M／4L，全部修掉；另三項資訊性說明中一項改了註解**
  - **M1** pipeline 模式下，單擊的工作在 Activity 頁「進行中」和 `GET /ai/usage` 都看不到（舊路靠 `TranscriptionService.inProgress`，新路的 job 在 SoloRunner 自己的 map）→ SoloRunner 加 `ActivityProgress()`／`ActiveManualUsage()`；新檔 `cmd/api/solo_job_source.go` 的 `composedSoloJobs` 把 `transcriptionService` 與 `subtitleSoloRunner` 合成一個來源給 `NewActivityService` 與 `NewAIUsageHandler`（ASR 腿在 service 裡是 `Solo=false`，不會重複計）。新增 `ActivityAndUsageSeeRunningJobs`。
  - **M2** goroutine 收尾順序是 `Release` → `delete(jobs)`，第二次點擊若在這個縫隙擠進來，它的 job 會被第一次的清理刪掉，整段期間 `Status` 拿不到 job_id → 改成「只刪自己的 job（比指標）→ 發終點事件 → `Release`」，順便讓「終點是該 job 最後一發」不靠別人不插隊。新增 `TerminalIsEmittedBeforeTheClaimIsReleased`、`CleanupOnlyForgetsItsOwnJob`。
  - **L1** 每次點擊同步跑兩次 ffprobe（閘門一次、`predicted_route` 一次）→ 只探一次，結果共用。新增 `PredictsTheRouteOnce`。
  - **L2** 估價在「沒存片長＋探測失敗」時會再探第二次 → `runtimeMinutes` 多回 `ladderProbed`，碰過檔案就不再探。新增 `AFailedLadderProbeIsNotRepeatedForTheRoute`。
  - **L3** 單集的 409 文案退回「這部影片」→ 依媒體類型用「這一集」。新增 handler 測試。
  - **L4** AC #12 的 zh-TW 無對應檔 → 已在 Completion Notes 與 Discovery Triage 寫明。
  - **資訊** `process_item.go:216` 寫入探測對 `RouteSkip` 跳過的註解已補上 `TranscribeWhenSkipped` 的說明；`ProcessItem` panic 無 recover 是既有狀態（pool 與 legacy 一樣），未改；前端終點判定是 -b 的事。
  - 修完：`go test ./...`、`go test -race -count=3 -run SoloRunner`、vet、staticcheck、lint:all 全綠。
- 🎭 A11y Pre-Flight: N/A (100% backend — no apps/web/ files touched)
- 🎨 UX Verification: SKIPPED — no UI changes in this story

### Discovery Triage

| 發現 | 分道 | 追蹤 |
|---|---|---|
| 語音辨識那條路的字幕有很多只有「♪♪」的句子（S01E02 有 22 句）；抽字幕路線的 `FilterSDH` 會丟純音符句（sub-6-4），語音辨識路線沒有這一步（`grep ♪ internal/` 只有 `sdh_filter.go`）。Alexyu 2026-10-06 討論中提出 | ③ | `disc-2026-10-asr-music-only-cues`（sprint-status，開單時立案） |
| `docs/deployment.md` 沒有 zh-TW 版（只有安裝指南有雙語）；bilingual rule 在部署文件上目前沒被執行 | ③ | 沿用既有狀態，不另立單（Alexyu 若要雙語部署文件再開） |

### File List

- `apps/api/internal/subtitle/solo_runner.go`（新）
- `apps/api/internal/subtitle/solo_runner_test.go`（新）
- `apps/api/internal/subtitle/process_item_solo_options_test.go`（新）
- `apps/api/internal/subtitle/pipeline.go`（改：`ProcessItemOptions.Regenerate`／`TranscribeWhenSkipped`、`preflightSkip`）
- `apps/api/internal/subtitle/process_item.go`（改：RouteSkip 分流、`transcribeFallback(skipStatus)`）
- `apps/api/internal/subtitle/worker_pool.go`（改：`IsInFlight`）
- `apps/api/internal/handlers/transcription_handler.go`（改：`SoloGenerator`、`SetSoloGenerator`、`startSolo`、`transcriptionStatusFor`、`TranscriptionStatusResponse.JobID`、estimate 傳 tracks JSON）
- `apps/api/internal/handlers/transcription_solo_handler_test.go`（新）
- `apps/api/internal/services/transcription_estimate.go`（改：plan extract、Route、SubtitleTracksJSON、SetRoutePredictor、路線來源、priceUSD）
- `apps/api/internal/services/transcription_estimate_route_test.go`（新）
- `apps/api/cmd/api/main.go`（改：SoloRunner 接線、SetSoloGenerator、SetRoutePredictor、Activity／ai-usage 合成來源、`fmt` import）
- `apps/api/cmd/api/solo_job_source.go`（新：`soloJobSource`／`composedSoloJobs`）
- `docs/deployment.md`（改）
- `_bmad-output/implementation-artifacts/disc-2026-10-single-generate-ignores-embedded-english-a.md`（新）
- `_bmad-output/implementation-artifacts/disc-2026-10-single-generate-ignores-embedded-english-b.md`（新，依賴本單）
- `_bmad-output/implementation-artifacts/sprint-status.yaml`（改）
- `_bmad-output/implementation-artifacts/dsr-6a-cost-on-paid-generate-buttons.md`（AC drift reference — see Completion Notes；本單沒改這個檔）

## Change Log

| 日期 | 內容 |
|---|---|
| 2026-10-06 | Bob create-story（依 Alexyu 裁定選項 1，電影一起修；拆成 -a 後端／-b 前端）。 |
| 2026-10-06 | T1：`Regenerate`／`TranscribeWhenSkipped`／`WorkerPool.IsInFlight`。 |
| 2026-10-06 | T2：`subtitle.SoloRunner`（單飛、閘門、寫入探測、預算、起點／終點事件）。 |
| 2026-10-06 | T3：handler `SoloGenerator` port；status 綁媒體類型並回 `job_id`。 |
| 2026-10-06 | [@contract-v1→v2] AC #2（dsr-6a）：`plan` 多 `extract`、新增 `route`（pipeline 模式才有）、`GET …/transcribe/status` 多 `job_id`。下游受影響：前端 `transcriptionService.ts` 的 `plan` union 與 `TranscriptionStatus` 型別（-b 處理並 ack v2）；legacy 模式 wire 完全不變。 |
| 2026-10-06 | T4：估價分路線；T5：main.go 接線＋docs；T6／T7：測試與檢查全綠，狀態改成 review。 |
| 2026-10-06 | CR 修正：M1 Activity／ai-usage 看得到單擊的工作、M2 收尾順序（刪自己的 job → 終點事件 → Release）、L1 只探一次、L2 估價不重探、L3 單集 409 文案。全部檢查重跑全綠。 |
| 2026-10-06 | AC #14 NAS 實測 2026-10-06 晚通過：估價 `plan=extract, route=extract, $0.46`；實跑 `route=translate`、第 8 軌、568 句、$0.34、6 分 12 秒；對官方英文軌 0 句落在沒人說話時；Jerlamarel 1 種寫法；sun→太陽；開頭重複台詞都在。順帶發現 15 句因「里」被當簡體字留成英文（另立 `disc-2026-10-simplified-leak-false-positive-li`）與名詞表被 ASR 錯字污染（`disc-2026-10-asr-harvest-pollutes-glossary`）。 狀態改成 done。 |
