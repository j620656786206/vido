# Disc：活動記錄寫出字幕花多久 A（後端）——每次字幕生成都進活動記錄，帶「花多久、花多少錢、為什麼沒做成」

Status: ready-for-dev

**Source:** 2026-10-19 Alexyu：「目前生成出來的字幕只有在活動會看到本月 AI 花費，但是字幕生成花了多少時間有地方紀錄嗎？」
⚖️ **2026-10-19 Alexyu 裁定：紀錄放在活動頁的每一筆紀錄後面。** 設計定稿：`disc-2026-10-activity-subtitle-run-time-design`（done，PR #819；Sally 裁定 1–6，6 在複審改判：手機右側只留相對時間）。

⚖️ **SM 裁定（Bob，2026-10-19）— 拆成兩張：** 原單 `disc-2026-10-activity-subtitle-run-time` 後端 5 項（查詢＋連片名、失敗原因分類、活動服務混排、main.go 接線、真遷移鏈測試）、前端 7 項（型別、時間／錢的寫法、失敗原因文字、列的右側兩行與手機第三行、活動記錄改成獨立元件、畫廊夾具＋基準、測試），兩邊都 >3 → Story Splitting Rule 強制拆。
- **A（本單，後端）**：`GET /api/v1/activity` 的 `recent.events` 多一種 `kind: "subtitle"`，欄位一次定好（AC #1 [@contract-v1]）。
- **B（前端）**：`…-b` 照 K1-D-v2／K1-M-v2 畫。照本單 AC #1 的合約寫，**不用等 A 合併才開工**。
- ⛔ **合併順序：B 先合併（或跟 A 同一次部署）。** 現在的前端不看 `kind`：`ActivityHub.tsx:312-320` 只看 `result`，`failed` 以外一律寫「解析完成」、說明寫 `ev.detail`。A 若先上線，每一筆字幕生成都會被畫成「解析完成」、說明空白。

## 查到的事（2026-10-19，本 session 讀過的程式碼與正式機）

- **活動記錄現在只有解析事件。** `RecentEvent`（`apps/api/internal/services/activity_service.go:173-180`）註解寫「v1 sources parse jobs only」；`recentSection`（`:294-322`）只讀 `parse.ListAll(ctx, recentScanWindow)`，留下 completed／failed，最多 `recentEventsMax = 8` 筆（`:91-93`），`at` 優先用 `completed_at`（`:307-310`）。`parse == nil` 時整段 `unavailable`（`:295-297`）。
- **正式機的 `parse_jobs` 是空的**（2026-10-19 `sqlite3 -readonly /mnt/cache/appdata/vido/vido.db`：`select status,count(*) from parse_jobs` 沒有任何列）。所以正式機的「活動記錄」現在**永遠不出現**（前端 `events.length === 0` 時 `return null`，`ActivityHub.tsx:309`）。本單上線後，這一段第一次有內容，而且幾乎全是字幕生成。
- **每次生成都有一列 `subtitle_runs`**：`models.SubtitleRun`（`apps/api/internal/models/subtitle_run.go:103-160`）有 `Status`、`CueCount`、`ErrorMessage`、`StartedAt`、`CompletedAt *time.Time`、`SpentUSD *float64`（NULL＝沒記錄，不是 $0，`:120-125`）、`Route`（`:151`，"" = migration 041 以前）。狀態只有 pending／running／completed／failed／skipped（`:14-26`）。路線字彙在 `:174-186`（deliver_direct／convert_then_deliver／translate／skip／no_text_source／asr／online）。媒體種類 movie／series／episode（`:32-36`）。
- **`started_at` 是這一部開始處理的時間**：管線在 `ProcessItem` 第 2 步才建 run 列（`apps/api/internal/subtitle/process_item.go:114-131`，`StartedAt: p.now().UTC()`），批次是一部一部依序跑，所以「花多久」＝這一部從開始到結束（含等抽軌名額的時間，見 `process_item.go` 的 extraction slot 註解）。舊路徑 `transcription_ledger.go:91` 同樣在開跑時建列。
- **結束時一定寫 `completed_at`**：失敗 `failItem`（`process_item.go:843-861`）、預算暫停 `pauseASRItem`（`:614-626`）、舊路徑 `closeLedgerRun`（`apps/api/internal/services/transcription_ledger.go:128-140`）都設 `CompletedAt`。正式機 34 列裡 completed／failed 沒有一列 `completed_at` 是 NULL（2026-10-19 查）。
- **失敗原因只有原始英文字串**：`error_message` 存的是 Go 錯誤全文（含伺服器路徑），例如正式機：
  - `subtitle pipeline: episode …: translate: SUBTITLE_TRANSLATE_FAILED: … AI_INVALID_RESPONSE: Cannot parse AI response`（末日光明 S01E04，2026-10-15，花了 $0.126）
  - `… AI_TIMEOUT: AI parsing timeout exceeded`（動物方城市2、石油天王）
  - `… SUBTITLE_EXTRACT_FAILED: ffmpeg subtitle extraction failed: ffmpeg timed out after 10m0s on …`（三部 4K 電影）
  - `… place: placer: write failed: write temp file: open /media/tv/Peacemaker/Season01/…: permission denied`（和平使者 S01E01，2026-09-02，**翻完才寫不進去，花了 $0.196**）
  
  正式機沒有任何一列帶 `SUBTITLE_TARGET_NOT_WRITABLE`（同次查詢 count = 0）——設計稿的「末日光明 S01E01 · 影片所在的資料夾寫不進字幕 · 4 秒 · 沒花錢」是示意，不是正式機資料。
- **前端已經有一張「原始錯誤 → 人話」的表**：`failureCopy`（`apps/web/src/components/subtitle/generationEventCopy.ts:39-71`）用子字串比對 `AI_BUDGET_EXCEEDED`／`no audio track found`／`ffmpeg not available`／`transcription unavailable`、`API key not configured`／`AI_CREDIT_EXHAUSTED`、`帳號沒有額度`／`AI_UNAUTHORIZED`、`status 401`／`AI_TIMEOUT`、`timed out`、`deadline exceeded`，其餘「生成失敗」。**它沒有「寫不進資料夾」那一條。** 「影片所在的資料夾寫不進字幕」這句只在後端 409 回應裡（`apps/api/internal/handlers/transcription_handler.go:346-352`、`:374-377`），是開跑前的檢查，不是失敗紀錄。
- **錯誤哨兵字串**（分類要比對的）：`subtitle.ErrSubtitleTargetNotWritable` = `"SUBTITLE_TARGET_NOT_WRITABLE: target directory is not writable"`（`internal/subtitle/errors.go:41`）；`fsprobe.ErrNotWritable` = `"target directory is not writable"`（`internal/fsprobe/writable.go:18`）；`ai.ErrBudgetExceeded` = `"AI_BUDGET_EXCEEDED: …"`（`internal/ai/types.go:43`）；`ai.ErrAITimeout`（`:13`）、`ErrAIInvalidResponse`（`:15`）、`ErrAIUnauthorized`（`:23`）；`CreditExhaustedError.Error()` 以 `AI_CREDIT_EXHAUSTED:` 開頭（`internal/ai/credit_exhausted.go:35-37`）；取消的 run 前綴 `CancelledRunPrefix = "cancelled: "`（`process_item.go:719`）；延後付費的 skip 前綴 `DeferredPaidRunPrefix`（`:709`，狀態是 skipped，`:756`）。
- **後端送不帶文字的代碼是既有規則**：`ActiveJob` 註解「copy lives on the client for i18n — the backend stays copy-free」（`activity_service.go:111-113`）；`ux3-2-2-activity-api.md:44-45` 同。
- **查詢可以走索引**：`idx_subtitle_runs_completed_at`（`internal/database/migrations/032_add_subtitle_run_spend_and_completed_at_indexes.go:52`）。時間欄位一律存 UTC 文字，字典序比較正確（`subtitle_run_repository.go:118-121` 的註解、`CompletedMediaRefsSince` 的註解）。活動頁每 15 秒輪詢一次（`apps/web/src/hooks/useActivity.ts:23`），所以查詢要便宜。
- **介面只有一個實作**：`SubtitleRunRepositoryInterface`（`subtitle_run_repository.go:21-61`）在全 repo 只有 `*SubtitleRunRepository` 實作（`git grep "func (.*) AutoProducedBetween"` 只命中它與 `usage_report_service_test.go:47` 的窄介面假物件）。服務都用窄介面（`spendRunSource`，`services/subtitle_spend_service.go:83-86`）。
- **接線**：`cmd/api/main.go:1386` `NewActivityService(...)`，`:1389` `SetStepReader(...)`（setter 加可選來源的先例，`activity_service.go:41-43`）。
- **片名怎麼拿**：電影 `movies.title`；單集 `episodes.series_id → series.title`、`episodes.season_number`／`episode_number`（`internal/database/migrations/006_media_entities_enhancement.go:63-75`）。正式機的電影片名有時是檔名（`[bitsearch.to] Wake.Up.Dead.Man…mkv`，未比對的片）——照原樣送，不加工。
- **既有測試**：`activity_service_test.go` 的 `fakeParse`（`:30-46`）、`TestActivity_RecentCapsAndFiltersTerminal`（`:213`）、`TestActivity_RecentPrefersCompletedAt`（`:233`）、`TestActivity_NilSourcesDegradeGracefully`（`:244-257`，全 nil 時 recent 要 unavailable）、`TestActivity_DownloadsSectionWireKeys`（`:324`，JSON 鍵名的寫法可照抄）。真遷移鏈的資料庫：`setupSubtitleRunDB`（`repository/subtitle_run_repository_test.go:22-33`）；片庫種子 `seedMovie`／`seedSeries`（`repository/chinese_subtitle_filter_test.go:85-104`）、`seedEpisode`（`repository/series_chinese_rollup_test.go:27-41`），同一個 package 可直接用。

## Story

身為只有我一個人在用的 NAS 片庫主人，
我想要活動頁的「活動記錄」把每次字幕生成都列出來，後面寫花了多久、花了多少錢、沒做成是為什麼，
這樣我不用翻資料庫，就知道一集翻譯值不值得、哪裡卡住。

## Acceptance Criteria

1. **AC #1 [@contract-v1]：** `GET /api/v1/activity` 的 `recent.events[]` 多一種 `kind: "subtitle"`（既有 `kind: "parse"` 的形狀、鍵名、意義完全不變）。字幕事件的鍵：

   | 鍵 | 型別 | 何時出現 |
   |---|---|---|
   | `kind` | `"subtitle"` | 一定 |
   | `result` | `"completed"` \| `"failed"` | 一定（＝ run 的 status） |
   | `at` | RFC 3339 UTC | 一定（＝ `completed_at`） |
   | `media_type` | `"movie"` \| `"episode"` \| `"series"` | 一定 |
   | `media_title` | string | 一定；電影＝電影片名，單集／影集＝影集片名；片子已不在片庫時 `""` |
   | `season_number`、`episode_number` | int | 只有 `media_type = "episode"` 且找得到那一集 |
   | `route` | string（`models.SubtitleRunRoute*` 字彙） | `route` 不是 `""` 時 |
   | `cue_count` | int | `result = "completed"` 且 > 0 |
   | `duration_seconds` | int（四捨五入到秒） | `completed_at − started_at ≥ 0` 時 |
   | `spent_usd` | number（原值，不先四捨五入） | `spent_usd` 不是 NULL 時；**0 也要送 `0`**（「沒花錢」）；NULL 就不送（沒記錄 ≠ $0） |
   | `failure_reason` | 下表的代碼之一 | 只有 `result = "failed"` |

   `failure_reason` 代碼（不帶任何中文，文字在前端）：`cancelled`、`budget_exceeded`、`credit_exhausted`、`target_not_writable`、`no_audio_track`、`ffmpeg_missing`、`asr_unconfigured`、`unauthorized`、`timeout`、`other`。字幕事件**不送** `detail`（裡面會是含伺服器路徑的原始錯誤）。
2. **哪些 run 會出現：** `status IN ('completed','failed') AND completed_at IS NOT NULL`。`skipped`（含延後付費 `deferred-paid:`、skip、no_text_source）、`pending`、`running` 都**不出現**（running 已經在「進行中」）。取消的 run（`error_message` 以 `cancelled: ` 開頭）照樣出現，`failure_reason = "cancelled"`。
3. **跟解析事件按時間混排：** 兩個來源合併後依 `at` 由新到舊；同一時間先比 `kind`（`subtitle` 在前）再比各自的 id，順序固定。總數上限維持 `recentEventsMax = 8`。字幕來源只需查最新 8 筆（`LIMIT 8`、`ORDER BY completed_at DESC`，走 `idx_subtitle_runs_completed_at`）。
4. **失敗原因分類是一個純函式：** `models.SubtitleRunFailureReason(errorMessage string) string`，依序比對（第一個命中就回傳）：
   1. 前綴 `cancelled: `，或含 `context canceled` → `cancelled`
   2. `AI_BUDGET_EXCEEDED` → `budget_exceeded`
   3. `AI_CREDIT_EXHAUSTED` 或 `帳號沒有額度` → `credit_exhausted`
   4. `SUBTITLE_TARGET_NOT_WRITABLE` 或 `target directory is not writable`，或同時含 `placer: write failed` 與（`permission denied` 或 `read-only file system`）→ `target_not_writable`
   5. `no audio track found` → `no_audio_track`
   6. `ffmpeg not available` → `ffmpeg_missing`
   7. `transcription unavailable` 或 `API key not configured` → `asr_unconfigured`
   8. `AI_UNAUTHORIZED` 或 `status 401` → `unauthorized`
   9. `AI_TIMEOUT`、`timed out` 或 `deadline exceeded` → `timeout`
   10. 其餘（含空字串、`AI_INVALID_RESPONSE`）→ `other`

   第 4 條排在 timeout 前面：寫入探測逾時的訊息（`fsprobe/writable.go:79`）同時含 `not writable` 與 `timed out`，要歸到寫不進。表格測試至少涵蓋上面「查到的事」列的 4 種正式機原字串（逐字貼上）＋每個代碼一條＋寫入探測逾時那條。函式註解寫明「比對字串與 `generationEventCopy.ts` 的 `failureCopy` TABLE 是同一套；新增一條要兩邊一起改」。
5. **片名、季、集用一條 SQL 拿到**：新 repository 方法（建議名 `RecentTerminalRuns(ctx, limit)`）`LEFT JOIN` movies／episodes／series，回傳 run＋`MediaTitle`＋`SeasonNumber *int`＋`EpisodeNumber *int`；片子被刪掉時那一筆照樣回傳、片名 `""`。不准每一筆再各查一次（輪詢每 15 秒一次）。
6. **fail-soft（每個來源各自降級）：** 解析來源與字幕來源都沒有或都出錯 → `recent.status = "unavailable"`（`TestActivity_NilSourcesDegradeGracefully` 不改照綠）。只有一個出錯 → `status = "ok"`、只放另一個來源的事件，`slog.Warn` 記下出錯的來源。字幕來源沒接（nil）時行為跟今天一模一樣。
7. **真資料形狀（Rule 28）：** repository 測試用真遷移鏈（`setupSubtitleRunDB` 或 `newMigratedLibraryDB`）種：
   - 《末日光明》影集＋S01E02 單集；一列 completed／translate／568 句／`started_at 2026-10-19 01:21:30.067Z`／`completed_at 01:25:01.473Z`／`spent_usd 0.197439`（正式機 run `07621d1c` 原值）；
   - 同影集 S01E04 一列 failed／`spent_usd 0.126108`／`error_message` 為上面 AI_INVALID_RESPONSE 那條原字串；
   - 和平使者 S01E01 一列 failed／placer permission denied 原字串／`spent_usd 0.196393`；
   - 一部電影 completed／asr／`spent_usd 0`；一列 `spent_usd` NULL（舊列）；一列 skipped（不能出現）；一列 running（不能出現）；一列電影已刪除（片名 `""`）。
   
   service 層驗：S01E02 那筆 `duration_seconds = 211`、`spent_usd = 0.197439`、`season_number = 1`、`episode_number = 2`、`media_title = "末日光明"`、`cue_count = 568`、`route = "translate"`；和平使者那筆 `failure_reason = "target_not_writable"` 且 `spent_usd = 0.196393`（寫不進≠沒花錢）；S01E04 那筆 `other`；`spent_usd 0` 的那筆 JSON 有 `"spent_usd":0`；NULL 的那筆 JSON 沒有 `spent_usd` 鍵。
8. **JSON 鍵名測試：** 照 `TestActivity_DownloadsSectionWireKeys` 的寫法，把字幕事件 `json.Marshal` 後比對 AC #1 的鍵（含 omitempty 的有／無）；同一測試確認 parse 事件的 JSON 跟今天逐字相同。
9. **註解跟著改：** `RecentEvent` 的「v1 sources parse jobs only」（`activity_service.go:173-174`）與 `ActivityService` 型別註解（`:25-30`）改成現況；`Kind` 註解列出 `"parse" | "subtitle"`。
10. `cd apps/api && go test ./... && go vet ./... && ~/go/bin/staticcheck ./...` 綠；本機起 API（`AI_PROVIDER=claude`）`curl /api/v1/activity` 貼一次輸出到 Completion Notes（Rule 28 本機 smoke）。

## Tasks / Subtasks

- [ ] **T1 models（AC #4）**：新檔 `internal/models/subtitle_run_failure.go`：十個具名常數（`SubtitleRunFailureCancelled = "cancelled"` …）＋ `SubtitleRunFailureReason`；表格測試 `subtitle_run_failure_test.go`（正式機原字串逐字）。
- [ ] **T2 repository（AC #2、#3、#5、#7）**
  - [ ] `SubtitleRunRepositoryInterface` 加 `RecentTerminalRuns(ctx, limit int) ([]RecentSubtitleRun, error)`；型別 `RecentSubtitleRun{ Run models.SubtitleRun; MediaTitle string; SeasonNumber, EpisodeNumber *int }` 放 repository（`AutoProducedCounts` 的先例）。
  - [ ] SQL：`SELECT <subtitleRunColumns 加 r. 前綴>, COALESCE(m.title, s.title, sd.title, ''), e.season_number, e.episode_number FROM subtitle_runs r LEFT JOIN movies m ON r.media_type='movie' AND m.id=r.media_id LEFT JOIN episodes e ON r.media_type='episode' AND e.id=r.media_id LEFT JOIN series s ON s.id=e.series_id LEFT JOIN series sd ON r.media_type='series' AND sd.id=r.media_id WHERE r.status IN (?,?) AND r.completed_at IS NOT NULL ORDER BY r.completed_at DESC, r.id ASC LIMIT ?`。`subtitleRunColumns` 是無前綴字串（`:101-104`），要嘛另寫一個帶 `r.` 的常數、要嘛用子查詢包一層——**不要**手抄 24 個欄位（Rule 15 DB Column Sync，`:94-100` 的 bugfix-20-1 警告）。掃描沿用 `scanSubtitleRun` 的欄位順序。
  - [ ] 真遷移鏈測試（AC #7 的資料）。
- [ ] **T3 service（AC #1、#3、#6、#8、#9）**
  - [ ] `RecentEvent` 加 AC #1 的欄位（指標＋omitempty：`SpentUSD *float64`、`DurationSeconds *int`、`SeasonNumber`／`EpisodeNumber *int`；`CueCount int`、`Route`、`MediaType`、`MediaTitle`、`FailureReason string` 用 omitempty）。`MediaTitle` 不加 omitempty（一定送，`""` 也送）。
  - [ ] 窄介面 `recentRunSource interface{ RecentTerminalRuns(ctx, limit int) ([]repository.RecentSubtitleRun, error) }`＋ setter `SetRecentRuns(src)`（不改 `NewActivityService` 簽名，免動 10 個既有測試呼叫）。
  - [ ] `recentSection` 改成兩來源各自取、合併、排序（AC #3）、截 8、各自降級（AC #6）。
  - [ ] 測試：混排順序、上限 8、skipped／running 不出現、一個來源出錯、JSON 鍵名。
- [ ] **T4 接線**：`cmd/api/main.go:1389` 旁加 `activityService.SetRecentRuns(repos.SubtitleRuns)`。
- [ ] **T5 驗證（AC #10）**：go test／vet／staticcheck；本機 smoke。

## Dev Notes

### 不要做的事

- **不要把原始 `error_message` 放進 API**（不論 `detail` 或新欄位）：裡面有 NAS 上的絕對路徑與 ffmpeg stderr。只送代碼。
- **不要在後端組任何中文**（「字幕完成」「翻譯 568 句」「3 分鐘」「沒花錢」都是前端的事，B 單）。
- **不要把 `spent_usd` 在後端四捨五入**：前端 `usd()` 用 decimal.js 四捨五入（`apps/web/src/lib/currency.ts:47-50`），兩邊算法一致是專案規則（`currency.ts:10-21`）。
- 不要動 `parse` 事件的任何欄位或排序來源；不要動 `pendingSection`。
- 不要改 `failureCopy`（前端，B 單會把它整理成按代碼查的表）。
- 不要在 `recentEventsMax` 上加字幕專用的另一個上限：一個總上限，最新的贏。

### 實作提示

- `at` 用 `completed_at`（AC #2 已保證非 NULL）。`duration_seconds = int(math.Round(completed.Sub(started).Seconds()))`，負數（時鐘倒退）就不送。
- 合併時兩邊的 `at` 都是 `time.Time`，用 `sort.SliceStable` 加 AC #3 的次序鍵。
- 解析事件今天用 `ListAll(ctx, 50)` 再過濾——不動；字幕事件 `LIMIT recentEventsMax` 就夠（合併後只留 8）。
- 「花多久」包含批次裡等抽軌名額的時間（run 在 `ProcessItem` 第 2 步建列）。這是「這一集從開始處理到結束」，B 單照寫就好，不需要另外扣。⚠️ 未查證（dev 開工時先確認）：抽軌名額的等待是否確實發生在建列之後——不影響任何 AC，只影響 Completion Notes 怎麼描述。

### 測試標準

- Go：`testify`（`require`／`assert`）＋表格測試；repository 測試一定走真遷移鏈（Rule 28）。
- `lint:all` 不含 staticcheck，push 前另跑 `~/go/bin/staticcheck ./...`。

### Project Structure Notes

- 只動 `apps/api/internal/models/`（新檔）、`apps/api/internal/repository/subtitle_run_repository.go`（＋測試）、`apps/api/internal/services/activity_service.go`（＋測試）、`apps/api/cmd/api/main.go` 一行。沒有 migration、沒有新 endpoint。
- 跟進行中的 `disc-2026-10-library-list-triage-a`（`models/chinese_subtitle.go`、`repository/chinese_subtitle_sql.go`、movie／series repository、`library_service.go`）**沒有共用檔案**。

### Time-dependent visual coverage

- N/A — no wall-clock-reading components touched（純後端）。

### References

- [Source: _bmad-output/implementation-artifacts/disc-2026-10-activity-subtitle-run-time-design.md#Sally 的設計裁定]
- [Source: apps/api/internal/services/activity_service.go:25-30, 87-94, 111-113, 167-180, 294-322]
- [Source: apps/api/internal/models/subtitle_run.go:14-36, 103-160, 174-186]
- [Source: apps/api/internal/repository/subtitle_run_repository.go:21-61, 94-122, 280-305]
- [Source: apps/api/internal/subtitle/process_item.go:114-131, 614-626, 709-719, 843-861]
- [Source: apps/api/internal/services/transcription_ledger.go:91, 128-140]
- [Source: apps/web/src/components/subtitle/generationEventCopy.ts:39-71]（前端那一半的比對表）
- [Source: apps/api/internal/handlers/transcription_handler.go:346-352, 374-377]（「寫不進字幕」的原句）
- [Source: project-context.md Rule 20（AC #1 有蓋 contract 戳，B 單要 ack）、Rule 24、Rule 28]

## Change Log

| 日期 | 變更 |
|---|---|
| 2026-10-19 | 建單（Bob）。AC #1 首次蓋 `[@contract-v1]`；既有 `parse` 事件為隱含 v0、不變（純增欄位）。 |

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

### Discovery Triage

- N/A — 建單時沒有發現本單範圍外的新工作。（設計稿 K1 的「掃描完成」列在活動記錄裡一直沒有後端來源——ux3-2-2 起就記為 greenfield（`ux3-2-2-activity-api.md` Real vs. greenfield 表、sprint-status `ux3-activity-hub`），不是新發現。）

### File List
