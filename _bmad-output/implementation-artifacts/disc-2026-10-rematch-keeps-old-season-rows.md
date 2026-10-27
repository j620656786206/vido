# disc-2026-10-rematch-keeps-old-season-rows：影集換成另一部之後，季的集數也跟著換，不再寫舊那部的「60 集」

Status: review

<!-- SM Bob create-story 2026-10-27；行號為 main `fc305cef`。Rule 24 lane ③，由 disc-2026-10-series-wrong-version-b 立案。 -->

**Source:** 做 -b 時發現。正式機唯讀查詢（2026-10-27，`sqlite3 -readonly /mnt/cache/appdata/vido/vido.db`）：將夜（`8f55e12f…`）第 1 季 `episode_count = 60`、`tmdb_id = 111783`，是 2018 真人版的季。#913 讓「重新解析」會換成 2026 動畫版（TMDb 282136，20 集），#915 讓使用者可以「重新選片」，但兩條路都只改影集那一列，**季那幾列不動**。

## 為什麼會這樣

- 季的資料放在 `seasons` 表，一季一列，掃描時依季資料夾建立（`media_ingest_service.go:225`）。
- 集數、季名、季海報由兩個地方補：背景補資料（`season_details_backfill_service.go`，只補空的欄位）、展開那一季時順手寫回（`series_season.go` 的 `rememberSeasonDetails`）。
- 換片時（`persistSeriesMatch`，`enrichment_manual_match.go:437`）沒人去碰季列。結果：
  - 季標題繼續寫舊那部的集數（「60 集」），要展開那一季一次才會變。
  - 舊那部的季名、季海報、季的 TMDb 編號也留著；`MergeSeasonDetails` 的規則是「空的不蓋掉有的」，新那部沒有的欄位永遠不會被清掉。
  - 新那部沒有的季（例如舊那部有第 2 季、新那部只有 1 季），會一直留著舊那部的資料。

## Acceptance Criteria

1. **換片就重設季**：影集的 TMDb 編號換了（自動比對、讀 NFO、重新解析、手動選片都算），寫完影集那一列之後，這部影集現有的每一列季資料都改成新那部的：季的 TMDb 編號、季名、簡介、海報、首播日、集數。**直接取代**，不是「空的不蓋」——新那部沒有的欄位要清空。
2. **新那部沒有這一季**：季列保留（集數檔案還掛在它底下），TMDb 那幾個欄位全部清空。季標題會顯示「劇集數未知」，不會冒充舊那部。
3. **不刪任何一列**：季列、集數列都不刪，集數的檔案路徑、字幕狀態不動。
4. **TMDb 編號沒變就不動**：同一部重新比對（例如重新解析又比對到同一部）不重設，保留展開時寫回的精確集數。
5. **拿不到新那部的資料就不動**：TMDb 讀取失敗 → 記 log，季列維持原樣（不因為一次網路錯誤把資料清光）；影集比對照樣成功。
6. **換成沒有 TMDb 編號的來源**（例如豆瓣）：新的編號是空的時，舊那部的季資料清空（AC 2 的做法）。⚠️ 實作時發現：搜尋路線比對到豆瓣時，影集那一列**不會清掉舊的 TMDb 編號**（`applyMetadataToSeries` 只在有新編號時才寫），所以這條目前走不到；那是另一個問題，另立 `disc-2026-10-douban-match-keeps-stale-tmdb-id`（P3）。
7. **前端**：「重新解析」成功後，影集的季清單也重新讀（`useReparseItem`，跟 #915 的手動選片一樣），不會停在一小時的快取。
8. **測試**：
   - 換片 → 季列換成新那部的值；新那部沒有的欄位被清空；新那部沒有的季，TMDb 欄位清空但列還在。
   - TMDb 編號沒變 → 不寫。
   - TMDb 讀取失敗 → 不寫、比對照樣成功。
   - 三條換片路線（搜尋、NFO、手動選片）都會觸發。
   - `useReparseItem`：影集才讓季清單失效。

## 不要做的事

- 不刪季列、不重建集數列。
- 不填影集那一列的 `number_of_seasons`／`number_of_episodes`（正式機 87 部全是空的，畫面本來就不顯示，另議）。
- 不改背景補資料、不改展開時寫回的規則。

## Tasks

- [x] Task 1 — `EnrichmentService` 加選填的季列存取（`SetSeasonRepository`，窄介面 `FindBySeriesID`／`Update`），`main.go` 接上（AC 1）
- [x] Task 2 — `persistSeriesMatch` 收「換片前的 TMDb 編號」，編號變了就重設季列（AC 1～6）
- [x] Task 3 — `useReparseItem` 影集多讓季清單失效（AC 7）
- [x] Task 4 — 測試、vet、staticcheck；NAS 備份實測將夜（AC 8）

## Dev Notes

- 三個呼叫點：`ApplyTMDbMatch`（`enrichment_manual_match.go:129`）、`enrichSeriesFromNFO`（`enrichment_series_identity.go:136`）、`enrichSeries`（`enrichment_service.go:471`）。三個都在套用新資料前就有 `series.TMDbID`。
- `GetTVShowDetails` 走 `CacheService`，搜尋路線剛在 `applyTMDbSeriesFacets` 抓過同一個編號，不會多打 TMDb。
- 季列存取：`repository.SeasonRepository` 的 `FindBySeriesID`／`Update`（`season_repository.go:106、191`）；`Update` 會寫所有欄位，可以寫 NULL。
- 設定方式照 `SetLibraryRoots`（窄依賴、nil = 舊行為），不擴大 `NewEnrichmentService` 的參數。

### Time-dependent visual coverage

- N/A — 後端＋一個 hook。

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（claude-opus-5-5），2026-10-27。

### Completion Notes List

- 新檔 `enrichment_series_seasons.go`：`SetSeasonRepository`（窄介面 `FindBySeriesID`／`Update`，nil = 舊行為）與 `resetSeasonsForNewShow`。TMDb 編號變了才動；每一列季資料**直接取代**成新那部的（季編號、季名、簡介、海報、首播日、集數），新那部沒有的欄位寫 NULL；季的評分是舊那部的，一律清掉；新那部沒有的季，TMDb 欄位全清、列保留。TMDb 讀取失敗或季列讀不到 → log、不動。
- `persistSeriesMatch` 多收「換片前的 TMDb 編號」，寫完影集那一列就呼叫重設。三個呼叫點（手動選片、tvshow.nfo、搜尋）都在套用新資料前記下舊編號。
- `main.go` 接上 `repos.Seasons`。
- 前端 `useReparseItem`：影集重新解析成功後，季清單、各季集數清單失效（跟 #915 的手動選片一樣）。
- 反向驗證：拿掉 `persistSeriesMatch` 裡的重設呼叫 → `TestApplyTMDbMatch_NewShowReplacesTheSeasonRows`、`TestEnrichSeries_EveryRematchPathResetsTheSeasons` 紅。
- **NAS 實測（2026-10-27，正式資料庫的備份＋臨時容器，只換執行檔；拿掉 AI 金鑰與密碼鎖，媒體唯讀）**：將夜（`8f55e12f…`）重新解析前 `series 83612|tmdb`、第 1 季 `60 集|111783|2018-10-31`；重新解析後 `series 282136|nfo`、第 1 季 `20 集|438147|2026-04-23`，季海報也換成新的；`GET /series/:id/seasons` 回 `episode_count: 20`；log `season rows reset for the new match … from_tmdb_id=83612 to_tmdb_id=282136 seasons=1`。測完容器與資料夾都刪了。
- 範圍外（另立）：豆瓣比對不會清掉舊的 TMDb 編號 → `disc-2026-10-douban-match-keeps-stale-tmdb-id`（P3）。影集那一列的 `number_of_seasons`／`number_of_episodes` 仍沒人寫（正式機 87 部全空，畫面不顯示），照 story 不做。
- `go test ./internal/services/... ./internal/repository/... ./internal/handlers/... ./cmd/api/...` 全過；`go vet`、`staticcheck` 乾淨；`useLibrary.spec` 36 個全過；`web:typecheck` 過。

### File List

- `apps/api/internal/services/enrichment_series_seasons.go`（新）
- `apps/api/internal/services/enrichment_series_seasons_test.go`（新）
- `apps/api/internal/services/enrichment_manual_match.go`
- `apps/api/internal/services/enrichment_series_identity.go`
- `apps/api/internal/services/enrichment_service.go`
- `apps/api/cmd/api/main.go`
- `apps/web/src/hooks/useLibrary.ts`、`useLibrary.spec.tsx`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`、`disc-2026-10-series-wrong-version-b.md`（標 done）
