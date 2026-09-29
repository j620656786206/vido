# Story: bugfix-last-scan-never-shown — 「上次掃描」終於會顯示上次掃了什麼

Status: review

<!-- SM Bob create-story 2026-09-30。Alexyu：「做下一張」。原單：內測實測（2026-08-31）「掃描明明完成過，上次掃描永遠顯示尚未執行過掃描」。
     行號為 main `2bbb677d`；每條「查到的事」都是 SM 本次親自讀過的位置。 -->

## Story

身為管理片庫的人，
我要在「設定 → 媒體庫掃描」看到上次掃描是什麼時候、找到幾個檔案、花了多久，
而且伺服器重開之後還記得，
不要掃完了還寫「尚未執行過掃描」。

## 背景（查到的事）

- **後端從來沒送過「上次掃描」。** `git grep -n "last_scan\|LastScan" -- '*.go'` 沒有任何結果。`GET /scanner/status`（`apps/api/internal/handlers/scanner_handler.go:101-104`）直接回 `ScanProgress`（`services/scanner_service.go:36-48`）：`files_found`、`files_created`…、`is_active`、`started_at`。掃完的結果 `ScanResult`（`:51-62`，由 `buildResult` `:711-727` 產生）只寫進 log（`scanner_handler.go:84-90`），沒有保存。
- **前端讀的欄位後端都沒有。** `apps/web/src/services/scannerService.ts:10-23` 的 `ScanStatus` 宣告 `isScanning`、`filesProcessed`、`estimatedTime`、`lastScanAt`、`lastScanFiles`、`lastScanDuration`，後端一個都沒送（後端是 `is_active`）。所以：
  - `ScannerSettings.tsx:169` 的 `formatLastScan(status.lastScanAt, …)` 永遠拿到 `undefined` → `:30` 回「尚未執行過掃描」。
  - `ScannerSettings.tsx:66` `status?.isScanning` 永遠是 false → 掃描中「掃描媒體庫」鈕不會停用（只靠 `triggerScan.isPending`），`hooks/useScanner.ts:26` 的「掃描中每 3 秒輪詢」也從來沒生效。
  - 和 `bugfix-scan-schedule-field-mismatch`（#598）同一類：7-3 前端照自己的想像寫欄位，各自的測試都 mock 成自己的形狀。
- **設計稿**（Flow E，E1-D `last-scan-section` `Km00R`／E1-M）：「2026-03-22 14:30 · 1,247 檔案 · 耗時 3 分 12 秒」。現在的格式化（`ScannerSettings.tsx:29-40`）用 `toLocaleString('zh-TW')`，會變成「2026/03/22 下午02:30」，而且耗時直接印後端字串；後端的 `Duration` 是 Go 的 `time.Duration.String()`（例如 `1.234567ms`），不能直接給人看。
- 排程掃描、媒體庫設定裡的掃描、手動掃描都走同一個 `ScannerService.StartScan`（`scanner_service.go:157`），所以在那裡記錄就全部涵蓋。
- 設定值的保存方式已有先例：掃描排程存在 settings `scan_schedule`（`services/scan_scheduler.go:14,131,171`）。
- 畫廊夾具 `settings-scanner-settings`（`routes/test/-gallery.fixtures.tsx:4563-4584`）用的是前端想像的形狀，要一起改。

## Acceptance Criteria

1. **後端 `[@contract-v1]`**：`GET /scanner/status` 在原本的欄位之外多回 `last_scan`：`{ completed_at, files_found, duration_ms }`；從來沒完成過掃描時是 `null`。
   - 只記**完成**的掃描（被取消的不算，維持上一次的紀錄）。
   - 存進 settings（`scan_last_result`），伺服器重開後照樣回得出來。存失敗只寫 log，不影響掃描本身。
2. **前端改讀後端真的欄位**：`ScanStatus` 對齊後端（`isActive`、`lastScan`…；刪掉後端沒有的 `filesProcessed`、`estimatedTime`、`lastScanAt`、`lastScanFiles`、`lastScanDuration`）。掃描中的判斷改用 `isActive`（按鈕停用、每 3 秒輪詢恢復作用）。
3. **顯示照設計稿**：「2026-03-22 14:30 · 1,247 檔案 · 耗時 3 分 12 秒」。
   - 耗時：不到 1 秒寫「不到 1 秒」；不到 1 分鐘寫「N 秒」；否則「M 分 N 秒」（整分鐘寫「M 分」；一小時以上寫「H 小時 M 分」）。
   - 沒有紀錄（`null`）：維持「尚未執行過掃描」。
4. **真實形狀的測試**（retro-dsr-AI1）：
   - Go：handler 回應的 JSON 鍵名（含 `last_scan` 三個鍵、`null` 的情況）；service 完成後記錄、取消不記錄、從 settings 讀回。
   - 前端：service 測試用後端真的 snake_case JSON 經過 `fetchApi` 轉換；元件測試三種耗時、`null`、掃描中停用。
   - 本機 :8090 真後端實測：按「掃描媒體庫」→ 重新整理 → 看到剛剛的時間與檔案數。
5. 畫廊夾具改成真的形狀，更新 `-darwin` 基準、刪掉舊 `-linux` 讓 CI 重產；與 E1-D 比對。

## Tasks / Subtasks

- [x] Task 1 — 後端記錄＋保存＋`last_scan`＋測試（AC #1, #4）
- [x] Task 2 — 前端型別對齊、`isActive`、格式化＋測試（AC #2–#4）
- [x] Task 3 — 畫廊夾具與基準；本機真後端實測；全量測試、lint、typecheck（AC #4, #5）

## Dev Notes

- `last_scan` 用自己的小結構，不直接丟 `ScanResult`：畫面只需要三個值，`duration_ms` 是數字才好格式化。
- 讀取：服務啟動後第一次要用時從 settings 載入，之後以記憶體為準；每次完成時先更新記憶體再寫 settings。
- 不處理「小片庫瞬間掃完看不到進度卡」（`bugfix-scan-instant-completion-no-feedback`，P2）——那張是 SSE 時機問題。不過這張做完後，瞬間完成的掃描至少會在「上次掃描」更新成剛剛的時間。
- 不處理掃描觸發失敗的英文訊息（`disc-2026-09-scan-trigger-error-english`）。
- 時區：畫面用瀏覽器當地時間顯示。

### Time-dependent visual coverage

- `ScannerSettings` 顯示的是固定的 `completed_at`（夾具給定），不讀現在時間 → 不需要 clock mock。時區相依：夾具時間在 CI（UTC）與本機顯示可能不同，沿用現有夾具的處理方式。

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（Amelia）

### Completion Notes List

- RED：Go service 4 條（沒掃過、完成後記錄＋重開讀回＋存檔鍵名、取消不覆蓋、存檔失敗仍記得）、handler 2 條（`last_scan` 物件三個鍵、`null`）編譯即紅；前端元件 6 條紅（格式、四種耗時、掃描中停用）。
- GREEN：`LastScanSummary`＋`SetSettingsRepo`／`GetLastScan`／`recordLastScan`（只在完成分支呼叫）；`scanStatusResponse` 內嵌 `ScanProgress`（欄位照舊攤平）＋`last_scan`；前端 `ScanStatus` 對齊後端、`isActive`、E1-D 格式。
- `useScanProgress.ts` 的 `STATUS_UPDATE` 分支從來沒有人 dispatch（`grep` 只有定義），且讀的正是後端沒有的欄位——型別對齊後直接刪掉這段死碼。
- 對抗式 review（subagent）：無 High。修掉 2 條 Med＋2 條 Low：
  - M1 開機後第一次讀取 settings 失敗（請求被中斷、DB busy）會被當成「沒掃過」一直到重開——改成只有讀成功或確定沒有這筆才算讀過，讀取不跟著請求取消，並移到鎖外（Low #4 一併解決）。
  - M2 所有資料夾都找不到（NAS 掛載點掉了）的掃描會把正常紀錄蓋成「0 檔案」——改成不記錄，保留上一次。
  - 存的紀錄是空的（`{}`）時當作沒有紀錄；一小時以上的耗時、數字用 zh-TW 格式。
  - 不修：排程或其他入口觸發的掃描完成後，本頁最多 30 秒才更新（本頁每 30 秒輪詢；要改得動 `useScanProgress`，它沒有 QueryClient）。視覺基準跟機器時區有關——既有狀況，不在本張。
- E2E stub（`scan-progress.spec.ts`、`empty-library.spec.ts`）改成後端真的形狀。
- 本機 :8090 真後端實測：「尚未執行過掃描」→ 按掃描 → 不用重整就變成「2026-09-30 00:18 · 29 檔案 · 耗時 不到 1 秒」；重整一樣；**重開伺服器（保留資料庫）後 `last_scan` 還在**。
- 全量：`pnpm nx test web` 291 檔／4,393 條綠；`pnpm nx test api` 綠；`lint:all`、`web:typecheck` 綠。
- 🔗 AC Drift: FOUND — `7-3-manual-scan-trigger-ui` 前端 `ScanStatus` 欄位（isScanning／lastScan*）→ 對齊後端 `ScanProgress`＋新 `last_scan`。
- 📎 Contract Stamps: `last_scan` on `GET /scanner/status` `[@contract-v1]`。
- 🎭 A11y Pre-Flight: PASS（只改文字內容）。
- 🎨 UX Verification: PASS — 畫廊 `settings-scanner-settings` 與 E1-D `last-scan-section`（`Km00R`）格式一致。

### Discovery Triage

### File List

- apps/api/internal/services/scanner_service.go、scanner_last_scan_test.go（新）
- apps/api/internal/handlers/scanner_handler.go、scanner_handler_test.go
- apps/api/cmd/api/main.go
- apps/web/src/services/scannerService.ts、scannerService.spec.ts
- apps/web/src/hooks/useScanner.ts、useScanner.spec.ts、useScanProgress.ts
- apps/web/src/components/settings/ScannerSettings.tsx、ScannerSettings.spec.tsx
- apps/web/src/routes/test/-gallery.fixtures.tsx
- tests/e2e/scan-progress.spec.ts、empty-library.spec.ts
- tests/visual/…/settings-scanner-settings/*-darwin.png（更新；舊 `-linux` 已刪，CI 重產）
- _bmad-output/implementation-artifacts/bugfix-last-scan-never-shown.md、sprint-status.yaml

## Change Log

| Date       | Change                  |
| ---------- | ----------------------- |
| 2026-09-30 | create-story（SM Bob）。 |
| 2026-09-30 | dev-story（Amelia）→ review。 |
