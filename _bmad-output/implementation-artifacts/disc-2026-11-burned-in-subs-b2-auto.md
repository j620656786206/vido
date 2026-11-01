# Story disc-2026-11-burned-in-subs-b2-auto：讓燒字幕檢查自己跑——新片、舊片庫、每月上限、批次跳過、花費分開列

Status: done

**Source:** 傘狀 `disc-2026-11-burned-in-subs`。⚖️ Alexyu 2026-11-01：**1A**（新片掃描後自動檢查、設每月上限、現有片庫做一次總檢查）、**2A**（燒了簡體也算有中文、批次生成跳過）；2026-11-02「接著做」。前置 -b1（PR #962）：`BurnedInSubtitleService.CheckSeason`、`episodes.burned_in_subtitle`／`burned_in_checked_at`、`ai_side_spend` kind `burned_in_check`。

## 做法

1. **找要檢查的季**：有集數「有檔案、`subtitle_tracks = '[]'`、`burned_in_subtitle IS NULL`」，而且（`burned_in_checked_at IS NULL` 或距今超過 7 天）的 (series, season)。按影集名、季號排序，一次一季、序列執行。
2. **判斷不了也要留紀錄**：-b1 判斷不了時什麼都不寫，下次掃描會一直重試、一直花錢。改成判斷不了時，在該季 `subtitle_tracks='[]'` 的集數寫 `burned_in_checked_at`（值維持 NULL），7 天後才再試。`no_episode_to_check` 不用寫。
3. **什麼時候跑**：
   - 開機後延遲 2 分鐘跑一輪（現有片庫的總檢查就是這一輪）。
   - 每次 scan-complete 後跑一輪（用 `ScannerService.AppendOnScanComplete`，不要覆蓋現有的 callback）。
   - 之後每 6 小時一輪（字幕軌背景清點會陸續把新集數寫成 `[]`）。
   - 同時只能有一輪在跑；正在跑時再被叫，直接略過。
4. **每月上限**：新設定 `VIDO_BURNED_IN_MONTHLY_CAP_USD`（預設 `1.00`，`0` 代表停用自動檢查）。每輪開始前、每季檢查前，算這個月 `ai_side_spend` 裡 `burned_in_check` 的總額；達到上限就停，log 一行 Warn（含已花多少、上限多少），下個月自動恢復。手動觸發的 API 不受上限限制（使用者自己按的）。沒設 Claude 金鑰 → 這輪直接結束，log Info，不算錯誤。
5. **批次生成跳過（2A）**：生成待辦清單（`HasTraditionalChineseSubtitle` 的呼叫點：`episode_repository.go` 兩處、`movie_repository.go` 一處）改成「已有繁中，**或畫面上燒了中文（zh-Hans／zh-Hant／zh）**」就離開清單。函式改名或新增一個意思清楚的函式，不要讓「Traditional」名不副實。
6. **檔案換了就重查**：集數的檔案被移走或換版本（`ClearMissingFile`、`ReleaseFileFromOtherEpisodes`、字幕軌因檔案簽章改變而重新清點時），把該集的 `burned_in_subtitle`、`burned_in_checked_at` 清成 NULL。
7. **花費分開列**：`GET /api/v1/subtitles/spend` 的 `parse_calls`／`parse_usd` 不再包含 `burned_in_check`；新增 `burned_in_calls`／`burned_in_usd`（同樣的 [from, to) 期間）。前端「本月 AI 花費」卡片（`apps/web/src/components/activity/SpendSection.tsx`）在有檢查費用的月份多一格「畫面字幕檢查」，寫法比照「檔名解析」那格（只有該月有呼叫時才出現）。

## Acceptance Criteria

1. [x] 找季的查詢與排序；跳過最近 7 天內判斷不了的季。
2. [x] 判斷不了時寫 `burned_in_checked_at`（值 NULL）。
3. [x] 開機延遲一輪、scan-complete 後一輪、每 6 小時一輪；同時只跑一輪。
4. [x] 每月上限（設定、預設 1.00、0＝停用）；到上限停並 Warn；手動 API 不受限；沒金鑰安靜結束。
5. [x] 生成待辦清單：燒了任何中文的集數離開清單（含 zh-Hans／zh）；測試覆蓋。
6. [x] 檔案移走／換版本／重新清點時清掉 burned 兩欄；測試覆蓋。
7. [x] spend API 分開 `burned_in_calls／burned_in_usd`；`parse_*` 不再含它；前端卡片多一格；前端單元測試＋gallery fixture 照慣例。
8. [x] `go build／vet／test`、`staticcheck`、`gofmt` 乾淨；`nx run web:test`、`web:typecheck`、`nx lint web` 乾淨。

## Dev Agent Record

### 改了什麼

- **找季**（`EpisodeRepository.FindSeasonsNeedingBurnedInCheck`）：有集數「有檔案、`subtitle_tracks='[]'`、`burned_in_subtitle IS NULL`、`burned_in_checked_at` 是 NULL 或早於 7 天前」的 (影集, 季)，排除已移除的影集，按影集名（不分大小寫）、季號排序。
- **判斷不了也留紀錄**：-b1 的 `CheckSeason` 判斷不了（截不到畫面、回覆讀不懂、OpenCC 失敗，含「第一集 none、第二集判斷不了」）時，改成在該季「`[]` 而且還沒有答案」的集數寫 `burned_in_checked_at`（值維持 NULL，`MarkSeasonBurnedInChecked`）。`no_episode_to_check`、模型呼叫本身失敗都不寫。手動 API 也走同一條，所以手動按到判斷不了的季，自動檢查也會等 7 天。
- **自己跑**（新的小型別 `services/burned_in_sweep.go` 的 `BurnedInSweeper`，`CheckSeason` 不動）：開機 2 分鐘後一輪（就是現有片庫的總檢查）、每次 scan-complete 一輪（`AppendOnScanComplete` 追加，不蓋掉別的 callback）、之後每 6 小時一輪；一季一季照順序跑；同時只跑一輪（`atomic.Bool`），正在跑時再被叫就直接略過、不排隊。main.go 照字幕軌清點的寫法接線，關機時先 cancel 再 `Wait()`，才關資料庫。
- **每月上限**：`VIDO_BURNED_IN_MONTHLY_CAP_USD`（`config.go`，預設 `1.00`，`0`＝不自動檢查、main 也不啟動它；負數或非數字退回預設並 Warn；有列進啟動時的設定 log）。每輪開始前、每季檢查前讀這個月 `ai_side_spend` 裡 `burned_in_check` 的總額，到了就停並 Warn（含已花、上限、環境變數名）。月份用跟花費卡片同一個「伺服器時區的日曆月」——把 `SubtitleSpendService.monthWindow` 抽成共用的 `CalendarMonthWindow`。讀不到花費就停這輪（不把「不知道」當成 0）。手動 API 不受上限。沒有 Claude 金鑰：這輪 Info 一行就結束，連季清單都不讀。
- **批次生成跳過（2A）**：`HasTraditionalChineseSubtitle` 換成意思清楚的 `HasTraditionalOrBurnedInChinese`（已有繁中，或畫面上燒了 zh-Hans／zh-Hant／zh 任何一種），三個呼叫點（集數兩處、電影一處）都換過去，舊函式刪掉。
- **檔案換了就重查**：`ClearMissingFile`、`ReleaseFileFromOtherEpisodes` 一起清 burned 兩欄；`UpdateSubtitleTracks` 在檔案簽章跟存著的不一樣時清掉（同一個檔案重讀、例如補讀字幕簡繁，就保留）。用 SQL 的 CASE 一次寫完，SQLite 的 SET 都是拿舊值比，所以比到的是舊簽章。
- **花費分開列**：新增 `AISideSpendRepository.TotalsByKindBetween`；`GET /api/v1/subtitles/spend` 的 `parse_calls`／`parse_usd` 不再含 `burned_in_check`，新增 `burned_in_calls`／`burned_in_usd`（同一段期間）。Swagger 註解同步。
- **前端**：型別與舊 API 相容（沒有這兩欄就當 0）；「本月 AI 花費」卡片在有檢查的月份多一格「畫面字幕檢查」，接在「檔名解析」後面，寫法、規則和它一樣（沒有就不出現、不畫「—」、不進「依模型」）；只有檢查花了錢的月份，卡片也會出現。新 gallery fixture `activity-spend-card/burned-in-check`；既有 fixture 只補 `burnedInCalls: 0`，畫面不變。
- **文件**：`.env.example`、`README.md`、`docs/development.md`、Unraid 安裝指南英文＋中文都加了這個環境變數（中英兩份表格被 Prettier 重新對齊）。

### 決定

- **首頁「本月 $x」把檢查費用加回去**：規格只說卡片多一格，但首頁讀數帶的總額原本是「翻譯＋語音辨識＋舊紀錄＋`parse_usd`」，檢查從 `parse_*` 拿出來後不加回去，首頁就會少算。所以 `monthSpendUsd` 也加上 `burned_in_*`，總額跟拆開前一樣。
- **正在跑時再被叫就略過，不排隊**（照規格）：跟字幕軌清點那種「排一輪」不同。掃描剛結束時字幕軌清點還沒把新集數寫成 `[]`，所以這輪可能撿不到新片，要等 6 小時那輪或下次掃描。要更快的話，之後可以改成「字幕軌清點跑完才叫它」。
- **一輪裡模型呼叫失敗就整輪停**（金鑰被拒、服務掛掉時每一季都會一樣失敗），下一輪再試；這種失敗不寫 `burned_in_checked_at`，因為不是「看了判斷不了」。
- **上限是每季檢查前看一次**，所以最後一季可能超過一點點（一季最多兩次呼叫，約 $0.005）。
- 金額跟其他欄位一樣四捨五入到分，所以一次檢查（約 $0.002）在卡片上會顯示 `1 次 · $0.00`，跟檔名解析目前的行為一致。

### 測試證據

- Go 新測試：`repository/episode_burned_in_sweep_test.go`（找季：排序、同季只列一次、已有答案／7 天內判斷不了／沒檔案／有字幕軌／沒讀過字幕軌／已移除影集都不列，8 天前判斷不了的會列；判斷不了的戳記只寫該季 `[]` 又沒答案的集數；換簽章清掉、同簽章保留、檔案移走清掉、檔案換到別集清掉；`TotalsByKindBetween` 分 kind、半開區間、沒有的 kind 不出現）、`repository/episode_burned_in_test.go`（燒繁、燒簡、燒中文看不出簡繁的集數離開生成清單，none 跟沒檢查過的留著，計數一致）、`models/chinese_subtitle_burned_test.go`（`HasTraditionalOrBurnedInChinese` 八種組合）、`services/burned_in_sweep_test.go`（照順序跑完、retry 時間點、月份範圍；上限：開始前到了就什麼都不做、跑到一半到了就停、別種花費不算、0 停用、讀不到花費就停；沒金鑰安靜結束、模型失敗整輪停、金鑰中途消失、列季失敗、ctx 取消；同時只跑一輪；開機延遲後跑一輪再照間隔跑；開機延遲中被取消就不跑——都注入時鐘和毫秒級間隔，`-race -count=5` 通過）、`services/burned_in_subtitle_service_test.go`（判斷不了會戳記、判斷出來／沒集數／模型失敗不戳記）、`services/subtitle_spend_service_test.go`（parse 跟 burned_in 分開、空的是 0、`CalendarMonthWindow` 跨年）、`config/config_test.go`（預設、覆寫、0、亂寫、負數）。
- 前端新測試：`SpendSection.spec.tsx`（第四格接在檔名解析後面、不進依模型、0 次不出現、只有檢查花錢的月份也顯示）、`subtitleSpendService.spec.ts`（對應欄位、舊 API 沒欄位讀成 0）、`HomeReadoutBand.spec.tsx`（總額加上檢查費用、只有檢查的月份也有總額）。
- `cd apps/api && go build ./... && go vet ./... && go test ./...`：44 個套件全 ok；`~/go/bin/staticcheck ./...` 無輸出、exit 0；改到的 Go 檔 `gofmt -l` 乾淨。
- `pnpm nx run web:test`：304 個檔案、5007 個測試全過；`pnpm nx run web:typecheck` 通過；`pnpm nx lint web` 0 errors（168 個 warning 都是既有的，改到的檔案單獨跑 eslint 沒有任何訊息）；改到的 web 檔案 Prettier 檢查通過。
- 沒有產生任何視覺基準圖；新的 gallery fixture 在 CI 會缺 `-linux` 基準，照慣例等 Visual Regression 自動開的 bootstrap PR。

### 本機實測（Opus，2026-11-02）

- 正式機資料庫副本（migration 049 自動套用）＋真的 Claude／ffmpeg／OpenCC；《超自然檔案》S1、《白髮》S1 的檔案指到從 NAS 剪下的片段，其他季維持 NAS 路徑（本機讀不到 → 截不到畫面）。
- 開機 2 分鐘後自動跑一輪：找到 47 季、全部檢查完、2 季有結果（超自然檔案 S1 → `none` 22 集；白髮 S1 → `zh` 58 集，這次抄到的字看不出簡繁）、其餘 45 季「截不到畫面」→ 寫了 `burned_in_checked_at`、不重試；花 US$0.006，8 秒跑完。
- `GET /subtitles/spend?period=month`：`burned_in_calls=3`、`burned_in_usd=0.01`，`parse_calls=0`（不再混在一起）。白髮的 `chinese_subtitle` → `zh`。

## Change Log

- 2026-11-02：建立。
- 2026-11-02：Amelia 實作完成（自動檢查排程、每月上限、判斷不了留紀錄、批次跳過燒字幕、換檔重查、花費分開列＋卡片新格），送 review。
