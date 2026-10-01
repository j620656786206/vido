# Story sub-7-8b: 等級表內建進 App ＋ CI 一鍵評新模型開 PR — 後端／CI／文件

Status: review

<!-- SM Bob create-story 2026-10-01，由 sub-7-8 拆出；依賴 sub-7-8a（`cmd/grade` 輸出格式）。⚖️ Alexyu 2026-10-01 裁定：不做遠端 feed，只做內建 JSON＋CI 開 PR。行號為 main `4004c7af`。 -->

## Story

身為維護者，
我要「加一個模型 = 按一次 GitHub Actions 按鈕」：它自己跑考卷、自己開 PR 更新等級表，
App 讀等級表而不是寫死在程式碼裡。

## 背景（查到的事）

- `ai/catalog.go:66-90` 的 `modelMetadata` 把 `qualityGrade`／`qualityNote` 寫死；`Catalog()` 直接複製進 `ModelInfo`。`TestCatalog_EveryPricedModelIsDescribed` 守「有價必有描述」。
- `GET /settings/models`（`handlers/model_settings_handler.go`）讀 `ModelCatalogService.Available` → `ai.Catalog()`；FE 已處理 `quality_grade` 缺席＝「尚未評測」。
- repo 現有 workflows 沒有任何 AI key secret（只有 TMDb／TestSprite／DockerHub）。
- `docs/development.md` 只有英文版；本單加的是維護者文件，不是使用者文件（雙語規則不觸發）。

## Acceptance Criteria

1. **等級表檔。** `apps/api/internal/ai/model_ratings.json`（`go:embed`，package init 解析失敗不能啟動）：`{schema: 1, sample_version, ratings: [{model_id, grade, zero_rate, natural_rate, cost_usd, prompt_version, lexicon_version, judge_model, judge_version, graded_at, source: golden|eval-1, note?}]}`。初版保留 eval-1 的 sonnet=A／haiku=B 兩列（`source: eval-1`，note 沿現行 `evalNote`），黃金樣本跑出來的列 `source: golden`，同一 model 以 golden 覆蓋 eval-1。
2. **catalog 改讀表。** `modelMetadata` 移除 `qualityGrade`／`qualityNote`；`Catalog()` 從等級表填 `QualityGrade`／`QualityNote`（golden 列文案「Vido 實測 YYYY-MM，黃金樣本 vN；裁判 <judge_model>」）；`ModelInfo` 不改欄位（0 bump）。等級表裡出現 catalog 沒有的 model → 啟動 warn、不列出。
3. **`cmd/grade --merge`。** 把 7-8a 的輸出 JSON 合併進等級表（同 model 取代、依 model_id 排序、不動其他列）；`grade: incomplete` 的結果拒合並 exit 1。
4. **CI。** `.github/workflows/model-grade.yml`：`workflow_dispatch` 輸入 `model_id`（必填）、`budget_usd`（預設 0.50）；用 repo secret `GRADE_ANTHROPIC_API_KEY` 跑 `go run ./cmd/grade` → `--merge` → 以 `gh pr create` 開 PR（分支 `chore/model-grade-<model>-<date>`，標題「chore(model-grade): <model> 評為 X」，內文貼結果 JSON 與成本）；失敗不開 PR。不在 PR CI 裡自動跑（要花錢）。
5. **文件。** `docs/development.md` 加「Model quality grades」一節：怎麼本機跑、怎麼按 workflow、等級怎麼算、裁判偏誤、為什麼不是遠端 feed。
6. **測試。** 等級表 schema／解析；每個 golden 列的 model 都在 catalog；`Catalog()` 填值與文案；`--merge` 取代／排序／拒 incomplete。

## Tasks / Subtasks

- [ ] Task 1 — 等級表 embed＋catalog 改讀（AC #1, #2）
- [ ] Task 2 — `--merge`（AC #3）
- [ ] Task 3 — workflow＋secret 說明（AC #4）
- [ ] Task 4 — 文件＋測試（AC #5, #6）

## Dev Notes

- secret 名稱 `GRADE_ANTHROPIC_API_KEY` 由 Alexyu 在 repo settings 新增；本單只能寫 workflow，無法驗證真跑——Completion Notes 要寫明「workflow 未在 CI 真跑」。
- 原單的遠端 feed、簽章、24h 快取：**不做**（裁定）。若將來要做，另立單並進 security posture。

## Dev Agent Record

### Agent Model Used

Claude Fable 5.1（Amelia）

### Completion Notes List

- 4 task 全數交付。**workflow 沒有在 CI 真跑**（要先有 repo secret `GRADE_ANTHROPIC_API_KEY`，Alexyu 在 Settings → Secrets 新增後按一次 Actions → Model Grade 才算驗過）。本機只驗了 YAML 可解析。
- **等級表**：`ai/model_ratings.json`（`go:embed`，schema 1，`DisallowUnknownFields`，啟動時解析失敗 panic）。初版兩列 eval-1（sonnet A／haiku B，含 10,304 句、花費、`judge_model: human`、原 `evalNote` 文案）；golden 列多帶 `sample_version`，同一 model 兩種來源並存時 golden 覆蓋。`ParseRatings` 拒：非 A/B/C（含 incomplete）、壞日期、未知來源、率超出 [0,1]、重複列、golden 列缺 judge／sample／prompt 版本、eval-1 列缺 note、未知欄位。
- **catalog 改讀表**：`modelMetadata` 拿掉 `qualityGrade`／`qualityNote` 兩欄；`Catalog()` 從 `Ratings()` 填 `QualityGrade`／`RatingNote(r)`（golden 文案「Vido 實測 YYYY-MM，黃金樣本 v1；裁判 claude-sonnet-5」）。`ModelInfo` 零改動（0 bump）。表裡有 catalog 不認識的 model → 啟動 warn 一次、不列出；測試改成反向守衛「每個被評的 model 都要在 catalog」。
- **`cmd/grade --merge report.json [--ratings path]`**：不需要 `--model`／key；拒 `incomplete`、拒 `sample_version` 與本 build 不同的報告（要重跑）；`ai.MergeRating` 同 model＋同 source 取代、依 model_id 排序、pretty JSON＋結尾換行、寫回前再 `ParseRatings` 一次。
- **workflow `model-grade.yml`**：`workflow_dispatch` 輸入 `model_id`／`budget_usd`（0.50）／`level`（standard）；沒 secret 先 `::error` 退出；裝 `opencc` 讓裁判看交付文字；結果寫 `$GITHUB_STEP_SUMMARY`；只有 `level=standard` 才 merge＋`peter-evans/create-pull-request@v6` 開 PR（分支 `chore/model-grade-<model>`，同名重跑更新同一個 PR，`add-paths` 只收 `model_ratings.json`，label `requires-manual-review`）；`concurrency` 依 model 分組。cmd/grade 預算打到上限 exit 3 → step 失敗 → 不開 PR。
- **docs**：`docs/development.md` 新增「Model quality grades」一節（怎麼算、本機怎麼跑、按鈕在哪、secret、為什麼不做遠端 feed）。英文維護者文件，不觸發雙語規則。
- 測試：`ai` 5 條（embed 載入含 eval-1、九種壞檔＋未知欄位、golden 覆蓋 eval-1 兩種順序、`RatingNote`、merge 取代／排序／拒 incomplete／並存）、catalog 兩條改寫；`cmd/grade` 2 條（merge flag 不驗 model、merge 寫 golden 列＋拒 incomplete＋拒異版樣本）。全量 `go test ./...` 綠、vet 綠、prettier 綠。
- 🔗 AC Drift：AC #2 文案原寫「黃金樣本 vN」，實作從 `sample_version`（`golden-v1`）推出「黃金樣本 v1」；AC #3 多擋一種情況（報告的 `sample_version` 與 build 不同）——避免舊報告合進新考卷。

### Discovery Triage

- 無新單。`backlog-gemini-translation-dispatch` 解掉時，workflow 的說明與 `cmd/grade` 的 Claude-only 檢查一併放開。

### File List

- apps/api/internal/ai/model_ratings.json、ratings.go、ratings_test.go、catalog.go、catalog_test.go
- apps/api/cmd/grade/main.go、merge.go、merge_test.go
- .github/workflows/model-grade.yml
- docs/development.md
- _bmad-output/implementation-artifacts/sub-7-8b-model-ratings-embed-ci.md、sub-7-8a-golden-sample-and-grader.md、sprint-status.yaml

## Change Log

| Date       | Change                                   |
| ---------- | ---------------------------------------- |
| 2026-10-01 | create-story（SM Bob，自 sub-7-8 拆出）。 |
| 2026-10-01 | dev-story（Amelia）→ review。 |
