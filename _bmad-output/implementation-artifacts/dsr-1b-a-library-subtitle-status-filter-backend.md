# Story DSR.1b-a：媒體庫清單 API 可以用「字幕狀態」篩選——「查看未找到項目」那條連結終於會真的篩

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As the person who just ran a batch subtitle search and pressed 「查看未找到項目」,
I want 媒體庫真的只列出「找不到字幕」的那些片,
so that 我不用在 1,284 部片裡自己一張一張找哪些還缺字幕。

## Context

`dsr-1b`（Flow A 手機）拆出來的**第一塊：純後端**。它同時收掉 `disc-2026-06-library-subtitle-status-filter`（2026-06-09 立案，P3，一直沒人接）——Alexyu 2026-09-22 指定與 dsr-1b 一起做。

🚨 **今天的產品在說謊。** `BatchSubtitleDialog.tsx:361` 的「查看未找到項目」導去 `/library?subtitleStatus=not_found`；`routes/library.tsx:49` 會接住這個參數並保留在網址上；然後**沒有任何人讀它**——`LibraryBrowseV2` 不傳、後端 `library_handler.go` 不解析。使用者按下去，看到的是**整個未篩選的媒體庫**，網址卻長得像有篩過。這條單子讓後端先能篩；前端接線在 `dsr-1b-b`。

⛔ **本張不進**：任何前端（`apps/web` 一個字都不改）、設計稿、`FindBySubtitleStatus`（那是批次抽軌用的，見 🔴 #4）、新的計數／facet 端點、swagger（媒體庫端點本來就不在 swagger 裡，見 🔴 #7）。

### 🔴 建單時查到的事（main `49cc74a3`；行號皆為現況）

1. **端點與參數解析都在 `apps/api/internal/handlers/library_handler.go`**：`GET /api/v1/library`（`:473-479` 註冊；`ListLibrary` `:41-110`）。既有篩選的解析方式：`type`（`:46-51`，不合法回 400 `VALIDATION_INVALID_FORMAT`）、`genres`（逗號切、trim、去空，`:55-67`）、`year_min`／`year_max`（`Atoi` 驗證 1888–2100，但**以原字串放進 `Filters`**，`:69-86`）、`unmatched`（`== "true"` 才放 `true`，`:92-94`）。`page`／`page_size`／`sort_by`／`sort_order` 的解析在**另一個檔** `movie_handler.go:305-329` 的 `parseListParams`。
2. **篩選容器是一個 `map[string]interface{}`**（`repository/repository.go:29-40` 的 `ListParams.Filters`），不是型別化的 struct。⚠️ `year_min` 放進去的是 `string`，repo 端用 `params.Filters["year_min"].(string)` 取——**型別斷言不合會靜默跳過篩選**（`ok == false`、不報錯）。新篩選要**兩邊約定同一個型別**，並用測試釘住。
3. **SQL 條件梯在兩個 repo 各一份**：`movie_repository.go:330-420`（`conditions := []string{notRemoved}` 之後依序 `search`／`genres`／`year_*`／`unmatched`，`unmatched` 在 `:377-379` 是 `(tmdb_id IS NULL OR tmdb_id = 0)`）；`series_repository.go:348-372` 同形（年份用 `first_air_date`）。**新條件加在這兩條梯子上**，其他都不用動。
4. **`FindBySubtitleStatus` 是陷阱**（`movie_repository.go:897-921`、`series_repository.go:894+`）：沒分頁、**沒有 `is_removed` 守門**、只給 `subtitle/batch.go:516-577` 用。拿它來做清單會讓已軟刪除的片復活。**不要用。**
5. **欄位早就有、索引早就有**：`migrations/018_add_subtitle_fields.go:18,25` 在 `movies`／`series` 各加了 `subtitle_status TEXT DEFAULT 'not_searched'`，`:32-33` 建了 `idx_movies_subtitle_status`／`idx_series_subtitle_status`。`series_repository.go:606-617` 的 `seriesSelectColumns` 已含 `subtitle_status`。
6. **合法值有 10 個，權威來源是 `models/movie.go:68-157`**：`type SubtitleStatus string`；`AllSubtitleStatuses()`（`:118-131`）＝`not_searched`／`searching`／`found`／`not_found`／`probing`／`extracting`／`translating`／`no_text_source`／`skipped`／`untranslated`；`IsValid()`（`:139`）。**驗證用 `models.SubtitleStatus(x).IsValid()`，不要在 handler 再抄一份字串清單**（sub-1-2 `[@contract-v2]` 加第 10 個值時，抄過的清單都會漏）。
7. **沒有 swagger 條目**：`docs/swagger.yaml` grep `library` 零命中，`ListLibrary` 沒有 `@Router` 註解。契約靠本張 AC #1 的 `[@contract-v1]` stamp，不靠 swagger。
8. **`listAll` 會 over-fetch 再合併**（`library_service.go:372-380`：兩個 repo 各抓 `page × pageSize` 再合）。新篩選在 repo 層各自套用，合併與總數算法不變——但**要有一條 `type=all` 的測試釘住 total**，不能只測單一 repo。
9. **既有測試的位置**：`handlers/library_handler_test.go` `TestLibraryHandler_ListLibrary_WithFilters`（`:746`；`unmatched filter passed to service` 在 `:852-867`，斷言用 `mock.MatchedBy(func(p repository.ListParams) bool {...})`）；`repository/movie_repository_test.go` `TestMovieListUnmatchedFilter`（`:1682`）、`TestListExcludesRemovedMovies`（`:2078`）；`series_repository_test.go` `TestSeriesListUnmatchedFilter`（`:1225`）；`services/library_service_test.go`（合併路徑）。新測試**放在這些旁邊、照同一個形狀**。

---

## Acceptance Criteria

1. **`GET /api/v1/library` 接受 `subtitle_status` 查詢參數 `[@contract-v1]`。**
   - 形狀：**逗號分隔的狀態清單**，例：`?subtitle_status=not_found`、`?subtitle_status=not_found,not_searched`。切法與 `genres` 相同（逗號切、trim、去空）。**值為小寫、大小寫敏感**（`NOT_FOUND` → 400；深連結一律從常數產生，不手打）；**重複值合併**（`not_found,not_found` ＝ 一個值），所以 IN 清單長度上限＝狀態總數；全空（`,,,`）＝沒有這個篩選。（契約補述於 /ship CR，2026-09-22）
   - 每個值都必須通過 `models.SubtitleStatus(v).IsValid()`；任一不合法 → **400 `VALIDATION_INVALID_FORMAT`**，訊息點名是哪個值（形式比照 `type` 那一段 `:46-51`）。空字串（`?subtitle_status=`）＝沒有這個篩選。
   - 放進 `params.Filters["subtitle_status"]` 的型別是 **`[]string`**（與 `genres` 相同；⛔ 不是原字串、不是 `[]models.SubtitleStatus`——repo 端用 `.([]string)` 取，型別對不上會靜默失效，🔴 #2）。
   - 與 `type`／`genres`／`year_*`／`unmatched`／排序／分頁**全部可以疊加**；`type=all` 時兩個 repo 都套用同一份條件，合併與 `total_items` 正確（🔴 #8）。
   - 回應形狀**不變**（`PaginatedResponse{Items, Page, PageSize, TotalItems, TotalPages}`）。

2. **兩個 repo 的 `List` 條件梯各加一條。** `movie_repository.go` 與 `series_repository.go`：`subtitle_status IN (?, ?, …)`，placeholder 數量隨清單長度；**沿用既有的 `notRemoved` 起頭**（軟刪除的片不能出現）；不動排序白名單、不動其他條件。⛔ 不呼叫、不修改 `FindBySubtitleStatus`（🔴 #4）。

3. **測試（紅／守，Rule 16）。**
   - `handlers/library_handler_test.go` `TestLibraryHandler_ListLibrary_WithFilters` 新增：（紅）`subtitle_status=not_found` → `Filters["subtitle_status"]` 是 `[]string{"not_found"}`；`not_found, not_searched`（含空白）→ 兩個值都在、已 trim；`subtitle_status=bogus` → 400 且 service **沒被呼叫**；`not_found,bogus` → 400；空字串 → `Filters` 沒有這個 key；與 `unmatched=true`＋`genres=動畫` 同時給時三個都在。
   - `repository/movie_repository_test.go`：（紅）`TestMovieListSubtitleStatusFilter`——種 4 部片：`found`／`not_found`／`not_searched`／`not_found` 但 `is_removed=1`；篩 `["not_found"]` 只回 1 部；篩 `["not_found","not_searched"]` 回 2 部；`total` 與 rows 一致；**軟刪除那部永遠不出現**（釘住 🔴 #4 的理由）。`series_repository_test.go` 同形 `TestSeriesListSubtitleStatusFilter`。
   - `services/library_service_test.go`：（紅）`type=all`＋`subtitle_status=not_found` 時兩個 repo 都收到同一份 `Filters["subtitle_status"]`，`TotalItems` ＝ 兩邊 total 相加。
   - （守）既有 `TestLibraryHandler_ListLibrary*`、`TestMovieList*`、`TestSeriesList*`、`TestListExcludesRemovedMovies` 一條不改、全綠。
   - **每一項修法做 mutation check**：把 `IN` 條件拿掉 → repo 測試紅；把 `IsValid` 驗證拿掉 → 400 測試紅；把 `Filters` 型別改成 `string` → repo 測試紅（靜默失效被抓到）。結果寫進 Completion Notes。

4. **文件與交接。**
   - `routes/library.tsx:17-22` 那段「not yet wired」的註解**本張不動**（前端不改），但本 story 的 Dev Notes 明寫：`dsr-1b-b` 接線時要把它刪掉。
   - `disc-2026-06-library-subtitle-status-filter` 的 sprint-status 條目補 ↪ 註記「後端由 dsr-1b-a AC #1 收，前端由 dsr-1b-b 收」（建單時已寫）。
   - `project-context.md` **不加規則**（沒有新的 gotcha；`Filters` 型別陷阱已存在於 `year_min`，本張只是照做）。

5. **CI 全綠**：`pnpm nx test api`（跑完刪 `apps/api/coverage/`）、`go vet`／`golangci-lint`（走 `pnpm run lint:all`）、`pnpm run format:check`。⛔ 絕不 `run_in_background` 跑測試；Nx 一律 `NX_DAEMON=false`。純後端、**沒有畫面改動**，視覺基準零變動、不會有 bootstrap PR。gh 操作一律帶 `GH_TOKEN=$(gh auth token --user j620656786206)`。

## Tasks / Subtasks

- [x] **Task 1 — handler：解析＋驗證 `subtitle_status`（AC: #1, #3）**
  - [x] 先寫 handler 測試（紅）→ 實作 → 綠
- [x] **Task 2 — 兩個 repo 的 `IN (...)` 條件＋repo 測試（AC: #2, #3）**
  - [x] 種資料時一定包含一部 `is_removed=1` 的 `not_found`
- [x] **Task 3 — service 合併路徑測試、mutation check、收尾（AC: #3, #4, #5）**

### Review Follow-ups (AI)

<!-- /ship 對抗式 CR（2026-09-22，獨立 context 的審查代理，Opus）— 1 HIGH／4 MEDIUM／4 LOW；Rule 24 ① 吸收的每一項都有測試＋mutation。 -->

- [x] [AI-Review][MEDIUM] CSV 沒去重也沒上限，`not_found` 重複 32767 次會撐爆 SQLite 變數上限 → 500 → 迴圈內 `seen` 去重（保序），IN 長度上限＝狀態總數；測試「重複合併成一個值」（M6 紅）
- [x] [AI-Review][MEDIUM] 測試裡沒有任何一列走「欄位 DEFAULT」路徑（`Create` 不寫 `subtitle_status`，真實資料庫多數列都是這樣）→ 三支測試的 `not_searched` 列改成不呼叫 `UpdateSubtitleStatus`，各加一條「DEFAULT 列被 `not_searched` 撈到」（M8 紅）
- [x] [AI-Review][MEDIUM] service 測試註解說守住 `listAll` 的 page×pageSize over-fetch，但只跑 page=1；CSV 那條只斷言 total → 加 `Page=2, PageSize=2` 子測試（斷言 Items 2、total 4、頁數 2、沒有漏進未篩選列）＋ CSV 那條補 `assert.Len`
- [x] [AI-Review][MEDIUM] 大小寫策略沒寫進契約 → ⚖️ 維持嚴格（小寫、大小寫敏感），AC #1 契約文字補述，加 `NOT_FOUND` → 400 測試
- [x] [AI-Review][LOW] 400 訊息把未限長的輸入原樣回填 → `truncateRunes(v, 64)`，測試 5000 字元的壞值回應 <600 bytes 且含 64 個 x＋`…`（M7 紅）
- [x] [AI-Review][LOW] repo 註解宣稱「handler 已驗證」但 `List` 是匯出方法、測試自己就繞過 handler → 改寫成事實（一律參數綁定、未知值只會比不中、驗證在 handler）
- [x] [AI-Review][LOW] 只有 `/library` 開了這個參數，`movie_handler`／`series_handler` 沒有 → 註解記明；不抽共用函式（三份解析碼分歧是既有狀況，另一件事）
- [x] [AI-Review][LOW] `ids` 這種泛用名字在 `repository` 測試套件易撞名；`AssertExpectations` 卡在函式中間 → 改 `movieIDs`；`AssertExpectations` 移到函式末尾

## Dev Notes

### 這張的重點

- **只有三個檔案會動到邏輯**：`library_handler.go`、`movie_repository.go`、`series_repository.go`。加上四支測試檔。
- **形狀照 `genres`，不照 `year_min`。** `genres` 是 `[]string` 走 LIKE；本張是 `[]string` 走 `IN`。`year_min` 的「驗證後放原字串」是歷史包袱，不要學。
- **驗證用 `IsValid()`。** 合法值清單只能有一個來源。
- **契約 stamp 在本張。** `dsr-1b-b` 的 Dev Notes 要寫 `confirmed against [@contract-v1] (Story dsr-1b-a AC #1)`。

### 上游契約（Rule 20）

- 本張**定義** `GET /library?subtitle_status=` `[@contract-v1]`（AC #1）。下游消費者：`dsr-1b-b`（建單時已寫成依賴）。既有的 `GET /library` 其他參數是 pre-Rule-20 的 implicit v0，本張不改它們的形狀。

### 建單裁定（2026-09-22，SM；Sally／Alexyu 可在 review 推翻）

1. ⚖️ **拆成三張**（依 `feedback_split_oversized_stories`）：`-a` 純後端（本張）→ `-b` 手機排序＋篩選抽屜＋字幕篩選前端接線（依賴本張）→ `-c` 手機四張畫面（空白／骨架／網格／E4-M）對齊。原本一張會同時是跨棧（後端 3＋前端 >3）又大於 `dsr-4b-1`。切分線：**後端 API／抽屜／頁面**，三張的檔案幾乎不重疊。
2. ⚖️ **多值（CSV）而不是單值。** 前端稿的字幕篩選是 pill 多選；而且 `not_found` 與 `no_text_source`「都算缺字幕」這類合併只有多值做得到。單值是多值的子集，沒有額外成本。
3. ⚖️ **不做計數 facet。** 稿上「套用篩選 · 128 部」的 128 由 `-b` 用既有的 `total_items` 顯示（先 fetch 再顯示），不另開端點。

### 不要做的事

- 不要碰 `apps/web`。
- 不要用或改 `FindBySubtitleStatus`。
- 不要在 handler 抄一份合法值清單。
- 不要把 `Filters["subtitle_status"]` 放成 `string`。
- 不要動 `parseListParams`（`movie_handler.go`）——它的「靜默忽略」是既有行為，另一件事。
- 不要加 swagger 註解（會變成「只有一個端點有 swagger」的半成品；要做是整批做）。

### 已知陷阱

- **`Filters` 的型別斷言靜默失效**（🔴 #2）——mutation check 必須包含「改型別」這一刀。
- **`listAll` 兩個 goroutine 各抓一份**（🔴 #8）——service 測試要斷言兩個 repo 都拿到條件。
- **SQLite 的 `IN` 要動態組 placeholder**——照 repo 內既有的組法（看 `genres` 那段怎麼把多值展開）。
- **行號以建單時為準**（2026-09-22，main `49cc74a3`）。

### Source tree

```
apps/api/internal/handlers/library_handler.go（+ library_handler_test.go）        ← Task 1
apps/api/internal/repository/movie_repository.go（+ _test）                       ← Task 2
apps/api/internal/repository/series_repository.go（+ _test）                      ← Task 2
apps/api/internal/services/library_service_test.go                                ← Task 3
```

### Cross-Stack Split Check

後端 task 3、前端 task **0** → 不觸發跨棧拆分（拆分已在傘狀層完成：`-a` 後端／`-b` `-c` 前端）。

### Time-dependent visual coverage

- **N/A — no wall-clock-reading components touched.** 純後端。

### References

- [Source: `apps/api/internal/handlers/library_handler.go:41-110, 473-479`；`movie_handler.go:305-329`；`repository/repository.go:29-55`；`repository/movie_repository.go:330-420, 897-940`；`repository/series_repository.go:348-372, 606-617, 894+`；`services/library_service.go:79-80, 321-380, 470`；`models/movie.go:68-157`；`database/migrations/018_add_subtitle_fields.go:18-33`；`subtitle/batch.go:516-577`]
- [Source: `apps/api/internal/handlers/library_handler_test.go:178, 746-867`；`repository/movie_repository_test.go:462, 1682, 2078`；`repository/series_repository_test.go:412, 1225`]
- [Source: `apps/web/src/routes/library.tsx:17-22, 49`；`components/subtitle/BatchSubtitleDialog.tsx:242, 361`（今天說謊的那條連結——本張不改、只在此記錄）]
- [Source: `sprint-status.yaml` → `disc-2026-06-library-subtitle-status-filter`（`:250`）、`disc-2026-09-library-has-filters-the-product-lacks`（`:1307`）、`dsr-1b-flow-a-mobile`（`:1302`）；`8-11-batch-subtitle-ui.md:183, 216`]
- [Source: project-context.md#Rule 16／#Rule 18／#Rule 20；`.claude/memory/feedback_split_oversized_stories.md`、`feedback_story_splitting_rule.md`、`feedback_gh_token_explicit_for_pr_ops.md`]

## Dev Agent Record

### Agent Model Used

Claude Fable 5.1 — `claude-fable-5-1`（dev-story，Amelia，2026-09-22）

### Debug Log References

- RED：handler 新 6 條——第一條 `mock.On` 不匹配即 panic，整組紅；repo 兩支各 2 條紅（篩選被忽略、回全部 3 列）；service 那支在 Task 2 之後寫，寫完即綠（它守的是合併路徑，紅由 mutation M1b 證明）。
- 既有 `gofmt -l internal/handlers/` 列出的 10 個未格式化檔案都是**既有**的，本張碰的 7 個檔案不在其中。
- `NX_DAEMON=false pnpm nx test api` 全綠（0 個 FAIL 套件），跑完已 `rm -rf apps/api/coverage`；`api:lint` 綠；`format:check` 綠。

### Completion Notes List

- Ultimate context engine analysis completed - comprehensive developer guide created（SM Bob，2026-09-22）
- **做了什麼（dev-story，2026-09-22）**
  - **handler（Task 1）**：`ListLibrary` 在 `unmatched` 之後解析 `subtitle_status`——逗號切、trim、去空；每個值過 `models.SubtitleStatus(v).IsValid()`，不合法回 400 `VALIDATION_INVALID_FORMAT`，訊息 `subtitle_status contains unknown value "bogus"`；存進 `Filters["subtitle_status"]` 的是 `[]string`。空字串＝沒有 key。註解寫明「不要在這裡再抄一份合法值清單」與 `year_min` 的型別陷阱。
  - **repo（Task 2）**：`movie_repository.List`／`series_repository.List` 的條件梯在 `unmatched` 後各加 `subtitle_status IN (?, …)`（placeholder 隨長度動態組、`args` 逐一綁），沿用 `notRemoved`／`notRemovedSeries` 起頭。沒有碰 `FindBySubtitleStatus`。
  - **service（Task 3）**：不改程式碼（`listAll` 本來就把同一份 `Filters` 丟給兩個 repo）；補一支真 DB 測試釘住「兩邊都篩、total 是相加」。
- **測試（Rule 16：紅／守）**
  - 紅：`library_handler_test.go` `TestLibraryHandler_ListLibrary_WithFilters` 新增 6 條（單值 `[]string`／CSV 切＋trim＋保序／未知值 400 且 service 沒被叫、訊息點名值／CSV 混一個壞值整包 400／空字串沒有 key／與 `type`＋`unmatched`＋`genres` 疊加）；`movie_repository_test.go` `TestMovieListSubtitleStatusFilter` 3 條（單值只回 1 列且軟刪除那列不出現、count＝rows／兩值走 IN 回 2／`Filters` 給錯型別＝no-op 不 panic）；`series_repository_test.go` `TestSeriesListSubtitleStatusFilter` 2 條。
  - 守：`library_service_test.go` `TestLibraryService_SubtitleStatusFilter_AllTypes` 3 條（`all`：2 電影＋1 影集、total 3、兩種 type 都有／CSV 放寬到 4／`movie` 只回 `found` 那一部）；既有 `TestLibraryHandler_ListLibrary*`、`TestMovieList*`、`TestSeriesList*`、`TestListExcludesRemovedMovies`、`TestLibraryService_*` 一條未改、全綠。
- **Mutation check（每一刀拿掉 → 必須紅）：6／6 紅**
  - M1 movie repo 拿掉 `IN` 條件 → repo 3 條紅；M1b 同一刀下 service 那支 3 條紅（證明合併路徑真的靠它）。
  - M2 series repo 拿掉 `IN` → repo 2 條紅。
  - M3 handler 拿掉 `IsValid` → 「未知值 400」紅。
  - M4 handler 把 `Filters` 存成 `string` → 「單值 `[]string`」紅（型別陷阱被抓到）。
  - M5 movie repo 組了 placeholder 但忘了綁 `args` → repo 3 條紅（SQLite 綁定數不合）。
- 🔗 AC Drift: NONE (checked: `subtitle_status\|subtitleStatus\|查看未找到項目` across _bmad-output/implementation-artifacts/*.md — 相關命中在 `8-11-batch-subtitle-ui.md` AC #6／Discovery Triage ③ 與 `disc-2026-06-library-subtitle-status-filter`：8-11 定的是**網址形狀** `?subtitleStatus=not_found`，本張定的是**後端查詢參數** `subtitle_status=<csv>`，前者仍成立、後者是它一直在等的下游，屬 REUSE 不是 DRIFT；`GET /library` 其他參數的形狀與行為一個都沒改)
- 📎 Contract Stamps: FOUND (1 stamped AC in this story — AC #1 `[@contract-v1]` `GET /library?subtitle_status=<csv>`，本張是**定義端**、無上游 stamp 可 ack；下游消費者 `dsr-1b-b` 建單時已寫 `confirmed against [@contract-v1] (Story dsr-1b-a AC #1)`。本張沒有 bump)
- 🎭 A11y Pre-Flight: N/A (100% backend — no apps/web/ files touched)
- 🎨 UX Verification: SKIPPED — no UI changes in this story
- Pre-existing fix: N/A（`nx test api` 全綠、無既有紅燈）
- **與故事字面的差異**：無。AC #4 的 `disc-2026-06` ↪ 註記建單時已寫；`routes/library.tsx:17-22` 依 AC #4 未動（留給 `-b`）。
- **/ship 對抗式 CR（2026-09-22，獨立 context，Opus；1 HIGH／4 MEDIUM／4 LOW）**：吸收 8 項（見 Review Follow-ups），mutation 再加 3 刀（M6 去重、M7 截斷、M8 DEFAULT 列）全紅，合計 **9／9 紅**；CR 後 `nx test api --skip-nx-cache` 全綠、`api:lint` 綠、`format:check` 綠。**沒有照做的**：
  - #1 HIGH **`GET /library/search` 忽略所有 `Filters`**（`FullTextSearch` 的 WHERE 只有 `MATCH`＋`notRemoved`；前端 `searchLibrary()` 今天就已經在送 `unmatched=true` 而被靜默丟掉）——這是既有缺口、與本張的三個檔案無關，但 `dsr-1b-b` 接線後「搜尋框有字＋字幕篩選亮著」會把本張消滅的謊言搬到搜尋結果裡。→ ③ `disc-2026-09-library-search-ignores-filters`（P2），並在 `dsr-1b-b` AC #2 加一條「搜尋模式下的誠實行為」待裁定（要 (a) 讓 search 端點也吃篩選、或 (b) 搜尋時把篩選 pill 停用並從網址移除）。
  - #2 的另一半：`Update` 會把零值 model 的 `subtitle_status` 寫成 `''`，`''` 對任何 IN 都不成立 → 那些列會從所有字幕篩選消失。是否用 `COALESCE(NULLIF(subtitle_status,''),'not_searched')` 兜底是資料一致性決定，非本張範圍 → 併入同一張 disc 的 ↪ 補記（機率低：現有寫入路徑都走 `UpdateSubtitleStatus`）。
  - #8 的抽共用函式：不做（見上）。

### Discovery Triage

- **建單時的發現（SM Bob 2026-09-22）：**
  - ① `disc-2026-06-library-subtitle-status-filter` 的後端半 → 本張 AC #1／#2 吸收（前端半 → `dsr-1b-b`）。
  - ③ `parseListParams` 對壞的 `page`／`sort_by` 靜默忽略、與同一個 handler 兩行後的年份 400 不一致 → 未立案（既有行為、非本張造成；若 CR 認為該立，用 `disc-2026-09-library-list-params-silent-ignore`）。
- **/ship CR 的發現（2026-09-22）：**
  - ③ `GET /library/search` 忽略所有 `Filters`（既有；`unmatched` 今天就被丟掉）→ `disc-2026-09-library-search-ignores-filters`；`dsr-1b-b` AC #2 加待裁定條目。
  - ③ `subtitle_status = ''` 的列對任何篩選都不成立（`Update` 零值寫入的理論路徑）→ 併入同一張 disc 的 ↪ 補記。
- **dev-story 期間的發現：** N/A — no out-of-scope work discovered。（`gofmt -l` 列出的 10 個既有未格式化檔案不在本張範圍，也非本張造成；若要收，屬 chore，不立案。）

### File List

- `apps/api/internal/handlers/library_handler.go` — 解析＋驗證 `subtitle_status`（`[]string`）、imports `fmt`／`models`、檔頭註解列出五個篩選
- `apps/api/internal/handlers/library_handler_test.go` — `TestLibraryHandler_ListLibrary_WithFilters` 新增 6 條
- `apps/api/internal/repository/movie_repository.go` — `List` 加 `subtitle_status IN (...)`
- `apps/api/internal/repository/movie_repository_test.go` — `TestMovieListSubtitleStatusFilter`（3 條）＋ `ids` helper
- `apps/api/internal/repository/series_repository.go` — `List` 加 `subtitle_status IN (...)`
- `apps/api/internal/repository/series_repository_test.go` — `TestSeriesListSubtitleStatusFilter`（2 條）
- `apps/api/internal/services/library_service_test.go` — `TestLibraryService_SubtitleStatusFilter_AllTypes`（3 條）
- `_bmad-output/implementation-artifacts/sprint-status.yaml` — 狀態 ready-for-dev → in-progress → review
- `_bmad-output/implementation-artifacts/dsr-1b-a-library-subtitle-status-filter-backend.md` — 本檔

## Change Log

- 2026-09-22 — 建單（SM Bob，create-story；main `49cc74a3`）。由 `dsr-1b-flow-a-mobile` 拆出的第一塊（純後端），同時吸收 `disc-2026-06-library-subtitle-status-filter` 的後端半。程式碼現況由唯讀稽核代理查證（handler／repo／models／migrations／tests 逐行）。
- 2026-09-22 — Task 1（dev-story，Amelia）：handler 解析＋驗證 `subtitle_status`（紅 6 → 綠）。
- 2026-09-22 — Task 2：兩個 repo 的 `IN (...)` 條件（紅 5 → 綠）；軟刪除那列由測試釘住不出現。
- 2026-09-22 — Task 3：service 合併路徑測試 3 條；mutation 6／6 紅；`nx test api`／`api:lint`／`format:check` 全綠；Status → review。
- 2026-09-22 — /ship 對抗式 CR：吸收 8 項（CSV 去重、DEFAULT 列測試、page 2 over-fetch 測試、大小寫契約補述、400 訊息截斷、repo 註解、`movieIDs`、`AssertExpectations` 歸位），mutation 9／9 紅；另立 `disc-2026-09-library-search-ignores-filters`（HIGH，既有缺口，非本張範圍）。
- 2026-09-22 — 合併：PR #508（squash `c3bbae2b`）；PR 上 CI 13 pass／0 fail（Lint、Unit、Go、4 個 E2E shard、3 個 Build、Serve Smoke、Visual Regression PR gate），純後端沒有基準線變動、沒有 bootstrap PR；合併後 `main` 的 Tests／Docker 綠（Visual Regression 見 sprint-status 註記）。Status → done。
