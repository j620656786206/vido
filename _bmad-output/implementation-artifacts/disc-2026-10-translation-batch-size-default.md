# Disc：翻譯每批送幾句改成「看模型」——Sonnet 50 句、Gemini Flash-Lite 20 句、其他照舊 10 句

Status: done

**Source:** `disc-2026-10-translation-batch-size-exam`（PR #802）的量測。Alexyu 2026-10-15 選 **A**：Gemini 改 20 句；Claude Sonnet 再考一次 50 句，還是一樣好就也改。第二次 Sonnet 50 句：0.5%／96.0%（第一次 0.5%／96.5%；10 句是 1.5%／90.0%）→ 改。

## 查到的事（2026-10-15，main `9a20495c`）

- 唯一的常數 `apps/api/internal/ai/prompts/subtitle_translator.go:39-41` `SubtitleTranslatorBatchSize = 10`。
- 讀它來切批的兩條路：片內字幕那條 `internal/subtitle/pipeline.go:909-923`（`TranslateTrack`；這裡已經有 `p.currentModelID(ctx)`，`:456-471`，ctx 的選擇 > 部署預設 > `ai.DefaultClaudeModel`）；聽寫那條 `internal/services/translation_service.go:426`（`WithBatchSize` 覆寫；`EffectiveModelID(ctx)` `:271` 永遠有值）。
- 讀它來估算的：試跑 20 句的估價 `internal/preview/service.go:153`；Gemini 控速的等待分鐘 `internal/services/generation_candidates.go:583-592`。
- 不改的：`translation_service.go:332` `demoteAbsoluteKeptBlocks`（「留英文達一整批的量就降級」的門檻，維持 10 句更保守）；估價的每分鐘費率 `estimateUSD`（`generation_candidates.go:1161-1186`，常數，Sonnet 實際會變便宜 → 報價偏高不偏低，另立單校正）。
- `ai.DefaultClaudeModel = "claude-sonnet-5"`（`internal/ai/claude.go:40`）：很多切批測試沒指定模型，原本吃 10，改完會變 50。

## Acceptance Criteria

1. `prompts.SubtitleTranslatorBatchSizeFor(model)`：`claude-sonnet-5` → 50、`gemini-3.5-flash-lite` → 20、其他（含空字串、Haiku、沒量過的）→ 10。註解寫量測數字與出處。
2. 兩條翻譯路、試跑估價、Gemini 等待估算都改讀它；`WithBatchSize` 明確指定時照舊優先。
3. 考卷工具：沒給 `--batch-size` 時用該模型的 App 批次，報告一律標 `batch_size`；`--merge` 用「該模型的 App 批次」判斷能不能發布（舊報告沒標＝10）。
4. 測試：`SubtitleTranslatorBatchSizeFor` 表格；`TranslateTrack` 60 句在 Sonnet／Gemini／Haiku 下分成 50+10／20×3／10×6；翻譯服務沒帶模型（＝Sonnet）25 句一次送完；merge 依模型判斷（Sonnet 50 收、舊的 10 拒）。原本測切批機制的 22 條測試改成明確指定一個 10 句的模型（`batch10Ctx`，Haiku），數字不變。
5. `go test ./...`、vet、staticcheck 綠。
6. 正式機驗證：末日光明 S01E04（有英文字幕軌、有舊的官方繁中字幕）用預設 Sonnet 跑一次：看請求數（約 9 次，原本約 45）、費用、耗時，人名跟舊官方字幕一致。

## Tasks / Subtasks

- [x] T1 `SubtitleTranslatorBatchSizeFor`（AC 1）
- [x] T2 四個讀取點（AC 2）
- [x] T3 考卷工具與 merge（AC 3）
- [x] T4 測試（AC 4、5）
- [x] T5 正式機驗證（AC 6）：❌ 2026-10-15 11:27 末日光明 S01E04（image 768ca070、預設 Sonnet、50 句一批）：474 句分 10 批，第 4 批（151–200 句）回應頂到 4096 tokens 上限被截斷、無法解析，**整個 run 失敗**，白花 US$0.126。前 3 批正常（輸出 748／2,064／845 tokens）。→ 當天 hotfix 把 Sonnet 退回 10 句（`fix/sonnet-batch-back-to-10`），Gemini 20 句保留；根因與「截斷就切半重翻」另立 `disc-2026-10-translation-batch-truncation`。

## Dev Notes

- 只改量過的模型。Haiku、其他 Claude、其他 Gemini 都維持 10，量過再加進表裡。
- `model_ratings.json` 裡 Gemini 的那列是 10 句一批量的；20 句四次都持平，不急著重新發布。
- 回應上限 `TranslationMaxTokens = 4096`（`translation_service.go:36`，兩條路共用）：考卷上 Sonnet 50 句兩次都沒有被截斷，**但正式機第一集就有一批頂到上限**（見 T5）——考卷 200 句來自多部片、句子短，估「50 句約 2,000 tokens」是錯的。
- 更大的批次讓「一批失敗」時留下的英文更多（Sonnet 一批 50 句）；既有的機制（品質沒過只重送那幾句、傳輸失敗重試、連兩批失敗就停）不變。

### Time-dependent visual coverage

N/A — 純後端。

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（claude-opus-5-5），2026-10-15。

### Completion Notes List

- 見 Tasks。新增 `TestSubtitleTranslatorBatchSizeFor`、`TestTranslateTrack_BatchSizeFollowsTheModel`、`TestMergeReport_AppBatchIsPerModel`；`TestTranslateWithGlossary_BatchSizeOption` 加「沒帶模型＝Sonnet 50」一段；`TestGenerationCandidates_PacedModelQuotesMoreMinutes` 改成 20 句一批的約 9 分鐘。
- 22 條切批機制測試改用 `batch10Ctx()`（`internal/subtitle/pipeline_test.go`、`internal/services/translation_service_test.go` 各定義一次）。

### Discovery Triage

- ③ 估價的每分鐘費率（Sonnet 0.48 美元／小時等）是用 10 句一批量的，改成 50 句後實際會變便宜 → `disc-2026-10-translation-rates-recalibrate-after-batch`（正式機跑幾集後用實際帳重算）。

### File List

- `apps/api/internal/ai/prompts/{subtitle_translator.go,subtitle_translator_test.go}`
- `apps/api/internal/subtitle/{pipeline.go,pipeline_test.go,pipeline_transient_test.go}`
- `apps/api/internal/services/{translation_service.go,translation_service_test.go,translation_cache_test.go,transcription_translation_resume_test.go,transcription_generation_test.go,translation_dispatcher_test.go,generation_candidates.go}`
- `apps/api/internal/preview/service.go`
- `apps/api/cmd/grade/{main.go,merge.go,merge_test.go}`
