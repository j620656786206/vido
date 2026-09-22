# Story DSR.1b-a2：媒體庫「搜尋」也要吃篩選——搜尋框有字的時候，類型／年份／未匹配／字幕篩選不再被悄悄丟掉

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As the person who has 「未匹配」 or 「缺字幕」 filter pills lit and then types in the library search box,
I want 搜尋結果仍然只包含符合那些篩選的片,
so that 亮著的 pill 與網址上的參數說的是真話，而不是搜尋一開始就整個失效。

## Context

⚖️ **Alexyu 2026-09-22 裁定：`disc-2026-09-library-search-ignores-filters` 走 (a)——後端搜尋端點也吃篩選。** 這張是 `dsr-1b-a` 的 /ship 對抗式 CR（HIGH #1）催生的**純後端**小單；`dsr-1b-b`（手機抽屜＋字幕篩選前端接線）**依賴本張與 `-a` 都先合併**。

🚨 **今天的缺口（既有，不是 `-a` 造成）**：`useLibrary.ts:51` 有 `q` 就改走 `GET /library/search`；該 handler 只解析 `q`／`type`／分頁／排序（`library_handler.go:406-430`），兩個 repo 的 `FullTextSearch` WHERE 只有 `MATCH ?`＋`notRemoved`（`movie_repository.go:457-510`、`series_repository.go:450+`），**`params.Filters` 一個字都不讀**。前端 `libraryService.ts:85-99` 的 `searchLibrary()` 今天就已經在送 `unmatched=true`，被靜默丟掉。所以「未匹配 pill 亮著＋搜尋框有字」現在就是假篩選；`-b` 接上 `subtitle_status` 之後會把 `-a` 剛消滅的謊言搬到搜尋結果裡。

⛔ **本張不進**：任何前端（`searchLibrary()` 補送參數是 `-b` AC #2 的事）、`sort_by` 在 FTS 的行為（FTS 一律 `ORDER BY rank`，維持）、`FindBySubtitleStatus`、swagger、把三個 handler 的篩選解析全部統一（只抽 `/library` 與 `/library/search` 共用的那一份；`movie_handler`／`series_handler` 另一件事）。

### 🔴 建單時查到的事（main `e9ea9948`；行號皆為現況）

1. **`SearchLibrary` handler**（`library_handler.go:406-430`）：`q` 少於 2 字 400 `VALIDATION_REQUIRED_FIELD`；`type` 驗證與 `ListLibrary` 一字不差的複製；`parseListParams(c)` 之後**直接**呼叫 service。`ListLibrary`（`:41-140`）的篩選解析（`genres`／`year_min`／`year_max`／`unmatched`／`subtitle_status`）是它沒有的那一段——兩個 handler 已經各有一份 `type` 驗證，再抄一份篩選解析就是第三份分歧碼（`-a` CR LOW #8 點名過）。
2. **service**（`library_service.go:218-290`）：`SearchLibrary` 把同一份 `params` 平行丟給 `movieRepo.FullTextSearch`／`seriesRepo.FullTextSearch`，再合併成 `LibrarySearchResults{Results, Movies, Series, TotalCount}`。**service 不用改**——`Filters` 本來就跟著 `params` 進去，只是 repo 不讀。
3. **repo `FullTextSearch`**：movie `:457-510`、series `:450+`。`query == ""` 時直接 `return r.List(ctx, params)`（那條路徑已經吃篩選）；否則 count 與 list 兩段 SQL 都是 `JOIN movies_fts … WHERE movies_fts MATCH ? AND <notRemovedQualified("m")>`，欄位用別名 `m`／`s`（`movieSelectColumnsQualified("m")`、`notRemovedQualified("m")` 在 `:616-635`）。**`List` 的條件梯（`:349-395`）用的是不帶別名的欄名**——要共用就得讓條件生成函式吃一個 alias 參數（`""` → 不加前綴、`"m"` → `m.`）。
4. **`List` 條件梯的五個條件**：`search`（`title LIKE`，FTS 路徑**不需要**——`MATCH` 已經是搜尋）、`genres`（`genres LIKE '%"g"%'` 逐一 AND）、`year_min`／`year_max`（`substr(release_date,1,4)`／series 用 `first_air_date`）、`unmatched`（`(tmdb_id IS NULL OR tmdb_id = 0)`）、`subtitle_status IN (...)`（`-a`）。`args` 順序：條件參數在前、`LIMIT ? OFFSET ?` 在後；FTS 的第一個參數是 `MATCH ?` 的 query。
5. **既有測試位置**：handler `TestLibraryHandler_SearchLibrary`（`library_handler_test.go:563`）；repo `TestMovieFullTextSearch*`（`movie_repository_test.go:1440-1583, 1765`）、`TestFullTextSearchExcludesRemovedMovies`（`:2169`）、`TestSeriesFullTextSearch*`（`series_repository_test.go:1286-1420`）；service `TestLibraryService_SearchLibrary`（`library_service_test.go:207`）。`-a` 新加的 `TestMovieListSubtitleStatusFilter`（`:2290+`）是種資料的範本（`Create` 之後 `UpdateSubtitleStatus`；`not_searched` 那列刻意留在欄位 DEFAULT）。
6. **`-a` 的三個守則沿用**：`Filters["subtitle_status"]` 是 `[]string`；驗證只在 handler、用 `models.SubtitleStatus(v).IsValid()`；`year_min` 存 `string` 是歷史包袱、不要學也不要改。
7. **前端今天送的搜尋參數**（`libraryService.ts:85-99`）：`q`／`page`／`page_size`／`type`／`sort_by`／`sort_order`／`unmatched`——沒有 `genres`／`year_*`／`subtitle_status`。補送是 `-b` 的事；本張後端要能吃全部五個。

---

## Acceptance Criteria

1. **`GET /api/v1/library/search` 接受與 `GET /library` 同一組篩選 `[@contract-v1]`**：`genres`（CSV）、`year_min`／`year_max`（1888–2100、min ≤ max）、`unmatched=true`、`subtitle_status`（CSV、小寫、`IsValid`、去重）。**解析與驗證邏輯是同一份程式碼**：從 `ListLibrary` 抽出 `parseLibraryFilters(c *gin.Context, params *repository.ListParams) bool`（回傳 `false` 表示已寫 400 並 return），`ListLibrary` 與 `SearchLibrary` 都呼叫它。`type` 的驗證也一併抽成 `parseLibraryMediaType(c) (string, bool)`。錯誤碼、訊息、`Filters` 的 key 與型別**與 `/library` 逐字相同**（`-a` AC #1 的 `[@contract-v1]` 不 bump——它的形狀沒變，只是多了一個消費端點）。回應形狀 `LibrarySearchResults` **不變**。

2. **兩個 repo 的 `FullTextSearch` 套用同一條件梯。** 把 `List` 裡的條件生成抽成一個私有函式（每個 repo 各一個，例：`movieListFilterConditions(params ListParams, alias string) (conds []string, args []interface{})`），涵蓋 `genres`／`year_*`／`unmatched`／`subtitle_status`；`search`（`title LIKE`）**留在 `List` 呼叫端**（FTS 不要它）。`List` 改用這個函式（alias `""`），`FullTextSearch` 的 **count 與 list 兩段 SQL 都**接上（alias `"m"`／`"s"`，接在 `MATCH ? AND <notRemoved>` 之後）；`args` 順序：`query` → 條件參數 → `LIMIT`／`OFFSET`。`ORDER BY rank` 不變。`query == ""` 走 `List` 的捷徑不變。

3. **測試（紅／守，Rule 16）。**
   - handler `TestLibraryHandler_SearchLibrary` 新增：（紅）`?q=駭客&subtitle_status=not_found&unmatched=true&genres=科幻&year_min=1999` → `mock.MatchedBy` 斷言四個 `Filters` key 都在、型別正確（`[]string`／`bool`／`[]string`／`string`）；`?q=駭客&subtitle_status=bogus` → 400 且 service 沒被叫；`?q=駭客&year_min=2020&year_max=2000` → 400。（守）既有 `SearchLibrary` 子測試不改。另加一條**結構測試**：`ListLibrary` 與 `SearchLibrary` 對同一組壞參數回**相同**的 400 body（證明是同一份程式碼，不是兩份剛好一樣）。
   - repo：（紅）`TestMovieFullTextSearchAppliesFilters`——種 4 部標題都含同一個詞（例：「星際」）的片：`found`／`not_found`／`not_found`＋`tmdb_id=0`／`not_found`＋`is_removed`；`q=星際`＋`subtitle_status=["not_found"]` → 2 部、count 2；再加 `unmatched=true` → 1 部；`genres` 與 `year_min` 各一條；**沒有篩選時結果與改前相同**（守既有 FTS 測試）。`series_repository_test.go` 同形。⚠️ 4 部片的標題要能被 `ftsPrefixQuery` 搜到——先看 `TestMovieFullTextSearchByTitle` 怎麼種資料。
   - service `TestLibraryService_SearchLibrary` 新增：（紅）`type=all`＋`subtitle_status=["not_found"]`＋`q` → `Results` 只含 not_found 的電影與影集、`TotalCount`＝兩邊相加。
   - `List` 路徑的**全部既有測試**（`TestMovieList*`、`TestSeriesList*`、`TestListExcludesRemovedMovies`、`-a` 的 `*SubtitleStatusFilter`）一條不改、全綠——抽函式不能改變 `List` 的行為。
   - **mutation check**（拿掉 → 必須紅）：FTS count SQL 不接條件（count 與 rows 不一致）；FTS list SQL 不接條件；`args` 順序放錯（query 放在條件之後）；`SearchLibrary` 不呼叫 `parseLibraryFilters`；`List` 改回自己的條件梯但漏 `subtitle_status`（`-a` 測試紅）。結果寫進 Completion Notes。

4. **文件與交接。**
   - `disc-2026-09-library-search-ignores-filters` 的 sprint-status 條目補 ↪「⚖️ 裁定 (a)，由 dsr-1b-a2 吸收」（建單時已寫）。
   - `dsr-1b-b` AC #2 的「搜尋模式」條目改成：`searchLibrary()` 補送 `genres`／`year_min`／`year_max`／`subtitle_status`，並 `confirmed against [@contract-v1] (Story dsr-1b-a2 AC #1)`（建單時已改）。
   - handler 檔頭註解（`-a` 寫的「只有 `/library` 讀這個參數」）改成事實。

5. **CI 全綠**：`NX_DAEMON=false pnpm nx test api`（跑完刪 `apps/api/coverage/`）、`pnpm nx run api:lint`、`pnpm run format:check`。⛔ 絕不 `run_in_background` 跑測試。純後端、零視覺基準變動。gh 操作一律帶 `GH_TOKEN=$(gh auth token --user j620656786206)`。

## Tasks / Subtasks

- [x] **Task 1 — handler：抽 `parseLibraryMediaType`／`parseLibraryFilters`，`SearchLibrary` 接上（AC: #1, #3）**
  - [x] 先跑 `TestLibraryHandler_ListLibrary*`／`TestLibraryHandler_SearchLibrary` 既有全綠 → 抽函式 → 仍綠 → 新測試紅 → 綠
- [x] **Task 2 — 兩個 repo：抽條件梯函式（帶 alias），`List` 改用、`FullTextSearch` count＋list 接上（AC: #2, #3）**
  - [x] 先跑 `TestMovieList*`／`TestSeriesList*`／`*SubtitleStatusFilter`／`*FullTextSearch*` 既有全綠 → 抽函式 → 仍綠 → 新測試紅 → 綠
- [x] **Task 3 — service 搜尋測試、mutation check、註解與交接（AC: #3, #4, #5）**

### Review Follow-ups (AI)

<!-- /ship 對抗式 CR（2026-09-22，獨立 context，Opus）— 0 HIGH／5 MEDIUM／5 LOW；先驗證了三個高風險點為正確（List 等價、FTS 別名、args aliasing）。 -->

- [x] [AI-Review][MEDIUM] `SearchLibrary` 函式註解沒提五個篩選參數（AC #4 只做一半）→ 補一行「Supports the same filters as GET /library …」
- [x] [AI-Review][MEDIUM] 兩個 goroutine 現在共讀同一張 `params.Filters` map，未來一次 repo 端寫入就是 concurrent map write → `library_service.go` fan-out 上方註解釘死「read-only from here on」
- [x] [AI-Review][MEDIUM] FTS 測試沒碰 `year_max`、也沒有 page 2（`OFFSET` 綁在 filterArgs 之後是最容易錯的地方）→ 加 `year_max` 與「page 2＋篩選」兩條
- [x] [AI-Review][LOW] `append(append(...))` 巢狀寫法脆弱 → 改成 `make`＋三次明確 `append`（兩個 repo）
- [x] [AI-Review][LOW] 「byte-identical 400 body」測試宣稱證明 provenance，其實只證明 contract parity → 改名與訊息，註明 provenance 由 mutation M4 釘住
- [x] [AI-Review][MEDIUM] `/library/search` 從「壞篩選值忽略回 200」變成「400」，與 `/library` 對齊——有意識地接受 → 記在 Completion Notes

## Dev Notes

### 這張的重點

- **不是加功能，是把兩條路收成一條。** `/library` 與 `/library/search` 從此共用「篩選怎麼解析」與「篩選怎麼變 SQL」；以後加第六個篩選只改兩處。
- **`List` 的行為一個位元都不能變。** 抽函式的第一步是「抽出來、`List` 改用、既有測試全綠」，然後才碰 FTS。
- **FTS 的 `args` 順序是最容易錯的地方**：`MATCH ?` 的 query 一定是第一個參數。
- **service 不改。**

### 上游契約（Rule 20）

- 本張**定義** `GET /library/search?genres=&year_min=&year_max=&unmatched=&subtitle_status=` `[@contract-v1]`（AC #1）。消費者：`dsr-1b-b`（建單時已寫 ack）。
- 本張**消費** `dsr-1b-a` AC #1 `[@contract-v1]`（`subtitle_status` 的形狀）：`confirmed against [@contract-v1] (Story dsr-1b-a AC #1)`——同一份解析碼，形狀不變、不 bump。

### 建單裁定（2026-09-22）

1. ⚖️ **Alexyu：走 (a)，後端搜尋端點吃篩選**（而不是前端搜尋時停用 pill）。
2. ⚖️ SM：**抽共用函式而不是複製**——`-a` CR LOW #8 已指出三份解析碼分歧；本張至少把兩個 library 端點收成一份。`movie_handler`／`series_handler` 那兩份不動（範圍外）。
3. ⚖️ SM：**`sort_by` 在 FTS 維持 `ORDER BY rank`**——搜尋結果依相關度排是既有行為，與篩選無關。
4. ⚖️ SM：**排在 `-b` 之前**（`-b` 依賴本張的契約），純後端半天內可收。

### 不要做的事

- 不要碰 `apps/web`（`searchLibrary()` 補送參數是 `-b`）。
- 不要動 `sort_by`／`ORDER BY rank`、`ftsPrefixQuery`、`query == ""` 的捷徑。
- 不要把 `search`（`title LIKE`）條件帶進 FTS。
- 不要動 `movie_handler.go`／`series_handler.go` 的解析碼。
- 不要改 `-a` 的契約形狀（同一份程式碼、同一個 `[@contract-v1]`）。

### 已知陷阱

- **alias 前綴**：`List` 不帶別名、FTS 帶 `m.`／`s.`——條件函式吃 alias，`""` 時不要產生 `.genres`。
- **count 與 list 兩段 SQL 都要接**——只接一段會讓 `TotalResults` 與 rows 不一致（mutation 必須有這一刀）。
- **FTS 種資料**：標題要能被 prefix query 命中；照 `TestMovieFullTextSearchByTitle` 的寫法，FTS 觸發器會在 `Create` 時同步 `movies_fts`。
- **`UpdateSubtitleStatus` 會改 `updated_at`**——FTS 走 `rank` 排序不受影響；`List` 的預設 `created_at desc` 也不受影響。
- **行號以建單時為準**（2026-09-22，main `e9ea9948`）。

### Source tree

```
apps/api/internal/handlers/library_handler.go（+ library_handler_test.go）      ← Task 1
apps/api/internal/repository/movie_repository.go（+ _test）                     ← Task 2
apps/api/internal/repository/series_repository.go（+ _test）                    ← Task 2
apps/api/internal/services/library_service_test.go                              ← Task 3
```

### Cross-Stack Split Check

後端 task 3、前端 task **0** → 不觸發跨棧拆分。規模：與 `dsr-1b-a` 同級（三個檔案的邏輯＋四支測試檔）。

### Time-dependent visual coverage

- **N/A — no wall-clock-reading components touched.** 純後端。

### References

- [Source: `apps/api/internal/handlers/library_handler.go:41-140, 406-430`；`services/library_service.go:29-34, 218-290`；`repository/movie_repository.go:349-395, 457-510, 616-635`；`repository/series_repository.go:344-390, 450+, 643`]
- [Source: `handlers/library_handler_test.go:563+`；`repository/movie_repository_test.go:1440-1583, 1765, 2169, 2290+`；`repository/series_repository_test.go:1286-1420`；`services/library_service_test.go:207+`]
- [Source: `apps/web/src/services/libraryService.ts:85-99`；`hooks/useLibrary.ts:45-58`（今天的呼叫端，本張不改）]
- [Source: `sprint-status.yaml` → `disc-2026-09-library-search-ignores-filters`、`dsr-1b-a-library-subtitle-status-filter-backend`（done，PR #508）、`dsr-1b-b-library-mobile-sort-filter-sheet`；`dsr-1b-a-…md` Review Follow-ups（CR HIGH #1、LOW #8）]
- [Source: project-context.md#Rule 16／#Rule 20／#Rule 24；`.claude/memory/feedback_gh_token_explicit_for_pr_ops.md`]

## Dev Agent Record

### Agent Model Used

Claude Fable 5.1 — `claude-fable-5-1`（dev-story，Amelia，2026-09-22）

### Debug Log References

- RED：handler 新 4 條——第一條 `mock.On("SearchLibrary", …MatchedBy)` 不匹配即紅；repo 兩支各 4／3 條紅（篩選被忽略、回全部 3 列）；service 那支在 Task 2 之後寫、寫完即綠（合併路徑由 mutation M1／M6 證明會紅）。
- 抽函式的順序照 Task 描述：先抽、`List` 改用、既有 `TestMovieList*`／`TestSeriesList*`／`*SubtitleStatusFilter`／`*FullTextSearch*` 全綠，再接 FTS。
- `gofmt -l` 列出的 `repository.go`／`episode_repository_test.go`／`season_repository_test.go` 是**既有**未格式化檔案，本張未觸碰。
- `NX_DAEMON=false pnpm nx test api --skip-nx-cache` 全綠，跑完已刪 `apps/api/coverage`；`api:lint` 綠；`format:check` 綠。

### Completion Notes List

- Ultimate context engine analysis completed - comprehensive developer guide created（SM Bob，2026-09-22）
- **做了什麼（dev-story，2026-09-22）**
  - **handler（Task 1）**：抽出 `parseLibraryMediaType(c) (string, bool)` 與 `parseLibraryFilters(c, *ListParams) bool`（genres／year_min／year_max／unmatched／subtitle_status；壞值寫 400 回 false）；`ListLibrary` 與 `SearchLibrary` 都改用。`SearchLibrary` 的 `q` 檢查與回應形狀不變。`-a` 寫的「只有 `/library` 讀這個參數」註解改成事實。
  - **repo（Task 2）**：`movieListFilterConditions(params, alias)`／`seriesListFilterConditions(params, alias)`——五個條件裡的四個（`search` 留在 `List` 呼叫端），`alias` 決定要不要加 `m.`／`s.` 前綴；`List` 改用（alias `""`），`FullTextSearch` 的 **count 與 list 兩段 SQL** 都在 `MATCH ? AND <notRemoved>` 之後接上，`args` 順序＝query → 條件 → LIMIT／OFFSET。`ORDER BY rank`、`query == ""` 走 `List` 的捷徑、`ftsPrefixQuery` 都沒動。
  - **service（Task 3）**：不改程式碼；補真 DB 測試釘住「兩邊都篩、`TotalCount` 相加、DEFAULT 列可被 `not_searched` 撈到」。
- **測試（Rule 16：紅／守）**
  - 紅：`library_handler_test.go` `TestLibraryHandler_SearchLibrary` 新增 4 條（四個 Filters key 型別正確／`subtitle_status=bogus` 400 且 service 沒被叫／年份反轉 400／**結構測試：五組壞參數在 `/library` 與 `/library/search` 回逐位元相同的 400 body**）；`movie_repository_test.go` `TestMovieFullTextSearchAppliesFilters` 5 條（無篩選 3 列不變／`subtitle_status` 同時縮 rows 與 count／＋`unmatched`／`genres`／`year_min`；軟刪除列永不出現）；`series_repository_test.go` `TestSeriesFullTextSearchAppliesFilters` 4 條。
  - 守：`library_service_test.go` `TestLibraryService_SearchLibrary_AppliesFilters` 3 條；既有 `TestLibraryHandler_*`、`TestMovieList*`／`TestSeriesList*`／`*SubtitleStatusFilter`／`*FullTextSearch*`／`TestListExcludesRemovedMovies`／`TestFullTextSearchExcludesRemovedMovies`／`TestLibraryService_*` 一條未改、全綠。
- **Mutation check（每一刀拿掉 → 必須紅）：6／6 紅**：M1 FTS count 不接條件（count 與 rows 不一致）／M2 FTS list 不接條件／M3 `args` 順序錯（query 放最後）／M4 `SearchLibrary` 不呼叫 `parseLibraryFilters`／M5 helper 漏 `subtitle_status`（`-a` 的 `List` 測試紅）／M6 series count 沒綁 args。
- 🔗 AC Drift: NONE (checked: `FullTextSearch\|SearchLibrary\|library/search` across _bmad-output/implementation-artifacts/*.md — 命中 `dsr-1b-a` CR HIGH #1 與 `disc-2026-09-library-search-ignores-filters`（本張就是它們的下游）、`ux3-*` 搜尋相關 story 提到 `/library/search` 的回應形狀（`LibrarySearchResults`）——形狀未變、`ORDER BY rank` 未變，屬 REUSE 不是 DRIFT)
- 📎 Contract Stamps: FOUND (2 stamped ACs across 2 files — this story AC #1 `[@contract-v1]`（定義 `/library/search` 的篩選集合）；upstream `dsr-1b-a` AC #1 `[@contract-v1]` referenced, ack line present in Dev Notes（`confirmed against [@contract-v1] (Story dsr-1b-a AC #1)`——同一份解析碼、形狀未變、不 bump）。下游 `dsr-1b-b` 建單時已 ack 本張)
- 🎭 A11y Pre-Flight: N/A (100% backend — no apps/web/ files touched)
- 🎨 UX Verification: SKIPPED — no UI changes in this story
- Pre-existing fix: N/A（`nx test api` 全綠）
- **與故事字面的差異**：無。
- **/ship 對抗式 CR（2026-09-22，獨立 context，Opus；0 HIGH／5 MEDIUM／5 LOW；`go vet`／`staticcheck`／`-race` 皆乾淨）**：吸收 6 項（見 Review Follow-ups）；CR 後 `nx test api --skip-nx-cache` 全綠。**有意識接受的行為變更**：`/library/search` 對壞的篩選值從「忽略、回 200」變成「400」，與 `/library` 一致——前端今天不會送這些值，`-b` 組深連結一律用常數。**沒有照做的**：
  - MEDIUM #2「合併後搜尋模式仍只有 `unmatched` 是真的」——前端 `searchLibrary()` 補送 `genres`／`year_*`／`subtitle_status` 是 `-b` AC #2（已 ack 本張契約），PR 說明「沒做的事」明寫。
  - LOW：三處 `AssertNotCalled` 近乎 tautology（無害、保留）；`q` 長度用 byte 數（既有、CJK 一個字就過——要改需 Alexyu 裁定，未立案）；兩個 repo helper 只差日期欄名（story 指定的形狀，之後要收再抽 `dateColumn` 參數）。

### Discovery Triage

- **建單時的發現（SM Bob 2026-09-22）：**
  - ① `disc-2026-09-library-search-ignores-filters` → 本張 AC #1／#2 吸收（⚖️ Alexyu 裁定 (a)）。
  - ③ `movie_handler.go`／`series_handler.go` 各自一份篩選解析碼 → 未立案（既有；若之後要統一，用 `disc-2026-09-list-filter-parsing-three-copies`）。
- **dev-story 期間的發現：** N/A — no out-of-scope work discovered。

### File List

- `apps/api/internal/handlers/library_handler.go` — `parseLibraryMediaType`／`parseLibraryFilters` 抽出；`ListLibrary`／`SearchLibrary` 改用；imports `repository`
- `apps/api/internal/handlers/library_handler_test.go` — `TestLibraryHandler_SearchLibrary` 新增 4 條
- `apps/api/internal/repository/movie_repository.go` — `movieListFilterConditions(params, alias)`；`List` 改用；`FullTextSearch` count＋list 接上
- `apps/api/internal/repository/movie_repository_test.go` — `TestMovieFullTextSearchAppliesFilters`（5 條）
- `apps/api/internal/repository/series_repository.go` — `seriesListFilterConditions(params, alias)`；`List` 改用；`FullTextSearch` count＋list 接上
- `apps/api/internal/repository/series_repository_test.go` — `TestSeriesFullTextSearchAppliesFilters`（4 條）
- `apps/api/internal/services/library_service_test.go` — `TestLibraryService_SearchLibrary_AppliesFilters`（3 條）
- `_bmad-output/implementation-artifacts/sprint-status.yaml` — 狀態 ready-for-dev → in-progress → review
- `_bmad-output/implementation-artifacts/dsr-1b-a2-library-search-applies-filters.md` — 本檔

## Change Log

- 2026-09-22 — 建單（SM Bob；main `e9ea9948`）。由 `dsr-1b-a` /ship CR HIGH #1 → `disc-2026-09-library-search-ignores-filters` → ⚖️ Alexyu 裁定 (a) 催生。純後端，排在 `dsr-1b-b` 之前。
- 2026-09-22 — Task 1（dev-story，Amelia）：handler 抽兩個共用 parser，`SearchLibrary` 接上（紅 4 → 綠）。
- 2026-09-22 — Task 2：兩個 repo 抽 alias-aware 條件梯，`List` 改用後既有測試全綠，再接 FTS count＋list（紅 9 → 綠）。
- 2026-09-22 — Task 3：service 搜尋測試 3 條；mutation 6／6 紅；`nx test api`／`api:lint`／`format:check` 全綠；Status → review。
- 2026-09-22 — /ship 對抗式 CR：吸收 6 項（`SearchLibrary` 註解、`Filters` read-only 註解、`year_max`＋page 2 FTS 測試、明確 `listArgs`、parity 測試改名、400 行為變更記錄）；0 HIGH。
- 2026-09-22 — 合併：PR #511（squash `2c7a37df`）；PR 上 CI 13 pass／0 fail；純後端、無基準線變動、無 bootstrap PR。Status → done。
