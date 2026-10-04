# Bugfix G: 刪掉從沒啟動過的「下載完成自動解析」整套程式

Status: review

**Source:** `bugfix-g-parse-jobs-routes-unregistered`（P2）。**⚖️ 裁定：** Alexyu 2026-10-04 party mode（John／Winston／Sally／Bob＋founder-pm）一致建議「刪掉」，Alexyu 選 1。

## 前提更正（2026-10-04 查證）

- 單子說「兩支 API 沒掛、但 parse_jobs 一直在被寫」——**後半不成立**：`ParseQueueService`、`ParseWorker`、`CompletionDetector`、`ParseJobHandler` 都**從來沒在 `cmd/api/main.go` 建立**（`git log -S` 查無），正式庫 `parse_jobs` 0 筆。Story 4-5 標 done，但功能一次都沒跑過。
- 同一結論 2026-09-15 已在 `disc-2026-09-downloads-v2-no-parse-status` 查證過，當時裁定改做「入庫狀態」（走 Sonarr/Radarr 匯入紀錄）——已出貨（dl-import-1/2）。
- 新片進庫的實際路徑：媒體庫掃描＋Sonarr/Radarr 匯入；下載完自動找字幕是 13-5。

## 做了什麼

- 刪：`services/parse_queue_service.go`、`workers/parse_worker.go`（整個 `workers` 套件）、`services/completion_detector.go`、`handlers/parse_job_handler.go`，與各自的測試；`DownloadHandler` 的可選 parse queue 注入與 `parse_status` 欄位（從未出現在回應裡，前端也沒讀）；`enrichment_service.go` 只給 parse queue 用的 `nullTMDbID`；E2E `tests/e2e/parse-trigger.spec.ts`（打的是從未存在的路由，靠「回應不 ok 就不檢查」空洞通過）。
- 保留：`parse_jobs` 表與 `ParseJobRepository`。「活動」頁的待處理數／最近活動、首頁的「今日已處理／需要注意」仍讀它——表是空的，前端在 0 筆時本來就不顯示（待處理 0 會隱藏、最近活動顯示空狀態），沒有假數字露出。刪表要做資料庫遷移、拿掉讀取要改活動頁 API 與設計稿，好處不抵風險；**與 party mode 中 Winston「連讀取一起拿掉」的建議不同，記錄在此**。
- 移：原本寄放在 `parse_queue_service_test.go` 的共用測試替身（`mockPQ*` movie/series/season/episode repo、parser、metadata）搬到 `services/shared_repo_mocks_test.go`，名字不變，enrichment／scanner／recommendation 的測試繼續用。
- 不碰：`parse/progress` SSE（媒體庫批次比對在用，是另一個功能）。

## 驗證

- `go build`、`go vet`、`go test ./...`、`staticcheck-2026.1`（零輸出）全綠。
- 前端無改動（沒有任何前端程式呼叫被刪的路由）。

## Change Log

- 2026-10-04 建立＋完成（Bob 開單、Opus 5.5 實作）。
