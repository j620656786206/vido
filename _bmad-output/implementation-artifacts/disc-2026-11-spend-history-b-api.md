# Story disc-2026-11-spend-history-b-api：一支 API 列出每次執行的花費、預估、模型與來源

Status: done

**Source:** 傘狀 `disc-2026-11-spend-history`。給 -d 的新頁讀。不做：畫面、改月報（`GET /subtitles/spend` 維持原樣）。

## 端點（⚖️ Alexyu 2026-11-01「然後實作」＝照這份欄位名做；合併時標 [@contract-v1]）

`GET /api/v1/subtitles/runs?month=2026-11&category=&page=1&page_size=50`

- `month`：伺服器時區的月份（跟 `GET /subtitles/spend` 同一套時鐘與月界）；省略＝本月。格式錯 400 `VALIDATION_INVALID_FORMAT`。
- `category`：`chinese_track`／`translate_english`／`asr`／`legacy`／`no_subtitle`，省略＝全部。只篩 `items`，`summary` 和 `series` 不受影響。

```json
{
  "month": "2026-11",
  "summary": {
    "spent_usd": 3.91, "runs": 12, "unpriced_runs": 0,
    "comparable_runs": 5, "comparable_spent_usd": 1.20, "comparable_estimate_usd": 1.35
  },
  "series": [
    { "date": "2026-11-01", "spent_usd": 3.91, "runs": 9,
      "by_category": { "chinese_track": 0, "translate_english": 0.81, "asr": 3.10, "legacy": 0 } }
  ],
  "items": [
    {
      "id": "…",
      "started_at": "2026-10-28T13:33:46Z",
      "completed_at": "2026-10-28T13:38:25Z",
      "status": "completed",
      "failure_reason": "",
      "media": { "id": "…", "type": "episode", "title": "Lioness", "season": 1, "episode": 1 },
      "route": "asr",
      "source": "audio",
      "category": "asr",
      "spent_usd": 0.52,
      "estimate_usd": null,
      "translation_model": "claude-sonnet-5",
      "asr_provider": null,
      "asr_model": null,
      "batch_id": null,
      "triggered_by": "manual"
    }
  ],
  "page": 1, "page_size": 50, "total": 12
}
```

- **category 對照**（-c 的表）：`deliver_direct`／`convert_then_deliver` → `chinese_track`；`translate` → `translate_english`；`asr` → `asr`；路線沒記 → `legacy`；`no_text_source`／`skip`／其他 → `no_subtitle`（不進 `series.by_category`）。
- 沒記錄的數字是 `null`，不是 0（PRODUCT.md 原則 1）。`summary.unpriced_runs` = `spent_usd` 是 NULL 的次數。
- `comparable_*`：只算 `spent_usd` 和 `estimate_usd` 都有值的執行，前端拿來算「實際是預估的 N%」。
- `translation_model` 只在有翻譯的路線（`translate`、`asr`）才給；其他給 `null`（`model_id` 在那些路線只是版本欄位）。
- `series`：當月每一天一筆（本月只到今天），沒有執行的日子 `spent_usd: 0, runs: 0`（那天是真的 0）；`by_category` 四鍵加總＝當天 `spent_usd`（`no_subtitle` 不花錢；NULL 花費當 0 計入 `runs` 但不加金額）。
- 含失敗與中途停止的執行（它們也花了錢）；`failure_reason` 用既有分類器（`models/subtitle_run_failure.go`），跟活動頁「最近」同一套。
- 片名：電影 `movies.title`；單集帶劇名＋季集；媒體已刪 → `title: null`。
- 排序：`started_at` 新到舊。

## Acceptance Criteria

1. [x] 端點照上面的形狀回應，標 [@contract-v1]，swag 註解寫齊。
2. [x] 分頁、`month`、`category` 篩選；`month` 格式錯回 400 `VALIDATION_INVALID_FORMAT`（沿用 7-6b）。
3. [x] `series` 每日一筆、含 0 的日子；`by_category` 加總等於當日 `spent_usd`；`summary` 不受 `category` 篩選影響。
4. [x] NULL 誠實：舊資料的 `estimate_usd`／`source`／`asr_*` 是 `null`；`spent_usd` NULL 的執行 `spent_usd: null`。
5. [x] Repository 用真的 SQLite 測（Rule 28），含 movie、episode、已刪媒體三種。

## Dev Agent Record

### 做了什麼

- **Repository**：新增 `FinishedRunsStartedBetween(from, to)`——`started_at` 落在 `[from, to)` 的 completed／failed／skipped 執行，新到舊（同時間用 id 排），沿用 `RecentTerminalRuns` 的片名 join（電影片名；單集＝劇名＋季集；媒體已刪＝空字串）。兩支共用新的 `queryRunsWithNames` 掃描，`RecentTerminalRuns` 行為不變。
- **Service**：`services.SubtitleRunHistoryService.History`（新檔），注入時鐘、伺服器時區月界（跟 `SubtitleSpendService.monthWindow` 同一套）。`RunCategory` 依 -c 的表把 route 對到五類。回應型別 `SubtitleRunHistory`（欄位名照本檔，一個都沒改），標 [@contract-v1]。
- **Handler**：`GET /api/v1/subtitles/runs`（`SubtitleRunHistoryHandler`，swag 註解齊），`main.go` 接線並註冊在 `/subtitles/spend` 旁。格式錯（`month` 不是 `YYYY-MM`、`category` 不認得、`page`／`page_size` 不是正整數或 `page_size` > 200）一律 400 `VALIDATION_INVALID_FORMAT`。
- `docs/swagger.*` 沒有收錄 `/subtitles/spend`，所以沒有動 docs。

### 實作時做的判斷（故事檔沒寫死的地方）

- **哪些執行算進來**：completed、failed（含中途停止，`cancelled` 也是 failed）、skipped（`skip`／`no_text_source` → `no_subtitle`）。pending／running 不列——結束前花費還沒蓋章。
- **月份歸屬看 `started_at`**（跟列表排序一致）；月報 `GET /subtitles/spend` 看 `completed_at` 而且只算 completed，所以兩邊合計本來就可能不同（本頁含失敗的錢）。
- `series.runs` 只算四個入圖類別（範例的 12 次／當日 9 次就是這樣）；`summary.runs`、`summary.unpriced_runs` 照字面算全部執行（skipped 的 NULL 花費也算 unpriced）。
- 金額：每天每類各自四捨五入到分，當天 `spent_usd` ＝四類加總（保證加起來剛好）；`summary.spent_usd` 是全部已記花費加總後再取到分。單筆 `spent_usd`／`estimate_usd` 原值不取整（跟活動頁一樣）。
- `comparable_*`：照字面算「`spent_usd` 和 `estimate_usd` 都有值」的所有執行（含失敗）。一筆都沒有時兩個金額是 `null`（不是 0），`comparable_runs` 是 0。
- 本月 `series` 只到今天；過去的月份整月；未來的月份 `series: []`。`route` 沒記也給 `null`（`category` 是 `legacy`）。`failure_reason` 只在 failed 給（跟活動頁一樣），其他是 `""`。電影的 `media.season`／`episode` 是 `null`；單集只有在還有片名時才帶季集（活動頁規則）。

### 檔案

- `apps/api/internal/repository/subtitle_run_repository.go`（介面＋`FinishedRunsStartedBetween`＋共用 `queryRunsWithNames`）
- `apps/api/internal/repository/subtitle_run_history_test.go`（新）
- `apps/api/internal/services/subtitle_run_history_service.go`（新）
- `apps/api/internal/services/subtitle_run_history_service_test.go`（新）
- `apps/api/internal/handlers/subtitle_run_history_handler.go`（新）
- `apps/api/internal/handlers/subtitle_run_history_handler_test.go`（新）
- `apps/api/cmd/api/main.go`（接線＋註冊路由）

### 測試

- Repository（Rule 28：真的 migration chain、app 的 `sqlite-utc` driver、檔案 DB、`foreign_keys(1)`）：電影、單集（劇名＋季集）、已刪媒體（片名空）、失敗執行、skipped；running／窗外（含剛好在終點）不列；本地時區的邊界跟 UTC 一樣比；新欄位 NULL 讀回 nil。
- Service：category 對照表；本月只到今天、沒執行的日子是 0、Taipei 00:30 歸在當地那天；`by_category` 加總＝當天；過去月份整月、未來月份空；`comparable_*` 只算兩值都有、沒有時 null；category 只篩 items（summary／series 相同）與分頁；單筆 null 誠實、`translation_model` 只在 translate／asr；失敗分類用既有分類器；格式錯回 `ErrInvalidRunHistoryQuery`。
- Handler（真的 service＋假 ledger）：預設本月／第 1 頁／50 筆；JSON 上沒記錄的鍵是 `null` 不是缺鍵；category 不影響 summary／series；九種格式錯都 400 `VALIDATION_INVALID_FORMAT`；ledger 錯 500。
- `gofmt`（改到的檔）乾淨；`go vet ./...` exit 0；`~/go/bin/staticcheck ./...` exit 0；`go test ./internal/... ./cmd/...` exit 0（44 個 package ok）。

### Review 補（2026-11-01）

- 「實際是預估的 N%」（`comparable_*`）只算**完成**的執行：失敗／中途停止只花了預估的一部分，算進去會把比例拉低。
- `unpriced_runs` 不算**跳過**的執行：跳過的沒買任何東西，花費空著不是「沒記到」。
- 9 月路線沒記的舊執行，只要有花錢就帶 `translation_model`：那時只有翻譯會花錢，`model_id` 就是翻譯模型。
- 拿正式機資料庫副本（2026-11-01，跑完 migration 048）實際呼叫：9 月 24 筆、US$9.14（全是 legacy），10 月 10/28 那天語音辨識 US$4.76；單集片名「特種部隊：母獅 S2E1」、季集都正確。
- 月份看**開始**時間、含失敗；月報 `/subtitles/spend` 看完成時間、只算完成，兩邊月合計可能不同（前端要寫清楚是「這個月開始的執行」）。

## Change Log

- 2026-11-01：建立（Bob）。
- 2026-11-01：實作完成（Amelia），Status → review。
- 2026-11-01：review 補三條規則＋正式機資料副本驗證。
