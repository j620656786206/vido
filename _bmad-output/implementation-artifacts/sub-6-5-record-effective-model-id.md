# Story 6.5: `subtitle_runs.model_id` 永遠記錄實際模型 —— 預設模型不再是空字串（後端）

Status: ready-for-dev

## Story

As a BYOK NAS owner,
I want every subtitle run to record the model that actually produced it,
so that I can tell what I paid for without reading container logs.

## Context — 這個 story 為什麼存在

eval-1 產品問題 5：用預設模型時 `subtitle_runs.model_id` 是空字串。鏈：`main.go:678` `modelID := cfg.GetClaudeModel()`（`config/api_keys.go:70`，env 未設時回 `""`）→ `subtitle.WithModelID("")` → `runVersion.ModelID=""` → 寫入 run。實際模型在 `ClaudeProvider.model`（`DefaultClaudeModel = "claude-haiku-4-5"`，`claude.go:31`）。

副作用不只報表：`RunVersion.ModelID` 也是 segment cache key 與 `FindCompletedRun` 的比對欄位（`segment_cache.go:148`、`subtitle_run_repository.go:217`）。空字串 ≠ 實際模型，意味著哪天預設模型改了，舊 cache 會被誤命中。

## Acceptance Criteria

1. **單一真相來源。** `ai.ClaudeProvider` 加 `Model() string` accessor；`ClaudeProviderHolder` 加 `EffectiveModel(ctx) (string, error)`（回傳 live client 的 model）。`main.go` 的 `WithModelID` 改用 holder 的 effective model；env 未設時得到 `claude-haiku-4-5`（或 sub-6-8a 落地後的預設）。

2. **hot-reload 一致。** holder 因 key/model 變更重建 client 時（fingerprint `key|model`），pipeline 讀到的 model id 必須跟著變——因此 `WithModelID` 改為接受 `func() string`（或 pipeline 在 run 建立時向 holder 詢問），**不得**在 boot 時快照一個字串。

3. **回填。** additive migration：`UPDATE subtitle_runs SET model_id='claude-haiku-4-5' WHERE model_id='' AND status='completed'`（eval-1 之前唯一可能的預設）。migration 註解記錄依據。

4. **測試。** (a) env 未設 → run 記 `claude-haiku-4-5`；(b) env 設 sonnet → 記 sonnet；(c) holder 重建後新 run 記新 model；(d) migration 回填只動空字串列；(e) segment cache key 含實際 model（既有 test 調整）。

## Tasks / Subtasks

- [ ] **Task 1 — accessor 與注入（AC: #1, #2）**
- [ ] **Task 2 — migration（AC: #3）**（Rule 15：SELECT/scan 已含 `model_id`，只回填）
- [ ] **Task 3 — 測試（AC: #4）**

## Dev Notes

- `WithModelID` 是 sub-5-1 引用的接線先例（`pipeline.go:343`）；改簽名要 grep 所有呼叫點（main.go:697 + tests）。
- 與 sub-6-8a（per-run 模型選擇）的關係：6-8a 落地後 run 的 model 來自請求；本 story 提供「沒指定時的 effective 預設」，6-8a 疊在上面。**先做本 story**。

### Time-dependent visual coverage

- N/A。

### References

- eval-1「產品問題 5」；後續 Backlog P0-5；`apps/api/cmd/api/main.go:678,697`、`apps/api/internal/services/claude_provider_holder.go:56-97`

## Dev Agent Record

### Agent Model Used

### Completion Notes List

### Discovery Triage

- （dev 填）

### File List
