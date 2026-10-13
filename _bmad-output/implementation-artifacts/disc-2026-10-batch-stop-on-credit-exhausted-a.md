# Disc：AI 帳號沒錢時，批次整批停下來（a：後端）

Status: review

**Source:** `disc-2026-09-asr-quota-429-treated-as-transient` 的 Discovery Triage／CR L1。2026-10-13 Alexyu 拍板：「沒錢就整批停」。

這張拆成兩半：

- **a（這張）**：後端。批次碰到「帳號沒錢」就停，剩下的片子標成「已暫停」，事件裡帶上是哪一家、去哪裡儲值。
- **b**：`disc-2026-10-batch-stop-on-credit-exhausted-b`。前端畫面，要等設計稿（F9b、F11-SPEC-LOG ⑤b／⑪）。

⚠️ **a 和 b 要一起部署。** 只部署 a 的話，前端不認得新的批次狀態，帳號沒錢那一次批次視窗會一直顯示「進行中」。

## 查到的事（2026-10-13，Bob，main `4e1834d6`）

- 批次在 `apps/api/internal/services/generation_batch.go` 的 `process`（`:703`）一部一部跑。
  - 每一部失敗都是「記一筆失敗、繼續下一部」（`default` 分支）。
  - 只有預算上限（`ai.ErrBudgetExceeded`）會整批停：`finish(..., GenerationBatchStatusBudgetCeiling, i+1, ...)`。`finish` 把還在排隊或正在跑的片子都改成 `paused`。
- 帳號沒錢的錯誤是 `*ai.CreditExhaustedError`（PR #757）。pipeline 模式的批次走 `cmd/api/generation_batch_runner_adapter.go` → `Pipeline.ProcessItem`，錯誤原樣往上丟，`errors.As` 拿得到。
- 只認 `*ai.CreditExhaustedError`，不認一般的 `ai.ErrAIQuotaExceeded`。Gemini 的 429 和 Claude 的 429 分不出是「打太快」還是「沒錢」，不能因為它們整批停。

## Story

身為自己付 AI 費用的人，
我希望帳號沒錢時，批次馬上整批停，剩下的片子留著等我儲值，
這樣不會看到一整排「失敗」，儲值後按「下次繼續」就能接著做。

## Acceptance Criteria

1. **新的批次狀態** `credit_exhausted`（常數 `GenerationBatchStatusCreditExhausted`）。
2. **碰到就停**：某一部回的錯誤 `errors.As` 得到 `*ai.CreditExhaustedError` 時，整批結束，狀態是 `credit_exhausted`。
   - 這一部和後面排隊的，全部標成 `paused`，跟預算上限一樣。不算失敗，`fail_count` 不加。
   - 判斷放在「批次被取消」之後、預算上限之前。
   - log 一行 `generation batch stopped — AI account has no credit`，帶 provider。
3. **事件帶上去哪裡儲值**：批次進度（`GenerationBatchProgress` 和 SSE `generation_batch_progress`）新增兩個欄位：
   - `credit_provider`：`"openai"` 或 `"anthropic"`。
   - `billing_url`：儲值網址。
   - 只有 `credit_exhausted` 時有值，其他狀態都是空字串。這是新增欄位，舊的欄位不動。
4. **不變的**：一般失敗、正在別處處理、被跳過、預算上限、取消，行為都跟現在一樣。一般的 `ai.ErrAIQuotaExceeded`（不是 `CreditExhaustedError`）照舊算這一部失敗、繼續下一部。
5. **測試**：
   - 第 2 部回 `CreditExhaustedError`（包好幾層）：第 3 部不會被執行；狀態 `credit_exhausted`；第 2、3 部是 `paused`；`fail_count` 0；`credit_provider`／`billing_url` 有值；SSE 結束事件也有。
   - 第 2 部回一般 `ErrAIQuotaExceeded`：第 3 部照樣執行，第 2 部 `failed`。
   - 其他狀態結束時，兩個新欄位是空字串。
6. **檢查**：`go build／vet／test ./...`、`~/go/bin/staticcheck ./...`、`pnpm run lint:all` 全綠。

## Tasks / Subtasks

- [x] T1 `generation_batch.go`：新狀態、`process` 新分支、`finish` 把排隊和正在跑的改成 `paused`、兩個新欄位進 struct 和 SSE（AC #1–#3）
- [x] T2 測試（AC #5）
- [x] T3 檢查（AC #6）

## Dev Notes

- 舊的「legacy 模式」批次（`RouteCGenerationRunner`）走 `TranscriptionService.RunTranscription`，錯誤會不會保留型別沒有驗。正式機用的是 pipeline 模式，這張不管 legacy。
- 不要動前端，那是 b。

### Time-dependent visual coverage

N/A — 純後端。

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（claude-opus-5-5），Amelia dev-story，2026-10-13。

### Debug Log References

- 突變測試：把新分支改成 `case false && errors.As(...)`，`TestGenerationBatch_EmptyAccountStopsTheBatch` 變紅（第 3 部照樣被執行）。還原後全綠。

### Completion Notes List

- `process` 在「批次被取消」之後新增一個分支：`errors.As` 拿到 `*ai.CreditExhaustedError` 就先把 provider 和儲值網址寫進這批的進度，再用 `credit_exhausted` 結束。
- `finish` 對 `credit_exhausted` 的處理跟預算上限一樣：還在排隊或正在跑的片子改成 `paused`。
- `credit_provider`、`billing_url` 加進 `GenerationBatchProgress`（GET 狀態、409、202 都會帶）和 SSE 事件；兩個鍵集合測試同步更新。
- 一般的 `ErrAIQuotaExceeded` 照舊只讓那一部失敗（有測試）。
- 檢查全綠：`go build ./...`、`go vet ./internal/services/`、`go test ./...`、`~/go/bin/staticcheck ./...`、`pnpm run lint:all`（0 errors）、`pnpm run format:check`。

### Senior Developer Review (AI) — 2026-10-13

結論：0 High／0 Medium／1 Low，接受。

- **L1（接受，寫進 b）**：前端還不認得 `credit_exhausted`。只部署 a 時，帳號沒錢那一次批次視窗會停在「進行中」。b 會補上；a 和 b 要一起部署。

### File List

- `apps/api/internal/services/generation_batch.go`
- `apps/api/internal/services/generation_batch_credit_test.go`（新）
- `apps/api/internal/services/generation_batch_test.go`
- `apps/api/internal/services/generation_batch_items_test.go`
- `_bmad-output/implementation-artifacts/disc-2026-10-batch-stop-on-credit-exhausted-a.md`（新）
- `_bmad-output/implementation-artifacts/disc-2026-10-batch-stop-on-credit-exhausted-b.md`（新）
- `_bmad-output/implementation-artifacts/disc-2026-10-batch-stop-on-credit-exhausted-pen-prompts.md`（新）
- `_bmad-output/implementation-artifacts/sprint-status.yaml`
