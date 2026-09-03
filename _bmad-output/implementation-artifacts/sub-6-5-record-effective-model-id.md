# Story 6.5: `subtitle_runs.model_id` 永遠記錄實際模型 —— 預設模型不再是空字串（後端）

Status: done

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

- [x] **Task 1 — accessor 與注入（AC: #1, #2）**
- [x] **Task 2 — migration（AC: #3）**（Rule 15：SELECT/scan 已含 `model_id`，只回填）
- [x] **Task 3 — 測試（AC: #4）**

## Dev Notes

- `WithModelID` 是 sub-5-1 引用的接線先例（`pipeline.go:343`）；改簽名要 grep 所有呼叫點（main.go:697 + tests）。
- 與 sub-6-8a（per-run 模型選擇）的關係：6-8a 落地後 run 的 model 來自請求；本 story 提供「沒指定時的 effective 預設」，6-8a 疊在上面。**先做本 story**。

### Time-dependent visual coverage

- N/A。

### References

- eval-1「產品問題 5」；後續 Backlog P0-5；`apps/api/cmd/api/main.go:678,697`、`apps/api/internal/services/claude_provider_holder.go:56-97`

## Dev Agent Record

### Agent Model Used

Claude Fable 5.1（dev-story，2026-09-04）

### Completion Notes List

- `ai.ClaudeProvider.Model()`：回傳實際送出的 model（override 或 `DefaultClaudeModel`）。`ClaudeProviderHolder.EffectiveModel()`：holder 的 override 或預設，不需 key／ctx——model 是 holder 的屬性不是 key 的屬性。
- pipeline：`modelID` 由字串改為 `func() string`；新 option `WithModelSource(fn)`，`WithModelID(s)` 改為包常數（既有測試零改動）；`runVersion` 經 `currentModelID()` 讀取，未接線時仍回 ""。`main.go` 兩處（`WithModelID(modelID)`、啟動 log）改讀 `claudeHolder.EffectiveModel`，`cfg.GetClaudeModel()` 的 boot 快照移除。
- migration 033 `backfill_subtitle_run_model_id`：`UPDATE subtitle_runs SET model_id='claude-haiku-4-5' WHERE model_id='' AND status='completed'`；failed／skipped 保留 "" 當誠實的 unknown；Down 為 no-op（回填後無法區分）。常數寫死在 migration，不讀 `ai.DefaultClaudeModel`——sub-6-8a 改預設後仍正確。
- 測試：provider `Model()` 預設／override；holder `EffectiveModel` 預設／override 且與 `Get()` 建出的 client 一致；pipeline `RunVersion` 讀 source 即時反映變更、無 option 保持空；migration 033 只回填空的 completed 列、不動 failed 與已標 sonnet 的列、註冊順序。
- ⚠️ story 編號註記：sub-6-10a 與 sub-7-1 的 story 檔原寫 migration 033／034——033 已被本 story 使用，兩檔改為「下一個空號」。
- 🔗 AC Drift: NONE (checked: 'WithModelID|model_id|RunVersion' across _bmad-output/implementation-artifacts/*.md — sub-1-5b/sub-1-6 定義的 RunVersion 四欄與 cache key 語意不變，只是 ModelID 的來源從 env 快照改為 holder；REUSE not DRIFT)
- 📎 Contract Stamps: FOUND (1 across 1 file — `ClaudeProviderHolder` `[@contract-v1]`（sub-2-1a）只加 `EffectiveModel` 一個 additive 方法，簽名與既有行為不變，不 bump；本 story 無 stamp)
- 🎭 A11y Pre-Flight: N/A (100% backend — no apps/web/ files touched)
- 🔌 Route Sync: N/A (no backend route touched)
- 🎨 UX Verification: SKIPPED — no UI changes in this story
- 全回歸：`pnpm nx test api` ✅、`pnpm nx test web` 255 files / 3125 tests ✅、`test:cleanup` 無殘留；`pnpm lint:all`：go vet／staticcheck 過，eslint 0 errors，prettier 唯一紅字為未追蹤本機檔。

### Discovery Triage

- N/A — no out-of-scope work discovered（story 編號註記屬文件修正，已在本 story 內處理）
- CR M4 → AC #4(c) 明文 deferred to sub-6-8a（該 story 讓 model 可變時一併測 holder→pipeline 的傳遞）。

### Change Log

| Date | Change |
| --- | --- |
| 2026-09-04 | Task 1 — `ClaudeProvider.Model()`、`ClaudeProviderHolder.EffectiveModel()`、`WithModelSource` + `currentModelID`、`main.go` 改讀 holder。 |
| 2026-09-04 | Task 2 — migration 033 回填空 model_id 的 completed 列。 |
| 2026-09-04 | Task 3 — 四組測試（ai／services／subtitle／migrations）。 |
| 2026-09-04 | CR fixes — holder `Get`／`TestKey` **永遠**把 `WithClaudeModel(EffectiveModel())` 放最後（H1：opts 裡偷渡的 model 不能再與 run row 不一致）；`currentModelID()` 未接線或回空字串時退回 `ai.DefaultClaudeModel`（H2）；migration 033 doc 記錄 segment cache 無法回填的一次性 miss 成本（H3）；AC #4(c)「holder 重建記新 model」裁定為 **deferred to sub-6-8a**（M4：`h.model` 為 write-once，本 story 無可變更路徑）；AC #4(a)(b) 的端到端斷言已由既有 `process_item_test.go:451`（`final.ModelID == "claude-haiku-4-5"`）涵蓋（M5）；sub-6-8a story 的過期描述更新（M6）；migration 測試改名 `IsRegistered`、移除假 import、補 pending／running／skipped 三態（L7）；`EffectiveModel` 加 write-once／sub-6-8a 需上鎖的註解（L8）。 |

### File List

- `apps/api/internal/ai/claude.go`（modified）+ `claude_test.go`
- `apps/api/internal/services/claude_provider_holder.go`（modified）+ `claude_provider_holder_test.go`
- `apps/api/internal/subtitle/pipeline.go`、`segment_cache.go`（modified）+ `segment_cache_test.go`
- `apps/api/cmd/api/main.go`（modified）
- `apps/api/internal/database/migrations/033_backfill_subtitle_run_model_id.go`（new）+ `_test.go`
- `_bmad-output/implementation-artifacts/sub-6-10a-candidate-identity-backend.md`、`sub-7-1-glossary-scope-tmdb.md`（migration 編號註記）
- `_bmad-output/implementation-artifacts/sprint-status.yaml`（status）

## Senior Developer Review (AI)

**Reviewer:** Claude Opus 5（adversarial CR，換模型慣例；impl by Fable 5.1） · **Date:** 2026-09-04 · **Outcome:** Changes Requested → all items resolved in-session (branch `fix/sub-6-4-5-7-code-review`) → **Approve**

Mandatory checks: Rule 7 PASS（0 codes）· Rule 20 N/A（`ClaudeProviderHolder` v1 additive 不 bump，reviewer 同意）· Rule 25 N/A。

### Action Items

- [x] [H1] `EffectiveModel()` 與 client 實際 model 可能不一致 — `Get`／`TestKey` 一律最後追加 `WithClaudeModel(EffectiveModel())`，單一擁有者。
- [x] [H2] 未接線仍可回 "" — `currentModelID()` 退回 `ai.DefaultClaudeModel`（含 source 回空字串）；測試 `NoModelOptionFallsBackToDefault`。
- [x] [H3] migration 033 與 segment cache 脫鉤未記錄 — migration doc 明寫一次性 cache miss 成本與 TTL。
- [x] [M4] AC #4(c) 不可實作 — 裁定 deferred to sub-6-8a，Change Log／Discovery Triage 記錄。
- [x] [M5] 無端到端 model_id 斷言 — 既有 `process_item_test.go:451` 已涵蓋，Completion Notes 引用。
- [x] [M6] sub-6-8a 描述過期 — 已更新為 `WithModelSource(claudeHolder.EffectiveModel)`。
- [x] [L7] migration 測試名不符、假 import — 改名 `IsRegistered`、移除 `var _ *sql.DB`、補三個狀態反例。
- [x] [L8] 無鎖讀取 — 註解記 write-once 與 sub-6-8a 義務。
