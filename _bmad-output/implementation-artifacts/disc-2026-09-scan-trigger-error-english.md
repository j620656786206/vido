# Story: disc-2026-09-scan-trigger-error-english — 掃描開不了時說中文，而且真的會說「已在進行中」

Status: review

<!-- SM Bob create-story 2026-09-30。Alexyu：「做下一張」。原單：bugfix-scan-schedule-field-mismatch 的 Discovery Triage。
     行號為 main `4aeb9293`；每條「查到的事」都是 SM 本次親自讀過的位置。 -->

## Story

身為按下「掃描媒體庫」／「立即掃描」的人，
掃描開不了時我要看到中文、看得懂的原因；
已經有掃描在跑時，要被告知「掃描已在進行中」，而不是什麼都沒發生或被說「掃描已啟動」。

## 背景（查到的事）

- 兩個入口失敗時都顯示後端原文：`components/settings/ScannerSettings.tsx:100` `apiErr.message || '掃描觸發失敗'`、`components/library/EmptyReadyForScan.tsx:49` 同樣寫法。後端與網路錯誤（`Failed to fetch`、`Internal server error`…）多為英文。
- 「掃描已在進行中」的分支（`ScannerSettings.tsx:97`，看 `SCANNER_ALREADY_RUNNING`）**到不了**：`handlers/scanner_handler.go:73-97` `TriggerScan` 一律先回 202，才在 goroutine 裡 `StartScan`；第二次按下時 `StartScan` 回 `SCANNER_ALREADY_RUNNING`（`services/scanner_service.go:261-266`）只寫進 log。結果：設定頁什麼都不說，空片庫畫面還會說「掃描已啟動」。
- 7-1 CR 刻意讓 handler 不先查 `IsScanActive`，以 `StartScan` 的鎖為唯一閘門（`scanner_handler_test.go:99` 註解）。先查一次只是為了**回饋**：鎖仍是閘門，兩個請求同時到時的少數情況維持現狀（第二個靜默不做）。

## Acceptance Criteria

1. 後端：送出掃描時若已有掃描在跑，回 409 `SCANNER_ALREADY_RUNNING`「掃描已在進行中」，不再回 202。其他情況不變。
2. 前端（兩個入口）：
   - `SCANNER_ALREADY_RUNNING` →「掃描已在進行中」（警告色）。
   - 其他失敗 → 伺服器給的是中文原因就照樣顯示（沿用 bugfix-10-5 AC #8 spec d），英文或沒有訊息則顯示「掃描沒有開始，請再試一次。」。
3. 測試：Go — 掃描中送出回 409 且不再呼叫 `StartScan`；前端 — 兩個入口的兩種失敗訊息。先紅後綠。

## Tasks / Subtasks

- [x] Task 1 — 後端 409＋測試（AC #1, #3）
- [x] Task 2 — 兩個入口的中文訊息＋測試（AC #2, #3）
- [x] Task 3 — 全量測試、lint、typecheck

## Dev Notes

- 空片庫畫面目前只有 success／error 兩種提示色；「已在進行中」用 error 以外的既有樣式即可（沿用 success 樣式或新增 warning 需看既有 class，不新增設計）。

### Time-dependent visual coverage

- N/A — no visual change.

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（Amelia）

### Completion Notes List

- RED → GREEN：Go `TriggerScan_AlreadyRunningIs409`（且不呼叫 `StartScan`）；前端兩個入口的英文換中文、中文原樣、「已在進行中」為警告樣式。
- 兼顧既有 AC：bugfix-10-5 AC #8 spec d 要求顯示伺服器訊息——改為「中文就原樣、英文就換成中文重試句」，與設定精靈（#611）同一規則。
- 對抗式 review（subagent）：無問題；其他呼叫者（server 端掃描、e2e、TestSprite）不受 409 影響。補 2 條測試（警告樣式、設定頁中文原樣）。
- 全量：`pnpm nx test web` 綠；`pnpm nx test api` 綠；`lint:all`、`web:typecheck` 綠。
- 🔗 AC Drift: NONE。📎 Contract Stamps: NONE（沿用既有 `SCANNER_ALREADY_RUNNING` 碼）。
- 🎭 A11y Pre-Flight: PASS（沿用 `role=alert` 提示）。
- 🎨 UX Verification: SKIPPED — 只改文字與既有警告色。

### Discovery Triage

### File List

- apps/api/internal/handlers/scanner_handler.go、scanner_handler_test.go
- apps/web/src/components/settings/ScannerSettings.tsx、ScannerSettings.spec.tsx
- apps/web/src/components/library/EmptyReadyForScan.tsx、EmptyReadyForScan.spec.tsx
- _bmad-output/implementation-artifacts/disc-2026-09-scan-trigger-error-english.md、sprint-status.yaml

## Change Log

| Date       | Change                  |
| ---------- | ----------------------- |
| 2026-09-30 | create-story（SM Bob）。 |
| 2026-09-30 | dev-story（Amelia）＋對抗式 review → review。 |
