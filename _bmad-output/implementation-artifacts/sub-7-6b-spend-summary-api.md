# Story sub-7-6b: 月報端點——這個月花了多少、依模型、略過與快取省了多少 — 後端

Status: review

<!-- SM Bob create-story 2026-10-01，由 sub-7-6 拆出；依賴 sub-7-6a（ledger 欄位與收據）。行號為 main `34b59615`。 -->

## Story

身為 BYOK 的使用者，
我要一個端點告訴我「這個月翻譯花了多少、語音辨識花了多少、各模型多少、略過省了多少、快取省了多少」，
沒記錄的東西寫「—」不寫 0，估算要說是估算。

## 背景（查到的事）

- ledger（7-6a）：`subtitle_runs.route／cache_hit_cues／batch_id`＋既有 `spent_usd／budget_usd／model_id／cue_count／status／completed_at`。時間欄是 Go 文字，**只能用 UTC 文字做字典序比較**（沿 `CompletedMediaRefsSince` 的寫法）。
- 估價公式 `services/generation_candidates.go:1009 estimateUSD(route, minutes, asrRate, model)` 是**每分鐘片長**：extract＝mins×rate、asr＝mins×asrRate＋mins×rate；片長階梯 `duration_seconds`（migration 035）→ TMDb runtime → 45 分假設（`unknownRuntimeMinutes`）。
- Rule 23：「本月」以月為界要注入時鐘（`HomeSummaryService` 的 `now` 欄位＋`now.Location()` 算月初）。
- Rule 7：新錯誤碼走 `SUBTITLE_`／`VALIDATION_` 既有前綴；`?period=` 不合法用 `VALIDATION_INVALID_FORMAT`。
- swag 只寫註解不產生（9R-17 AC #3 慣例）。

## Acceptance Criteria

1. `GET /api/v1/subtitles/spend?period=month` `[@contract-v1]` 回 `{period, from, to, translated_runs, translated_usd, asr_runs, asr_usd, skipped_deliver_count, skipped_saved_usd_estimate, skipped_saved_runtime_assumed(bool), cache_hit_cues, cache_measured_runs, cache_saved_usd_estimate, by_model:[{model_id, runs, usd}], by_batch?}`。`translated` = route translate；`asr` = route asr；只算 `completed`；`spent_usd` NULL 的列不計入金額但計入 runs 數並回 `unpriced_runs`。
2. **略過省下**：`route ∈ {deliver_direct, convert_then_deliver}` 且 completed 的列，各以片長階梯算「若翻譯要花多少」（`estimateUSD(extract, minutes, _, model_id)`）；任一列片長是假設的 → `skipped_saved_runtime_assumed=true`（前端據此加 `≈`）。
3. **快取省下**：對 `cache_hit_cues IS NOT NULL` 且 `cue_count > cache_hit_cues` 的 translate 列：`hits × (spent_usd / (cue_count − cache_hit_cues))`；沒有任何量到的列 → `cache_saved_usd_estimate` 為 null、`cache_measured_runs=0`（前端顯示「—」）。
4. **本次批次**：`?batch_id=` 可選；回 `by_batch{batch_id, runs, usd, cue_count, cache_hit_cues, model_id}` 供 F8 收據。
5. 月界以伺服器時區的當月，時鐘注入可測；`period` 只接受 `month`（其他 400 `VALIDATION_INVALID_FORMAT`）。
6. **測試**：repository 月彙總 SQL 真 sqlite（含跨月邊界、UTC 文字比較、NULL spent、各 route）、估算公式（假設片長旗標）、端點 shape、400。

## Tasks / Subtasks

- [x] Task 1 — repository 月彙總查詢（AC #1, #3, #4）
- [x] Task 2 — 略過省下估算（片長階梯、旗標）（AC #2）
- [x] Task 3 — handler ＋ 契約戳記 ＋ 測試（AC #5, #6）

## Dev Notes

- 不另寫費率表：全部走 `ai.PricingFor`／`translationRatePerMinute`。
- 舊列（7-6a 之前）沒有 route：計入 `unrouted_runs`，金額計入 `translated_usd`？——**不**，誠實規則：另列 `unrouted_usd`，前端標「早期紀錄」。

## Dev Agent Record

### Agent Model Used

Claude Fable 5.1（Amelia）

### Completion Notes List

- 與 AC 的兩處收斂：① `cache_saved` 只對「有量到且模型真的翻了幾句」的列算每句均價（全部命中、$0 的列量到但省下算 0——沒有均價可推，誠實）；② `by_batch` 的 `usd` 含 failed／paused 列（錢已花），`cue_count`／cache 只算 completed。
- 新測試 13 條：repository 2（半開區間＋本地時區正規化＋失敗列排除；batch 查詢）、services 9（分流與 by_model 排序、未定價／未標路線分開報、快取省下三種列、略過省下三階梯與旗標、當月邊界（台北 10/1 00:30 → UTC 9/30 16:00）、批次收據、多模型批次、ledger 錯誤上拋）、handler 3（200 shape 含 null、400、500）。
- 🔗 AC Drift: NONE。📎 Contract Stamps: `SubtitleSpendSummary` [@contract-v1]（doc 註解＋swag 註解）。
- 全量：`pnpm nx test api` 綠、go vet、staticcheck、lint:all、prettier。

### Discovery Triage

- 無新單。

### File List

- apps/api/internal/repository/subtitle_run_repository.go、subtitle_run_repository_test.go
- apps/api/internal/services/subtitle_spend_service.go、subtitle_spend_service_test.go
- apps/api/internal/handlers/subtitle_spend_handler.go、subtitle_spend_handler_test.go
- apps/api/cmd/api/main.go
- _bmad-output/implementation-artifacts/sub-7-6b-spend-summary-api.md、sprint-status.yaml

## Change Log

| Date       | Change                                    |
| ---------- | ----------------------------------------- |
| 2026-10-01 | create-story（SM Bob，自 sub-7-6 拆出）。 |
| 2026-10-01 | dev-story（Amelia）→ review。 |
