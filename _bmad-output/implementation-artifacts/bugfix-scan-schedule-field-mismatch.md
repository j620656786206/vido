# Story: bugfix-scan-schedule-field-mismatch — 媒體庫的「掃描排程」終於存得起來

Status: review

<!-- SM Bob create-story 2026-09-29。Alexyu：「做掃描排程」。TestSprite 九月衝刺（PR #597）抓到、本機 Playwright 實測確認。
     行號為 main `5a42738c`；每條「查到的事」都是 SM 本次親自讀過的位置。 -->

## Story

身為想讓 Vido 每天自動掃描新片的人，
我要在「設定 → 媒體庫掃描」選「每天」之後，它真的存起來、重新整理後還是「每天」，
而且失敗時用中文告訴我，而不是一行英文。

## 背景（查到的事）

- 前端 `apps/web/src/services/scannerService.ts`：`ScheduleConfig { frequency }`（`:33-35`），`getSchedule` 直接回傳 API 資料（`:103-105`），`updateSchedule` 送 `{ frequency }`（`:107-112`）。
- 後端 `apps/api/internal/handlers/scanner_handler.go`：`scheduleRequest { Interval string \`json:"interval" binding:"required"\` }`（`:106-109`）；GET 回 `{"interval": …}`（`:120`）；PUT 缺 `interval` → 400 `SCANNER_SCHEDULE_INVALID`「Request body must contain an 'interval' field」（`:131-134`）；成功回 `{"interval": …}`（`:149`）。值只收 `manual`／`hourly`／`daily`（`services/scan_scheduler.go:21-31`），存進 settings `scan_schedule`（`:129-133`）。
- 結果（本機 :8090 Playwright 實測，PR #597 報告）：選「每天」→ `PUT {"frequency":"daily"}` 回 400；畫面讀 `schedule?.frequency`（`ScannerSettings.tsx:145`）得 undefined → 永遠顯示「僅手動」。自 Story 7-2（後端）／7-3（前端）於 2026-03-23 各自實作起就不通。
- 錯誤提示 `ScannerSettings.tsx:85-92` 用 `apiErr.message || '排程更新失敗'`——後端訊息是英文，所以使用者看到英文。
- 兩邊測試各自綠：前端 `scannerService.spec.ts:75-95`、`useScanner.spec.ts:28,33,73,118`、`ScannerSettings.spec.tsx:63,146-147` 都 mock 成 `frequency`；E2E `tests/e2e/scan-progress.spec.ts:182-184` 也把 schedule stub 成 `{ frequency: 'manual' }`；後端 `scanner_handler_test.go:220-242` 用 `interval`。**沒有任何測試讓前端碰到真的後端回應。**
- 畫廊 `routes/test/-gallery.fixtures.tsx:4584` 用 `{ frequency: 'hourly' } satisfies ScheduleConfig`（掃描設定夾具）。`:4180`、`:4213` 的 `frequency` 是**備份**排程，不相干。

## Acceptance Criteria

1. **前端改用後端的欄位名 `interval`**（後端契約不動）：`ScheduleConfig { interval: ScheduleInterval }`；`updateSchedule` 送 `{ interval }`；型別 `ScheduleFrequency` 改名 `ScheduleInterval`，所有使用處（`ScannerSettings`、`useScanner`、各 spec、畫廊 `:4584`、E2E stub）一起改。
2. 選「每天」→ 存成功、選單顯示「每天」、重新整理後仍是「每天」。
3. 失敗時提示改成中文「排程沒有存成功，請再試一次。」（不再顯示後端英文訊息）。
4. **真實形狀的測試**（retro-dsr-AI1：碰外部 API 至少一條用真資料結構跑的測試）：新增 E2E `tests/e2e/scan-schedule.spec.ts`，**不 stub** `/scanner/schedule`（只 stub `/setup/status` 讓頁面進得去），選「每天」→ 重新整理 → 仍是「每天」，最後改回「僅手動」。
5. 單元測試：service 送 `{"interval":"daily"}`、讀 `interval`；元件失敗時顯示中文。先紅後綠。

## Tasks / Subtasks

- [x] Task 1 — service／hook／元件改 `interval`＋中文錯誤（AC #1–#3）
- [x] Task 2 — 單元測試改寫＋新增（AC #5）
- [x] Task 3 — E2E 真後端測試（AC #4）；畫廊夾具與 E2E stub 更新
- [x] Task 4 — `pnpm nx test web`、`pnpm nx test api`、`lint:all`、typecheck 全綠

## Dev Notes

- 不改後端：`interval` 是既有契約，後端測試與 `manual/full-app-audit.spec.ts:283` 都依它。
- 本機跑新 E2E：以 `serve-test-env.sh`（:8090）驗證；CI 的 E2E 後端是真的 `bin/api`。
- 掃描觸發失敗的提示（`ScannerSettings.tsx:80`）同樣可能顯示英文——不在本張，記 Discovery Triage。

### Time-dependent visual coverage

- N/A — no wall-clock-reading components touched.

### References

- 發現：`testsprite-2026-09-quota-run`（PR #597）；原始 story `7-2-scheduled-scan-service.md`、`7-3-manual-scan-trigger-ui.md`

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（Amelia）

### Completion Notes List

- RED：單元測試改成真實形狀（`interval`）後 4 條紅；新 E2E 對舊版前端（:8090 舊 dist）跑，`PUT /scanner/schedule` 回 **400**。
- GREEN：`ScheduleConfig { interval }`、`ScheduleInterval`、送 `{ interval }`；錯誤提示改中文「排程沒有存成功，請再試一次。」。重建 dist 後新 E2E 與 `scan-progress.spec.ts` 5／5 過（本機 :8090 真後端），結束時排程已被 afterEach 改回 `manual`。
- 全量：`pnpm nx test web` 291 檔／4,375 條綠；`pnpm nx test api` 綠；`lint:all`、`web:typecheck` 綠。
- 🔗 AC Drift: FOUND — `7-3-manual-scan-trigger-ui` 的前端欄位 `frequency` → 本張改回後端契約 `interval`（`7-2-scheduled-scan-service` 定義）。
- 📎 Contract Stamps: NONE（沿用既有 API，無 stamp）。
- 🎭 A11y Pre-Flight: PASS（只改資料欄位與文案）。
- 🎨 UX Verification: SKIPPED — 畫面無變化（畫廊夾具值仍是「每小時」）。

### Discovery Triage

- ③ — 「掃描媒體庫」觸發失敗時（`ScannerSettings.tsx:76-80`）同樣直接顯示後端英文訊息 → `disc-2026-09-scan-trigger-error-english`（P4）。

### File List

- apps/web/src/services/scannerService.ts、scannerService.spec.ts
- apps/web/src/hooks/useScanner.ts、useScanner.spec.ts
- apps/web/src/components/settings/ScannerSettings.tsx、ScannerSettings.spec.tsx
- apps/web/src/routes/test/-gallery.fixtures.tsx
- tests/e2e/scan-schedule.spec.ts（新）、tests/e2e/scan-progress.spec.ts
- _bmad-output/implementation-artifacts/bugfix-scan-schedule-field-mismatch.md、sprint-status.yaml

## Change Log

| Date       | Change                  |
| ---------- | ----------------------- |
| 2026-09-29 | create-story（SM Bob）。 |
| 2026-09-29 | dev-story（Amelia）：前端改用 `interval`、中文錯誤、真後端 E2E → review。 |
