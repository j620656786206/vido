# Story 6.6: `POST /settings/keys/test` 以傳輸成功判定金鑰有效 —— Sonnet 下不再誤報 `AI_INVALID_RESPONSE`（後端）

Status: done

## Story

As a BYOK NAS owner,
I want the key test to tell me my key works whenever the provider accepted it,
so that a valid Sonnet key is not reported as broken just because a 1-token reply came back empty.

## Context — 這個 story 為什麼存在

eval-1 產品問題 6：`CLAUDE_MODEL=claude-sonnet-5` 下 `POST /settings/keys/test` 回 `AI_INVALID_RESPONSE: Cannot parse AI response`，key 其實有效。鏈：`ClaudeProviderHolder.TestKey`（`claude_provider_holder.go:141`）→ `CompleteText(ctx, "", "hi", 1)` → `claude.go:436` `text == "" → ErrAIInvalidResponse`。Sonnet 5 在 `max_tokens=1` 下常回空 text block（例如先吐換行或工具前綴），HTTP 200、有 usage，但被當成解析失敗。

## Acceptance Criteria

1. **判定改為傳輸層。** `ClaudeProvider` 新增 `Ping(ctx) error`：送 `max_tokens=1` 的最小請求，**只看 `send` 是否回錯**（401/403 → `ErrAIUnauthorized`、404 → `ErrAIModelNotFound`、逾時／5xx 照 `classifyErr`），**不檢查回文內容**。`TestKey` 改呼叫 `Ping`。`CompleteText` 的空回文語意**不動**（其他呼叫者仍需要）。

2. **Gemini 同級。** `GeminiProvider` 也加 `Ping`；`ai.TextCompleter` 介面若加 `Ping` 需檢查所有實作與 fakes（`grep -rn "TextCompleter"`）。

3. **回應資訊。** 200 回 `{valid:true, model:"<effective model>"}`（additive 欄位，讓設定頁能顯示「已驗證：claude-sonnet-5」）。Swagger 同步。

4. **測試。** (a) fake server 回 200 且 content 為空陣列 → valid；(b) 401 → `AI_UNAUTHORIZED`；(c) 404 → `AI_MODEL_NOT_FOUND`；(d) 逾時 → `AI_TIMEOUT`；(e) handler 回 `model` 欄位；(f) FE `ApiKeysForm` 若顯示 model 需補 spec（可選，FE 半 ≤1 task）。

## Tasks / Subtasks

- [x] **Task 1 — `Ping`（AC: #1, #2）**
- [x] **Task 2 — handler／Swagger／FE 顯示（AC: #3）**
- [x] **Task 3 — 測試（AC: #4）**

## Dev Notes

- `TestKey` 的 throwaway provider 仍要走同一 Governor（`claude_provider_holder.go:136-140` 註解）——`Ping` 走 `send` 就自然繼承。
- 與 sub-6-5 共用 `Model()` accessor；若 6-5 未先落地，本 story 自行加 accessor 並在 6-5 合併時去重。

### Time-dependent visual coverage

- N/A（若動 `ApiKeysForm` 只是文字）。

### References

- eval-1「產品問題 6」；後續 Backlog P0-6；`apps/api/internal/handlers/key_settings_handler.go:134-175`、`apps/api/internal/ai/claude.go:363,425-442`

## Dev Agent Record

### Agent Model Used

Claude Fable 5.1（dev-story，2026-09-04）

### Completion Notes List

- `ai.Pinger` 介面（`provider.go`）；`ClaudeProvider.Ping`：一次 `max_tokens=1` 的 Messages 呼叫走既有 `send`（governed→retry→classifyErr），只看傳輸結果——401/403 → `ErrAIUnauthorized`、404 → `ErrAIModelNotFound`、逾時 → `ErrAITimeout`、2xx 空 content = PASS。`GeminiProvider.Ping`：同級化，`maxOutputTokens=1` 的 generateContent，狀態碼對映（400/401/403 → unauthorized，Gemini 用 400 報壞 key）。
- `ClaudeProviderHolder.TestKey` 改呼叫 `Ping`（resolved 與 candidate 兩路徑；`pingOf` 做 `ai.Pinger` 窄化，非 Pinger 的 completer 退回 CompleteText 路徑）；throwaway provider 仍共用 Governor。
- handler 200 回 additive `model`（tester 實作 `EffectiveModel()` 時才附，測試 fake 不受影響）；Swagger 更新。**FE 顯示（AC #3 可選）本 story 不做**——`ApiKeysForm` 目前只顯示 valid/invalid，`model` 欄位留給 sub-6-8b 的設定頁工作一併消費。
- 🔗 AC Drift: NONE (checked: 'TestKey|keys/test|CompleteText' across _bmad-output/implementation-artifacts/*.md — sub-2-1a AC 的「最小真實呼叫驗證 auth 路徑」語意保留，只是判定由回文改為傳輸；REUSE not DRIFT)
- 📎 Contract Stamps: FOUND (1 across 1 file — `ClaudeProviderHolder` `[@contract-v1]`（sub-2-1a）：`TestKey` 簽名不變、錯誤 sentinel 集合不變，只是空回文不再算失敗；不 bump、Change Log 記)
- 🎭 A11y Pre-Flight: N/A (100% backend — no apps/web/ files touched)
- 🔌 Route Sync: `POST /api/v1/settings/keys/test` verified at `key_settings_handler.go` RegisterRoutes（既有路由，未新增）
- 🎨 UX Verification: SKIPPED — no UI changes in this story
- 全回歸：`pnpm nx test api` ✅、`pnpm nx test web` 255 files / 3125 tests ✅、`test:cleanup` 無殘留；`pnpm lint:all` go vet／staticcheck 過、eslint 0 errors、prettier 唯一紅字為未追蹤本機檔。

### Discovery Triage

- N/A — no out-of-scope work discovered
- CR H4：Gemini `Ping` 目前無呼叫者（key-test 端點只收 `claude`）——不是本 story 範圍外的新工作，是 AC #2 的「同級化」先做零件、接線隨 Gemini key UI（sub-7-8）一起；已記於程式註解與 Change Log。

### Change Log

| Date | Change |
| --- | --- |
| 2026-09-04 | Task 1 — `ai.Pinger`、`ClaudeProvider.Ping`、`GeminiProvider.Ping`（+ `geminiGenerationConfig.MaxOutputTokens`）。 |
| 2026-09-04 | Task 2 — holder `TestKey` → `Ping`；handler 200 additive `model`；Swagger。FE 顯示延至 sub-6-8b。 |
| 2026-09-04 | Task 3 — claude Ping ×4（空 content／401／404／timeout）、gemini Ping ×4、holder ×2、handler model 欄位。 |
| 2026-09-04 | CR fixes — 移除 `pingOf`／`pingViaComplete` 死碼退路（會重新引入空回文失敗），改為 `ai.Pinger` 斷言失敗即回錯（H1）；`var _ Pinger = (*ClaudeProvider)(nil)` / `(*GeminiProvider)(nil)` 編譯期證明；handler 401/403 → `AI_UNAUTHORIZED`、404 → `AI_MODEL_NOT_FOUND`（H2；Rule 7 code-list update only，project-context 清單與 mega-line 已同步，FE 不依賴 code）；Gemini Ping：key 改走 `x-goog-api-key` header 不進 URL（H7）、`max_output_tokens` 統一 snake（H3）、400 只在 body 含 `API_KEY_INVALID`／"API key not valid" 才判 unauthorized（H3），測試斷言 body 與 header；Gemini Ping 未接線明文記錄（H4，待 sub-7-8 Gemini key row）；`ModelReporter` 介面 + holder 編譯期斷言（M5）；Ping 註解改述 governed/budget 事實（M6）；timeout 測試移除全域覆寫、縮短 sleep（L8）；Swagger 引號、sub-6-8b 補消費端交接註記（L9）。 |

### File List

- `apps/api/internal/ai/provider.go`、`claude.go`、`gemini.go`（modified）+ `claude_test.go`、`gemini_test.go`
- `apps/api/internal/services/claude_provider_holder.go`（modified）+ `claude_provider_holder_test.go`
- `apps/api/internal/handlers/key_settings_handler.go`（modified）+ `key_settings_handler_test.go`
- `_bmad-output/implementation-artifacts/sub-6-6-key-test-transport-success.md`（this file）
- `_bmad-output/implementation-artifacts/sprint-status.yaml`（status）

## Senior Developer Review (AI)

**Reviewer:** Claude Opus 5（adversarial CR，換模型慣例；impl by Fable 5.1） · **Date:** 2026-09-04 · **Outcome:** Changes Requested → all items resolved in-session on the same branch → **Approve**

Mandatory checks: Rule 7 PASS（code-list update only，prefix 17 不變）· Rule 20 N/A（holder v1 不 bump，reviewer 同意）· Rule 25 mega-line 新增一條 entry，無合併衝突 · Rule 27 ⑤ 修正（H7）。

### Action Items

- [x] [H1] `pingViaComplete` 死碼退路會重新引入 bug — 刪除，改斷言 `ai.Pinger` 並失敗即回錯；兩個 provider 加編譯期證明。
- [x] [H2] 401／404 都回 `AI_PROVIDER_ERROR` — 改 `AI_UNAUTHORIZED`／`AI_MODEL_NOT_FOUND`，handler 測試同步，project-context Rule 7 清單同步。
- [x] [H3] Gemini 400 一律當壞 key、body 丟棄、casing 混用 — 只在 body 說 `API_KEY_INVALID` 才判；`max_output_tokens` snake；測試斷言 body。
- [x] [H4] Gemini Ping 無呼叫者 — 明文記錄（註解 + Change Log + Discovery Triage），待 sub-7-8。
- [x] [M5] `model` 欄位無編譯期契約 — `ModelReporter` 介面 + `var _ ModelReporter = (*services.ClaudeProviderHolder)(nil)`。
- [x] [M6] 「transport alone」註解不實 — 改述 governed／budget 事實。
- [x] [M7] Gemini key 進 URL 會漏進 log — 改 `x-goog-api-key` header。
- [x] [L8] timeout 測試覆寫全域 + 0.42s — 移除覆寫、sleep 80ms／timeout 20ms。
- [x] [L9] Swagger 引號 + sub-6-8b 缺交接 — 改單引號；sub-6-8b Dev Notes 加「Inherited from sub-6-6」。
