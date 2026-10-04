# Story infra-optin-usage-report-a2: 使用者自己打開後，Vido 每週送一次匿名計數；送了什麼，設定頁看得到

Status: ready-for-dev

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

**Epic:** standalone（`infra-optin-usage-report` 家族）· **Priority:** P1 · **Size:** M（後端 only）
**Source:** PRD amendment `_bmad-output/planning-artifacts/prd/prd-telemetry-amendment.md`（P1-040-1~7、NFR-T1~T6，2026-10-02 核准）；Alexyu 2026-10-04 裁定接收端用自架 Umami。
**Depends on:** `infra-optin-usage-report-a1-ledger-version`（計數與版本號）
**Blocks:** `infra-optin-usage-report-b-frontend`（前端開關、精靈步驟、上次送出內容）

---

## Story

As a Vido user who chose to help,
I want Vido to send one anonymous count per week — and to show me exactly what it sent,
so that the maintainer can see Vido is being relied on, without anything about my library ever leaving my NAS.

---

## Context —— 查到的事（2026-10-04 逐行查證，SM Bob）

**設定存取（直接沿用）：**
- 設定是 key-value 表，加 key 不需要 migration（`apps/api/internal/database/migrations/003_create_settings_table.go:23-29`，`key` 主鍵 + `value` + `type`）。
- 預設關閉的讀法：`SettingsService.GetBoolWithDefault` `apps/api/internal/services/settings_service.go:209-214`（找不到 key 就回預設值）。
- 單一設定的 GET/PUT 端點範本：`apps/api/internal/handlers/localization_handler.go:29-35`（`/subtitles/localization` 的 `GET`／`PUT`）。
- **路由順序陷阱**：專屬的 `/settings/...` 路由必須在 `settingsHandler` 之前註冊，否則被 `/settings/:key` 吃掉——`apps/api/cmd/api/main.go:1261-1270`（`logHandler`、`backupHandler`、`modelSettingsHandler` 等行都註明「Must be before settingsHandler」，`settingsHandler` 在 `:1270`）。

**首次設定精靈後端：**
- 精靈只在最後一步一次寫入：`apps/api/internal/services/setup_service.go:103`（`CompleteSetup`），最後 `:198` 寫 `setup_completed`。
- 每步驗證是 switch，**新步驟 id 會回 `unknown step`**：`setup_service.go:213-228`。
- Request 結構 `apps/api/internal/models/settings.go:35-46`（`SetupConfig`）。

**背景排程（照抄模式）：**
- 啟動：`main.go:1336`（`go backupScheduler.Start(schedulerCtx)`）、`:1340-1341`（scan scheduler 自己的 ctx）。
- 關閉：`main.go:1418-1419`（cancel + Stop）、`:1460`。
- 預設關閉的排程讀法前例：`apps/api/internal/services/backup_scheduler.go:150-160`（沒設定就回 `Enabled: false`）。
- 目前**沒有任何排程把「上次執行時間」存進 DB**（backup scheduler 只放記憶體）；本單要新增，才能跨重啟維持 7 天節奏。

**記錄遮罩陷阱：**
- DB log 會遮罩 32 字以上的連續十六進位字串：`apps/api/internal/logger/db_handler.go:27`。安裝編號若用無連字號的 32 位 hex 會在 log 裡變成遮罩字；用標準 UUID 字串（有連字號）即可避開。

**Umami 收資料方式（研究結果，2026-10-04；來源見 References）：**
- `POST {umami}/api/send`，不需要 token。body：`{"type":"event","payload":{"website":"<uuid>","hostname":…,"url":…,"name":…,"data":{…},"id":…}}`；有 `name` 才算自訂事件。
- **User-Agent 會被機器人過濾**：Go 預設的 `Go-http-client/1.1`、`Vido/1.0` 都會被判成機器人；被判機器人時回 **HTTP 200 `{"beep":"boop"}`**（假成功）。實測不會被判機器人的格式：`Mozilla/5.0 (X11; Linux x86_64) Vido/<version>`。
- 成功回應含 `sessionId`——**要用它判斷成功，不能只看 200**。
- Umami 不存 IP，但會用代理標頭的 IP 查「國家」並存下來。PRD 允許的欄位不含國家，所以送出時帶 `payload.ip: "127.0.0.1"` 跳過地理查詢，並用 `payload.id` = 安裝編號區分不同機器。
- 事件名稱 ≤50 字，不可以 `= + - @` 開頭；`data` 是扁平物件。
- ⚠️ 未查證（dev 開工時先確認）：上述 `payload.ip` 行為與 Alex NAS 上的 Umami 版本是否一致——用真實送出驗證（AC #9）。

---

## Acceptance Criteria

### AC #1 — 設定與預設（P1-040-1、NFR-T3）
- 新增設定：是否開啟（**預設關閉**）、安裝編號、上次送出時間、上次送出的完整內容。key 名稱由 dev 依現有 settings key 慣例決定。
- 全新安裝、從未操作 → 不送任何東西。
- 關閉後，下一次排程不送。

### AC #2 — 安裝編號（P1-040-4a）
- 第一次「開啟」時才產生；隨機 UUID（v4 字串格式）。**不得**由硬體、使用者、帳號、網路資訊推導。
- 關閉再開啟沿用同一個編號（「同一台」的定義不變）。

### AC #3 — 回報內容（P1-040-4、4b、NFR-T1、NFR-T2）
- 使用者資料只有：安裝編號、Vido 版本（a1 AC #4）、最近 7 天自動產生的字幕總數、依來源的三個分組數（a1 AC #5）。
- 其餘欄位只能是 Umami 協定需要的**固定常數**（website、hostname、url、事件名稱、`ip: "127.0.0.1"`），不含任何使用者資料。
- 送出的整個 body 就是設定頁要顯示的「上次送出內容」——同一份 bytes，不另外組一份給畫面看。

### AC #4 — 頻率與排程（P1-040-5、P1-040-6、NFR-T4）
- 背景排程定期檢查（建議每小時一次）：開啟 + 接收端已設定 + 距離上次**嘗試** ≥ 7 天 → 送一次。
- 成功與失敗都記錄嘗試時間 → 不論成敗，每台最多 7 天一次，不會連續重試。
- 開啟後的第一次送出在下一次檢查時發生（≤ 1 小時），不用等 7 天。
- 遵守 Rule 14：接受 ctx、可取消；在 `main.go` 啟動與關閉段接線（比照 `:1336-1341`、`:1418-1419`），並在 DB 關閉前停止。

### AC #5 — 送出與失敗處理（P1-040-6、NFR-T5）
- User-Agent 用不會被判機器人的格式（見 Context），版本號取自 a1。
- 成功的判斷：回應含 `sessionId`。`{"beep":"boop"}`、非 2xx、逾時、連不上 → 一律視為失敗。
- 失敗：只寫 slog Warn（Rule 2），**不**跳任何使用者看得到的錯誤、**不**影響其他功能、**不**更新「上次送出內容」（畫面仍顯示上一次成功送出的內容）。
- HTTP client 建一次重用（Rule 14），有逾時。

### AC #6 — 接收端設定（Alexyu 2026-10-04 裁定：自架 Umami）
- Umami 網址與 website ID 在 release build 時注入（比照 `Dockerfile:91` 的 `-X`，值放 CI 設定，不寫進程式碼）。
- **未注入（本機開發、fork 自建）→ 功能不可用**：排程不送，API 回報「不可用」，讓前端可以隱藏或說明。

### AC #7 — 設定 API（P1-040-1、P1-040-3）
- `GET /api/v1/settings/usage-report`：回傳是否可用、是否開啟、上次送出時間、上次送出的完整內容（原樣字串；從未送出則為 null）。
- `PUT /api/v1/settings/usage-report`：設定開或關，回傳同 GET 的結構。
- `{success, data}` 信封（Rule 3）；JSON snake_case（Rule 6）；**在 `settingsHandler` 之前註冊**（`main.go:1261-1270`）。
- Swagger 註解補上（Rule 15）。

### AC #8 — 首次設定精靈後端（P1-040-2）
- `SetupConfig` 增加「是否開啟匿名回報」欄位，預設 false（`models/settings.go:35-46`）。
- `CompleteSetup` 依該欄位寫入設定（`setup_service.go:103` 起）；false 時不產生安裝編號。
- `ValidateStep` 新增此步驟的 case（`setup_service.go:213-228`），不再回 `unknown step`。步驟 id 由 dev 與 b 對齊（建議 `usage-report`）。

### AC #9 — 測試（Rule 28 real-shape）
- **內容白名單**（NFR-T1）：測試斷言送出的使用者資料欄位集合 = 允許清單。
- **不含片庫資料**（NFR-T2）：在有片名、檔名、路徑、API 金鑰的真 DB 上組出 body，斷言這些值都不出現在 body 裡。
- **預設關閉**（NFR-T3）：全新 DB → 排程不送。
- **頻率**（NFR-T4）：可控時鐘下，7 天內第二次檢查不送；成功與失敗都算一次嘗試。
- **失敗不打擾**（NFR-T5）：接收端連不上／回 `{"beep":"boop"}`／回 400 → 無錯誤外溢、「上次送出內容」不變。
- **Umami 回應**：用真實回應錄下的 fixture（成功含 `sessionId`、`{"beep":"boop"}`、`400 Website not found.`），不要手打假的 happy path。
- **本機 smoke**（寫進 Completion Notes）：對 Alex NAS 上的 Umami 實際送一次，貼回應，並確認 Umami 後台事件資料裡**沒有國家欄位**。

### AC #10 — 使用者文件（P1-040-7，Rule 17 雙語）
- 新增 `docs/usage-report.md` + `docs/usage-report.zh-TW.md`：送什麼（附一份實際 body 範例）、絕對不送什麼、怎麼開關、資料存在哪、多久送一次。
- `README.md` 加一行連結（目前 README 只有中文版）。

### AC #11 — 範圍紅線
- 不做前端（b）。
- 不做 Umami 的備份與網站建立（NAS 維運，見 Discovery Triage `ops-umami-usage-report-prep`）。
- 不新增 Rule 7 錯誤前綴（對外送出失敗不經 API 回傳；設定 API 的驗證錯誤沿用 `VALIDATION_*`）。

---

## Tasks / Subtasks

- [ ] Task 1: 設定 keys、預設讀法、安裝編號產生（AC #1、#2）
- [ ] Task 2: 回報 body 組裝（呼叫 a1 的計數與版本）（AC #3）
- [ ] Task 3: Umami sender：UA、`sessionId` 判斷、逾時、client 重用、build-time 網址／website ID 注入（AC #5、#6）
- [ ] Task 4: 每週排程 + main.go 啟動／關閉接線（AC #4）
- [ ] Task 5: `GET`／`PUT /api/v1/settings/usage-report` handler + 路由順序 + Swagger（AC #7）
- [ ] Task 6: 精靈後端：`SetupConfig`、`CompleteSetup`、`ValidateStep`（AC #8）
- [ ] Task 7: 測試矩陣 + 本機 smoke（AC #9）
- [ ] Task 8: 雙語使用者文件 + README 連結（AC #10）

Cross-stack split check：後端 7、前端 0 → 不需拆（前端已獨立成 `infra-optin-usage-report-b-frontend`）。

---

## Dev Notes

### Rule 27（五個要求）在這張單怎麼對應
| 要求 | 本單做法 |
|---|---|
| ① 限速 | 每台 7 天最多一次，排程本身就是限速器；不另建 `rate.Limiter`（理由寫在程式註解） |
| ② 快取 | N/A——只送不收 |
| ③ 降級 | 失敗安靜、不重試、不影響其他功能（AC #5） |
| ④ 錯誤碼 | 不經 API 外露，無新錯誤碼（AC #11） |
| ⑤ 金鑰 | 沒有密鑰（website ID 不是秘密）；網址與 ID 由 build 注入，不寫死 |

### 已知陷阱
- **假成功**：Umami 對機器人回 200 `{"beep":"boop"}`。只看 status code 會把每次失敗都記成成功。
- **路由被吃掉**：沒排在 `settingsHandler` 前面，`/settings/usage-report` 會被 `/settings/:key` 當成一個名叫 `usage-report` 的設定 key。
- **log 遮罩**：無連字號的 32 位 hex 安裝編號會在 DB log 被遮罩（`db_handler.go:27`）。
- **畫面內容與實際送出不一致**：「上次送出內容」必須存送出的原始 bytes，不要重新序列化。

### 不要做的事
- 不要把任何片名、檔名、路徑、模型名稱、花費放進 `data`。
- 不要在「關閉」時刪除安裝編號（AC #2）。
- 不要因為送出失敗而改變任何字幕或掃描行為。

### Project Structure Notes
- 後端全部在 `apps/api/`（Rule 1）；handler → service → repository（Rule 4）；介面放 services／repository 套件（Rule 11）。
- 文件在 `docs/`（Rule 17）。

### Time-dependent visual coverage
N/A — no wall-clock-reading components touched（後端 only；前端在 b）。

### References
- PRD：`_bmad-output/planning-artifacts/prd/prd-telemetry-amendment.md`；`prd/non-functional-requirements.md`（NFR-S7、S8）
- `project-context.md` Rule 2、3、4、6、11、13、14、15、17、27、28
- Umami：[Sending stats (v3)](https://docs.umami.is/docs/api/sending-stats)、[send route.ts](https://github.com/umami-software/umami/blob/master/src/app/api/send/route.ts)、[detect.ts](https://github.com/umami-software/umami/blob/master/src/lib/detect.ts)、[ip.ts](https://github.com/umami-software/umami/blob/master/src/lib/ip.ts)、[Track events](https://docs.umami.is/docs/track-events)（研究日期 2026-10-04，Umami master = v3.4.0）

---

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

- Ultimate context engine analysis completed - comprehensive developer guide created（SM Bob，2026-10-04）

### Discovery Triage

- ③ `ops-umami-usage-report-prep`（backlog，2026-10-04 SM 建檔）：上線前在 NAS 的 Umami 建立 Vido 專用 website、補上 Umami 資料庫的定期備份（ai-context `ops-risk.md` 風險 #2）。屬 NAS 維運、不在 Vido repo；**必須在第一個 release 帶上接收端設定之前完成**。

### File List
