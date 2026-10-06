# Story: 媒體庫的「字幕」篩選跟海報標籤說同一件事：有沒有中文字幕（第一階段：電影）

Status: review

**Source:** `disc-2026-10-subtitle-filter-disagrees-with-badges`（P2，Alexyu 2026-10-06 看本機 POC 時發現；sprint-status.yaml:1163）。
**⚖️ 裁定（Alexyu 2026-10-06）：** 「缺字幕」＝**沒有中文字幕**。片內中文軌、影片旁邊的中文字幕檔、Vido 生成或下載的中文字幕，都算「有」；只有英文算「缺」。**標籤與篩選用同一套規則。** 影集要靠每集摘要（`disc-2026-10-episode-list-subtitle-badge`）。

## Story

身為 Vido 的使用者，
我希望在媒體庫勾「有中文字幕」或「缺中文字幕」時，出來的片跟海報上的標籤說的是同一件事，
這樣我才能一眼找出哪些片還需要處理，不會勾了「缺字幕」卻永遠是 0 部，或勾「有字幕」只出 3 部、海報卻掛了 40 張標籤。

## 查到的事（2026-10-06，Bob 逐一開檔確認）

**篩選跟標籤現在是兩套規則**
- **篩選看「Vido 上網找過沒」：** `movieListFilterConditions` 只做 `subtitle_status IN (...)`（`apps/api/internal/repository/movie_repository.go:659-666`）；影集同一段在 `series_repository.go:679-686`。`/library` 與 `/library/search` 共用這兩個函式（`movie_repository.go:617-622` 註解）。參數由 `parseLibraryFilters` 解析、驗證 `models.SubtitleStatus(v).IsValid()`（`apps/api/internal/handlers/library_handler.go:127-149`），契約是 `dsr-1b-a` AC #1 `[@contract-v1]`。
- **三顆篩選鈕：** `SUBTITLE_STATUS_FILTER_OPTIONS` = `found`→「有字幕」、`not_found`→「缺字幕」、`not_searched`→「還沒搜尋」（`apps/web/src/components/library/subtitleStatusFilter.ts:15-19`）；鈕的 `data-testid` 是 `filter-subtitle-${value}`（`FilterPanel.tsx:381`）。
- **標籤看「檔案裡有什麼」：** `deriveSubtitleStatus`（`apps/web/src/utils/libraryStatus.ts:149-218`）：先看 `found`＋語言（`:153-168`），再看 pipeline 結論（`:185-207`），最後看字幕軌（`:209-211`，`deriveFromTracks` `:128-139`），`not_found` 才說缺（`:214`）。
- **片內 `chi` 軌現在被標成「有字幕」：** 繁／簡集合只有 `zh-hant, zh-tw, zh, zh-hk`／`zh-hans, zh-cn`（`libraryStatus.ts:82-83`），`chi`／`zho` 兩邊都不在，所以落到「有軌就是有字幕」（`:137`）。只有英文軌的片也是「有字幕」（同一行）。這就是 Alexyu 看到的「約 40 張海報掛標籤」：37 部 `chi`＋3 部英文。
- **海報只顯示「例外」：** `pickPosterBadge` 把穩定狀態（已入庫＋繁中）藏起來（`libraryStatus.ts:220-232`）。同一個 `deriveSubtitleStatus` 也用在清單列（`LibraryListRowV2.tsx:50`）、詳情頁標頭（`LocalDetailV2.tsx:310`）、首頁大圖（`HeroBanner.tsx:93-95`，把「繁中」改寫成「繁中字幕 ✓ 已就緒」）。

**資料長什麼樣**
- **`subtitle_tracks` 是一個 JSON 陣列**，每軌 `{"language","format","external","title"(可省略),"stream_index"}`（`apps/api/internal/services/ffprobe_service.go:36-49`）。片內軌的語言是 ffprobe 的 `language` 標籤，沒有就 `und`（`:209-220`）。
- **旁邊的字幕檔也在裡面**：掃描時 `MergeSubtitleTracks(embedded, DetectExternalSubtitles(...))`（`enrichment_service.go:756-766`）。字幕檔的語言**照檔名**：`Movie.zh-TW.srt`→`zh-TW`、`Movie.srt`→`und`、`Movie.chi.forced.srt`→`chi.forced`（大小寫照原樣，`ffprobe_service.go:272-291`）；只認 `.srt/.ass/.ssa/.sub`（`:238-243`）。違反「語言看內容」規則（`project-context.md:177`），但 `services` 不能 import `subtitle`（`project-context.md:547-557`，Rule 19），所以掃描時沒辦法呼叫 `subtitle.Detect`。
- **`subtitle_tracks` 是 NULL 有兩種意思**：(1) 從沒讀過；(2) 讀過但片內、旁邊都沒有任何字幕——`writeSubs` 在空陣列時**不寫**（`enrichment_service.go:758-760`）。而且影片技術資訊已經有值時，重新整理會**整段跳過**（`:773-776`），所以舊的 NULL 不會自己變好。
- **NFO 路徑寫的不是 JSON**：`applyNFOTechInfo` 寫成逗號字串，例如 `chi,eng`（`enrichment_service.go:948-954`）；前端 `trackLangs` 遇到非 JSON 回 `null`＝不知道（`libraryStatus.ts:113-125`）。
- **影集層級的 `subtitle_tracks` 從來沒人寫**：整個 `apps/api` grep `SubtitleTracks *=\|SubtitleTracks:`（排除測試），只有電影的兩個寫入點（`enrichment_service.go:762`、`:953`）和 repository 的「保留舊值」（`series_repository.go:1115`）。所以影集的結論今天只能靠影集那一列的 `subtitle_status`／`subtitle_language`。
- **`subtitle_language` 會是什麼：** Vido 生成成功寫 `found`＋`zh-Hant`（`transcription_service.go:1287-1291`）；只做出英文寫 `untranslated`＋`en`（`:1278-1286`、`:1292-1296`）；抽片內字幕那條交付 `zh-Hant`（`apps/api/internal/subtitle/pipeline.go:726`）；線上下載寫 placer 正規化過的標籤（`subtitle/manager.go:40-45`），`zh-tw/cht`→`zh-Hant`、`zh-cn/chs`→`zh-Hans`、`zh` 保持 `zh`（繁簡不明）、其他照原樣（`subtitle/placer.go:133-156`）。
- **片內 `chi` 軌其實可以從軌道名稱猜繁簡**：`embeddedLanguage`（`apps/api/internal/subtitle/inventory.go:203-235`）看 `title` 裡的「繁／Traditional／簡／Simplified／粵／Cantonese」；ffprobe 從 PR #674 起才把 `title` 存進來（`ffprobe_service.go:41-43`、`:218`），之前掃的片沒有。
- **既有「哪些標籤算中文」的 Go 版本：** `isChineseTag`＝`chi, zho, zh, zh-hans, zh-hant, zh-cn, zh-tw, zh-hk, chs, cht`，刻意不含 `yue`（理由是「不要拿粵語當翻譯來源」，`apps/api/internal/subtitle/extractor.go:105-127`）。

**資料庫與既有工具**
- **SQLite 驅動是 modernc**（`apps/api/internal/database/database.go:9`），版本 v1.44.0（`apps/api/go.mod:21`）。它支援在 Go 裡註冊 SQL 函式：`RegisterDeterministicScalarFunction`，「之後開的連線都能用」（`~/go/pkg/mod/modernc.org/sqlite@v1.44.0/sqlite.go:442-460`）。app 包的 `sqlite-utc` 驅動包的是同一個已註冊的驅動，「帶著套件層級的自訂函式」（`internal/database/utc_driver.go:28-39`）。repository 的測試直接 `sql.Open("sqlite", ":memory:")`、只 import modernc、不經過 `internal/database`（`internal/repository/movie_repository_test.go:12`、`:16-17`）。`apps/api/internal` 非測試程式 grep `json_each\|json_extract\|json_valid` 零命中——今天沒有任何 SQL 在解析 JSON。
- **列表讀取全部走 `scanMovie`／`scanSeries`**（`movie_repository.go:576-582` 註解、`:672-736`；`series_repository.go:692`），`scanMovie` 已經有「讀完順便算出給前端的欄位」的前例：`ProductionCountries`（`:731-736`；欄位宣告 `models/movie.go:292-295` `db:"-"`）。
- **前端把回應的 key 轉成 camelCase，值不動**（`apps/web/src/utils/caseTransform.ts:6-17`）。
- **深連結：** 批次字幕的「查看未找到項目」導到 `/library?subtitleStatus=not_found`（`apps/web/src/components/subtitle/BatchSubtitleDialog.tsx:356-362`）；路由收 `subtitleStatus` 字串（`apps/web/src/routes/library.tsx:17-23`、`:50`）。
- **E2E 綁著舊的鈕與參數：** `tests/e2e/library-mobile.spec.ts` 用 `subtitle_status`（`:31-33`、`:171`、`:195`）、`移除缺字幕篩選`（`:176`、`:196`、`:250`、`:307`）、三個 `filter-subtitle-*` testid 與字（`:321-323`）；stub 依 `subtitle_status` 篩（`tests/support/helpers/library-stubs.ts:18`、`:29`）。

**NAS 實際（取自 sprint-status.yaml:1163，Alexyu session 2026-10-06 唯讀查詢；Bob 這次未重查——沙箱連不到 NAS）**
- 電影 55：37 部 `not_searched`＋片內中文軌、3 部只有英文軌、12 部 `subtitle_tracks` 是 NULL、3 部 `found`。影集 81：全部 `not_searched`、`subtitle_tracks` NULL。
- 結果：勾「有字幕」只出 3 部；勾「缺字幕」永遠 0 部（沒有一部是 `not_found`）。
- **已查證（2026-10-06，NAS 唯讀 `sqlite3 -readonly /mnt/cache/appdata/vido/vido.db`）：** `subtitle_tracks IS NULL` 的電影共 13 部（12 部 not_searched＋1 部 found）：**10 部 `parse_status=failed`、`video_codec IS NULL`**（比對失敗、從沒讀過檔案）；1 部 success 但沒讀過檔案；2 部讀過檔案（有 `video_codec`）但字幕軌是 NULL。所以它們落在「不知道」是對的——不是「讀過、真的沒字幕」。

## 設計

### 一條規則、一份程式碼

在 `apps/api/internal/models`（leaf 套件，誰都能 import，`project-context.md:559-560`）新增**唯一的**判斷函式：

`models.ChineseSubtitleVerdict(status, language, tracks string) ChineseSubtitle`，回傳五個值之一：
`zh_hant`（有繁中）／`zh_hans`（有簡中、沒有繁中）／`zh`（有中文、分不出繁簡）／`none`（確定沒有中文）／`unknown`（不知道）。

這個函式用在兩個地方，所以**不可能**說法不一：
1. **給畫面：** `scanMovie`／`scanSeries` 讀完一列就算出來，放進新欄位 `chinese_subtitle`（照 `ProductionCountries` 前例，`db:"-"`）。列表、搜尋、詳情頁全部經過這兩個函式，所以海報、清單列、詳情頁、首頁大圖拿到的是同一個值。前端**不再自己算**有沒有中文。
2. **給篩選：** 在 repository 套件把同一個函式註冊成 SQLite 函式 `vido_chinese_subtitle(subtitle_status, subtitle_language, subtitle_tracks)`，篩選條件寫成 `vido_chinese_subtitle(...) IN (...)`。

為什麼不用「新增一欄存起來」：要存就得在 6 個以上的寫入點（`movie_repository.go:229-239`、`:918`、`subtitle_generation_status.go:38`、`enriched_metadata_update.go:89`、影集對應的幾處）每次都重算，漏一個就變舊資料；還要回填。為什麼不用 SQL 的 `json_each` 自己寫一份：那是第二份規則，而且要處理 `chi.forced` 這種檔名、NFO 的逗號字串，SQL 寫不乾淨，遲早跟 Go 版不一樣。用 Go 函式註冊進 SQLite：一份程式碼、永遠是最新資料、不用 migration。代價是每次列表每列跑一次 Go 函式（幾百部片是毫秒級）。

### 判斷規則（照裁定）

標籤怎麼算「中文」：轉小寫、`_` 當 `-`、用 `.` 切開，**任一段**符合就算（處理 `zh-TW.forced`、`chi.sdh`）：
- 繁：`zh-hant`、`zh-tw`、`zh-hk`、`zh-mo`、`cht`、`zh-hant-tw`、`zh-hant-hk`
- 簡：`zh-hans`、`zh-cn`、`zh-sg`、`chs`、`zh-hans-cn`
- 中文但繁簡不明：`chi`、`zho`、`zh`
- **粵語不算中文（⚖️ Alexyu 2026-10-06）：** `yue`，以及 `chi`／`zho`／`zh` 軌但 `title` 判斷為粵語（粵／Cantonese／yue）的，一律當「已知的非中文」——跟 `eng` 同一類。
- 片內軌若標籤是「繁簡不明」但有 `title`，用與 `embeddedLanguage` **同一份**關鍵字判斷繁／簡（做法：把關鍵字判斷搬進 `models`，`subtitle.embeddedLanguage` 改成呼叫它，管理字幕對話框與本規則從此一致）。

依序判斷：
1. **有中文證據就是「有」**（任一成立）：`subtitle_status = found` 且 `subtitle_language` 是中文；或字幕軌（JSON 陣列，或 NFO 的逗號字串）裡任一軌是中文——片內或旁邊、文字或圖片格式都算（裁定：片內中文軌就算有）。繁簡：有任一繁中證據→`zh_hant`；否則有簡中→`zh_hans`；否則 `zh`。
2. **沒有中文證據時，能確定才說「缺」**（`none`）：
   - 字幕軌讀得出來，且每一軌的語言都是已知的非中文（例如只有 `eng`）——含空陣列 `[]`。
   - 或字幕軌是 NULL，但 `subtitle_status` 是 `not_found`（線上找過、沒有；沿用今天 `libraryStatus.ts:213-214` 的意思）或 `untranslated`（Vido 只做出英文；而 pipeline 遇到片內中文會優先用它，`extractor.go:133-140`，所以走到這裡代表片內沒有中文文字軌）。
3. **其餘都是「不知道」**（`unknown`）：字幕軌是 NULL 且不屬於上面兩種狀態；或有 `und`（沒標語言）的軌、又沒有任何中文軌——`Movie.srt` 這種沒標語言的檔在台灣使用者的片庫裡很可能就是中文，說「缺」是假話。

**⚖️ 已裁定（Alexyu 2026-10-06）：粵語字幕不算「有中文」。** 只有粵語（或粵語＋英文）字幕的片是 `none`（缺中文）。與 `extractor.go:105-127` 不拿 `yue` 當翻譯來源一致。

### 篩選（後端）

- 新查詢參數 `chinese_subtitle`（CSV）：`has`（＝`zh_hant`／`zh_hans`／`zh`）、`missing`（＝`none`）、`unknown`。`/library` 與 `/library/search` 都收（同一個 `parseLibraryFilters`）。不認得的值回 400，訊息格式照 `subtitle_status`。
- **舊的 `subtitle_status` 參數不動**（`dsr-1b-a` `[@contract-v1]` 不 bump），兩個參數同時給時是 AND。前端從此不再送它（見下），它留給 API 使用者。

### 畫面（前端）

- 篩選：「字幕」區改成新規則的鈕（字與顆數見 ⏸ D1、D2），URL 參數 `chineseSubtitle`，送 `chinese_subtitle`。鈕、篩選膠囊、「沒有結果」那句、手機抽屜的「套用篩選 · N 部」全部讀同一張表（沿用 `subtitleStatusFilter.ts` 的「一張表」做法，改名或新檔都可，但只能有一張）。
- 前端不再讀寫 `subtitleStatus` URL 參數。批次字幕的「查看未找到項目」改導到 `?chineseSubtitle=missing`（線上找不到＝沒有中文，見規則 2）。舊書籤 `?subtitleStatus=...` 會變成沒篩選——可以接受（深連結只有這一個產生點，`BatchSubtitleDialog.tsx:361`）。
- 標籤：`deriveSubtitleStatus` 改讀 `chineseSubtitle`：
  - `zh_hant`→繁中（穩定，海報不顯示）；`zh_hans`→簡中；`zh`→（字見 D2；算「有」，跟繁中一樣是穩定狀態、海報不顯示）。
  - 不是「有」時：`no_text_source`／`skipped`／`untranslated` 仍顯示「無字幕源／已略過／未翻譯」（它們說的是**為什麼缺**、下一步是什麼）；否則 `none`→（字見 D2，今天的「缺字幕」）；`unknown`→不顯示（照舊，`libraryStatus.ts:18-19`）。
  - 處理中三態（`probing`／`extracting`／`translating`）仍不顯示（`:203-206`）。
  - 這樣「有中文」那一顆裡的每一部，標籤都是繁中／簡中／中文；「缺中文」那一顆裡的每一部，標籤都是缺中文或無字幕源／已略過／未翻譯。**有／缺的分界與篩選逐筆相同**。
- `HANT`／`HANS`／`subtitleLangLabel`／`trackLangs` 保留——管理字幕對話框與詳情頁「字幕軌」那列還在用（`DetailTechInfoV2.tsx:15`、`ManageSubtitleDialogV2.tsx:76`）。

### 影集：分兩階段

- **第一階段（本單）：** 影集套同一個函式，但影集的 `subtitle_tracks` 從來沒值，所以 81 部幾乎全是 `unknown`；只有影集層級被線上下載過中文（`subtitle_handler.go:518` 寫 `found`）的才是「有」。畫面上影集標籤跟今天一樣（今天 `not_searched`＋沒有軌也是不顯示）。
- **第二階段（新單 `disc-2026-10-subtitle-filter-series-phase-2`）：** 影集的結論改成各集彙總（例如任一集缺→缺）。**前置：** `disc-2026-10-episode-list-subtitle-badge` 的每集摘要。⚠️ 那張單的設計是「打開季清單時才讀旁邊字幕檔、不存」（它的 AC #2），而媒體庫篩選要在 SQL 裡跨 81 部影集篩，**每集的旁邊字幕檔結果也必須存進資料庫**（或在掃描時存）。這條需求已寫進那張 story 的 Dev Notes。也跟 `disc-2026-10-series-header-subtitle-badge-stale`（詳情頁影集標頭）共用同一個彙總，建議一起做。

## ⚖️ 裁定（Alexyu 2026-10-06：「D1 三顆、D2 中文、粵語不算」）

- **D1 → 三顆：**「有中文字幕」「缺中文字幕」「不知道」。三顆加起來＝全部。
- **D2 → 明講中文：** 只有英文（或確定沒字幕、或只有粵語）的片標「缺中文」；片內中文分不出繁簡標「中文」（算有，海報不顯示，清單列與詳情頁顯示；首頁大圖「中文字幕 ✓ 已就緒」）；鈕寫「有中文字幕／缺中文字幕／不知道」。
- **粵語不算中文**（見「判斷規則」）。

## Acceptance Criteria

1. **AC #1 `[@contract-v1]`：電影與影集的 JSON 多一個 `chinese_subtitle` 欄位**，值是 `zh_hant`／`zh_hans`／`zh`／`none`／`unknown` 之一，**永遠出現**（不 omitempty）。所有讀取路徑都有（列表、搜尋、詳情、首頁最近新增）——由 `scanMovie`／`scanSeries` 填，不得在其他地方另算。
2. **AC #2 `[@contract-v1]`：`GET /api/v1/library` 與 `GET /api/v1/library/search` 接受 `chinese_subtitle=<csv>`**，值 `has`／`missing`／`unknown`，小寫、去重；不認得的值回 400 `VALIDATION_INVALID_FORMAT`，訊息 `chinese_subtitle contains unknown value "<v>"`。與 `genres`／`year_*`／`unmatched`／`subtitle_status` 同時給時全部 AND。解析寫在 `parseLibraryFilters`（`Filters["chinese_subtitle"]` 存 `[]string`）。`subtitle_status` 的行為與 400 訊息逐字不變。
3. **規則只有一份。** `models.ChineseSubtitleVerdict` 是唯一實作；篩選用註冊進 SQLite 的同一個函式（`RegisterDeterministicScalarFunction`，註冊在 `internal/repository` 套件的 `init()`，讓 repository 的測試也拿得到）。前端不再從 `subtitleTracks` 推「有沒有中文」。
4. **規則照「設計 → 判斷規則」**，Go 表格測試至少涵蓋：`chi`＋`eng`→`zh`；只有 `eng`→`none`；`[]`→`none`；NULL＋`not_searched`→`unknown`；NULL＋`not_found`→`none`；NULL＋`untranslated`＋`en`→`none`；`found`＋`zh-Hant`→`zh_hant`；`found`＋`zh`→`zh`；NFO 字串 `chi,eng`→`zh`；外掛 `zh-TW`→`zh_hant`、`chi.forced`→`zh`、只有 `und`→`unknown`、`eng`＋`und`→`unknown`；`chi`＋title「繁體中文」→`zh_hant`、title「简体」→`zh_hans`、只有 `chi`＋title「粵語」→`none`、只有 `yue`→`none`、`yue`＋`chi`（無粵語 title）→`zh`；圖片格式 `chi`（`hdmv_pgs_subtitle`）＋`no_text_source`→`zh`；`found`＋`en`＋NULL→`unknown`；壞 JSON（不是逗號字串也不是 JSON）→`unknown`，**不可讓查詢失敗**。
5. **篩選與欄位逐筆一致（Rule 28 真實形狀）：** 用真的 migration 鏈建的資料庫（`migrations.NewRunner`＋`RegisterAll(GetAll())`＋`Up`），塞入 AC #4 的每一種電影與兩種影集，對 `has`／`missing`／`unknown` 各打一次 `List` 與 `FullTextSearch`：回來的 id 集合＝「`chinese_subtitle` 欄位落在該組」的 id 集合；三組聯集＝全部、兩兩不相交。再用 handler 測試打一次 `/library?chinese_subtitle=missing` 與錯誤值 400。
6. **掃描寫入：** ffprobe 成功、片內與旁邊都沒有字幕時，`subtitle_tracks` 寫 `[]`（不再留 NULL）；ffprobe 沒裝或失敗、旁邊也沒字幕時維持不寫（＝不知道）。既有測試中斷言「空就不寫」的要改成新行為並在測試名稱說明。
7. **標籤（照「設計 → 畫面」＋D2）：** `deriveSubtitleStatus` 讀 `chineseSubtitle`；`pickPosterBadge` 的穩定狀態集合包含繁中與「中文」；`HeroBanner` 的已就緒文字涵蓋「中文」。`libraryStatus.spec.ts` 用 AC #4 的同一組情境（前端拿到的 `chineseSubtitle`＋`subtitleStatus`）斷言標籤。
8. **篩選畫面（照 D1、D2）：** 鈕、膠囊、無結果那句、手機抽屜計數全讀同一張表；URL `chineseSubtitle`、wire `chinese_subtitle`；testid 改成 `filter-chinese-${value}`；前端不再送 `subtitle_status`；`BatchSubtitleDialog` 的「查看未找到項目」導到 `?chineseSubtitle=missing`。
9. **NAS 形狀的驗收測試（前後端各一）：** 用「設計」那段的真實分布做 fixture：37 部 `chi`＋`eng`、3 部只有 `eng`、12 部 NULL＋`not_searched`、3 部 `found`＋`zh-Hant`、2 部影集 NULL＋`not_searched` → 有 40／缺 3／不知道 14；清單列模式下，有那 40 部的標籤都是繁中／中文、缺那 3 部都是「缺中文」（或 D2 選 B 的字）。
10. **E2E：** 改寫 `tests/e2e/library-mobile.spec.ts` 的字幕篩選案例（鈕、膠囊移除、深連結改 `?chineseSubtitle=missing`、wire 帶 `chinese_subtitle`）與 stub；拿掉／改名 testid 前先 grep `tests/e2e`。
11. **設計稿（Sally）：** 裁定後更新 A6p-M（`Bz0YN`）與電腦版篩選欄的「字幕」區、J2-D 標籤字彙（`flow-j-specs/j2-d`），重匯出截圖、只提交真的有變的圖。
12. **檢查全綠：** `go build ./... && go vet ./... && go test ./...`（apps/api）、`~/go/bin/staticcheck ./...`、`pnpm nx test web`、`pnpm run lint:all`、`python3 scripts/check-design-tokens.py`；視覺基準照 /ship 的 `-linux` 流程（不本機產 `-linux`）。

## Tasks / Subtasks

- [x] T1 後端：`models` 的判斷函式＋中文標籤集合＋關鍵字判斷（`subtitle.embeddedLanguage` 改呼叫它）；`Movie`／`Series` 加 `ChineseSubtitle`（`db:"-"`、`json:"chinese_subtitle"`），`scanMovie`／`scanSeries` 填值；表格測試（AC #1、#3、#4）
- [x] T2 後端：repository `init()` 註冊 SQLite 函式；`movieListFilterConditions`／`seriesListFilterConditions` 加 `chinese_subtitle`；`parseLibraryFilters` 解析＋驗證；真實形狀測試＋handler 測試（AC #2、#3、#5）
- [x] T3 後端：`applyFFprobeTechInfo` 成功探測、沒有字幕時寫 `[]`；更新 enrichment 測試（AC #6）
- [x] T4 設計（Sally，2026-10-06）：A6p-M `Bz0YN`（三顆鈕＋說明）、A3p-M `h1v1U6`、I5-D `vpDLh`（電腦版篩選欄新增「字幕」區）、A4p-D `b1H71g`（清單列「缺中文」「中文」示範）、11 張畫面的海報徽章「缺字幕」→「缺中文」、J2-D `ZpQaw`（J2-2／J2-3／J2-5 改字，新增 J2-6「有沒有中文字幕」字彙對照）；「不知道」單獨出現的字定為「不知道有沒有中文字幕」；截圖 15 張（AC #11）
- [x] T5 前端：型別加 `chineseSubtitle`；`deriveSubtitleStatus`／`pickPosterBadge`／`HeroBanner` 改讀它；`libraryStatus.spec.ts`＋gallery fixtures 補欄位（AC #7、#9）
- [x] T6 前端：篩選表、`FilterPanel`／`FilterChips`／`LibraryBrowseV2`／`LibraryFilterSheetV2`／路由／`libraryService`（list 與 search）／`BatchSubtitleDialog` 深連結；各 spec（AC #8、#9）
- [x] T7 前端：E2E＋stub 改寫；視覺基準（AC #10、#12）
- [x] T8 檢查＋sprint-status（AC #12）

後端 3 項（T1–T3）、前端 3 項（T5–T7）、設計 1 項 → 不需要拆單（門檻是兩邊都 >3）。

## Dev Notes

### 契約（Rule 20）

- 本單**定義** AC #1 `[@contract-v1]`（`chinese_subtitle` 欄位）與 AC #2 `[@contract-v1]`（`chinese_subtitle` 查詢參數）。下游：`disc-2026-10-subtitle-filter-series-phase-2`（建單時要 ack）。
- 本單**消費**、不改：`confirmed against [@contract-v1] (Story dsr-1b-a AC #1)`（`subtitle_status` CSV 參數，形狀不變、不 bump）；`confirmed against [@contract-v1] (Story dsr-1b-a2 AC #1)`（`/library/search` 用同一個 `parseLibraryFilters`，只多一個參數，既有形狀不變、不 bump）。兩張上游都 done。

### 不要做的事

- **不要在前端或 SQL 另寫一份「有沒有中文」的判斷**（AC #3）。前端 `HANT`／`HANS` 只留給「這一軌叫什麼語言」用。
- **不要建任何 index、view、trigger 用到 `vido_chinese_subtitle`**：資料庫檔案會變成外部工具（`sqlite3` CLI、備份還原的唯讀連線 `backup_service.go:941`）打不開那個物件。它只能出現在查詢裡。
- 不要改 `missingZhHantSubtitleWhere`／`hasZhHantSubtitleWhere`（生成批次的範圍、首頁覆蓋率，`movie_repository.go:974-1035`）——首頁覆蓋率另立單（見 Discovery Triage）。
- 不要改掃描時字幕檔「語言照檔名」的寫法（Rule 19 擋住、範圍外）；「有沒有中文」用檔名標籤夠準，錯的多半是繁簡。
- 不要拿掉後端的 `subtitle_status` 參數或改它的行為（AC #2）。
- 不要改 `.pen`——T4 是 Sally 的。

### 已知陷阱

- **註冊時機：** modernc 的自訂函式只對「註冊之後開的連線」有效（`sqlite.go:447-448`）。放在 repository 套件的 `init()`，app 與 repository 測試都會在開連線前跑到。用 mattn `sqlite3` 驅動的測試（例如 `internal/repository/explore_block_repository_test.go`、`media_library_repository_test.go`）拿不到——它們不碰電影／影集列表，若有人改了要注意。
- **`movieSelectColumnsQualified` 用逗號切欄位**（`movie_repository.go:599-605`）：不要把函式呼叫塞進 `movieSelectColumns`；欄位值在 Go 端（scan 時）算，SQL 只用在 WHERE。
- **SQL 函式不能回錯誤讓整個列表 500：** 輸入壞掉一律回 `unknown`（AC #4 最後一條）。參數可能是 `nil`（NULL）。
- **`FullTextSearch` 帶 alias `m`／`s`**：函式參數要用 `col(...)` 加前綴（`movie_repository.go:624-629` 的寫法）。
- **`setupTestDB` 是手寫的 `CREATE TABLE`**（`movie_repository_test.go:16-60`），不算 Rule 28 的真實形狀；AC #5 要用 migration 鏈。
- **gallery fixtures 很多片沒有新欄位**：沒有 `chineseSubtitle` 時前端要當 `unknown`（不顯示標籤），不要偷偷退回去讀 `subtitleTracks`。需要顯示標籤的 fixture 補上欄位；visual 基準可能有變（原本 `chi` 軌顯示「有字幕」的卡片會變成不顯示）。
- 那 12 部 NULL 的電影：唯讀查 NAS（`sqlite3 -readonly /mnt/cache/appdata/vido/vido.db`）`SELECT video_codec IS NOT NULL, metadata_source, COUNT(*) FROM movies WHERE subtitle_tracks IS NULL AND (is_removed=0 OR is_removed IS NULL) GROUP BY 1,2`。有 `video_codec` 的那些是「讀過、沒字幕」，但因為 `enrichment_service.go:773-776` 會跳過，重新整理也不會變——記到 `disc-2026-10-movie-subtitle-tracks-unknown-refresh`，本單不處理。
- `untranslated` 那條「NULL 也算缺」的推論依據是 pipeline 遇到片內中文會優先用（`extractor.go:133-140`）；若 dev 發現有別的路徑會寫 `untranslated`，回報 SM。

### Time-dependent visual coverage

N/A — 不碰讀時鐘的元件（`libraryStatus.ts`、篩選元件都不讀 `Date`）。

### References

- 裁定：sprint-status.yaml:1163；`.claude/memory/project_cn_subtitle_policy.md`（陸港澳轉繁，與本單無衝突：本單只判斷「有沒有中文」，不轉換）
- `bugfix-subtitle-dialog-real-inventory.md`（`ListSidecars`／`embeddedLanguage`、語言看內容）
- `disc-2026-10-episode-list-subtitle-badge.md`（影集第二階段的前置）
- `dsr-1b-a-library-subtitle-status-filter-backend.md`、`dsr-1b-a2-library-search-applies-filters.md`、`dsr-1b-b-library-mobile-sort-filter-sheet.md`（舊篩選的來龍去脈）
- `project-context.md` Rule 19（`:532-624`）、Rule 20（`:626-715`）、Rule 28（`:1170-1195`）

## Dev Agent Record

### Agent Model Used

Claude Opus 5.5（Amelia，dev-story，非互動模式，2026-10-06）

### Debug Log References

- 紅燈先行：`models` 表格測試先寫、`go test` 編譯失敗（undefined `ChineseSubtitleZh`）→ 實作後綠。
- 突變檢查（AC #5 真實形狀測試）：(1) 把 `ChineseSubtitleVerdict` 的 `case SubtitleStatusNotFound, SubtitleStatusUntranslated` 拿掉 `NotFound` → `TestChineseSubtitleFilter_RealShape_FilterMatchesField` 紅（`field for "NULL not_found"`）；(2) 讓 SQL 轉接器忽略 `subtitle_tracks`（`chineseSubtitleSQL` 第三參數傳 `""`）→ 同一支測試紅（`movie List has`／`missing`、`FullTextSearch` 計數全不符）。兩者皆已還原，還原後 `go test ./internal/repository/ ./internal/models/` 綠。
- 本機 visual：4 張與本單無關的 diff（`parse-floating-parse-progress-card`、`notifications-new-media-notifications`、`library-poster-card-menu`、`retry-retry-notifications`）——整頁截圖裡的側欄數字（電影 1、儲存空間 0.7/1.0 TB）來自本機 :8080 上已在跑、有本機資料的 API（`reuseExistingServer`），不是本單造成，未更新。

### Completion Notes List

- 2026-10-06 Bob（SM）：Ultimate context engine analysis completed - comprehensive developer guide created。
- 2026-10-06 Alexyu 裁定 D1 三顆、D2 中文、粵語不算 → ready-for-dev。T4（設計稿）要等 Sally 手上的篩選欄動畫規格畫面畫完再動 `.pen`。
- **T1**：`apps/api/internal/models/chinese_subtitle.go` 是唯一的規則（`ChineseSubtitleVerdict`、`ChineseTitleScript`、篩選分組 `ChineseSubtitleFilterVerdicts`）；`subtitle.embeddedLanguage` 改呼叫 `models.ChineseTitleScript`（行為不變，`TestEmbeddedLanguage` 綠）；`Movie`／`Series` 加 `ChineseSubtitle`（`db:"-"`、`json:"chinese_subtitle"`、無 omitempty），只在 `scanMovie`／`scanSeries` 填。
- **T2**：`internal/repository/chinese_subtitle_sql.go` 在 `init()` 註冊 `vido_chinese_subtitle`（modernc `MustRegisterDeterministicScalarFunction`），只用在 WHERE；壞輸入一律回 `unknown`、永不回錯。`movieListFilterConditions`／`seriesListFilterConditions` 加 `chinese_subtitle`（帶 FTS alias）；`parseLibraryFilters` 解析＋400 `chinese_subtitle contains unknown value "<v>"`；`subtitle_status` 一字未動。
- **T3**：`applyFFprobeTechInfo` 的 `writeSubs` 多一個 `probed` 參數：探測成功且片內、旁邊都沒字幕 → 寫 `[]`；沒裝 ffprobe／探測失敗 → 照舊不寫。原本沒有任何既有測試斷言「空就不寫」，所以是新增三支測試（名稱說明新行為）。
- **T5**：`deriveSubtitleStatus` 只讀 `chineseSubtitle`（`zh_hant`→繁中、`zh_hans`→簡中、`zh`→中文〔success、steady〕；非「有」時 `no_text_source`／`skipped`／`untranslated` 照舊、處理中三態 null、`none`→缺中文、其餘 null）。不再從 `subtitleTracks` 推「有沒有中文」；舊 fixture 沒欄位 → 不顯示。`HeroBanner`：「中文」→「中文字幕 ✓ 已就緒」、可點的集合改成「缺中文／未翻譯／簡中」。gallery 首頁大圖 fixture 補 `chineseSubtitle: 'zh_hant'`（其餘 fixture 不經過標籤推導）。
- **T6**：新的一張表 `components/library/chineseSubtitleFilter.ts`（舊 `subtitleStatusFilter.ts` 刪除）；`FilterValues.chineseSubtitle`、URL `chineseSubtitle`、wire `chinese_subtitle`、testid `filter-chinese-${value}`；前端不再送 `subtitle_status`（`LibraryListParams.subtitleStatus` 移除）。`BatchSubtitleDialog` 深連結改 `?chineseSubtitle=missing`。
- **T7**：`tests/e2e/library-mobile.spec.ts`＋`tests/support/helpers/library-stubs.ts` 改寫（stub 依 `chinese_subtitle` 分組篩；新增斷言：不送 `subtitle_status`、篩完每張海報都是「缺中文」）；`tests/e2e/hero-banner.spec.ts` stub 補 `chinese_subtitle`。視覺基準：`library-filter-panel`（default／hover／focus）與 `library-mobile-sheets/sort-filter` 更新 `-darwin`、`git rm` 對應 `-linux`（等 CI bootstrap PR）。
- **測試證據**：Go `go build ./... && go vet ./... && go test ./...` 42 個套件 ok、0 FAIL；`~/go/bin/staticcheck-2026.1 ./...` 0；`pnpm nx test web` 298 檔／4597 測試全綠；`pnpm run lint:all` 0 error（172 個既有 warning）、prettier 全過；`python3 scripts/check-design-tokens.py` 一致；Playwright `library-mobile.spec.ts`＋`hero-banner.spec.ts`（chromium）20/20 passed；visual 本單相關 fixture 綠。
- **E2E 環境說明**：本機 :8080 API 與 :4200 Vite 已在跑（10:59 起），`reuseExistingServer` 會重用。Vite 從這個工作樹即時編譯，所以前端改動有吃到；API 是舊的，但這兩支 e2e 的 API 全部 `page.route` stub，不受影響。後端走真實路徑的驗證由 Go handler 整合測試（`library_chinese_subtitle_integration_test.go`）負責。
- **與 story 的偏離／自行決定（請 CR／Sally 確認）**：
  1. 粵語 title 判斷套用在**所有**中文標籤（含 `zh-HK`＋「Cantonese」），不只 `chi`／`zho`／`zh`——與 `embeddedLanguage` 既有優先序一致（story 要求兩邊同一份判斷）。
  2. 「已知的非中文」要長得像語言碼（2–3 個字母＋可選子標籤，排除 `und`／`mul`／`mis`）；`forced`／`sdh`／`cc`／`default` 這類檔名旗標略過不算語言。所以 `Movie.forced.srt`、NFO 寫成 `Chinese` 這種全字 → `unknown`，絕不說「缺」。`zh-*` 未列出的子標籤依 `hant/tw/hk/mo`、`hans/cn/sg` 判繁簡，`zh-yue` 當粵語。
  3. 讀不懂的 `subtitle_tracks`（壞 JSON、非標籤字串）當作 NULL 處理（所以壞值＋`not_found` 仍是 `none`）。
  4. 單獨出現的篩選膠囊與「沒有結果」那句，「不知道」寫成「不知道有沒有中文字幕」（表格多一欄 `chipLabel`），鈕仍是 D1 的「不知道」。這是 T4 該定的字，留給 Sally。
  5. 「中文」標籤用 success 色（與繁中同為穩定的「有」）。
- **順手修正**：`LibraryBrowseV2` 原本把 URL 上未驗證的 `subtitleStatus` 原樣送上 wire（與表格檔註解「bogus 不可上 wire」矛盾）；新版只送經 `parseChineseSubtitleCsv` 過濾後的值，並有單元測試。
- **`untranslated` 寫入路徑**：只有 `services/transcription_service.go`（聽聲音生成那條，story 已列），片內字幕 pipeline 沒有寫；未發現新路徑。
- 🔗 AC Drift: FOUND — `dsr-1b-b` AC #2（字幕篩選三顆 `found`／`not_found`／`not_searched`、URL `subtitleStatus`、testid `filter-subtitle-*` → 三顆 `has`／`missing`／`unknown`、URL `chineseSubtitle`、testid `filter-chinese-*`）；`8-11` AC #6（「查看未找到項目」`?subtitleStatus=not_found` → `?chineseSubtitle=missing`）；`ux3-0-1`／`ux3-0-2`／`sub-1-7b`／`sub-2-2b` 的標籤推導（「有字幕／缺字幕」與「從字幕軌推繁簡」→ 只讀後端 `chineseSubtitle`，字改「缺中文／中文」）。皆為本單裁定的刻意改變。測試面同步更新：`tests/e2e/library-mobile.spec.ts`、`tests/e2e/hero-banner.spec.ts`、`tests/support/helpers/library-stubs.ts`、gallery `homepage-hero-banner`／`library-mobile-sheets/sort-filter` fixture、上述 web spec。
- 📎 Contract Stamps: FOUND（本單定義 AC #1、AC #2 `[@contract-v1]` 兩個新契約；上游 `dsr-1b-a` AC #1、`dsr-1b-a2` AC #1 `[@contract-v1]` 只消費、形狀不變，Dev Notes 已有 ack；`models.SubtitleStatus` `[@contract-v3]` 值集合未動）。
- 🎭 A11y Pre-Flight: PASS（FilterPanel／FilterChips／LibraryFilterSheetV2／PosterCardV2／HeroBanner 5 個元件；觸碰檔案 jsx-a11y warning 0、本單新增 0；篩選鈕保留 `aria-pressed`、膠囊移除鈕 `aria-label` 從同一張表產生）。
- 🎨 UX Verification（2026-10-06，主 session 對照）：本機 `/library?chineseSubtitle=missing`（電腦版 1440）與 I5-D／A3p-D／A6p-M 新稿比對——三顆鈕的字、已選膠囊「缺中文字幕」、海報徽章「缺中文」一致。已知差異（都是本單之前就有、本單不動）：① 電腦版篩選欄分區順序，碼是「狀態→字幕→年份」，I5-D 是「年份→狀態→字幕」；② 「缺中文」徽章顏色，稿是 warning 橘、碼是 neutral 灰 → 立 `disc-2026-10-missing-chinese-badge-tint-drift`。

  | 區域 | 設計稿 | 實作 | 一致？ | 待辦 |
  |---|---|---|---|---|
  | 篩選「字幕」區 | A6p-M／電腦版篩選欄仍是 有字幕／缺字幕／還沒搜尋 | 有中文字幕／缺中文字幕／不知道 | 否（稿未更新） | T4 Sally |
  | 標籤字彙 | J2-D 仍是 缺字幕／有字幕 | 缺中文／中文 | 否（稿未更新） | T4 Sally |


### Code Review（2026-10-06，主 session 對抗式審查，與 dev 不同 context）

- 讀過：`models/chinese_subtitle.go`、`repository/chinese_subtitle_sql.go`、`utils/libraryStatus.ts` diff；實際起 API（seed DB＋手改 9 部片的 `subtitle_tracks`）打 `/library?chinese_subtitle=has|missing|unknown` → 6／3／9＝18 全部，`bogus` 回 400；`/library/search` 帶參數正常。走的是 app 真的 `sqlite-utc` 驅動，證明 SQL 函式在正式路徑有註冊。
- **舊 bug 一併消滅（記錄）：** 舊 `trackLangs` 遇到 API 回 `subtitle_tracks: null` 時 `JSON.parse(null)` → `[]` → 標「缺字幕」——NAS 上 13 部電影＋81 部影集因此掛著「缺字幕」、篩選卻是 0。新版不再由前端推斷，NULL 一律 `unknown`。
- **LOW（不修，記錄）：** `subtitle_status=no_text_source`／`skipped` 且字幕軌只有 `und`（或 NULL）時，判定是 `unknown`（落在「不知道」），但徽章顯示「無字幕源／已略過」。兩者不矛盾（徽章說的是原因），但「不知道」那顆裡會出現這兩種徽章。
- **LOW（不修，記錄）：** `und` 軌即使 `title` 寫「繁體中文」也不會被認成中文（`classifyTrack` 只在標籤已是中文時才看 title）。結果是「不知道」，不會誤說缺。
- 結論：無阻擋項。

### File List

- apps/api/internal/models/chinese_subtitle.go（新）
- apps/api/internal/models/chinese_subtitle_test.go（新）
- apps/api/internal/models/movie.go
- apps/api/internal/models/series.go
- apps/api/internal/repository/chinese_subtitle_sql.go（新）
- apps/api/internal/repository/chinese_subtitle_filter_test.go（新）
- apps/api/internal/repository/movie_repository.go
- apps/api/internal/repository/series_repository.go
- apps/api/internal/handlers/library_handler.go
- apps/api/internal/handlers/library_handler_test.go
- apps/api/internal/handlers/library_chinese_subtitle_integration_test.go（新）
- apps/api/internal/services/enrichment_service.go
- apps/api/internal/services/enrichment_local_analysis_test.go
- apps/api/internal/subtitle/inventory.go
- apps/web/src/types/library.ts
- apps/web/src/utils/libraryStatus.ts
- apps/web/src/utils/libraryStatus.spec.ts
- apps/web/src/components/homepage/HeroBanner.tsx
- apps/web/src/components/homepage/HeroBanner.spec.tsx
- apps/web/src/components/library/chineseSubtitleFilter.ts（新）
- apps/web/src/components/library/chineseSubtitleFilter.spec.ts（新）
- apps/web/src/components/library/subtitleStatusFilter.ts（刪）
- apps/web/src/components/library/subtitleStatusFilter.spec.ts（刪）
- apps/web/src/components/library/FilterPanel.tsx
- apps/web/src/components/library/FilterPanel.spec.tsx
- apps/web/src/components/library/FilterChips.tsx
- apps/web/src/components/library/FilterChips.spec.tsx
- apps/web/src/components/library/LibraryBrowseV2.tsx
- apps/web/src/components/library/LibraryBrowseV2.spec.tsx
- apps/web/src/components/library/LibraryFilterSheetV2.tsx
- apps/web/src/components/library/LibraryFilterSheetV2.spec.tsx
- apps/web/src/components/library/PosterCardV2.tsx
- apps/web/src/components/library/PosterCardV2.spec.tsx
- apps/web/src/components/media/LocalDetailV2.tsx（註解）
- apps/web/src/components/subtitle/BatchSubtitleDialog.tsx
- apps/web/src/components/subtitle/BatchSubtitleDialog.spec.tsx
- apps/web/src/routes/library.tsx
- apps/web/src/routes/test/-gallery.fixtures.tsx
- apps/web/src/services/libraryService.ts
- apps/web/src/services/libraryService.spec.ts
- tests/e2e/library-mobile.spec.ts
- tests/e2e/hero-banner.spec.ts
- tests/support/helpers/library-stubs.ts
- tests/visual/components.visual.spec.ts-snapshots/components/library-filter-panel/{default,hover,focus}-visual-darwin.png（更新）
- tests/visual/components.visual.spec.ts-snapshots/components/library-filter-panel/{default,hover,focus}-visual-linux.png（git rm，等 CI bootstrap）
- tests/visual/components.visual.spec.ts-snapshots/components/library-mobile-sheets/sort-filter/default-visual-darwin.png（更新）
- tests/visual/components.visual.spec.ts-snapshots/components/library-mobile-sheets/sort-filter/default-visual-linux.png（git rm，等 CI bootstrap）
- _bmad-output/implementation-artifacts/sprint-status.yaml（只改本單那一行）
- _bmad-output/implementation-artifacts/dsr-1b-b-library-mobile-sort-filter-sheet.md (AC drift reference — see Completion Notes；檔案未改)
- _bmad-output/implementation-artifacts/8-11-batch-subtitle-ui.md (AC drift reference — see Completion Notes；檔案未改)

### Discovery Triage

- **③** 影集的中文字幕結論要改成各集彙總，且每集的旁邊字幕檔結果要存進資料庫 → `disc-2026-10-subtitle-filter-series-phase-2`（建單時已立；雙向：該條目指回本單、`disc-2026-10-episode-list-subtitle-badge.md` Dev Notes 已加註）。
- **③** 12 部電影 `subtitle_tracks` 是 NULL，「讀過沒字幕」的那些永遠不會被重新讀（`enrichment_service.go:758-760`、`:773-776`）→ `disc-2026-10-movie-subtitle-tracks-unknown-refresh`（建單時已立）。
- **③** 首頁「字幕覆蓋率」只數 `subtitle_language = 'zh-Hant'`（`movie_repository.go:1031-1035`，`home_summary_service.go:157`），片內有繁中軌的片不算已覆蓋，首頁因此一直給「產生字幕」的門（`HomeReadoutBand.tsx:405`、`:424`）——第三套定義 → `disc-2026-10-home-coverage-counts-only-vido-zh-hant`（建單時已立）。

## Change Log

- 2026-10-06 Bob create-story：drafted — 待裁定 2 項。新定義 AC #1、AC #2 `[@contract-v1]`（新契約，非 bump）。
- 2026-10-06 Bob create-story（草稿）→ Alexyu 裁定 → ready-for-dev。
- 2026-10-06 Amelia dev-story：T1–T3、T5–T8 完成（T4 設計稿依指示跳過，留給 Sally）→ review。新定義的 AC #1、AC #2 `[@contract-v1]` 已實作（新契約，非 bump）；上游 `dsr-1b-a`／`dsr-1b-a2` `[@contract-v1]` 未變。
- 2026-10-06 Sally T4 設計稿完成；主 session 對抗式 CR（見下）與 UX 對照。
