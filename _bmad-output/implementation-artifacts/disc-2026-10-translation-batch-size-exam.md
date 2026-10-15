# Disc：翻譯每批送幾句——考卷量 10／20／50／100

Status: review

**Source:** 2026-10-15 FB Ming Jwo：「你的問題是行數太少，第一梯隊模型可以到 150 行，gemini 可以到 100 行」。程式裡 `SubtitleTranslatorBatchSize = 10` 的註解只寫「在費用和品質之間取平衡」，從沒量過。Alexyu 選 A：先考再回、再補考一輪再改。

## 查到的事

- `apps/api/internal/ai/prompts/subtitle_translator.go:39-41` `SubtitleTranslatorBatchSize = 10`；吃它的地方：`subtitle/pipeline.go:909-923`（切批）、`services/translation_service.go:426`、`preview/service.go:153`、`services/generation_candidates.go:583-592`（Gemini 控速的等待分鐘數）。
- 考卷工具 `apps/api/cmd/grade/main.go:46` `translateChunk = 20`：每次呼叫翻譯服務最多 20 句，`--batch-size` 比 20 大也只會送 20。第一輪的「50」「100」四次其實都是 20 句一批（兩種耗時一模一樣 0.78 分是線索）。`--batch-size` 只准本機端點用，Claude 不能考。
- `cmd/grade/merge.go` 沒擋非預設批次的報告：一份 batch-50 的 Gemini 報告可以被 `--merge` 發布到評分表。

## Acceptance Criteria

1. 考卷工具：`--batch-size` 大於 20 時每次就送那麼多句；Claude 也能用 `--batch-size`（報告標上批次）。
2. `--merge` 拒絕任何不是用 app 預設批次量出來的報告（實驗不能變成正式評分）。
3. 測試：100 句、`--batch-size 50` → 2 次請求、報告標 50；Claude 的 `--batch-size 50` 能 parse；batch 50 的報告 merge 被拒、batch 10 照收。
4. 量測：golden-v2 200 句、附名詞表、production prompt、評審 claude-sonnet-5；Gemini 3.5 Flash-Lite 10／20／50／100、Claude Sonnet 5 10／50。原始報告在 `eval/golden-v2-2026-10/`。
5. 第一輪被截成 20 句的四份報告改名 `*-batch20-run{1..4}.json`、`batch_size` 改 20、加 `batch_note` 說明。

## 結果（2026-10-15）

| 模型 | 每批 | 次數 | 翻壞率（0 分） | 自然率（2 分） | 等級 | 人名對不上 | 翻 200 句 |
|---|---|---|---|---|---|---|---|
| Gemini 3.5 Flash-Lite | 10 | 1 | 2.0% | 89.5% | A | 1 | 4.1 分（每 13 秒一批） |
| | 20 | 4 | 2.0／2.0／2.5／3.0% | 89.0～92.0% | A ×4 | 1 ×4 | 0.78 分 |
| | 50 | 2 | 5.0／2.0% | 88.0／91.5% | A ×2 | **4**／1 | 0.3 分 |
| | 100 | 2 | 1.0／**10.0%** | 90.5／**81.5%** | A／**B** | 1／**19** | 0.25 分 |
| Claude Sonnet 5 | 10 | 1 | 1.5% | 90.0% | A | 1 | 1.6 分 |
| | 50 | 1 | **0.5%** | **96.5%** | A | 1 | 1.0 分 |

- **Gemini 100 句一批會整批亂掉人名**：第二次 19 句對不上名詞表、0 分率 10%、降到 B。50 句一批也出現過一次 4 句人名不對。20 句四次都穩。
- **Claude 50 句一批反而更好**（0.5%／96.5%），翻譯那部分的費用也降（整份報告 US$0.32 對 US$0.53，評審約 US$0.25 兩邊一樣）。只考一次。
- 這份考卷 200 句來自多部片、名詞表跟著每批句子組；真實整集是連續 835 句，名詞表固定，結果可能更好或更壞，正式機實測才算數。

## 費用

考卷評審（Claude）合計約 US$2.2：第一輪 US$0.47（10→「50／100」其實是 20）、補考第一輪 US$0.46（同樣被截成 20）、修好後 US$1.24。Gemini 翻譯在免費方案內。

## Tasks / Subtasks

- [x] T1 考卷工具修正（AC 1、2）
- [x] T2 測試（AC 3）
- [x] T3 量測與報告改名（AC 4、5）
- [ ] T4 改不改預設：等 Alexyu 拍板（見 sprint-status）

## Dev Notes

- 改預設是另一張單：它會動到切批、估價、控速等待、試跑 20 句，以及「每批重複送的說明＋名詞表」的費用估算。

### Time-dependent visual coverage

N/A — 工具與量測。

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（claude-opus-5-5），2026-10-15。

### Completion Notes List

- `cmd/grade/main.go`：`chunkSize := max(translateChunk, cfg.batchSize)`；Claude 的 `--batch-size` 不再擋，報告標 `batch_size`。
- `cmd/grade/merge.go`：`BatchSize` 非 0 且不等於 `SubtitleTranslatorBatchSize` 就拒絕。
- 測試 `TestRun_BatchSizeAboveTheChunkIsHonoured`、`TestMergeReport_RefusesABatchSizeExperiment`；移除舊的「Claude 不准 batch size」那條負向案例。

### Discovery Triage

- ① 考卷工具把 `--batch-size` 截成 20 → 本單 AC 1 修。
- ① merge 沒擋實驗報告 → 本單 AC 2 修。

### File List

- `apps/api/cmd/grade/{main.go,main_test.go,merge.go,merge_test.go}`
- `eval/golden-v2-2026-10/{gemini-3.5-flash-lite-batch20-run1..4,gemini-3.5-flash-lite-batch50-run1..2,gemini-3.5-flash-lite-batch100-run1..2,claude-sonnet-5-batch50}.json`
