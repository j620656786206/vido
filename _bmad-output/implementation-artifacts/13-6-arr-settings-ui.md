# Story 13.6: 連線設定 — Sonarr／Radarr 連線卡片

Status: review

## Story

As a self-hoster who already runs Sonarr and Radarr next to Vido,
I want to type their address and API key into Vido's settings, test it, and pick the quality profile and root folder,
so that Vido can reach them — for requests today, and for showing whether a finished download made it into the library next.

## Context

Epic 13（請求系統）的設定畫面缺口：13-4a／13-4b 早就做好後端（`/api/v1/settings/{sonarr|radarr}` 的 GET／PUT／test／品質設定檔／根資料夾，設定存 settings 表，金鑰加密存 secrets，60 秒健康檢查），但前端沒有畫面，只能用 curl 設定。

這張單在 2026-09-15 被拉到最前面：Alexyu 裁定下載頁要做「**入庫狀態**」，而且判斷方式**走 Sonarr/Radarr 的匯入紀錄**（見 `disc-2026-09-downloads-v2-no-parse-status`）。沒有這個設定畫面，使用者根本接不上 Sonarr/Radarr，後面兩張（`dl-import-1`、`dl-import-2`）就沒有意義。

查證（2026-09-15，唯讀）：線上 Vido 目前**沒有**任何 Sonarr/Radarr 設定；NAS 上跑的是 Sonarr 4.0.19、Radarr 6.3，`history?downloadId=` 可用，series／movie 都有 `tmdbId`。

## Acceptance Criteria

1. **位置**：連線設定頁（`/settings/connection`）在 qBittorrent 卡片下方依序放 Sonarr、Radarr 兩張卡；三張卡都有標題（qBittorrent／下載器、Sonarr／影集的搜尋與匯入、Radarr／電影的搜尋與匯入）。頁面副標改成「設定 Vido 連到 qBittorrent、Sonarr 與 Radarr 的方式。」。
2. **欄位**：啟用開關、網址、API 金鑰（只寫不讀）、品質設定檔、根資料夾、測試連線、儲存設定。沒設定過的卡片開關預設打開。
3. **狀態徽章說真話**：沒填網址或金鑰→「未設定」（中性）；關閉→「已停用」（中性）；健康→「已連線」（青碧）；連不上→「連不上」（硃砂）；還沒檢查過→「尚未檢查」（中性）。
4. **清單**：品質設定檔與根資料夾來自 Sonarr/Radarr 本身，只在**已儲存且啟用**的連線上抓；在那之前下拉停用並寫「啟用並儲存連線後可以選擇」；抓失敗寫「讀不到 X 的清單」；存著的值在伺服器上已不存在時明示「找不到」；已連線但沒選品質設定檔或根資料夾時提醒「請求會停在等待中」；伺服器沒有根資料夾時說去哪裡新增。
5. **金鑰**：GET 永遠不回金鑰；已儲存時欄位留空代表不變更；**換了網址就要重新貼上金鑰**（已存的金鑰不會被送到新網址）；儲存成功後清空欄位。
6. **錯誤說人話**：金鑰錯、連不到、逾時、未填各有一句能照做的話；儲存被伺服器拒絕（409 `DVR_TEST_FAILED`）時同時說「設定沒有儲存」與真正原因（`cause_code`）；Sonarr v3 顯示它自己的「需要 Sonarr v4」。
7. **讀取失敗**：取代表單，不顯示一張「好像沒設定過」的空表單。
8. 設計稿：C4-D／C4-M 補卡片標題與副標；新增 C23-D／C23-M（連線設定往下捲：Sonarr 已連線、Radarr 未設定）。
9. 單元測試、e2e、typecheck、lint、Go 測試、視覺基準全綠。

## Tasks / Subtasks

- [x] **Task 1 — 設計稿（AC #1, #8）**：C4-D／C4-M 卡片標題；C23-D（Qva0y）／C23-M（p37q9）新畫面（C4 直接拉長會蓋到手機列，所以拆成往下捲的第二張）；匯出腳本 SCREENS、CLAUDE.md 流程說明補 c23
- [x] **Task 2 — 前端**：`services/dvrSettings.ts`（含 `DvrSettingsApiError.causeCode`）、`hooks/useDvrSettings.ts`（每個 plugin 一組 query key；設定查詢不在焦點／重連時重抓，免得蓋掉輸入中的內容）、`components/settings/ArrConnectionForm.tsx`、`routes/settings/connection.tsx`
- [x] **Task 3 — 後端（CR H1）**：`handlers/response.go` 的 `APIError` 新增 `cause_code`（omitempty）；`dvr_settings_handler.go` 的 `respondError` 沿 Cause 找到最內層 `PluginError`，頂層 code 不變（409 合約），`cause_code`＋`suggestion` 帶真正原因
- [x] **Task 4 — 同頁 qBittorrent 卡片（CR M4）**：輸入框改 `--bg-tertiary`＋44px（照 C4-D），按鈕 44px，儲存錯誤加 `role="alert"`
- [x] **Task 5 — 測試**：`ArrConnectionForm.spec.tsx` 19 個、`dvr_settings_handler_test.go` 新增 2 個、e2e `tests/e2e/arr-settings.spec.ts` 2 個（有狀態的 stub：儲存後讀回、清單解鎖、提醒出現）；圖庫夾具 2 個
- [x] **Task 6 — 驗證**：`pnpm nx test web` 258 檔全綠、Go handlers／services／plugins 全綠、typecheck、lint 0 errors；e2e chromium 2/2；視覺：新夾具 2 張 `-darwin`，`settings-qbittorrent-form` 3 張重生 `-darwin` 並 `git rm` 3 張 `-linux`；設計稿匯出 191/191、token 一致
- [x] **Task 7 — 對抗式 CR（乾淨 context agent）**：1H／3M／10L，見 Completion Notes

## Dev Notes

### 不要做的事

- **不改後端「留空沿用已存金鑰」的行為**：13-4a 的測試把「換網址不重貼金鑰」寫成預期（[@contract-v1] AC #4），改它要走 Rule 20。這張只在前端擋，後端另立 `disc-2026-09-arr-stored-key-follows-new-url`。
- **不做入庫狀態本身**：那是 `dl-import-1`（後端）與 `dl-import-2`（前端）。
- **不處理 ENCRYPTION_KEY 換掉導致金鑰解不開**：qBittorrent 卡片有專門分支，這裡目前只會顯示「連不上」與通用錯誤（CR L8，記錄不修）。

### 固定詞彙檢查

| 顏色 | 用在哪 | 為什麼對 |
| --- | --- | --- |
| 青碧 | 已連線徽章、測試成功 | 檢查有答案了 |
| 硃砂 | 連不上徽章、測試失敗、讀取失敗 | 壞了 |
| 赭 | 儲存被拒 | 你要求儲存，但沒發生 |
| 中性 | 未設定／已停用／尚未檢查；「還沒選設定檔」「設定檔找不到了」提醒 | 不是故障；事前提醒不給語意色（DESIGN.md §赭 2026-09-11） |
| 泥金 | 開關打開、儲存按鈕 | 主要動作 |

### References

- 13-4a／13-4b 故事檔（AC #4 設定端點合約）
- `apps/api/internal/plugins/manager.go`（健康狀態、`GetClient` 需要啟用＋網址＋金鑰）
- `apps/api/internal/services/dvr_settings_service.go`（測試後才儲存、留空沿用金鑰）
- `ux-design.pen` C4-D（6UCtX）、C4-M（2H4OM）、C23-D（Qva0y）、C23-M（p37q9）

## Dev Agent Record

### Agent Model Used

Claude Opus 5 (1M context)

### Debug Log References

- 設計稿：C4-D 直接拉長到 2067px 會壓到 Flow C 手機列，改成新增 C23-D 放在桌機列最後；C23-M 放在手機列最後。
- 手機稿高度用子節點相對座標算，第一次少算 settings-content 的下邊距造成裁切警告，改用實際量測後定在 1540。
- 本機視覺全套仍有 `retry-retry-notifications`、`glossary-panel-v2/seeded` 等既有不符（本機資料差異），與本單無關。

### Completion Notes List

對抗式 CR 結果與處理：

- **H1 測試失敗／儲存被拒時文案繞圈、Sonarr v3 看不到原因** → 後端 `cause_code`；前端 `describeDvrError(error, name, 'test' | 'save')`。附前後端測試。
- **M2 已連線但沒選品質設定檔或根資料夾，請求會永遠卡住** → 卡片提醒（中性，依 2026-09-11 裁定）。附測試＋e2e。
- **M3 存著的設定檔 id 不在清單裡，畫面顯示「請選擇」卻送出舊值** → 加「找不到這個設定檔（#7）」選項與提醒。附測試。
- **M4 qBittorrent 卡片輸入框與卡片同色** → 照 C4-D 改。視覺基準重生。
- **L 已處理**：改任何欄位都收回「設定已儲存」；測試／儲存進行中整組停用；網址欄改 `type="text"`（避免瀏覽器擋 `192.168.1.100:8989`）；金鑰欄 `autoComplete="off"`；拿掉英文 Go 錯誤字串的 tooltip；沒有根資料夾時說去哪新增；儲存被拒改赭；首次設定開關預設打開（設計稿同步）。
- **L 記錄不修**：ENCRYPTION_KEY 解不開的專門訊息（L8）；徽章不會自己刷新（要重新整理頁面）。
- **範圍外**：後端金鑰會跟著新網址走 → `disc-2026-09-arr-stored-key-follows-new-url`。

### File List

- `apps/web/src/services/dvrSettings.ts`（新）
- `apps/web/src/hooks/useDvrSettings.ts`（新）
- `apps/web/src/components/settings/ArrConnectionForm.tsx`（新）＋`.spec.tsx`（新）
- `apps/web/src/components/settings/QBittorrentForm.tsx`
- `apps/web/src/routes/settings/connection.tsx`
- `apps/web/src/routes/test/-gallery.fixtures.tsx`
- `apps/api/internal/handlers/response.go`、`dvr_settings_handler.go`、`dvr_settings_handler_test.go`
- `tests/e2e/arr-settings.spec.ts`（新）
- `tests/visual/components.visual.spec.ts-snapshots/components/settings-arr-connection-form/`（新 2 張 `-darwin`）、`settings-qbittorrent-form/`（3 張 `-darwin` 重生、3 張 `-linux` 刪除）
- `ux-design.pen`、`_bmad-output/pen-tokens.json`、`_bmad-output/screenshots/flow-c-search-settings/`（c4-d、c4-m、c23-d、c23-m）
- `scripts/export-pen-screenshots.py`、`CLAUDE.md`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`、本檔

## Change Log

| 日期 | 內容 |
| --- | --- |
| 2026-09-15 | 建立並實作（dev＋乾淨 context agent 對抗式 CR：1H／3M 全修，L 大多處理，後端金鑰問題另立）。狀態 → review。 |
