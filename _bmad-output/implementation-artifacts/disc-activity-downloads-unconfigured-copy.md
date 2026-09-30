# Story: disc-activity-downloads-unconfigured-copy — 沒設定 qBittorrent 時，活動中心不再說「無法載入」

Status: review

<!-- SM Bob create-story 2026-09-30。Alexyu：「做下一個」→ 裁定 A（整區不顯示）。原單：TestSprite B4（TC103）＋本機重現（2026-08-31）。
     行號為 main `18a8af54`；每條「查到的事」都是 SM 本次親自讀過的位置。 -->

## Story

身為還沒接 qBittorrent 的人，
我打開活動中心時，
不要看到「下載：無法載入，請稍後再試」——那不是載入失敗，重試也沒用。

## 背景（查到的事）

- 後端 `apps/api/internal/services/activity_service.go:194-205` `downloadsSection`：`GetDownloadCounts` 任何錯誤都回 `status: "unavailable"`。沒設定時的錯誤是 `qbittorrent.ConnectionError{Code: QBITTORRENT_NOT_CONFIGURED}`（`services/download_service.go:107-112`）。
- 前端 `components/activity/ActivityHub.tsx:204-209`：`unavailable` → `ActivitySectionError`「無法載入，請稍後再試」＋重試（`ActivityStates.tsx:85`）。`:211` 已有「沒有任何下載 → 整區不顯示」的規則。
- 全空判斷 `isEmpty`（`ActivityHub.tsx:68-79`）要求 `downloads.status === 'ok'`，所以沒設定時也永遠到不了 K3 空狀態。
- 設計稿：Flow K（`uLr8t`）沒有「未設定」稿；⚖️ Alexyu 2026-09-30 裁定 **A：整區不顯示**（和「沒有下載就不顯示」同一規則；提醒設定交給下載頁 d11 與服務狀態頁）。

## Acceptance Criteria

1. **後端 `[@contract-v1]`**：`GET /activity` 的 `downloads.status` 在 qBittorrent 未設定時為 `"not_configured"`（新值，additive）；其他失敗維持 `"unavailable"`＋錯誤訊息。
2. **前端**：`not_configured` → 下載區整區不顯示（不顯示錯誤、不顯示重試）。
3. 全空判斷把 `not_configured` 當成「沒有下載」：其他區塊也都空時，顯示既有 K3 空狀態。
4. 測試：Go — 未設定回 `not_configured`、連不上仍是 `unavailable`；前端 — 不顯示下載區、全空時顯示空狀態、`unavailable` 行為不變。先紅後綠。

## Tasks / Subtasks

- [x] Task 1 — 後端 `not_configured`＋測試（AC #1, #4）
- [x] Task 2 — 前端型別、隱藏、全空判斷＋測試（AC #2–#4）
- [x] Task 3 — 全量測試、lint、typecheck

## Dev Notes

- 用 `errors.As` 找 `*qbittorrent.ConnectionError` 且 `Code == ErrCodeNotConfigured`，不比對字串。
- 其他畫面（首頁服務狀態等）怎麼呈現「未設定」不在本張。

### Time-dependent visual coverage

- N/A — no visual change beyond hiding an existing section.

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（Amelia）

### Completion Notes List

- RED → GREEN：Go 2 條（未設定 → `not_configured` 且無錯誤訊息；連線失敗仍 `unavailable`）；前端 3 條（隱藏、算作全空、`unavailable` 照舊）。
- 本機 :8090（沒設 qBittorrent）直接打 `GET /activity`：`downloads.status = "not_configured"`。
- 對抗式 review（subagent）：無 High。處理：
  - Med：TestSprite TC101／TC103 的產生腳本寫死了舊行為（斷言有「下載」標題與「重試」鈕）——改成斷言沒有重試鈕、下載區隱藏；測試計畫本身（TC103「沒有錯誤狀態」）與新行為一致，不用改。
  - Low：`QBittorrentService.GetConfig` 吞掉讀取 host 的錯誤，資料庫出錯會被當成「沒設定」而把下載區藏起來——改成只有「找不到這筆設定」才算沒設定，其他讀取錯誤往上傳（仍顯示「無法載入」）。新增 2 條測試。
  - Low：「隱藏」那條測試原本 `total: 0`，舊規則也會隱藏——改成 `total: 5`。
- 全量：`pnpm nx test web` 綠；`pnpm nx test api` 綠；`lint:all`、`web:typecheck` 綠。
- 🔗 AC Drift: NONE。📎 Contract Stamps: `downloads.status = "not_configured"` on `GET /activity` `[@contract-v1]`。
- 🎭 A11y Pre-Flight: PASS（只是少顯示一區）。
- 🎨 UX Verification: PASS — ⚖️ 裁定 A，無新畫面；全空時沿用 K3。

### Discovery Triage

### File List

- apps/api/internal/services/activity_service.go、activity_service_test.go
- apps/api/internal/services/qbittorrent_service.go、qbittorrent_service_test.go
- apps/web/src/services/activityService.ts
- apps/web/src/components/activity/ActivityHub.tsx、ActivityHub.spec.tsx
- testsprite_tests/TC101_Activity_hub_loads_at_activity.py、TC103_Activity_renders_cleanly_when_there_are_no_recent_events.py
- _bmad-output/implementation-artifacts/disc-activity-downloads-unconfigured-copy.md、sprint-status.yaml

## Change Log

| Date       | Change                                   |
| ---------- | ---------------------------------------- |
| 2026-09-30 | create-story（SM Bob）；Alexyu 裁定 A。 |
| 2026-09-30 | dev-story（Amelia）＋對抗式 review → review。 |
