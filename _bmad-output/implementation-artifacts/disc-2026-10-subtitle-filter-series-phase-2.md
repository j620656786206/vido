# Story：影集「有沒有中文字幕」改看各集——媒體庫篩選、海報標籤、影集頁標頭一起變對

Status: done

**Source:** `disc-2026-10-subtitle-filter-series-phase-2`（P2）＋ `disc-2026-10-series-header-subtitle-badge-stale`（P2）兩張一起做（兩張都寫「共用同一個彙總，建議一起做」）。Alexyu 2026-10-08 從三張候選裡選這張。
**上游：** `disc-2026-10-subtitle-filter-disagrees-with-badges` AC #1／#2 `[@contract-v1]`（`chinese_subtitle` 欄位、`chinese_subtitle` 篩選參數）——本單 **confirmed against [@contract-v1]**：欄位與參數的形狀、值集合都不變，只改影集的值怎麼算。`disc-2026-10-episode-list-subtitle-badge-a`（`episodes.subtitle_tracks`，#717）提供每集資料。

> ⚖️ **SM 裁定（Bob 2026-10-08）：**
> 1. **影集的結論＝各集裡「最需要處理」的那個。** 只看有本地檔的集數。順序：缺中文 > 不知道 > 只有簡中 > 中文（繁簡未知）> 繁中。任一集缺 → 缺（上游 story「影集：分兩階段」寫的「任一集缺→缺」）；全部有中文才算「有」；有一集不知道就不能說「有」（上游裁定「能確定才說」）。
> 2. **沒有任何有檔集數的影集**，照舊用影集那一列自己的紀錄。
> 3. **算在讀取時，不存欄位**：在 SQL 裡把每集的 `vido_chinese_subtitle(...)` 交給新的聚合函式取「最需要處理的」，列表、搜尋、詳情、篩選全部走同一段——跟上游「不存欄位、不會過期」的做法一致。
> 4. **影集頁標頭不改字、不加「缺 N 集」**：標頭本來就讀 `chineseSubtitle`，值對了字就對了。「缺 1 集」這種新字樣要設計稿，另立條目給 Sally。
> 5. **不需要前端改動**（前端早就只讀 `chineseSubtitle`）；只補驗收測試。

## Story

身為 Vido 的使用者，
我希望媒體庫勾「缺中文字幕」時，缺了幾集中文的影集會出現；影集頁上方的字幕狀態也跟季清單每一集的圖示說同一件事，
這樣我才找得到哪些影集還要補字幕，不會 81 部影集全部是「不知道」。

## 查到的事（2026-10-08）

- 影集的 `chinese_subtitle` 現在由 `scanSeries` 用影集那一列算（`series_repository.go:765`）；影集那一列的 `subtitle_tracks` 從沒人寫，所以 81 部全落在「不知道」。
- 篩選：`seriesListFilterConditions` → `chineseSubtitleFilterCondition(groups, col)`，用 `vido_chinese_subtitle(影集欄位)`（`chinese_subtitle_sql.go:64-77`）。
- 每個影集讀取都經過 `seriesSelectColumns`＋`scanSeries`；搜尋用 `seriesSelectColumnsQualified("s")`（用逗號切欄位再加前綴）。`TestEverySeriesReadPathReturnsEveryColumn` 守著。
- `episodes.subtitle_tracks` 由背景補齊寫（#717）；`episodes(series_id)` 有索引（migration 006）。
- 前端：海報、清單列、影集頁標頭都走 `deriveSubtitleStatus(media)`，第一步就讀 `chineseSubtitle`（`libraryStatus.ts:144`）。

## Acceptance Criteria

1. **影集的 `chinese_subtitle`＝有檔集數的「最需要處理」結論**（裁定 1）；沒有有檔集數 → 影集那一列的結論（裁定 2）。所有讀取路徑都一樣（詳情、列表、搜尋、最近新增…），由同一段 SQL 算出、`scanSeries` 讀進來。
2. **篩選跟欄位逐筆一致**：`chinese_subtitle=has/missing/unknown` 對影集用同一個彙總；用 migration 鏈建的真實資料庫測：每組回來的影集 id＝欄位落在該組的影集 id，三組不重疊、聯集是全部。
3. **規則只有一份**：「哪個最需要處理」的順序寫在 `models`（Go），SQL 聚合函式呼叫它；不在 SQL 用 CASE 另寫一份。
4. **《末日光明》形狀的測試**：S1 第 1 集只有英文片內字幕、2–8 集旁邊有繁中、S2 片內繁中 → 整部 `none`（缺）；把 S1E1 也補上繁中 → `zh_hant`；某集 `subtitle_tracks` 還是 NULL → `unknown`；沒有任何有檔集數 → 照影集那一列。
5. **壞資料不讓列表 500**：聚合函式遇到怪值當「不知道」。
6. **前端驗收測試**：一部影集 `chineseSubtitle: 'none'` 在清單列／標頭顯示「缺中文」、`'zh_hant'` 顯示「繁中」（若既有測試已涵蓋就註明，不重寫）。
7. `go build／vet／test ./...`、`staticcheck`、`lint:all`（看「✖ N problems (0 errors」那行）全綠。

## Tasks / Subtasks

- [x] T1 `models.WorstChineseSubtitle`（順序）＋表格測試（AC #3）
- [x] T2 repository：註冊聚合函式；影集讀取欄位加彙總、`scanSeries` 用它；篩選條件改用同一段（AC #1、#2、#5）
- [x] T3 測試：真實資料庫篩選一致性＋《末日光明》形狀＋所有讀取路徑（AC #2、#4）
- [x] T4 前端驗收（AC #6）、閘門（AC #7）

## Dev Notes

- ⚠️ 聚合函式跟 `vido_chinese_subtitle` 一樣**只能用在查詢裡**，不能進索引／view／trigger（外部工具沒有這個函式）。
- `seriesSelectColumnsQualified` 是用逗號切欄位的，彙總那段（子查詢）不能直接塞進 `seriesSelectColumns`，要分開拼。
- 效能：81 部 × 約 30 集，每次列表約 2,400 次判斷（每次是一小段 JSON），可接受；有 `episodes(series_id)` 索引。

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（dev）；對抗式 CR 換模型（Sonnet 5.5）。

### Completion Notes List

- 2026-10-08 Bob（SM）：建單，合併兩張 backlog，五條裁定見上。
- 2026-10-08 Amelia（dev）：T1–T4 完成。
  - `models.WorstChineseSubtitle`：缺 > 不知道 > 簡中 > 中文 > 繁中；怪值當不知道。
  - SQLite 聚合函式 `vido_worst_chinese_subtitle`（呼叫上面那個 Go 函式）；`seriesChineseSubtitleSQL(table)`＝有檔集數的最差結論，沒有就用影集那一列。
  - `seriesSelectColumns` 改成「存的欄位＋這段彙總」，所有讀取（詳情、列表、搜尋、各種 Find…）都帶；`scanSeries` 讀它。篩選也用同一段。
  - 測試用的手寫資料表補 `episodes`（每次讀影集都會查它）。
  - 用正式機資料庫的副本（81 部、2,406 集，模擬背景讀完）量：整個影集列表 22ms、一頁 24 部 7ms。
  - 前端不用改（本來就只讀 `chineseSubtitle`）；`libraryStatus.spec.ts` 補影集形狀的驗收。
- 2026-10-08 對抗式 CR（Sonnet 5.5）：0H／2M／3L。
  - **M1（不修，立案）** 影片刪掉後集數那一列不會消失（`episodes` 沒有「已移除」），那一集的舊結論會一直算進整部 → `disc-2026-10-episode-rows-outlive-deleted-files`。
  - **M2（不修，照裁定）** 有檔集數存在時，影集那一列自己的「找到字幕」不再算數——影集層級的線上搜尋結果對應不到哪一集（裁定 1、2）。
  - **L5（已補測試）** 空字串檔案路徑＝沒有檔；有一集有檔就由各集決定。
  - L3（每條讀取路徑都算彙總，背景工作用不到）、L4（SQL 壞掉時的退路沒有 log）記錄不修：目前片庫大小下可以忽略。
  - 閘門：`go vet`、`go test ./...` 全綠、`staticcheck` 乾淨、`lint:all` 0 errors、prettier 綠、web `libraryStatus.spec` 綠；突變測試（篩選改回只看影集那一列）會讓一致性測試紅。

- 2026-10-08 /ship：PR #721。

### Discovery Triage

- **③** 影片刪掉後集數那一列不會消失 → `disc-2026-10-episode-rows-outlive-deleted-files`
- **③** 影集頁標頭要不要寫「缺 1 集」（新字樣，要設計稿）→ `disc-2026-10-series-header-missing-episode-count`

### File List

- apps/api/internal/models/chinese_subtitle.go ＋ _test.go
- apps/api/internal/repository/chinese_subtitle_sql.go、series_repository.go、series_repository_test.go、series_chinese_rollup_test.go（新）
- apps/web/src/utils/libraryStatus.spec.ts
