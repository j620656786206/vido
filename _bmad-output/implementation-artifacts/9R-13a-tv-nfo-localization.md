# Story 9R-13a: TV .nfo 在地化 —— tvshow.nfo / 每集 nfo 的 backup-and-replace 路線

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

**Epic:** epic-9R-subtitle-route-c（Track 6 尾款）· **Priority:** P1（differentiator 的 TV 半邊）
**Depends:** 9R-13 ✅（movies 版已出貨）、9R-6 ✅、9R-7 ✅、9R-S1 ✅（spike 裁定 replace-only）
**Source:** sprint-status `9R-13a-tv-nfo-localization`（Rule-24 ③ from 9R-13，S1 movies-first re-spec 遞延）。
2026-08-19 策略重審：.nfo 在地化是 ADR 自封的「category-level differentiator」，但 TV 半邊缺席 —— 而繁中 NAS 使用者的庫**以 TV／動漫為大宗**，差異化最薄的地方正是最重要的地方。

> **⚠️ 2026-08-21 RE-AUTHORED（第二版）。** 初稿（2026-08-19）**在 Task 清單處截斷** —— 無 Dev Notes、References、Dev Agent Record、Discovery Triage。
> 本版逐行 grep 實查 `main` HEAD `97a4c45c`，補齊全部 template 區段，並**更正／新增五項會決定實作對錯的事實與裁定**（見「初稿缺口」）。
> 其中 **#1 是會直接把檔案寫到錯誤位置的事實錯誤**，**#2 需要 Alexyu 一句裁定**。

---

## Story

As a NAS owner whose library is mostly TV shows,
I want `tvshow.nfo` and per-episode `.nfo` localized to zh-TW like movies already are,
so that my player shows 繁中劇情與集名 for the media type that dominates my library — with my originals always backed up before any replace.

---

## Context —— 現況逐行查證 @ `97a4c45c`

### 可鏡射的既有件（❌ 不得重新發明）

| 資產 | 位置 | 說明 |
|---|---|---|
| `NFOLocalizerService` | `internal/services/nfo_localizer_service.go:24` | 持有 `translation` + `glossaryRepo` + `logger`；`NewNFOLocalizerService` **不在 boot 檢查 IsConfigured**（sub-2-1a 熱重載，CR M1 的教訓） |
| `LocalizeMovieNFO` | 同檔 `:66` | 五步驟：可用性 → 檔案路徑檢查 → `movieToNFO` → `loadGlossary` + `translateFields` → `marshalNFO` + 寫入 |
| `translateFields` | 同檔 `:106` | 單批 `TranslateRequest`（9R-7）翻 title/plot/genres/**actor roles**；**丟掉空字串欄位**；per-field fail-soft（查無翻譯保留原值） |
| `loadGlossary` | 同檔 `:88` | `LookupByMedia(mediaID, false)`；err 或空 → nil（fail-soft） |
| `writeAdditiveNFO` | 同檔 `:202` | **電影專屬**的雙槽策略；`default` 分支即 backup-and-replace 樣板（`.orig` 已存在則不覆蓋備份） |
| `NFOLocalizeResult` | 同檔 `:56` | `{Path, BackupPath, Replaced}` —— **TV 直接沿用，不需新型別** |
| `SeriesNFO` struct | `internal/services/nfo_generator.go:32` | **已存在**：`tvshow` root、Title/OriginalTitle/Year/Plot/Genres/Actors/UniqueIDs/Thumb/Rating/Status |
| `GenerateSeriesNFO` | 同檔 `:127` | series → SeriesNFO 的完整欄位映射（含 credits 前 10 名、tmdb/imdb uniqueid）—— **`seriesToNFO` 直接鏡射它** |
| `marshalNFO` | 同檔 `:179` | XML 序列化 |
| handler 樣板 | `internal/handlers/nfo_localizer_handler.go` | 窄介面 `NFOLocalizerMovieGetter` + `NFOLocalizerServiceInterface` + `RegisterRoutes`；main.go:960 `if nfoLocalizer != nil` 才掛載 |
| episode 窄介面先例 | `internal/handlers/transcription_handler.go:27` `TranscriptionEpisodeGetter` | `FindByID(ctx, id) (*models.Episode, error)`，`*repository.EpisodeRepository` 已滿足（Rule 11 樣板） |
| 整劇列舉 | `repository.EpisodeRepository.FindBySeriesID:124` | 已 `ORDER BY season_number, episode_number` |

### 初稿（2026-08-19）缺口 —— dev 必讀

| # | 初稿的說法 | 實況（grep 實證） | 本版裁定 |
|---|---|---|---|
| **1** 🔴 | 沒說 `tvshow.nfo` 要寫在哪，暗示比照 movies 的 `filepath.Dir(mediaFilePath)` | **`Series.FilePath` 存的是「劇集資料夾」不是檔案** —— `media_ingest_service.go:147` 寫入 `in.SeriesDir`，由 `SeriesDirFor:94` 從單集路徑往上爬過季資料夾（`seasonDirPattern:85` 認得 `Season 02` / `S02` / `Specials` / `第二季`）得出 | `tvshow.nfo` = `filepath.Join(series.FilePath.String, "tvshow.nfo")`。**🔴 絕對不可對 series 路徑呼叫 `filepath.Dir()`** —— 那會把檔案寫到劇集資料夾的**上一層**（整個媒體庫根目錄），三家播放器都掃不到，而且會污染使用者的庫根 |
| **2** ⚖️ | 「沿用既有 NFO 錯誤前綴，不發明新 code family」 | **`NFO_` 根本不在 Rule 7 的權威前綴集**（16 個來源，無 `NFO_`）。9R-13 出貨的 `NFO_LOCALIZE_DISABLED` / `NFO_LOCALIZE_FAILED`（`nfo_localizer_handler.go:44,65`）是**既有的 Rule 7 違規**，只是沒被抓到 | **需 Alexyu 裁定**，AC #7 已寫入預設方案（擴充 Rule 7 至 17 個來源）與備選（改用 `METADATA_`）。**不得沉默地再加第三個 `NFO_` 碼** |
| **3** | 「`SeriesNFO` struct 與 `marshalNFO` 已在 `nfo_generator.go`」 | ✅ 正確。但初稿沒提 **`GenerateSeriesNFO:127` 已經有完整的 series→NFO 欄位映射** | `seriesToNFO` **逐行鏡射 `GenerateSeriesNFO`**（含 credits 前 10、Status、Thumb），不要重新想欄位 |
| **4** | 「episode 的 NFO struct 是本案淨新增」 | ✅ 正確。但沒說 root element 是什麼 | Kodi 的每集 NFO root 是 **`<episodedetails>`**（不是 `episode`）。必要欄位：`title` / `showtitle` / `season` / `episode` / `plot` / `aired` / `runtime` / `rating` / `thumb` / `uniqueid`。**`showtitle` 需要父劇集標題** ⇒ 批次時**只查一次 series**（AC #4.3） |
| **5** | AC #1 說 cast roles「若 series 側查得到角色資料」 | `models.Series.CreditsJSON` 存在且 `GenerateSeriesNFO` 已解析 —— **查得到就是查得到**，不是不確定 | 直接翻 `Actors[i].Role`，與 movies 版 `translateFields:130` 同款。`models.Episode` **沒有** credits ⇒ 每集 NFO 無 actor 區塊（明示的範圍事實，非遺漏） |

---

## Acceptance Criteria

### AC #1 — `LocalizeTVShowNFO(ctx, series models.Series) (*NFOLocalizeResult, error)`

1.1 前置檢查鏡射 movies 版：`!s.IsAvailable()` → error；`!series.FilePath.Valid || == ""` → error（訊息明說「掃描媒體庫後才會有劇集資料夾」）。
1.2 `seriesToNFO(series) SeriesNFO` —— **逐行鏡射 `nfo_generator.go:127 GenerateSeriesNFO`** 的欄位映射（Title / OriginalTitle / Year=FirstAirDate / Plot=Overview / Genres / Rating=VoteAverage / Thumb=PosterPath / Status / Actors 取 credits cast 前 10 / uniqueid tmdb+imdb）。
1.3 翻譯：`loadGlossary(ctx, series.ID)` + `translateFields`（title / plot / genres / actor roles）。
- 🔴 `translateFields` 目前簽名吃 `MovieNFO`。**不得為了 TV 把它改成泛型或複製一份** —— 抽出共用的「欄位切片 ↔ 結果回填」邏輯，或讓兩者共用一個以 `[]TranslationField` 為介面的小函式。**movies 版行為必須 byte-unchanged**（AC #6 回歸釘）。
1.4 寫入 `filepath.Join(series.FilePath.String, "tvshow.nfo")`：
- 檔案不存在 → 直接寫（無原檔可保護，`Replaced=false`、`BackupPath=""`）。
- 檔案存在 → **首次**備份為 `tvshow.nfo.orig`（備份已存在則**不覆蓋備份**，保住最原始的那份），再原槽覆寫（`Replaced=true`、`BackupPath` 填入）。
- 🔴 **不可 `filepath.Dir(series.FilePath.String)`** —— 見「初稿缺口 #1」。

### AC #2 — `LocalizeEpisodeNFO(ctx, episode models.Episode, showTitle string) (*NFOLocalizeResult, error)`

2.1 新增 `EpisodeNFO` struct（`nfo_generator.go`，緊鄰 `SeriesNFO`）：
```go
type EpisodeNFO struct {
    XMLName   xml.Name      `xml:"episodedetails"`
    Title     string        `xml:"title"`
    ShowTitle string        `xml:"showtitle,omitempty"`
    Season    int           `xml:"season"`
    Episode   int           `xml:"episode"`
    Plot      string        `xml:"plot,omitempty"`
    Aired     string        `xml:"aired,omitempty"`
    Runtime   int           `xml:"runtime,omitempty"`
    Rating    float64       `xml:"rating,omitempty"`
    Thumb     string        `xml:"thumb,omitempty"`
    UniqueIDs []NFOUniqueID `xml:"uniqueid,omitempty"`
}
```
2.2 `episodeToNFO(episode, showTitle)`：`Title`←`Episode.Title`（NullString）、`Plot`←**`Episode.Overview`**（episodes 表**沒有 plot 欄位**，與 movies 版 `Overview→Plot` 同款映射）、`Aired`←`AirDate`、`Runtime`←`Runtime`、`Rating`←`VoteAverage`、`Thumb`←`StillPath`、`uniqueid tmdb`←`TMDbID`。
- **無 actor 區塊** —— `models.Episode` 沒有 credits（明示的範圍事實）。
2.3 翻譯：只有 `title` + `plot` 兩個欄位。
- 🔴 **glossary key 用父 SERIES id，不是 episode id**（`episode.SeriesID`）。理由與 sub-5-5 CR H1 的 `glossaryMediaKey` 完全相同：per-episode key 會讓同劇每集各自為政、名詞庫互相看不到。**測試釘住**。
2.4 寫入 `filepath.Join(filepath.Dir(episode.FilePath.String), <basename>+".nfo")` —— episode 的 `FilePath` **是檔案**（`media_ingest_service.go:266`），所以這裡**要**用 `Dir()`。備份語意同 AC #1.4（`.nfo.orig`，首次才備份）。
2.5 `!episode.FilePath.Valid || == ""` → error。

### AC #3 — 🔴 Replace opt-in 閘門（本案最重要的紅線）

3.1 新路由（掛在既有 `NFOLocalizerHandler` 上，同一個 `RegisterRoutes`）：
- `POST /api/v1/series/:id/localize-nfo`（可選 query `?include_episodes=true`）
- `POST /api/v1/episodes/:id/localize-nfo`
3.2 兩者的 request body 都必須帶 `{"confirm_replace": true}`。缺席、`false`、或 body 非法 → **HTTP 409**（`StatusConflict`，`subtitle_pipeline_handler.go:101` 的既有樣式）＋錯誤碼（見 AC #7），訊息以 **zh-TW** 說明「TV 的 nfo 是單槽名稱、沒有 additive 選項，這是覆寫操作；原檔會先備份為 `.nfo.orig`」（Rule 3）。
3.3 🔴 **未經 confirm 的呼叫必須零檔案寫入、零翻譯呼叫**（不只是不寫檔 —— 也不可先花錢翻譯再拒絕）。閘門必須在 service 呼叫**之前**。測試同時斷言**檔案系統無變化**與**翻譯 provider 呼叫次數 == 0**。
3.4 服務不可用（`!IsAvailable()`）→ 503，鏡射 movies 版 `:44`。
3.5 movies 路由 `POST /movies/:id/localize-nfo` **語意與程式碼一律不動**（它是 additive，本來就不需要 confirm）。

### AC #4 — `include_episodes` 批次整劇（fail-soft）

4.1 先做 tvshow.nfo，再以 `EpisodeRepository.FindBySeriesID` 逐集處理。
4.2 單集失敗 → 記錄並**續行**（Rule 13 case 3）；回傳彙總：`{show: *NFOLocalizeResult, episodes: [...], succeeded, failed, skipped}`（`skipped` = 無 FilePath 的集數）。
4.3 **series 只查一次**：`showTitle` 從已載入的 series 取，不可逐集回頭查 series（N+1）。
4.4 整劇零集數、或全部無 FilePath → 回傳 show 結果 + 空 episodes 清單，**不是錯誤**。

### AC #5 — 測試

5.1 **槽位矩陣（tvshow）**：不存在 → 直接寫、`Replaced=false`；存在且無備份 → 建 `.orig` + 覆寫、`Replaced=true`；存在且**備份已存在** → **備份內容不被覆蓋**（斷言 `.orig` 的 bytes 仍是最初那份）+ 覆寫。
5.2 **槽位矩陣（episode）**：同上三案。
5.3 🔴 **路徑正確性**：斷言 tvshow.nfo 落在 `series.FilePath` **本身**（不是它的父目錄）。用一個 `t.TempDir()/Show/Season01/ep.mkv` 的真實結構，同時斷言 episode nfo 落在 `Season01/` 內。
5.4 🔴 **confirm 閘門**：缺席 / `false` / 空 body 三案 → 409、**臨時目錄內檔案數不變**、**fake completer 呼叫次數 == 0**。
5.5 **glossary series-key**：episode 路徑的 `LookupByMedia` 收到的是 `episode.SeriesID`（fake repo 記錄參數）。
5.6 **欄位 fail-soft**：翻譯回傳缺 `plot` → 保留原值；`Title` 為 NULL → 不送翻譯、不 panic。
5.7 **批次 fail-soft**：3 集其中 1 集無 FilePath → succeeded=2 / skipped=1，且**已成功的兩集檔案確實存在**。
5.8 🔴 **movies 回歸釘**：`LocalizeMovieNFO` 既有 7 條測試**一行不改**仍全綠（AC #1.3 共用重構的證明）。
5.9 handler 測試（`nfo_localizer_handler_test.go` **目前不存在**，本案新建）。

### AC #6 — 範圍紅線

- ❌ **不動 movies 路徑**：`LocalizeMovieNFO` / `movieToNFO` / `writeAdditiveNFO` 的行為 byte-unchanged（`translateFields` 若被抽共用，必須有回歸測試證明）。
- ❌ **不做 FE**：連 movies 版路由都還**零前端呼叫者**（策略重審實證）。nfo 的 UI 曝光是獨立 FE story（見 Discovery Triage）。
- ❌ **不做自動觸發**：nfo 在地化是否納入 9R-10b 的常設同意政策，留給該 story 一併裁定。
- ❌ **不碰字幕管線**（9R-5 / 9R-8 剛出貨的區域一律不動）。
- ❌ **不新增 season.nfo**（Kodi 的季資料夾 nfo）—— 範圍外，需要時另案。

### AC #7 — ⚖️ Rule 7 前綴裁定 —— **已裁定：方案 A（Alexyu, 2026-08-21）**

7.1 **事實**：`NFO_` **不在** Rule 7 的權威前綴集（16 個來源）。9R-13 已出貨的 `NFO_LOCALIZE_DISABLED` / `NFO_LOCALIZE_FAILED` 是既有違規。本案還會再加一個（AC #3.2 的 409）。
7.2 ⚖️ **裁定＝方案 A（Alexyu 2026-08-21）**：**擴充 Rule 7 為 17 個來源**，新增 `NFO_`。備選 7.3 不採用。
- 同步 `project-context.md` Rule 7 的**碼清單**與**權威前綴集句子**（16 → 17）。
- 同步 `_bmad/bmm/workflows/4-implementation/code-review/instructions.xml` Step 3 的 inline prefix 清單與 HTML 註解同步日期（Rule 7 原文明文要求）。
- 理由：nfo 在地化有自己的 service、handler 與檔案寫入語意，是獨立子系統；而現有 `METADATA_*` 全部在講 TMDb／Douban 等**metadata provider** 的失敗，語意會撞車。
7.3 ~~**備選方案（未採用）**~~：把三個碼都改成 `METADATA_`（`METADATA_NFO_LOCALIZE_DISABLED` 之類）。代價很小 —— movies 路由**零前端呼叫者**，wire 變更的爆炸半徑是零。
7.4 🔴 **不論選哪個，都不得沉默地再加第三個未註冊的 `NFO_` 碼。**
7.5 若選 7.2，Rule 25（mega-line）適用：`project-context.md` 的 `Last Updated:` 行必須**前置**本案條目，不得整側取代。

---

## Tasks / Subtasks

- [x] **Task 1 —— NFO 型別與映射（AC #1.2 / #2.1 / #2.2）** `internal/services/nfo_generator.go`
  - [x] 1.1 新增 `EpisodeNFO` struct（root `episodedetails`），緊鄰 `SeriesNFO`
  - [x] 1.2 `seriesToNFO(series)` —— 逐行鏡射 `GenerateSeriesNFO:127`
  - [x] 1.3 `episodeToNFO(episode, showTitle)` —— `Overview→Plot`、無 actor 區塊
  - [x] 1.4 XML 序列化健全性測試（root 元素名、omitempty 行為）

- [x] **Task 2 —— 翻譯層共用（AC #1.3 / #2.3）** `internal/services/nfo_localizer_service.go`
  - [x] 2.1 抽出 movies/TV 共用的「欄位切片 → `TranslateRequest` → 回填」邏輯
  - [x] 2.2 🔴 `LocalizeMovieNFO` 行為 byte-unchanged（既有 7 條測試零編輯，AC #5.8）
  - [x] 2.3 episode 的 glossary key 用 `episode.SeriesID`（AC #2.3 紅線）

- [x] **Task 3 —— 寫入層（AC #1.4 / #2.4）** 同檔
  - [x] 3.1 `writeReplaceNFO(targetPath string, data []byte) (*NFOLocalizeResult, error)` —— 首次備份 `.orig` 後覆寫；**不要**改動或重用 `writeAdditiveNFO`（那是電影雙槽專屬）
  - [x] 3.2 `LocalizeTVShowNFO` —— 🔴 `filepath.Join(series.FilePath.String, "tvshow.nfo")`，**無 `Dir()`**
  - [x] 3.3 `LocalizeEpisodeNFO` —— `filepath.Dir(episode.FilePath.String)` + basename（episode 的 FilePath **是檔案**）
  - [x] 3.4 槽位矩陣測試 ×6（AC #5.1 / #5.2）+ 路徑正確性測試（AC #5.3）

- [x] **Task 4 —— 批次整劇（AC #4）** 同檔
  - [x] 4.1 `LocalizeSeriesNFOWithEpisodes`：show 先行 → `FindBySeriesID` 逐集 → 彙總計數
  - [x] 4.2 series 只查一次、showTitle 傳遞（AC #4.3，禁 N+1）
  - [x] 4.3 單集失敗續行、無 FilePath 記 skipped（AC #5.7）

- [x] **Task 5 —— 路由與 confirm 閘門（AC #3）** `internal/handlers/nfo_localizer_handler.go` + `cmd/api/main.go`
  - [x] 5.1 新窄介面 `NFOLocalizerSeriesGetter` / `NFOLocalizerEpisodeGetter`（Rule 11，鏡射 `TranscriptionEpisodeGetter:27`）
  - [x] 5.2 `NFOLocalizerServiceInterface` 擴充三個方法；main.go 注入 `repos.Series` / `repos.Episodes`
  - [x] 5.3 兩條新路由 + `confirm_replace` 閘門（🔴 **在呼叫 service 之前**，AC #3.3）
  - [x] 5.4 新建 `nfo_localizer_handler_test.go`（目前不存在）：confirm 三案 + 503 + 404 + 成功路徑

- [x] **Task 6 —— Rule 7 裁定落地（AC #7）**
  - [x] 6.1 依裁定擴充 `project-context.md` Rule 7（碼清單 + 16→17 句子）**或**改前綴
  - [x] 6.2 若擴充：同步 `code-review/instructions.xml` Step 3 的 prefix 清單與同步日期
  - [x] 6.3 若動到 `project-context.md`：Rule 25 —— `Last Updated:` **前置**新條目，不得整側取代

- [x] **Task 7 —— 閘門（Rule 12 / Rule 15）**
  - [x] 7.1 `pnpm nx test api`
  - [x] 7.2 `pnpm nx lint api`（釘版 staticcheck-2026.1）
  - [x] 7.3 `pnpm run lint:all`（含全 repo `format:check`）
  - [x] 7.4 `gofmt -l` 本案檔案為空

---

## Dev Notes

### 三個最容易踩的坑

1. 🔴 **對 `series.FilePath` 呼叫 `filepath.Dir()`。** 它**已經是劇集資料夾**（`media_ingest_service.go:147` ← `SeriesDirFor`）。多一層 `Dir()` 會把 `tvshow.nfo` 寫到**媒體庫根目錄**，播放器掃不到，而且污染使用者的庫根。movies 版之所以有 `Dir()`，是因為 movie 的 `FilePath` 是**檔案**。
2. 🔴 **拿 episode id 當 glossary key。** 必須用 `episode.SeriesID` —— 同劇每集要共用同一份名詞庫，否則「隱形戰士」在第 3 集和第 4 集會翻成兩個名字。這正是 sub-5-5 CR H1 修過的同一個 bug。
3. 🔴 **先翻譯再檢查 confirm。** 那會讓一次被拒絕的請求仍然花掉 Claude 的錢。閘門在 handler、在 service 呼叫之前。

### 架構護欄

- **Rule 11（介面定義在 consumer 側）**：`NFOLocalizerSeriesGetter` / `NFOLocalizerEpisodeGetter` 定義在 `handlers`，各自只有一個方法。`*repository.SeriesRepository` / `*repository.EpisodeRepository` 結構性滿足。**不要把兩者合成一個胖介面**（`TranscriptionEpisodeGetter:22-29` 的註解明說理由）。
- **Rule 13（錯誤處理完整性）**：批次的每一個吞掉的錯誤都要 `s.logger.Warn` 帶 `episode_id` / `season` / `episode` / `error`。
- **Rule 3（API 回應格式）**：新錯誤訊息一律 zh-TW（9R-10a 的先例：**既有**英文訊息不在範圍內，不要順手改）。
- **Rule 4 / 19（分層）**：handler → service → repository。`services` **不得** import `subtitle`；本案不碰 subtitle 套件。
- **Rule 20（AC 契約版本）**：`NFOLocalizeResult`（`nfo_localizer_service.go:56`）**未帶 `[@contract-vN]` 戳記** ⇒ 隱含 v0。本案**沿用不改其形狀**（TV 只是把 `Replaced` 用得更常見）⇒ 不欠 bump、不欠 ack。批次的彙總型別是**新的**，且今天零消費者 ⇒ 可不戳記；dev 若判斷 FE story 會消費它，才 stamp v1。
- **Rule 7**：見 AC #7 —— 本案是**唯一**能把既有違規一併收乾淨的時機。
- **Rule 16（斷言品質）**：AC #5.3 的路徑斷言要比對**完整絕對路徑**，不是 `Contains("tvshow.nfo")`（那條在錯誤目錄下也會過）。

### Testing standards

- Go 測試與被測檔**同 package 同目錄**（Rule 9）。
- 既有 `nfo_localizer_service_test.go` 有 **7 條**測試與 fake completer 慣例 —— 直接沿用該檔的 helper，不要另建 fixture 體系。
- handler 測試目前**不存在**，本案新建；鏡射其他 handler 測試的 gin `httptest` 樣式。
- 檔案系統測試一律用 `t.TempDir()` 建**真實目錄結構**（`Show/Season01/ep.mkv`），不要 mock 檔案系統 —— 本案的整個風險就在路徑計算。

### Project Structure Notes

- 修改：`internal/services/nfo_generator.go`、`internal/services/nfo_localizer_service.go`、`internal/handlers/nfo_localizer_handler.go`、`cmd/api/main.go`（注入兩個 repo）。
- 新增：`internal/handlers/nfo_localizer_handler_test.go`。
- 可能修改（依 AC #7 裁定）：`project-context.md`、`_bmad/bmm/workflows/4-implementation/code-review/instructions.xml`。
- 無新目錄、無新 package、**零新第三方相依**（`encoding/xml` / `path/filepath` 皆 stdlib）。

### Time-dependent visual coverage

- **N/A —— 本 story 不觸及任何 `apps/web/src/components/**` 檔案**（純 Go 後端）。Rule 23 不適用。

### References

- [Source: `apps/api/internal/services/nfo_localizer_service.go#20-252`] — movies 版完整實作（前置檢查 / 翻譯 / 雙槽寫入 / 備份語意）
- [Source: `apps/api/internal/services/nfo_generator.go#32-45,127-177,179`] — `SeriesNFO`、`GenerateSeriesNFO` 欄位映射、`marshalNFO`
- [Source: `apps/api/internal/services/media_ingest_service.go#83-104,143-150,260-267`] — 🔴 `SeriesDirFor` / `Series.FilePath = SeriesDir`（目錄）/ `Episode.FilePath`（檔案）
- [Source: `apps/api/internal/handlers/nfo_localizer_handler.go#13-69`] — handler 樣板、`NFO_*` 錯誤碼現況
- [Source: `apps/api/internal/handlers/transcription_handler.go#22-29`] — Rule 11 窄介面樣板（`TranscriptionEpisodeGetter`）
- [Source: `apps/api/internal/repository/episode_repository.go#107,124`] — `FindByID` / `FindBySeriesID`
- [Source: `apps/api/internal/models/episode.go#9-38`] — `Episode` 欄位（有 `Overview`，**無** plot、**無** credits）
- [Source: `apps/api/cmd/api/main.go#960-962`] — nfo handler 掛載點（`if nfoLocalizer != nil`）
- [Source: `apps/api/internal/handlers/subtitle_pipeline_handler.go#101`] — 409 `StatusConflict` 既有樣式
- [Source: `project-context.md#Rule 7`] — 16 個權威前綴（**無 `NFO_`**）與「新增子系統須擴充清單並同步 instructions.xml」的原文
- [Source: `project-context.md#Rule 3 / 9 / 11 / 13 / 16 / 20 / 25`]
- [Source: sprint-status `9R-S1-nfo-localization-spike`] — TV 單槽、無 additive 選項的裁定來源
- [Source: `_bmad-output/planning-artifacts/strategy-review-whatsub-2026-08-19.md`] — differentiator 定位與「movies 路由零前端呼叫者」的實證

---

## Dev Agent Record

### Agent Model Used

claude-opus-5[1m] (BMAD `dev-story`, 2026-08-21)

### Debug Log References

- **真 RED（測試抓到既有 bug）**：`TestNFOHandler_MovieRouteStillNeedsNoConfirmation` 第一次執行**直接 panic** ——
  堆疊指向 `nfo_localizer_handler.go` 的 **既有** `LocalizeMovie`：`GetByID` 的介面允許回 `(nil, nil)`，而它只檢查 `err != nil`，
  下一行就 dereference ⇒ **nil pointer panic ＋ 帶堆疊的 500**，本該是乾淨的 404。與 9R-10a CR L3 在 transcribe 路由修過的是同一類缺陷。
  依 dev-story Step 7「快速可修就地修」補上 `|| movie == nil`，並加迴歸測試 `TestNFOHandler_NilMovieWithoutErrorIs404NotAPanic`。**Pre-existing fix，非本案引入。**
- **`nfo_generator.go` 的 42+/20− 有一半是格式**：`gofmt -w` 順手把**既有**的 struct tag 對齊漂移修掉了（`MovieNFO`/`SeriesNFO` 的欄位對齊）。
  已用 `git show origin/main:...` + `gofmt -l` **實證 main 本來就 unclean**（bugfix-e CR H1 的同款情境與處置）⇒ 保留修正並在此誠實揭露，**不宣稱「只加了一個 struct」**。

### Completion Notes List

**實作總結（7/7 task、7/7 AC）**

1. **AC #1 / #1.4 —— `LocalizeTVShowNFO`。** 🔴 本案最大的風險點已釘住：`series.FilePath` **已經是劇集資料夾**
   （`media_ingest_service.go:147` ← `SeriesDirFor`，會爬過 `Season 02`／`S02`／`第二季`），所以寫入路徑是
   `filepath.Join(series.FilePath.String, "tvshow.nfo")`，**沒有 `filepath.Dir()`**。
   `TestNFOLocalizer_TVShow_WritesInsideTheShowFolder` 用真實 `<tmp>/Buffy/Season01/xxx.mkv` 結構，
   **正向斷言完整絕對路徑** ＋ **反向斷言它不等於「上一層」的路徑**（`assert.NotEqual(..., filepath.Dir(showDir))`），
   讓「多一層 Dir」這個錯誤不可能悄悄通過。
2. **AC #2 —— `LocalizeEpisodeNFO`。** 新 `EpisodeNFO` struct，root element **`<episodedetails>`**（測試逐字斷言 —— 三家播放器都只認這個名字）。
   `Overview→<plot>`、`AirDate→<aired>`、`StillPath→<thumb>`、`SeasonNumber`/`EpisodeNumber`。**無 actor 區塊**（`models.Episode` 無 credits，測試以 `NotContains("<actor>")` 釘住是刻意不是遺漏）。
   episode 的 `FilePath` **是檔案**，所以這裡**用** `Dir()` —— 與 series 相反，測試同時反向斷言它**沒有**落在劇集資料夾。
3. **AC #2.3 —— glossary key 用父 series。** `s.loadGlossary(ctx, episode.SeriesID)`，
   `TestNFOLocalizer_Episode_GlossaryKeysOnTheParentSeries` 用 spy 記錄實際傳入的 key 並斷言 `"series-1"`。
   （per-episode key 會讓同一個角色在第 3 集和第 4 集翻成兩個名字 —— sub-5-5 CR H1 在字幕側修過的同一個 bug。）
4. **AC #1.3 —— 翻譯層共用而非複製。** 抽出 `translateKeyedFields`（丟空欄位／全空短路不打 API／缺 key 保留原值），
   `translateFields`(movies) / `translateSeriesFields` / `translateEpisodeFields` 三者共用。
   🔴 **movies 行為 byte-unchanged 的實證**：`nfo_localizer_service_test.go` 的 `git diff --numstat` 是 **291 added / 0 deleted** ——
   既有 7 條 movies 測試**一行未改**仍全綠。
   `ShowTitle` 刻意**不翻**（它是父劇集標題，已由 `LocalizeTVShowNFO` 翻過一次；逐集翻會花 N 倍 token 產生同一個字串，還可能得到 N 個不同譯名）。
5. **AC #3 —— confirm 閘門。** `requireReplaceConfirmation` 位於 handler、**在任何 lookup 與 localize 呼叫之前**。
   409 + 新碼 `NFO_REPLACE_NOT_CONFIRMED`，訊息 zh-TW（Rule 3）。
   **四案表格測試**（無 body／欄位缺席／`false`／無關 body）全部斷言 `loc.calls == 0` —— 不只「沒寫檔」，是**連翻譯都沒被呼叫**。
   movies 路由**不受影響**（它是 additive，本來就不需要 confirm），有專門測試釘住。
6. **AC #4 —— 批次 fail-soft。** `LocalizeSeriesNFOWithEpisodes`：show 先行（非 fail-soft，寫不了就是沒得開始）→ `FindBySeriesID` 逐集。
   無 FilePath → `Skipped`；單集失敗 → `Failed` 並**續行**；`lister.calls == 1` 釘住**無 N+1**（showTitle 隨參數下傳）。
   lister 本身失敗 → **降級為 show-only 而不是回報整體失敗**（show 檔已經在磁碟上了，回報失敗會蓋掉這個事實）。
7. **AC #7 —— ⚖️ 裁定 A 落地。** Rule 7 前綴集 **16 → 17**，正式註冊 `NFO_`：
   - `project-context.md`：碼清單新增一行（含 9R-13 既有的兩碼 + 本案的 `NFO_REPLACE_NOT_CONFIRMED`）、權威前綴集句子 16→17。
   - `code-review/instructions.xml` Step 3：inline prefix 清單、HTML 同步日期註解、**以及 auto-fix 路徑對照表**（`internal/{services,handlers}/nfo_*.go` → `NFO_`）。
   - **零 wire 變更**：9R-13 已出貨的兩個碼字串一字未動 —— 這是「把家族註冊起來」不是「改名」。
   - Rule 25：`Last Updated:` **前置**新條目、原條目改為 `Prior:` 保留（未整側取代）。

**新增測試（25 條）**
- `nfo_localizer_service_test.go` **+12**：tvshow 路徑正確性／備份三態（無原檔／有原檔／備份已存在不覆蓋）／無資料夾路徑報錯／
  episode 路徑正確性＋`<episodedetails>`＋無 actor／episode 備份／glossary series-key spy／缺 key 保留原值／無檔案路徑報錯／
  批次 skip+續行+無 N+1／lister 失敗降級／未接 lister 只做 show。
- `nfo_localizer_handler_test.go` **+13（新檔）**：confirm 四案表格＋episode confirm／movies 不需 confirm／
  兩條 happy path（含 `include_episodes`）／503／404（err 與 nil-nil 兩種）／無資料夾 400／500／getter 為 nil 時 TV 路由不註冊／nil-movie 迴歸守衛。

**閘門結果（全部實跑）**

| 閘門 | 結果 |
|---|---|
| `pnpm nx test api` | ✅ 全綠（0 FAIL） |
| `pnpm nx test web` | ✅ exit 0，235 檔 / 2722 測試 |
| `pnpm nx lint api` | ✅ go vet + staticcheck-2026.1 |
| `pnpm run lint:all` | ✅ **0 errors** / 119 warnings（main 既有基準） |
| `prettier --check .` | ✅ |
| `gofmt -l`（本案 6 檔） | ✅ 乾淨 |
| `pnpm run test:cleanup` | ✅ No test processes found |

**強制稽核項**

- 🔗 **AC Drift: NONE** —— grep `localize-nfo` / `LocalizeMovieNFO` / `writeAdditiveNFO` 於 `_bmad-output/implementation-artifacts/*.md`，
  命中集中在 9R-13 與本檔，**全部 REUSE 非 DRIFT**：movies 的兩槽 additive 語意、路由、錯誤碼字串**一律未變**，
  `translateFields` 雖被重構到共用核心但外部行為 byte-unchanged（既有 7 條測試零編輯全綠即證）。
- 📎 **Contract Stamps: NONE** —— 本 story、`nfo_localizer_service.go`、`nfo_localizer_handler.go`、9R-13 story 檔**皆無** `[@contract-v*]` 戳記
  ⇒ Rule 20 前向唯一原則下屬隱含 v0，不欠 bump／不欠 ack。新增的 `NFOSeriesLocalizeResult` 今天零消費者，未戳記。
- 🔒 **Rule 7 Wire Format: PASS（並修復既有違規）** —— 新增 1 碼 `NFO_REPLACE_NOT_CONFIRMED`；
  同一批把 9R-13 遺留的未註冊 `NFO_` 家族正式登記（16→17 來源），`project-context.md` 與 `instructions.xml` **兩處皆已同步**（Rule 7 原文的要求）。
- 🔒 **Rule 25 Mega-line: PASS** —— `Last Updated:` 前置本案條目、原 sub-1-3 條目改為 `Prior:` 完整保留（union，非整側取代）。
- 🔌 **Route Sync: 2 條新路由已註冊** —— `POST /api/v1/series/:id/localize-nfo` 與 `POST /api/v1/episodes/:id/localize-nfo`
  於 `nfo_localizer_handler.go` `RegisterRoutes` 註冊，經 `cmd/api/main.go:961` 的 `nfoLocalizerHandler.RegisterRoutes(apiV1)` 掛載
  （並由 `TestNFOHandler_TVRoutesAbsentWhenGettersAreNil` 反向釘住 getter 為 nil 時不註冊）。
- 🎭 **A11y Pre-Flight: N/A**（100% backend —— `git status` 零 `apps/web/` 檔案）。
- 🎨 **UX Verification: SKIPPED** —— no UI changes in this story。
- 🕰️ **Rule 23: N/A** —— 未觸及任何 `apps/web/src/components/**`。
- **Pre-existing failures: NONE**；**Pre-existing fix ×1**：movies handler 的 `(nil, nil)` panic（見 Debug Log）。

### Discovery Triage

**YES —— 3 項。**

| Lane | 發現 | 追蹤 |
|---|---|---|
| ① expand-scope-in-place | **9R-13 的 `NFO_` Rule 7 違規** | 由 **AC #7** 就地吸收（⚖️ Alexyu 裁定 A）。`project-context.md` + `instructions.xml` 兩處同步完成 |
| ① expand-scope-in-place | **movies handler 的 `(nil, nil)` nil-pointer panic** | 就地修復（2 行）＋迴歸測試 `TestNFOHandler_NilMovieWithoutErrorIs404NotAPanic`。屬 Step 7「快速可修就地修」，且該檔本來就在本案 File List 內 |
| ③ backlog-with-carry-forward-link | **nfo 在地化仍然零前端入口** | movies 版自 9R-13 出貨至今零前端呼叫者；本案補齊 TV 後端後，使用者**依然沒有任何地方可以點**。已立案 `backlog-nfo-localization-frontend-entry`，雙向連結本 story |

`nfo_localizer_handler_test.go` 的「不存在」問題已由本案**直接解決**（新建並涵蓋新舊全部路由），不另立條目。

### File List

| 檔案 | 變更 |
|---|---|
| `apps/api/internal/services/nfo_generator.go` | modified —— 新增 `EpisodeNFO`（root `episodedetails`）；⚠️ `gofmt` 順手修掉既有 struct tag 對齊漂移（純格式，main 本來就 unclean，已實證） |
| `apps/api/internal/services/nfo_localizer_service.go` | modified —— `translateKeyedFields` 共用核心、`NFOEpisodeLister` + `SetEpisodeLister`、`NFOSeriesLocalizeResult`、`LocalizeTVShowNFO` / `LocalizeEpisodeNFO` / `LocalizeSeriesNFOWithEpisodes`、`translateSeriesFields` / `translateEpisodeFields`、`seriesToNFO` / `episodeToNFO`、`writeReplaceNFO` |
| `apps/api/internal/services/nfo_localizer_service_test.go` | modified —— **+12 測試 / 0 行刪除**（既有 7 條 movies 測試零編輯） |
| `apps/api/internal/handlers/nfo_localizer_handler.go` | modified —— 兩個窄介面、`replaceConfirmation`、`requireReplaceConfirmation` 閘門、`LocalizeSeries` / `LocalizeEpisode`、**Pre-existing fix**（nil-movie 404） |
| `apps/api/internal/handlers/nfo_localizer_handler_test.go` | **new** —— 13 條測試 |
| `apps/api/cmd/api/main.go` | modified —— `SetEpisodeLister(repos.Episodes)` + handler 注入 `seriesService` / `repos.Episodes` |
| `project-context.md` | modified —— Rule 7 碼清單 +1 行、權威前綴集 16→17（`NFO_`）、Rule 25 mega-line 前置 |
| `_bmad/bmm/workflows/4-implementation/code-review/instructions.xml` | modified —— Step 3 prefix 清單 16→17、同步日期、auto-fix 路徑對照表 +2 條 |
| `_bmad-output/implementation-artifacts/9R-13a-tv-nfo-localization.md` | modified —— tasks 全勾、Dev Agent Record、File List、Change Log、Status → review |
| `_bmad-output/implementation-artifacts/sprint-status.yaml` | modified —— 裁定紀錄 + ready-for-dev → in-progress → review |

### Change Log

| 日期 | 變更 |
|---|---|
| 2026-08-21 | **Task 1（AC #1.2 / #2.1-2.2）** —— `EpisodeNFO`（root `episodedetails`，無 actor 區塊）＋ `seriesToNFO`（逐行鏡射 `GenerateSeriesNFO`）＋ `episodeToNFO`（`Overview→plot`）。 |
| 2026-08-21 | **Task 2（AC #1.3 / #2.3）** —— 抽出 `translateKeyedFields` 共用核心；movies 行為 byte-unchanged（既有 7 條測試零編輯全綠）；episode glossary key 用 `SeriesID`（spy 釘住）；`ShowTitle` 刻意不逐集翻。 |
| 2026-08-21 | **Task 3（AC #1.4 / #2.4）** —— `writeReplaceNFO`（首次備份 `.orig`，既有備份永不覆蓋）；🔴 tvshow 寫在 `series.FilePath` **本身**、episode 寫在 `Dir(episode.FilePath)`，兩邊都以完整絕對路徑正反向斷言。 |
| 2026-08-21 | **Task 4（AC #4）** —— `LocalizeSeriesNFOWithEpisodes` 逐集 fail-soft、skip 無檔案者、`lister.calls == 1` 釘住無 N+1、lister 失敗降級為 show-only。 |
| 2026-08-21 | **Task 5（AC #3）** —— 兩條 TV 路由 + `requireReplaceConfirmation` 閘門（**在 lookup／localize 之前**）；四案表格測試斷言 `calls == 0`；新建 handler 測試檔（13 條）；**Pre-existing fix**：movies handler `(nil, nil)` panic → 404。 |
| 2026-08-21 | **Task 6（AC #7，⚖️ 裁定 A）** —— Rule 7 註冊 `NFO_`（16→17 來源），同步 `project-context.md`（碼清單＋前綴集句子）與 `code-review/instructions.xml`（清單＋日期＋auto-fix 路徑）；零 wire 變更；Rule 25 mega-line 前置保留 `Prior:`。 |
| 2026-08-21 | **Task 7（Rule 12 / 15）** —— 閘門全綠：`nx test api` 0 FAIL、`nx test web` exit 0 / 2722、`nx lint api`、`lint:all` 0 errors、prettier、本案 6 檔 gofmt、test cleanup 無殘留。Rule 24 ③ 立案 `backlog-nfo-localization-frontend-entry`。Status → review。 |

---

## Senior Developer Review (AI)

**Reviewer:** Bob (SM) 代跑 `/code-review`，claude-opus-5[1m] · **Date:** 2026-08-21 · **Outcome:** APPROVED WITH FIXES

> ⚠️ **同 context 自審警告。** 由**實作者本人在同一 session** 執行，非跨模型獨立審查。
> **本輪動到 `project-context.md` 這種全域規則檔，強烈建議 PR 上另跑一輪跨模型審查。**

**Git vs Story File List：0 落差**（10 個檔案逐一對上）。

### 強制閘門

| 檢查 | 結果 |
|---|---|
| 🔒 Rule 7 Wire Format | **PASS** —— 掃描本案 3 個 Go 生產檔：0 個錯誤碼**常數**（本案的碼是 handler 內的 inline 字面值，與 9R-13 同款）。實際使用的三個碼 `NFO_LOCALIZE_DISABLED` / `NFO_LOCALIZE_FAILED` / `NFO_REPLACE_NOT_CONFIRMED` **全部已註冊**（本案把前綴集 16→17 並同步 `instructions.xml`）⇒ 掃描後**零違規** |
| 🔒 Rule 20 Contract Bump | **N/A** —— diff 中 `[@contract-vN→vM]` 命中數 0；本案與 9R-13 皆無戳記 |
| 🔒 Rule 25 Mega-line | **PASS** —— `project-context.md` 的 `Last Updated:` 由本案**前置**新條目，原 sub-1-3 條目改為 `Prior:` 完整保留（union，非整側取代）。非 rebase 情境，但因為本案就是動 mega-line 的那一方，仍逐項確認基準條目未遺失 |

### findings：0 HIGH / 4 MEDIUM / 3 LOW —— **7/7 全修**

| # | 嚴重度 | 發現 | 處置 |
|---|---|---|---|
| **M1** | MEDIUM | **glossary 的 N+1 —— 我只修了 series 那一半。** AC #4.3 明文要求「series 只查一次」，我照做了；但 `LocalizeEpisodeNFO` 內部**每一集都重新 `loadGlossary(episode.SeriesID)`**，24 集就是 24 次讀同一份名詞庫。與 AC 想防的是**同一類**問題，只是換了個軸。 | ✅ 修 —— 抽出 `localizeEpisodeWithGlossary(ctx, ep, showTitle, glossary)`；批次載入一次後下傳，單集入口仍自行載入。新增 `TestNFOLocalizer_Batch_LoadsTheShowGlossaryOncePerRunNotPerEpisode`：4 集的 run 斷言 spy 只收到 **2** 次（show 一次 + 迴圈一次），修前是 1+4=5 |
| **M2** | MEDIUM | **gin 路由參數名沒有任何守衛，而不一致會 boot 時 panic。** 兩條新路由用 `:id`；`series_handler.go:317-322`、`douban_rating_handler.go:127`、`transcription_handler.go:74` 目前**也都用 `:id`**（已逐一查證）—— 但**沒有任何測試釘住這件事**。將來有人寫 `/series/:seriesId/...` 就會讓整個 API **啟動時 panic**，而唯一會抓到的是 CI 的 serve-smoke gate（很晚、且訊息難解讀）。 | ✅ 修 —— `TestNFOHandler_TVRoutesCoexistWithTheExistingSeriesAndEpisodeRoutes`：在同一個 gin group 先註冊其他 handler 現有的 5 種路由形狀，再 `require.NotPanics` 註冊本案的兩條 |
| **M3** | MEDIUM | **`series.FilePath` 指向檔案時，錯誤訊息無法行動。** `tvshow.nfo` 是 **JOIN** 到這個路徑上的。欄位是自由字串（`SaveSeriesFromTMDb` 就收檔案路徑，雖然今天沒有生產呼叫者），一旦是檔案就會在 `os.WriteFile` 深處炸出 `not a directory`，路徑還是操作者從沒輸入過的組合。 | ✅ 修 —— 寫入前 `os.Stat` + `IsDir()`，回「`%q` 不是可讀取的目錄 —— 請重新掃描媒體庫」。+2 測試（指向檔案／目錄不存在），並斷言錯誤訊息**不含** `localize fields`（證明守衛在**付錢翻譯之前**就擋下） |
| **M4** | MEDIUM | **`?include_episodes=1` 會靜默只做 show 檔。** 原本比對字面字串 `"true"`，所以 `=1`、`=TRUE`、或裸寫 `?include_episodes` 全部落到 show-only 分支，然後**回 200** —— 呼叫者以為 24 集都做了，其實一集都沒做。 | ✅ 修 —— `wantsEpisodes(c)`：有帶參數就是要，除非值明確是 `false`/`0`/`no`。+7 案表格測試，用「回應是否含 `succeeded`」分辨走了哪個分支 |
| **L1** | LOW | **`.orig` 可能變成孤兒** —— 使用者若刪掉在地化後的 `tvshow.nfo`，`.orig` 會留著，下次執行走「檔案不存在 → 直接寫」分支、`Replaced=false`。 | ✅ 修（記錄型）—— 這是正確行為（備份永不被動）；不自動還原，因為那會把使用者的刪除動作反轉。已於 `writeReplaceNFO` 註解說明 |
| **L2** | LOW | **未 commit 狀態未記錄**（CR 步驟 3 透明度項）。 | ✅ 修（記錄型）—— 截至 review 完成，**10 個檔案**位於 `feat/9R-13a-tv-nfo-localization` 且**尚未 commit**，交由 `/ship` |
| **L3** | LOW | **沒跑過 `-race`。** | ✅ 跑了 —— `internal/services` 與 `internal/handlers` **皆乾淨**（`MockRetryRepository` 的既有 race 已於 9R-5 CR 修掉，這次沒有復發） |

### 看過但**判定不是** finding（避免下一輪重查）

- **🔴 有沒有繞過 confirm 閘門的呼叫端？** grep `LocalizeTVShowNFO|LocalizeEpisodeNFO|LocalizeSeriesNFOWithEpisodes` 全庫（排除測試）：**只有 handler**。沒有 scanner／enrichment／排程繞過閘門。這是本輪最想否證的假設。
- **路徑穿越**：`tvshow.nfo` 是常數；episode 用 `filepath.Base` 剝掉目錄 ⇒ 兩條路徑都無法穿越。
- **`translateKeyedFields` 重構是否改了 movies 行為**：全空短路（回 `nil, nil` ⇒ 所有 `ok` 為 false ⇒ 原值保留）、錯誤傳遞、欄位順序皆逐行等價；`nfo_localizer_service_test.go` numstat **348+/0−** 證明既有 7 條測試零編輯。
- **`seriesToNFO` 的 `Genres` 別名問題**：`GenerateSeriesNFO` 是 `nfo.Genres = series.Genres`（別名），我的 `seriesToNFO` 用 `append([]string(nil), ...)` **複製** —— 因為翻譯會就地改寫，別名會污染呼叫者的 `models.Series`。與 `movieToNFO` 一致。
- **批次先寫 show 再列舉 episodes**：若列舉失敗，show 檔已落磁碟 —— 這是刻意的（回報整體失敗會蓋掉「show 已完成」這個事實），已有測試與註解。
- **`ShouldBindJSON` 的錯誤被忽略**：任何非法 body 都落到「未確認」⇒ 409，方向安全。

### 修後閘門（全部重跑）

| 閘門 | 結果 |
|---|---|
| `pnpm nx test api` | ✅ 全綠（0 FAIL） |
| `pnpm nx test web` | ✅ exit 0，235 檔 / 2722 測試 |
| `go test -race`（services + handlers） | ✅ 兩套件皆乾淨 |
| `pnpm nx lint api` | ✅ go vet + staticcheck-2026.1 |
| `pnpm run lint:all` | ✅ 0 errors / 119 warnings（既有基準） |
| `prettier --check .` | ✅ |
| `gofmt -l`（本案檔案） | ✅ 乾淨 |
| `pnpm run test:cleanup` | ✅ No test processes found |

**測試總數**：`nfo_localizer_service_test.go` **22** 條（7 既有 movies + 15 新）· `nfo_localizer_handler_test.go` **15** 條（全新）⇒ **本案新增 30 條**。

### Action Items

無 —— 4 MEDIUM + 3 LOW **全數在 review 內修畢**。
Carry-forward 僅 `backlog-nfo-localization-frontend-entry`（dev-story 已立案，需 Sally 設計輪）。

### Change Log（review 追加）

| 日期 | 變更 |
|---|---|
| 2026-08-21 | **CR 修復 7/7** —— **M1** glossary N+1（我只修了 series 那一半，episode 側每集重讀）→ 抽 `localizeEpisodeWithGlossary`，批次載入一次；測試斷言 4 集 run 只讀 2 次。**M2** gin 路由參數名不一致會 boot panic 而無測試守衛 → 加共存測試（先註冊其他 handler 的 5 種形狀再 `NotPanics`）。**M3** `series.FilePath` 指向檔案會炸出無法行動的 `not a directory` → 寫入前 `Stat`+`IsDir`，錯誤訊息可行動且在付費翻譯之前。**M4** `?include_episodes=1` 靜默只做 show 卻回 200 → `wantsEpisodes` 改為「有帶就是要」，+7 案表格。**L1/L2** 記錄型；**L3** `-race` 兩套件乾淨。**否證項**：grep 全庫確認**沒有任何繞過 confirm 閘門的呼叫端**。+5 測試（共 30）。閘門全部重跑綠。Status review → done。 |
