# Story 7.7: 內嵌預設 TMDb key，設定頁保留自填（後端為主）

Status: ready-for-dev

## Story

As someone installing Vido for the first time,
I want metadata to work without applying for a TMDb developer account,
so that the only key I must bring is the one that actually costs money.

## Context

party-mode 2026-09-03 查證：TMDb rate limit ≈ 50 req/s 且以 **key + IP** 計，自架使用者 IP 各異互不排擠；條款 non-transferable／personal to you（所有請求算 Alexyu 的使用）、**非商業免費，商業化要另簽**；Jellyfin／Radarr／Sonarr 皆內嵌。**前置：** sub-6-9（attribution 合規）。

## Acceptance Criteria

1. **內嵌方式。** build-time 注入：`-ldflags "-X github.com/vido/api/internal/config.bundledTMDbKey=$TMDB_BUNDLED_KEY"`，CI 從 repo secret 取；**原始碼與 git 歷史不得出現 key**。本機開發無注入 → 行為同今日（要求自填）。
2. **解析優先序。** `KeyResolver`（`key_resolver.go:99` `KeyTMDb`）：settings 表自填 → env `TMDB_API_KEY` → bundled。`source` 回 `bundled` 新值（additive enum；FE `ApiKeysForm` 的 source 顯示「內建」）。
3. **設定頁。** TMDb 列預設顯示「使用內建金鑰 · 如遇限流可改用自己的」，欄位保留；attribution 區（sub-6-9）就在其下。
4. **限流降級。** 收到 429 時 slog + `service_health` 標 `rate_limited`（既有狀態值），UI 側欄點變琥珀（既有 `DOT_SHAPE.rate_limited`）；文案提示改自填 key。
5. **文件。** `docs/deployment*.md`：內建 key 的授權說明（非商業、attribution 已內建）、如何自填；`README*.md` 一句。
6. **測試。** 解析優先序三層；`source=bundled` 回傳；無注入時回 `none`；429 → health 狀態。

## Tasks / Subtasks

- [ ] **Task 1 — ldflags 注入 + resolver 優先序（AC: #1, #2）**
- [ ] **Task 2 — 429 降級（AC: #4）**
- [ ] **Task 3 — 設定頁文案 + source 顯示（AC: #3）**
- [ ] **Task 4 — 文件 + 測試（AC: #5, #6）**

## Dev Notes

- `backlog-tmdb-runtime-key-resolution`（TMDb client 目前 boot 時讀 env、改 key 要重啟）——本 story **順手 PROMOTE**：resolver 三層優先序落地時把 TMDb client 改成透過 holder（鏡射 `ClaudeProviderHolder`），否則「自填 key」仍要重啟，AC #3 的文案會說謊。Rule 24 lane ①，加 AC #7：「自填／清除 key 免重啟生效」。
- 安全：bundled key 只在 `sanitizeAttr` 之下 log；不進 `/health` 回應。

### Time-dependent visual coverage

- N/A。

### References

- party-mode 2026-09-03 TMDb 條款查證（eval-1 backlog P1-7）；`services/key_resolver.go`；`ApiKeysForm.tsx:55-60`

## Dev Agent Record

### Agent Model Used

### Completion Notes List

### Discovery Triage

- （dev 填）

### File List
