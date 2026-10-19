# Disc：媒體庫「全部」頁往下捲不再漏片、不再重複，「評分」排序真的照評分排

Status: ready-for-dev

**Blocked-by:** `disc-2026-10-library-list-triage-a`（同樣改 `MovieRepository.List`／`SeriesRepository.List` 的 ORDER BY 與 `listAll`；A 先合，本單把 A 加進去的 Go 端比較分支一起換掉，見「與 A 的分工」）。

**Source:** Rule 24 lane ③，2026-10-19 由 `disc-2026-10-library-list-triage-a` 建單時發現（見該檔 Discovery Triage）。⚖️ **Alexyu 2026-10-19：要修。**
原單描述：「全部」頁面 `listAll` 對每個 repo 要 `page × pageSize` 筆，但 repo 把 `PageSize` 壓到 100 → 任一類型超過 100 部時第 101 部之後漏片／順序錯；其他排序（評分等）沒有次序鍵，同分翻頁會重複／漏。

> 🔴 **Bob 建單時查到比原單更嚴重的事（已在正式機發生，不必等到 100 部）：「評分」排序今天就是壞的。** 見「查到的事」第 4、5 條。本單一起修（lane ①，AC #3）。

## 查到的事（2026-10-19，本 session 讀過的程式碼與正式機資料）

1. **頁大小上限 100**：`MaxPageSize = 100`（`apps/api/internal/repository/repository.go:46`），`Validate()` 把超過的壓回 100（`repository.go:68-70`）。`MovieRepository.List` 一進來就呼叫 `params.Validate()`（`movie_repository.go:333`），`SeriesRepository.List` 同款（`series_repository.go:326`）。
2. **「全部」頁的合併做法**：`listAll` 把兩邊都改成 `Page=1, PageSize=page×pageSize`（`library_service.go:380-382`），平行各拿一次（`:392-400`），合併後 `sort.Slice` + `compareLibraryItems`（`:423-425`，`sort.Slice` 不是穩定排序），再切 `[(page-1)×pageSize, page×pageSize)`（`:428-436`）；總數＝兩邊 `TotalResults` 相加（`:439-450`）。前端「全部」就是不帶 `type`（`apps/web/src/services/libraryService.ts:68`、`LibraryBrowseV2.tsx:157`），每頁 36（`hooks/useLibraryInfinite.ts:12`、`:20`），一直要到 `page < totalPages` 為止（`:22-23`）。
3. **超過 100 部的算術**（例：電影 150、影集 0，依標題排）：第 1 頁要 36、第 2 頁要 72、第 3 頁要 108 → 被壓成 100 → 合併只有 100 筆，切 `[72,108)` 只拿到 28 筆；第 4、5 頁切 `[108,144)`、`[144,180)` 都是空的，但 `totalPages = ceil(150/36) = 5`，前端照樣去要 → **第 101～150 部永遠不會出現**。兩邊都有片時更糟：合併後的前 100 名一定對，但第 101 名起會用另一邊的片「補位」，下一頁又因為重新合併而重複或跳過。正式機目前電影 55、影集 81（2026-10-19 `sqlite3 -readonly /mnt/cache/appdata/vido/vido.db` 查），還沒碰到這一條。
4. **「評分」排的是一個沒人寫的舊欄位**：前端「評分」送 `sort_by=rating`（`SortSelector.tsx:22`、`libraryService.ts:69`），handler 原樣傳下去（`handlers/movie_handler.go:343-345`），SQL 對照表把 `rating` 對到舊欄位 `rating`（`movie_repository.go:343`、`series_repository.go:337`）。但片庫畫面顯示的是 `vote_average`（`LibraryListRowV2.tsx:114-117`），而且掃描／比對路徑只寫 `vote_average`、不寫 `rating`（`enrichment_service.go:991`、`:1187`、`:486`，`enrichment_manual_match.go:503`，`media_ingest_service.go:195`；只有 `converters.go:50`、`:160`、`:264`、`:326` 兩個都寫）。**正式機 2026-10-19 實測：電影 55 部 `rating` 全是 NULL、影集 81 部也全是 NULL；`vote_average` 有值的電影 45、影集 80。**
   - 所以「電影」或「影集」分頁選「評分」= `ORDER BY NULL` = 沒有排序。正式機 `GET /api/v1/library?type=movie&sort_by=rating&sort_order=desc&page_size=10` 回：7.6、7.3、8.1、6.0、（無）、7.8、7.0、8.5、7.7、7.7。
   - 「全部」頁的 Go 合併是用 `vote_average` 比（`library_service.go:524-530`、`getVoteAverage` `:576-584`）——所以只有「拿進來的那幾筆」之間有排，拿哪幾筆是亂的。正式機實測（36／頁）：第 1 頁最高 9.17，**但片庫裡最高是 9.5**；第 2 頁又從 8.2 開始，**跟第 1 頁重複**。
5. **沒有次序鍵**：兩個 repo 都只有 `ORDER BY %s %s LIMIT ? OFFSET ?`（`movie_repository.go:380-386`、`series_repository.go:374-380`）。同值的列在不同 `LIMIT` 的兩次查詢之間不保證同一順序，翻頁會重複／漏。正式機有：電影 4 組同分 `vote_average`、1 組同一個 `created_at`、3 組同名電影、1 組同名影集。
6. **時間欄位比對是安全的**：`created_at` 存成固定寬度 `2006-01-02 15:04:05.000000000Z`（`database/dbtime/dbtime.go:13`、`:19-20`；正式機 136 筆長度都是 30），所以文字排序＝時間排序。
7. **其他吃 `List` 的地方**也受「沒有次序鍵」影響、會順便被修好：匯出（`export_service.go:424-460`，依 `created_at` 每頁 100 翻完全部）、官方字幕探勘（`subtitle/miner/miner.go:274`，依 `title` 每頁要 200、實際被壓成 100 再翻頁——頁數用壓過的值算，所以不漏頁，但同名影集跨頁可能重複／漏）、`movie_service.go:143`、`FullTextSearch` 的退回路徑（`movie_repository.go:424`、`:432`）。
8. **篩選條件已經是共用函式**：`movieListFilterConditions(params, alias)`（`movie_repository.go:625-683`）、`seriesListFilterConditions`（`series_repository.go:649`）；`notRemoved`／`notRemovedSeries`（`movie_repository.go:612`、`series_repository.go:626`）。`search`（標題 LIKE）在 `List` 裡另外加（`movie_repository.go:357-360`、`series_repository.go:351-354`）。
9. **沒有批次依 id 取片的方法**：兩個 repo 只有 `FindByID`（`movie_repository.go:104`、`series_repository.go:105`）。欄位清單 `movieSelectColumns`（`movie_repository.go:585-596`）、`seriesSelectColumns`（`series_repository.go:609`，含影集字幕彙總子查詢，用的是表名 `series`）；掃描函式 `scanMovie`（`:686`）、`scanSeries`（`series_repository.go:714`）。
10. **既有測試把錯的行為寫死了**：`TestMovieListWithSorting`（`movie_repository_test.go:641-698`）與 `TestSeriesListWithSorting`（`series_repository_test.go:552-609`）只種舊欄位 `Rating`、斷言 `result[0].Rating == 9.0`——跟正式機的資料形狀相反。
11. `NewLibraryService` 有 11 個呼叫點（`cmd/api/main.go:1070`、`handlers/route_c_uuid_integration_test.go:97`、`handlers/library_chinese_subtitle_integration_test.go:42`、其餘在 `services/library_service_test.go`）；`main.go` 用 `repos.Movies`／`repos.Series`（`repository/registry.go:11-12`、`:40-43`）。

## 做法比較（⚖️ SM 裁定：B）

| | 做法 | 對不對 | 代價 |
|---|---|---|---|
| A | 內部呼叫解除 100 上限，照舊「兩邊各拿 page×pageSize 再在 Go 合併」，另外把 Go 的比較函式改成跟 SQL 完全一致＋加次序鍵 | 只有在 **Go 與 SQL 兩份排序規則逐位元一致**時才對（NULL 放哪、浮點、字串比較、字幕排名）。今天的「評分」壞掉就是這兩份規則分岔的結果 | 改動最小；第 p 頁每邊要拿 p×36 筆，捲到底總共讀 O(p²) 筆；A 單又多一份要跟著同步的規則 |
| **B** | **一條 SQL（`UNION ALL`）先排出這一頁的 (類型, id)，再用 id 批次取完整資料** | **只有一份排序規則（SQLite），全部頁與單一類型同一份 ORDER BY 產生器，構造上就不會分岔** | 新增一個查詢、兩個 `FindByIDs`；每頁固定 1 次排序查詢＋2 次計數＋最多 2 次取資料，跟第幾頁無關 |
| C | 每個來源各記自己的位移（keyset／cursor） | 對 | 要改 API 合約（現在是頁碼制），前端無限捲動也要改 |

**選 B。** 理由：(1) 正確性不靠「兩份規則要一樣」——那正是這個 bug 的成因；(2) 刪掉 `compareLibraryItems` 之後，以後加新排序只要改一個地方；(3) 符合 Alexyu 2026-08-26「架構層面不要短解」（`.claude/memory/feedback_architecture_prefer_long_solutions.md`）。不選 A 的原因寫在上表；不選 C 是因為要動合約，而 B 已經足夠。

## 與 A 的分工（`disc-2026-10-library-list-triage-a`）

- A 先合：A 在兩個 repo 加了 `sort_by=subtitle_status` 的特例 ORDER BY（排名函式 `vido_subtitle_triage_rank`＋`created_at DESC, id ASC`），並在 `compareLibraryItems` 加了 `subtitle_status` 分支（A 的 T2、T3）。
- 本單：把 A 的特例**搬進**共用的排序產生器（第 5 種鍵），並**刪掉** `compareLibraryItems`（含 A 加的分支）。A 的 AC #4、#5 測試保留、必須仍綠——它們變成本單的回歸測試。
- confirmed against [@contract-v1] (Story disc-2026-10-library-list-triage-a AC #1)：排名順序、同級 `created_at` 新的在前、再同 `id` 由小到大、`desc` 只反轉排名不反轉次序鍵——本單不改這份合約，只是把它變成所有排序共用的次序鍵規則。
- 如果 Alexyu 決定本單先做（例如想先修好「評分」）：本單就不碰 `subtitle_status`，A 的 T3 改成「在共用排序產生器加第 5 種鍵」，不要再加 Go 比較分支。

## Story

身為只有我一個人在用的 NAS 片庫主人，
我想要媒體庫不管選哪種排序、往下捲多遠，每部片都剛好出現一次、而且順序是對的，
這樣片庫長到幾百部之後我還能信任這個列表，選「評分」也真的看到最高分的在最前面。

## Acceptance Criteria

1. **超過 100 部也不漏、不重複（核心）。** Go 測試用真遷移鏈資料庫（Rule 28：`newMigratedLibraryDB` 或 `services` 的 `setupTestDB`）種 **130 部電影＋120 部影集**，刻意製造同值：`vote_average` 只用 5 種值＋一部分 NULL、`rating` 全部 NULL（正式機形狀）、標題有重複、`release_date`／`first_air_date` 只有 6 種、`created_at` 只有 10 種（用 `UPDATE` 直接設）、字幕狀態涵蓋 A 的 6 級排名；其中 3 部（2 電影、1 影集）設 `is_removed=1`。
   對 **每一種** `sort_by` ∈ {`created_at`, `title`, `release_date`, `rating`, `subtitle_status`} × `sort_order` ∈ {`asc`, `desc`} × `type` ∈ {all, movie, tv}，用 `page_size=36` 從第 1 頁翻到 `total_pages`，把每頁接起來：
   - 筆數 ＝ `total_items` ＝ 該類型未刪除的片數（movie 128、tv 119、all 247）；
   - 沒有任何 (類型, id) 重複；集合剛好等於種進去、未刪除的那些；
   - 順序符合「排序鍵（依方向）→ `created_at` 新的在前 → `id` 小的在前（→ all 再加類型）」，逐對檢查。字幕排名用 A 的 `models.SubtitleTriageRank` 當答案。
2. **跟頁大小無關。** 同一組資料，`page_size` 7、36、100 翻完接起來的序列**完全相同**（逐筆比對 (類型, id)）。
3. **「評分」照畫面上的分數排。** `sort_by=rating` 與 `sort_by=vote_average` 都排 `vote_average`（電影、影集、全部都一樣）；舊欄位 `rating` 不再參與排序。`desc` 時沒有評分的片排最後、`asc` 時排最前（SQLite 的 NULL 規則，跟今天 `vote_average` 排序的行為相同）。`TestMovieListWithSorting`／`TestSeriesListWithSorting` 改成種 `VoteAverage`、`Rating` 留 NULL，斷言改看 `VoteAverage`。
4. **所有排序都有次序鍵。** `MovieRepository.List`、`SeriesRepository.List` 與「全部」的查詢用**同一個**排序產生器：`ORDER BY <鍵> <方向>, created_at DESC, id ASC`；`sort_by=created_at` 時是 `created_at <方向>, id ASC`；不認得的 `sort_by` 照舊退回 `created_at`。匯出、字幕探勘、`/movies` 列表因此一起有穩定順序（不另寫測試，靠 AC #1 的 movie／tv 列）。
5. **「全部」頁改成一份排序。** `listAll` 改成：一條 `UNION ALL` 查詢（電影與影集各自套**同一組** WHERE：未刪除＋`search`＋`movieListFilterConditions`／`seriesListFilterConditions`）排出這一頁的 (類型, id)，再用 `FindByIDs` 取完整資料並照查詢回來的順序組回去。`compareLibraryItems` 與只有它在用的 `getTitle`／`getReleaseDate`／`getVoteAverage`／`getCreatedAt`（含 A 加的 `getTriageRank`）刪掉（刪前 `grep -rn` 全 `apps/api` 確認沒有別人用；本 session 查過只有 `library_service.go:424` 一處呼叫）。每頁的查詢次數固定，跟頁碼無關。
6. **沒動到的照舊。** 回應形狀不變（`items`／`page`／`page_size`／`total_items`／`total_pages`）；`total_items` 仍是兩邊相加；篩選結果不變——既有 `TestLibraryService_SubtitleStatusFilter_AllTypes`（`library_service_test.go:932-1016`）、`TestLibraryService_ListLibrary`（`:411-527`）、`TestLibraryService_GetRecentlyAdded`、`TestLibraryService_CombinedFilters`、兩個 handler 整合測試、A 的 AC #4／#5 測試全綠。取資料時如果某個 id 剛好被刪掉（兩次查詢之間），那一筆略過、不回錯。
7. **本機實測（Rule 28 的補充）。** 用 AC #1 那組 247 筆資料起本機 API（或寫成一個 `go test` 走 handler），`GET /api/v1/library?sort_by=rating&sort_order=desc&page_size=36` 翻完全部頁，Completion Notes 貼：總筆數、不重複筆數、第 1 頁前 5 個 `vote_average`、最後一頁最後 3 個。
8. `cd apps/api && go test ./... && go vet ./... && ~/go/bin/staticcheck ./...` 綠（`lint:all` 不含 staticcheck）。

## Tasks / Subtasks

- [ ] **T1 共用排序產生器（AC #3、#4）**
  - [ ] `repository/` 新增 `librarySortKey(kind, sortBy string) string`（kind＝`movie`／`series`）：`title`→`title`；`release_date`→`release_date`／`first_air_date`（影集另認 `first_air_date`）；`rating`、`vote_average`→`vote_average`；`subtitle_status`→ A 的排名運算式（電影用 `vido_chinese_subtitle(...)`、影集用 `seriesChineseSubtitleSQL("series")`）；`id`、`updated_at`→同名欄位；其餘→`created_at`。
  - [ ] `libraryOrderBy(kind, sortBy, order string) string` 產生完整 ORDER BY（含次序鍵規則）。
  - [ ] `MovieRepository.List`、`SeriesRepository.List` 拿掉各自的 `sortColumnMap` 與 A 的特例分支，改用它。
- [ ] **T2 「全部」的排序查詢（AC #1、#5）**
  - [ ] 新增 `repository/library_repository.go`：`LibraryRepository{db}`、`LibraryKey{Type, ID string}`、`ListKeys(ctx, params) ([]LibraryKey, *PaginationResult, error)`。查詢形狀：
    ```sql
    SELECT kind, id FROM (
      SELECT 'movie' AS kind, id, <movie 鍵> AS sort_key, created_at FROM movies WHERE <電影 WHERE>
      UNION ALL
      SELECT 'series', id, <series 鍵>, created_at FROM series WHERE <影集 WHERE>
    ) ORDER BY sort_key <方向>, created_at DESC, id ASC, kind ASC
    LIMIT ? OFFSET ?
    ```
    （SQLite 的複合查詢 ORDER BY 只能用結果欄名，所以包一層子查詢。）計數＝兩個 `COUNT(*)` 相加。
  - [ ] 把 `List` 裡組 WHERE 的那段抽成 `movieListWhere(params)`／`seriesListWhere(params)`（含 `notRemoved`、`search`、篩選），`List` 與 `ListKeys` 共用——不要複製一份。
  - [ ] `interfaces.go` 加 `LibraryRepositoryInterface`；`registry.go` 加 `Library` 欄位並在 `NewRepositories`／`NewRepositoriesWithCache` 建立。
- [ ] **T3 批次取資料（AC #5）**：`MovieRepository.FindByIDs(ctx, ids []string) ([]models.Movie, error)`、`SeriesRepository.FindByIDs`（`WHERE id IN (...)`，用 `movieSelectColumns`／`seriesSelectColumns`＋`scanMovie`／`scanSeries`；空清單直接回空）；兩個介面與 `testutil/mocks.go` 的 mock 補上。
- [ ] **T4 service（AC #5、#6）**：`NewLibraryService` 加第 4 個參數 `libraryRepo repository.LibraryRepositoryInterface`（**刻意用位置參數**：編譯器會逼 11 個呼叫點全部接線，不會有 nil 的「全部」頁）；`main.go:1070` 傳 `repos.Library`；測試裡有真 DB 的傳 `repository.NewLibraryRepository(db)`，用 mock 的傳 `nil`（那些測試不走 `listAll`）。`listAll` 改成 `ListKeys` → 依類型分組 `FindByIDs`（兩邊平行）→ 照 keys 順序組回；刪掉 `compareLibraryItems` 與 getter。
- [ ] **T5 測試（AC #1–#3、#6、#7）**
  - [ ] 新 `repository/library_repository_test.go`：AC #1 的資料產生器（放成測試小工具，`services` 測試也要用的話就各自寫一份小的，不要為測試開 export）＋ AC #1、#2 的表格測試（5 鍵 × 2 方向 × 3 類型）。
  - [ ] 改 `TestMovieListWithSorting`、`TestSeriesListWithSorting`（AC #3）。
  - [ ] `services/library_service_test.go` 加一個走 `ListLibrary(…, "all")` 的跨頁測試（>100 部電影、page_size 36），證明 service 這層組回去的順序與 `ListKeys` 一致、沒有重複。
  - [ ] AC #7 本機實測；跑 AC #8。

## Dev Notes

### 不要做的事

- 不要只把 `MaxPageSize` 調大或給內部呼叫開後門就收工（那是做法 A：仍然是兩份排序規則）。
- 不要保留任何 Go 端的排序比較（`sort.Slice`／`compareLibraryItems`）在「全部」路徑上——順序只能有 SQL 一個來源。
- 不要改 API 合約（參數、回應形狀、頁碼制）；不要動前端。
- 不要動 `SearchLibrary`（`library_service.go:218-294`）：搜尋是 FTS 相關度、而且兩邊各自分頁回傳（`Movies`／`Series` 各有自己的 pagination），是另一個合約，不在本單。
- 不要回填或刪掉舊欄位 `rating`（`converters.go` 還在寫、`json:"rating"` 還在回）；本單只是不再拿它排序。
- 不要改 A 的排名規則或 `SubtitleTriageRank`。

### 已知陷阱

- **WHERE 不能各寫一份**：`List` 和 `ListKeys` 的條件一旦分岔，「全部」的總數會跟單一類型對不起來。T2 抽函式就是為了這個。
- 影集的字幕彙總子查詢寫死表名 `series`（`series_repository.go:609`、`chinese_subtitle_sql.go`），`UNION` 的影集分支不要給 `series` 取別名，否則子查詢對不到外層。
- `FindByIDs` 回來的順序是資料庫的順序，**一定要照 keys 重新排**；測試要能抓到「忘了重排」（AC #1 的逐對檢查會抓到）。
- `IN (...)` 的參數上限：一頁最多 100 個 id，遠低於 SQLite 預設上限，不需分批。
- `rating`／`vote_average` 是 REAL、可能 NULL；`title` 是 TEXT。`UNION` 後的 `sort_key` 欄混型別沒關係——同一次查詢裡同一個鍵兩邊型別相同。
- A 的 `vido_subtitle_triage_rank` 是在 Go 註冊的 SQL 函式；`ListKeys` 用的是同一個 `*sql.DB` 連線池，函式一定在。
- `listAll` 第 3 頁以後就不會再「每邊拿 108 筆」——如果有測試偷偷依賴 over-fetch 的行為（例如 mock 的 `List` 期望 `PageSize=page×pageSize`），改成驗新的呼叫。

### 測試標準

- Go：`testify`（`require`／`assert`）、表格測試；資料庫一律走真遷移鏈（Rule 28），不要用 `movie_repository_test.go:16-28` 那個手寫 schema 的 `setupTestDB` 做新測試。
- AC #1 的測試失敗訊息要印出「哪個鍵、哪個方向、哪個類型、第幾頁、哪一筆重複或順序錯」，方便以後加排序時看懂。

### Project Structure Notes

- 新檔：`apps/api/internal/repository/library_repository.go`（＋`_test.go`）。改：`repository/movie_repository.go`、`series_repository.go`、`interfaces.go`、`registry.go`、`testutil/mocks.go`、`services/library_service.go`（＋測試）、`cmd/api/main.go`、兩個 handler 整合測試。沒有 migration、沒有新 endpoint。
- Rule 11：新介面放 `repository/interfaces.go`。

### Time-dependent visual coverage

- N/A — no wall-clock-reading components touched（純後端）。

### References

- [Source: apps/api/internal/services/library_service.go:325-461, 503-584]
- [Source: apps/api/internal/repository/repository.go:42-89]、[movie_repository.go:331-414, 585-683]、[series_repository.go:323-408, 609-649]
- [Source: apps/api/internal/database/dbtime/dbtime.go:13-20]
- [Source: apps/web/src/hooks/useLibraryInfinite.ts:12-23]、[components/library/SortSelector.tsx:18-23]
- [Source: _bmad-output/implementation-artifacts/disc-2026-10-library-list-triage-a.md — AC #1 [@contract-v1]、T2、T3、Discovery Triage]
- [Source: project-context.md Rule 11、Rule 20、Rule 24、Rule 28]
- 正式機資料（2026-10-19，`sqlite3 -readonly /mnt/cache/appdata/vido/vido.db` 與 `GET http://192.168.50.52:8088/api/v1/library`）：`rating` 全 NULL、`vote_average` 電影 45／55、影集 80／81；「全部／評分」第 1 頁最高 9.17、片庫最高 9.5、第 2 頁從 8.2 重來。

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

### Discovery Triage

- **① expand-scope-in-place** — 「評分」排序排的是沒人寫的舊欄位 `rating`（正式機全 NULL），單一類型完全沒排、「全部」頁漏掉最高分並跨頁重複 → AC #3 追蹤。
- **① expand-scope-in-place** — 匯出、字幕探勘（`miner.go:274` 依 `title`）也用同一個沒有次序鍵的 `List` → AC #4 一起修，不另開單。

### File List
