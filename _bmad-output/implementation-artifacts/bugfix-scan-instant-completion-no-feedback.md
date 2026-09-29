# Story: bugfix-scan-instant-completion-no-feedback — 小片庫一下就掃完，也看得到「掃描完成」

Status: done

<!-- SM Bob create-story 2026-09-30。Alexyu：「做下一張」。原單：TestSprite TC064＋本機 Playwright 重現（2026-08-31）。
     行號為 main `0a0901db`；每條「查到的事」都是 SM 本次親自讀過的位置。 -->

## Story

身為第一次掃描小片庫的人，
我按下「掃描媒體庫」之後，
要看到掃描進度卡和「掃描完成」提示（找到幾個、新增幾個），
而不是什麼都沒發生、以為按鈕壞了。

## 背景（查到的事）

- 順序問題：`ScannerSettings.tsx:78-85` 先 `await triggerScan.mutateAsync()`（POST `/scanner/scan`，後端在 goroutine 開始掃，`scanner_handler.go:73-97` 立刻回 202），**之後**才 `requestScanTracking()` 叫殼層的 `ScanProgress` 打開 SSE（`hooks/useScanProgress.ts:136-146`、`components/scanner/ScanProgress.tsx:47`）。小片庫幾毫秒就掃完，`scan_progress`／`scan_complete` 在 SSE 連上前就廣播完了，卡片和完成提示全部錯過。
- 後端 SSE（`apps/api/internal/sse/handler.go:24-37`）先 `hub.Register()` 才送 `connected` 事件——所以**等到 `connected`（或連線 open）再送 POST**，之後的每個事件都一定收得到。
- SSE 故意「用到才開」（`ScanProgress.tsx:39-45` 註解：保護 E2E 的 networkidle 穩定），這張維持這個做法。
- 第二個入口：空片庫畫面的「掃描媒體庫」（`components/library/EmptyReadyForScan.tsx:30-38`）**完全沒有**叫 `requestScanTracking`——從這裡掃，不論大小片庫都看不到進度卡。朋友第一次用多半就是從這裡按。
- 本機實測（bugfix-last-scan-never-shown，:8090 種子資料 29 個檔案）：掃描耗時 3 毫秒。

## Acceptance Criteria

1. 按「掃描媒體庫」（設定頁與空片庫畫面兩個入口）時，**先**打開掃描 SSE、等到連上，**再**送出掃描請求。
   - 最多等 3 秒；連不上就照常送出（不能因為 SSE 壞掉就不能掃描）。
   - 殼層沒有掛 `ScanProgress`（例如測試環境）時不等。
2. 結果：小片庫瞬間掃完，也會出現既有的「掃描完成」提示（找到／新增／更新…）。
3. 送出失敗（例如「掃描已在進行中」）的既有提示不變；SSE 已經開著也沒關係（正好看得到那個進行中的掃描）。
4. 空片庫畫面既有的「掃描已啟動」提示不變。
5. 測試：
   - 單元：`requestScanTracking` 會等到連上（`connected` 或 open）才完成；逾時也會完成；沒人訂閱時立刻完成；已經連上時立刻完成。兩個入口都「先等追蹤、再送 POST」。先紅後綠。
   - 本機 :8090 真後端實測：小片庫按掃描 → 看到「掃描完成」提示。

## Tasks / Subtasks

- [x] Task 1 — `useScanProgress`：`startTracking`／`requestScanTracking` 回傳「連上了」的 Promise（含 3 秒逾時）＋測試（AC #1, #5）
- [x] Task 2 — 兩個入口改成先等追蹤再送出＋測試（AC #1–#4）
- [x] Task 3 — 本機真後端實測；全量測試、lint、typecheck

## Dev Notes

- 不改畫面（進度卡與完成提示都已有設計稿 E2／E3）。後端原本不打算動；對抗式 review 找到 SSE hub 的註冊是排隊的（見 Completion Notes），一併修。
- 不改「用到才開 SSE」的原則：只是把「開」往前移到 POST 之前。

### Time-dependent visual coverage

- N/A — no visual change.

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（Amelia）

### Completion Notes List

- RED → GREEN：`requestScanTracking` 等到連上（`connected`／open）、3 秒逾時、沒人訂閱與已連上時立刻完成；兩個入口「先等再送」且等待中按鈕停用。
- 本機 :8090 真後端（production build）實測：按「掃描媒體庫」→ 立刻出現「掃描完成 找到 29 檔案 · 更新 17 · 無法匯入 12」。另把 SSE 握手人為延遲 800 ms：`GET /events` 在 +250 ms、`POST /scanner/scan` 在 +1057 ms——確認一定先連上才送出。
- 對抗式 review（subagent）：無 High。修掉 2 條 Med＋1 條 Low：
  - M1（後端）`Hub.Register` 原本把新連線丟進 Run 迴圈的佇列，handler 馬上送 `connected`；和同時到的廣播互搶時，第一個事件可能送不到新連線。改成 `Register` 當場加入名單（hub 已關閉則回傳已關閉的 client）。新測試：`Register` 一回傳就在名單裡（舊碼第 0 輪就失敗）。
  - M2（前端）斷線後排定的「10 秒後重連」會在使用者重新點擊之後觸發，把剛開好的新連線關掉——`connectSSE` 開頭先取消它。新測試在舊碼上失敗。
  - L3 連線失敗時立刻放行等待中的按鈕，不用等滿 3 秒。
  - 不修：瀏覽器還以為連著、其實已斷（NAS 睡眠後）時會直接送出——要等 30 秒 keep-alive 才會發現；罕見。
- 全量：`pnpm nx test web` 291 檔／4,403 條綠；`pnpm nx test api` 綠（含 `internal/sse -race`）；`lint:all`、`web:typecheck` 綠。
- 🔗 AC Drift: NONE。📎 Contract Stamps: NONE（SSE 事件格式未變）。
- 🎭 A11y Pre-Flight: PASS（等待中按鈕以 disabled 呈現，沿用既有「掃描中」樣式）。
- 🎨 UX Verification: SKIPPED — 無畫面變化（沿用 E2／E3 的進度卡與完成提示）。

### Discovery Triage

### File List

- apps/api/internal/sse/hub.go、hub_test.go
- apps/web/src/hooks/useScanProgress.ts、useScanProgress.spec.ts
- apps/web/src/components/settings/ScannerSettings.tsx、ScannerSettings.spec.tsx
- apps/web/src/components/library/EmptyReadyForScan.tsx、EmptyReadyForScan.spec.tsx
- _bmad-output/implementation-artifacts/bugfix-scan-instant-completion-no-feedback.md、sprint-status.yaml

## Change Log

| Date       | Change                  |
| ---------- | ----------------------- |
| 2026-09-30 | create-story（SM Bob）。 |
| 2026-09-30 | dev-story（Amelia）＋對抗式 review → review。 |
| 2026-09-30 | PR #609 合併（CI 空片庫 E2E 讀計數太早，改成等請求被處理）→ done。 |
