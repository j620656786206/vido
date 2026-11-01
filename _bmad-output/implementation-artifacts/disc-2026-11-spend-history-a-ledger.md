# Story disc-2026-11-spend-history-a-ledger：每次執行都記下「當初預估多少」「用哪個聽寫引擎」「字幕從哪來」

Status: done

**Source:** 傘狀 `disc-2026-11-spend-history`（Alexyu 2026-11-01 提出，party mode 同日討論，⚖️ 選 B：四張一起排）。本單只做後端記錄，讓 -b 的列表 API 和 -d 的頁面有資料可讀。不做：API、畫面、回填舊資料、改月報。

## 查到的事（2026-11-01，Bob 逐一開檔確認）

### 現在每筆執行記了什麼

- `subtitle_runs`（`internal/models/subtitle_run.go:103`）有：開始／結束時間、`media_id`／`media_type`、`model_id`（翻譯模型）、`status`、`spent_usd`、`budget_usd`、`route`、`batch_id`、`triggered_by`。
- 正式機（2026-11-01 唯讀查）：9/2 起 106 筆、合計 US$32.48；completed 98（US$31.24）、failed 8（US$1.24）。`route` 是空的 24 筆（migration 041 之前）。

### 缺的三樣

1. **當初預估**：從沒存過。`budget_usd` 是花費上限（預設 US$5，`config.go` `AI_RUN_BUDGET_USD`），不是預估。單集預估來自 `TranscriptionEstimateService.Estimate`（`internal/services/transcription_estimate.go:243`，詳細頁對話框打 `GET /…/transcribe/estimate`，`transcription_handler.go:619`、`:654`）；批次的每項預估來自 `generation_candidates.go` 的 `EstimatedUSD`（`:331`）。批次項目 `GenerationBatchItem`（`generation_batch.go:152`）不帶預估。
2. **聽寫引擎**：沒存。`ASRProviderHolder.Provider(ctx)`（`asr_provider_holder.go:205`）回 `elevenlabs`／`openai`／`self_hosted`；Scribe 模型常數 `ai.ScribeModel = "scribe_v2"`（`ai/scribe.go:61`），Whisper 用 `ASR_MODEL` env（預設空＝供應商預設）。
3. **片旁英文字幕 vs 片內字幕**：都記成 `route=translate`。router 的片旁字幕決策用 `StreamIndex: -1` 標記（`subtitle/router.go` `sidecarVerdict`）。

### 執行紀錄在哪裡建立

- 管線模式：`subtitle/process_item.go:118-133` 建列，`:191` 寫 `run.Route = string(decision.Kind)`；聽寫後備 `:616`、`:653` 改成 `asr`。
- legacy／Route C 聽寫：`services/transcription_ledger.go:72-97` `openLedgerRun`。

## 設計（Bob × Winston）

**新欄位（migration 048，全部 nullable；NULL＝沒記錄，不是 0）：**

| 欄位 | 型別 | 內容 |
|---|---|---|
| `estimate_usd` | REAL | 這次執行開始時，用**跟對話框同一支** `Estimate` 算出的預估（含續跑折抵） |
| `asr_provider` | TEXT | 真的有跑聽寫時才寫：`elevenlabs`／`openai`／`self_hosted` |
| `asr_model` | TEXT | 同上，例如 `scribe_v2`；拿不到就 NULL |
| `source` | TEXT | `embedded_text`（片內文字字幕）／`sidecar`（片旁英文字幕）／`audio`（聽寫）；`skip`、`no_text_source` 等沒處理字幕的路線留 NULL |

- **預估在後端開始時重算，不是讓前端傳。** 理由：一套程式涵蓋單集、批次、legacy；不信任 client；`Estimate` 的探測結果有快取（sub-5-4），重算很便宜。代價：批次排很久才輪到時，若中間改了模型，記的是「開始那一刻」的預估。這要寫進欄位註解。
- 算預估失敗**不能讓執行失敗**：Warn 一次，`estimate_usd` 留 NULL。
- `route` 不改：`sidecar` 走 `source` 欄位，月報（sub-7-6b 依 `route` 分 translated／asr）不受影響。
- `asr_provider`／`asr_model` 在**真的呼叫聽寫**的那條路寫（`process_item.go` 的 ASR 後備、`transcription_ledger.go`），不是看設定猜。

## Story

身為想知道每次生成花多少錢的人，我希望 Vido 從今天起每次執行都記下當初預估、聽寫引擎和字幕來源，之後的歷程頁才有東西可以比。

## Acceptance Criteria

1. Migration 048 加入四個 nullable 欄位；`SubtitleRun` model、repository `Create`／`Update`／讀取都帶上（Rule 28：用真的 SQLite 測讀寫來回）。
2. 管線模式每次執行建列後，`estimate_usd` 等於同一支 `Estimate` 對同一部片的 `EstimatedUSD`；估價器沒接上或回錯時，欄位是 NULL，執行照常跑完。
3. `source`：片內文字字幕 → `embedded_text`；片旁英文字幕（`StreamIndex == -1`）→ `sidecar`；聽寫後備與 Route C 聽寫 → `audio`；skip／no_text_source → NULL。
4. 有跑聽寫的執行寫入 `asr_provider`（和拿得到的 `asr_model`）；沒跑聽寫的執行兩欄都是 NULL。
5. 舊資料不回填，讀出來是 NULL；`GET /subtitles/spend` 的回應不變（既有測試照過）。
6. `subtitle_run_receipt` 事件 payload 若由 `ReceiptPayload()` 產生，新欄位**有值才出現**（沿用「沒記錄的鍵不出現」規則）。

## Tasks / Subtasks

- [x] Migration 048 ＋ registry ＋ migration 測試（AC 1）
- [x] model／repository 欄位（AC 1、5）
- [x] 管線：建列時算預估（注入估價介面，subtitle → services 方向合法）、依 decision 寫 `source`（AC 2、3）
- [x] 聽寫兩條路寫 `asr_provider`／`asr_model`／`source=audio`（AC 3、4）
- [x] Receipt payload（AC 6）
- [x] 測試：管線、ledger、repository 來回、月報不變

## Dev Notes

- `estimate_usd` 只記「開始那一刻」的數字；之後的 -b 才決定怎麼呈現「預估 vs 實際」。
- 續跑（resume）的執行：`Estimate` 已扣掉續跑折抵（`generation_candidates.go:388` 註解），照記即可。
- 本機 darwin 沒有 ffprobe 也沒關係：估價器在測試裡用 fake。

## Dev Agent Record

### 做了什麼

- **Migration 048**（`add_subtitle_run_history_columns`）：`subtitle_runs` 加 `estimate_usd REAL`、`asr_provider`／`asr_model`／`source TEXT`，全部 nullable、冪等（`columnExists` 守門，041 寫法）；舊資料不回填。
- **Model／repository**：`SubtitleRun` 加 `EstimateUSD *float64`、`ASRProvider`／`ASRModel`／`Source string`（空字串存 NULL，沿用 `nullIfEmpty`）；`subtitleRunColumns` 24 → 28 欄，INSERT／UPDATE／scan 同步（Rule 15）。新常數 `SubtitleRunSource{EmbeddedText,Sidecar,Audio}`。`ReceiptPayload()` 新鍵有值才出現。
- **預估怎麼接**：新增 `services.TranscriptionRunEstimator`（`run_estimator.go`）——讀電影／單集那一列，照 `EstimateMovie`／`EstimateEpisode` handler 一模一樣組 `TranscriptionEstimateTarget`，呼叫同一支 `TranscriptionEstimateService.Estimate`，回 `EstimatedUSD`。series、找不到、沒檔案路徑都回 error → NULL。
  - **管線（單集 click、批次、pool）**：`subtitle.RunEstimator` 窄介面＋`WithRunEstimator` option；`ProcessItem` 建列時（pre-flight 之後）呼叫，失敗或沒接 → NULL＋Warn 一次，執行照跑。
  - **legacy／Route C（legacy 單集 click、legacy 批次）**：`TranscriptionService.SetRunEstimator`；`openLedgerRun` 開列時算（在續跑判斷與抽音軌之前）。管線的 ASR 後備是「呼叫方自己記列」，不開第二列也不重算。
  - `main.go`：把 `routePredictor`＋`transcriptionEstimateService` 的建構往前搬到管線建構之前（route predictor／sidecar source 改用 `cfg.SubtitlePipelineEnabled()` 判斷，跟原本 `subtitleSoloRunner != nil` 等價），一個 adapter 同時給管線和 TranscriptionService。
- **source**：管線路由後依 decision 寫：`Track.StreamIndex == -1` → `sidecar`；其他有 track → `embedded_text`；skip／no_text_source（沒 track）→ NULL。ASR 後備只要服務有接手（不是 `ErrTranscriptionDisabled`）就改 `audio`（成功、預算暫停、中途失敗都算）。Route C 的列一律 `audio`。
- **asr_provider／asr_model**：`ASRProviderHolder.Identity(ctx)`——Scribe → `scribe_v2`（`ai.ScribeModel`）；Whisper／自架 → `ASR_MODEL`，沒設就空（NULL）。`TranscriptionService` 只在**真的進入聽寫那一步**才蓋章（translate-only 續跑不蓋）；蓋在自己的列，或管線經 `services.WithASRIdentityTarget(ctx, run)` 傳下來的那一列。
- `route` 沒動；`GET /subtitles/spend` 相關測試照過。

### 跟故事字面不同的地方（請 review）

- **FreeOnly（掃描後的自動免費線）不算預估**：這條線本來就把付費路線全部延後，預估只會標一個不會發生的價；而且 `Estimate` 會 ffprobe、還會在背景啟動片旁字幕對時檢查，每次掃描每一項都跑一次對 NAS 是白花的資源。這些列 `estimate_usd` 為 NULL。
- ~~Whisper 沒設 `ASR_MODEL` 時 `asr_model` 是 NULL~~ → **review 後改成記 `whisper-1`**：Whisper client 沒設模型時，請求裡送的就是 `ai.WhisperModel`（`ai/whisper.go:186`），所以這是「實際送出的模型」，不是猜的。
- **review 補：預估最多算 20 秒**（`runEstimateTimeout`）。對話框或批次清單剛估過時探測有快取，不會碰到；冷的大檔 ffprobe 不能拖住執行開始。

### 檔案

- 新增：`apps/api/internal/database/migrations/048_add_subtitle_run_history_columns.go`、`048_add_subtitle_run_history_columns_test.go`、`apps/api/internal/services/run_estimator.go`、`run_estimator_test.go`
- 修改：`apps/api/cmd/api/main.go`、`internal/models/subtitle_run.go`（＋`_test`）、`internal/repository/subtitle_run_repository.go`（＋`_test`、`subtitle_run_recent_test.go`）、`internal/services/asr_provider_holder.go`（＋`_test`）、`internal/services/transcription_ledger.go`（＋`_test`）、`internal/services/transcription_service.go`、`internal/subtitle/pipeline.go`、`internal/subtitle/process_item.go`、`internal/subtitle/process_item_asr_test.go`、`internal/subtitle/process_item_ledger_test.go`

### 測試結果（apps/api，2026-11-01）

- 新測試：migration 048（NULL 讀回、冪等、註冊）；repository 真 SQLite 來回＋未設時真的是 NULL（Rule 28）；receipt 新鍵有值才出現；holder `Identity`；run estimator target 與 handler 一致、各種無數字情況；Route C 列記預估／audio／引擎（假 ffmpeg 跑完整聽寫）、預估失敗與沒接 → NULL 且執行完成、caller 的列被蓋章；管線預估、失敗／沒接、FreeOnly 不算、source 五種 decision、ASR 後備成功／暫停／失敗／被拒。
- `go test ./internal/... ./cmd/...`：44 個套件全過。
- `go vet ./...`：0；`~/go/bin/staticcheck ./...`：0。
- `gofmt -l ./internal ./cmd`：本次改動的檔案全乾淨；清單裡仍有約 60 個既有檔案（本機 go1.26.3 的 gofmt 規則比較新，非本次改動）。

## Change Log

- 2026-11-01：建立（Bob）。
- 2026-11-01：實作完成，Status → review（Amelia）。
- 2026-11-01：review 補兩處：Whisper 預設模型記 `whisper-1`、預估加 20 秒上限（含測試）。
