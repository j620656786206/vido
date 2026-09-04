# Story 6.8a: 每次產生可選模型 + 依模型估價；預設改 Sonnet（後端）

Status: review

## Story

As a BYOK NAS owner,
I want to choose which model translates each batch and see the price of each choice before I confirm,
so that the default gives me the quality eval-1 proved, and switching to a cheaper model is my visible, informed decision — never a silent config flip.

## Context — 這個 story 為什麼存在

eval-1 裁定（Alexyu 2026-09-03）：**預設改 `claude-sonnet-5`**（全檔評分 0 分率 1.3% vs Haiku 3.6%、2 分率 89.6% vs 71.8%，任何評分版本都穩過），**Haiku 由使用者自選省錢**，確認框必須明示 2.7× 價差（Sonnet ≈ $0.48/hr 片長、Haiku ≈ $0.18/hr）。

現況：模型是**程序級**——`ClaudeProviderHolder` fingerprint `key|model`（`claude_provider_holder.go:80`）、pipeline 的 model 來源是 `WithModelSource(claudeHolder.EffectiveModel)`（sub-6-5 起；`WithModelID` 只剩測試用的常數包裝）、`GenerationBatchStartRequest` 只有 `scope/media_ids/budget_usd`（`generation_batch_handler.go:58`）。估價 `GenerationCandidate.EstimatedUSD` 單一數字（`generation_candidates.go:102`）。

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

- [x] **Task 1 — 預設與清單（AC: #1, #2）**
- [x] **Task 2 — 依模型估價（AC: #3）**
- [x] **Task 3 — 請求帶 model_id 與 pipeline 貫穿（AC: #4）**
- [x] **Task 4 — `GetFor` 與 ctx model（AC: #5, #6）**
- [x] **Task 5 — 測試（AC: #7）**

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

Claude Opus 5（dev-story，2026-09-04）

### Debug Log References

- `go test ./...`（apps/api）全綠；`pnpm nx run api:lint`（go vet + staticcheck）綠；`prettier --check` 綠。

### Completion Notes List

- **Task 1（AC #1/#2）**：`ai.DefaultClaudeModel` → `claude-sonnet-5`（註解寫入 eval-1 數字與「為什麼預設是品質、省錢是使用者的可見選擇」）。新 `ai/catalog.go`：`ModelInfo` `[@contract-v1]`、`modelMetadata`（display/tier/measured grade）、`Catalog()`、`ProviderOf`、`IsSelectableModel`。**兩張表同一組 key、一起維護**，`TestCatalog_EveryPricedModelIsDescribed` 是守門測試（沿用 sub-6-2 的 timeout 表手法）。`retired` 旗標：`gemini-2.0-flash` 保留定價列（舊 run 仍要按當時費率計帳）但**絕不出現在可選清單**（已被 Google 關閉，選了必 404）。`quality_grade` 只填實測的 sonnet=A／haiku=B，未評測**留空不填**（留空＝「還沒評」，不是「一樣好」）。新 `services.ModelCatalogService`（`Available`／`Supports`／`DefaultModel`）＋ `GET /api/v1/settings/models`（新 handler，**註冊順序必須在 settingsHandler 之前**，否則被 `/settings/:key` 吃掉）。
- **Task 2（AC #3）**：**成本模型重新校準**（見下方「AC 偏離與重大裁定」）。`GenerationCandidateResult` additive `estimates_by_model`（`{model: {total_usd, per_candidate}}`）與 `estimated_minutes_by_model`（片長 × 17%／11%，eval-1 實測）。`EstimatedUSD`／`estimated_total_usd` 語意不變＝**預設模型**的估價（舊 FE 不壞）。unwritable 列一樣不計入任何模型的總額（sub-6-1 語意）。ASR 部分不隨模型變（不同供應商、按音檔分鐘計費），有測試釘住。
- **Task 3（AC #4）**：`GenerationBatchStartRequest.ModelID`／`SubtitlePipelineRunRequest.ModelID`（均 optional，additive 不 bump）；兩處都在**動任何東西之前**用 `ModelValidator`（`ModelCatalogService.Supports`）驗證，不合法 → 400 `VALIDATION_INVALID_FORMAT`，訊息指向 `GET /settings/models` 而不是回嘴使用者拼錯（金鑰被移除時，本來合法的 model 會瞬間變不合法）。`ProcessItemOptions.ModelID` → `ai.WithModelID(ctx)` → `Pipeline.currentModelID(ctx)`（RunVersion＋segment cache key）。批次的選擇掛在 batch ctx 上（與共用 Budget 同一手法），所以 runner port 簽名不動。
- **Task 4（AC #5/#6）**：`ai.WithModelID`／`ModelIDFromContext`（ctx value，理由寫在 `model_context.go`：否則 `ChunkTranslator`／`TranslateContext` 兩個 stamped 介面要為了一個沒人用的參數而 bump）。`ClaudeProviderHolder` 的單一 `cached` 改為**有界 LRU**（`maxCachedClients = 4`，`GetFor(ctx, model)`，`Get` 讀 ctx）；所有 client 共用**同一個 Governor**（測試用 `assert.Same` 釘住 —— 換 client 不能等於換預算池）。未知 model 在 holder 就擋（`ErrAIModelNotFound`），不讓它變成使用者付錢才發現的 404。`RecordLLM` 自然記到正確 model（provider 自己帶）。
- **Task 5（AC #7）**：(a) 清單端點：有／無 Gemini key 的差異、keyless → 空陣列 200、未評測不吐 grade、`is_default`；(b) 估價：兩模型比值 ≈ 2.67×（eval-1 實測而非 3× 定價比）、逐列加總＝footer、ASR 部分不動、處理時間 17%／11%、無 catalog 仍報預設一個數字；(c) 批次帶 haiku → 每個 item 的 ctx 都是 haiku；ProcessItem 帶 model → run row 記該 model、chunk 請求的 ctx 帶該 model；(d) 兩個端點非法 model → 400 且**什麼都沒開始**；(e) `GetFor` LRU 上限、LRU 順序（用過的不會被踢）、Governor 同一實例、未知 model 被拒；(f) 換模型重跑不會命中舊模型的 segment cache；(g) 全回歸綠。

#### AC 偏離與重大裁定（需 Alexyu 過目）

1. **成本模型重新校準（超出 AC #3 字面）**。AC 說「用 `PricingFor(model)` × 既有 token 估算」，但既有估算是 `translationUSDPerMinute = 0.0004`（M1 pilot 單一影片校準）。對照 eval-1 的 12h20m 實測（Haiku $2.229／Sonnet $5.951），這個常數**低估約 7 倍** —— 90 分鐘的片會報 $0.04，實際 Haiku $0.27、Sonnet $0.72。只按比例縮放會讓兩個模型**都**錯 7 倍。既然整個 story 的目的是「按下去之前看到的金額就是你要付的金額」，我把它換成 eval-1 實測的 per-model 費率（haiku 0.00301／sonnet 0.00804 每分鐘片長），未實測模型由 **Sonnet 錨點**按混合定價比縮放、且**永不低於錨點**（估高使用者不會被驚嚇，估低會）。原常數自己的註解就寫著「等有真實用量資料再校準」。
2. **`estimates_by_model` 放在 `result` 而非 `AnalysisSnapshot`**。AC 寫在 snapshot 上，但那組數字**就是**這份報價：sweep 被取消／失敗時 `result` 會被清成 nil，報價必須跟著消失。掛在 snapshot 層要多一份手動失效邏輯，而「顯示過期價格」正是這個畫面最不能犯的錯。FE 讀 `snapshot.result.estimates_by_model`，一樣一跳。
3. **Gemini 不在 `KeyResolver` 的封閉 key 集合裡**（只有 claude／tmdb／openai），AC #2 假設可以用 `KeyResolver.Has` 判定。改用注入的 env 判定（`GEMINI_API_KEY`），並立案 `backlog-gemini-key-in-resolver`。
4. **`CLAUDE_MODEL` 在 `docs/deployment*.md` 根本沒有段落**（AC #1 假設有）。這次補上表格列與說明段。

- 🔗 AC Drift: NONE against other stories (checked: 'EstimatedUSD|estimated_total_usd|ProcessItemOptions|Start\(' — sub-4-1 AC #7 additive、sub-4-2 `[@contract-v3]` additive、sub-6-5 effective-model 單一真相保留且擴充為 per-run；本 story 自身的四點偏離見上)
- 📎 Contract Stamps: FOUND (sub-4-1 AC #7 `[@contract-v1]` additive ack；sub-4-2 `[@contract-v3]` additive ack；`ChunkTranslator`／`TranslateContext` `[@contract-v1]` **未動**，這正是走 ctx value 的理由；新 `ai.ModelInfo` 與 models 端點自帶 `[@contract-v1]`)
- 🎭 A11y Pre-Flight: N/A（純後端）
- 🔌 Route Sync: 新增 `GET /api/v1/settings/models`（Swagger 註解完整；`docs/swagger.json` 全庫已久未重生成，沿用現行慣例只維護註解）
- 🎨 UX Verification: N/A（純後端；FE 為 sub-6-8b）

### Discovery Triage

- ① expand-scope-in-place — 估價常數低估 7 倍 → 依 eval-1 實測重新校準（見上 #1）。
- ① expand-scope-in-place — `CLAUDE_MODEL` 無文件段落 → 補 `docs/deployment.md`。
- ③ backlog-with-carry-forward-link — Gemini 不在 KeyResolver 封閉集合 → `backlog-gemini-key-in-resolver`。
- ③ backlog-with-carry-forward-link — `docs/deployment.zh-TW.md` 仍不存在 → 既有 `backlog-deployment-doc-zh-tw-twin` RE-HIT。

### Change Log

| Date | Change |
| --- | --- |
| 2026-09-04 | Task 1 — 預設改 Sonnet；`ai/catalog.go`；`services.ModelCatalogService`；`GET /settings/models`。 |
| 2026-09-04 | Task 2 — 成本模型 per-model 重新校準；`estimates_by_model`／`estimated_minutes_by_model`。 |
| 2026-09-04 | Task 3 — 兩個端點的 `model_id` + 400 驗證；`ProcessItemOptions.ModelID`；batch ctx。 |
| 2026-09-04 | Task 4 — `ai.WithModelID`；holder `GetFor` + 有界 LRU + 共用 Governor。 |
| 2026-09-04 | Task 5 — 五個套件的測試；`docs/deployment.md`；project-context。 |

### File List

- `apps/api/internal/ai/catalog.go`、`catalog_test.go`、`model_context.go`（new）
- `apps/api/internal/ai/claude.go`、`budget.go`（modified）
- `apps/api/internal/services/model_catalog.go`、`model_catalog_test.go`（new）
- `apps/api/internal/services/claude_provider_holder.go`、`generation_candidates.go`、`generation_batch.go`（modified）+ 各自 `_test.go`
- `apps/api/internal/handlers/model_settings_handler.go`、`model_settings_handler_test.go`（new）
- `apps/api/internal/handlers/generation_batch_handler.go`、`subtitle_pipeline_handler.go`（modified）+ 各自 `_test.go`、`route_c_uuid_integration_test.go`
- `apps/api/internal/subtitle/pipeline.go`、`process_item.go`、`segment_cache.go`（modified）+ `process_item_test.go`、`segment_cache_test.go`、`pipeline_transient_test.go`
- `apps/api/cmd/api/main.go`（modified）
- `docs/deployment.md`、`project-context.md`、`_bmad-output/implementation-artifacts/sub-6-8a-per-run-model-selection-backend.md`、`sprint-status.yaml`
