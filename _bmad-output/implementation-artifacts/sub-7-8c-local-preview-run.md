# Story sub-7-8c: 還沒評測的模型可以「試跑 20 句」、結果標「你的實測」— 後端＋前端

Status: ready-for-dev

<!-- SM Bob create-story 2026-10-01，由 sub-7-8 拆出；依賴 sub-7-8a（考卷 embed）與 7-8b（等級表讀法）。行號為 main `4004c7af`。
     ⚠️ Task 0 設計稿：ModelPicker 列的三態（尚未評測＋按鈕／試跑中／你的實測）目前**沒有任何 .pen 稿**；要先請 Sally 補 F16／F19 的該列狀態與 J 系列成本按鈕規格，Alexyu 跑 inline agent，再開工 FE。 -->

## Story

身為選到一個「尚未評測」模型的使用者，
我要能花一點小錢用考卷前 20 句試跑它，看到 0 分率、2 分率和真的花了多少，
結果只留在我這台機器、標「你的實測」，不會被當成 Vido 的官方等級。

## 背景（查到的事）

- FE 已有占位文案：`components/subtitle/consent/ModelPicker.tsx:138-140`「· 可花約 $0.01 試跑 20 句」（純文字，不可按；註解點名 P1-8 接手）。`consentSelection.ts:639 ModelChoice` 有 `qualityGrade?／qualityNote?／isBestGrade`。
- 「約 $0.01」只對 Sonnet 等級成立（Haiku 20 句 ≈ $0.004、Opus ≈ $0.06）→ 金額要從 `input_per_1m／output_per_1m` 用考卷 20 句的 token 估算，不寫死。
- 試跑**不是**媒體 run：`subtitle_runs` 以 media 為主鍵，不塞；花費記在回應與 settings 即可（sub-7-6 月報不含試跑——誠實規則，文件寫明）。
- settings 表 `SettingsRepository.SetString`（`repository/settings_repository.go:233`）可存 `models.local_grade.<model_id>` JSON；`GET /settings/models` 讀出填 `local_grade?`（additive，`ModelInfo` 加選填欄位 = 0 bump）。
- 路由注意：`modelSettingsHandler.RegisterRoutes` 必須在 `settingsHandler` 之前（`main.go:1244` 既有註記），新 POST 同組。

## Acceptance Criteria

1. **端點。** `POST /api/v1/settings/models/:id/preview` `[@contract-v1]`：`:id` 必須 `ModelCatalogService.Supports`（否則 400 既有錯誤碼）；用考卷前 20 句（embed）跑 7-8a 的翻譯＋規則層＋裁判；`ai.Budget` 上限 = 估價 ×3（最多 $0.20）；回 `{model_id, cues: 20, zero_rate, natural_rate, cost_usd, judge_model, graded_at}`；同時寫入 settings `models.local_grade.<id>`。同一 model 60 秒內重複呼叫 429（省錢）。
2. **清單帶你的實測。** `GET /settings/models` 每列多 `local_grade?: {zero_rate, natural_rate, cost_usd, graded_at}`（additive）。官方 `quality_grade` 存在時 `local_grade` 仍回，FE 以官方為主。
3. **估價。** `GET /settings/models` 每列多 `preview_estimate_usd`（additive）：考卷 20 句的估計 token × 該模型費率，供按鈕文案。
4. **FE 三態。** ModelPicker 無官方等級的列：「尚未評測 · 試跑 20 句（約 $X）」按鈕 → 試跑中（按鈕 disabled＋spinner＋「約 1 分鐘」）→ 「你的實測：0 分 x%・2 分 y%・花 $z」（`--text-secondary`，不給 A/B/C 字母，不套官方徽章色）；失敗回「試跑失敗，沒有扣款」或帶實際金額。按鈕在 consent 對話框內，不觸發外層 submit。
5. **設計稿。** Task 0：Sally 補 F16-D／F16-M 該列三態＋規格註記（J 系列成本按鈕語彙），Alexyu 跑 inline agent，截圖入 repo，Sally review 後才開工 Task 3。
6. **測試。** handler：400／429／200 shape／settings 寫入／預算超額回 incomplete；FE：三態 spec、按鈕不冒泡、金額文案讀 `preview_estimate_usd`；gallery fixture 三張。

## Tasks / Subtasks

- [ ] Task 0 — 設計稿（Sally → Alexyu inline agent → 截圖）（AC #5）
- [ ] Task 1 — preview 端點＋settings 寫入＋節流（AC #1）
- [ ] Task 2 — `GET /settings/models` 加 `local_grade`／`preview_estimate_usd`（AC #2, #3）
- [ ] Task 3 — FE 三態＋fixtures＋specs（AC #4, #6）

## Dev Notes

- BE 2＋FE 1＋設計 1，門檻內不再拆；FE 若因三態與手機版超過 3 task 再拆 c2。
- 試跑花的錢不進 sub-7-6 月報：文案與 docs 要講「試跑費用不計入本月字幕花費」。

## Dev Agent Record

### Agent Model Used

### Completion Notes List

### Discovery Triage

### File List

## Change Log

| Date       | Change                                   |
| ---------- | ---------------------------------------- |
| 2026-10-01 | create-story（SM Bob，自 sub-7-8 拆出）。 |
