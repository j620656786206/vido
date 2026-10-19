# Disc：修字幕清單 A（後端）——媒體庫可以「依字幕狀態」排序，缺中文的排最前面

Status: ready-for-dev

**Source:** 2026-10-19 `/impeccable critique 媒體庫 列表`（21／40；快照 `.impeccable/critique/2026-10-19T00-18-51Z__ps-web-src-components-library-librarylistrowv2-tsx.md`，Persona「Alex: no 缺中文-first sort」「NAS self-hoster: can't answer how many of 136 still lack 繁中」）＋ Alexyu 看正式機列表：「中間一大片空白，看起來很奇怪」。
⚖️ **2026-10-19 Alexyu 裁定 1A**：列表的工作是「修字幕清單」——字幕狀態緊跟片名、**可以依字幕狀態排序**、失敗列有出路。⚖️ **2B**：critique 的問題全部一起處理。設計定稿：`disc-2026-10-library-list-triage-design`（done；Sally 裁定 6「排序多一個『字幕狀態』：缺中文 → 無字幕源／未翻譯／已略過 → 不知道 → 簡中 → 中文 → 繁中」）。

⚖️ **SM 裁定（Bob，2026-10-19）— 拆成三張：** 原單 `disc-2026-10-library-list-triage` 前端 7＋項、後端 4 項，觸發 Story Splitting Rule（跨棧、兩邊都 >3）。拆成：
- **A（本單，後端）**：`sort_by=subtitle_status`。單獨上線就能用網址排序，不影響現有畫面。
- **B1（前端，版面）**：`…-b1` 表格列＋手機三行列＋失敗列＋規格灰字。不依賴 A。
- **B2（前端，頁首與工具列）**：`…-b2` 「N 部缺中文」連結、第五個排序選項、P3 小地方。依賴 A（排序）與 B1（骨架照表格畫）。
- 「N 部缺中文」**不需要後端**：沿用既有 `chinese_subtitle=missing` 篩選的 `totalItems`（見 B2）。所以 A 只剩排序。

## 查到的事（2026-10-19，本 session 讀過的程式碼）

- **後端沒有排序白名單在 handler**：`parseListParams` 把 `sort_by` 原樣放進 `params.SortBy`（`apps/api/internal/handlers/movie_handler.go:343-345`）；`ListLibrary` 呼叫 `parseListParams`（`library_handler.go:187-199`）。白名單在 repository：`MovieRepository.List` 的 `sortColumnMap`（`movie_repository.go:336-353`），認不得的值**默默退回 `created_at`**（`:348`）；`SeriesRepository.List` 同款（`series_repository.go:329-346`，`release_date` 是 `first_air_date` 的別名）。
- **ORDER BY 只有一個欄位，沒有次序鍵**：`ORDER BY %s %s LIMIT ? OFFSET ?`（`movie_repository.go:380-386`、`series_repository.go:374-380`）。字幕狀態只有 6 級，同級的列會非常多，沒有次序鍵的 OFFSET 分頁會重複／漏列 → 本單必須加次序鍵。
- **「全部」頁面是在 Go 裡合併排序的**：`listAll` 對兩個 repo 各拿 `page × pageSize` 筆（`library_service.go:380-382`），合併後用 `compareLibraryItems` 排（`:423-425`），再切片（`:428-436`）。`compareLibraryItems`（`:504-540`）只認 title／release_date／rating／vote_average，其餘走 created_at → **SQL 排了、Go 這裡沒跟上的話，「全部」頁面順序會錯**。
- **徽章用的判定已經有了**：`models.ChineseSubtitleVerdict(status, language, tracks)`（`models/chinese_subtitle.go:309-349`）是唯一實作，已註冊成 SQL 函式 `vido_chinese_subtitle`（`repository/chinese_subtitle_sql.go:23`、`:32-34`）；影集用 `seriesChineseSubtitleSQL(table)`＝有檔案的集裡「最需要處理」的那個判定，沒有集就用影集本身那列（`chinese_subtitle_sql.go:127-134`），已在 `seriesSelectColumns` 裡選成 `chinese_subtitle_rollup`（`series_repository.go:609`）。
- **前端徽章＝判定＋`subtitle_status`**（`apps/web/src/utils/libraryStatus.ts:139-194`）：
  1. 判定 `zh_hant`／`zh_hans`／`zh` → 繁中／簡中／中文（不管 status，`:144-156`）；
  2. 否則 status `no_text_source`／`skipped`／`untranslated` → 無字幕源／已略過／未翻譯（`:165-177`）；
  3. status `probing`／`extracting`／`translating` → **沒有徽章**（`:178-181`），B1 起列表會寫灰字「不知道」；
  4. 判定 `none` → 缺中文（`:190`）；
  5. 其餘 → 沒有徽章（不知道，`:193`）。
- 已有一個很接近的排名：`chineseSubtitleNeed`（`chinese_subtitle.go:372-384`，none 5 > unknown 4 > zh_hans 3 > zh 2 > zh_hant 1）——但它**只看判定、不看 status**，排不出「無字幕源／未翻譯／已略過」那一級，不能直接拿來用。
- `SubtitleStatus` 全部值：`not_searched`／`searching`／`found`／`not_found`／`probing`／`extracting`／`translating`／`no_text_source`／`skipped`／`untranslated`（`models/movie.go:74-111`）。
- 有現成的「真遷移鏈」測試資料庫與種子：`newMigratedLibraryDB`、`seedMovie`、`seedSeries`（`repository/chinese_subtitle_filter_test.go:26-35`、`:85-104`）、`seedEpisode`／`seedSee`（`series_chinese_rollup_test.go:27-55`）。

## Story

身為只有我一個人在用的 NAS 片庫主人，
我想要媒體庫能「依字幕狀態」排序，缺中文的排最前面，
這樣打開列表就是一張待修清單，不用一部一部點進去看。

## Acceptance Criteria

1. **AC #1 [@contract-v1]：** `GET /api/v1/library?sort_by=subtitle_status&sort_order=asc`（`type` 為 `all`／`movie`／`tv` 都一樣）依「字幕排名」由小到大排：
   `0 缺中文 → 1 無字幕源／未翻譯／已略過 → 2 不知道（含 probing／extracting／translating）→ 3 簡中 → 4 中文 → 5 繁中`。
   同一級：`created_at` 新的在前，再同就 `id` 由小到大。`sort_order=desc` 只反轉排名，次序鍵不反轉。回應形狀不變（`items`／`page`／`page_size`／`total_items`／`total_pages`）。
2. **排名＝列表上看到的那顆徽章。** 新增 `models.SubtitleTriageRank(verdict ChineseSubtitle, status SubtitleStatus) int`，規則逐條對應 `libraryStatus.ts:139-194`；表格測試列出「5 種判定 × 10 種 status＋空字串」全部組合與預期排名（不合法的判定字串當 `unknown`）。測試檔頂端註明前端那一半在 B2 的對表測試。
3. **影集用影集的判定。** 影集的排名＝`SubtitleTriageRank(seriesChineseSubtitleSQL 的結果, series.subtitle_status)`：《See》形狀（一集缺中文、其他集繁中）排在缺中文那一級，不是繁中。
4. **「全部」頁面跟單一類型同一個順序。** `listAll` 合併時 `compareLibraryItems` 認得 `subtitle_status`，用同一個排名與同一組次序鍵；電影＋影集混在一起、跨兩頁（`page_size=2`）測：第 1 頁＋第 2 頁接起來＝一次拿全部的順序，不重複、不漏。
5. **真資料形狀（Rule 28）。** 用 `newMigratedLibraryDB` 種設計稿那 5 部：執法記錄（英文軌、`not_found` → 缺中文）、鏢人（`und` 軌 → 不知道）、Chainsaw Man（`parse_status=failed`、沒有軌 → 不知道）、牧師的最後誘惑（`chi` 軌 → 中文）、阿凡達：火與燼（`zh-Hant` 軌 → 繁中），外加一部 `untranslated`。`asc` 的順序是：執法記錄 → untranslated 那部 → （鏢人、Chainsaw Man 依 created_at）→ 牧師 → 阿凡達。
6. **沒動到的照舊。** `title`／`release_date`／`rating`／`created_at` 與不認得的 `sort_by`（退回 `created_at`）結果與今天相同（既有 `TestMovieListWithSorting`、`library_service_test.go:484-530` 的 sort 子測試全綠）。
7. `go test ./...`、`go vet ./...`、`~/go/bin/staticcheck ./...` 綠。

## Tasks / Subtasks

- [ ] **T1 models（AC #2）**
  - [ ] `internal/models/chinese_subtitle.go` 加 `SubtitleTriageRank` 與六個具名常數（`SubtitleRankMissing = 0` … `SubtitleRankHant = 5`）；註解引用 `libraryStatus.ts:139-194` 並說明「排的是列表上看到的那顆徽章」。
  - [ ] 表格測試（`chinese_subtitle_test.go` 或同目錄新檔）。
- [ ] **T2 repository（AC #1、#3、#5）**
  - [ ] `chinese_subtitle_sql.go` 再註冊一個 deterministic 純量函式 `vido_subtitle_triage_rank(verdict, status)`，直接呼叫 `models.SubtitleTriageRank`（同 `vido_chinese_subtitle` 的做法：一份實作；同樣只准用在查詢裡，`:20-22` 的警告照抄）。壞輸入不回錯，當 `unknown`。
  - [ ] `MovieRepository.List`：`params.SortBy == "subtitle_status"` 時，ORDER BY 改成 `vido_subtitle_triage_rank(vido_chinese_subtitle(subtitle_status, subtitle_language, subtitle_tracks), subtitle_status) <ASC|DESC>, created_at DESC, id ASC`。
  - [ ] `SeriesRepository.List`：同上，判定用 `seriesChineseSubtitleSQL("series")`、status 用 `series.subtitle_status`。
- [ ] **T3 service（AC #4）**：`compareLibraryItems` 加 `case "subtitle_status"`：比 `SubtitleTriageRank(item 的 ChineseSubtitle, item 的 SubtitleStatus)`，同級比 `CreatedAt`（新的在前）再比 `ID`（小的在前）；加 `getTriageRank(item)` 小幫手，跟 `getCreatedAt` 等放一起。
- [ ] **T4 測試（AC #1、#3、#4、#5、#6、#7）**
  - [ ] repository 真遷移鏈測試：電影 5＋1 部的 asc／desc 順序；`page_size=2` 翻三頁接起來等於一次拿；影集 See 形狀。
  - [ ] service 測試：`listAll` 混合電影＋影集跨頁（照 `library_service_test.go:985` 那條「page 2 of 2 exercises listAll's over-fetch」的寫法）。
  - [ ] 跑 `go test ./...`、`go vet`、staticcheck。

## Dev Notes

### 不要做的事

- **不要把 `"subtitle_status": "subtitle_status"` 加進 `sortColumnMap`。** 那個欄位真的存在，SQL 會照字母排（`extracting` < `found` < `no_text_source` …）、看起來「有在排」但順序全錯。要用特例分支；測試資料要刻意讓「字母順序 ≠ 排名順序」，這樣誤用會紅。
- 不要改 `ChineseSubtitleVerdict`、`chineseSubtitleNeed`、`WorstChineseSubtitle` 或 `libraryStatus.ts`——⚖️ 2026-10-06 D2（中文算「有」）不動；排名只是讀它們。
- 不要動 handler：`sort_by` 本來就原樣傳到 repository（`movie_handler.go:343-345`），不需要新驗證。也不要動 `/library/search`（它用 FTS `ORDER BY rank`，`movie_repository.go:466`）。
- 不要動其他排序的次序鍵（見 Discovery Triage）。

### 實作提示

- SQLite 的 ORDER BY 可以重複寫影集那段相關子查詢；片庫 136～1,284 部，成本可忽略。⚠️ 未查證（dev 開工時先確認）：ORDER BY 能不能直接引用 SELECT 的別名 `chinese_subtitle_rollup` 放進函式參數裡——可以就用別名，不行就重複運算式。
- `created_at` 在 Go 是 `time.Time`、在 SQLite 是文字；兩邊都照「新的在前」，再用 `id` 收尾，確保 SQL 與 Go 合併順序一致。
- `listAll` 每邊最多只拿 100 筆（見 Discovery Triage ③），本單**不修**，測試資料量控制在 100 以下。

### 測試標準

- Go：`testify`（`require`／`assert`），表格測試；repository 測試一定要用 `newMigratedLibraryDB`（Rule 28），不要用 `setupTestDB` 那個手寫 schema（`movie_repository_test.go:16-28`）。
- 改動完跑：`cd apps/api && go test ./... && go vet ./... && ~/go/bin/staticcheck ./...`（`lint:all` 不含 staticcheck）。

### Project Structure Notes

- 只動 `apps/api/internal/models/`、`apps/api/internal/repository/`、`apps/api/internal/services/library_service.go` 與對應測試。沒有 migration、沒有新欄位、沒有新 endpoint。

### Time-dependent visual coverage

- N/A — no wall-clock-reading components touched（純後端）。

### References

- [Source: _bmad-output/implementation-artifacts/disc-2026-10-library-list-triage-design.md#Sally 的設計裁定 第 6 條]
- [Source: apps/web/src/utils/libraryStatus.ts:139-194]（徽章規則＝排名規則）
- [Source: apps/api/internal/models/chinese_subtitle.go:309-349, 372-384]
- [Source: apps/api/internal/repository/chinese_subtitle_sql.go:12-34, 121-134]
- [Source: apps/api/internal/repository/movie_repository.go:331-388]、[series_repository.go:323-382, 605-609]
- [Source: apps/api/internal/services/library_service.go:325-445, 504-540]
- [Source: project-context.md Rule 20（AC #1 有蓋 contract 戳，B2 要 ack）、Rule 24、Rule 28]

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

### Discovery Triage

- **③ backlog-with-carry-forward-link** — `disc-2026-10-library-all-view-past-100`（Bob 建單時發現）：`listAll` 對每個 repo 要 `page × pageSize` 筆（`library_service.go:382`），但 repo 的 `params.Validate()`（`movie_repository.go:334`）把 `PageSize` 壓到 `MaxPageSize = 100`（`repository/repository.go:46`、`:68-70`）。前端一頁 36 筆（`apps/web/src/hooks/useLibraryInfinite.ts:12`），第 3 頁起要 108 筆只拿到 100 → 任一類型超過 100 部時，「全部」頁面捲到第 101 部之後會漏片／順序錯。正式機目前電影 55、影集 81，還沒踩到。同一張單順便處理：其他排序（評分等）也沒有次序鍵，同分的列翻頁會重複／漏。

### File List
