# Disc：片子的「類型」存成簡體字（剧情、动作冒险），還有英文的 Sci-Fi & Fantasy

Status: review

**Source:** 2026-10-12 正式機驗證 `disc-2026-10-production-countries-never-filled`（PR #742）。開機補抓把 136 部片的類型補上了，但補上的全是簡體或英文，媒體庫「類型」篩選和列表的類型標籤都看得到。

## 查到的事（2026-10-12，Bob 逐一開檔／唯讀查 NAS／直接問 TMDb 確認）

### 正式機現況

- `GET http://192.168.50.52:8088/api/v1/library/genres` 回 20 個名稱，全是簡體或英文：`Sci-Fi & Fantasy`、`War & Politics`、冒险、剧情、动作、动作冒险、动画、喜剧、奇幻、家庭、恐怖、悬疑、惊悚、爱情、犯罪、电视电影、真人秀、科幻、纪录、西部。
- 唯讀查 DB（`json_each(genres)` 分組）：影集 剧情 58、动作冒险 33、Sci-Fi & Fantasy 25、动画 16、悬疑 10、喜剧 8、犯罪 7、War & Politics 4、真人秀 3、纪录 2、家庭 2；電影 冒险 19、动作 16、奇幻 13、惊悚 10、剧情 10、犯罪 9、科幻 7、恐怖 7、喜剧 7、悬疑 5、动画 3、爱情 2、家庭 2、西部 1、电视电影 1。
- `metadata_source`：電影 tmdb 44、nfo 1、NULL 10；影集 tmdb 81。也就是說，**正式機上每一個類型名稱都來自 TMDb**。

### 根因：TMDb 的「台灣中文」類型名稱本來就是簡體

2026-10-12 用正式機的 API key 直接打 TMDb `/3/genre/movie/list` 與 `/3/genre/tv/list`，語言用 zh-TW、zh-CN、zh-HK、en-US 各一次：

- **zh-TW、zh-CN、zh-HK 三種回的完全一樣，都是簡體**，例如 28 动作、18 剧情、10759 动作冒险。
- 有兩個 TV 類型在三種中文裡都只有英文：10765 `Sci-Fi & Fantasy`、10768 `War & Politics`。
- `/3/tv/106449?language=zh-TW`（凡人修仙傳）的 genres 也是 动画／动作冒险／剧情。

完整對照（id → TMDb 中文 → en-US）：12 冒险／Adventure、14 奇幻／Fantasy、16 动画／Animation、18 剧情／Drama、27 恐怖／Horror、28 动作／Action、35 喜剧／Comedy、36 历史／History、37 西部／Western、53 惊悚／Thriller、80 犯罪／Crime、99 纪录／Documentary、878 科幻／Science Fiction、9648 悬疑／Mystery、10402 音乐／Music、10749 爱情／Romance、10751 家庭／Family、10752 战争／War、10759 动作冒险／Action & Adventure、10762 儿童／Kids、10763 新闻／News、10764 真人秀／Reality、10765 Sci-Fi & Fantasy（中文也是英文）、10766 肥皂剧／Soap、10767 脱口秀／Talk、10768 War & Politics（中文也是英文）、10770 电视电影／TV Movie。

所以下面兩個地方的前提已經不成立：

- `apps/api/internal/tmdb/fallback.go:252-253` 的註解說「Genres come back populated in every language, so zh-TW genre names now always win」（bugfix-d，#244）。zh-TW 是會贏，但它本身就是簡體。
- `apps/web/src/lib/genres.ts` 的 `genreNamesFor` 註解說「These are the SAME strings the library stores (enrichment writes TMDb's zh-TW `genre.name`)」。實際上 library 存的是簡體，而修改資訊對話框（`MetadataEditorDialog.tsx:17` import `genreNamesFor`）提供的是繁體。所以使用者一編輯，同一部片會混著「劇情」和「剧情」，篩選清單也會兩個都列。

### 前端早就有一張正確的台灣用語表

`apps/web/src/lib/genres.ts` 的 `GENRE_MAP` 是 TMDb id 對到台灣用語，例如 28 動作、18 劇情、10759 動作冒險、10765 科幻奇幻、10768 戰爭政治。共 27 個 id，跟上面 TMDb 的 27 個一一對上。探索區塊（`ExploreBlocksSettings.tsx`、`ExploreBlockEditModal.tsx`）和編輯器都已經用這張表。**後端只要存成同一套名字。**

### 所有寫入都會經過同一個地方

- `models.Movie.GenresJSON()`（`apps/api/internal/models/movie.go:382`）和 `models.Series.GenresJSON()`（`apps/api/internal/models/series.go:153`）把 `Genres` 轉成 JSON 存進 DB。
- 呼叫它的地方（grep `GenresJSON()` 於 `apps/api/internal`，排除測試）：
  - `repository/movie_repository.go:39`、`:200`、`:834`
  - `repository/series_repository.go:39`、`:203`、`:867`
  - `repository/enriched_metadata_update.go:65`、`:132`
  - `services/context_facets_backfill_service.go:170`
- 也就是說，建立、更新、批次、比對寫入、開機補抓、手動編輯（經 repository Update），**全部都經過 `GenresJSON()`**。在這裡把名字換成台灣用語，就涵蓋了每一條寫入路徑。
- 設定 `Genres` 的來源很多（grep `.Genres = ` 與 `Genres:`）：
  - TMDb：`enrichment_service.go:1005`、`:1034`、`:1060`，`enrichment_manual_match.go:505`，`context_facets_backfill_service.go:169`
  - metadata provider 的 `item.Genres`：`enrichment_service.go:488`、`:1198`，`media_ingest_service.go:198`
  - 手動：`metadata_edit_service.go:113`、`:206`，`movie_handler.go:223`，`series_handler.go:222`
  - Douban／Wikipedia：`douban/scraper.go:243`、`metadata/wikipedia_provider.go:380`
- 改來源要改一大串，改出口只要一處。

### 限制：`models` 必須是 leaf

`apps/api/internal/boundaries_test.go:64` 規定 `models` 不能 import 任何 `internal/` 套件。所以台灣用語表要放在 `models` 套件裡面（或它的子目錄），不能放 `zhtw`（`zhtw` 會 import `ai/prompts`）。

### 讀取、篩選、翻譯快取

- 篩選清單：`MovieRepository.GetDistinctGenres`（`movie_repository.go:498`）把每列的 genres JSON 拆開去重，影集同理（`series_repository.go:492`）。修好存的值，清單就跟著變。
- 篩選本身：`genres LIKE ?`（`movie_repository.go:637`、`series_repository.go:661`，grep 命中）用的是名稱字串，所以舊列一定要一起轉，不然選「劇情」會找不到存成「剧情」的片。
- 翻譯快取鑰匙：`subtitle/segment_cache.go:104-113` 的 `MetadataHash` 含 `Genres`。改名後，那部片重新生成時快取不會命中一次。昨天補國家／類別時已經換過一次鑰匙，所以實際影響只有「昨天到部署之間新生成的片」。
- 列表也看得到：`LibraryTable.tsx:222` 顯示前三個類型標籤。

### 舊資料怎麼轉

用 migration（下一號是 046；`database/migrations/045_fix_reversed_harvested_glossary_terms.go:13` 是目前最後一號，grep `NewMigrationBase(46` 0 筆）。它把 `movies`、`series` 每一列的 genres 套同一個轉換函式，有變才寫，不動 `updated_at`。

## Story

身為 Vido 唯一的使用者（台灣），
我希望片子的「類型」一律是台灣用語的繁體字（劇情、動作冒險、科幻奇幻），
這樣媒體庫的類型篩選、列表標籤和修改資訊對話框講的是同一套字，不會出現簡體或英文，也不會「劇情」「剧情」各算一個。

## Acceptance Criteria

1. **一張表、一個函式：** `models` 套件新增 TMDb 類型名稱對照，把下面三種名字都對到 `GENRE_MAP` 的台灣用語：TMDb 中文（簡體）名、TMDb en-US 名（不分大小寫，例如 `drama`、`Sci-Fi & Fantasy`）、本來就是台灣用語的名字（對到自己）。共 27 個 TMDb id。函式 `CanonicalGenres(names []string) []string` 的規則：
   - 前後空白去掉。
   - 對得到就換成台灣用語；對不到的原樣保留，不丟掉。
   - 換完去重，保留第一次出現的順序。
   - 空字串丟掉；`nil` 回空切片。
2. **所有寫入都轉：** `Movie.GenresJSON()` 與 `Series.GenresJSON()` 先套 `CanonicalGenres`，再存。結果寫回 `m.Genres`／`s.Genres`，讓記憶體裡的 model 跟 DB 一致，這跟現有「nil 改成 `[]`」是同一種副作用。真 DB 測試要涵蓋：
   - `Create` 時給 `["剧情","Sci-Fi & Fantasy"]`，讀回來是 `["劇情","科幻奇幻"]`。
   - `FillContextFacets` 給 `["动画"]`，讀回來是 `["動畫"]`。
3. **舊資料轉好：** migration 046 把 `movies`、`series` 所有列的 genres 套 `CanonicalGenres`（含 `metadata_source='manual'` 的列）。
   - 只有內容真的變了才 UPDATE，不動 `updated_at`。
   - JSON 壞掉的列跳過並記 log，不能讓 migration 失敗。
   - 跑第二次不會再改任何列。
   - 測試：簡體列被轉、台灣用語列不動、英文列被轉、壞 JSON 列跳過且 migration 成功、`updated_at` 不變。
4. **前後端同一張表：** Go 測試讀 `apps/web/src/lib/genres.ts` 的 `GENRE_MAP`，斷言 Go 表的 27 個「id → 台灣用語」跟它完全一致。任何一邊改了，另一邊沒跟，測試就紅。
5. **註解改成現況：**
   - `tmdb/fallback.go:245-253` 補一句：TMDb 的 zh-TW 類型名稱本身就是簡體，台灣用語在存檔時由 `models.CanonicalGenres` 換。
   - `apps/web/src/lib/genres.ts` 的 `genreNamesFor` 註解改成：library 存的名字由後端統一換成這張表的台灣用語。
6. **檢查：** `go build／vet／test ./...`、`~/go/bin/staticcheck ./...`、`pnpm run lint:all` 全綠，`boundaries_test.go` 的 leaf 測試仍綠。
7. **正式機實測（部署後補）：**
   - (a) 部署前備份 DB。
   - (b) `GET /api/v1/library/genres` 只剩台灣用語：沒有簡體、沒有 `Sci-Fi & Fantasy`／`War & Politics`。
   - (c) 唯讀查 DB 兩次（隔幾分鐘）：`json_each(genres)` 分組沒有任何簡體名。凡人修仙傳是 `["動畫","動作冒險","劇情"]`。
   - (d) 頁面上點一個類型篩選（例：劇情），有結果。
8. **（dev 時併入，lane ①）舊連結也找得到：** 篩選條件裡的類型名稱也先套 `CanonicalGenres`（`movieListFilterConditions`／series 同名函式），所以舊書籤 `?genres=剧情` 或英文 `Drama` 會找到存成「劇情」的片。真 DB 測試：存 `剧情`、用 `剧情` 篩選，找到那一部。
9. **（CR 併入，lane ①）沒進片庫的片的詳細頁也是台灣用語：** 從探索點進一部還沒在片庫的片（`TMDbDetailV2`），以及舊的 `MediaDetailPanel`，類型直接顯示 TMDb 回的 `genre.name`（简体）。改用 `tmdbGenreName(genre)`：已知 id 讀 `GENRE_MAP`，不認得的 id 保留 TMDb 名稱。

## Tasks / Subtasks

- [x] T1 `apps/api/internal/models/genre_names.go`：27 個 id 的台灣用語＋別名表，以及 `CanonicalGenres`（AC #1）
  - [x] 表驅動測試：簡體、英文（大小寫）、台灣用語、未知名稱原樣、去重保序、空白與空字串、nil。
  - [x] AC #4 對表測試：從 `models` 套件目錄算出 repo 根目錄，讀 `apps/web/src/lib/genres.ts`，用 regexp 抓 `^\s*(\d+): '([^']+)'`。
- [x] T2 `Movie.GenresJSON`／`Series.GenresJSON` 套 `CanonicalGenres`（AC #2）
  - [x] 真 DB 測試放 `repository/context_facets_test.go` 或新檔：Create 讀回、FillContextFacets 讀回。
- [x] T3 migration `046_canonicalize_genre_names.go`＋測試（AC #3）
- [x] T4 改兩處註解（AC #5）
- [x] T5 篩選條件也套 `CanonicalGenres`（AC #8，dev 時併入）
- [x] T7 CR 修正：`tmdbGenreName`（AC #9）、`FillContextFacets` 自己也轉（CR L1）
- [ ] T6 正式機實測（AC #7，部署後）— ⏳ 刻意延後：要等合併、ghcr 映像更新、NAS 重建後才能量。

全部後端（T1–T3）＋一個前端註解。前端 task 0 個實質改動，不用拆單。

## Dev Notes

### 不要做的事

- **不要**改 TMDb client 或 fallback 鏈去換語言。三種中文都是簡體，換語言沒有用。
- **不要**在每個設定 `Genres` 的地方各轉一次，那有十幾處。只在 `GenresJSON()` 這個出口轉。
- **不要**把對照表放進 `zhtw` 或其他套件。`models` 是 leaf（`boundaries_test.go:64`）。
- **不要**對未知名稱跑 OpenCC。`models` 不能依賴轉換器；而且正式機上所有名字都來自 TMDb，27 個都在表裡。未知的原樣保留就好。
- 不改前端的顯示邏輯和 `GENRE_MAP`。前端那張表就是標準答案。
- 不改探索區塊。它存的是 id（`parseGenreIdList`），不受影響。

### 已知陷阱

- 英文名比對要不分大小寫：舊版編輯器存過小寫英文鍵（`git show 315339ab^:apps/web/src/components/metadata-editor/GenreSelector.tsx:24-41`：action、adventure、animation、comedy、crime、documentary、drama、family、fantasy、history、horror、music、mystery、romance、**sci-fi**、thriller、war、western）。除了 `sci-fi`（→ 科幻）以外都跟 TMDb en-US 名同字，`sci-fi` 要另外當別名。
- 去重很重要：一部片同時有「剧情」和「劇情」（手動編輯過）時，轉完只能剩一個「劇情」。
- `genres` 欄是 `NOT NULL DEFAULT '[]'`。migration 遇到 `''` 或 `'[]'` 直接跳過。
- migration 包在 runner 給的 `*sql.Tx` 裡（看 045 的 `Up(tx *sql.Tx)`）。先把所有列讀完、關掉 rows，再逐列 UPDATE，不要邊讀邊寫。
- 翻譯快取鑰匙會再換一次（見上）。寫進 Completion Notes。

### Time-dependent visual coverage

N/A — no wall-clock-reading components touched（後端＋一行前端註解）。

### References

- 根因：本檔「查到的事」各條的 `path:line`，以及 2026-10-12 對 TMDb genre list 的直接查詢。
- 前一張：`_bmad-output/implementation-artifacts/disc-2026-10-production-countries-never-filled.md`（「正式機驗證 2026-10-12」一節）。
- 舊修正：bugfix-d（#244）`tmdb/fallback.go` 的 per-field merge。

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（claude-opus-5-5），Amelia dev-story，2026-10-12。

### Debug Log References

- 紅：`TestGenresJSON_StoresTaiwanNames` 改 `GenresJSON` 前 `["动画","War \u0026 Politics"]`；`TestGenreWrites_StoreTaiwanNames` 拿掉篩選的 `CanonicalGenres` 時，`剧情` 篩選找不到片。
- 本機 Playwright：`movies.api`／`series.api`／`metadata-editor.api` 三支 53 passed、1 skipped（`AI_PROVIDER=claude`）。

### Completion Notes List

- Ultimate context engine analysis completed - comprehensive developer guide created
- 對照表 `models.GenreNamesByTMDbID`（27 個 id，跟前端 `GENRE_MAP` 一字不差，`TestGenreNames_MatchWebGenreMap` 讀 `genres.ts` 比對）＋別名（TMDb 中文名／en-US 名，不分大小寫；另加舊編輯器的 `sci-fi`）。TMDb 中文名跟台灣用語同字的（犯罪、家庭、奇幻、恐怖、西部、科幻、真人秀）不列別名。
- `Movie.GenresJSON`／`Series.GenresJSON` 先 `CanonicalGenres` 再存，並寫回 model，所以建立／更新的 API 回應也是台灣用語。
- migration 046：先讀完再寫、有變才 UPDATE、不動 `updated_at`、壞 JSON 跳過記 log，跑第二次 0 列。
- 舊測試裡用英文類型存、讀回比英文的 6 處改成台灣用語（`movie_test.go`、`series_test.go`、`movie_repository_test.go`、`series_repository_test.go`、`library_service_test.go` ×3），E2E `movies.api`／`series.api` 的 faker 類型也改成台灣用語（它們斷言回應等於送出的值）。
- ⚠️ 翻譯快取：metadata hash 含類型，部署後所有類型被改名的片，重新生成時快取會再失效一次（昨天補國家／類別已失效過一次，所以多出來的只有「昨天到部署之間新生成的片」）。
- 檢查：`go build／vet／test ./...`（42 個套件 ok）、`~/go/bin/staticcheck ./...`、`pnpm run lint:all`（0 errors）全綠。

### Senior Developer Review (AI) — 2026-10-12

結論：0 High／1 Medium／3 Low，在範圍內的全修。

- **M1（已修，AC #9）** 從探索點進還沒進片庫的片，詳細頁（`TMDbDetailV2.tsx:75`）的類型直接用 TMDb 回的 `g.name`，一樣是簡體。存檔的修正管不到這條，因為它不經過 DB。新增 `lib/genres.ts` `tmdbGenreName`，`TMDbDetailV2` 與 `MediaDetailPanel` 都改用它。測試：`genres.spec.ts` 兩條；`MediaDetailPanel.spec.tsx` 一條；路由 spec 的 TMDb 假資料改成 TMDb 真實樣子 `{ id: 28, name: '动作' }`，畫面要顯示「動作」。把修正拿掉，這條會紅（已驗）。
- **L1（已修）** `FillContextFacets` 收的是已經組好的 JSON，會繞過 `GenresJSON()`。現在它自己 parse 後套同一個轉換；不是 JSON 陣列就回錯。真 DB 測試：直接給 `["动画"]` 讀回 `["動畫"]`。
- **L2（接受）** migration 046 直接用現行的 `models.CanonicalGenres`，不是凍結一份副本。好處是跟寫入路徑不會分歧；代價是以後改表，「重跑舊 migration」的結果會跟著變。這個轉換是冪等的，接受。
- **L3（接受，範圍外、不立案）** `GetDistinctGenres` 用字碼排序，中文不是照筆畫。這是本來就有的行為，跟這張無關，也不是 bug。

### Discovery Triage

- ① 沒進片庫的片的詳細頁也顯示 TMDb 簡體類型 → 併入 AC #9／T7（CR M1）。
- ① 篩選條件也要轉（不然舊連結 `?genres=剧情` 找不到）→ 併入 AC #8／T5。
- 其他：N/A — no other out-of-scope work discovered.

### File List

- `apps/api/internal/models/genre_names.go`（新）
- `apps/api/internal/models/genre_names_test.go`（新）
- `apps/api/internal/models/movie.go`、`series.go`（`GenresJSON`）
- `apps/api/internal/models/movie_test.go`、`series_test.go`
- `apps/api/internal/database/migrations/046_canonicalize_genre_names.go`（新）
- `apps/api/internal/database/migrations/046_canonicalize_genre_names_test.go`（新）
- `apps/api/internal/repository/movie_repository.go`、`series_repository.go`（篩選）
- `apps/api/internal/repository/context_facets_test.go`、`movie_repository_test.go`、`series_repository_test.go`
- `apps/api/internal/services/library_service_test.go`
- `apps/api/internal/tmdb/fallback.go`（註解）
- `apps/web/src/lib/genres.ts`（註解＋`tmdbGenreName`）
- `apps/web/src/lib/genres.spec.ts`
- `apps/web/src/components/media/TMDbDetailV2.tsx`、`MediaDetailPanel.tsx`、`MediaDetailPanel.spec.tsx`
- `apps/web/src/routes/media/-$type.$id.spec.tsx`
- `apps/api/internal/repository/context_facets.go`（`FillContextFacets` 也轉）
- `tests/e2e/movies.api.spec.ts`、`tests/e2e/series.api.spec.ts`
- `_bmad-output/implementation-artifacts/disc-2026-10-genre-names-simplified.md`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`

## Change Log

| 日期 | 內容 |
|---|---|
| 2026-10-12 | Bob create-story（TMDb 三種中文類型名都是簡體；在 `GenresJSON()` 出口統一換成前端 `GENRE_MAP` 的台灣用語＋migration 046 轉舊列）。 |
| 2026-10-12 | Amelia dev-story：`GenresJSON` 出口統一換台灣用語＋migration 046＋篩選也轉（AC #8 併入）＋前後端對表測試；全套綠。T6 正式機實測待部署。 |
| 2026-10-12 | Code review 0H/1M/3L：M1（探索詳細頁的 TMDb 類型）、L1（FillContextFacets 自己也轉）已修；L2、L3 接受。前端 300 檔 4680 條、Go 全套、staticcheck、lint 綠。 |
