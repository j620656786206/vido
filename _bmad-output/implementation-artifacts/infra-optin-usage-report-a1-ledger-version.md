# Story infra-optin-usage-report-a1: 字幕紀錄分得出「自動／手動」、線上字幕也入帳，程式知道自己的版本號

Status: review

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

**Epic:** standalone（`infra-optin-usage-report` 家族，Alexyu 2026-10-04 裁定獨立單、不掛 epic）· **Priority:** P1 · **Size:** M（後端 only）
**Source:** PRD amendment `_bmad-output/planning-artifacts/prd/prd-telemetry-amendment.md`（P1-040，2026-10-02 核准）；founder-pm 產品決定 2026-10-02。
**Blocks:** `infra-optin-usage-report-a2-reporter`（回報要數的東西，這張先把它記下來）
**⚖️ Alexyu 裁定（2026-10-04）：** ① 範圍照本單做（分自動／手動、線上字幕入帳），不走「先不分、不數線上」的簡化版；② 首頁「今天處理了」開始計入線上找到字幕的影片——接受這個畫面變化。

---

## Story

As the Vido maintainer measuring the north-star metric（每週有自動產生繁中字幕的安裝數）,
I want every subtitle Vido produces to be recorded with whether it was automatic or user-triggered and which source path it came from, and the running binary to know its own release version,
so that the weekly usage report (a2) can count "subtitles automatically produced in the last 7 days" honestly, broken down by source.

---

## Context —— 查到的事（2026-10-04 逐行查證，SM Bob）

**北極星要數的東西，現在資料庫裡記不出來。三個缺口：**

1. **分不出自動還是手動。** `subtitle_runs` 的欄位清單 `apps/api/internal/repository/subtitle_run_repository.go:81-84` 沒有任何「誰觸發」的欄位。唯一相近的 `batch_id` 只在同意批次時才有值（`apps/api/internal/subtitle/process_item.go:125-127` 從 ctx 取 `GenerationBatchIDFromContext`；`apps/api/internal/services/transcription_ledger.go:83` 同），所以「自動」和「單項手動」都是空字串，分不開。
2. **線上字幕（射手網／OpenSubtitles）完全不入帳。** 線上引擎找到字幕後只更新影片／影集的狀態欄：`apps/api/internal/subtitle/engine.go:372-386` `updateSubtitleFound` 呼叫 `UpdateSubtitleStatus`，**不寫 `subtitle_runs`**。而「下載完成自動找字幕」正是走這條：`apps/api/internal/subtitle/request_trigger.go:137`（電影）、`:165`（影集）呼叫 `t.engine.Process(...)`。
3. **程式不知道自己的版本號。** 執行期沒有任何版本變數（後端 Explore 掃過 `ldflags`／`main.version`／`debug.ReadBuildInfo`／`VIDO_VERSION`，皆無）。Dockerfile 唯一的 `-X` 注入是 TMDb 金鑰 `Dockerfile:91`；CI 只在 OCI label 推導 semver（`.github/workflows/docker.yml:119-128`，`type=semver,pattern={{version}}`）。

**現有可以直接沿用的東西：**
- Route 字彙表 `apps/api/internal/models/subtitle_run.go:161-168`：`deliver_direct`／`convert_then_deliver`／`translate`／`skip`／`no_text_source`／`asr`。以字串存、不是 typed enum（同檔 `:155-160` 註解說明理由）。
- 7 天區間讀取 `CompletedRunsBetween` `apps/api/internal/repository/subtitle_run_repository.go:343-346`（已正規化 UTC 比對）。
- 依 route 分組的前例：`apps/api/internal/services/subtitle_spend_service.go:162-176`。
- 加欄位 migration 的冪等寫法：`apps/api/internal/database/migrations/041_add_subtitle_run_ledger_columns.go:32-49`（`columnExists` guard）。最新 migration 是 `042_normalize_go_string_timestamps.go` → 本單用 **043**。
- ldflags 注入前例：`Dockerfile:91`（`-X github.com/vido/api/internal/config.bundledTMDbKey=...`）。

**自動產生字幕的入口（本單要標成 auto 的地方）：**
- 免費線 AutoGenerator：`apps/api/internal/subtitle/auto_generation.go:505` 以 `ProcessItemOptions{FreeOnly: true}` 呼叫 `ProcessItem`。⚠️ FreeOnly 時 `translate`／`no_text_source` 會被延後（`process_item.go:190-197`），所以自動線目前只會產出 `deliver_direct`／`convert_then_deliver`。
- 下載完成觸發：`request_trigger.go:137`、`:165`（線上引擎）。

**其他 `ProcessItem` 呼叫點（預設 manual）：** `apps/api/internal/subtitle/worker_pool.go:464`（`queued.opts` 帶選項過 queue）、`apps/api/internal/subtitle/batch.go:485`、`apps/api/cmd/api/generation_batch_runner_adapter.go:85`。
⚠️ 未查證（dev 開工時先確認）：`WorkerPool` 的 enqueue 來源是否只有手動端點；若有自動來源，同樣要帶 auto。

---

## Acceptance Criteria

### AC #1 — `subtitle_runs` 增加觸發來源欄位（migration 043）
- 新欄位記錄這次 run 是 **自動** 還是 **使用者觸發**。欄位名與值由 dev 依 Rule 6 決定（建議 `trigger TEXT`，值 `auto`／`manual`）；舊資料保持空值（不回填、不猜）。
- Migration 用 `columnExists` guard（冪等），比照 `041`。
- Rule 15 欄位同步：model struct、`subtitleRunColumns`、INSERT、row scan 全部帶上新欄位。

### AC #2 — 自動入口一律寫成 auto，其餘一律 manual
- AutoGenerator（`auto_generation.go:505`）與下載完成觸發（`request_trigger.go:137,165`）產生的 run 標 auto。
- 傳遞方式：沿用既有「選項跟著 queue 走」的 `ProcessItemOptions`（`worker_pool.go:464` 用 `queued.opts`），**不要只靠 ctx**（ctx 不會跟著 queue 傳）。線上引擎同理加在它的 `ProcessOptions`。
- 沒有明確標 auto 的呼叫一律 manual（預設值寫在程式裡，不靠呼叫端記得）。

### AC #3 — 線上引擎產出的字幕入帳
- 線上引擎成功放置字幕時，寫一筆 `subtitle_runs`：`status=completed`、`completed_at`、`route` 用新的字串值（建議 `online`，加進 `models/subtitle_run.go` 字彙表並更新 `:155-160` 註解）、觸發來源依 AC #2。
- 找不到字幕／失敗：**什麼都不寫**（不寫 failed）。理由：首頁「需要注意」格會數 `status=failed` 的 run（`apps/api/internal/services/home_summary_service.go:211`），線上搜尋找不到是常態，寫 failed 會讓那格數字暴增。
- 寫帳失敗不得讓找字幕失敗（比照 `transcription_ledger.go:73-90` 的「bookkeeping 失敗只 Warn」語意）。

### AC #4 — 執行期版本號
- 新增一個 build 時注入的版本字串（比照 `Dockerfile:91` 的 `-X`），release image 帶 tag 的 semver（例：`0.1.2`），本機／未注入時為 `dev`。
- CI（`.github/workflows/docker.yml`）把 metadata-action 推出的 version 傳進 build arg。
- 提供一個讀取函式給 a2 用；**不新增 API 端點**（本單範圍外）。

### AC #5 — 給 a2 用的讀取方法
- Repository（或 service）提供「最近 N 天內 completed、且觸發來源為 auto 的 run，依來源分組計數」的讀取，分組對應 PRD P1-040-4b：
  - 內建字幕：`deliver_direct`、`convert_then_deliver`、`translate`
  - 線上來源：AC #3 的新 route
  - 語音辨識：`asr`
  - `skip`、`no_text_source` **不算**產出
- 回傳結構自訂，但要讓 a2 一次拿到「總數」與「三個分組數」。

### AC #5b — 既有讀取端的行為（迴歸）
- **每月花費**（`apps/api/internal/services/subtitle_spend_service.go:131` 讀 `CompletedRunsBetween`，`:197-215` 依 route switch）：新 route 沒有對應 case → 不計入任何花費列。測試釘住：加入 online run 後，花費摘要的數字與加入前完全相同。
- **首頁「今天處理了」**（`home_summary_service.go:188` 讀 `CompletedMediaRefsSince`）：線上找到字幕的影片**會開始被算進去**。這是預期中的行為改變（線上找到字幕也是 Vido 處理了它；**Alexyu 2026-10-04 已同意**），在 PR 說明裡寫清楚。
- **首頁「需要注意」**（`home_summary_service.go:211` 數 failed）：因 AC #3 不寫 failed，數字不變；測試釘住。

### AC #6 — 測試（Rule 28 real-shape）
- Migration：用真的 migration chain（`migrations.NewRunner` + `RegisterAll(GetAll())` + `Up`）建 DB，驗證 043 冪等（跑兩次不壞）、舊資料新欄位為空。
- AutoGenerator 路徑寫出的 run = auto；generation batch／worker pool 路徑 = manual。
- 線上引擎成功 → 一筆 completed run（新 route、正確 trigger）；失敗 → 無 completed run；寫帳失敗 → 字幕仍放置成功。
- AC #5 讀取：在真 DB 放入跨越 7 天邊界、各 route、auto/manual 混合的資料，驗證計數與分組。
- 版本：未注入時回 `dev`。

### AC #7 — 範圍紅線
- 不送任何東西出去（那是 a2）。
- 不改任何 API 回應、SSE、前端。
- 不回填歷史資料的 trigger。

---

## Tasks / Subtasks

- [x] Task 1: Migration 043 + model/repo 欄位同步（AC #1）
- [x] Task 2: `ProcessItemOptions`／線上引擎 `ProcessOptions` 加觸發來源；AutoGenerator 與 request trigger 標 auto（AC #2）
- [x] Task 3: 線上引擎成功時寫 `subtitle_runs`（新 route）；字彙表與註解更新（AC #3）
- [x] Task 4: 版本字串 ldflags 注入 + Dockerfile build arg + CI 傳值（AC #4）
- [x] Task 5: 依來源分組的 7 天計數讀取（AC #5）
- [x] Task 6: 測試矩陣（AC #6）

Cross-stack split check：後端 6、前端 0 → 不需拆。

---

## Dev Notes

### 已知陷阱
- **Rule 15 欄位同步**：新欄位漏進 `subtitleRunColumns` 或 scan，讀回來永遠是零值且單元測試不會發現（`project-context.md` Rule 15 bugfix-20-1 前例）。用真 DB 測（Rule 28）。
- **ctx 不跨 queue**：`BatchID` 用 ctx 傳（`process_item.go:125-127`），但 worker pool 的 item 是排隊後才處理；觸發來源要走 options。
- **時間格式**：`CompletedRunsBetween` 依賴 UTC 字串比對（`subtitle_run_repository.go:340-342` 註解）；寫 `completed_at` 一律 `.UTC()`（bugfix-h 前例）。
- **線上引擎無 runs repo**：`Engine` 目前沒有注入 `subtitle_runs` repository，需要在 `main.go` 接線（Rule 15 main.go wiring）。

### 不要做的事
- 不要新增 Rule 7 錯誤前綴（本單沒有對外 API 錯誤）。
- 不要動 `backup_service.go:296` 寫死的 `AppVersion: "1.0.0"`（已另立 backlog，見 Discovery Triage）。

### Project Structure Notes
- 全部在 `apps/api/`（Rule 1）。Migration 檔名：`internal/database/migrations/043_<name>.go`。

### Time-dependent visual coverage
N/A — no wall-clock-reading components touched（後端 only）。

### References
- PRD：`_bmad-output/planning-artifacts/prd/prd-telemetry-amendment.md`（P1-040-4、P1-040-4b）
- `project-context.md` Rule 1、6、13、14、15、28
- 上列 Context 所有 `path:line`

---

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（2026-10-04，dev-story）

### Debug Log References

### Completion Notes List

- Ultimate context engine analysis completed - comprehensive developer guide created（SM Bob，2026-10-04）
- **dev 開工前重新核對（2026-10-04，main = d012d3ec）**：story 引用的行號在 main 上仍成立；`WorkerPool` 的 enqueue 來源不必再追——觸發來源改成「只有明確說 Automatic 才是 auto」，任何沒標的呼叫一律 manual，所以 worker pool、consent batch、線上批次都自動是 manual。
- **AC #1**：migration 043 加 `subtitle_runs.triggered_by TEXT`（可為 NULL、`columnExists` 冪等、不回填）。**欄名用 `triggered_by` 不用 `trigger`：TRIGGER 是 SQLite 關鍵字。** model 加 `TriggeredBy` 與 `SubtitleRunTriggeredAuto`／`Manual` 常數；repository 欄位常數、INSERT 佔位、UPDATE、values、scan 全部同步到 24 欄（順手修正已過時的「All 20 columns」註解）。
- **AC #2**：`ProcessItemOptions.Automatic`（走 options 不走 ctx，worker pool 是排隊後處理）；`process_item.go` 建 run 時以 `runTrigger(opts.Automatic)` 寫入；AutoGenerator 傳 `Automatic: true`。線上引擎 `ProcessOptions.Automatic`，request trigger 電影與影集兩條都傳 true。Route C 轉錄的 ledger 列明確寫 manual（只有人會啟動它）。
- **AC #3**：`Engine.SetRunLedger` + `recordDelivered`：放置成功才寫一筆 `completed`／`route=online`／`output_path`／`started_at`（Process 開頭取）／`completed_at`／`triggered_by`。找不到或失敗**什麼都不寫**；寫帳失敗只 Warn、字幕照樣算成功。新增字彙 `SubtitleRunRouteOnline = "online"`，route 註解同步。`main.go` 接 `subtitleEngine.SetRunLedger(repos.SubtitleRuns)`。
- **AC #4**：`config.Version()`（未注入或空白 → `dev`）；Dockerfile `ARG VIDO_VERSION=dev` 併入同一個 `-ldflags` 的 `-X …config.buildVersion`；`docker.yml` build-args 傳 `steps.meta.outputs.version`（tag 建置 = `0.1.2` 這種 semver，分支建置 = 分支名，PR = `pr-NN`）。ldflags 實測：`go test -ldflags "-X …buildVersion=9.9.9"` 讀出 `9.9.9`。⚠️ 本機沒有跑 docker build（api-builder stage 要拉完整 go mod）；Dockerfile 只動了 ARG 與 `-X` 一段，CI 的 Docker job 會實建。
- **AC #5**：`SubtitleRunRepository.AutoProducedBetween(from, to)` → `AutoProducedCounts{Embedded, Online, ASR}` + `Total()`。SQL：`completed` + `triggered_by='auto'` + 半開區間 `[from, to)` + `GROUP BY route`；`deliver_direct`／`convert_then_deliver`／`translate` → Embedded，`online` → Online，`asr` → ASR，其餘（含空 route）不算。
- **AC #5b 迴歸**：每月花費——加入 online run 前後 `MonthSummary` 完全相等（測試釘住）；首頁「需要注意」——線上找不到時 failed 數為 0（測試釘住）；首頁「今天處理了」——線上找到字幕的影片會開始被算進去（Alexyu 2026-10-04 已同意的預期變化）。
- **測試**：migration 043 欄位存在＋舊列 NULL＋重跑冪等；repository round-trip 第 24 欄、未標記讀回空字串、`AutoProducedBetween` 分組／邊界（`from` 算、`to` 不算）／manual 與 pre-043 與 failed 與 skip 與無 route 皆不算、空結果為 0；pipeline 預設 manual、Automatic → auto；AutoGenerator 與 request trigger（電影＋影集）都帶 Automatic；engine 用**真 repository＋真 migration chain**驗證 online 列（Rule 28）、預設 manual、找不到不寫任何列、寫帳失敗仍成功、沒接 ledger 時行為不變；`config.Version` 兩態。
- **Gate**：`go test ./...` 全綠、`go vet ./...` 0、`staticcheck-2026.1 ./...` 0、`pnpm nx test web` 全綠、`test:cleanup` 無殘留。觸碰的 Go 檔 gofmt 乾淨。
- 🔗 AC Drift: FOUND（範圍界定，非破壞）— `sub-7-6a-run-ledger.md` AC #3 [@contract-v1]「每個 run 終態發一次 `subtitle_run_receipt`」：該契約的範圍是**管線的 lane**（`recordTerminal`）；本單新增的線上引擎 ledger 列**不發收據**（它不是管線 item，前端也沒有對應的收據畫面），收據 payload 欄位一個不動。
- 📎 Contract Stamps: FOUND（upstream `sub-7-6a` v1 ×1——`ReceiptPayload` 未改，confirmed against [@contract-v1]；`ProcessOutcome`／`RouteKind` [@contract-v1] 未改，ledger 新字彙是字串、不是 `RouteKind` 成員）。
- 🎭 A11y Pre-Flight: N/A (100% backend — no apps/web/ files touched)
- 🎨 UX Verification: SKIPPED — no UI changes in this story
- Pre-existing：無新增失敗。

### Discovery Triage

- ③ `disc-backup-appversion-hardcoded`（backlog，2026-10-04 SM 建檔）：`backup_service.go:296` 寫死 `AppVersion: "1.0.0"`，AC #4 有了真版本號後應改用；不阻擋本單。

### File List

- apps/api/internal/database/migrations/043_add_subtitle_run_triggered_by.go（新）
- apps/api/internal/database/migrations/043_add_subtitle_run_triggered_by_test.go（新）
- apps/api/internal/models/subtitle_run.go
- apps/api/internal/repository/subtitle_run_repository.go
- apps/api/internal/repository/subtitle_run_repository_test.go
- apps/api/internal/subtitle/pipeline.go
- apps/api/internal/subtitle/process_item.go
- apps/api/internal/subtitle/process_item_ledger_test.go
- apps/api/internal/subtitle/auto_generation.go
- apps/api/internal/subtitle/auto_generation_test.go
- apps/api/internal/subtitle/engine.go
- apps/api/internal/subtitle/engine_ledger_test.go（新）
- apps/api/internal/subtitle/request_trigger.go
- apps/api/internal/subtitle/request_trigger_test.go
- apps/api/internal/services/transcription_ledger.go
- apps/api/internal/services/subtitle_spend_service_test.go
- apps/api/internal/config/version.go（新）
- apps/api/internal/config/version_test.go（新）
- apps/api/cmd/api/main.go
- Dockerfile
- .github/workflows/docker.yml
- _bmad-output/implementation-artifacts/sprint-status.yaml
- _bmad-output/implementation-artifacts/sub-7-6a-run-ledger.md（AC drift reference — see Completion Notes；未修改）

### Change Log

| Date | Change |
| --- | --- |
| 2026-10-04 | dev-story：migration 043 `triggered_by`、自動入口標 auto、線上引擎成功入帳（route=online）、build-time 版本號、`AutoProducedBetween` 分組計數；Status → review |
