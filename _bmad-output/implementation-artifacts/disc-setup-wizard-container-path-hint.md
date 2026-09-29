# Story: disc-setup-wizard-container-path-hint — 精靈填錯資料夾路徑時，告訴你該填哪裡

Status: done

<!-- SM Bob create-story 2026-09-30。Alexyu：「做下一個」。原單：內測實測（2026-08-31，朋友的 Synology）。
     行號為 main `e9bad4e3`；每條「查到的事」都是 SM 本次親自讀過的位置。 -->

## Story

身為第一次在 NAS 上裝 Vido 的人，
我在設定精靈的「媒體庫」那一步填了 NAS 上看到的路徑（例如 `/video/Movies`），
要被清楚告知「Vido 在 Docker 裡看不到這個路徑，要填容器裡的路徑，例如 /media/movies」，
而不是一行英文「library entry 0: path does not exist: /video/Movies」。

## 背景（查到的事）

- 精靈每按一次「下一步」都會呼叫 `validateStep`（`apps/web/src/components/setup/SetupWizard.tsx:68-93`），失敗時把後端訊息原樣放進紅框 `setup-error`（`:86-87`、`:170-176`）。
- 後端 `services/setup_service.go:239-278` `validateMediaFolderStep`：`os.Stat(path)` 失敗 → `library entry %d: path does not exist: %s`（`:251-254`）；還有 `path is required`、`path is not a directory`、`content_type must be…`，全是英文，而且用 0 起算的「library entry 0」。handler 原樣回給前端（`handlers/setup_handler.go:89-96` `SETUP_VALIDATION_FAILED`）。
- 同一個檔案其他步驟也是英文：`language is required`（`:221`）、`invalid qBittorrent URL`（`:234`）、`invalid TMDb API key format`（`:287`）。
- 容器裡能看到的媒體根目錄是 `VIDO_MEDIA_DIRS`（`config/config.go:167`，預設 `/media`）；`docker-compose.yml:40` 把 `${MEDIA_PATH}` 掛到 `/media:ro`。精靈輸入框的 placeholder 已是 `/media/movies`（`MediaLibrarySetupStep.tsx:92`）。
- `SetupService` 目前拿不到媒體根目錄（`setup_service.go:37-42`，`main.go:179`）。
- 設計稿 Flow N（`dWaWT`）沒有精靈的錯誤狀態稿；錯誤紅框是既有元件，本張只改框裡的文字。

## Acceptance Criteria

1. 媒體庫那一步，路徑不存在時（zh-TW）：
   「找不到「/video/Movies」。Vido 在 Docker 裡，只看得到掛進容器的資料夾——請填容器裡的路徑，例如 /media/movies、/media/tv。」
   - 例子取自容器裡真的存在的資料夾：先列媒體根目錄（`VIDO_MEDIA_DIRS`，照設定順序——它可能本身就是電影／影集資料夾），再列根目錄底下的子資料夾（照字母排序），合計最多 4 個。預設 Docker 會是「/media、/media/movies、/media/tv」。
   - 一個都找不到時：「找不到「…」。Vido 在 Docker 裡，只看得到掛進容器的資料夾，但目前容器裡沒有 /media——請在 docker-compose 的 volumes 把媒體資料夾掛到 /media。」（根目錄名稱照實際設定）
2. 同一步其他訊息改 zh-TW：沒填路徑「還有資料夾沒填路徑。」、是檔案「「…」是檔案，不是資料夾。」、類型錯「資料夾的類型只能是電影或影集。」
3. 其他步驟的驗證訊息改 zh-TW：語言「請選擇語言。」、qBittorrent「qBittorrent 網址看起來不對，請填完整網址，例如 http://192.168.1.10:8080。」、TMDb「TMDb 金鑰格式不對，請確認有完整貼上。」
4. 前端兩個英文後備字串改 zh-TW：「這一步沒有通過檢查，請再試一次。」、「設定沒有完成，請再試一次。」
5. 測試：Go — 不存在時列出真的子資料夾（排序、最多 4）、只有根目錄、什麼都沒有三種；其他訊息改成中文。前端 — 後備字串。先紅後綠。

## Tasks / Subtasks

- [x] Task 1 — `SetupService` 取得媒體根目錄＋路徑提示＋中文訊息＋測試（AC #1–#3, #5）
- [x] Task 2 — 前端後備字串＋測試（AC #4, #5）
- [x] Task 3 — 全量測試、lint、typecheck；本機 :8090 精靈實測

## Dev Notes

- 只列根目錄**直接**底下的資料夾，不往下遞迴（NAS 上可能很大）；隱藏資料夾（`.` 開頭、`@eaDir` 這類 Synology 系統資料夾）不列。
- 不改前端版面，不需要設計稿（訊息進既有紅框）。
- 設定頁「媒體庫管理」新增路徑時的處理（存下並標「找不到」）不在本張。

### Time-dependent visual coverage

- N/A — no visual change.

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（Amelia）

### Completion Notes List

- RED → GREEN：Go 新增 7 條（列出容器裡的資料夾、只有根目錄、多個根目錄排前面、去重、什麼都沒掛、沒有權限、同一步其他中文訊息）＋既有測試的期望字串改中文；前端 3 條（沒訊息的後備、完成設定的英文換中文、伺服器寫好的中文原樣顯示）。
- 本機 :8090 直接打 `POST /setup/validate-step`：`/video/Movies` → 「找不到「/video/Movies」。…例如 …/media/movies、…/media/tv、…」；qBittorrent 網址 `abc` → 中文。
- 對抗式 review（subagent）：無 High。修掉 1 條 Med＋3 條 Low：
  - Med 完成設定失敗原本一律換成「請再試一次」，會蓋掉伺服器寫給人看的原因（金鑰無法儲存、設定已完成）——改成訊息是中文就原樣顯示，英文才換；「設定已完成」的後端訊息改中文「設定已經完成過了，請重新整理頁面。」。
  - Low 路徑存在但沒權限讀時不再說「找不到」，改說權限（PUID／PGID）。
  - Low 建議路徑去重（根目錄互相包含、結尾斜線）。
  - Low `setupService.spec.ts` 的舊英文字串換掉。
- 全量：`pnpm nx test web` 綠；`pnpm nx test api` 綠；`lint:all`、`web:typecheck` 綠。
- 🔗 AC Drift: NONE。📎 Contract Stamps: NONE（錯誤訊息文字，不是欄位契約）。
- 🎭 A11y Pre-Flight: PASS（沿用既有 `role=alert` 紅框）。
- 🎨 UX Verification: SKIPPED — 版面不變，只改紅框裡的文字；Flow N 沒有錯誤狀態稿。

### Discovery Triage

### File List

- apps/api/internal/services/setup_service.go、setup_service_test.go
- apps/api/internal/handlers/setup_handler.go
- apps/api/cmd/api/main.go
- apps/web/src/components/setup/SetupWizard.tsx、SetupWizard.spec.tsx
- apps/web/src/services/setupService.spec.ts
- _bmad-output/implementation-artifacts/disc-setup-wizard-container-path-hint.md、sprint-status.yaml

## Change Log

| Date       | Change                  |
| ---------- | ----------------------- |
| 2026-09-30 | create-story（SM Bob）。 |
| 2026-09-30 | dev-story（Amelia）＋對抗式 review → review。 |
| 2026-09-30 | PR #611 合併 → done。 |
