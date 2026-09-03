# Story 6.6: `POST /settings/keys/test` 以傳輸成功判定金鑰有效 —— Sonnet 下不再誤報 `AI_INVALID_RESPONSE`（後端）

Status: ready-for-dev

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

- [ ] **Task 1 — `Ping`（AC: #1, #2）**
- [ ] **Task 2 — handler／Swagger／FE 顯示（AC: #3）**
- [ ] **Task 3 — 測試（AC: #4）**

## Dev Notes

- `TestKey` 的 throwaway provider 仍要走同一 Governor（`claude_provider_holder.go:136-140` 註解）——`Ping` 走 `send` 就自然繼承。
- 與 sub-6-5 共用 `Model()` accessor；若 6-5 未先落地，本 story 自行加 accessor 並在 6-5 合併時去重。

### Time-dependent visual coverage

- N/A（若動 `ApiKeysForm` 只是文字）。

### References

- eval-1「產品問題 6」；後續 Backlog P0-6；`apps/api/internal/handlers/key_settings_handler.go:134-175`、`apps/api/internal/ai/claude.go:363,425-442`

## Dev Agent Record

### Agent Model Used

### Completion Notes List

### Discovery Triage

- （dev 填）

### File List
