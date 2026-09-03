# Story 6.8a: 每次產生可選模型 + 依模型估價；預設改 Sonnet（後端）

Status: ready-for-dev

## Story

As a BYOK NAS owner,
I want to choose which model translates each batch and see the price of each choice before I confirm,
so that the default gives me the quality eval-1 proved, and switching to a cheaper model is my visible, informed decision — never a silent config flip.

## Context — 這個 story 為什麼存在

eval-1 裁定（Alexyu 2026-09-03）：**預設改 `claude-sonnet-5`**（全檔評分 0 分率 1.3% vs Haiku 3.6%、2 分率 89.6% vs 71.8%，任何評分版本都穩過），**Haiku 由使用者自選省錢**，確認框必須明示 2.7× 價差（Sonnet ≈ $0.48/hr 片長、Haiku ≈ $0.18/hr）。

現況：模型是**程序級**——`ClaudeProviderHolder` fingerprint `key|model`（`claude_provider_holder.go:80`）、pipeline `WithModelID` 在 boot 接線（`main.go:697`）、`GenerationBatchStartRequest` 只有 `scope/media_ids/budget_usd`（`generation_batch_handler.go:58`）。估價 `GenerationCandidate.EstimatedUSD` 單一數字（`generation_candidates.go:102`）。

**Depends on:** sub-6-5（effective model 單一真相）。**Consumed by:** sub-6-8b（前端）。

## Acceptance Criteria

1. **預設改 Sonnet。** `ai.DefaultClaudeModel = "claude-sonnet-5"`（`claude.go:31`）；`defaultLLMPricing` 已有列。`docs/deployment*.md` 的 `CLAUDE_MODEL` 段更新預設值與理由（引用 eval-1 數字）。`CLAUDE_MODEL` env 仍可覆蓋（operator 全域預設）。

2. **可選模型清單端點。** `GET /api/v1/settings/models` `[@contract-v1]` 回 `{models:[{id, provider, display_name, tier, input_per_1m, output_per_1m, is_default, quality_grade?, quality_note?}]}`。清單來源＝`defaultLLMPricing` 的 key 過濾出**目前有 key 的 provider**（Claude 有 key → claude-*；Gemini 有 key → gemini-*；sub-5-2 `KeyResolver.Has` 判定）。`quality_grade` 本 story 只填 eval-1 實測的兩個（sonnet=A、haiku=B）與註記「Vido 實測 2026-09」；其餘留空（P1-8 接手評分 feed）。Swagger 完整。

3. **候選估價依模型。** `AnalysisSnapshot` 加 `estimates_by_model: {<model_id>: {total_usd, per_candidate?: {media_id: usd}}}`（additive，sub-4-1 `[@contract-v1]` 不 bump、ack + Change Log）；`EstimatedUSD` 既有欄位維持＝預設模型估價（舊 FE 不壞）。估價用 `PricingFor(model)`（`budget.go` 已為 sub-4-1 曝露）× 既有 token 估算；ASR 部分不隨模型變。同時 additive 加 `estimated_minutes_by_model`（依 eval-1 實測處理速度：片長 × 11%（haiku）／17%（sonnet），未知模型用 sonnet）。

4. **批次與單項帶 `model_id`。** `GenerationBatchStartRequest` 加 `model_id string`（可選；空＝effective default）——sub-4-2 `[@contract-v3]` additive 不 bump；FR12 單項端點同樣加。驗證：不在 AC #2 清單內 → 400 `VALIDATION_INVALID_FORMAT`「不支援的模型」。`ProcessItemOptions` 加 `ModelID`；pipeline 的 `runVersion.ModelID` 以 opts 優先、否則 effective default（sub-6-5）。

5. **provider 依 model 取得。** `ClaudeProviderHolder` 加 `GetFor(ctx, model) (ai.TextCompleter, error)`：以 `key|model` fingerprint 快取**有界**（最多 4 個 model，LRU，Rule 14），共用同一 Governor（`claude.go:105` 註解：新 client 絕不能等於新 budget pool）。`TranslationService.TranslateChunk` 需要知道 model → 透過 ctx value（`ai.ModelFromContext`）或擴 `ChunkTranslator` 簽名；**裁定：ctx value**（不動 stamped 介面）。Gemini 走 `factory.go` 同型式。

6. **成本記帳與 cache 正確。** `RecordLLM(p.model, …)` 已用實際 model；segment cache key 含 `ModelID` → 換模型重跑不會誤命中舊譯（既有語意，補測試斷言）。

7. **測試。** (a) models 端點：有／無 Gemini key 的清單差異、is_default 正確；(b) 估價：兩模型數字比例 ≈ 定價比、ASR 部分不變；(c) batch 帶 haiku → run 記 haiku、provider fingerprint 命中；(d) 非法 model → 400；(e) `GetFor` LRU 上限與 Governor 同一實例斷言；(f) 全回歸。

## Tasks / Subtasks

- [ ] **Task 1 — 預設與清單（AC: #1, #2）**
- [ ] **Task 2 — 依模型估價（AC: #3）**
- [ ] **Task 3 — 請求帶 model_id 與 pipeline 貫穿（AC: #4）**
- [ ] **Task 4 — `GetFor` 與 ctx model（AC: #5, #6）**
- [ ] **Task 5 — 測試（AC: #7）**

（後端 5 task；前端另立 sub-6-8b。）

## Dev Notes

### 既有可重用零件

| 需求 | 現成零件 |
| --- | --- |
| 定價表／估價同源 | `ai/budget.go` `defaultLLMPricing`、`PricingFor` |
| holder 重建與 fingerprint | `claude_provider_holder.go:68-97` |
| additive 信封先例 | sub-5-1 AC #5、sub-5-3 series 欄位 |
| key 判定 | `KeyResolver.Has(ctx, services.KeyClaude)`（main.go:665） |
| 處理速度數據 | eval-1「每部花費／片長／處理時間」表 |

### Rule 20 契約盤點（dev 開工前 grep 確認）

- sub-4-1 AC #7 `[@contract-v1]`（snapshot）— additive ack。
- sub-4-2 `[@contract-v3]`（batch start request）— additive ack。
- `ChunkTranslator`／`TranslateContext` `[@contract-v1]` — 不動。

### 為什麼不做「設定頁全域切模型」就好

BYOK 下價差是每一次按下去的事，不是一次性設定。全域預設（env）保留給 operator；**每次產生的選擇**是本 story 的產品語意。

### Time-dependent visual coverage

- N/A — 純後端。

### References

- eval-1「後續 Backlog」P0-8（Alexyu 改裁：預設 Sonnet）；「每部花費／片長／處理時間」表
- `apps/api/internal/handlers/generation_batch_handler.go:58`、`apps/api/internal/services/generation_candidates.go:93-176`、`apps/api/internal/services/claude_provider_holder.go`

## Dev Agent Record

### Agent Model Used

### Completion Notes List

### Discovery Triage

- （dev 填）

### File List
