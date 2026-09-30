# Story sub-7-7b: 內建 TMDb 金鑰——設定頁與精靈的文字

Status: ready-for-dev

<!-- SM Bob create-story 2026-09-30，由 sub-7-7 拆出；依賴 sub-7-7a 先合併（後端已回 source="bundled"、429 會標 rate_limited）。
     行號為 main `3e6e354a`。 -->

## Story

身為打開「金鑰設定」的人，
我要看得出 TMDb 現在用的是內建金鑰、填自己的會馬上生效，
被限流時知道該去哪裡填自己的金鑰；精靈也不要再嚇我說「跳過會抓不到資料」。

## 背景（查到的事）

- `services/keySettingsService.ts:20`：`KeySource = 'secret' | 'env' | 'none'`——後端現在會回 `'bundled'`。
- `components/settings/ApiKeysForm.tsx:93-102` `stateLabel`：沒有 bundled 分支，會落到「尚未設定」；`:104-113` `stateToneClass` 同。`:73` TMDb 列 hint「儲存後需重啟伺服器才會生效」——7-7a 之後這句是錯的（Claude／OpenAI 那兩列 `:63,:86` 的「儲存後立即生效」才對）。`:516` 清除確認文案「清除後將改用環境變數的金鑰；若環境變數也未設定，相關功能會停用」對 TMDb 不再成立（會退回內建）。`:573-580` env 覆蓋提示只在 `source === 'env'` 顯示。
- `ApiKeysForm.spec.tsx:138, 150-151` 釘著舊文案。
- `components/setup/ApiKeysStep.tsx:23` TMDb 欄位 hint、`:55` 跳過警語「跳過 API 金鑰設定將會限制部分功能，例如自動取得元資料…」——正式映像檔有內建金鑰時不成立（原始碼編譯時仍成立；可用 `GET /settings/keys` 的 `source` 判斷）。
- `components/settings/serviceLabels.ts:28-35` TMDb 的 `fixHint` 只在 `BrokenServicesBanner`（error／disconnected）顯示，`rate_limited` 被 `isBroken()` 排除（`:67-69`）——AC 的「限流時提示改自填」要另找位置（`ServiceStatusCard.tsx:17-20` 的 warning 樣式旁）。
- 設計稿：C6 金鑰設定（`flow-c-search-settings/`）沒有「內建」狀態；請 Sally 補一列狀態文字即可，不需新畫面。

## Acceptance Criteria

1. `KeySource` 加 `'bundled'`；TMDb 列 source 為 bundled 時顯示「使用內建金鑰 · 如遇限流可改用自己的」（info 色調），欄位保留可填；attribution 區（sub-6-9）位置不變。
2. TMDb 列 hint 改為「儲存後立即生效，無需重啟伺服器」；spec 的舊文案斷言同步。
3. 清除確認文案依 TMDb／其他分開：TMDb 說「清除後將改用內建金鑰（或環境變數）」。
4. 精靈 API 金鑰步驟：後端 `source === 'bundled'` 時，TMDb 欄位 hint 說明已內建、可留空；跳過警語不再點名「自動取得元資料」。原始碼編譯（source none）維持今天的文案。
5. 服務狀態頁 TMDb 卡片為 `rate_limited` 時顯示一行「內建金鑰被限流：到「金鑰設定」填自己的金鑰可立即解除」並連到 `/settings/keys`。
6. 測試：`ApiKeysForm.spec` 三種 source 標籤＋TMDb hint＋清除文案；`ApiKeysStep.spec` bundled／none 兩種文案；`ServiceStatusCard.spec` rate_limited 提示；web 全量、lint、typecheck、prettier 綠；視覺 gallery 若加 fixture 需 `-linux` bootstrap PR。

## Tasks / Subtasks

- [ ] Task 1 — 型別＋`ApiKeysForm` 標籤／hint／清除文案＋spec（AC #1–#3, #6）
- [ ] Task 2 — 精靈 `ApiKeysStep` 依 source 切文案＋spec（AC #4, #6）
- [ ] Task 3 — 服務狀態 rate_limited 提示＋spec（AC #5, #6）
- [ ] Task 4 — Sally 對 C6 補「內建」狀態列（設計稿＋截圖）；全量驗證

## Dev Notes

- 純前端，不改 API；`GET /settings/keys` 已回 `source`。
- `stateLabel` 的 default 分支保留給未知值。

### Time-dependent visual coverage

- N/A。

## Dev Agent Record

### Agent Model Used

### Completion Notes List

### Discovery Triage

### File List

## Change Log

| Date       | Change                                   |
| ---------- | ---------------------------------------- |
| 2026-09-30 | create-story（SM Bob，自 sub-7-7 拆出）。 |
