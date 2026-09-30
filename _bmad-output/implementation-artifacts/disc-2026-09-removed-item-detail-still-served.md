# Story: disc-2026-09-removed-item-detail-still-served — 檔案已經不在的電影，舊連結點進去說「找不到」

Status: done

<!-- SM Bob create-story 2026-09-30。Alexyu：「處理這張被移除的片的單子」→ ⚖️ 裁定 A。
     行號為 main `02af614e`；每條「查到的事」都是 SM 本次親自讀過的位置。 -->

## Story

身為從舊連結或書籤點進電影詳情頁的人，
如果這部片的檔案已經不在 NAS 上（掃描標成已移除），
我要看到和其他地方一致的「找不到這部影片」，而不是一個看起來一切正常的頁面。

## 背景（查到的事）

- 「已移除」＝掃描時在 NAS 上找不到檔案（`services/scanner_service.go` `detectRemovedFiles` → `MarkRemoved`）；自 #617 起，資料夾連不到或是空的不會再被誤標，檔案回來會自動恢復。
- 媒體庫、首頁、搜尋都已排除已移除的片（`repository/movie_repository.go` 的 `is_removed = 0` 條件）。
- 但 `GET /movies/:id`（`handlers/movie_handler.go:101-124`）照常回 200，詳情頁把它當一般片顯示（TestSprite 九月衝刺以 `seed-mv-201` 核實）。
- 詳情頁已有「找不到」狀態：API 回 404 時顯示「找不到這部影片 — 這個項目可能已從媒體庫移除，或連結已失效。」＋「返回媒體庫」（TC137 驗證過）。

## Acceptance Criteria

1. ⚖️ Alexyu 2026-09-30 裁定 **A**：`GET /movies/:id` 對已移除的電影回 404（與不存在的 id 相同），詳情頁因此顯示既有的「找不到這部影片」。
2. 資料不刪：只是不顯示；檔案回來時下一次掃描會恢復（#617）。
3. 測試：handler 已移除 → 404（先紅後綠）；本機 :8090 用種子片 `seed-mv-201` 實測頁面。

## Tasks / Subtasks

- [x] Task 1 — handler 已移除回 404＋測試
- [x] Task 2 — 本機實測；全量測試、lint

## Dev Notes

- 只擋對外的詳情端點；內部服務（掃描、字幕）仍可讀到這列。
- 影集沒有被標移除的流程，不在本張。

### Time-dependent visual coverage

- N/A — reuses the existing not-found state.

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（Amelia）

### Completion Notes List

- RED：`TestMovieHandler_GetByID/removed_(file_gone)_is_a_404` 在舊碼回 200。GREEN：`IsRemoved` → `NotFoundError`。
- 本機 :8090：`GET /api/v1/movies/seed-mv-201` → 404、`seed-mv-001` → 200；Playwright 開 `/media/movie/seed-mv-201` 顯示「找不到這部影片 這個項目可能已從媒體庫移除，或連結已失效。 返回媒體庫」。
- 自我對抗檢查（變更 10 行）：只影響 `GET /movies/:id`；子路由（豆瓣評分等）只在詳情頁載入成功後才呼叫；其他連到詳情頁的入口（活動、下載匯入狀態）會落在同一個「找不到」，與媒體庫一致。
- 全量：`pnpm nx test api` 綠；`lint:all` 綠。
- 🔗 AC Drift: NONE。📎 Contract Stamps: NONE（404 語意沿用 dsr-2 AC #8）。
- 🎭 A11y Pre-Flight: N/A。🎨 UX Verification: PASS — 沿用既有「找不到」狀態。

### Discovery Triage

- 無。

### File List

- apps/api/internal/handlers/movie_handler.go、movie_handler_test.go
- _bmad-output/implementation-artifacts/disc-2026-09-removed-item-detail-still-served.md、sprint-status.yaml

## Change Log

| Date       | Change                                                        |
| ---------- | ------------------------------------------------------------- |
| 2026-09-30 | create-story（SM Bob）；⚖️ 裁定 A；dev-story（Amelia）→ review。 |
| 2026-09-30 | PR #619 合併 → done。 |
