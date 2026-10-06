# Bugfix: 「聽聲音生成字幕」的報價改用這條路線自己量到的翻譯費率，不再少報一半

Status: review

**Source:** Alexyu 2026-10-05。See S01E02 單集「生成字幕」按鈕報價 US$0.80，實際花 US$1.36（多 70%）。產品原則：「會諂媚的讀數比沒有讀數更糟」。同時吸收 `disc-2026-09-asr-route-translation-rate-unmeasured`（P3，2026-09-17 dsr-6a CR 懷疑過、當時未查證；這次有實測，確認是真的）。

## Story

身為 Vido 的使用者，
我希望按下「生成字幕」前看到的價錢，接近真的會扣的錢，
這樣我才能放心按，不會事後才發現多花了一半。

## 查到的事（2026-10-06，Bob 逐一開檔／查 NAS 確認）

- **報價公式：** `estimateUSD`（`apps/api/internal/services/generation_candidates.go:1009-1021`）在 `RouteASR` 時算「語音辨識每分鐘費率 ＋ `translationRatePerMinute(model)`」；`translationRatePerMinute`（`:189-210`）查 `translationUSDPerMinuteByModel`（`:166-169`，Sonnet `0.00804`、Haiku `0.00301`）。
- **那張費率表量的是「抽字幕再翻」：** 註解寫明是 eval-1 翻了 12h20m、10,304 句（`:146-151`）。NAS 帳本 `subtitle_runs` 裡 2026-09-02／03 那 21 筆（`route` 欄是空的、有 `cue_count`）就是 eval-1；Sonnet 每句約 $0.00053–0.00058（例：2,217 句 $1.18、567 句 $0.33）。
- **「聽聲音」的翻譯走另一條程式、沒有 prompt caching：** 單集／單部電影的 `runPipeline` → `translateSRT`（`apps/api/internal/services/transcription_service.go:1217`）→ `TranslateWithGlossaryHarvest`（`:1968`）。`:1946-1952` 的 CR M2 註解寫明這條沒有快取、片子背景資訊每批重送重計費。抽字幕那條走 `subtitle/pipeline.go:942` 的 `TranslateChunk`（可用快取）。
- **批次的 ASR 項目也走同一條：** `apps/api/cmd/api/asr_adapter.go:36` 與 `apps/api/internal/services/generation_batch_runner.go:38` 都呼叫 `RunTranscription` → 同一個 `runPipeline`。所以**批次候選清單的 `RouteASR` 報價也一樣少報**（`generation_candidates.go:811`、`:842` 都呼叫 `estimateUSD`）。
- **續跑（只剩翻譯）也少報：** `TranscriptionEstimateService.priceUSD`（`apps/api/internal/services/transcription_estimate.go:157-172`）在 `translate_only` 時用 `estimateUSD(RouteExtract, …)`，也就是抽字幕路線的費率；但續跑實際一樣走 `translateSRT`（`runPipeline` 開頭的 `tryTranslateOnlyResume`，`transcription_service.go` 同函式）。
- **實測（NAS 帳本，`sqlite3 -readonly /mnt/cache/appdata/vido/vido.db`）：** `route=asr`、`claude-sonnet-5`、659 句、`spent_usd=1.36365`、語音 3,417 秒。log：輸入 227,210／輸出 22,682 tokens、66 次呼叫。算回去：語音 3,417/60 × $0.006 = $0.342（報價這段準）；翻譯 227,210×$3/M ＋ 22,682×$15/M = **$1.022**，每句 **$0.00155**——是抽字幕路線的 **2.7 倍**。66 次呼叫 ≈ 每批 10 句（`translation_service.go:411` `SubtitleTranslatorBatchSize`），每次約 3,400 輸入 tokens，大多是每批固定重送的部分，所以費用跟「句數」成正比。
- **價格表：** `apps/api/internal/ai/budget.go:48-49`：Haiku 4.5 $1／$5、Sonnet 5 $3／$15（每百萬 tokens）。

## 設計

**分路線各用各的費率**，抽字幕那張表不動：

- 新增「聽聲音那條的翻譯費率」表，Sonnet 5 一列，數字＝**每句實測成本 × 媒體庫平均每分鐘句數**：
  $0.001551／句（S01E02）× 13.92 句／分（eval-1：10,304 句 ÷ 740 分）＝ **$0.0216／分**。
  - 為什麼不直接用 S01E02 的每分鐘數字（$0.0179）：See 是台詞很少的劇（官方字幕每分鐘約 10 句），拿它當全媒體庫的費率，台詞多的片會再被少報。每句成本是結構性的（每批固定重送），穩；句數密度用 9 部片的平均，比 1 集可靠。
  - 代價：See S01E02 這種台詞少的片，新報價 $1.57（實際 $1.36，**多報約 15%**）。照 `translationCalibrationModel` 既有原則（`generation_candidates.go:173-181`：「寧可往上給，不要讓使用者被往上嚇到」），往上偏是可以接受的方向。
- 其他模型照既有規則從 Sonnet 錨點按價格比例換算（Haiku ＝ 0.0216 × 6/18 ＝ 0.0072）；沒價格的模型用錨點。
- `estimateUSD(RouteASR, …)` 改用新表。批次清單與單集按鈕共用同一個函式，一處修好兩邊。
- 單集續跑（只剩翻譯）改用新表，不再借 `RouteExtract`。
- `RouteExtract`、花費頁「省下多少」（`subtitle_spend_service.go:206` 用 `RouteExtract`）**不動**。

## Acceptance Criteria

1. **ASR 路線費率：** `generation_candidates.go` 有一張 ASR 路線專用的翻譯費率表（Sonnet 5 = `0.0216`），註解寫出上面的推導與數字來源；其他模型用與 `translationRatePerMinute` 相同的錨點＋價格比例規則（共用一個 helper，不要複製一份邏輯）。
2. **`estimateUSD(RouteASR, …)`** 的翻譯那半改用 ASR 路線費率；`RouteExtract` 結果完全不變。
3. **單集續跑：** `priceUSD` 的 `translate_only` 用 ASR 路線費率；`full`＋不翻譯（只付語音）、什麼都不付兩種情況不變。
4. **回歸保護（用真實數字）：** 測試用 S01E02 的實測：56.95 分、Sonnet 5、雲端語音 → 報價 **≥ $1.36**（不再少報），且 **≤ $1.36 × 1.25**（不亂灌水）。
5. **既有不變式仍成立：** ASR 比抽字幕貴；換模型只動翻譯那半；單集按鈕與批次清單對同一片同一路線報同一個價；自架語音仍只付翻譯。
6. **文件：** `docs/deployment.md` 「Which model translates, and what it costs」那段補一句：$0.48／$0.18 每小時是「片內有文字字幕」的情況；要聽聲音時翻譯約 Sonnet $1.30／小時、Haiku $0.43／小時，另加語音辨識。
7. **sprint-status：** 本條目、以及被吸收的 `disc-2026-09-asr-route-translation-rate-unmeasured` 都更新狀態並互相連結。
8. 檢查全綠：`go build ./... && go vet ./... && go test ./...`（apps/api）、`~/go/bin/staticcheck ./...`、`pnpm run lint:all`。

## Tasks / Subtasks

- [x] T1 費率表＋共用 helper＋`estimateUSD` 改用（AC #1、#2）
- [x] T2 `priceUSD` 續跑改用（AC #3）
- [x] T3 測試：更新被釘死的舊數字（`transcription_estimate_test.go` 的 0.84／0.54／0.48／0.39 等、`generation_candidates_test.go` 的模型差距），新增 AC #4 回歸測試（AC #4、#5）
- [x] T4 文件＋sprint-status（AC #6、#7）
- [x] T5 檢查（AC #8）

後端 3 項、前端 0 項 → 不拆單。

## Dev Notes

### 不要做的事

- **不要改 `translationUSDPerMinuteByModel`**（抽字幕那張）：它的實測是對的，候選清單的 extract 項目、花費頁「省下多少」都靠它。
- **不要在這張單接上 ASR 那條的 prompt caching**：那是真的省錢，但會改變實際行為，屬 `backlog-asr-leg-unify-gated-pipeline`（sprint-status 第 1022 行已記「併軌接上 caching 後此成本自然消失」）。接上之後要回來重量這張表——在註解寫明。
- 不要改前端：前端只顯示 `estimated_usd`，數字由後端決定。
- 不要把 S01E02 的字幕內容放進 repo（公開 repo，`.claude/memory/feedback_public_repo_no_transcripts.md`）——只放數字。

### 已知陷阱

- 金額一律 `decimal`，費率用字串常數（`tech-money-decimal-arithmetic`，`generation_candidates.go:160-165` 的註解）。
- `estimateUSD` 對兩段相加後才 `roundUSD`；續跑那條只有一段，同樣最後才捨入。
- `TestAnalyze_OnlyTheTranslationHalfMovesWithTheModel` 用 `translationRatePerMinute` 算模型差距，ASR 路線改費率後要改成新 helper，不然會紅。

### Time-dependent visual coverage

N/A — 純後端估價，沒有畫面改動。

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（claude-opus-5-5）

### Debug Log References

- NAS 帳本（read-only）：`route=asr`、Sonnet 5、659 句、`spent_usd=1.36365`、`asr_seconds` 3,417。
- 變異測試：把 `estimateUSD` 的 RouteASR 改回 `translationRatePerMinute` → `TestEstimateUSD_ASRRouteCoversTheMeasuredSeeS01E02Bill`、`TestTranscriptionEstimate_FullRunPaysSpeechRecognitionAndTranslation` 都紅，還原後綠。

### Completion Notes List

- 新增 `asrLegTranslationUSDPerMinuteByModel`（Sonnet 5 `0.0216`）與 `asrLegTranslationRatePerMinute`；原本的錨點＋價格比例邏輯抽成 `ratePerMinuteFrom`，兩張表共用（AC #1）。
- `estimateUSD(RouteASR)` 改用新費率；`RouteExtract` 不變（AC #2）。
- `priceUSD` 的 `translate_only` 改用新費率（AC #3）。
- 新報價（60 分鐘、Sonnet、雲端語音）：$0.84 → $1.66；Haiku $0.54 → $0.79；續跑 60 分 $0.48 → $1.30。S01E02：$0.80 → $1.57（實際 $1.36）。
- 新測試：`TestASRLegTranslationRate_IsMeasuredSeparatelyFromTheExtractRoute`、`TestEstimateUSD_ASRRouteCoversTheMeasuredSeeS01E02Bill`；續跑測試改成「＝ASR 路線的翻譯那半、且 > 抽字幕費率」（AC #4、#5）。
- 檢查：`go build／vet` 綠、`go test ./...` 42 個套件全綠、`staticcheck-2026.1 ./...` 0 項、`pnpm run lint:all` 0 errors（AC #8）。

### File List

- `apps/api/internal/services/generation_candidates.go`
- `apps/api/internal/services/generation_candidates_test.go`
- `apps/api/internal/services/transcription_estimate.go`
- `apps/api/internal/services/transcription_estimate_test.go`
- `docs/deployment.md`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`
- `_bmad-output/implementation-artifacts/bugfix-generation-estimate-under-quotes.md`

## Change Log

- 2026-10-06 Bob create-story：ready-for-dev。Ultimate context engine analysis completed - comprehensive developer guide created.
- 2026-10-06 dev-story：review。
