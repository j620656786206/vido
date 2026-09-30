# Story sub-7-7b: 內建 TMDb 金鑰——設定頁與精靈的文字

Status: review

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

- [x] Task 1 — 型別＋`ApiKeysForm` 標籤／hint／清除文案＋spec（AC #1–#3, #6）
- [x] Task 2 — 精靈 `ApiKeysStep` 依 source 切文案＋spec（AC #4, #6）
- [x] Task 3 — 服務狀態 rate_limited 提示＋spec（AC #5, #6）
- [ ] Task 4 — 設計稿 C7-D／C7-M／N4-D 文案對齊（**交 Alexyu 跑 Pencil Inline AI Agent**，依 `feedback_pen_inline_agent_workflow`；提示詞見下）→ ⌘S → `python3 scripts/export-pen-screenshots.py` → 只 stage `c7-d`／`c7-m`／`n4-d` 三張 → commit
- [x] Task 5 — 全量驗證（web 4,400 綠、typecheck、lint、prettier；visual 本機比對綠）

### Task 4 提示詞（Sally 2026-09-30，節點錨定；貼給 Pencil Inline AI Agent 原樣執行）

> 請只改文字內容，不動任何版面、顏色、字級。
>
> 1. `o6JBg`（C7-D）與 `oZooA`（C7-M）：內容改成「用於中繼資料與海報。儲存後立即生效，無需重啟伺服器。」
> 2. `vb4nH`（C7-D）與 `K2pzyD`（C7-M）：內容改成「使用內建金鑰」（狀態膠囊維持靛青 info 色調，不改樣式）。
> 3. `bvNts`（C7-D）與 `CJsb9`（C7-M）：內容改成「目前使用 Vido 內建的金鑰；如遇速率限制，填入自己的金鑰會立即改用。」
> 4. `gCE7n`（N4-D 精靈第 4 步 TMDb 欄位的 placeholder）：內容改成「可留空，使用內建金鑰」。
>
> 改完請確認 C7-D、C7-M、N4-D 三張沒有任何節點被裁切（`problems` 為空）。

**設計裁定理由**：C7 原本示範「一把 secret、一把 env、一把 none」三種狀態；內建金鑰上線後，新裝機第一眼看到的 TMDb 狀態就是「使用內建金鑰」，比 env 更值得留在稿上；env 的覆蓋提示與 secret 的遮罩列在 Claude／OpenAI 列仍有代表。舊句「儲存後需重啟伺服器才會生效」從來不成立（存的金鑰連重啟都到不了 client），必須從稿上拿掉。

## Dev Notes

- 純前端，不改 API；`GET /settings/keys` 已回 `source`。
- `stateLabel` 的 default 分支保留給未知值。
- 精靈的 `ApiKeysStep` 直接用 `useKeySettings()`（後端沒有 setup gate，`/settings/keys` 在精靈階段可打）；查詢失敗或還在載入＝「不是內建」，維持舊文案，不會對原始碼編譯的安裝亂承諾。
- 服務狀態頁的提示放在卡片下方而不是 `BrokenServicesBanner`：`isBroken()` 刻意把 `rate_limited` 排除在「壞掉」之外，這條規則不動；`serviceLabels.ts` 新增 `rateLimitedHint`（只有 TMDb 有，其他服務被限流不會憑空冒出建議）。文字用中性 `text-secondary`，不再幫赭色膠囊多穿一層顏色。
- 視覺基準：`settings-api-keys-form`（桌機／手機／no-encryption-key）三張因 TMDb hint 換句而重拍 darwin；桌機兩張差異低於門檻、`test:visual:update` 不會重寫，依 /ship 規則刪掉重拍（`update-missing`）。linux 三張已 `git rm`，由 CI bootstrap PR 補。`setup-api-keys-step` 未變（圖庫未種 `/settings/keys`，走非內建文案）。

### Time-dependent visual coverage

- N/A。

## Dev Agent Record

### Agent Model Used

Claude Fable 5.1（Amelia）

### Completion Notes List

- 新測試 9 條：`ApiKeysForm.spec` 4（bundled 標籤與色調、輸入框保留＋內建說明列＋不回遮罩、說明列只在 TMDb、TMDb／其他列清除文案分流）＋改寫 1（TMDb 列改承諾立即生效）；`ApiKeysStep.spec` 3（內建文案、原始碼編譯文案、狀態未知不承諾）；`ServiceStatusDashboard.spec` 2（rate_limited 提示與連結、非 rate_limited／非 TMDb 不出現）。
- 全量：`pnpm nx test web` 288 檔／4,400 條綠；`web:typecheck` 綠；eslint 改到的 8 檔 0 errors；prettier 綠；`pnpm run test:visual` 本機比對綠（3 張 darwin 重拍後）。
- 🔗 AC Drift: NONE。📎 Contract Stamps: `KeySource` 加 `'bundled'`（[@contract-v1→v2] 加值，對應後端 7-7a）。🎭 A11y Pre-Flight: PASS（新增的是 `<p>` 文字與既有 `Link`，無新互動元件）。🎨 UX Verification: PARTIAL — 程式碼文案與 Task 4 提示詞一致；`.pen` 更新待 Alexyu 跑 inline agent 後由 Sally 唯讀 review。

### Discovery Triage

- 無新單。

### File List

- apps/web/src/services/keySettingsService.ts
- apps/web/src/components/settings/ApiKeysForm.tsx、ApiKeysForm.spec.tsx
- apps/web/src/components/settings/ServiceStatusDashboard.tsx、ServiceStatusDashboard.spec.tsx、serviceLabels.ts
- apps/web/src/components/setup/ApiKeysStep.tsx、ApiKeysStep.spec.tsx
- apps/web/src/routes/test/-gallery.fixtures.tsx（註解）
- tests/visual/components.visual.spec.ts-snapshots/components/settings-api-keys-form/**（3 darwin 重拍、3 linux 刪除待 bootstrap）
- _bmad-output/implementation-artifacts/sub-7-7b-bundled-tmdb-key-fe.md、sprint-status.yaml

## Change Log

| Date       | Change                                   |
| ---------- | ---------------------------------------- |
| 2026-09-30 | create-story（SM Bob，自 sub-7-7 拆出）。 |
| 2026-09-30 | dev-story（Amelia）→ review；Task 4（.pen）交 Alexyu 跑 inline agent。 |
