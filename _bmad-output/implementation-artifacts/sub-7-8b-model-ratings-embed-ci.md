# Story sub-7-8b: 等級表內建進 App ＋ CI 一鍵評新模型開 PR — 後端／CI／文件

Status: ready-for-dev

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

### Completion Notes List

### Discovery Triage

### File List

## Change Log

| Date       | Change                                   |
| ---------- | ---------------------------------------- |
| 2026-10-01 | create-story（SM Bob，自 sub-7-8 拆出）。 |
